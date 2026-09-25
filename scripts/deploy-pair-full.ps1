# =====================================================================
# PairCode deploy: replace backend exe + sync UI artifacts to D:\PairCode
# Atomic: precheck -> backup -> stop -> write -> verify -> restart -> self-check
# On failure: automatic rollback to backup and restart of the old build.
# ASCII-only on purpose (avoid PS 5.1 encoding pitfalls).
#
# Origin: temp/deploy-20260913/deploy-fix.ps1 (run 2026-09-13, deploy OK),
# moved into scripts/ when temp/ was cleaned up. Paths that used to be
# hard-coded to temp/ and D:\PairCode are now parameters; deploy logic unchanged.
#
# Difference vs scripts/deploy-pair-exe.ps1:
#   deploy-pair-exe.ps1  -> replaces pair.exe only.
#   deploy-pair-full.ps1 -> replaces pair.exe AND syncs UI artifacts (shell +
#                           region bundles) from a manifest, with sha256
#                           precheck/write-verify and automatic rollback.
#
# Run detached from the agent session (the 9090 restart kills the session);
# see scripts/pair-switch/launch-switch.cmd for the ping-delay trick.
#
# Manifest format (JSON array, one entry per file):
#   [{ "src": "<abs src>", "dst": "<abs dst>", "kind": "shell|exe|ui|...",
#      "srcSha": "<UPPER-CASE SHA256>", "srcSize": <bytes> }]
# Precheck fails if any src is missing or its SHA256 != srcSha.
# =====================================================================
param(
    [switch]$DryRun,
    [string]$Root         = (Join-Path $env:TEMP 'paircode-deploy'),
    [string]$Manifest     = '',
    [string]$ExePath      = '',
    [string]$InstallDir   = '',
    [string]$BaseUrl      = 'http://127.0.0.1:9090',
    [string]$StatsUrl     = '',
    [string]$ExpectShell  = '',
    [string]$RollbackRoot = ''
)

$ErrorActionPreference = 'Continue'

if (-not $Manifest) { $Manifest = Join-Path $Root 'deploy-manifest.json' }
$Log        = Join-Path $Root 'deploy.log'
$ResultFile = Join-Path $Root 'RESULT.txt'
$BackupRoot = Join-Path $Root ('backup-' + (Get-Date -Format 'yyyyMMdd-HHmmss'))
$HealthUrl  = ($BaseUrl.TrimEnd('/')) + '/api/health'
$HomeUrl    = ($BaseUrl.TrimEnd('/')) + '/'

New-Item -ItemType Directory -Force -Path $Root | Out-Null

function Log([string]$m) {
    $line = (Get-Date -Format 'HH:mm:ss') + '  ' + $m
    Add-Content -Path $Log -Value $line -Encoding UTF8
    Write-Host $line
}
function Sha256([string]$p) { return (Get-FileHash -Path $p -Algorithm SHA256).Hash }

function Stop-Pair {
    $name = [System.IO.Path]::GetFileNameWithoutExtension($ExePath)
    $p = Get-Process -Name $name -ErrorAction SilentlyContinue
    if (-not $p) { Log ('no ' + $name + '.exe running'); return $true }
    Log ('stopping ' + $name + '.exe pid=' + (($p | ForEach-Object { $_.Id }) -join ','))
    try { $p | Stop-Process -Force -ErrorAction Stop } catch {
        Log ('FAIL: cannot stop ' + $name + '.exe (started as admin?) -> ' + $_.Exception.Message)
        return $false
    }
    Start-Sleep -Seconds 3
    if (Get-Process -Name $name -ErrorAction SilentlyContinue) { Log ('FAIL: ' + $name + '.exe still running'); return $false }
    Log ($name + '.exe stopped')
    return $true
}

function Start-Pair {
    $wd = $InstallDir
    if (-not $wd) { $wd = Split-Path $ExePath -Parent }
    Start-Process -FilePath $ExePath -WorkingDirectory $wd
    Log 'pair.exe start requested'
}

