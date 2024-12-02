package middleware

import (
	"context"
	"net/http"
	"txchain/pkg/format"
)

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

func ValidateQuery[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, err := format.DecodeParam[T](r)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(format.ErrJsonDecode, err), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyRequest, v)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ValidateBody[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, err := format.DecodeBody[T](r)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(format.ErrJsonDecode, err), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyRequest, v)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
