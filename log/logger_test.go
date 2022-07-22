package log

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func TestMain(m *testing.M) {
	err := setupTestLoggers()
	if err != nil {
		log.Fatal("cannot set up test loggers", err)
	}
	tempDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	log.Println("temp dir created at:", tempDir)
	LogPath = tempDir
	r := m.Run()
	err = CloseLogger()
	if err != nil {
		log.Fatalf("CloseLogger() failed %v", err)
	}
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
	if err := SetupGlobalLogger(runtime.GOMAXPROCS(-1)); err != nil {
		return err
	}
	return SetupSubLoggers(logTest.SubLoggers)
}

func SetupDisabled() error {
	logTest := Config{
		Enabled:         convert.BoolPtr(false),
		SubLoggerConfig: SubLoggerConfig{Output: "console"},
	}
	RWM.Lock()
	GlobalLogConfig = &logTest
	RWM.Unlock()

	if err := SetupGlobalLogger(runtime.GOMAXPROCS(-1)); err != nil {
		return err
	}
	return SetupSubLoggers(logTest.SubLoggers)
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

	fields := &logFields{}
	buff := newTestBuffer()

	var err error
	fields.output, err = multiWriter(io.Discard, buff)
	if err != nil {
		t.Fatal(err)
	}

	payload := "woooooooooooooooooooooooooooooooooooow"
	fields.output.StageLogEvent(func() string { return payload }, "", "", "", "", false, false)
	if err != nil {
		t.Fatal(err)
	}

	<-buff.Finished
	if contents := buff.Read(); !strings.Contains(contents, payload) {
		t.Errorf("received: '%v' but expected: '%v'", contents, payload)
	}

	fields.output, err = multiWriter(&WriteShorter{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	fields.output.StageLogEvent(func() string { return payload }, "", "", "", "", false, false) // Will display error: Logger write error: *log.WriteShorter short write

	fields.output, err = multiWriter(&WriteError{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	fields.output.StageLogEvent(func() string { return payload }, "", "", "", "", false, false) // Will display error: Logger write error: *log.WriteError write error
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

func TestConfigureSubLogger(t *testing.T) {
	t.Parallel()
	mw := &multiWriterHolder{writers: []io.Writer{newTestBuffer()}}
	err := configureSubLogger("LOG", "INFO", mw)
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

func TestStageNewLogEvent(t *testing.T) {
	t.Parallel()
	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	fields := &logFields{output: mw}
	fields.output.StageLogEvent(func() string { return "out" }, "header", "SUBLOGGER", " space ", "", false, false)

	<-w.Finished
	if contents := w.Read(); contents != "header space  space out\n" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "header space  space out\n")
	}
}

func TestInfo(t *testing.T) {
	t.Parallel()
	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	sl := registerNewSubLogger("TESTYMCTESTALOTINFO")
	sl.SetLevels(splitLevel("INFO"))
	sl.SetOutput(mw)

	Info(sl, "Hello")
	<-w.Finished
	contents := w.Read()

	if !strings.Contains(contents, "Hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "Hello")
	}

	Infof(sl, "%s", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Infoln(sl, "hello", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello hello")
	}

	_, err := SetLevel("TESTYMCTESTALOTINFO", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it should return if functionality
	// is not enabled.
	Info(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}

	Infoln(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}
}

func TestDebug(t *testing.T) {
	t.Parallel()
	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	sl := registerNewSubLogger("TESTYMCTESTALOTDEBUG")
	sl.SetLevels(splitLevel("DEBUG"))
	sl.SetOutput(mw)

	Debug(sl, "Hello")
	<-w.Finished
	contents := w.Read()

	if !strings.Contains(contents, "Hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "Hello")
	}

	Debugf(sl, "%s", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Debugln(sl, "hello", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello hello")
	}

	_, err := SetLevel("TESTYMCTESTALOTDEBUG", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it should return if functionality
	// is not enabled.
	Debug(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}

	Debugln(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}
}

func TestWarn(t *testing.T) {
	t.Parallel()
	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	sl := registerNewSubLogger("TESTYMCTESTALOTWARN")
	sl.SetLevels(splitLevel("WARN"))
	sl.SetOutput(mw)

	Warn(sl, "Hello")
	<-w.Finished
	contents := w.Read()

	if !strings.Contains(contents, "Hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "Hello")
	}

	Warnf(sl, "%s", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Warnln(sl, "hello", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello hello")
	}

	_, err := SetLevel("TESTYMCTESTALOTWARN", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it shhould return if functionality
	// is not enabled.
	Warn(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}

	Warnln(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}
}

func TestError(t *testing.T) {
	t.Parallel()
	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	sl := registerNewSubLogger("TESTYMCTESTALOTERROR")
	sl.SetLevels(splitLevel("ERROR"))
	sl.SetOutput(mw)

	Error(sl, "Hello")
	<-w.Finished
	contents := w.Read()

	if !strings.Contains(contents, "Hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "Hello")
	}

	Errorf(sl, "%s", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Errorln(sl, "hello", "hello")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hello hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello hello")
	}

	_, err := SetLevel("TESTYMCTESTALOTERROR", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it shhould return if functionality
	// is not enabled.
	Error(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}

	Errorln(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}
}

func TestSubLoggerName(t *testing.T) {
	t.Parallel()
	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	mw.StageLogEvent(func() string { return "out" }, "header", "SUBLOGGER", "||", time.RFC3339, true, false)
	<-w.Finished
	contents := w.Read()
	if !strings.Contains(contents, "SUBLOGGER") {
		t.Error("Expected SUBLOGGER in output")
	}

	mw.StageLogEvent(func() string { return "out" }, "header", "SUBLOGGER", "||", time.RFC3339, false, false)
	<-w.Finished
	contents = w.Read()
	if strings.Contains(contents, "SUBLOGGER") {
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

type testBuffer struct {
	value    string
	Finished chan struct{}
}

func (tb *testBuffer) Write(p []byte) (int, error) {
	tb.value = string(p)
	tb.Finished <- struct{}{}
	return len(p), nil
}

func (tb *testBuffer) Read() string {
	defer func() { tb.value = "" }()
	return tb.value
}

func newTestBuffer() *testBuffer {
	return &testBuffer{Finished: make(chan struct{}, 1)}
}

//  2140294	       770.0 ns/op	       0 B/op	       0 allocs/op
func BenchmarkNewLogEvent(b *testing.B) {
	mw := &multiWriterHolder{writers: []io.Writer{io.Discard}}
	for i := 0; i < b.N; i++ {
		mw.StageLogEvent(func() string { return "somedata" }, "header", "sublog", "||", time.RFC3339, true, false)
	}
}

// BenchmarkInfo-8   	 1000000	     64971 ns/op	      47 B/op	       1 allocs/op
func BenchmarkInfo(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

// BenchmarkInfoDisabled-8 47124242	        24.16 ns/op	       0 B/op	       0 allocs/op
func BenchmarkInfoDisabled(b *testing.B) {
	if err := SetupDisabled(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Info(Global, "Hello this is an info benchmark")
	}
}

// BenchmarkInfof-8   	 1000000	     72641 ns/op	     178 B/op	       4 allocs/op
func BenchmarkInfof(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infof(Global, "Hello this is an infof benchmark %v %v %v\n", n, 1, 2)
	}
}

// BenchmarkInfoln-8   	 1000000	     68152 ns/op	     121 B/op	       3 allocs/op
func BenchmarkInfoln(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Infoln(Global, "Hello this is an infoln benchmark")
	}
}
