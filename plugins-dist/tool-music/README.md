# tool-music — 音乐创作工具面（纯 goja 零依赖）

音乐域的 **host 半**：把「乐曲」表示成可 diff 的**文本真相源**（工程 JSON），MIDI / 乐谱只是由它生成的产物。

- 形态：**磁盘 goja 轨插件**（零 npm 依赖；沙箱内无 `require`，故单文件自包含）
- 二进制安全：MIDI 读写走宿主 `ctx.fs.readFileBase64` / `writeFileBase64`
- 配套 UI 半：`ui-music`（乐谱渲染 + 播放器）
- 依赖第三方库：**无**（SMF/ABC/MusicXML/SVG 全部自实现，许可面干净）

## 工具面

| 工具 | 作用 |
| --- | --- |
| `music_project` | 工程管理：`mode=create/show/update`（标题/速度/拍号/调号/轨道定义） |
| `music_edit` | 命令式编辑：note.add/remove/move/transpose/quantize/velocity/set、chord.add（14 种性质）、scale.add（13 种音阶）、track.add/remove/rename/instrument/repeat、set.tempo/meter/key |
| `music_import` | 导入 MIDI(SMF) / ABC(子集) / MusicXML(partwise) → 工程 JSON |
| `music_export` | 导出 MIDI(SMF fmt1) / ABC / MusicXML 3.1 / SVG 简易乐谱 |
| `music_verify` | 7 项判据自检（见下） |

## 工程格式（真相源，可 diff）

```json
{
  "format": "paircode.music.project/1",
  "meta": { "title": "验证曲", "createdAt": "…", "updatedAt": "…" },
  "tempo": 100,
  "meter": [3, 4],
  "key": "G",
  "ppq": 480,
  "tracks": [
    { "id": "t1", "name": "主旋律", "channel": 0, "program": 0,
      "notes": [ { "start": 0, "dur": 480, "pitch": 60, "vel": 90 } ] }
  ],
  "artifacts": { "midi": "music.mid", "svg": "music.score.svg" }
}
```

**音符时间一律是整数 tick**（`ppq` 默认 480）：避免浮点漂移、保证位级确定性，也让 `git diff` 可读。

## 校验判据（`music_verify`）

| # | 判据 | 说明 |
| --- | --- | --- |
| V1 | 音符合法 | pitch 为整型且在 `[0,127]` |
| V2 | 时值合法 | `start ≥ 0`、`dur > 0`，均为整型 tick |
| V3 | 轨道合法 | id 唯一、通道 `[0,15]`、音色 `[0,127]`、每条轨道非空 |
| V4 | **SMF 回读一致** | 导出 MIDI → 重新解析 → 音符（含 vel）/速度/拍号/ppq/**轨道通道与音色**逐字段一致 |
| V5 | **确定性** | 同一工程两次导出 MIDI **位级一致**（不含时间戳等易变字段） |
| V6 | **文本真相源往返** | ABC 导出 → 导入 → 音符集合（start/dur/pitch）一致（含和弦 `[CEG]` 与休止符） |
| V7 | MusicXML 结构合法 | `<part id="…">` 与 `</part>` 配对且数量等于轨道数 |

V4/V5/V6 是本插件的核心契约：**文本真相源 ⇄ 二进制产物 ⇄ 图形产物三者必须一致**；任一 FAIL 视为构建失败。

## 用法示例

```js
// 1. 建工程（含一条轨道）
music_project { mode: "create", path: "music.project.json", title: "小星星", tempo: 100, meter: [4,4], key: "C",
                tracks: [{ id: "t1", name: "主旋律", program: 0 }] }
// 2. 写音符 / 和弦 / 音阶
music_edit { ops: [
  { op: "note.add", track: "t1", start: 0, dur: 480, pitch: 60 },
  { op: "chord.add", track: "t1", start: 960, dur: 960, root: "C4", quality: "maj7" },
  { op: "scale.add", track: "t1", start: 1920, stepDur: 240, steps: 8, root: "G4", scale: "major" },
  { op: "track.repeat", id: "t1", times: 2 }
] }
// 3. 导出 + 校验
music_export { format: "midi" }        // → music.mid（二进制安全写入）
music_export { format: "svg" }         // → music.score.svg（可截图给模型「看」）
music_verify { }                       // → 7 项判据报告
```

## 已知限制

- ABC 解析覆盖**子集**：`X/T/M/L/Q/K` + 声部 `V:`、音名（`^`/`_`/`=`、八度撇逗）、时值（整数与 `a/b` 分数）、和弦 `[...]`、休止 `z`；不支持 `%%` 指令、装饰音、连音线语义。
- ABC 往返**不保留轨道归属**（各声部合并比对音符集合）。
- MIDI 导出为 **format 1**；不支持 SMPTE 时间基（`division` 高位为 1 时明确报错）；不导出歌词/tempo 曲线（单速度）。
- MusicXML 为 **partwise** 简化实现（无 `<backup>/<forward>` 高级用法）；导入按声明顺序还原音符。
- SVG 乐谱为**简易高音谱表**（不加区分谱号/调号符号），定位用途是「让模型能看图核对音高与节奏」，不是排版级乐谱。

## 本地验证

```bash
node --check plugins-dist/tool-music/index.js      # 语法
node _temp/tool-music-test.cjs                     # 全链路：建→编→导（4 格式）→校验→回读→边界拒绝
```
