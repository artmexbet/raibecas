package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config holds all configuration for the auth service
type Config struct {
	Server   ServerConfig   `yaml:"server" env-prefix:"SERVER_"`
	Database DatabaseConfig `yaml:"database" env-prefix:"DB_"`
	Redis    RedisConfig    `yaml:"redis" env-prefix:"REDIS_"`
	NATS     NATSConfig     `yaml:"nats" env-prefix:"NATS_"`
	JWT      JWTConfig      `yaml:"jwt" env-prefix:"JWT_"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            string        `env:"PORT" env-default:"8081"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" env-default:"10s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" env-default:"10s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" env-default:"5s"`
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

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `env:"HOST" env-default:"localhost"`
	Port     string `env:"PORT" env-default:"6379"`
	Password string `env:"PASSWORD" env-default:""`
	DB       int    `env:"DB" env-default:"0"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL            string        `env:"URL" env-default:"nats://localhost:4222"`
	MaxReconnects  int           `env:"MAX_RECONNECTS" env-default:"10"`
	ReconnectWait  time.Duration `env:"RECONNECT_WAIT" env-default:"2s"`
	ConnectionName string        `env:"CONNECTION_NAME" env-default:"auth-service"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret          string        `env:"SECRET" env-required:"true"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TTL" env-default:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TTL" env-default:"168h"`
	Issuer          string        `env:"ISSUER" env-default:"raibecas-auth"`
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
