// main_test.go — agent 包测试入口：全局工具集目录测试隔离。
//
// ★ 2026-09-04 工具集全局化后，globalToolsetDir() 默认指向
// <InstallDir>/.pair/toolsets/；而测试进程 InstallDir() 回退到 getwd
// （仓库根），其下恰好存在 .pair/toolsets/default.json —— 若测试不隔离，
// hasWorkspaceToolsets() 返回 true，插件工具可见性收敛会把测试注册的
// 工具误禁用（「未加入工作区工具集」），大量 JS 插件测试失败。
//
// 因此 TestMain 把全局工具集目录重定向到一个空的临时目录（全包生效）；
// 工具集相关测试仍可自行 SetGlobalToolsetDirForTest(t.TempDir()) 按需
// 隔离（cleanup 恢复 testGlobalToolsetDir，保持包级隔离不泄漏真实目录）。
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// testGlobalToolsetDir 测试进程内的全局工具集隔离目录（TestMain 创建）。
var testGlobalToolsetDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "pair-agent-toolset-test-*")
	if err != nil {
		// 创不了临时目录（极端环境）：兜底用当前目录下的不存在路径，
		// 至少避免读到仓库根真实工具集。
		dir = os.TempDir()
	}
	testGlobalToolsetDir = dir
	SetGlobalToolsetDirForTest(dir)
	// ★ 预写 .preset-seeded 标记（内容=当前预设版本号）：seedPresetToolsets
	//   读到版本号 >= 当前版本后直接返回，隔离目录永不被播种（保持空）——
	//   否则 TestAgentBaseInitPlugins 等触发 AgentBase.Init → seedPresetToolsets
	//   会把 planning/default.json 等预置模式写入隔离目录，hasWorkspaceToolsets()
	//   变 true 后后续测试的插件工具被可见性收敛误禁用（「未加入工作区工具集」
	//   连锁失败）。工具集专项测试自行 SetGlobalToolsetDirForTest(t.TempDir())
	//   不受影响（它们的 temp dir 无标记，播种/自建工具集按其用例预期发生）。
	_ = os.WriteFile(filepath.Join(dir, presetSeedMarker), []byte(fmt.Sprintf("%d", presetSeedVersion)), 0o644)
	code := m.Run()
	SetGlobalToolsetDirForTest("")
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
