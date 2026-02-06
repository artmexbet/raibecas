package server

import (
	"log/slog"

	"github.com/artmexbet/raibecas/libs/dto"
	"github.com/artmexbet/raibecas/libs/dto/documents"
	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/documents/internal/service"
)

// MetadataHandler handles NATS requests for metadata (authors, categories, tags)
type MetadataHandler struct {
	service *service.DocumentService
	logger  *slog.Logger
}

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler(service *service.DocumentService, logger *slog.Logger) *MetadataHandler {
	return &MetadataHandler{
		service: service,
		logger:  logger,
	}
}

// HandleListAuthors handles list authors requests
func (h *MetadataHandler) HandleListAuthors(msg *natsw.Message) error {
	authors, err := h.service.ListAuthors(msg.Ctx)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list authors", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert to dto
	dtoAuthors := make([]documents.Author, len(authors))
	for i, author := range authors {
		dtoAuthors[i] = documents.Author{
			ID:        author.ID,
			Name:      author.Name,
			Bio:       author.Bio,
			CreatedAt: author.CreatedAt,
			UpdatedAt: author.UpdatedAt,
		}
	}

	response := documents.ListAuthorsResponse{
		Authors: dtoAuthors,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleCreateAuthor handles create author requests
func (h *MetadataHandler) HandleCreateAuthor(msg *natsw.Message) error {
	var req documents.CreateAuthorRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid create author request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized create author attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	author, err := h.service.CreateAuthor(msg.Ctx, req.Name)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to create author", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	response := documents.CreateAuthorResponse{
		Author: documents.Author{
			ID:        author.ID,
			Name:      author.Name,
			Bio:       author.Bio,
			CreatedAt: author.CreatedAt,
			UpdatedAt: author.UpdatedAt,
		},
	}

	return msg.RespondEasyJSON(&response)
}

// HandleListCategories handles list categories requests
func (h *MetadataHandler) HandleListCategories(msg *natsw.Message) error {
	categories, err := h.service.ListCategories(msg.Ctx)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list categories", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert to dto
	dtoCategories := make([]documents.Category, len(categories))
	for i, category := range categories {
		dtoCategories[i] = documents.Category{
			ID:          category.ID,
			Title:       category.Title,
			Description: category.Description,
			CreatedAt:   category.CreatedAt,
		}
	}

	response := documents.ListCategoriesResponse{
		Categories: dtoCategories,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleCreateCategory handles create category requests
func (h *MetadataHandler) HandleCreateCategory(msg *natsw.Message) error {
	var req documents.CreateCategoryRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid create category request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized create category attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	category, err := h.service.CreateCategory(msg.Ctx, req.Title)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to create category", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	response := documents.CreateCategoryResponse{
		Category: documents.Category{
			ID:          category.ID,
			Title:       category.Title,
			Description: category.Description,
			CreatedAt:   category.CreatedAt,
		},
	}

	return msg.RespondEasyJSON(&response)
}

// HandleListTags handles list tags requests
func (h *MetadataHandler) HandleListTags(msg *natsw.Message) error {
	tags, err := h.service.ListTags(msg.Ctx)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to list tags", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	// Convert to dto
	dtoTags := make([]documents.Tag, len(tags))
	for i, tag := range tags {
		dtoTags[i] = documents.Tag{
			ID:        tag.ID,
			Title:     tag.Title,
			CreatedAt: tag.CreatedAt,
		}
	}

	response := documents.ListTagsResponse{
		Tags: dtoTags,
	}

	return msg.RespondEasyJSON(&response)
}

// HandleCreateTag handles create tag requests
func (h *MetadataHandler) HandleCreateTag(msg *natsw.Message) error {
	var req documents.CreateTagRequest
	if err := req.UnmarshalJSON(msg.Data); err != nil {
		h.logger.ErrorContext(msg.Ctx, "invalid create tag request", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Check authorization (admin only)
	if !h.isAdmin(msg) {
		h.logger.WarnContext(msg.Ctx, "unauthorized create tag attempt")
		return h.respondError(msg, dto.ErrCodeUnauthorized)
	}

	tag, err := h.service.CreateTag(msg.Ctx, req.Title)
	if err != nil {
		h.logger.ErrorContext(msg.Ctx, "failed to create tag", "error", err)
		return h.respondError(msg, dto.ErrCodeInternal)
	}

	response := documents.CreateTagResponse{
		Tag: documents.Tag{
			ID:        tag.ID,
			Title:     tag.Title,
			CreatedAt: tag.CreatedAt,
		},
	}

	return msg.RespondEasyJSON(&response)
}

// Helper methods

func (h *MetadataHandler) respondError(msg *natsw.Message, code dto.ErrorCode) error {
	response := dto.ErrorResponse{
		Error: string(code),
	}
	return msg.RespondEasyJSON(&response)
}

func (h *MetadataHandler) isAdmin(msg *natsw.Message) bool {
	// Extract role from message headers (set by gateway)
	role := msg.Header.Get("X-User-Role")
	return role == "Admin" || role == "SuperAdmin"
}
