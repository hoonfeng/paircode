//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type fix struct {
	old, new string
}

func main() {
	fixes := []fix{
		{`github.com/hoonfeng/paircode/internal/core`, `github.com/hoonfeng/paircode/internal/core`},
		{`github.com/hoonfeng/paircode/internal/agenttools`, `github.com/hoonfeng/paircode/internal/agenttools`},
	}

	count := 0
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".pair") {
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
		for _, f := range fixes {
			if strings.Contains(content, f.old) {
				content = strings.ReplaceAll(content, f.old, f.new)
				changed = true
			}
		}
		if changed {
			os.WriteFile(path, []byte(content), 0644)
			fmt.Printf("  [FIXED] %s\n", path)
			count++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n完成！共修正 %d 个文件。\n", count)
}
