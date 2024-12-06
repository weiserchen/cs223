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
	"txchain/pkg/database"
	"txchain/pkg/format"
	"txchain/pkg/middleware"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func EchoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middleware.UnmarshalID(r)
		format.WriteResponseStr(w, id, http.StatusOK)
	})
}

func TestRouterEngine(t *testing.T) {
	t.Parallel()

	pgc, err := database.NewContainerTableUsers(t, "17.1")
	defer func() {
		if pgc != nil {
			testcontainers.CleanupContainer(t, pgc.Container)
		}
	}()
	require.NoError(t, err)

	env := map[string]string{}
	env[ConfigTableUser] = "true"
	env[ConfigDatabaseURL] = pgc.Endpoint()

	cfg, err := NewConfig(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		CustomEnv(env, os.Getenv),
		os.Args,
	)
	require.NoError(t, err)

	r := New(cfg)
	handler := EchoHandler()
	append1 := middleware.AppendID("1")
	append2 := middleware.AppendID("2")
	append3 := middleware.AppendID("3")
	append4 := middleware.AppendID("4")
	append5 := middleware.AppendID("5")

	apiV1 := r.Prefix("/api/v1")
	apiV1.Apply(append1)
	{
		apiV1.Get("/path/1", handler)
		apiV1.Post("/path/2", handler)

		admin := apiV1.Prefix("/admin")
		admin.Apply(append3)
		{
			admin.Get("/abc", handler)
			admin.Patch("/xyz", handler)

			secret := admin.Prefix("/secret")
			secret.Apply(append5)
			{
				secret.Get("/token", handler)
			}
		}
	}

	apiV2 := r.Prefix("/api/v2")
	apiV2.Apply(append2)
	{
		apiV2.Delete("/path/1", handler)
		apiV2.Put("/path/2", handler)

		admin := apiV2.Prefix("/admin")
		admin.Apply(append4)
		{
			admin.Head("/apple", handler)
			admin.Options("/banana", handler)
		}
	}

	expectedRoutes := []Route{
		{
			method:  http.MethodGet,
			path:    "/api/v1/path/1",
			handler: append1(handler),
		},
		{
			method:  http.MethodPost,
			path:    "/api/v1/path/2",
			handler: append1(handler),
		},
		{
			method:  http.MethodGet,
			path:    "/api/v1/admin/abc",
			handler: append1(append3(handler)),
		},
		{
			method:  http.MethodPatch,
			path:    "/api/v1/admin/xyz",
			handler: append1(append3(handler)),
		},
		{
			method:  http.MethodGet,
			path:    "/api/v1/admin/secret/token",
			handler: append1(append3(append5(handler))),
		},
		{
			method:  http.MethodDelete,
			path:    "/api/v2/path/1",
			handler: append2(handler),
		},
		{
			method:  http.MethodPut,
			path:    "/api/v2/path/2",
			handler: append2(handler),
		},
		{
			method:  http.MethodHead,
			path:    "/api/v2/admin/apple",
			handler: append2(append4(handler)),
		},
		{
			method:  http.MethodOptions,
			path:    "/api/v2/admin/banana",
			handler: append2(append4(handler)),
		},
	}
	gotRoutes := r.Routes()

	sort.Slice(expectedRoutes, func(i, j int) bool {
		p1 := fmt.Sprintf("%s %s", expectedRoutes[i].method, expectedRoutes[i].path)
		p2 := fmt.Sprintf("%s %s", expectedRoutes[j].method, expectedRoutes[j].path)
		return p1 < p2
	})
	sort.Slice(gotRoutes, func(i, j int) bool {
		p1 := fmt.Sprintf("%s %s", gotRoutes[i].method, gotRoutes[i].path)
		p2 := fmt.Sprintf("%s %s", gotRoutes[j].method, gotRoutes[j].path)
		return p1 < p2
	})

	require.Equal(t, len(expectedRoutes), len(gotRoutes))
	for i := 0; i < len(expectedRoutes); i++ {
		expectedPath := fmt.Sprintf("%s %s", expectedRoutes[i].method, expectedRoutes[i].path)
		gotPath := fmt.Sprintf("%s %s", gotRoutes[i].method, gotRoutes[i].path)
		require.Equal(t, expectedPath, gotPath)
		t.Run(expectedPath, func(t *testing.T) {
			require.Equal(t, expectedRoutes[i].method, gotRoutes[i].method, expectedPath, gotPath)
			require.Equal(t, expectedRoutes[i].path, gotRoutes[i].path, expectedPath, gotPath)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(""))
			expectedRoutes[i].handler.ServeHTTP(recorder, request)
			expectedID := recorder.Body.String()

			recorder = httptest.NewRecorder()
			request = httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(""))
			gotRoutes[i].handler.ServeHTTP(recorder, request)
			gotID := recorder.Body.String()

			require.Equal(t, expectedID, gotID)
		})
	}
}
