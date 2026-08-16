<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content source-modal">
      <div class="modal-header">
        <h2>⎔ 源代码管理</h2>
        <button class="modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body">
        <p style="color:var(--text-muted); text-align:center; margin-top:40px;">
          Git 集成开发中<br/><br/>
          功能规划：<br/>
          · Git 状态查看<br/>
          · 暂存/提交/推送<br/>
          · 分支管理<br/>
          · Diff 对比
        </p>
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineEmits(['close'])
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  width: 400px;
  max-height: 60vh;
  display: flex;
  flex-direction: column;
}
.modal-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid var(--border-color); }
.modal-header h2 { font-size: 16px; }
.modal-close { background: none; border: none; color: var(--text-secondary); font-size: 20px; cursor: pointer; }
.modal-body { flex: 1; overflow: auto; padding: 16px; }
.modal-footer { padding: 10px 16px; border-top: 1px solid var(--border-color); display: flex; justify-content: flex-end; }
.btn-secondary { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 6px 16px; cursor: pointer; border-radius: 3px; }
</style>
