package log

import (
	"fmt"
	"log"
)

// Infoln takes a pointer subLogger struct and interface sends to StageLogEvent
func Infoln(sl *SubLogger, a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stageln(fields.logger.InfoHeader, a...)
}

// Infof takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Infof(sl *SubLogger, format string, a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	sl.getFields().stagef(fields.logger.InfoHeader, format, a...)
}

// Debugln takes a pointer subLogger struct, string and interface sends to StageLogEvent
func Debugln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stageln(fields.logger.DebugHeader, v...)
}

// Debugf takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Debugf(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	sl.getFields().stagef(fields.logger.DebugHeader, data, v...)
}

// Warnln takes a pointer subLogger struct & interface formats and sends to StageLogEvent
func Warnln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stageln(fields.logger.WarnHeader, v...)
}

// Warnf takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Warnf(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	sl.getFields().stagef(fields.logger.WarnHeader, data, v...)
}

// Errorln takes a pointer subLogger struct, string & interface formats and sends to StageLogEvent
func Errorln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stageln(fields.logger.ErrorHeader, v...)
}

// Errorf takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Errorf(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	sl.getFields().stagef(fields.logger.ErrorHeader, data, v...)
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}

// enabled checks if the log level is enabled
func (l *logFields) enabled(header string) string {
	switch header {
	case l.logger.InfoHeader:
		if l.info {
			return "info"
		}
	case l.logger.WarnHeader:
		if l.warn {
			return "warn"
		}
	case l.logger.ErrorHeader:
		if l.error {
			return "error"
		}
	case l.logger.DebugHeader:
		if l.debug {
			return "debug"
		}
	}
	return ""
}

// stage stages a log event
func (l *logFields) stage(header string, deferFunc deferral) {
	if l == nil {
		return
	}
	if level := l.enabled(header); level != "" {
		l.output.StageLogEvent(deferFunc,
			header,
			l.name,
			l.logger.Spacer,
			l.logger.TimestampFormat,
			l.botName,
			level,
			l.logger.ShowLogSystemName,
			l.logger.BypassJobChannelFilledWarning,
			l.structuredLogging,
			l.structuredFields)
	}
	logFieldsPool.Put(l)
}

// stageln stages a log event
func (l *logFields) stageln(header string, a ...interface{}) {
	l.stage(header, func() string { return fmt.Sprint(a...) })
}

// stagef stages a log event
func (l *logFields) stagef(header, format string, a ...interface{}) {
	l.stage(header, func() string { return fmt.Sprintf(format, a...) })
}

// WithFields allows the user to add custom fields to a structured log output
// NOTE: If structured logging is disabled, this function will do not add
// new fields to the log output.
func WithFields(sl *SubLogger, structuredFields map[Key]interface{}) *logFields {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return nil
	}
	fields.structuredFields = structuredFields
	return fields
}

// Errorln formats using the default formats for its operands and writes to
// standard output as an error message. A new line is automatically applied.
func (l *logFields) Errorln(a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.ErrorHeader, a...)
}

// Errorf formats according to a format specifier and writes to standard output
// as an error message. A new line is automatically applied.
func (l *logFields) Errorf(format string, a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.ErrorHeader, format, a...)
}

// Warnln formats using the default formats for its operands and writes to
// standard output as a warning message. A new line is automatically applied.
func (l *logFields) Warnln(a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.WarnHeader, a...)
}

// Warnf formats according to a format specifier and writes to standard output
// as a warning message. A new line is automatically applied.
func (l *logFields) Warnf(format string, a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.WarnHeader, format, a...)
}

// Infoln formats using the default formats for its operands and writes to
// standard output as an informational message. A new line is automatically applied.
func (l *logFields) Infoln(a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.InfoHeader, a...)
}

// Infof formats according to a format specifier and writes to standard output
// as an informational message. A new line is automatically applied.
func (l *logFields) Infof(format string, a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.InfoHeader, format, a...)
}

// Debugln formats using the default formats for its operands and writes to
// standard output as a debug message. A new line is automatically applied.
func (l *logFields) Debugln(a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.DebugHeader, a...)
}

// Debugf formats according to a format specifier and writes to standard output
// as a debug message. A new line is automatically applied.
func (l *logFields) Debugf(format string, a ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.DebugHeader, format, a...)
}
