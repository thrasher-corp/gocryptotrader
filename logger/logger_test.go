package logger

import (
	"os"
	"testing"
)

func BenchmarkInfo(b *testing.B) {
	logger = newLogger(GlobalLogConfig)
	addSubLogger("testlog", "INFO|DEBUG|WARN|ERROR", os.Stdout)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info("testlog", "Hello this is an info benchmark")
	}
}

func BenchmarkAllDisabled(b *testing.B) {
	logger = newLogger(GlobalLogConfig)
	addSubLogger("testlog", "", os.Stdout)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info("testlog", "Hello this is an info benchmark")
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
