package cryptodotcom

import (
	"context"
	"fmt"
	"maps"
	"math"
	"net/url"
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
	"github.com/thrasher-corp/gocryptotrader/currency"
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

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must refuse an unsupported asset")
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		if mockTests {
			assert.Equalf(t, testInstrumentPairs[a], pairs, "FetchTradablePairs should return the pairs of the tradable %s instruments", a)
		} else {
			assert.NotEmptyf(t, pairs, "FetchTradablePairs should return %s pairs", a)
		}
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

func TestTradablePairs(t *testing.T) {
	t.Parallel()
	instruments := []InstrumentResponse{
		{Symbol: "BTC_USD", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.BTC, QuoteCurrency: currency.USD, Tradable: true, MarginBuyEnabled: true},
		{Symbol: "ETH_USD", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.ETH, QuoteCurrency: currency.USD, Tradable: true},
		{Symbol: "XRP_USD", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.XRP, QuoteCurrency: currency.USD, MarginSellEnabled: true},
		{Symbol: "SHIB_BTC@OEX_HK", InstrumentType: InstrumentTypeCurrencyPair, BaseCurrency: currency.SHIB, QuoteCurrency: currency.BTC, Tradable: true, MarginBuyEnabled: true},
		{Symbol: "BTCUSD-PERP", InstrumentType: InstrumentTypePerpetualSwap, BaseCurrency: currency.BTC, QuoteCurrency: currency.USD, Tradable: true},
		{Symbol: "BTCUSD-261030", InstrumentType: InstrumentTypeFuture, BaseCurrency: currency.BTC, QuoteCurrency: currency.USD, Tradable: true},
	}
	for a, exp := range map[asset.Item]currency.Pairs{
		asset.Spot:          {testSpotPair, currency.NewPairWithDelimiter("ETH", "USD", currency.UnderscoreDelimiter)},
		asset.Margin:        {testSpotPair},
		asset.PerpetualSwap: {testPerpetualPair},
		asset.Futures:       {testFuturesPair},
		asset.Options:       {},
	} {
		pairs, err := tradablePairs(instruments, a)
		require.NoErrorf(t, err, "tradablePairs must not error for %s", a)
		assert.Equalf(t, exp, pairs, "tradablePairs should return the pairs of the tradable %s instruments a pair names", a)
	}
	_, err := tradablePairs([]InstrumentResponse{{Symbol: "BTCUSDPERP", InstrumentType: InstrumentTypePerpetualSwap, Tradable: true}}, asset.PerpetualSwap)
	assert.ErrorIs(t, err, errMalformedInstrumentName, "tradablePairs should refuse a derivative named without a dash")
}

func TestPairFromInstrument(t *testing.T) {
	t.Parallel()
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
		p, err := pairFromInstrument(tc.a, tc.name)
		require.NoErrorf(t, err, "pairFromInstrument must not error for %s %s", tc.a, tc.name)
		assert.Equalf(t, tc.exp, p, "pairFromInstrument should split %s at its last delimiter", tc.name)
	}
	for _, name := range []string{"BTCUSDPERP", "-PERP", "BTCUSD-"} {
		_, err := pairFromInstrument(asset.PerpetualSwap, name)
		assert.ErrorIsf(t, err, errMalformedInstrumentName, "pairFromInstrument should refuse %q", name)
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
		assert.Equalf(t, tc.exp, orderAsset(tc.name, tc.execInst), "orderAsset should place an order on %s with %v", tc.name, tc.execInst)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateOrderExecutionLimits must refuse an unsupported asset")
	for _, a := range ex.GetAssetTypes(false) {
		require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
		p := testPair(t, a)
		l, err := ex.GetOrderExecutionLimits(a, p)
		require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", a, p)
		if !mockTests {
			assert.Positivef(t, l.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load %s %s's price step", a, p)
			assert.Positivef(t, l.AmountStepIncrementSize, "UpdateOrderExecutionLimits should load %s %s's quantity step", a, p)
			continue
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
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers must refuse an unsupported asset")
	achUSD := currency.NewPairWithDelimiter("ACH", "USD", currency.UnderscoreDelimiter)
	shazPerpetual := currency.NewPairWithDelimiter("SHAZUSD", "PERP", currency.DashDelimiter)
	if mockTests {
		// The recorded tickers of every instrument hold ACH_USD's and SHAZUSD-PERP's, so those pairs are the ones enabled
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: achUSD, asset.PerpetualSwap: shazPerpetual} {
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must store %s's available pair", a)
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must store %s's enabled pair", a)
		}
	}
	for _, a := range ex.GetAssetTypes(true) {
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
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

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must refuse an empty pair")
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap} {
		p := testPair(t, a)
		got, err := ex.UpdateTicker(t.Context(), p, a)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s %s", a, p)
		if !mockTests {
			assert.Positivef(t, got.Last, "UpdateTicker should return %s %s's last price", a, p)
			continue
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
	}
}

func TestTickerPrice(t *testing.T) {
	t.Parallel()
	ethBTC := currency.NewPairWithDelimiter("ETH", "BTC", currency.UnderscoreDelimiter)
	got := e.tickerPrice(&TickerResponse{InstrumentName: "ETH_BTC", LastPrice: 0.0325, Volume24Hour: 12, USDVolume24Hour: 31000}, asset.Spot, ethBTC)
	exp := &ticker.Price{Last: 0.0325, BaseVolume: 12, Pair: ethBTC, ExchangeName: e.Name, AssetType: asset.Spot}
	assert.Equal(t, exp, got, "tickerPrice should leave out the USD volume of a pair not quoted in USD")
}

func TestPercentFromFraction(t *testing.T) {
	t.Parallel()
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
		assert.Equalf(t, tc.exp, percentFromFraction(tc.fraction), "percentFromFraction should return %s as a percentage", tc.fraction)
	}
	var fraction types.Number
	require.NoError(t, json.Unmarshal([]byte(`"NaN"`), &fraction), "Unmarshal must not error")
	assert.True(t, math.IsNaN(percentFromFraction(fraction)), "percentFromFraction should return NaN for a fraction that is not a number")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must refuse an empty pair")
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualSwap} {
		p := testPair(t, a)
		got, err := ex.UpdateOrderbook(t.Context(), p, a)
		require.NoErrorf(t, err, "UpdateOrderbook must not error for %s %s", a, p)
		if !mockTests {
			assert.NotEmptyf(t, got.Bids, "UpdateOrderbook should return %s %s's bids", a, p)
			assert.NotEmptyf(t, got.Asks, "UpdateOrderbook should return %s %s's asks", a, p)
			continue
		}
		exp := testOrderbooks[a]
		exp.Exchange, exp.Pair, exp.Asset, exp.ValidateOrderbook = ex.Name, p, a, ex.ValidateOrderbook
		assert.NotZerof(t, got.InsertedAt, "UpdateOrderbook should time when %s %s's book was stored", a, p)
		got.InsertedAt = time.Time{}
		assert.Equalf(t, &exp, got, "UpdateOrderbook should return %s %s's book", a, p)
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

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must refuse an unsupported asset")
	skipLiveWithoutCredentials(t)
	for _, a := range ex.GetAssetTypes(false) {
		got, err := ex.UpdateAccountBalances(t.Context(), a)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
		require.Lenf(t, got, 1, "UpdateAccountBalances must return the one account for %s", a)
		if !mockTests {
			continue
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
	}
	if !mockTests {
		return
	}
	// The balances are saved as a snapshot, so a balance the response no longer holds is removed
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	for _, a := range ex.GetAssetTypes(false) {
		stale := accounts.NewSubAccount(a, "")
		stale.Balances.Set(currency.BTC, accounts.Balance{Total: 1, Free: 1})
		require.NoErrorf(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stale}, false), "Save must not error for %s", a)
		_, err = ex.UpdateAccountBalances(t.Context(), a)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
		stored, err := ex.Accounts.CurrencyBalances(creds, a)
		require.NoErrorf(t, err, "CurrencyBalances must not error for %s", a)
		assert.NotContainsf(t, stored, currency.BTC, "UpdateAccountBalances should remove a %s balance the response no longer holds", a)
	}
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
	assert.NoError(t, newTestExchange(t).ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error")
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

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
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

func TestPageWalletHistory(t *testing.T) {
	t.Parallel()
	ids := func(start, n int) []string {
		page := make([]string, n)
		for i := range page {
			page[i] = strconv.Itoa(start + i)
		}
		return page
	}
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

	pages = nil
	got, err = pageWalletHistory(t.Context(), func(_ context.Context, page uint64) ([]string, error) {
		pages = append(pages, page)
		require.LessOrEqual(t, len(pages), testServerRequestLimit, "pageWalletHistory must stop within the test server's request limit")
		return ids(0, maxWalletHistoryPageSize), nil
	}, func(id *string) string { return *id })
	require.NoError(t, err, "pageWalletHistory must not error for a venue ignoring the page")
	assert.Equal(t, ids(0, maxWalletHistoryPageSize), got, "pageWalletHistory should return a repeated page's records once")
	assert.Equal(t, []uint64{0, 1}, pages, "pageWalletHistory should stop at a page holding nothing new")

	_, err = pageWalletHistory(t.Context(), func(_ context.Context, page uint64) ([]string, error) {
		pages = append(pages, page)
		require.LessOrEqual(t, len(pages), testServerRequestLimit, "pageWalletHistory must stop within the test server's request limit")
		return nil, errInvalidCount
	}, func(id *string) string { return *id })
	assert.ErrorIs(t, err, errInvalidCount, "pageWalletHistory should return a page's error")
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
		assert.Equalf(t, exp, depositStatus(status), "depositStatus should describe status %s", status)
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
		assert.Equalf(t, exp, withdrawalStatus(status), "withdrawalStatus should describe status %s", status)
	}
	assert.Equal(t, currency.BTC, withdrawalCurrency(&WithdrawalResponse{Symbol: currency.BTC}), "withdrawalCurrency should fall back to the symbol create-withdrawal's example names")
}

func TestSplitAddressTag(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string][2]string{
		"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf?1234567890": {"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf", "1234567890"},
		"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf":            {"2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf", ""},
	} {
		address, tag := splitAddressTag(in)
		assert.Equalf(t, exp, [2]string{address, tag}, "splitAddressTag should split %s", in)
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
	_, err := e.GetHistoricTrades(t.Context(), testSpotPair, asset.Spot, testWindowStart, testWindowStart)
	require.ErrorIs(t, err, common.ErrStartEqualsEnd, "GetHistoricTrades must refuse an empty window")
	_, err = e.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Spot, testWindowStart, testWindowStart.Add(time.Minute))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricTrades must refuse an empty pair")
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

func TestGetHistoricTradesWindows(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the windows requested are checked against a test server")
	}
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
	t.Parallel()
	start := testWindowStart
	end := start.Add(time.Hour)
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }

	// Two records share a time a page splits, and one falls before the window
	records := []testPagedRecord{{"6", at(50)}, {"5", at(40)}, {"4", at(30)}, {"3", at(30)}, {"2", at(20)}, {"1", at(-10)}}
	var windows [][2]time.Time
	got, err := testPager(t, records, 3, &windows).backwards(t.Context(), start, end, make(map[string]struct{}))
	require.NoError(t, err, "backwards must not error")
	assert.Equal(t, records[:5], got, "backwards should return every record in the window once")
	expWindows := [][2]time.Time{{start, end}, {start, at(30).Add(time.Nanosecond)}, {start, at(20).Add(time.Nanosecond)}}
	assert.Equal(t, expWindows, windows, "backwards should end each page a nanosecond after the oldest record so far")

	// A full page sharing one time holds nothing new when requested again, so its time is passed over
	records = []testPagedRecord{{"C", at(30)}, {"B", at(30)}, {"A", at(30)}, {"D", at(10)}}
	windows = nil
	got, err = testPager(t, records, 2, &windows).backwards(t.Context(), start, end, make(map[string]struct{}))
	require.NoError(t, err, "backwards must not error for a time a page cannot page within")
	assert.Equal(t, []testPagedRecord{records[0], records[1], records[3]}, got, "backwards should pass over a time holding more records than a page")
	expWindows = [][2]time.Time{{start, end}, {start, at(30).Add(time.Nanosecond)}, {start, at(30)}}
	assert.Equal(t, expWindows, windows, "backwards should end the page after a page holding nothing new at its oldest time")

	// A full page sharing the last time of the window passes over it at once, as a page ending a nanosecond later is
	// the page itself
	records = []testPagedRecord{{"B", end.Add(-time.Nanosecond)}, {"A", end.Add(-time.Nanosecond)}, {"C", end.Add(-time.Nanosecond)}, {"D", at(10)}}
	windows = nil
	got, err = testPager(t, records, 2, &windows).backwards(t.Context(), start, end, make(map[string]struct{}))
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
	got, err = pager.backwards(t.Context(), start, end, make(map[string]struct{}))
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
	seen := map[string]struct{}{"6": {}}
	got, err = pager.backwards(t.Context(), at(35), at(45), seen)
	require.NoError(t, err, "backwards must not error for records seen before")
	assert.Equal(t, []testPagedRecord{{"5", at(40)}}, got, "backwards should leave out records seen before and outside the window")
	assert.Equal(t, map[string]struct{}{"6": {}, "5": {}}, seen, "backwards should add the records it returns to those seen")

	pager.fetch = func(_ context.Context, start, end time.Time) ([]testPagedRecord, error) {
		windows = append(windows, [2]time.Time{start, end})
		require.LessOrEqual(t, len(windows), testServerRequestLimit, "the pager must stop within the test server's request limit")
		return nil, errInvalidCount
	}
	_, err = pager.backwards(t.Context(), start, end, make(map[string]struct{}))
	assert.ErrorIs(t, err, errInvalidCount, "backwards should return a page's error")
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

func TestHistoryPagerWarnsOfPassedOverTimes(t *testing.T) {
	// Not parallel, as it captures the log
	takeWarnings := captureWarnings(t)
	start := testWindowStart
	end := start.Add(time.Hour)
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }
	warning := func(limit int, at time.Time) string {
		return "test history holds " + strconv.Itoa(limit) + " or more records at " + at.String() + ", which no request can page within, so any more are left out"
	}
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
		var windows [][2]time.Time
		_, err := testPager(t, tc.records, tc.limit, &windows).backwards(t.Context(), start, end, make(map[string]struct{}))
		require.NoErrorf(t, err, "backwards must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, takeWarnings(), "backwards should warn of each time it passes over for %s", tc.name)
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
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
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

func TestCandles(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the candle window is checked against a test server")
	}
	// Crypto.com takes in a monthly candle opening at a window's end, so the window ends a millisecond early; the reply was
	// recorded live for this request
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"public/get-candlestick": testResult(`{"interval":"1M","data":[{"o":"58562.87","h":"66925.83","l":"57737.07","c":"62840.00","v":"126396.4943","t":1782864000000},{"o":"62843.05","h":"81500.00","l":"62216.48","c":"78559.98","v":"122373.96006","t":1785542400000},{"o":"78571.17","h":"87427.66","l":"74897.07","c":"83564.32","v":"109085.85444","t":1788220800000}],"instrument_name":"BTC_USD"}`),
	})
	start, end := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	got, err := ex.candles(t.Context(), "BTC_USD", kline.OneMonth, start, end)
	require.NoError(t, err, "candles must not error")
	exp := []kline.Candle{
		{Time: time.UnixMilli(1782864000000), Open: 58562.87, High: 66925.83, Low: 57737.07, Close: 62840, Volume: 126396.4943},
		{Time: time.UnixMilli(1785542400000), Open: 62843.05, High: 81500, Low: 62216.48, Close: 78559.98, Volume: 122373.96006},
		{Time: time.UnixMilli(1788220800000), Open: 78571.17, High: 87427.66, Low: 74897.07, Close: 83564.32, Volume: 109085.85444},
	}
	assert.Equal(t, exp, got, "candles should return the monthly candles opening within the window")
	assert.Equal(t, []testRequest{{Method: "public/get-candlestick", Params: "count=300&end_ts=1790812799999&instrument_name=BTC_USD&start_ts=1782864000000&timeframe=1M"}}, received(), "candles should end a monthly window a millisecond early")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must refuse an empty currency")
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

	ex := newResponseServerExchange(t, `{"id":1,"method":"private/get-deposit-address","code":0,"result":{"deposit_address_list":[
		{"currency":"USDC","address":"0xabc","status":"1","network":"ETH"},
		{"currency":"USDC","address":"Gh7?memo","status":"1","network":"SOL"}]}}`)
	_, err = ex.GetDepositAddress(t.Context(), currency.USDC, "", "")
	assert.ErrorIs(t, err, errChainRequired, "GetDepositAddress should refuse a currency with active addresses on several chains without a chain")
	assert.True(t, ex.Features.Supports.RESTCapabilities.MultiChainDepositRequiresChainSet, "Crypto.com should require a chain for a currency deposited on several networks, so the deposit address sync asks for each")
	got, err = ex.GetDepositAddress(t.Context(), currency.USDC, "", "SOL")
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

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must validate the request")
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("a live withdrawal would move funds")
	}
	got, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:      e.Name,
		Currency:      currency.BTC,
		Amount:        1,
		Type:          withdraw.Crypto,
		ClientOrderID: "my_withdrawal_002",
		Crypto:        withdraw.CryptoRequest{Address: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf", Chain: "BTC"},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, &withdraw.ExchangeResponse{Name: e.Name, ID: "2220"}, got, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")

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
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFeeByType must refuse an unsupported fee type")
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
		fee, err := e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(c, currency.EMPTYCODE)})
		require.NoErrorf(t, err, "GetFeeByType must not error for %s's withdrawal fee", c)
		assert.Equalf(t, exp, fee, "GetFeeByType should return %s's withdrawal fee on its default network, or its only one", c)
	}
	ex := newResponseServerExchange(t, `{"id":1,"method":"private/get-currency-networks","code":0,"result":{"currency_map":{"USDC":{"default_network":"SOL","network_list":[{"network_id":"ETH"},{"network_id":"BNB"}]}}}}`)
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.USDC, currency.EMPTYCODE)})
	assert.ErrorIs(t, err, errNoDefaultNetwork, "GetFeeByType should refuse a currency whose default network it does not list")
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

