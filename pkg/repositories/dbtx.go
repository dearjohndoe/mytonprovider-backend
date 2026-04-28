// Package repositories содержит общий DBTX-интерфейс, который реализуется
// и *pgxpool.Pool, и pgx.Tx — это позволяет одному и тому же набору
// репо-методов работать как в pool-режиме (read-only / standalone writes),
// так и внутри явной транзакции (в result-handler'ах).
package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
