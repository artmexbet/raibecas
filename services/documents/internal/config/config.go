package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Database  DatabaseConfig  `yaml:"database" env-prefix:"DB_"`
	NATS      NATSConfig      `yaml:"nats" env-prefix:"NATS_"`
	MinIO     MinIOConfig     `yaml:"minio" env-prefix:"MINIO_"`
	Telemetry TelemetryConfig `yaml:"telemetry" env-prefix:"TELEMETRY_"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string `env:"HOST" env-default:"localhost"`
	Port     string `env:"PORT" env-default:"5432"`
	User     string `env:"USER" env-default:"raibecas"`
	Password string `env:"PASSWORD" env-required:"true"`
	DBName   string `env:"NAME" env-default:"raibecas"`
	SSLMode  string `env:"SSL_MODE" env-default:"disable"`
	MaxConns int    `env:"MAX_CONNS" env-default:"25"`
	MinConns int    `env:"MIN_CONNS" env-default:"5"`
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL            string `env:"URL" env-default:"nats://localhost:4222"`
	ConnectionName string `env:"CONNECTION_NAME" env-default:"documents-service"`
	MaxReconnects  int    `env:"MAX_RECONNECTS" env-default:"-1"`
}

// MinIOConfig holds MinIO configuration
type MinIOConfig struct {
	Endpoint  string `env:"ENDPOINT" env-default:"localhost:9000"`
	AccessKey string `env:"ACCESS_KEY" env-default:"raibecas"`
	SecretKey string `env:"SECRET_KEY" env-required:"true"`
	Bucket    string `env:"BUCKET" env-default:"raibecas-documents"`
	UseSSL    bool   `env:"USE_SSL" env-default:"false"`
}

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	Enabled        bool   `env:"ENABLED" env-default:"true"`
	ServiceName    string `env:"SERVICE_NAME" env-default:"documents"`
	ServiceVersion string `env:"SERVICE_VERSION" env-default:"1.0.0"`
	OTLPEndpoint   string `env:"OTLP_ENDPOINT" env-default:"localhost:4318"`
	ExportTimeout  string `env:"EXPORT_TIMEOUT" env-default:"30s"`
	BatchTimeout   string `env:"BATCH_TIMEOUT" env-default:"5s"`
	MaxQueueSize   int    `env:"MAX_QUEUE_SIZE" env-default:"2048"`
	MaxExportBatch int    `env:"MAX_EXPORT_BATCH" env-default:"512"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	return cfg, nil
}
