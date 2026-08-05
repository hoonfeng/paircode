Get-Process | Where-Object { $_.MainWindowTitle -ne '' } | ForEach-Object { $_.ProcessName + ' | ' + $_.Id + ' | ' + $_.MainWindowTitle }
