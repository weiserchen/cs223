package router

import (
	"context"
	"errors"
	"io"
	"txchain/pkg/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDatabaseConnection = errors.New("unable to connect to database")
)

const (
	ServiceUser     = "User"
	ServiceEvent    = "Event"
	ServiceEventLog = "EventLog"
)

const (
	ConfigServerHost          = "SERVER_HOST"
	ConfigServerPort          = "SERVER_PORT"
	ConfigTableUser           = "USER_TABLE"
	ConfigTableEvent          = "EVENT_TABLE"
	ConfigTableEventLog       = "EVENT_LOG_TABLE"
	ConfigServiceUserAddr     = "USER_SERVICE"
	ConfigServiceEventAddr    = "EVENT_SERVICE"
	ConfigServiceEventLogAddr = "EVENT_LOG_SERVICE"
	ConfigDatabaseURL         = "DATABASE_URL"
)

type Config struct {
	Ctx    context.Context
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Getenv func(string) string
	Args   []string
	DBURL  string
	DBConn *pgxpool.Pool
	DB     *database.DB
	Peers  map[string]string
}

func NewConfig(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	getenv func(string) string,
	args []string,
) *Config {
	return &Config{
		Ctx:    ctx,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Getenv: getenv,
		Args:   args,
		DB:     &database.DB{},
		Peers:  map[string]string{},
	}
}

func CustomEnv(env map[string]string, next func(string) string) func(string) string {
	return func(key string) string {
		if value, ok := env[key]; ok {
			return value
		}
		if next != nil {
			return next(key)
		}
		return ""
	}
}
