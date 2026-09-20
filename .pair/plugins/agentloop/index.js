// ═══════════════════════════════════════════════════════════════
// agentloop — Agent 循环 JS 实现（agentloop 核心外置）
//
// ★ 2026-08-19 架构升级：循环策略从 Go（internal/agent/loop.go）外置到 JS。
//   Go 保留能力（Provider.Chat / Registry.Execute / emit / persist /
//   approve / buildCallContext / 绕圈检测），经能力代理对象注入：
//
//     loop.llm.chat(msgs, tools, onChunk) → assistant
//     loop.tools.list() / loop.tools.run(name, argsJson)
//     loop.events.emit(event)
//     loop.persist.batch(msgs)
//     loop.approve.ask(tc) → {approved, feedback}  // 审核门（动作执行；驳回记录进共享状态）
//     loop.approve.state.get()/set(obj)             // 共享审核状态（最近驳回/历史，不计数）
//     loop.context.build(msgs, ephemeral) → callMsgs
//     loop.compact(msgs) → msgs
//     loop.circling.track(name, args, failed) / loop.circling.detect()
//     loop.store.get(key) / loop.store.set(key, value)
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

// ═══════════════════════════════════════════════════════════════
// 会话交接（handoff）策略实现 — 2026-09-12 从 Go（internal/agent/handoff.go）外置
//
// 宿主在两个**跨轮边界**调用本实现（执行位置仍在宿主：用户输入边界在会话装配
// 之前、段边界在 Run 之外，插件无从自主介入）：
//   · onUserTurn  用户输入（新提交对话 / 点继续执行）—— 带判官（任务关系可能变化）
//   · onSegment   段续跑边界（同一条消息内的分段，Run 之间）—— judge=null
//
// ★ 铁律：整理只替换「喂 LLM 的历史视图」，落盘/展示始终为完整时间线；
//   一次 Run 的 step 之间不整理（轮内前缀 append-only —— 缓存命中率的前提）。
//
// ★ 口径单一真源：锚点指纹 / token 估算 / 阈值 / 规则摘要 / 相关性解析经
//   ctx.handoff 桥接自 Go —— 两侧判定同源，避免锚点或阈值漂移。
// ═══════════════════════════════════════════════════════════════

// hTruncRunes 与 Go truncRunesAgent 等价（TrimSpace → rune 截断 → 超长加省略号）。
function hTruncRunes(s, n) {
  const t = String(s == null ? '' : s).trim();
  const r = Array.from(t);
  return r.length <= n ? t : r.slice(0, n).join('') + '…';
}

function hRuneLen(s) { return Array.from(String(s == null ? '' : s)).length; }

// hAnchorAt 计算交接基点指纹（序列锚）——等价 Go handoffAnchorAt。
function hAnchorAt(U, history, keep, T) {
  const hist = U.stripSystem(history) || [];
  if (!hist.length) return '';
  if (!(keep >= 1)) keep = 1;
  let idx = hist.length - keep;
  if (idx < 0) idx = 0;
  // 边界对齐：基点本身不能是孤立 tool（其配对 tool_call 会被折叠）
  while (idx > 0 && hist[idx] && hist[idx].role === 'tool') idx--;
  let n = T.anchorSeq;
  if (idx + n > hist.length) n = hist.length - idx;
  const parts = [];
  for (let k = 0; k < n; k++) parts.push(U.fingerprint(hist[idx + k]));
  return parts.join(T.anchorSep);
}

// hAnchorIndex 定位锚点消息下标（等价 Go handoffAnchorIndex）：
// ① 记录内 MsgCount 推回生成时基点（精确、穿透重复消息）② 从前往后首个匹配
// （历史前部不变 → 跨轮稳定）③ 返回 -1（调用方按锚点丢失保守处理）。
function hAnchorIndex(U, hist, rec, T) {
  if (!rec || !rec.anchor) return -1;
  const parts = String(rec.anchor).split(T.anchorSep);
  const n = parts.length;
  if (!n) return -1;
  const matchAt = (i) => {
    if (i < 0 || i + n > hist.length) return false;
    for (let k = 0; k < n; k++) {
      if (U.fingerprint(hist[i + k]) !== parts[k]) return false;
    }
    return true;
  };
  if (rec.msgCount > 0) {
    let idx = rec.msgCount - U.keepForRelevance(rec.relevance);
    if (idx > hist.length) idx = hist.length;
    while (idx > 0 && idx < hist.length && hist[idx] && hist[idx].role === 'tool') idx--;
    if (matchAt(idx)) return idx;
  }
  for (let i = 0; i + n <= hist.length; i++) {
    if (matchAt(i)) return i;
  }
  return -1;
}

// hIncrement 自基点起的增量消息（含基点本身）——等价 Go handoffIncrement。
function hIncrement(U, history, rec, T) {
  const hist = U.stripSystem(history) || [];
  if (!rec || !rec.anchor) return hist;
  const i = hAnchorIndex(U, hist, rec, T);
  if (i >= 0) return hist.slice(i);
  let keep = T.keepRecent;
  if (rec.relevance) keep = U.keepForRelevance(rec.relevance);
  return hist.length > keep ? hist.slice(hist.length - keep) : hist;
}

