package log

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func TestMain(m *testing.M) {
	err := setupTestLoggers()
	if err != nil {
		log.Fatal("cannot set up test loggers", err)
	}
	tempDir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	log.Println("temp dir created at:", tempDir)
	LogPath = tempDir
	r := m.Run()
	err = os.Remove(tempDir)
	if err != nil {
		log.Println("failed to remove temp file:", tempDir)
	}
	os.Exit(r)
}

func setupTestLoggers() error {
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
				Name:   "lOg",
				Level:  "INFO|DEBUG|WARN|ERROR",
				Output: "stdout",
			}},
	}
	RWM.Lock()
	GlobalLogConfig = &logTest
	RWM.Unlock()
	if err := SetupGlobalLogger(); err != nil {
		return err
	}
	return SetupSubLoggers(logTest.SubLoggers)
}

func SetupDisabled() error {
	logTest := Config{
		Enabled: convert.BoolPtr(false),
	}
	RWM.Lock()
	GlobalLogConfig = &logTest
	RWM.Unlock()

	if err := SetupGlobalLogger(); err != nil {
		return err
	}
	return SetupSubLoggers(logTest.SubLoggers)
}

func BenchmarkInfo(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

func TestAddWriter(t *testing.T) {
	t.Parallel()
	_, err := multiWriter(io.Discard, io.Discard)
	if !errors.Is(err, errWriterAlreadyLoaded) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWriterAlreadyLoaded)
	}

	mw, err := multiWriter()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	err = mw.Add(io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.Add(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.Add(os.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	if total := len(mw.writers); total != 3 {
		t.Errorf("expected m.Writers to be 3 %v", total)
	}
}

func TestRemoveWriter(t *testing.T) {
	t.Parallel()
	mw, err := multiWriter()
	if err != nil {
		t.Fatal(err)
	}
	err = mw.Add(io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.Add(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.Add(os.Stdout)
	if err != nil {
		t.Fatal(err)
	}
	total := len(mw.writers)
	err = mw.Remove(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.Remove(os.Stdout)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.Remove(&bytes.Buffer{})
	if !errors.Is(err, errWriterNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWriterNotFound)
	}

	if len(mw.writers) != total-2 {
		t.Errorf("expected m.Writers to be %v got %v", total-2, len(mw.writers))
	}
}

type WriteShorter struct{}

func (w *WriteShorter) Write(p []byte) (int, error) {
	return 1, nil
}

type WriteError struct{}

func (w *WriteError) Write(p []byte) (int, error) {
	return 0, errWriteError
}

var errWriteError = errors.New("write error")

func TestMultiWriterWrite(t *testing.T) {
	t.Parallel()
	mw, err := multiWriter(io.Discard, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}

	payload := "woooooooooooooooooooooooooooooooooooow"
	l, err := mw.Write([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if l != len(payload) {
		t.Fatal("unexpected return")
	}

	mw, err = multiWriter(&WriteShorter{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mw.Write([]byte(payload))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("received: '%v' but expected: '%v'", err, io.ErrShortWrite)
	}

	mw, err = multiWriter(&WriteError{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	_, err = mw.Write([]byte(payload))
	if !errors.Is(err, errWriteError) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errWriteError)
	}
}

func TestGetWriters(t *testing.T) {
	t.Parallel()
	_, err := getWriters(nil)
	if !errors.Is(err, errSubloggerConfigIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errSubloggerConfigIsNil)
	}

	outputWriters := "stDout|stderr|filE"

	_, err = getWriters(&SubLoggerConfig{Output: outputWriters})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	outputWriters = "stdout|stderr|file|noobs"
	_, err = getWriters(&SubLoggerConfig{Output: outputWriters})
	if !errors.Is(err, errUnhandledOutputWriter) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errUnhandledOutputWriter)
	}
}

func TestGenDefaultSettings(t *testing.T) {
	t.Parallel()
	if cfg := GenDefaultSettings(); cfg.Enabled == nil {
		t.Fatal("unexpected items in struct")
	}
}

func TestLevel(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestCloseLogger(t *testing.T) {
	t.Parallel()
	if err := CloseLogger(); err != nil {
		t.Errorf("CloseLogger() failed %v", err)
	}
}

func TestConfigureSubLogger(t *testing.T) {
	t.Parallel()
	err := configureSubLogger("LOG", "INFO", os.Stdin)
	if err != nil {
		t.Skipf("configureSubLogger() returned unexpected error %v", err)
	}
	levels := Global.GetLevels()
	if (levels != Levels{Info: true, Debug: false}) {
		t.Error("configureSubLogger() incorrectly configure subLogger")
	}
	if Global.name != "LOG" {
		t.Error("configureSubLogger() Failed to uppercase name")
	}
}

func TestSplitLevel(t *testing.T) {
	t.Parallel()
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
	if err := SetupDisabled(); err != nil {
		b.Fatal(err)
	}

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
	t.Parallel()
	w := &bytes.Buffer{}
	RWM.Lock()
	err := logger.newLogEvent("out", "header", "SUBLOGGER", w)
	RWM.Unlock()
	if err != nil {
		t.Fatal(err)
	}

	if w.String() == "" {
		t.Error("newLogEvent() failed expected output got empty string")
	}

	RWM.Lock()
	err = logger.newLogEvent("out", "header", "SUBLOGGER", nil)
	RWM.Unlock()
	if err == nil {
		t.Error("Error expected with output is set to nil")
	}
}

func TestInfo(t *testing.T) {
	t.Parallel()
	w := &bytes.Buffer{}

	sl := registerNewSubLogger("TESTYMCTESTALOTINFO")
	sl.SetLevels(splitLevel("INFO|WARN|DEBUG|ERROR"))
	sl.SetOutput(w)

	Info(sl, "Hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Infof(sl, "%s", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Infoln(sl, "hello", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	_, err := SetLevel("TESTYMCTESTALOTINFO", "")
	if err != nil {
		t.Fatal(err)
	}

	Info(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}

	Infoln(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}
}

func TestDebug(t *testing.T) {
	t.Parallel()
	w := &bytes.Buffer{}

	sl := registerNewSubLogger("TESTYMCTESTALOTDEBUG")
	sl.SetLevels(splitLevel("INFO|WARN|DEBUG|ERROR"))
	sl.SetOutput(w)

	Debug(sl, "Hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Debugf(sl, "%s", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Debugln(sl, "hello", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	_, err := SetLevel("TESTYMCTESTALOTDEBUG", "")
	if err != nil {
		t.Fatal(err)
	}

	Debug(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}

	Debugln(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}
}

func TestWarn(t *testing.T) {
	t.Parallel()
	w := &bytes.Buffer{}

	sl := registerNewSubLogger("TESTYMCTESTALOTWARN")
	sl.SetLevels(splitLevel("INFO|WARN|DEBUG|ERROR"))
	sl.SetOutput(w)

	Warn(sl, "Hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Warnf(sl, "%s", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Warnln(sl, "hello", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	_, err := SetLevel("TESTYMCTESTALOTWARN", "")
	if err != nil {
		t.Fatal(err)
	}

	Warn(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}

	Warnln(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}
}

func TestError(t *testing.T) {
	t.Parallel()
	w := &bytes.Buffer{}

	sl := registerNewSubLogger("TESTYMCTESTALOTERROR")
	sl.SetLevels(splitLevel("INFO|WARN|DEBUG|ERROR"))
	sl.SetOutput(w)

	Error(sl, "Hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Errorf(sl, "%s", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	Errorln(sl, "hello", "hello")

	if w.String() == "" {
		t.Error("expected Info() to write output to buffer")
	}
	w.Reset()

	_, err := SetLevel("TESTYMCTESTALOTERROR", "")
	if err != nil {
		t.Fatal(err)
	}

	Error(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}

	Errorln(sl, "HelloHello")

	if w.String() != "" {
		t.Error("Expected output buffer to be empty but wrote to output", w.String())
	}
}

func TestSubLoggerName(t *testing.T) {
	t.Parallel()
	w := &bytes.Buffer{}
	registerNewSubLogger("sublogger")
	RWM.Lock()
	err := logger.newLogEvent("out", "header", "SUBLOGGER", w)
	RWM.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.String(), "SUBLOGGER") {
		t.Error("Expected SUBLOGGER in output")
	}

	RWM.Lock()
	logger.ShowLogSystemName = false
	RWM.Unlock()
	w.Reset()
	RWM.Lock()
	err = logger.newLogEvent("out", "header", "SUBLOGGER", w)
	RWM.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(w.String(), "SUBLOGGER") {
		t.Error("Unexpected SUBLOGGER in output")
	}
}

func TestNewSubLogger(t *testing.T) {
	t.Parallel()
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
	if !errors.Is(err, ErrSubLoggerAlreadyRegistered) {
		t.Fatalf("received: %v but expected: %v", err, ErrSubLoggerAlreadyRegistered)
	}
}

func BenchmarkNewLogEvent(b *testing.B) {
	var bro bytes.Buffer
	l := Logger{Spacer: " "}
	for i := 0; i < b.N; i++ {
		_ = l.newLogEvent("somedata", "header", "sublog", &bro)
	}
}

func TestRotateWrite(t *testing.T) {
	t.Parallel()
	empty := Rotate{Rotate: convert.BoolPtr(true), FileName: "test.txt"}
	payload := make([]byte, defaultMaxSize*megabyte+1)
	_, err := empty.Write(payload)
	if !errors.Is(err, errExceedsMaxFileSize) {
		t.Fatalf("received: %v but expected: %v", err, errExceedsMaxFileSize)
	}

	empty.MaxSize = 1
	payload = make([]byte, 1*megabyte+1)
	_, err = empty.Write(payload)
	if !errors.Is(err, errExceedsMaxFileSize) {
		t.Fatalf("received: %v but expected: %v", err, errExceedsMaxFileSize)
	}

	// test write
	payload = make([]byte, 1*megabyte-1)
	_, err = empty.Write(payload)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	// test rotate
	payload = make([]byte, 1*megabyte)
	_, err = empty.Write(payload)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}

func TestOpenNew(t *testing.T) {
	t.Parallel()
	empty := Rotate{}
	err := empty.openNew()
	if !errors.Is(err, errFileNameIsEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errFileNameIsEmpty)
	}

	empty.FileName = "wow.txt"
	err = empty.openNew()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
}
