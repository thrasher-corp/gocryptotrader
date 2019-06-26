package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func getWriters(s *SubLoggerConfig) io.Writer {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	outputWriters := strings.Split(s.Output, "|")
	for x := range outputWriters {
		switch outputWriters[x] {
		case "stdout":
			m.Add(os.Stdout)
		case "stderr":
			m.Add(os.Stderr)
		case "file":
			temp, err := createFileHandle(s.Name)
			if err != nil {
				fmt.Printf("File handle error %v", err)
			}
			m.Add(temp)
		default:
			m.Add(ioutil.Discard)
		}
	}
	return m
}

func GenDefaultSettings() (log Config) {
	t := func(t bool) *bool { return &t }(true)
	log = Config{
		Enabled: t,
		SubLoggerConfig: SubLoggerConfig{
			Level:  "INFO|DEBUG|WARN|ERROR",
			Output: "stdout",
		},
		AdvancedSettings: advancedSettings{
			Spacer:          " | ",
			TimeStampFormat: timestampFormat,
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

func configureSubLogger(logger, levels string, output io.Writer) {
	found, logPtr := validSubLogger(logger)
	if !found {
		return
	}

	logPtr.output = output

	logPtr.levels = splitLevel(levels)
	subLoggers[logger] = logPtr
}

func SetupSubLoggers(s []SubLoggerConfig) {
	for x := range s {
		output := getWriters(&s[x])
		configureSubLogger(s[x].Name, s[x].Level, output)
	}
}

func SetupGlobalLogger() {
	for x := range subLoggers {
		subLoggers[x].levels = splitLevel(GlobalLogConfig.Level)
		subLoggers[x].output = getWriters(&GlobalLogConfig.SubLoggerConfig)
	}

	logger = newLogger(GlobalLogConfig)
}

func splitLevel(level string) (l levels) {
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

func registerNewSubLogger(logger string) *subLogger {
	temp := subLogger{
		name:   logger,
		output: os.Stdout,
	}

	temp.levels = splitLevel("INFO|WARN|DEBUG|ERROR")

	subLoggers[logger] = &temp

	return &temp
}

// register all loggers at package init()
func init() {
	Global = registerNewSubLogger("log")

	SubSystemConnMgr = registerNewSubLogger("connection")
	SubSystemCommMgr = registerNewSubLogger("comms")
	SubSystemConfMgr = registerNewSubLogger("config")
	SubSystemOrdrMgr = registerNewSubLogger("order")
	SubSystemPortMgr = registerNewSubLogger("portfolio")
	SubSystemSyncMgr = registerNewSubLogger("sync")
	SubSystemTimeMgr = registerNewSubLogger("timekeeper")
	SubSystemWsocMgr = registerNewSubLogger("websocket")
	SubSystemEvntMgr = registerNewSubLogger("event")

	SubSystemExchSys = registerNewSubLogger("exchange")
	SubSystemGrpcSys = registerNewSubLogger("grpc")
	SubSystemRestSys = registerNewSubLogger("rest")

	SubSystemTicker = registerNewSubLogger("ticker")
	SubSystemOrderBook = registerNewSubLogger("orderbook")
}
