package pipeline

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/artmexbet/raibecas/services/index/internal/chunker"
	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

type embedder interface {
	Embedding(ctx context.Context, text string) ([]float64, error)
}

type chunkWriter interface {
	WriteChunks(ctx context.Context, chunks []domain.Chunk) error
}

type storageReader interface {
	Get(ctx context.Context, filePath string) (io.ReadCloser, error)
}

type Pipeline struct {
	cfg        config.Pipeline
	chunkerCfg chunker.Config
	embedder   embedder
	writer     chunkWriter
	storage    storageReader
}

func New(cfg *config.Config, emb embedder, writer chunkWriter, storage storageReader) *Pipeline {
	c := chunker.Config{
		ChunkSize:    cfg.Pipeline.ChunkSize,
		ChunkOverlap: cfg.Pipeline.ChunkOverlap,
		MaxChunks:    cfg.Pipeline.MaxChunks,
	}
	return &Pipeline{
		cfg:        cfg.Pipeline,
		chunkerCfg: c,
		embedder:   emb,
		writer:     writer,
		storage:    storage,
	}
}

func (p *Pipeline) Index(ctx context.Context, doc domain.Document) error {
	var content string

	// Если есть прямой контент (legacy API), используем его
	if doc.Content != "" {
		content = doc.Content
	} else if doc.FilePath != "" {
		// Иначе читаем из файла
		reader, err := p.storage.Get(ctx, doc.FilePath)
		if err != nil {
			return fmt.Errorf("get document file: %w", err)
		}
		defer func() {
			if closeErr := reader.Close(); closeErr != nil {
				slog.WarnContext(ctx, "failed to close reader", "err", closeErr)
			}
		}()

		// Читаем содержимое файла
		contentBytes, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("read document file: %w", err)
		}
		content = string(contentBytes)
	} else {
		return fmt.Errorf("document has neither content nor file_path")
	}

	// Очистка и нормализация текста
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("document content is empty")
	}

	// Разбиваем на чанки
	chunks := chunker.SplitText(p.chunkerCfg, content)
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks generated")
	}

	// Генерируем эмбеддинги для каждого чанка
	var result []domain.Chunk
	for _, ch := range chunks {
		embedding, err := p.embedder.Embedding(ctx, ch.Text)
		if err != nil {
			return fmt.Errorf("embed chunk %d: %w", ch.Ordinal, err)
		}

		result = append(result, domain.Chunk{
			DocumentID: doc.ID,
			Ordinal:    ch.Ordinal,
			Text:       ch.Text,
			Embedding:  embedding,
			Metadata:   mergeMetadata(doc.Metadata, ch.Metadata),
		})
	}

	// Записываем чанки в векторную БД
	if err := p.writer.WriteChunks(ctx, result); err != nil {
		return fmt.Errorf("write chunks: %w", err)
	}

	slog.InfoContext(ctx, "document indexed",
		"document_id", doc.ID,
		"chunks_count", len(result),
	)

	return nil
}

// mergeMetadata объединяет метаданные документа и чанка
func mergeMetadata(docMeta, chunkMeta map[string]string) map[string]string {
	result := make(map[string]string, len(docMeta)+len(chunkMeta))
	for k, v := range docMeta {
		result[k] = v
	}
	for k, v := range chunkMeta {
		result[k] = v
	}
	return result
}
