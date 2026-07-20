package main

// 层级5: 文本渲染测试

func runLayer5Text() {
	// === T5.1: font-size / color ===
	runTest(TestCase{
		Name:   "T5.1 font-size 和 color",
		Width:  200, Height: 100,
		HTML:   `<div style="font-size:20px;color:red;">Hello</div>`,
		Checks: []PixelCheck{
			check("文字颜色红色", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T5.2: text-align center ===
	runTest(TestCase{
		Name:   "T5.2 text-align center",
		Width:  300, Height: 100,
		HTML:   `<div style="text-align:center;width:300px;background:#eee;color:red;">Center</div>`,
		Checks: []PixelCheck{
			check("文字红色", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T5.3: text-decoration underline ===
	runTest(TestCase{
		Name:   "T5.3 text-decoration underline",
		Width:  200, Height: 100,
		HTML:   `<div style="text-decoration:underline;color:red;">Underline</div>`,
		Checks: []PixelCheck{
			check("文字颜色红色", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T5.4: line-height ===
	runTest(TestCase{
		Name:   "T5.4 line-height",
		Width:  200, Height: 100,
		HTML:   `<div style="line-height:2;color:red;">Text</div>`,
		Checks: []PixelCheck{
			check("文字颜色红色", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T5.5: white-space nowrap ===
	runTest(TestCase{
		Name:   "T5.5 white-space nowrap",
		Width:  50, Height: 50,
		HTML:   `<div style="white-space:nowrap;color:red;">Long text that should not wrap</div>`,
		Checks: []PixelCheck{
			check("文字颜色红色", 10, 10, 255, 0, 0, 255),
		},
	})
}
