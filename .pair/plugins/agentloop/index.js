// ═══════════════════════════════════════════════════════════════
// agentloop — Agent 循环 JS 实现（agentloop 核心外置）
//
// ★ 2026-08-19 架构升级：循环策略从 Go（internal/agent/loop.go）外置到 JS。
//   Go 保留能力（Provider.Chat / Registry.Execute / emit / persist /
//   approve / buildCallContext / 绕圈检测），经能力代理对象注入：
//
//     loop.llm.chat(msgs, tools, onChunk) → assistant
//     loop.tools.list() / loop.tools.run(name, argsJson)
//     loop.tools.runParallel([{id,name,args}]) → 结果|null（纯只读并行）
//     loop.events.emit(event)
//     loop.persist.batch(msgs)
//     loop.approve.ask(tc) → {approved, feedback}  // 审核门（动作执行；驳回记录进共享状态）
//     loop.approve.state.get()/set(obj)             // 共享审核状态（最近驳回/历史，不计数）
//     loop.context.build(msgs, ephemeral) → callMsgs
//     loop.compact(msgs) → msgs
//     loop.circling.track(name, args, failed) / loop.circling.detect()
//     loop.store.get(key) / loop.store.set(key, value)
//     loop.delegate.run({task, system?, maxIterations?, agentName?}) → 子 agent
//     loop.ctrl.*（暂停/停止/队列/钩子/日志/step 边界）
//
//   本插件实现循环业务（策略层）：
//     - turn/step 双层循环（一次 Run = 一个 turn，每轮 LLM+工具 = 一个 step）
//     - 流式事件分发（thinking/content 增量透传）
//     - 工具执行 + 审核决策（loop.approve.ask + 共享状态 approve.state 增强）
//     - 自然终止检测（无 tool_call + 有正文 → 完成/跟进/下一阶段）
//     - content-only 防护 / 绕圈检测注入
//     - 每轮持久化（loop.persist.batch）
//
//   可回退：停用/删除本插件 → Loop.Run 自动还原 Go 默认循环。
//
// ═══════════════════════════════════════════════════════════════

