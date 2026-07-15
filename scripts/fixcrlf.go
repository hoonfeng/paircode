//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"os"
)

func main() {
	path := "build.bat"
	d, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	d = bytes.ReplaceAll(d, []byte{10}, []byte{13, 10})
	if err := os.WriteFile(path, d, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("CRLF OK: %d bytes\n", len(d))
}
