package cc

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"testing"
	"txchain/pkg/database"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestDBTxExecutor(t *testing.T) {
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

	// success stages
	testSuccessfulExecutorStage(t, conn)

	// commit failure
	testCommitFailureExecutorStage(t, conn)

	// rollback stage
	testRollbackExecutorStage(t, conn)

	// force complete stage
	testForceCompleteExecutorStage(t, conn)
}

func testSuccessfulExecutorStage(
	t *testing.T,
	conn *pgxpool.Pool,
) {
	successExecCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, successExecCtx)
	require.NoError(t, err)

	stages := defaultStages()
	stage1 := stages[execStage1]
	stage2 := stages[execStage2]
	stage3 := stages[execStage3]
	checkpointer := DefaultCheckpointer(conn)
	successExecutor := NewTxExecutor(successExecCtx, checkpointer)
	successExecutor.
		CommitStage(stage1).
		Stage(stage2).
		Stage(stage3)

	v, err := successExecutor.Run()
	require.NoError(t, err)
	require.Equal(t, ExecStatusCommitted, successExecutor.execCtx.Status)

	r, ok := v.(Result)
	require.True(t, ok)
	require.EqualValues(t, Result{1}, r)

	expectedInput := Input{
		Value: []int{1},
	}
	require.EqualValues(t, expectedInput, successExecCtx.Input)

	err = successExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, successExecutor.execCtx, ExecStatusCommitted)

	executeCount := 0
	for successExecutor.Next() {
		executeCount++
		err = successExecutor.Execute()
		require.NoError(t, err)

		err = successExecutor.Checkpoint()
		require.NoError(t, err)
		testCheckpoint(t, conn, successExecutor.execCtx, ExecStatusCommitted)
	}

	require.Equal(t, 2, executeCount)
	expectedInput = Input{
		Value: []int{1, 2, 3},
	}
	require.EqualValues(t, expectedInput, successExecCtx.Input)

	successExecutor.execCtx.Status = ExecStatusCompleted
	err = successExecutor.Checkpoint()
	require.NoError(t, err)

	testCheckpoint(t, conn, successExecutor.execCtx, ExecStatusCompleted)
}

func testCommitFailureExecutorStage(
	t *testing.T,
	conn *pgxpool.Pool,
) {
	commitFailureExecCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, commitFailureExecCtx)
	require.NoError(t, err)

	stages := defaultStages()
	failureStage := stages[execStageFailure]
	stage2 := stages[execStage2]
	stage3 := stages[execStage3]
	checkpointer := DefaultCheckpointer(conn)
	commitFailureExecutor := NewTxExecutor(commitFailureExecCtx, checkpointer)
	commitFailureExecutor.
		CommitStage(failureStage).
		Stage(stage2).
		Stage(stage3)

	_, err = commitFailureExecutor.Run()
	require.Error(t, err)
	require.True(t, errors.Is(ErrTxExecAborted, errors.Unwrap(err)), err)
	require.Equal(t, ExecStatusAborted, commitFailureExecutor.execCtx.Status)

	err = commitFailureExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, commitFailureExecutor.execCtx, ExecStatusAborted)
}

func testRollbackExecutorStage(
	t *testing.T,
	conn *pgxpool.Pool,
) {
	rollbackExecCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, rollbackExecCtx)
	require.NoError(t, err)

	stages := defaultStages()
	stage1 := stages[execStage1]
	stage2 := stages[execStage2]
	failureStage := stages[execStageFailure]
	checkpointer := DefaultCheckpointer(conn)
	rollbackExecutor := NewTxExecutor(rollbackExecCtx, checkpointer)
	rollbackExecutor.
		CommitStage(stage1).
		Stage(stage2).
		Stage(failureStage)

	v, err := rollbackExecutor.Run()
	require.NoError(t, err)
	require.Equal(t, ExecStatusCommitted, rollbackExecutor.execCtx.Status)

	r, ok := v.(Result)
	require.True(t, ok)
	require.EqualValues(t, Result{1}, r)

	expectedInput := Input{
		Value: []int{1},
	}
	require.EqualValues(t, expectedInput, rollbackExecCtx.Input)

	err = rollbackExecutor.Execute()
	require.NoError(t, err)

	err = rollbackExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, rollbackExecutor.execCtx, ExecStatusCommitted)

	err = rollbackExecutor.Execute()
	require.Error(t, err)
	rollbackExecutor.execCtx.Status = ExecStatusRollback

	expectedInput = Input{
		Value: []int{1, 2},
	}
	require.EqualValues(t, expectedInput, rollbackExecutor.execCtx.Input)

	err = rollbackExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, rollbackExecutor.execCtx, ExecStatusRollback)

	rollbackCount := 0
	for rollbackExecutor.Next() {
		rollbackCount++
		err = rollbackExecutor.Rollback()
		require.NoError(t, err)
		err = rollbackExecutor.Checkpoint()
		require.NoError(t, err)
	}

	require.Equal(t, 1, rollbackCount)
	expectedInput = Input{
		Value: []int{1},
	}
	require.EqualValues(t, expectedInput, rollbackExecutor.execCtx.Input)
	testCheckpoint(t, conn, rollbackExecutor.execCtx, ExecStatusRollback)

	rollbackExecutor.execCtx.Status = ExecStatusCompleted
	err = rollbackExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, rollbackExecutor.execCtx, ExecStatusCompleted)
}

