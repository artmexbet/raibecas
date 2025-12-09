package ingestion_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artmexbet/raibecas/services/index/internal/domain"
	"github.com/artmexbet/raibecas/services/index/internal/ingestion"
)

// Mock implementations
type mockPipeline struct {
	indexedDocs []domain.Document
	err         error
}

func (m *mockPipeline) Index(ctx context.Context, doc domain.Document) error {
	if m.err != nil {
		return m.err
	}
	m.indexedDocs = append(m.indexedDocs, doc)
	return nil
}

type mockStorage struct {
	savedFiles map[string]string
	err        error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		savedFiles: make(map[string]string),
	}
}

func (m *mockStorage) Save(ctx context.Context, documentID string, reader io.Reader) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	filePath := "storage/" + documentID + ".txt"
	m.savedFiles[filePath] = string(content)
	return filePath, nil
}

func (m *mockStorage) Get(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}

	content, ok := m.savedFiles[filePath]
	if !ok {
		return nil, io.EOF
	}

	return io.NopCloser(strings.NewReader(content)), nil
}

func (m *mockStorage) Delete(ctx context.Context, filePath string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.savedFiles, filePath)
	return nil
}

func TestHTTPIngestor_IndexFile(t *testing.T) {
	// Setup
	mockPipe := &mockPipeline{}
	mockStore := newMockStorage()

	ingestor := ingestion.NewHTTPIngestor(mockPipe, mockStore)

	// Prepare multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	fileContent := "This is test document content"
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write([]byte(fileContent)); err != nil {
		t.Fatalf("Write file content error = %v", err)
	}

	// Add form fields
	writer.WriteField("id", "test-doc-123")
	writer.WriteField("title", "Test Document")
	writer.WriteField("source_uri", "http://example.com/doc")

	writer.Close()

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/index", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Act
	resp, err := ingestor.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify file was saved
	if len(mockStore.savedFiles) != 1 {
		t.Errorf("Expected 1 saved file, got %d", len(mockStore.savedFiles))
	}

	// Verify document was indexed
	if len(mockPipe.indexedDocs) != 1 {
		t.Fatalf("Expected 1 indexed document, got %d", len(mockPipe.indexedDocs))
	}

	indexedDoc := mockPipe.indexedDocs[0]
	if indexedDoc.ID != "test-doc-123" {
		t.Errorf("Document ID = %s, want %s", indexedDoc.ID, "test-doc-123")
	}
	if indexedDoc.Title != "Test Document" {
		t.Errorf("Document Title = %s, want %s", indexedDoc.Title, "Test Document")
	}
	if indexedDoc.FilePath == "" {
		t.Error("Document FilePath should not be empty")
	}
}

func TestHTTPIngestor_IndexFile_MissingFile(t *testing.T) {
	// Setup
	mockPipe := &mockPipeline{}
	mockStore := newMockStorage()

	ingestor := ingestion.NewHTTPIngestor(mockPipe, mockStore)

	// Create request without file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("id", "test-doc-123")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/index", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Act
	resp, err := ingestor.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHTTPIngestor_IndexFile_MissingID(t *testing.T) {
	// Setup
	mockPipe := &mockPipeline{}
	mockStore := newMockStorage()

	ingestor := ingestion.NewHTTPIngestor(mockPipe, mockStore)

	// Prepare multipart form data without ID
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/index", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Act
	resp, err := ingestor.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHTTPIngestor_IndexJSON_Legacy(t *testing.T) {
	// Setup
	mockPipe := &mockPipeline{}
	mockStore := newMockStorage()

	ingestor := ingestion.NewHTTPIngestor(mockPipe, mockStore)

	// Create JSON request
	jsonBody := `{"id":"test-doc-456","content":"Test content","title":"JSON Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/index/json", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := ingestor.Test(req)

	// Assert
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	// Verify document was indexed
	if len(mockPipe.indexedDocs) != 1 {
		t.Fatalf("Expected 1 indexed document, got %d", len(mockPipe.indexedDocs))
	}

	indexedDoc := mockPipe.indexedDocs[0]
	if indexedDoc.ID != "test-doc-456" {
		t.Errorf("Document ID = %s, want %s", indexedDoc.ID, "test-doc-456")
	}
}
