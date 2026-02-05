package handler

import (
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/libs/dto"
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
	var req domain.CreateDocumentRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid create document request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized create document attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	doc, err := h.service.CreateDocument(msg.Ctx, req)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to create document", "error", err)
		if errors.Is(err, service.ErrInvalidInput) {
			return h.respondError(msg, dto.ErrCodeInvalidRequest)
		}
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	return h.respond(msg, doc)
}

// HandleGetDocument handles document retrieval requests
func (h *DocumentHandler) HandleGetDocument(msg *natsw.Message) error {
	var req struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
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

	return msg.RespondEasyJSON(doc)
}

// HandleGetDocumentContent handles document content retrieval requests
func (h *DocumentHandler) HandleGetDocumentContent(msg *natsw.Message) error {
	var req struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
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

	response := struct {
		Content string `json:"content"`
	}{
		Content: string(content),
	}

	return h.respond(msg, response)
}

// HandleListDocuments handles document listing requests
func (h *DocumentHandler) HandleListDocuments(msg *natsw.Message) error {
	var req struct {
		Limit      int        `json:"limit"`
		Offset     int        `json:"offset"`
		AuthorID   *uuid.UUID `json:"author_id,omitempty"`
		CategoryID *int       `json:"category_id,omitempty"`
		Search     string     `json:"search,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid list documents request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Set default limit
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	var categoryID *int32
	if req.CategoryID != nil {
		cid := int32(*req.CategoryID)
		categoryID = &cid
	}

	docs, total, err := h.service.ListDocuments(msg.Ctx, domain.ListDocumentsParams{
		Limit:      req.Limit,
		Offset:     req.Offset,
		AuthorID:   req.AuthorID,
		CategoryID: categoryID,
		Search:     req.Search,
	})
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list documents", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	response := struct {
		Documents []domain.Document `json:"documents"`
		Total     int               `json:"total"`
	}{
		Documents: docs,
		Total:     total,
	}

	return h.respond(msg, response)
}

// HandleUpdateDocument handles document update requests
func (h *DocumentHandler) HandleUpdateDocument(msg *natsw.Message) error {
	var req struct {
		ID      uuid.UUID                    `json:"id"`
		Updates domain.UpdateDocumentRequest `json:"updates"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid update document request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized update document attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	doc, err := h.service.UpdateDocument(msg.Ctx, req.ID, req.Updates)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return h.respondError(msg, dto.ErrCodeNotFound)
		}
		h.logger.ErrorContext(msg.Ctx, "failed to update document", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	return msg.RespondEasyJSON(doc)
}

// HandleDeleteDocument handles document deletion requests
func (h *DocumentHandler) HandleDeleteDocument(msg *natsw.Message) error {
	var req struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
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

	response := struct {
		Success bool `json:"success"`
	}{
		Success: true,
	}

	data, _ := json.Marshal(response)
	return msg.Respond(data)
}

// HandleListDocumentVersions handles document versions listing requests
func (h *DocumentHandler) HandleListDocumentVersions(msg *natsw.Message) error {
	var req struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid list versions request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	versions, err := h.service.ListDocumentVersions(msg.Ctx, req.ID)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list document versions", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	data, _ := json.Marshal(versions)
	return msg.Respond(data)
}

// HandleDocumentIndexed handles document indexed events from index-python
func (h *DocumentHandler) HandleDocumentIndexed(msg *natsw.Message) error {
	var event domain.DocumentIndexedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
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

func (h *DocumentHandler) respond(msg *natsw.Message, data any) error {
	resp := &dto.StandardResponse{
		Success: true,
		Data:    data,
	}
	return msg.RespondEasyJSON(resp)
}
