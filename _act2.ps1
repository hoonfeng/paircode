Add-Type @"
using System;
using System.Runtime.InteropServices;
public class Win32 {
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool MoveWindow(IntPtr hWnd, int X, int Y, int nWidth, int nHeight, bool bRepaint);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
  public struct RECT { public int Left, Top, Right, Bottom; }
}
"@
$hwnd = [IntPtr]0x13A1017E
$rect = New-Object Win32+RECT
[Win32]::GetWindowRect($hwnd, [ref]$rect) | Out-Null
Write-Host "before: $($rect.Left),$($rect.Top) $($rect.Right-$rect.Left)x$($rect.Bottom-$rect.Top)"
[Win32]::ShowWindow($hwnd, 9) | Out-Null   # SW_RESTORE
[Win32]::MoveWindow($hwnd, 100, 50, 1280, 800, $true) | Out-Null
[Win32]::SetForegroundWindow($hwnd) | Out-Null
Start-Sleep -Milliseconds 800
[Win32]::GetWindowRect($hwnd, [ref]$rect) | Out-Null
Write-Host "after: $($rect.Left),$($rect.Top) $($rect.Right-$rect.Left)x$($rect.Bottom-$rect.Top)"
