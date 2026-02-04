package livetest

import (
	"os"
	"strings"
)

// LiveTestingSkipped is the log message when live testing is skipped
const LiveTestingSkipped = "Live testing skipped for %s exchange"

// ShouldSkip returns true when CI should avoid live endpoint testing
func ShouldSkip() bool {
	return envIsTrue("GCT_SKIP_LIVE_TESTS")
}

func envIsTrue(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	return strings.EqualFold(value, "true") || value == "1"
}
