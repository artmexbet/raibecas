package storage

import (
	"context"
	"io"
)

// Storage - интерфейс для хранения документов
type Storage interface {
	// Save сохраняет документ и возвращает путь к нему
	Save(ctx context.Context, documentID string, reader io.Reader) (string, error)

	// Get возвращает reader для чтения документа
	Get(ctx context.Context, filePath string) (io.ReadCloser, error)

	// Delete удаляет документ
	Delete(ctx context.Context, filePath string) error
}
