package codegraph

import (
	"path/filepath"
	"testing"
)

func TestCalcCyclomaticComplexity(t *testing.T) {
	tests := []struct {
		source   string
		name     string
		expected int // >= 基线 1
	}{
		{"func simple() {}", "无分支", 1},
		{"func branch() {\n\tif true {}\n}", "单个if", 2},
		{"func multi() {\n\tif a {}\n\tif b {}\n\tfor i:=0;; {}\n}", "多分支", 4},
		{"func complex() {\n\tif a && b {}\n\tif c || d {}\n\tfor _,_:= range x {}\n\tswitch y {\n\tcase 1:\n\tcase 2:\n\t}\n}", "复杂分支", 8},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calcCyclomaticComplexity(tc.source)
			if got < tc.expected {
				t.Errorf("calcCyclomaticComplexity(%q) = %d, 期望 >= %d", tc.name, got, tc.expected)
			}
		})
	}
}

func TestComplexityGrade(t *testing.T) {
	tests := []struct {
		complexity int
		grade      string
	}{
		{1, "A"},
		{5, "A"},
		{6, "B"},
		{10, "B"},
		{11, "C"},
		{20, "C"},
		{21, "D"},
		{30, "D"},
		{31, "E"},
		{100, "E"},
	}

	for _, tc := range tests {
		t.Run(tc.grade, func(t *testing.T) {
			got := complexityGrade(tc.complexity)
			if got != tc.grade {
				t.Errorf("complexityGrade(%d) = %s, 期望 %s", tc.complexity, got, tc.grade)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	// 4000 chars ≈ 1000 tokens
	got := estimateTokens("hello world")
	if got != 2 { // 11 chars / 4 = 2
		t.Errorf("estimateTokens = %d, 期望 2", got)
	}
}

func TestReadFileLines(t *testing.T) {
	// 用当前源文件测试读取行
	root := filepath.Dir(".")
	content, err := readFileLines(".", filepath.Join(root, "query_advanced_test.go"), 1, 5)
	if err != nil {
		t.Fatalf("readFileLines 失败: %v", err)
	}
	if content == "" {
		t.Error("readFileLines 返回空内容")
	}
}

func TestMemoryBrief(t *testing.T) {
	mb := MemoryBrief{
		Title:   "测试记忆",
		Summary: "这是一个测试",
		Tags:    []string{"test", "go"},
	}
	if mb.Title != "测试记忆" {
		t.Errorf("MemoryBrief.Title 设置失败")
	}
	if len(mb.Tags) != 2 {
		t.Errorf("MemoryBrief.Tags 应为 2 个")
	}
}
