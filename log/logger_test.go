package log

import (
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var (
	testConfigEnabled = &Config{
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
			},
		},
	}
	testConfigDisabled = &Config{
		Enabled:         convert.BoolPtr(false),
		SubLoggerConfig: SubLoggerConfig{Output: "console"},
	}

	tempDir string
)

func TestMain(m *testing.M) {
	err := setupTestLoggers()
	if err != nil {
		log.Fatal("cannot set up test loggers", err)
	}
	tempDir, err = os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	log.Println("temp dir created at:", tempDir)
	err = SetLogPath(tempDir)
	if err != nil {
		log.Fatal("Cannot set log path", err)
	}
	r := m.Run()
	err = CloseLogger()
	if err != nil {
		log.Fatalf("CloseLogger() failed %v", err)
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		log.Fatal("failed to remove temp file:", tempDir, err)
	}
	os.Exit(r)
}

func setupTestLoggers() error {
	err := SetGlobalLogConfig(testConfigEnabled)
	if err != nil {
		return err
	}
	err = SetupGlobalLogger("test", false)
	if err != nil {
		return err
	}
	return SetupSubLoggers(testConfigEnabled.SubLoggers)
}

func SetupDisabled() error {
	err := SetGlobalLogConfig(testConfigDisabled)
	if err != nil {
		return err
	}
	err = SetupGlobalLogger("test", false)
	if err != nil {
		return err
	}
	return SetupSubLoggers(testConfigDisabled.SubLoggers)
}

func TestSetGlobalLogConfig(t *testing.T) {
	t.Parallel()
	err := SetGlobalLogConfig(nil)
	require.ErrorIs(t, err, errConfigNil)

	err = SetGlobalLogConfig(testConfigEnabled)
	require.NoError(t, err)
}

func TestSetLogPath(t *testing.T) {
	t.Parallel()
	err := SetLogPath("")
	require.ErrorIs(t, err, errLogPathIsEmpty)

	err = SetLogPath(tempDir)
	require.NoError(t, err)

	if path := GetLogPath(); path != tempDir {
		t.Fatalf("received: '%v' but expected: '%v'", path, tempDir)
	}
}

func TestSetFileLoggingState(t *testing.T) {
	t.Parallel()

	SetFileLoggingState(true)
	if !getFileLoggingState() {
		t.Fatal("unexpected value")
	}

	SetFileLoggingState(false)
	if getFileLoggingState() {
		t.Fatal("unexpected value")
	}
}

func getFileLoggingState() bool {
	mu.RLock()
	defer mu.RUnlock()
	return fileLoggingConfiguredCorrectly
}

