package log

import (
	"fmt"
	"log"
)

// Info takes a pointer subLogger struct and string sends to StageLogEvent
func Info(sl *SubLogger, data string) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.info {
		fields.output.StageLogEvent(func() string { return data },
			fields.logger.InfoHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Infoln takes a pointer subLogger struct and interface sends to StageLogEvent
func Infoln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.info {
		fields.output.StageLogEvent(func() string { return fmt.Sprintln(v...) },
			fields.logger.InfoHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Infof takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Infof(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.info {
		fields.output.StageLogEvent(func() string { return fmt.Sprintf(data, v...) },
			fields.logger.InfoHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Debug takes a pointer subLogger struct and string sends to StageLogEvent
func Debug(sl *SubLogger, data string) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.debug {
		fields.output.StageLogEvent(func() string { return data },
			fields.logger.DebugHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Debugln takes a pointer subLogger struct, string and interface sends to StageLogEvent
func Debugln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.debug {
		fields.output.StageLogEvent(func() string { return fmt.Sprintln(v...) },
			fields.logger.DebugHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Debugf takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Debugf(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.debug {
		fields.output.StageLogEvent(func() string { return fmt.Sprintf(data, v...) },
			fields.logger.DebugHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Warn takes a pointer subLogger struct & string and sends to StageLogEvent
func Warn(sl *SubLogger, data string) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.warn {
		fields.output.StageLogEvent(func() string { return data },
			fields.logger.WarnHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Warnln takes a pointer subLogger struct & interface formats and sends to StageLogEvent
func Warnln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.warn {
		fields.output.StageLogEvent(func() string { return fmt.Sprintln(v...) },
			fields.logger.WarnHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Warnf takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Warnf(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.warn {
		fields.output.StageLogEvent(func() string { return fmt.Sprintf(data, v...) },
			fields.logger.WarnHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Error takes a pointer subLogger struct & interface formats and sends to StageLogEvent
func Error(sl *SubLogger, data ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.error {
		fields.output.StageLogEvent(func() string { return fmt.Sprint(data...) },
			fields.logger.ErrorHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Errorln takes a pointer subLogger struct, string & interface formats and sends to StageLogEvent
func Errorln(sl *SubLogger, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.error {
		fields.output.StageLogEvent(func() string { return fmt.Sprintln(v...) },
			fields.logger.ErrorHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

// Errorf takes a pointer subLogger struct, string and interface formats sends to StageLogEvent
func Errorf(sl *SubLogger, data string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.error {
		fields.output.StageLogEvent(func() string { return fmt.Sprintf(data, v...) },
			fields.logger.ErrorHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName,
			fields.logger.BypassJobChannelFilledWarning)
	}
	logFieldsPool.Put(fields)
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}
