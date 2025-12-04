package chunker

import (
	"strings"
)

type Config struct {
	ChunkSize    int
	ChunkOverlap int
	MaxChunks    int
}

type Chunk struct {
	Text     string
	Ordinal  int
	Metadata map[string]string
}

func NewDefaultConfig() Config {
	return Config{ChunkSize: 700, ChunkOverlap: 80, MaxChunks: 0}
}

func SplitText(cfg Config, text string) []Chunk {
	if cfg.ChunkSize <= 0 {
		return nil
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	var chunks []Chunk
	step := cfg.ChunkSize - cfg.ChunkOverlap
	if step <= 0 {
		step = cfg.ChunkSize
	}
	for idx, start := 0, 0; start < len(runes); idx, start = idx+1, start+step {
		end := start + cfg.ChunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk == "" {
			continue
		}
		chunks = append(chunks, Chunk{Text: chunk, Ordinal: idx})
		if cfg.MaxChunks > 0 && len(chunks) >= cfg.MaxChunks {
			break
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}