func TestNewOrderRequest(t *testing.T) {
	t.Parallel()
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
		s := testSubmit()
		if tc.modify != nil {
			tc.modify(s)
		}
		got, err := e.newOrderRequest(s)
		require.NoErrorf(t, err, "newOrderRequest must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "newOrderRequest should place %s", tc.name)
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
		s := testSubmit()
		tc.modify(s)
		_, err := e.newOrderRequest(s)
		assert.ErrorIsf(t, err, tc.err, "newOrderRequest should refuse %s", tc.name)
	}

	// An asset whose pairs have no request format names no instrument
	ex := newTestExchange(t)
	require.NoError(t, ex.CurrencyPairs.Store(asset.PerpetualSwap, &currency.PairStore{AssetEnabled: true}), "Store must store perpetual swaps without a request format")
	_, err = ex.newOrderRequest(testSubmit())
	assert.ErrorIs(t, err, currency.ErrPairFormatIsNil, "newOrderRequest should refuse a pair whose instrument it cannot name")
}

func TestSetIsolationID(t *testing.T) {
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
		routes := make(map[string]testRoute)
		if tc.positions != nil {
			routes["private/get-positions"] = tc.positions
		}
		ex, received := newRoutesServerExchange(t, routes)
		require.NoErrorf(t, ex.setIsolationID(t.Context(), submit(tc.marginType), tc.req), "setIsolationID must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, tc.req, "setIsolationID should name the isolated position expected for %s", tc.name)
		assert.Equalf(t, tc.requests, received(), "setIsolationID should send the position lookups expected for %s", tc.name)
	}

	r := closing()
	err := newUnavailableServerExchange(t).setIsolationID(t.Context(), submit(margin.Isolated), r)
	assert.ErrorIs(t, err, request.ErrBadStatus, "setIsolationID should return the error looking up the isolated position")
	assert.Equal(t, closing(), r, "setIsolationID should leave the order unchanged when it cannot look up the isolated position")
}

