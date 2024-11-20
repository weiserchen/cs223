package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
		values, err = format.EncodeParam[In](*params)
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

	endpoint = fmt.Sprintf("http://%s%s", addr, path)
	req, err = http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRequestParam, err)
	}

	if method == http.MethodGet {
		req.URL.RawQuery = values.Encode()
	}

	res, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	defer res.Body.Close()

	if res.StatusCode != code {
		return nil, fmt.Errorf("%w: %v", ErrBadResponseCode, res.StatusCode)
	}

	var resp Out
	if err = json.NewDecoder(res.Body).Decode(&resp); err != nil {
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
	return Request[In, Out](client, http.MethodPut, addr, path, code, params)
}

func DeleteRequest[In, Out any](client *http.Client, addr, path string, code int, params *In) (*Out, error) {
	return Request[In, Out](client, http.MethodDelete, addr, path, code, params)
}

func GetRequestGetUser(client *http.Client, addr string, params *RequestGetUser) (*ResponseGetUser, error) {
	return GetRequest[RequestGetUser, ResponseGetUser](client, addr, PathGetUser, http.StatusOK, params)
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

func PostRequestCreateEventLog(client *http.Client, addr string, params *RequestCreateEventLog) (*ResponseCreateEventLog, error) {
	return PostRequest[RequestCreateEventLog, ResponseCreateEventLog](client, addr, PathCreateEvent, http.StatusCreated, params)
}
