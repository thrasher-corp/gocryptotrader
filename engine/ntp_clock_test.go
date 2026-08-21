package engine

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClockMeasurerEmptyPools(t *testing.T) {
	calls := 0
	measurer := clockMeasurer{
		queryOne: func(context.Context, string) (clockObservation, error) {
			calls++
			return clockObservation{}, errors.New("unexpected query")
		},
	}

	_, err := measurer.measure(t.Context(), nil)
	require.ErrorIs(t, err, errNoNTPPoolsConfigured, "an empty pool list must return the dedicated sentinel")
	assert.NotErrorIs(t, err, errNoValidNTPServer, "an empty pool list should not report attempted servers")
	assert.Zero(t, calls, "an empty pool list should not query a source")
}

func TestClockMeasurerOrderedFallback(t *testing.T) {
	errFirst := errors.New("first source failed")
	want := clockObservation{
		source:          "second.test",
		clockCorrection: 12 * time.Millisecond,
		uncertainty:     4 * time.Millisecond,
		roundTrip:       3 * time.Millisecond,
	}
	var calls []string
	measurer := clockMeasurer{
		queryOne: func(_ context.Context, source string) (clockObservation, error) {
			calls = append(calls, source)
			if source == "first.test" {
				return clockObservation{}, errFirst
			}
			if source == "second.test" {
				return want, nil
			}
			return clockObservation{}, errors.New("later source must not be queried")
		},
	}

	got, err := measurer.measure(t.Context(), []string{"first.test", "second.test", "third.test"})
	require.NoError(t, err, "a valid fallback source must produce a measurement")
	assert.Equal(t, want, got, "measurement should return the first valid observation")
	assert.Equal(t, []string{"first.test", "second.test"}, calls, "measurement should visit sources in order and stop after success")
}

func TestClockMeasurerJoinsAllFailures(t *testing.T) {
	errFirst := errors.New("first failure")
	errSecond := errors.New("second failure")
	measurer := clockMeasurer{
		queryOne: func(_ context.Context, source string) (clockObservation, error) {
			switch source {
			case "first.test":
				return clockObservation{}, errFirst
			case "second.test":
				return clockObservation{}, errSecond
			default:
				return clockObservation{}, errors.New("unexpected source")
			}
		},
	}

	_, err := measurer.measure(t.Context(), []string{"first.test", "second.test"})
	require.ErrorIs(t, err, errNoValidNTPServer, "the aggregate must retain the no-valid-server sentinel")
	assert.ErrorIs(t, err, errFirst, "the aggregate should retain the first source failure")
	assert.ErrorIs(t, err, errSecond, "the aggregate should retain the second source failure")
	assert.Contains(t, err.Error(), "first.test", "the aggregate should identify the first source")
	assert.Contains(t, err.Error(), "second.test", "the aggregate should identify the second source")
}

