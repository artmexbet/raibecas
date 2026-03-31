-- Create bookmarks table for saved publications and quotes
CREATE TABLE IF NOT EXISTS document_bookmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    kind VARCHAR(32) NOT NULL,
    quote_text TEXT,
    quote_context TEXT,
    page_label VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_document_bookmarks_kind CHECK (kind IN ('publication', 'quote')),
    CONSTRAINT chk_document_bookmarks_quote CHECK (
        (kind = 'publication' AND quote_text IS NULL AND quote_context IS NULL AND page_label IS NULL) OR
        (kind = 'quote' AND quote_text IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_document_bookmarks_publication
    ON document_bookmarks(user_id, document_id, kind)
    WHERE kind = 'publication';

CREATE INDEX IF NOT EXISTS idx_document_bookmarks_user_created
    ON document_bookmarks(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_document_bookmarks_document
    ON document_bookmarks(document_id);

CREATE INDEX IF NOT EXISTS idx_document_bookmarks_user_kind
    ON document_bookmarks(user_id, kind);

