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
func TestUserAPI(t *testing.T) {
	pgc, err := database.NewContainerTableUsers(t, "17.1")
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
	routes := getRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}
	r.Build()

	client := defaultClient()
	server := httptest.NewServer(r.Routes())

	userNameAlice := "alice"
	hostEventsAlice := []int64{1, 2, 3}
	reqCreateUser := &apiV1.RequestCreateUser{
		UserName:   userNameAlice,
		HostEvents: hostEventsAlice,
	}
	respCreateUser, err := apiV1.PostRequestCreateUser(client, server.URL, reqCreateUser)
	require.NoError(t, err)

	reqGetUser := &apiV1.RequestGetUser{
		UserID: respCreateUser.UserID,
	}
	respGetUser, err := apiV1.GetRequestGetUser(client, server.URL, reqGetUser)
	require.NoError(t, err)
	require.Equal(t, respCreateUser.UserID, respGetUser.UserID)
	require.Equal(t, reqCreateUser.UserName, respGetUser.UserName)
	require.Equal(t, reqCreateUser.HostEvents, respGetUser.HostEvents)

	reqGetUserID := &apiV1.RequestGetUserID{
		UserName: reqCreateUser.UserName,
	}
	respGetUserID, err := apiV1.GetRequestGetUserID(client, server.URL, reqGetUserID)
	require.NoError(t, err)
	require.Equal(t, respCreateUser.UserID, respGetUserID.UserID)

	reqGetUserName := &apiV1.RequestGetUserName{
		UserID: respCreateUser.UserID,
	}
	respGetUserName, err := apiV1.GetRequestGetUserName(client, server.URL, reqGetUserName)
	require.NoError(t, err)
	require.Equal(t, reqCreateUser.UserName, respGetUserName.UserName)

	reqGetUserHostEvents := &apiV1.RequestGetUserHostEvents{
		UserID: respCreateUser.UserID,
	}
	respGetUserHostEvents, err := apiV1.GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	require.Equal(t, reqCreateUser.HostEvents, respGetUserHostEvents.HostEvents)

	reqAddUserHostEvent := &apiV1.RequestAddUserHostEvent{
		UserID:  respCreateUser.UserID,
		EventID: 4,
	}
	_, err = apiV1.PutRequestAddUserHostEvent(client, server.URL, reqAddUserHostEvent)
	require.NoError(t, err)

	reqRemoveUserHostEvent := &apiV1.RequestRemoveUserHostEvent{
		UserID:  respCreateUser.UserID,
		EventID: 2,
	}
	_, err = apiV1.PutRequestRemoveUserHostEvent(client, server.URL, reqRemoveUserHostEvent)
	require.NoError(t, err)

	respGetUserHostEvents, err = apiV1.GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	newHostEvents := slices.Sorted(slices.Values(respGetUserHostEvents.HostEvents))
	require.Equal(t, []int64{1, 3, 4}, newHostEvents)

	// test duplicate add/remove
	_, err = apiV1.PutRequestAddUserHostEvent(client, server.URL, reqAddUserHostEvent)
	require.NoError(t, err)

	_, err = apiV1.PutRequestRemoveUserHostEvent(client, server.URL, reqRemoveUserHostEvent)
	require.NoError(t, err)

	respGetUserHostEvents, err = apiV1.GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	newHostEvents = slices.Sorted(slices.Values(respGetUserHostEvents.HostEvents))
	require.Equal(t, []int64{1, 3, 4}, newHostEvents)

	reqDeleteUser := &apiV1.RequestDeleteUser{
		UserID: respCreateUser.UserID,
	}
	_, err = apiV1.DeleteRequestDeleteUser(client, server.URL, reqDeleteUser)
	require.NoError(t, err)

	// test user not exists
	_, err = apiV1.GetRequestGetUser(client, server.URL, reqGetUser)
	require.Error(t, err)
}
