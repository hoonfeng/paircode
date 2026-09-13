param([switch]$DryRun)
$ErrorActionPreference = 'Stop'
$src    = 'E:\paircode-master\temp\build\pair-cgo2.exe'
$dst    = 'D:\PairCode\pair.exe'
$ts     = Get-Date -Format 'yyyyMMdd-HHmmss'
$bakDir = "E:\paircode-master\temp\sync-D-PairCode\backup-manual-$ts"
$log    = 'E:\paircode-master\temp\build\switch-pair.log'
$url    = 'http://127.0.0.1:9090/api/health'

function Log([string]$m) {
  $line = "$(Get-Date -Format 'HH:mm:ss') $m"
  Write-Host $line
  Add-Content -Path $log -Value $line -Encoding UTF8
}

Log "════ 替换 pair.exe $(if($DryRun){'[DryRun 演练]'}else{'[实际执行]'}) ════"

if (-not (Test-Path $src)) { Log "❌ 源文件不存在: $src"; exit 1 }
$si = Get-Item $src
Log "源(新产物): $($si.Length) bytes, 修改 $($si.LastWriteTime)"
Log "  SHA256: $((Get-FileHash $src -Algorithm SHA256).Hash)"
if (Test-Path $dst) {
  $di = Get-Item $dst
  Log "目标(现役): $($di.Length) bytes, 修改 $($di.LastWriteTime)"
  Log "  SHA256: $((Get-FileHash $dst -Algorithm SHA256).Hash)"
} else { Log "⚠ 目标不存在: $dst" }

$procs = @(Get-Process pair -ErrorAction SilentlyContinue)
if ($procs.Count -gt 0) { Log "运行中: $(($procs | ForEach-Object { "PID $($_.Id) @$($_.StartTime)" }) -join ', ')" }
else { Log "当前无运行中的 pair 进程" }

try { $r = Invoke-WebRequest -Uri $url -TimeoutSec 5 -UseBasicParsing; Log "9090 健康: HTTP $($r.StatusCode) — $($r.Content)" }
catch { Log "9090 健康: 无响应（$($_.Exception.Message)）" }

if ($DryRun) { Log "════ DryRun 结束：未做任何修改 ════"; exit 0 }

if ($procs.Count -gt 0) {
  $procs | Stop-Process -Force
  Log "已发送停止信号: PID $(($procs.Id) -join ',')"
  Start-Sleep -Seconds 3
  if (Get-Process pair -ErrorAction SilentlyContinue) { Log "❌ 进程仍在运行（可能需管理员权限）；已中止，未做任何修改"; exit 2 }
  Log "✅ 进程已退出"
}

New-Item -ItemType Directory -Path $bakDir -Force | Out-Null
Copy-Item $dst (Join-Path $bakDir 'pair.exe') -Force
Log "✅ 已备份 → $bakDir\pair.exe"

Copy-Item $src $dst -Force
$h1 = (Get-FileHash $src -Algorithm SHA256).Hash
$h2 = (Get-FileHash $dst -Algorithm SHA256).Hash
if ($h1 -ne $h2) { Log "❌ 哈希不匹配，替换失败；可用 $bakDir\pair.exe 回滚"; exit 3 }
Log "✅ 替换完成，SHA256 一致: $h2"

Start-Process -FilePath $dst -WorkingDirectory 'D:\PairCode'
Log "已启动新 pair.exe"

$ok = $false
for ($i = 0; $i -lt 40; $i++) {
  Start-Sleep -Seconds 1
  try { $r = Invoke-WebRequest -Uri $url -TimeoutSec 3 -UseBasicParsing
        if ($r.StatusCode -eq 200) { Log "✅ 9090 已就绪（等待 $($i+1) 秒）: $($r.Content)"; $ok = $true; break } } catch {}
}
if (-not $ok) { Log "❌ 40 秒内 9090 未就绪；回滚: Copy-Item '$bakDir\pair.exe' '$dst' -Force" }
Log "════ 完成 ════"
