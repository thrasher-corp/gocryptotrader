package log

import (
	"fmt"
	"io"
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
func (sl *SubLogger) SetOutput(o io.Writer) {
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

	sl.mtx.RLock()
	defer sl.mtx.RUnlock()
	return &logFields{
		info:   sl.levels.Info,
		warn:   sl.levels.Warn,
		debug:  sl.levels.Debug,
		error:  sl.levels.Error,
		name:   sl.name,
		output: sl.output,
		logger: logger,
	}
}
