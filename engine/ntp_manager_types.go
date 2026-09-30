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
// With all four replies, verifying a correct clock needs two nearby servers, otherwise distant servers' wide ranges form the majority.
// Cloudflare and SIDN's any.time.nl are anycast, which covers most regions, and NIST and NICT are independent national time labs.
// Smeared and unsmeared time can differ by up to a second around a leap second, so all four insert leap seconds instead.
// Cloudflare, SIDN and NIST state this, and NICT documented it for the 2015 leap second.
// Sources: https://developers.cloudflare.com/time-services/ntp/, https://time.nl/, https://www.nist.gov/pml/time-and-frequency-division/time-distribution/internet-time-service-its, https://www.nict.go.jp/en/data/nict-news/NICT_NEWS_1504_E.pdf
// One query per server every 15 minutes respects NIST's 4-second minimum spacing and NICT's limit of 20 queries an hour: https://tf.nist.gov/tf-cgi/servers.cgi, https://www.nict.go.jp/en/sts/ntp_faq.html
// NTP Pool forbids shipping plain pool.ntp.org as a default, and GCT has no Pool vendor zone: https://www.ntppool.org/en/vendors.html
var defaultNTPServers = []string{"time.cloudflare.com:123", "any.time.nl:123", "time.nist.gov:123", "ntp.nict.jp:123"}

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
