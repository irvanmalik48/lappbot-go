package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"lappbot/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valkey-io/valkey-go"
)

type Store struct {
	db     *pgxpool.Pool
	Valkey valkey.Client
}

func New(cfg *config.Config) (*Store, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	config.MaxConns = 50
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Database connection established")

	vk, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%d", cfg.ValkeyHost, cfg.ValkeyPort)},
		Password:    cfg.ValkeyPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create valkey client: %w", err)
	}

	return &Store{db: pool, Valkey: vk}, nil
}

func (s *Store) Close() {
	s.db.Close()
	s.Valkey.Close()
}

func (s *Store) GetPool() *pgxpool.Pool {
	return s.db
}

func (s *Store) Ping() (map[string]time.Duration, error) {
	res := make(map[string]time.Duration)

	start := time.Now()
	if err := s.db.Ping(context.Background()); err != nil {
		return nil, err
	}
	res["database"] = time.Since(start)

	start = time.Now()
	if err := s.Valkey.Do(context.Background(), s.Valkey.B().Ping().Build()).Error(); err != nil {
		return nil, err
	}
	res["valkey"] = time.Since(start)

	return res, nil
}
func (s *Store) SetConnection(adminID int64, chatID int64) error {
	key := fmt.Sprintf("conn:%d", adminID)
	return s.Valkey.Do(context.Background(), s.Valkey.B().Set().Key(key).Value(fmt.Sprintf("%d", chatID)).Ex(30*time.Minute).Build()).Error()
}

func (s *Store) GetConnection(adminID int64) (int64, error) {
	key := fmt.Sprintf("conn:%d", adminID)
	val, err := s.Valkey.Do(context.Background(), s.Valkey.B().Get().Key(key).Build()).ToInt64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (s *Store) Disconnect(adminID int64) error {
	key := fmt.Sprintf("conn:%d", adminID)
	return s.Valkey.Do(context.Background(), s.Valkey.B().Del().Key(key).Build()).Error()
}

type ConnectionHistoryItem struct {
	ChatID    int64  `json:"chat_id"`
	ChatTitle string `json:"chat_title"`
}

func (s *Store) AddConnectionHistory(adminID int64, chatID int64, chatTitle string) error {
	key := fmt.Sprintf("conn_hist:%d", adminID)
	item := ConnectionHistoryItem{ChatID: chatID, ChatTitle: chatTitle}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	s.Valkey.Do(context.Background(), s.Valkey.B().Lrem().Key(key).Count(0).Element(string(data)).Build())
	s.Valkey.Do(context.Background(), s.Valkey.B().Lpush().Key(key).Element(string(data)).Build())
	return s.Valkey.Do(context.Background(), s.Valkey.B().Ltrim().Key(key).Start(0).Stop(9).Build()).Error()
}

func (s *Store) GetConnectionHistory(adminID int64) ([]ConnectionHistoryItem, error) {
	key := fmt.Sprintf("conn_hist:%d", adminID)
	vals, err := s.Valkey.Do(context.Background(), s.Valkey.B().Lrange().Key(key).Start(0).Stop(-1).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	var history []ConnectionHistoryItem
	for _, v := range vals {
		var item ConnectionHistoryItem
		if err := json.Unmarshal([]byte(v), &item); err == nil {
			history = append(history, item)
		}
	}
	return history, nil
}
