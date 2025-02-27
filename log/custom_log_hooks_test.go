package log

import "testing"

func TestSetCustomLoghook(t *testing.T) {
	t.Parallel()
	logHook := func(_ string, _ string, _ ...any) (bypassLibraryLogSystem bool) {
		return false
	}
	SetCustomLogHook(logHook)
}
