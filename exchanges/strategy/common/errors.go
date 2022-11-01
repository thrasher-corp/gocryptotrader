package common

import "errors"

var (
	ErrIsNil          = errors.New("strategy is nil")
	ErrAlreadyRunning = errors.New("strategy is already running")
	ErrNotRunning     = errors.New("strategy not running")
	ErrConfigIsNil    = errors.New("strategy configuration is nil")
	ErrReporterIsNil  = errors.New("strategy reporter is nil")
)
