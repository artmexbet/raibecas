package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// NATS subjects for document service communication
const (
	SubjectDocumentsList   = "documents.list"
	SubjectDocumentsGet    = "documents.get"
	SubjectDocumentsCreate = "documents.create"
	SubjectDocumentsUpdate = "documents.update"
	SubjectDocumentsDelete = "documents.delete"

	// Default timeout for NATS requests
	defaultTimeout = 5 * time.Second
)

// NATSDocumentConnector implements server.DocumentServiceConnector using NATS for communication
type NATSDocumentConnector struct {
	conn    *nats.Conn
	timeout time.Duration
}

// NewNATSDocumentConnector creates a new NATS-based document service connector
func NewNATSDocumentConnector(conn *nats.Conn, timeout time.Duration) *NATSDocumentConnector {
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return &NATSDocumentConnector{
		conn:    conn,
		timeout: timeout,
	}
}

// ListDocuments retrieves a list of documents based on query parameters
func (c *NATSDocumentConnector) ListDocuments(ctx context.Context, query domain.ListDocumentsQuery) (*domain.ListDocumentsResponse, error) {
	reqData, err := query.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectDocumentsList, reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send list request: %w", err)
	}

	var response domain.ListDocumentsResponse
	if err := response.UnmarshalJSON(msg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list response: %w", err)
	}

	return &response, nil
}

// GetDocument retrieves a single document by ID
func (c *NATSDocumentConnector) GetDocument(ctx context.Context, id uuid.UUID) (*domain.DocumentResponse, error) {
	req := domain.IDRequest{ID: id.String()}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectDocumentsGet, reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send get request: %w", err)
	}

	var response domain.DocumentResponse
	if err := response.UnmarshalJSON(msg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get response: %w", err)
	}

	return &response, nil
}

// CreateDocument creates a new document
func (c *NATSDocumentConnector) CreateDocument(ctx context.Context, req domain.CreateDocumentRequest) (*domain.DocumentResponse, error) {
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectDocumentsCreate, reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send create request: %w", err)
	}

	var response domain.DocumentResponse
	if err := response.UnmarshalJSON(msg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create response: %w", err)
	}

	return &response, nil
}

// UpdateDocument updates an existing document
func (c *NATSDocumentConnector) UpdateDocument(ctx context.Context, id uuid.UUID, req domain.UpdateDocumentRequest) (*domain.DocumentResponse, error) {
	reqPayload := domain.UpdateDocumentPayload{
		ID:      id.String(),
		Updates: req,
	}

	reqData, err := reqPayload.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectDocumentsUpdate, reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to send update request: %w", err)
	}

	var response domain.DocumentResponse
	if err := response.UnmarshalJSON(msg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update response: %w", err)
	}

	return &response, nil
}

// DeleteDocument deletes a document by ID
func (c *NATSDocumentConnector) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	req := domain.IDRequest{ID: id.String()}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	msg, err := c.conn.RequestWithContext(ctx, SubjectDocumentsDelete, reqData)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}

	// Check for error response
	var errorResp domain.ErrorResponse
	if err := errorResp.UnmarshalJSON(msg.Data); err == nil && errorResp.Error != "" {
		return fmt.Errorf("delete failed: %s - %s", errorResp.Error, errorResp.Message)
	}

	return nil
}
