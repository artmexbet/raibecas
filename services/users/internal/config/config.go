package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type NATS struct {
	Host string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port int    `yaml:"port" env:"PORT" env-default:"4222"`
}

func (n *NATS) GetURL() string {
	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

type Metrics struct {
	Port int `yaml:"port" env:"PORT" env-default:"9091"`
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

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	Enabled        bool          `env:"ENABLED" env-default:"true"`
	ServiceName    string        `env:"SERVICE_NAME" env-default:"users"`
	ServiceVersion string        `env:"SERVICE_VERSION" env-default:"1.0.0"`
	OTLPEndpoint   string        `env:"OTLP_ENDPOINT" env-default:"localhost:4318"`
	ExportTimeout  time.Duration `env:"EXPORT_TIMEOUT" env-default:"30s"`
	BatchTimeout   time.Duration `env:"BATCH_TIMEOUT" env-default:"5s"`
	MaxQueueSize   int           `env:"MAX_QUEUE_SIZE" env-default:"2048"`
	MaxExportBatch int           `env:"MAX_EXPORT_BATCH" env-default:"512"`
}

type Config struct {
	NATS      NATS            `yaml:"nats" env-prefix:"NATS_"`
	Database  DatabaseConfig  `yaml:"database" env-prefix:"DB_"`
	Metrics   Metrics         `yaml:"metrics" env-prefix:"METRICS_"`
	Telemetry TelemetryConfig `yaml:"telemetry" env-prefix:"TELEMETRY_"`
}

func Load() (Config, error) {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)
	return cfg, err
}
