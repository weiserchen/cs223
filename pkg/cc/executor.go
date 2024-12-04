package cc

import (
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrTxExecEmpty          = errors.New("tx empty execution")
	ErrTxExecAborted        = errors.New("tx execution aborted")
	ErrTxExecUnrecoverable  = errors.New("unrecoverable tx execution error")
	ErrTxExecForceComplete  = errors.New("failed to force complete executor")
	ErrTxExecCheckpoint     = errors.New("failed to perform execution checkpoint")
	ErrTxExecStageEmptyFunc = errors.New("empty exec stage func")
)

type RetryFunc = func(retryTime int) time.Duration
type StageFunc = func(input any) (result any, output any, err error)
type RollbackFunc = func(input any) (output any, err error)
type CompleteFunc = func(input any) (output any, err error)

func ConstantRetry(i int) RetryFunc {
	return func(retryTime int) time.Duration {
		return time.Duration(i) * time.Millisecond
	}
}

// start from 1 ms
func ExponentialBackoffRetry(max time.Duration) RetryFunc {
	return func(retryTime int) time.Duration {
		waitPeriod := time.Duration(math.Pow(2, float64(retryTime-1))) * time.Millisecond
		if waitPeriod > max {
			waitPeriod = max
		}
		return waitPeriod
	}
}

type TxExecutorManager struct {
	recvQueue chan *TxExecutor
	retryFunc func(int) time.Duration
}

func NewTxExecutorManager(retryFunc RetryFunc) *TxExecutorManager {
	return &TxExecutorManager{
		recvQueue: make(chan *TxExecutor),
		retryFunc: retryFunc,
	}
}

func (mgr *TxExecutorManager) Send(exec *TxExecutor) {
	mgr.recvQueue <- exec
}

func (mgr *TxExecutorManager) Run() {
	for exec := range mgr.recvQueue {
		if exec.execCtx.Status == ExecStatusForceComplete {
			go func() {
				for exec.Next() {
					if err := exec.ForceComplete(); err != nil {
						mgr.retry(exec)
						return
					}

					if err := exec.Checkpoint(); err != nil {
						mgr.retry(exec)
						return
					}
					exec.retryTime = 0
				}

				exec.execCtx.Status = ExecStatusCompleted
				if err := exec.Checkpoint(); err != nil {
					mgr.retry(exec)
				}
			}()
		} else {
			go func() {
				for exec.Next() {
					if exec.execCtx.Status == ExecStatusForceComplete {
						// use another branch to handle
						mgr.retry(exec)
						return
					} else {
						if err := exec.Execute(); err != nil {
							// normal case -> just retry
							if !errors.Is(err, ErrTxExecUnrecoverable) {
								mgr.retry(exec)
								return
							}
							// unrecoverable -> force complete
							exec.execCtx.Status = ExecStatusForceComplete
						}
					}

					if err := exec.Checkpoint(); err != nil {
						mgr.retry(exec)
						return
					}
					exec.retryTime = 0
				}

				exec.execCtx.Status = ExecStatusCompleted
				if err := exec.Checkpoint(); err != nil {
					mgr.retry(exec)
				}
			}()
		}
	}
}

func (mgr *TxExecutorManager) retry(exec *TxExecutor) {
	exec.retryTime += 1
	// log.Println(mgr.retryFunc(exec.retryTime))
	time.Sleep(mgr.retryFunc(exec.retryTime))
	mgr.Send(exec)
}

type TxExecutor struct {
	execCtx      *TxExecutorContext
	checkpointer func(*TxExecutorContext) error
	retryTime    int
	commitStage  *TxExecutorStage
	stages       []*TxExecutorStage
}

func NewTxExecutor(execCtx *TxExecutorContext, checkpointer CheckpointFunc) *TxExecutor {
	return &TxExecutor{
		execCtx:      execCtx,
		checkpointer: checkpointer,
	}
}

func (exec *TxExecutor) CommitStage(stage *TxExecutorStage) *TxExecutor {
	exec.commitStage = stage
	return exec
}

