package v1

import (
	"log"
	"net/http"
	"time"
	"txchain/pkg/database"
	"txchain/pkg/format"
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

type RequestTxCreateEvent struct {
	UserID       int64     `json:"user_id" schema:"user_id"`
	EventName    string    `json:"event_name" schema:"event_name"`
	EventInfo    string    `json:"event_info" schema:"event_info"`
	StartAt      time.Time `json:"start_at" schema:"start_at"`
	EndAt        time.Time `json:"end_at" schema:"end_at"`
	Location     string    `json:"location" schema:"location"`
	Participants []int64   `json:"participants" schema:"participants"`
}

type ResponseTxCreateEvent struct {
	EventID int64 `json:"event_id" schema:"event_id"`
}

func HandleTxCreateEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		client := &http.Client{
			Timeout: DefaultTimeout,
		}

		req := middleware.UnmarshalRequest[RequestTxCreateEvent](r)
		serviceUser := cfg.Peers[router.ServiceUser]
		serviceEvent := cfg.Peers[router.ServiceEvent]
		serviceEventLog := cfg.Peers[router.ServiceEventLog]
		log.Println(cfg.Peers)
		log.Println(serviceUser, serviceEvent, serviceEventLog)
		log.Println(cfg.Getenv(router.ConfigServiceUserAddr), cfg.Getenv(router.ConfigServiceEventAddr), cfg.Getenv(router.ConfigServiceEventLogAddr))

		event := &APIEvent{
			EventName:    req.EventName,
			EventInfo:    req.EventInfo,
			HostID:       req.UserID,
			StartAt:      req.StartAt,
			EndAt:        req.EndAt,
			Location:     req.Location,
			Participants: req.Participants,
		}
		reqCreateEvent := &RequestCreateEvent{
			Event: event,
		}

		var respCreateEvent *ResponseCreateEvent
		respCreateEvent, err = PostRequestCreateEvent(client, serviceEvent, reqCreateEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxCreateEvent, err), http.StatusInternalServerError)
			return
		}

		event.EventID = respCreateEvent.EventID
		reqCreateEventLog := &RequestCreateEventLog{
			UserID:    event.HostID,
			EventID:   event.EventID,
			EventType: string(database.EventCreate),
			Event:     event,
		}
		_, err = PostRequestCreateEventLog(client, serviceEventLog, reqCreateEventLog)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxCreateEvent, err), http.StatusInternalServerError)
			return
		}

		reqAddUserHostEvent := &RequestAddUserHostEvent{
			UserID:  event.HostID,
			EventID: event.EventID,
		}
		_, err = PutRequestAddUserHostEvent(client, serviceUser, reqAddUserHostEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxCreateEvent, err), http.StatusInternalServerError)
			return
		}

		resp := &ResponseTxCreateEvent{
			EventID: event.EventID,
		}
		format.WriteJsonResponse(w, resp, http.StatusCreated)
	})
}

type RequestTxUpdateEvent struct {
	UserID    int64     `json:"user_id" schema:"user_id"`
	EventID   int64     `json:"event_id" schema:"event_id"`
	EventName string    `json:"event_name" schema:"event_name"`
	EventInfo string    `json:"event_info" schema:"event_info"`
	StartAt   time.Time `json:"start_at" schema:"start_at"`
	EndAt     time.Time `json:"end_at" schema:"end_at"`
	Location  string    `json:"location" schema:"location"`
}

type ResponseTxUpdateEvent struct {
}

func HandleTxUpdateEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		client := &http.Client{
			Timeout: DefaultTimeout,
		}

		req := middleware.UnmarshalRequest[RequestTxUpdateEvent](r)
		serviceEvent := cfg.Peers[router.ServiceEvent]
		serviceEventLog := cfg.Peers[router.ServiceEventLog]

		event := &APIEvent{
			EventID:   req.EventID,
			EventName: req.EventName,
			EventInfo: req.EventInfo,
			HostID:    req.UserID,
			StartAt:   req.StartAt,
			EndAt:     req.EndAt,
			Location:  req.Location,
		}
		reqUpdateEvent := &RequestUpdateEvent{
			Event: event,
		}
		_, err = PutRequestUpdateEvent(client, serviceEvent, reqUpdateEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxUpdateEvent, err), http.StatusInternalServerError)
			return
		}

		reqCreateEventLog := &RequestCreateEventLog{
			UserID:    req.UserID,
			EventID:   req.EventID,
			EventType: string(database.EventUpdate),
			Event:     event,
		}
		_, err = PostRequestCreateEventLog(client, serviceEventLog, reqCreateEventLog)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxUpdateEvent, err), http.StatusInternalServerError)
			return
		}

		resp := &ResponseTxUpdateEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestTxDeleteEvent struct {
	UserID  int64 `json:"user_id" schema:"user_id"`
	EventID int64 `json:"event_id" schema:"event_id"`
}

