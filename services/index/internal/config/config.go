package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTP struct {
	Host    string        `yaml:"host" env:"HOST" env-default:"0.0.0.0"`
	Port    string        `yaml:"port" env:"PORT" env-default:"8082"`
	Timeout time.Duration `yaml:"timeout" env:"TIMEOUT" env-default:"15s"`
}

func (h *HTTP) Address() string {
	return fmt.Sprintf("%s:%s", h.Host, h.Port)
}

type NATS struct {
	URL      string        `yaml:"url" env:"URL" env-default:"nats://localhost:4222"`
	Stream   string        `yaml:"stream" env:"STREAM" env-default:"DOCUMENTS"`
	Subject  string        `yaml:"subject" env:"SUBJECT" env-default:"index.documents"`
	Queue    string        `yaml:"queue" env:"QUEUE" env-default:"index-workers"`
	Durable  string        `yaml:"durable" env:"DURABLE" env-default:"index-service"`
	AckWait  time.Duration `yaml:"ack_wait" env:"ACK_WAIT" env-default:"30s"`
	MaxInFly int           `yaml:"max_in_flight" env:"MAX_IN_FLIGHT" env-default:"32"`
}

type Qdrant struct {
	Host            string `yaml:"host" env:"HOST" env-default:"localhost"`
	Port            int    `yaml:"port" env:"PORT" env-default:"6333"`
	CollectionName  string `yaml:"collection_name" env:"COLLECTION_NAME" env-default:"documents"`
	VectorDimension uint64 `yaml:"vector_dimension" env:"VECTOR_DIMENSION" env-default:"768"`
	Distance        string `yaml:"distance" env:"DISTANCE" env-default:"Cosine"`
}

func (q *Qdrant) Address() string {
	return fmt.Sprintf("%s:%d", q.Host, q.Port)
}

type Ollama struct {
	Protocol       string        `yaml:"protocol" env:"PROTOCOL" env-default:"http"`
	Host           string        `yaml:"host" env:"HOST" env-default:"localhost"`
	Port           string        `yaml:"port" env:"PORT" env-default:"11434"`
	EmbeddingModel string        `yaml:"embedding_model" env:"EMBEDDING_MODEL" env-default:"embeddinggemma"`
	Timeout        time.Duration `yaml:"timeout" env:"TIMEOUT" env-default:"30s"`
}

func (o *Ollama) Address() string {
	return fmt.Sprintf("%s://%s:%s", o.Protocol, o.Host, o.Port)
}

type Pipeline struct {
	ChunkSize    int `yaml:"chunk_size" env:"CHUNK_SIZE" env-default:"700"`
	ChunkOverlap int `yaml:"chunk_overlap" env:"CHUNK_OVERLAP" env-default:"80"`
	MaxChunks    int `yaml:"max_chunks" env:"MAX_CHUNKS" env-default:"500"`
}

type Storage struct {
	Type    string `yaml:"type" env:"TYPE" env-default:"filesystem"`
	BaseDir string `yaml:"base_dir" env:"BASE_DIR" env-default:"./data/documents"`
}

type Config struct {
	HTTP     HTTP     `yaml:"http" env-prefix:"HTTP_"`
	NATS     NATS     `yaml:"nats" env-prefix:"NATS_"`
	Qdrant   Qdrant   `yaml:"qdrant" env-prefix:"QDRANT_"`
	Ollama   Ollama   `yaml:"ollama" env-prefix:"OLLAMA_"`
	Pipeline Pipeline `yaml:"pipeline" env-prefix:"PIPELINE_"`
	Storage  Storage  `yaml:"storage" env-prefix:"STORAGE_"`
	UseNATS  bool     `yaml:"use_nats" env:"USE_NATS" env-default:"false"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
