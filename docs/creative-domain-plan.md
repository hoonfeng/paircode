# 创作域支持方案（图像设计 / 绘制 / UI 设计 / 建模 / 音乐 / Live2D）

> 文档状态：**方案与契约草案**（plan + contract）——供实现、验收、评审三路共用。
> 事实基线：2026-09 通过 GitHub / npm registry / 上游 README 实检索（非记忆推测），来源见 §7。
> 目标读者：框架层（插件运行时 / 工具面）+ 域插件实现者。

---

## 0. 一句话目标与核心结论

**目标**：把 PairCode 从「代码 IDE」扩展为「**可编程创作域工作台**」——Agent 与用户在会话里协作产出
图像、UI 设计稿、3D/CAD 模型、音乐、2D 角色（Live2D 类），并且每一步都能**预览、校验、导出**。

**核心结论（推导结果）**：

1. 缺的不是可扩展骨架，而是**创作产物的统一模型**。现有插件体系已具备承载能力
  （UI 槽位 / 静态资产托管 / Node 桥 npm / MCP / 视觉验证工具），但没有任何"创作产物"概念。
2. 四个域（矢量与 UI 设计、3D/CAD、音乐、2D 角色）在架构上是**同构**的，可归一到
  一套原语：**文本真相源 + 浏览器渲染器 + 粒子操作集 + 校验器 + 导出器 + 三通道预览**。
3. 落地成本最低、收益最高的一步是**把已有的「fenced code block → 专用渲染器」雏形升级为插件化渲染器注册表**，
  并给编辑器加「文件类型 → 视图」注册表；其余能力（导出/校验/可视化编辑）在此之上增量叠加。
4. 不引入需要授权或传染性许可的**重型内核**：选型优先 MIT / Apache-2.0 / BSD 且可
  在浏览器或现有 Node 桥内运行者；专业工具（Blender / FreeCAD / Figma / ComfyUI）走**可选 MCP**，
  不进默认分发（详见 §6）。

---

## 1. 现状诊断（带代码证据）

### 1.1 已经具备的扩展面

| 能力 | 位置 | 对创作域的意义 |
|---|---|---|
| UI 槽位系统（single + list 双类） | `plugins-src/ui-app/src/plugin-runtime.js:24-41`：`titlebar/activitybar/sidebar/editor/right-panel/chat/statusbar` + `overlay/titlebar-right/editor-toolbar/chat-tools/statusbar-items` | 可新增面板/工具条，无需改壳 |
| 插件静态资产托管 | `cmd/companion/web_server.go:395`（`/plugins-assets/<plugin>/<file>`，`http.ServeFile`） | 可自带预编译的 three.js / Tone.js / JSCAD bundle，运行时零编译 |
| Node 桥 + npm 依赖 | `internal/agent/node_plugins.go:120-186`（`nodePluginRuntime`：dependencies 非空即走 node 轨；`nativeDepHints` 已列 `sharp`/`@resvg/resvg-js`/`@napi-rs/canvas` 平台提示） | 服务端导出链（PNG/STL/WAV/glTF/PDF）不需要新机制 |
| 视觉验证工具 | `.pair/plugins/tool-web`（`screenshot` / `web_debug` / `read_image` / `web_fetch`） | 设计渲染的闭环校验（截图 + 视觉模型判读） |
| MCP 接入 | 宿主 `mcp_add` / 市场 | 专业工具（Blender/FreeCAD/ComfyUI/Figma）按需外接 |

### 1.2 缺口（本次要补的）

| 缺口 | 证据 | 影响 |
|---|---|---|
| 编辑器视图硬编码，仅 4 类 | `plugins-src/ui-app/src/components/EditorArea.vue:37-73`：文本 / 图片 / Hex / Markdown，`getFileType()` 内写死扩展名表 | `.svg/.scad/.gltf/.mid/.psd/.inp` 等只能当文本或二进制看 |
| 会话内渲染是硬编码白名单 | `MarkdownRenderer.vue:152`（`chartLangRe` = mermaid + 16 种自研布局 DSL）、`:168-181`、`parseLayout():245` | 已能"说布局 → 画出来"，但**不可插件扩展**，无法加乐谱 / 3D / 角色 |
| 无预览面板 | `components/` 无 Preview/Canvas 组件（有 TerminalPanel/OutputPanel） | 生成的 HTML/网页/模型没有统一查看处 |
| 无音频能力 | 全仓 grep `audio/tts/speech` 仅命中有头浏览器 `mute-audio` 与 DOM 标签白名单 | 音乐域零基础 |
| 无产物生成工具 | 工具面仅有 `read_image`（视觉输入），无任何图像/模型/音频产出工具 | Agent 只能"写代码"，不能"做东西" |
| 无产物校验 | 无 SVG/glTF/STL/乐谱/rig 校验器 | 产物错误只能靠人眼，无法回灌 Agent |

