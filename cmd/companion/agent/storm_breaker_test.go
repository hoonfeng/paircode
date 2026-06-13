package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ─── Provider 重试测试 ──

type retryRecordSink struct {
	mu      sync.Mutex
	retries []RetryInfo
}

func (s *retryRecordSink) notify(info RetryInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retries = append(s.retries, info)
}

func TestSendWithRetry_Success(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`OK`))
	}))
	defer srv.Close()

	ctx := context.Background()
	resp, err := sendWithRetry(ctx, http.DefaultClient, "test", func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	})
	if err != nil {
		t.Fatalf("sendWithRetry failed: %v", err)
	}
	defer resp.Body.Close()
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestSendWithRetry_RetryThenSuccess(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"overloaded"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`OK`))
	}))
	defer srv.Close()

	ctx := context.Background()
	resp, err := sendWithRetry(ctx, http.DefaultClient, "test", func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	})
	if err != nil {
		t.Fatalf("sendWithRetry failed: %v", err)
	}
	defer resp.Body.Close()
	if calls != 3 {
		t.Fatalf("expected 3 calls (2 retries), got %d", calls)
	}
}

func TestSendWithRetry_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	_, err := sendWithRetry(ctx, http.DefaultClient, "test", func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	})
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthError, got %T: %v", err, err)
	}
	if authErr.Status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", authErr.Status)
	}
}

func TestSendWithRetry_RetryNotify(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sink := &retryRecordSink{}
	ctx := WithRetryNotify(context.Background(), sink.notify)
	_, err := sendWithRetry(ctx, http.DefaultClient, "test", func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	})
	if err != nil {
		t.Fatalf("sendWithRetry failed: %v", err)
	}
	sink.mu.Lock()
	n := len(sink.retries)
	sink.mu.Unlock()
	if n != 1 {
		t.Fatalf("expected 1 retry notification, got %d", n)
	}
}

// ─── IsConnReset 测试 ──

