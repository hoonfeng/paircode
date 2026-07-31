package main

// 层级3: Flexbox 全面测试

func runLayer3Flexbox() {
	// === T3.1: flex-direction row (默认) ===
	runTest(TestCase{
		Name:   "T3.1 flex-direction row",
		Width:  300, Height: 100,
		HTML: `<div style="display:flex;flex-direction:row;">
		         <div style="width:80px;height:80px;background:red;"></div>
		         <div style="width:80px;height:80px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("弹性行左红色", 10, 10, 255, 0, 0, 255),
			check("弹性行右蓝色", 95, 10, 0, 0, 255, 255),
		},
	})

	// === T3.2: flex-direction column ===
	runTest(TestCase{
		Name:   "T3.2 flex-direction column",
		Width:  100, Height: 300,
		HTML: `<div style="display:flex;flex-direction:column;">
		         <div style="width:80px;height:80px;background:red;"></div>
		         <div style="width:80px;height:80px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("弹性列上红色", 10, 10, 255, 0, 0, 255),
			check("弹性列下蓝色", 10, 95, 0, 0, 255, 255),
		},
	})

	// === T3.3: justify-content center ===
	runTest(TestCase{
		Name:   "T3.3 justify-content center",
		Width:  400, Height: 100,
		HTML: `<div style="display:flex;justify-content:center;">
		         <div style="width:80px;height:80px;background:red;"></div>
		         <div style="width:80px;height:80px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("居中后的红色(大约x=120)", 120, 10, 255, 0, 0, 255),
			check("居中后的蓝色(大约x=200)", 200, 10, 0, 0, 255, 255),
		},
	})

	// === T3.4: justify-content space-between ===
	runTest(TestCase{
		Name:   "T3.4 justify-content space-between",
		Width:  400, Height: 100,
		HTML: `<div style="display:flex;justify-content:space-between;">
		         <div style="width:80px;height:80px;background:red;"></div>
		         <div style="width:80px;height:80px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("space-between左红色", 5, 10, 255, 0, 0, 255),
			check("space-between右蓝色", 315, 10, 0, 0, 255, 255),
		},
	})

	// === T3.5: align-items center ===
	runTest(TestCase{
		Name:   "T3.5 align-items center",
		Width:  200, Height: 200,
		HTML: `<div style="display:flex;align-items:center;height:200px;">
		         <div style="width:80px;height:80px;background:red;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("垂直居中红色(y≈60)", 10, 65, 255, 0, 0, 255),
		},
	})

	// === T3.6: flex-wrap wrap ===
	runTest(TestCase{
		Name:   "T3.6 flex-wrap wrap",
		Width:  150, Height: 200,
		HTML: `<div style="display:flex;flex-wrap:wrap;width:150px;">
		         <div style="width:80px;height:80px;background:red;"></div>
		         <div style="width:80px;height:80px;background:blue;"></div>
		         <div style="width:80px;height:80px;background:green;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("第1行红色", 10, 10, 255, 0, 0, 255),
			check("第2行蓝色(换行)", 10, 95, 0, 0, 255, 255),
		},
	})

	// === T3.7: flex-grow 等分 ===
	runTest(TestCase{
		Name:   "T3.7 flex-grow 等分",
		Width:  300, Height: 100,
		HTML: `<div style="display:flex;">
		         <div style="height:80px;background:red;flex-grow:1;"></div>
		         <div style="height:80px;background:blue;flex-grow:1;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("flex-grow红色区域", 10, 10, 255, 0, 0, 255),
			check("flex-grow蓝色区域(中间)", 155, 10, 0, 0, 255, 255),
		},
	})

	// === T3.8: gap 间距 ===
	runTest(TestCase{
		Name:   "T3.8 flex gap 间距",
		Width:  300, Height: 100,
		HTML: `<div style="display:flex;gap:30px;">
		         <div style="width:80px;height:80px;background:red;"></div>
		         <div style="width:80px;height:80px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("gap后红色在x=0", 10, 10, 255, 0, 0, 255),
			check("gap区域透明(x=90~110)", 100, 10, 0, 0, 0, 255),
			check("gap后蓝色在x=110", 115, 10, 0, 0, 255, 255),
		},
	})

	// === T3.9: order 排序 ===
	runTest(TestCase{
		Name:   "T3.9 order 排序",
		Width:  300, Height: 100,
		HTML: `<div style="display:flex;">
		         <div style="width:80px;height:80px;background:red;order:2;"></div>
		         <div style="width:80px;height:80px;background:blue;order:1;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("order:1的蓝色在前", 10, 10, 0, 0, 255, 255),
			check("order:2的红色在后", 95, 10, 255, 0, 0, 255),
		},
	})

	// === T3.10: align-self ===
	runTest(TestCase{
		Name:   "T3.10 align-self flex-end",
		Width:  200, Height: 200,
		HTML: `<div style="display:flex;height:200px;">
		         <div style="width:80px;height:80px;background:blue;align-self:flex-end;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("align-self底部对齐(y≈120)", 10, 125, 0, 0, 255, 255),
		},
	})

	// === T3.11: flex-direction row-reverse ===
	runTest(TestCase{
		Name:   "T3.11 flex-direction row-reverse",
		Width:  300, Height: 100,
		HTML: `<div style="display:flex;flex-direction:row-reverse;">
		         <div style="width:80px;height:80px;background:red;"></div>
		         <div style="width:80px;height:80px;background:blue;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("row-reverse蓝色在左侧", 115, 10, 0, 0, 255, 255),
			check("row-reverse红色在右侧", 210, 10, 255, 0, 0, 255),
		},
	})
}
