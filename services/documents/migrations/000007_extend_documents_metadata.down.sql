DROP INDEX IF EXISTS idx_document_authors_type;
DROP INDEX IF EXISTS idx_document_authors_author;
DROP INDEX IF EXISTS idx_document_authors_document;
DROP INDEX IF EXISTS idx_documents_document_type;

ALTER TABLE documents
    DROP COLUMN IF EXISTS document_type_id;

DROP TABLE IF EXISTS document_authors;
DROP TABLE IF EXISTS authorship_types;
DROP TABLE IF EXISTS document_types;

