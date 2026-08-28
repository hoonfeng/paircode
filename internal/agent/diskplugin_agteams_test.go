package agent

// diskplugin_agteams_test.go — agent-teams 插件装载验证（宿主能力面适配）。

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentTeamsDiskLoad(t *testing.T) {
	code, err := os.ReadFile(filepath.Join(jsNativeWorkspace, ".pair", "plugins", "agent-teams", "index.js"))
	if err != nil {
		t.Skipf("agent-teams 未安装: %v", err)
	}
	host, reg := loadJSCodeForTest(t, string(code), filepath.Join(jsNativeWorkspace, ".pair", "plugins", "agent-teams"))
	names := reg.Names()
	t.Logf("工具总数: %d", len(names))
	for _, n := range names {
		t.Logf("  - %s", n)
	}
	if def, ok := host.GetJSDef("dyn-1"); ok {
		t.Logf("诊断: %v", def.diag)
		t.Logf("console: %v", def.console)
	}
	_ = host
}
