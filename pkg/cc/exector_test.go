package cc

import (
	"errors"
	"testing"

	"github.com/emirpasic/gods/v2/queues/arrayqueue"
	"github.com/emirpasic/gods/v2/stacks/arraystack"
	"github.com/stretchr/testify/require"
)

func TestTxExecutorInMemory(t *testing.T) {
	successStack := arraystack.New[int]()
	pushStageFunc := func(i int) StageFunc {
		return func(v any) (any, any, error) {
			s := v.(*arraystack.Stack[int])
			s.Push(i)
			return i, s, nil
		}
	}
	failureStageFunc := func(v any) (any, any, error) {
		return v, v, errors.New("some error")
	}
	popRollbackFunc := func(v any) (any, error) {
		s := v.(*arraystack.Stack[int])
		s.Pop()
		return s, nil
	}
	noopRollbackFunc := func(v any) (any, error) {
		return v, nil
	}
	incCompleteFunc := func(v any) (any, error) {
		i := v.(*int)
		*i++
		return nil, nil
	}
	noopCompleteFunc := func(v any) (any, error) {
		return v, nil
	}
	queueCheckpointer := func(ckptQueue *arrayqueue.Queue[int], stack *arraystack.Stack[int]) CheckpointFunc {
		return func(tec *TxExecutorContext) error {
			v, _ := stack.Peek()
			ckptQueue.Enqueue(v)
			return nil
		}
	}

	failureStage := NewExecutorStage()
	failureStage.
		Stage(failureStageFunc).
		RollbackStage(noopRollbackFunc).
		CompleteStage(noopCompleteFunc)

	stage1 := NewExecutorStage()
	stage1.
		Stage(pushStageFunc(1)).
		RollbackStage(popRollbackFunc).
		CompleteStage(incCompleteFunc)

	stage2 := NewExecutorStage()
	stage2.
		Stage(pushStageFunc(2)).
		RollbackStage(popRollbackFunc).
		CompleteStage(incCompleteFunc)

	stage3 := NewExecutorStage()
	stage3.
		Stage(pushStageFunc(3)).
		RollbackStage(popRollbackFunc).
		CompleteStage(incCompleteFunc)

	service := "service-a"
	partition := uint64(3)
	attrs := []string{"apple", "banana"}
	ctrlCtx := NewTxControlContext(partition, service, attrs)

	// success stages
	successExecCtx := &TxExecutorContext{
		CtrlCtx: ctrlCtx,
		Status:  ExecStatusPending,
		Req:     successStack,
		Curr:    0,
	}

	successQueue := arrayqueue.New[int]()
	successCheckpointer := queueCheckpointer(successQueue, successStack)

	successExecutor := NewTxExecutor(successExecCtx, successCheckpointer)
	successExecutor.
		CommitStage(stage1).
		Stage(stage2).
		Stage(stage3)

	v, err := successExecutor.Run()
	require.NoError(t, err)
	require.Equal(t, ExecStatusCommitted, successExecutor.execCtx.Status)

	i, ok := v.(int)
	require.True(t, ok)
	require.Equal(t, 1, i)
	err = successExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{1}, successStack.Values())
	require.Equal(t, []int{1}, successQueue.Values())

	executeCount := 0
	for successExecutor.Next() {
		executeCount++
		err = successExecutor.Execute()
		require.NoError(t, err)
		err = successExecutor.Checkpoint()
		require.NoError(t, err)
	}

	require.Equal(t, 2, executeCount)

	successExecutor.execCtx.Status = ExecStatusCompleted
	err = successExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{3, 2, 1}, successStack.Values())
	require.Equal(t, []int{1, 2, 3, 3}, successQueue.Values())

	// first stage failed
	commitFailureStack := arraystack.New[int]()
	commitFailureExecCtx := &TxExecutorContext{
		CtrlCtx: ctrlCtx,
		Status:  ExecStatusPending,
		Req:     commitFailureStack,
		Curr:    0,
	}
	commitFailureQueue := arrayqueue.New[int]()
	commitFailureCheckpointer := queueCheckpointer(commitFailureQueue, commitFailureStack)
	commitFailureExecutor := NewTxExecutor(commitFailureExecCtx, commitFailureCheckpointer)
	commitFailureExecutor.
		CommitStage(failureStage).
		Stage(stage2).
		Stage(stage3)

	_, err = commitFailureExecutor.Run()
	require.Error(t, err)
	require.True(t, errors.Is(ErrTxExecAborted, errors.Unwrap(err)), err)
	require.Equal(t, ExecStatusAborted, commitFailureExecutor.execCtx.Status)

	// rollback stage
	rollbackStack := arraystack.New[int]()
	rollbackExecCtx := &TxExecutorContext{
		CtrlCtx: ctrlCtx,
		Status:  ExecStatusPending,
		Req:     rollbackStack,
		Curr:    0,
	}
	rollbackQueue := arrayqueue.New[int]()
	rollbackCheckpointer := queueCheckpointer(rollbackQueue, rollbackStack)
	rollbackExecutor := NewTxExecutor(rollbackExecCtx, rollbackCheckpointer)
	rollbackExecutor.
		CommitStage(stage1).
		Stage(stage2).
		Stage(failureStage)

	v, err = rollbackExecutor.Run()
	require.NoError(t, err)
	require.Equal(t, ExecStatusCommitted, rollbackExecutor.execCtx.Status)

	i, ok = v.(int)
	require.True(t, ok)
	require.Equal(t, 1, i)
	err = rollbackExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{1}, rollbackStack.Values())
	require.Equal(t, []int{1}, rollbackQueue.Values())

	err = rollbackExecutor.Execute()
	require.NoError(t, err)
	err = rollbackExecutor.Checkpoint()
	require.NoError(t, err)

	err = rollbackExecutor.Execute()
	require.Error(t, err)
	rollbackExecutor.execCtx.Status = ExecStatusRollback
	err = rollbackExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{2, 1}, rollbackStack.Values())
	require.Equal(t, []int{1, 2, 2}, rollbackQueue.Values())

	rollbackCount := 0
	for rollbackExecutor.Next() {
		rollbackCount++
		err = rollbackExecutor.Rollback()
		require.NoError(t, err)
		err = rollbackExecutor.Checkpoint()
		require.NoError(t, err)
	}

	require.Equal(t, 1, rollbackCount)

	rollbackExecutor.execCtx.Status = ExecStatusCompleted
	err = rollbackExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{1}, rollbackStack.Values())
	require.Equal(t, []int{1, 2, 2, 1, 1}, rollbackQueue.Values())

	// recover stage
	completeStack := arraystack.New[int]()
	completeExecCtx := &TxExecutorContext{
		CtrlCtx: ctrlCtx,
		Status:  ExecStatusPending,
		Req:     completeStack,
		Curr:    0,
	}
	completeQueue := arrayqueue.New[int]()
	completeCheckpointer := queueCheckpointer(completeQueue, completeStack)
	completeExecutor := NewTxExecutor(completeExecCtx, completeCheckpointer)
	completeExecutor.
		CommitStage(stage1).
		Stage(failureStage).
		Stage(failureStage)

	v, err = completeExecutor.Run()
	require.NoError(t, err)
	require.Equal(t, ExecStatusCommitted, completeExecutor.execCtx.Status)

	i, ok = v.(int)
	require.True(t, ok)
	require.Equal(t, 1, i)
	err = completeExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{1}, completeStack.Values())
	require.Equal(t, []int{1}, completeQueue.Values())

	err = completeExecutor.Execute()
	require.Error(t, err)
	completeExecutor.execCtx.Status = ExecStatusForceComplete
	err = completeExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{1}, completeStack.Values())
	require.Equal(t, []int{1, 1}, completeQueue.Values())

	completeCount := 0
	for completeExecutor.Next() {
		completeCount++
		err = completeExecutor.ForceComplete()
		require.NoError(t, err)
		err = completeExecutor.Checkpoint()
		require.NoError(t, err)
	}

	require.Equal(t, 2, completeCount)

	completeExecutor.execCtx.Status = ExecStatusCompleted
	err = completeExecutor.Checkpoint()
	require.NoError(t, err)

	require.Equal(t, []int{1}, completeStack.Values())
	require.Equal(t, []int{1, 1, 1, 1, 1}, completeQueue.Values())
}

func TestTxExecutorInDB(t *testing.T) {

}
