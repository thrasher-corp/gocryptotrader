package loggerv2

import (
	"testing"
)

func BenchmarkInfo(b *testing.B) {
	addSubLogger("testlog", "DEBUG|WARN|ERROR")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info("testlog", "Hello this is an info benchmark")
	}
}

func BenchmarkInfof(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infof("sys", "Hello this is an info benchmark %v", n)
	}
}
