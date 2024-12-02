package cc

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

var (
	ErrTxStageContextDecode    = errors.New("failed to decode tx stage context")
	ErrTxControlContextDecode  = errors.New("failed to decode tx control context")
	ErrTxExecutorContextDecode = errors.New("failed to decode tx executor context")
)

type TxStageContext struct {
	Partition uint64   `json:"partition"`
	Service   string   `json:"service"`
	Timestamp uint64   `json:"timestamp"`
	Attrs     []string `json:"attrs"`
}

func NewTxStageContext(partition uint64, service string, timestamp uint64, attrs []string) *TxStageContext {
	return &TxStageContext{
		Partition: partition,
		Service:   service,
		Timestamp: timestamp,
		Attrs:     attrs,
	}
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
}

func NewTxControlContext(partition uint64, service string, attrs []string) *TxControlContext {
	return &TxControlContext{
		Service:   service,
		Partition: partition,
		Attrs:     attrs,
	}
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
	ExecStatusPending       ExecStatus = 0
	ExecStatusCommitted     ExecStatus = 1
	ExecStatusAborted       ExecStatus = 2
	ExecStatusRollback      ExecStatus = 3
	ExecStatusForceComplete ExecStatus = 4
	ExecStatusCompleted     ExecStatus = 5
)

type TxExecutorContext struct {
	ExecID     uint64            `json:"exec_id"`
	CtrlCtx    *TxControlContext `json:"ctrl_ctx"`
	Receivers  []string          `json:"receivers"`
	Timestamps []uint64          `json:"timestamps"`
	Req        any               `json:"req"`
	Result     any               `json:"result"`
	Status     ExecStatus        `json:"status"`
	Curr       int               `json:"curr"`
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
