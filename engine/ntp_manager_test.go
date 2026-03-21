package engine

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func stubQueryNTPOffset(t *testing.T, fn func(context.Context, []string) (time.Duration, error)) {
	t.Helper()
	original := queryNTPOffsetFunc
	queryNTPOffsetFunc = fn
	t.Cleanup(func() {
		queryNTPOffsetFunc = original
	})
}

func stubQueryNTPOffsetFromPool(t *testing.T, fn func(context.Context, *net.Dialer, string) (time.Duration, error)) {
	t.Helper()
	original := queryNTPOffsetFromPoolFunc
	queryNTPOffsetFromPoolFunc = fn
	t.Cleanup(func() {
		queryNTPOffsetFromPoolFunc = original
	})
}

func TestSetupNTPManager(t *testing.T) {
	_, err := setupNTPManager(nil, false)
	require.ErrorIs(t, err, errNilConfig, "setupNTPManager must return errNilConfig for a nil config")

	_, err = setupNTPManager(&config.NTPClientConfig{}, false)
	require.ErrorIs(t, err, errNilNTPConfigValues, "setupNTPManager must return errNilNTPConfigValues for incomplete config values")

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
	}
	m, err := setupNTPManager(cfg, false)
	require.NoError(t, err, "setupNTPManager must not error for a valid config")

	require.NotNil(t, m, "manager must not be nil for a valid config")
}

func TestNTPManagerIsRunning(t *testing.T) {
	var m *ntpManager
	assert.False(t, m.IsRunning(), "IsRunning should return false for a nil manager")

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err := setupNTPManager(cfg, false)
	require.NoError(t, err, "setupNTPManager must not error for a valid config")

	assert.False(t, m.IsRunning(), "IsRunning should return false before the manager starts")

	err = m.Start()
	require.NoError(t, err, "Start must not error for an enabled manager")
	assert.True(t, m.IsRunning(), "IsRunning should return true after the manager starts")
}

func TestNTPManagerStart(t *testing.T) {
	var m *ntpManager
	err := m.Start()
	require.ErrorIs(t, err, ErrNilSubsystem, "Start must return ErrNilSubsystem for a nil manager")

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Pool:                      []string{"ntp.invalid:123"},
	}
	m, err = setupNTPManager(cfg, true)
	require.NoError(t, err, "setupNTPManager must not error for a valid config")
	stubQueryNTPOffset(t, func(context.Context, []string) (time.Duration, error) {
		return 0, nil
	})

	err = m.Start()
	require.ErrorIs(t, err, errNTPManagerDisabled, "Start must return errNTPManagerDisabled when the manager level is disabled")

	m.level = 1
	err = m.Start()
	require.NoError(t, err, "Start must not error once the manager level is enabled")

	err = m.Start()
	require.ErrorIs(t, err, ErrSubSystemAlreadyStarted, "Start must return ErrSubSystemAlreadyStarted when called twice")
}

func TestNTPManagerStop(t *testing.T) {
	var m *ntpManager
	err := m.Stop()
	require.ErrorIs(t, err, ErrNilSubsystem, "Stop must return ErrNilSubsystem for a nil manager")

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err = setupNTPManager(cfg, true)
	require.NoError(t, err, "setupNTPManager must not error for a valid config")

	err = m.Stop()
	require.ErrorIs(t, err, ErrSubSystemNotStarted, "Stop must return ErrSubSystemNotStarted before the manager starts")

	err = m.Start()
	require.NoError(t, err, "Start must not error for an enabled manager")

	err = m.Stop()
	require.NoError(t, err, "Stop must not error after the manager starts")
}

