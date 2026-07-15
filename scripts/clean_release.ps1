# Clean up tesseract training tools from release package
$keep = @('tesseract.exe','tesseract-uninstall.exe','winpath.exe')
$dir = 'release/PairCode/bin/tesseract'
Get-ChildItem "$dir/*.exe" | Where-Object { $keep -notcontains $_.Name } | ForEach-Object {
    Remove-Item $_.FullName -Force
    Write-Host "removed $($_.Name)"
}
if (Test-Path "$dir/doc") {
    Remove-Item "$dir/doc" -Recurse -Force
    Write-Host "removed doc/"
}
# Size check
$total = (Get-ChildItem -Recurse release/PairCode | Measure-Object -Property Length -Sum).Sum
Write-Host "Release package: $([math]::Round($total/1MB,1)) MB"
