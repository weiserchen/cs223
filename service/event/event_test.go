package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
	apiV1 "txchain/pkg/api/v1"
	"txchain/pkg/database"
	"txchain/pkg/router"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	tctr "github.com/testcontainers/testcontainers-go"
)

func defaultClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}
func TestEventAPI(t *testing.T) {
	t.Parallel()

	pgc, err := database.NewContainerTableEvents(t, "17.1")
	defer func() {
		if pgc != nil {
			tctr.CleanupContainer(t, pgc.Container)
		}
	}()
	require.NoError(t, err)

	env := getDefaultEnv()
	env[router.ConfigDatabaseURL] = pgc.Endpoint()
	cfg := getDefaultConfig(env)
	r := router.New(cfg)
	routes := apiV1.NewEventRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}
	r.Build()

	client := defaultClient()
	server := httptest.NewServer(r.Routes())

	event := &apiV1.APIEvent{
		EventName:    "Test Event",
		EventInfo:    "This is a test event.",
		HostID:       1,
		StartAt:      time.Date(2000, 12, 25, 18, 0, 0, 0, time.Local),
		EndAt:        time.Date(2000, 12, 25, 22, 0, 0, 0, time.Local),
		Location:     "Aldrich Park",
		Participants: []int64{10, 20, 30},
	}

	// Create event
	reqCreateEvent := &apiV1.RequestCreateEvent{
		Event: event,
	}
	respCreateEvent, err := apiV1.PostRequestCreateEvent(client, server.URL, reqCreateEvent)
	require.NoError(t, err)
	event.EventID = respCreateEvent.EventID

	// Get event
	reqGetEvent := &apiV1.RequestGetEvent{
		EventID: event.EventID,
	}
	respGetEvent, err := apiV1.GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event.Location, respGetEvent.Event.Location)
	require.Equal(t, event.Participants, respGetEvent.Event.Participants)

	// Add/Remove event participant
	reqAddEventParticipant := &apiV1.RequestAddEventParticipant{
		EventID:       event.EventID,
		ParticipantID: 40,
	}
	_, err = apiV1.PutRequestAddEventParticipant(client, server.URL, reqAddEventParticipant)
	require.NoError(t, err)

	reqRemoveEventParticipant := &apiV1.RequestRemoveEventParticipant{
		EventID:       event.EventID,
		ParticipantID: 10,
	}
	_, err = apiV1.PutRequestRemoveEventParticipant(client, server.URL, reqRemoveEventParticipant)
	require.NoError(t, err)
	event.Participants = []int64{20, 30, 40}

	respGetEvent, err = apiV1.GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event.Location, respGetEvent.Event.Location)
	require.Equal(t, event.Participants, slices.Sorted(slices.Values(respGetEvent.Event.Participants)))

	// Duplicate add/remove event
	_, err = apiV1.PutRequestAddEventParticipant(client, server.URL, reqAddEventParticipant)
	require.NoError(t, err)
	_, err = apiV1.PutRequestRemoveEventParticipant(client, server.URL, reqRemoveEventParticipant)
	require.NoError(t, err)
	respGetEvent, err = apiV1.GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.Participants, slices.Sorted(slices.Values(respGetEvent.Event.Participants)))

	// Update event
	event.EventName = "Updated Event"
	event.EventInfo = "The event is updated."
	event.StartAt = time.Date(2000, 12, 25, 17, 0, 0, 0, time.Local)
	event.EndAt = time.Date(2000, 12, 25, 21, 0, 0, 0, time.Local)
	event.Location = "ARC"
	reqUpdateEvent := &apiV1.RequestUpdateEvent{
		Event: event,
	}
	_, err = apiV1.PutRequestUpdateEvent(client, server.URL, reqUpdateEvent)
	require.NoError(t, err)

	respGetEvent, err = apiV1.GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event.Location, respGetEvent.Event.Location)
	require.Equal(t, event.Participants, slices.Sorted(slices.Values(respGetEvent.Event.Participants)))

	// Delete event
	reqDeleteEvent := &apiV1.RequestDeleteEvent{
		EventID: event.EventID,
	}
	_, err = apiV1.DeleteRequestDeleteEvent(client, server.URL, reqDeleteEvent)
	require.NoError(t, err)

	// Get non-existing event
	_, err = apiV1.GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.Error(t, err)
}
