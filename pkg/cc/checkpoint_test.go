package cc

import (
	"context"
	"testing"
	"txchain/pkg/database"
	"txchain/pkg/format"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

type testAPI int

const (
	testAPITxResult testAPI = iota
)

func TestTxDedupHooks(t *testing.T) {
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

	rootCtx := format.InsertTraceContext(ctx)

	table := database.NewTxTable[testAPI](conn)

	beforeCount, afterCount := 0, 0
	testBeforeHook := func(ctx context.Context, tx pgx.Tx) error {
		beforeCount++
		return TxDedupBeforeHook(ctx, tx)
	}
	testAfterHook := func(ctx context.Context, tx pgx.Tx) error {
		afterCount++
		return TxDedupAfterHook(ctx, tx)
	}

	table.BeforeHook(testAPITxResult, testBeforeHook)
	table.AfterHook(testAPITxResult, testAfterHook)

	stringResultFunc := func(ctx context.Context, tx pgx.Tx) (string, error) {
		return "string", nil
	}
	nilResultFunc := func(ctx context.Context, tx pgx.Tx) (any, error) {
		return nil, nil
	}
	type Input struct {
		Value int
	}
	structResultFunc := func(ctx context.Context, tx pgx.Tx) (Input, error) {
		return Input{100}, nil
	}

	stringStageCtx := &TxStageContext{
		Partition: 3,
		Service:   "service-string",
		Timestamp: 10,
	}
	strCtx := SetTxStageCtx(rootCtx, stringStageCtx)

	stringResultLifeCycle := database.NewTxLifeCycle[testAPI, string](table)
	strResult, err := database.UnwrapResult(
		strCtx,
		func(ctx context.Context) (string, error) {
			return stringResultLifeCycle.Start(testAPITxResult, ctx, stringResultFunc)
		},
	)
	require.NoError(t, err)
	require.Equal(t, "string", strResult)
	require.Equal(t, 1, beforeCount)
	require.Equal(t, 1, afterCount)

	dupStringResultLifeCycle := database.NewTxLifeCycle[testAPI, string](table)
	strResult, err = database.UnwrapResult(
		strCtx,
		func(ctx context.Context) (string, error) {
			return dupStringResultLifeCycle.Start(testAPITxResult, ctx, stringResultFunc)
		},
	)
	require.NoError(t, err)
	require.Equal(t, "string", strResult)
	require.Equal(t, 2, beforeCount)
	require.Equal(t, 1, afterCount)

	nilStageCtx := &TxStageContext{
		Partition: 3,
		Service:   "service-nil",
		Timestamp: 10,
	}
	nilCtx := SetTxStageCtx(rootCtx, nilStageCtx)

	nilResultLifeCycle := database.NewTxLifeCycle[testAPI, any](table)
	nilResult, err := database.UnwrapResult(
		nilCtx,
		func(ctx context.Context) (any, error) {
			return nilResultLifeCycle.Start(testAPITxResult, ctx, nilResultFunc)
		},
	)
	require.NoError(t, err)
	require.Nil(t, nilResult, nilResult)
	require.Equal(t, 3, beforeCount)
	require.Equal(t, 2, afterCount)

	nilResultDupLifeCycle := database.NewTxLifeCycle[testAPI, any](table)
	nilResult, err = database.UnwrapResult(
		nilCtx,
		func(ctx context.Context) (any, error) {
			return nilResultDupLifeCycle.Start(testAPITxResult, ctx, nilResultFunc)
		},
	)
	require.NoError(t, err)
	require.Nil(t, nilResult, nilResult)
	require.Equal(t, 4, beforeCount)
	require.Equal(t, 2, afterCount)

	structStageCtx := &TxStageContext{
		Partition: 3,
		Service:   "service-struct",
		Timestamp: 10,
	}
	structCtx := SetTxStageCtx(rootCtx, structStageCtx)

	structResultLifeCycle := database.NewTxLifeCycle[testAPI, Input](table)
	structResult, err := database.UnwrapResult(
		structCtx,
		func(ctx context.Context) (Input, error) {
			return structResultLifeCycle.Start(testAPITxResult, ctx, structResultFunc)
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, Input{100}, structResult)
	require.Equal(t, 5, beforeCount)
	require.Equal(t, 3, afterCount)

	structResultDupLifeCycle := database.NewTxLifeCycle[testAPI, Input](table)
	structResult, err = database.UnwrapResult(
		structCtx,
		func(ctx context.Context) (Input, error) {
			return structResultDupLifeCycle.Start(testAPITxResult, ctx, structResultFunc)
		},
	)
	require.NoError(t, err)
	require.EqualValues(t, Input{100}, structResult)
	require.Equal(t, 6, beforeCount)
	require.Equal(t, 3, afterCount)
}
