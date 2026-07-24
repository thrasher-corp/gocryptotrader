package log

import (
	"errors"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
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
	require.NoError(t, err, "multiWriter must not error")

	payload := "woooooooooooooooooooooooooooooooooooow"
	f.output.StageLogEvent(func() string { return payload }, "", "", "", "", "", "", false, false, false, nil)

	<-buff.Finished
	assert.Contains(t, buff.Read(), payload, "buffer should contain the payload")

	f.output, err = multiWriter(&WriteShorter{}, io.Discard)
	require.NoError(t, err, "multiWriter must not error")
	f.output.StageLogEvent(func() string { return payload }, "", "", "", "", "", "", false, false, false, nil) // Will display error: Logger write error: *log.WriteShorter short write

	f.output, err = multiWriter(&WriteError{}, io.Discard)
	require.NoError(t, err, "multiWriter must not error")
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

func TestGetFields(t *testing.T) {
	mu.Lock()
	originalConfig := globalLogConfig
	originalLogger := logger
	originalPool := logFieldsPool
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		globalLogConfig = originalConfig
		logger = originalLogger
		logFieldsPool = originalPool
		mu.Unlock()
	})

	enabled := true
	mu.Lock()
	globalLogConfig = &Config{Enabled: &enabled}
	mu.Unlock()
	mu.RLock()
	f := (*SubLogger)(nil).getFields()
	mu.RUnlock()
	assert.Nil(t, f, "getFields should return nil for a nil sublogger")

	sl := &SubLogger{}
	mu.Lock()
	globalLogConfig = nil
	mu.Unlock()
	mu.RLock()
	f = sl.getFields()
	mu.RUnlock()
	assert.Nil(t, f, "getFields should return nil without a global config")

	mu.Lock()
	globalLogConfig = &Config{}
	mu.Unlock()
	mu.RLock()
	f = sl.getFields()
	mu.RUnlock()
	assert.Nil(t, f, "getFields should return nil without an enabled setting")

	mu.Lock()
	enabled = false
	globalLogConfig.Enabled = &enabled
	mu.Unlock()
	mu.RLock()
	f = sl.getFields()
	mu.RUnlock()
	assert.Nil(t, f, "getFields should return nil when logging is disabled")

	output := &multiWriterHolder{}
	sl = &SubLogger{
		name:              "FIELDS",
		levels:            Levels{Info: true, Debug: true, Warn: true, Error: true},
		output:            output,
		botName:           "bot",
		structuredLogging: true,
	}
	currentLogger := Logger{InfoHeader: "current"}
	staleLogger := Logger{InfoHeader: "stale"}
	mu.Lock()
	enabled = true
	logger = staleLogger
	pooled := originalPool.New().(*fields) //nolint:forcetypeassert // Not necessary from a pool
	pooled.structuredFields = ExtraFields{"stale": true}
	logFieldsPool = &sync.Pool{New: func() any { return pooled }}
	logger = currentLogger
	mu.Unlock()
	mu.RLock()
	f = sl.getFields()
	if f == nil {
		mu.RUnlock()
		require.NotNil(t, f, "getFields result must not be nil when logging is enabled")
		return
	}
	assert.Equal(t, sl.levels.Info, f.info, "getFields should copy the info level")
	assert.Equal(t, sl.levels.Debug, f.debug, "getFields should copy the debug level")
	assert.Equal(t, sl.levels.Warn, f.warn, "getFields should copy the warn level")
	assert.Equal(t, sl.levels.Error, f.error, "getFields should copy the error level")
	assert.Equal(t, sl.name, f.name, "getFields should copy the sublogger name")
	assert.Same(t, output, f.output, "getFields should copy the output")
	assert.Equal(t, sl.botName, f.botName, "getFields should copy the bot name")
	assert.Equal(t, sl.structuredLogging, f.structuredLogging, "getFields should copy structured logging")
	assert.Same(t, &logger, f.logger, "getFields should use the current logger")
	assert.Equal(t, currentLogger, *f.logger, "getFields should return the current logger")
	assert.Nil(t, f.structuredFields, "getFields should discard stale structured fields")
	logFieldsPool.Put(f)
	mu.RUnlock()
}

