package logger

import "fmt"

func Info(subSystem, data string) {
	sub := subSystemData(subSystem)
	if !sub.Info {
		return
	}

	logger.newLogEvent(data, logger.InfoHeader, sub.output)
}

func Infoln(subSystem string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Info {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.InfoHeader, sub.output)
}

func Infof(subSystem, data string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Info {
		return
	}

	Info(subSystem, fmt.Sprintf(data, v...))
}

func Debug(subSystem, data string) {
	sub := subSystemData(subSystem)
	if !sub.Debug {
		return
	}

	logger.newLogEvent(data, logger.DebugHeader, sub.output)
}

func Debugln(subSystem string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Debug {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.DebugHeader, sub.output)
}

func Debugf(subSystem, data string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Debug {
		return
	}

	Debug(subSystem, fmt.Sprintf(data, v...))
}

func Warn(subSystem, data string) {
	sub := subSystemData(subSystem)
	if !sub.Warn {
		return
	}

	logger.newLogEvent(data, logger.WarnHeader, sub.output)
}

func Warnln(subSystem string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Warn {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.WarnHeader, sub.output)
}

func Warnf(subSystem, data string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Warn {
		return
	}

	Warn(subSystem, fmt.Sprintf(data, v...))
}

func Error(subSystem string, data ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Error {
		return
	}

	logger.newLogEvent(fmt.Sprint(data...), logger.ErrorHeader, sub.output)
}

func Errorln(subSystem string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Error {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.ErrorHeader, sub.output)
}

func Errorf(subSystem, data string, v ...interface{}) {
	sub := subSystemData(subSystem)
	if !sub.Error {
		return
	}

	Error(subSystem, fmt.Sprintf(data, v...))
}
