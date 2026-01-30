CREATE TABLE IF NOT EXISTS approved_users (
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by BIGINT,
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX idx_approved_users_group ON approved_users(group_id);
