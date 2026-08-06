Add-Type @"
using System;
using System.Runtime.InteropServices;
public class W32E {
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@
$p = Get-Process -Id 41544 -ErrorAction SilentlyContinue
if ($p) {
  [W32E]::ShowWindow($p.MainWindowHandle, 3) | Out-Null
  Write-Output ("maximized pid=41544 hwnd=0x{0:X}" -f $p.MainWindowHandle.ToInt64())
}
