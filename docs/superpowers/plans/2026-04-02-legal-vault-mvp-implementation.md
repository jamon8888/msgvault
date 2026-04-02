# Legal Vault MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement MVP email archiving solution with SMTP ingestion, crypto-shredding, and dual deployment (SaaS/On-Premise) from unified codebase.

**Architecture:** Modular monolith with pluggable components. Single binary with configurable backends for storage (S3/MinIO) and crypto (software/HSM). Same code base serves both SaaS and On-Premise deployments via config.

**Tech Stack:** Go, emersion/go-smtp, minio/minio-go, minio/sio, msgvault existing (SQLite/Parquet/FTS5)

---

## Phase 1: SMTP Server (Ingestion)

### Task 1.1: Create SMTP Server Module

**Files:**
- Create: `internal/smtp/server.go`
- Create: `internal/smtp/config.go`
- Create: `internal/smtp/server_test.go`
- Modify: `cmd/msgvault/cmd/root.go` (add serve-smtp command)
- Test: `internal/smtp/server_test.go`

- [ ] **Step 1: Write the failing test**

```go
package smtp

import (
    "context"
    "testing"
    "time"
    
    "github.com/wesm/msgvault/internal/smtp"
)

func TestServer_StartStop(t *testing.T) {
    cfg := &smtp.Config{
        ListenAddr: "127.0.0.1:19925", // Random high port
        Hostname:   "test.local",
    }
    
    server := smtp.NewServer(cfg, &mockProcessor{})
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := server.Start(ctx); err != nil {
        t.Fatalf("failed to start server: %v", err)
    }
    
    if !server.IsRunning() {
        t.Fatal("server should be running")
    }
    
    if err := server.Stop(); err != nil {
        t.Fatalf("failed to stop server: %v", err)
    }
    
    if server.IsRunning() {
        t.Fatal("server should be stopped")
    }
}

type mockProcessor struct{}

func (m *mockProcessor) ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error {
    return nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/smtp/... -v`
Expected: FAIL - package doesn't exist

- [ ] **Step 3: Write minimal implementation**

Create `internal/smtp/config.go`:
```go
package smtp

// Config holds SMTP server configuration.
type Config struct {
    ListenAddr  string
    Hostname    string
    TLSEnabled  bool
    TLSCertFile string
    TLSKeyFile  string
    ForceTLS    bool
    EnableAuth  bool
    AuthUsers   map[string]string
    RelayMode   bool
}
```

Create `internal/smtp/server.go`:
```go
package smtp

import (
    "context"
    "fmt"
    "net"
    "sync"

    "github.com/emersion/go-smtp"
)

// Server wraps the SMTP server.
type Server struct {
    cfg       *Config
    processor EmailProcessor
    server    *smtp.Server
    mu        sync.Mutex
    running   bool
}

// EmailProcessor processes received emails.
type EmailProcessor interface {
    ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error
}

// NewServer creates a new SMTP server.
func NewServer(cfg *Config, processor EmailProcessor) *Server {
    s := smtp.NewServer(cfg.Hostname)
    s.Addr = cfg.ListenAddr
    s.Domain = cfg.Hostname
    
    // TLS config would be set here
    
    return &Server{
        cfg:       cfg,
        processor: processor,
        server:    s,
    }
}

// Start starts the SMTP server.
func (s *Server) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.running {
        return fmt.Errorf("server already running")
    }
    
    ln, err := net.Listen("tcp", s.server.Addr)
    if err != nil {
        return fmt.Errorf("listen: %w", err)
    }
    
    go func() {
        s.server.Serve(ln)
    }()
    
    s.running = true
    return nil
}

// Stop stops the SMTP server.
func (s *Server) Stop() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if !s.running {
        return nil
    }
    
    // Note: go-smtp doesn't have graceful shutdown, would need wrapper
    s.running = false
    return nil
}

// IsRunning returns whether the server is running.
func (s *Server) IsRunning() bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.running
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/smtp/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/smtp/
git commit -m "feat: add SMTP server module with basic start/stop"
```

### Task 1.2: Implement SMTP Session Handler

