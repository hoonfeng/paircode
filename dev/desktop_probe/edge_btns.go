// Command edge_btns renders the companion frontend in Edge headless and
// extracts the real geometry of the chat-input button row (ibb-btns,
// obtn, send-btn, input-bottom-bar, input-wrapper) via getBoundingClientRect.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func edgePath() string {
	for _, p := range []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func main() {
	log.SetFlags(0)
	edge := edgePath()
	if edge == "" {
		log.Fatal("edge not found")
	}
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")

	collector := `
<script>
window.__errs = [];
window.addEventListener('error', e => { window.__errs.push('ERR:' + e.message); });
const sel = ['.input-wrapper','.input-bottom-bar','.ibb-btns','.obtn','.send-btn','.chat-input-area'];
setTimeout(() => {
  const out = sel.map(s => {
    const els = document.querySelectorAll(s);
    const arr = [];
    els.forEach(e => {
      const r = e.getBoundingClientRect();
      const cs = getComputedStyle(e);
      arr.push(s + '|' + Math.round(r.x) + ',' + Math.round(r.y) + ',' + Math.round(r.width) + ',' + Math.round(r.height) + '|gap=' + cs.gap + ' pad=' + cs.padding + '|text=' + (e.textContent||'').trim().slice(0,20));
    });
    return arr.join('\n');
  }).join('\n');
  document.title = 'RESULT<<' + out + '>>RESULT';
}, 3000);
setTimeout(() => { if (document.title.indexOf('RESULT') < 0) { document.title = 'ERRORS<<' + (window.__errs||[]).join('|') + '>>ERRORS'; } }, 6000);
</script>
`
	// Copy dist to a temp dir, inject the collector before </body>, and load
	// that copy in Edge (headless supports a single URL).
	tmpDist := filepath.Join(os.TempDir(), "wbui_edge_btns_dist")
	os.RemoveAll(tmpDist)
	cpDir(distDir, tmpDist)
	idx := filepath.Join(tmpDist, "index.html")
	dat, err := os.ReadFile(idx)
	if err != nil {
		log.Fatal(err)
	}
	html := string(dat)
	html = strings.Replace(html, "</body>", collector+"</body>", 1)
	os.WriteFile(idx, []byte(html), 0644)

	// Serve the injected copy over http (file:// blocks module scripts; the
	// iife build still emits <script type="module">).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	go http.Serve(ln, http.FileServer(http.Dir(tmpDist)))
	fileURL := fmt.Sprintf("http://127.0.0.1:%d/index.html", port)

	profile := filepath.Join(os.TempDir(), "edge_btn_prof")
	args := []string{
		"--headless=new", "--disable-gpu", "--no-sandbox", "--hide-scrollbars",
		"--allow-file-access-from-files",
		"--window-size=1280,800",
		"--virtual-time-budget=10000",
		"--user-data-dir=" + profile,
		"--dump-dom",
		fileURL,
	}
	cmd := exec.Command(edge, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("edge exit: %v\n%s", err, string(out))
	}
	re := regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	mt := re.FindStringSubmatch(string(out))
	fmt.Println("=== TITLE ===")
	if mt != nil {
		fmt.Println("  " + truncate(mt[1], 300))
	} else {
		fmt.Println("  (no title tag) — head:", truncate(string(out), 400))
	}
	re2 := regexp.MustCompile(`RESULT<<(.*?)>>RESULT`)
	m := re2.FindStringSubmatch(string(out))
	if m == nil {
		re3 := regexp.MustCompile(`ERRORS<<(.*?)>>ERRORS`)
		m3 := re3.FindStringSubmatch(string(out))
		if m3 != nil {
			fmt.Println("=== EDGE JS ERRORS ===")
			fmt.Println("  " + m3[1])
			return
		}
		fmt.Println("NO RESULT — edge output tail:", truncate(string(out), 900))
		return
	}
	fmt.Println("=== EDGE CHAT-INPUT BUTTON ROW ===")
	for _, line := range strings.Split(m[1], "\n") {
		if strings.TrimSpace(line) != "" {
			fmt.Println("  " + line)
		}
	}
	_ = runtime.GOOS
	_ = time.Now
}

func cpDir(src, dst string) {
	entries, _ := os.ReadDir(src)
	os.MkdirAll(dst, 0755)
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			cpDir(s, d)
		} else {
			b, err := os.ReadFile(s)
			if err == nil {
				os.WriteFile(d, b, 0644)
			}
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
