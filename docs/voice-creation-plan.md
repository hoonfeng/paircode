# 人声创作子域详规（说话 / 歌声）——非换声 · 非生成式

> 归属：`docs/creative-domain-plan.md`（创作域总方案）的音乐分支详规。
> 硬边界（用户设定）：**不使用换声（voice conversion）模型，不使用生成式模型**；创作必须直接基于用户提供的人声（说话 / 歌声）。
> 事实基线：2026-09 npm registry / GitHub / 上游 README 实检索（§10），非记忆推测。

---

## 0. 定位：不"造声音"，只"编辑与重组用户自己的声音"

三条不可协商的边界及其**机制化保证**：

| 边界 | 定义 | 机制保证（可验收） |
|---|---|---|
| 不换声 | 不得把用户音色替换为其他说话人/模型音色 | 帧级可追溯（sourceMap）+ 音色一致性检验（§7.2、§7.3） |
| 不生成 | 不得用 TTS/歌声合成/生成式模型产出音频或音色 | 依赖审计（禁模型包与模型权重文件）+ 确定性可复现（§7.1、§7.4） |
| 不无源合成 | 输出必须是用户素材的变换，不可出现源之外的新音源 | 输出-源互相关/指纹检查 + 链可见（§7.5、§7.6） |

**能力上限（诚实声明）**：本子域能"修正、重组、编排用户已经唱/说出来的声音"（音准、时值、音域微调、切片重排、和声叠加、效果、编排）；
**不能**让用户唱出其从未发出的音素，也不能把男声"变成"女声音色（那属于换声模型范畴，明确排除）。
跨音域（>±12 半音）与"说话→歌唱"的极限区间，质量会降到不可用，必须在 UI 里显式提示而非偷偷降级。

---

## 1. 事实基线：可用内核

### 1.1 核心发现 — `audiojs` 原子化 DSP 库族（全部 MIT，2026-09 活跃）

`audiojs` 组织（maintainers: dfcreative / jamen；71 个仓库，2026-09 大量推送）发布了一整套
**单职责、ESM、无模型** 的音频 DSP 原子包 `@audio/*`，恰好覆盖人声创作全链：

| 环节 | 包（版本 / 许可 / 近 30 天下载） | 能力 |
|---|---|---|
| 解码/编码 | `@audio/decode` 3.15.0 MIT、`@audio/encode` 1.6.5 MIT | Node+浏览器统一解码/编码 |
| 采集 | `@audio/mic` 1.1.3 MIT（1659） | 麦克风采集（Node/浏览器） |
| 基频 F0 | `@audio/pitch-yin` 1.0.5 MIT（2391，YIN de Cheveigné & Kawahara 2002）、`@audio/mir-melody` 1.1.3 MIT（1288，帧级 F0 + voicing） | 音高轨迹 |
| 分段/速度 | `@audio/onset` 1.0.1 MIT（2714，谱通量/能量通量 ODF + 自适应峰值）、`@audio/beat` 2.1.3 MIT（2722，tempo 估计 + beat tracking）、`@audio/vad` 1.0.1 MIT（能量 + 谱平坦度） | 音符/音节/节拍/静音边界 |
| **音高修正** | `@audio/tune-snap` 1.1.3 MIT（YIN → 音符分段 → 音阶吸附 → PSOLA 重调，Auto-Tune 类）、`@audio/tune-midi` 1.0.2 MIT（按参考 MIDI/音符重调，**Melodyne 类**） | 只改音高/时值，不动音色 |
| 变调 | `@audio/shift` 1.1.4 MIT（15 原子：pvoc / peak-locked / PSOLA / WSOLA / granular / LPC / SMS / HPSS / formant / transient / delay / sample …） | 按素材性质选算法 |
| 时间伸缩 | `@audio/stretch` 2.0.5 MIT（9 原子：WSOLA / PSOLA / pvoc / pvoc-lock / PGHI / SMS / hybrid / transient / PaulStretch） | 语速、时值、loop、节奏对齐 |
| 共振峰/音色 | `@audio/shift-formant`（倒谱包络重贴）、`@audio/shift-lpc`（源-滤波残差重调）、`@audio/shift-hpss`、`@audio/shift-sms` | 变调不"花栗鼠化" |
| 频谱工具 | `@audio/stft` 1.0.6 MIT、`@audio/spectral` 1.2.4 MIT（8 特征）、`@audio/sinusoidal` 1.0.1 MIT（部分音跟踪/加性合成/残差）、`@audio/auditory` 1.0.2 MIT（Bark/ERB/Mel/Gammatone）、`@audio/resample` 1.2.2 MIT | 谱编辑与可视化 |
| 效果链 | `@audio/effect` 2.1.5、`@audio/eq` 1.0.3、`@audio/dynamics` 0.2.6（压缩/限幅/门/扩展）、`@audio/reverb` 1.0.2、`@audio/saturate` 1.0.3、`@audio/filter` 3.2.1、`@audio/denoise` 0.3.9（门/谱减/Wiener/OM-LSA/dehum/declick/decrackle/declip）、`@audio/spatial` 1.0.3（声像/加宽/Haas）—— 均 MIT | 人声处理机架 |
| 响度交付 | `@audio/loudness` 1.2.1 MIT（7677，BS.1770-4 LUFS / true peak / LRA / ReplayGain / DR / speech contrast） | 播客/歌曲交付标准 |
| 符号与编排 | `@audio/midi` 1.0.3 MIT（SMF 解析/写入/soundfont 渲染/MML）、`@audio/note` 1.0.2 MIT（Hz↔MIDI↔音名/cents/scales/吸附）、`@audio/mir` 1.1.3 MIT（chroma/chord/key） | 人声 ↔ MIDI 双向 |
| 工程与验收 | `@audio/chain` 0.1.0 MIT（measure → pick → configure → run，输出**可见 recipe**）、`@audio/assert` 0.2.0 MIT（峰值/RMS/静音/削波/DC/**音高**/**golden 回归**）、`@audio/wam` 0.1.2 MIT（原子 → AudioWorklet/WAM 节点） | 可审计 + 可回归 |
| 人声提取 | `@audio/vocals` 1.0.5 MIT（1452，mid/side 中心声道人声隔离/去除） | 从立体声混音取回**用户自己的**人声（纯 DSP） |
| 辅助合成（非本链核心） | `@audio/synth` 1.2.2、`@audio/voice` 1.0.2（vocal tract / voder / glottal source） | 伴奏与辅助音源（源-滤波合成，非模型） |
| 神经通道（**不启用**） | 仓库 `audiojs/neural`（runtime adapter / denoise / amp / separation）存在，npm 无 `@audio/neural` 包（404） | 路线图，与"非模型"立场自洽 |

