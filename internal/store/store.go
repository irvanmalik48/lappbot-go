package store

import (
	"context"
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
