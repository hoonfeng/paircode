// Model Scraper — 从各厂商 API/文档抓取最新模型列表，更新 config/models.json。
//
// 用法（在项目根目录执行）：
//   go run ./cmd/model-scraper/
//
// 输出：覆盖 config/models.json 为各厂商最新可用模型列表。
// 传入 API Key 可查询真实可用模型：
//   go run ./cmd/model-scraper/ --api-key=sk-xxx
//
// 也可只查询特定厂商：
//   go run ./cmd/model-scraper/ --provider=openai --api-key=sk-xxx

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ModelListMap 按服务商分组模型列表。
type ModelListMap map[string][]string

func main() {
	apiKey := flag.String("api-key", "", "OpenAI API Key（可选，传入则从真实 API 查询可用模型）")
	provider := flag.String("provider", "all", "要查询的厂商（all/deepseek/openai/anthropic，默认 all）")
	output := flag.String("output", "", "输出路径（默认 config/models.json）")
	flag.Parse()

	outPath := *output
	if outPath == "" {
		root := findProjectRoot()
		outPath = filepath.Join(root, "config", "models.json")
	}

	fmt.Println("🤖 模型列表爬取工具")
	fmt.Println("==================")
	fmt.Printf("输出路径: %s\n\n", outPath)

	result := make(ModelListMap)
	p := *provider

	if p == "all" || p == "deepseek" {
		models := fetchDeepSeekModels()
		if len(models) == 0 {
			models = defaultDeepSeek()
		}
		result["deepseek"] = models
	}

	if p == "all" || p == "openai" {
		var models []string
		if *apiKey != "" {
			models = fetchOpenAIModels(*apiKey)
		}
		if len(models) == 0 {
			models = fetchOpenAIDocs()
		}
		if len(models) == 0 {
			models = defaultOpenAI()
		}
		result["openai"] = models
	}

	if p == "all" || p == "anthropic" {
		models := fetchAnthropicModels()
		if len(models) == 0 {
			models = defaultAnthropic()
		}
		result["anthropic"] = models
	}

	// 始终保留兼容性厂商
	result["openai-compatible"] = []string{"custom"}
	result["custom"] = []string{"custom"}

	// 输出 JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON 编码失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "创建目录失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ 已写入 %s\n", outPath)
	fmt.Printf("   厂商: %s\n\n", strings.Join(sortedKeys(result), ", "))

	for _, k := range sortedKeys(result) {
		if len(result[k]) > 0 {
			fmt.Printf("  %s: %d 个模型\n", k, len(result[k]))
		}
	}
}

// findProjectRoot 查找项目根目录（向上找 go.mod）。
func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

func sortedKeys(m ModelListMap) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ─────────────────────────────────────────────
// DeepSeek
// ─────────────────────────────────────────────

func fetchDeepSeekModels() []string {
	fmt.Println("📡 [DeepSeek] 获取中...")

	// 从中文文档抓取
	models := tryDeepSeekVNDocs()
	if len(models) > 0 {
		fmt.Printf("📡 [DeepSeek] API 文档提取到 %d 个模型\n", len(models))
		return models
	}

	// 从 API 列表端点获取
	models = tryDeepSeekAPI()
	if len(models) > 0 {
		fmt.Printf("📡 [DeepSeek] API 端点返回 %d 个模型\n", len(models))
		return models
	}

	fmt.Println("📡 [DeepSeek] 获取失败")
	return nil
}

func tryDeepSeekVNDocs() []string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api-docs.deepseek.com/zh-cn/")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	text := string(body)
	// 匹配 deepseek-xxx 模型名
	re := regexp.MustCompile(`deepseek-[a-z][a-z0-9-]*[a-z0-9]`)
	matches := re.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	found := make(map[string]bool)
	for _, m := range matches {
		found[m] = true
	}

	// 确保核心模型在
	cores := []string{
		"deepseek-chat", "deepseek-reasoner",
		"deepseek-v3", "deepseek-r1",
		"deepseek-v4-pro", "deepseek-v4-flash",
		"deepseek-coder",
	}
	for _, c := range cores {
		found[c] = true
	}

	result := make([]string, 0, len(found))
	for m := range found {
		result = append(result, m)
	}
	sort.Strings(result)
	return result
}

func tryDeepSeekAPI() []string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.deepseek.com/v1/models")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	var models []string
	for _, m := range result.Data {
		if strings.Contains(m.ID, "embedding") || strings.HasPrefix(m.ID, "ft:") {
			continue
		}
		models = append(models, m.ID)
	}
	sort.Strings(models)
	return models
}

func defaultDeepSeek() []string {
	return []string{
		"deepseek-chat",
		"deepseek-reasoner",
		"deepseek-v3",
		"deepseek-r1",
		"deepseek-v4-pro",
		"deepseek-v4-flash",
	}
}

// ─────────────────────────────────────────────
// OpenAI
// ─────────────────────────────────────────────

