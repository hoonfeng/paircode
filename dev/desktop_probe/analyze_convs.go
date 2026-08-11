package main

// analyze_convs3.go — 打印指定对话/索引附近的消息序列（检查 user→assistant→tool 配对）。

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type storedMsg struct {
	Idx     int     `json:"Idx"`
	Message message `json:"Message"`
}

type message struct {
	Role    string `json:"Role"`
	Content string `json:"Content"`
}

func short(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func dump(name, dir string, from, to int) {
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	defer f.Close()
	var msgs []storedMsg
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var sm storedMsg
		if err := json.Unmarshal([]byte(line), &sm); err != nil {
			continue
		}
		msgs = append(msgs, sm)
	}
	fmt.Printf("===== %s (%d 条) [%d..%d] =====\n", name, len(msgs), from, to)
	if to > len(msgs)-1 {
		to = len(msgs) - 1
	}
	for i := from; i <= to && i < len(msgs); i++ {
		m := msgs[i]
		fmt.Printf("[%04d] %-9s %s\n", i, m.Message.Role, short(m.Message.Content, 50))
	}
	fmt.Println()
}

func main() {
	dir := ".pair/conversations"
	dump("conv_1786105664116711500.jsonl", dir, 325, 350) // 08/10 最新对话 user→tool 异常
}
