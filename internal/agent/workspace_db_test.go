package agent

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestPruneCodeGraphs 验证：清理已移除工作区的图谱缓存。
func TestPruneCodeGraphs(t *testing.T) {
	oldRoots := WorkspaceRoots
	defer func() { WorkspaceRoots = oldRoots }()
	WorkspaceRoots = []string{"C:/keep/one"}

	// 预置两个项目缓存（键为 normRoot 格式：Clean 后小写）
	keepKey := normRoot("C:/keep/one")
	rmKey := normRoot("C:/removed/two")
	cgEntriesMu.Lock()
	cgEntries[keepKey] = &cgEntry{}
	cgEntries[rmKey] = &cgEntry{}
	cgEntriesMu.Unlock()
	defer func() {
		cgEntriesMu.Lock()
		delete(cgEntries, keepKey)
		delete(cgEntries, rmKey)
		cgEntriesMu.Unlock()
	}()

	PruneCodeGraphs(WorkspaceRoots)
	cgEntriesMu.Lock()
	_, keep := cgEntries[keepKey]
	_, removed := cgEntries[rmKey]
	cgEntriesMu.Unlock()
	if !keep {
		t.Fatalf("仍活跃的工作区缓存不应被清除")
	}
	if removed {
		t.Fatalf("已移除工作区的缓存应被清除")
	}
}

// TestIsActiveRoot 验证：root 是否仍为当前工作区之一。
func TestIsActiveRoot(t *testing.T) {
	oldRoots := WorkspaceRoots
	defer func() { WorkspaceRoots = oldRoots }()
	WorkspaceRoots = []string{"F:\\syproject\\gou-ide", "F:\\syproject\\wb-ui"}

	if !isActiveRoot("f:\\syproject\\GOU-IDE") {
		t.Fatalf("大小写/分隔符差异应仍判定为活跃")
	}
	if isActiveRoot("F:\\syproject\\removed") {
		t.Fatalf("已移除工作区不应判定为活跃")
	}
}

// TestSetWorkspaceRootClosesOldDB 验证：SetWorkspaceRoot 切换 root 后（多工作区隔离），
// ★ 2026-08-23 语义更新：旧 root 的 DB/store 句柄保留在缓存（运行中会话绑定旧工作区，
//   需要继续读写）——切换不再关闭；删除工作区时经 CloseWorkspaceDB 显式关闭（Windows
//   句柄释放 → 可删除该工作区 .pair 目录）。
func TestSetWorkspaceRootClosesOldDB(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	m := NewSessionManager()
	defer func() {
		// 测试收尾：关闭 dir2 的连接，避免 TempDir 清理被占用
		if db := m.SQLiteDB(); db != nil {
			if raw, ok := db.RawDB().(*sql.DB); ok {
				_ = raw.Close()
			}
		}
	}()

	m.SetWorkspaceRoot(dir1)
	if m.RawDB() == nil {
		t.Fatalf("dir1 后应有 DB")
	}
	// 切换 root
	m.SetWorkspaceRoot(dir2)
	if m.RawDB() == nil {
		t.Fatalf("dir2 后应有 DB")
	}
	// ★ 2026-08-23 多工作区隔离：旧 root 句柄保留（切换不关闭）——正在执行的会话
	//   绑定 dir1，其工具/持久化仍可访问 dir1 的 store/DB。
	if m.RawDBFor(dir1) == nil {
		t.Fatalf("旧工作区 dir1 的 DB 句柄应保留（运行中会话隔离）")
	}
	if m.StoreFor(dir1) == nil {
		t.Fatalf("旧工作区 dir1 的 store 句柄应保留")
	}
	// ★ 关键：CloseWorkspaceDB(dir1) 后 dir1 的 .pair 应可删除（句柄显式释放）
	m.CloseWorkspaceDB(dir1)
	if err := os.RemoveAll(filepath.Join(dir1, ".pair")); err != nil {
		t.Fatalf("删除工作区后 .pair 无法删除（DB 句柄未释放）: %v", err)
	}
	// dir1 根目录也可删（不再被占用）
	if err := os.RemoveAll(dir1); err != nil {
		t.Fatalf("删除工作区后根目录无法删除: %v", err)
	}
}
