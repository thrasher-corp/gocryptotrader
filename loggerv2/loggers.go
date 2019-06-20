package loggerv2

import "fmt"

func Info(subSystem, data string) {
	if !subsystemLoggers[subSystem].Info {
		return
	}
	logger.newLogEvent(data, logger.InfoHeader, subsystemLoggers[subSystem].output)
}

func Infoln(subSystem string, v ...interface{}) {
	if !subsystemLoggers[subSystem].Info {
		return
	}
	logger.newLogEvent(fmt.Sprintln(v...), logger.InfoHeader, subsystemLoggers[subSystem].output)
}

func Infof(subSystem, data string, v ...interface{}) {
	if !subsystemLoggers[subSystem].Info {
		return
	}
	Info(subSystem, fmt.Sprintf(data, v...))
}

func Debug(subSystem, data string) {
	if !subsystemLoggers[subSystem].Debug {
		return
	}
	logger.newLogEvent(data, logger.DebugHeader, subsystemLoggers[subSystem].output)
}

func Debugln(subSystem string, v ...interface{}) {
	if !subsystemLoggers[subSystem].Debug {
		return
	}
	logger.newLogEvent(fmt.Sprintln(v...), logger.DebugHeader, subsystemLoggers[subSystem].output)
}

func Debugf(subSystem, data string, v ...interface{}) {
	Debug(subSystem, fmt.Sprintf(data, v...))
}

func Warn(subSystem, data string) {
	if !subsystemLoggers[subSystem].Warn {
		return
	}
	logger.newLogEvent(data, logger.WarnHeader, subsystemLoggers[subSystem].output)
}

func Warnln(subSystem string, v ...interface{}) {
	if !subsystemLoggers[subSystem].Warn {
		return
	}
	logger.newLogEvent(fmt.Sprintln(v...), logger.WarnHeader, subsystemLoggers[subSystem].output)
}

func Warnf(subSystem, data string, v ...interface{}) {
	if !subsystemLoggers[subSystem].Warn {
		return
	}
	Warn(subSystem, fmt.Sprintf(data, v...))
}

func Error(subSystem string, data error) {
	if !subsystemLoggers[subSystem].Error {
		return
	}
	logger.newLogEvent(fmt.Sprint(data), logger.ErrorHeader, subsystemLoggers[subSystem].output)
}

func Errorln(subSystem string, v ...interface{}) {
	if !subsystemLoggers[subSystem].Error {
		return
	}
	logger.newLogEvent(fmt.Sprintln(v...), logger.ErrorHeader, subsystemLoggers[subSystem].output)
}

func Errorf(subSystem, data string, v ...interface{}) {
	if !subsystemLoggers[subSystem].Error {
		return
	}
	Warn(subSystem, fmt.Sprintf(data, v...))
}
