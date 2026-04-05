# msgvault Agent Quickstart

You have access to `msgvault`, an offline Gmail archive tool. The user's email
archive is stored locally in a SQLite database with Parquet-based analytics.
Use the commands below to help the user set up, sync, explore, search, and
manage their email archive.

All data is stored in `~/.msgvault/` by default (override with `MSGVAULT_HOME`).

## Setup and account management

### Initialize the database

```bash
msgvault init-db
```

Creates the SQLite database and schema at `~/.msgvault/msgvault.db`. Safe to
run multiple times — tables are only created if they don't exist.

### Add a Gmail account

```bash
# Browser-based OAuth (default)
msgvault add-account user@gmail.com

# Headless / device code flow (for SSH sessions, no browser)
msgvault add-account user@gmail.com --headless
```

Requires `oauth.client_secrets` to be set in `~/.msgvault/config.toml` pointing
to a Google Cloud OAuth client secrets JSON file.

### Configuration

The config file is at `~/.msgvault/config.toml`:

```toml
[oauth]
client_secrets = "/path/to/client_secret.json"

[sync]
rate_limit_qps = 5
```

## Syncing email

### Full sync

Downloads all messages from Gmail. Supports resumption — if interrupted, just
run again to continue from the last checkpoint.

```bash
# Sync everything
msgvault sync-full user@gmail.com

# Sync a date range
msgvault sync-full user@gmail.com --after 2024-01-01 --before 2024-12-31

# Sync with a Gmail search query
msgvault sync-full user@gmail.com --query "from:someone@example.com"

# Limit message count (useful for testing)
msgvault sync-full user@gmail.com --limit 100

# Force fresh sync (skip resume from checkpoint)
msgvault sync-full user@gmail.com --noresume
```

Sync is **read-only** — it never modifies anything in Gmail.

### Incremental sync

Fetches only changes since the last sync using the Gmail History API. Much
faster than a full sync. Requires a prior full sync.

```bash
msgvault sync user@gmail.com
```

If Gmail's history has expired (~7 days), it will suggest running a full sync.

### Build the analytics cache

The TUI and aggregate queries use Parquet files for fast analytics. The cache
is built automatically when launching the TUI, but you can also build it
manually:

```bash
# Incremental update (only new messages)
msgvault build-cache

# Full rebuild from scratch
msgvault build-cache --full-rebuild
```

Cache files are stored in `~/.msgvault/analytics/`.

## Check archive status

```bash
msgvault stats
```

Shows message count, thread count, attachment count, label count, source
(account) count, and database size.

## Search messages

Use Gmail-like query syntax. All commands support `--json` for structured output.

```bash
# Full-text search
msgvault search "project update" --json

# Filter by sender
msgvault search from:alice@example.com

# Combine filters
msgvault search from:alice@example.com has:attachment after:2024-01-01

# Filter by label
msgvault search label:INBOX newer_than:30d

# Size filters
msgvault search larger:5M

# Pagination
msgvault search "quarterly report" --limit 100 --offset 50
```

### Supported search operators

| Operator      | Description                          | Example                    |
|---------------|--------------------------------------|----------------------------|
| `from:`       | Sender email address                 | `from:bob@example.com`     |
| `to:`         | Recipient email address              | `to:team@company.com`      |
| `cc:`         | CC recipient                         | `cc:manager@company.com`   |
| `bcc:`        | BCC recipient                        | `bcc:hr@company.com`       |
| `subject:`    | Subject text                         | `subject:meeting`          |
| `label:`      | Gmail label (or `l:`)                | `label:IMPORTANT`          |
| `has:`        | `has:attachment`                     | `has:attachment`           |
| `before:`     | Messages before date                 | `before:2024-06-01`        |
| `after:`      | Messages after date                  | `after:2024-01-01`         |
| `older_than:` | Relative date                        | `older_than:1y`            |
| `newer_than:` | Relative date                        | `newer_than:7d`            |
| `larger:`     | Minimum size                         | `larger:10M`               |
| `smaller:`    | Maximum size                         | `smaller:100K`             |

Bare words and `"quoted phrases"` perform full-text search across subject and body.

## View a single message

```bash
# By internal ID (from search results)
msgvault show-message 12345

# By Gmail message ID
msgvault show-message 18f0abc123def

# Structured output
msgvault show-message 12345 --json
```

The JSON output includes: `id`, `source_message_id`, `subject`, `from`, `to`,
`cc`, `bcc`, `sent_at`, `labels`, `body_text`, `body_html`, `attachments`.

