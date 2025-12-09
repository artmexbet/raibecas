package domain

type Document struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`   // прямой контент (для legacy API)
	FilePath  string            `json:"file_path"` // путь к файлу в storage
	SourceURI string            `json:"source_uri"`
	Metadata  map[string]string `json:"metadata"`
}

type Chunk struct {
	DocumentID string            `json:"document_id"`
	Ordinal    int               `json:"ordinal"`
	Text       string            `json:"text"`
	Embedding  []float64         `json:"embedding"`
	Metadata   map[string]string `json:"metadata"`
}
