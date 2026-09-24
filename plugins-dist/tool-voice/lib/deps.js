'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/deps.js — @audio/* ESM 依赖懒加载（CJS 内动态 import）
//
// 为什么懒加载：
//  ① goja 沙箱轨装载时（.pair/plugins 扫描）不触发本模块——index.js 先做
//     运行时自检让位（见 index.js isNodeRuntime）；
//  ② Node 桥轨的 apply() 可同步注册工具，首次调用时再加载依赖——
//     避免 apply 阶段异步加载失败导致「插件装载失败」；
//  ③ 依赖一次性缓存，跨工具调用复用（YIN/PSOLA 无状态）。
// ═══════════════════════════════════════════════════════════════

let cached = null;
let loading = null;

async function loadDeps() {
  if (cached) return cached;
  if (loading) return loading;
  loading = (async () => {
    const [snap, wsola, yin, onset, loud, note] = await Promise.all([
      import('@audio/tune-snap'),
      import('@audio/stretch-wsola'),
      import('@audio/pitch-yin'),
      import('@audio/onset'),
      import('@audio/loudness'),
      import('@audio/note'),
    ]);
    cached = {
      snap: snap.default,
      wsola: wsola.default,
      yin: yin.default,
      spectralFlux: onset.spectralFlux,
      energyFlux: onset.energyFlux,
      peakPick: onset.peakPick,
      lufs: loud.lufs,
      truepeak: loud.truepeak,
      lra: loud.lra,
      note,
      versions: {
        'tune-snap': '1.1.3',
        'stretch-wsola': '1.2.1',
        'pitch-yin': '1.0.5',
        onset: '1.0.1',
        loudness: '1.2.1',
        note: '1.0.2',
      },
    };
    return cached;
  })();
  return loading;
}

function depsLoaded() {
  return cached !== null;
}

// depAudit：本插件声明的依赖清单（供 7.1 依赖审计比对白名单）。
const DECLARED = {
  '@audio/loudness': '1.2.1',
  '@audio/note': '1.0.2',
  '@audio/onset': '1.0.1',
  '@audio/pitch-yin': '1.0.5',
  '@audio/stretch-wsola': '1.2.1',
  '@audio/tune-snap': '1.1.3',
};

module.exports = { loadDeps, depsLoaded, DECLARED };