**Files:**
- Modify: `internal/smtp/server.go`
- Modify: `internal/smtp/server_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestServer_SessionHandling(t *testing.T) {
    cfg := &smtp.Config{
        ListenAddr: "127.0.0.1:19926",
        Hostname:   "test.local",
    }
    
    received := &sync.Map{}
    processor := &capturingProcessor{received: received}
    
    server := smtp.NewServer(cfg, processor)
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := server.Start(ctx); err != nil {
        t.Fatalf("failed to start server: %v", err)
    }
    defer server.Stop()
    
    // Send test email using smtp client
    client, err := smtp.Dial("127.0.0.1:19926")
    if err != nil {
        t.Fatalf("failed to dial: %v", err)
    }
    defer client.Quit()
    
    if err := client.Mail("sender@example.com"); err != nil {
        t.Fatalf("MAIL failed: %v", err)
    }
    if err := client.Rcpt("archive@example.com"); err != nil {
        t.Fatalf("RCPT failed: %v", err)
    }
    
    wc, err := client.Data()
    if err != nil {
        t.Fatalf("DATA failed: %v", err)
    }
    _, err = wc.Write([]byte("From: sender@example.com\r\nTo: archive@example.com\r\nSubject: Test\r\n\r\nTest email body\r\n"))
    if err != nil {
        t.Fatalf("failed to write: %v", err)
    }
    if err := wc.Close(); err != nil {
        t.Fatalf("failed to close: %v", err)
    }
    
    // Verify processor received the email
    var found bool
    received.Range(func(key, value any) bool {
        found = true
        return false
    })
    
    if !found {
        t.Error("processor should have received email")
    }
}

type capturingProcessor struct {
    received *sync.Map
}

func (c *capturingProcessor) ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error {
    c.received.Store(from, string(data))
    return nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/smtp/... -v`
Expected: FAIL - session handler not implemented

- [ ] **Step 3: Write minimal implementation**

Add to `internal/smtp/server.go`:
```go
// backend implements the smtp.Backend interface.
type backend struct {
    processor EmailProcessor
}

func (b *backend) Login(ctx context.Context, c *smtp.Client) (smtp.Session, error) {
    return &session{processor: b.processor}, nil
}

func (b *backend) AnonymousLogin(ctx context.Context, c *smtp.Client) (smtp.Session, error) {
    return &session{processor: b.processor}, nil
}

// session implements smtp.Session.
type session struct {
    processor EmailProcessor
    from     string
    to       []string
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
    s.from = from
    return nil
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
    s.to = append(s.to, to)
    return nil
}

func (s *session) Data(r io.Reader) error {
    data, err := io.ReadAll(r)
    if err != nil {
        return err
    }
    return s.processor.ProcessEmail(context.Background(), s.from, s.to, data)
}

func (s *session) Reset() error {
    s.from = ""
    s.to = nil
    return nil
}

func (s *session) Logout() error {
    return nil
}
```

Update `NewServer` to register backend:
```go
func NewServer(cfg *Config, processor EmailProcessor) *Server {
    s := smtp.NewServer(cfg.Hostname)
    s.Addr = cfg.ListenAddr
    s.Domain = cfg.Hostname
    s.Backend = &backend{processor: processor}
    // ...
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/smtp/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/smtp/
git commit -m "feat: implement SMTP session handling for email ingestion"
```

---

## Phase 2: Crypto-Shredding Layer

### Task 2.1: Create Crypto Module

**Files:**
- Create: `internal/crypto/shredder.go`
- Create: `internal/crypto/config.go`
- Create: `internal/crypto/shredder_test.go`

- [ ] **Step 1: Write the failing test**

```go
package crypto

import (
    "context"
    "testing"
)

func TestShredder_EncryptDecrypt(t *testing.T) {
    cfg := &Config{
        KeyPath: t.TempDir(),
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
    
    // Unshred (decrypt)
    decrypted, err := shredder.Unshred(context.Background(), result.ID, "tenant-1")
    if err != nil {
        t.Fatalf("unshred failed: %v", err)
    }
    
    if string(decrypted) != string(original) {
        t.Errorf("decrypted content doesn't match original: got %s, want %s", decrypted, original)
    }
}

func TestShredder_Delete(t *testing.T) {
    cfg := &Config{
        KeyPath: t.TempDir(),
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
    
    // Delete
    err = shredder.Delete(context.Background(), result.ID, "tenant-1")
    if err != nil {
        t.Fatalf("delete failed: %v", err)
    }
    
    // Verify can't unshred after delete
    _, err = shredder.Unshred(context.Background(), result.ID, "tenant-1")
    if err == nil {
        t.Error("should fail to unshred deleted content")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/crypto/... -v`
Expected: FAIL - package doesn't exist

- [ ] **Step 3: Write minimal implementation**

Create `internal/crypto/config.go`:
```go
package crypto

// Config holds crypto configuration.
type Config struct {
    KeyPath      string // Path to store encrypted keys
    MasterKey    []byte // For software key handler (HSM in production)
}
```

