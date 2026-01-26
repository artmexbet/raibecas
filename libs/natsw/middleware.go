package natsw

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"
)

// LoggingMiddleware логирует все входящие и исходящие сообщения
func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg *Message) error {
			start := time.Now()

			logger.Debug("nats message received",
				"subject", msg.Subject,
				"reply", msg.Reply,
				"size", len(msg.Data),
			)

			err := next(msg)

			duration := time.Since(start)

			if err != nil {
				logger.Error("nats handler failed",
					"subject", msg.Subject,
					"duration", duration,
					"error", err,
				)
			} else {
				logger.Debug("nats handler completed",
					"subject", msg.Subject,
					"duration", duration,
				)
			}

			return err
		}
	}
}

// RecoverMiddleware защищает от паник в обработчиках
func RecoverMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg *Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					err = fmt.Errorf("panic recovered: %v\n%s", r, stack)

					slog.Error("panic in nats handler",
						"subject", msg.Subject,
						"panic", r,
						"stack", string(stack),
					)
				}
			}()

			return next(msg)
		}
	}
}

// TimeoutMiddleware добавляет таймаут для обработки сообщений
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg *Message) error {
			ctx, cancel := context.WithTimeout(msg.Ctx, timeout)
			defer cancel()

			// Обновляем контекст в сообщении
			msg.Ctx = ctx

			done := make(chan error, 1)

			go func() {
				done <- next(msg)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return fmt.Errorf("handler timeout exceeded (%v): %w", timeout, ctx.Err())
			}
		}
	}
}

// RetryMiddleware повторяет обработку сообщения при ошибках
func RetryMiddleware(maxRetries int, delay time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg *Message) error {
			var err error

			for attempt := 0; attempt <= maxRetries; attempt++ {
				err = next(msg)
				if err == nil {
					return nil
				}

				if attempt < maxRetries {
					slog.Warn("handler failed, retrying",
						"subject", msg.Subject,
						"attempt", attempt+1,
						"max_retries", maxRetries,
						"error", err,
					)
					time.Sleep(delay * time.Duration(attempt+1))
				}
			}

			return fmt.Errorf("handler failed after %d retries: %w", maxRetries, err)
		}
	}
}

// MetadataMiddleware добавляет метаданные в контекст
func MetadataMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg *Message) error {
			// Извлекаем метаданные из headers и добавляем в контекст
			if msg.Header != nil {
				ctx := msg.Ctx
				if requestID := msg.Header.Get("X-Request-Id"); requestID != "" {
					ctx = context.WithValue(ctx, "request_id", requestID)
				}
				if userID := msg.Header.Get("X-User-Id"); userID != "" {
					ctx = context.WithValue(ctx, "user_id", userID)
				}
				msg.Ctx = ctx
			}

			return next(msg)
		}
	}
}

// RateLimitMiddleware ограничивает количество сообщений в секунду
func RateLimitMiddleware(messagesPerSecond int) Middleware {
	ticker := time.NewTicker(time.Second / time.Duration(messagesPerSecond))

	return func(next HandlerFunc) HandlerFunc {
		return func(msg *Message) error {
			select {
			case <-ticker.C:
				return next(msg)
			case <-msg.Ctx.Done():
				return msg.Ctx.Err()
			}
		}
	}
}

// ValidationMiddleware проверяет валидность сообщения перед обработкой
type Validator interface {
	Validate() error
}

func ValidationMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(msg *Message) error {
			// Базовая валидация
			if msg.Subject == "" {
				return fmt.Errorf("empty subject")
			}

			return next(msg)
		}
	}
}

// ChainMiddleware объединяет несколько middleware в одну цепочку
func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
