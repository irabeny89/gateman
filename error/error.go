package errs

import "errors"

var (
	ErrNotJSON = errors.New("Content-Type must be application/json")
	ErrBodyTooLarge = errors.New("Request body too large")
)