**结论**：骨架够用，缺的是**产物模型 + 渲染器/视图注册点 + 域内工具与校验**。

---

## 2. 生态横评（实检索事实）

统一评估维度：**许可证能否随 IDE 分发** → **运行位置**（浏览器 / Node / 外部进程）→
**真相源是否文本可 diff**（Agent 能否直接生成与增量修改）→ 成熟度。

### 2.1 图像设计 / 矢量绘制

| 项目 | 许可 | 形态 | 评注 |
|---|---|---|---|
| `konva` v10.5.0 / `fabric` v7.4.0 / `paper` | MIT | 框架无关 Canvas/SVG 库 | **首选**：Vue 项目零适配，可自建选择/变形/对齐/导出 |
| `SVG-Edit/svgedit` 7.8k★ | MIT | 完整浏览器 SVG 编辑器 | 可 iframe 全量嵌入，快速获得成熟编辑体验 |
| `excalidraw/excalidraw` v0.18.1 | MIT | React 组件 | 白板/手绘最强，但需 React runtime（当前壳是 Vue） |
| `tldraw/tldraw` v5.4.2 | **生产需 license key**（README 明示） | React SDK | 能力最强但**不作为默认依赖** |
| `ZSeven-W/openpencil` 5.9k★ | MIT | Rust+TS 桌面应用（Design-as-Code、内建 MCP） | 架构参照物（Prompt→Canvas、Agent Teams），非嵌入件 |
| `vercel/satori` | MPL-2.0 | HTML/CSS → SVG | 设计稿导出 PNG/SVG 的关键件 |
| `linebender/resvg`（Rust 库）/ `@resvg/resvg-js` v2.6.2 / `lovell/sharp` v0.35.4 | Apache-2.0 / **MPL-2.0** / Apache-2.0 | SVG→PNG / 图像处理 | Node 桥导出链（`resvg-js` 是 Node 绑定，许可与上游 Rust 库不同） |
| `visioncortex/vtracer` 7.0k★ | MIT | 位图→矢量 | 让"手绘/截图 → 可编辑矢量"成立 |
| `pend`/Penpot 60k★ | MPL-2.0 | Clojure 服务端设计平台 | 不适合嵌入，仅作 FOSS 设计平台参照 |

**取舍**：默认路径 = **文本真相源（SVG / Excalidraw JSON / 自研绘图 DSL）+ 自建 Vue 画布（Konva 或 SVG-DOM）+ 导出（satori / resvg / 浏览器截图）**；
Excalidraw 作为**可选**重画布插件（需要 React 隔离壳）；tldraw 排除。

### 2.2 UI 设计

| 项目/spec | 许可 | 说明 |
|---|---|---|
| `google-labs-code/design.md` 27.9k★ | Apache-2.0 | **DESIGN.md 规范**：YAML 设计 token + markdown 设计理由；`npx @google/design.md lint` 可校验 token 引用与 WCAG 对比度并输出结构化 JSON |
| `GLips/Figma-Context-MCP` 15.9k★ | MIT | Figma → Agent 布局信息 |
| `awdr74100/figwright` | MIT | 双向 Figma MCP（设计↔代码） |
| `onlook-dev/onlook` 26.7k★ | Apache-2.0 | React 可视化编辑（Agent 驱动） |
| `bradtraversy/design-resources-for-developers` 66.9k★ | MIT | 组件/资源索引 |

**取舍**：UI 设计域不追求做 Figma 替代，而是做 **「DESIGN.md（token + 理由）→ 组件实现 → 截图验证 → 设计评审」闭环**，
把现有 16 种布局 DSL（`MarkdownRenderer.parseLayout`）升级为**可插件注册的 UI 布局渲染器族**，
并加 `@google/design.md` 风格 lint（对比度/Token 引用）作为校验器。

### 2.3 3D / CAD 建模

