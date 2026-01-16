package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/natsw"
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
	client  *natsw.Client
	timeout time.Duration
}

// NewNATSDocumentConnector creates a new NATS-based document service connector
func NewNATSDocumentConnector(conn *nats.Conn, timeout time.Duration) *NATSDocumentConnector {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	// Создаём клиент с автоматической пропагацией trace context
	client := natsw.NewClient(conn)

	return &NATSDocumentConnector{
		client:  client,
		timeout: timeout,
	}
}

// ListDocuments retrieves a list of documents based on query parameters
func (c *NATSDocumentConnector) ListDocuments(ctx context.Context, query domain.ListDocumentsQuery) (*domain.ListDocumentsResponse, error) {
	reqData, err := query.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsList)
	msg.Data = reqData

	// RequestMsg автоматически пропагирует trace context
	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list request: %w", err)
	}

	var response domain.ListDocumentsResponse
	if err := response.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list response: %w", err)
	}

	return &response, nil
}

// GetDocument retrieves a single document by ID
func (c *NATSDocumentConnector) GetDocument(ctx context.Context, id uuid.UUID) (*domain.GetDocumentResponse, error) {
	// Create request payload
	type getDocRequest struct {
		ID string `json:"id"`
	}
	req := getDocRequest{ID: id.String()}
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsGet)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send get request: %w", err)
	}

	var response domain.GetDocumentResponse
	if err := response.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get response: %w", err)
	}

	return &response, nil
}

// CreateDocument creates a new document
func (c *NATSDocumentConnector) CreateDocument(ctx context.Context, req domain.CreateDocumentRequest) (*domain.CreateDocumentResponse, error) {
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsCreate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create request: %w", err)
	}

	var response domain.CreateDocumentResponse
	if err := response.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create response: %w", err)
	}

	return &response, nil
}

// UpdateDocument updates an existing document
func (c *NATSDocumentConnector) UpdateDocument(ctx context.Context, id uuid.UUID, req domain.UpdateDocumentRequest) (*domain.UpdateDocumentResponse, error) {
	// Create request payload with ID and updates
	type updateDocRequest struct {
		ID      string                       `json:"id"`
		Updates domain.UpdateDocumentRequest `json:"updates"`
	}
	reqPayload := updateDocRequest{
		ID:      id.String(),
		Updates: req,
	}

	reqData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsUpdate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send update request: %w", err)
	}

	var response domain.UpdateDocumentResponse
	if err := response.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update response: %w", err)
	}

	return &response, nil
}

// DeleteDocument deletes a document by ID
func (c *NATSDocumentConnector) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	// Create request payload
	type deleteDocRequest struct {
		ID string `json:"id"`
	}
	req := deleteDocRequest{ID: id.String()}
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsDelete)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}

	// Check for error response
	var errorResp domain.ErrorResponse
	if err := errorResp.UnmarshalJSON(respMsg.Data); err == nil && errorResp.Error != "" {
		return fmt.Errorf("delete failed: %s - %s", errorResp.Error, errorResp.Message)
	}

	return nil
}
