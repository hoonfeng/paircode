package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/agent"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取工作目录失败: %v\n", err)
		os.Exit(1)
	}

	convDir := filepath.Join(root, ".pair", "conversations")
	entries, err := os.ReadDir(convDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取 %s 失败: %v\n", convDir, err)
		os.Exit(1)
	}

	store := agent.NewMessageStore(root)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "conv_") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		convID := strings.TrimSuffix(name, ".jsonl")
		fmt.Printf("合并 %s ... ", convID)
		if err := store.MergeAllAssistantRuns(convID); err != nil {
			fmt.Printf("失败: %v\n", err)
		} else {
			fmt.Println("完成")
		}
	}
	fmt.Println("全部合并完成")
}
