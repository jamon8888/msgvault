#!/bin/bash
# Update root.go to Hacienda branding

FILE="/mnt/c/Users/NMarchitecte/Documents/msgvault/cmd/msgvault/cmd/root.go"

# Update Use, Short, Long
sed -i 's/Use:   "msgvault"/Use:   "hacienda"/' "$FILE"
sed -i 's/Short: "Offline email archive tool"/Short: "Offline email archive and intelligence tool"/' "$FILE"
sed -i 's/msgvault is an offline email archive tool/hacienda is an offline email archive and intelligence tool/' "$FILE"
sed -i 's/email data locally with full-text search capabilities/email data locally with full-text search, semantic search, and PII-filtered AI access/' "$FILE"

# Update oauthSetupHint
sed -i 's/To use msgvault/To use Hacienda/' "$FILE"
sed -i 's/https:\/\/msgvault.io\/guides\/oauth-setup/https:\/\/hacienda.io\/guides\/oauth-setup/' "$FILE"
sed -i 's/msgvault home directory/Hacienda home directory/' "$FILE"
sed -i 's/~\/.msgvault\/client_secret.json/~\/.hacienda\/client_secret.json/' "$FILE"

# Update remove-account example
sed -i 's/msgvault remove-account/hacienda remove-account/' "$FILE"
sed -i 's/msgvault add-account/hacienda add-account/' "$FILE"

# Update PersistentFlags defaults
sed -i "s/\~\/.msgvault\/config.toml/\~\/.hacienda\/config.toml/" "$FILE"
sed -i 's/MSGVAULT_HOME/HACIENDA_HOME/' "$FILE"

echo "root.go updated"