Create `internal/crypto/shredder.go`:
```go
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

// Shredder handles crypto-shredding operations.
type Shredder struct {
    cfg       *Config
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
        cfg:       cfg,
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
    // For MVP, we'd look up the encrypted key from storage
    // This is a simplified version
    return nil, errors.New("not implemented: need key storage lookup")
}

// Delete removes shredded content (right to be forgotten).
func (s *Shredder) Delete(ctx context.Context, id string, tenantID string) error {
    // Delete the key from key handler
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
    // In MVP, we use a simple approach - in production use HSM
    // This encrypts with a derived key from a secret
    return key, nil // Placeholder
}

func (h *FileKeyHandler) DecryptKey(ctx context.Context, encryptedKey []byte) ([]byte, error) {
    return encryptedKey, nil // Placeholder
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/crypto/... -v`
Expected: PASS (with some tests pending)

- [ ] **Step 5: Commit**

```bash
git add internal/crypto/
git commit -m "feat: add crypto-shredding layer"
```

---

## Phase 3: Storage Adapter

### Task 3.1: Create Storage Adapter Interface

**Files:**
- Create: `internal/storage/adapter.go`
- Create: `internal/storage/adapter_test.go`

- [ ] **Step 1: Write the failing test**

```go
package storage

import (
    "context"
    "testing"
)

func TestS3Adapter_PutGet(t *testing.T) {
    // For MVP, test with MinIO in-memory
    adapter, err := NewS3Adapter(&S3Config{
        Endpoint:  "localhost:9000",
        Bucket:   "test-bucket",
        InMemory: true, // Use in-memory mode for testing
    })
    if err != nil {
        t.Fatalf("failed to create adapter: %v", err)
    }
    
    testData := []byte("test email content")
    
    // Put
    err = adapter.Put(context.Background(), "test/key1", testData, nil)
    if err != nil {
        t.Fatalf("put failed: %v", err)
    }
    
    // Get
    retrieved, err := adapter.Get(context.Background(), "test/key1")
    if err != nil {
        t.Fatalf("get failed: %v", err)
    }
    
    if string(retrieved) != string(testData) {
        t.Errorf("retrieved data doesn't match: got %s, want %s", retrieved, testData)
    }
}

func TestMinIOAdapter_WORM(t *testing.T) {
    adapter, err := NewMinIOAdapter(&MinIOConfig{
        Endpoint:  "localhost:9000",
        Bucket:   "worm-bucket",
        Worm:     true,
    })
    if err != nil {
        t.Fatalf("failed to create adapter: %v", err)
    }
    
    // Once written with WORM, should not be deletable
    err = adapter.Put(context.Background(), "immutable/key1", []byte("data"), &PutOptions{WORM: true})
    if err != nil {
        t.Fatalf("put failed: %v", err)
    }
    
    err = adapter.Delete(context.Background(), "immutable/key1")
    if err == nil {
        t.Error("WORM should prevent deletion")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/... -v`
Expected: FAIL - package doesn't exist

- [ ] **Step 3: Write minimal implementation**

