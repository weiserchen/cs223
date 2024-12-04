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

type APIEventLog struct {
	LogID     int64     `json:"log_id" schema:"log_id"`
	EventID   int64     `json:"event_id" schema:"event_id"`
	UserID    int64     `json:"user_id" schema:"user_id"`
	EventType string    `json:"event_type" schema:"event_type"`
	Content   string    `json:"content" schema:"content"`
	UpdatedAt time.Time `json:"updated_at" schema:"updated_at"`
}

func APIEventLogToDatabaseEventLog(apiEventLog *APIEventLog) *database.EventLog {
	return &database.EventLog{
		LogID:     apiEventLog.LogID,
		EventID:   apiEventLog.EventID,
		UserID:    apiEventLog.UserID,
		EventType: database.EventType(apiEventLog.EventType),
		Content:   apiEventLog.Content,
		// UpdatedAt: &apiEventLog.UpdatedAt,
	}
}

func DatabaseEventLogToAPIEventLog(dbEventLog *database.EventLog) *APIEventLog {
	return &APIEventLog{
		LogID:     dbEventLog.LogID,
		EventID:   dbEventLog.EventID,
		UserID:    dbEventLog.UserID,
		EventType: string(dbEventLog.EventType),
		Content:   dbEventLog.Content,
		UpdatedAt: dbEventLog.UpdatedAt,
	}
}

type RequestCreateEventLog struct {
	UserID    int64     `json:"user_id" schema:"user_id"`
	EventID   int64     `json:"event_id" schema:"event_id"`
	EventType string    `json:"event_type" schema:"event_type"`
	Event     *APIEvent `json:"event" schema:"event"`
}

type ResponseCreateEventLog struct {
	LogID int64 `json:"log_id" schema:"log_id"`
}

func HandleCreateEventLog(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var logID int64
		var err error

		req := middleware.MarshalRequest[RequestCreateEventLog](r)
		logID, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (int64, error) {
				return cfg.DB.EventLogStore.CreateEventLog(
					ctx,
					req.EventID,
					req.UserID,
					database.EventType(req.EventType),
					APIEventToDatabaseEvent(req.Event),
				)
			},
		)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrCreateEventLog, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseCreateEventLog{
			LogID: logID,
		}
		format.WriteJsonResponse(w, resp, http.StatusCreated)
	})
}

type RequestGetEventLogs struct {
	EventID int64 `json:"event_id" schema:"event_id"`
}

type ResponseGetEventLogs struct {
	EventLogs []*APIEventLog `json:"event_logs"`
}

func HandleGetEventLogs(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var dbEventLogs []*database.EventLog
		var err error

		req := middleware.MarshalRequest[RequestGetEventLogs](r)
		dbEventLogs, err = cfg.DB.EventLogStore.GetEventLogs(r.Context(), req.EventID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrGetEventLogs, err), http.StatusInternalServerError)
			return
		}

		var apiEventLogs []*APIEventLog
		for _, dbEventLog := range dbEventLogs {
			apiEventLogs = append(apiEventLogs, DatabaseEventLogToAPIEventLog(dbEventLog))
		}

		resp := ResponseGetEventLogs{
			EventLogs: apiEventLogs,
		}
		format.WriteJsonResponse(w, resp, http.StatusOK)
	})
}
