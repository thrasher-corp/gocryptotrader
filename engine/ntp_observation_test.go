package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectNTPInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		configured int
		samples    []ntpInterval
		want       ntpInterval
		votes      int
		err        error
	}{
		{name: "one of one", configured: 1, samples: []ntpInterval{{-10, 10}}, want: ntpInterval{-10, 10}, votes: 1},
		{name: "two of two", configured: 2, samples: []ntpInterval{{-10, 10}, {0, 20}}, want: ntpInterval{0, 10}, votes: 2},
		{name: "two of three", configured: 3, samples: []ntpInterval{{-10, 10}, {0, 20}}, want: ntpInterval{0, 10}, votes: 2},
		{name: "three of four", configured: 4, samples: []ntpInterval{{-10, 10}, {0, 20}, {5, 30}, {80, 90}}, want: ntpInterval{5, 10}, votes: 3},
		{name: "all identical", configured: 4, samples: []ntpInterval{{1, 2}, {1, 2}, {1, 2}, {1, 2}}, want: ntpInterval{1, 2}, votes: 4},
		{name: "missing cannot lower majority", configured: 4, samples: []ntpInterval{{0, 10}, {0, 10}}, err: errNTPInsufficientSources},
		{name: "no replies", configured: 1, err: errNTPInsufficientSources},
		{name: "split sources", configured: 4, samples: []ntpInterval{{-110, -90}, {-110, -90}, {90, 110}, {90, 110}}, err: errNTPDisagreement},
		{name: "disjoint majority regions", configured: 3, samples: []ntpInterval{{-100, 100}, {-90, -80}, {80, 90}}, err: errNTPDisagreement},
		{name: "touching endpoints", configured: 2, samples: []ntpInterval{{-10, 0}, {0, 10}}, want: ntpInterval{0, 0}, votes: 2},
		{name: "same endpoint different groups", configured: 3, samples: []ntpInterval{{-10, 0}, {0, 10}, {-5, 5}}, want: ntpInterval{-5, 5}, votes: 2},
		{name: "preserve full majority uncertainty", configured: 3, samples: []ntpInterval{{0, 20}, {10, 30}, {20, 40}}, want: ntpInterval{10, 30}, votes: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, votes, err := selectNTPInterval(tc.samples, tc.configured)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "selection should explain unavailable agreement")
				return
			}
			require.NoError(t, err, "majority interval must be available")
			assert.Equal(t, tc.want, got, "selection should retain the complete majority region")
			assert.Equal(t, tc.votes, votes, "reported votes should hold throughout the selected region")
		})
	}
}

func TestNTPIntervalClassification(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		interval ntpInterval
		want     ntpClockState
	}{
		{name: "within asymmetric bounds", interval: ntpInterval{-25, 50}, want: ntpHealthy},
		{name: "ahead", interval: ntpInterval{-100, -26}, want: ntpAhead},
		{name: "behind", interval: ntpInterval{51, 100}, want: ntpBehind},
		{name: "touch ahead boundary", interval: ntpInterval{-26, -25}, want: ntpUnknown},
		{name: "touch behind boundary", interval: ntpInterval{50, 51}, want: ntpUnknown},
		{name: "spans both bounds", interval: ntpInterval{-100, 100}, want: ntpUnknown},
		{name: "exact upper boundary", interval: ntpInterval{50, 50}, want: ntpHealthy},
		{name: "exact lower boundary", interval: ntpInterval{-25, -25}, want: ntpHealthy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.interval.classify(50, 25), "classification should use the whole interval and correct offset sign")
		})
	}
}

func TestNTPIntervalFaultAssumption(t *testing.T) {
	t.Parallel()
	// True correction is +80 ms.
	// Two broad correct intervals and one faulty narrow interval can agree on a region excluding that value.
	correct := ntpInterval{-20 * time.Millisecond, 180 * time.Millisecond}
	faulty := ntpInterval{-30 * time.Millisecond, 30 * time.Millisecond}
	interval, _, err := selectNTPInterval([]ntpInterval{correct, correct, faulty}, 4)
	require.NoError(t, err, "three overlapping replies must meet the configured quorum")
	assert.Equal(t, ntpInterval{-20 * time.Millisecond, 30 * time.Millisecond}, interval, "a faulty reply should demonstrate the limit of three-reply agreement")
	assert.Equal(t, ntpHealthy, interval.classify(50*time.Millisecond, 50*time.Millisecond), "classification should expose the possible false healthy verdict without a quorum of correct intervals")

	interval, _, err = selectNTPInterval([]ntpInterval{correct, correct, faulty, correct}, 4)
	require.NoError(t, err, "a quorum of correct replies must preserve their supported region")
	assert.Equal(t, correct, interval, "three correct intervals should retain the true correction despite one faulty reply")
	assert.Equal(t, ntpUnknown, interval.classify(50*time.Millisecond, 50*time.Millisecond), "the full supported interval should prevent a false healthy verdict")
}

