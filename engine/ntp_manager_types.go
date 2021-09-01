package engine

import (
	"errors"
	"time"
)

const (
	defaultNTPCheckInterval = time.Second * 30
	defaultRetryLimit       = 3
	// NTPManagerName is an exported subsystem name
	NTPManagerName = "ntp_timekeeper"
)

var (
	errNilNTPConfigValues = errors.New("nil allowed time differences received")
	errNTPManagerDisabled = errors.New("NTP manager disabled")
)

// ntpManager starts the NTP manager
type ntpManager struct {
	started                   int32
	shutdown                  chan struct{}
	level                     int64
	allowedDifference         time.Duration
	allowedNegativeDifference time.Duration
	pools                     []string
	checkInterval             time.Duration
	retryLimit                int
	loggingEnabled            bool
}

type ntpPacket struct {
	Settings       uint8  // leap yr indicator, ver number, and mode
	Stratum        uint8  // stratum of local clock
	Poll           int8   // poll exponent
	Precision      int8   // precision exponent
	RootDelay      uint32 // root delay
	RootDispersion uint32 // root dispersion
	ReferenceID    uint32 // reference id
	RefTimeSec     uint32 // reference timestamp sec
	RefTimeFrac    uint32 // reference timestamp fractional
	OrigTimeSec    uint32 // origin time secs
	OrigTimeFrac   uint32 // origin time fractional
	RxTimeSec      uint32 // receive time secs
	RxTimeFrac     uint32 // receive time frac
	TxTimeSec      uint32 // transmit time secs
	TxTimeFrac     uint32 // transmit time frac
}
