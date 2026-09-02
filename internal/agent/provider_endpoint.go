// provider_endpoint.go — LLM 协议与端点解析（★ 2026-09-02）
//
// 配置语义回归「基础地址」：设置里只存 base（如 https://api.deepseek.com/v1、
// https://api.anthropic.com），完整请求端点由 ResolveEndpointURL 按 Protocol 拼接。
// 兼容旧配置：BaseURL 若已是完整端点（含 /chat/completions 等后缀）→ 直接用不重复拼。
//
// 协议枚举（与前端 config/models.js 的 protocol 取值、core.GetProviderProtocol 对齐）：
//   - openai-completions ：/chat/completions（OpenAI 兼容，默认）
//   - openai-responses   ：/responses（OpenAI Responses API，2026-08-28 引入）
//   - anthropic-messages ：/v1/messages（Anthropic Messages API）

package agent

import "strings"

// LLM 协议常量（models.json 每服务商 protocol 字段的合法值）。
const (
	ProtocolOpenAICompletions = "openai-completions"
	ProtocolOpenAIResponses   = "openai-responses"
	ProtocolAnthropicMessages = "anthropic-messages"
)

// DefaultProtocol 协议未配置时的默认值。
const DefaultProtocol = ProtocolOpenAICompletions

// ResolveEndpointURL 基础地址 + 协议 → 完整请求端点。
//   - base 已含目标协议路径后缀（旧「完整端点」配置）→ 原样使用（去尾斜杠）
//   - base 含其他协议路径后缀（如 /chat/completions 而 protocol=responses）→ 剥掉后重拼
//   - base 为纯域名 → 按协议拼标准路径
//
// ⚠️ 响应协议切换端点形态不同，配置中的 base 应只含基础路径（如 /v1）。
func ResolveEndpointURL(baseURL, protocol string) string {
	b := strings.TrimRight(baseURL, "/")
	if b == "" {
		return ""
	}
	switch protocol {
	case ProtocolOpenAIResponses:
		if strings.HasSuffix(b, "/responses") {
			return b
		}
		return ensureV1(stripLLMEndpointSuffixes(b)) + "/responses"
	case ProtocolAnthropicMessages:
		if strings.HasSuffix(b, "/messages") {
			return b
		}
		return ensureV1(stripLLMEndpointSuffixes(b)) + "/messages"
	default: // openai-completions / 空
		if strings.HasSuffix(b, "/chat/completions") {
			return b
		}
		return stripLLMEndpointSuffixes(b) + "/chat/completions"
	}
}

// stripLLMEndpointSuffixes 剥掉已知的协议路径后缀（多协议互切时清洗旧形态）。
func stripLLMEndpointSuffixes(b string) string {
	for _, s := range []string{"/chat/completions", "/completions", "/responses", "/messages"} {
		if strings.HasSuffix(b, s) {
			return strings.TrimRight(strings.TrimSuffix(b, s), "/")
		}
	}
	return b
}

// ensureV1 确保基础路径含 /v1 段（responses/messages 协议的标准基线路径）。
func ensureV1(b string) string {
	if strings.HasSuffix(b, "/v1") {
		return b
	}
	return b + "/v1"
}
