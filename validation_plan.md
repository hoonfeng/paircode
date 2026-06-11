# goui 组件验证覆盖计划

> 本文档记录每个组件的验证覆盖状态和待办事项。
> 更新日期: 2026-06-01

## 验证框架

使用 `internal/validate` 包（`autovalidate.go`）进行统一管理：
- **Phase 1 — Build 结构验证**：检查 Widget() 非 nil、Parent 指针一致性
- **Phase 2 — 布局验证**：Layout 后检查尺寸合法性（非负/非 INF/非 NaN）
- **Phase 3 — Element 一致性验证**：调用各 Element 的 `Validate()` 方法

验证套件入口：`validate.Suite` + `Register(widgetType, buildCheck, layoutCheck, stateCheck)`

---

## 组件验证覆盖矩阵

### 已实现 Validate() 的 Element（共 21 个）

| # | Element 类型 | Widget 类型 | Validate() | Phase1(Build) | Phase2(Layout) | Phase3(State) | 备注 |
|---|-------------|------------|:----------:|:-------------:|:--------------:|:-------------:|------|
| 1 | BaseElement | —（基类） | ✅ | ✅ 通用 | ✅ 通用 | ✅ parent 指针在子类中检查 | 基类默认实现 |
| 2 | StatelessElement | StatelessWidget | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 builder 检查 | 无状态基类 |
| 3 | StatefulElement | StatefulWidget | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 state/child 一致性 | 有状态基类 |
| 4 | ContainerElement | Container | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 SingleChild 一致性 | — |
| 5 | ButtonElement | Button | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 SingleChild 一致性 | — |
| 6 | CardElement | Card | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 SingleChild 一致性 | — |
| 7 | ScrollViewElement | ScrollView | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 maxScroll 校验 | — |
| 8 | FlexElement | Flex/Row/Column | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 MultiChild 一致性 | — |
| 9 | DialogOverlayElement | Dialog | ✅ | ✅ 通用 | ✅ 通用 | ✅ 含 SingleChild 一致性 | — |
| 10 | TextElement | Text | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 11 | CheckboxElement | Checkbox | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 12 | SwitchElement | Switch | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 13 | RadioButtonElement | RadioButton | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 14 | SliderElement | Slider | ✅ | ✅ 通用 | ✅ 通用 | ✅ 值范围校验 | — |
| 15 | ProgressBarElement | ProgressBar | ✅ | ✅ 通用 | ✅ 通用 | ✅ 值范围 [0,1] 校验 | — |
| 16 | InputElement | Input | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 17 | IconElement | Icon | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 18 | SpacerElement | Spacer | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 19 | DividerElement | Divider | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 20 | ImageElement | Image | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |
| 21 | TreeViewElement | TreeView | ✅ | ✅ 通用 | ✅ 通用 | ✅ 子 Element 数量=0 | — |

### 暂无 Validate() 的 Element（共 2 个）

| # | Element 类型 | Widget 类型 | Validate() | 建议优先级 | 说明 |
|---|-------------|------------|:----------:|:----------:|------|
| 22 | RadioGroupElement | RadioGroup | ❌ 缺失 | 低 | 内部 Element，由 RadioButton 管理，暂不独立验证 |
| 23 | RenderObjectElement | — | ❌ 缺失 | 低 | 低级渲染基类，暂不验证 |

---

## 验证套件注册 API

```go
// 创建套件
suite := validate.NewSuite()

// 注册全局验证器（"*" 匹配所有类型）
suite.Register("*",
    validate.BuildCheckStandard(), // Phase 1: Widget 非 nil + Parent 指针
    validate.LayoutCheckStandard(), // Phase 2: 尺寸非负/非 INF/非 NaN
    nil,                           // Phase 3: 使用 widget 内建 ValidateElementTree
)

// 运行
report := suite.RunAll(&validate.SuiteContext{
    Root:        rootElement,
    Constraints: &constraint,
})
```

## 运行结果（2026-06-01）

| 阶段 | 结果 | 详情 |
|------|:----:|------|
| Phase 1 — Build 结构验证 | ✅ PASS | 87 节点, 0 失败, 87 通过 |
| Phase 2 — 布局验证 | ✅ PASS | 87 元素, 所有尺寸验证通过 |
| Phase 3 — Element 一致性验证 | ✅ PASS | 所有 Validate() 通过 |
| **总计** | **✅ ALL PASS** | **19 种 Widget 类型, 87 个 Element** |

---

## 待办项（TODO）

### P0 — 关键（无）
所有主要组件均已通过验证。

### P1 — 重要
1. **RadioGroupElement.Validate()**：添加 Validate() 方法，验证 groupName 一致性
2. **RenderObjectElement.Validate()**：添加基础 Validate() 方法（尺寸+Widget 非 nil）

### P2 — 增强
3. **per-type 注册示例**：在 self_validation 中添加按组件类型注册的具体验证 demo
4. **验证函数单元测试**：为 `internal/validate/autovalidate.go` 编写单元测试
5. **组件特定检查增强**：为各组件添加更深入的验证（如 Container 的 Padding 合理性、Input 的 MaxLength 等）

### P3 — 未来
6. **集成到 GUI 模式**：将验证套件集成到 examples/test/main.go 的可选验证路径中
7. **错误聚合与报告改进**：增加 JSON/结构化报告输出能力
8. **性能指标**：记录每个阶段的执行耗时，建立基准
