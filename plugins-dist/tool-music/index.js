// tool-music — 音乐创作工具面（纯 goja 零依赖）
//
// 设计原则（对齐《创作域支持方案》§195「真相源必须文本」）：
//   · **真相源 = 工程 JSON**（music.project.json：轨道/音符/速度/拍号/调号），全部可 diff；
//   · 二进制（MIDI）与图形（SVG 乐谱）只是**产物**，随时可由工程重新生成；
//   · 音符时间一律用 **tick 整型**（ppq 默认 480）——避免浮点漂移，保证位级确定性。
//
// 本插件是磁盘 goja 轨插件：沙箱内没有 require / Buffer / Node API，
// 一律走 ctx 服务（ctx.fs）。二进制安全靠宿主 ctx.fs.readFileBase64 / writeFileBase64，
// base64 编解码由本文件自带（不依赖 btoa/atob，Node 侧亦可直接测试）。

// ── 常量表 ─────────────────────────────────────────────────
var PPQ_DEFAULT = 480;

var NOTE_NAMES = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B'];

// 音阶（半音间隔）
var SCALES = {
  major: [0, 2, 4, 5, 7, 9, 11],
  minor: [0, 2, 3, 5, 7, 8, 10],
  harmonicMinor: [0, 2, 3, 5, 7, 8, 11],
  melodicMinor: [0, 2, 3, 5, 7, 9, 11],
  dorian: [0, 2, 3, 5, 7, 9, 10],
  phrygian: [0, 1, 3, 5, 7, 8, 10],
  lydian: [0, 2, 4, 6, 7, 9, 11],
  mixolydian: [0, 2, 4, 5, 7, 9, 10],
  locrian: [0, 1, 3, 5, 6, 8, 10],
  pentatonicMajor: [0, 2, 4, 7, 9],
  pentatonicMinor: [0, 3, 5, 7, 10],
  chromatic: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11],
};

// 和弦（相对根音的半音）
var CHORD_QUALITIES = {
  major: [0, 4, 7],
  minor: [0, 3, 7],
  dim: [0, 3, 6],
  aug: [0, 4, 8],
  sus2: [0, 2, 7],
  sus4: [0, 5, 7],
  maj7: [0, 4, 7, 11],
  min7: [0, 3, 7, 10],
  dom7: [0, 4, 7, 10],
  dim7: [0, 3, 6, 9],
  halfDim7: [0, 3, 6, 10],
  add9: [0, 4, 7, 14],
  maj9: [0, 4, 7, 11, 14],
  min9: [0, 3, 7, 10, 14],
  power: [0, 7, 12],
};

// SMF meta 事件类型
var META_TEMPO = 0x51;
var META_TIME_SIG = 0x58;
var META_KEY_SIG = 0x59;
var META_TRACK_NAME = 0x03;
var META_END = 0x2f;

// ── 基础工具 ───────────────────────────────────────────────
function pad2(n) { return n < 10 ? '0' + n : '' + n; }
function isInt(v) { return typeof v === 'number' && isFinite(v) && Math.floor(v) === v; }
function clamp(v, lo, hi) { return v < lo ? lo : (v > hi ? hi : v); }
function nowIso() { return new Date().toISOString(); }

// midi → 音名（C4 = 60，Scientific Pitch Notation）
function pitchName(p) {
  var n = ((p % 12) + 12) % 12;
  var oct = Math.floor(p / 12) - 1;
  return NOTE_NAMES[n] + oct;
}

