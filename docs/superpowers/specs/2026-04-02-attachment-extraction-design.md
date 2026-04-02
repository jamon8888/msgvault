# Attachment Extraction + Semantic Search Design

## Overview

Add optional attachment content extraction and embedding-based semantic search to msgvault, enabling:
1. **RAG for AI assistants** - Search archived attachments via LLM to answer questions
2. **Compliance/Discovery** - Find specific documents/emails by content (legal, audit)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    msgvault                                  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌────────────────────────────────┐   │
│  │  Tabula         │───▶│  Text Chunks + Metadata        │   │
│  │  (extraction)  │    │  (PDF, DOCX, TXT, HTML, etc)   │   │
│  └─────────────────┘    └───────────────┬────────────────┘   │
│                                         │                    │
│                                         ▼                    │
│  ┌─────────────────┐    ┌────────────────────────────────┐   │
│  │  Ollama         │───▶│  Embeddings (nomic-embed-text) │   │
│  │  (local LLM)    │    │  1536-dim vectors              │   │
│  └─────────────────┘    └───────────────┬────────────────┘   │
│                                         │                    │
│                                         ▼                    │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  DuckDB VSS (Vector Store + HNSW Index)                 │ │
│  │  - attachment_vectors (id, message_id, embedding)      │ │
│  │  - attachment_text (id, chunks, metadata)              │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Extraction Engine (`internal/extractor/`)

**Purpose**: Extract text from attachments

**Supported formats (Phase 1)**:
- PDF - using `gen2brain/go-fitz` (MuPDF bindings, fast)
- DOCX - using `gomutex/godocx` (pure Go)
- TXT (simple text reading)

**Future phases**: HTML, ODT, EPUB

**Implementation**:
- `extractor.go` - Main extraction service
- `fitz.go` - PDF extraction via go-fitz
- `docx.go` - DOCX extraction via godocx
- `chunker.go` - Split text into chunks (8192-token chunks, 512-token overlap)

**Extraction modes**:
- `OnSync` - Extract during email sync (recent N days) - extraction step runs after sync completes
- `OnDemand` - Extract when explicitly requested
- `Hybrid` - Default: on-sync recent, on-demand older

### 2. Embedding Service (`internal/embedding/`)

**Purpose**: Generate embeddings from extracted text

**Implementation**:
- `ollama.go` - Ollama API client
- `service.go` - Embedding service interface

**Config**:
```toml
[embedding]
model = "nomic-embed-text"  # Default model
dimensions = 1536
ollama_url = "http://localhost:11434"
```

### 3. Vector Store (`internal/vector/`)

**Purpose**: Store and search embeddings using DuckDB VSS

**Note**: DuckDB VSS is a separate extension. Use `marcboeker/go-duckdb` with `INSTALL vss FROM community` and `LOAD vss`. Fall back to SQLite text search if VSS unavailable.

**Implementation**:
- `duckdb.go` - DuckDB VSS integration
- `store.go` - Vector storage service

**Tables**:
- `attachment_vectors` - (id, message_id, attachment_id, chunk_index, embedding)
- `attachment_text` - (id, message_id, attachment_id, chunk_text, metadata)

### 4. MCP Tools

**New tools**:
- `search_attachments` - Semantic search across attachment content
- `extract_attachment` - On-demand extraction for specific attachment

**Modified tools**:
- `search_messages` - Add option to include attachment content in search
- `get_message` - Include extracted attachment text in response

## Data Flow

### On-Sync Extraction (Hybrid mode)
1. Email sync completes - triggered by adding extraction step to sync command or background worker that runs after sync
2. For each new attachment (last N days):
   - Download attachment from storage
   - Extract text via go-fitz (PDF) or godocx (DOCX)
   - Split into chunks (8192-token chunks, 512-token overlap)
   - Generate embeddings via Ollama
   - Store in DuckDB VSS
3. Update message record with extraction status

### On-Demand Extraction
1. User requests search for attachment content
2. Check if attachment already extracted
3. If not:
   - Download + extract + embed in background
   - Return "processing" status
4. Execute semantic search (via MCP tool search_attachments)
5. Return results with source snippets

## Configuration

```toml
[extraction]
enabled = true           # Enable/disable feature
mode = "hybrid"          # on_sync, on_demand, hybrid
recent_days = 30         # Extract on-sync for last N days
formats = ["pdf", "docx", "txt"]  # Phase 1 formats

[embedding]
enabled = true           # Generate embeddings
model = "nomic-embed-text"
dimensions = 1536
ollama_url = "http://localhost:11434"

[vector]
store = "duckdb"        # duckdb (with VSS)
index_type = "hnsw"      # hnsw for fast KNN
```

## Error Handling

- **Extraction fails**: Log error, skip attachment, don't fail sync
- **Ollama unavailable**: Queue for retry, fallback to keyword search
- **DuckDB VSS fails**: Fall back to SQLite text search
- **Large attachments**: Stream extraction, chunk by chunk

## Testing

1. **Unit tests**: Extraction, embedding, vector store
2. **Integration tests**: Full pipeline with sample PDFs/DOCXs
3. **Performance tests**: Extraction speed, search latency
4. **Mock mode**: For CI without Ollama/DuckDB VSS

## Dependencies

```go
github.com/gen2brain/go-fitz   // PDF extraction (MuPDF)
github.com/gomutex/godocx       // DOCX extraction (pure Go)
github.com/marcboeker/go-duckdb // DuckDB + VSS
// Ollama: HTTP client (no Go package needed)
```

## Future Enhancements

- [ ] OCR for image-only PDFs (Tesseract)
- [ ] Table extraction from PDFs
- [ ] Multiple embedding models
- [ ] Re-index on content change