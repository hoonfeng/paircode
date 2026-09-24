# @paircode/tool-voice

人声创作工具面（说话 / 歌声）—— **纯 DSP**，不含换声模型、不含生成式模型、不含 TTS。

> PairCode 官方插件（`keywords: paircode`）。运行时轨：**Node 桥**（声明了 `@audio/*`
> 运行期依赖 → 安装走 `.pair/cordis/node/` + 真实 `npm install` + `bridge.js` 装载）。

## 定位：不"造声音"，只编辑用户自己的声音

| 边界 | 机制保证（可自动验收） |
|---|---|
| 不换声 | 7.3 音色一致性：低次谐波包络比 H2 ≤ 10%、H3 ≤ 15%（PSOLA 改激励不改声道）；7.2 sourceMap 帧级可追溯 |
| 不生成 | 7.1 依赖审计（禁止清单 + 模型权重后缀零命中）；7.4 两次渲染**位级一致**（生成式模型无法复现） |
| 不无源合成 | 7.6 输出有声帧 100% 可由源帧解释（按 sourceMap 反查）；7.5 recipe 链可见 |

能力上限（诚实声明）：只能修正、重组、编排**用户已经唱/说出来的声音**（音准、时值、
音域微调、停顿删减）；不能让用户唱出从未发出的音素，也不能把男声"变成"女声。

## 工具

| 工具 | 作用 | 要点 |
|---|---|---|
| `voice_import` | 导入素材（WAV）到 `voice.project.json` | 解析 PCM u8/i16/i24/i32 与 float32/64；登记 sha256/峰值/RMS/DC；可自动分析 |
| `voice_analyze` | 分析：`f0` / `notes` / `vad` / `slices` / `loudness` | 结果幂等；F0 轨迹旁挂 JSON（工程可 diff） |
| `voice_edit` | 追加编辑命令 | `pitch.snap` / `time.warp` / `speech.removeSilence` |
| `voice_render` | 渲染（源 + edits 的纯函数） | 产出 `.wav` + `.sourcemap.json`（输出↔源时间映射）+ `.recipe.json`（算法/版本/参数） |
| `voice_verify` | 六项专项自检 | 可对任意渲染结果调用 |

### 编辑命令

```jsonc
{ "op": "pitch.snap", "scale": "minor", "root": 0, "strength": 1, "tolerance": 12 }
// 音阶吸附（Auto-Tune 类，PSOLA）。tolerance 默认 12 音分 —— 必须大于 YIN 检测器
// 的系统性偏差（实测 -5 ¢ 量级），否则会把"本来就准的音"当跑调去修。
{ "op": "time.warp", "factor": 0.85 }          // WSOLA 时值伸缩（缩短 15%）
{ "op": "speech.removeSilence", "minGap": 0.35, "keep": 0.08 }   // 删停顿，保留呼吸边距
```

## 依赖（全部 MIT）

`@audio/tune-snap` · `@audio/stretch-wsola` · `@audio/pitch-yin` · `@audio/onset` ·
`@audio/loudness` · `@audio/note` —— 版本在 `package.json` 中精确锁定。

## 典型链路

```bash
# 1) 导入（相对主项目根） → 自动分析
voice_import { "path": "assets/take1.wav" }
# 2) 编辑：删停顿 → 缩短 15% → 按 C 小调吸附
voice_edit { "ops": [ {"op":"speech.removeSilence","minGap":0.3,"keep":0.05},
                      {"op":"time.warp","factor":0.85},
                      {"op":"pitch.snap","scale":"minor","root":0} ] }
# 3) 渲染 + 自检
voice_render {}
voice_verify {}
```

## 判据口径（避免误读）

- **7.2 帧级可追溯**：sourceMap 是分段线性段表（输出↔源时间）。判据分层：
  *保波形链*（切片/停顿删减等样本级搬运）断言**波形级**逐段归一化互相关 ≥ 0.99；
  *含改波形 op 的链*（变调/伸缩会改变周期结构与相位）断言**结构级**：包络相关 ≥ 0.95
  且每个有声段包络相关 ≥ 0.9。波形级数值一并如实上报，但不作断言 —— 对 PSOLA
  输出要求逐样本相关是物理上不成立的。
- **7.3 音色一致性**：只对低次谐波严格（音色主体）。高次谐波在时域重叠相加算法下
  本就改变（实测 H4/H5 约 24%/44%），对其设严阈值会误报。

## 形态说明（开发者）

- 本插件依赖真实 Node（`Buffer`、`node:fs`、ESM 的 `@audio/*`）。在**磁盘扫描轨**
  （`.pair/plugins/` 被当作 goja 沙箱插件装载）会在 `apply()` 首行做运行时自检并
  **显式让位**（不注册任何工具）——避免"装上但半死"的工具面污染。市场安装会自动
  判定为 Node 桥轨（`dependencies` 非空）。
- 工具在装载后仍受工作区**工具集白名单**收敛：需要把 `voice_*` 加入当前工具集，
  agent 才可见（cordis/插件面板始终可见）。

## 许可

MIT。`@audio/*` 均为 MIT（audiojs 组织）。
