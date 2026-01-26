package natsw

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Message - обёртка над nats.Msg с контекстом
type Message struct {
	*nats.Msg
	Ctx context.Context
}

// UnmarshalData десериализует данные сообщения в структуру
func (m *Message) UnmarshalData(v interface{}) error {
	return json.Unmarshal(m.Data, v)
}

// RespondJSON отправляет ответ на запрос с JSON-сериализацией
func (m *Message) RespondJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	return m.Respond(data)
}

// HandlerFunc - обработчик сообщений с контекстом
type HandlerFunc func(*Message) error

// Middleware - middleware функция
type Middleware func(next HandlerFunc) HandlerFunc

// Client - обёртка над NATS connection с поддержкой middleware и context
type Client struct {
	conn        *nats.Conn
	middlewares []Middleware
	logger      *slog.Logger
	propagator  propagation.TextMapPropagator
	tracer      trace.Tracer
}

// ClientOption - опция для конфигурации клиента
type ClientOption func(*Client)

// NewClient создаёт новый NATS клиент с middleware
func NewClient(conn *nats.Conn, opts ...ClientOption) *Client {
	c := &Client{
		conn:       conn,
		logger:     slog.Default(),
		propagator: otel.GetTextMapPropagator(),
		tracer:     otel.Tracer("natsw"),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithMiddleware добавляет middleware в цепочку
func WithMiddleware(mw Middleware) ClientOption {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, mw)
	}
}

// WithLogger устанавливает кастомный logger
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
		// Автоматически добавляем logging middleware
		c.middlewares = append(c.middlewares, LoggingMiddleware(logger))
	}
}

// WithRecover добавляет recover middleware для защиты от паник
func WithRecover() ClientOption {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, RecoverMiddleware())
	}
}

// WithPropagator устанавливает кастомный trace propagator
func WithPropagator(propagator propagation.TextMapPropagator) ClientOption {
	return func(c *Client) {
		c.propagator = propagator
	}
}

// WithTracer устанавливает кастомный tracer
func WithTracer(tracer trace.Tracer) ClientOption {
	return func(c *Client) {
		c.tracer = tracer
	}
}

// Subscribe подписывается на subject
func (c *Client) Subscribe(subject string, handler HandlerFunc) (*nats.Subscription, error) {
	return c.conn.Subscribe(subject, func(msg *nats.Msg) {
		// Извлекаем контекст с trace информацией
		ctx := c.extractContext(msg)

		// Создаём обёртку с контекстом
		message := &Message{
			Msg: msg,
			Ctx: ctx,
		}

		// Применяем middleware
		finalHandler := c.applyMiddlewares(handler)

		if err := finalHandler(message); err != nil {
			c.logger.Error("handler error",
				"subject", subject,
				"error", err,
			)
		}
	})
}

// QueueSubscribe подписывается на subject с queue group
func (c *Client) QueueSubscribe(subject, queue string, handler HandlerFunc) (*nats.Subscription, error) {
	return c.conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		ctx := c.extractContext(msg)

		message := &Message{
			Msg: msg,
			Ctx: ctx,
		}

		finalHandler := c.applyMiddlewares(handler)

		if err := finalHandler(message); err != nil {
			c.logger.Error("handler error",
				"subject", subject,
				"queue", queue,
				"error", err,
			)
		}
	})
}

// Publish публикует сообщение с автоматической пропагацией trace context
func (c *Client) Publish(ctx context.Context, subject string, data []byte) error {
	// Создаём span для публикации
	ctx, span := c.tracer.Start(ctx, fmt.Sprintf("nats.publish %s", subject))
	defer span.End()

	msg := nats.NewMsg(subject)
	msg.Data = data

	// Инжектируем trace context в headers
	c.propagator.Inject(ctx, &headerCarrier{header: msg.Header})

	return c.conn.PublishMsg(msg)
}

// PublishMsg публикует готовое сообщение с пропагацией trace context
func (c *Client) PublishMsg(ctx context.Context, msg *nats.Msg) error {
	ctx, span := c.tracer.Start(ctx, fmt.Sprintf("nats.publish %s", msg.Subject))
	defer span.End()

	// Инжектируем trace context в headers
	c.propagator.Inject(ctx, &headerCarrier{header: msg.Header})

	return c.conn.PublishMsg(msg)
}

// RequestMsg выполняет синхронный request-reply с пропагацией trace
func (c *Client) RequestMsg(ctx context.Context, msg *nats.Msg) (*Message, error) {
	// Создаём span для request
	ctx, span := c.tracer.Start(ctx, fmt.Sprintf("nats.request %s", msg.Subject))
	defer span.End()

	// Инжектируем trace context в headers
	c.propagator.Inject(ctx, &headerCarrier{header: msg.Header})

	resp, err := c.conn.RequestMsgWithContext(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Извлекаем trace context из ответа
	respCtx := c.extractContext(resp)

	return &Message{
		Msg: resp,
		Ctx: respCtx,
	}, nil
}

// Conn возвращает базовое NATS соединение для прямого использования
func (c *Client) Conn() *nats.Conn {
	return c.conn
}

// Close закрывает NATS соединение
func (c *Client) Close() {
	c.conn.Close()
}

// extractContext извлекает context из NATS сообщения с trace информацией
func (c *Client) extractContext(msg *nats.Msg) context.Context {
	ctx := context.Background()

	// Извлекаем trace context из headers через propagator
	if msg.Header != nil {
		ctx = c.propagator.Extract(ctx, &headerCarrier{header: msg.Header})
	}

	return ctx
}

// applyMiddlewares применяет все middleware к обработчику
func (c *Client) applyMiddlewares(handler HandlerFunc) HandlerFunc {
	// Применяем middleware в обратном порядке
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

// headerCarrier адаптер для передачи trace context через NATS headers
type headerCarrier struct {
	header nats.Header
}

func (hc *headerCarrier) Get(key string) string {
	return hc.header.Get(key)
}

func (hc *headerCarrier) Set(key, value string) {
	hc.header.Set(key, value)
}

func (hc *headerCarrier) Keys() []string {
	keys := make([]string, 0, len(hc.header))
	for k := range hc.header {
		keys = append(keys, k)
	}
	return keys
}
