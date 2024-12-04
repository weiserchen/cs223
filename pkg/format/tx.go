package format

import (
	"context"
	"errors"
)

var (
	ErrNoTraceCtx = errors.New("no trace context")
)

type contextKey int

const (
	contextKeyTraceContext contextKey = iota
)

type TraceContext struct {
	values map[any]any
}

func NewTraceContext() *TraceContext {
	return &TraceContext{
		values: map[any]any{},
	}
}

func GetTraceContext(ctx context.Context) (*TraceContext, bool) {
	traceCtx, ok := ctx.Value(contextKeyTraceContext).(*TraceContext)
	return traceCtx, ok
}

func SetTraceContext(ctx context.Context, traceCtx *TraceContext) context.Context {
	return context.WithValue(ctx, contextKeyTraceContext, traceCtx)
}

func InsertTraceContext(ctx context.Context) context.Context {
	traceCtx := NewTraceContext()
	return context.WithValue(ctx, contextKeyTraceContext, traceCtx)
}

func (tc *TraceContext) Get(key any) (any, bool) {
	value, ok := tc.values[key]
	return value, ok
}

func (tc *TraceContext) Set(key, value any) {
	tc.values[key] = value
}
