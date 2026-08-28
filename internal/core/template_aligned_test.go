package core

// ═══════════════════════════════════════════════════════════════
// template_aligned_test.go — Round3 ⑥.3（G2/G4 裁剪版）：配置模板对齐测试
//
// 断言 3 个 template（settings/models/mcp）：
//   - 顶层键集 ⊆ 对应加载结构体键集（漂移防护：模板里出现加载器不认识的键 = 漂移）
//   - settings 模板默认值 == Default()
//   - AI 业务字段（provider/baseURL/apiKey/模型，omitempty）不出现在模板属预期
// ═══════════════════════════════════════════════════════════════

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// templatePath 仓库 config/ 模板路径（测试 cwd = internal/core）。
func templatePath(name string) string {
	return filepath.Join("..", "..", "config", name)
}

func readTemplateMap(t *testing.T, name string) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(templatePath(name))
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", name, err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("解析 %s 失败: %v", name, err)
	}
	return m
}

// jsonValueEqual 比较两个 JSON 原始值：字节相等，或两侧均为「空值等价」
// （null / [] / {} / "" 之间互相等价——零值序列化差异不算漂移）。
func jsonValueEqual(a, b json.RawMessage) bool {
	if string(a) == string(b) {
		return true
	}
	isEmpty := func(raw json.RawMessage) bool {
		s := string(raw)
		return s == "null" || s == "[]" || s == "{}" || s == `""`
	}
	return isEmpty(a) && isEmpty(b)
}

// TestTemplateSchemaAligned settings/models/mcp 模板键集 ⊆ 加载结构体键集，默认值一致。
func TestTemplateSchemaAligned(t *testing.T) {
	// ── settings.template.json：键集 ⊆ AppSettings；默认值 == Default() ──
	known := settingsKnownKeys()
	tmpl := readTemplateMap(t, "settings.template.json")
	for k := range tmpl {
		if !known[k] {
			t.Errorf("settings.template.json 键 %q 不在 AppSettings 中（模板漂移）", k)
		}
	}
	// 默认值对齐：模板键值 == Default() 对应字段（JSON 化后比较；
	// null/[]/{} 视作等价空值——Default() 零值 map/slice 序列化为 null，
	// 模板用空容器更可读，语义一致）
	def := Default()
	defJSON, _ := json.Marshal(def)
	var defMap map[string]json.RawMessage
	_ = json.Unmarshal(defJSON, &defMap)
	// ★ 模板增强默认（有意的模板侧默认，Default() 不设）——以测试固定差异：
	//   ignoreDirs 模板预置 ["node_modules"]（跳过依赖目录的合理出厂默认）
	templateEnhancements := map[string]bool{"ignoreDirs": true}
	// Default() 只设非零字段；模板中出现的键必须与 Default() 值一致
	for k, rawTmpl := range tmpl {
		rawDef, ok := defMap[k]
		if !ok {
			if templateEnhancements[k] {
				continue
			}
			// Default() 零值字段：模板默认必须为零值等价（空串/0/false/null）
			var v any
			if err := json.Unmarshal(rawTmpl, &v); err != nil {
				continue
			}
			switch tv := v.(type) {
			case string:
				if tv != "" {
					t.Errorf("模板键 %q 默认 %q 与 Default() 零值不一致（模板漂移）", k, tv)
				}
			case float64:
				if tv != 0 {
					t.Errorf("模板键 %q 默认 %v 与 Default() 零值不一致", k, tv)
				}
			case bool:
				if tv {
					t.Errorf("模板键 %q 默认 true 与 Default() 零值不一致", k)
				}
			case nil:
			default:
				t.Errorf("模板键 %q 类型 %T 非零值（Default() 未设默认）", k, tv)
			}
			continue
		}
		if templateEnhancements[k] {
			continue // 模板增强默认（见上注），跳过比较
		}
		if !jsonValueEqual(rawTmpl, rawDef) {
			t.Errorf("模板键 %q 默认值 %s != Default() %s（模板漂移）", k, rawTmpl, rawDef)
		}
	}

	// ── models.template.json：条目键 ⊆ ProviderEntry；顶层服务商名为自由键 ──
	modelsData, err := os.ReadFile(templatePath("models.template.json"))
	if err != nil {
		t.Fatalf("读取 models.template.json 失败: %v", err)
	}
	var modelsRaw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(modelsData, &modelsRaw); err != nil {
		t.Fatalf("解析 models.template.json 失败: %v", err)
	}
	provKnown := structJSONKeys(ProviderEntry{})
	for prov, fields := range modelsRaw {
		for k := range fields {
			if !provKnown[k] {
				t.Errorf("models.template.json 服务商 %q 条目键 %q 不在 ProviderEntry 中（模板漂移）", prov, k)
			}
		}
	}

	// ── mcp.template.json：顶层键 ⊆ {servers}（mcpFile 结构） ──
	mcpData, err := os.ReadFile(templatePath("mcp.template.json"))
	if err != nil {
		t.Fatalf("读取 mcp.template.json 失败: %v", err)
	}
	var mcpRaw map[string]json.RawMessage
	if err := json.Unmarshal(mcpData, &mcpRaw); err != nil {
		t.Fatalf("解析 mcp.template.json 失败: %v", err)
	}
	for k := range mcpRaw {
		if k != "servers" {
			t.Errorf("mcp.template.json 键 %q 不在 mcpFile 结构（加载器 DisallowUnknownFields 会拒绝）", k)
		}
	}
}
