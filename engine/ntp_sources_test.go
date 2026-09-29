package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNTPSourceNormalisation(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{" NTP.EXAMPLE.:00123 ", "ntp.example", "[::ffff:192.0.2.1]:123", "192.0.2.1", "[2001:db8::1]:124"})
	require.NoError(t, err, "valid sources must parse")
	require.Len(t, survey.sources, 3, "equivalent configured entries must collapse before fixing the denominator")
	assert.Equal(t, "ntp.example", survey.sources[0].host, "DNS name should be normalised")
	assert.Equal(t, "123", survey.sources[0].port, "port should be canonical")
	assert.Equal(t, "192.0.2.1", survey.sources[1].host, "mapped IPv4 should be normalised")
	assert.Equal(t, "124", survey.sources[2].port, "explicit alternate port should be preserved")
	for _, name := range []string{"", ":123", "host:0", "host:65536", "host:ntp", "[::1]:"} {
		_, err := newNTPSurvey([]string{name})
		assert.ErrorIsf(t, err, errNTPSource, "%q should be rejected", name)
	}
}

func TestNTPSurveyDuplicateAddresses(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"one.invalid", "two.invalid:124", "three.invalid", "four.invalid"})
	require.NoError(t, err, "four configured names must parse")
	survey.resolve = func(_ context.Context, _, host string) ([]netip.Addr, error) {
		switch host {
		case "one.invalid", "three.invalid":
			return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
		case "two.invalid":
			return []netip.Addr{netip.MustParseAddr("::ffff:192.0.2.1")}, nil
		default:
			return []netip.Addr{netip.MustParseAddr("192.0.2.2")}, nil
		}
	}
	var queried []string
	survey.query = func(_ context.Context, address string) (*ntp.Response, error) {
		queried = append(queried, address)
		return &ntp.Response{ClockOffset: time.Millisecond, RootDistance: time.Millisecond}, nil
	}
	result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, []string{"192.0.2.1:123", "192.0.2.2:123"}, queried, "aliases and alternate ports should not query the same IP twice")
	assert.Equal(t, 4, result.configured, "address collisions should not shrink the configured denominator")
	assert.Equal(t, 2, result.usable, "only distinct responding IPs should provide evidence")
	assert.Equal(t, ntpUnknown, result.state, "two votes out of four should not establish health")
	assert.ErrorIs(t, result.reason, errNTPInsufficientSources, "collisions should count as missing evidence")
	assert.ErrorIs(t, result.diagnostics, errNTPSourceAddress, "collisions should retain their diagnostic reason")
	assert.ErrorContains(t, result.diagnostics, "two.invalid", "collision diagnostic should identify the configured source")
	var history ntpReporting
	notices := history.update(&result, 0)
	require.Len(t, notices, 1, "duplicate endpoints must explain the missing quorum")
	assert.Equal(t, "GoCryptoTrader couldn't verify whether your computer's clock is accurate. 2/4 configured time servers provided usable time measurements. Some configured entries resolve to the same IP address. GoCryptoTrader will keep running.", notices[0].message, "duplicate entries should not be described as a network outage")
}

func TestNTPSurveyMissingSourceAndDNSRefresh(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"one.invalid", "two.invalid", "three.invalid", "four.invalid"})
	require.NoError(t, err, "configured names must parse")
	instant := time.Now()
	survey.now = func() time.Time { return instant }
	lookups := 0
	survey.resolve = func(_ context.Context, _, host string) ([]netip.Addr, error) {
		lookups++
		switch host {
		case "one.invalid":
			return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
		case "two.invalid":
			return []netip.Addr{netip.MustParseAddr("192.0.2.2")}, nil
		case "three.invalid":
			return []netip.Addr{netip.MustParseAddr("192.0.2.3")}, nil
		default:
			return nil, errors.New("test DNS failure")
		}
	}
	queries := 0
	survey.query = func(context.Context, string) (*ntp.Response, error) {
		queries++
		return &ntp.Response{ClockOffset: -100 * time.Millisecond, RootDistance: 10 * time.Millisecond}, nil
	}
	result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, ntpAhead, result.state, "three agreeing sources should establish a clock error despite one DNS failure")
	assert.Equal(t, ntpInterval{-110 * time.Millisecond, -90 * time.Millisecond}, result.interval, "selection should retain the measurement uncertainty")
	assert.Equal(t, 3, result.agreeing, "agreement should count usable distinct sources")
	assert.Equal(t, 4, result.configured, "DNS failure should not reduce the denominator")
	result = survey.observe(t.Context(), 15*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, ntpAhead, result.state, "cached endpoints should remain usable next round")
	assert.Equal(t, 5, lookups, "only the failed lookup should retry at the next round")
	assert.Equal(t, 6, queries, "each responding source should receive one request per round")
	result = survey.observe(t.Context(), time.Hour, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, ntpAhead, result.state, "hourly DNS refresh should preserve the verdict")
	assert.Equal(t, 9, lookups, "successful records should refresh hourly while failed lookups retry each round")
}

