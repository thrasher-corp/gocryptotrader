package loggerv2

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

func newPosixConsoleLogger() *Logger {
	return &Logger{
		InfoWriter:  os.Stdout,
		WarnWriter:  os.Stdout,
		DebugWriter: os.Stdout,
		ErrorWriter: os.Stderr,
		Timestamp:   timestampFormat,
		ErrorHeader: "\033[31m[ERROR]\033[0m ",
		InfoHeader:  "\033[32m[INFO]\033[0m  ",
		WarnHeader:  "\033[33m[WARN]\033[0m  ",
		DebugHeader: "\033[34m[DEBUG]\033[0m ",
	}
}

func newWindowsConsoleLogger() *Logger {
	return &Logger{
		InfoWriter:  os.Stdout,
		WarnWriter:  os.Stdout,
		DebugWriter: os.Stdout,
		ErrorWriter: os.Stderr,
		Timestamp:   timestampFormat,
		ErrorHeader: "[ERROR] ",
		InfoHeader:  "[INFO] ",
		WarnHeader:  "[WARN] ",
		DebugHeader: "[DEBUG] ",
	}
}

func NewLogger() *Logger {
	if runtime.GOOS == "windows" {
		return newWindowsConsoleLogger()
	}
	return newPosixConsoleLogger()
}

var logger = NewLogger()

func (l *Logger) newLogEvent(data, header string, w io.Writer) {
	if w == nil {
		return
	}
	e := eventPool.Get().(*LogEvent)
	e.output = w
	e.data = e.data[:0]
	e.data = append(e.data, []byte(header)...)
	if l.Timestamp != "" {
		e.data = time.Now().AppendFormat(e.data, l.Timestamp)
	}
	e.data = append(e.data, []byte(data)...)
	e.data = append(e.data, '\n')
	e.mu.Lock()
	e.output.Write(e.data)
	e.mu.Unlock()
	eventPool.Put(e)
}

func Info(subSystem, data string) {
	if !subsystemLoggers[subSystem].Info {
		return
	}
	logger.newLogEvent(data, logger.InfoHeader, logger.InfoWriter)
}

func Infof(subSystem, data string, v ...interface{}) {
	Info(subSystem, fmt.Sprintf(data, v...))
}

func Debug(subSystem, data string) {
	logger.newLogEvent(data, logger.DebugHeader, logger.DebugWriter)
}

func Debugf(subSystem, data string, v ...interface{}) {
	Debug(subSystem, fmt.Sprintf(data, v...))
}

func Warn(subSystem, data string) {
	logger.newLogEvent(data, logger.WarnHeader, logger.WarnWriter)
}

func Warnf(subSystem, data string, v ...interface{}) {
	Warn(subSystem, fmt.Sprintf(data, v...))
}

func addSubLogger(subsystem, flags string) {
	temp := subLogger{}
	enabledLevels := strings.Split(flags, "|")
	for x := range enabledLevels {
		switch level := enabledLevels[x]; level {
		case "DEBUG":
			temp.Debug = true
		case "INFO":
			temp.Info = true
		case "WARN":
			temp.Warn = true
		case "ERROR":
			temp.Error = true
		}
	}
	subsystemLoggers[subsystem] = temp
}

func init() {
	addSubLogger("syslog", "INFO|DEBUG|WARN|ERROR")
}
