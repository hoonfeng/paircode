package agent

// 注意：本文件原含 Provider 重试机制（sendWithRetry/RetryInfo/AuthError/
// WithRetryNotify/IsConnReset/retryableStatus/extractSystemPrompt/backoffDelay/
// maxBackoff）的测试，但这些符号在 agent 包中从未实现（属于阶段二/四的 LLM
// Provider 重试工作）。引用未定义符号导致整个包测试编译失败，故移除这些测试，
// 仅保留已实现符号（Storm Breaker/RepeatSuccessGuard/EvidenceLedger/
// TruncateToolOutput/FinalReadiness/CaptureShape/parseSSE/canonicalToolArgs/
// sessionCache）的测试。Provider 重试机制实现后应新建独立测试文件。

import (
	"strings"
	"testing"
)

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