## Aggregate analytics

These commands show top senders, domains, and labels ranked by message count.
All support `--limit` (`-n`), `--after`, `--before`, and `--json`.

```bash
# Top senders by email address
msgvault list-senders --limit 20 --json

# Top sender domains
msgvault list-domains --after 2024-01-01 --json

# Labels with message counts
msgvault list-labels --json
```

Each row in JSON output contains: `key`, `count`, `total_size`, `attachment_size`.

## Export a message as .eml

```bash
msgvault export-eml 12345
msgvault export-eml 12345 --output message.eml
```

Exports the raw MIME data as a standard `.eml` file compatible with most email
clients.

## Deletion management

Messages are staged for deletion in the TUI (select messages, press `d`).
Staged deletions are stored as manifests and must be explicitly executed.

```bash
# List all deletion batches (pending, in-progress, completed, failed)
msgvault list-deletions

# Show details of a specific batch
msgvault show-deletion <batch-id>

# Cancel a pending batch
msgvault cancel-deletion <batch-id>

# Execute pending deletions (permanent, fast — no recovery)
msgvault delete-staged --yes

# Execute a specific batch
msgvault delete-staged <batch-id>

# Move to trash instead (recoverable for 30 days, slower)
msgvault delete-staged --trash

# Dry run — show what would be deleted without doing it
msgvault delete-staged --dry-run

# Specify which account to delete from
msgvault delete-staged --account user@gmail.com
```

**Warning:** `delete-staged` without `--trash` permanently deletes messages from
Gmail. This is irreversible. Always verify with `--dry-run` first.

## Verify archive integrity

```bash
msgvault verify user@gmail.com
msgvault verify user@gmail.com --sample 500
```

Compares local message count with Gmail, checks raw MIME data integrity, and
samples random messages to verify they can be decompressed.

## Maintenance

```bash
# Repair invalid UTF-8 encoding in message text fields
msgvault repair-encoding

# Self-update to latest release
msgvault update
msgvault update --check   # Check only, don't install
msgvault update --yes     # Skip confirmation

# Show version info
msgvault version
```

## Attachment extraction and search

Extract text from PDF, DOCX, and TXT attachments for full-text (BM25) or semantic (Ollama) search.

```bash
# Extract and index text from all unprocessed attachments
msgvault extract-attachments

# Limit to 50 attachments
msgvault extract-attachments --limit 50

# Re-process already indexed attachments
msgvault extract-attachments --reprocess

# Process only PDF files
msgvault extract-attachments --format pdf

# Process multiple formats
msgvault extract-attachments --format pdf,docx
```

### Export attachments

```bash
# Export a single attachment by SHA-256 content hash
msgvault export-attachment <hash>
msgvault export-attachment <hash> -o file.pdf

# Export as JSON with base64-encoded data
msgvault export-attachment <hash> --json

# Export raw base64 to stdout
msgvault export-attachment <hash> --base64

# Export all attachments from a message
msgvault export-attachments <message-id>
msgvault export-attachments <message-id> -o /tmp/attachments
```

## NAS / Remote deployment

```bash
# Export OAuth token to a remote msgvault instance
msgvault export-token you@gmail.com --to https://nas.local:8080

# With explicit API key
msgvault export-token you@gmail.com --to https://nas.local:8080 --api-key your-key

# Allow HTTP for trusted networks (e.g., Tailscale)
msgvault export-token you@gmail.com --to http://nas.local:8080 --allow-insecure

# Uses MSGVAULT_REMOTE_URL and MSGVAULT_REMOTE_API_KEY env vars if set
```

## Account management

```bash
# Update account display name
msgvault update-account you@gmail.com --display-name "Work Account"
```

## Testing and demos

```bash
# Create a smaller database with the N most recent messages
msgvault create-subset -o /tmp/demo --rows 1000

# Use the subset
MSGVAULT_HOME=/tmp/demo msgvault tui
```

## Legal Vault (SMTP ingestion)

```bash
# Start SMTP server for email journaling
msgvault serve-archive --smtp-host mail.example.com --smtp-port 2525

# With MinIO storage
msgvault serve-archive --smtp-host mail.example.com --storage minio

# With WORM (immutable storage)
msgvault serve-archive --smtp-host mail.example.com --worm
```

## Interactive TUI

```bash
# Launch the TUI (auto-builds analytics cache if needed)
msgvault tui

# Filter by account
msgvault tui --account user@gmail.com

# Force local database (override remote config)
msgvault tui --local
```