| 项目 | 许可 | 运行位置 | 评注 |
|---|---|---|---|
| `@jscad/modeling` v2.13.0（OpenJSCAD 3.2k★） | MIT | 浏览器 + Node | 纯 JS CSG，真相源=JS 脚本，导出 STL/3MF/DXF/SVG **首选内核** |
| `sgenoud/replicad` v1.1.0（684★） | MIT | 浏览器（OCCT wasm） | 参数化 CAD，支持 STEP 导出（体积大，按需懒加载） |
| `elalish/manifold` v3.5.3（2.3k★） | Apache-2.0 | wasm | 拓扑稳健布尔运算，作为网格修复/布尔后端 |
| `openscad/openscad` 10.2k★（GPL-2.0+，GitHub 标 NOASSERTION）+ `openscad-wasm`（GPL-2.0） | **GPL** | wasm | 语言生态强但 GPL 传染 + 体积大 → 走**可选外部工具**，不默认分发 |
| `mrdoob/three.js` v0.186.0 / `BabylonJS` | MIT / Apache-2.0 | 浏览器 | 视图层首选 three.js（生态/体积可控） |
| `google/model-viewer` v4.3.1 | Apache-2.0 | 浏览器 | 轻量 glTF/AR 查看，适合"只读预览" |
| `@gltf-transform/core` v4.5.0 | MIT | Node + 浏览器 | glTF 优化/转换（压缩、LOD） |
| `CadQuery` / `gumyr/build123d` | Apache-2.0 | 外部 Python | 工程级 STEP（Node 桥可调），按需 |
| `neka-nat/freecad-mcp`（2.3k★, MIT） / `ahujasid/blender-mcp`（28.7k★, MIT） | MIT | 外部进程 MCP | 专业建模器接入，可选 |
| `microsoft/TRELLIS`（13.6k★, MIT）/ `VAST-AI-Research/TripoSR`（7.0k★, MIT）/ `Hunyuan3D`（许可待核） | — | GPU 服务 | 图像/文本→3D，**可选**远端或本地服务 |
| `kovacsv/Online3DViewer`（3.7k★, MIT） | MIT | 浏览器 | 20+ 格式查看参照（STEP/IGES/BREP），可作为格式清单来源 |

**取舍**：默认 = **JSCAD（MIT，脚本即模型）+ three.js 视图 + glTF/STL 导出**；
工程级（STEP/BIM）× 专业雕刻 × AI 生成 = **三类可选外部通道**（MCP/外部进程/远端服务）。

### 2.4 音乐

| 项目 | 许可 | 评注 |
|---|---|---|
| `Tonejs/Tone.js` v15.1.22（14.7k★） | MIT | WebAudio 合成/时序；`OfflineAudioContext` 可**离线渲染 WAV** |
| `0xfe/vexflow` v5.0.0（4.4k★） / `opensheetmusicdisplay` v2.1.2（2.0k★, BSD-3） / `paulrosen/abcjs` v6.7.0（2.3k★） | MIT / BSD-3 / MIT | 记谱渲染（OSMD 支持 Node headless，适合导出 PNG/PDF） |
| `@tonejs/midi` v2.0.28 / `midi-writer-js` v3.2.1 | MIT | MIDI 读写（Agent 产物落地格式） |
| `smplr` v1.0.0 / `soundfont-player` | MIT | 采样音源（听到真实音色） |
| `FluidSynth`（LGPL-2.1）/ `MuseScore`(GPL-3) / `LilyPond`(GPL) | 传染性 | 高质量排版/渲染，走**可选外部工具** |
| `ace-step/ACE-Step`（4.8k★, Apache-2.0）/ `YuE`（8.9k★, Apache-2.0）/ `Magenta`（Apache-2.0） | 宽松 | 音乐生成模型（本地/远端） |
| `tidalcycles/strudel` / `Meadowlark` | **AGPL-3.0** | 规避（不进分发） |
| `LMMS`（GPL-2.0） | 传染 | DAW 参照 |
| **【人声子域】** `audiojs` 原子库族 `@audio/*`（40+ 包：`pitch-yin` / `tune-snap` / `tune-midi` / `shift-*`(15) / `stretch-*`(9) / `onset` / `beat` / `vad` / `denoise` / `eq` / `dynamics` / `reverb` / `loudness` / `chain` / `assert` / `vocals` / `midi` / `wam`） | **MIT**（逐包实测 2026-09） | 单职责 ESM DSP 原子，**无模型**，恰好覆盖"人声创作"全链；详见 §2.4.1 |
| **【人声子域】** `signalsmith-stretch` v1.3.2 MIT、`@soundtouchjs/audio-worklet` v2.1.1 MPL-2.0、`wavesurfer.js` v7.12.12 BSD-3、`@echogarden/sonic-wasm` Apache-2.0、`node-web-audio-api` v2.2.0 BSD-3 | 宽松 / 弱 copyleft | 高质量伸缩 / 实时 worklet / 波形编辑 UI / 服务端渲染 |

**取舍**：默认 = **MIDI / ABC / MusicXML 文本真相源 + Tone.js 播放 + VexFlow/OSMD/abcjs 渲染 + 离线 WAV 导出**；
AI 音乐生成与专业排版走可选通道。

#### 2.4.1 音乐·人声子域（说话 / 歌声）——非换声 · 非生成式

用户约束：**不用换声（voice conversion）模型、不用生成式模型**，直接基于用户提供的人声创作。

