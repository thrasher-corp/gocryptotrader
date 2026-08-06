package engine

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func testNTPManager(t *testing.T, level int) *ntpManager {
	t.Helper()
	ahead := 20 * time.Millisecond
	behind := 50 * time.Millisecond
	m, err := setupNTPManager(&config.NTPClientConfig{
		Level:                     level,
		Pool:                      []string{"first.test:123", "second.test:123"},
		AllowedDifference:         &behind,
		AllowedNegativeDifference: &ahead,
	})
	require.NoError(t, err, "NTP manager setup must not error")
	return m
}

func registerNTPManagerCleanup(t *testing.T, m *ntpManager) {
	t.Helper()
	t.Cleanup(func() {
		if m.IsRunning() {
			require.NoError(t, m.Stop(), "started manager cleanup must stop")
		}
	})
}

func TestSetupNTPManager(t *testing.T) {
	_, err := setupNTPManager(nil)
	require.ErrorIs(t, err, errNilConfig, "nil config must return errNilConfig")

	_, err = setupNTPManager(&config.NTPClientConfig{})
	require.ErrorIs(t, err, errNilNTPConfigValues, "nil tolerances must return errNilNTPConfigValues")

	m := testNTPManager(t, config.NTPClientPeriodic)
	_, err = m.measure(t.Context(), nil)
	require.ErrorIs(t, err, errNoNTPPoolsConfigured, "setup measurement must retain the empty-pool sentinel")

	endpoint, serverResult := startNTPTestServer(t, func(request []byte) ([]byte, error) {
		return makeNTPTestResponse(request, time.Second, 1)
	})
	observation, err := m.measure(t.Context(), []string{endpoint})
	require.NoError(t, err, "setup measurement must use the production NTP query path")
	require.NoError(t, <-serverResult, "the setup measurement server must complete its exchange")
	assert.Greater(t, observation.clockCorrection, time.Duration(0), "setup measurement should retain the signed correction")
	assert.Equal(t, endpoint, observation.source, "setup measurement should retain the queried source")
	assert.Equal(t, 30*time.Minute, m.checkInterval, "manager cadence should be thirty minutes")
}

func TestSetupNTPClient(t *testing.T) {
	existing := &ntpManager{}
	disabled := &Engine{ntpManager: existing}
	disabledInput := &readTrackingReader{}
	require.NoError(
		t,
		disabled.setupNTPClient(t.Context(), disabledInput),
		"disabled NTP client setup must not error",
	)
	assert.Same(t, existing, disabled.ntpManager, "disabled NTP client should not replace the manager")
	assert.Zero(t, disabledInput.reads, "disabled NTP client should not prompt")

	setupFailure := &Engine{
		Config:     &config.Config{},
		Settings:   Settings{CoreSettings: CoreSettings{EnableNTPClient: true}},
		ntpManager: existing,
	}
	setupFailureInput := &readTrackingReader{}
	require.NoError(
		t,
		setupFailure.setupNTPClient(t.Context(), setupFailureInput),
		"NTP manager setup failure must not abort startup",
	)
	assert.Nil(t, setupFailure.ntpManager, "NTP manager setup failure should clear a stale manager")
	assert.Zero(t, setupFailureInput.reads, "NTP manager setup failure should not prompt")

	for _, level := range []int{config.NTPClientDisabled, config.NTPClientPeriodic} {
		ahead := 20 * time.Millisecond
		behind := 50 * time.Millisecond
		bot := &Engine{
			Config: &config.Config{NTPClient: config.NTPClientConfig{
				Level:                     level,
				Pool:                      []string{"first.test:123"},
				AllowedDifference:         &behind,
				AllowedNegativeDifference: &ahead,
			}},
			Settings: Settings{CoreSettings: CoreSettings{EnableNTPClient: true}},
		}
		input := &readTrackingReader{}
		require.NoError(
			t,
			bot.setupNTPClient(t.Context(), input),
			"non-startup NTP client setup must not error",
		)
		require.NotNil(t, bot.ntpManager, "enabled NTP client setup must construct the manager")
		assert.False(t, bot.ntpManager.IsRunning(), "normal engine setup should leave the manager stopped")
		assert.Zero(t, input.reads, "non-startup NTP client setup should not prompt")
	}
}

