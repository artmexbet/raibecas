package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds all configuration for the auth service
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	NATS     NATSConfig     `yaml:"nats"`
	JWT      JWTConfig      `yaml:"jwt"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            string        `env:"SERVER_PORT" env-default:"8081"`
	ReadTimeout     time.Duration `env:"SERVER_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout    time.Duration `env:"SERVER_WRITE_TIMEOUT" env-default:"10s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" env-default:"5s"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string `env:"DB_HOST" env-default:"localhost"`
	Port     string `env:"DB_PORT" env-default:"5432"`
	User     string `env:"DB_USER" env-default:"raibecas"`
	Password string `env:"DB_PASSWORD" env-required:"true"`
	DBName   string `env:"DB_NAME" env-default:"raibecas"`
	SSLMode  string `env:"DB_SSL_MODE" env-default:"disable"`
	MaxConns int    `env:"DB_MAX_CONNS" env-default:"25"`
	MinConns int    `env:"DB_MIN_CONNS" env-default:"5"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `env:"REDIS_HOST" env-default:"localhost"`
	Port     string `env:"REDIS_PORT" env-default:"6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB" env-default:"0"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL            string        `env:"NATS_URL" env-default:"nats://localhost:4222"`
	MaxReconnects  int           `env:"NATS_MAX_RECONNECTS" env-default:"10"`
	ReconnectWait  time.Duration `env:"NATS_RECONNECT_WAIT" env-default:"2s"`
	ConnectionName string        `env:"NATS_CONNECTION_NAME" env-default:"auth-service"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret          string        `env:"JWT_SECRET" env-required:"true"`
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TTL" env-default:"15m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TTL" env-default:"168h"`
	Issuer          string        `env:"JWT_ISSUER" env-default:"raibecas-auth"`
}

// Load loads configuration from environment variables using cleanenv
func Load() (*Config, error) {
	var cfg Config

	// Read environment variables
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}
