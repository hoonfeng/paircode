package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeImage(t *testing.T) {
	// 使用项目根目录的图片做端到端测试
	imgPath := filepath.Join("..", "..", "..", "1afd3ed5d7bc107f83241f13b15ead0c.jpg")
	result, err := analyzeImage(imgPath, "high", 8)
	if err != nil {
		t.Fatalf("analyzeImage 失败: %v", err)
	}
	if result == "" {
		t.Fatal("analyzeImage 返回空结果")
	}
	t.Logf("分析结果:\n%s", result)

	// 验证关键信息存在
	checks := []string{"图色分析结果", "颜色分布", "×", "px", "#"}
	for _, c := range checks {
		if !strings.Contains(result, c) {
			t.Errorf("结果中未找到期望内容: %s", c)
		}
	}
}

func TestAnalyzeImageLowDetail(t *testing.T) {
	imgPath := filepath.Join("..", "..", "..", "1afd3ed5d7bc107f83241f13b15ead0c.jpg")
	result, err := analyzeImage(imgPath, "low", 5)
	if err != nil {
		t.Fatalf("analyzeImage (low) 失败: %v", err)
	}
	if result == "" {
		t.Fatal("analyzeImage (low) 返回空结果")
	}
	t.Logf("概要分析结果:\n%s", result)

	if !strings.Contains(result, "图色分析结果") {
		t.Errorf("概要结果中未找到标题")
	}
}

func TestAnalyzeImageNonExistent(t *testing.T) {
	_, err := analyzeImage("nonexistent.png", "high", 8)
	if err == nil {
		t.Fatal("期望对不存在的文件返回错误，但未出错")
	}
}

func TestColorNameFn(t *testing.T) {
	tests := []struct {
		r, g, b uint32
		want    string
	}{
		{255, 255, 255, "白色"},
		{0, 0, 0, "黑色"},
		{255, 0, 0, "红色"},
		{0, 255, 0, "绿色"},
		{0, 0, 255, "蓝色"},
		{128, 128, 128, "中灰"},
		{200, 200, 200, "浅灰"},
		{255, 255, 0, "黄色"},
	}
	for _, tt := range tests {
		got := colorName(tt.r, tt.g, tt.b)
		if got != tt.want {
			t.Errorf("colorName(%d,%d,%d) = %q, 期望 %q", tt.r, tt.g, tt.b, got, tt.want)
		}
	}
}

func TestOcrTextParser(t *testing.T) {
	// 模拟 Tesseract TSV 输出
	tsv := `level	page_num	block_num	par_num	line_num	word_num	left	top	width	height	conf	text
1	1	0	0	0	0	0	0	500	500	-1	
2	1	1	0	0	0	100	100	300	50	-1	
3	1	1	1	0	0	100	100	300	50	-1	
4	1	1	1	1	0	100	100	300	50	-1	
5	1	1	1	1	1	100	100	200	30	95	Hello
5	1	1	1	1	2	300	100	100	30	90	World
`
	results := parseTesseractTSV(tsv)
	if len(results) == 0 {
		t.Fatal("parseTesseractTSV 返回空结果")
	}
	// 应该合并为一行 "Hello World"
	found := false
	for _, r := range results {
		if strings.Contains(r.Text, "Hello") && strings.Contains(r.Text, "World") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("未找到合并后的 Hello World，结果: %+v", results)
	}
}

func TestFindTesseractPath(t *testing.T) {
	// 1. 不加 root（通过 PATH/安装路径查找）
	sysPath := findTesseract("")
	t.Logf("Tesseract（无 root）: %q", sysPath)

	// 2. 用项目根目录查找（应找到 bin/tesseract/ 或 tesseract/）
	//    获取项目根目录（通过 go.mod 位置）
	projectRoot := findProjectRoot()
	t.Logf("项目根目录: %s", projectRoot)
	projPath := findTesseract(projectRoot)
	t.Logf("Tesseract（项目目录）: %q", projPath)

	if projPath != "" {
		// 验证是 bin/tesseract/ 路径
		if !strings.Contains(projPath, "tesseract") {
			t.Errorf("项目 Tesseract 路径应包含 tesseract，实际: %s", projPath)
		}
	}
}

// findProjectRoot 向上查找包含 go.mod 的目录。
func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
