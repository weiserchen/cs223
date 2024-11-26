package v1

import (
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

func NewUserRoutes(cfg *router.Config) []router.Route {
	r := router.New(cfg)

	apiV1 := r.Prefix("/api/v1")
	{
		user := apiV1.Prefix("/user")
		{
			user.Get("/", middleware.ValidateQuery[RequestGetUser](HandleGetUser(cfg)))
			user.Post("/", middleware.ValidateBody[RequestCreateUser](HandleCreateUser(cfg)))
			user.Delete("/", middleware.ValidateBody[RequestDeleteUser](HandleDeleteUser(cfg)))

			user.Get("/id", middleware.ValidateQuery[RequestGetUserID](HandleGetUserID(cfg)))

			name := user.Prefix("/name")
			{
				name.Get("/", middleware.ValidateQuery[RequestGetUserName](HandleGetUserName(cfg)))
				name.Put("/", middleware.ValidateBody[RequestUpdateUserName](HandleUpdateUserName(cfg)))
			}

			hostEvents := user.Prefix("/host_events")
			{
				hostEvents.Get("/", middleware.ValidateQuery[RequestGetUserHostEvents](HandleGetUserHostEvents(cfg)))
				hostEvents.Put("/add", middleware.ValidateBody[RequestAddUserHostEvent](HandleAddUserHostEvent(cfg)))
				hostEvents.Put("/remove", middleware.ValidateBody[RequestRemoveUserHostEvent](HandleRemoveUserHostEvent(cfg)))
			}

		}

		tx := apiV1.Prefix("/tx")
		{
			event := tx.Prefix("/event")
			{
				event.Post("/", middleware.ValidateBody[RequestTxCreateEvent](HandleTxCreateEvent(cfg)))
				event.Put("/", middleware.ValidateBody[RequestTxUpdateEvent](HandleTxUpdateEvent(cfg)))
				event.Delete("/", middleware.ValidateBody[RequestTxDeleteEvent](HandleTxDeleteEvent(cfg)))
			}
		}
	}

	return r.Routes()
}

func NewEventRoutes(cfg *router.Config) []router.Route {
	r := router.New(cfg)

	apiV1 := r.Prefix("/api/v1")
	{
		event := apiV1.Prefix("/event")
		{
			event.Get("/", middleware.ValidateQuery[RequestGetEvent](HandleGetEvent(cfg)))
			event.Post("/", middleware.ValidateBody[RequestCreateEvent](HandleCreateEvent(cfg)))
			event.Put("/", middleware.ValidateBody[RequestUpdateEvent](HandleUpdateEvent(cfg)))
			event.Delete("/", middleware.ValidateBody[RequestDeleteEvent](HandleDeleteEvent(cfg)))

			participants := event.Prefix("participants")
			{
				participants.Put("/add", middleware.ValidateBody[RequestAddEventParticipant](HandleAddEventParticipant(cfg)))
				participants.Put("/remove", middleware.ValidateBody[RequestRemoveEventParticipant](HandleRemoveEventParticipant(cfg)))
			}
		}

		tx := apiV1.Prefix("/tx")
		{
			event := tx.Prefix("/event")
			{
				event.Put("/join", middleware.ValidateBody[RequestTxJoinEvent](HandleTxJoinEvent(cfg)))
				event.Put("/leave", middleware.ValidateBody[RequestTxLeaveEvent](HandleTxLeaveEvent(cfg)))
			}
		}
	}

	return r.Routes()
}

func NewEventLogRoutes(cfg *router.Config) []router.Route {
	r := router.New(cfg)

	apiV1 := r.Prefix("/api/v1")
	{
		apiV1.Get("/event_logs", middleware.ValidateQuery[RequestGetEventLogs](HandleGetEventLogs(cfg)))
		apiV1.Post("/event_log", middleware.ValidateBody[RequestCreateEventLog](HandleCreateEventLog(cfg)))
	}

	return r.Routes()
}