- 结论：可用 `@audio/*`（MIT）纯 DSP 原子搭出完整链 —— 导入（`decode`/`mic`）→ 分析（`pitch-yin` 基频、
  `onset`/`beat` 分段与节拍、`vad`、`mir` 调性、`loudness` 响度）→ 处理（`tune-snap` 音阶吸附、`tune-midi` 按 MIDI 重调、
  `shift-*` 变调含共振峰保持、`stretch-*` 时值伸缩、`denoise`/`eq`/`dynamics`/`reverb`/`saturate`/`spatial` 效果）→
  编排（`midi` 符号驱动、切片重排、和声叠加）→ 渲染（离线确定性）→ 导出（WAV/MIDI/乐谱）。
- 三条创作闭环：**A 音高-时间编辑（Melodyne 类）**、**B 人声→MIDI→人声（用自己声音唱新旋律）**、**C 切片编排（人声乐器化）**。
- 承诺靠机制落地而非口头：**帧级可追溯（sourceMap 重建 ≥0.99 互相关）+ 音色一致性（包络偏差阈值）+ 确定性可复现（≤-120 dBFS）+
  依赖审计（禁模型包与权重文件）** —— 生成式/换声产物在机制上无法通过（详见详规 §7）。
- **排除**：RVC / so-vits-svc / DiffSinger / Fish-Speech / ElevenLabs / ACE-Step / YuE / Suno / utaformatix（换声与生成式）；
  `pitchfinder`（**GPL**）、`essentia.js`（**AGPL-3.0**）、`aubio`（GPL-3.0）、Rubber Band（GPL-2.0）、`peaks.js`/`lamejs`（LGPL）不进默认链。
- 能力上限（诚实标注）：可"修正/重组/编排用户已发出的声音"，**不能**唱出用户从未发出的音素，也不能跨性别音色（那需换声模型）；
  >±7 半音变调、>1.5× 拉伸会显著劣化，必须 UI 提示。
- **已 PoC 实证**（2026-09，`_temp/voice-poc/poc.mjs` + `poc-report.txt`）：装 `@audio/{pitch-yin,tune-snap,loudness,onset}`
  （25 包，9 s）实跑六环 —— YIN 检出 F0（系统偏差 ≈ -5.5 ¢）、音阶吸附只动跑调段（+24.7 ¢，音阶内段 +0.0 ¢）、
  低次谐波包络偏差 4.2%（音色保持）、**两次渲染位级一致（maxDiff = 0）**、LUFS -16.41 → -16.40、onset 过检 18 vs 期望 3
  → 实测结论已回写详规 §7.0（阈值标定）与 §2.2/§8.9（切片后处理、吸附二次精修）。

> **人声子域完整详规（六段链契约 / 工程 schema / 插件工具与 UI / 两套场景预设 / 六项专项验收 / 质量上限 / 分期）：见 `docs/voice-creation-plan.md`。**

### 2.5 Live2D / 2D 角色（含"会话形象"）

