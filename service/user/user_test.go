package main

import (
	"log"
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
	t.Parallel()

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
	routes := apiV1.NewUserRoutes(cfg)
	for _, route := range routes {
		r.AddRoute(route)
	}
	r.Build()

	client := defaultClient()
	server := httptest.NewServer(r.Routes())

	log.Println(server.URL)

	user := struct {
		ID         int64
		Name       string
		HostEvents []int64
	}{
		Name:       "Alice",
		HostEvents: []int64{1, 2, 3},
	}

	// Create user
	reqCreateUser := &apiV1.RequestCreateUser{
		UserName:   user.Name,
		HostEvents: user.HostEvents,
	}
	respCreateUser, err := apiV1.PostRequestCreateUser(client, server.URL, reqCreateUser)
	require.NoError(t, err)
	user.ID = respCreateUser.UserID

	// Get user
	reqGetUser := &apiV1.RequestGetUser{
		UserID: user.ID,
	}
	respGetUser, err := apiV1.GetRequestGetUser(client, server.URL, reqGetUser)
	require.NoError(t, err)
	require.Equal(t, user.ID, respGetUser.UserID)
	require.Equal(t, user.Name, respGetUser.UserName)
	require.Equal(t, user.HostEvents, respGetUser.HostEvents)

	// Get user id
	reqGetUserID := &apiV1.RequestGetUserID{
		UserName: user.Name,
	}
	respGetUserID, err := apiV1.GetRequestGetUserID(client, server.URL, reqGetUserID)
	require.NoError(t, err)
	require.Equal(t, user.ID, respGetUserID.UserID)

	// Get user name
	reqGetUserName := &apiV1.RequestGetUserName{
		UserID: user.ID,
	}
	respGetUserName, err := apiV1.GetRequestGetUserName(client, server.URL, reqGetUserName)
	require.NoError(t, err)
	require.Equal(t, user.Name, respGetUserName.UserName)

	// Get user host event
	reqGetUserHostEvents := &apiV1.RequestGetUserHostEvents{
		UserID: user.ID,
	}
	respGetUserHostEvents, err := apiV1.GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	require.Equal(t, user.HostEvents, respGetUserHostEvents.HostEvents)

	// Update user name
	user.Name = "Bob"
	reqUpdateUserName := &apiV1.RequestUpdateUserName{
		UserID:   user.ID,
		UserName: user.Name,
	}
	_, err = apiV1.PutRequestUpdateUserName(client, server.URL, reqUpdateUserName)
	require.NoError(t, err)

	respGetUserName, err = apiV1.GetRequestGetUserName(client, server.URL, reqGetUserName)
	require.NoError(t, err)
	require.Equal(t, user.Name, respGetUserName.UserName)

	// Add user host event
	reqAddUserHostEvent := &apiV1.RequestAddUserHostEvent{
		UserID:  user.ID,
		EventID: 4,
	}
	_, err = apiV1.PutRequestAddUserHostEvent(client, server.URL, reqAddUserHostEvent)
	require.NoError(t, err)

	// Remove user host event
	reqRemoveUserHostEvent := &apiV1.RequestRemoveUserHostEvent{
		UserID:  user.ID,
		EventID: 2,
	}
	_, err = apiV1.PutRequestRemoveUserHostEvent(client, server.URL, reqRemoveUserHostEvent)
	require.NoError(t, err)

	// Get user host events
	respGetUserHostEvents, err = apiV1.GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	newHostEvents := slices.Sorted(slices.Values(respGetUserHostEvents.HostEvents))
	require.Equal(t, []int64{1, 3, 4}, newHostEvents)

	// Duplicate add/remove host events
	_, err = apiV1.PutRequestAddUserHostEvent(client, server.URL, reqAddUserHostEvent)
	require.NoError(t, err)

	_, err = apiV1.PutRequestRemoveUserHostEvent(client, server.URL, reqRemoveUserHostEvent)
	require.NoError(t, err)

	respGetUserHostEvents, err = apiV1.GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	newHostEvents = slices.Sorted(slices.Values(respGetUserHostEvents.HostEvents))
	require.Equal(t, []int64{1, 3, 4}, newHostEvents)

	// Delete user
	reqDeleteUser := &apiV1.RequestDeleteUser{
		UserID: user.ID,
	}
	_, err = apiV1.DeleteRequestDeleteUser(client, server.URL, reqDeleteUser)
	require.NoError(t, err)

	// Get non-exist user
	_, err = apiV1.GetRequestGetUser(client, server.URL, reqGetUser)
	require.Error(t, err)
}
