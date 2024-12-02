package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"txchain/pkg/format"
)

func Request[In, Out any](client *http.Client, method, addr, path string, code int, params *In) (*Out, error) {
	var err error
	var endpoint string
	var req *http.Request
	var res *http.Response
	var b []byte
	var body io.Reader
	var values url.Values

	if method == http.MethodGet {
		values, err = format.EncodeParam(*params)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidRequestParam, err)
		}
	} else {
		b, err = json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidRequestParam, err)
		}
		body = bytes.NewReader(b)
	}

	endpoint = fmt.Sprintf("%s%s", addr, path)
	req, err = http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequestParam, err)
	}

	if method == http.MethodGet {
		req.URL.RawQuery = values.Encode()
	}

	log.Printf("Request: %s %s. body: %s", req.Method, req.URL.String(), string(b))

	res, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	defer res.Body.Close()

	if res.StatusCode != code {
		if res.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %d", ErrBadResponseCode, res.StatusCode)
		}
		if res.StatusCode == http.StatusServiceUnavailable {
			return nil, fmt.Errorf("%w: %d", ErrServiceUnavailable, res.StatusCode)
		}

		var errResp format.ErrorResponse
		if err = json.NewDecoder(res.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("%w: %d. error msg: %v", ErrBadResponseCode, res.StatusCode, err)
		}
		return nil, fmt.Errorf("%w: %d. error msg: %v", ErrBadResponseCode, res.StatusCode, errResp)
	}

	var resp Out
	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponseBody, err)
	}

	return &resp, nil
}

func GetRequest[In, Out any](client *http.Client, addr, path string, code int, params *In) (*Out, error) {
	return Request[In, Out](client, http.MethodGet, addr, path, code, params)
}

func PutRequest[In, Out any](client *http.Client, addr, path string, code int, params *In) (*Out, error) {
	return Request[In, Out](client, http.MethodPut, addr, path, code, params)
}

func PostRequest[In, Out any](client *http.Client, addr, path string, code int, params *In) (*Out, error) {
	return Request[In, Out](client, http.MethodPost, addr, path, code, params)
}

func DeleteRequest[In, Out any](client *http.Client, addr, path string, code int, params *In) (*Out, error) {
	return Request[In, Out](client, http.MethodDelete, addr, path, code, params)
}

func GetRequestGetUser(client *http.Client, addr string, params *RequestGetUser) (*ResponseGetUser, error) {
	return GetRequest[RequestGetUser, ResponseGetUser](client, addr, PathGetUser, http.StatusOK, params)
}

func GetRequestGetUserID(client *http.Client, addr string, params *RequestGetUserID) (*ResponseGetUserID, error) {
	return GetRequest[RequestGetUserID, ResponseGetUserID](client, addr, PathGetUserID, http.StatusOK, params)
}

func GetRequestGetUserName(client *http.Client, addr string, params *RequestGetUserName) (*ResponseGetUserName, error) {
	return GetRequest[RequestGetUserName, ResponseGetUserName](client, addr, PathGetUserName, http.StatusOK, params)
}

func GetRequestGetUserHostEvents(client *http.Client, addr string, params *RequestGetUserHostEvents) (*ResponseGetUserHostEvents, error) {
	return GetRequest[RequestGetUserHostEvents, ResponseGetUserHostEvents](client, addr, PathGetUserHostEvents, http.StatusOK, params)
}

func PostRequestCreateUser(client *http.Client, addr string, params *RequestCreateUser) (*ResponseCreateUser, error) {
	return PostRequest[RequestCreateUser, ResponseCreateUser](client, addr, PathCreateUser, http.StatusCreated, params)
}

func DeleteRequestDeleteUser(client *http.Client, addr string, params *RequestDeleteUser) (*ResponseDeleteUser, error) {
	return DeleteRequest[RequestDeleteUser, ResponseDeleteUser](client, addr, PathDeleteUser, http.StatusNoContent, params)
}

func PutRequestUpdateUserName(client *http.Client, addr string, params *RequestUpdateUserName) (*ResponseUpdateUserName, error) {
	return PutRequest[RequestUpdateUserName, ResponseUpdateUserName](client, addr, PathUpdateUserName, http.StatusNoContent, params)
}

func PutRequestAddUserHostEvent(client *http.Client, addr string, params *RequestAddUserHostEvent) (*ResponseAddUserHostEvent, error) {
	return PutRequest[RequestAddUserHostEvent, ResponseAddUserHostEvent](client, addr, PathAddUserHostEvent, http.StatusNoContent, params)
}

