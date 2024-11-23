package v1

import (
	"net/http"
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

func NewUserRoutes(cfg *router.Config) []router.Route {
	return []router.Route{
		{
			Verb:        http.MethodGet,
			Path:        PathGetUser,
			Handler:     HandleGetUser(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[RequestGetUser]},
		},
		{
			Verb:        http.MethodGet,
			Path:        PathGetUserID,
			Handler:     HandleGetUserID(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[RequestGetUserID]},
		},
		{
			Verb:        http.MethodGet,
			Path:        PathGetUserName,
			Handler:     HandleGetUserName(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[RequestGetUserName]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathUpdateUserName,
			Handler:     HandleUpdateUserName(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestUpdateUserName]},
		},
		{
			Verb:        http.MethodGet,
			Path:        PathGetUserHostEvents,
			Handler:     HandleGetUserHostEvents(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[RequestGetUserHostEvents]},
		},
		{
			Verb:        http.MethodPost,
			Path:        PathCreateUser,
			Handler:     HandleCreateUser(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestCreateUser]},
		},
		{
			Verb:        http.MethodDelete,
			Path:        PathDeleteUser,
			Handler:     HandleDeleteUser(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestDeleteUser]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathAddUserHostEvent,
			Handler:     HandleAddUserHostEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestAddUserHostEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathRemoveUserHostEvent,
			Handler:     HandleRemoveUserHostEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestRemoveUserHostEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        PathTxCreateEvent,
			Handler:     HandleTxCreateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestTxCreateEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        PathTxUpdateEvent,
			Handler:     HandleTxUpdateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestTxUpdateEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        PathTxDeleteEvent,
			Handler:     HandleTxDeleteEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestTxDeleteEvent]},
		},
	}
}

func NewEventRoutes(cfg *router.Config) []router.Route {
	return []router.Route{
		{
			Verb:        http.MethodGet,
			Path:        PathGetEvent,
			Handler:     HandleGetEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[RequestGetEvent]},
		},
		{
			Verb:        http.MethodPost,
			Path:        PathCreateEvent,
			Handler:     HandleCreateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestCreateEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathUpdateEvent,
			Handler:     HandleUpdateEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestUpdateEvent]},
		},
		{
			Verb:        http.MethodDelete,
			Path:        PathDeleteEvent,
			Handler:     HandleDeleteEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestDeleteEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathAddEventParticipant,
			Handler:     HandleAddEventParticipant(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestAddEventParticipant]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathRemoveEventParticipant,
			Handler:     HandleRemoveEventParticipant(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestRemoveEventParticipant]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathTxJoinEvent,
			Handler:     HandleTxJoinEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestTxJoinEvent]},
		},
		{
			Verb:        http.MethodPut,
			Path:        PathTxLeaveEvent,
			Handler:     HandleTxLeaveEvent(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestTxLeaveEvent]},
		},
	}
}

func NewEventLogRoutes(cfg *router.Config) []router.Route {
	return []router.Route{
		{
			Verb:        http.MethodGet,
			Path:        PathGetEventLogs,
			Handler:     HandleGetEventLogs(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateQuery[RequestGetEventLogs]},
		},
		{
			Verb:        http.MethodPost,
			Path:        PathCreateEventLog,
			Handler:     HandleCreateEventLog(cfg),
			Middlewares: []middleware.Middlerware{middleware.ValidateBody[RequestCreateEventLog]},
		},
	}
}
