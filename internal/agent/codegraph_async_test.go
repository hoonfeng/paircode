package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/pkg/codegraph"
)

// TestGetCodeGraphAsyncNotBlocking 验证：缓存图存在时 getCodeGraph 立即返回
// （即使已到 30s 变更检测点，增量构建也只会在后台执行，不阻塞调用方）。
func TestGetCodeGraphAsyncNotBlocking(t *testing.T) {
	root := t.TempDir()
	key := normRoot(root)

	// 预置缓存图（lastCheck=31s 前 → 触发变更检测）
	g := codegraph.NewGraph()
	g.AddEntity(&codegraph.Entity{
		ID: "x:func:a", Kind: codegraph.EntityFunction, Name: "a",
		FilePath: "a.go", Line: 1,
	})
	cgEntriesMu.Lock()
	cgEntries[key] = &cgEntry{graph: g, lastCheck: time.Now().Add(-31 * time.Second)}
	cgEntriesMu.Unlock()
	defer func() {
		cgEntriesMu.Lock()
		delete(cgEntries, key)
		cgEntriesMu.Unlock()
	}()

	start := time.Now()
	got, err := getCodeGraph(root)
	elapsed := time.Since(start)

	if err != nil || got == nil {
		t.Fatalf("getCodeGraph 应返回缓存图, err=%v", err)
	}
	if got.Stats().EntityCount != 1 {
		t.Fatalf("缓存图实体数应为 1, got %d", got.Stats().EntityCount)
	}
	// 异步化后：返回应立即（后台构建 16s 级，不能阻塞在调用里）
	if elapsed > 2*time.Second {
		t.Fatalf("getCodeGraph 被阻塞了 %v（应立即返回，构建在后台）", elapsed)
	}
	t.Logf("getCodeGraph 返回耗时: %v（未阻塞）", elapsed)

	// 等待后台构建完成（空目录构建应很快）
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		cgEntriesMu.Lock()
		e := cgEntries[key]
		cgEntriesMu.Unlock()
		if e != nil && time.Since(e.lastCheck) < 10*time.Second {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestGetCachedCodeGraphOnly 验证：getCachedCodeGraph 不触发构建（临时 root 无
// graph.json，若触发构建会创建 .pair/codegraph 目录；只读缓存不应创建）。
func TestGetCachedCodeGraphOnly(t *testing.T) {
	root := t.TempDir()
	key := normRoot(root)

	if g, ok := getCachedCodeGraph(root); ok || g != nil {
		t.Fatalf("空缓存应返回 nil,false")
	}

	g := codegraph.NewGraph()
	cgEntriesMu.Lock()
	cgEntries[key] = &cgEntry{graph: g, lastCheck: time.Now()}
	cgEntriesMu.Unlock()
	defer func() {
		cgEntriesMu.Lock()
		delete(cgEntries, key)
		cgEntriesMu.Unlock()
	}()

	got, ok := getCachedCodeGraph(root)
	if !ok || got == nil {
		t.Fatalf("缓存命中应返回图")
	}
	// 不应创建任何构建产物
	if _, err := os.Stat(filepath.Join(root, ".pair", "codegraph")); err == nil {
		t.Fatalf("getCachedCodeGraph 不应触发构建（产生了 codegraph 目录）")
	}
}

// TestBuildCodeGraphStatsFast 验证：注入路径 buildCodeGraphStats 走缓存，耗时忽略不计
// （即使图谱已 30s 未检测，也不会同步触发构建）。
func TestBuildCodeGraphStatsFast(t *testing.T) {
	root := t.TempDir()
	oldRoots := WorkspaceRoots
	WorkspaceRoots = []string{root}
	defer func() { WorkspaceRoots = oldRoots }()

	// 无缓存：应返回空（不触发构建、不报错）
	if s := buildCodeGraphStats(WorkspaceRoots); s != "" {
		t.Fatalf("无缓存时应返回空串, got %q", s)
	}
	if _, err := os.Stat(filepath.Join(root, ".pair", "codegraph")); err == nil {
		t.Fatalf("buildCodeGraphStats 不应触发构建")
	}

	// 有缓存：返回统计
	g := codegraph.NewGraph()
	g.AddEntity(&codegraph.Entity{ID: "p:file:main.go", Kind: codegraph.EntityFile, Name: "main.go", FilePath: "main.go", Line: 1})
	key := normRoot(root)
	cgEntriesMu.Lock()
	cgEntries[key] = &cgEntry{graph: g, lastCheck: time.Now().Add(-31 * time.Second)}
	cgEntriesMu.Unlock()
	defer func() {
		cgEntriesMu.Lock()
		delete(cgEntries, key)
		cgEntriesMu.Unlock()
	}()

	start := time.Now()
	s := buildCodeGraphStats(WorkspaceRoots)
	elapsed := time.Since(start)
	t.Logf("buildCodeGraphStats 耗时: %v, len=%d", elapsed, len(s))
	if elapsed > time.Second {
		t.Fatalf("buildCodeGraphStats 不应触发同步构建（耗时 %v）", elapsed)
	}
	if s == "" {
		t.Fatalf("有缓存时应返回统计, got empty")
	}
}