func TestNTPManagerLifecycle(t *testing.T) {
	var nilManager *ntpManager
	assert.False(t, nilManager.IsRunning(), "nil manager should not be running")
	require.ErrorIs(t, nilManager.Start(), ErrNilSubsystem, "nil manager Start must return ErrNilSubsystem")
	require.ErrorIs(t, nilManager.Stop(), ErrNilSubsystem, "nil manager Stop must return ErrNilSubsystem")

	for _, level := range []int{
		config.NTPClientStartup,
		config.NTPClientDisabled,
		42,
	} {
		m := testNTPManager(t, level)
		measurements := 0
		m.measure = func(context.Context, []string) (clockObservation, error) {
			measurements++
			return clockObservation{}, nil
		}
		err := m.Start()
		assert.ErrorIs(t, err, errNTPManagerDisabled, "non-periodic level should return errNTPManagerDisabled")
		assert.False(t, m.IsRunning(), "disabled manager should roll back its running state")
		assert.Zero(t, measurements, "Start should not perform a synchronous measurement")
	}

	m := testNTPManager(t, config.NTPClientPeriodic)
	measurements := 0
	m.measure = func(context.Context, []string) (clockObservation, error) {
		measurements++
		return clockObservation{}, nil
	}
	assert.False(t, m.IsRunning(), "new manager should not be running")
	err := m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted, "manager stopped before Start should return ErrSubSystemNotStarted")
	require.NoError(t, m.Start(), "periodic manager must start")
	registerNTPManagerCleanup(t, m)
	assert.True(t, m.IsRunning(), "started manager should be running")
	assert.Zero(t, measurements, "Start should defer measurement until the first ticker event")
	err = m.Start()
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted, "second start should return ErrSubSystemAlreadyStarted")

	require.NoError(t, m.Stop(), "running manager must stop")
	assert.False(t, m.IsRunning(), "stopped manager should not be running")
	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted, "second stop should return ErrSubSystemNotStarted")
}

func TestNTPProcessTime(t *testing.T) {
	stopped := testNTPManager(t, config.NTPClientPeriodic)
	err := stopped.processTime(t.Context())
	require.ErrorIs(t, err, ErrSubSystemNotStarted, "stopped processTime must return ErrSubSystemNotStarted")

	t.Run("success", func(t *testing.T) {
		m := testNTPManager(t, config.NTPClientPeriodic)
		measurements := 0
		m.measure = func(context.Context, []string) (clockObservation, error) {
			measurements++
			return clockObservation{
				source:          "first.test:123",
				clockCorrection: 5 * time.Millisecond,
				uncertainty:     10 * time.Millisecond,
				roundTrip:       12 * time.Millisecond,
			}, nil
		}
		require.NoError(t, m.Start(), "periodic manager must start before processing time")
		registerNTPManagerCleanup(t, m)
		require.NoError(t, m.processTime(t.Context()), "successful processTime must not error")
		assert.Equal(t, 1, measurements, "successful processTime should measure exactly once")
	})

	t.Run("measurement failure", func(t *testing.T) {
		errMeasurement := errors.New("measurement failed")
		m := testNTPManager(t, config.NTPClientPeriodic)
		m.measure = func(context.Context, []string) (clockObservation, error) {
			return clockObservation{}, errMeasurement
		}
		require.NoError(t, m.Start(), "periodic manager must start before processing time")
		registerNTPManagerCleanup(t, m)
		err := m.processTime(t.Context())
		require.ErrorIs(t, err, errMeasurement, "measurement error must retain its identity")
	})
}

func TestClockObservationReport(t *testing.T) {
	m := testNTPManager(t, config.NTPClientStartup)
	observation := clockObservation{
		source:          "first.test:123",
		clockCorrection: 45 * time.Millisecond,
		uncertainty:     10 * time.Millisecond,
		roundTrip:       12 * time.Millisecond,
	}
	message, warning := m.clockObservationReport(observation, clockSyncInconclusive)
	assert.True(t, warning, "Inconclusive should use warning severity")
	for _, expected := range []string{
		"state=inconclusive",
		"correction=+45ms",
		"positive means local behind",
		"root-distance=±10ms",
		"RTT=12ms",
		`source="first.test:123"`,
		"-20ms ahead",
		"+50ms behind",
		"no conclusion was possible",
		"closer or lower-latency NTP server",
	} {
		assert.Contains(t, message, expected, "Inconclusive report should contain required diagnostics")
	}

	_, warning = m.clockObservationReport(clockObservation{}, clockSyncInSync)
	assert.False(t, warning, "InSync should use debug severity")

	for _, tc := range []struct {
		state clockSyncState
		label string
	}{
		{state: clockSyncAhead, label: "state=ahead"},
		{state: clockSyncBehind, label: "state=behind"},
	} {
		message, warning = m.clockObservationReport(clockObservation{}, tc.state)
		assert.True(t, warning, "conclusive out-of-sync state should use warning severity")
		assert.Contains(t, message, tc.label, "conclusive report should identify the clock state")
	}
}

