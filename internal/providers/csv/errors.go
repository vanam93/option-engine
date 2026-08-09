package csv

import (
	"errors"
	"fmt"
)

// Sentinel errors for CSV provider operations.
var (
	ErrNotConfigured = errors.New("csv provider not configured")
	ErrNotConnected  = errors.New("csv provider not connected")
	ErrFileNotFound  = errors.New("csv data file not found")
	ErrNoData        = errors.New("csv data file contains no candles")
)

// ParseError describes a malformed CSV row.
type ParseError struct {
	Line    int64
	Field   string
	Message string
}

func (e *ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("csv parse error at line %d (%s): %s", e.Line, e.Field, e.Message)
	}
	return fmt.Sprintf("csv parse error at line %d: %s", e.Line, e.Message)
}

// RowError wraps iterator-level failures with file context.
type RowError struct {
	File string
	Line int64
	Err  error
}

func (e *RowError) Error() string {
	return fmt.Sprintf("csv row error in %s at line %d: %v", e.File, e.Line, e.Err)
}

func (e *RowError) Unwrap() error { return e.Err }
