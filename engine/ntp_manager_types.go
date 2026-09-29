package engine

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	defaultNTPCheckInterval = 15 * time.Minute
	ntpClockWatchInterval   = 15 * time.Second
	// NTPManagerName is an exported subsystem name.
	NTPManagerName = "ntp_timekeeper"
)

var (
	errNilNTPConfigValues = errors.New("nil allowed time differences received")
	errNTPManagerDisabled = errors.New("NTP manager disabled")
	errNTPConfig          = errors.New("invalid NTP configuration")
	errNTPClockChanged    = errors.New("local clock changed during the observation")
)

// defaultNTPServers are used when configured pool is empty.
// Four organisations give independent votes, and none smears leap seconds, so they agree during a leap second: https://developers.cloudflare.com/time-services/ntp/
// NTP Pool forbids shipping plain pool.ntp.org as a default, and GCT has no Pool vendor zone: https://www.ntppool.org/en/vendors.html
// NIST refuses clients querying more than once every 4 seconds, far below GCT's 15-minute interval: https://tf.nist.gov/tf-cgi/servers.cgi
var defaultNTPServers = []string{"time.cloudflare.com:123", "time.nist.gov:123", "ptbtime1.ptb.de:123", "ntp.se:123"}

type ntpRun struct {
	cancel context.CancelFunc
	done   chan struct{}
}

type ntpClockReading struct {
	wall    time.Time
	elapsed time.Duration
	// awake stops while computer sleeps.
	// It is read explicitly because Go's clock doesn't stop on Windows.
	awake time.Duration
}

type ntpManager struct {
	lifecycle                 sync.Mutex
	run                       *ntpRun
	level                     int
	allowedDifference         time.Duration
	allowedNegativeDifference time.Duration
	survey                    *ntpSurvey
	loggingEnabled            bool
	clock                     func() (ntpClockReading, error)
	interval                  func() time.Duration

	// Only worker goroutine uses these fields.
	// Stop waits for worker to exit before a new one starts.
	nextRound time.Duration
	history   ntpReporting
}