### 1.2 补充内核（非 audiojs）

| 库 | 许可 | 用途 |
|---|---|---|
| `signalsmith-stretch` v1.3.2 | **MIT** | 高质量 time-stretch / pitch-shift（JS/WASM，含共振峰控制） |
| `@soundtouchjs/audio-worklet` v2.1.1（55k 下载/月）+ `@soundtouchjs/formant-correction-worklet` | **MPL-2.0** | 实时 AudioWorklet 时间伸缩 / LPC 共振峰保持 |
| `@echogarden/sonic-wasm` v0.2.0 | Apache-2.0 | Sonic 时间伸缩 wasm（轻量） |
| `wavesurfer.js` v7.12.12 + `waveform-data` | **BSD-3-Clause** | 波形/选区/频谱 UI 组件（v7 插件体系） |
| `tone` v15.1.22 | MIT | Transport/Part/Sequence/Sampler/GrainPlayer/效果器 + `OfflineAudioContext` 离线渲染 |
| `node-web-audio-api` v2.2.0（ircam） / `web-audio-api` v1.5.6 | BSD-3-Clause / MIT | Node 端 WebAudio → 服务端批量渲染 |
| `@tonejs/midi` 2.0.28 / `midi-writer-js` 3.2.1 / `@audio/midi` | MIT | MIDI 读写（可 diff 的真相源） |
| `vexflow` 5.0.0 MIT / `opensheetmusicdisplay` 2.1.2 BSD-3 / `abcjs` 6.7.0 MIT | — | 把分析结果画成乐谱 |
| `ag-psd`（无关人声）/ `mpg123-decoder`、`ogg-opus-decoder`、`@wasm-audio-decoders/ogg-vorbis` | MIT | 浏览器端多格式解码兜底 |

### 1.3 可选（模型辅助，**默认关闭**、需用户显式启用、进审计白名单）

`demucs` MIT（源分离）、`basic-pitch` Apache-2.0（音频→MIDI，替代纯 DSP 音符切分）、`marl/crepe` MIT（F0）、`openai/whisper` MIT（歌词/词级时间对齐）、`xiph/rnnoise` BSD-3（降噪）。
说明：这些是**判别式分析/分离模型**，不生成新音色、不替换说话人身份；仍默认关闭以守住"纯 DSP"承诺。

### 1.4 明确排除（许可或立场）

