# CLAUDE.md

## General Workflow

When a task involves multiple steps (e.g., implement + commit + PR), complete ALL steps in sequence without stopping. If creating a branch, committing, and opening a PR, finish the entire chain.

Always commit after every turn. Don't wait for the user to ask — if you made changes, commit them before responding. Do not ask "shall I commit?" or "want me to commit?" — just commit. Committing is not a destructive or risky action; it is the expected default after every change.

PR descriptions should be concise and changelog-oriented: what changed, why, and how to use it. Do not include test plans, design decisions, or implementation details — those belong in specs and commit messages.

## Project Overview

msgvault is an offline email archive tool that exports and stores email data locally with full-text search, semantic attachment search, and PII-safe MCP access. It supports Gmail, Microsoft 365/Outlook, and any IMAP server. The goal is to archive decades of email from multiple accounts, make it searchable (including attachment content), and eventually delete emails from the source once safely archived.

## Architecture (Go)

Single-binary Go application:

```
msgvault/
├── cmd/msgvault/            # CLI entrypoint
│   └── cmd/                 # Cobra commands
├── internal/                # Core packages
│   ├── tui/                 # Bubble Tea TUI
│   ├── query/               # DuckDB query engine over Parquet
│   ├── store/               # SQLite database access
│   ├── deletion/            # Deletion staging and manifest
│   ├── gmail/               # Gmail API client
│   ├── microsoft/           # Microsoft OAuth2 + XOAUTH2
│   ├── imap/                # Generic IMAP client (password + XOAUTH2)
│   ├── smtp/                # SMTP server (Legal Vault journaling)
│   ├── sync/                # Sync orchestration
│   ├── oauth/               # Google OAuth2 flows (browser + device)
│   ├── mime/                # MIME parsing
│   ├── mcp/                 # MCP server (AI assistant integration)
│   ├── pii/                 # PII filtering (wuming + NER + legal)
│   ├── search/              # BM25 + vector search for attachments
│   ├── extractor/           # PDF/DOCX/TXT text extraction
│   ├── embedding/           # Ollama embedding service
│   ├── vector/              # DuckDB VSS vector store
│   ├── crypto/              # AES-256-GCM crypto-shredding
│   ├── remote/              # Remote HTTP client (for serve mode)
│   ├── scheduler/           # Cron-based sync scheduler
│   ├── api/                 # HTTP API server (serve mode)
│   ├── config/              # Configuration management
│   ├── export/              # EML export utilities
│   ├── importer/            # MBOX/IMAP import logic
│   ├── applemail/           # Apple Mail .emlx parsing
│   ├── emlx/                # .emlx file format handling
│   ├── mbox/                # MBOX file parsing
│   ├── fileutil/            # File permission utilities
│   ├── textutil/            # Text processing utilities
│   ├── update/              # Self-update functionality
│   └── storage/             # Storage abstraction layer
│
├── go.mod                   # Go module
└── Makefile                 # Build targets
```

## Quick Commands

```bash
# Build
make build                    # Debug build
make build-release            # Release build (optimized)
make install                  # Install to ~/.local/bin or GOPATH
make test                     # Run tests
make lint                     # Run linter

# CLI usage
./msgvault init-db                                    # Initialize database
./msgvault add-account you@gmail.com                  # Browser OAuth
./msgvault add-account you@gmail.com --headless       # Device flow
./msgvault add-account you@acme.com --oauth-app acme  # Named OAuth app
./msgvault add-o365 you@company.com                   # Microsoft 365 / Outlook
./msgvault add-imap --host imap.example.com --username user@example.com  # Generic IMAP
./msgvault sync-full you@gmail.com --limit 100        # Sync with limit
./msgvault sync-full you@gmail.com --after 2024-01-01 # Sync date range
./msgvault sync-incremental you@gmail.com             # Incremental sync

# TUI and analytics
./msgvault tui                                        # Launch TUI
./msgvault tui --account you@gmail.com                # Filter by account
./msgvault tui --local                                # Force local (override remote config)
./msgvault build-cache                                # Build Parquet cache
./msgvault build-cache --full-rebuild                 # Full rebuild
./msgvault stats                                      # Show archive stats

# Apple Mail import
./msgvault import-emlx                                # Auto-discover accounts
./msgvault import-emlx ~/Library/Mail                 # Explicit mail directory
./msgvault import-emlx --account me@gmail.com         # Specific account(s)
./msgvault import-emlx /path/to/dir --identifier me@gmail.com  # Manual fallback

# Attachment extraction and search
./msgvault extract-attachments                        # Extract & index attachment text
./msgvault extract-attachments --limit 50 --reprocess # Re-extract with limit
./msgvault export-attachment <hash> -o file.pdf       # Export single attachment
./msgvault export-attachments <msg-id> -o /tmp/att    # Export all from message

# MCP server (PII-filtered)
./msgvault mcp                                        # Start MCP server

# Daemon mode (NAS/server deployment)
./msgvault serve                                      # Start HTTP API + scheduled syncs
./msgvault export-token you@gmail.com --to https://nas:8080  # Push token to remote

# Legal Vault (SMTP ingestion)
./msgvault serve-archive --smtp-host mail.example.com # SMTP journaling server

# Subset creation (testing/demos)
./msgvault create-subset -o /tmp/demo --rows 1000     # Create smaller DB

# Maintenance
./msgvault repair-encoding                            # Fix UTF-8 encoding issues
./msgvault update-account you@gmail.com --display-name "Work"  # Update account
```

