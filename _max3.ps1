Add-Type @"
using System;
using System.Runtime.InteropServices;
public class W32D {
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@
Stop-Process -Id 38312 -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500
$p = Get-Process -Id 59676 -ErrorAction SilentlyContinue
if ($p) {
  [W32D]::ShowWindow($p.MainWindowHandle, 3) | Out-Null
  Write-Output ("maximized new desktop pid=59676 hwnd=0x{0:X}" -f $p.MainWindowHandle.ToInt64())
} else {
  Write-Output "new desktop not found"
}
