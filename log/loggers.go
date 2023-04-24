package log

import (
	"context"
	"fmt"
	"log"
)

// Info takes a pointer subLogger struct and string sends to StageLogEvent
func Info(sl *SubLogger, data string) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stage(fields.logger.InfoHeader, data)
}

// Infoln takes a pointer subLogger struct and interface sends to StageLogEvent
func Infoln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stageln(fields.logger.InfoHeader, v...)
}

// Infof takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Infof(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	sl.getFields().stagef(fields.logger.InfoHeader, data, v...)
}

// Debug takes a pointer subLogger struct and string sends to StageLogEvent
func Debug(sl *SubLogger, data string) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stage(fields.logger.DebugHeader, data)
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

// Warn takes a pointer subLogger struct & string and sends to StageLogEvent
func Warn(sl *SubLogger, data string) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stage(fields.logger.WarnHeader, data)
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

// Error takes a pointer subLogger struct & interface formats and sends to StageLogEvent
func Error(sl *SubLogger, data string) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	fields.stage(fields.logger.ErrorHeader, data)
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
func (l *logFields) stage(header string, data string) {
	if l == nil {
		return
	}
	if level := l.enabled(header); level != "" {
		l.output.StageLogEvent(func() string { return data },
			header,
			l.name,
			l.logger.Spacer,
			l.logger.TimestampFormat,
			l.instance,
			level,
			l.logger.ShowLogSystemName,
			l.logger.BypassJobChannelFilledWarning,
			l.logger.StructuredLogging,
			l.structuredFields)
	}
	logFieldsPool.Put(l)
}

// stageln stages a log event
func (l *logFields) stageln(header string, data ...interface{}) {
	if l == nil {
		return
	}
	if level := l.enabled(header); level != "" {
		l.output.StageLogEvent(func() string { return fmt.Sprint(data...) },
			header,
			l.name,
			l.logger.Spacer,
			l.logger.TimestampFormat,
			l.instance,
			level,
			l.logger.ShowLogSystemName,
			l.logger.BypassJobChannelFilledWarning,
			l.logger.StructuredLogging,
			l.structuredFields)
	}
	logFieldsPool.Put(l)
}

// stagef stages a log event
func (l *logFields) stagef(header string, data string, v ...interface{}) {
	if l == nil {
		return
	}
	if level := l.enabled(header); level != "" {
		l.output.StageLogEvent(func() string { return fmt.Sprintf(data, v...) },
			header,
			l.name,
			l.logger.Spacer,
			l.logger.TimestampFormat,
			l.instance,
			level,
			l.logger.ShowLogSystemName,
			l.logger.BypassJobChannelFilledWarning,
			l.logger.StructuredLogging,
			l.structuredFields)
	}
	logFieldsPool.Put(l)
}

// WithFields allows the user to add fields to a structured log output
func WithFields(sl *SubLogger, structuredFields map[string]interface{}) *logFields {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return nil
	}
	fields.structuredFields = structuredFields
	return fields
}

func (l *logFields) Error(data string) {
	mu.RLock()
	defer mu.RUnlock()
	l.stage(l.logger.ErrorHeader, data)
}

func (l *logFields) Errorln(data ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.ErrorHeader, data...)
}

func (l *logFields) Errorf(data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.ErrorHeader, data, v...)
}

func (l *logFields) Warn(data string) {
	mu.RLock()
	defer mu.RUnlock()
	l.stage(l.logger.WarnHeader, data)
}

func (l *logFields) Warnln(data ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.WarnHeader, data...)
}

func (l *logFields) Warnf(data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.WarnHeader, data, v...)
}

func (l *logFields) Info(data string) {
	mu.RLock()
	defer mu.RUnlock()
	l.stage(l.logger.InfoHeader, data)
}

func (l *logFields) Infoln(data ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.InfoHeader, data...)
}

func (l *logFields) Infof(data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.InfoHeader, data, v...)
}

func (l *logFields) Debug(data string) {
	mu.RLock()
	defer mu.RUnlock()
	l.stage(l.logger.DebugHeader, data)
}

func (l *logFields) Debugln(data ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stageln(l.logger.DebugHeader, data...)
}

func (l *logFields) Debugf(data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	l.stagef(l.logger.DebugHeader, data, v...)
}

// WithContext allows the user to add a context to a structured log output
func WithContext(ctx context.Context, sl *SubLogger) context.Context {
	return context.WithValue(ctx, ContextValue, sl)
}

// ContextValue is the key for the context value
var ContextValue = contextKey("logger")

// contextKey is a custom type for the context key
type contextKey string