func TestNTPIntervalAgreementOrder(t *testing.T) {
	t.Parallel()
	// Comparing with ntpd-rs showed its older selector could change its answer when touching ranges arrived in different order.
	// Both versions select a range where the largest number of server ranges overlap.
	// These cases check GCT keeps its result unchanged when input order changes.
	for _, tc := range []struct {
		name    string
		samples [3]ntpInterval
		want    ntpInterval
		err     error
	}{
		{name: "touching groups", samples: [3]ntpInterval{{-10, 0}, {0, 10}, {-5, 5}}, want: ntpInterval{-5, 5}},
		{name: "full uncertainty", samples: [3]ntpInterval{{0, 60}, {10, 70}, {20, 40}}, want: ntpInterval{10, 60}},
		{name: "competing majorities", samples: [3]ntpInterval{{-100, 100}, {-90, -80}, {80, 90}}, err: errNTPDisagreement},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
				got, votes, err := selectNTPInterval([]ntpInterval{tc.samples[order[0]], tc.samples[order[1]], tc.samples[order[2]]}, 3)
				if tc.err != nil {
					assert.ErrorIs(t, err, tc.err, "source order should not resolve competing explanations")
					continue
				}
				require.NoError(t, err, "connected majority must be available in every order")
				assert.Equal(t, tc.want, got, "source order should not narrow the supported uncertainty")
				assert.Equal(t, 2, votes, "two sources should support every point in the interval")
			}
		})
	}
}

func FuzzNTPIntervalAgreement(f *testing.F) {
	f.Add(int64(-100), int64(100), int64(-50), int64(50))
	f.Add(int64(0), int64(0), int64(0), int64(0))
	f.Add(int64(-9223372036854775808), int64(9223372036854775807), int64(-1), int64(1))
	f.Fuzz(func(t *testing.T, a, b, c, d int64) {
		first := ntpInterval{time.Duration(min(a, b)), time.Duration(max(a, b))}
		second := ntpInterval{time.Duration(min(c, d)), time.Duration(max(c, d))}
		got, votes, err := selectNTPInterval([]ntpInterval{first, second}, 2)
		lower, upper := max(first.lower, second.lower), min(first.upper, second.upper)
		if lower > upper {
			assert.ErrorIs(t, err, errNTPDisagreement, "disjoint sources should not agree")
			return
		}
		require.NoError(t, err, "overlapping sources must agree")
		assert.Equal(t, ntpInterval{lower, upper}, got, "two-source agreement should equal their intersection")
		assert.Equal(t, 2, votes, "both configured sources should be required")
	})
}

