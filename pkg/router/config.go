package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"txchain/pkg/cc"
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
	TxMgr  *cc.TxManager
}

func NewConfig(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	getenv func(string) string,
	args []string,
) (cfg *Config, err error) {
	cfg = &Config{
		Ctx:    ctx,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Getenv: getenv,
		Args:   args,
		DB:     &database.DB{},
		Peers:  map[string]string{},
	}

	cfg.Ctx = context.Background()

	cfg.DBURL = cfg.Getenv(ConfigDatabaseURL)
	conn, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}
	cfg.DBConn = conn
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	if cfg.Getenv(ConfigTableUser) == "true" {
		cfg.DB.UserStore = database.NewTableUser(cfg.DBConn)
	}
	if cfg.Getenv(ConfigTableEvent) == "true" {
		cfg.DB.EventStore = database.NewTableEvent(cfg.DBConn)
	}
	if cfg.Getenv(ConfigTableEventLog) == "true" {
		cfg.DB.EventLogStore = database.NewTableEventLog(cfg.DBConn)
	}
	log.Println(cfg.Getenv(ConfigServiceUserAddr), cfg.Getenv(ConfigServiceEventAddr), cfg.Getenv(ConfigServiceEventLogAddr))
	cfg.Peers[ServiceUser] = "http://" + cfg.Getenv(ConfigServiceUserAddr)
	cfg.Peers[ServiceEvent] = "http://" + cfg.Getenv(ConfigServiceEventAddr)
	cfg.Peers[ServiceEventLog] = "http://" + cfg.Getenv(ConfigServiceEventLogAddr)

	services := []string{ServiceUser, ServiceEvent, ServiceEventLog}
	cfg.TxMgr = cc.NewTxManager(cfg.DBConn, 0, services)

	return cfg, nil
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
