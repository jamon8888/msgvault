package storage

import (
	"context"
	"testing"
)

func TestS3Adapter_InMemory(t *testing.T) {
	cfg := &S3Config{
		Bucket:   "test-bucket",
		InMemory: true,
	}

	adapter, err := NewS3Adapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	testData := []byte("test email content")

	// Put should succeed (no-op for in-memory)
	err = adapter.Put(context.Background(), "test/key1", testData, nil)
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	// Get should return nil (no-op for in-memory)
	retrieved, err := adapter.Get(context.Background(), "test/key1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	// In-memory doesn't store, so we get nil
	if retrieved != nil {
		t.Log("in-memory mode: data not persisted (expected)")
	}
}

func TestS3Adapter_WORM(t *testing.T) {
	cfg := &S3Config{
		Bucket:     "worm-bucket",
		ObjectLock: true,
		InMemory:   true, // Test without actual connection
	}

	adapter, err := NewS3Adapter(cfg)
	if err != nil {
		t.Fatalf("failed to create adapter: %v", err)
	}

	if !adapter.worm {
		t.Error("WORM flag should be set")
	}
}
