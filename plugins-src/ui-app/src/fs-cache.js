// fs-cache.js — 目录列表（/fs/list）共享缓存
//
// ★ 2026-10-15 性能修复：此前每个 FileTreeItem 各自请求并把结果存在**组件局部**
//   `children` ref 里，由此产生两处可感知卡顿：
//   ① 侧栏视图切换（Sidebar 用 v-if 链）会销毁重建组件 —— `state.expandedDirs`
//      记录的展开态还在，子节点却随组件实例一起丢失 → 切回左侧栏是「空树 /
//      点了没反应」（要么重新点两次，要么等一次网络往返）。
//   ② app-actions 每 5s（窗口可见时）派发 refresh-tree，FileTreeItem 对**每个**
//      已展开目录各发一次 /fs/list；返回后 `children.value = entries` **整体替换
//      数组** → 整棵子树重新渲染（展开十几层即十几次并发请求 + 全树重挂）。
//
//   本模块把目录列表收敛为**一份共享缓存**：同路径并发请求去重（in-flight）、
//   短 TTL 内复用、组件重建也能立即拿到内容；写操作后按前缀失效。
import { reactive } from 'vue'
import api from './api.js'

// TTL 略小于周期刷新间隔（5s）⇒ 周期刷新仍会真正重取，
// 而「视图切换 / 组件重挂 / 折叠再展开」这类高频动作全部命中缓存。
const TTL = 4000

// path -> entries[]（reactive：模板/computed 可直接追踪，缓存更新自动反映）
export const dirMap = reactive({})
// path -> 写入时间戳
const stampMap = {}
// path(+browse) -> Promise（同路径并发去重）
const inflight = new Map()

// 读缓存（未加载过返回 null，区别于「已加载但为空目录」的 []）
export function cachedDir(path) {
  if (!path) return null
  const v = dirMap[path]
  return Array.isArray(v) ? v : null
}

function isFresh(path) {
  return cachedDir(path) !== null && (Date.now() - (stampMap[path] || 0)) < TTL
}

// 写缓存（请求返回、本地写操作后同步 patch 都用它）
export function setDir(path, entries) {
  if (!path) return
  dirMap[path] = Array.isArray(entries) ? entries : []
  stampMap[path] = Date.now()
}

// 列目录：缓存优先 → 并发去重 → 网络；失败时回落既有缓存（不把树清空）
export function listDir(path, opts) {
  const o = opts || {}
  if (!path) return Promise.resolve([])
  if (!o.force && isFresh(path)) return Promise.resolve(dirMap[path])
  const key = o.browse ? path + '|browse' : path
  const running = inflight.get(key)
  if (running) return running
  const params = o.browse ? { path: path, browse: '1' } : { path: path }
  const p = api.apiGet('/fs/list', params).then(function (entries) {
    setDir(path, entries)
    inflight.delete(key)
    return dirMap[path]
  }, function () {
    inflight.delete(key)
    return cachedDir(path) || []
  })
  inflight.set(key, p)
  return p
}

// 失效单个目录及其子孙（新建/删除/重命名/复制后子目录内容也会变）
export function invalidateDirDeep(path) {
  if (!path) return
  const base = String(path).replace(/[\\/]+$/, '')
  for (const k of Object.keys(dirMap)) {
    if (k === base || k.indexOf(base + '\\') === 0 || k.indexOf(base + '/') === 0) {
      delete dirMap[k]
      delete stampMap[k]
    }
  }
}

// 全量失效（刷新按钮 / 切换工作区）
export function invalidateAllDirs() {
  for (const k of Object.keys(dirMap)) delete dirMap[k]
  for (const k of Object.keys(stampMap)) delete stampMap[k]
  inflight.clear()
}
