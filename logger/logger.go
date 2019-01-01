package logger

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

func init() {
	setDefaultOutputs()
}

// SetupLogger configure logger instance with user provided settings
func SetupLogger() {
	if Logger.Enabled {
		setupOutputs()
		logLevel()
		if Logger.ColourOutput {
			colourOutput()
		}
	} else {
		clearAllLoggers()
	}

}

// setDefaultOutputs() this setups some defaults used by the logger that way it can be used without any user configuration
func setDefaultOutputs() {
	debugLogger = log.New(os.Stdout,
		"[DEBUG]: ",
		log.Ldate|log.Ltime)

	infoLogger = log.New(os.Stdout,
		"[INFO]: ",
		log.Ldate|log.Ltime)

	warnLogger = log.New(os.Stdout,
		"[WARN]: ",
		log.Ldate|log.Ltime)

	errorLogger = log.New(os.Stdout,
		"[ERROR]: ",
		log.Ldate|log.Ltime)

	fatalLogger = log.New(os.Stdout,
		"[FATAL]: ",
		log.Ldate|log.Ltime)
}

// colorOutput() sets the prefix of each log type to matching colour
// TODO: add windows support

func colourOutput() {
	if runtime.GOOS == "win32" {
	} else {
		debugLogger.SetPrefix("\033[34m[DEBUG]\033[0m: ")
		infoLogger.SetPrefix("\033[32m[INFO]\033[0m: ")
		warnLogger.SetPrefix("\033[33m[WARN]\033[0m: ")
		errorLogger.SetPrefix("\033[31m[ERROR]\033[0m: ")
		fatalLogger.SetPrefix("\033[31m[FATAL]\033[0m: ")
	}
}

// clearAllLoggers() sets all logger flags to 0 and outputs to Discard

func clearAllLoggers() {
	debugLogger.SetFlags(0)
	infoLogger.SetFlags(0)
	warnLogger.SetFlags(0)
	errorLogger.SetFlags(0)
	fatalLogger.SetFlags(0)

	debugLogger.SetOutput(ioutil.Discard)
	infoLogger.SetOutput(ioutil.Discard)
	warnLogger.SetOutput(ioutil.Discard)
	errorLogger.SetOutput(ioutil.Discard)
	fatalLogger.SetOutput(ioutil.Discard)
}

// setupOutputs() sets up the io.writer to use for logging
// TODO: Fix up rotating at the moment its a quick job

func setupOutputs() {
	if len(Logger.File) > 0 {
		logPath := path.Join(common.GetDefaultDataDir(runtime.GOOS), "logs")
		err := common.CheckDir(logPath, true)
		if err != nil {
			Error(err)
		}
		logFile := path.Join(logPath, Logger.File)
		if Logger.Rotate {
			currentTime := time.Now()
			newName := currentTime.Format("2006-01-02 15-04-05")
			newFile := newName + " " + Logger.File
			err := os.Rename(logFile, path.Join(logPath, newFile))
			if err != nil {
				Errorf("Failed to rename old log file %s", err)
			}
			logFileHandle, logfileErr = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		} else {
			logFileHandle, logfileErr = os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		}
		if logfileErr != nil {
			Errorf("Failed to open log file: %s", logfileErr)
		}
		logOutput = io.MultiWriter(os.Stdout, logFileHandle)
	} else {
		logOutput = os.Stdout
	}
}

// CloseLogFile close the handler for any open log files
func CloseLogFile() (err error) {
	if logFileHandle != nil {
		err = logFileHandle.Close()
	}
	return nil
}
