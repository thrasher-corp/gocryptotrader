package logger

import (
	"fmt"
)

func printDetails(sl *subLogger) {
	return
	fmt.Printf("%+v\n", sl)
}

func Info(sl *subLogger, data string) {
	printDetails(sl)
	if !sl.Info {
		return
	}

	logger.newLogEvent(data, logger.InfoHeader, sl.output)
}

func Infoln(sl *subLogger, v ...interface{}) {
	printDetails(sl)
	if !sl.Info {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.InfoHeader, sl.output)
}

func Infof(sl *subLogger, data string, v ...interface{}) {
	if !sl.Info {
		return
	}

	Info(sl, fmt.Sprintf(data, v...))
}

func Debug(sl *subLogger, data string) {
	printDetails(sl)
	if !sl.Debug {
		return
	}

	logger.newLogEvent(data, logger.DebugHeader, sl.output)
}

func Debugln(sl *subLogger, v ...interface{}) {
	printDetails(sl)
	if !sl.Debug {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.DebugHeader, sl.output)
}

func Debugf(sl *subLogger, data string, v ...interface{}) {
	if !sl.Debug {
		return
	}

	Debug(sl, fmt.Sprintf(data, v...))
}

func Warn(sl *subLogger, data string) {
	printDetails(sl)
	if !sl.Warn {
		return
	}

	logger.newLogEvent(data, logger.WarnHeader, sl.output)
}

func Warnln(sl *subLogger, v ...interface{}) {
	printDetails(sl)
	if !sl.Warn {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.WarnHeader, sl.output)
}

func Warnf(sl *subLogger, data string, v ...interface{}) {
	if !sl.Warn {
		return
	}

	Warn(sl, fmt.Sprintf(data, v...))
}

func Error(sl *subLogger, data ...interface{}) {
	printDetails(sl)
	if !sl.Error {
		return
	}

	logger.newLogEvent(fmt.Sprint(data...), logger.ErrorHeader, sl.output)
}

func Errorln(sl *subLogger, v ...interface{}) {
	printDetails(sl)
	if !sl.Error {
		return
	}

	logger.newLogEvent(fmt.Sprintln(v...), logger.ErrorHeader, sl.output)
}

func Errorf(sl *subLogger, data string, v ...interface{}) {
	if !sl.Error {
		return
	}

	Error(sl, fmt.Sprintf(data, v...))
}
