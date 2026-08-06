Get-Process | Where-Object { $_.MainWindowTitle -like '*PairCode*' } | ForEach-Object {
  Write-Output ("pid={0} hwnd=0x{1:X} title='{2}'" -f $_.Id, $_.MainWindowHandle.ToInt64(), $_.MainWindowTitle)
}