func TestFetchNTPTime(t *testing.T) {
	var m *ntpManager
	_, err := m.FetchNTPTime()
	require.ErrorIs(t, err, ErrNilSubsystem, "FetchNTPTime must return ErrNilSubsystem for a nil manager")

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
		Pool:                      []string{"ntp.invalid:123"},
	}
	m, err = setupNTPManager(cfg, true)
	require.NoError(t, err, "setupNTPManager must not error for a valid config")
	stubQueryNTPOffset(t, func(context.Context, []string) (time.Duration, error) {
		return 2 * time.Second, nil
	})

	_, err = m.FetchNTPTime()
	require.ErrorIs(t, err, ErrSubSystemNotStarted, "FetchNTPTime must return ErrSubSystemNotStarted before the manager starts")

	err = m.Start()
	require.NoError(t, err, "Start must not error for an enabled manager")

	before := time.Now()
	tt, err := m.FetchNTPTime()
	require.NoError(t, err, "FetchNTPTime must not error after the manager starts")

	assert.WithinDuration(t, before.Add(2*time.Second), tt, 250*time.Millisecond, "FetchNTPTime should apply the mocked NTP offset")
}

func TestProcessTime(t *testing.T) {
	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
		Pool:                      []string{"ntp.invalid:123"},
	}
	m, err := setupNTPManager(cfg, true)
	require.NoError(t, err, "setupNTPManager must not error for a valid config")
	stubQueryNTPOffset(t, func(context.Context, []string) (time.Duration, error) {
		return 0, nil
	})

	err = m.processTime(context.Background())
	require.ErrorIs(t, err, ErrSubSystemNotStarted, "processTime must return ErrSubSystemNotStarted before the manager starts")

	err = m.Start()
	require.NoError(t, err, "Start must not error for an enabled manager")

	err = m.processTime(context.Background())
	require.NoError(t, err, "processTime must not error when the mocked offset is within threshold")

	m.allowedDifference = time.Duration(1)
	m.allowedNegativeDifference = time.Duration(1)
	stubQueryNTPOffset(t, func(context.Context, []string) (time.Duration, error) {
		return 2 * time.Second, nil
	})
	err = m.processTime(context.Background())
	require.NoError(t, err, "processTime must not error when the mocked offset is outside threshold")
}

func TestNTPTimestampToTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		seconds    uint32
		fractional uint32
		expected   time.Time
	}{
		{
			name:       "unix epoch",
			seconds:    ntpEpochOffset,
			fractional: 0,
			expected:   time.Unix(0, 0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, ntpTimestampToTime(tc.seconds, tc.fractional).Equal(tc.expected), "ntpTimestampToTime should convert NTP timestamps to the expected time")
		})
	}
}

func TestTimeToNTPTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "unix epoch",
			input:    time.Unix(0, 0),
			expected: time.Unix(0, 0),
		},
		{
			name:     "with nanoseconds",
			input:    time.Unix(123, 456000000),
			expected: time.Unix(123, 456000000),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sec, frac := timeToNTPTimestamp(tc.input)
			converted := ntpTimestampToTime(sec, frac)
			assert.WithinDuration(t, tc.expected, converted, time.Microsecond, "timeToNTPTimestamp should round trip with ntpTimestampToTime")
		})
	}
}

func TestCheckNTPOffset(t *testing.T) {
	wantErr := errors.New("boom")
	stubQueryNTPOffset(t, func(context.Context, []string) (time.Duration, error) {
		return 0, wantErr
	})

	_, err := checkNTPOffset(context.Background(), []string{"ntp.invalid:123"})
	require.ErrorIs(t, err, wantErr, "checkNTPOffset must return the mocked query error")
}

func TestQueryNTPOffsetNoValidServer(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	stubQueryNTPOffsetFromPool(t, func(context.Context, *net.Dialer, string) (time.Duration, error) {
		return 0, wantErr
	})

	_, err := queryNTPOffset(context.Background(), []string{"ntp.invalid:123"})
	require.Error(t, err, "queryNTPOffset must error when no pool can be reached")
	require.ErrorIs(t, err, errNoValidNTPServer, "queryNTPOffset must wrap errNoValidNTPServer when all pools fail")
	require.ErrorIs(t, err, wantErr, "queryNTPOffset must retain the last underlying pool error for debugging")
}

