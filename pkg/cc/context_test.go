package cc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStageContext(t *testing.T) {
	service := "service-a"
	partition := uint64(3)
	ts := uint64(100)
	attrs := []string{"apple", "banana"}
	stageCtx := NewTxStageContext(partition, service, ts, attrs)

	encodedStageCtx := stageCtx.Encode()
	decodedStageCtx, err := DecodeTxStageContext(encodedStageCtx)
	require.NoError(t, err)
	require.Equal(t, stageCtx.Service, decodedStageCtx.Service)
	require.Equal(t, stageCtx.Partition, decodedStageCtx.Partition)
	require.Equal(t, stageCtx.Timestamp, decodedStageCtx.Timestamp)
	require.Equal(t, stageCtx.Attrs, decodedStageCtx.Attrs)
}

func TestControlContext(t *testing.T) {
	service := "service-a"
	partition := uint64(3)
	attrs := []string{"apple", "banana"}
	ctrlCtx := NewTxControlContext(partition, service, attrs)

	encodedCtrlCtx := ctrlCtx.Encode()
	decodedCtrlCtx, err := DecodeTxControlContext(encodedCtrlCtx)
	require.NoError(t, err)
	require.Equal(t, ctrlCtx.Service, decodedCtrlCtx.Service)
	require.Equal(t, ctrlCtx.Partition, decodedCtrlCtx.Partition)
	require.Equal(t, ctrlCtx.Attrs, decodedCtrlCtx.Attrs)
}

func TestExecutorContext(t *testing.T) {
	service := "service-a"
	partition := uint64(3)
	attrs := []string{"apple", "banana"}
	ctrlCtx := NewTxControlContext(partition, service, attrs)

	execID := uint64(200)
	receivers := []string{"service-a", "service-b", "service-c"}
	timestamps := []uint64{100, 135, 127}
	req := "request"
	result := 10
	status := ExecStatusPending
	execCtx := &TxExecutorContext{
		ExecID:     execID,
		CtrlCtx:    ctrlCtx,
		Receivers:  receivers,
		Timestamps: timestamps,
		Req:        req,
		Result:     result,
		Status:     status,
		Curr:       1,
	}

	encodedExecCtx := execCtx.Encode()
	decodedExecCtx, err := DecodeTxExecutorContext(encodedExecCtx)
	require.NoError(t, err)
	require.Equal(t, execCtx.ExecID, decodedExecCtx.ExecID)
	require.Equal(t, execCtx.CtrlCtx, decodedExecCtx.CtrlCtx)
	require.Equal(t, execCtx.Receivers, decodedExecCtx.Receivers)
	require.Equal(t, execCtx.Timestamps, decodedExecCtx.Timestamps)
	require.EqualValues(t, execCtx.Req, decodedExecCtx.Req)
	require.EqualValues(t, execCtx.Result, decodedExecCtx.Result)
	require.Equal(t, execCtx.Status, decodedExecCtx.Status)
	require.Equal(t, execCtx.Curr, decodedExecCtx.Curr)
}
