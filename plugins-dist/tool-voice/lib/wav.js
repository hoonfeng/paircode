'use strict';
// ═══════════════════════════════════════════════════════════════
// lib/wav.js — WAV 解析 / 写出（零第三方依赖）
//
// 读取：RIFF/WAVE 容器，支持 PCM u8 / i16 / i24 / i32、IEEE float32/float64、
//       WAVE_FORMAT_EXTENSIBLE（0xFFFE，按 SubFormat GUID 取真实格式）。
// 写出：PCM i16 / i24 / i32 与 IEEE float32（默认 i24 —— 交付默认 24-bit）。
//
// 设计约束（人声子域）：
//   · 解析结果一律归一化到 [-1, 1]（Float32Array 每声道一份）——后续 DSP 与
//     @audio/* 的输入约定一致；
//   · 不做采样率归一化（保持源采样率）——避免隐式重采样引入伪影，
//     同时让「输出是源的变换」这一承诺在采样率维度上也是恒等的。
// ═══════════════════════════════════════════════════════════════

const RIFF = 0x52494646; // 'RIFF'
const WAVE = 0x57415645; // 'WAVE'
const FMT_ = 0x666d7420; // 'fmt '
const DATA = 0x64617461; // 'data'

const FORMAT_PCM = 1;
const FORMAT_FLOAT = 3;
const FORMAT_EXTENSIBLE = 0xfffe;

// readStr 读 4 字节 ASCII 标签（调试用）。
function readStr(buf, off) {
  return String.fromCharCode(buf[off], buf[off + 1], buf[off + 2], buf[off + 3]);
}

// parseWav 解析 WAV 缓冲。
// 返回 { fs, channels, frames, bitDepth, format, extensible, data: Float32Array[] }
function parseWav(buf) {
  if (!Buffer.isBuffer(buf) || buf.length < 44) throw new Error('WAV 数据过短（<44 字节）');
  if (buf.readUInt32BE(0) !== RIFF) throw new Error('不是 RIFF 容器（WAV 头缺失）');
  if (buf.readUInt32BE(8) !== WAVE) throw new Error('RIFF 容器不是 WAVE 类型');

  let off = 12;
  let fmt = null;
  let dataOff = -1;
  let dataLen = -1;
  while (off + 8 <= buf.length) {
    const id = buf.readUInt32BE(off);
    const size = buf.readUInt32LE(off + 4);
    const body = off + 8;
    if (id === FMT_ && size >= 16) {
      let audioFormat = buf.readUInt16LE(body);
      const channels = buf.readUInt16LE(body + 2);
      const fs = buf.readUInt32LE(body + 4);
      const bitDepth = buf.readUInt16LE(body + 14);
      let extensible = false;
      if (audioFormat === FORMAT_EXTENSIBLE) {
        extensible = true;
        if (size >= 40) {
          // SubFormat GUID 前 2 字节 = 真实格式码
          audioFormat = buf.readUInt16LE(body + 24);
        }
      }
      fmt = { audioFormat, channels, fs, bitDepth, extensible };
    } else if (id === DATA) {
      dataOff = body;
      dataLen = Math.min(size, buf.length - body);
    }
    // chunk 按偶数字节对齐（奇数长度补 1 字节）
    off = body + size + (size % 2);
  }
  if (!fmt) throw new Error('WAV 缺 fmt 块');
  if (dataOff < 0) throw new Error('WAV 缺 data 块');
  if (!(fmt.channels > 0) || !(fmt.fs > 0)) throw new Error(`WAV fmt 非法（ch=${fmt.channels}, fs=${fmt.fs}）`);

  const bytesPerSample = fmt.bitDepth >> 3;
  if (bytesPerSample <= 0) throw new Error(`WAV 位深非法: ${fmt.bitDepth}`);
  const frameBytes = bytesPerSample * fmt.channels;
  const frames = Math.floor(dataLen / frameBytes);
  const chans = [];
  for (let c = 0; c < fmt.channels; c++) chans.push(new Float32Array(frames));

  const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  const isFloat = fmt.audioFormat === FORMAT_FLOAT;
  const isPcm = fmt.audioFormat === FORMAT_PCM;
  if (!isFloat && !isPcm) {
    throw new Error(`不支持的 WAV 编码格式 0x${fmt.audioFormat.toString(16)}（支持 PCM 与 IEEE float）`);
  }

  for (let i = 0; i < frames; i++) {
    for (let c = 0; c < fmt.channels; c++) {
      const p = dataOff + i * frameBytes + c * bytesPerSample;
      let v = 0;
      if (isFloat) {
        if (fmt.bitDepth === 32) v = dv.getFloat32(p, true);
        else if (fmt.bitDepth === 64) v = dv.getFloat64(p, true);
        else throw new Error(`IEEE float 仅支持 32/64 位（当前 ${fmt.bitDepth}）`);
      } else if (fmt.bitDepth === 8) {
        v = (dv.getUint8(p) - 128) / 128; // u8 无符号，128 为零点
      } else if (fmt.bitDepth === 16) {
        v = dv.getInt16(p, true) / 32768;
      } else if (fmt.bitDepth === 24) {
        const b0 = dv.getUint8(p), b1 = dv.getUint8(p + 1), b2 = dv.getInt8(p + 2);
        v = ((b2 << 16) | (b1 << 8) | b0) / 8388608;
      } else if (fmt.bitDepth === 32) {
        v = dv.getInt32(p, true) / 2147483648;
      } else {
        throw new Error(`不支持的 PCM 位深 ${fmt.bitDepth}`);
      }
      chans[c][i] = v;
    }
  }
  return {
    fs: fmt.fs,
    channels: fmt.channels,
    frames,
    bitDepth: fmt.bitDepth,
    format: fmt.audioFormat,
    extensible: fmt.extensible,
    data: chans,
  };
}

