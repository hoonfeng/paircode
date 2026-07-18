$path = "F:\syproject\gou-ide\cmd\desktop\web-ui\dist\assets\index-1iBlH2r-.js"
$data = [System.IO.File]::ReadAllText($path)

# Find hUe
$idx = $data.IndexOf("hUe")
if ($idx -ge 0) {
    $start = [Math]::Max(0, $idx - 30)
    $len = [Math]::Min(100, $data.Length - $start)
    Write-Host "hUe at $idx :"
    Write-Host $data.Substring($start, $len)
}

# Find .mount( near the end (last 200KB)
Write-Host "`n=== Last mount( call ==="
$endIdx = $data.Length
$searchStart = [Math]::Max(0, $endIdx - 200000)
$lastPart = $data.Substring($searchStart)
$mountIdx = $lastPart.LastIndexOf(".mount(")
if ($mountIdx -ge 0) {
    $ctxStart = [Math]::Max(0, $mountIdx - 20)
    $ctx = $lastPart.Substring($ctxStart, [Math]::Min(80, $lastPart.Length - $ctxStart))
    Write-Host $ctx
} else {
    Write-Host "No .mount( found in last 200KB"
}

# Find __vue_app__  near the end
Write-Host "`n=== __vue_app__ in last 200KB ==="
$vueIdx = $lastPart.IndexOf("__vue_app__")
if ($vueIdx -ge 0) {
    $ctxStart = [Math]::Max(0, $vueIdx - 40)
    $ctx = $lastPart.Substring($ctxStart, [Math]::Min(100, $lastPart.Length - $ctxStart))
    Write-Host $ctx
} else {
    Write-Host "No __vue_app__ found in last 200KB"
}

# Find createApp in last 200KB
Write-Host "`n=== createApp/mount call in last 200KB ==="
$pat = "createApp|mount\(|app\.mount"
$matches = [regex]::Matches($lastPart, $pat)
foreach ($m in $matches) {
    $ctxStart = [Math]::Max(0, $m.Index - 20)
    $ctx = $lastPart.Substring($ctxStart, [Math]::Min(60, $lastPart.Length - $ctxStart))
    Write-Host $ctx
}
