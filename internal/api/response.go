package api

import (
	"time"
)

// Response is the standard envelope for every Intelligence API endpoint.
type Response struct {
	Success    bool           `json:"success"`
	Timestamp  time.Time      `json:"timestamp"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Pagination *Pagination    `json:"pagination,omitempty"`
	Filters    Filter         `json:"filters,omitempty"`
	Data       any            `json:"data"`
	Errors     []string       `json:"errors,omitempty"`
}

// OK builds a successful response.
func OK(filter Filter, data any) Response {
	return Response{
		Success:   true,
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]any{"version": "v1"},
		Filters:   filter,
		Data:      data,
	}
}

// OKList builds a successful paginated list response.
func OKList(filter Filter, data any, pagination Pagination) Response {
	return Response{
		Success:    true,
		Timestamp:  time.Now().UTC(),
		Metadata:   map[string]any{"version": "v1"},
		Pagination: &pagination,
		Filters:    filter,
		Data:       data,
	}
}

// Fail builds an error response.
func Fail(filter Filter, statusErr error) Response {
	msg := "internal error"
	if statusErr != nil {
		msg = statusErr.Error()
	}
	return Response{
		Success:   false,
		Timestamp: time.Now().UTC(),
		Filters:   filter,
		Data:      nil,
		Errors:    []string{msg},
	}
}
