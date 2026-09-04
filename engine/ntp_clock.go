package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/beevik/ntp"
)

const (
	ntpDialTimeout  = 5 * time.Second
	ntpIOTimeout    = 5 * time.Second
	maximumDuration = time.Duration(1<<63 - 1)
	minimumDuration = -maximumDuration - 1
)

var (
	errNoNTPPoolsConfigured    = errors.New("no NTP pools configured")
	errNoValidNTPServer        = errors.New("no valid NTP server")
	errNegativeNTPRoundTrip    = errors.New("negative NTP round-trip time")
	errNegativeNTPRootDistance = errors.New("negative NTP root distance")
	errNTPRootDistanceBelowRTT = errors.New("NTP root distance below half round-trip time")
)

// clockCorrection is server time minus local time: a positive value means the local clock is behind.
// uncertainty is the NTP root distance. It bounds how far clockCorrection can be from the true offset.
type clockObservation struct {
	source          string
	clockCorrection time.Duration
	uncertainty     time.Duration
	roundTrip       time.Duration
}

type queryClockFunc func(context.Context, string) (clockObservation, error)

type clockMeasurer struct {
	queryOne queryClockFunc
}

// Servers are asked in order and the first valid reply wins.
// Selection never depends on the answer: a server is not skipped because its result is inconvenient.
func (m clockMeasurer) measure(ctx context.Context, pools []string) (clockObservation, error) {
	if len(pools) == 0 {
		return clockObservation{}, errNoNTPPoolsConfigured
	}

	failures := make([]error, 0, len(pools))
	for i := range pools {
		if err := ctx.Err(); err != nil {
			return clockObservation{}, err
		}

		observation, err := m.queryOne(ctx, pools[i])
		// Drop even a successful observation after cancellation so shutdown
		// cannot reach the startup prompt.
		if contextErr := ctx.Err(); contextErr != nil {
			return clockObservation{}, contextErr
		}
		if err == nil {
			return observation, nil
		}
		failures = append(failures, fmt.Errorf("NTP server %q: %w", pools[i], err))
	}

	return clockObservation{}, fmt.Errorf("%w: %w", errNoValidNTPServer, errors.Join(failures...))
}

func queryClock(ctx context.Context, source string) (clockObservation, error) {
	if err := ctx.Err(); err != nil {
		return clockObservation{}, err
	}

	dialer := &net.Dialer{Timeout: ntpDialTimeout}
	response, err := ntp.QueryWithOptions(source, ntp.QueryOptions{
		Timeout: ntpIOTimeout,
		Dialer: func(_, remoteAddress string) (net.Conn, error) {
			return dialer.DialContext(ctx, "udp", remoteAddress)
		},
	})
	if err != nil {
		return clockObservation{}, fmt.Errorf("query: %w", err)
	}
	if err := response.Validate(); err != nil {
		return clockObservation{}, fmt.Errorf("validate response: %w", err)
	}
	return observationFromResponse(source, response)
}

func observationFromResponse(source string, response *ntp.Response) (clockObservation, error) {
	if response.RTT < 0 {
		return clockObservation{}, errNegativeNTPRoundTrip
	}
	if response.RootDistance < 0 {
		return clockObservation{}, errNegativeNTPRootDistance
	}
	// Path asymmetry can put all of the delay in one direction, so half the round-trip
	// time is the smallest honest uncertainty. A smaller value means the reply contradicts itself.
	if response.RootDistance < response.RTT/2 {
		return clockObservation{}, errNTPRootDistanceBelowRTT
	}

	return clockObservation{
		source:          source,
		clockCorrection: response.ClockOffset,
		uncertainty:     response.RootDistance,
		roundTrip:       response.RTT,
	}, nil
}

type clockSyncState uint8

const (
	clockSyncInconclusive clockSyncState = iota
	clockSyncInSync
	clockSyncAhead
	clockSyncBehind
)

// evaluateClock compares the complete uncertainty interval with the tolerance window.
// A single value would report network delay as clock error, so a result is conclusive
// only when the whole interval falls outside the window.
func evaluateClock(
	observation clockObservation,
	allowedDifference,
	allowedNegativeDifference time.Duration,
) clockSyncState {
	low := saturatingSubtract(observation.clockCorrection, observation.uncertainty)
	high := saturatingAdd(observation.clockCorrection, observation.uncertainty)
	// A positive correction means the local clock is behind, so allowedDifference is the upper bound.
	behindBound := allowedDifference
	aheadBound := -allowedNegativeDifference

	switch {
	case low > behindBound:
		return clockSyncBehind
	case high < aheadBound:
		return clockSyncAhead
	case low >= aheadBound && high <= behindBound:
		return clockSyncInSync
	default:
		return clockSyncInconclusive
	}
}

// Clamping keeps evaluateClock total: a very large correction cannot wrap and give the opposite result.
func saturatingAdd(a, b time.Duration) time.Duration {
	if b > 0 && a > maximumDuration-b {
		return maximumDuration
	}
	if b < 0 && a < minimumDuration-b {
		return minimumDuration
	}
	return a + b
}

func saturatingSubtract(a, b time.Duration) time.Duration {
	if b > 0 && a < minimumDuration+b {
		return minimumDuration
	}
	if b < 0 && a > maximumDuration+b {
		return maximumDuration
	}
	return a - b
}