function Wait-Health([int]$maxSeconds) {
    for ($i = 0; $i -lt $maxSeconds; $i++) {
        Start-Sleep -Seconds 1
        try {
            $r = Invoke-WebRequest -Uri $HealthUrl -TimeoutSec 3 -UseBasicParsing
            if ($r.StatusCode -eq 200) { Log ('health OK after ' + $i + 's: ' + $r.Content); return $true }
        } catch { }
    }
    return $false
}

function Http-Code([string]$url) {
    try {
        $r = Invoke-WebRequest -Uri $url -TimeoutSec 8 -UseBasicParsing
        return [int]$r.StatusCode
    } catch {
        try { return [int]$_.Exception.Response.StatusCode.value__ } catch { return -1 }
    }
}

function Rollback([string]$why) {
    Log ('ROLLBACK triggered: ' + $why)
    Stop-Pair | Out-Null
    Start-Sleep -Seconds 2
    if (Test-Path $BackupRoot) {
        Copy-Item -Path (Join-Path $BackupRoot '*') -Destination $RollbackRoot -Recurse -Force -ErrorAction SilentlyContinue
        Log ('restored from ' + $BackupRoot + ' -> ' + $RollbackRoot)
    } else { Log 'no backup dir / rollback root - cannot restore' }
    Start-Pair
    $h = Wait-Health 40
    Log ('rollback health: ' + $h)
    Set-Content -Path $ResultFile -Value ('ROLLBACK: ' + $why) -Encoding UTF8
    exit 90
}

Log '================ deploy start ================'
Log ('manifest: ' + $Manifest)
Log ('backup  : ' + $BackupRoot)

if (-not (Test-Path $Manifest)) { Log 'FAIL: manifest missing'; exit 1 }
$rawJson = Get-Content $Manifest -Raw -Encoding UTF8
$parsed  = ConvertFrom-Json $rawJson
$items   = @()
foreach ($x in $parsed) { $items += $x }
if ($items.Count -eq 0) { Log 'FAIL: manifest has no items'; exit 1 }
Log ('manifest items: ' + $items.Count)

# Derive the paths that used to be hard-coded, unless explicitly given.
if (-not $ExePath) {
    $exeItem = $items | Where-Object { $_.dst -match '\.exe$' } | Select-Object -First 1
    if ($exeItem) { $ExePath = $exeItem.dst } else { $ExePath = $items[0].dst }
}
if (-not $InstallDir)   { $InstallDir = Split-Path $ExePath -Parent }
if (-not $RollbackRoot) { $RollbackRoot = [System.IO.Path]::GetPathRoot($items[0].dst) }
Log ('exe     : ' + $ExePath)
Log ('installd: ' + $InstallDir)
Log ('rollbk  : ' + $RollbackRoot)

foreach ($it in $items) {
    if (-not (Test-Path $it.src)) { Log ('FAIL precheck missing src: ' + $it.src); exit 1 }
    if ((Sha256 $it.src) -ne $it.srcSha) { Log ('FAIL precheck sha mismatch: ' + $it.src); exit 1 }
}
Log 'precheck OK (sources exist and match manifest hashes)'

New-Item -ItemType Directory -Force -Path $BackupRoot | Out-Null
$bkCount = 0
foreach ($it in $items) {
    if (Test-Path $it.dst) {
        # relative path inside the backup = dst minus its volume root (e.g. "D:\")
        $rel = $it.dst.Substring($RollbackRoot.Length)
        $b = Join-Path $BackupRoot $rel
        New-Item -ItemType Directory -Force -Path (Split-Path $b) | Out-Null
        Copy-Item -Path $it.dst -Destination $b -Force
        $bkCount++
    }
}
Log ('backup done: ' + $bkCount + ' files -> ' + $BackupRoot)

