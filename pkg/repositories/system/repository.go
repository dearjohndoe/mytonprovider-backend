package system

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mytonprovider-backend/pkg/repositories"
)

type repository struct {
	db repositories.DBTX
}

type Repository interface {
	SetParam(ctx context.Context, key string, value string) (err error)
	GetParam(ctx context.Context, key string) (value string, err error)

	// WithTx возвращает Repository, привязанный к транзакции tx.
	WithTx(tx pgx.Tx) Repository

	// MarkProcessedTx помечает job_id как обработанный в рамках tx.
	// Возвращает inserted=true при первой обработке; false — если job_id
	// уже присутствует (дубль), в этом случае tx остаётся открытой и
	// caller должен сделать rollback или commit без писаний.
	MarkProcessedTx(
		ctx context.Context,
		tx pgx.Tx,
		jobID, jobType, agentID string,
	) (inserted bool, err error)
}

func (r *repository) WithTx(tx pgx.Tx) Repository {
	return &repository{db: tx}
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
		if errors.Is(err, pgx.ErrNoRows) {
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

func (r *repository) MarkProcessedTx(
	ctx context.Context,
	tx pgx.Tx,
	jobID, jobType, agentID string,
) (inserted bool, err error) {
	const op = "system.MarkProcessedTx"

	query := `
		INSERT INTO system.processed_jobs (job_id, type, agent_id)
		VALUES ($1, $2, NULLIF($3, ''))
		ON CONFLICT (job_id) DO NOTHING
	`

	tag, err := tx.Exec(ctx, query, jobID, jobType, agentID)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return tag.RowsAffected() == 1, nil
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db: db,
	}
}

var _ repositories.DBTX = (*pgxpool.Pool)(nil)