| 项 | 原因 |
|---|---|
| RVC（`rvc-web-runtime` 等）、so-vits-svc、Fish-Speech、ElevenLabs、`utaformatix`/DiffSinger、ACE-Step、YuE、Suno | **换声或生成式**——本子域立场排除（npm 上确有这些包，需在审计中显式拒绝） |
| `pitchfinder` v2.3.4（**GPL-3.0**，实测 npm license 字段）、`aubio`/`aubiojs`（GPL-3.0）、`essentia.js`（**AGPL-3.0**）、`TarsosDSP`（GPL）、`breakfastquay/rubberband`（GPL-2.0，`rubberband-wasm` 同） | 传染性许可，不进默认分发 |
| `soundtouchjs`（LGPL-2.1）、`peaks.js`（LGPL-3.0）、`lamejs`（LGPL-3.0） | 弱 copyleft，仅在"可替换"形态下可选；MP3 编码默认走可选外部 `ffmpeg`，交付默认 WAV/FLAC |
| Celemony Melodyne / Antares Auto-Tune / iZotope RX / Descript | 专有，仅作**交互与质量参照**，不引用代码 |
| WORLD（`mmorise/World`，GitHub 标 NOASSERTION） | 默认链不需要（`@audio/*` 已覆盖同能力）；如将来采用须先核实许可 |

### 1.5 取舍结论

**默认链 = `@audio/*`（MIT 原子） + `signalsmith-stretch`（MIT） + `@soundtouchjs/audio-worklet`（MPL-2.0） + Tone.js + wavesurfer.js + `@audio/assert`。**
工程建议：`@audio/*` 属 2026 年新发布、单一维护者、下载量百级 → **vendor 关键原子**（把 `pitch-yin`/`shift-psola`/`shift-formant`/`stretch-wsola`/`stretch-transient`/`tune-*`/`onset`/`beat`/`loudness` 源码拷入并保留 MIT 声明），锁定版本，避免上游漂移。

---

## 2. 六段链契约（导入 → 分析 → 处理 → 编排 → 渲染 → 导出）

每段给出：输入 → 输出（文本真相源）→ 实现 → 失败模式与处置。

### 2.1 导入（ingest）
- 输入：文件（wav/mp3/m4a/ogg/flac）、麦克风录音、或工作区已有音频。
- 处理：`@audio/decode`（Node）或 `AudioContext.decodeAudioData`（浏览器）→ 归一化为 48 kHz / 单声道（人声）+ 原始副本；计算 `sha256`、峰值/RMS/DC（`@audio/assert`）。
- 输出：`sources[]`（路径、hash、采样率、声道、时长）+ 波形峰值缓存（供 UI）。
- 失败模式：格式不支持 → 回退 wasm 解码器；削波/直流偏移 → 提示并给"修复"op（`denoise.declip` / DC blocker）。
- 可选：从立体声混音取回人声（`@audio/vocals` mid/side）；失败即提示改用可选模型通道。

### 2.2 分析（analyze）
- 任务（可单独触发、结果缓存到工程）：`f0`（YIN，`@audio/pitch-yin`，hop 可选 128/256/512）、`notes`（F0 + onset 分段 → 音符段：起始/时长/MIDI/cents/置信度）、`grid`（`@audio/beat`：bpm + 偏移 + 拍点表）、`slices`（`@audio/onset` 谱通量自适应峰值 → 音节/音素段）、`vad`（有声/无声/静音段）、`key`（`@audio/mir` chroma/调性）、`loudness`（`@audio/loudness` LUFS/true peak/LRA）。
- 输出：`analysis` 段（纯 JSON，帧数组可选用 base64 压缩或外部 `.f0.bin`）。
- 失败模式：F0 八度错误 → 提供"八度修正"op（对 F0 轨迹做 ±12 平移后重算音符）；弱能量段误检 → voicing 阈值可调。
- **实测要点（PoC 实跑，见 §7.0）**：`@audio/onset` 默认谱通量 + 自适应峰值对**含颤音的稳态音会过检**
  （2.2s / 3 段素材实测检出 18 个 onset vs 期望 3 个）→ 切片必须后处理：最小间隔合并、能量/VAD 门限、结合 F0 连续性；
  且 YIN 在本素材上有约 **-5.5 ¢** 的系统性偏差 → `tolerance`（默认 8¢）必须大于检测器系统偏差（建议 10–15¢），
  否则会把"本来就准的音"当跑调去修。

### 2.3 处理（process）
- 操作族（全部是"对用户素材的变换"）：
  - `pitch.snap{scale,root,strength}`：音阶吸附（Auto-Tune 类，`@audio/tune-snap`）
  - `pitch.target{notes|csv|midi}`：按目标音符/曲线重调（`@audio/tune-midi` + `@audio/shift-psola`）
  - `pitch.curve{points[],depth}`：手绘音高曲线（颤音/滑音/装饰音）
  - `pitch.transpose{semitones,formantKeep=true}`：整体移调（`shift-formant` / `shift-lpc`，男/女音域微调）
  - `time.warp{segments[{src,dst}]}`：时值/语速（`stretch-wsola` 用于说话、`stretch-psola` 用于歌声、`stretch-transient` 保护爆破音）
  - `time.quantize{grid,strength}`：对齐节拍网格（`beat` 的拍点）
  - `slice.cut{method:onset|vad|manual, minLen}` / `slice.resequence{order[]}` / `slice.loop{...}`：切片与重排
  - `speech.removeSilence{minGap,maxKeep}` / `speech.fillers{marks[]}`：停顿与填充词删减
  - `harmony.add{intervals[3,4,5,7], voices, detune, spread, humanize}`：和声/合唱叠加（多个 `shift-pvoc-lock`/`shift-formant` 副本 + 微小延迟与 detune + `spatial` 加宽）
  - `fx.*`：`eq` / `dynamics`（压缩/门/限幅）/ `denoise`（谱减/Wiener/dehum/declip）/ `reverb` / `saturate` / `spatial`
  - `gain.loudness{target:-14, standard:"BS.1770-4"}`：响度标准化
