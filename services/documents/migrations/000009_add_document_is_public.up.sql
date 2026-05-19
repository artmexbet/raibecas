ALTER TABLE documents ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE documents SET is_public = TRUE;
CREATE INDEX IF NOT EXISTS idx_documents_is_public ON documents(is_public);
