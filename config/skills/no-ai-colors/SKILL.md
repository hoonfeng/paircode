---
name: no-ai-colors
description: 前端和GUI开发时禁止使用AI生成的配色方案，应使用设计系统或手动设计的颜色
activation: auto
globs: "*.css *.scss *.less *.tsx *.jsx *.html *.vue *.svelte *.ts *.js"
version: 1.0.0
---

# 禁止使用 AI 配色

## 原则
在前端和 GUI 开发中，**禁止使用 AI 模型生成的颜色/配色方案**（包括但不限于：主题色、背景色、文字色、边框色、渐变、阴影等）。AI 生成的配色往往存在以下问题：

1. **缺乏品牌一致性** — 不符合品牌设计系统的规范
2. **视觉不协调** — AI 生成的色彩组合缺乏专业设计师的调优
3. **可访问性差** — 容易忽略 WCAG 对比度标准，导致文字难以阅读
4. **风格飘忽不定** — 每次生成结果不一致，无法形成统一风格

## 应该怎么做

### 1. 使用设计系统（Design System）的颜色
- 优先使用项目中已有的设计系统/主题变量（如 Tailwind 的 `theme.colors`、Material Design 的调色板、CSS 自定义属性等）
- 如果项目有设计稿（Figma/Sketch），严格按照设计稿的颜色取值
- 使用品牌色板中的固定色值

### 2. 手动选择颜色时的准则
- 遵循 WCAG 2.1 AA 标准（普通文本对比度 ≥ 4.5:1，大文本 ≥ 3:1）
- 主色选择 1-2 种，辅助色 2-3 种，中性色（灰阶）5-8 种
- 使用 HSL 调色：固定色相(H)，调整饱和度(S)和亮度(L)生成同色系变体
- 语义色：成功绿、警告橙、错误红、信息蓝 — 使用行业通用色值

### 3. 配色辅助工具（推荐使用，而非 AI）
- **调色板生成**: Coolors.co, Paletton.com
- **对比度检查**: WebAIM Contrast Checker, Stark 插件
- **渐变色**: CSS Gradient, UI Gradients
- **Material Design 调色板**: material.io/design/color

### 4. 常见场景的颜色规范

| 场景 | 推荐方案 |
|------|---------|
| 品牌主色 | 从品牌 Logo 或设计稿提取，使用 `hsl()` 定义 |
| 文字色 | 主文字 `hsl(0, 0%, 13%)`（接近 #222），次要文字 `hsl(0, 0%, 45%)` |
| 背景色 | 白色 `#fff`、浅灰 `hsl(0, 0%, 96%)`、深色模式使用设计系统变量 |
| 边框色 | `hsl(0, 0%, 85%)` 或使用主题变量 |
| 链接色 | 从品牌色推导，确保对比度 ≥ 4.5:1 |
| 成功/错误 | 绿色 `#22c55e`、红色 `#ef4444` 等标准色 |

## 示例

### ❌ 错误做法（AI 生成的随机配色）
```css
:root {
  --primary: #7C3AED;    /* AI 随机生成的紫色 */
  --secondary: #F59E0B;  /* 与主色不协调 */
  --background: #F8FAFC; /* 没有考虑设计系统 */
}
```

### ✅ 正确做法（使用设计系统 / 手动调色）
```css
:root {
  /* 从品牌色板提取 */
  --primary: #2563EB;        /* 品牌蓝 */
  --primary-light: #60A5FA;  /* 同色系变体 */
  --primary-dark: #1D4ED8;
  --secondary: #059669;      /* 品牌绿 */
  --neutral-50: #F8FAFC;
  --neutral-300: #CBD5E1;
  --neutral-900: #0F172A;
  /* 语义色 — 使用标准色值 */
  --success: #22C55E;
  --warning: #F59E0B;
  --error: #EF4444;
  --info: #3B82F6;
}
```
