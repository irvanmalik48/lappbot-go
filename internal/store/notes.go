package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	"github.com/jackc/pgx/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Note struct {
	ID        string
	ChatID    int64
	Name      string
	Content   string
	Type      string
	FileID    string
	CreatedBy int64
	CreatedAt any
}

func (s *Store) SaveNote(chatID int64, name, content, noteType, fileID string, createdBy int64) error {
	id, err := gonanoid.New()
	if err != nil {
		return err
	}
	q := `INSERT INTO notes (id, chat_id, name, content, type, file_id, created_by) 
          VALUES ($1, $2, $3, $4, $5, $6, $7)
          ON CONFLICT (chat_id, name) DO UPDATE SET content = $4, type = $5, file_id = $6, created_by = $7`
	_, err = s.db.Exec(context.Background(), q, id, chatID, name, content, noteType, fileID, createdBy)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("notes:"+strconv.FormatInt(chatID, 10)).Build())
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("note:"+strconv.FormatInt(chatID, 10)+":"+name).Build())
	}
	return err
}

func (s *Store) GetNote(chatID int64, name string) (*Note, error) {
	cacheKey := "note:" + strconv.FormatInt(chatID, 10) + ":" + name
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		var n Note
		if err := json.Unmarshal(val, &n); err == nil {
			return &n, nil
		}
	}

	q := `SELECT id, chat_id, name, content, type, file_id, created_by FROM notes WHERE chat_id = $1 AND name = $2`
	var n Note
	err = s.db.QueryRow(context.Background(), q, chatID, name).Scan(
		&n.ID, &n.ChatID, &n.Name, &n.Content, &n.Type, &n.FileID, &n.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if data, err := json.Marshal(n); err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Set().Key(cacheKey).Value(string(data)).Ex(1*time.Hour).Build())
	}
	return &n, nil
}

func (s *Store) DeleteNote(chatID int64, name string) error {
	q := `DELETE FROM notes WHERE chat_id = $1 AND name = $2`
	_, err := s.db.Exec(context.Background(), q, chatID, name)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("notes:"+strconv.FormatInt(chatID, 10)).Build())
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key("note:"+strconv.FormatInt(chatID, 10)+":"+name).Build())
	}
	return err
}

func (s *Store) ClearAllNotes(chatID int64) error {
	q := `DELETE FROM notes WHERE chat_id = $1`
	_, err := s.db.Exec(context.Background(), q, chatID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("notes:%d", chatID)).Build())
	}
	return err
}

func (s *Store) GetNotes(chatID int64) ([]Note, error) {
	cacheKey := "notes:" + strconv.FormatInt(chatID, 10)
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		var notes []Note
		if err := json.Unmarshal(val, &notes); err == nil {
			return notes, nil
		}
	}

	q := `SELECT name, type FROM notes WHERE chat_id = $1 ORDER BY name ASC`
	rows, err := s.db.Query(context.Background(), q, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]Note, 0)
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.Name, &n.Type); err == nil {
			notes = append(notes, n)
		}
	}

	if data, err := json.Marshal(notes); err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Set().Key(cacheKey).Value(string(data)).Ex(10*time.Minute).Build())
	}
	return notes, nil
}
