Add-Type @"
using System;
using System.Runtime.InteropServices;
public class TW {
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern bool SetWindowText(IntPtr h, string t);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern bool BringWindowToTop(IntPtr h);
}
"@
$h = [IntPtr]41749762
$mode = $args[0]
if ($mode -eq 'set') {
  [TW]::SetWindowText($h, 'PCIDESK-TEMP') | Out-Null
  [TW]::BringWindowToTop($h) | Out-Null
  [TW]::SetForegroundWindow($h) | Out-Null
  Write-Output 'title set to PCIDESK-TEMP'
} else {
  [TW]::SetWindowText($h, 'PairCode IDE') | Out-Null
  Write-Output 'title restored'
}