func TestIsConnReset(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{io.ErrUnexpectedEOF, true},
		{io.EOF, true},
		{context.Canceled, false},
		{context.DeadlineExceeded, false},
		{fmt.Errorf("some other error"), false},
		{nil, false},
	}
	for _, tc := range tests {
		got := IsConnReset(tc.err)
		if got != tc.want {
			t.Errorf("IsConnReset(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

// ─── retryableStatus 测试 ──

func TestRetryableStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{408, true},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{400, false},
		{401, false},
		{403, false},
		{422, false},
		{200, false},
	}
	for _, tc := range tests {
		got := retryableStatus(tc.code)
		if got != tc.want {
			t.Errorf("retryableStatus(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// ─── PrefixShape / CacheDiagnostics 测试 ──

func TestCaptureShape(t *testing.T) {
	system := "You are a helpful assistant."
	defs := []ToolDefinition{
		{Type: "function", Function: FunctionDefinition{Name: "read_file", Description: "Read a file"}},
		{Type: "function", Function: FunctionDefinition{Name: "write_file", Description: "Write a file"}},
	}
	shape := CaptureShape(system, defs)
	if shape.SystemHash == "" {
		t.Error("SystemHash should not be empty")
	}
	if shape.ToolsHash == "" {
		t.Error("ToolsHash should not be empty")
	}
	if shape.PrefixHash == "" {
		t.Error("PrefixHash should not be empty")
	}
}

func TestCompareShape_NoChange(t *testing.T) {
	system := "You are a helpful assistant."
	defs := []ToolDefinition{
		{Type: "function", Function: FunctionDefinition{Name: "read_file", Description: "Read a file"}},
	}
	shape1 := CaptureShape(system, defs)
	shape2 := CaptureShape(system, defs)
	diag := CompareShape(shape1, shape2)
	if diag.PrefixChanged {
		t.Errorf("expected no change, got reasons: %v", diag.ChangeReasons)
	}
}

func TestCompareShape_SystemChange(t *testing.T) {
	defs := []ToolDefinition{
		{Type: "function", Function: FunctionDefinition{Name: "read_file", Description: "Read a file"}},
	}
	shape1 := CaptureShape("system v1", defs)
	shape2 := CaptureShape("system v2", defs)
	diag := CompareShape(shape1, shape2)
	if !diag.PrefixChanged {
		t.Error("expected prefix change")
	}
	if len(diag.ChangeReasons) != 1 || diag.ChangeReasons[0] != "system" {
		t.Errorf("expected change reason 'system', got %v", diag.ChangeReasons)
	}
}

func TestCompareShape_ToolsChange(t *testing.T) {
	system := "You are a helpful assistant."
	defs1 := []ToolDefinition{
		{Type: "function", Function: FunctionDefinition{Name: "read_file", Description: "Read a file"}},
	}
	defs2 := []ToolDefinition{
		{Type: "function", Function: FunctionDefinition{Name: "read_file", Description: "Read a file"}},
		{Type: "function", Function: FunctionDefinition{Name: "write_file", Description: "Write a file"}},
	}
	shape1 := CaptureShape(system, defs1)
	shape2 := CaptureShape(system, defs2)
	diag := CompareShape(shape1, shape2)
	if !diag.PrefixChanged {
		t.Error("expected prefix change")
	}
	if len(diag.ChangeReasons) != 1 || diag.ChangeReasons[0] != "tools" {
		t.Errorf("expected change reason 'tools', got %v", diag.ChangeReasons)
	}
}

func TestSessionCache(t *testing.T) {
	var sc sessionCache
	sc.record(100, 200)
	sc.record(50, 150)
	hit, miss := sc.Snapshot()
	if hit != 150 {
		t.Errorf("expected hit=150, got %d", hit)
	}
	if miss != 350 {
		t.Errorf("expected miss=350, got %d", miss)
	}
}

// ─── Storm Breaker 测试 ──

func TestBatchStormSignature_AllFailed(t *testing.T) {
	calls := []ToolCall{
		{Function: FunctionCall{Name: "read_file"}},
		{Function: FunctionCall{Name: "write_file"}},
	}
	errMsgs := []string{"file not found", "permission denied"}
	blocked := []bool{false, false}
	sig, ok := BatchStormSignature(calls, errMsgs, blocked)
	if !ok {
		t.Fatal("expected ok=true for all failed calls")
	}
	if sig == "" {
		t.Error("expected non-empty signature")
	}
}

func TestBatchStormSignature_SomeBlocked(t *testing.T) {
	calls := []ToolCall{
		{Function: FunctionCall{Name: "read_file"}},
		{Function: FunctionCall{Name: "write_file"}},
	}
	errMsgs := []string{"file not found", ""}
	blocked := []bool{false, true}
	_, ok := BatchStormSignature(calls, errMsgs, blocked)
	if ok {
		t.Fatal("expected ok=false when some calls are blocked")
	}
}

func TestBatchStormSignature_SomeSuccess(t *testing.T) {
	calls := []ToolCall{
		{Function: FunctionCall{Name: "read_file"}},
		{Function: FunctionCall{Name: "write_file"}},
	}
	errMsgs := []string{"file not found", ""}
	blocked := []bool{false, false}
	_, ok := BatchStormSignature(calls, errMsgs, blocked)
	if ok {
		t.Fatal("expected ok=false when some calls succeed")
	}
}

func TestApplyStormBreaker(t *testing.T) {
	calls := []ToolCall{
		{ID: "1", Function: FunctionCall{Name: "read_file"}},
	}
	errMsgs := []string{"file not found"}
	blocked := []bool{false}
	results := []string{"Error: file not found"}

	storm := &StormSig{}
	// First call: under threshold
	results, triggered := ApplyStormBreaker(calls, results, errMsgs, blocked, storm)
	if triggered {
		t.Error("expected no trigger on first failure")
	}
	if storm.count != 1 {
		t.Errorf("expected count=1, got %d", storm.count)
	}
	// Second call (same sig): still under threshold
	results, triggered = ApplyStormBreaker(calls, results, errMsgs, blocked, storm)
	if triggered {
		t.Error("expected no trigger on second failure")
	}
	if storm.count != 2 {
		t.Errorf("expected count=2, got %d", storm.count)
	}
	// Third call: trigger
	results, triggered = ApplyStormBreaker(calls, results, errMsgs, blocked, storm)
	if !triggered {
		t.Fatal("expected trigger on third failure")
	}
	if !strings.Contains(results[0], "[循环防护]") {
		t.Error("expected loop guard message in results[0]")
	}
}

func TestApplyStormBreaker_ResetsOnSuccess(t *testing.T) {
	calls := []ToolCall{
		{ID: "1", Function: FunctionCall{Name: "read_file"}},
	}
	errMsgs := []string{"file not found"}
	blocked := []bool{false}
	results := []string{"Error: file not found"}

	storm := &StormSig{}
	ApplyStormBreaker(calls, results, errMsgs, blocked, storm)
	// Now simulate a successful call (different signature)
	errMsgs2 := []string{""}
	blocked2 := []bool{false}
	results2 := []string{"success"}
	ApplyStormBreaker(calls, results2, errMsgs2, blocked2, storm)
	if storm.count != 0 {
		t.Errorf("expected count reset to 0, got %d", storm.count)
	}
}

// ─── RepeatSuccessGuard 测试 ──

func TestRepeatSuccessGuard_ReadOnly(t *testing.T) {
	g := NewRepeatSuccessGuard()
	blocked, _ := g.Check("read_file", `{"path":"test.txt"}`, true)
	if blocked {
		t.Error("read-only tools should not be blocked")
	}
}

func TestRepeatSuccessGuard_BlocksAfterThreshold(t *testing.T) {
	g := NewRepeatSuccessGuard()
	args := `{"path":"test.txt","content":"hello"}`

	// Record 2 successful calls
	g.Record("write_file", args, false)
	g.Record("write_file", args, false)

	// Third call should be blocked
	blocked, msg := g.Check("write_file", args, false)
	if !blocked {
		t.Fatal("expected block on third identical call")
	}
	if !strings.Contains(msg, "[循环防护]") {
		t.Error("expected loop guard message")
	}
}

func TestRepeatSuccessGuard_DifferentArgs(t *testing.T) {
	g := NewRepeatSuccessGuard()
	g.Record("write_file", `{"path":"a.txt"}`, false)
	blocked, _ := g.Check("write_file", `{"path":"b.txt"}`, false)
	if blocked {
		t.Error("different args should not be blocked")
	}
}

func TestRepeatSuccessGuard_Reset(t *testing.T) {
	g := NewRepeatSuccessGuard()
	g.Record("write_file", `{}`, false)
	g.Record("write_file", `{}`, false)
	g.Reset()
	blocked, _ := g.Check("write_file", `{}`, false)
	if blocked {
		t.Error("after reset, calls should not be blocked")
	}
}

// ─── TruncateToolOutput 测试 ──

func TestTruncateToolOutput_UnderLimit(t *testing.T) {
	s := "short output"
	body, notice := TruncateToolOutput(s)
	if notice != "" {
		t.Errorf("expected no notice, got %q", notice)
	}
	if body != s {
		t.Errorf("expected body=%q, got %q", s, body)
	}
}

func TestTruncateToolOutput_OverLimit(t *testing.T) {
	s := strings.Repeat("a", maxToolOutputBytes+1000)
	body, notice := TruncateToolOutput(s)
	if notice == "" {
		t.Error("expected notice for truncated output")
	}
	if len(body) >= len(s) {
		t.Error("truncated body should be shorter than original")
	}
	if !strings.Contains(body, "[截断") {
		t.Error("expected truncation marker in body")
	}
}

// ─── EvidenceLedger 测试 ──

func TestEvidenceLedger(t *testing.T) {
	l := NewEvidenceLedger()
	if l.HasSuccessfulReceipt() {
		t.Error("new ledger should have no receipts")
	}
	l.Record("read_file", `{"path":"test.go"}`, true, true)
	if !l.HasSuccessfulReceipt() {
		t.Error("expected successful receipt")
	}
	idx, ok := l.LatestWriterIndex()
	if ok {
		t.Error("read-only tool should not be a writer")
	}
	l.Record("write_file", `{"path":"test.go"}`, true, false)
	idx, ok = l.LatestWriterIndex()
	if !ok {
		t.Fatal("expected writer index")
	}
	if idx != 1 {
		t.Errorf("expected index=1, got %d", idx)
	}
	l.Reset()
	if l.HasSuccessfulReceipt() {
		t.Error("after reset, should have no receipts")
	}
}

// ─── FinalReadiness 测试 ──

func TestCheckFinalReadiness_NoTodos(t *testing.T) {
	r := CheckFinalReadiness(nil)
	if r.HasTodos {
		t.Error("expected no todos")
	}
}

func TestCheckFinalReadiness_AllComplete(t *testing.T) {
	todos := []string{"Done: fixed bug", "Completed: added test", "✅ done"}
	r := CheckFinalReadiness(todos)
	if len(r.Incomplete) != 0 {
		t.Errorf("expected no incomplete, got %v", r.Incomplete)
	}
}

func TestCheckFinalReadiness_SomeIncomplete(t *testing.T) {
	todos := []string{"Fix the login bug", "Add unit tests"}
	r := CheckFinalReadiness(todos)
	if len(r.Incomplete) != 2 {
		t.Errorf("expected 2 incomplete, got %v", r.Incomplete)
	}
}

// ─── parseSSE 测试（流式中断恢复）──

func TestParseSSE_StreamInterrupted(t *testing.T) {
	// Simulate a partial stream that gets interrupted
	partialStream := "data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n"
	r := strings.NewReader(partialStream)
	msg, err := parseSSE(r, nil)
	if err != nil {
		t.Fatalf("parseSSE failed: %v", err)
	}
	if msg.Content != "hello world" {
		t.Errorf("expected content 'hello world', got %q", msg.Content)
	}
}

func TestParseSSE_FullStream(t *testing.T) {
	fullStream := "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\ndata: [DONE]\n\n"
	r := strings.NewReader(fullStream)
	msg, err := parseSSE(r, nil)
	if err != nil {
		t.Fatalf("parseSSE failed: %v", err)
	}
	if msg.Content != "hello world" {
		t.Errorf("expected content 'hello world', got %q", msg.Content)
	}
}

// ─── canonicalToolArgs 测试 ──

func TestCanonicalToolArgs(t *testing.T) {
	input := `  {"path":  "test.go" ,  "content":"hello"}  `
	expected := `{"content":"hello","path":"test.go"}`
	got := canonicalToolArgs(input)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestCanonicalToolArgs_InvalidJSON(t *testing.T) {
	// Invalid JSON should be returned as-is (trimmed)
	input := `not json at all`
	got := canonicalToolArgs(input)
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

// ─── RetryInfo 序列化测试 ──

func TestRetryInfoJSON(t *testing.T) {
	info := RetryInfo{Attempt: 1, Max: 10, Delay: 1000000000}
	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded RetryInfo
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Attempt != info.Attempt || decoded.Max != info.Max {
		t.Errorf("round-trip failed: %+v -> %+v", info, decoded)
	}
}

// ─── extractSystemPrompt 测试 ──

func TestExtractSystemPrompt(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "You are a helpful assistant."},
		{Role: RoleUser, Content: "Hello"},
	}
	prompt := extractSystemPrompt(msgs)
	if prompt != "You are a helpful assistant." {
		t.Errorf("expected system prompt, got %q", prompt)
	}
}

func TestExtractSystemPrompt_NoSystem(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "Hello"},
	}
	prompt := extractSystemPrompt(msgs)
	if prompt != "" {
		t.Errorf("expected empty, got %q", prompt)
	}
}

// ─── backoffDelay 测试 ──

func TestBackoffDelay(t *testing.T) {
	// First retry should be ~500ms + jitter
	d1 := backoffDelay(1, 0)
	if d1 < 500*time.Millisecond || d1 > 750*time.Millisecond {
		t.Errorf("delay 1 out of range: %v", d1)
	}
	// Third retry should be ~2000ms + jitter
	d3 := backoffDelay(3, 0)
	if d3 < 2000*time.Millisecond || d3 > 2250*time.Millisecond {
		t.Errorf("delay 3 out of range: %v", d3)
	}
	// Respect Retry-After
	d := backoffDelay(1, 5*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
	// Cap at maxBackoff
	d = backoffDelay(10, 0)
	if d > maxBackoff+300*time.Millisecond {
		t.Errorf("delay exceeded maxBackoff: %v", d)
	}
}
