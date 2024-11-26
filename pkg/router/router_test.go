package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"txchain/pkg/format"
	"txchain/pkg/middleware"

	"github.com/stretchr/testify/require"
)

func EchoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middleware.MarshalID(r)
		format.WriteResponseStr(w, id, http.StatusOK)
	})
}

func TestRouterEngine(t *testing.T) {
	cfg := NewConfig(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		os.Getenv,
		os.Args,
	)

	r := New(cfg)
	handler := EchoHandler()
	append1 := middleware.AppendID("1")
	append2 := middleware.AppendID("2")
	append3 := middleware.AppendID("3")
	append4 := middleware.AppendID("3")
	append5 := middleware.AppendID("3")

	apiV1 := r.Prefix("/api/v1")
	apiV1.ApplyMiddleware(append1)
	{
		apiV1.Get("/path/1", handler)
		apiV1.Post("/path/2", handler)

		admin := apiV1.Prefix("/admin")
		admin.ApplyMiddleware(append3)
		{
			admin.Get("/abc", handler)
			admin.Patch("/xyz", handler)

			secret := admin.Prefix("/secret")
			secret.ApplyMiddleware(append5)
			{
				secret.Get("/token", handler)
			}
		}
	}

	apiV2 := r.Prefix("/api/v2")
	apiV2.ApplyMiddleware(append2)
	{
		apiV2.Delete("/path/1", handler)
		apiV2.Put("/path/2", handler)

		admin := apiV2.Prefix("/admin")
		admin.ApplyMiddleware(append4)
		{
			admin.Head("/apple", handler)
			admin.Options("/banana", handler)
		}
	}

	expectedRoutes := []Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/path/1",
			Handler: append1(handler),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/path/2",
			Handler: append1(handler),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/admin/abc",
			Handler: append1(append3(handler)),
		},
		{
			Method:  http.MethodPatch,
			Path:    "/api/v1/admin/xyz",
			Handler: append1(append3(handler)),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/admin/secret/token",
			Handler: append1(append3(append5(handler))),
		},
		{
			Method:  http.MethodDelete,
			Path:    "/api/v2/path/1",
			Handler: append2(handler),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/v2/path/2",
			Handler: append2(handler),
		},
		{
			Method:  http.MethodHead,
			Path:    "/api/v2/admin/apple",
			Handler: append2(append4(handler)),
		},
		{
			Method:  http.MethodOptions,
			Path:    "/api/v2/admin/banana",
			Handler: append2(append4(handler)),
		},
	}
	gotRoutes := r.Routes()

	sort.Slice(expectedRoutes, func(i, j int) bool {
		p1 := fmt.Sprintf("%s %s", expectedRoutes[i].Method, expectedRoutes[i].Path)
		p2 := fmt.Sprintf("%s %s", expectedRoutes[j].Method, expectedRoutes[j].Path)
		return p1 < p2
	})
	sort.Slice(gotRoutes, func(i, j int) bool {
		p1 := fmt.Sprintf("%s %s", gotRoutes[i].Method, gotRoutes[i].Path)
		p2 := fmt.Sprintf("%s %s", gotRoutes[j].Method, gotRoutes[j].Path)
		return p1 < p2
	})

	require.Equal(t, len(expectedRoutes), len(gotRoutes))
	for i := 0; i < len(expectedRoutes); i++ {
		expectedPath := fmt.Sprintf("%s %s", expectedRoutes[i].Method, expectedRoutes[i].Path)
		gotPath := fmt.Sprintf("%s %s", gotRoutes[i].Method, gotRoutes[i].Path)
		require.Equal(t, expectedPath, gotPath)
		t.Run(expectedPath, func(t *testing.T) {
			require.Equal(t, expectedRoutes[i].Method, gotRoutes[i].Method, expectedPath, gotPath)
			require.Equal(t, expectedRoutes[i].Path, gotRoutes[i].Path, expectedPath, gotPath)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(""))
			expectedRoutes[i].Handler.ServeHTTP(recorder, request)
			expectedID := recorder.Body.String()

			recorder = httptest.NewRecorder()
			request = httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(""))
			gotRoutes[i].Handler.ServeHTTP(recorder, request)
			gotID := recorder.Body.String()

			require.Equal(t, expectedID, gotID)
		})
	}
}