| 项目 | 许可 | 评注 |
|---|---|---|
| `zeikar/iki`（`@ikijs/engine` v0.3.2，MIT，2026-09 新） | MIT | **最契合**：`.iki` = 明文 JSON rig 格式；WebGL2 运行时；`@ikijs/editor` 提供无 UI 编辑核心（EditorDocument/命令/undo/atlas UV/**auto-rigger**）；`@ikijs/mcp` 暴露 `compose_layers_from_parts`/`measure_layers`/**`auto_rig_from_layers`**（角色命名 PNG/PSD 图层 → 绑定模型）。状态：0.x 早期 |
| `Inochi2D/inochi2d`（1.8k★, BSD-2）/ `inochi-creator`（BSD-2） | BSD-2 | 成熟开源 Live2D 替代，但原生（D 语言）；web 侧弱 |
| Live2D Cubism SDK / Spine | **专有**（分发需发布许可） | 仅"用户自带 SDK/资产"兼容通道，**不随 IDE 分发** |
| `tsunehimatoi/psd2live`（GPL-3）/ `umamoorg/umamo`（GPL-3） | 传染 | PSD→Live2D 自动绑定参照实现，**算法参照，不嵌入** |
| `Open-LLM-VTuber`（13.8k★）/ `zeikar/charivo`（MIT）/ `moeru-ai/airi`（49.2k★） | 混合 | **会话 + 角色**参照：LLM + STT/TTS + 口型/眨眼/表情参数驱动 |
| `@mediapipe/tasks-vision` v1.0.1 | Apache-2.0 | 面部捕捉 → 参数驱动（可选） |
| `ag-psd` v31.0.2 | MIT | PSD 分层读写（**角色建模的输入格式**） |

**取舍**：走**开放 rig 格式（优先 `.iki` 兼容）+ PSD/PNG 分层导入（ag-psd）+ 自动绑定 + WebGL 渲染 + 参数驱动**
这条 MIT 全链；Cubism 仅作导入/播放的用户自带通道。同时——**该能力直接服务"会话形象"**（§4.4）。

### 2.6 会话交互（对话式创作）参照

- `OpenPencil`：Prompt→Canvas 流式生成，选中元素后聊天修改（会话即编辑器）。
- `onlook`：可视化编辑 + Agent 协作。
- `figwright` / `Figma-Context-MCP`：设计 ↔ 代码双向。
- `Open-LLM-VTuber` / `charivo`：会话 + 角色 + 语音。

→ 共同点：**会话是操作入口，画布是反馈界面，产物是文件**。本方案据此设计 §4.4 的三条会话通道。

---

## 3. 统一抽象：创作域原生（Creative Artifact）

四个域归一到同一组原语（这是"更好的支持"的实质——一套机制服务全部创作方向）：

```
Artifact := {
  id, kind,            // svg | ui | model3d | score | audio | rig
  source,              // ★ 文本真相源（SVG / JSON / JS / ABC / MusicXML / .iki）——可 diff、可 apply_patch
  renderers[],         // 浏览器渲染器（预编译 bundle，经 /plugins-assets/ 提供）
  views[],             // 编辑器视图（按扩展名/魔法字节匹配）
  ops[],               // 粒子操作集（与可视化编辑、Agent 工具共用同一命令模型）
  validators[],        // 域内校验（结构/对比度/水密/音域/参数范围）
  exporters[],         // 导出（png/svg/gltf/stl/3mf/mid/wav/pdf）
  preview              // 三通道预览（编辑器视图 / 会话内嵌 / 叠加面板）
}
```

**六条铁律**（选型与实现的判定标准）：

1. **真相源必须文本**：二进制（MIDI/PSD/glb/wav）只作**产物**，源一律可 diff（ABC/MusicXML/JSON/脚本/SVG）。
2. **渲染器在浏览器**：即时反馈，零安装；重内核（OCCT/OpenSCAD/排版器）**懒加载或外置**。
3. **操作即命令**：同一 `ops` 序列同时服务可视化编辑（用户拖拽 → ops）与 Agent（工具调用 → ops），可撤销、可复现。
4. **校验先于展示**：每个产物渲染前过 validator，错误结构化回灌 Agent（而非让人眼看）。
5. **预览三通道**：同一 artifact 在编辑器视图、会话消息内、叠加/独立面板三处可渲染（同一渲染器复用）。
6. **导出即证据**：导出成功 = 产物可被外部工具打开（用 Node 脚本二次校验，见 §5 验收）。

---

## 4. 架构落地（对现有代码的扩展点）

### 4.1 框架层：四个新扩展点（一次性，全部为加法）

| 编号 | 扩展点 | 落点 | 契约 |
|---|---|---|---|
| **E1** | 渲染器注册表 | `plugin-runtime.js`（新增 `ui.registerRenderer`）；消费点 `MarkdownRenderer.vue:152` 的 `chartLangRe` 改为查注册表，内建 mermaid/layout 降级为**默认条目** | `registerRenderer({ id, languages:[], exts:[], mount(el, ctx), unmount?, actions? })` |
| **E2** | 编辑器视图注册表 | `EditorArea.vue:37-73` 的 if/else 链改为查注册表；`getFileType()` 保留 text/image/binary 为默认 | `registerView({ id, match:{exts:[],mime:[]}, mount(el, {path, getContent, onChange}), toolbar? })` |
| **E3** | Artifact 工具面服务 | 宿主工具侧（host 半可用）：`ctx.artifact.open/applyOps/validate/export`；工具名建议 `artifact_open` / `artifact_apply` / `artifact_validate` / `artifact_export` | 统一参数：`{ path, kind, ops?, target? }`；返回结构化产物（含校验结果与产物路径） |
| **E4** | 预览/媒体托管 | 复用 `/plugins-assets/`（插件 bundle）+ `/fs/*`（工作区产物）；新增只读 `artifact-preview` 静态路径（如需，产物落 `_temp/` 或工作区可见目录） | 生成物必须可被 `screenshot`/`web_debug` 直接访问以做视觉校验 |

> 设计约束：E1/E2 必须**纯加法**——插件未注册时行为与今天完全一致（mermaid + 16 布局 DSL 照旧，4 类视图照旧）。

### 4.2 域插件（每个域一对 host/ui 插件，沿用现有三形态）

| 域 | host 半（工具） | ui 半（渲染/编辑） | 关键依赖（许可证） |
|---|---|---|---|
| 矢量绘制/图像 | `tool-art`：新建/编辑/导出（PNG/SVG）、位图转矢量 | `ui-art`：SVG 画布（选择/变形/对齐/图层）、导出、颜色/对比度检查 | 自研画布 + Konva(MIT) 或 SVG-DOM；导出 `@resvg/resvg-js`(MPL-2.0) / satori(MPL-2.0) |
| UI 设计 | `tool-design`：DESIGN.md token 读写、lint、组件生成 | `ui-design`：布局 DSL 渲染器族（升级 parseLayout）、设计稿预览、design token 面板 | mermaid(MIT, 已有) + DESIGN.md spec(Apache) + 浏览器渲染 |
| 3D / CAD | `tool-model`：脚本建模（JSCAD DSL）、布尔、导出 STL/3MF/glTF、几何校验 | `ui-model`：three.js 视图（轨道/网格/线框/剖面）、多视角截图 | three(MIT) + @jscad/modeling(MIT) + @gltf-transform(MIT) + manifold(Apache，可选) |
| 音乐 | `tool-music`：MIDI/ABC/MusicXML 读写、编曲操作、离线渲染 WAV | `ui-music`：乐谱渲染 + 播放器 + 钢琴卷帘（可选） | Tone(MIT) + VexFlow/OSMD/abcjs + @tonejs/midi |
| 音乐·人声 | `tool-voice`：`voice_import` / `voice_analyze` / `voice_edit` / `voice_render` / `voice_export` / **`voice_verify`**（非换声非生成式自检） | `ui-voice`：波形+F0 曲线编辑、音符网格、切片视图、多轨编排、效果机架 | `@audio/*`(MIT) + signalsmith-stretch(MIT) + @soundtouchjs/audio-worklet(MPL-2.0) + wavesurfer.js(BSD-3) + Tone(MIT) |
| 2D 角色（Live2D 类） | `tool-rig`：PSD/PNG 分层导入、parts 命名、自动绑定、参数/物理编辑、导出 `.iki` | `ui-rig`：WebGL 角色预览 + 参数滑块 + 表情/动作库 | ag-psd(MIT) + WebGL2 自研或 `@ikijs/*`(MIT) + MediaPipe(Apache，可选) |

> 说明：`ui-*` 插件自带 Vite lib 产物（现有 `scripts/build-ui.mjs` 可扩展 `--region`），
> 第三方库二选一：打进 bundle 或从 `/plugins-assets/<plugin>/vendor/*` 懒加载（推荐后者，控制首屏体积）。

### 4.3 会话内嵌（消息渲染器）：把"说"变成"看/改"

升级 `MarkdownRenderer` 的白名单为注册表后，会话即刻获得（示例语言标签）：

```
```svg / ```art      → 矢量画布（可选中、可直接「应用到文件」）
```ui / ```layout    → UI 设计稿渲染（已有 16 种布局 DSL 成为其中一个渲染器）
```scad / ```jscad   → 3D 模型预览（three.js，可旋转/线框/导出）
```gltf / ```stl     → 模型查看（model-viewer 轻量路径）
```abc / ```musicxml → 乐谱渲染 + 播放（VexFlow/OSMD + Tone.js）
```midi              → 播放器 + 钢琴卷帘
```vocal             → 人声：波形 + F0 曲线静态图（说话/歌声分析结果，可「应用到工程」）
```iki / ```rig      → 角色预览 + 参数驱动
```

每条消息区块附「打开为文件 / 应用到当前文件 / 导出」动作 → **会话产出直接沉淀为工作区文件**（可 apply_patch、可版本管理）。

### 4.4 会话交互层（用户补充重点）

| 编号 | 通道 | 实现 | 价值 |
|---|---|---|---|
| **S1** | 消息内嵌渲染（上节） | E1 渲染器注册表 | 会话里"看得见"产物 |
| **S2** | `chat-tools` 槽位工具条 | 现有 list 槽位，插件注册：新建画布/模型/乐谱、当前 artifact 切换、导出 | 创作入口零跳转 |
| **S3** | 画布 ↔ 文件 ↔ Agent 三方同步 | E3 `artifact_apply` + 文件写前快照（既有机制） | 用户手改与 Agent 改**同一真相源**，均可撤销 |
| **S4** | Live2D 会话形象 | `ui-rig` 插件 + 独立面板/叠加槽位；参数由会话状态驱动（说话/思考/表情） | 会话伴侣化（参照 Open-LLM-VTuber/charivo 架构：LLM + TTS/STT + rig 参数） |
| **S5** | 多模态反馈回路 | 已有视觉（screenshot/read_image）；音乐用**乐谱图 + 波形图**作为可判读产物；3D 用**多视角拼图** | 让 LLM 能"看到"并自我纠正 |

> S4 明确边界：**语音（TTS/STT）不在本方案 P0/P1 范围**（当前仓库无音频依赖），仅预留参数驱动接口。

### 4.5 校验与导出矩阵（"完成"的判定标准）

| 域 | 校验器 | 导出器 | 校验手段（可自动化） |
|---|---|---|---|
| 矢量/图像 | SVG 结构、视口越界、颜色对比度 | PNG/SVG/PDF | 渲染后截图 + `read_image` |
| UI 设计 | DESIGN.md token 引用 + WCAG 对比度 | HTML/截图 | 无头浏览器 DOM 断言 + 截图 |
| 3D/CAD | 三角数/水密/体积/包围盒/manifold 检查 | STL/3MF/glTF | 脚本断言 + 多视角截图 |
| 音乐 | 音域/时值/轨道合法性（SMF 解析） | MID/MusicXML/WAV | `@tonejs/midi` 解析产物 + 乐谱截图 |
| 音乐·人声 | **非换声非生成式六项**（依赖审计/帧级可追溯/音色一致性/确定性可复现/链可见/无新增音源）+ 响度（BS.1770-4）+ F0 序列合法性 | WAV/FLAC/MID/乐谱图/分轨 | `voice_verify` + `@audio/assert`（golden 回归）+ 波形/F0/乐谱截图 + `read_image` |
| 2D 角色 | rig schema 校验 + 参数范围 + 部件重叠 | `.iki`/JSON | schema validator + 渲染截图 |

---

## 5. 分期路线与验收

### P0（框架打通 + 最小可用闭环）

1. E1 渲染器注册表 + 3 个新渲染器：`abc`（乐谱）、`jscad`（3D）、`svg`（矢量）。
2. E2 编辑器视图注册表 + 2 个新视图：`.svg`（矢量）、`.abc/.mid`（乐谱/播放）。
3. E3 最小 `artifact_export`（浏览器端：PNG/STL/MID；导出物写工作区）。
4. S2 会话工具条（新建/导出两个按钮即算通过）。

**验收（可复现，独立二进制 + 非默认端口）**：

- 会话投喂 ```` ```abc ````、```` ```jscad ````、```` ```svg ```` 三个代码块 → 三张截图 + `read_image` 判定渲染正确；
- `.svg` 文件在编辑器内以画布打开（非文本）；导出 PNG 存在且尺寸正确；
- 导出 STL 用 Node 脚本统计三角数 > 0；导出 MID 用 `@tonejs/midi` 解析出音符数 > 0；
- 插件全部停用时，`MarkdownRenderer` 行为与改造前逐字节一致（回归）。

### P1（域能力补齐）

5. 矢量画布可编辑（选择/移动/变形/对齐/图层）→ ops 回写文件（S3）。
6. UI 设计域：DESIGN.md token + lint（对比度）+ 布局 DSL 渲染器化。
7. 服务端导出链（Node 桥）：`@resvg/resvg-js` / satori（PNG/SVG，均 MPL-2.0——弱 copyleft，随分发须保留声明并同步 `docs/THIRD_PARTY_NOTICES.md`）、ffmpeg（WAV→MP3/MP4，可选）、gltf-transform。
8. `artifact_validate` 全量校验器 + 错误结构化回灌 Agent。

### P2（高级与外部）

9. 2D 角色全链：PSD 分层 → parts 命名 → 自动绑定 → 参数/物理 → 会话形象（S4）。
10. 可选外部专业通道（MCP/外部进程）：Blender / FreeCAD / Figma / ComfyUI（图像生成）/ MuseScore。
11. AI 生成通道（3D：TRELLIS/TripoSR；音乐：ACE-Step/YuE；图像：ComfyUI/远端 API）——**均需用户显式启用**。

### V-P0 / V-P1 / V-P2（音乐·人声子域，详见 `docs/voice-creation-plan.md` §9）

- **V-P0**：导入（文件+麦克风）→ 分析（F0/音符/节拍/静音）→ 编辑（音阶吸附 / 按音符重调 / 时值伸缩 / 删停顿）→
  离线渲染 + WAV/MIDI 导出；**`voice_verify` 最小集 = 依赖审计 + 确定性 + sourceMap 重建**；UI 最小波形+F0 编辑器；会话内嵌 ` ```vocal `。
- **V-P1**：人声→MIDI→人声闭环、切片与重排（人声乐器化）、和声叠加、效果机架 + 响度交付、服务端批量渲染、`voice_verify` 全六项 + `@audio/assert` golden 回归入 CI。
- **V-P2**：可选模型辅助（demucs 分离 / basic-pitch 转 MIDI / whisper 歌词对齐，默认关闭且进审计白名单）、音素级切片标签、MP3/视频导出（外部 ffmpeg）。

---

## 6. 边界、风险与明确排除

| 项 | 处置 |
|---|---|
| tldraw 生产 license、Live2D Cubism / Spine 专有分发 | **不默认依赖**；仅"用户自带"通道 |
| GPL/AGPL 传染（openscad-wasm、psd2live、umamo、Strudel、Meadowlark、live2d-widget） | **不进默认分发**；算法参照或可选外部工具/MCP |
| wasm 体积（OCCT / OpenSCAD / manifold） | 懒加载或外置进程；P0 只用 JSCAD（纯 JS） |
| `zeikar/iki` 0.x 早期（7★） | 采用**格式兼容 + 自研最小渲染**策略，避免硬依赖；上游 API 变动不影响我们 |
| 二进制不可 diff（MIDI/PSD/glb/wav） | 只作产物；真相源保持文本 |
| MPL-2.0 依赖（`@resvg/resvg-js`、`satori`、Penpot） | 弱 copyleft（文件级）：可随 IDE 分发，但须保留许可声明并更新 `docs/THIRD_PARTY_NOTICES.md`；不修改其源文件即可正常使用 |
| 音频许可雷区（人声子域） | `pitchfinder`(GPL)、`essentia.js`(AGPL-3.0)、`aubio`(GPL-3.0)、Rubber Band(GPL-2.0)、`peaks.js`/`lamejs`/`soundtouchjs`(LGPL) **不进默认分发**；默认链只用 MIT/BSD/Apache |
| `@audio/*` 上游为 2026 年新包（单维护者、下载量百级） | **vendor 关键原子**（pitch-yin / shift-psola / shift-formant / stretch-wsola / stretch-transient / tune-* / onset / beat / loudness）+ 锁版本 + 保留 MIT 声明 |
| 声音权/肖像权与素材授权 | 本方案不提供换声能力（降低滥用面）；输出携带 provenance 可追溯；用户素材与伴奏音源授权责任在用户，UI 需提示 |
| 版权/素材来源 | 生成物与导入素材需提示用户注意授权（尤其字库、音源、模型权重） |
| 音频/语音输入输出 | 本方案**不含 TTS/STT**（不引入语音合成）；人声子域只做"用户素材的 DSP 变换 + 编排 + 渲染导出" |

---

## 7. 事实来源（本次实检索，2026-09）

- GitHub REST API（`/search/repositories`、`/repos/*`）：stars / license / 最近推送时间，脚本 `_temp/creative-research/gh*.mjs`。
- npm registry（`registry.npmjs.org/<pkg>`、`/-/v1/search`）：版本 / license / 时间 / 依赖数，脚本 `_temp/creative-research/npm.mjs`。
- 上游 README 实抓：`zeikar/iki`（`.iki` 格式、MCP 工具、auto-rig、与 Live2D/Inochi2D 对比表）、
  `google-labs-code/design.md`（DESIGN.md 规范与 lint）、`ZSeven-W/openpencil`（Design-as-Code、Prompt→Canvas）、
  `tldraw/tldraw`（生产 license 要求）。
- 本地代码证据：`EditorArea.vue:37-73`、`MarkdownRenderer.vue:152/168-181/245`、`plugin-runtime.js:24-41`、
  `cmd/companion/web_server.go:395`、`internal/agent/node_plugins.go:120-186`。
- 人声子域（2026-09 实检索）：npm registry 逐包查版本/license/type + `api.npmjs.org/downloads` 下载量（`@audio/*` 40+ 包全 MIT、
  `signalsmith-stretch` MIT、`@soundtouchjs/*` MPL-2.0、`pitchfinder` GPL、`essentia.js` AGPL-3.0、`rubberband-wasm` GPLv2、
  `wavesurfer.js` BSD-3、`node-web-audio-api` BSD-3）；GitHub REST API（`xiph/rnnoise` BSD-3、`facebookresearch/demucs` MIT、
  `spotify/basic-pitch` Apache-2.0、`marl/crepe` MIT、`MTG/essentia.js` AGPL-3.0、`mmorise/World` NOASSERTION）；
  上游页面实抓（`github.com/audiojs/tune` 的 `snap`/`tuneMidi` API 与实现路径、`github.com/orgs/audiojs/repositories` 71 仓库清单）。

**已核实**（npm registry 实查）：`@resvg/resvg-js` v2.6.2 = MPL-2.0、`satori` v0.33.4 = MPL-2.0、
`vexflow` v5.0.0 = MIT（GitHub 因许可文件格式标 NOASSERTION）、`@jscad/modeling` v2.13.0 = MIT。
**待核实项（落地前必须确认）**：`Tencent-Hunyuan/Hunyuan3D-2.1` 许可（GitHub 标 NOASSERTION）；
Live2D Cubism SDK 最新条款正文；`@ikijs/*` 是否随版本调整 schema（0.x 早期）。
