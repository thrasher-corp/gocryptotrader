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
	errSubloggerConfigIsNil              = errors.New("sublogger config is nil")
	errUnhandledOutputWriter             = errors.New("unhandled output writer")
	errLogPathIsEmpty                    = errors.New("log path is empty")
	errConfigNil                         = errors.New("config is nil")
	errFileLoggingNotConfiguredCorrectly = errors.New("file logging not configured correctly")
)

// getWriters returns a new multi writer holder from sub logger configuration.
// Note: Calling function must have mutex lock in place.
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
			if !fileLoggingConfiguredCorrectly {
				return nil, errFileLoggingNotConfiguredCorrectly
			}
			writer = globalLogFile
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
func SetGlobalLogConfig(incoming *Config) error {
	if incoming == nil {
		return errConfigNil
	}
	var fileConf loggerFileConfig
	if incoming.LoggerFileConfig != nil {
		fileConf = *incoming.LoggerFileConfig
	}
	subs := make([]SubLoggerConfig, len(incoming.SubLoggers))
	copy(subs, incoming.SubLoggers)
	mu.Lock()
	defer mu.Unlock()
	globalLogConfig.SubLoggerConfig = incoming.SubLoggerConfig
	globalLogConfig.Enabled = convert.BoolPtr(incoming.Enabled != nil && *incoming.Enabled)
	globalLogConfig.LoggerFileConfig = &fileConf
	globalLogConfig.AdvancedSettings = incoming.AdvancedSettings
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

// configureSubLogger configures a new sub logger. Note: Calling function must
// have mutex lock in place.
func configureSubLogger(subLogger, levels string, output *multiWriterHolder) error {
	logPtr, found := SubLoggers[subLogger]
	if !found {
		return fmt.Errorf("sub logger %v not found", subLogger)
	}

	err := logPtr.setOutput(output)
	if err != nil {
		return err
	}
	logPtr.setLevels(splitLevel(levels))
	SubLoggers[subLogger] = logPtr
	return nil
}

// SetupSubLoggers configure all sub loggers with provided configuration values
func SetupSubLoggers(s []SubLoggerConfig) error {
	mu.Lock()
	defer mu.Unlock()
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

// SetupGlobalLogger setup the global loggers with the default global config
// values.
func SetupGlobalLogger(botName string, structuredOutput bool) error {
	mu.Lock()
	defer mu.Unlock()

	if fileLoggingConfiguredCorrectly {
		globalLogFile = &Rotate{
			FileName: globalLogConfig.LoggerFileConfig.FileName,
			MaxSize:  globalLogConfig.LoggerFileConfig.MaxSize,
			Rotate:   globalLogConfig.LoggerFileConfig.Rotate,
		}
	}

	writers, err := getWriters(&globalLogConfig.SubLoggerConfig)
	if err != nil {
		return err
	}

	for _, subLogger := range SubLoggers {
		subLogger.setLevels(splitLevel(globalLogConfig.Level))
		subLogger.structuredLogging = structuredOutput
		err = subLogger.setOutput(writers)
		if err != nil {
			return err
		}
		subLogger.botName = botName
	}
	logger = newLogger(globalLogConfig, botName)
	return nil
}

// SetFileLoggingState can set file logging state if it is correctly configured
// or not. This will bypass the ability to log to file if set as false.
func SetFileLoggingState(correctlyConfigured bool) {
	mu.Lock()
	fileLoggingConfiguredCorrectly = correctlyConfigured
	mu.Unlock()
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

// registerNewSubLogger registers a new sub logger. Note: Calling function must
// have mutex lock in place.
func registerNewSubLogger(subLogger string) *SubLogger {
	tempHolder, err := getWriters(&SubLoggerConfig{
		Name:   strings.ToUpper(subLogger),
		Level:  "INFO|WARN|DEBUG|ERROR",
		Output: "stdout",
	})
	if err != nil {
		return nil
	}

	temp := &SubLogger{
		name:              strings.ToUpper(subLogger),
		output:            tempHolder,
		levels:            splitLevel("INFO|WARN|DEBUG|ERROR"),
		botName:           logger.botName,
		structuredLogging: globalLogConfig != nil && globalLogConfig.AdvancedSettings.StructuredLogging,
	}
	SubLoggers[subLogger] = temp
	return temp
}

// register all loggers at package init()
func init() {
	// Start persistent worker to handle logs
	go loggerWorker()

	mu.Lock()
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
	mu.Unlock()
}
