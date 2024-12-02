package middleware

import (
	"net/http"
	"txchain/pkg/cc"
)

type contextKey string

const (
	contextKeyRequest           contextKey = "request"
	contextKeyAppendID          contextKey = "append-id"
	contextKeyTxStageContext    contextKey = "tx-stage-context"
	contextKeyTxControlContext  contextKey = "tx-control-context"
	contextKeyTxExecutorContext contextKey = "tx-executor-context"
)

const (
	headerTxStageContext    = "X-Tx-Stage-Context"
	headerTxControlContext  = "X-Tx-Control-Context"
	headerTxExecutorContext = "X-Tx-Executor-Context"
)

type Middlerware func(next http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middlerware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func MarshalRequest[T any](r *http.Request) T {
	return r.Context().Value(contextKeyRequest).(T)
}

func MarshalID(r *http.Request) string {
	id, ok := r.Context().Value(contextKeyAppendID).(string)
	if !ok {
		return ""
	}
	return id
}

func MarshalTxStageContext(r *http.Request) (*cc.TxStageContext, bool) {
	stageCtx, ok := r.Context().Value(contextKeyTxStageContext).(*cc.TxStageContext)
	return stageCtx, ok
}

func MarshalTxControlContext(r *http.Request) (*cc.TxControlContext, bool) {
	ctrlCtx, ok := r.Context().Value(contextKeyTxControlContext).(*cc.TxControlContext)
	return ctrlCtx, ok
}

func MarshalTxExecutorContext(r *http.Request) (*cc.TxExecutorContext, bool) {
	execCtx, ok := r.Context().Value(contextKeyTxExecutorContext).(*cc.TxExecutorContext)
	return execCtx, ok
}
