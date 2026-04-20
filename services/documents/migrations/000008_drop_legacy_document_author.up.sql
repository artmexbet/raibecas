DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'documents'
          AND column_name = 'author_id'
    ) THEN
        INSERT INTO document_authors (document_id, author_id, type_id)
        SELECT d.id, d.author_id, at.id
        FROM documents d
        JOIN authorship_types at ON at.title = 'автор'
        WHERE d.author_id IS NOT NULL
        ON CONFLICT DO NOTHING;

        DROP INDEX IF EXISTS idx_documents_author;
        ALTER TABLE documents DROP COLUMN author_id;
    END IF;
END $$;

