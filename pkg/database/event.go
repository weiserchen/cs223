package database

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventStore interface {
	GetEvent(ctx context.Context, eventID int64) (*Event, error)
	CreateEvent(ctx context.Context, event *Event) (int64, error)
	UpdateEvent(ctx context.Context, event *Event) error
	DeleteEvent(ctx context.Context, eventID int64) error
	AddParticipant(ctx context.Context, eventID, participantID int64) error
	RemoveParticipant(ctx context.Context, eventID, userID int64) error
}

type Event struct {
	ID           int64     `json:"event_id"`
	Name         string    `json:"event_name"`
	Info         string    `json:"event_info"`
	HostID       int64     `json:"host_id"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	Location     string    `json:"location"`
	Participants []int64   `json:"participants"`
}

type TableEvent struct {
	conn *pgxpool.Pool
}

var _ EventStore = (*TableEvent)(nil)

func NewTableEvent(conn *pgxpool.Pool) *TableEvent {
	return &TableEvent{
		conn: conn,
	}
}

func (table *TableEvent) GetEvent(ctx context.Context, eventID int64) (*Event, error) {
	var event Event

	query := `
		SELECT 
			event_name,
			event_info,
			host_id,
			start_at,
			end_at,
			location,
			participants
		FROM Events
		WHERE event_id = $1;
	`

	event.ID = eventID
	if err := table.conn.QueryRow(
		ctx,
		query,
		eventID,
	).Scan(
		&event.Name,
		&event.Info,
		&event.HostID,
		&event.StartAt,
		&event.EndAt,
		&event.Location,
		&event.Participants,
	); err != nil {
		return nil, err
	}

	return &event, nil
}

func (table *TableEvent) CreateEvent(ctx context.Context, event *Event) (int64, error) {
	var eventID int64

	query := `
		INSERT INTO Events (
			event_name,
			event_info,
			host_id,
			start_at,
			end_at,
			location,
			participants
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING event_id;
	`

	if err := table.conn.QueryRow(
		ctx,
		query,
		event.Name,
		event.Info,
		event.HostID,
		event.StartAt,
		event.EndAt,
		event.Location,
		event.Participants,
	).Scan(&eventID); err != nil {
		return -1, err
	}

	return eventID, nil
}

func (table *TableEvent) UpdateEvent(ctx context.Context, updateEvent *Event) (err error) {
	var oldEvent, newEvent Event
	var tx pgx.Tx

	tx, commit, err := BeginTx(ctx, table.conn)
	if err != nil {
		return err
	}
	defer func() {
		err = commit(err)
	}()

	oldEvent.ID = updateEvent.ID
	readQuery := `
		SELECT
			event_name,
			event_info,
			start_at,
			end_at,
			location
		FROM Events
		WHERE event_id = $1;
	`
	if err = tx.QueryRow(
		ctx,
		readQuery,
		oldEvent.ID,
	).Scan(
		&oldEvent.Name,
		&oldEvent.Info,
		&oldEvent.StartAt,
		&oldEvent.EndAt,
		&oldEvent.Location,
	); err != nil {
		return err
	}

	// copy data from old event to new event
	newEvent.ID = oldEvent.ID
	newEvent.Name = oldEvent.Name
	newEvent.Info = oldEvent.Info
	newEvent.StartAt = oldEvent.StartAt
	newEvent.EndAt = oldEvent.EndAt
	newEvent.Location = oldEvent.Location

	// merge new update
	if updateEvent.Name != "" {
		newEvent.Name = updateEvent.Name
	}
	if updateEvent.Info != "" {
		newEvent.Info = updateEvent.Info
	}
	if !updateEvent.StartAt.IsZero() {
		newEvent.StartAt = updateEvent.StartAt
	}
	if !updateEvent.EndAt.IsZero() {
		newEvent.EndAt = updateEvent.EndAt
	}
	if updateEvent.Location != "" {
		newEvent.Location = updateEvent.Location
	}

	updateQuery := `
		UPDATE Events
		SET
			event_name = $2,
			event_info = $3,
			start_at = $4,
			end_at = $5,
			location = $6
		WHERE event_id = $1;
	`

	if _, err := tx.Exec(
		ctx,
		updateQuery,
		newEvent.ID,
		newEvent.Name,
		newEvent.Info,
		newEvent.StartAt,
		newEvent.EndAt,
		newEvent.Location,
	); err != nil {
		return err
	}

	return nil
}

func (table *TableEvent) DeleteEvent(ctx context.Context, eventID int64) error {
	query := `
		DELETE FROM Events
		WHERE event_id = $1;
	`

	if _, err := table.conn.Exec(
		ctx,
		query,
		eventID,
	); err != nil {
		return err
	}

	return nil
}

func (table *TableEvent) AddParticipant(ctx context.Context, eventID, participantID int64) (err error) {
	var participants []int64

	tx, commit, err := BeginTx(ctx, table.conn)
	if err != nil {
		return err
	}
	defer func() {
		err = commit(err)
	}()

	readQuery := `
		SELECT participants
		FROM Events
		WHERE event_id = $1;
	`
	if err = tx.QueryRow(
		ctx,
		readQuery,
		eventID,
	).Scan(
		&participants,
	); err != nil {
		return err
	}

	if !slices.Contains(participants, participantID) {
		participants = append(participants, participantID)
	}

	updateQuery := `
		UPDATE Events
		SET participants = $2
		WHERE event_id = $1;
	`

	if _, err := tx.Exec(
		ctx,
		updateQuery,
		eventID,
		participants,
	); err != nil {
		return err
	}

	return nil
}

func (table *TableEvent) RemoveParticipant(ctx context.Context, eventID, participantID int64) (err error) {
	var participants []int64

	tx, commit, err := BeginTx(ctx, table.conn)
	if err != nil {
		return err
	}
	defer func() {
		err = commit(err)
	}()

	readQuery := `
		SELECT participants
		FROM Events
		WHERE event_id = $1;
	`
	if err = tx.QueryRow(
		ctx,
		readQuery,
		eventID,
	).Scan(
		&participants,
	); err != nil {
		return err
	}

	participants = slices.DeleteFunc(participants, func(id int64) bool {
		return id == participantID
	})

	updateQuery := `
		UPDATE Events
		SET participants = $2
		WHERE event_id = $1;
	`

	if _, err := tx.Exec(
		ctx,
		updateQuery,
		eventID,
		participants,
	); err != nil {
		return err
	}

	return nil
}
