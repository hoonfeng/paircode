Add-Type @"
using System;
using System.Runtime.InteropServices;
public class WinPos2 {
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT lpRect);
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
}
"@
$p = Get-Process -Name main -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowTitle -ne '' } | Select-Object -First 1
if ($p) {
    $r = New-Object WinPos2+RECT
    [WinPos2]::GetWindowRect($p.MainWindowHandle, [ref]$r) | Out-Null
    Write-Output ("RECT: " + $r.Left + "," + $r.Top + "," + $r.Right + "," + $r.Bottom + " title=" + $p.MainWindowTitle)
} else {
    Write-Output "not found"
}
