// Command edge_mainarea measures the main-area width in Edge (reference) for
// the Vue desktop app, to compare with wb-ui's 98px.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: http.FileServer(http.Dir(distDir))}
	go srv.Serve(ln)
	defer srv.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	// Write a probe JS into a temp file served from dist.
	probe := `
<script>
window.addEventListener('load', function() {
	setTimeout(function() {
		var ma = document.querySelector('.main-area');
		var rp = document.querySelector('.right-panel');
		var welcome = document.querySelector('.welcome-text');
		var welcomeSub = document.querySelector('.welcome-sub');
		var wt = welcome ? welcome.getBoundingClientRect() : null;
		var ws = welcomeSub ? welcomeSub.getBoundingClientRect() : null;
		var out = {
			mainArea: ma ? {x: ma.getBoundingClientRect().x, w: ma.getBoundingClientRect().width, h: ma.getBoundingClientRect().height} : null,
			rightPanel: rp ? {x: rp.getBoundingClientRect().x, w: rp.getBoundingClientRect().width} : null,
			welcomeText: wt ? {x: wt.x, y: wt.y, w: wt.width, h: wt.height} : null,
			welcomeSub: ws ? {x: ws.x, y: ws.y, w: ws.width, h: ws.height} : null
		};
		document.title = 'WBMAINAREA:' + JSON.stringify(out);
	}, 3000);
});
</script>
`
	probePath := filepath.Join(distDir, "probe_mainarea.html")
	html, _ := os.ReadFile(filepath.Join(distDir, "index.html"))
	htmlStr := string(html)
	htmlStr = strings.Replace(htmlStr, "</head>", probe+"</head>", 1)
	os.WriteFile(probePath, []byte(htmlStr), 0o644)
	defer os.Remove(probePath)

	edge := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
	url := fmt.Sprintf("http://127.0.0.1:%d/probe_mainarea.html", port)
	cmd := exec.Command(edge,
		"--headless", "--disable-gpu", "--no-sandbox", "--hide-scrollbars",
		"--force-device-scale-factor=1", "--window-size=1280,800",
		"--virtual-time-budget=8000",
		"--dump-dom", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("edge: %v", err)
	}
	// Find the WBMAINAREA title.
	s := string(out)
	idx := strings.Index(s, "WBMAINAREA:")
	if idx < 0 {
		fmt.Println("no WBMAINAREA in output, first 500 chars:")
		fmt.Println(s[:min(500, len(s))])
		return
	}
	end := strings.Index(s[idx:], "<")
	if end < 0 {
		end = len(s) - idx
	}
	payload := s[idx+len("WBMAINAREA:"): idx+end]
	fmt.Println("EDGE MAIN-AREA:", payload)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &m); err == nil {
		b, _ := json.MarshalIndent(m, "", "  ")
		fmt.Println(string(b))
	}
	time.Sleep(500 * time.Millisecond)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
