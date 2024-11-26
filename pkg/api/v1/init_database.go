package v1

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/stretchr/testify/require"
)

type User struct {
	Id         int64
	Name       string
	HostEvents []int64
}

type Event struct {
	Id           int64
	Name         string
	HostId       int64
	Participants []int64
}

func initUsers(t *testing.T, client *http.Client, serverUser *httptest.Server) []User {
	gofakeit.Seed(123)

	users := make([]User, 0, 100)
	userNames := make(map[string]struct{})
	for len(userNames) < 100 {
		userName := gofakeit.Name()
		userNames[userName] = struct{}{}
	}

	eventIDs := make([]int64, 100)
	for i := range eventIDs {
		eventIDs[i] = int64(i + 1)
	}
	rand.Shuffle(len(eventIDs), func(i, j int) { eventIDs[i], eventIDs[j] = eventIDs[j], eventIDs[i] })
	eventIndex := 0

	for userName := range userNames {
		numEvents := rand.Intn(4)
		hostEvents := []int64{}
		for j := 0; j < numEvents && eventIndex < len(eventIDs); j++ {
			hostEvents = append(hostEvents, eventIDs[eventIndex])
			eventIndex++
		}

		requestCreateUser := &RequestCreateUser{
			UserName:   userName,
			HostEvents: hostEvents,
		}
		responseCreateUser, err := PostRequestCreateUser(client, serverUser.URL, requestCreateUser)
		require.NoError(t, err, "Failed to create user %s", userName)

		user := User{
			Id:         responseCreateUser.UserID,
			Name:       userName,
			HostEvents: hostEvents,
		}
		users = append(users, user)
		t.Logf("Created user: %+v", user)
	}
	return users
}

func initEvents(t *testing.T, client *http.Client, serverEvent *httptest.Server, users []User) []Event {
	events := make([]Event, 0, 100)
	for _, user := range users {
		for _, eventID := range user.HostEvents {
			numParticipants := rand.Intn(11)
			participantSet := make(map[int64]struct{})
			for len(participantSet) < numParticipants {
				participant := users[rand.Intn(len(users))].Id
				if participant != user.Id {
					participantSet[participant] = struct{}{}
				}
			}

			participants := make([]int64, 0, len(participantSet))
			for p := range participantSet {
				participants = append(participants, p)
			}

			eventName := gofakeit.Sentence(3)
			requestCreateEvent := &RequestTxCreateEvent{
				UserID:       user.Id,
				EventName:    eventName,
				EventInfo:    gofakeit.Sentence(10),
				StartAt:      time.Now(),
				EndAt:        time.Now().Add(time.Hour),
				Location:     gofakeit.City(),
				Participants: participants,
			}
			responseCreateEvent, err := PostRequestTxCreateEvent(client, serverEvent.URL, requestCreateEvent)
			require.NoError(t, err, "Failed to create event %d for user %d", eventID, user.Id)

			event := Event{
				Id:           responseCreateEvent.EventID,
				Name:         eventName,
				HostId:       user.Id,
				Participants: participants,
			}
			events = append(events, event)
			t.Logf("Created event: %+v", event)
		}
	}
	return events
}
