//go:build integration
// +build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wesm/msgvault/internal/embedding"
	"github.com/wesm/msgvault/internal/extractor"
	"github.com/wesm/msgvault/internal/vector"
)

func TestFullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test 1: Extract text from sample PDF
	t.Run("ExtractPDF", func(t *testing.T) {
		// Create a minimal test PDF or skip if no test file
		testFile := filepath.Join("testdata", "sample.pdf")
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Skip("testdata/sample.pdf not found")
		}

		ext := &extractor.ExtractorService{}
		text, err := ext.Extract("pdf", testFile)
		if err != nil {
			t.Fatalf("extract failed: %v", err)
		}
		if text == "" {
			t.Error("expected non-empty text")
		}
		t.Logf("Extracted %d characters", len(text))
	})

	// Test 2: Chunk text
	t.Run("ChunkText", func(t *testing.T) {
		text := "This is a test document. It has multiple sentences. " +
			"We want to split this into chunks. Each chunk should be " +
			"approximately the same size. This helps with embedding generation."

		chunks := extractor.ChunkText(text, 50, 10)
		if len(chunks) == 0 {
			t.Error("expected at least one chunk")
		}
		t.Logf("Created %d chunks", len(chunks))
	})

	// Test 3: Generate embedding
	t.Run("GenerateEmbedding", func(t *testing.T) {
		client := embedding.NewOllamaClient("http://localhost:11434")
		svc := embedding.NewEmbeddingService(client, "nomic-embed-text")

		text := "test document for embedding"
		emb, err := svc.Embed(text)
		if err != nil {
			t.Skipf("skipping embedding test: %v (is Ollama running?)", err)
		}
		if len(emb) == 0 {
			t.Error("expected non-empty embedding")
		}
		t.Logf("Generated embedding with %d dimensions", len(emb))
	})

	// Test 4: Store and search in vector DB
	t.Run("VectorStore", func(t *testing.T) {
		store, err := vector.NewDuckDBStore(":memory:")
		if err != nil {
			t.Skipf("skipping vector store test: %v", err)
		}
		defer store.Close()

		if err := store.InitSchema(); err != nil {
			t.Fatalf("init schema: %v", err)
		}

		// Insert test vector
		testVec := make([]float64, 1536)
		for i := range testVec {
			testVec[i] = float64(i%10) / 10.0
		}

		err = store.InsertVector(1, 1, 1, 0, testVec)
		if err != nil {
			t.Fatalf("insert vector: %v", err)
		}

		err = store.InsertText(1, 1, 1, 0, "test document content")
		if err != nil {
			t.Fatalf("insert text: %v", err)
		}

		// Search
		results, err := store.Search(testVec, 1)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected at least one result")
		}

		t.Logf("Search returned %d results", len(results))
	})

	// Test 5: Full pipeline (requires Ollama and vector store)
	t.Run("FullPipeline", func(t *testing.T) {
		t.Skip("Full E2E test requires full environment setup")
		// This would test:
		// 1. Load attachment
		// 2. Extract text
		// 3. Chunk
		// 4. Embed
		// 5. Store
		// 6. Search and verify
	})
}
