CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    content TEXT,
    type TEXT NOT NULL,
    file_id TEXT,
    created_by BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(chat_id, name)
);

ALTER TABLE groups ADD COLUMN IF NOT EXISTS notes_private BOOLEAN DEFAULT FALSE;
