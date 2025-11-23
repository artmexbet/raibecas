package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Qdrant struct {
	Host string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port int    `yaml:"port" env:"PORT" env-default:"6333"`

	CollectionName  string `yaml:"collection_name" env:"COLLECTION_NAME" env-default:"documents"`
	RetrievePayload bool   `yaml:"retrieve_payload" env:"RETRIEVE_PAYLOAD" env-default:"true"`
	CountOfResults  uint64 `yaml:"count_of_results" env:"COUNT_OF_RESULTS" env-default:"5"`
}

func (q *Qdrant) GetAddress() string {
	return fmt.Sprintf("%s:%d", q.Host, q.Port)
}

type Ollama struct {
	Protocol string `yaml:"protocol" env:"PROTOCOL" env-default:"http"`
	Host     string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port     string `yaml:"port" env:"PORT" env-default:"11434"`

	EmbeddingModel  string `yaml:"embedding-model" env:"EMBEDDING_MODEL" env-default:"embeddinggemma"`
	GenerationModel string `yaml:"generation-model" env:"GENERATION_MODEL" env-default:"gemma3:4b"`
}

func (o *Ollama) GetAddress() string {
	return fmt.Sprintf("%s://%s:%s", o.Protocol, o.Host, o.Port)
}

type Redis struct {
	Host string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port string `yaml:"port" env:"PORT" env-default:"6379"`
}

type Config struct {
	Qdrant *Qdrant `yaml:"qdrant" env-prefix:"QDRANT_"`
	Ollama *Ollama `yaml:"ollama" env-prefix:"OLLAMA_"`
	Redis  *Redis  `yaml:"redis" env-prefix:"REDIS_"`
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
