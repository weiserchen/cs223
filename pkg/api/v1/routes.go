package v1

import (
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

func NewUserRoutes(cfg *router.Config) []router.Route {
	r := router.New(cfg)

	apiV1 := r.Prefix("/api/v1")
	apiV1.Apply(middleware.TxParticipant(cfg.TxMgr, cfg.Logger, "user"))
	{
		user := apiV1.Prefix("/user")
		{
			user.Get("/", HandleGetUser(cfg)).Apply(middleware.ValidateQuery[RequestGetUser])
			user.Post("/", HandleCreateUser(cfg)).Apply(middleware.ValidateBody[RequestCreateUser])
			user.Delete("/", HandleDeleteUser(cfg)).Apply(middleware.ValidateBody[RequestDeleteUser])
			user.Get("/id", HandleGetUserID(cfg)).Apply(middleware.ValidateQuery[RequestGetUserID])

			name := user.Prefix("/name")
			{
				name.Get("/", HandleGetUserName(cfg)).Apply(middleware.ValidateQuery[RequestGetUserName])
				name.Put("/", HandleUpdateUserName(cfg)).Apply(middleware.ValidateBody[RequestUpdateUserName])
			}

			hostEvents := user.Prefix("/host_events")
			{
				hostEvents.Get("/", HandleGetUserHostEvents(cfg)).Apply(middleware.ValidateQuery[RequestGetUserHostEvents])
				hostEvents.Put("/add", HandleAddUserHostEvent(cfg)).Apply(middleware.ValidateBody[RequestAddUserHostEvent])
				hostEvents.Put("/remove", HandleRemoveUserHostEvent(cfg)).Apply(middleware.ValidateBody[RequestRemoveUserHostEvent])
			}

		}

		tx := apiV1.Prefix("/tx")
		{
			event := tx.Prefix("/event")
			{
				event.Post("/", HandleTxCreateEvent(cfg)).Apply(middleware.ValidateBody[RequestTxCreateEvent])
				event.Put("/", HandleTxUpdateEvent(cfg)).Apply(middleware.ValidateBody[RequestTxUpdateEvent])
				event.Delete("/", HandleTxDeleteEvent(cfg)).Apply(middleware.ValidateBody[RequestTxDeleteEvent])
			}
		}
	}

	return r.Routes()
}

func NewEventRoutes(cfg *router.Config) []router.Route {
	r := router.New(cfg)

	apiV1 := r.Prefix("/api/v1")
	apiV1.Apply(middleware.TxParticipant(cfg.TxMgr, cfg.Logger, "event"))
	{
		event := apiV1.Prefix("/event")
		{
			event.Get("/", HandleGetEvent(cfg)).Apply(middleware.ValidateQuery[RequestGetEvent])
			event.Post("/", HandleCreateEvent(cfg)).Apply(middleware.ValidateBody[RequestCreateEvent])
			event.Put("/", HandleUpdateEvent(cfg)).Apply(middleware.ValidateBody[RequestUpdateEvent])
			event.Delete("/", HandleDeleteEvent(cfg)).Apply(middleware.ValidateBody[RequestDeleteEvent])

			participants := event.Prefix("participants")
			{
				participants.Put("/add", HandleAddEventParticipant(cfg)).Apply(middleware.ValidateBody[RequestAddEventParticipant])
				participants.Put("/remove", HandleRemoveEventParticipant(cfg)).Apply(middleware.ValidateBody[RequestRemoveEventParticipant])
			}
		}

		tx := apiV1.Prefix("/tx")
		{
			event := tx.Prefix("/event")
			{
				event.Put("/join", HandleTxJoinEvent(cfg)).Apply(middleware.ValidateBody[RequestTxJoinEvent])
				event.Put("/leave", HandleTxLeaveEvent(cfg)).Apply(middleware.ValidateBody[RequestTxLeaveEvent])
			}
		}
	}

	return r.Routes()
}

func NewEventLogRoutes(cfg *router.Config) []router.Route {
	r := router.New(cfg)

	apiV1 := r.Prefix("/api/v1")
	apiV1.Apply(middleware.TxParticipant(cfg.TxMgr, cfg.Logger, "event_log"))
	{
		apiV1.Get("/event_logs", HandleGetEventLogs(cfg)).Apply(middleware.ValidateQuery[RequestGetEventLogs])
		apiV1.Post("/event_log", HandleCreateEventLog(cfg)).Apply(middleware.ValidateBody[RequestCreateEventLog])
	}

	return r.Routes()
}

func NewTestTxRoutes(cfg *router.Config) []router.Route {
	r := router.New(cfg)

	return r.Routes()
}
