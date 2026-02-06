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
)

// Server represents the NATS server with subscriptions
type Server struct {
	client  *natsw.Client
	handler *DocumentHandler
}

// New creates a new server instance
func New(client *natsw.Client, handler *DocumentHandler) *Server {
	return &Server{
		client:  client,
		handler: handler,
	}
}

// Start registers all NATS subscriptions
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

	return nil
}
