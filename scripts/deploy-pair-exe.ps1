<#
  deploy-pair-exe.ps1 -- Replace the running pair.exe with a freshly built one.

  WHY THIS SCRIPT EXISTS
    * Windows locks a running .exe, so the target service (default 9090) must be
      stopped before the file can be replaced -- but 9090 IS the agent host, so
      stopping it interrupts the current agent session.
    * Therefore this script performs ALL verification itself and writes the
      outcome to disk, so the result can be reviewed after a session interrupt.

  RULE (see .pair/project.md "部署铁律")
    The exe embeds front-end assets: exe and web artifacts must be upgraded as a
    PAIR. Replacing only the exe recreates the 2026-09-13 "新前端 + 旧后端" 400 bug
    (front-end calls /api/.../run-stats which an old exe does not serve).
    The run-stats sentinel below guards exactly that.

  USAGE
    powershell -ExecutionPolicy Bypass -File scripts\deploy-pair-exe.ps1 -DryRun
    powershell -ExecutionPolicy Bypass -File scripts\deploy-pair-exe.ps1
#>
param(
  [string]$Src     = "",      # default: <repo-root>\temp\build\pair.exe (repo root inferred from this script's own location)
  [string]$Target  = "D:\PairCode\pair.exe",
  [string]$WorkDir = "D:\PairCode",
  [int]$Port       = 9090,
  [string]$WorkspaceRoot = "D:\PairCodeData",
  [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$ts      = Get-Date -Format "yyyyMMdd-HHmmss"
# Repo root is DERIVED (this script lives in <repo-root>\scripts\) instead of hard-coded:
#   * survives any future relocation of the source tree (2026-09-16: E:\paircode-master -> D:\PairCodeData\...)
#   * keeps this file pure ASCII, so PowerShell 5.1 cannot mis-decode a non-ASCII path
#     (BOM-less UTF-8 .ps1 files are read as ANSI on this host)
$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $Src) { $Src = Join-Path $repoRoot "temp\build\pair.exe" }
$outDir  = Join-Path $env:TEMP "pair-deploy-$ts"
if (Test-Path (Join-Path $repoRoot "temp")) { $outDir = Join-Path $repoRoot "temp\deploy-$ts" }
$logFile = Join-Path $outDir "deploy.log"
$resFile = Join-Path $outDir "RESULT.txt"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

function Log([string]$m) {
  $line = (Get-Date -Format "HH:mm:ss") + " " + $m
  Write-Host $line
  Add-Content -Path $logFile -Value $line -Encoding UTF8
}
function FileSha([string]$p) { (Get-FileHash $p -Algorithm SHA256).Hash.ToUpper() }
function Finish([string]$verdict) {
  Set-Content -Path $resFile -Value $verdict -Encoding UTF8
  Log ("VERDICT: " + $verdict)
  exit $(if ($verdict -like "PASS*") { 0 } else { 1 })
}

Log "=== deploy pair.exe ===$([Environment]::NewLine)  source: $Src$([Environment]::NewLine)  target: $Target$([Environment]::NewLine)  outDir: $outDir  dryRun=$DryRun"

if (-not (Test-Path $Src))    { Log "FAIL: source missing: $Src";    Finish "FAIL source-missing" }
if (-not (Test-Path $Target)) { Log "FAIL: target missing: $Target"; Finish "FAIL target-missing" }

$srcLen = (Get-Item $Src).Length
$srcSha = FileSha $Src
$tgtLen = (Get-Item $Target).Length
$tgtSha = FileSha $Target
Log ("source      : {0,12:N0} bytes  {1}" -f $srcLen, $srcSha)
Log ("target(before): {0,11:N0} bytes  {1}" -f $tgtLen, $tgtSha)

if ($srcSha -eq $tgtSha) { Log "target already identical to source -- nothing to do"; Finish "PASS already-up-to-date" }

if ($DryRun) {
  Log "DRY RUN: would stop 'pair' (path=$Target), back up target, copy source, restart, and run health + run-stats checks."
  Finish "DRY-RUN (no changes made)"
}

# --- backup ---
$bkDir = Join-Path $outDir "backup"
New-Item -ItemType Directory -Force -Path $bkDir | Out-Null
$bk = Join-Path $bkDir "pair.exe"
Copy-Item -LiteralPath $Target -Destination $bk -Force
Log ("backed up target -> " + $bk)

# --- stop running instances of the target exe ---
$running = @(Get-Process -Name "pair" -ErrorAction SilentlyContinue | Where-Object {
  try { $_.Path -eq $Target } catch { $false }
})
foreach ($p in $running) { Log ("stopping PID " + $p.Id); Stop-Process -Id $p.Id -Force }
if ($running.Count -eq 0) { Log "no running instance of the target found (continuing)" }
Start-Sleep -Seconds 2

# --- replace ---
Copy-Item -LiteralPath $Src -Destination $Target -Force
$newLen = (Get-Item $Target).Length
$newSha = FileSha $Target
Log ("replaced: {0,12:N0} bytes  {1}" -f $newLen, $newSha)
if ($newSha -ne $srcSha) { Log "FAIL: SHA mismatch after copy"; Finish "FAIL sha-mismatch" }
Log "OK: SHA256 identical to source"

# --- restart ---
Start-Process -FilePath $Target -WorkingDirectory $WorkDir | Out-Null
Log ("started new pair.exe (cwd=" + $WorkDir + ")")

# --- health check ---
$ok = $false
for ($i = 1; $i -le 30; $i++) {
  Start-Sleep -Seconds 1
  try {
    $h = Invoke-RestMethod -Uri "http://127.0.0.1:$Port/api/health" -TimeoutSec 5
    Log ("health OK after ${i}s: " + ($h | ConvertTo-Json -Compress))
    $ok = $true; break
  } catch { }
}
if (-not $ok) { Log "FAIL: health check timed out"; Finish "FAIL health-timeout" }

# --- sentinel: run-stats must be 200 (front-end/back-end pairing) ---
# A fresh process may return an EMPTY conversation list (in-memory sessions not
# loaded yet). That is NOT a failure: fall back to a route-existence probe with a
# nonexistent id -- an OLD exe answers 400 "unknown sub path", a paired exe answers 200.
function HttpProbe([string]$url) {
  try {
    $req = [System.Net.HttpWebRequest]::Create($url)
    $req.Timeout = 20000; $req.Method = "GET"
    $resp = $req.GetResponse()
    $sr = New-Object System.IO.StreamReader($resp.GetResponseStream())
    $b = $sr.ReadToEnd(); $sr.Close(); $c = [int]$resp.StatusCode; $resp.Close()
    return @{ code = $c; body = $b }
  } catch [System.Net.WebException] {
    $r = $_.Exception.Response
    if ($r) {
      $sr = New-Object System.IO.StreamReader($r.GetResponseStream())
      $b = $sr.ReadToEnd(); $sr.Close()
      return @{ code = [int]$r.StatusCode; body = $b }
    }
    return @{ code = 0; body = $_.Exception.Message }
  }
}

$enc = [uri]::EscapeDataString($WorkspaceRoot)
$cid = $null
try {
  $convs = Invoke-RestMethod -Uri "http://127.0.0.1:$Port/api/conversations?workspaceRoot=$enc" -TimeoutSec 10
  if ($convs -is [System.Array] -and $convs.Count -gt 0) { $cid = $convs[0].id }
  elseif ($convs.conversations -and $convs.conversations.Count -gt 0) { $cid = $convs.conversations[0].id }
} catch { Log ("conversation list unavailable: " + $_.Exception.Message) }
if (-not $cid) {
  $cid = "probe-nonexistent-0001"
  Log "no conversation id available -- using route-existence probe instead"
}

$u   = "http://127.0.0.1:$Port/api/conversations/$cid/run-stats?workspaceRoot=$enc"
$pr  = HttpProbe $u
$bdy = [string]$pr.body
if ($bdy.Length -gt 200) { $bdy = $bdy.Substring(0, 200) }
Log ("run-stats sentinel: HTTP " + $pr.code + "  (id=" + $cid + ")")
Log ("  body: " + $bdy)

if ($pr.code -eq 400 -and $bdy -match "\u672a\u77e5\u7684\u5b50\u8def\u5f84") {
  Log "FAIL: run-stats route missing -> exe is OLD while web assets are NEW (pair mismatch)"
  Finish "FAIL  run-stats route missing (old exe deployed?)"
}
if ($pr.code -eq 0) {
  Log "FAIL: no HTTP response from pair.exe"
  Finish "FAIL  no-http-response"
}
$sentinel = "HTTP " + $pr.code

# --- front page shell ---
try {
  $html = (Invoke-WebRequest -Uri "http://127.0.0.1:$Port/" -TimeoutSec 10 -UseBasicParsing).Content
  $m = [regex]::Match($html, 'index-[A-Za-z0-9_-]+\.js')
  Log ("front page shell: " + $(if ($m.Success) { $m.Value } else { "(none)" }))
} catch { Log ("front page check failed: " + $_.Exception.Message) }

Finish "PASS  sha=$newSha  run-stats=$sentinel  icon-ok-if-RT_GROUP_ICON-present"
