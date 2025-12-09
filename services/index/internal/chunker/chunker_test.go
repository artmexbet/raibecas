package chunker

import "testing"

func TestSplitTextBasic(t *testing.T) {
	cfg := Config{ChunkSize: 10, ChunkOverlap: 2}
	chunks := SplitText(cfg, "abcdefghijklmnopqrstuvwxyz")
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	if chunks[0].Text != "abcdefghij" {
		t.Fatalf("unexpected first chunk: %q", chunks[0].Text)
	}
}

func TestSplitTextEmpty(t *testing.T) {
	cfg := Config{ChunkSize: 5}
	chunks := SplitText(cfg, "   ")
	if len(chunks) != 0 {
		t.Fatal("expected no chunks")
	}
}
