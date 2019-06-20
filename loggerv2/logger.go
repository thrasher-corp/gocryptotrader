package loggerv2

import (
	"io"
	"time"
)

func newLogger(c *LoggerConfig) *Logger {
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
	e.data = append(e.data, []byte(data)...)
	e.mu.Lock()
	e.output.Write(e.data)
	e.mu.Unlock()
	eventPool.Put(e)
}

func CloseLogger() {
	closeAllFiles()
}
