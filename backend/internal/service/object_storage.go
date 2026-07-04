package service

import (
	"context"
	"io"
	"time"
)

// ObjectStorage abstracts the S3-compatible object store that holds document
// binaries. It exists to satisfy docs/19-secure-development-standard.md §6
// (File Upload Security): files live in object storage, never on the
// application server filesystem, and are served exclusively through
// time-limited presigned URLs — the API never streams stored bytes directly.
// Services depend on this interface (dependency inversion, same pattern as
// CampaignRefRepository); the MinIO/S3 implementation lives in
// internal/storage.
type ObjectStorage interface {
	Put(ctx context.Context, key, contentType string, size int64, r io.Reader) error
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}
