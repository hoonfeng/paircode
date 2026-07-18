<template>
  <div class="modal-overlay">
    <div class="modal-content settings-modal">
      <div class="modal-header">
        <h2><SvgIcon name="settings" :size="18" /> 设置</h2>
        <button class="modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="modal-body">
        <div class="settings-tabs">
          <button v-for="tab in tabs" :key="tab.id"
                  :class="{ active: activeTab === tab.id }"
                  @click="activeTab = tab.id">{{ tab.label }}</button>
        </div>
        <div class="settings-body">
          <!-- ═══ AI 模型 ═══ -->
          <div v-if="activeTab === 'ai'">
            <div class="setting-group">
              <div class="group-title">服务商与模型</div>
              <div class="setting-row">
                <label>服务商</label>
                <select v-model="local.provider" @change="onProviderChange">
                  <option value="" disabled>选择服务商</option>
                  <option v-for="p in providers" :key="p" :value="p">{{ providerLabel(p) }}</option>
                </select>
              </div>
              <div class="setting-row">
                <label>API 地址</label>
                <input type="text" v-model="local.baseURL" placeholder="https://api.openai.com/v1" />
              </div>
              <div class="setting-row">
                <label>API Key</label>
                <input type="password" v-model="local.apiKey" placeholder="sk-..." />
              </div>
              <div class="setting-row">
                <label>主模型</label>
                <select v-model="local.executeModel">
                  <option value="" disabled>选择模型</option>
                  <option v-for="m in modelsForProvider" :key="m" :value="m">{{ m }}</option>
                  <option value="custom">自定义</option>
                </select>
                <input v-if="local.executeModel === 'custom'" v-model="local.executeModelCustom"
                       placeholder="手动输入模型名" class="set-input-sm flex-1" />
              </div>
              <div class="setting-row">
                <label>规划模型</label>
                <select v-model="local.planModel">
                  <option value="" disabled>选择模型</option>
                  <option v-for="m in modelsForProvider" :key="m" :value="m">{{ m }}</option>
                  <option value="custom">自定义</option>
                </select>
                <input v-if="local.planModel === 'custom'" v-model="local.planModelCustom"
                       placeholder="手动输入模型名" class="set-input-sm flex-1" />
              </div>
              <div class="setting-row">
                <label>审核模型</label>
                <select v-model="local.reviewModel">
                  <option value="" disabled>选择模型</option>
                  <option v-for="m in modelsForProvider" :key="m" :value="m">{{ m }}</option>
                  <option value="custom">自定义</option>
                </select>
                <input v-if="local.reviewModel === 'custom'" v-model="local.reviewModelCustom"
                       placeholder="手动输入模型名" class="set-input-sm flex-1" />
              </div>
              <div class="setting-row">
                <label>温度</label>
                <input type="range" min="0" max="2" step="0.1" v-model.number="local.temperature" />
                <span class="range-val">{{ local.temperature }}</span>
                <span class="setting-hint" v-if="local.temperature > 0.8">⚠️ 高温度降低代码稳定性，建议 ≤0.5</span>
              </div>
              <div class="setting-row">
                <label>最大 Token</label>
                <input type="number" v-model.number="local.maxTokens" min="4096" max="128000" />
                <span class="setting-hint" v-if="local.maxTokens < 8192">⚠️ 过小会导致思考/回复被截断，建议 ≥8192</span>
              </div>
              <div class="setting-row">
                <label>上下文 Token</label>
                <input type="number" v-model.number="local.contextMaxTokens" min="4096" max="1000000" />
              </div>
              <div class="setting-row">
                <label>思考模式</label>
                <select v-model="local.thinkingMode" style="flex:1">
                  <option value="">禁用</option>
                  <option value="thinking">启用思考</option>
                  <option value="non-thinking">非思考模式</option>
                </select>
              </div>
            </div>

            <div class="setting-group" style="margin-top:16px">
              <div class="group-title">上下文压缩（压缩历史后减少 Token 消耗）</div>
              <div class="setting-row">
                <label>启用压缩</label>
                <input type="checkbox" v-model="local.compressEnabled" />
              </div>
              <div class="setting-row">
                <label>压缩服务商</label>
                <select v-model="local.compressProvider" @change="onCompressProviderChange">
                  <option value="" disabled>选择服务商</option>
                  <option v-for="p in providers" :key="p" :value="p">{{ providerLabel(p) }}</option>
                </select>
              </div>
              <div class="setting-row">
                <label>压缩 API 地址</label>
                <input type="text" v-model="local.compressBaseURL" placeholder="https://api.deepseek.com/v1" />
              </div>
              <div class="setting-row">
                <label>压缩 API Key</label>
                <input type="password" v-model="local.compressApiKey" placeholder="留空则复用主 Key" />
              </div>
              <div class="setting-row">
                <label>压缩模型</label>
                <select v-model="local.compressModel">
                  <option value="" disabled>选择模型</option>
                  <option v-for="m in compressModelsForProvider" :key="m" :value="m">{{ m }}</option>
                </select>
              </div>
              <div class="setting-row">
                <label>压缩思考模式</label>
                <select v-model="local.compressThinkingMode" style="flex:1">
                  <option value="non-thinking">非思考（推荐）</option>
                  <option value="thinking">启用思考</option>
                  <option value="">禁用</option>
                </select>
              </div>
            </div>
          </div>

          <!-- ═══ Agent 行为 ═══ -->
          <div v-if="activeTab === 'agent'">
            <div class="setting-group">
              <div class="group-title">Agent 行为</div>
              <div class="setting-row">
                <label>最大迭代次数</label>
                <input type="number" v-model.number="local.maxIterations" min="1" max="200" />
              </div>
              <div class="setting-row">
                <label>最大并行 Agent 数</label>
                <input type="number" v-model.number="local.maxParallel" min="1" max="20" />
              </div>
              <div class="setting-row">
                <label>审核重试次数</label>
                <input type="number" v-model.number="local.reviewRetries" min="0" max="20" />
              </div>
              <div class="setting-row">
                <label>破坏性操作需确认</label>
                <input type="checkbox" v-model="local.requireHumanApprovalForDestructive" />
              </div>
              <div class="setting-row">
                <label>审核模式</label>
                <select v-model="local.reviewMode" style="flex:1">
                  <option value="auto">AI审核（自动审批写操作）</option>
                  <option value="manual">手动审批（每次需用户确认）</option>
                  <option value="off">关闭审核（全部放行）</option>
                </select>
              </div>
              <div class="setting-row">
                <label>拒绝后自动迭代</label>
                <input type="checkbox" v-model="local.autoIterateOnRejection" />
              </div>
              <div class="setting-row">
                <label>自主模式</label>
                <input type="checkbox" v-model="local.autonomous" />
              </div>
              <div class="setting-row">
                <label>AI 审核（规划/审核 Agent）</label>
                <input type="checkbox" v-model="local.aiReview" />
              </div>
              <div class="setting-row">
                <label>Lua 工具</label>
                <input type="checkbox" v-model="local.luaTools" />
              </div>
              <div class="setting-row">
                <label>自动 Git 提交</label>
                <input type="checkbox" v-model="local.autoCommit" />
                <span class="setting-hint">任务完成时自动 git add + commit</span>
              </div>
            </div>
            <div class="setting-group" style="margin-top:12px">
              <div class="group-title">搜索与忽略</div>
              <div class="setting-row">
                <label>SearXNG 地址</label>
                <input type="text" v-model="local.searxngUrl" placeholder="留空使用 DuckDuckGo" />
              </div>
              <div class="setting-row">
                <label>忽略目录</label>
                <input type="text" v-model="local.ignoreDirsText" placeholder="node_modules,.git,dist" />
              </div>
            </div>
          </div>

          <!-- ═══ 编辑器 ═══ -->
          <div v-if="activeTab === 'editor'">
            <div class="setting-group">
              <div class="group-title">编辑器</div>
              <div class="setting-row">
                <label>字号</label>
                <input type="number" v-model.number="local.editorFontSize" min="10" max="32" />
              </div>
              <div class="setting-row">
                <label>制表符大小</label>
                <input type="number" v-model.number="local.tabSize" min="1" max="8" />
              </div>
              <div class="setting-row">
                <label>自动换行</label>
                <input type="checkbox" v-model="local.wordWrap" />
              </div>
              <div class="setting-row">
                <label>隐藏 Minimap</label>
                <input type="checkbox" v-model="local.hideMinimap" />
              </div>
            </div>
            <div class="setting-group" style="margin-top:12px">
              <div class="group-title">字体风格</div>
              <div class="setting-row">
                <label>编辑器字族</label>
                <input type="text" v-model="local.fontFamily" placeholder="'Cascadia Code', monospace" />
              </div>
              <div class="setting-row">
                <label>加粗</label>
                <input type="checkbox" v-model="local.editorFontBold" />
              </div>
              <div class="setting-row">
                <label>斜体</label>
                <input type="checkbox" v-model="local.editorFontItalic" />
              </div>
              <div class="setting-row">
                <label>下划线</label>
                <input type="checkbox" v-model="local.editorFontUnderline" />
              </div>
            </div>
            <div class="setting-group" style="margin-top:12px">
              <div class="group-title">界面字体</div>
              <div class="setting-row">
                <label>UI 字族</label>
                <input type="text" v-model="local.uiFontFamily" placeholder="Inter, sans-serif" />
              </div>
              <div class="setting-row">
                <label>加粗</label>
                <input type="checkbox" v-model="local.uiFontBold" />
              </div>
              <div class="setting-row">
                <label>斜体</label>
                <input type="checkbox" v-model="local.uiFontItalic" />
              </div>
              <div class="setting-row">
                <label>下划线</label>
                <input type="checkbox" v-model="local.uiFontUnderline" />
              </div>
            </div>
          </div>

          <!-- ═══ 终端 ═══ -->
          <div v-if="activeTab === 'terminal'">
            <div class="setting-group">
              <div class="group-title">终端</div>
              <div class="setting-row">
                <label>默认 Shell</label>
                <select v-model="local.defaultShell" style="flex:1">
                  <option value="auto">自动检测</option>
                  <option value="cmd">cmd</option>
                  <option value="powershell">PowerShell</option>
                  <option value="git-bash">Git Bash</option>
                </select>
              </div>
              <div class="setting-row">
                <label>终端字号</label>
                <input type="number" v-model.number="local.termFontSize" min="10" max="24" />
              </div>
              <div class="setting-row">
                <label>编码</label>
                <select v-model="local.termEncoding" style="flex:1">
                  <option value="auto">自动</option>
                  <option value="utf-8">UTF-8</option>
                  <option value="gbk">GBK</option>
                </select>
              </div>
            </div>
          </div>

          <!-- ═══ 外观 ═══ -->
          <div v-if="activeTab === 'appearance'">
            <div class="setting-group">
              <div class="group-title">选择主题</div>
              <div class="theme-grid">
                <div v-for="th in themeList" :key="th.id"
                     :class="['theme-card', { selected: local.theme === th.id }]"
                     @click="local.theme = th.id">
                  <div class="theme-preview">
                    <div class="tp-activity" :style="{background: th.colors.activity}"></div>
                    <div class="tp-main">
                      <div class="tp-sidebar" :style="{background: th.colors.sidebar}"></div>
                      <div class="tp-editor">
                        <div class="tp-line" :style="{background: th.colors.line1}"></div>
                        <div class="tp-line tp-line-accent" :style="{background: th.colors.line2}"></div>
                        <div class="tp-line" :style="{background: th.colors.line3}"></div>
                      </div>
                      <div class="tp-accent-bar" :style="{background: th.colors.accent}"></div>
                    </div>
                  </div>
                  <div class="theme-name">{{ th.label }}</div>
                  <div class="theme-font">{{ th.fontDesc }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- ═══ 指令 ═══ -->
          <div v-if="activeTab === 'instructions'">
            <div class="setting-group">
              <div class="group-title">系统级指令（所有工作区共享）</div>
              <div class="setting-row-vertical">
                <textarea v-model="local.systemInstructions" class="inst-textarea"
                          placeholder="输入系统级指令，Agent 在每个对话中都会遵守…" rows="6"></textarea>
              </div>
            </div>
            <div class="setting-group" style="margin-top:16px">
              <div class="group-title">项目级指令</div>
              <div class="setting-row-vertical" style="display:flex;flex-direction:column;gap:6px">
                <div class="project-inst-hint">
                  当前工作区：<code>{{ wsRoot || '未设置' }}</code>
                  <button v-if="wsRoot" class="btn-xs" @click="reloadProjectInst">重新加载</button>
                </div>
                <textarea v-model="local.projectInstructions" class="inst-textarea"
                          placeholder="输入此工作区的项目级指令，存储在 .pair/instructions.md…" rows="6"></textarea>
              </div>
            </div>
          </div>

          <!-- ═══ 思想 ═══ -->
          <div v-if="activeTab === 'philosophy'">
            <div class="setting-group">
              <div class="group-title">思想注入（Philosophy）</div>
              <div class="setting-row">
                <label>启用思想注入</label>
                <input type="checkbox" v-model="local.philosophyEnabled" />
              </div>
            </div>
            <div v-if="local.philosophyEnabled" class="setting-group" style="margin-top:12px">
              <div class="group-title">经典选择（选中后注入 Agent 系统提示）</div>
              <div class="classics-list">
                <label v-for="c in classicList" :key="c.id" class="classic-item">
                  <input type="checkbox" :value="c.id" v-model="local.philosophySelected" />
                  <span>{{ c.name }}</span>
                </label>
              </div>
            </div>
            <div v-if="local.philosophyEnabled" class="setting-group" style="margin-top:16px">
              <div class="group-title">主 Agent 哲学</div>
              <div class="setting-row-vertical">
                <textarea v-model="local.mainAgentPhilosophy" class="inst-textarea"
                          placeholder="为主 Agent 定制的专属哲学指引（可选）…" rows="3"></textarea>
              </div>
            </div>
            <div v-if="local.philosophyEnabled" class="setting-group" style="margin-top:12px">
              <div class="group-title">子 Agent 角色哲学</div>
              <div v-for="role in roleList" :key="role.id" class="setting-row-vertical" style="margin-bottom:8px">
                <div class="role-phil-label">{{ role.name }}</div>
                <textarea :value="local.philosophyRoles[role.id] || ''"
                          @input="onRolePhilInput(role.id, $event)"
                          class="inst-textarea" rows="2"
                          :placeholder="role.name + '的哲学指引（可选）'"></textarea>
              </div>
            </div>
          </div>

        </div>
      </div>
      <div class="modal-footer">
        <button class="btn-secondary" @click="resetForm">撤销</button>
        <button class="btn-primary" @click="saveSettings">保存设置</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch, computed } from 'vue'
