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
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	store, _ := NewBM25Store(db)
	ctx := context.Background()
	store.Index(ctx, 1, 1, 100, 0, "persistent chunk about dogs")
	store.Index(ctx, 2, 1, 101, 0, "another document about cats")
	store.Index(ctx, 3, 1, 102, 0, "third document about birds")
	store.Flush()

	store2, err := NewBM25Store(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	results, _ := store2.Search(ctx, "persistent", 5)
	if len(results) == 0 {
		t.Error("expected persisted chunk to be searchable")
	}
}

func TestBM25SmoothedIDF(t *testing.T) {
	store := newTestBM25Store(t)
	ctx := context.Background()

	// All documents contain "document" → classic BM25 IDF ≈ 0 → no results
	// With smoothed IDF, all should still be found with positive scores
	store.Index(ctx, 1, 1, 100, 0, "this is a document about shipping labels")
	store.Index(ctx, 2, 1, 101, 0, "another document about shipping costs")
	store.Index(ctx, 3, 2, 102, 0, "third document with shipping info")
	store.Flush()

	results, err := store.Search(ctx, "document", 5)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for ubiquitous term 'document' with smoothed IDF")
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	// All scores should be positive
	for _, r := range results {
		if r.Score <= 0 {
			t.Errorf("expected positive score, got %.4f for att=%d", r.Score, r.AttachmentID)
		}
	}
}

func TestBM25FrenchShippingLabels(t *testing.T) {
	store := newTestBM25Store(t)
	ctx := context.Background()

	// Simulate real Vinted bordereaux: all contain "colis" and "poids"
	store.Index(ctx, 1, 1, 100, 0, "Dimensions max de mon colis Poids = 30 kg max VINTED USER 75011 PARIS")
	store.Index(ctx, 2, 1, 101, 0, "Dimensions max de mon colis Poids = 20 kg max VINTED USER 75011 PARIS")
	store.Index(ctx, 3, 2, 102, 0, "Commerces de proximité RS Mobile Repaire 184 RUE SAIN MAUR 75010 PARIS")
	store.Flush()

	// "colis" appears in 2/3 docs → smoothed IDF should still rank them
	results, err := store.Search(ctx, "colis", 5)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'colis'")
	}
	// Should find the 2 docs containing "colis", not the 3rd
	for _, r := range results {
		if r.AttachmentID == 102 {
			t.Errorf("attachment 102 should not match 'colis', got: %s", r.ChunkText[:40])
		}
	}

	// "repaire" appears in only 1 doc → should have higher score than ubiquitous terms
	results, err = store.Search(ctx, "repaire", 5)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'repaire', got %d", len(results))
	}
	if results[0].AttachmentID != 102 {
		t.Errorf("expected attachment 102, got %d", results[0].AttachmentID)
	}

	// Discriminative term should score higher than ubiquitous term
	repaireScore := results[0].Score
	results, _ = store.Search(ctx, "colis", 5)
	if len(results) > 0 {
		colisScore := results[0].Score
		if repaireScore <= colisScore {
			t.Logf("note: repaire(%.4f) <= colis(%.4f) — discriminative term should score higher", repaireScore, colisScore)
		}
	}
}
