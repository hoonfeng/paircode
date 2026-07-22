package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile(`cmd\desktop\web-ui\dist\assets\style-HxyH992o.css`)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		ll := strings.ToLower(line)
		if strings.Contains(ll, "sidebar") || strings.Contains(ll, "width") || strings.Contains(ll, "flex-grow") || strings.Contains(ll, "flex:") || strings.Contains(ll, "main-content") || strings.Contains(ll, "statusbar") {
			fmt.Printf("L%04d: %s\n", i+1, line)
		}
	}
}
