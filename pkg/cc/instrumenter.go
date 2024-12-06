package cc

import (
	"context"
	"log"
	"txchain/pkg/database"
	"txchain/pkg/format"
)

type TxRecorder interface {
	VisitBefore(ctx context.Context)
	VisitAfter(ctx context.Context)
}

var _ TxRecorder = (*TxInstrumenter)(nil)

type TxInstrumenter struct {
	recorders []TxRecorder
}

func NewTxInstrumenter() *TxInstrumenter {
	return &TxInstrumenter{}
}

func (inst *TxInstrumenter) Recorder(recorder TxRecorder) {
	inst.recorders = append(inst.recorders, recorder)
}

func (inst *TxInstrumenter) VisitBefore(ctx context.Context) {
	for _, recorder := range inst.recorders {
		recorder.VisitBefore(ctx)
	}
}

func (inst *TxInstrumenter) VisitAfter(ctx context.Context) {
	for _, recorder := range inst.recorders {
		recorder.VisitAfter(ctx)
	}
}

type TraceRecorder struct {
	requests  [][]any
	responses [][]any
}

var _ TxRecorder = (*TraceRecorder)(nil)

func NewTraceRecorder(partitions uint64) *TraceRecorder {
	partitions = GenPartitions(partitions)
	requests := make([][]any, partitions)
	responses := make([][]any, partitions)
	for partition := range partitions {
		requests[partition] = []any{}
		responses[partition] = []any{}
	}
	return &TraceRecorder{
		requests:  requests,
		responses: responses,
	}
}

func (r *TraceRecorder) VisitBefore(ctx context.Context) {
	execCtx, ok := GetTxExecCtx(ctx)
	if !ok {
		return
	}

	partition := execCtx.CtrlCtx.Partition
	r.requests[partition] = append(r.requests[partition], execCtx.Input)
}

func (r *TraceRecorder) VisitAfter(ctx context.Context) {
	log.Println("trace-recorder")
	stageCtx, ok := GetTxStageCtx(ctx)
	if !ok {
		return
	}

	log.Println("have stage-ctx")

	traceCtx, ok := format.GetTraceContext(ctx)
	if !ok {
		return
	}

	log.Println("have trace-ctx")

	result, ok := database.GetResult(traceCtx)
	if !ok {
		return
	}

	log.Println("instrument:", result)

	partition := stageCtx.Partition
	r.responses[partition] = append(r.responses[partition], result)

}

func (r *TraceRecorder) GetReq(partition uint64) []any {
	return r.requests[partition]
}

func (r *TraceRecorder) GetResp(partition uint64) []any {
	return r.responses[partition]
}
