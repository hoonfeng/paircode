$procs = Get-Process | Where-Object { $_.MainWindowTitle -ne '' }
foreach ($p in $procs) {
    Write-Output ($p.ProcessName + ' | ' + $p.MainWindowTitle)
}
