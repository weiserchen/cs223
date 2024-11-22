package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	apiV1 "txchain/pkg/api/v1"
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

func main() {
	env := map[string]string{
		router.ConfigServerHost:          "localhost",
		router.ConfigServerPort:          "8300",
		router.ConfigTableEventLog:       "true",
		router.ConfigServiceUserAddr:     "localhost:8100",
		router.ConfigServiceEventAddr:    "localhost:8200",
		router.ConfigServiceEventLogAddr: "localhost:8300",
	}
	cfg := router.NewConfig(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		router.CustomEnv(env, os.Getenv),
		os.Args,
	)

	r := router.New(cfg)
	routes := getRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}

	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func getRoutes(cfg *router.Config) []router.Route {
	return []router.Route{
		{
			Verb:        http.MethodGet,
			Path:        apiV1.PathGetEventLogs,
			Handler:     apiV1.HandleGetEventLogs(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[apiV1.RequestGetEventLogs]},
		},
		{
			Verb:        http.MethodPost,
			Path:        apiV1.PathCreateEventLog,
			Handler:     apiV1.HandleCreateEventLog(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestCreateEventLog]},
		},
	}
}
