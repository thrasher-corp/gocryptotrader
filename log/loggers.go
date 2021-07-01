package log

import (
	"fmt"
	"log"
)

// Info takes a pointer subLogger struct and string sends to newLogEvent
func (sl *subLogger) Info(data string) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.info {
		return
	}

	displayError(fields.logger.newLogEvent(data, fields.logger.InfoHeader, fields.name, fields.output))
}

// Infoln takes a pointer subLogger struct and interface sends to newLogEvent
func (sl *subLogger) Infoln(v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.info {
		return
	}

	displayError(fields.logger.newLogEvent(fmt.Sprintln(v...), fields.logger.InfoHeader, fields.name, fields.output))
}

// Infof takes a pointer subLogger struct, string & interface formats and sends to Info()
func (sl *subLogger) Infof(data string, v ...interface{}) {
	sl.Info(fmt.Sprintf(data, v...))
}

// Debug takes a pointer subLogger struct and string sends to multiwriter
func (sl *subLogger) Debug(data string) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.debug {
		return
	}

	displayError(fields.logger.newLogEvent(data, fields.logger.DebugHeader, fields.name, fields.output))
}

// Debugln  takes a pointer subLogger struct, string and interface sends to newLogEvent
func (sl *subLogger) Debugln(v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.debug {
		return
	}

	displayError(fields.logger.newLogEvent(fmt.Sprintln(v...), fields.logger.DebugHeader, fields.name, fields.output))
}

// Debugf takes a pointer subLogger struct, string & interface formats and sends to Info()
func (sl *subLogger) Debugf(data string, v ...interface{}) {
	sl.Debug(fmt.Sprintf(data, v...))
}

// Warn takes a pointer subLogger struct & string  and sends to newLogEvent()
func (sl *subLogger) Warn(data string) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.warn {
		return
	}

	displayError(fields.logger.newLogEvent(data, fields.logger.WarnHeader, fields.name, fields.output))
}

// Warnln takes a pointer subLogger struct & interface formats and sends to newLogEvent()
func (sl *subLogger) Warnln(v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.warn {
		return
	}

	displayError(fields.logger.newLogEvent(fmt.Sprintln(v...), fields.logger.WarnHeader, fields.name, fields.output))
}

// Warnf takes a pointer subLogger struct, string & interface formats and sends to Warn()
func (sl *subLogger) Warnf(data string, v ...interface{}) {
	sl.Warn(fmt.Sprintf(data, v...))
}

// Error takes a pointer subLogger struct & interface formats and sends to newLogEvent()
func (sl *subLogger) Error(data ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.error {
		return
	}

	displayError(fields.logger.newLogEvent(fmt.Sprint(data...), fields.logger.ErrorHeader, fields.name, fields.output))
}

// Errorln takes a pointer subLogger struct, string & interface formats and sends to newLogEvent()
func (sl *subLogger) Errorln(v ...interface{}) {
	fields := getFields(sl)
	if fields == nil {
		return
	}
	if !fields.error {
		return
	}

	displayError(fields.logger.newLogEvent(fmt.Sprintln(v...), fields.logger.ErrorHeader, fields.name, fields.output))
}

// Errorf takes a pointer subLogger struct, string & interface formats and sends to Debug()
func (sl *subLogger) Errorf(data string, v ...interface{}) {
	sl.Error(fmt.Sprintf(data, v...))
}

func displayError(err error) {
	if err != nil {
		log.Printf("Logger write error: %v\n", err)
	}
}

func enabled() bool {
	RWM.Lock()
	defer RWM.Unlock()
	if GlobalLogConfig == nil || GlobalLogConfig.Enabled == nil {
		return false
	}
	if *GlobalLogConfig.Enabled {
		return true
	}
	return false
}

func getFields(sl *subLogger) *logFields {
	if !enabled() {
		return nil
	}
	if sl == nil {
		return nil
	}
	RWM.RLock()
	defer RWM.RUnlock()
	return &logFields{
		info:   sl.Enabled.Info,
		warn:   sl.Enabled.Warn,
		debug:  sl.Enabled.Debug,
		error:  sl.Enabled.Error,
		name:   sl.name,
		output: sl.output,
		logger: *logger,
	}
}
