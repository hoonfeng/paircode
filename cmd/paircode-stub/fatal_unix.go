//go:build stub && !windows

package main

import (
	"fmt"
	"os"
)

// fatal 在类 Unix 平台直接写 stderr（终端启动时可见）。
func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
