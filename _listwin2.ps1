Get-Process | Where-Object { $_.MainWindowTitle -like '*PairCode*' -and $_.ProcessName -eq 'desktop' } | ForEach-Object {
  Write-Output ("pid={0} hwnd=0x{1:X}" -f $_.Id, $_.MainWindowHandle.ToInt64())
}
