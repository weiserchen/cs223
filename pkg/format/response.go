package format

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	ErrorMsg string `json:"error_msg"`
}

func NewErrorResponse(errType error, err error) *ErrorResponse {
	return &ErrorResponse{
		ErrorMsg: fmt.Errorf("%w: %v", errType, err).Error(),
	}
}

func WriteResponse(w http.ResponseWriter, msg []byte, code int) (int, error) {
	w.WriteHeader(code)
	return w.Write(msg)
}

func WriteResponseStr(w http.ResponseWriter, msg string, code int) (int, error) {
	return WriteResponse(w, []byte(msg), code)
}

func WriteJsonResponse[T any](w http.ResponseWriter, msg T, code int) (int, error) {
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(msg)
	if err != nil {
		b, _ = json.Marshal(NewErrorResponse(ErrJsonEncode, fmt.Errorf("%v", msg)))
	}
	return WriteResponse(w, b, code)
}

func WriteTextResponse(w http.ResponseWriter, msg string, code int) (int, error) {
	w.Header().Set("Content-Type", "text/plain")
	return WriteResponseStr(w, msg, code)
}
