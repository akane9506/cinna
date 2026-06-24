package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func CheckSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
				AND table_name = 'allowed_users'
		)
		AND EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
				AND table_name = 'shopping_list'
		)
		AND EXISTS (
			SELECT 1
			FROM pg_type
			WHERE typname = 'shopping_category'
		);
	`
	var ok bool
	if err := pool.QueryRow(ctx, query).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errors.New("database schema incomplete")
	}
	return nil
}