// 音名 → midi（支持 C4 / C#4 / Db4 / C-1）
var LETTER_SEMI = { C: 0, D: 2, E: 4, F: 5, G: 7, A: 9, B: 11 };
function parsePitchName(name) {
  if (typeof name === 'number' && isInt(name)) return name;
  var m = /^([A-Ga-g])([#b]{0,2})(-?\d+)$/.exec(String(name).trim());
  if (!m) throw new Error('无法解析音名: ' + name + '（示例：C4 / F#5 / Bb3）');
  var semi = LETTER_SEMI[m[1].toUpperCase()];
  var acc = m[2];
  for (var i = 0; i < acc.length; i++) semi += acc[i] === '#' ? 1 : -1;
  return semi + (parseInt(m[3], 10) + 1) * 12;
}

// 调号名 → 主音 pitch class（如 "C" / "Am" / "F# minor"）
function parseKey(key) {
  var s = String(key || 'C').trim();
  var minor = false;
  var m = /^([A-Ga-g])([#b]?)\s*(m|min|minor|小调)?$/i.exec(s);
  if (!m) throw new Error('无法解析调号: ' + key + '（示例：C / Am / F# minor）');
  var semi = LETTER_SEMI[m[1].toUpperCase()] + (m[2] === '#' ? 1 : (m[2] === 'b' ? -1 : 0));
  if (m[3] && /^m$|min/i.test(m[3])) minor = true;
  return { root: ((semi % 12) + 12) % 12, minor: minor };
}

function keyToString(key) { return key.minor ? NOTE_NAMES[key.root] + 'm' : NOTE_NAMES[key.root]; }

// ── base64（自带，二进制安全；不依赖 btoa/atob）────────────
var B64_TABLE = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
var B64_REV = null;

function b64Encode(bytes) {
  var out = [];
  for (var i = 0; i < bytes.length; i += 3) {
    var b0 = bytes[i] & 0xff;
    var b1 = i + 1 < bytes.length ? bytes[i + 1] & 0xff : -1;
    var b2 = i + 2 < bytes.length ? bytes[i + 2] & 0xff : -1;
    out.push(B64_TABLE[b0 >> 2]);
    out.push(B64_TABLE[((b0 & 3) << 4) | (b1 >= 0 ? b1 >> 4 : 0)]);
    out.push(b1 >= 0 ? B64_TABLE[((b1 & 15) << 2) | (b2 >= 0 ? b2 >> 6 : 0)] : '=');
    out.push(b2 >= 0 ? B64_TABLE[b2 & 63] : '=');
  }
  return out.join('');
}

function b64Decode(str) {
  if (!B64_REV) {
    B64_REV = {};
    for (var i = 0; i < B64_TABLE.length; i++) B64_REV[B64_TABLE[i]] = i;
  }
  var s = String(str).replace(/[\r\n\t ]/g, '');
  var out = [];
  var buf = 0;
  var bits = 0;
  for (var j = 0; j < s.length; j++) {
    var ch = s[j];
    if (ch === '=') break;
    var v = B64_REV[ch];
    if (v === undefined) throw new Error('非法 base64 字符: ' + ch);
    buf = (buf << 6) | v;
    bits += 6;
    if (bits >= 8) {
      bits -= 8;
      out.push((buf >> bits) & 0xff);
    }
  }
  return out;
}

// ── VLQ（MIDI 可变长量）────────────────────────────────────
function vlqBytes(n) {
  if (!isInt(n) || n < 0) throw new Error('VLQ 需要非负整数，收到: ' + n);
  var buf = n & 0x7f;
  var out = [];
  n = Math.floor(n / 128);
  while (n > 0) {
    buf = (buf << 8) | 0x80 | (n & 0x7f);
    n = Math.floor(n / 128);
  }
  for (;;) {
    out.push(buf & 0xff);
    if (buf & 0x80) buf >>= 8; else break;
  }
  return out;
}

function readVlq(bytes, pos) {
  var value = 0;
  var b;
  var guard = 0;
  do {
    if (pos >= bytes.length) throw new Error('VLQ 读取越界');
    b = bytes[pos++];
    value = (value << 7) | (b & 0x7f);
    if (++guard > 4) throw new Error('VLQ 超过 4 字节（文件可能损坏）');
  } while (b & 0x80);
  return { value: value, pos: pos };
}

// ── SMF（标准 MIDI 文件）编码 ───────────────────────────────
function be32(n) { return [(n >>> 24) & 0xff, (n >>> 16) & 0xff, (n >>> 8) & 0xff, n & 0xff]; }
function be16(n) { return [(n >>> 8) & 0xff, n & 0xff]; }
function asciiBytes(s) { var out = []; for (var i = 0; i < s.length; i++) out.push(s.charCodeAt(i) & 0xff); return out; }
// UTF-8 编码（track name 等 meta 文本）。★ 不能把非 ASCII 直接替成 '?'：
// 中文轨道名经 MIDI 往返会丢失（实测「主旋律」→「???」）。
function utf8Bytes(s) {
  var out = [];
  for (var i = 0; i < s.length; i++) {
    var c = s.charCodeAt(i);
    if (c < 0x80) out.push(c);
    else if (c < 0x800) out.push(0xc0 | (c >> 6), 0x80 | (c & 0x3f));
    else if (c >= 0xd800 && c <= 0xdbff && i + 1 < s.length) {
      var c2 = s.charCodeAt(i + 1);
      if (c2 >= 0xdc00 && c2 <= 0xdfff) {
        var cp = 0x10000 + ((c - 0xd800) << 10) + (c2 - 0xdc00);
        out.push(0xf0 | (cp >> 18), 0x80 | ((cp >> 12) & 0x3f), 0x80 | ((cp >> 6) & 0x3f), 0x80 | (cp & 0x3f));
        i++;
        continue;
      }
      out.push(63);
    } else if (c >= 0xdc00 && c <= 0xdfff) out.push(63);
    else out.push(0xe0 | (c >> 12), 0x80 | ((c >> 6) & 0x3f), 0x80 | (c & 0x3f));
  }
  return out;
}

// UTF-8 解码（非法序列回退 Latin-1，保证不抛错、不吞字节）
function utf8Decode(bytes) {
  var s = '';
  var i = 0;
  while (i < bytes.length) {
    var b = bytes[i++];
    if (b < 0x80) { s += String.fromCharCode(b); continue; }
    var n = 0;
    var cp = 0;
    if ((b & 0xe0) === 0xc0) { n = 1; cp = b & 0x1f; }
    else if ((b & 0xf0) === 0xe0) { n = 2; cp = b & 0x0f; }
    else if ((b & 0xf8) === 0xf0) { n = 3; cp = b & 0x07; }
    else { s += String.fromCharCode(b); continue; }
    if (i + n > bytes.length) { s += String.fromCharCode(b); break; }
    var ok = true;
    for (var k = 0; k < n; k++) {
      var bb = bytes[i + k];
      if ((bb & 0xc0) !== 0x80) { ok = false; break; }
      cp = (cp << 6) | (bb & 0x3f);
    }
    if (!ok) { s += String.fromCharCode(b); continue; }
    i += n;
    if (cp > 0xffff) {
      cp -= 0x10000;
      s += String.fromCharCode(0xd800 + (cp >> 10), 0xdc00 + (cp & 0x3ff));
    } else s += String.fromCharCode(cp);
  }
  return s;
}

function metaEvent(delta, type, data) {
  return vlqBytes(delta).concat([0xff, type, data.length]).concat(data);
}

function tempoToMeta(bpm) {
  var usPerQuarter = Math.round(60000000 / bpm);
  return [(usPerQuarter >> 16) & 0xff, (usPerQuarter >> 8) & 0xff, usPerQuarter & 0xff];
}

function keySigMeta(key) {
  // -7..7 升降号个数 + 大调/小调标记
  var sharpsMajor = { 0: 0, 7: 1, 2: 2, 9: 3, 4: 4, 11: 5, 6: 6, 1: 7 };
  var flatsMajor = { 5: 1, 10: 2, 3: 3, 8: 4, 1: 5, 6: 6, 11: 7 };
  var sf = 0;
  if (sharpsMajor[key.root] !== undefined) sf = sharpsMajor[key.root];
  else if (flatsMajor[key.root] !== undefined) sf = -flatsMajor[key.root];
  return [(sf & 0xff), key.minor ? 1 : 0];
}

// 工程 → SMF 字节数组（format 1）
function projectToSmf(proj) {
  var ppq = proj.ppq || PPQ_DEFAULT;
  var tracks = proj.tracks || [];
  var chunks = [];

  // 轨道 0：tempo / 拍号 / 调号
  var meta = [];
  meta = meta.concat(metaEvent(0, META_TRACK_NAME, utf8Bytes(proj.meta && proj.meta.title ? proj.meta.title : 'PairCode Music')));
  meta = meta.concat(metaEvent(0, META_TEMPO, tempoToMeta(proj.tempo || 120)));
  var meter = proj.meter || [4, 4];
  meta = meta.concat(metaEvent(0, META_TIME_SIG, [meter[0], Math.round(Math.log(meter[1]) / Math.log(2)), 24, 8]));
  var key = parseKey(proj.key || 'C');
  meta = meta.concat(metaEvent(0, META_KEY_SIG, keySigMeta(key)));
  meta = meta.concat(metaEvent(0, META_END, []));
  chunks.push(meta);

  // 音符轨
  for (var ti = 0; ti < tracks.length; ti++) {
    var tr = tracks[ti];
    var ch = clamp(isInt(tr.channel) ? tr.channel : ti, 0, 15);
    var evts = [];
    var name = tr.name || tr.id;
    evts = evts.concat(metaEvent(0, META_TRACK_NAME, utf8Bytes(String(name))));
    var program = isInt(tr.program) ? tr.program : 0;
    evts = evts.concat(vlqBytes(0).concat([0xc0 | ch, clamp(program, 0, 127)]));

    // 事件表：同 tick 时 note-off 优先（避免同音粘连）
    var raw = [];
    var notes = tr.notes || [];
    for (var ni = 0; ni < notes.length; ni++) {
      var n = notes[ni];
      raw.push({ tick: n.start, order: 1, kind: 'on', pitch: n.pitch, vel: n.vel === undefined ? 90 : n.vel });
      raw.push({ tick: n.start + n.dur, order: 0, kind: 'off', pitch: n.pitch, vel: 0 });
    }
    raw.sort(function (a, b) {
      if (a.tick !== b.tick) return a.tick - b.tick;
      if (a.order !== b.order) return a.order - b.order;
      return a.pitch - b.pitch;
    });
    var last = 0;
    for (var ri = 0; ri < raw.length; ri++) {
      var e = raw[ri];
      var delta = e.tick - last;
      last = e.tick;
      if (e.kind === 'on') evts = evts.concat(vlqBytes(delta).concat([0x90 | ch, clamp(e.pitch, 0, 127), clamp(e.vel, 1, 127)]));
      else evts = evts.concat(vlqBytes(delta).concat([0x80 | ch, clamp(e.pitch, 0, 127), 0]));
    }
    evts = evts.concat(metaEvent(0, META_END, []));
    chunks.push(evts);
  }

  // 组装文件
  var out = asciiBytes('MThd').concat(be32(6), be16(1), be16(chunks.length), be16(ppq));
  for (var ci = 0; ci < chunks.length; ci++) {
    out = out.concat(asciiBytes('MTrk'), be32(chunks[ci].length)).concat(chunks[ci]);
  }
  return out;
}

// SMF 字节数组 → 工程片段 { tempo, meter, key?, tracks:[{name, notes}] }
function smfToProject(bytes) {
  var pos = 0;
  function need(n) { if (pos + n > bytes.length) throw new Error('SMF 读取越界（偏移 ' + pos + '）'); }
  function readStr(n) { need(n); var s = ''; for (var i = 0; i < n; i++) s += String.fromCharCode(bytes[pos++]); return s; }
  function read32() { need(4); var v = ((bytes[pos] << 24) | (bytes[pos + 1] << 16) | (bytes[pos + 2] << 8) | bytes[pos + 3]) >>> 0; pos += 4; return v; }
  function read16() { need(2); var v = (bytes[pos] << 8) | bytes[pos + 1]; pos += 2; return v; }

  if (readStr(4) !== 'MThd') throw new Error('不是 SMF 文件：缺少 MThd');
  var hlen = read32();
  if (hlen < 6) throw new Error('MThd 长度异常: ' + hlen);
  var format = read16();
  var ntrks = read16();
  var division = read16();
  pos += hlen - 6;
  if (division & 0x8000) throw new Error('暂不支持 SMPTE 时间基（division=0x' + division.toString(16) + '）');

  var tracks = [];
  var tempo = null;
  var meter = null;
  for (var t = 0; t < ntrks; t++) {
    var id = readStr(4);
    if (id !== 'MTrk') throw new Error('轨道 ' + t + ' 缺少 MTrk（读到 ' + id + '）');
    var len = read32();
    var end = pos + len;
    var name = '';
    var notes = [];
    var pending = {};
    var tick = 0;
    var running = 0;
    var program = 0;
    var channel = -1;
    while (pos < end) {
      var vr = readVlq(bytes, pos);
      tick += vr.value;
      pos = vr.pos;
      need(1);
      var status = bytes[pos];
      if (status & 0x80) { pos++; running = status; } else { status = running; }
      if (status === 0xff) {
        need(1);
        var type = bytes[pos++];
        var lr = readVlq(bytes, pos);
        pos = lr.pos;
        var dlen = lr.value;
        need(dlen);
        var data = [];
        for (var i = 0; i < dlen; i++) data.push(bytes[pos + i]);
        pos += dlen;
        if (type === META_TRACK_NAME) {
          name = utf8Decode(data);
        } else if (type === META_TEMPO && dlen >= 3) {
          var us = (data[0] << 16) | (data[1] << 8) | data[2];
          tempo = Math.round(60000000 / us);
        } else if (type === META_TIME_SIG && dlen >= 2) {
          meter = [data[0], Math.round(Math.pow(2, data[1]))];
        }
      } else if (status === 0xf0 || status === 0xf7) {
        var slr = readVlq(bytes, pos);
        pos = slr.pos;
        pos += slr.value;
      } else {
        var hi = status & 0xf0;
        if (channel < 0) channel = status & 0x0f;
        if (hi === 0x90 || hi === 0x80) {
          need(2);
          var pitch = bytes[pos++];
          var vel = bytes[pos++];
          if (hi === 0x90 && vel > 0) {
            if (pending[pitch] === undefined) pending[pitch] = [];
            pending[pitch].push({ tick: tick, vel: vel });
          } else {
            var st = pending[pitch];
            var onset = st && st.length ? st.shift() : null;
            if (onset) notes.push({ start: onset.tick, dur: tick - onset.tick, pitch: pitch, vel: onset.vel });
          }
        } else if (hi === 0xc0) {
          // program change：必须读取，否则 MIDI 往返后音色丢失（实测 32 → 0）
          need(1);
          program = bytes[pos++];
          channel = status & 0x0f;
        } else if (hi === 0xd0) { need(1); pos += 1; }
        else { need(2); pos += 2; }
      }
    }
    if (pos !== end) pos = end;
    tracks.push({ name: name, notes: notes, program: program, channel: channel < 0 ? 0 : channel });
  }

  // 主 tempo 轨（无音符）与有音符轨分开：返回全部，交给调用方组装
  return { format: format, ppq: division, tempo: tempo, meter: meter, tracks: tracks };
}

// ── ABC 记谱（生成 / 解析子集）──────────────────────────────
// ABC 音名（含八度记号，不含时值）：大写 = 第 4 八度（C4..B4），小写高八度，逗号降/撇号升
function abcPitchToken(pitch) {
  var oct = Math.floor(pitch / 12) - 1; // C4=60 → oct 4
  var pc = ((pitch % 12) + 12) % 12;
  var letter = ['C', '^C', 'D', '^D', 'E', 'F', '^F', 'G', '^G', 'A', '^A', 'B'][pc];
  var s = letter;
  if (oct > 4) s += new Array(oct - 3).join("'");
  else if (oct < 4) s += new Array(5 - oct).join(',');
  return s;
}

// ABC 时值后缀（相对 L 单位）：1 → ''；整数 → '3'；2 的幂分数 → '/2'；其余分数 → '3/2'
// ★ 必须覆盖分数形式，否则 V6（ABC 往返）在非 L 整数倍的时值上必然失败。
function abcLenSuffix(ticks, unit) {
  var mul = ticks / unit;
  if (Math.abs(mul - 1) < 1e-9) return '';
  if (isInt(mul)) return String(mul);
  for (var q = 2; q <= 8; q++) {
    var p = mul * q;
    if (Math.abs(p - Math.round(p)) < 1e-9) {
      p = Math.round(p);
      return p === 1 ? ('/' + q) : (p + '/' + q);
    }
  }
  return ''; // 无法精确表示（tick 与 L 单位不成简单比例）——由 V6 暴露
}

function projectToAbc(proj) {
  var ppq = proj.ppq || PPQ_DEFAULT;
  var unit = ppq / 2; // L:1/8
  var meter = proj.meter || [4, 4];
  var lines = [];
  lines.push('X:1');
  lines.push('T:' + ((proj.meta && proj.meta.title) ? proj.meta.title : 'PairCode Music'));
  lines.push('M:' + meter[0] + '/' + meter[1]);
  lines.push('L:1/8');
  lines.push('Q:1/4=' + (proj.tempo || 120));
  lines.push('K:' + (proj.key || 'C'));
  var tracks = proj.tracks || [];
  for (var i = 0; i < tracks.length; i++) {
    var tr = tracks[i];
    lines.push('V:' + (i + 1) + ' name="' + String(tr.name || tr.id) + '"');
    var notes = (tr.notes || []).slice().sort(function (a, b) { return a.start - b.start || a.pitch - b.pitch; });
    var out = '';
    var cursor = 0;
    var gi = 0;
    while (gi < notes.length) {
      var start = notes[gi].start;
      var group = [];
      while (gi < notes.length && notes[gi].start === start) { group.push(notes[gi]); gi++; }
      if (start > cursor) out += 'z' + abcLenSuffix(start - cursor, unit) + ' ';
      // 同起点且同时值的多音 → ABC 和弦 [CEG]；否则逐个（ABC 无法表示不等时值的重叠音）
      var sameDur = true;
      for (var d = 1; d < group.length; d++) if (group[d].dur !== group[0].dur) sameDur = false;
      if (group.length > 1 && sameDur) {
        var inner = '';
        for (var ci = 0; ci < group.length; ci++) inner += abcPitchToken(group[ci].pitch);
        out += '[' + inner + ']' + abcLenSuffix(group[0].dur, unit) + ' ';
        cursor = start + group[0].dur;
      } else {
        for (var ei = 0; ei < group.length; ei++) {
          out += abcPitchToken(group[ei].pitch) + abcLenSuffix(group[ei].dur, unit) + ' ';
          cursor = group[ei].start + group[ei].dur;
        }
      }
    }
    lines.push(out.trim());
  }
  return lines.join('\n') + '\n';
}

// ABC 子集解析：X/T/M/L/Q/K 头 + 音符（A-Ga-g + ^_= + 八度撇逗 + 时值数字/斜杠）
function abcToProject(text, ppq) {
  var lines = String(text).split(/\r?\n/);
  var head = {};
  var body = [];
  for (var i = 0; i < lines.length; i++) {
    var ln = lines[i];
    var hm = /^([A-Za-z]):\s*(.*)$/.exec(ln);
    // ★ V: 是「声部标记」而非头部字段：必须留给 body 分支处理（用于重置时间轴）。
    //   若在此当作头部吞掉，多轨 ABC 的第二声部会被累加到第一声部末尾
    //   （实测 V6 差异：t2 音符整体偏移 3840 tick）。
    if (hm && 'XTLMQK'.indexOf(hm[1].toUpperCase()) >= 0) { head[hm[1].toUpperCase()] = hm[2].trim(); continue; }
    if (/^%/.test(ln)) continue;
    body.push(ln);
  }
  var meter = [4, 4];
  if (head.M) { var mm = /^(\d+)\/(\d+)$/.exec(head.M); if (mm) meter = [parseInt(mm[1], 10), parseInt(mm[2], 10)]; }
  var tempo = 120;
  if (head.Q) { var qm = /=\s*(\d+)/.exec(head.Q); if (qm) tempo = parseInt(qm[1], 10); }
  var unitLen = ppq / 2; // L:1/8 默认
  if (head.L) {
    var lm = /^(\d+)\/(\d+)$/.exec(head.L);
    if (lm) unitLen = Math.round(ppq * 4 * parseInt(lm[1], 10) / parseInt(lm[2], 10));
  }
  var notes = [];
  var cursor = 0;
  for (var b = 0; b < body.length; b++) {
    var line = body[b];
    // 多声部：每段（V:）时间轴独立从 0 起算 —— 不重置会让后续声部整体偏移，
    // 直接导致 V6（ABC 往返音符集合一致）在多轨工程上失败。
    if (/^\s*V:/.test(line)) { cursor = 0; continue; }
    if (/^\s*%%/.test(line)) continue;
    var tokens = line.split(/\s+/);
    for (var t = 0; t < tokens.length; t++) {
      var tk = tokens[t];
      if (!tk) continue;
      // 和弦 [CEG]（内部只含音名，时值在括号外）
      var cm = /^\[([^\]]+)\](\d*\/?\d*)?$/.exec(tk);
      if (cm) {
        var cDur = cm[2] ? lenOfAbcDuration(cm[2], unitLen) : unitLen;
        var innerRe = /(\^|_|=)?([A-Ga-g])(,*)('*)/g;
        var im;
        while ((im = innerRe.exec(cm[1])) !== null) {
          var iSemi = LETTER_SEMI[im[2].toUpperCase()];
          if (im[1] === '^') iSemi += 1;
          else if (im[1] === '_') iSemi -= 1;
          var iOct = 0;
          if (im[2] === im[2].toLowerCase()) iOct = 1;
          iOct -= im[3].length;
          iOct += im[4].length;
          notes.push({ start: cursor, dur: cDur, pitch: (4 + iOct + 1) * 12 + iSemi, vel: 90 });
        }
        cursor += cDur;
        continue;
      }
      var rm = /^z(\d*\/?\d*)$/.exec(tk);
      if (rm) { cursor += restLen(rm[1], unitLen); continue; }
      var nm = /^(\^|_|=)?([A-Ga-g])(,*)('*)(\d*\/?\d*)?$/.exec(tk);
      if (!nm) continue;
      var letterSemi = LETTER_SEMI[nm[2].toUpperCase()];
      if (nm[1] === '^') letterSemi += 1;
      else if (nm[1] === '_') letterSemi -= 1;
      var octShift = 0;
      if (nm[2] === nm[2].toLowerCase()) octShift = 1;       // 小写 = 第 5 八度
      octShift -= nm[3].length;                              // 逗号降八度
      octShift += nm[4].length;                              // 撇号升八度
      var pitch = (4 + octShift + 1) * 12 + letterSemi;      // 基准 C4 = 60
      var dur = unitLen;
      if (nm[5]) dur = lenOfAbcDuration(nm[5], unitLen);
      notes.push({ start: cursor, dur: dur, pitch: pitch, vel: 90 });
      cursor += dur;
    }
  }
  return {
    format: 'paircode.music.project/1',
    meta: { title: head.T || 'ABC Import', createdAt: nowIso(), updatedAt: nowIso() },
    tempo: tempo, meter: meter, key: head.K || 'C', ppq: ppq,
    tracks: [{ id: 't1', name: head.T || 'Track 1', channel: 0, program: 0, notes: notes }],
    artifacts: {},
  };
}

function restLen(spec, unitLen) {
  if (!spec) return unitLen;
  return lenOfAbcDuration(spec, unitLen);
}

function lenOfAbcDuration(spec, unitLen) {
  if (!spec) return unitLen;
  var num = /^(\d+)$/.exec(spec);
  if (num) return unitLen * parseInt(num[1], 10);
  var frac = /^(\d*)\/(\d*)$/.exec(spec);
  if (frac) {
    var a = frac[1] === '' ? 1 : parseInt(frac[1], 10);
    var b = frac[2] === '' ? 2 : parseInt(frac[2], 10);
    return Math.round(unitLen * a / b);
  }
  return unitLen;
}

// ── MusicXML（生成 / 简化解析）───────────────────────────────
function stepAlterOf(pitch) {
  var pc = ((pitch % 12) + 12) % 12;
  var table = [
    { step: 'C', alter: 0 }, { step: 'C', alter: 1 }, { step: 'D', alter: 0 }, { step: 'D', alter: 1 },
    { step: 'E', alter: 0 }, { step: 'F', alter: 0 }, { step: 'F', alter: 1 }, { step: 'G', alter: 0 },
    { step: 'G', alter: 1 }, { step: 'A', alter: 0 }, { step: 'A', alter: 1 }, { step: 'B', alter: 0 },
  ];
  return table[pc];
}

function xmlEscape(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function projectToMusicXml(proj) {
  var ppq = proj.ppq || PPQ_DEFAULT;
  var meter = proj.meter || [4, 4];
  var divisions = ppq;
  var out = [];
  out.push('<?xml version="1.0" encoding="UTF-8"?>');
  out.push('<!DOCTYPE score-partwise PUBLIC "-//Recordare//DTD MusicXML 3.1 Partwise//EN" "http://www.musicxml.org/dtds/partwise.dtd">');
  out.push('<score-partwise version="3.1">');
  out.push('  <work><work-title>' + xmlEscape((proj.meta && proj.meta.title) || 'PairCode Music') + '</work-title></work>');
  out.push('  <part-list>');
  var tracks = proj.tracks || [];
  for (var i = 0; i < tracks.length; i++) {
    out.push('    <score-part id="P' + (i + 1) + '"><part-name>' + xmlEscape(tracks[i].name || tracks[i].id) + '</part-name></score-part>');
  }
  out.push('  </part-list>');
  for (var t = 0; t < tracks.length; t++) {
    var tr = tracks[t];
    out.push('  <part id="P' + (t + 1) + '">');
    out.push('    <measure number="1">');
    out.push('      <attributes>');
    out.push('        <divisions>' + divisions + '</divisions>');
    out.push('        <key><fifths>' + keyFifths(parseKey(proj.key || 'C')) + '</fifths></key>');
    out.push('        <time><beats>' + meter[0] + '</beats><beat-type>' + meter[1] + '</beat-type></time>');
    out.push('        <clef><sign>G</sign><line>2</line></clef>');
    out.push('      </attributes>');
    out.push('      <direction><direction-type><metronome><beat-unit>quarter</beat-unit><per-minute>' + (proj.tempo || 120) + '</per-minute></metronome></direction-type></direction>');
    var notes = (tr.notes || []).slice().sort(function (a, b) { return a.start - b.start; });
    var cursor = 0;
    var measureTicks = Math.round(ppq * 4 * meter[0] / meter[1]);
    var measureNo = 1;
    for (var n = 0; n < notes.length; n++) {
      var nt = notes[n];
      while (nt.start - cursor >= measureTicks) {
        out.push('    </measure>');
        measureNo++;
        out.push('    <measure number="' + measureNo + '">');
        cursor += measureTicks;
      }
      if (nt.start > cursor) {
        out.push(xmlRest(nt.start - cursor));
        cursor = nt.start;
      }
      out.push(xmlNote(nt));
      cursor = nt.start + nt.dur;
    }
    out.push('    </measure>');
    out.push('  </part>');
  }
  out.push('</score-partwise>');
  return out.join('\n') + '\n';
}

function keyFifths(key) {
  var sharps = { 0: 0, 7: 1, 2: 2, 9: 3, 4: 4, 11: 5, 6: 6, 1: 7 };
  var flats = { 5: 1, 10: 2, 3: 3, 8: 4, 1: 5, 6: 6, 11: 7 };
  if (sharps[key.root] !== undefined) return sharps[key.root];
  if (flats[key.root] !== undefined) return -flats[key.root];
  return 0;
}

function xmlTypeOf(dur, ppq) {
  var q = dur / ppq; // 以四分音符为 1
  var map = { 0.25: '16th', 0.5: 'eighth', 1: 'quarter', 2: 'half', 4: 'whole' };
  if (map[q]) return map[q];
  if (q < 1) return 'eighth';
  if (q < 2) return 'quarter';
  if (q < 4) return 'half';
  return 'whole';
}

function xmlNote(nt) {
  var sa = stepAlterOf(nt.pitch);
  var oct = Math.floor(nt.pitch / 12) - 1;
  var s = '      <note>';
  s += '<pitch><step>' + sa.step + '</step>' + (sa.alter ? '<alter>' + sa.alter + '</alter>' : '') + '<octave>' + oct + '</octave></pitch>';
  s += '<duration>' + nt.dur + '</duration>';
  s += '<voice>1</voice><type>' + xmlTypeOf(nt.dur, 480) + '</type>';
  if (nt.vel !== undefined && nt.vel !== 90) s += '<notations><dynamics><other-dynamics>' + nt.vel + '</other-dynamics></dynamics></notations>';
  s += '</note>';
  return s;
}

function xmlRest(dur) {
  return '      <note><rest/><duration>' + dur + '</duration><voice>1</voice><type>quarter</type></note>';
}

// MusicXML 简化解析：按 part 顺序，声明顺序即音符顺序，忽略 <backup>/<forward> 的高级用法
function musicXmlToProject(text, ppq) {
  var parts = String(text).split(/<part\s+id="/);
  var tracks = [];
  var tempo = null;
  var meter = null;
  var tm = /<per-minute>\s*(\d+)\s*<\/per-minute>/.exec(text);
  if (tm) tempo = parseInt(tm[1], 10);
  var mm = /<time>\s*<beats>\s*(\d+)\s*<\/beats>\s*<beat-type>\s*(\d+)\s*<\/beat-type>/.exec(text);
  if (mm) meter = [parseInt(mm[1], 10), parseInt(mm[2], 10)];
  var divisions = 1;
  var dm = /<divisions>\s*(\d+)\s*<\/divisions>/.exec(text);
  if (dm) divisions = parseInt(dm[1], 10);

  for (var p = 1; p < parts.length; p++) {
    var chunk = parts[p];
    var nameM = /<part-name>([^<]*)<\/part-name>/.exec(parts[p - 1]) || /<score-part id="[^"]*">\s*<part-name>([^<]*)<\/part-name>/.exec(text);
    var trackName = nameM ? nameM[1] : ('Part ' + p);
    var notes = [];
    var cursor = 0;
    var re = /<note\b[^>]*>([\s\S]*?)<\/note>/g;
    var m;
    while ((m = re.exec(chunk)) !== null) {
      var inner = m[1];
      if (/<chord\s*\/>/.test(inner)) {
        // 和弦音：与上一个音同起点
        var prevStart = notes.length ? notes[notes.length - 1].start : cursor;
        cursor = prevStart;
      }
      var durM = /<duration>\s*(\d+)\s*<\/duration>/.exec(inner);
      var dur = durM ? Math.round(parseInt(durM[1], 10) * ppq / divisions) : Math.round(ppq / 2);
      if (/<rest\s*\/>/.test(inner)) { cursor += dur; continue; }
      var stepM = /<step>([A-G])<\/step>/.exec(inner);
      var octM = /<octave>(-?\d+)<\/octave>/.exec(inner);
      var altM = /<alter>(-?\d+)<\/alter>/.exec(inner);
      if (!stepM || !octM) { cursor += dur; continue; }
      var semi = LETTER_SEMI[stepM[1]] + (altM ? parseInt(altM[1], 10) : 0);
      var pitch = (parseInt(octM[1], 10) + 1) * 12 + semi;
      notes.push({ start: cursor, dur: dur, pitch: pitch, vel: 90 });
      cursor += dur;
    }
    if (notes.length || p === 1) tracks.push({ id: 't' + p, name: trackName, channel: clamp(p - 1, 0, 15), program: 0, notes: notes });
  }
  return {
    format: 'paircode.music.project/1',
    meta: { title: (function () { var w = /<work-title>([^<]*)<\/work-title>/.exec(text); return w ? w[1] : 'MusicXML Import'; })(), createdAt: nowIso(), updatedAt: nowIso() },
    tempo: tempo || 120, meter: meter || [4, 4], key: 'C', ppq: ppq,
    tracks: tracks.length ? tracks : [{ id: 't1', name: 'Part 1', channel: 0, program: 0, notes: [] }],
    artifacts: {},
  };
}

// ── SVG 乐谱（简易五线谱，供 LLM「看图」核对）────────────────
function projectToSvg(proj) {
  var ppq = proj.ppq || PPQ_DEFAULT;
  var meter = proj.meter || [4, 4];
  var tracks = proj.tracks || [];
  var lineGap = 10;
  var staffH = lineGap * 4;
  var staffGap = 70;
  var left = 70;
  var pxPerQuarter = 48;
  var measureTicks = Math.round(ppq * 4 * meter[0] / meter[1]);
  var totalTicks = 0;
  for (var i = 0; i < tracks.length; i++) {
    var ns = tracks[i].notes || [];
    for (var j = 0; j < ns.length; j++) totalTicks = Math.max(totalTicks, ns[j].start + ns[j].dur);
  }
  if (totalTicks <= 0) totalTicks = measureTicks;
  var width = left + Math.ceil(totalTicks / ppq * pxPerQuarter) + 60;
  var height = 40 + tracks.length * (staffH + staffGap) + 40;
  var s = [];
  s.push('<svg xmlns="http://www.w3.org/2000/svg" width="' + width + '" height="' + height + '" viewBox="0 0 ' + width + ' ' + height + '">');
  s.push('<rect width="100%" height="100%" fill="#fbfbf8"/>');
  s.push('<text x="12" y="22" font-family="sans-serif" font-size="14" fill="#222">' + xmlEscape((proj.meta && proj.meta.title) || 'PairCode Music') + '　♩=' + (proj.tempo || 120) + '　' + meter[0] + '/' + meter[1] + '　' + (proj.key || 'C') + '</text>');

  for (var t = 0; t < tracks.length; t++) {
    var top = 50 + t * (staffH + staffGap);
    // 五线
    for (var li = 0; li < 5; li++) {
      var y = top + li * lineGap;
      s.push('<line x1="' + left + '" y1="' + y + '" x2="' + (width - 30) + '" y2="' + y + '" stroke="#555" stroke-width="1"/>');
    }
    s.push('<text x="12" y="' + (top + staffH - 2) + '" font-family="sans-serif" font-size="11" fill="#333">' + xmlEscape(tracks[t].name || tracks[t].id) + '</text>');
    // 谱号（简化：高音 G）
    s.push('<text x="' + (left - 26) + '" y="' + (top + staffH - 2) + '" font-family="serif" font-size="34" fill="#222">𝄞</text>');
    // 小节线
    var measures = Math.max(1, Math.ceil(totalTicks / measureTicks));
    for (var mi = 1; mi <= measures; mi++) {
      var mx = left + mi * measureTicks / ppq * pxPerQuarter;
      s.push('<line x1="' + mx.toFixed(1) + '" y1="' + top + '" x2="' + mx.toFixed(1) + '" y2="' + (top + staffH) + '" stroke="#555" stroke-width="1"/>');
    }
    // 音符（E4=64 为最下线）
    var notes = (tracks[t].notes || []).slice().sort(function (a, b) { return a.start - b.start; });
    for (var n = 0; n < notes.length; n++) {
      var nt = notes[n];
      var x = left + nt.start / ppq * pxPerQuarter + 6;
      var yN = (top + staffH) - (nt.pitch - 64) * (lineGap / 2);
      var filled = nt.dur <= ppq;
      s.push('<ellipse cx="' + x.toFixed(1) + '" cy="' + yN.toFixed(1) + '" rx="4.6" ry="3.4" fill="' + (filled ? '#111' : '#fbfbf8') + '" stroke="#111" stroke-width="1.2"/>');
      // 符干
      var stemUp = nt.pitch < 71;
      var sy1 = yN, sy2 = stemUp ? yN - 30 : yN + 30;
      s.push('<line x1="' + (stemUp ? (x + 4.4) : (x - 4.4)).toFixed(1) + '" y1="' + sy1.toFixed(1) + '" x2="' + (stemUp ? (x + 4.4) : (x - 4.4)).toFixed(1) + '" y2="' + sy2.toFixed(1) + '" stroke="#111" stroke-width="1.2"/>');
      // 二分音符/全音符：空心/无符干已在上面处理
      if (nt.dur >= ppq * 4) { /* 全音符：无符干（简化省略） */ }
      // 加线（超出五线范围）
      if (nt.pitch > 77 || nt.pitch < 64) {
        var edgeY = nt.pitch > 77 ? top : top + staffH;
        if (yN < top || yN > top + staffH) {
          s.push('<line x1="' + (x - 9).toFixed(1) + '" y1="' + yN.toFixed(1) + '" x2="' + (x + 9).toFixed(1) + '" y2="' + yN.toFixed(1) + '" stroke="#111" stroke-width="1.2"/>');
        }
      }
    }
  }
  s.push('</svg>');
  return s.join('\n') + '\n';
}

// ── 工程读写 ───────────────────────────────────────────────
function emptyProject(title) {
  return {
    format: 'paircode.music.project/1',
    meta: { title: title || '未命名乐曲', createdAt: nowIso(), updatedAt: nowIso() },
    tempo: 120,
    meter: [4, 4],
    key: 'C',
    ppq: PPQ_DEFAULT,
    tracks: [],
    artifacts: {},
  };
}

function normalizeProject(proj) {
  if (!proj || typeof proj !== 'object') throw new Error('工程内容不是对象');
  if (!proj.format) proj.format = 'paircode.music.project/1';
  if (!proj.meta) proj.meta = { title: '未命名乐曲' };
  if (!isInt(proj.tempo) || proj.tempo < 20 || proj.tempo > 400) proj.tempo = 120;
  if (!Array.isArray(proj.meter) || proj.meter.length !== 2) proj.meter = [4, 4];
  if (!proj.key) proj.key = 'C';
  if (!isInt(proj.ppq) || proj.ppq <= 0) proj.ppq = PPQ_DEFAULT;
  if (!Array.isArray(proj.tracks)) proj.tracks = [];
  if (!proj.artifacts || typeof proj.artifacts !== 'object') proj.artifacts = {};
  for (var i = 0; i < proj.tracks.length; i++) {
    var tr = proj.tracks[i];
    if (!tr.id) tr.id = 't' + (i + 1);
    if (!Array.isArray(tr.notes)) tr.notes = [];
    if (!isInt(tr.channel)) tr.channel = clamp(i, 0, 15);
    if (!isInt(tr.program)) tr.program = 0;
  }
  return proj;
}

function loadProject(ctx, path) {
  if (!ctx.fs.exists(path)) throw new Error('工程不存在: ' + path + '（先用 music_project mode=create 创建）');
  var txt;
  try { txt = ctx.fs.readFile(path); } catch (e) { throw new Error('读取工程失败: ' + e.message); }
  var proj;
  try { proj = JSON.parse(txt); } catch (e) { throw new Error('工程 JSON 解析失败: ' + e.message); }
  return normalizeProject(proj);
}

function saveProject(ctx, path, proj) {
  proj.meta = proj.meta || {};
  proj.meta.updatedAt = nowIso();
  var json = JSON.stringify(proj, null, 2);
  ctx.fs.writeFile(path, json);
  return json.length;
}

function projectSummary(proj, path) {
  var ppq = proj.ppq || PPQ_DEFAULT;
  var lines = [];
  lines.push('工程: ' + path);
  lines.push('标题: ' + ((proj.meta && proj.meta.title) || '(无)') + '　速度: ' + proj.tempo + '　拍号: ' + proj.meter[0] + '/' + proj.meter[1] + '　调号: ' + proj.key + '　ppq: ' + ppq);
  var totalNotes = 0;
  var totalTicks = 0;
  lines.push('轨道数: ' + proj.tracks.length);
  for (var i = 0; i < proj.tracks.length; i++) {
    var tr = proj.tracks[i];
    var notes = tr.notes || [];
    totalNotes += notes.length;
    var lo = 999, hi = -1, end = 0;
    for (var j = 0; j < notes.length; j++) {
      lo = Math.min(lo, notes[j].pitch);
      hi = Math.max(hi, notes[j].pitch);
      end = Math.max(end, notes[j].start + notes[j].dur);
    }
    totalTicks = Math.max(totalTicks, end);
    var range = notes.length ? (pitchName(lo) + '..' + pitchName(hi)) : '(空)';
    lines.push('  - ' + tr.id + ' 「' + (tr.name || '') + '」 通道 ' + tr.channel + ' 音色 ' + tr.program +
      ' 音符 ' + notes.length + ' 音域 ' + range + ' 末 tick ' + end);
  }
  var secs = totalTicks / ppq * (60 / proj.tempo);
  lines.push('合计: 音符 ' + totalNotes + '　时长 ' + secs.toFixed(2) + 's');
  var arts = Object.keys(proj.artifacts || {});
  if (arts.length) {
    lines.push('产物:');
    for (var a = 0; a < arts.length; a++) lines.push('  - ' + arts[a] + ' → ' + proj.artifacts[arts[a]]);
  }
  return lines.join('\n');
}

// ── 选择器与编辑操作 ───────────────────────────────────────
function selectNotes(track, filter) {
  var notes = track.notes || [];
  if (!filter) return notes.map(function (_, i) { return i; });
  var idx = [];
  for (var i = 0; i < notes.length; i++) {
    var n = notes[i];
    if (isInt(filter.index) && i !== filter.index) continue;
    if (isInt(filter.pitch) && n.pitch !== filter.pitch) continue;
    if (isInt(filter.pitchMin) && n.pitch < filter.pitchMin) continue;
    if (isInt(filter.pitchMax) && n.pitch > filter.pitchMax) continue;
    if (isInt(filter.startFrom) && n.start < filter.startFrom) continue;
    if (isInt(filter.startTo) && n.start > filter.startTo) continue;
    if (filter.name !== undefined && (track.name || '') !== filter.name) continue;
    idx.push(i);
  }
  return idx;
}

function findTrack(proj, idOrIndex) {
  if (idOrIndex === undefined || idOrIndex === null || idOrIndex === '') {
    if (!proj.tracks.length) throw new Error('工程没有任何轨道（先加 track.add）');
    return proj.tracks[0];
  }
  for (var i = 0; i < proj.tracks.length; i++) if (proj.tracks[i].id === idOrIndex) return proj.tracks[i];
  if (isInt(idOrIndex)) {
    for (var j = 0; j < proj.tracks.length; j++) if (j === idOrIndex) return proj.tracks[j];
  }
  throw new Error('找不到轨道: ' + idOrIndex + '（现有: ' + proj.tracks.map(function (t) { return t.id; }).join(', ') + '）');
}

function sortNotes(track) {
  track.notes.sort(function (a, b) { return a.start - b.start || a.pitch - b.pitch; });
}

function applyOps(proj, ops) {
  var log = [];
  for (var i = 0; i < ops.length; i++) {
    var op = ops[i];
    if (!op || !op.op) throw new Error('第 ' + (i + 1) + ' 条 op 缺少 op 字段');
    var name = op.op;
    if (name === 'set.tempo') {
      if (!isInt(op.bpm) || op.bpm < 20 || op.bpm > 400) throw new Error('set.tempo 需要 bpm ∈ [20,400]');
      proj.tempo = op.bpm;
      log.push('set.tempo → ' + op.bpm);
    } else if (name === 'set.meter') {
      if (!isInt(op.num) || !isInt(op.den)) throw new Error('set.meter 需要 num/den 整数');
      proj.meter = [op.num, op.den];
      log.push('set.meter → ' + op.num + '/' + op.den);
    } else if (name === 'set.key') {
      var k = parseKey(op.key);
      proj.key = keyToString(k);
      log.push('set.key → ' + proj.key);
    } else if (name === 'project.rename') {
      proj.meta.title = op.title || proj.meta.title;
      log.push('project.rename → ' + proj.meta.title);
    } else if (name === 'track.add') {
      var id = op.id || ('t' + (proj.tracks.length + 1));
      for (var x = 0; x < proj.tracks.length; x++) if (proj.tracks[x].id === id) throw new Error('轨道 id 已存在: ' + id);
      proj.tracks.push({
        id: id, name: op.name || id,
        channel: isInt(op.channel) ? clamp(op.channel, 0, 15) : clamp(proj.tracks.length, 0, 15),
        program: isInt(op.program) ? clamp(op.program, 0, 127) : 0,
        notes: [],
      });
      log.push('track.add → ' + id);
    } else if (name === 'track.remove') {
      var before = proj.tracks.length;
      proj.tracks = proj.tracks.filter(function (t) { return t.id !== op.id; });
      if (proj.tracks.length === before) throw new Error('找不到要删除的轨道: ' + op.id);
      log.push('track.remove → ' + op.id);
    } else if (name === 'track.rename') {
      var tr1 = findTrack(proj, op.id);
      tr1.name = op.name || tr1.name;
      log.push('track.rename → ' + tr1.id + ' = ' + tr1.name);
    } else if (name === 'track.instrument') {
      var tr2 = findTrack(proj, op.id);
      if (!isInt(op.program)) throw new Error('track.instrument 需要 program(0..127)');
      tr2.program = clamp(op.program, 0, 127);
      log.push('track.instrument → ' + tr2.id + ' program=' + tr2.program);
    } else if (name === 'note.add') {
      var tr3 = findTrack(proj, op.track);
      checkNote(op);
      tr3.notes.push({ start: op.start, dur: op.dur, pitch: op.pitch, vel: op.vel === undefined ? 90 : op.vel });
      sortNotes(tr3);
      log.push('note.add → ' + tr3.id + ' ' + pitchName(op.pitch) + ' @' + op.start + ' dur ' + op.dur);
    } else if (name === 'chord.add') {
      var tr4 = findTrack(proj, op.track);
      var quality = op.quality || 'major';
      var iv = CHORD_QUALITIES[quality];
      if (!iv) throw new Error('未知和弦性质: ' + quality + '（可用: ' + Object.keys(CHORD_QUALITIES).join(', ') + '）');
      var root = parsePitchName(op.root);
      var dur = isInt(op.dur) ? op.dur : proj.ppq;
      var vel = op.vel === undefined ? 90 : op.vel;
      for (var c = 0; c < iv.length; c++) {
        var p = root + iv[c];
        if (p > 127) throw new Error('和弦超出音域: ' + pitchName(p));
        tr4.notes.push({ start: op.start || 0, dur: dur, pitch: p, vel: vel });
      }
      sortNotes(tr4);
      log.push('chord.add → ' + tr4.id + ' ' + pitchName(root) + ' ' + quality + '（' + iv.length + ' 音）');
    } else if (name === 'scale.add') {
      var tr5 = findTrack(proj, op.track);
      var scaleName = op.scale || 'major';
      var sc = SCALES[scaleName];
      if (!sc) throw new Error('未知音阶: ' + scaleName + '（可用: ' + Object.keys(SCALES).join(', ') + '）');
      var base = parsePitchName(op.root);
      var steps = isInt(op.steps) ? op.steps : sc.length;
      var stepDur = isInt(op.stepDur) ? op.stepDur : Math.round(proj.ppq / 2);
      var dir = op.down ? -1 : 1;
      var cursor = op.start || 0;
      for (var si = 0; si < steps; si++) {
        var oct = Math.floor(si / sc.length);
        var p2 = base + sc[si % sc.length] + 12 * oct * dir;
        if (p2 < 0 || p2 > 127) continue;
        tr5.notes.push({ start: cursor, dur: stepDur, pitch: p2, vel: 90 });
        cursor += stepDur;
      }
      sortNotes(tr5);
      log.push('scale.add → ' + tr5.id + ' ' + pitchName(base) + ' ' + scaleName + ' ' + steps + ' 音');
    } else if (name === 'note.remove') {
      var tr6 = findTrack(proj, op.track);
      var idx6 = selectNotes(tr6, op.filter);
      var removed = idx6.length;
      var keep = [];
      for (var n6 = 0; n6 < tr6.notes.length; n6++) if (idx6.indexOf(n6) < 0) keep.push(tr6.notes[n6]);
      tr6.notes = keep;
      log.push('note.remove → ' + tr6.id + ' 删除 ' + removed + ' 个音符');
    } else if (name === 'note.move') {
      var tr7 = findTrack(proj, op.track);
      var idx7 = selectNotes(tr7, op.filter);
      var ds = isInt(op.deltaStart) ? op.deltaStart : 0;
      var dp = isInt(op.deltaPitch) ? op.deltaPitch : 0;
      for (var n7 = 0; n7 < idx7.length; n7++) {
        var nt7 = tr7.notes[idx7[n7]];
        nt7.start = Math.max(0, nt7.start + ds);
        nt7.pitch = clamp(nt7.pitch + dp, 0, 127);
      }
      sortNotes(tr7);
      log.push('note.move → ' + tr7.id + ' 移动 ' + idx7.length + ' 个（Δtick ' + ds + '，Δpitch ' + dp + '）');
    } else if (name === 'note.transpose') {
      var tr8 = findTrack(proj, op.track);
      if (!isInt(op.semitones)) throw new Error('note.transpose 需要 semitones（半音数，可负）');
      var idx8 = selectNotes(tr8, op.filter);
      for (var n8 = 0; n8 < idx8.length; n8++) {
        var nt8 = tr8.notes[idx8[n8]];
        var np = nt8.pitch + op.semitones;
        if (np < 0 || np > 127) throw new Error('移调越界: ' + pitchName(nt8.pitch) + ' + ' + op.semitones);
        nt8.pitch = np;
      }
      log.push('note.transpose → ' + tr8.id + ' ' + idx8.length + ' 个音符 ' + (op.semitones >= 0 ? '+' : '') + op.semitones + ' 半音');
    } else if (name === 'note.quantize') {
      var tr9 = findTrack(proj, op.track);
      var grid = isInt(op.grid) ? op.grid : Math.round(proj.ppq / 4);
      if (grid <= 0) throw new Error('note.quantize 需要 grid > 0');
      var strength = op.strength === undefined ? 1 : op.strength;
      var idx9 = selectNotes(tr9, op.filter);
      for (var n9 = 0; n9 < idx9.length; n9++) {
        var nt9 = tr9.notes[idx9[n9]];
        var target = Math.round(nt9.start / grid) * grid;
        nt9.start = Math.max(0, Math.round(nt9.start + (target - nt9.start) * strength));
      }
      sortNotes(tr9);
      log.push('note.quantize → ' + tr9.id + ' 网格 ' + grid + ' 吸附 ' + idx9.length + ' 个');
    } else if (name === 'note.velocity') {
      var tr10 = findTrack(proj, op.track);
      var idx10 = selectNotes(tr10, op.filter);
      for (var n10 = 0; n10 < idx10.length; n10++) {
        var nt10 = tr10.notes[idx10[n10]];
        if (isInt(op.set)) nt10.vel = clamp(op.set, 1, 127);
        if (typeof op.scale === 'number') nt10.vel = clamp(Math.round(nt10.vel * op.scale), 1, 127);
        if (isInt(op.delta)) nt10.vel = clamp(nt10.vel + op.delta, 1, 127);
      }
      log.push('note.velocity → ' + tr10.id + ' 调整 ' + idx10.length + ' 个');
    } else if (name === 'note.set') {
      var tr11 = findTrack(proj, op.track);
      var idx11 = selectNotes(tr11, op.filter);
      var fields = op.fields || {};
      for (var n11 = 0; n11 < idx11.length; n11++) {
        var nt11 = tr11.notes[idx11[n11]];
        if (isInt(fields.dur)) nt11.dur = fields.dur;
        if (isInt(fields.vel)) nt11.vel = clamp(fields.vel, 1, 127);
        if (isInt(fields.pitch)) nt11.pitch = clamp(fields.pitch, 0, 127);
      }
      log.push('note.set → ' + tr11.id + ' 修改 ' + idx11.length + ' 个');
    } else if (name === 'track.repeat') {
      var tr12 = findTrack(proj, op.track);
      var times = isInt(op.times) ? op.times : 1;
      if (times < 1) throw new Error('track.repeat 需要 times ≥ 1');
      var span = 0;
      for (var q = 0; q < tr12.notes.length; q++) span = Math.max(span, tr12.notes[q].start + tr12.notes[q].dur);
      var added = [];
      for (var r = 1; r <= times; r++) {
        for (var w = 0; w < tr12.notes.length; w++) {
          var src = tr12.notes[w];
          added.push({ start: src.start + span * r, dur: src.dur, pitch: src.pitch, vel: src.vel });
        }
      }
      tr12.notes = tr12.notes.concat(added);
      sortNotes(tr12);
      log.push('track.repeat → ' + tr12.id + ' ×' + (times + 1) + '（段长 ' + span + ' tick，新增 ' + added.length + ' 音符）');
    } else {
      throw new Error('未知 op: ' + name);
    }
  }
  return log;
}

function checkNote(op) {
  if (!isInt(op.start) || op.start < 0) throw new Error('note.add 需要 start ≥ 0 的整数 tick');
  if (!isInt(op.dur) || op.dur <= 0) throw new Error('note.add 需要 dur > 0 的整数 tick');
  if (!isInt(op.pitch) || op.pitch < 0 || op.pitch > 127) throw new Error('note.add 需要 pitch ∈ [0,127]');
  if (op.vel !== undefined && (!isInt(op.vel) || op.vel < 1 || op.vel > 127)) throw new Error('note.add 的 vel 需 ∈ [1,127]');
}

// ── 校验（六项判据）─────────────────────────────────────────
function verifyProject(ctx, proj, opts) {
  var checks = [];
  function add(id, name, pass, metric) { checks.push({ id: id, name: name, pass: !!pass, metric: metric }); }

  // V1 音符合法（音域 + 整型）
  var badPitch = 0;
  var lo = 999, hi = -1;
  for (var i = 0; i < proj.tracks.length; i++) {
    var notes1 = proj.tracks[i].notes || [];
    for (var j = 0; j < notes1.length; j++) {
      var n1 = notes1[j];
      if (!isInt(n1.pitch) || n1.pitch < 0 || n1.pitch > 127) badPitch++;
      else { lo = Math.min(lo, n1.pitch); hi = Math.max(hi, n1.pitch); }
    }
  }
  add('V1', '音符合法：pitch 整型且 ∈ [0,127]', badPitch === 0,
    badPitch === 0 ? ('全部合法；音域 ' + (hi >= lo ? pitchName(lo) + '..' + pitchName(hi) : '(空)')) : (badPitch + ' 个非法音高'));

  // V2 时值合法（start ≥ 0、dur > 0 整型）
  var badTime = 0;
  for (var a = 0; a < proj.tracks.length; a++) {
    var notes2 = proj.tracks[a].notes || [];
    for (var b = 0; b < notes2.length; b++) {
      var n2 = notes2[b];
      if (!isInt(n2.start) || n2.start < 0 || !isInt(n2.dur) || n2.dur <= 0) badTime++;
    }
  }
  add('V2', '时值合法：start ≥ 0、dur > 0（整型 tick）', badTime === 0, badTime === 0 ? '全部合法' : (badTime + ' 个非法时值'));

  // V3 轨道合法（id 唯一、通道/音色范围、非空）
  var ids = {};
  var dup = [];
  var chBad = 0;
  var emptyTracks = 0;
  for (var t = 0; t < proj.tracks.length; t++) {
    var tr = proj.tracks[t];
    if (ids[tr.id]) dup.push(tr.id);
    ids[tr.id] = true;
    if (!isInt(tr.channel) || tr.channel < 0 || tr.channel > 15) chBad++;
    if (!isInt(tr.program) || tr.program < 0 || tr.program > 127) chBad++;
    if (!(tr.notes || []).length) emptyTracks++;
  }
  add('V3', '轨道合法：id 唯一 + 通道/音色在范围 + 每轨非空', proj.tracks.length > 0 && dup.length === 0 && chBad === 0 && emptyTracks === 0,
    '轨道 ' + proj.tracks.length + '；重复 id ' + dup.length + '；通道/音色越界 ' + chBad + '；空轨 ' + emptyTracks);

  // V4 SMF 回读一致（核心：导出 → 重新解析 → 逐音符比对）
  var smfMetric = '';
  var smfPass = false;
  try {
    var bytes = projectToSmf(proj);
    var parsed = smfToProject(bytes);
    var expect = [];
    for (var e1 = 0; e1 < proj.tracks.length; e1++) {
      var en = proj.tracks[e1].notes || [];
      for (var e2 = 0; e2 < en.length; e2++) expect.push(en[e2].start + ':' + en[e2].dur + ':' + en[e2].pitch + ':' + (en[e2].vel === undefined ? 90 : en[e2].vel));
    }
    var got = [];
    // parsed.tracks[0] 是 meta 轨（无音符）
    for (var g1 = 1; g1 < parsed.tracks.length; g1++) {
      var gn = parsed.tracks[g1].notes || [];
      for (var g2 = 0; g2 < gn.length; g2++) got.push(gn[g2].start + ':' + gn[g2].dur + ':' + gn[g2].pitch + ':' + gn[g2].vel);
    }
    expect.sort();
    got.sort();
    var same = expect.length === got.length;
    if (same) {
      for (var k = 0; k < expect.length; k++) if (expect[k] !== got[k]) { same = false; break; }
    }
    var tempoOk = Math.abs((parsed.tempo || 0) - (proj.tempo || 120)) <= 1;
    var meterOk = parsed.meter && parsed.meter[0] === proj.meter[0] && parsed.meter[1] === proj.meter[1];
    var ppqOk = parsed.ppq === (proj.ppq || PPQ_DEFAULT);
    // 轨道级元信息（通道/音色）必须一并回读一致：否则「导出→再导入」会静默丢音色
    var metaOk = true;
    var metaDetail = '';
    for (var ti = 0; ti < proj.tracks.length; ti++) {
      var pt2 = parsed.tracks[ti + 1]; // [0] 是 meta 轨
      if (!pt2) { metaOk = false; metaDetail = '轨道 ' + proj.tracks[ti].id + ' 在 SMF 中缺失'; break; }
      var wantProg = proj.tracks[ti].program || 0;
      var wantCh = proj.tracks[ti].channel || 0;
      if (pt2.program !== wantProg) { metaOk = false; metaDetail = '轨道 ' + proj.tracks[ti].id + ' 音色 ' + pt2.program + ' ≠ ' + wantProg; break; }
      if (pt2.channel !== wantCh) { metaOk = false; metaDetail = '轨道 ' + proj.tracks[ti].id + ' 通道 ' + pt2.channel + ' ≠ ' + wantCh; break; }
    }
    smfPass = same && tempoOk && meterOk && ppqOk && metaOk;
    smfMetric = 'SMF 字节 ' + bytes.length + '；音符 期望 ' + expect.length + ' / 回读 ' + got.length +
      '；速度 ' + (tempoOk ? '一致' : ('不一致 ' + parsed.tempo + ' vs ' + proj.tempo)) +
      '；拍号 ' + (meterOk ? '一致' : '不一致') + '；ppq ' + (ppqOk ? '一致' : '不一致') +
      '；轨道通道/音色 ' + (metaOk ? '一致' : ('不一致（' + metaDetail + '）'));
  } catch (e) {
    smfMetric = 'SMF 往返失败: ' + e.message;
  }
  add('V4', 'SMF 回读一致：导出 MIDI 再解析，音符/速度/拍号/ppq 与工程逐字段一致', smfPass, smfMetric);

  // V5 确定性（两次导出字节完全相同）
  var detPass = false;
  var detMetric = '';
  try {
    var b1 = projectToSmf(proj);
    var b2 = projectToSmf(proj);
    detPass = b1.length === b2.length;
    if (detPass) {
      for (var d = 0; d < b1.length; d++) if (b1[d] !== b2[d]) { detPass = false; break; }
    }
    detMetric = '两次导出 ' + b1.length + ' 字节' + (detPass ? '，逐字节一致' : '，不一致');
  } catch (e) {
    detMetric = '导出失败: ' + e.message;
  }
  add('V5', '确定性：同一工程两次导出 MIDI 位级一致', detPass, detMetric);

  // V6 文本真相源往返（ABC 导出 → 解析 → 音符集合一致）
  var abcPass = false;
  var abcMetric = '';
  try {
    var abc = projectToAbc(proj);
    var back = abcToProject(abc, proj.ppq || PPQ_DEFAULT);
    var srcSet = [];
    for (var s1 = 0; s1 < proj.tracks.length; s1++) {
      var sn = proj.tracks[s1].notes || [];
      for (var s2 = 0; s2 < sn.length; s2++) srcSet.push(sn[s2].start + ':' + sn[s2].dur + ':' + sn[s2].pitch);
    }
    var backSet = [];
    for (var s3 = 0; s3 < back.tracks.length; s3++) {
      var bn = back.tracks[s3].notes || [];
      for (var s4 = 0; s4 < bn.length; s4++) backSet.push(bn[s4].start + ':' + bn[s4].dur + ':' + bn[s4].pitch);
    }
    srcSet.sort();
    backSet.sort();
    abcPass = srcSet.length === backSet.length;
    if (abcPass) {
      for (var s5 = 0; s5 < srcSet.length; s5++) if (srcSet[s5] !== backSet[s5]) { abcPass = false; break; }
    }
    abcMetric = 'ABC ' + abc.length + ' 字符；音符 期望 ' + srcSet.length + ' / 回读 ' + backSet.length;
  } catch (e) {
    abcMetric = 'ABC 往返失败: ' + e.message;
  }
  add('V6', '文本真相源往返：ABC 导出再导入，音符集合（start/dur/pitch）一致', abcPass, abcMetric);

  // MusicXML 可生成性（几何校验，避免"导出的乐谱无法解析"）
  var xmlPass = false;
  var xmlMetric = '';
  try {
    var xml = projectToMusicXml(proj);
    // ★ 必须匹配 `<part id="…"`：`<part-list>` / `<part-name>` 也以 `<part` 开头，
    //   用 `<part\b` 会把它们一起计数（实测 part 5/2 的假失败就来自这里）。
    var openTags = (xml.match(/<part\s+id="/g) || []).length;
    var closeTags = (xml.match(/<\/part>/g) || []).length;
    var notesXml = (xml.match(/<note>/g) || []).length;
    xmlPass = openTags === closeTags && openTags === proj.tracks.length;
    xmlMetric = 'MusicXML ' + xml.length + ' 字符；part ' + openTags + '/' + closeTags + '；<note> ' + notesXml;
  } catch (e) {
    xmlMetric = 'MusicXML 生成失败: ' + e.message;
  }
  add('V7', 'MusicXML 结构合法：part 标签配对且与轨道数一致', xmlPass, xmlMetric);

  var passAll = true;
  for (var c = 0; c < checks.length; c++) if (!checks[c].pass) passAll = false;

  var lines = [];
  lines.push('## 音乐工程校验（' + (passAll ? '✅ 全部通过' : '❌ 存在失败项') + '）');
  lines.push('');
  for (var p = 0; p < checks.length; p++) {
    var ck = checks[p];
    lines.push('  ' + (ck.pass ? '✅' : '❌') + ' ' + ck.id + ' ' + ck.name);
    lines.push('      指标: ' + ck.metric);
  }
  return { checks: checks, pass: passAll, report: lines.join('\n') };
}

// ── 工具实现 ───────────────────────────────────────────────
function argStr(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  return String(v);
}
function argInt(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  var n = typeof v === 'number' ? v : parseInt(v, 10);
  return isInt(n) ? n : def;
}
function argBool(args, key, def) {
  var v = args[key];
  if (v === undefined || v === null || v === '') return def;
  if (typeof v === 'boolean') return v;
  return v === 'true' || v === '1' || v === 1;
}

function musicProject(args, exec, ctx) {
  var path = argStr(args, 'path', 'music.project.json');
  var mode = argStr(args, 'mode', 'show');
  if (mode === 'create') {
    if (ctx.fs.exists(path) && !argBool(args, 'overwrite', false)) {
      return '工程已存在: ' + path + '（要重建请传 overwrite=true，或改用 mode=update 修改元信息）';
    }
    var proj = emptyProject(argStr(args, 'title', undefined));
    if (isInt(args.tempo)) proj.tempo = args.tempo;
    if (args.meter) proj.meter = args.meter;
    if (args.key) proj.key = keyToString(parseKey(args.key));
    if (args.ppq) proj.ppq = args.ppq;
    // 便捷：创建时可直接给轨道定义（数组 of {id,name,program,channel}）
    if (args.tracks && args.tracks.length) {
      for (var i = 0; i < args.tracks.length; i++) {
        var t = args.tracks[i] || {};
        proj.tracks.push({
          id: t.id || ('t' + (i + 1)), name: t.name || ('轨道 ' + (i + 1)),
          channel: isInt(t.channel) ? clamp(t.channel, 0, 15) : clamp(i, 0, 15),
          program: isInt(t.program) ? clamp(t.program, 0, 127) : 0,
          notes: [],
        });
      }
    }
    saveProject(ctx, path, proj);
    return '✅ 已创建音乐工程: ' + path + '\n\n' + projectSummary(proj, path);
  }
  if (mode === 'show') {
    var p1 = loadProject(ctx, path);
    return projectSummary(p1, path);
  }
  if (mode === 'update') {
    var p2 = loadProject(ctx, path);
    var ops = [];
    if (args.title) ops.push({ op: 'project.rename', title: args.title });
    if (isInt(args.tempo)) ops.push({ op: 'set.tempo', bpm: args.tempo });
    if (args.meter) ops.push({ op: 'set.meter', num: args.meter[0], den: args.meter[1] });
    if (args.key) ops.push({ op: 'set.key', key: args.key });
    if (!ops.length) return '未提供任何要更新的字段（title/tempo/meter/key 至少一个）';
    var log = applyOps(p2, ops);
    saveProject(ctx, path, p2);
    return '✅ 已更新工程: ' + path + '\n\n' + log.join('\n') + '\n\n' + projectSummary(p2, path);
  }
  throw new Error('未知 mode: ' + mode + '（可用 create / show / update）');
}

function musicEdit(args, exec, ctx) {
  var path = argStr(args, 'path', 'music.project.json');
  var proj = loadProject(ctx, path);
  var ops = args.ops;
  if (args.op) ops = [{ op: args.op, track: args.track, start: args.start, dur: args.dur, pitch: args.pitch, vel: args.vel, semitones: args.semitones, filter: args.filter, root: args.root, quality: args.quality, scale: args.scale, steps: args.steps, grid: args.grid, bpm: args.bpm, num: args.num, den: args.den, key: args.key, title: args.title, times: args.times, id: args.id, name: args.name, program: args.program, set: args.set, delta: args.delta, deltaStart: args.deltaStart, deltaPitch: args.deltaPitch, fields: args.fields }];
  if (!ops || !ops.length) throw new Error('需要 ops 数组（或 op + 同层参数）');
  if (argStr(args, 'mode', 'append') === 'replace') {
    // 只重置元信息，不动音符（音符合并语义见文档）
    throw new Error('mode=replace 暂不支持（编辑链是命令式的，直接追加即可）');
  }
  var log = applyOps(proj, ops);
  saveProject(ctx, path, proj);
  return '✅ 已应用 ' + ops.length + ' 条编辑命令到 ' + path + '\n\n' + log.join('\n') + '\n\n' + projectSummary(proj, path);
}

function musicImport(args, exec, ctx) {
  var path = argStr(args, 'path', undefined);
  if (!path) throw new Error('需要 path（要导入的源文件）');
  var fmt = argStr(args, 'format', 'auto');
  var out = argStr(args, 'out', 'music.project.json');
  var ppq = argInt(args, 'ppq', PPQ_DEFAULT);
  if (fmt === 'auto') {
    var lower = path.toLowerCase();
    if (/\.midi?$/.test(lower)) fmt = 'midi';
    else if (/\.abc$/.test(lower)) fmt = 'abc';
    else if (/\.(musicxml|xml)$/.test(lower)) fmt = 'musicxml';
    else fmt = 'abc';
  }
  var proj = null;
  var detail = '';
  if (fmt === 'midi') {
    var b64;
    try { b64 = ctx.fs.readFileBase64(path); } catch (e) { throw new Error('读取 MIDI 失败（需要二进制读取能力）: ' + e.message); }
    var bytes = b64Decode(b64);
    var parsed = smfToProject(bytes);
    // 组装工程：meta 轨（无音符）跳过，其余按顺序成轨
    var tracks = [];
    var order = 0;
    for (var i = 0; i < parsed.tracks.length; i++) {
      if (!parsed.tracks[i].notes.length) continue;
      tracks.push({
        id: 't' + (++order), name: parsed.tracks[i].name || ('轨道 ' + order),
        channel: isInt(parsed.tracks[i].channel) ? clamp(parsed.tracks[i].channel, 0, 15) : clamp(order - 1, 0, 15),
        program: isInt(parsed.tracks[i].program) ? clamp(parsed.tracks[i].program, 0, 127) : 0,
        notes: parsed.tracks[i].notes,
      });
    }
    proj = emptyProject(path.split(/[\\/]/).pop());
    proj.ppq = parsed.ppq || ppq;
    proj.tempo = parsed.tempo || 120;
    proj.meter = parsed.meter || [4, 4];
    proj.tracks = tracks;
    detail = 'SMF format ' + parsed.format + '，' + parsed.tracks.length + ' 个 chunk（含 meta）';
  } else if (fmt === 'abc') {
    var txt = ctx.fs.readFile(path);
    proj = abcToProject(txt, ppq);
    detail = 'ABC ' + txt.length + ' 字符';
  } else if (fmt === 'musicxml') {
    var xml = ctx.fs.readFile(path);
    proj = musicXmlToProject(xml, ppq);
    detail = 'MusicXML ' + xml.length + ' 字符';
  } else {
    throw new Error('未知 format: ' + fmt + '（可用 midi / abc / musicxml / auto）');
  }
  if (!proj.tracks.length) throw new Error('导入结果没有任何音符轨道（源文件可能为空或格式不受支持）');
  saveProject(ctx, out, proj);
  var total = 0;
  for (var t = 0; t < proj.tracks.length; t++) total += proj.tracks[t].notes.length;
  return '✅ 已导入 ' + fmt.toUpperCase() + ' → ' + out + '\n来源: ' + path + '（' + detail + '）\n\n' + projectSummary(proj, out);
}

function musicExport(args, exec, ctx) {
  var path = argStr(args, 'path', 'music.project.json');
  var proj = loadProject(ctx, path);
  var fmt = argStr(args, 'format', 'midi');
  var out = argStr(args, 'out', '');
  var info = '';
  if (fmt === 'midi') {
    if (!out) out = 'music.mid';
    var bytes = projectToSmf(proj);
    ctx.fs.writeFileBase64(out, b64Encode(bytes));
    info = 'SMF format 1，' + bytes.length + ' 字节，' + (proj.tracks.length + 1) + ' 个 chunk（meta + ' + proj.tracks.length + ' 轨）';
  } else if (fmt === 'abc') {
    if (!out) out = 'music.abc';
    var abc = projectToAbc(proj);
    ctx.fs.writeFile(out, abc);
    info = 'ABC ' + abc.length + ' 字符';
  } else if (fmt === 'musicxml') {
    if (!out) out = 'music.musicxml';
    var xml = projectToMusicXml(proj);
    ctx.fs.writeFile(out, xml);
    info = 'MusicXML ' + xml.length + ' 字符';
  } else if (fmt === 'svg') {
    if (!out) out = 'music.score.svg';
    var svg = projectToSvg(proj);
    ctx.fs.writeFile(out, svg);
    info = '乐谱 SVG ' + svg.length + ' 字符（可用 read_image 让模型直接「看」乐谱核对）';
  } else {
    throw new Error('未知 format: ' + fmt + '（可用 midi / abc / musicxml / svg）');
  }
  proj.artifacts = proj.artifacts || {};
  proj.artifacts[fmt] = out;
  saveProject(ctx, path, proj);
  return '✅ 已导出 ' + fmt.toUpperCase() + ' → ' + out + '\n' + info + '（工程 ' + path + ' 已登记产物）';
}

function musicVerify(args, exec, ctx) {
  var path = argStr(args, 'path', 'music.project.json');
  var proj = loadProject(ctx, path);
  var res = verifyProject(ctx, proj, args);
  // ★ 旁挂校验报告（供「音乐」面板展示）：**不写进工程 JSON** ——
  //   工程是音乐真相源，必须保持纯净可 diff，不能混入带时间戳的验证结果。
  var reportPath = argStr(args, 'report', 'music.verify.json');
  var reportErr = '';
  try {
    ctx.fs.writeFile(reportPath, JSON.stringify({ ts: nowIso(), project: path, pass: res.pass, checks: res.checks }, null, 2));
  } catch (e) {
    reportErr = '\n⚠️ 校验报告写入失败（' + reportPath + '）: ' + ((e && e.message) ? e.message : e) + '（不影响校验结论）';
  }
  var tail = res.pass ? '\n\n工程: ' + path + '（全部判据通过，可交付）' : '\n\n工程: ' + path + '（存在失败项——修完再导出）';
  return res.report + tail + '\n校验报告: ' + reportPath + '（供 UI 面板展示）' + reportErr;
}

// ── 工具声明 ───────────────────────────────────────────────
var TOOL_DEFS = [
  {
    name: 'music_project',
    description: '音乐工程管理（文本真相源 music.project.json）：创建/查看/更新标题、速度、拍号、调号、轨道定义。音符时间一律用整数 tick（ppq 默认 480），保证可 diff 与位级确定性；MIDI/乐谱都只是由工程生成的产物。',
    usageGuide: '创作第一步：mode=create 建工程（可带 tracks=[{id,name,program,channel}]）→ music_edit 加音符/和弦/音阶 → music_export 导出 MIDI/ABC/MusicXML/SVG 乐谱 → music_verify 校验。mode=show 只看摘要，mode=update 改元信息。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '工程文件路径（默认 <主项目根>/music.project.json；相对主项目根解析，跨项目传绝对路径）' },
        mode: { type: 'string', description: 'create（新建）| show（默认，查看摘要）| update（改元信息）' },
        title: { type: 'string', description: '可选：乐曲标题' },
        tempo: { type: 'integer', description: '可选：速度 BPM（20..400，默认 120）' },
        meter: { type: 'array', description: '可选：拍号 [分子, 分母]，默认 [4,4]' },
        key: { type: 'string', description: '可选：调号（C / Am / F# minor）' },
        ppq: { type: 'integer', description: '可选：每四分音符 tick 数（默认 480）' },
        tracks: { type: 'array', description: '可选（仅 create）：轨道定义 [{id,name,program,channel}]' },
        overwrite: { type: 'boolean', description: '可选（仅 create）：工程已存在时是否重建（默认 false）' },
      },
    },
  },
  {
    name: 'music_edit',
    description: '向音乐工程追加编辑命令（命令式 op，与 UI 操作同源）。支持：音符（note.add/remove/move/transpose/quantize/velocity/set）、和弦（chord.add，14 种性质）、音阶跑动（scale.add，13 种音阶）、轨道（track.add/remove/rename/instrument/repeat）、全局（set.tempo/set.meter/set.key/project.rename）。所有 op 幂等可重放，作用对象是 ticket 级精确的工程 JSON。',
    usageGuide: 'op 示例：{"op":"note.add","track":"t1","start":0,"dur":480,"pitch":60} / {"op":"chord.add","track":"t1","start":0,"dur":1920,"root":"C4","quality":"maj7"} / {"op":"note.quantize","grid":120} / {"op":"track.repeat","id":"t1","times":2}。filter 选择器支持 index/pitch/pitchMin/pitchMax/startFrom/startTo。改完用 music_verify 校验、music_export 出产物。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/music.project.json；相对主项目根解析）' },
        ops: { type: 'array', description: '编辑命令数组（按顺序应用）' },
        op: { type: 'string', description: '可选：单条 op 名（与同层参数合成为一条命令）' },
        track: { type: 'string', description: '可选：目标轨道 id（或序号）' },
      },
      required: ['ops'],
    },
  },
  {
    name: 'music_import',
    description: '把外部记谱导入为音乐工程（文本真相源）：MIDI（SMF，二进制安全读取 + 变长量/事件解析）、ABC（子集：X/T/M/L/Q/K + 音名升降号与八度、时值数字与斜杠、休止符 z）、MusicXML（partwise，含 chord/rest/duration/divisions）。导入即转成 tick 级工程 JSON，之后全部可 diff。',
    usageGuide: 'format=auto 按扩展名判断（.mid/.abc/.musicxml|.xml）。导入产物是工程 JSON（默认 music.project.json），随后可用 music_edit 改、music_export 重新导出。若导入 MIDI 得到的轨道名为空，用 track.rename 补。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '要导入的源文件（相对主项目根解析，跨项目传绝对路径）' },
        format: { type: 'string', description: 'auto（默认）| midi | abc | musicxml' },
        out: { type: 'string', description: '可选：输出的工程路径（默认 <主项目根>/music.project.json；相对主项目根解析）' },
        ppq: { type: 'integer', description: '可选：ABC/MusicXML 导入时的 ppq（MIDI 以其自身 division 为准）' },
      },
      required: ['path'],
    },
  },
  {
    name: 'music_export',
    description: '把音乐工程导出为产物：MIDI（SMF format 1，meta 轨含速度/拍号/调号 + 每轨 program change 与音符事件，同 tick 时 note-off 优先）、ABC、MusicXML 3.1（partwise，part 与轨道一一对应）、SVG 简易乐谱（五线谱 + 小节线 + 加线，供模型「看图」核对）。导出不改变音符，只登记 artifacts。',
    usageGuide: '先 music_verify 通过再导出最稳（V4 保证导出的 MIDI 能被自己回读）。SVG 乐谱配合 read_image 做听觉之外的可判读产物；MIDI 用 writeFileBase64 写入，二进制安全。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/music.project.json；相对主项目根解析）' },
        format: { type: 'string', description: 'midi（默认）| abc | musicxml | svg' },
        out: { type: 'string', description: '可选：输出路径（默认 <主项目根>/music.mid / music.abc / music.musicxml / music.score.svg）' },
      },
    },
  },
  {
    name: 'music_verify',
    description: '音乐工程专项自检（7 项判据）：V1 音符合法（pitch 整型 ∈[0,127]）、V2 时值合法（start≥0、dur>0）、V3 轨道合法（id 唯一/通道音色在范围/非空）、V4 SMF 回读一致（导出 MIDI 再解析，音符/速度/拍号/ppq/轨道通道与音色逐字段比对）、V5 确定性（两次导出位级一致）、V6 文本真相源往返（ABC 导出再导入音符集合一致）、V7 MusicXML 结构合法（part 标签配对）。任一 FAIL 说明产物与真相源不一致，应视为构建失败。结果同时写成旁挂报告（默认 music.verify.json）供 UI 面板展示——写报告不改工程本体。',
    usageGuide: '导出前跑一遍最省事；也可在 CI 里对工程文件跑（只读）。V4/V5/V6 是本插件的核心契约：文本真相源 ⇄ 二进制产物 ⇄ 图形产物三者必须一致。',
    category: '创作',
    parameters: {
      type: 'object',
      properties: {
        path: { type: 'string', description: '可选：工程路径（默认 <主项目根>/music.project.json；相对主项目根解析）' },
        report: { type: 'string', description: '可选：旁挂校验报告路径（默认 <主项目根>/music.verify.json）' },
      },
    },
  },
];

