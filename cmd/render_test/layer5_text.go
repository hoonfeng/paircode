package main

// 层级5: 文本渲染测试

func runLayer5Text() {
	// === T5.1: font-size / color ===
	runTest(TestCase{
		Name:   "T5.1 font-size 和 color",
		Width:  200, Height: 100,
		HTML:   `<div style="font-size:20px;color:red;">Hello</div>`,
		Checks: []PixelCheck{
			checkRegion("文字颜色红色", 0, 0, 80, 40, 255, 0, 0, 255), // 区域内存在纯红文字像素
		},
	})

	// === T5.2: text-align center ===
	runTest(TestCase{
		Name:   "T5.2 text-align center",
		Width:  300, Height: 100,
		HTML:   `<div style="text-align:center;width:300px;background:#eee;color:red;">Center</div>`,
		Checks: []PixelCheck{
			checkRegion("文字红色", 120, 0, 80, 40, 255, 0, 0, 255), // 居中文字区域存在纯红
		},
	})

	// === T5.3: text-decoration underline ===
	runTest(TestCase{
		Name:   "T5.3 text-decoration underline",
		Width:  200, Height: 100,
		HTML:   `<div style="text-decoration:underline;color:red;">Underline</div>`,
		Checks: []PixelCheck{
			checkRegion("文字颜色红色", 0, 0, 100, 40, 255, 0, 0, 255), // 文字+下划线区域存在纯红
		},
	})

	// === T5.4: line-height ===
	runTest(TestCase{
		Name:   "T5.4 line-height",
		Width:  200, Height: 100,
		HTML:   `<div style="line-height:2;color:red;">Text</div>`,
		Checks: []PixelCheck{
			checkRegion("文字颜色红色", 0, 0, 80, 50, 255, 0, 0, 255), // line-height 2 文字区域存在纯红
		},
	})

	// === T5.5: white-space nowrap ===
	runTest(TestCase{
		Name:   "T5.5 white-space nowrap",
		Width:  50, Height: 50,
		HTML:   `<div style="white-space:nowrap;color:red;">Long text that should not wrap</div>`,
		Checks: []PixelCheck{
			checkRegion("文字颜色红色", 0, 0, 50, 40, 255, 0, 0, 255), // nowrap 文字区域存在纯红
		},
	})
}