func TestPooledFieldsUseCurrentLogger(t *testing.T) {
	w := newTestBuffer()
	sl := &SubLogger{
		name:   "CURRENT",
		levels: Levels{Info: true},
		output: &multiWriterHolder{writers: []io.Writer{w}},
	}
	enabled := true
	currentLogger := Logger{InfoHeader: "current", Spacer: " "}
	staleLogger := Logger{InfoHeader: "stale", Spacer: " "}

	mu.Lock()
	originalConfig := globalLogConfig
	originalHook := customLogHook
	originalLogger := logger
	originalPool := logFieldsPool
	globalLogConfig = &Config{Enabled: &enabled}
	customLogHook = nil
	logger = staleLogger
	pooled := originalPool.New().(*fields) //nolint:forcetypeassert // Not necessary from a pool
	logFieldsPool = &sync.Pool{New: func() any { return pooled }}
	logger = currentLogger
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		globalLogConfig = originalConfig
		customLogHook = originalHook
		logger = originalLogger
		logFieldsPool = originalPool
		mu.Unlock()
	})

	Infoln(sl, "message")
	barrier := make(chan struct{})
	jobsChannel <- &job{Passback: barrier}
	<-barrier
	wroteOutput := false
	select {
	case <-w.Finished:
		wroteOutput = true
	default:
	}
	assert.True(t, wroteOutput, "Infoln should write with the current logger")
	output := w.Read()
	assert.Contains(t, output, currentLogger.InfoHeader, "Infoln should use the current logger header")
	assert.NotContains(t, output, staleLogger.InfoHeader, "Infoln should not use a stale logger header")
}