var IMPLS = {
  music_project: musicProject,
  music_edit: musicEdit,
  music_import: musicImport,
  music_export: musicExport,
  music_verify: musicVerify,
};

// 双轨入口：goja 沙箱把本文件当函数体执行 → 顶层 return；
// Node 侧（测试/打包）走 module.exports。
var PLUGIN = {
  name: 'tool-music',
  purpose: '音乐创作工具面（纯 goja 零依赖）：工程 JSON 文本真相源 + MIDI/ABC/MusicXML 读写 + SVG 乐谱 + 7 项自检 —— 音符级精确、可 diff、产物可回读',
  // ★ logger 必须显式 inject：宿主按注入表装配 ctx 服务（jsplugin.go:1839 case "logger"），
  //   未声明时 ctx.logger 为 undefined —— 装载日志会静默丢失。
  inject: ['fs', 'logger'],
  apply: function (ctx) {
    for (var i = 0; i < TOOL_DEFS.length; i++) {
      (function (t) {
        ctx.tools.register({
          name: t.name,
          description: t.description,
          usageGuide: t.usageGuide,
          category: t.category,
          readOnly: !!t.readOnly,
          parameters: t.parameters,
          execute: function (args) { return IMPLS[t.name](args || {}, {}, ctx); },
        });
      })(TOOL_DEFS[i]);
    }
    try {
      // ★ 宿主 logger 是「服务工厂」：ctx.logger(scope) → {log,info,warn,…}
      //   （见 jsplugin.go buildLoggerService）；写成 ctx.logger.info 会静默不输出。
      if (ctx && typeof ctx.logger === 'function') {
        ctx.logger('tool-music').info('已注册 ' + TOOL_DEFS.length + ' 个工具（goja 轨 · 零依赖 · 文本真相源）');
      }
    } catch (_) { /* ignore */ }
  },
};

if (typeof module !== 'undefined' && module.exports) module.exports = PLUGIN;
return PLUGIN;
