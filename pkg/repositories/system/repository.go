package system

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	db *pgxpool.Pool
}

type Repository interface {
	SetParam(ctx context.Context, key string, value string) (err error)
	GetParam(ctx context.Context, key string) (value string, err error)
}

func (r *repository) SetParam(ctx context.Context, key string, value string) (err error) {
	query := `
		INSERT INTO system.params (key, value)
		VALUES (
			$1,
			$2
		)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
			updated_at = now();
	`

	_, err = r.db.Exec(ctx, query, key, value)

	return
}

func (r *repository) GetParam(ctx context.Context, key string) (value string, err error) {
	query := `
		SELECT value
		FROM system.params
		WHERE key = $1
		LIMIT 1;
	`

	rows, err := r.db.Query(ctx, query, key)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = nil
			return
		}

		return
	}
	defer rows.Close()

	if rows.Next() {
		if rErr := rows.Scan(&value); rErr != nil {
			err = rErr
			return
		}
	}

	err = rows.Err()

	return
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db: db,
	}
}
