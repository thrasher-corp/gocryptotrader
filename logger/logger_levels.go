package logger

import (
	"log"
	"strings"
)

func logLevel() {
	clearAllLoggers()
	enabledLevels := strings.Split(Logger.Level, "|")

	for x := range enabledLevels {
		switch level := enabledLevels[x]; level {
		case "DEBUG":
			debugLogger.SetOutput(logOutput)
			debugLogger.SetFlags(log.Ldate | log.Ltime)
		case "INFO":
			infoLogger.SetOutput(logOutput)
			infoLogger.SetFlags(log.Ldate | log.Ltime)
		case "WARN":
			warnLogger.SetOutput(logOutput)
			warnLogger.SetFlags(log.Ldate | log.Ltime)
		case "ERROR":
			errorLogger.SetOutput(logOutput)
			errorLogger.SetFlags(log.Ldate | log.Ltime)
		case "FATAL":
			fatalLogger.SetOutput(logOutput)
			fatalLogger.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		default:
			continue
		}
	}
}
