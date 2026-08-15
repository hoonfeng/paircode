package agent

import (
	"testing"
)

// TestStaleRealKBs 真实项目知识库（gou-ide + wb-ui）不再有误报过期条目。
func TestStaleRealKBs(t *testing.T) {
	roots := []string{`F:\syproject\gou-ide`, `F:\syproject\wb-ui`}
	got := buildKBStaleness(roots)
	if got != "" {
		t.Errorf("真实知识库不应有误报过期条目：\n%s", got)
	} else {
		t.Log("真实知识库过期检查：无误报 ✓")
	}
}
