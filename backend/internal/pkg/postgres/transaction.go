package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier — repozitoriylar ishlatadigan minimal SQL yuzasi.
//
// `*pgxpool.Pool` ham, `pgx.Tx` ham buni QONDIRADI. Shu sabab bitta
// repozitoriy implementatsiyasi ham oddiy (pool) yo'lda, ham tranzaksiya
// ichida ishlay oladi — kodni ikki nusxaga bo'lish shart emas.
//
// Busiz repozitoriylar `*pgxpool.Pool` ni qattiq ushlab turardi va
// [WithTx] ni ular bilan birga ishlatib bo'lmasdi: aynan shu sabab
// ko'p-yozuvli oqimlar tranzaksiya o'rniga KOMPENSATSIYA bilan
// qoplanardi (yozuv uzilsa, oldingisini qo'lda o'chirish).
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx runs fn inside a single PostgreSQL transaction. It commits on success
// and rolls back on any non-nil return value or panic.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx rollback: %v (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx commit: %w", err)
	}
	return nil
}
