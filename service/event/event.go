package main

import (
	"context"
	"fmt"
	"os"
	apiV1 "txchain/pkg/api/v1"
	"txchain/pkg/router"
)

func main() {
	r, err := getRouter()
	if err != nil {
		os.Exit(1)
	}
	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func getDefaultEnv() map[string]string {
	return map[string]string{
		router.ConfigServerHost:          "localhost",
		router.ConfigServerPort:          "8200",
		router.ConfigTableEvent:          "true",
		router.ConfigDatabaseURL:         "localhost:5432", // TODO: just a placeholder
		router.ConfigServiceUserAddr:     "localhost:8100",
		router.ConfigServiceEventAddr:    "localhost:8200",
		router.ConfigServiceEventLogAddr: "localhost:8300",
	}
}

func getDefaultConfig(env map[string]string) (*router.Config, error) {
	return router.NewConfig(
		context.Background(),
		os.Stdin,
		os.Stdout,
		os.Stderr,
		router.CustomEnv(env, os.Getenv),
		os.Args,
	)
}

func getRouter() (*router.Engine, error) {
	env := getDefaultEnv()
	cfg, err := getDefaultConfig(env)
	if err != nil {
		return nil, err
	}
	r := router.New(cfg)
	routes := apiV1.NewEventRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}
	return r, nil
}
