package main

import (
	"fmt"

	"wb-ui/css"
)

func main() {
	// 测试 unicode 类名解析（CM6 的 module class ͼ1）
	p := css.NewParser(".ͼ1 { position: relative; display: flex } .cm-editor { height: 100% } .cm-line { display: block }")
	rules := p.ParseStyleSheet()
	fmt.Printf("rules=%d\n", len(rules))
	for _, r := range rules {
		if sr, ok := r.(*css.StyleRule); ok {
			fmt.Printf("  rule selectors=%v decls=%d\n", sr.Selectors, len(sr.Declarations))
		}
	}
}
