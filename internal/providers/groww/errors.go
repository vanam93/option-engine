package groww

import (
	"errors"
	"fmt"
)

// Sentinel errors for Groww provider operations.
var (
	ErrNotConfigured    = errors.New("groww provider not configured")
	ErrNotConnected     = errors.New("groww provider not connected")
	ErrNotAuthenticated = errors.New("groww provider not authenticated")
	ErrUnauthorized     = errors.New("groww unauthorized")
	ErrForbidden        = errors.New("groww forbidden")
	ErrNotFound         = errors.New("groww resource not found")
	ErrRateLimited      = errors.New("groww rate limited")
	ErrServer           = errors.New("groww server error")
	ErrTimeout          = errors.New("groww request timeout")
	ErrAPIFailure       = errors.New("groww api failure")
)

// APIError wraps HTTP and API-level failures with context.
type HTTPError struct {
	StatusCode int
	Code       string
	Message    string
	Err        error
}

func (e *HTTPError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("groww http %d (%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("groww http %d: %s", e.StatusCode, e.Message)
}

func (e *HTTPError) Unwrap() error { return e.Err }

// classifyHTTP maps status codes to typed errors.
func classifyHTTP(status int, code, message string) error {
	base := &HTTPError{StatusCode: status, Code: code, Message: message}
	switch status {
	case 401:
		base.Err = ErrUnauthorized
	case 403:
		base.Err = ErrForbidden
	case 404:
		base.Err = ErrNotFound
	case 429:
		base.Err = ErrRateLimited
	default:
		if status >= 500 {
			base.Err = ErrServer
		} else {
			base.Err = ErrAPIFailure
		}
	}
	return base
}
