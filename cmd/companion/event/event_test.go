package event

import (
	"sync"
	"testing"
)

func TestKindValues(t *testing.T) {
	tests := []struct {
		kind Kind
		name string
	}{
		{TurnStarted, "TurnStarted"},
		{Reasoning, "Reasoning"},
		{Text, "Text"},
		{Message, "Message"},
		{ToolDispatch, "ToolDispatch"},
		{ToolResult, "ToolResult"},
		{UsageEvent, "UsageEvent"},
		{Notice, "Notice"},
		{Phase, "Phase"},
		{ApprovalRequest, "ApprovalRequest"},
		{AskRequest, "AskRequest"},
		{TurnDone, "TurnDone"},
		{CompactionStarted, "CompactionStarted"},
		{CompactionDone, "CompactionDone"},
		{ToolProgress, "ToolProgress"},
		{Retrying, "Retrying"},
		{Steer, "Steer"},
	}
	if len(tests) != 17 {
		t.Errorf("expected 17 kinds, got %d", len(tests))
	}
	for _, tt := range tests {
		if int(tt.kind) < 0 || int(tt.kind) >= len(tests) {
			t.Errorf("kind %s has invalid value %d", tt.name, tt.kind)
		}
	}
}

func TestSinkDiscard(t *testing.T) {
	Discard.Emit(Event{Kind: Text, Text: "hello"})
	// Should not panic
}

func TestFuncSink(t *testing.T) {
	var mu sync.Mutex
	var got Event
	fn := FuncSink(func(e Event) {
		mu.Lock()
		got = e
		mu.Unlock()
	})
	fn.Emit(Event{Kind: Text, Text: "world"})
	mu.Lock()
	if got.Text != "world" {
		t.Errorf("expected 'world', got %q", got.Text)
	}
	mu.Unlock()
}

func TestNilFuncSink(t *testing.T) {
	var fn FuncSink // nil
	fn.Emit(Event{Kind: Text}) // should not panic
}

func TestSync(t *testing.T) {
	var mu sync.Mutex
	var count int
	inner := FuncSink(func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})
	s := Sync(inner)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Emit(Event{Kind: Text})
		}()
	}
	wg.Wait()

	if count != 10 {
		t.Errorf("expected 10 events, got %d", count)
	}
}

func TestSyncNil(t *testing.T) {
	s := Sync(nil)
	if s == nil {
		t.Error("Sync(nil) should return Discard, not nil")
	}
	s.Emit(Event{Kind: Text}) // should not panic
}

func TestPricingCost(t *testing.T) {
	p := &Pricing{CacheHit: 1.0, Input: 2.0, Output: 3.0}
	u := &Usage{CacheHitTokens: 100, CacheMissTokens: 200, CompletionTokens: 300}
	cost := p.Cost(u)
	expected := (100*1.0 + 200*2.0 + 300*3.0) / 1e6
	if cost != expected {
		t.Errorf("expected %f, got %f", expected, cost)
	}
	// nil pricing
	if (*Pricing)(nil).Cost(u) != 0 {
		t.Error("nil Pricing.Cost should return 0")
	}
	// nil usage
	if p.Cost(nil) != 0 {
		t.Error("Cost(nil) should return 0")
	}
}

func TestPricingSymbol(t *testing.T) {
	if (*Pricing)(nil).Symbol() != "¥" {
		t.Error("nil Pricing.Symbol should return ¥")
	}
	p := &Pricing{Currency: "$"}
	if p.Symbol() != "$" {
		t.Errorf("expected $, got %s", p.Symbol())
	}
}

func TestFileDiff(t *testing.T) {
	fd := FileDiff{Diff: "--- a/foo\n+++ b/foo\n@@ -1 +1 @@\n-old\n+new", Added: 1, Removed: 1}
	if fd.Added != 1 || fd.Removed != 1 {
		t.Errorf("FileDiff fields mismatch")
	}
}

func TestToolFields(t *testing.T) {
	tool := Tool{
		ID:       "call_123",
		Name:     "read_file",
		Args:     `{"path":"foo.go"}`,
		ReadOnly: true,
		Partial:  false,
		Profile:  &Profile{Model: "deepseek", Effort: "high"},
	}
	if tool.ID != "call_123" || tool.Name != "read_file" || !tool.ReadOnly {
		t.Errorf("Tool fields mismatch")
	}
	if tool.Profile.Model != "deepseek" {
		t.Errorf("Profile.Model mismatch")
	}
}

func TestApproval(t *testing.T) {
	a := Approval{ID: "req_1", Tool: "write_file", Subject: "write foo.go"}
	if a.ID != "req_1" || a.Tool != "write_file" {
		t.Errorf("Approval fields mismatch")
	}
}

func TestAskRoundTrip(t *testing.T) {
	ask := Ask{
		ID: "ask_1",
		Questions: []AskQuestion{
			{
				ID:     "q1",
				Header: "确认",
				Prompt: "要执行吗？",
				Options: []AskOption{
					{Label: "是", Description: "执行操作"},
					{Label: "否"},
				},
				Multi: false,
			},
		},
	}
	if len(ask.Questions) != 1 {
		t.Errorf("expected 1 question, got %d", len(ask.Questions))
	}
	if ask.Questions[0].Options[0].Label != "是" {
		t.Errorf("expected '是', got %s", ask.Questions[0].Options[0].Label)
	}
}

func TestCompaction(t *testing.T) {
	c := Compaction{Trigger: "auto", Messages: 15, Summary: "briefing", Archive: "archive.log"}
	if c.Trigger != "auto" || c.Messages != 15 {
		t.Errorf("Compaction fields mismatch")
	}
}

func TestEventStruct(t *testing.T) {
	e := Event{
		Kind:      Text,
		Text:      "hello",
		Reasoning: "thinking...",
	}
	if e.Kind != Text || e.Text != "hello" {
		t.Errorf("Event fields mismatch")
	}
}

func TestLevelValues(t *testing.T) {
	if LevelInfo != 0 || LevelWarn != 1 {
		t.Errorf("Level values: info=%d, warn=%d", LevelInfo, LevelWarn)
	}
}

func TestAskAnswer(t *testing.T) {
	aa := AskAnswer{QuestionID: "q1", Selected: []string{"是"}}
	if aa.QuestionID != "q1" || len(aa.Selected) != 1 {
		t.Errorf("AskAnswer fields mismatch")
	}
}

func TestCacheDiagnostics(t *testing.T) {
	cd := CacheDiagnostics{
		PrefixHash:          "abc123",
		PrefixChanged:       true,
		PrefixChangeReasons: []string{"system"},
	}
	if !cd.PrefixChanged || len(cd.PrefixChangeReasons) != 1 {
		t.Errorf("CacheDiagnostics fields mismatch")
	}
}
