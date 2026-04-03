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
	// Need multiple documents for BM25 IDF to be positive
	store.Index(ctx, 1, 1, 100, 0, "persistent chunk about dogs")
	store.Index(ctx, 2, 1, 101, 0, "another document about cats")
	store.Index(ctx, 3, 1, 102, 0, "third document about birds")
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