func TestReferencePriceType(t *testing.T) {
	t.Parallel()
	for pt, exp := range map[order.PriceType]string{
		order.LastPrice:  ReferencePriceTypeLastPrice,
		order.MarkPrice:  ReferencePriceTypeMarkPrice,
		order.IndexPrice: ReferencePriceTypeIndexPrice,
	} {
		got, err := referencePriceType(pt)
		require.NoErrorf(t, err, "referencePriceType must not error for %s", pt)
		assert.Equalf(t, exp, got, "referencePriceType should map %s", pt)
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

	ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/advanced/create-order": testResult(advancedOrderIDResult)})
	trigger = testTriggerSubmit()
	trigger.ClientOrderID = ""
	got, err = ex.SubmitOrder(t.Context(), trigger)
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
	// Crypto.com opens a new isolated position for an order naming none
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

func TestSubmitOrderTracksMarginOrders(t *testing.T) {
	t.Parallel()
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
	got, err := ex.submitOrder(t.Context(), s, placing("9", nil), nil)
	require.NoError(t, err, "submitOrder must not error for a margin order")
	require.NotEmpty(t, sent.ClientOrderID, "submitOrder must give a margin order without a client order ID one")
	assert.Equal(t, sent.ClientOrderID, got.ClientOrderID, "submitOrder should report the client order ID it gave the order")
	assert.True(t, trackedWhenSent, "submitOrder should track a margin order by its client order ID before placing it, as its trades can arrive first")
	assert.True(t, ex.isMarginOrder("9", ""), "submitOrder should track a placed margin order by its order ID")

	// A spot order reusing the client order ID stops it marking a margin order, which its order ID still marks
	s.AssetType, s.ClientOrderID = asset.Spot, sent.ClientOrderID
	_, err = ex.submitOrder(t.Context(), s, placing("10", nil), nil)
	require.NoError(t, err, "submitOrder must not error for a spot order")
	assert.False(t, trackedWhenSent, "submitOrder should stop tracking a client order ID a spot order reuses before placing it")
	assert.False(t, ex.isMarginOrder("10", s.ClientOrderID), "submitOrder should track a placed spot order as no margin order")
	assert.True(t, ex.isMarginOrder("9", s.ClientOrderID), "the margin order should still be tracked by its order ID")

	// A margin order Crypto.com refuses is no longer tracked by its client order ID
	s.AssetType, s.ClientOrderID = asset.Margin, "refused"
	_, err = ex.submitOrder(t.Context(), s, placing("", &APIError{Code: 306, Message: "INSUFFICIENT_AVAILABLE_BALANCE"}), nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "submitOrder must return Crypto.com's refusal")
	assert.True(t, trackedWhenSent, "submitOrder should track a margin order by its client order ID before placing it")
	assert.False(t, ex.isMarginOrder("", "refused"), "submitOrder should stop tracking the client order ID of an order Crypto.com refused")

	// A margin order its own checks refuse is neither sent nor tracked by its client order ID
	s.ClientOrderID = strings.Repeat("c", 37)
	_, err = ex.submitOrder(t.Context(), s, placing("", nil), nil)
	require.ErrorIs(t, err, errClientOrderIDTooLong, "submitOrder must refuse a client order ID of 37 characters")
	assert.False(t, ex.isMarginOrder("", s.ClientOrderID), "submitOrder should not track the client order ID of an order its checks refuse")

	// A margin order whose request fails otherwise may have been placed, so is still tracked
	s.ClientOrderID = "unknown"
	_, err = ex.submitOrder(t.Context(), s, placing("", context.Canceled), nil)
	require.ErrorIs(t, err, context.Canceled, "submitOrder must return the request's error")
	assert.True(t, ex.isMarginOrder("", "unknown"), "submitOrder should go on tracking the client order ID of an order whose outcome is unknown")
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
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("the orders would be placed live in lists")
	}
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

	// A later list's answer accepting an order without its ID keeps the responses of the lists placed before it too
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
	for _, tc := range []struct {
		name   string
		orders []*order.Submit
		err    error
		named  string
	}{
		{"ten orders then one with a client order ID of 37 characters", marginOrders, errClientOrderIDTooLong, "order 10: "},
		{"an order then a trigger order without a trigger price", []*order.Submit{marginOrders[0], noTriggerPrice}, errInvalidReferencePrice, "order 1: "},
	} {
		_, err = ex.WebsocketSubmitOrders(t.Context(), tc.orders)
		require.ErrorIsf(t, err, tc.err, "WebsocketSubmitOrders must refuse %s", tc.name)
		assert.ErrorContainsf(t, err, tc.named, "WebsocketSubmitOrders should name the order it refuses by its index among %s", tc.name)
		for _, s := range tc.orders {
			assert.Falsef(t, ex.isMarginOrder("", s.ClientOrderID), "WebsocketSubmitOrders should not track client order ID %q of %s, sending none of them", s.ClientOrderID, tc.name)
		}
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
	assert.ErrorIs(t, responses[1].SubmissionError, order.ErrUnableToPlaceOrder, "listSubmitResponses should report an order without an entry")
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

func TestWebsocketSubmitOrdersTracksMarginOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the orders would be placed live in lists")
	}
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
	assert.True(t, ex.isMarginOrder("", "margin_placed"), "WebsocketSubmitOrders should track a placed margin order by its client order ID")
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
	_, err = ex.WebsocketSubmitOrders(done, []*order.Submit{marginOrder("margin_unknown")})
	require.ErrorIs(t, err, context.Canceled, "WebsocketSubmitOrders must return the request's error")
	assert.True(t, ex.isMarginOrder("", "margin_unknown"), "WebsocketSubmitOrders should go on tracking the client order IDs of a list whose outcome is unknown")

	// An order an answered list gives no outcome for may have been placed, so is still tracked
	venue.fail("private/create-order-list", nil)
	venue.respond("private/create-order-list", `[{"code":0,"index":0,"client_oid":"margin_answered","order_id":"31"}]`)
	got, err := ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{marginOrder("margin_answered"), marginOrder("margin_unanswered")})
	require.NoError(t, err, "WebsocketSubmitOrders must not error for a list answered for one of its orders")
	require.Len(t, got, 2, "WebsocketSubmitOrders must return a response for every order")
	assert.ErrorIs(t, got[1].SubmissionError, order.ErrUnableToPlaceOrder, "WebsocketSubmitOrders should report an order the answer gives no outcome for")
	assert.True(t, ex.isMarginOrder("", "margin_unanswered"), "WebsocketSubmitOrders should go on tracking the client order ID of an order the answer gives no outcome for")
}