import { state } from '../main.js'
import api from '../api.js'
import SvgIcon from './SvgIcon.vue'
import { applyTheme } from '../main.js'

const emit = defineEmits(['close'])
const activeTab = ref('ai')

const tabs = [
  { id: 'ai', label: 'AI' },
  { id: 'agent', label: 'Agent' },
  { id: 'editor', label: '编辑器' },
  { id: 'terminal', label: '终端' },
  { id: 'appearance', label: '外观' },
  { id: 'instructions', label: '指令' },
  { id: 'philosophy', label: '思想' },
]

const providers = ref([])
const modelsMap = ref({})
const classicList = ref([])
const roleList = ref([])
const wsRoot = ref('')

const local = reactive({
  provider: '',
  baseURL: '',
  apiKey: '',
  executeModel: '',
  executeModelCustom: '',
  planModel: '',
  reviewModel: '',
  temperature: 0.3,
  maxTokens: 16384,
  contextMaxTokens: 1000000,
  thinkingMode: 'thinking',
  // 压缩
  compressEnabled: false,
  compressProvider: '',
  compressBaseURL: '',
  compressApiKey: '',
  compressModel: '',
  compressThinkingMode: 'non-thinking',
  planModelCustom: '',
  reviewModelCustom: '',
  compressModelCustom: '',
  // Agent
  maxIterations: 50,
  maxParallel: 3,
  reviewRetries: 3,
  requireHumanApprovalForDestructive: true,
  reviewMode: 'auto',
  autoIterateOnRejection: false,
  autonomous: false,
  aiReview: false,
  luaTools: true,
  autoCommit: true,
  searxngUrl: '',
  ignoreDirsText: '',
  // 编辑器
  editorFontSize: 14,
  tabSize: 2,
  wordWrap: false,
  hideMinimap: false,
  fontFamily: "'Cascadia Code', 'Fira Code', Consolas, monospace",
  editorFontBold: false,
  editorFontItalic: false,
  editorFontUnderline: false,
  uiFontFamily: '',
  uiFontBold: false,
  uiFontItalic: false,
  uiFontUnderline: false,
  // 外观
  theme: 'dark',
  // 终端
  defaultShell: 'auto',
  termFontSize: 13,
  termEncoding: 'auto',
  // 指令
  systemInstructions: '',
  projectInstructions: '',
  // 思想
  philosophyEnabled: false,
  philosophySelected: [],
  mainAgentPhilosophy: '',
  philosophyRoles: {},

})

