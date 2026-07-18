package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	path := "F:\\syproject\\wb-ui\\jsc\\interpreter.go"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}
	content := string(data)
	replaced := strings.ReplaceAll(content, "toString(this)", "strVal(this)")
	if content == replaced {
		fmt.Println("No replacements made")
	} else {
		count := strings.Count(content, "toString(this)")
		fmt.Printf("Replaced %d occurrences\n", count)
	}
	err = os.WriteFile(path, []byte(replaced), 0644)
	if err != nil {
		fmt.Println("Error writing:", err)
		return
	}
	fmt.Println("Done")
}
