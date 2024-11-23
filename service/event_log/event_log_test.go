package main

import (
	"log"
	"net/http"
	"net/http/httptest"
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
		Timeout: 600 * time.Second,
	}
}
func TestEventLogAPI(t *testing.T) {
	t.Parallel()

	pgc, err := database.NewContainerTableEventLogs(t, "17.1")
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
	routes := apiV1.NewEventLogRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}
	r.Build()

	client := defaultClient()
	server := httptest.NewServer(r.Routes())

	log.Println(server.URL)

	event1Create := &apiV1.APIEvent{
		EventID:      1,
		EventName:    "Test Event",
		EventInfo:    "This is a test event.",
		HostID:       1,
		StartAt:      time.Date(2000, 12, 25, 18, 0, 0, 0, time.Local),
		EndAt:        time.Date(2000, 12, 25, 22, 0, 0, 0, time.Local),
		Location:     "Aldrich Park",
		Participants: []int64{10, 20, 30},
	}

	event1CreateLog := &apiV1.APIEventLog{
		EventID:   1,
		UserID:    1,
		EventType: string(database.EventCreate),
		Content:   database.GenLogCreateEvent(apiV1.APIEventToDatabaseEvent(event1Create)),
	}

	// Create event
	reqCreateEventLog := &apiV1.RequestCreateEventLog{
		UserID:    event1Create.HostID,
		EventID:   event1Create.EventID,
		EventType: string(database.EventCreate),
		Event:     event1Create,
	}
	respCreateEventLog, err := apiV1.PostRequestCreateEventLog(client, server.URL, reqCreateEventLog)
	require.NoError(t, err)
	event1CreateLog.LogID = respCreateEventLog.LogID

	// GetEvent
	reqGetEventLogs := &apiV1.RequestGetEventLogs{
		EventID: event1Create.EventID,
	}
	respGetEventLogs, err := apiV1.GetRequestGetEventLogs(client, server.URL, reqGetEventLogs)
	require.NoError(t, err)
	require.Equal(t, 1, len(respGetEventLogs.EventLogs))
	dbEvent1CreateLog := respGetEventLogs.EventLogs[0]
	require.Equal(t, event1CreateLog.LogID, dbEvent1CreateLog.LogID)
	require.Equal(t, event1CreateLog.UserID, dbEvent1CreateLog.UserID)
	require.Equal(t, event1CreateLog.EventID, dbEvent1CreateLog.EventID)
	require.Equal(t, event1CreateLog.EventType, dbEvent1CreateLog.EventType)
	require.Equal(t, event1CreateLog.Content, dbEvent1CreateLog.Content)
}