if ($DryRun) {
    Log 'DRY RUN: precheck + backup verified. No service stop, no writes.'
    Log ('would write ' + $items.Count + ' files incl. ' + $ExePath)
    foreach ($it in $items) {
        Log ('   [' + $it.kind + '] ' + (Split-Path $it.dst -Leaf) + '  <-  ' + (Split-Path $it.src -Leaf))
    }
    Set-Content -Path $ResultFile -Value 'DRYRUN OK' -Encoding UTF8
    exit 0
}
if (-not (Stop-Pair)) { Log 'ABORT: cannot stop service, nothing changed'; exit 2 }

$w = 0
foreach ($it in $items) {
    $dir = Split-Path $it.dst
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
    Copy-Item -Path $it.src -Destination $it.dst -Force
    if ((Sha256 $it.dst) -ne $it.srcSha) { Log ('FAIL write verify: ' + $it.dst); Rollback 'copy verification failed' }
    $w++
}
Log ('written and verified: ' + $w + ' files')

Start-Pair
if (-not (Wait-Health 45)) { Rollback 'health check failed after restart' }
Log 'health OK'

$code = -1
if ($StatsUrl) {
    $code = Http-Code $StatsUrl
    Log ('run-stats endpoint HTTP ' + $code + '  (400=old backend, 200=fixed)')
    if ($code -ne 200) { Rollback ('run-stats returned ' + $code + ' instead of 200') }
} else {
    Log 'run-stats check skipped (no -StatsUrl given)'
}

$shellOk = 'skipped'
if ($ExpectShell) {
    # NOTE (2026-09-25): was "$home = Invoke-WebRequest ..." -- $home is a
    # READ-ONLY automatic variable in Windows PowerShell 5.1, so the assignment
    # threw and the empty catch swallowed it -> this check always logged
    # "False". Renamed to $page and added retries (first request right after
    # restart may race the service warm-up).
    $shellOk = $false
    for ($i = 1; $i -le 3; $i++) {
        try {
            $page = Invoke-WebRequest -Uri $HomeUrl -TimeoutSec 8 -UseBasicParsing
            if ($page.Content -match [regex]::Escape($ExpectShell)) { $shellOk = $true; break }
        } catch { }
        if ($i -lt 3) { Start-Sleep -Seconds 2 }
    }
    Log ('front page references ' + $ExpectShell + ': ' + $shellOk)
} else {
    Log 'front-page shell check skipped (no -ExpectShell given)'
}

# Archive obsolete shell bundles: keep only the index-*.js that this deploy
# just wrote (older ones accumulate after every UI rebuild). Moved to the run
# root (NOT BackupRoot -- BackupRoot is replayed verbatim by Rollback).
$shellNames = @()
foreach ($it in $items) { if ($it.dst -match '\\assets\\index-[^\\]+\.js$') { $shellNames += (Split-Path $it.dst -Leaf) } }
if ($shellNames.Count -gt 0) {
    $webAssetsDir = Join-Path $InstallDir '.pair\assets\runtime\web\assets'
    if (Test-Path $webAssetsDir) {
        $obsoleteDir = Join-Path $Root 'obsolete-shell'
        $moved = 0
        foreach ($f in (Get-ChildItem -Path $webAssetsDir -Filter 'index-*.js' -ErrorAction SilentlyContinue)) {
            if ($shellNames -notcontains $f.Name) {
                New-Item -ItemType Directory -Force -Path $obsoleteDir | Out-Null
                Move-Item $f.FullName (Join-Path $obsoleteDir $f.Name) -Force
                $moved++
            }
        }
        if ($moved -gt 0) { Log ('obsolete shell bundles archived: ' + $moved + ' -> ' + $obsoleteDir) }
    }
}

Log '================ deploy DONE ================'
$summary = @()
$summary += 'OK'
if ($StatsUrl) { $summary += ('run-stats HTTP ' + $code) }
$summary += ('backup dir ' + $BackupRoot)
$summary += ('new shell referenced ' + $shellOk)
Set-Content -Path $ResultFile -Value ($summary -join "`n") -Encoding UTF8
exit 0
