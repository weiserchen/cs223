package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	tctr "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisContainer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	req := tctr.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisC, err := tctr.GenericContainer(ctx, tctr.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	tctr.CleanupContainer(t, redisC)
}

func TestPostgresContainer(t *testing.T) {
	t.Parallel()

	var err error
	var pgc *PgContainer
	var dummy bool

	pgc, err = NewPostgresContainer(t, "17.1")
	defer tctr.CleanupContainer(t, pgc.Container)
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, pgc.Endpoint())
	require.NoError(t, err)

	testQuery := `
		SELECT true;
	`
	err = conn.QueryRow(ctx, testQuery).Scan(&dummy)
	require.NoError(t, err)
	require.True(t, dummy)

}

func TestLocalSchemaCreation(t *testing.T) {
	t.Parallel()

	var err error
	var pgc *PgContainer

	testcases := []struct {
		name   string
		table  string
		script string
	}{
		{
			name:   "Users Table",
			table:  "Users",
			script: "./schema/users.sql",
		},
		{
			name:   "Events Table",
			table:  "Events",
			script: "./schema/events.sql",
		},
		{
			name:   "EventLogs Table",
			table:  "EventLogs",
			script: "./schema/event_logs.sql",
		},
	}

	pgc, err = NewPostgresContainer(t, "17.1")
	defer tctr.CleanupContainer(t, pgc.Container)
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, pgc.Endpoint())
	require.NoError(t, err)

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var scriptPath string
			var err error
			var dummy int
			var b []byte

			scriptPath, err = filepath.Abs(tc.script)
			require.NoError(t, err)

			b, err = os.ReadFile(scriptPath)
			require.NoError(t, err)

			_, err = conn.Exec(ctx, string(b))
			require.NoError(t, err)

			tableExistsQuery := fmt.Sprintf(
				`SELECT COUNT(*) FROM %s;`,
				tc.table,
			)

			err = conn.QueryRow(ctx, tableExistsQuery).Scan(&dummy)
			require.NoError(t, err)
			require.Zero(t, dummy)
		})
	}
}

func TestDistributedSchemaCreation(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name   string
		script string
		tables []string
	}{
		{
			name:   "Users Table",
			script: "schema/users.sql",
			tables: []string{"Users"},
		},
		{
			name:   "Events Table",
			script: "schema/events.sql",
			tables: []string{"Events"},
		},
		{
			name:   "EventLogs Table",
			script: "schema/event_logs.sql",
			tables: []string{"EventLogs"},
		},
		{
			name:   "Tx Tables",
			script: "schema/tx.sql",
			tables: []string{"TxSenderClocks", "TxReceiverClocks", "TxExecutor", "TxResult"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			var pgc *PgContainer

			pgc, err = NewContainerTables(t, "17.1", tc.script, tc.tables)
			defer func() {
				if pgc != nil {
					tctr.CleanupContainer(t, pgc.Container)
				}
			}()

			require.NoError(t, err)
		})
	}
}