func TestNTPSurveyKissOfDeath(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"RATE", "DENY", "RSTR"} {
		t.Run(code, func(t *testing.T) {
			t.Parallel()
			survey, err := newNTPSurvey([]string{"192.0.2.1"})
			require.NoError(t, err, "source must parse")
			survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
			}
			queries := 0
			survey.query = func(context.Context, string) (*ntp.Response, error) {
				queries++
				return &ntp.Response{KissCode: code, Poll: 16 * time.Second}, ntp.ErrKissOfDeath
			}
			for _, elapsed := range []time.Duration{0, 15 * time.Minute, 30 * time.Minute, 45 * time.Minute, time.Hour, 90 * time.Minute} {
				result := survey.observe(t.Context(), elapsed, 50*time.Millisecond, 50*time.Millisecond)
				assert.Equal(t, ntpUnknown, result.state, "KoD should never provide clock evidence")
				assert.ErrorIs(t, result.reason, errNTPInsufficientSources, "suppressed endpoint should remain a missing vote")
				if code == "RATE" && elapsed == 15*time.Minute {
					assert.ErrorIs(t, result.diagnostics, errNTPSourceBackoff, "delayed retry should explain the remaining backoff")
				} else if code != "RATE" {
					assert.ErrorIs(t, result.diagnostics, ntp.ErrKissOfDeath, "refusal should remain visible after the original reply")
				}
			}
			if code == "RATE" {
				assert.Equal(t, 3, queries, "RATE should double the retry interval across successive refusals")
			} else {
				assert.Equal(t, 1, queries, "DENY and RSTR should suppress further queries for this manager lifetime")
			}
		})
	}
}

func TestNTPSurveyDNSRecovery(t *testing.T) {
	t.Parallel()
	dnsError := errors.New("test DNS unavailable")
	for _, failure := range []struct {
		name string
		err  error
	}{{"lookup error", dnsError}, {"empty answer", nil}} {
		t.Run(failure.name, func(t *testing.T) {
			t.Parallel()
			survey, err := newNTPSurvey([]string{"unavailable.invalid"})
			require.NoError(t, err, "source must parse")
			lookups := 0
			survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
				lookups++
				if lookups == 1 {
					return nil, failure.err
				}
				return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
			}
			var queried []string
			survey.query = func(_ context.Context, address string) (*ntp.Response, error) {
				queried = append(queried, address)
				return &ntp.Response{RootDistance: time.Millisecond}, nil
			}
			result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpUnknown, result.state, "failed resolution should leave accuracy unknown")
			assert.ErrorIs(t, result.reason, errNTPInsufficientSources, "failed resolution should leave a missing vote")
			if failure.err != nil {
				assert.ErrorIs(t, result.diagnostics, failure.err, "lookup failure should retain its diagnostic cause")
			} else {
				assert.ErrorIs(t, result.diagnostics, errNTPSourceAddress, "empty answer should explain the missing address")
			}
			assert.Empty(t, queried, "failed resolution should not send an NTP packet")
			var history ntpReporting
			notices := history.update(&result, 0)
			require.Len(t, notices, 1, "failed initial lookup must explain missing evidence")
			assert.Contains(t, notices[0].message, "DNS settings", "lookup failure without cached addresses should suggest checking DNS")
			assert.NotContains(t, notices[0].message, "time servers responded", "lookup failures should not claim a server was queried")
			result = survey.observe(t.Context(), 15*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpHealthy, result.state, "next scheduled round should recover after resolution succeeds")
			assert.NoError(t, result.diagnostics, "recovery should discard old diagnostics")
			assert.Equal(t, []string{"192.0.2.1:123"}, queried, "recovery should query the newly resolved endpoint")
			result = survey.observe(t.Context(), 30*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpHealthy, result.state, "successful cached answer should remain usable")
			assert.Equal(t, 2, lookups, "successful lookup should not be repeated within an hour")
			result = survey.observe(t.Context(), 75*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpHealthy, result.state, "hourly refresh should preserve verification")
			assert.Equal(t, 3, lookups, "successful answer should refresh one hour after resolution")
		})
	}
}

