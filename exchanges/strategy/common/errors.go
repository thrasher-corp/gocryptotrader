package common

import "errors"

var (
	ErrInvalidUUID = errors.New("invalid UUID")

	ErrIsNil          = errors.New("strategy is nil")
	ErrNotFound       = errors.New("strategy not found")
	ErrAlreadyRunning = errors.New("strategy is already running")
	ErrNotRunning     = errors.New("strategy not running")
	ErrConfigIsNil    = errors.New("strategy configuration is nil")
	ErrReporterIsNil  = errors.New("strategy reporter is nil")
)
