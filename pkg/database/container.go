package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/docker/go-connections/nat"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PgContainer struct {
	Container   tc.Container
	Host        string
	Port        nat.Port
	ExposedPort nat.Port
}

func (pgc *PgContainer) Endpoint() string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		testDBUser,
		testDBPassword,
		pgc.Host,
		pgc.Port.Port(),
		testDB,
	)
}

type PgContainerOption struct {
	Runtime     Runtime
	InitScripts []string
}

type Runtime string

var (
	RuntimeDocker Runtime = "docker"
	RuntimePodman Runtime = "podman"
)

var (
	testDB         = "test"
	testDBUser     = "postgres"
	testDBPassword = "postgres"
	testDBData     = "/data/postgres"
)

func NewPostgresContainer(version string, options ...PgContainerOption) (*PgContainer, error) {
	var option PgContainerOption
	if len(options) > 0 {
		option = options[0]
	}

	var provider tc.ProviderType
	switch option.Runtime {
	case RuntimeDocker:
		provider = tc.ProviderDocker
	case RuntimePodman:
		provider = tc.ProviderPodman
	default:
		provider = tc.ProviderDefault
	}

	ctx := context.Background()
	pgPort := "5432/tcp"

	req := tc.GenericContainerRequest{
		ProviderType: provider,
		ContainerRequest: tc.ContainerRequest{
			Image:        fmt.Sprintf("postgres:%s", version),
			ExposedPorts: []string{pgPort},
			Env: map[string]string{
				"POSTGRES_USER":     testDBUser,
				"POSTGRES_PASSWORD": testDBPassword,
				"POSTGRES_DB":       testDB,
				"PGDATA":            testDBData,
			},

			WaitingFor: wait.ForAll(
				wait.ForSQL(nat.Port(pgPort), "pgx", func(host string, port nat.Port) string {
					dbURL := fmt.Sprintf(
						"postgres://%s:%s@%s:%s/%s",
						testDBUser,
						testDBPassword,
						host,
						port.Port(),
						testDB,
					)
					log.Println(dbURL)
					return dbURL
				}).WithStartupTimeout(5 * time.Second),
			).WithDeadline(time.Second * 60),
		},
		Started: true,
	}

	container, err := tc.GenericContainer(ctx, req)
	if err != nil {
		return nil, err
	}

	port, err := container.MappedPort(ctx, nat.Port(pgPort))
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	pgc := &PgContainer{
		Container:   container,
		Host:        host,
		Port:        port,
		ExposedPort: nat.Port(pgPort),
	}

	return pgc, nil
}
