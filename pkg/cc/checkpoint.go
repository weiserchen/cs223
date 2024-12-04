package cc

import (
	"context"
	"encoding/json"
	"errors"
	"txchain/pkg/database"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTxDryRun = errors.New("dry run a tx")
)

type CheckpointFunc = func(execCtx *TxExecutorContext) error
type CheckpointRetriverFunc = func(execID uint64) (ExecStatus, *TxExecutorContext, error)

func DefaultCheckpointer(conn *pgxpool.Pool) CheckpointFunc {
	return func(execCtx *TxExecutorContext) error {
		return UpdateCheckpointExecutorContext(conn, execCtx)
	}
}

func DefaultCheckpointRetriever(conn *pgxpool.Pool) CheckpointRetriverFunc {
	return func(execID uint64) (ExecStatus, *TxExecutorContext, error) {
		return GetTxExecutorCheckpoint(conn, execID)
	}
}

func GetTxExecutorCheckpoint(conn *pgxpool.Pool, execID uint64) (ExecStatus, *TxExecutorContext, error) {
	query := `
		SELECT status, checkpoint
		FROM TxExecutor
		WHERE exec_id = $1;
	`

	var status ExecStatus
	var b []byte
	var execCtx TxExecutorContext
	ctx := context.Background()
	row := conn.QueryRow(ctx, query, execID)
	if err := row.Scan(&status, &b); err != nil {
		return status, nil, err
	}
	if err := json.Unmarshal(b, &execCtx); err != nil {
		return status, nil, err
	}

	return status, &execCtx, nil
}

func GetAllTxExecutorCheckpoint(conn *pgxpool.Pool, status ExecStatus) ([]*TxExecutorContext, error) {
	query := `
		SELECT checkpoint
		FROM TxExecutor
		WHERE status = $1;
	`

	var result []*TxExecutorContext
	ctx := context.Background()
	rows, err := conn.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var b []byte
		var execCtx TxExecutorContext

		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &execCtx); err != nil {
			return nil, err
		}

		result = append(result, &execCtx)
	}

	return result, nil
}

func InsertCheckpointExecutorContext(conn *pgxpool.Pool, execCtx *TxExecutorContext) error {
	b, err := json.Marshal(execCtx)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO TxExecutor (status, checkpoint)
		VALUES ($1, $2)
		RETURNING exec_id;
	`

	ctx := context.Background()
	row := conn.QueryRow(ctx, query, execCtx.Status, b)
	err = row.Scan(&execCtx.ExecID)
	return err
}

func UpdateCheckpointExecutorContext(conn *pgxpool.Pool, execCtx *TxExecutorContext) error {
	b, err := json.Marshal(execCtx)
	if err != nil {
		return err
	}

	query := `
		UPDATE TxExecutor 
		SET
			status = $2, 
			checkpoint = $3
		WHERE exec_id = $1;
	`

	ctx := context.Background()
	execID := execCtx.ExecID
	_, err = conn.Exec(ctx, query, execID, execCtx.Status, b)
	return err
}

func DeleteAllExecutorCheckpoints(conn *pgxpool.Pool) error {
	query := `
		TRUNCATE TABLE TxExecutor;
	`

	ctx := context.Background()
	_, err := conn.Exec(ctx, query)
	return err
}

var _ database.TxHookFunc = TxDedupBeforeHook
var _ database.TxHookFunc = TxDedupAfterHook

func TxDedupBeforeHook(ctx context.Context, tx pgx.Tx) error {
	stageCtx, ok := GetTxStageCtx(ctx)
	// tx not enabled
	if !ok {
		return nil
	}

	var b []byte
	var err error

	query := `
		SELECT content
		FROM TxResult
		WHERE prt = $1 AND svc = $2 AND ts = $3;
	`

	row := tx.QueryRow(ctx, query, stageCtx.Partition, stageCtx.Service, stageCtx.Timestamp)
	err = row.Scan(&b)
	// no previous tx result
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		if stageCtx.DryRun {
			return ErrTxDryRun
		}
		// result not exists -> resume tx
		return nil
	}

	var txResult any
	if err = json.Unmarshal(b, &txResult); err != nil {
		return err
	}

	traceCtx, ok := format.GetTraceContext(ctx)
	if !ok {
		return format.ErrNoTraceCtx
	}

	// tx had already executed -> return previous result
	database.SetResult(traceCtx, txResult)
	return database.ErrTxAlreadyExecuted
}

func TxDedupAfterHook(ctx context.Context, tx pgx.Tx) error {
	stageCtx, ok := GetTxStageCtx(ctx)
	// tx not enabled
	if !ok {
		return nil
	}

	traceCtx, ok := format.GetTraceContext(ctx)
	if !ok {
		return format.ErrNoTraceCtx
	}

	txResult, ok := database.GetResult(traceCtx)
	if !ok {
		return database.ErrNoTxResult
	}

	b, err := json.Marshal(txResult)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO TxResult (prt, svc, ts, content)
		VALUES ($1, $2, $3, $4);
	`

	_, err = tx.Exec(ctx, query, stageCtx.Partition, stageCtx.Service, stageCtx.Timestamp, b)
	return err
}
