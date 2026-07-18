package main
import ("fmt";"os";"strings")
func main() {
	d, e := os.ReadFile("cmd/desktop/web-ui/dist/assets/index-1iBlH2r-.js")
	if e != nil { fmt.Println("Err:", e); return }
	s := string(d)
	idx := strings.Index(s, "await")
	if idx < 0 { fmt.Println("await not found"); return }
	start := idx - 20
	if start < 0 { start = 0 }
	end := idx + 60
	if end > len(s) { end = len(s) }
	fmt.Printf("First 'await' at byte %d:\n%s\n", idx, s[start:end])
	// Show all await usages (first 10)
	count := 0
	pos := 0
	for count < 10 {
		p := strings.Index(s[pos:], "await")
		if p < 0 { break }
		absPos := pos + p
		ctxStart := absPos - 10
		if ctxStart < 0 { ctxStart = 0 }
		ctxEnd := absPos + 30
		if ctxEnd > len(s) { ctxEnd = len(s) }
		fmt.Printf("  %d: ...%s...\n", absPos, s[ctxStart:ctxEnd])
		pos = absPos + 5
		count++
	}
}
