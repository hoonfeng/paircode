const fs = require('fs');
const path = require('path');
const content = fs.readFileSync(path.join(__dirname, 'cmd/companion/web-ui/dist/assets/index-C1XOhOtW.js'), 'utf8');
// Find "Cannot access" in the file to locate the problematic code
const idx = content.indexOf('Cannot access');
if (idx >= 0) {
  console.log('Found "Cannot access" at position:', idx);
  const start = Math.max(0, idx - 500);
  const end = Math.min(content.length, idx + 200);
  console.log('Context around error:');
  console.log(content.substring(start, end));
} else {
  console.log('Cannot access not found directly, searching for TDZ patterns...');
}

// Search for "d" variable initialization in the setup context
const setupIdx = content.indexOf('setup=');
if (setupIdx >= 0) {
  console.log('\nFound setup= at position:', setupIdx);
  console.log('Context:', content.substring(setupIdx, setupIdx + 300));
}
