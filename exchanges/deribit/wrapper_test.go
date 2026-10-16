package deribit

import (
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestGetAssetPairByInstrument(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo} {
		pairs, err := e.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		require.NotEmptyf(t, pairs, "the test config must hold %s pairs", a)
		for _, p := range pairs {
			instrument := formatPairString(a, p)
			extractedAsset, extractedPair, err := getAssetPairByInstrument(instrument)
			require.NoErrorf(t, err, "getAssetPairByInstrument must not error for %s", instrument)
			formatted, err := e.FormatExchangeCurrency(extractedPair, a)
			require.NoErrorf(t, err, "FormatExchangeCurrency must not error for %s", instrument)
			assert.Equalf(t, p.String(), formatted.String(), "getAssetPairByInstrument should return %s's pair", instrument)
			assert.Equalf(t, a, extractedAsset, "getAssetPairByInstrument should return %s's asset", instrument)
		}
	}
	_, _, err := getAssetPairByInstrument("")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "getAssetPairByInstrument should reject an empty instrument")
	_, _, err = getAssetPairByInstrument("thisIsAFakeCurrency")
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "getAssetPairByInstrument should reject an unknown format")
}

func TestGetAssetFromInstrument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		instrument string
		exp        asset.Item
		err        error
	}{
		{instrument: "BNB_USDC", exp: asset.Spot},
		{instrument: "BTC-30DEC22", exp: asset.Futures},
		{instrument: "BTCDVOL_USDC-1OCT25", exp: asset.Futures},
		{instrument: "ADA_USDC-PERPETUAL", exp: asset.Futures},
		{instrument: "PAXG_USDC-12SEP25-3320-P", exp: asset.Options},
		{instrument: "ETH-3OCT25-4800-P", exp: asset.Options},
		{instrument: "ETH-FS-26JUN26_26DEC25", exp: asset.FutureCombo},
		{instrument: "BTC-FS-28NOV25_PERP", exp: asset.FutureCombo},
		{instrument: "BTC-USDC-FS-28NOV25_PERP", exp: asset.FutureCombo},
		{instrument: "BTC_USDC-PBUT-31OCT25-90000_100000_102000", exp: asset.OptionCombo},
		{instrument: "BTC_USDC-CS-31OCT25-107000_111000", exp: asset.OptionCombo},
		{instrument: "BTC-ICOND-14NOV25-100000_105000_125000_130000", exp: asset.OptionCombo},
		{instrument: "BTC-PCAL-14NOV25_7NOV25-112000", exp: asset.OptionCombo},
		{instrument: "XRP_USDC-CBUT-26SEP25-2d9_3d2_3d4", exp: asset.OptionCombo},
		{instrument: "ETH-CS-26SEP25-5000_5500", exp: asset.OptionCombo},
		{instrument: "HELLOMOTO", err: errUnsupportedInstrumentFormat},
		{instrument: "hi-my-name-is-moto", err: errUnsupportedInstrumentFormat},
	} {
		a, err := getAssetFromInstrument(tc.instrument)
		require.ErrorIsf(t, err, tc.err, "getAssetFromInstrument must return the expected error for %s", tc.instrument)
		assert.Equalf(t, tc.exp, a, "getAssetFromInstrument should return %s's asset", tc.instrument)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		pair    string
		isMaker bool
		exp     float64
		err     error
	}{
		{pair: "BTC-PERPETUAL", isMaker: true},
		{pair: "BTC-PERPETUAL", exp: 0.5},
		{pair: "BTC-21OCT22", isMaker: true, exp: 0.1},
		{pair: "BTC-21OCT22", exp: 0.5},
		{pair: "SOL-21OCT22", isMaker: true, exp: 0.02},
		{pair: "ETH_USDC-PERPETUAL", isMaker: true},
		{pair: "ETH_USDC-PERPETUAL", exp: 0.5},
		{pair: "SOL-21OCT22-20-C", exp: 0.3},
		{pair: "SOL-21OCT22-20-C", isMaker: true},
		{pair: "BTC-21OCT22-20000-P", isMaker: true, exp: 0.3},
		{pair: "BTC-STRG-21OCT22-19000_21000", exp: 0.3},
		{pair: "BTC-FS-30DEC22_28OCT22", exp: 0.5},
		{pair: "BTC_USDC", err: asset.ErrNotSupported},
		{pair: "some_instrument_builder", err: asset.ErrNotSupported},
	} {
		p, err := currency.NewPairFromString(tc.pair)
		require.NoErrorf(t, err, "NewPairFromString must not error for %s", tc.pair)
		fee, err := calculateTradingFee(&exchange.FeeBuilder{Pair: p, IsMaker: tc.isMaker, Amount: 1, PurchasePrice: 1000})
		require.ErrorIsf(t, err, tc.err, "calculateTradingFee must return the expected error for %s", tc.pair)
		assert.InDeltaf(t, tc.exp, fee, 1e-12, "calculateTradingFee should return %s's fee with maker %t", tc.pair, tc.isMaker)
	}
}

func TestGetOfflineTradeFee(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 0.6, getOfflineTradeFee(1000, 2), 1e-12, "getOfflineTradeFee should charge the worst case rate of 0.03 percent")
}

func TestFormatFuturesTradablePair(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode(perpString)}:                    "BTC-PERPETUAL",
		{Delimiter: currency.DashDelimiter, Base: currency.AVAX, Quote: currency.NewCode("USDC-PERPETUAL")}:             "AVAX_USDC-PERPETUAL",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("30DEC22")}:                     "ETH-30DEC22",
		{Delimiter: currency.DashDelimiter, Base: currency.SOL, Quote: currency.NewCode("30DEC22")}:                     "SOL-30DEC22",
		{Delimiter: currency.DashDelimiter, Base: currency.NewCode("BTCDVOL"), Quote: currency.NewCode("USDC-28JUN23")}: "BTCDVOL_USDC-28JUN23",
	} {
		assert.Equalf(t, exp, formatFuturesTradablePair(pair), "formatFuturesTradablePair should format %s", exp)
	}
}

func TestOptionPairToString(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("30MAY24-61000-C")}:      "BTC-30MAY24-61000-C",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("1JUN24-3200-P")}:        "ETH-1JUN24-3200-P",
		{Delimiter: currency.DashDelimiter, Base: currency.SOL, Quote: currency.NewCode("USDC-31MAY24-162-P")}:   "SOL_USDC-31MAY24-162-P",
		{Delimiter: currency.DashDelimiter, Base: currency.MATIC, Quote: currency.NewCode("USDC-6APR24-0d98-P")}: "MATIC_USDC-6APR24-0d98-P",
		{Delimiter: currency.DashDelimiter, Base: currency.MATIC, Quote: currency.NewCode("USDC-8JUN24-0D99-P")}: "MATIC_USDC-8JUN24-0d99-P",
		{Delimiter: currency.DashDelimiter, Base: currency.MATIC, Quote: currency.NewCode("USDC-6DEC29-0D87-C")}: "MATIC_USDC-6DEC29-0d87-C",
	} {
		assert.Equalf(t, exp, optionPairToString(pair), "optionPairToString should format %s", exp)
	}
}

func TestOptionComboPairToString(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.NewCode("XRP"), Quote: currency.NewCode("USDC-CS-26SEP25-3D3_3D5")}:    "XRP_USDC-CS-26SEP25-3d3_3d5",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDC-PBUT-31OCT25-90000_100000_102000")}: "BTC_USDC-PBUT-31OCT25-90000_100000_102000",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("CS-26SEP25-5000_5500")}:                  "ETH-CS-26SEP25-5000_5500",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("PCAL-14NOV25_7NOV25-112000")}:            "BTC-PCAL-14NOV25_7NOV25-112000",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("STRG-21OCT22")}:                          "BTC-STRG-21OCT22",
	} {
		assert.Equalf(t, exp, optionComboPairToString(pair), "optionComboPairToString should format %s", exp)
	}
}

func TestFutureComboPairToString(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]string{
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("FS-28NOV25_PERP")}:      "BTC-FS-28NOV25_PERP",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("FS-28NOV25_PERP")}:      "ETH-FS-28NOV25_PERP",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("FS-30JAN26_26DEC25")}:   "BTC-FS-30JAN26_26DEC25",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDC-FS-28NOV25_PERP")}: "BTC_USDC-FS-28NOV25_PERP",
		{Delimiter: currency.DashDelimiter, Base: currency.ETH, Quote: currency.NewCode("USDC-FS-28NOV25_PERP")}: "ETH_USDC-FS-28NOV25_PERP",
		// A pair without a dash, and one with the USDC prefix but no dash after it, are left as they are
		{Base: currency.BTC, Quote: currency.USDT}:                                                         "BTCUSDT",
		{Delimiter: currency.DashDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDC-PERPETUAL")}: "BTC-USDC-PERPETUAL",
	} {
		assert.Equalf(t, exp, futureComboPairToString(pair), "futureComboPairToString should format %s", exp)
	}
}

func TestFormatPairString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{a: asset.Spot, pair: currency.NewPair(currency.BTC, currency.USDC), exp: "BTC_USDC"},
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode("USDC-PERPETUAL")), exp: "BTC_USDC-PERPETUAL"},
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode(perpString)), exp: "BTC-PERPETUAL"},
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode("USDC-27MAR26")), exp: "BTC_USDC-27MAR26"},
		{a: asset.Options, pair: currency.NewPair(currency.NewCode("PAXG"), currency.NewCode("USDC-30MAY25")), exp: "PAXG_USDC-30MAY25"},
		{a: asset.Options, pair: currency.NewPair(currency.BTC, currency.NewCode("14JUN24-62000-C")), exp: "BTC-14JUN24-62000-C"},
		{a: asset.OptionCombo, pair: currency.NewPair(currency.BTC, currency.NewCode("USDC-PCOND-7MAR26-65500_66000_70000_70500")), exp: "BTC_USDC-PCOND-7MAR26-65500_66000_70000_70500"},
		{a: asset.FutureCombo, pair: currency.NewPair(currency.BTC, currency.NewCode("USDC-FS-27MAR26_PERP")), exp: "BTC_USDC-FS-27MAR26_PERP"},
	} {
		assert.Equalf(t, tc.exp, formatPairString(tc.a, tc.pair), "formatPairString should format %s %s", tc.a, tc.exp)
	}
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		timeInForce string
		postOnly    bool
		exp         order.TimeInForce
		err         error
	}{
		{timeInForce: "good_til_cancelled", exp: order.GoodTillCancel},
		{timeInForce: "good_til_cancelled", postOnly: true, exp: order.GoodTillCancel | order.PostOnly},
		{timeInForce: "good_til_day", exp: order.GoodTillDay},
		{timeInForce: "good_til_day", postOnly: true, exp: order.GoodTillDay | order.PostOnly},
		{timeInForce: "fill_or_kill", exp: order.FillOrKill},
		{timeInForce: "immediate_or_cancel", exp: order.ImmediateOrCancel},
		{timeInForce: "abcd", exp: order.UnknownTIF, err: order.ErrInvalidTimeInForce},
		{exp: order.UnknownTIF},
	} {
		tif, err := timeInForceFromString(tc.timeInForce, tc.postOnly)
		require.ErrorIsf(t, err, tc.err, "timeInForceFromString must return the expected error for %q", tc.timeInForce)
		assert.Equalf(t, tc.exp, tif, "timeInForceFromString should convert %q with post only %t", tc.timeInForce, tc.postOnly)
	}
}

func TestGetResolutionFromInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		exp      string
		err      error
	}{
		{interval: kline.HundredMilliseconds, exp: "100ms"},
		{interval: kline.OneMin, exp: "1"},
		{interval: kline.ThreeMin, exp: "3"},
		{interval: kline.FiveMin, exp: "5"},
		{interval: kline.TenMin, exp: "10"},
		{interval: kline.FifteenMin, exp: "15"},
		{interval: kline.ThirtyMin, exp: "30"},
		{interval: kline.OneHour, exp: "60"},
		{interval: kline.TwoHour, exp: "120"},
		{interval: kline.ThreeHour, exp: "180"},
		{interval: kline.SixHour, exp: "360"},
		{interval: kline.TwelveHour, exp: "720"},
		{interval: kline.OneDay, exp: "1D"},
		{interval: kline.Raw, exp: "raw"},
		{interval: kline.FourHour, err: kline.ErrUnsupportedInterval},
	} {
		resolution, err := e.GetResolutionFromInterval(tc.interval)
		require.ErrorIsf(t, err, tc.err, "GetResolutionFromInterval must return the expected error for %s", tc.interval)
		assert.Equalf(t, tc.exp, resolution, "GetResolutionFromInterval should convert %s", tc.interval)
	}
}

// TestQuoteVolume pins which instruments have a quote volume to report. get_instruments gives
// quote_currency USD for a future, BTC for a BTC option and USDC for a USDC one, and Deribit serves
// volume_notional only where that currency is the one volume_usd is denominated in
func TestQuoteVolume(t *testing.T) {
	t.Parallel()
	const volumeUSD = 4.7
	for _, a := range []asset.Item{asset.Options, asset.OptionCombo} {
		// a BTC quoted option: Deribit serves no volume_notional, its premium turnover in BTC
		assert.Zerof(t, quoteVolume(volumeUSD, 0, a), "%s quoted in the base currency should report no turnover, having none in it", a)
		// a USDC quoted one, where volume_notional is that turnover
		assert.Equalf(t, 23.25, quoteVolume(23.25, 23.25, a), "%s quoted in USDC should report volume_notional", a)
	}
	for _, a := range []asset.Item{asset.Futures, asset.FutureCombo, asset.Spot} {
		assert.Equalf(t, volumeUSD, quoteVolume(volumeUSD, 0, a), "%s should fall back to volume_usd where no volume_notional is served", a)
		// A USDC quoted instrument, where volume_usd differs by the USDC price
		assert.Equalf(t, 27088304.60512001, quoteVolume(27085692.54, 27088304.60512001, a),
			"%s quoted in USDC should report volume_notional rather than volume_usd", a)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  bool
		err  error
	}{
		{a: asset.Spot, err: currency.ErrCurrencyPairEmpty},
		{a: asset.Spot, pair: currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)},
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode(perpString)), exp: true},
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode("USDC-PERPETUAL")), exp: true},
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode("27MAR26"))},
		{a: asset.FutureCombo, pair: currency.NewPair(currency.BTC, currency.NewCode("FS-27SEP24_PERP"))},
		{a: asset.OptionCombo, pair: currency.NewPair(currency.BTC, currency.NewCode("STRG-21OCT22"))},
	} {
		isPerpetual, err := e.IsPerpetualFutureCurrency(tc.a, tc.pair)
		require.ErrorIsf(t, err, tc.err, "IsPerpetualFutureCurrency must return the expected error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, isPerpetual, "IsPerpetualFutureCurrency should report %s %s", tc.a, tc.pair)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must reject an empty pair")
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "GetPairs must not error for %s", a)
		require.NotEmptyf(t, pairs, "the test config must hold %s pairs", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		assert.NotEmptyf(t, resp, "GetCurrencyTradeURL should return a URL for %s", a)
	}
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode("USDC-PERPETUAL")), exp: tradeBaseURL + tradeFutures + "BTC_USDC-PERPETUAL"},
		{a: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode(perpString)), exp: tradeBaseURL + tradeFutures + "BTC_PERPETUAL"},
		{a: asset.Spot, pair: currency.NewPair(currency.BTC, currency.USDC), exp: tradeBaseURL + tradeSpot + "BTC_USDC"},
		{a: asset.Options, pair: currency.NewPair(currency.BTC, currency.NewCode("14JUN24-62000-C")), exp: tradeBaseURL + tradeOptions + "BTC/BTC-14JUN24"},
		{a: asset.Options, pair: currency.NewPair(currency.BTC, currency.NewCode("14jun24-62000-p")), exp: tradeBaseURL + tradeOptions + "BTC/BTC-14JUN24"},
		{a: asset.Options, pair: currency.NewPair(currency.BTC, currency.NewCode("14JUN24-62000-X")), exp: tradeBaseURL + tradeOptions + "BTC"},
		{a: asset.Options, pair: currency.NewPair(currency.BTC, currency.NewCode("C")), exp: tradeBaseURL + tradeOptions + "BTC"},
	} {
		resp, err := e.GetCurrencyTradeURL(t.Context(), tc.a, tc.pair)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, resp, "GetCurrencyTradeURL should link %s %s", tc.a, tc.pair)
	}
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Margin, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL should reject an unsupported asset")
}

func TestOrderDetail(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	o := &Order{
		OrderID:             "ETH-584849853",
		OrderState:          "open",
		OrderType:           "stop_limit",
		TimeInForce:         "good_til_cancelled",
		InstrumentName:      "ETH-PERPETUAL",
		CreationTimestamp:   types.Time(time.UnixMilli(1550659803407)),
		LastUpdateTimestamp: types.Time(time.UnixMilli(1550659803470)),
		Direction:           DirectionSell,
		Price:               OrderPrice{Value: 140},
		Label:               "client-1",
		PostOnly:            true,
		ReduceOnly:          true,
		Amount:              40,
		FilledAmount:        10,
		AveragePrice:        141.5,
		TriggerPrice:        145,
	}
	detail, err := ex.orderDetail(o)
	require.NoError(t, err, "orderDetail must not error")
	exp := &order.Detail{
		Exchange:             ex.Name,
		OrderID:              "ETH-584849853",
		ClientOrderID:        "client-1",
		Pair:                 currency.NewPairWithDelimiter("ETH", perpString, currency.DashDelimiter),
		AssetType:            asset.Futures,
		Side:                 order.Sell,
		Type:                 order.StopLimit,
		Status:               order.PartiallyFilled,
		TimeInForce:          order.GoodTillCancel | order.PostOnly,
		ReduceOnly:           true,
		Price:                140,
		TriggerPrice:         145,
		Amount:               40,
		ExecutedAmount:       10,
		RemainingAmount:      30,
		AverageExecutedPrice: 141.5,
		Date:                 time.UnixMilli(1550659803407),
		LastUpdated:          time.UnixMilli(1550659803470),
	}
	assert.Equal(t, exp, detail, "orderDetail should convert every field")

	for _, tc := range []struct {
		mutate func(*Order)
		err    error
	}{
		{mutate: func(o *Order) { o.InstrumentName = "ETH" }, err: errUnsupportedInstrumentFormat},
		{mutate: func(o *Order) { o.Direction = "up" }, err: order.ErrSideIsInvalid},
		{mutate: func(o *Order) { o.OrderType = "iceberg" }, err: errUnsupportedOrderType},
		{mutate: func(o *Order) { o.OrderState = "lost" }, err: errUnsupportedOrderStatus},
		{mutate: func(o *Order) { o.TimeInForce = "forever" }, err: order.ErrInvalidTimeInForce},
	} {
		bad := *o
		tc.mutate(&bad)
		_, err := ex.orderDetail(&bad)
		assert.ErrorIs(t, err, tc.err, "orderDetail should return the conversion failure")
	}
}

func TestUserTradeFill(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	userTrade := &UserTrade{
		TradeID:        "48079254",
		InstrumentName: "BTC-PERPETUAL",
		Timestamp:      types.Time(time.UnixMilli(1590484156350)),
		OrderID:        "4008965646",
		Direction:      DirectionBuy,
		Price:          8954,
		Amount:         20,
		Label:          "client-2",
	}
	data, err := ex.userTradeFill(userTrade)
	require.NoError(t, err, "userTradeFill must not error")
	exp := &fill.Data{
		ID:            "48079254",
		TradeID:       "48079254",
		Timestamp:     time.UnixMilli(1590484156350),
		Exchange:      ex.Name,
		AssetType:     asset.Futures,
		CurrencyPair:  currency.NewPairWithDelimiter("BTC", perpString, currency.DashDelimiter),
		Side:          order.Buy,
		OrderID:       "4008965646",
		ClientOrderID: "client-2",
		Price:         8954,
		Amount:        20,
	}
	assert.Equal(t, exp, data, "userTradeFill should convert every field")

	bad := *userTrade
	bad.InstrumentName = "BTC"
	_, err = ex.userTradeFill(&bad)
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "userTradeFill should reject an instrument it cannot resolve")
	bad = *userTrade
	bad.Direction = "up"
	_, err = ex.userTradeFill(&bad)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "userTradeFill should reject an unknown direction")
}

func TestParseOrderType(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[string]order.Type{
		"limit":         order.Limit,
		"market":        order.Market,
		"market_limit":  order.Market,
		"stop_limit":    order.StopLimit,
		"stop_market":   order.StopMarket,
		"take_limit":    order.TakeProfit,
		"take_market":   order.TakeProfitMarket,
		"trailing_stop": order.TrailingStop,
		"liquidation":   order.Liquidation,
	} {
		parsed, err := parseOrderType(orderType)
		require.NoErrorf(t, err, "parseOrderType must not error for %s", orderType)
		assert.Equalf(t, exp, parsed, "parseOrderType should convert %s", orderType)
	}
	_, err := parseOrderType("LIMIT")
	assert.ErrorIs(t, err, errUnsupportedOrderType, "parseOrderType should reject a type Deribit does not send")
}

func TestParseOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		state  string
		filled float64
		exp    order.Status
	}{
		{state: "open", exp: order.Open},
		{state: "open", filled: 1, exp: order.PartiallyFilled},
		{state: "triggered", exp: order.Open},
		{state: "untriggered", exp: order.Pending},
		{state: "filled", filled: 1, exp: order.Filled},
		{state: "cancelled", exp: order.Cancelled},
		{state: "cancelled", filled: 1, exp: order.PartiallyFilledCancelled},
		{state: "rejected", exp: order.Rejected},
		{state: "archive", filled: 1, exp: order.Closed},
	} {
		status, err := parseOrderStatus(tc.state, tc.filled)
		require.NoErrorf(t, err, "parseOrderStatus must not error for %s", tc.state)
		assert.Equalf(t, tc.exp, status, "parseOrderStatus should convert %s with %v filled", tc.state, tc.filled)
	}
	_, err := parseOrderStatus("pending", 0)
	assert.ErrorIs(t, err, errUnsupportedOrderStatus, "parseOrderStatus should reject a state Deribit does not send")
}

const (
	// mockWrapperOptionCombo is the option combo the mock wrapper responses were recorded for
	mockWrapperOptionCombo = "BTC-CS-26MAR27-100000_130000"
	// wrapperTradesSpot is the spot pair the trade tape tests use, as Deribit serves no trade tape for the spot it routes
	// to Coinbase Exchange, such as BTC_USDC
	wrapperTradesSpot = "ETH_BTC"
	// mockHistoricTradesStart and mockHistoricTradesEnd bound the minute of BTC-PERPETUAL trades the mock historic
	// trades were recorded for; Deribit only serves the last 24 hours of trades, so live tests ask for recent ones
	mockHistoricTradesStart = 1792116000000
	mockHistoricTradesEnd   = 1792116060000
	// mockWrapperEditOrderID, mockWrapperStateOrderID and mockWrapperCancelOrderID are the orders of Deribit's
	// documented edit, order state and cancel examples, and mockWrapperLabel and mockWrapperLabelInstrument the label and
	// instrument of its open orders by label example
	mockWrapperEditOrderID     = "438994"
	mockWrapperStateOrderID    = "ETH-331562"
	mockWrapperCancelOrderID   = "ETH-SLIS-12"
	mockWrapperLabel           = "fooBar"
	mockWrapperLabelInstrument = "BTC-15FEB19-3250-P"
)

// mockWrapperBuySubmit returns the order of Deribit's documented buy example
func mockWrapperBuySubmit() *order.Submit {
	return &order.Submit{
		Exchange:      "Deribit",
		Pair:          currency.NewPairWithDelimiter("ETH", perpString, currency.DashDelimiter),
		AssetType:     asset.Futures,
		Side:          order.Buy,
		Type:          order.Market,
		Amount:        40,
		ClientOrderID: "market0000234",
	}
}

// mockWrapperSellSubmit returns the order of Deribit's documented sell example
func mockWrapperSellSubmit() *order.Submit {
	return &order.Submit{
		Exchange:         "Deribit",
		Pair:             currency.NewPairWithDelimiter("ETH", perpString, currency.DashDelimiter),
		AssetType:        asset.Futures,
		Side:             order.Sell,
		Type:             order.StopLimit,
		Amount:           123,
		Price:            145.61,
		TriggerPrice:     145,
		TriggerPriceType: order.LastPrice,
	}
}

// mockWrapperWithdrawRequest returns the withdrawal of Deribit's documented withdraw example, without the priority and
// nonce the wrapper does not send
func mockWrapperWithdrawRequest() *withdraw.Request {
	return &withdraw.Request{
		Exchange: "Deribit",
		Currency: currency.BTC,
		Amount:   0.4,
		Type:     withdraw.Crypto,
		Crypto:   withdraw.CryptoRequest{Address: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz"},
	}
}

var listedWrapperOptionCombo = sync.OnceValues(func() (string, error) { return listedMarketDataInstrument(KindOptionCombo) })

// wrapperTestPair returns the pair of an asset the wrapper tests use: BTC_USDC, BTC-PERPETUAL and the mock option and
// combos, or listed ones live
func wrapperTestPair(t *testing.T, a asset.Item) currency.Pair {
	t.Helper()
	var name string
	switch a {
	case asset.Spot:
		name = "BTC_USDC"
	case asset.Futures:
		name = "BTC-PERPETUAL"
	case asset.Options:
		name = marketDataOption(t)
	case asset.FutureCombo:
		name = marketDataFutureCombo(t)
	case asset.OptionCombo:
		name = mockWrapperOptionCombo
		if !mockTests {
			var err error
			name, err = listedWrapperOptionCombo()
			require.NoError(t, err, "listedWrapperOptionCombo must not error")
		}
	default:
		require.Failf(t, "no test pair", "the wrapper tests have no %s pair", a)
	}
	pair, err := currency.NewPairFromString(name)
	require.NoErrorf(t, err, "NewPairFromString must not error for %s", name)
	return pair
}

// setWrapperTestPairs makes pairs the only available and enabled pairs of an asset
func setWrapperTestPairs(t *testing.T, ex *Exchange, a asset.Item, pairs ...currency.Pair) {
	t.Helper()
	require.NoError(t, ex.SetPairs(pairs, a, false), "SetPairs must not error for the available pairs")
	require.NoError(t, ex.SetPairs(pairs, a, true), "SetPairs must not error for the enabled pairs")
}

// wrapperHistoricTradesWindow returns the minute of BTC-PERPETUAL trades GetHistoricTrades tests ask for: the mock
// window, or the last minute live
func wrapperHistoricTradesWindow() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(mockHistoricTradesStart), time.UnixMilli(mockHistoricTradesEnd)
	}
	end = time.Now()
	return end.Add(-time.Minute), end
}

// newWrapperServerExchange returns an exchange whose REST requests the answer function serves by method and params,
// for responses the mock recording cannot express, such as pages and failures
func newWrapperServerExchange(t *testing.T, answer func(method string, params json.RawMessage) string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "Reading the request body should not error") {
			return
		}
		var req websocketVenueRequest
		if !assert.NoError(t, json.Unmarshal(raw, &req), "Decoding the request body should not error") {
			return
		}
		_, err = w.Write([]byte(answer(req.Method, req.Params)))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

func TestWebsocketForWrapper(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	assert.False(t, ex.websocketForWrapper(false), "websocketForWrapper should send public requests over HTTP without a connection")
	assert.False(t, ex.websocketForWrapper(true), "websocketForWrapper should send private requests over HTTP without a connection")
}

func TestWrapperOverWebsocket(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the mock websocket records the requests the wrapper sends; live tests share Deribit's websocket")
	}
	ex, venue := newWebsocketTestExchange(t)
	require.True(t, ex.websocketForWrapper(true), "websocketForWrapper must send private requests over an authenticated connection")
	_, err := ex.UpdateTicker(t.Context(), wrapperTestPair(t, asset.Futures), asset.Futures)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.Len(t, venue.received("public/ticker"), 1, "UpdateTicker should request the ticker over the websocket")
	_, err = ex.GetOrderInfo(t.Context(), mockWrapperStateOrderID, currency.EMPTYPAIR, asset.Futures)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Len(t, venue.received("private/get_order_state"), 1, "GetOrderInfo should request the order over the websocket")

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	assert.True(t, ex.websocketForWrapper(false), "websocketForWrapper should send public requests over an unauthenticated connection")
	assert.False(t, ex.websocketForWrapper(true), "websocketForWrapper should send private requests over HTTP without an authenticated connection")
}

func TestPairInstrument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a    asset.Item
		pair string
		exp  string
	}{
		{a: asset.Spot, pair: "BTC_USDC", exp: "BTC_USDC"},
		{a: asset.Futures, pair: "ETH_USDC-PERPETUAL", exp: "ETH_USDC-PERPETUAL"},
		{a: asset.Options, pair: "XRP_USDC-27NOV26-2d5-C", exp: "XRP_USDC-27NOV26-2d5-C"},
		{a: asset.FutureCombo, pair: "BTC-FS-27NOV26_PERP", exp: "BTC-FS-27NOV26_PERP"},
		{a: asset.OptionCombo, pair: "BTC-CS-26MAR27-100000_130000", exp: "BTC-CS-26MAR27-100000_130000"},
	} {
		pair, err := currency.NewPairFromString(tc.pair)
		require.NoErrorf(t, err, "NewPairFromString must not error for %s", tc.pair)
		name, err := e.pairInstrument(pair, tc.a)
		require.NoErrorf(t, err, "pairInstrument must not error for %s", tc.pair)
		assert.Equalf(t, tc.exp, name, "pairInstrument should return %s's instrument name", tc.pair)
	}
	_, err := e.pairInstrument(currency.EMPTYPAIR, asset.Futures)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "pairInstrument should reject an empty pair")
	_, err = e.pairInstrument(currency.NewBTCUSDT(), asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "pairInstrument should reject an unsupported asset")
}

func TestAssetKind(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]string{
		asset.Spot:        KindSpot,
		asset.Futures:     KindFuture,
		asset.Options:     KindOption,
		asset.FutureCombo: KindFutureCombo,
		asset.OptionCombo: KindOptionCombo,
	} {
		kind, err := e.assetKind(a)
		require.NoErrorf(t, err, "assetKind must not error for %s", a)
		assert.Equalf(t, exp, kind, "assetKind should return %s's kind", a)
	}
	_, err := e.assetKind(asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "assetKind should reject an unsupported asset")
}

func TestStringToAssetKind(t *testing.T) {
	t.Parallel()
	for kind, exp := range map[string]asset.Item{
		KindSpot:        asset.Spot,
		KindFuture:      asset.Futures,
		"OPTION":        asset.Options,
		KindFutureCombo: asset.FutureCombo,
		KindOptionCombo: asset.OptionCombo,
	} {
		a, err := e.StringToAssetKind(kind)
		require.NoErrorf(t, err, "StringToAssetKind must not error for %s", kind)
		assert.Equalf(t, exp, a, "StringToAssetKind should return %s's asset", kind)
	}
	_, err := e.StringToAssetKind(KindCombo)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "StringToAssetKind should reject a kind that is no asset")
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for _, a := range ex.GetAssetTypes(false) {
		pairs, err := ex.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		assert.Truef(t, pairs.Contains(wrapperTestPair(t, a), true), "FetchTradablePairs should return %s's test pair", a)
	}
	assert.True(t, ex.isCoinbaseRoutedSpot("BTC_USDC"), "FetchTradablePairs should remember the spot Deribit routes to Coinbase Exchange")
	assert.False(t, ex.isCoinbaseRoutedSpot(wrapperTradesSpot), "FetchTradablePairs should not remember spot Deribit matches itself")
	_, err := ex.FetchTradablePairs(t.Context(), asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs should reject an unsupported asset")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	for _, a := range ex.GetAssetTypes(false) {
		pairs, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		assert.Truef(t, pairs.Contains(wrapperTestPair(t, a), true), "UpdateTradablePairs should store %s's test pair", a)
	}
	assert.True(t, ex.isCoinbaseRoutedSpot("BTC_USDC"), "UpdateTradablePairs should remember the spot Deribit routes to Coinbase Exchange")
}

func TestTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	instruments := []Instrument{
		{InstrumentName: "BTC_USDC", IsActive: true, IsCoinbaseExchangeRouted: true},
		{InstrumentName: "ETH_USDC", IsActive: true, IsCoinbaseSpotRouted: true},
		{InstrumentName: "ETH_BTC", IsActive: true},
		{InstrumentName: "ETH_BTC", IsActive: true},
		{InstrumentName: "SOL_USDC", IsCoinbaseExchangeRouted: true},
	}
	pairs, err := ex.tradablePairs(asset.Spot, instruments)
	require.NoError(t, err, "tradablePairs must not error")
	exp := currency.Pairs{
		currency.NewPairWithDelimiter("BTC", "USDC", currency.UnderscoreDelimiter),
		currency.NewPairWithDelimiter("ETH", "USDC", currency.UnderscoreDelimiter),
		currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter),
	}
	assert.Equal(t, exp, pairs, "tradablePairs should return each active instrument once")
	assert.Equal(t, map[string]struct{}{"BTC_USDC": {}, "ETH_USDC": {}}, ex.coinbaseRoutedSpot, "tradablePairs should remember the active spot Deribit routes to Coinbase Exchange")

	_, err = ex.tradablePairs(asset.Futures, []Instrument{{InstrumentName: "BTC-PERPETUAL", IsActive: true, IsCoinbaseExchangeRouted: true}})
	require.NoError(t, err, "tradablePairs must not error for futures")
	assert.Equal(t, map[string]struct{}{"BTC_USDC": {}, "ETH_USDC": {}}, ex.coinbaseRoutedSpot, "tradablePairs should only remember routed instruments for spot")

	_, err = ex.tradablePairs(asset.Spot, []Instrument{{InstrumentName: "", IsActive: true}})
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "tradablePairs should reject an instrument it cannot parse")
}

func TestGenerateSubscriptionsLeavesOutCoinbaseRoutedStreams(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	routed := currency.NewPairWithDelimiter("BTC", "USDC", currency.UnderscoreDelimiter)
	matched := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	setWrapperTestPairs(t, ex, asset.Spot, routed, matched)
	for _, a := range ex.GetAssetTypes(false) {
		if a != asset.Spot {
			require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
		}
	}
	_, err := ex.tradablePairs(asset.Spot, []Instrument{{InstrumentName: "BTC_USDC", IsActive: true, IsCoinbaseExchangeRouted: true}})
	require.NoError(t, err, "tradablePairs must not error")
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneDay},
	}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	channels := make([]string, len(subs))
	for i := range subs {
		channels[i] = subs[i].QualifiedChannel
	}
	assert.ElementsMatch(t, []string{"ticker.BTC_USDC.100ms", "ticker.ETH_BTC.100ms", "trades.ETH_BTC.100ms", "chart.trades.ETH_BTC.1D"}, channels, "generateSubscriptions should leave out the trades and candles of Coinbase-routed spot")
}

func TestIsUnstreamedSubscription(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.coinbaseRoutedSpot = map[string]struct{}{"BTC_USDC": {}}
	routed := currency.NewPairWithDelimiter("BTC", "USDC", currency.UnderscoreDelimiter)
	matched := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		exp  bool
	}{
		{name: "routed trades", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{routed}}, exp: true},
		{name: "routed candles", sub: &subscription.Subscription{Asset: asset.Spot, Channel: ChannelChartTrades, Pairs: currency.Pairs{routed}}, exp: true},
		{name: "routed ticker", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{routed}}},
		{name: "matched trades", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{matched}}},
		{name: "futures trades", sub: &subscription.Subscription{Asset: asset.Futures, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{routed}}},
		{name: "trades of several pairs", sub: &subscription.Subscription{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{routed, matched}}},
		{name: "an unknown channel", sub: &subscription.Subscription{Asset: asset.Spot, Channel: "bogus", Pairs: currency.Pairs{routed}}},
	} {
		assert.Equalf(t, tc.exp, ex.isUnstreamedSubscription(tc.sub), "isUnstreamedSubscription should report %s correctly", tc.name)
	}
}

// wrapperTickerPair returns the pair UpdateTickers is tested with: the test pair, except for mock options, whose book
// summaries were recorded for another option
func wrapperTickerPair(t *testing.T, a asset.Item) currency.Pair {
	t.Helper()
	if mockTests && a == asset.Options {
		return currency.NewPairWithDelimiter("BTC", "25DEC26-104000-C", currency.DashDelimiter)
	}
	return wrapperTestPair(t, a)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for _, a := range ex.GetAssetTypes(false) {
		pair := wrapperTickerPair(t, a)
		setWrapperTestPairs(t, ex, a, pair)
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
		tick, err := ticker.GetTicker(ex.Name, pair, a)
		require.NoErrorf(t, err, "GetTicker must not error for %s", a)
		assert.NotZerof(t, tick.MarkPrice, "UpdateTickers should store %s's mark price", a)
	}

	pair := wrapperTestPair(t, asset.Futures)
	unsummarised := currency.NewPairWithDelimiter("BTC", "01JAN20", currency.DashDelimiter)
	setWrapperTestPairs(t, ex, asset.Futures, pair, unsummarised)
	err := ex.UpdateTickers(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, errBookSummaryNotFound, "UpdateTickers should report a pair Deribit does not summarise")
	tick, err := ticker.GetTicker(ex.Name, pair, asset.Futures)
	require.NoError(t, err, "GetTicker must not error for the summarised pair")
	if mockTests {
		exp := &ticker.Price{
			ExchangeName: ex.Name,
			Pair:         pair,
			AssetType:    asset.Futures,
			LastUpdated:  time.UnixMilli(1792117610662),
			Bid:          82032,
			Ask:          82032.5,
			Last:         82036,
			Close:        82036,
			High:         83000,
			Low:          79709.5,
			BaseVolume:   25208.76381846,
			QuoteVolume:  2049538340,
			MarkPrice:    82032.7,
			OpenInterest: 869347230,
		}
		assert.Equal(t, exp, tick, "UpdateTickers should store the summarised pair's ticker")
	}

	assert.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Margin), asset.ErrNotSupported, "UpdateTickers should reject an unsupported asset")
}

func TestSummaryCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a          asset.Item
		instrument string
		exp        currency.Code
	}{
		{a: asset.Spot, instrument: "ETH_BTC", exp: currency.ETH},
		{a: asset.Futures, instrument: "BTC-PERPETUAL", exp: currency.BTC},
		{a: asset.Futures, instrument: "ETH_USDC-PERPETUAL", exp: currency.USDC},
		{a: asset.Options, instrument: "XRP_USDC-30OCT26-1d46-P", exp: currency.USDC},
		{a: asset.Options, instrument: "BTC-27NOV26-90000-C", exp: currency.BTC},
		{a: asset.FutureCombo, instrument: "ETH_USDC-FS-27NOV26_PERP", exp: currency.USDC},
		{a: asset.OptionCombo, instrument: "BTC-CS-26MAR27-100000_130000", exp: currency.BTC},
	} {
		assert.Truef(t, tc.exp.Equal(summaryCurrency(tc.a, tc.instrument)), "summaryCurrency should summarise %s under %s", tc.instrument, tc.exp)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for _, a := range ex.GetAssetTypes(false) {
		pair := wrapperTestPair(t, a)
		tick, err := ex.UpdateTicker(t.Context(), pair, a)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s", a)
		assert.Equalf(t, a, tick.AssetType, "UpdateTicker should return %s's ticker", a)
		assert.NotZerof(t, tick.LastUpdated, "UpdateTicker should return %s's ticker time", a)
	}
	if mockTests {
		pair := wrapperTestPair(t, asset.Futures)
		tick, err := ex.UpdateTicker(t.Context(), pair, asset.Futures)
		require.NoError(t, err, "UpdateTicker must not error")
		exp := &ticker.Price{
			ExchangeName: ex.Name,
			Pair:         pair,
			AssetType:    asset.Futures,
			LastUpdated:  time.UnixMilli(1792109806455),
			Bid:          82312.5,
			BidSize:      96920,
			Ask:          82313,
			AskSize:      10,
			Last:         82306.5,
			Close:        82306.5,
			High:         83000,
			Low:          79709.5,
			BaseVolume:   26600.15102531,
			QuoteVolume:  2162019170,
			MarkPrice:    82302.33,
			IndexPrice:   82285.56,
			OpenInterest: 862179360,
		}
		assert.Equal(t, exp, tick, "UpdateTicker should convert the ticker")
	}
	_, err := e.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Futures)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker should reject an empty pair")
}

func TestTickerResponseTickerPrice(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "27NOV26-90000-C", currency.DashDelimiter)
	data := &TickerResponse{
		Timestamp:     types.Time(time.UnixMilli(1792109806455)),
		BestBidPrice:  0.018,
		BestBidAmount: 3.3,
		BestAskPrice:  0.0185,
		BestAskAmount: 16,
		LastPrice:     0.0182,
		MarkPrice:     0.0183,
		IndexPrice:    82285.56,
		OpenInterest:  450,
		Stats:         TickerStats{High: 0.02, Low: 0.017, Volume: 12, VolumeUSD: 18000, VolumeNotional: 0},
	}
	exp := &ticker.Price{
		ExchangeName: "Deribit",
		Pair:         pair,
		AssetType:    asset.Options,
		LastUpdated:  time.UnixMilli(1792109806455),
		Bid:          0.018,
		BidSize:      3.3,
		Ask:          0.0185,
		AskSize:      16,
		Last:         0.0182,
		Close:        0.0182,
		High:         0.02,
		Low:          0.017,
		BaseVolume:   12,
		MarkPrice:    0.0183,
		IndexPrice:   82285.56,
		OpenInterest: 450,
	}
	assert.Equal(t, exp, data.tickerPrice("Deribit", pair, asset.Options), "tickerPrice should convert the ticker, leaving out an option's USD volume")
}

