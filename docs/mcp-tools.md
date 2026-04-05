# MCP Tools Reference

msgvault exposes 10 tools over the Model Context Protocol (MCP) for AI assistants. All responses are automatically PII-filtered through a 3-pass pipeline.

## Starting the MCP Server

```bash
msgvault mcp
```

The server communicates over stdio (stdin/stdout). Configure your AI assistant to use it:

### Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "msgvault": {
      "command": "msgvault",
      "args": ["mcp"]
    }
  }
}
```

## Tool Reference

### 1. `search_messages`

Search emails using Gmail-like query syntax.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | Yes | — | Gmail-style search query (e.g., `from:alice subject:meeting after:2024-01-01`) |
| `account` | string | No | — | Filter by account email address |
| `include_attachments` | boolean | No | false | Also search attachment content using semantic vector search |
| `limit` | number | No | 20 | Maximum results to return |
| `offset` | number | No | 0 | Pagination offset |

**Example:**

```json
{
  "query": "from:bob@company.com has:attachment after:2024-01-01",
  "limit": 10
}
```

**Supported query operators:**

| Operator | Example | Description |
|----------|---------|-------------|
| `from:` | `from:alice@example.com` | Sender email |
| `to:` | `to:team@company.com` | Recipient email |
| `cc:` / `bcc:` | `cc:manager@company.com` | CC/BCC recipient |
| `subject:` | `subject:meeting` | Subject text |
| `label:` / `l:` | `label:INBOX` | Gmail label |
| `has:attachment` | `has:attachment` | Has attachments |
| `before:` / `after:` | `before:2024-06-01` | Date filter |
| `older_than:` / `newer_than:` | `newer_than:7d` | Relative date (d/w/m/y) |
| `larger:` / `smaller:` | `larger:5M` | Size filter (K/M/G) |

### 2. `get_message`

Get full message details including body text, recipients, labels, and attachments.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | number | Yes | Message ID (from search results) |

**Example:**

```json
{
  "id": 12345
}
```

**Response includes:** `id`, `source_message_id`, `subject`, `from`, `to`, `cc`, `bcc`, `sent_at`, `labels`, `body_text`, `body_html`, `attachments`.

### 3. `get_attachment`

Get attachment content by ID. Returns metadata as text and file content as an embedded resource blob.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `attachment_id` | number | Yes | Attachment ID (from get_message response) |

**Example:**

```json
{
  "attachment_id": 67890
}
```

### 4. `export_attachment`

Save an attachment to the local filesystem. Returns the saved file path.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `attachment_id` | number | Yes | — | Attachment ID |
| `destination` | string | No | `~/Downloads` | Directory to save to |

**Example:**

```json
{
  "attachment_id": 67890,
  "destination": "/tmp/exports"
}
```

### 5. `list_messages`

List messages with optional filters. Returns message summaries sorted by date.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `account` | string | No | — | Filter by account email |
| `from` | string | No | — | Filter by sender email |
| `to` | string | No | — | Filter by recipient email |
| `label` | string | No | — | Filter by Gmail label |
| `after` | string | No | — | Only messages after this date (YYYY-MM-DD) |
| `before` | string | No | — | Only messages before this date (YYYY-MM-DD) |
| `has_attachment` | boolean | No | — | Only messages with attachments |
| `limit` | number | No | 20 | Maximum results |
| `offset` | number | No | 0 | Pagination offset |

**Example:**

```json
{
  "from": "newsletter@example.com",
  "has_attachment": true,
  "after": "2024-01-01",
  "limit": 50
}
```

### 6. `get_stats`

Get archive overview: total messages, size, attachment count, and accounts.

**Parameters:** None

**Example:**

```json
{}
```

### 7. `aggregate`

Get grouped statistics (top senders, domains, labels, or message volume over time).

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `group_by` | string | Yes | — | Dimension: `sender`, `recipient`, `domain`, `label`, `time` |
| `account` | string | No | — | Filter by account email |
| `limit` | number | No | 50 | Maximum results |
| `after` | string | No | — | Date filter (YYYY-MM-DD) |
| `before` | string | No | — | Date filter (YYYY-MM-DD) |

**Example:**

```json
{
  "group_by": "sender",
  "limit": 20,
  "after": "2024-01-01"
}
```

### 8. `stage_deletion`

Stage messages for deletion. Does NOT delete immediately — use `delete-staged` CLI command to execute.

Use EITHER `query` OR structured filters, not both.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `account` | string | No | Filter by account email |
| `query` | string | No | Gmail-style search query (cannot combine with structured filters) |
| `from` | string | No | Filter by sender email |
| `domain` | string | No | Filter by sender domain (e.g., `linkedin.com`) |
| `label` | string | No | Filter by Gmail label (e.g., `CATEGORY_PROMOTIONS`) |
| `after` | string | No | Date filter (YYYY-MM-DD) |
| `before` | string | No | Date filter (YYYY-MM-DD) |
| `has_attachment` | boolean | No | Only messages with attachments |

**Example:**

```json
{
  "domain": "marketing.example.com",
  "before": "2023-01-01"
}
```

### 9. `search_attachments`

Search attachment content using semantic vector search (or BM25 if Ollama is not configured).

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | Yes | — | Search query (e.g., `find contracts about payment terms`) |
| `limit` | number | No | 10 | Maximum results |
| `attachment_types` | string | No | — | Filter by types: `pdf`, `docx`, `txt` (comma-separated) |

**Example:**

```json
{
  "query": "quarterly revenue report",
  "limit": 5,
  "attachment_types": "pdf"
}
```

**Note:** Requires running `msgvault extract-attachments` first to index attachment content.

### 10. `extract_attachment`

Extract and index text content from a specific attachment for semantic search.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `attachment_id` | number | Yes | — | Attachment ID to extract |
| `force` | boolean | No | false | Force re-extraction even if already extracted |

**Example:**

```json
{
  "attachment_id": 67890,
  "force": true
}
```

## PII Filtering

All tool responses are automatically filtered through a 3-pass PII detection pipeline:

1. **Structured PII** (wuming): email, phone, IBAN, credit card, SSN, NIR
2. **Named Entity Recognition** (prose): PERSON, ORG, GPE, MONEY, DATE, etc.
3. **Legal patterns** (regex): case numbers, bar references, jurisdiction-specific identifiers

PII is replaced with descriptive tags: `[EMAIL]`, `[PHONE]`, `[PERSON]`, `[MONEY]`, etc.

See [docs/pii-filtering.md](pii-filtering.md) for full details.

## Semantic Search Setup

By default, attachment search uses BM25 (keyword matching). To enable semantic search:

1. Install [Ollama](https://ollama.ai) and pull an embedding model:
   ```bash
   ollama pull nomic-embed-text
   ```

2. Configure in `~/.msgvault/config.toml`:
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

3. Extract attachment text:
   ```bash
   msgvault extract-attachments
   ```

4. Use `search_attachments` in your AI assistant conversations.
