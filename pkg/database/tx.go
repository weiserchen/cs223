package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func BeginTx(ctx context.Context, conn *pgxpool.Pool) (pgx.Tx, func(error) error, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}

	commitFunc := func(err error) error {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				err = fmt.Errorf("%w: %v", rollbackErr, err)
			}
		} else {
			err = tx.Commit(ctx)
		}
		return err
	}

	return tx, commitFunc, nil
}
