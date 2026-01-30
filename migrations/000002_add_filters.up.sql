CREATE TABLE IF NOT EXISTS filters (
    id TEXT PRIMARY KEY,
    group_id BIGINT NOT NULL,
    trigger TEXT NOT NULL,
    response TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(group_id, trigger)
);

CREATE INDEX idx_filters_group ON filters(group_id);