func PutRequestRemoveUserHostEvent(client *http.Client, addr string, params *RequestRemoveUserHostEvent) (*ResponseRemoveUserHostEvent, error) {
	return PutRequest[RequestRemoveUserHostEvent, ResponseRemoveUserHostEvent](client, addr, PathRemoveUserHostEvent, http.StatusNoContent, params)
}

func GetRequestGetEvent(client *http.Client, addr string, params *RequestGetEvent) (*ResponseGetEvent, error) {
	return GetRequest[RequestGetEvent, ResponseGetEvent](client, addr, PathGetEvent, http.StatusOK, params)
}

func PostRequestCreateEvent(client *http.Client, addr string, params *RequestCreateEvent) (*ResponseCreateEvent, error) {
	return PostRequest[RequestCreateEvent, ResponseCreateEvent](client, addr, PathCreateEvent, http.StatusCreated, params)
}

func PutRequestUpdateEvent(client *http.Client, addr string, params *RequestUpdateEvent) (*ResponseUpdateEvent, error) {
	return PutRequest[RequestUpdateEvent, ResponseUpdateEvent](client, addr, PathUpdateEvent, http.StatusNoContent, params)
}

func DeleteRequestDeleteEvent(client *http.Client, addr string, params *RequestDeleteEvent) (*ResponseDeleteEvent, error) {
	return DeleteRequest[RequestDeleteEvent, ResponseDeleteEvent](client, addr, PathDeleteEvent, http.StatusNoContent, params)
}

func PutRequestAddEventParticipant(client *http.Client, addr string, params *RequestAddEventParticipant) (*ResponseAddEventParticipant, error) {
	return PutRequest[RequestAddEventParticipant, ResponseAddEventParticipant](client, addr, PathAddEventParticipant, http.StatusNoContent, params)
}

func PutRequestRemoveEventParticipant(client *http.Client, addr string, params *RequestRemoveEventParticipant) (*ResponseRemoveEventParticipant, error) {
	return PutRequest[RequestRemoveEventParticipant, ResponseRemoveEventParticipant](client, addr, PathRemoveEventParticipant, http.StatusNoContent, params)
}

func GetRequestGetEventLogs(client *http.Client, addr string, params *RequestGetEventLogs) (*ResponseGetEventLogs, error) {
	return GetRequest[RequestGetEventLogs, ResponseGetEventLogs](client, addr, PathGetEventLogs, http.StatusOK, params)
}

func PostRequestCreateEventLog(client *http.Client, addr string, params *RequestCreateEventLog) (*ResponseCreateEventLog, error) {
	return PostRequest[RequestCreateEventLog, ResponseCreateEventLog](client, addr, PathCreateEventLog, http.StatusCreated, params)
}

func PostRequestTxCreateEvent(client *http.Client, addr string, params *RequestTxCreateEvent) (*ResponseTxCreateEvent, error) {
	return PostRequest[RequestTxCreateEvent, ResponseTxCreateEvent](client, addr, PathTxCreateEvent, http.StatusCreated, params)
}

func PutRequestTxUpdateEvent(client *http.Client, addr string, params *RequestTxUpdateEvent) (*ResponseTxUpdateEvent, error) {
	return PutRequest[RequestTxUpdateEvent, ResponseTxUpdateEvent](client, addr, PathTxUpdateEvent, http.StatusNoContent, params)
}

func DeleteRequestTxDeleteEvent(client *http.Client, addr string, params *RequestTxDeleteEvent) (*ResponseTxDeleteEvent, error) {
	return DeleteRequest[RequestTxDeleteEvent, ResponseTxDeleteEvent](client, addr, PathTxDeleteEvent, http.StatusNoContent, params)
}

func PutRequestTxJoinEvent(client *http.Client, addr string, params *RequestTxJoinEvent) (*ResponseTxJoinEvent, error) {
	return PutRequest[RequestTxJoinEvent, ResponseTxJoinEvent](client, addr, PathTxJoinEvent, http.StatusNoContent, params)
}

func PutRequestTxLeaveEvent(client *http.Client, addr string, params *RequestTxLeaveEvent) (*ResponseTxLeaveEvent, error) {
	return PutRequest[RequestTxLeaveEvent, ResponseTxLeaveEvent](client, addr, PathTxLeaveEvent, http.StatusNoContent, params)
}
