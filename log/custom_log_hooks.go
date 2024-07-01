package log

// CustomLogHook is a function type for external log handling. It should return
// true if the library's internal logging system should be bypassed, or false
// if the library's internal logging system should be used.
type CustomLogHook func(header, subLoggerName string, a ...any) (bypassLibraryLogSystem bool)

var customLogHook CustomLogHook

// SetCustomLogHook sets a custom log hook function that allows the complete
// bypass of the library's internal logging system. This is useful for
// implementing custom log handling.
func SetCustomLogHook(h CustomLogHook) {
	mu.Lock()
	customLogHook = h
	mu.Unlock()
}
