package engine

import (
	"errors"
	"slices"
	"time"
)

var (
	errNTPInsufficientSources = errors.New("not enough time servers responded")
	errNTPDisagreement        = errors.New("time servers do not establish a single majority interval")
)

// ntpInterval is estimated range of how far local clock is off (server time minus local time).
// Positive means computer's clock is behind, matching beevik's ClockOffset sign.
type ntpInterval struct {
	lower time.Duration
	upper time.Duration
}

type ntpClockState uint8

const (
	ntpUnknown ntpClockState = iota
	ntpHealthy
	ntpAhead
	ntpBehind
)

func (interval ntpInterval) classify(allowedDifference, allowedNegativeDifference time.Duration) ntpClockState {
	switch {
	case interval.lower >= -allowedNegativeDifference && interval.upper <= allowedDifference:
		return ntpHealthy
	case interval.upper < -allowedNegativeDifference:
		return ntpAhead
	case interval.lower > allowedDifference:
		return ntpBehind
	default:
		return ntpUnknown
	}
}

// selectNTPInterval finds offsets that a majority of configured servers agree on.
// Servers that didn't answer count towards configured total, so failures can't lower required majority.
// If agreed offsets form separate ranges, result is unknown rather than picking one.
// This is GCT policy, not a translation of RFC or ntpd-rs selector.
// ntpd-rs instead selects a range where the largest number of server ranges overlap.
// If more than half of configured servers give ranges including actual clock error, returned range also includes it.
// With four servers configured and only three answering, one wrong range can exclude actual clock error despite agreement.
// https://www.rfc-editor.org/rfc/rfc5905.html#section-11.2.1
// https://github.com/pendulum-project/ntpd-rs/blob/4cb963e34652530e2da46c8b812b00f9ce129dd8/statime-algo/src/filter.rs#L864
func selectNTPInterval(samples []ntpInterval, configured int) (ntpInterval, int, error) {
	quorum := configured/2 + 1
	if len(samples) < quorum {
		return ntpInterval{}, 0, errNTPInsufficientSources
	}
	type endpoint struct {
		value time.Duration
		start bool
	}
	points := make([]endpoint, 0, 2*len(samples))
	for _, sample := range samples {
		points = append(points, endpoint{sample.lower, true}, endpoint{sample.upper, false})
	}
	slices.SortFunc(points, func(a, b endpoint) int {
		switch {
		case a.value < b.value:
			return -1
		case a.value > b.value:
			return 1
		default:
			return 0
		}
	})
	var regions []ntpInterval
	agreeing := len(samples)
	add := func(lower, upper time.Duration, count int) {
		if count < quorum {
			return
		}
		agreeing = min(agreeing, count)
		if len(regions) != 0 && regions[len(regions)-1].upper == lower {
			regions[len(regions)-1].upper = upper
		} else {
			regions = append(regions, ntpInterval{lower, upper})
		}
	}
	count := 0
	for i := 0; i < len(points); {
		value := points[i].value
		ends := 0
		for i < len(points) && points[i].value == value {
			if points[i].start {
				count++
			} else {
				ends++
			}
			i++
		}
		// Two ranges that touch at one point count as agreeing at that point.
		add(value, value, count)
		count -= ends
		if i < len(points) {
			add(value, points[i].value, count)
		}
	}
	if len(regions) != 1 {
		return ntpInterval{}, 0, errNTPDisagreement
	}
	return regions[0], agreeing, nil
}
