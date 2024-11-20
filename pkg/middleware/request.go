package middleware

import (
	"context"
	"net/http"
	"txchain/pkg/format"
)

type contextKey string

var (
	contextKeyRequestBody  contextKey = "request-body"
	contextKeyRequestQuery contextKey = "request-query"
)

type Middlerware func(next http.Handler) http.Handler

func Chain(handler http.Handler, middlewares ...Middlerware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func Noop(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func ValidateBody[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, err := format.DecodeBody[T](r)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(format.ErrJsonDecode, err), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyRequestBody, v)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ValidateQuery[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, err := format.DecodeParam[T](r)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(format.ErrJsonDecode, err), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyRequestQuery, v)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MarshalBody[T any](r *http.Request) T {
	return r.Context().Value(contextKeyRequestBody).(T)
}

func MarshalQuery[T any](r *http.Request) T {
	return r.Context().Value(contextKeyRequestQuery).(T)
}
