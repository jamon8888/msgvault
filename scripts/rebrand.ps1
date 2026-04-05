# Update root.go to Hacienda branding
$file = "C:\Users\NMarchitecte\Documents\msgvault\cmd\msgvault\cmd\root.go"
$content = Get-Content $file -Raw

$content = $content -replace 'Use:   "msgvault"', 'Use:   "hacienda"'
$content = $content -replace 'Short: "Offline email archive tool"', 'Short: "Offline email archive and intelligence tool"'
$content = $content -replace 'msgvault is an offline email archive tool', 'hacienda is an offline email archive and intelligence tool'
$content = $content -replace 'email data locally with full-text search capabilities', 'email data locally with full-text search, semantic search, and PII-filtered AI access'
$content = $content -replace 'To use msgvault, you need a Google Cloud OAuth credential:', 'To use Hacienda, you need a Google Cloud OAuth credential:'
$content = $content -replace 'https://msgvault.io/guides/oauth-setup/', 'https://hacienda.io/guides/oauth-setup/'
$content = $content -replace 'msgvault home directory', 'Hacienda home directory'
$content = $content -replace '~\.msgvault/client_secret.json', '~/.hacienda/client_secret.json'
$content = $content -replace 'msgvault remove-account', 'hacienda remove-account'
$content = $content -replace 'msgvault add-account', 'hacienda add-account'
$content = $content -replace '~\/.msgvault/config.toml', '~/.hacienda/config.toml'
$content = $content -replace 'MSGVAULT_HOME', 'HACIENDA_HOME'

Set-Content $file -Value $content -NoNewline
Write-Host "Updated root.go"