func TestBookSummaryTickerPrice(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USDC", currency.UnderscoreDelimiter)
	summary := &BookSummary{
		CreationTimestamp: types.Time(time.UnixMilli(1792117610661)),
		BidPrice:          82001.12,
		AskPrice:          82001.13,
		LastPrice:         82001.5,
		High:              83000,
		Low:               80000,
		Volume:            12.5,
		VolumeUSD:         1025000,
		MarkPrice:         82007.58,
		OpenInterest:      0,
	}
	exp := &ticker.Price{
		ExchangeName: "Deribit",
		Pair:         pair,
		AssetType:    asset.Spot,
		LastUpdated:  time.UnixMilli(1792117610661),
		Bid:          82001.12,
		Ask:          82001.13,
		Last:         82001.5,
		Close:        82001.5,
		High:         83000,
		Low:          80000,
		BaseVolume:   12.5,
		QuoteVolume:  1025000,
		MarkPrice:    82007.58,
	}
	assert.Equal(t, exp, summary.tickerPrice("Deribit", pair, asset.Spot), "tickerPrice should convert the summary, taking the USD volume without a notional volume")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for _, a := range ex.GetAssetTypes(false) {
		pair := wrapperTestPair(t, a)
		book, err := ex.UpdateOrderbook(t.Context(), pair, a)
		require.NoErrorf(t, err, "UpdateOrderbook must not error for %s", a)
		assert.Equalf(t, a, book.Asset, "UpdateOrderbook should return %s's book", a)
	}
	if mockTests {
		pair := wrapperTestPair(t, asset.Futures)
		book, err := ex.UpdateOrderbook(t.Context(), pair, asset.Futures)
		require.NoError(t, err, "UpdateOrderbook must not error")
		assert.Equal(t, orderbook.Levels{{Price: 82032, Amount: 16120}, {Price: 82031.5, Amount: 10}, {Price: 82031, Amount: 30}, {Price: 82028.5, Amount: 70}, {Price: 82027.5, Amount: 62230}}, book.Bids, "UpdateOrderbook should store the bids best first")
		assert.Equal(t, orderbook.Levels{{Price: 82032.5, Amount: 105130}, {Price: 82033, Amount: 102450}, {Price: 82033.5, Amount: 10}, {Price: 82034, Amount: 77770}, {Price: 82034.5, Amount: 3600}}, book.Asks, "UpdateOrderbook should store the asks best first")
		assert.Equal(t, time.UnixMilli(1792117611194), book.LastUpdated, "UpdateOrderbook should store the book's time")
		assert.Equal(t, int64(178660908974), book.LastUpdateID, "UpdateOrderbook should store the book's change ID")
	}
	_, err := e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Futures)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook should reject an empty pair")
}