func TestNTPSurveyAddressRecovery(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"timeout", "invalid reply", "RATE", "DENY", "RSTR"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			survey, err := newNTPSurvey([]string{"pool.ntp.org:123"})
			require.NoError(t, err, "default source must parse")
			// Start with cached answer so regression does not depend on DNS shuffle.
			survey.sources[0].addresses = []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("192.0.2.2")}
			survey.sources[0].refreshAfter = time.Hour
			queryError := errors.New("test query timeout")
			if failure == "invalid reply" {
				queryError = ntp.ErrInvalidTime
			}
			var queried []string
			survey.query = func(_ context.Context, address string) (*ntp.Response, error) {
				queried = append(queried, address)
				if address == "192.0.2.1:123" {
					switch failure {
					case "RATE", "DENY", "RSTR":
						return &ntp.Response{KissCode: failure}, ntp.ErrKissOfDeath
					default:
						return nil, queryError
					}
				}
				return &ntp.Response{RootDistance: time.Millisecond}, nil
			}
			result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpUnknown, result.state, "first failed address should leave verification unknown")
			assert.Equal(t, []string{"192.0.2.1:123"}, queried, "a failed query should not trigger another packet in the same round")
			result = survey.observe(t.Context(), 15*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpHealthy, result.state, "next round should recover using the second cached address")
			assert.NoError(t, result.diagnostics, "successful replacement should discard the earlier failure")
			assert.Equal(t, []string{"192.0.2.1:123", "192.0.2.2:123"}, queried, "second round should query only the next address")
			result = survey.observe(t.Context(), 30*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpHealthy, result.state, "working address should remain preferred")
			assert.Equal(t, []string{"192.0.2.1:123", "192.0.2.2:123", "192.0.2.2:123"}, queried, "successful address should not rotate back to a failed member")
		})
	}
}

func TestNTPSurveyEligibleAddresses(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"one.invalid", "two.invalid:124", "three.invalid"})
	require.NoError(t, err, "sources must parse")
	denied := netip.MustParseAddr("192.0.2.1")
	delayed := netip.MustParseAddr("192.0.2.2")
	available := netip.MustParseAddr("192.0.2.3")
	alternate := netip.MustParseAddr("192.0.2.4")
	survey.endpoints[denied] = ntpEndpoint{denied: true}
	survey.endpoints[delayed] = ntpEndpoint{nextAllowed: time.Hour, backoff: time.Hour}
	survey.sources[0].addresses = []netip.Addr{denied, delayed, available}
	survey.sources[1].addresses = []netip.Addr{netip.MustParseAddr("::ffff:192.0.2.3"), alternate}
	survey.sources[2].addresses = []netip.Addr{denied, delayed}
	for i := range survey.sources {
		survey.sources[i].refreshAfter = time.Hour
	}
	var queried []string
	survey.query = func(_ context.Context, address string) (*ntp.Response, error) {
		queried = append(queried, address)
		return &ntp.Response{RootDistance: time.Millisecond}, nil
	}
	result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, ntpHealthy, result.state, "eligible distinct endpoints should establish the configured majority")
	assert.Equal(t, 3, result.configured, "skipped endpoints should not reduce the configured denominator")
	assert.Equal(t, 2, result.agreeing, "only distinct usable IPs should contribute votes")
	assert.Equal(t, []string{"192.0.2.3:123", "192.0.2.4:124"}, queried, "selection should skip refused, delayed and already used mapped IPs")
}

