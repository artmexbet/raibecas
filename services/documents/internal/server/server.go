package server

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/artmexbet/raibecas/libs/natsw"
)

const (
	subjectDocumentsCreate      = "documents.create"
	subjectBookmarksList        = "documents.bookmarks.list"
	subjectBookmarksCreate      = "documents.bookmarks.create"
	subjectBookmarksDelete      = "documents.bookmarks.delete"
	subjectDocumentsGet         = "documents.get"
	subjectDocumentsGetContent  = "documents.get.content"
	subjectDocumentsList        = "documents.list"
	subjectDocumentsUpdate      = "documents.update"
	subjectDocumentsDelete      = "documents.delete"
	subjectDocumentsVersions    = "documents.versions"
	subjectDocumentsReindex     = "documents.reindex"
	subjectDocumentIndexed      = "indexing.document.indexed"
	subjectDocumentsCoverUpload = "documents.cover.upload"

	// Metadata subjects
	subjectAuthorsList         = "documents.authors.list"
	subjectAuthorsCreate       = "documents.authors.create"
	subjectCategoriesList      = "documents.categories.list"
	subjectCategoriesCreate    = "documents.categories.create"
	subjectDocumentTypesList   = "documents.types.list"
	subjectAuthorshipTypesList = "documents.authorship-types.list"
	subjectTagsList            = "documents.tags.list"
	subjectTagsCreate          = "documents.tags.create"

	// JetStream consumer for indexing events
	indexedConsumerDurable = "documents-indexed-consumer"
	indexedConsumerStream  = "INDEXING"
)

// Server represents the NATS server with subscriptions
type Server struct {
	client          *natsw.Client
	jsCtx           *natsw.JetStreamContext
	handler         *DocumentHandler
	metadataHandler *MetadataHandler
	consumeCtx      jetstream.ConsumeContext // JetStream consumer context for graceful stop
}

// New creates a new server instance
func New(client *natsw.Client, jsCtx *natsw.JetStreamContext, handler *DocumentHandler, metadataHandler *MetadataHandler) *Server {
	return &Server{
		client:          client,
		jsCtx:           jsCtx,
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
	s.client.Subscribe(subjectBookmarksList, s.handler.HandleListBookmarks)
	s.client.Subscribe(subjectBookmarksCreate, s.handler.HandleCreateBookmark)
	s.client.Subscribe(subjectBookmarksDelete, s.handler.HandleDeleteBookmark)
	s.client.Subscribe(subjectDocumentsGet, s.handler.HandleGetDocument)
	s.client.Subscribe(subjectDocumentsGetContent, s.handler.HandleGetDocumentContent)
	s.client.Subscribe(subjectDocumentsList, s.handler.HandleListDocuments)
	s.client.Subscribe(subjectDocumentsUpdate, s.handler.HandleUpdateDocument)
	s.client.Subscribe(subjectDocumentsDelete, s.handler.HandleDeleteDocument)
	s.client.Subscribe(subjectDocumentsVersions, s.handler.HandleListDocumentVersions)
	s.client.Subscribe(subjectDocumentsCoverUpload, s.handler.HandleUploadCover)
	s.client.Subscribe(subjectDocumentsReindex, s.handler.HandleReindexDocument)

	// Event subscriptions via JetStream (guaranteed delivery with ACK/NAK)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	consumeCtx, err := s.jsCtx.ConsumeStream(ctx, natsw.ConsumerConfig{
		Stream:        indexedConsumerStream,
		Durable:       indexedConsumerDurable,
		FilterSubject: subjectDocumentIndexed,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
	}, s.handler.HandleDocumentIndexed)
	if err != nil {
		return fmt.Errorf("failed to start JetStream consumer for %s: %w", subjectDocumentIndexed, err)
	}
	s.consumeCtx = consumeCtx

	// Metadata operations (request-reply)
	s.client.Subscribe(subjectAuthorsList, s.metadataHandler.HandleListAuthors)
	s.client.Subscribe(subjectAuthorsCreate, s.metadataHandler.HandleCreateAuthor)
	s.client.Subscribe(subjectCategoriesList, s.metadataHandler.HandleListCategories)
	s.client.Subscribe(subjectCategoriesCreate, s.metadataHandler.HandleCreateCategory)
	s.client.Subscribe(subjectDocumentTypesList, s.metadataHandler.HandleListDocumentTypes)
	s.client.Subscribe(subjectAuthorshipTypesList, s.metadataHandler.HandleListAuthorshipTypes)
	s.client.Subscribe(subjectTagsList, s.metadataHandler.HandleListTags)
	s.client.Subscribe(subjectTagsCreate, s.metadataHandler.HandleCreateTag)

	return nil
}

// Stop gracefully stops the JetStream consumer.
func (s *Server) Stop() {
	if s.consumeCtx != nil {
		s.consumeCtx.Stop()
	}
}
