// uiboot.go — 外部兼容 UI 插件发现：装配 DSH WebBootGraph 等价 boot 图。
//
// 背景（对齐 docs/ui-plugin-refactor-spec.md §3.2/§4.1/§4.3）：
//   - 每个 UI 区域/功能是一个独立磁盘插件包（<InstallDir>/.pair/plugins/ui-*），
//     其 package.json 含 外部兼容二段式 manifest 的 dsh.ui 段（platform/slot/kind/
//     scope/inject/immediately/subSlots）；
//   - 服务端把「已装配的区域包清单 + 各包 dsh.ui 段 + 各 bundle 内容 hash」组装为
//     /api/ui-boot 的 boot 图（结构字段与 DSH WebBootGraph 一一对应）：
//     rev + entries[{id,url,rev,inject,immediately,external}]；
//   - 薄壳只消费这一张图装载 region 包（不再逐包拼 listPlugins+detail）。
//
// 不变量：
//   - 只纳入含 dsh.ui 段的包（manifest 独立发现，不依赖 build-ui.mjs 硬编码区域清单）；
//   - 无 dsh.ui 段的旧包（client.js 直载）仍走 /api/plugins + clientCode 旧路径（向后兼容）；
//   - 任一区域包 bundle 缺失/损坏 → 该 entry 跳过（发现层幂等，不致命）。
package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DshUIManifest package.json 的 dsh.ui 段（外部兼容 client 半清单）。
// 字段对齐 DSH dsh-client 的 dsh.client + cordis entry（platform/slot/kind/scope/
// inject/immediately），并扩展 subSlots（一个子槽一位认领者，声明即认领）。
type DshUIManifest struct {
	Platform    string   `json:"platform,omitempty"`    // 客户端平台（恒 "web"）
	Slot        string   `json:"slot,omitempty"`        // 本包注册到哪个薄壳槽位（target slot）
	Kind        string   `json:"kind,omitempty"`        // single | list（对齐 DSH SlotKind）
	Scope       string   `json:"scope,omitempty"`       // root | session | session-maybe（数据作用域）
	Inject      []string `json:"inject,omitempty"`      // client 半的 cordis/服务依赖
	Immediately *bool    `json:"immediately,omitempty"` // 是否 stage-one 预取（缺省 true）
	SubSlots    []string `json:"subSlots,omitempty"`    // 本包向下开放的子槽（声明即认领）
}

// GlobalPluginDsh GlobalPluginPackage 的 dsh 段（等价 dsh.client + cordis entry）。
// ui 子段为 外部兼容 client 半清单；无 dsh.ui 段 → 旧 client.js 直载路径。
type GlobalPluginDsh struct {
	UI *DshUIManifest `json:"ui,omitempty"`
}

// UIBootEntry DSH WebBootGraph 单条 entry（id/url/rev/inject/immediately/external
// 字段与 DSH manifest.ts 的 WebBootEntry 一一对应）。
type UIBootEntry struct {
	ID          string   `json:"id"`          // == package.json name（唯一装配 key）
	URL         string   `json:"url"`         // 客户端 bundle 端点（/plugins-assets/<name>/<assets-相对路径>?rev=）
	Rev         string   `json:"rev"`         // 该 bundle 内容 hash（cache-busting 一致性锚）
	Inject      []string `json:"inject"`      // cordis 服务依赖（信息性/装配等待依据）
	Immediately bool     `json:"immediately"` // stage-one 预取标记
	External    []string `json:"external"`    // 非基线模块说明符（= __PAIRCODE_CORE 词条）
}

// UIBootGraph 外部兼容 boot 图（GET /api/ui-boot 返回）。
type UIBootGraph struct {
	Rev     string        `json:"rev"`     // 全图一致性锚（各 bundle hash 摘要）
	Entries []UIBootEntry `json:"entries"` // 按 id 排序（发现层幂等、输出确定性）
}

// UIBootExternalCore 共享核心模块说明符：区域 bundle 一律 externals 到 __PAIRCODE_CORE。
const UIBootExternalCore = "@paircode/core"

// BuildUIBootGraph 扫描磁盘 UI 插件包，装配 外部兼容 boot 图。
// 仅纳入含 dsh.ui 段的包（独立 manifest 发现）；bundle 缺失/损坏的包跳过（幂等）。
func BuildUIBootGraph() UIBootGraph {
	return BuildUIBootGraphFrom(globalPluginsDir())
}

