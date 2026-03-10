package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zalhui/URLShortener/internal/config/db"
)

type DB struct {
	Pool *pgxpool.Pool
}

const (
	defaultTimeout = 5 * time.Second
)

func NewDB(ctx context.Context, cfg *db.DBConfig) (*DB, error) {
	pool, err := pgxpool.New(ctx, cfg.GetDBConnString())
	if err != nil {
		return nil, fmt.Errorf("NewDB failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("NewDB failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return db.Pool.Ping(ctx)
}