func TestNewOrderAmendment(t *testing.T) {
	t.Parallel()
	cro := currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter)
	notFound := func(string) (string, *APIError) { return "", &APIError{Code: 40401, Message: "NOT_FOUND"} }
	const limitOrder = `{"order_id":"1","order_type":"LIMIT","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"],"side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD"}`
	orderLookup := []testRequest{{Method: "private/get-order-detail", Params: `{"order_id":"1"}`}}
	advancedLookup := []testRequest{
		{Method: "private/get-order-detail", Params: `{"order_id":"1"}`},
		{Method: "private/advanced/get-order-detail", Params: `{"order_id":"1"}`},
	}
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
		ex, received := newRoutesServerExchange(t, tc.routes)
		got, err := ex.newOrderAmendment(t.Context(), &tc.m)
		require.NoErrorf(t, err, "newOrderAmendment must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "newOrderAmendment should amend %s", tc.name)
		assert.Equalf(t, tc.requests, received(), "newOrderAmendment should send the order lookups expected for %s", tc.name)
	}

	refuser, _ := newRoutesServerExchange(t, nil)
	answering := func(detail string) *Exchange {
		ex, _ := newRoutesServerExchange(t, map[string]testRoute{"private/get-order-detail": testResult(detail)})
		return ex
	}
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
		_, err := tc.ex.newOrderAmendment(t.Context(), tc.m)
		assert.ErrorIsf(t, err, tc.err, "newOrderAmendment should refuse %s", tc.name)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must refuse a nil modification")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{Pair: testPerpetualPair, AssetType: asset.Options, OrderID: "1"})
	require.ErrorIs(t, err, asset.ErrNotSupported, "ModifyOrder must refuse an unsupported asset")
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("the amended order would be placed live")
	}
	m := &order.Modify{Exchange: e.Name, OrderID: tradingTestOrderID, Type: order.Limit, Side: order.Buy, AssetType: asset.PerpetualSwap, Pair: testPerpetualPair, Price: 50000, Amount: 0.0002}
	got, err := e.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error")
	exp := &order.ModifyResponse{Exchange: e.Name, OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID, Pair: testPerpetualPair, Type: order.Limit, Side: order.Buy, AssetType: asset.PerpetualSwap, Price: 50000, Amount: 0.0002}
	assert.Equal(t, exp, got, "ModifyOrder should amend the order's price and quantity")

	cro := currency.NewPairWithDelimiter("CRO", "USD", currency.UnderscoreDelimiter)
	m = &order.Modify{Exchange: e.Name, OrderID: "5755600460443882762", Type: order.StopLimit, Side: order.Sell, AssetType: asset.Spot, Pair: cro, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279}
	got, err = e.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error for a trigger order")
	exp = &order.ModifyResponse{Exchange: e.Name, OrderID: "5755600460443882762", ClientOrderID: tradingTestOrderClientOrderID, Pair: cro, Type: order.StopLimit, Side: order.Sell, AssetType: asset.Spot, Price: 0.0278, Amount: 10, TriggerPrice: 0.0279}
	assert.Equal(t, exp, got, "ModifyOrder should amend a trigger order with the advanced order methods")

	// An OTO list's limit leg, which only the advanced order methods know, has its quantity restated
	ex, received := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail":          func(string) (string, *APIError) { return "", &APIError{Code: 40401, Message: "NOT_FOUND"} },
		"private/advanced/get-order-detail": testResult(`{"order_id":"5755600460443890101","order_type":"LIMIT","side":"BUY","quantity":"0.0100","limit_price":"78000.0","status":"ACTIVE","instrument_name":"BTC_USD"}`),
		"private/advanced/amend-order":      testResult(`{"order_id":"5755600460443890101","client_oid":"1792836351900"}`),
	})
	m = &order.Modify{Exchange: ex.Name, OrderID: "5755600460443890101", AssetType: asset.Spot, Pair: testSpotPair, Price: 77000}
	got, err = ex.ModifyOrder(t.Context(), m)
	require.NoError(t, err, "ModifyOrder must not error for an order list's leg")
	exp = &order.ModifyResponse{Exchange: ex.Name, OrderID: "5755600460443890101", ClientOrderID: "1792836351900", Pair: testSpotPair, Type: order.Limit, AssetType: asset.Spot, Price: 77000, Amount: 0.01}
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
	// reported, and a trigger market order is sent no limit price, which would make it a trigger limit order
	for orderType, exp := range map[string]order.Type{"STOP_LOSS": order.Stop, "TAKE_PROFIT": order.TakeProfit} {
		for _, typ := range []order.Type{order.UnknownType, order.AnyType} {
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
		}
	}

	// An order is amended by its order ID or its client order ID, and takes any new client order ID the modification gives
	for _, tc := range []struct {
		name string
		m    order.Modify
		exp  string
	}{
		{"named by its client order ID", order.Modify{ClientOrderID: "original"}, `{"orig_client_oid":"original","new_price":"77000","new_quantity":"0.01"}`},
		{"named by its client order ID and given a new one", order.Modify{ClientOrderID: "original", NewClientOrderID: "renamed"}, `{"orig_client_oid":"original","client_oid":"renamed","new_price":"77000","new_quantity":"0.01"}`},
		{"named by its order ID and given a client order ID", order.Modify{OrderID: "1", NewClientOrderID: "renamed"}, `{"order_id":"1","client_oid":"renamed","new_price":"77000","new_quantity":"0.01"}`},
	} {
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: testSpotPair, asset.PerpetualSwap: testPerpetualPair} {
			ex, received = newRoutesServerExchange(t, map[string]testRoute{"private/amend-order": testResult(`{"order_id":"1"}`)})
			m := tc.m
			m.Exchange, m.Type, m.AssetType, m.Pair, m.Price, m.Amount = ex.Name, order.Limit, a, p, 77000, 0.01
			_, err = ex.ModifyOrder(t.Context(), &m)
			require.NoErrorf(t, err, "ModifyOrder must not error for a %s order %s", a, tc.name)
			assert.Equalf(t, []testRequest{{Method: "private/amend-order", Params: tc.exp}}, received(), "ModifyOrder should amend a %s order %s", a, tc.name)
		}
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
	if !mockTests {
		t.Skip("the amended order would be placed live")
	}
	ex, venue := newWebsocketTestExchange(t)
	m := &order.Modify{Exchange: ex.Name, OrderID: tradingTestOrderID, Type: order.Limit, Side: order.Buy, AssetType: asset.PerpetualSwap, Pair: testPerpetualPair, Price: 50000, Amount: 0.0002}
	got, err := ex.WebsocketModifyOrder(t.Context(), m)
	require.NoError(t, err, "WebsocketModifyOrder must not error")
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
}

