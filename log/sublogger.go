package log

import (
	"errors"
	"fmt"
	"strings"
)

var errMultiWriterHolderIsNil = errors.New("multiwriter holder is nil")

// NewSubLogger allows for a new sub logger to be registered.
func NewSubLogger(name string) (*SubLogger, error) {
	if name == "" {
		return nil, errEmptyLoggerName
	}
	name = strings.ToUpper(name)
	mu.Lock()
	defer mu.Unlock()
	if _, ok := SubLoggers[name]; ok {
		return nil, fmt.Errorf("'%v' %w", name, ErrSubLoggerAlreadyRegistered)
	}
	return registerNewSubLogger(name), nil
}

// setOutput overrides the default output with a new writer
func (sl *SubLogger) setOutput(o *multiWriterHolder) error {
	if o == nil {
		return errMultiWriterHolderIsNil
	}
	sl.output = o
	return nil
}

// setLevels overrides the default levels with new levels; levelception
func (sl *SubLogger) setLevels(newLevels Levels) {
	sl.levels = newLevels
}

// getFields checks out and populates sub logger fields for a potential log job.
// The caller must hold mu's read lock until the value is returned to
// logFieldsPool by stageln, stagef, or stage.
func (sl *SubLogger) getFields() *fields {
	if sl == nil || globalLogConfig == nil || globalLogConfig.Enabled == nil || !*globalLogConfig.Enabled {
		return nil
	}
	f := logFieldsPool.Get().(*fields) //nolint:forcetypeassert // Not necessary from a pool
	f.structuredFields = nil
	f.info = sl.levels.Info
	f.warn = sl.levels.Warn
	f.debug = sl.levels.Debug
	f.error = sl.levels.Error
	f.name = sl.name
	f.output = sl.output
	f.botName = sl.botName
	f.structuredLogging = sl.structuredLogging
	return f
}
