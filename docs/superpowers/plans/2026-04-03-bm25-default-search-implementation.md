# BM25 Default Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Ollama-dependent vector search with BM25 as the default, making attachment search work out-of-the-box with zero external dependencies.

**Architecture:** Add a unified `SearchStore` interface with two implementations: BM25 (default, pure Go) and DuckDB VSS (optional, requires Ollama). BM25 stores chunks in SQLite and builds an in-memory index at startup.

**Tech Stack:** Go, `github.com/crawlab-team/bm25`, SQLite (existing), DuckDB VSS (optional)

---

### Task 1: Create SearchStore Interface

**Files:**
- Create: `internal/search/store.go`
- Modify: `internal/config/config.go`

- [ ] **Step 1: Create the SearchStore interface**

```go
// internal/search/store.go
package search

import "context"

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
	Score        float64 // BM25: TF-IDF-like; VSS: cosine distance. Relative, not absolute.
}
```

- [ ] **Step 2: Add Provider field to EmbeddingConfig**

In `internal/config/config.go`, add to `EmbeddingConfig`:

```go
type EmbeddingConfig struct {
	Enabled    bool   `toml:"enabled"`
	Provider   string `toml:"provider"`   // "bm25" (default) or "ollama"
	Model      string `toml:"model"`      // Ollama model name
	Dimensions int    `toml:"dimensions"` // Embedding dimension
	OllamaURL  string `toml:"ollama_url"` // Ollama server URL
}
```

- [ ] **Step 3: Set default provider in NewDefaultConfig**

Set `Embedding.Provider = "bm25"` in the default config.

- [ ] **Step 4: Build and verify**

Run: `wsl bash -c "cd /mnt/c/Users/NMarchitecte/Documents/msgvault && CGO_ENABLED=1 go build ./internal/search/... ./internal/config/..."`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/search/store.go internal/config/config.go
git commit -m "feat: add SearchStore interface and provider config"
```

---

### Task 2: Implement BM25 Search Store

**Files:**
- Create: `internal/search/bm25.go`
- Create: `internal/search/bm25_test.go`

**Design:** BM25 index is in-memory for fast search. Chunks are stored in SQLite for persistence. On startup, chunks are loaded from SQLite and the BM25 index is rebuilt.

- [ ] **Step 1: Add BM25 dependency**

Run: `go get github.com/crawlab-team/bm25@latest`

- [ ] **Step 2: Write failing test**

```go
// internal/search/bm25_test.go
package search

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestBM25Store(t *testing.T) *BM25Store {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewBM25Store(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestBM25IndexAndSearch(t *testing.T) {
	store := newTestBM25Store(t)
	ctx := context.Background()

	store.Index(ctx, 1, 1, 100, 0, "The quick brown fox jumps over the lazy dog")
	store.Index(ctx, 2, 1, 101, 0, "A fast red cat runs through the green forest")
	store.Index(ctx, 3, 2, 102, 0, "Python programming language for data science")
	store.Flush()

	results, err := store.Search(ctx, "quick fox", 2)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].AttachmentID != 100 {
		t.Errorf("expected attachment 100, got %d", results[0].AttachmentID)
	}
}

func TestBM25Delete(t *testing.T) {
	store := newTestBM25Store(t)
	ctx := context.Background()
	store.Index(ctx, 1, 1, 100, 0, "test document about dogs")
	store.Index(ctx, 2, 1, 101, 0, "another document about cats")
	store.Flush()

	store.Delete(ctx, 100)

	results, _ := store.Search(ctx, "dogs", 5)
	if len(results) > 0 {
		t.Error("expected no results after delete")
	}
}

func TestBM25GetChunksByAttachmentID(t *testing.T) {
	store := newTestBM25Store(t)
	ctx := context.Background()
	store.Index(ctx, 1, 1, 100, 0, "chunk one text")
	store.Index(ctx, 2, 1, 100, 1, "chunk two text")
	store.Index(ctx, 3, 1, 100, 2, "chunk three text")
	store.Flush()

	texts, err := store.GetChunksByAttachmentID(100)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(texts) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(texts))
	}
	if texts[0] != "chunk one text" {
		t.Errorf("expected 'chunk one text' first, got %s", texts[0])
	}
}

