<template>
  <div class="modal-overlay" @click.self="$emit('close')">
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
              </div>
              <div class="setting-row">
                <label>审核模型</label>
                <select v-model="local.reviewModel">
                  <option value="" disabled>选择模型</option>
                  <option v-for="m in modelsForProvider" :key="m" :value="m">{{ m }}</option>
                  <option value="custom">自定义</option>
                </select>
              </div>
              <div class="setting-row">
                <label>温度</label>
                <input type="range" min="0" max="2" step="0.1" v-model.number="local.temperature" />
                <span class="range-val">{{ local.temperature }}</span>
              </div>
              <div class="setting-row">
                <label>最大 Token</label>
                <input type="number" v-model.number="local.maxTokens" min="256" max="128000" />
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
                  <option value="custom">自定义</option>
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
                <label>自动审核</label>
                <input type="checkbox" v-model="local.autoReview" />
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

          <!-- ═══ MCP/技能 ═══ -->
          <div v-if="activeTab === 'mcp'">
            <!-- 模式切换：简单模式 / JSON 编辑模式 -->
            <div class="mcp-mode-toggle">
              <button :class="{ active: mcpEditMode === 'simple' }" @click="mcpEditMode='simple'">
                <SvgIcon name="list" :size="12" /> 列表视图
              </button>
              <button :class="{ active: mcpEditMode === 'json' }" @click="mcpEditMode='json'">
                <SvgIcon name="code" :size="12" /> JSON 编辑
              </button>
            </div>

            <!-- ═══ 列表视图 ═══ -->
            <div v-if="mcpEditMode === 'simple'">
              <div class="setting-group">
                <div class="group-title">MCP 服务器配置</div>
                <div class="setting-row">
                  <label>自动连接</label>
                  <input type="checkbox" v-model="local.autoConnectMCP" />
                </div>
                <div class="mcp-table">
                  <div class="mcp-th-row">
                    <span class="mcp-th">名称</span>
                    <span class="mcp-th">命令</span>
                    <span class="mcp-th">参数</span>
                    <span class="mcp-th">层级</span>
                    <span class="mcp-th mcp-th-action">操作</span>
                  </div>
                  <div v-for="(mcp, idx) in localMcpList" :key="mcp.name + idx" class="mcp-tr-row">
                    <input v-model="mcp.name" class="mcp-td-input" placeholder="名称" />
                    <input v-model="mcp.command" class="mcp-td-input" placeholder="命令" />
                    <input :value="(mcp.args||[]).join(' ')"
                           @input="mcp.args = $event.target.value.split(/\s+/).filter(Boolean)"
                           class="mcp-td-input" placeholder="参数 (空格分隔)" />
                    <select v-model="mcp.level" class="mcp-td-select">
                      <option value="user">用户级</option>
                      <option value="project">项目级</option>
                    </select>
                    <div class="mcp-td-actions">
                      <button class="mcp-row-btn expand-btn" @click="toggleMcpExpand(idx)" title="展开 JSON">
                        <SvgIcon name="code" :size="12" />
                      </button>
                      <button class="mcp-row-btn del-btn" @click="localMcpList.splice(idx, 1)" title="删除">×</button>
                    </div>
                    <!-- 展开的 JSON 编辑行 -->
                    <div v-if="mcp._expanded" class="mcp-json-row" :style="{ gridColumn: '1 / -1' }">
                      <div class="mcp-json-editor">
                        <div class="mcp-json-label">完整配置 (JSON)：</div>
                        <textarea class="mcp-json-textarea"
                                  :value="formatMcpJson(mcp)"
                                  @input="updateMcpFromJson(idx, $event.target.value)"
                                  rows="5"></textarea>
                      </div>
                    </div>
                  </div>
                  <div v-if="localMcpList.length === 0" class="mcp-empty">暂无 MCP 配置 — 点击下方添加</div>
                </div>
                <div class="add-mcp-row">
                  <button class="add-mcp-btn" @click="addMcpRow">
                    <SvgIcon name="plus" :size="12" /> 添加 MCP 服务器
                  </button>
                  <button class="add-mcp-btn-secondary" @click="loadMcpFromServer">
                    <SvgIcon name="refresh" :size="12" /> 从服务器加载
                  </button>
                </div>
              </div>

              <div class="setting-group" style="margin-top:16px">
                <div class="group-title">技能启用</div>
                <div v-if="localSkills.length === 0" class="mcp-empty">暂无已安装技能</div>
                <div v-for="(sk, idx) in localSkills" :key="sk.name" class="skill-row">
                  <input type="checkbox" v-model="sk.enabled" />
                  <SvgIcon name="code" :size="14" />
                  <span class="skill-name">{{ sk.name }}</span>
                  <span class="skill-desc">{{ sk.description }}</span>
                  <button class="skill-remove-btn" @click="localSkills.splice(idx, 1)">×</button>
                </div>
              </div>
            </div>

            <!-- ═══ JSON 编辑模式 ═══ -->
            <div v-if="mcpEditMode === 'json'">
              <div class="setting-group">
                <div class="group-title">MCP 配置 (JSON 全量编辑)</div>
                <div class="setting-row-vertical">
                  <div class="mcp-json-tip">
                    完整 MCP 服务器配置，支持 name / command / args / env / enabled 等字段。
                    修改后需保存设置生效。
                    <a href="https://modelcontextprotocol.io" target="_blank" class="mcp-json-doc-link">查看文档 →</a>
                  </div>
                  <textarea class="mcp-json-editor-full"
                            v-model="mcpJsonText"
                            @input="mcpJsonDirty = true"
                            rows="14"
                            spellcheck="false"></textarea>
                  <div class="mcp-json-actions">
                    <button class="btn-xs" @click="formatMcpJsonText">格式化</button>
                    <button class="btn-xs" @click="validateMcpJson">验证</button>
                    <span v-if="mcpJsonValid === true" class="mcp-json-valid">✓ JSON 格式正确</span>
                    <span v-else-if="mcpJsonValid === false" class="mcp-json-invalid">✗ JSON 格式错误</span>
                  </div>
                </div>
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
  { id: 'mcp', label: 'MCP/技能' },
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
  temperature: 0.7,
  maxTokens: 4096,
  contextMaxTokens: 1000000,
  thinkingMode: '',
  // 压缩
  compressEnabled: false,
  compressProvider: '',
  compressBaseURL: '',
  compressApiKey: '',
  compressModel: '',
  compressThinkingMode: 'non-thinking',
  // Agent
  maxIterations: 50,
  maxParallel: 3,
  reviewRetries: 3,
  requireHumanApprovalForDestructive: true,
  autoReview: false,
  autoIterateOnRejection: false,
  autonomous: false,
  aiReview: false,
  luaTools: true,
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
  // MCP
  autoConnectMCP: true,
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
  { id: 'cute', label: '分分可爱风', fontDesc: 'Nunito + Fira Code',
    colors: { activity: '#8e3a5a', sidebar: '#fce4ec', editor: '#fff5f7',
              line1: '#f8d7e0', line2: '#e8439344', line3: '#e8b8c8', accent: '#e84393' } },
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

