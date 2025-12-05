package config

import (
	"os"
	"testing"
	"time"
)

func TestHTTP_Address(t *testing.T) {
	tests := []struct {
		name string
		http HTTP
		want string
	}{
		{
			name: "default values",
			http: HTTP{Host: "0.0.0.0", Port: "8082"},
			want: "0.0.0.0:8082",
		},
		{
			name: "custom values",
			http: HTTP{Host: "localhost", Port: "9000"},
			want: "localhost:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.http.Address(); got != tt.want {
				t.Errorf("HTTP.Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQdrant_Address(t *testing.T) {
	tests := []struct {
		name   string
		qdrant Qdrant
		want   string
	}{
		{
			name:   "default values",
			qdrant: Qdrant{Host: "localhost", Port: 6333},
			want:   "localhost:6333",
		},
		{
			name:   "custom values",
			qdrant: Qdrant{Host: "qdrant.example.com", Port: 6334},
			want:   "qdrant.example.com:6334",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.qdrant.Address(); got != tt.want {
				t.Errorf("Qdrant.Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOllama_Address(t *testing.T) {
	tests := []struct {
		name   string
		ollama Ollama
		want   string
	}{
		{
			name:   "default http",
			ollama: Ollama{Protocol: "http", Host: "localhost", Port: "11434"},
			want:   "http://localhost:11434",
		},
		{
			name:   "https",
			ollama: Ollama{Protocol: "https", Host: "ollama.example.com", Port: "443"},
			want:   "https://ollama.example.com:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ollama.Address(); got != tt.want {
				t.Errorf("Ollama.Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Сохраняем и очищаем переменные окружения
	oldEnv := make(map[string]string)
	envVars := []string{
		"HTTP_HOST", "HTTP_PORT", "NATS_URL", "QDRANT_HOST", "QDRANT_PORT",
		"OLLAMA_HOST", "OLLAMA_PORT", "STORAGE_BASE_DIR", "USE_NATS",
	}

	for _, key := range envVars {
		if val, exists := os.LookupEnv(key); exists {
			oldEnv[key] = val
		}
		os.Unsetenv(key)
	}

	defer func() {
		for key, val := range oldEnv {
			os.Setenv(key, val)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Проверяем значения по умолчанию
	if cfg.HTTP.Host != "0.0.0.0" {
		t.Errorf("HTTP.Host = %v, want 0.0.0.0", cfg.HTTP.Host)
	}
	if cfg.HTTP.Port != "8082" {
		t.Errorf("HTTP.Port = %v, want 8082", cfg.HTTP.Port)
	}
	if cfg.Qdrant.Port != 6333 {
		t.Errorf("Qdrant.Port = %v, want 6333", cfg.Qdrant.Port)
	}
	if cfg.Ollama.Port != "11434" {
		t.Errorf("Ollama.Port = %v, want 11434", cfg.Ollama.Port)
	}
	if cfg.Pipeline.ChunkSize != 700 {
		t.Errorf("Pipeline.ChunkSize = %v, want 700", cfg.Pipeline.ChunkSize)
	}
	if cfg.UseNATS != false {
		t.Errorf("UseNATS = %v, want false", cfg.UseNATS)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("HTTP_HOST", "testhost")
	os.Setenv("HTTP_PORT", "9999")
	os.Setenv("QDRANT_HOST", "qdrant-test")
	os.Setenv("QDRANT_PORT", "7777")
	os.Setenv("USE_NATS", "true")

	defer func() {
		os.Unsetenv("HTTP_HOST")
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("QDRANT_HOST")
		os.Unsetenv("QDRANT_PORT")
		os.Unsetenv("USE_NATS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Host != "testhost" {
		t.Errorf("HTTP.Host = %v, want testhost", cfg.HTTP.Host)
	}
	if cfg.HTTP.Port != "9999" {
		t.Errorf("HTTP.Port = %v, want 9999", cfg.HTTP.Port)
	}
	if cfg.Qdrant.Host != "qdrant-test" {
		t.Errorf("Qdrant.Host = %v, want qdrant-test", cfg.Qdrant.Host)
	}
	if cfg.Qdrant.Port != 7777 {
		t.Errorf("Qdrant.Port = %v, want 7777", cfg.Qdrant.Port)
	}
	if cfg.UseNATS != true {
		t.Errorf("UseNATS = %v, want true", cfg.UseNATS)
	}
}

func TestNATS_Defaults(t *testing.T) {
	os.Unsetenv("NATS_URL")
	os.Unsetenv("NATS_STREAM")
	os.Unsetenv("NATS_ACK_WAIT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.NATS.URL != "nats://localhost:4222" {
		t.Errorf("NATS.URL = %v, want nats://localhost:4222", cfg.NATS.URL)
	}
	if cfg.NATS.Stream != "DOCUMENTS" {
		t.Errorf("NATS.Stream = %v, want DOCUMENTS", cfg.NATS.Stream)
	}
	if cfg.NATS.AckWait != 30*time.Second {
		t.Errorf("NATS.AckWait = %v, want 30s", cfg.NATS.AckWait)
	}
}
