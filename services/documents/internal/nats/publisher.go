package nats

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/documents/internal/domain"
)

const (
	topicDocumentCreated = "corpus.document.created"
	topicDocumentUpdated = "corpus.document.updated"
	topicDocumentDeleted = "corpus.document.deleted"
)

// Publisher handles publishing events to NATS
type Publisher struct {
	client *natsw.Client
	logger *slog.Logger
}

// NewPublisher creates a new event publisher
func NewPublisher(client *natsw.Client, logger *slog.Logger) *Publisher {
	return &Publisher{
		client: client,
		logger: logger,
	}
}

// PublishDocumentCreated publishes a document created event
func (p *Publisher) PublishDocumentCreated(ctx context.Context, event domain.DocumentCreatedEvent) error {
	data, err := event.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, topicDocumentCreated, data); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	p.logger.InfoContext(ctx, "published document.created event",
		"document_id", event.DocumentID,
		"title", event.Title,
	)

	return nil
}

// PublishDocumentUpdated publishes a document updated event
func (p *Publisher) PublishDocumentUpdated(ctx context.Context, event domain.DocumentUpdatedEvent) error {
	data, err := event.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, topicDocumentUpdated, data); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	p.logger.InfoContext(ctx, "published document.updated event",
		"document_id", event.DocumentID,
		"old_version", event.OldVersion,
		"new_version", event.NewVersion,
	)

	return nil
}

// PublishDocumentDeleted publishes a document deleted event
func (p *Publisher) PublishDocumentDeleted(ctx context.Context, event domain.DocumentDeletedEvent) error {
	data, err := event.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, topicDocumentDeleted, data); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	p.logger.InfoContext(ctx, "published document.deleted event",
		"document_id", event.DocumentID,
	)

	return nil
}
