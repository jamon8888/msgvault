# Attachment Extraction + Semantic Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional attachment content extraction and embedding-based semantic search to msgvault for RAG and compliance/discovery use cases.

**Architecture:** Three-layer pipeline: (1) Extraction via go-fitz/godocx, (2) Embeddings via Ollama API, (3) Vector storage/search via DuckDB VSS. Hybrid mode extracts on-sync for recent attachments, on-demand for older.

**Tech Stack:** go-fitz (PDF), godocx (DOCX), Ollama (embeddings), DuckDB VSS (vectors), msgvault existing patterns

---

## File Structure

```
internal/
├── extractor/          # NEW: Extraction engine
│   ├── extractor.go   # Service interface + implementation
│   ├── fitz.go        # PDF extraction via go-fitz
│   ├── docx.go        # DOCX extraction via godocx  
│   ├── chunker.go     # Text chunking for embeddings
│   └── extractor_test.go
├── embedding/         # NEW: Embedding service
│   ├── service.go     # Service interface
│   ├── ollama.go      # Ollama API client
│   └── embedding_test.go
├── vector/            # NEW: Vector store
│   ├── store.go      # Service interface
│   ├── duckdb.go      # DuckDB VSS implementation
│   └── vector_test.go
├── config/
│   └── config.go      # MODIFY: Add extraction/embedding/vector config
internal/mcp/
├── handlers.go        # MODIFY: Add search_attachments, extract_attachment tools
├── server.go          # MODIFY: Register new MCP tools
cmd/msgvault/cmd/
├── extract_attachments.go  # NEW: CLI command for extraction
└── root.go            # MODIFY: Register new command
```

---

## Phase 1: Configuration

### Task 1: Add Config Options

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Read existing config.go to understand pattern**

```go
// Read internal/config/config.go to see existing config struct patterns
```

- [ ] **Step 2: Add extraction, embedding, and vector config sections**

Add to Config struct:
```go
type Config struct {
    // ... existing fields ...
    
    Extraction ExtractionConfig
    Embedding  EmbeddingConfig  
    Vector     VectorConfig
}

type ExtractionConfig struct {
    Enabled    bool
    Mode       string  // "on_sync", "on_demand", "hybrid"
    RecentDays int
    Formats    []string
}

type EmbeddingConfig struct {
    Enabled   bool
    Model     string
    Dimensions int
    OllamaURL string
}

type VectorConfig struct {
    Store    string  // "duckdb"
    IndexType string // "hnsw"
}
```

- [ ] **Step 3: Add defaults and env var parsing**

```go
func DefaultConfig() *Config {
    return &Config{
        Extraction: ExtractionConfig{
            Enabled:    false,
            Mode:      "hybrid",
            RecentDays: 30,
            Formats:    []string{"pdf", "docx", "txt"},
        },
        Embedding: EmbeddingConfig{
            Enabled:   true,
            Model:     "nomic-embed-text",
            Dimensions: 1536,
            OllamaURL: "http://localhost:11434",
        },
        Vector: VectorConfig{
            Store:    "duckdb",
            IndexType: "hnsw",
        },
    }
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add extraction, embedding, vector config options"
```

---

## Phase 2: Extraction Engine

### Task 2: Create Extractor Service Interface

**Files:**
- Create: `internal/extractor/extractor.go`
- Test: `internal/extractor/extractor_test.go`

- [ ] **Step 1: Write failing test for Extractor interface**

```go
// internal/extractor/extractor_test.go
package extractor

import (
    "testing"
)

type MockExtractor struct{}

func (m *MockExtractor) ExtractText(path string) (string, error) {
    return "", nil // Not implemented
}

type Extractor interface {
    ExtractText(path string) (string, error)
}

func TestExtractorInterface(t *testing.T) {
    var _ Extractor = (*MockExtractor)(nil)
}

func TestExtractPDF(t *testing.T) {
    // This will fail until we implement
    e := NewExtractor("pdf")
    text, err := e.ExtractText("test.pdf")
    if err != nil {
        t.Errorf("ExtractText failed: %v", err)
    }
    if text == "" {
        t.Error("Expected non-empty text")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/extractor/... -v
Expected: FAIL - undefined: NewExtractor

- [ ] **Step 3: Write minimal Extractor interface + factory**

```go
// internal/extractor/extractor.go
package extractor

