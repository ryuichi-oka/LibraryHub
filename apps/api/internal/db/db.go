package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	// 業務時刻の基準を Asia/Tokyo に固定する。
	cfg.ConnConfig.RuntimeParams["timezone"] = "Asia/Tokyo"
	return pgxpool.NewWithConfig(ctx, cfg)
}
