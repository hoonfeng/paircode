$errs = $null
$null = [System.Management.Automation.Language.Parser]::ParseFile('E:\paircode-master\scripts\deploy-pair-exe.ps1', [ref]$null, [ref]$errs)
if ($errs.Count -gt 0) { $errs | ForEach-Object { Write-Output ('  SYNTAX ERR: ' + $_.Message + ' @line ' + $_.Extent.StartLineNumber) } }
else { Write-Output '  [1] syntax OK' }

$s = -join ([char[]](0x672a,0x77e5,0x7684,0x5b50,0x8def,0x5f84))
Write-Output ('  [2] regex match positive: ' + (($s + ': run-stats') -match "\u672a\u77e5\u7684\u5b50\u8def\u5f84"))
Write-Output ('  [3] regex match negative: ' + ('{"startAt":0}' -match "\u672a\u77e5\u7684\u5b50\u8def\u5f84"))

function HttpProbe([string]$url) {
  try {
    $req = [System.Net.HttpWebRequest]::Create($url)
    $req.Timeout = 20000; $req.Method = "GET"
    $resp = $req.GetResponse()
    $sr = New-Object System.IO.StreamReader($resp.GetResponseStream())
    $b = $sr.ReadToEnd(); $sr.Close(); $c = [int]$resp.StatusCode; $resp.Close()
    return @{ code = $c; body = $b }
  } catch [System.Net.WebException] {
    $r = $_.Exception.Response
    if ($r) {
      $sr = New-Object System.IO.StreamReader($r.GetResponseStream())
      $b = $sr.ReadToEnd(); $sr.Close()
      return @{ code = [int]$r.StatusCode; body = $b }
    }
    return @{ code = 0; body = $_.Exception.Message }
  }
}

$a = HttpProbe 'http://127.0.0.1:9090/api/conversations/probe-nonexistent-0001/run-stats?workspaceRoot=E%3A%5Cpaircode-master'
Write-Output ('  [4] probe run-stats -> HTTP ' + $a.code)
Write-Output ('      body: ' + $a.body.Substring(0, [Math]::Min(90, $a.body.Length)))
$b = HttpProbe 'http://127.0.0.1:9090/api/definitely-not-a-route-xyz'
Write-Output ('  [5] probe bad route -> HTTP ' + $b.code)
Write-Output ('      body: ' + (($b.body -replace '\s+',' ')).Substring(0, [Math]::Min(90, $b.body.Length)))
$c = HttpProbe 'http://127.0.0.1:9090/'
Write-Output ('  [6] probe home -> HTTP ' + $c.code + ' (len ' + $c.body.Length + ')')
