package v1

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"txchain/pkg/router"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

const (
	DefaultUserServerAddr     = "127.0.0.1:8100"
	DefaultEventServerAddr    = "127.0.0.1:8200"
	DefaultEventLogServerAddr = "127.0.0.1:8300"
)

func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

func DefaultEnv() map[string]string {
	return map[string]string{}
}

func DefaultConfig(env map[string]string) *router.Config {
	cfg := router.NewConfig(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		router.CustomEnv(env, os.Getenv),
		os.Args,
	)
	return cfg
}

func DefaultUserRouter(
	databaseURL string,
	serverUserAddr string,
	serverEventAddr string,
	serverEventLogAddr string,
) *router.Router {
	env := DefaultEnv()
	env[router.ConfigTableUser] = "true"
	env[router.ConfigDatabaseURL] = databaseURL
	env[router.ConfigServiceUserAddr] = serverUserAddr
	env[router.ConfigServiceEventAddr] = serverEventAddr
	env[router.ConfigServiceEventLogAddr] = serverEventLogAddr

	cfg := DefaultConfig(env)
	r := router.New(cfg)
	routes := NewUserRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}

	r.Build()
	return r
}

func DefaultEventRouter(
	databaseURL string,
	serverUserAddr string,
	serverEventAddr string,
	serverEventLogAddr string,
) *router.Router {
	env := DefaultEnv()
	env[router.ConfigTableEvent] = "true"
	env[router.ConfigDatabaseURL] = databaseURL
	env[router.ConfigServiceUserAddr] = serverUserAddr
	env[router.ConfigServiceEventAddr] = serverEventAddr
	env[router.ConfigServiceEventLogAddr] = serverEventLogAddr

	cfg := DefaultConfig(env)
	r := router.New(cfg)
	routes := NewEventRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}

	r.Build()
	return r
}

func DefaultEventLogRouter(
	databaseURL string,
	serverUserAddr string,
	serverEventAddr string,
	serverEventLogAddr string,
) *router.Router {
	env := DefaultEnv()
	env[router.ConfigTableEventLog] = "true"
	env[router.ConfigDatabaseURL] = databaseURL
	env[router.ConfigServiceUserAddr] = serverUserAddr
	env[router.ConfigServiceEventAddr] = serverEventAddr
	env[router.ConfigServiceEventLogAddr] = serverEventLogAddr

	cfg := DefaultConfig(env)
	r := router.New(cfg)
	routes := NewEventLogRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}

	r.Build()
	return r
}

func NewTestServer(t *testing.T, handler http.Handler, addr string) *httptest.Server {
	t.Helper()

	server := httptest.NewUnstartedServer(handler)
	l, err := net.Listen("tcp", addr)
	require.NoError(t, err)
	server.Listener.Close()
	server.Listener = l

	server.Start()
	return server
}
