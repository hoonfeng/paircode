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
//     loop.approve.ask(tc) → {approved, feedback, blocked, reason}
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
//     - 工具执行 + 审核决策（loop.approve.ask 含黑白名单/连续驳回）
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
    // ── 配置注册化：声明本插件需要的配置段（前端设置面板动态渲染）──
    const reg = ctx.registerSettings({
      key: 'agentloop',
      title: 'Agent 循环',
      fields: [
        { name: 'systemAppend', label: '系统提示词追加', type: 'textarea', default: '', hint: '追加到系统提示词末尾（如行为规范/角色设定）' },
        { name: 'maxIterations', label: '最大迭代数', type: 'number', default: 0, hint: '0=不覆盖（默认 50）' },
        { name: 'maxContextTokens', label: '上下文 token 上限', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'autonomous', label: '自主模式', type: 'checkbox', default: false, hint: '覆盖全局自主模式开关' },
        { name: 'maxAutonomousMinutes', label: '自主时间预算（分钟）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'checkpointInterval', label: '检查点间隔（迭代数）', type: 'number', default: 0, hint: '0=不覆盖' },
        { name: 'reviewMode', label: '审核模式', type: 'select', default: '', options: ['', 'auto', 'manual', 'off'], hint: '空=不覆盖（auto=AI审核/manual=手动审批/off=放行）' },
        { name: 'autoCommit', label: '完成自动提交', type: 'checkbox', default: false, hint: '覆盖完成自动提交开关' },
        { name: 'reviewBlacklist', label: '审核黑名单', type: 'text', default: '', hint: '逗号分隔工具名（命中需审核）' },
        { name: 'reviewWhitelist', label: '审核白名单', type: 'text', default: '', hint: '逗号分隔工具名（命中跳过审核，黑名单优先）' },
      ],
    });

    // ── 运行时读取配置（registerSettings 返回当前值=已存值合并默认）──
    const cfg = (reg && reg.value) || ctx.getSettings('agentloop') || {};

    // ── 参数级装配（保留）：CreateLoop 时覆盖装配参数（提示词/迭代/审核模式）──
    ctx.loopFactory.register((opts) => {
      const over = {};
      if (typeof cfg.systemAppend === 'string' && cfg.systemAppend) {
        over.system = (opts.system || '') + '\n\n' + cfg.systemAppend;
      }
      if (cfg.maxIterations != null && Number(cfg.maxIterations) > 0) over.maxIterations = Number(cfg.maxIterations);
      if (cfg.maxContextTokens != null && Number(cfg.maxContextTokens) > 0) over.maxContextTokens = Number(cfg.maxContextTokens);
      if (typeof cfg.autonomous === 'boolean') over.autonomous = cfg.autonomous;
      if (cfg.maxAutonomousMinutes != null && Number(cfg.maxAutonomousMinutes) > 0) over.maxAutonomousMinutes = Number(cfg.maxAutonomousMinutes);
      if (cfg.checkpointInterval != null && Number(cfg.checkpointInterval) > 0) over.checkpointInterval = Number(cfg.checkpointInterval);
      if (typeof cfg.reviewMode === 'string' && cfg.reviewMode) over.reviewMode = cfg.reviewMode;
      if (typeof cfg.autoCommit === 'boolean') over.autoCommit = cfg.autoCommit;
      if (typeof cfg.reviewBlacklist === 'string' && cfg.reviewBlacklist) {
        over.reviewBlacklist = cfg.reviewBlacklist.split(/[,，]/).map(s => s.trim()).filter(Boolean);
      }
      if (typeof cfg.reviewWhitelist === 'string' && cfg.reviewWhitelist) {
        over.reviewWhitelist = cfg.reviewWhitelist.split(/[,，]/).map(s => s.trim()).filter(Boolean);
      }
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
            msgs = loop.compact(msgs);
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
              // 12a. 审批门（黑白名单 + ReviewMode + RequiresApproval + 连续驳回，Go 侧实现）
              const approved = [];
              let blocked = false, blockReason = '';
              for (const tc of tcs) {
                loop.events.emit({ type: 'tool_call', tool: tc.function.name, args: tc.function.arguments, callId: tc.id });
                const ap = loop.approve.ask(tc);
                if (!ap.approved) {
                  const rej = (ap.feedback || '').trim() || REJ_DEFAULT;
                  if (ap.blocked) { blocked = true; blockReason = ap.reason; break; }
                  loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: rej, callId: tc.id });
                  msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: rej });
                  loop.circling.track(tc.function.name, tc.function.arguments, true);
                  continue;
                }
                approved.push(tc);
              }
              if (blocked) {
                loop.events.emit({ type: 'error', content: blockReason });
                return { msgs, error: blockReason };
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
                    if (r.name === 'generate_commit_message') loop.store.set('commitMessage', output);
                  }
                } else {
                  for (const tc of approved) {
                    const res = loop.tools.run(tc.function.name, tc.function.arguments);
                    const output = res.error ? 'Error: ' + res.error : res.content;
                    loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: output, callId: tc.id });
                    msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: output });
                    loop.circling.track(tc.function.name, tc.function.arguments, !!res.error);
                    if (tc.function.name === 'generate_commit_message') loop.store.set('commitMessage', output);
                  }
                }
              } else if (approved.length === 1) {
                const tc = approved[0];
                const res = loop.tools.run(tc.function.name, tc.function.arguments);
                const output = res.error ? 'Error: ' + res.error : res.content;
                loop.events.emit({ type: 'tool_result', tool: tc.function.name, content: output, callId: tc.id });
                msgs.push({ role: 'tool', toolCallId: tc.id, name: tc.function.name, content: output });
                loop.circling.track(tc.function.name, tc.function.arguments, !!res.error);
                if (tc.function.name === 'generate_commit_message') loop.store.set('commitMessage', output);
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

          // ── 16. 绕圈检测：重复操作/反复失败 → 注入「换思路」──
          const nudge = loop.circling.detect();
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
