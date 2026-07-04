// Package storage provides the S3-compatible object storage implementation
// of the service.ObjectStorage interface (MinIO in dev/e2e, S3/R2 in prod —
// docs/14-deployment-strategy.md).
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds the connection settings for an S3-compatible endpoint.
type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// MinioStorage implements service.ObjectStorage against an S3-compatible
// object store using the MinIO client SDK.
type MinioStorage struct {
	client *minio.Client
	bucket string
}

// NewMinioStorage constructs the S3 client and ensures the configured bucket
// exists, creating it idempotently when missing.
func NewMinioStorage(ctx context.Context, cfg Config) (*MinioStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage.NewMinioStorage: create client for %q: %w", cfg.Endpoint, err)
	}

	if err := ensureBucket(ctx, client, cfg.Bucket); err != nil {
		return nil, fmt.Errorf("storage.NewMinioStorage: %w", err)
	}

	return &MinioStorage{client: client, bucket: cfg.Bucket}, nil
}

// Put streams the object bytes to the bucket under key with the declared
// content type.
func (s *MinioStorage) Put(ctx context.Context, key, contentType string, size int64, r io.Reader) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("storage.MinioStorage.Put: put object %q: %w", key, err)
	}
	return nil
}

// PresignGet returns a time-limited download URL for a stored object, so the
// API never serves file bytes directly (docs/19-secure-development-standard.md §6).
func (s *MinioStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("storage.MinioStorage.PresignGet: presign object %q: %w", key, err)
	}
	return u.String(), nil
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", bucket, err)
	}
	if exists {
		return nil
	}

	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		// A concurrent boot may create the bucket between the existence
		// check and MakeBucket; owning it already is success, not failure.
		if isBucketAlreadyOwned(err) {
			return nil
		}
		return fmt.Errorf("create bucket %q: %w", bucket, err)
	}
	return nil
}

func isBucketAlreadyOwned(err error) bool {
	code := minio.ToErrorResponse(err).Code
	return code == "BucketAlreadyOwnedByYou" || code == "BucketAlreadyExists"
}
