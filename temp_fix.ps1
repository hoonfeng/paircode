$files = @(
    "F:\syproject\GWui\app\app_animation.go",
    "F:\syproject\GWui\app\app_ellipsis.go",
    "F:\syproject\GWui\app\app_pseudo.go"
)

$utf8NoBom = New-Object System.Text.UTF8Encoding $false

foreach ($f in $files) {
    $content = Get-Content $f -Encoding UTF8
    Write-Host "=== $f ==="
    Write-Host "First line: " $content[0]
    
    if ($content[0] -match '//go:build') {
        Write-Host "  Already has build tag, skipping"
        continue
    }
    
    $newContent = @("//go:build skia", "") + $content
    $writer = New-Object System.IO.StreamWriter($f, $false, $utf8NoBom)
    try {
        $newContent | ForEach-Object { $writer.WriteLine($_) }
    } finally {
        $writer.Close()
    }
    Write-Host "  Added //go:build skia tag, done"
}
