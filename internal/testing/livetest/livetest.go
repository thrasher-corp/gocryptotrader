package livetest

import (
	"os"
	"strings"
)

// LiveTestingSkipped is the log message when live testing is skipped
const LiveTestingSkipped = "Live testing skipped for %s exchange"

// ShouldSkipLiveTests returns true when CI should avoid live endpoint testing
func ShouldSkipLiveTests() bool {
	return envIsTrue("GCT_SKIP_LIVE_TESTS")
}

func envIsTrue(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	return strings.EqualFold(value, "true") || value == "1"
}
