# Attachment Content Search

msgvault can extract text from email attachments and index it for full-text (BM25) or semantic (Ollama) search, enabling you to search the contents of PDFs, DOCX files, and text files attached to your emails.

## Overview

```
Email Attachment → Text Extraction → Chunking → Indexing → Search
     PDF/DOCX/TXT      go-fitz/godocx    Overlapping    BM25 or Vector
                                          Chunks         Search
```

## Supported Formats

| Format | Extractor | Library | Platform |
|--------|-----------|---------|----------|
| PDF | `PDFExtractor` | go-fitz (libmupdf) | Linux, macOS |
| DOCX | `DOCXExtractor` | godocx | All platforms |
| TXT | `TXTExtractor` | Built-in | All platforms |

**Note:** PDF extraction via go-fitz requires CGO and libmupdf. It is not available on Windows. On Windows, PDF attachments are skipped during extraction.

## Quick Start

### BM25 Search (Default, No External Dependencies)

BM25 works out of the box with no additional configuration:

```bash
# Extract and index text from all unprocessed attachments
msgvault extract-attachments

# Check progress
msgvault stats
```

Once indexed, attachment content is searchable via the MCP `search_attachments` tool.

### Semantic Search (Ollama)

For semantic/vector search, configure Ollama embeddings:

1. Install [Ollama](https://ollama.ai) and pull an embedding model:
   ```bash
   ollama pull nomic-embed-text
   ```

2. Configure msgvault:
   ```toml
   [embedding]
   enabled = true
   provider = "ollama"
   model = "nomic-embed-text"
   dimensions = 768
   ollama_url = "http://localhost:11434"

   [vector]
   store = "duckdb"
   index_type = "hnsw"
   ```

3. Extract and index:
   ```bash
   msgvault extract-attachments
   ```

## CLI Commands

### `extract-attachments`

Extract text from attachments and index it for search.

```bash
# Extract all unprocessed attachments (default: 100 at a time)
msgvault extract-attachments

# Limit to 50 attachments
msgvault extract-attachments --limit 50

# Re-process already indexed attachments
msgvault extract-attachments --reprocess

# Process specific formats only
msgvault extract-attachments --format pdf
msgvault extract-attachments --format pdf,docx
msgvault extract-attachments --format txt
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--limit` | `-l` | int | 100 | Maximum attachments to process |
| `--reprocess` | | bool | false | Re-extract already indexed attachments |
| `--format` | | string | `pdf,docx,txt` | Comma-separated list of formats to process |

### `export-attachment`

Export a single attachment by its SHA-256 content hash.

```bash
# Export to stdout (binary)
msgvault export-attachment <hash>

# Export to file
msgvault export-attachment <hash> -o report.pdf

# Export as JSON with base64-encoded data
msgvault export-attachment <hash> --json

# Export raw base64 to stdout
msgvault export-attachment <hash> --base64
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output file path (use `-` for stdout) |
| `--json` | | Output as JSON with base64-encoded data |
| `--base64` | | Output raw base64 to stdout |

**Note:** `--json`, `--base64`, and `--output` are mutually exclusive.

### `export-attachments`

Export all attachments from a message as individual files.

```bash
# Export all attachments from a message to current directory
msgvault export-attachments <message-id>

# Export to specific directory
msgvault export-attachments <message-id> -o /tmp/attachments
```

Filenames are sanitized and deduplicated. Files are never overwritten (numeric suffix on conflict).

## How It Works

### Text Extraction

1. **PDF**: Uses go-fitz (libmupdf binding) to extract text from each page
2. **DOCX**: Uses godocx to walk document paragraphs and runs
3. **TXT**: Reads plain text directly

### Text Chunking

Extracted text is split into overlapping chunks for better search results:

- **Chunk size**: 1000 bytes
- **Overlap**: 200 bytes
- Overlap ensures search terms that span chunk boundaries are still found

### Indexing

**BM25 (default):**
- Chunks are stored in SQLite (`attachment_chunks` table)
- In-memory BM25 index is built with Okapi parameters: k1=1.5, b=0.6
- Smoothed IDF ensures non-zero scores even when terms appear in all documents
- Zero external dependencies, pure Go

**Vector Search (Ollama):**
- Each chunk is embedded via Ollama API
- Vectors stored in DuckDB with HNSW index for approximate nearest neighbor search
- Cosine distance used for similarity scoring
- Requires Ollama running locally

### Search

Attachment content search is available through MCP tools:

- **`search_attachments`**: Semantic search over attachment content
- **`search_messages`** with `include_attachments: true`: Combined message + attachment search
- **`extract_attachment`**: Extract and index a specific attachment

## BM25 vs Ollama: Which to Use?

| Feature | BM25 | Ollama |
|---------|------|--------|
| Setup | None required | Requires Ollama + embedding model |
| Speed | Instant | Depends on model size |
| Accuracy | Keyword matching | Semantic understanding |
| Dependencies | None | Ollama, DuckDB VSS |
| Windows support | Full | Not available |
| Best for | Exact term matching | Concept/fuzzy matching |

**Recommendation:** Start with BM25 (default). Add Ollama if you need semantic search capabilities like "find contracts about payment terms" without knowing the exact keywords.

## Configuration Reference

```toml
[extraction]
enabled = false           # Enable automatic extraction during sync
mode = "hybrid"           # "on_sync", "on_demand", "hybrid"
recent_days = 30          # Days to look back for on_sync mode
formats = ["pdf", "docx", "txt"]  # Supported formats

[embedding]
enabled = false           # Enable embedding generation
provider = "bm25"         # "bm25" (default) or "ollama"
model = "nomic-embed-text"  # Ollama model name
dimensions = 768          # Embedding dimension
ollama_url = "http://localhost:11434"

[vector]
store = "duckdb"          # Vector store backend
index_type = "hnsw"       # HNSW index for ANN search
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| PDF extraction skipped on Windows | go-fitz/libmupdf not available on Windows; use Linux/macOS or extract DOCX/TXT only |
| "Ollama connection refused" | Ensure Ollama is running: `ollama serve` |
| "Model not found" | Pull the embedding model: `ollama pull nomic-embed-text` |
| Slow extraction | Use `--limit` to process in batches; PDF extraction is CPU-intensive |
| No results from `search_attachments` | Run `extract-attachments` first; check that attachments exist with `msgvault stats` |
| DuckDB VSS not available | Not supported on Windows; use BM25 instead |
