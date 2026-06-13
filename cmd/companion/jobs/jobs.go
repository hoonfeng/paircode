// Package jobs is the session-scoped background-job registry for gou-ide's agent.
// A Manager owns a context whose lifetime is the session, NOT a single turn — so
// a job started in one turn keeps running across turns and is cancelled only when
// Close() is called. Tools reach the Manager through the call context via
// WithManager / FromContext.
//
// The Manager accumulates a one-line completion summary that the controller drains
// into the next turn (DrainCompletedNote) so the model itself learns of completions.
//
// 移植自 DeepSeek-Reasonix (internal/jobs/)，适配 gou-ide 的事件系统。
package jobs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

// Status is a job's lifecycle state.
type Status string

const (
	Running Status = "running"
	Done    Status = "done"
	Failed  Status = "failed"
	Killed  Status = "killed"
)

// View is a read-only snapshot of a job for the status bar.
type View struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	StartedAt int64  `json:"startedAt"` // unix milliseconds
}

// Result is one job's terminal (or current) state returned by Wait.
type Result struct {
	ID     string
	Kind   string
	Label  string
	Status Status
	Output string // the terminal result text, or the streamed buffer when no result was set
}

// Job is one background job. The mutex guards the streaming buffer and the
// terminal fields; the run goroutine writes them, readers (Output/Wait/snapshots)
// take the same lock.
type Job struct {
	ID    string
	Kind  string // "bash" | "task" | "build" | ...
	Label string

	mu         sync.Mutex
	buf        bytes.Buffer
	readOffset int
	status     Status
	result     string
	resultRead bool // result already surfaced by Output
	startedAt  int64
	cancel     context.CancelFunc
	done       chan struct{}
}

// Manager is the session's background-job table. It is safe for concurrent use.
// onEvent is the agent's event callback for emitting notices.
type Manager struct {
	onEvent func(agent.Event)
	root    context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	mu        sync.Mutex
	seq       int
	jobs      map[string]*Job
	order     []string
	completed []string // finished-job summaries awaiting drain into the next turn
}

// NewManager returns a Manager whose jobs run under a fresh session-scoped
// context (cancelled by Close). onEvent receives job-lifecycle notices; pass
// the agent's OnEvent callback (must be goroutine-safe, e.g. via bridge's
// pushEvent).
func NewManager(onEvent func(agent.Event)) *Manager {
	if onEvent == nil {
		onEvent = func(agent.Event) {}
	}
	root, cancel := context.WithCancel(context.Background())
	return &Manager{onEvent: onEvent, root: root, cancel: cancel, jobs: map[string]*Job{}}
}

// jobWriter appends a job's streamed output under its lock so a concurrent
// Output read never races the producing goroutine.
type jobWriter struct{ j *Job }

func (w jobWriter) Write(p []byte) (int, error) {
	w.j.mu.Lock()
	defer w.j.mu.Unlock()
	return w.j.buf.Write(p)
}

// Start launches run on a goroutine under the manager's session context and
// returns the job immediately. run streams output to the writer and returns the
// terminal result text (a task's final answer; a build job streams everything to
// the buffer and returns ""). The job is marked killed when its context was
// cancelled, failed on any other error, else done.
func (m *Manager) Start(kind, label string, run func(ctx context.Context, out io.Writer) (string, error)) *Job {
	m.mu.Lock()
	m.seq++
	id := fmt.Sprintf("%s-%d", kind, m.seq)
	ctx, cancel := context.WithCancel(m.root)
	j := &Job{ID: id, Kind: kind, Label: label, status: Running, startedAt: nowMs(), cancel: cancel, done: make(chan struct{})}
	m.jobs[id] = j
	m.order = append(m.order, id)
	m.mu.Unlock()

	m.onEvent(agent.Event{Type: agent.EventNotice, Content: fmt.Sprintf("background %s started: %s", kind, id)})

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		result, err := run(ctx, jobWriter{j})

		var st Status
		switch {
		case ctx.Err() != nil:
			st = Killed
		case err != nil:
			st = Failed
			if result == "" {
				result = err.Error()
			}
		default:
			st = Done
		}

		m.recordCompletion(id, kind, label, st, err)

		j.mu.Lock()
		j.result = result
		if j.status != Killed {
			j.status = st
		}
		j.mu.Unlock()
		close(j.done)
	}()
	return j
}

// recordCompletion queues the finished-job summary for DrainCompletedNote and
// emits a closing notice.
func (m *Manager) recordCompletion(id, kind, label string, st Status, err error) {
	tag := id
	if label != "" {
		tag = fmt.Sprintf("%s (%s)", id, label)
	}
	m.mu.Lock()
	m.completed = append(m.completed, fmt.Sprintf("%s — %s", tag, st))
	m.mu.Unlock()

	text := fmt.Sprintf("background %s finished: %s", kind, id)
	evtType := agent.EventNotice
	switch st {
	case Failed:
		text = fmt.Sprintf("background %s failed: %s — %v", kind, id, err)
	case Killed:
		text = fmt.Sprintf("background %s killed: %s", kind, id)
	}
	m.onEvent(agent.Event{Type: evtType, Content: text})
}

