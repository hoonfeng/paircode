// llm-trace 插件：LLM 请求/响应完整追踪（JSONL 落盘），用于分析 provider
// 前缀缓存断裂（系统提示/工具定义跨轮次变化导致命中率下降）。
//
// 数据面：ctx.llmtrace.register(fn)（宿主 agentloop JS 循环 llm.chat 前后触发），
//   注册即生效，每次 register 覆盖旧回调（单注册语义），插件卸载自动撤销。
//
// 输出：<dir>/llm-trace-YYYYMMDD.jsonl（每行一个事件 JSON，request/response 两相）。
//   - dir 由 config.dir 指定（默认 .pair/logs/llm-trace，相对工作区根）；
//   - 按天分文件；单文件超 config.maxFileMB（默认 64）时切换序号文件
//     llm-trace-YYYYMMDD-<n>.jsonl，避免无界增长；
//   - 每条事件保留完整 msgs/tools/usage/content/reasoning/toolCalls——
//     「缓存前缀断裂」分析需要完整内容比对（与仅形状哈希的 WB_CACHE_DIAG 互补）。
//   - 敏感注意：日志含完整对话与工具定义，勿提交到公开仓库（.gitignore 建议加 <dir>）。
//
// 事件字段：{phase: request|response, when, turn, step, provider, msgs?, tools?,
//            usage?, content?, reasoning?, toolCalls?, stopReason, err?}
//   usage 含 promptCacheHitTokens/promptCacheMissTokens + prompt 构成细分
//   （systemTokens/skillsTokens/mcpTokens/toolTokens/historyTokens/otherTokens）。
//
// 查看/分析：node 读 JSONL 逐行 JSON.parse；或用 grep 对比相邻请求的
//   msgs[0].content（系统提示）与 tools JSON 是否逐字节一致。

return {
  name: 'llm-trace',
  inject: ['fs', 'logger'],
  apply(ctx, config) {
    const dir = (config && config.dir) || '.pair/logs/llm-trace';
    const maxFileMB = (config && config.maxFileMB) || 64;

    const fs = ctx.fs;
    const log = ctx.logger('llm-trace');
    // 启动时确保目录存在（appendFile 不会自动建目录）
    fs.mkdir(dir, true);

    // 当前写入文件路径（按天 + 超限序号轮转）
    let curPath = null;
    let curSize = 0;

    function todayStr() {
      const d = new Date();
      const p = (n) => String(n).padStart(2, '0');
      return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}`;
    }

    function nextPath() {
      const day = todayStr();
      if (!curPath || !curPath.endsWith(`-${day}.jsonl`)) {
        curPath = `${dir}/llm-trace-${day}.jsonl`;
        curSize = 0;
        return curPath;
      }
      // 按天文件已有但超限 → 轮转序号
      if (curSize >= maxFileMB * 1024 * 1024) {
        let n = 1;
        let cand = `${dir}/llm-trace-${day}-${n}.jsonl`;
        while (fs.exists(cand)) {
          n += 1;
          cand = `${dir}/llm-trace-${day}-${n}.jsonl`;
        }
        curPath = cand;
        curSize = 0;
      }
      return curPath;
    }

    function appendLine(obj) {
      try {
        const line = JSON.stringify(obj);
        const p = nextPath();
        fs.appendFile(p, line + '\n');
        curSize += line.length + 1;
      } catch (e) {
        log.error('[llm-trace] 写日志失败: ' + e);
      }
    }

    ctx.llmtrace.register((ev) => {
      // 瘦身：request 相 msgs 可能很大；不裁剪（分析需要完整内容）
      appendLine(ev);
    });

    log.info(`[llm-trace] 已注册 LLM 追踪，输出目录: ${dir}`);
  },
};
