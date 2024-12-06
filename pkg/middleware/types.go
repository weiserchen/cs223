package middleware

import (
	"net/http"
)

type contextKey string

const (
	contextKeyRequest  contextKey = "request"
	contextKeyAppendID contextKey = "append-id"
)

const (
	headerTxStageContext       = "X-Tx-Stage-Context"
	headerTxControlContext     = "X-Tx-Control-Context"
	headerTxExecutorContext    = "X-Tx-Executor-Context"
	headerTxLoggerID           = "X-Tx-Logger-ID"
	headerTxSerializationLevek = "X-Tx-Serialization-Level"
)

type SerializationLevel string

const (
	SerializationLevelNone           = "none"
	SerializationLevelOriginOrdering = "origin-ordering"
)

const (
	DefaultLoggerID = "default"

	DefSerializationLevel = "none"
)

type Middlerware func(next http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middlerware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func UnmarshalRequest[T any](r *http.Request) T {
	return r.Context().Value(contextKeyRequest).(T)
}

func UnmarshalID(r *http.Request) string {
	id, ok := r.Context().Value(contextKeyAppendID).(string)
	if !ok {
		return ""
	}
	return id
}
