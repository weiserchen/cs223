package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisContainer(t *testing.T) {
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	tc.CleanupContainer(t, redisC)
}

func TestPostgresContainer(t *testing.T) {
	var err error
	var pgc *PgContainer
	var dummy int
	var userScript string
	var b []byte

	pgc, err = NewPostgresContainer(
		"17.1",
	)
	require.NoError(t, err)

	userScript, err = filepath.Abs("../schema/users.sql")
	require.NoError(t, err)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, pgc.Endpoint())
	require.NoError(t, err)

	b, err = os.ReadFile(userScript)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(b))
	require.NoError(t, err)

	usersTableExistsQuery := `
		SELECT COUNT(*) FROM Users;
	`
	err = pool.QueryRow(ctx, usersTableExistsQuery).Scan(&dummy)
	require.NoError(t, err)
	require.Zero(t, dummy)

	tc.CleanupContainer(t, pgc.Container)
}
