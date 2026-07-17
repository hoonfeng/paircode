// ─── api.js（向后兼容包装层） ─────────────────────────────────
// 实际实现在 sdk.js（双模式通信 SDK），本文件仅做 re-export，
// 保持现有 import api from './api.js' 的调用方不受影响。
//
// 当前为 Web 模式（HTTP+WS）；设置 window.__DESKTOP_MODE__ = true
// 且 window.desktopBridge 存在时自动切换为桌面模式（Go 直调）。
// ─────────────────────────────────────────────────────────────

import sdk from './sdk.js'

// 重新导出所有命名导出
export const apiGet = sdk.apiGet
export const apiPost = sdk.apiPost
export const apiPut = sdk.apiPut
export const apiDelete = sdk.apiDelete

export const initWebSocket = sdk.initWebSocket
export const closeWebSocket = sdk.closeWebSocket
export const isWebSocketOpen = sdk.isWebSocketOpen

export const chatStart = sdk.chatStart
export const chatStop = sdk.chatStop
export const answerChat = sdk.answerChat
export const approveChat = sdk.approveChat
export const sendFeedback = sdk.sendFeedback
export const chatRollback = sdk.chatRollback

export const getMessages = sdk.getMessages
export const getMessagesCount = sdk.getMessagesCount

export const getModels = sdk.getModels

export const getMcpList = sdk.getMcpList
export const saveMcpItem = sdk.saveMcpItem

export const getSkillsList = sdk.getSkillsList
export const readSkill = sdk.readSkill
export const deleteSkill = sdk.deleteSkill
export const saveSkillStatus = sdk.saveSkillStatus

export const getInstructions = sdk.getInstructions
export const saveInstructions = sdk.saveInstructions

export const getPhilosophy = sdk.getPhilosophy
export const savePhilosophy = sdk.savePhilosophy

export default sdk
