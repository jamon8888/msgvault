package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/wesm/msgvault/internal/crypto"
	"github.com/wesm/msgvault/internal/smtp"
	"github.com/wesm/msgvault/internal/storage"
)

var (
	smtpPort    int
	smtpHost   string
	storageType string
	wormEnabled bool
	minioEndpoint string
	minioBucket   string
	minioDataPath string
	s3Endpoint   string
	s3Bucket     string
)

var serveArchiveCmd = &cobra.Command{
	Use:   "serve-archive",
	Short: "Run email archiving server (SMTP + eDiscovery)",
	Long: `Run the Legal Vault email archiving server.
    
This starts:
- SMTP server for email ingestion (journaling)
- HTTP API for eDiscovery search (future)
- Crypto-shredding for RGPD compliance

The server listens on port 2525 by default for incoming SMTP connections.
Configure your email server to forward journaled emails to this port.`,
	RunE: runServeArchive,
}

func init() {
	rootCmd.AddCommand(serveArchiveCmd)

	serveArchiveCmd.Flags().IntVar(&smtpPort, "smtp-port", 2525, "SMTP listen port")
	serveArchiveCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "SMTP hostname (required)")
	serveArchiveCmd.Flags().StringVar(&storageType, "storage", "minio", "Storage type: s3, minio")
	serveArchiveCmd.Flags().BoolVar(&wormEnabled, "worm", false, "Enable WORM (immutable storage)")
	
	// MinIO options
	serveArchiveCmd.Flags().StringVar(&minioEndpoint, "minio-endpoint", "localhost:9000", "MinIO endpoint")
	serveArchiveCmd.Flags().StringVar(&minioBucket, "minio-bucket", "archives", "MinIO bucket")
	serveArchiveCmd.Flags().StringVar(&minioDataPath, "minio-data-path", "/data/archives", "MinIO data path (on-premise)")
	
	// S3 options
	serveArchiveCmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", "", "S3 endpoint (for cloud)")
	serveArchiveCmd.Flags().StringVar(&s3Bucket, "s3-bucket", "", "S3 bucket (for cloud)")
}

func runServeArchive(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Validate required fields
	if smtpHost == "" {
		return fmt.Errorf("--smtp-host is required")
	}

	// Initialize storage
	var store storage.StorageAdapter
	var err error

	switch storageType {
	case "minio":
		store, err = storage.NewMinIOAdapter(&storage.S3Config{
			Endpoint:   minioEndpoint,
			Bucket:     minioBucket,
			ObjectLock: wormEnabled,
		})
	case "s3":
		if s3Endpoint == "" || s3Bucket == "" {
			return fmt.Errorf("s3-endpoint and s3-bucket required for s3 storage")
		}
		store, err = storage.NewS3Adapter(&storage.S3Config{
			Endpoint:    s3Endpoint,
			Bucket:      s3Bucket,
			ObjectLock:  wormEnabled,
			UseSSL:      true,
		})
	default:
		return fmt.Errorf("unsupported storage type: %s", storageType)
	}
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer store.Close()

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

	smtpServer, err := smtp.NewServer(smtpCfg, ingest)
	if err != nil {
		return fmt.Errorf("create SMTP server: %w", err)
	}

	if err := smtpServer.Start(); err != nil {
		return fmt.Errorf("start SMTP: %w", err)
	}
	defer smtpServer.Stop()

	fmt.Println("Legal Vault Archive Server")
	fmt.Println("============================")
	fmt.Printf("  SMTP: :%d (hostname: %s)\n", smtpPort, smtpHost)
	fmt.Printf("  Storage: %s\n", storageType)
	fmt.Printf("  WORM: %v\n", wormEnabled)
	fmt.Println("\nServer running. Press Ctrl+C to stop.")

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
		return fmt.Errorf("shred failed: %w", err)
	}

	// 2. Store encrypted data
	return i.store.Put(ctx, result.ID, result.EncryptedData, nil)
}
