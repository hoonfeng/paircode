Add-Type @"
using System;
using System.Runtime.InteropServices;
public class W32F {
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint flags, uint dx, uint dy, uint data, UIntPtr extra);
}
"@
# desktop 窗口 (256,256)-(1550,1094)，内容区在窗口内
# 文件树项在窗口内 x≈60-320, y≈40-120
$hwnd = [IntPtr]0x750864
[W32F]::SetForegroundWindow($hwnd) | Out-Null
Start-Sleep -Milliseconds 500
# 点击文件树第一个文件（窗口内 x=100, y=85）→ 屏幕 (256+100, 256+85)=(356,341)
$sx = 256 + 100; $sy = 256 + 85
[W32F]::SetCursorPos($sx, $sy) | Out-Null
Start-Sleep -Milliseconds 200
[W32F]::mouse_event(0x0002, 0, 0, 0, [UIntPtr]::Zero)  # LEFTDOWN
[W32F]::mouse_event(0x0004, 0, 0, 0, [UIntPtr]::Zero)  # LEFTUP
Write-Output "clicked ($sx,$sy)"
