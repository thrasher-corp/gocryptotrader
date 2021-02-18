package base

import "errors"

var (
	ErrCustomSettingsUnsupported          = errors.New("custom settings not supported")
	ErrSimultaneousProcessingNotSupported = errors.New("does not support simultaneous processing and could not be loaded")
	ErrStrategyNotFound                   = errors.New("not found. Please ensure the strategy-settings field 'name' is spelled properly in your .start config")
	ErrInvalidCustomSettings              = errors.New("invalid custom settings in config")
)
