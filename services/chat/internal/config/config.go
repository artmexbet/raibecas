package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Qdrant struct {
	Host string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port int    `yaml:"port" env:"PORT" env-default:"6333"`

	CollectionName  string `yaml:"collection_name" env:"COLLECTION_NAME" env-default:"documents"`
	RetrievePayload bool   `yaml:"retrieve_payload" env:"RETRIEVE_PAYLOAD" env-default:"true"`
	CountOfResults  uint64 `yaml:"count_of_results" env:"COUNT_OF_RESULTS" env-default:"5"`
	VectorDimension uint64 `yaml:"vector_dimension" env:"VECTOR_DIMENSION" env-default:"768"`
}

func (q *Qdrant) GetAddress() string {
	return fmt.Sprintf("%s:%d", q.Host, q.Port)
}

type ContextGeneration struct {
	VectorDimension int    `yaml:"vector_dimension" env:"VECTOR_DIMENSION"`
	BasePrompt      string `yaml:"base_prompt" env:"BASE_PROMPT"` //todo: maybe use file?
	ContextPrompt   string `yaml:"context_prompt" env:"CONTEXT_PROMPT"`
	QueryPrompt     string `yaml:"query_prompt" env:"QUERY_PROMPT"`
}

type Ollama struct {
	Protocol string `yaml:"protocol" env:"PROTOCOL" env-default:"http"`
	Host     string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port     string `yaml:"port" env:"PORT" env-default:"11434"`

	EmbeddingModel  string `yaml:"embedding-model" env:"EMBEDDING_MODEL" env-default:"embeddinggemma"`
	GenerationModel string `yaml:"generation-model" env:"GENERATION_MODEL" env-default:"gemma3:4b"`

	StreamAnswers bool              `yaml:"stream_answers" env:"STREAM_ANSWERS" env-default:"false"`
	Context       ContextGeneration `yaml:"context_generation" env-prefix:"CONTEXT_GENERATION_"`
}

func (o *Ollama) GetAddress() string {
	return fmt.Sprintf("%s://%s:%s", o.Protocol, o.Host, o.Port)
}

type Redis struct {
	Host       string        `yaml:"host" env:"HOST" env-default:"localhost"`
	Port       string        `yaml:"port" env:"PORT" env-default:"6379"`
	DB         int           `yaml:"db" env:"DB" env-default:"0"`
	ChatTTL    time.Duration `yaml:"chat_ttl" env:"CHAT_TTL" env-default:"86400s"` // 24 hours in seconds
	MessageTTL time.Duration `yaml:"message_ttl" env:"MESSAGE_TTL" env-default:"86400s"`
}

func (r *Redis) GetAddress() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type HTTP struct {
	Host string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port string `yaml:"port" env:"PORT" env-default:"8080"`
}

func (h *HTTP) GetAddress() string {
	return fmt.Sprintf("%s:%s", h.Host, h.Port)
}

type Config struct {
	Qdrant Qdrant `yaml:"qdrant" env-prefix:"QDRANT_"`
	Ollama Ollama `yaml:"ollama" env-prefix:"OLLAMA_"`
	Redis  Redis  `yaml:"redis" env-prefix:"REDIS_"`

	HTTP    HTTP `yaml:"http" env-prefix:"HTTP_"`
	UseHTTP bool `yaml:"use_http" env:"USE_HTTP" env-default:"false"`
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