func TestCalculateNTPOffset(t *testing.T) {
	t.Parallel()

	origin := time.Unix(100, 0)
	receive := time.Unix(101, 0)
	transmit := time.Unix(101, 500000000)
	destination := time.Unix(102, 0)

	offset := calculateNTPOffset(origin, receive, transmit, destination)
	assert.Equal(t, 250*time.Millisecond, offset, "calculateNTPOffset should apply the RFC 5905 offset formula")
}

func TestValidateNTPResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		packet  *ntpPacket
		wantErr error
	}{
		{
			name:    "nil packet",
			packet:  nil,
			wantErr: errInvalidNTPResponse,
		},
		{
			name: "invalid mode",
			packet: &ntpPacket{
				Settings:  0x03,
				Stratum:   1,
				RxTimeSec: 1,
				TxTimeSec: 1,
			},
			wantErr: errInvalidNTPMode,
		},
		{
			name: "invalid leap indicator alarm",
			packet: &ntpPacket{
				Settings:  0xC4,
				Stratum:   1,
				RxTimeSec: 1,
				TxTimeSec: 1,
			},
			wantErr: errInvalidNTPResponse,
		},
		{
			name: "invalid stratum",
			packet: &ntpPacket{
				Settings:  0x04,
				Stratum:   0,
				RxTimeSec: 1,
				TxTimeSec: 1,
			},
			wantErr: errInvalidNTPStratum,
		},
		{
			name: "unsynchronised high stratum",
			packet: &ntpPacket{
				Settings:  0x04,
				Stratum:   16,
				RxTimeSec: 1,
				TxTimeSec: 1,
			},
			wantErr: errInvalidNTPStratum,
		},
		{
			name: "zero receive timestamp",
			packet: &ntpPacket{
				Settings:  0x04,
				Stratum:   1,
				TxTimeSec: 1,
			},
			wantErr: errZeroNTPReceiveTime,
		},
		{
			name: "zero transmit timestamp",
			packet: &ntpPacket{
				Settings:  0x04,
				Stratum:   1,
				RxTimeSec: 1,
			},
			wantErr: errZeroNTPTransmitTime,
		},
		{
			name: "valid packet",
			packet: &ntpPacket{
				Settings:  0x04,
				Stratum:   1,
				RxTimeSec: 1,
				TxTimeSec: 1,
			},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateNTPResponse(tc.packet)
			if tc.wantErr == nil {
				require.NoError(t, err, "validateNTPResponse must not error for a valid packet")
				return
			}
			require.ErrorIs(t, err, tc.wantErr, "validateNTPResponse must return the expected validation error")
		})
	}
}

func TestValidateNTPOriginateTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		packet     *ntpPacket
		seconds    uint32
		fractional uint32
		wantErr    error
	}{
		{
			name:    "nil packet",
			packet:  nil,
			wantErr: errInvalidNTPResponse,
		},
		{
			name: "mismatched originate timestamp",
			packet: &ntpPacket{
				OrigTimeSec:  2,
				OrigTimeFrac: 3,
			},
			seconds:    1,
			fractional: 2,
			wantErr:    errInvalidNTPOriginate,
		},
		{
			name: "matching originate timestamp",
			packet: &ntpPacket{
				OrigTimeSec:  1,
				OrigTimeFrac: 2,
			},
			seconds:    1,
			fractional: 2,
			wantErr:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateNTPOriginateTimestamp(tc.packet, tc.seconds, tc.fractional)
			if tc.wantErr == nil {
				require.NoError(t, err, "validateNTPOriginateTimestamp must not error for matching originate timestamps")
				return
			}
			require.ErrorIs(t, err, tc.wantErr, "validateNTPOriginateTimestamp must return the expected validation error")
		})
	}
}
