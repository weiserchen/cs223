package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5/pgxpool"
	tctr "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PgContainer struct {
	Container   tctr.Container
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
	Runtime Runtime
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

type TableOption struct {
	FS fs.FS
}

var (
	//go:embed schema/*.sql
	schema embed.FS
)

func NewPostgresContainer(
	t *testing.T,
	version string,
	options ...PgContainerOption,
) (pgc *PgContainer, err error) {
	var option PgContainerOption
	if len(options) > 0 {
		option = options[0]
	}

	var provider tctr.ProviderType
	switch option.Runtime {
	case RuntimeDocker:
		provider = tctr.ProviderDocker
	case RuntimePodman:
		provider = tctr.ProviderPodman
	default:
		provider = tctr.ProviderDefault
	}

	ctx := context.Background()
	pgPort := "5432/tcp"

	req := tctr.GenericContainerRequest{
		ProviderType: provider,
		ContainerRequest: tctr.ContainerRequest{
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

	container, err := tctr.GenericContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tctr.CleanupContainer(t, container)
		}
	}()

	port, err := container.MappedPort(ctx, nat.Port(pgPort))
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	pgc = &PgContainer{
		Container:   container,
		Host:        host,
		Port:        port,
		ExposedPort: nat.Port(pgPort),
	}

	return pgc, nil
}

func NewContainerTables(
	t *testing.T,
	version, script string,
	tables []string,
	options ...TableOption,
) (pgc *PgContainer, err error) {
	var option TableOption
	var b []byte
	var filesystem fs.FS

	if len(options) > 0 {
		option = options[0]
	}
	if option.FS != nil {
		filesystem = option.FS
	} else {
		filesystem = os.DirFS("./")
	}

	pgc, err = NewPostgresContainer(t, version)
	defer func() {
		if err != nil && pgc != nil {
			tctr.CleanupContainer(t, pgc.Container)
		}
	}()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, pgc.Endpoint())
	if err != nil {
		return nil, err
	}
	defer func() {
		conn.Close()
	}()

	b, err = fs.ReadFile(filesystem, script)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx, string(b))
	if err != nil {
		return nil, err
	}

	for _, table := range tables {
		tableExistsQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s;`, table)

		var dummy int
		err = conn.QueryRow(ctx, tableExistsQuery).Scan(&dummy)
		if err != nil || dummy != 0 {
			tctr.CleanupContainer(t, pgc.Container)
			return nil, err
		}
	}

	return pgc, nil
}

func NewContainerTableUsers(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		"schema/users.sql",
		[]string{"Users"},
		TableOption{
			FS: &schema,
		},
	)
}

func NewContainerTableEvents(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		"schema/events.sql",
		[]string{"Events"},
		TableOption{
			FS: &schema,
		},
	)
}

func NewContainerTableEventLogs(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		"schema/event_logs.sql",
		[]string{"EventLogs"},
		TableOption{
			FS: &schema,
		},
	)
}

func NewContainerTablesTx(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		"schema/tx.sql",
		[]string{"TxSenderClocks", "TxReceiverClocks", "TxExecutor", "TxResult"},
		TableOption{
			FS: &schema,
		},
	)
}
