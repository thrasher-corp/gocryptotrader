package logger

import (
	"io"
	"runtime"
	"strconv"
	"time"
)

func newLogger(c *Config) *Logger {
	return &Logger{
		Timestamp:   c.AdvancedSettings.TimeStampFormat,
		Spacer:      c.AdvancedSettings.Spacer,
		ErrorHeader: c.AdvancedSettings.Headers.Error,
		InfoHeader:  c.AdvancedSettings.Headers.Info,
		WarnHeader:  c.AdvancedSettings.Headers.Warn,
		DebugHeader: c.AdvancedSettings.Headers.Debug,
	}
}

func SetupGlobalLogger() {
	logger = newLogger(GlobalLogConfig)
}

func shortPath(path string) string {
	var ls, ps int
	for i, c := range path {
		if c == '/' {
			ps = ls
			ls = i
		}
	}
	if path[ps] == '/' {
		ps++
	}
	return path[ps:]
}

func (l *Logger) newLogEvent(data, header string, w io.Writer) {
	if w == nil {
		return
	}
	e := eventPool.Get().(*LogEvent)
	e.output = w
	e.data = e.data[:0]
	e.data = append(e.data, []byte(header)...)
	e.data = append(e.data, l.Spacer...)
	if l.Timestamp != "" {
		e.data = time.Now().AppendFormat(e.data, l.Timestamp)
	}
	e.data = append(e.data, l.Spacer...)
	_, fn, line, _ := runtime.Caller(3)
	lineByte := []byte(strconv.Itoa(line))
	e.data = append(e.data, shortPath(fn)...)
	e.data = append(e.data, '@')
	e.data = append(e.data, lineByte...)
	e.data = append(e.data, l.Spacer...)
	e.data = append(e.data, []byte(data)...)

	e.output.Write(e.data)
	e.data = (e.data)[:0]
	eventPool.Put(e)
}

func CloseLogger() {
	closeAllFiles()
}

func subSystemData(subsystem string) (sl *subLogger) {
	if val, found := subsystemLoggers[subsystem]; found {
		sl = &val
		return
	}
	temp := subsystemLoggers["log"]
	return &temp
}
