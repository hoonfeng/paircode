// typed_tool_test.go 测试强类型工具辅助 DefineTool：schema 反射 + 必填校验 + panic recovery。

package agent

import (
	"context"
	"strings"
	"testing"
)

// readArgs 测试用参数 struct。
type readArgs struct {
	Path   string `json:"path"`             // 必填
	Offset int    `json:"offset,omitempty"` // 可选
	Limit  int    `json:"limit,omitempty"`  // 可选
}

func TestDefineTool_Schema(t *testing.T) {
	tool := DefineTool("read_file", "读文件", func(_ context.Context, _ readArgs) (string, error) {
		return "", nil
	})
	if tool.Name != "read_file" {
		t.Errorf("Name 应为 read_file，得 %q", tool.Name)
	}
	props, _ := tool.Parameters["properties"].(map[string]any)
	if props == nil {
		t.Fatal("properties 应为 map")
	}
	// path 应是 string
	p, ok := props["path"].(map[string]any)
	if !ok || p["type"] != "string" {
		t.Errorf("path 应是 string 类型，得 %v", props["path"])
	}
	// offset 应是 integer
	o, ok := props["offset"].(map[string]any)
	if !ok || o["type"] != "integer" {
		t.Errorf("offset 应是 integer 类型，得 %v", props["offset"])
	}
	// required 应含 path，不含 offset/limit
	req, _ := tool.Parameters["required"].([]string)
	if !containsStr(req, "path") {
		t.Errorf("required 应含 path，得 %v", req)
	}
	if containsStr(req, "offset") || containsStr(req, "limit") {
		t.Errorf("required 不应含可选字段，得 %v", req)
	}
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func TestDefineTool_RequiredValidation(t *testing.T) {
	tool := DefineTool("read_file", "读文件", func(_ context.Context, a readArgs) (string, error) {
		return "ok:" + a.Path, nil
	})
	// 缺必填 path → 报错
	if _, err := tool.Handler(context.Background(), map[string]any{"offset": 1}); err == nil {
		t.Error("缺必填 path 应报错")
	}
	// 有必填 path → 正常
	got, err := tool.Handler(context.Background(), map[string]any{"path": "x.txt"})
	if err != nil {
		t.Fatalf("正常调用应成功: %v", err)
	}
	if got != "ok:x.txt" {
		t.Errorf("返回值不符，得 %q", got)
	}
	// 可选字段正确解析
	got2, _ := tool.Handler(context.Background(), map[string]any{"path": "y.txt", "offset": 10, "limit": 5})
	if got2 != "ok:y.txt" {
		t.Errorf("带可选字段返回不符，得 %q", got2)
	}
}

// TestDefineTool_PanicRecovery handler panic 不应崩 agent，转 error。
func TestDefineTool_PanicRecovery(t *testing.T) {
	tool := DefineTool("boom", "会炸", func(_ context.Context, _ readArgs) (string, error) {
		panic("炸了")
	})
	_, err := tool.Handler(context.Background(), map[string]any{"path": "x"})
	if err == nil || !strings.Contains(err.Error(), "panic") {
		t.Errorf("panic 应转 error 含 panic 字样，得 %v", err)
	}
}

// TestDefineTool_NestedStruct 嵌套 struct schema 生成。
type nestedArgs struct {
	Name string `json:"name"`
	Meta struct {
		Tags []string `json:"tags,omitempty"`
	} `json:"meta,omitempty"`
}

func TestDefineTool_NestedStruct(t *testing.T) {
	tool := DefineTool("nested", "嵌套", func(_ context.Context, _ nestedArgs) (string, error) {
		return "", nil
	})
	props, _ := tool.Parameters["properties"].(map[string]any)
	meta, ok := props["meta"].(map[string]any)
	if !ok || meta["type"] != "object" {
		t.Errorf("meta 应是 object，得 %v", props["meta"])
	}
	metaProps, _ := meta["properties"].(map[string]any)
	tags, ok := metaProps["tags"].(map[string]any)
	if !ok || tags["type"] != "array" {
		t.Errorf("tags 应是 array，得 %v", metaProps["tags"])
	}
}

// TestDefineTool_BadJSON 坏参数类型 → 解析失败报错。
func TestDefineTool_BadJSON(t *testing.T) {
	tool := DefineTool("read_file", "读", func(_ context.Context, _ readArgs) (string, error) {
		return "", nil
	})
	// offset 传字符串（类型不符）→ 解析失败
	_, err := tool.Handler(context.Background(), map[string]any{"path": "x", "offset": "不是数字"})
	if err == nil {
		t.Error("类型不符应解析失败")
	}
}
