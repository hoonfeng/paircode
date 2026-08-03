package main

// 层级1: 基础 CSS 颜色与盒模型

func runLayer1ColorAndBox() {
	// === T1.1: #hex 颜色 ===
	runTest(TestCase{
		Name:   "T1.1 #hex 颜色",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:#ff0000;"></div>`,
		Checks: []PixelCheck{check("红色#ff0000", 10, 10, 255, 0, 0, 255)},
	})

	// === T1.2: #rrggbb 短格式 ===
	runTest(TestCase{
		Name:   "T1.2 #rgb 短格式",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:#0f0;"></div>`,
		Checks: []PixelCheck{check("绿色#0f0", 10, 10, 0, 255, 0, 255)},
	})

	// === T1.3: rgb() 函数 ===
	runTest(TestCase{
		Name:   "T1.3 rgb() 函数",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:rgb(0,0,255);"></div>`,
		Checks: []PixelCheck{check("rgb(0,0,255)", 10, 10, 0, 0, 255, 255)},
	})

	// === T1.4: rgba() 带透明度 ===
	runTest(TestCase{
		Name:   "T1.4 rgba() 透明度",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:rgba(255,0,0,0.5);"></div>`,
		Checks: []PixelCheck{check("rgba半透明", 10, 10, 127, 0, 0, 255)}, // 黑背景上 premultiplied
	})

	// === T1.5: hsl() 颜色 ===
	runTest(TestCase{
		Name:   "T1.5 hsl() 颜色",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:hsl(120,100%,50%);"></div>`,
		Checks: []PixelCheck{check("hsl绿色", 10, 10, 0, 255, 0, 255)},
	})

	// === T1.6: hsla() 颜色 ===
	runTest(TestCase{
		Name:   "T1.6 hsla() 颜色",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:hsla(240,100%,50%,0.7);"></div>`,
		Checks: []PixelCheck{check("hsla蓝色半透明", 10, 10, 0, 0, 178, 255)}, // 黑背景上 premultiplied (255×0.7)
	})

	// === T1.7: named color ===
	runTest(TestCase{
		Name:   "T1.7 named color",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:red;"></div>`,
		Checks: []PixelCheck{check("named红色", 10, 10, 255, 0, 0, 255)},
	})

	// === T1.8: transparent ===
	runTest(TestCase{
		Name:   "T1.8 transparent",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:transparent;border:1px solid red;"></div>`,
		Checks: []PixelCheck{check("transparent背景", 10, 10, 0, 0, 0, 255)}, // body 黑背景可见
	})

	// === T1.9: background shorthand ===
	runTest(TestCase{
		Name:   "T1.9 background shorthand",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:#00ff00;"></div>`,
		Checks: []PixelCheck{check("background简写", 10, 10, 0, 255, 0, 255)},
	})

	// === T1.10: margin ===
	runTest(TestCase{
		Name:   "T1.10 margin 间距",
		Width:  200, Height: 200,
		HTML: `<div style="width:100px;height:100px;background:red;"></div>
		       <div style="width:100px;height:100px;background:blue;margin-top:20px;"></div>`,
		Checks: []PixelCheck{
			check("第一个div红色", 10, 10, 255, 0, 0, 255),
			check("margin间隙(黑色背景)", 10, 115, 0, 0, 0, 255), // body margin 8px：红 div 8..108，蓝 div 从 128 起
		},
	})

	// === T1.11: padding ===
	runTest(TestCase{
		Name:   "T1.11 padding 内边距",
		Width:  200, Height: 200,
		HTML: `<div style="width:100px;height:100px;background:red;padding:20px;">
		         <div style="width:60px;height:60px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("外层红色padding区域", 15, 15, 255, 0, 0, 255),
		},
	})

	// === T1.12: border ===
	runTest(TestCase{
		Name:   "T1.12 border 边框",
		Width:  200, Height: 200,
		HTML:   `<div style="width:100px;height:100px;background:red;border:5px solid blue;"></div>`,
		Checks: []PixelCheck{
			check("蓝色边框", 10, 10, 0, 0, 255, 255), // body margin 8px：border 8..13
			check("红色内容区", 20, 20, 255, 0, 0, 255),
		},
	})

	// === T1.13: border-radius ===
	runTest(TestCase{
		Name:   "T1.13 border-radius 圆角",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:red;border-radius:10px;"></div>`,
		Checks: []PixelCheck{
			check("红色内容中心", 40, 40, 255, 0, 0, 255),
		},
	})

	// === T1.14: box-sizing border-box ===
	runTest(TestCase{
		Name:   "T1.14 box-sizing:border-box",
		Width:  200, Height: 200,
		HTML:   `<div style="width:100px;height:100px;background:red;border:10px solid blue;box-sizing:border-box;"></div>`,
		Checks: []PixelCheck{
			check("蓝色边框", 15, 15, 0, 0, 255, 255), // body margin 8px：border 8..18
			check("红色内容区缩进", 25, 25, 255, 0, 0, 255),
		},
	})
}
