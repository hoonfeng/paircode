<template>
  <svg class="svg-icon" :width="size" :height="size" viewBox="0 0 24 24"
       fill="none" stroke="currentColor" stroke-width="2"
       stroke-linecap="round" stroke-linejoin="round">
    <!-- Folder -->
    <template v-if="name === 'folder'">
      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
    </template>
    <!-- Folder Open -->
    <template v-else-if="name === 'folder-open'">
      <path d="M6 17l-3-9h18l-3 9H6z"/>
      <path d="M4 8V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v3"/>
    </template>
    <!-- File -->
    <template v-else-if="name === 'file'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
    </template>
    <!-- File Code -->
    <template v-else-if="name === 'file-code'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
      <line x1="10" y1="12" x2="8" y2="14"/><line x1="10" y1="16" x2="8" y2="18"/>
      <line x1="14" y1="12" x2="16" y2="14"/><line x1="14" y1="16" x2="16" y2="18"/>
    </template>
    <!-- File Text / Document -->
    <template v-else-if="name === 'file-text'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
      <line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="15" y2="17"/>
    </template>
    <!-- Search -->
    <template v-else-if="name === 'search'">
      <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
    </template>
    <!-- Terminal / Console -->
    <template v-else-if="name === 'terminal'">
      <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
    </template>
    <!-- Chat / Message -->
    <template v-else-if="name === 'chat'">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
    </template>
    <!-- Gear / Settings -->
    <template v-else-if="name === 'settings'">
      <circle cx="12" cy="12" r="3"/>
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
    </template>
    <!-- Home -->
    <template v-else-if="name === 'home'">
      <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
      <polyline points="9 22 9 12 15 12 15 22"/>
    </template>
    <!-- Chevron Right -->
    <template v-else-if="name === 'chevron-right'">
      <polyline points="9 6 15 12 9 18"/>
    </template>
    <!-- Chevron Down (Rotated chevron-right) -->
    <template v-else-if="name === 'chevron-down'">
      <polyline points="6 9 12 15 18 9"/>
    </template>
    <!-- Plus / Add -->
    <template v-else-if="name === 'plus'">
      <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
    </template>
    <!-- Close / X -->
    <template v-else-if="name === 'close'">
      <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
    </template>
    <!-- Refresh -->
    <template v-else-if="name === 'refresh'">
      <polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
    </template>
    <!-- Hard Drive / Disk -->
    <template v-else-if="name === 'drive'">
      <line x1="22" y1="12" x2="2" y2="12"/><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/><line x1="6" y1="16" x2="6.01" y2="16"/><line x1="10" y1="16" x2="10.01" y2="16"/>
    </template>
    <!-- Source Control / Git Branch -->
    <template v-else-if="name === 'source-control'">
      <line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>
    </template>
    <!-- Git Branch -->
    <template v-else-if="name === 'git-branch'">
      <line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>
    </template>
    <!-- Git Pull -->
    <template v-else-if="name === 'git-pull'">
      <circle cx="18" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><path d="M13 6h3a2 2 0 0 1 2 2v7"/><line x1="6" y1="18" x2="6" y2="9"/><polyline points="9 9 6 6 3 9"/>
    </template>
    <!-- Git Push -->
    <template v-else-if="name === 'git-push'">
      <circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M13 18h-2a2 2 0 0 1-2-2V9"/><line x1="6" y1="6" x2="6" y2="15"/><polyline points="9 15 6 18 3 15"/>
    </template>
    <!-- Output / Window -->
    <template v-else-if="name === 'output'">
      <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/>
    </template>
    <!-- Warning / Alert -->
    <template v-else-if="name === 'warning'">
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
    </template>
    <!-- Undo -->
    <template v-else-if="name === 'undo'">
      <polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/>
    </template>
    <!-- Redo -->
    <template v-else-if="name === 'redo'">
      <polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
    </template>
    <!-- Package / Box / Store -->
    <template v-else-if="name === 'package'">
      <line x1="16.5" y1="9.4" x2="7.5" y2="4.21"/><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/>
    </template>
    <!-- Globe / External -->
    <template v-else-if="name === 'globe'">
      <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
    </template>
    <!-- Refresh / Cycle (for agent) -->
    <template v-else-if="name === 'cycle'">
      <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10"/><path d="M20.49 15a9 9 0 0 1-14.85 3.36L1 14"/>
    </template>
    <!-- Send (arrow up) -->
    <template v-else-if="name === 'send'">
      <line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/>
    </template>
    <!-- Wrench / Tool -->
    <template v-else-if="name === 'wrench'">
      <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>
    </template>
    <!-- Database -->
    <template v-else-if="name === 'database'">
      <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
    </template>
    <!-- User / Person -->
    <template v-else-if="name === 'user'">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
    </template>
    <!-- Info -->
    <template v-else-if="name === 'info'">
      <circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/>
    </template>
    <!-- Lightbulb / Suggestion -->
    <template v-else-if="name === 'lightbulb'">
      <path d="M9 18h6"/><path d="M10 22h4"/><path d="M15.09 14c.18-.98.65-1.74 1.41-2.5A4.65 4.65 0 0 0 18 8 6 6 0 0 0 6 8c0 1 .23 2.23 1.5 3.5A4.61 4.61 0 0 1 8.91 14"/>
    </template>
    <!-- Sparkles / Auto -->
    <template v-else-if="name === 'sparkles'">
      <path d="M13.5 4L15 8l4 .5L15 12l1.5 4-4-2-4 2L10 12l-4-3.5L10 8z"/><line x1="3" y1="18" x2="3" y2="21"/><line x1="21" y1="18" x2="21" y2="21"/><line x1="7" y1="20" x2="11" y2="20"/><line x1="17" y1="20" x2="19" y2="20"/>
    </template>
    <!-- Bot / AI -->
    <template v-else-if="name === 'bot'">
      <rect x="3" y="11" width="18" height="10" rx="2"/><circle cx="12" cy="5" r="2"/><path d="M12 7v4"/><line x1="8" y1="16" x2="8" y2="16"/><line x1="16" y1="16" x2="16" y2="16"/>
    </template>
    <!-- File Type Icons -->
    <template v-else-if="name === 'file-js'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="8" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">JS</text>
    </template>
    <template v-else-if="name === 'file-ts'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="8" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">TS</text>
    </template>
    <template v-else-if="name === 'file-go'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="9" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">Go</text>
    </template>
    <template v-else-if="name === 'file-py'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="7" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">Py</text>
    </template>
    <template v-else-if="name === 'file-java'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="6" y="17" font-size="8" fill="currentColor" font-weight="bold" stroke="none">Java</text>
    </template>
    <template v-else-if="name === 'file-html'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="6" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">HTML</text>
    </template>
    <template v-else-if="name === 'file-css'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="7" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">CSS</text>
    </template>
    <template v-else-if="name === 'file-json'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="5" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">{ }</text>
    </template>
    <template v-else-if="name === 'file-md'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="7" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">MD</text>
    </template>
    <template v-else-if="name === 'file-vue'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <text x="7" y="17" font-size="9" fill="currentColor" font-weight="bold" stroke="none">Vue</text>
    </template>
    <!-- Copy -->
    <template v-else-if="name === 'copy'">
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
    </template>
    <!-- Minus -->
    <template v-else-if="name === 'minus'">
      <line x1="5" y1="12" x2="19" y2="12"/>
    </template>
    <!-- Edit / Rename -->
    <template v-else-if="name === 'edit'">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
    </template>
    <!-- Trash / Delete -->
    <template v-else-if="name === 'trash'">
      <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
    </template>
    <!-- File Plus / New File -->
    <template v-else-if="name === 'file-plus'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/>
    </template>
    <!-- Folder Plus / New Folder -->
    <template v-else-if="name === 'message-square'">
    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
  </template>
  <template v-else-if="name === 'folder-plus'">
      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2v3"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>
    </template>
    <!-- Brain / Thinking -->
    <template v-else-if="name === 'brain'">
      <path d="M12 2a4 4 0 0 0-4 4v1a5 5 0 0 0-5 5v1a4 4 0 0 0 3 3.87V17a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3v-.13A4 4 0 0 0 21 13v-1a5 5 0 0 0-5-5V6a4 4 0 0 0-4-4z"/>
      <path d="M9 12v2"/><path d="M15 12v2"/><path d="M12 9v5"/>
    </template>
    <!-- Check / Success -->
    <template v-else-if="name === 'check'">
      <polyline points="20 6 9 17 4 12"/>
    </template>
    <!-- Clock / Pending -->
    <template v-else-if="name === 'clock'">
      <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
    </template>
    <!-- Help / Question -->
    <template v-else-if="name === 'help'">
      <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/>
    </template>
    <!-- Shield / Approval -->
    <template v-else-if="name === 'shield'">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
    </template>
    <!-- Code / Brackets -->
    <template v-else-if="name === 'code'">
      <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
    </template>
    <!-- List / Menu -->
    <template v-else-if="name === 'list'">
      <line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/>
    </template>
    <!-- Layers / Stack / Context -->
    <template v-else-if="name === 'layers'">
      <polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/>
    </template>
    <!-- Eye / Show -->
    <template v-else-if="name === 'eye'">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
    </template>
    <!-- Eye Off / Hide -->
    <template v-else-if="name === 'eye-off'">
      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
      <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
      <line x1="1" y1="1" x2="23" y2="23"/>
    </template>
    <!-- Bug -->
    <template v-else-if="name === 'bug'">
      <rect x="8" y="2" width="8" height="4" rx="1" ry="1"/>
      <path d="M20 12h-3a5 5 0 0 1-5 5 5 5 0 0 1-5-5H4"/>
      <path d="M4 8h16"/>
      <path d="M12 2v7"/>
      <path d="M9 17l-3 4"/>
      <path d="M15 17l3 4"/>
    </template>
    <!-- Check Circle -->
    <template v-else-if="name === 'check-circle'">
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
      <polyline points="22 4 12 14.01 9 11.01"/>
    </template>
    <template v-else>
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
      <line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="15" y2="17"/>
    </template>
  </svg>
</template>

<script setup>
defineProps({
  name: { type: String, required: true },
  size: { type: Number, default: 16 },
})
</script>

<style scoped>
.svg-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
}
</style>
