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
	errSubloggerConfigIsNil  = errors.New("sublogger config is nil")
	errUnhandledOutputWriter = errors.New("unhandled output writer")
)

func getWriters(s *SubLoggerConfig) (io.Writer, error) {
	if s == nil {
		return nil, errSubloggerConfigIsNil
	}
	mw, err := MultiWriter()
	if err != nil {
		return nil, err
	}
	outputWriters := strings.Split(s.Output, "|")
	for x := range outputWriters {
		var writer io.Writer
		switch strings.ToLower(outputWriters[x]) {
		case "stdout", "console":
			writer = os.Stdout
		case "stderr":
			writer = os.Stderr
		case "file":
			if FileLoggingConfiguredCorrectly {
				writer = GlobalLogFile
			}
		default:
			// Note: Do not want to add a ioutil.discard here as this adds
			// additional routines for every write for no reason.
			return nil, fmt.Errorf("%w: %s", errUnhandledOutputWriter, outputWriters[x])
		}
		err = mw.Add(writer)
		if err != nil {
			return nil, err
		}
	}
	return mw, nil
}

// GenDefaultSettings return struct with known sane/working logger settings
func GenDefaultSettings() Config {
	return Config{
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

func configureSubLogger(subLogger, levels string, output io.Writer) error {
	logPtr, found := SubLoggers[subLogger]
	if !found {
		return fmt.Errorf("sub logger %v not found", subLogger)
	}

	logPtr.output = output
	logPtr.Levels = splitLevel(levels)
	SubLoggers[subLogger] = logPtr
	return nil
}

// SetupSubLoggers configure all sub loggers with provided configuration values
func SetupSubLoggers(s []SubLoggerConfig) {
	for x := range s {
		output, err := getWriters(&s[x])
		if err != nil {
			return
		}
		err = configureSubLogger(strings.ToUpper(s[x].Name), s[x].Level, output)
		if err != nil {
			continue
		}
	}
}

// SetupGlobalLogger setup the global loggers with the default global config values
func SetupGlobalLogger() {
	RWM.Lock()
	if FileLoggingConfiguredCorrectly {
		GlobalLogFile = &Rotate{
			FileName: GlobalLogConfig.LoggerFileConfig.FileName,
			MaxSize:  GlobalLogConfig.LoggerFileConfig.MaxSize,
			Rotate:   GlobalLogConfig.LoggerFileConfig.Rotate,
		}
	}

	for x := range SubLoggers {
		SubLoggers[x].Levels = splitLevel(GlobalLogConfig.Level)
		SubLoggers[x].output, _ = getWriters(&GlobalLogConfig.SubLoggerConfig)
	}

	logger = newLogger(GlobalLogConfig)
	RWM.Unlock()
}

func splitLevel(level string) (l Levels) {
	enabledLevels := strings.Split(level, "|")
	for x := range enabledLevels {
		switch level := enabledLevels[x]; level {
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
	temp := SubLogger{
		name:   strings.ToUpper(subLogger),
		output: os.Stdout,
	}

	temp.Levels = splitLevel("INFO|WARN|DEBUG|ERROR")
	SubLoggers[subLogger] = &temp
	return &temp
}

// register all loggers at package init()
func init() {
	Global = registerNewSubLogger("LOG")

	ConnectionMgr = registerNewSubLogger("CONNECTION")
	BackTester = registerNewSubLogger("BACKTESTER")
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
}
