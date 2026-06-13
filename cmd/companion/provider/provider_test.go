package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"
)

// --- SanitizeToolPairing ---

func toolIDsAnswered(msgs []Message) bool {
	answered := map[string]bool{}
	for _, m := range msgs {
		if m.Role == RoleTool {
			answered[m.ToolCallID] = true
		}
	}
	for _, m := range msgs {
		for _, tc := range m.ToolCalls {
			if !answered[tc.ID] {
				return false
			}
		}
	}
	return true
}

func TestSanitizeToolPairingBackfillsDanglingCall(t *testing.T) {
	in := []Message{
		{Role: RoleUser, Content: "list files"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "ls"}}},
		{Role: RoleUser, Content: "never mind"},
	}
	out := SanitizeToolPairing(in)
	if !toolIDsAnswered(out) {
		t.Fatalf("dangling tool_call left unanswered: %+v", out)
	}
	if out[2].Role != RoleTool || out[2].ToolCallID != "c1" {
		t.Fatalf("expected a backfilled tool result for c1 at index 2, got %+v", out[2])
	}
}

func TestSanitizeToolPairingKeepsCallOrderAndMultiple(t *testing.T) {
	in := []Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "a"}, {ID: "b"}, {ID: "c"}}},
		{Role: RoleTool, ToolCallID: "b", Content: "B"},
		{Role: RoleTool, ToolCallID: "a", Content: "A"},
	}
	out := SanitizeToolPairing(in)
	if !toolIDsAnswered(out) {
		t.Fatalf("not all calls answered: %+v", out)
	}
	gotOrder := []string{out[1].ToolCallID, out[2].ToolCallID, out[3].ToolCallID}
	want := []string{"a", "b", "c"}
	for i := range want {
		if gotOrder[i] != want[i] {
			t.Fatalf("tool results out of call order: got %v want %v", gotOrder, want)
		}
	}
}

func TestSanitizeToolPairingDropsOrphanToolMessage(t *testing.T) {
	in := []Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleTool, ToolCallID: "ghost", Content: "leftover"},
		{Role: RoleAssistant, Content: "hello"},
	}
	out := SanitizeToolPairing(in)
	for _, m := range out {
		if m.Role == RoleTool {
			t.Fatalf("orphan tool message survived: %+v", out)
		}
	}
	if len(out) != 2 {
		t.Fatalf("want 2 messages after dropping the orphan, got %d: %+v", len(out), out)
	}
}

func TestSanitizeToolPairingLeavesWellFormedUnchanged(t *testing.T) {
	in := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "q"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "ls"}}},
		{Role: RoleTool, ToolCallID: "c1", Name: "ls", Content: "main.go"},
		{Role: RoleAssistant, Content: "done"},
	}
	out := SanitizeToolPairing(in)
	if len(out) != len(in) {
		t.Fatalf("well-formed history changed length: %d -> %d", len(in), len(out))
	}
	for i := range in {
		if out[i].Role != in[i].Role || out[i].Content != in[i].Content || out[i].ToolCallID != in[i].ToolCallID {
			t.Fatalf("well-formed message %d mutated: %+v -> %+v", i, in[i], out[i])
		}
	}
}

