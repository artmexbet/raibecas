package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPConfig struct {
	Host            string        `env:"HOST" env-default:"0.0.0.0"`
	Port            int           `env:"PORT" env-default:"8080"`
	Timeout         time.Duration `env:"TIMEOUT" env-default:"30s"`
	RPS             int           `env:"RPS" env-default:"100"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"5s"`
}

type NATSConfig struct {
	URL            string        `env:"URL" env-default:"nats://localhost:4222"`
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" env-default:"5s"`
	MaxReconnects  int           `env:"MAX_RECONNECTS" env-default:"10"`
	ReconnectWait  time.Duration `env:"RECONNECT_WAIT" env-default:"2s"`
}

type TelemetryConfig struct {
	Enabled        bool          `env:"ENABLED" env-default:"true"`
	ServiceName    string        `env:"SERVICE_NAME" env-default:"gateway"`
	ServiceVersion string        `env:"SERVICE_VERSION" env-default:"1.0.0"`
	OTLPEndpoint   string        `env:"OTLP_ENDPOINT" env-default:"localhost:4318"`
	ExportTimeout  time.Duration `env:"EXPORT_TIMEOUT" env-default:"30s"`
	BatchTimeout   time.Duration `env:"BATCH_TIMEOUT" env-default:"5s"`
	MaxQueueSize   int           `env:"MAX_QUEUE_SIZE" env-default:"2048"`
	MaxExportBatch int           `env:"MAX_EXPORT_BATCH" env-default:"512"`
}

type CORSConfig struct {
	AllowOrigins string `env:"ALLOW_ORIGINS" env-default:"http://localhost:3000"`
}

type ChatServiceConfig struct {
	WebSocketURL string `env:"WS_URL" env-default:"ws://localhost:8082/ws/chat"`
	HTTPURL      string `env:"HTTP_URL" env-default:"http://localhost:8082"`
}

type Config struct {
	HTTP        HTTPConfig        `env-prefix:"GATEWAY_"`
	NATS        NATSConfig        `env-prefix:"NATS_"`
	Telemetry   TelemetryConfig   `env-prefix:"TELEMETRY_"`
	CORS        CORSConfig        `env-prefix:"CORS_"`
	ChatService ChatServiceConfig `env-prefix:"CHAT_"`
}

// Load loads configuration from environment variables using cleanenv
func Load() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
