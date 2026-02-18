ALTER TABLE groups ADD COLUMN log_channel_id BIGINT;
ALTER TABLE groups ADD COLUMN log_categories TEXT DEFAULT '[]';
