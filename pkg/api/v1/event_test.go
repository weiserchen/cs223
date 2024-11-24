package v1

import (
	"slices"
	"testing"
	"time"
	"txchain/pkg/database"

	"github.com/stretchr/testify/require"
	tctr "github.com/testcontainers/testcontainers-go"
)

func TestEventAPI(t *testing.T) {
	t.Parallel()

	pgc, err := database.NewContainerTableEvents(t, "17.1")
	defer func() {
		if pgc != nil {
			tctr.CleanupContainer(t, pgc.Container)
		}
	}()
	require.NoError(t, err)

	r := DefaultEventRouter(
		pgc.Endpoint(),
		DefaultUserServerAddr,
		DefaultEventServerAddr,
		DefaultEventLogServerAddr,
	)

	client := DefaultHTTPClient()
	server := NewTestServer(t, r.Routes(), DefaultEventServerAddr)

	event := &APIEvent{
		EventName:    "Test Event",
		EventInfo:    "This is a test event.",
		HostID:       1,
		StartAt:      time.Date(2000, 12, 25, 18, 0, 0, 0, time.Local),
		EndAt:        time.Date(2000, 12, 25, 22, 0, 0, 0, time.Local),
		Location:     "Aldrich Park",
		Participants: []int64{10, 20, 30},
	}

	// Create event
	reqCreateEvent := &RequestCreateEvent{
		Event: event,
	}
	respCreateEvent, err := PostRequestCreateEvent(client, server.URL, reqCreateEvent)
	require.NoError(t, err)
	event.EventID = respCreateEvent.EventID

	// Get event
	reqGetEvent := &RequestGetEvent{
		EventID: event.EventID,
	}
	respGetEvent, err := GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event.Location, respGetEvent.Event.Location)
	require.Equal(t, event.Participants, respGetEvent.Event.Participants)

	// Add/Remove event participant
	reqAddEventParticipant := &RequestAddEventParticipant{
		EventID:       event.EventID,
		ParticipantID: 40,
	}
	_, err = PutRequestAddEventParticipant(client, server.URL, reqAddEventParticipant)
	require.NoError(t, err)

	reqRemoveEventParticipant := &RequestRemoveEventParticipant{
		EventID:       event.EventID,
		ParticipantID: 10,
	}
	_, err = PutRequestRemoveEventParticipant(client, server.URL, reqRemoveEventParticipant)
	require.NoError(t, err)
	event.Participants = []int64{20, 30, 40}

	respGetEvent, err = GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event.Location, respGetEvent.Event.Location)
	require.Equal(t, event.Participants, slices.Sorted(slices.Values(respGetEvent.Event.Participants)))

	// Duplicate add/remove event
	_, err = PutRequestAddEventParticipant(client, server.URL, reqAddEventParticipant)
	require.NoError(t, err)
	_, err = PutRequestRemoveEventParticipant(client, server.URL, reqRemoveEventParticipant)
	require.NoError(t, err)
	respGetEvent, err = GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.Participants, slices.Sorted(slices.Values(respGetEvent.Event.Participants)))

	// Update event
	event.EventName = "Updated Event"
	event.EventInfo = "The event is updated."
	event.StartAt = time.Date(2000, 12, 25, 17, 0, 0, 0, time.Local)
	event.EndAt = time.Date(2000, 12, 25, 21, 0, 0, 0, time.Local)
	event.Location = "ARC"
	reqUpdateEvent := &RequestUpdateEvent{
		Event: event,
	}
	_, err = PutRequestUpdateEvent(client, server.URL, reqUpdateEvent)
	require.NoError(t, err)

	respGetEvent, err = GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event.Location, respGetEvent.Event.Location)
	require.Equal(t, event.Participants, slices.Sorted(slices.Values(respGetEvent.Event.Participants)))

	// Delete event
	reqDeleteEvent := &RequestDeleteEvent{
		EventID: event.EventID,
	}
	_, err = DeleteRequestDeleteEvent(client, server.URL, reqDeleteEvent)
	require.NoError(t, err)

	// Get non-existing event
	_, err = GetRequestGetEvent(client, server.URL, reqGetEvent)
	require.Error(t, err)
}
