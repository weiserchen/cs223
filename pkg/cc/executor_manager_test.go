package cc

import (
	"context"
	"testing"
	"time"
	"txchain/pkg/database"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestTxExecutorManager(t *testing.T) {
	pgc, err := database.NewContainerTablesTx(t, "17.1")
	defer func() {
		if pgc != nil {
			testcontainers.CleanupContainer(t, pgc.Container)
		}
	}()
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, pgc.Endpoint())
	require.NoError(t, err)

	concurrency := 100
	retryFunc := ExponentialBackoffRetry(100 * time.Millisecond)
	execMgr := NewTxExecutorManager(retryFunc)
	go execMgr.Run()

	var execCtxs []*TxExecutorContext
	// successful
	for range concurrency {
		go testSuccessExecFunc(t, conn, execMgr)
	}

	execCtxs = testAllExecutor(t, conn, concurrency, ExecStatusCompleted)
	testSumEqual(t, execCtxs, 6*concurrency)
	testDeleteAllExecutor(t, conn)

	// commit failure
	for range concurrency {
		go testCommitFailureExecFunc(t, conn)
	}

	_ = testAllExecutor(t, conn, concurrency, ExecStatusAborted)
	testDeleteAllExecutor(t, conn)

	// force complete
	for range concurrency {
		go testForceCompleteExecFunc(t, conn, execMgr)
	}

	execCtxs = testAllExecutor(t, conn, concurrency, ExecStatusCompleted)
	testSumEqual(t, execCtxs, 1*concurrency)
	testDeleteAllExecutor(t, conn)

	// flaky success
	for range concurrency {
		go testFlakySuccessfulExecFunc(t, conn, execMgr)
	}

	execCtxs = testAllExecutor(t, conn, concurrency, ExecStatusCompleted)
	testSumEqual(t, execCtxs, 6*concurrency)
	testDeleteAllExecutor(t, conn)

	// flaky force complete
	for range concurrency {
		go testFlakyForceCompleteExecFunc(t, conn, execMgr)
	}

	execCtxs = testAllExecutor(t, conn, concurrency, ExecStatusCompleted)
	testSumEqual(t, execCtxs, 1*concurrency)
	testDeleteAllExecutor(t, conn)
}

func testSuccessExecFunc(
	t *testing.T,
	conn *pgxpool.Pool,
	execMgr *TxExecutorManager,
) {
	execCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, execCtx)
	require.NoError(t, err)

	stages := defaultStages()
	stage1 := stages[execStage1]
	stage2 := stages[execStage2]
	stage3 := stages[execStage3]
	checkpointer := DefaultCheckpointer(conn)
	executor := NewTxExecutor(execCtx, checkpointer)
	executor.
		CommitStage(stage1).
		Stage(stage2).
		Stage(stage3)

	_, err = executor.Run()
	require.NoError(t, err)

	execMgr.Send(executor)
}

func testCommitFailureExecFunc(
	t *testing.T,
	conn *pgxpool.Pool,
) {
	execCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, execCtx)
	require.NoError(t, err)

	stages := defaultStages()
	failureStage := stages[execStageFailure]
	stage2 := stages[execStage2]
	stage3 := stages[execStage3]
	checkpointer := DefaultCheckpointer(conn)
	executor := NewTxExecutor(execCtx, checkpointer)
	executor.
		CommitStage(failureStage).
		Stage(stage2).
		Stage(stage3)

	_, err = executor.Run()
	require.Error(t, err)

	executor.execCtx.Status = ExecStatusAborted
	err = executor.Checkpoint()
	require.NoError(t, err)
}

func testForceCompleteExecFunc(
	t *testing.T,
	conn *pgxpool.Pool,
	execMgr *TxExecutorManager,
) {
	execCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, execCtx)
	require.NoError(t, err)

	stages := defaultStages()
	stage1 := stages[execStage1]
	unrecoverableFailureStage := stages[execStageUnrecoverableFailure]
	stage3 := stages[execStage3]
	checkpointer := DefaultCheckpointer(conn)
	executor := NewTxExecutor(execCtx, checkpointer)
	executor.
		CommitStage(stage1).
		Stage(unrecoverableFailureStage).
		Stage(stage3)

	_, err = executor.Run()
	require.NoError(t, err)

	execMgr.Send(executor)
}

func testFlakySuccessfulExecFunc(
	t *testing.T,
	conn *pgxpool.Pool,
	execMgr *TxExecutorManager,
) {
	execCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, execCtx)
	require.NoError(t, err)

	stages := defaultStages()
	stage1 := stages[execStage1]
	flakyStage2 := stages[execStage2Flaky]
	flakyStage3 := stages[execStage3Flaky]
	checkpointer := DefaultCheckpointer(conn)
	executor := NewTxExecutor(execCtx, checkpointer)
	executor.
		CommitStage(stage1).
		Stage(flakyStage2).
		Stage(flakyStage3)

	_, err = executor.Run()
	require.NoError(t, err)

	execMgr.Send(executor)
}

func testFlakyForceCompleteExecFunc(
	t *testing.T,
	conn *pgxpool.Pool,
	execMgr *TxExecutorManager,
) {
	execCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, execCtx)
	require.NoError(t, err)

	stages := defaultStages()
	stage1 := stages[execStage1]
	flakyStage3 := stages[execStage3Flaky]
	flakyFailureStage := stages[execStageFailureFlaky]
	checkpointer := DefaultCheckpointer(conn)
	executor := NewTxExecutor(execCtx, checkpointer)
	executor.
		CommitStage(stage1).
		Stage(flakyFailureStage).
		Stage(flakyStage3)

	_, err = executor.Run()
	require.NoError(t, err)

	execMgr.Send(executor)
}

func testSumEqual(t *testing.T, execCtxs []*TxExecutorContext, expectedSum int) {
	t.Helper()

	sum := 0
	for _, execCtx := range execCtxs {
		input, err := UnmarshalInput[Input](execCtx.Input)
		require.NoError(t, err)
		for _, n := range input.Value {
			sum += n
		}
	}
	require.Equal(t, expectedSum, sum)
}

func testAllExecutor(
	t *testing.T,
	conn *pgxpool.Pool,
	count int,
	status ExecStatus,
) []*TxExecutorContext {
	t.Helper()

	var execCtxs []*TxExecutorContext
	var err error
	require.Eventually(t, func() bool {
		execCtxs, err = GetAllTxExecutorCheckpoint(conn, status)
		if err != nil {
			return false
		}
		if len(execCtxs) != count {
			return false
		}
		return true
	}, 10*time.Second, 500*time.Millisecond)

	return execCtxs
}

func testDeleteAllExecutor(t *testing.T, conn *pgxpool.Pool) {
	t.Helper()

	err := DeleteAllExecutorCheckpoints(conn)
	require.NoError(t, err)
}
