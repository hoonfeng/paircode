package main

// 层级7: 高阶 CSS 测试

func runLayer7AdvancedCSS() {
	// === T7.1: calc() 基础运算 ===
	runTest(TestCase{
		Name:   "T7.1 calc() 基础运算",
		Width:  200, Height: 100,
		HTML:   `<div style="width:calc(100px + 50px);height:50px;background:red;"></div>`,
		Checks: []PixelCheck{
			check("calc后宽度150px", 10, 10, 255, 0, 0, 255),
			check("calc外透明(超出150px)", 165, 10, 0, 0, 0, 255), // div 8..158，165 在 body 内黑背景
		},
	})

	// === T7.2: CSS custom properties (var) ===
	runTest(TestCase{
		Name:   "T7.2 CSS var() 自定义属性",
		Width:  100, Height: 100,
		HTML:   `<div style="--main-color:red;width:80px;height:80px;background:var(--main-color);"></div>`,
		Checks: []PixelCheck{
			check("var引用自定义属性", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T7.3: CSS var fallback ===
	runTest(TestCase{
		Name:   "T7.3 CSS var() fallback",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:var(--undefined-color,blue);"></div>`,
		Checks: []PixelCheck{
			check("var fallback 蓝色", 10, 10, 0, 0, 255, 255),
		},
	})

	// === T7.4: @media 媒体查询 width ===
	// viewport 宽 200（Width:200）：min-width:500px 不匹配，max-width:400px 匹配 → 蓝色
	runTest(TestCase{
		Name:   "T7.4 @media min-width",
		Width:  200, Height: 100,
		HTML: `<html><head><style>
		         @media (min-width: 500px) { .box { background: red; } }
		         @media (max-width: 400px) { .box { background: blue; } }
		       </style></head><body>
		         <div class="box" style="width:80px;height:80px;"></div>
		       </body></html>`,
		Checks: []PixelCheck{
			check("200px<400px 应匹配蓝色", 10, 10, 0, 0, 255, 255),
		},
	})

	// === T7.5: 复杂选择器 ===
	runTest(TestCase{
		Name:   "T7.5 复杂选择器",
		Width:  200, Height: 200,
		HTML: `<html><head><style>
		         .container > .child { background: red; }
		       </style></head><body>
		         <div class="container"><div class="child" style="width:80px;height:80px;"></div></div>
		       </body></html>`,
		Checks: []PixelCheck{
			check("子选择器红色", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T7.6: 兄弟选择器 ===
	runTest(TestCase{
		Name:   "T7.6 兄弟选择器",
		Width:  200, Height: 200,
		HTML: `<html><head><style>
		         .a ~ .b { background: red; }
		       </style></head><body>
		         <div class="a"></div><div class="b" style="width:80px;height:80px;"></div>
		       </body></html>`,
		Checks: []PixelCheck{
			check("兄弟选择器红色", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T7.7: 属性选择器 ===
	runTest(TestCase{
		Name:   "T7.7 属性选择器",
		Width:  200, Height: 200,
		HTML: `<html><head><style>
		         div[data-test] { background: red; }
		       </style></head><body>
		         <div data-test="val" style="width:80px;height:80px;"></div>
		       </body></html>`,
		Checks: []PixelCheck{
			check("属性选择器红色", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T7.8: :hover 伪类 ===
	runTest(TestCase{
		Name:   "T7.8 :hover 伪类",
		Width:  200, Height: 200,
		HTML: `<html><head><style>
		         div:hover { background: red; }
		       </style></head><body>
		         <div style="width:80px;height:80px;background:blue;"></div>
		       </body></html>`,
		Checks: []PixelCheck{
			check("非hover状态蓝色", 10, 10, 0, 0, 255, 255),
		},
	})

	// === T7.9: :first-child ===
	runTest(TestCase{
		Name:   "T7.9 :first-child",
		Width:  200, Height: 200,
		HTML: `<html><head><style>
		         .parent :first-child { background: red; }
		       </style></head><body>
		         <div class="parent"><div style="width:80px;height:80px;"></div><div style="width:80px;height:80px;background:blue;"></div></div>
		       </body></html>`,
		Checks: []PixelCheck{
			check("first-child红色", 10, 10, 255, 0, 0, 255),
			check("第二个child蓝色", 10, 95, 0, 0, 255, 255),
		},
	})

	// === T7.10: inline style 优先级 ===
	runTest(TestCase{
		Name:   "T7.10 inline style 优先级",
		Width:  200, Height: 100,
		HTML: `<html><head><style>div { background: blue; }</style></head><body>
		         <div style="width:80px;height:80px;background:red;"></div>
		       </body></html>`,
		Checks: []PixelCheck{
			check("inline style 优先", 10, 10, 255, 0, 0, 255),
		},
	})
}