func (m *Manager) get(id string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[id]
}

// Output returns the job's output produced since the last Output call plus its
// current status. ok is false when the id is unknown.
func (m *Manager) Output(id string) (text string, status Status, ok bool) {
	j := m.get(id)
	if j == nil {
		return "", "", false
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	full := j.buf.String()
	text = full[j.readOffset:]
	j.readOffset = len(full)
	if text == "" && j.status != Running && j.result != "" && !j.resultRead {
		text = j.result
		j.resultRead = true
	}
	return text, j.status, true
}

// Kill cancels a running job. Returns false when the id is unknown or the job
// has already finished.
func (m *Manager) Kill(id string) bool {
	j := m.get(id)
	if j == nil {
		return false
	}
	j.mu.Lock()
	running := j.status == Running
	if running {
		j.status = Killed
	}
	j.mu.Unlock()
	if !running {
		return false
	}
	j.cancel()
	return true
}

// Wait blocks until the named jobs (or every currently-running job when ids is
// empty) reach a terminal state, or ctx is cancelled, or timeoutSec elapses
// (0 = no timeout). It returns each target's snapshot regardless of why it
// returned, so a timeout still reports partial progress.
func (m *Manager) Wait(ctx context.Context, ids []string, timeoutSec int) []Result {
	targets := m.resolve(ids)
	if len(targets) == 0 {
		return nil
	}
	var timeout <-chan time.Time
	if timeoutSec > 0 {
		t := time.NewTimer(time.Duration(timeoutSec) * time.Second)
		defer t.Stop()
		timeout = t.C
	}
	for _, j := range targets {
		select {
		case <-j.done:
		case <-ctx.Done():
			return m.results(targets)
		case <-timeout:
			return m.results(targets)
		}
	}
	return m.results(targets)
}

// resolve maps requested ids to jobs; an empty list selects all jobs
// (both running and completed).
func (m *Manager) resolve(ids []string) []*Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*Job
	if len(ids) == 0 {
		for _, id := range m.order {
			if j := m.jobs[id]; j != nil {
				out = append(out, j)
			}
		}
		return out
	}
	for _, id := range ids {
		if j := m.jobs[id]; j != nil {
			out = append(out, j)
		}
	}
	return out
}

func (m *Manager) results(targets []*Job) []Result {
	out := make([]Result, 0, len(targets))
	for _, j := range targets {
		j.mu.Lock()
		text := j.result
		if text == "" {
			text = j.buf.String()
		}
		out = append(out, Result{ID: j.ID, Kind: j.Kind, Label: j.Label, Status: j.status, Output: text})
		j.mu.Unlock()
	}
	return out
}

// Running returns a snapshot of the still-running jobs (for the status bar).
func (m *Manager) Running() []View {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []View
	for _, id := range m.order {
		j := m.jobs[id]
		j.mu.Lock()
		if j.status == Running {
			out = append(out, View{ID: j.ID, Kind: j.Kind, Label: j.Label, Status: string(j.status), StartedAt: j.startedAt})
		}
		j.mu.Unlock()
	}
	return out
}

// DrainCompletedNote returns (and clears) a one-line summary of jobs that
// finished since the last drain, for the controller to fold into the next turn
// so the model learns of completions. "" when nothing finished.
func (m *Manager) DrainCompletedNote() string {
	m.mu.Lock()
	c := m.completed
	m.completed = nil
	m.mu.Unlock()
	if len(c) == 0 {
		return ""
	}
	return "Background jobs finished since your last message: " + strings.Join(c, "; ") +
		". Read their output with job_output or wait if you still need it."
}

// Close cancels the session context and waits for every background job
// goroutine to return before unblocking.
func (m *Manager) Close() {
	m.cancel()
	m.wg.Wait()
}

func nowMs() int64 { return time.Now().UnixMilli() }

// --- call-context injection ---

type ctxKey struct{}

// WithManager stamps ctx with the job manager so tools can reach it via
// FromContext. The agent sets this on every tool call's context.
func WithManager(ctx context.Context, m *Manager) context.Context {
	return context.WithValue(ctx, ctxKey{}, m)
}

// FromContext returns the job manager set by the agent, if any. ok is false for
// a plain context (headless tests, calls outside the run loop).
func FromContext(ctx context.Context) (*Manager, bool) {
	m, ok := ctx.Value(ctxKey{}).(*Manager)
	return m, ok && m != nil
}
