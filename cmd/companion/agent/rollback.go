// rollback.go 对话回滚系统：按用户消息维度跟踪文件快照，支持一键回退到指定消息前的状态。
// 每次用户发送消息时，后续的文件编辑快照自动关联到该消息。
// 用户点击回退按钮时，恢复该消息关联的所有快照 + 删除该消息之后的对话历史。

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ─── 数据结构 ──────────────────────────────────────────────

// MsgSnapEntry 一条消息关联的快照。
type MsgSnapEntry struct {
	File       string `json:"file"`       // 工作区相对路径
	SnapshotID string `json:"snapshotId"` // file@timestamp（如 "a.go@20260716_120000.000000"）
}

// convSnapData 一个对话的 msgIdx → snapshots 映射。
type convSnapData struct {
	Snapshots map[int][]*MsgSnapEntry `json:"snapshots"` // msgIdx → []snapEntry
}

// rollbackData 持久化数据结构，按根目录+对话ID组织。
type rollbackData struct {
	Convs map[string]*convSnapData `json:"convs"` // key: convId
}

// ─── 全局跟踪器 ────────────────────────────────────────────

// SnapshotTracker 管理快照到消息的映射。
// 跨所有工作区根目录的对话共享一个跟踪器，但按 root+convId 隔离数据。
type SnapshotTracker struct {
	mu       sync.Mutex
	root     string           // 工作区根
	current  map[string]int  // convId → 当前正在处理的 msgIdx
	dirty    bool
}

// GlobalTracker 全局快照跟踪器实例。
var GlobalTracker *SnapshotTracker

// InitTracker 初始化全局跟踪器。
// root 为工作区根目录。
func InitTracker(root string) {
	GlobalTracker = &SnapshotTracker{
		root:    root,
		current: make(map[string]int),
	}
}

// SetCurrentMsg 设置指定对话的当前消息索引。
// 在用户消息被持久化后调用，后续的 SnapshotBeforeWrite 快照将关联到此消息。
func (t *SnapshotTracker) SetCurrentMsg(convId string, msgIdx int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current[convId] = msgIdx
	t.dirty = true
}

// Record 记录一个快照关联到当前消息。
// 由 SnapshotBeforeWrite 的包装函数调用。
func (t *SnapshotTracker) Record(root, convId string, msgIdx int, file, snapshotID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data := t.load()
	if data.Convs == nil {
		data.Convs = make(map[string]*convSnapData)
	}
	cd, ok := data.Convs[convId]
	if !ok {
		cd = &convSnapData{Snapshots: make(map[int][]*MsgSnapEntry)}
		data.Convs[convId] = cd
	}

	entry := &MsgSnapEntry{File: file, SnapshotID: snapshotID}
	cd.Snapshots[msgIdx] = append(cd.Snapshots[msgIdx], entry)

	return t.save(data)
}

// GetSnapshots 获取指定消息关联的所有快照。
func (t *SnapshotTracker) GetSnapshots(convId string, msgIdx int) []*MsgSnapEntry {
	t.mu.Lock()
	defer t.mu.Unlock()

	data := t.load()
	if data.Convs == nil {
		return nil
	}
	cd, ok := data.Convs[convId]
	if !ok {
		return nil
	}
	return cd.Snapshots[msgIdx]
}

// RemoveAfterMsg 删除指定消息及之后的所有跟踪数据。
func (t *SnapshotTracker) RemoveAfterMsg(convId string, msgIdx int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data := t.load()
	if data.Convs == nil {
		return nil
	}
	cd, ok := data.Convs[convId]
	if !ok {
		return nil
	}
	for idx := range cd.Snapshots {
		if idx >= msgIdx {
			delete(cd.Snapshots, idx)
		}
	}
	return t.save(data)
}

// ─── 存储 ──────────────────────────────────────────────────

func (t *SnapshotTracker) rollbackDir() string {
	return filepath.Join(t.root, RollbackDir)
}

func (t *SnapshotTracker) filePath() string {
	return filepath.Join(t.rollbackDir(), "msg-snapshots.json")
}

func (t *SnapshotTracker) load() *rollbackData {
	data := &rollbackData{Convs: make(map[string]*convSnapData)}
	raw, err := os.ReadFile(t.filePath())
	if err != nil {
		return data
	}
	json.Unmarshal(raw, data)
	if data.Convs == nil {
		data.Convs = make(map[string]*convSnapData)
	}
	return data
}

func (t *SnapshotTracker) save(data *rollbackData) error {
	dir := t.rollbackDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.filePath(), raw, 0644)
}