func TestClockMeasurerContextStopsFailover(t *testing.T) {
	t.Run("canceled before first attempt", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		calls := 0
		measurer := clockMeasurer{
			queryOne: func(context.Context, string) (clockObservation, error) {
				calls++
				return clockObservation{}, errors.New("unexpected query")
			},
		}

		_, err := measurer.measure(ctx, []string{"first.test"})
		require.ErrorIs(t, err, context.Canceled, "measurement must preserve context cancellation")
		assert.NotErrorIs(t, err, errNoValidNTPServer, "cancellation should not collapse into a server aggregate")
		assert.Zero(t, calls, "a pre-canceled context should prevent the first query")
	})

	t.Run("canceled during an attempt", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		var calls []string
		measurer := clockMeasurer{
			queryOne: func(_ context.Context, source string) (clockObservation, error) {
				calls = append(calls, source)
				cancel()
				return clockObservation{}, errors.New("query interrupted")
			},
		}

		_, err := measurer.measure(ctx, []string{"first.test", "second.test"})
		require.ErrorIs(t, err, context.Canceled, "measurement must preserve cancellation after an attempt")
		assert.NotErrorIs(t, err, errNoValidNTPServer, "cancellation should not collapse into a server aggregate")
		assert.Equal(t, []string{"first.test"}, calls, "cancellation should prevent further attempts")
	})

	t.Run("successful result after cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		measurer := clockMeasurer{
			queryOne: func(context.Context, string) (clockObservation, error) {
				cancel()
				return clockObservation{
					clockCorrection: time.Second,
					uncertainty:     time.Millisecond,
				}, nil
			},
		}

		observation, err := measurer.measure(ctx, []string{"clock.test:123"})
		require.ErrorIs(t, err, context.Canceled, "measurement must discard a successful observation returned after cancellation")
		assert.Zero(t, observation, "measurement should not return the discarded observation")
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancel()
		calls := 0
		measurer := clockMeasurer{
			queryOne: func(context.Context, string) (clockObservation, error) {
				calls++
				return clockObservation{}, errors.New("unexpected query")
			},
		}

		_, err := measurer.measure(ctx, []string{"first.test"})
		require.ErrorIs(t, err, context.DeadlineExceeded, "measurement must preserve the deadline error")
		assert.NotErrorIs(t, err, errNoValidNTPServer, "a deadline should not collapse into a server aggregate")
		assert.Zero(t, calls, "an expired deadline should prevent the first query")
	})
}

func TestNTPTimeoutConstants(t *testing.T) {
	// Pin the user-visible bounds; these assertions do not prove transport wiring.
	assert.Equal(t, 5*time.Second, ntpDialTimeout, "the dial timeout should preserve the five-second bound")
	assert.Equal(t, 5*time.Second, ntpIOTimeout, "the connected I/O timeout should preserve the five-second bound")
}

func TestObservationFromResponse(t *testing.T) {
	tests := []struct {
		name       string
		correction time.Duration
	}{
		{name: "positive correction", correction: 23 * time.Millisecond},
		{name: "zero correction", correction: 0},
		{name: "negative correction", correction: -17 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &ntp.Response{
				ClockOffset:  tt.correction,
				RootDistance: 11 * time.Millisecond,
				RTT:          20 * time.Millisecond,
			}

			observation, err := observationFromResponse("clock.test:321", response)
			require.NoError(t, err, "a consistent response must produce an observation")
			assert.Equal(t, "clock.test:321", observation.source, "the observation should retain the configured source")
			assert.Equal(t, tt.correction, observation.clockCorrection, "clock correction should map without a sign change")
			assert.Equal(t, response.RootDistance, observation.uncertainty, "uncertainty should map root distance exactly once")
			assert.Equal(t, response.RTT, observation.roundTrip, "round trip should map without modification")
		})
	}
}

func TestObservationFromResponseRejectsInvalidMetadata(t *testing.T) {
	tests := []struct {
		name     string
		response *ntp.Response
		wantErr  error
	}{
		{
			name:     "negative round trip",
			response: &ntp.Response{RTT: -time.Nanosecond},
			wantErr:  errNegativeNTPRoundTrip,
		},
		{
			name:     "negative root distance",
			response: &ntp.Response{RootDistance: -time.Nanosecond},
			wantErr:  errNegativeNTPRootDistance,
		},
		{
			name:     "root distance below half round trip",
			response: &ntp.Response{RTT: 10 * time.Millisecond, RootDistance: 5*time.Millisecond - time.Nanosecond},
			wantErr:  errNTPRootDistanceBelowRTT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := observationFromResponse("clock.test", tt.response)
			require.ErrorIs(t, err, tt.wantErr, "invalid response metadata must return its dedicated sentinel")
		})
	}
}

