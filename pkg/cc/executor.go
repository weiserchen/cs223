package cc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTxExecEmpty          = errors.New("tx empty execution")
	ErrTxExecAborted        = errors.New("tx execution aborted")
	ErrTxExecUnrecoverable  = errors.New("unrecoverable tx execution error")
	ErrTxExecForceComplete  = errors.New("failed to force complete executor")
	ErrTxExecCheckpoint     = errors.New("failed to perform execution checkpoint")
	ErrTxExecStageEmptyFunc = errors.New("empty exec stage func")
)

type CommitStageFunc = func(any) (any, any, error)
type RetryFunc = func(int) time.Duration
type StageFunc = func(any) (any, any, error)
type RollbackFunc = func(any) (any, error)
type CompleteFunc = func(any) (any, error)
type CheckpointFunc = func(*TxExecutorContext) error

func constantRetry(duration time.Duration) RetryFunc {
	return func(retryTime int) time.Duration {
		return time.Millisecond * duration
	}
}

func DefaultCheckpointer(conn *pgxpool.Pool) CheckpointFunc {
	return func(execCtx *TxExecutorContext) error {
		return CheckpointExecutorContext(conn, execCtx)
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

type TxExecutorManager struct {
	recvQueue   chan *TxExecutor
	retryPeriod func(int) time.Duration
}

func NewTxExecutorManager() *TxExecutorManager {
	return &TxExecutorManager{
		recvQueue:   make(chan *TxExecutor),
		retryPeriod: constantRetry(time.Second),
	}
}

func CheckpointExecutorContext(conn *pgxpool.Pool, execCtx *TxExecutorContext) error {
	b, err := json.Marshal(execCtx)
	if err != nil {
		return err
	}

	query := `
		UPDATE SET
			status = $2, 
			checkpoint = $3
		WHERE exec_id = $1;
	`

	ctx := context.Background()
	execID := execCtx.ExecID
	_, err = conn.Exec(ctx, query, execID, execCtx.Status, b)
	return err
}

func (mgr *TxExecutorManager) Send(exec *TxExecutor) {
	mgr.recvQueue <- exec
}

func (mgr *TxExecutorManager) Run() {
	for exec := range mgr.recvQueue {
		if exec.execCtx.Status == ExecStatusForceComplete {
			go func() {
				if err := exec.ForceComplete(); err != nil {
					exec.retryTime += 1
					time.Sleep(mgr.retryPeriod(exec.retryTime))
					mgr.Send(exec)
				}
			}()
		} else {
			go func() {
				for exec.Next() {
					if err := exec.Execute(); err != nil {
						if errors.Is(ErrTxExecUnrecoverable, err) {
							exec.execCtx.Status = ExecStatusForceComplete
							err = exec.ForceComplete()
						}
						if err != nil {
							mgr.retry(exec)
						}
						return
					}
					if err := exec.Checkpoint(); err != nil {
						mgr.retry(exec)
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
	time.Sleep(mgr.retryPeriod(exec.retryTime))
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
	req := exec.execCtx.Req
	stage := exec.stages[curr]
	stage.Execute(req)
	if stage.Error() != nil {
		return stage.Error()
	}

	exec.execCtx.Curr += 1
	exec.execCtx.Req = stage.output
	return nil
}

func (exec *TxExecutor) Rollback() error {
	exec.execCtx.Curr -= 1
	curr := exec.execCtx.Curr
	req := exec.execCtx.Req
	stage := exec.stages[curr]
	stage.Rollback(req)
	if stage.Error() != nil {
		return stage.Error()
	}

	exec.execCtx.Req = stage.output
	return nil
}

func (exec *TxExecutor) ForceComplete() error {
	curr := exec.execCtx.Curr
	stage := exec.stages[curr]
	stage.Complete(exec.execCtx.Req)
	if stage.Error() != nil {
		return fmt.Errorf("%w: %v", ErrTxExecForceComplete, stage.Error())
	}
	exec.execCtx.Curr += 1
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
		req := exec.execCtx.Req
		commitStage := exec.commitStage
		commitStage.Execute(req)
		if commitStage.Error() != nil {
			exec.execCtx.Status = ExecStatusAborted
			return nil, fmt.Errorf("%w: %v", ErrTxExecAborted, commitStage.Error())
		}

		exec.execCtx.Req = commitStage.output
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
	result, err := stage.rollbackFunc(v)
	stage.result = result
	stage.err = err
}

func (stage *TxExecutorStage) Complete(v any) {
	if stage.completeFunc == nil {
		stage.err = ErrTxExecStageEmptyFunc
		return
	}
	result, err := stage.completeFunc(v)
	stage.result = result
	stage.err = err
}

func (stage *TxExecutorStage) Result() any {
	return stage.result
}

func (stage *TxExecutorStage) Error() error {
	return stage.err
}
