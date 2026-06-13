package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvalRecordStore(t *testing.T) {
	root := t.TempDir()

	store := GetEvalStore(root)
	if store == nil {
		t.Fatal("GetEvalStore returned nil")
	}
	if store.Count() != 0 {
		t.Fatalf("expected 0 records, got %d", store.Count())
	}

	// Append a record
	id := store.Append(&EvalRecord{
		Task:   "Test task",
		Total:  85,
		Scores: EvalScores{Completion: 32, Correctness: 25, Depth: 18, Efficiency: 10},
		Strengths:  []string{"good plan"},
		Weaknesses: []string{"missing test"},
		Feedback:   "decent",
		ToolCalls:  12,
	})
	if id == "" {
		t.Fatal("expected non-empty id")
	}

	if store.Count() != 1 {
		t.Fatalf("expected 1 record, got %d", store.Count())
	}

	// Stats
	st := store.Stats()
	if st.TotalRecords != 1 {
		t.Fatalf("expected 1 in stats, got %d", st.TotalRecords)
	}
	if st.AvgTotal != 85 {
		t.Fatalf("expected avg 85, got %.1f", st.AvgTotal)
	}

	// Mark iteration
	store.MarkIteration(1)
	last := store.LastRecord()
	if last == nil || last.Iteration != 1 {
		t.Fatal("expected iteration=1")
	}

	// Recent
	recent := store.Recent(5)
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent, got %d", len(recent))
	}

	// Check persistence: new store instance
	store2 := GetEvalStore(root)
	if store2.Count() != 1 {
		t.Fatalf("expected 1 record after reload, got %d", store2.Count())
	}

	// AnalyzeOptimization (needs >=2 records)
	hints := store.AnalyzeOptimization()
	if hints != nil {
		t.Logf("got %d hints with 1 record (should be nil):", len(hints))
	}

	// Optimization only kicks in with >=2 records
	store.Append(&EvalRecord{Total: 55, Scores: EvalScores{Completion: 20, Correctness: 15, Depth: 12, Efficiency: 8},
		Weaknesses: []string{"missing test", "incomplete"}})
	hints = store.AnalyzeOptimization()
	if len(hints) > 0 {
		t.Logf("optimization hints: %d", len(hints))
		for _, h := range hints {
			t.Logf("  [%s/%s] %s: %s", h.Dimension, h.Severity, h.Suggestion, h.Reason)
		}
	}

	// All records (reverse chronological)
	all := store.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 all records, got %d", len(all))
	}

	// Cleanup
	os.RemoveAll(filepath.Join(root, evalsDirName))
}

func TestEvalRecordStoreEmpty(t *testing.T) {
	ResetEvalStoreForTest() // 重置全局单例，新 root 创建新 store
	root := t.TempDir()
	store := GetEvalStore(root)
	if store == nil {
		t.Fatal("store is nil")
	}
	if st := store.Stats(); st.TotalRecords != 0 {
		t.Fatalf("expected 0, got %d", st.TotalRecords)
	}
	if hints := store.AnalyzeOptimization(); hints != nil {
		t.Fatalf("expected nil hints for empty store")
	}
	if all := store.All(); len(all) != 0 {
		t.Fatalf("expected empty all")
	}
}

// TestEvalPreview verifies the task preview truncation
func TestEvalPreview(t *testing.T) {
	short := "short task"
	if s := TaskPreview(short); s != short {
		t.Fatalf("expected '%s', got '%s'", short, s)
	}

	long := ""
	for i := 0; i < 50; i++ {
		long += "word "
	}
	preview := TaskPreview(long)
	if len([]rune(preview)) > 210 {
		t.Fatalf("preview too long: %d chars", len([]rune(preview)))
	}
}
