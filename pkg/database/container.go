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

const (
	testDB         = "test"
	testDBUser     = "postgres"
	testDBPassword = "postgres"
	testDBData     = "/data/postgres"

	tableUser             = "Users"
	tableEvent            = "Events"
	tableEventLog         = "EventLogs"
	tableTxResult         = "TxResult"
	tableTxExecutor       = "TxExecutor"
	tableTxSenderClocks   = "TxSenderClocks"
	tableTxReceiverClocks = "TxReceiverClocks"

	scriptUser     = "schema/users.sql"
	scriptEvent    = "schema/events.sql"
	scriptEventLog = "schema/event_logs.sql"
	scriptTx       = "schema/tx.sql"
)

var (
	txTables       = []string{tableTxResult, tableTxExecutor, tableTxSenderClocks, tableTxReceiverClocks}
	userTables     = append(txTables, tableUser)
	eventTables    = append(txTables, tableEvent)
	eventLogTables = append(txTables, tableEventLog)

	txScripts       = []string{scriptTx}
	userScripts     = append(txScripts, scriptUser)
	eventScripts    = append(txScripts, scriptEvent)
	eventLogScripts = append(txScripts, scriptEventLog)
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

	pgc = &PgContainer{}
	container, err := tctr.GenericContainer(ctx, req)
	if err != nil {
		return pgc, err
	}
	defer func() {
		if err != nil {
			tctr.CleanupContainer(t, container)
		}
	}()

	port, err := container.MappedPort(ctx, nat.Port(pgPort))
	if err != nil {
		return pgc, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return pgc, err
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
	version string,
	scripts []string,
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

	pgc = &PgContainer{}
	pgc, err = NewPostgresContainer(t, version)
	defer func() {
		if err != nil && pgc != nil {
			tctr.CleanupContainer(t, pgc.Container)
		}
	}()
	if err != nil {
		return pgc, err
	}

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, pgc.Endpoint())
	if err != nil {
		return pgc, err
	}
	defer func() {
		conn.Close()
	}()

	scriptsInitFunc := func() (err error) {
		for _, script := range scripts {
			b, err = fs.ReadFile(filesystem, script)
			if err != nil {
				return err
			}
			if _, err = conn.Exec(ctx, string(b)); err != nil {
				return err
			}
		}
		return nil
	}

	if err = scriptsInitFunc(); err != nil {
		return pgc, err
	}

	for _, table := range tables {
		tableExistsQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s;`, table)

		var dummy int
		err = conn.QueryRow(ctx, tableExistsQuery).Scan(&dummy)
		if err != nil || dummy != 0 {
			tctr.CleanupContainer(t, pgc.Container)
			return pgc, err
		}
	}

	return pgc, nil
}

func NewContainerTableUsers(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		userScripts,
		userTables,
		TableOption{
			FS: &schema,
		},
	)
}

func NewContainerTableEvents(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		eventScripts,
		eventTables,
		TableOption{
			FS: &schema,
		},
	)
}

func NewContainerTableEventLogs(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		eventLogScripts,
		eventLogTables,
		TableOption{
			FS: &schema,
		},
	)
}

func NewContainerTablesTx(t *testing.T, version string) (*PgContainer, error) {
	return NewContainerTables(
		t,
		version,
		txScripts,
		txTables,
		TableOption{
			FS: &schema,
		},
	)
}
