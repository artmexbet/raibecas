package connector

import (
	"context"
	"encoding/json"
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
	SubjectNotesList            = "documents.notes.list"
	SubjectNotesGet             = "documents.notes.get"
	SubjectNotesCreate          = "documents.notes.create"
	SubjectNotesUpdate          = "documents.notes.update"
	SubjectNotesDelete          = "documents.notes.delete"
	SubjectDocumentsGet         = "documents.get"
	SubjectDocumentsGetContent  = "documents.get.content"
	SubjectDocumentsCreate      = "documents.create"
	SubjectDocumentsUpdate      = "documents.update"
	SubjectDocumentsDelete      = "documents.delete"
	SubjectDocumentsCoverUpload = "documents.cover.upload"
	SubjectDocumentsReindex     = "documents.reindex"
	SubjectCorpusSearch         = "corpus.search"

	// Metadata subjects
	SubjectAuthorsList         = "documents.authors.list"
	SubjectAuthorsCreate       = "documents.authors.create"
	SubjectCategoriesList      = "documents.categories.list"
	SubjectCategoriesCreate    = "documents.categories.create"
	SubjectDocumentTypesList   = "documents.types.list"
	SubjectAuthorshipTypesList = "documents.authorship-types.list"
	SubjectTagsList            = "documents.tags.list"
	SubjectTagsCreate          = "documents.tags.create"

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
func (c *NATSDocumentConnector) ListDocuments(ctx context.Context, query domain.ListDocumentsQuery, userRole string) (*domain.ListDocumentsResponse, error) {
	// Convert to dto type
	dtoQuery := documents.ListDocumentsQuery{
		Page:           query.Page,
		Limit:          query.Limit,
		AuthorID:       query.AuthorID,
		CategoryID:     query.CategoryID,
		DocumentTypeID: query.DocumentTypeID,
		TagID:          query.TagID,
		Search:         query.Search,
	}

	reqData, err := dtoQuery.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsList)
	msg.Data = reqData
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

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
func (c *NATSDocumentConnector) GetDocument(ctx context.Context, id uuid.UUID, userRole string) (*domain.GetDocumentResponse, error) {
	req := documents.GetDocumentRequest{ID: id}
	reqData, err := req.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsGet)
	msg.Data = reqData
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send get request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
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
		CategoryID:      intValue(req.CategoryID),
		DocumentTypeID:  req.DocumentTypeID,
		Participants:    toDTOParticipantRefs(req.Participants),
		PublicationDate: req.PublicationDate,
		TagIDs:          req.TagIDs,
		Content:         req.Content,
		IsPublic:        req.IsPublic,
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
		CategoryID:      intValue(req.CategoryID),
		DocumentTypeID:  req.DocumentTypeID,
		Participants:    toDTOParticipantRefs(req.Participants),
		PublicationDate: req.PublicationDate,
		TagIDs:          req.TagIDs,
		IsPublic:        req.IsPublic,
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
		DocumentType:    nil,
		Participants:    nil,
		PublicationDate: dto.PublicationDate,
		CoverURL:        dto.CoverURL,
		Indexed:         dto.Indexed,
		IsPublic:        dto.IsPublic,
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

	if dto.DocumentType != nil {
		doc.DocumentType = &domain.DocumentType{
			ID:   dto.DocumentType.ID,
			Name: dto.DocumentType.Name,
		}
	}

	if len(dto.Participants) > 0 {
		doc.Participants = make([]domain.DocumentParticipant, len(dto.Participants))
		for i, participant := range dto.Participants {
			doc.Participants[i] = domain.DocumentParticipant{
				Author: domain.Author{
					ID:   participant.Author.ID,
					Name: participant.Author.Name,
				},
				AuthorshipType: domain.AuthorshipType{
					ID:    participant.AuthorshipType.ID,
					Title: participant.AuthorshipType.Title,
				},
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

// ReindexDocument triggers reindexing of a document
func (c *NATSDocumentConnector) ReindexDocument(ctx context.Context, id uuid.UUID, userRole string) error {
	req := documents.ReindexDocumentRequest{ID: id}
	reqData, err := easyjson.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal reindex request: %w", err)
	}

	msg := nats.NewMsg(SubjectDocumentsReindex)
	msg.Data = reqData
	if userRole != "" {
		msg.Header.Set("X-User-Role", userRole)
	}

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send reindex request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return errResp
	}

	return nil
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

// ListDocumentTypes retrieves all document types.
func (c *NATSDocumentConnector) ListDocumentTypes(ctx context.Context) (*domain.ListDocumentTypesResponse, error) {
	msg := nats.NewMsg(SubjectDocumentTypesList)
	msg.Data = []byte("{}")

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list document types request: %w", err)
	}

	var dtoResponse documents.ListDocumentTypesResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list document types response: %w", err)
	}

	documentTypes := make([]domain.DocumentType, len(dtoResponse.DocumentTypes))
	for i, dtoType := range dtoResponse.DocumentTypes {
		documentTypes[i] = domain.DocumentType{ID: dtoType.ID, Name: dtoType.Name}
	}

	return &domain.ListDocumentTypesResponse{DocumentTypes: documentTypes}, nil
}

// ListAuthorshipTypes retrieves all authorship types.
func (c *NATSDocumentConnector) ListAuthorshipTypes(ctx context.Context) (*domain.ListAuthorshipTypesResponse, error) {
	msg := nats.NewMsg(SubjectAuthorshipTypesList)
	msg.Data = []byte("{}")

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send list authorship types request: %w", err)
	}

	var dtoResponse documents.ListAuthorshipTypesResponse
	if err := dtoResponse.UnmarshalJSON(respMsg.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list authorship types response: %w", err)
	}

	authorshipTypes := make([]domain.AuthorshipType, len(dtoResponse.AuthorshipTypes))
	for i, dtoType := range dtoResponse.AuthorshipTypes {
		authorshipTypes[i] = domain.AuthorshipType{ID: dtoType.ID, Title: dtoType.Title}
	}

	return &domain.ListAuthorshipTypesResponse{AuthorshipTypes: authorshipTypes}, nil
}

