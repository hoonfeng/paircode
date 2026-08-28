// 文件快照系统 —— 在修改文件前自动备份，改坏了一键从快照恢复。
// 快照存储在 {workspaceRoot}/.pair/snapshots/ 下，按原文件相对路径组织。
// 比 git 更轻量：无需提交，自动触发，恢复零成本。

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SnapshotDirRel .pair/snapshots/ 相对路径常量。
const SnapshotDirRel = ".pair" + string(filepath.Separator) + "snapshots"

// MaxSnapshotsPerFile 每个文件最多保留的快照数。超过时删除最旧的。
const MaxSnapshotsPerFile = 20

// snapshotRoot 计算 {root}/.pair/snapshots/ 绝对路径。
func snapshotRoot(root string) string {
	return filepath.Join(root, SnapshotDirRel)
}

// SnapshotBeforeWrite 在修改文件前创建快照。
// root 为工作区根，absPath 为待修改文件的绝对路径。
// 若文件不存在（新建）则跳过。快照存储为 {root}/.pair/snapshots/{relPath}/{timestamp}。
// 返回快照标识，供恢复时引用。无快照（新建文件）返回空字符串。
func SnapshotBeforeWrite(root, absPath string) (snapshotID string, err error) {
	// 文件尚不存在 → 新建，无需快照
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", nil
	}

	relPath, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", fmt.Errorf("snapshot: 无法计算相对路径: %w", err)
	}
	// 规范化分隔符
	relPath = filepath.ToSlash(relPath)

	// 创建快照目录
	snapDir := filepath.Join(snapshotRoot(root), relPath)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		return "", fmt.Errorf("snapshot: 创建快照目录失败: %w", err)
	}

	// 读取文件内容
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("snapshot: 读取文件失败: %w", err)
	}

	// 快照文件名：时间戳（精确到微秒，避免冲突）
	ts := time.Now().Format("20060102_150405.000000")
	snapFile := filepath.Join(snapDir, ts)
	if err := os.WriteFile(snapFile, data, 0o644); err != nil {
		return "", fmt.Errorf("snapshot: 写入快照失败: %w", err)
	}

	// 清理旧快照：只保留最近 MaxSnapshotsPerFile 份，删除最旧的
	cleanupOldSnapshots(snapDir)

	// 快照标识 = relPath + "@" + timestamp，供 restore_snapshot 工具定位
	snapshotID = relPath + "@" + ts
	return snapshotID, nil
}

// cleanupOldSnapshots 清理 snaphot 目录，只保留最近 MaxSnapshotsPerFile 份。
func cleanupOldSnapshots(snapDir string) {
	entries, err := os.ReadDir(snapDir)
	if err != nil || len(entries) <= MaxSnapshotsPerFile {
		return
	}
	// 按文件名（时间戳）排序，只保留最新的 MaxSnapshotsPerFile 个
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name() // 文件名升序=时间升序
	})
	toDelete := len(entries) - MaxSnapshotsPerFile
	for i := 0; i < toDelete; i++ {
		os.Remove(filepath.Join(snapDir, entries[i].Name()))
	}
}

// RestoreSnapshot 从快照恢复指定文件。
// snapshotIndex：0=最旧（原始），-1=最新，>0=第 N 份（1 基从最旧算）。
// 默认传 0 使多次改错后仍能回到最初状态。
func RestoreSnapshot(root, relPath string, snapshotIndex int) error {
	relPath = filepath.ToSlash(filepath.Clean(relPath))
	snapDir := filepath.Join(snapshotRoot(root), relPath)

	// 列出快照文件
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		return fmt.Errorf("snapshot: 无快照可用（%s）", relPath)
	}

	// 收集快照文件，按时间戳升序（最旧在前）
	snapshots := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			snapshots = append(snapshots, e.Name())
		}
	}
	if len(snapshots) == 0 {
		return fmt.Errorf("snapshot: 无快照可用（%s）", relPath)
	}
	sort.Strings(snapshots) // 升序：最旧在前

	var target string
	switch {
	case snapshotIndex == 0:
		target = snapshots[0] // 最旧（原始）
	case snapshotIndex == -1:
		target = snapshots[len(snapshots)-1] // 最新
	case snapshotIndex >= 1 && snapshotIndex <= len(snapshots):
		target = snapshots[snapshotIndex-1] // 第 N 份（1 基）
	default:
		return fmt.Errorf("snapshot: index %d 越界（可用 1-%d，0=最旧，-1=最新）", snapshotIndex, len(snapshots))
	}

	// 读快照内容
	data, err := os.ReadFile(filepath.Join(snapDir, target))
	if err != nil {
		return fmt.Errorf("snapshot: 读取快照失败: %w", err)
	}

	// 写回原文件
	absPath := filepath.Join(root, relPath)
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return fmt.Errorf("snapshot: 恢复文件失败: %w", err)
	}
	return nil
}

