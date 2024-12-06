package v1

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"txchain/pkg/router"

	"github.com/brianvoe/gofakeit"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func createSchedule(cfg *router.Config, scheduleLen int, proportion []int) ([]string, error) {
	eventCount, err := pgInstance.getEventCount(cfg.Ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to query events: %w", err)
	}
	participantCount, err := pgInstance.getParticipantCount(cfg.Ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to query participants: %w", err)
	}
	var schedule []string
	total := 0
	for i := range len(proportion) {
		total += proportion[i]
	}
	for len(schedule) < scheduleLen {
		random := rand.Intn(total) + 1
		if random < proportion[0] {
			schedule = append(schedule, "Create")
			eventCount++
		} else if random < proportion[0]+proportion[1] {
			schedule = append(schedule, "Update")
		} else if random < proportion[0]+proportion[1]+proportion[2] {
			if eventCount > 0 {
				schedule = append(schedule, "Delete")
				eventCount--
			}
		} else if random < proportion[0]+proportion[1]+proportion[2]+proportion[3] {
			schedule = append(schedule, "Join")
		} else {
			if participantCount > 0 {
				schedule = append(schedule, "Leave")
			}
		}
	}
	return schedule, nil
}

func (pg *postgres) getEventCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM Events`
	var count int
	err := pg.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("unable to query events: %w", err)
	}
	return count, nil
}

func (pg *postgres) getParticipantCount(ctx context.Context) (int, error) {
	query := `SELECT SUM(array_length(participants, 1)) FROM Events WHERE participants IS NOT NULL;`
	var participantCount int
	err := pg.db.QueryRow(ctx, query).Scan(&participantCount)
	if err != nil {
		return 0, fmt.Errorf("unable to query total participants: %w", err)
	}
	return participantCount, nil
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

func getRandomUser() (int64, error) {
	ctx := context.Background()
	IDs, err := pgInstance.getAllUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("error fetching user IDs: %w", err)
	}
	if len(IDs) == 0 {
		return 0, fmt.Errorf("no users found")
	}
	id := IDs[rand.Intn(len(IDs))]
	return id, nil
}

func getRandomUsers(num int, userID int64) ([]int64, error) {
	ctx := context.Background()
	IDs, err := pgInstance.getAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching user IDs: %w", err)
	}
	if len(IDs) == 0 {
		return nil, fmt.Errorf("no users found")
	}
	var randomIDs []int64
	for i := 0; i < num; i++ {
		if IDs[rand.Intn(len(IDs))] != userID {
			randomIDs = append(randomIDs, IDs[rand.Intn(len(IDs))])
		}
	}
	return randomIDs, err
}

func getRandomEvent() (int64, error) {
	ctx := context.Background()
	IDs, err := pgInstance.getAllEvents(ctx)
	if err != nil {
		return 0, fmt.Errorf("error fetching events IDs: %w", err)
	}
	if len(IDs) == 0 {
		return 0, fmt.Errorf("no events found")
	}
	id := IDs[rand.Intn(len(IDs))]
	return id, nil
}

func (pg *postgres) getEventHost(ctx context.Context, eventID int64) (int64, error) {
	query := `SELECT host FROM Events WHERE ID = $1`
	var hostID int64
	err := pg.db.QueryRow(ctx, query, eventID).Scan(&hostID)
	if err != nil {
		return 0, fmt.Errorf("Error fetching host ID: %w", err)
	}
	return hostID, nil
}

func (pg *postgres) getParticipants(ctx context.Context, eventID int64) ([]int64, error) {
	query := `SELECT participants FROM Events WHERE ID = $1`
	var participants []int64
	err := pg.db.QueryRow(ctx, query, eventID).Scan(&participants)
	if err != nil {
		return nil, fmt.Errorf("Error fetching participants ID: %w", err)
	}
	return participants, nil
}

func randomJoin(eventID int64) (int64, error) {
	ctx := context.Background()
	participants, err := pgInstance.getParticipants(ctx, eventID)
	if err != nil {
		return 0, fmt.Errorf("Error fetching particpants ID: %w", err)
	}
	users, err := pgInstance.getAllUsers(ctx)
	pMap := make(map[int64]struct{})
	for _, p := range participants {
		pMap[p] = struct{}{}
	}
	var result []int64
	for _, u := range users {
		if _, found := pMap[u]; !found {
			result = append(result, u)
		}
	}
	return result[rand.Intn(len(result))], nil
}

func randomLeave(eventID int64) (int64, error) {
	ctx := context.Background()
	participants, err := pgInstance.getParticipants(ctx, eventID)
	if err != nil {
		return 0, fmt.Errorf("Error fetching particpants ID: %w", err)
	}
	return participants[rand.Intn(len(participants))], nil
}

func schedule(t *testing.T, schedule []string) {
	var serverUser, serverEvent *httptest.Server
	var client *http.Client
	ctx := context.Background()
	client = DefaultHTTPClient()
	for i := 0; i < len(schedule); i++ {
		if schedule[i] == "Create" {
			userID, err := getRandomUser()
			if err != nil {
				fmt.Errorf("no users found")
			}
			participantsIDs, err := getRandomUsers(3, userID)
			eventCreate := &APIEvent{
				EventName:    gofakeit.Sentence(3),
				EventInfo:    gofakeit.Sentence(10),
				HostID:       userID,
				StartAt:      gofakeit.Date(),
				EndAt:        gofakeit.Date().Add(12 * time.Hour),
				Location:     gofakeit.City(),
				Participants: participantsIDs,
			}
			reqTxCreateEvent := &RequestTxCreateEvent{
				UserID:       userID,
				EventName:    eventCreate.EventName,
				EventInfo:    eventCreate.EventInfo,
				StartAt:      eventCreate.StartAt,
				EndAt:        eventCreate.EndAt,
				Location:     eventCreate.Location,
				Participants: eventCreate.Participants,
			}
			respTxCreateEvent, err := PostRequestTxCreateEvent(client, serverUser.URL, reqTxCreateEvent)
			require.NoError(t, err)
			eventCreate.EventID = respTxCreateEvent.EventID
		} else if schedule[i] == "Update" {
			eventID, err := getRandomEvent()
			hostID, err := pgInstance.getEventHost(ctx, eventID)
			eventUpdate := &APIEvent{
				EventID:   eventID,
				EventName: gofakeit.Sentence(3),
				EventInfo: gofakeit.Sentence(10),
				HostID:    hostID,
				StartAt:   gofakeit.Date(),
				EndAt:     gofakeit.Date().Add(12 * time.Hour),
				Location:  gofakeit.City(),
			}
			reqTxUpdateEvent := &RequestTxUpdateEvent{
				UserID:    eventUpdate.HostID,
				EventID:   eventUpdate.EventID,
				EventName: eventUpdate.EventName,
				EventInfo: eventUpdate.EventInfo,
				StartAt:   eventUpdate.StartAt,
				EndAt:     eventUpdate.EndAt,
				Location:  eventUpdate.Location,
			}
			_, err = PutRequestTxUpdateEvent(client, serverUser.URL, reqTxUpdateEvent)
			require.NoError(t, err)
		} else if schedule[i] == "Delete" {
			eventID, err := getRandomEvent()
			hostID, err := pgInstance.getEventHost(ctx, eventID)
			eventUpdate := &APIEvent{
				EventID: eventID,
				HostID:  hostID,
			}
			reqTxDeleteEvent := &RequestTxDeleteEvent{
				UserID:  eventUpdate.HostID,
				EventID: eventUpdate.EventID,
			}
			_, err = DeleteRequestTxDeleteEvent(client, serverUser.URL, reqTxDeleteEvent)
			require.NoError(t, err)
		} else if schedule[i] == "Join" {
			eventID, err := getRandomEvent()
			hostID, err := pgInstance.getEventHost(ctx, eventID)
			joinID, err := randomJoin(eventID)
			reqTxJoinEvent := &RequestTxJoinEvent{
				EventID:       eventID,
				HostID:        hostID,
				ParticipantID: joinID,
			}
			_, err = PutRequestTxJoinEvent(client, serverEvent.URL, reqTxJoinEvent)
			require.NoError(t, err)
		} else if schedule[i] == "Leave" {
			eventID, err := getRandomEvent()
			hostID, err := pgInstance.getEventHost(ctx, eventID)
			leaveID, err := randomLeave(eventID)
			reqTxLeaveEvent := &RequestTxLeaveEvent{
				EventID:       eventID,
				HostID:        hostID,
				ParticipantID: leaveID,
			}
			_, err = PutRequestTxLeaveEvent(client, serverEvent.URL, reqTxLeaveEvent)
			require.NoError(t, err)
		}
	}
}
