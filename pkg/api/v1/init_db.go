package v1

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	db *pgxpool.Pool
}

var (
	pgInstance *postgres
	pgOnce     sync.Once
)

func NewPG(ctx context.Context, connString string) (*postgres, error) {
	pgOnce.Do(func() {
		db, err := pgxpool.New(ctx, connString)
		if err != nil {
			fmt.Errorf("unable to create connection pool: %w", err)
			return
		}

		pgInstance = &postgres{db}
	})

	return pgInstance, nil
}

func (pg *postgres) Ping(ctx context.Context) error {
	return pg.db.Ping(ctx)
}

func (pg *postgres) Close() {
	pg.db.Close()
}

func (pg *postgres) bulkInsertUsers(ctx context.Context, users []User) error {
	query := `INSERT INTO Users (user_name)
		VALUES (@userName)`
	batch := &pgx.Batch{}
	for _, user := range users {
		args := pgx.NamedArgs{
			"userName": user.Name,
		}
		batch.Queue(query, args)
	}
	results := pg.db.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(users); i++ {
		_, err := results.Exec()
		return fmt.Errorf("unable to insert row: %w", err)
	}

	return results.Close()
}

func (pg *postgres) bulkInsertEvents(ctx context.Context, events []Event) error {
	query := `INSERT INTO Events (event_name, event_info, start_at, end_at, location)
		VALUES (@eventName, @eventInfo, @startAt, @endAt, @loaction)`
	batch := &pgx.Batch{}
	for _, event := range events {
		args := pgx.NamedArgs{
			"eventName": event.EventName,
			"eventInfo": event.EventInfo,
			"startAt":   event.StartAt,
			"endAt":     event.EndAt,
			"location":  event.Location,
		}
		batch.Queue(query, args)
	}
	results := pg.db.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(events); i++ {
		_, err := results.Exec()
		return fmt.Errorf("unable to insert row: %w", err)
	}

	return results.Close()
}

type User struct {
	ID         int64
	Name       string
	HostEvents []int64
}

type Event struct {
	EventID      int64
	EventName    string
	EventInfo    string
	StartAt      time.Time
	EndAt        time.Time
	Location     string
	Participants []int64
	Host         int64
}

type EventLog struct {
	LogID     int64
	EventID   int64
	UserID    int64
	EventType string
	Update    string
	UpdatedAt time.Time
}

func createRandomUsers(initUserNum int) {
	users := []User{}
	for i := 0; i < initUserNum; i++ {
		users = append(users, User{
			Name: gofakeit.Name(),
		})
	}

}

func createRandomEvents(initEventNum int) {
	events := []Event{}
	for i := 0; i < initEventNum; i++ {
		events = append(events, Event{
			EventName: gofakeit.Sentence(3),
			EventInfo: gofakeit.Sentence(10),
			StartAt:   gofakeit.Date(),
			EndAt:     gofakeit.Date().Add(time.Hour),
			Location:  gofakeit.City(),
		})
	}

}

