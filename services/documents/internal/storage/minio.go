package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/artmexbet/raibecas/services/documents/internal/config"
)

const (
	contentTypeMarkdown = "text/markdown"
	coverPresignedTTL   = 24 * time.Hour

	// defaultRegion is the default region for MinIO (single-server deployments).
	// Setting it explicitly prevents the SDK from issuing a GET ?location= request.
	defaultRegion = "us-east-1"
)

// MinIOStorage implements Storage interface using MinIO
type MinIOStorage struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
	logger        *slog.Logger
	tracer        trace.Tracer
}

// NewMinIOStorage creates a new MinIO storage instance
func NewMinIOStorage(cfg config.MinIOConfig, logger *slog.Logger, tracer trace.Tracer) (*MinIOStorage, error) {
	client, err := newMinIOClient(cfg.Endpoint, cfg.UseSSL, cfg.AccessKey, cfg.SecretKey, "")
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	presignClient := client
	if cfg.PublicEndpoint != "" {
		// Create a separate client that points to the public endpoint.
		// We set Region explicitly so the SDK does NOT issue a
		// GET /{bucket}/?location= probe (which would fail because the
		// public endpoint is unreachable from inside the Docker network).
		presignClient, err = newMinIOClient(cfg.PublicEndpoint, cfg.UseSSL, cfg.AccessKey, cfg.SecretKey, defaultRegion)
		if err != nil {
			return nil, fmt.Errorf("create minio presign client: %w", err)
		}
	}

	return &MinIOStorage{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		logger:        logger,
		tracer:        tracer,
	}, nil
}

// EnsureBucket ensures the storage bucket exists
func (s *MinIOStorage) EnsureBucket(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "minio.EnsureBucket",
		trace.WithAttributes(attribute.String("minio.bucket", s.bucket)),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "check bucket existence failed")
		return fmt.Errorf("check bucket existence: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "create bucket failed")
			return fmt.Errorf("create bucket: %w", err)
		}
		span.SetAttributes(attribute.Bool("minio.bucket_created", true))
		s.logger.InfoContext(ctx, "created minio bucket", "bucket", s.bucket)
	}

	return nil
}

// SaveDocument saves document content and returns the storage path
func (s *MinIOStorage) SaveDocument(ctx context.Context, documentID uuid.UUID, version int, content []byte) (string, error) {
	path := s.buildPath(documentID, version)

	ctx, span := s.tracer.Start(ctx, "minio.SaveDocument",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.object_key", path),
			attribute.String("document.id", documentID.String()),
			attribute.Int("document.version", version),
			attribute.Int("minio.object_size", len(content)),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	reader := bytes.NewReader(content)

	_, err := s.client.PutObject(ctx, s.bucket, path, reader, int64(len(content)), minio.PutObjectOptions{
		ContentType: contentTypeMarkdown,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save document failed")
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
	ctx, span := s.tracer.Start(ctx, "minio.GetDocument",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.object_key", path),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	reader, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get object failed")
		return nil, fmt.Errorf("get document from minio: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	content, err := io.ReadAll(reader)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "read content failed")
		return nil, fmt.Errorf("read document content: %w", err)
	}

	span.SetAttributes(attribute.Int("minio.object_size", len(content)))
	return content, nil
}

// GetDocumentReader returns a reader for streaming document content
func (s *MinIOStorage) GetDocumentReader(ctx context.Context, path string) (io.ReadCloser, error) {
	ctx, span := s.tracer.Start(ctx, "minio.GetDocumentReader",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.object_key", path),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	reader, err := s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get document reader failed")
		return nil, fmt.Errorf("get document reader from minio: %w", err)
	}

	return reader, nil
}

// DeleteDocument deletes a document by path
func (s *MinIOStorage) DeleteDocument(ctx context.Context, path string) error {
	ctx, span := s.tracer.Start(ctx, "minio.DeleteDocument",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.object_key", path),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	if err := s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete document failed")
		return fmt.Errorf("delete document from minio: %w", err)
	}

	s.logger.InfoContext(ctx, "deleted document from minio", "path", path)
	return nil
}

// ListVersions lists all versions for a document
func (s *MinIOStorage) ListVersions(ctx context.Context, documentID uuid.UUID) ([]string, error) {
	prefix := fmt.Sprintf("%s/", documentID.String())

	ctx, span := s.tracer.Start(ctx, "minio.ListVersions",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.prefix", prefix),
			attribute.String("document.id", documentID.String()),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var versions []string
	for object := range objectCh {
		if object.Err != nil {
			span.RecordError(object.Err)
			span.SetStatus(codes.Error, "list objects failed")
			return nil, fmt.Errorf("list objects: %w", object.Err)
		}
		versions = append(versions, object.Key)
	}

	span.SetAttributes(attribute.Int("minio.versions_count", len(versions)))
	return versions, nil
}

// buildPath constructs the storage path for a document version
func (s *MinIOStorage) buildPath(documentID uuid.UUID, version int) string {
	return fmt.Sprintf("%s/v%d.md", documentID.String(), version)
}

// SaveCover saves a cover image and returns the storage path
func (s *MinIOStorage) SaveCover(ctx context.Context, documentID uuid.UUID, data []byte, contentType string) (string, error) {
	ext := extensionByContentType(contentType)
	path := fmt.Sprintf("covers/%s%s", documentID.String(), ext)

	ctx, span := s.tracer.Start(ctx, "minio.SaveCover",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.object_key", path),
			attribute.String("document.id", documentID.String()),
			attribute.String("minio.content_type", contentType),
			attribute.Int("minio.object_size", len(data)),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	reader := bytes.NewReader(data)

	_, err := s.client.PutObject(ctx, s.bucket, path, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save cover failed")
		return "", fmt.Errorf("save cover to minio: %w", err)
	}

	s.logger.InfoContext(ctx, "saved cover to minio",
		"document_id", documentID,
		"path", path,
		"size", len(data),
	)

	return path, nil
}

// GetCoverPresignedURL generates a presigned GET URL for a cover image
func (s *MinIOStorage) GetCoverPresignedURL(ctx context.Context, path string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "minio.GetCoverPresignedURL",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.object_key", path),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	presignedURL, err := s.presignClient.PresignedGetObject(ctx, s.bucket, path, coverPresignedTTL, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate presigned url failed")
		return "", fmt.Errorf("generate presigned url for cover: %w", err)
	}

	return presignedURL.String(), nil
}

// DeleteCover deletes a cover image by path
func (s *MinIOStorage) DeleteCover(ctx context.Context, path string) error {
	ctx, span := s.tracer.Start(ctx, "minio.DeleteCover",
		trace.WithAttributes(
			attribute.String("minio.bucket", s.bucket),
			attribute.String("minio.object_key", path),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	if err := s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete cover failed")
		return fmt.Errorf("delete cover from minio: %w", err)
	}
	s.logger.InfoContext(ctx, "deleted cover from minio", "path", path)
	return nil
}

// extensionByContentType returns file extension for a given MIME type
func extensionByContentType(ct string) string {
	switch strings.ToLower(ct) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func newMinIOClient(endpoint string, useSSL bool, accessKey, secretKey, region string) (*minio.Client, error) {
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	}
	if region != "" {
		opts.Region = region
		opts.BucketLookup = minio.BucketLookupPath
	}
	return minio.New(endpoint, opts)
}
