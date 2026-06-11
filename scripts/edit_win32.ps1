param(
    [string]$filePath = "internal/window/win32/win32.go"
)

$file = Join-Path (Get-Location) $filePath
$content = Get-Content $file -Raw -Encoding UTF8

# The pattern includes the tab before procScreenToClient
$old = "`tprocScreenToClient         = user32.NewProc(`"ScreenToClient`")`r`n`tprocGetDpiForWindow"
$new = "`tprocScreenToClient         = user32.NewProc(`"ScreenToClient`")`r`n`tprocClientToScreen         = user32.NewProc(`"ClientToScreen`")`r`n`tprocGetDpiForWindow"

if ($content.Contains($old)) {
    $content = $content.Replace($old, $new)
    Set-Content $file -NoNewline -Value $content -Encoding UTF8
    Write-Host "OK: procClientToScreen added"
} else {
    Write-Host "FAIL: pattern not found"
    # Show hex dump of the first few chars around the match
    $idx = $content.IndexOf("procScreenToClient")
    if ($idx -ge 0) {
        $before = $content.Substring([Math]::Max(0, $idx - 10), $idx + 100 - [Math]::Max(0, $idx - 10))
        Write-Host "Context bytes:"
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($before)
        for ($i = 0; $i -lt $bytes.Length; $i++) {
            Write-Host ("  byte {0}: 0x{1:X2} = {2}" -f $i, $bytes[$i], [char]$bytes[$i])
        }
    }
}
