//go:build windows

package gitpanel

import (
	"os"
	"testing"
)

// mustWrite 测试内写文件（原在 main 的 search_test.go 共用，gitpanel 抽出后留本地一份）。
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
