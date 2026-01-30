CREATE TABLE IF NOT EXISTS blacklists (
    id TEXT PRIMARY KEY,
    group_id BIGINT NOT NULL,
    type TEXT NOT NULL, -- regex, sticker_set, emoji
    value TEXT NOT NULL,
    action TEXT NOT NULL DEFAULT 'delete', -- delete, soft_warn, hard_warn, kick, mute, ban
    action_duration TEXT, -- e.g. "1h", optional
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(group_id, type, value)
);

CREATE INDEX idx_blacklists_group ON blacklists(group_id);