func TestAmendOrderFallback(t *testing.T) {
	t.Parallel()
	notFound := func(string) (string, *APIError) { return "", &APIError{Code: 212, Message: "INVALID_ORDERID"} }
	legAmended := testResult(`{"order_id":"5755600460443890101","client_oid":"leg"}`)
	leg := func() *orderAmendment {
		return &orderAmendment{order: &AmendOrderRequest{OrderID: "5755600460443890101", OriginalClientOrderID: "leg", NewPrice: 77000, NewQuantity: 0.01}}
	}
	amendment := testRequest{Method: "private/amend-order", Params: `{"order_id":"5755600460443890101","orig_client_oid":"leg","new_price":"77000","new_quantity":"0.01"}`}
	advancedAmendment := testRequest{Method: "private/advanced/amend-order", Params: `{"order_id":"5755600460443890101","client_oid":"leg","new_quantity":"0.01","new_price":"77000"}`}
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
		ex, received := newRoutesServerExchange(t, tc.routes)
		got, err := amendOrder(t.Context(), tc.amendment, ex.AmendOrder, ex.AmendAdvancedOrder)
		require.NoErrorf(t, err, "amendOrder must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "amendOrder should return the amended order's IDs for %s", tc.name)
		assert.Equalf(t, tc.requests, received(), "amendOrder should send the amendments expected for %s", tc.name)
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
		ex, received := newRoutesServerExchange(t, tc.routes)
		got, err := amendOrder(t.Context(), tc.amendment, ex.AmendOrder, ex.AmendAdvancedOrder)
		assert.Nilf(t, got, "amendOrder should return no IDs for %s", tc.name)
		var apiErr *APIError
		require.ErrorAsf(t, err, &apiErr, "amendOrder must return the trading methods' refusal of %s", tc.name)
		assert.Equalf(t, tc.apiErr, apiErr, "amendOrder should return Crypto.com's code and message for %s", tc.name)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "amendOrder should return why it did not amend %s", tc.name)
		}
		assert.Equalf(t, tc.requests, received(), "amendOrder should send the amendments expected for %s", tc.name)
	}
}

func TestIsTriggerOrder(t *testing.T) {
	t.Parallel()
	for _, orderType := range []order.Type{order.Stop, order.StopMarket, order.StopLimit, order.TakeProfit, order.TakeProfitMarket, order.TakeProfitLimit} {
		assert.Truef(t, isTriggerOrder(orderType), "isTriggerOrder should report %s a trigger order", orderType)
	}
	for _, orderType := range []order.Type{order.Limit, order.Market, order.UnknownType, order.AnyType, order.TrailingStop} {
		assert.Falsef(t, isTriggerOrder(orderType), "isTriggerOrder should not report %s a trigger order", orderType)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must refuse a nil cancel")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{}), order.ErrOrderIDNotSet, "CancelOrder must refuse a cancel without an order ID")
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		orderID := placeTradingTestOrder(t)
		assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: orderID, Type: order.Limit}), "CancelOrder should cancel the order the test placed")
		return
	}
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: tradingTestOrderID, Type: order.Limit}), "CancelOrder should cancel an order")
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: "5755600460443882762", Type: order.StopLimit}), "CancelOrder should cancel a trigger order with the advanced order methods")

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
	for _, code := range []int64{codeUnauthorised, codeInvalidNonce, codeIPIllegal, codeUserTierInvalid, codeRequestTimeout, codeTooManyRequests, codeInternalError} {
		assert.Falsef(t, refusedForOrder(fmt.Errorf("wrapped: %w", &APIError{Code: code})), "refusedForOrder should not report a request refused with code %d", code)
	}
	for _, code := range []int64{40401, 212, 316} {
		assert.Truef(t, refusedForOrder(fmt.Errorf("wrapped: %w", &APIError{Code: code})), "refusedForOrder should report an order refused with code %d", code)
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

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, errOrderListEmpty, "CancelBatchOrders must refuse no orders")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1"}, {}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must refuse an order without an ID")
	if !mockTests {
		t.Skip("the batch is checked against a test server")
	}
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
		"1": "CANCELLED", "2": "CANCELLED", "3": "CANCELLED", "4": "CANCELLED", "5": "CANCELLED", "6": "CANCELLED",
		"7": "CANCELLED", "8": "CANCELLED", "9": "CANCELLED", "10": "no result for the order", "11": "CANCELLED", "my_sl_order": "CANCELLED",
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
	exp = &order.CancelBatchResponse{Status: map[string]string{"1": "42901 TOO_MANY_REQUESTS", "2": "CANCELLED"}}
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
	assert.Equal(t, "CANCELLED", got.Status["10"], "CancelBatchOrders should report the orders cancelled before a list failed")
	assert.NotContains(t, got.Status, "11", "CancelBatchOrders should report no outcome for an order whose list failed")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must refuse a nil cancel")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must refuse an unsupported asset")
	if !mockTests {
		t.Skip("cancelling every order is checked against a test server")
	}
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
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"1": "CANCELLED", "4": "CANCELLED"}}, got, "CancelAllOrders should cancel the asset's orders only")
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
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"1": "CANCELLED"}}, got, "CancelAllOrders should return the outcomes of the lists sent before one failed")
}

