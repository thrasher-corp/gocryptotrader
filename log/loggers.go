package log

import (
	"fmt"
	"log"
)

// Info takes a pointer subLogger struct and string sends to newLogEvent
func Info(sl *SubLogger, data string) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.info {
		fields.output.StageLogEvent(data,
			fields.logger.InfoHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Infoln takes a pointer subLogger struct and interface sends to newLogEvent
func Infoln(sl *SubLogger, v ...interface{}) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.info {
		fields.output.StageLogEvent(fmt.Sprintln(v...),
			fields.logger.InfoHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Infof takes a pointer subLogger struct, string & interface formats and sends to Info()
func Infof(sl *SubLogger, data string, v ...interface{}) {
	Info(sl, fmt.Sprintf(data, v...))
}

// Debug takes a pointer subLogger struct and string sends to multiwriter
func Debug(sl *SubLogger, data string) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.debug {
		fields.output.StageLogEvent(data,
			fields.logger.DebugHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Debugln  takes a pointer subLogger struct, string and interface sends to newLogEvent
func Debugln(sl *SubLogger, v ...interface{}) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.debug {
		fields.output.StageLogEvent(fmt.Sprintln(v...),
			fields.logger.DebugHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Debugf takes a pointer subLogger struct, string & interface formats and sends to Info()
func Debugf(sl *SubLogger, data string, v ...interface{}) {
	Debug(sl, fmt.Sprintf(data, v...))
}

// Warn takes a pointer subLogger struct & string  and sends to newLogEvent()
func Warn(sl *SubLogger, data string) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.warn {
		fields.output.StageLogEvent(data,
			fields.logger.WarnHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Warnln takes a pointer subLogger struct & interface formats and sends to newLogEvent()
func Warnln(sl *SubLogger, v ...interface{}) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.warn {
		fields.output.StageLogEvent(fmt.Sprintln(v...),
			fields.logger.WarnHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Warnf takes a pointer subLogger struct, string & interface formats and sends to Warn()
func Warnf(sl *SubLogger, data string, v ...interface{}) {
	Warn(sl, fmt.Sprintf(data, v...))
}

// Error takes a pointer subLogger struct & interface formats and sends to newLogEvent()
func Error(sl *SubLogger, data ...interface{}) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.error {
		fields.output.StageLogEvent(fmt.Sprint(data...),
			fields.logger.ErrorHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Errorln takes a pointer subLogger struct, string & interface formats and sends to newLogEvent()
func Errorln(sl *SubLogger, v ...interface{}) {
	fields := sl.getFields()
	if fields == nil {
		return
	}
	if fields.error {
		fields.output.StageLogEvent(fmt.Sprintln(v...),
			fields.logger.ErrorHeader,
			fields.name,
			fields.logger.Spacer,
			fields.logger.TimestampFormat,
			fields.logger.ShowLogSystemName)
	}
	logFieldsPool.Put(fields)
}

// Errorf takes a pointer subLogger struct, string & interface formats and sends to Debug()
func Errorf(sl *SubLogger, data string, v ...interface{}) {
	Error(sl, fmt.Sprintf(data, v...))
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}
