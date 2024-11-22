package v1

const (
	PathGetUser             = "/api/v1/user"
	PathGetUserID           = "/api/v1/user_id"
	PathGetUserName         = "/api/v1/user_name"
	PathUpdateUserName      = "/api/v1/user_name"
	PathGetUserHostEvents   = "/api/v1/user_host_events"
	PathCreateUser          = "/api/v1/user"
	PathDeleteUser          = "/api/v1/user"
	PathAddUserHostEvent    = "/api/v1/add_user_host_events"
	PathRemoveUserHostEvent = "/api/v1/remove_user_host_events"

	PathGetEvent               = "/api/v1/event"
	PathCreateEvent            = "/api/v1/event"
	PathUpdateEvent            = "/api/v1/event"
	PathDeleteEvent            = "/api/v1/event"
	PathAddEventParticipant    = "/api/v1/add_event_participants"
	PathRemoveEventParticipant = "/api/v1/remove_event_participants"

	PathCreateEventLog = "/api/v1/event_log"
	PathGetEventLogs   = "/api/v1/event_logs"

	PathTxCreateEvent = "/api/v1/tx/event"
	PathTxUpdateEvent = "/api/v1/tx/event"
	PathTxDeleteEvent = "/api/v1/tx/event"
	PathTxJoinEvent   = "/api/v1/tx/join_event"
	PathTxLeaveEvent  = "/api/v1/tx/leave_event"
)