const modelsForProvider = computed(() => {
  if (!local.provider || !modelsMap.value[local.provider]) return []
  return modelsMap.value[local.provider]
})

const compressModelsForProvider = computed(() => {
  if (!local.compressProvider || !modelsMap.value[local.compressProvider]) return []
  return modelsMap.value[local.compressProvider]
})

// ─── 主题 ───
const themeList = [
  { id: 'dark', label: '暗色科技风', fontDesc: 'Inter + JetBrains Mono',
    colors: { activity: '#0d1117', sidebar: '#161b22', editor: '#0d1117',
              line1: '#21262d', line2: '#58a6ff33', line3: '#30363d', accent: '#58a6ff' } },
  { id: 'light', label: '白色简约风', fontDesc: 'Inter + JetBrains Mono',
    colors: { activity: '#2c2c2c', sidebar: '#f8f9fa', editor: '#ffffff',
              line1: '#e8eaed', line2: '#1a73e833', line3: '#dadce0', accent: '#1a73e8' } },
  { id: 'warm', label: '暖色温暖风', fontDesc: 'Noto Serif SC + Source Code Pro',
    colors: { activity: '#5c4033', sidebar: '#f5ece0', editor: '#faf3e8',
              line1: '#efe4d4', line2: '#b8733344', line3: '#d6c8b8', accent: '#b87333' } },
  { id: 'night', label: '暗夜紫风', fontDesc: 'Inter + JetBrains Mono',
    colors: { activity: '#12101a', sidebar: '#1a1726', editor: '#12101a',
              line1: '#221f30', line2: '#9b8ec444', line3: '#2d2940', accent: '#9b8ec4' } },
]