func TestObservationFromResponseAcceptsBoundaryAndLargeDistance(t *testing.T) {
	tests := []struct {
		name         string
		roundTrip    time.Duration
		rootDistance time.Duration
	}{
		{
			name:         "half round trip boundary",
			roundTrip:    10 * time.Millisecond,
			rootDistance: 5 * time.Millisecond,
		},
		{
			name:         "large root distance",
			roundTrip:    10 * time.Millisecond,
			rootDistance: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observation, err := observationFromResponse("clock.test", &ntp.Response{
				RTT:          tt.roundTrip,
				RootDistance: tt.rootDistance,
			})
			require.NoError(t, err, "consistent response metadata must be accepted")
			assert.Equal(t, tt.rootDistance, observation.uncertainty, "the accepted observation should retain root distance")
		})
	}
}

func TestEvaluateClock(t *testing.T) {
	const (
		aheadTolerance  = 20 * time.Millisecond
		behindTolerance = 50 * time.Millisecond
	)
	tests := []struct {
		name       string
		correction time.Duration
		radius     time.Duration
		want       clockSyncState
	}{
		{name: "zero point", want: clockSyncInSync},
		{name: "whole interval on both bounds", correction: 15 * time.Millisecond, radius: 35 * time.Millisecond, want: clockSyncInSync},
		{name: "interval crosses both bounds", correction: 15 * time.Millisecond, radius: 45 * time.Millisecond, want: clockSyncInconclusive},
		{name: "interval crosses behind bound", correction: 45 * time.Millisecond, radius: 10 * time.Millisecond, want: clockSyncInconclusive},
		{name: "interval touches behind bound from outside", correction: 60 * time.Millisecond, radius: 10 * time.Millisecond, want: clockSyncInconclusive},
		{name: "interval entirely behind", correction: 61 * time.Millisecond, radius: 10 * time.Millisecond, want: clockSyncBehind},
		{name: "interval touches ahead bound from outside", correction: -30 * time.Millisecond, radius: 10 * time.Millisecond, want: clockSyncInconclusive},
		{name: "interval entirely ahead", correction: -31 * time.Millisecond, radius: 10 * time.Millisecond, want: clockSyncAhead},
		{name: "large correction remains behind", correction: 500 * time.Millisecond, radius: 100 * time.Millisecond, want: clockSyncBehind},
		{name: "large radius overlaps both bounds", radius: 100 * time.Millisecond, want: clockSyncInconclusive},
		{name: "exact behind point boundary", correction: behindTolerance, want: clockSyncInSync},
		{name: "one nanosecond beyond behind point boundary", correction: behindTolerance + time.Nanosecond, want: clockSyncBehind},
		{name: "exact ahead point boundary", correction: -aheadTolerance, want: clockSyncInSync},
		{name: "one nanosecond beyond ahead point boundary", correction: -aheadTolerance - time.Nanosecond, want: clockSyncAhead},
		{name: "positive endpoint does not wrap high", correction: time.Duration(math.MaxInt64), radius: time.Duration(math.MaxInt64), want: clockSyncInconclusive},
		{name: "negative endpoint does not wrap low", correction: time.Duration(math.MinInt64), radius: time.Duration(math.MaxInt64), want: clockSyncInconclusive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateClock(clockObservation{
				clockCorrection: tt.correction,
				uncertainty:     tt.radius,
			}, behindTolerance, aheadTolerance)
			assert.Equal(t, tt.want, got, "the interval classifier should return the expected state")
		})
	}
}

func TestQueryClockLoopbackNegativeCorrectionSign(t *testing.T) {
	endpoint, serverResult := startNTPTestServer(t, func(request []byte) ([]byte, error) {
		return makeNTPTestResponse(request, -2*time.Second, 1)
	})

	observation, err := queryClock(t.Context(), endpoint)
	require.NoError(t, err, "the real loopback query must accept a valid NTP response")
	require.NoError(t, <-serverResult, "the loopback NTP server must complete its exchange")
	assert.Equal(t, endpoint, observation.source, "the real query should retain the allocated endpoint")
	assert.Less(t, observation.clockCorrection, time.Duration(0), "a server behind local time should produce a negative correction")
}

