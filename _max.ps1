Add-Type @"
using System;
using System.Runtime.InteropServices;
public class W32 {
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT r);
  public struct RECT { public int L, T, R, B; }
}
"@
$hwnd = [IntPtr]0x5A0864
$r = New-Object W32+RECT
[W32]::GetWindowRect($hwnd, [ref]$r) | Out-Null
Write-Output ("before rect: {0},{1}-{2},{3}" -f $r.L,$r.T,$r.R,$r.B)
[W32]::ShowWindow($hwnd, 3) | Out-Null
Start-Sleep -Milliseconds 1500
[W32]::GetWindowRect($hwnd, [ref]$r) | Out-Null
Write-Output ("after rect: {0},{1}-{2},{3}" -f $r.L,$r.T,$r.R,$r.B)
