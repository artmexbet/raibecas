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
	pipeline Pipeline
}

func NewConsumer(cfg *config.NATS, conn *nats.Conn, pipeline Pipeline) (*Consumer, error) {
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}

	return &Consumer{
		cfg:      cfg,
		conn:     conn,
		js:       js,
		pipeline: pipeline,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	if c.cfg.Subject == "" {
		return fmt.Errorf("subject is empty")
	}

	_, err := c.js.AddConsumer(c.cfg.Stream, &nats.ConsumerConfig{
		Durable:       c.cfg.Durable,
		FilterSubject: c.cfg.Subject,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.cfg.AckWait,
		MaxAckPending: c.cfg.MaxInFly,
	})
	if err != nil && !errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
		slog.Warn("add consumer", "err", err)
	}

	_, err = c.js.QueueSubscribe(c.cfg.Subject, c.cfg.Queue, func(msg *nats.Msg) {
		defer func() {
			_ = msg.Ack()
		}()
		if err := c.handleMessage(ctx, msg.Data); err != nil {
			slog.Error("handle message", "err", err)
			_ = msg.Nak()
		}
	}, nats.ManualAck(), nats.AckWait(c.cfg.AckWait), nats.MaxAckPending(c.cfg.MaxInFly))
	if err != nil {
		return fmt.Errorf("queue subscribe: %w", err)
	}

	return nil
}

func (c *Consumer) handleMessage(ctx context.Context, data []byte) error {
	var event domain.DocumentIndexEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Валидация
	if event.DocumentID == "" {
		return fmt.Errorf("document_id is required")
	}
	if event.FilePath == "" {
		return fmt.Errorf("file_path is required")
	}

	// Создаем документ из события
	doc := domain.Document{
		ID:        event.DocumentID,
		Title:     event.Title,
		FilePath:  event.FilePath,
		SourceURI: event.SourceURI,
		Metadata:  event.Metadata,
	}

	// Индексируем документ
	return c.pipeline.Index(ctx, doc)
}

func (c *Consumer) Stop() error {
	c.conn.Close()
	return nil
}
