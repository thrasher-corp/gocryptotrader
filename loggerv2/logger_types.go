package loggerv2

import (
	"io"
	"sync"
)

const timestampFormat = "| 02/01/2006 15:04:05 | "

type Logger struct {
	InfoWriter, ErrorWriter, DebugWriter, WarnWriter io.Writer
	Timestamp                                        string
	InfoHeader, ErrorHeader, DebugHeader, WarnHeader string
}

type subLogger struct {
	Info, Debug, Warn, Error bool
}

var subsystemLoggers = map[string]subLogger{}

type LogEvent struct {
	data   []byte
	output io.Writer
	mu     sync.Mutex
}

var eventPool = &sync.Pool{
	New: func() interface{} {
		return &LogEvent{
			data: make([]byte, 0, 120),
		}
	},
}
