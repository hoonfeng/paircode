# 单边圆角边框逐像素对比测试

验证 `paintRoundedBorderSide`（选中项蓝色竖线：`border-left: 2px solid
#58a6ff; border-radius: 6px; background: #1f2b3d`）与浏览器渲染逐像素一致。

## 原理
wb-ui 用 `border_pixel_rgb_probe.go` 渲染同一 div 并输出目标区域精确 RGB；
浏览器（Chromium/Edge 同内核）用 `border_px_ref.html` 渲染；`cmp_border_px.py`
把两者逐像素**分类对比**（竖线主体/过渡/背景/空白），判断几何位置一致性。

## 步骤
1. 启动静态服务（供浏览器加载参照页）：
   ```
   cd dev/desktop_probe && python -m http.server 9093
   ```
2. 用 web_debug 打开 `http://localhost:9093/border_px_ref.html` 并截图
   （无头浏览器 = 浏览器渲染，viewport 300x200，div 绝对定位 100,100）：
   截图保存为 `screenshots/webdebug_*.png`
3. 渲染 wb-ui 侧并输出像素：
   ```
   set CGO_ENABLED=1&& go run ./dev/desktop_probe/border_pixel_rgb_probe.go > tmp\wb_rgb.txt
   ```
4. 逐像素对比（几何分类，容差 ≤6 像素）：
   ```
   python dev/desktop_probe/cmp_border_px.py tmp\wb_rgb.txt screenshots\webdebug_*.png
   ```
   输出 `PASS`（差异 ≤6）或 `FAIL`（并列出差异像素）。

## 期望结果（2026-08-04 已验证）
wb-ui 与浏览器几何差异 **0 像素**，PASS：

| 行 | 弧带/竖线 |
|---|---|
| y=100（box 顶） | x+3..x+5（3px 凸出弧带） |
| y=101 | x+1..x+3（3px） |
| y=102 | x+1..x+2（2px 渐细） |
| y=103+ | x..x+1（中段 2px，贴背景左缘） |

差异仅在抗锯齿 alpha 分布（浏览器从左到右渐变、wb-ui 纯色）——几何一致，
视觉无差异。

## 实现（2026-08-04 重写）
paintRoundedBorderSide 用标准 CSS 月牙几何（一次 FillPath，Skia AA 平滑）：
- 外弧：圆角矩形外边界，半径 r（圆心在各角 (x±r, y±r)）
- 内弧：椭圆——沿边方向半径 r-width、垂直方向 r
  （CSS Backgrounds and Borders §5.1：内弧半径 = 外弧半径 - 相邻边框宽度；
  单边场景下相邻边宽 0，故垂直方向保持 r）
- 边框区域 = 外弧与内弧之间的月牙环带，中段为直边
- 与浏览器逐像素一致（几何差异 0），顶部/底部弧带天然镜像对称

此前版本（12 轮迭代）用 taperH/taperV 逐行渐细带 + fillBand 亚像素 alpha
渐隐反推浏览器像素，魔法数字（width-0.5-i、alphaLo 0.22 等）堆叠导致边缘
生硬、锯齿/不对称；标准几何一次填充即正确且平滑。