// hShouldHandoff 触发门槛（等价 Go ShouldHandoff）：token 阈值 或 条数阈值。
function hShouldHandoff(U, history, maxCtx, T) {
  if (!U.enabled) return { ok: false, reason: '已关闭（PAIR_HANDOFF=0）' };
  const hist = U.stripSystem(history) || [];
  if (!hist.length) return { ok: false, reason: '' };
  const tokens = U.estimateTokens(hist);
  const trigger = U.thresholds(maxCtx).triggerTokens;
  if (tokens >= trigger) {
    return { ok: true, reason: `历史 ${hist.length} 条 / ~${tokens} tokens ≥ 触发阈值 ${trigger}` };
  }
  if (hist.length >= T.minMsgs) {
    return { ok: true, reason: `历史 ${hist.length} 条 ≥ 触发条数 ${T.minMsgs}（~${tokens} tokens）` };
  }
  return { ok: false, reason: '' };
}

// hShouldRefresh 刷新判断（等价 Go ShouldRefreshHandoff）：
// 增量 = 锚后内容 tokens − 生成时保留段基线 KeptTokens（只看新增量，防抖动）。
function hShouldRefresh(U, history, rec, T) {
  if (!rec || !String(rec.text || '').trim()) {
    return { need: true, inc: U.estimateTokens(U.stripSystem(history) || []) };
  }
  const hist = U.stripSystem(history) || [];
  if (hAnchorIndex(U, hist, rec, T) < 0) return { need: true, inc: U.estimateTokens(hist) };
  const incTokens = U.estimateTokens(hIncrement(U, history, rec, T));
  let delta = incTokens - (rec.keptTokens || 0);
  if (delta < 0) delta = 0;
  return { need: delta >= T.refreshTokens, inc: delta };
}

// hComposeView 组装注入视图：[交接消息] + [基点起增量（含基点，原文保留）]
// —— 等价 Go ComposeHandoffView（视图内至多一份交接块）。
function hComposeView(U, history, rec, text, T) {
  const inc = hIncrement(U, history, rec, T);
  const out = [{ role: 'user', content: text }];
  for (const m of inc) {
    if (U.isHandoffText(m.content || '')) continue; // 跳过历史中的旧交接块
    out.push(m);
  }
  return out;
}

// hSanitizeBody 清洗 LLM 输出（等价 Go sanitizeHandoffBody）。
function hSanitizeBody(s, U) {
  s = String(s == null ? '' : s).trim();
  if (s.startsWith('```')) {
    const i = s.indexOf('\n');
    if (i >= 0) s = s.slice(i + 1);
    s = s.trim();
    if (s.endsWith('```')) s = s.slice(0, s.length - 3);
    s = s.trim();
  }
  const i = s.indexOf('\n');
  if (i >= 0 && s.slice(0, i).indexOf(U.title) >= 0) s = s.slice(i + 1).trim();
  return s;
}

// hSample 构建 LLM 整理的输入文本（等价 Go handoffSample：尾部取样 + 规则摘要 + 上次交接）。
function hSample(U, history, prev, task, T) {
  const hist = U.stripSystem(history) || [];
  let start = 0;
  if (hist.length > T.inputMaxMsgs) start = hist.length - T.inputMaxMsgs;
  let b = '';
  if (start > 0) {
    const early = hist.slice(0, start);
    b += '（更早 ' + early.length + ' 条历史｜规则摘要）\n' + U.ruleSummary(early) + '\n\n';
  }
  for (const m of hist.slice(start)) {
    if (U.isHandoffText(m.content || '')) continue;
    let role = '工具';
    if (m.role === 'user') role = '用户';
    else if (m.role === 'assistant') role = '助手';
    const toolInfo = (m.toolCalls && m.toolCalls.length) ? ' [工具调用]' : '';
    let content = String(m.content == null ? '' : m.content).trim();
    if (!content) content = '（无正文）';
    b += role + ':' + toolInfo + ' ' + hTruncRunes(content, T.inputMsgRunes) + '\n';
  }
  if (!b) b = '（历史无可取内容）\n';

  let out = '';
  out += '你是对话交接助手。请阅读下方【即将继续的任务】与【对话历史节选】，输出一份《会话交接·提交消息》，供继续工作的 AI 助手快速恢复状态。\n\n';
  out += '要求：\n';
  out += '1. 先判断历史与「即将继续的任务」的相关性：高（同一任务的延续）/ 部分（同项目不同任务）/ 无关（全新任务）。无关或部分相关时，历史中与任务无关的细节只保留一句概述，不展开。\n';
  out += '2. 输出 Markdown 正文，包含以下小节（无内容的小节省略）：\n';
  out += '## 与当前任务的相关性\n（一行：高/部分/无关 + 一句话说明）\n';
  out += '## 任务目标\n## 已完成\n## 当前状态（关键文件、构建/测试结果、产物落点）\n## 关键决策与坑\n## 待完成 / 下一步\n';
  out += '3. 具体优于笼统：保留文件名、命令、结论、错误原因等可执行信息；不要编造历史中没有的内容。\n';
  out += '4. 全文不超过 700 字，只输出正文（不要额外解释、不要代码围栏）。\n\n';
  out += '【即将继续的任务】\n' + hTruncRunes(task, T.taskRunes) + '\n';
  if (prev && String(prev.text || '').trim()) {
    out += '\n【上一份交接（可作基线，无需重复其全文）】\n' + hTruncRunes(prev.text, T.prevTextRunes) + '\n';
  }
  out += '\n【对话历史节选（尾部 ' + Math.min(hist.length, T.inputMaxMsgs) + ' 条）】\n' + b;
  return out;
}