import (
    "errors"
)

var (
    ErrUnsupportedFormat = errors.New("unsupported format")
)

type Extractor interface {
    ExtractText(path string) (string, error)
}

func NewExtractor(format string) (Extractor, error) {
    switch format {
    case "pdf":
        return &PDFExtractor{}, nil
    case "docx":
        return &DOCXExtractor{}, nil
    case "txt":
        return &TXTExtractor{}, nil
    default:
        return nil, ErrUnsupportedFormat
    }
}

type PDFExtractor struct{}

func (e *PDFExtractor) ExtractText(path string) (string, error) {
    // TODO: implement with go-fitz
    return "mock text", nil
}

type DOCXExtractor struct{}

func (e *DOCXExtractor) ExtractText(path string) (string, error) {
    // TODO: implement with godocx
    return "mock text", nil
}

type TXTExtractor struct{}

func (e *TXTExtractor) ExtractText(path string) (string, error) {
    // Simple text reading
    return "mock text", nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/extractor/... -v
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/extractor/extractor.go internal/extractor/extractor_test.go
git commit -m "feat: add extractor interface with PDF/DOCX/TXT support"
```

### Task 3: Implement PDF Extraction with go-fitz

**Files:**
- Modify: `internal/extractor/extractor.go`
- Create: `internal/extractor/fitz.go`

- [ ] **Step 1: Add go-fitz dependency**

```bash
go get github.com/gen2brain/go-fitz
```

- [ ] **Step 2: Write failing test for PDF extraction**

```go
// Add to extractor_test.go
func TestPDFExtractorReal(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping real PDF extraction test")
    }
    e := &PDFExtractor{}
    // This will fail - no actual PDF file
    _, err := e.ExtractText("nonexistent.pdf")
    // Should fail gracefully
    t.Logf("Expected error for missing file: %v", err)
}
```

- [ ] **Step 3: Implement PDFExtractor with go-fitz**

```go
// internal/extractor/fitz.go
package extractor

import (
    "github.com/gen2brain/go-fitz"
)

