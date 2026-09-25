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
//            usage?, usageEstimated?, content?, reasoning?, toolCalls?, stopReason, err?}
//   ★ usage = **API 真实返回**的用量：{source:"api", promptTokens, completionTokens,
//     totalTokens, promptCacheHitTokens, promptCacheMissTokens, cacheHitRate?, apiRaw?}
//     · 前六项为 provider 报文归一化后的真实值（**非**本地估算）；
//     · apiRaw = provider **原始报文**（未归一化），含各协议/网关的扩展字段：
//       OpenAI prompt_tokens_details.cached_tokens、DeepSeek prompt_cache_hit_tokens
//       /prompt_cache_miss_tokens、reasoning_tokens 等——核对真实用量以此为准。
//   ★ usageEstimated = 本地**估算**的 prompt 构成：{source:"local-estimate",
//     systemTokens, skillsTokens, mcpTokens, toolTokens, historyTokens, otherTokens}
//     · 仅用于占比可视化参考，**不得**当作真实用量做成本/命中率分析。
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
    // ★ 目录确保**延迟**到首次写入（见 ensureDir）：无工作区 / 未绑定会话时
    //   ctx.fs 无法解析相对路径（GoError: 工作区根为空），此前在 apply 里直接
    //   fs.mkdir 会**抛出并中断整个插件装载** —— 每次启动日志出现
    //   「llm-trace 装载失败」，该插件的 LLM 追踪能力全程缺失。
    //   改为降级：apply 阶段不再触碰 fs（注册照常完成），目录在工作区就绪后的
    //   首次写入时创建；届时仍失败则只在写入路径记错，不影响插件装载。
    let dirReady = false;
    function ensureDir() {
      if (dirReady) {
        return true;
      }
      try {
        fs.mkdir(dir, true);
        dirReady = true;
      } catch (e) {
        // 无工作区/未绑定会话：保持未就绪，下次写入再试（不抛出、不中断装载）
        return false;
      }
      return true;
    }
    // 尽力而为：有工作区时启动即建目录（失败静默，交给 ensureDir 重试）。
    try {
      fs.mkdir(dir, true);
      dirReady = true;
    } catch (e) {
      log.info('[llm-trace] 暂无可解析的工作区，目录将在首次写入时创建: ' + dir);
    }

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
        if (!ensureDir()) {
          // 尚无工作区：跳过本次写入（会话/工作区就绪后自动恢复）
          return;
        }
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
