package log

import (
	"errors"
	"fmt"
)

var (
	errEmptyLoggerName = errors.New("cannot have empty logger name")
	// ErrSubLoggerAlreadyRegistered Returned when a sublogger is registered multiple times
	ErrSubLoggerAlreadyRegistered = errors.New("sub logger already registered")
)

func newLogger(c *Config) Logger {
	return Logger{
		TimestampFormat:   c.AdvancedSettings.TimeStampFormat,
		Spacer:            c.AdvancedSettings.Spacer,
		ErrorHeader:       c.AdvancedSettings.Headers.Error,
		InfoHeader:        c.AdvancedSettings.Headers.Info,
		WarnHeader:        c.AdvancedSettings.Headers.Warn,
		DebugHeader:       c.AdvancedSettings.Headers.Debug,
		ShowLogSystemName: *c.AdvancedSettings.ShowLogSystemName,
	}
}

// CloseLogger is called on shutdown of application
func CloseLogger() error {
	return GlobalLogFile.Close()
}

// Level retries the current sublogger levels
func Level(name string) (Levels, error) {
	RWM.RLock()
	defer RWM.RUnlock()
	subLogger, found := SubLoggers[name]
	if !found {
		return Levels{}, fmt.Errorf("logger %s not found", name)
	}
	return subLogger.levels, nil
}

// SetLevel sets sublogger levels
func SetLevel(s, level string) (Levels, error) {
	RWM.Lock()
	defer RWM.Unlock()
	subLogger, found := SubLoggers[s]
	if !found {
		return Levels{}, fmt.Errorf("sub logger %v not found", s)
	}
	subLogger.SetLevels(splitLevel(level))
	return subLogger.levels, nil
}
