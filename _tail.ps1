$path = "F:\syproject\gou-ide\cmd\desktop\web-ui\dist\assets\index-1iBlH2r-.js"
$data = [System.IO.File]::ReadAllText($path)
$tail = $data.Substring($data.Length - 80)
Write-Host $tail
