// typed_tool.go 强类型工具定义辅助（泛型）。
// 参考 ADK functiontool：用 struct + json tag 反射自动生成 JSON Schema + 入口校验 + panic recovery。
// 作为新工具的推荐写法，不强制改造现有工具（现有 objSchema 手写 schema 仍保留）。
//
// 示例：
//
//	type readArgs struct {
//	    Path   string `json:"path"`
//	    Offset int    `json:"offset,omitempty"`
//	}
//	reg.Register(DefineTool("read_file", "读文件", func(ctx context.Context, a readArgs) (string, error) {
//	    return os.ReadFile(a.Path)
//	}))

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// DefineTool 用泛型 struct 定义工具。
//   - TArgs 的 json tag 自动生成 JSON Schema（name→type→properties）。
//   - 无 omitempty tag 的字段作 required；入口校验必填字段非零值。
//   - handler 收到已 json.Unmarshal 的 TArgs；handler panic 被 recover 转 error，不崩 agent。
//
// 支持的字段类型：string/int*/float*/bool/array/slice/map/struct(嵌套)/指针。
// 不支持的类型跳过（不进 schema）。
func DefineTool[TArgs any](name, description string, handler func(ctx context.Context, args TArgs) (string, error)) *Tool {
	props := schemaOf[TArgs]()
	required := requiredOf[TArgs]()
	return &Tool{
		Name:        name,
		Description: description,
		Parameters:  objSchema(props, required...),
		Handler: func(ctx context.Context, raw map[string]any) (result string, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("工具 %s panic: %v", name, r)
					result = ""
				}
			}()
			// raw map → JSON → TArgs（复用 json tag 语义，忽略未知字段）
			buf, mErr := json.Marshal(raw)
			if mErr != nil {
				return "", fmt.Errorf("参数序列化失败: %w", mErr)
			}
			var a TArgs
			if uErr := json.Unmarshal(buf, &a); uErr != nil {
				return "", fmt.Errorf("参数解析失败: %w", uErr)
			}
			if vErr := validateRequired(a, required); vErr != nil {
				return "", vErr
			}
			return handler(ctx, a)
		},
	}
}

// schemaOf 反射 TArgs 生成 JSON Schema 的 properties（field name → type schema）。
func schemaOf[T any]() map[string]any {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return structProps(t)
}

// structProps 返回 struct 类型的 properties map（非 struct 返回空）。
func structProps(t reflect.Type) map[string]any {
	props := map[string]any{}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return props
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := jsonNameOf(f)
		if name == "-" || name == "" {
			continue
		}
		if prop := typeSchemaOf(f.Type); prop != nil {
			props[name] = prop
		}
	}
	return props
}

// jsonNameOf 读 json tag 取字段名。无 tag 用字段名；"json:\"-\"" 跳过。
func jsonNameOf(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name
	}
	parts := strings.SplitN(tag, ",", 2)
	if parts[0] == "-" {
		return "-"
	}
	if parts[0] != "" {
		return parts[0]
	}
	return f.Name
}

// typeSchemaOf 反射单个字段类型 → JSON Schema 片段。不支持的类型返回 nil（跳过）。
func typeSchemaOf(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		items := typeSchemaOf(t.Elem())
		if items == nil {
			items = map[string]any{}
		}
		return map[string]any{"type": "array", "items": items}
	case reflect.Map:
		return map[string]any{"type": "object"}
	case reflect.Struct:
		return map[string]any{"type": "object", "properties": structProps(t)}
	}
	return nil
}

// requiredOf 反射 TArgs，返回无 omitempty 的字段名（必填）。
func requiredOf[T any]() []string {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var req []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := jsonNameOf(f)
		if name == "-" || name == "" {
			continue
		}
		// 有 omitempty = 可选；无 = 必填
		if !strings.Contains(tag, "omitempty") {
			req = append(req, name)
		}
	}
	return req
}

// validateRequired 校验必填字段非零值。LLM 可能漏传必填参数，此处兜底拦截。
func validateRequired(v any, required []string) error {
	if len(required) == 0 {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil // 指针 nil 不校验（TArgs 通常非指针）
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	t := rv.Type()
	for _, name := range required {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if jsonNameOf(f) != name {
				continue
			}
			if rv.Field(i).IsZero() {
				return fmt.Errorf("缺少必填参数: %s", name)
			}
		}
	}
	return nil
}
