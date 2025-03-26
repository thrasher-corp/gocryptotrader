package log

import (
	"io"
	"sync"
)

const (
	timestampFormat = " 02/01/2006 15:04:05 "
	spacer          = " | "
	// DefaultMaxFileSize for logger rotation file
	DefaultMaxFileSize int64 = 100

	// defaultBufferCapacity has 200kb of memory per buffer, there has been some
	// instances where it was 3/4 of this. This size so as to not need a resize.
	defaultBufferCapacity     = 200000
	defaultJobChannelCapacity = 10000
)

var (
	logger = Logger{}
	// FileLoggingConfiguredCorrectly flag set during config check if file logging meets requirements
	fileLoggingConfiguredCorrectly bool
	// GlobalLogConfig holds global configuration options for logger
	globalLogConfig = &Config{}
	// GlobalLogFile hold global configuration options for file logger
	globalLogFile = &Rotate{}

	jobsPool    = &sync.Pool{New: func() any { return new(job) }}
	jobsChannel = make(chan *job, defaultJobChannelCapacity)

	// Note: Logger state within logFields will be persistent until it's garbage
	// collected. This is a little bit more efficient.
	logFieldsPool = &sync.Pool{New: func() any { return &fields{logger: logger} }}

	// LogPath system path to store log files in
	logPath string

	// read/write mutex for logger
	mu = &sync.RWMutex{}
)

type job struct {
	Writers           []io.Writer
	fn                deferral
	Header            string
	SubLoggerName     string
	Spacer            string
	TimestampFormat   string
	ShowLogSystemName bool
	Instance          string
	StructuredFields  map[Key]any
	StructuredLogging bool
	Severity          string
	Passback          chan<- struct{}
}

// Config holds configuration settings loaded from bot config
type Config struct {
	Enabled *bool `json:"enabled"`
	SubLoggerConfig
	LoggerFileConfig *loggerFileConfig `json:"fileSettings,omitempty"`
	AdvancedSettings advancedSettings  `json:"advancedSettings"`
	SubLoggers       []SubLoggerConfig `json:"subloggers,omitempty"`
}

type advancedSettings struct {
	ShowLogSystemName             *bool   `json:"showLogSystemName"`
	Spacer                        string  `json:"spacer"`
	TimeStampFormat               string  `json:"timeStampFormat"`
	Headers                       headers `json:"headers"`
	BypassJobChannelFilledWarning bool    `json:"bypassJobChannelFilledWarning"`
	StructuredLogging             bool    `json:"structuredLogging"`
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

type loggerFileConfig struct {
	FileName string `json:"filename,omitempty"`
	Rotate   *bool  `json:"rotate,omitempty"`
	MaxSize  int64  `json:"maxsize,omitempty"`
}

// Logger each instance of logger settings
type Logger struct {
	ShowLogSystemName                                bool
	BypassJobChannelFilledWarning                    bool
	TimestampFormat                                  string
	InfoHeader, ErrorHeader, DebugHeader, WarnHeader string
	Spacer                                           string
	Level                                            string
	botName                                          string
}

// Levels flags for each sub logger type
type Levels struct {
	Info, Debug, Warn, Error bool
}

type multiWriterHolder struct {
	writers []io.Writer
}

// ExtraFields is a map of key value pairs that can be added to a structured
// log output.
type ExtraFields map[Key]any

// Key is used for structured logging fields to ensure no collisions occur.
// Unexported keys are default fields which cannot be overwritten.
type Key string
