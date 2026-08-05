/* real_data.js — 折叠状态模拟的真实会话数据
 *
 * 数据来源（优先级）：
 *   1. window.__CONV_JSONL__：宿主（fold_sim.exe -conv <jsonl>）注入的
 *      历史对话 jsonl 全文 → 由内置解析器 parseConversationJsonl 直接解析生成；
 *   2. FALLBACK_COMBOS：内置手工数据（无注入时回退，保持页面可独立演示）。
 *
 * 解析器 parseConversationJsonl(text) 与 conv_real.py 逻辑等价：
 *   - user 消息        → {user:{content}}
 *   - assistant 消息   → {assistant:{_folded,_summary,segments:[thinking/tool_call/content]}}
 *   - tool 消息的结果  → 并入上一条 tool_call 段的 result
 *   - 前 FOLD_FIRST 个 assistant 回合标记 _folded=true（演示整条折叠）
 *   - 最多保留 MAX_TURNS 个 assistant 回合
 */
var FALLBACK_COMBOS = [
 // ══ 完整 run：一个 agent 气泡完整输出一次 run 内的全部消息 ══
 // （思考 → 工具调用 → 正文 → 思考 → 工具调用 → 正文…完整循环，全部展开）
 {
  "assistant": {
   "_folded": true,
   "_summary": "已定位终端主题色不生效根因：term.value 在 onMounted 时为空导致 theme 赋值被吞。修复 loadTheme 空值保护并在实例创建后设置 theme，终端恢复深色 #0d1117。点击展开可查看完整 run（3 次思考 + 3 次工具调用 + 2 段正文）。",
   "_time": "10:05",
   "segments": [
    {
     "type": "thinking",
     "content": "用户报告终端面板 xterm 主题色没生效，背景一直是默认黑色而不是 IDE 深色主题的 #0d1117。\n\n先看 TerminalPanel 是怎么设置主题的——大概率是读 CSS 变量 --term-bg。如果 getComputedStyle 拿不到变量值（桩实现返回空），xterm 就回退默认。",
     "_collapsed": false
    },
    {
     "type": "tool_call",
     "name": "search_content",
     "argsRaw": "{\"glob\": \"*.vue\", \"max_results\": 15, \"path\": \"F:\\\\syproject\\\\gou-ide\\\\cmd\\\\companion\\\\web-ui\\\\src\", \"pattern\": \"--term-bg|term-bg\"}",
     "result": "cmd/companion/web-ui/src/components/TerminalPanel.vue:42: const termBg = getComputedStyle(document.documentElement).getPropertyValue('--term-bg')\ncmd/companion/web-ui/src/components/TerminalPanel.vue:43: const termFg = getComputedStyle(document.documentElement).getPropertyValue('--term-fg')\ncmd/companion/web-ui/src/index.css:105: :root { --term-bg: #0d1117; --term-fg: #d4d4d4; }\ncmd/companion/web-ui/src/index.css:106: .theme-light { --term-bg: #ffffff; --term-fg: #1f2328; }",
     "_expanded": true
    },
    {
     "type": "content",
     "content": "确认了：TerminalPanel 通过 `getComputedStyle(document.documentElement).getPropertyValue('--term-bg')` 读取主题色。变量定义在 `:root`，值为 #0d1117。\n\n如果拿到的是空串，xterm 就会用默认背景。下一步验证 getComputedStyle 的级联计算是否返回该变量。"
    },
    {
     "type": "thinking",
     "content": "getComputedStyle 的 CSS 变量级联是最近修复的（verify_bindings_fixes 有 --main-bg/--accent 断言）。但 TerminalPanel 读的是 documentElement（html 元素）上的变量，而测试断言的是 div 元素。:root 选择器特异性是伪类级别，应该能匹配 html。\n\n另一个可能：getComputedStyle 返回的变量值带引号或空格，xterm 的 theme 对象拿到 ' #0d1117' 之类。先跑验证程序确认。",
     "_collapsed": false
    },
    {
     "type": "tool_call",
     "name": "run_command",
     "argsRaw": "{\"command\": \"set CGO_ENABLED=1 && go run ./dev/verify_bindings_fixes\"}",
     "result": "[1] prefers-color-scheme 动态化\n  ✓ matchMedia(dark).matches = \"true\"\n  ✓ matchMedia(light).matches = \"false\"\n[2] CSS.supports 选择器形式\n  ✓ CSS.supports('.foo:hover') = \"true\"\n[3] getComputedStyle 级联计算\n  ✓ id=a --main-bg（CSS 变量级联） = \"#fff\"\n  ✓ id=a --accent = \"#ff0\"\n  ✓ id=b --main-bg（inline 覆盖变量） = \"#000\"\n[4] 样式表缓存 + 动态注入\n  ✓ 注入后 --x 生效（缓存失效） = \"#abc\"\n[5] 指针能力动态化\n  ✓ hover:none = \"true\"\n结果：25 通过，0 失败",
     "_expanded": true
    },
    {
     "type": "thinking",
     "content": "25/25 全过，级联计算本身没问题。那问题可能在 TerminalPanel 的读取时机——如果它在主题 class 应用之前就读取，或者 getComputedStyle 返回的字符串被 xterm 解析异常。\n\n读 TerminalPanel.vue 的 loadTheme 逻辑确认。",
     "_collapsed": false
    },
    {
     "type": "tool_call",
     "name": "read_file",
     "argsRaw": "{\"limit\": 40, \"offset\": 35, \"path\": \"F:\\\\syproject\\\\gou-ide\\\\cmd\\\\companion\\\\web-ui\\\\src\\\\components\\\\TerminalPanel.vue\"}",
     "result": "  const loadTheme = () => {\n    const termBg = getComputedStyle(document.documentElement).getPropertyValue('--term-bg').trim()\n    const termFg = getComputedStyle(document.documentElement).getPropertyValue('--term-fg').trim()\n    term.value.options.theme = { background: termBg || '#0d1117', foreground: termFg || '#d4d4d4' }\n  }\n  onMounted(loadTheme)\n  watch(() => props.themeKey, loadTheme)",
     "_expanded": false
    },
    {
     "type": "content",
     "content": "根因找到：`term.value` 在 `onMounted(loadTheme)` 时可能还是 null（xterm 实例用 ref 延迟初始化），`term.value.options.theme = ...` 抛异常被吞，主题就永远不生效。\n\n修复：loadTheme 里加空值保护，并把 `term.value.options.theme = {...}` 改为在实例创建后调用 `term.value.options.theme`（xterm 支持运行中改主题）。修改后终端主题色恢复正常，深色 #0d1117 正确渲染。"
    }
   ]
  }
  }
];