func testForceCompleteExecutorStage(
	t *testing.T,
	conn *pgxpool.Pool,
) {
	completeExecCtx := defaultExecCtx()

	err := InsertCheckpointExecutorContext(conn, completeExecCtx)
	require.NoError(t, err)

	stages := defaultStages()
	stage1 := stages[execStage1]
	failureStage := stages[execStageFailure]
	checkpointer := DefaultCheckpointer(conn)
	completeExecutor := NewTxExecutor(completeExecCtx, checkpointer)
	completeExecutor.
		CommitStage(stage1).
		Stage(failureStage).
		Stage(failureStage)

	v, err := completeExecutor.Run()
	require.NoError(t, err)
	require.Equal(t, ExecStatusCommitted, completeExecutor.execCtx.Status)

	r, ok := v.(Result)
	require.True(t, ok)
	require.EqualValues(t, Result{1}, r)

	expectedInput := Input{
		Value: []int{1},
	}
	require.EqualValues(t, expectedInput, completeExecutor.execCtx.Input)

	err = completeExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, completeExecutor.execCtx, ExecStatusCommitted)

	err = completeExecutor.Execute()
	require.Error(t, err)
	completeExecutor.execCtx.Status = ExecStatusForceComplete

	err = completeExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, completeExecutor.execCtx, ExecStatusForceComplete)

	completeCount := 0
	for completeExecutor.Next() {
		completeCount++
		err = completeExecutor.ForceComplete()
		require.NoError(t, err)
		err = completeExecutor.Checkpoint()
		require.NoError(t, err)
	}

	require.Equal(t, 2, completeCount)
	expectedInput = Input{
		Value: []int{1, 0, 0},
	}
	require.EqualValues(t, expectedInput, completeExecutor.execCtx.Input)
	testCheckpoint(t, conn, completeExecutor.execCtx, ExecStatusForceComplete)

	completeExecutor.execCtx.Status = ExecStatusCompleted
	err = completeExecutor.Checkpoint()
	require.NoError(t, err)
	testCheckpoint(t, conn, completeExecutor.execCtx, ExecStatusCompleted)
}

func testCheckpoint(
	t *testing.T,
	conn *pgxpool.Pool,
	execCtx *TxExecutorContext,
	expectedStatus ExecStatus,
) {
	checkpointRetriever := DefaultCheckpointRetriever(conn)
	ckptStatus, ckptExecCtx, err := checkpointRetriever(execCtx.ExecID)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, ckptStatus)
	if expectedStatus != ExecStatusAborted {
		checkTxExecCtx[Input, Result](t, execCtx, ckptExecCtx)
	}
}

type Input struct {
	Value []int
}
type Result struct {
	Value int
}

func pushStageFunc(i int) StageFunc {
	return func(v any) (any, any, error) {
		s := v.(Input)
		s.Value = append(s.Value, i)
		r := Result{i}
		return r, s, nil
	}
}

func failureStageFunc(v any) (any, any, error) {
	return v, v, errors.New("some error")
}

func unrecoverableFailureStageFunc(v any) (any, any, error) {
	return v, v, ErrTxExecUnrecoverable
}

func popRollbackFunc(v any) (any, error) {
	s := v.(Input)
	s.Value = s.Value[:len(s.Value)-1]
	return s, nil
}

func forceCompleteFunc(v any) (any, error) {
	s := v.(Input)
	s.Value = append(s.Value, 0)
	return s, nil
}

func flakyStageFunc(i int) StageFunc {
	return func(v any) (any, any, error) {
		n := rand.Int63n(10000)
		if n < 6000 {
			return nil, nil, errors.New("flaky error")
		}
		s := v.(Input)
		s.Value = append(s.Value, i)
		r := Result{i}
		return r, s, nil
	}
}

func flakyCompleteFunc(v any) (any, error) {
	n := rand.Int63n(10000)
	if n < 6000 {
		return nil, errors.New("flaky error")
	}
	s := v.(Input)
	s.Value = append(s.Value, 0)
	return s, nil
}

