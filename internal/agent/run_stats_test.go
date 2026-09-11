package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRunStatsPersistAndLoad 覆盖运行统计的核心链路（★ 2026-09-12 后端统计改造）：
//   - Begin/addStep/addLLMCall/addToolCall/End 累加与定格
//   - token 速度派生（分母 = Σ LLM 生成阶段耗时，不含工具耗时）
//   - 落盘 .pair/run-stats.json 后，**清空内存**（模拟进程重启）仍能读回
func TestRunStatsPersistAndLoad(t *testing.T) {
	dir := t.TempDir()
	conv := "conv_runstats_unit"

	s := BeginRunStatsFor(dir, conv)
	if s == nil {
		t.Fatal("BeginRunStatsFor 返回 nil")
	}
	// 3 步：2 次工具调用（各 100ms/300ms）+ 3 次 LLM 调用（总 2000ms，生成 1500ms）
	s.addStep()
	s.addStep()
	s.addStep()
	s.addToolCall(100 * time.Millisecond)
	s.addToolCall(300 * time.Millisecond)
	s.addLLMCall(&Usage{PromptTokens: 1000, CompletionTokens: 400}, 700, 500)
	s.addLLMCall(&Usage{PromptTokens: 1200, CompletionTokens: 600}, 800, 600)
	s.addLLMCall(nil, 500, 400) // provider 未回传 usage 的调用（只计时）

	// 运行中：内存可见（running=true）、速度按生成耗时派生
	live := RunStatsFor(dir, conv)
	if !live.Running {
		t.Error("运行中 running 应为 true")
	}
	if live.Steps != 3 || live.ToolCalls != 2 || live.LLMCalls != 3 {
		t.Errorf("累加异常 steps=%d tools=%d llm=%d", live.Steps, live.ToolCalls, live.LLMCalls)
	}
	if live.ToolMs != 400 {
		t.Errorf("工具耗时累计 = %d，期望 400", live.ToolMs)
	}
	if live.GenMs != 1500 || live.LLMMs != 2000 {
		t.Errorf("LLM 计时异常 genMs=%d llmMs=%d", live.GenMs, live.LLMMs)
	}
	// 速度 = 1000 token ÷ 1.5s = 666.67 t/s（★ 分母是生成耗时，不是墙钟）
	if live.TokensPerSecond < 666 || live.TokensPerSecond > 668 {
		t.Errorf("token 速度 = %.2f，期望 ≈666.67（1000 输出 token / 1.5s 生成耗时）", live.TokensPerSecond)
	}
	if live.CompletionTokens != 1000 || live.PromptTokens != 2200 {
		t.Errorf("token 用量异常 prompt=%d completion=%d", live.PromptTokens, live.CompletionTokens)
	}

	ended := EndRunStatsFor(dir, conv)
	if ended == nil || ended.EndAt == 0 || ended.Running {
		t.Fatal("EndRunStatsFor 未定格")
	}
	if ended.DurationMs < 0 {
		t.Errorf("定格后 durationMs 不应为负：%d", ended.DurationMs)
	}

	// 落盘存在
	path := filepath.Join(dir, ".pair", "run-stats.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("未落盘 %s: %v", path, err)
	}

	// 模拟进程重启：清空内存缓存后应能从磁盘读回同一结果
	dropRunStatsMemoryForTest()
	disk := RunStatsFor(dir, conv)
	if disk.Steps != 3 || disk.ToolCalls != 2 || disk.LLMCalls != 3 {
		t.Errorf("读盘后统计不符 steps=%d tools=%d llm=%d", disk.Steps, disk.ToolCalls, disk.LLMCalls)
	}
	if disk.TokensPerSecond < 666 || disk.TokensPerSecond > 668 {
		t.Errorf("读盘后 token 速度 = %.2f，期望 ≈666.67", disk.TokensPerSecond)
	}
	if !disk.Running == false || disk.EndAt == 0 {
		t.Error("读盘后应为已定格（EndAt>0, running=false）")
	}

	// ★ 路径规范化：等价写法（尾分隔符/\.）必须命中同一条目
	equiv := filepath.Join(dir, ".") + string(os.PathSeparator)
	if got := RunStatsFor(equiv, conv); got.Steps != 3 {
		t.Errorf("等价路径 %q 未命中同一统计（steps=%d）——root 需规范化", equiv, got.Steps)
	}
}

// dropRunStatsMemoryForTest 清空运行统计内存缓存（模拟进程重启；测试用）。
func dropRunStatsMemoryForTest() {
	runStatsMu.Lock()
	runStatsByRoot = make(map[string]map[string]*RunStats)
	runStatsMu.Unlock()
}
