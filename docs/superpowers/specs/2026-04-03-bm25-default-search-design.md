# Spec: BM25 Default + Ollama Optional for Attachment Search

## Problem

The current attachment search requires:
1. Ollama running locally (external dependency)
2. DuckDB VSS extension (CGO, build issues on Windows)
3. Vector embeddings for every chunk (slow, heavy)

This makes the feature unusable out-of-the-box on Windows.

## Solution

Replace the vector store with a unified `SearchStore` interface:
- **BM25** (default): Pure Go, zero dependencies, ~15MB binary
- **DuckDB VSS** (optional): For users who want Ollama semantic search

## Config

```toml
[embedding]
provider = "bm25"  # "bm25" (default) or "ollama"
model = "nomic-embed-text"  # only if provider = "ollama"
ollama_url = "http://localhost:11434"  # only if provider = "ollama"
```

## Interface

```go
type SearchStore interface {
    Index(ctx context.Context, id int64, messageID int64, attachmentID int64, chunkIndex int, text string) error
    Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
    Delete(ctx context.Context, attachmentID int64) error
    Close() error
}

type SearchResult struct {
    AttachmentID int64
    MessageID    int64
    ChunkIndex   int
    ChunkText    string
    Score        float64  // BM25: TF-IDF-like; VSS: cosine distance. Treat as relative, not absolute.
}
```

## Files

### New
- `internal/search/store.go` - SearchStore interface + SearchResult
- `internal/search/bm25.go` - BM25 implementation using `github.com/crawlab-team/bm25`
- `internal/search/bm25_test.go` - Tests

### Modified
- `internal/config/config.go` - Add `Provider` field to EmbeddingConfig
- `internal/mcp/server.go` - Initialize BM25 by default, Ollama+VSS if provider="ollama"
- `internal/mcp/handlers.go` - Replace VectorStore with SearchStore
- `internal/vector/duckdb.go` - Add SearchStore adapter wrapping existing methods (Index calls InsertVector+InsertText, Search calls existing Search, Delete removes by attachmentID)
- `cmd/msgvault/cmd/extract_attachments.go` - Use SearchStore instead of VectorStore

## BM25 Implementation Details

- **Storage**: SQLite-backed (reuse existing msgvault.db). Create `attachment_chunks` table with `(id, message_id, attachment_id, chunk_index, chunk_text)`.
- **Index**: BM25 index built on startup from the `attachment_chunks` table. Rebuild incrementally on new Index() calls.
- **Persistence**: No separate JSON file. Chunks live in SQLite, BM25 index rebuilt at startup (fast for < 100k chunks).
- **Delete**: Remove chunks from SQLite by attachmentID, rebuild BM25 index.
- **Tradeoff**: BM25 index is rebuilt in-memory at startup. For 20+ year archives with 100k+ chunks, this takes ~1-2 seconds. Acceptable for CLI/MCP use.

## DuckDB VSS Adapter

When `provider = "ollama"`, the existing DuckDB VSS implementation is wrapped to implement `SearchStore`:
- `Index()` → calls existing `InsertVector()` + `InsertText()`
- `Search()` → calls existing `Search()` (ANN query), joins with `attachment_text` for chunk text
- `Delete()` → deletes from both `attachment_vectors` and `attachment_text` by attachmentID

## Migration Path

- Existing DuckDB VSS users: change config to `provider = "ollama"`, existing data works
- New users: BM25 works immediately, no setup needed
- Switching providers: run `extract-attachments --reprocess` to re-index with new provider

## Binary Size

- Current: ~15MB (with sqlite3 CGO)
- With BM25: ~15MB (+ ~500KB for BM25 lib)
- With Ollama (current): requires external Ollama install
