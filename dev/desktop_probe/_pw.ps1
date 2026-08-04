Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Runtime.InteropServices;
public class PW {
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern IntPtr GetWindowDC(IntPtr h);
  [DllImport("user32.dll")] public static extern int ReleaseDC(IntPtr h, IntPtr dc);
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h, IntPtr dc, uint flags);
  [DllImport("gdi32.dll")] public static extern IntPtr CreateCompatibleDC(IntPtr dc);
  [DllImport("gdi32.dll")] public static extern IntPtr CreateCompatibleBitmap(IntPtr dc, int w, int h);
  [DllImport("gdi32.dll")] public static extern IntPtr SelectObject(IntPtr dc, IntPtr obj);
  [DllImport("gdi32.dll")] public static extern bool DeleteObject(IntPtr obj);
  [DllImport("gdi32.dll")] public static extern bool DeleteDC(IntPtr dc);
  [DllImport("gdi32.dll")] public static extern bool BitBlt(IntPtr dst, int x, int y, int w, int h, IntPtr src, int sx, int sy, uint rop);
  public struct RECT { public int L, T, R, B; }
}
"@
$h = [IntPtr]41749762
$r = New-Object PW+RECT
[PW]::GetWindowRect($h, [ref]$r) | Out-Null
$w = $r.R - $r.L; $ht = $r.B - $r.T
Write-Output "win $w x $ht"
$hdcWin = [PW]::GetWindowDC($h)
$hdcMem = [PW]::CreateCompatibleDC($hdcWin)
$hbmp = [PW]::CreateCompatibleBitmap($hdcWin, $w, $ht)
$old = [PW]::SelectObject($hdcMem, $hbmp)
$ok = [PW]::PrintWindow($h, $hdcMem, 0x00000002)  # PW_RENDERFULLCONTENT
Write-Output "printwindow=$ok"
# 保存
$bmp = [System.Drawing.Bitmap]::FromHbitmap($hbmp)
$bmp.Save("F:\syproject\gou-ide\dev\desktop_probe\pw_desktop.png", [System.Drawing.Imaging.ImageFormat]::Png)
Write-Output "saved"
[PW]::SelectObject($hdcMem, $old) | Out-Null
[PW]::DeleteObject($hbmp) | Out-Null
[PW]::DeleteDC($hdcMem) | Out-Null
[PW]::ReleaseDC($h, $hdcWin) | Out-Null
$bmp.Dispose()
