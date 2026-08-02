package api

import "errors"

var (
	ErrDisabled      = errors.New("api: intelligence api disabled")
	ErrNotFound      = errors.New("api: resource not found")
	ErrInvalidFilter = errors.New("api: invalid filter")
)

const componentName = "intelligence_api"