func toDTOParticipantRefs(participants []domain.DocumentParticipantRef) []documents.DocumentParticipantRef {
	if len(participants) == 0 {
		return nil
	}
	result := make([]documents.DocumentParticipantRef, 0, len(participants))
	for _, participant := range participants {
		authorID, err := uuid.Parse(participant.AuthorID)
		if err != nil {
			continue
		}
		result = append(result, documents.DocumentParticipantRef{AuthorID: authorID, TypeID: participant.TypeID})
	}
	return result
}

func intValue[T ~int | *int](value T) int {
	switch v := any(value).(type) {
	case int:
		return v
	case *int:
		if v == nil {
			return 0
		}
		return *v
	default:
		return 0
	}
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

// --- Notes ---

// ListNotes retrieves a list of notes based on query parameters.
func (c *NATSDocumentConnector) ListNotes(ctx context.Context, query domain.ListNotesQuery) (*domain.ListNotesResponse, error) {
	dtoQuery := documents.ListNotesQuery{
		Page:       query.Page,
		Limit:      query.Limit,
		Search:     query.Search,
		DocumentID: query.DocumentID,
		BookmarkID: query.BookmarkID,
		UserID:     query.UserID,
	}

	reqData, err := dtoQuery.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notes list request: %w", err)
	}

	msg := nats.NewMsg(SubjectNotesList)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send notes list request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.ListNotesResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notes list response: %w", err)
	}

	return &domain.ListNotesResponse{
		Items:      convertNotes(dtoResponse.Items),
		Total:      dtoResponse.Total,
		Page:       dtoResponse.Page,
		Limit:      dtoResponse.Limit,
		TotalPages: dtoResponse.TotalPages,
	}, nil
}

// GetNote retrieves a single note by ID.
func (c *NATSDocumentConnector) GetNote(ctx context.Context, userID, noteID uuid.UUID) (*domain.GetNoteResponse, error) {
	dtoReq := documents.GetNoteRequest{
		ID:     noteID,
		UserID: userID,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get note request: %w", err)
	}

	msg := nats.NewMsg(SubjectNotesGet)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send get note request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.GetNoteResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get note response: %w", err)
	}

	return &domain.GetNoteResponse{Item: convertNote(dtoResponse.Item)}, nil
}

