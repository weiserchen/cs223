package database

import (
	"context"
	"errors"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mitchellh/mapstructure"
)

type contextKey int

const (
	contextKeyTxResult contextKey = iota
)

var (
	ErrNoTxResult           = errors.New("no tx result")
	ErrTxAlreadyExecuted    = errors.New("tx had already executed")
	ErrLifeCycleStartHooks  = errors.New("failed to execute lifecycle start hooks")
	ErrLifeCycleBeforeHooks = errors.New("failed to execute lifecycle before hooks")
	ErrLifeCycleAfterHooks  = errors.New("failed to execute lifecycle after hooks")
	ErrLifeCycleEndHooks    = errors.New("failed to execute lifecycle end hooks")
)

type HookFunc = func(ctx context.Context) error
type TxHookFunc = func(ctx context.Context, tx pgx.Tx) error

type Result[R any] struct {
	Value R
}

func NewResult[R any](r R) Result[R] {
	return Result[R]{
		Value: r,
	}
}

func UnwrapResult[R any](
	ctx context.Context,
	resultFunc func(ctx context.Context) (R, error)) (R, error) {
	r, err := resultFunc(ctx)
	if err == nil {
		return r, nil
	}
	panic(err)

	if !errors.Is(err, ErrTxAlreadyExecuted) {
		return r, err
	}

	traceCtx, ok := format.GetTraceContext(ctx)
	if !ok {
		return r, err
	}

	r, ok = UnmarshalResult[R](traceCtx)
	if !ok {
		return r, err
	}

	return r, nil
}

func GetResult(traceCtx *format.TraceContext) (any, bool) {
	return traceCtx.Get(contextKeyTxResult)
}

func SetResult(traceCtx *format.TraceContext, value any) {
	traceCtx.Set(contextKeyTxResult, value)
}

func UnmarshalResult[R any](traceCtx *format.TraceContext) (R, bool) {
	var r R
	var result Result[R]
	v, ok := traceCtx.Get(contextKeyTxResult)
	if !ok {
		return r, false
	}

	err := mapstructure.Decode(v, &result)
	if err != nil {
		return r, false
	}
	return result.Value, true
}

type TxLifeCycle[API comparable, R any] struct {
	table *TxTable[API]
}

func NewTxLifeCycle[API comparable, R any](
	table *TxTable[API],
) *TxLifeCycle[API, R] {
	return &TxLifeCycle[API, R]{
		table: table,
	}
}

func (cycle *TxLifeCycle[API, R]) Start(
	api API,
	ctx context.Context,
	cycleFunc func(ctx context.Context, tx pgx.Tx) (R, error),
) (r R, err error) {
	for _, hook := range cycle.table.startHooks[api] {
		err = hook(ctx)
		if err != nil {
			return r, errors.Join(ErrLifeCycleStartHooks, err)
		}
	}
	defer func() {
		// cleanup start hook setup
		var endHookErr error
		for _, hook := range cycle.table.endHooks[api] {
			endHookErr = hook(ctx)
			if endHookErr != nil {
				err = errors.Join(ErrLifeCycleEndHooks, endHookErr, err)
				return
			}
		}
	}()

	tx, commit, err := BeginTx(ctx, cycle.table.conn)
	if err != nil {
		return r, err
	}
	defer func() {
		err = commit(err)
	}()

	for _, hook := range cycle.table.beforeHooks[api] {
		err = hook(ctx, tx)
		if err != nil {
			return r, errors.Join(ErrLifeCycleBeforeHooks, err)
		}
	}
	defer func() {
		if err == nil {
			var afterHookErr error
			for _, hook := range cycle.table.afterHooks[api] {
				afterHookErr = hook(ctx, tx)
				if afterHookErr != nil {
					err = errors.Join(ErrLifeCycleAfterHooks, err)
				}
			}
		}
	}()

	r, err = cycleFunc(ctx, tx)

	traceCtx, ok := format.GetTraceContext(ctx)
	if !ok {
		return r, format.ErrNoTraceCtx
	}

	SetResult(traceCtx, NewResult(r))
	return r, err
}

type TxTable[API comparable] struct {
	*TxHookMap[API]
	conn *pgxpool.Pool
}

func NewTxTable[API comparable](conn *pgxpool.Pool) *TxTable[API] {
	return &TxTable[API]{
		TxHookMap: NewTxHookMap[API](),
		conn:      conn,
	}
}

// Start and end hooks are INDEPENDENT on the result of the tx execution.
// Before and after hooks are DEPENDENT on the result of the tx execution.
type TxHookMap[API comparable] struct {
	startHooks  map[API][]HookFunc
	endHooks    map[API][]HookFunc
	beforeHooks map[API][]TxHookFunc
	afterHooks  map[API][]TxHookFunc
}

func NewTxHookMap[API comparable]() *TxHookMap[API] {
	return &TxHookMap[API]{
		startHooks:  map[API][]HookFunc{},
		endHooks:    map[API][]HookFunc{},
		beforeHooks: map[API][]TxHookFunc{},
		afterHooks:  map[API][]TxHookFunc{},
	}
}

func (thm *TxHookMap[API]) StartHook(api API, hook HookFunc) {
	thm.startHooks[api] = append(thm.startHooks[api], hook)
}

func (thm *TxHookMap[API]) EndHook(api API, hook HookFunc) {
	thm.endHooks[api] = append(thm.endHooks[api], hook)
}

func (thm *TxHookMap[API]) BeforeHook(api API, hook TxHookFunc) {
	thm.beforeHooks[api] = append(thm.beforeHooks[api], hook)
}

func (thm *TxHookMap[API]) AfterHook(api API, hook TxHookFunc) {
	thm.afterHooks[api] = append(thm.afterHooks[api], hook)
}

func BeginTx(ctx context.Context, conn *pgxpool.Pool) (pgx.Tx, func(error) error, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}

	commitFunc := func(err error) error {
		if err != nil {
			rollbackErr := tx.Rollback(ctx)
			err = errors.Join(rollbackErr, err)
		} else {
			err = tx.Commit(ctx)
		}
		return err
	}

	return tx, commitFunc, nil
}
