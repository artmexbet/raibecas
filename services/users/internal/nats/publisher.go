package nats

import (
	"context"

	"github.com/artmexbet/raibecas/libs/natsw"
)

// Publisher wraps NATS client for publishing events
type Publisher struct {
	client *natsw.Client
}

// NewPublisher creates a new NATS publisher
func NewPublisher(client *natsw.Client) *Publisher {
	return &Publisher{
		client: client,
	}
}

// Publish publishes a message to the specified subject
func (p *Publisher) Publish(ctx context.Context, subject string, data []byte) error {
	return p.client.Publish(ctx, subject, data)
}
