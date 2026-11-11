package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	// testSpotPair, testPerpetualPair and testFuturesPair are the pairs of the recorded instruments, which hold no
	// other pair but SHIB_BTC@OEX_HK's, which no pair names
	testSpotPair      = currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter)
	testPerpetualPair = currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.DashDelimiter)
	testFuturesPair   = currency.NewPairWithDelimiter("BTCUSD", "261030", currency.DashDelimiter)
	// testInstrumentPairs are each asset's pairs of the recorded instruments
	testInstrumentPairs = map[asset.Item]currency.Pairs{
		asset.Spot:          {testSpotPair},
		asset.Margin:        {testSpotPair},
		asset.PerpetualSwap: {testPerpetualPair},
		asset.Futures:       {testFuturesPair},
	}
	// testWindowStart is the start of the windows the history tests request, 2026-10-24T00:00:00Z, a day before the
	// recordings were made, so the live venue serves the same windows; it is local, as the times the code decodes are
	testWindowStart = time.UnixMilli(1792800000000)
)

// liveTradablePairs fetches each asset's tradable pairs once, for the live tests
var liveTradablePairs = sync.OnceValues(func() (map[asset.Item]currency.Pairs, error) {
	pairs := make(map[asset.Item]currency.Pairs)
	for _, a := range e.GetAssetTypes(false) {
		p, err := e.FetchTradablePairs(context.Background(), a)
		if err != nil {
			return nil, err
		}
		pairs[a] = p
	}
	return pairs, nil
})

// testPair returns the pair a test of an asset uses: the recorded instruments' in mock mode, and live the asset's first
// enabled pair Crypto.com still trades, else, as the dated futures contract the recordings hold expires, the asset's
// first BTC pair, else its first pair
func testPair(t *testing.T, a asset.Item) currency.Pair {
	t.Helper()
	if mockTests {
		return testInstrumentPairs[a][0]
	}
	pairs, err := liveTradablePairs()
	require.NoError(t, err, "FetchTradablePairs must not error")
	require.NotEmptyf(t, pairs[a], "Crypto.com must trade %s pairs", a)
	enabled, err := e.GetEnabledPairs(a)
	require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
	for _, p := range enabled {
		if pairs[a].Contains(p, true) {
			return p
		}
	}
	for _, p := range pairs[a] {
		if p.Base.String() == "BTC" || p.Base.String() == "BTCUSD" {
			return p
		}
	}
	return pairs[a][0]
}

// numberDecimal returns a number's decimal as the code takes it from a types.Number, which decimals built otherwise
// may not equal in form
func numberDecimal(f float64) decimal.Decimal {
	return types.Number(f).Decimal()
}

func TestFetchTradablePairs(t *testing.T) { //nolint:tparallel // the cases share e and run in order
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must refuse an unsupported asset")
	// The cases share e, so they send it their requests in order
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			pairs, err := e.FetchTradablePairs(t.Context(), a)
			require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
			if mockTests {
				assert.Equalf(t, testInstrumentPairs[a], pairs, "FetchTradablePairs should return the pairs of the tradable %s instruments", a)
			} else {
				assert.NotEmptyf(t, pairs, "FetchTradablePairs should return %s pairs", a)
			}
		})
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	for _, a := range ex.GetAssetTypes(false) {
		pairs, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		if mockTests {
			assert.Equalf(t, testInstrumentPairs[a].Strings(), pairs.Strings(), "UpdateTradablePairs should store the pairs of the tradable %s instruments", a)
		} else {
			assert.NotEmptyf(t, pairs, "UpdateTradablePairs should store %s pairs", a)
		}
	}
}

func TestTradablePairs(t *testing.T) { //nolint:tparallel // the cases share instruments in order, before the later call of the same function
	t.Parallel()
	instruments := []InstrumentResponse{
		{Symbol: "BTC_USD", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.BTC, QuoteCurrency: currency.USD, Tradable: true, MarginBuyEnabled: true},
		{Symbol: "ETH_USD", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.ETH, QuoteCurrency: currency.USD, Tradable: true},
		{Symbol: "XRP_USD", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.XRP, QuoteCurrency: currency.USD, MarginSellEnabled: true},
		{Symbol: "SHIB_BTC@OEX_HK", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.SHIB, QuoteCurrency: currency.BTC, Tradable: true, MarginBuyEnabled: true},
		{Symbol: "BTCUSD-PERP", InstrumentType: InstrumentTypePerpetualSwap, BaseCurrency: currency.BTC, QuoteCurrency: currency.USD, Tradable: true},
		{Symbol: "BTCUSD-261030", InstrumentType: InstrumentTypeFuture, BaseCurrency: currency.BTC, QuoteCurrency: currency.USD, Tradable: true},
	}
	// The cases share instruments in order, before the malformed name below, so each also checks that a call keeps
	// nothing from the last
	for a, exp := range map[asset.Item]currency.Pairs{
		asset.Spot:          {testSpotPair, currency.NewPairWithDelimiter("ETH", "USD", currency.UnderscoreDelimiter)},
		asset.Margin:        {testSpotPair},
		asset.PerpetualSwap: {testPerpetualPair},
		asset.Futures:       {testFuturesPair},
		asset.Options:       {},
	} {
		t.Run(a.String(), func(t *testing.T) {
			pairs, err := tradablePairs(instruments, a)
			require.NoErrorf(t, err, "tradablePairs must not error for %s", a)
			assert.Equalf(t, exp, pairs, "tradablePairs should return the pairs of the tradable %s instruments a pair names", a)
		})
	}
	_, err := tradablePairs([]InstrumentResponse{{Symbol: "BTCUSDPERP", InstrumentType: InstrumentTypePerpetualSwap, Tradable: true}}, asset.PerpetualSwap)
	assert.ErrorIs(t, err, errMalformedInstrumentName, "tradablePairs should refuse a derivative named without a dash")
}

func TestPairFromInstrument(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		a    asset.Item
		name string
		exp  currency.Pair
	}{
		{a: asset.Spot, name: "BTC_USD", exp: testSpotPair},
		{a: asset.Margin, name: "BTC_USD", exp: testSpotPair},
		{a: asset.PerpetualSwap, name: "BTCUSD-PERP", exp: testPerpetualPair},
		{a: asset.Futures, name: "BTCUSD-261030", exp: testFuturesPair},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			p, err := pairFromInstrument(tc.a, tc.name)
			require.NoErrorf(t, err, "pairFromInstrument must not error for %s %s", tc.a, tc.name)
			assert.Equalf(t, tc.exp, p, "pairFromInstrument should split %s at its last delimiter", tc.name)
		})
	}
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, name := range []string{"BTCUSDPERP", "-PERP", "BTCUSD-"} {
		t.Run(name, func(t *testing.T) {
			_, err := pairFromInstrument(asset.PerpetualSwap, name)
			assert.ErrorIsf(t, err, errMalformedInstrumentName, "pairFromInstrument should refuse %q", name)
		})
	}
	_, err := pairFromInstrument(asset.Options, "BTC_USD")
	assert.ErrorIs(t, err, asset.ErrNotSupported, "pairFromInstrument should refuse an unsupported asset")
	_, err = e.instrumentName(asset.Spot, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "instrumentName should refuse an empty pair")
}

func TestOrderAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		execInst []string
		exp      asset.Item
	}{
		{name: "BTC_USD", exp: asset.Spot},
		{name: "BTC_USD", execInst: []string{ExecutionInstructionPostOnly, ExecutionInstructionMarginOrder}, exp: asset.Margin},
		{name: "BTCUSD-PERP", execInst: []string{ExecutionInstructionIsolatedMargin}, exp: asset.PerpetualSwap},
		{name: "BTCUSD-261030", exp: asset.Futures},
	} {
		t.Run(tc.exp.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, orderAsset(tc.name, tc.execInst), "orderAsset should place an order on %s with %v", tc.name, tc.execInst)
		})
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) { //nolint:tparallel // the cases share ex and run in order
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateOrderExecutionLimits must refuse an unsupported asset")
	// The cases share ex in order, so each also checks that an update keeps nothing from the last
	for _, a := range ex.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
			p := testPair(t, a)
			l, err := ex.GetOrderExecutionLimits(a, p)
			require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", a, p)
			if !mockTests {
				assert.Positivef(t, l.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load %s %s's price step", a, p)
				assert.Positivef(t, l.AmountStepIncrementSize, "UpdateOrderExecutionLimits should load %s %s's quantity step", a, p)
				return
			}
			exp := limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(ex.Name, a, p),
				PriceStepIncrementSize:  0.01,
				AmountStepIncrementSize: 0.00001,
				MinimumBaseAmount:       0.00001,
			}
			switch a {
			case asset.PerpetualSwap:
				exp.PriceStepIncrementSize, exp.AmountStepIncrementSize, exp.MinimumBaseAmount = 0.1, 0.0001, 0.0001
			case asset.Futures:
				exp.PriceStepIncrementSize, exp.AmountStepIncrementSize, exp.MinimumBaseAmount = 0.1, 0.0001, 0.0001
				exp.Expiry = time.UnixMilli(1793347200000)
			}
			assert.NotZerof(t, l.UpdatedAt, "UpdateOrderExecutionLimits should time %s %s's limits", a, p)
			l.UpdatedAt = time.Time{}
			assert.Equalf(t, exp, l, "UpdateOrderExecutionLimits should load %s %s's steps", a, p)
		})
	}
}

func TestUpdateTickers(t *testing.T) { //nolint:tparallel // the cases share ex and run in order before the checks of what they stored
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers must refuse an unsupported asset")
	achUSD := currency.NewPairWithDelimiter("ACH", "USD", currency.UnderscoreDelimiter)
	shazPerpetual := currency.NewPairWithDelimiter("SHAZUSD", "PERP", currency.DashDelimiter)
	// The recorded tickers of every instrument hold ACH_USD's and SHAZUSD-PERP's, so those pairs are the ones enabled; live,
	// each asset's is one Crypto.com still trades, as it sends no ticker for a futures contract once that expires
	enabled := map[asset.Item]currency.Pair{asset.Spot: achUSD, asset.PerpetualSwap: shazPerpetual}
	if !mockTests {
		for _, a := range ex.GetAssetTypes(true) {
			enabled[a] = testPair(t, a)
		}
	}
	for a, p := range enabled {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must store %s's available pair", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must store %s's enabled pair", a)
	}
	// The cases share ex, whose stored tickers the checks below read, so they run in order before them
	for _, a := range ex.GetAssetTypes(true) {
		t.Run(a.String(), func(t *testing.T) {
			require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
		})
	}
	if !mockTests {
		for _, a := range ex.GetAssetTypes(true) {
			pairs, err := ex.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
			_, err = ticker.GetTicker(ex.Name, pairs[0], a)
			assert.NoErrorf(t, err, "UpdateTickers should store %s %s's ticker", a, pairs[0])
		}
		return
	}
	got, err := ticker.GetTicker(ex.Name, achUSD, asset.Spot)
	require.NoError(t, err, "UpdateTickers must store ACH_USD's ticker")
	exp := &ticker.Price{
		Last:                0.005159,
		High:                0.005229,
		Low:                 0.004969,
		Bid:                 0.005159,
		Ask:                 0.005179,
		BaseVolume:          479210,
		QuoteVolume:         2470.19,
		PercentChange24Hour: -0.79,
		Pair:                achUSD,
		ExchangeName:        ex.Name,
		AssetType:           asset.Spot,
		LastUpdated:         time.UnixMilli(1792893939837),
	}
	assert.Equal(t, exp, got, "UpdateTickers should store every field of a spot ticker, with its USD volume as the quote volume")
	got, err = ticker.GetTicker(ex.Name, shazPerpetual, asset.PerpetualSwap)
	require.NoError(t, err, "UpdateTickers must store SHAZUSD-PERP's ticker")
	exp = &ticker.Price{
		Last:                22.44,
		High:                23.09,
		Low:                 22.38,
		Bid:                 22.76,
		Ask:                 22.8,
		BaseVolume:          16.48,
		QuoteVolume:         371.92,
		PercentChange24Hour: -2.82,
		OpenInterest:        5.6,
		Pair:                shazPerpetual,
		ExchangeName:        ex.Name,
		AssetType:           asset.PerpetualSwap,
		LastUpdated:         time.UnixMilli(1792893900501),
	}
	assert.Equal(t, exp, got, "UpdateTickers should store every field of a perpetual swap's ticker")
}

func TestUpdateTicker(t *testing.T) { //nolint:tparallel // the cases share ex and run in order
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must refuse an empty pair")
	// The cases share ex in order, so each also checks that an update keeps nothing from the last
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap} {
		t.Run(a.String(), func(t *testing.T) {
			p := testPair(t, a)
			got, err := ex.UpdateTicker(t.Context(), p, a)
			require.NoErrorf(t, err, "UpdateTicker must not error for %s %s", a, p)
			if !mockTests {
				assert.Positivef(t, got.Last, "UpdateTicker should return %s %s's last price", a, p)
				return
			}
			exp := &ticker.Price{
				Last:                80815.22,
				High:                81007.04,
				Low:                 79128.21,
				Bid:                 80816.57,
				Ask:                 80816.58,
				BaseVolume:          2201.6316,
				QuoteVolume:         176643650.4,
				PercentChange24Hour: -0.02,
				Pair:                p,
				ExchangeName:        ex.Name,
				AssetType:           a,
				LastUpdated:         time.UnixMilli(1792893939611),
			}
			if a == asset.PerpetualSwap {
				exp = &ticker.Price{
					Last:                80816.4,
					High:                81004.7,
					Low:                 79117.7,
					Bid:                 80818.1,
					Ask:                 80818.2,
					BaseVolume:          2497.9661,
					QuoteVolume:         200360781.94,
					PercentChange24Hour: -0.01,
					OpenInterest:        6180.2299,
					Pair:                p,
					ExchangeName:        ex.Name,
					AssetType:           a,
					LastUpdated:         time.UnixMilli(1792893941098),
				}
			}
			assert.Equalf(t, exp, got, "UpdateTicker should return every field of %s %s's ticker", a, p)
		})
	}
}

func TestTickerPrice(t *testing.T) {
	t.Parallel()
	ethBTC := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	got := e.tickerPrice(&TickerResponse{InstrumentName: "ETH_BTC", LastPrice: 0.0325, Volume24Hour: 12, USDVolume24Hour: 31000}, asset.Spot, ethBTC)
	exp := &ticker.Price{Last: 0.0325, BaseVolume: 12, Pair: ethBTC, ExchangeName: e.Name, AssetType: asset.Spot}
	assert.Equal(t, exp, got, "tickerPrice should leave out the USD volume of a pair not quoted in USD")

	t.Run("every field of a pair quoted in USD", func(t *testing.T) {
		t.Parallel()
		updated := time.UnixMilli(1792836351902)
		resp := &TickerResponse{InstrumentName: "BTC_USD", High24Hour: 81000, Low24Hour: 79000, LastPrice: 80000, Volume24Hour: 12, USDVolume24Hour: 960000, PriceChange24Hour: -0.0078, BestBidPrice: 79999.5, BestAskPrice: 80000.5, OpenInterest: 5, Timestamp: types.Time(updated)}
		exp := &ticker.Price{Last: 80000, High: 81000, Low: 79000, Bid: 79999.5, Ask: 80000.5, BaseVolume: 12, QuoteVolume: 960000, PercentChange24Hour: -0.78, OpenInterest: 5, Pair: testSpotPair, ExchangeName: e.Name, AssetType: asset.Spot, LastUpdated: updated}
		assert.Equal(t, exp, e.tickerPrice(resp, asset.Spot, testSpotPair), "tickerPrice should map every field, the USD volume as a USD pair's quote volume")
	})
}

func TestPercentFromFraction(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		fraction types.Number
		exp      float64
	}{
		{-0.0078, -0.78},
		{-0.0156, -1.56},
		{0, 0},
		{types.Number(math.Inf(1)), math.Inf(1)},
		{types.Number(math.Inf(-1)), math.Inf(-1)},
	} {
		t.Run(strconv.FormatFloat(tc.fraction.Float64(), 'g', -1, 64), func(t *testing.T) {
			assert.Equalf(t, tc.exp, percentFromFraction(tc.fraction), "percentFromFraction should return %s as a percentage", tc.fraction)
		})
	}
	var fraction types.Number
	require.NoError(t, json.Unmarshal([]byte(`"NaN"`), &fraction), "Unmarshal must not error")
	assert.True(t, math.IsNaN(percentFromFraction(fraction)), "percentFromFraction should return NaN for a fraction that is not a number")
}

func TestUpdateOrderbook(t *testing.T) { //nolint:tparallel // the cases share ex and run in order
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must refuse an empty pair")
	// The cases share ex in order, before the call below, so each also checks that an update keeps nothing from the last
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualSwap} {
		t.Run(a.String(), func(t *testing.T) {
			p := testPair(t, a)
			got, err := ex.UpdateOrderbook(t.Context(), p, a)
			require.NoErrorf(t, err, "UpdateOrderbook must not error for %s %s", a, p)
			if !mockTests {
				assert.NotEmptyf(t, got.Bids, "UpdateOrderbook should return %s %s's bids", a, p)
				assert.NotEmptyf(t, got.Asks, "UpdateOrderbook should return %s %s's asks", a, p)
				return
			}
			exp := testOrderbooks[a]
			exp.Exchange, exp.Pair, exp.Asset, exp.ValidateOrderbook, exp.DeferToWebsocketWithin = ex.Name, p, a, ex.ValidateOrderbook, 10*time.Second
			assert.Equalf(t, &exp, got, "UpdateOrderbook should return %s %s's REST snapshot", a, p)
			stored, err := orderbook.Get(ex.Name, p, a)
			require.NoErrorf(t, err, "orderbook.Get must return %s %s's stored book", a, p)
			assert.NotZerof(t, stored.InsertedAt, "UpdateOrderbook should time when %s %s's book was stored", a, p)
			stored.InsertedAt = time.Time{}
			exp.DeferToWebsocketWithin = 0
			assert.Equalf(t, &exp, stored, "UpdateOrderbook should store %s %s's REST snapshot without a websocket book", a, p)
		})
	}

	empty := newResponseServerExchange(t, `{"id":1,"method":"public/get-book","code":0,"result":{"depth":150,"instrument_name":"BTC_USD","data":[]}}`)
	_, err = empty.UpdateOrderbook(t.Context(), testSpotPair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookNotFound, "UpdateOrderbook should error for a response holding no book")
}

// testOrderbooks are the books recorded for each asset, as UpdateOrderbook stores them
var testOrderbooks = map[asset.Item]orderbook.Book{
	asset.Spot: {
		Bids: orderbook.Levels{
			{Price: 80848.56, Amount: 0.24979, OrderCount: 8},
			{Price: 80847.76, Amount: 0.15236, OrderCount: 2},
			{Price: 80847.75, Amount: 0.21313, OrderCount: 2},
		},
		Asks: orderbook.Levels{
			{Price: 80848.57, Amount: 1.0302, OrderCount: 12},
			{Price: 80849.33, Amount: 0.0025, OrderCount: 1},
			{Price: 80849.37, Amount: 0.18944, OrderCount: 1},
		},
		LastUpdated:  time.UnixMilli(1792899076792),
		RestSnapshot: true,
	},
	asset.PerpetualSwap: {
		Bids: orderbook.Levels{
			{Price: 80846.3, Amount: 0.1601, OrderCount: 5},
			{Price: 80846.2, Amount: 0.0012, OrderCount: 1},
			{Price: 80845.1, Amount: 0.03, OrderCount: 1},
		},
		Asks: orderbook.Levels{
			{Price: 80846.4, Amount: 0.0463, OrderCount: 6},
			{Price: 80847.4, Amount: 0.0876, OrderCount: 1},
			{Price: 80849.4, Amount: 0.0025, OrderCount: 1},
		},
		LastUpdated:  time.UnixMilli(1792899077501),
		RestSnapshot: true,
	},
}

// walletLockProbe records, each time the account store looks up the credentials it saves under, whether the wallet
// lock is held
type walletLockProbe struct {
	*Exchange
	lockHeld []bool
}

func (p *walletLockProbe) GetCredentials(ctx context.Context) (*accounts.Credentials, error) {
	held := !p.walletMu.TryLock()
	if !held {
		p.walletMu.Unlock()
	}
	p.lockHeld = append(p.lockHeld, held)
	return p.Exchange.GetCredentials(ctx)
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must refuse an unsupported asset")
	_, err = newUnavailableServerExchange(t).UpdateAccountBalances(t.Context(), asset.Spot)
	require.ErrorIs(t, err, request.ErrBadStatus, "UpdateAccountBalances must return the request's failure")
	older := newResponseServerExchange(t, `{"id":1,"code":0,"result":{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1","update_timestamp_ms":1607063412000}]}]}}`)
	newer := accounts.NewSubAccount(asset.Spot, "")
	newer.Balances.Set(currency.BTC, accounts.Balance{Total: 2, Free: 2, UpdatedAt: time.UnixMilli(1607063412001)})
	require.NoError(t, older.Accounts.Save(t.Context(), accounts.SubAccounts{newer}, false), "Save must not error")
	refused, err := older.UpdateAccountBalances(t.Context(), asset.Margin)
	require.ErrorContains(t, err, "out of sequence", "UpdateAccountBalances must return the store's refusal of an older reply")
	assert.Nil(t, refused, "UpdateAccountBalances should return no wallet the store refused")

	t.Run("credentials refused before any request", func(t *testing.T) {
		t.Parallel()
		unauthenticated := newTestExchange(t)
		setTestCredentials(unauthenticated, &accounts.Credentials{})
		unauthenticated.SkipAuthCheck = false
		_, err := unauthenticated.UpdateAccountBalances(t.Context(), asset.Spot)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "UpdateAccountBalances should refuse empty credentials")
	})
	t.Run("in-flight REST snapshot preserves newer websocket balances", func(t *testing.T) {
		t.Parallel()
		var exch *Exchange
		exch, received := newRoutesServerExchange(t, map[string]testRoute{
			"private/user-balance": func(string) (string, *APIError) {
				push := &websocketChannelPush{Data: []byte(`[{"position_balances":[{"instrument_name":"BTC","quantity":"2"},{"instrument_name":"ETH","quantity":"3"}]}]`)}
				assert.NoError(t, exch.wsHandleUserBalance(t.Context(), push), "the newer balance should be saved")
				return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}`, nil
			},
		})
		got, err := exch.UpdateAccountBalances(t.Context(), asset.PerpetualSwap)
		require.NoError(t, err, "UpdateAccountBalances must return the stored wallet")
		require.Len(t, received(), 1, "the REST request must reach the server")
		require.Len(t, got, 1, "UpdateAccountBalances must return the one wallet")
		assert.Equal(t, asset.PerpetualSwap, got[0].AssetType, "the stored wallet should be labelled for the asset that asked")
		assert.Equal(t, 2.0, got[0].Balances[currency.BTC].Total, "the returned BTC balance should be the newer one")
		assert.Equal(t, 3.0, got[0].Balances[currency.ETH].Total, "the returned ETH balance should be the newer one")
		balances, err := exch.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "cached balances must be available")
		assert.Equal(t, 2.0, balances[currency.BTC].Total, "the newer BTC balance should survive")
		assert.Equal(t, 3.0, balances[currency.ETH].Total, "the newer ETH balance should survive")
	})
	t.Run("a push for other credentials leaves the snapshot to be saved", func(t *testing.T) {
		t.Parallel()
		var exch *Exchange
		exch, _ = newRoutesServerExchange(t, map[string]testRoute{
			"private/user-balance": func(string) (string, *APIError) {
				ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: "other-secret"})
				push := &websocketChannelPush{Data: []byte(`[{"position_balances":[{"instrument_name":"ETH","quantity":"3"}]}]`)}
				assert.NoError(t, exch.wsHandleUserBalance(ctx, push), "the other credentials' balance should be saved")
				return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}`, nil
			},
		})
		got, err := exch.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		require.Len(t, got, 1, "UpdateAccountBalances must return the one wallet")
		assert.Equal(t, 1.0, got[0].Balances[currency.BTC].Total, "the snapshot should be returned")
		balances, err := exch.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "cached balances must be available")
		assert.Equal(t, 1.0, balances[currency.BTC].Total, "the snapshot should be saved")
		assert.NotContains(t, balances, currency.ETH, "the other credentials' push should not reach this wallet")
	})
	t.Run("a push whose save fails still holds the snapshot back", func(t *testing.T) {
		t.Parallel()
		var exch *Exchange
		var pushes atomic.Int64
		exch, _ = newRoutesServerExchange(t, map[string]testRoute{
			"private/user-balance": func(string) (string, *APIError) {
				// A balance timed before the stored one fails the save, which still saves the balances beside it
				push := &websocketChannelPush{Data: []byte(`[{"position_balances":[{"instrument_name":"BTC","quantity":"5","update_timestamp_ms":1607063412000},{"instrument_name":"ETH","quantity":"3"}]}]`)}
				assert.ErrorContains(t, exch.wsHandleUserBalance(t.Context(), push), "out of sequence", "the push's older BTC balance should be refused")
				pushes.Add(1)
				return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}`, nil
			},
		})
		stored := &websocketChannelPush{Data: []byte(`[{"position_balances":[{"instrument_name":"BTC","quantity":"2","update_timestamp_ms":1607063412001}]}]`)}
		require.NoError(t, exch.wsHandleUserBalance(t.Context(), stored), "the stored balance must be saved")
		_, err := exch.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must return the stored wallet")
		require.Equal(t, int64(1), pushes.Load(), "the push must arrive during the request")
		balances, err := exch.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "cached balances must be available")
		assert.Equal(t, 2.0, balances[currency.BTC].Total, "the stored BTC balance should survive")
		assert.Equal(t, 3.0, balances[currency.ETH].Total, "the ETH balance the failed save stored should survive")
	})
	t.Run("a snapshot whose save failed still holds an older one back", func(t *testing.T) {
		t.Parallel()
		var exch *Exchange
		var requests atomic.Int64
		exch, _ = newRoutesServerExchange(t, map[string]testRoute{
			"private/user-balance": func(string) (string, *APIError) {
				switch requests.Add(1) {
				case 1:
					_, err := exch.UpdateAccountBalances(t.Context(), asset.Spot)
					assert.ErrorContains(t, err, "out of sequence", "the newer snapshot's older BTC balance should be refused")
					return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}`, nil
				default:
					// A balance timed before the stored one fails the save, which still saves the balances beside it
					return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"5","update_timestamp_ms":1607063412000},{"instrument_name":"ETH","quantity":"3"}]}]}`, nil
				}
			},
		})
		stored := &websocketChannelPush{Data: []byte(`[{"position_balances":[{"instrument_name":"BTC","quantity":"2","update_timestamp_ms":1607063412001}]}]`)}
		require.NoError(t, exch.wsHandleUserBalance(t.Context(), stored), "the stored balance must be saved")
		_, err := exch.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must return the stored wallet")
		require.Equal(t, int64(2), requests.Load(), "the newer snapshot must be requested during the older one")
		balances, err := exch.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "cached balances must be available")
		assert.Equal(t, 2.0, balances[currency.BTC].Total, "the stored BTC balance should survive")
		assert.Equal(t, 3.0, balances[currency.ETH].Total, "the ETH balance the failed save stored should survive")
	})
	t.Run("a snapshot requested later but read earlier is not saved over the one answered first", func(t *testing.T) {
		t.Parallel()
		earlierReached, laterRead, earlierSaved := make(chan struct{}), make(chan struct{}), make(chan struct{})
		var requests atomic.Int64
		exch, _ := newRoutesServerExchange(t, map[string]testRoute{
			"private/user-balance": func(string) (string, *APIError) {
				if requests.Add(1) == 1 {
					close(earlierReached)
					<-laterRead
					return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"2"},{"instrument_name":"ETH","quantity":"3"}]}]}`, nil
				}
				close(laterRead)
				<-earlierSaved
				return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}`, nil
			},
		})
		earlierErr := make(chan error, 1)
		go func() {
			_, err := exch.UpdateAccountBalances(t.Context(), asset.Spot)
			earlierErr <- err
			close(earlierSaved)
		}()
		<-earlierReached
		got, err := exch.UpdateAccountBalances(t.Context(), asset.Margin)
		require.NoError(t, <-earlierErr, "the snapshot answered first must be saved")
		require.NoError(t, err, "UpdateAccountBalances must return the stored wallet")
		require.Len(t, got, 1, "UpdateAccountBalances must return the one wallet")
		assert.Equal(t, asset.Margin, got[0].AssetType, "the stored wallet should be labelled for the asset that asked")
		assert.Equal(t, 2.0, got[0].Balances[currency.BTC].Total, "the stored BTC balance should be returned")
		assert.Equal(t, 3.0, got[0].Balances[currency.ETH].Total, "the stored ETH balance should be returned")
		balances, err := exch.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "cached balances must be available")
		assert.Equal(t, 2.0, balances[currency.BTC].Total, "the stored BTC balance should survive")
		assert.Equal(t, 3.0, balances[currency.ETH].Total, "the stored ETH balance should survive")
	})
	t.Run("a snapshot requested before a newer one completed is not saved over it", func(t *testing.T) {
		t.Parallel()
		// Every asset's snapshot saves the one wallet, so a newer one requested for any asset holds the older back
		for _, newer := range []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures} {
			t.Run(newer.String(), func(t *testing.T) {
				t.Parallel()
				var exch *Exchange
				var requests atomic.Int64
				exch, _ = newRoutesServerExchange(t, map[string]testRoute{
					"private/user-balance": func(string) (string, *APIError) {
						if requests.Add(1) == 1 {
							_, err := exch.UpdateAccountBalances(t.Context(), newer)
							assert.NoErrorf(t, err, "the newer snapshot requested for %s should be saved", newer)
							return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}`, nil
						}
						return `{"data":[{"position_balances":[{"instrument_name":"ETH","quantity":"3"}]}]}`, nil
					},
				})
				got, err := exch.UpdateAccountBalances(t.Context(), asset.Spot)
				require.NoError(t, err, "UpdateAccountBalances must return the stored wallet")
				require.Len(t, got, 1, "UpdateAccountBalances must return the one wallet")
				assert.Equal(t, 3.0, got[0].Balances[currency.ETH].Total, "the newer snapshot should be returned")
				assert.NotContains(t, got[0].Balances, currency.BTC, "the older snapshot should not be returned")
				balances, err := exch.GetCachedCurrencyBalances(t.Context(), asset.Spot)
				require.NoError(t, err, "cached balances must be available")
				assert.Equal(t, 3.0, balances[currency.ETH].Total, "the newer snapshot should stay saved")
				assert.NotContains(t, balances, currency.BTC, "the older snapshot should not be saved")
			})
		}
	})
	t.Run("a snapshot holds the wallet lock while it is saved", func(t *testing.T) {
		t.Parallel()
		exch, _ := newRoutesServerExchange(t, map[string]testRoute{
			"private/user-balance": testResult(`{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}`),
		})
		probe := &walletLockProbe{Exchange: exch}
		exch.Accounts = accounts.MustNewAccounts(probe)
		_, err := exch.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must save the snapshot")
		assert.Equal(t, []bool{true}, probe.lockHeld, "the wallet lock should be held while the snapshot is saved, so that no push is saved between its check and its save")
	})
	skipLiveWithoutCredentials(t)
	// The cases share ex and its account store in order, so each also checks that an update keeps nothing from the last
	for _, a := range ex.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			got, err := ex.UpdateAccountBalances(t.Context(), a)
			require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
			require.Lenf(t, got, 1, "UpdateAccountBalances must return the one account for %s", a)
			if !mockTests {
				return
			}
			// Crypto.com does not time the balances, so saving them times them
			for c, b := range got[0].Balances {
				assert.NotZerof(t, b.UpdatedAt, "UpdateAccountBalances should time %s's %s balance", a, c)
				b.UpdatedAt = time.Time{}
				got[0].Balances[c] = b
			}
			exp := accounts.NewSubAccount(a, "")
			exp.Balances.Set(currency.CRO, accounts.Balance{Total: 24422.72427884, Hold: 22.72427884, Free: 24400})
			exp.Balances.Set(currency.USD, accounts.Balance{Total: 3113.50747209, Hold: 12.5, Free: 3101.00747209})
			assert.Equalf(t, accounts.SubAccounts{exp}, got, "UpdateAccountBalances should return each position balance for %s, its free quantity in decimal arithmetic", a)
		})
	}
	if !mockTests {
		return
	}
	// The balances are saved as a snapshot, so a balance the response no longer holds is removed
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	// The cases each save a stale balance to the one wallet of ex and refresh it, so they run in order
	for _, a := range ex.GetAssetTypes(false) {
		t.Run("removes stale balance for "+a.String(), func(t *testing.T) {
			stale := accounts.NewSubAccount(asset.Spot, "")
			stale.Balances.Set(currency.BTC, accounts.Balance{Total: 1, Free: 1})
			require.NoErrorf(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stale}, false), "Save must not error for %s", a)
			_, err := ex.UpdateAccountBalances(t.Context(), a)
			require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
			stored, err := ex.Accounts.CurrencyBalances(creds, asset.Spot)
			require.NoErrorf(t, err, "CurrencyBalances must not error for %s", a)
			assert.NotContainsf(t, stored, currency.BTC, "UpdateAccountBalances should remove a %s balance the response no longer holds", a)
		})
	}
}

func TestUpdateAccountBalancesPublishesUnderSpot(t *testing.T) {
	// Not parallel: while a subscriber holds the global dispatcher open, every test's ticker and book publishes queue on it,
	// and a full queue returns them an error
	require.NoError(t, dispatch.EnsureRunning(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit), "dispatch.EnsureRunning must not error")
	ex := newResponseServerExchange(t, `{"id":1,"code":0,"result":{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"1"}]}]}}`)
	pipe, err := ex.SubscribeAccountBalances()
	require.NoError(t, err, "SubscribeAccountBalances must not error")
	t.Cleanup(func() { assert.NoError(t, pipe.Release(), "Release should not error") })
	got, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, got, 1, "UpdateAccountBalances must return the one wallet")
	select {
	case published := <-pipe.Channel():
		wallet, ok := published.(*accounts.SubAccount)
		require.True(t, ok, "the store must publish a sub account")
		assert.Equal(t, asset.Spot, wallet.AssetType, "the store should publish the wallet under spot, whichever asset asks")
		assert.Equal(t, asset.Margin, got[0].AssetType, "UpdateAccountBalances should still label its copy for the asset that asked")
	case <-time.After(time.Minute):
		require.Fail(t, "the store must publish the wallet it saves")
	}
}

// TestAccountWallet holds every asset to the one wallet Crypto.com keeps: whichever asset refreshes it, each asset reads
// the wallet as the last refresh or push left it, and the store counts it once
func TestAccountWallet(t *testing.T) { //nolint:tparallel // the cases share ex and its one wallet and run in order
	t.Parallel()
	var refreshes atomic.Int64
	assets := asset.Items{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures}
	ex, _ := newRoutesServerExchange(t, map[string]testRoute{"private/user-balance": func(string) (string, *APIError) {
		switch n := refreshes.Add(1); {
		case n <= int64(len(assets)):
			return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"10","reserved_qty":"2"},{"instrument_name":"ETH","quantity":"3"}]}]}`, nil
		case n == int64(len(assets))+1:
			return `{"data":[{"position_balances":[{"instrument_name":"BTC","quantity":"5","reserved_qty":"1"}]}]}`, nil
		default:
			return `{"data":[{"position_balances":[]}]}`, nil
		}
	}})
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret, SubAccount: "sub-42"})
	requireWallet := func(step string, exp map[currency.Code]accounts.Balance) {
		t.Helper()
		for _, a := range append(assets, asset.All) {
			subAccts, err := ex.GetCachedSubAccounts(ctx, a)
			require.NoErrorf(t, err, "GetCachedSubAccounts must not error for %s %s", a, step)
			require.Lenf(t, subAccts, 1, "GetCachedSubAccounts must return the one wallet for %s %s", a, step)
			expAsset := a
			if a == asset.All {
				expAsset = asset.Spot
			}
			assert.Equalf(t, expAsset, subAccts[0].AssetType, "GetCachedSubAccounts should label the wallet for %s %s", a, step)
			currencies, err := ex.GetCachedCurrencyBalances(ctx, a)
			require.NoErrorf(t, err, "GetCachedCurrencyBalances must not error for %s %s", a, step)
			require.Lenf(t, subAccts[0].Balances, len(exp), "GetCachedSubAccounts must hold the wallet's currencies for %s %s", a, step)
			require.Lenf(t, currencies, len(exp), "GetCachedCurrencyBalances must hold the wallet's currencies for %s %s", a, step)
			for c, want := range exp {
				for name, got := range map[string]accounts.Balance{"GetCachedSubAccounts": subAccts[0].Balances[c], "GetCachedCurrencyBalances": currencies[c]} {
					assert.Equalf(t, want.Total, got.Total, "%s should hold %s's total for %s %s", name, c, a, step)
					assert.Equalf(t, want.Hold, got.Hold, "%s should hold %s's hold for %s %s", name, c, a, step)
					assert.Equalf(t, want.Free, got.Free, "%s should hold %s's free balance for %s %s", name, c, a, step)
				}
			}
		}
		stored, err := ex.Accounts.CurrencyBalances(nil, asset.All)
		require.NoErrorf(t, err, "CurrencyBalances must not error %s", step)
		require.Lenf(t, stored, len(exp), "the store must hold the wallet's currencies %s", step)
		for c, want := range exp {
			assert.Equalf(t, want.Total, stored[c].Total, "the store should count %s once %s", c, step)
		}
	}
	// The cases share ex and the route's count of refreshes, which the wallet checks below depend on, so they run in order
	for _, a := range assets {
		t.Run(a.String(), func(t *testing.T) {
			got, err := ex.UpdateAccountBalances(ctx, a)
			require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
			require.Lenf(t, got, 1, "UpdateAccountBalances must return the one wallet for %s", a)
			assert.Equalf(t, a, got[0].AssetType, "UpdateAccountBalances should label the wallet for %s", a)
		})
	}
	requireWallet("after a refresh of each asset", map[currency.Code]accounts.Balance{
		currency.BTC: {Total: 10, Hold: 2, Free: 8},
		currency.ETH: {Total: 3, Free: 3},
	})
	require.NoError(t, ex.wsHandleUserBalance(ctx, &websocketChannelPush{Data: []byte(`[{"position_balances":[{"instrument_name":"BTC","quantity":"7","reserved_qty":"1"}]}]`)}), "wsHandleUserBalance must not error")
	requireWallet("after a push", map[currency.Code]accounts.Balance{
		currency.BTC: {Total: 7, Hold: 1, Free: 6},
		currency.ETH: {Total: 3, Free: 3},
	})
	_, err := ex.UpdateAccountBalances(ctx, asset.PerpetualSwap)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	requireWallet("after a refresh that no longer holds a currency", map[currency.Code]accounts.Balance{
		currency.BTC: {Total: 5, Hold: 1, Free: 4},
	})
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	requireWallet("with an asset disabled", map[currency.Code]accounts.Balance{
		currency.BTC: {Total: 5, Hold: 1, Free: 4},
	})

	_, err = ex.GetCachedSubAccounts(ctx, asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCachedSubAccounts should refuse an unsupported asset")
	_, err = ex.GetCachedCurrencyBalances(ctx, asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCachedCurrencyBalances should refuse an unsupported asset")

	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret, SubAccount: "sub-43"})
	_, err = ex.GetCachedSubAccounts(other, asset.Margin)
	assert.ErrorIs(t, err, accounts.ErrNoSubAccounts, "GetCachedSubAccounts should hold other credentials' wallet apart")
	_, err = ex.GetCachedCurrencyBalances(other, asset.Margin)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "GetCachedCurrencyBalances should hold other credentials' wallet apart")

	_, err = ex.UpdateAccountBalances(ctx, asset.Margin)
	require.NoError(t, err, "UpdateAccountBalances must not error for an empty wallet")
	emptied, err := ex.GetCachedSubAccounts(ctx, asset.Spot)
	require.NoError(t, err, "GetCachedSubAccounts must not error for an empty wallet")
	require.Len(t, emptied, 1, "GetCachedSubAccounts must return the one wallet when it is empty")
	assert.Empty(t, emptied[0].Balances, "GetCachedSubAccounts should hold nothing of a wallet refreshed empty")
	_, err = ex.GetCachedCurrencyBalances(ctx, asset.Spot)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "GetCachedCurrencyBalances should hold nothing of a wallet refreshed empty")
}

func TestBalanceSubAccount(t *testing.T) {
	t.Parallel()
	got := balanceSubAccount(asset.Margin, []UserBalanceResponse{{PositionBalances: []CollateralBalance{
		{InstrumentName: "USD", Quantity: preciseNumber("-120.5"), UpdateTime: types.Time(time.UnixMilli(1792894000000))},
		{InstrumentName: "BTC", Quantity: preciseNumber("0.5"), ReservedQuantity: preciseNumber("0.6")},
	}}})
	exp := accounts.NewSubAccount(asset.Margin, "")
	exp.Balances.Set(currency.USD, accounts.Balance{Total: -120.5, Borrowed: 120.5, UpdatedAt: time.UnixMilli(1792894000000)})
	exp.Balances.Set(currency.BTC, accounts.Balance{Total: 0.5, Hold: 0.6})
	assert.Equal(t, exp, got, "balanceSubAccount should hold a negative quantity as borrowed, and no free balance below zero")

	got = balanceSubAccount(asset.Spot, []UserBalanceResponse{{
		PositionBalances:  []CollateralBalance{{InstrumentName: "USD", Quantity: preciseNumber("100"), ReservedQuantity: preciseNumber("10")}},
		IsolatedPositions: []IsolatedPositionBalance{{IsolationID: "19848526", PositionBalances: []CollateralBalance{{InstrumentName: "USD", Quantity: preciseNumber("8.9632670590686")}}}},
	}})
	exp = accounts.NewSubAccount(asset.Spot, "")
	exp.Balances.Set(currency.USD, accounts.Balance{Total: 108.9632670590686, Hold: 18.9632670590686, Free: 90})
	assert.Equal(t, exp, got, "balanceSubAccount should add an isolated position's balance to the account's, as held")

	got = balanceSubAccount(asset.Spot, []UserBalanceResponse{
		{IsolatedPositions: []IsolatedPositionBalance{{IsolationID: "1", PositionBalances: []CollateralBalance{
			{InstrumentName: "ETH", Quantity: preciseNumber("0.2"), UpdateTime: types.Time(time.UnixMilli(1792894002000))},
			{InstrumentName: "XRP", Quantity: preciseNumber("-0.2")},
			{InstrumentName: "CRO", Quantity: preciseNumber("20")},
		}}}},
		{
			PositionBalances: []CollateralBalance{
				{InstrumentName: "ETH", Quantity: preciseNumber("0.4"), ReservedQuantity: preciseNumber("0.1"), UpdateTime: types.Time(time.UnixMilli(1792894000000))},
				{InstrumentName: "BTC", Quantity: preciseNumber("-0.5")},
				{InstrumentName: "XRP", Quantity: preciseNumber("-0.1")},
			},
			IsolatedPositions: []IsolatedPositionBalance{
				{IsolationID: "2", PositionBalances: []CollateralBalance{{InstrumentName: "ETH", Quantity: preciseNumber("0.3"), UpdateTime: types.Time(time.UnixMilli(1792894001000))}}},
				{IsolationID: "3", PositionBalances: []CollateralBalance{{InstrumentName: "BTC", Quantity: preciseNumber("0.2")}}},
			},
		},
	})
	exp = accounts.NewSubAccount(asset.Spot, "")
	exp.Balances.Set(currency.ETH, accounts.Balance{Total: 0.9, Hold: 0.6, Free: 0.3, UpdatedAt: time.UnixMilli(1792894002000)})
	exp.Balances.Set(currency.BTC, accounts.Balance{Total: -0.3, Hold: 0.2, Borrowed: 0.5})
	exp.Balances.Set(currency.XRP, accounts.Balance{Total: -0.3, Borrowed: 0.3})
	exp.Balances.Set(currency.CRO, accounts.Balance{Total: 20, Hold: 20})
	assert.Equal(t, exp, got, "balanceSubAccount should add every entry's isolated balances in decimal arithmetic, a negative one as borrowed, keeping the latest update time")

	// A currency's amounts are summed exactly before they are converted, so amounts that cancel leave no total
	got = balanceSubAccount(asset.Spot, []UserBalanceResponse{{
		PositionBalances: []CollateralBalance{
			{InstrumentName: "USD", Quantity: preciseNumber("10000")},
			{InstrumentName: "EUR", Quantity: preciseNumber("10008.9632670590686")},
		},
		IsolatedPositions: []IsolatedPositionBalance{
			{IsolationID: "1", PositionBalances: []CollateralBalance{{InstrumentName: "USD", Quantity: preciseNumber("8.9632670590686")}}},
			{IsolationID: "2", PositionBalances: []CollateralBalance{
				{InstrumentName: "USD", Quantity: preciseNumber("-10008.9632670590686")},
				{InstrumentName: "EUR", Quantity: preciseNumber("-10008.9632670590686")},
			}},
		},
	}})
	exp = accounts.NewSubAccount(asset.Spot, "")
	exp.Balances.Set(currency.USD, accounts.Balance{Hold: 8.9632670590686, Free: 10000, Borrowed: 10008.9632670590686})
	exp.Balances.Set(currency.EUR, accounts.Balance{Free: 10008.9632670590686, Borrowed: 10008.9632670590686})
	assert.Equal(t, exp, got, "balanceSubAccount should sum a currency's amounts exactly, leaving no total where they cancel")

	got = balanceSubAccount(asset.Spot, []UserBalanceResponse{{
		PositionBalances: []CollateralBalance{
			{InstrumentName: "USD", Quantity: preciseNumber("10000"), ReservedQuantity: preciseNumber("10000")},
			{InstrumentName: "EUR", Quantity: preciseNumber("-10000")},
		},
		IsolatedPositions: []IsolatedPositionBalance{
			{IsolationID: "1", PositionBalances: []CollateralBalance{
				{InstrumentName: "USD", Quantity: preciseNumber("8.9632670590686")},
				{InstrumentName: "EUR", Quantity: preciseNumber("-8.9632670590686")},
			}},
			{IsolationID: "2", PositionBalances: []CollateralBalance{
				{InstrumentName: "USD", Quantity: preciseNumber("0.003")},
				{InstrumentName: "EUR", Quantity: preciseNumber("-0.003")},
			}},
		},
	}})
	exp = accounts.NewSubAccount(asset.Spot, "")
	exp.Balances.Set(currency.USD, accounts.Balance{Total: 10008.9662670590686, Hold: 10008.9662670590686})
	exp.Balances.Set(currency.EUR, accounts.Balance{Total: -10008.9662670590686, Borrowed: 10008.9662670590686})
	assert.Equal(t, exp, got, "balanceSubAccount should sum a currency's held and borrowed amounts exactly")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex := newTestExchange(t)
	assert.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error")
}

// testWalletRoutes answer the deposit and withdrawal history requests GetAccountFundingHistory and GetWithdrawalsHistory
// send with the docs' examples, made consistent
var testWalletRoutes = map[string]testRoute{
	"private/get-deposit-history": testResult(`{"deposit_list":[
		{"currency":"XRP","fee":0.0,"create_time":1607063412000,"id":"2220","update_time":1607063460000,"amount":"100","address":"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf?1234567890","status":"1","txid":"0x2ac9d2c3b6f9c4a14b0db94f77b4d44ec7d5d0a33a8cb4fe6e47f21e8ebb2f05"}]}`),
	"private/get-withdrawal-history": testResult(`{"withdrawal_list":[
		{"currency":"XRP","client_wid":"my_withdrawal_002","fee":1.0,"create_time":1607063412000,"id":"2220","update_time":1607063460000,"amount":100,"address":"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf?1234567890","status":"5","txid":"0x2ac9d2c3b6f9c4a14b0db94f77b4d44ec7d5d0a33a8cb4fe6e47f21e8ebb2f06","network_id":null},
		{"currency":"BTC","client_wid":"my_withdrawal_003","fee":0.0004,"create_time":1607063500000,"id":"2221","update_time":1607063560000,"amount":0.5,"address":"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf","status":"1","txid":"","network_id":"BTC"}]}`),
}

func TestGetAccountFundingHistory(t *testing.T) { //nolint:tparallel // the cases run in order before the GetAccountFundingHistory call after them
	t.Parallel()

	// The cases run in order, before the GetAccountFundingHistory call below
	for history, routes := range map[string]map[string]testRoute{
		"deposit":    {"private/get-deposit-history": testRefusedOnce(`{"deposit_list":[]}`), "private/get-withdrawal-history": testResult(`{"withdrawal_list":[]}`)},
		"withdrawal": {"private/get-deposit-history": testResult(`{"deposit_list":[]}`), "private/get-withdrawal-history": testRefusedOnce(`{"withdrawal_list":[]}`)},
	} {
		t.Run(history, func(t *testing.T) {
			refusing, _ := newRoutesServerExchange(t, routes)
			_, err := refusing.GetAccountFundingHistory(t.Context())
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "GetAccountFundingHistory must return a refused %s history's error", history)
			assert.Equalf(t, int64(50001), apiErr.Code, "GetAccountFundingHistory should return Crypto.com's refusal of the %s history", history)
		})
	}
	ex := e
	var received func() []testRequest
	if mockTests {
		ex, received = newRoutesServerExchange(t, testWalletRoutes)
	}
	skipLiveWithoutCredentials(t)
	got, err := ex.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		return
	}
	exp := []exchange.FundingHistory{
		{
			ExchangeName:    ex.Name,
			Status:          "arrived",
			TransferID:      "2220",
			Timestamp:       time.UnixMilli(1607063412000),
			Currency:        "XRP",
			Amount:          100,
			TransferType:    "deposit",
			CryptoToAddress: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf",
			CryptoTxID:      "0x2ac9d2c3b6f9c4a14b0db94f77b4d44ec7d5d0a33a8cb4fe6e47f21e8ebb2f05",
		},
		{
			ExchangeName:    ex.Name,
			Status:          "completed",
			TransferID:      "2220",
			Timestamp:       time.UnixMilli(1607063412000),
			Currency:        "XRP",
			Amount:          100,
			Fee:             1,
			TransferType:    "withdrawal",
			CryptoToAddress: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf",
			CryptoTxID:      "0x2ac9d2c3b6f9c4a14b0db94f77b4d44ec7d5d0a33a8cb4fe6e47f21e8ebb2f06",
		},
		{
			ExchangeName:    ex.Name,
			Status:          "processing",
			TransferID:      "2221",
			Timestamp:       time.UnixMilli(1607063500000),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0004,
			TransferType:    "withdrawal",
			CryptoToAddress: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf",
			CryptoChain:     "BTC",
		},
	}
	assert.Equal(t, exp, got, "GetAccountFundingHistory should return every deposit and withdrawal")
	expRequests := []testRequest{
		{Method: "private/get-deposit-history", Params: `{"page_size":"200"}`},
		{Method: "private/get-withdrawal-history", Params: `{"page_size":"200"}`},
	}
	assert.Equal(t, expRequests, received(), "GetAccountFundingHistory should request a full page of each history")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := newRefusingServerExchange(t).GetWithdrawalsHistory(t.Context(), currency.XRP, asset.Spot)
	require.ErrorIs(t, err, request.ErrBadStatus, "GetWithdrawalsHistory must return the request's failure")
	ex := e
	var received func() []testRequest
	if mockTests {
		ex, received = newRoutesServerExchange(t, testWalletRoutes)
	}
	skipLiveWithoutCredentials(t)
	got, err := ex.GetWithdrawalsHistory(t.Context(), currency.XRP, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if !mockTests {
		return
	}
	exp := []exchange.WithdrawalHistory{
		{
			Status:          "completed",
			TransferID:      "2220",
			Timestamp:       time.UnixMilli(1607063412000),
			Currency:        "XRP",
			Amount:          100,
			Fee:             1,
			TransferType:    "withdrawal",
			CryptoToAddress: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf",
			CryptoTxID:      "0x2ac9d2c3b6f9c4a14b0db94f77b4d44ec7d5d0a33a8cb4fe6e47f21e8ebb2f06",
		},
		{
			Status:          "processing",
			TransferID:      "2221",
			Timestamp:       time.UnixMilli(1607063500000),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0004,
			TransferType:    "withdrawal",
			CryptoToAddress: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf",
			CryptoChain:     "BTC",
		},
	}
	assert.Equal(t, exp, got, "GetWithdrawalsHistory should return every withdrawal")
	assert.Equal(t, []testRequest{{Method: "private/get-withdrawal-history", Params: `{"currency":"XRP","page_size":"200"}`}}, received(), "GetWithdrawalsHistory should request a full page of the currency's withdrawals")
}

func TestWithdrawalHistory(t *testing.T) {
	t.Parallel()
	expIDs := make([]string, maxWalletHistoryPageSize+1)
	fullPage := make([]string, maxWalletHistoryPageSize)
	for i := range expIDs {
		expIDs[i] = strconv.Itoa(2220 + i)
		if i < maxWalletHistoryPageSize {
			fullPage[i] = `{"id":` + expIDs[i] + `}`
		}
	}
	ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-withdrawal-history": func(params string) (string, *APIError) {
		if strings.Contains(params, `"page":"1"`) {
			return `{"withdrawal_list":[{"id":"` + expIDs[maxWalletHistoryPageSize] + `"}]}`, nil
		}
		return `{"withdrawal_list":[` + strings.Join(fullPage, ",") + `]}`, nil
	}})
	got, err := ex.withdrawalHistory(t.Context(), currency.XRP)
	require.NoError(t, err, "withdrawalHistory must not error")
	gotIDs := make([]string, len(got))
	for i := range got {
		gotIDs[i] = string(got[i].ID)
	}
	assert.Equal(t, expIDs, gotIDs, "withdrawalHistory should return every page's withdrawals, whichever way each sends its ID")
	assert.Equal(t, []testRequest{
		{Method: "private/get-withdrawal-history", Params: `{"currency":"XRP","page_size":"200"}`},
		{Method: "private/get-withdrawal-history", Params: `{"currency":"XRP","page_size":"200","page":"1"}`},
	}, received(), "withdrawalHistory should request the currency's withdrawals a full page at a time")

	_, err = newRefusingServerExchange(t).withdrawalHistory(t.Context(), currency.XRP)
	assert.ErrorIs(t, err, request.ErrBadStatus, "withdrawalHistory should return the request's failure")
}

func TestWalletRequestsUseContextCredentials(t *testing.T) {
	t.Parallel()
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "context-key", Secret: testAPISecret})
	for name, tc := range map[string]struct {
		call func(*Exchange) error
		keys []string
	}{
		"UpdateAccountBalances": {
			func(ex *Exchange) error { _, err := ex.UpdateAccountBalances(ctx, asset.Margin); return err },
			[]string{"context-key"},
		},
		"GetAccountFundingHistory": {
			func(ex *Exchange) error { _, err := ex.GetAccountFundingHistory(ctx); return err },
			[]string{"context-key", "context-key"},
		},
		"GetWithdrawalsHistory": {
			func(ex *Exchange) error {
				_, err := ex.GetWithdrawalsHistory(ctx, currency.XRP, asset.Spot)
				return err
			},
			[]string{"context-key"},
		},
		"withdrawalHistory": {
			func(ex *Exchange) error { _, err := ex.withdrawalHistory(ctx, currency.XRP); return err },
			[]string{"context-key"},
		},
		"WithdrawCryptocurrencyFunds": {
			func(ex *Exchange) error {
				_, err := ex.WithdrawCryptocurrencyFunds(ctx, &withdraw.Request{
					Exchange: ex.Name,
					Currency: currency.BTC,
					Amount:   1,
					Type:     withdraw.Crypto,
					Crypto:   withdraw.CryptoRequest{Address: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf"},
				})
				return err
			},
			[]string{"context-key"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex, keys := newAPIKeyRecorder(t, `{"id":1,"code":0,"result":{"data":[],"deposit_list":[],"withdrawal_list":[],"id":"2220"}}`)
			require.NoErrorf(t, tc.call(ex), "%s must not error", name)
			assert.Equalf(t, tc.keys, keys(), "%s should send each request with the context's credentials", name)
		})
	}
}

// testWalletRecord is a wallet history record, by its ID, with a detail a repeated record can differ in
type testWalletRecord struct {
	ID     string
	Status string
}

func TestPageWalletHistory(t *testing.T) {
	t.Parallel()
	ids := func(start, n int) []string {
		page := make([]string, n)
		for i := range page {
			page[i] = strconv.Itoa(start + i)
		}
		return page
	}
	t.Run("every page up to a short one", func(t *testing.T) {
		t.Parallel()
		var pages []uint64
		got, err := pageWalletHistory(t.Context(), func(_ context.Context, page uint64) ([]string, error) {
			pages = append(pages, page)
			require.LessOrEqual(t, len(pages), testServerRequestLimit, "pageWalletHistory must stop within the test server's request limit")
			if page == 0 {
				return ids(0, maxWalletHistoryPageSize), nil
			}
			return ids(maxWalletHistoryPageSize, 3), nil
		}, func(id *string) string { return *id })
		require.NoError(t, err, "pageWalletHistory must not error")
		assert.Equal(t, ids(0, maxWalletHistoryPageSize+3), got, "pageWalletHistory should return every page's records")
		assert.Equal(t, []uint64{0, 1}, pages, "pageWalletHistory should stop after a short page")
	})

	t.Run("a venue ignoring the page", func(t *testing.T) {
		t.Parallel()
		var pages []uint64
		got, err := pageWalletHistory(t.Context(), func(_ context.Context, page uint64) ([]string, error) {
			pages = append(pages, page)
			require.LessOrEqual(t, len(pages), testServerRequestLimit, "pageWalletHistory must stop within the test server's request limit")
			return ids(0, maxWalletHistoryPageSize), nil
		}, func(id *string) string { return *id })
		require.NoError(t, err, "pageWalletHistory must not error for a venue ignoring the page")
		assert.Equal(t, ids(0, maxWalletHistoryPageSize), got, "pageWalletHistory should return a repeated page's records once")
		assert.Equal(t, []uint64{0, 1}, pages, "pageWalletHistory should stop at a page holding nothing new")
	})

	t.Run("a record repeated on a later page or the same one", func(t *testing.T) {
		t.Parallel()
		first := make([]testWalletRecord, maxWalletHistoryPageSize)
		for i := range first {
			first[i] = testWalletRecord{ID: strconv.Itoa(i), Status: "first"}
		}
		var pages []uint64
		records, err := pageWalletHistory(t.Context(), func(_ context.Context, page uint64) ([]testWalletRecord, error) {
			pages = append(pages, page)
			require.LessOrEqual(t, len(pages), testServerRequestLimit, "pageWalletHistory must stop within the test server's request limit")
			if page == 0 {
				return first, nil
			}
			return []testWalletRecord{{ID: "0", Status: "repeated"}, {ID: "new", Status: "first"}, {ID: "new", Status: "repeated"}}, nil
		}, func(r *testWalletRecord) string { return r.ID })
		require.NoError(t, err, "pageWalletHistory must not error for pages repeating a record")
		assert.Equal(t, append(slices.Clone(first), testWalletRecord{ID: "new", Status: "first"}), records, "pageWalletHistory should keep the first of a record repeated on a later page or the same one")
		assert.Equal(t, []uint64{0, 1}, pages, "pageWalletHistory should stop after the short page")
	})

	t.Run("a full page holding one new record", func(t *testing.T) {
		t.Parallel()
		var pages []uint64
		got, err := pageWalletHistory(t.Context(), func(_ context.Context, page uint64) ([]string, error) {
			pages = append(pages, page)
			require.LessOrEqual(t, len(pages), testServerRequestLimit, "pageWalletHistory must stop within the test server's request limit")
			switch page {
			case 0:
				return ids(0, maxWalletHistoryPageSize), nil
			case 1:
				// A full page holding one record the page before did not
				return ids(1, maxWalletHistoryPageSize), nil
			default:
				return ids(maxWalletHistoryPageSize+1, 1), nil
			}
		}, func(id *string) string { return *id })
		require.NoError(t, err, "pageWalletHistory must not error for a full page holding one new record")
		assert.Equal(t, ids(0, maxWalletHistoryPageSize+2), got, "pageWalletHistory should go on past a full page holding one new record")
		assert.Equal(t, []uint64{0, 1, 2}, pages, "pageWalletHistory should stop at the short page, not at a full page holding one new record")
	})

	t.Run("a page's error", func(t *testing.T) {
		t.Parallel()
		var pages []uint64
		_, err := pageWalletHistory(t.Context(), func(_ context.Context, page uint64) ([]string, error) {
			pages = append(pages, page)
			require.LessOrEqual(t, len(pages), testServerRequestLimit, "pageWalletHistory must stop within the test server's request limit")
			return nil, errInvalidCount
		}, func(id *string) string { return *id })
		assert.ErrorIs(t, err, errInvalidCount, "pageWalletHistory should return a page's error")
		assert.Equal(t, []uint64{0}, pages, "pageWalletHistory should stop at a page's error")
	})
}

func TestTransferStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]string{
		DepositStatusNotArrived: "not arrived",
		DepositStatusArrived:    "arrived",
		DepositStatusFailed:     "failed",
		DepositStatusPending:    "pending",
		"9":                     "9",
	} {
		t.Run("deposit status "+status, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, depositStatus(status), "depositStatus should describe status %s", status)
		})
	}
	for status, exp := range map[string]string{
		WithdrawalStatusPending:           "pending",
		WithdrawalStatusProcessing:        "processing",
		WithdrawalStatusRejected:          "rejected",
		WithdrawalStatusPaymentInProgress: "payment in progress",
		WithdrawalStatusPaymentFailed:     "payment failed",
		WithdrawalStatusCompleted:         "completed",
		WithdrawalStatusCancelled:         "cancelled",
		"9":                               "9",
	} {
		t.Run("withdrawal status "+status, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, withdrawalStatus(status), "withdrawalStatus should describe status %s", status)
		})
	}
	assert.Equal(t, currency.BTC, withdrawalCurrency(&WithdrawalResponse{Symbol: currency.BTC}), "withdrawalCurrency should fall back to the symbol create-withdrawal's example names")
}

func TestSplitAddressTag(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string][2]string{
		"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf?1234567890": {"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf", "1234567890"},
		"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf":            {"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf", ""},
	} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			address, tag := splitAddressTag(in)
			assert.Equalf(t, exp, [2]string{address, tag}, "splitAddressTag should split %s", in)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetRecentTrades must refuse an empty pair")
	p := testPair(t, asset.Spot)
	got, err := e.GetRecentTrades(t.Context(), p, asset.Spot)
	require.NoError(t, err, "GetRecentTrades must not error")
	if !mockTests {
		assert.NotEmpty(t, got, "GetRecentTrades should return trades")
		return
	}
	exp := []trade.Data{
		{TID: "1792899071637623911", Exchange: e.Name, CurrencyPair: p, AssetType: asset.Spot, Side: order.Sell, Price: 80849.37, Amount: 0.00002, Timestamp: time.Unix(0, 1792899071637623911)},
		{TID: "1792899071550749165", Exchange: e.Name, CurrencyPair: p, AssetType: asset.Spot, Side: order.Sell, Price: 80850.58, Amount: 0.03092, Timestamp: time.Unix(0, 1792899071550749165)},
		{TID: "1792899071545150829", Exchange: e.Name, CurrencyPair: p, AssetType: asset.Spot, Side: order.Sell, Price: 80850.58, Amount: 0.00002, Timestamp: time.Unix(0, 1792899071545150827)},
	}
	assert.Equal(t, exp, got, "GetRecentTrades should return every field of the latest trades, timed to the nanosecond")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	t.Run("requests seven days at a time", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"public/get-trades": testResult(`{"data":[]}`)})
		start := testWindowStart.Add(-8 * 24 * time.Hour)
		_, err := ex.GetHistoricTrades(t.Context(), testSpotPair, asset.Spot, start, testWindowStart)
		require.NoError(t, err, "GetHistoricTrades must not error")
		ns := func(t time.Time) string { return strconv.FormatInt(t.UnixNano(), 10) }
		split := testWindowStart.Add(-tradesHistoryWindow)
		exp := []testRequest{
			{Method: "public/get-trades", Params: "count=150&end_ts=" + ns(testWindowStart) + "&instrument_name=BTC_USD&start_ts=" + ns(split)},
			{Method: "public/get-trades", Params: "count=150&end_ts=" + ns(split) + "&instrument_name=BTC_USD&start_ts=" + ns(start)},
		}
		assert.Equal(t, exp, received(), "GetHistoricTrades should request seven days at a time from the end")
	})
	// Serial, as live it requests the trades through e, which the calls below share
	t.Run("returns the trades before a window's last seven days", func(t *testing.T) {
		// BTCUSD-260626 expired on 2026-06-26, so the window's last seven days hold only its last two trades, and the three
		// before them only a second request returns; the test server answers each request as Crypto.com answered it live
		p := currency.NewPairWithDelimiter("BTCUSD", "260626", currency.DashDelimiter)
		start, end := time.Date(2026, time.June, 24, 0, 0, 0, 0, time.UTC), time.Date(2026, time.July, 2, 0, 0, 0, 0, time.UTC)
		ex := e
		if mockTests {
			ns := func(at time.Time) string { return strconv.FormatInt(at.UnixNano(), 10) }
			split := end.Add(-tradesHistoryWindow)
			answers := map[string]string{
				"count=150&end_ts=" + ns(end) + "&instrument_name=BTCUSD-260626&start_ts=" + ns(split): `{"data":[
					{"d":"1782395584888467444","t":1782395584888,"tn":1782395584888467444,"q":"0.0049","p":"58695.0","s":"sell","i":"BTCUSD-260626","m":"4611686018428638301"},
					{"d":"1782395584888465806","t":1782395584888,"tn":1782395584888465806,"q":"0.0001","p":"59016.4","s":"sell","i":"BTCUSD-260626","m":"4611686018428638300"}]}`,
				"count=150&end_ts=" + ns(split) + "&instrument_name=BTCUSD-260626&start_ts=" + ns(start): `{"data":[
					{"d":"1782323279014412130","t":1782323279014,"tn":1782323279014412130,"q":"0.0001","p":"59091.0","s":"sell","i":"BTCUSD-260626","m":"4611686018428638299"},
					{"d":"1782309868179660268","t":1782309868179,"tn":1782309868179660268,"q":"0.0001","p":"61220.7","s":"sell","i":"BTCUSD-260626","m":"4611686018428638298"},
					{"d":"1782269951061082861","t":1782269951061,"tn":1782269951061082861,"q":"0.0002","p":"62584.4","s":"buy","i":"BTCUSD-260626","m":"4611686018428638297"}]}`,
			}
			ex, _ = newRoutesServerExchange(t, map[string]testRoute{"public/get-trades": func(params string) (string, *APIError) {
				answer, ok := answers[params]
				if !ok {
					return "", &APIError{Code: 40004, Message: "no answer recorded for " + params}
				}
				return answer, nil
			}})
		}
		got, err := ex.GetHistoricTrades(t.Context(), p, asset.Futures, start, end)
		require.NoError(t, err, "GetHistoricTrades must not error")
		exp := []trade.Data{
			{TID: "1782269951061082861", Exchange: ex.Name, CurrencyPair: p, AssetType: asset.Futures, Side: order.Buy, Price: 62584.4, Amount: 0.0002, Timestamp: time.Unix(0, 1782269951061082861)},
			{TID: "1782309868179660268", Exchange: ex.Name, CurrencyPair: p, AssetType: asset.Futures, Side: order.Sell, Price: 61220.7, Amount: 0.0001, Timestamp: time.Unix(0, 1782309868179660268)},
			{TID: "1782323279014412130", Exchange: ex.Name, CurrencyPair: p, AssetType: asset.Futures, Side: order.Sell, Price: 59091, Amount: 0.0001, Timestamp: time.Unix(0, 1782323279014412130)},
			{TID: "1782395584888465806", Exchange: ex.Name, CurrencyPair: p, AssetType: asset.Futures, Side: order.Sell, Price: 59016.4, Amount: 0.0001, Timestamp: time.Unix(0, 1782395584888465806)},
			{TID: "1782395584888467444", Exchange: ex.Name, CurrencyPair: p, AssetType: asset.Futures, Side: order.Sell, Price: 58695, Amount: 0.0049, Timestamp: time.Unix(0, 1782395584888467444)},
		}
		assert.Equal(t, exp, got, "GetHistoricTrades should return the trades before the window's last seven days with those in them, oldest first")
	})
	_, err := e.GetHistoricTrades(t.Context(), testSpotPair, asset.Spot, testWindowStart, testWindowStart)
	require.ErrorIs(t, err, common.ErrStartEqualsEnd, "GetHistoricTrades must refuse an empty window")
	_, err = e.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Spot, testWindowStart, testWindowStart.Add(time.Minute))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricTrades must refuse an empty pair")
	_, err = newUnavailableServerExchange(t).GetHistoricTrades(t.Context(), testSpotPair, asset.Spot, testWindowStart, testWindowStart.Add(time.Minute))
	require.ErrorIs(t, err, request.ErrBadStatus, "GetHistoricTrades must return the request's failure")
	unknownSide := newResponseServerExchange(t, `{"id":-1,"method":"public/get-trades","code":0,"result":{"data":[{"d":"1","t":1792800000001,"tn":1792800000001000000,"q":"0.1","p":"81097","s":"hold","i":"BTC_USD"}]}}`)
	_, err = unknownSide.GetHistoricTrades(t.Context(), testSpotPair, asset.Spot, testWindowStart, testWindowStart.Add(time.Minute))
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetHistoricTrades must refuse a trade of an unknown side")
	p := testPair(t, asset.Spot)
	end := testWindowStart.Add(time.Second)
	got, err := e.GetHistoricTrades(mockRequestContext(t), p, asset.Spot, testWindowStart, end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	for i := range got {
		assert.Falsef(t, got[i].Timestamp.Before(testWindowStart) || !got[i].Timestamp.Before(end), "GetHistoricTrades should only return trades in the window, not %s", got[i].Timestamp)
	}
	if !mockTests {
		return
	}
	exp := []trade.Data{
		// The two fills of one taker order share a time, and keep the venue's order
		{TID: "1792800000042959688", Exchange: e.Name, CurrencyPair: p, AssetType: asset.Spot, Side: order.Buy, Price: 81097, Amount: 0.00009, Timestamp: time.Unix(0, 1792800000042959687)},
		{TID: "1792800000042959687", Exchange: e.Name, CurrencyPair: p, AssetType: asset.Spot, Side: order.Buy, Price: 81097, Amount: 0.00003, Timestamp: time.Unix(0, 1792800000042959687)},
	}
	assert.Equal(t, exp, got, "GetHistoricTrades should return the window's trades, oldest first")
}

// testPagedRecord is a record a paged history returns, by its ID and time
type testPagedRecord struct {
	ID string
	At time.Time
}

// testPager returns a history pager over records, which it serves newest first, limit at a time, as a history request
// does, recording each window requested; a pager asking for more windows than the test server would answer fails the
// test there, so one that no longer stops ends its test rather than hanging it
func testPager(t *testing.T, records []testPagedRecord, limit int, windows *[][2]time.Time) *historyPager[testPagedRecord] {
	t.Helper()
	return &historyPager[testPagedRecord]{
		exchangeName: "test",
		limit:        limit,
		fetch: func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
			*windows = append(*windows, [2]time.Time{start, end})
			require.LessOrEqual(t, len(*windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
			var page []testPagedRecord
			for _, r := range records {
				if !r.At.Before(start) && r.At.Before(end) && len(page) < limit {
					page = append(page, r)
				}
			}
			return page, nil
		},
		record: func(r *testPagedRecord) (string, time.Time) { return r.ID, r.At },
	}
}

func TestHistoryPagerBackwards(t *testing.T) {
	// Not parallel, nor is its case of passed-over times, as that case captures the global log; it runs first, so the
	// calls below cannot log the pager's first warnings before its capture starts
	t.Run("warns of passed-over times", func(t *testing.T) {
		takeWarnings := captureWarnings(t)
		start := testWindowStart
		end := start.Add(time.Hour)
		at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }
		warning := func(limit int, at time.Time) string {
			return "test history holds " + strconv.Itoa(limit) + " or more records at " + at.String() + ", which no request can page within, so any more are left out"
		}
		// The cases share the log capture in order, so each takes only the warnings it logged
		for _, tc := range []struct {
			name    string
			records []testPagedRecord
			limit   int
			exp     []string
		}{
			{"a time a page splits", []testPagedRecord{{"6", at(50)}, {"5", at(40)}, {"4", at(30)}, {"3", at(30)}, {"2", at(20)}}, 3, nil},
			{"a time holding more records than a page", []testPagedRecord{{"C", at(30)}, {"B", at(30)}, {"A", at(30)}, {"D", at(10)}}, 2, []string{warning(2, at(30))}},
			{"the window's last time holding more records than a page", []testPagedRecord{{"B", end.Add(-time.Nanosecond)}, {"A", end.Add(-time.Nanosecond)}, {"C", end.Add(-time.Nanosecond)}}, 2, []string{warning(2, end.Add(-time.Nanosecond))}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var windows [][2]time.Time
				_, err := testPager(t, tc.records, tc.limit, &windows).backwards(t.Context(), start, end, make(common.Seen[string]))
				require.NoErrorf(t, err, "backwards must not error for %s", tc.name)
				assert.Equalf(t, tc.exp, takeWarnings(), "backwards should warn of each time it passes over for %s", tc.name)
			})
		}
	})

	start := testWindowStart
	end := start.Add(time.Hour)
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }

	// Two records share a time a page splits, and one falls before the window
	records := []testPagedRecord{{"6", at(50)}, {"5", at(40)}, {"4", at(30)}, {"3", at(30)}, {"2", at(20)}, {"1", at(-10)}}
	var windows [][2]time.Time
	got, err := testPager(t, records, 3, &windows).backwards(t.Context(), start, end, make(common.Seen[string]))
	require.NoError(t, err, "backwards must not error")
	assert.Equal(t, records[:5], got, "backwards should return every record in the window once")
	expWindows := [][2]time.Time{{start, end}, {start, at(30).Add(time.Nanosecond)}, {start, at(20).Add(time.Nanosecond)}}
	assert.Equal(t, expWindows, windows, "backwards should end each page a nanosecond after the oldest record so far")

	// A full page sharing one time holds nothing new when requested again, so its time is passed over
	records = []testPagedRecord{{"C", at(30)}, {"B", at(30)}, {"A", at(30)}, {"D", at(10)}}
	windows = nil
	got, err = testPager(t, records, 2, &windows).backwards(t.Context(), start, end, make(common.Seen[string]))
	require.NoError(t, err, "backwards must not error for a time a page cannot page within")
	assert.Equal(t, []testPagedRecord{records[0], records[1], records[3]}, got, "backwards should pass over a time holding more records than a page")
	expWindows = [][2]time.Time{{start, end}, {start, at(30).Add(time.Nanosecond)}, {start, at(30)}}
	assert.Equal(t, expWindows, windows, "backwards should end the page after a page holding nothing new at its oldest time")

	// A full page sharing the last time of the window passes over it at once, as a page ending a nanosecond later is
	// the page itself
	records = []testPagedRecord{{"B", end.Add(-time.Nanosecond)}, {"A", end.Add(-time.Nanosecond)}, {"C", end.Add(-time.Nanosecond)}, {"D", at(10)}}
	windows = nil
	got, err = testPager(t, records, 2, &windows).backwards(t.Context(), start, end, make(common.Seen[string]))
	require.NoError(t, err, "backwards must not error for the window's last time holding a page of records")
	assert.Equal(t, []testPagedRecord{records[0], records[1], records[3]}, got, "backwards should pass over the window's last time holding more records than a page")
	assert.Equal(t, [][2]time.Time{{start, end}, {start, end.Add(-time.Nanosecond)}}, windows, "backwards should end the next page at the window's last time")

	// A full page holding nothing before the window's end, as a venue ignoring it returns, holds none of the window
	windows = nil
	pager := testPager(t, nil, 2, &windows)
	pager.fetch = func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
		windows = append(windows, [2]time.Time{start, end})
		require.LessOrEqual(t, len(windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
		if len(windows) > 2 {
			// A pager that does not stop would request the same window again without end
			return nil, errInvalidCount
		}
		return []testPagedRecord{{"Y", end.Add(time.Minute)}, {"X", end}}, nil
	}
	got, err = pager.backwards(t.Context(), start, end, make(common.Seen[string]))
	require.NoError(t, err, "backwards must not error for a page after the window")
	assert.Empty(t, got, "backwards should leave out records after the window")
	assert.Len(t, windows, 1, "backwards should stop at a page holding nothing before the window's end")

	// Records seen before, and records outside the window a venue returns, are left out
	pager = testPager(t, nil, 3, &windows)
	pager.fetch = func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
		windows = append(windows, [2]time.Time{start, end})
		require.LessOrEqual(t, len(windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
		return []testPagedRecord{{"7", at(45)}, {"6", at(44)}, {"5", at(40)}, {"0", at(34)}}, nil
	}
	seen := common.Seen[string]{"6": true}
	got, err = pager.backwards(t.Context(), at(35), at(45), seen)
	require.NoError(t, err, "backwards must not error for records seen before")
	assert.Equal(t, []testPagedRecord{{"5", at(40)}}, got, "backwards should leave out records seen before and outside the window")
	assert.Equal(t, common.Seen[string]{"6": true, "5": true}, seen, "backwards should add the records it returns to those seen")

	pager.fetch = func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
		windows = append(windows, [2]time.Time{start, end})
		require.LessOrEqual(t, len(windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
		return nil, errInvalidCount
	}
	_, err = pager.backwards(t.Context(), start, end, make(common.Seen[string]))
	assert.ErrorIs(t, err, errInvalidCount, "backwards should return a page's error")

	windows = nil
	pager.fetch = func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
		windows = append(windows, [2]time.Time{start, end})
		require.LessOrEqual(t, len(windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
		if len(windows) == 1 {
			return []testPagedRecord{{"C", at(50)}, {"B", at(40)}, {"A", at(30)}}, nil
		}
		return nil, errInvalidCount
	}
	got, err = pager.backwards(t.Context(), start, end, make(common.Seen[string]))
	assert.ErrorIs(t, err, errInvalidCount, "backwards should return the error of a page following a full one")
	assert.Nil(t, got, "backwards should return no records when a later page fails")
	assert.Len(t, windows, 2, "backwards should not request a failed page again")
}

// captureWarnings has the log hand the warnings logged to the function returned, which takes those logged since it was
// last called. The log hook and config are process-wide, so a test capturing them must not run in parallel, and cleanup
// returns both to the log package's defaults
func captureWarnings(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var warnings []string
	require.NoError(t, log.SetGlobalLogConfig(log.GenDefaultSettings()), "SetGlobalLogConfig must not error")
	// Setting the logger up gives it the default config's headers, by which the hook tells warnings apart
	require.NoError(t, log.SetupGlobalLogger(t.Name(), false), "SetupGlobalLogger must not error")
	log.SetCustomLogHook(func(header, _ string, a ...any) bool {
		if header == "[WARN]" {
			mu.Lock()
			warnings = append(warnings, fmt.Sprint(a...))
			mu.Unlock()
		}
		return true
	})
	t.Cleanup(func() {
		log.SetCustomLogHook(nil)
		assert.NoError(t, log.SetGlobalLogConfig(&log.Config{}), "SetGlobalLogConfig should not error")
	})
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		taken := warnings
		warnings = nil
		return taken
	}
}

func TestHistoryPagerPage(t *testing.T) {
	t.Parallel()
	// A window in the past, as one ending later is cut short at the current time
	start := testWindowStart.Add(-3 * privateHistoryWindow)
	end := start.Add(2*privateHistoryWindow + time.Hour)
	var windows [][2]time.Time
	pager := testPager(t, nil, 100, &windows)
	pager.fetch = func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
		windows = append(windows, [2]time.Time{start, end})
		require.LessOrEqual(t, len(windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
		// Each span holds a record of its own and one shared with the others, which page returns once
		return []testPagedRecord{{ID: end.String(), At: end.Add(-time.Minute)}, {ID: "shared", At: start}}, nil
	}
	got, err := pager.page(t.Context(), start, end, privateHistoryWindow)
	require.NoError(t, err, "page must not error")
	expWindows := [][2]time.Time{
		{end.Add(-privateHistoryWindow), end},
		{end.Add(-2 * privateHistoryWindow), end.Add(-privateHistoryWindow)},
		{start, end.Add(-2 * privateHistoryWindow)},
	}
	assert.Equal(t, expWindows, windows, "page should request the window a span at a time from its end")
	exp := []testPagedRecord{
		{ID: end.String(), At: end.Add(-time.Minute)},
		{ID: "shared", At: end.Add(-privateHistoryWindow)},
		{ID: end.Add(-privateHistoryWindow).String(), At: end.Add(-privateHistoryWindow - time.Minute)},
		{ID: end.Add(-2 * privateHistoryWindow).String(), At: end.Add(-2*privateHistoryWindow - time.Minute)},
	}
	assert.Equal(t, exp, got, "page should return each record once")

	windows = nil
	_, err = pager.page(t.Context(), time.Now().Add(-time.Hour), time.Now().Add(time.Hour), privateHistoryWindow)
	require.NoError(t, err, "page must not error for a window ending after the current time")
	require.Len(t, windows, 1, "page must request a window ending after the current time once")
	assert.WithinDuration(t, time.Now(), windows[0][1], time.Minute, "page should end a window ending after the current time at it")

	windows = nil
	pager.fetch = func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
		windows = append(windows, [2]time.Time{start, end})
		require.LessOrEqual(t, len(windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
		if len(windows) == 1 {
			return []testPagedRecord{{ID: "1", At: end.Add(-time.Minute)}}, nil
		}
		return nil, errInvalidCount
	}
	got, err = pager.page(t.Context(), start, end, privateHistoryWindow)
	assert.ErrorIs(t, err, errInvalidCount, "page should return the error of a span following one that returned records")
	assert.Nil(t, got, "page should return no records when a later span fails")
	assert.Len(t, windows, 2, "page should not request a failed span again")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	t.Run("current two weeks", func(t *testing.T) {
		t.Parallel()
		start, firstWeek, bothWeeks := currentTwoWeeksReplies()
		for _, tc := range []struct {
			name  string
			reply string
			err   error
		}{
			{name: "first week has a candle", reply: firstWeek, err: kline.ErrInsufficientCandleData},
			{name: "both weeks have a candle", reply: bothWeeks},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, _ := newRoutesServerExchange(t, map[string]testRoute{"public/get-candlestick": testResult(tc.reply)})
				item, err := ex.GetHistoricCandles(t.Context(), testSpotPair, asset.Spot, kline.TwoWeek, start, start.Add(kline.TwoWeek.Duration()))
				skipIfTwoWeeksEnded(t, start)
				if tc.err != nil {
					assert.ErrorIs(t, err, tc.err, "GetHistoricCandles should error with too few weeks for a two-week candle")
					assert.Nil(t, item, "GetHistoricCandles should return no candles")
					return
				}
				require.NoError(t, err, "GetHistoricCandles must not error")
				require.Len(t, item.Candles, 1, "GetHistoricCandles must return the current two weeks' candle")
				exp := kline.Candle{Time: time.UnixMilli(start.UnixMilli()), Open: 100, High: 104, Low: 98, Close: 103, Volume: 8, ValidationIssues: kline.PartialCandle}
				assert.Equal(t, exp, item.Candles[0], "GetHistoricCandles should return the two weeks' candle, marked partial")
			})
		}
	})
	p := testPair(t, asset.Spot)
	end := testWindowStart.Add(3 * time.Hour)
	got, err := e.GetHistoricCandles(t.Context(), p, asset.Spot, kline.OneHour, testWindowStart, end)
	require.NoError(t, err, "GetHistoricCandles must not error")
	if !mockTests {
		assert.NotEmpty(t, got.Candles, "GetHistoricCandles should return candles")
		return
	}
	exp := &kline.Item{
		Exchange: e.Name,
		Pair:     p,
		Asset:    asset.Spot,
		Interval: kline.OneHour,
		Candles: []kline.Candle{
			{Time: testWindowStart, Open: 81097, High: 81120.26, Low: 80508.11, Close: 80535.23, Volume: 202.44773},
			{Time: testWindowStart.Add(time.Hour), Open: 80533.86, High: 80886.45, Low: 80524.08, Close: 80837.08, Volume: 138.48729},
			{Time: testWindowStart.Add(2 * time.Hour), Open: 80837.08, High: 80907.2, Low: 80744, Close: 80905.17, Volume: 55.85107},
		},
	}
	assert.Equal(t, exp, got, "GetHistoricCandles should return the window's candles")
	_, err = e.GetHistoricCandles(t.Context(), p, asset.Spot, kline.OneMin, testWindowStart, testWindowStart.Add(301*time.Minute))
	assert.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "GetHistoricCandles should refuse a window of more candles than one request returns")

	// Crypto.com's 14D candles open on alternate Mondays, a week from GoCryptoTrader's two-week boundaries, so two-week
	// candles are built from 7D ones; each timeframe is answered with the candles Crypto.com returned live on 2026-10-28
	recorded := map[string]string{
		"7D": `{"interval":"7D","instrument_name":"BTC_USD","data":[
			{"o":"84467.87","h":"87240.82","l":"82513.99","c":"86510.00","v":"24803.36193","t":1790553600000},
			{"o":"86510.02","h":"86992.01","l":"80320.41","c":"83687.77","v":"23733.03674","t":1791158400000},
			{"o":"83687.78","h":"84276.15","l":"79688.01","c":"82015.65","v":"26721.30058","t":1791763200000},
			{"o":"82014.96","h":"84889.90","l":"79128.21","c":"80054.34","v":"22655.15212","t":1792368000000}]}`,
		"14D": `{"interval":"14D","instrument_name":"BTC_USD","data":[
			{"o":"86510.02","h":"86992.01","l":"79688.01","c":"82015.65","v":"50454.33732","t":1791158400000},
			{"o":"82014.96","h":"84889.90","l":"79128.21","c":"80833.01","v":"29548.97771","t":1792368000000}]}`,
	}
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"public/get-candlestick": func(params string) (string, *APIError) {
			query, err := url.ParseQuery(params)
			if err != nil {
				return "", &APIError{Code: 40001, Message: err.Error()}
			}
			return recorded[query.Get("timeframe")], nil
		},
	})
	start, end := time.UnixMilli(1790553600000), time.UnixMilli(1792972800000)
	got, err = ex.GetHistoricCandles(t.Context(), testSpotPair, asset.Spot, kline.TwoWeek, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error for two-week candles")
	firstVolume, secondVolume := 24803.36193, 26721.30058
	exp = &kline.Item{
		Exchange: ex.Name,
		Pair:     testSpotPair,
		Asset:    asset.Spot,
		Interval: kline.TwoWeek,
		Candles: []kline.Candle{
			{Time: start, Open: 84467.87, High: 87240.82, Low: 80320.41, Close: 83687.77, Volume: firstVolume + 23733.03674},
			{Time: start.Add(kline.TwoWeek.Duration()), Open: 83687.78, High: 84889.9, Low: 79128.21, Close: 80054.34, Volume: secondVolume + 22655.15212},
		},
	}
	assert.Equal(t, exp, got, "GetHistoricCandles should build each two-week candle from two weekly ones")
	expRequests := []testRequest{{Method: "public/get-candlestick", Params: "count=300&end_ts=1792972799999&instrument_name=BTC_USD&start_ts=1790553600000&timeframe=7D"}}
	assert.Equal(t, expRequests, received(), "GetHistoricCandles should request weekly candles for two-week ones")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	t.Run("current two weeks", func(t *testing.T) {
		t.Parallel()
		start, firstWeek, bothWeeks := currentTwoWeeksReplies()
		for _, tc := range []struct {
			name  string
			reply string
			err   error
		}{
			{name: "first week has a candle", reply: firstWeek, err: kline.ErrInsufficientCandleData},
			{name: "both weeks have a candle", reply: bothWeeks},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, _ := newRoutesServerExchange(t, map[string]testRoute{"public/get-candlestick": testResult(tc.reply)})
				item, err := ex.GetHistoricCandlesExtended(t.Context(), testSpotPair, asset.Spot, kline.TwoWeek, start, start.Add(kline.TwoWeek.Duration()))
				skipIfTwoWeeksEnded(t, start)
				if tc.err != nil {
					assert.ErrorIs(t, err, tc.err, "GetHistoricCandlesExtended should error with too few weeks for a two-week candle")
					assert.Nil(t, item, "GetHistoricCandlesExtended should return no candles")
					return
				}
				require.NoError(t, err, "GetHistoricCandlesExtended must not error")
				require.Len(t, item.Candles, 1, "GetHistoricCandlesExtended must return the current two weeks' candle")
				exp := kline.Candle{Time: time.UnixMilli(start.UnixMilli()), Open: 100, High: 104, Low: 98, Close: 103, Volume: 8, ValidationIssues: kline.PartialCandle}
				assert.Equal(t, exp, item.Candles[0], "GetHistoricCandlesExtended should return the two weeks' candle, marked partial")
			})
		}
	})
	var received func() []testRequest
	ex := e
	if mockTests {
		ex, received = newRoutesServerExchange(t, map[string]testRoute{
			// Each window's request is answered with one candle, opening at the window's start
			"public/get-candlestick": func(params string) (string, *APIError) {
				query, err := url.ParseQuery(params)
				if err != nil {
					return "", &APIError{Code: 40001, Message: err.Error()}
				}
				return `{"interval":"1m","instrument_name":"BTCUSD-PERP","data":[{"o":"1","h":"2","l":"0.5","c":"1.5","v":"10","t":` + query.Get("start_ts") + `}]}`, nil
			},
		})
	}
	p := testPair(t, asset.PerpetualSwap)
	end := testWindowStart.Add(450 * time.Minute)
	got, err := ex.GetHistoricCandlesExtended(t.Context(), p, asset.PerpetualSwap, kline.OneMin, testWindowStart, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	if !mockTests {
		assert.NotEmpty(t, got.Candles, "GetHistoricCandlesExtended should return candles")
		return
	}
	expRequests := []testRequest{
		{Method: "public/get-candlestick", Params: "count=300&end_ts=1792817999999&instrument_name=BTCUSD-PERP&start_ts=1792800000000&timeframe=1m"},
		{Method: "public/get-candlestick", Params: "count=300&end_ts=1792826999999&instrument_name=BTCUSD-PERP&start_ts=1792818000000&timeframe=1m"},
	}
	assert.Equal(t, expRequests, received(), "GetHistoricCandlesExtended should request 300 candles at a time, each window ending a millisecond before the next opens")
	require.Len(t, got.Candles, 450, "GetHistoricCandlesExtended must return a candle for every minute, padding those not returned")
	assert.Equal(t, kline.Candle{Time: testWindowStart, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10}, got.Candles[0], "GetHistoricCandlesExtended should return the first window's candle")
	assert.Equal(t, kline.Candle{Time: testWindowStart.Add(300 * time.Minute), Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10}, got.Candles[300], "GetHistoricCandlesExtended should return the second window's candle")
}

// currentTwoWeeksReplies returns the start of the current two weeks and Crypto.com's candlestick results holding its
// first week's 7D candle alone, and both weeks'. A request for the two weeks pads a result holding the first week alone
// with an empty second week, then drops that as an empty partial candle, leaving one week, too few to make the
// two-week candle
func currentTwoWeeksReplies() (start time.Time, firstWeek, bothWeeks string) {
	start = time.Now().UTC().Truncate(kline.TwoWeek.Duration())
	first := `{"o":"100","h":"102","l":"99","c":"101","v":"5","t":` + strconv.FormatInt(start.UnixMilli(), 10) + `}`
	second := `{"o":"101","h":"104","l":"98","c":"103","v":"3","t":` + strconv.FormatInt(start.AddDate(0, 0, 7).UnixMilli(), 10) + `}`
	reply := func(candles string) string {
		return `{"interval":"7D","instrument_name":"BTC_USD","data":[` + candles + `]}`
	}
	return start, reply(first), reply(first + "," + second)
}

// skipIfTwoWeeksEnded skips a test whose request for the two weeks from start was made as or after they ended, when the
// request could have fallen in the next two weeks
func skipIfTwoWeeksEnded(t *testing.T, start time.Time) {
	t.Helper()
	if !time.Now().UTC().Truncate(kline.TwoWeek.Duration()).Equal(start) {
		t.Skip("The current two weeks ended during the test, so its request could have fallen in the next two weeks")
	}
}

func TestCandles(t *testing.T) {
	t.Parallel()
	// Crypto.com takes in a monthly candle opening at a window's end, so the window ends a millisecond early: live, a window
	// ending at October's opening returns October's candle with the others. The reply was recorded live for this request
	ex := e
	var received func() []testRequest
	if mockTests {
		ex, received = newRoutesServerExchange(t, map[string]testRoute{
			"public/get-candlestick": testResult(`{"interval":"1M","data":[{"o":"58562.87","h":"66925.83","l":"57737.07","c":"62840.00","v":"126396.4943","t":1782864000000},{"o":"62843.05","h":"81500.00","l":"62216.48","c":"78559.98","v":"122373.96006","t":1785542400000},{"o":"78571.17","h":"87427.66","l":"74897.07","c":"83564.32","v":"109085.85444","t":1788220800000}],"instrument_name":"BTC_USD"}`),
		})
	}
	start, end := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	got, err := ex.candles(t.Context(), "BTC_USD", kline.OneMonth, start, end)
	require.NoError(t, err, "candles must not error")
	exp := []kline.Candle{
		{Time: time.UnixMilli(1782864000000), Open: 58562.87, High: 66925.83, Low: 57737.07, Close: 62840, Volume: 126396.4943},
		{Time: time.UnixMilli(1785542400000), Open: 62843.05, High: 81500, Low: 62216.48, Close: 78559.98, Volume: 122373.96006},
		{Time: time.UnixMilli(1788220800000), Open: 78571.17, High: 87427.66, Low: 74897.07, Close: 83564.32, Volume: 109085.85444},
	}
	assert.Equal(t, exp, got, "candles should return the monthly candles opening within the window")
	if mockTests {
		assert.Equal(t, []testRequest{{Method: "public/get-candlestick", Params: "count=300&end_ts=1790812799999&instrument_name=BTC_USD&start_ts=1782864000000&timeframe=1M"}}, received(), "candles should end a monthly window a millisecond early")
	}
	_, err = newUnavailableServerExchange(t).candles(t.Context(), "BTC_USD", kline.OneMonth, start, end)
	assert.ErrorIs(t, err, request.ErrBadStatus, "candles should return the request's failure")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must refuse an empty currency")
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		got, err := e.GetDepositAddress(t.Context(), currency.CRO, "", "")
		require.NoError(t, err, "GetDepositAddress must not error")
		if !mockTests {
			assert.NotEmpty(t, got.Address, "GetDepositAddress should return an address")
			return
		}
		exp := &deposit.Address{Address: "cro1q9vfn0fjtyfvx3h8k2rwwgvxnz9k4j6m3y5pth", Chain: "CRO"}
		assert.Equal(t, exp, got, "GetDepositAddress should return the currency's only active address")
		got, err = e.GetDepositAddress(t.Context(), currency.CRO, "", "cro")
		require.NoError(t, err, "GetDepositAddress must not error for a chain")
		assert.Equal(t, exp, got, "GetDepositAddress should match a chain in either case")
		_, err = e.GetDepositAddress(t.Context(), currency.CRO, "", "ETH")
		assert.ErrorIs(t, err, errNoDepositAddress, "GetDepositAddress should refuse a chain whose address is inactive")
	})

	ex := newResponseServerExchange(t, `{"id":1,"method":"private/get-deposit-address","code":0,"result":{"deposit_address_list":[
		{"currency":"USDC","address":"0xabc","status":"1","network":"ETH"},
		{"currency":"USDC","address":"Gh7?memo","status":"1","network":"SOL"}]}}`)
	_, err = ex.GetDepositAddress(t.Context(), currency.USDC, "", "")
	assert.ErrorIs(t, err, errChainRequired, "GetDepositAddress should refuse a currency with active addresses on several chains without a chain")
	assert.True(t, ex.Features.Supports.RESTCapabilities.MultiChainDepositRequiresChainSet, "Crypto.com should require a chain for a currency deposited on several networks, so the deposit address sync asks for each")
	got, err := ex.GetDepositAddress(t.Context(), currency.USDC, "", "SOL")
	require.NoError(t, err, "GetDepositAddress must not error for one of several chains")
	assert.Equal(t, &deposit.Address{Address: "Gh7", Tag: "memo", Chain: "SOL"}, got, "GetDepositAddress should split the address tag")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must refuse an empty currency")
	skipLiveWithoutCredentials(t)
	c := currency.NewCode("MATIC")
	if !mockTests {
		c = currency.BTC
	}
	got, err := e.GetAvailableTransferChains(t.Context(), c)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	if !mockTests {
		assert.NotEmpty(t, got, "GetAvailableTransferChains should return BTC's networks")
		return
	}
	assert.Equal(t, []string{"BNB", "ETH", "MATIC"}, got, "GetAvailableTransferChains should return the currency's networks")
	_, err = e.GetAvailableTransferChains(t.Context(), currency.NewCode("NOPE"))
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "GetAvailableTransferChains should refuse a currency without networks")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) { //nolint:tparallel // the withdrawal runs on e in order, before the later call of the same method
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must validate the request")
	req := &withdraw.Request{
		Exchange: e.Name,
		Currency: currency.BTC,
		Amount:   1,
		Type:     withdraw.Crypto,
		Crypto:   withdraw.CryptoRequest{Address: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf"},
	}
	_, err = newUnavailableServerExchange(t).WithdrawCryptocurrencyFunds(t.Context(), req)
	require.ErrorIs(t, err, request.ErrBadStatus, "WithdrawCryptocurrencyFunds must return the request's failure")
	placed, err := newResponseServerExchange(t, `{"id":-1,"method":"private/create-withdrawal","code":0,"result":{"id":2220,"amount":1,"fee":0.0004,"symbol":"BTC","address":"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf","client_wid":"my_withdrawal_002","create_time":1607063412000,"network_id":null}}`).WithdrawCryptocurrencyFunds(t.Context(), req)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not report a withdrawal Crypto.com took, with a bare ID, as failed")
	assert.Equal(t, "2220", placed.ID, "WithdrawCryptocurrencyFunds should return a bare withdrawal ID")
	ctx := t.Context()
	t.Run("a BTC withdrawal", func(t *testing.T) {
		skipLiveUnlessCanManipulate(t)
		w := &withdraw.Request{
			Exchange:      e.Name,
			Currency:      currency.BTC,
			Amount:        1,
			Type:          withdraw.Crypto,
			ClientOrderID: "my_withdrawal_002",
			Crypto:        withdraw.CryptoRequest{Address: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf", Chain: "BTC"},
		}
		if !mockTests {
			// Live, a small amount goes to GoCryptoTrader's donation address, which the API key must have whitelisted
			w = &withdraw.Request{Exchange: e.Name, Currency: currency.BTC, Amount: 0.001, Type: withdraw.Crypto, Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress, Chain: "BTC"}}
		}
		got, err := e.WithdrawCryptocurrencyFunds(ctx, w)
		require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
		if mockTests {
			assert.Equal(t, &withdraw.ExchangeResponse{Name: e.Name, ID: "2220"}, got, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")
		} else {
			assert.NotEmpty(t, got.ID, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")
		}
	})

	// The address tag, which currencies such as XRP need, is sent with the address
	ex, sent := newSignedRecorder(t, "private/create-withdrawal", `{"id":"2221"}`)
	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:      ex.Name,
		Currency:      currency.XRP,
		Amount:        25.5,
		Type:          withdraw.Crypto,
		ClientOrderID: "my_withdrawal_003",
		Crypto:        withdraw.CryptoRequest{Address: "rAddress", AddressTag: "1234567890", Chain: "XRP"},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error for an address with a tag")
	params := sent()
	require.Len(t, params, 1, "WithdrawCryptocurrencyFunds must send one withdrawal")
	assert.JSONEq(t, `{"client_wid":"my_withdrawal_003","currency":"XRP","amount":"25.5","address":"rAddress","address_tag":"1234567890","network_id":"XRP"}`, params[0], "WithdrawCryptocurrencyFunds should send the address tag and chain")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
	_, err = e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must refuse a nil fee builder")
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.InternationalBankDepositFee})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFeeByType must refuse an unsupported fee type")
	// An estimate needs no credentials, which a backtest may not have
	offline := new(Exchange)
	offline.SetDefaults()
	offlineTaker, err := offline.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, PurchasePrice: 80000, Amount: 0.5})
	require.NoError(t, err, "GetFeeByType must not error for an offline taker fee")
	assert.Equal(t, 200.0, offlineTaker, "GetFeeByType should estimate a taker trade's fee at the schedule's highest taker rate")
	offlineMaker, err := offline.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, PurchasePrice: 80000, Amount: 0.5, IsMaker: true})
	require.NoError(t, err, "GetFeeByType must not error for an offline maker fee")
	assert.Equal(t, 100.0, offlineMaker, "GetFeeByType should estimate a maker trade's fee at the schedule's highest maker rate")
	_, err = offline.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testSpotPair})
	assert.Error(t, err, "GetFeeByType should refuse a trade fee for a pair of no asset")
	require.NoError(t, offline.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.EMPTYPAIR}, false), "StorePairs must store the available pairs")
	_, err = offline.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFeeByType should refuse a trade fee for an empty pair")
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		taker, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testSpotPair, PurchasePrice: 80000, Amount: 0.5})
		require.NoError(t, err, "GetFeeByType must not error for a taker fee")
		maker, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testSpotPair, PurchasePrice: 80000, Amount: 0.5, IsMaker: true})
		require.NoError(t, err, "GetFeeByType must not error for a maker fee")
		if !mockTests {
			assert.Positive(t, taker, "GetFeeByType should return a taker fee")
			return
		}
		assert.InDelta(t, 27.6, taker, 1e-9, "GetFeeByType should charge 6.9 basis points of a taker trade's value")
		assert.InDelta(t, 26, maker, 1e-9, "GetFeeByType should charge 6.5 basis points of a maker trade's value")
		for c, exp := range map[currency.Code]float64{currency.NewCode("MATIC"): 20, currency.NewCode("AGLD"): 0} {
			t.Run(c.String(), func(t *testing.T) {
				t.Parallel()
				fee, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(c, currency.EMPTYCODE)})
				require.NoErrorf(t, err, "GetFeeByType must not error for %s's withdrawal fee", c)
				assert.Equalf(t, exp, fee, "GetFeeByType should return %s's withdrawal fee on its default network, or its only one", c)
			})
		}
	})

	ex := newResponseServerExchange(t, `{"id":1,"method":"private/get-currency-networks","code":0,"result":{"currency_map":{"USDC":{"default_network":"SOL","network_list":[{"network_id":"ETH"},{"network_id":"BNB"}]}}}}`)
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.USDC, currency.EMPTYCODE)})
	assert.ErrorIs(t, err, errNoDefaultNetwork, "GetFeeByType should refuse a currency whose default network it does not list")
	ex = newResponseServerExchange(t, `{"id":1,"code":40101,"message":"Unauthorized"}`)
	var apiErr *APIError
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testSpotPair})
	require.ErrorAs(t, err, &apiErr, "GetFeeByType must report a refused fee rate lookup")
	assert.Equal(t, &APIError{Code: 40101, Message: "Unauthorized"}, apiErr, "GetFeeByType should report the fee rate lookup's error")
	apiErr = nil
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.USDC, currency.EMPTYCODE)})
	require.ErrorAs(t, err, &apiErr, "GetFeeByType must report a refused network lookup")
	assert.Equal(t, &APIError{Code: 40101, Message: "Unauthorized"}, apiErr, "GetFeeByType should report the network lookup's error")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetServerTime should not be supported")
}

// testSubmit returns the order the order tests submit: tradingTestOrder, a post-only buy of BTCUSD-PERP far below its
// market
func testSubmit() *order.Submit {
	return &order.Submit{
		Exchange:      e.Name,
		Type:          order.Limit,
		Side:          order.Buy,
		Pair:          testPerpetualPair,
		AssetType:     asset.PerpetualSwap,
		TimeInForce:   order.GoodTillCancel | order.PostOnly,
		Price:         50000.5,
		Amount:        0.0001,
		ClientOrderID: tradingTestOrderClientOrderID,
	}
}

// testTriggerSubmit returns the trigger order the order tests submit: advancedTriggerOrder, a sell STOP_LIMIT of 10
// CRO_USD at 0.0279 placed once the last price falls to 0.028
func testTriggerSubmit() *order.Submit {
	return &order.Submit{
		Exchange:      e.Name,
		Type:          order.StopLimit,
		Side:          order.Sell,
		Pair:          currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter),
		AssetType:     asset.Spot,
		Price:         0.0279,
		Amount:        10,
		TriggerPrice:  0.028,
		ClientOrderID: tradingTestOrderClientOrderID,
	}
}

func TestNewOrderRequest(t *testing.T) { //nolint:tparallel // the cases build orders on e in order, among later calls of newOrderRequest
	t.Parallel()

	// The cases build their orders on e in order, before the calls below build more on e and on another exchange
	for _, tc := range []struct {
		name   string
		modify func(*order.Submit)
		exp    *orderRequest
	}{
		{
			name: "a post-only limit buy",
			exp: &orderRequest{clientOrderID: tradingTestOrderClientOrderID, order: &CreateOrderRequest{
				InstrumentName:        "BTCUSD-PERP",
				Side:                  SideBuy,
				OrderType:             OrderTypeLimit,
				Price:                 50000.5,
				Quantity:              0.0001,
				ClientOrderID:         tradingTestOrderClientOrderID,
				TimeInForce:           TimeInForceGoodTillCancel,
				ExecutionInstructions: []string{ExecutionInstructionPostOnly},
			}},
		},
		{
			name: "a reduce-only market sell on an isolated position",
			modify: func(s *order.Submit) {
				s.Type, s.Side, s.Price, s.TimeInForce, s.ReduceOnly, s.MarginType, s.Leverage = order.Market, order.Short, 80000, order.ImmediateOrCancel, true, margin.Isolated, 10
			},
			exp: &orderRequest{clientOrderID: tradingTestOrderClientOrderID, order: &CreateOrderRequest{
				InstrumentName:        "BTCUSD-PERP",
				Side:                  SideSell,
				OrderType:             OrderTypeMarket,
				Quantity:              0.0001,
				ClientOrderID:         tradingTestOrderClientOrderID,
				TimeInForce:           TimeInForceImmediateOrCancel,
				ExecutionInstructions: []string{ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin},
				Leverage:              10,
			}},
		},
		{
			name: "a margin market buy by its quote amount",
			modify: func(s *order.Submit) {
				s.Type, s.Pair, s.AssetType, s.Price, s.Amount, s.QuoteAmount, s.TimeInForce, s.Leverage = order.Market, testSpotPair, asset.Margin, 0, 0, 25, order.UnknownTIF, 3
			},
			exp: &orderRequest{clientOrderID: tradingTestOrderClientOrderID, order: &CreateOrderRequest{
				InstrumentName: "BTC_USD",
				Side:           SideBuy,
				OrderType:      OrderTypeMarket,
				Notional:       25,
				ClientOrderID:  tradingTestOrderClientOrderID,
				SpotMargin:     SpotMarginMargin,
				Leverage:       3,
			}},
		},
		{
			name: "a spot limit buy stating no leverage",
			modify: func(s *order.Submit) {
				s.Pair, s.AssetType, s.TimeInForce, s.Leverage = testSpotPair, asset.Spot, order.UnknownTIF, 1
			},
			exp: &orderRequest{clientOrderID: tradingTestOrderClientOrderID, order: &CreateOrderRequest{
				InstrumentName: "BTC_USD",
				Side:           SideBuy,
				OrderType:      OrderTypeLimit,
				Price:          50000.5,
				Quantity:       0.0001,
				ClientOrderID:  tradingTestOrderClientOrderID,
			}},
		},
		{
			name:   "a stop limit",
			modify: func(s *order.Submit) { *s = *testTriggerSubmit(); s.Leverage = 1 },
			exp: &orderRequest{clientOrderID: tradingTestOrderClientOrderID, trigger: &CreateAdvancedOrderRequest{
				InstrumentName:     "CRO_USD",
				Side:               SideSell,
				OrderType:          OrderTypeStopLimit,
				Quantity:           10,
				Price:              0.0279,
				ReferencePrice:     0.028,
				ReferencePriceType: ReferencePriceTypeLastPrice,
				ClientOrderID:      tradingTestOrderClientOrderID,
			}},
		},
		{
			name:   "a take profit limit",
			modify: func(s *order.Submit) { *s = *testTriggerSubmit(); s.Type = order.TakeProfitLimit },
			exp: &orderRequest{clientOrderID: tradingTestOrderClientOrderID, trigger: &CreateAdvancedOrderRequest{
				InstrumentName:     "CRO_USD",
				Side:               SideSell,
				OrderType:          OrderTypeTakeProfitLimit,
				Quantity:           10,
				Price:              0.0279,
				ReferencePrice:     0.028,
				ReferencePriceType: ReferencePriceTypeLastPrice,
				ClientOrderID:      tradingTestOrderClientOrderID,
			}},
		},
		{
			name: "a margin take profit triggered by the mark price",
			modify: func(s *order.Submit) {
				*s = *testTriggerSubmit()
				s.Type, s.AssetType, s.TriggerPriceType, s.ReduceOnly = order.TakeProfit, asset.Margin, order.MarkPrice, true
			},
			exp: &orderRequest{clientOrderID: tradingTestOrderClientOrderID, trigger: &CreateAdvancedOrderRequest{
				InstrumentName:        "CRO_USD",
				Side:                  SideSell,
				OrderType:             OrderTypeTakeProfit,
				Quantity:              10,
				ReferencePrice:        0.028,
				ReferencePriceType:    ReferencePriceTypeMarkPrice,
				ClientOrderID:         tradingTestOrderClientOrderID,
				ExecutionInstructions: []string{ExecutionInstructionReduceOnly, ExecutionInstructionMarginOrder},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testSubmit()
			if tc.modify != nil {
				tc.modify(s)
			}
			got, err := e.newOrderRequest(s)
			require.NoErrorf(t, err, "newOrderRequest must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newOrderRequest should place %s", tc.name)
		})
	}

	// A margin order without a client order ID is given a unique one, as Crypto.com would give it the request's nonce
	s := testSubmit()
	s.Pair, s.AssetType, s.ClientOrderID = testSpotPair, asset.Margin, ""
	got, err := e.newOrderRequest(s)
	require.NoError(t, err, "newOrderRequest must not error for a margin order without a client order ID")
	_, err = uuid.Parse(got.clientOrderID)
	require.NoError(t, err, "newOrderRequest must give a margin order without a client order ID a UUID")
	assert.Equal(t, got.clientOrderID, got.order.ClientOrderID, "newOrderRequest should send the client order ID it gave the order")
	again, err := e.newOrderRequest(s)
	require.NoError(t, err, "newOrderRequest must not error for a margin order without a client order ID")
	assert.NotEqual(t, got.clientOrderID, again.clientOrderID, "newOrderRequest should give each margin order its own client order ID")
	*s = *testTriggerSubmit()
	s.AssetType, s.ClientOrderID = asset.Margin, ""
	got, err = e.newOrderRequest(s)
	require.NoError(t, err, "newOrderRequest must not error for a margin trigger order without a client order ID")
	assert.NotEmpty(t, got.trigger.ClientOrderID, "newOrderRequest should give a margin trigger order without a client order ID one")
	s.AssetType = asset.Spot
	got, err = e.newOrderRequest(s)
	require.NoError(t, err, "newOrderRequest must not error for a spot order without a client order ID")
	assert.Empty(t, got.trigger.ClientOrderID, "newOrderRequest should leave a spot order's client order ID to Crypto.com")

	// The refusals run in order on e, before newOrderRequest builds an order on another exchange below
	for _, tc := range []struct {
		name   string
		modify func(*order.Submit)
		err    error
	}{
		{"an unsupported asset", func(s *order.Submit) { s.AssetType = asset.Options }, asset.ErrNotSupported},
		{"a take profit leg", func(s *order.Submit) { s.RiskManagementModes.TakeProfit.Price = 90000 }, errRiskManagementUnsupported},
		{"a trailing stop", func(s *order.Submit) { s.Type = order.TrailingStop }, order.ErrUnsupportedOrderType},
		{"a good till time order", func(s *order.Submit) { s.TimeInForce = order.GoodTillTime }, order.ErrUnsupportedTimeInForce},
		{"isolated margin on spot", func(s *order.Submit) { s.Pair, s.AssetType, s.MarginType = testSpotPair, asset.Spot, margin.Isolated }, margin.ErrMarginTypeUnsupported},
		{"leverage on spot", func(s *order.Submit) { s.Pair, s.AssetType, s.Leverage = testSpotPair, asset.Spot, 2 }, errLeverageUnsupported},
		{"leverage on a trigger order", func(s *order.Submit) { *s = *testTriggerSubmit(); s.AssetType, s.Leverage = asset.Margin, 1.5 }, errTriggerOrderMargin},
		{"a trigger order by its quote amount", func(s *order.Submit) { *s = *testTriggerSubmit(); s.Amount, s.QuoteAmount = 0, 1 }, errTriggerOrderQuantity},
		{"a trigger order with leverage", func(s *order.Submit) { *s = *testTriggerSubmit(); s.AssetType, s.Leverage = asset.Margin, 2 }, errTriggerOrderMargin},
		{"an unknown trigger price type", func(s *order.Submit) { *s = *testTriggerSubmit(); s.TriggerPriceType = order.UnknownPriceType }, errInvalidPriceType},
		{"a limit order with a trigger price", func(s *order.Submit) { s.TriggerPrice = 1 }, errTriggerPriceUnsupported},
		{"a limit order by its quote amount", func(s *order.Submit) { s.Amount, s.QuoteAmount = 0, 5 }, errQuoteAmountUnsupported},
		{"a market sell by its quote amount", func(s *order.Submit) {
			s.Type, s.Side, s.Price, s.Amount, s.QuoteAmount, s.TimeInForce = order.Market, order.Sell, 0, 0, 5, order.UnknownTIF
		}, errQuoteAmountUnsupported},
		{"an order without an amount", func(s *order.Submit) { s.Amount = 0 }, order.ErrAmountIsInvalid},
		{"a client order ID of 37 characters", func(s *order.Submit) { s.ClientOrderID = strings.Repeat("c", 37) }, errClientOrderIDTooLong},
		{"a trigger order without a trigger price", func(s *order.Submit) { *s = *testTriggerSubmit(); s.TriggerPrice = 0 }, errInvalidReferencePrice},
		// Validate accepts sides combined, which name no side to place
		{"a side both buy and sell", func(s *order.Submit) { s.Side = order.Buy | order.Sell }, order.ErrSideIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testSubmit()
			tc.modify(s)
			_, err := e.newOrderRequest(s)
			assert.ErrorIsf(t, err, tc.err, "newOrderRequest should refuse %s", tc.name)
		})
	}

	// An asset whose pairs have no request format names no instrument
	ex := newTestExchange(t)
	require.NoError(t, ex.CurrencyPairs.Store(asset.PerpetualSwap, &currency.PairStore{AssetEnabled: true}), "Store must store perpetual swaps without a request format")
	_, err = ex.newOrderRequest(testSubmit())
	assert.ErrorIs(t, err, currency.ErrPairFormatIsNil, "newOrderRequest should refuse a pair whose instrument it cannot name")
}

func TestSetIsolationID(t *testing.T) { //nolint:tparallel // the cases run in order, before the failed position lookup
	t.Parallel()
	closing := func() *orderRequest {
		return &orderRequest{clientOrderID: "close", order: &CreateOrderRequest{
			InstrumentName:        "BTCUSD-PERP",
			Side:                  SideSell,
			OrderType:             OrderTypeMarket,
			Quantity:              0.0002,
			ClientOrderID:         "close",
			ExecutionInstructions: []string{ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin},
		}}
	}
	onPosition := closing()
	onPosition.order.IsolationID = "19848526"
	stopLoss := func() *orderRequest {
		return &orderRequest{clientOrderID: "stop", trigger: &CreateAdvancedOrderRequest{InstrumentName: "BTCUSD-PERP", Side: SideSell, OrderType: OrderTypeStopLoss, Quantity: 0.0002, ReferencePrice: 70000, ClientOrderID: "stop"}}
	}
	submit := func(marginType margin.Type) *order.Submit {
		return &order.Submit{Exchange: e.Name, Type: order.Market, Side: order.Sell, Pair: testPerpetualPair, AssetType: asset.PerpetualSwap, Amount: 0.0002, ReduceOnly: true, MarginType: marginType, ClientOrderID: "close"}
	}
	lookup := []testRequest{{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`}}
	// The cases run in order, before setIsolationID's failed position lookup below
	for _, tc := range []struct {
		name       string
		marginType margin.Type
		req        *orderRequest
		positions  testRoute
		exp        *orderRequest
		requests   []testRequest
	}{
		{name: "a cross margin order", marginType: margin.Multi, req: closing(), exp: closing()},
		{name: "an isolated margin trigger order", marginType: margin.Isolated, req: stopLoss(), exp: stopLoss()},
		{
			name:       "an isolated margin order with an isolated position open",
			marginType: margin.Isolated,
			req:        closing(),
			positions:  testPositionRoutes["private/get-positions"],
			exp:        onPosition,
			requests:   lookup,
		},
		{
			name:       "an isolated margin order without an isolated position",
			marginType: margin.Isolated,
			req:        closing(),
			positions:  testResult(`{"data":[{"quantity":"-0.1984","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"0"}]}`),
			exp:        closing(),
			requests:   lookup,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			routes := make(map[string]testRoute)
			if tc.positions != nil {
				routes["private/get-positions"] = tc.positions
			}
			ex, received := newRoutesServerExchange(t, routes)
			require.NoErrorf(t, ex.setIsolationID(t.Context(), submit(tc.marginType), tc.req), "setIsolationID must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, tc.req, "setIsolationID should name the isolated position expected for %s", tc.name)
			assert.Equalf(t, tc.requests, received(), "setIsolationID should send the position lookups expected for %s", tc.name)
		})
	}

	r := closing()
	err := newUnavailableServerExchange(t).setIsolationID(t.Context(), submit(margin.Isolated), r)
	assert.ErrorIs(t, err, request.ErrBadStatus, "setIsolationID should return the error looking up the isolated position")
	assert.Equal(t, closing(), r, "setIsolationID should leave the order unchanged when it cannot look up the isolated position")
}

func TestReferencePriceType(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for pt, exp := range map[order.PriceType]string{
		order.LastPrice:  ReferencePriceTypeLastPrice,
		order.MarkPrice:  ReferencePriceTypeMarkPrice,
		order.IndexPrice: ReferencePriceTypeIndexPrice,
	} {
		t.Run(pt.String(), func(t *testing.T) {
			got, err := referencePriceType(pt)
			require.NoErrorf(t, err, "referencePriceType must not error for %s", pt)
			assert.Equalf(t, exp, got, "referencePriceType should map %s", pt)
		})
	}
	_, err := referencePriceType(order.UnknownPriceType)
	assert.ErrorIs(t, err, errInvalidPriceType, "referencePriceType should refuse an unknown price type")
}

// testSubmitResponse returns the submit response of a submitted order placed with an order ID, its wall-clock times
// cleared
func testSubmitResponse(t *testing.T, s *order.Submit, orderID string) *order.SubmitResponse {
	t.Helper()
	exp, err := s.DeriveSubmitResponse(orderID)
	require.NoError(t, err, "DeriveSubmitResponse must not error")
	exp.Date, exp.LastUpdated = time.Time{}, time.Time{}
	return exp
}

// clearSubmitTimes checks a submit response is timed and clears its wall-clock times
func clearSubmitTimes(t *testing.T, sr *order.SubmitResponse) {
	t.Helper()
	assert.NotZero(t, sr.Date, "The submit response should be dated")
	assert.NotZero(t, sr.LastUpdated, "The submit response should be timed")
	sr.Date, sr.LastUpdated = time.Time{}, time.Time{}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must refuse a nil order")
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		s := testSubmit()
		if !mockTests {
			s.ClientOrderID = newTradingTestClientOrderID()
		}
		got, err := e.SubmitOrder(t.Context(), s)
		require.NoError(t, err, "SubmitOrder must not error")
		if !mockTests {
			cancelTradingOrderAfterTest(t, got.OrderID)
			return
		}
		clearSubmitTimes(t, got)
		assert.Equal(t, testSubmitResponse(t, s, tradingTestOrderID), got, "SubmitOrder should return the order's ID")

		trigger := testTriggerSubmit()
		trigger.TriggerPriceType = order.UnknownPriceType
		_, err = e.SubmitOrder(t.Context(), trigger)
		require.ErrorIs(t, err, errInvalidPriceType, "SubmitOrder must refuse an order it cannot place")
	})

	ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/advanced/create-order": testResult(advancedOrderIDResult)})
	trigger := testTriggerSubmit()
	trigger.ClientOrderID = ""
	got, err := ex.SubmitOrder(t.Context(), trigger)
	require.NoError(t, err, "SubmitOrder must not error for a trigger order")
	clearSubmitTimes(t, got)
	exp := testSubmitResponse(t, trigger, "5755600460443882762")
	exp.ClientOrderID = "c5f682ed-7108-4f1c-b755-972fcdca0f02"
	assert.Equal(t, exp, got, "SubmitOrder should return the client order ID Crypto.com gave an order placed without one")
	expRequests := []testRequest{{
		Method: "private/advanced/create-order",
		Params: `{"instrument_name":"CRO_USD","side":"SELL","type":"STOP_LIMIT","quantity":"10","price":"0.0279","ref_price":"0.028","ref_price_type":"LAST_PRICE"}`,
	}}
	assert.Equal(t, expRequests, received(), "SubmitOrder should place a trigger order with the advanced order methods")

	// Crypto.com answers before matching an order, so a market order is reported new rather than filled
	ex, _ = newRoutesServerExchange(t, map[string]testRoute{"private/create-order": testResult(`{"order_id":"1","client_oid":"market"}`)})
	got, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Type: order.Market, Side: order.Sell, Pair: testSpotPair, AssetType: asset.Spot, Amount: 0.001, ClientOrderID: "market"})
	require.NoError(t, err, "SubmitOrder must not error for a market order")
	assert.Equal(t, order.New, got.Status, "SubmitOrder should report a market order new, as Crypto.com answers before matching it")

	// An isolated margin order names the instrument's isolated position, listed after its cross margin position, as
	// Crypto.com opens a new isolated position for an order naming none; the cases run in order, as SubmitOrder is
	// called again below
	for _, tc := range []struct {
		name      string
		positions testRoute
		params    string
	}{
		{
			name:      "with an isolated position open",
			positions: testPositionRoutes["private/get-positions"],
			params:    `{"instrument_name":"BTCUSD-PERP","side":"SELL","type":"MARKET","quantity":"0.0002","client_oid":"close","exec_inst":["REDUCE_ONLY","ISOLATED_MARGIN"],"isolation_id":"19848526"}`,
		},
		{
			name:      "without an isolated position",
			positions: testResult(`{"data":[{"quantity":"-0.1984","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"0"}]}`),
			params:    `{"instrument_name":"BTCUSD-PERP","side":"SELL","type":"MARKET","quantity":"0.0002","client_oid":"close","exec_inst":["REDUCE_ONLY","ISOLATED_MARGIN"]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, received = newRoutesServerExchange(t, map[string]testRoute{
				"private/get-positions": tc.positions,
				"private/create-order":  testResult(`{"order_id":"1","client_oid":"close"}`),
			})
			_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Type: order.Market, Side: order.Sell, Pair: testPerpetualPair, AssetType: asset.PerpetualSwap, Amount: 0.0002, ReduceOnly: true, MarginType: margin.Isolated, ClientOrderID: "close"})
			require.NoErrorf(t, err, "SubmitOrder must not error for an isolated margin order %s", tc.name)
			expRequests = []testRequest{
				{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`},
				{Method: "private/create-order", Params: tc.params},
			}
			assert.Equalf(t, expRequests, received(), "SubmitOrder should look up the instrument's isolated position for an isolated margin order %s", tc.name)
		})
	}

	// An isolated margin order whose isolated position cannot be looked up is not placed, as it could open a second one
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-positions": func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} },
		"private/create-order":  testResult(`{"order_id":"1","client_oid":"close"}`),
	})
	_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Type: order.Market, Side: order.Sell, Pair: testPerpetualPair, AssetType: asset.PerpetualSwap, Amount: 0.0002, ReduceOnly: true, MarginType: margin.Isolated, ClientOrderID: "close"})
	var lookupErr *APIError
	require.ErrorAs(t, err, &lookupErr, "SubmitOrder must return the error looking up the isolated position")
	assert.Equal(t, []testRequest{{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`}}, received(), "SubmitOrder should not place an isolated margin order whose isolated position it could not look up")
}

func TestSubmitOrderWith(t *testing.T) {
	t.Parallel()
	t.Run("an older order's update keeps a reused client order ID tracked", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		older := []byte(`{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.order","channel":"user.order","data":[{"order_id":"101","client_oid":"shared","instrument_name":"BTC_USD","exec_inst":["MARGIN_ORDER"],"side":"BUY","order_type":"LIMIT","status":"FILLED","quantity":"0.001","limit_price":"50000"}]}}`)
		require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, older), "the older order's update must be handled")
		placing := func(_ context.Context, r *CreateOrderRequest) (*OrderIDResponse, error) {
			assert.NoError(t, ex.wsHandleChannelData(t.Context(), nil, older), "the older order's update should be handled while the new order is in flight")
			filled, err := ex.userTradeFill(&WebsocketUserTrade{OrderID: "102", ClientOrderID: r.ClientOrderID, InstrumentName: "BTC_USD", Side: "BUY"})
			if assert.NoError(t, err, "userTradeFill should not error for the new order's early trade") {
				assert.Equal(t, asset.Margin, filled.AssetType, "the new margin order's trade arriving before its order ID should still be filled as margin")
			}
			return &OrderIDResponse{OrderID: "102", ClientOrderID: r.ClientOrderID}, nil
		}
		_, err := ex.submitOrderWith(t.Context(), &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: "shared"}, placing, nil)
		require.NoError(t, err, "submitOrderWith must not error")
		assert.True(t, ex.isMarginOrder("101", ""), "the older margin order should stay tracked by its order ID")
		assert.True(t, ex.isMarginOrder("102", ""), "the new margin order should be tracked by its order ID")
		assert.False(t, ex.isMarginOrder("", "shared"), "the reused client order ID should no longer be tracked once the new order's ID is known")
	})
	ex := newTestExchange(t)
	// placing records the order sent and whether its client order ID was tracked as a margin order's when it was sent,
	// then answers it with orderID, or with err
	var sent *CreateOrderRequest
	var trackedWhenSent bool
	placing := func(orderID string, err error) func(context.Context, *CreateOrderRequest) (*OrderIDResponse, error) {
		return func(_ context.Context, r *CreateOrderRequest) (*OrderIDResponse, error) {
			sent, trackedWhenSent = r, ex.isMarginOrder("", r.ClientOrderID)
			if err != nil {
				return nil, err
			}
			return &OrderIDResponse{OrderID: orderID, ClientOrderID: r.ClientOrderID}, nil
		}
	}
	s := &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001}
	got, err := ex.submitOrderWith(t.Context(), s, placing("9", nil), nil)
	require.NoError(t, err, "submitOrderWith must not error for a margin order")
	require.NotEmpty(t, sent.ClientOrderID, "submitOrderWith must give a margin order without a client order ID one")
	assert.Equal(t, sent.ClientOrderID, got.ClientOrderID, "submitOrderWith should report the client order ID it gave the order")
	assert.True(t, trackedWhenSent, "submitOrderWith should track a margin order by its client order ID before placing it, as its trades can arrive first")
	assert.True(t, ex.isMarginOrder("9", ""), "submitOrderWith should track a placed margin order by its order ID")
	assert.False(t, ex.isMarginOrder("", sent.ClientOrderID), "submitOrderWith should stop tracking a placed margin order by its client order ID, which other orders can share, once its order ID is known")

	// A spot order reusing the client order ID is tracked as no margin order, and the margin order still by its order ID
	s.AssetType, s.ClientOrderID = asset.Spot, sent.ClientOrderID
	_, err = ex.submitOrderWith(t.Context(), s, placing("10", nil), nil)
	require.NoError(t, err, "submitOrderWith must not error for a spot order")
	assert.False(t, trackedWhenSent, "submitOrderWith should not track a client order ID a spot order reuses when placing it")
	assert.False(t, ex.isMarginOrder("10", s.ClientOrderID), "submitOrderWith should track a placed spot order as no margin order")
	assert.True(t, ex.isMarginOrder("9", s.ClientOrderID), "the margin order should still be tracked by its order ID")

	// A margin order Crypto.com refuses is no longer tracked by its client order ID
	s.AssetType, s.ClientOrderID = asset.Margin, "refused"
	_, err = ex.submitOrderWith(t.Context(), s, placing("", &APIError{Code: 306, Message: "INSUFFICIENT_AVAILABLE_BALANCE"}), nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "submitOrderWith must return Crypto.com's refusal")
	assert.True(t, trackedWhenSent, "submitOrderWith should track a margin order by its client order ID before placing it")
	assert.False(t, ex.isMarginOrder("", "refused"), "submitOrderWith should stop tracking the client order ID of an order Crypto.com refused")

	// A margin order its own checks refuse is neither sent nor tracked by its client order ID
	s.ClientOrderID = strings.Repeat("c", 37)
	_, err = ex.submitOrderWith(t.Context(), s, placing("", nil), nil)
	require.ErrorIs(t, err, errClientOrderIDTooLong, "submitOrderWith must refuse a client order ID of 37 characters")
	assert.False(t, ex.isMarginOrder("", s.ClientOrderID), "submitOrderWith should not track the client order ID of an order its checks refuse")

	// A margin order whose request fails otherwise may have been placed, so is still tracked
	s.ClientOrderID = "unknown"
	_, err = ex.submitOrderWith(t.Context(), s, placing("", context.Canceled), nil)
	require.ErrorIs(t, err, context.Canceled, "submitOrderWith must return the request's error")
	assert.True(t, ex.isMarginOrder("", "unknown"), "submitOrderWith should go on tracking the client order ID of an order whose outcome is unknown")

	// A spot order reusing that client order ID stops it marking a margin order before the spot order is placed, as the
	// spot order's trades can arrive first
	s.AssetType = asset.Spot
	_, err = ex.submitOrderWith(t.Context(), s, placing("11", nil), nil)
	require.NoError(t, err, "submitOrderWith must not error for a spot order reusing a client order ID")
	assert.False(t, trackedWhenSent, "submitOrderWith should stop tracking a client order ID a spot order reuses before placing it")
	assert.False(t, ex.isMarginOrder("11", s.ClientOrderID), "submitOrderWith should track a placed spot order as no margin order")

	t.Run("acknowledgement omits client ID", func(t *testing.T) {
		t.Parallel()
		// A margin order is sent with the client order ID it is given, else one generated for it
		for _, tc := range []struct{ name, clientOrderID string }{
			{name: "given", clientOrderID: "margin_given"},
			{name: "generated"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestExchange(t)
				var sentClientOrderID string
				acknowledge := func(_ context.Context, r *CreateOrderRequest) (*OrderIDResponse, error) {
					sentClientOrderID = r.ClientOrderID
					return &OrderIDResponse{OrderID: "41"}, nil
				}
				s := &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: tc.clientOrderID}
				_, err := ex.submitOrderWith(t.Context(), s, acknowledge, nil)
				require.NoError(t, err, "submitOrderWith must not error for an acknowledgement without a client order ID")
				require.NotEmpty(t, sentClientOrderID, "submitOrderWith must send a margin order with a client order ID")
				assert.True(t, ex.isMarginOrder("41", ""), "submitOrderWith should track the placed margin order by its order ID")
				assert.False(t, ex.isMarginOrder("", sentClientOrderID), "submitOrderWith should stop tracking the client order ID it sent the margin order with, which the acknowledgement omits")
			})
		}
	})
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrder must refuse a nil order")
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t)
	s := testSubmit()
	s.Exchange = ex.Name
	if !mockTests {
		s.ClientOrderID = newTradingTestClientOrderID()
	}
	got, err := ex.WebsocketSubmitOrder(t.Context(), s)
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	if !mockTests {
		cancelTradingOrderAfterTest(t, got.OrderID)
		return
	}
	clearSubmitTimes(t, got)
	assert.Equal(t, testSubmitResponse(t, s, tradingTestOrderID), got, "WebsocketSubmitOrder should return the order's ID")
	requests := venue.received("private/create-order")
	require.Len(t, requests, 1, "WebsocketSubmitOrder must send one order")
	assert.JSONEq(t, tradingTestOrderParams, string(requests[0].Params), "WebsocketSubmitOrder should send the order")

	trigger := testTriggerSubmit()
	trigger.Exchange = ex.Name
	got, err = ex.WebsocketSubmitOrder(t.Context(), trigger)
	require.NoError(t, err, "WebsocketSubmitOrder must not error for a trigger order")
	clearSubmitTimes(t, got)
	assert.Equal(t, testSubmitResponse(t, trigger, "5755600460443882762"), got, "WebsocketSubmitOrder should return the trigger order's ID")
	assert.Len(t, venue.received("private/advanced/create-order"), 1, "WebsocketSubmitOrder should place a trigger order with the advanced order methods")
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, errOrderListEmpty, "WebsocketSubmitOrders must refuse no orders")
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{testSubmit(), nil})
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrders must refuse a nil order")
	t.Run("orders placed in a list", func(t *testing.T) {
		skipLiveUnlessCanManipulate(t)
		ex, venue := newWebsocketTestExchange(t)
		orders := []*order.Submit{testSubmit(), testSubmit()}
		orders[1].ClientOrderID = "api_leg2"
		for i, s := range orders {
			s.Exchange = ex.Name
			if !mockTests {
				s.ClientOrderID = newTradingTestClientOrderID() + "-" + strconv.Itoa(i)
			}
		}
		got, err := ex.WebsocketSubmitOrders(t.Context(), orders)
		require.NoError(t, err, "WebsocketSubmitOrders must not error")
		require.Len(t, got, len(orders), "WebsocketSubmitOrders must return a response for every order")
		if !mockTests {
			for i := range got {
				if assert.NoErrorf(t, got[i].SubmissionError, "WebsocketSubmitOrders should place order %d", i) {
					cancelTradingOrderAfterTest(t, got[i].OrderID)
				}
			}
			return
		}
		for i, orderID := range []string{"5755600460443882762", "5755600460443882763"} {
			clearSubmitTimes(t, got[i])
			assert.Equalf(t, testSubmitResponse(t, orders[i], orderID), got[i], "WebsocketSubmitOrders should return order %d's ID", i)
		}
		requests := venue.received("private/create-order-list")
		require.Len(t, requests, 1, "WebsocketSubmitOrders must send the orders in one list")
		assert.JSONEq(t, `{"contingency_type":"LIST","order_list":[`+tradingTestOrderParams+`,`+strings.Replace(tradingTestOrderParams, tradingTestOrderClientOrderID, "api_leg2", 1)+`]}`, string(requests[0].Params), "WebsocketSubmitOrders should send the orders in a list")
	})
	// The cases below set the mock websocket's answers and refusals, which Crypto.com cannot be made to give
	if !mockTests {
		return
	}
	t.Run("tracks margin orders", func(t *testing.T) {
		t.Parallel()
		ex, venue := newWebsocketTestExchange(t)
		marginOrder := func(clientOrderID string) *order.Submit {
			return &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: clientOrderID}
		}
		spot := marginOrder("spot_placed")
		spot.AssetType = asset.Spot
		trigger := marginOrder("margin_trigger")
		trigger.Type, trigger.Price, trigger.TriggerPrice = order.StopLimit, 49000, 49500
		var trackedWhenSent, triggerTrackedWhenSent, listTrackedWhenNextSent atomic.Bool
		venue.beforeAnswering("private/create-order-list", func() { trackedWhenSent.Store(ex.isMarginOrder("", "margin_placed")) })
		venue.beforeAnswering("private/advanced/create-order-list", func() {
			triggerTrackedWhenSent.Store(ex.isMarginOrder("", "margin_trigger"))
			listTrackedWhenNextSent.Store(ex.isMarginOrder("11", ""))
		})
		venue.respond("private/create-order-list", `[{"code":0,"index":0,"client_oid":"margin_placed","order_id":"11"},{"code":318,"index":1,"message":"EXCEEDS_MAX_ALLOWED_ORDERS"},{"code":0,"index":2,"client_oid":"spot_placed","order_id":"13"}]`)
		venue.respond("private/advanced/create-order-list", `[{"code":0,"index":0,"client_oid":"margin_trigger","order_id":"21"}]`)
		_, err := ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{marginOrder("margin_placed"), marginOrder("margin_rejected"), spot, trigger})
		require.NoError(t, err, "WebsocketSubmitOrders must not error")
		assert.True(t, trackedWhenSent.Load(), "WebsocketSubmitOrders should track a margin order by its client order ID before placing it, as its trades can arrive first")
		assert.True(t, triggerTrackedWhenSent.Load(), "WebsocketSubmitOrders should track a margin trigger order by its client order ID before placing it")
		assert.True(t, listTrackedWhenNextSent.Load(), "WebsocketSubmitOrders should track a list's placed orders by their order IDs before sending the next list")
		assert.True(t, ex.isMarginOrder("11", ""), "WebsocketSubmitOrders should track a placed margin order by its order ID")
		assert.False(t, ex.isMarginOrder("", "margin_placed"), "WebsocketSubmitOrders should stop tracking a placed margin order by its client order ID, which other orders can share, once its order ID is known")
		assert.False(t, ex.isMarginOrder("", "margin_rejected"), "WebsocketSubmitOrders should stop tracking the client order ID of an order Crypto.com rejected")
		assert.False(t, ex.isMarginOrder("13", "spot_placed"), "WebsocketSubmitOrders should track a placed spot order as no margin order")
		assert.True(t, ex.isMarginOrder("21", ""), "WebsocketSubmitOrders should track a placed margin trigger order by its order ID")

		venue.fail("private/create-order-list", &APIError{Code: 40001, Message: "BAD_REQUEST"})
		_, err = ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{marginOrder("margin_refused")})
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "WebsocketSubmitOrders must return Crypto.com's refusal of a list")
		assert.False(t, ex.isMarginOrder("", "margin_refused"), "WebsocketSubmitOrders should stop tracking the client order IDs of a list Crypto.com refused")

		// A list whose request fails otherwise may have been placed, so its orders are still tracked
		done, cancel := context.WithCancel(t.Context())
		cancel()
		// A frame can be sent with its context already ended, so its refusal waits until the request has given up on it
		answered := make(chan struct{})
		venue.beforeAnswering("private/create-order-list", func() { <-answered })
		_, err = ex.WebsocketSubmitOrders(done, []*order.Submit{marginOrder("margin_unknown")})
		close(answered)
		venue.beforeAnswering("private/create-order-list", nil)
		require.ErrorIs(t, err, context.Canceled, "WebsocketSubmitOrders must return the request's error")
		assert.True(t, ex.isMarginOrder("", "margin_unknown"), "WebsocketSubmitOrders should go on tracking the client order IDs of a list whose outcome is unknown")

		// An order an answered list gives no outcome for may have been placed, so is still tracked
		venue.fail("private/create-order-list", nil)
		venue.respond("private/create-order-list", `[{"code":0,"index":0,"client_oid":"margin_answered","order_id":"31"}]`)
		got, err := ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{marginOrder("margin_answered"), marginOrder("margin_unanswered")})
		require.NoError(t, err, "WebsocketSubmitOrders must not error for a list answered for one of its orders")
		require.Len(t, got, 2, "WebsocketSubmitOrders must return a response for every order")
		assert.ErrorIs(t, got[1].SubmissionError, errOrderListOutcomeUnknown, "WebsocketSubmitOrders should report an order the answer gives no outcome for as having an unknown outcome")
		assert.True(t, ex.isMarginOrder("", "margin_unanswered"), "WebsocketSubmitOrders should go on tracking the client order ID of an order the answer gives no outcome for")

		t.Run("acknowledgement omits client ID", func(t *testing.T) {
			t.Parallel()
			ex, venue := newWebsocketTestExchange(t)
			// A margin order is sent with the client order ID it is given, else one generated for it, and the answer names
			// neither
			given := &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: "margin_given"}
			generated := &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001}
			venue.respond("private/create-order-list", `[{"code":0,"index":0,"order_id":"41"},{"code":0,"index":1,"order_id":"42"}]`)
			_, err := ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{given, generated})
			require.NoError(t, err, "WebsocketSubmitOrders must not error for an answer without client order IDs")
			requests := venue.received("private/create-order-list")
			require.Len(t, requests, 1, "WebsocketSubmitOrders must send both orders in one list")
			var sent createOrderListParams
			require.NoError(t, json.Unmarshal(requests[0].Params, &sent), "the list's params must decode")
			require.Len(t, sent.OrderList, 2, "the list must hold both orders")
			for i, orderID := range []string{"41", "42"} {
				clientOrderID := sent.OrderList[i].ClientOrderID
				require.NotEmptyf(t, clientOrderID, "order %s must be sent with a client order ID", orderID)
				assert.Truef(t, ex.isMarginOrder(orderID, ""), "WebsocketSubmitOrders should track margin order %s by its order ID", orderID)
				assert.Falsef(t, ex.isMarginOrder("", clientOrderID), "WebsocketSubmitOrders should stop tracking client order ID %q, which the answer omits, once order %s is placed", clientOrderID, orderID)
			}
		})
		t.Run("missing acknowledgement leaves the outcome unknown", func(t *testing.T) {
			t.Parallel()
			ex, venue := newWebsocketTestExchange(t)
			venue.respond("private/create-order-list", `[{"code":0,"index":0,"order_id":"41"},{"code":318,"index":2,"message":"EXCEEDS_MAX_ALLOWED_ORDERS"}]`)
			orders := make([]*order.Submit, 0, 3)
			for _, id := range []string{"answered", "unanswered", "rejected"} {
				orders = append(orders, &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: id})
			}
			got, err := ex.WebsocketSubmitOrders(t.Context(), orders)
			require.NoError(t, err, "WebsocketSubmitOrders must not error for a list answered for some of its orders")
			require.Len(t, got, len(orders), "WebsocketSubmitOrders must return a response for every order")
			assert.Equal(t, order.New, got[0].Status, "WebsocketSubmitOrders should report the accepted order new")
			assert.NoError(t, got[0].SubmissionError, "WebsocketSubmitOrders should report no error for the accepted order")
			assert.Equal(t, order.UnknownStatus, got[1].Status, "WebsocketSubmitOrders should give the unanswered order an unknown status")
			assert.ErrorIs(t, got[1].SubmissionError, errOrderListOutcomeUnknown, "WebsocketSubmitOrders should report the unanswered order's outcome unknown")
			assert.NotErrorIs(t, got[1].SubmissionError, order.ErrUnableToPlaceOrder, "WebsocketSubmitOrders should not report the unanswered order as not placed")
			assert.Equal(t, order.Rejected, got[2].Status, "WebsocketSubmitOrders should report the refused order rejected")
			assert.ErrorIs(t, got[2].SubmissionError, order.ErrUnableToPlaceOrder, "WebsocketSubmitOrders should report the refused order not placed")
			assert.True(t, ex.isMarginOrder("", "unanswered"), "WebsocketSubmitOrders should go on tracking the unanswered order's client order ID")
			assert.False(t, ex.isMarginOrder("", "rejected"), "WebsocketSubmitOrders should stop tracking the refused order's client order ID")
		})
	})
	ex, venue := newWebsocketTestExchange(t)
	orders := []*order.Submit{testSubmit(), testTriggerSubmit(), testSubmit()}
	for _, s := range orders {
		s.Exchange = ex.Name
	}
	orders[2].ClientOrderID = "api_leg2"
	venue.respond("private/create-order-list", `[{"code":0,"index":0,"client_oid":"api_leg1","order_id":"5755600460443882762"},{"code":318,"index":1,"message":"EXCEEDS_MAX_ALLOWED_ORDERS"}]`)
	got, err := ex.WebsocketSubmitOrders(t.Context(), orders)
	require.NoError(t, err, "WebsocketSubmitOrders must not error")
	require.Len(t, got, 3, "WebsocketSubmitOrders must return a response for every order")
	clearSubmitTimes(t, got[0])
	assert.Equal(t, testSubmitResponse(t, orders[0], "5755600460443882762"), got[0], "WebsocketSubmitOrders should return the first list order's ID")
	clearSubmitTimes(t, got[1])
	assert.Equal(t, testSubmitResponse(t, orders[1], "289991951579667814"), got[1], "WebsocketSubmitOrders should return the trigger order's ID from its own list")
	assert.ErrorIs(t, got[2].SubmissionError, order.ErrUnableToPlaceOrder, "WebsocketSubmitOrders should report an order Crypto.com rejects")
	var apiErr *APIError
	require.ErrorAs(t, got[2].SubmissionError, &apiErr, "WebsocketSubmitOrders must report Crypto.com's error")
	assert.Equal(t, &APIError{Code: 318, Message: "EXCEEDS_MAX_ALLOWED_ORDERS"}, apiErr, "WebsocketSubmitOrders should report Crypto.com's code and message")
	got[2].SubmissionError = nil
	exp := &order.SubmitResponse{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testPerpetualPair, AssetType: asset.PerpetualSwap, ClientOrderID: "api_leg2", Status: order.Rejected}
	assert.Equal(t, exp, got[2], "WebsocketSubmitOrders should mark a rejected order rejected")
	requests := venue.received("private/create-order-list")
	require.Len(t, requests, 1, "WebsocketSubmitOrders must send the orders in one list")
	assert.JSONEq(t, `{"contingency_type":"LIST","order_list":[`+tradingTestOrderParams+`,`+strings.Replace(tradingTestOrderParams, tradingTestOrderClientOrderID, "api_leg2", 1)+`]}`, string(requests[0].Params), "WebsocketSubmitOrders should list the orders other than trigger orders")
	assert.Len(t, venue.received("private/advanced/create-order-list"), 1, "WebsocketSubmitOrders should send the trigger order in a list of its own")

	// A list whose request fails stops the lists after it, and the responses of those placed before it come with the error
	venue.fail("private/advanced/create-order-list", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"})
	got, err = ex.WebsocketSubmitOrders(t.Context(), orders)
	require.ErrorAs(t, err, &apiErr, "WebsocketSubmitOrders must return the error of a list whose request failed")
	require.Len(t, got, 3, "WebsocketSubmitOrders must return the orders' responses with the error")
	require.NotNil(t, got[0], "WebsocketSubmitOrders must return the response of an order placed before a list failed")
	assert.Equal(t, "5755600460443882762", got[0].OrderID, "WebsocketSubmitOrders should return the ID of an order placed before a list failed")
	assert.Nil(t, got[1], "WebsocketSubmitOrders should leave the response of an order whose list failed unset")

	// Eleven orders go in two lists, and the second failing keeps the first's responses
	orders = make([]*order.Submit, 11)
	entries := make([]string, 10)
	for i := range orders {
		orders[i] = testSubmit()
		orders[i].Exchange = ex.Name
	}
	for i := range entries {
		entries[i] = `{"code":0,"index":` + strconv.Itoa(i) + `,"order_id":"` + strconv.Itoa(100+i) + `"}`
	}
	venue.respond("private/create-order-list", "["+strings.Join(entries, ",")+"]")
	venue.beforeAnswering("private/create-order-list", func() {
		venue.fail("private/create-order-list", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"})
	})
	got, err = ex.WebsocketSubmitOrders(t.Context(), orders)
	require.ErrorAs(t, err, &apiErr, "WebsocketSubmitOrders must return the error of a later list whose request failed")
	require.Len(t, got, 11, "WebsocketSubmitOrders must return the orders' responses with the error")
	require.NotNil(t, got[9], "WebsocketSubmitOrders must return the responses of the list placed before one failed")
	assert.Equal(t, "109", got[9].OrderID, "WebsocketSubmitOrders should return the IDs of the list placed before one failed")
	assert.Nil(t, got[10], "WebsocketSubmitOrders should leave the response of an order whose list failed unset")

	// A later list's answer accepting an order without its ID keeps the responses of the lists placed before it too; the
	// cases run in order, as both set the exchange of orders[0] and WebsocketSubmitOrders is called again below
	for _, tc := range []struct {
		name   string
		orders []*order.Submit
		arm    func(*websocketVenue)
	}{
		{"the trading methods' second list", orders, func(v *websocketVenue) {
			v.beforeAnswering("private/create-order-list", func() { v.respond("private/create-order-list", `[{"code":0,"index":0}]`) })
		}},
		{"the advanced order methods' list", []*order.Submit{orders[0], testTriggerSubmit()}, func(v *websocketVenue) {
			v.respond("private/advanced/create-order-list", `[{"code":0,"index":0}]`)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, venue = newWebsocketTestExchange(t)
			for _, s := range tc.orders {
				s.Exchange = ex.Name
			}
			venue.respond("private/create-order-list", "["+strings.Join(entries, ",")+"]")
			tc.arm(venue)
			got, err = ex.WebsocketSubmitOrders(t.Context(), tc.orders)
			require.ErrorIsf(t, err, order.ErrOrderIDNotSet, "WebsocketSubmitOrders must return the error reading %s's answer", tc.name)
			require.Lenf(t, got, len(tc.orders), "WebsocketSubmitOrders must return the orders' responses with the error reading %s's answer", tc.name)
			require.NotNilf(t, got[0], "WebsocketSubmitOrders must return the responses of the list placed before %s", tc.name)
			assert.Equalf(t, "100", got[0].OrderID, "WebsocketSubmitOrders should return the IDs of the list placed before %s", tc.name)
			assert.Nilf(t, got[len(got)-1], "WebsocketSubmitOrders should leave the response of the order in %s unset, as its answer cannot be read", tc.name)
		})
	}

	// Every order is checked before any list is sent, so an order Crypto.com would refuse stops the lists before its own,
	// and is named by its index among the orders
	ex, venue = newWebsocketTestExchange(t)
	marginOrders := make([]*order.Submit, 11)
	for i := range marginOrders {
		marginOrders[i] = &order.Submit{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: testSpotPair, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: "margin_" + strconv.Itoa(i)}
	}
	marginOrders[10].ClientOrderID = strings.Repeat("c", 37)
	noTriggerPrice := testTriggerSubmit()
	noTriggerPrice.Exchange, noTriggerPrice.TriggerPrice = ex.Name, 0
	// The cases share ex and its venue in order, whose requests are checked after them
	for _, tc := range []struct {
		name   string
		orders []*order.Submit
		err    error
		named  string
	}{
		{"ten orders then one with a client order ID of 37 characters", marginOrders, errClientOrderIDTooLong, "order 10: "},
		{"an order then a trigger order without a trigger price", []*order.Submit{marginOrders[0], noTriggerPrice}, errInvalidReferencePrice, "order 1: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err = ex.WebsocketSubmitOrders(t.Context(), tc.orders)
			require.ErrorIsf(t, err, tc.err, "WebsocketSubmitOrders must refuse %s", tc.name)
			assert.ErrorContainsf(t, err, tc.named, "WebsocketSubmitOrders should name the order it refuses by its index among %s", tc.name)
			for _, s := range tc.orders {
				assert.Falsef(t, ex.isMarginOrder("", s.ClientOrderID), "WebsocketSubmitOrders should not track client order ID %q of %s, sending none of them", s.ClientOrderID, tc.name)
			}
		})
	}
	assert.Empty(t, venue.received("private/create-order-list"), "WebsocketSubmitOrders should send no list of orders holding one it refuses")
	assert.Empty(t, venue.received("private/advanced/create-order-list"), "WebsocketSubmitOrders should send no list of trigger orders beside one it refuses")

	// An isolated margin order is listed on the instrument's isolated position, which the recorded positions hold
	ex, venue = newWebsocketTestExchange(t)
	isolated := testSubmit()
	isolated.Exchange, isolated.MarginType = ex.Name, margin.Isolated
	venue.respond("private/create-order-list", `[{"code":0,"index":0,"order_id":"1"}]`)
	_, err = ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{isolated})
	require.NoError(t, err, "WebsocketSubmitOrders must not error for an isolated margin order")
	requests = venue.received("private/create-order-list")
	require.Len(t, requests, 1, "WebsocketSubmitOrders must send the isolated margin order in a list")
	isolatedParams := strings.Replace(tradingTestOrderParams, `"exec_inst":["POST_ONLY"]`, `"exec_inst":["POST_ONLY","ISOLATED_MARGIN"],"isolation_id":"19848526"`, 1)
	assert.JSONEq(t, `{"contingency_type":"LIST","order_list":[`+isolatedParams+`]}`, string(requests[0].Params), "WebsocketSubmitOrders should list an isolated margin order on the instrument's isolated position")

	// With the instrument's isolated position open, every isolated margin order on it is listed on that position
	ex, venue = newWebsocketTestExchange(t)
	first, second := testSubmit(), testSubmit()
	first.Exchange, first.MarginType = ex.Name, margin.Isolated
	second.Exchange, second.MarginType, second.ClientOrderID = ex.Name, margin.Isolated, "api_leg2"
	venue.respond("private/create-order-list", `[{"code":0,"index":0,"order_id":"1"},{"code":0,"index":1,"order_id":"2"}]`)
	_, err = ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{first, second})
	require.NoError(t, err, "WebsocketSubmitOrders must not error for isolated margin orders on an open isolated position")
	requests = venue.received("private/create-order-list")
	require.Len(t, requests, 1, "WebsocketSubmitOrders must send both isolated margin orders in a list")
	assert.JSONEq(t, `{"contingency_type":"LIST","order_list":[`+isolatedParams+`,`+strings.Replace(isolatedParams, tradingTestOrderClientOrderID, "api_leg2", 1)+`]}`, string(requests[0].Params), "WebsocketSubmitOrders should list every isolated margin order on the instrument's open isolated position")

	// Crypto.com opens a new isolated position for each order naming none, and refuses a second on one instrument, so a
	// second isolated margin order on an instrument without one is refused before any order is sent
	routesEx, lookedUp := newRoutesServerExchange(t, map[string]testRoute{"private/get-positions": testResult(`{"data":[]}`)})
	dependent := []*order.Submit{testSubmit(), testSubmit(), testSubmit()}
	for i, pair := range []currency.Pair{testPerpetualPair, perpETHUSD, testPerpetualPair} {
		dependent[i].Exchange, dependent[i].Pair, dependent[i].MarginType, dependent[i].ClientOrderID = routesEx.Name, pair, margin.Isolated, "isolated_"+strconv.Itoa(i)
	}
	_, err = routesEx.WebsocketSubmitOrders(t.Context(), dependent)
	require.ErrorIs(t, err, errIsolatedPositionNotOpen, "WebsocketSubmitOrders must refuse a second isolated margin order on an instrument without an isolated position")
	assert.ErrorContains(t, err, "order 2: isolated position not open yet: order 0 opens the one on BTCUSD-PERP", "WebsocketSubmitOrders should name the order it refuses and the order opening the position")
	expLookups := []testRequest{
		{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`},
		{Method: "private/get-positions", Params: `{"instrument_name":"ETHUSD-PERP"}`},
		{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`},
	}
	assert.Equal(t, expLookups, lookedUp(), "WebsocketSubmitOrders should look up each isolated margin order's position and place none")

	// The refusal spans the lists the orders are sent in, and covers a reduce-only close, which names no position either
	ex, lookedUp = newRoutesServerExchange(t, map[string]testRoute{"private/get-positions": testResult(`{"data":[]}`)})
	venue = newWebsocketVenue(t)
	connectWebsocketVenue(t, ex, venue)
	acrossLists := make([]*order.Submit, 11)
	for i := range acrossLists {
		acrossLists[i] = testSubmit()
		acrossLists[i].Exchange, acrossLists[i].ClientOrderID = ex.Name, "order_"+strconv.Itoa(i)
	}
	opener, closer := acrossLists[0], acrossLists[10]
	opener.Type, opener.Price, opener.TimeInForce, opener.MarginType = order.Market, 0, order.UnknownTIF, margin.Isolated
	closer.Type, closer.Side, closer.Price, closer.TimeInForce, closer.ReduceOnly, closer.MarginType = order.Market, order.Sell, 0, order.UnknownTIF, true, margin.Isolated
	responses, err := ex.WebsocketSubmitOrders(t.Context(), acrossLists)
	require.ErrorIs(t, err, errIsolatedPositionNotOpen, "WebsocketSubmitOrders must refuse an isolated margin close in a later list than the order opening its position")
	assert.ErrorContains(t, err, "order 10: isolated position not open yet: order 0 opens the one on BTCUSD-PERP", "WebsocketSubmitOrders should name the close and the order opening the position")
	assert.Nil(t, responses, "WebsocketSubmitOrders should return no responses for orders refused before any list is sent")
	expLookups = []testRequest{
		{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`},
		{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`},
	}
	assert.Equal(t, expLookups, lookedUp(), "WebsocketSubmitOrders should look up the isolated position of each isolated margin order only")
	assert.Empty(t, venue.received("private/create-order-list"), "WebsocketSubmitOrders should send no list of orders holding one it refuses")

	// An isolated margin order whose isolated position cannot be looked up is refused before any order is sent, as it could
	// open a second one
	unknownEx, unknownLookups := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-positions": func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} },
	})
	unknown := []*order.Submit{testSubmit(), testSubmit()}
	for _, s := range unknown {
		s.Exchange = unknownEx.Name
	}
	unknown[1].MarginType = margin.Isolated
	got, err = unknownEx.WebsocketSubmitOrders(t.Context(), unknown)
	require.ErrorAs(t, err, &apiErr, "WebsocketSubmitOrders must return the error looking up an isolated margin order's position")
	assert.Equal(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "WebsocketSubmitOrders should return Crypto.com's error looking up the position")
	assert.ErrorContains(t, err, "order 1: ", "WebsocketSubmitOrders should name the order whose position it could not look up by its index")
	assert.Nil(t, got, "WebsocketSubmitOrders should return no responses, sending no order")
	assert.Equal(t, []testRequest{{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`}}, unknownLookups(), "WebsocketSubmitOrders should look up only the isolated margin order's position")
}

func TestListSubmitResponses(t *testing.T) {
	t.Parallel()
	orders := []*order.Submit{testSubmit(), testSubmit()}
	responses := make([]*order.SubmitResponse, 2)
	require.NoError(t, listSubmitResponses(orders, []int{1}, []OrderListEntryResponse{{Index: 3}}, responses), "listSubmitResponses must not error for an entry out of range")
	assert.Nil(t, responses[0], "listSubmitResponses should only set the responses of the list's orders")
	assert.ErrorIs(t, responses[1].SubmissionError, errOrderListOutcomeUnknown, "listSubmitResponses should report an order without an entry as having an unknown outcome")
	assert.Equal(t, order.UnknownStatus, responses[1].Status, "listSubmitResponses should give an order without an entry an unknown status")
	t.Run("entries indexed at the list's length or below zero", func(t *testing.T) {
		t.Parallel()
		outside := make([]*order.SubmitResponse, 2)
		require.NoError(t, listSubmitResponses([]*order.Submit{testSubmit(), testSubmit()}, []int{1}, []OrderListEntryResponse{{Index: 1, Code: 306}, {Index: -1, Code: 306}}, outside), "listSubmitResponses must not error for entries outside the list")
		require.NotNil(t, outside[1], "listSubmitResponses must set the response of the list's order")
		assert.Equal(t, order.UnknownStatus, outside[1].Status, "listSubmitResponses should ignore entries outside the list")
	})
	err := listSubmitResponses(orders, []int{0}, []OrderListEntryResponse{{Index: 0}}, responses)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "listSubmitResponses should refuse an accepted entry without an order ID")
	assert.Nil(t, responses[0], "listSubmitResponses should leave a list's responses unset when it cannot read an entry")

	// The orders after an entry it cannot read may have been placed, so none of the list is reported rejected
	orders = []*order.Submit{testSubmit(), testSubmit(), testSubmit()}
	responses = make([]*order.SubmitResponse, 3)
	err = listSubmitResponses(orders, []int{0, 1, 2}, []OrderListEntryResponse{{Index: 0, OrderID: "100"}, {Index: 1}, {Index: 2, OrderID: "102"}}, responses)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "listSubmitResponses should refuse an accepted entry without an order ID among others")
	assert.Equal(t, make([]*order.SubmitResponse, 3), responses, "listSubmitResponses should leave every response of a list unset when it cannot read one of its entries")
}

func TestNewOrderAmendment(t *testing.T) { //nolint:tparallel // the cases run in order, the refusals on one shared exchange
	t.Parallel()
	cro := currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter)
	notFound := func(string) (string, *APIError) { return "", &APIError{Code: 40401, Message: "NOT_FOUND"} }
	const limitOrder = `{"order_id":"1","order_type":"LIMIT","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"],"side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD"}`
	orderLookup := []testRequest{{Method: "private/get-order-detail", Params: `{"order_id":"1"}`}}
	advancedLookup := []testRequest{
		{Method: "private/get-order-detail", Params: `{"order_id":"1"}`},
		{Method: "private/advanced/get-order-detail", Params: `{"order_id":"1"}`},
	}
	// The cases run in order, before the refusals below call newOrderAmendment too
	for _, tc := range []struct {
		name     string
		m        order.Modify
		routes   map[string]testRoute
		exp      *orderAmendment
		requests []testRequest
	}{
		{
			name: "a limit order given its price and quantity",
			m:    order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.02},
			exp:  &orderAmendment{order: &AmendOrderRequest{OrderID: "1", NewPrice: 77000, NewQuantity: 0.02}, price: 77000, amount: 0.02},
		},
		{
			name: "a limit order named by its client order ID and given a new one",
			m:    order.Modify{ClientOrderID: "original", NewClientOrderID: "renamed", Type: order.Limit, AssetType: asset.PerpetualSwap, Pair: testPerpetualPair, Price: 77000, Amount: 0.02},
			exp:  &orderAmendment{order: &AmendOrderRequest{OriginalClientOrderID: "original", ClientOrderID: "renamed", NewPrice: 77000, NewQuantity: 0.02}, price: 77000, amount: 0.02},
		},
		{
			name: "a stop limit order given its prices and quantity",
			m:    order.Modify{OrderID: "5755600460443882762", Type: order.StopLimit, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279},
			exp:  &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0.0278)), NewReferencePrice: 0.0279}, price: 0.0278, amount: 10},
		},
		{
			name: "a stop limit order named by its client order ID",
			m:    order.Modify{ClientOrderID: "stop", Type: order.StopLimit, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279},
			exp:  &orderAmendment{trigger: &AmendAdvancedOrderRequest{ClientOrderID: "stop", NewQuantity: 10, NewPrice: new(types.Number(0.0278)), NewReferencePrice: 0.0279}, price: 0.0278, amount: 10},
		},
		{
			name: "a stop order given a limit price, which would make it a stop limit order",
			m:    order.Modify{OrderID: "5755600460443882762", Type: order.Stop, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279},
			exp:  &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0)), NewReferencePrice: 0.0279}, amount: 10},
		},
		{
			name: "a stop market order given its quantity alone",
			m:    order.Modify{OrderID: "5755600460443882762", Type: order.StopMarket, AssetType: asset.Spot, Pair: cro, Amount: 10, TriggerPrice: 0.0279},
			exp:  &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0)), NewReferencePrice: 0.0279}, amount: 10},
		},
		{
			name: "a take profit order given its quantity alone",
			m:    order.Modify{OrderID: "5755600460443882762", Type: order.TakeProfit, AssetType: asset.Spot, Pair: cro, Amount: 10, TriggerPrice: 0.0281},
			exp:  &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0)), NewReferencePrice: 0.0281}, amount: 10},
		},
		{
			name: "a take profit market order given a limit price, which would make it a take profit limit order",
			m:    order.Modify{OrderID: "5755600460443882762", Type: order.TakeProfitMarket, AssetType: asset.Spot, Pair: cro, Price: 0.0282, Amount: 10, TriggerPrice: 0.0281},
			exp:  &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0)), NewReferencePrice: 0.0281}, amount: 10},
		},
		{
			name:     "a limit order given its price, its quantity restated",
			m:        order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000},
			routes:   map[string]testRoute{"private/get-order-detail": testResult(limitOrder)},
			exp:      &orderAmendment{order: &AmendOrderRequest{OrderID: "1", NewPrice: 77000, NewQuantity: 0.01}, price: 77000, amount: 0.01, timeInForce: order.GoodTillCancel | order.PostOnly},
			requests: orderLookup,
		},
		{
			name:     "a limit order given its quantity, its price restated",
			m:        order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Amount: 0.02},
			routes:   map[string]testRoute{"private/get-order-detail": testResult(limitOrder)},
			exp:      &orderAmendment{order: &AmendOrderRequest{OrderID: "1", NewPrice: 78000, NewQuantity: 0.02}, price: 78000, amount: 0.02, timeInForce: order.GoodTillCancel | order.PostOnly},
			requests: orderLookup,
		},
		{
			name:     "a limit order given its price, quantity and own time in force",
			m:        order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.02, TimeInForce: order.PostOnly},
			routes:   map[string]testRoute{"private/get-order-detail": testResult(limitOrder)},
			exp:      &orderAmendment{order: &AmendOrderRequest{OrderID: "1", NewPrice: 77000, NewQuantity: 0.02}, price: 77000, amount: 0.02, timeInForce: order.GoodTillCancel | order.PostOnly},
			requests: orderLookup,
		},
		{
			name: "a stop loss order modified without its type, which only the advanced order methods hold",
			m:    order.Modify{OrderID: "1", AssetType: asset.Spot, Pair: cro, Price: 0.0279, Amount: 10},
			routes: map[string]testRoute{
				"private/get-order-detail":          notFound,
				"private/advanced/get-order-detail": testResult(`{"order_id":"1","order_type":"STOP_LOSS","side":"SELL","quantity":"10","limit_price":"0","ref_price":"0.028","status":"ACTIVE","instrument_name":"CRO_USD"}`),
			},
			exp:      &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "1", NewQuantity: 10, NewPrice: new(types.Number(0))}, amount: 10, orderType: order.Stop},
			requests: advancedLookup,
		},
		{
			name:     "a take profit order modified as any type, which the trading methods report",
			m:        order.Modify{OrderID: "1", Type: order.AnyType, AssetType: asset.Spot, Pair: cro, Price: 0.0279, Amount: 10},
			routes:   map[string]testRoute{"private/get-order-detail": testResult(`{"order_id":"1","order_type":"TAKE_PROFIT","side":"SELL","quantity":"10","limit_price":"0","ref_price":"0.03","status":"ACTIVE","instrument_name":"CRO_USD"}`)},
			exp:      &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "1", NewQuantity: 10, NewPrice: new(types.Number(0))}, amount: 10, orderType: order.TakeProfit},
			requests: orderLookup,
		},
		{
			name: "an order list's limit leg, which only the advanced order methods hold",
			m:    order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000},
			routes: map[string]testRoute{
				"private/get-order-detail":          notFound,
				"private/advanced/get-order-detail": testResult(`{"order_id":"1","order_type":"LIMIT","side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD"}`),
			},
			exp:      &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "1", NewQuantity: 0.01, NewPrice: new(types.Number(77000))}, price: 77000, amount: 0.01},
			requests: advancedLookup,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, received := newRoutesServerExchange(t, tc.routes)
			got, err := ex.newOrderAmendment(t.Context(), &tc.m)
			require.NoErrorf(t, err, "newOrderAmendment must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newOrderAmendment should amend %s", tc.name)
			assert.Equalf(t, tc.requests, received(), "newOrderAmendment should send the order lookups expected for %s", tc.name)
		})
	}

	refuser, _ := newRoutesServerExchange(t, nil)
	answering := func(detail string) *Exchange {
		ex, _ := newRoutesServerExchange(t, map[string]testRoute{"private/get-order-detail": testResult(detail)})
		return ex
	}
	// The refusals share refuser in order
	for _, tc := range []struct {
		name string
		ex   *Exchange
		m    *order.Modify
		err  error
	}{
		{"a nil modification", refuser, nil, order.ErrModifyOrderIsNil},
		{"an unsupported asset", refuser, &order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Options, Pair: testPerpetualPair, Price: 77000, Amount: 0.02}, asset.ErrNotSupported},
		{"an order whose lookup fails", newUnavailableServerExchange(t), &order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000}, request.ErrBadStatus},
		{"an order of an unknown type", answering(strings.Replace(limitOrder, `"LIMIT"`, `"TRAILING_STOP"`, 1)), &order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000}, order.ErrUnrecognisedOrderType},
		{"an order of an unknown time in force", answering(strings.Replace(limitOrder, `"GOOD_TILL_CANCEL"`, `"GOOD_TILL_DATE"`, 1)), &order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000}, order.ErrInvalidTimeInForce},
		{"a time in force other than the order's", answering(limitOrder), &order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.02, TimeInForce: order.ImmediateOrCancel}, errTimeInForceNotAmendable},
		{"a new client order ID for a trigger order", refuser, &order.Modify{OrderID: "5755600460443882762", NewClientOrderID: "renamed", Type: order.StopLimit, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279}, errClientOrderIDNotAmendable},
		{"a trigger price for a limit order", refuser, &order.Modify{OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.02, TriggerPrice: 78000}, errTriggerPriceUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.ex.newOrderAmendment(t.Context(), tc.m)
			assert.ErrorIsf(t, err, tc.err, "newOrderAmendment should refuse %s", tc.name)
		})
	}
}

func TestModifyOrder(t *testing.T) { //nolint:tparallel // the cases run in order, as ModifyOrder is called again after them
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must refuse a nil modification")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{Pair: testPerpetualPair, AssetType: asset.Options, OrderID: "1"})
	require.ErrorIs(t, err, asset.ErrNotSupported, "ModifyOrder must refuse an unsupported asset")
	cro := currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter)
	ctx := t.Context()
	t.Run("an order amended", func(t *testing.T) {
		skipLiveUnlessCanManipulate(t)
		m := &order.Modify{Exchange: e.Name, OrderID: tradingTestOrderID, Type: order.Limit, Side: order.Buy, AssetType: asset.PerpetualSwap, Pair: testPerpetualPair, Price: 50000, Amount: 0.0002}
		if !mockTests {
			m.OrderID = placeTradingTestOrder(t)
			cancelTradingOrderAfterTest(t, m.OrderID)
		}
		got, err := e.ModifyOrder(ctx, m)
		require.NoError(t, err, "ModifyOrder must not error")
		if !mockTests {
			// Crypto.com answers an amend with the order ID it was given, as its documented example shows
			assert.Equal(t, m.OrderID, got.OrderID, "ModifyOrder should return the amended order's ID")
			return
		}
		exp := &order.ModifyResponse{Exchange: e.Name, OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID, Pair: testPerpetualPair, Type: order.Limit, Side: order.Buy, AssetType: asset.PerpetualSwap, Price: 50000, Amount: 0.0002}
		assert.Equal(t, exp, got, "ModifyOrder should amend the order's price and quantity")

		m = &order.Modify{Exchange: e.Name, OrderID: "5755600460443882762", Type: order.StopLimit, Side: order.Sell, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279}
		got, err = e.ModifyOrder(ctx, m)
		require.NoError(t, err, "ModifyOrder must not error for a trigger order")
		exp = &order.ModifyResponse{Exchange: e.Name, OrderID: "5755600460443882762", ClientOrderID: tradingTestOrderClientOrderID, Pair: cro, Type: order.StopLimit, Side: order.Sell, AssetType: asset.Spot, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279}
		assert.Equal(t, exp, got, "ModifyOrder should amend a trigger order with the advanced order methods")
	})

	// An OTO list's limit leg, which only the advanced order methods know, has its quantity restated
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail":          func(string) (string, *APIError) { return "", &APIError{Code: 40401, Message: "NOT_FOUND"} },
		"private/advanced/get-order-detail": testResult(`{"order_id":"5755600460443890101","order_type":"LIMIT","side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD"}`),
		"private/advanced/amend-order":      testResult(`{"order_id":"5755600460443890101","client_oid":"1792836351900"}`),
	})
	m := &order.Modify{Exchange: ex.Name, OrderID: "5755600460443890101", AssetType: asset.Spot, Pair: testSpotPair, Price: 77000}
	got, err := ex.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error for an order list's leg")
	exp := &order.ModifyResponse{Exchange: ex.Name, OrderID: "5755600460443890101", ClientOrderID: "1792836351900", Pair: testSpotPair, Type: order.Limit, AssetType: asset.Spot, Price: 77000, Amount: 0.01}
	assert.Equal(t, exp, got, "ModifyOrder should restate the leg's quantity, and report its type")
	expRequests := []testRequest{
		{Method: "private/get-order-detail", Params: `{"order_id":"5755600460443890101"}`},
		{Method: "private/advanced/get-order-detail", Params: `{"order_id":"5755600460443890101"}`},
		{Method: "private/advanced/amend-order", Params: `{"order_id":"5755600460443890101","new_quantity":"0.01","new_price":"77000"}`},
	}
	assert.Equal(t, expRequests, received(), "ModifyOrder should look an order up among the advanced orders and amend it with the advanced order methods")

	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 1, Amount: 1, TriggerPrice: 1})
	assert.ErrorIs(t, err, errTriggerPriceUnsupported, "ModifyOrder should refuse a trigger price for an order other than a trigger order")

	// An order fetched to restate its price reports its own time in force, which an amendment cannot change
	ex, _ = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail": testResult(`{"order_id":"1","order_type":"LIMIT","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"],"side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD"}`),
		"private/amend-order":      testResult(`{"order_id":"1"}`),
	})
	m = &order.Modify{Exchange: ex.Name, OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Amount: 0.02, TimeInForce: order.GoodTillCancel | order.PostOnly}
	got, err = ex.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error for the order's own time in force")
	assert.Equal(t, order.GoodTillCancel|order.PostOnly, got.TimeInForce, "ModifyOrder should report the order's time in force")
	m.TimeInForce = order.ImmediateOrCancel
	_, err = ex.ModifyOrder(t.Context(), m)
	assert.ErrorIs(t, err, errTimeInForceNotAmendable, "ModifyOrder should refuse to change an order's time in force")

	// With its quantity and price given, the order is still fetched for the time in force the modification gives, which
	// may name Crypto.com's default or leave it to be taken
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail": testResult(`{"order_id":"1","order_type":"LIMIT","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"],"side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD"}`),
		"private/amend-order":      testResult(`{"order_id":"1"}`),
	})
	m = &order.Modify{Exchange: ex.Name, OrderID: "1", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.02, TimeInForce: order.ImmediateOrCancel}
	_, err = ex.ModifyOrder(t.Context(), m)
	assert.ErrorIs(t, err, errTimeInForceNotAmendable, "ModifyOrder should refuse to change an order's time in force with its quantity and price given")
	assert.Equal(t, []testRequest{{Method: "private/get-order-detail", Params: `{"order_id":"1"}`}}, received(), "ModifyOrder should fetch the order to compare its time in force, and send no amendment changing it")
	m.TimeInForce = order.PostOnly
	got, err = ex.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error for a post only time in force, which Crypto.com places good till cancelled")
	assert.Equal(t, order.GoodTillCancel|order.PostOnly, got.TimeInForce, "ModifyOrder should report the order's time in force")
	m.TimeInForce = order.UnknownTIF
	_, err = ex.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error without a time in force")
	amendment := testRequest{Method: "private/amend-order", Params: `{"order_id":"1","new_price":"77000","new_quantity":"0.02"}`}
	expRequests = []testRequest{
		{Method: "private/get-order-detail", Params: `{"order_id":"1"}`},
		{Method: "private/get-order-detail", Params: `{"order_id":"1"}`},
		amendment,
		amendment,
	}
	assert.Equal(t, expRequests, received(), "ModifyOrder should fetch the order for a time in force given, and not with its quantity and price given without one")

	// With its quantity and price given, the leg is amended at once, and by the advanced order methods once the trading
	// methods refuse it
	notFound := func(string) (string, *APIError) { return "", &APIError{Code: 212, Message: "INVALID_ORDERID"} }
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/amend-order":          notFound,
		"private/advanced/amend-order": testResult(`{"order_id":"5755600460443890101","client_oid":"1792836351900"}`),
	})
	m = &order.Modify{Exchange: ex.Name, OrderID: "5755600460443890101", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.01}
	got, err = ex.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error for an order list's leg the trading methods refuse")
	exp = &order.ModifyResponse{Exchange: ex.Name, OrderID: "5755600460443890101", ClientOrderID: "1792836351900", Pair: testSpotPair, Type: order.Limit, AssetType: asset.Spot, Price: 77000, Amount: 0.01}
	assert.Equal(t, exp, got, "ModifyOrder should amend the leg with the advanced order methods")
	requests := received()
	require.Len(t, requests, 2, "ModifyOrder must try the trading methods, then the advanced order methods")
	assert.Equal(t, "private/amend-order", requests[0].Method, "ModifyOrder should first amend with the trading methods")
	assert.Equal(t, "private/advanced/amend-order", requests[1].Method, "ModifyOrder should then amend with the advanced order methods")
	assert.JSONEq(t, `{"order_id":"5755600460443890101","new_quantity":"0.01","new_price":"77000"}`, requests[1].Params, "ModifyOrder should restate the amendment for the advanced order methods")

	// An order modified without its type, as the order manager modifies one, is amended as the type it has, which is
	// reported, and a trigger market order is sent no limit price, which would make it a trigger limit order; the cases
	// and their modification types run in order, as ModifyOrder is called again below
	for orderType, exp := range map[string]order.Type{"STOP_LOSS": order.Stop, "TAKE_PROFIT": order.TakeProfit} {
		t.Run(orderType, func(t *testing.T) {
			for _, typ := range []order.Type{order.UnknownType, order.AnyType} {
				t.Run(typ.String(), func(t *testing.T) {
					ex, received = newRoutesServerExchange(t, map[string]testRoute{
						"private/get-order-detail":          notFound,
						"private/advanced/get-order-detail": testResult(`{"order_id":"1","order_type":"` + orderType + `","side":"SELL","quantity":"10","limit_price":"0","ref_price":"0.028","status":"ACTIVE","instrument_name":"CRO_USD"}`),
						"private/advanced/amend-order":      testResult(`{"order_id":"1"}`),
					})
					got, err = ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, OrderID: "1", Type: typ, AssetType: asset.Spot, Pair: cro, Price: 0.0279, Amount: 10})
					require.NoErrorf(t, err, "ModifyOrder must not error for a %s order modified with type %s", orderType, typ)
					requests = received()
					require.Lenf(t, requests, 3, "ModifyOrder must look a %s order up among the advanced orders, then amend it, for type %s", orderType, typ)
					assert.Equalf(t, testRequest{Method: "private/advanced/amend-order", Params: `{"order_id":"1","new_quantity":"10","new_price":"0"}`}, requests[2], "ModifyOrder should send no limit price for a %s order modified with type %s", orderType, typ)
					assert.Equalf(t, exp, got.Type, "ModifyOrder should report a %s order's own type for a modification with type %s", orderType, typ)
				})
			}
		})
	}

	// An order is amended by its order ID or its client order ID, and takes any new client order ID the modification
	// gives; the cases and their assets run in order, as ModifyOrder is called again below
	for _, tc := range []struct {
		name string
		m    order.Modify
		exp  string
	}{
		{"named by its client order ID", order.Modify{ClientOrderID: "original"}, `{"orig_client_oid":"original","new_price":"77000","new_quantity":"0.01"}`},
		{"named by its client order ID and given a new one", order.Modify{ClientOrderID: "original", NewClientOrderID: "renamed"}, `{"orig_client_oid":"original","client_oid":"renamed","new_price":"77000","new_quantity":"0.01"}`},
		{"named by its order ID and given a client order ID", order.Modify{OrderID: "1", NewClientOrderID: "renamed"}, `{"order_id":"1","client_oid":"renamed","new_price":"77000","new_quantity":"0.01"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for a, p := range map[asset.Item]currency.Pair{asset.Spot: testSpotPair, asset.PerpetualSwap: testPerpetualPair} {
				t.Run(a.String(), func(t *testing.T) {
					ex, received = newRoutesServerExchange(t, map[string]testRoute{"private/amend-order": testResult(`{"order_id":"1"}`)})
					m := tc.m
					m.Exchange, m.Type, m.AssetType, m.Pair, m.Price, m.Amount = ex.Name, order.Limit, a, p, 77000, 0.01
					_, err = ex.ModifyOrder(t.Context(), &m)
					require.NoErrorf(t, err, "ModifyOrder must not error for a %s order %s", a, tc.name)
					assert.Equalf(t, []testRequest{{Method: "private/amend-order", Params: tc.exp}}, received(), "ModifyOrder should amend a %s order %s", a, tc.name)
				})
			}
		})
	}

	// The advanced order methods' amend takes a client order ID only to name the order, so neither a trigger order nor an
	// order the trading methods refuse is given a new one
	ex, received = newRoutesServerExchange(t, map[string]testRoute{"private/amend-order": notFound})
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, OrderID: "5755600460443882762", NewClientOrderID: "renamed", Type: order.StopLimit, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279})
	assert.ErrorIs(t, err, errClientOrderIDNotAmendable, "ModifyOrder should refuse a new client order ID for a trigger order")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, OrderID: "5755600460443890101", NewClientOrderID: "renamed", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.01})
	assert.ErrorIs(t, err, errClientOrderIDNotAmendable, "ModifyOrder should refuse a new client order ID for an order the trading methods refuse")
	expRequests = []testRequest{{Method: "private/amend-order", Params: `{"order_id":"5755600460443890101","client_oid":"renamed","new_price":"77000","new_quantity":"0.01"}`}}
	assert.Equal(t, expRequests, received(), "ModifyOrder should not amend an order it is to rename with the advanced order methods")

	// A trigger order, and an order the trading methods refuse, are named to the advanced order methods by the client order
	// ID the modification names them by
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/amend-order":          notFound,
		"private/advanced/amend-order": testResult(`{"order_id":"1"}`),
	})
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, ClientOrderID: "stop", Type: order.StopLimit, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279})
	require.NoError(t, err, "ModifyOrder must not error for a trigger order named by its client order ID")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, ClientOrderID: "leg", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.01})
	require.NoError(t, err, "ModifyOrder must not error for an order list's leg named by its client order ID")
	expRequests = []testRequest{
		{Method: "private/advanced/amend-order", Params: `{"client_oid":"stop","new_quantity":"10","new_price":"0.0278","new_ref_price":"0.0279"}`},
		{Method: "private/amend-order", Params: `{"orig_client_oid":"leg","new_price":"77000","new_quantity":"0.01"}`},
		{Method: "private/advanced/amend-order", Params: `{"client_oid":"leg","new_quantity":"0.01","new_price":"77000"}`},
	}
	assert.Equal(t, expRequests, received(), "ModifyOrder should name an order to the advanced order methods by its client order ID")

	// A trigger order the trading methods' order detail reports is amended by the advanced order methods
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail":     testResult(`{"order_id":"1","order_type":"STOP_LOSS","side":"SELL","quantity":"10","limit_price":"0","ref_price":"0.028","status":"ACTIVE","instrument_name":"CRO_USD"}`),
		"private/advanced/amend-order": testResult(`{"order_id":"1"}`),
	})
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, OrderID: "1", AssetType: asset.Spot, Pair: cro, Price: 0.0279, Amount: 10})
	require.NoError(t, err, "ModifyOrder must not error for a trigger order the trading methods report")
	expRequests = []testRequest{
		{Method: "private/get-order-detail", Params: `{"order_id":"1"}`},
		{Method: "private/advanced/amend-order", Params: `{"order_id":"1","new_quantity":"10","new_price":"0"}`},
	}
	assert.Equal(t, expRequests, received(), "ModifyOrder should amend a trigger order the trading methods report with the advanced order methods")

	ex, _ = newRoutesServerExchange(t, map[string]testRoute{"private/amend-order": notFound, "private/advanced/amend-order": notFound})
	_, err = ex.ModifyOrder(t.Context(), m)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "ModifyOrder must return Crypto.com's error when neither amend accepts the order")
	assert.Equal(t, int64(212), apiErr.Code, "ModifyOrder should return Crypto.com's code")
	_, err = newUnavailableServerExchange(t).ModifyOrder(t.Context(), m)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ModifyOrder should return a failed request's error without falling back")

	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/amend-order": func(string) (string, *APIError) {
			return "", &APIError{Code: codeTooManyRequests, Message: "TOO_MANY_REQUESTS"}
		},
		"private/advanced/amend-order": notFound,
	})
	_, err = ex.ModifyOrder(t.Context(), m)
	require.ErrorAs(t, err, &apiErr, "ModifyOrder must return Crypto.com's refusal of the request")
	assert.Equal(t, int64(codeTooManyRequests), apiErr.Code, "ModifyOrder should return the rate limit's code")
	assert.Len(t, received(), 1, "ModifyOrder should not retry a request Crypto.com refused as a whole with the advanced order methods")
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "WebsocketModifyOrder must refuse a nil modification")
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t)
	m := &order.Modify{Exchange: ex.Name, OrderID: tradingTestOrderID, Type: order.Limit, Side: order.Buy, AssetType: asset.PerpetualSwap, Pair: testPerpetualPair, Price: 50000, Amount: 0.0002}
	if !mockTests {
		m.OrderID = placeTradingTestOrder(t)
		cancelTradingOrderAfterTest(t, m.OrderID)
	}
	got, err := ex.WebsocketModifyOrder(t.Context(), m)
	require.NoError(t, err, "WebsocketModifyOrder must not error")
	if !mockTests {
		// Crypto.com answers an amend with the order ID it was given, as its documented example shows
		assert.Equal(t, m.OrderID, got.OrderID, "WebsocketModifyOrder should return the amended order's ID")
		return
	}
	exp := &order.ModifyResponse{Exchange: ex.Name, OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID, Pair: testPerpetualPair, Type: order.Limit, Side: order.Buy, AssetType: asset.PerpetualSwap, Price: 50000, Amount: 0.0002}
	assert.Equal(t, exp, got, "WebsocketModifyOrder should amend the order's price and quantity")
	requests := venue.received("private/amend-order")
	require.Len(t, requests, 1, "WebsocketModifyOrder must send one amendment")
	assert.JSONEq(t, `{"order_id":"6530219466236720401","new_price":"50000","new_quantity":"0.0002"}`, string(requests[0].Params), "WebsocketModifyOrder should send the amendment")

	m.Type, m.OrderID, m.TriggerPrice = order.Stop, "5755600460443882762", 0.0279
	got, err = ex.WebsocketModifyOrder(t.Context(), m)
	require.NoError(t, err, "WebsocketModifyOrder must not error for a trigger order")
	assert.Zero(t, got.Price, "WebsocketModifyOrder should keep a stop market order without a limit price")
	requests = venue.received("private/advanced/amend-order")
	require.Len(t, requests, 1, "WebsocketModifyOrder must amend a trigger order with the advanced order methods")
	assert.JSONEq(t, `{"order_id":"5755600460443882762","new_quantity":"0.0002","new_price":"0","new_ref_price":"0.0279"}`, string(requests[0].Params), "WebsocketModifyOrder should amend a stop market order without a limit price, which would make it a stop limit order")

	venue.fail("private/advanced/amend-order", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"})
	_, err = ex.WebsocketModifyOrder(t.Context(), m)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "WebsocketModifyOrder must return Crypto.com's refusal")
	assert.Equal(t, int64(50001), apiErr.Code, "WebsocketModifyOrder should return Crypto.com's refusal")
}

func TestAmendOrderWith(t *testing.T) {
	t.Parallel()
	notFound := func(string) (string, *APIError) { return "", &APIError{Code: 212, Message: "INVALID_ORDERID"} }
	legAmended := testResult(`{"order_id":"5755600460443890101","client_oid":"leg"}`)
	leg := func() *orderAmendment {
		return &orderAmendment{order: &AmendOrderRequest{OrderID: "5755600460443890101", OriginalClientOrderID: "leg", NewPrice: 77000, NewQuantity: 0.01}}
	}
	amendment := testRequest{Method: "private/amend-order", Params: `{"order_id":"5755600460443890101","orig_client_oid":"leg","new_price":"77000","new_quantity":"0.01"}`}
	advancedAmendment := testRequest{Method: "private/advanced/amend-order", Params: `{"order_id":"5755600460443890101","client_oid":"leg","new_quantity":"0.01","new_price":"77000"}`}
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name      string
		amendment *orderAmendment
		routes    map[string]testRoute
		exp       *OrderIDResponse
		requests  []testRequest
	}{
		{
			name:      "a trigger order, which only the advanced order methods amend",
			amendment: &orderAmendment{trigger: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0.0278)), NewReferencePrice: 0.0279}},
			routes:    map[string]testRoute{"private/amend-order": legAmended, "private/advanced/amend-order": testResult(advancedOrderIDResult)},
			exp:       &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"},
			requests:  []testRequest{{Method: "private/advanced/amend-order", Params: `{"order_id":"5755600460443882762","new_quantity":"10","new_price":"0.0278","new_ref_price":"0.0279"}`}},
		},
		{
			name:      "an order the trading methods amend",
			amendment: leg(),
			routes:    map[string]testRoute{"private/amend-order": legAmended, "private/advanced/amend-order": testResult(advancedOrderIDResult)},
			exp:       &OrderIDResponse{OrderID: "5755600460443890101", ClientOrderID: "leg"},
			requests:  []testRequest{amendment},
		},
		{
			name:      "an order list's leg, which the trading methods refuse",
			amendment: leg(),
			routes:    map[string]testRoute{"private/amend-order": notFound, "private/advanced/amend-order": legAmended},
			exp:       &OrderIDResponse{OrderID: "5755600460443890101", ClientOrderID: "leg"},
			requests:  []testRequest{amendment, advancedAmendment},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, received := newRoutesServerExchange(t, tc.routes)
			got, err := amendOrderWith(t.Context(), tc.amendment, ex.AmendOrder, ex.AmendAdvancedOrder)
			require.NoErrorf(t, err, "amendOrderWith must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "amendOrderWith should return the amended order's IDs for %s", tc.name)
			assert.Equalf(t, tc.requests, received(), "amendOrderWith should send the amendments expected for %s", tc.name)
		})
	}

	renamed := leg()
	renamed.order.ClientOrderID = "renamed"
	for _, tc := range []struct {
		name      string
		amendment *orderAmendment
		routes    map[string]testRoute
		apiErr    *APIError
		err       error
		requests  []testRequest
	}{
		{
			name:      "an order whose amendment Crypto.com refuses as a whole",
			amendment: leg(),
			routes: map[string]testRoute{
				"private/amend-order": func(string) (string, *APIError) {
					return "", &APIError{Code: codeTooManyRequests, Message: "TOO_MANY_REQUESTS"}
				},
				"private/advanced/amend-order": legAmended,
			},
			apiErr:   &APIError{Code: codeTooManyRequests, Message: "TOO_MANY_REQUESTS"},
			requests: []testRequest{amendment},
		},
		{
			name:      "an order list's leg given a new client order ID",
			amendment: renamed,
			routes:    map[string]testRoute{"private/amend-order": notFound, "private/advanced/amend-order": legAmended},
			apiErr:    &APIError{Code: 212, Message: "INVALID_ORDERID"},
			err:       errClientOrderIDNotAmendable,
			requests:  []testRequest{{Method: "private/amend-order", Params: `{"order_id":"5755600460443890101","orig_client_oid":"leg","client_oid":"renamed","new_price":"77000","new_quantity":"0.01"}`}},
		},
		{
			name:      "an order list's leg the advanced order methods answer without a result",
			amendment: leg(),
			routes:    map[string]testRoute{"private/amend-order": notFound, "private/advanced/amend-order": testResult("")},
			apiErr:    &APIError{Code: 212, Message: "INVALID_ORDERID"},
			err:       common.ErrNoResponse,
			requests:  []testRequest{amendment, advancedAmendment},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, tc.routes)
			got, err := amendOrderWith(t.Context(), tc.amendment, ex.AmendOrder, ex.AmendAdvancedOrder)
			assert.Nilf(t, got, "amendOrderWith should return no IDs for %s", tc.name)
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "amendOrderWith must return the trading methods' refusal of %s", tc.name)
			assert.Equalf(t, tc.apiErr, apiErr, "amendOrderWith should return Crypto.com's code and message for %s", tc.name)
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "amendOrderWith should return why it did not amend %s", tc.name)
			}
			assert.Equalf(t, tc.requests, received(), "amendOrderWith should send the amendments expected for %s", tc.name)
		})
	}
}

func TestIsTriggerOrder(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, orderType := range []order.Type{order.Stop, order.StopMarket, order.StopLimit, order.TakeProfit, order.TakeProfitMarket, order.TakeProfitLimit} {
		t.Run(orderType.String(), func(t *testing.T) {
			assert.Truef(t, isTriggerOrder(orderType), "isTriggerOrder should report %s a trigger order", orderType)
		})
	}
	for _, orderType := range []order.Type{order.Limit, order.Market, order.UnknownType, order.AnyType, order.TrailingStop} {
		t.Run(orderType.String(), func(t *testing.T) {
			t.Parallel()
			assert.Falsef(t, isTriggerOrder(orderType), "isTriggerOrder should not report %s a trigger order", orderType)
		})
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must refuse a nil cancel")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{}), order.ErrOrderIDNotSet, "CancelOrder must refuse a cancel without an order ID")
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		if !mockTests {
			orderID := placeTradingTestOrder(t)
			assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: orderID, Type: order.Limit}), "CancelOrder should cancel the order the test placed")
			return
		}
		assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: tradingTestOrderID, Type: order.Limit}), "CancelOrder should cancel an order")
		assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: "5755600460443882762", Type: order.StopLimit}), "CancelOrder should cancel a trigger order with the advanced order methods")
	})

	notFound := func(string) (string, *APIError) { return "", &APIError{Code: 40401, Message: "NOT_FOUND"} }

	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"private/cancel-order":          notFound,
		"private/advanced/cancel-order": testResult(`{"order_id":"5755600460443890101"}`),
	})
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "5755600460443890101", Type: order.Limit}), "CancelOrder must cancel an order list's leg")
	expRequests := []testRequest{
		{Method: "private/cancel-order", Params: `{"order_id":"5755600460443890101"}`},
		{Method: "private/advanced/cancel-order", Params: `{"order_id":"5755600460443890101"}`},
	}
	assert.Equal(t, expRequests, received(), "CancelOrder should cancel an order the trading methods refuse with the advanced order methods")

	ex, _ = newRoutesServerExchange(t, map[string]testRoute{"private/cancel-order": notFound, "private/advanced/cancel-order": notFound})
	err := ex.CancelOrder(t.Context(), &order.Cancel{ClientOrderID: "gone"})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "CancelOrder must return Crypto.com's error when neither cancel finds the order")
	assert.Equal(t, int64(40401), apiErr.Code, "CancelOrder should return Crypto.com's code")
	require.ErrorIs(t, newUnavailableServerExchange(t).CancelOrder(t.Context(), &order.Cancel{OrderID: "1"}), request.ErrBadStatus, "CancelOrder must return a failed request's error without falling back")

	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/cancel-order": func(string) (string, *APIError) {
			return "", &APIError{Code: codeTooManyRequests, Message: "TOO_MANY_REQUESTS"}
		},
	})
	err = ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "1"})
	require.ErrorAs(t, err, &apiErr, "CancelOrder must return Crypto.com's refusal of the request")
	assert.Equal(t, int64(codeTooManyRequests), apiErr.Code, "CancelOrder should return the rate limit's code")
	assert.Len(t, received(), 1, "CancelOrder should not retry a request Crypto.com refused as a whole with the advanced order methods")

	// A trigger order, and an order the trading methods refuse, are named to the advanced order methods by the client order
	// ID the cancel names them by
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/cancel-order":          notFound,
		"private/advanced/cancel-order": testResult(`{"order_id":"1"}`),
	})
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{ClientOrderID: "stop", Type: order.Stop}), "CancelOrder must cancel a trigger order named by its client order ID")
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{ClientOrderID: "leg"}), "CancelOrder must cancel an order list's leg named by its client order ID")
	expRequests = []testRequest{
		{Method: "private/advanced/cancel-order", Params: `{"client_oid":"stop"}`},
		{Method: "private/cancel-order", Params: `{"client_oid":"leg"}`},
		{Method: "private/advanced/cancel-order", Params: `{"client_oid":"leg"}`},
	}
	assert.Equal(t, expRequests, received(), "CancelOrder should name an order to the advanced order methods by its client order ID")
}

func TestRefusedForOrder(t *testing.T) {
	t.Parallel()
	assert.False(t, refusedForOrder(nil), "refusedForOrder should not report no error")
	assert.False(t, refusedForOrder(request.ErrBadStatus), "refusedForOrder should not report a failed request")
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, code := range []int64{codeUnauthorised, codeInvalidNonce, codeIPIllegal, codeUserTierInvalid, codeRequestTimeout, codeTooManyRequests, codeInternalError} {
		t.Run(strconv.FormatInt(code, 10), func(t *testing.T) {
			assert.Falsef(t, refusedForOrder(fmt.Errorf("wrapped: %w", &APIError{Code: code})), "refusedForOrder should not report a request refused with code %d", code)
		})
	}
	for _, code := range []int64{40401, 212, 316} {
		t.Run(strconv.FormatInt(code, 10), func(t *testing.T) {
			t.Parallel()
			assert.Truef(t, refusedForOrder(fmt.Errorf("wrapped: %w", &APIError{Code: code})), "refusedForOrder should report an order refused with code %d", code)
		})
	}
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.WebsocketCancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "WebsocketCancelOrder must refuse a nil cancel")
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t)
	if !mockTests {
		orderID := placeTradingTestOrder(t)
		assert.NoError(t, ex.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: orderID, Type: order.Limit}), "WebsocketCancelOrder should cancel the order the test placed")
		return
	}
	require.NoError(t, ex.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: tradingTestOrderID, Type: order.Limit}), "WebsocketCancelOrder must not error")
	assert.Len(t, venue.received("private/cancel-order"), 1, "WebsocketCancelOrder should cancel an order")
	require.NoError(t, ex.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "5755600460443882762", Type: order.TakeProfit}), "WebsocketCancelOrder must not error for a trigger order")
	assert.Len(t, venue.received("private/advanced/cancel-order"), 1, "WebsocketCancelOrder should cancel a trigger order with the advanced order methods")
}

func TestCancelOrderWith(t *testing.T) {
	t.Parallel()
	notFound := &APIError{Code: 40401, Message: "NOT_FOUND"}
	for _, tc := range []struct {
		name        string
		cancel      *order.Cancel
		cancelErr   error
		advancedErr error
		exp         []string
		err         error
		apiErrs     []*APIError
	}{
		{name: "nil cancel", err: order.ErrCancelOrderIsNil},
		{name: "no order ID", cancel: &order.Cancel{Type: order.Limit}, err: order.ErrOrderIDNotSet},
		{name: "trigger order", cancel: &order.Cancel{OrderID: "1", Type: order.StopLimit}, exp: []string{"advanced 1/"}},
		{name: "trigger order refused", cancel: &order.Cancel{ClientOrderID: "c", Type: order.TakeProfit}, advancedErr: notFound, exp: []string{"advanced /c"}, apiErrs: []*APIError{notFound}},
		{name: "regular order", cancel: &order.Cancel{OrderID: "1", Type: order.Limit}, exp: []string{"regular 1/"}},
		{name: "leg the trading methods refuse", cancel: &order.Cancel{OrderID: "1", ClientOrderID: "c"}, cancelErr: notFound, exp: []string{"regular 1/c", "advanced 1/c"}},
		{name: "refused by both", cancel: &order.Cancel{ClientOrderID: "c"}, cancelErr: notFound, advancedErr: &APIError{Code: 316, Message: "NO_ACTIVE_ORDER"}, exp: []string{"regular /c", "advanced /c"}, apiErrs: []*APIError{notFound, {Code: 316, Message: "NO_ACTIVE_ORDER"}}},
		{name: "request refused as a whole", cancel: &order.Cancel{OrderID: "1"}, cancelErr: &APIError{Code: codeTooManyRequests, Message: "TOO_MANY_REQUESTS"}, exp: []string{"regular 1/"}, apiErrs: []*APIError{{Code: codeTooManyRequests, Message: "TOO_MANY_REQUESTS"}}},
		{name: "request failed", cancel: &order.Cancel{OrderID: "1"}, cancelErr: request.ErrBadStatus, exp: []string{"regular 1/"}, err: request.ErrBadStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls []string
			cancel := func(_ context.Context, r *CancelOrderRequest) (*OrderIDResponse, error) {
				calls = append(calls, "regular "+r.OrderID+"/"+r.ClientOrderID)
				return &OrderIDResponse{OrderID: r.OrderID}, tc.cancelErr
			}
			cancelAdvanced := func(_ context.Context, r *AdvancedOrderIDRequest) (*OrderIDResponse, error) {
				calls = append(calls, "advanced "+r.OrderID+"/"+r.ClientOrderID)
				return &OrderIDResponse{OrderID: r.OrderID}, tc.advancedErr
			}
			err := cancelOrderWith(t.Context(), tc.cancel, cancel, cancelAdvanced)
			switch {
			case tc.err != nil:
				require.ErrorIs(t, err, tc.err, "cancelOrderWith must return the expected error")
			case len(tc.apiErrs) > 0:
				for _, exp := range tc.apiErrs {
					assert.ErrorContainsf(t, err, exp.Error(), "cancelOrderWith should return Crypto.com's refusal %s", exp)
				}
			default:
				require.NoError(t, err, "cancelOrderWith must not error")
			}
			assert.Equal(t, tc.exp, calls, "cancelOrderWith should send the cancels expected")
		})
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, errOrderListEmpty, "CancelBatchOrders must refuse no orders")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1"}, {}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must refuse an order without an ID")
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"private/cancel-order-list": func(params string) (string, *APIError) {
			if strings.Contains(params, `"order_id":"11"`) {
				return `[{"code":0,"index":0}]`, nil
			}
			return `[{"code":0,"index":0},{"code":0,"index":2},{"code":316,"index":1,"message":"NO_ACTIVE_ORDER"},{"code":0,"index":3},{"code":0,"index":4},{"code":0,"index":5},{"code":0,"index":6},{"code":0,"index":7},{"code":0,"index":8}]`, nil
		},
		// The order the trading methods could not cancel is an order list's leg, which the advanced order methods cancel
		"private/advanced/cancel-order-list": testResult(`[{"code":0,"index":0},{"code":0,"index":1}]`),
	})
	cancels := make([]order.Cancel, 0, 12)
	for i := range 11 {
		cancels = append(cancels, order.Cancel{OrderID: strconv.Itoa(i + 1)})
	}
	cancels = append(cancels, order.Cancel{ClientOrderID: "my_sl_order", Type: order.Stop})
	got, err := ex.CancelBatchOrders(t.Context(), cancels)
	require.NoError(t, err, "CancelBatchOrders must not error")
	exp := &order.CancelBatchResponse{Status: map[string]string{
		"1": "PENDING_CANCEL", "2": "PENDING_CANCEL", "3": "PENDING_CANCEL", "4": "PENDING_CANCEL", "5": "PENDING_CANCEL", "6": "PENDING_CANCEL",
		"7": "PENDING_CANCEL", "8": "PENDING_CANCEL", "9": "PENDING_CANCEL", "10": "no result for the order", "11": "PENDING_CANCEL", "my_sl_order": "PENDING_CANCEL",
	}}
	assert.Equal(t, exp, got, "CancelBatchOrders should return every order's outcome")
	requests := received()
	require.Len(t, requests, 3, "CancelBatchOrders must send two lists of up to 10 and a list of trigger orders")
	assert.JSONEq(t, `{"contingency_type":"LIST","order_list":[{"order_id":"11"}]}`, requests[1].Params, "CancelBatchOrders should send the orders past 10 in a second list")
	assert.Equal(t, testRequest{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[{"client_oid":"my_sl_order"},{"order_id":"2"}]}`}, requests[2], "CancelBatchOrders should cancel trigger orders, and the orders the trading methods refused, with the advanced order methods")

	ex, _ = newRoutesServerExchange(t, map[string]testRoute{
		"private/cancel-order-list":          testResult(`[{"code":316,"index":0,"message":"NO_ACTIVE_ORDER"},{"code":316,"index":1,"message":"NO_ACTIVE_ORDER"}]`),
		"private/advanced/cancel-order-list": testResult(`[{"code":40401,"index":0,"message":"NOT_FOUND"}]`),
	})
	got, err = ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1"}, {OrderID: "2"}})
	require.NoError(t, err, "CancelBatchOrders must not error for orders neither method cancels")
	exp = &order.CancelBatchResponse{Status: map[string]string{"1": "316 NO_ACTIVE_ORDER; 40401 NOT_FOUND", "2": "316 NO_ACTIVE_ORDER; no result for the order"}}
	assert.Equal(t, exp, got, "CancelBatchOrders should report both refusals of an order neither method cancels")

	// An entry refused for the request rather than its order, which the advanced order methods would refuse too, is not
	// retried with them
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/cancel-order-list": testResult(`[{"code":42901,"index":0,"message":"TOO_MANY_REQUESTS"},{"code":0,"index":1}]`),
	})
	got, err = ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1"}, {OrderID: "2"}})
	require.NoError(t, err, "CancelBatchOrders must not error for an entry refused for the request")
	exp = &order.CancelBatchResponse{Status: map[string]string{"1": "42901 TOO_MANY_REQUESTS", "2": "PENDING_CANCEL"}}
	assert.Equal(t, exp, got, "CancelBatchOrders should report an entry refused for the request")
	assert.Len(t, received(), 1, "CancelBatchOrders should not retry an entry refused for the request with the advanced order methods")

	// A list whose request fails stops the lists after it, and the outcomes of those sent before it come with the error
	ex, _ = newRoutesServerExchange(t, map[string]testRoute{
		"private/cancel-order-list": func(params string) (string, *APIError) {
			if strings.Contains(params, `"order_id":"11"`) {
				return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}
			}
			return `[{"code":0,"index":0},{"code":0,"index":1},{"code":0,"index":2},{"code":0,"index":3},{"code":0,"index":4},{"code":0,"index":5},{"code":0,"index":6},{"code":0,"index":7},{"code":0,"index":8},{"code":0,"index":9}]`, nil
		},
	})
	got, err = ex.CancelBatchOrders(t.Context(), cancels[:len(cancels)-1])
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "CancelBatchOrders must return the error of a list whose request failed")
	require.NotNil(t, got, "CancelBatchOrders must return the outcomes of the lists sent before one failed")
	assert.Len(t, got.Status, 10, "CancelBatchOrders should return the outcome of each order of the lists sent before one failed")
	assert.Equal(t, "PENDING_CANCEL", got.Status["10"], "CancelBatchOrders should report the orders cancelled before a list failed")
	assert.NotContains(t, got.Status, "11", "CancelBatchOrders should report no outcome for an order whose list failed")

	// The recorded mock holds a cancel of trigger orders only, so live the batch cancels two orders the test places
	skipLiveUnlessCanManipulate(t)
	cancels = []order.Cancel{{OrderID: "289991951579667814", Type: order.Stop}, {ClientOrderID: "my_tp_order", Type: order.TakeProfit}}
	exp = &order.CancelBatchResponse{Status: map[string]string{"289991951579667814": "PENDING_CANCEL", "my_tp_order": "PENDING_CANCEL"}}
	if !mockTests {
		first := placeTradingTestOrder(t)
		cancelTradingOrderIfTestFails(t, first)
		second := placeTradingTestOrder(t)
		cancelTradingOrderIfTestFails(t, second)
		cancels = []order.Cancel{{OrderID: first}, {OrderID: second}}
		exp = &order.CancelBatchResponse{Status: map[string]string{first: "PENDING_CANCEL", second: "PENDING_CANCEL"}}
	}
	got, err = e.CancelBatchOrders(t.Context(), cancels)
	require.NoError(t, err, "CancelBatchOrders must not error")
	assert.Equal(t, exp, got, "CancelBatchOrders should report every order pending cancellation")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must refuse a nil cancel")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must refuse an unsupported asset")
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-open-orders": testResult(`{"data":[
			{"order_id":"1","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD","exec_inst":[]},
			{"order_id":"2","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD","exec_inst":["MARGIN_ORDER"]},
			{"order_id":"3","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTCUSD-PERP"}]}`),
		"private/advanced/get-open-orders": testResult(`{"data":[
			{"order_id":"4","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD","list_id":"9"},
			{"order_id":"5","order_type":"STOP_LOSS","side":"SELL","status":"ACTIVE","instrument_name":"BTC_USD","exec_inst":["MARGIN_ORDER"]}]}`),
		"private/cancel-order-list":          testResult(`[{"code":0,"index":0}]`),
		"private/advanced/cancel-order-list": testResult(`[{"code":0,"index":0}]`),
	})
	got, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error")
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"1": "PENDING_CANCEL", "4": "PENDING_CANCEL"}}, got, "CancelAllOrders should cancel the asset's orders only")
	expRequests := []testRequest{
		{Method: "private/get-open-orders", Params: `{}`},
		{Method: "private/advanced/get-open-orders", Params: `{}`},
		{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[{"order_id":"1"}]}`},
		{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[{"order_id":"4"}]}`},
	}
	assert.Equal(t, expRequests, received(), "CancelAllOrders should list the open orders and cancel the asset's by their IDs, an order list's leg with the advanced order methods")

	got, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures, Pair: testFuturesPair})
	require.NoError(t, err, "CancelAllOrders must not error without orders to cancel")
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{}}, got, "CancelAllOrders should cancel nothing when the asset has no open orders")
	assert.Equal(t, testRequest{Method: "private/advanced/get-open-orders", Params: `{"instrument_name":"BTCUSD-261030"}`}, received()[5], "CancelAllOrders should list the pair's open orders")

	// A list whose request fails stops the lists after it, and the outcomes of those sent before it come with the error
	ex, _ = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-open-orders":            testResult(`{"data":[{"order_id":"1","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD","exec_inst":[]}]}`),
		"private/advanced/get-open-orders":   testResult(`{"data":[{"order_id":"4","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD","list_id":"9"}]}`),
		"private/cancel-order-list":          testResult(`[{"code":0,"index":0}]`),
		"private/advanced/cancel-order-list": func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} },
	})
	got, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "CancelAllOrders must return the error of a list whose request failed")
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"1": "PENDING_CANCEL"}}, got, "CancelAllOrders should return the outcomes of the lists sent before one failed")

	// An instrument the other tests place no orders on, so that cancelling its orders live leaves theirs
	skipLiveUnlessCanManipulate(t)
	ex = e
	if mockTests {
		ex, received = newRoutesServerExchange(t, map[string]testRoute{
			"private/get-open-orders":          testResult(`{"data":[{"order_id":"6","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"ETHUSD-PERP","exec_inst":["POST_ONLY"]}]}`),
			"private/advanced/get-open-orders": testResult(`{"data":[]}`),
			"private/cancel-order-list":        testResult(`[{"code":0,"index":0}]`),
		})
	}
	got, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualSwap, Pair: perpETHUSD})
	require.NoError(t, err, "CancelAllOrders must not error for a perpetual's orders")
	if !mockTests {
		for orderID, status := range got.Status {
			assert.Equalf(t, "PENDING_CANCEL", status, "CancelAllOrders should cancel order %s", orderID)
		}
		return
	}
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"6": "PENDING_CANCEL"}}, got, "CancelAllOrders should cancel the perpetual's open orders")
	expRequests = []testRequest{
		{Method: "private/get-open-orders", Params: `{"instrument_name":"ETHUSD-PERP"}`},
		{Method: "private/advanced/get-open-orders", Params: `{"instrument_name":"ETHUSD-PERP"}`},
		{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[{"order_id":"6"}]}`},
	}
	assert.Equal(t, expRequests, received(), "CancelAllOrders should list the perpetual's open orders and cancel them by their IDs")
}

func TestListCancelStatus(t *testing.T) {
	t.Parallel()
	t.Run("an acknowledgement leaves cancellation pending", func(t *testing.T) {
		t.Parallel()
		status := make(map[string]string)
		refused := listCancelStatus([]orderCancel{{OrderID: "1"}}, []OrderListEntryResponse{{Index: 0, Code: 0}}, status)
		assert.Empty(t, refused, "an acknowledged cancellation should not be refused")
		assert.Equal(t, order.PendingCancel.String(), status["1"], "the order should be pending cancellation")
	})
	status := make(map[string]string)
	refused := listCancelStatus([]orderCancel{{OrderID: "1"}, {ClientOrderID: "c"}, {OrderID: "3"}}, []OrderListEntryResponse{{Index: 1}, {Index: -1}, {Index: 3}, {Index: 2, Code: 316, Message: "NO_ACTIVE_ORDER"}}, status)
	assert.Equal(t, map[string]string{"1": "no result for the order", "c": "PENDING_CANCEL", "3": "316 NO_ACTIVE_ORDER"}, status, "listCancelStatus should key an order by its client order ID without an order ID, and leave out entries out of range")
	assert.Equal(t, []int{2}, refused, "listCancelStatus should return the indexes of the orders Crypto.com refused to cancel")

	// Entries refused for the request rather than their orders are reported, and left out of the orders refused
	for _, code := range []int64{codeUnauthorised, codeInvalidNonce, codeIPIllegal, codeUserTierInvalid, codeRequestTimeout, codeTooManyRequests, codeInternalError} {
		t.Run(strconv.FormatInt(code, 10), func(t *testing.T) {
			t.Parallel()
			status := make(map[string]string)
			refused := listCancelStatus([]orderCancel{{OrderID: "1"}, {OrderID: "2"}}, []OrderListEntryResponse{{Index: 0, Code: types.Number(code), Message: "REFUSED"}, {Index: 1, Code: 316, Message: "NO_ACTIVE_ORDER"}}, status)
			assert.Equalf(t, map[string]string{"1": strconv.FormatInt(code, 10) + " REFUSED", "2": "316 NO_ACTIVE_ORDER"}, status, "listCancelStatus should report an entry refused with code %d", code)
			assert.Equalf(t, []int{1}, refused, "listCancelStatus should leave out an entry refused with code %d for the request", code)
		})
	}
}

func TestGetOrderInfo(t *testing.T) { //nolint:tparallel // the order is looked up on e in order, before the later calls of the same method
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must refuse an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must refuse an unsupported asset")
	ctx := t.Context()
	t.Run("an order of the trading methods", func(t *testing.T) {
		skipLiveUnlessCanManipulate(t)
		orderID := "6530219466236720403"
		if !mockTests {
			orderID = placeTradingTestOrder(t)
			cancelTradingOrderAfterTest(t, orderID)
		}
		got, err := e.GetOrderInfo(ctx, orderID, testPerpetualPair, asset.PerpetualSwap)
		require.NoError(t, err, "GetOrderInfo must not error")
		if mockTests {
			exp := &order.Detail{
				TimeInForce:        order.GoodTillCancel,
				Price:              50000,
				Amount:             0.01,
				RemainingAmount:    0.01,
				CostAsset:          currency.USD,
				FeeAsset:           currency.USD,
				Exchange:           e.Name,
				OrderID:            "6530219466236720403",
				ClientOrderID:      "1613571154900",
				AccountID:          "52e7c00f-1324-5a6a-bf6e-de445bde21a5",
				Type:               order.Limit,
				Side:               order.Buy,
				Status:             order.Cancelled,
				AssetType:          asset.PerpetualSwap,
				Date:               time.Unix(0, 1613575617173123456),
				CloseTime:          time.UnixMilli(1613575617209),
				LastUpdated:        time.UnixMilli(1613575617209),
				Pair:               testPerpetualPair,
				MarginType:         margin.Multi,
				SettlementCurrency: currency.USD,
			}
			assert.Equal(t, exp, got, "GetOrderInfo should return every field of the order")
		} else {
			placed := tradingTestOrder()
			exp := &order.Detail{Exchange: e.Name, OrderID: orderID, Pair: testPerpetualPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Limit, Price: placed.Price.Float64(), Amount: placed.Quantity.Float64()}
			terms := &order.Detail{Exchange: got.Exchange, OrderID: got.OrderID, Pair: got.Pair, AssetType: got.AssetType, Side: got.Side, Type: got.Type, Price: got.Price, Amount: got.Amount}
			assert.Equal(t, exp, terms, "GetOrderInfo should return the order the test placed")
		}
		_, err = e.GetOrderInfo(ctx, orderID, testSpotPair, asset.Spot)
		assert.ErrorIs(t, err, errOrderAssetMismatch, "GetOrderInfo should refuse an order of another asset")
	})

	ex, _ := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail":          func(string) (string, *APIError) { return "", &APIError{Code: 40401, Message: "NOT_FOUND"} },
		"private/advanced/get-order-detail": testResult(`{"order_id":"5755600460443890102","order_type":"STOP_LOSS","side":"SELL","quantity":"0.0100","ref_price":"74000.00","status":"ACTIVE","instrument_name":"BTC_USD","exec_inst":["MARGIN_ORDER"]}`),
	})
	got, err := ex.GetOrderInfo(t.Context(), "5755600460443890102", currency.EMPTYPAIR, asset.Margin)
	require.NoError(t, err, "GetOrderInfo must not error for a trigger order")
	exp := &order.Detail{Amount: 0.01, TriggerPrice: 74000, RemainingAmount: 0.01, CostAsset: currency.USD, Exchange: ex.Name, OrderID: "5755600460443890102", Type: order.Stop, Side: order.Sell, Status: order.Active, AssetType: asset.Margin, Pair: testSpotPair, MarginType: margin.Multi}
	assert.Equal(t, exp, got, "GetOrderInfo should look an order the trading methods do not know up among the advanced orders")

	_, err = newUnavailableServerExchange(t).GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderInfo should return a failed request's error without falling back")
}

// testOpenOrdersRoutes answer the open order requests of the order tests: two BTCUSD-PERP orders, one on an isolated
// position, a margin order, and an OTO list's limit leg and stop loss
var testOpenOrdersRoutes = map[string]testRoute{
	"private/get-open-orders": testResult(`{"data":[
		{"order_id":"6530219466236720401","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02","order_type":"LIMIT","time_in_force":"GOOD_TILL_CANCEL","side":"BUY","exec_inst":["POST_ONLY"],"quantity":"0.0001","limit_price":"50000.5","order_value":"5.000050","avg_price":"0.0","cumulative_quantity":"0.0000","status":"ACTIVE","instrument_name":"BTCUSD-PERP","fee_instrument_name":"USD","create_time":"1792890506012","create_time_ns":"1792890506012345678","update_time":"1792890506012"},
		{"order_id":"6530219466236720402","order_type":"LIMIT","side":"BUY","exec_inst":["ISOLATED_MARGIN"],"quantity":"0.0100","limit_price":"50000.0","status":"ACTIVE","instrument_name":"BTCUSD-PERP","isolation_id":"6530219466236720402","leverage":10,"create_time":"1792890500001","update_time":"1792890500001"},
		{"order_id":"6530219466236720405","order_type":"LIMIT","side":"SELL","exec_inst":["MARGIN_ORDER"],"quantity":"0.0100","limit_price":"95000.0","status":"ACTIVE","instrument_name":"BTC_USD","create_time":"1792890500002","update_time":"1792890500002"}]}`),
	"private/advanced/get-open-orders": testResult(`{"data":[
		{"order_id":"5755600460443890101","order_type":"LIMIT","side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD","list_id":"6498090546073120187","contingency_type":"SPOT_ATTACH","create_time":"1792836351902","update_time":"1792836351902"},
		{"order_id":"5755600460443890102","order_type":"STOP_LOSS","side":"SELL","quantity":"0.0100","ref_price":"74000.00","status":"ACTIVE","instrument_name":"BTC_USD","list_id":"6498090546073120187","contingency_type":"SPOT_ATTACH","create_time":"1792836351902","update_time":"1792836351902"}]}`),
}

func TestGetActiveOrders(t *testing.T) { //nolint:tparallel // the assets share one exchange in order, before its requests are checked
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must refuse a nil request")
	skipLiveWithoutCredentials(t)
	ex := e
	var received func() []testRequest
	if mockTests {
		ex, received = newRoutesServerExchange(t, testOpenOrdersRoutes)
	}
	// The assets share ex in order, before the calls below, whose last requests are checked
	for _, a := range ex.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
		})
	}
	if !mockTests {
		return
	}
	got, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for spot")
	exp := order.FilteredOrders{
		{Price: 78000, Amount: 0.01, RemainingAmount: 0.01, CostAsset: currency.USD, Exchange: ex.Name, OrderID: "5755600460443890101", Type: order.Limit, Side: order.Buy, Status: order.Active, AssetType: asset.Spot, Date: time.UnixMilli(1792836351902), LastUpdated: time.UnixMilli(1792836351902), Pair: testSpotPair},
		{TriggerPrice: 74000, Amount: 0.01, RemainingAmount: 0.01, CostAsset: currency.USD, Exchange: ex.Name, OrderID: "5755600460443890102", Type: order.Stop, Side: order.Sell, Status: order.Active, AssetType: asset.Spot, Date: time.UnixMilli(1792836351902), LastUpdated: time.UnixMilli(1792836351902), Pair: testSpotPair},
	}
	assert.Equal(t, exp, got, "GetActiveOrders should return the spot orders, those the advanced order methods hold included")
	got, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.Sell, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for a side")
	assert.Equal(t, exp[1:], got, "GetActiveOrders should return only the orders of the side requested")
	got, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.Stop})
	require.NoError(t, err, "GetActiveOrders must not error for a type")
	assert.Equal(t, exp[1:], got, "GetActiveOrders should return only the orders of the type requested")
	got, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{spotETHUSD, currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter)}, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for pairs")
	assert.Empty(t, got, "GetActiveOrders should return only the orders of the pairs requested")

	got, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Pairs: currency.Pairs{testPerpetualPair}, Side: order.Buy, Type: order.Limit})
	require.NoError(t, err, "GetActiveOrders must not error for a pair")
	exp = order.FilteredOrders{
		{TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 50000.5, Amount: 0.0001, RemainingAmount: 0.0001, CostAsset: currency.USD, FeeAsset: currency.USD, Exchange: ex.Name, OrderID: "6530219466236720401", ClientOrderID: tradingTestOrderClientOrderID, Type: order.Limit, Side: order.Buy, Status: order.Active, AssetType: asset.PerpetualSwap, Date: time.Unix(0, 1792890506012345678), LastUpdated: time.UnixMilli(1792890506012), Pair: testPerpetualPair, MarginType: margin.Multi, SettlementCurrency: currency.USD},
		{Leverage: 10, Price: 50000, Amount: 0.01, RemainingAmount: 0.01, CostAsset: currency.USD, Exchange: ex.Name, OrderID: "6530219466236720402", Type: order.Limit, Side: order.Buy, Status: order.Active, AssetType: asset.PerpetualSwap, Date: time.UnixMilli(1792890500001), LastUpdated: time.UnixMilli(1792890500001), Pair: testPerpetualPair, MarginType: margin.Isolated, SettlementCurrency: currency.USD},
	}
	assert.Equal(t, exp, got, "GetActiveOrders should return the pair's orders")
	requests := received()
	assert.Equal(t, []testRequest{
		{Method: "private/get-open-orders", Params: `{"instrument_name":"BTCUSD-PERP"}`},
		{Method: "private/advanced/get-open-orders", Params: `{"instrument_name":"BTCUSD-PERP"}`},
	}, requests[len(requests)-2:], "GetActiveOrders should request the one pair's orders")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must refuse a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, StartTime: testWindowStart, EndTime: testWindowStart.Add(-time.Hour)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOrderHistory must refuse a start after the end")
	ns := func(t time.Time) string { return `"` + strconv.FormatInt(t.UnixNano(), 10) + `"` }
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex := e
		var received func() []testRequest
		if mockTests {
			ex, received = newRoutesServerExchange(t, map[string]testRoute{
				"private/get-order-history": testResult(`{"data":[
					{"order_id":"18342500","client_oid":"1613571154800","order_type":"LIMIT","side":"BUY","quantity":"0.0500","limit_price":"51283.0","avg_price":"51278.5","cumulative_quantity":"0.0500","cumulative_value":"2563.925","cumulative_fee":"1.0256","status":"FILLED","instrument_name":"BTC_USD","exec_inst":["MARGIN_ORDER"],"fee_instrument_name":"USD","create_time":"1792634400000","update_time":"1792634400100"},
					{"order_id":"18342311","order_type":"MARKET","side":"BUY","order_value":"25","quantity":"0","cumulative_quantity":"0.0003","cumulative_value":"24.99","status":"FILLED","instrument_name":"BTC_USD","create_time":"1792630800000","update_time":"1792630800001"}]}`),
				"private/advanced/get-order-history": testResult(`{"data":[]}`),
			})
		}
		start := testWindowStart.Add(-2 * privateHistoryWindow)
		end := start.Add(36 * time.Hour)
		got, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, StartTime: start, EndTime: end})
		require.NoError(t, err, "GetOrderHistory must not error")
		if !mockTests {
			return
		}
		exp := order.FilteredOrders{{
			Amount:          0,
			QuoteAmount:     25,
			ExecutedAmount:  0.0003,
			Cost:            24.99,
			CostAsset:       currency.USD,
			Exchange:        ex.Name,
			OrderID:         "18342311",
			Type:            order.Market,
			Side:            order.Buy,
			Status:          order.Filled,
			AssetType:       asset.Spot,
			Date:            time.UnixMilli(1792630800000),
			CloseTime:       time.UnixMilli(1792630800001),
			LastUpdated:     time.UnixMilli(1792630800001),
			Pair:            testSpotPair,
			RemainingAmount: 0,
		}}
		assert.Equal(t, exp, got, "GetOrderHistory should return the asset's past orders, a market buy by notional with its notional as the quote amount")
		day := end.Add(-privateHistoryWindow)
		expRequests := []testRequest{
			{Method: "private/get-order-history", Params: `{"start_time":` + ns(day) + `,"end_time":` + ns(end) + `,"limit":100}`},
			{Method: "private/get-order-history", Params: `{"start_time":` + ns(start) + `,"end_time":` + ns(day) + `,"limit":100}`},
			{Method: "private/advanced/get-order-history", Params: `{"start_time":` + ns(day) + `,"end_time":` + ns(end) + `,"limit":"100"}`},
			{Method: "private/advanced/get-order-history", Params: `{"start_time":` + ns(start) + `,"end_time":` + ns(day) + `,"limit":"100"}`},
		}
		requests := received()
		require.Len(t, requests, len(expRequests), "GetOrderHistory must request each history a day at a time")
		for i := range expRequests {
			assert.Equalf(t, expRequests[i].Method, requests[i].Method, "GetOrderHistory request %d should be for its method", i)
			assert.JSONEqf(t, expRequests[i].Params, requests[i].Params, "GetOrderHistory request %d should request its day", i)
		}
	})

	// Without a start time each history is requested for the day before the end, the advanced order methods' trigger
	// orders with the trading methods' orders, which the request's filters choose from
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-history":          testResult(`{"data":[{"order_id":"1","order_type":"LIMIT","side":"BUY","quantity":"1","limit_price":"80000","cumulative_quantity":"1","status":"FILLED","instrument_name":"BTC_USD","create_time":"1792634400000","update_time":"1792634400100"}]}`),
		"private/advanced/get-order-history": testResult(`{"data":[{"order_id":"2","order_type":"STOP_LOSS","side":"SELL","quantity":"1","ref_price":"79000","status":"CANCELED","instrument_name":"BTC_USD","create_time":"1792634400200","update_time":"1792634400300"}]}`),
	})
	end := time.UnixMilli(1792634400000).Add(time.Hour)
	got, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, EndTime: end})
	require.NoError(t, err, "GetOrderHistory must not error without a start time")
	orderIDs := make([]string, len(got))
	for i := range got {
		orderIDs[i] = got[i].OrderID
	}
	assert.Equal(t, []string{"1", "2"}, orderIDs, "GetOrderHistory should return the trading methods' orders and the advanced order methods' trigger orders")
	day := end.Add(-privateHistoryWindow)
	expRequests := []testRequest{
		{Method: "private/get-order-history", Params: `{"start_time":` + ns(day) + `,"end_time":` + ns(end) + `,"limit":100}`},
		{Method: "private/advanced/get-order-history", Params: `{"start_time":` + ns(day) + `,"end_time":` + ns(end) + `,"limit":"100"}`},
	}
	requests := received()
	require.Len(t, requests, len(expRequests), "GetOrderHistory must request each history's day before the end without a start time")
	for i := range expRequests {
		assert.Equalf(t, expRequests[i].Method, requests[i].Method, "GetOrderHistory request %d should be for its method without a start time", i)
		assert.JSONEqf(t, expRequests[i].Params, requests[i].Params, "GetOrderHistory request %d should request the day before the end without a start time", i)
	}
	got, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.Sell, Type: order.AnyType, EndTime: end})
	require.NoError(t, err, "GetOrderHistory must not error for a side")
	require.Len(t, got, 1, "GetOrderHistory must return only the orders of the side requested")
	assert.Equal(t, "2", got[0].OrderID, "GetOrderHistory should return the sell of the side requested")
	got, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.Stop, EndTime: end})
	require.NoError(t, err, "GetOrderHistory must not error for a type")
	require.Len(t, got, 1, "GetOrderHistory must return only the orders of the type requested")
	assert.Equal(t, "2", got[0].OrderID, "GetOrderHistory should return the stop loss of the type requested")
	got, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{spotETHUSD, currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter)}, Side: order.AnySide, Type: order.AnyType, EndTime: end})
	require.NoError(t, err, "GetOrderHistory must not error for pairs")
	assert.Empty(t, got, "GetOrderHistory should return only the orders of the pairs requested")
}

func TestOrderRecord(t *testing.T) {
	t.Parallel()
	id, at := orderRecord(&OrderResponse{OrderID: "1", CreateTime: types.Time(time.UnixMilli(1792803600000)), CreateTimeNano: types.Time(time.Unix(0, 1792803600000123456))})
	assert.Equal(t, testPagedRecord{"1", time.Unix(0, 1792803600000123456)}, testPagedRecord{id, at}, "orderRecord should time an order by its nanosecond creation time")
	id, at = orderRecord(&OrderResponse{OrderID: "2", CreateTime: types.Time(time.UnixMilli(1792803600000))})
	assert.Equal(t, testPagedRecord{"2", time.UnixMilli(1792803600000)}, testPagedRecord{id, at}, "orderRecord should time an order by its millisecond creation time without a nanosecond one")
}

func TestOrderSideString(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for side, exp := range map[order.Side]string{
		order.Buy: SideBuy, order.Bid: SideBuy, order.Long: SideBuy,
		order.Sell: SideSell, order.Ask: SideSell, order.Short: SideSell,
	} {
		t.Run(side.String(), func(t *testing.T) {
			got, err := orderSideString(side)
			require.NoErrorf(t, err, "orderSideString must not error for %s", side)
			assert.Equalf(t, exp, got, "orderSideString should map %s", side)
		})
	}
	for _, side := range []order.Side{order.UnknownSide, order.AnySide, order.ClosePosition} {
		t.Run(side.String(), func(t *testing.T) {
			t.Parallel()
			_, err := orderSideString(side)
			assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "orderSideString should refuse %s", side)
		})
	}
}

func TestOrderSideFromString(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for s, exp := range map[string]order.Side{"BUY": order.Buy, "buy": order.Buy, "SELL": order.Sell, "sell": order.Sell} {
		t.Run(s, func(t *testing.T) {
			got, err := orderSideFromString(s)
			require.NoErrorf(t, err, "orderSideFromString must not error for %s", s)
			assert.Equalf(t, exp, got, "orderSideFromString should map %s", s)
		})
	}
	_, err := orderSideFromString("HOLD")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "orderSideFromString should refuse an unknown side")
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		orderType order.Type
		exp       string
		trigger   bool
	}{
		{orderType: order.Limit, exp: OrderTypeLimit},
		{orderType: order.Market, exp: OrderTypeMarket},
		{orderType: order.Stop, exp: OrderTypeStopLoss, trigger: true},
		{orderType: order.StopMarket, exp: OrderTypeStopLoss, trigger: true},
		{orderType: order.StopLimit, exp: OrderTypeStopLimit, trigger: true},
		{orderType: order.TakeProfit, exp: OrderTypeTakeProfit, trigger: true},
		{orderType: order.TakeProfitMarket, exp: OrderTypeTakeProfit, trigger: true},
		{orderType: order.TakeProfitLimit, exp: OrderTypeTakeProfitLimit, trigger: true},
	} {
		t.Run(tc.orderType.String(), func(t *testing.T) {
			got, trigger, err := orderTypeString(tc.orderType)
			require.NoErrorf(t, err, "orderTypeString must not error for %s", tc.orderType)
			assert.Equalf(t, tc.exp, got, "orderTypeString should map %s", tc.orderType)
			assert.Equalf(t, tc.trigger, trigger, "orderTypeString should tell whether %s is a trigger order", tc.orderType)
		})
	}
	for _, orderType := range []order.Type{order.UnknownType, order.TrailingStop, order.OCO, order.AnyType} {
		t.Run(orderType.String(), func(t *testing.T) {
			t.Parallel()
			_, _, err := orderTypeString(orderType)
			assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "orderTypeString should refuse %s", orderType)
		})
	}
}

func TestOrderTypeFromString(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for s, exp := range map[string]order.Type{
		OrderTypeLimit:           order.Limit,
		OrderTypeMarket:          order.Market,
		OrderTypeStopLoss:        order.Stop,
		OrderTypeStopLimit:       order.StopLimit,
		OrderTypeTakeProfit:      order.TakeProfit,
		OrderTypeTakeProfitLimit: order.TakeProfitLimit,
	} {
		t.Run(s, func(t *testing.T) {
			got, err := orderTypeFromString(s)
			require.NoErrorf(t, err, "orderTypeFromString must not error for %s", s)
			assert.Equalf(t, exp, got, "orderTypeFromString should map %s", s)
			back, _, err := orderTypeString(got)
			require.NoErrorf(t, err, "orderTypeString must not error for %s", got)
			assert.Equalf(t, s, back, "orderTypeString should map %s back to %s", got, s)
		})
	}
	_, err := orderTypeFromString("TRAILING_STOP")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "orderTypeFromString should refuse an unknown type")
}

func TestTimeInForceParams(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		tif         order.TimeInForce
		timeInForce string
		postOnly    bool
	}{
		{tif: order.UnknownTIF},
		{tif: order.GoodTillCancel, timeInForce: TimeInForceGoodTillCancel},
		{tif: order.ImmediateOrCancel, timeInForce: TimeInForceImmediateOrCancel},
		{tif: order.FillOrKill, timeInForce: TimeInForceFillOrKill},
		{tif: order.PostOnly, timeInForce: TimeInForceGoodTillCancel, postOnly: true},
		{tif: order.GoodTillCancel | order.PostOnly, timeInForce: TimeInForceGoodTillCancel, postOnly: true},
	} {
		t.Run(subtestName(tc.tif.String()), func(t *testing.T) {
			timeInForce, postOnly, err := timeInForceParams(tc.tif)
			require.NoErrorf(t, err, "timeInForceParams must not error for %s", tc.tif)
			assert.Equalf(t, tc.timeInForce, timeInForce, "timeInForceParams should map %s", tc.tif)
			assert.Equalf(t, tc.postOnly, postOnly, "timeInForceParams should tell whether %s is post only", tc.tif)
		})
	}
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tif := range []order.TimeInForce{order.GoodTillDay, order.GoodTillTime, order.GoodTillCrossing, order.GoodTillDay | order.PostOnly} {
		t.Run(tif.String(), func(t *testing.T) {
			_, _, err := timeInForceParams(tif)
			assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "timeInForceParams should refuse %s", tif)
		})
	}
	for _, tif := range []order.TimeInForce{order.TimeInForce(1 << 15), order.ImmediateOrCancel | order.PostOnly, order.FillOrKill | order.PostOnly, order.GoodTillCancel | order.FillOrKill} {
		t.Run(tif.String(), func(t *testing.T) {
			t.Parallel()
			_, _, err := timeInForceParams(tif)
			assert.ErrorIsf(t, err, order.ErrInvalidTimeInForce, "timeInForceParams should refuse %s", tif)
		})
	}
}

func TestTimeInForceFromResponse(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name        string
		timeInForce string
		execInst    []string
		exp         order.TimeInForce
	}{
		{name: "empty"},
		{name: "good till cancel", timeInForce: TimeInForceGoodTillCancel, exp: order.GoodTillCancel},
		{name: "immediate or cancel", timeInForce: TimeInForceImmediateOrCancel, exp: order.ImmediateOrCancel},
		{name: "fill or kill", timeInForce: TimeInForceFillOrKill, exp: order.FillOrKill},
		{name: "post only", timeInForce: TimeInForceGoodTillCancel, execInst: []string{ExecutionInstructionPostOnly}, exp: order.GoodTillCancel | order.PostOnly},
		{name: "smart post only", timeInForce: TimeInForceGoodTillCancel, execInst: []string{ExecutionInstructionReduceOnly, ExecutionInstructionSmartPostOnly}, exp: order.GoodTillCancel | order.PostOnly},
		{name: "reduce only", timeInForce: TimeInForceGoodTillCancel, execInst: []string{ExecutionInstructionReduceOnly}, exp: order.GoodTillCancel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := timeInForceFromResponse(tc.timeInForce, tc.execInst)
			require.NoErrorf(t, err, "timeInForceFromResponse must not error for %s %v", tc.timeInForce, tc.execInst)
			assert.Equalf(t, tc.exp, got, "timeInForceFromResponse should map %s %v", tc.timeInForce, tc.execInst)
		})
	}
	_, err := timeInForceFromResponse("GOOD_TILL_DATE", nil)
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "timeInForceFromResponse should refuse an unknown time in force")
}

func TestEffectiveTimeInForce(t *testing.T) {
	t.Parallel()
	for tif, exp := range map[order.TimeInForce]order.TimeInForce{
		order.UnknownTIF:                      order.GoodTillCancel,
		order.PostOnly:                        order.GoodTillCancel | order.PostOnly,
		order.GoodTillCancel:                  order.GoodTillCancel,
		order.GoodTillCancel | order.PostOnly: order.GoodTillCancel | order.PostOnly,
		order.ImmediateOrCancel:               order.ImmediateOrCancel,
		order.FillOrKill:                      order.FillOrKill,
	} {
		t.Run(subtestName(tif.String()), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, effectiveTimeInForce(tif), "effectiveTimeInForce should give the time in force Crypto.com places an order with %s", tif)
		})
	}
}

func TestOrderStatus(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name     string
		status   string
		executed types.PreciseNumber
		exp      order.Status
	}{
		{name: "pending", status: OrderStatusPending, exp: order.Pending},
		{name: "new", status: OrderStatusNew, exp: order.New},
		{name: "active", status: OrderStatusActive, exp: order.Active},
		{name: "partially filled", status: OrderStatusActive, executed: preciseNumber("0.5"), exp: order.PartiallyFilled},
		{name: "filled", status: OrderStatusFilled, executed: preciseNumber("1"), exp: order.Filled},
		{name: "cancelled", status: OrderStatusCanceled, exp: order.Cancelled},
		{name: "partially cancelled", status: OrderStatusCanceled, executed: preciseNumber("0.5"), exp: order.PartiallyCancelled},
		{name: "rejected", status: OrderStatusRejected, exp: order.Rejected},
		{name: "expired", status: OrderStatusExpired, exp: order.Expired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orderStatus(&OrderResponse{Status: tc.status, CumulativeQuantity: tc.executed})
			require.NoErrorf(t, err, "orderStatus must not error for %s", tc.status)
			assert.Equalf(t, tc.exp, got, "orderStatus should map %s with %v executed", tc.status, tc.executed)
		})
	}
	_, err := orderStatus(&OrderResponse{Status: "SUSPENDED"})
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "orderStatus should refuse an unknown status")
}

func TestOrderDetail(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	o := &OrderResponse{
		AccountID:             "52e7c00f-1324-5a6z-bfgt-de445bde21a5",
		OrderID:               "19848525",
		ClientOrderID:         "1613571154900",
		OrderType:             OrderTypeStopLimit,
		TimeInForce:           TimeInForceGoodTillCancel,
		Side:                  SideSell,
		ExecutionInstructions: []string{ExecutionInstructionReduceOnly, ExecutionInstructionPostOnly},
		Quantity:              preciseNumber("0.01"),
		LimitPrice:            preciseNumber("50000"),
		OrderValue:            preciseNumber("500"),
		AveragePrice:          preciseNumber("50001"),
		CumulativeQuantity:    preciseNumber("0.004"),
		CumulativeValue:       preciseNumber("200.004"),
		CumulativeFee:         preciseNumber("0.05"),
		Status:                OrderStatusActive,
		FeeInstrumentName:     currency.USD,
		CreateTime:            types.Time(time.UnixMilli(1613575617173)),
		CreateTimeNano:        types.Time(time.Unix(0, 1613575617173123456)),
		UpdateTime:            types.Time(time.UnixMilli(1613575618173)),
		ReferencePrice:        preciseNumber("49000"),
		Leverage:              10,
		IsolationID:           "19848526",
	}
	got, err := e.orderDetail(o, asset.PerpetualSwap, testPerpetualPair)
	require.NoError(t, err, "orderDetail must not error")
	exp := &order.Detail{
		Exchange:             e.Name,
		OrderID:              "19848525",
		ClientOrderID:        "1613571154900",
		AccountID:            "52e7c00f-1324-5a6z-bfgt-de445bde21a5",
		Pair:                 testPerpetualPair,
		AssetType:            asset.PerpetualSwap,
		Side:                 order.Sell,
		Type:                 order.StopLimit,
		Status:               order.PartiallyFilled,
		TimeInForce:          order.GoodTillCancel | order.PostOnly,
		ReduceOnly:           true,
		Price:                50000,
		Amount:               0.01,
		TriggerPrice:         49000,
		AverageExecutedPrice: 50001,
		ExecutedAmount:       0.004,
		RemainingAmount:      0.006,
		Cost:                 200.004,
		CostAsset:            currency.USD,
		Fee:                  0.05,
		FeeAsset:             currency.USD,
		Leverage:             10,
		Date:                 time.Unix(0, 1613575617173123456),
		LastUpdated:          time.UnixMilli(1613575618173),
		MarginType:           margin.Isolated,
		SettlementCurrency:   currency.USD,
	}
	assert.Equal(t, exp, got, "orderDetail should map every field of an open order on an isolated position")

	closed := *o
	closed.Status = OrderStatusFilled
	closed.CumulativeQuantity = preciseNumber("0.01")
	closed.CreateTimeNano = types.Time{}
	closed.IsolationID = "0"
	got, err = e.orderDetail(&closed, asset.PerpetualSwap, testPerpetualPair)
	require.NoError(t, err, "orderDetail must not error for a filled order")
	exp.Status = order.Filled
	exp.ExecutedAmount = 0.01
	exp.RemainingAmount = 0
	exp.Date = time.UnixMilli(1613575617173)
	exp.CloseTime = time.UnixMilli(1613575618173)
	exp.MarginType = margin.Multi
	assert.Equal(t, exp, got, "orderDetail should close a filled cross margin order at its last update, dated by its millisecond creation time without a nanosecond one")

	notional := &OrderResponse{OrderID: "1", OrderType: OrderTypeMarket, Side: SideBuy, Status: OrderStatusFilled, OrderValue: preciseNumber("100"), CumulativeQuantity: preciseNumber("0.002"), CumulativeValue: preciseNumber("100")}
	got, err = e.orderDetail(notional, asset.Spot, testSpotPair)
	require.NoError(t, err, "orderDetail must not error for a market buy by notional")
	exp = &order.Detail{Exchange: e.Name, OrderID: "1", Pair: testSpotPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Status: order.Filled, ExecutedAmount: 0.002, QuoteAmount: 100, Cost: 100, CostAsset: currency.USD}
	assert.Equal(t, exp, got, "orderDetail should give a market buy by notional its notional as the quote amount, its quantity unset")
	// A stop loss or take profit buy is a market buy once triggered, which can spend a notional too; the cases run in
	// order on e, as orderDetail is called on it again below
	for orderType, typ := range map[string]order.Type{OrderTypeStopLoss: order.Stop, OrderTypeTakeProfit: order.TakeProfit} {
		t.Run(orderType, func(t *testing.T) {
			trigger := &OrderResponse{OrderID: "2", OrderType: orderType, Side: SideBuy, Status: OrderStatusPending, Quantity: preciseNumber("0"), OrderValue: preciseNumber("7600"), ReferencePrice: preciseNumber("76000")}
			got, err := e.orderDetail(trigger, asset.Spot, testSpotPair)
			require.NoErrorf(t, err, "orderDetail must not error for a %s buy by notional", orderType)
			expTrigger := &order.Detail{Exchange: e.Name, OrderID: "2", Pair: testSpotPair, AssetType: asset.Spot, Side: order.Buy, Type: typ, Status: order.Pending, TriggerPrice: 76000, QuoteAmount: 7600, CostAsset: currency.USD}
			assert.Equalf(t, expTrigger, got, "orderDetail should give a %s buy by notional its notional as the quote amount", orderType)
		})
	}
	// A trigger buy placed by its quantity keeps its quantity, and its order value is no quote amount
	byQuantity := &OrderResponse{OrderID: "3", OrderType: OrderTypeStopLoss, Side: SideBuy, Status: OrderStatusPending, Quantity: preciseNumber("0.1"), OrderValue: preciseNumber("7600"), ReferencePrice: preciseNumber("76000")}
	quantityDetail, quantityErr := e.orderDetail(byQuantity, asset.Spot, testSpotPair)
	require.NoError(t, quantityErr, "orderDetail must not error for a stop loss buy by quantity")
	assert.Equal(t, 0.1, quantityDetail.Amount, "orderDetail should give a stop loss buy by quantity its quantity")
	assert.Zero(t, quantityDetail.QuoteAmount, "orderDetail should give a stop loss buy by quantity no quote amount")

	t.Run("executed beyond its quantity", func(t *testing.T) {
		overfilled := closed
		overfilled.CumulativeQuantity = preciseNumber("0.02")
		got, err := e.orderDetail(&overfilled, asset.PerpetualSwap, testPerpetualPair)
		require.NoError(t, err, "orderDetail must not error for an order executed beyond its quantity")
		assert.Zero(t, got.RemainingAmount, "orderDetail should leave nothing remaining of an order executed beyond its quantity")
	})

	// The refusals run in order on e, after the calls above, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		name string
		o    OrderResponse
		err  error
	}{
		{"an unknown side", OrderResponse{Side: "HOLD", OrderType: OrderTypeLimit, Status: OrderStatusNew}, order.ErrSideIsInvalid},
		{"an unknown type", OrderResponse{Side: SideBuy, OrderType: "TRAILING", Status: OrderStatusNew}, order.ErrUnrecognisedOrderType},
		{"an unknown status", OrderResponse{Side: SideBuy, OrderType: OrderTypeLimit, Status: "SUSPENDED"}, order.ErrUnsupportedStatusType},
		{"an unknown time in force", OrderResponse{Side: SideBuy, OrderType: OrderTypeLimit, Status: OrderStatusNew, TimeInForce: "GOOD_TILL_DATE"}, order.ErrInvalidTimeInForce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.orderDetail(&tc.o, asset.Spot, testSpotPair)
			assert.ErrorIsf(t, err, tc.err, "orderDetail should refuse %+v", tc.o)
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Options, testSpotPair)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must refuse an unsupported asset")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must refuse an empty pair")
	// The cases run in order on e, after the refusals above, so each also checks that a call keeps nothing from the last
	for a, exp := range map[asset.Item]string{
		asset.Spot:          "https://crypto.com/exchange/trade/BTC_USD",
		asset.Margin:        "https://crypto.com/exchange/trade/BTC_USD",
		asset.PerpetualSwap: "https://crypto.com/exchange/trade/BTCUSD-PERP",
		asset.Futures:       "https://crypto.com/exchange/trade/BTCUSD-261030",
	} {
		t.Run(a.String(), func(t *testing.T) {
			got, err := e.GetCurrencyTradeURL(t.Context(), a, testInstrumentPairs[a][0])
			require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
			assert.Equalf(t, exp, got, "GetCurrencyTradeURL should return the %s pair's trading page", a)
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) { //nolint:tparallel // the assets request the instruments through e in order
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must refuse a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must refuse an unsupported futures asset")
	// The assets request the instruments through e in order, so each also checks that a call keeps nothing from the last
	for _, a := range []asset.Item{asset.PerpetualSwap, asset.Futures} {
		t.Run(a.String(), func(t *testing.T) {
			got, err := e.GetFuturesContractDetails(t.Context(), a)
			require.NoErrorf(t, err, "GetFuturesContractDetails must not error for %s", a)
			if !mockTests {
				assert.NotEmptyf(t, got, "GetFuturesContractDetails should return %s contracts", a)
				return
			}
			exp := []futures.Contract{{
				Exchange:           e.Name,
				Name:               testPerpetualPair,
				Underlying:         currency.NewBTCUSD(),
				Asset:              asset.PerpetualSwap,
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USD,
				MarginCurrency:     currency.USD,
				Multiplier:         1,
				MaxLeverage:        100,
			}}
			if a == asset.Futures {
				exp[0].Name, exp[0].Asset, exp[0].Type, exp[0].EndDate = testFuturesPair, asset.Futures, futures.LongDated, time.UnixMilli(1793347200000)
			}
			assert.Equalf(t, exp, got, "GetFuturesContractDetails should return every field of the %s contract", a)
		})
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	// The cases run in order on e, so each also checks that a call keeps nothing from the last
	for a, exp := range map[asset.Item]bool{asset.PerpetualSwap: true, asset.Futures: false, asset.Spot: false} {
		t.Run(a.String(), func(t *testing.T) {
			got, err := e.IsPerpetualFutureCurrency(a, testInstrumentPairs[asset.PerpetualSwap][0])
			require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
			assert.Equalf(t, exp, got, "IsPerpetualFutureCurrency should tell whether %s is perpetual", a)
		})
	}
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	_, _, err := e.GetCollateralCurrencyForContract(asset.Spot, testSpotPair)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetCollateralCurrencyForContract must refuse a spot asset")
	c, a, err := e.GetCollateralCurrencyForContract(asset.Futures, testFuturesPair)
	require.NoError(t, err, "GetCollateralCurrencyForContract must not error")
	assert.Equal(t, currency.USD, c, "GetCollateralCurrencyForContract should margin a contract in USD")
	assert.Equal(t, asset.Futures, a, "GetCollateralCurrencyForContract should return the contract's asset")
}

func TestGetCollateralMode(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	_, err := e.GetCollateralMode(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCollateralMode must refuse spot")
	// The cases run in order on e, after the refusal above, so each also checks that a call keeps nothing from the last
	for _, a := range []asset.Item{asset.Margin, asset.PerpetualSwap, asset.Futures} {
		t.Run(a.String(), func(t *testing.T) {
			got, err := e.GetCollateralMode(t.Context(), a)
			require.NoErrorf(t, err, "GetCollateralMode must not error for %s", a)
			assert.Equalf(t, collateral.MultiMode, got, "GetCollateralMode should margin %s with every collateral eligible currency", a)
		})
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must refuse a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates must refuse a dated futures asset")
	got, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, IncludePredictedRate: true})
	require.NoError(t, err, "GetLatestFundingRates must not error")
	require.Len(t, got, 1, "GetLatestFundingRates must return the pair's rate")
	assert.NotZero(t, got[0].TimeChecked, "GetLatestFundingRates should time when the rate was checked")
	if !mockTests {
		return
	}
	got[0].TimeChecked = time.Time{}
	exp := []fundingrate.LatestRateResponse{{
		Exchange:              e.Name,
		Asset:                 asset.PerpetualSwap,
		Pair:                  testPerpetualPair,
		LatestRate:            fundingrate.Rate{Time: time.UnixMilli(1792899065000), Rate: numberDecimal(0.000009839)},
		PredictedUpcomingRate: fundingrate.Rate{Time: time.UnixMilli(1792899059000), Rate: numberDecimal(0.000006556)},
		TimeOfNextRate:        time.UnixMilli(1792899065000).Truncate(time.Hour).Add(time.Hour),
	}}
	assert.Equal(t, exp, got, "GetLatestFundingRates should return the current rate, settling at the end of the hour, and the estimated next rate")

	all, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualSwap})
	require.NoError(t, err, "GetLatestFundingRates must not error without a pair")
	enabled, err := e.GetEnabledPairs(asset.PerpetualSwap)
	require.NoError(t, err, "GetEnabledPairs must not error")
	assert.Len(t, all, len(enabled), "GetLatestFundingRates should return every enabled perpetual swap's rate without a pair")
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must refuse a nil request")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetHistoricalFundingRates must refuse an asset other than perpetual swaps")
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, StartDate: testWindowStart})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetHistoricalFundingRates must refuse a window without an end")
	end := testWindowStart.Add(3 * time.Hour)
	got, err := e.GetHistoricalFundingRates(mockRequestContext(t), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, StartDate: testWindowStart, EndDate: end})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	if !mockTests {
		assert.NotEmpty(t, got.FundingRates, "GetHistoricalFundingRates should return rates")
		return
	}
	rates := []fundingrate.Rate{
		{Time: testWindowStart, Rate: numberDecimal(0.000007063)},
		{Time: testWindowStart.Add(time.Hour), Rate: numberDecimal(0.000008003)},
		{Time: testWindowStart.Add(2 * time.Hour), Rate: numberDecimal(0.000008003)},
	}
	exp := &fundingrate.HistoricalRates{
		Exchange:       e.Name,
		Asset:          asset.PerpetualSwap,
		Pair:           testPerpetualPair,
		StartDate:      testWindowStart,
		EndDate:        end,
		LatestRate:     rates[2],
		FundingRates:   rates,
		TimeOfNextRate: testWindowStart.Add(3 * time.Hour),
	}
	assert.Equal(t, exp, got, "GetHistoricalFundingRates should return the window's settled rates, oldest first")

	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"public/get-valuations": func(params string) (string, *APIError) {
			if strings.Contains(params, "estimated_funding_rate") {
				return `{"instrument_name":"BTCUSD-PERP","data":[{"v":"0.0000045","t":1792890479000}]}`, nil
			}
			return `{"instrument_name":"BTCUSD-PERP","data":[{"v":"0.00001","t":1791504000000}]}`, nil
		},
		"private/get-transactions": testResult(`{"data":[
			{"journal_id":"1","journal_type":"FUNDING","transaction_qty":"-0.0805","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1791504000123","event_timestamp_ns":"1791504000123456789"},
			{"journal_id":"2","journal_type":"FUNDING","transaction_qty":"0.012","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1791504300000"}]}`),
	})
	// A window of 15 days, past by the time of the recordings
	windowStart := testWindowStart.Add(-15 * 24 * time.Hour)
	got, err = ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, StartDate: windowStart, EndDate: testWindowStart, IncludePredictedRate: true, IncludePayments: true})
	require.NoError(t, err, "GetHistoricalFundingRates must not error with payments")
	// Both payments are made in the hour the window's one rate settles, so are that rate's payment
	payment := decimal.Decimal{}.Add(preciseNumber("-0.0805").Decimal()).Add(preciseNumber("0.012").Decimal())
	rate := fundingrate.Rate{Time: windowStart, Rate: numberDecimal(0.00001), Payment: payment}
	exp = &fundingrate.HistoricalRates{
		Exchange:              ex.Name,
		Asset:                 asset.PerpetualSwap,
		Pair:                  testPerpetualPair,
		StartDate:             windowStart,
		EndDate:               testWindowStart,
		LatestRate:            rate,
		PredictedUpcomingRate: fundingrate.Rate{Time: time.UnixMilli(1792890479000), Rate: numberDecimal(0.0000045)},
		FundingRates:          []fundingrate.Rate{rate},
		PaymentSum:            payment,
		PaymentCurrency:       currency.USD,
		TimeOfNextRate:        windowStart.Add(time.Hour),
	}
	assert.Equal(t, exp, got, "GetHistoricalFundingRates should sum the window's funding payments and pay each to the rate of its hour")
	var windows []string
	for _, r := range received() {
		if r.Method == "public/get-valuations" && strings.Contains(r.Params, "funding_hist") {
			windows = append(windows, r.Params)
		}
	}
	assert.Equal(t, []string{
		"count=1000&end_ts=1792800000000&instrument_name=BTCUSD-PERP&start_ts=1791590400000&valuation_type=funding_hist",
		"count=1000&end_ts=1791590400000&instrument_name=BTCUSD-PERP&start_ts=1791504000000&valuation_type=funding_hist",
	}, windows, "GetHistoricalFundingRates should request the window 14 days at a time from its end")
}

func TestFundingPayments(t *testing.T) {
	t.Parallel()
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-transactions": testResult(`{"data":[
			{"journal_id":"3","journal_type":"FUNDING","transaction_qty":"0.004","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1792803900000"},
			{"journal_id":"2","journal_type":"FUNDING","transaction_qty":"0.012","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1792800300000"},
			{"journal_id":"1","journal_type":"FUNDING","transaction_qty":"-0.0805","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1792800000123","event_timestamp_ns":"1792800000123456789"}]}`),
	})
	end := testWindowStart.Add(2 * time.Hour)
	history := func() *fundingrate.HistoricalRates {
		return &fundingrate.HistoricalRates{
			Exchange:     ex.Name,
			Asset:        asset.PerpetualSwap,
			Pair:         testPerpetualPair,
			StartDate:    testWindowStart,
			EndDate:      end,
			FundingRates: []fundingrate.Rate{{Time: testWindowStart, Rate: numberDecimal(0.00001)}},
		}
	}
	got := history()
	require.NoError(t, ex.fundingPayments(t.Context(), "BTCUSD-PERP", got), "fundingPayments must not error")
	exp := history()
	exp.FundingRates[0].Payment = decimal.Decimal{}.Add(preciseNumber("0.012").Decimal()).Add(preciseNumber("-0.0805").Decimal())
	// The payment of 0.004 is made in an hour whose rate the history does not hold, so it is summed but paid to no rate
	exp.PaymentSum = decimal.Decimal{}.Add(preciseNumber("0.004").Decimal()).Add(preciseNumber("0.012").Decimal()).Add(preciseNumber("-0.0805").Decimal())
	exp.PaymentCurrency = currency.USD
	assert.Equal(t, exp, got, "fundingPayments should sum the window's funding payments and pay each to the rate of its hour")
	expRequests := []testRequest{{Method: "private/get-transactions", Params: `{"instrument_name":"BTCUSD-PERP","journal_type":"FUNDING","start_time":"1792800000000000000","end_time":"1792807200000000000","limit":100}`}}
	assert.Equal(t, expRequests, received(), "fundingPayments should request the instrument's funding transactions over the window")

	// Crypto.com sends a journal ID bare or quoted, and above 2^53, so a payment is known by every digit of its ID: one ID
	// in both forms is one payment, and the ID after it another
	ex, _ = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-transactions": testResult(`{"data":[
			{"journal_id":9007199254740992,"journal_type":"FUNDING","transaction_qty":"1","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1792800060000"},
			{"journal_id":"9007199254740992","journal_type":"FUNDING","transaction_qty":"1","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1792800060000"},
			{"journal_id":9007199254740993,"journal_type":"FUNDING","transaction_qty":"2","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1792800060000"}]}`),
	})
	got = history()
	require.NoError(t, ex.fundingPayments(t.Context(), "BTCUSD-PERP", got), "fundingPayments must not error for journal IDs in both forms")
	exp = history()
	exp.FundingRates[0].Payment = decimal.Decimal{}.Add(preciseNumber("1").Decimal()).Add(preciseNumber("2").Decimal())
	exp.PaymentSum = exp.FundingRates[0].Payment
	exp.PaymentCurrency = currency.USD
	assert.Equal(t, exp, got, "fundingPayments should count a payment once whichever form its journal ID takes, and tell apart IDs above 2^53")

	ex, _ = newRoutesServerExchange(t, map[string]testRoute{"private/get-transactions": testResult(`{"data":[]}`)})
	got = history()
	require.NoError(t, ex.fundingPayments(t.Context(), "BTCUSD-PERP", got), "fundingPayments must not error for a window without payments")
	exp = history()
	exp.PaymentCurrency = currency.USD
	assert.Equal(t, exp, got, "fundingPayments should report a window without payments in USD")

	got = history()
	err := newRefusingServerExchange(t).fundingPayments(t.Context(), "BTCUSD-PERP", got)
	assert.ErrorIs(t, err, request.ErrBadStatus, "fundingPayments should return the error requesting the funding transactions")
	assert.Equal(t, history(), got, "fundingPayments should leave the history unchanged when it cannot request the funding transactions")

	// A span failing after one that returned a payment to the history's rate fails the window, which then keeps none of
	// its payments; later requests get an empty list, so a pager retrying the failure returns rather than hangs the test
	var requests atomic.Int64
	ex, _ = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-transactions": func(string) (string, *APIError) {
			switch requests.Add(1) {
			case 1:
				return `{"data":[{"journal_id":"1","journal_type":"FUNDING","transaction_qty":"3","instrument_name":"BTCUSD-PERP","event_timestamp_ms":"1792800060000"}]}`, nil
			case 2:
				return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}
			default:
				return `{"data":[]}`, nil
			}
		},
	})
	got = history()
	got.StartDate = end.Add(-2 * privateHistoryWindow)
	exp = history()
	exp.StartDate = got.StartDate
	err = ex.fundingPayments(t.Context(), "BTCUSD-PERP", got)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "fundingPayments should return the error requesting a later span of the window")
	assert.Equal(t, exp, got, "fundingPayments should leave the history unchanged when a later span of the window fails")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.BTC.Item, Quote: currency.USD.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetOpenInterest must refuse a spot pair")
	shazPerpetual := currency.NewPairWithDelimiter("SHAZUSD", "PERP", currency.DashDelimiter)
	k := key.PairAsset{Base: shazPerpetual.Base.Item, Quote: shazPerpetual.Quote.Item, Asset: asset.PerpetualSwap}
	if !mockTests {
		p := testPair(t, asset.PerpetualSwap)
		k = key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: asset.PerpetualSwap}
	}
	got, err := e.GetOpenInterest(t.Context(), k)
	require.NoError(t, err, "GetOpenInterest must not error")
	require.Len(t, got, 1, "GetOpenInterest must return the pair's open interest")
	if mockTests {
		exp := []futures.OpenInterest{{Key: key.ExchangeAssetPair{Exchange: e.Name, Base: k.Base, Quote: k.Quote, Asset: asset.PerpetualSwap}, OpenInterest: 5.6}}
		assert.Equal(t, exp, got, "GetOpenInterest should return the pair's open interest from its ticker")
	}
	_, err = e.GetOpenInterest(t.Context())
	assert.NoError(t, err, "GetOpenInterest should not error for every enabled derivative")
}

// testPositionRoutes answer the requests of the position tests as REST does, without the margins, prices and leverage
// only the websocket's position channels carry: a cross margin and an isolated BTCUSD-PERP position, the account's
// balance holding another isolated position's and then the isolated position's, its settings and the instrument's mark
// price
var testPositionRoutes = map[string]testRoute{
	"private/get-positions": testResult(`{"data":[
		{"account_id":"858dbc8b-22fd-49fa-bff4-d342d98a8acb","quantity":"-0.1984","cost":"-10159.573500","open_position_pnl":"-497.743736","open_pos_cost":"-10159.352200","session_pnl":"2.236145","update_timestamp_ms":"1613552240770","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP"},
		{"account_id":"858dbc8b-22fd-49fa-bff4-d342d98a8acb","quantity":"0.0002","cost":"10.959000","open_position_pnl":"-0.003000","open_pos_cost":"10.959000","session_pnl":"-0.043000","update_timestamp_ms":"1613552240771","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"19848526","isolation_type":"ISOLATED_MARGIN"}]}`),
	"private/user-balance": testResult(`{"data":[{"instrument_name":"USD","total_available_balance":"4721.05898582","total_margin_balance":"7595.42571782","total_cash_balance":"7890.00320721","total_collateral_value":"7651.18811483","total_effective_leverage":"1.90401230",
		"position_balances":[{"instrument_name":"USD","quantity":"3113.50747209","reserved_qty":"12.5"}],
		"isolated_positions":[
			{"instrument_name":"USD","isolation_id":"19848527","isolation_type":"ISOLATED_MARGIN","leverage":"5","total_available_balance":"1.1","total_margin_balance":"3.2","total_initial_margin":"0.64","total_maintenance_margin":"0.32","total_cash_balance":"3.2","total_collateral_value":"3.2","total_effective_leverage":"0.2"},
			{"instrument_name":"USD","isolation_id":"19848526","isolation_type":"ISOLATED_MARGIN","leverage":"2","total_available_balance":"0.4","total_margin_balance":"5.397","total_initial_margin":"2.7","total_maintenance_margin":"1.35","total_cash_balance":"5.4","total_collateral_value":"5.4","total_effective_leverage":"0.4"}]}]}`),
	"private/get-account-settings": testResult(`[{"leverage":"20","stp_id":"100","stp_scope":"S","stp_inst":"M"}]`),
	"public/get-valuations":        testResult(`{"instrument_name":"BTCUSD-PERP","data":[{"v":"80819.5","t":1792899065000}]}`),
	"public/get-instruments":       testResult(`{"data":[{"symbol":"BTCUSD-PERP","inst_type":"PERPETUAL_SWAP","max_leverage":"10","tradable":true}]}`),
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary must refuse a nil request")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spot, Pair: testSpotPair})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesPositionSummary must refuse a spot pair")
	if !mockTests {
		t.Run("live", func(t *testing.T) {
			t.Parallel()
			skipLiveWithoutCredentials(t)
			_, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: testPair(t, asset.PerpetualSwap)})
			if err != nil {
				assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetFuturesPositionSummary should only error for a pair without a position")
			}
		})
	}

	// The isolated position is listed first, so that only the preference for the cross margin position finds that
	cross := maps.Clone(testPositionRoutes)
	cross["private/get-positions"] = testResult(`{"data":[
		{"account_id":"858dbc8b-22fd-49fa-bff4-d342d98a8acb","quantity":"0.0002","cost":"10.959000","open_position_pnl":"-0.003000","open_pos_cost":"10.959000","session_pnl":"-0.043000","update_timestamp_ms":"1613552240771","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"19848526","isolation_type":"ISOLATED_MARGIN"},
		{"account_id":"858dbc8b-22fd-49fa-bff4-d342d98a8acb","quantity":"-0.1984","cost":"-10159.573500","open_position_pnl":"-497.743736","open_pos_cost":"-10159.352200","session_pnl":"2.236145","update_timestamp_ms":"1613552240770","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP"}]}`)
	ex, _ := newRoutesServerExchange(t, cross)
	got, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair})
	require.NoError(t, err, "GetFuturesPositionSummary must not error")
	exp := &futures.PositionSummary{
		Pair:                   testPerpetualPair,
		Asset:                  asset.PerpetualSwap,
		MarginType:             margin.Multi,
		CollateralMode:         collateral.MultiMode,
		Currency:               currency.USD,
		AvailableEquity:        decimal.MustFromString("4721.05898582"),
		CashBalance:            decimal.MustFromString("7890.00320721"),
		NotionalLeverage:       numberDecimal(1.9040123),
		MarginBalance:          decimal.MustFromString("7595.42571782"),
		NotionalSize:           decimal.MustFromString("-0.1984").Abs().Mul(numberDecimal(80819.5)),
		Leverage:               numberDecimal(10),
		MarkPrice:              numberDecimal(80819.5),
		CurrentSize:            decimal.MustFromString("-0.1984"),
		ContractSettlementType: futures.Linear,
		AverageOpenPrice:       decimal.MustFromString("-10159.352200").Div(decimal.MustFromString("-0.1984")).Abs(),
		UnrealisedPNL:          decimal.MustFromString("-497.743736"),
		FreeCollateral:         decimal.MustFromString("4721.05898582"),
		TotalCollateral:        decimal.MustFromString("7651.18811483"),
	}
	assert.Equal(t, exp, got, "GetFuturesPositionSummary should summarise the cross margin position with the account's margin, the lower of the account's and the instrument's maximum leverage, valued at the mark price")

	isolated := maps.Clone(testPositionRoutes)
	isolated["private/get-positions"] = testResult(`{"data":[{"quantity":"0.0002","open_pos_cost":"10.959000","open_position_pnl":"-0.003000","instrument_name":"BTCUSD-PERP","isolation_id":"19848526","isolation_type":"ISOLATED_MARGIN"}]}`)
	ex, _ = newRoutesServerExchange(t, isolated)
	got, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair})
	require.NoError(t, err, "GetFuturesPositionSummary must not error for an isolated position")
	exp = &futures.PositionSummary{
		Pair:                         testPerpetualPair,
		Asset:                        asset.PerpetualSwap,
		MarginType:                   margin.Isolated,
		CollateralMode:               collateral.MultiMode,
		Currency:                     currency.USD,
		AvailableEquity:              decimal.MustFromString("0.4"),
		CashBalance:                  decimal.MustFromString("5.4"),
		NotionalLeverage:             numberDecimal(0.4),
		MarginBalance:                decimal.MustFromString("5.397"),
		IsolatedMargin:               decimal.MustFromString("5.397"),
		NotionalSize:                 decimal.MustFromString("0.0002").Abs().Mul(numberDecimal(80819.5)),
		Leverage:                     numberDecimal(2),
		MaintenanceMarginRequirement: decimal.MustFromString("1.35"),
		InitialMarginRequirement:     decimal.MustFromString("2.7"),
		MarkPrice:                    numberDecimal(80819.5),
		CurrentSize:                  decimal.MustFromString("0.0002"),
		ContractSettlementType:       futures.Linear,
		AverageOpenPrice:             decimal.MustFromString("10.959000").Div(decimal.MustFromString("0.0002")).Abs(),
		UnrealisedPNL:                decimal.MustFromString("-0.003000"),
		FreeCollateral:               decimal.MustFromString("0.4"),
		TotalCollateral:              decimal.MustFromString("5.4"),
	}
	assert.Equal(t, exp, got, "GetFuturesPositionSummary should summarise an isolated position with its own margin, margin requirements and leverage")

	orphan := maps.Clone(isolated)
	orphan["private/user-balance"] = testResult(`{"data":[{"instrument_name":"USD","isolated_positions":[]}]}`)
	ex, _ = newRoutesServerExchange(t, orphan)
	_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair})
	assert.ErrorIs(t, err, errNoIsolatedPosition, "GetFuturesPositionSummary should error for an isolated position the balance does not hold")

	ex = newResponseServerExchange(t, `{"id":1,"method":"private/get-positions","code":0,"result":{"data":[]}}`)
	_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair})
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetFuturesPositionSummary should error without a position")
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionOrders must refuse a nil request")
	_, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.PerpetualSwap})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetFuturesPositionOrders must refuse a request without pairs")
	skipLiveWithoutCredentials(t)
	ex := e
	if mockTests {
		ex, _ = newRoutesServerExchange(t, map[string]testRoute{
			"private/get-order-history":          testResult(`{"data":[{"order_id":"18342500","order_type":"LIMIT","side":"BUY","quantity":"0.0500","limit_price":"51283.0","avg_price":"51278.5","cumulative_quantity":"0.0500","cumulative_value":"2563.925","status":"FILLED","instrument_name":"BTCUSD-PERP","create_time":"1792807200000","update_time":"1792807200100"}]}`),
			"private/advanced/get-order-history": testResult(`{"data":[]}`),
		})
	}
	got, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.PerpetualSwap, Pairs: currency.Pairs{testPair(t, asset.PerpetualSwap)}, StartDate: testWindowStart, EndDate: testWindowStart.Add(12 * time.Hour)})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	if !mockTests {
		return
	}
	exp := []futures.PositionResponse{{
		Pair:                   testPerpetualPair,
		Asset:                  asset.PerpetualSwap,
		ContractSettlementType: futures.Linear,
		Orders: []order.Detail{{
			Price:                51283,
			Amount:               0.05,
			AverageExecutedPrice: 51278.5,
			ExecutedAmount:       0.05,
			Cost:                 2563.925,
			CostAsset:            currency.USD,
			Exchange:             ex.Name,
			OrderID:              "18342500",
			Type:                 order.Limit,
			Side:                 order.Buy,
			Status:               order.Filled,
			AssetType:            asset.PerpetualSwap,
			Date:                 time.UnixMilli(1792807200000),
			CloseTime:            time.UnixMilli(1792807200100),
			LastUpdated:          time.UnixMilli(1792807200100),
			Pair:                 testPerpetualPair,
			MarginType:           margin.Multi,
			SettlementCurrency:   currency.USD,
		}},
	}}
	assert.Equal(t, exp, got, "GetFuturesPositionOrders should return the pair's orders")
}

func TestIsIsolated(t *testing.T) {
	t.Parallel()
	for id, exp := range map[string]bool{"": false, "0": false, "19848526": true} {
		t.Run(subtestName(id), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, isIsolated(id), "isIsolated should tell whether isolation ID %q names an isolated position", id)
		})
	}
}

func TestIsolatedPositionID(t *testing.T) { //nolint:tparallel // the cases run in order, before the failed request
	t.Parallel()
	const (
		isolated      = `{"quantity":"0.0002","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"19848526","isolation_type":"ISOLATED_MARGIN"}`
		otherIsolated = `{"quantity":"0.01","instrument_name":"ETHUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"19848530","isolation_type":"ISOLATED_MARGIN"}`
	)
	// The cases run in order, before isolatedPositionID's failed request below
	for _, tc := range []struct {
		name      string
		positions testRoute
		exp       string
	}{
		{"an isolated position listed after the cross margin position", testPositionRoutes["private/get-positions"], "19848526"},
		{"an isolated position listed first", testResult(`{"data":[` + isolated + `,{"quantity":"-0.1984","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP"}]}`), "19848526"},
		{"an isolated position listed after another instrument's", testResult(`{"data":[` + otherIsolated + `,` + isolated + `]}`), "19848526"},
		{"a cross margin position of isolation ID 0", testResult(`{"data":[{"quantity":"-0.1984","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"0"}]}`), ""},
		{"a cross margin position without an isolation ID", testResult(`{"data":[{"quantity":"-0.1984","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP"}]}`), ""},
		{"another instrument's isolated position", testResult(`{"data":[` + otherIsolated + `]}`), ""},
		{"no position", testResult(`{"data":[]}`), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-positions": tc.positions})
			got, err := ex.isolatedPositionID(t.Context(), "BTCUSD-PERP")
			require.NoErrorf(t, err, "isolatedPositionID must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "isolatedPositionID should return the isolation ID of the instrument's isolated position, if any, for %s", tc.name)
			assert.Equalf(t, []testRequest{{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`}}, received(), "isolatedPositionID should request the instrument's positions for %s", tc.name)
		})
	}
	_, err := newUnavailableServerExchange(t).isolatedPositionID(t.Context(), "BTCUSD-PERP")
	assert.ErrorIs(t, err, request.ErrBadStatus, "isolatedPositionID should return the error requesting the positions")
}

func TestIsolatedPosition(t *testing.T) {
	t.Parallel()
	ex, received := newRoutesServerExchange(t, testPositionRoutes)
	got, err := ex.isolatedPosition(t.Context(), "BTCUSD-PERP")
	require.NoError(t, err, "isolatedPosition must not error")
	exp := &IsolatedPositionBalance{
		InstrumentName:         "USD",
		TotalAvailableBalance:  preciseNumber("0.4"),
		TotalMarginBalance:     preciseNumber("5.397"),
		TotalInitialMargin:     preciseNumber("2.7"),
		TotalMaintenanceMargin: preciseNumber("1.35"),
		TotalCashBalance:       preciseNumber("5.4"),
		TotalCollateralValue:   preciseNumber("5.4"),
		TotalEffectiveLeverage: 0.4,
		IsolationID:            "19848526",
		Leverage:               2,
		IsolationType:          "ISOLATED_MARGIN",
	}
	assert.Equal(t, exp, got, "isolatedPosition should return the balance of the instrument's isolated position, found by its isolation ID")
	lookups := []testRequest{
		{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`},
		{Method: "private/user-balance", Params: `{}`},
	}
	assert.Equal(t, lookups, received(), "isolatedPosition should look up the instrument's positions, then the account's balance")

	_, err = newUnavailableServerExchange(t).isolatedPosition(t.Context(), "BTCUSD-PERP")
	assert.ErrorIs(t, err, request.ErrBadStatus, "isolatedPosition should return the error requesting the positions")

	cross := maps.Clone(testPositionRoutes)
	cross["private/get-positions"] = testResult(`{"data":[{"quantity":"-0.1984","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"0"}]}`)
	ex, received = newRoutesServerExchange(t, cross)
	_, err = ex.isolatedPosition(t.Context(), "BTCUSD-PERP")
	assert.ErrorIs(t, err, errNoIsolatedPosition, "isolatedPosition should error without an isolated position")
	assert.Equal(t, lookups[:1], received(), "isolatedPosition should not request the balance without an isolated position")

	balanceFails := maps.Clone(testPositionRoutes)
	balanceFails["private/user-balance"] = func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} }
	ex, _ = newRoutesServerExchange(t, balanceFails)
	_, err = ex.isolatedPosition(t.Context(), "BTCUSD-PERP")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "isolatedPosition must return the error requesting the balance")
	assert.Equal(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "isolatedPosition should return Crypto.com's error requesting the balance")

	orphan := maps.Clone(testPositionRoutes)
	orphan["private/user-balance"] = testResult(`{"data":[{"instrument_name":"USD","isolated_positions":[{"instrument_name":"USD","isolation_id":"19848527","isolation_type":"ISOLATED_MARGIN","leverage":"5"}]}]}`)
	ex, received = newRoutesServerExchange(t, orphan)
	_, err = ex.isolatedPosition(t.Context(), "BTCUSD-PERP")
	assert.ErrorIs(t, err, errNoIsolatedPosition, "isolatedPosition should error for an isolated position the balance does not hold")
	assert.ErrorContains(t, err, "19848526", "isolatedPosition should name the isolated position the balance does not hold")
	assert.Equal(t, lookups, received(), "isolatedPosition should look the isolated position up in the account's balance")
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.Spot, testSpotPair, margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetLeverage must refuse a spot pair")
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		got, err := e.GetLeverage(t.Context(), asset.PerpetualSwap, testPair(t, asset.PerpetualSwap), margin.Multi, order.UnknownSide)
		require.NoError(t, err, "GetLeverage must not error")
		if !mockTests {
			assert.Positive(t, got, "GetLeverage should return the account's leverage")
			return
		}
		assert.Equal(t, 20.0, got, "GetLeverage should return the account's leverage for a cross margin position")
	})

	ex, _ := newRoutesServerExchange(t, testPositionRoutes)
	got, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Isolated, order.UnknownSide)
	require.NoError(t, err, "GetLeverage must not error for an isolated position")
	assert.Equal(t, 2.0, got, "GetLeverage should return an isolated position's leverage")
	_, err = ex.GetLeverage(t.Context(), asset.PerpetualSwap, currency.NewPairWithDelimiter("ETHUSD", "PERP", currency.DashDelimiter), margin.Isolated, order.UnknownSide)
	assert.ErrorIs(t, err, errNoIsolatedPosition, "GetLeverage should error without an isolated position")

	t.Run("the contract cap limits cross margin leverage", func(t *testing.T) {
		t.Parallel()
		exch, received := newRoutesServerExchange(t, map[string]testRoute{
			"private/get-account-settings": testResult(`[{"leverage":"50"}]`),
			"public/get-instruments":       testResult(`{"data":[{"symbol":"BTCUSD-PERP","inst_type":"PERPETUAL_SWAP","max_leverage":"20","tradable":true},{"symbol":"ETHUSD-PERP","inst_type":"PERPETUAL_SWAP","max_leverage":"10","tradable":true}]}`),
		})
		got, err := exch.GetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, order.UnknownSide)
		require.NoError(t, err, "GetLeverage must return the effective contract limit")
		assert.Equal(t, 20.0, got, "leverage should be capped by the instrument")
		assert.Equal(t, []testRequest{{Method: "private/get-account-settings", Params: `{}`}, {Method: "public/get-instruments"}}, received(), "GetLeverage should read the account's leverage and the instrument's maximum")
	})
	t.Run("the account's leverage under the contract cap", func(t *testing.T) {
		t.Parallel()
		exch, _ := newRoutesServerExchange(t, map[string]testRoute{
			"private/get-account-settings": testResult(`[{"leverage":"5"}]`),
			"public/get-instruments":       testResult(`{"data":[{"symbol":"BTCUSD-PERP","inst_type":"PERPETUAL_SWAP","max_leverage":"20","tradable":true}]}`),
		})
		got, err := exch.GetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, order.UnknownSide)
		require.NoError(t, err, "GetLeverage must not error under the contract cap")
		assert.Equal(t, 5.0, got, "the account's lower leverage should apply")
	})
	for _, tc := range []struct {
		name        string
		instruments string
		err         error
	}{
		{"an instrument missing from the catalogue", `{"data":[{"symbol":"ETHUSD-PERP","max_leverage":"20"}]}`, errInstrumentNotFound},
		{"an instrument without a maximum leverage", `{"data":[{"symbol":"BTCUSD-PERP"}]}`, errNoMaximumLeverage},
		{"an instrument whose maximum leverage is not a number", `{"data":[{"symbol":"BTCUSD-PERP","max_leverage":"NaN"}]}`, errNoMaximumLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			exch, _ := newRoutesServerExchange(t, map[string]testRoute{
				"private/get-account-settings": testResult(`[{"leverage":"50"}]`),
				"public/get-instruments":       testResult(tc.instruments),
			})
			_, err := exch.GetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, order.UnknownSide)
			assert.ErrorIs(t, err, tc.err, "GetLeverage should refuse an instrument it cannot cap")
		})
	}
	t.Run("the instrument catalogue's failure", func(t *testing.T) {
		t.Parallel()
		exch, _ := newRoutesServerExchange(t, map[string]testRoute{
			"private/get-account-settings": testResult(`[{"leverage":"50"}]`),
			"public/get-instruments":       func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} },
		})
		_, err := exch.GetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, order.UnknownSide)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "GetLeverage must return the catalogue's failure")
		assert.Equal(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "GetLeverage should return the catalogue's error")
	})
	_, err = ex.GetLeverage(t.Context(), asset.PerpetualSwap, currency.EMPTYPAIR, margin.Multi, order.UnknownSide)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage should refuse an empty pair before any request")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Spot, testSpotPair, margin.Multi, 2, order.UnknownSide), futures.ErrNotFuturesAsset, "SetLeverage must refuse a spot pair")
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, 2.5, order.UnknownSide), errLeverageNotWhole, "SetLeverage must refuse an account leverage that is not a whole number")
	routes := maps.Clone(testPositionRoutes)
	routes["private/get-accounts"] = testResult(`{"this_account":{"uuid":"52e7c00f-1324-5a6a-bf6e-de445bde21a5"}}`)
	routes["private/change-account-leverage"] = testResult("")
	routes["private/change-isolated-margin-leverage"] = testResult("")
	ex := e
	var received func() []testRequest
	if mockTests {
		ex, received = newRoutesServerExchange(t, routes)
	}
	skipLiveUnlessCanManipulate(t)
	leverage := 10.0
	if !mockTests {
		// Setting the leverage the account has leaves its settings as they were
		settings, err := e.GetAccountSettings(t.Context())
		require.NoError(t, err, "GetAccountSettings must not error")
		require.NotEmpty(t, settings, "GetAccountSettings must return the account's settings")
		leverage = settings[0].Leverage.Float64()
	}
	require.NoError(t, ex.SetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, leverage, order.UnknownSide), "SetLeverage must not error")
	// The isolated case needs an isolated position, which the live tests do not open
	if !mockTests {
		return
	}
	require.NoError(t, ex.SetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Isolated, 3.5, order.UnknownSide), "SetLeverage must not error for an isolated position")
	exp := []testRequest{
		{Method: "private/get-accounts", Params: `{}`},
		{Method: "private/change-account-leverage", Params: `{"account_id":"52e7c00f-1324-5a6a-bf6e-de445bde21a5","leverage":10}`},
		{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`},
		{Method: "private/user-balance", Params: `{}`},
		{Method: "private/change-isolated-margin-leverage", Params: `{"isolation_id":"19848526","leverage":"3.5"}`},
	}
	assert.Equal(t, exp, received(), "SetLeverage should set the account's leverage for cross margin positions and an isolated position's own")
}

func TestChangePositionMargin(t *testing.T) { //nolint:tparallel // the cases share e and the test's exchanges in order
	t.Parallel()
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must refuse a nil request")
	// The refusals run in order on e, before ChangePositionMargin is called again below
	for _, tc := range []struct {
		name string
		req  *margin.PositionChangeRequest
		err  error
	}{
		{"a spot pair", &margin.PositionChangeRequest{Asset: asset.Spot}, futures.ErrNotFuturesAsset},
		{"cross margin", &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, MarginType: margin.Multi}, margin.ErrMarginTypeUnsupported},
		{"no new margin", &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, MarginType: margin.Isolated}, margin.ErrNewAllocatedMarginRequired},
		{"no original margin", &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, MarginType: margin.Isolated, NewAllocatedMargin: 1}, margin.ErrOriginalPositionMarginRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.ChangePositionMargin(t.Context(), tc.req)
			require.ErrorIsf(t, err, tc.err, "ChangePositionMargin must refuse %s", tc.name)
		})
	}
	// gctcli's float flags parse NaN and infinities, which are refused before any request; the cases share refuser in
	// order, whose requests are checked after them
	refuser, refused := newRoutesServerExchange(t, nil)
	for _, tc := range []struct {
		name                            string
		originalMargin, allocatedMargin float64
	}{
		{"a new margin of NaN", 5.4, math.NaN()},
		{"a new margin of +Inf", 5.4, math.Inf(1)},
		{"an original margin of NaN", math.NaN(), 8},
		{"an original margin of +Inf", math.Inf(1), 8},
		{"a new margin of -Inf", 5.4, math.Inf(-1)},
		{"an original margin of -Inf", math.Inf(-1), 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: tc.originalMargin, NewAllocatedMargin: tc.allocatedMargin})
			assert.ErrorIsf(t, err, errMarginNotFinite, "ChangePositionMargin should refuse %s", tc.name)
		})
	}
	assert.Empty(t, refused(), "ChangePositionMargin should send no request for a margin that is not a finite number")
	_, err = refuser.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, MarginType: margin.Isolated, OriginalAllocatedMargin: 5.4, NewAllocatedMargin: 8})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangePositionMargin should refuse a request without a pair")
	assert.Empty(t, refused(), "ChangePositionMargin should send no request without a pair")
	routes := maps.Clone(testPositionRoutes)
	routes["private/create-isolated-margin-transfer"] = testResult("")
	ex, received := newRoutesServerExchange(t, routes)
	// The margins share ex in order, whose transfers are checked after them
	for _, newMargin := range []float64{8, 3.4} {
		t.Run(strconv.FormatFloat(newMargin, 'f', -1, 64), func(t *testing.T) {
			got, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: 5.4, NewAllocatedMargin: newMargin})
			require.NoErrorf(t, err, "ChangePositionMargin must not error for a new margin of %v", newMargin)
			exp := &margin.PositionChangeResponse{Exchange: ex.Name, Pair: testPerpetualPair, Asset: asset.PerpetualSwap, AllocatedMargin: newMargin, MarginType: margin.Isolated}
			assert.Equalf(t, exp, got, "ChangePositionMargin should return the new margin of %v", newMargin)
		})
	}
	var transfers []string
	for _, r := range received() {
		if r.Method == "private/create-isolated-margin-transfer" {
			transfers = append(transfers, r.Params)
		}
	}
	assert.Equal(t, []string{
		`{"isolation_id":"19848526","direction":"CREDIT","amount":"2.6"}`,
		`{"isolation_id":"19848526","direction":"DEBIT","amount":"2"}`,
	}, transfers, "ChangePositionMargin should credit or debit the position the margin's difference")

	noIsolated := maps.Clone(routes)
	noIsolated["private/get-positions"] = testResult(`{"data":[{"quantity":"-0.1984","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"0"}]}`)
	ex, received = newRoutesServerExchange(t, noIsolated)
	got, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: 5.4, NewAllocatedMargin: 8})
	assert.ErrorIs(t, err, errNoIsolatedPosition, "ChangePositionMargin should refuse a pair without an isolated position")
	assert.Nil(t, got, "ChangePositionMargin should return no response for a pair without an isolated position")
	assert.Equal(t, []testRequest{{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`}}, received(), "ChangePositionMargin should transfer no margin without an isolated position")

	transferRefused := maps.Clone(routes)
	transferRefused["private/create-isolated-margin-transfer"] = func(string) (string, *APIError) {
		return "", &APIError{Code: 306, Message: "INSUFFICIENT_AVAILABLE_BALANCE"}
	}
	ex, _ = newRoutesServerExchange(t, transferRefused)
	got, err = ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: 5.4, NewAllocatedMargin: 8})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "ChangePositionMargin must return Crypto.com's refusal of the transfer")
	assert.Equal(t, &APIError{Code: 306, Message: "INSUFFICIENT_AVAILABLE_BALANCE"}, apiErr, "ChangePositionMargin should return Crypto.com's code and message")
	assert.Nil(t, got, "ChangePositionMargin should return no response for a transfer Crypto.com refuses")

	// Live, margin moves to the account's isolated position on the pair and back, leaving the position as it was
	skipLiveUnlessCanManipulate(t)
	ex = e
	if mockTests {
		ex, received = newRoutesServerExchange(t, routes)
	}
	got, err = ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: 5.4, NewAllocatedMargin: 8})
	if !mockTests && errors.Is(err, errNoIsolatedPosition) {
		t.Skip("ChangePositionMargin needs an isolated position, which the account does not hold on the pair and the live tests do not open")
	}
	require.NoError(t, err, "ChangePositionMargin must not error")
	restore := &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: 8, NewAllocatedMargin: 5.4}
	// The cleanup moves the margin back only when the test stops first or Crypto.com refuses the move, as a move whose
	// answer is lost may have happened
	unrestored := true
	t.Cleanup(func() {
		if unrestored {
			_, err := ex.ChangePositionMargin(context.WithoutCancel(t.Context()), restore)
			assert.NoError(t, err, "ChangePositionMargin should move the margin the test added back")
		}
	})
	assert.Equal(t, &margin.PositionChangeResponse{Exchange: ex.Name, Pair: testPerpetualPair, Asset: asset.PerpetualSwap, AllocatedMargin: 8, MarginType: margin.Isolated}, got, "ChangePositionMargin should return the new margin")
	got, err = ex.ChangePositionMargin(t.Context(), restore)
	var restoreRefused *APIError
	unrestored = errors.As(err, &restoreRefused)
	require.NoError(t, err, "ChangePositionMargin must not error moving the margin back")
	assert.Equal(t, &margin.PositionChangeResponse{Exchange: ex.Name, Pair: testPerpetualPair, Asset: asset.PerpetualSwap, AllocatedMargin: 5.4, MarginType: margin.Isolated}, got, "ChangePositionMargin should return the original margin")
	if !mockTests {
		return
	}
	positions := testRequest{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`}
	balance := testRequest{Method: "private/user-balance", Params: `{}`}
	exp := []testRequest{
		positions,
		balance,
		{Method: "private/create-isolated-margin-transfer", Params: `{"isolation_id":"19848526","direction":"CREDIT","amount":"2.6"}`},
		positions,
		balance,
		{Method: "private/create-isolated-margin-transfer", Params: `{"isolation_id":"19848526","direction":"DEBIT","amount":"2.6"}`},
	}
	assert.Equal(t, exp, received(), "ChangePositionMargin should look the position up before each transfer, crediting it the margin's difference and debiting it back")
}

// TestOrderAndAccountRequestFailures holds the order and account paths to the failure of each request they make. Each row
// answers the other requests and refuses the one it names, once, so a path retrying the refusal fails its row rather than
// hangs the test
func TestOrderAndAccountRequestFailures(t *testing.T) { //nolint:tparallel // the cases share one listing request
	t.Parallel()
	listing := &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}
	activeOrders := func(ex *Exchange) error {
		_, err := ex.GetActiveOrders(t.Context(), listing)
		return err
	}
	cancelAll := func(ex *Exchange) error {
		_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
		return err
	}
	orderHistory := func(ex *Exchange) error {
		_, err := ex.GetOrderHistory(t.Context(), listing)
		return err
	}
	positionOrders := func(ex *Exchange) error {
		_, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.PerpetualSwap, Pairs: currency.Pairs{testPerpetualPair}})
		return err
	}
	positionSummary := func(ex *Exchange) error {
		_, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair})
		return err
	}
	leverage := func(ex *Exchange) error {
		_, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, order.UnknownSide)
		return err
	}
	setLeverage := func(marginType margin.Type) func(*Exchange) error {
		return func(ex *Exchange) error {
			return ex.SetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, marginType, 2, order.UnknownSide)
		}
	}
	depositAddress := func(ex *Exchange) error {
		_, err := ex.GetDepositAddress(t.Context(), currency.BTC, "", "")
		return err
	}
	// The listings answered before a refusal hold an order, so a path returning what it has despite the refusal fails its
	// row; an hour ago lies within the window a history request defaults to
	listed := testResult(`{"data":[{"order_id":"1","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD","create_time":` + strconv.FormatInt(time.Now().Add(-time.Hour).UnixMilli(), 10) + `}]}`)
	open := map[string]testRoute{"private/get-open-orders": listed, "private/advanced/get-open-orders": listed}
	history := map[string]testRoute{"private/get-order-history": listed, "private/advanced/get-order-history": listed}
	// The listing cases share one request, which their calls could change, so the cases run one at a time
	for name, tc := range map[string]struct {
		routes  map[string]testRoute
		refused string
		call    func(*Exchange) error
	}{
		"GetActiveOrders' open orders":               {open, "private/get-open-orders", activeOrders},
		"GetActiveOrders' open trigger orders":       {open, "private/advanced/get-open-orders", activeOrders},
		"CancelAllOrders' open orders":               {open, "private/get-open-orders", cancelAll},
		"GetOrderHistory's orders":                   {history, "private/get-order-history", orderHistory},
		"GetOrderHistory's trigger orders":           {history, "private/advanced/get-order-history", orderHistory},
		"GetFuturesPositionOrders' orders":           {history, "private/get-order-history", positionOrders},
		"GetFuturesPositionSummary's positions":      {testPositionRoutes, "private/get-positions", positionSummary},
		"GetFuturesPositionSummary's balances":       {testPositionRoutes, "private/user-balance", positionSummary},
		"GetFuturesPositionSummary's mark price":     {testPositionRoutes, "public/get-valuations", positionSummary},
		"GetFuturesPositionSummary's settings":       {testPositionRoutes, "private/get-account-settings", positionSummary},
		"GetLeverage's settings":                     {testPositionRoutes, "private/get-account-settings", leverage},
		"SetLeverage's accounts":                     {testPositionRoutes, "private/get-accounts", setLeverage(margin.Multi)},
		"SetLeverage's isolated position's balances": {testPositionRoutes, "private/user-balance", setLeverage(margin.Isolated)},
		"GetDepositAddress' addresses":               {map[string]testRoute{}, "private/get-deposit-address", depositAddress},
	} {
		t.Run(name, func(t *testing.T) {
			routes := maps.Clone(tc.routes)
			routes[tc.refused] = testRefusedOnce(`{"data":[]}`)
			ex, _ := newRoutesServerExchange(t, routes)
			var apiErr *APIError
			require.ErrorAsf(t, tc.call(ex), &apiErr, "%s must return the request's failure", name)
			assert.Equalf(t, int64(50001), apiErr.Code, "%s should return Crypto.com's refusal", name)
		})
	}

	// An order neither the trading methods nor the advanced order methods know reports both refusals
	ex, _ := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail":          func(string) (string, *APIError) { return "", &APIError{Code: 212, Message: "INVALID_ORDERID"} },
		"private/advanced/get-order-detail": testRefusedOnce(`{}`),
	})
	_, err := ex.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	assert.ErrorContains(t, err, "INVALID_ORDERID", "GetOrderInfo should report the trading methods' refusal")
	assert.ErrorContains(t, err, "INTERNAL_SERVER_ERROR", "GetOrderInfo should report the advanced order methods' refusal")
}

// TestWrapperRefusals holds the wrapper to the sentinel errors of what it cannot use: an asset, a pair or a reply
func TestWrapperRefusals(t *testing.T) {
	t.Parallel()
	listing := func(a asset.Item, pairs ...currency.Pair) *order.MultiOrderRequest {
		return &order.MultiOrderRequest{AssetType: a, Pairs: pairs, Side: order.AnySide, Type: order.AnyType}
	}
	activeOrders := func(req *order.MultiOrderRequest) func(*Exchange) error {
		return func(ex *Exchange) error {
			_, err := ex.GetActiveOrders(t.Context(), req)
			return err
		}
	}
	orderHistory := func(req *order.MultiOrderRequest) func(*Exchange) error {
		return func(ex *Exchange) error {
			_, err := ex.GetOrderHistory(t.Context(), req)
			return err
		}
	}
	positionOrders := func(a asset.Item, p currency.Pair) func(*Exchange) error {
		return func(ex *Exchange) error {
			_, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: a, Pairs: currency.Pairs{p}})
			return err
		}
	}
	positionSummary := func(p currency.Pair) func(*Exchange) error {
		return func(ex *Exchange) error {
			_, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: p})
			return err
		}
	}
	leverage := func(p currency.Pair, marginType margin.Type) func(*Exchange) error {
		return func(ex *Exchange) error {
			_, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, p, marginType, order.UnknownSide)
			return err
		}
	}
	orderInfo := func(ex *Exchange) error {
		_, err := ex.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
		return err
	}
	// An hour ago lies within the window a history request defaults to
	created := strconv.FormatInt(time.Now().Add(-time.Hour).UnixMilli(), 10)
	orders := func(instrument, side string) testRoute {
		return testResult(`{"data":[{"order_id":"1","order_type":"LIMIT","side":"` + side + `","status":"ACTIVE","instrument_name":"` + instrument + `","create_time":` + created + `}]}`)
	}
	empty := testResult(`{"data":[]}`)
	withOrders := func(method string, route testRoute) map[string]testRoute {
		routes := map[string]testRoute{"private/get-open-orders": empty, "private/advanced/get-open-orders": empty, "private/get-order-history": empty, "private/advanced/get-order-history": empty}
		routes[method] = route
		return routes
	}
	noValuation := maps.Clone(testPositionRoutes)
	noValuation["public/get-valuations"] = empty
	noSettings := maps.Clone(testPositionRoutes)
	noSettings["private/get-account-settings"] = testResult(`[]`)
	for name, tc := range map[string]struct {
		routes map[string]testRoute
		call   func(*Exchange) error
		exp    error
	}{
		"GetActiveOrders an unsupported asset":            {nil, activeOrders(listing(asset.Options)), asset.ErrNotSupported},
		"GetOrderHistory an unsupported asset":            {nil, orderHistory(listing(asset.Options)), asset.ErrNotSupported},
		"GetActiveOrders an empty pair":                   {nil, activeOrders(listing(asset.Spot, currency.EMPTYPAIR)), currency.ErrCurrencyPairEmpty},
		"GetOrderHistory an empty pair":                   {nil, orderHistory(listing(asset.Spot, currency.EMPTYPAIR)), currency.ErrCurrencyPairEmpty},
		"GetActiveOrders a malformed instrument":          {withOrders("private/get-open-orders", orders("_USD", "BUY")), activeOrders(listing(asset.Spot)), errMalformedInstrumentName},
		"GetActiveOrders an unknown side":                 {withOrders("private/get-open-orders", orders("BTC_USD", "SIDEWAYS")), activeOrders(listing(asset.Spot)), order.ErrSideIsInvalid},
		"GetOrderHistory a malformed instrument":          {withOrders("private/get-order-history", orders("_USD", "BUY")), orderHistory(listing(asset.Spot)), errMalformedInstrumentName},
		"GetOrderHistory an unknown side":                 {withOrders("private/get-order-history", orders("BTC_USD", "SIDEWAYS")), orderHistory(listing(asset.Spot)), order.ErrSideIsInvalid},
		"GetFuturesPositionOrders a spot asset":           {nil, positionOrders(asset.Spot, testSpotPair), futures.ErrNotFuturesAsset},
		"GetFuturesPositionOrders an empty pair":          {nil, positionOrders(asset.PerpetualSwap, currency.EMPTYPAIR), currency.ErrCurrencyPairEmpty},
		"GetFuturesPositionOrders a malformed instrument": {withOrders("private/get-order-history", orders("-PERP", "BUY")), positionOrders(asset.PerpetualSwap, testPerpetualPair), errMalformedInstrumentName},
		"GetOrderInfo a malformed instrument":             {map[string]testRoute{"private/get-order-detail": testResult(`{"order_id":"1","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"_USD"}`)}, orderInfo, errMalformedInstrumentName},
		"GetFuturesPositionSummary an empty pair":         {nil, positionSummary(currency.EMPTYPAIR), currency.ErrCurrencyPairEmpty},
		"GetFuturesPositionSummary no mark price":         {noValuation, positionSummary(testPerpetualPair), common.ErrNoResponse},
		"GetLeverage an isolated position's empty pair":   {nil, leverage(currency.EMPTYPAIR, margin.Isolated), currency.ErrCurrencyPairEmpty},
		"GetLeverage no settings":                         {noSettings, leverage(testPerpetualPair, margin.Multi), common.ErrNoResponse},
		"SetLeverage an isolated position's empty pair": {nil, func(ex *Exchange) error {
			return ex.SetLeverage(t.Context(), asset.PerpetualSwap, currency.EMPTYPAIR, margin.Isolated, 2, order.UnknownSide)
		}, currency.ErrCurrencyPairEmpty},
		"UpdateTicker a reply without the instrument": {map[string]testRoute{"public/get-tickers": testResult(`{"data":[{"i":"ETH_USD","a":"1"}]}`)}, func(ex *Exchange) error {
			_, err := ex.UpdateTicker(t.Context(), testSpotPair, asset.Spot)
			return err
		}, ticker.ErrTickerNotFound},
		"GetDepositAddress a reply without an address": {map[string]testRoute{"private/get-deposit-address": testResult(`{"deposit_address_list":[]}`)}, func(ex *Exchange) error {
			_, err := ex.GetDepositAddress(t.Context(), currency.BTC, "", "")
			return err
		}, errNoDepositAddress},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex, _ := newRoutesServerExchange(t, tc.routes)
			assert.ErrorIsf(t, tc.call(ex), tc.exp, "%s should be refused", name)
		})
	}
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, "Cryptodotcom", ex.Name, "SetDefaults should name the exchange")
	assert.True(t, ex.Enabled, "SetDefaults should enable the exchange")
	assert.False(t, ex.Verbose, "SetDefaults should leave verbose output off")
	assert.Equal(t, config.APICredentialsValidatorConfig{RequiresKey: true, RequiresSecret: true}, ex.API.CredentialsValidator, "SetDefaults should require an API key and secret")
	underscore := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	dash := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	expPairs := currency.FullStore{
		asset.Spot:          {AssetEnabled: true, RequestFormat: underscore, ConfigFormat: underscore},
		asset.Margin:        {AssetEnabled: true, RequestFormat: underscore, ConfigFormat: underscore},
		asset.PerpetualSwap: {AssetEnabled: true, RequestFormat: dash, ConfigFormat: dash},
		asset.Futures:       {AssetEnabled: true, RequestFormat: dash, ConfigFormat: dash},
	}
	assert.Equal(t, expPairs, ex.CurrencyPairs.Pairs, "SetDefaults should enable each asset, naming spot and margin pairs with an underscore and derivatives with a dash")
	expFeatures := exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                    true,
				TickerFetching:                    true,
				KlineFetching:                     true,
				TradeFetching:                     true,
				OrderbookFetching:                 true,
				AutoPairUpdates:                   true,
				AccountInfo:                       true,
				CryptoDeposit:                     true,
				CryptoWithdrawal:                  true,
				GetOrder:                          true,
				GetOrders:                         true,
				CancelOrders:                      true,
				CancelOrder:                       true,
				SubmitOrder:                       true,
				ModifyOrder:                       true,
				DepositHistory:                    true,
				WithdrawalHistory:                 true,
				TradeFee:                          true,
				CryptoWithdrawalFee:               true,
				FundingRateFetching:               true,
				PredictedFundingRate:              true,
				MultiChainDeposits:                true,
				MultiChainWithdrawals:             true,
				MultiChainDepositRequiresChainSet: true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				GetOrders:              true,
				TickerFetching:         true,
				KlineFetching:          true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				FundingRateFetching:    true,
				PredictedFundingRate:   true,
				SubmitOrder:            true,
				SubmitOrders:           true,
				ModifyOrder:            true,
				CancelOrder:            true,
				DeadMansSwitch:         true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				MessageSequenceNumbers: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission | exchange.AutoWithdrawCryptoWithSetup,
			Kline:               kline.ExchangeCapabilitiesSupported{Intervals: true, DateRanges: true},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates:                    true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{kline.OneHour: true},
				Positions:                       true,
				CollateralMode:                  true,
				Leverage:                        true,
				OpenInterest:                    exchange.OpenInterestSupport{Supported: true, SupportedViaTicker: true, SupportsRestBatch: true},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				// No two-week interval, as two-week candles are built from 7D ones
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.SevenDay},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 300,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}
	assert.Equal(t, expFeatures, ex.Features, "SetDefaults should set the features Crypto.com supports and those enabled by default")
	require.Len(t, ex.Features.Subscriptions, len(defaultSubscriptions), "SetDefaults must subscribe the default channels")
	for i := range defaultSubscriptions {
		assert.NotSamef(t, defaultSubscriptions[i], ex.Features.Subscriptions[i], "SetDefaults should copy default subscription %d, so changing an exchange's subscriptions leaves the defaults", i)
	}
	expURLs := map[string]string{
		exchange.RestSpot.String():         apiURL,
		exchange.WebsocketSpot.String():    websocketMarketDataURL,
		exchange.WebsocketPrivate.String(): websocketUserURL,
		exchange.WebsocketTrade.String():   websocketUserURL,
	}
	assert.Equal(t, expURLs, ex.API.Endpoints.GetURLMap(), "SetDefaults should point REST and the websocket's connections at Crypto.com")
	require.NotNil(t, ex.Requester, "SetDefaults must set the requester up")
	assert.Equal(t, rateLimits, ex.Requester.GetRateLimiterDefinitions(), "SetDefaults should hold requests to Crypto.com's rate limits")
	require.NotNil(t, ex.Websocket, "SetDefaults must make the websocket manager")
	assert.False(t, ex.Websocket.IsInitialised(), "SetDefaults should leave the websocket for Setup to set up")
	assert.Equal(t, defaultWebsocketConnectDelay, ex.websocketConnectDelay, "SetDefaults should have a websocket connection wait before it sends anything")
	assert.Equal(t, defaultBookResubscribeDelay, ex.bookResubscribeDelay, "SetDefaults should set the delay before a book channel is subscribed again")
	require.NotNil(t, ex.marginOrders, "SetDefaults must make the margin order cache")
	assert.Zero(t, ex.marginOrders.Len(), "SetDefaults should track no margin orders")
	assert.Equal(t, exchange.DefaultWebsocketResponseMaxLimit, ex.WebsocketResponseMaxLimit, "SetDefaults should set the default websocket response limit")
	assert.Equal(t, exchange.DefaultWebsocketResponseCheckTimeout, ex.WebsocketResponseCheckTimeout, "SetDefaults should set the default websocket response check timeout")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	raw, err := testExchangeConfig()
	require.NoError(t, err, "testExchangeConfig must load Crypto.com's test configuration")
	// Each case decodes its own copy of the configuration, as Setup changes the configuration it is given
	setUp := func(t *testing.T) (*Exchange, *config.Exchange) {
		t.Helper()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.Name = t.Name()
		cfg := new(config.Exchange)
		require.NoError(t, json.Unmarshal(raw, cfg), "the test configuration must decode")
		return ex, cfg
	}
	t.Run("a nil configuration", func(t *testing.T) {
		t.Parallel()
		ex, _ := setUp(t)
		refusal := (*config.Exchange)(nil).Validate()
		require.Error(t, refusal, "Validate must refuse a nil configuration")
		assert.ErrorIs(t, ex.Setup(nil), refusal, "Setup should return the configuration's refusal")
		assert.False(t, ex.LoadedByConfig, "Setup should load nothing from a nil configuration")
	})
	t.Run("a disabled exchange", func(t *testing.T) {
		t.Parallel()
		ex, cfg := setUp(t)
		cfg.Enabled = false
		require.NoError(t, ex.Setup(cfg), "Setup must not error for a disabled exchange")
		assert.False(t, ex.IsEnabled(), "Setup should disable an exchange its configuration disables")
		assert.False(t, ex.LoadedByConfig, "Setup should not load a disabled exchange's configuration")
		assert.False(t, ex.Websocket.IsInitialised(), "Setup should not set a disabled exchange's websocket up")
	})
	t.Run("an enabled exchange", func(t *testing.T) {
		t.Parallel()
		ex, cfg := setUp(t)
		require.NoError(t, ex.Setup(cfg), "Setup must not error")
		assert.True(t, ex.IsEnabled(), "Setup should enable an exchange its configuration enables")
		assert.True(t, ex.LoadedByConfig, "Setup should load the exchange's configuration")
		assert.Same(t, cfg, ex.Config, "Setup should keep the configuration it loaded")
		assert.True(t, ex.Websocket.IsInitialised(), "Setup should set the websocket up")
		assert.True(t, ex.Websocket.IsEnabled(), "Setup should enable the websocket the configuration enables")
		urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
		require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
		assert.Equal(t, []string{websocketMarketDataURL, websocketUserURL}, urls, "Setup should set the market data connection up, and the user and trade connections, which share Crypto.com's user URL")
	})
	t.Run("the configuration's endpoints", func(t *testing.T) {
		t.Parallel()
		ex, cfg := setUp(t)
		cfg.API.Endpoints = map[string]string{
			exchange.RestSpot.String():         uatAPIURL,
			exchange.WebsocketSpot.String():    uatWebsocketMarketDataURL,
			exchange.WebsocketPrivate.String(): uatWebsocketUserURL,
			exchange.WebsocketTrade.String():   uatWebsocketUserURL,
		}
		require.NoError(t, ex.Setup(cfg), "Setup must not error for Crypto.com's UAT sandbox")
		restURL, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
		require.NoError(t, err, "GetURL must not error")
		assert.Equal(t, uatAPIURL, restURL, "Setup should send REST requests to the configured URL")
		urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
		require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
		assert.Equal(t, []string{uatWebsocketMarketDataURL, uatWebsocketUserURL}, urls, "Setup should connect the websocket to the configured URLs")
	})
	t.Run("a configuration it cannot apply", func(t *testing.T) {
		t.Parallel()
		ex, cfg := setUp(t)
		cfg.ProxyAddress = "://no-scheme"
		assert.ErrorIs(t, ex.Setup(cfg), exchange.ErrSettingProxyAddress, "Setup should return the error applying its configuration")
		assert.False(t, ex.Websocket.IsInitialised(), "Setup should not set the websocket up once applying its configuration fails")
	})
	t.Run("a websocket it cannot set up", func(t *testing.T) {
		t.Parallel()
		ex, cfg := setUp(t)
		ex.Websocket = nil
		assert.ErrorIs(t, ex.Setup(cfg), common.ErrNilPointer, "Setup should return the error setting the websocket up")
	})
}

func TestInstrumentIsAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		inst   InstrumentResponse
		assets asset.Items
	}{
		{name: "a pair margin trading is disabled for", inst: InstrumentResponse{InstrumentType: InstrumentTypeCurrencyPair}, assets: asset.Items{asset.Spot}},
		{name: "a pair enabled for margin buys", inst: InstrumentResponse{InstrumentType: InstrumentTypeCurrencyPair, MarginBuyEnabled: true}, assets: asset.Items{asset.Spot, asset.Margin}},
		{name: "a pair enabled for margin sells", inst: InstrumentResponse{InstrumentType: InstrumentTypeCurrencyPair, MarginSellEnabled: true}, assets: asset.Items{asset.Spot, asset.Margin}},
		{name: "a perpetual swap whose margin flags are set", inst: InstrumentResponse{InstrumentType: InstrumentTypePerpetualSwap, MarginBuyEnabled: true, MarginSellEnabled: true}, assets: asset.Items{asset.PerpetualSwap}},
		{name: "a future", inst: InstrumentResponse{InstrumentType: InstrumentTypeFuture}, assets: asset.Items{asset.Futures}},
		{name: "a deposit currency", inst: InstrumentResponse{InstrumentType: InstrumentTypeCurrency}},
		{name: "an instrument without a type", inst: InstrumentResponse{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures, asset.Options} {
				assert.Equalf(t, slices.Contains(tc.assets, a), instrumentIsAsset(&tc.inst, a), "instrumentIsAsset should tell whether %s trades as %s", tc.name, a)
			}
		})
	}
}

func TestInstrumentPair(t *testing.T) {
	t.Parallel()
	btcUSD := InstrumentResponse{Symbol: "BTC_USD", BaseCurrency: currency.BTC, QuoteCurrency: currency.USD}
	shibBTC := InstrumentResponse{Symbol: "SHIB_BTC@OEX_HK", BaseCurrency: currency.SHIB, QuoteCurrency: currency.BTC}
	for _, tc := range []struct {
		name string
		inst InstrumentResponse
		a    asset.Item
		exp  currency.Pair
		ok   bool
	}{
		{name: "a spot pair", inst: btcUSD, a: asset.Spot, exp: testSpotPair, ok: true},
		{name: "a margin pair", inst: btcUSD, a: asset.Margin, exp: testSpotPair, ok: true},
		{name: "a spot instrument whose name does not end in its quote", inst: shibBTC, a: asset.Spot, exp: currency.EMPTYPAIR},
		{name: "a margin instrument whose name does not end in its quote", inst: shibBTC, a: asset.Margin, exp: currency.EMPTYPAIR},
		{name: "a spot instrument whose name does not start with its base", inst: InstrumentResponse{Symbol: "WBTC_USD", BaseCurrency: currency.BTC, QuoteCurrency: currency.USD}, a: asset.Spot, exp: currency.EMPTYPAIR},
		{name: "a perpetual swap, named by its underlying", inst: InstrumentResponse{Symbol: "BTCUSD-PERP", BaseCurrency: currency.BTC, QuoteCurrency: currency.USD}, a: asset.PerpetualSwap, exp: testPerpetualPair, ok: true},
		{name: "a future, named by its underlying and expiry", inst: InstrumentResponse{Symbol: "BTCUSD-261030", BaseCurrency: currency.BTC, QuoteCurrency: currency.USD}, a: asset.Futures, exp: testFuturesPair, ok: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, ok, err := instrumentPair(&tc.inst, tc.a)
			require.NoErrorf(t, err, "instrumentPair must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, p, "instrumentPair should return the pair of %s", tc.name)
			assert.Equalf(t, tc.ok, ok, "instrumentPair should tell whether a pair names %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		inst InstrumentResponse
		a    asset.Item
		err  error
	}{
		{name: "a derivative named without a dash", inst: InstrumentResponse{Symbol: "BTCUSDPERP"}, a: asset.PerpetualSwap, err: errMalformedInstrumentName},
		{name: "a spot instrument named without an underscore", inst: InstrumentResponse{Symbol: "BTCUSD", BaseCurrency: currency.BTC, QuoteCurrency: currency.USD}, a: asset.Spot, err: errMalformedInstrumentName},
		{name: "an unsupported asset", inst: btcUSD, a: asset.Options, err: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, ok, err := instrumentPair(&tc.inst, tc.a)
			assert.ErrorIsf(t, err, tc.err, "instrumentPair should refuse %s", tc.name)
			assert.Equalf(t, currency.EMPTYPAIR, p, "instrumentPair should return no pair for %s", tc.name)
			assert.Falsef(t, ok, "instrumentPair should report no pair for %s", tc.name)
		})
	}
}

func TestInstrumentName(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	for _, tc := range []struct {
		name string
		a    asset.Item
		p    currency.Pair
		exp  string
	}{
		{name: "a spot pair", a: asset.Spot, p: testSpotPair, exp: "BTC_USD"},
		{name: "a margin pair", a: asset.Margin, p: testSpotPair, exp: "BTC_USD"},
		{name: "a perpetual swap", a: asset.PerpetualSwap, p: testPerpetualPair, exp: "BTCUSD-PERP"},
		{name: "a future", a: asset.Futures, p: testFuturesPair, exp: "BTCUSD-261030"},
		{name: "a spot pair without a delimiter", a: asset.Spot, p: currency.NewBTCUSD(), exp: "BTC_USD"},
		{name: "a perpetual swap in lower case with another delimiter", a: asset.PerpetualSwap, p: currency.NewPairWithDelimiter("btcusd", "perp", currency.ForwardSlashDelimiter), exp: "BTCUSD-PERP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.instrumentName(tc.a, tc.p)
			require.NoErrorf(t, err, "instrumentName must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "instrumentName should name %s as Crypto.com does", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		a    asset.Item
		p    currency.Pair
		err  error
	}{
		{name: "an empty pair", a: asset.Spot, p: currency.EMPTYPAIR, err: currency.ErrCurrencyPairEmpty},
		{name: "a pair of an unsupported asset", a: asset.Options, p: testSpotPair, err: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.instrumentName(tc.a, tc.p)
			assert.ErrorIsf(t, err, tc.err, "instrumentName should refuse %s", tc.name)
		})
	}
}

func TestOrderbookLevels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		levels []OrderbookLevel
		exp    orderbook.Levels
	}{
		{name: "no levels", exp: orderbook.Levels{}},
		{
			name:   "levels in the order given",
			levels: []OrderbookLevel{{Price: 80848.56, Quantity: 0.24979, OrderCount: 8}, {Price: 80847.76, Quantity: 0.15236, OrderCount: 2}},
			exp:    orderbook.Levels{{Price: 80848.56, Amount: 0.24979, OrderCount: 8}, {Price: 80847.76, Amount: 0.15236, OrderCount: 2}},
		},
		{name: "a level without orders", levels: []OrderbookLevel{{Price: 80848.56}}, exp: orderbook.Levels{{Price: 80848.56}}},
		{name: "an order count of the largest int64", levels: []OrderbookLevel{{Price: 1, Quantity: 2, OrderCount: math.MaxInt64}}, exp: orderbook.Levels{{Price: 1, Amount: 2, OrderCount: math.MaxInt64}}},
		{name: "an order count above the largest int64", levels: []OrderbookLevel{{Price: 1, Quantity: 2, OrderCount: math.MaxInt64 + 1}}, exp: orderbook.Levels{{Price: 1, Amount: 2, OrderCount: math.MaxInt64}}},
		{name: "the largest order count", levels: []OrderbookLevel{{Price: 1, Quantity: 2, OrderCount: math.MaxUint64}}, exp: orderbook.Levels{{Price: 1, Amount: 2, OrderCount: math.MaxInt64}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, orderbookLevels(tc.levels), "orderbookLevels should return each level, its order count capped at the largest int64")
		})
	}
}

func TestCountWalletSave(t *testing.T) {
	t.Parallel()
	account := accounts.Credentials{Key: testAPIKey, Secret: testAPISecret}
	subAccount := accounts.Credentials{Key: testAPIKey, Secret: testAPISecret, SubAccount: "sub-42"}
	for _, tc := range []struct {
		name   string
		counts map[accounts.Credentials]uint64
		creds  accounts.Credentials
		exp    map[accounts.Credentials]uint64
	}{
		{name: "the first save of any credentials' wallet", creds: account, exp: map[accounts.Credentials]uint64{account: 1}},
		{name: "another save of a wallet", counts: map[accounts.Credentials]uint64{account: 4}, creds: account, exp: map[accounts.Credentials]uint64{account: 5}},
		{name: "a save of another sub account's wallet", counts: map[accounts.Credentials]uint64{account: 4}, creds: subAccount, exp: map[accounts.Credentials]uint64{account: 4, subAccount: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := &Exchange{walletSaves: tc.counts}
			ex.walletMu.Lock()
			ex.countWalletSave(&tc.creds)
			ex.walletMu.Unlock()
			assert.Equal(t, tc.exp, ex.walletSaves, "countWalletSave should count each credentials' wallet saves apart")
		})
	}
}

// testCachedWallet returns the wallet the cached balance tests save under spot, as UpdateAccountBalances and the balance
// pushes save it
func testCachedWallet() *accounts.SubAccount {
	wallet := accounts.NewSubAccount(asset.Spot, "")
	wallet.Balances.Set(currency.BTC, accounts.Balance{Total: 2, Hold: 0.5, Free: 1.5, UpdatedAt: time.UnixMilli(1792894000000)})
	wallet.Balances.Set(currency.USD, accounts.Balance{Total: 3113.5, Hold: 12.5, Free: 3101, UpdatedAt: time.UnixMilli(1792894000000)})
	return wallet
}

// newWalletServerExchange returns a test server exchange, which must send no request, holding the sub accounts given
func newWalletServerExchange(t *testing.T, subAccts ...*accounts.SubAccount) *Exchange {
	t.Helper()
	ex, _ := newRoutesServerExchange(t, nil)
	if len(subAccts) != 0 {
		require.NoError(t, ex.Accounts.Save(t.Context(), subAccts, true), "Save must save the sub accounts")
	}
	return ex
}

func TestGetCachedSubAccounts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a     asset.Item
		label asset.Item
	}{
		{a: asset.Spot, label: asset.Spot},
		{a: asset.Margin, label: asset.Margin},
		{a: asset.PerpetualSwap, label: asset.PerpetualSwap},
		{a: asset.Futures, label: asset.Futures},
		{a: asset.All, label: asset.Spot},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWalletServerExchange(t, testCachedWallet())
			got, err := ex.GetCachedSubAccounts(t.Context(), tc.a)
			require.NoErrorf(t, err, "GetCachedSubAccounts must not error for %s", tc.a)
			assert.Equalf(t, accounts.SubAccounts{{AssetType: tc.label, Balances: testCachedWallet().Balances}}, got, "GetCachedSubAccounts should return the wallet saved under spot, labelled %s, for %s", tc.label, tc.a)
			stored, err := ex.Accounts.SubAccounts(nil, asset.All)
			require.NoError(t, err, "SubAccounts must not error")
			assert.Equalf(t, accounts.SubAccounts{{AssetType: asset.Spot, Balances: testCachedWallet().Balances}}, stored, "GetCachedSubAccounts should label a copy for %s, leaving the wallet saved under spot", tc.a)
		})
	}
	t.Run("every sub account saved under spot", func(t *testing.T) {
		t.Parallel()
		sub := accounts.NewSubAccount(asset.Spot, "sub-42")
		sub.Balances.Set(currency.ETH, accounts.Balance{Total: 3, Free: 3, UpdatedAt: time.UnixMilli(1792894000000)})
		ex := newWalletServerExchange(t, testCachedWallet(), sub)
		got, err := ex.GetCachedSubAccounts(t.Context(), asset.Futures)
		require.NoError(t, err, "GetCachedSubAccounts must not error")
		exp := accounts.SubAccounts{
			{AssetType: asset.Futures, Balances: testCachedWallet().Balances},
			{ID: "sub-42", AssetType: asset.Futures, Balances: sub.Balances},
		}
		assert.ElementsMatch(t, exp, got, "GetCachedSubAccounts should label every sub account saved under spot")
	})
	t.Run("a sub account saved under another asset", func(t *testing.T) {
		t.Parallel()
		marginWallet := accounts.NewSubAccount(asset.Margin, "")
		marginWallet.Balances.Set(currency.ETH, accounts.Balance{Total: 3, Free: 3, UpdatedAt: time.UnixMilli(1792894000000)})
		ex := newWalletServerExchange(t, testCachedWallet(), marginWallet)
		got, err := ex.GetCachedSubAccounts(t.Context(), asset.Margin)
		require.NoError(t, err, "GetCachedSubAccounts must not error")
		assert.Equal(t, accounts.SubAccounts{{AssetType: asset.Margin, Balances: testCachedWallet().Balances}}, got, "GetCachedSubAccounts should read margin's wallet from spot, not from a sub account saved under margin")
	})
	t.Run("an unsupported asset", func(t *testing.T) {
		t.Parallel()
		_, err := newWalletServerExchange(t, testCachedWallet()).GetCachedSubAccounts(t.Context(), asset.Options)
		assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCachedSubAccounts should refuse an unsupported asset")
	})
	t.Run("no wallet saved", func(t *testing.T) {
		t.Parallel()
		ex := newWalletServerExchange(t)
		for _, a := range []asset.Item{asset.Margin, asset.All} {
			_, err := ex.GetCachedSubAccounts(t.Context(), a)
			assert.ErrorIsf(t, err, accounts.ErrNoSubAccounts, "GetCachedSubAccounts should find no wallet for %s before one is saved", a)
		}
	})
	t.Run("credentials it cannot use", func(t *testing.T) {
		t.Parallel()
		ex := newWalletServerExchange(t, testCachedWallet())
		setTestCredentials(ex, &accounts.Credentials{})
		ex.SkipAuthCheck = false
		_, err := ex.GetCachedSubAccounts(t.Context(), asset.Spot)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetCachedSubAccounts should refuse empty credentials")
	})
	t.Run("the wallet UpdateAccountBalances saved", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex := newTestExchange(t)
		saved, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		require.Len(t, saved, 1, "UpdateAccountBalances must save the one wallet")
		for _, a := range append(ex.GetAssetTypes(false), asset.All) {
			got, err := ex.GetCachedSubAccounts(t.Context(), a)
			require.NoErrorf(t, err, "GetCachedSubAccounts must not error for %s", a)
			label := a
			if a == asset.All {
				label = asset.Spot
			}
			assert.Equalf(t, accounts.SubAccounts{{ID: saved[0].ID, AssetType: label, Balances: saved[0].Balances}}, got, "GetCachedSubAccounts should return the saved wallet for %s", a)
		}
	})
}

func TestGetCachedCurrencyBalances(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures, asset.All} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			got, err := newWalletServerExchange(t, testCachedWallet()).GetCachedCurrencyBalances(t.Context(), a)
			require.NoErrorf(t, err, "GetCachedCurrencyBalances must not error for %s", a)
			assert.Equalf(t, testCachedWallet().Balances, got, "GetCachedCurrencyBalances should return the balances of the wallet saved under spot for %s", a)
		})
	}
	t.Run("every sub account saved under spot", func(t *testing.T) {
		t.Parallel()
		sub := accounts.NewSubAccount(asset.Spot, "sub-42")
		sub.Balances.Set(currency.ETH, accounts.Balance{Total: 3, Free: 3, UpdatedAt: time.UnixMilli(1792894000000)})
		got, err := newWalletServerExchange(t, testCachedWallet(), sub).GetCachedCurrencyBalances(t.Context(), asset.PerpetualSwap)
		require.NoError(t, err, "GetCachedCurrencyBalances must not error")
		exp := testCachedWallet().Balances
		exp.Set(currency.ETH, accounts.Balance{Total: 3, Free: 3, UpdatedAt: time.UnixMilli(1792894000000)})
		assert.Equal(t, exp, got, "GetCachedCurrencyBalances should return the balances of every sub account saved under spot")
	})
	t.Run("balances saved under another asset", func(t *testing.T) {
		t.Parallel()
		marginWallet := accounts.NewSubAccount(asset.Margin, "")
		marginWallet.Balances.Set(currency.ETH, accounts.Balance{Total: 3, Free: 3, UpdatedAt: time.UnixMilli(1792894000000)})
		ex := newWalletServerExchange(t, testCachedWallet(), marginWallet)
		for _, a := range []asset.Item{asset.Margin, asset.All} {
			got, err := ex.GetCachedCurrencyBalances(t.Context(), a)
			require.NoErrorf(t, err, "GetCachedCurrencyBalances must not error for %s", a)
			assert.Equalf(t, testCachedWallet().Balances, got, "GetCachedCurrencyBalances should read %s's balances from spot only", a)
		}
	})
	t.Run("an unsupported asset", func(t *testing.T) {
		t.Parallel()
		_, err := newWalletServerExchange(t, testCachedWallet()).GetCachedCurrencyBalances(t.Context(), asset.Options)
		assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCachedCurrencyBalances should refuse an unsupported asset")
	})
	t.Run("no balances saved", func(t *testing.T) {
		t.Parallel()
		for name, ex := range map[string]*Exchange{
			"no wallet":       newWalletServerExchange(t),
			"an empty wallet": newWalletServerExchange(t, accounts.NewSubAccount(asset.Spot, "")),
		} {
			_, err := ex.GetCachedCurrencyBalances(t.Context(), asset.Margin)
			assert.ErrorIsf(t, err, accounts.ErrNoBalances, "GetCachedCurrencyBalances should find no balances in %s", name)
		}
	})
	t.Run("credentials it cannot use", func(t *testing.T) {
		t.Parallel()
		ex := newWalletServerExchange(t, testCachedWallet())
		setTestCredentials(ex, &accounts.Credentials{})
		ex.SkipAuthCheck = false
		_, err := ex.GetCachedCurrencyBalances(t.Context(), asset.Spot)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetCachedCurrencyBalances should refuse empty credentials")
	})
	t.Run("the wallet UpdateAccountBalances saved", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex := newTestExchange(t)
		saved, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		require.Len(t, saved, 1, "UpdateAccountBalances must save the one wallet")
		for _, a := range append(ex.GetAssetTypes(false), asset.All) {
			got, err := ex.GetCachedCurrencyBalances(t.Context(), a)
			if len(saved[0].Balances) == 0 {
				assert.ErrorIsf(t, err, accounts.ErrNoBalances, "GetCachedCurrencyBalances should find no balances in an empty wallet for %s", a)
				continue
			}
			require.NoErrorf(t, err, "GetCachedCurrencyBalances must not error for %s", a)
			assert.Equalf(t, saved[0].Balances, got, "GetCachedCurrencyBalances should return the saved wallet's balances for %s", a)
		}
	})
}

func TestWithdrawalCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		w    WithdrawalResponse
		exp  currency.Code
	}{
		{name: "a withdrawal naming its currency", w: WithdrawalResponse{Currency: currency.XRP}, exp: currency.XRP},
		{name: "a withdrawal naming its currency beside a symbol", w: WithdrawalResponse{Currency: currency.XRP, Symbol: currency.BTC}, exp: currency.XRP},
		{name: "a withdrawal naming its symbol, as create-withdrawal's example does", w: WithdrawalResponse{Symbol: currency.BTC}, exp: currency.BTC},
		{name: "a withdrawal naming neither", exp: currency.EMPTYCODE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, withdrawalCurrency(&tc.w), "withdrawalCurrency should return the currency of %s", tc.name)
		})
	}
}

func TestDepositStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, status, exp string
	}{
		{name: "not arrived", status: DepositStatusNotArrived, exp: "not arrived"},
		{name: "arrived", status: DepositStatusArrived, exp: "arrived"},
		{name: "failed", status: DepositStatusFailed, exp: "failed"},
		{name: "pending", status: DepositStatusPending, exp: "pending"},
		{name: "an unknown status", status: "9", exp: "9"},
		{name: "no status", exp: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, depositStatus(tc.status), "depositStatus should describe status %q", tc.status)
		})
	}
}

func TestWithdrawalStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, status, exp string
	}{
		{name: "pending", status: WithdrawalStatusPending, exp: "pending"},
		{name: "processing", status: WithdrawalStatusProcessing, exp: "processing"},
		{name: "rejected", status: WithdrawalStatusRejected, exp: "rejected"},
		{name: "payment in progress", status: WithdrawalStatusPaymentInProgress, exp: "payment in progress"},
		{name: "payment failed", status: WithdrawalStatusPaymentFailed, exp: "payment failed"},
		{name: "completed", status: WithdrawalStatusCompleted, exp: "completed"},
		{name: "cancelled", status: WithdrawalStatusCancelled, exp: "cancelled"},
		{name: "an unknown status", status: "9", exp: "9"},
		{name: "no status", exp: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, withdrawalStatus(tc.status), "withdrawalStatus should describe status %q", tc.status)
		})
	}
}

func TestTradeTime(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		record TradeResponse
		exp    time.Time
	}{
		{
			name:   "a trade timed to the nanosecond",
			record: TradeResponse{TradeTime: types.Time(time.UnixMilli(1792899071637)), TradeTimeNano: types.Time(time.Unix(0, 1792899071637623911))},
			exp:    time.Unix(0, 1792899071637623911),
		},
		{name: "a trade timed to the millisecond", record: TradeResponse{TradeTime: types.Time(time.UnixMilli(1792899071637))}, exp: time.UnixMilli(1792899071637)},
		{name: "a trade without a time", exp: time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, tradeTime(&tc.record), "tradeTime should time %s", tc.name)
		})
	}
}

func TestTradeData(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	for _, tc := range []struct {
		name   string
		trades []TradeResponse
		a      asset.Item
		p      currency.Pair
		exp    []trade.Data
	}{
		{name: "no trades", a: asset.Spot, p: testSpotPair, exp: []trade.Data{}},
		{
			name: "trades of either side, sent in either case",
			trades: []TradeResponse{
				{TradeID: "1792899071637623911", TradeTime: types.Time(time.UnixMilli(1792899071637)), TradeTimeNano: types.Time(time.Unix(0, 1792899071637623911)), Quantity: 0.00002, Price: 80849.37, Side: "sell", InstrumentName: "BTC_USD"},
				{TradeID: "1792899071550749165", TradeTime: types.Time(time.UnixMilli(1792899071550)), Quantity: 0.03092, Price: 80850.58, Side: "BUY", InstrumentName: "BTC_USD"},
			},
			a: asset.Spot,
			p: testSpotPair,
			exp: []trade.Data{
				{TID: "1792899071637623911", Exchange: e.Name, CurrencyPair: testSpotPair, AssetType: asset.Spot, Side: order.Sell, Price: 80849.37, Amount: 0.00002, Timestamp: time.Unix(0, 1792899071637623911)},
				{TID: "1792899071550749165", Exchange: e.Name, CurrencyPair: testSpotPair, AssetType: asset.Spot, Side: order.Buy, Price: 80850.58, Amount: 0.03092, Timestamp: time.UnixMilli(1792899071550)},
			},
		},
		{
			name:   "a trade of the asset and pair asked for",
			trades: []TradeResponse{{TradeID: "1", TradeTimeNano: types.Time(time.Unix(0, 1792899071637623912)), Quantity: 0.0001, Price: 80816.4, Side: "Buy", InstrumentName: "BTCUSD-PERP"}},
			a:      asset.PerpetualSwap,
			p:      testPerpetualPair,
			exp:    []trade.Data{{TID: "1", Exchange: e.Name, CurrencyPair: testPerpetualPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Price: 80816.4, Amount: 0.0001, Timestamp: time.Unix(0, 1792899071637623912)}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.tradeData(tc.trades, tc.a, tc.p)
			require.NoErrorf(t, err, "tradeData must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "tradeData should return %s as GoCryptoTrader holds them", tc.name)
		})
	}
	t.Run("a trade of an unknown side after a valid one", func(t *testing.T) {
		got, err := e.tradeData([]TradeResponse{{TradeID: "1", Side: "buy"}, {TradeID: "2", Side: "hold"}}, asset.Spot, testSpotPair)
		assert.ErrorIs(t, err, order.ErrSideIsInvalid, "tradeData should refuse a trade of an unknown side")
		assert.Nil(t, got, "tradeData should return none of the trades when it refuses one")
	})
}

func TestWalletCurrency(t *testing.T) {
	t.Parallel()
	networks := testResult(`{"update_time":1641151604000,"currency_map":{
		"USDC":{"full_name":"USD Coin","default_network":"ETH","network_list":[
			{"network_id":"ETH","withdrawal_fee":"1.5","withdraw_enabled":true,"min_withdrawal_amount":"3","deposit_enabled":true,"confirmation_required":"12"},
			{"network_id":"SOL","withdrawal_fee":"0.5","withdraw_enabled":false,"min_withdrawal_amount":"1","deposit_enabled":true,"confirmation_required":"1"}]},
		"BTC":{"full_name":"Bitcoin","default_network":null,"network_list":[{"network_id":"BTC","withdrawal_fee":"0.0004","withdraw_enabled":true,"min_withdrawal_amount":"0.001","deposit_enabled":true,"confirmation_required":"2"}]}}}`)
	usdc := &WalletCurrency{FullName: "USD Coin", DefaultNetwork: "ETH", Networks: []CurrencyNetwork{
		{NetworkID: "ETH", WithdrawalFee: preciseNumber("1.5"), WithdrawalEnabled: true, MinimumWithdrawalAmount: preciseNumber("3"), DepositEnabled: true, ConfirmationsRequired: 12},
		{NetworkID: "SOL", WithdrawalFee: preciseNumber("0.5"), MinimumWithdrawalAmount: preciseNumber("1"), DepositEnabled: true, ConfirmationsRequired: 1},
	}}
	for _, tc := range []struct {
		name string
		c    currency.Code
	}{
		{name: "a currency among several", c: currency.USDC},
		{name: "a currency named in lower case", c: currency.NewCode("usdc")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-currency-networks": networks})
			got, err := ex.walletCurrency(t.Context(), tc.c)
			require.NoErrorf(t, err, "walletCurrency must not error for %s", tc.name)
			assert.Equalf(t, usdc, got, "walletCurrency should return the networks of %s", tc.name)
			assert.Equalf(t, []testRequest{{Method: "private/get-currency-networks", Params: `{}`}}, received(), "walletCurrency should request every currency's networks for %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name     string
		networks testRoute
		c        currency.Code
	}{
		{name: "a currency without networks", networks: networks, c: currency.NewCode("NOPE")},
		{name: "an empty currency", networks: networks, c: currency.EMPTYCODE},
		{name: "a reply holding no currencies", networks: testResult(`{"currency_map":{}}`), c: currency.USDC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, _ := newRoutesServerExchange(t, map[string]testRoute{"private/get-currency-networks": tc.networks})
			got, err := ex.walletCurrency(t.Context(), tc.c)
			assert.ErrorIsf(t, err, currency.ErrCurrencyNotFound, "walletCurrency should refuse %s", tc.name)
			assert.Nilf(t, got, "walletCurrency should return no networks for %s", tc.name)
		})
	}
	t.Run("the networks request's failure", func(t *testing.T) {
		t.Parallel()
		ex, _ := newRoutesServerExchange(t, map[string]testRoute{"private/get-currency-networks": func(string) (string, *APIError) {
			return "", &APIError{Code: 40101, Message: "UNAUTHORIZED"}
		}})
		_, err := ex.walletCurrency(t.Context(), currency.USDC)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "walletCurrency must return Crypto.com's refusal")
		assert.Equal(t, &APIError{Code: 40101, Message: "UNAUTHORIZED"}, apiErr, "walletCurrency should return Crypto.com's code and message")
	})
	t.Run("the account's currency networks", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		c := currency.NewCode("MATIC")
		if !mockTests {
			c = currency.BTC
		}
		got, err := e.walletCurrency(t.Context(), c)
		require.NoErrorf(t, err, "walletCurrency must not error for %s", c)
		if mockTests {
			exp := &WalletCurrency{FullName: "Polygon", DefaultNetwork: "ETH", Networks: []CurrencyNetwork{
				{NetworkID: "BNB", WithdrawalFee: preciseNumber("0.80000000"), WithdrawalEnabled: true, MinimumWithdrawalAmount: preciseNumber("1.6"), DepositEnabled: true},
				{NetworkID: "ETH", WithdrawalFee: preciseNumber("20.00000000"), WithdrawalEnabled: true, MinimumWithdrawalAmount: preciseNumber("40.0"), DepositEnabled: true},
				{NetworkID: "MATIC", WithdrawalFee: preciseNumber("0.08000000"), MinimumWithdrawalAmount: preciseNumber("0.16"), DepositEnabled: true},
			}}
			assert.Equal(t, exp, got, "walletCurrency should return the recorded MATIC networks")
		} else {
			assert.NotEmptyf(t, got.Networks, "walletCurrency should return %s's networks", c)
		}
	})
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	// It sends nothing, so the call through e runs live without credentials, and can move no funds
	for _, tc := range []struct {
		name string
		req  *withdraw.Request
	}{
		{name: "no request"},
		{name: "a withdrawal to an international bank", req: &withdraw.Request{Exchange: e.Name, Currency: currency.EUR, Amount: 100, Type: withdraw.Fiat, Fiat: withdraw.FiatRequest{IsExpressWire: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), tc.req)
			assert.ErrorIsf(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported for %s", tc.name)
			assert.Nilf(t, got, "WithdrawFiatFundsToInternationalBank should return no withdrawal for %s", tc.name)
		})
	}
}

func TestSubmitResponse(t *testing.T) {
	t.Parallel()
	withoutClientOrderID := testSubmit()
	withoutClientOrderID.ClientOrderID = ""
	placed := func(clientOrderID string) order.SubmitResponse {
		return order.SubmitResponse{
			Exchange:      e.Name,
			Type:          order.Limit,
			Side:          order.Buy,
			Pair:          testPerpetualPair,
			AssetType:     asset.PerpetualSwap,
			TimeInForce:   order.GoodTillCancel | order.PostOnly,
			Price:         50000.5,
			Amount:        0.0001,
			ClientOrderID: clientOrderID,
			Status:        order.New,
			OrderID:       tradingTestOrderID,
		}
	}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		resp   OrderIDResponse
		exp    order.SubmitResponse
	}{
		{
			name:   "an order placed with a client order ID",
			submit: testSubmit(),
			resp:   OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID},
			exp:    placed(tradingTestOrderClientOrderID),
		},
		{
			name:   "an order keeping its client order ID over another its answer names",
			submit: testSubmit(),
			resp:   OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: "1792890506012"},
			exp:    placed(tradingTestOrderClientOrderID),
		},
		{
			name:   "an order placed without a client order ID, given Crypto.com's",
			submit: withoutClientOrderID,
			resp:   OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: "1792890506012"},
			exp:    placed("1792890506012"),
		},
		{
			name:   "an order placed without a client order ID, its answer naming none",
			submit: withoutClientOrderID,
			resp:   OrderIDResponse{OrderID: tradingTestOrderID},
			exp:    placed(""),
		},
		{
			name:   "a market order, new until Crypto.com matches it",
			submit: &order.Submit{Exchange: e.Name, Type: order.Market, Side: order.Sell, Pair: testSpotPair, AssetType: asset.Spot, Amount: 0.001, ClientOrderID: "market"},
			resp:   OrderIDResponse{OrderID: "1", ClientOrderID: "market"},
			exp:    order.SubmitResponse{Exchange: e.Name, Type: order.Market, Side: order.Sell, Pair: testSpotPair, AssetType: asset.Spot, Amount: 0.001, ClientOrderID: "market", Status: order.New, OrderID: "1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := submitResponse(tc.submit, &tc.resp)
			require.NoErrorf(t, err, "submitResponse must not error for %s", tc.name)
			clearSubmitTimes(t, got)
			assert.Equalf(t, &tc.exp, got, "submitResponse should return the submit response of %s", tc.name)
		})
	}
	t.Run("an answer without an order ID", func(t *testing.T) {
		t.Parallel()
		got, err := submitResponse(testSubmit(), &OrderIDResponse{ClientOrderID: tradingTestOrderClientOrderID})
		assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "submitResponse should refuse an answer without an order ID")
		assert.Nil(t, got, "submitResponse should return no response for an answer without an order ID")
	})
}

func TestTrackListSent(t *testing.T) {
	t.Parallel()
	t.Run("the list's spot and margin orders", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		orders := []*order.Submit{{AssetType: asset.Margin}, {AssetType: asset.Spot}, {AssetType: asset.Margin}, {AssetType: asset.PerpetualSwap}}
		requests := []*orderRequest{{clientOrderID: "margin"}, {clientOrderID: "spot"}, {clientOrderID: "unsent"}, {clientOrderID: "perpetual"}}
		ex.trackListSent(orders, requests, []int{0, 1, 3})
		assert.True(t, ex.isMarginOrder("", "margin"), "trackListSent should track the list's margin order by its client order ID")
		assert.False(t, ex.isMarginOrder("", "spot"), "trackListSent should not track the list's spot order")
		assert.False(t, ex.isMarginOrder("", "perpetual"), "trackListSent should not track the list's derivative order")
		assert.False(t, ex.isMarginOrder("", "unsent"), "trackListSent should not track a margin order outside the list")
		assert.Equal(t, uint64(1), ex.marginOrders.Len(), "trackListSent should track the list's margin order alone")
	})
	t.Run("a spot order reusing a margin order's client order ID", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.trackMarginOrder(asset.Margin, "", "shared")
		ex.trackListSent([]*order.Submit{{AssetType: asset.Spot}}, []*orderRequest{{clientOrderID: "shared"}}, []int{0})
		assert.False(t, ex.isMarginOrder("", "shared"), "trackListSent should stop tracking a client order ID a spot order of the list reuses, as its trades can arrive first")
	})
	t.Run("an empty list", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		ex.trackMarginOrder(asset.Margin, "", "tracked")
		ex.trackListSent([]*order.Submit{{AssetType: asset.Spot}}, []*orderRequest{{clientOrderID: "tracked"}}, nil)
		assert.True(t, ex.isMarginOrder("", "tracked"), "trackListSent should change nothing for an empty list")
		assert.Equal(t, uint64(1), ex.marginOrders.Len(), "trackListSent should track nothing for an empty list")
	})
}

func TestTrackListRefused(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		err     error
		tracked bool
	}{
		{name: "a list Crypto.com refused", err: &APIError{Code: 40001, Message: "BAD_REQUEST"}},
		{name: "a list Crypto.com refused, its refusal wrapped", err: fmt.Errorf("sending order list: %w", &APIError{Code: 40001, Message: "BAD_REQUEST"})},
		{name: "a list whose request was cancelled", err: context.Canceled, tracked: true},
		{name: "a list whose request failed without Crypto.com's answer", err: request.ErrBadStatus, tracked: true},
		{name: "no error", tracked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.SetDefaults()
			requests := []*orderRequest{{clientOrderID: "first"}, {clientOrderID: "second"}, {clientOrderID: "unsent"}}
			for _, r := range requests {
				ex.trackMarginOrder(asset.Margin, "", r.clientOrderID)
			}
			ex.trackListRefused(requests, []int{0, 1}, tc.err)
			for _, clientOrderID := range []string{"first", "second"} {
				assert.Equalf(t, tc.tracked, ex.isMarginOrder("", clientOrderID), "trackListRefused should leave %s tracked only when Crypto.com did not refuse the list, for %s", clientOrderID, tc.name)
			}
			assert.Truef(t, ex.isMarginOrder("", "unsent"), "trackListRefused should leave an order outside the list tracked, for %s", tc.name)
		})
	}
}

func TestTrackListAnswered(t *testing.T) {
	t.Parallel()
	rejected := fmt.Errorf("%w: %w", order.ErrUnableToPlaceOrder, &APIError{Code: 318, Message: "EXCEEDS_MAX_ALLOWED_ORDERS"})
	for _, tc := range []struct {
		name           string
		a              asset.Item
		response       order.SubmitResponse
		clientTracked  bool
		orderIsMargin  bool
		trackedOrderID string
	}{
		{name: "a margin order placed", a: asset.Margin, response: order.SubmitResponse{Status: order.New, OrderID: "11"}, orderIsMargin: true, trackedOrderID: "11"},
		{name: "a spot order placed", a: asset.Spot, response: order.SubmitResponse{Status: order.New, OrderID: "13"}, trackedOrderID: "13"},
		{name: "a margin order Crypto.com rejected", a: asset.Margin, response: order.SubmitResponse{Status: order.Rejected, SubmissionError: rejected}},
		{name: "a margin order whose outcome is unknown", a: asset.Margin, response: order.SubmitResponse{Status: order.UnknownStatus, SubmissionError: errOrderListOutcomeUnknown}, clientTracked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.SetDefaults()
			// The client order IDs are tracked as margin orders' as when the list was sent, a spot order's as when another
			// margin order shares it
			ex.trackMarginOrder(asset.Margin, "", "answered")
			ex.trackMarginOrder(asset.Margin, "", "unsent")
			orders := []*order.Submit{{AssetType: tc.a}, {AssetType: asset.Margin}}
			requests := []*orderRequest{{clientOrderID: "answered"}, {clientOrderID: "unsent"}}
			ex.trackListAnswered(orders, requests, []int{0}, []*order.SubmitResponse{&tc.response, nil})
			assert.Equalf(t, tc.clientTracked, ex.isMarginOrder("", "answered"), "trackListAnswered should track the client order ID of %s only while its outcome is unknown", tc.name)
			if tc.trackedOrderID != "" {
				assert.Equalf(t, tc.orderIsMargin, ex.isMarginOrder(tc.trackedOrderID, "answered"), "trackListAnswered should track %s by its order ID", tc.name)
			}
			assert.Truef(t, ex.isMarginOrder("", "unsent"), "trackListAnswered should leave an order outside the list tracked, for %s", tc.name)
		})
	}
}

func TestModifyResponse(t *testing.T) {
	t.Parallel()
	cro := currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter)
	limit := order.Modify{Exchange: e.Name, OrderID: "1", ClientOrderID: "c", Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000}
	for _, tc := range []struct {
		name string
		m    order.Modify
		a    orderAmendment
		resp OrderIDResponse
		exp  *order.ModifyResponse
	}{
		{
			name: "an amendment restating the order's quantity and time in force",
			m:    limit,
			a:    orderAmendment{price: 77000, amount: 0.01, timeInForce: order.GoodTillCancel | order.PostOnly},
			resp: OrderIDResponse{OrderID: "1", ClientOrderID: "c"},
			exp:  &order.ModifyResponse{Exchange: e.Name, OrderID: "1", ClientOrderID: "c", Pair: testSpotPair, Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 77000, Amount: 0.01},
		},
		{
			name: "a modification naming no type, given the order's, whose amended price is no limit price",
			m:    order.Modify{Exchange: e.Name, OrderID: "1", Side: order.Sell, AssetType: asset.Spot, Pair: cro, Price: 0.0279, Amount: 10, TriggerPrice: 0.028},
			a:    orderAmendment{amount: 10, orderType: order.Stop},
			resp: OrderIDResponse{OrderID: "1"},
			exp:  &order.ModifyResponse{Exchange: e.Name, OrderID: "1", Pair: cro, Type: order.Stop, Side: order.Sell, AssetType: asset.Spot, Amount: 10, TriggerPrice: 0.028},
		},
		{
			name: "an order given a new client order ID",
			m:    order.Modify{Exchange: e.Name, OrderID: "1", ClientOrderID: "original", NewClientOrderID: "renamed", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.01},
			a:    orderAmendment{price: 77000, amount: 0.01},
			resp: OrderIDResponse{OrderID: "1", ClientOrderID: "renamed"},
			exp:  &order.ModifyResponse{Exchange: e.Name, OrderID: "1", ClientOrderID: "renamed", Pair: testSpotPair, Type: order.Limit, AssetType: asset.Spot, Price: 77000, Amount: 0.01},
		},
		{
			name: "an order named by its client order ID, the answer giving its order ID",
			m:    order.Modify{Exchange: e.Name, ClientOrderID: "c", Type: order.Limit, AssetType: asset.Spot, Pair: testSpotPair, Price: 77000, Amount: 0.01},
			a:    orderAmendment{price: 77000, amount: 0.01},
			resp: OrderIDResponse{OrderID: "1", ClientOrderID: "c"},
			exp:  &order.ModifyResponse{Exchange: e.Name, OrderID: "1", ClientOrderID: "c", Pair: testSpotPair, Type: order.Limit, AssetType: asset.Spot, Price: 77000, Amount: 0.01},
		},
		{
			name: "an answer naming neither ID",
			m:    limit,
			a:    orderAmendment{price: 77000, amount: 0.01},
			exp:  &order.ModifyResponse{Exchange: e.Name, OrderID: "1", ClientOrderID: "c", Pair: testSpotPair, Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Price: 77000, Amount: 0.01},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := modifyResponse(&tc.m, &tc.a, &tc.resp)
			require.NoErrorf(t, err, "modifyResponse must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "modifyResponse should return the modify response of %s", tc.name)
		})
	}
	t.Run("no modification", func(t *testing.T) {
		t.Parallel()
		_, refusal := (*order.Modify)(nil).DeriveModifyResponse()
		require.Error(t, refusal, "DeriveModifyResponse must refuse no modification")
		got, err := modifyResponse(nil, &orderAmendment{}, &OrderIDResponse{OrderID: "1"})
		assert.ErrorIs(t, err, refusal, "modifyResponse should return DeriveModifyResponse's refusal")
		assert.Nil(t, got, "modifyResponse should return no response without a modification")
	})
}

// acceptCancelList answers a cancel list request accepting each of its orders
func acceptCancelList(params string) (string, *APIError) {
	var list struct {
		Orders []json.RawMessage `json:"order_list"`
	}
	if err := json.Unmarshal([]byte(params), &list); err != nil {
		return "", &APIError{Code: 40001, Message: "BAD_REQUEST"}
	}
	entries := make([]string, len(list.Orders))
	for i := range entries {
		entries[i] = `{"code":0,"index":` + strconv.Itoa(i) + `}`
	}
	return "[" + strings.Join(entries, ",") + "]", nil
}

// assertJSONRequests checks a test server received the POST requests expected, in order, comparing their params as JSON
func assertJSONRequests(t *testing.T, exp, got []testRequest) {
	t.Helper()
	require.Len(t, got, len(exp), "the test server must receive the requests expected")
	for i := range exp {
		assert.Equalf(t, exp[i].Method, got[i].Method, "request %d should be for its method", i)
		assert.JSONEqf(t, exp[i].Params, got[i].Params, "request %d should send its params", i)
	}
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	t.Run("lists of up to 10, the trading methods' before the advanced order methods'", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/cancel-order-list": acceptCancelList, "private/advanced/cancel-order-list": acceptCancelList})
		// 11 orders of each kind fill a list of 10 and start another, and the advanced order sent first is still cancelled
		// after the trading methods' lists
		cancels := make([]orderCancel, 1, 22)
		cancels[0] = orderCancel{OrderID: "101", Advanced: true}
		exp := map[string]string{"101": "PENDING_CANCEL", "c11": "PENDING_CANCEL", "c111": "PENDING_CANCEL"}
		regularIDs := make([]string, 0, 10)
		advancedIDs := make([]string, 1, 10)
		advancedIDs[0] = `{"order_id":"101"}`
		for i := range 10 {
			id := strconv.Itoa(i + 1)
			cancels = append(cancels, orderCancel{OrderID: id})
			exp[id] = "PENDING_CANCEL"
			regularIDs = append(regularIDs, `{"order_id":"`+id+`"}`)
		}
		cancels = append(cancels, orderCancel{ClientOrderID: "c11"})
		for i := range 9 {
			id := strconv.Itoa(102 + i)
			cancels = append(cancels, orderCancel{OrderID: id, Advanced: true})
			exp[id] = "PENDING_CANCEL"
			advancedIDs = append(advancedIDs, `{"order_id":"`+id+`"}`)
		}
		cancels = append(cancels, orderCancel{ClientOrderID: "c111", Advanced: true})
		got, err := ex.cancelOrders(t.Context(), cancels)
		require.NoError(t, err, "cancelOrders must not error")
		assert.Equal(t, exp, got, "cancelOrders should report every order pending cancellation, keyed by its order ID or else its client order ID")
		assertJSONRequests(t, []testRequest{
			{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[` + strings.Join(regularIDs, ",") + `]}`},
			{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[{"client_oid":"c11"}]}`},
			{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[` + strings.Join(advancedIDs, ",") + `]}`},
			{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[{"client_oid":"c111"}]}`},
		}, received())
	})
	for _, tc := range []struct {
		name     string
		cancels  []orderCancel
		regular  testRoute
		advanced testRoute
		exp      map[string]string
		requests []testRequest
	}{
		{
			name:     "an order the trading methods refuse, which the advanced order methods cancel",
			cancels:  []orderCancel{{OrderID: "1"}, {OrderID: "2"}, {OrderID: "3", Advanced: true}},
			regular:  testResult(`[{"code":0,"index":0},{"code":316,"index":1,"message":"NO_ACTIVE_ORDER"}]`),
			advanced: acceptCancelList,
			exp:      map[string]string{"1": "PENDING_CANCEL", "2": "PENDING_CANCEL", "3": "PENDING_CANCEL"},
			requests: []testRequest{
				{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[{"order_id":"1"},{"order_id":"2"}]}`},
				{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[{"order_id":"3"},{"order_id":"2"}]}`},
			},
		},
		{
			name:     "an order both methods refuse",
			cancels:  []orderCancel{{ClientOrderID: "c"}},
			regular:  testResult(`[{"code":316,"index":0,"message":"NO_ACTIVE_ORDER"}]`),
			advanced: testResult(`[{"code":40401,"index":0,"message":"NOT_FOUND"}]`),
			exp:      map[string]string{"c": "316 NO_ACTIVE_ORDER; 40401 NOT_FOUND"},
			requests: []testRequest{
				{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[{"client_oid":"c"}]}`},
				{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[{"client_oid":"c"}]}`},
			},
		},
		{
			name:     "an order the trading methods refuse and the advanced order methods' answer leaves out",
			cancels:  []orderCancel{{OrderID: "1"}},
			regular:  testResult(`[{"code":316,"index":0,"message":"NO_ACTIVE_ORDER"}]`),
			advanced: testResult(`[]`),
			exp:      map[string]string{"1": "316 NO_ACTIVE_ORDER; no result for the order"},
			requests: []testRequest{
				{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[{"order_id":"1"}]}`},
				{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[{"order_id":"1"}]}`},
			},
		},
		{
			name:     "an order refused for the request, which the advanced order methods would refuse too",
			cancels:  []orderCancel{{OrderID: "1"}},
			regular:  testResult(`[{"code":42901,"index":0,"message":"TOO_MANY_REQUESTS"}]`),
			exp:      map[string]string{"1": "42901 TOO_MANY_REQUESTS"},
			requests: []testRequest{{Method: "private/cancel-order-list", Params: `{"contingency_type":"LIST","order_list":[{"order_id":"1"}]}`}},
		},
		{
			name:     "a trigger order the advanced order methods refuse",
			cancels:  []orderCancel{{OrderID: "3", Advanced: true}},
			advanced: testResult(`[{"code":40401,"index":0,"message":"NOT_FOUND"}]`),
			exp:      map[string]string{"3": "40401 NOT_FOUND"},
			requests: []testRequest{{Method: "private/advanced/cancel-order-list", Params: `{"order_list":[{"order_id":"3"}]}`}},
		},
		{name: "no orders", exp: map[string]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			routes := make(map[string]testRoute)
			if tc.regular != nil {
				routes["private/cancel-order-list"] = tc.regular
			}
			if tc.advanced != nil {
				routes["private/advanced/cancel-order-list"] = tc.advanced
			}
			ex, received := newRoutesServerExchange(t, routes)
			got, err := ex.cancelOrders(t.Context(), tc.cancels)
			require.NoErrorf(t, err, "cancelOrders must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "cancelOrders should report the outcome of %s", tc.name)
			assertJSONRequests(t, tc.requests, received())
		})
	}
	t.Run("a list whose request fails, stopping the lists after it", func(t *testing.T) {
		t.Parallel()
		var lists atomic.Int64
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/cancel-order-list": func(params string) (string, *APIError) {
			if lists.Add(1) == 1 {
				return acceptCancelList(params)
			}
			return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}
		}})
		cancels := make([]orderCancel, 1, 12)
		cancels[0] = orderCancel{OrderID: "101", Advanced: true}
		exp := make(map[string]string, 10)
		for i := range 11 {
			id := strconv.Itoa(i + 1)
			cancels = append(cancels, orderCancel{OrderID: id})
			if i < 10 {
				exp[id] = "PENDING_CANCEL"
			}
		}
		got, err := ex.cancelOrders(t.Context(), cancels)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "cancelOrders must return the error of the list whose request failed")
		assert.Equal(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "cancelOrders should return Crypto.com's refusal")
		assert.Equal(t, exp, got, "cancelOrders should return the outcomes of the list sent before the one that failed")
		requests := received()
		require.Len(t, requests, 2, "cancelOrders must send no list after the one that failed")
		assert.Equal(t, "private/cancel-order-list", requests[1].Method, "cancelOrders should send the trading methods' second list last")
	})
	t.Run("an advanced order list whose request fails", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{
			"private/cancel-order-list": acceptCancelList,
			"private/advanced/cancel-order-list": func(string) (string, *APIError) {
				return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}
			},
		})
		got, err := ex.cancelOrders(t.Context(), []orderCancel{{OrderID: "1"}, {OrderID: "2", Advanced: true}})
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "cancelOrders must return the error of the advanced order list whose request failed")
		assert.Equal(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "cancelOrders should return Crypto.com's refusal")
		assert.Equal(t, map[string]string{"1": "PENDING_CANCEL"}, got, "cancelOrders should return the outcomes of the lists sent before the one that failed")
		assert.Len(t, received(), 2, "cancelOrders should send both lists")
	})
	for _, tc := range []struct {
		name    string
		cancels []orderCancel
	}{
		{name: "an order named by neither ID", cancels: []orderCancel{{OrderID: "1"}, {}}},
		{name: "a trigger order named by neither ID", cancels: []orderCancel{{Advanced: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, nil)
			got, err := ex.cancelOrders(t.Context(), tc.cancels)
			assert.ErrorIsf(t, err, errOrderIDRequired, "cancelOrders should refuse %s", tc.name)
			assert.Emptyf(t, got, "cancelOrders should report no outcome for %s", tc.name)
			assert.Emptyf(t, received(), "cancelOrders should send no list for %s", tc.name)
		})
	}
	t.Run("orders the account holds", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		// The recording cancels two trigger orders; live, the test places an order of its own to cancel
		cancels := []orderCancel{{OrderID: "289991951579667814", Advanced: true}, {ClientOrderID: "my_tp_order", Advanced: true}}
		exp := map[string]string{"289991951579667814": "PENDING_CANCEL", "my_tp_order": "PENDING_CANCEL"}
		if !mockTests {
			orderID := placeTradingTestOrder(t)
			cancels, exp = []orderCancel{{OrderID: orderID}}, map[string]string{orderID: "PENDING_CANCEL"}
		}
		got, err := e.cancelOrders(t.Context(), cancels)
		require.NoError(t, err, "cancelOrders must not error")
		assert.Equal(t, exp, got, "cancelOrders should report each order pending cancellation")
	})
}

func TestOrderCancelCancelID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		c    orderCancel
		exp  string
	}{
		{name: "an order ID", c: orderCancel{OrderID: "1"}, exp: "1"},
		{name: "an order ID beside a client order ID", c: orderCancel{OrderID: "1", ClientOrderID: "c"}, exp: "1"},
		{name: "a client order ID", c: orderCancel{ClientOrderID: "c"}, exp: "c"},
		{name: "a trigger order's client order ID", c: orderCancel{ClientOrderID: "c", Advanced: true}, exp: "c"},
		{name: "neither ID", c: orderCancel{}, exp: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, tc.c.cancelID(), "cancelID should key an order named by %s", tc.name)
		})
	}
}

func TestOrderByID(t *testing.T) {
	t.Parallel()
	held := testResult(`{"order_id":"1","client_oid":"c","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTC_USD"}`)
	exp := &OrderResponse{OrderID: "1", ClientOrderID: "c", OrderType: OrderTypeLimit, Side: SideBuy, Status: OrderStatusActive, InstrumentName: "BTC_USD"}
	refused := func(code int64, message string) testRoute {
		return func(string) (string, *APIError) { return "", &APIError{Code: code, Message: message} }
	}
	for _, tc := range []struct {
		name                   string
		orderID, clientOrderID string
		routes                 map[string]testRoute
		advanced               bool
		requests               []testRequest
	}{
		{
			name:     "an order the trading methods hold",
			orderID:  "1",
			routes:   map[string]testRoute{"private/get-order-detail": held},
			requests: []testRequest{{Method: "private/get-order-detail", Params: `{"order_id":"1"}`}},
		},
		{
			name:          "an order named by its client order ID",
			clientOrderID: "c",
			routes:        map[string]testRoute{"private/get-order-detail": held},
			requests:      []testRequest{{Method: "private/get-order-detail", Params: `{"client_oid":"c"}`}},
		},
		{
			name:     "an order the advanced order methods hold",
			orderID:  "1",
			routes:   map[string]testRoute{"private/get-order-detail": refused(40401, "NOT_FOUND"), "private/advanced/get-order-detail": held},
			advanced: true,
			requests: []testRequest{
				{Method: "private/get-order-detail", Params: `{"order_id":"1"}`},
				{Method: "private/advanced/get-order-detail", Params: `{"order_id":"1"}`},
			},
		},
		{
			name:          "an order the advanced order methods hold, named by both IDs",
			orderID:       "1",
			clientOrderID: "c",
			routes:        map[string]testRoute{"private/get-order-detail": refused(212, "INVALID_ORDERID"), "private/advanced/get-order-detail": held},
			advanced:      true,
			requests: []testRequest{
				{Method: "private/get-order-detail", Params: `{"order_id":"1","client_oid":"c"}`},
				{Method: "private/advanced/get-order-detail", Params: `{"order_id":"1","client_oid":"c"}`},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, tc.routes)
			got, advanced, err := ex.orderByID(t.Context(), tc.orderID, tc.clientOrderID)
			require.NoErrorf(t, err, "orderByID must not error for %s", tc.name)
			assert.Equalf(t, exp, got, "orderByID should return %s", tc.name)
			assert.Equalf(t, tc.advanced, advanced, "orderByID should tell whether the advanced order methods hold %s", tc.name)
			assertJSONRequests(t, tc.requests, received())
		})
	}
	t.Run("an order neither method holds", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-order-detail": refused(40401, "NOT_FOUND"), "private/advanced/get-order-detail": refused(316, "NO_ACTIVE_ORDER")})
		got, advanced, err := ex.orderByID(t.Context(), "1", "")
		assert.ErrorContains(t, err, "40401 NOT_FOUND", "orderByID should report the trading methods' refusal")
		assert.ErrorContains(t, err, "316 NO_ACTIVE_ORDER", "orderByID should report the advanced order methods' refusal")
		assert.Nil(t, got, "orderByID should return no order")
		assert.False(t, advanced, "orderByID should report no advanced order")
		assert.Len(t, received(), 2, "orderByID should look the order up with both methods")
	})
	t.Run("a lookup refused for the request", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-order-detail": refused(codeTooManyRequests, "TOO_MANY_REQUESTS")})
		got, _, err := ex.orderByID(t.Context(), "1", "")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "orderByID must return Crypto.com's refusal")
		assert.Equal(t, &APIError{Code: codeTooManyRequests, Message: "TOO_MANY_REQUESTS"}, apiErr, "orderByID should return the refusal of the request")
		assert.Nil(t, got, "orderByID should return no order")
		assert.Len(t, received(), 1, "orderByID should not look up with the advanced order methods an order whose request was refused, which they would refuse too")
	})
	t.Run("a lookup that fails", func(t *testing.T) {
		t.Parallel()
		got, _, err := newUnavailableServerExchange(t).orderByID(t.Context(), "1", "")
		assert.ErrorIs(t, err, request.ErrBadStatus, "orderByID should return the request's failure")
		assert.Nil(t, got, "orderByID should return no order")
	})
	t.Run("an order named by neither ID", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, nil)
		_, _, err := ex.orderByID(t.Context(), "", "")
		assert.ErrorIs(t, err, errOrderIDRequired, "orderByID should refuse an order named by neither ID")
		assert.Empty(t, received(), "orderByID should send no lookup for an order named by neither ID")
	})
	t.Run("an order the account placed", func(t *testing.T) {
		t.Parallel()
		skipLiveUnlessCanManipulate(t)
		orderID := "6530219466236720403"
		if !mockTests {
			orderID = placeTradingTestOrder(t)
			cancelTradingOrderAfterTest(t, orderID)
		}
		got, advanced, err := e.orderByID(t.Context(), orderID, "")
		require.NoError(t, err, "orderByID must not error")
		assert.Equal(t, orderID, got.OrderID, "orderByID should return the order")
		assert.Equal(t, "BTCUSD-PERP", got.InstrumentName, "orderByID should return the order's instrument")
		assert.False(t, advanced, "orderByID should find the order with the trading methods")
	})
	t.Run("an order Crypto.com does not know", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex := e
		if mockTests {
			ex, _ = newRoutesServerExchange(t, map[string]testRoute{"private/get-order-detail": refused(40401, "NOT_FOUND"), "private/advanced/get-order-detail": refused(40401, "NOT_FOUND")})
		}
		_, _, err := ex.orderByID(t.Context(), "1", "")
		var apiErr *APIError
		assert.ErrorAs(t, err, &apiErr, "orderByID should return Crypto.com's refusal of an order it does not know")
	})
}

func TestRequestInstrument(t *testing.T) { //nolint:tparallel // the cases run in order on e
	t.Parallel()
	for _, tc := range []struct {
		name  string
		a     asset.Item
		pairs currency.Pairs
		exp   string
	}{
		{name: "no pairs, every instrument", a: asset.Spot},
		{name: "a spot pair", a: asset.Spot, pairs: currency.Pairs{testSpotPair}, exp: "BTC_USD"},
		{name: "a perpetual swap", a: asset.PerpetualSwap, pairs: currency.Pairs{testPerpetualPair}, exp: "BTCUSD-PERP"},
		{name: "several pairs, every instrument", a: asset.Spot, pairs: currency.Pairs{testSpotPair, spotETHUSD}},
		{name: "several pairs, one of them empty", a: asset.Spot, pairs: currency.Pairs{testSpotPair, currency.EMPTYPAIR}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.requestInstrument(tc.a, tc.pairs)
			require.NoErrorf(t, err, "requestInstrument must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "requestInstrument should name the instrument of %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name  string
		a     asset.Item
		pairs currency.Pairs
		err   error
	}{
		{name: "an empty pair", a: asset.Spot, pairs: currency.Pairs{currency.EMPTYPAIR}, err: currency.ErrCurrencyPairEmpty},
		{name: "a pair of an unsupported asset", a: asset.Options, pairs: currency.Pairs{testSpotPair}, err: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.requestInstrument(tc.a, tc.pairs)
			assert.ErrorIsf(t, err, tc.err, "requestInstrument should refuse %s", tc.name)
		})
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	routes := map[string]testRoute{
		"private/get-open-orders":          testResult(`{"data":[{"order_id":"1","order_type":"LIMIT","side":"BUY","status":"ACTIVE","instrument_name":"BTCUSD-PERP"}]}`),
		"private/advanced/get-open-orders": testResult(`{"data":[{"order_id":"2","order_type":"STOP_LOSS","side":"SELL","status":"ACTIVE","instrument_name":"BTCUSD-PERP","ref_price":"74000"}]}`),
	}
	expOpen := []OrderResponse{{OrderID: "1", OrderType: OrderTypeLimit, Side: SideBuy, Status: OrderStatusActive, InstrumentName: "BTCUSD-PERP"}}
	expAdvanced := []OrderResponse{{OrderID: "2", OrderType: OrderTypeStopLoss, Side: SideSell, Status: OrderStatusActive, InstrumentName: "BTCUSD-PERP", ReferencePrice: preciseNumber("74000")}}
	for _, tc := range []struct {
		name   string
		pairs  currency.Pairs
		params string
	}{
		{name: "a pair's orders", pairs: currency.Pairs{testPerpetualPair}, params: `{"instrument_name":"BTCUSD-PERP"}`},
		{name: "every instrument's orders", params: `{}`},
		{name: "several pairs' orders, from every instrument's", pairs: currency.Pairs{testPerpetualPair, currency.NewPairWithDelimiter("ETHUSD", "PERP", currency.DashDelimiter)}, params: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, routes)
			open, advanced, err := ex.openOrders(t.Context(), asset.PerpetualSwap, tc.pairs)
			require.NoErrorf(t, err, "openOrders must not error for %s", tc.name)
			assert.Equalf(t, expOpen, open, "openOrders should return the trading methods' open orders for %s", tc.name)
			assert.Equalf(t, expAdvanced, advanced, "openOrders should return the advanced order methods' open orders for %s", tc.name)
			assertJSONRequests(t, []testRequest{{Method: "private/get-open-orders", Params: tc.params}, {Method: "private/advanced/get-open-orders", Params: tc.params}}, received())
		})
	}
	t.Run("an empty pair", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, nil)
		_, _, err := ex.openOrders(t.Context(), asset.PerpetualSwap, currency.Pairs{currency.EMPTYPAIR})
		assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "openOrders should refuse an empty pair")
		assert.Empty(t, received(), "openOrders should send no request for an empty pair")
	})
	for _, tc := range []struct {
		name, refused string
		requests      int
	}{
		{name: "the open orders' failure", refused: "private/get-open-orders", requests: 1},
		{name: "the open trigger orders' failure", refused: "private/advanced/get-open-orders", requests: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			failing := maps.Clone(routes)
			failing[tc.refused] = func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} }
			ex, received := newRoutesServerExchange(t, failing)
			open, advanced, err := ex.openOrders(t.Context(), asset.PerpetualSwap, nil)
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "openOrders must return %s", tc.name)
			assert.Equalf(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "openOrders should return Crypto.com's refusal for %s", tc.name)
			assert.Nilf(t, open, "openOrders should return no open orders for %s", tc.name)
			assert.Nilf(t, advanced, "openOrders should return no open trigger orders for %s", tc.name)
			assert.Lenf(t, received(), tc.requests, "openOrders should send no request after %s", tc.name)
		})
	}
	t.Run("the account's open orders", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		ex := e
		if mockTests {
			ex, _ = newRoutesServerExchange(t, routes)
		}
		p := testPair(t, asset.PerpetualSwap)
		name, err := ex.instrumentName(asset.PerpetualSwap, p)
		require.NoError(t, err, "instrumentName must not error")
		open, advanced, err := ex.openOrders(t.Context(), asset.PerpetualSwap, currency.Pairs{p})
		require.NoError(t, err, "openOrders must not error")
		for _, o := range slices.Concat(open, advanced) {
			assert.Equalf(t, name, o.InstrumentName, "openOrders should return only %s's orders", name)
		}
	})
}

func TestOrderDetails(t *testing.T) { //nolint:tparallel // the cases share e and the orders in order
	t.Parallel()
	orders := []OrderResponse{
		{OrderID: "1", OrderType: OrderTypeLimit, Side: SideBuy, Status: OrderStatusActive, InstrumentName: "BTC_USD", Quantity: preciseNumber("0.01"), LimitPrice: preciseNumber("78000")},
		{OrderID: "2", OrderType: OrderTypeLimit, Side: SideBuy, Status: OrderStatusActive, InstrumentName: "BTC_USD", ExecutionInstructions: []string{ExecutionInstructionMarginOrder}, Quantity: preciseNumber("0.01"), LimitPrice: preciseNumber("78000")},
		{OrderID: "3", OrderType: OrderTypeLimit, Side: SideBuy, Status: OrderStatusActive, InstrumentName: "BTCUSD-PERP", Quantity: preciseNumber("0.01"), LimitPrice: preciseNumber("78000")},
		{OrderID: "4", OrderType: OrderTypeLimit, Side: SideBuy, Status: OrderStatusActive, InstrumentName: "BTCUSD-261030", Quantity: preciseNumber("0.01"), LimitPrice: preciseNumber("78000")},
		{OrderID: "5", OrderType: OrderTypeMarket, Side: SideSell, Status: OrderStatusFilled, InstrumentName: "BTC_USD", Quantity: preciseNumber("0.02"), CumulativeQuantity: preciseNumber("0.02"), UpdateTime: types.Time(time.UnixMilli(1792890500000))},
	}
	active := order.Detail{Exchange: e.Name, Side: order.Buy, Type: order.Limit, Status: order.Active, Price: 78000, Amount: 0.01, RemainingAmount: 0.01, CostAsset: currency.USD}
	detail := func(orderID string, a asset.Item, p currency.Pair) order.Detail {
		d := active
		d.OrderID, d.AssetType, d.Pair = orderID, a, p
		switch {
		case a.IsFutures():
			d.MarginType, d.SettlementCurrency = margin.Multi, currency.USD
		case a == asset.Margin:
			d.MarginType = margin.Multi
		}
		return d
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []order.Detail
	}{
		{a: asset.Spot, exp: []order.Detail{
			detail("1", asset.Spot, testSpotPair),
			{Exchange: e.Name, OrderID: "5", Pair: testSpotPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Market, Status: order.Filled, Amount: 0.02, ExecutedAmount: 0.02, CostAsset: currency.USD, LastUpdated: time.UnixMilli(1792890500000), CloseTime: time.UnixMilli(1792890500000)},
		}},
		{a: asset.Margin, exp: []order.Detail{detail("2", asset.Margin, testSpotPair)}},
		{a: asset.PerpetualSwap, exp: []order.Detail{detail("3", asset.PerpetualSwap, testPerpetualPair)}},
		{a: asset.Futures, exp: []order.Detail{detail("4", asset.Futures, testFuturesPair)}},
		{a: asset.Options, exp: []order.Detail{}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			got, err := e.orderDetails(orders, tc.a)
			require.NoErrorf(t, err, "orderDetails must not error for %s", tc.a)
			assert.Equalf(t, tc.exp, got, "orderDetails should return only the %s orders, in order", tc.a)
		})
	}
	t.Run("no orders", func(t *testing.T) {
		got, err := e.orderDetails(nil, asset.Spot)
		require.NoError(t, err, "orderDetails must not error without orders")
		assert.Empty(t, got, "orderDetails should return no details without orders")
	})
	for _, tc := range []struct {
		name    string
		invalid OrderResponse
		err     error
	}{
		{name: "a malformed instrument", invalid: OrderResponse{OrderID: "6", OrderType: OrderTypeLimit, Side: SideBuy, Status: OrderStatusActive, InstrumentName: "_USD"}, err: errMalformedInstrumentName},
		{name: "an order of an unknown side", invalid: OrderResponse{OrderID: "6", OrderType: OrderTypeLimit, Side: "HOLD", Status: OrderStatusActive, InstrumentName: "BTC_USD"}, err: order.ErrSideIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.orderDetails([]OrderResponse{orders[0], tc.invalid}, asset.Spot)
			assert.ErrorIsf(t, err, tc.err, "orderDetails should refuse %s", tc.name)
			assert.Nilf(t, got, "orderDetails should return none of the orders when it refuses %s", tc.name)
		})
	}
}

func TestOrderHistory(t *testing.T) {
	t.Parallel()
	ns := func(at time.Time) string { return strconv.FormatInt(at.UnixNano(), 10) }
	ms := func(at time.Time) string { return strconv.FormatInt(at.UnixMilli(), 10) }
	end := testWindowStart
	start := end.Add(-12 * time.Hour)
	day := end.Add(-privateHistoryWindow)
	window := func(method, instrument string, from, to time.Time) testRequest {
		limit := `100`
		if strings.HasPrefix(method, "private/advanced/") {
			limit = `"100"`
		}
		params := `"start_time":"` + ns(from) + `","end_time":"` + ns(to) + `","limit":` + limit
		if instrument != "" {
			params = `"instrument_name":"` + instrument + `",` + params
		}
		return testRequest{Method: method, Params: "{" + params + "}"}
	}
	// Order 0 was created before every window the cases request, which is left out
	orders := testResult(`{"data":[{"order_id":"1","create_time":"` + ms(end.Add(-time.Hour)) + `"},{"order_id":"0","create_time":"` + ms(end.Add(-25*time.Hour)) + `"}]}`)
	triggers := testResult(`{"data":[{"order_id":"2","order_type":"STOP_LOSS","create_time":"` + ms(end.Add(-2*time.Hour)) + `"}]}`)
	exp := []OrderResponse{
		{OrderID: "1", CreateTime: types.Time(end.Add(-time.Hour))},
		{OrderID: "2", OrderType: OrderTypeStopLoss, CreateTime: types.Time(end.Add(-2 * time.Hour))},
	}
	for _, tc := range []struct {
		name       string
		instrument string
		start      time.Time
		requests   []testRequest
	}{
		{
			name:     "every instrument's orders over a window of a day or less",
			start:    start,
			requests: []testRequest{window("private/get-order-history", "", start, end), window("private/advanced/get-order-history", "", start, end)},
		},
		{
			name:       "an instrument's orders",
			instrument: "BTCUSD-PERP",
			start:      start,
			requests:   []testRequest{window("private/get-order-history", "BTCUSD-PERP", start, end), window("private/advanced/get-order-history", "BTCUSD-PERP", start, end)},
		},
		{
			name:     "no start time, the day before the end",
			requests: []testRequest{window("private/get-order-history", "", day, end), window("private/advanced/get-order-history", "", day, end)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-order-history": orders, "private/advanced/get-order-history": triggers})
			got, err := ex.orderHistory(t.Context(), tc.instrument, tc.start, end)
			require.NoErrorf(t, err, "orderHistory must not error for %s", tc.name)
			assert.Equalf(t, exp, got, "orderHistory should return the window's orders, then its trigger orders, for %s", tc.name)
			assertJSONRequests(t, tc.requests, received())
		})
	}
	t.Run("a window longer than a day, a day at a time from its end", func(t *testing.T) {
		t.Parallel()
		from := end.Add(-36 * time.Hour)
		ex, received := newRoutesServerExchange(t, map[string]testRoute{
			"private/get-order-history":          testResult(`{"data":[{"order_id":"1","create_time":"` + ms(end.Add(-time.Hour)) + `"},{"order_id":"3","create_time":"` + ms(end.Add(-30*time.Hour)) + `"}]}`),
			"private/advanced/get-order-history": testResult(`{"data":[]}`),
		})
		got, err := ex.orderHistory(t.Context(), "", from, end)
		require.NoError(t, err, "orderHistory must not error")
		expOrders := []OrderResponse{{OrderID: "1", CreateTime: types.Time(end.Add(-time.Hour))}, {OrderID: "3", CreateTime: types.Time(end.Add(-30 * time.Hour))}}
		assert.Equal(t, expOrders, got, "orderHistory should return each day's orders, the latest day's first")
		assertJSONRequests(t, []testRequest{
			window("private/get-order-history", "", day, end),
			window("private/get-order-history", "", from, day),
			window("private/advanced/get-order-history", "", day, end),
			window("private/advanced/get-order-history", "", from, day),
		}, received())
	})
	t.Run("no end time, the current time", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-order-history": testResult(`{"data":[]}`), "private/advanced/get-order-history": testResult(`{"data":[]}`)})
		before := time.Now()
		_, err := ex.orderHistory(t.Context(), "", time.Time{}, time.Time{})
		after := time.Now()
		require.NoError(t, err, "orderHistory must not error without a window")
		requests := received()
		require.Len(t, requests, 2, "orderHistory must request each history once")
		for _, r := range requests {
			var sent struct {
				StartTime int64 `json:"start_time,string"`
				EndTime   int64 `json:"end_time,string"`
			}
			require.NoErrorf(t, json.Unmarshal([]byte(r.Params), &sent), "%s's params must decode", r.Method)
			assert.GreaterOrEqualf(t, sent.EndTime, before.UnixNano(), "orderHistory should end %s's window at the current time", r.Method)
			assert.LessOrEqualf(t, sent.EndTime, after.UnixNano(), "orderHistory should end %s's window at the current time", r.Method)
			assert.Equalf(t, privateHistoryWindow.Nanoseconds(), sent.EndTime-sent.StartTime, "orderHistory should start %s's window the day before its end", r.Method)
		}
	})
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
	}{
		{name: "a start after the end", start: end, end: start, err: common.ErrStartAfterEnd},
		{name: "a start at the end", start: end, end: end, err: common.ErrStartEqualsEnd},
		{name: "a start after the current time", start: time.Now().Add(time.Hour), end: time.Now().Add(2 * time.Hour), err: common.ErrStartAfterTimeNow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, nil)
			_, err := ex.orderHistory(t.Context(), "", tc.start, tc.end)
			assert.ErrorIsf(t, err, tc.err, "orderHistory should refuse %s", tc.name)
			assert.Emptyf(t, received(), "orderHistory should send no request for %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name, refused string
		requests      int
	}{
		{name: "the orders' failure", refused: "private/get-order-history", requests: 1},
		{name: "the trigger orders' failure", refused: "private/advanced/get-order-history", requests: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			routes := map[string]testRoute{"private/get-order-history": orders, "private/advanced/get-order-history": triggers}
			routes[tc.refused] = func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} }
			ex, received := newRoutesServerExchange(t, routes)
			got, err := ex.orderHistory(t.Context(), "", start, end)
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "orderHistory must return %s", tc.name)
			assert.Equalf(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "orderHistory should return Crypto.com's refusal for %s", tc.name)
			assert.Nilf(t, got, "orderHistory should return no orders for %s", tc.name)
			assert.Lenf(t, received(), tc.requests, "orderHistory should send no request after %s", tc.name)
		})
	}
	t.Run("the account's order history", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		to := time.Now()
		from := to.Add(-time.Hour)
		ex := e
		if mockTests {
			ex, _ = newRoutesServerExchange(t, map[string]testRoute{
				"private/get-order-history":          testResult(`{"data":[{"order_id":"1","create_time":"` + ms(to.Add(-time.Minute)) + `"}]}`),
				"private/advanced/get-order-history": testResult(`{"data":[{"order_id":"2","create_time":"` + ms(from.Add(-time.Minute)) + `"}]}`),
			})
		}
		got, err := ex.orderHistory(mockRequestContext(t), "", from, to)
		require.NoError(t, err, "orderHistory must not error")
		for i := range got {
			_, at := orderRecord(&got[i])
			assert.Falsef(t, at.Before(from) || !at.Before(to), "orderHistory should return only orders created in the window, not order %s at %s", got[i].OrderID, at)
		}
	})
}

func TestLatestValuation(t *testing.T) {
	t.Parallel()
	t.Run("the latest value", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"public/get-valuations": testResult(`{"instrument_name":"BTCUSD-PERP","data":[{"v":"80819.5","t":1792899065000}]}`)})
		got, err := ex.latestValuation(t.Context(), "BTCUSD-PERP", ValuationTypeMarkPrice)
		require.NoError(t, err, "latestValuation must not error")
		assert.Equal(t, &Valuation{Value: 80819.5, Timestamp: types.Time(time.UnixMilli(1792899065000))}, got, "latestValuation should return the value")
		assert.Equal(t, []testRequest{{Method: "public/get-valuations", Params: "count=1&instrument_name=BTCUSD-PERP&valuation_type=mark_price"}}, received(), "latestValuation should request the instrument's one latest value of the type")
	})
	t.Run("the first of several values, the latest", func(t *testing.T) {
		t.Parallel()
		ex, _ := newRoutesServerExchange(t, map[string]testRoute{"public/get-valuations": testResult(`{"instrument_name":"BTCUSD-PERP","data":[{"v":"80819.5","t":1792899065000},{"v":"80810","t":1792899064000}]}`)})
		got, err := ex.latestValuation(t.Context(), "BTCUSD-PERP", ValuationTypeMarkPrice)
		require.NoError(t, err, "latestValuation must not error")
		assert.Equal(t, &Valuation{Value: 80819.5, Timestamp: types.Time(time.UnixMilli(1792899065000))}, got, "latestValuation should return the first value, Crypto.com's latest")
	})
	t.Run("no value", func(t *testing.T) {
		t.Parallel()
		ex, _ := newRoutesServerExchange(t, map[string]testRoute{"public/get-valuations": testResult(`{"instrument_name":"BTCUSD-PERP","data":[]}`)})
		got, err := ex.latestValuation(t.Context(), "BTCUSD-PERP", ValuationTypeMarkPrice)
		assert.ErrorIs(t, err, common.ErrNoResponse, "latestValuation should refuse a reply without a value")
		assert.Nil(t, got, "latestValuation should return no value")
	})
	t.Run("the request's failure", func(t *testing.T) {
		t.Parallel()
		ex, _ := newRoutesServerExchange(t, map[string]testRoute{"public/get-valuations": func(string) (string, *APIError) {
			return "", &APIError{Code: 40004, Message: "Invalid instrument_name"}
		}})
		_, err := ex.latestValuation(t.Context(), "BTCUSD-PERP", ValuationTypeMarkPrice)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "latestValuation must return Crypto.com's refusal")
		assert.Equal(t, &APIError{Code: 40004, Message: "Invalid instrument_name"}, apiErr, "latestValuation should return Crypto.com's code and message")
	})
	for _, tc := range []struct {
		name, instrument, valuationType string
		err                             error
	}{
		{name: "no instrument", valuationType: ValuationTypeMarkPrice, err: errInstrumentNameRequired},
		{name: "no valuation type", instrument: "BTCUSD-PERP", err: errValuationTypeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, nil)
			_, err := ex.latestValuation(t.Context(), tc.instrument, tc.valuationType)
			assert.ErrorIsf(t, err, tc.err, "latestValuation should refuse %s", tc.name)
			assert.Emptyf(t, received(), "latestValuation should send no request for %s", tc.name)
		})
	}
	t.Run("a perpetual swap's funding rate", func(t *testing.T) {
		t.Parallel()
		name, err := e.instrumentName(asset.PerpetualSwap, testPair(t, asset.PerpetualSwap))
		require.NoError(t, err, "instrumentName must not error")
		got, err := e.latestValuation(t.Context(), name, ValuationTypeFundingRate)
		require.NoErrorf(t, err, "latestValuation must not error for %s", name)
		if mockTests {
			assert.Equal(t, &Valuation{Value: 0.000009839, Timestamp: types.Time(time.UnixMilli(1792899065000))}, got, "latestValuation should return the recorded funding rate")
		} else {
			assert.NotZerof(t, got.Timestamp.Time(), "latestValuation should time %s's funding rate", name)
		}
	})
}

func TestTransactionTime(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		record TransactionResponse
		exp    time.Time
	}{
		{
			name:   "a transaction timed to the nanosecond",
			record: TransactionResponse{EventTime: types.Time(time.UnixMilli(1792800000123)), EventTimeNano: types.Time(time.Unix(0, 1792800000123456789))},
			exp:    time.Unix(0, 1792800000123456789),
		},
		{name: "a transaction timed to the millisecond", record: TransactionResponse{EventTime: types.Time(time.UnixMilli(1792800300000))}, exp: time.UnixMilli(1792800300000)},
		{name: "a transaction without a time", exp: time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, transactionTime(&tc.record), "transactionTime should time %s", tc.name)
		})
	}
}

func TestFindIsolatedPosition(t *testing.T) {
	t.Parallel()
	// Each case searches balances of its own, so that the position found can be told apart by its address
	balances := func() []UserBalanceResponse {
		return []UserBalanceResponse{
			{PositionBalances: []CollateralBalance{{InstrumentName: "USD", Quantity: preciseNumber("100")}}},
			{IsolatedPositions: []IsolatedPositionBalance{{IsolationID: "19848526", Leverage: 2}, {IsolationID: "19848527", Leverage: 5}}},
			{IsolatedPositions: []IsolatedPositionBalance{{IsolationID: "19848528", Leverage: 3}, {IsolationID: "19848527", Leverage: 10}}},
		}
	}
	for _, tc := range []struct {
		name            string
		isolationID     string
		entry, position int
	}{
		{name: "a position of the first entry holding isolated positions", isolationID: "19848526", entry: 1, position: 0},
		{name: "the first of two matching positions", isolationID: "19848527", entry: 1, position: 1},
		{name: "a position of a later entry", isolationID: "19848528", entry: 2, position: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := balances()
			got := findIsolatedPosition(b, func(p *IsolatedPositionBalance) bool { return p.IsolationID == tc.isolationID })
			assert.Samef(t, &b[tc.entry].IsolatedPositions[tc.position], got, "findIsolatedPosition should return %s", tc.name)
		})
	}
	t.Run("no matching position", func(t *testing.T) {
		t.Parallel()
		var searched []string
		got := findIsolatedPosition(balances(), func(p *IsolatedPositionBalance) bool {
			searched = append(searched, p.IsolationID)
			return false
		})
		assert.Nil(t, got, "findIsolatedPosition should return no position when none matches")
		assert.Equal(t, []string{"19848526", "19848527", "19848528", "19848527"}, searched, "findIsolatedPosition should search every isolated position, in order")
	})
	t.Run("no balances", func(t *testing.T) {
		t.Parallel()
		got := findIsolatedPosition(nil, func(*IsolatedPositionBalance) bool { return true })
		assert.Nil(t, got, "findIsolatedPosition should return no position without balances")
	})
}

func TestCrossMarginLeverage(t *testing.T) {
	t.Parallel()
	refused := func(string) (string, *APIError) { return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"} }
	btcPerpetual := func(maximum string) testRoute {
		return testResult(`{"data":[{"symbol":"BTCUSD-PERP","inst_type":"PERPETUAL_SWAP","max_leverage":"` + maximum + `","tradable":true}]}`)
	}
	for _, tc := range []struct {
		name        string
		settings    string
		instruments testRoute
		exp         types.Number
	}{
		{name: "the account's leverage below the instrument's maximum", settings: `[{"leverage":"5"}]`, instruments: btcPerpetual("20"), exp: 5},
		{name: "the account's leverage at the instrument's maximum", settings: `[{"leverage":"20"}]`, instruments: btcPerpetual("20"), exp: 20},
		{name: "the account's leverage above the instrument's maximum", settings: `[{"leverage":"50"}]`, instruments: btcPerpetual("20"), exp: 20},
		{name: "the first of several settings", settings: `[{"leverage":"10"},{"leverage":"30"}]`, instruments: btcPerpetual("20"), exp: 10},
		{
			name:        "the instrument among others",
			settings:    `[{"leverage":"50"}]`,
			instruments: testResult(`{"data":[{"symbol":"ETHUSD-PERP","max_leverage":"10"},{"symbol":"BTCUSD-PERP","max_leverage":"25"},{"symbol":"BTCUSD-261030","max_leverage":"5"}]}`),
			exp:         25,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-account-settings": testResult(tc.settings), "public/get-instruments": tc.instruments})
			got, err := ex.crossMarginLeverage(t.Context(), "BTCUSD-PERP")
			require.NoErrorf(t, err, "crossMarginLeverage must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "crossMarginLeverage should return the lower of the account's leverage and the instrument's maximum for %s", tc.name)
			assert.Equalf(t, []testRequest{{Method: "private/get-account-settings", Params: `{}`}, {Method: "public/get-instruments"}}, received(), "crossMarginLeverage should read the account's settings and the instruments for %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name        string
		instruments testRoute
		err         error
	}{
		{name: "an instrument missing from the instruments", instruments: testResult(`{"data":[{"symbol":"ETHUSD-PERP","max_leverage":"20"}]}`), err: errInstrumentNotFound},
		{name: "an instrument without a maximum leverage", instruments: testResult(`{"data":[{"symbol":"BTCUSD-PERP"}]}`), err: errNoMaximumLeverage},
		{name: "a negative maximum leverage", instruments: btcPerpetual("-1"), err: errNoMaximumLeverage},
		{name: "a maximum leverage that is not a number", instruments: btcPerpetual("NaN"), err: errNoMaximumLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, _ := newRoutesServerExchange(t, map[string]testRoute{"private/get-account-settings": testResult(`[{"leverage":"50"}]`), "public/get-instruments": tc.instruments})
			got, err := ex.crossMarginLeverage(t.Context(), "BTCUSD-PERP")
			assert.ErrorIsf(t, err, tc.err, "crossMarginLeverage should refuse %s", tc.name)
			assert.Zerof(t, got, "crossMarginLeverage should return no leverage for %s", tc.name)
		})
	}
	t.Run("no settings", func(t *testing.T) {
		t.Parallel()
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-account-settings": testResult(`[]`)})
		_, err := ex.crossMarginLeverage(t.Context(), "BTCUSD-PERP")
		assert.ErrorIs(t, err, common.ErrNoResponse, "crossMarginLeverage should refuse a reply without settings")
		assert.Len(t, received(), 1, "crossMarginLeverage should not read the instruments without settings")
	})
	for _, tc := range []struct {
		name   string
		routes map[string]testRoute
		sent   int
	}{
		{name: "the settings' failure", routes: map[string]testRoute{"private/get-account-settings": refused}, sent: 1},
		{name: "the instruments' failure", routes: map[string]testRoute{"private/get-account-settings": testResult(`[{"leverage":"50"}]`), "public/get-instruments": refused}, sent: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, received := newRoutesServerExchange(t, tc.routes)
			_, err := ex.crossMarginLeverage(t.Context(), "BTCUSD-PERP")
			var apiErr *APIError
			require.ErrorAsf(t, err, &apiErr, "crossMarginLeverage must return %s", tc.name)
			assert.Equalf(t, &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}, apiErr, "crossMarginLeverage should return Crypto.com's refusal for %s", tc.name)
			assert.Lenf(t, received(), tc.sent, "crossMarginLeverage should send no request after %s", tc.name)
		})
	}
	t.Run("the account's leverage on a perpetual swap", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		name, err := e.instrumentName(asset.PerpetualSwap, testPair(t, asset.PerpetualSwap))
		require.NoError(t, err, "instrumentName must not error")
		got, err := e.crossMarginLeverage(t.Context(), name)
		require.NoErrorf(t, err, "crossMarginLeverage must not error for %s", name)
		if mockTests {
			assert.Equal(t, types.Number(20), got, "crossMarginLeverage should return the recorded account's leverage, below the instrument's maximum of 100")
		} else {
			assert.Positivef(t, got.Float64(), "crossMarginLeverage should return %s's leverage", name)
		}
	})
}
