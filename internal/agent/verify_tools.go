package agent

// verify_tools.go — 记忆与项目知识库的自动过期验证工具。
//
// 记忆和知识库随时间推移可能包含对已删除文件/符号的引用，
// 这些过时信息会误导 Agent。每隔一段时间运行一次可保持数据新鲜。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/pkg/memory"
	"github.com/hoonfeng/paircode/pkg/verify"
)

// registerVerifyTools 注册记忆/知识库验证工具。
func registerVerifyTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "memory_verify",
		UsageGuide: "验证所有记忆条目引用的文件和目录是否仍然存在。过时记忆会误导 agent，建议定期运行。比手动检查更高效（自动解析引用路径并检测有效性）。",
		Description: "验证所有记忆条目中引用的文件和目录是否仍然存在。" +
			"如果条目引用了已不存在的文件，可能是过时信息，建议更新或删除。" +
			"返回验证报告，包含每个过期条目的问题描述。",
		Parameters: objSchema(props{}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return runMemoryVerify()
		},
	})

	r.Register(&Tool{
		Name: "project_info_verify",
		UsageGuide: "验证知识库条目引用的文件和目录是否仍然存在。项目重构后文件移动可能导致旧引用失效，运行此工具可发现并清理过时条目。",
		Description: "验证所有知识库条目中引用的文件和目录是否仍然存在。" +
			"如果条目引用了已不存在的文件/目录，可能是过时信息，建议更新或删除。" +
			"返回验证报告，包含每个过期条目的问题描述。",
		Parameters: objSchema(props{}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return runKBVerify()
		},
	})
}

// runMemoryVerify 执行记忆验证，返回可读报告文本。
func runMemoryVerify() (string, error) {
	roots := WorkspaceRoots
	if len(roots) == 0 {
		return "无工作区，跳过验证", nil
	}

	v := &verify.Verifier{WorkspaceRoots: roots}

	// 从 memory 包读取所有条目
	entries := memory.List()
	memories := make([]verify.MemoryEntry, 0, len(entries))
	for _, e := range entries {
		// 尝试读取记忆正文（.pair/memory/<name>.md）
		body := readMemoryBody(e.ID)
		memories = append(memories, verify.MemoryEntry{
			ID:        e.ID,
			Title:     e.Title,
			Summary:   e.Summary,
			KeyPoints: e.KeyPoints,
			// body 只用作引用检查，不传给 report
		})
		_ = body
	}

	report := v.VerifyAll(memories, nil)
	return formatVerifyReport("记忆", report), nil
}

// runKBVerify 执行知识库验证，返回可读报告文本。
func runKBVerify() (string, error) {
	roots := WorkspaceRoots
	if len(roots) == 0 {
		return "无工作区，跳过验证", nil
	}

	v := &verify.Verifier{WorkspaceRoots: roots}

	// 从知识库目录读取所有条目
	kbDir := projectInfoDir(roots[0])
	entries := scanInfoEntries(kbDir)
	kbs := make([]verify.KBEntry, 0, len(entries))
	for _, e := range entries {
		kbs = append(kbs, verify.KBEntry{
			Path:    e.Path,
			Title:   e.Title,
			Content: e.Content,
		})
	}

	report := v.VerifyAll(nil, kbs)
	return formatVerifyReport("知识库", report), nil
}

// formatVerifyReport 格式化验证报告为可读文本。
func formatVerifyReport(source string, r *verify.Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## %s 验证报告 (%s)\n", source, r.CheckedAt))
	b.WriteString(fmt.Sprintf("共检查 %d 条，正常 %d 条", r.MemoryCount+r.KBCount, r.OKCount))
	if len(r.Stale) > 0 {
		b.WriteString(fmt.Sprintf("，发现 **%d 条可能过期**：\n\n", len(r.Stale)))
		for i, s := range r.Stale {
			b.WriteString(fmt.Sprintf("%d. **%s** (%s)\n", i+1, s.Title, s.ID))
			for _, issue := range s.Issues {
				b.WriteString(fmt.Sprintf("   - %s\n", issue))
			}
		}
		b.WriteString("\n建议：\n")
		b.WriteString("- 对过期条目可删除或用更新类工具刷新（工具名称与用法见 tools 参数 schema）\n")
		b.WriteString("- 对知识库过期条目可删除或用更新类工具刷新\n")
		b.WriteString("- 定期执行过期检查类工具保持数据新鲜\n")
	} else {
		b.WriteString("，全部正常。\n")
	}
	return b.String()
}

// readMemoryBody 尝试读取记忆文件正文（.pair/memory/<name>.md），用于引用检查。
func readMemoryBody(name string) string {
	for _, root := range WorkspaceRoots {
		path := filepath.Join(root, ".pair", "memory", name+".md")
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
	}
	return ""
}

// ── 自动验证集成 ──────────────────────────────────────

// AutoVerifyStale 由外部（如 loop 或编排循环）调用，自动检查并报告过期条目。
// 返回一个字符串摘要（无过期→空字符串），供 Loop 注入到上下文或日志。
func AutoVerifyStale() string {
	roots := WorkspaceRoots
	if len(roots) == 0 {
		return ""
	}
	v := &verify.Verifier{WorkspaceRoots: roots}

	// 检查记忆
	entries := memory.List()
	mems := make([]verify.MemoryEntry, 0, len(entries))
	for _, e := range entries {
		mems = append(mems, verify.MemoryEntry{
			ID:      e.ID,
			Title:   e.Title,
			Summary: e.Summary,
		})
	}

	r := v.VerifyAll(mems, nil)
	if len(r.Stale) == 0 {
		return ""
	}

	// 只报告前 5 条，避免撑爆上下文
	var b strings.Builder
	b.WriteString(fmt.Sprintf("⚠️ 检测到 %d 条可能过期的记忆/知识库条目。\n", len(r.Stale)))
	for i, s := range r.Stale {
		if i >= 5 {
			b.WriteString(fmt.Sprintf("  …等 %d 条\n", len(r.Stale)-5))
			break
		}
		b.WriteString(fmt.Sprintf("- %s「%s」", s.Source, s.Title))
		if len(s.Issues) > 0 {
			b.WriteString(": " + s.Issues[0])
		}
		b.WriteString("\n")
	}
	b.WriteString("可运行过期检查类工具查看完整报告（工具名称与用法见 tools 参数 schema）。")
	return b.String()
}
