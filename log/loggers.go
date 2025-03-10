package log

import (
	"fmt"
	"log"
)

// Infoln is a logging function that takes a sublogger and an arbitrary number
// of any arguments. This writes to configured io.Writer(s) as an
// information message using default formats for its operands. A new line is
// automatically added to the output.
func Infoln(sl *SubLogger, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.stageln(f.logger.InfoHeader, a...)
	}
}

// InfolnWithFields is a logging function that takes a sublogger, additional
// structured logging fields and an arbitrary number of any arguments.
// This writes to configured io.Writer(s) as an information message using
// default formats for its operands. A new line is automatically added to the
// output. If structured logging is not enabled, the fields will be ignored.
func InfolnWithFields(sl *SubLogger, extra ExtraFields, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stageln(f.logger.InfoHeader, a...)
	}
}

// Infof is a logging function that takes a sublogger, a format string along
// with optional arguments. This writes to configured io.Writer(s) as an
// information message which formats according to the format specifier.
// A new line is automatically added to the output.
func Infof(sl *SubLogger, format string, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		sl.getFields().stagef(f.logger.InfoHeader, format, a...)
	}
}

// InfofWithFields is a logging function that takes a sublogger, additional
// structured logging fields, a format string along with optional arguments.
// This writes to configured io.Writer(s) as an information message which
// formats according to the format specifier. A new line is automatically added
// to the output. If structured logging is not enabled, the fields will be
// ignored.
func InfofWithFields(sl *SubLogger, extra ExtraFields, format string, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stagef(f.logger.InfoHeader, format, a...)
	}
}

// Debugln is a logging function that takes a sublogger and an arbitrary number
// of any arguments. This writes to configured io.Writer(s) as an
// debug message using default formats for its operands. A new line is
// automatically added to the output.
func Debugln(sl *SubLogger, v ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.stageln(f.logger.DebugHeader, v...)
	}
}

// DebuglnWithFields is a logging function that takes a sublogger, additional
// structured logging fields and an arbitrary number of any arguments.
// This writes to configured io.Writer(s) as an debug message using default
// formats for its operands. A new line is automatically added to the
// output. If structured logging is not enabled, the fields will be ignored.
func DebuglnWithFields(sl *SubLogger, extra ExtraFields, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stageln(f.logger.DebugHeader, a...)
	}
}

// Debugf is a logging function that takes a sublogger, a format string along
// with optional arguments. This writes to configured io.Writer(s) as an
// debug message which formats according to the format specifier. A new line is
// automatically added to the output.
func Debugf(sl *SubLogger, data string, v ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		sl.getFields().stagef(f.logger.DebugHeader, data, v...)
	}
}

// DebugfWithFields is a logging function that takes a sublogger, additional
// structured logging fields, a format string along with optional arguments.
// This writes to configured io.Writer(s) as an debug message which formats
// according to the format specifier. A new line is automatically added to the
// output. If structured logging is not enabled, the fields will be ignored.
func DebugfWithFields(sl *SubLogger, extra ExtraFields, format string, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stagef(f.logger.DebugHeader, format, a...)
	}
}

// Warnln is a logging function that takes a sublogger and an arbitrary number
// of any arguments. This writes to configured io.Writer(s) as an
// warning message using default formats for its operands. A new line is
// automatically added to the output.
func Warnln(sl *SubLogger, v ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.stageln(f.logger.WarnHeader, v...)
	}
}

// WarnlnWithFields is a logging function that takes a sublogger, additional
// structured logging fields and an arbitrary number of any arguments.
// This writes to configured io.Writer(s) as an warning message using default
// formats for its operands. A new line is automatically added to the
// output. If structured logging is not enabled, the fields will be ignored.
func WarnlnWithFields(sl *SubLogger, extra ExtraFields, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stageln(f.logger.WarnHeader, a...)
	}
}

// Warnf is a logging function that takes a sublogger, a format string along
// with optional arguments. This writes to configured io.Writer(s) as an
// warning message which formats according to the format specifier. A new line
// is automatically added to the output.
func Warnf(sl *SubLogger, data string, v ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		sl.getFields().stagef(f.logger.WarnHeader, data, v...)
	}
}

// WarnfWithFields is a logging function that takes a sublogger, additional
// structured logging fields, a format string along with optional arguments.
// This writes to configured io.Writer(s) as an warning message which formats
// according to the format specifier. A new line is automatically added to the
// output. If structured logging is not enabled, the fields will be ignored.
func WarnfWithFields(sl *SubLogger, extra ExtraFields, format string, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stagef(f.logger.WarnHeader, format, a...)
	}
}

// Errorln is a logging function that takes a sublogger and an arbitrary number
// of any arguments. This writes to configured io.Writer(s) as an
// error message using default formats for its operands. A new line is
// automatically added to the output.
func Errorln(sl *SubLogger, v ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.stageln(f.logger.ErrorHeader, v...)
	}
}

// ErrorlnWithFields is a logging function that takes a sublogger, additional
// structured logging fields and an arbitrary number of any arguments.
// This writes to configured io.Writer(s) as an error message using default
// formats for its operands. A new line is automatically added to the
// output. If structured logging is not enabled, the fields will be ignored.
func ErrorlnWithFields(sl *SubLogger, extra ExtraFields, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stageln(f.logger.ErrorHeader, a...)
	}
}

// Errorf is a logging function that takes a sublogger, a format string along
// with optional arguments. This writes to configured io.Writer(s) as an
// error message which formats according to the format specifier. A new line
// is automatically added to the output.
func Errorf(sl *SubLogger, data string, v ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		sl.getFields().stagef(f.logger.ErrorHeader, data, v...)
	}
}

// ErrorfWithFields is a logging function that takes a sublogger, additional
// structured logging fields, a format string along with optional arguments.
// This writes to configured io.Writer(s) as an error message which formats
// according to the format specifier. A new line is automatically added to the
// output. If structured logging is not enabled, the fields will be ignored.
func ErrorfWithFields(sl *SubLogger, extra ExtraFields, format string, a ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if f := sl.getFields(); f != nil {
		f.structuredFields = extra
		f.stagef(f.logger.ErrorHeader, format, a...)
	}
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}

// enabled checks if the log level is enabled
func (l *fields) enabled(header string) string {
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
func (l *fields) stage(header string, deferFunc deferral) {
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

// stageln logs a message with the given header and arguments. It uses the
// custom log hook if set, otherwise falls back to the library's internal log
// system.
func (l *fields) stageln(header string, a ...any) {
	if customLogHook != nil && customLogHook(header, l.name, a...) {
		return
	}
	l.stage(header, func() string { return fmt.Sprint(a...) })
}

// stagef logs a formatted message with the given header and arguments. It uses
// the custom log hook if set, otherwise falls back to the library's internal
// log system.
func (l *fields) stagef(header, format string, a ...any) {
	if customLogHook != nil && customLogHook(header, l.name, fmt.Sprintf(format, a...)) {
		return
	}
	l.stage(header, func() string { return fmt.Sprintf(format, a...) })
}