func fetchOpenAIModels(apiKey string) []string {
	fmt.Println("📡 [OpenAI] 通过 API 获取...")
	if apiKey == "" {
		return nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("📡 [OpenAI] API 请求失败: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("📡 [OpenAI] 解析失败: %v\n", err)
		return nil
	}

	modelSet := make(map[string]bool)
	for _, m := range result.Data {
		id := m.ID
		// 排除非对话模型
		if strings.Contains(id, "embedding") ||
			strings.Contains(id, "whisper") ||
			strings.Contains(id, "tts") ||
			strings.Contains(id, "dall-e") ||
			strings.Contains(id, "moderation") ||
			strings.HasPrefix(id, "ft:") ||
			strings.HasPrefix(id, "gpt-3.5") ||
			strings.HasPrefix(id, "text-") ||
			strings.HasPrefix(id, "babbage") ||
			strings.HasPrefix(id, "davinci") ||
			strings.HasPrefix(id, "curie") ||
			strings.HasPrefix(id, "ada") ||
			strings.HasPrefix(id, "realtime") ||
			strings.Count(id, ":") > 0 {
			continue
		}
		modelSet[id] = true
	}

	// 确保核心模型在
	cores := []string{
		"gpt-4o", "gpt-4o-mini",
		"gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano",
		"o1", "o3-mini", "o4-mini",
		"gpt-4.5-preview",
	}
	for _, c := range cores {
		modelSet[c] = true
	}

	models := make([]string, 0, len(modelSet))
	for m := range modelSet {
		models = append(models, m)
	}
	sort.Strings(models)
	fmt.Printf("📡 [OpenAI] API 返回 %d 个可用模型\n", len(models))
	return models
}

func fetchOpenAIDocs() []string {
	fmt.Println("📡 [OpenAI] 从网页提取...")

	client := &http.Client{Timeout: 10 * time.Second}
	urls := []string{
		"https://openai.com/api/pricing/",
		"https://platform.openai.com/docs/models",
	}

	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		text := string(body)
		// 匹配 gpt-4o, gpt-4.1, o1, o3-mini, o4-mini 等
		re := regexp.MustCompile(`\b(gpt-4[.\w-]+|o[1-9][.\w-]*(?:-mini|-preview)?)\b`)
		matches := re.FindAllString(text, -1)

		found := make(map[string]bool)
		for _, m := range matches {
			m = strings.TrimRight(m, "-.")
			if len(m) < 4 || strings.HasPrefix(m, "gpt-3") {
				continue
			}
			found[m] = true
		}

		if len(found) >= 3 {
			models := make([]string, 0, len(found))
			for m := range found {
				models = append(models, m)
			}
			sort.Strings(models)
			fmt.Printf("📡 [OpenAI] 网页提取到 %d 个模型\n", len(models))
			return models
		}
	}

	fmt.Println("📡 [OpenAI] 网页提取失败")
	return nil
}

func defaultOpenAI() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4.1",
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		"o1",
		"o3-mini",
		"o4-mini",
		"gpt-4.5-preview",
	}
}

// ─────────────────────────────────────────────
// Anthropic
// ─────────────────────────────────────────────

func fetchAnthropicModels() []string {
	fmt.Println("📡 [Anthropic] 获取中...")

	client := &http.Client{Timeout: 10 * time.Second}

	// 从 doc 页面获取（跳过地理限制就是最全的）
	// 用 models.anthropic.com 作为替代入口
	urls := []string{
		"https://docs.anthropic.com/en/docs/about-claude/models",
	}

	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		text := string(body)

		// 匹配 claude 模型名：claude-3-5-sonnet-20241022, claude-4-sonnet-20250514 等
		re := regexp.MustCompile(`claude-(?:3[.-])?[45]?(?:[.-])?(?:sonnet|haiku|opus)(?:-latest)?(?:-\d{8})?`)
		matches := re.FindAllString(text, -1)

		found := make(map[string]bool)
		for _, m := range matches {
			found[m] = true
		}

		// 更宽松的匹配：claude-4-sonnet 等
		re2 := regexp.MustCompile(`claude-(?:3[.-]|4[.-])(?:sonnet|haiku|opus)(?:-latest|-\d{8})?`)
		matches2 := re2.FindAllString(text, -1)
		for _, m := range matches2 {
			found[m] = true
		}

		// 确保基础模型在
		cores := []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
			"claude-4-sonnet-20250514",
			"claude-4-haiku-latest",
		}
		for _, c := range cores {
			found[c] = true
		}

		if len(found) > 0 {
			models := make([]string, 0, len(found))
			for m := range found {
				models = append(models, m)
			}
			sort.Strings(models)
			fmt.Printf("📡 [Anthropic] 提取到 %d 个模型\n", len(models))
			return models
		}
	}

	fmt.Println("📡 [Anthropic] 获取失败")
	return nil
}

func defaultAnthropic() []string {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-4-sonnet-20250514",
		"claude-4-haiku-latest",
	}
}
