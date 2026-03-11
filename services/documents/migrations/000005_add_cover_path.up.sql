-- Add cover_path column to documents table for cover image support
ALTER TABLE documents ADD COLUMN IF NOT EXISTS cover_path VARCHAR(500);

