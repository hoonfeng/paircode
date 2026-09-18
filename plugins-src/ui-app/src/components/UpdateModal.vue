<!--
  UpdateModal — 全局「软件更新」弹窗（状态栏徽标点击打开）。
  内容复用 UpdateCard（检查 → 下载 → 校验 → 安装重启），数据源 /api/update/*（内核接口）。
  与「关于 PairCode」弹窗内的同一卡片共享状态（后端单例），两处操作等价。
-->
<template>
  <Modal max-width="640px" @close="$emit('close')">
    <template #title>
      <SvgIcon name="download" :size="14" /> 软件更新
    </template>
    <div class="um">
      <UpdateCard :current-version="version" />
      <div class="um-tips">
        <p>· 安装只替换程序与内置资源，<b>config/ 与 .pair/ 下的用户数据、配置、技能不会被覆盖</b>。</p>
        <p>· 「立即安装并重启」会在替换完成后自动重启服务；「仅安装，稍后重启」等你下次重启生效。</p>
        <p>· 更新源 / 发布频道 / 自动检查 / 校验开关：设置面板 → 软件更新。</p>
      </div>
    </div>
  </Modal>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import Modal from './Modal.vue'
import SvgIcon from './SvgIcon.vue'
import UpdateCard from './UpdateCard.vue'
import api from '../api.js'

defineEmits(['close'])

const version = ref('')

onMounted(async () => {
  try {
    const info = await api.apiGet('/system/info')
    if (info && info.version) version.value = info.version
  } catch { /* 取不到就显示占位符 */ }
})
</script>

<style scoped>
.um { padding: 12px; }
.um-tips {
  margin-top: 10px;
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.7;
}
.um-tips b { color: var(--text-primary); }
</style>
