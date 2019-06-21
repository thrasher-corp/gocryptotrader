package logger

import (
	"os"
	"testing"
)

func BenchmarkInfo(b *testing.B) {
	t := func(t bool) *bool { return &t }(true)
	logTest := Config{
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

	logger = newLogger(&logTest)
	SetupSubLogger(logTest.SubLoggers)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info("log", "Hello this is an info benchmark")
	}
}

func BenchmarkInfoDisabled(b *testing.B) {

	logTest := Config{
		SubLoggers: []subLoggers{
			{
				Name:   "log",
				Level:  "DEBUG|WARN|ERROR",
				Output: "stdout",
			}},
	}
	logger = newLogger(&logTest)
	SetupSubLogger(logTest.SubLoggers)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info("log", "Hello this is an info benchmark")
	}
}

func BenchmarkInfof(b *testing.B) {
	logger = newLogger(GlobalLogConfig)
	addSubLogger("sys", "DEBUG|WARN|ERROR", os.Stdout)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infof("sys", "Hello this is an infof benchmark %v %v %v\n", n, 1, 2)
	}
}

func BenchmarkInfoln(b *testing.B) {
	logger = newLogger(GlobalLogConfig)
	addSubLogger("sys", "INFO|DEBUG|WARN|ERROR", os.Stdout)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infoln("sys", "Hello this is an infoln benchmark")
	}
}