func TestSanitizeToolPairingClosesTruncatedArgs(t *testing.T) {
	cases := []struct{ in, want string }{
		{`{`, `{}`},
		{`{"time": 2`, `{"time": 2}`},
		{`{"command": "ls -la`, `{"command": "ls -la"}`},
		{`{"a": 1,`, `{"a": 1}`},
		{`{"a":`, `{"a":null}`},
		{`{"path": "C:\\tmp\`, `{"path": "C:\\tmp"}`},
		{`{"items": [1, 2`, `{"items": [1, 2]}`},
		{`total garbage`, `{}`},
		{`{"ok": true}`, `{"ok": true}`},
		{``, ``},
	}
	for _, c := range cases {
		in := []Message{
			{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Name: "bash", Arguments: c.in}}},
			{Role: RoleTool, ToolCallID: "c1", Content: "r"},
		}
		out := SanitizeToolPairing(in)
		if got := out[0].ToolCalls[0].Arguments; got != c.want {
			t.Errorf("args %q repaired to %q, want %q", c.in, got, c.want)
		}
		if in[0].ToolCalls[0].Arguments != c.in {
			t.Errorf("stored history mutated for %q: %q", c.in, in[0].ToolCalls[0].Arguments)
		}
	}
}

// --- Pricing.Cost ---

func TestPricingCostNil(t *testing.T) {
	var p *Pricing
	if got := p.Cost(&Usage{PromptTokens: 100}); got != 0 {
		t.Errorf("nil Pricing.Cost = %f, want 0", got)
	}
}

func TestPricingCostNilUsage(t *testing.T) {
	p := &Pricing{Input: 2.0, Output: 10.0}
	if got := p.Cost(nil); got != 0 {
		t.Errorf("nil Usage.Cost = %f, want 0", got)
	}
}

func TestPricingCostBothNil(t *testing.T) {
	var p *Pricing
	if got := p.Cost(nil); got != 0 {
		t.Errorf("both nil.Cost = %f, want 0", got)
	}
}

func TestPricingCostCalculation(t *testing.T) {
	p := &Pricing{
		CacheHit: 0.5,
		Input:    2.0,
		Output:   10.0,
	}
	u := &Usage{
		CacheHitTokens:   1_000_000,
		CacheMissTokens:  500_000,
		CompletionTokens: 200_000,
	}
	got := p.Cost(u)
	if got != 3.5 {
		t.Errorf("Cost = %f, want 3.5", got)
	}
}

func TestPricingCostZeroTokens(t *testing.T) {
	p := &Pricing{Input: 2.0, Output: 10.0}
	u := &Usage{}
	if got := p.Cost(u); got != 0 {
		t.Errorf("zero tokens Cost = %f, want 0", got)
	}
}

// --- Pricing.Symbol ---

func TestPricingSymbolDefault(t *testing.T) {
	p := &Pricing{}
	if got := p.Symbol(); got != "¥" {
		t.Errorf("empty Currency.Symbol() = %q, want ¥", got)
	}
}

func TestPricingSymbolNil(t *testing.T) {
	var p *Pricing
	if got := p.Symbol(); got != "¥" {
		t.Errorf("nil.Symbol() = %q, want ¥", got)
	}
}

func TestPricingSymbolCustom(t *testing.T) {
	p := &Pricing{Currency: "$"}
	if got := p.Symbol(); got != "$" {
		t.Errorf("Symbol() = %q, want $", got)
	}
}

// --- Registry ---

func TestRegisterAndNew(t *testing.T) {
	// Use a fresh sub-registry by testing Register+New logic directly
	// (global registry shared across tests, so we test the mechanics)
	kind := "test_" + t.Name()
	Register(kind, func(cfg Config) (Provider, error) {
		return &mockProvider{name: cfg.Name}, nil
	})
	p, err := New(kind, Config{Name: "test-instance"})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if p.Name() != "test-instance" {
		t.Fatalf("Name = %q, want %q", p.Name(), "test-instance")
	}
}

func TestNewUnknownKind(t *testing.T) {
	_, err := New("nonexistent_"+t.Name(), Config{})
	if err == nil || !strings.Contains(err.Error(), "unknown kind") {
		t.Fatalf("want 'unknown kind' error, got %v", err)
	}
}

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Stream(ctx context.Context, req Request) (<-chan Chunk, error) {
	ch := make(chan Chunk)
	close(ch)
	return ch, nil
}

// --- Kinds ---

func TestKindsReturnsRegistered(t *testing.T) {
	kind := "kinds_test_" + t.Name()
	Register(kind, func(cfg Config) (Provider, error) {
		return &mockProvider{}, nil
	})
	found := false
	for _, k := range Kinds() {
		if k == kind {
			found = true
		}
	}
	if !found {
		t.Fatalf("Kinds() should include %q, got %v", kind, Kinds())
	}
}

// --- AuthError ---

func TestAuthErrorMessage(t *testing.T) {
	e := &AuthError{Provider: "deepseek", KeyEnv: "DEEPSEEK_API_KEY", Status: 401}
	msg := e.Error()
	if !strings.Contains(msg, "deepseek") || !strings.Contains(msg, "DEEPSEEK_API_KEY") {
		t.Fatalf("AuthError message missing details: %s", msg)
	}
}

// --- APIError ---

func TestAPIErrorMessage(t *testing.T) {
	e := &APIError{Provider: "test", Status: 429, Body: "rate limited"}
	msg := e.Error()
	if !strings.Contains(msg, "429") || !strings.Contains(msg, "rate limited") {
		t.Fatalf("APIError message missing details: %s", msg)
	}
}

func TestAPIErrorMessageNoBody(t *testing.T) {
	e := &APIError{Provider: "test", Status: 500}
	msg := e.Error()
	if strings.Contains(msg, "rate limited") {
		t.Fatalf("APIError should not have body: %s", msg)
	}
}

// --- StreamInterruptedError ---

func TestStreamInterruptedError(t *testing.T) {
	e := &StreamInterruptedError{Err: net.ErrClosed}
	if !IsStreamInterrupted(e) {
		t.Fatal("IsStreamInterrupted should detect StreamInterruptedError")
	}
	if !errors.Is(e, net.ErrClosed) {
		t.Fatal("StreamInterruptedError should unwrap")
	}
}

func TestStreamInterruptedErrorNil(t *testing.T) {
	if IsStreamInterrupted(nil) {
		t.Fatal("nil should not be a stream interrupted error")
	}
}

// --- CanonicalizeSchema ---

func TestCanonicalizeSchemaEmpty(t *testing.T) {
	got := CanonicalizeSchema(nil)
	if string(got) != `{"type":"object"}` {
		t.Fatalf("empty schema should become {\"type\":\"object\"}, got %s", string(got))
	}
}

func TestCanonicalizeSchemaSortsRequired(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","required":["z","a","m"]}`)
	got := CanonicalizeSchema(raw)
	var result map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	req, _ := result["required"].([]any)
	if len(req) != 3 || req[0] != "a" || req[1] != "m" || req[2] != "z" {
		t.Fatalf("required should be sorted: got %v", req)
	}
}

// --- RetryableStatus ---

func TestRetryableStatus(t *testing.T) {
	for _, s := range []int{408, 429, 500, 502, 503, 504, 529, 599} {
		if !RetryableStatus(s) {
			t.Errorf("status %d should be retryable", s)
		}
	}
	for _, s := range []int{200, 400, 401, 402, 403, 404, 422} {
		if RetryableStatus(s) {
			t.Errorf("status %d should not be retryable", s)
		}
	}
}

// --- transientErr ---

func TestTransientErr(t *testing.T) {
	if transientErr(nil) {
		t.Error("nil should not be transient")
	}
	if transientErr(context.Canceled) || transientErr(context.DeadlineExceeded) {
		t.Error("ctx cancel/deadline should not be transient")
	}
	if !transientErr(errors.New("connection reset")) {
		t.Error("network-ish error should be transient")
	}
}

// --- IsConnReset ---

func TestIsConnReset(t *testing.T) {
	if IsConnReset(nil) {
		t.Error("nil is not a conn reset")
	}
	if IsConnReset(context.Canceled) || IsConnReset(context.DeadlineExceeded) {
		t.Error("ctx cancel/deadline must not look like a recoverable reset")
	}
	if IsConnReset(errors.New("decode stream: invalid character")) {
		t.Error("a plain protocol error must not be treated as a conn reset")
	}
	for _, err := range []error{
		io.ErrUnexpectedEOF,
		&net.OpError{Op: "read", Err: syscall.ECONNRESET},
		fmt.Errorf("read stream: %w", &net.OpError{Op: "read", Err: errors.New("wsarecv: forcibly closed")}),
	} {
		if !IsConnReset(err) {
			t.Errorf("want conn reset for %v", err)
		}
	}
}

// --- backoffDelay ---

func TestBackoffDelay(t *testing.T) {
	if d := backoffDelay(1, 0); d < 500*time.Millisecond || d >= 750*time.Millisecond {
		t.Errorf("attempt 1 base delay = %v, want [500ms,750ms)", d)
	}
	if d := backoffDelay(20, 0); d > maxBackoff+250*time.Millisecond {
		t.Errorf("delay %v exceeds cap+jitter", d)
	}
	if d := backoffDelay(5, 3*time.Second); d != 3*time.Second {
		t.Errorf("Retry-After should win: %v", d)
	}
	if d := backoffDelay(1, time.Hour); d != maxBackoff {
		t.Errorf("Retry-After should be capped to %v, got %v", maxBackoff, d)
	}
}

// --- SendWithRetry ---

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func statusResp(status int, hdr map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range hdr {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("body")), Header: h}
}

func newDummyReq(ctx context.Context) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodPost, "http://x/y", nil)
}

