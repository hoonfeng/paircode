/* real_data.js — 折叠状态模拟的真实会话数据
 *
 * 数据来源（优先级）：
 *   1. window.__CONV_JSONL__：宿主（fold_sim.exe -conv <jsonl>）注入的
 *      历史对话 jsonl 全文 → 由内置解析器 parseConversationJsonl 直接解析生成；
 *   2. FALLBACK_COMBOS：内置手工数据（无注入时回退，保持页面可独立演示）。
 *
 * 解析器 parseConversationJsonl(text) 与 conv_real.py 逻辑等价：
 *   - user 消息        → {user:{content}}
 *   - assistant 消息   → {assistant:{_folded,_summary,segments:[thinking/tool_call/content]}}
 *   - tool 消息的结果  → 并入上一条 tool_call 段的 result
 *   - 前 FOLD_FIRST 个 assistant 回合标记 _folded=true（演示整条折叠）
 *   - 最多保留 MAX_TURNS 个 assistant 回合
 */
var FALLBACK_COMBOS = [
 {
  "assistant": {
   "_folded": true,
   "_summary": "",
   "segments": [
    {
     "type": "tool_call",
     "name": "search_content",
     "argsRaw": "{\"glob\": \"_comp_probe_out.txt\", \"max_results\": 10, \"path\": \"F:\\\\syproject\\\\gou-ide\\\\dev\\\\desktop_probe\", \"pattern\": \"PROBE_RESULT|RECTS|saved\"}",
     "result": "dev/desktop_probe/_comp_probe_out.txt:7382: saved F:\\syproject\\gou-ide\\dev\\desktop_probe\\_comp_render.png\ndev/desktop_probe/_comp_probe_out.txt:7383: PROBE_RESULT: {{\".comp-bar\":{\"x\":0,\"y\":0,\"w\":0,\"h\":0,\"cls\":\"comp-bar\",\"bg\":\"\",\"children\":6},\".ctx-bar\":{\"x\":0,\"y\":0,\"w\":0,\"h\":0,\"cls\":\"ctx-bar\",\"bg\":\"\",\"children\":1}} <nil> 0x2c73b1fbe0c0}\ndev/desktop_probe/_comp_probe_out.txt:7384: RECTS: {{\"c\":{\"x\":0,\"y\":0,\"w\":0,\"h\":0},\"x\":{\"x\":0,\"y\":0,\"w\":0,\"h\":0}} <nil> 0x2c73b1fbe0c0}\n",
     "_expanded": false
    }
   ]
  }
 },
 {
  "assistant": {
   "_folded": true,
   "_summary": "comp-bar rect 全 0——它当前不可见。查 ConvSidebar.vue 中 comp-bar 的渲染条件",
   "segments": [
    {
     "type": "thinking",
     "content": "comp-bar rect 全 0（x=0,y=0,w=0,h=0）——说明 comp-bar 在**渲染视口外**（可能被 CSS 隐藏或 display:none，或未布局）。这解释了为什么在像素图里找不到它。\n\ncomp-bar 是会话侧边栏底部的上下文构成条，可能当前对话没有内容时隐藏，或者在另一个面板。ctx-bar children=1。\n\n回顾之前会话的验证方法：之前验证 comp-bar 圆角时用了注入 + WB_DIAG_INJECT 等。可能 comp-bar 需要特定状态才显示。\n\n从 git 状态看，之前会话的验证（454bfa0 提交 \"验证 comp-bar/ctx-bar 圆角在 CPU/GPU 双路径均正常\"）是怎么做的？看 gou-ide 的 454bfa0 提交内容和 wb-ui 的 ae74de4 提交内容，了解验证工具链。\n\n其实更直接：**把 comp-bar 手动放到可见位置验证圆角**。或者用之前会话的 clip_repro / gpu_nested_repro / comp_render_probe 的验证法。\n\n回顾之前对话上下文（记忆「浮窗遮罩GPU不显示」等），以及 _comp_render.png 的实际内容——刚才缩略图分析显示主色 #19192A 32.6% + #2A2A2A 22.4% —— 这不是真实前端渲染应有的配色（#0d1117 背景）？等等，缩略图有缩放插值，颜色可能失真。\n\n我担心的是 comp_bar_probe 里注入的 seg 用 inline style + class='comp-bar-seg'，但这些 seg 可能没有对应 CSS（comp-bar 的 seg 是渐变背景）。之前会话验证时 comp-bar 应该可见。\n\n关键问题：**comp-bar 为什么 rect=0？** 可能：\n1. comp-bar 在 ConvSidebar 里，而当前布局下 ConvSidebar 是横向模式（.conv-sidebar-horizontal .comp-bar 有额外样式）\n2. comp-bar 只在特定条件下渲染\n\n看看 ConvSidebar.vue 中 comp-bar 的使用逻辑。",
     "_collapsed": true
    },
    {
     "type": "content",
     "content": "comp-bar rect 全 0——它当前不可见。查 ConvSidebar.vue 中 comp-bar 的渲染条件："
    },
    {
     "type": "tool_call",
     "name": "search_content",
     "argsRaw": "{\"glob\": \"*.vue\", \"max_results\": 15, \"path\": \"F:\\\\syproject\\\\gou-ide\\\\cmd\\\\companion\\\\web-ui\", \"pattern\": \"comp-bar\"}",
     "result": "cmd/companion/web-ui/src/components/ConvSidebar.vue:103: <div class=\"comp-bar-wrap\">\ncmd/companion/web-ui/src/components/ConvSidebar.vue:104: <div class=\"comp-bar-title\">上下文构成</div>\ncmd/companion/web-ui/src/components/ConvSidebar.vue:105: <div class=\"comp-bar\">\ncmd/companion/web-ui/src/components/ConvSidebar.vue:106: <div v-if=\"convCtxStats.systemTokens > 0\" class=\"comp-bar-seg comp-system\" :style=\"{ width: compSystemPct + '%' }\"\ncmd/companion/web-ui/src/components/ConvSidebar.vue:108: <div v-if=\"convCtxStats.skillsTokens > 0\" class=\"comp-bar-seg comp-skills\" :style=\"{ width: compSkillsPct + '%' }\"\ncmd/companion/web-ui/src/components/ConvSidebar.vue:110: <div v-if=\"convCtxStats.mcpTokens > 0\" class=\"comp-bar-seg comp-mcp\" :style=\"{ width: compMCPPct + '%' }\"\ncmd/companion/web-ui/src/components/ConvSidebar.vue:112: <div v-if=\"convCtxStats.toolTokens > 0\" class=\"comp-bar-seg comp-tool\" :style=\"{ width: compToolPct + '%' }\"\ncmd/companion/web-ui/src/components/ConvSidebar.vue:114: <div v-if=\"convCtxStats.historyTokens > 0\" class=\"comp-bar-seg comp-history\" :style=\"{ width: compHistoryPct + '%' }\"\ncmd/companion/web-ui/src/components/ConvSidebar.vue:116: <div v-if=\"convCtxStats.otherTokens > 0\" class=\"comp-bar-seg comp-other\" :style=\"{ width: compOtherPct + '%' }\"\ncmd/companion/web-ui/src/components/ConvSidebar.vue:527: .comp-bar-wrap { margin-top: 8px; padding-top: 6px; border-top: 1px solid var(--border-color); }\ncmd/companion/web-ui/src/components/ConvSidebar.vue:528: .comp-bar-title { font-size: 10px; color: var(--text-muted); margin-bottom: 4px; text-transform: uppercase; letter-spacing: 0.3px; }\ncmd/companion/web-ui/src/components/ConvSidebar.vue:529: .comp-bar { display: flex; height: 12px; background: var(--bg-primary); border-radius: 6px; overflow: hidden; border: 1px solid var(--border-color); }\ncmd/companion/web-ui/src/components/ConvSidebar.vue:530: .comp-bar-seg { height: 100%; transition: width 0.3s ease; min-width: 2px; }\ncmd/companion/web-ui/src/components/ConvSidebar.vue:821: .conv-sidebar-horizontal .comp-bar {\ncmd/companion/web-ui/src/components/ConvSidebar.vue:824: .conv-sidebar-horizontal .comp-bar-title {\n[已达上限 15 条，可能还有更多匹配——请缩小 pattern 或 path]\n",
     "_expanded": false
    }
   ]
  }
 },
 {
  "assistant": {
   "_folded": false,
   "_summary": "comp-bar 依赖统计 token 条件才显示。改用专门的复现工具验证 ClipRoundRect 修复。看 cli",
   "segments": [
    {
     "type": "thinking",
     "content": "comp-bar 依赖 `convCtxStats.systemTokens > 0` 等条件。之前会话注入的 6 个 seg 是在 DOM 里手动创建的（comp_bar_probe 用 innerHTML 替换）——但 rect 还是 0，说明 comp-bar 容器本身不可见（可能侧边栏折叠或视口位置不对）。\n\n其实从 comp_bar_probe 输出：`.comp-bar` 的 rect 全 0，但 children=6（注入成功）。说明容器 display 或布局尺寸为 0。可能：\n1. conv-sidebar 处于 horizontal 模式？\n2. comp-bar-wrap 在某 collapsed 容器里\n\n不过——**重点不是通过真实 UI 找 comp-bar**。重点是验证 ClipRoundRect 的 RRect 标准实现是否解决了圆角泄漏。之前会话已经建立了一个更直接的验证路径：clip_repro.go / gpu_nested_repro.go（专门复现 ClipRoundRect 泄漏的独立程序）。\n\n看 clip_repro.go 和 gpu_comp_repro.go 是什么——用它们验证。",
     "_collapsed": true
    },
    {
     "type": "content",
     "content": "comp-bar 依赖统计 token 条件才显示。改用专门的复现工具验证 ClipRoundRect 修复。看 clip_repro.go："
    },
    {
     "type": "tool_call",
     "name": "read_file",
     "argsRaw": "{\"limit\": 80, \"offset\": 1, \"path\": \"F:\\\\syproject\\\\gou-ide\\\\dev\\\\desktop_probe\\\\clip_repro.go\"}",
     "result": "// Command clip_repro 最小复现：真实渲染参数下的 ClipRoundRect 行为\npackage main\n\nimport (\n\t\"fmt\"\n\t\"image\"\n\t\"image/png\"\n\t\"os\"\n\t\"wb-ui/platform/graphics\"\n)\n\nfunc dumpPixel(img *image.RGBA, w, h int, x, y int, label string) {\n\toff := (y*w + x) * 4\n\tfmt.Printf(\"[pix] %s (%d,%d) = #%02X%02X%02X\\n\", label, x, y,\n\t\timg.Pix[off], img.Pix[off+1], img.Pix[off+2])\n}\n\nfunc main() {\n\t// 场景 A：与单元测试相同（无 scale、小坐标）\n\tcA := graphics.NewCanvas(80, 80)\n\tcA.ClipRoundRect(10, 10, 40, 40, 6)\n\tcA.FillRect(0, 0, 80, 80, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgA := toRGBA(cA.Pixels(), 80, 80)\n\tdumpPixel(imgA, 80, 80, 10, 10, \"A:clip左上角(10,10) 期望背景(透明/黑)\")\n\tdumpPixel(imgA, 80, 80, 12, 10, \"A:clip顶部圆角起点附近(12,10) 期望背景\")\n\tdumpPixel(imgA, 80, 80, 16, 10, \"A:clip顶部(16,10) 期望绿\")\n\tdumpPixel(imgA, 80, 80, 11, 11, \"A:clip圆角内侧(11,11) 期望过渡或绿\")\n\n\t// 场景 B：真实参数（2560x1600 canvas + Scale(2,2) + 大坐标）\n\tcB := graphics.NewCanvas(2560, 1600)\n\tcB.Scale(2, 2)\n\tcB.ClipRoundRect(1039, 686.8, 233, 12, 6)\n\tcB.FillRect(0, 0, 1280, 800, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgB := toRGBA(cB.Pixels(), 2560, 1600)\n\t// 设备坐标 = 逻辑坐标 * 2。comp-bar 左上角逻辑 (1039,687) → 设备 (2078,1374)\n\tdumpPixel(imgB, 2560, 1600, 2078, 1374, \"B:comp左上角(2078,1374) 期望背景\")\n\tdumpPixel(imgB, 2560, 1600, 2082, 1374, \"B:顶部圆角起点(2082,1374) 期望背景\")\n\tdumpPixel(imgB, 2560, 1600, 2090, 1374, \"B:顶部圆角内(2090,1374) 期望绿\")\n\tdumpPixel(imgB, 2560, 1600, 2078, 1380, \"B:左缘中部(2078,1380) 期望绿\")\n\tdumpPixel(imgB, 2560, 1600, 2080, 1376, \"B:圆角弧上(2080,1376) 期望过渡/绿\")\n\n\t// 场景 C：无 Scale 大坐标（排除 Scale 因素）\n\tcC := graphics.NewCanvas(1280, 800)\n\tcC.ClipRoundRect(1039, 686.8, 233, 12, 6)\n\tcC.FillRect(0, 0, 1280, 800, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgC := toRGBA(cC.Pixels(), 1280, 800)\n\tdumpPixel(imgC, 1280, 800, 1039, 687, \"C:comp左上角(1039,687) 期望背景\")\n\tdumpPixel(imgC, 1280, 800, 1045, 687, \"C:顶部圆角起点(1045,687) 期望背景\")\n\tdumpPixel(imgC, 1280, 800, 1050, 687, \"C:顶部圆角内(1050,687) 期望绿\")\n\tdumpPixel(imgC, 1280, 800, 1039, 693, \"C:左缘圆心高度(1039,693) 期望绿\")\n\t// 场景 D：FillRoundRect 直接画圆角矩形（真实 comp-bar 参数）\n\tcD := graphics.NewCanvas(2560, 1600)\n\tcD.Scale(2, 2)\n\tcD.FillRoundRect(1039, 696, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})\n\timgD := toRGBA(cD.Pixels(), 2560, 1600)\n\tfmt.Println(\"\\n[场景D] FillRoundRect(1039,696,233,12,r=6) 期望圆角: 顶部行 y=696 仅 x=1045 附近红色\")\n\t// 顶部行（设备 y=1392）\n\tfor _, px := range []int{2078, 2082, 2086, 2090, 2094} {\n\t\tdumpPixel(imgD, 2560, 1600, px, 1392, fmt.Sprintf(\"D:顶行y696 x%d\", px/2))\n\t}\n\tfor _, px := range []int{2078, 2082, 2086, 2090} {\n\t\tdumpPixel(imgD, 2560, 1600, px, 1394, fmt.Sprintf(\"D:y697 x%d\", px/2))\n\t}\n\tdumpPixel(imgD, 2560, 1600, 2078, 1400, \"D:y700 x1039(左缘中)\")\n\tdumpPixel(imgD, 2560, 1600, 2082, 1400, \"D:y700 x1041\")\n\tdumpPixel(imgD, 2560, 1600, 2540, 1392, \"D:y696 x1270(右上)\")\n\n\t// 场景 E：StrokeRoundRect（border）\n\tcE := graphics.NewCanvas(2560, 1600)\n\tcE.Scale(2, 2)\n\tcE.StrokeRoundRect(1039, 696, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgE := toRGBA(cE.Pixels(), 2560, 1600)\n\tfmt.Println(\"\\n[场景E] StrokeRoundRect(1039,696,233,12,r=6,w=1) 期望圆角边框\")\n\tfor _, px := range []int{2078, 2082, 2086, 2090, 2094} {\n\t\tdumpPixel(imgE, 2560, 1600, px, 1392, fmt.Sprintf(\"E:顶行y696 x%d\", px/2))\n\t}\n\tdumpPixel(imgE, 2560, 1600, 2078, 1400, \"E:y700 x1039(左缘中)\")\n\t// 场景 F：1x（无 Scale）FillRoundRect + StrokeRoundRect\n\tcF := graphics.NewCanvas(1280, 800)\n\tcF.FillRoundRect(1039, 696, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})",
     "_expanded": false
    }
   ]
  }
 },
 {
  "assistant": {
   "_folded": false,
   "_summary": "运行 CPU 复现工具验证 RRect 修复：",
   "segments": [
    {
     "type": "thinking",
     "content": "clip_repro.go 是专门的 CPU 复现工具。用 RRect 实现后重新运行它，看圆角是否正常。",
     "_collapsed": true
    },
    {
     "type": "content",
     "content": "运行 CPU 复现工具验证 RRect 修复："
    },
    {
     "type": "tool_call",
     "name": "run_command",
     "argsRaw": "{\"command\": \"cd F:\\\\syproject\\\\gou-ide && set CGO_ENABLED=1 && go run ./dev/desktop_probe/clip_repro.go 2>&1\"}",
     "result": "[pix] A:clip左上角(10,10) 期望背景(透明/黑) (10,10) = #000000\n[pix] A:clip顶部圆角起点附近(12,10) 期望背景 (12,10) = #000000\n[pix] A:clip顶部(16,10) 期望绿 (16,10) = #00FF00\n[pix] A:clip圆角内侧(11,11) 期望过渡或绿 (11,11) = #001000\n[pix] B:comp左上角(2078,1374) 期望背景 (2078,1374) = #000000\n[pix] B:顶部圆角起点(2082,1374) 期望背景 (2082,1374) = #000000\n[pix] B:顶部圆角内(2090,1374) 期望绿 (2090,1374) = #00FF00\n[pix] B:左缘中部(2078,1380) 期望绿 (2078,1380) = #000800\n[pix] B:圆角弧上(2080,1376) 期望过渡/绿 (2080,1376) = #000000\n[pix] C:comp左上角(1039,687) 期望背景 (1039,687) = #000000\n[pix] C:顶部圆角起点(1045,687) 期望背景 (1045,687) = #00FF00\n[pix] C:顶部圆角内(1050,687) 期望绿 (1050,687) = #00FF00\n[pix] C:左缘圆心高度(1039,693) 期望绿 (1039,693) = #00D800\n\n[场景D] FillRoundRect(1039,696,233,12,r=6) 期望圆角: 顶部行 y=696 仅 x=1045 附近红色\n[pix] D:顶行y696 x1039 (2078,1392) = #000000\n[pix] D:顶行y696 x1041 (2082,1392) = #000000\n[pix] D:顶行y696 x1043 (2086,1392) = #5E0000\n[pix] D:顶行y696 x1045 (2090,1392) = #FF0000\n[pix] D:顶行y696 x1047 (2094,1392) = #FF0000\n[pix] D:y697 x1039 (2078,1394) = #000000\n[pix] D:y697 x1041 (2082,1394) = #4C0000\n[pix] D:y697 x1043 (2086,1394) = #FF0000\n[pix] D:y697 x1045 (2090,1394) = #FF0000\n[pix] D:y700 x1039(左缘中) (2078,1400) = #680000\n[pix] D:y700 x1041 (2082,1400) = #FF0000\n[pix] D:y696 x1270(右上) (2540,1392) = #000000\n\n[场景E] StrokeRoundRect(1039,696,233,12,r=6,w=1) 期望圆角边框\n[pix] E:顶行y696 x1039 (2078,1392) = #000000\n[pix] E:顶行y696 x1041 (2082,1392) = #000000\n[pix] E:顶行y696 x1043 (2086,1392) = #006400\n[pix] E:顶行y696 x1045 (2090,1392) = #00FF00\n[pix] E:顶行y696 x1047 (2094,1392) = #00FF00\n[pix] E:y700 x1039(左缘中) (2078,1400) = #006F00\n\n[场景F] 1x FillRoundRect+StrokeRoundRect r=6\n[pix] F:顶行y696 x1039 (1039,696) = #000000\n[pix] F:顶行y696 x1041 (1041,696) = #000000\n[pix] F:顶行y696 x1043 (1043,696) = #30B900\n[pix] F:顶行y696 x1045 (1045,696) = #00FF00\n[pix] F:顶行y696 x1047 (1047,696) = #00FF00\n[pix] F:顶行y696 x1049 (1049,696) = #00FF00\n[pix] F:y697 x1039 (1039,697) = #000000\n[pix] F:y697 x1041 (1041,697) = #4C9600\n[pix] F:y697 x1043 (1043,697) = #B94600\n[pix] F:y697 x1045 (1045,697) = #FF0000\n[pix] F:y700 x1039 (1039,700) = #33B900\n[pix] F:y696 x1271(右上) (1271,696) = #000000\n[pix] F:y700 x1271(右缘) (1271,700) = #35B600\n\n[场景G] 矩形clip ∩ 圆角clip 相交后 FillRect+StrokeRoundRect\n[pix] G:顶行y696 x1039 (1039,696) = #000000\n[pix] G:顶行y696 x1041 (1041,696) = #140000\n[pix] G:顶行y696 x1043 (1043,696) = #21DC00\n[pix] G:顶行y696 x1045 (1045,696) = #33CC00\n[pix] G:顶行y696 x1047 (1047,696) = #33CC00\n[pix] G:y697 x1039 (1039,697) = #000000\n[pix] G:y697 x1041 (1041,697) = #3FBA00\n[pix] G:y697 x1043 (1043,697) = #ED1200\n[pix] G:y697 x1045 (1045,697) = #FF0000\n[pix] G:y700 x1045 (1045,700) = #FF0000\n[pix] G:y700 x1271(右缘) (1271,700) = #529700\n\n[场景H] 2x CTM: ClipRoundRect+FillRoundRect+StrokeRoundRect r=6\n[pix] H:顶行y1392 x2078 (2078,1392) = #000000\n[pix] H:顶行y1392 x2082 (2082,1392) = #000000\n[pix] H:顶行y1392 x2086 (2086,1392) = #2ACA00\n[pix] H:顶行y1392 x2090 (2090,1392) = #00FF00\n[pix] H:顶行y1392 x2094 (2094,1392) = #00FF00\n[pix] H:y1394 x2082 (2082,1394) = #3C9C00\n[pix] H:y1394 x2086 (2086,1394) = #CE3100\n[pix] H:y1394 x2090 (2090,1394) = #FF0000\n[pix] H:y1400 x2542(右缘) (2542,1400) = #00FF00\n[pix] H:y1400 x2090 (2090,1400) = #FF0000\n\n[场景I] 300次历史绘制后 comp-bar 圆角\n[pix] I:顶行y1392 x2078 (2078,1392) = #000000\n[pix] I:顶行y1392 x2082 (2082,1392) = #000000\n[pix] I:顶行y1392 x2086 (2086,1392) = #000000\n[pix] I:顶行y1392 x2090 (2090,1392) = #000000\n[pix] I:顶行y1392 x2094 (2094,1392) = #000000\n[pix] I:y1394 x2090 (2090,1394) = #000000\n[pix] I:y1400 x2542(右缘) (2542,1400) = #000000\n[pix] I:y1400 x2090 (2090,1400) = #000000\n\n[场景J] 两个渐变后 FillRoundRect 圆角\n[pix] J:顶行y1392 x2078 (2078,1392) = #000000\n[pix] J:顶行y1392 x2082 (2082,1392) = #000000\n[pix] J:顶行y1392 x2086 (2086,1392) = #C80000\n[pix] J:顶行y1392 x2090 (2090,1392) = #FF0000\n[pix] J:顶行y1392 x2094 (2094,1392) = #FF0000\n[pix] J:y1400 x2542(右缘) (2542,1400) = #FF0000\n[pix] J:y1400 x2090 (2090,1400) = #FF0000\n\n[场景K] FillRoundRect 后渐变再 FillRoundRect\n[pix] K:顶行y1392 x2078 (2078,1392) = #000000\n[pix] K:顶行y1392 x2082 (2082,1392) = #000000\n[pix] K:顶行",
     "_expanded": false
    }
   ]
  }
 },
 {
  "assistant": {
   "_folded": false,
   "_summary": "CPU 场景基本正常（圆角弧 AA 正确）。但场景 I（300 次历史绘制后）全黑，需确认其定义：",
   "segments": [
    {
     "type": "thinking",
     "content": "分析 CPU 结果：\n\n**场景 A**：clip 圆角正常：(10,10)、(12,10) 背景色，(16,10) 绿，(11,11) 过渡。✔\n**场景 B**：comp-bar 参数 Scale(2,2)：(2078,1374) 背景、(2082,1374) 背景、(2090,1374) 绿、(2078,1380) 背景(期望绿——不对！左缘中部应该是绿的)。\n\n等等，B 场景 (2078,1380) 期望绿，结果是 #000800（接近黑）。但这是 ClipRoundRect(1039, 686.8, 233, 12, 6) —— 圆角矩形高 12px，r=6，是药丸形。左缘 x=1039，在 y=686.8+6=692.8 处才是圆心高度（最宽）。y=1380 设备 = 690 逻辑，距离顶部 686.8 只有 3.2px —— 在圆角弧区域内（r=6），x=1039 处尚未进入矩形（圆弧外）。所以 #000800 正确（弧外）。✔\n\n而 (2078,1380) 期望\"绿\"的注释是旧的（clip_repro 注释基于旧的几何认知）。实际圆弧几何：y=690 距顶 3.2px，弧宽约 r-sqrt(r²-3.2²)=6-sqrt(36-10.24)=6-5.07=0.93px，所以 x=1039 处弧外，黑。合理。\n\n**场景 B (2080,1376)**：y=688 距顶 1.2px，弧宽 6-sqrt(36-1.44)=6-5.88=0.12px，x=1040 弧外 → 黑。✔（注释期望过渡/绿，实际黑，但几何上正确）\n\n**场景 C**（1x 无 scale，x=1039）：(1039,687) 黑 ✔（距顶 0.2px，弧外）；(1045,687) 绿？—— 距顶 0.2px，弧宽 0.05px，x=1045 距左缘 6px → 在弧内？弧外到 x=1039+0.05=1039.05，所以 x=1045 完全在矩形内 → 绿 ✔（x=1045 已过圆弧区）。✔\n\n**场景 D** FillRoundRect 圆角矩形填充：顶行 y=696 在 x1043 处 #5E0000（过渡）、x1045 全红、x1039/1041 黑。药丸形：顶行（y=696 距顶 0px）应该全黑（圆弧顶点在 y=696+... 等等，r=6，顶边圆弧从 y=696 到 y=696+6。y=696 是第一行，应该在弧上。x=1039 是左缘，弧的起点。x=1039 黑、1041 黑、1043 过渡、1045 红 —— 这是圆角矩形正确的 AA 过渡（r=6 圆弧从 (1039+6, 696) 开始 → x=1045 是弧的切点，之后是直边）。✔ \n\n嗯等等，圆弧切点在 x=1039+6=1045。所以 y=696（弧顶点行）上，x<1045 是弧外（黑），x=1045 开始是弧上/内。x=1043 #5E0000 过渡。x=1045 红。完全正确！\n\n**场景 E/F** 同构正确。\n\n**场景 I** 全黑——300 次历史绘制后 comp-bar 圆角变成全黑？这可能是场景 I 的特殊性（300 次 Save/Restore 后状态损坏？）。但 I 场景的注释说明它是\"300次历史绘制后\"——需要看 clip_repro.go 场景 I 的实现。不过 I 全黑可能意味着某种状态污染。让我看看 clip_repro.go 的场景 I 代码。\n\n其实等等——场景 I 可能特意测试「历史绘制后圆角 clip 是否失效」，如果全黑可能是**修复前**的 bug 复现点。但现在 RRect 实现下 I 仍全黑？\n\n看场景 I 代码。另外 GPU 路径验证：gpu_comp_repro.go。\n\n先看 clip_repro.go 剩余部分（场景 I 的定义）。",
     "_collapsed": true
    },
    {
     "type": "content",
     "content": "CPU 场景基本正常（圆角弧 AA 正确）。但场景 I（300 次历史绘制后）全黑，需确认其定义："
    },
    {
     "type": "tool_call",
     "name": "read_file",
     "argsRaw": "{\"limit\": 60, \"offset\": 80, \"path\": \"F:\\\\syproject\\\\gou-ide\\\\dev\\\\desktop_probe\\\\clip_repro.go\"}",
     "result": "\tcF.FillRoundRect(1039, 696, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})\n\tcF.StrokeRoundRect(1039, 696, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgF := toRGBA(cF.Pixels(), 1280, 800)\n\tfmt.Println(\"\\n[场景F] 1x FillRoundRect+StrokeRoundRect r=6\")\n\tfor _, px := range []int{1039, 1041, 1043, 1045, 1047, 1049} {\n\t\tdumpPixel(imgF, 1280, 800, px, 696, fmt.Sprintf(\"F:顶行y696 x%d\", px))\n\t}\n\tfor _, px := range []int{1039, 1041, 1043, 1045} {\n\t\tdumpPixel(imgF, 1280, 800, px, 697, fmt.Sprintf(\"F:y697 x%d\", px))\n\t}\n\tdumpPixel(imgF, 1280, 800, 1039, 700, \"F:y700 x1039\")\n\tdumpPixel(imgF, 1280, 800, 1271, 696, \"F:y696 x1271(右上)\")\n\tdumpPixel(imgF, 1280, 800, 1271, 700, \"F:y700 x1271(右缘)\")\n\t// 场景 G：Clip(矩形祖先) + ClipRoundRect 相交（真实渲染的核心差异）\n\tcG := graphics.NewCanvas(1280, 800)\n\tcG.Clip(graphics.Rect{X: 48, Y: 30, Width: 1232, Height: 770}) // sidebar 区域矩形\n\tcG.ClipRoundRect(1039, 695.8, 233, 12, 6)\n\tcG.FillRect(1039, 695.8, 233, 12, graphics.Color{R: 255, G: 0, B: 0, A: 255})\n\tcG.StrokeRoundRect(1039, 695.8, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgG := toRGBA(cG.Pixels(), 1280, 800)\n\tfmt.Println(\"\\n[场景G] 矩形clip ∩ 圆角clip 相交后 FillRect+StrokeRoundRect\")\n\tfor _, px := range []int{1039, 1041, 1043, 1045, 1047} {\n\t\tdumpPixel(imgG, 1280, 800, px, 696, fmt.Sprintf(\"G:顶行y696 x%d\", px))\n\t}\n\tfor _, px := range []int{1039, 1041, 1043, 1045} {\n\t\tdumpPixel(imgG, 1280, 800, px, 697, fmt.Sprintf(\"G:y697 x%d\", px))\n\t}\n\tdumpPixel(imgG, 1280, 800, 1045, 700, \"G:y700 x1045\")\n\tdumpPixel(imgG, 1280, 800, 1271, 700, \"G:y700 x1271(右缘)\")\n\n\t// 场景 H：2x CTM 下 FillRoundRect + StrokeRoundRect + ClipRoundRect\n\tcH := graphics.NewCanvas(2560, 1600)\n\tcH.Scale(2, 2)\n\tcH.ClipRoundRect(1039, 695.8, 233, 12, 6)\n\tcH.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})\n\tcH.StrokeRoundRect(1039, 695.8, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgH := toRGBA(cH.Pixels(), 2560, 1600)\n\tfmt.Println(\"\\n[场景H] 2x CTM: ClipRoundRect+FillRoundRect+StrokeRoundRect r=6\")\n\tfor _, px := range []int{2078, 2082, 2086, 2090, 2094} {\n\t\tdumpPixel(imgH, 2560, 1600, px, 1392, fmt.Sprintf(\"H:顶行y1392 x%d\", px))\n\t}\n\tfor _, px := range []int{2082, 2086, 2090} {\n\t\tdumpPixel(imgH, 2560, 1600, px, 1394, fmt.Sprintf(\"H:y1394 x%d\", px))\n\t}\n\tdumpPixel(imgH, 2560, 1600, 2542, 1400, \"H:y1400 x2542(右缘)\")\n\tdumpPixel(imgH, 2560, 1600, 2090, 1400, \"H:y1400 x2090\")\n\n\t// 场景 I：先大量绘制（文本/渐变/圆角）模拟真实渲染历史，再画 comp-bar\n\tcI := graphics.NewCanvas(2560, 1600)\n\tcI.Scale(2, 2)\n\t// 模拟真实渲染的几百次操作\n\tfor i := 0; i < 300; i++ {\n\t\txx := float64((i * 7) % 1200)\n\t\tyy := float64((i * 13) % 700)\n\t\tcI.FillRect(xx, yy, 30, 14, graphics.Color{R: uint8(i % 255), G: 40, B: 80, A: 255})\n\t\tif i%3 == 0 {\n\t\t\tcI.FillLinearGradient(xx+100, yy, 40, 10, graphics.Color{R: 10, G: 20, B: 30, A: 255}, graphics.Color{R: 90, G: 100, B: 110, A: 255})\n\t\t}\n\t\tif i%5 == 0 {\n\t\t\tcI.Save()",
     "_expanded": false
    }
   ]
  }
 },
 {
  "assistant": {
   "_folded": false,
   "_summary": "",
   "segments": [
    {
     "type": "tool_call",
     "name": "read_file",
     "argsRaw": "{\"limit\": 40, \"offset\": 140, \"path\": \"F:\\\\syproject\\\\gou-ide\\\\dev\\\\desktop_probe\\\\clip_repro.go\"}",
     "result": "\t\t\tcI.ClipRoundRect(xx, yy, 40, 20, 6)\n\t\t\tcI.FillRect(xx, yy, 40, 20, graphics.Color{R: 200, G: 200, B: 200, A: 255})\n\t\t\tcI.Restore()\n\t\t}\n\t}\n\t// 真实渲染上下文：sidebar 矩形 clip + comp-bar 圆角 clip\n\tcI.Clip(graphics.Rect{X: 48, Y: 30, Width: 278, Height: 748})\n\tcI.ClipRoundRect(1039, 695.8, 233, 12, 6)\n\tcI.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})\n\tcI.StrokeRoundRect(1039, 695.8, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})\n\timgI := toRGBA(cI.Pixels(), 2560, 1600)\n\tfmt.Println(\"\\n[场景I] 300次历史绘制后 comp-bar 圆角\")\n\tfor _, px := range []int{2078, 2082, 2086, 2090, 2094} {\n\t\tdumpPixel(imgI, 2560, 1600, px, 1392, fmt.Sprintf(\"I:顶行y1392 x%d\", px))\n\t}\n\tdumpPixel(imgI, 2560, 1600, 2090, 1394, \"I:y1394 x2090\")\n\tdumpPixel(imgI, 2560, 1600, 2542, 1400, \"I:y1400 x2542(右缘)\")\n\tdumpPixel(imgI, 2560, 1600, 2090, 1400, \"I:y1400 x2090\")\n\n\t// 场景 J：渐变绘制后立即 FillRoundRect（验证 shader 释放是否污染）\n\tcJ := graphics.NewCanvas(2560, 1600)\n\tcJ.Scale(2, 2)\n\tcJ.FillLinearGradient(48, 30, 278, 748, graphics.Color{R: 10, G: 20, B: 30, A: 255}, graphics.Color{R: 90, G: 100, B: 110, A: 255})\n\tcJ.FillLinearGradient(429, 67, 601, 711, graphics.Color{R: 100, G: 120, B: 140, A: 255}, graphics.Color{R: 200, G: 210, B: 220, A: 255})\n\tcJ.ClipRoundRect(1039, 695.8, 233, 12, 6)\n\tcJ.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})\n\timgJ := toRGBA(cJ.Pixels(), 2560, 1600)\n\tfmt.Println(\"\\n[场景J] 两个渐变后 FillRoundRect 圆角\")\n\tfor _, px := range []int{2078, 2082, 2086, 2090, 2094} {\n\t\tdumpPixel(imgJ, 2560, 1600, px, 1392, fmt.Sprintf(\"J:顶行y1392 x%d\", px))\n\t}\n\tdumpPixel(imgJ, 2560, 1600, 2542, 1400, \"J:y1400 x2542(右缘)\")\n\tdumpPixel(imgJ, 2560, 1600, 2090, 1400, \"J:y1400 x2090\")\n\n\t// 场景 K：FillRoundRect 后紧接 FillLinearGradient，再 FillRoundRect\n\tcK := graphics.NewCanvas(2560, 1600)\n\tcK.Scale(2, 2)\n\tcK.FillRect(0, 0, 100, 100, graphics.Color{R: 1, G: 2, B: 3, A: 255})\n\tcK.ClipRoundRect(1039, 695.8, 233, 12, 6)\n\tcK.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})",
     "_expanded": false
    }
   ]
  }
 }
];


