package store

import (
	"context"
	"strconv"

	"github.com/goccy/go-json"
)

func (s *Store) SetLogChannel(telegramID int64, channelID int64) error {
	q := `UPDATE groups SET log_channel_id = $1 WHERE telegram_id = $2`
	_, err := s.db.Exec(context.Background(), q, channelID, telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}

func (s *Store) SetLogCategories(telegramID int64, categories []string) error {
	data, err := json.Marshal(categories)
	if err != nil {
		return err
	}
	q := `UPDATE groups SET log_categories = $1 WHERE telegram_id = $2`
	_, err = s.db.Exec(context.Background(), q, string(data), telegramID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("group:"+strconv.FormatInt(telegramID, 10)).Build())
	}
	return err
}
