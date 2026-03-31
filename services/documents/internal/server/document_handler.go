package server

import (
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/libs/dto"
	"github.com/artmexbet/raibecas/libs/dto/documents"
	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
	"github.com/artmexbet/raibecas/services/documents/internal/service"
)

// DocumentHandler handles NATS requests for documents
type DocumentHandler struct {
	service *service.DocumentService
	logger  *slog.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(service *service.DocumentService, logger *slog.Logger) *DocumentHandler {
	return &DocumentHandler{
		service: service,
		logger:  logger,
	}
}

// HandleCreateDocument handles document creation requests
func (h *DocumentHandler) HandleCreateDocument(msg *natsw.Message) error {
	var req documents.CreateDocumentRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid create document request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized create document attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	// Convert to domain type
	domainReq := domain.CreateDocumentRequest{
		Title:           req.Title,
		Description:     req.Description,
		AuthorID:        req.AuthorID,
		CategoryID:      req.CategoryID,
		PublicationDate: req.PublicationDate,
		Content:         req.Content,
		TagIDs:          req.TagIDs,
		CreatedBy:       req.CreatedBy,
	}

	doc, err := h.service.CreateDocument(msg.Ctx, domainReq)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to create document", "error", err)
		if errors.Is(err, service.ErrInvalidInput) {
			return h.respondError(msg, dto.ErrCodeInvalidRequest)
		}
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert domain document to dto
	dtoDoc := convertDomainToDTO(*doc)
	response := documents.CreateDocumentResponse{
		Document: dtoDoc,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleGetDocument handles document retrieval requests
func (h *DocumentHandler) HandleGetDocument(msg *natsw.Message) error {
	var req documents.GetDocumentRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid get document request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	doc, err := h.service.GetDocument(msg.Ctx, req.ID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		h.logger.ErrorContext(msg.Ctx, "failed to get document", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert domain document to dto
	dtoDoc := convertDomainToDTO(*doc)
	response := documents.GetDocumentResponse{
		Document: dtoDoc,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleGetDocumentContent handles document content retrieval requests
func (h *DocumentHandler) HandleGetDocumentContent(msg *natsw.Message) error {
	var req documents.GetDocumentContentRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid get document content request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	content, err := h.service.GetDocumentContent(msg.Ctx, req.ID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		h.logger.ErrorContext(msg.Ctx, "failed to get document content", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	response := documents.GetDocumentContentResponse{
		Content: string(content),
	}

	return msg.RespondEasyJSON(&response)
}

// HandleListDocuments handles document listing requests
func (h *DocumentHandler) HandleListDocuments(msg *natsw.Message) error {
	var req documents.ListDocumentsQuery
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid list documents request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Set default limit
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var categoryID *int32
	if req.CategoryID != 0 {
		cid := int32(req.CategoryID)
		categoryID = &cid
	}

	var authorID *uuid.UUID
	if req.AuthorID != uuid.Nil {
		authorID = &req.AuthorID
	}

	docs, total, err := h.service.ListDocuments(msg.Ctx, domain.ListDocumentsParams{
		Limit:      limit,
		Offset:     req.Offset,
		AuthorID:   authorID,
		CategoryID: categoryID,
		Search:     req.Search,
	})
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list documents", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert domain documents to dto
	dtoDocs := make([]documents.Document, len(docs))
	for i, doc := range docs {
		dtoDocs[i] = convertDomainToDTO(doc)
	}

	response := documents.ListDocumentsResponse{
		Documents: dtoDocs,
		Total:     total,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleListBookmarks handles bookmark listing requests.
func (h *DocumentHandler) HandleListBookmarks(msg *natsw.Message) error {
	var req documents.ListBookmarksQuery
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid list bookmarks request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	page := max(req.Page, 1)
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 16
	}

	if req.UserID == uuid.Nil {
		if userIDValue := msg.Header.Get("X-User-ID"); userIDValue != "" {
			userID, err := uuid.Parse(userIDValue)
			if err != nil {
				h.logger.ErrorContext(msg.Ctx, "invalid X-User-ID header", "error", err)
				return h.respondError(msg, dto.ErrCodeInvalidRequest)
			}
			req.UserID = userID
		}
	}

	items, total, err := h.service.ListBookmarks(msg.Ctx, domain.ListBookmarksParams{
		Page:   page,
		Limit:  limit,
		Search: req.Search,
		Kind:   domain.BookmarkKind(req.Kind),
		UserID: req.UserID,
	})
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list bookmarks", "error", err)
		if errors.Is(err, service.ErrInvalidInput) {
			return h.respondError(msg, dto.ErrCodeInvalidRequest)
		}
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	dtoItems := make([]documents.BookmarkItem, len(items))
	for i, item := range items {
		dtoItems[i] = convertBookmarkDomainToDTOItem(item)
	}

	response := documents.ListBookmarksResponse{
		Items:      dtoItems,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	return msg.RespondEasyJSON(response)
}

// HandleCreateBookmark handles bookmark creation requests.
func (h *DocumentHandler) HandleCreateBookmark(msg *natsw.Message) error {
	var req documents.CreateBookmarkRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid create bookmark request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	if req.UserID == uuid.Nil {
		if userIDValue := msg.Header.Get("X-User-ID"); userIDValue != "" {
			userID, err := uuid.Parse(userIDValue)
			if err != nil {
				h.logger.ErrorContext(msg.Ctx, "invalid X-User-ID header", "error", err)
				return h.respondError(msg, dto.ErrCodeInvalidRequest)
			}
			req.UserID = userID
		}
	}

	item, err := h.service.CreateBookmark(msg.Ctx, domain.CreateBookmarkRequest{
		UserID:     req.UserID,
		DocumentID: req.DocumentID,
		Kind:       domain.BookmarkKind(req.Kind),
		QuoteText:  req.QuoteText,
		Context:    req.Context,
		PageLabel:  req.PageLabel,
	})
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to create bookmark", "error", err)
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			return h.respondError(msg, dto.ErrCodeInvalidRequest)
		case errors.Is(err, service.ErrNotFound):
			return h.respondError(msg, dto.ErrCodeNotFound)
		default:
			return h.respondError(msg, dto.ErrCodeInternal)
		}
	}

	response := documents.CreateBookmarkResponse{Item: convertBookmarkDomainToDTOItem(*item)}
	return msg.RespondEasyJSON(response)
}

// HandleDeleteBookmark handles bookmark deletion requests.
func (h *DocumentHandler) HandleDeleteBookmark(msg *natsw.Message) error {
	var req documents.DeleteBookmarkRequest
	if err := msg.UnmarshalEasyJSON(&req); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid delete bookmark request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	if req.UserID == uuid.Nil {
		if userIDValue := msg.Header.Get("X-User-ID"); userIDValue != "" {
			userID, err := uuid.Parse(userIDValue)
			if err != nil {
				h.logger.ErrorContext(msg.Ctx, "invalid X-User-ID header", "error", err)
				return h.respondError(msg, dto.ErrCodeInvalidRequest)
			}
			req.UserID = userID
		}
	}

	if err := h.service.DeleteBookmark(msg.Ctx, req.UserID, req.ID); err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to delete bookmark", "error", err)
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			return h.respondError(msg, dto.ErrCodeInvalidRequest)
		case errors.Is(err, service.ErrNotFound):
			return h.respondError(msg, dto.ErrCodeNotFound)
		default:
			return h.respondError(msg, dto.ErrCodeInternal)
		}
	}

	response := documents.DeleteBookmarkResponse{Success: true}
	return msg.RespondEasyJSON(response)
}

// HandleUpdateDocument handles document update requests
func (h *DocumentHandler) HandleUpdateDocument(msg *natsw.Message) error {
	var req documents.UpdateDocumentRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid update document request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized update document attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	// Convert to domain type
	domainReq := domain.UpdateDocumentRequest{
		Title:           req.Title,
		Description:     req.Description,
		AuthorID:        req.AuthorID,
		CategoryID:      req.CategoryID,
		PublicationDate: req.PublicationDate,
		Content:         req.Content,
		TagIDs:          req.TagIDs,
		Changes:         req.Changes,
		UpdatedBy:       req.UpdatedBy,
	}

	doc, err := h.service.UpdateDocument(msg.Ctx, req.ID, domainReq)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		h.logger.ErrorContext(msg.Ctx, "failed to update document", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert domain document to dto
	dtoDoc := convertDomainToDTO(*doc)
	response := documents.UpdateDocumentResponse{
		Document: dtoDoc,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleDeleteDocument handles document deletion requests
func (h *DocumentHandler) HandleDeleteDocument(msg *natsw.Message) error {
	var req documents.DeleteDocumentRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid delete document request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized delete document attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	if err := h.service.DeleteDocument(msg.Ctx, req.ID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		h.logger.ErrorContext(msg.Ctx, "failed to delete document", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	response := documents.DeleteDocumentResponse{
		Success: true,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleListDocumentVersions handles document versions listing requests
func (h *DocumentHandler) HandleListDocumentVersions(msg *natsw.Message) error {
	var req documents.ListDocumentVersionsRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid list versions request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	versions, err := h.service.ListDocumentVersions(msg.Ctx, req.ID)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list document versions", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert domain versions to dto
	dtoVersions := make([]documents.DocumentVersion, len(versions))
	for i, v := range versions {
		dtoVersions[i] = documents.DocumentVersion{
			ID:          v.ID,
			DocumentID:  v.DocumentID,
			Version:     v.Version,
			ContentPath: v.ContentPath,
			Changes:     v.Changes,
			CreatedBy:   v.CreatedBy,
			CreatedAt:   v.CreatedAt,
		}
	}

	response := documents.ListDocumentVersionsResponse{
		Versions: dtoVersions,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleDocumentIndexed handles document indexed events from index-python
func (h *DocumentHandler) HandleDocumentIndexed(msg *natsw.Message) error {
	var event domain.DocumentIndexedEvent
	if err := event.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid document indexed event", "error", err)
		return nil // Don't fail on event processing
	}

	indexed := event.Status == "success"
	if err := h.service.MarkDocumentIndexed(msg.Ctx, event.DocumentID, indexed); err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to mark document as indexed",
			"document_id", event.DocumentID,
			"error", err,
		)
	} else {
		h.logger.InfoContext(msg.Ctx, "marked document as indexed",
			"document_id", event.DocumentID,
			"chunks_count", event.ChunksCount,
		)
	}

	return nil
}

// HandleUploadCover handles cover image upload requests
func (h *DocumentHandler) HandleUploadCover(msg *natsw.Message) error {
	var req documents.UploadCoverRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid upload cover request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized upload cover attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	coverURL, err := h.service.UploadCover(msg.Ctx, req.ID, req.Data, req.ContentType)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to upload cover", "error", err)
		if errors.Is(err, service.ErrNotFound) {
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	response := documents.UploadCoverResponse{
		CoverURL: coverURL,
	}

	return msg.RespondEasyJSON(&response)
}

// isAdmin checks if the user has admin role from message context
func (h *DocumentHandler) isAdmin(msg *natsw.Message) bool {
	// Extract role from message headers (set by gateway)
	role := msg.Header.Get("X-User-Role")
	return role == "Admin" || role == "SuperAdmin"
}

// respondError sends an error response using easyjson
func (h *DocumentHandler) respondError(msg *natsw.Message, errCode dto.ErrorCode) error {
	resp := &dto.ErrorResponse{
		Success: false,
		Error:   string(errCode),
	}
	return msg.RespondEasyJSON(resp)
}

// convertDomainToDTO converts domain.Document to documents.Document DTO
func convertDomainToDTO(doc domain.Document) documents.Document {
	dtoDoc := documents.Document{
		ID:              doc.ID,
		Title:           doc.Title,
		Description:     doc.Description,
		AuthorID:        doc.AuthorID,
		CategoryID:      doc.CategoryID,
		PublicationDate: doc.PublicationDate,
		ContentPath:     doc.ContentPath,
		CurrentVersion:  doc.CurrentVersion,
		Indexed:         doc.Indexed,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
		CoverURL:        doc.CoverURL,
	}

	if doc.Author != nil {
		dtoDoc.Author = &documents.Author{
			ID:        doc.Author.ID,
			Name:      doc.Author.Name,
			Bio:       doc.Author.Bio,
			CreatedAt: doc.Author.CreatedAt,
			UpdatedAt: doc.Author.UpdatedAt,
		}
	}

	if doc.Category != nil {
		dtoDoc.Category = &documents.Category{
			ID:          doc.Category.ID,
			Title:       doc.Category.Title,
			Description: doc.Category.Description,
			CreatedAt:   doc.Category.CreatedAt,
		}
	}

	if len(doc.Tags) > 0 {
		dtoDoc.Tags = make([]documents.Tag, len(doc.Tags))
		for i, tag := range doc.Tags {
			dtoDoc.Tags[i] = documents.Tag{
				ID:        tag.ID,
				Title:     tag.Title,
				CreatedAt: tag.CreatedAt,
			}
		}
	}

	return dtoDoc
}

func convertBookmarkDomainToDTOItem(item domain.BookmarkItem) documents.BookmarkItem {
	return documents.BookmarkItem{
		ID:        item.ID,
		Kind:      documents.BookmarkKind(item.Kind),
		SavedAt:   item.SavedAt,
		Document:  convertDomainToDTO(item.Document),
		QuoteText: item.QuoteText,
		Context:   item.Context,
		PageLabel: item.PageLabel,
	}
}
