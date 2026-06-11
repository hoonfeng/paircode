package agent

// Lua 自定义工具（动态增减 + 优化）—— 嵌入 gopher-lua，让用户/Agent 用 .lua 脚本定义工具：
// 放进工具目录即热加载、改脚本即优化、删文件即移除。companion 在每次发送时热重载（见 agent_bridge.go）。
//
// 安全沙箱：只开 base/string/table/math，不开 os/io/package（防文件/系统越权）；每次调用新建状态（隔离）；
// 单次执行 10s 超时（防死循环）。自定义工具默认 RequiresApproval（写类，需审批）。
//
// 脚本格式——每个 .lua 文件 return 一个表：
//   return {
//     name = "word_count",
//     description = "统计文本字数",
//     parameters = { type="object", properties={ text={ type="string", description="文本" } }, required={"text"} },
//     run = function(args) return "字数: " .. #(args.text or "") end,
//   }

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// LoadLuaTools 扫描目录下所有 *.lua，注册为 agent 工具，返回成功注册的工具名。
// 目录不存在 → 返回 nil；单个脚本解析失败 → 跳过（不阻断其余）。
func LoadLuaTools(r *Registry, dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var loaded []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if tool, err := buildLuaTool(string(src), e.Name()); err == nil {
			r.Register(tool)
			loaded = append(loaded, tool.Name)
		}
	}
	return loaded
}

// newSandboxLua 建受限 Lua 状态：只开 base/string/table/math（无 os/io/package）。
func newSandboxLua() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	// 抹掉 base 库里仍可加载外部代码的危险函数。
	for _, danger := range []string{"dofile", "loadfile", "load", "loadstring", "require", "collectgarbage"} {
		L.SetGlobal(danger, lua.LNil)
	}
	return L
}

// buildLuaTool 执行脚本取返回表，构造 agent.Tool（元信息 + run 闭包）。
func buildLuaTool(src, fileName string) (*Tool, error) {
	L := newSandboxLua()
	defer L.Close()
	if err := L.DoString(src); err != nil {
		return nil, fmt.Errorf("%s 加载失败: %w", fileName, err)
	}
	tbl, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("%s 须 return 一个表", fileName)
	}
	name := luaField(tbl, "name")
	if name == "" {
		return nil, fmt.Errorf("%s 缺 name 字段", fileName)
	}
	if _, ok := tbl.RawGetString("run").(*lua.LFunction); !ok {
		return nil, fmt.Errorf("%s 缺 run 函数", fileName)
	}
	params, _ := luaToGo(tbl.RawGetString("parameters")).(map[string]any)
	if params == nil {
		params = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	desc := luaField(tbl, "description")
	return &Tool{
		Name:             name,
		Description:      strings.TrimSpace(desc + "（Lua 自定义工具）"),
		Parameters:       params,
		RequiresApproval: true, // 自定义脚本工具默认需审批（安全）
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return runLuaTool(ctx, src, args)
		},
	}, nil
}

// runLuaTool 新建沙箱状态执行脚本的 run(args)，返回字符串结果（隔离、10s 超时）。
func runLuaTool(ctx context.Context, src string, args map[string]any) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	L := newSandboxLua()
	L.SetContext(cctx)
	defer L.Close()
	if err := L.DoString(src); err != nil {
		return "", err
	}
	tbl, ok := L.Get(-1).(*lua.LTable)
	if !ok {
		return "", fmt.Errorf("脚本未返回表")
	}
	fn, ok := tbl.RawGetString("run").(*lua.LFunction)
	if !ok {
		return "", fmt.Errorf("脚本缺 run 函数")
	}
	L.Push(fn)
	L.Push(goToLua(L, args))
	if err := L.PCall(1, 1, nil); err != nil {
		return "", fmt.Errorf("Lua 工具执行出错: %w", err)
	}
	return luaResultStr(L.Get(-1)), nil
}

// ─── Lua ↔ Go 值转换 ─────────────────────────────────────────

func luaField(t *lua.LTable, key string) string {
	if s, ok := t.RawGetString(key).(lua.LString); ok {
		return string(s)
	}
	return ""
}

// luaToGo 把 Lua 值转 Go（表→[]any 数组 / map[string]any）。
func luaToGo(lv lua.LValue) any {
	switch v := lv.(type) {
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case lua.LBool:
		return bool(v)
	case *lua.LTable:
		isArray := v.Len() > 0
		v.ForEach(func(k, _ lua.LValue) {
			if _, ok := k.(lua.LNumber); !ok {
				isArray = false
			}
		})
		if isArray {
			arr := make([]any, 0, v.Len())
			for i := 1; i <= v.Len(); i++ {
				arr = append(arr, luaToGo(v.RawGetInt(i)))
			}
			return arr
		}
		m := map[string]any{}
		v.ForEach(func(k, val lua.LValue) {
			if ks, ok := k.(lua.LString); ok {
				m[string(ks)] = luaToGo(val)
			}
		})
		return m
	}
	return nil
}

// goToLua 把 Go 值转 Lua（map→table / slice→array）。
func goToLua(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case string:
		return lua.LString(val)
	case float64:
		return lua.LNumber(val)
	case int:
		return lua.LNumber(val)
	case bool:
		return lua.LBool(val)
	case []any:
		t := L.NewTable()
		for _, e := range val {
			t.Append(goToLua(L, e))
		}
		return t
	case map[string]any:
		t := L.NewTable()
		for k, e := range val {
			t.RawSetString(k, goToLua(L, e))
		}
		return t
	}
	return lua.LNil
}

// luaResultStr run 返回值转字符串（字符串原样，数字/其它走 gopher-lua 的 String()）。
func luaResultStr(lv lua.LValue) string {
	if s, ok := lv.(lua.LString); ok {
		return string(s)
	}
	if lv == lua.LNil {
		return "(无返回)"
	}
	return lv.String()
}