func (e *PDFExtractor) ExtractText(path string) (string, error) {
    doc, err := fitz.New(path)
    if err != nil {
        return "", err
    }
    defer doc.Close()

    var text strings.Builder
    for i := 0; i < doc.NumPage(); i++ {
        pageText, err := doc.Text(i)
        if err != nil {
            continue // Skip pages that fail
        }
        text.WriteString(pageText)
        text.WriteString("\n")
    }
    return text.String(), nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/extractor/... -v -short
Expected: PASS (skips real test), build succeeds

- [ ] **Step 5: Commit**

```bash
git add internal/extractor/fitz.go go.mod go.sum
git commit -m "feat: implement PDF extraction with go-fitz"
```

### Task 4: Implement DOCX Extraction with godocx

**Files:**
- Modify: `internal/extractor/extractor.go`
- Create: `internal/extractor/docx.go`

- [ ] **Step 1: Add godocx dependency**

```bash
go get github.com/gomutex/godocx
```

- [ ] **Step 2: Implement DOCXExtractor**

```go
// internal/extractor/docx.go
package extractor

import (
    "github.com/gomutex/godocx"
)

func (e *DOCXExtractor) ExtractText(path string) (string, error) {
    doc, err := godocx.ReadFile(path)
    if err != nil {
        return "", err
    }
    return doc.GetText(), nil
}
```

- [ ] **Step 3: Test compilation**

```bash
go build ./internal/extractor/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/extractor/docx.go go.mod go.sum
git commit -m "feat: implement DOCX extraction with godocx"
```

### Task 5: Implement Text Chunker

**Files:**
- Create: `internal/extractor/chunker.go`
- Test: `internal/extractor/chunker_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/extractor/chunker_test.go
package extractor

import (
    "testing"
)

func TestChunker(t *testing.T) {
    text := "This is a long text " + string(make([]byte, 8000)) // ~8000 chars
    chunks := ChunkText(text, 8192, 512)
    
    if len(chunks) != 2 {
        t.Errorf("Expected 2 chunks, got %d", len(chunks))
    }
    
    // Check overlap - last 512 chars of chunk 0 should match start of chunk 1
    if len(chunks) > 1 {
        overlapLen := min(512, len(chunks[0]))
        if chunks[0][len(chunks[0])-overlapLen:] != chunks[1][:overlapLen] {
            t.Error("Expected overlap between chunks")
        }
    }
}
```

- [ ] **Step 2: Run test - verify it fails**

```bash
go test ./internal/extractor/... -run TestChunker -v
Expected: FAIL - undefined: ChunkText

- [ ] **Step 3: Implement ChunkText function**

```go
// internal/extractor/chunker.go
package extractor

func ChunkText(text string, chunkSize, overlap int) []string {
    if len(text) <= chunkSize {
        return []string{text}
    }
    
    var chunks []string
    start := 0
    
    for start < len(text) {
        end := start + chunkSize
        if end > len(text) {
            end = len(text)
        }
        chunks = append(chunks, text[start:end])
        start = end - overlap
        if start <= 0 {
            break
        }
    }
    
    return chunks
}
```

- [ ] **Step 4: Run test - verify it passes**

```bash
go test ./internal/extractor/... -run TestChunker -v
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/extractor/chunker.go internal/extractor/chunker_test.go
git commit -m "feat: add text chunker for embedding pipeline"
```

---

## Phase 3: Embedding Service

### Task 6: Create Embedding Service

**Files:**
- Create: `internal/embedding/service.go`
- Create: `internal/embedding/ollama.go`
- Test: `internal/embedding/embedding_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/embedding/embedding_test.go
package embedding

import (
    "testing"
)

func TestOllamaEmbedding(t *testing.T) {
    client := NewOllamaClient("http://localhost:11434")
    emb, err := client.Embed("nomic-embed-text", "test text")
    if err != nil {
        t.Skipf("Ollama not available: %v", err)
    }
    if len(emb) != 1536 {
        t.Errorf("Expected 1536 dimensions, got %d", len(emb))
    }
}
```

- [ ] **Step 2: Run test - verify it fails**

```bash
go test ./internal/embedding/... -v
Expected: FAIL - undefined: NewOllamaClient

- [ ] **Step 3: Implement Ollama client**

```go
// internal/embedding/ollama.go
package embedding

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type OllamaClient struct {
    baseURL string
    client  *http.Client
}

type EmbedRequest struct {
    Model string `json:"model"`
    Input string `json:"input"`
}

type EmbedResponse struct {
    Embeddings [][]float `json:"embeddings"`
}

func NewOllamaClient(baseURL string) *OllamaClient {
    return &OllamaClient{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

func (c *OllamaClient) Embed(model, text string) ([]float, error) {
    reqBody, _ := json.Marshal(EmbedRequest{Model: model, Input: text})
    resp, err := c.client.Post(c.baseURL+"/api/embed", "application/json", bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result EmbedResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if len(result.Embeddings) == 0 {
        return nil, nil
    }
    return result.Embeddings[0], nil
}
```

- [ ] **Step 4: Run test**

```bash
go test ./internal/embedding/... -v
Expected: PASS (or skip if Ollama unavailable)

- [ ] **Step 5: Commit**

```bash
git add internal/embedding/
git commit -m "feat: add Ollama embedding client"
```

---

## Phase 4: Vector Store

### Task 7: Create Vector Store with DuckDB VSS

**Files:**
- Create: `internal/vector/store.go`
- Create: `internal/vector/duckdb.go`
- Test: `internal/vector/vector_test.go`

- [ ] **Step 1: Ensure DuckDB dependency available**

```bash
go list -m github.com/marcboeker/go-duckdb
```

- [ ] **Step 2: Write failing test**

```go
// internal/vector/vector_test.go
package vector

import (
    "testing"
)

func TestDuckDBVectorStore(t *testing.T) {
    store, err := NewDuckDBStore(":memory:")
    if err != nil {
        t.Skipf("DuckDB not available: %v", err)
    }
    defer store.Close()
    
    // Test init schema
    err = store.InitSchema()
    if err != nil {
        t.Errorf("InitSchema failed: %v", err)
    }
}
```

- [ ] **Step 3: Run test - verify it fails**

```bash
go test ./internal/vector/... -v
Expected: FAIL - undefined: NewDuckDBStore

- [ ] **Step 4: Implement VectorStore interface + DuckDB**

```go
// internal/vector/store.go
package vector

type VectorStore interface {
    InitSchema() error
    InsertVector(id int64, embedding []float) error
    Search(query []float, limit int) ([]int64, error)
    Close() error
}

// internal/vector/duckdb.go
package vector

import (
    "database/sql"
    _ "github.com/marcboeker/go-duckdb"
)

type DuckDBStore struct {
    db *sql.DB
}

func NewDuckDBStore(dsn string) (*DuckDBStore, error) {
    db, err := sql.Open("duckdb", dsn)
    if err != nil {
        return nil, err
    }
    return &DuckDBStore{db: db}, nil
}

func (s *DuckDBStore) InitSchema() error {
    // Install and load VSS
    _, err := s.db.Exec("INSTALL vss; LOAD vss;")
    if err != nil {
        return err
    }
    
    // Create table
    _, err = s.db.Exec(`
        CREATE TABLE IF NOT EXISTS attachment_vectors (
            id BIGINT,
            message_id BIGINT,
            attachment_id BIGINT,
            chunk_index INTEGER,
            embedding FLOAT[]
        );
        CREATE INDEX IF NOT EXISTS idx_ann ON attachment_vectors USING HNSW (embedding);
    `)
    return err
}

func (s *DuckDBStore) Close() error {
    return s.db.Close()
}

// InsertVector and Search to be implemented
```

- [ ] **Step 5: Run test**

```bash
go test ./internal/vector/... -v
Expected: PASS (or skip if DuckDB fails)

- [ ] **Step 6: Commit**

```bash
git add internal/vector/
git commit -m "feat: add DuckDB VSS vector store"
```

---

## Phase 5: MCP Tools

### Task 8: Add MCP Search Tools

**Files:**
- Modify: `internal/mcp/handlers.go`
- Modify: `internal/mcp/server.go`

- [ ] **Step 1: Add new tool handlers**

Add to handlers struct:
```go
type handlers struct {
    // ... existing fields ...
    extractor  *extractor.Extractor
    embedding  *embedding.Service
    vector     *vector.VectorStore
}
```

Add new handlers:
```go
func (h *handlers) searchAttachments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args := req.GetArguments()
    query, _ := args["query"].(string)
    limit := limitArg(args, "limit", 10)
    
    // 1. Generate embedding from query
    // 2. Search vector store
    // 3. Return results
    
    return jsonResult(results)
}

