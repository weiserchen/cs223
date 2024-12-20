package v1

import (
	"context"
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
	if apiEvent == nil {
		return nil
	}
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

		req := middleware.UnmarshalRequest[RequestGetEvent](r)
		dbEvent, err = cfg.DB.EventStore.GetEvent(r.Context(), req.EventID)
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

		req := middleware.UnmarshalRequest[RequestCreateEvent](r)
		dbEvent = APIEventToDatabaseEvent(req.Event)
		eventID, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (int64, error) {
				return cfg.DB.EventStore.CreateEvent(ctx, dbEvent)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestUpdateEvent](r)
		dbEvent = APIEventToDatabaseEvent(req.Event)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.EventStore.UpdateEvent(ctx, dbEvent)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestDeleteEvent](r)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.EventStore.DeleteEvent(ctx, req.EventID)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestAddEventParticipant](r)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.EventStore.AddParticipant(ctx, req.EventID, req.ParticipantID)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestRemoveEventParticipant](r)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.EventStore.RemoveParticipant(ctx, req.EventID, req.ParticipantID)
			},
		)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrRemoveEventParticipant, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseRemoveEventParticipant{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}
