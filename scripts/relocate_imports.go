//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var replacements = map[string]string{
	`github.com/hoonfeng/paircode/internal/agent`:       `github.com/hoonfeng/paircode/internal/agent`,
	`github.com/hoonfeng/paircode/internal/agenttools`:  `github.com/hoonfeng/paircode/internal/agenttools`,
	`github.com/hoonfeng/paircode/internal/core`:        `github.com/hoonfeng/paircode/internal/core`,
	// debugger 包已移除，被通用 debug_tools.go 替代
	`github.com/hoonfeng/paircode/internal/hook`:        `github.com/hoonfeng/paircode/internal/hook`,
	`github.com/hoonfeng/paircode/internal/codetypes`:   `github.com/hoonfeng/paircode/internal/codetypes`,
	`github.com/hoonfeng/paircode/internal/uiapi`:       `github.com/hoonfeng/paircode/internal/uiapi`,
	`github.com/hoonfeng/paircode/internal/pty`:         `github.com/hoonfeng/paircode/internal/pty`,
	`github.com/hoonfeng/paircode/internal/permission`:  `github.com/hoonfeng/paircode/internal/permission`,
	`github.com/hoonfeng/paircode/internal/provider`:    `github.com/hoonfeng/paircode/internal/provider`,
	`github.com/hoonfeng/paircode/internal/jobs`:        `github.com/hoonfeng/paircode/internal/jobs`,
	`github.com/hoonfeng/paircode/internal/langsrv`:     `github.com/hoonfeng/paircode/internal/langsrv`,
	`github.com/hoonfeng/paircode/internal/roleprompts`: `github.com/hoonfeng/paircode/internal/roleprompts`,
	`github.com/hoonfeng/paircode/internal/vterm`:       `github.com/hoonfeng/paircode/internal/vterm`,
	`github.com/hoonfeng/paircode/internal/ui/marketplace`: `github.com/hoonfeng/paircode/internal/ui/marketplace`,
	`github.com/hoonfeng/paircode/internal/ui/mcp`:      `github.com/hoonfeng/paircode/internal/ui/mcp`,
	`github.com/hoonfeng/paircode/internal/ui/skills`:   `github.com/hoonfeng/paircode/internal/ui/skills`,
}

func main() {
	count := 0
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == ".pair") {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		changed := false
		for old, new := range replacements {
			if strings.Contains(content, old) {
				content = strings.ReplaceAll(content, old, new)
				changed = true
			}
		}
		if changed {
			os.WriteFile(path, []byte(content), 0644)
			fmt.Printf("  [UPDATED] %s\n", path)
			count++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n完成！共更新 %d 个文件。\n", count)
}
