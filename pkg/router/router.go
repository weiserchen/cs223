package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"txchain/pkg/database"
	"txchain/pkg/middleware"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RouterMode string

type Route struct {
	Method  string
	Path    string
	Handler http.Handler
}

type Router struct {
	routes      map[string]map[string]http.Handler
	groups      map[string]*Router
	middlewares []middleware.Middlerware
}

func NewRouter() *Router {
	routes := map[string]map[string]http.Handler{}
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	}
	for _, method := range methods {
		routes[method] = make(map[string]http.Handler)
	}
	return &Router{
		routes:      routes,
		groups:      map[string]*Router{},
		middlewares: []middleware.Middlerware{},
	}
}

func (r *Router) Prefix(prefix string) *Router {
	if group, ok := r.groups[prefix]; ok {
		return group
	}
	group := NewRouter()
	r.groups[prefix] = group
	return group
}

func (r *Router) ApplyMiddleware(middlewares ...middleware.Middlerware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *Router) Routes() []Route {
	routes := []Route{}

	for method, route := range r.routes {
		for path, handler := range route {
			routes = append(routes, Route{
				Method:  method,
				Path:    filepath.Clean(path),
				Handler: middleware.Chain(handler, r.middlewares...),
			})
		}
	}

	for prefix, group := range r.groups {
		for _, route := range group.Routes() {
			routes = append(routes, Route{
				Method:  route.Method,
				Path:    filepath.Clean(path.Join(prefix, route.Path)),
				Handler: middleware.Chain(route.Handler, r.middlewares...),
			})
		}
	}
	return routes
}

func (r *Router) Get(path string, handler http.Handler) {
	r.routes[http.MethodGet][path] = handler
}

func (r *Router) Post(path string, handler http.Handler) {
	r.routes[http.MethodPost][path] = handler
}

func (r *Router) Put(path string, handler http.Handler) {
	r.routes[http.MethodPut][path] = handler
}

func (r *Router) Patch(path string, handler http.Handler) {
	r.routes[http.MethodPatch][path] = handler
}

func (r *Router) Delete(path string, handler http.Handler) {
	r.routes[http.MethodDelete][path] = handler
}

func (r *Router) Head(path string, handler http.Handler) {
	r.routes[http.MethodHead][path] = handler
}

func (r *Router) Options(path string, handler http.Handler) {
	r.routes[http.MethodOptions][path] = handler
}

type Engine struct {
	mux          *http.ServeMux
	router       *Router
	customRoutes []Route
	cfg          *Config
}

func New(cfg *Config) *Engine {
	return &Engine{
		router: NewRouter(),
		cfg:    cfg,
	}
}

func (r *Engine) Prefix(prefix string) *Router {
	return r.router.Prefix(prefix)
}

func (r *Engine) ApplyMiddleware(middlewares ...middleware.Middlerware) {
	r.router.ApplyMiddleware(middlewares...)
}

func (r *Engine) Routes() []Route {
	return r.router.Routes()
}

func (r *Engine) AddRoute(route Route) {
	r.customRoutes = append(r.customRoutes, route)
}

func (r *Engine) Handler() http.Handler {
	mux := http.NewServeMux()
	routes := r.Routes()
	routes = append(routes, r.customRoutes...)
	for _, route := range routes {
		path := fmt.Sprintf("%s %s", route.Method, route.Path)
		log.Println(path)
		mux.Handle(path, route.Handler)
	}
	r.mux = mux
	return r.mux
}

func (r *Engine) Build() (err error) {
	r.cfg.Ctx = context.Background()

	r.cfg.DBURL = r.cfg.Getenv(ConfigDatabaseURL)
	conn, err := pgxpool.New(context.Background(), r.cfg.DBURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}
	r.cfg.DBConn = conn
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	if r.cfg.Getenv(ConfigTableUser) == "true" {
		r.cfg.DB.UserStore = database.NewTableUser(r.cfg.DBConn)
	}
	if r.cfg.Getenv(ConfigTableEvent) == "true" {
		r.cfg.DB.EventStore = database.NewTableEvent(r.cfg.DBConn)
	}
	if r.cfg.Getenv(ConfigTableEventLog) == "true" {
		r.cfg.DB.EventLogStore = database.NewTableEventLog(r.cfg.DBConn)
	}
	log.Println(r.cfg.Getenv(ConfigServiceUserAddr), r.cfg.Getenv(ConfigServiceEventAddr), r.cfg.Getenv(ConfigServiceEventLogAddr))
	r.cfg.Peers[ServiceUser] = "http://" + r.cfg.Getenv(ConfigServiceUserAddr)
	r.cfg.Peers[ServiceEvent] = "http://" + r.cfg.Getenv(ConfigServiceEventAddr)
	r.cfg.Peers[ServiceEventLog] = "http://" + r.cfg.Getenv(ConfigServiceEventLogAddr)
	return nil
}

func (r *Engine) Run() error {
	r.Build()

	interrupts := []os.Signal{
		os.Interrupt,
	}
	ctx, cancel := signal.NotifyContext(r.cfg.Ctx, interrupts...)
	defer cancel()
	r.cfg.Ctx = ctx

	host, port := r.cfg.Getenv(ConfigServerHost), r.cfg.Getenv(ConfigServerPort)
	addr := fmt.Sprintf("%s:%s", host, port)

	return http.ListenAndServe(addr, r.mux)
}
