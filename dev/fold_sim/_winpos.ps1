Add-Type @"
using System;
using System.Runtime.InteropServices;
public class WinPos {
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
}
"@
$p = Get-Process fold_sim -ErrorAction SilentlyContinue | Select-Object -First 1
if ($p) {
  $r = New-Object WinPos+RECT
  [WinPos]::GetWindowRect($p.MainWindowHandle, [ref]$r) | Out-Null
  Write-Output ("fold_sim rect: {0},{1} {2}x{3}" -f $r.Left, $r.Top, ($r.Right - $r.Left), ($r.Bottom - $r.Top))
  [WinPos]::SetForegroundWindow($p.MainWindowHandle) | Out-Null
} else { Write-Output "fold_sim not running" }