func TestListCancelStatus(t *testing.T) {
	t.Parallel()
	status := make(map[string]string)
	refused := listCancelStatus([]orderCancel{{OrderID: "1"}, {ClientOrderID: "c"}, {OrderID: "3"}}, []OrderListEntryResponse{{Index: 1}, {Index: -1}, {Index: 3}, {Index: 2, Code: 316, Message: "NO_ACTIVE_ORDER"}}, status)
	assert.Equal(t, map[string]string{"1": "no result for the order", "c": "CANCELLED", "3": "316 NO_ACTIVE_ORDER"}, status, "listCancelStatus should key an order by its client order ID without an order ID, and leave out entries out of range")
	assert.Equal(t, []int{2}, refused, "listCancelStatus should return the indexes of the orders Crypto.com refused to cancel")

	// Entries refused for the request rather than their orders are reported, and left out of the orders refused
	for _, code := range []int64{codeUnauthorised, codeInvalidNonce, codeIPIllegal, codeUserTierInvalid, codeRequestTimeout, codeTooManyRequests, codeInternalError} {
		status = make(map[string]string)
		refused = listCancelStatus([]orderCancel{{OrderID: "1"}, {OrderID: "2"}}, []OrderListEntryResponse{{Index: 0, Code: types.Number(code), Message: "REFUSED"}, {Index: 1, Code: 316, Message: "NO_ACTIVE_ORDER"}}, status)
		assert.Equalf(t, map[string]string{"1": strconv.FormatInt(code, 10) + " REFUSED", "2": "316 NO_ACTIVE_ORDER"}, status, "listCancelStatus should report an entry refused with code %d", code)
		assert.Equalf(t, []int{1}, refused, "listCancelStatus should leave out an entry refused with code %d for the request", code)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must refuse an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must refuse an unsupported asset")
	skipLiveWithoutCredentials(t)
	if !mockTests {
		t.Skip("no order is known to look up live")
	}
	got, err := e.GetOrderInfo(t.Context(), "6530219466236720403", testPerpetualPair, asset.PerpetualSwap)
	require.NoError(t, err, "GetOrderInfo must not error")
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
	_, err = e.GetOrderInfo(t.Context(), "6530219466236720403", testSpotPair, asset.Spot)
	assert.ErrorIs(t, err, errOrderAssetMismatch, "GetOrderInfo should refuse an order of another asset")

	ex, _ := newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-detail":          func(string) (string, *APIError) { return "", &APIError{Code: 40401, Message: "NOT_FOUND"} },
		"private/advanced/get-order-detail": testResult(`{"order_id":"5755600460443890102","order_type":"STOP_LOSS","side":"SELL","quantity":"0.0100","ref_price":"74000.00","status":"ACTIVE","instrument_name":"BTC_USD","exec_inst":["MARGIN_ORDER"]}`),
	})
	got, err = ex.GetOrderInfo(t.Context(), "5755600460443890102", currency.EMPTYPAIR, asset.Margin)
	require.NoError(t, err, "GetOrderInfo must not error for a trigger order")
	exp = &order.Detail{Amount: 0.01, TriggerPrice: 74000, RemainingAmount: 0.01, CostAsset: currency.USD, Exchange: ex.Name, OrderID: "5755600460443890102", Type: order.Stop, Side: order.Sell, Status: order.Active, AssetType: asset.Margin, Pair: testSpotPair, MarginType: margin.Multi}
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

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must refuse a nil request")
	skipLiveWithoutCredentials(t)
	ex := e
	var received func() []testRequest
	if mockTests {
		ex, received = newRoutesServerExchange(t, testOpenOrdersRoutes)
	}
	for _, a := range ex.GetAssetTypes(false) {
		_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
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
	ns := func(t time.Time) string { return `"` + strconv.FormatInt(t.UnixNano(), 10) + `"` }
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

	// Without a start time each history is requested for the day before the end, the advanced order methods' trigger
	// orders with the trading methods' orders, which the request's filters choose from
	ex, received = newRoutesServerExchange(t, map[string]testRoute{
		"private/get-order-history":          testResult(`{"data":[{"order_id":"1","order_type":"LIMIT","side":"BUY","quantity":"1","limit_price":"80000","cumulative_quantity":"1","status":"FILLED","instrument_name":"BTC_USD","create_time":"1792634400000","update_time":"1792634400100"}]}`),
		"private/advanced/get-order-history": testResult(`{"data":[{"order_id":"2","order_type":"STOP_LOSS","side":"SELL","quantity":"1","ref_price":"79000","status":"CANCELED","instrument_name":"BTC_USD","create_time":"1792634400200","update_time":"1792634400300"}]}`),
	})
	end = time.UnixMilli(1792634400000).Add(time.Hour)
	got, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, EndTime: end})
	require.NoError(t, err, "GetOrderHistory must not error without a start time")
	orderIDs := make([]string, len(got))
	for i := range got {
		orderIDs[i] = got[i].OrderID
	}
	assert.Equal(t, []string{"1", "2"}, orderIDs, "GetOrderHistory should return the trading methods' orders and the advanced order methods' trigger orders")
	day = end.Add(-privateHistoryWindow)
	expRequests = []testRequest{
		{Method: "private/get-order-history", Params: `{"start_time":` + ns(day) + `,"end_time":` + ns(end) + `,"limit":100}`},
		{Method: "private/advanced/get-order-history", Params: `{"start_time":` + ns(day) + `,"end_time":` + ns(end) + `,"limit":"100"}`},
	}
	requests = received()
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
	for side, exp := range map[order.Side]string{
		order.Buy: SideBuy, order.Bid: SideBuy, order.Long: SideBuy,
		order.Sell: SideSell, order.Ask: SideSell, order.Short: SideSell,
	} {
		got, err := orderSideString(side)
		require.NoErrorf(t, err, "orderSideString must not error for %s", side)
		assert.Equalf(t, exp, got, "orderSideString should map %s", side)
	}
	for _, side := range []order.Side{order.UnknownSide, order.AnySide, order.ClosePosition} {
		_, err := orderSideString(side)
		assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "orderSideString should refuse %s", side)
	}
}

func TestOrderSideFromString(t *testing.T) {
	t.Parallel()
	for s, exp := range map[string]order.Side{"BUY": order.Buy, "buy": order.Buy, "SELL": order.Sell, "sell": order.Sell} {
		got, err := orderSideFromString(s)
		require.NoErrorf(t, err, "orderSideFromString must not error for %s", s)
		assert.Equalf(t, exp, got, "orderSideFromString should map %s", s)
	}
	_, err := orderSideFromString("HOLD")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "orderSideFromString should refuse an unknown side")
}

func TestOrderTypeString(t *testing.T) {
	t.Parallel()
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
		got, trigger, err := orderTypeString(tc.orderType)
		require.NoErrorf(t, err, "orderTypeString must not error for %s", tc.orderType)
		assert.Equalf(t, tc.exp, got, "orderTypeString should map %s", tc.orderType)
		assert.Equalf(t, tc.trigger, trigger, "orderTypeString should tell whether %s is a trigger order", tc.orderType)
	}
	for _, orderType := range []order.Type{order.UnknownType, order.TrailingStop, order.OCO, order.AnyType} {
		_, _, err := orderTypeString(orderType)
		assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "orderTypeString should refuse %s", orderType)
	}
}

func TestOrderTypeFromString(t *testing.T) {
	t.Parallel()
	for s, exp := range map[string]order.Type{
		OrderTypeLimit:           order.Limit,
		OrderTypeMarket:          order.Market,
		OrderTypeStopLoss:        order.Stop,
		OrderTypeStopLimit:       order.StopLimit,
		OrderTypeTakeProfit:      order.TakeProfit,
		OrderTypeTakeProfitLimit: order.TakeProfitLimit,
	} {
		got, err := orderTypeFromString(s)
		require.NoErrorf(t, err, "orderTypeFromString must not error for %s", s)
		assert.Equalf(t, exp, got, "orderTypeFromString should map %s", s)
		back, _, err := orderTypeString(got)
		require.NoErrorf(t, err, "orderTypeString must not error for %s", got)
		assert.Equalf(t, s, back, "orderTypeString should map %s back to %s", got, s)
	}
	_, err := orderTypeFromString("TRAILING_STOP")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "orderTypeFromString should refuse an unknown type")
}

