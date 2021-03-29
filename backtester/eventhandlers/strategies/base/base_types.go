package base

import "errors"

// Error vars related to strategies and invalid config settings
var (
	ErrCustomSettingsUnsupported          = errors.New("custom settings not supported")
	ErrSimultaneousProcessingNotSupported = errors.New("does not support simultaneous processing and could not be loaded")
	ErrStrategyNotFound                   = errors.New("not found. Please ensure the strategy-settings field 'name' is spelled properly in your .start config")
	ErrInvalidCustomSettings              = errors.New("invalid custom settings in config")
	ErrTooMuchBadData                     = errors.New("backtesting cannot continue as there is too much invalid data. Please review your dataset")
)
