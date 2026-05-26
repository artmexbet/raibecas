-- Create notes table for user annotations
CREATE TABLE IF NOT EXISTS notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    bookmark_id UUID REFERENCES document_bookmarks(id) ON DELETE SET NULL,
    position_in_document TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notes_user_created
    ON notes(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notes_document
    ON notes(document_id) WHERE document_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_notes_bookmark
    ON notes(bookmark_id) WHERE bookmark_id IS NOT NULL;