- 每个 op：确定性（固定 seed）、幂等、可撤销、可序列化（与 UI 拖拽共用同一命令模型）。
- 输出：`edits[]`（有序 op 列表）+ 每次应用后的 diff 摘要。

### 2.4 编排（arrange）
- 多轨：主唱（源素材）/ 和声层（同源变调副本）/ 切片层（采样器）/ 伴奏（用户提供文件，或 `Tone.js`/`@audio/synth` 合成器）。
- 时间线：拍网格（来自 `grid`）、段落标记（intro/verse/chorus…）、每轨的 op 与增益自动化。
- MIDI 驱动：`@audio/midi` 的 SMF 可直接驱动采样器与和声音高（"人声乐器化"）。
- 输出：`tracks[]` + `arrangement`（时间线、段落、自动化曲线）—— 皆为 JSON。

### 2.5 渲染（render）
- 离线确定性渲染：浏览器 `OfflineAudioContext`（`@audio/wam` 把原子挂为 worklet 节点）或 Node 端 `node-web-audio-api`（服务端批量）。
- 目标：`mix`（混音）/ `stems`（分轨）/ `midi`（符号）/ `score`（乐谱图）/ `preview`（低码率试听）。
- 输出：音频文件 + **`provenance`**（sourceMap + 渲染参数 + 确定性声明）+ `recipe`（处理链清单）。

### 2.6 导出（export）
- 格式：WAV（24-bit 默认）/ FLAC；MP3/M4A 走可选外部 `ffmpeg`；MIDI（SMF）；MusicXML/ABC；工程 JSON；可选视频（波形/乐谱动画，`ffmpeg`）。
- 同时导出 `voice.project.json`（可 diff 真相源）+ `.f0.bin` 等大数组旁挂文件 + `checks.json`（§7 验收结果）。
- 失败模式：响度不达标 → 阻断并给修正建议；provenance 缺失 → 视为渲染失败（本子域不允许无来源产物）。

---

## 3. 三条创作闭环（"能创作"的实质）

### 闭环 A：音高-时间编辑（Melodyne 类）
`F0 曲线 / 音符块可视化编辑` → `edits: pitch.snap | pitch.target | pitch.curve | time.warp`
→ `PSOLA/pvoc 重调 + 与干声交叉淡化` → 试听/渲染。
UI：波形 + F0 曲线叠加（拖拽=整体移调；逐音符块=单音吸附/微调 cents；颤音深度/滑音时间可调）。
**这是"修音 + 重唱式微调"，音色 100% 来自用户**（PSOLA 修改激励而非替换声道）。

### 闭环 B：人声 → MIDI → 人声（用自己声音唱新旋律）
`onset + F0 → 音符段 → MIDI（文本真相源）` → `用户在 MIDI 里改旋律/时值/换调`
→ `pitch.target{notes: 编辑后的 MIDI}` 用 PSOLA 把人声重调到新旋律 → 渲染。
全程无模型；MIDI 可 diff、可 apply_patch、可版本管理——符合总方案"文本真相源"铁律。
限制：只能重排/重调**已有音素**（唱不出发音表外的词），超出即提示。

### 闭环 C：切片编排（人声乐器化）
`onset/VAD 自动切片 + 人工修边` → `slices[]`（含时长/音高/标签）→ `网格对齐` → `Sampler/GrainPlayer 回放重排 / loop`
→ MIDI 驱动精确节奏 → 渲染分轨。
典型产物：vocal chop、ad-lib 叠层、节奏化念白、采样器化的"人声鼓组"。

---

## 4. 工程真相源 `voice.project.json`

