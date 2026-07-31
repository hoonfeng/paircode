package main

// 层级2: 尺寸与定位

func runLayer2SizingPosition() {
	// === T2.1: width/height 固定值 ===
	runTest(TestCase{
		Name:   "T2.1 width/height 固定值",
		Width:  200, Height: 200,
		HTML:   `<div style="width:150px;height:100px;background:red;"></div>`,
		Checks: []PixelCheck{
			check("红色区域可见", 10, 10, 255, 0, 0, 255),
			check("超出高度透明", 10, 110, 0, 0, 0, 255), // body 背景
		},
	})

	// === T2.2: 百分比宽度 ===
	runTest(TestCase{
		Name:   "T2.2 百分比宽度",
		Width:  200, Height: 100,
		HTML:   `<div style="width:50%;height:100%;background:red;"></div>`,
		Checks: []PixelCheck{
			check("50%宽度红色", 10, 10, 255, 0, 0, 255),
			check("超出50%的部分透明", 110, 10, 0, 0, 0, 255),
		},
	})

	// === T2.3: min-width ===
	runTest(TestCase{
		Name:   "T2.3 min-width",
		Width:  200, Height: 100,
		HTML:   `<div style="width:10px;min-width:100px;height:50px;background:red;"></div>`,
		Checks: []PixelCheck{
			check("min-width 强制 100px", 50, 10, 255, 0, 0, 255),
		},
	})

	// === T2.4: max-width ===
	runTest(TestCase{
		Name:   "T2.4 max-width",
		Width:  200, Height: 100,
		HTML:   `<div style="width:500px;max-width:100px;height:50px;background:red;"></div>`,
		Checks: []PixelCheck{
			check("max-width 限制为 100px", 10, 10, 255, 0, 0, 255),
			check("超出100px的部分透明", 110, 10, 0, 0, 0, 255),
		},
	})

	// === T2.5: position relative + top/left ===
	runTest(TestCase{
		Name:   "T2.5 position relative",
		Width:  200, Height: 200,
		HTML:   `<div style="width:100px;height:100px;background:red;position:relative;top:30px;left:30px;"></div>`,
		Checks: []PixelCheck{
			check("relative偏移后位置", 35, 35, 255, 0, 0, 255),
			check("原始位置透明", 5, 5, 0, 0, 0, 255),
		},
	})

	// === T2.6: position absolute ===
	runTest(TestCase{
		Name:   "T2.6 position absolute",
		Width:  200, Height: 200,
		HTML:   `<div style="position:relative;width:200px;height:200px;background:#333;"><div style="position:absolute;top:50px;left:50px;width:50px;height:50px;background:red;"></div></div>`,
		Checks: []PixelCheck{
			check("absolute 定位 50,50", 55, 55, 255, 0, 0, 255),
			check("absolute 外透明", 45, 45, 51, 51, 51, 255), // #333 背景
		},
	})

	// === T2.7: z-index 层叠 ===
	runTest(TestCase{
		Name:   "T2.7 z-index 层叠",
		Width:  200, Height: 200,
		HTML: `<div style="position:relative;width:200px;height:200px;">
		         <div style="position:absolute;top:0;left:0;width:100px;height:100px;background:red;z-index:1;"></div>
		         <div style="position:absolute;top:20px;left:20px;width:100px;height:100px;background:blue;z-index:2;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("z-index高的蓝色在上层", 30, 30, 0, 0, 255, 255),
			check("红色只在z-index低层", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T2.8: float left ===
	runTest(TestCase{
		Name:   "T2.8 float left",
		Width:  200, Height: 200,
		HTML:   `<div style="float:left;width:80px;height:80px;background:red;"></div><div style="float:left;width:80px;height:80px;background:blue;"></div>`,
		Checks: []PixelCheck{
			check("float左红色", 10, 10, 255, 0, 0, 255),
			check("float左蓝色", 90, 10, 0, 0, 255, 255),
		},
	})

	// === T2.9: float right ===
	runTest(TestCase{
		Name:   "T2.9 float right",
		Width:  200, Height: 200,
		HTML:   `<div style="float:right;width:80px;height:80px;background:red;"></div>`,
		Checks: []PixelCheck{
			check("float右红色(在右侧)", 150, 10, 255, 0, 0, 255),
		},
	})

	// === T2.10: clear both ===
	runTest(TestCase{
		Name:   "T2.10 clear both",
		Width:  200, Height: 200,
		HTML: `<div style="float:left;width:80px;height:80px;background:red;"></div>
		        <div style="clear:both;width:100px;height:50px;background:blue;"></div>`,
		Checks: []PixelCheck{
			check("clear后的blue在浮动下方", 10, 90, 0, 0, 255, 255),
		},
	})
}