// toMono 多声道下混为单声道（算术平均；人声轨通常已是单声道，立体声下混仅用于
// 混合素材——不做 mid/side 提取，那需要用户显式选择）。
function toMono(chans) {
  if (!chans || chans.length === 0) throw new Error('无声道数据');
  if (chans.length === 1) return chans[0];
  const n = chans[0].length;
  const out = new Float32Array(n);
  for (let i = 0; i < n; i++) {
    let s = 0;
    for (let c = 0; c < chans.length; c++) s += chans[c][i];
    out[i] = s / chans.length;
  }
  return out;
}

// writeWav 把每声道 Float32Array 写成 WAV Buffer（默认 24-bit PCM）。
// opts: { fs, bitDepth = 24, float = false }
function writeWav(chans, opts) {
  const o = opts || {};
  const fs = Math.round(Number(o.fs) || 0);
  if (!(fs > 0)) throw new Error('writeWav 需要有效采样率 fs');
  const list = Array.isArray(chans) ? chans : [chans];
  if (list.length === 0) throw new Error('writeWav 无声道数据');
  const frames = list[0].length;
  const bitDepth = o.float ? 32 : (Number(o.bitDepth) || 24);
  const bytesPerSample = bitDepth >> 3;
  const audioFormat = o.float ? FORMAT_FLOAT : FORMAT_PCM;
  const dataBytes = frames * list.length * bytesPerSample;
  const header = 44;
  const buf = Buffer.alloc(header + dataBytes);
  const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);

  buf.write('RIFF', 0, 'ascii');
  buf.writeUInt32LE(36 + dataBytes, 4);
  buf.write('WAVE', 8, 'ascii');
  buf.write('fmt ', 12, 'ascii');
  buf.writeUInt32LE(16, 16);
  buf.writeUInt16LE(audioFormat, 20);
  buf.writeUInt16LE(list.length, 22);
  buf.writeUInt32LE(fs, 24);
  buf.writeUInt32LE(fs * list.length * bytesPerSample, 28); // byteRate
  buf.writeUInt16LE(list.length * bytesPerSample, 32); // blockAlign
  buf.writeUInt16LE(bitDepth, 34);
  buf.write('data', 36, 'ascii');
  buf.writeUInt32LE(dataBytes, 40);

  // 削波保护：写出前钳制并记录真实越界量（由调用方决定是否阻断）
  const clamp = (v) => (v > 1 ? 1 : v < -1 ? -1 : v);
  let p = header;
  for (let i = 0; i < frames; i++) {
    for (let c = 0; c < list.length; c++) {
      const v = clamp(list[c][i]);
      if (o.float) {
        dv.setFloat32(p, v, true);
      } else if (bitDepth === 16) {
        dv.setInt16(p, Math.max(-32768, Math.min(32767, Math.round(v * 32767))), true);
      } else if (bitDepth === 24) {
        const s = Math.max(-8388608, Math.min(8388607, Math.round(v * 8388607)));
        const u = s < 0 ? s + 16777216 : s;
        dv.setUint8(p, u & 0xff);
        dv.setUint8(p + 1, (u >> 8) & 0xff);
        dv.setUint8(p + 2, (u >> 16) & 0xff);
      } else if (bitDepth === 32) {
        dv.setInt32(p, Math.max(-2147483648, Math.min(2147483647, Math.round(v * 2147483647))), true);
      } else {
        throw new Error(`不支持的写出位深 ${bitDepth}`);
      }
      p += bytesPerSample;
    }
  }
  return buf;
}

// 便捷：解析后直接下混单声道。
function loadMono(buf) {
  const w = parseWav(buf);
  return { fs: w.fs, mono: toMono(w.data), meta: { channels: w.channels, frames: w.frames, bitDepth: w.bitDepth, format: w.format } };
}

module.exports = { parseWav, writeWav, toMono, loadMono, readStr };