## Key Files

### CLI (`cmd/msgvault/cmd/`)
- `root.go` - Cobra root command, config loading
- `syncfull.go` - Full sync command implementation
- `sync.go` - Incremental sync command
- `tui.go` - TUI command, cache auto-build
- `build_cache.go` - Parquet cache builder (DuckDB)
- `repair_encoding.go` - UTF-8 encoding repair
- `import_emlx.go` - Apple Mail .emlx import command
- `addaccount.go` - Gmail OAuth account setup
- `addo365.go` - Microsoft 365/Outlook OAuth setup
- `addimap.go` - Generic IMAP account setup
- `extract_attachments.go` - Extract & index attachment text
- `export_attachment.go` - Export single attachment by content hash
- `export_attachments.go` - Export all attachments from a message
- `export_token.go` - Export OAuth token to remote instance
- `update_account.go` - Update account settings
- `create_subset.go` - Create smaller DB subset for testing
- `serve_archive.go` - Legal Vault SMTP ingestion server
- `mcp.go` - MCP server command
- `serve.go` - HTTP API daemon command
- `deletions.go` - Deletion management (list/show/cancel/execute)

### Core (`internal/`)
- `tui/model.go` - Bubble Tea TUI model and update logic
- `tui/view.go` - View rendering with lipgloss styling
- `query/engine.go` - DuckDB query engine over Parquet files
- `query/sqlite.go` - SQLite query engine (fallback)
- `store/store.go` - SQLite database operations
- `store/schema.sql` - Core SQLite schema
- `store/schema_sqlite.sql` - FTS5 virtual table
- `deletion/manifest.go` - Deletion staging and manifest generation
- `gmail/client.go` - Gmail API client with rate limiting
- `microsoft/oauth.go` - Microsoft OAuth2 + XOAUTH2 manager
- `imap/client.go` - Generic IMAP client (implements gmail.API)
- `smtp/server.go` - SMTP server for email ingestion
- `oauth/oauth.go` - OAuth2 flows (browser + device)
- `sync/sync.go` - Sync orchestration, MIME parsing
- `mime/parse.go` - MIME message parsing
- `mcp/server.go` - MCP server with PII-filtered handlers
- `mcp/handlers.go` - MCP tool implementations
- `pii/filter.go` - 3-pass PII filtering (wuming + NER + legal)
- `pii/ner.go` - Named entity recognition via prose
- `pii/legal.go` - Jurisdiction-specific legal pattern detection
- `search/bm25.go` - BM25 full-text search over attachment chunks
- `search/parser.go` - Gmail-like query parser
- `search/vector.go` - Vector search adapter (DuckDB VSS)
- `extractor/extractor.go` - PDF/DOCX/TXT text extraction service
- `extractor/chunker.go` - Text chunking for search indexing
- `extractor/fitz.go` - PDF extraction via go-fitz (libmupdf)
- `extractor/docx.go` - DOCX extraction via godocx
- `embedding/service.go` - Ollama embedding service
- `vector/duckdb.go` - DuckDB VSS vector store with HNSW index
- `crypto/shredder.go` - AES-256-GCM crypto-shredding
- `crypto/keyhandler.go` - File-based key management
- `remote/store.go` - Remote HTTP client for serve mode
- `remote/engine.go` - Remote query engine implementation
- `scheduler/scheduler.go` - Cron-based sync scheduler
- `api/server.go` - HTTP API server endpoints
- `config/config.go` - Configuration management
- `export/eml.go` - EML export utilities
- `importer/mbox.go` - MBOX import logic
- `importer/imap.go` - IMAP import logic

