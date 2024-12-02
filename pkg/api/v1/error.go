package v1

import (
	"errors"
	"time"
)

const (
	DefaultTimeout = 30 * time.Second
)

var (
	ErrGetUser             = errors.New("failed to get user")
	ErrGetUserID           = errors.New("failed to get user id")
	ErrGetUserName         = errors.New("failed to get user name")
	ErrGetUserHostEvents   = errors.New("failed to get user host events")
	ErrCreateUser          = errors.New("failed to create user")
	ErrDeleteUser          = errors.New("failed to delete user")
	ErrUpdateUserName      = errors.New("failed to update user name")
	ErrAddUserHostEvent    = errors.New("failed to add user host event")
	ErrRemoveUserHostEvent = errors.New("failed to remove user host event")

	ErrGetEvent               = errors.New("failed to get event")
	ErrCreateEvent            = errors.New("failed to create event")
	ErrUpdateEvent            = errors.New("failed to update event")
	ErrDeleteEvent            = errors.New("failed to delete event")
	ErrAddEventParticipant    = errors.New("failed to add event participant")
	ErrRemoveEventParticipant = errors.New("failed to remove event participant")

	ErrCreateEventLog = errors.New("failed to create event log")
	ErrGetEventLogs   = errors.New("failed to get event logs")

	ErrTxCreateEvent = errors.New("tx: failed to create event")
	ErrTxUpdateEvent = errors.New("tx: failed to update event")
	ErrTxDeleteEvent = errors.New("tx: failed to delete event")
	ErrTxJoinEvent   = errors.New("tx: failed to join event")
	ErrTxLeaveEvent  = errors.New("tx: failed to leave event")

	ErrTestTxFilterType = errors.New("test tx: invalid tx filter type")
	ErrTestTxFilterOp   = errors.New("test tx: invalid tx filter operation")

	ErrInvalidRequestParam = errors.New("invalid request param")
	ErrBadRequest          = errors.New("bad request")
	ErrBadResponseCode     = errors.New("bad response code")
	ErrInvalidResponseBody = errors.New("invalid response body")
	ErrServiceUnavailable  = errors.New("service unavailable")
)
