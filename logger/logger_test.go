package logger

import (
	"io/ioutil"
	"os"
	"testing"
)

var (
	trueptr  = func(b bool) *bool { return &b }(true)
	falseptr = func(b bool) *bool { return &b }(false)
)

func SetupTest() {
	logTest := Config{
		Enabled: trueptr,
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
		SubLoggers: []SubLoggerConfig{
			{
				Name:   "log",
				Level:  "INFO|DEBUG|WARN|ERROR",
				Output: "stdout",
			}},
	}
	logger = newLogger(&logTest)
	SetupSubLoggers(logTest.SubLoggers)
}

func SetupTestDisabled() {
	logTest := Config{
		Enabled: falseptr,
	}
	logger = newLogger(&logTest)
	SetupSubLoggers(logTest.SubLoggers)
}

func TestAddWriter(t *testing.T) {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	m.Add(ioutil.Discard)
	m.Add(os.Stdin)
	m.Add(os.Stdout)

	total := len(m.writers)

	if total != 3 {
		t.Errorf("expected m.Writers to be 3 %v", total)
	}
}

func TestLoggerDisabled(t *testing.T) {

}

func BenchmarkInfo(b *testing.B) {
	SetupTest()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

func BenchmarkInfoDisabled(b *testing.B) {
	logTest := Config{
		SubLoggers: []SubLoggerConfig{
			{
				Name:   "log",
				Level:  "DEBUG|WARN|ERROR",
				Output: "stdout",
			}},
	}
	logger = newLogger(&logTest)
	SetupSubLoggers(logTest.SubLoggers)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

func BenchmarkInfof(b *testing.B) {
	SetupTest()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infof(Global, "Hello this is an infof benchmark %v %v %v\n", n, 1, 2)
	}
}

func BenchmarkInfoln(b *testing.B) {
	SetupTest()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infoln(Global, "Hello this is an infoln benchmark")
	}
}
