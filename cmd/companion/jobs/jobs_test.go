package jobs

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

func collectEvents() (func() []agent.Event, func(agent.Event)) {
	var mu sync.Mutex
	var evs []agent.Event
	return func() []agent.Event {
			mu.Lock()
			defer mu.Unlock()
			out := make([]agent.Event, len(evs))
			copy(out, evs)
			return out
		}, func(e agent.Event) {
			mu.Lock()
			evs = append(evs, e)
			mu.Unlock()
		}
}

func TestNewManager(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestStartAndKill(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()

	var ran atomic.Bool
	j := m.Start("test", "test-job", func(ctx context.Context, out io.Writer) (string, error) {
		ran.Store(true)
		<-ctx.Done()
		return "killed", nil
	})
	if j == nil {
		t.Fatal("Start returned nil")
	}
	if j.ID == "" {
		t.Fatal("job ID is empty")
	}

	time.Sleep(50 * time.Millisecond)
	if !ran.Load() {
		t.Fatal("job did not start running")
	}

	m.Kill(j.ID)
	time.Sleep(50 * time.Millisecond)
	_, status, ok := m.Output(j.ID)
	if !ok {
		t.Fatal("Output returned ok=false")
	}
	if status != Killed {
		t.Errorf("expected Killed, got %s", status)
	}
}

func TestOutput(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()

	j := m.Start("test", "test-output", func(ctx context.Context, out io.Writer) (string, error) {
		fmt.Fprint(out, "line1\n")
		fmt.Fprint(out, "line2\n")
		return "done", nil
	})

	m.Wait(context.Background(), []string{j.ID}, 5)
	text, status, ok := m.Output(j.ID)
	if !ok {
		t.Fatal("Output returned ok=false")
	}
	if status != Done {
		t.Errorf("expected Done, got %s", status)
	}
	if !strings.Contains(text, "line1") || !strings.Contains(text, "line2") {
		t.Errorf("expected output containing lines, got %q", text)
	}
}

func TestWait(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()

	j := m.Start("test", "test-wait", func(ctx context.Context, out io.Writer) (string, error) {
		return "completed", nil
	})

	results := m.Wait(context.Background(), []string{j.ID}, 5)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != Done {
		t.Errorf("expected Done, got %s", results[0].Status)
	}
	if results[0].Output != "completed" {
		t.Errorf("expected 'completed', got %q", results[0].Output)
	}
}

func TestDrainCompletedNote(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()

	m.Start("test", "drain-test", func(ctx context.Context, out io.Writer) (string, error) {
		return "done", nil
	})

	// Wait for completion
	results := m.Wait(context.Background(), nil, 5)
	if len(results) == 0 {
		t.Fatal("no results from Wait")
	}

	note := m.DrainCompletedNote()
	if note == "" {
		t.Fatal("expected non-empty note")
	}
	if !strings.Contains(note, "done") && !strings.Contains(note, "finished") {
		t.Errorf("expected note mentioning completion, got %q", note)
	}

	// Second drain should be empty
	note2 := m.DrainCompletedNote()
	if note2 != "" {
		t.Errorf("expected empty note on second drain, got %q", note2)
	}
}

func TestRunning(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()

	// Start a long-running job
	_ = m.Start("test", "long-run", func(ctx context.Context, out io.Writer) (string, error) {
		<-ctx.Done()
		return "", nil
	})

	time.Sleep(50 * time.Millisecond)
	running := m.Running()
	if len(running) != 1 {
		t.Errorf("expected 1 running job, got %d", len(running))
	}
}

func TestWithFromContext(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()

	ctx := WithManager(context.Background(), m)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext returned ok=false")
	}
	if got != m {
		t.Fatal("FromContext returned different manager")
	}

	_, ok = FromContext(context.Background())
	if ok {
		t.Fatal("FromContext on plain context should return ok=false")
	}
}

func TestMultipleJobs(t *testing.T) {
	_, onEvent := collectEvents()
	m := NewManager(onEvent)
	defer m.Close()

	for i := 0; i < 3; i++ {
		j := m.Start("test", fmt.Sprintf("job-%d", i), func(ctx context.Context, out io.Writer) (string, error) {
			return fmt.Sprintf("result-%d", i), nil
		})
		_ = j
	}

	results := m.Wait(context.Background(), nil, 5)
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}
