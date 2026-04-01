package connector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mailru/easyjson"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/dto"
	"github.com/artmexbet/raibecas/libs/dto/documents"
	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

var (
	ErrInvalidRequest = errors.New("invalid_request")
	ErrNotFound       = errors.New("not_found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrInternal       = errors.New("internal_error")
)

// NATS subjects for document service communication
const (
	SubjectDocumentsList        = "documents.list"
	SubjectBookmarksList        = "documents.bookmarks.list"
	SubjectBookmarksCreate      = "documents.bookmarks.create"
	SubjectBookmarksDelete      = "documents.bookmarks.delete"
	SubjectDocumentsGet         = "documents.get"
	SubjectDocumentsGetContent  = "documents.get.content"
	SubjectDocumentsCreate      = "documents.create"
	SubjectDocumentsUpdate      = "documents.update"
	SubjectDocumentsDelete      = "documents.delete"
	SubjectDocumentsCoverUpload = "documents.cover.upload"

	// Metadata subjects
	SubjectAuthorsList      = "documents.authors.list"
	SubjectAuthorsCreate    = "documents.authors.create"
	SubjectCategoriesList   = "documents.categories.list"
	SubjectCategoriesCreate = "documents.categories.create"
	SubjectTagsList         = "documents.tags.list"
	SubjectTagsCreate       = "documents.tags.create"

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

// ListBookmarks retrieves a list of bookmarks based on query parameters.
func (c *NATSDocumentConnector) ListBookmarks(ctx context.Context, query domain.ListBookmarksQuery) (*domain.ListBookmarksResponse, error) {
	dtoQuery := documents.ListBookmarksQuery{
		Page:   query.Page,
		Limit:  query.Limit,
		Search: query.Search,
		Kind:   documents.BookmarkKind(query.Kind),
		UserID: query.UserID,
	}

	reqData, err := easyjson.Marshal(dtoQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bookmarks list request: %w", err)
	}

	msg := nats.NewMsg(SubjectBookmarksList)
	msg.Data = reqData
	if query.UserID != uuid.Nil {
		msg.Header.Set("X-User-ID", query.UserID.String())
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send bookmarks list request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.ListBookmarksResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bookmarks list response: %w", err)
	}

	return &domain.ListBookmarksResponse{
		Items:      convertBookmarks(dtoResponse.Items),
		Total:      dtoResponse.Total,
		Page:       dtoResponse.Page,
		Limit:      dtoResponse.Limit,
		TotalPages: dtoResponse.TotalPages,
	}, nil
}

// CreateBookmark saves a bookmark for the authenticated user.
func (c *NATSDocumentConnector) CreateBookmark(ctx context.Context, req domain.CreateBookmarkRequest) (*domain.CreateBookmarkResponse, error) {
	dtoReq := documents.CreateBookmarkRequest{
		UserID:     req.UserID,
		DocumentID: req.DocumentID,
		Kind:       documents.BookmarkKind(req.Kind),
		QuoteText:  req.QuoteText,
		Context:    req.Context,
		PageLabel:  req.PageLabel,
	}

	reqData, err := easyjson.Marshal(dtoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create bookmark request: %w", err)
	}

	msg := nats.NewMsg(SubjectBookmarksCreate)
	msg.Data = reqData
	if req.UserID != uuid.Nil {
		msg.Header.Set("X-User-ID", req.UserID.String())
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create bookmark request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.CreateBookmarkResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create bookmark response: %w", err)
	}

	return &domain.CreateBookmarkResponse{Item: convertBookmark(dtoResponse.Item)}, nil
}

// DeleteBookmark removes a bookmark for the authenticated user.
func (c *NATSDocumentConnector) DeleteBookmark(ctx context.Context, userID, bookmarkID uuid.UUID) error {
	dtoReq := documents.DeleteBookmarkRequest{
		ID:     bookmarkID,
		UserID: userID,
	}

	reqData, err := easyjson.Marshal(dtoReq)
	if err != nil {
		return fmt.Errorf("failed to marshal delete bookmark request: %w", err)
	}

	msg := nats.NewMsg(SubjectBookmarksDelete)
	msg.Data = reqData
	if userID != uuid.Nil {
		msg.Header.Set("X-User-ID", userID.String())
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send delete bookmark request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return errResp
	}

	var dtoResponse documents.DeleteBookmarkResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return fmt.Errorf("failed to unmarshal delete bookmark response: %w", err)
	}
	if !dtoResponse.Success {
		return fmt.Errorf("delete bookmark failed")
	}

	return nil
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
func (c *NATSDocumentConnector) CreateDocument(ctx context.Context, req domain.CreateDocumentRequest, userRole string) (*domain.CreateDocumentResponse, error) {
	// Convert domain request to dto
	dtoReq := documents.CreateDocumentRequest{
		Title:           req.Title,
		Description:     req.Description,
		AuthorID:        req.AuthorID,
		CategoryID:      req.CategoryID,
		PublicationDate: req.PublicationDate,
		TagIDs:          req.TagIDs,
		Content:         req.Content, // Передаем контент вместе с остальными данными
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsCreate)
	msg.Data = reqData

	// Set user role in NATS header for authorization
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

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
func (c *NATSDocumentConnector) UpdateDocument(ctx context.Context, req domain.UpdateDocumentRequest, userRole string) (*domain.UpdateDocumentResponse, error) {
	// Convert domain request to dto
	dtoReq := documents.UpdateDocumentRequest{
		ID:              req.ID,
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

	// Set user role in NATS header
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

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
func (c *NATSDocumentConnector) DeleteDocument(ctx context.Context, id uuid.UUID, userRole string) error {
	req := documents.DeleteDocumentRequest{ID: id}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsDelete)
	msg.Data = reqData

	// Set user role in NATS header
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

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
		CoverURL:        dto.CoverURL,
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
	for i, dtoDocument := range dtoDocuments {
		result[i] = convertDocument(dtoDocument)
	}
	return result
}

func convertBookmark(dtoBookmark documents.BookmarkItem) domain.BookmarkItem {
	return domain.BookmarkItem{
		ID:        dtoBookmark.ID,
		Kind:      domain.BookmarkKind(dtoBookmark.Kind),
		SavedAt:   dtoBookmark.SavedAt,
		Document:  convertDocument(dtoBookmark.Document),
		QuoteText: dtoBookmark.QuoteText,
		Context:   dtoBookmark.Context,
		PageLabel: dtoBookmark.PageLabel,
	}
}

func convertBookmarks(dtoBookmarks []documents.BookmarkItem) []domain.BookmarkItem {
	result := make([]domain.BookmarkItem, len(dtoBookmarks))
	for i, dtoBookmark := range dtoBookmarks {
		result[i] = convertBookmark(dtoBookmark)
	}
	return result
}

// UploadCover uploads a cover image for a document
func (c *NATSDocumentConnector) UploadCover(ctx context.Context, id uuid.UUID, data []byte, contentType string, userRole string) (string, error) {
	req := documents.UploadCoverRequest{
		ID:          id,
		Data:        data,
		ContentType: contentType,
	}

	reqData, err := req.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal upload cover request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsCoverUpload)
	msg.Data = reqData
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send upload cover request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return "", errResp
	}

	var dtoResponse documents.UploadCoverResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return "", fmt.Errorf("failed to unmarshal upload cover response: %w", err)
	}

	return dtoResponse.CoverURL, nil
}

// checkErrorResponse checks if NATS response contains an error and returns it
func checkErrorResponse(data []byte) error {
	var errorResp dto.ErrorResponse
	if err := errorResp.UnmarshalJSON(data); err == nil && errorResp.Error != "" {
		switch errorResp.Error {
		case string(dto.ErrCodeInvalidRequest):
			return ErrInvalidRequest
		case string(dto.ErrCodeNotFound):
			return ErrNotFound
		case string(dto.ErrCodeUnauthorized):
			return ErrUnauthorized
		case string(dto.ErrCodeForbidden):
			return ErrForbidden
		case string(dto.ErrCodeInternal):
			return ErrInternal
		default:
			return fmt.Errorf("nats error response: %s", errorResp.Error)
		}
	}
	return nil
}

// Metadata methods

// ListAuthors retrieves all authors
func (c *NATSDocumentConnector) ListAuthors(ctx context.Context) (*domain.ListAuthorsResponse, error) {
	msg := nats.NewMsg(SubjectAuthorsList)
	msg.Data = []byte("{}")

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list authors request: %w", err)
	}

	var dtoResponse documents.ListAuthorsResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list authors response: %w", err)
	}

	// Convert to domain type
	authors := make([]domain.Author, len(dtoResponse.Authors))
	for i, dtoAuthor := range dtoResponse.Authors {
		authors[i] = domain.Author{
			ID:   dtoAuthor.ID,
			Name: dtoAuthor.Name,
		}
	}

	return &domain.ListAuthorsResponse{
		Authors: authors,
	}, nil
}

