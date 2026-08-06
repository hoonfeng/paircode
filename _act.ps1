Add-Type @"
using System;
using System.Runtime.InteropServices;
public class W32G {
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
}
"@
[W32G]::SetForegroundWindow([IntPtr]0x13A1017E) | Out-Null
Write-Output "activated"