func TestNTPSurveyEndpointLifetime(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"changing.invalid"})
	require.NoError(t, err, "source must parse")
	address := netip.MustParseAddr("192.0.2.1")
	survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{address}, nil
	}
	var queried []string
	survey.query = func(_ context.Context, remote string) (*ntp.Response, error) {
		queried = append(queried, remote)
		return &ntp.Response{RootDistance: time.Millisecond}, nil
	}
	for hour := range 100 {
		result := survey.observe(t.Context(), time.Duration(hour)*time.Hour, 50*time.Millisecond, 50*time.Millisecond)
		require.Equal(t, ntpHealthy, result.state, "each replacement address must supply usable evidence")
		assert.Empty(t, survey.endpoints, "ordinary replies should not create endpoint records")
		address = address.Next()
	}
	require.Len(t, queried, 100, "hourly replacements must each receive one request")
	assert.Equal(t, "192.0.2.100:123", queried[99], "the last request should use the current DNS answer")
}

func TestNTPSurveyRestrictionsSurviveDNSReplacement(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"RATE", "DENY", "RSTR"} {
		t.Run(code, func(t *testing.T) {
			t.Parallel()
			survey, err := newNTPSurvey([]string{"changing.invalid"})
			require.NoError(t, err, "source must parse")
			address := netip.MustParseAddr("192.0.2.1")
			survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
				return []netip.Addr{address}, nil
			}
			var queried []string
			survey.query = func(_ context.Context, remote string) (*ntp.Response, error) {
				queried = append(queried, remote)
				if remote == "192.0.2.1:123" {
					return &ntp.Response{KissCode: code}, ntp.ErrKissOfDeath
				}
				return &ntp.Response{RootDistance: time.Millisecond}, nil
			}
			result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
			assert.ErrorIs(t, result.diagnostics, ntp.ErrKissOfDeath, "original endpoint should refuse the query")
			address = netip.MustParseAddr("192.0.2.2")
			result = survey.observe(t.Context(), time.Hour, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpHealthy, result.state, "a different endpoint should remain usable")
			address = netip.MustParseAddr("192.0.2.1")
			survey.observe(t.Context(), 2*time.Hour, 50*time.Millisecond, 50*time.Millisecond)
			result = survey.observe(t.Context(), 150*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpUnknown, result.state, "the original endpoint should remain restricted after returning in DNS")
			if code == "RATE" {
				assert.Equal(t, []string{"192.0.2.1:123", "192.0.2.2:123", "192.0.2.1:123"}, queried, "a second RATE should retain its history and double the delay to an hour")
				assert.ErrorIs(t, result.diagnostics, errNTPSourceBackoff, "second RATE should still prevent a query after thirty minutes")
			} else {
				assert.Equal(t, []string{"192.0.2.1:123", "192.0.2.2:123"}, queried, "a refused endpoint should never be queried again by this survey")
				assert.ErrorIs(t, result.diagnostics, ntp.ErrKissOfDeath, "refusal should remain the diagnostic cause")
			}
		})
	}
}

