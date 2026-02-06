package connector

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/dto/documents"
	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// NATS subjects for document service communication
const (
	SubjectDocumentsList       = "documents.list"
	SubjectDocumentsGet        = "documents.get"
	SubjectDocumentsGetContent = "documents.get_content"
	SubjectDocumentsCreate     = "documents.create"
	SubjectDocumentsUpdate     = "documents.update"
	SubjectDocumentsDelete     = "documents.delete"

	// Default timeout for NATS requests
	defaultTimeout = 5 * time.Second
)

// NATSDocumentConnector implements server.DocumentServiceConnector using NATS for communication
type NATSDocumentConnector struct {
	client  *natsw.Client
	timeout time.Duration
}

// NewNATSDocumentConnector creates a new NATS-based document service connector
func NewNATSDocumentConnector(client *natsw.Client, timeout time.Duration) *NATSDocumentConnector {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &NATSDocumentConnector{
		client:  client,
		timeout: timeout,
	}
}

// ListDocuments retrieves a list of documents based on query parameters
func (c *NATSDocumentConnector) ListDocuments(ctx context.Context, query domain.ListDocumentsQuery) (*domain.ListDocumentsResponse, error) {
	// Convert to dto type
	dtoQuery := documents.ListDocumentsQuery{
		Page:       query.Page,
		Limit:      query.Limit,
		AuthorID:   query.AuthorID,
		CategoryID: query.CategoryID,
		TagID:      query.TagID,
		Search:     query.Search,
	}

	reqData, err := dtoQuery.MarshalJSON()
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

	var dtoResponse documents.ListDocumentsResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list response: %w", err)
	}

	// Convert back to domain type
	response := &domain.ListDocumentsResponse{
		Documents:  convertDocuments(dtoResponse.Documents),
		Total:      dtoResponse.Total,
		Page:       dtoResponse.Page,
		Limit:      dtoResponse.Limit,
		TotalPages: dtoResponse.TotalPages,
	}

	return response, nil
}

// GetDocument retrieves a single document by ID
func (c *NATSDocumentConnector) GetDocument(ctx context.Context, id uuid.UUID) (*domain.GetDocumentResponse, error) {
	req := documents.GetDocumentRequest{ID: id}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsGet)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send get request: %w", err)
	}

	var dtoResponse documents.GetDocumentResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get response: %w", err)
	}

	// Получаем контент документа отдельным запросом (временно)
	// todo: не возвращать контент целиком
	contentReq := documents.GetDocumentContentRequest{ID: id}
	contentReqData, err := contentReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content request: %w", err)
	}

	contentMsg := nats.NewMsg(SubjectDocumentsGetContent)
	contentMsg.Data = contentReqData

	contentRespMsg, err := c.client.RequestMsg(ctx, contentMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to send content request: %w", err)
	}

	var contentResponse documents.GetDocumentContentResponse
	if err := contentResponse.UnmarshalJSON(contentRespMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content response: %w", err)
	}

	// Конвертируем и добавляем контент
	doc := convertDocument(dtoResponse.Document)
	doc.Content = &contentResponse.Content

	response := &domain.GetDocumentResponse{
		Document: doc,
	}

	return response, nil
}

// CreateDocument creates a new document
func (c *NATSDocumentConnector) CreateDocument(ctx context.Context, req domain.CreateDocumentRequest) (*domain.CreateDocumentResponse, error) {
	// Convert domain request to dto
	dtoReq := documents.CreateDocumentRequest{
		Title:           req.Title,
		Description:     req.Description,
		AuthorID:        req.AuthorID,
		CategoryID:      req.CategoryID,
		PublicationDate: req.PublicationDate,
		TagIDs:          req.TagIDs,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsCreate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create request: %w", err)
	}

	var dtoResponse documents.CreateDocumentResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create response: %w", err)
	}

	// Convert dto response back to domain
	response := &domain.CreateDocumentResponse{
		Document: convertDocument(dtoResponse.Document),
	}

	return response, nil
}

// UpdateDocument updates an existing document
func (c *NATSDocumentConnector) UpdateDocument(ctx context.Context, id uuid.UUID, req domain.UpdateDocumentRequest) (*domain.UpdateDocumentResponse, error) {
	// Convert domain request to dto
	dtoReq := documents.UpdateDocumentRequest{
		ID:              id,
		Title:           req.Title,
		Description:     req.Description,
		AuthorID:        req.AuthorID,
		CategoryID:      req.CategoryID,
		PublicationDate: req.PublicationDate,
		TagIDs:          req.TagIDs,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsUpdate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send update request: %w", err)
	}

	var dtoResponse documents.UpdateDocumentResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update response: %w", err)
	}

	// Convert dto response back to domain
	response := &domain.UpdateDocumentResponse{
		Document: convertDocument(dtoResponse.Document),
	}

	return response, nil
}

// DeleteDocument deletes a document by ID
func (c *NATSDocumentConnector) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	req := documents.DeleteDocumentRequest{ID: id}
	reqData, err := req.MarshalJSON()
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

// Helper functions for converting between dto and domain types

func convertDocument(dto documents.Document) domain.Document {
	doc := domain.Document{
		ID:              dto.ID,
		Title:           dto.Title,
		Description:     dto.Description,
		PublicationDate: dto.PublicationDate,
		Additional: domain.Additional{
			CreatedAt: dto.CreatedAt,
			UpdatedAt: dto.UpdatedAt,
		},
	}

	// Convert Author - dto has more fields than domain
	if dto.Author != nil {
		doc.Author = domain.Author{
			ID:   dto.Author.ID,
			Name: dto.Author.Name,
		}
	}

	// Convert Category - dto has more fields than domain
	if dto.Category != nil {
		doc.Category = domain.Category{
			ID:    dto.Category.ID,
			Title: dto.Category.Title,
		}
	}

	// Convert Tags - dto has more fields than domain
	if len(dto.Tags) > 0 {
		doc.Tags = make([]domain.Tag, len(dto.Tags))
		for i, tag := range dto.Tags {
			doc.Tags[i] = domain.Tag{
				ID:    tag.ID,
				Title: tag.Title,
			}
		}
	}

	return doc
}

func convertDocuments(dtoDocuments []documents.Document) []domain.Document {
	result := make([]domain.Document, len(dtoDocuments))
	for i, dto := range dtoDocuments {
		result[i] = convertDocument(dto)
	}
	return result
}
