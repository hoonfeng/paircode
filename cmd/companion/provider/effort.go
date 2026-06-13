package provider

import (
	"fmt"
	"strings"
)

// Reasoning protocol constants.
const (
	ReasoningProtocolAuto     = "auto"
	ReasoningProtocolDeepSeek = "deepseek"
	ReasoningProtocolOpenAI   = "openai"
	ReasoningProtocolNone     = "none"
)

// EffortCapability describes the abstract effort levels a provider/model can set.
type EffortCapability struct {
	Supported bool
	Levels    []string
	Default   string
}

// modelCapability describes the reasoning capability of a known model.
type modelCapability struct {
	Protocol string
	Levels   []string
	Default  string
}

// knownModelCapabilities maps known model IDs to their reasoning capabilities.
var knownModelCapabilities = map[string]modelCapability{}

// RegisterModelCapability registers a model's reasoning capability.
// Provider implementations can call this in init().
func RegisterModelCapability(model, protocol string, levels []string, def string) {
	knownModelCapabilities[model] = modelCapability{
		Protocol: protocol,
		Levels:   levels,
		Default:  def,
	}
}

func init() {
	RegisterModelCapability("deepseek-v4-flash", ReasoningProtocolDeepSeek,
		[]string{"high", "max"}, "high")
	RegisterModelCapability("deepseek-v4-pro", ReasoningProtocolDeepSeek,
		[]string{"high", "max"}, "high")
}

// EffortCapabilityForModel returns the effort levels available for a model.
func EffortCapabilityForModel(model string) EffortCapability {
	if cap, ok := knownModelCapabilities[model]; ok {
		levels := make([]string, 0, len(cap.Levels)+1)
		levels = append(levels, "auto")
		levels = append(levels, cap.Levels...)
		return EffortCapability{Supported: true, Levels: levels, Default: cap.Default}
	}
	return EffortCapability{}
}

// NormalizeEffort maps a user-supplied effort level into a normalized value.
// Empty means auto/provider default.
func NormalizeEffort(model, raw string) (string, error) {
	level := normalizeEffortLevel(raw)
	if level == "" {
		return "", fmt.Errorf("usage: /effort auto|<level>")
	}
	if level == "auto" {
		return "", nil
	}
	if cap, ok := knownModelCapabilities[model]; ok {
		for _, l := range cap.Levels {
			if level == l {
				return level, nil
			}
		}
		switch {
		case hasString(cap.Levels, "high") && hasString(cap.Levels, "max"):
			// DeepSeek-style: map low/medium→high, xhigh→max
			switch level {
			case "low", "medium":
				return "high", nil
			case "xhigh":
				return "max", nil
			}
		case hasString(cap.Levels, "low") && hasString(cap.Levels, "medium") && hasString(cap.Levels, "high"):
			// OpenAI-style: direct mapping
			switch level {
			case "low", "medium", "high":
				return level, nil
			case "xhigh", "max":
				return "high", nil
			}
		}
	}
	return "", fmt.Errorf("effort not configurable for model %q", model)
}

// EffortDisplay returns the selected effort level, using "auto" for default.
func EffortDisplay(effort string) string {
	if strings.TrimSpace(effort) == "" {
		return "auto"
	}
	return normalizeEffortLevel(effort)
}

func normalizeEffortLevel(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "auto", "":
		return "auto"
	case "off":
		return "off"
	case "low":
		return "low"
	case "medium", "med":
		return "medium"
	case "high":
		return "high"
	case "xhigh", "extra-high", "extra_high":
		return "xhigh"
	case "max", "maximum":
		return "max"
	case "disabled":
		return "disabled"
	case "adaptive":
		return "adaptive"
	default:
		return ""
	}
}

func hasString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
