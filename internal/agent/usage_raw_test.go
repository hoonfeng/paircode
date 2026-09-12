package agent

// usage_raw_test.go — Usage.Raw（provider 原始 usage 报文）保留性回归。
//
// 背景：llm-trace 需要记录「API 真实返回的 token 使用」，而 Usage 的归一化
// （prompt_cache_hit_tokens ← prompt_tokens_details.cached_tokens、hit 反推 miss、
// input_tokens→prompt_tokens 等）会改写字段并丢弃未建模的厂商扩展字段。
// 因此 UnmarshalJSON 必须**原样**保留服务端报文到 Raw，且 Raw 不参与对外序列化。

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUsageRawPreserved(t *testing.T) {
	body := `{"prompt_tokens":100,"completion_tokens":5,"total_tokens":105,
	          "prompt_cache_hit_tokens":80,"prompt_cache_miss_tokens":20,
	          "prompt_tokens_details":{"cached_tokens":80},
	          "some_vendor_ext":{"x":1},"reasoning_tokens":3}`
	var u Usage
	if err := json.Unmarshal([]byte(body), &u); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if u.PromptTokens != 100 || u.CompletionTokens != 5 || u.TotalTokens != 105 {
		t.Fatalf("归一化基础字段异常: %+v", u)
	}
	if u.PromptCacheHitTokens != 80 || u.PromptCacheMissTokens != 20 {
		t.Fatalf("缓存字段归一化异常: %+v", u)
	}
	if u.Raw == nil {
		t.Fatalf("原始报文未保留")
	}
	if u.Raw["prompt_tokens"] != float64(100) {
		t.Errorf("Raw 值异常: %+v", u.Raw)
	}
	// 未建模的厂商扩展字段：归一化会丢弃，Raw 必须保留（真实用量核对的关键）
	if _, ok := u.Raw["some_vendor_ext"]; !ok {
		t.Errorf("扩展字段丢失: %+v", u.Raw)
	}
	if _, ok := u.Raw["reasoning_tokens"]; !ok {
		t.Errorf("reasoning_tokens 丢失: %+v", u.Raw)
	}
	// 不得污染对外序列化（token 统计落盘 / WS usage 事件体积不变）
	b, _ := json.Marshal(u)
	if strings.Contains(string(b), "some_vendor_ext") || strings.Contains(string(b), "\"Raw\"") {
		t.Errorf("Raw 不应参与 json.Marshal: %s", string(b))
	}
}

func TestUsageRawOpenAICachedOnly(t *testing.T) {
	// 网关只给 prompt_tokens_details.cached_tokens：归一化补 hit，miss 本地反推，
	// 但 Raw 必须保持服务端原样（分析时能看出「miss 是推导值」）。
	var u Usage
	body := `{"prompt_tokens":200,"completion_tokens":1,"prompt_tokens_details":{"cached_tokens":150}}`
	if err := json.Unmarshal([]byte(body), &u); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if u.PromptCacheHitTokens != 150 || u.PromptCacheMissTokens != 50 {
		t.Fatalf("归一化异常: hit=%d miss=%d", u.PromptCacheHitTokens, u.PromptCacheMissTokens)
	}
	if _, has := u.Raw["prompt_cache_miss_tokens"]; has {
		t.Errorf("Raw 不应包含本地推导的 miss: %+v", u.Raw)
	}
	if _, has := u.Raw["prompt_tokens_details"]; !has {
		t.Errorf("Raw 应保留 cached_tokens 详情: %+v", u.Raw)
	}
}

// 链路级：OpenAI 兼容 SSE 流 → parseSSE → Chunk.Usage.Raw 必须透传原始报文
// （llm-trace 的 response 事件正是取这条链路的 Usage）。
func TestParseSSEUsageRawEndToEnd(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"hi"}}]}`,
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,` +
			`"completion_tokens":7,"total_tokens":107,"prompt_cache_hit_tokens":80,` +
			`"prompt_cache_miss_tokens":20,"some_vendor_ext":1,"reasoning_tokens":3}}`,
		`data: [DONE]`,
		``,
	}, "\n")
	var got *Usage
	if _, err := parseSSE(strings.NewReader(sse), func(c Chunk) {
		if c.Usage != nil {
			got = c.Usage
		}
	}); err != nil {
		t.Fatalf("parseSSE 失败: %v", err)
	}
	if got == nil {
		t.Fatal("未从 SSE 流取到 usage")
	}
	if got.PromptTokens != 100 || got.CompletionTokens != 7 || got.TotalTokens != 107 {
		t.Errorf("基础用量异常: %+v", got)
	}
	if got.PromptCacheHitTokens != 80 || got.PromptCacheMissTokens != 20 {
		t.Errorf("缓存用量异常: %+v", got)
	}
	if got.Raw == nil {
		t.Fatal("原始报文未透传到 Chunk.Usage.Raw")
	}
	if got.Raw["some_vendor_ext"] != float64(1) || got.Raw["reasoning_tokens"] != float64(3) {
		t.Errorf("厂商扩展字段丢失: %+v", got.Raw)
	}
}
