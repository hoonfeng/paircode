#!/usr/bin/env bash
# Generate Windows icon resources for cmd/companion before `go build`.
#
# WHY: on Windows, Go only links *.syso files found in the package directory;
#      companion.rc itself is NEVER compiled by Go. Since .gitignore ignores
#      *.ico and *.syso, these artifacts do not exist in a fresh checkout --
#      they must be regenerated before every build, otherwise pair.exe loses
#      its icon (and version info).
#
# Usage:  bash scripts/make-windows-rsrc.sh
# Env:    WINDRES=...  PYTHON=...  (optional overrides)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIR="$ROOT/cmd/companion"
WR="${WINDRES:-$ROOT/temp/gotool/mingw/mingw64/bin/windres.exe}"
PY="${PYTHON:-python}"

cd "$DIR"

echo "[1/2] Generate icon.ico from icon_256.png ..."
"$PY" - <<'PYEOF'
from PIL import Image
im = Image.open('icon_256.png').convert('RGBA')
im.save('icon.ico', sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
print('      icon.ico written (%dx%d source)' % im.size)
PYEOF

if [ ! -x "$WR" ] && [ ! -f "$WR" ]; then
  echo "ERROR: windres not found at: $WR" >&2
  echo "       set WINDRES=/path/to/windres.exe" >&2
  exit 2
fi

echo "[2/2] Compile companion.rc -> rsrc_windows_amd64.syso ..."
"$WR" -I. --input=companion.rc --output=rsrc_windows_amd64.syso \
      --output-format=coff --target=pe-x86-64

ls -l icon.ico rsrc_windows_amd64.syso
echo "OK: icon resources ready (go build will now link them automatically)"
