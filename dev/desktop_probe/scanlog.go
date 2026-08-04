package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		ln := sc.Text()
		if !strings.Contains(ln, "[rr]") && !strings.Contains(ln, "[rrx]") && !strings.Contains(ln, "[clip]") {
			continue
		}
		// 提取首个坐标 y
		fields := strings.Fields(ln)
		for _, fld := range fields {
			if idx := strings.Index(fld, ","); idx > 0 {
				yStr := fld[idx+1:]
				y, err := strconv.ParseFloat(yStr, 64)
				if err == nil && y >= 540 && y <= 720 {
					fmt.Printf("%d: %s\n", lineNo, ln)
				}
				break
			}
		}
	}
}