func TestOrderBookLevels(t *testing.T) {
	t.Parallel()
	levels := orderBookLevels([]OrderBookLevel{{Price: 1.5, Amount: 10}, {Price: 0, Amount: 20}, {Price: -0.5, Amount: 30}})
	assert.Equal(t, orderbook.Levels{{Price: 1.5, Amount: 10}, {Price: -0.5, Amount: 30}}, levels, "orderBookLevels should convert the levels, leaving out a spread's zero price")
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex := newTestExchange(t)
	subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, subAccounts, 1, "UpdateAccountBalances must return one subaccount")
	assert.Equal(t, asset.Futures, subAccounts[0].AssetType, "UpdateAccountBalances should save the balances under the asset requested")
	if mockTests {
		btc := subAccounts[0].Balances[currency.BTC]
		assert.Equal(t, 302.60065765, btc.Total, "UpdateAccountBalances should take the cash balance as the total")
		assert.Equal(t, 301.35396172, btc.Free, "UpdateAccountBalances should take the withdrawable funds as free")
		assert.InDelta(t, 302.60065765-301.35396172, btc.Hold, 1e-9, "UpdateAccountBalances should hold the rest of the balance")
		assert.Equal(t, 100.0, subAccounts[0].Balances[currency.ETH].Total, "UpdateAccountBalances should return every currency's balance")
	}
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances should reject an unsupported asset")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	assert.NoError(t, newTestExchange(t).ValidateAPICredentials(t.Context(), asset.Futures), "ValidateAPICredentials should not error")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	history, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		return
	}
	require.Len(t, history, 6, "GetAccountFundingHistory must return a deposit and a withdrawal for each mock currency")
	exp := []exchange.FundingHistory{
		{
			ExchangeName:    e.Name,
			Status:          "completed",
			TransferID:      "230669110fdaf0a0dbcdc079b6b8b43d5af29cc73683835b9bc6b3406c065fda",
			Timestamp:       time.UnixMilli(1549295017670),
			Currency:        "BTC",
			Amount:          5,
			TransferType:    "deposit",
			CryptoToAddress: "2N35qDKDY22zmJq9eSyiAerMD4enJ1xx6ax",
			CryptoTxID:      "230669110fdaf0a0dbcdc079b6b8b43d5af29cc73683835b9bc6b3406c065fda",
		},
		{
			ExchangeName:    e.Name,
			Status:          "unconfirmed",
			TransferID:      "1",
			Timestamp:       time.UnixMilli(1550571443070),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0001,
			TransferType:    "withdrawal",
			CryptoToAddress: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz",
		},
	}
	assert.Equal(t, exp, history[:2], "GetAccountFundingHistory should convert the deposits and withdrawals")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	history, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if mockTests {
		exp := []exchange.WithdrawalHistory{{
			Status:          "unconfirmed",
			TransferID:      "1",
			Timestamp:       time.UnixMilli(1550571443070),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0001,
			TransferType:    "withdrawal",
			CryptoToAddress: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBz",
		}}
		assert.Equal(t, exp, history, "GetWithdrawalsHistory should convert the withdrawals")
	}
	all, err := e.GetWithdrawalsHistory(t.Context(), currency.EMPTYCODE, asset.Futures)
	require.NoError(t, err, "GetWithdrawalsHistory must not error for every currency")
	if mockTests {
		assert.Len(t, all, 3, "GetWithdrawalsHistory should return the withdrawals of each currency available to the account")
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	spot, err := currency.NewPairFromString(wrapperTradesSpot)
	require.NoError(t, err, "NewPairFromString must not error")
	for a, pair := range map[asset.Item]currency.Pair{asset.Futures: wrapperTestPair(t, asset.Futures), asset.Spot: spot} {
		trades, err := e.GetRecentTrades(t.Context(), pair, a)
		require.NoErrorf(t, err, "GetRecentTrades must not error for %s", a)
		require.NotEmptyf(t, trades, "GetRecentTrades must return %s trades", a)
		assert.Equalf(t, a, trades[0].AssetType, "GetRecentTrades should return %s trades", a)
	}
	if mockTests {
		pair := wrapperTestPair(t, asset.Futures)
		trades, err := e.GetRecentTrades(t.Context(), pair, asset.Futures)
		require.NoError(t, err, "GetRecentTrades must not error")
		require.Len(t, trades, 5, "GetRecentTrades must return the mock trades")
		exp := trade.Data{
			TID:          "452905643",
			Exchange:     e.Name,
			CurrencyPair: pair,
			AssetType:    asset.Futures,
			Side:         order.Buy,
			Price:        82043,
			Amount:       25450,
			Timestamp:    time.UnixMilli(1792117615950),
		}
		assert.Equal(t, exp, trades[0], "GetRecentTrades should convert the trades")
	}
	_, err = e.GetRecentTrades(t.Context(), currency.EMPTYPAIR, asset.Futures)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetRecentTrades should reject an empty pair")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	pair := wrapperTestPair(t, asset.Futures)
	start, end := wrapperHistoricTradesWindow()
	trades, err := e.GetHistoricTrades(t.Context(), pair, asset.Futures, start, end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	require.NotEmpty(t, trades, "GetHistoricTrades must return trades")
	for i := range trades {
		assert.Falsef(t, trades[i].Timestamp.Before(start) || trades[i].Timestamp.After(end), "GetHistoricTrades should return trade %s within the range", trades[i].TID)
	}
	if mockTests {
		require.Len(t, trades, 5, "GetHistoricTrades must return the mock trades")
		exp := trade.Data{
			TID:          "452901060",
			Exchange:     e.Name,
			CurrencyPair: pair,
			AssetType:    asset.Futures,
			Side:         order.Buy,
			Price:        82003,
			Amount:       10,
			Timestamp:    time.UnixMilli(1792116001473),
		}
		assert.Equal(t, exp, trades[0], "GetHistoricTrades should convert the trades")
	}
	_, err = e.GetHistoricTrades(t.Context(), pair, asset.Futures, end, start)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades should reject a start after the end")
}

func TestGetHistoricTradesPaging(t *testing.T) {
	t.Parallel()
	const start = 1792116000000
	page := func(hasMore bool, trades ...string) string {
		return `{"jsonrpc":"2.0","result":{"has_more":` + strconv.FormatBool(hasMore) + `,"trades":[` + strings.Join(trades, ",") + `]}}`
	}
	tradeAt := func(id string, offset int64) string {
		return `{"trade_id":"` + id + `","direction":"buy","price":82000,"amount":10,"timestamp":` + strconv.FormatInt(start+offset, 10) + `}`
	}
	var mu sync.Mutex
	var starts []int64
	ex := newWrapperServerExchange(t, func(_ string, params json.RawMessage) string {
		var req struct {
			StartTimestamp int64 `json:"start_timestamp"`
		}
		assert.NoError(t, json.Unmarshal(params, &req), "Decoding the params should not error")
		mu.Lock()
		starts = append(starts, req.StartTimestamp)
		mu.Unlock()
		if req.StartTimestamp == start {
			return page(true, tradeAt("1", 1), tradeAt("2", 2), tradeAt("3", 3))
		}
		return page(false, tradeAt("3", 3), tradeAt("4", 4))
	})
	pair := wrapperTestPair(t, asset.Futures)
	trades, err := ex.GetHistoricTrades(t.Context(), pair, asset.Futures, time.UnixMilli(start), time.UnixMilli(start+60000))
	require.NoError(t, err, "GetHistoricTrades must not error")
	ids := make([]string, len(trades))
	for i := range trades {
		ids[i] = trades[i].TID
	}
	assert.Equal(t, []string{"1", "2", "3", "4"}, ids, "GetHistoricTrades should page forward without repeating trades")
	assert.Equal(t, []int64{start, start + 3}, starts, "GetHistoricTrades should start each page at the last page's final trade")

	stalled := newWrapperServerExchange(t, func(string, json.RawMessage) string {
		return page(true, tradeAt("1", 0), tradeAt("2", 0))
	})
	_, err = stalled.GetHistoricTrades(t.Context(), pair, asset.Futures, time.UnixMilli(start), time.UnixMilli(start+60000))
	assert.ErrorIs(t, err, errTradePagingStalled, "GetHistoricTrades should stop when a page cannot advance")
}

// wrapperCandlesWindow returns the 4 hours of BTC-PERPETUAL candles the candle tests ask for: the mock range, or the
// last 4 whole hours live
func wrapperCandlesWindow() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(marketDataRangeStart), time.UnixMilli(marketDataRangeEnd)
	}
	end = time.Now().Truncate(time.Hour)
	return end.Add(-4 * time.Hour), end
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := wrapperTestPair(t, asset.Futures)
	start, end := wrapperCandlesWindow()
	candles, err := e.GetHistoricCandles(t.Context(), pair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error")
	require.Len(t, candles.Candles, 4, "GetHistoricCandles must return a candle per hour")
	if mockTests {
		exp := kline.Candle{Time: time.UnixMilli(marketDataRangeStart).UTC(), Open: 81600, High: 81763, Low: 81312, Close: 81588, Volume: 962.20147922}
		assert.Equal(t, exp, candles.Candles[0], "GetHistoricCandles should convert the candles")
	}
	_, err = e.GetHistoricCandles(t.Context(), pair, asset.Futures, kline.FifteenSecond, start, end)
	assert.ErrorIs(t, err, kline.ErrCannotConstructInterval, "GetHistoricCandles should reject an interval Deribit's candles cannot build")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair := wrapperTestPair(t, asset.Futures)
	start, end := wrapperCandlesWindow()
	candles, err := e.GetHistoricCandlesExtended(t.Context(), pair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	assert.Len(t, candles.Candles, 4, "GetHistoricCandlesExtended should return a candle per hour")
}

func TestChartCandles(t *testing.T) {
	t.Parallel()
	pair := wrapperTestPair(t, asset.Futures)
	ex := newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"status":"ok","ticks":[1792022400000,1792026000000],"open":[1,2],"high":[3,4],"low":[0.5,1],"close":[2,3],"volume":[10]}}`)
	_, err := ex.chartCandles(t.Context(), pair, asset.Futures, kline.OneHour, time.UnixMilli(marketDataRangeStart), time.UnixMilli(marketDataRangeEnd))
	assert.ErrorIs(t, err, errCandleFieldsMismatch, "chartCandles should reject candle fields of different lengths")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	serverTime, err := e.GetServerTime(t.Context(), asset.Futures)
	require.NoError(t, err, "GetServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1792107016663), serverTime, "GetServerTime should return Deribit's time")
	} else {
		assert.WithinDuration(t, time.Now(), serverTime, time.Minute, "GetServerTime should return the current time")
	}
}

// newLiveWrapperOrder returns a post only BTC-PERPETUAL limit buy no market move fills while the test runs, which is
// cancelled when the test ends
func newLiveWrapperOrder(t *testing.T) *order.Submit {
	t.Helper()
	arg := newLiveTradingOrderRequest(t, DirectionBuy)
	return &order.Submit{
		Exchange:      e.Name,
		Pair:          wrapperTestPair(t, asset.Futures),
		AssetType:     asset.Futures,
		Side:          order.Buy,
		Type:          order.Limit,
		Amount:        arg.Amount,
		Price:         arg.Price,
		ClientOrderID: arg.Label,
		TimeInForce:   order.PostOnly,
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder should reject a nil order")
	liquidation := mockWrapperBuySubmit()
	liquidation.Type = order.Liquidation
	_, err = e.SubmitOrder(t.Context(), liquidation)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "SubmitOrder should reject an order type Deribit cannot place")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		resp, err := e.SubmitOrder(t.Context(), newLiveWrapperOrder(t))
		require.NoError(t, err, "SubmitOrder must not error")
		assert.Equal(t, order.Open, resp.Status, "SubmitOrder should leave the order open")
		return
	}
	for _, s := range []*order.Submit{mockWrapperBuySubmit(), mockWrapperSellSubmit()} {
		resp, err := e.SubmitOrder(t.Context(), s)
		require.NoErrorf(t, err, "SubmitOrder must not error for a %s %s", s.Type, s.Side)
		assert.Equal(t, "ETH-584864807", resp.OrderID, "SubmitOrder should return the order's ID")
		assert.Equal(t, order.Filled, resp.Status, "SubmitOrder should return the order's status")
		assert.Equal(t, 198.75, resp.Price, "SubmitOrder should return the order's price")
		assert.Equal(t, 202.8, resp.AverageExecutedPrice, "SubmitOrder should return the order's average price")
		assert.Zero(t, resp.RemainingAmount, "SubmitOrder should return the order's remaining amount")
		assert.Equal(t, time.UnixMilli(1590486335742), resp.Date, "SubmitOrder should return the order's creation time")
		assert.Equal(t, 7.766e-05, resp.Fee, "SubmitOrder should total the trades' fees")
		assert.Equal(t, currency.ETH, resp.FeeAsset, "SubmitOrder should return the trades' fee currency")
		exp := []order.TradeHistory{{Price: 202.8, Amount: 21, Fee: 7.766e-05, Exchange: e.Name, TID: "ETH-2696097", Type: order.Limit, Side: order.Sell, Timestamp: time.UnixMilli(1590486335742), FeeAsset: "ETH"}}
		assert.Equal(t, exp, resp.Trades, "SubmitOrder should return the trades the order made")
	}
}

func TestSubmitOrderRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit order.Submit
		exp    *OrderRequest
	}{
		{
			name:   "a post only limit order",
			submit: order.Submit{Type: order.Limit, Amount: 10, Price: 100, ClientOrderID: "label", ReduceOnly: true, TimeInForce: order.GoodTillCancel | order.PostOnly},
			exp:    &OrderRequest{Amount: 10, Type: "limit", Label: "label", Price: 100, TimeInForce: "good_til_cancelled", PostOnly: true, RejectPostOnly: true, ReduceOnly: true},
		},
		{
			name:   "an immediate or cancel market order",
			submit: order.Submit{Type: order.Market, Amount: 10, Price: 100, TimeInForce: order.ImmediateOrCancel},
			exp:    &OrderRequest{Amount: 10, Type: "market", TimeInForce: "immediate_or_cancel"},
		},
		{
			name:   "a fill or kill limit order",
			submit: order.Submit{Type: order.Limit, Amount: 10, Price: 100, TimeInForce: order.FillOrKill},
			exp:    &OrderRequest{Amount: 10, Type: "limit", Price: 100, TimeInForce: "fill_or_kill"},
		},
		{
			name:   "a stop limit order on the mark price",
			submit: order.Submit{Type: order.StopLimit, Amount: 10, Price: 100, TriggerPrice: 95, TriggerPriceType: order.MarkPrice},
			exp:    &OrderRequest{Amount: 10, Type: "stop_limit", Price: 100, TriggerPrice: 95, Trigger: "mark_price"},
		},
		{
			name:   "a stop market order on the index price",
			submit: order.Submit{Type: order.StopMarket, Amount: 10, TriggerPrice: 95, TriggerPriceType: order.IndexPrice},
			exp:    &OrderRequest{Amount: 10, Type: "stop_market", TriggerPrice: 95, Trigger: "index_price"},
		},
		{
			name:   "a take profit limit order on the last price",
			submit: order.Submit{Type: order.TakeProfit, Amount: 10, Price: 110, TriggerPrice: 105},
			exp:    &OrderRequest{Amount: 10, Type: "take_limit", Price: 110, TriggerPrice: 105, Trigger: "last_price"},
		},
		{
			name:   "a good till day take profit market order",
			submit: order.Submit{Type: order.TakeProfitMarket, Amount: 10, TriggerPrice: 105, TimeInForce: order.GoodTillDay},
			exp:    &OrderRequest{Amount: 10, Type: "take_market", TriggerPrice: 105, Trigger: "last_price", TimeInForce: "good_til_day"},
		},
		{
			name:   "a trailing stop",
			submit: order.Submit{Type: order.TrailingStop, Amount: 10, TriggerPriceType: order.MarkPrice, TrackingMode: order.Distance, TrackingValue: 50},
			exp:    &OrderRequest{Amount: 10, Type: "trailing_stop", Trigger: "mark_price", TriggerOffset: 50},
		},
	} {
		req, err := submitOrderRequest(&tc.submit)
		require.NoErrorf(t, err, "submitOrderRequest must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, req, "submitOrderRequest should request %s", tc.name)
	}
	for _, tc := range []struct {
		name   string
		submit order.Submit
		err    error
	}{
		{name: "a liquidation", submit: order.Submit{Type: order.Liquidation}, err: order.ErrUnsupportedOrderType},
		{name: "a good till time order", submit: order.Submit{Type: order.Limit, TimeInForce: order.GoodTillTime}, err: order.ErrUnsupportedTimeInForce},
		{name: "an unknown trigger price", submit: order.Submit{Type: order.StopMarket, TriggerPriceType: order.UnknownPriceType}, err: order.ErrUnknownPriceType},
		{name: "a trailing stop by percentage", submit: order.Submit{Type: order.TrailingStop, TrackingMode: order.Percentage, TrackingValue: 1}, err: errTrailingStopDistance},
		{name: "a trailing stop without a distance", submit: order.Submit{Type: order.TrailingStop, TrackingMode: order.Distance}, err: errTrailingStopDistance},
	} {
		_, err := submitOrderRequest(&tc.submit)
		assert.ErrorIsf(t, err, tc.err, "submitOrderRequest should reject %s", tc.name)
	}
}

func TestTradeHistory(t *testing.T) {
	t.Parallel()
	userTrade := &UserTrade{TradeID: "ETH-2696097", Direction: DirectionSell, OrderType: "limit", Price: 202.8, Amount: 21, Fee: 7.766e-05, FeeCurrency: currency.ETH, Liquidity: "M", Timestamp: types.Time(time.UnixMilli(1590486335742))}
	history, err := tradeHistory("Deribit", userTrade)
	require.NoError(t, err, "tradeHistory must not error")
	exp := order.TradeHistory{Price: 202.8, Amount: 21, Fee: 7.766e-05, Exchange: "Deribit", TID: "ETH-2696097", Type: order.Limit, Side: order.Sell, Timestamp: time.UnixMilli(1590486335742), IsMaker: true, FeeAsset: "ETH"}
	assert.Equal(t, exp, history, "tradeHistory should convert the trade")

	bad := *userTrade
	bad.Direction = "up"
	_, err = tradeHistory("Deribit", &bad)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "tradeHistory should reject an unknown direction")
	bad = *userTrade
	bad.OrderType = "iceberg"
	_, err = tradeHistory("Deribit", &bad)
	assert.ErrorIs(t, err, errUnsupportedOrderType, "tradeHistory should reject an unknown order type")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder should reject a nil modification")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		placed, err := e.SubmitOrder(t.Context(), newLiveWrapperOrder(t))
		require.NoError(t, err, "SubmitOrder must not error")
		resp, err := e.ModifyOrder(t.Context(), &order.Modify{OrderID: placed.OrderID, Pair: placed.Pair, AssetType: asset.Futures, Price: placed.Price - 10, TimeInForce: order.PostOnly})
		require.NoError(t, err, "ModifyOrder must not error")
		assert.Equal(t, placed.Price-10, resp.Price, "ModifyOrder should move the order's price")
		assert.Equal(t, placed.Amount, resp.Amount, "ModifyOrder should keep the order's amount")
		return
	}
	futuresPair := wrapperTestPair(t, asset.Futures)
	resp, err := e.ModifyOrder(t.Context(), &order.Modify{OrderID: mockWrapperEditOrderID, Pair: futuresPair, AssetType: asset.Futures, Amount: 4, Price: 222})
	require.NoError(t, err, "ModifyOrder must not error")
	exp := &order.ModifyResponse{
		Exchange:        e.Name,
		OrderID:         "94166",
		ClientOrderID:   "i_love_deribit",
		Pair:            futuresPair,
		Type:            order.Limit,
		Side:            order.Buy,
		Status:          order.Open,
		AssetType:       asset.Futures,
		TimeInForce:     order.GoodTillCancel,
		Price:           50111,
		Amount:          150,
		RemainingAmount: 150,
		Date:            time.UnixMilli(1616155547764),
		LastUpdated:     time.UnixMilli(1616155550773),
	}
	assert.Equal(t, exp, resp, "ModifyOrder should return the edited order")

	ethPair := currency.NewPairWithDelimiter("ETH", perpString, currency.DashDelimiter)
	_, err = e.ModifyOrder(t.Context(), &order.Modify{OrderID: mockWrapperStateOrderID, Pair: ethPair, AssetType: asset.Futures, Price: 118.94, TimeInForce: order.PostOnly})
	require.NoError(t, err, "ModifyOrder must keep the amount of an order edited without one")
	labelled, err := currency.NewPairFromString(mockWrapperLabelInstrument)
	require.NoError(t, err, "NewPairFromString must not error")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{ClientOrderID: mockWrapperLabel, Pair: labelled, AssetType: asset.Options, Price: 0.003})
	require.NoError(t, err, "ModifyOrder must edit an order by its label")

	twoOrders := `{"jsonrpc":"2.0","result":[{"order_id":"1","instrument_name":"BTC-15FEB19-3250-P","amount":10},{"order_id":"2","instrument_name":"BTC-15FEB19-3250-P","amount":10}]}`
	_, err = newResponseServerExchange(t, twoOrders).ModifyOrder(t.Context(), &order.Modify{ClientOrderID: mockWrapperLabel, Pair: labelled, AssetType: asset.Options, Price: 0.003})
	assert.ErrorIs(t, err, errOrderLabelNotUnique, "ModifyOrder should reject a label several orders share")
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{}), order.ErrOrderIDNotSet, "CancelOrder should reject an order without an ID")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		placed, err := e.SubmitOrder(t.Context(), newLiveWrapperOrder(t))
		require.NoError(t, err, "SubmitOrder must not error")
		assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: placed.OrderID, Pair: placed.Pair, AssetType: asset.Futures}), "CancelOrder should not error")
		return
	}
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: mockWrapperCancelOrderID, AssetType: asset.Futures}), "CancelOrder should not error")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders should reject no orders")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{}})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders should reject an order without an ID")

	ex := newWrapperServerExchange(t, func(_ string, params json.RawMessage) string {
		if strings.Contains(string(params), "unknown") {
			return `{"jsonrpc":"2.0","error":{"code":11044,"message":"not_open_order"}}`
		}
		return `{"jsonrpc":"2.0","result":{"order_id":"known","order_state":"cancelled"}}`
	})
	resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "known"}, {OrderID: "unknown"}})
	require.NoError(t, err, "CancelBatchOrders must not error")
	assert.Equal(t, map[string]string{"known": "cancelled", "unknown": "authenticated request failed: code 11044: not_open_order"}, resp.Status, "CancelBatchOrders should report each order's state or error")

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: mockWrapperCancelOrderID}})
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, map[string]string{mockWrapperCancelOrderID: "untriggered"}, resp.Status, "CancelBatchOrders should report the cancelled order's state")
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders should reject a nil cancellation")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures, Type: order.Market})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "CancelAllOrders should reject market orders, which never rest")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		placed, err := e.SubmitOrder(t.Context(), newLiveWrapperOrder(t))
		require.NoError(t, err, "SubmitOrder must not error")
		resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{Pair: placed.Pair, AssetType: asset.Futures, Type: order.AnyType})
		require.NoError(t, err, "CancelAllOrders must not error")
		assert.Contains(t, resp.Status, placed.OrderID, "CancelAllOrders should cancel the pair's order")
		return
	}
	exp := map[string]string{"BTC_USDC-13133482": "cancelled"}
	resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{Pair: wrapperTestPair(t, asset.Futures), AssetType: asset.Futures, Type: order.AnyType})
	require.NoError(t, err, "CancelAllOrders must not error for a pair")
	assert.Equal(t, exp, resp.Status, "CancelAllOrders should report the pair's cancelled orders")
	resp, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Type: order.Limit})
	require.NoError(t, err, "CancelAllOrders must not error for an asset")
	assert.Equal(t, exp, resp.Status, "CancelAllOrders should report the asset's cancelled orders")
}

func TestCancelOrderTypeParam(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[order.Type]string{
		order.AnyType:          "all",
		order.UnknownType:      "all",
		order.Limit:            "limit",
		order.Stop:             "stop",
		order.StopLimit:        "stop",
		order.StopMarket:       "stop",
		order.TakeProfit:       "take",
		order.TakeProfitMarket: "take",
		order.TrailingStop:     "trailing_stop",
	} {
		param, err := cancelOrderTypeParam(orderType)
		require.NoErrorf(t, err, "cancelOrderTypeParam must not error for %s", orderType)
		assert.Equalf(t, exp, param, "cancelOrderTypeParam should cancel %s orders as %s", orderType, exp)
	}
	_, err := cancelOrderTypeParam(order.Market)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "cancelOrderTypeParam should reject market orders")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		placed, err := e.SubmitOrder(t.Context(), newLiveWrapperOrder(t))
		require.NoError(t, err, "SubmitOrder must not error")
		detail, err := e.GetOrderInfo(t.Context(), placed.OrderID, placed.Pair, asset.Futures)
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.Equal(t, placed.OrderID, detail.OrderID, "GetOrderInfo should return the order")
		return
	}
	ethPair := currency.NewPairWithDelimiter("ETH", perpString, currency.DashDelimiter)
	detail, err := e.GetOrderInfo(t.Context(), mockWrapperStateOrderID, ethPair, asset.Futures)
	require.NoError(t, err, "GetOrderInfo must not error")
	exp := &order.Detail{
		Exchange:             e.Name,
		OrderID:              mockWrapperStateOrderID,
		Pair:                 ethPair,
		AssetType:            asset.Futures,
		Side:                 order.Sell,
		Type:                 order.Limit,
		Status:               order.Filled,
		TimeInForce:          order.GoodTillCancel,
		Price:                118.94,
		Amount:               37,
		ExecutedAmount:       37,
		AverageExecutedPrice: 118.94,
		Date:                 time.UnixMilli(1550219749176),
		LastUpdated:          time.UnixMilli(1550219810944),
	}
	assert.Equal(t, exp, detail, "GetOrderInfo should convert the order")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	address, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
	require.NoError(t, err, "GetDepositAddress must not error")
	if mockTests {
		assert.Equal(t, &deposit.Address{Address: "2N8udZGBc1hLRCFsU9kGwMPpmYUwMFTuCwB"}, address, "GetDepositAddress should return the current address")
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	assert.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds should reject a nil request")
	if !mockTests {
		t.Skip("withdrawals move funds out of the account, so only the mock tests them")
	}
	resp, err := e.WithdrawCryptocurrencyFunds(t.Context(), mockWrapperWithdrawRequest())
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, &withdraw.ExchangeResponse{Name: e.Name, ID: "1", Status: "cancelled"}, resp, "WithdrawCryptocurrencyFunds should return the withdrawal")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders should reject a nil request")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		placed, err := e.SubmitOrder(t.Context(), newLiveWrapperOrder(t))
		require.NoError(t, err, "SubmitOrder must not error")
		for _, pairs := range []currency.Pairs{nil, {placed.Pair}} {
			orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType, Pairs: pairs})
			require.NoError(t, err, "GetActiveOrders must not error")
			assert.True(t, slices.ContainsFunc(orders, func(d order.Detail) bool { return d.OrderID == placed.OrderID }), "GetActiveOrders should return the open order")
		}
		return
	}
	orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for every pair")
	require.Len(t, orders, 1, "GetActiveOrders must return the mock open order")
	assert.Equal(t, "146062", orders[0].OrderID, "GetActiveOrders should convert the open order")

	labelled, err := currency.NewPairFromString(mockWrapperLabelInstrument)
	require.NoError(t, err, "NewPairFromString must not error")
	orders, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType, Pairs: currency.Pairs{labelled}})
	require.NoError(t, err, "GetActiveOrders must not error for a pair")
	assert.Len(t, orders, 1, "GetActiveOrders should return the pair's open order")
	orders, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.Sell, Type: order.AnyType, Pairs: currency.Pairs{wrapperTestPair(t, asset.Futures)}})
	require.NoError(t, err, "GetActiveOrders must not error for a filtered pair")
	assert.Empty(t, orders, "GetActiveOrders should filter out other pairs' and sides' orders")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetOrderHistory should require pairs")

	skipLiveWithoutCredentials(t)
	pair := wrapperTestPair(t, asset.Futures)
	orders, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType, Pairs: currency.Pairs{pair}})
	require.NoError(t, err, "GetOrderHistory must not error")
	if mockTests {
		require.Len(t, orders, 1, "GetOrderHistory must return the mock order")
		assert.Equal(t, "146475", orders[0].OrderID, "GetOrderHistory should convert the order")
		assert.Equal(t, order.Filled, orders[0].Status, "GetOrderHistory should convert the order's status")
	}
}

func TestFilteredOrders(t *testing.T) {
	t.Parallel()
	req := &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType}
	_, err := e.filteredOrders(req, []Order{{InstrumentName: "BTC-PERPETUAL", Direction: "up", OrderType: "limit", OrderState: "open", TimeInForce: "good_til_cancelled"}})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "filteredOrders should reject an order it cannot convert")
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails should reject spot")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.FutureCombo)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails should reject future combos")

	contracts, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
	pair := wrapperTestPair(t, asset.Futures)
	index := slices.IndexFunc(contracts, func(c futures.Contract) bool { return c.Name.Equal(pair) })
	require.NotEqual(t, -1, index, "GetFuturesContractDetails must return BTC-PERPETUAL")
	perpetual := contracts[index]
	assert.Equal(t, futures.Perpetual, perpetual.Type, "GetFuturesContractDetails should return a perpetual")
	assert.Equal(t, futures.Inverse, perpetual.SettlementType, "GetFuturesContractDetails should return an inverse contract")
	assert.Zero(t, perpetual.EndDate, "GetFuturesContractDetails should leave out a perpetual's expiry")
	if mockTests {
		exp := futures.Contract{
			Exchange:           e.Name,
			Name:               pair,
			Underlying:         currency.NewBTCUSD(),
			Asset:              asset.Futures,
			StartDate:          time.UnixMilli(1534242287000),
			IsActive:           true,
			Status:             "open",
			Type:               futures.Perpetual,
			SettlementType:     futures.Inverse,
			SettlementCurrency: currency.BTC,
			Multiplier:         10,
			MaxLeverage:        50,
		}
		assert.Equal(t, exp, perpetual, "GetFuturesContractDetails should convert the contract")
	}
}

func TestFuturesContract(t *testing.T) {
	t.Parallel()
	instrument := &Instrument{
		InstrumentName:      "ETH_USDC-6NOV26",
		InstrumentType:      "linear",
		SettlementPeriod:    "week",
		State:               "open",
		IsActive:            true,
		BaseCurrency:        currency.ETH,
		QuoteCurrency:       currency.USDC,
		SettlementCurrency:  currency.USDC,
		CreationTimestamp:   types.Time(time.UnixMilli(1791532800000)),
		ExpirationTimestamp: types.Time(time.UnixMilli(1793952000000)),
		ContractSize:        0.01,
		MaxLeverage:         25,
	}
	contract, err := e.futuresContract(instrument)
	require.NoError(t, err, "futuresContract must not error")
	exp := &futures.Contract{
		Exchange:           e.Name,
		Name:               currency.NewPairWithDelimiter("ETH", "USDC-6NOV26", currency.UnderscoreDelimiter),
		Underlying:         currency.NewPair(currency.ETH, currency.USDC),
		Asset:              asset.Futures,
		StartDate:          time.UnixMilli(1791532800000),
		EndDate:            time.UnixMilli(1793952000000),
		IsActive:           true,
		Status:             "open",
		Type:               futures.Weekly,
		SettlementType:     futures.Linear,
		SettlementCurrency: currency.USDC,
		Multiplier:         0.01,
		MaxLeverage:        25,
	}
	assert.Equal(t, exp, contract, "futuresContract should convert a linear weekly future")
	_, err = e.futuresContract(&Instrument{})
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "futuresContract should reject an instrument it cannot parse")
}

func TestFuturesContractType(t *testing.T) {
	t.Parallel()
	for period, exp := range map[string]futures.ContractType{
		"perpetual": futures.Perpetual,
		"day":       futures.Daily,
		"week":      futures.Weekly,
		"month":     futures.Monthly,
		"hour":      futures.Unknown,
	} {
		assert.Equalf(t, exp, futuresContractType(period), "futuresContractType should convert %s", period)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	assert.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Margin), asset.ErrNotSupported, "UpdateOrderExecutionLimits should reject an unsupported asset")
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Futures), "UpdateOrderExecutionLimits must not error")
	level, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, asset.Futures, wrapperTestPair(t, asset.Futures)))
	require.NoError(t, err, "GetOrderExecutionLimits must not error")
	assert.Equal(t, 0.5, level.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load the tick size")
	assert.Equal(t, 10.0, level.MinimumBaseAmount, "UpdateOrderExecutionLimits should load the minimum amount")
	assert.Equal(t, 10.0, level.AmountStepIncrementSize, "UpdateOrderExecutionLimits should load the amount increment")
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary should reject a nil request")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesPositionSummary should reject spot")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPositionSummary should reject an empty pair")

	skipLiveWithoutCredentials(t)
	pair := wrapperTestPair(t, asset.Futures)
	summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: pair})
	require.NoError(t, err, "GetFuturesPositionSummary must not error")
	if mockTests {
		exp := &futures.PositionSummary{
			Pair:                         pair,
			Asset:                        asset.Futures,
			Currency:                     currency.BTC,
			NotionalSize:                 decimal.MustFromFloat(0),
			Leverage:                     decimal.MustFromFloat(100),
			MaintenanceMarginRequirement: decimal.MustFromFloat(0),
			InitialMarginRequirement:     decimal.MustFromFloat(0),
			EstimatedLiquidationPrice:    decimal.MustFromFloat(0),
			MarkPrice:                    decimal.MustFromFloat(3556.62),
			CurrentSize:                  decimal.MustFromFloat(0),
			ContractSize:                 decimal.MustFromFloat(0),
			ContractMultiplier:           decimal.MustFromFloat(10),
			ContractSettlementType:       futures.Inverse,
			AverageOpenPrice:             decimal.MustFromFloat(0),
			UnrealisedPNL:                decimal.MustFromFloat(0),
			RealisedPNL:                  decimal.MustFromFloat(0),
		}
		assert.Equal(t, exp, summary, "GetFuturesPositionSummary should convert the position")
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetOpenInterest should require a pair")
	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDC.Item, Asset: asset.Spot})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest should reject spot")

	pair := wrapperTestPair(t, asset.Futures)
	resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: asset.Futures})
	require.NoError(t, err, "GetOpenInterest must not error")
	require.Len(t, resp, 1, "GetOpenInterest must return the pair's open interest")
	if mockTests {
		assert.Equal(t, 862167020.0, resp[0].OpenInterest, "GetOpenInterest should return the summary's open interest")
	} else {
		assert.Positive(t, resp[0].OpenInterest, "GetOpenInterest should return the pair's open interest")
	}
}

// wrapperFundingRecords is the mock hourly funding history of BTC-PERPETUAL, as Deribit sends it
const wrapperFundingRecords = `[{"index_price":81556.85,"interest_1h":1.3808122211552417e-05,"interest_8h":6.311465614099483e-05,"prev_index_price":81576.93,"timestamp":1792026000000},{"index_price":80827.82,"interest_1h":1.105698059002636e-05,"interest_8h":7.917005619894089e-05,"prev_index_price":81416.4,"timestamp":1792036800000}]`

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	pair := wrapperTestPair(t, asset.Futures)
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates should reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: currency.NewPairWithDelimiter("BTC", "27NOV26", currency.DashDelimiter)})
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates should reject a dated future")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":[]}`).GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: pair})
	assert.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetLatestFundingRates should report no funding records")

	ex := e
	if mockTests {
		// The request covers the hours before now, which a recording cannot match
		ex = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":`+wrapperFundingRecords+`}`)
	}
	rates, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: pair})
	require.NoError(t, err, "GetLatestFundingRates must not error")
	require.Len(t, rates, 1, "GetLatestFundingRates must return the pair's rate")
	assert.Equal(t, pair, rates[0].Pair, "GetLatestFundingRates should return the pair's rate")
	if mockTests {
		exp := fundingrate.Rate{Time: time.UnixMilli(1792036800000), Rate: decimal.MustFromFloat(7.917005619894089e-05)}
		assert.Equal(t, exp, rates[0].LatestRate, "GetLatestFundingRates should return the latest record's 8 hour rate")
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	pair := wrapperTestPair(t, asset.Futures)
	start, end := time.UnixMilli(marketDataRangeStart), time.UnixMilli(marketDataRangeEnd)
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates should reject a nil request")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: start, EndDate: end, IncludePayments: true})
	assert.ErrorIs(t, err, common.ErrNotYetImplemented, "GetHistoricalFundingRates should reject payments")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: end, EndDate: start})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricalFundingRates should reject a start after the end")

	if !mockTests {
		end = time.Now()
		start = end.Add(-48 * time.Hour)
	}
	rates, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: start, EndDate: end})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	require.NotEmpty(t, rates.FundingRates, "GetHistoricalFundingRates must return rates")
	assert.True(t, slices.IsSortedFunc(rates.FundingRates, func(a, b fundingrate.Rate) int { return a.Time.Compare(b.Time) }), "GetHistoricalFundingRates should return the rates oldest first")
	if mockTests {
		require.Len(t, rates.FundingRates, 4, "GetHistoricalFundingRates must return the mock hours")
		exp := fundingrate.Rate{Time: time.UnixMilli(1792026000000), Rate: decimal.MustFromFloat(1.3808122211552417e-05)}
		assert.Equal(t, exp, rates.FundingRates[0], "GetHistoricalFundingRates should return each hour's rate")
		assert.Equal(t, exp.Time, rates.StartDate, "GetHistoricalFundingRates should start at the first rate")
		assert.Equal(t, rates.FundingRates[3], rates.LatestRate, "GetHistoricalFundingRates should return the last rate as the latest")
	}
}

