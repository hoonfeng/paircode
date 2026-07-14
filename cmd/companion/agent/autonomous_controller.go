// 自主控制器辅助：外层规划工具注册 + 编辑距离等工具
//
//go:build windows

package agent

import (
	"math"
	"strings"
)

// ── 编辑距离计算（用于任务相似度检测，防重复） ──

func editDistance(a, b string) int {
	m, n := len(a), len(b)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}
	prev := make([]int, n+1)
	curr := make([]int, n+1)
	for j := 0; j <= n; j++ {
		prev[j] = j
	}
	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[n]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func taskSimilarity(a, b string) float64 {
	if a == "" && b == "" {
		return 1
	}
	if a == "" || b == "" {
		return 0
	}
	norm := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r >= 0x4e00 {
				b.WriteRune(r)
			} else if r >= 'A' && r <= 'Z' {
				b.WriteRune(r - 'A' + 'a')
			}
		}
		ns := b.String()
		if len(ns) > 100 {
			ns = ns[:100]
		}
		return ns
	}
	sa, sb := norm(a), norm(b)
	if sa == "" && sb == "" {
		return 1
	}
	if sa == "" || sb == "" {
		return 0
	}
	dist := editDistance(sa, sb)
	maxLen := math.Max(float64(len(sa)), float64(len(sb)))
	if maxLen > 0 {
		return 1 - float64(dist)/maxLen
	}
	return 1
}

func RegisterPlanOnlyTools(r *Registry) {
	registerPlanTool(r)
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