func TestNTPResponseInterval(t *testing.T) {
	t.Parallel()
	for _, response := range []*ntp.Response{
		nil,
		{RootDistance: -1},
		{ClockOffset: time.Duration(math.MaxInt64), RootDistance: 1},
		{ClockOffset: time.Duration(math.MinInt64), RootDistance: 1},
	} {
		_, err := ntpResponseInterval(response, 0)
		assert.ErrorIs(t, err, errNTPMeasurement, "invalid or overflowing response should not produce evidence")
	}
	interval, err := ntpResponseInterval(&ntp.Response{ClockOffset: 20 * time.Millisecond, RootDistance: 10 * time.Millisecond, RTT: 20 * time.Millisecond}, 0)
	require.NoError(t, err, "valid response must produce an interval")
	assert.Equal(t, ntpInterval{10 * time.Millisecond, 30 * time.Millisecond}, interval, "positive correction should mean the local clock is behind")
	interval, err = ntpResponseInterval(&ntp.Response{RootDistance: time.Millisecond}, 10*time.Second)
	require.NoError(t, err, "aged response must retain a conservative bound")
	assert.Equal(t, ntpInterval{-1150 * time.Microsecond, 1150 * time.Microsecond}, interval, "ten seconds of collection should add 150 microseconds at 15 ppm")
	_, err = ntpResponseInterval(&ntp.Response{RootDistance: time.Duration(math.MaxInt64)}, time.Second)
	assert.ErrorIs(t, err, errNTPMeasurement, "age growth should reject overflow")
	_, err = ntpResponseInterval(&ntp.Response{}, -time.Second)
	assert.ErrorIs(t, err, errNTPMeasurement, "negative measurement age should be rejected")
}

func TestNTPSurveyUncertainMeasurement(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"192.0.2.1"})
	require.NoError(t, err, "source must parse")
	instant := time.Now()
	survey.now = func() time.Time { return instant }
	survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
	}
	survey.query = func(context.Context, string) (*ntp.Response, error) {
		return &ntp.Response{ClockOffset: 50 * time.Millisecond, RootDistance: time.Millisecond}, nil
	}
	result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, ntpUnknown, result.state, "interval crossing tolerance should not establish clock health")
	assert.Equal(t, ntpInterval{49 * time.Millisecond, 51 * time.Millisecond}, result.interval, "unknown should retain the actual uncertainty interval")
	assert.ErrorIs(t, result.reason, errNTPUncertain, "uncertainty should be distinguishable from missing sources")
	assert.Equal(t, 1, result.usable, "valid uncertain response should remain usable evidence")
}

// TestNTPSourceAddresses adapts beevik/ntp TestOfflineFixHostPort inputs: https://github.com/beevik/ntp/blob/953b63646f5273d44de88b68e5862ad155ed4660/ntp_test.go#L55
// Copyright Brett Vickers. See THIRD_PARTY_NOTICES at the repository root.
func TestNTPSourceAddresses(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, host, port string
		invalid          bool
	}{
		{"192.168.1.1", "192.168.1.1", "123", false},
		{"192.168.1.1:123", "192.168.1.1", "123", false},
		{"192.168.1.1:1000", "192.168.1.1", "1000", false},
		{"[192.168.1.1]:1000", "192.168.1.1", "1000", false},
		{"www.example.com", "www.example.com", "123", false},
		{"www.example.com:123", "www.example.com", "123", false},
		{"www.example.com:1000", "www.example.com", "1000", false},
		{"[www.example.com]:1000", "www.example.com", "1000", false},
		{"::1", "::1", "123", false},
		{"[::1]", "::1", "123", false},
		{"[::1]:123", "::1", "123", false},
		{"[::1]:1000", "::1", "1000", false},
		{"fe80::1", "fe80::1", "123", false},
		{"[fe80::1]", "fe80::1", "123", false},
		{"[fe80::1]:123", "fe80::1", "123", false},
		{"[fe80::1]:1000", "fe80::1", "1000", false},
		{"[fe80::", "", "", true},
		{"[fe80::]@", "", "", true},
		{"ff06:0:0:0:0:0:0:c3", "ff06::c3", "123", false},
		{"[ff06:0:0:0:0:0:0:c3]", "ff06::c3", "123", false},
		{"[ff06:0:0:0:0:0:0:c3]:123", "ff06::c3", "123", false},
		{"[ff06:0:0:0:0:0:0:c3]:1000", "ff06::c3", "1000", false},
		{"::ffff:192.168.1.1", "192.168.1.1", "123", false},
		{"[::ffff:192.168.1.1]", "192.168.1.1", "123", false},
		{"[::ffff:192.168.1.1]:123", "192.168.1.1", "123", false},
		{"[::ffff:192.168.1.1]:1000", "192.168.1.1", "1000", false},
		{"", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			host, port, err := parseNTPSource(tc.name)
			if tc.invalid {
				assert.ErrorIs(t, err, errNTPSource, "invalid address should fail at the configuration boundary")
				return
			}
			require.NoError(t, err, "accepted beevik address form must parse")
			assert.Equal(t, tc.host, host, "host should preserve the address with canonical IP spelling")
			assert.Equal(t, tc.port, port, "port should be explicit")
		})
	}
}

