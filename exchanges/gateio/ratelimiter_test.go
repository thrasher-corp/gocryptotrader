package gateio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimits(t *testing.T) {
	for epl := request.EndpointLimit(1); epl <= perpetualUpdateRiskEPL; epl++ {
		if epl == websocketRateLimitNotNeededEPL {
			continue
		}
		assert.NotEmptyf(t, packageRateLimits[epl], "Empty rate limit not found for const %v", epl)
	}
}
