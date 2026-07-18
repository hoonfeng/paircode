package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	path := "F:/syproject/gou-ide/cmd/desktop/web-ui-minimal/dist/assets/app-CHLaWOgr.js"
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	s := string(data)

	// Fix: Replace je.mount(Mi) with (function(){var m=je.mount;return m.call(je,Mi)})()
	oldEnd := `,je.mount(Mi),window.__S7__="OK"}`
	newEnd := `,(function(){var m=je.mount;return m.call(je,Mi)})(),window.__S7__="OK"}`

	if strings.Contains(s, oldEnd) {
		s = strings.Replace(s, oldEnd, newEnd, 1)
		fmt.Println("Fix applied: je.mount(Mi) → .call() wrapper")
	} else {
		fmt.Println("Fix NOT applied: pattern not found")
		// Try partial match
		if strings.Contains(s, "je.mount(Mi)") {
			fmt.Println("  Partial 'je.mount(Mi)' FOUND in bundle")
		} else {
			fmt.Println("  'je.mount(Mi)' NOT found, checking alternatives...")
			// Search for mount call with different patterns
			patterns := []string{".mount(", "mount(Mi)", "je.mount"}
			for _, p := range patterns {
				if strings.Contains(s, p) {
					idx := strings.Index(s, p)
					start := idx - 20
					if start < 0 { start = 0 }
					end := idx + 40
					if end > len(s) { end = len(s) }
					fmt.Printf("  Found '%s' at %d: ...%s...\n", p, idx, s[start:end])
				}
			}
		}
	}

	// Write backup
	backupPath := path + ".bak2"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		os.WriteFile(backupPath, data, 0644)
		fmt.Println("Backup created:", backupPath)
	}

	if err := os.WriteFile(path, []byte(s), 0644); err != nil {
		panic(err)
	}
	fmt.Println("Written")
}
