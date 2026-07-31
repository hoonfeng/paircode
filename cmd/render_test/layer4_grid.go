package main

// 层级4: Grid 测试

func runLayer4Grid() {
	// === T4.1: grid-template-columns 等宽 ===
	runTest(TestCase{
		Name:   "T4.1 grid 等宽列",
		Width:  300, Height: 200,
		HTML: `<div style="display:grid;grid-template-columns:100px 100px 100px;">
		         <div style="width:100px;height:100px;background:red;"></div>
		         <div style="width:100px;height:100px;background:green;"></div>
		         <div style="width:100px;height:100px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("第1列红色", 10, 10, 255, 0, 0, 255),
			check("第2列绿色", 110, 10, 0, 255, 0, 255),
			check("第3列蓝色", 210, 10, 0, 0, 255, 255),
		},
	})

	// === T4.2: grid gap ===
	runTest(TestCase{
		Name:   "T4.2 grid gap 间距",
		Width:  300, Height: 200,
		HTML: `<div style="display:grid;grid-template-columns:100px 100px;gap:20px;">
		         <div style="width:100px;height:80px;background:red;"></div>
		         <div style="width:100px;height:80px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("第1列红色", 10, 10, 255, 0, 0, 255),
			check("gap区域透明", 115, 10, 0, 0, 0, 255),
			check("第2列蓝色", 130, 10, 0, 0, 255, 255),
		},
	})
}