func (exec *TxExecutor) Stage(stage *TxExecutorStage) *TxExecutor {
	exec.stages = append(exec.stages, stage)
	return exec
}

func (exec *TxExecutor) Checkpoint() error {
	return exec.checkpointer(exec.execCtx)
}

func (exec *TxExecutor) Next() bool {
	status := exec.execCtx.Status
	curr := exec.execCtx.Curr
	if status == ExecStatusRollback {
		return curr > 0
	} else {
		return curr < len(exec.stages)
	}
}

func (exec *TxExecutor) Execute() error {
	curr := exec.execCtx.Curr
	input := exec.execCtx.Input
	stage := exec.stages[curr]
	stage.Execute(input)
	if stage.Err() != nil {
		return stage.Err()
	}

	exec.execCtx.Curr += 1
	exec.execCtx.Input = stage.output
	return nil
}

func (exec *TxExecutor) Rollback() error {
	exec.execCtx.Curr -= 1
	curr := exec.execCtx.Curr
	input := exec.execCtx.Input
	stage := exec.stages[curr]
	stage.Rollback(input)
	if stage.Err() != nil {
		return stage.Err()
	}

	exec.execCtx.Input = stage.output
	return nil
}

func (exec *TxExecutor) ForceComplete() error {
	curr := exec.execCtx.Curr
	stage := exec.stages[curr]
	stage.Complete(exec.execCtx.Input)
	if stage.Err() != nil {
		return fmt.Errorf("%w: %v", ErrTxExecForceComplete, stage.Err())
	}
	exec.execCtx.Curr += 1
	exec.execCtx.Input = stage.output
	return nil
}

func (exec *TxExecutor) Run() (any, error) {
	switch exec.execCtx.Status {
	case ExecStatusAborted:
		return nil, ErrTxExecAborted
	case ExecStatusForceComplete:
		return nil, ErrTxExecUnrecoverable
	case ExecStatusCommitted | ExecStatusCompleted:
		return exec.execCtx.Result, nil
	default:
		input := exec.execCtx.Input
		commitStage := exec.commitStage
		commitStage.Execute(input)
		if commitStage.Err() != nil {
			exec.execCtx.Status = ExecStatusAborted
			return nil, fmt.Errorf("%w: %v", ErrTxExecAborted, commitStage.Err())
		}

		exec.execCtx.Input = commitStage.output
		exec.execCtx.Result = commitStage.result
		exec.execCtx.Status = ExecStatusCommitted

		return commitStage.result, nil
	}
}

type TxExecutorStage struct {
	stageFunc    StageFunc
	rollbackFunc RollbackFunc
	completeFunc CompleteFunc
	output       any
	result       any
	err          error
}

func NewExecutorStage() *TxExecutorStage {
	return &TxExecutorStage{}
}

func (stage *TxExecutorStage) Stage(f StageFunc) *TxExecutorStage {
	stage.stageFunc = f
	return stage
}

func (stage *TxExecutorStage) RollbackStage(f RollbackFunc) *TxExecutorStage {
	stage.rollbackFunc = f
	return stage
}

func (stage *TxExecutorStage) CompleteStage(f CompleteFunc) *TxExecutorStage {
	stage.completeFunc = f
	return stage
}

func (stage *TxExecutorStage) Execute(v any) {
	if stage.stageFunc == nil {
		stage.err = ErrTxExecStageEmptyFunc
		return
	}
	result, output, err := stage.stageFunc(v)
	stage.result = result
	stage.output = output
	stage.err = err
}

func (stage *TxExecutorStage) Rollback(v any) {
	if stage.rollbackFunc == nil {
		stage.err = ErrTxExecStageEmptyFunc
		return
	}
	output, err := stage.rollbackFunc(v)
	stage.output = output
	stage.err = err
}

func (stage *TxExecutorStage) Complete(v any) {
	if stage.completeFunc == nil {
		stage.err = ErrTxExecStageEmptyFunc
		return
	}
	output, err := stage.completeFunc(v)
	stage.output = output
	stage.err = err
}

func (stage *TxExecutorStage) Result() any {
	return stage.result
}

func (stage *TxExecutorStage) Err() error {
	return stage.err
}