// ─── 回滚执行 ──────────────────────────────────────────────

// RollbackDir .pair/rollback/ 相对路径常量。
const RollbackDir = ".pair" + string(filepath.Separator) + "rollback"

// RollbackToMsg 回滚到指定消息之前的状态：
// 1. 恢复该消息关联的所有文件快照
// 2. 删除被恢复文件的后续快照
// 3. 截断对话历史（删除该消息之后的所有消息）
// root 为工作区根，convId 为对话 ID，msgIdx 为用户消息索引（0 基），
// store 为对话存储接口（用于截断对话历史）。
func RollbackToMsg(root, convId string, msgIdx int, store ConversationStore) error {
	// 读取快照映射
	tracker := &SnapshotTracker{root: root}
	snapshots := tracker.GetSnapshots(convId, msgIdx)
	if len(snapshots) == 0 {
		return fmt.Errorf("没有找到消息 %d 关联的快照", msgIdx)
	}

	// 按文件去重（一个文件可能有多次编辑，只恢复原始快照）
	seen := make(map[string]bool)
	var restored int
	for _, entry := range snapshots {
		if seen[entry.File] {
			continue
		}
		seen[entry.File] = true

		// 从 snapshotID 解析出时间戳
		parts := strings.SplitN(entry.SnapshotID, "@", 2)
		if len(parts) != 2 {
			continue
		}
		relPath := parts[0]
		ts := parts[1]

		// 读取快照文件内容
		snapFile := filepath.Join(root, SnapshotDirRel, relPath, ts)
		data, err := os.ReadFile(snapFile)
		if err != nil {
			return fmt.Errorf("读取快照失败 %s: %w", entry.SnapshotID, err)
		}

		// 写回原文件
		absPath := filepath.Join(root, relPath)
		if err := os.WriteFile(absPath, data, 0644); err != nil {
			return fmt.Errorf("恢复文件失败 %s: %w", relPath, err)
		}
		restored++
	}

	// 清理后续消息的跟踪数据
	if err := tracker.RemoveAfterMsg(convId, msgIdx); err != nil {
		return fmt.Errorf("清理跟踪数据失败: %w", err)
	}

	// 截断对话历史：只保留 msgIdx 条消息（msgIdx 是 0 基，so keep msgIdx+1 messages including the user msg itself）
	// The user wants to keep messages up to and including msgIdx, and delete everything after.
	// msgIdx is 0-based. So keep count = msgIdx + 1 messages.
	if store != nil {
		keepCount := msgIdx + 1
		if err := store.TruncateTo(convId, keepCount); err != nil {
			return fmt.Errorf("截断对话历史失败: %w", err)
		}
	}

	return nil
}

// SnapshotBeforeWriteWithTracking 在修改文件前创建快照并关联到当前消息。
// 如果 GlobalTracker 已初始化且当前有活跃消息，则自动关联。
// 否则仅创建快照（兼容旧行为）。
func SnapshotBeforeWriteWithTracking(root, absPath string) (snapshotID string, err error) {
	snapshotID, err = SnapshotBeforeWrite(root, absPath)
	if err != nil || snapshotID == "" {
		return snapshotID, err
	}

	// 如果全局跟踪器已初始化，关联快照到当前消息
	if GlobalTracker != nil {
		GlobalTracker.mu.Lock()
		convId := ""
		msgIdx := -1
		// 找最近一次有 current 的对话
		// 简单取第一个（通常只有一个活跃对话）
		for cid, idx := range GlobalTracker.current {
			convId = cid
			msgIdx = idx
			break
		}
		GlobalTracker.mu.Unlock()

		if convId != "" && msgIdx >= 0 {
			relPath, _ := filepath.Rel(root, absPath)
			relPath = filepath.ToSlash(relPath)
			GlobalTracker.Record(root, convId, msgIdx, relPath, snapshotID)
		}
	}

	return snapshotID, nil
}

// ─── GetTracker 暴露跟踪器供 web_server 使用 ──────────────

// GetTracker 返回全局跟踪器。
func GetTracker() *SnapshotTracker {
	return GlobalTracker
}

// RollbackDataFilePath 返回跟踪数据的文件路径（供测试/诊断用）。
func RollbackDataFilePath(root string) string {
	t := &SnapshotTracker{root: root}
	return t.filePath()
}

// ─── 排序支持 ──────────────────────────────────────────────

// SortMsgSnapshots 按文件名排序快照列表（输出稳定）。
func SortMsgSnapshots(entries []*MsgSnapEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].File < entries[j].File
	})
}