/* ── 内置解析器：jsonl 全文 → combos 数组（与 conv_real.py 逻辑等价） ── */
(function (global) {
  'use strict';

  function parseConversationJsonl(jsonlText, opts) {
    opts = opts || {};
    var MAX_TURNS = opts.maxTurns || 6;
    var FOLD_FIRST = opts.foldFirst || 2;
    var RESULT_LIMIT = opts.resultLimit || 4000;

    var lines = String(jsonlText || '').split('\n');
    var combos = [];
    var cur = null;

    function flush() {
      if (cur !== null && cur.assistant && cur.assistant.segments && cur.assistant.segments.length) {
        combos.push(cur);
      }
      cur = null;
    }

    for (var i = 0; i < lines.length; i++) {
      var line = lines[i].trim();
      if (!line) continue;
      var rec;
      try { rec = JSON.parse(line); } catch (e) { continue; }
      var m = rec.message || {};
      var segs = rec.segments || [];
      var role = m.role;

      if (role === 'user') {
        flush();
        combos.push({ user: { content: m.content || '' } });
      } else if (role === 'assistant') {
        flush();
        cur = {
          assistant: {
            _folded: false,
            _summary: (m.content || '').slice(0, 60),
            segments: []
          }
        };
        for (var j = 0; j < segs.length; j++) {
          var s = segs[j];
          var t = s.type;
          if (t === 'thinking') {
            cur.assistant.segments.push({ type: 'thinking', content: s.content || '', _collapsed: true });
          } else if (t === 'tool_call') {
            cur.assistant.segments.push({
              type: 'tool_call',
              name: s.name || '',
              argsRaw: s.argsRaw || '',
              result: (s.result || '').slice(0, RESULT_LIMIT),
              _expanded: false
            });
          } else if (t === 'content') {
            cur.assistant.segments.push({ type: 'content', content: s.content || '' });
          }
          // tool_result 段：结果已并入 tool_call 的 result 字段，跳过
        }
      } else if (role === 'tool') {
        // tool 消息：内容并入上一 tool_call 的 result（若为空）
        if (cur !== null && cur.assistant && cur.assistant.segments && cur.assistant.segments.length) {
          var last = cur.assistant.segments[cur.assistant.segments.length - 1];
          if (last.type === 'tool_call' && !last.result) {
            last.result = (m.content || '').slice(0, RESULT_LIMIT);
          }
        }
      }
    }
    flush();

    // 只保留段数最多的 assistant 回合：一个 agent 输出气泡完整展示一次 run 内的全部消息
    // （丢弃 user 气泡与其他回合，避免页面零散多个气泡；该回合默认整条折叠显示完成摘要）
    var best = null;
    var bestCount = -1;
    for (var k = 0; k < combos.length; k++) {
      var ck = combos[k];
      if (!ck.assistant || !ck.assistant.segments) continue;
      var n = ck.assistant.segments.length;
      if (n > bestCount) {
        bestCount = n;
        best = ck;
      }
    }
    if (best === null) return [];
    best.assistant._folded = true;
    // 摘要取正文段最后一段（无正文时用工具结果摘要）
    var segs = best.assistant.segments || [];
    var summaryText = '';
    for (var j2 = segs.length - 1; j2 >= 0; j2--) {
      if (segs[j2].type === 'content' && segs[j2].content) {
        summaryText = segs[j2].content;
        break;
      }
    }
    if (!summaryText) {
      for (var j3 = segs.length - 1; j3 >= 0; j3--) {
        if (segs[j3].type === 'tool_call' && segs[j3].result) {
          summaryText = segs[j3].result;
          break;
        }
      }
    }
    if (!summaryText) summaryText = 'agent 输出（点击展开查看完整 run）';
    summaryText = summaryText.replace(/\s+/g, ' ').trim();
    if (summaryText.length > 90) summaryText = summaryText.slice(0, 90) + '…';
    best.assistant._summary = summaryText;
    return [best];
  }
  global.parseConversationJsonl = parseConversationJsonl;

  // ── 数据装配：注入优先，无注入回退内置数据 ──
  var injected = global.__CONV_JSONL__;
  var combos = null;
  if (injected && typeof injected === 'string' && injected.length > 10) {
    try {
      combos = parseConversationJsonl(injected);
    } catch (e) { combos = null; }
  }
  global.REAL_COMBOS = (combos && combos.length) ? combos : FALLBACK_COMBOS;
})(window);