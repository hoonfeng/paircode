package main

// edge_folded_ref: Edge headless 渲染 folded-summary 最小结构（340px 视口），
// 内嵌 script 把「完成摘要」标题布局信息写入 #__result，--dump-dom 输出。
// 用法：go run ./dev/desktop_probe/edge_folded_ref.go
// 输出：tmp/folded_ref_result.txt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	html := `<html><head><style>
		body { margin: 0; font-family: "Microsoft YaHei", sans-serif; }
		.msg-item { display: flex; gap: 8px; align-items: flex-start; }
		.msg-avatar { width: 24px; height: 24px; flex-shrink: 0; background: #ccc; }
		.msg-bubble { flex: 1; min-width: 0; max-width: 85%; font-size: 13px; line-height: 1.6; word-break: break-word; overflow-wrap: break-word; }
		.folded-summary { display: flex; align-items: center; gap: 5px; padding: 5px 10px; background: #fff; border: 1px solid #ccc; font-size: 12px; cursor: pointer; }
		.folded-title { color: #111; font-weight: 500; }
		.folded-desc { color: #888; }
	</style></head><body>
	<div class="msg-item">
		<div class="msg-avatar"></div>
		<div class="msg-bubble">
			<div class="folded-summary">
				<svg class="folded-chevron" viewBox="0 0 8 8" width="9" height="9"><path d="M2.6 1.2 L6.8 4 L2.6 6.8 Z"/></svg>
				<svg class="svg-icon" width="11" height="11" viewBox="0 0 24 24"><path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/></svg>
				<span class="folded-title">完成摘要</span>
				<span class="folded-desc">已为你完成全部请求并生成了完整摘要，共修改 12 个文件，包含样式修复、布局修复、事件派发、渲染树重建等多项改进，所有测试均已通过并完成提交。</span>
			</div>
		</div>
	</div>
	<script>
		window.addEventListener('load', function() {
			var el = document.querySelector('.folded-title');
			var r = el.getBoundingClientRect();
			var fs = getComputedStyle(document.querySelector('.folded-summary'));
			var mb = document.querySelector('.msg-bubble').getBoundingClientRect();
			var fsb = document.querySelector('.folded-summary').getBoundingClientRect();
			var items = [];
			document.querySelector('.folded-summary').querySelectorAll('*').forEach(function(n){
				if (n.textContent && n.textContent.trim().length > 0) {
					var r2 = n.getBoundingClientRect();
					items.push((n.tagName || '') + '.' + (n.className || '') + '[' + n.textContent.trim().slice(0,8) + ']@(' + Math.round(r2.x) + ',' + Math.round(r2.y) + ' ' + Math.round(r2.width) + 'x' + Math.round(r2.height) + ')');
				}
			});
			var out = document.createElement('div');
			out.id = '__result';
			out.textContent = 'viewport=' + innerWidth + 'x' + innerHeight +
				' | bubble=(' + Math.round(mb.x) + ',' + Math.round(mb.width) + ')' +
				' | summary=(' + Math.round(fsb.x) + ',' + Math.round(fsb.width) + ')' +
				' | title=' + el.textContent + '@(' + Math.round(r.x) + ',' + Math.round(r.y) + ' ' + Math.round(r.width) + 'x' + Math.round(r.height) + ')' +
				' | titleLines=' + Math.round(r.height / 16) +
				' | display=' + fs.display + ' gap=' + fs.gap + ' align=' + fs.alignItems +
				' | items: ' + items.join(' ; ');
			document.body.appendChild(out);
		});
	</script>
	</body></html>`

	wd, _ := os.Getwd()
	htmlPath := filepath.Join(wd, "tmp", "folded_ref.html")
	os.WriteFile(htmlPath, []byte(html), 0o644)

	edge := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
	if _, err := os.Stat(edge); err != nil {
		edge = "C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe"
	}
	cmd := exec.Command(edge,
		"--headless=new",
		"--disable-gpu",
		"--window-size=340,600",
		"--virtual-time-budget=3000",
		"--dump-dom",
		"file:///"+filepath.ToSlash(htmlPath),
	)
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "edge failed: %v\n", err)
		os.Exit(1)
	}
	// 提取 #__result 内容
	s := string(out)
	idx := strings.Index(s, "id=\"__result\"")
	result := ""
	if idx >= 0 {
		rest := s[idx:]
		start := strings.Index(rest, ">")
		end := strings.Index(rest, "</div>")
		if start >= 0 && end > start {
			result = rest[start+1 : end]
		}
	}
	outPath := filepath.Join(wd, "tmp", "folded_ref_result.txt")
	os.WriteFile(outPath, []byte(result), 0o644)
	fmt.Println("RESULT:", result)
}
