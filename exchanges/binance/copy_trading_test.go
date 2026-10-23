package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFuturesLeadTraderStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeadTraderStatus(t.Context())
	require.NoError(t, err, "GetFuturesLeadTraderStatus must not error")
	if mockTests {
		exp := &LeadTraderStatusResponse{
			Code:    "000000",
			Message: "success",
			Data:    LeadTraderStatus{IsLeadTrader: true, Time: types.Time(time.UnixMilli(1717382310843))},
			Success: true,
		}
		assert.Equal(t, exp, result, "GetFuturesLeadTraderStatus should decode every field")
		return
	}
	assert.True(t, result.Success, "GetFuturesLeadTraderStatus should succeed")
}

func TestGetFuturesLeadTradingSymbolWhitelist(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeadTradingSymbolWhitelist(t.Context())
	require.NoError(t, err, "GetFuturesLeadTradingSymbolWhitelist must not error")
	if mockTests {
		exp := &LeadTradingSymbolWhitelistResponse{
			Code:    "000000",
			Message: "success",
			Data: []LeadTradingSymbol{
				{Symbol: "BTCUSDT", BaseAsset: currency.BTC, QuoteAsset: currency.USDT},
				{Symbol: "ETHUSDT", BaseAsset: currency.ETH, QuoteAsset: currency.USDT},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesLeadTradingSymbolWhitelist should decode every field")
		return
	}
	assert.NotEmpty(t, result.Data, "GetFuturesLeadTradingSymbolWhitelist should return symbols")
}
