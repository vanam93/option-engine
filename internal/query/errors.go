package query

import "errors"

var (
	ErrDisabled    = errors.New("query: api disabled")
	ErrNotFound    = errors.New("query: not found")
	ErrInvalidFilter = errors.New("query: invalid filter")
)

const componentName = "query_api"