func TestNTPSurveyRetainsLastKnownAddresses(t *testing.T) {
	t.Parallel()
	for _, failure := range []struct {
		name string
		err  error
	}{
		{"empty", nil},
		{"lookup error", errors.New("test DNS outage")},
		{"not found", &net.DNSError{Err: "no such host", IsNotFound: true}},
	} {
		t.Run(failure.name, func(t *testing.T) {
			t.Parallel()
			survey, err := newNTPSurvey([]string{"clock.invalid"})
			require.NoError(t, err, "source must parse")
			lookups := 0
			survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
				lookups++
				switch lookups {
				case 1:
					return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
				case 2:
					return nil, failure.err
				default:
					return []netip.Addr{netip.MustParseAddr("192.0.2.2")}, nil
				}
			}
			var queried []string
			survey.query = func(_ context.Context, address string) (*ntp.Response, error) {
				queried = append(queried, address)
				correction := time.Duration(0)
				if len(queried) == 2 {
					correction = 100 * time.Millisecond
				}
				return &ntp.Response{ClockOffset: correction, RootDistance: time.Millisecond}, nil
			}
			first := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
			require.Equal(t, ntpHealthy, first.state, "first reply must establish initial healthy evidence")
			second := survey.observe(t.Context(), time.Hour, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpBehind, second.state, "failed refresh should use a fresh measurement from the last-known address")
			assert.Zero(t, second.failures, "successful cached query should not inherit the failed refresh as missing evidence")
			assert.NoError(t, second.reason, "a failed refresh should not change a conclusive verdict")
			if failure.err == nil {
				assert.ErrorIs(t, second.diagnostics, errNTPSourceAddress, "empty refresh should remain in detailed diagnostics")
			} else {
				assert.ErrorIs(t, second.diagnostics, failure.err, "failed refresh should remain in detailed diagnostics")
			}
			assert.Equal(t, []string{"192.0.2.1:123", "192.0.2.1:123"}, queried, "failed refresh should retain the known address")
			third := survey.observe(t.Context(), 75*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
			assert.Equal(t, ntpHealthy, third.state, "next successful resolution should restore healthy evidence from the new address")
			assert.NoError(t, third.diagnostics, "successful new round should discard old diagnostics")
			assert.Equal(t, []string{"192.0.2.1:123", "192.0.2.1:123", "192.0.2.2:123"}, queried, "non-empty refresh should replace the old address")
			assert.Equal(t, 3, lookups, "unsuccessful refresh should retry on the next normal round")
		})
	}
}

