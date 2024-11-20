package database

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStore interface {
	GetUser(ctx context.Context, userID int64) (*User, error)
	GetID(ctx context.Context, userName string) (int64, error)
	GetName(ctx context.Context, userID int64) (string, error)
	GetHostEvents(ctx context.Context, userID int64) ([]int64, error)
	CreateUser(ctx context.Context, userName string, hostEvents []int64) (int64, error)
	DeleteUser(ctx context.Context, userID int64) error
	UpdateName(ctx context.Context, userID int64, newName string) error
	AddHostEvent(ctx context.Context, userID int64, eventID int64) error
	RemoveHostEvent(ctx context.Context, userID int64, eventID int64) error
}

type User struct {
	ID         int64   `json:"user_id"`
	Name       string  `json:"user_name"`
	HostEvents []int64 `json:"host_events"`
}

type TableUser struct {
	conn *pgxpool.Pool
}

var _ UserStore = (*TableUser)(nil)

func NewTableUser(conn *pgxpool.Pool) *TableUser {
	return &TableUser{
		conn: conn,
	}
}

func (table *TableUser) GetUser(ctx context.Context, userID int64) (*User, error) {
	var user User

	query := `
		SELECT user_name, host_events
		FROM Users
		WHERE user_id = $1;
	`

	user.ID = userID
	if err := table.conn.QueryRow(
		ctx,
		query,
		userID,
	).Scan(&user.Name, user.HostEvents); err != nil {
		return nil, err
	}

	return &user, nil
}

func (table *TableUser) GetID(ctx context.Context, userName string) (int64, error) {
	var id int64

	query := `
		SELECT user_id
		FROM Users
		WHERE user_name = $1;
	`

	if err := table.conn.QueryRow(
		ctx,
		query,
		userName,
	).Scan(&id); err != nil {
		return -1, err
	}

	return id, nil
}

func (table *TableUser) GetName(ctx context.Context, userID int64) (string, error) {
	var name string

	query := `
		SELECT user_name
		FROM Users
		WHERE user_id = $1;
	`

	if err := table.conn.QueryRow(
		ctx,
		query,
		userID,
	).Scan(&name); err != nil {
		return "", err
	}

	return name, nil
}

func (table *TableUser) GetHostEvents(ctx context.Context, userID int64) ([]int64, error) {
	var hostEvents []int64

	query := `
		SELECT host_events
		FROM Users
		WHERE user_id = $1;
	`

	if err := table.conn.QueryRow(
		ctx,
		query,
		userID,
	).Scan(&hostEvents); err != nil {
		return nil, err
	}

	return hostEvents, nil
}

func (table *TableUser) CreateUser(ctx context.Context, userName string, hostEvents []int64) (int64, error) {
	var userID int64

	query := `
		INSERT INTO Users (user_name, host_events)
		VALUES ($1, $2)
		RETURNING user_id;
	`

	if err := table.conn.QueryRow(
		ctx,
		query,
		userName,
		hostEvents,
	).Scan(&userID); err != nil {
		return -1, err
	}

	return userID, nil
}

func (table *TableUser) DeleteUser(ctx context.Context, userID int64) error {
	query := `
		DELETE FROM Users
		WHERE user_id = $1;
	`

	if _, err := table.conn.Exec(
		ctx,
		query,
		userID,
	); err != nil {
		return err
	}

	return nil
}

func (table *TableUser) UpdateName(ctx context.Context, userID int64, newName string) error {
	query := `
		UPDATE Users
		SET user_name = $2
		WHERE user_id = $1;
	`

	if _, err := table.conn.Exec(
		ctx,
		query,
		userID,
		newName,
	); err != nil {
		return err
	}

	return nil
}

func (table *TableUser) AddHostEvent(ctx context.Context, userID int64, eventID int64) (err error) {
	var hostEvents []int64
	var tx pgx.Tx

	tx, commit, err := BeginTx(ctx, table.conn)
	if err != nil {
		return err
	}
	defer func() {
		err = commit(err)
	}()

	readQuery := `
		SELECT host_events
		FROM Users
		WHERE user_id = $1;
	`
	if err = tx.QueryRow(
		ctx,
		readQuery,
		userID,
	).Scan(&hostEvents); err != nil {
		return err
	}

	if !slices.Contains(hostEvents, eventID) {
		hostEvents = append(hostEvents, eventID)
	}

	updateQuery := `
		UPDATE Users
		SET host_events = $2
		WHERE user_id = $1;
	`

	if _, err := tx.Exec(
		ctx,
		updateQuery,
		userID,
		hostEvents,
	); err != nil {
		return err
	}

	return nil
}

func (table *TableUser) RemoveHostEvent(ctx context.Context, userID int64, eventID int64) (err error) {
	var hostEvents []int64
	var tx pgx.Tx

	tx, commit, err := BeginTx(ctx, table.conn)
	if err != nil {
		return err
	}
	defer func() {
		err = commit(err)
	}()

	readQuery := `
		SELECT host_events
		FROM Users
		WHERE user_id = $1;
	`
	if err = tx.QueryRow(
		ctx,
		readQuery,
		userID,
	).Scan(&hostEvents); err != nil {
		return err
	}

	hostEvents = slices.DeleteFunc(hostEvents, func(id int64) bool {
		return id == eventID
	})

	updateQuery := `
		UPDATE Users
		SET host_events = $2
		WHERE user_id = $1;
	`

	if _, err := tx.Exec(
		ctx,
		updateQuery,
		userID,
		hostEvents,
	); err != nil {
		return err
	}

	return nil
}
