package store

import (
	"context"
	"errors"

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
	q := `SELECT id, telegram_id, title, greeting_enabled, greeting_message, goodbye_enabled, goodbye_message, captcha_enabled 
          FROM groups WHERE telegram_id = $1`

	var g Group
	err := s.db.QueryRow(context.Background(), q, telegramID).Scan(
		&g.ID, &g.TelegramID, &g.Title, &g.GreetingEnabled, &g.GreetingMessage, &g.GoodbyeEnabled, &g.GoodbyeMessage, &g.CaptchaEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
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
	return err
}

func (s *Store) UpdateGroupGreeting(telegramID int64, enabled bool, message string) error {
	q := `UPDATE groups SET greeting_enabled = $1, greeting_message = $2 WHERE telegram_id = $3`
	_, err := s.db.Exec(context.Background(), q, enabled, message, telegramID)
	return err
}

func (s *Store) UpdateGroupGoodbye(telegramID int64, enabled bool, message string) error {
	q := `UPDATE groups SET goodbye_enabled = $1, goodbye_message = $2 WHERE telegram_id = $3`
	_, err := s.db.Exec(context.Background(), q, enabled, message, telegramID)
	return err
}

func (s *Store) UpdateGroupCaptcha(telegramID int64, enabled bool) error {
	q := `UPDATE groups SET captcha_enabled = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, enabled, telegramID)
	return err
}
