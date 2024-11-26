package middleware

import (
	"context"
	"net/http"
	"txchain/pkg/format"
)

type contextKey string

const (
	contextKeyRequestBody  contextKey = "request-body"
	contextKeyRequestQuery contextKey = "request-query"
	contextKeyAppendID     contextKey = "append-id"
)

type Middlerware func(next http.Handler) http.Handler

func AppendID(id string) Middlerware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var newID string
			prevID, ok := r.Context().Value(contextKeyAppendID).(string)
			if !ok {
				newID = id
			} else {
				newID = prevID + "." + id
			}
			ctx := context.WithValue(r.Context(), contextKeyAppendID, newID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Chain(handler http.Handler, middlewares ...Middlerware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
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

func MarshalID(r *http.Request) string {
	id, ok := r.Context().Value(contextKeyAppendID).(string)
	if !ok {
		return ""
	}
	return id
}
