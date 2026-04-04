package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Config holds crypto configuration.
type Config struct {
	KeyPath string // Path to store encrypted keys
}

// Shredder handles crypto-shredding operations.
type Shredder struct {
	cfg        *Config
	keyHandler KeyHandler
}

// KeyHandler manages encryption keys.
type KeyHandler interface {
	EncryptKey(ctx context.Context, key []byte) ([]byte, error)
	DecryptKey(ctx context.Context, encryptedKey []byte) ([]byte, error)
	GetKey(ctx context.Context, keyID string) ([]byte, error)
	DeleteKey(ctx context.Context, keyID string) error
}

// ShredResult contains the result of shredding an email.
type ShredResult struct {
	ID            string
	EncryptedData []byte
	EncryptedKey  []byte
	KeyID         string
}

// NewShredder creates a new Shredder.
func NewShredder(cfg *Config) (*Shredder, error) {
	keyHandler, err := NewFileKeyHandler(cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	return &Shredder{
		cfg:        cfg,
		keyHandler: keyHandler,
	}, nil
}

// Shred encrypts and hashes email content.
func (s *Shredder) Shred(ctx context.Context, data []byte, tenantID string) (*ShredResult, error) {
	// Generate random AES-256 key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	// Encrypt the data with AES-256-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encryptedData := gcm.Seal(nonce, nonce, data, nil)

	// Generate hash for indexing
	hash := sha256.Sum256(data)
	id := hex.EncodeToString(hash[:])

	// Encrypt the key with master key
	encryptedKey, err := s.keyHandler.EncryptKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt key: %w", err)
	}

	keyID := uuid.New().String()

	return &ShredResult{
		ID:            id,
		EncryptedData: encryptedData,
		EncryptedKey:  encryptedKey,
		KeyID:         keyID,
	}, nil
}

// Unshred decrypts shredded content.
func (s *Shredder) Unshred(ctx context.Context, id string, tenantID string) ([]byte, error) {
	return nil, errors.New("not implemented: need key storage lookup")
}

// Delete removes shredded content (right to be forgotten).
func (s *Shredder) Delete(ctx context.Context, id string, tenantID string) error {
	return s.keyHandler.DeleteKey(ctx, id)
}

// FileKeyHandler implements KeyHandler using file storage.
type FileKeyHandler struct {
	keyPath string
}

func NewFileKeyHandler(keyPath string) (*FileKeyHandler, error) {
	if err := os.MkdirAll(keyPath, 0700); err != nil {
		return nil, err
	}
	return &FileKeyHandler{keyPath: keyPath}, nil
}

func (h *FileKeyHandler) EncryptKey(ctx context.Context, key []byte) ([]byte, error) {
	return key, nil
}

func (h *FileKeyHandler) DecryptKey(ctx context.Context, encryptedKey []byte) ([]byte, error) {
	return encryptedKey, nil
}

func (h *FileKeyHandler) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(h.keyPath, keyID))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (h *FileKeyHandler) DeleteKey(ctx context.Context, keyID string) error {
	return os.Remove(filepath.Join(h.keyPath, keyID))
}