func TestFieldsCustomLogHook(t *testing.T) {
	testLogger := Logger{InfoHeader: "[INFO]", Spacer: " "}
	mu.Lock()
	originalHook := customLogHook
	originalPool := logFieldsPool
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		customLogHook = originalHook
		logFieldsPool = originalPool
		mu.Unlock()
	})

	for _, tc := range []struct {
		name             string
		operation        string
		hookSet          bool
		bypass           bool
		call             func(*fields)
		expectedHookArgs []any
		expectedOutput   string
	}{
		{name: "stageln without hook", operation: "stageln", call: func(f *fields) { f.stageln(testLogger.InfoHeader, "message", 1) }, expectedOutput: "message1"},
		{name: "stageln hook falls through", operation: "stageln", hookSet: true, call: func(f *fields) { f.stageln(testLogger.InfoHeader, "message", 1) }, expectedHookArgs: []any{"message", 1}, expectedOutput: "message1"},
		{name: "stageln hook bypasses", operation: "stageln", hookSet: true, bypass: true, call: func(f *fields) { f.stageln(testLogger.InfoHeader, "message", 1) }, expectedHookArgs: []any{"message", 1}},
		{name: "stagef without hook", operation: "stagef", call: func(f *fields) { f.stagef(testLogger.InfoHeader, "formatted %s %d", "message", 1) }, expectedOutput: "formatted message 1"},
		{name: "stagef hook falls through", operation: "stagef", hookSet: true, call: func(f *fields) { f.stagef(testLogger.InfoHeader, "formatted %s %d", "message", 1) }, expectedHookArgs: []any{"formatted message 1"}, expectedOutput: "formatted message 1"},
		{name: "stagef hook bypasses", operation: "stagef", hookSet: true, bypass: true, call: func(f *fields) { f.stagef(testLogger.InfoHeader, "formatted %s %d", "message", 1) }, expectedHookArgs: []any{"formatted message 1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newTestBuffer()
			var hookArgs []any
			mu.Lock()
			if tc.hookSet {
				customLogHook = func(_, _ string, a ...any) bool {
					hookArgs = append(hookArgs, a...)
					return tc.bypass
				}
			} else {
				customLogHook = nil
			}
			logFieldsPool = &sync.Pool{New: func() any {
				return &fields{
					info:   true,
					name:   "HOOK",
					output: &multiWriterHolder{writers: []io.Writer{w}},
					logger: &testLogger,
				}
			}}
			f := logFieldsPool.Get().(*fields) //nolint:forcetypeassert // Not necessary from a pool
			mu.Unlock()

			mu.RLock()
			tc.call(f)
			mu.RUnlock()
			barrier := make(chan struct{})
			jobsChannel <- &job{Passback: barrier}
			<-barrier
			wroteOutput := false
			select {
			case <-w.Finished:
				wroteOutput = true
			default:
			}
			if tc.expectedOutput != "" {
				assert.Truef(t, wroteOutput, "%s should write internal output", tc.operation)
				assert.Containsf(t, w.Read(), tc.expectedOutput, "%s should write the correct output", tc.operation)
			} else {
				assert.Falsef(t, wroteOutput, "%s should bypass internal output", tc.operation)
				assert.Emptyf(t, w.Read(), "%s should not write internal output", tc.operation)
			}
			assert.Equalf(t, tc.expectedHookArgs, hookArgs, "%s should pass the correct hook arguments", tc.operation)
		})
	}

	message := []byte("before")
	w := newTestBuffer()
	mu.Lock()
	customLogHook = func(_, _ string, a ...any) bool {
		assert.Equal(t, []any{"before"}, a, "stagef should pass the formatted message to the hook")
		copy(message, "after!")
		return false
	}
	logFieldsPool = &sync.Pool{New: func() any {
		return &fields{
			info:   true,
			name:   "HOOK",
			output: &multiWriterHolder{writers: []io.Writer{w}},
			logger: &testLogger,
		}
	}}
	f := logFieldsPool.Get().(*fields) //nolint:forcetypeassert // Not necessary from a pool
	mu.Unlock()

	mu.RLock()
	f.stagef(testLogger.InfoHeader, "%s", message)
	mu.RUnlock()
	barrier := make(chan struct{})
	jobsChannel <- &job{Passback: barrier}
	<-barrier
	wroteSnapshot := false
	select {
	case <-w.Finished:
		wroteSnapshot = true
	default:
	}
	assert.True(t, wroteSnapshot, "stagef should write fall-through output")
	output := w.Read()
	assert.Contains(t, output, "before", "stagef should reuse the hook-formatted message")
	assert.NotContains(t, output, "after!", "stagef should not format mutable arguments twice")
}

func TestFieldsCustomLogHookRecyclesFields(t *testing.T) {
	previousMaxProcs := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previousMaxProcs) })
	previousGCPercent := debug.SetGCPercent(-1)
	t.Cleanup(func() { debug.SetGCPercent(previousGCPercent) })

	testLogger := Logger{InfoHeader: "[INFO]"}
	mu.Lock()
	originalHook := customLogHook
	originalPool := logFieldsPool
	customLogHook = func(_, _ string, _ ...any) bool { return true }
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		customLogHook = originalHook
		logFieldsPool = originalPool
		mu.Unlock()
	})

	for _, tc := range []struct {
		name string
		call func(*fields)
	}{
		{name: "stageln", call: func(f *fields) { f.stageln(testLogger.InfoHeader, "message") }},
		{name: "stagef", call: func(f *fields) { f.stagef(testLogger.InfoHeader, "%s", "message") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const attempts = 1000
			// Race builds may deliberately discard sync.Pool puts, so sample
			// enough operations to distinguish drops from a missing Put path.
			poolMisses := 0
			mu.Lock()
			logFieldsPool = &sync.Pool{New: func() any {
				poolMisses++
				return &fields{name: "HOOK", logger: &testLogger}
			}}
			mu.Unlock()

			for range attempts {
				mu.RLock()
				f := logFieldsPool.Get().(*fields) //nolint:forcetypeassert // Not necessary from a pool
				tc.call(f)
				mu.RUnlock()
			}
			assert.Lessf(t, poolMisses, attempts/2, "%s should return fields to the pool", tc.name)
		})
	}
}

