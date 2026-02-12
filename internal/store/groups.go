package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Group struct {
	ID                        string
	TelegramID                int64
	Title                     string
	GreetingEnabled           bool
	GreetingMessage           string
	GoodbyeEnabled            bool
	GoodbyeMessage            string
	CaptchaEnabled            bool
	AntiraidUntil             *time.Time
	RaidActionTime            string
	AutoAntiraidThreshold     int
	AntifloodConsecutiveLimit int
	AntifloodTimerLimit       int
	AntifloodTimerDuration    string
	AntifloodAction           string
	AntifloodDelete           bool
	WarnLimit                 int
	WarnAction                string
	WarnDuration              string
	NotesPrivate              bool
	ActionTopicID             *int64
	CreatedAt                 any
}

func (s *Store) GetGroup(telegramID int64) (*Group, error) {
	cacheKey := "group:" + strconv.FormatInt(telegramID, 10)
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		var g Group
		if err := json.Unmarshal(val, &g); err == nil {
			return &g, nil
		}
	}

	q := `SELECT id, telegram_id, title, greeting_enabled, greeting_message, goodbye_enabled, goodbye_message, captcha_enabled,
                 antiraid_until, raid_action_time, auto_antiraid_threshold,
                 antiflood_consecutive_limit, antiflood_timer_limit, antiflood_timer_duration, antiflood_action, antiflood_delete,
                 warn_limit, warn_action, warn_duration, notes_private, action_topic_id
          FROM groups WHERE telegram_id = $1`

	var g Group
	err = s.db.QueryRow(context.Background(), q, telegramID).Scan(
		&g.ID, &g.TelegramID, &g.Title, &g.GreetingEnabled, &g.GreetingMessage, &g.GoodbyeEnabled, &g.GoodbyeMessage, &g.CaptchaEnabled,
		&g.AntiraidUntil, &g.RaidActionTime, &g.AutoAntiraidThreshold,
		&g.AntifloodConsecutiveLimit, &g.AntifloodTimerLimit, &g.AntifloodTimerDuration, &g.AntifloodAction, &g.AntifloodDelete,
		&g.WarnLimit, &g.WarnAction, &g.WarnDuration, &g.NotesPrivate, &g.ActionTopicID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if data, err := json.Marshal(g); err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Set().Key(cacheKey).Value(string(data)).Ex(1*time.Hour).Build())
	}

	return &g, nil
}

func (s *Store) CreateGroup(telegramID int64, title string) error {
	id, err := gonanoid.New()
	if err != nil {
		return err
	}
	q := `INSERT INTO groups (id, telegram_id, title, greeting_enabled, greeting_message, goodbye_enabled, goodbye_message) 
          VALUES ($1, $2, $3, true, 'Welcome {firstname} (ID: {userid}) to the group!', true, 'Goodbye {firstname} (ID: {userid}), see you soon!') 
          ON CONFLICT (telegram_id) DO UPDATE SET title = $3`
	_, err = s.db.Exec(context.Background(), q, id, telegramID, title)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetGreetingStatus(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET greeting_enabled = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetGreetingMessage(telegramID int64, message string) error {
	q := `UPDATE groups SET greeting_message = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, message, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetGoodbyeStatus(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET goodbye_enabled = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetGoodbyeMessage(telegramID int64, message string) error {
	q := `UPDATE groups SET goodbye_message = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, message, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) UpdateGroupCaptcha(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET captcha_enabled = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetAntiraidUntil(telegramID int64, until *time.Time) error {
	q := `UPDATE groups SET antiraid_until = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, until, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetRaidActionTime(telegramID int64, duration string) error {
	q := `UPDATE groups SET raid_action_time = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, duration, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetAutoAntiraidThreshold(telegramID int64, threshold int) error {
	q := `UPDATE groups SET auto_antiraid_threshold = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, threshold, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetAntifloodConsecutiveLimit(telegramID int64, limit int) error {
	q := `UPDATE groups SET antiflood_consecutive_limit = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, limit, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetAntifloodTimer(telegramID int64, limit int, duration string) error {
	q := `UPDATE groups SET antiflood_timer_limit = $1, antiflood_timer_duration = $2 WHERE telegram_id = $3`
	_, err := s.db.Exec(context.Background(), q, limit, duration, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetAntifloodAction(telegramID int64, action string) error {
	q := `UPDATE groups SET antiflood_action = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, action, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetAntifloodDelete(telegramID int64, delete bool) error {
	q := `UPDATE groups SET antiflood_delete = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, delete, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetWarnLimit(telegramID int64, limit int) error {
	q := `UPDATE groups SET warn_limit = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, limit, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetWarnAction(telegramID int64, action string) error {
	q := `UPDATE groups SET warn_action = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, action, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetWarnDuration(telegramID int64, duration string) error {
	q := `UPDATE groups SET warn_duration = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, duration, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetNotesPrivate(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET notes_private = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetActionTopic(telegramID, topicID int64) error {
	q := `UPDATE groups SET action_topic_id = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, topicID, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}
