package loggerv2

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func newLogger(c *LoggerConfig) *Logger {
	return &Logger{
		WarnWriter:  os.Stdout,
		DebugWriter: os.Stdout,
		ErrorWriter: os.Stderr,
		Timestamp:   c.AdvancedSettings.TimeStampFormat,
		ErrorHeader: c.AdvancedSettings.Headers.Error,
		InfoHeader:  c.AdvancedSettings.Headers.Info,
		WarnHeader:  c.AdvancedSettings.Headers.Warn,
		DebugHeader: c.AdvancedSettings.Headers.Debug,
	}
}

func SetupGlobalLogger() {
	logger = newLogger(GlobalLogConfig)
}

func (l *Logger) newLogEvent(data, header string, w io.Writer) {
	if w == nil {
		return
	}
	e := eventPool.Get().(*LogEvent)
	e.output = w
	e.data = e.data[:0]
	e.data = append(e.data, []byte(header)...)
	e.data = append(e.data, spacer...)
	if l.Timestamp != "" {
		e.data = time.Now().AppendFormat(e.data, l.Timestamp)
	}
	e.data = append(e.data, spacer...)
	e.data = append(e.data, ' ')
	e.data = append(e.data, []byte(data)...)
	e.mu.Lock()
	e.output.Write(e.data)
	e.mu.Unlock()
	eventPool.Put(e)
}

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

func Error(subSystem, data string) {
	if !subsystemLoggers[subSystem].Error {
		return
	}
	logger.newLogEvent(data, logger.ErrorHeader, subsystemLoggers[subSystem].output)
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

func addSubLogger(subsystem, flags string, output io.Writer) {
	temp := subLogger{
		output: output,
	}
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

func SetupSubLogger(s []subLoggers) {
	for x := range s {
		output := getWriter(s[x].Output)
		addSubLogger(s[x].Name, s[x].Level, output)
	}
}

func getWriter(output string) io.Writer {
	switch output {
	case "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	}
	return ioutil.Discard
}
