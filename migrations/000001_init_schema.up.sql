CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS groups (
    id TEXT PRIMARY KEY,
    telegram_id BIGINT NOT NULL UNIQUE,
    title TEXT,
    greeting_enabled BOOLEAN DEFAULT FALSE,
    greeting_message TEXT,
    captcha_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS warns (
    id TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    reason TEXT,
    created_by BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_warns_user_group ON warns(user_id, group_id);

CREATE TABLE IF NOT EXISTS bans (
    id TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    until_date TIMESTAMP WITH TIME ZONE,
    type TEXT NOT NULL,
    reason TEXT,
    created_by BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_bans_user_group ON bans(user_id, group_id);
