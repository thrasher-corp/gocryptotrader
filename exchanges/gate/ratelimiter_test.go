package gate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimits(t *testing.T) {
	for epl := request.EndpointLimit(1); epl <= otcOrderDetailEPL; epl++ {
		if epl == websocketRateLimitNotNeededEPL {
			continue
		}
		assert.NotEmptyf(t, packageRateLimits[epl], "Empty rate limit not found for const %v", epl)
	}
}

func TestRateLimitHelpers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		limiter  func() *request.RateLimiterWithWeight
		interval time.Duration
		actions  int
	}{
		{"hundredPerDay", hundredPerDayRateLimit, 24 * time.Hour, 100},
		{"hundredPer10Seconds", hundredPer10SecondsRateLimit, 10 * time.Second, 100},
		{"hundredPer60Seconds", hundredPer60SecondsRateLimit, time.Minute, 100},
		{"standard", standardRateLimit, 10 * time.Second, 200},
		{"personalAccount", personalAccountRateLimit, 10 * time.Second, 80},
		{"orderClose", orderCloseRateLimit, time.Second, 200},
		{"tenPer10Seconds", tenPer10SecondsRateLimit, 10 * time.Second, 10},
		{"spotOrderPlacement", spotOrderPlacementRateLimit, time.Second, 10},
		{"fivePerSecond", fivePerSecondRateLimit, time.Second, 5},
		{"otherPrivateEndpoint", otherPrivateEndpointRateLimit, 10 * time.Second, 150},
		{"perpetualOrderplacement", perpetualOrderplacementRateLimit, time.Second, 100},
		{"deliverySubmitCancelAmend", deliverySubmitCancelAmendRateLimit, 10 * time.Second, 500},
		{"optionsSubmitCancelAmend", optionsSubmitCancelAmendRateLimit, time.Second, 200},
		{"withdrawFromWallet", withdrawFromWalletRateLimit, 3 * time.Second, 1},
		{"fifteenPer10Seconds", fifteenPer10SecondsRateLimit, 10 * time.Second, 15},
		{"uidTransfer", uidTransferRateLimit, 10 * time.Second, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, request.NewRateLimitWithWeight(tc.interval, tc.actions, 1), tc.limiter(), "rate limit should allow the documented requests per interval")
		})
	}
}
