package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/services/index/internal/config"
	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

type Pipeline interface {
	Index(ctx context.Context, doc domain.Document) error
}

type Consumer struct {
	cfg      *config.NATS
	conn     *nats.Conn
	js       nats.JetStreamContext
	fetcher  Fetcher
	pipeline Pipeline
}

type indexMessage struct {
	DocumentID string            `json:"document_id"`
	Title      string            `json:"title"`
	SourceURI  string            `json:"source_uri"`
	Metadata   map[string]string `json:"metadata"`
	Content    string            `json:"content"`
}

func NewConsumer(cfg *config.NATS, conn *nats.Conn, fetcher Fetcher, pipeline Pipeline) (*Consumer, error) {
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	return &Consumer{cfg: cfg, conn: conn, js: js, fetcher: fetcher, pipeline: pipeline}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	if c.cfg.Subject == "" {
		return fmt.Errorf("subject is empty")
	}

	_, err := c.js.AddConsumer(c.cfg.Subject, &nats.ConsumerConfig{
		Durable:       c.cfg.Durable,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.cfg.AckWait,
		MaxAckPending: c.cfg.MaxInFly,
	})
	if err != nil && !errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
		slog.Warn("add consumer", "err", err)
	}

	_, err = c.js.QueueSubscribe(c.cfg.Subject, c.cfg.Queue, func(msg *nats.Msg) {
		defer msg.Ack() //nolint:errcheck // acknowledge by default
		if err := c.handleMessage(ctx, msg.Data); err != nil {
			slog.Error("handle message", "err", err)
			msg.Nak() //nolint:errcheck // negative acknowledge on error
		}
	}, nats.ManualAck(), nats.AckWait(c.cfg.AckWait), nats.MaxAckPending(c.cfg.MaxInFly))
	if err != nil {
		return fmt.Errorf("queue subscribe: %w", err)
	}

	<-ctx.Done()
	return nil
}

func (c *Consumer) handleMessage(ctx context.Context, data []byte) error {
	var msg indexMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}

	doc := domain.Document{
		ID:        msg.DocumentID,
		Title:     msg.Title,
		Content:   msg.Content,
		SourceURI: msg.SourceURI,
		Metadata:  msg.Metadata,
	}

	if doc.Content == "" && c.fetcher != nil {
		fetched, err := c.fetcher.Fetch(doc.ID)
		if err != nil {
			return fmt.Errorf("fetch document %s: %w", doc.ID, err)
		}
		doc.Content = fetched.Content
		if doc.Metadata == nil {
			doc.Metadata = fetched.Metadata
		}
	}

	return c.pipeline.Index(ctx, doc)
}