func TestGetHistoricalFundingRatesPaging(t *testing.T) {
	t.Parallel()
	const end = 1792036800000
	const hour = int64(time.Hour / time.Millisecond)
	records := func(last int64, count int) string {
		parts := make([]string, count)
		for i := range parts {
			parts[i] = `{"interest_1h":0.00001,"interest_8h":0.00008,"timestamp":` + strconv.FormatInt(last-int64(count-1-i)*hour, 10) + `}`
		}
		return `{"jsonrpc":"2.0","result":[` + strings.Join(parts, ",") + `]}`
	}
	var mu sync.Mutex
	var ends []int64
	ex := newWrapperServerExchange(t, func(_ string, params json.RawMessage) string {
		var req struct {
			EndTimestamp int64 `json:"end_timestamp"`
		}
		assert.NoError(t, json.Unmarshal(params, &req), "Decoding the params should not error")
		mu.Lock()
		ends = append(ends, req.EndTimestamp)
		mu.Unlock()
		if req.EndTimestamp == end {
			return records(end, maximumFundingRatePage)
		}
		return records(req.EndTimestamp+1, 2)
	})
	pair := wrapperTestPair(t, asset.Futures)
	rates, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: time.UnixMilli(end - 1000*hour), EndDate: time.UnixMilli(end)})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	assert.Len(t, rates.FundingRates, maximumFundingRatePage+2, "GetHistoricalFundingRates should combine the pages")
	firstOfLast := end - int64(maximumFundingRatePage-1)*hour
	assert.Equal(t, []int64{end, firstOfLast - 1}, ends, "GetHistoricalFundingRates should end each earlier page before the last page's first record")
	assert.True(t, slices.IsSortedFunc(rates.FundingRates, func(a, b fundingrate.Rate) int { return a.Time.Compare(b.Time) }), "GetHistoricalFundingRates should return the pages oldest first")
}

