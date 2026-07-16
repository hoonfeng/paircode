package permission

import (
	"context"
	"encoding/json"
	"testing"
)

func TestParseDecision(t *testing.T) {
	tests := []struct {
		input string
		want  Decision
	}{
		{"allow", Allow},
		{"ALLOW", Allow},
		{"deny", Deny},
		{"Deny", Deny},
		{"ask", Ask},
		{"", Ask},
		{"unknown", Ask},
	}
	for _, tt := range tests {
		got := ParseDecision(tt.input)
		if got != tt.want {
			t.Errorf("ParseDecision(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseRule(t *testing.T) {
	tests := []struct {
		input   string
		wantOK  bool
		want    Rule
	}{
		{"write_file", true, Rule{Tool: "write_file"}},
		{"write_file(/etc/*)", true, Rule{Tool: "write_file", Subject: "/etc/*"}},
		{"shell_exec(go build*)", true, Rule{Tool: "shell_exec", Subject: "go build*"}},
		{"edit_file=main.go", true, Rule{Tool: "edit_file", Subject: "main.go", Literal: true}},
		{"", false, Rule{}},
		{"  read_file  ", true, Rule{Tool: "read_file"}},
	}
	for _, tt := range tests {
		got, ok := ParseRule(tt.input)
		if ok != tt.wantOK {
			t.Errorf("ParseRule(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
		}
		if got.Tool != tt.want.Tool || got.Subject != tt.want.Subject || got.Literal != tt.want.Literal {
			t.Errorf("ParseRule(%q) = %+v, want %+v", tt.input, got, tt.want)
		}
	}
}

func TestPolicy_Decide(t *testing.T) {
	p := New("ask",
		[]string{"read_file", "write_file(/tmp/*)"},
		[]string{"shell_exec"},
		[]string{"delete_file"},
	)

	tests := []struct {
		name     string
		tool     string
		readOnly bool
		args     json.RawMessage
		want     Decision
	}{
		{"deny wins", "delete_file", false, json.RawMessage(`{}`), Deny},
		{"ask for shell", "shell_exec", false, json.RawMessage(`{"command":"rm -rf"}`), Ask},
		{"allow read-only", "read_file", true, json.RawMessage(`{}`), Allow},
		{"allow write to /tmp", "write_file", false, json.RawMessage(`{"file_path":"/tmp/x.txt"}`), Allow},
		{"ask other write", "write_file", false, json.RawMessage(`{"file_path":"/etc/x.txt"}`), Ask},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Decide(tt.tool, tt.readOnly, tt.args)
			if got != tt.want {
				t.Errorf("Decide(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestGate_Check(t *testing.T) {
	p := New("ask", nil, []string{"shell_exec"}, []string{"delete_file"})
	g := NewGate(p, nil) // nil Approver = auto-allow for Ask

	// Deny
	allowed, reason, err := g.Check(context.Background(), "delete_file", json.RawMessage(`{}`), false)
	if allowed || err != nil {
		t.Errorf("deny: got allowed=%v, err=%v, want denied", allowed, err)
	}
	if reason == "" {
		t.Error("deny reason should not be empty")
	}

	// Ask with nil Approver → allowed
	allowed, reason, err = g.Check(context.Background(), "shell_exec", json.RawMessage(`{"command":"go build"}`), false)
	if !allowed || err != nil {
		t.Errorf("ask+nil approver: got allowed=%v, err=%v, want allowed", allowed, err)
	}

	// Allow
	allowed, reason, err = g.Check(context.Background(), "read_file", json.RawMessage(`{"path":"main.go"}`), true)
	if !allowed || err != nil {
		t.Errorf("allow: got allowed=%v, err=%v, want allowed", allowed, err)
	}
}

func TestSubject(t *testing.T) {
	tests := []struct {
		args json.RawMessage
		want string
	}{
		{json.RawMessage(`{"command":"go build"}`), "go build"},
		{json.RawMessage(`{"file_path":"main.go"}`), "main.go"},
		{json.RawMessage(`{"path":"/etc/hosts"}`), "/etc/hosts"},
		{json.RawMessage(`{"pattern":"*.go"}`), "*.go"},
		{json.RawMessage(`{"name":"my-skill"}`), "my-skill"},
		{json.RawMessage(`{}`), ""},
		{nil, ""},
	}
	for _, tt := range tests {
		got := Subject(tt.args)
		if got != tt.want {
			t.Errorf("Subject(%s) = %q, want %q", string(tt.args), got, tt.want)
		}
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.ts", false},
		{"go*", "go build", true},
		{"/etc/*", "/etc/hosts", true},
		{"/etc/*", "/var/log", false},
		{"rm -rf*", "rm -rf /", true},
		{"test?", "test1", true},
		{"test?", "test12", false},
		{"*", "anything", true},
	}
	for _, tt := range tests {
		got := matchGlob(tt.pattern, tt.name)
		if got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}

func TestIsFileMutationTool(t *testing.T) {
	tests := []struct {
		tool string
		want bool
	}{
		{"write_file", true},
		{"edit_file", true},
		{"delete_file", true},
		{"write_file_approve", false},
		{"read_file", false},
		{"shell_exec", false},
		{"search_content", false},
	}
	for _, tt := range tests {
		got := IsFileMutationTool(tt.tool)
		if got != tt.want {
			t.Errorf("IsFileMutationTool(%q) = %v, want %v", tt.tool, got, tt.want)
		}
	}
}

func TestGate_WithApprover(t *testing.T) {
	p := New("deny", nil, []string{"write_file"}, nil)
	approved := false
	approver := &mockApprover{
		fn: func(ctx context.Context, toolName, subject string, args json.RawMessage) (bool, bool, error) {
			approved = true
			return true, false, nil
		},
	}
	g := NewGate(p, approver)

	allowed, _, err := g.Check(context.Background(), "write_file", json.RawMessage(`{"file_path":"test.txt"}`), false)
	if !allowed || err != nil {
		t.Errorf("expected allowed, got allowed=%v, err=%v", allowed, err)
	}
	if !approved {
		t.Error("approver was not called")
	}
}

type mockApprover struct {
	fn func(ctx context.Context, toolName, subject string, args json.RawMessage) (bool, bool, error)
}

func (m *mockApprover) Approve(ctx context.Context, toolName, subject string, args json.RawMessage) (bool, bool, error) {
	return m.fn(ctx, toolName, subject, args)
}
