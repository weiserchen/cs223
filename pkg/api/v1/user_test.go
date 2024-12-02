package v1

import (
	"log"
	"slices"
	"testing"
	"txchain/pkg/database"

	"github.com/stretchr/testify/require"
	tctr "github.com/testcontainers/testcontainers-go"
)

func TestUserAPI(t *testing.T) {
	t.Parallel()

	pgc, err := database.NewContainerTableUsers(t, "17.1")
	defer func() {
		if pgc != nil {
			tctr.CleanupContainer(t, pgc.Container)
		}
	}()
	require.NoError(t, err)

	r, err := DefaultUserRouter(
		pgc.Endpoint(),
		DefaultUserServerAddr,
		DefaultEventServerAddr,
		DefaultEventLogServerAddr,
	)
	require.NoError(t, err)

	client := DefaultHTTPClient()
	server := NewTestServer(t, r.Handler(), DefaultUserServerAddr)

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
	reqCreateUser := &RequestCreateUser{
		UserName:   user.Name,
		HostEvents: user.HostEvents,
	}
	respCreateUser, err := PostRequestCreateUser(client, server.URL, reqCreateUser)
	require.NoError(t, err)
	user.ID = respCreateUser.UserID

	// Get user
	reqGetUser := &RequestGetUser{
		UserID: user.ID,
	}
	respGetUser, err := GetRequestGetUser(client, server.URL, reqGetUser)
	require.NoError(t, err)
	require.Equal(t, user.ID, respGetUser.UserID)
	require.Equal(t, user.Name, respGetUser.UserName)
	require.Equal(t, user.HostEvents, respGetUser.HostEvents)

	// Get user id
	reqGetUserID := &RequestGetUserID{
		UserName: user.Name,
	}
	respGetUserID, err := GetRequestGetUserID(client, server.URL, reqGetUserID)
	require.NoError(t, err)
	require.Equal(t, user.ID, respGetUserID.UserID)

	// Get user name
	reqGetUserName := &RequestGetUserName{
		UserID: user.ID,
	}
	respGetUserName, err := GetRequestGetUserName(client, server.URL, reqGetUserName)
	require.NoError(t, err)
	require.Equal(t, user.Name, respGetUserName.UserName)

	// Get user host event
	reqGetUserHostEvents := &RequestGetUserHostEvents{
		UserID: user.ID,
	}
	respGetUserHostEvents, err := GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	require.Equal(t, user.HostEvents, respGetUserHostEvents.HostEvents)

	// Update user name
	user.Name = "Bob"
	reqUpdateUserName := &RequestUpdateUserName{
		UserID:   user.ID,
		UserName: user.Name,
	}
	_, err = PutRequestUpdateUserName(client, server.URL, reqUpdateUserName)
	require.NoError(t, err)

	respGetUserName, err = GetRequestGetUserName(client, server.URL, reqGetUserName)
	require.NoError(t, err)
	require.Equal(t, user.Name, respGetUserName.UserName)

	// Add user host event
	reqAddUserHostEvent := &RequestAddUserHostEvent{
		UserID:  user.ID,
		EventID: 4,
	}
	_, err = PutRequestAddUserHostEvent(client, server.URL, reqAddUserHostEvent)
	require.NoError(t, err)

	// Remove user host event
	reqRemoveUserHostEvent := &RequestRemoveUserHostEvent{
		UserID:  user.ID,
		EventID: 2,
	}
	_, err = PutRequestRemoveUserHostEvent(client, server.URL, reqRemoveUserHostEvent)
	require.NoError(t, err)

	// Get user host events
	respGetUserHostEvents, err = GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	newHostEvents := slices.Sorted(slices.Values(respGetUserHostEvents.HostEvents))
	require.Equal(t, []int64{1, 3, 4}, newHostEvents)

	// Duplicate add/remove host events
	_, err = PutRequestAddUserHostEvent(client, server.URL, reqAddUserHostEvent)
	require.NoError(t, err)

	_, err = PutRequestRemoveUserHostEvent(client, server.URL, reqRemoveUserHostEvent)
	require.NoError(t, err)

	respGetUserHostEvents, err = GetRequestGetUserHostEvents(client, server.URL, reqGetUserHostEvents)
	require.NoError(t, err)
	newHostEvents = slices.Sorted(slices.Values(respGetUserHostEvents.HostEvents))
	require.Equal(t, []int64{1, 3, 4}, newHostEvents)

	// Delete user
	reqDeleteUser := &RequestDeleteUser{
		UserID: user.ID,
	}
	_, err = DeleteRequestDeleteUser(client, server.URL, reqDeleteUser)
	require.NoError(t, err)

	// Get non-exist user
	_, err = GetRequestGetUser(client, server.URL, reqGetUser)
	require.Error(t, err)
}