func TestLoggersDiscardStaleStructuredFields(t *testing.T) {
	mu.Lock()
	originalPool := logFieldsPool
	originalHook := customLogHook
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		logFieldsPool = originalPool
		customLogHook = originalHook
		mu.Unlock()
	})

	for _, tc := range []struct {
		name                string
		withFieldsOperation string
		logWithFields       func(*SubLogger, ExtraFields, string)
		logPlain            func(*SubLogger, string)
	}{
		{name: "Infoln", withFieldsOperation: "InfolnWithFields", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { InfolnWithFields(sl, extra, message) }, logPlain: func(sl *SubLogger, message string) { Infoln(sl, message) }},
		{name: "Infof", withFieldsOperation: "InfoWithFieldsf", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { InfoWithFieldsf(sl, extra, "%s", message) }, logPlain: func(sl *SubLogger, message string) { Infof(sl, "%s", message) }},
		{name: "Debugln", withFieldsOperation: "DebuglnWithFields", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { DebuglnWithFields(sl, extra, message) }, logPlain: func(sl *SubLogger, message string) { Debugln(sl, message) }},
		{name: "Debugf", withFieldsOperation: "DebugWithFieldsf", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { DebugWithFieldsf(sl, extra, "%s", message) }, logPlain: func(sl *SubLogger, message string) { Debugf(sl, "%s", message) }},
		{name: "Warnln", withFieldsOperation: "WarnlnWithFields", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { WarnlnWithFields(sl, extra, message) }, logPlain: func(sl *SubLogger, message string) { Warnln(sl, message) }},
		{name: "Warnf", withFieldsOperation: "WarnWithFieldsf", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { WarnWithFieldsf(sl, extra, "%s", message) }, logPlain: func(sl *SubLogger, message string) { Warnf(sl, "%s", message) }},
		{name: "Errorln", withFieldsOperation: "ErrorlnWithFields", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { ErrorlnWithFields(sl, extra, message) }, logPlain: func(sl *SubLogger, message string) { Errorln(sl, message) }},
		{name: "Errorf", withFieldsOperation: "ErrorWithFieldsf", logWithFields: func(sl *SubLogger, extra ExtraFields, message string) { ErrorWithFieldsf(sl, extra, "%s", message) }, logPlain: func(sl *SubLogger, message string) { Errorf(sl, "%s", message) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newTestBuffer()
			sl := &SubLogger{
				name:              "PLAIN",
				levels:            Levels{Info: true, Debug: true, Warn: true, Error: true},
				output:            &multiWriterHolder{writers: []io.Writer{w}},
				structuredLogging: true,
			}
			pooled := &fields{logger: &logger}
			mu.Lock()
			customLogHook = nil
			logFieldsPool = &sync.Pool{New: func() any { return pooled }}
			mu.Unlock()

			tc.logWithFields(sl, ExtraFields{"stale": true}, "structured")
			barrier := make(chan struct{})
			jobsChannel <- &job{Passback: barrier}
			<-barrier
			wroteStructured := false
			select {
			case <-w.Finished:
				wroteStructured = true
			default:
			}
			assert.Truef(t, wroteStructured, "%s should write structured output", tc.withFieldsOperation)
			assert.Containsf(t, w.Read(), "stale", "%s should write structured fields", tc.withFieldsOperation)

			tc.logPlain(nil, "ignored")
			tc.logPlain(sl, tc.name)
			barrier = make(chan struct{})
			jobsChannel <- &job{Passback: barrier}
			<-barrier
			wrotePlain := false
			select {
			case <-w.Finished:
				wrotePlain = true
			default:
			}
			assert.Truef(t, wrotePlain, "%s should write plain output", tc.name)
			output := w.Read()
			assert.Containsf(t, output, tc.name, "%s should write the correct output", tc.name)
			assert.NotContainsf(t, output, "stale", "%s should discard stale structured fields", tc.name)

			sl.setLevelsProtected(Levels{})
			tc.logPlain(sl, "ignored")
			barrier = make(chan struct{})
			jobsChannel <- &job{Passback: barrier}
			<-barrier
			wroteDisabled := false
			select {
			case <-w.Finished:
				wroteDisabled = true
			default:
			}
			assert.Falsef(t, wroteDisabled, "%s should not write when disabled", tc.name)
			assert.Emptyf(t, w.Read(), "%s should not stage disabled output", tc.name)
		})
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

// BenchmarkFormattedDisabled measures level-disabled formatted logging while
// global logging remains enabled.
func BenchmarkFormattedDisabled(b *testing.B) {
	enabled := true
	benchmarkLogger := Logger{
		InfoHeader:  "[INFO]",
		DebugHeader: "[DEBUG]",
		WarnHeader:  "[WARN]",
		ErrorHeader: "[ERROR]",
	}
	mu.Lock()
	originalEnabled := globalLogConfig.Enabled
	originalHook := customLogHook
	originalLogger := logger
	originalPool := logFieldsPool
	globalLogConfig.Enabled = &enabled
	customLogHook = nil
	logger = benchmarkLogger
	logFieldsPool = &sync.Pool{New: func() any { return &fields{logger: &logger} }}
	mu.Unlock()
	b.Cleanup(func() {
		mu.Lock()
		globalLogConfig.Enabled = originalEnabled
		customLogHook = originalHook
		logger = originalLogger
		logFieldsPool = originalPool
		mu.Unlock()
	})

	barrier := make(chan struct{})
	jobsChannel <- &job{Passback: barrier}
	<-barrier

	sl := &SubLogger{}
	for _, tc := range []struct {
		name string
		logf func(*SubLogger, string, ...any)
	}{
		{name: "Infof", logf: Infof},
		{name: "Debugf", logf: Debugf},
		{name: "Warnf", logf: Warnf},
		{name: "Errorf", logf: Errorf},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tc.logf(sl, "formatted %s", "message")
			}
		})
	}
}

