package log

import (
	"fmt"
	"strings"
)

// NewSubLogger allows for a new sub logger to be registered.
func NewSubLogger(name string) (*SubLogger, error) {
	if name == "" {
		return nil, errEmptyLoggerName
	}
	name = strings.ToUpper(name)
	RWM.RLock()
	if _, ok := SubLoggers[name]; ok {
		RWM.RUnlock()
		return nil, fmt.Errorf("'%v' %w", name, ErrSubLoggerAlreadyRegistered)
	}
	RWM.RUnlock()
	return registerNewSubLogger(name), nil
}

// SetOutput overrides the default output with a new writer
func (sl *SubLogger) SetOutput(o *multiWriterHolder) {
	sl.mtx.Lock()
	sl.output = o
	sl.mtx.Unlock()
}

// SetLevels overrides the default levels with new levels; levelception
func (sl *SubLogger) SetLevels(newLevels Levels) {
	sl.mtx.Lock()
	sl.levels = newLevels
	sl.mtx.Unlock()
}

// GetLevels returns current functional log levels
func (sl *SubLogger) GetLevels() Levels {
	sl.mtx.RLock()
	defer sl.mtx.RUnlock()
	return sl.levels
}

func (sl *SubLogger) getFields() *logFields {
	RWM.RLock()
	defer RWM.RUnlock()

	if sl == nil ||
		(GlobalLogConfig != nil &&
			GlobalLogConfig.Enabled != nil &&
			!*GlobalLogConfig.Enabled) {
		return nil
	}

	fields := logFieldsPool.Get().(*logFields) // nolint:forcetypeassert // Not neccessary from a pool

	sl.mtx.RLock()
	defer sl.mtx.RUnlock()
	fields.info = sl.levels.Info
	fields.warn = sl.levels.Warn
	fields.debug = sl.levels.Debug
	fields.error = sl.levels.Error
	fields.name = sl.name
	fields.output = sl.output
	fields.logger = logger
	return fields
}