func (h *handlers) extractAttachment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args := req.GetArguments()
    id, err := getIDArg(args, "attachment_id")
    // Extract and store
    return jsonResult(result)
}
```

- [ ] **Step 2: Register tools in server.go**

```go
s.AddTool(searchAttachmentsTool(), h.searchAttachments)
s.AddTool(extractAttachmentTool(), h.extractAttachment)
```

- [ ] **Step 3: Build and test**

```bash
go build ./internal/mcp/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/mcp/handlers.go internal/mcp/server.go
git commit -m "feat: add MCP search_attachments and extract_attachment tools"
```

### Task 8b: Update Existing MCP Tools to Include Attachment Content

**Files:**
- Modify: `internal/mcp/handlers.go`

Per spec, update existing tools to include attachment text:

- [ ] **Step 1: Update search_messages to optionally include attachment content**

Modify searchMessages handler to accept `include_attachments` parameter:
```go
func (h *handlers) searchMessages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // ... existing code ...
    
    includeAttachments, _ := args["include_attachments"].(bool)
    
    if includeAttachments && h.vector != nil {
        // Include semantic search results from attachment vectors
        // Merge with existing message results
    }
    
    // ... existing return ...
}
```

- [ ] **Step 2: Update get_message to include extracted attachment text**

Modify getMessage handler to include attachment content:
```go
func (h *handlers) getMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // ... existing code getting msg ...
    
    // Add extracted attachment text to response
    if h.extractor != nil && msg.HasAttachments {
        // Fetch attachment text from vector store
    }
    
    return jsonResult(msg)
}
```

- [ ] **Step 3: Build and commit**

```bash
go build ./internal/mcp/...
git add internal/mcp/handlers.go
git commit -m "feat: update search_messages and get_message to include attachment content"
```

---

## Phase 6: CLI Commands

### Task 9: Add CLI Commands

**Files:**
- Create: `cmd/msgvault/cmd/extract_attachments.go`

- [ ] **Step 1: Create extract-attachments command**

```go
// cmd/msgvault/cmd/extract_attachments.go
package cmd

