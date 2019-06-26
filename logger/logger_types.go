package logger

import (
	"io"
	"sync"
)

const timestampFormat = " 02/01/2006 15:04:05 "

const spacer = "|"

// Config holds configuration settings loaded from bot config
type Config struct {
	Enabled *bool `json:"enabled"`
	SubLoggerConfig
	AdvancedSettings advancedSettings  `json:"advancedSettings"`
	SubLoggers       []SubLoggerConfig `json:"subloggers,omitempty"`
}

type advancedSettings struct {
	Spacer          string  `json:"spacer"`
	TimeStampFormat string  `json:"timeStampFormat"`
	Headers         headers `json:"headers"`
}

type headers struct {
	Info  string `json:"info"`
	Warn  string `json:"warn"`
	Debug string `json:"debug"`
	Error string `json:"error"`
}

// SubLoggerConfig holds sub logger configuration settings loaded from bot config
type SubLoggerConfig struct {
	Name   string `json:"name,omitempty"`
	Level  string `json:"level"`
	Output string `json:"output"`
}

// Logger each instance of logger settings
type Logger struct {
	Timestamp                                        string
	InfoHeader, ErrorHeader, DebugHeader, WarnHeader string
	Spacer                                           string
}

type levels struct {
	Info, Debug, Warn, Error bool
}

type subLogger struct {
	name string
	levels
	output io.Writer
}

type LogEvent struct {
	data   []byte
	output io.Writer
}

type multiWriter struct {
	writers []io.Writer
	mu      sync.Mutex
}

var (
	logger          = &Logger{}
	GlobalLogConfig = &Config{} // GlobalLogConfig hold global configuration options for logger
	GlobalLogFile   io.Writer   // GlobalLogFile handle to global log file

	eventPool = &sync.Pool{
		New: func() interface{} {
			return &LogEvent{
				data: make([]byte, 0, 80),
			}
		},
	}

	LogPath string
)
