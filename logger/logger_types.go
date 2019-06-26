package logger

import (
	"io"
	"sync"
)

const timestampFormat = " 02/01/2006 15:04:05 "

const spacer = "|"

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

type SubLoggerConfig struct {
	Name   string `json:"name,omitempty"`
	Level  string `json:"level"`
	Output string `json:"output"`
}

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
	GlobalLogConfig = &Config{}
	subLoggers      = map[string]*subLogger{}
	eventPool       = &sync.Pool{
		New: func() interface{} {
			return &LogEvent{
				data: make([]byte, 0, 80),
			}
		},
	}

	LogPath string

	Global           *subLogger
	SubSystemConnMgr *subLogger
	SubSystemCommMgr *subLogger
	SubSystemConfMgr *subLogger
	SubSystemOrdrMgr *subLogger
	SubSystemPortMgr *subLogger
	SubSystemSyncMgr *subLogger
	SubSystemTimeMgr *subLogger
	SubSystemWsocMgr *subLogger
	SubSystemEvntMgr *subLogger

	SubSystemExchSys *subLogger
	SubSystemGrpcSys *subLogger
	SubSystemRestSys *subLogger

	SubSystemTicker    *subLogger
	SubSystemOrderBook *subLogger
)
