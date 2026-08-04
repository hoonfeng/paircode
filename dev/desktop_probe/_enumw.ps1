Add-Type @"
using System;
using System.Text;
using System.Collections.Generic;
using System.Runtime.InteropServices;
public class EW {
  public delegate bool EnumProc(IntPtr hWnd, IntPtr lParam);
  [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc cb, IntPtr lParam);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
  [DllImport("user32.dll")] public static extern int GetWindowTextW(IntPtr h, StringBuilder sb, int max);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  public struct RECT { public int L, T, R, B; }
}
"@
$found = New-Object System.Collections.Generic.List[string]
$cb = [EW+EnumProc]{
  param($h, $l)
  $pid2 = 0
  [EW]::GetWindowThreadProcessId($h, [ref]$pid2) | Out-Null
  if ($pid2 -eq 48548) {
    $sb = New-Object System.Text.StringBuilder 256
    [EW]::GetWindowTextW($h, $sb, 256) | Out-Null
    $r = New-Object EW+RECT
    [EW]::GetWindowRect($h, [ref]$r) | Out-Null
    $vis = [EW]::IsWindowVisible($h)
    $found.Add("hwnd=$h pid=$pid2 vis=$vis title='$($sb.ToString())' rect=$($r.L),$($r.T),$($r.R),$($r.B)")
  }
  return $true
}
[EW]::EnumWindows($cb, [IntPtr]::Zero) | Out-Null
$found | ForEach-Object { Write-Output $_ }
if ($found.Count -eq 0) { Write-Output "no windows for pid 48548" }
