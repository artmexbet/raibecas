-- Create document_versions table
CREATE TABLE IF NOT EXISTS document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version INT NOT NULL,
    content_path VARCHAR(500) NOT NULL,
    changes TEXT,
    created_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(document_id, version)
);

-- Create index
CREATE INDEX IF NOT EXISTS idx_document_versions_document ON document_versions(document_id, version DESC);
