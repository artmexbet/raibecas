package domain

type DocumentIndexEvent struct {
	DocumentID string            `json:"document_id"`
	Title      string            `json:"title"`
	FilePath   string            `json:"file_path"` // путь к файлу в storage
	SourceURI  string            `json:"source_uri"`
	Metadata   map[string]string `json:"metadata"`
}
