package agent

import "testing"

// TestSearchMarketNPMPlugins_Filter 验证 npm 插件搜索的官方命名约定过滤：
//   - @paircode/* scope 包 → 收录
//   - paircode-plugin-*（带 paircode 关键词）→ 收录
//   - paircode-terminal（WhatsApp 配对码，无关裸名）→ 排除
//   - 描述含 paircode 的无关包 → 排除
func TestSearchMarketNPMPlugins_Filter(t *testing.T) {
	// 用真实 npm 搜索（query=paircode 时旧逻辑会命中 paircode-terminal）
	entries := searchMarketNPMPlugins("paircode")
	byName := map[string]MarketEntry{}
	for _, e := range entries {
		byName[e.ID] = e
	}

	// ① paircode-terminal 绝不能被收录（旧 bug：名字含 paircode 被误收）
	if _, ok := byName["paircode-terminal"]; ok {
		t.Errorf("paircode-terminal 被误收录（无关包，应排除）: %+v", byName["paircode-terminal"])
	}
	// ② 收录的必须全部符合官方命名约定
	for id := range byName {
		low := toLower(id)
		if !hasPrefix(low, "@paircode/") && !hasPrefix(low, "paircode-plugin-") {
			t.Errorf("收录了不符合官方命名约定的包: %s", id)
		}
	}
	t.Logf("真实 npm 搜索 'paircode' 收录 %d 个官方包: %v", len(entries), func() []string {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.ID)
		}
		return names
	}())
}

// toLower/hasPrefix 内联辅助（不依赖 strings import，保持测试独立）
func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
