Add-Type @"
using System;
using System.Runtime.InteropServices;
public class W32b {
  [DllImport("user32.dll")] public static extern bool BringWindowToTop(IntPtr h);
  [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int cx, int cy, uint flags);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
}
"@
$h = (Get-Process -Id 48548).MainWindowHandle
[W32b]::BringWindowToTop($h) | Out-Null
[W32b]::SetWindowPos($h, [IntPtr](-1), 26, 26, 0, 0, 0x0001 -bor 0x0002) | Out-Null  # HWND_TOP + SWP_NOSIZE + SWP_NOMOVE
[W32b]::SetForegroundWindow($h) | Out-Null
Start-Sleep -Milliseconds 800
$fg = [W32b]::GetForegroundWindow()
Write-Output "fg=$fg target=$h equal=$($fg -eq $h)"