func TestTimeInForceParams(t *testing.T) {
	t.Parallel()
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
		timeInForce, postOnly, err := timeInForceParams(tc.tif)
		require.NoErrorf(t, err, "timeInForceParams must not error for %s", tc.tif)
		assert.Equalf(t, tc.timeInForce, timeInForce, "timeInForceParams should map %s", tc.tif)
		assert.Equalf(t, tc.postOnly, postOnly, "timeInForceParams should tell whether %s is post only", tc.tif)
	}
	for _, tif := range []order.TimeInForce{order.GoodTillDay, order.GoodTillTime, order.GoodTillCrossing, order.GoodTillDay | order.PostOnly} {
		_, _, err := timeInForceParams(tif)
		assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "timeInForceParams should refuse %s", tif)
	}
	for _, tif := range []order.TimeInForce{order.TimeInForce(1 << 15), order.ImmediateOrCancel | order.PostOnly, order.FillOrKill | order.PostOnly, order.GoodTillCancel | order.FillOrKill} {
		_, _, err := timeInForceParams(tif)
		assert.ErrorIsf(t, err, order.ErrInvalidTimeInForce, "timeInForceParams should refuse %s", tif)
	}
}

func TestTimeInForceFromResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		timeInForce string
		execInst    []string
		exp         order.TimeInForce
	}{
		{},
		{timeInForce: TimeInForceGoodTillCancel, exp: order.GoodTillCancel},
		{timeInForce: TimeInForceImmediateOrCancel, exp: order.ImmediateOrCancel},
		{timeInForce: TimeInForceFillOrKill, exp: order.FillOrKill},
		{timeInForce: TimeInForceGoodTillCancel, execInst: []string{ExecutionInstructionPostOnly}, exp: order.GoodTillCancel | order.PostOnly},
		{timeInForce: TimeInForceGoodTillCancel, execInst: []string{ExecutionInstructionReduceOnly, ExecutionInstructionSmartPostOnly}, exp: order.GoodTillCancel | order.PostOnly},
		{timeInForce: TimeInForceGoodTillCancel, execInst: []string{ExecutionInstructionReduceOnly}, exp: order.GoodTillCancel},
	} {
		got, err := timeInForceFromResponse(tc.timeInForce, tc.execInst)
		require.NoErrorf(t, err, "timeInForceFromResponse must not error for %s %v", tc.timeInForce, tc.execInst)
		assert.Equalf(t, tc.exp, got, "timeInForceFromResponse should map %s %v", tc.timeInForce, tc.execInst)
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
		assert.Equalf(t, exp, effectiveTimeInForce(tif), "effectiveTimeInForce should give the time in force Crypto.com places an order with %s", tif)
	}
}

func TestOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status   string
		executed types.PreciseNumber
		exp      order.Status
	}{
		{status: OrderStatusPending, exp: order.Pending},
		{status: OrderStatusNew, exp: order.New},
		{status: OrderStatusActive, exp: order.Active},
		{status: OrderStatusActive, executed: preciseNumber("0.5"), exp: order.PartiallyFilled},
		{status: OrderStatusFilled, executed: preciseNumber("1"), exp: order.Filled},
		{status: OrderStatusCanceled, exp: order.Cancelled},
		{status: OrderStatusCanceled, executed: preciseNumber("0.5"), exp: order.PartiallyCancelled},
		{status: OrderStatusRejected, exp: order.Rejected},
		{status: OrderStatusExpired, exp: order.Expired},
	} {
		got, err := orderStatus(&OrderResponse{Status: tc.status, CumulativeQuantity: tc.executed})
		require.NoErrorf(t, err, "orderStatus must not error for %s", tc.status)
		assert.Equalf(t, tc.exp, got, "orderStatus should map %s with %v executed", tc.status, tc.executed)
	}
	_, err := orderStatus(&OrderResponse{Status: "SUSPENDED"})
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "orderStatus should refuse an unknown status")
}

func TestOrderDetail(t *testing.T) {
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
	// A stop loss or take profit buy is a market buy once triggered, which can spend a notional too
	for orderType, typ := range map[string]order.Type{OrderTypeStopLoss: order.Stop, OrderTypeTakeProfit: order.TakeProfit} {
		trigger := &OrderResponse{OrderID: "2", OrderType: orderType, Side: SideBuy, Status: OrderStatusPending, Quantity: preciseNumber("0"), OrderValue: preciseNumber("7600"), ReferencePrice: preciseNumber("76000")}
		got, err := e.orderDetail(trigger, asset.Spot, testSpotPair)
		require.NoErrorf(t, err, "orderDetail must not error for a %s buy by notional", orderType)
		expTrigger := &order.Detail{Exchange: e.Name, OrderID: "2", Pair: testSpotPair, AssetType: asset.Spot, Side: order.Buy, Type: typ, Status: order.Pending, TriggerPrice: 76000, QuoteAmount: 7600, CostAsset: currency.USD}
		assert.Equalf(t, expTrigger, got, "orderDetail should give a %s buy by notional its notional as the quote amount", orderType)
	}
	// A trigger buy placed by its quantity keeps its quantity, and its order value is no quote amount
	byQuantity := &OrderResponse{OrderID: "3", OrderType: OrderTypeStopLoss, Side: SideBuy, Status: OrderStatusPending, Quantity: preciseNumber("0.1"), OrderValue: preciseNumber("7600"), ReferencePrice: preciseNumber("76000")}
	quantityDetail, quantityErr := e.orderDetail(byQuantity, asset.Spot, testSpotPair)
	require.NoError(t, quantityErr, "orderDetail must not error for a stop loss buy by quantity")
	assert.Equal(t, 0.1, quantityDetail.Amount, "orderDetail should give a stop loss buy by quantity its quantity")
	assert.Zero(t, quantityDetail.QuoteAmount, "orderDetail should give a stop loss buy by quantity no quote amount")

	for _, tc := range []struct {
		o   OrderResponse
		err error
	}{
		{OrderResponse{Side: "HOLD", OrderType: OrderTypeLimit, Status: OrderStatusNew}, order.ErrSideIsInvalid},
		{OrderResponse{Side: SideBuy, OrderType: "TRAILING", Status: OrderStatusNew}, order.ErrUnrecognisedOrderType},
		{OrderResponse{Side: SideBuy, OrderType: OrderTypeLimit, Status: "SUSPENDED"}, order.ErrUnsupportedStatusType},
		{OrderResponse{Side: SideBuy, OrderType: OrderTypeLimit, Status: OrderStatusNew, TimeInForce: "GOOD_TILL_DATE"}, order.ErrInvalidTimeInForce},
	} {
		_, err := e.orderDetail(&tc.o, asset.Spot, testSpotPair)
		assert.ErrorIsf(t, err, tc.err, "orderDetail should refuse %+v", tc.o)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Options, testSpotPair)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must refuse an unsupported asset")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must refuse an empty pair")
	for a, exp := range map[asset.Item]string{
		asset.Spot:          "https://crypto.com/exchange/trade/BTC_USD",
		asset.Margin:        "https://crypto.com/exchange/trade/BTC_USD",
		asset.PerpetualSwap: "https://crypto.com/exchange/trade/BTCUSD-PERP",
		asset.Futures:       "https://crypto.com/exchange/trade/BTCUSD-261030",
	} {
		got, err := e.GetCurrencyTradeURL(t.Context(), a, testInstrumentPairs[a][0])
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		assert.Equalf(t, exp, got, "GetCurrencyTradeURL should return the %s pair's trading page", a)
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must refuse a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must refuse an unsupported futures asset")
	for _, a := range []asset.Item{asset.PerpetualSwap, asset.Futures} {
		got, err := e.GetFuturesContractDetails(t.Context(), a)
		require.NoErrorf(t, err, "GetFuturesContractDetails must not error for %s", a)
		if !mockTests {
			assert.NotEmptyf(t, got, "GetFuturesContractDetails should return %s contracts", a)
			continue
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
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]bool{asset.PerpetualSwap: true, asset.Futures: false, asset.Spot: false} {
		got, err := e.IsPerpetualFutureCurrency(a, testInstrumentPairs[asset.PerpetualSwap][0])
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Equalf(t, exp, got, "IsPerpetualFutureCurrency should tell whether %s is perpetual", a)
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

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateralMode(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCollateralMode must refuse spot")
	for _, a := range []asset.Item{asset.Margin, asset.PerpetualSwap, asset.Futures} {
		got, err := e.GetCollateralMode(t.Context(), a)
		require.NoErrorf(t, err, "GetCollateralMode must not error for %s", a)
		assert.Equalf(t, collateral.MultiMode, got, "GetCollateralMode should margin %s with every collateral eligible currency", a)
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

	got = history()
	err := newUnavailableServerExchange(t).fundingPayments(t.Context(), "BTCUSD-PERP", got)
	assert.ErrorIs(t, err, request.ErrBadStatus, "fundingPayments should return the error requesting the funding transactions")
	assert.Equal(t, history(), got, "fundingPayments should leave the history unchanged when it cannot request the funding transactions")
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
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary must refuse a nil request")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spot, Pair: testSpotPair})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesPositionSummary must refuse a spot pair")
	skipLiveWithoutCredentials(t)
	if !mockTests {
		_, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: testPair(t, asset.PerpetualSwap)})
		if err != nil {
			assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetFuturesPositionSummary should only error for a pair without a position")
		}
		return
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
		Leverage:               numberDecimal(20),
		MarkPrice:              numberDecimal(80819.5),
		CurrentSize:            decimal.MustFromString("-0.1984"),
		ContractSettlementType: futures.Linear,
		AverageOpenPrice:       decimal.MustFromString("-10159.352200").Div(decimal.MustFromString("-0.1984")).Abs(),
		UnrealisedPNL:          decimal.MustFromString("-497.743736"),
		FreeCollateral:         decimal.MustFromString("4721.05898582"),
		TotalCollateral:        decimal.MustFromString("7651.18811483"),
	}
	assert.Equal(t, exp, got, "GetFuturesPositionSummary should summarise the cross margin position with the account's margin and leverage, valued at the mark price")

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
		assert.Equalf(t, exp, isIsolated(id), "isIsolated should tell whether isolation ID %q names an isolated position", id)
	}
}

