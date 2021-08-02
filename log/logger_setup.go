package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func getWriters(s *SubLoggerConfig) io.Writer {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	outputWriters := strings.Split(s.Output, "|")
	for x := range outputWriters {
		switch outputWriters[x] {
		case "stdout", "console":
			m.Add(os.Stdout)
		case "stderr":
			m.Add(os.Stderr)
		case "file":
			if FileLoggingConfiguredCorrectly {
				m.Add(GlobalLogFile)
			}
		default:
			m.Add(ioutil.Discard)
		}
	}
	return m
}

// GenDefaultSettings return struct with known sane/working logger settings
func GenDefaultSettings() (log Config) {
	log = Config{
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
	return
}

func configureSubLogger(logger, levels string, output io.Writer) error {
	found, logPtr := validSubLogger(logger)
	if !found {
		return fmt.Errorf("logger %v not found", logger)
	}

	logPtr.output = output

	logPtr.Levels = splitLevel(levels)
	subLoggers[logger] = logPtr

	return nil
}

// SetupSubLoggers configure all sub loggers with provided configuration values
func SetupSubLoggers(s []SubLoggerConfig) {
	for x := range s {
		output := getWriters(&s[x])
		err := configureSubLogger(strings.ToUpper(s[x].Name), s[x].Level, output)
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

	for x := range subLoggers {
		subLoggers[x].Levels = splitLevel(GlobalLogConfig.Level)
		subLoggers[x].output = getWriters(&GlobalLogConfig.SubLoggerConfig)
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

func registerNewSubLogger(logger string) *SubLogger {
	temp := SubLogger{
		name:   strings.ToUpper(logger),
		output: os.Stdout,
	}

	temp.Levels = splitLevel("INFO|WARN|DEBUG|ERROR")
	subLoggers[logger] = &temp

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
}