### Remote mode

When `[remote].url` is configured in `config.toml`, the TUI connects to a remote
msgvault server instead of the local database. This is useful for accessing an
archive on a NAS or server from another machine.

```toml
[remote]
url = "http://nas.local:8080"
api_key = "your-api-key"
```

In remote mode, deletion staging and attachment export are disabled for safety.

### TUI keybindings

| Key              | Action                                         |
|------------------|-------------------------------------------------|
| `j`/`k`, `↑`/`↓`| Navigate rows                                  |
| `Enter`          | Drill down into selection                       |
| `Esc`/`Backspace`| Go back                                        |
| `Tab`            | Cycle view (Senders → Sender Names → To → To Names → Domains → Labels → Time) |
| `t`              | Jump to Time view (cycle granularity when already in Time) |
| `s`              | Cycle sort (Name → Count → Size)                |
| `r`              | Reverse sort direction                          |
| `t`              | Cycle time granularity (Year/Month/Day)         |
| `a`              | Filter by account                               |
| `f`              | Filter by attachments                           |
| `/`              | Search                                          |
| `Space`          | Toggle selection                                |
| `A`              | Select all visible                              |
| `x`              | Clear selection                                 |
| `d`              | Stage selected for deletion                     |
| `D`              | Stage all matching current filter for deletion  |
| `?`              | Help                                            |
| `q`              | Quit                                            |

## MCP Server

msgvault exposes 10 tools over stdio for AI assistants. All responses are automatically PII-filtered.

```bash
msgvault mcp
```

### Available MCP Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `search_messages` | Search emails with Gmail-like query syntax | `query`, `account`, `include_attachments`, `limit`, `offset` |
| `get_message` | Get full message details including body and attachments | `id` |
| `get_attachment` | Get attachment content by ID | `attachment_id` |
| `export_attachment` | Save attachment to local filesystem | `attachment_id`, `destination` |
| `list_messages` | List messages with optional filters | `account`, `from`, `to`, `label`, `after`, `before`, `has_attachment`, `limit`, `offset` |
| `get_stats` | Get archive overview (messages, size, attachments, accounts) | none |
| `aggregate` | Grouped statistics (senders, domains, labels, time) | `group_by`, `account`, `limit`, `after`, `before` |
| `stage_deletion` | Stage messages for deletion (does not delete immediately) | `account`, `query`, `from`, `domain`, `label`, `after`, `before`, `has_attachment` |
| `search_attachments` | Search attachment content (BM25 or semantic) | `query`, `limit`, `attachment_types` |
| `extract_attachment` | Extract and index text from an attachment | `attachment_id`, `force` |

### PII Filtering

All MCP tool responses pass through a 3-pass PII detection pipeline:
1. **Structured PII** (wuming): email, phone, IBAN, credit card, SSN, NIR
2. **Named Entity Recognition** (prose): PERSON, ORG, GPE, MONEY, DATE, etc.
3. **Legal patterns** (regex): case numbers, bar refs, jurisdiction-specific identifiers (FR, UK, US, DE)

PII is replaced with descriptive tags (e.g., `[EMAIL]`, `[PHONE]`, `[PERSON]`).

### Semantic Search Configuration

By default, attachment search uses BM25 (pure Go, no external dependencies). To enable semantic search with Ollama:

```toml
[embedding]
enabled = true
provider = "ollama"
model = "nomic-embed-text"
ollama_url = "http://localhost:11434"

[vector]
store = "duckdb"
index_type = "hnsw"
```

Then extract attachment text:
```bash
msgvault extract-attachments
```

## Typical agent workflow

1. **Check status**: `msgvault stats` — see what's in the archive.
2. **Search**: `msgvault search <query> --json` — find relevant messages.
3. **Read details**: `msgvault show-message <id> --json` — get full message content.
4. **Analyze**: `list-senders`, `list-domains`, `list-labels` with `--json` for patterns.
5. **Sync new mail**: `msgvault sync user@gmail.com` if archive is stale.

## Tips

- **Always use `--json`** for programmatic access. Available on: `search`,
  `show-message`, `list-senders`, `list-domains`, `list-labels`.
- Search results return an `id` field — use it with `show-message` for full content.
- Date filters use `YYYY-MM-DD` format.
- Relative date units: `d` (days), `w` (weeks), `m` (months), `y` (years).
- All query/search commands are read-only and never modify data.
- Deletion requires explicit staging (TUI) + execution (`delete-staged`).
- Use `--verbose` (`-v`) on any command for debug logging.
