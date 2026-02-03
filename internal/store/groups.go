package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Group struct {
	ID              string
	TelegramID      int64
	Title           string
	GreetingEnabled bool
	GreetingMessage string
	GoodbyeEnabled  bool
	GoodbyeMessage  string
	CaptchaEnabled  bool
	CreatedAt       interface{}
}

func (s *Store) GetGroup(telegramID int64) (*Group, error) {
	cacheKey := fmt.Sprintf("group:%d", telegramID)
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		var g Group
		if err := json.Unmarshal(val, &g); err == nil {
			return &g, nil
		}
	}

	q := `SELECT id, telegram_id, title, greeting_enabled, greeting_message, goodbye_enabled, goodbye_message, captcha_enabled 
          FROM groups WHERE telegram_id = $1`

	var g Group
	err = s.db.QueryRow(context.Background(), q, telegramID).Scan(
		&g.ID, &g.TelegramID, &g.Title, &g.GreetingEnabled, &g.GreetingMessage, &g.GoodbyeEnabled, &g.GoodbyeMessage, &g.CaptchaEnabled,
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
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("group:%d", telegramID)).Build())
	}
	return err
}

func (s *Store) SetGreetingStatus(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET greeting_enabled = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("group:%d", telegramID)).Build())
	}
	return err
}

func (s *Store) SetGreetingMessage(telegramID int64, message string) error {
	q := `UPDATE groups SET greeting_message = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, message, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("group:%d", telegramID)).Build())
	}
	return err
}

func (s *Store) SetGoodbyeStatus(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET goodbye_enabled = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("group:%d", telegramID)).Build())
	}
	return err
}

func (s *Store) SetGoodbyeMessage(telegramID int64, message string) error {
	q := `UPDATE groups SET goodbye_message = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, message, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("group:%d", telegramID)).Build())
	}
	return err
}

func (s *Store) UpdateGroupCaptcha(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET captcha_enabled = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("group:%d", telegramID)).Build())
	}
	return err
}
