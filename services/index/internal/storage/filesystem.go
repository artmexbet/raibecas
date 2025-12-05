package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FileSystem - реализация Storage на основе файловой системы
type FileSystem struct {
	baseDir string
}

// NewFileSystem создает новое хранилище на основе файловой системы
func NewFileSystem(baseDir string) (*FileSystem, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create base directory: %w", err)
	}
	return &FileSystem{baseDir: baseDir}, nil
}

// Save сохраняет документ и возвращает относительный путь к нему
func (fs *FileSystem) Save(ctx context.Context, documentID string, reader io.Reader) (string, error) {
	// Создаем подпапку на основе первых символов documentID для лучшего распределения
	subDir := ""
	if len(documentID) >= 2 {
		subDir = documentID[:2]
	}

	dirPath := filepath.Join(fs.baseDir, subDir)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("create subdirectory: %w", err)
	}

	// Генерируем уникальное имя файла
	filename := fmt.Sprintf("%s_%s.txt", documentID, uuid.New().String()[:8])
	fullPath := filepath.Join(dirPath, filename)

	// Создаем файл
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Копируем данные
	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(fullPath) //nolint:errcheck // best effort cleanup
		return "", fmt.Errorf("write file: %w", err)
	}

	// Возвращаем относительный путь
	relPath := filepath.Join(subDir, filename)
	return relPath, nil
}

// Get возвращает reader для чтения документа
func (fs *FileSystem) Get(ctx context.Context, filePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(fs.baseDir, filePath)

	// Проверяем, что путь находится внутри baseDir (защита от path traversal)
	absBasePath, err := filepath.Abs(fs.baseDir)
	if err != nil {
		return nil, fmt.Errorf("get absolute base path: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("get absolute file path: %w", err)
	}

	if !filepath.HasPrefix(absFullPath, absBasePath) {
		return nil, fmt.Errorf("invalid file path: path traversal detected")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

// Delete удаляет документ
func (fs *FileSystem) Delete(ctx context.Context, filePath string) error {
	fullPath := filepath.Join(fs.baseDir, filePath)

	// Проверяем, что путь находится внутри baseDir
	absBasePath, err := filepath.Abs(fs.baseDir)
	if err != nil {
		return fmt.Errorf("get absolute base path: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("get absolute file path: %w", err)
	}

	if !filepath.HasPrefix(absFullPath, absBasePath) {
		return fmt.Errorf("invalid file path: path traversal detected")
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}
