ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS author_id UUID REFERENCES authors(id);

UPDATE documents d
SET author_id = source.author_id
FROM (
    SELECT DISTINCT ON (da.document_id)
        da.document_id,
        da.author_id
    FROM document_authors da
    JOIN authorship_types at ON at.id = da.type_id
    WHERE at.title = 'автор'
    ORDER BY da.document_id, da.author_id
) AS source
WHERE d.id = source.document_id
  AND d.author_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_documents_author ON documents(author_id);

