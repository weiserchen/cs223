package main

import (
	"context"
	"fmt"
	"os"
	apiV1 "txchain/pkg/api/v1"
	"txchain/pkg/router"
)

func main() {
	r := getRouter()
	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func getDefaultEnv() map[string]string {
	return map[string]string{
		router.ConfigServerHost:          "localhost",
		router.ConfigServerPort:          "8100",
		router.ConfigTableUser:           "true",
		router.ConfigDatabaseURL:         "localhost:5432", // TODO: just a placeholder
		router.ConfigServiceUserAddr:     "localhost:8100",
		router.ConfigServiceEventAddr:    "localhost:8200",
		router.ConfigServiceEventLogAddr: "localhost:8300",
	}
}

func getDefaultConfig(env map[string]string) *router.Config {
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

func getRouter() *router.Engine {
	env := getDefaultEnv()
	cfg := getDefaultConfig(env)
	r := router.New(cfg)
	routes := apiV1.NewUserRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}
	return r
}