```jsonc
{
  "version": 1,
  "sources": [{ "id": "v1", "path": "assets/take1.wav", "sha256": "…", "sr": 48000, "ch": 1 }],
  "analysis": {
    "f0":     { "algo": "yin@1.0.5", "hop": 256, "file": "cache/v1.f0.bin", "confidence": 0.91 },
    "notes":  [{ "t": 1.234, "dur": 0.512, "midi": 57, "cents": -12, "conf": 0.93 }],
    "grid":   { "bpm": 96, "offset": 0.120, "beats": [0.12, 0.745, 1.37] },
    "slices": [{ "a": 1.230, "b": 1.402, "label": "ka", "midi": 57 }],
    "vad":    [{ "a": 0.0, "b": 1.1, "voiced": true }],
    "key":    { "root": 9, "mode": "minor", "conf": 0.78 },
    "loudness": { "lufs": -19.2, "truePeak": -2.1, "lra": 6.4 }
  },
  "edits": [
    { "op": "speech.removeSilence", "minGap": 0.35, "keep": 0.08 },
    { "op": "pitch.snap", "scale": "minor", "root": 9, "strength": 0.85 },
    { "op": "time.warp", "segments": [{ "src": [1.23, 1.40], "dst": 0.25 }] },
    { "op": "harmony.add", "intervals": [3, 4, 7], "voices": 3, "detune": 6, "spread": 0.35, "humanize": 0.02 },
    { "op": "fx.eq", "stages": [{ "type": "highpass", "f": 90 }] },
    { "op": "gain.loudness", "target": -14, "standard": "BS.1770-4" }
  ],
  "tracks": [{ "id": "lead", "source": "v1", "edits": "inherit" }, { "id": "harm", "from": "lead", "edits": "harmony.add@3" }],
  "arrangement": { "sections": [{ "name": "verse", "t": 0, "len": 16 }], "automation": [] },
  "render": { "engine": "offline", "sr": 48000, "bitDepth": 24, "targets": ["mix", "stems", "midi"] },
  "provenance": { "deterministic": true, "seed": 0, "sourceMap": "out/sourcemap.bin", "recipe": "out/recipe.json" }
}
```

设计要点：
1. **analysis 与 edits 分离**——重分析不丢编辑；edits 可重放（渲染 = 源 + edits 的纯函数）。
2. **edits 即命令模型**——UI 拖拽与 Agent 工具写的是同一种 op（总方案 §3 铁律 3）。
3. **provenance 是一等公民**——本子域的"非换声/非生成式"承诺靠它落地（§7）。
4. 大数组（F0 帧、sourceMap）旁挂二进制文件，JSON 保持可 diff。

---

## 5. 插件契约

### 5.1 host 工具

| 工具 | 输入 | 输出 | 说明 |
|---|---|---|---|
| `voice_import` | `{ path \| url \| mic, analyze?: [] }` | `{ sourceId, meta, warnings }` | 导入后可自动跑分析 |
| `voice_analyze` | `{ project, source, tasks: [f0,notes,grid,slices,vad,key,loudness] }` | 分析结果摘要 + 工程 diff | 结果缓存，重复调用幂等 |
| `voice_edit` | `{ project, ops: [...] }` | `{ applied, diff, warnings }` | 校验 op 合法性与参数范围 |
| `voice_render` | `{ project, targets: [mix,stems,midi,score] }` | `{ files, provenance, checks }` | 离线确定性渲染 |
| `voice_export` | `{ project, format }` | `{ files }` | WAV/FLAC/MIDI/MusicXML/ABC（MP3 需 ffmpeg） |
| `voice_verify` | `{ project, output }` | `{ pass, findings[], metrics }` | **执行 §7 六项专项验收**，可对任意渲染结果调用 |

### 5.2 ui 插件（槽位：`editor` / `right-panel` / `chat-tools`）

- 波形 + F0 曲线编辑视图（`wavesurfer.js` 波形 + 自绘 F0 叠加与音符块）。
- 音符网格（音高-时间）视图：拖移/伸缩/cents 微调/量化。
- 切片视图：自动标记 + 拖边 + 重排序列 + 试听。
- 多轨编排视图：主唱/和声/切片/伴奏 + 段落标记 + 增益自动化。
- 效果机架：eq/dynamics/denoise/reverb/saturate/spatial 参数即 op（改动写入工程）。
- 会话内嵌渲染（总方案 E1）：` ```vocal `（波形 + F0 静态图）、` ```harmony `（和声结构图）、渲染后自动附 A/B 差分波形。
- 说话场景专用：停顿板（一键删停顿，`speech.removeSilence` 可视化预览）、语速曲线、填充词标记（可选 ASR）。

---

## 6. 两套场景预设

| 维度 | 说话（speech） | 歌声（singing） |
|---|---|---|
| 主算法 | `stretch-wsola`（时域、低 CPU、适合语音）+ `shift-psola` | `shift-psola` / `shift-pvoc-lock` + `shift-formant` |
| 核心编辑 | 停顿删减、语速曲线、填充词、去噪/去嗡、响度（-16 LUFS 播客 / -14 通用） | 音准吸附、颤音/滑音、音域微调、和声、切片与 loop |
| 保护重点 | 爆破音/齿音（`shift-transient`、EQ 去齿音、declick） | 共振峰（`shift-formant`/`shift-lpc`）、气声、尾音 |
| 典型失败 | 时域算法把爆破音"抹平" | 纯 pvoc 变调导致共振峰偏移（"花栗鼠"） |
| 交付 | WAV 单声道/立体声 + 可选字幕 | 混音 + 分轨（干声/和声/伴奏）+ MIDI |

