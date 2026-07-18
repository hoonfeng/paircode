package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile("cmd/desktop/web-ui/dist/assets/index-1iBlH2r-.js")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	s := string(data)
	// Check for import statements
	if idx := strings.Index(s, "import "); idx >= 0 {
		start := idx
		end := idx + 100
		if end > len(s) { end = len(s) }
		fmt.Printf("FOUND import at byte %d: %s\n", idx, s[start:end])
	} else {
		fmt.Println("NO import statement found (IIFE bundle is self-contained)")
	}
	// Also check for "export "
	if idx := strings.Index(s, "\nexport "); idx >= 0 {
		start := idx
		end := idx + 50
		if end > len(s) { end = len(s) }
		fmt.Printf("FOUND export at byte %d: %s\n", idx, s[start:end])
	}
}
