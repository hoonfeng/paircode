const fs = require('fs');
let content = fs.readFileSync('cmd/companion/web-ui/src/components/GitPanel.vue', 'utf8');
const repl = [
  ["api.apiPost('/git/add', { files: [] })", "api.apiPost('/git/add', { files: [] }, gitParams())"],
  ["api.apiPost('/git/add', { files: [path] })", "api.apiPost('/git/add', { files: [path] }, gitParams())"],
  ["api.apiPost('/git/reset', { files: [path] })", "api.apiPost('/git/reset', { files: [path] }, gitParams())"],
  ["api.apiPost('/git/discard', { files: [path] })", "api.apiPost('/git/discard', { files: [path] }, gitParams())"],
  ["api.apiPost('/git/commit', { message: commitMsg.value, all: false })", "api.apiPost('/git/commit', { message: commitMsg.value, all: false }, gitParams())"],
  ["api.apiPost('/git/branch', { action: 'switch', name })", "api.apiPost('/git/branch', { action: 'switch', name }, gitParams())"],
  ["api.apiPost('/git/branch', { action: 'delete', name })", "api.apiPost('/git/branch', { action: 'delete', name }, gitParams())"],
  ["api.apiPost('/git/branch', { action: 'create-switch', name: newBranchName.value })", "api.apiPost('/git/branch', { action: 'create-switch', name: newBranchName.value }, gitParams())"],
  ["api.apiPost('/git/branch', { action: 'create', name: newBranchName.value })", "api.apiPost('/git/branch', { action: 'create', name: newBranchName.value }, gitParams())"],
  ["api.apiPost('/git/push', body)", "api.apiPost('/git/push', body, gitParams())"],
  ["api.apiPost('/git/pull', {})", "api.apiPost('/git/pull', {}, gitParams())"],
  ["api.apiPost('/git/stash', { action: 'push', message: stashMsg.value })", "api.apiPost('/git/stash', { action: 'push', message: stashMsg.value }, gitParams())"],
  ["api.apiPost('/git/stash', { action: 'pop', index })", "api.apiPost('/git/stash', { action: 'pop', index }, gitParams())"],
  ["api.apiPost('/git/stash', { action: 'drop', index })", "api.apiPost('/git/stash', { action: 'drop', index }, gitParams())"],
  ["api.apiPost('/git/ignore', { content: ignoreContent.value })", "api.apiPost('/git/ignore', { content: ignoreContent.value }, gitParams())"],
  ["api.apiGet('/git/ignore')", "api.apiGet('/git/ignore', gitParams())"],
  ["api.apiGet('/git/diff', { file: path, staged: staged ? 'true' : 'false' })", "api.apiGet('/git/diff', gitParams({ file: path, staged: staged ? 'true' : 'false' }))"],
];
let count = 0;
for (const [o, n] of repl) {
  const idx = content.indexOf(o);
  if (idx >= 0) { content = content.replace(o, n); count++; }
  else { console.log('NOT FOUND: ' + o.substring(0, 50)); }
}
fs.writeFileSync('cmd/companion/web-ui/src/components/GitPanel.vue', content, 'utf8');
console.log('Done: ' + count + ' replacements');