func TestIsolatedPositionID(t *testing.T) {
	t.Parallel()
	const (
		isolated      = `{"quantity":"0.0002","instrument_name":"BTCUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"19848526","isolation_type":"ISOLATED_MARGIN"}`
		otherIsolated = `{"quantity":"0.01","instrument_name":"ETHUSD-PERP","type":"PERPETUAL_SWAP","isolation_id":"19848530","isolation_type":"ISOLATED_MARGIN"}`
	)
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
		ex, received := newRoutesServerExchange(t, map[string]testRoute{"private/get-positions": tc.positions})
		got, err := ex.isolatedPositionID(t.Context(), "BTCUSD-PERP")
		require.NoErrorf(t, err, "isolatedPositionID must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "isolatedPositionID should return the isolation ID of the instrument's isolated position, if any, for %s", tc.name)
		assert.Equalf(t, []testRequest{{Method: "private/get-positions", Params: `{"instrument_name":"BTCUSD-PERP"}`}}, received(), "isolatedPositionID should request the instrument's positions for %s", tc.name)
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
	skipLiveWithoutCredentials(t)
	got, err := e.GetLeverage(t.Context(), asset.PerpetualSwap, testPair(t, asset.PerpetualSwap), margin.Multi, order.UnknownSide)
	require.NoError(t, err, "GetLeverage must not error")
	if !mockTests {
		assert.Positive(t, got, "GetLeverage should return the account's leverage")
		return
	}
	assert.Equal(t, 20.0, got, "GetLeverage should return the account's leverage for a cross margin position")
	ex, _ := newRoutesServerExchange(t, testPositionRoutes)
	got, err = ex.GetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Isolated, order.UnknownSide)
	require.NoError(t, err, "GetLeverage must not error for an isolated position")
	assert.Equal(t, 2.0, got, "GetLeverage should return an isolated position's leverage")
	_, err = ex.GetLeverage(t.Context(), asset.PerpetualSwap, currency.NewPairWithDelimiter("ETHUSD", "PERP", currency.DashDelimiter), margin.Isolated, order.UnknownSide)
	assert.ErrorIs(t, err, errNoIsolatedPosition, "GetLeverage should error without an isolated position")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Spot, testSpotPair, margin.Multi, 2, order.UnknownSide), futures.ErrNotFuturesAsset, "SetLeverage must refuse a spot pair")
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, 2.5, order.UnknownSide), errLeverageNotWhole, "SetLeverage must refuse an account leverage that is not a whole number")
	if !mockTests {
		t.Skip("setting leverage is checked against a test server")
	}
	routes := maps.Clone(testPositionRoutes)
	routes["private/get-accounts"] = testResult(`{"this_account":{"uuid":"52e7c00f-1324-5a6a-bf6e-de445bde21a5"}}`)
	routes["private/change-account-leverage"] = testResult("")
	routes["private/change-isolated-margin-leverage"] = testResult("")
	ex, received := newRoutesServerExchange(t, routes)
	require.NoError(t, ex.SetLeverage(t.Context(), asset.PerpetualSwap, testPerpetualPair, margin.Multi, 10, order.UnknownSide), "SetLeverage must not error")
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

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must refuse a nil request")
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
		_, err := e.ChangePositionMargin(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "ChangePositionMargin must refuse %s", tc.name)
	}
	// gctcli's float flags parse NaN and infinities, which are refused before any request
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
		_, err := refuser.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: tc.originalMargin, NewAllocatedMargin: tc.allocatedMargin})
		assert.ErrorIsf(t, err, errMarginNotFinite, "ChangePositionMargin should refuse %s", tc.name)
	}
	assert.Empty(t, refused(), "ChangePositionMargin should send no request for a margin that is not a finite number")
	_, err = refuser.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, MarginType: margin.Isolated, OriginalAllocatedMargin: 5.4, NewAllocatedMargin: 8})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangePositionMargin should refuse a request without a pair")
	assert.Empty(t, refused(), "ChangePositionMargin should send no request without a pair")
	if !mockTests {
		t.Skip("changing a position's margin is checked against a test server")
	}
	routes := maps.Clone(testPositionRoutes)
	routes["private/create-isolated-margin-transfer"] = testResult("")
	ex, received := newRoutesServerExchange(t, routes)
	for _, newMargin := range []float64{8, 3.4} {
		got, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualSwap, Pair: testPerpetualPair, MarginType: margin.Isolated, OriginalAllocatedMargin: 5.4, NewAllocatedMargin: newMargin})
		require.NoErrorf(t, err, "ChangePositionMargin must not error for a new margin of %v", newMargin)
		exp := &margin.PositionChangeResponse{Exchange: ex.Name, Pair: testPerpetualPair, Asset: asset.PerpetualSwap, AllocatedMargin: newMargin, MarginType: margin.Isolated}
		assert.Equalf(t, exp, got, "ChangePositionMargin should return the new margin of %v", newMargin)
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
}
