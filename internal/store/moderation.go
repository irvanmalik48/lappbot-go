package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func (s *Store) AddWarn(userID, groupID int64, reason string, createdBy int64) (int, error) {
	id, err := gonanoid.New()
	if err != nil {
		return 0, err
	}
	qInsert := `INSERT INTO warns (id, user_id, group_id, reason, created_by) 
			VALUES ($1, $2, $3, $4, $5)`

	_, err = s.db.Exec(context.Background(), qInsert, id, userID, groupID, reason, createdBy)
	if err != nil {
		return 0, err
	}

	qCount := `SELECT COUNT(*) FROM warns WHERE user_id = $1 AND group_id = $2`

	var count int
	err = s.db.QueryRow(context.Background(), qCount, userID, groupID).Scan(&count)
	return count, err
}

func (s *Store) GetWarnCount(userID, groupID int64) (int, error) {
	q := `SELECT COUNT(*) FROM warns WHERE user_id = $1 AND group_id = $2`
	var count int
	err := s.db.QueryRow(context.Background(), q, userID, groupID).Scan(&count)
	return count, err
}

func (s *Store) ResetWarns(userID, groupID int64) error {
	q := `DELETE FROM warns WHERE user_id = $1 AND group_id = $2`
	_, err := s.db.Exec(context.Background(), q, userID, groupID)
	return err
}

func (s *Store) ResetAllWarns(groupID int64) error {
	q := `DELETE FROM warns WHERE group_id = $1`
	_, err := s.db.Exec(context.Background(), q, groupID)
	return err
}

func (s *Store) RemoveLastWarn(userID, groupID int64) error {
	q := `DELETE FROM warns WHERE id IN (
		SELECT id FROM warns WHERE user_id = $1 AND group_id = $2 
		ORDER BY created_at DESC LIMIT 1
	)`
	_, err := s.db.Exec(context.Background(), q, userID, groupID)
	return err
}

func (s *Store) GetActiveWarns(userID, groupID int64, since time.Time) (int, error) {
	q := `SELECT COUNT(*) FROM warns WHERE user_id = $1 AND group_id = $2 AND created_at >= $3`
	var count int
	err := s.db.QueryRow(context.Background(), q, userID, groupID, since).Scan(&count)
	return count, err
}

func (s *Store) BanUser(userID, groupID int64, until time.Time, reason string, createdBy int64, banType string) error {
	id, err := gonanoid.New()
	if err != nil {
		return err
	}

	var untilPtr *time.Time
	if !until.IsZero() {
		untilPtr = &until
	}

	q := `INSERT INTO bans (id, user_id, group_id, until_date, type, reason, created_by) 
          VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = s.db.Exec(context.Background(), q, id, userID, groupID, untilPtr, banType, reason, createdBy)
	return err
}

type BlacklistItem struct {
	ID             string    `db:"id" json:"id"`
	GroupID        int64     `db:"group_id" json:"group_id"`
	Type           string    `db:"type" json:"type"`
	Value          string    `db:"value" json:"value"`
	Action         string    `db:"action" json:"action"`
	ActionDuration string    `db:"action_duration" json:"action_duration"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

func (s *Store) AddBlacklistItem(groupID int64, kind, value, action, duration string) error {
	id, err := gonanoid.New()
	if err != nil {
		return err
	}
	q := `INSERT INTO blacklists (id, group_id, type, value, action, action_duration) 
          VALUES ($1, $2, $3, $4, $5, $6)
          ON CONFLICT (group_id, type, value) DO UPDATE 
          SET action = EXCLUDED.action, action_duration = EXCLUDED.action_duration`
	_, err = s.db.Exec(context.Background(), q, id, groupID, kind, value, action, duration)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("blacklist:%d", groupID)).Build())
	}
	return err
}

func (s *Store) RemoveBlacklistItem(groupID int64, kind, value string) error {
	q := `DELETE FROM blacklists WHERE group_id = $1 AND type = $2 AND value = $3`
	_, err := s.db.Exec(context.Background(), q, groupID, kind, value)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("blacklist:%d", groupID)).Build())
	}
	return err
}

func (s *Store) GetBlacklist(groupID int64) ([]BlacklistItem, error) {
	cacheKey := fmt.Sprintf("blacklist:%d", groupID)
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(cacheKey).Build()).AsBytes()
	if err == nil {
		var items []BlacklistItem
		if err := json.Unmarshal(val, &items); err == nil {
			return items, nil
		}
	}

	q := `SELECT id, group_id, type, value, action, COALESCE(action_duration, '') as action_duration, created_at FROM blacklists WHERE group_id = $1`
	rows, err := s.db.Query(context.Background(), q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BlacklistItem
	for rows.Next() {
		var i BlacklistItem
		if err := rows.Scan(&i.ID, &i.GroupID, &i.Type, &i.Value, &i.Action, &i.ActionDuration, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}

	if data, err := json.Marshal(items); err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Set().Key(cacheKey).Value(string(data)).Ex(10*time.Minute).Build())
	}
	return items, nil
}

func (s *Store) AddApprovedUser(userID, groupID, createdBy int64) error {
	q := `INSERT INTO approved_users (user_id, group_id, created_by) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	_, err := s.db.Exec(context.Background(), q, userID, groupID, createdBy)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("approved:%d:%d", groupID, userID)).Build())
	}
	return err
}

func (s *Store) RemoveApprovedUser(userID, groupID int64) error {
	q := `DELETE FROM approved_users WHERE user_id = $1 AND group_id = $2`
	_, err := s.db.Exec(context.Background(), q, userID, groupID)
	if err == nil {
		s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(fmt.Sprintf("approved:%d:%d", groupID, userID)).Build())
	}
	return err
}

func (s *Store) IsApprovedUser(userID, groupID int64) (bool, error) {
	cacheKey := "approved:" + strconv.FormatInt(groupID, 10) + ":" + strconv.FormatInt(userID, 10)
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(cacheKey).Build()).ToString()
	if err == nil {
		return val == "1", nil
	}

	q := `SELECT EXISTS(SELECT 1 FROM approved_users WHERE user_id = $1 AND group_id = $2)`
	var exists bool
	err = s.db.QueryRow(context.Background(), q, userID, groupID).Scan(&exists)
	if err == nil {
		v := "0"
		if exists {
			v = "1"
		}
		s.Valkey.Do(context.Background(), s.Valkey.B().Set().Key(cacheKey).Value(v).Ex(10*time.Minute).Build())
	}
	return exists, err
}

func (s *Store) GetApprovedUsers(groupID int64) ([]int64, error) {
	q := `SELECT user_id FROM approved_users WHERE group_id = $1`
	rows, err := s.db.Query(context.Background(), q, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []int64
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		users = append(users, uid)
	}
	return users, nil
}
