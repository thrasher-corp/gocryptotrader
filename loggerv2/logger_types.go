package loggerv2

import (
	"io"
	"sync"
)

const timestampFormat = " 02/01/2006 15:04:05 "

const spacer = "|"

type LoggerConfig struct {
	Enabled          *bool            `json:"enabled"`
	AdvancedSettings advancedSettings `json:"advancedSettings"`
	SubLoggers       []subLoggers     `json:"subloggers"`
}

type headers struct {
	Info  string `json:"info"`
	Warn  string `json:"warn"`
	Debug string `json:"debug"`
	Error string `json:"error"`
}

type advancedSettings struct {
	Spacer          string  `json:"spacer"`
	TimeStampFormat string  `json:"timeStampFormat"`
	Headers         headers `json:"headers"`
}

type subLoggers struct {
	Name   string `json:"name"`
	Level  string `json:"level"`
	Output string `json:"output"`
}

type Logger struct {
	Timestamp                                        string
	InfoHeader, ErrorHeader, DebugHeader, WarnHeader string
	Spacer                                           string
}

type subLogger struct {
	Info, Debug, Warn, Error bool
	output                   io.Writer
}

type LogEvent struct {
	data   []byte
	output io.Writer
	mu     sync.Mutex
}

type multiWriter struct {
	writers []io.Writer
	mu      sync.Mutex
}

var (
	logger           = &Logger{}
	GlobalLogConfig  = &LoggerConfig{}
	subsystemLoggers = map[string]subLogger{}
	eventPool        = &sync.Pool{
		New: func() interface{} {
			return &LogEvent{
				data: make([]byte, 0, 80),
			}
		},
	}

	mw = MultiWriter()
)
