CREATE TABLE IF NOT EXISTS document_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS authorship_types (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO document_types (name)
VALUES
    ('Не указан'),
    ('Статья'),
    ('Монография'),
    ('Сборник'),
    ('Рецензия')
ON CONFLICT (name) DO NOTHING;

INSERT INTO authorship_types (title)
VALUES
    ('автор'),
    ('редактор'),
    ('рецензент')
ON CONFLICT (title) DO NOTHING;

ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS document_type_id INT REFERENCES document_types(id);

UPDATE documents
SET document_type_id = dt.id
FROM document_types dt
WHERE documents.document_type_id IS NULL
  AND dt.name = 'Не указан';

ALTER TABLE documents
    ALTER COLUMN document_type_id SET NOT NULL;

ALTER TABLE documents
    ALTER COLUMN category_id DROP NOT NULL;

CREATE TABLE IF NOT EXISTS document_authors (
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    type_id INT NOT NULL REFERENCES authorship_types(id),
    PRIMARY KEY (document_id, author_id, type_id)
);

INSERT INTO document_authors (document_id, author_id, type_id)
SELECT d.id, d.author_id, at.id
FROM documents d
CROSS JOIN authorship_types at
WHERE d.author_id IS NOT NULL
  AND at.title = 'автор'
ON CONFLICT DO NOTHING;

ALTER TABLE documents
    DROP COLUMN IF EXISTS author_id;

CREATE INDEX IF NOT EXISTS idx_documents_document_type ON documents(document_type_id);
CREATE INDEX IF NOT EXISTS idx_document_authors_document ON document_authors(document_id);
CREATE INDEX IF NOT EXISTS idx_document_authors_author ON document_authors(author_id);
CREATE INDEX IF NOT EXISTS idx_document_authors_type ON document_authors(type_id);

