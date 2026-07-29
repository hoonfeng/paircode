package agent

// Lua 自定义工具（动态增减 + 优化）—— 嵌入 gopher-lua，让用户/Agent 用 .lua 脚本定义工具：
// 放进工具目录即热加载、改脚本即优化、删文件即移除。companion 在每次发送时热重载（见 agent_bridge.go）。
//
// 安全沙箱：
//   - 已开启：base/string/table/math/coroutine + 安全 os 子集（time/date/clock/difftime）
//   - 禁用：io、debug、package、os.execute/os.remove/os.rename/os.exit
//   - agent 桥接函数：run_command / read_file / write_file / list_files /
//     json_encode / json_decode / timestamp / log / env
//   - 每次调用新建状态（隔离）；单次执行 10s 超时（防死循环）。
//   - 自定义工具默认 RequiresApproval（写类，需审批）。
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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

import (
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

// newSandboxLua 建受限 Lua 状态：base/string/table/math/coroutine + 安全 os 子集。
// 禁用：io、debug、package、以及 os 库中的危险函数。
func newSandboxLua() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	// 安全库：base / table / string / math / coroutine
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
		{lua.CoroutineLibName, lua.OpenCoroutine},
	} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	// 打开 os 库后抹掉危险函数
	L.Push(L.NewFunction(lua.OpenOs))
	L.Push(lua.LString(lua.OsLibName))
	L.Call(1, 0)
	for _, danger := range []string{"execute", "remove", "rename", "exit", "tmpname", "setlocale"} {
		L.SetGlobal("os", L.GetGlobal("os").(*lua.LTable))
		osTbl := L.GetGlobal("os").(*lua.LTable)
		osTbl.RawSetString(danger, lua.LNil)
	}
	// 抹掉 base 库里仍可加载外部代码的危险函数
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
		UsageGuide:       fmt.Sprintf("自定义 Lua 工具「%s」，在 .pair/tools/ 下定义。自动从脚本元信息生成参数 Schema，用 `lua_tool_list` 查看所有可用工具。", name),
		Category:         "lua-tool",
		Enabled:          true,
		Parameters:       params,
		RequiresApproval: true, // 自定义脚本工具默认需审批（安全）
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return runLuaTool(ctx, src, args)
		},
	}, nil
}

