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
		router.ConfigServerPort:          "8100",
		router.ConfigTableUser:           "true",
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
	routes := getUserRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}

	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func getUserRoutes(cfg *router.Config) []router.Route {
	return []router.Route{
		{
			Verb:        http.MethodGet,
			Path:        apiV1.PathGetUser,
			Handler:     apiV1.HandleGetUser(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[apiV1.RequestCreateUser]},
		},
		{
			Verb:        http.MethodGet,
			Path:        apiV1.PathGetUserID,
			Handler:     apiV1.HandleGetUserID(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[apiV1.RequestGetUserID]},
		},
		{
			Verb:        http.MethodGet,
			Path:        apiV1.PathGetUserName,
			Handler:     apiV1.HandleGetUserName(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[apiV1.RequestGetUserName]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathUpdateUserName,
			Handler:     apiV1.HandleUpdateUserName(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestUpdateUserName]},
		},
		{
			Verb:        http.MethodGet,
			Path:        apiV1.PathGetUserHostEvents,
			Handler:     apiV1.HandleGetUserHostEvents(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[apiV1.RequestGetUserHostEvents]},
		},
		{
			Verb:        http.MethodPost,
			Path:        apiV1.PathCreateUser,
			Handler:     apiV1.HandleCreateUser(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestCreateUser]},
		},
		{
			Verb:        http.MethodDelete,
			Path:        apiV1.PathDeleteUser,
			Handler:     apiV1.HandleDeleteUser(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestDeleteUser]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathAddUserHostEvent,
			Handler:     apiV1.HandleAddUserHostEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestAddUserHostEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        apiV1.PathRemoveUserHostEvent,
			Handler:     apiV1.HandleRemoveUserHostEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestRemoveUserHostEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        apiV1.PathTxCreateEvent,
			Handler:     apiV1.HandleTxCreateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestTxCreateEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        apiV1.PathTxUpdateEvent,
			Handler:     apiV1.HandleTxUpdateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestTxUpdateEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        apiV1.PathTxDeleteEvent,
			Handler:     apiV1.HandleTxDeleteEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[apiV1.RequestTxDeleteEvent]},
		},
	}
}
