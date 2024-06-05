package log

import "testing"

func TestSetCustomLoghook(t *testing.T) {
	t.Parallel()
	logHook := func(header, subLoggerName string, a ...interface{}) (bypassLibraryLogSystem bool) {
		return false
	}
	SetCustomLogHook(logHook)
}
