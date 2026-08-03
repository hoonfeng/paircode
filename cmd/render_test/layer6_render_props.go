package main

// 层级6: 渲染属性测试

func runLayer6RenderProps() {
	// === T6.1: opacity ===
	runTest(TestCase{
		Name:   "T6.1 opacity 透明度",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:red;opacity:0.5;"></div>`,
		Checks: []PixelCheck{
			check("opacity=0.5的红色(黑背景混合)", 10, 10, 127, 0, 0, 255), // premultiplied+黑背景
		},
	})

	// === T6.2: visibility hidden ===
	runTest(TestCase{
		Name:   "T6.2 visibility hidden",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:red;visibility:hidden;"></div>`,
		Checks: []PixelCheck{
			check("hidden不可见", 10, 10, 0, 0, 0, 255), // body 黑背景（元素隐藏后露出）
		},
	})

	// === T6.3: display:none ===
	runTest(TestCase{
		Name:   "T6.3 display none",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:red;display:none;"></div>`,
		Checks: []PixelCheck{
			check("none不可见", 10, 10, 0, 0, 0, 0), // display:none → body 内容高度 0，区域透明
		},
	})

	// === T6.4: overflow hidden ===
	runTest(TestCase{
		Name:   "T6.4 overflow hidden",
		Width:  100, Height: 100,
		HTML:   `<div style="width:50px;height:50px;overflow:hidden;background:red;"><div style="width:100px;height:100px;background:blue;"></div></div>`,
		Checks: []PixelCheck{
			check("overflow隐藏后子元素蓝色", 10, 10, 0, 0, 255, 255), // 容器内被子元素蓝色覆盖
			check("溢出部分不可见", 60, 10, 0, 0, 0, 255), // 容器 8..58 外 body 黑背景
		},
	})

	// === T6.5: cursor 属性 ===
	runTest(TestCase{
		Name:   "T6.5 cursor pointer",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:red;cursor:pointer;"></div>`,
		Checks: []PixelCheck{
			check("cursor不影响渲染", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T6.6: user-select ===
	runTest(TestCase{
		Name:   "T6.6 user-select none",
		Width:  100, Height: 100,
		HTML:   `<div style="width:80px;height:80px;background:red;user-select:none;"></div>`,
		Checks: []PixelCheck{
			check("user-select不影响渲染", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T6.7: box-shadow 存在 ===
	runTest(TestCase{
		Name:   "T6.7 box-shadow",
		Width:  150, Height: 150,
		HTML:   `<div style="width:80px;height:80px;background:red;box-shadow:10px 10px 5px rgba(0,0,0,0.5);"></div>`,
		Checks: []PixelCheck{
			check("红色内容", 10, 10, 255, 0, 0, 255),
		},
	})

	// === T6.8: transform 基本 ===
	runTest(TestCase{
		Name:   "T6.8 transform translate",
		Width:  200, Height: 200,
		HTML:   `<div style="width:80px;height:80px;background:red;transform:translate(30px,30px);"></div>`,
		Checks: []PixelCheck{
			check("transform偏移后", 40, 40, 255, 0, 0, 255),
		},
	})

	// === T6.9: 垂直居中组合 ===
	runTest(TestCase{
		Name:   "T6.9 垂直水平居中（flex）",
		Width:  300, Height: 300,
		HTML: `<div style="display:flex;justify-content:center;align-items:center;width:300px;height:300px;background:#222;">
		         <div style="width:100px;height:100px;background:red;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("flex居中红色", 110, 110, 255, 0, 0, 255), // 容器 8..308 居中 100px → 108..208
		},
	})
}
