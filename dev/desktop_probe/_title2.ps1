Add-Type @"
using System;
using System.Text;
using System.Runtime.InteropServices;
public class F2 {
  public delegate bool EnumProc(IntPtr hWnd, IntPtr lParam);
  [DllImport("user32.dll")] public static extern bool EnumWindows(EnumProc cb, IntPtr lParam);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
  [DllImport("user32.dll")] public static extern int GetWindowTextW(IntPtr h, StringBuilder sb, int max);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern bool SetWindowText(IntPtr h, string t);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  public struct RECT { public int L, T, R, B; }
}
"@
$out = New-Object System.Collections.Generic.List[string]
$cb = [F2+EnumProc]{
  param($h, $l)
  $pid2 = 0
  [F2]::GetWindowThreadProcessId($h, [ref]$pid2) | Out-Null
  $proc = Get-Process -Id $pid2 -ErrorAction SilentlyContinue
  if ($proc -and $proc.ProcessName -eq 'dt2') {
    $sb = New-Object System.Text.StringBuilder 256
    [F2]::GetWindowTextW($h, $sb, 256) | Out-Null
    $vis = [F2]::IsWindowVisible($h)
    $r = New-Object F2+RECT
    [F2]::GetWindowRect($h, [ref]$r) | Out-Null
    $out.Add("hwnd=$h vis=$vis title='$($sb.ToString())' rect=$($r.L),$($r.T),$($r.R),$($r.B)")
    [F2]::SetWindowText($h, 'PCIDESK-DT2') | Out-Null
    $out.Add("  -> title set to PCIDESK-DT2")
  }
  return $true
}
[F2]::EnumWindows($cb, [IntPtr]::Zero) | Out-Null
$out | ForEach-Object { Write-Output $_ }