### TUI Keybindings
- `j/k` or `↑/↓` - Navigate rows
- `Enter` - Drill down into selection
- `Esc` or `Backspace` - Go back
- `Tab` - Cycle views (Senders → Sender Names → Recipients → Recipient Names → Domains → Labels → Time)
- `s` - Cycle sort field (Name → Count → Size)
- `r` - Reverse sort direction
- `t` - Jump to Time view (cycle granularity when already in Time)
- `a` - Filter by account
- `f` - Filter by attachments
- `Space` - Toggle selection
- `A` - Select all visible
- `x` - Clear selection
- `d` - Stage selected for deletion
- `D` - Stage all messages matching current filter
- `/` - Search
- `?` - Help
- `q` - Quit

## Database Schema

Core tables:
- `sources` - Gmail accounts with history_id for incremental sync
- `conversations` - Gmail thread abstraction
- `messages` - Message metadata, foreign key to conversation
- `message_raw` - Raw MIME blob (zlib compressed)
- `labels` / `message_labels` - Gmail labels (many-to-many)
- `participants` / `message_recipients` - From/To/Cc/Bcc addresses
- `attachments` - Attachment metadata with content-hash deduplication
- `messages_fts` - FTS5 virtual table
- `sync_runs` / `sync_checkpoints` - Sync state for resumability

Schema files in `internal/store/`:
- `schema.sql` - Core SQLite schema
- `schema_sqlite.sql` - FTS5 virtual table

## Parquet Analytics

The TUI uses denormalized Parquet files for fast aggregate queries (~3000x faster than SQLite JOINs).

```
~/.msgvault/
├── msgvault.db              # SQLite: System of record
└── analytics/               # Parquet: Aggregate analytics
    ├── messages/year=*/     # Partitioned by year
    └── _last_sync.json      # Incremental sync state
```

**Workflow:**
1. Sync emails: `./msgvault sync-full you@gmail.com`
2. Launch TUI: `./msgvault tui` (auto-builds cache if needed)

**Parquet schema:**
- Denormalized: `from_email`, `from_domain`, `to_emails[]`, `labels[]`, etc.
- Partitioned by `year` for efficient time-range queries
- Compact: small fraction of SQLite size (excludes message bodies)

The TUI automatically builds/updates the Parquet cache on launch when new messages are detected.

## Implementation Status

### Completed
- **Gmail Sync**: Full/incremental sync, OAuth (browser + headless), rate limiting, resumable checkpoints
- **Microsoft 365/Outlook**: OAuth2 + XOAUTH2 over IMAP, personal/org auto-detection, scope correction
- **Generic IMAP**: Password auth, TLS/STARTTLS, any IMAP server
- **MIME Parsing**: Subject, body (text/HTML), attachments, charset detection
- **Parquet ETL**: DuckDB-based SQLite → Parquet export with incremental updates
- **Query Engine**: DuckDB over Parquet for fast aggregate analytics
- **TUI**: Full-featured TUI with drill-down navigation, search, selection, deletion staging
- **UTF-8 Repair**: Comprehensive encoding repair for all string fields
- **Deletion Execution**: Execute staged deletions via Gmail API (trash or permanent delete)
- **PII Filtering**: 3-pass pipeline (wuming + NER + legal) on all MCP responses
- **BM25 Attachment Search**: Pure Go BM25 over attachment text chunks (SQLite-backed)
- **Attachment Extraction**: PDF (go-fitz), DOCX (godocx), TXT text extraction and chunking
- **Vector Search**: Ollama embeddings + DuckDB VSS with HNSW index (Linux/macOS)
- **Crypto-Shredding**: AES-256-GCM encryption with per-message keys for RGPD compliance
- **SMTP Server**: Email ingestion server for Legal Vault journaling mode
- **MCP Server**: 10 tools with automatic PII filtering for AI assistant integration
- **Remote Mode**: HTTP API + remote query engine for NAS/server deployment
- **MBOX/Apple Mail Import**: Offline import from MBOX exports and .emlx directories

### Not Yet Implemented
- **Master key encryption**: FileKeyHandler EncryptKey/DecryptKey are pass-through (no master key wrapping)
- **Unshred operation**: Crypto-shredding decryption not yet implemented
- **Windows vector search**: DuckDB VSS not available on Windows
- **Windows PDF extraction**: go-fitz/libmupdf not available on Windows
- **Web UI**: Browser-based interface
- **Incremental sync for IMAP/Microsoft 365**: Full sync required each time
- **Headless OAuth for Microsoft 365**: Browser required

