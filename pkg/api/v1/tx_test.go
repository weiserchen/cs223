package v1

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
	"txchain/pkg/database"

	"github.com/stretchr/testify/require"
	tctr "github.com/testcontainers/testcontainers-go"
)

func TestCalendarTxAPI(t *testing.T) {
	var serverUser, serverEvent, serverEventLog *httptest.Server
	var client *http.Client

	{
		pgc, err := database.NewContainerTableUsers(t, "17.1")
		defer func() {
			if pgc != nil {
				tctr.CleanupContainer(t, pgc.Container)
			}
		}()
		require.NoError(t, err)

		r := DefaultUserRouter(
			pgc.Endpoint(),
			DefaultUserServerAddr,
			DefaultEventServerAddr,
			DefaultEventLogServerAddr,
		)

		serverUser = NewTestServer(t, r.Routes(), DefaultUserServerAddr)
	}
	{
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

		serverEvent = NewTestServer(t, r.Routes(), DefaultEventServerAddr)
	}
	{
		pgc, err := database.NewContainerTableEventLogs(t, "17.1")
		defer func() {
			if pgc != nil {
				tctr.CleanupContainer(t, pgc.Container)
			}
		}()
		require.NoError(t, err)

		r := DefaultEventLogRouter(
			pgc.Endpoint(),
			DefaultUserServerAddr,
			DefaultEventServerAddr,
			DefaultEventLogServerAddr,
		)

		serverEventLog = NewTestServer(t, r.Routes(), DefaultEventLogServerAddr)
	}

	client = DefaultHTTPClient()

	user := struct {
		ID         int64
		Name       string
		HostEvents []int64
	}{
		Name:       "Alice",
		HostEvents: []int64{100, 200, 300},
	}

	// Create user
	reqCreateUser := &RequestCreateUser{
		UserName:   user.Name,
		HostEvents: user.HostEvents,
	}
	respCreateUser, err := PostRequestCreateUser(client, serverUser.URL, reqCreateUser)
	require.NoError(t, err)
	user.ID = respCreateUser.UserID

	event1Create := &APIEvent{
		EventName:    "Test Tx Event",
		EventInfo:    "This is a test tx event.",
		HostID:       user.ID,
		StartAt:      time.Date(2000, 12, 25, 18, 0, 0, 0, time.Local),
		EndAt:        time.Date(2000, 12, 25, 22, 0, 0, 0, time.Local),
		Location:     "Aldrich Park",
		Participants: []int64{10, 20, 30},
	}

	// Create event via tx
	reqTxCreateEvent := &RequestTxCreateEvent{
		UserID:       user.ID,
		EventName:    event1Create.EventName,
		EventInfo:    event1Create.EventInfo,
		StartAt:      event1Create.StartAt,
		EndAt:        event1Create.EndAt,
		Location:     event1Create.Location,
		Participants: event1Create.Participants,
	}
	respTxCreateEvent, err := PostRequestTxCreateEvent(client, serverUser.URL, reqTxCreateEvent)
	require.NoError(t, err)
	event1Create.EventID = respTxCreateEvent.EventID

	reqGetUser := &RequestGetUser{
		UserID: user.ID,
	}
	user.HostEvents = slices.Sorted(slices.Values(append(user.HostEvents, respTxCreateEvent.EventID)))
	respGetUser, err := GetRequestGetUser(client, serverUser.URL, reqGetUser)
	require.NoError(t, err)
	require.Equal(t, user.ID, respGetUser.UserID)
	require.Equal(t, user.Name, respGetUser.UserName)
	require.Equal(t, user.HostEvents, slices.Sorted(slices.Values(respGetUser.HostEvents)))

	reqGetEvent := &RequestGetEvent{
		EventID: event1Create.EventID,
	}
	respGetEvent, err := GetRequestGetEvent(client, serverEvent.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event1Create.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event1Create.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event1Create.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event1Create.HostID, respGetEvent.Event.HostID)
	require.Equal(t, event1Create.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event1Create.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event1Create.Location, respGetEvent.Event.Location)
	require.Equal(t, event1Create.Participants, respGetEvent.Event.Participants)

	// Join event via tx
	var joinParticipantID int64 = 40
	reqTxJoinEvent := &RequestTxJoinEvent{
		EventID:       event1Create.EventID,
		HostID:        event1Create.HostID,
		ParticipantID: joinParticipantID,
	}
	_, err = PutRequestTxJoinEvent(client, serverEvent.URL, reqTxJoinEvent)
	require.NoError(t, err)

	// Update event via tx
	event1Update := &APIEvent{
		EventID:   event1Create.EventID,
		EventName: "Updated Test Tx Event",
		EventInfo: "This is a updated test tx event.",
		HostID:    user.ID,
		StartAt:   time.Date(2000, 12, 25, 19, 0, 0, 0, time.Local),
		EndAt:     time.Date(2000, 12, 25, 21, 0, 0, 0, time.Local),
		Location:  "ARC",
	}
	reqTxUpdateEvent := &RequestTxUpdateEvent{
		UserID:    event1Update.HostID,
		EventID:   event1Update.EventID,
		EventName: event1Update.EventName,
		EventInfo: event1Update.EventInfo,
		StartAt:   event1Update.StartAt,
		EndAt:     event1Update.EndAt,
		Location:  event1Update.Location,
	}
	_, err = PutRequestTxUpdateEvent(client, serverUser.URL, reqTxUpdateEvent)
	require.NoError(t, err)

	// Leave event via tx
	var leaveParticipantID int64 = 20
	reqTxLeaveEvent := &RequestTxLeaveEvent{
		EventID:       event1Create.EventID,
		HostID:        event1Create.HostID,
		ParticipantID: leaveParticipantID,
	}
	_, err = PutRequestTxLeaveEvent(client, serverEvent.URL, reqTxLeaveEvent)
	require.NoError(t, err)

	event1Update.Participants = []int64{10, 30, 40}
	respGetEvent, err = GetRequestGetEvent(client, serverEvent.URL, reqGetEvent)
	require.NoError(t, err)
	require.Equal(t, event1Update.EventID, respGetEvent.Event.EventID)
	require.Equal(t, event1Update.EventName, respGetEvent.Event.EventName)
	require.Equal(t, event1Update.EventInfo, respGetEvent.Event.EventInfo)
	require.Equal(t, event1Update.HostID, respGetEvent.Event.HostID)
	require.Equal(t, event1Update.StartAt, respGetEvent.Event.StartAt)
	require.Equal(t, event1Update.EndAt, respGetEvent.Event.EndAt)
	require.Equal(t, event1Update.Location, respGetEvent.Event.Location)
	require.Equal(t, event1Update.Participants, slices.Sorted(slices.Values(respGetEvent.Event.Participants)))

	// Delete event via tx
	reqTxDeleteEvent := &RequestTxDeleteEvent{
		UserID:  event1Update.HostID,
		EventID: event1Update.EventID,
	}
	_, err = DeleteRequestTxDeleteEvent(client, serverUser.URL, reqTxDeleteEvent)
	require.NoError(t, err)

	_, err = GetRequestGetEvent(client, serverEvent.URL, reqGetEvent)
	require.Error(t, err)

	user.HostEvents = slices.DeleteFunc(user.HostEvents, func(id int64) bool {
		return id == event1Update.EventID
	})
	respGetUser, err = GetRequestGetUser(client, serverUser.URL, reqGetUser)
	require.NoError(t, err)
	require.Equal(t, user.ID, respGetUser.UserID)
	require.Equal(t, user.Name, respGetUser.UserName)
	require.Equal(t, user.HostEvents, slices.Sorted(slices.Values(respGetUser.HostEvents)))

	// Check event logs
	event1CreateLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    event1Create.HostID,
		EventType: string(database.EventCreate),
		Content:   database.GenLogCreateEvent(APIEventToDatabaseEvent(event1Create)),
	}

	event1JoinLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    joinParticipantID,
		EventType: string(database.EventJoin),
		Content:   database.GenLogJoinEvent(event1Create.EventID, joinParticipantID),
	}

	event1UpdateLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    event1Create.HostID,
		EventType: string(database.EventUpdate),
		Content:   database.GenLogUpdateEvent(APIEventToDatabaseEvent(event1Update)),
	}

	event1LeaveLog := &APIEventLog{
		EventID:   event1Update.EventID,
		UserID:    leaveParticipantID,
		EventType: string(database.EventLeave),
		Content:   database.GenLogLeaveEvent(event1Update.EventID, leaveParticipantID),
	}

	event1DeleteLog := &APIEventLog{
		EventID:   event1Update.EventID,
		UserID:    event1Update.HostID,
		EventType: string(database.EventDelete),
		Content:   database.GenLogDeleteEvent(event1Update.EventID),
	}

	reqGetEventLogs := &RequestGetEventLogs{
		EventID: event1Create.EventID,
	}
	respGetEventLogs, err := GetRequestGetEventLogs(client, serverEventLog.URL, reqGetEventLogs)
	require.NoError(t, err)
	require.Equal(t, 5, len(respGetEventLogs.EventLogs))

	dbEvent1CreateLog := respGetEventLogs.EventLogs[0]
	require.Equal(t, event1CreateLog.UserID, dbEvent1CreateLog.UserID)
	require.Equal(t, event1CreateLog.EventID, dbEvent1CreateLog.EventID)
	require.Equal(t, event1CreateLog.EventType, dbEvent1CreateLog.EventType)
	require.Equal(t, event1CreateLog.Content, dbEvent1CreateLog.Content)

	dbEvent1JoinLog := respGetEventLogs.EventLogs[1]
	require.Equal(t, event1JoinLog.UserID, dbEvent1JoinLog.UserID)
	require.Equal(t, event1JoinLog.EventID, dbEvent1JoinLog.EventID)
	require.Equal(t, event1JoinLog.EventType, dbEvent1JoinLog.EventType)
	require.Equal(t, event1JoinLog.Content, dbEvent1JoinLog.Content)

	dbEvent1UpdateLog := respGetEventLogs.EventLogs[2]
	require.Equal(t, event1UpdateLog.UserID, dbEvent1UpdateLog.UserID)
	require.Equal(t, event1UpdateLog.EventID, dbEvent1UpdateLog.EventID)
	require.Equal(t, event1UpdateLog.EventType, dbEvent1UpdateLog.EventType)
	require.Equal(t, event1UpdateLog.Content, dbEvent1UpdateLog.Content)

	dbEvent1LeaveLog := respGetEventLogs.EventLogs[3]
	require.Equal(t, event1LeaveLog.UserID, dbEvent1LeaveLog.UserID)
	require.Equal(t, event1LeaveLog.EventID, dbEvent1LeaveLog.EventID)
	require.Equal(t, event1LeaveLog.EventType, dbEvent1LeaveLog.EventType)
	require.Equal(t, event1LeaveLog.Content, dbEvent1LeaveLog.Content)

	dbEvent1DeleteLog := respGetEventLogs.EventLogs[4]
	require.Equal(t, event1DeleteLog.UserID, dbEvent1DeleteLog.UserID)
	require.Equal(t, event1DeleteLog.EventID, dbEvent1DeleteLog.EventID)
	require.Equal(t, event1DeleteLog.EventType, dbEvent1DeleteLog.EventType)
	require.Equal(t, event1DeleteLog.Content, dbEvent1DeleteLog.Content)
}
