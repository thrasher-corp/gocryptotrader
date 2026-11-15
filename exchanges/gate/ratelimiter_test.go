package gate

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimits(t *testing.T) {
	// Cases read the shared packageRateLimits, so they stay serial like the test
	for epl := request.EndpointLimit(1); epl <= otcOrderDetailEPL; epl++ {
		if epl == websocketRateLimitNotNeededEPL {
			continue
		}
		t.Run(strconv.FormatUint(uint64(epl), 10), func(t *testing.T) {
			assert.NotEmptyf(t, packageRateLimits[epl], "Empty rate limit not found for const %v", epl)
		})
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
		{"standard", standardRateLimit, 10 * time.Second, 200},
		{"personalAccount", personalAccountRateLimit, 10 * time.Second, 80},
		{"orderClose", orderCloseRateLimit, time.Second, 200},
		{"spotOrderPlacement", spotOrderPlacementRateLimit, time.Second, 10},
		{"otherPrivateEndpoint", otherPrivateEndpointRateLimit, 10 * time.Second, 150},
		{"perpetualOrderplacement", perpetualOrderplacementRateLimit, time.Second, 100},
		{"deliverySubmitCancelAmend", deliverySubmitCancelAmendRateLimit, 10 * time.Second, 500},
		{"optionsSubmitCancelAmend", optionsSubmitCancelAmendRateLimit, time.Second, 200},
		{"withdrawFromWallet", withdrawFromWalletRateLimit, 3 * time.Second, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, request.NewRateLimitWithWeight(tc.interval, tc.actions, 1), tc.limiter(), "rate limit should allow the documented requests per interval")
		})
	}
}

func TestHundredPerDayRateLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(24*time.Hour, 100, 1), hundredPerDayRateLimit(), "rate limit should allow 100 requests a day")
}

func TestHundredPer10SecondsRateLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(10*time.Second, 100, 1), hundredPer10SecondsRateLimit(), "rate limit should allow 100 requests every 10 seconds")
}

func TestHundredPer60SecondsRateLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(time.Minute, 100, 1), hundredPer60SecondsRateLimit(), "rate limit should allow 100 requests a minute")
}

func TestTenPer10SecondsRateLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(10*time.Second, 10, 1), tenPer10SecondsRateLimit(), "rate limit should allow 10 requests every 10 seconds")
}

func TestFifteenPer10SecondsRateLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(10*time.Second, 15, 1), fifteenPer10SecondsRateLimit(), "rate limit should allow 15 requests every 10 seconds")
}

func TestFivePerSecondRateLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(time.Second, 5, 1), fivePerSecondRateLimit(), "rate limit should allow 5 requests a second")
}

func TestUIDTransferRateLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, request.NewRateLimitWithWeight(10*time.Second, 1, 1), uidTransferRateLimit(), "rate limit should allow 1 request every 10 seconds")
}
