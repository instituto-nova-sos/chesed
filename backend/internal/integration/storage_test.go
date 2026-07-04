//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"

	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/instituto-nova-sos/chesed/internal/storage"
)

// TestStorage boots its own MinIO container (independent from the Postgres
// harness) and exercises the real S3 wire protocol: idempotent bucket
// creation, Put → presigned GET round-trip, and the expiry contract of
// presigned URLs (docs/19-secure-development-standard.md §6).
func TestStorage(t *testing.T) {
	ctx := context.Background()

	minioC, err := tcminio.Run(ctx, "minio/minio:latest")
	require.NoError(t, err, "start minio container")
	t.Cleanup(func() { _ = minioC.Terminate(context.Background()) })

	endpoint, err := minioC.ConnectionString(ctx)
	require.NoError(t, err)

	cfg := storage.Config{
		Endpoint:  endpoint,
		Bucket:    "chesed-docs-test",
		AccessKey: minioC.Username,
		SecretKey: minioC.Password,
		UseSSL:    false,
	}

	store, err := storage.NewMinioStorage(ctx, cfg)
	require.NoError(t, err, "first construction must create the bucket")

	// Compile-time proof the implementation satisfies the service contract.
	var _ service.ObjectStorage = store

	t.Run("bucket creation is idempotent", func(t *testing.T) {
		again, err := storage.NewMinioStorage(ctx, cfg)
		require.NoError(t, err, "second construction must treat the existing bucket as success")
		require.NotNil(t, again)
	})

	t.Run("put then presigned GET round-trips bytes and content type", func(t *testing.T) {
		content := []byte("%PDF-1.7 chesed storage round-trip payload")
		key := "documents/integration/round-trip.pdf"

		err := store.Put(ctx, key, "application/pdf", int64(len(content)), bytes.NewReader(content))
		require.NoError(t, err)

		presignedURL, err := store.PresignGet(ctx, key, 15*time.Minute)
		require.NoError(t, err)

		resp, err := http.Get(presignedURL)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
		require.Equal(t, content, body)
	})

	t.Run("presigned URL carries the requested expiry", func(t *testing.T) {
		expiry := 15 * time.Minute

		presignedURL, err := store.PresignGet(ctx, "documents/integration/round-trip.pdf", expiry)
		require.NoError(t, err)

		parsed, err := url.Parse(presignedURL)
		require.NoError(t, err)
		require.Equal(t,
			strconv.Itoa(int(expiry.Seconds())),
			parsed.Query().Get("X-Amz-Expires"),
			"presigned URL must encode the configured TTL")
	})
}
