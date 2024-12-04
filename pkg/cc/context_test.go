package cc

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStageContext(t *testing.T) {
	stageCtx := &TxStageContext{
		Partition: 3,
		Service:   "service-a",
		Timestamp: 100,
		Attrs:     []string{"apple", "banana"},
		DryRun:    true,
	}

	encodedStageCtx := stageCtx.Encode()
	decodedStageCtx, err := DecodeTxStageContext(encodedStageCtx)
	require.NoError(t, err)
	checkTxStageCtx(t, stageCtx, decodedStageCtx)
}

func TestControlContext(t *testing.T) {
	ctrlCtx := &TxControlContext{
		Partition: 3,
		Service:   "service-a",
		Attrs:     []string{"apple", "banana"},
		DryRun:    true,
	}

	encodedCtrlCtx := ctrlCtx.Encode()
	decodedCtrlCtx, err := DecodeTxControlContext(encodedCtrlCtx)
	require.NoError(t, err)
	checkTxCtrlCtx(t, ctrlCtx, decodedCtrlCtx)
}

func TestExecutorContext(t *testing.T) {
	ctrlCtx := &TxControlContext{
		Partition: 3,
		Service:   "service-a",
		Attrs:     []string{"apple", "banana"},
		DryRun:    true,
	}

	type Input struct {
		Value string
	}
	type Result struct {
		Value int
	}
	execCtx := &TxExecutorContext{
		ExecID:     200,
		CtrlCtx:    ctrlCtx,
		Receivers:  []string{"service-a", "service-b", "service-c"},
		Timestamps: []uint64{100, 135, 127},
		Input:      Input{"input"},
		Result:     Result{10},
		Status:     ExecStatusPending,
		Curr:       1,
		Method:     http.MethodPost,
		Endpoint:   "127.0.0.1:8080",
	}

	encodedExecCtx := execCtx.Encode()
	decodedExecCtx, err := DecodeTxExecutorContext(encodedExecCtx)
	require.NoError(t, err)
	checkTxExecCtx[Input, Result](t, execCtx, decodedExecCtx)
}

func checkTxStageCtx(t *testing.T, expected, got *TxStageContext) {
	t.Helper()

	require.NotNil(t, expected)
	require.NotNil(t, got)
	require.Equal(t, expected.Partition, got.Partition)
	require.Equal(t, expected.Service, got.Service)
	require.Equal(t, expected.Timestamp, got.Timestamp)
	require.Equal(t, expected.Attrs, got.Attrs)
	require.Equal(t, expected.DryRun, got.DryRun)
}

func checkTxCtrlCtx(t *testing.T, expected, got *TxControlContext) {
	t.Helper()

	require.NotNil(t, expected)
	require.NotNil(t, got)
	require.Equal(t, expected.Partition, got.Partition)
	require.Equal(t, expected.Service, got.Service)
	require.Equal(t, expected.Attrs, got.Attrs)
	require.Equal(t, expected.DryRun, got.DryRun)
}

func checkTypeEqual[T any](t *testing.T, expected, got any) {
	t.Helper()

	expectedValue, err := UnmarshalInput[T](expected)
	require.NoError(t, err)
	gotValue, err := UnmarshalInput[T](got)
	require.NoError(t, err)
	require.EqualValues(t, expectedValue, gotValue)
}

func checkTxExecCtx[A, B any](t *testing.T, expected, got *TxExecutorContext) {
	t.Helper()

	require.NotNil(t, expected)
	require.NotNil(t, got)

	checkTxCtrlCtx(t, expected.CtrlCtx, got.CtrlCtx)
	require.Equal(t, expected.ExecID, got.ExecID)
	require.Equal(t, expected.Receivers, got.Receivers)
	require.Equal(t, expected.Timestamps, got.Timestamps)
	require.Equal(t, expected.Status, got.Status)

	checkTypeEqual[A](t, expected.Input, got.Input)
	checkTypeEqual[B](t, expected.Result, got.Result)

	require.Equal(t, expected.Curr, got.Curr)
	require.Equal(t, expected.Method, got.Method)
	require.Equal(t, expected.Endpoint, got.Endpoint)
}
