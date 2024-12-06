package v1

import (
	"context"
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

		req := middleware.UnmarshalRequest[RequestGetUser](r)
		user, err = cfg.DB.UserStore.GetUser(r.Context(), req.UserID)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrGetUser, err), http.StatusInternalServerError)
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

		req := middleware.UnmarshalRequest[RequestGetUserID](r)
		userID, err = cfg.DB.UserStore.GetID(r.Context(), req.UserName)
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

		req := middleware.UnmarshalRequest[RequestGetUserName](r)
		userName, err = cfg.DB.UserStore.GetName(r.Context(), req.UserID)
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

		req := middleware.UnmarshalRequest[RequestGetUserHostEvents](r)
		hostEvents, err = cfg.DB.UserStore.GetHostEvents(r.Context(), req.UserID)
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

		req := middleware.UnmarshalRequest[RequestCreateUser](r)
		userID, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (int64, error) {
				return cfg.DB.UserStore.CreateUser(ctx, req.UserName, req.HostEvents)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestDeleteUser](r)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.UserStore.DeleteUser(r.Context(), req.UserID)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestUpdateUserName](r)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.UserStore.UpdateName(r.Context(), req.UserID, req.UserName)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestAddUserHostEvent](r)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.UserStore.AddHostEvent(r.Context(), req.UserID, req.EventID)
			},
		)
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

		req := middleware.UnmarshalRequest[RequestRemoveUserHostEvent](r)
		_, err = database.UnwrapResult(
			r.Context(),
			func(ctx context.Context) (any, error) {
				return cfg.DB.UserStore.RemoveHostEvent(r.Context(), req.UserID, req.EventID)
			},
		)
		if err != nil {
			format.WriteJsonResponse(w, format.NewErrorResponse(ErrRemoveUserHostEvent, err), http.StatusInternalServerError)
			return
		}

		resp := ResponseRemoveUserHostEvent{}
		format.WriteJsonResponse(w, resp, http.StatusNoContent)
	})
}