// ─── 模型 ───
function providerLabel(p) {
  const labels = { deepseek: 'DeepSeek', openai: 'OpenAI', anthropic: 'Anthropic', 'openai-compatible': '兼容 OpenAI', custom: '自定义' }
  return labels[p] || p
}

async function loadModels() {
  try {
    const data = await api.getModels()
    providers.value = data.providers || []
    modelsMap.value = data.models || {}
    // 使用后端返回的默认 API 地址
    if (data.providerBaseURLs) {
      runtimeProviderBaseURLs = data.providerBaseURLs
    }
  } catch (e) {
    // fallback
    providers.value = ['deepseek', 'openai', 'anthropic', 'openai-compatible']
    modelsMap.value = {
      deepseek: ['deepseek-r1', 'deepseek-v4-pro', 'deepseek-v4-flash'],
      openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4.1', 'gpt-4.1-mini', 'gpt-4.1-nano', 'o1', 'o3-mini', 'o4-mini'],
      anthropic: ['claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022', 'claude-4-sonnet-20250514', 'claude-4-haiku-latest'],
      'openai-compatible': ['custom'],
    }
  }
}

// ─── 服务商默认 API 地址（硬编码兜底，运行时优先用后端返回的值）───
const defaultProviderBaseURLs = {
  deepseek: 'https://api.deepseek.com/v1',
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
  'openai-compatible': '',
  custom: '',
}
// 运行时从后端获取的 providerBaseURLs（loadModels 时更新）
let runtimeProviderBaseURLs = {}