func TestHandleStartupNTPPolicyStates(t *testing.T) {
	for _, tc := range []struct {
		name        string
		observation clockObservation
		input       string
		wantLevel   int
	}{
		{
			name: "in sync",
			observation: clockObservation{
				clockCorrection: 0,
				uncertainty:     time.Millisecond,
			},
			wantLevel: config.NTPClientStartup,
		},
		{
			name: "inconclusive",
			observation: clockObservation{
				clockCorrection: 45 * time.Millisecond,
				uncertainty:     10 * time.Millisecond,
			},
			wantLevel: config.NTPClientStartup,
		},
		{
			name: "behind choose alert",
			observation: clockObservation{
				clockCorrection: 100 * time.Millisecond,
				uncertainty:     time.Millisecond,
			},
			input:     "a\n",
			wantLevel: config.NTPClientStartup,
		},
		{
			name: "behind choose periodic",
			observation: clockObservation{
				clockCorrection: 100 * time.Millisecond,
				uncertainty:     time.Millisecond,
			},
			input:     "w\n",
			wantLevel: config.NTPClientPeriodic,
		},
		{
			name: "ahead choose disabled",
			observation: clockObservation{
				clockCorrection: -100 * time.Millisecond,
				uncertainty:     time.Millisecond,
			},
			input:     "d\n",
			wantLevel: config.NTPClientDisabled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := testNTPManager(t, config.NTPClientStartup)
			measurements := 0
			m.measure = func(context.Context, []string) (clockObservation, error) {
				measurements++
				observation := tc.observation
				observation.source = "first.test:123"
				observation.roundTrip = time.Millisecond
				return observation, nil
			}
			bot := &Engine{
				Config:     &config.Config{NTPClient: config.NTPClientConfig{Level: config.NTPClientStartup}},
				ntpManager: m,
			}
			input := strings.NewReader(tc.input)
			err := bot.handleStartupNTPPolicy(t.Context(), iotest.OneByteReader(input), ntpStartupBudget)
			require.NoError(t, err, "startup NTP policy must not error")
			assert.Equal(t, 1, measurements, "startup policy should perform one aggregate measurement")
			assert.Equal(t, tc.wantLevel, bot.Config.NTPClient.Level, "config Level should match the selected policy")
			assert.Equal(t, tc.wantLevel, m.level, "manager Level should match the selected policy")
			if tc.input != "" {
				assert.Zero(t, input.Len(), "prompting startup policy should consume the selected response")
			}
		})
	}
}

func TestHandleStartupNTPPolicySkipsOtherLevels(t *testing.T) {
	for _, level := range []int{config.NTPClientDisabled, config.NTPClientPeriodic, 42} {
		m := testNTPManager(t, level)
		measurements := 0
		m.measure = func(context.Context, []string) (clockObservation, error) {
			measurements++
			return clockObservation{}, nil
		}
		bot := &Engine{Config: &config.Config{NTPClient: config.NTPClientConfig{Level: level}}, ntpManager: m}
		err := bot.handleStartupNTPPolicy(t.Context(), strings.NewReader(""), ntpStartupBudget)
		require.NoError(t, err, "non-startup NTP policy must not error")
		assert.Zero(t, measurements, "non-startup Level should not measure")
	}
}