/* ── 内置解析器：jsonl 全文 → combos 数组（与 conv_real.py 逻辑等价） ── */
(function (global) {
  'use strict';

  function parseConversationJsonl(jsonlText, opts) {
    opts = opts || {};
    var MAX_TURNS = opts.maxTurns || 6;
    var FOLD_FIRST = opts.foldFirst || 2;
    var RESULT_LIMIT = opts.resultLimit || 4000;

    var lines = String(jsonlText || '').split('\n');
    var combos = [];
    var cur = null;

    function flush() {
      if (cur !== null && cur.assistant && cur.assistant.segments && cur.assistant.segments.length) {
        combos.push(cur);
      }
      cur = null;
    }

    for (var i = 0; i < lines.length; i++) {
      var line = lines[i].trim();
      if (!line) continue;
      var rec;
      try { rec = JSON.parse(line); } catch (e) { continue; }
      var m = rec.message || {};
      var segs = rec.segments || [];
      var role = m.role;

      if (role === 'user') {
        flush();
        combos.push({ user: { content: m.content || '' } });
      } else if (role === 'assistant') {
        flush();
        cur = {
          assistant: {
            _folded: false,
            _summary: (m.content || '').slice(0, 60),
            segments: []
          }
        };
        for (var j = 0; j < segs.length; j++) {
          var s = segs[j];
          var t = s.type;
          if (t === 'thinking') {
            cur.assistant.segments.push({ type: 'thinking', content: s.content || '', _collapsed: true });
          } else if (t === 'tool_call') {
            cur.assistant.segments.push({
              type: 'tool_call',
              name: s.name || '',
              argsRaw: s.argsRaw || '',
              result: (s.result || '').slice(0, RESULT_LIMIT),
              _expanded: false
            });
          } else if (t === 'content') {
            cur.assistant.segments.push({ type: 'content', content: s.content || '' });
          }
          // tool_result 段：结果已并入 tool_call 的 result 字段，跳过
        }
      } else if (role === 'tool') {
        // tool 消息：内容并入上一 tool_call 的 result（若为空）
        if (cur !== null && cur.assistant && cur.assistant.segments && cur.assistant.segments.length) {
          var last = cur.assistant.segments[cur.assistant.segments.length - 1];
          if (last.type === 'tool_call' && !last.result) {
            last.result = (m.content || '').slice(0, RESULT_LIMIT);
          }
        }
      }
    }
    flush();

    // 前 FOLD_FIRST 个 assistant 回合设为整条折叠
    var folded = 0;
    var out = [];
    for (var k = 0; k < combos.length; k++) {
      var c = combos[k];
      if (c.assistant) {
        if (folded < FOLD_FIRST) { c.assistant._folded = true; folded++; }
        if (out.length >= MAX_TURNS) break;
        out.push(c);
      } else {
        out.push(c); // user 消息保留
      }
    }
    return out;
  }
  global.parseConversationJsonl = parseConversationJsonl;

  // ── 数据装配：注入优先，无注入回退内置数据 ──
  var injected = global.__CONV_JSONL__;
  var combos = null;
  if (injected && typeof injected === 'string' && injected.length > 10) {
    try {
      combos = parseConversationJsonl(injected);
    } catch (e) { combos = null; }
  }
  global.REAL_COMBOS = (combos && combos.length) ? combos : FALLBACK_COMBOS;
})(window);