function getProviderBaseURL(provider) {
  return runtimeProviderBaseURLs[provider] || defaultProviderBaseURLs[provider] || ''
}

function onProviderChange() {
  // 自动填充对应服务商的 API 地址
  local.baseURL = getProviderBaseURL(local.provider)
  // 如果当前模型不在新服务商列表中，重置
  const models = modelsMap.value[local.provider] || []
  if (models.length > 0) {
    if (!models.includes(local.executeModel)) local.executeModel = models[0]
    if (!models.includes(local.planModel)) local.planModel = models[0]
    if (!models.includes(local.reviewModel)) local.reviewModel = models[0]
  }
}

function onCompressProviderChange() {
  // 自动填充压缩服务商的 API 地址
  local.compressBaseURL = getProviderBaseURL(local.compressProvider)
  const models = modelsMap.value[local.compressProvider] || []
  if (models.length > 0 && !models.includes(local.compressModel)) {
    local.compressModel = models[0]
  }
}

function onRolePhilInput(roleId, event) {
  local.philosophyRoles[roleId] = event.target.value
}

// ─── 加载指令 ───
async function loadInstructions() {
  try {
    const sys = await api.getInstructions('system')
    local.systemInstructions = sys.content || ''
  } catch {}
  try {
    const proj = await api.getInstructions('project')
    local.projectInstructions = proj.content || ''
  } catch {}
}