// BenchmarkCustomLogHookBypass measures logging when a custom hook handles the
// event and bypasses the internal logging system.
func BenchmarkCustomLogHookBypass(b *testing.B) {
	enabled := true
	benchmarkLogger := Logger{
		InfoHeader:  "[INFO]",
		DebugHeader: "[DEBUG]",
		WarnHeader:  "[WARN]",
		ErrorHeader: "[ERROR]",
	}
	mu.Lock()
	originalEnabled := globalLogConfig.Enabled
	originalHook := customLogHook
	originalLogger := logger
	originalPool := logFieldsPool
	globalLogConfig.Enabled = &enabled
	customLogHook = func(_, _ string, _ ...any) bool { return true }
	logger = benchmarkLogger
	logFieldsPool = &sync.Pool{New: func() any { return &fields{logger: &logger} }}
	mu.Unlock()
	b.Cleanup(func() {
		mu.Lock()
		globalLogConfig.Enabled = originalEnabled
		customLogHook = originalHook
		logger = originalLogger
		logFieldsPool = originalPool
		mu.Unlock()
	})

	sl := &SubLogger{name: "BENCHMARK"}
	for _, tc := range []struct {
		name string
		log  func(*SubLogger)
	}{
		{name: "Infoln", log: func(sl *SubLogger) { Infoln(sl, "message") }},
		{name: "Infof", log: func(sl *SubLogger) { Infof(sl, "formatted %s", "message") }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tc.log(sl)
			}
		})
	}
}

