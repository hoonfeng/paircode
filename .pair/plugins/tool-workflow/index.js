// ═══════════════════════════════════════════════════════════════
// tool-workflow — workflow 编排工具（Round3 ③.3，对齐 workflow 范式）
//
// 编排在插件、能力在宿主：schema/描述在插件，运行器在宿主
// （internal/agent/workflow.go，goja 执行 + agent/pipeline/parallel 钩子）。
// execute 经 ctx.hostTool.exec 路由回宿主执行器。
// ═══════════════════════════════════════════════════════════════
const tools = [
  {
    name: 'workflow',
    description:
      '执行一个 workflow 编排脚本（对齐 workflow 范式）。script 为 JS 函数体（末尾 return 结果）；钩子：agent(prompt, opts?) 后台委托子 Agent 并等待完成返回其最终正文；pipeline(items, ...stages) 逐项过阶段（阶段抛错该项为 null）；parallel(thunks) 批量执行（宿主侧并发，barrier 等待）；phase(title)/log(msg) 记录进度；args 为脚本内可读输入。返回 JSON {ok, output, logs, phases}。',
    parameters: {
      type: 'object',
      properties: {
        script: { type: 'string', description: 'workflow 脚本体（JS，末尾 return 结果；可用 agent/pipeline/parallel/phase/log/args）' },
        meta: { type: 'string', description: '可选：{name, description, phases} 元信息（记录用）' },
        args: { type: 'string', description: '可选：输入参数 JSON 对象（脚本内 args 全局可读）' },
      },
      required: ['script'],
    },
  },
];

return {
  name: 'tool-workflow',
  purpose: 'workflow 编排工具面（宿主 goja 运行器：agent/pipeline/parallel/phase/log）',
  apply(ctx) {
    for (const t of tools) {
      ctx.tools.register({
        name: t.name,
        description: t.description,
        usageGuide: t.usageGuide,
        category: t.category,
        readOnly: t.readOnly,
        requiresApproval: t.requiresApproval,
        systemTool: t.systemTool,
        parameters: t.parameters,
        execute: (args) => ctx.hostTool.exec(t.name, args || {}),
      });
    }
  },
};