async function loadPhilosophy() {
  try {
    const data = await api.getPhilosophy()
    local.philosophyEnabled = data.enabled || false
    local.philosophySelected = data.selected || []
    const roles = data.roles || {}
    local.mainAgentPhilosophy = roles['main'] || ''
    const rolePhil = { ...roles }
    delete rolePhil['main']
    local.philosophyRoles = rolePhil
    classicList.value = data.availableClassics || []
    roleList.value = data.availableRoles || []
  } catch {
    classicList.value = [
      { id: 'tao-te-ching', name: '《道德经》' },
      { id: 'huangdi-yinfu-jing', name: '《黄帝阴符经》' },
      { id: 'sunzi-bingfa', name: '《孙子兵法》' },
    ]
    roleList.value = [
      { id: 'planner', name: '规划 Agent' },
      { id: 'reviewer', name: '审核 Agent' },
    ]
  }
}



// ─── 加载设置到 local ───
function loadSettings() {
  const s = state.settings
  if (!s) return
  local.provider = s.provider || ''
  local.baseURL = s.baseURL || ''
  local.apiKey = s.apiKey || ''
  local.executeModel = s.executeModel || s.model || ''
  // 检测自定义模型：如果模型名不在下拉列表中，标记为自定义并填入自定义输入框
  const execModels = modelsMap.value[local.provider] || []
  local.executeModelCustom = ''
  if (local.executeModel && execModels.length > 0 && !execModels.includes(local.executeModel) && local.executeModel !== 'custom') {
    local.executeModelCustom = local.executeModel
    local.executeModel = 'custom'
  }
  local.planModel = s.planModel || ''
  local.planModelCustom = ''
  if (local.planModel && execModels.length > 0 && !execModels.includes(local.planModel) && local.planModel !== 'custom') {
    local.planModelCustom = local.planModel
    local.planModel = 'custom'
  }
  local.reviewModel = s.reviewModel || ''
  local.reviewModelCustom = ''
  if (local.reviewModel && execModels.length > 0 && !execModels.includes(local.reviewModel) && local.reviewModel !== 'custom') {
    local.reviewModelCustom = local.reviewModel
    local.reviewModel = 'custom'
  }
  local.temperature = s.temperature ?? 0.3
  local.maxTokens = s.maxTokens || 16384
  local.contextMaxTokens = s.contextMaxTokens || 1000000
  local.thinkingMode = s.thinkingMode || 'thinking'
  // 压缩
  local.compressEnabled = !!s.compressEnabled
  local.compressProvider = s.compressProvider || ''
  local.compressBaseURL = s.compressBaseURL || ''
  local.compressApiKey = s.compressApiKey || ''
  local.compressModel = s.compressModel || ''
  local.compressModelCustom = ''
  const compressModels = modelsMap.value[local.compressProvider] || []
  if (local.compressModel && compressModels.length > 0 && !compressModels.includes(local.compressModel) && local.compressModel !== 'custom') {
    local.compressModelCustom = local.compressModel
    local.compressModel = 'custom'
  }
  local.compressThinkingMode = s.compressThinkingMode || 'non-thinking'
  // Agent
  local.maxIterations = s.maxIterations || 50
  local.maxParallel = s.maxParallel || 3
  local.reviewRetries = s.reviewRetries || 3
  local.requireHumanApprovalForDestructive = s.requireHumanApprovalForDestructive !== false
  local.reviewMode = s.reviewMode || 'auto'
  local.autoIterateOnRejection = !!s.autoIterateOnRejection
  local.autonomous = !!s.autonomous
  local.aiReview = !!s.aiReview
  local.luaTools = s.luaTools !== false
  local.autoCommit = s.autoCommit !== false
  local.searxngUrl = s.searxngUrl || ''
  local.ignoreDirsText = (s.ignoreDirs || []).join(', ')
  // 编辑器
  local.editorFontSize = s.editorFontSize || 14
  local.tabSize = s.tabSize || 2
  local.wordWrap = !!s.wordWrap
  local.hideMinimap = !!s.hideMinimap
  local.fontFamily = s.fontFamily || "'Cascadia Code', 'Fira Code', Consolas, monospace"
  local.editorFontBold = !!s.editorFontBold
  local.editorFontItalic = !!s.editorFontItalic
  local.editorFontUnderline = !!s.editorFontUnderline
  local.uiFontFamily = s.uiFontFamily || ''
  local.uiFontBold = !!s.uiFontBold
  local.uiFontItalic = !!s.uiFontItalic
  local.uiFontUnderline = !!s.uiFontUnderline
  // 外观
  local.theme = s.theme || 'dark'
  // 终端
  local.defaultShell = s.defaultShell || 'auto'
  local.termFontSize = s.termFontSize || 13
  local.termEncoding = s.termEncoding || 'auto'
  // MCP
  local.autoConnectMCP = s.autoConnectMCP !== false
}

