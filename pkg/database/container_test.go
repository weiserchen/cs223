package database

import (
	"context"
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
func TestDistributedSchemaCreation(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name    string
		scripts []string
		tables  []string
	}{
		{
			name:    "Users Table",
			scripts: userScripts,
			tables:  userTables,
		},
		{
			name:    "Events Table",
			scripts: eventScripts,
			tables:  eventTables,
		},
		{
			name:    "EventLogs Table",
			scripts: eventLogScripts,
			tables:  eventLogTables,
		},
		{
			name:    "Tx Tables",
			scripts: txScripts,
			tables:  txTables,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			var pgc *PgContainer

			pgc, err = NewContainerTables(t, "17.1", tc.scripts, tc.tables)
			defer func() {
				if pgc != nil {
					tctr.CleanupContainer(t, pgc.Container)
				}
			}()

			require.NoError(t, err)
		})
	}
}