// ListSnapshots 列出指定文件的所有快照（按时间倒序）。
// root 为工作区根，relPath 为工作区相对路径。返回快照标识列表。
func ListSnapshots(root, relPath string) ([]string, error) {
	relPath = filepath.ToSlash(relPath)
	snapDir := filepath.Join(snapshotRoot(root), relPath)

	entries, err := os.ReadDir(snapDir)
	if err != nil {
		return nil, nil // 无快照不报错
	}

	snapshots := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			snapshots = append(snapshots, relPath+"@"+e.Name())
		}
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i] > snapshots[j] // 倒序（最新在前）
	})
	return snapshots, nil
}

// RegisterSnapshotTools 注册快照相关工具。
// 包括 restore_snapshot（从快照恢复文件）和 list_snapshots（查看快照列表）。
func RegisterSnapshotTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "restore_snapshot",
		Description: "从快照恢复指定文件。快照在 edit/multi_edit/write 修改前自动创建。" +
			"默认恢复到最旧快照（原始文件）。可用 list_snapshots 查看快照列表。" +
			"指定 index 参数恢复特定版本（0=最旧原始文件，-1=最新，1~N=第 N 份从最旧算）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"path":  strProp("要恢复的文件路径（工作区相对路径，如 \"cmd/main.go\"）"),
				"index": strProp("可选快照索引：0=最旧(原始/默认)，-1=最新，1~N=第 N 份"),
			},
			"required": []string{"path"},
		},
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			relPath := argStr(args, "path")
			if relPath == "" {
				return "", fmt.Errorf("path 不能为空")
			}
			idxStr := argStr(args, "index")
			idx := 0 // 默认恢复到最旧（原始）
			if idxStr != "" {
				fmt.Sscanf(idxStr, "%d", &idx)
			}

			// 解析为绝对路径并验证在工作区内
			absPath, err := resolvePath(root, relPath)
			if err != nil {
				return "", err
			}

			// 重新计算相对路径（resolvePath 可能已修正路径格式）
			resolvedRel, _ := filepath.Rel(root, absPath)
			if err := RestoreSnapshot(root, resolvedRel, idx); err != nil {
				return "", err
			}

			label := "最旧（原始）"
			switch idx {
			case 0:
				label = "最旧（原始）"
			case -1:
				label = "最新"
			default:
				label = fmt.Sprintf("第 %d 份", idx)
			}
			return "✅ 已从快照恢复 " + relPath + "（" + label + "）", nil
		},
	})

	r.Register(&Tool{
		Name:        "list_snapshots",
		Description: "列出指定文件的所有可用快照（按时间倒序，带索引号）。用 restore_snapshot 的 index 参数可恢复指定版本。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"path": strProp("文件路径（工作区相对路径）"),
			},
			"required": []string{"path"},
		},
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			relPath := argStr(args, "path")
			if relPath == "" {
				return "", fmt.Errorf("path 不能为空")
			}
			snapshots, err := ListSnapshots(root, relPath)
			if err != nil {
				return "", err
			}
			if len(snapshots) == 0 {
				return "（无快照。文件修改时自动创建快照。）", nil
			}
			// 升序列出（从最旧到最新），显示索引号
			// ListSnapshots 返回倒序，反转
			sorted := make([]string, len(snapshots))
			for i, s := range snapshots {
				sorted[len(sorted)-1-i] = s
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("文件 %s 的快照（共 %d 个，索引 1=%d=最旧原始）：\n\n", relPath, len(snapshots), 1))
			for i, s := range sorted {
				parts := strings.SplitN(s, "@", 2)
				ts := ""
				if len(parts) == 2 {
					ts = formatSnapshotTime(parts[1])
				}
				b.WriteString(fmt.Sprintf("  [%d] • %s\n", i+1, ts))
			}
			b.WriteString(fmt.Sprintf("\n用法：\n"))
			b.WriteString(fmt.Sprintf("  restore_snapshot(path=%q)           → 恢复最旧（原始）\n", relPath))
			b.WriteString(fmt.Sprintf("  restore_snapshot(path=%q, index=-1) → 恢复最新\n", relPath))
			b.WriteString(fmt.Sprintf("  restore_snapshot(path=%q, index=1)  → 恢复第 1 份\n", relPath))
			return b.String(), nil
		},
	})
}

// formatSnapshotTime 把时间戳文件名转为可读格式。
func formatSnapshotTime(ts string) string {
	t, err := time.Parse("20060102_150405.000000", ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}
