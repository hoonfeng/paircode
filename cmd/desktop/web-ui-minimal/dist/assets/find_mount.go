package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	data, _ := os.ReadFile("F:/syproject/gou-ide/cmd/desktop/web-ui-minimal/dist/assets/app-CHLaWOgr.js")
	s := string(data)

	// Search for "s(i," which is the call to original mount
	idx := strings.Index(s, "s(i,")
	if idx < 0 {
		idx = strings.Index(s, "s(")
	}
	fmt.Printf("s( at offset: %d\n", idx)
	
	// Show surrounding context
	start := idx - 10
	if start < 0 { start = 0 }
	end := idx + 200
	if end > len(s) { end = len(s) }
	fmt.Printf("Context:\n%s\n", s[start:end])
	
	// Also look for the complete mount wrapper
	// The bundle has: return t.mount=n=>{...},t
	mountIdx := strings.Index(s, "mount=n=>{")
	if mountIdx >= 0 {
		fmt.Printf("\nmount=n=>{ at: %d\n", mountIdx)
		end2 := mountIdx + 600
		if end2 > len(s) { end2 = len(s) }
		fmt.Printf("Mount context:\n%s\n", s[mountIdx:end2])
	}
	
	// And look for the return with comma-t pattern
	commaT := strings.Index(s, "return t.mount=")
	if commaT >= 0 {
		fmt.Printf("\nreturn t.mount= at: %d\n", commaT)
		end3 := commaT + 700
		if end3 > len(s) { end3 = len(s) }
		fmt.Printf("Context:\n%s\n", s[commaT:end3])
	}
}
