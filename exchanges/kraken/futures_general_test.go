package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetFuturesNotifications(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesNotifications(t.Context())
	require.NoError(t, err, "GetFuturesNotifications must not error")
	if mockTests {
		exp := &FuturesNotificationsResponse{
			Notifications: []FuturesNotification{
				{
					EffectiveTime:           time.Date(2026, 10, 14, 6, 0, 0, 0, time.UTC),
					Note:                    "Scheduled maintenance of the derivatives trading engine.",
					Priority:                "high",
					Type:                    "maintenance",
					ExpectedDowntimeMinutes: 30,
				},
				{
					EffectiveTime: time.Date(2026, 10, 30, 15, 0, 0, 0, time.UTC),
					Note:          "Month contracts with maturity 30/Oct/2026 expire and settle.",
					Priority:      "medium",
					Type:          "settlement",
				},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 34, 21, 500000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesNotifications should decode every field")
		return
	}
	assert.NotNil(t, result.Notifications, "GetFuturesNotifications should return the notifications")
}