function onProviderChange() {
  // 如果当前模型不在新服务商列表中，重置
  const models = modelsMap.value[local.provider] || []
  if (models.length > 0) {
    if (!models.includes(local.executeModel)) local.executeModel = models[0]
    if (!models.includes(local.planModel)) local.planModel = models[0]
    if (!models.includes(local.reviewModel)) local.reviewModel = models[0]
  }
}

function onCompressProviderChange() {
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

// ─── MCP / Skills ───
const localMcpList = ref([])
const localSkills = ref([])
const mcpEditMode = ref('simple')
const mcpJsonText = ref('')
const mcpJsonDirty = ref(false)
const mcpJsonValid = ref(null) // null=未知, true=正确, false=错误

function loadMcpSkills() {
  try {
    const saved = JSON.parse(localStorage.getItem('paircode-mcp-config') || '[]')
    localMcpList.value = saved.map(m => ({ ...m, level: m.level || 'user', _expanded: false }))
  } catch {
    localMcpList.value = []
  }
  // 从 JSON 文本同步
  syncMcpJsonFromList()
}

function syncMcpJsonFromList() {
  const obj = {}
  for (const m of localMcpList.value) {
    const entry = { name: m.name, command: m.command, args: m.args || [] }
    if (m.env) entry.env = m.env
    obj[m.name] = entry
  }
  mcpJsonText.value = JSON.stringify(obj, null, 2)
}

function syncMcpListFromJson() {
  try {
    const obj = JSON.parse(mcpJsonText.value)
    const list = []
    for (const [name, entry] of Object.entries(obj)) {
      list.push({
        name: entry.name || name,
        command: entry.command || '',
        args: entry.args || [],
        env: entry.env || {},
        level: entry.level || 'user',
        _expanded: false,
      })
    }
    localMcpList.value = list
    mcpJsonValid.value = true
    return true
  } catch (e) {
    mcpJsonValid.value = false
    return false
  }
}

function formatMcpJson(mcp) {
  const obj = {
    name: mcp.name,
    command: mcp.command,
    args: mcp.args || [],
  }
  if (mcp.env && Object.keys(mcp.env).length > 0) {
    obj.env = mcp.env
  }
  return JSON.stringify(obj, null, 2)
}

function updateMcpFromJson(idx, text) {
  try {
    const parsed = JSON.parse(text)
    if (parsed.name) localMcpList.value[idx].name = parsed.name
    if (parsed.command) localMcpList.value[idx].command = parsed.command
    if (parsed.args) localMcpList.value[idx].args = parsed.args
    if (parsed.env) localMcpList.value[idx].env = parsed.env
    if (parsed.level) localMcpList.value[idx].level = parsed.level
    localMcpList.value[idx]._jsonError = false
  } catch {
    localMcpList.value[idx]._jsonError = true
  }
  saveMcpConfig()
}

function toggleMcpExpand(idx) {
  localMcpList.value[idx]._expanded = !localMcpList.value[idx]._expanded
}

function addMcpRow() {
  localMcpList.value.push({
    name: '新服务器',
    command: 'npx',
    args: ['-y', '@modelcontextprotocol/server-...'],
    env: {},
    level: 'user',
    _expanded: false,
  })
  syncMcpJsonFromList()
}

async function loadMcpFromServer() {
  try {
    const results = await api.apiGet('/mcp/list', { level: 'all' })
    if (results && results.length > 0) {
      for (const mcp of results) {
        const exists = localMcpList.value.find(m => m.name === mcp.name)
        if (!exists) {
          localMcpList.value.push({
            name: mcp.name,
            command: mcp.command,
            args: mcp.args || [],
            env: {},
            level: mcp.level || 'user',
            _expanded: false,
          })
        }
      }
      saveMcpConfig()
      window.$toast?.('已加载 ' + results.length + ' 个 MCP 配置', 'success')
    } else {
      window.$toast?.('服务器暂无 MCP 配置', 'info')
    }
  } catch (err) {
    window.$toast?.('加载失败: ' + err.message, 'error')
  }
}

function saveMcpConfig() {
  try {
    const data = localMcpList.value.map(m => ({
      name: m.name,
      command: m.command,
      args: m.args || [],
      env: m.env || {},
      level: m.level || 'user',
    }))
    localStorage.setItem('paircode-mcp-config', JSON.stringify(data))
    syncMcpJsonFromList()
  } catch {}
}

function formatMcpJsonText() {
  try {
    const obj = JSON.parse(mcpJsonText.value)
    mcpJsonText.value = JSON.stringify(obj, null, 2)
    mcpJsonValid.value = true
    syncMcpListFromJson()
  } catch {
    mcpJsonValid.value = false
  }
}

function validateMcpJson() {
  try {
    JSON.parse(mcpJsonText.value)
    mcpJsonValid.value = true
  } catch {
    mcpJsonValid.value = false
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
  local.planModel = s.planModel || ''
  local.reviewModel = s.reviewModel || ''
  local.temperature = s.temperature ?? 0.7
  local.maxTokens = s.maxTokens || 4096
  local.contextMaxTokens = s.contextMaxTokens || 1000000
  local.thinkingMode = s.thinkingMode || ''
  // 压缩
  local.compressEnabled = !!s.compressEnabled
  local.compressProvider = s.compressProvider || ''
  local.compressBaseURL = s.compressBaseURL || ''
  local.compressApiKey = s.compressApiKey || ''
  local.compressModel = s.compressModel || ''
  local.compressThinkingMode = s.compressThinkingMode || 'non-thinking'
  // Agent
  local.maxIterations = s.maxIterations || 50
  local.maxParallel = s.maxParallel || 3
  local.reviewRetries = s.reviewRetries || 3
  local.requireHumanApprovalForDestructive = s.requireHumanApprovalForDestructive !== false
  local.autoReview = !!s.autoReview
  local.autoIterateOnRejection = !!s.autoIterateOnRejection
  local.autonomous = !!s.autonomous
  local.aiReview = !!s.aiReview
  local.luaTools = s.luaTools !== false
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
  loadMcpSkills()
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
      planModel: local.planModel === 'custom' ? '' : local.planModel,
      reviewModel: local.reviewModel === 'custom' ? '' : local.reviewModel,
      temperature: String(local.temperature),
      maxTokens: local.maxTokens,
      contextMaxTokens: local.contextMaxTokens,
      thinkingMode: local.thinkingMode,
      // 压缩
      compressEnabled: local.compressEnabled,
      compressProvider: local.compressProvider,
      compressBaseURL: local.compressBaseURL,
      compressApiKey: local.compressApiKey,
      compressModel: local.compressModel,
      compressThinkingMode: local.compressThinkingMode,
      // Agent
      maxIterations: local.maxIterations,
      maxParallel: local.maxParallel,
      maxReviewRetries: local.reviewRetries,
      reviewRetries: local.reviewRetries,
      requireHumanApprovalForDestructive: local.requireHumanApprovalForDestructive,
      autoReview: local.autoReview,
      autoIterateOnRejection: local.autoIterateOnRejection,
      autonomous: local.autonomous,
      aiReview: local.aiReview,
      luaTools: local.luaTools,
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

/* ── MCP / Skills ── */
.mcp-mode-toggle {
  display: flex;
  gap: 4px;
  margin-bottom: 12px;
  background: var(--bg-tertiary);
  border-radius: 6px;
  padding: 3px;
}
.mcp-mode-toggle button {
  flex: 1;
  background: none;
  border: none;
  color: var(--text-secondary);
  padding: 5px 10px;
  font-size: 12px;
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}
.mcp-mode-toggle button.active {
  background: var(--bg-primary);
  color: var(--text-primary);
  box-shadow: 0 1px 2px rgba(0,0,0,0.1);
}

/* ── MCP 表格 ── */
.mcp-table {
  display: flex;
  flex-direction: column;
  gap: 1px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  overflow: hidden;
  margin-bottom: 8px;
}
.mcp-th-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1.5fr 90px 60px;
  background: var(--bg-tertiary);
  padding: 6px 8px;
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 600;
}
.mcp-th { padding: 0 4px; }
.mcp-th-action { text-align: center; }

.mcp-tr-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1.5fr 90px 60px;
  padding: 4px 6px;
  gap: 4px;
  align-items: center;
  border-top: 1px solid var(--border-color);
  position: relative;
}
.mcp-tr-row:hover { background: var(--bg-hover); }

.mcp-td-input {
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 4px 6px;
  font-size: 12px;
  outline: none;
  border-radius: 3px;
  width: 100%;
  box-sizing: border-box;
}
.mcp-td-input:focus { border-color: var(--accent); }
.mcp-td-select {
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 4px 4px;
  font-size: 11px;
  outline: none;
  border-radius: 3px;
  width: 100%;
}
.mcp-td-actions {
  display: flex;
  gap: 2px;
  justify-content: center;
}
.mcp-row-btn {
  background: none;
  border: 1px solid transparent;
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 14px;
  display: flex;
  align-items: center;
}
.mcp-row-btn:hover { background: var(--bg-hover); color: var(--text-primary); }
.del-btn:hover { color: #c03; border-color: #c03; }

/* ── 展开的 JSON 编辑 ── */
.mcp-json-row {
  padding: 6px;
  background: var(--bg-tertiary);
  border-top: 1px solid var(--border-color);
}
.mcp-json-editor { display: flex; flex-direction: column; gap: 4px; }
.mcp-json-label { font-size: 11px; color: var(--text-muted); }
.mcp-json-textarea {
  width: 100%;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 6px 8px;
  font-size: 11px;
  font-family: var(--font-code);
  outline: none;
  border-radius: 3px;
  resize: vertical;
  min-height: 80px;
  box-sizing: border-box;
}
.mcp-json-textarea:focus { border-color: var(--accent); }

/* ── 全量 JSON 编辑 ── */
.mcp-json-tip {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 6px;
  line-height: 1.5;
}
.mcp-json-doc-link {
  color: var(--accent);
  text-decoration: none;
  font-weight: 500;
}
.mcp-json-doc-link:hover { text-decoration: underline; }
.mcp-json-editor-full {
  width: 100%;
  background: var(--input-bg);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 10px 12px;
  font-size: 12px;
  font-family: var(--font-code);
  outline: none;
  border-radius: 6px;
  resize: vertical;
  min-height: 200px;
  box-sizing: border-box;
  line-height: 1.5;
  tab-size: 2;
}
.mcp-json-editor-full:focus { border-color: var(--accent); }
.mcp-json-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}
.mcp-json-valid { font-size: 11px; color: #6a9955; }
.mcp-json-invalid { font-size: 11px; color: #c03; }

.btn-xs {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 3px 10px;
  font-size: 11px;
  cursor: pointer;
  border-radius: 3px;
}
.btn-xs:hover { background: var(--bg-hover); }

.add-mcp-row { display: flex; gap: 6px; align-items: center; margin-top: 6px; }
.add-mcp-btn {
  background: var(--accent);
  border: none;
  color: #fff;
  padding: 6px 14px;
  cursor: pointer;
  border-radius: 5px;
  font-size: 12px;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 4px;
}
.add-mcp-btn:hover { filter: brightness(1.1); }
.add-mcp-btn-secondary {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 6px 14px;
  cursor: pointer;
  border-radius: 5px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
}
.add-mcp-btn-secondary:hover { color: var(--text-primary); }

/* ── Skills ── */
.mcp-empty {
  font-size: 11px;
  color: var(--text-muted);
  padding: 12px;
  text-align: center;
}
.skill-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 6px;
  font-size: 12px;
  border-radius: 3px;
}
.skill-row:hover { background: var(--bg-hover); }
.skill-name { font-size: 13px; color: var(--text-primary); flex-shrink: 0; }
.skill-desc { font-size: 11px; color: var(--text-muted); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.skill-remove-btn { background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 14px; padding: 0 4px; }
.skill-remove-btn:hover { color: #c03; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 10px 16px; border-top: 1px solid var(--border-color); }
.btn-secondary { background: var(--bg-tertiary); border: 1px solid var(--border-color); color: var(--text-primary); padding: 6px 16px; cursor: pointer; border-radius: 3px; }
.btn-primary { background: var(--accent); border: none; color: #fff; padding: 6px 16px; cursor: pointer; border-radius: 3px; }
</style>
