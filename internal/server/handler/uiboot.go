// HandleUIBoot GET /api/ui-boot：外部兼容 boot 图（UI 插件发现/装载的单一入口）。
//
// 契约（docs/ui-plugin-refactor-spec.md §3.2/§3.5-M1）：返回 DSH WebBootGraph 等价结构
//   { rev, entries: [{ id, url, rev, inject, immediately, external }] }
// 由后端从「已装配的磁盘 UI 插件包清单 + 各包 dsh.ui 段 + 各 bundle 内容 hash」组装，
// 前端薄壳只消费这一张图装载 region 包（不再逐包拼 listPlugins+detail）。
package handler

import (
	"net/http"

	"github.com/hoonfeng/paircode/internal/agent"
)

// HandleUIBoot GET /api/ui-boot：输出 外部兼容 boot 图。
func HandleUIBoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	jsonResp(w, agent.BuildUIBootGraph())
}
