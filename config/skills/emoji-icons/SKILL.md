---
name: emoji-icons
description: 禁止使用 Emoji 作为图标，应使用 SVG/图标组件库
activation: auto
globs: "*.css *.scss *.less *.tsx *.jsx *.html *.vue *.svelte *.ts *.js"
version: 1.0.0
---

# 禁止使用 Emoji 作为图标

## 原则
在 UI 开发中，**禁止使用 Emoji（表情符号）替代正式图标**。Emoji 在不同平台、浏览器和设备上的渲染效果差异巨大，且缺乏图标的精确性、一致性和可访问性。

## 为什么禁止

1. **跨平台不一致** — 同一 emoji 在 Windows、macOS、iOS、Android、Linux 上渲染完全不同，甚至同一个平台不同版本也不一样
2. **无像素级控制** — Emoji 是字体字符，无法精确控制颜色、大小、描边、对齐等样式细节
3. **可访问性问题** — 屏幕阅读器会将 emoji 作为文字读出而非功能性图标，干扰视障用户
4. **缺乏语义** — Emoji 含义模糊，不同文化背景的用户理解不同，不如图标直观
5. **显示延迟** — Emoji 字体文件加载有延迟，可能出现空白方块（□）或错误的渲染
6. **无法响应式调整** — 图标需要支持多种尺寸（16px~64px），emoji 在不同字号下可能被裁切或模糊
7. **不支持交互状态** — 无法实现 hover、active、disabled 等交互状态的样式变化

## 应该怎么做

### 1. 使用 SVG 图标
```tsx
// ✅ 正确：使用 SVG 图标组件
import { SearchIcon, CloseIcon } from '@components/icons';

<Button>
  <SearchIcon size={16} />
  搜索
</Button>
```

### 2. 使用图标组件库
- **Lucide Icons** — 轻量、Tree-shakable、React/Vue 适配
- **Phosphor Icons** — 多风格（填充/线框/双色），6000+ 图标
- **Heroicons** — Tailwind 官方推荐，简单统一
- **Remix Icon** — 2700+ 开源图标，风格一致
- **Material Icons** — Google Material Design 图标库

### 3. 使用 CSS 自定义图标（字体图标）
```css
/* ✅ 正确：使用 Icon Font（如 Font Awesome / Iconfont） */
.icon-search::before {
  content: '\F002';
  font-family: 'Font Awesome 6 Free';
  font-weight: 900;
}
```

### 4. 手动创建 SVG 内联图标
```tsx
// ✅ 正确：内联 SVG（精确控制）
const HeartIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
  </svg>
);
```

## 例外情况

以下场景**可以**使用 Emoji：
- **内容中的装饰性文字** — 如文案中的 📢、✅、⚠️（作为文字修饰而非交互图标）
- **非正式/原型阶段** — 快速原型或玩笑项目，且明确标注会在发布前替换
- **Markdown 文档内** — README、Wiki、注释中的视觉标识（如 ✅ 已完成）

## 示例

### ❌ 错误做法（使用 Emoji 替代图标）
```tsx
// ❌ Emoji 在不同系统显示完全不同
<button>🔍 搜索</button>        {/* Windows 显示 🔍 但 Mac 是不同风格 */}
<button>✏️ 编辑</button>        {/* 某些系统显示文本样式而非彩色 */}
<button>🗑️ 删除</button>       {/* 颜色、粗细无法控制 */}
<button>⚙️ 设置</button>        {/* 对齐不一致 */}

// ❌ Emoji 作为纯图标（无文字）
<button>❤️</button>             {/* 屏幕阅读器读出"爱心"，而非"收藏" */}
<span>📁</span>                 {/* 无交互态，无法 hover 变化 */}
```

### ✅ 正确做法（使用标准图标）
```tsx
// ✅ SVG 图标组件（Lucide）
import { Search, Edit, Trash2, Settings, Heart } from 'lucide-react';

<button><Search size={16} /> 搜索</button>
<button><Edit size={16} /> 编辑</button>
<button><Trash2 size={16} /> 删除</button>
<button><Settings size={16} /> 设置</button>
<button aria-label="收藏"><Heart size={20} /></button>
```

```css
/* ✅ CSS 图标类（统一管理） */
.icon-folder { background: url('assets/icons/folder.svg') no-repeat center; }
.icon-folder:hover { opacity: 0.8; }
.icon-folder:active { transform: scale(0.95); }
```

## 检查清单

| 检查项 | 说明 |
|--------|------|
| ❌ 禁止使用 Unicode Emoji 作为 UI 图标 | ❌ `🔍 搜索`、❌ `❤️`、❌ `📁` |
| ❌ 禁止使用 Emoji 作为按钮/链接的唯一标识 | ❌ `<button>➕</button>` |
| ✅ 使用 SVG 图标或图标库 | ✅ `<SearchIcon />`、✅ `lucide-react` |
| ✅ 图标支持 aria-label 和工具提示 | ✅ `<Icon aria-label="收藏" />` |
| ✅ 交互状态（hover/active/focus） | ✅ CSS 过渡和状态样式 |
