// provider_http.go — LLM 流式 HTTP 请求共享骨架（★ 2026-09-02）
//
// OpenAIProvider 原有的重试循环被抽为通用 helper，供新协议适配器
// （Anthropic messages / OpenAI responses）复用：
//
//   - 指数退避 + 抖动（0.5s 起，上限 30s）
//   - 401/403 认证错误不重试；408/429/5xx 可重试；其他 4xx 客户端错误不重试
//   - 流式解析失败：未产出内容（首帧前断/网络瞬断）→ 安全重试；
//     已产出内容 → 不重试（避免重复输出/工具重复执行），带已累积内容返回错误
//
// 认证头差异由调用方经 Headers/AuthHeader 指定（OpenAI 系 Bearer、Anthropic x-api-key）。

package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// llmStreamCall 一次流式 POST 的参数集。
type llmStreamCall struct {
	URL        string            // 最终请求端点（已拼接协议路径）
	Body       []byte            // 序列化后的请求体
	Headers    map[string]string // 额外请求头（如 anthropic-version；Content-Type/Accept 自动加）
	AuthHeader string            // 认证头名：默认 Authorization（Bearer）；Anthropic 用 x-api-key
	APIKey     string
}

// defaultLLMClient 默认 HTTP 客户端（★ 2026-09-02 由 OpenAIProvider.client() 提取共享）：
// 连接池/keep-alive 复用 + LLM 专用超时（见 llm* 变量）。
func defaultLLMClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   llmDialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   llmTLSHandshakeTimeout,
			ResponseHeaderTimeout: llmResponseHeaderTimeout, // 服务器须在该时限内返回响应头（thinking 排队可能较久）
			ForceAttemptHTTP2:     true,
		},
		Timeout: llmClientTimeout, // SSE 流式读取兜底
	}
}

// postLLMStream 带重试的流式 POST。onOK 在 HTTP 200 时解析响应流（返回 Message；err 携带已累积产出）。
// notify 为重试通知（nil 安全）。重试策略与 OpenAIProvider.Chat 原实现一致（2026-08-21 策略）。
func postLLMStream(ctx context.Context, client *http.Client, call llmStreamCall,
	onOK func(io.Reader) (Message, error), notify func(attempt int, errMsg string)) (Message, error) {

	maxRetries := llmMaxRetries
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避 + 抖动：0.5s, 1s, 2s, 4s, 8s, 16s, 30s, 30s...
			delay := time.Duration(1<<(attempt-1)) * llmRetryBaseDelay
			if delay > llmRetryMaxDelay {
				delay = llmRetryMaxDelay
			}
			delay += time.Duration(rand.Intn(250)) * time.Millisecond
			select {
			case <-ctx.Done():
				return Message{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, call.URL, bytes.NewReader(call.Body))
		if err != nil {
			return Message{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		if call.AuthHeader != "" {
			// 非 Bearer 认证（如 Anthropic x-api-key）
			req.Header.Set(call.AuthHeader, call.APIKey)
		} else {
			req.Header.Set("Authorization", "Bearer "+call.APIKey)
		}
		for k, v := range call.Headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return Message{}, err
			}
			lastErr = fmt.Errorf("LLM 请求失败 (第%d次): %w", attempt+1, err)
			log.Printf("[llm] 网络请求失败（第%d次，退避后重试）: %v", attempt+1, err)
			if notify != nil {
				notify(attempt+1, err.Error())
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			msg, perr := onOK(resp.Body)
			resp.Body.Close()
			if perr == nil {
				return msg, nil
			}
			// ★ 已产出内容 → 不自动重试（重试会重复输出；若流中含 tool_calls 还会导致工具重复执行）
			if msg.Content != "" || msg.Reasoning != "" || len(msg.ToolCalls) > 0 {
				return msg, fmt.Errorf("LLM 流式响应中断（已接收内容 %d 字符，不自动重试以免重复）: %w",
					len(msg.Content)+len(msg.Reasoning), perr)
			}
			lastErr = fmt.Errorf("LLM 流式读取失败 (第%d次): %w", attempt+1, perr)
			log.Printf("[llm] 流式读取失败（第%d次，退避后重试）: %v", attempt+1, perr)
			if notify != nil {
				notify(attempt+1, perr.Error())
			}
			continue
		}

		// 处理非 200 状态码
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		statusCode := resp.StatusCode
		bodyStr := strings.TrimSpace(string(b))

		// 401/403 → 认证错误，不重试
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			lastErr = fmt.Errorf("LLM HTTP %d (认证失败): %s", statusCode, bodyStr)
			log.Printf("[llm] 认证失败（不重试）: %v", lastErr)
			return Message{}, lastErr
		}
		// 可重试状态码：408（超时）、429（限流）、5xx（服务端错误）
		if statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode <= 599) {
			lastErr = fmt.Errorf("LLM HTTP %d (第%d次): %s", statusCode, attempt+1, bodyStr)
			log.Printf("[llm] HTTP %d（第%d次，退避后重试）: %s", statusCode, attempt+1, bodyStr)
			if notify != nil {
				notify(attempt+1, fmt.Sprintf("HTTP %d: %s", statusCode, bodyStr))
			}
			continue
		}
		// 其他 4xx → 客户端错误，不重试
		lastErr = fmt.Errorf("LLM HTTP %d: %s", statusCode, bodyStr)
		log.Printf("[llm] HTTP %d（客户端错误，不重试）: %s", statusCode, bodyStr)
		emitBridgeEvent("agent/request-error", map[string]any{"error": lastErr.Error()})
		return Message{}, lastErr
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("LLM 请求失败（已达最大重试次数 %d）", maxRetries)
	}
	emitBridgeEvent("agent/request-error", map[string]any{"error": lastErr.Error()})
	return Message{}, lastErr
}
