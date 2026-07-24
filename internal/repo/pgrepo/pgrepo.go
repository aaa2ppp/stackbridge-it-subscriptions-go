package pgrepo

import (
	"context"
	"fmt"
	"net"
	"subscriptions/internal/repo"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PGRepo struct {
	db *pgxpool.Pool
}

func Open(ctx context.Context, cfg repo.Config) (*PGRepo, error) {
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, cfg.DBName, cfg.User, cfg.Password, cfg.SSLMode)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &PGRepo{db: pool}, nil
}

func (repo *PGRepo) Close() {
	repo.db.Close()
}
