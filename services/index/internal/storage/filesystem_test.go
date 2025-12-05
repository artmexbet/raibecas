package storage_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/storage"
)

func TestFileSystem_Save(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	fs, err := storage.NewFileSystem(tempDir)
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	ctx := context.Background()
	documentID := "test-doc-123"
	content := "This is a test document content"

	// Act
	filePath, err := fs.Save(ctx, documentID, strings.NewReader(content))

	// Assert
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if filePath == "" {
		t.Error("Save() returned empty file path")
	}

	// Verify file exists and has correct content
	reader, err := fs.Get(ctx, filePath)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer reader.Close()

	savedContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(savedContent) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(savedContent), content)
	}
}

func TestFileSystem_Get(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	fs, err := storage.NewFileSystem(tempDir)
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	ctx := context.Background()
	documentID := "test-doc-456"
	content := "Another test content"

	// Save a file first
	filePath, err := fs.Save(ctx, documentID, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Act
	reader, err := fs.Get(ctx, filePath)

	// Assert
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer reader.Close()

	retrievedContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(retrievedContent) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(retrievedContent), content)
	}
}

func TestFileSystem_Get_NonExistentFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	fs, err := storage.NewFileSystem(tempDir)
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	ctx := context.Background()

	// Act
	_, err = fs.Get(ctx, "non-existent-file.txt")

	// Assert
	if err == nil {
		t.Error("Get() should return error for non-existent file")
	}
}

func TestFileSystem_Get_PathTraversal(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	fs, err := storage.NewFileSystem(tempDir)
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	ctx := context.Background()

	// Act - try path traversal attack
	_, err = fs.Get(ctx, "../../../etc/passwd")

	// Assert
	if err == nil {
		t.Error("Get() should prevent path traversal attacks")
	}

	if !strings.Contains(err.Error(), "path traversal") {
		t.Errorf("Expected path traversal error, got: %v", err)
	}
}

func TestFileSystem_Delete(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	fs, err := storage.NewFileSystem(tempDir)
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	ctx := context.Background()
	documentID := "test-doc-789"
	content := "Content to be deleted"

	// Save a file first
	filePath, err := fs.Save(ctx, documentID, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(tempDir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatal("File should exist before deletion")
	}

	// Act
	err = fs.Delete(ctx, filePath)

	// Assert
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("File should not exist after deletion")
	}
}

func TestFileSystem_Delete_NonExistentFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	fs, err := storage.NewFileSystem(tempDir)
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	ctx := context.Background()

	// Act - delete non-existent file should not error
	err = fs.Delete(ctx, "non-existent-file.txt")

	// Assert
	if err != nil {
		t.Errorf("Delete() should not error for non-existent file, got: %v", err)
	}
}

func TestFileSystem_LargeFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	fs, err := storage.NewFileSystem(tempDir)
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}

	ctx := context.Background()
	documentID := "large-doc"

	// Create a large content (1MB)
	largeContent := bytes.Repeat([]byte("a"), 1024*1024)

	// Act
	filePath, err := fs.Save(ctx, documentID, bytes.NewReader(largeContent))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify
	reader, err := fs.Get(ctx, filePath)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer reader.Close()

	retrievedContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(retrievedContent) != len(largeContent) {
		t.Errorf("Content size mismatch: got %d, want %d", len(retrievedContent), len(largeContent))
	}
}