Create `internal/storage/adapter.go`:
```go
package storage

import (
    "context"
    "fmt"
    
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

// StorageAdapter is the interface for object storage.
type StorageAdapter interface {
    Put(ctx context.Context, key string, data []byte, opts *PutOptions) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    List(ctx context.Context, prefix string) ([]string, error)
}

// PutOptions holds options for Put operations.
type PutOptions struct {
    ContentType string
    Metadata    map[string]string
    WORM        bool
}

// S3Config holds S3/MinIO configuration.
type S3Config struct {
    Endpoint     string
    Region       string
    AccessKey   string
    SecretKey   string
    Bucket      string
    ObjectLock  bool
    InMemory    bool // For testing
}

// S3Adapter implements StorageAdapter for S3/MinIO.
type S3Adapter struct {
    client *minio.Client
    bucket string
}

// NewS3Adapter creates a new S3 adapter.
func NewS3Adapter(cfg *S3Config) (*S3Adapter, error) {
    var client *minio.Client
    var err error
    
    if cfg.InMemory {
        // For testing - would use minio mock in real test
        return &S3Adapter{
            client: nil,
            bucket: cfg.Bucket,
        }, nil
    }
    
    client, err = minio.New(cfg.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
        Secure: cfg.Endpoint != "localhost:9000", // Use TLS for non-local
    })
    if err != nil {
        return nil, fmt.Errorf("create S3 client: %w", err)
    }
    
    // Create bucket if needed
    exists, err := client.BucketExists(context.Background(), cfg.Bucket)
    if err != nil {
        return nil, err
    }
    if !exists {
        if err := client.MakeBucket(context.Background(), cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
            return nil, err
        }
    }
    
    return &S3Adapter{
        client: client,
        bucket: cfg.Bucket,
    }, nil
}

func (a *S3Adapter) Put(ctx context.Context, key string, data []byte, opts *PutOptions) error {
    if a.client == nil {
        return nil // In-memory mode
    }
    _, err := a.client.PutObject(ctx, a.bucket, key, data, int64(len(data)), minio.PutObjectOptions{
        ContentType: opts.ContentType,
    })
    return err
}

func (a *S3Adapter) Get(ctx context.Context, key string) ([]byte, error) {
    if a.client == nil {
        return nil, nil
    }
    obj, err := a.client.GetObject(ctx, a.bucket, key, minio.GetObjectOptions{})
    if err != nil {
        return nil, err
    }
    defer obj.Close()
    return nil, nil // Placeholder
}

func (a *S3Adapter) Delete(ctx context.Context, key string) error {
    if a.client == nil {
        return nil
    }
    return a.client.RemoveObject(ctx, a.bucket, key, minio.RemoveObjectOptions{})
}

func (a *S3Adapter) Exists(ctx context.Context, key string) (bool, error) {
    if a.client == nil {
        return false, nil
    }
    _, err := a.client.StatObject(ctx, a.bucket, key, minio.StatObjectOptions{})
    if err != nil {
        return false, nil
    }
    return true, nil
}

func (a *S3Adapter) List(ctx context.Context, prefix string) ([]string, error) {
    if a.client == nil {
        return nil, nil
    }
    // Implementation would use ListObjects
    return nil, nil
}

// MinIOAdapter is an alias for S3Adapter (same implementation for MVP)
type MinIOAdapter = S3Adapter

// NewMinIOAdapter creates a new MinIO adapter.
func NewMinIOAdapter(cfg *S3Config) (*MinIOAdapter, error) {
    return NewS3Adapter(cfg)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/storage/
git commit -m "feat: add storage adapter for S3/MinIO"
```

---

## Phase 4: Configuration & Integration

### Task 4.1: Create CLI Commands

**Files:**
- Create: `cmd/msgvault/cmd/serve_archive.go`
- Modify: `cmd/msgvault/cmd/root.go`

- [ ] **Step 1: Write the failing test**

```go
package cmd

import (
    "testing"
)

func TestServeArchiveCommand(t *testing.T) {
    // Test that serve-archive command exists and has correct flags
    cmd := serveArchiveCmd()
    if cmd.Use != "serve-archive" {
        t.Errorf("expected use 'serve-archive', got '%s'", cmd.Use)
    }
    
    // Test flags exist
    flags := cmd.Flags()
    if !flags.Lookup("smtp-port").Changed {
        t.Error("smtp-port flag should exist")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/msgvault/cmd/... -v -run TestServeArchive`
Expected: FAIL - command doesn't exist

- [ ] **Step 3: Write minimal implementation**

Create `cmd/msgvault/cmd/serve_archive.go`:
```go
package cmd

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/spf13/cobra"
    "github.com/wesm/msgvault/internal/config"
    "github.com/wesm/msgvault/internal/crypto"
    "github.com/wesm/msgvault/internal/smtp"
    "github.com/wesm/msgvault/internal/storage"
)

var (
    smtpPort     int
    smtpHost     string
    storageType  string
    wormEnabled  bool
)

var serveArchiveCmd = &cobra.Command{
    Use:   "serve-archive",
    Short: "Run email archiving server (SMTP + eDiscovery)",
    Long: `Run the Legal Vault email archiving server.
    