func TestAddWriter(t *testing.T) {
	t.Parallel()
	_, err := multiWriter(io.Discard, io.Discard)
	require.ErrorIs(t, err, errWriterAlreadyLoaded)

	mw, err := multiWriter()
	require.NoError(t, err)

	err = mw.add(io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.add(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	err = mw.add(os.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	err = mw.add(nil)
	require.ErrorIs(t, err, errWriterIsNil)

	if total := len(mw.writers); total != 3 {
		t.Errorf("expected m.Writers to be 3 %v", total)
	}
}

type WriteShorter struct{}

func (w *WriteShorter) Write(_ []byte) (int, error) {
	return 1, nil
}

type WriteError struct{}

func (w *WriteError) Write(_ []byte) (int, error) {
	return 0, errWriteError
}

var errWriteError = errors.New("write error")

func TestMultiWriterWrite(t *testing.T) {
	t.Parallel()

	f := &fields{}
	buff := newTestBuffer()

	var err error
	f.output, err = multiWriter(io.Discard, buff)
	if err != nil {
		t.Fatal(err)
	}

	payload := "woooooooooooooooooooooooooooooooooooow"
	f.output.StageLogEvent(func() string { return payload }, "", "", "", "", "", "", false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	<-buff.Finished
	if contents := buff.Read(); !strings.Contains(contents, payload) {
		t.Errorf("received: '%v' but expected: '%v'", contents, payload)
	}

	f.output, err = multiWriter(&WriteShorter{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	f.output.StageLogEvent(func() string { return payload }, "", "", "", "", "", "", false, false, false, nil) // Will display error: Logger write error: *log.WriteShorter short write

	f.output, err = multiWriter(&WriteError{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	f.output.StageLogEvent(func() string { return payload }, "", "", "", "", "", "", false, false, false, nil) // Will display error: Logger write error: *log.WriteError write error
}

func TestGetWriters(t *testing.T) {
	t.Parallel()
	err := getWritersProtected(nil)
	require.ErrorIs(t, err, errSubloggerConfigIsNil)

	outputWriters := "stDout|stderr|filE"

	mu.Lock()
	fileLoggingConfiguredCorrectly = false
	_, err = getWriters(&SubLoggerConfig{Output: outputWriters})
	require.ErrorIs(t, err, errFileLoggingNotConfiguredCorrectly)

	fileLoggingConfiguredCorrectly = true
	_, err = getWriters(&SubLoggerConfig{Output: outputWriters})
	require.NoError(t, err)

	mu.Unlock()

	outputWriters = "stdout|stderr|noobs"
	err = getWritersProtected(&SubLoggerConfig{Output: outputWriters})
	require.ErrorIs(t, err, errUnhandledOutputWriter)
}

func getWritersProtected(s *SubLoggerConfig) error {
	mu.RLock()
	defer mu.RUnlock()
	_, err := getWriters(s)
	return err
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
	mu.Lock()
	defer mu.Unlock()
	err := configureSubLogger("LOG", "INFO", mw)
	if err != nil {
		t.Skipf("configureSubLogger() returned unexpected error %v", err)
	}
	if (Global.levels != Levels{Info: true}) {
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

	f := &fields{output: mw}
	f.output.StageLogEvent(func() string { return "out" }, "header", "SUBLOGGER", " space ", "", "", "", false, false, false, nil)

	<-w.Finished
	if contents := w.Read(); contents != "header space  space out\n" { //nolint:dupword // False positive
		t.Errorf("received: '%v' but expected: '%v'", contents, "header space  space out\n") //nolint:dupword // False positive
	}
}

func TestInfo(t *testing.T) {
	t.Parallel()

	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	sl, err := NewSubLogger("TESTYMCTESTALOTINFO")
	if err != nil {
		t.Fatal(err)
	}
	sl.setLevelsProtected(splitLevel("INFO"))
	err = sl.setOutputProtected(mw)
	if err != nil {
		t.Fatal(err)
	}

	Infof(nil, "%s", "bad")

	Infof(sl, "%s", "hello")
	<-w.Finished
	contents := w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Infoln(nil, "hello", "bad")

	Infoln(sl, "hello", "goodbye")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hellogoodbye") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hellogoodbye")
	}

	_, err = SetLevel("TESTYMCTESTALOTINFO", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it should return if functionality
	// is not enabled.
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

	sl, err := NewSubLogger("TESTYMCTESTALOTDEBUG")
	if err != nil {
		t.Fatal(err)
	}
	sl.setLevelsProtected(splitLevel("DEBUG"))
	err = sl.setOutputProtected(mw)
	if err != nil {
		t.Fatal(err)
	}

	Debugf(nil, "%s", "bad")

	Debugf(sl, "%s", "hello")
	<-w.Finished
	contents := w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Debugln(nil, ":sun_with_face:", "bad")

	Debugln(sl, ":sun_with_face:", ":angrysun:")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, ":sun_with_face::angrysun:") {
		t.Errorf("received: '%v' but expected: '%v'", contents, ":sun_with_face::angrysun:")
	}

	_, err = SetLevel("TESTYMCTESTALOTDEBUG", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it should return if functionality
	// is not enabled.
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

	sl, err := NewSubLogger("TESTYMCTESTALOTWARN")
	if err != nil {
		t.Fatal(err)
	}
	sl.setLevelsProtected(splitLevel("WARN"))
	err = sl.setOutputProtected(mw)
	if err != nil {
		t.Fatal(err)
	}

	Warnf(nil, "%s", "silly")

	Warnf(sl, "%s", "hello")
	<-w.Finished
	contents := w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Warnln(nil, "super", "silly")

	Warnln(sl, "hello", "world")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "helloworld") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "helloworld")
	}

	_, err = SetLevel("TESTYMCTESTALOTWARN", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it shhould return if functionality
	// is not enabled.
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

	sl, err := NewSubLogger("TESTYMCTESTALOTERROR")
	if err != nil {
		t.Fatal(err)
	}
	sl.setLevelsProtected(splitLevel("ERROR"))
	err = sl.setOutputProtected(nil)
	assert.ErrorIs(t, err, errMultiWriterHolderIsNil)

	err = sl.setOutputProtected(mw)
	if err != nil {
		t.Fatal(err)
	}

	Errorf(nil, "%s", "oh wow")

	Errorf(sl, "%s", "hello")
	<-w.Finished
	contents := w.Read()
	if !strings.Contains(contents, "hello") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hello")
	}

	Errorln(nil, "nil", "days")

	Errorln(sl, "hello", "goodbye")
	<-w.Finished
	contents = w.Read()
	if !strings.Contains(contents, "hellogoodbye") {
		t.Errorf("received: '%v' but expected: '%v'", contents, "hellogoodbye")
	}

	_, err = SetLevel("TESTYMCTESTALOTERROR", "")
	if err != nil {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// Should not write to buffer at all as it shhould return if functionality
	// is not enabled.
	Errorln(sl, "HelloHello")
	contents = w.Read()
	if contents != "" {
		t.Errorf("received: '%v' but expected: '%v'", contents, "")
	}
}

func (sl *SubLogger) setLevelsProtected(newLevels Levels) {
	mu.Lock()
	sl.setLevels(newLevels)
	mu.Unlock()
}

func (sl *SubLogger) setOutputProtected(o *multiWriterHolder) error {
	mu.Lock()
	defer mu.Unlock()
	return sl.setOutput(o)
}

func TestSubLoggerName(t *testing.T) {
	t.Parallel()
	w := newTestBuffer()
	mw := &multiWriterHolder{writers: []io.Writer{w}}

	mw.StageLogEvent(func() string { return "out" }, "header", "SUBLOGGER", "||", "", "", time.RFC3339, true, false, false, nil)
	<-w.Finished
	contents := w.Read()
	if !strings.Contains(contents, "SUBLOGGER") {
		t.Error("Expected SUBLOGGER in output")
	}

	mw.StageLogEvent(func() string { return "out" }, "header", "SUBLOGGER", "||", "", "", time.RFC3339, false, false, false, nil)
	<-w.Finished
	contents = w.Read()
	if strings.Contains(contents, "SUBLOGGER") {
		t.Error("Unexpected SUBLOGGER in output")
	}
}

func TestNewSubLogger(t *testing.T) {
	t.Parallel()
	_, err := NewSubLogger("")
	require.ErrorIs(t, err, errEmptyLoggerName)

	sl, err := NewSubLogger("TESTERINOS")
	require.NoError(t, err)

	Debugln(sl, "testerinos")

	_, err = NewSubLogger("TESTERINOS")
	require.ErrorIs(t, err, ErrSubLoggerAlreadyRegistered)
}

func TestRotateWrite(t *testing.T) {
	t.Parallel()
	empty := Rotate{Rotate: convert.BoolPtr(true), FileName: "test.txt"}
	payload := make([]byte, defaultMaxSize*megabyte+1)
	_, err := empty.Write(payload)
	require.ErrorIs(t, err, errExceedsMaxFileSize)

	empty.MaxSize = 1
	payload = make([]byte, 1*megabyte+1)
	_, err = empty.Write(payload)
	require.ErrorIs(t, err, errExceedsMaxFileSize)

	// test write
	payload = make([]byte, 1*megabyte-1)
	_, err = empty.Write(payload)
	require.NoError(t, err)

	// test rotate
	payload = make([]byte, 1*megabyte)
	_, err = empty.Write(payload)
	require.NoError(t, err)

	err = empty.Close()
	require.NoError(t, err)
}

func TestOpenNew(t *testing.T) {
	t.Parallel()
	empty := Rotate{}
	err := empty.openNew()
	require.ErrorIs(t, err, errFileNameIsEmpty)

	empty.FileName = "wow.txt"
	err = empty.openNew()
	require.NoError(t, err)

	err = empty.Close()
	require.NoError(t, err)
}

type testBuffer struct {
	value    []byte
	Finished chan struct{}
}

func (tb *testBuffer) Write(p []byte) (int, error) {
	cpy := make([]byte, len(p))
	copy(cpy, p)
	tb.value = cpy
	tb.Finished <- struct{}{}
	return len(p), nil
}

func (tb *testBuffer) Read() string {
	defer func() { tb.value = tb.value[:0] }()
	return string(tb.value)
}

func (tb *testBuffer) ReadRaw() []byte {
	defer func() { tb.value = tb.value[:0] }()
	cpy := make([]byte, len(tb.value))
	copy(cpy, tb.value)
	return cpy
}

func newTestBuffer() *testBuffer {
	return &testBuffer{Finished: make(chan struct{}, 1)}
}

// 2140294	       770.0 ns/op	       0 B/op	       0 allocs/op
func BenchmarkNewLogEvent(b *testing.B) {
	mw := &multiWriterHolder{writers: []io.Writer{io.Discard}}
	for b.Loop() {
		mw.StageLogEvent(func() string { return "somedata" }, "header", "sublog", "||", "", "", time.RFC3339, true, false, false, nil)
	}
}

// BenchmarkInfo-8   	 1000000	     64971 ns/op	      47 B/op	       1 allocs/op
func BenchmarkInfo(b *testing.B) {
	for b.Loop() {
		Infoln(Global, "Hello this is an info benchmark")
	}
}

// BenchmarkInfoDisabled-8 47124242	        24.16 ns/op	       0 B/op	       0 allocs/op
func BenchmarkInfoDisabled(b *testing.B) {
	if err := SetupDisabled(); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		Infoln(Global, "Hello this is an info benchmark")
	}
}

// BenchmarkInfof-8   	 1000000	     72641 ns/op	     178 B/op	       4 allocs/op
func BenchmarkInfof(b *testing.B) {
	for n := range b.N {
		Infof(Global, "Hello this is an infof benchmark %v %v %v\n", n, 1, 2)
	}
}

// BenchmarkInfoln-8   	 1000000	     68152 ns/op	     121 B/op	       3 allocs/op
func BenchmarkInfoln(b *testing.B) {
	for b.Loop() {
		Infoln(Global, "Hello this is an infoln benchmark")
	}
}

type testCapture struct {
	Message   string    `json:"message"`
	Timestamp int64     `json:"timestamp"`
	Severity  string    `json:"severity"`
	SubLogger string    `json:"sublogger"`
	BotName   string    `json:"botname"`
	ID        uuid.UUID `json:"id"`
}

func TestWithFields(t *testing.T) {
	t.Parallel()
	writer := newTestBuffer()
	mwh := &multiWriterHolder{writers: []io.Writer{writer}}

	sl, err := NewSubLogger("TESTSTRUCTUREDLOGGING")
	if err != nil {
		t.Fatal(err)
	}
	sl.structuredLogging = true
	sl.setLevelsProtected(splitLevel("DEBUG|ERROR|INFO|WARN"))
	err = sl.setOutputProtected(mwh)
	if err != nil {
		t.Fatal(err)
	}

	id, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	ErrorlnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	ErrorlnWithFields(sl, ExtraFields{"id": id}, "hello")
	<-writer.Finished
	var captured testCapture
	bro := writer.ReadRaw()
	err = json.Unmarshal(bro, &captured)
	if err != nil {
		t.Fatal(err, string(bro))
	}
	checkCapture(t, &captured, id, "hello", "error")

	ErrorfWithFields(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	ErrorfWithFields(sl, ExtraFields{"id": id}, "%v", "good")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	checkCapture(t, &captured, id, "good", "error")

	DebuglnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	DebuglnWithFields(sl, ExtraFields{"id": id}, "sir")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	checkCapture(t, &captured, id, "sir", "debug")

	DebugfWithFields(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	DebugfWithFields(sl, ExtraFields{"id": id}, "%v", "how")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	checkCapture(t, &captured, id, "how", "debug")

	WarnlnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	WarnlnWithFields(sl, ExtraFields{"id": id}, "are")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	checkCapture(t, &captured, id, "are", "warn")

	WarnfWithFields(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	WarnfWithFields(sl, ExtraFields{"id": id}, "%v", "you")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	checkCapture(t, &captured, id, "you", "warn")

	InfolnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	InfolnWithFields(sl, ExtraFields{"id": id}, "today")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	checkCapture(t, &captured, id, "today", "info")

	InfofWithFields(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	InfofWithFields(sl, ExtraFields{"id": id}, "%v", "?")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	checkCapture(t, &captured, id, "?", "info")

	// Conflicting fields
	InfofWithFields(nil, ExtraFields{botName: "lol"}, "%v", "nilerinos")
	InfofWithFields(sl, ExtraFields{botName: "lol"}, "%v", "?")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	if err != nil {
		t.Fatal(err)
	}
	if captured.BotName != "test" {
		t.Fatalf("received: '%v' but expected: '%v'", captured.BotName, "test")
	}
}

func checkCapture(t *testing.T, c *testCapture, expID uuid.UUID, expMessage, expSeverity string) {
	t.Helper()

	if c.ID != expID {
		t.Errorf("received: '%v' but expected: '%v'", c.ID, expID)
	}

	if c.Message != expMessage {
		t.Errorf("received: '%v' but expected: '%v'", c.Message, expMessage)
	}

	if c.Severity != expSeverity {
		t.Errorf("received: '%v' but expected: '%v'", c.Severity, expSeverity)
	}

	if c.SubLogger != "TESTSTRUCTUREDLOGGING" {
		t.Errorf("received: '%v' but expected: '%v'", c.SubLogger, "TESTSTRUCTUREDLOGGING")
	}

	if c.BotName != "test" {
		t.Errorf("received: '%v' but expected: '%v'", c.BotName, "test")
	}
}
