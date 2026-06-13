package provider

import (
	"strings"
	"testing"
)

func TestEffortCapabilityForModelKnown(t *testing.T) {
	cap := EffortCapabilityForModel("deepseek-v4-pro")
	if !cap.Supported {
		t.Fatal("deepseek-v4-pro should support effort")
	}
	if len(cap.Levels) < 2 {
		t.Fatalf("expected levels >= 2, got %v", cap.Levels)
	}
	if cap.Default != "high" {
		t.Fatalf("default = %q, want 'high'", cap.Default)
	}
}

func TestEffortCapabilityForModelUnknown(t *testing.T) {
	cap := EffortCapabilityForModel("unknown-model")
	if cap.Supported {
		t.Fatal("unknown model should not support effort")
	}
}

func TestNormalizeEffortAuto(t *testing.T) {
	result, err := NormalizeEffort("deepseek-v4-pro", "auto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("auto should return empty, got %q", result)
	}
}

func TestNormalizeEffortDeepSeek(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"high", "high"},
		{"max", "max"},
		{"low", "high"},   // maps to high
		{"medium", "high"}, // maps to high
		{"xhigh", "max"},   // maps to max
	}
	for _, tt := range tests {
		result, err := NormalizeEffort("deepseek-v4-pro", tt.input)
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
			continue
		}
		if result != tt.want {
			t.Errorf("input %q: got %q, want %q", tt.input, result, tt.want)
		}
	}
}

func TestNormalizeEffortInvalid(t *testing.T) {
	_, err := NormalizeEffort("deepseek-v4-pro", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid effort level")
	}
}

func TestNormalizeEffortEmptyInput(t *testing.T) {
	result, err := NormalizeEffort("deepseek-v4-pro", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("empty input should return empty (auto), got %q", result)
	}
}

func TestNormalizeEffortUnknownModel(t *testing.T) {
	_, err := NormalizeEffort("unknown", "high")
	if err == nil || !strings.Contains(err.Error(), "not configurable") {
		t.Fatalf("want 'not configurable' error, got %v", err)
	}
}

func TestEffortDisplay(t *testing.T) {
	if got := EffortDisplay(""); got != "auto" {
		t.Fatalf("empty = %q, want 'auto'", got)
	}
	if got := EffortDisplay("high"); got != "high" {
		t.Fatalf("'high' = %q, want 'high'", got)
	}
	if got := EffortDisplay("  MAX  "); got != "max" {
		t.Fatalf("'  MAX  ' = %q, want 'max'", got)
	}
}

func TestEffortDisplayAuto(t *testing.T) {
	if got := EffortDisplay("auto"); got != "auto" {
		t.Fatalf("'auto' = %q, want 'auto'", got)
	}
}

func TestRegisterModelCapability(t *testing.T) {
	RegisterModelCapability("test-model-v1", ReasoningProtocolOpenAI,
		[]string{"low", "medium", "high"}, "medium")
	cap := EffortCapabilityForModel("test-model-v1")
	if !cap.Supported {
		t.Fatal("registered model should support effort")
	}
	if cap.Default != "medium" {
		t.Fatalf("default = %q, want 'medium'", cap.Default)
	}
}
