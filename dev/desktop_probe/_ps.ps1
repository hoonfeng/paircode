Get-CimInstance Win32_Process -Filter "Name like '%desktop%' or Name like '%probe%'" | Select-Object ProcessId, Name, CommandLine | Format-List | Out-String -Width 300
