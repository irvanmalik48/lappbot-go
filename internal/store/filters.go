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

type Filter struct {
	ID        string
	GroupID   int64
	Trigger   string
	Response  string
	CreatedAt interface{}
}

func (s *Store) AddFilter(groupID int64, trigger, response string) error {
	id, err := gonanoid.New()
	if err != nil {
		return err
	}
	q := `INSERT INTO filters (id, group_id, trigger, response) VALUES ($1, $2, $3, $4)
	      ON CONFLICT (group_id, trigger) DO UPDATE SET response = $4`
	_, err = s.db.Exec(context.Background(), q, id, groupID, trigger, response)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("filters:%d", groupID)).Build())
	}
	return err
}

func (s *Store) GetFilter(groupID int64, trigger string) (string, error) {
	q := `SELECT response FROM filters WHERE group_id = $1 AND trigger = $2`
	var response string
	err := s.db.QueryRow(context.Background(), q, groupID, trigger).Scan(&response)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return response, nil
}

func (s *Store) GetFilters(groupID int64) ([]Filter, error) {
	cacheKey := fmt.Sprintf("filters:%d", groupID)
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		var filters []Filter
		if err := json.Unmarshal(val, &filters); err == nil {
			return filters, nil
		}
	}

	q := `SELECT id, trigger, response FROM filters WHERE group_id = $1 ORDER BY trigger ASC`
	rows, err := s.db.Query(context.Background(), q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []Filter
	for rows.Next() {
		var f Filter
		if err := rows.Scan(&f.ID, &f.Trigger, &f.Response); err != nil {
			continue
		}
		filters = append(filters, f)
	}

	if data, err := json.Marshal(filters); err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Set().Key(cacheKey).Value(string(data)).Ex(10*time.Minute).Build())
	}

	return filters, nil
}

func (s *Store) DeleteFilter(groupID int64, trigger string) error {
	q := `DELETE FROM filters WHERE group_id = $1 AND trigger = $2`
	_, err := s.db.Exec(context.Background(), q, groupID, trigger)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("filters:%d", groupID)).Build())
	}
	return err
}