func TestNTPDefaultServerGeometry(t *testing.T) {
	t.Parallel()
	// Root distances in milliseconds, estimated from median ping times of three probes per city in September 2026.
	// These are zero-offset examples, not measured clock accuracy.
	// A correct clock should get a verdict in these cities, also when any one server does not answer.
	// Nairobi and Dubai have only Cloudflare nearby, so they need local servers.
	for city, tc := range map[string]struct {
		distances map[string]float64
		state     ntpClockState
	}{
		"Sydney":       {map[string]float64{"time.cloudflare.com": 2.2, "any.time.nl": 1.4, "time.nist.gov": 86.1, "ntp.nict.jp": 66.4}, ntpHealthy},
		"Auckland":     {map[string]float64{"time.cloudflare.com": 3.0, "any.time.nl": 13.9, "time.nist.gov": 98.0, "ntp.nict.jp": 125.8}, ntpHealthy},
		"Tokyo":        {map[string]float64{"time.cloudflare.com": 2.3, "any.time.nl": 1.4, "time.nist.gov": 65.2, "ntp.nict.jp": 1.1}, ntpHealthy},
		"Singapore":    {map[string]float64{"time.cloudflare.com": 2.4, "any.time.nl": 1.6, "time.nist.gov": 106.7, "ntp.nict.jp": 35.0}, ntpHealthy},
		"Seoul":        {map[string]float64{"time.cloudflare.com": 3.8, "any.time.nl": 59.8, "time.nist.gov": 90.7, "ntp.nict.jp": 19.9}, ntpHealthy},
		"Manila":       {map[string]float64{"time.cloudflare.com": 12.6, "any.time.nl": 93.8, "time.nist.gov": 107.1, "ntp.nict.jp": 38.9}, ntpHealthy},
		"Mumbai":       {map[string]float64{"time.cloudflare.com": 2.2, "any.time.nl": 1.4, "time.nist.gov": 130.5, "ntp.nict.jp": 69.3}, ntpHealthy},
		"Frankfurt":    {map[string]float64{"time.cloudflare.com": 2.3, "any.time.nl": 1.4, "time.nist.gov": 63.1, "ntp.nict.jp": 121.7}, ntpHealthy},
		"New York":     {map[string]float64{"time.cloudflare.com": 2.6, "any.time.nl": 1.7, "time.nist.gov": 20.8, "ntp.nict.jp": 82.3}, ntpHealthy},
		"Sao Paulo":    {map[string]float64{"time.cloudflare.com": 2.3, "any.time.nl": 1.5, "time.nist.gov": 76.1, "ntp.nict.jp": 138.0}, ntpHealthy},
		"Johannesburg": {map[string]float64{"time.cloudflare.com": 2.0, "any.time.nl": 1.1, "time.nist.gov": 136.3, "ntp.nict.jp": 197.3}, ntpHealthy},
		"Nairobi":      {map[string]float64{"time.cloudflare.com": 5.8, "any.time.nl": 86.3, "time.nist.gov": 139.1, "ntp.nict.jp": 154.5}, ntpUnknown},
		"Dubai":        {map[string]float64{"time.cloudflare.com": 6.0, "any.time.nl": 105.9, "time.nist.gov": 117.9, "ntp.nict.jp": 131.1}, ntpUnknown},
	} {
		hosts := make([]string, 0, len(defaultNTPServers))
		radii := make([]time.Duration, 0, len(defaultNTPServers))
		for _, server := range defaultNTPServers {
			host, _, err := parseNTPSource(server)
			require.NoError(t, err, "default server must parse")
			distance, ok := tc.distances[host]
			require.Truef(t, ok, "%s must have a measured distance for default server %s", city, host)
			hosts = append(hosts, host)
			radii = append(radii, time.Duration(distance*float64(time.Millisecond)))
		}
		for missing := -1; missing < len(radii); missing++ {
			if missing >= 0 && tc.state != ntpHealthy {
				break
			}
			answering := "all servers answering"
			if missing >= 0 {
				answering = hosts[missing] + " not answering"
			}
			var intervals []ntpInterval
			for i, radius := range radii {
				if i != missing {
					intervals = append(intervals, ntpInterval{-radius, radius})
				}
			}
			interval, _, err := selectNTPInterval(intervals, len(radii))
			require.NoErrorf(t, err, "%s with %s must reach agreement", city, answering)
			assert.Equalf(t, tc.state, interval.classify(50*time.Millisecond, 50*time.Millisecond), "%s with %s should give the expected verdict for a correct clock", city, answering)
		}
	}

	// The previous defaults had one nearby server outside Europe and North America, so distant servers formed the majority.
	interval, _, err := selectNTPInterval([]ntpInterval{
		{-2500 * time.Microsecond, 2500 * time.Microsecond},
		{-87 * time.Millisecond, 87 * time.Millisecond},
		{-152 * time.Millisecond, 152 * time.Millisecond},
		{-152 * time.Millisecond, 152 * time.Millisecond},
	}, 4)
	require.NoError(t, err, "previous defaults from Sydney must reach agreement")
	assert.Equal(t, ntpUnknown, interval.classify(50*time.Millisecond, 50*time.Millisecond), "previous defaults from Sydney should not verify a correct clock")
}
