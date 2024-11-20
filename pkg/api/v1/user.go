package v1

import (
	"fmt"
	"net/http"
	"txchain/pkg/database"
	"txchain/pkg/format"
	"txchain/pkg/middleware"
	"txchain/pkg/router"
)

type RequestGetUser struct {
	UserID int64 `json:"user_id" schema:"user_id"`
}

type ResponseGetUser struct {
	UserID     int64   `json:"user_id" schema:"user_id"`
	UserName   string  `json:"user_name" schema:"user_name"`
	HostEvents []int64 `json:"host_events" schema:"host_events"`
}

func HandleGetUser(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user *database.User
		var err error

		req := middleware.MarshalQuery[RequestGetUser](r)
		user, err = cfg.DB.UserStore.GetUser(cfg.Ctx, req.UserID)
		if err != nil {
			format.WriteJsonResponse(w, fmt.Errorf("%w: %v", ErrGetUser, err).Error(), http.StatusInternalServerError)
			return
		}

		resp := ResponseGetUser{
			UserID:     user.ID,
			UserName:   user.Name,
			HostEvents: user.HostEvents,
		}
		format.WriteJsonResponse(w, resp, http.StatusOK)
	})
}

type RequestGetUserID struct {
	UserName string `json:"user_name" schema:"user_name"`
}

type ResponseGetUserID struct {
	UserID int64 `json:"user_id" schema:"user_name"`
}

func HandleGetUserID(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID int64
		var err error

		req := middleware.MarshalQuery[RequestGetUserID](r)
		userID, err = cfg.DB.UserStore.GetID(cfg.Ctx, req.UserName)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrGetUserID, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseGetUserID{
			UserID: userID,
		}
		format.WriteJsonResponse(w, resp, http.StatusOK)
	})
}

type RequestGetUserName struct {
	UserID int64 `json:"user_id" schema:"user_id"`
}

type ResponseGetUserName struct {
	UserName string `json:"user_name" schema:"user_name"`
}

func HandleGetUserName(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userName string
		var err error

		req := middleware.MarshalQuery[RequestGetUserName](r)
		userName, err = cfg.DB.UserStore.GetName(cfg.Ctx, req.UserID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrGetUserName, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseGetUserName{
			UserName: userName,
		}
		format.WriteJsonResponse(w, resp, http.StatusOK)
	})
}

type RequestGetUserHostEvents struct {
	UserID int64 `json:"user_id" schema:"user_id"`
}

type ResponseGetUserHostEvents struct {
	HostEvents []int64 `json:"host_events" schema:"host_events"`
}

func HandleGetUserHostEvents(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var hostEvents []int64
		var err error

		req := middleware.MarshalQuery[RequestGetUserHostEvents](r)
		hostEvents, err = cfg.DB.UserStore.GetHostEvents(cfg.Ctx, req.UserID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrGetUserHostEvents, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseGetUserHostEvents{
			HostEvents: hostEvents,
		}
		format.WriteJsonResponse(w, resp, http.StatusOK)
	})
}

type RequestCreateUser struct {
	UserName   string  `json:"user_name" schema:"user_name"`
	HostEvents []int64 `json:"host_events" schema:"host_events"`
}

type ResponseCreateUser struct {
	UserID int64 `json:"user_id"`
}

func HandleCreateUser(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID int64
		var err error

		req := middleware.MarshalBody[RequestCreateUser](r)
		userID, err = cfg.DB.UserStore.CreateUser(cfg.Ctx, req.UserName, req.HostEvents)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrCreateUser, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseCreateUser{
			UserID: userID,
		}
		format.WriteJsonResponse(w, resp, http.StatusCreated)
	})
}

type RequestDeleteUser struct {
	UserID int64 `json:"user_id" schema:"user_id"`
}

type ResponseDeleteUser struct {
}

func HandleDeleteUser(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		req := middleware.MarshalBody[RequestDeleteUser](r)
		err = cfg.DB.UserStore.DeleteUser(cfg.Ctx, req.UserID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrDeleteEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseDeleteUser{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestUpdateUserName struct {
	UserID   int64  `json:"user_id" schema:"user_id"`
	UserName string `json:"user_name" schema:"user_name"`
}

type ResponseUpdateUserName struct {
}

func HandleUpdateUserName(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		req := middleware.MarshalBody[RequestUpdateUserName](r)
		err = cfg.DB.UserStore.UpdateName(cfg.Ctx, req.UserID, req.UserName)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrUpdateUserName, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseUpdateUserName{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestAddUserHostEvent struct {
	UserID  int64 `json:"user_id" schema:"user_id"`
	EventID int64 `json:"event_id" schema:"event_id"`
}

type ResponseAddUserHostEvent struct {
}

func HandleAddUserHostEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		req := middleware.MarshalBody[RequestAddUserHostEvent](r)
		err = cfg.DB.UserStore.AddHostEvent(cfg.Ctx, req.UserID, req.EventID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrAddUserHostEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseAddUserHostEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}

type RequestRemoveUserHostEvent struct {
	UserID  int64 `json:"user_id" schema:"user_id"`
	EventID int64 `json:"event_id" schema:"event_id"`
}

type ResponseRemoveUserHostEvent struct {
}

func HandleRemoveUserHostEvent(cfg *router.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		req := middleware.MarshalBody[RequestRemoveUserHostEvent](r)
		err = cfg.DB.UserStore.RemoveHostEvent(cfg.Ctx, req.UserID, req.EventID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrRemoveUserHostEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseRemoveUserHostEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}
