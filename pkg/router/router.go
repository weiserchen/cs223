package router

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"txchain/pkg/middleware"
)

type RouterMode string

type Route struct {
	method  string
	path    string
	handler http.Handler
}

func NewRoute(method, path string, handler http.Handler) *Route {
	return &Route{
		method:  method,
		path:    path,
		handler: handler,
	}
}

func (r *Route) Apply(middlewares ...middleware.Middlerware) {
	r.handler = middleware.Chain(r.handler, middlewares...)
}

type Router struct {
	routes      map[string]map[string]*Route
	groups      map[string]*Router
	middlewares []middleware.Middlerware
}

func NewRouter() *Router {
	routes := map[string]map[string]*Route{}
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
		routes[method] = make(map[string]*Route)
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

func (r *Router) Apply(middlewares ...middleware.Middlerware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *Router) Routes() []Route {
	routes := []Route{}

	for method, paths := range r.routes {
		for path, route := range paths {
			routes = append(routes, Route{
				method:  method,
				path:    filepath.Clean(path),
				handler: middleware.Chain(route.handler, r.middlewares...),
			})
		}
	}

	for prefix, group := range r.groups {
		for _, route := range group.Routes() {
			routes = append(routes, Route{
				method:  route.method,
				path:    filepath.Clean(path.Join(prefix, route.path)),
				handler: middleware.Chain(route.handler, r.middlewares...),
			})
		}
	}
	return routes
}

func (r *Router) Get(path string, handler http.Handler) *Route {
	route := NewRoute(http.MethodGet, path, handler)
	r.routes[http.MethodGet][path] = route
	return route
}

func (r *Router) Post(path string, handler http.Handler) *Route {
	route := NewRoute(http.MethodPost, path, handler)
	r.routes[http.MethodPost][path] = route
	return route
}

func (r *Router) Put(path string, handler http.Handler) *Route {
	route := NewRoute(http.MethodPut, path, handler)
	r.routes[http.MethodPut][path] = route
	return route
}

func (r *Router) Patch(path string, handler http.Handler) *Route {
	route := NewRoute(http.MethodPatch, path, handler)
	r.routes[http.MethodPatch][path] = route
	return route
}

func (r *Router) Delete(path string, handler http.Handler) *Route {
	route := NewRoute(http.MethodDelete, path, handler)
	r.routes[http.MethodDelete][path] = route
	return route
}

func (r *Router) Head(path string, handler http.Handler) *Route {
	route := NewRoute(http.MethodHead, path, handler)
	r.routes[http.MethodHead][path] = route
	return route
}

func (r *Router) Options(path string, handler http.Handler) *Route {
	route := NewRoute(http.MethodOptions, path, handler)
	r.routes[http.MethodOptions][path] = route
	return route
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

func (r *Engine) Apply(middlewares ...middleware.Middlerware) {
	r.router.Apply(middlewares...)
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
		path := fmt.Sprintf("%s %s", route.method, route.path)
		log.Println(path)
		mux.Handle(path, route.handler)
	}
	r.mux = mux
	return r.mux
}

func (r *Engine) Run() error {
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
