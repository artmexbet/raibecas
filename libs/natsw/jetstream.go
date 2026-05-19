package natsw

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamConfig describes a JetStream stream to be created or updated.
type StreamConfig struct {
	Name       string
	Subjects   []string
	MaxAge     time.Duration
	Storage    jetstream.StorageType
	Retention  jetstream.RetentionPolicy
	MaxDeliver int
}

// ConsumerConfig describes a durable JetStream consumer.
type ConsumerConfig struct {
	Stream        string
	Durable       string
	FilterSubject string
	AckWait       time.Duration
	MaxDeliver    int
}

// JetStreamContext wraps the new JetStream API for stream/consumer management.
type JetStreamContext struct {
	js     jetstream.JetStream
	client *Client
	logger *slog.Logger
}

// JetStream initialises a JetStreamContext from the underlying NATS connection.
func (c *Client) JetStream() (*JetStreamContext, error) {
	js, err := jetstream.New(c.conn)
	if err != nil {
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}
	return &JetStreamContext{
		js:     js,
		client: c,
		logger: c.logger,
	}, nil
}

// EnsureStream creates or updates a JetStream stream.
func (jsc *JetStreamContext) EnsureStream(ctx context.Context, cfg StreamConfig) (jetstream.Stream, error) {
	jsCfg := jetstream.StreamConfig{
		Name:      cfg.Name,
		Subjects:  cfg.Subjects,
		Storage:   cfg.Storage,
		Retention: cfg.Retention,
		MaxAge:    cfg.MaxAge,
	}

	stream, err := jsc.js.CreateOrUpdateStream(ctx, jsCfg)
	if err != nil {
		return nil, fmt.Errorf("create/update stream %q: %w", cfg.Name, err)
	}

	jsc.logger.Info("jetstream stream ensured",
		"stream", cfg.Name,
		"subjects", cfg.Subjects,
		"retention", cfg.Retention.String(),
	)

	return stream, nil
}

// ConsumeStream creates a durable consumer and starts consuming messages.
// The handler receives messages wrapped in natsw.Message with trace context.
// On handler success the message is acked; on error it is nacked for redelivery.
func (jsc *JetStreamContext) ConsumeStream(
	ctx context.Context,
	cfg ConsumerConfig,
	handler HandlerFunc,
) (jetstream.ConsumeContext, error) {
	consumer, err := jsc.js.CreateOrUpdateConsumer(ctx, cfg.Stream, jetstream.ConsumerConfig{
		Durable:       cfg.Durable,
		FilterSubject: cfg.FilterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.AckWait,
		MaxDeliver:    cfg.MaxDeliver,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("create/update consumer %q on stream %q: %w", cfg.Durable, cfg.Stream, err)
	}

	jsc.logger.Info("jetstream consumer created",
		"consumer", cfg.Durable,
		"stream", cfg.Stream,
		"filter", cfg.FilterSubject,
	)

	consumeCtx, err := consumer.Consume(func(jsMsg jetstream.Msg) {
		// Extract headers for trace propagation
		headers := jsMsg.Headers()
		rawMsg := &nats.Msg{
			Subject: jsMsg.Subject(),
			Data:    jsMsg.Data(),
			Header:  headers,
		}

		// Extract trace context
		msgCtx := jsc.client.extractContext(rawMsg)
		_, span := jsc.client.tracer.Start(msgCtx, fmt.Sprintf("nats.jetstream.handle %s", jsMsg.Subject()))
		defer span.End()

		message := &Message{
			Msg: rawMsg,
			Ctx: msgCtx,
		}

		// Apply middleware chain
		finalHandler := jsc.client.applyMiddlewares(handler)

		if err := finalHandler(message); err != nil {
			jsc.logger.Error("jetstream handler error, nacking message",
				"subject", jsMsg.Subject(),
				"consumer", cfg.Durable,
				"error", err,
			)
			if nakErr := jsMsg.Nak(); nakErr != nil {
				jsc.logger.Error("failed to nak message", "error", nakErr)
			}
			return
		}

		if ackErr := jsMsg.Ack(); ackErr != nil {
			jsc.logger.Error("failed to ack message", "error", ackErr)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("start consuming %q: %w", cfg.Durable, err)
	}

	jsc.logger.Info("jetstream consumer started",
		"consumer", cfg.Durable,
		"stream", cfg.Stream,
	)

	return consumeCtx, nil
}

// Publish publishes a message to a JetStream subject with acknowledgement.
// Returns the publish ack or an error.
func (jsc *JetStreamContext) Publish(ctx context.Context, subject string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}

	// Inject trace context into headers
	jsc.client.propagator.Inject(ctx, &headerCarrier{header: msg.Header})

	ack, err := jsc.js.PublishMsg(ctx, msg, opts...)
	if err != nil {
		return nil, fmt.Errorf("jetstream publish to %q: %w", subject, err)
	}

	return ack, nil
}

// Raw returns the underlying jetstream.JetStream for advanced usage.
func (jsc *JetStreamContext) Raw() jetstream.JetStream {
	return jsc.js
}
