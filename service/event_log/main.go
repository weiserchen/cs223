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
		router.ConfigServerPort:          "8200",
		router.ConfigTableEvent:          "true",
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
			Path:        apiV1.PathGetEvent,
			Handler:     apiV1.HandleGetEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[apiV1.RequestGetEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        apiV1.PathCreateEvent,
			Handler:     apiV1.HandleCreateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestCreateEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathUpdateEvent,
			Handler:     apiV1.HandleUpdateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestUpdateEvent]},
		},
		{
			Verb:        http.MethodDelete,
			Path:        apiV1.PathDeleteEvent,
			Handler:     apiV1.HandleDeleteEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestDeleteEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathAddEventParticipant,
			Handler:     apiV1.HandleAddEventParticipant(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestAddEventParticipant]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathRemoveEventParticipant,
			Handler:     apiV1.HandleRemoveEventParticipant(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestRemoveEventParticipant]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathTxJoinEvent,
			Handler:     apiV1.HandleTxJoinEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestTxJoinEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathTxLeaveEvent,
			Handler:     apiV1.HandleTxLeaveEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestTxLeaveEvent]},
		},
	}
}
