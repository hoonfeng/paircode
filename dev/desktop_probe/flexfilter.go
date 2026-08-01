// Command flexfilter greps the flex debug log for ibb-btns / obtn rows.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	f, err := os.Open("dev/desktop_probe/probe_out.txt")
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, "[flex]") {
			continue
		}
		// ibb-btns gap=2, send-btn, input-bottom-bar, obtn items
		if strings.Contains(line, "gap=2.0") || strings.Contains(line, "space-between") ||
			strings.Contains(line, "input-bottom") || strings.Contains(line, "send") ||
			strings.Contains(line, "bw(est)=3") {
			fmt.Println(line)
		}
	}
}
