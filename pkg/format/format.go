package format

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/schema"
	"github.com/mitchellh/mapstructure"
)

var (
	schemaDecoder = schema.NewDecoder()
	schemaEncoder = schema.NewEncoder()
)

var (
	ErrJsonEncode   = errors.New("failed to encode json")
	ErrJsonDecode   = errors.New("failed to decode json")
	ErrSchemaDecode = errors.New("failed to decode schema")
)

func Encode[T any](w http.ResponseWriter, r *http.Request, status int, v T) error {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("%w: %v", ErrJsonDecode, err)
	}
	return nil
}

func DecodeBody[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("%w: %v", ErrJsonDecode, err)
	}
	return v, nil
}

func DecodeParam[T any](r *http.Request) (T, error) {
	var v T
	if err := schemaDecoder.Decode(&v, r.URL.Query()); err != nil {
		return v, fmt.Errorf("%w: %v", ErrSchemaDecode, err)
	}
	return v, nil
}

func EncodeParam[T any](v T) (url.Values, error) {
	values := url.Values{}
	if err := schemaEncoder.Encode(v, values); err != nil {
		return nil, err
	}
	return values, nil
}

func UnmarshalInput[T any](v any) (T, error) {
	var input T
	err := mapstructure.Decode(v, &input)
	return input, err
}
