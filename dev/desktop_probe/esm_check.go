// Command esm_check 检查 dist bundle 是否包含真正的 ESM import/export 语句（行首或 `;import` 形式）
package main

import (
	"fmt"
	"os"
	"regexp"
)

func main() {
	path := `cmd\companion\web-ui\dist\assets\index-DBMdwrBl.js`
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("ERR:", err)
		return
	}
	src := string(data)
	fmt.Printf("len=%d\n", len(src))
	// 真正的 ESM import 语句：行首（或 ; 后）import xxx from/import * as/import {/import '...
	re := regexp.MustCompile(`(?m)(^|[;}])\s*import\s+([*{'"a-zA-Z_$])`)
	matches := re.FindAllStringIndex(src, 5)
	fmt.Printf("true import stmts: %d\n", len(matches))
	for _, m := range matches {
		start := m[0]
		if start > 60 {
			start -= 60
		} else {
			start = 0
		}
		fmt.Printf("  ...%q\n", src[start:m[1]])
	}
	// export default / export { 语句
	re2 := regexp.MustCompile(`(?m)(^|[;}])\s*export\s+(default|\{)`)
	matches2 := re2.FindAllStringIndex(src, 5)
	fmt.Printf("true export stmts: %d\n", len(matches2))
	for _, m := range matches2 {
		start := m[0]
		if start > 60 {
			start -= 60
		} else {
			start = 0
		}
		fmt.Printf("  ...%q\n", src[start:m[1]])
	}
}
