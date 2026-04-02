package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

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
	Close() error
}

// PutOptions holds options for Put operations.
type PutOptions struct {
	ContentType string
	Metadata    map[string]string
	WORM        bool
}

// S3Config holds S3/MinIO configuration.
type S3Config struct {
	Endpoint    string
	Region      string
	AccessKey   string
	SecretKey   string
	Bucket      string
	ObjectLock  bool
	InMemory    bool
	UseSSL      bool
}

// S3Adapter implements StorageAdapter for S3/MinIO.
type S3Adapter struct {
	client *minio.Client
	bucket string
	worm   bool
}

// NewS3Adapter creates a new S3 adapter.
func NewS3Adapter(cfg *S3Config) (*S3Adapter, error) {
	wormEnabled := cfg.ObjectLock

	if cfg.InMemory {
		return &S3Adapter{
			client: nil,
			bucket: cfg.Bucket,
			worm:   wormEnabled,
		}, nil
	}

	useSSL := cfg.UseSSL
	if cfg.Endpoint == "localhost:9000" {
		useSSL = false
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	exists, err := client.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		opts := minio.MakeBucketOptions{}
		if cfg.Region != "" {
			opts.Region = cfg.Region
		}
		if err := client.MakeBucket(context.Background(), cfg.Bucket, opts); err != nil {
			return nil, err
		}
	}

	if cfg.ObjectLock {
		// Note: Object lock is set at bucket creation time
		// This requires specific bucket configuration at provisioning time
	}

	return &S3Adapter{
		client: client,
		bucket: cfg.Bucket,
		worm:   cfg.ObjectLock,
	}, nil
}

func (a *S3Adapter) Put(ctx context.Context, key string, data []byte, opts *PutOptions) error {
	if a.client == nil {
		return nil
	}

	_, err := a.client.PutObject(ctx, a.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
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

	return io.ReadAll(obj)
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

	var keys []string
	for obj := range a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix: prefix,
	}) {
		if obj.Err != nil {
			return keys, obj.Err
		}
		keys = append(keys, obj.Key)
	}
	return keys, nil
}

func (a *S3Adapter) Close() error {
	return nil
}

// MinIOAdapter is an alias for S3Adapter.
type MinIOAdapter = S3Adapter

// NewMinIOAdapter creates a new MinIO adapter.
func NewMinIOAdapter(cfg *S3Config) (*MinIOAdapter, error) {
	return NewS3Adapter(cfg)
}