func TestQueryClockRejectsMalformedLoopbackResponse(t *testing.T) {
	endpoint, serverResult := startNTPTestServer(t, func([]byte) ([]byte, error) {
		return []byte{1, 2, 3}, nil
	})

	_, err := queryClock(t.Context(), endpoint)
	require.Error(t, err, "a malformed NTP response must fail the real query boundary")
	require.NoError(t, <-serverResult, "the malformed-response server must complete its exchange")
}

func TestClockMeasurerRejectsValidationFailure(t *testing.T) {
	endpoint, serverResult := startNTPTestServer(t, func(request []byte) ([]byte, error) {
		return makeNTPTestResponse(request, 0, 0)
	})

	_, err := (clockMeasurer{queryOne: queryClock}).measure(t.Context(), []string{endpoint})
	require.Error(t, err, "a response rejected by Validate must make the source invalid")
	require.NoError(t, <-serverResult, "the invalid-response server must complete its exchange")
	assert.ErrorIs(t, err, errNoValidNTPServer, "a rejected response should contribute to the no-valid-server aggregate")
	assert.ErrorIs(t, err, ntp.ErrKissOfDeath, "the aggregate should retain the validation failure")
}

type ntpTestResponder func([]byte) ([]byte, error)

func startNTPTestServer(t *testing.T, responder ntpTestResponder) (endpoint string, serverResult <-chan error) {
	t.Helper()

	var listener net.ListenConfig
	connection, err := listener.ListenPacket(t.Context(), "udp4", "127.0.0.1:0")
	require.NoError(t, err, "the loopback UDP listener must start")
	t.Cleanup(func() {
		_ = connection.Close()
	})

	result := make(chan error, 1)
	go func() {
		if err := connection.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			result <- fmt.Errorf("set loopback server deadline: %w", err)
			return
		}

		request := make([]byte, 512)
		n, client, err := connection.ReadFrom(request)
		if err != nil {
			result <- fmt.Errorf("read loopback NTP request: %w", err)
			return
		}
		response, err := responder(request[:n])
		if err != nil {
			result <- err
			return
		}
		if _, err := connection.WriteTo(response, client); err != nil {
			result <- fmt.Errorf("write loopback NTP response: %w", err)
			return
		}
		result <- nil
	}()

	return connection.LocalAddr().String(), result
}

func makeNTPTestResponse(request []byte, serverOffset time.Duration, stratum uint8) ([]byte, error) {
	const ntpPacketLength = 48
	if len(request) < ntpPacketLength {
		return nil, fmt.Errorf("NTP request too short: %d", len(request))
	}

	serverTime := time.Now().Add(serverOffset)
	response := make([]byte, ntpPacketLength)
	response[0] = request[0]&0x38 | 4
	response[1] = stratum
	response[2] = request[2]
	response[3] = 0xec
	binary.BigEndian.PutUint32(response[4:8], 1311) // 20 ms in NTP short format, rounded.
	binary.BigEndian.PutUint32(response[8:12], 655) // 10 ms in NTP short format, rounded.
	copy(response[12:16], "LOCL")
	putNTPTime(response[16:24], serverTime.Add(-time.Second))
	copy(response[24:32], request[40:48])
	putNTPTime(response[32:40], serverTime)
	putNTPTime(response[40:48], serverTime)
	return response, nil
}

func putNTPTime(destination []byte, value time.Time) {
	const ntpEpochOffsetSeconds = 2_208_988_800
	seconds := uint64(value.Unix() + ntpEpochOffsetSeconds)      //nolint:gosec // Test timestamps are after the NTP epoch.
	fraction := uint64(value.Nanosecond()) << 32 / 1_000_000_000 //nolint:gosec // Nanosecond is non-negative and below one second.
	binary.BigEndian.PutUint64(destination, seconds<<32|fraction)
}
