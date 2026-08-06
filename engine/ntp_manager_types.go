package engine

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

const (
	defaultNTPCheckInterval = 30 * time.Minute
	ntpStartupBudget        = 10 * time.Second
	// NTPManagerName is an exported subsystem name
	NTPManagerName = "ntp_timekeeper"
)

var (
	errNilNTPConfigValues = errors.New("nil allowed time differences received")
	errNTPManagerDisabled = errors.New("NTP manager disabled")
)

// ntpManager starts the NTP manager
type ntpManager struct {
	started                   atomic.Bool
	shutdown                  chan struct{}
	level                     int
	allowedDifference         time.Duration
	allowedNegativeDifference time.Duration
	pools                     []string
	checkInterval             time.Duration
	measure                   func(context.Context, []string) (clockObservation, error)
}