---

## 7. 验收：**非换声 / 非生成式**专项（机制保证，非口头承诺）

> 六项全部可自动化，由 `voice_verify` 与 CI 执行；任一项失败 = 本子域的承诺被破坏，构建/渲染失败。

| # | 检查 | 判定方式 | 阈值（默认） | 对应边界 |
|---|---|---|---|---|
| 7.1 | **依赖审计** | `scripts/check-voice-toolchain.mjs` 扫依赖树与工作区，匹配禁止清单（rvc/so-vits/diff-singer/fish-speech/elevenlabs/suno/ace-step/yue/utaformatix/tts/voice-conversion）+ 禁止权重后缀（`.pth/.onnx/.ckpt/.safetensors/.gguf`） | 命中数 = 0（除非在显式 optional 白名单） | 不生成/不换声 |
| 7.2 | **帧级可追溯** | 渲染输出携带 `sourceMap`；用 sourceMap 从源重建输出，与真实输出做逐段互相关 | 相关系数 ≥ 0.99 | 不无源合成 |
| 7.3 | **音色一致性** | 输出 vs 源的低次谐波包络比（H2/H1、H3/H1）与分段谱包络相关性（**不要用全谐波偏差**，见 §7.0 实测） | 低次谐波偏差 ≤ 10%（实测 4.2%）、H3 ≤ 15%；H4/H5 允许 25–45%（PSOLA 时域特性） | 不换声 |
| 7.4 | **确定性可复现** | 同 `project` + 同源 + 固定 seed 渲染两次，比较逐样本差 | **位级一致（实测 maxDiff = 0）**；若实现引入并行浮点，放宽至 ≤ -180 dBFS | 不生成（生成式无法位级复现） |
| 7.5 | **链可见** | `recipe.json` + `project.edits` 覆盖全部处理阶段（来自 `@audio/chain` 的可见 recipe），禁止未登记的黑箱步骤 | 每个 stage 有算法名与参数 | 不黑箱 |
| 7.6 | **无新增音源** | 输出中除源素材外不得存在独立音源（互相关/指纹 + 人工抽检；和声/切片层必须可由源帧映射解释） | 无未解释音源 | 不无源合成 |

#### 7.0 原型实测基准（2026-09，脚本 `docs/research/voice-poc/poc.mjs`，输出 `docs/research/voice-poc/poc-report.txt`）

脚本：合成 2.2 s 单声道"人声样"素材（3 个有声段 220 / 246.94 / 289.20 Hz，5.5 Hz 颤音，5 次谐波堆），
用本方案内核实跑六环。**实测结果即阈值标定依据**：

> 复跑方式：`mkdir -p _temp/voice-poc && cd _temp/voice-poc && npm i @audio/pitch-yin @audio/tune-snap @audio/loudness @audio/onset && cp ../../docs/research/voice-poc/poc.mjs . && node poc.mjs`

| 环节 | 实测 |
|---|---|
| 1 分析 | YIN 检出 219.27 / 246.16 / 288.31 Hz（各 25 帧），相对合成目标 **-5.8 / -5.5 / -5.3 ¢**（系统性偏低） |
| 2 处理 | `snap(major, root=C)`：段1/段2 在 `tolerance` 内 **不动（+0.0 ¢）**；段3 **+24.7 ¢**（288.31 → 292.44，目标 D4=293.66，残留 ≈ -7 ¢）。耗时 361 ms / 2.2 s 单声道（单线程 Node，≈0.16× 实时） |
| 3 音色 | 谐波包络比偏差：H2/H1 **4.2%**、H3 11.8%、H4 24.1%、H5 43.6% → PSOLA 保低次谐波、改高次谐波（阈值据此设定，见 7.3） |
| 4 确定性 | 两次渲染逐样本 **maxDiff = 0（位级一致）** |
| 5 响度 | LUFS -16.41 → -16.40；true peak -9.87 → -9.87（吸附几乎不改响度） |
| 6 切片 | 谱通量 onset **18 个（期望 3）** → 默认 ODF 峰值不可直接采信（见 §2.2 实测要点） |

**三条由实测得出的工程结论**（实现时必须遵守，否则会踩坑）：
1. **切片需后处理**：ODF 峰值 ∩ 能量/VAD 门限 ∩ F0 连续性，并合并最小间隔（默认 60–120 ms）。
2. **音色一致性判据必须分层**：低次谐波（音色主体）严格，高次谐波放宽；用 LPC/MFCC 全谱偏差会误报。
3. **音准精度需要二次迭代**：段中位数吸附后残留 ≈ 7 ¢ > 人耳可辨阈值（≈5 ¢）→ 吸附后用逐帧曲线目标做二次精修，
   或把 `strength=1` 的段级吸附改为"段级 + 逐帧微调"两级。

