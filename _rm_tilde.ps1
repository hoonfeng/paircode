$path = "F:\syproject\gou-ide\companion.exe~"
if (Test-Path $path) {
    Remove-Item $path -Force -ErrorAction Stop
    Write-Host "DELETED"
} else {
    Write-Host "NOT_FOUND"
}