func TestHandleStartupNTPPolicyFailures(t *testing.T) {
	errMeasurement := errors.New("measurement failed")

	t.Run("measurement failure", func(t *testing.T) {
		m := testNTPManager(t, config.NTPClientStartup)
		measurements := 0
		m.measure = func(context.Context, []string) (clockObservation, error) {
			measurements++
			return clockObservation{}, errMeasurement
		}
		bot := &Engine{Config: &config.Config{NTPClient: config.NTPClientConfig{Level: config.NTPClientStartup}}, ntpManager: m}
		input := &readTrackingReader{}
		err := bot.handleStartupNTPPolicy(t.Context(), input, ntpStartupBudget)
		require.NoError(t, err, "measurement failure must not abort startup")
		assert.Equal(t, 1, measurements, "measurement failure should not trigger an outer retry")
		assert.Zero(t, input.reads, "measurement failure should not prompt")
		assert.Equal(t, config.NTPClientStartup, bot.Config.NTPClient.Level, "measurement failure should preserve Level")
	})

	t.Run("parent canceled", func(t *testing.T) {
		parent, cancel := context.WithCancel(t.Context())
		cancel()
		m := testNTPManager(t, config.NTPClientStartup)
		m.measure = func(ctx context.Context, _ []string) (clockObservation, error) {
			return clockObservation{}, ctx.Err()
		}
		bot := &Engine{Config: &config.Config{NTPClient: config.NTPClientConfig{Level: config.NTPClientStartup}}, ntpManager: m}
		input := &readTrackingReader{}
		err := bot.handleStartupNTPPolicy(parent, input, ntpStartupBudget)
		require.ErrorIs(t, err, context.Canceled, "parent cancellation must abort startup")
		assert.Zero(t, input.reads, "parent cancellation should not prompt")
	})

	t.Run("parent deadline", func(t *testing.T) {
		parent, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
		defer cancel()
		m := testNTPManager(t, config.NTPClientStartup)
		m.measure = func(ctx context.Context, _ []string) (clockObservation, error) {
			return clockObservation{}, ctx.Err()
		}
		bot := &Engine{Config: &config.Config{NTPClient: config.NTPClientConfig{Level: config.NTPClientStartup}}, ntpManager: m}
		input := &readTrackingReader{}
		err := bot.handleStartupNTPPolicy(parent, input, ntpStartupBudget)
		require.ErrorIs(t, err, context.DeadlineExceeded, "parent deadline must abort startup")
		assert.Zero(t, input.reads, "parent deadline should not prompt")
	})

	t.Run("child budget", func(t *testing.T) {
		m := testNTPManager(t, config.NTPClientStartup)
		m.measure = func(ctx context.Context, pools []string) (clockObservation, error) {
			assert.Len(t, pools, 2, "configured pool list should reach the aggregate measurement")
			<-ctx.Done()
			return clockObservation{}, ctx.Err()
		}
		bot := &Engine{Config: &config.Config{NTPClient: config.NTPClientConfig{Level: config.NTPClientStartup}}, ntpManager: m}
		input := &readTrackingReader{}
		err := bot.handleStartupNTPPolicy(t.Context(), input, 0)
		require.NoError(t, err, "child budget expiry must not abort startup")
		assert.Zero(t, input.reads, "child budget expiry should not prompt")
	})

	t.Run("prompt EOF", func(t *testing.T) {
		m := testNTPManager(t, config.NTPClientStartup)
		m.measure = func(context.Context, []string) (clockObservation, error) {
			return clockObservation{
				clockCorrection: 100 * time.Millisecond,
				uncertainty:     time.Millisecond,
			}, nil
		}
		bot := &Engine{Config: &config.Config{NTPClient: config.NTPClientConfig{Level: config.NTPClientStartup}}, ntpManager: m}
		err := bot.handleStartupNTPPolicy(t.Context(), strings.NewReader(""), ntpStartupBudget)
		require.ErrorIs(t, err, io.EOF, "prompt EOF must abort startup and retain io.EOF")
	})
}

func TestHandleStartupNTPPolicyBudget(t *testing.T) {
	m := testNTPManager(t, config.NTPClientStartup)
	var observedDeadline time.Time
	var hasDeadline bool
	m.measure = func(ctx context.Context, _ []string) (clockObservation, error) {
		observedDeadline, hasDeadline = ctx.Deadline()
		return clockObservation{uncertainty: time.Millisecond}, nil
	}
	bot := &Engine{Config: &config.Config{NTPClient: config.NTPClientConfig{Level: config.NTPClientStartup}}, ntpManager: m}
	before := time.Now()
	require.NoError(
		t,
		bot.handleStartupNTPPolicy(t.Context(), strings.NewReader(""), ntpStartupBudget),
		"startup policy must use the bounded context",
	)
	after := time.Now()
	require.True(t, hasDeadline, "startup measurement context must have a deadline")
	assert.False(t, observedDeadline.Before(before.Add(ntpStartupBudget)), "startup deadline should not precede the requested budget")
	assert.False(t, observedDeadline.After(after.Add(ntpStartupBudget)), "startup deadline should not exceed the requested budget")
	assert.Equal(t, 10*time.Second, ntpStartupBudget, "startup budget should remain ten seconds")
}

type readTrackingReader struct {
	reads int
}

func (r *readTrackingReader) Read([]byte) (int, error) {
	r.reads++
	return 0, io.EOF
}
