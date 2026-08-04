Get-Process | Where-Object { $_.MainWindowTitle -ne '' } | Select-Object ProcessName, Id, MainWindowTitle | Format-Table -AutoSize | Out-String -Width 250
