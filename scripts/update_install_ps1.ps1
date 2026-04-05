# Update install.ps1 to use jamon8888 fork
$content = Get-Content 'C:\Users\NMarchitecte\Documents\msgvault\scripts\install.ps1' -Raw
$content = $content -replace 'wesm/msgvault', 'jamon8888/msgvault'
Set-Content -Path 'C:\Users\NMarchitecte\Documents\msgvault\scripts\install.ps1' -Value $content -NoNewline
Write-Host "Updated install.ps1"