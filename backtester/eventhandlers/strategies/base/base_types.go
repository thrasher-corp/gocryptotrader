package base

import "errors"

var (
	// ErrCustomSettingsUnsupported used when custom settings are found in the start config when they shouldn't be
	ErrCustomSettingsUnsupported = errors.New("custom settings not supported")
	// ErrSimultaneousProcessingNotSupported used when strategy does not support simultaneous processing
	// but start config is set to use it
	ErrSimultaneousProcessingNotSupported = errors.New("does not support simultaneous processing and could not be loaded")
	ErrSimultaneousProcessingOnly         = errors.New("this strategy only supports simultaneous processing")
	// ErrStrategyNotFound used when strategy specified in start config does not exist
	ErrStrategyNotFound = errors.New("not found. Please ensure the strategy-settings field 'name' is spelled properly in your .start config")
	// ErrInvalidCustomSettings used when bad custom settings are found in the start config
	ErrInvalidCustomSettings = errors.New("invalid custom settings in config")
	// ErrTooMuchBadData used when there is too much missing data
	ErrTooMuchBadData = errors.New("backtesting cannot continue as there is too much invalid data. Please review your dataset")
)
