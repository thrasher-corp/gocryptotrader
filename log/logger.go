package log

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

var (
	errEmptyLoggerName = errors.New("cannot have empty logger name")
	// ErrSubLoggerAlreadyRegistered Returned when a sublogger is registered multiple times
	ErrSubLoggerAlreadyRegistered = errors.New("sub logger already registered")
)

func newLogger(c *Config) Logger {
	return Logger{
		TimestampFormat:               c.AdvancedSettings.TimeStampFormat,
		Spacer:                        c.AdvancedSettings.Spacer,
		ErrorHeader:                   c.AdvancedSettings.Headers.Error,
		InfoHeader:                    c.AdvancedSettings.Headers.Info,
		WarnHeader:                    c.AdvancedSettings.Headers.Warn,
		DebugHeader:                   c.AdvancedSettings.Headers.Debug,
		ShowLogSystemName:             c.AdvancedSettings.ShowLogSystemName != nil && *c.AdvancedSettings.ShowLogSystemName,
		BypassJobChannelFilledWarning: c.AdvancedSettings.BypassJobChannelFilledWarning,
	}
}

// CloseLogger is called on shutdown of application
func CloseLogger() error {
	mu.Lock()
	globalLogConfig.Enabled = convert.BoolPtr(false)
	close(jobsChannel)
	workerWg.Wait()
	mu.Unlock()
	return globalLogFile.Close()
}

// Level retries the current sublogger levels
func Level(name string) (Levels, error) {
	mu.RLock()
	defer mu.RUnlock()
	subLogger, found := SubLoggers[name]
	if !found {
		return Levels{}, fmt.Errorf("logger %s not found", name)
	}
	return subLogger.levels, nil
}

// SetLevel sets sublogger levels
func SetLevel(s, level string) (Levels, error) {
	mu.Lock()
	defer mu.Unlock()
	subLogger, found := SubLoggers[s]
	if !found {
		return Levels{}, fmt.Errorf("sub logger %v not found", s)
	}
	subLogger.SetLevels(splitLevel(level))
	return subLogger.levels, nil
}
