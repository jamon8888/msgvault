# msgvault

[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Docs](https://img.shields.io/badge/Docs-msgvault.io-blue)](https://msgvault.io)
[![Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white)](https://discord.gg/fDnmxB8Wkq)

[Documentation](https://msgvault.io) · [Setup Guide](https://msgvault.io/guides/oauth-setup/) · [Interactive TUI](https://msgvault.io/usage/tui/)

> **Alpha software.** APIs, storage format, and CLI flags may change without notice. Back up your data.

Archive a lifetime of email. Analytics and search in milliseconds, entirely offline.

## Why msgvault?

Your messages are yours. Decades of correspondence, attachments, and history shouldn't be locked behind a web interface or an API. msgvault downloads a complete local copy and then everything runs offline. Search, analytics, and the MCP server all work against local data with no network access required.

Currently supports Gmail, Microsoft 365/Outlook, and IMAP sync, plus offline imports from MBOX exports and Apple Mail (.emlx) directories.

## Features

- **Full Gmail backup**: raw MIME, attachments, labels, and metadata
- **Microsoft 365 / Outlook.com**: OAuth2 + XOAUTH2 over IMAP, personal and organizational accounts
- **Generic IMAP sync**: archive mail from any standard IMAP server with password auth
- **MBOX / Apple Mail import**: import email from MBOX exports or Apple Mail (.emlx) directories
- **Interactive TUI**: drill-down analytics over your entire message history, powered by DuckDB over Parquet — connects to a remote `msgvault serve` instance or runs locally
- **Full-text search**: FTS5 with Gmail-like query syntax (`from:`, `has:attachment`, date ranges)
- **Attachment content search**: BM25 full-text search over extracted PDF, DOCX, and TXT content
- **Semantic search**: optional Ollama embeddings + DuckDB VSS for vector similarity search
- **MCP server**: 10 tools with automatic PII filtering for Claude Desktop and other AI agents
- **PII protection**: 3-pass filtering pipeline (structured PII + NER + legal patterns) on all MCP responses
- **DuckDB analytics**: millisecond aggregate queries across hundreds of thousands of messages in the TUI, CLI, and MCP server
- **Incremental sync**: Gmail History API picks up only new and changed messages
- **Multi-account**: archive several Gmail, Microsoft 365, and IMAP accounts in a single database
- **Resumable**: interrupted syncs resume from the last checkpoint
- **Content-addressed attachments**: deduplicated by SHA-256
- **Crypto-shredding**: AES-256-GCM encryption for RGPD (GDPR) right-to-be-erasure compliance
- **Legal Vault**: SMTP journaling server for email ingestion with crypto-shredding

## Installation

**macOS / Linux:**
```bash
curl -fsSL https://msgvault.io/install.sh | bash
```

**Windows (PowerShell):**
```powershell
powershell -ExecutionPolicy ByPass -c "irm https://msgvault.io/install.ps1 | iex"
```

The installer detects your OS and architecture, downloads the latest release from [GitHub Releases](https://github.com/wesm/msgvault/releases), verifies the SHA-256 checksum, and installs the binary. You can review the script ([bash](https://msgvault.io/install.sh), [PowerShell](https://msgvault.io/install.ps1)) before running, or download a release binary directly from GitHub.

To build from source instead (requires **Go 1.25+** and a C/C++ compiler for CGO and to statically link DuckDB):

```bash
git clone https://github.com/wesm/msgvault.git
cd msgvault
make install
```

**Conda-Forge:**

You can install msgvault [from conda-forge](https://prefix.dev/channels/conda-forge/packages/msgvault) using Pixi or Conda:

```bash
pixi global install msgvault
conda install -c conda-forge msgvault
```

## Quick Start

> **Prerequisites:** You need a Google Cloud OAuth credential before adding an account.
> Follow the **[OAuth Setup Guide](https://msgvault.io/guides/oauth-setup/)** to create one (~5 minutes).

```bash
msgvault init-db
msgvault add-account you@gmail.com          # opens browser for OAuth
msgvault sync-full you@gmail.com --limit 100
msgvault tui
```

## Commands

| Command | Description |
|---------|-------------|
| `init-db` | Create the database |
| `add-account EMAIL` | Authorize a Gmail account (use `--headless` for servers) |
| `add-o365 EMAIL` | Add a Microsoft 365 / Outlook.com account via OAuth |
| `add-imap` | Add a generic IMAP account (username/password) |
| `sync-full EMAIL` | Full sync (`--limit N`, `--after`/`--before` for date ranges) |
| `sync EMAIL` | Sync only new/changed messages |
| `tui` | Launch the interactive TUI (`--account` to filter, `--local` to force local) |
| `search QUERY` | Search messages (`--account` to filter, `--json` for machine output) |
| `show-message ID` | View full message details (`--json` for machine output) |
| `mcp` | Start the MCP server for AI assistant integration |
| `serve` | Run daemon with scheduled sync and HTTP API for remote TUI |
| `stats` | Show archive statistics |
| `list-accounts` | List synced email accounts |
| `verify EMAIL` | Verify archive integrity against Gmail |
| `export-eml` | Export a message as `.eml` |
| `import-mbox` | Import email from an MBOX export or `.zip` of MBOX files |
| `import-emlx` | Import email from an Apple Mail directory tree |
| `extract-attachments` | Extract and index text from attachments for semantic search |
| `export-attachment` | Export a single attachment by SHA-256 content hash |
| `export-attachments` | Export all attachments from a message to a directory |
| `build-cache` | Rebuild the Parquet analytics cache |
| `update` | Update msgvault to the latest version |
| `update-account` | Update account settings (`--display-name`) |
| `setup` | Interactive first-run configuration wizard |
| `repair-encoding` | Fix UTF-8 encoding issues |
| `export-token` | Export OAuth token to a remote msgvault instance |
| `create-subset` | Create a smaller database for testing/demos |
| `serve-archive` | Run Legal Vault SMTP ingestion server |
| `list-senders` / `list-domains` / `list-labels` | Explore metadata |
| `list-deletions` / `show-deletion` / `delete-staged` | Manage staged deletions |

See the [CLI Reference](https://msgvault.io/cli-reference/) for full details.

## Importing from MBOX or Apple Mail

Import email from providers that offer MBOX exports or from a local Apple Mail data directory:

```bash
msgvault init-db
msgvault import-mbox you@example.com /path/to/export.mbox
msgvault import-mbox you@example.com /path/to/export.zip   # zip of MBOX files
msgvault import-emlx                                        # auto-discover Apple Mail accounts
msgvault import-emlx you@example.com ~/Library/Mail/V10     # explicit path
```

## Configuration

All data lives in `~/.msgvault/` by default (override with `MSGVAULT_HOME`).

```toml
# ~/.msgvault/config.toml
[oauth]
client_secrets = "/path/to/client_secret.json"

[microsoft]
client_id = "your-azure-app-client-id"
tenant_id = "common"  # optional, defaults to "common"

[sync]
rate_limit_qps = 5
```

See the [Configuration Guide](https://msgvault.io/configuration/) for all options.

### Multiple OAuth Apps (Google Workspace)

Some Google Workspace organizations require OAuth apps within their org.
To use multiple OAuth apps, add named apps to `config.toml`:

```toml
[oauth]
client_secrets = "/path/to/default_secret.json"   # for personal Gmail

[oauth.apps.acme]
client_secrets = "/path/to/acme_workspace_secret.json"
```

Then specify the app when adding accounts:

```bash
msgvault add-account you@acme.com --oauth-app acme
msgvault add-account personal@gmail.com              # uses default
```

To switch an existing account to a different OAuth app:

```bash
msgvault add-account you@acme.com --oauth-app acme   # re-authorizes
```

## MCP Server

msgvault includes an MCP server that lets AI assistants search, analyze, and read your archived messages. Connect it to Claude Desktop or any MCP-capable agent and query your full message history conversationally.

All MCP responses are automatically PII-filtered through a 3-pass pipeline (structured PII + named entity recognition + legal pattern detection) to prevent leakage of sensitive data.

10 tools are available: `search_messages`, `get_message`, `get_attachment`, `export_attachment`, `list_messages`, `get_stats`, `aggregate`, `stage_deletion`, `search_attachments`, `extract_attachment`.

See the [MCP documentation](https://msgvault.io/usage/chat/) for setup instructions.

## Attachment Content Search

msgvault can extract text from PDF, DOCX, and TXT attachments and index it for full-text (BM25) or semantic (Ollama) search:

```bash
# Extract and index text from all unprocessed attachments
msgvault extract-attachments

# Re-process already indexed attachments
msgvault extract-attachments --reprocess

# Limit to 50 attachments, PDF only
msgvault extract-attachments --limit 50 --format pdf

# Search attachment content via MCP (BM25 by default)
# Or enable Ollama embeddings for semantic search:
```

```toml
[embedding]
enabled = true
provider = "ollama"
model = "nomic-embed-text"
ollama_url = "http://localhost:11434"
```

See [docs/attachment-search.md](docs/attachment-search.md) for full details.

## PII Protection

All MCP responses pass through a 3-pass PII filtering pipeline:

1. **Structured PII** (wuming): email addresses, phone numbers, IBANs, credit cards, SSNs, NIRs
2. **Named Entity Recognition** (prose): persons, organizations, locations, money, dates
3. **Legal patterns** (regex): case numbers, bar references, jurisdiction-specific identifiers (FR, UK, US, DE)

PII is replaced with descriptive tags (e.g., `[EMAIL]`, `[PHONE]`, `[PERSON]`) to preserve context while protecting sensitive data.

See [docs/pii-filtering.md](docs/pii-filtering.md) for configuration and jurisdiction details.

## Legal Vault

msgvault includes a Legal Vault mode for organizations that need to journal and archive email via SMTP:

```bash
msgvault serve-archive --smtp-host mail.example.com --smtp-port 2525
```

Features:
- SMTP server for email ingestion (journaling mode)
- AES-256-GCM crypto-shredding for RGPD compliance
- Content-addressed storage with per-message encryption keys
- WORM (immutable storage) support via MinIO

See [docs/legal-vault.md](docs/legal-vault.md) for setup instructions.

## Daemon Mode (NAS/Server)

Run msgvault as a long-running daemon for scheduled syncs and remote access:

```bash
msgvault serve
```

Configure scheduled syncs in `config.toml`:

```toml
[[accounts]]
email = "you@gmail.com"
schedule = "0 2 * * *"   # 2am daily (cron)
enabled = true

[server]
api_port = 8080
bind_addr = "0.0.0.0"
api_key = "your-secret-key"
```

The TUI can connect to a remote server by configuring `[remote].url`. Use `--local` to force local database when remote is configured. See [docs/api.md](docs/api.md) for the HTTP API reference.

## Documentation

- [Email Provider Configuration](docs/email-provider-configuration.md): Gmail, Microsoft 365, IMAP setup
- [PII Filtering](docs/pii-filtering.md): PII protection for MCP responses
- [Attachment Search](docs/attachment-search.md): BM25 and semantic search for attachments
- [MCP Tools](docs/mcp-tools.md): All 10 MCP tools with parameters and examples
- [Legal Vault](docs/legal-vault.md): SMTP ingestion and crypto-shredding
- [HTTP API](docs/api.md): REST API reference for serve mode
- [Setup Guide](https://msgvault.io/guides/oauth-setup/): OAuth, first sync, headless servers
- [Searching](https://msgvault.io/usage/searching/): query syntax and operators
- [Interactive TUI](https://msgvault.io/usage/tui/): keybindings, views, deletion staging
- [CLI Reference](https://msgvault.io/cli-reference/): all commands and flags
- [Multi-Account](https://msgvault.io/usage/multi-account/): managing multiple email accounts
- [Configuration](https://msgvault.io/configuration/): config file and environment variables
- [Architecture](https://msgvault.io/architecture/storage/): SQLite, Parquet, and attachment storage
- [Troubleshooting](https://msgvault.io/troubleshooting/): common issues and fixes
- [Development](https://msgvault.io/development/): contributing, testing, building

## Community

Join the [msgvault Discord](https://discord.gg/fDnmxB8Wkq) to ask questions, share feedback, report issues, and connect with other users.

## Development

```bash
git clone https://github.com/wesm/msgvault.git
cd msgvault
make install-hooks  # install pre-commit hook (requires prek)
make test           # run tests
make lint           # run linter (auto-fix)
make install        # build and install
```

Pre-commit hooks are managed by [prek](https://prek.j178.dev/) (`brew install prek`).

## License

MIT. See [LICENSE](LICENSE) for details.
