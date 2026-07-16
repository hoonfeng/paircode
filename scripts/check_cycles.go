// 循环依赖检测。用 go list 构建 import 图，DFS 检测环。
// 与 go mod tidy + go vet 互补：go vet 检不完循环包导入。
//
//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// 列出所有包
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "go list 失败: %v\n", err)
		os.Exit(1)
	}

	type pkg struct {
		Name    string
		ImportPath string
		Imports   []string
	}
	var pkgs []pkg
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			break
		}
		pkgs = append(pkgs, p)
	}

	// 构建导入图
	imports := map[string][]string{}
	for _, p := range pkgs {
		imports[p.ImportPath] = p.Imports
	}

	// DFS 检测环
	module := "github.com/hoonfeng/paircode"
	visited := map[string]bool{}
	stack := map[string]bool{}
	var cycles [][]string

	var dfs func(path string, chain []string)
	dfs = func(path string, chain []string) {
		if stack[path] {
			// 找到环：从 chain 中截取环
			cycle := []string{path}
			for i := len(chain) - 1; i >= 0; i-- {
				cycle = append(cycle, chain[i])
				if chain[i] == path {
					break
				}
			}
			// 反转
			for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
				cycle[i], cycle[j] = cycle[j], cycle[i]
			}
			cycles = append(cycles, cycle)
			return
		}
		if visited[path] {
			return
		}
		visited[path] = true
		stack[path] = true
		for _, imp := range imports[path] {
			if strings.HasPrefix(imp, module) {
				dfs(imp, append(chain, path))
			}
		}
		stack[path] = false
	}

	for _, p := range pkgs {
		if strings.HasPrefix(p.ImportPath, module) {
			dfs(p.ImportPath, nil)
		}
	}

	if len(cycles) > 0 {
		fmt.Fprintf(os.Stderr, "❌ 发现 %d 个循环依赖:\n", len(cycles))
		for _, cycle := range cycles {
			fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(cycle, " → "))
		}
		os.Exit(1)
	}
	fmt.Println("✅ 无循环依赖")
}
