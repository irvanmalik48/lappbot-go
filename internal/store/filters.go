package store

import (
	"context"
	"errors"

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
	return filters, nil
}

func (s *Store) DeleteFilter(groupID int64, trigger string) error {
	q := `DELETE FROM filters WHERE group_id = $1 AND trigger = $2`
	_, err := s.db.Exec(context.Background(), q, groupID, trigger)
	return err
}
