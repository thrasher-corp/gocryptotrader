package logger

import (
	"fmt"
	"log"
	"os"
)

// Info handler takes any input returns unformatted output to infoLogger writer
func Info(v ...interface{}) {
	infoLogger.Print(v...)
}

// Infof handler takes any input infoLogger returns formatted output to infoLogger writer
func Infof(data string, v ...interface{}) {
	infoLogger.Printf(data, v...)
}

// Infoln handler takes any input infoLogger returns formatted output to infoLogger writer
func Infoln(v ...interface{}) {
	infoLogger.Println(v...)
}

// Print aliased to Standard log.Print
var Print = log.Print

// Printf alaised to tandard log.Printf
var Printf = log.Printf

// Println  alaised to tandard log.Println
var Println = log.Println

// Debug handler takes any input returns unformatted output to infoLogger writer
func Debug(v ...interface{}) {
	debugLogger.Print(v...)
}

// Debugf handler takes any input infoLogger returns formatted output to infoLogger writer
func Debugf(data string, v ...interface{}) {
	debugLogger.Printf(data, v...)
}

// Debugln handler takes any input infoLogger returns formatted output to infoLogger writer
func Debugln(v ...interface{}) {
	debugLogger.Println(v...)
}

// Warn handler takes any input returns unformatted output to warnLogger writer
func Warn(v ...interface{}) {
	warnLogger.Print(v...)
}

// Warnf handler takes any input returns unformatted output to warnLogger writer
func Warnf(data string, v ...interface{}) {
	warnLogger.Printf(data, v...)
}

// Error handler takes any input returns unformatted output to errorLogger writer
func Error(v ...interface{}) {
	errorLogger.Print(v...)
}

// Errorf handler takes any input returns unformatted output to errorLogger writer
func Errorf(data string, v ...interface{}) {
	errorLogger.Printf(data, v...)
}

// Fatal  handler takes any input returns unformatted output to fatalLogger writer
func Fatal(v ...interface{}) {
	// Send to Output instead of Fatal to allow us to increase the output depth by 1 to make sure the correct file is displayed
	fatalLogger.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

// Fatalf  handler takes any input returns unformatted output to fatalLogger writer
func Fatalf(data string, v ...interface{}) {
	// Send to Output instead of Fatal to allow us to increase the output depth by 1 to make sure the correct file is displayed
	fatalLogger.Output(2, fmt.Sprintf(data, v...))
	os.Exit(1)
}