return {
  name: 'agentloop',
  purpose: 'Agent 循环 JS 实现（核心外置：策略在 JS，能力在 Go；含配置注册化）',
  inject: ['logger'],
  apply(ctx, config) {
    // ═══════════════════════════════════════════════════════════
    // 配置注册（分散化：本插件承载 Agent 相关的全部配置）——
    //   · ai 组：服务商/模型/密钥/温度/token（binding → AppSettings 顶层）
    //   · agentloop 组：循环参数覆盖 + Agent 行为（旧字段非 binding 存
    //     pluginSettings['agentloop'] 保持业务读取兼容；新字段 binding 全局）
    //   · instructions 组：系统级指令（binding → systemInstructions）
    // ═══════════════════════════════════════════════════════════
    // ── AI：多例配置列表（★ 2026-08-20 改变模式：不再单例表单+预设分组，
    //    主视图直接列出已添加的配置；点「添加新配置」弹出表单去设置模型和 Key）──
    // type='preset-manager'：SettingsModal 在 AI tab 内渲染「AI 配置」列表
    //   （添加/编辑/应用/删除），数据经 /api/ai-presets（config/ai-presets.json）。
    //   每条配置 = 完整 AI 配置快照（provider/baseURL/apiKey/模型/参数）。
    //   「应用」只把配置名写入 settings.preset——装配时按 preset 从 ai-presets.json
    //   展开整套配置（key/模型/参数唯一来源），settings 不再冗余存 key/模型。
    //   settings 顶层现有值仅兜底（兼容无预设旧配置）；应用某条配置后装配即用该配置。
    ctx.registerSettings({
      key: 'ai',
      title: 'AI',
      fields: [
          { name: 'presets', label: 'AI 配置', type: 'preset-manager',
            hint: 'AI 配置列表：每条 = 一个服务商 + 该服务商的 API Key。模型在对话面板中按会话选择，不在 AI 设置中指定。点「＋ 添加新配置」填写服务商和 Key；点「应用」整套配置生效。',
            // ★ 2026-09-01 schema 驱动：AI 配置表单字段由插件注册，前端按此动态渲染
            presetFields: [
              { name: 'provider', label: '服务商', type: 'select', source: 'providers', required: true, hint: '该 Key 所属的服务商（服务商地址/模型在「服务商」面板维护）' },
              { name: 'apiKey', label: 'API Key', type: 'password', required: true, placeholder: 'sk-…', hint: '该服务商的 API Key' },
            ], },
          // 注意：配置名称（name）为每配置标识，由 PresetManager 内置渲染不在此注册；
          // model 选择对话面板按会话独立选择，不在此注册。
      ],
    })

    // ── Agent：循环参数覆盖 + 行为 ──
    const reg = ctx.registerSettings({
      key: 'agentloop',
      title: 'Agent',
      fields: [
        { name: 'systemAppend', label: '系统提示词追加', type: 'textarea', default: '', hint: '追加到系统提示词末尾（如行为规范/角色设定）' },
        { name: 'maxIterations', label: '最大迭代数', type: 'number', default: 0, hint: '0=不覆盖（默认 50）' },
        { name: 'autonomous', label: '自主模式', type: 'checkbox', default: false, hint: '勾选=强制开启自主（不勾=跟随全局开关，不再强制关闭）' },
        { name: 'maxAutonomousMinutes', label: '自主时间预算（分钟）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'checkpointInterval', label: '检查点间隔（迭代数）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'reviewMode', label: '审核模式', type: 'select', default: '', options: ['', 'auto', 'manual', 'off'], hint: '空=不覆盖（auto=AI审核/manual=手动审批/off=放行）' },
        { name: 'reviewBlacklist', label: '审核黑名单', type: 'text', default: '', hint: '逗号分隔工具名（命中需审核）' },
        { name: 'reviewWhitelist', label: '审核白名单', type: 'text', default: '', hint: '逗号分隔工具名（命中跳过审核，黑名单优先）' },
        { name: 'autoIterateOnRejection', label: '拒绝后自动迭代', type: 'checkbox', binding: 'autoIterateOnRejection' },
        { name: 'ignoreDirs', label: '忽略目录', type: 'tags', binding: 'ignoreDirs',
          hint: '逗号分隔（node_modules, dist, .git…）' },
        { name: 'stagedToolGroups', label: '首步极简工具候选组', type: 'textarea', default: '',
          hint: '每行一组（组内逗号分隔，语义等价工具）；首步只注入这些工具，第 2 步起恢复全量。命中规则：每组取工具面中第一个存在的名字。空=默认组（read/write/edit/搜索/命令等 8 组）。' },
      ],
    })

    // ── 指令：系统级 ──
    ctx.registerSettings({
      key: 'instructions',
      title: '指令',
      fields: [
        { name: 'systemInstructions', label: '系统级指令（所有工作区共享）', type: 'textarea', binding: 'systemInstructions',
          placeholder: '输入全局系统指令…' },
      ],
    })

      // ── 服务商：维护服务商列表（名称/API URL（完整端点）/模型列表 + 每模型参数）──
      // type='provider-manager'：SettingsModal 渲染 CRUD 面板，数据经 /api/models（config/models.json）。
      // 模型参数（温度/思考档位/输出上限/上下文窗口/多模态）在服务商编辑表单内逐模型维护，
      // 存 settings.json 顶层 modelParams（装配器按 服务商+模型 精确匹配）。
      // ★ 2026-09-01 Key 回归 AI 配置：API Key 在「AI 配置」中按配置填写，服务商不再维护 Key。
      // ★ 2026-08-21 模型参数区 schema 驱动：字段定义全部在本 modelParamFields 声明，
      //   前端 ProviderManager 按此动态渲染（新增参数无需改前端组件）。
      // AI tab 的 provider 下拉（optionsSource='providers'）与模型下拉（optionsSource='models'）均来自此处维护的数据。
      ctx.registerSettings({
        key: 'providers',
        title: '服务商',
        fields: [
          { name: 'providers', label: '服务商列表', type: 'provider-manager',
            hint: '维护服务商：名称、API URL（完整端点，含 /chat/completions，直接作为请求地址）、可用模型列表，及每模型独立参数（温度/思考/输出上限/上下文窗口/多模态）。API Key 请在「AI 配置」中填写。AI tab 的下拉与联动均来自此处。',
            // ★ 2026-08-21 添加模型区 schema 驱动：modelEditor 声明组件配置（label/placeholder），
            //   前端 ProviderManager 按此渲染模型编辑器（与 modelParamFields 同层，新增配置无需改前端组件）。
            modelEditor: { label: '可用模型（回车或逗号分隔添加；支持整段粘贴）', placeholder: '输入模型名，回车添加…' },
            modelParamFields: [
              { name: 'temperature', label: '温度', type: 'select', options: ['', '0', '0.1', '0.2', '0.3', '0.4', '0.5', '0.6', '0.7', '0.8', '0.9', '1.0', '1.2', '1.5', '2.0'], hint: '温度（随机性），空=默认' },
              { name: 'thinkingMode', label: '思考档位', type: 'select', options: ['', 'none', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max'], hint: '思考档位（OpenAI 定义），空=默认' },
              { name: 'maxTokens', label: '输出 Token', type: 'number', min: 0, step: 1024, hint: '最大输出 Token（0=默认）' },
              { name: 'contextMaxTokens', label: '上下文窗口', type: 'number', min: 0, step: 4096, hint: '上下文窗口（0=默认）' },
              { name: 'multimodal', label: '多模态', type: 'checkbox', hint: '勾选=该模型支持图片输入（对话中可直接粘贴/拖拽图片，agentloop 自动以多模态格式发送）' },
            ],
          },
        ],
      })

    // ═══════════════════════════════════════════════════════════
    // ★ 配置消费插件化（2026-08-19）：LLM Provider 参数装配器
    //   Go 内核（buildWebProvider/Review/Plan/工具集分析）不再直接读配置业务字段，
    //   统一经 agent.ResolveProviderParams() → 本装配器（ctx.providerFactory）获取。
    //   本装配器是 AI 连接参数的唯一业务来源：读 ai 组配置（binding 顶层，经
    //   ctx.app.settings 快照）→ 返回 overrides（非空字段覆盖）。
    // ═══════════════════════════════════════════════════════════
    ctx.providerFactory.register((current) => {
      const s = (ctx.app && ctx.app.settings) || {};
      const over = {};
      // ★ baseURL/apiKey/模型：Go 端 ResolveProviderParams 已按「激活预设 → settings 顶层
      //   → models.json 服务商」展开注入 current；此处仅当 current 为空时用 settings 兜底。
      // ★ 统一模型（2026-08-21）：不再拆分 规划/审核 模型，Go 端 planModel/reviewModel 已
      //   统一为执行模型，此处直接透传。
      const baseURL = (current.baseURL || s.baseURL || '').trim();
      const apiKey = (current.apiKey || s.apiKey || '').trim();
      const model = (current.model || s.executeModel || s.model || '').trim();
      const planModel = (current.planModel || s.planModel || '').trim();
      const reviewModel = (current.reviewModel || s.reviewModel || '').trim();
      if (baseURL) over.baseURL = baseURL;
      if (apiKey) over.apiKey = apiKey;
      if (model) over.model = model;
      if (planModel) over.planModel = planModel;
      if (reviewModel) over.reviewModel = reviewModel;
      // ★ 2026-08-20 模型级参数（settings.modelParams[服务商][模型]）优先；无则回退全局（兼容旧配置）
      const provider = current.provider || s.provider || '';
      let mp = (s.modelParams && s.modelParams[provider] && s.modelParams[provider][model]) ||
               (current.modelParams && current.modelParams[provider] && current.modelParams[provider][model]) || null;
      // ★ 2026-08-21 兼容：provider key 编码损坏/改名（如 settings.json 被 GBK 保存污染）
      //   → 精确匹配失败后按「模型名唯一匹配」兜底（modelParams[任意provider][model]）。
      if (!mp && s.modelParams) {
        for (const pk of Object.keys(s.modelParams)) {
          const pm = s.modelParams[pk];
          if (pm && typeof pm === 'object' && pm[model]) { mp = pm[model]; break }
        }
      }
      if (mp) {
        if (mp.temperature !== undefined && mp.temperature !== null && mp.temperature !== '') {
          const t = parseFloat(mp.temperature);
          if (!isNaN(t) && t >= 0) over.temperature = t;
        }
        if (mp.thinkingMode) over.thinkingMode = mp.thinkingMode;
        if (mp.maxTokens && Number(mp.maxTokens) > 0) over.maxTokens = Number(mp.maxTokens);
        // ★ 2026-08-21 多模态：模型级参数标记该模型支持图片输入 → Provider 以多模态格式发送
        if (mp.multimodal === true) over.multimodal = true;
      }
      // ★ 上下文窗口层级：模型级 > 服务商级（models.json 每服务商 contextMaxTokens）> 全局
      const pctx = (current.providerContextMaxTokens && Number(current.providerContextMaxTokens) > 0)
        ? Number(current.providerContextMaxTokens) : 0;
      let cctx = (mp && mp.contextMaxTokens) ? Number(mp.contextMaxTokens) : 0;
      if (!(cctx > 0) && pctx > 0) cctx = pctx;
      if (!(cctx > 0) && s.contextMaxTokens) cctx = Number(s.contextMaxTokens);
      if (cctx > 0) over.contextMaxTokens = cctx;
      // ★ 全局温度/思考/输出（模型级与服务商级未配置时兜底，兼容旧配置）
      if (!(mp && mp.temperature !== undefined && mp.temperature !== null && mp.temperature !== '')) {
        if (s.temperature !== undefined && s.temperature !== null && s.temperature !== '') {
          const t = parseFloat(s.temperature);
          if (!isNaN(t) && t >= 0) over.temperature = t;
        }
      }
      if (!(mp && mp.thinkingMode) && s.thinkingMode) over.thinkingMode = s.thinkingMode;
      if (!(mp && mp.maxTokens && Number(mp.maxTokens) > 0) && s.maxTokens && Number(s.maxTokens) > 0) {
        over.maxTokens = Number(s.maxTokens);
      }
      return over;
    });

    // ── 参数级装配（保留）：CreateLoop 时覆盖装配参数（提示词/迭代/审核模式）──
    ctx.loopFactory.register((opts) => {
      // ★ 2026-08-21 修复：动态读取配置（registerSettings 返回的 value 是 apply 时
      //   静态快照，设置面板保存后不刷新 → 装配器一直用旧值 → maxIterations 等
      //   设置保存不生效）。改为每次 Create 时实时读 pluginSettings.agentloop。
      const cfg = ctx.getSettings('agentloop') || {};
      const over = {};
      if (typeof cfg.systemAppend === 'string' && cfg.systemAppend) {
        over.system = (opts.system || '') + '\n\n' + cfg.systemAppend;
      }
      if (cfg.maxIterations != null && Number(cfg.maxIterations) > 0) over.maxIterations = Number(cfg.maxIterations);
      if (cfg.maxContextTokens != null && Number(cfg.maxContextTokens) > 0) over.maxContextTokens = Number(cfg.maxContextTokens);
      // ★ 2026-08-19 修复：仅强制开启（true 才覆盖）——false 不再覆盖全局，
      //   消除「保存设置面板即强制关闭全局自主模式」的默认值缺陷。
      if (cfg.autonomous === true) over.autonomous = true;
      if (cfg.maxAutonomousMinutes != null && Number(cfg.maxAutonomousMinutes) > 0) over.maxAutonomousMinutes = Number(cfg.maxAutonomousMinutes);
      if (cfg.checkpointInterval != null && Number(cfg.checkpointInterval) > 0) over.checkpointInterval = Number(cfg.checkpointInterval);
      if (typeof cfg.reviewMode === 'string' && cfg.reviewMode) over.reviewMode = cfg.reviewMode;
      if (typeof cfg.reviewBlacklist === 'string' && cfg.reviewBlacklist) {
        over.reviewBlacklist = cfg.reviewBlacklist.split(/[,，]/).map(s => s.trim()).filter(Boolean);
      }
      if (typeof cfg.reviewWhitelist === 'string' && cfg.reviewWhitelist) {
        over.reviewWhitelist = cfg.reviewWhitelist.split(/[,，]/).map(s => s.trim()).filter(Boolean);
      }
      // ★ 2026-08-27 首步极简工具候选组（插件注册化配置）：textarea 每行一组、
      //   组内逗号分隔 → 数组数组传给 Go（Loop.StagedToolGroups，nil=默认组）
      if (typeof cfg.stagedToolGroups === 'string' && cfg.stagedToolGroups.trim()) {
        const groups = cfg.stagedToolGroups.split('\n')
          .map(l => l.trim()).filter(Boolean)
          .map(l => l.split(/[,，]/).map(s => s.trim()).filter(Boolean))
          .filter(g => g.length > 0);
        if (groups.length > 0) over.stagedToolGroups = groups;
      }
      // ai 组 contextMaxTokens（binding 顶层，经 ctx.app.settings 快照）→ 覆盖循环上下文窗口
      const aiTop = (ctx.app && ctx.app.settings) || {};
      const ctxMax = Number(aiTop.contextMaxTokens);
      if (ctxMax > 0) over.maxContextTokens = ctxMax;
      return over;
    });

    // ═══════════════════════════════════════════════════════════
    // ★ JS 循环实现（agentloop 核心外置）：Loop.Run 委托本 run() 驱动循环。
    //   Go 侧前置：msgs 组装（system+历史+任务）、staleMsg、工具精简、
    //   OnToolUpdate 桥、Run 开始时压缩一次——全部完成后再调本函数。
    //   返回 { msgs, error? }：msgs = 完整消息列表（Go 更新 History 并收尾）。
    // ═══════════════════════════════════════════════════════════
    ctx.loopFactory.registerLoop({
      id: 'agentloop',
      async run({ task, msgs, tools, meta, loop }) {
        const maxIter = meta.maxIterations || 30;
        const autonomous = !!meta.autonomous;
        const maxBudgetMin = meta.maxAutonomousMinutes || 0;

        // ── 运行期状态 ──
        let contentOnlyIters = 0;          // 连续 content-only 轮数
        const ephemeral = [];              // 临时消息（反馈/提示/跟进），每次 build 后清空
        // 自主时间预算（跨 Run 经 store 保留起点）
        let startTime = loop.store.get('autonomousStart');
        if (!startTime) {
          startTime = Date.now();
          loop.store.set('autonomousStart', startTime);
        }

        // 工具调用数统计（endStep 用）
        const REJ_DEFAULT = '用户拒绝了此操作。请勿重试该操作；改用其他方式达成目标，或先向用户说明你为何需要它。';
        const NUDGE_CONTENT = '[系统提示] 你已经连续三轮只输出文字而没有调用任何工具。如果任务已完成，直接自然总结；如还需继续，请调用工具推进。';

        // ═══════════════════════════════════════════════════════
        // ★ 策略外置常量（2026-08-27）：压缩/绕圈/审批的「判定策略」本插件自持
        //   （阈值/窗口/提示文本均可在此自定义；数据面/执行面在 Go）。
        // ═══════════════════════════════════════════════════════
        // 压缩策略：两档阈值 + 冷却 + 硬地板（对齐 Go 原实现默认值，可调）
        const COMPACT = { thresholdEarly: 0.45, thresholdFull: 0.90, cooldownEarly: 3, cooldownFull: 10, hardFloor: 120000 };
        // 绕圈检测策略：窗口 + 重复/失败阈值
        const CIRCLING = { window: 12, repeatStop: 3, failStop: 2 };

        // 压缩判定（策略 JS / 执行 Go compact.apply）：返回处理后的 msgs
        function maybeCompact(msgs) {
          if (!loop.compact || !loop.compact.estimate) return msgs; // 无数据面（回退路径）→ 不动
          const est = loop.compact.estimate(msgs);
          if (!est.maxContextTokens || est.maxContextTokens <= 0) return msgs; // 未配置窗口 → 压缩关闭
          if ((est.cooldown || 0) > 0) return msgs; // 冷却期（apply 后 Go 侧自动设置）
          let ratio = est.ratio;
          if (est.maxContextTokens > COMPACT.hardFloor && est.tokens >= COMPACT.hardFloor) {
            ratio = COMPACT.thresholdFull; // 绝对硬地板：超大窗口强制全量压缩
          }
          if (ratio < COMPACT.thresholdEarly) return msgs; // 未超任何阈值
          if (ratio < COMPACT.thresholdFull) {
            const r = loop.compact.apply(msgs, 'early');
            return (r && r.msgs) ? r.msgs : msgs;
          }
          const r = loop.compact.apply(msgs, 'full');
          return (r && r.dropped > 0 && r.msgs) ? r.msgs : msgs;
        }

        // 绕圈判定（策略 JS / 数据 Go circling.state）：
        // 从尾部倒扫「连续相同操作（间无其他操作）」——重复或连续失败超阈值 → 提示换思路
        function shortSig(sig) { return String(sig || '').replace(/\|/g, ' ').slice(0, 60); }
        function detectCircling() {
          if (!loop.circling || !loop.circling.state) return ''; // 无数据面 → 不检测
          const st = loop.circling.state();
          const n = st.length;
          if (n < 2) return '';
          const last = st[n - 1].sig;
          // 1. 连续相同签名（纯重复）
          let sameCount = 1;
          for (let i = n - 2; i >= 0 && st[i].sig === last; i--) sameCount++;
          if (sameCount >= CIRCLING.repeatStop) {
            if (loop.circling.clear) loop.circling.clear();
            return '[系统提示·打破死循环] 你已连续 ' + sameCount +
              ' 次执行同一操作 `' + shortSig(last) + '`，中间没有任何其他操作——像在原地绕圈。请停下来换思路：先读取当前状态确认事实，或换工具、换方式推进。别继续重复同一步。';
          }
          // 2. 连续相同签名+失败
          let failCount = 0;
          for (let i = n - 1; i >= 0 && st[i].sig === last; i--) {
            if (st[i].failed) failCount++; else break;
          }
          if (failCount >= CIRCLING.failStop) {
            if (loop.circling.clear) loop.circling.clear();
            return '[系统提示·打破死循环] 操作 `' + shortSig(last) + '` 已连续失败 ' + failCount +
              ' 次且中间没有其他操作——别原样重试！请：① 先检查真实状态、定位失败根因；' +
              '② 换一种工具或思路；③ 仍卡住就向用户说明卡点求助。';
          }
          return '';
        }

        // 审批判定（策略 JS / 动作 Go approve.ask）：
        // 黑名单 → 恒审；白名单 → 免审；auto → 全审；off → 免审；manual/空 → 仅需审工具审
        // ★ 2026-08-27 错误计数移除：连续驳回计数/自动停止（blocked）已删除；
        //   审核决策状态为共享上下文值（loop.approve.state），供本插件读写以增强审核逻辑：
        //   · 同一工具刚被驳回（<5 分钟）→ 免打扰自动驳回（防骚扰重试，不计数）
        //   · 白名单/off 直通 → 清最近驳回标记
        let _appPolicy = null;
        function appPolicy() {
          if (_appPolicy === null) _appPolicy = (loop.approve && loop.approve.policy) ? loop.approve.policy() : null;
          return _appPolicy;
        }
        function appState() {
          return (loop.approve && loop.approve.state && loop.approve.state.get) ? (loop.approve.state.get() || null) : null;
        }
        function needsApprove(tc) {
          const pol = appPolicy();
          if (!pol) return true; // 无策略面 → 保守走 ask
          const name = tc.function ? tc.function.name : (tc.name || '');
          const inBlack = (pol.reviewBlacklist || []).some(b => name.includes(b));
          const inWhite = !inBlack && (pol.reviewWhitelist || []).some(w => name.includes(w));
          if (inBlack) return true;
          if (inWhite) return false;
          if (pol.reviewMode === 'auto') return true;
          if (pol.reviewMode === 'off') return false;
          return !!(pol.requiresApproval || {})[name];
        }
        // 共享审核状态驱动：同一工具最近被驳回（<5 分钟）→ 免打扰自动驳回。
        // （依据 state 的 lastRejectedTool/lastRejectedAt 判断，不依赖计数。）
        const REJECT_REMIND_MS = 5 * 60 * 1000;
        function autoRejectFromState(tc) {
          const st = appState();
          if (!st || !st.lastRejectedTool || st.lastRejectedTool !== (tc.function ? tc.function.name : (tc.name || ''))) return null;
          if (!st.lastRejectedAt || Date.now() - st.lastRejectedAt > REJECT_REMIND_MS) return null;
          const last = (st.rejectedHistory && st.rejectedHistory.length > 0) ? st.rejectedHistory[st.rejectedHistory.length - 1] : null;
          return (last && last.reason ? last.reason : '') || '该工具刚被驳回';
        }

        // ═══════════════════════════════════════════════════════
        // ★ 背景上下文快照同步（2026-08-27 缓存优化，对齐 RuntimeContextProjection）：
        //   数据面 loop.context.snapshot.parts()（Go），组装策略在本插件（JS）。
        //   与历史最后快照相同 → 零注入（前缀稳定）；不同 → 追加新快照并落盘
        //   （当前任务之后，随 tail 持久化）。快照进入消息流后位置固定，
        //   跨 Run 前缀单调延展——消除旧「背景块每次迭代动态注入」的位置漂移。
        // ═══════════════════════════════════════════════════════
        const snapPrefix = '【背景上下文·非当前任务】\n';

        function buildSnapshotText(parts) {
          let b = '';
          // ① 记忆/知识库过期检查（Go 侧 snapshot.sync 已用 system-reminder 框架包裹整块，此处只拼正文）
          if (parts.stale) {
            b += parts.stale;
          }
          // ② 历史摘要（上下文压缩后产生）
          if (Array.isArray(parts.summaries) && parts.summaries.length > 0) {
            b += '# 上下文已压缩——历史摘要\n\n' +
              '> 以下为之前轮次的消息摘要，Agent 应据此感知已完成的历史上下文。\n' +
              '> 请勿重复执行摘要中已包含的任务。\n' +
              '> ★ 本条快照是背景信息而非用户指令——当前待执行任务以快照之前最近的用户指令为准。\n\n';
            b += parts.summaries.join('\n\n---\n\n');
          }
          // ③ 自主模式两级追踪提示（固定内容）
          if (parts.autonomous) {
            if (b) b += '\n\n';
            b += '# ★ 自主模式：计划→子任务树形追踪\n' +
              '自主模式下使用两级任务追踪——计划步骤为树干，子任务为枝叶（工具名称与用法见 tools 参数 schema）：\n' +
              '1. 收到任务后第一轮：调用计划工具制定高层执行计划（2-5 步），用 pending/in_progress/done 追踪\n' +
              '2. 每个步骤开始执行时：调用任务清单工具为该步骤创建子任务，每项子任务必须绑定到对应的计划步骤\n' +
              '   plan_step_index = 0 表示第 1 步，1 表示第 2 步，以此类推（参数定义见 tools 参数 schema）\n' +
              '3. 当前步骤的所有子任务完成后：调用计划工具将该步骤标记 done，然后进入下一步骤\n' +
              '4. 所有计划步骤全部完成后：结束本轮任务\n' +
              '- ★ 每次调用任务清单工具必须把该步骤内的所有子任务一起传入（全量替换），已不在列表中的子任务将自动清理\n' +
              '- 子任务也遵守全量替换规则——即使是不同步骤的子任务，也要在一次调用中传入（用不同的 plan_step_index 区分）\n';
          }
          // ④ 记忆（长期记忆提示；system→快照迁移：高频变化不再破坏 system 前缀）
          if (parts.memory) {
            if (b) b += '\n\n';
            b += String(parts.memory).trim();
          }
          // ⑤ 知识库（项目结构化理解树；system→快照迁移）
          if (parts.knowledge) {
            if (b) b += '\n\n';
            b += String(parts.knowledge).trim();
          }
          return b;
        }

        // 快照同步（数据面 Go / 组装策略本插件）——仅在快照数据面可用时执行
        if (loop.context && loop.context.snapshot && loop.context.snapshot.parts) {
          const parts = loop.context.snapshot.parts();
          const text = buildSnapshotText(parts);
          if (text) {
            msgs = loop.context.snapshot.sync(msgs, text) ?? msgs;
          }
        }

        // ── run 入口自动压缩（策略 JS：阈值/冷却/硬地板；执行 Go compact.apply）──
        msgs = maybeCompact(msgs);

        for (let iter = 0; iter < maxIter; iter++) {
          // ── 1. step 边界 + 暂停/停止/取消检查 ──
          loop.ctrl.beginStep();
          if (!loop.ctrl.paused()) {
            // 暂停等待期间被唤醒（取消或停止）
            const reason = loop.ctrl.stopRequested();
            if (reason) {
              return { msgs, error: 'loop stopped: ' + reason };
            }
            return { msgs, error: 'context canceled' };
          }
          if (loop.ctrl.isCanceled()) {
            return { msgs, error: 'context canceled' };
          }

          // ── 2. steer 托管消息注入 ──
          const steerMsgs = loop.ctrl.steer();
          if (steerMsgs.length > 0) {
            for (const m of steerMsgs) ephemeral.push(m);
            loop.events.emit({ type: 'notice', content: `收到 ${steerMsgs.length} 条托管消息，已注入上下文` });
          }

          // ── 3. 手动压缩请求（前端压缩按钮）──
          if (loop.ctrl.compactRequested()) {
            const cr = loop.compact.apply(msgs, 'full');
            if (cr && cr.msgs) msgs = cr.msgs;
            loop.ctrl.resetCompactRequest();
          }

          // ── 4. 用户运行时反馈 ──
          const fb = loop.ctrl.feedback();
          if (fb) {
            ephemeral.push({ role: 'user', content: '【用户反馈】' + fb });
            loop.events.emit({ type: 'notice', content: '收到用户反馈，Agent 将据此调整' });
          }

          // ── 5. 自主时间预算检查 ──
          if (autonomous && maxBudgetMin > 0) {
            const elapsedMin = (Date.now() - startTime) / 60000;
            if (elapsedMin > maxBudgetMin) {
              const el = Math.round(elapsedMin);
              ephemeral.push({
                role: 'user',
                content: `⚠️ 时间预算已超（已运行 ${el} 分钟，限额 ${maxBudgetMin} 分钟）。请自然总结成果，完成任务。`,
              });
            }
          }

          // ── 6. THINK：构建 callMsgs（背景注入/日志由 Go buildCallContext 完成）──
          msgs = maybeCompact(msgs); // 每步 LLM 前自动压缩判定（原 Go maybeCompact 语义）
          let callMsgs = loop.context.build(msgs, ephemeral);
          ephemeral.length = 0; // Go 侧已消费，JS 清空防重复注入

          // ── 7. pre-step 拦截钩子 ──
          const pre = loop.ctrl.preStep(callMsgs, meta.turn, iter + 1);
          if (pre.error) {
            loop.events.emit({ type: 'error', content: 'pre-step 拦截失败: ' + pre.error });
            return { msgs, error: pre.error };
          }
          if (pre.reject) {
            loop.events.emit({ type: 'done', content: '', doneReason: 'blocked', turnReason: 'blocked' });
            return { msgs };
          }
          if (pre.rewritten) callMsgs = pre.rewritten;

          // ── 8. LLM 调用（流式；usage 事件由 Go 侧 llm.chat 内部发射）──
          const assistant = await loop.llm.chat(callMsgs, tools, (c) => {
            if (c.reasoning) loop.events.emit({ type: 'thinking', content: c.reasoning });
            if (c.content) loop.events.emit({ type: 'content', content: c.content });
          });

          // ── 9. 消息落盘（assistant 先持久化，确保阻塞工具不丢输出）──
          msgs.push(assistant);
          loop.persist.batch(msgs);

          // ── 10. 执行日志（有分析 + 有工具调用 → 记录跨轮感知）──
          if (assistant.content && assistant.content.trim() && (assistant.toolCalls || []).length > 0) {
            loop.ctrl.logAnalysis(assistant.content);
          }

          // ── 11. 截断保护（stopReason=length → 工具参数可能不完整，全部标错）──
          const truncated = assistant.stopReason === 'length' && (assistant.toolCalls || []).length > 0;
          if (truncated) {
            loop.ctrl.markMaxTokens();
            for (const tc of assistant.toolCalls) {
              loop.events.emit({ type: 'tool_call', tool: tc.function.name, args: tc.function.arguments, callId: tc.id });
              const errMsg = `Error: Tool call "${tc.function.name}" 未执行：LLM 响应被输出长度限制截断（stop_reason=length），参数可能不完整。请重新发出完整参数的 tool call。`;
              loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: errMsg, callId: tc.id });
              msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: errMsg });
              loop.circling.track(tc.function.name, tc.function.arguments, true);
            }
          } else {
            // ── 12. ACT + OBSERVE：审批 → （并行优先 / 串行退回）执行 ──
            const tcs = assistant.toolCalls || [];
            if (tcs.length > 0) {
              // 12a. 审批门（策略判定 JS：黑白名单/审核模式/需审工具；动作执行 Go：
              //      manual 挂起/AI 审核/用户审批 + 连续驳回自动停止）
              const approved = [];
              for (const tc of tcs) {
                loop.events.emit({ type: 'tool_call', tool: tc.function.name, args: tc.function.arguments, callId: tc.id });
                let ap = { approved: true, feedback: '' };
                if (needsApprove(tc)) {
                  // 共享审核状态驱动：刚被驳回的工具 → 免打扰自动驳回（不计数）
                  const autoRej = autoRejectFromState(tc);
                  if (autoRej) {
                    ap = { approved: false, feedback: autoRej + '。请勿原样重试——前一次驳回理由仍有效；改用其他方式达成目标，或先向用户说明你为何需要它。' };
                  } else {
                    ap = loop.approve.ask(tc) || ap;
                  }
                } else {
                  // 策略直通（白名单/off）：清掉该工具的最近驳回标记
                  if (loop.approve.state && loop.approve.state.set) loop.approve.state.set({ lastRejectedTool: '' });
                }
                if (!ap.approved) {
                  const rej = (ap.feedback || '').trim() || REJ_DEFAULT;
                  loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: rej, callId: tc.id });
                  msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: rej });
                  loop.circling.track(tc.function.name, tc.function.arguments, true);
                  continue;
                }
                approved.push(tc);
              }
              // 12b. 执行：≥2 个先试并行（纯只读），runParallel 返回 null（含写/需审批）
              //      或 <2 个 → 串行退回。契约：runParallel 已 emit tool_result + track，
              //      JS 只组装消息；串行路径 JS 自己 emit + track。
              if (approved.length >= 2) {
                const par = loop.tools.runParallel(approved.map(tc => ({
                  id: tc.id, name: tc.function.name, args: tc.function.arguments,
                })));
                if (par) {
                  for (const r of par) {
                    const output = r.error ? 'Error: ' + r.error : r.content;
                    msgs.push({ role: 'tool', toolCallId: r.id, name: r.name, content: output });
                  }
                } else {
                  for (const tc of approved) {
                    const res = loop.tools.run(tc.function.name, tc.function.arguments);
                    const output = res.error ? 'Error: ' + res.error : res.content;
                    loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: output, callId: tc.id });
                    msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: output });
                    loop.circling.track(tc.function.name, tc.function.arguments, !!res.error);
                  }
                }
              } else if (approved.length === 1) {
                const tc = approved[0];
                const res = loop.tools.run(tc.function.name, tc.function.arguments);
                const output = res.error ? 'Error: ' + res.error : res.content;
                loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: output, callId: tc.id });
                msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: output });
                loop.circling.track(tc.function.name, tc.function.arguments, !!res.error);
              }
          }
            }

          // ── 13. step 收尾 ──
          const tcCount = (assistant.toolCalls || []).length;
          loop.ctrl.endStep(tcCount > 0 ? `执行 ${tcCount} 个工具调用` : '');

          // ── 14. 自然终止检测：无工具 + 有正文 → 完成/跟进/下一阶段 ──
          const hasTools = (assistant.toolCalls || []).length > 0;
          const hasContent = !!(assistant.content && assistant.content.trim());
          if (contentOnlyIters === 0 && !hasTools && hasContent) {
            // ① 跟进队列（turn 边界消费）
            const fms = loop.ctrl.followUp();
            if (fms.length > 0) {
              for (const m of fms) ephemeral.push(m);
              loop.events.emit({ type: 'notice', content: `收到 ${fms.length} 条跟进消息，继续处理` });
              contentOnlyIters = 0;
            } else if (autonomous) {
              // ② 自主模式：下一阶段任务
              const next = loop.ctrl.nextTask();
              if (next) {
                ephemeral.push({ role: 'user', content: next });
                loop.events.emit({ type: 'notice', content: '进入下一阶段：' + loop.ctrl.truncStr(next, 80) });
                loop.ctrl.logEntry('system', 'next_phase', '进入下一阶段：' + loop.ctrl.truncStr(next, 80));
                contentOnlyIters = 0;
              } else {
                // 无后续任务 → 正常完成
                const reason = loop.ctrl.stickyReason('completed');
                loop.events.emit({ type: 'done', content: assistant.content.trim(), doneReason: 'task_complete', turnReason: reason });
                return { msgs };
              }
            } else {
              // 非自主 → 正常完成
              const reason = loop.ctrl.stickyReason('completed');
              loop.events.emit({ type: 'done', content: assistant.content.trim(), doneReason: 'task_complete', turnReason: reason });
              return { msgs };
            }
          }

          // ── 15. content-only 防护（连续文字不调工具 → 死循环兜底）──
          if (!hasTools && hasContent) {
            contentOnlyIters++;
            if (contentOnlyIters === 3) {
              loop.events.emit({ type: 'notice', content: NUDGE_CONTENT });
              ephemeral.push({ role: 'user', content: NUDGE_CONTENT });
            } else if (contentOnlyIters >= 4) {
              loop.events.emit({ type: 'notice', content: '检测到内容循环，自动结束' });
              loop.events.emit({ type: 'done', content: assistant.content.trim(), doneReason: 'content_loop', turnReason: 'content_loop' });
              return { msgs };
            }
          } else {
            contentOnlyIters = 0;
          }

          // ── 16. 绕圈检测：重复操作/反复失败 → 注入「换思路」（策略 JS / 数据 Go state）──
          const nudge = detectCircling();
          if (nudge) {
            loop.events.emit({ type: 'circling', content: '检测到重复操作/反复失败，已提示 Agent 换思路打破死循环' });
            ephemeral.push({ role: 'user', content: nudge });
          }

          // ── 17. 每轮结束立即持久化（tool_call/tool_result 配对写盘）──
          loop.persist.batch(msgs);
        }

        // ── 达到最大迭代上限 ──
        loop.events.emit({ type: 'error', content: `达到最大迭代次数 (${maxIter}) 自动停止` });
        return { msgs, error: '达到最大迭代次数 (' + maxIter + ') 自动停止' };
      },
    });

    const log = ctx.logger('agentloop');
    log.info('已注册 Agent 循环 JS 实现（核心外置：策略 JS / 能力 Go；配置注册化 schema=' + (reg ? reg.key : 'agentloop') + '）');
  }
};