// ─── 初始化 ───
onMounted(async () => {
  wsRoot.value = state.workspaceRoot || ''
  if (state.settingsLoaded) loadSettings()
  await loadModels()
  await loadInstructions()
  await loadPhilosophy()
  })

watch(() => state.settingsLoaded, (v) => { if (v) loadSettings() })

function reloadProjectInst() {
  loadInstructions()
}

const resetForm = () => {
  loadSettings()
}

const saveSettings = async () => {
  try {
    const settings = {
      ...state.settings,
      provider: local.provider,
      baseURL: local.baseURL,
      apiKey: local.apiKey,
      executeModel: local.executeModel === 'custom' ? local.executeModelCustom : local.executeModel,
      planModel: local.planModel === 'custom' ? local.planModelCustom : local.planModel,
      reviewModel: local.reviewModel === 'custom' ? local.reviewModelCustom : local.reviewModel,
      temperature: String(local.temperature),
      maxTokens: local.maxTokens,
      contextMaxTokens: local.contextMaxTokens,
      thinkingMode: local.thinkingMode,
      // 压缩
      compressEnabled: local.compressEnabled,
      compressProvider: local.compressProvider,
      compressBaseURL: local.compressBaseURL,
      compressApiKey: local.compressApiKey,
      compressModel: local.compressModel === 'custom' ? local.compressModelCustom : local.compressModel,
      compressThinkingMode: local.compressThinkingMode,
      // Agent
      maxIterations: local.maxIterations,
      maxParallel: local.maxParallel,
      maxReviewRetries: local.reviewRetries,
      reviewRetries: local.reviewRetries,
      requireHumanApprovalForDestructive: local.requireHumanApprovalForDestructive,
      reviewMode: local.reviewMode,
      autoIterateOnRejection: local.autoIterateOnRejection,
      autonomous: local.autonomous,
      aiReview: local.aiReview,
      luaTools: local.luaTools,
      autoCommit: local.autoCommit,
      searxngUrl: local.searxngUrl,
      ignoreDirs: local.ignoreDirsText.split(',').map(s => s.trim()).filter(Boolean),
      // 编辑器
      editorFontSize: local.editorFontSize,
      tabSize: local.tabSize,
      wordWrap: local.wordWrap,
      hideMinimap: local.hideMinimap,
      fontFamily: local.fontFamily,
      editorFontBold: local.editorFontBold,
      editorFontItalic: local.editorFontItalic,
      editorFontUnderline: local.editorFontUnderline,
      uiFontFamily: local.uiFontFamily,
      uiFontBold: local.uiFontBold,
      uiFontItalic: local.uiFontItalic,
      uiFontUnderline: local.uiFontUnderline,
      // 外观
      theme: local.theme,
      // 终端
      defaultShell: local.defaultShell,
      termFontSize: local.termFontSize,
      termEncoding: local.termEncoding,
      // MCP
      autoConnectMCP: local.autoConnectMCP,
    }
    await api.apiPut('/settings', settings)
    state.settings = settings
    applyTheme(local.theme)
    // 保存系统指令
    await api.saveInstructions('system', local.systemInstructions)
    // 保存项目指令
    await api.saveInstructions('project', local.projectInstructions)
    // 保存思想配置
    const roles = { ...local.philosophyRoles }
    if (local.mainAgentPhilosophy) roles['main'] = local.mainAgentPhilosophy
    await api.savePhilosophy({
      enabled: local.philosophyEnabled,
      selected: local.philosophySelected,
      roles: roles,
    })
    window.$toast('设置已保存', 'success')
    emit('close')
  } catch (err) {
    window.$toast('保存失败: ' + err.message, 'error')
  }
}
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
  width: 80vw;
  max-width: 720px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
}
.modal-header h2 { font-size: 16px; color: var(--text-primary); display:flex;align-items:center;gap:6px; }
.modal-close { background: none; border: none; color: var(--text-secondary); font-size: 20px; cursor: pointer; }
.modal-close:hover { color: var(--text-primary); }
.modal-body { flex: 1; display: flex; overflow: hidden; }
.settings-tabs {
  width: 100px;
  border-right: 1px solid var(--border-color);
  padding: 4px 0;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.settings-tabs button {
  display: block; width: 100%; text-align: left;
  padding: 7px 10px; background: none; border: none; border-right: 2px solid transparent;
  color: var(--text-secondary); font-size: 13px; cursor: pointer;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}
.settings-tabs button.active { background: var(--bg-active); color: var(--text-primary); border-right-color: var(--accent); }
.settings-tabs button:hover { color: var(--text-primary); background: var(--bg-hover); }
.settings-body { flex: 1; overflow: auto; padding: 12px 16px; }
.setting-group { margin-bottom: 16px; }
.group-title { font-size: 13px; font-weight: 600; color: var(--text-secondary); margin-bottom: 8px; padding-bottom: 4px; border-bottom: 1px solid var(--border-color); }
.setting-row { display: flex; align-items: center; gap: 8px; padding: 4px 0; }
.setting-row label { width: 120px; font-size: 13px; color: var(--text-primary); flex-shrink: 0; }
.setting-row-vertical { padding: 4px 0; }
.setting-row input[type="text"],
.setting-row input[type="password"],
.setting-row input[type="number"],
.setting-row select {
  flex: 1; background: var(--input-bg); border: 1px solid var(--border-color);
  color: var(--text-primary); padding: 4px 8px; font-size: 13px; outline: none; border-radius: 3px;
}
.setting-row input:focus, .setting-row select:focus { border-color: var(--accent); }
.setting-row input[type="range"] { flex: 1; }
.range-val { width: 30px; text-align: center; font-size: 12px; color: var(--text-secondary); }
.setting-row input[type="checkbox"] { width: 16px; height: 16px; }
.set-input-sm { background: var(--input-bg); border: 1px solid var(--border-color); color: var(--text-primary); padding: 4px 6px; font-size: 12px; outline: none; border-radius: 3px; width: 100px; }
.set-input-sm.flex-1 { flex: 1; }

/* ── 主题卡片 ── */
.theme-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin: 8px 0; }
.theme-card { border: 2px solid var(--border-color); border-radius: var(--border-radius-lg); padding: 10px; cursor: pointer; transition: all 0.15s; background: var(--bg-primary); }
.theme-card:hover { border-color: var(--text-muted); transform: translateY(-1px); box-shadow: var(--shadow-sm); }
.theme-card.selected { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
.theme-preview { height: 56px; border-radius: 4px; overflow: hidden; display: flex; flex-direction: row; margin-bottom: 6px; }
.tp-activity { width: 10px; flex-shrink: 0; }
.tp-main { flex: 1; display: flex; flex-direction: row; }
.tp-sidebar { width: 28px; }
.tp-editor { flex: 1; padding: 6px 4px; display: flex; flex-direction: column; gap: 3px; }
.tp-line { height: 4px; border-radius: 2px; }
.tp-line-accent { height: 3px; }
.tp-accent-bar { width: 4px; flex-shrink: 0; }
.theme-name { font-size: 13px; font-weight: 600; color: var(--text-primary); text-align: center; }
.theme-font { font-size: 10px; color: var(--text-muted); text-align: center; margin-top: 2px; }

/* ── 指令 ── */
.inst-textarea {
  width: 100%; box-sizing: border-box;
  background: var(--input-bg); border: 1px solid var(--border-color);
  color: var(--text-primary); padding: 6px 8px; font-size: 13px;
  outline: none; border-radius: 3px; font-family: var(--font-code, monospace);
  resize: vertical; min-height: 60px;
}
.inst-textarea:focus { border-color: var(--accent); }
.project-inst-hint { font-size: 12px; color: var(--text-muted); display:flex;align-items:center;gap:6px; }
.project-inst-hint code { background: var(--bg-tertiary); padding: 1px 4px; border-radius: 3px; font-size: 11px; }
.btn-xs { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 2px 8px; font-size: 11px; cursor: pointer; border-radius: 3px; }
.btn-xs:hover { background: var(--bg-hover); }

/* ── 思想 ── */
.classics-list { display: flex; flex-wrap: wrap; gap: 6px; padding: 4px 0; }
.classic-item { display: flex; align-items: center; gap: 4px; font-size: 13px; color: var(--text-primary); cursor: pointer; padding: 4px 8px; border-radius: 4px; background: var(--bg-tertiary); }
.classic-item:hover { background: var(--bg-hover); }
.role-phil-label { font-size: 12px; font-weight: 600; color: var(--text-secondary); margin-bottom: 2px; }

.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 10px 16px; border-top: 1px solid var(--border-color); }
.btn-secondary { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 6px 16px; cursor: pointer; border-radius: 3px; }
.btn-primary { background: var(--accent); border: none; color: #fff; padding: 6px 16px; cursor: pointer; border-radius: 3px; }
</style>
