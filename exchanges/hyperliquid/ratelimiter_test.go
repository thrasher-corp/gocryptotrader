package hyperliquid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimits(t *testing.T) {
	t.Parallel()
	limits := rateLimits
	// Info requests weigh 2, 20 or 60, and per-item requests one more per 20 items they can return
	for _, tc := range []struct {
		epl    request.EndpointLimit
		weight request.Weight
	}{
		{epl: infoStandardEPL, weight: 20},
		{epl: infoLightEPL, weight: 2},
		{epl: infoRecentTradesEPL, weight: 21},
		{epl: infoUserRoleEPL, weight: 60},
		{epl: infoHistoricalOrdersEPL, weight: 120},
		{epl: infoFundingHistoryEPL, weight: 45},
		{epl: infoUserLedgerEPL, weight: 120},
		{epl: infoUserFillsEPL, weight: 120},
		{epl: infoUserFundingEPL, weight: 45},
		{epl: infoStakingHistoryEPL, weight: 120},
		{epl: infoTWAPHistoryEPL, weight: 120},
	} {
		require.Containsf(t, limits, tc.epl, "rateLimits must define endpoint limit %d", tc.epl)
		assert.Equalf(t, tc.weight, limits[tc.epl].Weight(), "rateLimits should weigh endpoint limit %d", tc.epl)
	}
	// Candle snapshots weigh 20 plus one per 60 candles, up to the 5000 most recent candles Hyperliquid serves
	for extraWeight := request.Weight(1); extraWeight <= 84; extraWeight++ {
		epl := candleEPLBase + request.EndpointLimit(extraWeight)
		require.Containsf(t, limits, epl, "rateLimits must define candle endpoint limit %d", epl)
		assert.Equalf(t, 20+extraWeight, limits[epl].Weight(), "rateLimits should weigh candle endpoint limit %d", epl)
	}
	// Actions weigh 1 plus one per 40 batched items, up to the 1000 item batch cap
	for extraWeight := range request.Weight(26) {
		epl := exchangeActionEPLBase + request.EndpointLimit(extraWeight)
		require.Containsf(t, limits, epl, "rateLimits must define action endpoint limit %d", epl)
		assert.Equalf(t, 1+extraWeight, limits[epl].Weight(), "rateLimits should weigh action endpoint limit %d", epl)
	}
	assert.Len(t, limits, 11+84+26, "rateLimits should define only the info, candle and action weights")

	first, second := new(Exchange), new(Exchange)
	first.SetDefaults()
	second.SetDefaults()
	assert.True(t, first.Requester.GetRateLimiterDefinitions()[infoStandardEPL].SharesBudgetWith(second.Requester.GetRateLimiterDefinitions()[infoStandardEPL]), "Exchange instances should share the IP budget")
}

func TestCandleEndpointLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		count uint64
		exp   request.EndpointLimit
	}{
		{count: 0, exp: 1},
		{count: 1, exp: 1},
		{count: 60, exp: 1},
		{count: 61, exp: 2},
		{count: maximumCandleCount, exp: 84},
		{count: maximumCandleCount + 1, exp: 84},
	} {
		assert.Equalf(t, candleEPLBase+tc.exp, candleEndpointLimit(tc.count), "candleEndpointLimit should weigh %d candles", tc.count)
	}
}

func TestExchangeActionEndpointLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		batchLength uint64
		exp         request.EndpointLimit
	}{
		{batchLength: 0, exp: 0},
		{batchLength: 1, exp: 0},
		{batchLength: 39, exp: 0},
		{batchLength: 40, exp: 1},
		{batchLength: 79, exp: 1},
		{batchLength: maximumActionBatchSize, exp: 25},
		{batchLength: maximumActionBatchSize + 1, exp: 25},
	} {
		assert.Equalf(t, exchangeActionEPLBase+tc.exp, exchangeActionEndpointLimit(tc.batchLength), "exchangeActionEndpointLimit should weigh a batch of %d", tc.batchLength)
	}
}