func TestNTPSurveyMixedFailureMessages(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		replies []string
		want    string
	}{
		{"refused", []string{"DENY", "RSTR"}, "0/2 configured time servers provided usable time measurements. Some time servers refused further queries. Check your time server configuration."},
		{"rate and invalid", []string{"RATE", "invalid"}, "0/2 configured time servers provided usable time measurements. Some time servers requested fewer queries. GoCryptoTrader will wait before querying them again. Some time measurements were unusable."},
		{"refused and unreachable", []string{"DENY", "timeout"}, "0/2 configured time servers provided usable time measurements. Some time servers refused further queries. Check your time server configuration. Not enough time servers responded. Check your connection to the configured time servers."},
		{"unknown kiss code", []string{"XNEW"}, "0/1 configured time servers provided usable time measurements. Some time measurements were unusable."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			names := make([]string, len(tc.replies))
			for i := range names {
				names[i] = fmt.Sprintf("192.0.2.%d", i+1)
			}
			survey, err := newNTPSurvey(names)
			require.NoError(t, err, "sources must parse")
			survey.resolve = func(_ context.Context, _, host string) ([]netip.Addr, error) {
				return []netip.Addr{netip.MustParseAddr(host)}, nil
			}
			survey.query = func(_ context.Context, address string) (*ntp.Response, error) {
				for i, name := range names {
					if address != name+":123" {
						continue
					}
					switch tc.replies[i] {
					case "timeout":
						return nil, &net.OpError{Op: "read", Net: "udp", Err: context.DeadlineExceeded}
					case "invalid":
						return nil, ntp.ErrInvalidTime
					default:
						return &ntp.Response{KissCode: tc.replies[i]}, ntp.ErrKissOfDeath
					}
				}
				return nil, errors.New("test unexpected address")
			}
			observation := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
			var history ntpReporting
			notices := history.update(&observation, 0)
			require.Len(t, notices, 1, "unknown must produce one initial notice")
			assert.Equal(t, "GoCryptoTrader couldn't verify whether your computer's clock is accurate. "+tc.want+" GoCryptoTrader will keep running.", notices[0].message, "notice should state only the observed causes")
			notices = history.update(&observation, 15*time.Minute)
			require.Len(t, notices, 1, "prolonged unknown must produce one warning")
			assert.True(t, notices[0].warning, "prolonged unknown should warn")
			assert.Empty(t, history.update(&observation, 30*time.Minute), "repeated failures should not repeat the warning")
		})
	}
}

func TestNTPSurveyCachedAddressesRetainMixedRestrictions(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"healthy.invalid", "restricted.invalid"})
	require.NoError(t, err, "sources must parse")
	healthy := netip.MustParseAddr("192.0.2.1")
	denied := netip.MustParseAddr("192.0.2.2")
	delayed := netip.MustParseAddr("192.0.2.3")
	survey.sources[0].addresses = []netip.Addr{healthy}
	survey.sources[1].addresses = []netip.Addr{denied, delayed}
	survey.endpoints[denied] = ntpEndpoint{denied: true}
	survey.endpoints[delayed] = ntpEndpoint{nextAllowed: 2 * time.Hour, backoff: 2 * time.Hour}
	for i := range survey.sources {
		survey.sources[i].refreshAfter = time.Hour
	}
	lookupError := errors.New("test DNS refresh unavailable")
	survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) { return nil, lookupError }
	var queried []string
	survey.query = func(_ context.Context, address string) (*ntp.Response, error) {
		queried = append(queried, address)
		return &ntp.Response{RootDistance: time.Millisecond}, nil
	}
	result := survey.observe(t.Context(), time.Hour, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, []string{"192.0.2.1:123"}, queried, "cached fallback should honour every endpoint restriction")
	assert.ErrorIs(t, result.diagnostics, lookupError, "refresh failures should remain available for debugging")
	assert.ErrorIs(t, result.diagnostics, ntp.ErrKissOfDeath, "refusal should survive a failed refresh")
	assert.ErrorIs(t, result.diagnostics, errNTPSourceBackoff, "backoff should survive a failed refresh")
	var history ntpReporting
	notices := history.update(&result, time.Hour)
	require.Len(t, notices, 1, "missing evidence must produce one notice")
	assert.Equal(t, "GoCryptoTrader couldn't verify whether your computer's clock is accurate. 1/2 configured time servers provided usable time measurements. Some time servers refused further queries. Check your time server configuration. Some time servers requested fewer queries. GoCryptoTrader will wait before querying them again. GoCryptoTrader will keep running.", notices[0].message, "mixed candidate restrictions should remain visible without blaming DNS connectivity")
}

