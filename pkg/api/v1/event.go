package v1

import (
	"net/http"
	"time"
	"txchain/pkg/database"
	"txchain/pkg/format"
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

type APIEvent struct {
	EventID      int64     `json:"event_id" schema:"event_id"`
	EventName    string    `json:"event_name" schema:"event_name"`
	EventInfo    string    `json:"event_info" schema:"event_info"`
	HostID       int64     `json:"host_id" schema:"host_id"`
	StartAt      time.Time `json:"start_at" schema:"start_at"`
	EndAt        time.Time `json:"end_at" schema:"end_at"`
	Location     string    `json:"location" schema:"location"`
	Participants []int64   `json:"participants" schema:"participants"`
}

func DatabaseEventToAPIEvent(dbEvent *database.Event) *APIEvent {
	return &APIEvent{
		EventID:      dbEvent.ID,
		EventName:    dbEvent.Name,
		EventInfo:    dbEvent.Info,
		HostID:       dbEvent.HostID,
		StartAt:      dbEvent.StartAt,
		EndAt:        dbEvent.EndAt,
		Location:     dbEvent.Location,
		Participants: dbEvent.Participants,
	}
}

func APIEventToDatabaseEvent(apiEvent *APIEvent) *database.Event {
	return &database.Event{
		ID:           apiEvent.EventID,
		Name:         apiEvent.EventName,
		Info:         apiEvent.EventInfo,
		HostID:       apiEvent.HostID,
		StartAt:      apiEvent.StartAt,
		EndAt:        apiEvent.EndAt,
		Location:     apiEvent.Location,
		Participants: apiEvent.Participants,
	}
}

type RequestGetEvent struct {
	EventID int64 `json:"event_id" schema:"event_id"`
}

type ResponseGetEvent struct {
	Event *APIEvent `json:"event" schema:"event"`
}

func HandleGetEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var dbEvent *database.Event
		var err error

		req := middleware.MarshalQuery[RequestGetEvent](r)
		dbEvent, err = cfg.DB.EventStore.GetEvent(cfg.Ctx, req.EventID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrGetEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseGetEvent{
			Event: DatabaseEventToAPIEvent(dbEvent),
		}
		format.WriteJsonResponse(w, resp, http.StatusOK)
	})
}

type RequestCreateEvent struct {
	Event *APIEvent `json:"event" schema:"event"`
}

type ResponseCreateEvent struct {
	EventID int64 `json:"event_id" schema:"event_id"`
}

func HandleCreateEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var eventID int64
		var dbEvent *database.Event
		var err error

		req := middleware.MarshalBody[RequestCreateEvent](r)
		dbEvent = APIEventToDatabaseEvent(req.Event)
		eventID, err = cfg.DB.EventStore.CreateEvent(cfg.Ctx, dbEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrCreateEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseCreateEvent{
			EventID: eventID,
		}
		format.WriteJsonResponse(w, resp, http.StatusCreated)
	})
}

type RequestUpdateEvent struct {
	Event *APIEvent `json:"event" schema:"event"`
}

type ResponseUpdateEvent struct {
}

func HandleUpdateEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var dbEvent *database.Event
		var err error

		req := middleware.MarshalBody[RequestUpdateEvent](r)
		dbEvent = APIEventToDatabaseEvent(req.Event)
		err = cfg.DB.EventStore.UpdateEvent(cfg.Ctx, dbEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrUpdateEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseUpdateEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestDeleteEvent struct {
	EventID int64 `json:"event_id" schema:"event_id"`
}

type ResponseDeleteEvent struct {
}

func HandleDeleteEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		req := middleware.MarshalBody[RequestDeleteEvent](r)
		err = cfg.DB.EventStore.DeleteEvent(cfg.Ctx, req.EventID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrDeleteEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseDeleteEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestAddEventParticipant struct {
	EventID       int64 `json:"event_id" schema:"event_id"`
	ParticipantID int64 `json:"participant_id" schema:"participant_id"`
}

type ResponseAddEventParticipant struct {
}

func HandleAddEventParticipant(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		req := middleware.MarshalBody[RequestAddEventParticipant](r)
		err = cfg.DB.EventStore.AddParticipant(cfg.Ctx, req.EventID, req.ParticipantID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrAddEventParticipant, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseAddEventParticipant{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestRemoveEventParticipant struct {
	EventID       int64 `json:"event_id" schema:"event_id"`
	ParticipantID int64 `json:"participant_id" schema:"participant_id"`
}

type ResponseRemoveEventParticipant struct {
}

func HandleRemoveEventParticipant(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		req := middleware.MarshalBody[RequestRemoveEventParticipant](r)
		err = cfg.DB.EventStore.AddParticipant(cfg.Ctx, req.EventID, req.ParticipantID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrRemoveEventParticipant, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseRemoveEventParticipant{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}
