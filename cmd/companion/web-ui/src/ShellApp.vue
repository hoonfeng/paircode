<template>
  <div class="shell-root">
    <!-- app-root 槽位（single）：整个 IDE 界面由 UI 插件渲染（ui-app）。
         无占用者时显示空态提示（UI 插件未装/未批准）。 -->
    <div v-if="!slot.owner.value" class="shell-empty">
      <div class="shell-empty-card">
        <div class="shell-brand">PairCode</div>
        <div class="shell-title">本地 AI 结对编程 IDE</div>
        <div class="shell-msg">正在装载 UI 插件（ui-app）…</div>
        <div class="shell-hint">
          未检测到 UI 插件占用 app-root 槽位。请确认
          <code>.pair/plugins/ui-app</code> 已安装，且其 client 半已在插件面板批准。
        </div>
      </div>
    </div>
    <div v-else :ref="slot.hostRef" class="plugin-slot-host plugin-slot-app-root shell-host"></div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { useSingleSlot, syncClientHalves, startPolling, stopPolling, registerBuiltinSlot } from './plugin-runtime.js'
import api from './api.js'

// ★ 一切皆插件：壳只有一个槽位——app-root（整个 IDE 界面由 UI 插件渲染）。
//   装配视图与插件占用统一（插件面板可见 app-root 占用者 ui-app）。
registerBuiltinSlot('app-root', {
  title: 'IDE 主界面（整个 UI）',
  desc: '由 ui-app 插件渲染整个 IDE（标题栏/活动栏/侧栏/编辑器/对话/状态栏）',
})

const slot = useSingleSlot('app-root')
slot.init()

onMounted(async () => {
  slot.start()
  // ★ 全局装载 client 半（必须最前执行，不依赖其他 await——否则 ui-app
  //   的 client 半永不装载，壳永远停在空态）。链路：
  //   listPlugins → 补 clientCode（列表可能省略）→ syncClientHalves →
  //   ui-app client 半执行 → loadScript UI bundle → registerSlot('app-root')
  //   → 槽位变化通知本组件 → 渲染插件到 hostRef 容器。
  try {
    const list = (await api.listPlugins()) || []
    for (const p of list) {
      if (p.hasClient && !p.clientCode) {
        try {
          const d = await api.getPluginDetail(p.name)
          if (d && d.clientCode) p.clientCode = d.clientCode
        } catch (e) { /* 忽略：detail 失败跳过 */ }
      }
    }
    await syncClientHalves(list)
    startPolling() // 事件轮询全局启动（host→client 事件分发；幂等）
  } catch (e) {
    console.warn('[shell] client 半装载失败', e)
  }
})

onUnmounted(() => {
  slot.stop()
  stopPolling()
})
</script>

<style scoped>
.shell-root {
  width: 100%;
  height: 100%;
  background: var(--bg-primary, #0d1117);
  color: var(--text-primary, #e6edf3);
}
.shell-host,
.plugin-slot-host {
  width: 100%;
  height: 100%;
}
.shell-empty {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}
.shell-empty-card {
  max-width: 440px;
  text-align: center;
  padding: 32px;
  border: 1px solid var(--border-color, #30363d);
  border-radius: 8px;
  background: var(--bg-secondary, #161b22);
}
.shell-brand {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: var(--accent, #58a6ff);
}
.shell-title {
  font-size: 13px;
  color: var(--text-secondary, #8b949e);
  margin: 6px 0 16px;
}
.shell-msg {
  font-size: 13px;
  margin-bottom: 10px;
}
.shell-hint {
  font-size: 12px;
  color: var(--text-muted, #6e7681);
  line-height: 1.6;
}
.shell-hint code {
  font-family: var(--font-code, monospace);
  font-size: 11px;
  background: var(--bg-tertiary, #21262d);
  padding: 1px 4px;
  border-radius: 3px;
}
</style>