func TestNTPSurveyLookupMessages(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"not found", &net.DNSError{Err: "no such host", IsNotFound: true}, "Some configured time server names have no IP address. Check the server names, DNS settings and network connection."},
		{"empty", nil, "Some configured time server names have no IP address. Check the server names, DNS settings and network connection."},
		{"timeout", &net.DNSError{Err: "lookup timed out", IsTimeout: true}, "Some configured time server names could not be looked up. Check your network connection and DNS settings."},
		{"temporary", &net.DNSError{Err: "temporary resolver failure", IsTemporary: true}, "Some configured time server names could not be looked up. Check your network connection and DNS settings."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			survey, err := newNTPSurvey([]string{"clock.invalid"})
			require.NoError(t, err, "source must parse")
			survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) { return nil, tc.err }
			result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
			var history ntpReporting
			notices := history.update(&result, 0)
			require.Len(t, notices, 1, "unresolved source must explain missing evidence")
			assert.Equal(t, "GoCryptoTrader couldn't verify whether your computer's clock is accurate. 0/1 configured time servers provided usable time measurements. "+tc.want+" GoCryptoTrader will keep running.", notices[0].message, "lookup failures should not claim an NTP server was queried")
		})
	}
}

func TestNTPSurveyExpiredRate(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"clock.invalid"})
	require.NoError(t, err, "source must parse")
	survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
	}
	requests := 0
	survey.query = func(context.Context, string) (*ntp.Response, error) {
		requests++
		if requests == 1 {
			return &ntp.Response{KissCode: "RATE"}, ntp.ErrKissOfDeath
		}
		return &ntp.Response{RootDistance: time.Millisecond}, nil
	}
	first := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
	require.Equal(t, ntpUnknown, first.state, "RATE must leave clock accuracy unknown")
	second := survey.observe(t.Context(), 30*time.Minute, 50*time.Millisecond, 50*time.Millisecond)
	assert.Equal(t, ntpHealthy, second.state, "expired RATE delay should permit fresh healthy evidence")
	assert.Equal(t, 2, requests, "RATE delay should be the only wait between requests")
}

func TestNTPSurveyMixedLookupMessages(t *testing.T) {
	t.Parallel()
	survey, err := newNTPSurvey([]string{"missing.invalid", "unavailable.invalid"})
	require.NoError(t, err, "sources must parse")
	survey.resolve = func(_ context.Context, _, host string) ([]netip.Addr, error) {
		if host == "missing.invalid" {
			return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
		}
		return nil, &net.DNSError{Err: "lookup timed out", IsTimeout: true}
	}
	result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
	var history ntpReporting
	notices := history.update(&result, 0)
	require.Len(t, notices, 1, "mixed lookup failures must explain missing evidence")
	assert.Equal(t, "GoCryptoTrader couldn't verify whether your computer's clock is accurate. 0/2 configured time servers provided usable time measurements. Some configured time server names have no IP address. Check the server names, DNS settings and network connection. Some configured time server names could not be looked up. Check your network connection and DNS settings. GoCryptoTrader will keep running.", notices[0].message, "missing names and resolver failures should retain both explanations")
}

func TestNTPSurveyOfflineGoResolver(t *testing.T) {
	t.Parallel()
	networkError := errors.New("test network unavailable")
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, network, _ string) (net.Conn, error) {
			return nil, &net.OpError{Op: "dial", Net: network, Err: networkError}
		},
	}
	survey, err := newNTPSurvey([]string{"clock.invalid"})
	require.NoError(t, err, "source must parse")
	survey.resolve = resolver.LookupNetIP
	result := survey.observe(t.Context(), 0, 50*time.Millisecond, 50*time.Millisecond)
	dnsError, ok := errors.AsType[*net.DNSError](result.diagnostics)
	require.True(t, ok, "Go resolver failure must retain its DNS error type")
	assert.False(t, dnsError.IsNotFound, "offline Go lookup should not report a missing name")
	assert.Contains(t, dnsError.Err, "test network unavailable", "Go resolver should retain transport failure details")
	var history ntpReporting
	notices := history.update(&result, 0)
	require.Len(t, notices, 1, "offline lookup must explain missing evidence")
	assert.Equal(t, "GoCryptoTrader couldn't verify whether your computer's clock is accurate. 0/1 configured time servers provided usable time measurements. Some configured time server names could not be looked up. Check your network connection and DNS settings. GoCryptoTrader will keep running.", notices[0].message, "offline Go resolver should suggest checking network and DNS without claiming an NTP reply was expected")
}
