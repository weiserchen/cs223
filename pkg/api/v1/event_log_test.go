package v1

import (
	"log"
	"testing"
	"time"
	"txchain/pkg/database"

	"github.com/stretchr/testify/require"
	tctr "github.com/testcontainers/testcontainers-go"
)

func TestEventLogAPI(t *testing.T) {
	t.Parallel()

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

	client := DefaultHTTPClient()
	server := NewTestServer(t, r.Handler(), DefaultEventLogServerAddr)

	log.Println(server.URL)

	var joinUserID int64 = 40
	var leaveUserID int64 = 20

	event1Create := &APIEvent{
		EventID:      1,
		EventName:    "Test Event",
		EventInfo:    "This is a test event.",
		HostID:       1,
		StartAt:      time.Date(2000, 12, 25, 18, 0, 0, 0, time.Local),
		EndAt:        time.Date(2000, 12, 25, 22, 0, 0, 0, time.Local),
		Location:     "Aldrich Park",
		Participants: []int64{10, 20, 30},
	}

	event1Update := &APIEvent{
		EventID:   1,
		EventName: "Test Event Update",
		EventInfo: "This is a updated test event.",
		StartAt:   time.Date(2000, 12, 25, 19, 0, 0, 0, time.Local),
		EndAt:     time.Date(2000, 12, 25, 21, 0, 0, 0, time.Local),
		Location:  "ARC",
	}

	event1CreateLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    event1Create.HostID,
		EventType: string(database.EventCreate),
		Content:   database.GenLogCreateEvent(APIEventToDatabaseEvent(event1Create)),
	}

	event1JoinLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    joinUserID,
		EventType: string(database.EventJoin),
		Content:   database.GenLogJoinEvent(event1Create.EventID, joinUserID),
	}

	event1LeaveLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    leaveUserID,
		EventType: string(database.EventLeave),
		Content:   database.GenLogLeaveEvent(event1Create.EventID, leaveUserID),
	}

	event1UpdateLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    event1Create.HostID,
		EventType: string(database.EventUpdate),
		Content:   database.GenLogUpdateEvent(APIEventToDatabaseEvent(event1Update)),
	}

	event1DeleteLog := &APIEventLog{
		EventID:   event1Create.EventID,
		UserID:    event1Create.HostID,
		EventType: string(database.EventDelete),
		Content:   database.GenLogDeleteEvent(event1Create.EventID),
	}

	// Create event
	reqCreateEventLog := &RequestCreateEventLog{
		UserID:    event1Create.HostID,
		EventID:   event1Create.EventID,
		EventType: string(database.EventCreate),
		Event:     event1Create,
	}
	respCreateEventLog, err := PostRequestCreateEventLog(client, server.URL, reqCreateEventLog)
	require.NoError(t, err)
	event1CreateLog.LogID = respCreateEventLog.LogID

	// Join event
	reqJoinEventLog := &RequestCreateEventLog{
		UserID:    joinUserID,
		EventID:   event1Create.EventID,
		EventType: string(database.EventJoin),
		Event:     nil,
	}
	respJoinEventLog, err := PostRequestCreateEventLog(client, server.URL, reqJoinEventLog)
	require.NoError(t, err)
	event1JoinLog.LogID = respJoinEventLog.LogID

	// Update event
	reqUpdateEventLog := &RequestCreateEventLog{
		UserID:    event1Create.HostID,
		EventID:   event1Create.EventID,
		EventType: string(database.EventUpdate),
		Event:     event1Update,
	}
	respUpdateEventLog, err := PostRequestCreateEventLog(client, server.URL, reqUpdateEventLog)
	require.NoError(t, err)
	event1UpdateLog.LogID = respUpdateEventLog.LogID

	// Leave event
	reqLeaveEventLog := &RequestCreateEventLog{
		UserID:    leaveUserID,
		EventID:   event1Create.EventID,
		EventType: string(database.EventLeave),
		Event:     nil,
	}
	respLeaveEventLog, err := PostRequestCreateEventLog(client, server.URL, reqLeaveEventLog)
	require.NoError(t, err)
	event1LeaveLog.LogID = respLeaveEventLog.LogID

	// Delete event
	reqDeleteEventLog := &RequestCreateEventLog{
		UserID:    event1Create.HostID,
		EventID:   event1Create.EventID,
		EventType: string(database.EventDelete),
		Event:     nil,
	}
	respDeleteEventLog, err := PostRequestCreateEventLog(client, server.URL, reqDeleteEventLog)
	require.NoError(t, err)
	event1DeleteLog.LogID = respDeleteEventLog.LogID

	// Get event logs
	reqGetEventLogs := &RequestGetEventLogs{
		EventID: event1Create.EventID,
	}
	respGetEventLogs, err := GetRequestGetEventLogs(client, server.URL, reqGetEventLogs)
	require.NoError(t, err)
	require.Equal(t, 5, len(respGetEventLogs.EventLogs))

	dbEvent1CreateLog := respGetEventLogs.EventLogs[0]
	require.Equal(t, event1CreateLog.LogID, dbEvent1CreateLog.LogID)
	require.Equal(t, event1CreateLog.UserID, dbEvent1CreateLog.UserID)
	require.Equal(t, event1CreateLog.EventID, dbEvent1CreateLog.EventID)
	require.Equal(t, event1CreateLog.EventType, dbEvent1CreateLog.EventType)
	require.Equal(t, event1CreateLog.Content, dbEvent1CreateLog.Content)

	dbEvent1JoinLog := respGetEventLogs.EventLogs[1]
	require.Equal(t, event1JoinLog.LogID, dbEvent1JoinLog.LogID)
	require.Equal(t, event1JoinLog.UserID, dbEvent1JoinLog.UserID)
	require.Equal(t, event1JoinLog.EventID, dbEvent1JoinLog.EventID)
	require.Equal(t, event1JoinLog.EventType, dbEvent1JoinLog.EventType)
	require.Equal(t, event1JoinLog.Content, dbEvent1JoinLog.Content)

	dbEvent1UpdateLog := respGetEventLogs.EventLogs[2]
	require.Equal(t, event1UpdateLog.LogID, dbEvent1UpdateLog.LogID)
	require.Equal(t, event1UpdateLog.UserID, dbEvent1UpdateLog.UserID)
	require.Equal(t, event1UpdateLog.EventID, dbEvent1UpdateLog.EventID)
	require.Equal(t, event1UpdateLog.EventType, dbEvent1UpdateLog.EventType)
	require.Equal(t, event1UpdateLog.Content, dbEvent1UpdateLog.Content)

	dbEvent1LeaveLog := respGetEventLogs.EventLogs[3]
	require.Equal(t, event1LeaveLog.LogID, dbEvent1LeaveLog.LogID)
	require.Equal(t, event1LeaveLog.UserID, dbEvent1LeaveLog.UserID)
	require.Equal(t, event1LeaveLog.EventID, dbEvent1LeaveLog.EventID)
	require.Equal(t, event1LeaveLog.EventType, dbEvent1LeaveLog.EventType)
	require.Equal(t, event1LeaveLog.Content, dbEvent1LeaveLog.Content)

	dbEvent1DeleteLog := respGetEventLogs.EventLogs[4]
	require.Equal(t, event1DeleteLog.LogID, dbEvent1DeleteLog.LogID)
	require.Equal(t, event1DeleteLog.UserID, dbEvent1DeleteLog.UserID)
	require.Equal(t, event1DeleteLog.EventID, dbEvent1DeleteLog.EventID)
	require.Equal(t, event1DeleteLog.EventType, dbEvent1DeleteLog.EventType)
	require.Equal(t, event1DeleteLog.Content, dbEvent1DeleteLog.Content)
}