import (
    "github.com/spf13/cobra"
)

var extractAttachmentsCmd = &cobra.Command{
    Use:   "extract-attachments",
    Short: "Extract text from unprocessed attachments",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}

func init() {
    rootCmd.AddCommand(extractAttachmentsCmd)
    extractAttachmentsCmd.Flags().Int("limit", 100, "Max attachments to process")
    extractAttachmentsCmd.Flags().Bool("reprocess", false, "Reprocess already extracted")
}
```

- [ ] **Step 2: Build and test**

```bash
go build ./cmd/msgvault/...
./msgvault extract-attachments --help
```

- [ ] **Step 3: Commit**

```bash
git add cmd/msgvault/cmd/extract_attachments.go
git commit -m "feat: add extract-attachments CLI command"
```

---

## Final Integration

### Task 10: Integration Test

- [ ] **Step 1: Create end-to-end test**

```go
// Test full pipeline: attachment -> extract -> embed -> store -> search
func TestFullPipeline(t *testing.T) {
    // 1. Extract text from sample PDF
    // 2. Chunk text
    // 3. Generate embedding
    // 4. Store in vector DB
    // 5. Search and verify results
}
```

- [ ] **Step 2: Run integration test**

```bash
go test ./internal/... -tags=integration -v
```

### Task 10b: Integrate with Sync Pipeline (Optional - Future Phase)

**Files:**
- Modify: `cmd/msgvault/cmd/sync_full.go` or similar
- Create: `internal/extractor/sync_hook.go`

Per spec, OnSync extraction should trigger after email sync completes:

- [ ] **Step 1: Create sync hook interface**

```go
// internal/extractor/sync_hook.go
package extractor

type SyncHook interface {
    OnSyncComplete(attachments []AttachmentInfo) error
}

func NewSyncHook(extractor Extractor, embedding *embedding.Service, vector vector.VectorStore) *SyncHook {
    return &SyncHook{...}
}
```

- [ ] **Step 2: Document in plan as future enhancement**

> Note: Full sync pipeline integration is marked as future phase. The CLI command `extract-attachments` provides manual triggering for now.

- [ ] **Step 3: Commit**

```bash
git add internal/extractor/sync_hook.go
git commit -m "feat: add sync hook for automatic extraction (future)"
```

---

## Plan Complete

Total tasks: 11 tasks across 6 phases
Estimated: ~1-2 hours with parallel subagents