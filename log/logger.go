package log

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

var (
	errEmptyLoggerName            = errors.New("cannot have empty logger name")
	errSubLoggerAlreadyregistered = errors.New("sub logger already registered")
)

// NewSubLogger allows for a new sub logger to be registered.
func NewSubLogger(name string) (*SubLogger, error) {
	if name == "" {
		return nil, errEmptyLoggerName
	}
	name = strings.ToUpper(name)
	if _, ok := SubLoggers[name]; ok {
		return nil, errSubLoggerAlreadyregistered
	}
	return registerNewSubLogger(name), nil
}

// SetOutput overrides the default output with a new writer
func (sl *SubLogger) SetOutput(o io.Writer) {
	RWM.Lock()
	sl.output = o
	RWM.Unlock()
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

	return &logFields{
		info:   sl.Info,
		warn:   sl.Warn,
		debug:  sl.Debug,
		error:  sl.Error,
		name:   sl.name,
		output: sl.output,
		logger: logger,
	}
}

func newLogger(c *Config) Logger {
	return Logger{
		Timestamp:         c.AdvancedSettings.TimeStampFormat,
		Spacer:            c.AdvancedSettings.Spacer,
		ErrorHeader:       c.AdvancedSettings.Headers.Error,
		InfoHeader:        c.AdvancedSettings.Headers.Info,
		WarnHeader:        c.AdvancedSettings.Headers.Warn,
		DebugHeader:       c.AdvancedSettings.Headers.Debug,
		ShowLogSystemName: *c.AdvancedSettings.ShowLogSystemName,
	}
}

func (l *Logger) newLogEvent(data, header, slName string, w io.Writer) error {
	if w == nil {
		return errors.New("io.Writer not set")
	}

	pool, ok := eventPool.Get().(*[]byte)
	if !ok {
		return errors.New("unable to type assert slice of bytes pointer")
	}

	*pool = append(*pool, header...)
	if l.ShowLogSystemName {
		*pool = append(*pool, l.Spacer...)
		*pool = append(*pool, slName...)
	}
	*pool = append(*pool, l.Spacer...)
	if l.Timestamp != "" {
		*pool = time.Now().AppendFormat(*pool, l.Timestamp)
	}
	*pool = append(*pool, l.Spacer...)
	*pool = append(*pool, data...)
	if data == "" || data[len(data)-1] != '\n' {
		*pool = append(*pool, '\n')
	}
	_, err := w.Write(*pool)
	*pool = (*pool)[:0]
	eventPool.Put(pool)

	return err
}

// CloseLogger is called on shutdown of application
func CloseLogger() error {
	return GlobalLogFile.Close()
}

func validSubLogger(s string) (bool, *SubLogger) {
	if v, found := SubLoggers[s]; found {
		return true, v
	}
	return false, nil
}

// Level retries the current sublogger levels
func Level(s string) (*Levels, error) {
	found, subLogger := validSubLogger(s)
	if !found {
		return nil, fmt.Errorf("logger %v not found", s)
	}

	return &subLogger.Levels, nil
}

// SetLevel sets sublogger levels
func SetLevel(s, level string) (*Levels, error) {
	found, subLogger := validSubLogger(s)
	if !found {
		return nil, fmt.Errorf("logger %v not found", s)
	}
	subLogger.Levels = splitLevel(level)
	return &subLogger.Levels, nil
}
