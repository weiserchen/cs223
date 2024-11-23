package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventLogStore interface {
	GetEventLogs(ctx context.Context, eventID int64) ([]*EventLog, error)
	CreateEventLog(ctx context.Context, eventID int64, userID int64, eventType EventType, event *Event) (int64, error)
}

type EventType string

var (
	EventCreate EventType = "create"
	EventUpdate EventType = "update"
	EventDelete EventType = "delete"
	EventJoin   EventType = "join"
	EventLeave  EventType = "leave"
)

var (
	ErrUnknownEventType = errors.New("unknown event type")
)

type EventLog struct {
	LogID     int64     `json:"log_id"`
	EventID   int64     `json:"event_id"`
	UserID    int64     `json:"user_id"`
	EventType EventType `json:"event_type"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"` // updated by database
}

var _ EventLogStore = (*TableEventLog)(nil)

type TableEventLog struct {
	conn *pgxpool.Pool
}

func NewTableEventLog(conn *pgxpool.Pool) *TableEventLog {
	return &TableEventLog{
		conn: conn,
	}
}

func (table *TableEventLog) CreateEventLog(
	ctx context.Context,
	eventID int64,
	userID int64,
	eventType EventType,
	event *Event,
) (int64, error) {
	var logID int64
	var err error
	var content string

	switch eventType {
	case EventCreate:
		content = GenLogCreateEvent(event)
	case EventUpdate:
		content = GenLogUpdateEvent(event)
	case EventDelete:
		content = GenLogDeleteEvent(eventID)
	case EventJoin:
		content = GenLogJoinEvent(eventID, userID)
	case EventLeave:
		content = GenLogLeaveEvent(eventID, userID)
	default:
		return -1, fmt.Errorf("%w: %v", ErrUnknownEventType, eventType)
	}

	query := `
		INSERT INTO EventLogs (
			event_id, 
			user_id,
			event_type,
			content
		)
		VALUES ($1, $2, $3, $4)
		RETURNING log_id;
	`
	if err = table.conn.QueryRow(
		ctx,
		query,
		eventID,
		userID,
		eventType,
		content,
	).Scan(&logID); err != nil {
		return -1, err
	}

	return logID, nil
}

func (table *TableEventLog) GetEventLogs(
	ctx context.Context,
	eventID int64,
) (eventLogs []*EventLog, err error) {
	var rows pgx.Rows

	query := `
		SELECT
			log_id,
			user_id,
			event_type,
			content,
			updated_at
		FROM EventLogs
		WHERE event_id = $1;
	`

	rows, err = table.conn.Query(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer func() {
		rows.Close()
		if err == nil {
			err = rows.Err()
		}
	}()

	for rows.Next() {
		var eventLog EventLog

		eventLog.EventID = eventID
		if err = rows.Scan(
			&eventLog.LogID,
			&eventLog.UserID,
			&eventLog.EventType,
			&eventLog.Content,
			&eventLog.UpdatedAt,
		); err != nil {
			return nil, err
		}

		eventLogs = append(eventLogs, &eventLog)
	}

	return eventLogs, nil
}

func GenLogCreateEvent(event *Event) string {
	return fmt.Sprintf(
		"event (%s) (%d) is created by user (%d). Info: %s. Time: [%v, %v]. Location: %s. Participants: %s.",
		event.Name,
		event.ID,
		event.HostID,
		event.Info,
		event.StartAt,
		event.EndAt,
		event.Location,
		fmt.Sprintf("%v", event.Participants),
	)
}

func GenLogUpdateEvent(event *Event) string {
	return fmt.Sprintf(
		"event (%s) (%d) is updated by user (%d). Info: %s. Time: [%v, %v]. Location: %s.",
		event.Name,
		event.ID,
		event.HostID,
		event.Info,
		event.StartAt,
		event.EndAt,
		event.Location,
	)
}

func GenLogDeleteEvent(eventID int64) string {
	return fmt.Sprintf("event (%d) is deleted", eventID)
}

func GenLogJoinEvent(eventID, userID int64) string {
	return fmt.Sprintf("user (%d) joins event (%d)", userID, eventID)
}

func GenLogLeaveEvent(eventID, userID int64) string {
	return fmt.Sprintf("user (%d) leaves event (%d)", userID, eventID)
}