type ResponseTxDeleteEvent struct {
}

func HandleTxDeleteEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		client := &http.Client{
			Timeout: DefaultTimeout,
		}

		req := middleware.UnmarshalRequest[RequestTxDeleteEvent](r)
		serviceUser := cfg.Peers[router.ServiceUser]
		serviceEvent := cfg.Peers[router.ServiceEvent]
		serviceEventLog := cfg.Peers[router.ServiceEventLog]

		reqRemoveUserHostEvent := &RequestRemoveUserHostEvent{
			UserID:  req.UserID,
			EventID: req.EventID,
		}
		_, err = PutRequestRemoveUserHostEvent(client, serviceUser, reqRemoveUserHostEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxDeleteEvent, err), http.StatusInternalServerError)
			return
		}

		reqGetEvent := &RequestGetEvent{
			EventID: req.EventID,
		}
		var respGetEvent *ResponseGetEvent
		respGetEvent, err = GetRequestGetEvent(client, serviceEvent, reqGetEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxDeleteEvent, err), http.StatusInternalServerError)
			return
		}

		reqDeleteEvent := &RequestDeleteEvent{
			EventID: req.EventID,
		}
		_, err = DeleteRequestDeleteEvent(client, serviceEvent, reqDeleteEvent)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxDeleteEvent, err), http.StatusInternalServerError)
			return
		}

		reqCreateEventLog := &RequestCreateEventLog{
			UserID:    req.UserID,
			EventID:   req.EventID,
			EventType: string(database.EventDelete),
			Event:     respGetEvent.Event,
		}
		_, err = PostRequestCreateEventLog(client, serviceEventLog, reqCreateEventLog)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxDeleteEvent, err), http.StatusInternalServerError)
			return
		}

		resp := &ResponseTxDeleteEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestTxJoinEvent struct {
	EventID       int64 `json:"event_id" schema:"event_id"`
	HostID        int64 `json:"host_id" schema:"host_id"`
	ParticipantID int64 `json:"participant_id" schema:"participant_id"`
}

type ResponseTxJoinEvent struct {
}

func HandleTxJoinEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		client := &http.Client{
			Timeout: DefaultTimeout,
		}

		req := middleware.UnmarshalRequest[RequestTxJoinEvent](r)
		serviceEvent := cfg.Peers[router.ServiceEvent]
		serviceEventLog := cfg.Peers[router.ServiceEventLog]

		reqAddEventParticipant := &RequestAddEventParticipant{
			EventID:       req.EventID,
			ParticipantID: req.ParticipantID,
		}
		_, err = PutRequestAddEventParticipant(client, serviceEvent, reqAddEventParticipant)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxJoinEvent, err), http.StatusInternalServerError)
			return
		}

		reqCreateEventLog := &RequestCreateEventLog{
			UserID:    req.ParticipantID,
			EventID:   req.EventID,
			EventType: string(database.EventJoin),
			Event:     &APIEvent{},
		}
		_, err = PostRequestCreateEventLog(client, serviceEventLog, reqCreateEventLog)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxJoinEvent, err), http.StatusInternalServerError)
			return
		}

		resp := &ResponseTxJoinEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestTxLeaveEvent struct {
	EventID       int64 `json:"event_id" schema:"event_id"`
	HostID        int64 `json:"host_id" schema:"host_id"`
	ParticipantID int64 `json:"participant_id" schema:"participant_id"`
}

type ResponseTxLeaveEvent struct {
}

func HandleTxLeaveEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		client := &http.Client{
			Timeout: DefaultTimeout,
		}

		req := middleware.UnmarshalRequest[RequestTxLeaveEvent](r)
		serviceEvent := cfg.Peers[router.ServiceEvent]
		serviceEventLog := cfg.Peers[router.ServiceEventLog]

		reqRemoveEventParticipant := &RequestRemoveEventParticipant{
			EventID:       req.EventID,
			ParticipantID: req.ParticipantID,
		}
		_, err = PutRequestRemoveEventParticipant(client, serviceEvent, reqRemoveEventParticipant)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxLeaveEvent, err), http.StatusInternalServerError)
			return
		}

		reqCreateEventLog := &RequestCreateEventLog{
			UserID:    req.ParticipantID,
			EventID:   req.EventID,
			EventType: string(database.EventLeave),
			Event:     &APIEvent{},
		}
		_, err = PostRequestCreateEventLog(client, serviceEventLog, reqCreateEventLog)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrTxLeaveEvent, err), http.StatusInternalServerError)
			return
		}

		resp := &ResponseTxLeaveEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}