## Testing with Real Gmail Data

```bash
./msgvault init-db
./msgvault add-account you@gmail.com
./msgvault sync-full you@gmail.com --after 2024-12-01 --before 2024-12-15
./msgvault tui
```

Sync is **read-only** - no modifications to Gmail.

## Go Development

After making any Go code changes, always run `go fmt ./...` and `go vet ./...` before committing. Stage ALL resulting changes, including formatting-only files.

## Git Workflow

When committing changes, always stage ALL modified files (including formatting, generated files, and ancillary changes). Run `git diff` and `git status` before committing to ensure nothing is left unstaged.

## Code Style & Linting

All code must pass formatting and linting checks before commit. A pre-commit
hook is available via [prek](https://prek.j178.dev/) to enforce this
automatically:

```bash
make install-hooks             # Install pre-commit hook via prek
make test                      # Run tests
make fmt                       # Format code (go fmt)
make lint                      # Run linter (auto-fix)
make lint-ci                   # Run linter (CI, no auto-fix)
go vet ./...                   # Check for issues
```

**Standards:**
- Default gofmt configuration
- Use `error` return values, wrap with context using `fmt.Errorf`
- Table-driven tests

## Code Conventions

- Use Bubble Tea for TUI, lipgloss for styling
- DuckDB for Parquet queries, go-duckdb driver
- SQLite via marcboeker/go-duckdb for cache building, mattn/go-sqlite3 for store
- Context-based cancellation for long operations
- Route all DB operations through `Store` struct
- Charset detection via gogs/chardet, encoding via golang.org/x/text/encoding

## SQL Guidelines

- **Never use SELECT DISTINCT with JOINs** - Use EXISTS subqueries instead (becomes semi-joins)
- EXISTS is faster (stops at first match) and avoids duplicates at the source
- Example - instead of:
  ```sql
  SELECT DISTINCT m.id FROM messages m
  JOIN message_recipients mr ON mr.message_id = m.id
  WHERE mr.recipient_type = 'from' AND ...
  ```
  Use:
  ```sql
  SELECT m.id FROM messages m
  WHERE EXISTS (
      SELECT 1 FROM message_recipients mr
      WHERE mr.message_id = m.id AND mr.recipient_type = 'from' AND ...
  )
  ```

- **Never JOIN or scan `message_bodies` in list/aggregate/search queries** — this table is separated from `messages` specifically to keep the messages B-tree small for fast scans. Only access `message_bodies` via direct PK lookup (`WHERE message_id = ?`) when displaying a single message detail view. For text search, use FTS5 (`messages_fts`); if FTS is unavailable, search `subject`/`snippet` only.

## Configuration

All data defaults to `~/.msgvault/`:
- `~/.msgvault/config.toml` - Configuration file
- `~/.msgvault/msgvault.db` - SQLite database
- `~/.msgvault/attachments/` - Content-addressed attachment storage
- `~/.msgvault/tokens/` - OAuth tokens per account
- `~/.msgvault/analytics/` - Parquet cache files
- `~/.msgvault/keys/` - Crypto-shredding keys
- `~/.msgvault/vector.duckdb` - Vector search index (if enabled)

Override with `MSGVAULT_HOME` environment variable.

```toml
[data]
# data_dir = "~/custom/path"

[oauth]
client_secrets = "/path/to/client_secret.json"

# Named OAuth apps for Google Workspace orgs
# [oauth.apps.acme]
# client_secrets = "/path/to/acme_secret.json"

[microsoft]
client_id = "your-azure-app-client-id"
tenant_id = "common"  # optional, defaults to "common"

[sync]
rate_limit_qps = 5

[extraction]
enabled = false
mode = "hybrid"
recent_days = 30
formats = ["pdf", "docx", "txt"]

[embedding]
enabled = false
provider = "bm25"        # "bm25" (default) or "ollama"
model = "nomic-embed-text"
dimensions = 1536
ollama_url = "http://localhost:11434"

[vector]
store = "duckdb"
index_type = "hnsw"

[chat]
server = "http://localhost:11434"
model = "gpt-oss-128k"
max_results = 20

[server]
api_port = 8080
bind_addr = "127.0.0.1"
api_key = ""
allow_insecure = false
cors_origins = []
cors_credentials = false
cors_max_age = 0

[remote]
url = "http://nas.local:8080"
api_key = ""
allow_insecure = false

[[accounts]]                    # Scheduled syncs
email = "you@gmail.com"
schedule = "0 2 * * *"
enabled = true
```