// CreateAuthor creates a new author
func (c *NATSDocumentConnector) CreateAuthor(ctx context.Context, req domain.CreateAuthorRequest, userRole string) (*domain.CreateAuthorResponse, error) {
	dtoReq := documents.CreateAuthorRequest{
		Name: req.Name,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create author request: %w", err)
	}

	msg := nats.NewMsg(SubjectAuthorsCreate)
	msg.Data = reqData

	// Set user role in NATS header
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create author request: %w", err)
	}

	// Check for error response
	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.CreateAuthorResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create author response: %w", err)
	}

	return &domain.CreateAuthorResponse{
		Author: domain.Author{
			ID:   dtoResponse.Author.ID,
			Name: dtoResponse.Author.Name,
		},
	}, nil
}

// ListCategories retrieves all categories
func (c *NATSDocumentConnector) ListCategories(ctx context.Context) (*domain.ListCategoriesResponse, error) {
	msg := nats.NewMsg(SubjectCategoriesList)
	msg.Data = []byte("{}")

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list categories request: %w", err)
	}

	var dtoResponse documents.ListCategoriesResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list categories response: %w", err)
	}

	// Convert to domain type
	categories := make([]domain.Category, len(dtoResponse.Categories))
	for i, dtoCategory := range dtoResponse.Categories {
		categories[i] = domain.Category{
			ID:    dtoCategory.ID,
			Title: dtoCategory.Title,
		}
	}

	return &domain.ListCategoriesResponse{
		Categories: categories,
	}, nil
}

