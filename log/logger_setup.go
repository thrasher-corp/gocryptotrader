package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

var (
	errSubloggerConfigIsNil   = errors.New("sublogger config is nil")
	errUnhandledOutputWriter  = errors.New("unhandled output writer")
	errLoggingStateAlreadySet = errors.New("correct file logging bool state already set")
	errLogPathIsEmpty         = errors.New("log path is empty")
	errConfigNil              = errors.New("config is nil")
)

func getWriters(s *SubLoggerConfig) (*multiWriterHolder, error) {
	if s == nil {
		return nil, errSubloggerConfigIsNil
	}

	outputWriters := strings.Split(s.Output, "|")
	writers := make([]io.Writer, 0, len(outputWriters))
	for x := range outputWriters {
		var writer io.Writer
		switch strings.ToLower(outputWriters[x]) {
		case "stdout", "console":
			writer = os.Stdout
		case "stderr":
			writer = os.Stderr
		case "file":
			if getFileLogginState() {
				writer = globalLogFile
			}
		default:
			// Note: Do not want to add an io.Discard here as this adds
			// additional write calls for no reason.
			return nil, fmt.Errorf("%w: %s", errUnhandledOutputWriter, outputWriters[x])
		}
		writers = append(writers, writer)
	}
	return multiWriter(writers...)
}

// GenDefaultSettings return struct with known sane/working logger settings
func GenDefaultSettings() *Config {
	return &Config{
		Enabled: convert.BoolPtr(true),
		SubLoggerConfig: SubLoggerConfig{
			Level:  "INFO|DEBUG|WARN|ERROR",
			Output: "console",
		},
		LoggerFileConfig: &loggerFileConfig{
			FileName: "log.txt",
			Rotate:   convert.BoolPtr(false),
			MaxSize:  0,
		},
		AdvancedSettings: advancedSettings{
			ShowLogSystemName: convert.BoolPtr(false),
			Spacer:            spacer,
			TimeStampFormat:   timestampFormat,
			Headers: headers{
				Info:  "[INFO]",
				Warn:  "[WARN]",
				Debug: "[DEBUG]",
				Error: "[ERROR]",
			},
		},
	}
}

// SetGlobalLogConfig sets the global config with the supplied config
func SetGlobalLogConfig(config *Config) error {
	if config == nil {
		return errConfigNil
	}
	mu.Lock()
	defer mu.Unlock()
	globalLogConfig = config
	return nil
}

// SetLogPath sets the log path for writing to file
func SetLogPath(newLogPath string) error {
	if newLogPath == "" {
		return errLogPathIsEmpty
	}
	mu.Lock()
	defer mu.Unlock()
	logPath = newLogPath
	return nil
}

// GetLogPath returns path of log file
func GetLogPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return logPath
}

func configureSubLogger(subLogger, levels string, output *multiWriterHolder) error {
	mu.Lock()
	defer mu.Unlock()
	logPtr, found := SubLoggers[subLogger]
	if !found {
		return fmt.Errorf("sub logger %v not found", subLogger)
	}

	logPtr.SetOutput(output)
	logPtr.SetLevels(splitLevel(levels))
	SubLoggers[subLogger] = logPtr
	return nil
}

// SetupSubLoggers configure all sub loggers with provided configuration values
func SetupSubLoggers(s []SubLoggerConfig) error {
	for x := range s {
		output, err := getWriters(&s[x])
		if err != nil {
			return err
		}
		err = configureSubLogger(strings.ToUpper(s[x].Name), s[x].Level, output)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetupGlobalLogger setup the global loggers with the default global config values
func SetupGlobalLogger() error {
	mu.Lock()
	defer mu.Unlock()

	if fileLoggingConfiguredCorrectly {
		globalLogFile = &Rotate{
			FileName: globalLogConfig.LoggerFileConfig.FileName,
			MaxSize:  globalLogConfig.LoggerFileConfig.MaxSize,
			Rotate:   globalLogConfig.LoggerFileConfig.Rotate,
		}
	}

	for _, subLogger := range SubLoggers {
		subLogger.SetLevels(splitLevel(globalLogConfig.Level))
		writers, err := getWriters(&globalLogConfig.SubLoggerConfig)
		if err != nil {
			return err
		}
		subLogger.SetOutput(writers)
	}
	logger = newLogger(globalLogConfig)
	return nil
}

// SetFileLoggingState can set file logging state if it is correctly configured
// or not. This will bypass the ability to log to file if set as false.
func SetFileLoggingState(correctlyConfigured bool) error {
	mu.Lock()
	defer mu.Unlock()

	if fileLoggingConfiguredCorrectly == correctlyConfigured {
		return fmt.Errorf("%w as %v", errLoggingStateAlreadySet, correctlyConfigured)
	}
	fileLoggingConfiguredCorrectly = correctlyConfigured
	return nil
}

func getFileLogginState() bool {
	mu.RLock()
	defer mu.RUnlock()
	return fileLoggingConfiguredCorrectly
}

func splitLevel(level string) (l Levels) {
	enabledLevels := strings.Split(level, "|")
	for x := range enabledLevels {
		switch enabledLevels[x] {
		case "DEBUG":
			l.Debug = true
		case "INFO":
			l.Info = true
		case "WARN":
			l.Warn = true
		case "ERROR":
			l.Error = true
		}
	}
	return
}

func registerNewSubLogger(subLogger string) *SubLogger {
	tempHolder, err := getWriters(&SubLoggerConfig{
		Name:   strings.ToUpper(subLogger),
		Level:  "INFO|WARN|DEBUG|ERROR",
		Output: "stdout"})
	if err != nil {
		return nil
	}

	temp := &SubLogger{
		name:   strings.ToUpper(subLogger),
		output: tempHolder,
		levels: splitLevel("INFO|WARN|DEBUG|ERROR"),
	}
	mu.Lock()
	SubLoggers[subLogger] = temp
	mu.Unlock()
	return temp
}

// register all loggers at package init()
func init() {
	// Start persistent worker to handle logs
	workerWg.Add(1)
	go loggerWorker()

	Global = registerNewSubLogger("LOG")

	ConnectionMgr = registerNewSubLogger("CONNECTION")
	CommunicationMgr = registerNewSubLogger("COMMS")
	APIServerMgr = registerNewSubLogger("API")
	ConfigMgr = registerNewSubLogger("CONFIG")
	DatabaseMgr = registerNewSubLogger("DATABASE")
	DataHistory = registerNewSubLogger("DATAHISTORY")
	OrderMgr = registerNewSubLogger("ORDER")
	PortfolioMgr = registerNewSubLogger("PORTFOLIO")
	SyncMgr = registerNewSubLogger("SYNC")
	TimeMgr = registerNewSubLogger("TIMEKEEPER")
	GCTScriptMgr = registerNewSubLogger("GCTSCRIPT")
	WebsocketMgr = registerNewSubLogger("WEBSOCKET")
	EventMgr = registerNewSubLogger("EVENT")
	DispatchMgr = registerNewSubLogger("DISPATCH")

	RequestSys = registerNewSubLogger("REQUESTER")
	ExchangeSys = registerNewSubLogger("EXCHANGE")
	GRPCSys = registerNewSubLogger("GRPC")
	RESTSys = registerNewSubLogger("REST")

	Ticker = registerNewSubLogger("TICKER")
	OrderBook = registerNewSubLogger("ORDERBOOK")
	Trade = registerNewSubLogger("TRADE")
	Fill = registerNewSubLogger("FILL")
	Currency = registerNewSubLogger("CURRENCY")
}
