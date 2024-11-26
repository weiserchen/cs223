package v1

const (
	PathGetUser             = "/api/v1/user"
	PathGetUserID           = "/api/v1/user/id"
	PathGetUserName         = "/api/v1/user/name"
	PathUpdateUserName      = "/api/v1/user/name"
	PathCreateUser          = "/api/v1/user"
	PathDeleteUser          = "/api/v1/user"
	PathGetUserHostEvents   = "/api/v1/user/host_events"
	PathAddUserHostEvent    = "/api/v1/user/host_events/add"
	PathRemoveUserHostEvent = "/api/v1/user/host_events/remove"

	PathGetEvent               = "/api/v1/event"
	PathCreateEvent            = "/api/v1/event"
	PathUpdateEvent            = "/api/v1/event"
	PathDeleteEvent            = "/api/v1/event"
	PathAddEventParticipant    = "/api/v1/event/participants/add"
	PathRemoveEventParticipant = "/api/v1/event/participants/remove"

	PathCreateEventLog = "/api/v1/event_log"
	PathGetEventLogs   = "/api/v1/event_logs"

	PathTxCreateEvent = "/api/v1/tx/event"
	PathTxUpdateEvent = "/api/v1/tx/event"
	PathTxDeleteEvent = "/api/v1/tx/event"
	PathTxJoinEvent   = "/api/v1/tx/event/join"
	PathTxLeaveEvent  = "/api/v1/tx/event/leave"
)