func (pg *postgres) getEventsWithNoHost(ctx context.Context) ([]int64, error) {
	query := `SELECT ID FROM Events WHERE Host IS NULL`
	rows, err := pg.db.Query(ctx, query)
	if err != nil {
		fmt.Errorf("unable to query events: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("unable to scan event ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (pg *postgres) getAllUsers(ctx context.Context) ([]int64, error) {
	query := `SELECT ID FROM Users`
	rows, err := pg.db.Query(ctx, query)
	if err != nil {
		fmt.Errorf("unable to query users: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("unable to scan user ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (pg *postgres) getAllEvents(ctx context.Context) ([]int64, error) {
	query := `SELECT ID FROM Events`
	rows, err := pg.db.Query(ctx, query)
	if err != nil {
		fmt.Errorf("unable to query events: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("unable to scan event ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (pg *postgres) makeHost(ctx context.Context) error {
	eventIDs, err := pg.getEventsWithNoHost(ctx)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}
	userIDs, err := pg.getAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}

	if len(userIDs) == 0 {
		return fmt.Errorf("no users available to assign as hosts")
	}

	for _, eventID := range eventIDs {
		randomUserID := userIDs[rand.Intn(len(userIDs))]

		query := `UPDATE Events SET host = $1 WHERE ID = $2`
		_, err := pg.db.Exec(ctx, query, randomUserID, eventID)
		if err != nil {
			return fmt.Errorf("failed to update event host: %w", err)
		}

		query = `SELECT host_events FROM Users WHERE ID = $1`
		var hostEvents []int64
		err = pg.db.QueryRow(ctx, query, randomUserID).Scan(&hostEvents)
		if err != nil {
			return fmt.Errorf("failed to fetch user's host events: %w", err)
		}
		hostEvents = append(hostEvents, eventID)

		query = `UPDATE Users SET host_events = $1 WHERE ID = $2`
		_, err = pg.db.Exec(ctx, query, hostEvents, randomUserID)
		if err != nil {
			return fmt.Errorf("failed to update user's host events: %w", err)
		}

		query = `SELECT event_info FROM Events WHERE ID = $1`
		var info string
		err = pg.db.QueryRow(ctx, query, eventID).Scan(&info)
		if err != nil {
			return fmt.Errorf("failed to fetch event info: %w", err)
		}

		query = `INSERT INTO EventLog (event_id, user_id, event_type, update, update_at) VALUES ($1, $2, $3, $4, $5)`
		_, err = pg.db.Exec(ctx, query, eventID, randomUserID, "Create", info, time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert event log: %w", err)
		}
	}
	return nil
}

func (pg *postgres) makeParticipants(ctx context.Context) error {
	eventIDs, err := pg.getAllEvents(ctx)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}
	userIDs, err := pg.getAllUsers(ctx)
	if err != nil {
		return fmt.Errorf("Error: %w", err)
	}

	if len(userIDs) == 0 {
		return fmt.Errorf("no users available to assign as hosts")
	}

	for _, eventID := range eventIDs {
		if rand.Intn(2) == 0 {
			continue
		}

		var hostID int64
		query := `SELECT host FROM Events WHERE ID = $1`
		err := pg.db.QueryRow(ctx, query, eventID).Scan(&hostID)
		if err != nil {
			return fmt.Errorf("failed to fetch event host: %w", err)
		}
		potentialParticipants := make([]int64, 0)
		for _, userID := range userIDs {
			if userID != hostID {
				potentialParticipants = append(potentialParticipants, userID)
			}
		}

		if len(potentialParticipants) == 0 {
			continue
		}

		numParticipants := rand.Intn(len(potentialParticipants)) + 1
		rand.Shuffle(len(potentialParticipants), func(i, j int) {
			potentialParticipants[i], potentialParticipants[j] = potentialParticipants[j], potentialParticipants[i]
		})
		participants := potentialParticipants[:numParticipants]

		query = `UPDATE Events SET participants = $1 WHERE ID = $2`
		_, err = pg.db.Exec(ctx, query, participants, eventID)
		if err != nil {
			return fmt.Errorf("failed to update event participants: %w", err)
		}

		query = `SELECT event_info FROM Events WHERE ID = $1`
		var info string
		err = pg.db.QueryRow(ctx, query, eventID).Scan(&info)
		if err != nil {
			return fmt.Errorf("failed to fetch event info: %w", err)
		}
		for _, participantID := range participants {
			query = `INSERT INTO EventLog (event_id, user_id, event_type, update, update_at) VALUES ($1, $2, $3, $4, $5)`
			_, err = pg.db.Exec(ctx, query, eventID, participantID, "Join", info, time.Now())
			if err != nil {
				return fmt.Errorf("failed to insert event log: %w", err)
			}
		}
	}
	return nil
}
