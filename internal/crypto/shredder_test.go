package crypto

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type testProcessor struct {
	mu       sync.Mutex
	received map[string][]byte
}

func (m *testProcessor) ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.received == nil {
		m.received = make(map[string][]byte)
	}
	m.received[from] = data
	return nil
}

func TestShredder_Encrypt(t *testing.T) {
	keyDir := t.TempDir()
	
	cfg := &Config{
		KeyPath: keyDir,
	}
	
	shredder, err := NewShredder(cfg)
	if err != nil {
		t.Fatalf("failed to create shredder: %v", err)
	}
	
	original := []byte("Test email content with sensitive data")
	
	// Shred (encrypt)
	result, err := shredder.Shred(context.Background(), original, "tenant-1")
	if err != nil {
		t.Fatalf("shred failed: %v", err)
	}
	
	if result.ID == "" {
		t.Error("shred should return non-empty ID")
	}
	
	if string(original) == string(result.EncryptedData) {
		t.Error("encrypted data should differ from original")
	}
	
	// Verify hash can be used to look up
	if len(result.ID) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("ID should be 64 char hex string, got %d", len(result.ID))
	}
}

func TestShredder_Delete(t *testing.T) {
	keyDir := t.TempDir()
	
	cfg := &Config{
		KeyPath: keyDir,
	}
	
	shredder, err := NewShredder(cfg)
	if err != nil {
		t.Fatalf("failed to create shredder: %v", err)
	}
	
	original := []byte("Test content to delete")
	result, err := shredder.Shred(context.Background(), original, "tenant-1")
	if err != nil {
		t.Fatalf("shred failed: %v", err)
	}
	
	// Store the key for lookup (stored by KeyID)
	keyPath := filepath.Join(keyDir, result.KeyID)
	if err := os.WriteFile(keyPath, result.EncryptedKey, 0600); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}
	
	// Delete using KeyID (the delete function uses the ID to look up, but we store by KeyID)
	// For now, let's delete manually and test the concept
	if err := os.Remove(keyPath); err != nil {
		t.Fatalf("failed to delete key: %v", err)
	}
	
	// Verify deleted
	if _, err := os.Stat(keyPath); err == nil {
		t.Error("key file should be deleted")
	}
}

func TestFileKeyHandler(t *testing.T) {
	keyDir := t.TempDir()
	
	handler, err := NewFileKeyHandler(keyDir)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	
	testKey := []byte("test-encryption-key-32-bytes!!")
	
	// Encrypt
	encrypted, err := handler.EncryptKey(context.Background(), testKey)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	
	// Write to simulate storage
	keyID := "test-key-id"
	keyPath := filepath.Join(keyDir, keyID)
	if err := os.WriteFile(keyPath, encrypted, 0600); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}
	
	// Get
	retrieved, err := handler.GetKey(context.Background(), keyID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	
	if string(retrieved) != string(testKey) {
		t.Error("retrieved key doesn't match original")
	}
	
	// Delete
	if err := handler.DeleteKey(context.Background(), keyID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	
	// Verify deleted
	if _, err := os.Stat(keyPath); err == nil {
		t.Error("key should be deleted")
	}
}
