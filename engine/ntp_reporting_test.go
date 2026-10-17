package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNTPReportingEpisodes(t *testing.T) {
	t.Parallel()
	var history ntpReporting
	unknown := ntpObservation{reason: errNTPInsufficientSources, configured: 4, failures: ntpUnreachable}
	notices := history.update(&unknown, 0)
	require.Len(t, notices, 1, "initial inability to verify must be reported")
	assert.False(t, notices[0].warning, "initial unknown should be informational")
	assert.Contains(t, notices[0].message, "Not enough time servers responded", "notice should explain the actual failure")
	assert.Empty(t, history.update(&unknown, time.Minute), "early unknown should not escalate")
	notices = history.update(&unknown, 15*time.Minute)
	require.Len(t, notices, 1, "prolonged inability to verify must be reported once")
	assert.True(t, notices[0].warning, "prolonged unknown should warn")
	unknown.reason = errNTPDisagreement
	assert.Empty(t, history.update(&unknown, 30*time.Minute), "reason change should not open another unknown episode")
	bad := ntpObservation{state: ntpAhead, configured: 4, usable: 3, agreeing: 3, interval: ntpInterval{-120 * time.Millisecond, -80 * time.Millisecond}}
	notices = history.update(&bad, 45*time.Minute)
	require.Len(t, notices, 1, "first proven error must warn")
	assert.True(t, notices[0].warning, "proven error should warn")
	assert.Equal(t, "Your computer's clock appears to be ahead of the correct time, based on agreement from 3/4 configured time servers. Check your automatic time synchronisation settings. GoCryptoTrader will keep running.", notices[0].message, "warning should give the direction and configured denominator")
	assert.Empty(t, history.update(&bad, time.Hour), "reconfirmed error should not repeat the warning")
	notices = history.update(&unknown, 75*time.Minute)
	require.Len(t, notices, 1, "new inability to verify must be reported")
	assert.Equal(t, ntpAhead, history.last, "unknown should not imply recovery")
	assert.Empty(t, history.update(&bad, 90*time.Minute), "unknown between same-direction errors should not create another onset")
	bad.state = ntpBehind
	notices = history.update(&bad, 105*time.Minute)
	require.Len(t, notices, 1, "direction reversal must be reported")
	assert.Contains(t, notices[0].message, "behind the correct time", "reversal should report the new direction")
	healthy := ntpObservation{state: ntpHealthy, configured: 4, usable: 4, agreeing: 4}
	notices = history.update(&healthy, 2*time.Hour)
	require.Len(t, notices, 1, "fresh healthy evidence must report recovery")
	assert.False(t, notices[0].warning, "recovery should be informational")
	assert.Contains(t, notices[0].message, "within the configured tolerance", "recovery should explain the verified state")
	assert.Empty(t, history.update(&healthy, 135*time.Minute), "repeated healthy evidence should be quiet")
}

func TestNTPReportingSingleSource(t *testing.T) {
	t.Parallel()
	var history ntpReporting
	notices := history.update(&ntpObservation{state: ntpBehind, configured: 1, usable: 1, agreeing: 1}, 0)
	require.Len(t, notices, 1, "single-source error must be reported")
	assert.Contains(t, notices[0].message, "without independent corroboration", "single-source verdict should disclose its limitation")
}

func TestNTPReportingSingleSourceRecovery(t *testing.T) {
	t.Parallel()
	for _, previous := range []ntpClockState{ntpUnknown, ntpAhead, ntpBehind} {
		var history ntpReporting
		history.update(&ntpObservation{state: previous, configured: 1}, 0)
		healthy := ntpObservation{state: ntpHealthy, configured: 1, usable: 1, agreeing: 1}
		notices := history.update(&healthy, 15*time.Minute)
		require.Len(t, notices, 1, "fresh single-source evidence must report recovery")
		assert.False(t, notices[0].warning, "recovery should be informational")
		assert.Equal(t, "Your computer's clock appears to be within the configured tolerance, based on one configured time server, without independent corroboration.", notices[0].message, "recovery should disclose the single-source limitation")
		assert.Empty(t, history.update(&healthy, 30*time.Minute), "repeated healthy evidence should remain quiet")
	}
}

func TestNTPReportingImpreciseMeasurements(t *testing.T) {
	t.Parallel()
	var history ntpReporting
	notices := history.update(&ntpObservation{reason: errNTPUncertain, configured: 4, usable: 4, agreeing: 3}, 0)
	require.Len(t, notices, 1, "imprecise measurements must be reported")
	assert.Equal(t, "GoCryptoTrader couldn't verify whether your computer's clock is accurate. The measurements are not precise enough to judge the configured tolerance. Time servers closer to you, set in ntpclient.pool, give more precise measurements. GoCryptoTrader will keep running.", notices[0].message, "imprecision should point to closer time servers")
}
