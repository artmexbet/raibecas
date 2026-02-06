package server

import (
	"github.com/artmexbet/raibecas/libs/natsw"
)

const (
	subjectDocumentsCreate     = "documents.create"
	subjectDocumentsGet        = "documents.get"
	subjectDocumentsGetContent = "documents.get.content"
	subjectDocumentsList       = "documents.list"
	subjectDocumentsUpdate     = "documents.update"
	subjectDocumentsDelete     = "documents.delete"
	subjectDocumentsVersions   = "documents.versions"
	subjectDocumentIndexed     = "indexing.document.indexed"

	// Metadata subjects
	subjectAuthorsList      = "documents.authors.list"
	subjectAuthorsCreate    = "documents.authors.create"
	subjectCategoriesList   = "documents.categories.list"
	subjectCategoriesCreate = "documents.categories.create"
	subjectTagsList         = "documents.tags.list"
	subjectTagsCreate       = "documents.tags.create"
)

// Server represents the NATS server with subscriptions
type Server struct {
	client          *natsw.Client
	handler         *DocumentHandler
	metadataHandler *MetadataHandler
}

// New creates a new server instance
func New(client *natsw.Client, handler *DocumentHandler, metadataHandler *MetadataHandler) *Server {
	return &Server{
		client:          client,
		handler:         handler,
		metadataHandler: metadataHandler,
	}
}

// Start registers all NATS subscriptions
//
//nolint:errcheck // Subscriptions are fire-and-forget, errors are logged in the client
func (s *Server) Start() error {
	// Document operations (request-reply)
	s.client.Subscribe(subjectDocumentsCreate, s.handler.HandleCreateDocument)
	s.client.Subscribe(subjectDocumentsGet, s.handler.HandleGetDocument)
	s.client.Subscribe(subjectDocumentsGetContent, s.handler.HandleGetDocumentContent)
	s.client.Subscribe(subjectDocumentsList, s.handler.HandleListDocuments)
	s.client.Subscribe(subjectDocumentsUpdate, s.handler.HandleUpdateDocument)
	s.client.Subscribe(subjectDocumentsDelete, s.handler.HandleDeleteDocument)
	s.client.Subscribe(subjectDocumentsVersions, s.handler.HandleListDocumentVersions)

	// Event subscriptions (pub-sub)
	s.client.Subscribe(subjectDocumentIndexed, s.handler.HandleDocumentIndexed)

	// Metadata operations (request-reply)
	s.client.Subscribe(subjectAuthorsList, s.metadataHandler.HandleListAuthors)
	s.client.Subscribe(subjectAuthorsCreate, s.metadataHandler.HandleCreateAuthor)
	s.client.Subscribe(subjectCategoriesList, s.metadataHandler.HandleListCategories)
	s.client.Subscribe(subjectCategoriesCreate, s.metadataHandler.HandleCreateCategory)
	s.client.Subscribe(subjectTagsList, s.metadataHandler.HandleListTags)
	s.client.Subscribe(subjectTagsCreate, s.metadataHandler.HandleCreateTag)

	return nil
}
