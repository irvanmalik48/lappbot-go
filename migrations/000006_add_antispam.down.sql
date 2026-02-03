ALTER TABLE groups DROP COLUMN antiraid_until;
ALTER TABLE groups DROP COLUMN raid_action_time;
ALTER TABLE groups DROP COLUMN auto_antiraid_threshold;

ALTER TABLE groups DROP COLUMN antiflood_consecutive_limit;
ALTER TABLE groups DROP COLUMN antiflood_timer_limit;
ALTER TABLE groups DROP COLUMN antiflood_timer_duration;
ALTER TABLE groups DROP COLUMN antiflood_action;
ALTER TABLE groups DROP COLUMN antiflood_delete;
