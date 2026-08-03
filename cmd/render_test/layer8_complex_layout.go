package main

// 层级8: 综合布局/IDE 模拟

func runLayer8ComplexLayout() {
	// === T8.1: flex 嵌套布局：顶部栏+内容区 ===
	runTest(TestCase{
		Name:   "T8.1 flex 嵌套布局",
		Width:  400, Height: 400,
		HTML: `<div style="display:flex;flex-direction:column;width:400px;height:400px;">
		         <div style="height:40px;background:#3a3a3a;"></div>
		         <div style="display:flex;flex:1;">
		           <div style="width:60px;background:#2d2d2d;"></div>
		           <div style="flex:1;background:#1e1e1e;"></div>
		         </div>
		       </div>`,
		Checks: []PixelCheck{
			check("顶部栏 #3a3a3a", 50, 10, 58, 58, 58, 255),
			check("左侧栏 #2d2d2d", 10, 50, 45, 45, 45, 255),
			check("内容区 #1e1e1e", 70, 50, 30, 30, 30, 255),
		},
	})

	// === T8.2: border + padding + margin 组合 ===
	runTest(TestCase{
		Name:   "T8.2 border+padding+margin 组合",
		Width:  300, Height: 200,
		HTML: `<div style="margin:20px;padding:10px;border:5px solid blue;background:#ddd;">
		         <div style="width:100px;height:80px;background:red;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("蓝色边框", 30, 30, 0, 0, 255, 255), // body margin8+margin20=28 起点，border 28..33
			check("#ddd padding区域", 38, 38, 221, 221, 221, 255), // padding 33..43
			check("内部红色", 46, 46, 255, 0, 0, 255), // 子 div 43 起
		},
	})

	// === T8.3: position absolute + flex 组合 ===
	runTest(TestCase{
		Name:   "T8.3 absolute + flex 组合",
		Width:  300, Height: 200,
		HTML: `<div style="position:relative;width:300px;height:200px;background:#333;">
		         <div style="position:absolute;top:20px;right:20px;width:50px;height:50px;background:red;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("右上红色定位", 240, 30, 255, 0, 0, 255), // 外层 8..308，absolute top20 right20 → 238..288 × 28..78
		},
	})

	// === T8.4: 多级 flex 居中 ===
	runTest(TestCase{
		Name:   "T8.4 多级 flex 居中",
		Width:  300, Height: 200,
		HTML: `<div style="display:flex;justify-content:center;align-items:center;width:300px;height:200px;background:#444;">
		         <div style="display:flex;justify-content:center;align-items:center;width:150px;height:100px;background:#666;">
		           <div style="width:50px;height:50px;background:red;"></div>
		         </div>
		       </div>`,
		Checks: []PixelCheck{
			check("多级居中红色", 135, 85, 255, 0, 0, 255), // 内层 83..233 × 58..158，红 133..183 × 83..133
		},
	})

	// === T8.5: float + clear 布局 ===
	runTest(TestCase{
		Name:   "T8.5 float 经典布局",
		Width:  400, Height: 300,
		HTML: `<div style="width:400px;background:#eee;">
		         <div style="float:left;width:100px;height:100px;background:red;"></div>
		         <div style="float:left;width:100px;height:100px;background:green;"></div>
		         <div style="float:left;width:100px;height:100px;background:blue;"></div>
		         <div style="clear:both;width:100%;height:50px;background:yellow;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("float红色", 10, 10, 255, 0, 0, 255),
			check("float绿色", 110, 10, 0, 255, 0, 255),
			check("float蓝色", 210, 10, 0, 0, 255, 255),
			check("clear后黄色", 10, 110, 255, 255, 0, 255),
		},
	})

	// === T8.6: 多列布局 (flex) ===
	runTest(TestCase{
		Name:   "T8.6 flex 多列布局",
		Width:  500, Height: 300,
		HTML: `<div style="display:flex;width:500px;height:200px;">
		         <div style="width:100px;background:#ff6b6b;"></div>
		         <div style="flex:1;background:#4ecdc4;"></div>
		         <div style="width:100px;background:#45b7d1;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("左侧 #ff6b6b", 10, 10, 255, 107, 107, 255),
			check("中间 #4ecdc4", 150, 10, 78, 205, 196, 255),
			check("右侧 #45b7d1", 410, 10, 69, 183, 209, 255),
		},
	})

	// === T8.7: 模拟 IDE 暗色主题侧边栏 ===
	runTest(TestCase{
		Name:   "T8.7 IDE 暗色主题",
		Width:  400, Height: 300,
		HTML: `<div style="display:flex;width:400px;height:300px;">
		         <div style="width:50px;background:#2c2c2c;display:flex;flex-direction:column;align-items:center;padding-top:10px;">
		           <div style="width:30px;height:30px;background:#0078d4;border-radius:4px;"></div>
		           <div style="width:30px;height:30px;background:#555;border-radius:4px;margin-top:8px;"></div>
		         </div>
		         <div style="flex:1;background:#1e1e1e;"></div>
		       </div>`,

		Checks: []PixelCheck{
			check("侧边栏 #2c2c2c", 10, 10, 44, 44, 44, 255),
			check("蓝色图标 #0078d4", 20, 20, 0, 120, 212, 255), // 侧边栏 8..58，图标 18..48 × 18..48（padding-top 10）
			check("灰色图标 #555", 20, 60, 85, 85, 85, 255),      // 图标2 margin-top 8 → 56..86
			check("内容区 #1e1e1e", 60, 10, 30, 30, 30, 255),
		},
	})

	// === T8.8: z-index 复杂层叠 ===
	runTest(TestCase{
		Name:   "T8.8 z-index 层叠上下文",
		Width:  200, Height: 200,
		HTML: `<div style="position:relative;width:200px;height:200px;background:#333;">
		         <div style="position:absolute;top:20px;left:20px;width:80px;height:80px;background:red;z-index:2;"></div>
		         <div style="position:absolute;top:40px;left:40px;width:80px;height:80px;background:blue;z-index:1;"></div>
		         <div style="position:absolute;top:10px;left:10px;width:80px;height:80px;background:green;z-index:3;"></div>
		       </div>`,
		Checks: []PixelCheck{
			check("z-index:3绿色在最上", 20, 20, 0, 255, 0, 255), // 绿 top10 left10 → 18..98（body margin 8）
			check("z-index:2红色在中间层(被绿色盖住部分)", 25, 25, 0, 255, 0, 255),
			check("z-index:3绿色盖住蓝色(50,50)", 50, 50, 0, 255, 0, 255), // 蓝 z1 在 48..128，但绿 z3 覆盖
		},
	})
}
