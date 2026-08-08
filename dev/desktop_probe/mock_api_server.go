// Command mock_api_server 为 Edge headless 参照提供真实浏览器数据：
//   - 静态托管 cmd/companion/web-ui（dist + ide_ref_modal.html）
//   - 拦截 /api/philosophy 返回 enabled:true + 7 角色 + 哲学内容
//     （模拟用户已启用思想注入——Edge 参照无需再靠 headless 勾选 checkbox）
// 用法：go run ./dev/desktop_probe/mock_api_server.go （默认 9097）
//   然后 EDGE_REF_TAB=philosophy go run ./dev/desktop_probe/edge_modal_ref.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	wd, _ := os.Getwd()
	webDir := filepath.Join(wd, "cmd", "companion", "web-ui")
	port := os.Getenv("MOCK_API_PORT")
	if port == "" {
		port = "9097"
	}
	port = strings.TrimSpace(port)
	if port == "" {
		port = "9097"
	}

	// /api/* mock
	http.HandleFunc("/api/philosophy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"enabled": true,
			"selected": ["tao-te-ching","sunzi-bingfa"],
			"roles": {
				"main": "以简洁、可维护为最高准则。",
				"planner": "1. 先拆解目标为可执行步骤。\n2. 每个步骤明确输入输出。\n3. 评估风险并规划回退。",
				"reviewer": "1. 审查代码的正确性与风格。\n2. 指出潜在边界条件问题。",
				"judge": "评测要客观，给出量化结论。",
				"explorer": "探索时优先理解整体结构。",
				"verifier": "验证要覆盖正常与异常路径。",
				"debugger": "调试要定位根因而非打补丁。",
				"executor": "执行要稳，先编译再运行。"
			},
			"availableClassics": [
				{"id":"tao-te-ching","name":"《道德经》"},{"id":"huangdi-yinfu-jing","name":"《黄帝阴符经》"},
				{"id":"sunzi-bingfa","name":"《孙子兵法》"},{"id":"lunyu","name":"《论语》"},
				{"id":"yijing","name":"《易经》"},{"id":"zhongyong","name":"《中庸》"},{"id":"daxue","name":"《大学》"}
			],
			"availableRoles": [
				{"id":"planner","name":"规划 Agent"},{"id":"reviewer","name":"审核 Agent"},
				{"id":"judge","name":"评测 Agent"},{"id":"explorer","name":"探索 Agent"},
				{"id":"verifier","name":"验证 Agent"},{"id":"debugger","name":"调试 Agent"},
				{"id":"executor","name":"执行 Agent"}
			]
		}`)
	})
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})

	// 静态文件
	fs := http.FileServer(http.Dir(webDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/" {
			p = "/index.html"
		}
		if strings.HasPrefix(p, "/api/") {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})

	fmt.Printf("mock api server: http://localhost:%s (web-ui: %s)\n", port, webDir)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
