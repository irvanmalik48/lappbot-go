ALTER TABLE groups ADD COLUMN antiraid_until TIMESTAMP WITH TIME ZONE;
ALTER TABLE groups ADD COLUMN raid_action_time VARCHAR(255) DEFAULT '1h';
ALTER TABLE groups ADD COLUMN auto_antiraid_threshold INT DEFAULT 0;

ALTER TABLE groups ADD COLUMN antiflood_consecutive_limit INT DEFAULT 0;
ALTER TABLE groups ADD COLUMN antiflood_timer_limit INT DEFAULT 0;
ALTER TABLE groups ADD COLUMN antiflood_timer_duration VARCHAR(255) DEFAULT '';
ALTER TABLE groups ADD COLUMN antiflood_action VARCHAR(255) DEFAULT 'mute';
ALTER TABLE groups ADD COLUMN antiflood_delete BOOLEAN DEFAULT false;
