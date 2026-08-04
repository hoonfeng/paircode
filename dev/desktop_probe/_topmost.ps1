$sig = @'
using System;
using System.Runtime.InteropServices;
using System.Text;
public class Win32 {
  [DllImport("user32.dll")] public static extern bool EnumWindows(EnumWindowsProc lpEnumFunc, IntPtr lParam);
  public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);
  [DllImport("user32.dll")] public static extern int GetWindowText(IntPtr hWnd, StringBuilder lpString, int nMaxCount);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr hWnd, IntPtr hWndInsertAfter, int X, int Y, int cx, int cy, uint uFlags);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
  public struct RECT { public int Left; public int Top; public int Right; public int Bottom; }
}
'@
Add-Type $sig

$top = [IntPtr]::new(-1)  # HWND_TOPMOST
$found = @()
$cb = [Win32+EnumWindowsProc]{
  param($h, $l)
  $sb = New-Object System.Text.StringBuilder 256
  [Win32]::GetWindowText($h, $sb, 256) | Out-Null
  $t = $sb.ToString()
  if ($t -like '*PairCode IDE*' -and [Win32]::IsWindowVisible($h)) {
    $r = New-Object Win32+RECT
    [Win32]::GetWindowRect($h, [ref]$r) | Out-Null
    Write-Output ("HWND=0x{0:X} title='{1}' rect=({2},{3})-({4},{5})" -f $h.ToInt64(), $t, $r.Left, $r.Top, $r.Right, $r.Bottom)
    [Win32]::SetWindowPos($h, $top, 0, 0, 0, 0, 0x0001 -bor 0x0002 -bor 0x0020) | Out-Null
    [Win32]::SetForegroundWindow($h) | Out-Null
  }
  return $true
}
[Win32]::EnumWindows($cb, [IntPtr]::Zero) | Out-Null
Write-Output "done"
