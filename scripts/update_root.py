#!/usr/bin/env python3
import os

file_path = os.path.join(
    os.path.dirname(__file__), "..", "cmd", "msgvault", "cmd", "root.go"
)
file_path = os.path.abspath(file_path)

with open(file_path, "r") as f:
    content = f.read()

content = content.replace('Use:   "msgvault"', 'Use:   "hacienda"')
content = content.replace(
    'Short: "Offline email archive tool"',
    'Short: "Offline email archive and intelligence tool"',
)
content = content.replace(
    "msgvault is an offline email archive tool",
    "hacienda is an offline email archive and intelligence tool",
)
content = content.replace(
    "email data locally with full-text search capabilities",
    "email data locally with full-text search, semantic search, and PII-filtered AI access",
)
content = content.replace(
    "To use msgvault, you need a Google Cloud OAuth credential:",
    "To use Hacienda, you need a Google Cloud OAuth credential:",
)
content = content.replace(
    "https://msgvault.io/guides/oauth-setup/", "https://hacienda.io/guides/oauth-setup/"
)
content = content.replace(
    ".msgvault/client_secret.json", ".hacienda/client_secret.json"
)
content = content.replace("msgvault remove-account", "hacienda remove-account")
content = content.replace("msgvault add-account", "hacienda add-account")
content = content.replace(".msgvault/config.toml", ".hacienda/config.toml")
content = content.replace("MSGVAULT_HOME", "HACIENDA_HOME")

with open(file_path, "w") as f:
    f.write(content)

print("Updated root.go")