通用验收（与其他域一致）：
- 产物可解析：WAV 头/采样率/帧数正确；SMF 可被 `@tonejs/midi` 解析出音符数 > 0；乐谱 XML/ABC 可被 VexFlow/OSMD/abcjs 渲染。
- 视觉校验：波形/F0/乐谱截图 + `read_image` 判定（复用 `tool-web`）。
- 响度交付：`@audio/loudness` 测得 LUFS 与目标偏差 ≤ 0.5，true peak ≤ -1 dBTP。
- 回归：`@audio/assert` 的 golden regression——固定素材 + 固定工程 → 输出信号特征（峰值/RMS/音高序列）与基线一致。

---

## 8. 质量上限与风险（诚实标注）

1. **变调范围**：PSOLA/pvoc 在 ±2 半音内几乎无瑕；±5 半音可接受；**>±7 半音显著劣化**（机械感/气声异常）；>±12 不可用 → UI 必须提示，不允许静默"硬做"。
2. **时值极限**：说话人声拉伸超过约 1.5× 到歌唱时值会出现"颤动/金属感"（WSOLA 更明显，PSOLA 稍好）；持续音延长 >2× 需改用切片循环并提示。
3. **清辅音/爆破音**：时域算法易失真 → 必须用 transient-aware 算法保护 attack；齿音需 EQ/去齿音。
4. **混响尾音**：切片会"带尾" → 优先 dry 分离或最短切片策略；素材混响过重时质量下降明显。
5. **从混音取回人声**：`@audio/vocals`（mid/side）仅对居中人声有效，侧向乐器/立体声混响无法去除；需要更好分离时进入**可选模型通道**（demucs），并须在 UI 明确标注"已启用模型辅助（分析/分离）"。
6. **上游风险**：`@audio/*` 为 2026 年新发布、单维护者、下载量百级 → **vendor 关键原子 + 锁版本 + 保留 MIT 声明**；`signalsmith-stretch` 仅有 wasm 产物，注意构建链。
7. **法律与伦理**：声音权/肖像权责任归属用户；本子域不提供换声能力（降低"模仿他人声音"的滥用面）；输出记录 provenance 供追溯；伴奏音源（soundfont）与素材授权须提示。
8. **性能**：F0/PSOLA 在 48 kHz 立体声长素材（>10 分钟）需分块 + Worker/AudioWorklet；服务端批量渲染走 `node-web-audio-api`。
9. **吸附精度**：`tune-snap` 的段级（段中位数）吸附实测残留 ≈ 7 ¢，且不处理音符内滑音/颤音曲线（v1 仅段级）；
   高质量需求需在此之上做**逐帧曲线精修**（`pitch.curve` op），并在 UI 标注"段级吸附 / 曲线精修"两档。

---

## 9. 分期（人声子域）