// runLuaTool 新建沙箱状态执行脚本的 run(args)，返回字符串结果（隔离、10s 超时）。
// 每次调用注入 agent 桥接函数，通过 cctx 传递超时，确保子进程随 Lua 超时一同终止。
func runLuaTool(ctx context.Context, src string, args map[string]any) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	L := newSandboxLua()
	L.SetContext(cctx)
	defer L.Close()

	// ── 注入 agent 桥接表 ──
	L.SetGlobal("agent", L.NewTable())
	agentTbl := L.GetGlobal("agent").(*lua.LTable)

	// agent.run_command({ command="...", cwd="..." }) — 执行 shell 命令
	agentTbl.RawSetString("run_command", L.NewFunction(func(L *lua.LState) int {
		argTbl := L.CheckTable(1)
		cmdStr := ""
		cwdStr := ""
		argTbl.ForEach(func(k, v lua.LValue) {
			if ks, ok := k.(lua.LString); ok {
				switch string(ks) {
				case "command":
					if vs, ok := v.(lua.LString); ok {
						cmdStr = string(vs)
					}
				case "cwd":
					if vs, ok := v.(lua.LString); ok {
						cwdStr = string(vs)
					}
				}
			}
		})
		if cmdStr == "" {
			L.Push(lua.LString("错误: command 参数不能为空"))
			return 1
		}
		c := exec.CommandContext(cctx, "cmd", "/C", "chcp 65001 >nul & "+cmdStr)
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if cwdStr != "" {
			c.Dir = cwdStr
		}
		out, err := c.CombinedOutput()
		result := string(out)
		if err != nil {
			result += "\n[退出: " + err.Error() + "]"
		}
		L.Push(lua.LString(result))
		return 1
	}))

	// agent.read_file(path) — 读取工作区内文件内容（UTF-8 文本，最长 512KB）
	agentTbl.RawSetString("read_file", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		root := workspaceRoot()
		full, err := resolvePath(root, path)
		if err != nil {
			L.Push(lua.LString("错误: " + err.Error()))
			return 1
		}
		data, err := os.ReadFile(full)
		if err != nil {
			L.Push(lua.LString("错误: 读取失败 - " + err.Error()))
			return 1
		}
		if len(data) > 512*1024 {
			L.Push(lua.LString("错误: 文件过大（>512KB），请在 run_command 中用 findstr/type 选择性读取"))
			return 1
		}
		L.Push(lua.LString(string(data)))
		return 1
	}))

	// agent.write_file(path, content) — 写入工作区内文件（覆盖）
	agentTbl.RawSetString("write_file", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		content := L.CheckString(2)
		root := workspaceRoot()
		full, err := resolvePath(root, path)
		if err != nil {
			L.Push(lua.LString("错误: " + err.Error()))
			return 1
		}
		parent := filepath.Dir(full)
		if err := os.MkdirAll(parent, 0o755); err != nil {
			L.Push(lua.LString("错误: 创建目录失败 - " + err.Error()))
			return 1
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			L.Push(lua.LString("错误: 写入失败 - " + err.Error()))
			return 1
		}
		L.Push(lua.LString("已写入 " + path + "（" + fmt.Sprintf("%d", len(content)) + " 字节）"))
		return 1
	}))

	// agent.list_files(dir, pattern?) — 列出目录内容
	agentTbl.RawSetString("list_files", L.NewFunction(func(L *lua.LState) int {
		dir := L.CheckString(1)
		pattern := L.OptString(2, "")
		root := workspaceRoot()
		full, err := resolvePath(root, dir)
		if err != nil {
			L.Push(lua.LString("错误: " + err.Error()))
			return 1
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			L.Push(lua.LString("错误: 读取目录失败 - " + err.Error()))
			return 1
		}
		var result []string
		for _, e := range entries {
			name := e.Name()
			if pattern != "" {
				if ok, _ := filepath.Match(pattern, name); !ok {
					continue
				}
			}
			if e.IsDir() {
				result = append(result, name+"/")
			} else {
				result = append(result, name)
			}
		}
		if len(result) == 0 {
			L.Push(lua.LString("（空目录或无匹配）"))
			return 1
		}
		L.Push(lua.LString(strings.Join(result, "\n")))
		return 1
	}))

	// agent.json_encode(value) — JSON 编码（Go 标准库）
	agentTbl.RawSetString("json_encode", L.NewFunction(func(L *lua.LState) int {
		val := L.Get(1)
		goVal := luaToGo(val)
		data, err := json.Marshal(goVal)
		if err != nil {
			L.Push(lua.LString("错误: JSON 编码失败 - " + err.Error()))
			return 1
		}
		L.Push(lua.LString(string(data)))
		return 1
	}))

	// agent.json_decode(str) — JSON 解码为 Lua 表
	agentTbl.RawSetString("json_decode", L.NewFunction(func(L *lua.LState) int {
		str := L.CheckString(1)
		var v any
		if err := json.Unmarshal([]byte(str), &v); err != nil {
			L.Push(lua.LString("错误: JSON 解码失败 - " + err.Error()))
			return 1
		}
		L.Push(goToLua(L, v))
		return 1
	}))

	// agent.timestamp() — 当前时间字符串（2006-01-02 15:04:05）
	agentTbl.RawSetString("timestamp", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(time.Now().Format("2006-01-02 15:04:05")))
		return 1
	}))

	// agent.log(level, message) — 结构化日志输出
	agentTbl.RawSetString("log", L.NewFunction(func(L *lua.LState) int {
		level := L.CheckString(1)
		msg := L.CheckString(2)
		line := fmt.Sprintf("[LuaTool %s] %s", level, msg)
		L.Push(lua.LString(line))
		return 1
	}))

	// agent.env(key) — 读取环境变量
	agentTbl.RawSetString("env", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		val := os.Getenv(key)
		L.Push(lua.LString(val))
		return 1
	}))

	// ── 执行脚本 ──
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

// workspaceRoot 取第一个工作区根目录作为 Lua 文件操作的基路径。
func workspaceRoot() string {
	if len(WorkspaceRoots) > 0 {
		return WorkspaceRoots[0]
	}
	return "."
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

// goToLua 把 Go 值转 Lua（map→table / slice→array / nil→LNil）。
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
	case nil:
		return lua.LNil
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
