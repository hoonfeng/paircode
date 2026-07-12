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
func RestoreSnapshot(root, relPath string) error {
	relPath = filepath.ToSlash(relPath)
	snapDir := filepath.Join(snapshotRoot(root), relPath)

	// 列出快照文件
	entries, err := os.ReadDir(snapDir)
	if err != nil {
		return fmt.Errorf("snapshot: 无快照可用（%s）", relPath)
	}

	// 按文件名（时间戳）排序，取最新
	snapshots := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			snapshots = append(snapshots, e.Name())
		}
	}
	if len(snapshots) == 0 {
		return fmt.Errorf("snapshot: 无快照可用（%s）", relPath)
	}
	sort.Strings(snapshots)
	latest := snapshots[len(snapshots)-1]

	// 读快照内容
	data, err := os.ReadFile(filepath.Join(snapDir, latest))
	if err != nil {
		return fmt.Errorf("snapshot: 读取快照失败: %w", err)
	}

	// 写回原文件
	absPath := filepath.Join(root, relPath)
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
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
		Description: "从最近的快照恢复指定文件。快照在 edit_file/multi_edit/write_file 修改前自动创建。" +
			"先用 list_snapshots 查看可用快照，再用本工具恢复到修改前的内容。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"path": strProp("要恢复的文件路径（工作区相对路径，如 \"cmd/main.go\"）"),
			},
			"required": []string{"path"},
		},
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			relPath := argStr(args, "path")
			if relPath == "" {
				return "", fmt.Errorf("path 不能为空")
			}

			// 解析为绝对路径并验证在工作区内
			absPath, err := resolvePath(root, relPath)
			if err != nil {
				return "", err
			}

			// 重新计算相对路径（resolvePath 可能已修正路径格式）
			resolvedRel, _ := filepath.Rel(root, absPath)
			if err := RestoreSnapshot(root, resolvedRel); err != nil {
				return "", err
			}

			return "✅ 已从快照恢复 " + relPath + " 到修改前的内容", nil
		},
	})

	r.Register(&Tool{
		Name:        "list_snapshots",
		Description: "列出指定文件的所有可用快照（按时间倒序）。",
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
			var b strings.Builder
			b.WriteString(fmt.Sprintf("文件 %s 的快照（共 %d 个）：\n\n", relPath, len(snapshots)))
			for _, s := range snapshots {
				parts := strings.SplitN(s, "@", 2)
				ts := ""
				if len(parts) == 2 {
					ts = formatSnapshotTime(parts[1])
				}
				b.WriteString(fmt.Sprintf("  • %s\n", ts))
			}
			b.WriteString("\n使用 restore_snapshot(path=" + relPath + ") 恢复到最近一份快照。")
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