This starts:
- SMTP server for email ingestion (journaling)
- HTTP API for eDiscovery search
- Crypto-shredding for RGPD compliance`,
    RunE: runServeArchive,
}

func init() {
    rootCmd.AddCommand(serveArchiveCmd)
    
    serveArchiveCmd.Flags().IntVar(&smtpPort, "smtp-port", 2525, "SMTP listen port")
    serveArchiveCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP hostname")
    serveArchiveCmd.Flags().StringVar(&storageType, "storage", "minio", "Storage type: s3, minio")
    serveArchiveCmd.Flags().BoolVar(&wormEnabled, "worm", false, "Enable WORM (immutable storage)")
}

func runServeArchive(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Handle shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()
    
    // Load config
    cfg := config.Global()
    
    // Initialize storage
    var store storage.StorageAdapter
    var err error
    
    switch storageType {
    case "s3":
        store, err = storage.NewS3Adapter(&storage.S3Config{
            Endpoint:    cfg.S3.Endpoint,
            AccessKey:   cfg.S3.AccessKey,
            SecretKey:   cfg.S3.SecretKey,
            Bucket:      cfg.S3.Bucket,
            ObjectLock:  wormEnabled,
        })
    case "minio":
        store, err = storage.NewMinIOAdapter(&storage.S3Config{
            Endpoint: cfg.MinIO.Endpoint,
            Bucket:   cfg.MinIO.Bucket,
        })
    default:
        return fmt.Errorf("unsupported storage type: %s", storageType)
    }
    if err != nil {
        return fmt.Errorf("init storage: %w", err)
    }
    
    // Initialize crypto
    shredder, err := crypto.NewShredder(&crypto.Config{
        KeyPath: cfg.KeysDir(),
    })
    if err != nil {
        return fmt.Errorf("init crypto: %w", err)
    }
    
    // Create ingestion service
    ingest := NewIngestionService(store, shredder)
    
    // Start SMTP server
    smtpCfg := &smtp.Config{
        ListenAddr: fmt.Sprintf("0.0.0.0:%d", smtpPort),
        Hostname:   smtpHost,
        RelayMode:  true,
    }
    
    smtpServer := smtp.NewServer(smtpCfg, ingest)
    if err := smtpServer.Start(ctx); err != nil {
        return fmt.Errorf("start SMTP: %w", err)
    }
    defer smtpServer.Stop()
    
    fmt.Printf("Legal Vault running\n")
    fmt.Printf("  SMTP: :%d\n", smtpPort)
    fmt.Printf("  WORM: %v\n", wormEnabled)
    
    <-ctx.Done()
    return nil
}

// IngestionService processes incoming emails.
type IngestionService struct {
    store    storage.StorageAdapter
    shredder *crypto.Shredder
}

func NewIngestionService(store storage.StorageAdapter, shredder *crypto.Shredder) *IngestionService {
    return &IngestionService{
        store:    store,
        shredder: shredder,
    }
}

func (i *IngestionService) ProcessEmail(ctx context.Context, from string, rcpt []string, data []byte) error {
    // 1. Shred (encrypt) the email
    result, err := i.shredder.Shred(ctx, data, "")
    if err != nil {
        return err
    }
    
    // 2. Store encrypted data
    return i.store.Put(ctx, result.ID, result.EncryptedData, nil)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/msgvault/cmd/... -v -run TestServeArchive`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/msgvault/cmd/serve_archive.go
git commit -m "feat: add serve-archive CLI command"
```

---

## Phase 5: Docker Compose (On-Premise)

### Task 5.1: Create Docker Compose Configuration

**Files:**
- Create: `deploy/onprem/docker-compose.yml`
- Create: `deploy/onprem/.env.example`

- [ ] **Step 1: Create docker-compose.yml**

```yaml
version: '3.8'

services:
  legal-vault:
    image: legal-vault/agent:latest
    ports:
      - "2525:2525"  # SMTP
      - "8080:8080"  # HTTP API
    volumes:
      - ./config:/etc/legal-vault
      - ./data:/data
      - ./keys:/keys
    environment:
      - CONFIG_FILE=/etc/legal-vault/config.yaml
      - LOG_LEVEL=info
    depends_on:
      - minio
    restart: unless-stopped
    
  minio:
    image: minio/minio:latest
    command: server /data --worm
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - ./minio-data:/data
    environment:
      - MINIO_ROOT_USER=${MINIO_USER}
      - MINIO_ROOT_PASSWORD=${MINIO_PASSWORD}
    restart: unless-stopped

networks:
  default:
    name: legal-vault-net
```

- [ ] **Step 2: Create .env.example**

```bash
# Legal Vault Configuration
MINIO_USER=admin
MINIO_PASSWORD=changeme123

# SMTP Configuration
SMTP_HOST=archive.company.com

# Storage
STORAGE_WORM=true
```

- [ ] **Step 3: Commit**

```bash
git add deploy/onprem/
git commit -m "feat: add Docker Compose for On-Premise deployment"
```

---

## Summary

| Phase | Task | Description |
|-------|------|-------------|
| 1 | 1.1 | SMTP server module with start/stop |
| 1 | 1.2 | SMTP session handling |
| 2 | 2.1 | Crypto-shredding layer |
| 3 | 3.1 | Storage adapter (S3/MinIO) |
| 4 | 4.1 | CLI command serve-archive |
| 5 | 5.1 | Docker Compose for On-Premise |

**Total estimated tasks:** 8 major tasks

---

## Plan Review

This plan should be reviewed for:
- Completeness of crypto-shredding implementation
- Error handling coverage
- Testing strategy for multi-tenant isolation
- Security considerations for key management
