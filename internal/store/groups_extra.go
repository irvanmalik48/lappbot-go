package store

import (
	"context"
)

func (s *Store) GetAllGroups() ([]Group, error) {
	q := `SELECT id, telegram_id, title FROM groups`
	rows, err := s.db.Query(context.Background(), q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.TelegramID, &g.Title); err != nil {
			continue
		}
		groups = append(groups, g)
	}
	return groups, nil
}