func TestBM25Persistence(t *testing.T) {
	// Create a temp SQLite DB
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, _ := NewBM25Store(db)
	ctx := context.Background()
	store.Index(ctx, 1, 1, 100, 0, "persistent chunk")
	store.Flush()

	// Create a new store from the same DB
	store2, err := NewBM25Store(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	// Should have loaded the chunk from SQLite
	results, _ := store2.Search(ctx, "persistent", 5)
	if len(results) == 0 {
		t.Error("expected persisted chunk to be searchable")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/search/... -v`
Expected: FAIL (BM25Store not defined)

- [ ] **Step 4: Create the SQLite schema**

Add to `internal/store/schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS attachment_chunks (
    id INTEGER PRIMARY KEY,
    message_id INTEGER NOT NULL,
    attachment_id INTEGER NOT NULL,
    chunk_index INTEGER NOT NULL,
    chunk_text TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chunks_attachment ON attachment_chunks(attachment_id);
```

- [ ] **Step 5: Implement BM25Store**

Key design:
- **SQLite-backed**: Chunks stored in `attachment_chunks` table
- **In-memory BM25 index**: Rebuilt from SQLite at startup and after Flush()
- **Flush()**: Call after batching Index() calls to rebuild index
- **Delete**: Removes from SQLite, marks dirty, rebuild on next Flush/Search

```go
// internal/search/bm25.go
package search

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"sync"

	"github.com/crawlab-team/bm25"
)

type BM25Store struct {
	mu       sync.RWMutex
	db       *sql.DB
	chunks   map[int64]chunk  // doc ID → chunk data
	docOrder []int64          // ordered list of doc IDs (matches BM25 index positions)
	bm25     *bm25.BM25
	nextID   int64
}

type chunk struct {
	messageID    int64
	attachmentID int64
	chunkIndex   int
	text         string
}

func NewBM25Store(db *sql.DB) (*BM25Store, error) {
	s := &BM25Store{
		db:       db,
		chunks:   make(map[int64]chunk),
		docOrder: make([]int64, 0),
	}

	// Ensure table exists
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attachment_chunks (
			id INTEGER PRIMARY KEY,
			message_id INTEGER NOT NULL,
			attachment_id INTEGER NOT NULL,
			chunk_index INTEGER NOT NULL,
			chunk_text TEXT NOT NULL
		)
	`); err != nil {
		return nil, err
	}

	// Load existing chunks from SQLite
	if err := s.loadFromDB(); err != nil {
		return nil, err
	}

	// Build BM25 index from loaded chunks
	s.rebuildIndex()
	return s, nil
}

func (s *BM25Store) loadFromDB() error {
	rows, err := s.db.Query(`SELECT id, message_id, attachment_id, chunk_index, chunk_text FROM attachment_chunks ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var c chunk
		if err := rows.Scan(&id, &c.messageID, &c.attachmentID, &c.chunkIndex, &c.text); err != nil {
			return err
		}
		s.chunks[id] = c
		s.docOrder = append(s.docOrder, id)
		if id >= s.nextID {
			s.nextID = id + 1
		}
	}
	return rows.Err()
}

func (s *BM25Store) Index(_ context.Context, id int64, messageID int64, attachmentID int64, chunkIndex int, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	docID := s.nextID
	s.nextID++

	// Persist to SQLite
	if _, err := s.db.Exec(
		`INSERT INTO attachment_chunks (id, message_id, attachment_id, chunk_index, chunk_text) VALUES (?, ?, ?, ?, ?)`,
		docID, messageID, attachmentID, chunkIndex, text,
	); err != nil {
		return err
	}

	s.chunks[docID] = chunk{
		messageID:    messageID,
		attachmentID: attachmentID,
		chunkIndex:   chunkIndex,
		text:         text,
	}
	s.docOrder = append(s.docOrder, docID)
	return nil
}

func (s *BM25Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rebuildIndex()
	return nil
}

func (s *BM25Store) Search(_ context.Context, query string, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.bm25 == nil {
		return nil, nil
	}

	terms := tokenize(query)
	if len(terms) == 0 {
		return nil, nil
	}

	// GetScores returns a slice indexed 0..N-1 matching docOrder
	scores := s.bm25.GetScores(terms)

	type scoredDoc struct {
		bm25Idx int
		score   float64
	}
	var scored []scoredDoc

	for i, score := range scores {
		if score > 0 {
			scored = append(scored, scoredDoc{i, score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}

	results := make([]SearchResult, len(scored))
	for i, sd := range scored {
		docID := s.docOrder[sd.bm25Idx]
		c := s.chunks[docID]
		results[i] = SearchResult{
			AttachmentID: c.attachmentID,
			MessageID:    c.messageID,
			ChunkIndex:   c.chunkIndex,
			ChunkText:    c.text,
			Score:        sd.score,
		}
	}

	return results, nil
}

func (s *BM25Store) GetChunksByAttachmentID(attachmentID int64) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT chunk_text FROM attachment_chunks WHERE attachment_id = ? ORDER BY chunk_index`,
		attachmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var texts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		texts = append(texts, text)
	}
	return texts, rows.Err()
}

func (s *BM25Store) Delete(_ context.Context, attachmentID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from SQLite
	if _, err := s.db.Exec(`DELETE FROM attachment_chunks WHERE attachment_id = ?`, attachmentID); err != nil {
		return err
	}

	// Remove from memory
	for docID, c := range s.chunks {
		if c.attachmentID == attachmentID {
			delete(s.chunks, docID)
		}
	}

	// Rebuild docOrder and BM25 index
	s.rebuildIndex()
	return nil
}

func (s *BM25Store) Close() error {
	return nil
}

func (s *BM25Store) rebuildIndex() {
	// Build documents list in docOrder (deterministic)
	docs := make([][]string, 0, len(s.docOrder))
	for _, docID := range s.docOrder {
		if c, ok := s.chunks[docID]; ok {
			docs = append(docs, tokenize(c.text))
		}
	}

	// Remove deleted docIDs from docOrder
	var newOrder []int64
	for _, docID := range s.docOrder {
		if _, ok := s.chunks[docID]; ok {
			newOrder = append(newOrder, docID)
		}
	}
	s.docOrder = newOrder

	if len(docs) == 0 {
		s.bm25 = nil
		return
	}

	s.bm25 = bm25.NewBM25(docs, bm25.DefaultConfig())
}

func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ',' || r == '.' || r == '!' || r == '?' || r == ';' || r == ':' || r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' || r == '"' || r == '\''
	})
	var result []string
	for _, w := range words {
		if len(w) > 0 {
			result = append(result, w)
		}
	}
	return result
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/search/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/search/bm25.go internal/search/bm25_test.go go.mod go.sum internal/store/schema.sql
git commit -m "feat: implement BM25 search store with SQLite persistence"
```

---

### Task 3: Update MCP Server to Use SearchStore

**Files:**
- Modify: `internal/mcp/server.go`
- Modify: `internal/mcp/handlers.go`

- [ ] **Step 1: Update handlers struct**

In `internal/mcp/handlers.go`, replace `vectorStore vector.VectorStore` with `searchStore search.SearchStore`:

```go
type handlers struct {
	engine         query.Engine
	attachmentsDir string
	dataDir        string
	piiFilter      *pii.Filter
	extractor      extractor.Service
	embedding      embedding.Service
	searchStore    search.SearchStore
}
```

Also rename `VectorMatch` to `AttachmentMatch` and update its usage.

- [ ] **Step 2: Update searchMessages handler**

Replace vector store calls with search store:

```go
// If include_attachments and search store is available, perform search
if includeAttachments && h.searchStore != nil && queryStr != "" {
    searchResults, err := h.searchStore.Search(ctx, queryStr, limit)
    if err == nil && len(searchResults) > 0 {
        matches := make([]AttachmentMatch, len(searchResults))
        for i, sr := range searchResults {
            matches[i] = AttachmentMatch{
                AttachmentID: sr.AttachmentID,
                ChunkText:   sr.ChunkText,
                Distance:    sr.Score,
            }
        }
        resp := SearchResultWithVectors{
            Messages:          results,
            AttachmentMatches: matches,
        }
        // ... PII filter ...
        return jsonResult(resp)
    }
}
```

- [ ] **Step 3: Update getMessage handler**

Replace `h.vectorStore.GetTextByAttachmentID()` with `h.searchStore.GetChunksByAttachmentID()`.

The `GetChunksByAttachmentID` method is already part of the SearchStore interface (see Task 1).

Update interface in `internal/search/store.go` to include:

```go
type SearchStore interface {
    Index(ctx context.Context, id int64, messageID int64, attachmentID int64, chunkIndex int, text string) error
    Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
    GetChunksByAttachmentID(attachmentID int64) ([]string, error)
    Delete(ctx context.Context, attachmentID int64) error
    Close() error
}
```

In the getMessage handler, replace:
```go
texts, err := h.vectorStore.GetTextByAttachmentID(att.ID)
```
with:
```go
texts, err := h.searchStore.GetChunksByAttachmentID(att.ID)
```

- [ ] **Step 4: Update Serve() in server.go**

Replace the vector/ollama initialization with:

```go
// Initialize search store
var searchStore search.SearchStore

if cfg != nil && cfg.Embedding.Provider == "ollama" {
    // Use Ollama + DuckDB VSS
    ollamaClient := embedding.NewOllamaClient(cfg.Embedding.OllamaURL)
    embeddingSvc = embedding.NewEmbeddingService(ollamaClient, cfg.Embedding.Model)

    vectorDSN := "vector.duckdb"
    vectorSvc, err := vector.NewDuckDBStore(vectorDSN)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Warning: Failed to initialize vector store: %v\n", err)
        vectorSvc = nil
    }
    if vectorSvc != nil {
        searchStore = search.NewVectorSearchAdapter(vectorSvc, embeddingSvc)
    }
}

// Default: BM25
if searchStore == nil {
    searchStore, err = search.NewBM25Store(s.DB())
    if err != nil {
        return fmt.Errorf("init BM25 store: %w", err)
    }
}

h := &handlers{
    engine:         engine,
    attachmentsDir: attachmentsDir,
    dataDir:        dataDir,
    piiFilter:      piiFilter,
    extractor:      extractorSvc,
    embedding:      embeddingSvc,
    searchStore:    searchStore,
}
```

- [ ] **Step 5: Build and verify**

Run: `wsl bash -c "cd /mnt/c/Users/NMarchitecte/Documents/msgvault && CGO_ENABLED=1 go build ./internal/mcp/... ./internal/search/..."`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/server.go internal/mcp/handlers.go internal/search/store.go internal/search/bm25.go
git commit -m "feat: wire BM25 search store into MCP server"
```

---

### Task 4: Create DuckDB VSS Adapter

**Files:**
- Create: `internal/search/vss_adapter.go`

- [ ] **Step 1: Write the adapter**

```go
// internal/search/vss_adapter.go
package search

import (
	"context"

	"github.com/wesm/msgvault/internal/embedding"
	"github.com/wesm/msgvault/internal/vector"
)

type VectorSearchAdapter struct {
	vs  vector.VectorStore
	emb embedding.Service
}

func NewVectorSearchAdapter(vs vector.VectorStore, emb embedding.Service) *VectorSearchAdapter {
	return &VectorSearchAdapter{vs: vs, emb: emb}
}

func (a *VectorSearchAdapter) Index(ctx context.Context, id int64, messageID int64, attachmentID int64, chunkIndex int, text string) error {
	emb, err := a.emb.Embed(text)
	if err != nil {
		return err
	}
	if err := a.vs.InsertVector(id, messageID, attachmentID, chunkIndex, emb); err != nil {
		return err
	}
	return a.vs.InsertText(id, messageID, attachmentID, chunkIndex, text)
}

func (a *VectorSearchAdapter) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	emb, err := a.emb.Embed(query)
	if err != nil {
		return nil, err
	}
	vecResults, err := a.vs.Search(emb, limit)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, len(vecResults))
	for i, vr := range vecResults {
		results[i] = SearchResult{
			AttachmentID: vr.AttachmentID,
			MessageID:    vr.MessageID,
			ChunkIndex:   vr.ChunkIndex,
			ChunkText:    vr.ChunkText,
			Score:        vr.Distance,
		}
	}
	return results, nil
}

func (a *VectorSearchAdapter) GetChunksByAttachmentID(attachmentID int64) ([]string, error) {
	return a.vs.GetTextByAttachmentID(attachmentID)
}

func (a *VectorSearchAdapter) Delete(ctx context.Context, attachmentID int64) error {
	// VSS doesn't have a direct delete by attachmentID; would need SQL
	return nil
}

func (a *VectorSearchAdapter) Close() error {
	return a.vs.Close()
}
```

- [ ] **Step 2: Build and verify**

Run: `wsl bash -c "cd /mnt/c/Users/NMarchitecte/Documents/msgvault && CGO_ENABLED=1 go build ./internal/search/..."`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/search/vss_adapter.go
git commit -m "feat: add DuckDB VSS adapter for SearchStore interface"
```

---

### Task 5: Update extract-attachments CLI

**Files:**
- Modify: `cmd/msgvault/cmd/extract_attachments.go`

- [ ] **Step 1: Replace VectorStore with SearchStore**

Change the command to use `search.SearchStore` instead of `vector.VectorStore`.

Respect the `provider` config:
- If `provider = "ollama"`: use VectorSearchAdapter (needs Ollama + DuckDB VSS)
- Otherwise (default): use BM25Store (no external deps)

```go
// Initialize search store based on config
var searchStore search.SearchStore

if cfg.Embedding.Provider == "ollama" {
    ollamaClient := embedding.NewOllamaClient(cfg.Embedding.OllamaURL)
    embeddingSvc := embedding.NewEmbeddingService(ollamaClient, cfg.Embedding.Model)
    vectorSvc, _ := vector.NewDuckDBStore(filepath.Join(cfg.Data.DataDir, "vector.duckdb"))
    searchStore = search.NewVectorSearchAdapter(vectorSvc, embeddingSvc)
} else {
    // BM25 uses the same SQLite DB
    searchStore, _ = search.NewBM25Store(s.DB())
}
defer searchStore.Close()
```

- [ ] **Step 2: Update processAttachment function**

Replace:
```go
vectorSvc.InsertVector(int64(i), 0, att.ID, i, embedding)
vectorSvc.InsertText(int64(i), 0, att.ID, i, chunk)
```

With:
```go
searchStore.Index(ctx, int64(i), 0, att.ID, i, chunk)
```

After processing all chunks for an attachment, call `Flush()` if it's a BM25Store:

```go
// After all chunks indexed for this attachment
if flusher, ok := searchStore.(interface{ Flush() error }); ok {
    flusher.Flush()
}
```

- [ ] **Step 3: Remove embedding check for BM25 mode**

When `provider = "bm25"` (default), the extract command should NOT require Ollama. Only check config if provider is "ollama":

```go
if cfg.Embedding.Provider == "ollama" && !cfg.Embedding.Enabled {
    return fmt.Errorf("embedding not enabled for ollama provider...")
}
// BM25 works without any external service
```

- [ ] **Step 4: Build and verify**

Run: `wsl bash -c "cd /mnt/c/Users/NMarchitecte/Documents/msgvault && CGO_ENABLED=1 go build ./cmd/msgvault/..."`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/msgvault/cmd/extract_attachments.go
git commit -m "feat: update extract-attachments to use SearchStore"
```

---

### Task 6: Integration Test

**Files:**
- Modify: `internal/integration/extract_test.go`

- [ ] **Step 1: Add BM25 integration test**

```go
func TestBM25FullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create in-memory SQLite DB for test
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	store, err := search.NewBM25Store(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Index some test chunks
	store.Index(ctx, 1, 1, 1, 0, "Invoice #1234 from Acme Corp for $500")
	store.Index(ctx, 2, 1, 2, 0, "Meeting notes from Q4 planning session")
	store.Index(ctx, 3, 2, 3, 0, "Project budget allocation for 2025")
	store.Flush()

	// Search
	results, err := store.Search(ctx, "invoice acme", 2)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].AttachmentID != 1 {
		t.Errorf("expected invoice result first, got attachment %d", results[0].AttachmentID)
	}

	t.Logf("Search returned %d results, top score: %.4f", len(results), results[0].Score)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/search/... ./internal/integration/... -v -short`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/integration/extract_test.go
git commit -m "test: add BM25 integration test"
```
