package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	data, _ := os.ReadFile("F:/syproject/gou-ide/cmd/desktop/web-ui-minimal/dist/assets/app-CHLaWOgr.js")
	s := string(data)

	// Search for mount wrapper around "mount=n=>{"
	i := strings.Index(s, "mount=n=>{")
	if i >= 0 {
		fmt.Printf("Found mount=n=>{ at offset %d\n", i)
		end := i + 400
		if end > len(s) {
			end = len(s)
		}
		fmt.Printf("Context:\n%s\n", s[i:end])
	} else {
		// Try other patterns
		for _, pat := range []string{"t.mount=n=>{", "mount=", ".mount=n"} {
			j := strings.Index(s, pat)
			if j >= 0 {
				fmt.Printf("Found %q at offset %d\n", pat, j)
				end := j + 300
				if end > len(s) {
					end = len(s)
				}
				fmt.Printf("Context:\n%s\n", s[j:end])
				break
			}
		}
	}
}