// BenchmarkCustomLogHookFallthrough measures formatted logging when a custom
// hook observes the event and the internal logger still handles it.
func BenchmarkCustomLogHookFallthrough(b *testing.B) {
	enabled := true
	benchmarkLogger := Logger{
		InfoHeader:                    "[INFO]",
		Spacer:                        " ",
		BypassJobChannelFilledWarning: true,
	}
	mu.Lock()
	originalEnabled := globalLogConfig.Enabled
	originalHook := customLogHook
	originalLogger := logger
	originalPool := logFieldsPool
	globalLogConfig.Enabled = &enabled
	customLogHook = func(_, _ string, _ ...any) bool { return false }
	logger = benchmarkLogger
	logFieldsPool = &sync.Pool{New: func() any { return &fields{logger: &logger} }}
	mu.Unlock()
	b.Cleanup(func() {
		mu.Lock()
		globalLogConfig.Enabled = originalEnabled
		customLogHook = originalHook
		logger = originalLogger
		logFieldsPool = originalPool
		mu.Unlock()
	})

	barrier := make(chan struct{})
	jobsChannel <- &job{Passback: barrier}
	<-barrier

	sl := &SubLogger{
		name:   "BENCHMARK",
		levels: Levels{Info: true},
		output: &multiWriterHolder{writers: []io.Writer{io.Discard}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		Infof(sl, "formatted %s %d", "message", 1)
	}
	barrier = make(chan struct{})
	jobsChannel <- &job{Passback: barrier}
	<-barrier
	b.StopTimer()
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
	require.NoError(t, err, "NewSubLogger must not error")
	sl.structuredLogging = true
	sl.setLevelsProtected(splitLevel("DEBUG|ERROR|INFO|WARN"))
	err = sl.setOutputProtected(mwh)
	require.NoError(t, err, "setOutputProtected must not error")

	id, err := uuid.NewV4()
	require.NoError(t, err, "uuid.NewV4 must not error")

	ErrorlnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	ErrorlnWithFields(sl, ExtraFields{"id": id}, "hello")
	<-writer.Finished
	var captured testCapture
	bro := writer.ReadRaw()
	err = json.Unmarshal(bro, &captured)
	require.NoErrorf(t, err, "json.Unmarshal must not error: %s", string(bro))
	checkCapture(t, &captured, id, "hello", "error")

	ErrorWithFieldsf(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	ErrorWithFieldsf(sl, ExtraFields{"id": id}, "%v", "good")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	checkCapture(t, &captured, id, "good", "error")

	DebuglnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	DebuglnWithFields(sl, ExtraFields{"id": id}, "sir")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	checkCapture(t, &captured, id, "sir", "debug")

	DebugWithFieldsf(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	DebugWithFieldsf(sl, ExtraFields{"id": id}, "%v", "how")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	checkCapture(t, &captured, id, "how", "debug")

	WarnlnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	WarnlnWithFields(sl, ExtraFields{"id": id}, "are")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	checkCapture(t, &captured, id, "are", "warn")

	WarnWithFieldsf(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	WarnWithFieldsf(sl, ExtraFields{"id": id}, "%v", "you")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	checkCapture(t, &captured, id, "you", "warn")

	InfolnWithFields(nil, ExtraFields{"id": id}, "nilerinos")
	InfolnWithFields(sl, ExtraFields{"id": id}, "today")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	checkCapture(t, &captured, id, "today", "info")

	InfoWithFieldsf(nil, ExtraFields{"id": id}, "%v", "nilerinos")
	InfoWithFieldsf(sl, ExtraFields{"id": id}, "%v", "?")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	checkCapture(t, &captured, id, "?", "info")

	// Conflicting fields
	InfoWithFieldsf(nil, ExtraFields{botName: "lol"}, "%v", "nilerinos")
	InfoWithFieldsf(sl, ExtraFields{botName: "lol"}, "%v", "?")
	<-writer.Finished
	err = json.Unmarshal(writer.ReadRaw(), &captured)
	require.NoError(t, err, "json.Unmarshal must not error")
	require.Equal(t, "test", captured.BotName, "BotName must match expected value")
}

func checkCapture(t *testing.T, c *testCapture, expID uuid.UUID, expMessage, expSeverity string) {
	t.Helper()

	assert.Equal(t, expID, c.ID, "ID should match")
	assert.Equal(t, expMessage, c.Message, "Message should match")
	assert.Equal(t, expSeverity, c.Severity, "Severity should match")
	assert.Equal(t, "TESTSTRUCTUREDLOGGING", c.SubLogger, "SubLogger should match")
	assert.Equal(t, "test", c.BotName, "BotName should match")
}
