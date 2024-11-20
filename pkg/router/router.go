package router

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"txchain/pkg/database"
	"txchain/pkg/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Route struct {
	Verb        string
	Path        string
	Handler     http.Handler
	Middlewares []middleware.Middlerware
}

type Router struct {
	mux *http.ServeMux
	cfg *Config
}

func New(cfg *Config) *Router {
	return &Router{
		mux: http.NewServeMux(),
		cfg: cfg,
	}
}

func (r *Router) AddRoute(route Route) {
	pattern := fmt.Sprintf("%s %s", route.Verb, route.Path)
	r.mux.Handle(pattern, middleware.Chain(route.Handler, route.Middlewares...))
}

func (r *Router) Run() error {
	interrupts := []os.Signal{
		os.Interrupt,
	}
	ctx, cancel := signal.NotifyContext(r.cfg.Ctx, interrupts...)
	defer cancel()
	r.cfg.Ctx = ctx

	r.cfg.DBURL = r.cfg.Getenv(ConfigDatabaseURL)
	conn, err := pgxpool.New(context.Background(), r.cfg.DBURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}
	defer conn.Close()
	r.cfg.DBConn = conn

	if r.cfg.Getenv(ConfigTableUser) == "true" {
		r.cfg.DB.UserStore = database.NewTableUser(r.cfg.DBConn)
		r.cfg.Peers[ServiceUser] = r.cfg.Getenv(ConfigServiceUserAddr)
	}
	if r.cfg.Getenv(ConfigTableEvent) == "true" {
		r.cfg.DB.EventStore = database.NewTableEvent(r.cfg.DBConn)
		r.cfg.Peers[ServiceEvent] = r.cfg.Getenv(ConfigServiceEventAddr)
	}
	if r.cfg.Getenv(ConfigTableEventLog) == "true" {
		r.cfg.DB.EventLogStore = database.NewTableEventLog(r.cfg.DBConn)
		r.cfg.Peers[ServiceEventLog] = r.cfg.Getenv(ConfigServiceEventLogAddr)
	}

	host, port := r.cfg.Getenv(ConfigServerHost), r.cfg.Getenv(ConfigServerPort)
	addr := fmt.Sprintf("%s:%s", host, port)

	return http.ListenAndServe(addr, r.mux)
}