func TestSendWithRetryFailsFastOnClientErrors(t *testing.T) {
	for _, status := range []int{400, 402, 422} {
		calls := 0
		cl := &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return statusResp(status, nil), nil
		})}
		_, err := SendWithRetry(context.Background(), cl, "p", "KEY", newDummyReq)
		if calls != 1 {
			t.Errorf("status %d retried (%d calls), should fail fast", status, calls)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Status != status {
			t.Errorf("status %d: want *APIError with Status=%d, got %v", status, status, err)
		}
	}
}

func TestSendWithRetryAuthError(t *testing.T) {
	calls := 0
	cl := &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return statusResp(401, nil), nil
	})}
	_, err := SendWithRetry(context.Background(), cl, "deepseek", "DEEPSEEK_API_KEY", newDummyReq)
	if calls != 1 {
		t.Errorf("401 retried (%d calls), should fail fast", calls)
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) || authErr.KeyEnv != "DEEPSEEK_API_KEY" {
		t.Errorf("want *AuthError naming the key env, got %v", err)
	}
}

func TestSendWithRetryRecoversAndNotifies(t *testing.T) {
	calls := 0
	cl := &http.Client{Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return statusResp(503, nil), nil
		}
		return statusResp(200, nil), nil
	})}
	var infos []RetryInfo
	ctx := WithRetryNotify(context.Background(), func(i RetryInfo) { infos = append(infos, i) })

	resp, err := SendWithRetry(ctx, cl, "p", "KEY", newDummyReq)
	if err != nil {
		t.Fatalf("should recover after one retry: %v", err)
	}
	if resp.StatusCode != 200 || calls != 2 {
		t.Fatalf("status=%d calls=%d, want 200 after 2 calls", resp.StatusCode, calls)
	}
	if len(infos) != 1 || infos[0].Attempt != 1 || infos[0].Max != MaxRetries {
		t.Fatalf("retry notify = %#v, want one Attempt 1/%d", infos, MaxRetries)
	}
}