// CreateNote creates a new note for the authenticated user.
func (c *NATSDocumentConnector) CreateNote(ctx context.Context, req domain.CreateNoteRequest) (*domain.CreateNoteResponse, error) {
	dtoReq := documents.CreateNoteRequest{
		UserID:             req.UserID,
		Title:              req.Title,
		Content:            req.Content,
		DocumentID:         req.DocumentID,
		BookmarkID:         req.BookmarkID,
		PositionInDocument: req.PositionInDocument,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create note request: %w", err)
	}

	msg := nats.NewMsg(SubjectNotesCreate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send create note request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.CreateNoteResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create note response: %w", err)
	}

	return &domain.CreateNoteResponse{Item: convertNote(dtoResponse.Item)}, nil
}

// UpdateNote updates an existing note for the authenticated user.
func (c *NATSDocumentConnector) UpdateNote(ctx context.Context, req domain.UpdateNoteRequest) (*domain.UpdateNoteResponse, error) {
	dtoReq := documents.UpdateNoteRequest{
		ID:                 req.ID,
		UserID:             req.UserID,
		Title:              req.Title,
		Content:            req.Content,
		DocumentID:         req.DocumentID,
		BookmarkID:         req.BookmarkID,
		PositionInDocument: req.PositionInDocument,
		ClearDocumentID:    req.ClearDocumentID,
		ClearBookmarkID:    req.ClearBookmarkID,
		ClearPosition:      req.ClearPosition,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update note request: %w", err)
	}

	msg := nats.NewMsg(SubjectNotesUpdate)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send update note request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return nil, errResp
	}

	var dtoResponse documents.UpdateNoteResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update note response: %w", err)
	}

	return &domain.UpdateNoteResponse{Item: convertNote(dtoResponse.Item)}, nil
}

// DeleteNote removes a note for the authenticated user.
func (c *NATSDocumentConnector) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	dtoReq := documents.DeleteNoteRequest{
		ID:     noteID,
		UserID: userID,
	}

	reqData, err := dtoReq.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal delete note request: %w", err)
	}

	msg := nats.NewMsg(SubjectNotesDelete)
	msg.Data = reqData

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send delete note request: %w", err)
	}

	if errResp := checkErrorResponse(respMsg.Data); errResp != nil {
		return errResp
	}

	var dtoResponse documents.DeleteNoteResponse
	if err := easyjson.Unmarshal(respMsg.Data, &dtoResponse); err != nil {
		return fmt.Errorf("failed to unmarshal delete note response: %w", err)
	}
	if !dtoResponse.Success {
		return fmt.Errorf("delete note failed")
	}

	return nil
}

func convertNote(dtoNote documents.NoteItem) domain.NoteItem {
	return domain.NoteItem{
		ID:                 dtoNote.ID,
		Title:              dtoNote.Title,
		Content:            dtoNote.Content,
		DocumentID:         dtoNote.DocumentID,
		BookmarkID:         dtoNote.BookmarkID,
		PositionInDocument: dtoNote.PositionInDocument,
		CreatedAt:          dtoNote.CreatedAt,
		UpdatedAt:          dtoNote.UpdatedAt,
	}
}

func convertNotes(dtoNotes []documents.NoteItem) []domain.NoteItem {
	result := make([]domain.NoteItem, len(dtoNotes))
	for i, dtoNote := range dtoNotes {
		result[i] = convertNote(dtoNote)
	}
	return result
}

// SemanticSearch sends a NATS request to corpus.search (index-python) and returns search results.
func (c *NATSDocumentConnector) SemanticSearch(ctx context.Context, query domain.SearchQuery) (*domain.SearchResponse, error) {
	reqPayload := map[string]any{
		"query": query.Q,
		"limit": query.Limit,
	}

	reqData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	msg := nats.NewMsg(SubjectCorpusSearch)
	msg.Data = reqData

	// Use a longer timeout for search (embedding generation + vector search)
	searchTimeout := c.timeout * 3 //nolint:mnd // search needs more time than typical NATS requests
	ctx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()

	respMsg, err := c.client.RequestMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send search request: %w", err)
	}

	// Parse the response — index-python returns {"success": bool, "data": {...}, "error": "..."}
	var rawResp struct {
		Success bool                  `json:"success"`
		Data    domain.SearchResponse `json:"data"`
		Error   string                `json:"error"`
	}
	if err := json.Unmarshal(respMsg.Data, &rawResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search response: %w", err)
	}

	if !rawResp.Success {
		return nil, fmt.Errorf("search failed: %s", rawResp.Error)
	}

	return &rawResp.Data, nil
}