// hBuildText 一次 LLM 整理（失败 → 规则式兜底），返回带 marker+标题前缀的注入文本。
function hBuildText(U, args, prev, history, task, T, log) {
  let body = '';
  const prov = args.provider;
  if (prov && typeof prov.chat === 'function') {
    const msg = prov.chat([{ role: 'user', content: hSample(U, history, prev, task, T) }]);
    if (msg && msg.content) {
      const s = hSanitizeBody(msg.content, U);
      if (hRuneLen(s) >= 40) body = s;
      else log('会话交接：LLM 输出过短（清洗后 ' + hRuneLen(s) + ' runes），回退规则式');
    } else {
      log('会话交接：LLM 整理失败（无输出），回退规则式');
    }
  } else {
    log('会话交接：无可用 Provider，使用规则式交接');
  }
  if (!body) body = U.ruleFallback(history);
  // ★ 2026-09-13：不再拼接「背景上下文」前缀（该实现已整体移除）——
  //   交接文本以标题行开头（与 Go 侧 BuildHandoffText 逐字节一致）。
  return U.title + '\n' + body;
}

// hJudge 轻量相关性判官（独立实例，只输出一个词；失败 → ''，保持现状零副作用）。
function hJudge(U, args, rec, task, T) {
  const judge = args.judge;
  if (!judge || typeof judge.chat !== 'function') return '';
  if (!rec || !String(rec.text || '').trim() || !String(task || '').trim()) return '';
  const prompt = '判断「当前任务」与「历史交接要点」的关系，只输出一个词：高、部分 或 无关。\n' +
    '高=同一任务的延续；部分=同项目不同任务；无关=全新任务。\n\n' +
    '【当前任务】\n' + hTruncRunes(task, T.judgeTaskRunes) + '\n\n' +
    '【历史交接要点】\n' + hTruncRunes(rec.text, T.judgePrevRunes) + '\n\n只输出一个词：';
  const msg = judge.chat([{ role: 'user', content: prompt }]);
  if (!msg || !msg.content) return '';
  return U.parseRelevance(msg.content);
}

