package agent

// 多模态能力门控（2026-09-09）：会话工具面按 Provider 多模态能力自适应——
// 非多模态模型禁用「视觉依赖工具」（产出/消费图片供 LLM 看的工具），
// 多模态模型恢复启用。消除「选了纯文本模型但工具面板仍挂着截图/看图工具，
// Agent 截图后模型根本看不到图」的错配。

// visionDependentTools 视觉依赖工具清单：
//   - submit_image：把工作区图片提交给模型视觉识别（多模态专属）
//   - screenshot_desktop/window/area：截屏（截图对纯文本模型无意义——
//     LLM 看不到图，只会白耗截图动作；对话粘贴图片走 pendingImages 队列，
//     由 Loop.injectPendingImages 按 supportsMultimodal 独立门控）
var visionDependentTools = []string{
	"submit_image",
	"screenshot_desktop",
	"screenshot_window",
	"screenshot_area",
}

// ApplyMultimodalToolGate 按会话 Provider 的多模态能力启停视觉依赖工具。
// 必须在会话工具集白名单收敛（ApplyToolsetWhitelistByName/ApplyConvToolsetWhitelist）
// 之后调用——后执行的启停覆盖白名单结果（含 SystemTool 恒可用项）。
//   - 支持多模态 → 全部启用（防上一会话禁用残留）；
//   - 不支持     → 全部禁用（工具在面板仍可见可管理，换多模态模型后自动恢复）。
//
// 未注册的工具直接跳过（插件未装/未启用时不报错）。
func ApplyMultimodalToolGate(reg *Registry, prov Provider) {
	if reg == nil {
		return
	}
	on := providerSupportsMultimodal(prov)
	for _, name := range visionDependentTools {
		if _, ok := reg.Get(name); !ok {
			continue
		}
		reg.SetToolEnabled(name, on)
	}
}
