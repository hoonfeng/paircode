package main

import (
	"fmt"
	"os"
)

func main() {
	b, err := os.ReadFile("index-1iBlH2r-.js")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	s := string(b)
	if len(s) > 300 {
		fmt.Println("=== FIRST 300 ===")
		fmt.Println(s[:300])
	}
	if len(s) > 400 {
		fmt.Println("=== LAST 400 ===")
		fmt.Println(s[len(s)-400:])
	}
}