// hBuildView 主流程（等价 Go BuildHandoffView）：判断 →（生成 / 复用）→ 组装视图。
// 返回 view 数组（启用整理）或 null（未达阈值 / 未启用 / 存储不支持）。
function hBuildView(U, args, log) {
  const T = U.thresholds(args.maxContextTokens);
  const convID = args.convID || '';
  const history = args.history || [];
  const store = args.store;
  if (!U.enabled || !convID || !store || !history.length) return null;
  if (!hShouldHandoff(U, history, args.maxContextTokens, T).ok) return null;
  if (typeof store.loadRecord !== 'function') return null; // 存储不支持交接记录 → 不启用

  const prev = store.loadRecord();
  let rec = null;
  let forceRefresh = false;
  let rel = '';
  if (prev && String(prev.text || '').trim()) {
    const curRel = prev.relevance || T.relHigh;
    const sr = hShouldRefresh(U, history, prev, T);
    if (!sr.need) {
      // B：复用期定期语义复检（增量达周期才调用判官）
      if (args.judge && (sr.inc - (prev.relCheckedInc || 0)) >= T.recheckTokens) {
        const newRel = hJudge(U, args, prev, args.task, T);
        if (newRel && newRel !== curRel) {
          forceRefresh = true;
          rel = newRel;
          log('会话交接：判官复检 ' + curRel + ' → ' + newRel + '（自复检增量 ~' + sr.inc + ' tokens），刷新交接');
        } else {
          if (newRel) log('会话交接：判官复检保持 ' + newRel + '（自复检增量 ~' + sr.inc + ' tokens）');
          prev.relCheckedInc = sr.inc; // 无论成败都推进（防失败时每轮重试风暴）
          store.saveRecord(prev);
        }
      }
      if (!forceRefresh) {
        rec = prev;
        const histArr = U.stripSystem(history) || [];
        const histN = histArr.length;
        const idx = hAnchorIndex(U, histArr, rec, T);
        log('会话交接：复用上次交接（增量 ~' + sr.inc + ' tokens < 刷新阈值 ' + T.refreshTokens +
          '；锚点 idx=' + idx + '/' + histN + ' 条，msgCount=' + (rec.msgCount || 0) +
          ' keep=' + U.keepForRelevance(rec.relevance) + '）');
        if (idx < 0) {
          // ★ 锚点未命中取证：打印锚点指纹与历史前几条指纹（定位失败 → 兜底保留段 → 每段断裂）
          const fp = [];
          for (let k = 0; k < Math.min(4, histArr.length); k++) {
            fp.push(k + ':' + String(U.fingerprint(histArr[k])).slice(0, 50));
          }
          const ap = String(rec.anchor || '').split(T.anchorSep)
            .map((s, i) => i + ':' + String(s).slice(0, 46)).join(' | ');
          log('会话交接：★锚点未命中（历史 ' + histN + ' 条，msgCount=' + (rec.msgCount || 0) +
            '）锚点指纹=' + ap);
          log('会话交接：★历史前 4 条指纹=' + fp.join(' | '));
        }
      }
    }
  }
  if (!rec) {
    const text = hBuildText(U, args, prev, history, args.task, T, log);
    if (!rel) rel = U.parseRelevance(text);
    if (!rel) rel = T.relHigh; // 解析失败 → 保守按高（不缩保留段）
    rec = {
      text: text,
      createdAt: new Date().toISOString(),
      anchor: hAnchorAt(U, history, U.keepForRelevance(rel), T),
      msgCount: (U.stripSystem(history) || []).length,
      relevance: rel,
      keptTokens: 0,
    };
    rec.keptTokens = U.estimateTokens(hIncrement(U, history, rec, T));
    store.saveRecord(rec);
    const gIdx = hAnchorIndex(U, U.stripSystem(history) || [], rec, T);
    log('会话交接：已生成新交接（' + rec.msgCount + ' 条历史，相关性 ' + rel + '；锚点 idx=' + gIdx + '）');
  }
  return hComposeView(U, history, rec, rec.text, T);
}

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
        // ★ 2026-09-12：「最大迭代数」配置项已移除——段内迭代安全上限由宿主按工具
        //   预算派生（tool_budget.go IterationLimit = 生效预算 + 5），段的结束由
        //   段预算 + 自动续跑（maxToolBudgetSegments）负责，不再提供迭代数配置。
        // ★ 2026-09-12 分段续跑配置化 + 双闸门（宿主 internal/agent/tool_budget.go）：
        //   配置项全部在本插件注册（存 settings.json 的 pluginSettings.agentloop），
        //   装配时经 ctx.loopFactory.register 的 overrides 透传给宿主 Loop。
        //   段结束 = 双闸门任一达上限：stepBudget（步数，一步 = 一次模型调用）
        //   或 toolCallBudget（段内实际执行的工具调用次数）；两者同时达上限时
        //   按步数闸报告（SegmentState.Reason = step_budget）。
        { name: 'stepBudget', label: '单段步数预算', type: 'number', default: 120,
          hint: '每个执行段内最多走多少步（一步 = 一次模型调用；模型一轮并列多个工具调用仍只算一步），达上限即结束本段并自动续跑下一段（同会话、历史保留）。0=默认 120；负数=不限' },
        { name: 'toolCallBudget', label: '单段工具调用预算', type: 'number', default: 120,
          hint: '每个执行段内最多执行多少次工具调用，达上限即结束本段并自动续跑下一段（同会话、历史保留）。0=默认 120；负数=不限' },
        { name: 'maxToolBudgetSegments', label: '最大自动续跑段数', type: 'number', default: 20,
          hint: '单轮任务最多自动续跑多少段（防失控），超过后停止续跑，再发一条消息即可继续。0=默认 20' },
        { name: 'autonomous', label: '自主模式', type: 'checkbox', default: false, hint: '勾选=强制开启自主（不勾=跟随全局开关，不再强制关闭）' },
        { name: 'maxAutonomousMinutes', label: '自主时间预算（分钟）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'checkpointInterval', label: '检查点间隔（迭代数）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'reviewMode', label: '审核模式', type: 'select', default: '', options: ['', 'auto', 'manual', 'off'], hint: '空=不覆盖（auto=AI审核/manual=手动审批/off=放行）' },
        { name: 'reviewBlacklist', label: '审核黑名单', type: 'text', default: '', hint: '逗号分隔工具名（命中需审核）' },
        { name: 'reviewWhitelist', label: '审核白名单', type: 'text', default: '', hint: '逗号分隔工具名（命中跳过审核，黑名单优先）' },
        { name: 'autoIterateOnRejection', label: '拒绝后自动迭代', type: 'checkbox', binding: 'autoIterateOnRejection' },
        { name: 'ignoreDirs', label: '忽略目录', type: 'tags', binding: 'ignoreDirs',
          hint: '逗号分隔（node_modules, dist, .git…）' },
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
            hint: '维护服务商：名称、API URL、LLM 协议、可用模型列表，及每模型独立参数（温度/思考/输出上限/上下文窗口/多模态）。API Key 请在「AI 配置」中填写。AI tab 的下拉与联动均来自此处。',
            // ★ 2026-08-21 添加模型区 schema 驱动：modelEditor 声明组件配置（label/placeholder），
            //   前端 ProviderManager 按此渲染模型编辑器（与 modelParamFields 同层，新增配置无需改前端组件）。
            modelEditor: { label: '可用模型（回车或逗号分隔添加；支持整段粘贴）', placeholder: '输入模型名，回车添加…' },
            // ★ 2026-09-02 LLM 协议：协议选项/文案由插件注册配置（前端不硬编码）。
            //   空=默认（openai-completions：OpenAI 兼容 /chat/completions）；
            //   anthropic-messages=Anthropic 原生 /messages；openai-responses=OpenAI Responses /responses。
            //   选非默认协议时 API URL 填基础地址（不含协议路径，如 https://api.anthropic.com/v1）。
            protocolLabel: 'LLM 协议',
            protocolOptions: ['', 'openai-completions', 'openai-responses', 'anthropic-messages'],
            protocolHint: '请求协议：空=默认 OpenAI 兼容 /chat/completions；anthropic-messages=Anthropic 原生 /messages；openai-responses=OpenAI Responses /responses。非默认协议时 API URL 填基础地址（不含协议路径）。',
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
    // ★ 配置消费插件化（2026-08-19）+ 决策全量迁插件（2026-09-03）：
    //   LLM Provider 参数装配器——AI 连接参数的唯一决策者。
    //   Go 内核（buildWebProvider/Review/Plan/工具集分析）不再做任何配置决策：
    //   只传裸基线（settings 顶层存储字段）+ 装配上下文（preset/conv*）经
    //   agent.ResolveProviderParams() → 本装配器获取最终参数。
    //   本装配器决策链：① 配置整套展开（会话配置 > 全局激活，经 ctx.aiPresets）
    //   → ② 会话级覆盖（conv*）→ ③ 服务商数据兜底（经 ctx.models）→ ④ Key 选择
    //   → ⑤ 模型级参数 → ⑥ 上下文窗口层级 → ⑦ 全局温度/思考/输出兜底
    //   → ⑧ 服务商配置为准（★ 2026-09-19：models.json 的温度/最大输出/上下文窗口
    //        覆盖上述 settings 取值——生成参数的唯一来源是「服务商」）
    //   → ⑨ 统一模型同步（plan/review 跟随执行模型）。
    // ═══════════════════════════════════════════════════════════
    ctx.providerFactory.register((current) => {
      const s = (ctx.app && ctx.app.settings) || {};
      const over = {};
      // ── ① 配置整套展开（会话配置 > 全局激活；配置不存在/无效 → 跳过）──
      const presetName = current.convPreset || current.preset || '';
      const pres = presetName ? (ctx.aiPresets.get(presetName) || null) : null;
      const presValid = !!(pres && (pres.provider || pres.executeModel));
      if (presValid) {
        if (pres.provider) over.provider = pres.provider;
        if (pres.baseURL) over.baseURL = pres.baseURL;
        if (pres.apiKey) over.apiKey = pres.apiKey;
        if (pres.executeModel) over.model = pres.executeModel;
        if (pres.temperature !== undefined && pres.temperature !== null && pres.temperature !== '') {
          const t = parseFloat(pres.temperature);
          if (!isNaN(t) && t >= 0) over.temperature = t;
        }
        if (pres.thinkingMode) over.thinkingMode = pres.thinkingMode;
        if (pres.maxTokens && Number(pres.maxTokens) > 0) over.maxTokens = Number(pres.maxTokens);
        if (pres.contextMaxTokens && Number(pres.contextMaxTokens) > 0) over.contextMaxTokens = Number(pres.contextMaxTokens);
        // 协议：预设显式指定优先；否则按预设服务商的协议（服务商级）
        if (pres.protocol) {
          over.protocol = pres.protocol;
        } else if (pres.provider) {
          const me0 = ctx.models.get(pres.provider);
          if (me0 && me0.protocol) over.protocol = me0.protocol;
        }
      }
      // ── ② 会话级覆盖（会话选定 服务商/模型 > 一切展开结果；历史链路无配置名时直接生效）──
      if (current.convProvider) over.provider = current.convProvider;
      if (current.convModel) over.model = current.convModel;
      // ── ③ 最终 服务商/模型（后续决策的依据）──
      const provider = over.provider || current.provider || s.provider || '';
      const model = over.model || current.model || s.executeModel || s.model || '';
      // ── ④ 服务商数据兜底（经 ctx.models 查 models.json：BaseURL/协议/Key/上下文）──
      const me = (provider && ctx.models.get(provider)) || {};
      const presetProvider = presValid ? (pres.provider || '') : '';
      const providerChanged = !!(current.convProvider && provider !== presetProvider);
      if (!over.baseURL && me.baseURL) over.baseURL = me.baseURL;
      if (!over.protocol && me.protocol) over.protocol = me.protocol;
      // ★ Key 选择（2026-09-01 Key 回归 AI 配置；2026-09-03 决策迁插件）：
      //   ① 配置展开的 Key（① 中 over.apiKey 非空即优先——会话所选配置的 Key 必须生效）
      //   ② 无有效配置或会话切了服务商 → 该服务商任一配置的 Key（AI 配置，遍历兜底）
      //   ③ 服务商级 Key（models.json，旧数据迁移兜底）
      if (!over.apiKey && provider) {
        if (!presValid || providerChanged) {
          const ps = ctx.aiPresets.list() || {};
          for (const k of Object.keys(ps)) {
            const pp = ps[k];
            if (pp && pp.provider === provider && pp.apiKey) { over.apiKey = pp.apiKey; break }
          }
        }
        if (!over.apiKey && me.apiKey) over.apiKey = me.apiKey;
      }
      // ── ⑤ 模型级参数（settings.modelParams[服务商][模型]；GBK 损坏 key 按模型名兜底）──
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
      // ── ⑥ 上下文窗口层级：模型级 > 服务商级（最终服务商）> 配置级 > 全局 ──
      let cctx = (mp && mp.contextMaxTokens) ? Number(mp.contextMaxTokens) : 0;
      if (!(cctx > 0) && me.contextMaxTokens) cctx = Number(me.contextMaxTokens);
      if (!(cctx > 0) && pres && Number(pres.contextMaxTokens) > 0) cctx = Number(pres.contextMaxTokens);
      if (!(cctx > 0) && s.contextMaxTokens) cctx = Number(s.contextMaxTokens);
      if (cctx > 0) over.contextMaxTokens = cctx;
      // ── ⑦ 全局温度/思考/输出兜底（模型级与配置级未配置时；兼容旧配置）──
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
      // ── ⑧ 服务商配置为准（★ 2026-09-19 修复）：温度/最大输出/上下文窗口的唯一来源 = models.json ──
      //   层级：模型级（me.modelParams[模型]）> 服务商级（me.temperature/maxTokens/contextMaxTokens）。
      //   上方 ⑤⑥⑦ 的 settings 取值仅在两处都未配置时兜底——不再覆盖服务商值。
      const mpm = (me.modelParams && me.modelParams[model]) || null;
      // 取「已配置」的值：undefined/null/空串/数字 0 一律视为未配置（0=不设），
      // 逐级回退：models.json 模型级 → models.json 服务商级。
      const svcPick = (k) => {
        for (const v of (mpm ? [mpm[k], me[k]] : [me[k]])) {
          if (v === undefined || v === null || v === '') continue;
          if (typeof v === 'number' && v <= 0) continue;
          if (typeof v === 'string' && v.trim() === '') continue;
          return v;
        }
        return '';
      };
      const svcTemp = svcPick('temperature');
      if (svcTemp !== '') {
        const t = parseFloat(svcTemp);
        if (!isNaN(t) && t >= 0) over.temperature = t;
      }
      const svcMax = Number(svcPick('maxTokens'));
      if (svcMax > 0) over.maxTokens = svcMax;
      const svcCtx = Number(svcPick('contextMaxTokens'));
      if (svcCtx > 0) over.contextMaxTokens = svcCtx;
      if (mpm && mpm.multimodal === true) over.multimodal = true;
      // ── ⑨ 统一模型同步（决策面在插件：规划/审核 一律跟随执行模型，不拆分）──
      if (model) { over.planModel = model; over.reviewModel = model; }
      return over;
    });

    // ── 参数级装配（保留）：CreateLoop 时覆盖装配参数（提示词/迭代/审核模式）──
    ctx.loopFactory.register((opts) => {
      // ★ 2026-08-21 修复：动态读取配置（registerSettings 返回的 value 是 apply 时
      //   静态快照，设置面板保存后不刷新 → 装配器一直用旧值 → 提示词/预算等
      //   设置保存不生效）。改为每次 Create 时实时读 pluginSettings.agentloop。
      const cfg = ctx.getSettings('agentloop') || {};
      const over = {};
      if (typeof cfg.systemAppend === 'string' && cfg.systemAppend) {
        over.system = (opts.system || '') + '\n\n' + cfg.systemAppend;
      }
      // ★ 2026-09-12 分段续跑配置化（宿主 tool_budget.go，双闸门）：单段步数预算、
      //   单段工具调用预算与续跑段数上限经装配参数透传。0 = 不覆盖（宿主默认
      //   120 步 / 120 次 / 20 段），预算负数 = 不限（故仅排除 0，负值照传）。
      if (cfg.stepBudget != null && Number.isFinite(Number(cfg.stepBudget)) && Number(cfg.stepBudget) !== 0) {
        over.stepBudget = Number(cfg.stepBudget);
      }
      if (cfg.toolCallBudget != null && Number.isFinite(Number(cfg.toolCallBudget)) && Number(cfg.toolCallBudget) !== 0) {
        over.toolCallBudget = Number(cfg.toolCallBudget);
      }
      if (cfg.maxToolBudgetSegments != null && Number(cfg.maxToolBudgetSegments) > 0) {
        over.maxToolBudgetSegments = Number(cfg.maxToolBudgetSegments);
      }
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
      // ★ 2026-09-19 上下文窗口以服务商配置（models.json）为准：宿主已按装配结果
      //   （服务商级 > settings 兜底）传入 opts.maxContextTokens；仅当宿主未传（<=0）时
      //   才用插件设置 / 全局 settings 兜底——不再无条件覆盖服务商值。
      if (!(Number(opts.maxContextTokens) > 0)) {
        if (cfg.maxContextTokens != null && Number(cfg.maxContextTokens) > 0) {
          over.maxContextTokens = Number(cfg.maxContextTokens);
        } else {
          const aiTop = (ctx.app && ctx.app.settings) || {};
          const ctxMax = Number(aiTop.contextMaxTokens);
          if (ctxMax > 0) over.maxContextTokens = ctxMax;
        }
      }
      return over;
    });

    // ═══════════════════════════════════════════════════════════
    // ★ JS 循环实现（agentloop 核心外置）：Loop.Run 委托本 run() 驱动循环。
    //   Go 侧前置：msgs 组装（system+历史+任务）、staleMsg、工具精简、
    //   OnToolUpdate 桥、Run 开始时精简一次——全部完成后再调本函数。
    //   返回 { msgs, error? }：msgs = 完整消息列表（Go 更新 History 并收尾）。
    // ═══════════════════════════════════════════════════════════
    ctx.loopFactory.registerLoop({
      id: 'agentloop',
      async run({ task, msgs, tools, meta, loop }) {
        // ★ 2026-09-12：「最大迭代数」配置项已移除——段内迭代安全上限由宿主按段
        //   预算派生（internal/agent/tool_budget.go IterationLimit =
        //   max(生效步数预算, 生效工具调用预算) + 5，双闸门均不限时以预算钳制上限
        //   兜底），经 meta.iterationLimit 传入。
        //   此处仅在旧宿主/参数缺失时按同一公式兜底自算；段的结束由双闸门预算 +
        //   自动续跑负责（见下方 step 14.5），本上限只是防失控的最后一道保险。
        const ITER_SLACK = 5;       // 对齐宿主 LoopIterationSlack
        const BUDGET_LIMIT = 10000; // 对齐宿主 MaxToolCallBudgetLimit
        const tbBudget = Number(meta.toolCallBudget || 0);
        const sbBudget = Number(meta.stepBudget || 0);
        // 双闸门取较大者派生迭代上限（任一闸门关闸时上限必须仍有余量）
        const budgetMax = Math.max(tbBudget > 0 ? tbBudget : 0, sbBudget > 0 ? sbBudget : 0);
        let maxIter = Number(meta.iterationLimit) > 0
          ? Number(meta.iterationLimit)
          : (budgetMax > 0 ? budgetMax + ITER_SLACK : BUDGET_LIMIT + ITER_SLACK);
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
        // ★ 策略外置常量（2026-08-27）：精简/绕圈/审批的「判定策略」本插件自持
        //   （阈值/窗口/提示文本均可在此自定义；数据面/执行面在 Go）。
        // ═══════════════════════════════════════════════════════
        // ★ 2026-09 决策：**段内只追加，不中途精简**。
        //   段内删历史 = 改写已发送过的字节 → provider 前缀缓存必断在删改点，
        //   被保留内容要全额重新 prefill（实测单次 miss 13K~54K tokens）。
        //   体积控制改由两道更强的闸负责：
        //     ① 双闸门任一达上限即结束本段并自动续跑下一段（宿主 tool_budget.go）：
        //        stepBudget 步（默认 120 步，一步 = 一次模型调用）或
        //        toolCallBudget 次工具调用（默认 120 次）；
        //     ② 下一段（下一次 Run）开始时重载历史并做跨段精简
        //        （宿主 CondenseHistoryByPressure / maybeCompact）。
        //   因此本函数只保留「前端精简按钮」路径；阈值只用于诊断日志。
        const COMPACT = { thresholdEarly: 0.45, thresholdFull: 0.90 };
        // 绕圈检测策略：窗口 + 重复/失败阈值
        const CIRCLING = { window: 12, repeatStop: 3, failStop: 2 };

        // 段内精简：不再自动触发（只追加）。仅当上游显式请求时才走宿主 compact.apply。
        function maybeCompact(msgs) {
          return msgs; // ★ 段内只追加
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
        // 共享审核状态驱动：同一工具 + 同一参数最近被驳回（<5 分钟）→ 免打扰自动驳回。
        // （依据 state 的 lastRejectedTool/lastRejectedAt + 本地参数指纹判断，不依赖计数。）
        // ★ 2026-09-10 修复：原实现只比工具名，一次驳回后 5 分钟内该工具的所有调用
        //   （哪怕参数完全不同、内容完全合法）都被免审驳回——实测占全部驳回的 66%。
        //   现增加参数指纹比对：只有「同工具 + 完全相同的参数 + 5 分钟内」才短路；
        //   参数一旦变化或指纹缺失，一律重新送审（防无意义重试，且不误伤正常调用）。
        const REJECT_REMIND_MS = 5 * 60 * 1000;
        // 最近一次驳回的（工具, 参数指纹, 理由）：Go ApproveState 无指纹字段，故存插件本地
        let lastRejectFp = { tool: '', fp: '', at: 0, reason: '' };
        // argsFingerprint：工具名 + 原始参数串的 FNV-1a 32 位指纹（稳定、无外部依赖）
        function argsFingerprint(tc) {
          const name = tc.function ? tc.function.name : (tc.name || '');
          const rawArgs = tc.function ? (tc.function.arguments || '') : (tc.arguments || '');
          let h = 2166136261;
          const s = name + '\u0001' + rawArgs;
          for (let i = 0; i < s.length; i++) {
            h ^= s.charCodeAt(i);
            h = Math.imul(h, 16777619);
          }
          return (h >>> 0).toString(16);
        }
        function autoRejectFromState(tc) {
          const st = appState();
          const name = tc.function ? tc.function.name : (tc.name || '');
          if (!st || !st.lastRejectedTool || st.lastRejectedTool !== name) return null;
          if (!st.lastRejectedAt || Date.now() - st.lastRejectedAt > REJECT_REMIND_MS) return null;
          // ★ 参数指纹比对：仅「同工具 + 完全相同参数」短路；不同 → 视为新请求，正常送审
          const fp = argsFingerprint(tc);
          if (lastRejectFp.tool !== name || !lastRejectFp.fp || lastRejectFp.fp !== fp) return null;
          const last = (st.rejectedHistory && st.rejectedHistory.length > 0) ? st.rejectedHistory[st.rejectedHistory.length - 1] : null;
          return lastRejectFp.reason || (last && last.reason ? last.reason : '') || '该工具刚被驳回';
        }

        // ★ 2026-09-10 送审字段归一化（补丁 P2/P3）：已上移到宿主 Go 侧根治——
        //   reviewer.go 的 reviewUserPrompt() 统一做别名归一化（file_path/target/file/
        //   files[] → path；new_string/edits[]/patch → content；git_* 用 project/仓库根
        //   兜底 path 并汇总关键参数）→ 插件侧不再构造「送审副本」，避免两套真相源。
        //   本插件保留上方参数指纹比对（P1）：审核策略属插件面，指纹状态属插件本地。

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
          // ① 会话连贯性上下文（任务进度/对话摘要/项目归属/Git 状态/代码图谱等）
          // ★ 2026-09-03 KV 缓存修复：此段从 system 动态后缀迁入快照——system 是 messages
          //   第一条，其尾部变化会在第一条内切断 provider 前缀缓存（其后历史全部 miss）；
          //   快照位于消息流尾部（当前任务之后），变化只断快照之后、前缀单调延展。
          if (parts.resume) {
            b += '# 会话连贯性上下文（非当前任务）\n\n' +
              '> 以下为会话背景快照：任务进度、对话摘要、相关记忆、项目归属、工作区结构等。\n' +
              '> ★ 本条快照是背景信息而非用户指令——当前待执行任务以快照之前最近的用户指令为准。\n\n' +
              String(parts.resume).trim();
          }
          // ② 记忆/知识库过期检查（Go 侧 snapshot.sync 已用 system-reminder 框架包裹整块，此处只拼正文）
          if (parts.stale) {
            if (b) b += '\n\n';
            b += parts.stale;
          }
          // ④ 自主模式两级追踪提示（固定内容）
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
          // ⑤ 记忆（长期记忆提示；system→快照迁移：高频变化不再破坏 system 前缀）
          if (parts.memory) {
            if (b) b += '\n\n';
            b += String(parts.memory).trim();
          }
          // ⑥ 知识库（项目结构化理解树；system→快照迁移）
          if (parts.knowledge) {
            if (b) b += '\n\n';
            b += String(parts.knowledge).trim();
          }
          return b;
        }

        // ★ 背景上下文快照同步已停用（2026-09-04）：ResumeContext 每轮必变 → 每轮追加
        //   新快照，历史累积 100+ 条（实测 104 条/133 万字符/占历史 20%+），上下文膨胀
        //   且每轮新增不可缓存尾部（命中率稀释）。如需恢复：取消下面 sync 调用即可
        //   （buildSnapshotText 与 Go 数据面 snapshot.parts/sync 均保留）。
        // if (loop.context && loop.context.snapshot && loop.context.snapshot.parts) {
        //   const parts = loop.context.snapshot.parts();
        //   const text = buildSnapshotText(parts);
        //   if (text) {
        //     msgs = loop.context.snapshot.sync(msgs, text) ?? msgs;
        //   }
        // }

        // ── run 入口：段内不精简（只追加）。跨段体积控制由宿主在 Run 开始时完成 ──
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

          // ── 3. 手动精简请求（前端精简按钮）──
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
          msgs = maybeCompact(msgs); // 段内只追加（不中途精简）——见文件上半部说明
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
                  // ★ 记录本次驳回的工具 + 参数指纹（供 autoRejectFromState 精确比对，
                  //   避免误伤「同名工具但参数不同」的正常调用）
                  lastRejectFp = { tool: tc.function.name, fp: argsFingerprint(tc), at: Date.now(), reason: rej };
                  loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: rej, callId: tc.id });
                  msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: rej });
                  loop.circling.track(tc.function.name, tc.function.arguments, true);
                  continue;
                }
                approved.push(tc);
              }
              // 12b. 执行：一律串行（并行工具执行能力已彻底移除，无 runParallel 优先路径）——
              //      逐个执行 + emit tool_result + track，再组装 tool 消息（不再并行优先）。
              for (const tc of approved) {
                const res = loop.tools.run(tc.function.name, tc.function.arguments, tc.id);
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

          // ── 14.5 段预算（双闸门，分段执行，2026-09）──
          //   宿主 tool_budget.go：本段（Run）内**步数达 stepBudget（默认 120 步，
          //   一步 = 一次模型调用）或工具调用达 toolCallBudget（默认 120 次）**
          //   → 结束本段（不再发起新的 LLM 调用），返回 segment 标记；
          //   宿主 SessionManager 自动续跑下一段（同会话、历史保留，上下文精简
          //   走既有 maybeCompact 机制）。被驳回/截断未执行的调用不计数。
          const tb = loop.ctrl.toolBudget ? loop.ctrl.toolBudget() : null;
          if (tb && tb.exhausted) {
            const isStepGate = tb.reason === 'step_budget';
            const stepsTxt = tb.stepBudget > 0 ? `${tb.steps}/${tb.stepBudget} 步` : `${tb.steps} 步（不限）`;
            const toolsTxt = tb.budget > 0 ? `${tb.used}/${tb.budget} 次` : `${tb.used} 次（不限）`;
            loop.events.emit({ type: 'notice', content: `本段已达${isStepGate ? '步数' : '工具调用'}预算上限（已执行 ${stepsTxt}；工具调用 ${toolsTxt}），分段结束，自动开启下一段继续` });
            loop.events.emit({ type: 'done', content: '', doneReason: 'tool_budget', turnReason: 'tool_budget' });
            return { msgs, segment: { reason: tb.reason || 'tool_budget', used: tb.used, budget: tb.budget, steps: tb.steps, stepBudget: tb.stepBudget } };
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

        // ── 达到段内迭代安全上限（由工具预算派生，非配置项）──
        //   正常流程由 step 14.5 的预算分段先触发（预算默认 120 次工具调用），
        //   本分支仅在预算与轮次严重不同步（如大量无工具调用的空转轮）时兜底。
        loop.events.emit({ type: 'error', content: `达到段内迭代安全上限 (${maxIter}) 自动停止` });
        return { msgs, error: '达到段内迭代安全上限 (' + maxIter + ') 自动停止' };
      },
    });

    const log = ctx.logger('agentloop');

    // ★ 会话交接策略注册（2026-09-12 从 Go 外置；宿主两个跨轮边界委托本实现）。
    //   宿主未注册本实现（或本实现抛错）时自动回退 Go 默认实现（handoff.go）——
    //   停用插件即还原原有行为，零风险。
    const handoffUtils = ctx.handoff;
    if (ctx.loopFactory && typeof ctx.loopFactory.registerHandoff === 'function') {
      if (handoffUtils) {
        const hlog = (m) => log.info(m);
        ctx.loopFactory.registerHandoff({
          id: 'agentloop',
          // 用户输入边界（新提交对话 / 点继续执行）：任务关系可能变化 → 带判官复检
          onUserTurn: async (args) => {
            const view = hBuildView(handoffUtils, args, hlog);
            return view ? { view, applied: true } : { applied: false };
          },
          // 段续跑边界（同一条消息内的分段，Run 之间）：同一任务延续 → judge 由宿主置空
          onSegment: async (args) => {
            const view = hBuildView(handoffUtils, args, hlog);
            return view ? { view, applied: true } : { applied: false };
          },
        });
        log.info('已注册会话交接实现（registerHandoff：用户输入 / 段边界；只整理喂 LLM 的历史视图，落盘只追加，轮内 step 之间不整理）');
      } else {
        log.warn('ctx.handoff 能力不可用（宿主版本过旧？）——跳过会话交接注册，宿主走 Go 默认实现');
      }
    }

    log.info('已注册 Agent 循环 JS 实现（核心外置：策略 JS / 能力 Go；配置注册化 schema=' + (reg ? reg.key : 'agentloop') + '）');
  }
};