// CreateCategory creates a new category
func (c *NATSDocumentConnector) CreateCategory(ctx context.Context, req domain.CreateCategoryRequest, userRole string) (*domain.CreateCategoryResponse, error) {
	dtoReq := documents.CreateCategoryRequest{
		Title: req.Title,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create category request: %w", err)
	}

	msg := nats.NewMsg(SubjectCategoriesCreate)
	msg.Data = reqData

	// Set user role in NATS header
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create category request: %w", err)
	}

	// Check for error response
	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.CreateCategoryResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create category response: %w", err)
	}

	return &domain.CreateCategoryResponse{
		Category: domain.Category{
			ID:    dtoResponse.Category.ID,
			Title: dtoResponse.Category.Title,
		},
	}, nil
}

// ListTags retrieves all tags
func (c *NATSDocumentConnector) ListTags(ctx context.Context) (*domain.ListTagsResponse, error) {
	msg := nats.NewMsg(SubjectTagsList)
	msg.Data = []byte("{}")

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list tags request: %w", err)
	}

	var dtoResponse documents.ListTagsResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list tags response: %w", err)
	}

	// Convert to domain type
	tags := make([]domain.Tag, len(dtoResponse.Tags))
	for i, dtoTag := range dtoResponse.Tags {
		tags[i] = domain.Tag{
			ID:    dtoTag.ID,
			Title: dtoTag.Title,
		}
	}

	return &domain.ListTagsResponse{
		Tags: tags,
	}, nil
}

// CreateTag creates a new tag
func (c *NATSDocumentConnector) CreateTag(ctx context.Context, req domain.CreateTagRequest, userRole string) (*domain.CreateTagResponse, error) {
	dtoReq := documents.CreateTagRequest{
		Title: req.Title,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create tag request: %w", err)
	}

	msg := nats.NewMsg(SubjectTagsCreate)
	msg.Data = reqData

	// Set user role in NATS header
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create tag request: %w", err)
	}

	// Check for error response
	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.CreateTagResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create tag response: %w", err)
	}

	return &domain.CreateTagResponse{
		Tag: domain.Tag{
			ID:    dtoResponse.Tag.ID,
			Title: dtoResponse.Tag.Title,
		},
	}, nil
}
