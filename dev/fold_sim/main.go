// Command fold_sim 在独立桌面窗口中加载 fold_sim.html——
// 用 Vue 模拟 agent 输出收缩状态（整条折叠/思考段折叠/工具调用展开收起），
// 排查 wb-ui 渲染引擎对收缩状态样式与内容的呈现。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/app"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/webkit"
)

func setupLoaders(wv *webkit.WebView, dir string) {
	absDir, _ := filepath.Abs(dir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDir, clean))
				if err != nil {
					log.Printf("[SCRIPT] FAIL src=%q err=%v", src, err)
				}
				return string(data), err
			}
		}
	}
}

func main() {
	log.SetFlags(log.Ltime)
	convPath := flag.String("conv", "", "历史对话 jsonl 路径（注入 window.__CONV_JSONL__，real_data.js 解析生成 REAL_COMBOS；省略则自动取 .pair/conversations 下最新 jsonl）")
	flag.Parse()
	log.Println("[fold_sim] agent 输出收缩状态模拟窗口")

	wd, _ := os.Getwd()
	dir := filepath.Join(wd, "dev", "fold_sim")

	// 镜像 host.go：初始化 FontManager（默认 Microsoft YaHei），保证 CJK 测量与绘制一致。
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}

	htmlData, err := os.ReadFile(filepath.Join(dir, "fold_sim.html"))
	if err != nil {
		log.Fatalf("[fold_sim] read fold_sim.html: %v", err)
	}

	// -conv 指定历史对话 jsonl：注入 window.__CONV_JSONL__，由 real_data.js 解析生成 REAL_COMBOS
	// 未指定时自动取 .pair/conversations 下最新的非 archived jsonl（保证首屏展示真实会话数据）
	resolvedConv := *convPath
	if resolvedConv == "" {
		convDir := filepath.Join(wd, ".pair", "conversations")
		matches, gerr := filepath.Glob(filepath.Join(convDir, "conv_*.jsonl"))
		if gerr == nil {
			var newest string
			var newestTime time.Time
			for _, m := range matches {
				if strings.HasSuffix(m, ".archived.jsonl") {
					continue
				}
				st, serr := os.Stat(m)
				if serr != nil {
					continue
				}
				if st.ModTime().After(newestTime) {
					newestTime = st.ModTime()
					newest = m
				}
			}
			if newest != "" {
				resolvedConv = newest
				log.Printf("[fold_sim] 自动选择最新会话: %s", filepath.Base(newest))
			}
		}
	}
	if resolvedConv != "" {
		raw, rerr := os.ReadFile(resolvedConv)
		if rerr != nil {
			log.Printf("[fold_sim] 读取 jsonl 失败（回退内置数据）: %v", rerr)
		} else {
			jsLiteral, jerr := json.Marshal(string(raw))
			if jerr != nil {
				log.Printf("[fold_sim] 序列化 jsonl 失败（回退内置数据）: %v", jerr)
			} else {
				inject := "<script>window.__CONV_JSONL__ = " + string(jsLiteral) + ";</script>"
				htmlData = []byte(strings.Replace(string(htmlData), `<script src="./real_data.js"></script>`, inject+"\n"+`<script src="./real_data.js"></script>`, 1))
				log.Printf("[fold_sim] 已注入 jsonl: %s（%d bytes）", resolvedConv, len(raw))
			}
		}
	}
	log.Printf("[fold_sim] HTML: %d bytes", len(htmlData))

	wv := webkit.NewWebView()
	setupLoaders(wv, dir)
	wv.Resize(1280, 800)
	if err := wv.LoadHTML(string(htmlData)); err != nil {
		log.Printf("[fold_sim] LoadHTML err: %v", err)
	}
	// 给 Vue mount + 渲染留时间（探针模式下 LoadHTML 后需等异步脚本执行完）
	time.Sleep(500 * time.Millisecond)
	if out := wv.ConsoleOutput(); out != "" {
		log.Printf("[CONSOLE] %s", out)
	}

	// 创建真实窗口并进入事件循环
	host, err := app.NewHost(wv, 1280, 800, "折叠状态模拟 - agent 输出收缩排查")
	if err != nil {
		log.Fatalf("[fold_sim] 创建窗口失败: %v", err)
	}
	log.Println("[fold_sim] 窗口已启动，事件循环运行中（关闭窗口即退出）")
	host.Run()
	log.Println("[fold_sim] 已退出")

	// 退出前输出诊断信息，便于排查
	if doc := wv.Document(); doc != nil {
		fmt.Fprintf(os.Stderr, "[fold_sim] document=%v\n", doc != nil)
	}
}
