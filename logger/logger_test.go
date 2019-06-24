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
		SubLoggers: []SubLoggers{
			{
				Name:   "log",
				Level:  "INFO|DEBUG|WARN|ERROR",
				Output: "stdout",
			}},
	}
	logger = newLogger(&logTest)
	SetupSubLogger(logTest.SubLoggers)
}

func TestRemoveWriter(t *testing.T) {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	m.Add(ioutil.Discard)
	m.Add(os.Stdin)
	m.Add(os.Stdout)

	total := len(m.writers)

	if total != 3 {
		t.Errorf("expected m.Writers to be 1 %v", total)
	}

	t.Log(m.writers)

	m.Remove(ioutil.Discard)

	t.Log(m.writers)
	t.Log(len(m.writers))
}

func BenchmarkInfo(b *testing.B) {
	SetupTest()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info("log", "Hello this is an info benchmark")
	}
}

func BenchmarkInfoDisabled(b *testing.B) {
	logTest := Config{
		SubLoggers: []SubLoggers{
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
	SetupTest()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infof("log", "Hello this is an infof benchmark %v %v %v\n", n, 1, 2)
	}
}

func BenchmarkInfoln(b *testing.B) {
	SetupTest()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infoln("log", "Hello this is an infoln benchmark")
	}
}
