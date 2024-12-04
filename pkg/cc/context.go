package cc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/mitchellh/mapstructure"
)

var (
	ErrTxStageContextDecode    = errors.New("failed to decode tx stage context")
	ErrTxControlContextDecode  = errors.New("failed to decode tx control context")
	ErrTxExecutorContextDecode = errors.New("failed to decode tx executor context")
)

type contextKeyTxCtx int

const (
	contextKeyTxStageCtx contextKeyTxCtx = iota
	contextKeyTxCtrlCtx
	contextKeyTxExecCtx
)

func GetTxStageCtx(ctx context.Context) (*TxStageContext, bool) {
	stageCtx, ok := ctx.Value(contextKeyTxStageCtx).(*TxStageContext)
	return stageCtx, ok
}

func SetTxStageCtx(ctx context.Context, stageCtx *TxStageContext) context.Context {
	return context.WithValue(ctx, contextKeyTxStageCtx, stageCtx)
}

func GetTxCtrlCtx(ctx context.Context) (*TxControlContext, bool) {
	ctrlCtx, ok := ctx.Value(contextKeyTxCtrlCtx).(*TxControlContext)
	return ctrlCtx, ok
}

func SetTxCtrlCtx(ctx context.Context, ctrlCtx *TxControlContext) context.Context {
	return context.WithValue(ctx, contextKeyTxCtrlCtx, ctrlCtx)
}

func GetTxExecCtx(ctx context.Context) (*TxExecutorContext, bool) {
	execCtx, ok := ctx.Value(contextKeyTxExecCtx).(*TxExecutorContext)
	return execCtx, ok
}

func SetTxExecCtx(ctx context.Context, execCtx *TxExecutorContext) context.Context {
	return context.WithValue(ctx, contextKeyTxExecCtx, execCtx)
}

func UnmarshalInput[T any](v any) (T, error) {
	var input T
	err := mapstructure.Decode(v, &input)
	return input, err
}

type TxStageContext struct {
	Partition uint64   `json:"partition"`
	Service   string   `json:"service"`
	Timestamp uint64   `json:"timestamp"`
	Attrs     []string `json:"attrs"`
	DryRun    bool     `json:"dry_run"`
}

func DecodeTxStageContext(encoded string) (*TxStageContext, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var stageCtx TxStageContext
	if err := json.Unmarshal(b, &stageCtx); err != nil {
		return nil, err
	}

	return &stageCtx, nil
}

func (stageCtx *TxStageContext) Encode() string {
	b, _ := json.Marshal(stageCtx)
	return base64.StdEncoding.EncodeToString(b)
}

type TxControlContext struct {
	Partition uint64   `json:"partition"`
	Service   string   `json:"service"`
	Attrs     []string `json:"attrs"`
	DryRun    bool     `json:"dry_run"`
}

func DecodeTxControlContext(encoded string) (*TxControlContext, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var ctrlCtx TxControlContext
	if err := json.Unmarshal(b, &ctrlCtx); err != nil {
		return nil, err
	}

	return &ctrlCtx, nil
}

func (ctrlCtx *TxControlContext) Encode() string {
	b, _ := json.Marshal(ctrlCtx)
	return base64.StdEncoding.EncodeToString(b)
}

type ExecStatus int

const (
	ExecStatusPending ExecStatus = iota
	ExecStatusCommitted
	ExecStatusAborted
	ExecStatusRollback
	ExecStatusForceComplete
	ExecStatusCompleted
)

type TxExecutorContext struct {
	ExecID     uint64
	CtrlCtx    *TxControlContext
	Receivers  []string
	Timestamps []uint64
	Input      any
	Result     any
	Status     ExecStatus
	Curr       int
	Method     string
	Endpoint   string
}

func DecodeTxExecutorContext(encoded string) (*TxExecutorContext, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var execCtx TxExecutorContext
	if err := json.Unmarshal(b, &execCtx); err != nil {
		return nil, err
	}

	return &execCtx, nil
}

func (execCtx *TxExecutorContext) Encode() string {
	b, _ := json.Marshal(execCtx)
	return base64.StdEncoding.EncodeToString(b)
}
