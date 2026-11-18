package request

import (
	"time"
)

// Reporter interface groups observability functionality over
// HTTP request latency.
type Reporter interface {
	Latency(name, method, path string, t time.Duration)
}

// SetupGlobalReporter sets a reporter interface to be used
// for all exchange requests
func SetupGlobalReporter(r Reporter) {
	globalReporter = r
}
