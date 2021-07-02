package log

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func TestMain(m *testing.M) {
	setupTestLoggers()
	os.Exit(m.Run())
}

func setupTestLoggers() {
	logTest := Config{
		Enabled: convert.BoolPtr(true),
		SubLoggerConfig: SubLoggerConfig{
			Output: "console",
			Level:  "INFO|WARN|DEBUG|ERROR",
		},
		AdvancedSettings: advancedSettings{
			ShowLogSystemName: convert.BoolPtr(true),
			Spacer:            " | ",
			TimeStampFormat:   timestampFormat,
			Headers: headers{
				Info:  "[INFO]",
				Warn:  "[WARN]",
				Debug: "[DEBUG]",
				Error: "[ERROR]",
			},
		},
		SubLoggers: []SubLoggerConfig{
			{
				Name:   "TEST",
				Level:  "INFO|DEBUG|WARN|ERROR",
				Output: "stdout",
			}},
	}
	RWM.Lock()
	GlobalLogConfig = &logTest
	RWM.Unlock()
	SetupGlobalLogger()
	SetupSubLoggers(logTest.SubLoggers)
}

func SetupDisabled() {
	logTest := Config{
		Enabled: convert.BoolPtr(false),
	}
	RWM.Lock()
	GlobalLogConfig = &logTest
	RWM.Unlock()

	SetupGlobalLogger()
	SetupSubLoggers(logTest.SubLoggers)
}

func BenchmarkInfo(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

func SetupTestDisabled(t *testing.T) {
	SetupDisabled()
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

func TestRemoveWriter(t *testing.T) {
	mw := MultiWriter()
	m := mw.(*multiWriter)

	m.Add(ioutil.Discard)
	m.Add(os.Stdin)
	m.Add(os.Stdout)

	total := len(m.writers)

	m.Remove(os.Stdin)
	m.Remove(os.Stdout)

	if len(m.writers) != total-2 {
		t.Errorf("expected m.Writers to be %v got %v", total-2, len(m.writers))
	}
}

func TestLevel(t *testing.T) {
	_, err := Level("LOG")
	if err != nil {
		t.Errorf("Failed to get log %s levels skipping", err)
	}

	_, err = Level("totallyinvalidlogger")
	if err == nil {
		t.Error("Expected error on invalid logger")
	}
}

func TestSetLevel(t *testing.T) {
	newLevel, err := SetLevel("LOG", "ERROR")
	if err != nil {
		t.Skipf("Failed to get log %s levels skipping", err)
	}

	if newLevel.Info || newLevel.Debug || newLevel.Warn {
		t.Error("failed to set level correctly")
	}

	if !newLevel.Error {
		t.Error("failed to set level correctly")
	}

	_, err = SetLevel("abc12345556665", "ERROR")
	if err == nil {
		t.Error("SetLevel() Should return error on invalid logger")
	}
}

func TestValidSubLogger(t *testing.T) {
	b, logPtr := validSubLogger("LOG")

	if !b {
		t.Skip("validSubLogger() should return found, pointer if valid logger found")
	}
	if logPtr == nil {
		t.Error("validSubLogger() should return a pointer and not nil")
	}
}

func TestCloseLogger(t *testing.T) {
	err := CloseLogger()
	if err != nil {
		t.Errorf("CloseLogger() failed %v", err)
	}
}

func TestConfigureSubLogger(t *testing.T) {
	err := configureSubLogger("LOG", "INFO", os.Stdin)
	if err != nil {
		t.Skipf("configureSubLogger() returned unexpected error %v", err)
	}
	if (Global.Levels != Levels{
		Info:  true,
		Debug: false,
	}) {
		t.Error("configureSubLogger() incorrectly configure subLogger")
	}
	if Global.name != "LOG" {
		t.Error("configureSubLogger() Failed to uppercase name")
	}
}

func TestSplitLevel(t *testing.T) {
	levelsInfoDebug := splitLevel("INFO|DEBUG")

	expected := Levels{
		Info:  true,
		Debug: true,
		Warn:  false,
		Error: false,
	}

	if levelsInfoDebug != expected {
		t.Errorf("splitLevel() returned invalid data expected: %+v got: %+v", expected, levelsInfoDebug)
	}
}

func BenchmarkInfoDisabled(b *testing.B) {
	SetupDisabled()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

func BenchmarkInfof(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infof(Global, "Hello this is an infof benchmark %v %v %v\n", n, 1, 2)
	}
}

func BenchmarkInfoln(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infoln(Global, "Hello this is an infoln benchmark")
	}
}

func TestNewLogEvent(t *testing.T) {
	w := &bytes.Buffer{}
	logger.newLogEvent("out", "header", "SUBLOGGER", w)

	if w.String() == "" {
		t.Error("newLogEvent() failed expected output got empty string")
	}

	err := logger.newLogEvent("out", "header", "SUBLOGGER", nil)
	if err == nil {
		t.Error("Error expected with output is set to nil")
	}
}

func TestInfo(t *testing.T) {
	w := &bytes.Buffer{}

	tempSL := SubLogger{
		"TESTYMCTESTALOT",
		splitLevel("INFO|WARN|DEBUG|ERROR"),
		w,
	}

	Info(&tempSL, "Hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}

	tempSL.output = nil
	w.Reset()

	SetLevel("TESTYMCTESTALOT", "INFO")
	Debug(&tempSL, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but Debug wrote to output")
	}
}

func TestSubLoggerName(t *testing.T) {
	w := &bytes.Buffer{}
	registerNewSubLogger("sublogger")
	logger.newLogEvent("out", "header", "SUBLOGGER", w)
	if !strings.Contains(w.String(), "SUBLOGGER") {
		t.Error("Expected SUBLOGGER in output")
	}

	logger.ShowLogSystemName = false
	w.Reset()
	logger.newLogEvent("out", "header", "SUBLOGGER", w)
	if strings.Contains(w.String(), "SUBLOGGER") {
		t.Error("Unexpected SUBLOGGER in output")
	}
}

func TestNewSubLogger(t *testing.T) {
	_, err := NewSubLogger("")
	if !errors.Is(err, errEmptyLoggerName) {
		t.Fatalf("received: %v but expected: %v", err, errEmptyLoggerName)
	}

	sl, err := NewSubLogger("TESTERINOS")
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	Debug(sl, "testerinos")

	_, err = NewSubLogger("TESTERINOS")
	if !errors.Is(err, errSubLoggerAlreadyregistered) {
		t.Fatalf("received: %v but expected: %v", err, errSubLoggerAlreadyregistered)
	}
}
