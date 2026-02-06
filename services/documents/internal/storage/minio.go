package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/artmexbet/raibecas/services/documents/internal/config"
)

const (
	contentTypeMarkdown = "text/markdown"
)

// MinIOStorage implements Storage interface using MinIO
type MinIOStorage struct {
	client *minio.Client
	bucket string
	logger *slog.Logger
}

// NewMinIOStorage creates a new MinIO storage instance
func NewMinIOStorage(cfg config.MinIOConfig, logger *slog.Logger) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIOStorage{
		client: client,
		bucket: cfg.Bucket,
		logger: logger,
	}, nil
}

// EnsureBucket ensures the storage bucket exists
func (s *MinIOStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket existence: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		s.logger.InfoContext(ctx, "created minio bucket", "bucket", s.bucket)
	}

	return nil
}

// SaveDocument saves document content and returns the storage path
func (s *MinIOStorage) SaveDocument(ctx context.Context, documentID uuid.UUID, version int, content []byte) (string, error) {
	path := s.buildPath(documentID, version)
	reader := bytes.NewReader(content)

	_, err := s.client.PutObject(ctx, s.bucket, path, reader, int64(len(content)), minio.PutObjectOptions{
		ContentType: contentTypeMarkdown,
	})
	if err != nil {
		return "", fmt.Errorf("save document to minio: %w", err)
	}

	s.logger.InfoContext(ctx, "saved document to minio",
		"document_id", documentID,
		"version", version,
		"path", path,
		"size", len(content),
	)

	return path, nil
}

// GetDocument retrieves document content by path
func (s *MinIOStorage) GetDocument(ctx context.Context, path string) ([]byte, error) {
	reader, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get document from minio: %w", err)
	}
	defer reader.Close() //nolint:errcheck

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read document content: %w", err)
	}

	return content, nil
}

// GetDocumentReader returns a reader for streaming document content
func (s *MinIOStorage) GetDocumentReader(ctx context.Context, path string) (io.ReadCloser, error) {
	reader, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get document reader from minio: %w", err)
	}

	return reader, nil
}

// DeleteDocument deletes a document by path
func (s *MinIOStorage) DeleteDocument(ctx context.Context, path string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete document from minio: %w", err)
	}

	s.logger.InfoContext(ctx, "deleted document from minio", "path", path)
	return nil
}

// ListVersions lists all versions for a document
func (s *MinIOStorage) ListVersions(ctx context.Context, documentID uuid.UUID) ([]string, error) {
	prefix := fmt.Sprintf("%s/", documentID.String())

	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var versions []string
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("list objects: %w", object.Err)
		}
		versions = append(versions, object.Key)
	}

	return versions, nil
}

// buildPath constructs the storage path for a document version
func (s *MinIOStorage) buildPath(documentID uuid.UUID, version int) string {
	return fmt.Sprintf("%s/v%d.md", documentID.String(), version)
}