func recoveryPushStageFunc(i int) StageFunc {
	return func(v any) (any, any, error) {
		var s Input
		var ok bool
		var err error
		s, ok = v.(Input)
		if !ok {
			s, err = format.UnmarshalInput[Input](v)
			if err != nil {
				return nil, nil, err
			}
		}
		s.Value = append(s.Value, i)
		r := Result{i}
		return r, s, nil
	}
}

func recoveryForceCompleteFunc(v any) (any, error) {
	var s Input
	var ok bool
	var err error
	s, ok = v.(Input)
	if !ok {
		s, err = format.UnmarshalInput[Input](v)
		if err != nil {
			return nil, err
		}
	}
	s.Value = append(s.Value, 0)
	return s, nil
}

func defaultExecCtx() *TxExecutorContext {
	ctrlCtx := &TxControlContext{
		Partition: 3,
		Service:   "service-a",
		Attrs:     []string{"apple", "banana"},
	}

	return &TxExecutorContext{
		CtrlCtx:  ctrlCtx,
		Status:   ExecStatusPending,
		Input:    Input{},
		Curr:     0,
		Method:   http.MethodPost,
		Endpoint: "127.0.0.1:8080",
	}
}

type execStage int

const (
	execStage1 execStage = iota
	execStage2
	execStage3
	execStageFailure
	execStageUnrecoverableFailure
	execStage1Flaky
	execStage2Flaky
	execStage3Flaky
	execStageFailureFlaky
	execStage1Recovery
	execStage2Recovery
	execStage3Recovery
)

func defaultStages() map[execStage]*TxExecutorStage {
	stages := map[execStage]*TxExecutorStage{}

	stage1 := NewExecutorStage()
	stage1.
		Stage(pushStageFunc(1)).
		RollbackStage(popRollbackFunc).
		CompleteStage(forceCompleteFunc)

	stage2 := NewExecutorStage()
	stage2.
		Stage(pushStageFunc(2)).
		RollbackStage(popRollbackFunc).
		CompleteStage(forceCompleteFunc)

	stage3 := NewExecutorStage()
	stage3.
		Stage(pushStageFunc(3)).
		RollbackStage(popRollbackFunc).
		CompleteStage(forceCompleteFunc)

	failureStage := NewExecutorStage()
	failureStage.
		Stage(failureStageFunc).
		RollbackStage(popRollbackFunc).
		CompleteStage(forceCompleteFunc)

	unrecoverableFailureStage := NewExecutorStage()
	unrecoverableFailureStage.
		Stage(unrecoverableFailureStageFunc).
		RollbackStage(popRollbackFunc).
		CompleteStage(forceCompleteFunc)

	flakyStage1 := NewExecutorStage()
	flakyStage1.
		Stage(flakyStageFunc(1)).
		RollbackStage(popRollbackFunc).
		CompleteStage(forceCompleteFunc)

	flakyStage2 := NewExecutorStage()
	flakyStage2.
		Stage(flakyStageFunc(2)).
		RollbackStage(popRollbackFunc).
		CompleteStage(forceCompleteFunc)

	flakyStage3 := NewExecutorStage()
	flakyStage3.
		Stage(flakyStageFunc(3)).
		RollbackStage(popRollbackFunc).
		CompleteStage(flakyCompleteFunc)

	flakyFailureStage := NewExecutorStage()
	flakyFailureStage.
		Stage(unrecoverableFailureStageFunc).
		RollbackStage(popRollbackFunc).
		CompleteStage(flakyCompleteFunc)

	recoveryStage1 := NewExecutorStage()
	recoveryStage1.
		Stage(recoveryPushStageFunc(1)).
		CompleteStage(recoveryForceCompleteFunc)

	recoveryStage2 := NewExecutorStage()
	recoveryStage2.
		Stage(recoveryPushStageFunc(2)).
		CompleteStage(recoveryForceCompleteFunc)

	recoveryStage3 := NewExecutorStage()
	recoveryStage3.
		Stage(recoveryPushStageFunc(3)).
		CompleteStage(recoveryForceCompleteFunc)

	stages[execStage1] = stage1
	stages[execStage2] = stage2
	stages[execStage3] = stage3
	stages[execStageFailure] = failureStage
	stages[execStageUnrecoverableFailure] = unrecoverableFailureStage
	stages[execStage1Flaky] = flakyStage1
	stages[execStage2Flaky] = flakyStage2
	stages[execStage3Flaky] = flakyStage3
	stages[execStageFailureFlaky] = flakyFailureStage
	stages[execStage1Recovery] = recoveryStage1
	stages[execStage2Recovery] = recoveryStage2
	stages[execStage3Recovery] = recoveryStage3

	return stages
}