// BuildUIBootGraphFrom 从指定插件目录装配 boot 图（测试注入用；dir 不存在 → 空图）。
func BuildUIBootGraphFrom(dir string) UIBootGraph {
	dirents, err := os.ReadDir(dir)
	if err != nil {
		return UIBootGraph{Rev: "", Entries: []UIBootEntry{}}
	}
	entries := make([]UIBootEntry, 0, 16)
	for _, de := range dirents {
		if !de.IsDir() {
			continue
		}
		name := de.Name()
		if name == "" || strings.HasPrefix(name, ".") || name == "node_modules" || strings.HasSuffix(name, "-src") {
			continue
		}
		pkgDir := filepath.Join(dir, name)
		pkg, ok := readGlobalPluginPackage(pkgDir)
		if !ok || pkg.Name == "" || pkg.Dsh == nil || pkg.Dsh.UI == nil {
			continue // 非插件包 / 无 dsh.ui 段 → 旧 client.js 路径，不进 boot 图
		}
		assetRel, rev, err := resolveUIBundle(pkgDir, pkg.Name)
		if err != nil {
			// bundle 缺失 → 跳过（该区域槽位空态占位，其余正常；发现层幂等）
			continue
		}
		immediately := true
		if pkg.Dsh.UI.Immediately != nil {
			immediately = *pkg.Dsh.UI.Immediately
		}
		inject := append([]string(nil), pkg.Dsh.UI.Inject...)
		if inject == nil {
			inject = []string{}
		}
		entries = append(entries, UIBootEntry{
			ID:          pkg.Name,
			URL:         "/plugins-assets/" + pkg.Name + "/" + assetRel + "?rev=" + rev,
			Rev:         rev,
			Inject:      inject,
			Immediately: immediately,
			External:    []string{UIBootExternalCore},
		})
	}
	// 输出确定性：按 id 排序（发现层幂等，删除某包目录后无需重排缓存）。
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return UIBootGraph{Rev: graphRev(entries), Entries: entries}
}

// graphRev 全图一致性锚 = 各 entry「id:rev」摘要（对齐 DSH WebBootGraph.rev 语义）。
func graphRev(entries []UIBootEntry) string {
	h := sha256.New()
	for _, e := range entries {
		_, _ = h.Write([]byte(e.ID))
		_, _ = h.Write([]byte(":"))
		_, _ = h.Write([]byte(e.Rev))
		_, _ = h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// resolveUIBundle 解析插件包的主 bundle 文件（相对 assets/ 的路径）+ 内容 hash。
// 优先级：①<name>.js（ui-editor → ui-editor.js）；②assets 下最大 .js（git-api →
// git-panel.js 等异构命名）；③任意 .js。排除 .map 等辅助文件。
func resolveUIBundle(pkgDir, name string) (assetRel, rev string, err error) {
	assetsDir := filepath.Join(pkgDir, "assets")
	dirents, err := os.ReadDir(assetsDir)
	if err != nil {
		return "", "", err
	}
	type cand struct {
		name string
		size int64
	}
	var cands []cand
	for _, d := range dirents {
		if d.IsDir() {
			continue
		}
		fileName := d.Name()
		if !strings.HasSuffix(strings.ToLower(fileName), ".js") {
			continue
		}
		if strings.HasSuffix(fileName, ".map") {
			continue
		}
		size := int64(0)
		if info, e := d.Info(); e == nil {
			size = info.Size()
		}
		cands = append(cands, cand{name: fileName, size: size})
	}
	if len(cands) == 0 {
		return "", "", fmt.Errorf("插件 %q 无 bundle（assets 下无 .js）", name)
	}
	// 优先级 1：<name>.js；否则优先级 2：assets 下最大 .js（主 bundle 通常最大）
	var exact, biggest *cand
	for i := range cands {
		if cands[i].name == name+".js" && exact == nil {
			exact = &cands[i]
		}
		if biggest == nil || cands[i].size > biggest.size {
			biggest = &cands[i]
		}
	}
	pick := biggest
	if exact != nil {
		pick = exact
	}
	data, err := os.ReadFile(filepath.Join(assetsDir, pick.name))
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	return filepath.ToSlash(filepath.Join("assets", pick.name)), hex.EncodeToString(sum[:]), nil
}

// readGlobalPluginPackage 读取插件包目录的 package.json（有效返回 pkg,true）。
// 判定与 LoadGlobalPlugins 同源：name + main 非空。
func readGlobalPluginPackage(pkgDir string) (GlobalPluginPackage, bool) {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return GlobalPluginPackage{}, false
	}
	var pkg GlobalPluginPackage
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Name == "" || pkg.Main == "" {
		return GlobalPluginPackage{}, false
	}
	return pkg, true
}

// UIBundleClientDirective 为「含 dsh.ui 段但无 client.js」的包提供 clientCode 装载
// 回退指令（旧 /api/plugins+clientCode 装载路径可据此定位 bundle URL 加载）。
// 有 client.js 的包不受影响（clientCode 仍为内联源码）。
func UIBundleClientDirective(pkgDir, name string) string {
	assetRel, _, err := resolveUIBundle(pkgDir, name)
	if err != nil {
		return ""
	}
	// 形如：//@paircode/ui-bundle:/plugins-assets/ui-editor/assets/ui-editor.js
	return "//@paircode/ui-bundle:" + "/plugins-assets/" + name + "/" + assetRel
}