func TestPerpetualInstrument(t *testing.T) {
	t.Parallel()
	name, err := e.perpetualInstrument(asset.Futures, wrapperTestPair(t, asset.Futures))
	require.NoError(t, err, "perpetualInstrument must not error")
	assert.Equal(t, "BTC-PERPETUAL", name, "perpetualInstrument should return the perpetual's instrument name")
	_, err = e.perpetualInstrument(asset.Spot, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "perpetualInstrument should reject spot")
	_, err = e.perpetualInstrument(asset.Futures, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "perpetualInstrument should reject an empty pair")
}

func TestSetupWebsocketURL(t *testing.T) {
	t.Parallel()
	raw, err := testExchangeConfig()
	require.NoError(t, err, "testExchangeConfig must not error")
	var exchCfg config.Exchange
	require.NoError(t, json.Unmarshal(raw, &exchCfg), "Unmarshal must not error")
	exchCfg.API.Endpoints = map[string]string{exchange.WebsocketSpot.String(): testnetWebsocketURL}
	ex := new(Exchange)
	ex.SetDefaults()
	require.NoError(t, ex.Setup(&exchCfg), "Setup must not error")
	assert.Equal(t, testnetWebsocketURL, ex.Websocket.GetWebsocketURL(), "Setup should connect to the configured websocket URL")
	defaultURL, err := ex.API.Endpoints.GetURL(exchange.RestFutures)
	require.NoError(t, err, "GetURL must not error")
	assert.Equal(t, apiURL, defaultURL, "Setup should keep the default REST URL")
}
