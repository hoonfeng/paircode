package agent

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestPluginTakeoverHostTool 插件接管宿主工具：claimTool 自动存档宿主执行器，
// 插件同名注册后 Registry 工具由插件接管，hostTool 仍可执行宿主 Go 实现。
func TestPluginTakeoverHostTool(t *testing.T) {
	reg := NewRegistry()
	root := t.TempDir()
	host := NewPluginHost(reg, nil, root)
	RegisterBuiltinPlugins(host)

	// 内置 read_file 已注册
	if _, ok := reg.Get("read_file"); !ok {
		t.Fatal("内置 read_file 未注册")
	}

	// JS 插件接管 read_file（execute 调 ctx.hostTool.exec）
	code := `
return {
  name: 'takeover-test',
  apply(ctx) {
    ctx.tools.register({
      name: 'read_file',
      description: '接管后的 read_file',
      parameters: { type: 'object', properties: { path: { type: 'string' } }, required: ['path'] },
      execute: (args) => {
        // 演示插件编排：调用宿主执行器并附加标记
        const out = ctx.hostTool.exec('read_file', args)
        return '[插件接管]' + out
      },
    })
  },
}
`
	id, err := host.DefineJSCodeFull(code, "js", "接管测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatalf("定义 %s 不存在", id)
	}
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}

	// ① 宿主执行器已存档（被接管的 read_file）
	if _, ok := HostToolMeta("read_file"); !ok {
		t.Fatal("read_file 宿主执行器未存档")
	}
	// 未被接管的 write_file 不应在 hostExecutors（仍留在 Registry 直接可用）
	if _, ok := HostToolMeta("write_file"); ok {
		t.Fatal("write_file 未被接管不应存档")
	}
	if _, ok := reg.Get("write_file"); !ok {
		t.Fatal("write_file 应保留在 Registry")
	}

	// ② Registry 中 read_file 已被插件接管（Handler 为 JS 桥）
	tool, ok := reg.Get("read_file")
	if !ok {
		t.Fatal("read_file 不存在")
	}
	if !strings.Contains(tool.Description, "接管后的 read_file") {
		t.Fatalf("read_file 未被插件接管，描述=%q", tool.Description)
	}

	// ③ hostTool 可执行宿主实现
	f := root + "/x.txt"
	if err := osWriteFile(f, "hello world"); err != nil {
		t.Fatalf("写测试文件失败: %v", err)
	}
	out, err := ExecuteHostTool("read_file", map[string]any{"path": f})
	if err != nil {
		t.Fatalf("ExecuteHostTool 失败: %v", err)
	}
	if !strings.Contains(out, "hello world") {
		t.Fatalf("hostTool 输出异常: %q", out)
	}

	// ④ 插件工具 execute 走 JS 桥（含插件编排标记）
	argsJSON, _ := jsonMarshal(map[string]any{"path": f})
	res, err := reg.Execute(nil, "read_file", argsJSON)
	if err != nil {
		t.Fatalf("插件 read_file 执行失败: %v", err)
	}
	if !strings.Contains(res, "[插件接管]") {
		t.Fatalf("插件 read_file 未走 JS 编排，结果=%q", res)
	}
}

// TestHostToolUnknown 未存档工具执行报错。
func TestHostToolUnknown(t *testing.T) {
	if _, err := ExecuteHostTool("not_exist_tool_xyz", nil); err == nil {
		t.Fatal("未存档工具应报错")
	}
}

// osWriteFile 测试辅助：写文件（os.WriteFile 直接调用，避开工作区校验）。
func osWriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// jsonMarshal 测试辅助：JSON 序列化。
func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}
