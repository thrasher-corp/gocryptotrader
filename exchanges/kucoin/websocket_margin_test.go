package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// marginIndexPair is the pair the margin index and mark price fixtures record: the price of USDT in BTC
var marginIndexPair = currency.NewPairWithDelimiter("USDT", "BTC", "-")

func TestProcessMarginIndexPrice(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*WsMarginIndexPrice](t, nil, &subscription.Subscription{Channel: indexPriceIndicatorChannel, Asset: asset.Margin, Pairs: currency.Pairs{marginIndexPair}, QualifiedChannel: indexPriceIndicatorChannel + ":USDT-BTC"})
		assert.Equal(t, "USDT-BTC", got.Symbol, "the index price should name its symbol")
		assert.Positive(t, got.Value, "the index price should carry a value")
		return
	}
	exp := &WsMarginIndexPrice{
		Symbol:      "USDT-BTC",
		Granularity: 1000,
		Timestamp:   types.Time(time.UnixMilli(1791899505000)),
		Value:       0.000012004262954,
	}
	assert.Equal(t, exp, wsPush(t, wsTestInstance(t), "wsMargin.json", "indexPrice"), "an Index Price push should be relayed as it comes")
}

func TestProcessMarginMarkPrice(t *testing.T) {
	t.Parallel()
	if !mockTests {
		got := liveWsData[*WsMarginMarkPrice](t, nil, &subscription.Subscription{Channel: markPriceIndicatorChannel, Asset: asset.Margin, Pairs: currency.Pairs{marginIndexPair}, QualifiedChannel: markPriceIndicatorChannel + ":USDT-BTC"})
		assert.Equal(t, "USDT-BTC", got.Symbol, "the mark price should name its symbol")
		assert.Positive(t, got.Value, "the mark price should carry a value")
		return
	}
	exp := &WsMarginMarkPrice{
		Symbol:      "USDT-BTC",
		Granularity: 1000,
		Timestamp:   types.Time(time.UnixMilli(1791899505000)),
		Value:       0.000012004262954,
	}
	assert.Equal(t, exp, wsPush(t, wsTestInstance(t), "wsMargin.json", "markPrice"), "a Mark Price push should be relayed as it comes")
}

func TestProcessCrossMarginPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: marginPositionChannel, Asset: asset.Margin, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	exp := &WsCrossMarginPosition{
		DebtRatio:                   0.0157,
		TotalAsset:                  0.0005243177228408,
		MarginCoefficientTotalAsset: 0.0005243177228408,
		TotalDebt:                   0.00000823,
		AssetList: map[currency.Code]WsCrossMarginAsset{
			currency.BTC:  {Total: 0.00002, Available: 0.00001, Hold: 0.00001},
			currency.USDT: {Total: 33.68855864, Available: 15.01916691, Hold: 18.66939173},
		},
		DebtList:  map[currency.Code]types.Number{currency.BTC: 0.00000823, currency.USDT: 0.6},
		Timestamp: types.Time(time.UnixMilli(1729912435657)),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsMargin.json", "crossMarginDebtRatio"), "a debt.ratio push should be relayed as it comes")
	exp = &WsCrossMarginPosition{Timestamp: types.Time(time.UnixMilli(1729912438657)), Type: "FROZEN_FL"}
	assert.Equal(t, exp, wsPush(t, ex, "wsMargin.json", "crossMarginPositionStatus"), "a position.status push should be relayed as it comes")
	require.ErrorIs(t, ex.processCrossMarginPosition(t.Context(), &WsMessage{Topic: marginPositionChannel, Subject: "position.unknown"}), errUnhandledSubject, "processCrossMarginPosition must reject an unknown subject")
}

func TestProcessIsolatedMarginPosition(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		liveWsSubscribe(t, &subscription.Subscription{Channel: isolatedMarginPositionChannel, Asset: asset.Margin, Pairs: currency.Pairs{spotTestPair}, Authenticated: true})
		return
	}
	ex := wsTestInstance(t)
	exp := &WsIsolatedMarginPosition{
		Tag:                  "BTC-USDT",
		Status:               "DEBT",
		StatusBizType:        "DEFAULT_DEBT",
		AccumulatedPrincipal: 5.01,
		ChangeAssets: map[currency.Code]WsIsolatedMarginAsset{
			currency.BTC:  {Total: 0.00043478, Hold: 0.00001, LiabilityPrincipal: 0.00002, LiabilityInterest: 0.00000001},
			currency.USDT: {Total: 0.98092004, Hold: 0.1, LiabilityPrincipal: 26, LiabilityInterest: 0.00025644},
		},
		Timestamp: types.Time(time.UnixMilli(1730121097742)),
	}
	assert.Equal(t, exp, wsPush(t, ex, "wsMargin.json", "isolatedMarginPosition"), "an Isolated Margin Position push should be relayed as it comes")
	require.ErrorIs(t, ex.processIsolatedMarginPosition(t.Context(), &WsMessage{Topic: isolatedMarginPositionChannel + ":BTC-USDT", Subject: "debt.ratio"}), errUnhandledSubject, "processIsolatedMarginPosition must reject an unknown subject")
}