### V-P0（最小可用：修音 + 时值 + 可验证）
1. `voice_import`（文件 + 麦克风录音）+ 波形/峰值缓存。
2. `voice_analyze`：`f0` / `notes` / `grid` / `vad`（YIN + onset + beat + 能量/谱平坦度）。
3. `voice_edit`：`pitch.snap`、`pitch.target(notes)`、`time.warp`（WSOLA/PSOLA）、`speech.removeSilence`。
4. `voice_render`（浏览器离线）+ `voice_export`（WAV/MIDI）。
5. **`voice_verify` 最小集：§7.1 依赖审计 + §7.4 确定性 + §7.2 sourceMap 重建**（三项即足以支撑"非换声非生成式"承诺）。
6. UI：波形 + F0 曲线最小编辑器；会话内嵌 ` ```vocal `（静态波形+F0 图）。

验收（阈值取 §7.0 实测标定值）：给一段用户录音 → （a）删停顿并缩短 15% → （b）按 C 小调吸附 → 渲染 WAV；
`voice_verify` 六项中 1/2/4 项 PASS；响度达标（LUFS 偏差 ≤ 0.5、true peak ≤ -1 dBTP）；
F0 前后对比图经 `read_image` 判定吸附方向与幅度正确；两次渲染**位级一致（maxDiff = 0）**；依赖审计 0 命中；
吸附后残留 ≤ 5 ¢（需二次精修，见 §8.9）。

### V-P1（创作能力铺开）
7. 人声 → MIDI → 人声闭环（`@audio/midi` 写 SMF + `pitch.target` 重调）。
8. 切片（onset/VAD + 人工修边）+ 重排 + 采样器/loop（Tone.Sampler）。
9. 和声/合唱叠加（多副本变调 + detune/spread/humanize）。
10. 效果机架（eq/dynamics/denoise/reverb/saturate/spatial）+ 响度交付（BS.1770-4）。
11. 服务端批量渲染（`node-web-audio-api`）+ FLAC/多轨导出 + 乐谱导出。
12. `voice_verify` 全六项 + `@audio/assert` golden 回归接入 CI。

### V-P2（高级与可选通道，默认关闭）
13. 可选模型辅助：demucs 分离、basic-pitch 转 MIDI、whisper 歌词对齐（UI 明确标注 + 审计白名单）。
14. 音素级切片标签（多语言）与切片重排的"歌词驱动"编排。
15. MP3/M4A/视频导出（外部 ffmpeg）；波形/F0 动画可视化。

---

## 10. 事实来源（本子域，2026-09 实检索）

- npm registry（`registry.npmjs.org`，逐包查询 version/license/type/时间）与下载量 API（`api.npmjs.org/downloads/point/last-month/*`）：
  `@audio/*` 共 40+ 包全部 MIT（实测：decode 3.15.0、encode 1.6.5、mic 1.1.3、pitch-yin 1.0.5、mir-melody 1.1.3、onset 1.0.1、beat 2.1.3、vad 1.0.1、tune-snap 1.1.3、tune-midi 1.0.2、shift 1.1.4、shift-psola 1.0.3、shift-pvoc-lock 1.0.2、shift-formant 1.1.4、shift-lpc 1.0.2、stretch 2.0.5、stretch-wsola 1.2.1、stretch-psola 1.2.1、stretch-transient 1.2.1、spectral 1.2.4、stft 1.0.6、sinusoidal 1.0.1、auditory 1.0.2、resample 1.2.2、eq 1.0.3、effect 2.1.5、dynamics 0.2.6、reverb 1.0.2、saturate 1.0.3、filter 3.2.1、denoise 0.3.9、spatial 1.0.3、loudness 1.2.1、chain 0.1.0、assert 0.2.0、midi 1.0.3、note 1.0.2、mir 1.1.3、synth 1.2.2、voice 1.0.2、vocals 1.0.5、wam 0.1.2)；
  `signalsmith-stretch` 1.3.2 MIT；`@soundtouchjs/*` MPL-2.0；`@echogarden/sonic-wasm` 0.2.0 Apache-2.0；
  `wavesurfer.js` 7.12.12 BSD-3；`peaks.js` 4.0.0 LGPL-3.0；`pitchfinder` 2.3.4 **GPL**；`essentia.js` 0.1.3 **AGPL-3.0**；
  `aubio` GPL-3.0；`rubberband-wasm` GPLv2；`soundtouchjs` LGPL-2.1；`lamejs` LGPL-3.0；`node-web-audio-api` 2.2.0 BSD-3；`web-audio-api` 1.5.6 MIT。
- GitHub REST API（stars/license/pushed）：`esp` 实测——`xiph/rnnoise` BSD-3（5839★）、`snakers4/silero-vad` MIT（10217★）、`facebookresearch/demucs` MIT（10366★）、`spotify/basic-pitch` Apache-2.0（5573★）、`marl/crepe` MIT（1415★）、`librosa/librosa` ISC（8612★）、`YannickJadoul/Parselmouth` GPL-3.0（1287★）、`MTG/essentia.js` AGPL-3.0（868★）、`breakfastquay/rubberband` GPL-2.0（780★）、`mmorise/World` NOASSERTION（1340★）。
- 上游页面实抓：`github.com/audiojs/tune`（`snap(data,{scale,root})` / `tuneMidi(data,{guide})`，实现为 pitch-yin → 吸附 → shift-psola 交叉淡化，对标 Logic Pitch Correction / Ableton tuner）、`github.com/orgs/audiojs/repositories`（71 仓库全清单，含 chain/assert/loudness/denoise/wam/neural 等）。
- 排除项实证：npm 上存在 `rvc-web-runtime`、`mcp-server-musicgpt`（含 voice conversion）、`utaformatix-data` 等换声/生成式包 —— 审计清单据此编写。
- **本地 PoC 实证**（2026-09，可复跑）：`docs/research/voice-poc/poc.mjs` + `poc-report.txt`（依赖 `@audio/pitch-yin` 1.0.5 /
  `@audio/tune-snap` 1.1.3 / `@audio/loudness` 1.2.1 / `@audio/onset` 1.0.1，`npm i` 共 25 包，9 s 装完）——
  六环实跑结果见 §7.0，验证了"分析 → 音阶吸附 → 音色保持 → 位级可复现 → 响度计量 → 切片"这条纯 DSP 链在本环境可用。

**待核实项**：`TarsosDSP` 许可（GitHub API 限速未取到）；`@audio/*` 各仓库根 LICENSE 文件与 npm `license: MIT` 字段的逐包一致性（落地前逐包核对）；`signalsmith-stretch` 的 wasm 构建链与体积。
