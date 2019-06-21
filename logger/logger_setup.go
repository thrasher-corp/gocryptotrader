package logger

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func addSubLogger(subsystem, flags string, output io.Writer) {
	temp := subLogger{
		output: output,
	}
	enabledLevels := strings.Split(flags, "|")
	for x := range enabledLevels {
		switch level := enabledLevels[x]; level {
		case "DEBUG":
			temp.Debug = true
		case "INFO":
			temp.Info = true
		case "WARN":
			temp.Warn = true
		case "ERROR":
			temp.Error = true
		}
	}
	subsystemLoggers[subsystem] = temp
}

func SetupSubLogger(s []subLoggers) {
	for x := range s {
		output := getWriters(&s[x])
		addSubLogger(s[x].Name, s[x].Level, output)
	}
}

func getWriters(s *subLoggers) io.Writer {
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

func GenDefaultSettings() (log LoggerConfig) {
	t := func(t bool) *bool { return &t }(true)
	log = LoggerConfig{
		Enabled: t,
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
		SubLoggers: []subLoggers{
			{
				Name:   "log",
				Level:  "INFO|DEBUG|WARN|ERROR",
				Output: "stdout",
			}},
	}
	return
}
