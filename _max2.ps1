Add-Type @"
using System;
using System.Runtime.InteropServices;
public class W32C {
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@
$p = Get-Process | Where-Object { $_.MainWindowTitle -eq 'PairCode IDE' -and $_.ProcessName -eq 'desktop' } | Select-Object -First 1
if ($p) {
  Write-Output ("found hwnd=0x{0:X}" -f $p.MainWindowHandle.ToInt64())
  Start-Sleep -Milliseconds 800
  [W32C]::ShowWindow($p.MainWindowHandle, 3) | Out-Null
  Write-Output "maximized"
} else {
  Write-Output "no desktop window found"
}
