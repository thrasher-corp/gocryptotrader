package kraken

import (
	"cmp"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	ethXBTPair          = currency.NewPairWithDelimiter("ETH", "XBT", currency.UnderscoreDelimiter)
	solUSDPair          = currency.NewPairWithDelimiter("SOL", "USD", currency.UnderscoreDelimiter)
	piXBTUSDPair        = currency.NewPairWithDelimiter("PI", "XBTUSD", currency.UnderscoreDelimiter)
	pfEURUSDPair        = currency.NewPairWithDelimiter("PF", "EURUSD", currency.UnderscoreDelimiter)
	pfAnthropicXUSDPair = currency.NewPairWithDelimiter("PF", "ANTHROPICXUSD", currency.UnderscoreDelimiter)
	fiXBTUSD260828Pair  = currency.NewPairWithDelimiter("FI", "XBTUSD_260828", currency.UnderscoreDelimiter)
)

// recordings holds the responses testdata/http.json records, which some tests serve from their own server
var recordings = sync.OnceValues(func() (*mock.VCRMock, error) {
	b, err := os.ReadFile("testdata/http.json")
	if err != nil {
		return nil, err
	}
	m := new(mock.VCRMock)
	return m, json.Unmarshal(b, m)
})

// recordedResponse returns a copy of the response testdata/http.json records for a GET request, which the caller may
// change
func recordedResponse(tb testing.TB, path, query string) []byte {
	tb.Helper()
	m, err := recordings()
	require.NoError(tb, err, "testdata/http.json must load")
	for _, r := range m.Routes[path][http.MethodGet] {
		if r.QueryString == query {
			return slices.Clone(r.Data)
		}
	}
	require.Failf(tb, "response must be recorded", "no response is recorded for GET %s?%s", path, query)
	return nil
}

// newRouteTestExchange returns an exchange named after the test whose REST requests are answered with the body routes
// holds for their path, which newHTTPTestExchange prefixes with /derivatives for the v3 API and /api for the history
// and charts APIs
func newRouteTestExchange(t *testing.T, routes map[string][]byte) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !assert.Truef(t, ok, "%s should not be requested", r.URL.Path) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, err := w.Write(body)
		assert.NoError(t, err, "Write should not error")
	})
}

// newErrorReplyTestExchange returns an exchange named after the test whose every REST request is answered with an
// error, as each API reports one: Spot REST in its error array, the v3 API with an error result, the history API with
// a status and reason, and the charts API with a plain text error status
func newErrorReplyTestExchange(t *testing.T) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		status, body := http.StatusBadRequest, "Invalid resolution"
		switch {
		case strings.HasPrefix(r.URL.Path, "/0/"):
			status, body = http.StatusOK, `{"error":["EService:Unavailable"]}`
		case strings.HasPrefix(r.URL.Path, "/derivatives/"):
			status, body = http.StatusOK, `{"result":"error","serverTime":"2026-10-09T02:29:09.686Z","error":"apiLimitExceeded"}`
		case strings.HasPrefix(r.URL.Path, "/api/history/"):
			body = `{"status":"bad_request","reason":"The request could not be understood by the server due malformed syntax."}`
		}
		w.WriteHeader(status)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	})
}

// newMarginTestExchange returns a test exchange that also stores margin pairs, an asset the wrapper does not support,
// so a margin request passes the pair manager's checks and reaches the wrapper's own
func newMarginTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newTestExchange(t)
	require.NoError(t, ex.SetAssetPairStore(asset.Margin, currency.PairStore{
		AssetEnabled:  true,
		Enabled:       currency.Pairs{spotTestPair},
		Available:     currency.Pairs{spotTestPair},
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	}), "SetAssetPairStore must not error")
	return ex
}

// listedAndExpiredInstruments returns the recorded instruments with the recorded FI_XBTUSD_260828 among them, as a
// listing taken while that contract expired would hold it
func listedAndExpiredInstruments(t *testing.T) []byte {
	t.Helper()
	var listed, expired struct {
		Instruments []json.RawMessage `json:"instruments"`
		Result      string            `json:"result"`
		ServerTime  string            `json:"serverTime"`
	}
	require.NoError(t, json.Unmarshal(recordedResponse(t, "/api/v3/instruments", ""), &listed), "Unmarshal must not error for the listed instruments")
	require.NoError(t, json.Unmarshal(recordedResponse(t, "/api/v3/instruments", "contractType=futures_inverse&expired=true"), &expired), "Unmarshal must not error for the expired instruments")
	listed.Instruments = append(listed.Instruments, expired.Instruments...)
	b, err := json.Marshal(&listed)
	require.NoError(t, err, "Marshal must not error")
	return b
}

// sortedPairs returns pairs sorted by their string, for pairs collected from a map
func sortedPairs(pairs currency.Pairs) currency.Pairs {
	return slices.SortedFunc(slices.Values(pairs), func(a, b currency.Pair) int { return strings.Compare(a.String(), b.String()) })
}

func TestBootstrap(t *testing.T) {
	t.Parallel()
	continueBootstrap, err := newErrorReplyTestExchange(t).Bootstrap(t.Context())
	assert.ErrorIs(t, err, errAPIResponse, "Bootstrap should return the error seeding the asset names")
	assert.True(t, continueBootstrap, "Bootstrap should not stop bootstrapping itself when seeding fails")

	ex := newTestExchange(t)
	continueBootstrap, err = ex.Bootstrap(t.Context())
	require.NoError(t, err, "Bootstrap must not error")
	assert.True(t, continueBootstrap, "Bootstrap should continue bootstrapping")
	if !mockTests {
		assert.True(t, ex.assetNames.seeded(), "Bootstrap should seed the asset names")
		return
	}
	exp := &assetNames{
		alternative: map[string]string{
			"XBT.M": "XBT.M", "XETH": "ETH", "XXBT": "XBT", "XXDG": "XDG", "ZUSD": "USD",
			"BTC": "XBT", "BTC.M": "XBT.M", "DOGE": "XDG", "ETH": "ETH", "USD": "USD",
		},
		display: map[string]string{"XBT": "BTC", "XBT.M": "BTC.M", "XDG": "DOGE"},
	}
	assert.Equal(t, exp, &ex.assetNames, "Bootstrap should seed the internal and display names of every asset")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Margin), asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an unsupported asset")
	errEx := newErrorReplyTestExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		assert.ErrorIsf(t, errEx.UpdateOrderExecutionLimits(t.Context(), a), errAPIResponse, "UpdateOrderExecutionLimits should return the %s request's error", a)
	}

	t.Run("expired futures", func(t *testing.T) {
		t.Parallel()
		expiredEx := newRouteTestExchange(t, map[string][]byte{"/derivatives/api/v3/instruments": listedAndExpiredInstruments(t)})
		require.NoError(t, expiredEx.UpdateOrderExecutionLimits(t.Context(), asset.Futures), "UpdateOrderExecutionLimits must not error")
		_, err := expiredEx.GetOrderExecutionLimits(asset.Futures, piXBTUSDPair)
		require.NoError(t, err, "UpdateOrderExecutionLimits must load a listed contract's limits")
		_, err = expiredEx.GetOrderExecutionLimits(asset.Futures, fiXBTUSD260828Pair)
		assert.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "UpdateOrderExecutionLimits should not load an expired contract's limits")
	})

	if !mockTests {
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTestPair, asset.Futures: futuresTestPair} {
			require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
			l, err := ex.GetOrderExecutionLimits(a, p)
			require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", a)
			assert.Positivef(t, l.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load the %s tick size", a)
		}
		return
	}

	// Every listed pair's limits are loaded, whatever its status
	spotKey := func(base, quote currency.Code) key.ExchangeAssetPair {
		return key.NewExchangeAssetPair(ex.Name, asset.Spot, currency.NewPair(base, quote))
	}
	futuresKey := func(p currency.Pair) key.ExchangeAssetPair {
		return key.NewExchangeAssetPair(ex.Name, asset.Futures, p)
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []limits.MinMaxLevel
	}{
		{
			a: asset.Spot,
			exp: []limits.MinMaxLevel{
				{Key: spotKey(currency.XBT, currency.USD), PriceStepIncrementSize: 0.1, MinimumBaseAmount: 0.00005, MinimumQuoteAmount: 0.5, AmountStepIncrementSize: 1e-8, QuoteStepIncrementSize: 1e-5},
				{Key: spotKey(currency.XDG, currency.USD), PriceStepIncrementSize: 1e-7, MinimumBaseAmount: 50, MinimumQuoteAmount: 0.5, AmountStepIncrementSize: 1e-8, QuoteStepIncrementSize: 1e-9},
				{Key: spotKey(currency.ETH, currency.XBT), PriceStepIncrementSize: 1e-6, MinimumBaseAmount: 0.001, MinimumQuoteAmount: 0.00002, AmountStepIncrementSize: 1e-8, QuoteStepIncrementSize: 1e-10},
				{Key: spotKey(currency.ACA, currency.USD), PriceStepIncrementSize: 1e-5, MinimumBaseAmount: 12000, MinimumQuoteAmount: 0.5, AmountStepIncrementSize: 1e-8, QuoteStepIncrementSize: 1e-5},
				{Key: spotKey(currency.NewCode("AIO"), currency.EUR), PriceStepIncrementSize: 1e-5, MinimumBaseAmount: 130, MinimumQuoteAmount: 0.45, AmountStepIncrementSize: 1e-5, QuoteStepIncrementSize: 1e-5},
			},
		},
		{
			// A contract's precision is its minimum and step size, and a negative precision a power of ten
			a: asset.Futures,
			exp: []limits.MinMaxLevel{
				{Key: futuresKey(pfAnthropicXUSDPair), PriceStepIncrementSize: 0.01, MinimumBaseAmount: 0.01, AmountStepIncrementSize: 0.01, Listed: time.Date(2026, 6, 15, 11, 21, 55, 0, time.UTC)},
				{Key: futuresKey(piXBTUSDPair), PriceStepIncrementSize: 0.5, MinimumBaseAmount: 1, AmountStepIncrementSize: 1, Listed: time.Date(2018, 8, 31, 0, 0, 0, 0, time.UTC)},
				{Key: futuresKey(pfEURUSDPair), PriceStepIncrementSize: 1e-5, MinimumBaseAmount: 10, AmountStepIncrementSize: 10, Listed: time.Date(2025, 4, 17, 12, 59, 47, 0, time.UTC)},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.a), "UpdateOrderExecutionLimits must not error")
			loaded := make([]limits.MinMaxLevel, len(tc.exp))
			for i := range tc.exp {
				l, err := limits.GetOrderExecutionLimits(tc.exp[i].Key)
				require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", tc.exp[i].Key.Pair())
				assert.WithinRangef(t, l.UpdatedAt, started, time.Now(), "UpdateOrderExecutionLimits should stamp the %s limits when loading them", tc.exp[i].Key.Pair())
				l.UpdatedAt = time.Time{}
				loaded[i] = l
			}
			assert.Equal(t, tc.exp, loaded, "UpdateOrderExecutionLimits should load every pair's limits")
		})
	}
}

func TestSpotPairFromWebsocketName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		exp  currency.Pair
		ok   bool
	}{
		{name: "XBT/USD", exp: currency.NewPair(currency.XBT, currency.USD), ok: true},
		{name: "XDG/USD", exp: currency.NewPair(currency.XDG, currency.USD), ok: true},
		{name: "ETH/XBT", exp: currency.NewPair(currency.ETH, currency.XBT), ok: true},
		{name: "XBTUSD"},
		{name: "/USD"},
		{name: "XBT/"},
		{name: ""},
	} {
		pair, ok := spotPairFromWebsocketName(tc.name)
		assert.Equalf(t, tc.ok, ok, "spotPairFromWebsocketName should report whether %q names a pair", tc.name)
		assert.Equalf(t, tc.exp, pair, "spotPairFromWebsocketName should return the pair %q names", tc.name)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an unsupported asset")
	errEx := newErrorReplyTestExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err = errEx.FetchTradablePairs(t.Context(), a)
		assert.ErrorIsf(t, err, errAPIResponse, "FetchTradablePairs should return the %s request's error", a)
	}

	for _, tc := range []struct {
		a   asset.Item
		exp currency.Pairs
	}{
		// Spot pairs are named as their websocket names name them, and those whose status is not online are left out
		{a: asset.Spot, exp: currency.Pairs{currency.NewPair(currency.ETH, currency.XBT), currency.NewPair(currency.XBT, currency.USD), currency.NewPair(currency.XDG, currency.USD)}},
		{a: asset.Futures, exp: currency.Pairs{pfAnthropicXUSDPair, pfEURUSDPair, piXBTUSDPair}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.FetchTradablePairs(t.Context(), tc.a)
			require.NoError(t, err, "FetchTradablePairs must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "FetchTradablePairs should return pairs")
				return
			}
			assert.Equal(t, tc.exp, sortedPairs(result), "FetchTradablePairs should return every tradable pair")
		})
	}

	t.Run("expired futures", func(t *testing.T) {
		t.Parallel()
		ex := newRouteTestExchange(t, map[string][]byte{"/derivatives/api/v3/instruments": listedAndExpiredInstruments(t)})
		result, err := ex.FetchTradablePairs(t.Context(), asset.Futures)
		require.NoError(t, err, "FetchTradablePairs must not error")
		assert.Equal(t, currency.Pairs{pfAnthropicXUSDPair, piXBTUSDPair, pfEURUSDPair}, result, "FetchTradablePairs should leave out an expired contract")
	})
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, newErrorReplyTestExchange(t).UpdateTradablePairs(t.Context()), errAPIResponse, "UpdateTradablePairs should return the request's error")

	t.Run("spot in maintenance", func(t *testing.T) {
		t.Parallel()
		// During maintenance no spot pair is online, which leaves no spot pairs to update
		var assetPairs struct {
			Error  []string                   `json:"error"`
			Result map[string]json.RawMessage `json:"result"`
		}
		require.NoError(t, json.Unmarshal(recordedResponse(t, "/0/public/AssetPairs", ""), &assetPairs), "Unmarshal must not error")
		for name, info := range assetPairs.Result {
			if status, err := jsonparser.GetString(info, "status"); err == nil && status == "online" {
				delete(assetPairs.Result, name)
			}
		}
		offline, err := json.Marshal(&assetPairs)
		require.NoError(t, err, "Marshal must not error")
		ex := newRouteTestExchange(t, map[string][]byte{
			"/0/public/AssetPairs":            offline,
			"/derivatives/api/v3/instruments": recordedResponse(t, "/api/v3/instruments", ""),
		})
		err = ex.UpdateTradablePairs(t.Context())
		assert.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "UpdateTradablePairs should report that no spot pair is tradable")
		pairs, err := ex.GetAvailablePairs(asset.Futures)
		require.NoError(t, err, "GetAvailablePairs must not error")
		assert.Equal(t, currency.Pairs{pfAnthropicXUSDPair, pfEURUSDPair, piXBTUSDPair}, sortedPairs(pairs), "UpdateTradablePairs should still update the futures pairs")
	})

	ex := newTestExchange(t)
	if !mockTests {
		require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
		for _, a := range ex.GetAssetTypes(false) {
			available, err := ex.GetAvailablePairs(a)
			require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
			assert.NotEmptyf(t, available, "UpdateTradablePairs should store %s pairs", a)
			enabled, err := ex.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
			assert.NotEmptyf(t, enabled, "UpdateTradablePairs should leave a %s pair enabled", a)
		}
		return
	}
	// PF_XBTUSD is not among the recorded instruments, so it is disabled while PI_XBTUSD stays enabled
	require.NoError(t, ex.CurrencyPairs.EnablePair(asset.Futures, piXBTUSDPair), "EnablePair must not error")
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	type pairs struct {
		available, enabled currency.Pairs
	}
	exp := map[asset.Item]pairs{
		asset.Spot:    {available: currency.Pairs{ethXBTPair, wsTestPair, wsTestDOGEPair}, enabled: currency.Pairs{wsTestPair}},
		asset.Futures: {available: currency.Pairs{pfAnthropicXUSDPair, pfEURUSDPair, piXBTUSDPair}, enabled: currency.Pairs{piXBTUSDPair}},
	}
	stored := make(map[asset.Item]pairs, len(exp))
	for a := range exp {
		available, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		enabled, err := ex.GetEnabledPairs(a)
		require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
		stored[a] = pairs{available: sortedPairs(available), enabled: sortedPairs(enabled)}
	}
	assert.Equal(t, exp, stored, "UpdateTradablePairs should store each asset's tradable pairs, keeping the enabled pairs still listed")
}

// spotTickers returns the spot tickers the recorded tickers update for the test configuration's available pairs
func spotTickers(exchangeName string) map[currency.Pair]*ticker.Price {
	return map[currency.Pair]*ticker.Price{
		wsTestPair: {
			Last: 82052.5, LastSize: 0.05117595, VolumeWeightedAveragePrice: 81847.63078, High: 83234.8, Low: 80328.6,
			Bid: 82052.4, BidSize: 1, Ask: 82052.5, AskSize: 3, BaseVolume: 4071.87007428, Open: 81683.8,
			Pair: wsTestPair, ExchangeName: exchangeName, AssetType: asset.Spot,
		},
		wsTestDOGEPair: {
			Last: 0.0848564, LastSize: 2984.51301529, VolumeWeightedAveragePrice: 0.08513219, High: 0.0892476, Low: 0.0810467,
			Bid: 0.0848346, BidSize: 1688, Ask: 0.0848406, AskSize: 13040, BaseVolume: 126427370.85487261, Open: 0.0839854,
			Pair: wsTestDOGEPair, ExchangeName: exchangeName, AssetType: asset.Spot,
		},
		ethXBTPair: {
			Last: 0.030282, LastSize: 0.01194053, VolumeWeightedAveragePrice: 0.030249, High: 0.031068, Low: 0.02981,
			Bid: 0.030276, BidSize: 1, Ask: 0.030277, AskSize: 3, BaseVolume: 994.3151442, Open: 0.030291,
			Pair: ethXBTPair, ExchangeName: exchangeName, AssetType: asset.Spot,
		},
		solUSDPair: {
			Last: 109.79, LastSize: 0.03102377, VolumeWeightedAveragePrice: 110.17342, High: 116.67, Low: 105.57,
			Bid: 109.78, BidSize: 10, Ask: 109.79, AskSize: 135, BaseVolume: 692573.12208018, Open: 109.52,
			Pair: solUSDPair, ExchangeName: exchangeName, AssetType: asset.Spot,
		},
	}
}

// futuresTicker returns the PF_XBTUSD ticker the recorded futures tickers update
func futuresTicker(exchangeName string) *ticker.Price {
	return &ticker.Price{
		Last: 81783, LastSize: 0.0099, VolumeWeightedAveragePrice: 81911.2221714, High: 83487, Low: 80320,
		Bid: 81782, BidSize: 0.0225, Ask: 81784, AskSize: 0.0873, BaseVolume: 8076.6192, QuoteVolume: 661565749.685,
		Open: 83241, PercentChange24Hour: -1.75, OpenInterest: 2357.1126, MarkPrice: 81780.434623335, IndexPrice: 81770.17,
		Pair: futuresTestPair, ExchangeName: exchangeName, AssetType: asset.Futures,
		LastUpdated: time.Date(2026, 10, 9, 0, 34, 6, 429728331, time.UTC),
	}
}

// storedTickers returns the tickers an exchange stored for an asset, sorted by pair
func storedTickers(t *testing.T, exchangeName string, a asset.Item) []*ticker.Price {
	t.Helper()
	tickers, err := ticker.GetExchangeTickers(exchangeName)
	require.NoError(t, err, "GetExchangeTickers must not error")
	tickers = slices.DeleteFunc(tickers, func(p *ticker.Price) bool { return p.AssetType != a })
	slices.SortFunc(tickers, func(a, b *ticker.Price) int { return strings.Compare(a.Pair.String(), b.Pair.String()) })
	return tickers
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateTickers(t.Context(), asset.Margin), asset.ErrNotSupported, "UpdateTickers must reject an unsupported asset")
	errEx := newErrorReplyTestExchange(t)
	assert.ErrorIs(t, errEx.UpdateTickers(t.Context(), asset.Spot), errAPIResponse, "UpdateTickers should return the error seeding the asset names")
	assert.ErrorIs(t, errEx.UpdateTickers(t.Context(), asset.Futures), errAPIResponse, "UpdateTickers should return the futures request's error")
	seedTestAssetNames(errEx)
	assert.ErrorIs(t, errEx.UpdateTickers(t.Context(), asset.Spot), errAPIResponse, "UpdateTickers should return the spot request's error")

	ex := newTestExchange(t)
	spot := spotTickers(ex.Name)
	for _, tc := range []struct {
		a   asset.Item
		exp []*ticker.Price
	}{
		// The tickers of the Bitnomial venue's BTC/USD:BTNL and of AI/USD, which is not an available pair, are left out
		{a: asset.Spot, exp: []*ticker.Price{spot[ethXBTPair], spot[solUSDPair], spot[wsTestPair], spot[wsTestDOGEPair]}},
		// FF_XBTUSD_261225's ticker is left out, as it is not an available pair
		{a: asset.Futures, exp: []*ticker.Price{futuresTicker(ex.Name)}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			require.NoError(t, ex.UpdateTickers(t.Context(), tc.a), "UpdateTickers must not error")
			result := storedTickers(t, ex.Name, tc.a)
			if !mockTests {
				assert.NotEmpty(t, result, "UpdateTickers should store tickers")
				return
			}
			if tc.a == asset.Spot {
				// Spot tickers carry no time, so they are stamped when stored
				for _, p := range result {
					assert.WithinRangef(t, p.LastUpdated, started, time.Now(), "UpdateTickers should stamp the %s ticker when storing it", p.Pair)
					p.LastUpdated = time.Time{}
				}
			}
			assert.Equal(t, tc.exp, result, "UpdateTickers should store every field of the available pairs' tickers")
		})
	}

	t.Run("crossed books", func(t *testing.T) {
		t.Parallel()
		// As Kraken served some books on 25 September 2026, the ETH/BTC and SOL/USD asks are below their bids
		tickers := recordedResponse(t, "/0/public/Ticker", "assetVersion=1")
		var err error
		for symbol, ask := range map[string]string{"ETH/BTC": `"0.030270"`, "SOL/USD": `"109.70000"`} {
			tickers, err = jsonparser.Set(tickers, []byte(ask), "result", symbol, "a", "[0]")
			require.NoErrorf(t, err, "Set must not error for %s", symbol)
		}
		crossedEx := newRouteTestExchange(t, map[string][]byte{"/0/public/Ticker": tickers})
		seedTestAssetNames(crossedEx)
		started := time.Now()
		err = crossedEx.UpdateTickers(t.Context(), asset.Spot)
		for _, p := range []currency.Pair{ethXBTPair, solUSDPair} {
			assert.ErrorContainsf(t, err, p.String(), "UpdateTickers should report the %s ticker it rejects", p)
		}
		result := storedTickers(t, crossedEx.Name, asset.Spot)
		for _, p := range result {
			assert.WithinRangef(t, p.LastUpdated, started, time.Now(), "UpdateTickers should stamp the %s ticker when storing it", p.Pair)
			p.LastUpdated = time.Time{}
		}
		stored := spotTickers(crossedEx.Name)
		assert.Equal(t, []*ticker.Price{stored[wsTestPair], stored[wsTestDOGEPair]}, result, "UpdateTickers should store every ticker it does not reject")
	})
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), spotTestPair, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an unsupported asset")
	_, err = ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Futures)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must reject an empty pair")
	_, err = newErrorReplyTestExchange(t).UpdateTicker(t.Context(), futuresTestPair, asset.Futures)
	assert.ErrorIs(t, err, errAPIResponse, "UpdateTicker should return the request's error")
	_, err = ex.UpdateTicker(t.Context(), currency.NewPair(currency.NewCode("AI"), currency.USD), asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTicker should not find the ticker of a pair that is not available")

	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  *ticker.Price
	}{
		{a: asset.Spot, pair: spotTestPair, exp: spotTickers(ex.Name)[wsTestPair]},
		{a: asset.Futures, pair: futuresTestPair, exp: futuresTicker(ex.Name)},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := ex.UpdateTicker(t.Context(), tc.pair, tc.a)
			require.NoError(t, err, "UpdateTicker must not error")
			if !mockTests {
				assert.Positive(t, result.Last, "UpdateTicker should return the last price")
				return
			}
			if tc.a == asset.Spot {
				assert.WithinRange(t, result.LastUpdated, started, time.Now(), "UpdateTicker should stamp a spot ticker when storing it")
				result.LastUpdated = time.Time{}
			}
			assert.Equal(t, tc.exp, result, "UpdateTicker should return every field of the ticker")
		})
	}
}

func TestFuturesTickerVolumes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		ticker            FuturesTicker
		expBase, expQuote float64
	}{
		{FuturesTicker{Symbol: "PF_XBTUSD", Volume24Hour: 8076.6192, QuoteVolume24Hour: 661565749.685}, 8076.6192, 661565749.685},
		{FuturesTicker{Symbol: "FF_XBTUSD_261225", Volume24Hour: 48.2406, QuoteVolume24Hour: 3976515.2019}, 48.2406, 3976515.2019},
		// An inverse contract is worth one unit of its quote currency, so its vol24h counts quote currency
		{FuturesTicker{Symbol: "PI_XBTUSD", Volume24Hour: 596069, QuoteVolume24Hour: 596102}, 0, 596102},
		{FuturesTicker{Symbol: "PI_ETHUSD", Volume24Hour: 93853, QuoteVolume24Hour: 93853}, 0, 93853},
	} {
		base, quote := futuresTickerVolumes(tc.ticker.Symbol, tc.ticker.Volume24Hour, tc.ticker.QuoteVolume24Hour)
		assert.Equalf(t, [2]float64{tc.expBase, tc.expQuote}, [2]float64{base, quote}, "futuresTickerVolumes should map %s's base and quote volumes", tc.ticker.Symbol)
	}
}

func TestInverseFuturesSymbol(t *testing.T) {
	t.Parallel()
	for symbol, exp := range map[string]bool{
		"PI_XBTUSD":        true,
		"pi_ethusd":        true,
		"FI_XBTUSD_261030": true,
		"PF_XBTUSD":        false,
		"FF_XBTUSD_261225": false,
		"PF_PIUSD":         false,
		"in_xbtusd":        false,
		"":                 false,
	} {
		assert.Equalf(t, exp, inverseFuturesSymbol(symbol), "inverseFuturesSymbol should report whether %q is an inverse contract", symbol)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must reject an empty pair")
	_, err = ex.UpdateOrderbook(t.Context(), spotTestPair, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an asset without pairs")
	_, err = newMarginTestExchange(t).UpdateOrderbook(t.Context(), spotTestPair, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an unsupported asset")
	disabled := newTestExchange(t)
	require.NoError(t, disabled.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err = disabled.UpdateOrderbook(t.Context(), futuresTestPair, asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotEnabled, "UpdateOrderbook must reject a disabled asset")
	errEx := newErrorReplyTestExchange(t)
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTestPair, asset.Futures: futuresTestPair} {
		_, err = errEx.UpdateOrderbook(t.Context(), p, a)
		assert.ErrorIsf(t, err, errAPIResponse, "UpdateOrderbook should return the %s request's error", a)
	}

	for _, tc := range []struct {
		a   asset.Item
		exp *orderbook.Book
	}{
		{
			a: asset.Spot,
			exp: &orderbook.Book{
				Bids: orderbook.Levels{{Price: 81854.5, Amount: 0.04}, {Price: 81854.3, Amount: 0.001}, {Price: 81852.3, Amount: 0.001}},
				Asks: orderbook.Levels{{Price: 81854.6, Amount: 2.209}, {Price: 81854.9, Amount: 0.025}, {Price: 81855.4, Amount: 0.118}},
				Pair: spotTestPair,
				// A level's timestamp is when it last changed, so the latest is when the book last changed
				LastUpdated: time.Unix(1791513068, 0),
			},
		},
		{
			a: asset.Futures,
			exp: &orderbook.Book{
				// Kraken sends the bids in ascending price order
				Bids:        orderbook.Levels{{Price: 81783, Amount: 0.1323}, {Price: 81782, Amount: 0.0413}, {Price: 81781, Amount: 0.5012}},
				Asks:        orderbook.Levels{{Price: 81784, Amount: 0.0001}, {Price: 81787, Amount: 0.0267}, {Price: 81788, Amount: 0.0079}},
				Pair:        futuresTestPair,
				LastUpdated: time.Date(2026, 10, 9, 0, 34, 10, 410000000, time.UTC),
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := ex.UpdateOrderbook(t.Context(), tc.exp.Pair, tc.a)
			require.NoError(t, err, "UpdateOrderbook must not error")
			if !mockTests {
				assert.NotEmpty(t, result.Bids, "UpdateOrderbook should return bids")
				assert.NotEmpty(t, result.Asks, "UpdateOrderbook should return asks")
				return
			}
			assert.WithinRange(t, result.InsertedAt, started, time.Now(), "UpdateOrderbook should insert the book now")
			tc.exp.InsertedAt = result.InsertedAt
			tc.exp.Exchange, tc.exp.Asset, tc.exp.ValidateOrderbook, tc.exp.RestSnapshot = ex.Name, tc.a, true, true
			assert.Equal(t, tc.exp, result, "UpdateOrderbook should return every level of the book")
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), spotTestPair, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an unsupported asset")
	errEx := newErrorReplyTestExchange(t)
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTestPair, asset.Futures: futuresTestPair} {
		_, err = errEx.GetRecentTrades(t.Context(), p, a)
		assert.ErrorIsf(t, err, errAPIResponse, "GetRecentTrades should return the %s request's error", a)
	}

	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  []trade.Data
	}{
		{
			a:    asset.Spot,
			pair: spotTestPair,
			exp: []trade.Data{
				{TID: "110774605", Side: order.Buy, Price: 81854.6, Amount: 0.01158006, Timestamp: time.Unix(0, 1791513068908400500)},
				{TID: "110774606", Side: order.Buy, Price: 81854.6, Amount: 0.00022645, Timestamp: time.Unix(0, 1791513070181931300)},
				{TID: "110774607", Side: order.Buy, Price: 81854.6, Amount: 0.00043336, Timestamp: time.Unix(0, 1791513070542724100)},
			},
		},
		{
			// A liquidation is a trade too
			a:    asset.Futures,
			pair: futuresTestPair,
			exp: []trade.Data{
				{TID: "2a615649-9ca2-4239-99fa-5c6f4a3186f2", Side: order.Buy, Price: 81830, Amount: 0.8988, Timestamp: time.Date(2026, 10, 9, 0, 33, 31, 277047879, time.UTC)},
				{TID: "6469aaa4-09a7-43ab-bc48-29e0021df4d9", Side: order.Sell, Price: 81831, Amount: 0.0357, Timestamp: time.Date(2026, 10, 9, 0, 33, 31, 283564147, time.UTC)},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRecentTrades(t.Context(), tc.pair, tc.a)
			require.NoError(t, err, "GetRecentTrades must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetRecentTrades should return trades")
				return
			}
			for i := range tc.exp {
				tc.exp[i].Exchange, tc.exp[i].CurrencyPair, tc.exp[i].AssetType = e.Name, tc.pair, tc.a
			}
			assert.Equal(t, tc.exp, result, "GetRecentTrades should return every trade")
		})
	}
}

// futuresTestExecutions returns the recorded PF_XBTUSD trades of 2026-10-09 00:00:00 UTC, each executed at 455ms
func futuresTestExecutions(exchangeName string) []trade.Data {
	executions := []trade.Data{
		{TID: "9cef6672-8ab8-4d34-8aed-0fd45a9c9851", Price: 81689, Amount: 0.0001},
		{TID: "c75bf7bf-c56e-4434-b1f8-809dfdae866b", Price: 81689, Amount: 0.0002},
		{TID: "e713e9a5-45fe-4668-9631-3b6e0dad20bc", Price: 81689, Amount: 0.0001},
		{TID: "34328430-c402-427f-a339-37c452655b90", Price: 81689, Amount: 0.0001},
		{TID: "81428c1d-82fa-4c9b-9135-668bda674df3", Price: 81689, Amount: 0.0001},
		{TID: "d44890bf-1c7d-4911-8ebd-a618c9f399e2", Price: 81689, Amount: 0.0001},
		{TID: "7a953137-ddab-47f6-94ac-82ffc2eb8bba", Price: 81690, Amount: 0.0002},
		{TID: "fef1801e-6278-40be-bd5f-c79621b65e9d", Price: 81690, Amount: 0.0001},
	}
	for i := range executions {
		executions[i].Exchange, executions[i].CurrencyPair, executions[i].AssetType = exchangeName, futuresTestPair, asset.Futures
		executions[i].Side, executions[i].Timestamp = order.Buy, time.UnixMilli(1791504000455)
	}
	return executions
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	now := time.Now()
	_, err := e.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, time.Time{}, now)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetHistoricTrades must reject a window without a start")
	_, err = e.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, now, now.Add(-time.Minute))
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades must reject a window ending before it starts")
	_, err = e.GetHistoricTrades(t.Context(), spotTestPair, asset.Margin, now.Add(-time.Minute), now)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades must reject an unsupported asset")
	errEx := newErrorReplyTestExchange(t)
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTestPair, asset.Futures: futuresTestPair} {
		_, err = errEx.GetHistoricTrades(t.Context(), p, a, now.Add(-time.Minute), now)
		assert.ErrorIsf(t, err, errAPIResponse, "GetHistoricTrades should return the %s request's error", a)
	}

	t.Run("futures token repeated", func(t *testing.T) {
		t.Parallel()
		const token = "MTc5MTUwNDAwMDQ1NS82MTcyNzE4Nzc0Njk="
		path := "/history/v3/market/PF_XBTUSD/executions"
		firstPage := recordedResponse(t, path, "before=1791504001000&since=1791504000000&sort=asc")
		// A page holding the token it was requested with would be requested again forever
		repeatedPage, err := jsonparser.Set(recordedResponse(t, path, "before=1791504001000&continuation_token="+token+"&since=1791504000000&sort=asc"), []byte(`"`+token+`"`), "continuationToken")
		require.NoError(t, err, "Set must not error")
		var requests atomic.Int32
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			n := requests.Add(1)
			assert.Equal(t, "/api"+path, r.URL.Path, "GetHistoricTrades should request the market's executions")
			body := firstPage
			switch {
			case n > 2:
				w.WriteHeader(http.StatusTooManyRequests)
				body = []byte(`{"status":"too_many_requests","reason":"the listing should have stopped"}`)
			case r.URL.Query().Get("continuation_token") == token:
				body = repeatedPage
			}
			_, err := w.Write(body)
			assert.NoError(t, err, "Write should not error")
		})
		result, err := ex.GetHistoricTrades(t.Context(), futuresTestPair, asset.Futures, time.UnixMilli(1791504000000), time.UnixMilli(1791504001000))
		require.NoError(t, err, "GetHistoricTrades must not error")
		assert.Equal(t, futuresTestExecutions(ex.Name), result, "GetHistoricTrades should stop listing once the continuation token repeats")
	})

	t.Run("futures direction unknown", func(t *testing.T) {
		t.Parallel()
		path := "/history/v3/market/PF_XBTUSD/executions"
		page, err := jsonparser.Set(recordedResponse(t, path, "before=1791504001000&since=1791504000000&sort=asc"), []byte(`"Unknown"`), "elements", "[1]", "event", "Execution", "execution", "takerOrder", "direction")
		require.NoError(t, err, "Set must not error")
		page = jsonparser.Delete(page, "continuationToken")
		ex := newRouteTestExchange(t, map[string][]byte{"/api" + path: page})
		result, err := ex.GetHistoricTrades(t.Context(), futuresTestPair, asset.Futures, time.UnixMilli(1791504000000), time.UnixMilli(1791504001000))
		require.NoError(t, err, "GetHistoricTrades must not error")
		exp := futuresTestExecutions(ex.Name)[:2]
		exp[1].Side = order.UnknownSide
		assert.Equal(t, exp, result, "GetHistoricTrades should keep a trade whose side Kraken could not decode")
	})

	if !mockTests {
		end := time.Now().Truncate(time.Second)
		start := end.Add(-time.Minute)
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTestPair, asset.Futures: futuresTestPair} {
			result, err := e.GetHistoricTrades(t.Context(), p, a, start, end)
			require.NoErrorf(t, err, "GetHistoricTrades must not error for %s", a)
			for i := range result {
				assert.WithinRangef(t, result[i].Timestamp, start, end, "GetHistoricTrades should return %s trade %d from the window", a, i)
			}
		}
		return
	}

	adiEUR := currency.NewPairWithDelimiter("ADI", "EUR", currency.UnderscoreDelimiter)
	spotTrade := func(pair currency.Pair, id string, side order.Side, price, amount float64, nanoseconds int64) trade.Data {
		return trade.Data{TID: id, Exchange: e.Name, CurrencyPair: pair, AssetType: asset.Spot, Side: side, Price: price, Amount: amount, Timestamp: time.Unix(0, nanoseconds)}
	}
	for _, tc := range []struct {
		name       string
		pair       currency.Pair
		a          asset.Item
		start, end time.Time
		exp        []trade.Data
	}{
		{
			// Since is inclusive, so the second page starts with the trade the first ended on. The window ends before the
			// second page's last trade, which ends the paging
			name:  "spot pages",
			pair:  spotTestPair,
			a:     asset.Spot,
			start: time.Unix(1791512024, 0),
			end:   time.Unix(1791512024, 210000000),
			exp: []trade.Data{
				spotTrade(spotTestPair, "110772754", order.Buy, 81967.1, 0.00646718, 1791512024048369200),
				spotTrade(spotTestPair, "110772755", order.Buy, 81967.1, 0.00117618, 1791512024098315700),
				spotTrade(spotTestPair, "110772756", order.Buy, 81967.1, 0.00058718, 1791512024145396000),
				spotTrade(spotTestPair, "110772757", order.Buy, 81967.1, 0.00364788, 1791512024202772100),
			},
		},
		{
			// Polling from the last trade returns it again without advancing the cursor, which ends the paging
			name:  "spot pages ending at the last trade",
			pair:  adiEUR,
			a:     asset.Spot,
			start: time.Unix(1791395250, 0),
			end:   time.Unix(1791417600, 0),
			exp: []trade.Data{
				spotTrade(adiEUR, "61409", order.Buy, 7.2683, 2.75228, 1791395250093427200),
				spotTrade(adiEUR, "61410", order.Sell, 7.2387, 2.75228, 1791401645809510000),
				spotTrade(adiEUR, "61411", order.Buy, 7.2448, 1.3803, 1791417503590423600),
				spotTrade(adiEUR, "61412", order.Sell, 7.216, 1.3803, 1791417573349830400),
			},
		},
		{
			// An empty page ends the paging
			name:  "spot window after the last trade",
			pair:  adiEUR,
			a:     asset.Spot,
			start: time.Unix(0, 1791417573349830303),
			end:   time.Unix(1791417600, 0),
		},
		{
			// The last page carries no continuation token
			name:  "futures pages",
			pair:  futuresTestPair,
			a:     asset.Futures,
			start: time.UnixMilli(1791504000000),
			end:   time.UnixMilli(1791504001000),
			exp:   futuresTestExecutions(e.Name),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetHistoricTrades(t.Context(), tc.pair, tc.a, tc.start, tc.end)
			require.NoError(t, err, "GetHistoricTrades must not error")
			assert.Equal(t, tc.exp, result, "GetHistoricTrades should return every trade of the window once")
		})
	}
}

// raiseCandleLimit lets an exchange request candles beyond the 720 most recent Kraken serves, so recorded candles stay
// requestable as they age
func raiseCandleLimit(ex *Exchange) {
	ex.Features.Enabled.Kline.GlobalResultLimit = 1000000
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	end := time.Now().Truncate(time.Hour)
	start := end.Add(-3 * time.Hour)
	_, err := ex.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricCandles must reject an empty pair")
	_, err = ex.GetHistoricCandles(t.Context(), currency.NewPairWithDelimiter("ETH", "USD", currency.UnderscoreDelimiter), asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, currency.ErrPairNotEnabled, "GetHistoricCandles must reject a pair that is not enabled")
	_, err = ex.GetHistoricCandles(t.Context(), spotTestPair, asset.Spot, kline.ThirtySecond, start, end)
	require.ErrorIs(t, err, kline.ErrCannotConstructInterval, "GetHistoricCandles must reject an interval it cannot build")
	_, err = ex.GetHistoricCandles(t.Context(), spotTestPair, asset.Spot, kline.OneDay, end.AddDate(0, 0, -1000), end.AddDate(0, 0, -998))
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "GetHistoricCandles must reject spot candles older than the 720 most recent")
	_, err = newMarginTestExchange(t).GetHistoricCandles(t.Context(), spotTestPair, asset.Margin, kline.OneHour, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles must reject an unsupported asset")
	errEx := newErrorReplyTestExchange(t)
	_, err = errEx.GetHistoricCandles(t.Context(), spotTestPair, asset.Spot, kline.OneHour, start, end)
	assert.ErrorIs(t, err, errAPIResponse, "GetHistoricCandles should return the spot request's error")
	_, err = errEx.GetHistoricCandles(t.Context(), futuresTestPair, asset.Futures, kline.OneHour, start, end)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetHistoricCandles should return the futures request's error")

	if !mockTests {
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTestPair, asset.Futures: futuresTestPair} {
			result, err := ex.GetHistoricCandles(t.Context(), p, a, kline.OneHour, start, end)
			require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", a)
			assert.Lenf(t, result.Candles, 3, "GetHistoricCandles should return every %s candle of the window", a)
		}
		return
	}

	raised := newTestExchange(t)
	raiseCandleLimit(raised)
	candle := func(seconds int64, o, h, l, c, v float64) kline.Candle {
		return kline.Candle{Time: time.Unix(seconds, 0), Open: o, High: h, Low: l, Close: c, Volume: v}
	}
	// Weekly candles start on Mondays, each built from its days' candles
	week := func(days []kline.Candle) kline.Candle {
		w := kline.Candle{Time: days[0].Time, Open: days[0].Open, Low: days[0].Low, Close: days[len(days)-1].Close}
		for i := range days {
			w.High, w.Low, w.Volume = max(w.High, days[i].High), min(w.Low, days[i].Low), w.Volume+days[i].Volume
		}
		return w
	}
	for _, tc := range []struct {
		name       string
		ex         *Exchange
		pair       currency.Pair
		a          asset.Item
		interval   kline.Interval
		start, end time.Time
		exp        []kline.Candle
	}{
		{
			// The day before the window and the days from its end, which the request also returns, are left out
			name:     "spot",
			ex:       raised,
			pair:     spotTestPair,
			a:        asset.Spot,
			interval: kline.OneDay,
			start:    time.Unix(1791158400, 0),
			end:      time.Unix(1791417600, 0),
			exp: []kline.Candle{
				candle(1791158400, 86506.6, 86973.6, 84965.7, 85751.5, 2531.71421762),
				candle(1791244800, 85751.5, 86683.4, 85110.1, 85542.1, 2145.39290006),
				candle(1791331200, 85542.0, 85588.9, 82720.3, 83276.1, 3278.10115136),
			},
		},
		{
			name:     "spot weeks",
			ex:       raised,
			pair:     spotTestPair,
			a:        asset.Spot,
			interval: kline.OneWeek,
			start:    time.Unix(1789344000, 0),
			end:      time.Unix(1790553600, 0),
			exp: []kline.Candle{
				week([]kline.Candle{
					candle(1789344000, 76800.2, 79581.2, 76349.8, 78193.1, 3875.65840124),
					candle(1789430400, 78184.2, 78242.1, 74891.5, 75585.1, 3929.44425693),
					candle(1789516800, 75585.1, 76475.0, 74957.2, 76146.1, 2837.98106161),
					candle(1789603200, 76146.2, 77095.3, 75936.5, 76354.8, 1857.39349184),
					candle(1789689600, 76354.7, 81373.8, 76231.0, 80878.1, 4498.72182339),
					candle(1789776000, 80878.1, 81915.0, 80823.7, 81226.5, 1676.12921419),
					candle(1789862400, 81234.9, 81458.4, 80101.0, 81164.0, 1573.99717196),
				}),
				week([]kline.Candle{
					candle(1789948800, 81164.0, 87446.7, 80839.5, 86593.8, 5026.76127235),
					candle(1790035200, 86598.0, 86705.3, 85100.0, 86196.7, 3199.34411042),
					candle(1790121600, 86199.4, 87274.4, 83501.0, 84384.6, 3504.64548619),
					candle(1790208000, 84384.5, 84914.8, 82832.3, 84380.0, 3362.85864323),
					candle(1790294400, 84380.0, 85247.4, 83163.6, 84090.5, 3084.16569499),
					candle(1790380800, 84090.5, 84451.3, 83777.0, 84426.7, 950.99825725),
					candle(1790467200, 84426.8, 85142.8, 84122.8, 84444.2, 1317.7453689),
				}),
			},
		},
		{
			name:     "futures",
			ex:       raised,
			pair:     futuresTestPair,
			a:        asset.Futures,
			interval: kline.OneHour,
			start:    time.Unix(1791489600, 0),
			end:      time.Unix(1791500400, 0),
			exp: []kline.Candle{
				{Time: time.UnixMilli(1791489600000), Open: 81726, High: 81849, Low: 81640, Close: 81773, Volume: 195.9643},
				{Time: time.UnixMilli(1791493200000), Open: 81773, High: 81773, Low: 81561, Close: 81652, Volume: 82.449},
				{Time: time.UnixMilli(1791496800000), Open: 81652, High: 81898, Low: 81632, Close: 81840, Volume: 83.9653},
			},
		},
		{
			// The charts serve every candle from a market's listing, unlike Get OHLC Data's 720 most recent
			name:     "futures older than the 720 most recent",
			ex:       ex,
			pair:     futuresTestPair,
			a:        asset.Futures,
			interval: kline.OneHour,
			start:    time.Unix(1760000400, 0),
			end:      time.Unix(1760007600, 0),
			exp: []kline.Candle{
				{Time: time.UnixMilli(1760000400000), Open: 121314, High: 121925, Low: 121312, Close: 121869, Volume: 125.4147},
				{Time: time.UnixMilli(1760004000000), Open: 121869, High: 122430, Low: 121714, Close: 122253, Volume: 192.7009000000001},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.ex.GetHistoricCandles(t.Context(), tc.pair, tc.a, tc.interval, tc.start, tc.end)
			require.NoError(t, err, "GetHistoricCandles must not error")
			exp := &kline.Item{Exchange: tc.ex.Name, Pair: tc.pair, Asset: tc.a, Interval: tc.interval, Candles: tc.exp}
			assert.Equal(t, exp, result, "GetHistoricCandles should return every candle of the window")
		})
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(1791489600, 0), time.Unix(1791504000, 0)
	_, err := e.GetHistoricCandlesExtended(t.Context(), spotTestPair, asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended must reject spot, whose candles are limited to the 720 most recent")
	_, err = e.GetHistoricCandlesExtended(t.Context(), futuresTestPair, asset.Futures, kline.ThirtySecond, start, end)
	require.ErrorIs(t, err, kline.ErrCannotConstructInterval, "GetHistoricCandlesExtended must reject an interval it cannot build")
	_, err = newErrorReplyTestExchange(t).GetHistoricCandlesExtended(t.Context(), futuresTestPair, asset.Futures, kline.OneHour, start, end)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetHistoricCandlesExtended should return the request's error")

	ex := newTestExchange(t)
	if !mockTests {
		recent := time.Now().Truncate(time.Hour)
		result, err := ex.GetHistoricCandlesExtended(t.Context(), futuresTestPair, asset.Futures, kline.OneHour, recent.Add(-6*time.Hour), recent)
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		assert.Len(t, result.Candles, 6, "GetHistoricCandlesExtended should return every candle of the window")
		return
	}
	// Two candles a request split the window into two requests
	ex.Features.Enabled.Kline.GlobalResultLimit = 2
	result, err := ex.GetHistoricCandlesExtended(t.Context(), futuresTestPair, asset.Futures, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	exp := &kline.Item{
		Exchange: ex.Name,
		Pair:     futuresTestPair,
		Asset:    asset.Futures,
		Interval: kline.OneHour,
		Candles: []kline.Candle{
			{Time: time.UnixMilli(1791489600000), Open: 81726, High: 81849, Low: 81640, Close: 81773, Volume: 195.9643},
			{Time: time.UnixMilli(1791493200000), Open: 81773, High: 81773, Low: 81561, Close: 81652, Volume: 82.449},
			{Time: time.UnixMilli(1791496800000), Open: 81652, High: 81898, Low: 81632, Close: 81840, Volume: 83.9653},
			{Time: time.UnixMilli(1791500400000), Open: 81840, High: 81886, Low: 81639, Close: 81688, Volume: 56.1316},
		},
	}
	assert.Equal(t, exp, result, "GetHistoricCandlesExtended should return the candles of every request")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := newErrorReplyTestExchange(t).GetServerTime(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, errAPIResponse, "GetServerTime should return the request's error")
	// Kraken's server time is the same for every asset
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		result, err := e.GetServerTime(t.Context(), a)
		require.NoErrorf(t, err, "GetServerTime must not error for %s", a)
		if !mockTests {
			assert.WithinDurationf(t, time.Now(), result, time.Minute, "GetServerTime should return the current time for %s", a)
			continue
		}
		assert.Equalf(t, time.Unix(1791504602, 0), result, "GetServerTime should return the server time for %s", a)
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject an unsupported futures asset")
	_, err = newErrorReplyTestExchange(t).GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, errAPIResponse, "GetFuturesContractDetails should return the request's error")

	// The maximum leverage is that of the first retail margin level
	maxLeverage := func(initialMargin float64) float64 { return 1 / initialMargin }
	listed := func(exchangeName string) []futures.Contract {
		return []futures.Contract{
			{
				Exchange:           exchangeName,
				Name:               pfAnthropicXUSDPair,
				Underlying:         currency.NewPair(currency.NewCode("ANTHROPICx"), currency.USD),
				Asset:              asset.Futures,
				StartDate:          time.Date(2026, 6, 15, 11, 21, 55, 0, time.UTC),
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USD,
				Multiplier:         1,
				MaxLeverage:        maxLeverage(0.15),
				// A pre-IPO perpetual's funding rate has a floor of its own
				FundingRateFloor:   decimal.MustFromFloat(6.25e-06),
				FundingRateCeiling: decimal.MustFromFloat(7e-06),
			},
			{
				Exchange:           exchangeName,
				Name:               piXBTUSDPair,
				Underlying:         currency.NewBTCUSD(),
				Asset:              asset.Futures,
				StartDate:          time.Date(2018, 8, 31, 0, 0, 0, 0, time.UTC),
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Inverse,
				SettlementCurrency: currency.BTC,
				Multiplier:         1,
				MaxLeverage:        maxLeverage(0.5),
				FundingRateFloor:   decimal.MustFromFloat(-0.005),
				FundingRateCeiling: decimal.MustFromFloat(0.005),
			},
			{
				Exchange:           exchangeName,
				Name:               pfEURUSDPair,
				Underlying:         currency.NewPair(currency.EUR, currency.USD),
				Asset:              asset.Futures,
				StartDate:          time.Date(2025, 4, 17, 12, 59, 47, 0, time.UTC),
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USD,
				Multiplier:         1,
				MaxLeverage:        maxLeverage(0.03),
				FundingRateFloor:   decimal.MustFromFloat(-0.001),
				FundingRateCeiling: decimal.MustFromFloat(0.001),
			},
		}
	}

	t.Run("listed", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
		require.NoError(t, err, "GetFuturesContractDetails must not error")
		if !mockTests {
			assert.NotEmpty(t, result, "GetFuturesContractDetails should return contracts")
			return
		}
		assert.Equal(t, listed(e.Name), result, "GetFuturesContractDetails should return every contract")
	})

	t.Run("expired", func(t *testing.T) {
		t.Parallel()
		ex := newRouteTestExchange(t, map[string][]byte{"/derivatives/api/v3/instruments": listedAndExpiredInstruments(t)})
		result, err := ex.GetFuturesContractDetails(t.Context(), asset.Futures)
		require.NoError(t, err, "GetFuturesContractDetails must not error")
		exp := append(listed(ex.Name), futures.Contract{
			Exchange:           ex.Name,
			Name:               fiXBTUSD260828Pair,
			Underlying:         currency.NewBTCUSD(),
			Asset:              asset.Futures,
			StartDate:          time.Date(2026, 7, 31, 15, 0, 46, 0, time.UTC),
			EndDate:            time.Date(2026, 8, 28, 15, 0, 0, 0, time.UTC),
			Type:               futures.Monthly,
			SettlementType:     futures.Inverse,
			SettlementCurrency: currency.BTC,
			Multiplier:         1,
			MaxLeverage:        maxLeverage(0.5),
		})
		assert.Equal(t, exp, result, "GetFuturesContractDetails should return an expired contract as inactive")
	})
}

func TestFuturesContractType(t *testing.T) {
	t.Parallel()
	// Kraken lists a fixed maturity future well before its cycle starts: weekly futures expire on Fridays, monthly ones
	// on a month's last Friday and quarterly ones on the last Friday of March, June, September and December
	for _, tc := range []struct {
		name        string
		lastTrading time.Time
		exp         futures.ContractType
	}{
		{name: "perpetual", exp: futures.Perpetual},
		// FF_XBTUSD_261009, listed a week before it expired
		{name: "weekly", lastTrading: time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC), exp: futures.Weekly},
		// FI_XBTUSD_260828, listed four weeks before it expired
		{name: "monthly", lastTrading: time.Date(2026, 8, 28, 15, 0, 0, 0, time.UTC), exp: futures.Monthly},
		// FI_XBTUSD_261030, listed five weeks before it expires
		{name: "monthly listed early", lastTrading: time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC), exp: futures.Monthly},
		// FF_XBTUSD_261225, listed seven months before it expires
		{name: "quarterly", lastTrading: time.Date(2026, 12, 25, 8, 0, 0, 0, time.UTC), exp: futures.Quarterly},
		// FF_XBTUSD_270326, which Kraken tags semiannual while a nearer quarterly future trades
		{name: "next quarterly", lastTrading: time.Date(2027, 3, 26, 8, 0, 0, 0, time.UTC), exp: futures.Quarterly},
		// A weekly future expiring on the first of May is still the last of April west of UTC
		{name: "weekly in another zone", lastTrading: time.Date(2026, 5, 1, 2, 0, 0, 0, time.UTC).In(time.FixedZone("UTC-7", -7*60*60)), exp: futures.Weekly},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, futuresContractType(tc.lastTrading), "futuresContractType should return the contract's cycle")
		})
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject an unsupported asset")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: pfAnthropicXUSDPair})
	require.ErrorIs(t, err, currency.ErrPairNotContainedInAvailablePairs, "GetLatestFundingRates must reject a pair that is not available")
	_, err = newErrorReplyTestExchange(t).GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures})
	assert.ErrorIs(t, err, errAPIResponse, "GetLatestFundingRates should return the request's error")

	t.Run("fixed maturity", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: currency.NewPairWithDelimiter("FF", "XBTUSD_260925", currency.UnderscoreDelimiter)})
		assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates should reject a fixed maturity future")
	})

	// Kraken's relative funding rate is a fraction of the price. Funding is hourly, so the latest rate applies from the
	// start of the server's hour and the predicted rate from the next
	checked := time.Date(2026, 10, 9, 0, 34, 8, 878000000, time.UTC)
	next := time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC)
	rate := func(exchangeName string, pair currency.Pair, latest, predicted float64) fundingrate.LatestRateResponse {
		r := fundingrate.LatestRateResponse{
			Exchange:       exchangeName,
			Asset:          asset.Futures,
			Pair:           pair,
			LatestRate:     fundingrate.Rate{Time: next.Add(-time.Hour), Rate: decimal.MustFromFloat(latest)},
			TimeOfNextRate: next,
			TimeChecked:    checked,
		}
		if predicted != 0 {
			r.PredictedUpcomingRate = fundingrate.Rate{Time: next, Rate: decimal.MustFromFloat(predicted)}
		}
		return r
	}

	t.Run("enabled perpetuals", func(t *testing.T) {
		t.Parallel()
		// The recorded tickers with those of PF_EURUSD and PI_XBTUSD, perpetuals that are available but not enabled
		var listed, others struct {
			Result     string            `json:"result"`
			Tickers    []json.RawMessage `json:"tickers"`
			ServerTime string            `json:"serverTime"`
		}
		require.NoError(t, json.Unmarshal(recordedResponse(t, "/api/v3/tickers", ""), &listed), "Unmarshal must not error for the listed tickers")
		require.NoError(t, json.Unmarshal(recordedResponse(t, "/api/v3/tickers", "contractType=flexible_futures&contractType=futures_inverse&symbol=PF_EURUSD&symbol=PI_XBTUSD"), &others), "Unmarshal must not error for the other tickers")
		listed.Tickers = append(listed.Tickers, others.Tickers...)
		tickers, err := json.Marshal(&listed)
		require.NoError(t, err, "Marshal must not error")
		ex := newRouteTestExchange(t, map[string][]byte{"/derivatives/api/v3/tickers": tickers})
		result, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures})
		require.NoError(t, err, "GetLatestFundingRates must not error")
		assert.Equal(t, []fundingrate.LatestRateResponse{rate(ex.Name, futuresTestPair, 1.8166270833333e-05, 0)}, result, "GetLatestFundingRates should return the rates of the enabled perpetuals")
		// A pair requested need only be available
		result, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: piXBTUSDPair})
		require.NoError(t, err, "GetLatestFundingRates must not error for PI_XBTUSD")
		assert.Equal(t, []fundingrate.LatestRateResponse{rate(ex.Name, piXBTUSDPair, 0.000107753595833333, 0)}, result, "GetLatestFundingRates should return the rate of the pair requested")
	})

	// The fixed maturity FF_XBTUSD_261225 is left out
	for _, tc := range []struct {
		name string
		req  *fundingrate.LatestRateRequest
		exp  []fundingrate.LatestRateResponse
	}{
		{name: "perpetuals", req: &fundingrate.LatestRateRequest{Asset: asset.Futures}, exp: []fundingrate.LatestRateResponse{rate(e.Name, futuresTestPair, 1.8166270833333e-05, 0)}},
		{
			name: "pair with predicted rate",
			req:  &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: futuresTestPair, IncludePredictedRate: true},
			exp:  []fundingrate.LatestRateResponse{rate(e.Name, futuresTestPair, 1.8166270833333e-05, 1.5901875e-05)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLatestFundingRates(t.Context(), tc.req)
			require.NoError(t, err, "GetLatestFundingRates must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetLatestFundingRates should return rates")
				return
			}
			assert.Equal(t, tc.exp, result, "GetLatestFundingRates should return each perpetual's rate")
		})
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  bool
	}{
		{a: asset.Spot, pair: spotTestPair},
		{a: asset.Spot, pair: futuresTestPair},
		{a: asset.Futures, pair: futuresTestPair, exp: true},
		{a: asset.Futures, pair: piXBTUSDPair, exp: true},
		{a: asset.Futures, pair: currency.NewPairWithDelimiter("pf", "ethusd", currency.UnderscoreDelimiter), exp: true},
		{a: asset.Futures, pair: currency.NewPairWithDelimiter("FF", "XBTUSD_261225", currency.UnderscoreDelimiter)},
		{a: asset.Futures, pair: fiXBTUSD260828Pair},
		{a: asset.Futures},
	} {
		result, err := e.IsPerpetualFutureCurrency(tc.a, tc.pair)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, result, "IsPerpetualFutureCurrency should report whether %s %s is a perpetual", tc.a, tc.pair)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	futuresKey := key.PairAsset{Base: futuresTestPair.Base.Item, Quote: futuresTestPair.Quote.Item, Asset: asset.Futures}
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.XBT.Item, Quote: currency.USD.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must reject an unsupported asset")
	_, err = newErrorReplyTestExchange(t).GetOpenInterest(t.Context())
	assert.ErrorIs(t, err, errAPIResponse, "GetOpenInterest should return the request's error")

	exp := []futures.OpenInterest{{Key: key.NewExchangeAssetPair(e.Name, asset.Futures, futuresTestPair), OpenInterest: 2357.1126}}
	for _, tc := range []struct {
		name string
		keys []key.PairAsset
		exp  []futures.OpenInterest
	}{
		// FF_XBTUSD_261225's ticker is left out, as it is not an available pair
		{name: "enabled pairs", exp: exp},
		{name: "requested pair", keys: []key.PairAsset{futuresKey}, exp: exp},
		{name: "other pair", keys: []key.PairAsset{{Base: piXBTUSDPair.Base.Item, Quote: piXBTUSDPair.Quote.Item, Asset: asset.Futures}}, exp: []futures.OpenInterest{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOpenInterest(t.Context(), tc.keys...)
			require.NoError(t, err, "GetOpenInterest must not error")
			if !mockTests {
				if len(tc.keys) == 0 || tc.keys[0] == futuresKey {
					require.Len(t, result, 1, "GetOpenInterest must return PF_XBTUSD's open interest")
					assert.Positive(t, result[0].OpenInterest, "GetOpenInterest should return PF_XBTUSD's open interest")
				}
				return
			}
			assert.Equal(t, tc.exp, result, "GetOpenInterest should return the open interest of the enabled pairs requested")
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must reject an empty pair")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Margin, spotTestPair)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an asset without pairs")
	_, err = newMarginTestExchange(t).GetCurrencyTradeURL(t.Context(), asset.Margin, spotTestPair)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an unsupported asset")

	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{a: asset.Spot, pair: spotTestPair, exp: "https://pro.kraken.com/app/trade/xbt-usd"},
		{a: asset.Spot, pair: currency.NewPairWithDelimiter("XDG", "USD", "/"), exp: "https://pro.kraken.com/app/trade/xdg-usd"},
		{a: asset.Futures, pair: futuresTestPair, exp: "https://futures.kraken.com/trade/futures/PF_XBTUSD"},
		{a: asset.Futures, pair: currency.NewPairWithDelimiter("pi", "xbtusd", "-"), exp: "https://futures.kraken.com/trade/futures/PI_XBTUSD"},
	} {
		result, err := e.GetCurrencyTradeURL(t.Context(), tc.a, tc.pair)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s %s", tc.a, tc.pair)
		assert.Equalf(t, tc.exp, result, "GetCurrencyTradeURL should return the trade page of %s %s", tc.a, tc.pair)
	}
}

func TestPrivateWebsocketAvailable(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	useTestCredentials(ex)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	assert.False(t, ex.privateWebsocketAvailable(), "privateWebsocketAvailable should be false while the websocket is not connected")

	private := newWsTradingTestExchange(t, "")
	private.Websocket.SetCanUseAuthenticatedEndpoints(false)
	assert.False(t, private.privateWebsocketAvailable(), "privateWebsocketAvailable should be false without authenticated websocket use")
	private.Websocket.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, private.privateWebsocketAvailable(), "privateWebsocketAvailable should be true with the private connection connected")

	// The private connection connects only with private subscriptions, so credentials alone leave it unconnected
	public := newTestExchange(t)
	seedTestAssetNames(public)
	useTestCredentials(public)
	public.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}
	useTestWebsocket(t, public, newMockWebsocketURL(t, new(mockSubscriptionServer).handle), wsPublicConnection, wsPrivateConnection)
	require.NoError(t, public.Websocket.Connect(t.Context()), "Connect must not error")
	require.True(t, public.Websocket.CanUseAuthenticatedEndpoints(), "authenticated websocket use must be allowed")
	assert.False(t, public.privateWebsocketAvailable(), "privateWebsocketAvailable should be false while only the public connection is connected")
}

// accountTestMarginAccounts is Kraken's documented Get wallets example, whose XBT margin account also holds no XRP, with
// two margin accounts added: one whose unrealised profit takes its available funds beyond its balance, and one short of
// margin
const accountTestMarginAccounts = `{"accounts":{` +
	`"cash":{"type":"cashAccount","balances":{"xbt":"141.31756797","xrp":"52465.1254"}},` +
	`"fi_xbtusd":{"type":"marginAccount","currency":"xbt",` +
	`"balances":{"FI_XBTUSD_171215":"50000","FI_XBTUSD_180615":"-15000","xbt":"141.31756797","xrp":"0"},` +
	`"auxiliary":{"usd":0,"pv":153.73891563,"pnl":12.42134766,"af":100.73891563,"funding":100.73891563},` +
	`"marginRequirements":{"im":52.8,"mm":23.76,"lt":39.6,"tt":15.84},` +
	`"triggerEstimates":{"im":3110,"mm":3000,"lt":2890,"tt":2830}},` +
	`"fi_ethusd":{"type":"marginAccount","currency":"eth","balances":{"FI_ETHUSD_261030":"20000","eth":"10.5"},` +
	`"auxiliary":{"usd":0,"pv":13.15,"pnl":2.65,"af":12.6,"funding":0},` +
	`"marginRequirements":{"im":0.55,"mm":0.275,"lt":0.206,"tt":0.1375},` +
	`"triggerEstimates":{"im":2410,"mm":2390,"lt":2380,"tt":2370}},` +
	`"fi_xrpusd":{"type":"marginAccount","currency":"xrp","balances":{"FI_XRPUSD_261030":"-25000","xrp":"6000"},` +
	`"auxiliary":{"usd":0,"pv":3849.5,"pnl":-2150.5,"af":-150.25,"funding":0},` +
	`"marginRequirements":{"im":3999.75,"mm":1999.88,"lt":1499.91,"tt":999.94},` +
	`"triggerEstimates":{"im":0.66,"mm":0.69,"lt":0.7,"tt":0.71}},` +
	`"flex":{"type":"multiCollateralMarginAccount","currencies":{` +
	`"XBT":{"quantity":0.1185308247,"value":4998.721054420551,"collateral":4886.49976674881,"available":0.1185308247},` +
	`"USD":{"quantity":5000,"value":5000,"collateral":5000,"available":5000},` +
	`"EUR":{"quantity":4540.5837374453,"value":4999.137289089901,"collateral":4886.906656949836,"available":4540.5837374453}},` +
	`"balanceValue":34995.52,"portfolioValue":34995.52,"collateralValue":34122.66,"initialMargin":0,` +
	`"initialMarginWithOrders":0,"maintenanceMargin":0,"pnl":0,"unrealizedFunding":0,"totalUnrealized":0,` +
	`"totalUnrealizedAsMargin":0,"marginEquity":34122.66,"availableMargin":34122.66}},` +
	`"result":"success","serverTime":"2016-02-25T09:45:53.818Z"}`

// accountTestFlexWallet returns a Get wallets reply holding an empty cash account and a multi-collateral wallet of
// currencies, a JSON object of each currency's quantity, value, collateral and available margin
func accountTestFlexWallet(currencies string) string {
	return `{"accounts":{"cash":{"type":"cashAccount","balances":{}},` +
		`"flex":{"type":"multiCollateralMarginAccount","currencies":` + currencies + `}},` +
		`"result":"success","serverTime":"2026-10-09T08:30:00.250Z"}`
}

// newAccountTestExchange returns an exchange named after the test, holding the test asset names, whose server answers
// each request to path with the next of replies, repeating the last
func newAccountTestExchange(t *testing.T, path string, replies ...string) *Exchange {
	t.Helper()
	var (
		m    sync.Mutex
		sent int
	)
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path, "request should be sent to the expected endpoint")
		m.Lock()
		reply := replies[min(sent, len(replies)-1)]
		sent++
		m.Unlock()
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	})
	seedTestAssetNames(ex)
	return ex
}

// accountTestSubAccounts checks that saving stamped each balance between earliest and latest, then clears the stamps and
// sorts the sub-accounts by ID, as Kraken's accounts are keyed by name and come in no order, so they compare with an
// expected value
func accountTestSubAccounts(t *testing.T, subAccts accounts.SubAccounts, earliest, latest time.Time) accounts.SubAccounts {
	t.Helper()
	for _, s := range subAccts {
		for c, b := range s.Balances {
			assert.WithinRangef(t, b.UpdatedAt, earliest, latest, "%s %s balance should be stamped when it is saved", s.ID, c)
			b.UpdatedAt = time.Time{}
			s.Balances[c] = b
		}
	}
	slices.SortFunc(subAccts, func(a, b *accounts.SubAccount) int { return strings.Compare(a.ID, b.ID) })
	return subAccts
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an unsupported asset")

	t.Run("spot", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		}
		started := time.Now()
		result, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		ended := time.Now()
		stored, err := ex.GetCachedSubAccounts(t.Context(), asset.Spot)
		require.NoError(t, err, "GetCachedSubAccounts must not error")
		if !mockTests {
			assert.Len(t, result, 1, "UpdateAccountBalances should return the spot account")
			return
		}
		// Kraken keys balances by internal names, which the seeded asset names translate; they do not know USDT, which
		// keeps its own name. The balance available for trading counts what remains of USD's credit line, which is
		// borrowed from as it is used
		exp := accounts.SubAccounts{
			{
				AssetType: asset.Spot,
				Balances: accounts.CurrencyBalances{
					currency.XBT:  {Currency: currency.XBT, Total: 1.2435, Hold: 0.8423, Free: 0.4012, AvailableWithoutBorrow: 0.4012},
					currency.USD:  {Currency: currency.USD, Total: 25435.21, Hold: 8249.76, Free: 20984.949999999997, AvailableWithoutBorrow: 17185.449999999997, Borrowed: 1200.5},
					currency.USDT: {Currency: currency.USDT, Total: 1500, Hold: 250, Free: 1250, AvailableWithoutBorrow: 1250},
				},
			},
		}
		assert.Equal(t, exp, accountTestSubAccounts(t, result, started, ended), "UpdateAccountBalances should return every balance")
		assert.Equal(t, exp, accountTestSubAccounts(t, stored, started, ended), "UpdateAccountBalances should save every balance")
	})

	t.Run("spot credit line limits", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, balance string
			exp           accounts.Balance
		}{
			{
				// Open orders hold more than the balance, drawing on the credit line
				name:    "orders beyond the balance",
				balance: `{"balance":"100.0000","credit":"5000.0000","credit_used":"0.0000","hold_trade":"1200.0000"}`,
				exp:     accounts.Balance{Currency: currency.USD, Total: 100, Hold: 1200, Free: 3900},
			},
			{
				// A credit line reduced below what is used leaves nothing available
				name:    "credit line reduced below its use",
				balance: `{"balance":"0.0000","credit":"1000.0000","credit_used":"1200.5000","hold_trade":"0.0000"}`,
				exp:     accounts.Balance{Currency: currency.USD, Borrowed: 1200.5},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountTestExchange(t, "/0/private/BalanceEx", `{"error":[],"result":{"ZUSD":`+tc.balance+`}}`)
				started := time.Now()
				result, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
				require.NoError(t, err, "UpdateAccountBalances must not error")
				exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.USD: tc.exp}}}
				assert.Equal(t, exp, accountTestSubAccounts(t, result, started, time.Now()), "UpdateAccountBalances should never report a negative available balance")
			})
		}
	})

	t.Run("futures", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		}
		started := time.Now()
		result, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		ended := time.Now()
		stored, err := ex.GetCachedSubAccounts(t.Context(), asset.Futures)
		require.NoError(t, err, "GetCachedSubAccounts must not error")
		if !mockTests {
			assert.NotEmpty(t, result, "UpdateAccountBalances should return the cash account")
			return
		}
		exp := accounts.SubAccounts{
			{
				ID:        "cash",
				AssetType: asset.Futures,
				Balances: accounts.CurrencyBalances{
					currency.XBT: {Currency: currency.XBT, Total: 141.31756797, Free: 141.31756797},
					currency.XRP: {Currency: currency.XRP, Total: 52465.1254, Free: 52465.1254},
				},
			},
			{
				// The contract symbols beside XBT key positions, which are not balances. The account's available funds
				// are free, and the rest of its balance is held as margin
				ID:        "fi_xbtusd",
				AssetType: asset.Futures,
				Balances: accounts.CurrencyBalances{
					currency.XBT: {Currency: currency.XBT, Total: 2.01451296, Hold: 0.10109333999999981, Free: 1.91341962},
				},
			},
			{
				// The margin available in each currency is free, and the rest of its quantity held
				ID:        "flex",
				AssetType: asset.Futures,
				Balances: accounts.CurrencyBalances{
					currency.USD: {Currency: currency.USD, Total: 5000, Hold: 124.5, Free: 4875.5},
					currency.XBT: {Currency: currency.XBT, Total: 0.1185308247, Hold: 0.0049999999999999906, Free: 0.1135308247},
				},
			},
		}
		assert.Equal(t, exp, accountTestSubAccounts(t, result, started, ended), "UpdateAccountBalances should return every account")
		assert.Equal(t, exp, accountTestSubAccounts(t, stored, started, ended), "UpdateAccountBalances should save every account")
	})

	t.Run("futures flex limits", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, currencies string
			exp              accounts.CurrencyBalances
		}{
			{
				// Unrealised profit takes the margin available beyond each quantity, all of which is then free
				name: "unrealised profit",
				currencies: `{"USD":{"quantity":5000,"value":5000,"collateral":5000,"available":5250.75},` +
					`"XBT":{"quantity":0.1185308247,"value":9699.62,"collateral":9481.38,"available":0.1215984214}}`,
				exp: accounts.CurrencyBalances{
					currency.USD: {Currency: currency.USD, Total: 5000, Free: 5000},
					currency.XBT: {Currency: currency.XBT, Total: 0.1185308247, Free: 0.1185308247},
				},
			},
			{
				// A margin shortfall leaves no margin available, so each quantity is held
				name: "margin shortfall",
				currencies: `{"USD":{"quantity":5000,"value":5000,"collateral":5000,"available":-310.4},` +
					`"XBT":{"quantity":0.1185308247,"value":9699.62,"collateral":9481.38,"available":-0.0037932468}}`,
				exp: accounts.CurrencyBalances{
					currency.USD: {Currency: currency.USD, Total: 5000, Hold: 5000},
					currency.XBT: {Currency: currency.XBT, Total: 0.1185308247, Hold: 0.1185308247},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountTestExchange(t, "/derivatives/api/v3/accounts", accountTestFlexWallet(tc.currencies))
				started := time.Now()
				result, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
				require.NoError(t, err, "UpdateAccountBalances must not error")
				exp := accounts.SubAccounts{
					{ID: "cash", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{}},
					{ID: "flex", AssetType: asset.Futures, Balances: tc.exp},
				}
				assert.Equal(t, exp, accountTestSubAccounts(t, result, started, time.Now()), "UpdateAccountBalances should keep each free balance within its quantity")
			})
		}
	})

	t.Run("futures margin limits", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t, "/derivatives/api/v3/accounts", accountTestMarginAccounts)
		started := time.Now()
		result, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		exp := accounts.SubAccounts{
			{
				ID:        "cash",
				AssetType: asset.Futures,
				Balances: accounts.CurrencyBalances{
					currency.XBT: {Currency: currency.XBT, Total: 141.31756797, Free: 141.31756797},
					currency.XRP: {Currency: currency.XRP, Total: 52465.1254, Free: 52465.1254},
				},
			},
			{
				// Unrealised profit takes the available funds beyond the balance, all of which is then free
				ID:        "fi_ethusd",
				AssetType: asset.Futures,
				Balances:  accounts.CurrencyBalances{currency.ETH: {Currency: currency.ETH, Total: 10.5, Free: 10.5}},
			},
			{
				// The available funds are in XBT, the account's currency, so they free none of its XRP
				ID:        "fi_xbtusd",
				AssetType: asset.Futures,
				Balances: accounts.CurrencyBalances{
					currency.XBT: {Currency: currency.XBT, Total: 141.31756797, Hold: 40.578652340000005, Free: 100.73891563},
					currency.XRP: {Currency: currency.XRP},
				},
			},
			{
				// A margin shortfall leaves no funds available, so the balance is held
				ID:        "fi_xrpusd",
				AssetType: asset.Futures,
				Balances:  accounts.CurrencyBalances{currency.XRP: {Currency: currency.XRP, Total: 6000, Hold: 6000}},
			},
			{
				ID:        "flex",
				AssetType: asset.Futures,
				Balances: accounts.CurrencyBalances{
					currency.XBT: {Currency: currency.XBT, Total: 0.1185308247, Free: 0.1185308247},
					currency.USD: {Currency: currency.USD, Total: 5000, Free: 5000},
					currency.EUR: {Currency: currency.EUR, Total: 4540.5837374453, Free: 4540.5837374453},
				},
			},
		}
		assert.Equal(t, exp, accountTestSubAccounts(t, result, started, time.Now()), "UpdateAccountBalances should keep each free balance within its balance")
	})

	t.Run("futures flex emptied", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t, "/derivatives/api/v3/accounts",
			accountTestFlexWallet(`{"USD":{"quantity":5000,"value":5000,"collateral":5000,"available":5000}}`),
			accountTestFlexWallet(`{}`),
		)
		_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
		require.NoError(t, err, "UpdateAccountBalances must not error for the funded wallet")
		started := time.Now()
		result, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
		require.NoError(t, err, "UpdateAccountBalances must not error for the emptied wallet")
		ended := time.Now()
		stored, err := ex.GetCachedSubAccounts(t.Context(), asset.Futures)
		require.NoError(t, err, "GetCachedSubAccounts must not error")
		exp := accounts.SubAccounts{
			{ID: "cash", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{}},
			{ID: "flex", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{}},
		}
		assert.Equal(t, exp, accountTestSubAccounts(t, result, started, ended), "UpdateAccountBalances should return the emptied wallet")
		assert.Equal(t, exp, accountTestSubAccounts(t, stored, started, ended), "UpdateAccountBalances should clear the emptied wallet's balances")
	})

	t.Run("error replies", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, path, reply string
			a                 asset.Item
			err               error
		}{
			{name: "asset names", path: "/0/public/Assets", reply: `{"error":["EService:Unavailable"]}`, a: asset.Spot, err: errAPIResponse},
			{name: "spot balances", path: "/0/private/BalanceEx", reply: `{"error":["EAPI:Invalid key"]}`, a: asset.Spot, err: request.ErrAuthRequestFailed},
			{
				name:  "futures wallets",
				path:  "/derivatives/api/v3/accounts",
				reply: `{"result":"error","error":"authenticationError","serverTime":"2026-10-09T08:30:00.250Z"}`,
				a:     asset.Futures,
				err:   request.ErrAuthRequestFailed,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, tc.path, r.URL.Path, "UpdateAccountBalances should stop at the failed request")
					_, err := w.Write([]byte(tc.reply))
					assert.NoError(t, err, "Write should not error")
				})
				if tc.path != "/0/public/Assets" {
					seedTestAssetNames(ex)
				}
				_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				require.ErrorIs(t, err, tc.err, "UpdateAccountBalances must return the error Kraken replied with")
				_, err = ex.GetCachedSubAccounts(t.Context(), tc.a)
				assert.ErrorIs(t, err, accounts.ErrNoSubAccounts, "UpdateAccountBalances should save nothing after an error")
			})
		}
	})
}

func TestAssetCode(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	seedTestAssetNames(ex)
	names := []string{"XXBT", "BTC", "ZUSD", "USD", "XXDG", "DOGE", "SOL"}
	codes := make(map[string]currency.Code, len(names))
	for _, name := range names {
		codes[name] = ex.assetCode(name)
	}
	// SOL stands for a recently listed asset, which the seeded names do not know and whose name is its own alternative
	exp := map[string]currency.Code{
		"XXBT": currency.XBT,
		"BTC":  currency.XBT,
		"ZUSD": currency.USD,
		"USD":  currency.USD,
		"XXDG": currency.XDG,
		"DOGE": currency.XDG,
		"SOL":  currency.SOL,
	}
	assert.Equal(t, exp, codes, "assetCode should return the alternative name of each internal or display name")
}

func TestEnsureAssetNames(t *testing.T) {
	t.Parallel()
	t.Run("seeded", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(http.ResponseWriter, *http.Request) {
			assert.Fail(t, "ensureAssetNames should not request the names it holds")
		})
		seedTestAssetNames(ex)
		assert.NoError(t, ex.ensureAssetNames(t.Context()), "ensureAssetNames should not error")
	})

	t.Run("error reply", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/public/Assets", r.URL.Path, "ensureAssetNames should request the asset names")
			_, err := w.Write([]byte(`{"error":["EService:Unavailable"]}`))
			assert.NoError(t, err, "Write should not error")
		})
		require.ErrorIs(t, ex.ensureAssetNames(t.Context()), errAPIResponse, "ensureAssetNames must return the error Kraken replied with")
		assert.False(t, ex.assetNames.seeded(), "ensureAssetNames should leave the names unseeded, so they are requested again")
	})

	ex := newTestExchange(t)
	require.NoError(t, ex.ensureAssetNames(t.Context()), "ensureAssetNames must not error")
	if !mockTests {
		alt, ok := ex.assetNames.alternativeName("XXBT")
		assert.True(t, ok, "ensureAssetNames should seed XXBT")
		assert.Equal(t, "XBT", alt, "ensureAssetNames should seed XXBT's alternative name")
		return
	}
	type seededNames struct {
		alternative, display map[string]string
	}
	// Internal names and display names both translate to alternative names, and only the display names that differ
	// translate back
	exp := seededNames{
		alternative: map[string]string{
			"XBT.M": "XBT.M",
			"XETH":  "ETH",
			"XXBT":  "XBT",
			"XXDG":  "XDG",
			"ZUSD":  "USD",
			"BTC":   "XBT",
			"BTC.M": "XBT.M",
			"DOGE":  "XDG",
			"ETH":   "ETH",
			"USD":   "USD",
		},
		display: map[string]string{"XBT": "BTC", "XBT.M": "BTC.M", "XDG": "DOGE"},
	}
	assert.Equal(t, exp, seededNames{ex.assetNames.alternative, ex.assetNames.display}, "ensureAssetNames should seed every asset name")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	t.Run("error replies", func(t *testing.T) {
		t.Parallel()
		for name, failed := range map[string]string{
			"asset names": "/0/public/Assets",
			"deposits":    "/0/private/DepositStatus",
			"withdrawals": "/0/private/WithdrawStatus",
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					reply := `{"error":[],"result":[]}`
					switch r.URL.Path {
					case failed:
						reply = `{"error":["EService:Unavailable"]}`
					case "/0/public/Assets":
						reply = `{"error":[],"result":{"XXBT":{"aclass":"currency","altname":"XBT","decimals":10,"display_decimals":5,"status":"enabled"}}}`
					}
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Write should not error")
				})
				_, err := ex.GetAccountFundingHistory(t.Context())
				assert.ErrorIs(t, err, errAPIResponse, "GetAccountFundingHistory should return the error Kraken replied with")
			})
		}
	})

	ex := newTestExchange(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	result, err := ex.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetAccountFundingHistory should return the recent deposits and withdrawals")
		return
	}
	// Kraken names assets by their internal names, which the seeded asset names translate; they do not know USDT, which
	// keeps its own name
	exp := []exchange.FundingHistory{
		{
			ExchangeName:    ex.Name,
			Status:          "Success",
			TransferID:      "FTRbJ2s-Wc8pQ4rKmzVy56xTGa7Lh",
			Description:     "Bitcoin",
			Timestamp:       time.Unix(1791031122, 0),
			Currency:        "XBT",
			Amount:          0.78125,
			Fee:             0.00001,
			TransferType:    "deposit",
			CryptoToAddress: "2Myd4eaAW96ojk38A2uDK4FbioCayvkEgVq",
			CryptoTxID:      "6544b41b607d8b2512baf801755a3a87b6890eacdb451be8a94059fb11f0a8d9",
		},
		{
			ExchangeName:    ex.Name,
			Status:          "Settled",
			TransferID:      "FTQcuak-V6Za8qrPnhsTx47yYLz8Tg",
			Description:     "Ether (Hex)",
			Timestamp:       time.Unix(1790328722, 0),
			Currency:        "ETH",
			Amount:          0.1383862742,
			Fee:             0.00005,
			TransferType:    "deposit",
			CryptoToAddress: "0xca210f4121dc891c9154026c3ae3d1832a005048",
			CryptoTxID:      "0x339c505eba389bf2c6bebb982cc30c6d82d0bd6a37521fa292890b6b180affc0",
		},
		{
			ExchangeName:    ex.Name,
			Status:          "Success",
			TransferID:      "FTRk9Pq-Wd2sLm7vXcB4nZy31QaHf",
			Description:     "Tether USD (TRC20)",
			Timestamp:       time.Unix(1789712722, 0),
			Currency:        "USDT",
			Amount:          500,
			TransferType:    "deposit",
			CryptoToAddress: "TNVTdTSPRnXkZVAXk1Tq9mW8tT1tqZ8JFr",
			CryptoTxID:      "a1c3e5f7091b2d4f6a8c0e2b4d6f8a0c2e4b6d8f0a2c4e6b8d0f2a4c6e8b0d2f",
		},
		{
			ExchangeName:    ex.Name,
			Status:          "Success",
			TransferID:      "BSNFZU2-MEFN4G-J3NEZV",
			Description:     "Tether USD (TRC20)",
			Timestamp:       time.Unix(1791421057, 0),
			Currency:        "USDT",
			Amount:          1996.5,
			Fee:             2.5,
			TransferType:    "withdrawal",
			CryptoToAddress: "TQmdxSuC16EhFg8FZWtYgrfFRosoRF7bCp",
			CryptoTxID:      "1c7a642fb7387bbc2c6a2c509fd1ae146937f4cf793b4079a4f0715e3a02615a",
			CryptoChain:     "Tron",
		},
		{
			ExchangeName:    ex.Name,
			Status:          "Success",
			TransferID:      "A2BF34S-O7LBNQ-UE4Y4O",
			Description:     "Ether",
			Timestamp:       time.Unix(1791416477, 0),
			Currency:        "ETH",
			Amount:          9.995,
			Fee:             0.005,
			TransferType:    "withdrawal",
			CryptoToAddress: "0x7cb275a5e07ba943fee972e165d80daa67cb2dd0",
			CryptoTxID:      "0x288b83c6b0904d8400ef44e1c9e2187b5c8f7ea3d838222d53f701a15b5c274d",
			CryptoChain:     "Ethereum",
		},
		{
			ExchangeName:    ex.Name,
			Status:          "Pending",
			TransferID:      "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg",
			Description:     "Bitcoin",
			Timestamp:       time.Unix(1790269786, 0),
			Currency:        "XBT",
			Amount:          0.72485,
			Fee:             0.0002,
			TransferType:    "withdrawal",
			CryptoToAddress: "bc1qm32pq4zk7ldx3ewt0j37s2gk8r6dv5e3hq9ka0",
			CryptoTxID:      "29323ce235cee8dae22503caba7a4a5a2ae8ad3a506879a03b1e87992923d804",
			CryptoChain:     "Bitcoin",
		},
	}
	assert.Equal(t, exp, result, "GetAccountFundingHistory should return every recent deposit and withdrawal")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	t.Run("every asset", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		}
		result, err := ex.GetWithdrawalsHistory(t.Context(), currency.EMPTYCODE, asset.Spot)
		require.NoError(t, err, "GetWithdrawalsHistory must not error")
		if !mockTests {
			assert.NotNil(t, result, "GetWithdrawalsHistory should return the recent withdrawals")
			return
		}
		// Kraken returns every asset's withdrawals without an asset filter, each under its internal name
		exp := []exchange.WithdrawalHistory{
			{
				Status:          "Success",
				TransferID:      "BSNFZU2-MEFN4G-J3NEZV",
				Description:     "Tether USD (TRC20)",
				Timestamp:       time.Unix(1791421057, 0),
				Currency:        "USDT",
				Amount:          1996.5,
				Fee:             2.5,
				TransferType:    "withdrawal",
				CryptoToAddress: "TQmdxSuC16EhFg8FZWtYgrfFRosoRF7bCp",
				CryptoTxID:      "1c7a642fb7387bbc2c6a2c509fd1ae146937f4cf793b4079a4f0715e3a02615a",
				CryptoChain:     "Tron",
			},
			{
				Status:          "Success",
				TransferID:      "A2BF34S-O7LBNQ-UE4Y4O",
				Description:     "Ether",
				Timestamp:       time.Unix(1791416477, 0),
				Currency:        "ETH",
				Amount:          9.995,
				Fee:             0.005,
				TransferType:    "withdrawal",
				CryptoToAddress: "0x7cb275a5e07ba943fee972e165d80daa67cb2dd0",
				CryptoTxID:      "0x288b83c6b0904d8400ef44e1c9e2187b5c8f7ea3d838222d53f701a15b5c274d",
				CryptoChain:     "Ethereum",
			},
			{
				Status:          "Pending",
				TransferID:      "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg",
				Description:     "Bitcoin",
				Timestamp:       time.Unix(1790269786, 0),
				Currency:        "XBT",
				Amount:          0.72485,
				Fee:             0.0002,
				TransferType:    "withdrawal",
				CryptoToAddress: "bc1qm32pq4zk7ldx3ewt0j37s2gk8r6dv5e3hq9ka0",
				CryptoTxID:      "29323ce235cee8dae22503caba7a4a5a2ae8ad3a506879a03b1e87992923d804",
				CryptoChain:     "Bitcoin",
			},
		}
		assert.Equal(t, exp, result, "GetWithdrawalsHistory should name each withdrawal's own asset")
	})

	ex := newTestExchange(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	result, err := ex.GetWithdrawalsHistory(t.Context(), currency.XBT, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetWithdrawalsHistory should return the recent withdrawals")
		return
	}
	exp := []exchange.WithdrawalHistory{
		{
			Status:          "Failure",
			TransferID:      "FTQcuak-V6Za8qrPnhsTx47yYLz8Tg",
			Description:     "Bitcoin",
			Timestamp:       time.Unix(1790270623, 0),
			Currency:        "XBT",
			Amount:          0.5,
			Fee:             0.00015,
			TransferType:    "withdrawal",
			CryptoToAddress: "bc1qa35lsx7r4d2tw0e9ufk63egf0872h3wq6n8yp5",
			CryptoTxID:      "29323ce212ceb2daf81255cbea8a5e1fd2ad7a626471e05e1f82929501e82934",
			CryptoChain:     "Bitcoin",
		},
		{
			Status:          "Pending",
			TransferID:      "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg",
			Description:     "Bitcoin",
			Timestamp:       time.Unix(1790269786, 0),
			Currency:        "XBT",
			Amount:          0.72485,
			Fee:             0.0002,
			TransferType:    "withdrawal",
			CryptoToAddress: "bc1qm32pq4zk7ldx3ewt0j37s2gk8r6dv5e3hq9ka0",
			CryptoTxID:      "29323ce235cee8dae22503caba7a4a5a2ae8ad3a506879a03b1e87992923d804",
			CryptoChain:     "Bitcoin",
		},
	}
	assert.Equal(t, exp, result, "GetWithdrawalsHistory should return every recent withdrawal of the currency")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must reject an empty currency")

	t.Run("address requests", func(t *testing.T) {
		t.Parallel()
		const (
			noAddresses = `{"error":[],"result":[]}`
			denied      = `{"error":["EGeneral:Permission denied"]}`
		)
		listed := map[string]any{"asset": "ETH", "method": "Ether (Hex)"}
		generated := map[string]any{"asset": "ETH", "method": "Ether (Hex)", "new": true}
		for _, tc := range []struct {
			name string
			// replies answers the address requests in turn
			replies []string
			exp     []map[string]any
			err     error
		}{
			{name: "no address generated", replies: []string{noAddresses, noAddresses}, exp: []map[string]any{listed, generated}, err: common.ErrNoResponse},
			{name: "addresses refused", replies: []string{denied}, exp: []map[string]any{listed}, err: request.ErrAuthRequestFailed},
			{name: "new address refused", replies: []string{noAddresses, denied}, exp: []map[string]any{listed, generated}, err: request.ErrAuthRequestFailed},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var (
					m        sync.Mutex
					requests []map[string]any
				)
				ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					reply := `{"error":[],"result":[{"method":"Ether (Hex)","limit":false,"fee":"0.0000000000","gen-address":true,"minimum":"0.0001000000"}]}`
					if r.URL.Path == "/0/private/DepositAddresses" {
						body, err := io.ReadAll(r.Body)
						assert.NoError(t, err, "ReadAll should not error")
						var params map[string]any
						assert.NoError(t, json.Unmarshal(body, &params), "Unmarshal should not error")
						delete(params, "nonce")
						m.Lock()
						reply = tc.replies[min(len(requests), len(tc.replies)-1)]
						requests = append(requests, params)
						m.Unlock()
					}
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Write should not error")
				})
				_, err := ex.GetDepositAddress(t.Context(), currency.ETH, "", "")
				require.ErrorIs(t, err, tc.err, "GetDepositAddress must return the expected error")
				m.Lock()
				defer m.Unlock()
				assert.Equal(t, tc.exp, requests, "GetDepositAddress should request a new address only when the method has none")
			})
		}
	})

	if !mockTests {
		// GetDepositAddress generates an address for a method without one, which changes the account
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		result, err := e.GetDepositAddress(t.Context(), currency.XBT, "", "")
		require.NoError(t, err, "GetDepositAddress must not error")
		assert.NotEmpty(t, result.Address, "GetDepositAddress should return an address")
		return
	}
	for _, tc := range []struct {
		name  string
		c     currency.Code
		chain string
		exp   *deposit.Address
		err   error
	}{
		{name: "first method", c: currency.XBT, exp: &deposit.Address{Address: "2N9fRkx5JTWXWHmXzZtvhQsufvoYRMq9ExV", Chain: "Bitcoin"}},
		// Chains are deposit methods, named in any case
		{name: "named chain", c: currency.USDT, chain: "tether usd (trc20)", exp: &deposit.Address{Address: "TNVTdTSPRnXkZVAXk1Tq9mW8tT1tqZ8JFr", Chain: "Tether USD (TRC20)"}},
		{name: "tag", c: currency.XRP, exp: &deposit.Address{Address: "rLHzPsX3oXdzU2qP17kHCH2G4csZv1rAJh", Tag: "1361101127", Chain: "Ripple XRP"}},
		// Kraken sends an EOS address's memo apart from its tag
		{name: "memo", c: currency.EOS, exp: &deposit.Address{Address: "krakenkraken", Tag: "4150096490", Chain: "EOS"}},
		// Ether (Hex) has no address yet, so one is generated
		{name: "new address", c: currency.ETH, exp: &deposit.Address{Address: "0xca210f4121dc891c9154026c3ae3d1832a005048", Chain: "Ether (Hex)"}},
		{name: "unknown chain", c: currency.XBT, chain: "Ethereum", err: errDepositMethodUnknown},
		// XBT.M, an opt-in rewards balance, cannot be deposited
		{name: "no methods", c: currency.NewCode("XBT.M"), err: errNoDepositMethods},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDepositAddress(t.Context(), tc.c, "", tc.chain)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "GetDepositAddress should return the expected error")
				return
			}
			require.NoError(t, err, "GetDepositAddress must not error")
			assert.Equal(t, tc.exp, result, "GetDepositAddress should return the method's first address")
		})
	}
}

// accountTestWithdrawal tests a withdrawal method of a request: its validation, the parameters it sends, which must be
// exp as JSON without the nonce, and the reference ID it returns
func accountTestWithdrawal(t *testing.T, name string, withdrawFunds func(*Exchange, context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error), req *withdraw.Request, exp, referenceID string) {
	t.Helper()
	_, err := withdrawFunds(e, t.Context(), nil)
	require.ErrorIsf(t, err, withdraw.ErrRequestCannotBeNil, "%s must reject a nil request", name)
	withoutKey := *req
	withoutKey.TradePassword = ""
	_, err = withdrawFunds(e, t.Context(), &withoutKey)
	require.ErrorIsf(t, err, errWithdrawalKeyRequired, "%s must reject a request without a withdrawal key name", name)

	// The recorded responses cannot check the withdrawal key, which the mock server only requires to be present
	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/private/Withdraw", r.URL.Path, "the withdrawal should be sent to Withdraw Funds")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "ReadAll should not error")
			assert.JSONEqf(t, exp, string(jsonparser.Delete(body, "nonce")), "%s should send the request's trade password as the withdrawal key name", name)
			_, err = w.Write([]byte(`{"error":[],"result":{"refid":"` + referenceID + `"}}`))
			assert.NoError(t, err, "Write should not error")
		})
		result, err := withdrawFunds(ex, t.Context(), req)
		require.NoErrorf(t, err, "%s must not error", name)
		assert.Equalf(t, &withdraw.ExchangeResponse{ID: referenceID}, result, "%s should return the reference ID", name)
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := withdrawFunds(e, t.Context(), req)
	require.NoErrorf(t, err, "%s must not error", name)
	if mockTests {
		assert.Equalf(t, &withdraw.ExchangeResponse{ID: referenceID}, result, "%s should return the recorded reference ID", name)
		return
	}
	assert.NotEmptyf(t, result.ID, "%s should return a reference ID", name)
}

// accountTestBankAccount returns a bank account a fiat withdrawal of c validates against
func accountTestBankAccount(c currency.Code) banking.Account {
	return banking.Account{
		Enabled:             true,
		BankName:            "Commerzbank",
		AccountName:         "GoCryptoTrader Test",
		AccountNumber:       "0532013000",
		IBAN:                "DE89370400440532013000",
		SWIFTCode:           "COBADEFFXXX",
		SupportedCurrencies: c.String(),
		SupportedExchanges:  "Kraken",
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	accountTestWithdrawal(t, "WithdrawCryptocurrencyFunds", (*Exchange).WithdrawCryptocurrencyFunds, &withdraw.Request{
		Exchange:      "Kraken",
		Currency:      currency.XBT,
		Amount:        0.725,
		Type:          withdraw.Crypto,
		TradePassword: "btc-wallet-1",
		Crypto:        withdraw.CryptoRequest{Address: "bc1qxdsh4sdd29h6ldehz0se5c61asq8cgwyjf2y3z"},
	}, `{"asset":"XBT","key":"btc-wallet-1","address":"bc1qxdsh4sdd29h6ldehz0se5c61asq8cgwyjf2y3z","amount":"0.725"}`, "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	accountTestWithdrawal(t, "WithdrawFiatFunds", (*Exchange).WithdrawFiatFunds, &withdraw.Request{
		Exchange:      "Kraken",
		Currency:      currency.EUR,
		Amount:        1500,
		Type:          withdraw.Fiat,
		TradePassword: "sepa-personal",
		Fiat:          withdraw.FiatRequest{Bank: accountTestBankAccount(currency.EUR)},
	}, `{"asset":"EUR","key":"sepa-personal","amount":"1500"}`, "AGBXF2K-QH7RNT-M3VCZP")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	accountTestWithdrawal(t, "WithdrawFiatFundsToInternationalBank", (*Exchange).WithdrawFiatFundsToInternationalBank, &withdraw.Request{
		Exchange:      "Kraken",
		Currency:      currency.USD,
		Amount:        2500,
		Type:          withdraw.Fiat,
		TradePassword: "swift-business",
		Fiat:          withdraw.FiatRequest{Bank: accountTestBankAccount(currency.USD)},
	}, `{"asset":"USD","key":"swift-business","amount":"2500"}`, "AGCMW4Q-PL2XVB-K8TNRD")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")

	t.Run("without credentials", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		ex.SetCredentials(&accounts.Credentials{})
		builder := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTestPair, PurchasePrice: 85000, Amount: 0.5}
		result, err := ex.GetFeeByType(t.Context(), builder)
		require.NoError(t, err, "GetFeeByType must not error without credentials")
		// Kraken Pro's lowest volume tier charges takers 0.4 percent, the most a spot trade is charged
		assert.Equal(t, 170.0, result, "GetFeeByType should estimate the trade fee without credentials")
		assert.Equal(t, exchange.OfflineTradeFee, builder.FeeType, "GetFeeByType should mark the trade fee as estimated offline")
	})

	t.Run("trade fee missing", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t, "/0/private/TradeVolume", `{"error":[],"result":{"asset_class":"currency","currency":"ZUSD","volume":"200709587.4223",`+
			`"inputs":{"domain_assets_on_platform":"35000000.0000","domain_futures_volume_30d":"1500000.0000","domain_spot_volume_30d":"200709587.4223"}}}`)
		_, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTestPair, PurchasePrice: 85000, Amount: 0.5})
		assert.ErrorIs(t, err, errTradeFeeMissing, "GetFeeByType should report a reply without the pair's fee")
	})

	for _, tc := range []struct {
		name    string
		builder *exchange.FeeBuilder
		exp     float64
		err     error
	}{
		{name: "offline trade fee", builder: &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: spotTestPair, PurchasePrice: 85000, Amount: 0.5}, exp: 170},
		// XBT/USD's taker fee is 0.1 percent and its maker fee 0.02 percent
		{name: "taker trade fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTestPair, PurchasePrice: 85000, Amount: 0.5}, exp: 42.5},
		{name: "maker trade fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTestPair, IsMaker: true, PurchasePrice: 85000, Amount: 0.5}, exp: 8.5},
		// EUR/USD is not on a maker/taker schedule, so makers pay its 0.2 percent fee too
		{
			name:    "maker trade fee without a maker schedule",
			builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewPair(currency.EUR, currency.USD), IsMaker: true, PurchasePrice: 1.1725, Amount: 1000},
			exp:     2.345,
		},
		{name: "trade fee of an empty pair", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, PurchasePrice: 85000, Amount: 0.5}, err: currency.ErrCurrencyPairEmpty},
		// PAX Gold charges Kraken's flat 0.005 and Paxos's 0.02 percent of the amount
		{name: "cryptocurrency withdrawal fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.PAXG, currency.USD), Amount: 10}, exp: 0.007},
		// SEPA (Bank Frick), EUR's first withdrawal method, charges a flat 1 EUR
		{name: "international bank withdrawal fee", builder: &exchange.FeeBuilder{FeeType: exchange.InternationalBankWithdrawalFee, FiatCurrency: currency.EUR, Amount: 1500}, exp: 1},
		// Bitcoin, XBT's first deposit method, charges a flat 0.00001
		{name: "cryptocurrency deposit fee", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyDepositFee, Pair: spotTestPair, Amount: 0.5}, exp: 0.00001},
		// A debit card, EUR's first deposit method, charges 0.25 EUR and 3.75 percent of the amount
		{name: "international bank deposit fee", builder: &exchange.FeeBuilder{FeeType: exchange.InternationalBankDepositFee, FiatCurrency: currency.EUR, Amount: 250}, exp: 9.625},
		{name: "unsupported fee type", builder: &exchange.FeeBuilder{FeeType: exchange.BankFee}, err: common.ErrFunctionNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requested := tc.builder.FeeType != exchange.OfflineTradeFee && tc.builder.FeeType != exchange.BankFee
			if !mockTests && requested {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetFeeByType(t.Context(), tc.builder)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "GetFeeByType should return the expected error")
				return
			}
			require.NoError(t, err, "GetFeeByType must not error")
			if mockTests || !requested {
				assert.Equal(t, tc.exp, result, "GetFeeByType should return the fee")
				return
			}
			assert.GreaterOrEqual(t, result, 0.0, "GetFeeByType should return a fee")
		})
	}
}

func TestWithdrawalFee(t *testing.T) {
	t.Parallel()
	t.Run("empty currency", func(t *testing.T) {
		t.Parallel()
		_, err := e.withdrawalFee(t.Context(), currency.EMPTYCODE, 1)
		assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "withdrawalFee should reject an empty currency")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.withdrawalFee(t.Context(), currency.XBT, 0.725)
		require.NoError(t, err, "withdrawalFee must not error")
		assert.GreaterOrEqual(t, result, 0.0, "withdrawalFee should return a fee")
		return
	}
	for _, tc := range []struct {
		c      currency.Code
		amount float64
		exp    float64
		err    error
	}{
		// Bitcoin, XBT's first method, charges a flat fee, while its second, Lightning, would charge a percentage
		{c: currency.XBT, amount: 0.725, exp: 0.000015},
		// PAX Gold charges Kraken's flat 0.005 and Paxos's 0.02 percent of the amount
		{c: currency.PAXG, amount: 10, exp: 0.007},
		// XBT.M, an opt-in rewards balance, cannot be withdrawn
		{c: currency.NewCode("XBT.M"), amount: 1, err: errNoWithdrawalMethods},
	} {
		t.Run(tc.c.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.withdrawalFee(t.Context(), tc.c, tc.amount)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "withdrawalFee should return the expected error")
				return
			}
			require.NoError(t, err, "withdrawalFee must not error")
			assert.Equal(t, tc.exp, result, "withdrawalFee should return the first method's fee")
		})
	}
}

func TestDepositFee(t *testing.T) {
	t.Parallel()
	_, err := e.depositFee(t.Context(), currency.EMPTYCODE, 1)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "depositFee must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.depositFee(t.Context(), currency.XBT, 0.5)
		require.NoError(t, err, "depositFee must not error")
		assert.GreaterOrEqual(t, result, 0.0, "depositFee should return a fee")
		return
	}
	for _, tc := range []struct {
		c      currency.Code
		amount float64
		exp    float64
		err    error
	}{
		// Bitcoin, XBT's first method, charges a flat fee, while its second, Lightning, charges a percentage
		{c: currency.XBT, amount: 0.5, exp: 0.00001},
		// A debit card, EUR's first method, charges 0.25 EUR and 3.75 percent of the amount
		{c: currency.EUR, amount: 250, exp: 9.625},
		// XBT.M, an opt-in rewards balance, cannot be deposited
		{c: currency.NewCode("XBT.M"), amount: 1, err: errNoDepositMethods},
	} {
		t.Run(tc.c.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.depositFee(t.Context(), tc.c, tc.amount)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "depositFee should return the expected error")
				return
			}
			require.NoError(t, err, "depositFee must not error")
			assert.Equal(t, tc.exp, result, "depositFee should return the first method's fee")
		})
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetAvailableTransferChains(t.Context(), currency.XBT)
		require.NoError(t, err, "GetAvailableTransferChains must not error")
		assert.NotEmpty(t, result, "GetAvailableTransferChains should return XBT's deposit methods")
		return
	}
	for c, exp := range map[currency.Code][]string{
		currency.XBT:  {"Bitcoin", "Bitcoin Lightning"},
		currency.USDT: {"Tether USD (ERC20)", "Tether USD (TRC20)"},
		// XBT.M, an opt-in rewards balance, cannot be deposited
		currency.NewCode("XBT.M"): {},
	} {
		result, err := e.GetAvailableTransferChains(t.Context(), c)
		require.NoErrorf(t, err, "GetAvailableTransferChains must not error for %s", c)
		assert.Equalf(t, exp, result, "GetAvailableTransferChains should return every deposit method of %s", c)
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	err := e.ValidateAPICredentials(t.Context(), asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "ValidateAPICredentials must reject an unsupported asset")

	t.Run("rejected key", func(t *testing.T) {
		t.Parallel()
		ex := newAccountTestExchange(t, "/0/private/BalanceEx", `{"error":["EAPI:Invalid key"]}`)
		assert.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), request.ErrAuthRequestFailed, "ValidateAPICredentials should reject a key Kraken does not know")
	})

	ex := newTestExchange(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		assert.NoErrorf(t, ex.ValidateAPICredentials(t.Context(), a), "ValidateAPICredentials should accept the %s credentials", a)
	}
}

func TestAuthenticateWebsocket(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, reply string
		err         error
	}{
		{name: "rejected key", reply: `{"error":["EAPI:Invalid key"]}`, err: request.ErrAuthRequestFailed},
		{name: "empty token", reply: `{"error":[],"result":{"token":"","expires":900}}`, err: errWebsocketTokenEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newAccountTestExchange(t, "/0/private/GetWebSocketsToken", tc.reply)
			assert.ErrorIs(t, ex.AuthenticateWebsocket(t.Context()), tc.err, "AuthenticateWebsocket should return the error")
		})
	}

	ex := newTestExchange(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	// Authenticating replaces a token held already, which may have expired
	ex.wsToken = "expired-token"
	require.NoError(t, ex.AuthenticateWebsocket(t.Context()), "AuthenticateWebsocket must not error")
	if mockTests {
		assert.Equal(t, wsTestToken, ex.wsToken, "AuthenticateWebsocket should hold the new token")
		return
	}
	assert.NotEqual(t, "expired-token", ex.wsToken, "AuthenticateWebsocket should hold a new token")
}

var (
	// ordersTestExpiry is 2026-10-10T00:00:00Z, given in another zone to prove expiries are sent in UTC
	ordersTestExpiry = time.Date(2026, 10, 10, 10, 0, 0, 0, time.FixedZone("AEST", 10*60*60))

	// ordersETHUSDPair and ordersXBTGBPPair are spot pairs in the test configuration's format
	ordersETHUSDPair = currency.NewPairWithDelimiter("ETH", "USD", "_")
	ordersXBTGBPPair = currency.NewPairWithDelimiter("XBT", "GBP", "_")
)

// ordersWsRoundTrip is a request the private connection's mock expects, without its req_id, and the replies it sends
type ordersWsRoundTrip struct {
	request string
	replies []string
}

// newOrdersWsTestExchange returns an exchange named after the test whose private connection is served by a mock
// expecting the round trips' requests in turn. Its REST requests are served from the recorded responses in mock tests
func newOrdersWsTestExchange(t *testing.T, trips ...ordersWsRoundTrip) *Exchange {
	t.Helper()
	ex := newTestExchange(t)
	useOrdersTestWebsocket(t, ex, trips...)
	return ex
}

// useOrdersTestWebsocket connects an exchange's private connection to a mock expecting the round trips' requests in
// turn, and allows authenticated websocket use, so the wrapper routes spot orders over it. A request beyond the round
// trips fails the test without a reply, and every round trip must be made by the end of the test
func useOrdersTestWebsocket(t *testing.T, ex *Exchange, trips ...ordersWsRoundTrip) {
	t.Helper()
	seedTestAssetNames(ex)
	ex.wsToken = wsTradingTestToken
	var mu sync.Mutex
	remaining := slices.Clone(trips)
	handler := func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		reqID, err := jsonparser.GetInt(msg, "req_id")
		if err != nil {
			return err
		}
		mu.Lock()
		if len(remaining) == 0 {
			mu.Unlock()
			assert.Failf(tb, "no further websocket request should be sent", "%s", msg)
			return nil
		}
		trip := remaining[0]
		remaining = remaining[1:]
		mu.Unlock()
		assert.JSONEq(tb, trip.request, string(jsonparser.Delete(slices.Clone(msg), "req_id")), "request should be the expected one")
		for _, r := range trip.replies {
			reply, err := jsonparser.Set([]byte(r), []byte(strconv.FormatInt(reqID, 10)), "req_id")
			if err != nil {
				return err
			}
			if err := w.WriteMessage(gws.TextMessage, reply); err != nil {
				return err
			}
		}
		return nil
	}
	ex.Features.Subscriptions = subscription.List{}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, handler), wsPrivateConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	// Disabling the websocket before it shuts down stops its connection monitor redialling the mock
	t.Cleanup(func() {
		if ex.Websocket.IsEnabled() {
			assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		}
	})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		assert.Empty(t, remaining, "every expected websocket request should be sent")
	})
}

// ordersWsRequest returns a private request of method carrying params, the request's parameters without its token
func ordersWsRequest(method, params string) string {
	return `{"method":"` + method + `","params":{` + params + `,"token":"` + wsTradingTestToken + `"}}`
}

// ordersWsReply returns a successful reply of method carrying result
func ordersWsReply(method, result string) string {
	return `{"method":"` + method + `","result":` + result + `,"success":true,"time_in":"2026-10-09T08:40:00.112305Z","time_out":"2026-10-09T08:40:00.118210Z"}`
}

// ordersWsRejection returns a reply of method rejecting a request with Kraken's reason
func ordersWsRejection(method, reason string) string {
	return `{"error":"` + reason + `","method":"` + method + `","success":false,"time_in":"2026-10-09T08:40:00.112305Z","time_out":"2026-10-09T08:40:00.118210Z"}`
}

// ordersClearSubmitTimes asserts that the Date and LastUpdated DeriveSubmitResponse sets from the clock fall between
// the submission and now, then clears them so the rest of the response can be compared
func ordersClearSubmitTimes(tb testing.TB, resp *order.SubmitResponse, submitted time.Time) {
	tb.Helper()
	now := time.Now()
	assert.WithinRange(tb, resp.Date, submitted, now, "Date should be when the order was submitted")
	assert.WithinRange(tb, resp.LastUpdated, submitted, now, "LastUpdated should be when the order was submitted")
	resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
}

// newOrdersFailingExchange returns an exchange named after the test whose REST requests Kraken refuses: spot requests
// with a rate limit error and futures requests with apiLimitExceeded
func newOrdersFailingExchange(t *testing.T) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/derivatives/") {
			_, _ = w.Write([]byte(`{"result":"error","error":"apiLimitExceeded","serverTime":"2026-10-09T08:42:00.001Z"}`))
			return
		}
		_, _ = w.Write([]byte(`{"error":["EAPI:Rate limit exceeded"]}`))
	})
}

// ordersStopLossLimitDetail returns the recorded open stop loss limit order OQCLML-BW3P3-BUCMWZ as an exchange named
// exch converts it: partially filled, with its trigger and limit prices taken from its description
func ordersStopLossLimitDetail(exch string) order.Detail {
	openTime := time.UnixMicro(1780582233729133)
	return order.Detail{
		Exchange:             exch,
		AssetType:            asset.Spot,
		OrderID:              "OQCLML-BW3P3-BUCMWZ",
		Pair:                 wsTestPair,
		Side:                 order.Buy,
		Type:                 order.StopLimit,
		Status:               order.PartiallyFilled,
		TimeInForce:          order.GoodTillTime,
		ReduceOnly:           true,
		Price:                30020,
		TriggerPrice:         30010,
		AverageExecutedPrice: 30012.5,
		Amount:               1.25,
		ExecutedAmount:       0.375,
		RemainingAmount:      0.875,
		Cost:                 11254.6875,
		CostAsset:            currency.USD,
		Fee:                  29.26219,
		FeeAsset:             currency.USD,
		Date:                 openTime,
		LastUpdated:          openTime,
	}
}

// ordersFuturesLimitDetail returns the recorded open futures limit order e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b as an
// exchange named exch converts it
func ordersFuturesLimitDetail(exch string) order.Detail {
	return order.Detail{
		Exchange:        exch,
		AssetType:       asset.Futures,
		OrderID:         "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b",
		ClientOrderID:   "gct-batch-1",
		Pair:            futuresTestPair,
		Side:            order.Buy,
		Type:            order.Limit,
		Status:          order.PartiallyFilled,
		ReduceOnly:      true,
		Price:           81450,
		Amount:          0.0007,
		ExecutedAmount:  0.0003,
		RemainingAmount: 0.0004,
		Date:            time.Date(2026, 10, 9, 8, 24, 59, 120000000, time.UTC),
		LastUpdated:     time.Date(2026, 10, 9, 8, 26, 14, 982000000, time.UTC),
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil submission")
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		err    error
	}{
		{"an unsupported asset", &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Buy, Amount: 1}, asset.ErrNotSupported},
		{"a limit order without a price", &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Amount: 1}, order.ErrPriceMustBeSetIfLimitOrder},
		{"a spot OCO order", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Sell, Price: 90000, TriggerPrice: 70000, Amount: 1}, order.ErrUnsupportedOrderType},
		{"a spot good till day order", &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 1, TimeInForce: order.GoodTillDay}, order.ErrUnsupportedTimeInForce},
		{"a spot good till crossing order", &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 1, TimeInForce: order.GoodTillCrossing}, order.ErrUnsupportedTimeInForce},
		{"a spot order triggered by the mark price", &order.Submit{AssetType: asset.Spot, Type: order.Stop, Side: order.Sell, TriggerPrice: 75000, Amount: 1, TriggerPriceType: order.MarkPrice}, order.ErrUnknownPriceType},
		{"a spot good till time order without an end time", &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 1, TimeInForce: order.GoodTillTime}, order.ErrInvalidTimeInForce},
		{"a spot limit order sized in the quote currency", &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 80000, QuoteAmount: 100}, order.ErrAmountIsInvalid},
		{"a spot market sell sized in the quote currency", &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, QuoteAmount: 100}, order.ErrAmountMustBeSet},
		{"a futures OCO order", &order.Submit{AssetType: asset.Futures, Type: order.OCO, Side: order.Sell, Price: 90000, TriggerPrice: 70000, Amount: 1}, order.ErrUnsupportedOrderType},
		{"a futures trailing stop without a tracking mode", &order.Submit{AssetType: asset.Futures, Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingValue: 1.5}, order.ErrUnknownTrackingMode},
		{"a futures trailing stop without a deviation", &order.Submit{AssetType: asset.Futures, Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingMode: order.Percentage}, errFuturesTrailingStopDeviationRequired},
		{"a futures trailing stop deviating more than 50 percent", &order.Submit{AssetType: asset.Futures, Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingMode: order.Percentage, TrackingValue: 60}, errFuturesInvalidTrailingStopDeviation},
		{"a futures stop without a trigger price", &order.Submit{AssetType: asset.Futures, Type: order.Stop, Side: order.Sell, Amount: 1}, errFuturesStopPriceRequired},
		{"a futures order triggered by more than one price", &order.Submit{AssetType: asset.Futures, Type: order.Stop, Side: order.Sell, TriggerPrice: 79000, Amount: 1, TriggerPriceType: order.IndexPrice | order.MarkPrice}, order.ErrUnknownPriceType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange, tc.submit.Pair = e.Name, spotTestPair
			if tc.submit.AssetType == asset.Futures {
				tc.submit.Pair = futuresTestPair
			}
			_, err := e.SubmitOrder(t.Context(), tc.submit)
			assert.ErrorIs(t, err, tc.err, "SubmitOrder should reject the submission")
		})
	}

	t.Run("spot over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{
			request: ordersWsRequest("add_order", `"order_type":"limit","side":"buy","order_qty":0.5,"limit_price":80000,"symbol":"BTC/USD"`),
			replies: []string{ordersWsReply("add_order", `{"order_id":"O3DEHV-GK5TH-0L91EW"}`)},
		})
		submitted := time.Now()
		resp, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.5})
		require.NoError(t, err, "SubmitOrder must not error")
		ordersClearSubmitTimes(t, resp, submitted)
		exp := &order.SubmitResponse{
			Exchange:  ex.Name,
			Type:      order.Limit,
			Side:      order.Buy,
			Pair:      spotTestPair,
			AssetType: asset.Spot,
			Price:     80000,
			Amount:    0.5,
			Status:    order.New,
			OrderID:   "O3DEHV-GK5TH-0L91EW",
		}
		assert.Equal(t, exp, resp, "SubmitOrder should place the order over the private websocket")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			name:   "spot over REST",
			submit: &order.Submit{Exchange: e.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.Stop, Side: order.Sell, TriggerPrice: 75000, Amount: 0.25},
			exp: &order.SubmitResponse{
				Exchange:     e.Name,
				Type:         order.Stop,
				Side:         order.Sell,
				Pair:         spotTestPair,
				AssetType:    asset.Spot,
				Amount:       0.25,
				TriggerPrice: 75000,
				Status:       order.New,
				OrderID:      "OM3IVF-M83OG-3QQVT8",
			},
		},
		{
			name:   "futures",
			submit: &order.Submit{Exchange: e.Name, Pair: futuresTestPair, AssetType: asset.Futures, Type: order.Stop, Side: order.Sell, TriggerPrice: 79000, Amount: 0.001},
			exp: &order.SubmitResponse{
				Exchange:     e.Name,
				Type:         order.Stop,
				Side:         order.Sell,
				Pair:         futuresTestPair,
				AssetType:    asset.Futures,
				Amount:       0.001,
				TriggerPrice: 79000,
				Status:       order.New,
				OrderID:      "770a8a11-d8a4-4525-b292-8b3d4ee2e308",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			submitted := time.Now()
			resp, err := e.SubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "SubmitOrder must not error")
			if !mockTests {
				assert.NotEmpty(t, resp.OrderID, "SubmitOrder should return the order ID")
				return
			}
			ordersClearSubmitTimes(t, resp, submitted)
			assert.Equal(t, tc.exp, resp, "SubmitOrder should return the placed order")
		})
	}
}

func TestSubmitSpotOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		submit  *order.Submit
		params  string
		orderID string
		exp     *order.SubmitResponse
	}{
		{
			name:    "limit order with a client order ID",
			submit:  &order.Submit{Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.5, ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876"},
			params:  `"order_type":"limit","side":"buy","order_qty":0.5,"limit_price":80000,"cl_ord_id":"6d1b345e-2821-40e2-ad83-4ecb18a06876"`,
			orderID: "ODANKU-453KT-QMF7LM",
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.5, ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876", Status: order.New},
		},
		{
			name:    "market order",
			submit:  &order.Submit{Type: order.Market, Side: order.Sell, Amount: 0.25},
			params:  `"order_type":"market","side":"sell","order_qty":0.25`,
			orderID: "OL51OD-S5OLC-C50IJ7",
			exp:     &order.SubmitResponse{Type: order.Market, Side: order.Sell, Amount: 0.25, Status: order.Filled},
		},
		{
			name:    "post only good till cancelled order",
			submit:  &order.Submit{Type: order.Limit, Side: order.Sell, Price: 90000, Amount: 0.5, TimeInForce: order.GoodTillCancel | order.PostOnly},
			params:  `"order_type":"limit","side":"sell","order_qty":0.5,"limit_price":90000,"post_only":true`,
			orderID: "OWHZKD-EBBFR-15PV05",
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Sell, TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 90000, Amount: 0.5, Status: order.New},
		},
		{
			name:    "immediate or cancel order",
			submit:  &order.Submit{Type: order.Limit, Side: order.Buy, Price: 81000, Amount: 0.5, TimeInForce: order.ImmediateOrCancel},
			params:  `"order_type":"limit","side":"buy","order_qty":0.5,"limit_price":81000,"time_in_force":"ioc"`,
			orderID: "OML2DK-TZ1PZ-WBOUPX",
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, TimeInForce: order.ImmediateOrCancel, Price: 81000, Amount: 0.5, Status: order.New},
		},
		{
			name:    "fill or kill order",
			submit:  &order.Submit{Type: order.Limit, Side: order.Buy, Price: 81000, Amount: 0.5, TimeInForce: order.FillOrKill},
			params:  `"order_type":"limit","side":"buy","order_qty":0.5,"limit_price":81000,"time_in_force":"fok"`,
			orderID: "OXDWIB-YOFPG-79PWEO",
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, TimeInForce: order.FillOrKill, Price: 81000, Amount: 0.5, Status: order.New},
		},
		{
			name:    "good till time order",
			submit:  &order.Submit{Type: order.Limit, Side: order.Sell, Price: 90000, Amount: 0.5, TimeInForce: order.GoodTillTime, EndTime: ordersTestExpiry},
			params:  `"order_type":"limit","side":"sell","order_qty":0.5,"limit_price":90000,"time_in_force":"gtd","expire_time":"2026-10-10T00:00:00Z"`,
			orderID: "OYD1DW-N84FE-0IZF0B",
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Sell, TimeInForce: order.GoodTillTime, Price: 90000, Amount: 0.5, Status: order.New},
		},
		{
			name:    "stop loss order",
			submit:  &order.Submit{Type: order.Stop, Side: order.Sell, TriggerPrice: 75000, Amount: 0.25},
			params:  `"order_type":"stop-loss","side":"sell","order_qty":0.25,"triggers":{"reference":"last","price":75000}`,
			orderID: "OY7Q2T-47IUR-JMWFFS",
			exp:     &order.SubmitResponse{Type: order.Stop, Side: order.Sell, Amount: 0.25, TriggerPrice: 75000, Status: order.New},
		},
		{
			name:    "reduce only stop loss limit order on the index price",
			submit:  &order.Submit{Type: order.StopLimit, Side: order.Buy, Price: 90500, TriggerPrice: 90000, TriggerPriceType: order.IndexPrice, Amount: 0.1, ReduceOnly: true},
			params:  `"order_type":"stop-loss-limit","side":"buy","order_qty":0.1,"limit_price":90500,"triggers":{"reference":"index","price":90000},"reduce_only":true`,
			orderID: "OSRZMP-OYD5R-4AEIS4",
			exp:     &order.SubmitResponse{Type: order.StopLimit, Side: order.Buy, ReduceOnly: true, Price: 90500, Amount: 0.1, TriggerPrice: 90000, Status: order.New},
		},
		{
			name:    "take profit market order",
			submit:  &order.Submit{Type: order.TakeProfitMarket, Side: order.Sell, TriggerPrice: 95000, Amount: 0.2},
			params:  `"order_type":"take-profit","side":"sell","order_qty":0.2,"triggers":{"reference":"last","price":95000}`,
			orderID: "OJ3ON6-HT2FG-S3036Z",
			exp:     &order.SubmitResponse{Type: order.TakeProfitMarket, Side: order.Sell, Amount: 0.2, TriggerPrice: 95000, Status: order.New},
		},
		{
			name:    "take profit limit order",
			submit:  &order.Submit{Type: order.TakeProfit | order.Limit, Side: order.Sell, Price: 94800, TriggerPrice: 95000, Amount: 0.2},
			params:  `"order_type":"take-profit-limit","side":"sell","order_qty":0.2,"limit_price":94800,"triggers":{"reference":"last","price":95000}`,
			orderID: "O2YB4X-9MSBG-WGMMQX",
			exp:     &order.SubmitResponse{Type: order.TakeProfit | order.Limit, Side: order.Sell, Price: 94800, Amount: 0.2, TriggerPrice: 95000, Status: order.New},
		},
		{
			name:    "trailing stop order by percentage",
			submit:  &order.Submit{Type: order.TrailingStop, Side: order.Sell, Amount: 0.3, TrackingMode: order.Percentage, TrackingValue: 2.5},
			params:  `"order_type":"trailing-stop","side":"sell","order_qty":0.3,"triggers":{"reference":"last","price":2.5,"price_type":"pct"}`,
			orderID: "OELZN4-5N04J-M3BR2D",
			exp:     &order.SubmitResponse{Type: order.TrailingStop, Side: order.Sell, Amount: 0.3, Status: order.New},
		},
		{
			name:    "trailing stop order by distance on the index price",
			submit:  &order.Submit{Type: order.TrailingStop, Side: order.Buy, Amount: 0.3, TrackingMode: order.Distance, TrackingValue: 1500, TriggerPriceType: order.IndexPrice},
			params:  `"order_type":"trailing-stop","side":"buy","order_qty":0.3,"triggers":{"reference":"index","price":1500,"price_type":"quote"}`,
			orderID: "O4TULE-1NM2Q-HXXCZH",
			exp:     &order.SubmitResponse{Type: order.TrailingStop, Side: order.Buy, Amount: 0.3, Status: order.New},
		},
	} {
		t.Run(tc.name+" over the private websocket", func(t *testing.T) {
			t.Parallel()
			ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{
				request: ordersWsRequest("add_order", tc.params+`,"symbol":"BTC/USD"`),
				replies: []string{ordersWsReply("add_order", `{"order_id":"`+tc.orderID+`"}`)},
			})
			tc.submit.Exchange, tc.submit.Pair, tc.submit.AssetType = ex.Name, spotTestPair, asset.Spot
			submitted := time.Now()
			resp, err := ex.submitSpotOrder(t.Context(), tc.submit)
			require.NoError(t, err, "submitSpotOrder must not error")
			ordersClearSubmitTimes(t, resp, submitted)
			tc.exp.Exchange, tc.exp.Pair, tc.exp.AssetType, tc.exp.OrderID = ex.Name, spotTestPair, asset.Spot, tc.orderID
			assert.Equal(t, tc.exp, resp, "submitSpotOrder should place the order over the private websocket")
		})
	}

	t.Run("rejected over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{
			request: ordersWsRequest("add_order", `"order_type":"limit","side":"buy","order_qty":1.2,"limit_price":26500.4,"symbol":"BTC/USD"`),
			replies: []string{ordersWsRejection("add_order", "EOrder:Insufficient funds")},
		})
		_, err := ex.submitSpotOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 26500.4, Amount: 1.2})
		require.ErrorIs(t, err, errAPIResponse, "submitSpotOrder must return the rejection as an API error")
		assert.ErrorContains(t, err, "EOrder:Insufficient funds", "the error should carry Kraken's reason")
	})

	t.Run("REST reply without a transaction ID", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/private/AddOrder", r.URL.Path, "submitSpotOrder should call Add Order")
			assert.JSONEq(t, `{"ordertype":"stop-loss","type":"sell","volume":"0.25","price":"75000","trigger":"last","pair":"XBTUSD"}`, tradingRequestBody(t, r), "submitSpotOrder should send the order")
			_, _ = w.Write([]byte(`{"error":[],"result":{"descr":{"order":"sell 0.25000000 XBTUSD @ stop loss 75000.0"}}}`))
		})
		_, err := ex.submitSpotOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.Stop, Side: order.Sell, TriggerPrice: 75000, Amount: 0.25})
		assert.ErrorIs(t, err, order.ErrPlaceFailed, "submitSpotOrder should report an order Kraken returned no transaction ID for as not placed")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			name:   "stop loss order",
			submit: &order.Submit{Type: order.Stop, Side: order.Sell, TriggerPrice: 75000, Amount: 0.25},
			exp:    &order.SubmitResponse{Type: order.Stop, Side: order.Sell, Amount: 0.25, TriggerPrice: 75000, Status: order.New, OrderID: "OM3IVF-M83OG-3QQVT8"},
		},
		{
			name:   "reduce only stop loss market order with a client order ID",
			submit: &order.Submit{Type: order.StopMarket, Side: order.Sell, TriggerPrice: 74000, Amount: 0.5, ReduceOnly: true, ClientOrderID: "gct-stop-market-1"},
			exp: &order.SubmitResponse{
				Type:          order.StopMarket,
				Side:          order.Sell,
				ReduceOnly:    true,
				Amount:        0.5,
				TriggerPrice:  74000,
				ClientOrderID: "gct-stop-market-1",
				Status:        order.New,
				OrderID:       "OYY4C7-3ZM21-M7RV93",
			},
		},
		{
			name:   "good till time stop loss limit order on the index price",
			submit: &order.Submit{Type: order.StopLimit, Side: order.Buy, Price: 90500, TriggerPrice: 90000, TriggerPriceType: order.IndexPrice, Amount: 0.1, TimeInForce: order.GoodTillTime, EndTime: ordersTestExpiry},
			exp: &order.SubmitResponse{
				Type:         order.StopLimit,
				Side:         order.Buy,
				TimeInForce:  order.GoodTillTime,
				Price:        90500,
				Amount:       0.1,
				TriggerPrice: 90000,
				Status:       order.New,
				OrderID:      "OXVM50-FN48W-18YG0H",
			},
		},
		{
			name:   "take profit order",
			submit: &order.Submit{Type: order.TakeProfit, Side: order.Sell, TriggerPrice: 95000, Amount: 0.2},
			exp:    &order.SubmitResponse{Type: order.TakeProfit, Side: order.Sell, Amount: 0.2, TriggerPrice: 95000, Status: order.New, OrderID: "OG0AY1-66MB6-W8XFSL"},
		},
		{
			name:   "take profit market order on the index price",
			submit: &order.Submit{Type: order.TakeProfitMarket, Side: order.Buy, TriggerPrice: 70000, TriggerPriceType: order.IndexPrice, Amount: 0.2},
			exp:    &order.SubmitResponse{Type: order.TakeProfitMarket, Side: order.Buy, Amount: 0.2, TriggerPrice: 70000, Status: order.New, OrderID: "OU8BAY-V7GMS-7FHIZP"},
		},
		{
			name:   "immediate or cancel take profit limit order",
			submit: &order.Submit{Type: order.TakeProfit | order.Limit, Side: order.Sell, Price: 94800, TriggerPrice: 95000, Amount: 0.2, TimeInForce: order.ImmediateOrCancel},
			exp: &order.SubmitResponse{
				Type:         order.TakeProfit | order.Limit,
				Side:         order.Sell,
				TimeInForce:  order.ImmediateOrCancel,
				Price:        94800,
				Amount:       0.2,
				TriggerPrice: 95000,
				Status:       order.New,
				OrderID:      "OXCEYT-TOWNO-UY89O5",
			},
		},
		{
			name:   "trailing stop order by percentage",
			submit: &order.Submit{Type: order.TrailingStop, Side: order.Sell, Amount: 0.3, TrackingMode: order.Percentage, TrackingValue: 2.5},
			exp:    &order.SubmitResponse{Type: order.TrailingStop, Side: order.Sell, Amount: 0.3, Status: order.New, OrderID: "OZHICX-5LU6D-JJX22M"},
		},
		{
			name:   "trailing stop order by distance on the index price",
			submit: &order.Submit{Type: order.TrailingStop, Side: order.Buy, Amount: 0.3, TrackingMode: order.Distance, TrackingValue: 1500, TriggerPriceType: order.IndexPrice},
			exp:    &order.SubmitResponse{Type: order.TrailingStop, Side: order.Buy, Amount: 0.3, Status: order.New, OrderID: "O67Q7H-A6HTI-P4Y25B"},
		},
	} {
		t.Run(tc.name+" over REST", func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange, tc.submit.Pair, tc.submit.AssetType = e.Name, spotTestPair, asset.Spot
			submitted := time.Now()
			resp, err := e.submitSpotOrder(t.Context(), tc.submit)
			require.NoError(t, err, "submitSpotOrder must not error")
			if !mockTests {
				assert.NotEmpty(t, resp.OrderID, "submitSpotOrder should return the order ID")
				return
			}
			ordersClearSubmitTimes(t, resp, submitted)
			tc.exp.Exchange, tc.exp.Pair, tc.exp.AssetType = e.Name, spotTestPair, asset.Spot
			assert.Equal(t, tc.exp, resp, "submitSpotOrder should place the order over REST")
		})
	}
	if !mockTests {
		return
	}
	_, err := e.submitSpotOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.Stop, Side: order.Sell, TriggerPrice: 75000, Amount: 25})
	require.ErrorIs(t, err, errAPIResponse, "submitSpotOrder must return Kraken's rejection as an API error")
	assert.ErrorContains(t, err, "EOrder:Insufficient funds", "the error should carry Kraken's reason")
}

func TestSpotOrderTypeString(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[order.Type]string{
		order.Market:                   "market",
		order.Limit:                    "limit",
		order.Stop:                     "stop-loss",
		order.StopMarket:               "stop-loss",
		order.StopLimit:                "stop-loss-limit",
		order.TakeProfit:               "take-profit",
		order.TakeProfitMarket:         "take-profit",
		order.TakeProfit | order.Limit: "take-profit-limit",
		order.TrailingStop:             "trailing-stop",
	} {
		got, err := spotOrderTypeString(orderType)
		require.NoErrorf(t, err, "spotOrderTypeString must not error for %s", orderType)
		assert.Equalf(t, exp, got, "spotOrderTypeString should convert %s", orderType)
	}
	for _, orderType := range []order.Type{order.OCO, order.IOS, order.Liquidation} {
		_, err := spotOrderTypeString(orderType)
		assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "spotOrderTypeString should reject %s", orderType)
	}
}

func TestSpotTimeInForceString(t *testing.T) {
	t.Parallel()
	for tif, exp := range map[order.TimeInForce]string{
		order.UnknownTIF:                      "",
		order.GoodTillCancel:                  "",
		order.PostOnly:                        "",
		order.GoodTillCancel | order.PostOnly: "",
		order.ImmediateOrCancel:               timeInForceIOC,
		order.FillOrKill:                      timeInForceFOK,
		order.GoodTillTime:                    timeInForceGTD,
		order.GoodTillTime | order.PostOnly:   timeInForceGTD,
	} {
		got, err := spotTimeInForceString(tif)
		require.NoErrorf(t, err, "spotTimeInForceString must not error for %s", tif)
		assert.Equalf(t, exp, got, "spotTimeInForceString should convert %s", tif)
	}
	for _, tif := range []order.TimeInForce{order.GoodTillDay, order.GoodTillCrossing, order.StopOrReduce} {
		_, err := spotTimeInForceString(tif)
		assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "spotTimeInForceString should reject %s", tif)
	}
}

func TestSpotTriggerReference(t *testing.T) {
	t.Parallel()
	for priceType, exp := range map[order.PriceType]string{
		order.UnknownPriceType: "",
		order.LastPrice:        "last",
		order.IndexPrice:       "index",
	} {
		got, err := spotTriggerReference(priceType)
		require.NoErrorf(t, err, "spotTriggerReference must not error for %s", priceType)
		assert.Equalf(t, exp, got, "spotTriggerReference should convert %s", priceType)
	}
	_, err := spotTriggerReference(order.MarkPrice)
	assert.ErrorIs(t, err, order.ErrUnknownPriceType, "spotTriggerReference should reject the mark price, which spot orders cannot trigger on")
}

func TestSubmitFuturesOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			name:   "stop order",
			submit: &order.Submit{Type: order.Stop, Side: order.Sell, TriggerPrice: 79000, Amount: 0.001},
			exp:    &order.SubmitResponse{Type: order.Stop, Side: order.Sell, Amount: 0.001, TriggerPrice: 79000, Status: order.New, OrderID: "770a8a11-d8a4-4525-b292-8b3d4ee2e308"},
		},
		{
			name: "reduce only stop limit order on the mark price with a client order ID",
			submit: &order.Submit{
				Type:             order.StopLimit,
				Side:             order.Sell,
				Price:            78900,
				TriggerPrice:     79000,
				TriggerPriceType: order.MarkPrice,
				Amount:           0.002,
				ReduceOnly:       true,
				ClientOrderID:    "gct-stop-limit-1",
			},
			exp: &order.SubmitResponse{
				Type:          order.StopLimit,
				Side:          order.Sell,
				ReduceOnly:    true,
				Price:         78900,
				Amount:        0.002,
				TriggerPrice:  79000,
				ClientOrderID: "gct-stop-limit-1",
				Status:        order.New,
				OrderID:       "517184b8-317e-4cd6-a794-ef3e69ee42bf",
			},
		},
		{
			name:   "stop market order on the index price",
			submit: &order.Submit{Type: order.StopMarket, Side: order.Buy, TriggerPrice: 84000, TriggerPriceType: order.IndexPrice, Amount: 0.001},
			exp:    &order.SubmitResponse{Type: order.StopMarket, Side: order.Buy, Amount: 0.001, TriggerPrice: 84000, Status: order.New, OrderID: "63d3a362-edfc-4f1b-b420-8a0b172bac98"},
		},
		{
			name:   "take profit order",
			submit: &order.Submit{Type: order.TakeProfit, Side: order.Sell, TriggerPrice: 86000, Amount: 0.001},
			exp:    &order.SubmitResponse{Type: order.TakeProfit, Side: order.Sell, Amount: 0.001, TriggerPrice: 86000, Status: order.New, OrderID: "cd1adc0e-2cbe-4c9c-9b0c-7526b5c7c567"},
		},
		{
			name:   "take profit market order",
			submit: &order.Submit{Type: order.TakeProfitMarket, Side: order.Buy, TriggerPrice: 76000, Amount: 0.001},
			exp:    &order.SubmitResponse{Type: order.TakeProfitMarket, Side: order.Buy, Amount: 0.001, TriggerPrice: 76000, Status: order.New, OrderID: "f6564232-dcf9-48c5-803b-8ae160fb6d85"},
		},
		{
			name:   "take profit limit order on the index price",
			submit: &order.Submit{Type: order.TakeProfit | order.Limit, Side: order.Sell, Price: 85900, TriggerPrice: 86000, TriggerPriceType: order.IndexPrice, Amount: 0.001},
			exp:    &order.SubmitResponse{Type: order.TakeProfit | order.Limit, Side: order.Sell, Price: 85900, Amount: 0.001, TriggerPrice: 86000, Status: order.New, OrderID: "4fd39d86-4af5-4b7a-9bc5-7495be737079"},
		},
		{
			name:   "trailing stop order by distance",
			submit: &order.Submit{Type: order.TrailingStop, Side: order.Sell, Amount: 0.002, TrackingMode: order.Distance, TrackingValue: 300},
			exp:    &order.SubmitResponse{Type: order.TrailingStop, Side: order.Sell, Amount: 0.002, Status: order.New, OrderID: "faf96f13-3aef-495d-8b35-a7dbe3417f93"},
		},
		{
			name:   "trailing stop order by percentage on the mark price",
			submit: &order.Submit{Type: order.TrailingStop, Side: order.Buy, Amount: 0.002, TrackingMode: order.Percentage, TrackingValue: 1.5, TriggerPriceType: order.MarkPrice},
			exp:    &order.SubmitResponse{Type: order.TrailingStop, Side: order.Buy, Amount: 0.002, Status: order.New, OrderID: "6bbd67fd-65cd-4527-88cf-39ebed970483"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange, tc.submit.Pair, tc.submit.AssetType = e.Name, futuresTestPair, asset.Futures
			submitted := time.Now()
			resp, err := e.submitFuturesOrder(t.Context(), tc.submit)
			require.NoError(t, err, "submitFuturesOrder must not error")
			if !mockTests {
				assert.NotEmpty(t, resp.OrderID, "submitFuturesOrder should return the order ID")
				return
			}
			ordersClearSubmitTimes(t, resp, submitted)
			tc.exp.Exchange, tc.exp.Pair, tc.exp.AssetType = e.Name, futuresTestPair, asset.Futures
			assert.Equal(t, tc.exp, resp, "submitFuturesOrder should place the order")
		})
	}
	if !mockTests {
		return
	}
	_, err := e.submitFuturesOrder(t.Context(), &order.Submit{
		Exchange:      e.Name,
		Pair:          futuresTestPair,
		AssetType:     asset.Futures,
		Type:          order.TrailingStop,
		Side:          order.Sell,
		Amount:        0.002,
		TrackingMode:  order.Percentage,
		TrackingValue: 1.5,
		ClientOrderID: "gct-trailing-1",
	})
	require.ErrorIs(t, err, order.ErrPlaceFailed, "submitFuturesOrder must report a placement Kraken refused as failed")
	assert.ErrorContains(t, err, "insufficientAvailableFunds", "the error should carry Kraken's status")
}

func TestFuturesOrderTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       string
	}{
		{order.Limit, order.UnknownTIF, "lmt"},
		{order.Limit, order.GoodTillCancel, "lmt"},
		{order.Limit, order.PostOnly, "post"},
		{order.Limit, order.GoodTillCancel | order.PostOnly, "post"},
		{order.Limit, order.ImmediateOrCancel, "ioc"},
		{order.Limit, order.FillOrKill, "fok"},
		{order.Market, order.UnknownTIF, "mkt"},
		{order.Market, order.ImmediateOrCancel, "mkt"},
		{order.Stop, order.UnknownTIF, "stp"},
		{order.StopMarket, order.UnknownTIF, "stp"},
		{order.StopLimit, order.UnknownTIF, "stp"},
		{order.TakeProfit, order.UnknownTIF, "take_profit"},
		{order.TakeProfitMarket, order.UnknownTIF, "take_profit"},
		{order.TakeProfit | order.Limit, order.UnknownTIF, "take_profit"},
		{order.TrailingStop, order.UnknownTIF, "trailing_stop"},
	} {
		got, err := futuresOrderTypeString(tc.orderType, tc.tif)
		require.NoErrorf(t, err, "futuresOrderTypeString must not error for %s %s", tc.orderType, tc.tif)
		assert.Equalf(t, tc.exp, got, "futuresOrderTypeString should convert %s %s", tc.orderType, tc.tif)
	}
	for _, orderType := range []order.Type{order.OCO, order.IOS, order.Liquidation} {
		_, err := futuresOrderTypeString(orderType, order.UnknownTIF)
		assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "futuresOrderTypeString should reject %s", orderType)
	}
}

func TestFuturesTriggerSignal(t *testing.T) {
	t.Parallel()
	for priceType, exp := range map[order.PriceType]string{
		order.UnknownPriceType: "",
		order.LastPrice:        "last",
		order.MarkPrice:        "mark",
		order.IndexPrice:       "index",
	} {
		got, err := futuresTriggerSignal(priceType)
		require.NoErrorf(t, err, "futuresTriggerSignal must not error for %s", priceType)
		assert.Equalf(t, exp, got, "futuresTriggerSignal should convert %s", priceType)
	}
	_, err := futuresTriggerSignal(order.IndexPrice | order.MarkPrice)
	assert.ErrorIs(t, err, order.ErrUnknownPriceType, "futuresTriggerSignal should reject a price type that is not one price")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		err    error
	}{
		{"a modification without a pair", &order.Modify{AssetType: asset.Spot, OrderID: "OHYO67-6LP66-HMQ437", Amount: 1}, order.ErrPairIsEmpty},
		{"a modification without an asset", &order.Modify{Pair: spotTestPair, OrderID: "OHYO67-6LP66-HMQ437", Amount: 1}, order.ErrAssetNotSet},
		{"a modification without an order identifier", &order.Modify{Pair: spotTestPair, AssetType: asset.Spot, Amount: 1}, order.ErrOrderIDNotSet},
		{"an unsupported asset", &order.Modify{Pair: spotTestPair, AssetType: asset.Margin, OrderID: "OHYO67-6LP66-HMQ437", Amount: 1}, asset.ErrNotSupported},
		{"a negative spot quantity", &order.Modify{Pair: spotTestPair, AssetType: asset.Spot, OrderID: "OHYO67-6LP66-HMQ437", Amount: -1}, order.ErrAmountIsInvalid},
	} {
		_, err = e.ModifyOrder(t.Context(), tc.modify)
		assert.ErrorIsf(t, err, tc.err, "ModifyOrder should reject %s", tc.name)
	}

	for _, tc := range []struct {
		name   string
		modify *order.Modify
		trip   ordersWsRoundTrip
		exp    *order.ModifyResponse
	}{
		{
			name: "by order ID",
			modify: &order.Modify{
				OrderID:       "OAIYAU-LGI3M-PFM5VW",
				ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
				Type:          order.StopLimit,
				Side:          order.Buy,
				Amount:        1.2,
				Price:         61031.3,
				TriggerPrice:  61000,
			},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("amend_order", `"order_id":"OAIYAU-LGI3M-PFM5VW","order_qty":1.2,"limit_price":61031.3,"trigger_price":61000`),
				replies: []string{ordersWsReply("amend_order", `{"amend_id":"TTW6PD-RC36L-ZZSWNU","order_id":"OAIYAU-LGI3M-PFM5VW","cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9"}`)},
			},
			exp: &order.ModifyResponse{
				OrderID:       "OAIYAU-LGI3M-PFM5VW",
				ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
				Type:          order.StopLimit,
				Side:          order.Buy,
				Price:         61031.3,
				Amount:        1.2,
				TriggerPrice:  61000,
			},
		},
		{
			name:   "by client order ID",
			modify: &order.Modify{ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9", Type: order.Limit, Side: order.Buy, Amount: 1.2, Price: 490795},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("amend_order", `"cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9","order_qty":1.2,"limit_price":490795`),
				replies: []string{ordersWsReply("amend_order", `{"amend_id":"TTW6PD-RC36L-ZZSWNU","cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9"}`)},
			},
			exp: &order.ModifyResponse{ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9", Type: order.Limit, Side: order.Buy, Price: 490795, Amount: 1.2},
		},
		{
			name:   "post only price change",
			modify: &order.Modify{OrderID: "OAIYAU-LGI3M-PFM5VW", Type: order.Limit, Side: order.Buy, TimeInForce: order.PostOnly, Price: 61031.3},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("amend_order", `"order_id":"OAIYAU-LGI3M-PFM5VW","limit_price":61031.3,"post_only":true`),
				replies: []string{ordersWsReply("amend_order", `{"amend_id":"TTW6PD-RC36L-ZZSWNU","order_id":"OAIYAU-LGI3M-PFM5VW"}`)},
			},
			exp: &order.ModifyResponse{OrderID: "OAIYAU-LGI3M-PFM5VW", Type: order.Limit, Side: order.Buy, TimeInForce: order.PostOnly, Price: 61031.3},
		},
		{
			name:   "post only good till cancelled price change",
			modify: &order.Modify{OrderID: "OAIYAU-LGI3M-PFM5VW", Type: order.Limit, Side: order.Buy, TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 61031.3},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("amend_order", `"order_id":"OAIYAU-LGI3M-PFM5VW","limit_price":61031.3,"post_only":true`),
				replies: []string{ordersWsReply("amend_order", `{"amend_id":"TTW6PD-RC36L-ZZSWNU","order_id":"OAIYAU-LGI3M-PFM5VW"}`)},
			},
			exp: &order.ModifyResponse{OrderID: "OAIYAU-LGI3M-PFM5VW", Type: order.Limit, Side: order.Buy, TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 61031.3},
		},
		{
			// Kraken takes post_only with a limit price change alone
			name:   "post only quantity change",
			modify: &order.Modify{OrderID: "OAIYAU-LGI3M-PFM5VW", Type: order.Limit, Side: order.Buy, TimeInForce: order.PostOnly, Amount: 1.2},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("amend_order", `"order_id":"OAIYAU-LGI3M-PFM5VW","order_qty":1.2`),
				replies: []string{ordersWsReply("amend_order", `{"amend_id":"TTW6PD-RC36L-ZZSWNU","order_id":"OAIYAU-LGI3M-PFM5VW"}`)},
			},
			exp: &order.ModifyResponse{OrderID: "OAIYAU-LGI3M-PFM5VW", Type: order.Limit, Side: order.Buy, TimeInForce: order.PostOnly, Amount: 1.2},
		},
	} {
		t.Run("spot "+tc.name+" over the private websocket", func(t *testing.T) {
			t.Parallel()
			ex := newOrdersWsTestExchange(t, tc.trip)
			tc.modify.Exchange, tc.modify.Pair, tc.modify.AssetType = ex.Name, spotTestPair, asset.Spot
			resp, err := ex.ModifyOrder(t.Context(), tc.modify)
			require.NoError(t, err, "ModifyOrder must not error")
			tc.exp.Exchange, tc.exp.Pair, tc.exp.AssetType = ex.Name, spotTestPair, asset.Spot
			assert.Equal(t, tc.exp, resp, "ModifyOrder should amend the order over the private websocket")
		})
	}

	for _, tc := range []struct {
		name   string
		modify order.Modify
		body   string
	}{
		{"post only price change", order.Modify{Price: 2655.5, TimeInForce: order.PostOnly}, `{"txid":"OHYO67-6LP66-HMQ437","limit_price":"2655.5","post_only":true}`},
		{"post only good till cancelled price change", order.Modify{Price: 2655.5, TimeInForce: order.GoodTillCancel | order.PostOnly}, `{"txid":"OHYO67-6LP66-HMQ437","limit_price":"2655.5","post_only":true}`},
		// Kraken takes post_only with a limit price change alone
		{"post only quantity change", order.Modify{Amount: 8, TimeInForce: order.PostOnly}, `{"txid":"OHYO67-6LP66-HMQ437","order_qty":"8"}`},
	} {
		t.Run("spot "+tc.name+" over REST", func(t *testing.T) {
			t.Parallel()
			ex := newOrdersRequestCheckExchange(t, "/0/private/AmendOrder", tc.body, `{"error":[],"result":{"amend_id":"TTW6PD-RC36L-ZZSWNU"}}`)
			tc.modify.Exchange, tc.modify.Pair, tc.modify.AssetType, tc.modify.OrderID = ex.Name, spotTestPair, asset.Spot, "OHYO67-6LP66-HMQ437"
			_, err := ex.ModifyOrder(t.Context(), &tc.modify)
			require.NoError(t, err, "ModifyOrder must not error")
		})
	}

	t.Run("futures edit failing", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersFailingExchange(t)
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: futuresTestPair, AssetType: asset.Futures, OrderID: "4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b", Price: 81520})
		assert.ErrorIs(t, err, errAPIResponse, "ModifyOrder should return Edit order's error")
	})

	t.Run("spot order rejected over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{
			request: ordersWsRequest("amend_order", `"order_id":"OZJW0T-MIQ3C-5NATAU","order_qty":1`),
			replies: []string{ordersWsRejection("amend_order", "EOrder:Unknown order")},
		})
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, OrderID: "OZJW0T-MIQ3C-5NATAU", Amount: 1})
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder should report an order Kraken does not know as not found")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		exp    *order.ModifyResponse
	}{
		{
			name: "spot order by order ID over REST",
			modify: &order.Modify{
				Exchange:      e.Name,
				OrderID:       "OHYO67-6LP66-HMQ437",
				ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876",
				Type:          order.Limit,
				Side:          order.Sell,
				AssetType:     asset.Spot,
				Pair:          ordersETHUSDPair,
				Price:         2655.5,
				Amount:        8,
			},
			exp: &order.ModifyResponse{
				Exchange:      e.Name,
				OrderID:       "OHYO67-6LP66-HMQ437",
				ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876",
				Pair:          ordersETHUSDPair,
				Type:          order.Limit,
				Side:          order.Sell,
				AssetType:     asset.Spot,
				Price:         2655.5,
				Amount:        8,
			},
		},
		{
			name:   "spot order by client order ID over REST",
			modify: &order.Modify{Exchange: e.Name, ClientOrderID: "gct-stop-market-1", Type: order.StopMarket, Side: order.Sell, AssetType: asset.Spot, Pair: spotTestPair, TriggerPrice: 73500},
			exp: &order.ModifyResponse{
				Exchange:      e.Name,
				ClientOrderID: "gct-stop-market-1",
				Pair:          spotTestPair,
				Type:          order.StopMarket,
				Side:          order.Sell,
				AssetType:     asset.Spot,
				TriggerPrice:  73500,
			},
		},
		{
			name:   "futures order's prices",
			modify: &order.Modify{Exchange: e.Name, ClientOrderID: "gct-stop-3", Type: order.StopLimit, Side: order.Sell, AssetType: asset.Futures, Pair: futuresTestPair, Price: 79000, TriggerPrice: 79100},
			exp: &order.ModifyResponse{
				Exchange:      e.Name,
				ClientOrderID: "gct-stop-3",
				Pair:          futuresTestPair,
				Type:          order.StopLimit,
				Side:          order.Sell,
				AssetType:     asset.Futures,
				Price:         79000,
				TriggerPrice:  79100,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.ModifyOrder(t.Context(), tc.modify)
			require.NoError(t, err, "ModifyOrder must not error")
			assert.Equal(t, tc.exp, resp, "ModifyOrder should return the modification")
		})
	}
	if !mockTests {
		return
	}
	t.Run("spot order Kraken does not know over REST", func(t *testing.T) {
		t.Parallel()
		_, err := e.ModifyOrder(t.Context(), &order.Modify{Exchange: e.Name, Pair: spotTestPair, AssetType: asset.Spot, OrderID: "OZJW0T-MIQ3C-5NATAU", Amount: 1})
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder should report an order Kraken does not know as not found")
	})
	t.Run("futures edit Kraken refuses", func(t *testing.T) {
		t.Parallel()
		_, err := e.ModifyOrder(t.Context(), &order.Modify{Exchange: e.Name, Pair: futuresTestPair, AssetType: asset.Futures, OrderID: "4d75883e-9327-4827-9256-023fc4b50f35", Price: 81600})
		require.ErrorIs(t, err, errOrderEditFailed, "ModifyOrder must report an edit Kraken refused as failed")
		assert.ErrorContains(t, err, "orderForEditNotFound", "the error should carry Kraken's status")
	})
}

func TestClientOrderIDWithoutOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderID, clientOrderID, exp string
	}{
		{"OHYO67-6LP66-HMQ437", "6d1b345e-2821-40e2-ad83-4ecb18a06876", ""},
		{"OHYO67-6LP66-HMQ437", "", ""},
		{"", "6d1b345e-2821-40e2-ad83-4ecb18a06876", "6d1b345e-2821-40e2-ad83-4ecb18a06876"},
		{"", "", ""},
	} {
		assert.Equalf(t, tc.exp, clientOrderIDWithoutOrderID(tc.orderID, tc.clientOrderID), "clientOrderIDWithoutOrderID should send %q for order ID %q and client order ID %q", tc.exp, tc.orderID, tc.clientOrderID)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancellation")
	err := e.CancelOrder(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.Spot, Pair: spotTestPair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOrder must reject a cancellation without an order identifier")
	err = e.CancelOrder(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.Margin, Pair: spotTestPair, OrderID: "OYVGEW-VYV5B-UUEXSK"})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelOrder must reject an unsupported asset")

	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
		trip   ordersWsRoundTrip
		err    error
	}{
		{
			name:   "by order ID",
			cancel: &order.Cancel{OrderID: "OM5CRX-N2HAL-GFGWE9", ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876"},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("cancel_order", `"order_id":["OM5CRX-N2HAL-GFGWE9"]`),
				replies: []string{ordersWsReply("cancel_order", `{"order_id":"OM5CRX-N2HAL-GFGWE9","cl_ord_id":"6d1b345e-2821-40e2-ad83-4ecb18a06876"}`)},
			},
		},
		{
			name:   "by client order ID",
			cancel: &order.Cancel{ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876"},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("cancel_order", `"cl_ord_id":["6d1b345e-2821-40e2-ad83-4ecb18a06876"]`),
				replies: []string{ordersWsReply("cancel_order", `{"order_id":"OLUMT4-UTEGU-ZYM7E9","cl_ord_id":"6d1b345e-2821-40e2-ad83-4ecb18a06876"}`)},
			},
		},
		{
			name:   "Kraken does not know",
			cancel: &order.Cancel{OrderID: "OZJW0T-MIQ3C-5NATAU"},
			trip: ordersWsRoundTrip{
				request: ordersWsRequest("cancel_order", `"order_id":["OZJW0T-MIQ3C-5NATAU"]`),
				replies: []string{ordersWsRejection("cancel_order", "EOrder:Unknown order")},
			},
			err: order.ErrOrderNotFound,
		},
	} {
		t.Run("spot order "+tc.name+" over the private websocket", func(t *testing.T) {
			t.Parallel()
			ex := newOrdersWsTestExchange(t, tc.trip)
			tc.cancel.Exchange, tc.cancel.AssetType, tc.cancel.Pair = ex.Name, asset.Spot, spotTestPair
			err := ex.CancelOrder(t.Context(), tc.cancel)
			if tc.err == nil {
				assert.NoError(t, err, "CancelOrder should cancel the order")
				return
			}
			assert.ErrorIs(t, err, tc.err, "CancelOrder should return the cancellation's error")
		})
	}

	t.Run("spot cancellation failing over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/private/GetWebSocketsToken", r.URL.Path, "only the websocket token should be requested")
			_, _ = w.Write([]byte(`{"error":["EAPI:Invalid key"]}`))
		})
		useOrdersTestWebsocket(t, ex)
		ex.wsToken = ""
		err := ex.CancelOrder(t.Context(), &order.Cancel{Exchange: ex.Name, AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OM5CRX-N2HAL-GFGWE9"})
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "CancelOrder should return the request's error")
	})

	t.Run("futures cancellation failing", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersFailingExchange(t)
		err := ex.CancelOrder(t.Context(), &order.Cancel{Exchange: ex.Name, AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b"})
		assert.ErrorIs(t, err, errAPIResponse, "CancelOrder should return Cancel order's error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
		err    error
	}{
		{"spot order by order ID over REST", &order.Cancel{AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OYVGEW-VYV5B-UUEXSK", ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876"}, nil},
		{"spot order by client order ID over REST", &order.Cancel{AssetType: asset.Spot, Pair: spotTestPair, ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876"}, nil},
		{"spot order Kraken does not know over REST", &order.Cancel{AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OZJW0T-MIQ3C-5NATAU"}, order.ErrOrderNotFound},
		{"futures order by order ID", &order.Cancel{AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b", ClientOrderID: "gct-batch-1"}, nil},
		{"futures order by client order ID", &order.Cancel{AssetType: asset.Futures, Pair: futuresTestPair, ClientOrderID: "gct-stop-1"}, nil},
		{"futures order Kraken does not find", &order.Cancel{AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "9ac37735-a838-4323-9dd2-5c672bd97b54"}, order.ErrOrderNotFound},
		{"futures order already filled", &order.Cancel{AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "d31f46d3-c3c1-4411-ab5d-a6b1e81ea6c9"}, order.ErrCancelFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.cancel.Exchange = e.Name
			err := e.CancelOrder(t.Context(), tc.cancel)
			if tc.err == nil || !mockTests {
				assert.NoError(t, err, "CancelOrder should cancel the order")
				return
			}
			assert.ErrorIs(t, err, tc.err, "CancelOrder should return the cancellation's error")
		})
	}
}

func TestFuturesCancelStatusError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, futuresCancelStatusError("cancelled"), "futuresCancelStatusError should accept a cancelled order")
	assert.ErrorIs(t, futuresCancelStatusError("notFound"), order.ErrOrderNotFound, "futuresCancelStatusError should report an order Kraken did not find as not found")
	err := futuresCancelStatusError("filled")
	require.ErrorIs(t, err, order.ErrCancelFailed, "futuresCancelStatusError must report a filled order as not cancelled")
	assert.ErrorContains(t, err, "filled", "the error should carry Kraken's status")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		cancels []order.Cancel
		err     error
	}{
		{"an order without an identifier", []order.Cancel{{AssetType: asset.Spot, Pair: spotTestPair}}, order.ErrOrderIDNotSet},
		{"an order identified by its client order ID alone", []order.Cancel{{AssetType: asset.Spot, Pair: spotTestPair, ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876"}}, order.ErrOrderIDNotSet},
		{
			name: "an unsupported asset after a valid order",
			cancels: []order.Cancel{
				{AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OP5V2Y-RYKVL-ET3V3B"},
				{AssetType: asset.Margin, Pair: spotTestPair, OrderID: "OP5V2Y-7YKVL-ET3V3B"},
			},
			err: asset.ErrNotSupported,
		},
	} {
		_, err := e.CancelBatchOrders(t.Context(), tc.cancels)
		assert.ErrorIsf(t, err, tc.err, "CancelBatchOrders should reject %s before cancelling any order", tc.name)
	}

	t.Run("spot orders over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{
			request: ordersWsRequest("cancel_order", `"order_id":["OM5CRX-N2HAL-GFGWE9","OZJW0T-MIQ3C-5NATAU"]`),
			replies: []string{
				ordersWsReply("cancel_order", `{"order_id":"OM5CRX-N2HAL-GFGWE9"}`),
				ordersWsRejection("cancel_order", "EOrder:Unknown order"),
			},
		})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{Exchange: ex.Name, AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OM5CRX-N2HAL-GFGWE9"},
			{Exchange: ex.Name, AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OZJW0T-MIQ3C-5NATAU"},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		exp := &order.CancelBatchResponse{Status: map[string]string{
			"OM5CRX-N2HAL-GFGWE9": order.Cancelled.String(),
			"OZJW0T-MIQ3C-5NATAU": "API error response: EOrder:Unknown order",
		}}
		assert.Equal(t, exp, resp, "CancelBatchOrders should report each order's outcome")
	})

	t.Run("spot orders over the private websocket replied to in any order", func(t *testing.T) {
		t.Parallel()
		// Kraken's replies follow no set order, and only a cancellation's reply names its order
		const a, b, c = "OM5CRX-N2HAL-GFGWE9", "OZJW0T-MIQ3C-5NATAU", "OAIYAU-LGI3M-PFM5VW"
		unknown, internal := "API error response: EOrder:Unknown order", "API error response: EGeneral:Internal error"
		for _, tc := range []struct {
			name    string
			ids     []string
			replies []string
			exp     map[string]string
		}{
			{
				name:    "a rejection first",
				ids:     []string{a, b},
				replies: []string{ordersWsRejection("cancel_order", "EOrder:Unknown order"), ordersWsReply("cancel_order", `{"order_id":"`+a+`"}`)},
				exp:     map[string]string{a: order.Cancelled.String(), b: unknown},
			},
			{
				name: "rejections alone",
				ids:  []string{a, b, c},
				replies: []string{
					ordersWsRejection("cancel_order", "EOrder:Unknown order"), ordersWsRejection("cancel_order", "EOrder:Unknown order"),
					ordersWsRejection("cancel_order", "EGeneral:Internal error"),
				},
				exp: map[string]string{a: unknown + "; " + internal, b: unknown + "; " + internal, c: unknown + "; " + internal},
			},
			{
				name:    "a success naming no order",
				ids:     []string{a, b},
				replies: []string{ordersWsReply("cancel_order", `{}`), ordersWsReply("cancel_order", `{"order_id":"`+a+`"}`)},
				exp:     map[string]string{a: order.Cancelled.String(), b: order.ErrCancelFailed.Error() + ": no reply named the order"},
			},
		} {
			ids, err := json.Marshal(tc.ids)
			require.NoErrorf(t, err, "Marshal must not error for %s", tc.name)
			ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{request: ordersWsRequest("cancel_order", `"order_id":`+string(ids)), replies: tc.replies})
			cancels := make([]order.Cancel, len(tc.ids))
			for i, id := range tc.ids {
				cancels[i] = order.Cancel{Exchange: ex.Name, AssetType: asset.Spot, Pair: spotTestPair, OrderID: id}
			}
			resp, err := ex.CancelBatchOrders(t.Context(), cancels)
			require.NoErrorf(t, err, "CancelBatchOrders must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, resp.Status, "CancelBatchOrders should report each order's outcome for %s", tc.name)
		}
	})

	t.Run("futures batch reporting some orders", func(t *testing.T) {
		t.Parallel()
		// The reply reports the first order as cancelled and the second as not found, and leaves out the third
		ex := newOrdersRequestCheckExchange(t, "/derivatives/api/v3/batchorder",
			url.Values{"json": {`{"batchOrder":[{"order":"cancel","order_id":"e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b"},` +
				`{"order":"cancel","order_id":"9ac37735-a838-4323-9dd2-5c672bd97b54"},{"order":"cancel","order_id":"d31f46d3-c3c1-4411-ab5d-a6b1e81ea6c9"}]}`}}.Encode(),
			`{"batchStatus":[{"order_id":"e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b","orderEvents":[],"status":"cancelled"},`+
				`{"order_id":"9ac37735-a838-4323-9dd2-5c672bd97b54","orderEvents":[],"status":"notFound"}],"result":"success","serverTime":"2026-10-09T08:42:00.003Z"}`)
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{Exchange: ex.Name, AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b"},
			{Exchange: ex.Name, AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "9ac37735-a838-4323-9dd2-5c672bd97b54"},
			{Exchange: ex.Name, AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "d31f46d3-c3c1-4411-ab5d-a6b1e81ea6c9"},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		exp := &order.CancelBatchResponse{Status: map[string]string{
			"e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b": order.Cancelled.String(),
			"9ac37735-a838-4323-9dd2-5c672bd97b54": order.ErrOrderNotFound.Error(),
			"d31f46d3-c3c1-4411-ab5d-a6b1e81ea6c9": order.ErrCancelFailed.Error() + ": no status returned",
		}}
		assert.Equal(t, exp, resp, "CancelBatchOrders should report each order's outcome, and an order Kraken reports nothing for as not cancelled")
	})

	for _, tc := range []struct {
		name    string
		cancels []order.Cancel
	}{
		{"spot", []order.Cancel{{AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OP5V2Y-RYKVL-ET3V3B"}, {AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OP5V2Y-7YKVL-ET3V3B"}}},
		{"futures", []order.Cancel{{AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b"}}},
	} {
		t.Run(tc.name+" batch failing", func(t *testing.T) {
			t.Parallel()
			ex := newOrdersFailingExchange(t)
			_, err := ex.CancelBatchOrders(t.Context(), tc.cancels)
			assert.ErrorIs(t, err, errAPIResponse, "CancelBatchOrders should return the batch request's error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	spotOrders := []order.Cancel{
		{AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OP5V2Y-RYKVL-ET3V3B"},
		{AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OP5V2Y-7YKVL-ET3V3B"},
	}
	futuresOrders := []order.Cancel{
		{AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b"},
		{AssetType: asset.Futures, Pair: futuresTestPair, OrderID: "1a2b3c4d-5e6f-4a7b-9c8d-0e1f2a3b4c5d"},
	}
	for _, tc := range []struct {
		name    string
		cancels []order.Cancel
		exp     map[string]string
	}{
		{
			name:    "spot orders cancelled by one Cancel Order Batch",
			cancels: spotOrders,
			exp:     map[string]string{"OP5V2Y-RYKVL-ET3V3B": order.Cancelled.String(), "OP5V2Y-7YKVL-ET3V3B": order.Cancelled.String()},
		},
		{
			name:    "a spot order cancelled on its own",
			cancels: []order.Cancel{{AssetType: asset.Spot, Pair: spotTestPair, OrderID: "OYVGEW-VYV5B-UUEXSK"}},
			exp:     map[string]string{"OYVGEW-VYV5B-UUEXSK": order.Cancelled.String()},
		},
		{
			name:    "futures orders",
			cancels: futuresOrders,
			exp:     map[string]string{"e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b": order.Cancelled.String(), "1a2b3c4d-5e6f-4a7b-9c8d-0e1f2a3b4c5d": order.Cancelled.String()},
		},
		{
			name:    "spot and futures orders",
			cancels: slices.Concat(spotOrders, futuresOrders),
			exp: map[string]string{
				"OP5V2Y-RYKVL-ET3V3B":                  order.Cancelled.String(),
				"OP5V2Y-7YKVL-ET3V3B":                  order.Cancelled.String(),
				"e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b": order.Cancelled.String(),
				"1a2b3c4d-5e6f-4a7b-9c8d-0e1f2a3b4c5d": order.Cancelled.String(),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.CancelBatchOrders(t.Context(), tc.cancels)
			require.NoError(t, err, "CancelBatchOrders must not error")
			if !mockTests {
				assert.Len(t, resp.Status, len(tc.cancels), "CancelBatchOrders should report each order's outcome")
				return
			}
			assert.Equal(t, &order.CancelBatchResponse{Status: tc.exp}, resp, "CancelBatchOrders should report each order's outcome")
		})
	}
}

func TestCancelSpotOrders(t *testing.T) {
	t.Parallel()
	ids := make([]string, 51)
	for i := range ids {
		ids[i] = "OC" + string(rune('A'+i/26)) + string(rune('A'+i%26)) + "XY-PZQ2L-VF3GJA"
	}
	firstChunk, err := json.Marshal(ids[:50])
	require.NoError(t, err, "Marshal must not error")

	t.Run("in chunks of 50 over the private websocket", func(t *testing.T) {
		t.Parallel()
		replies := make([]string, 50)
		for i, id := range ids[:50] {
			replies[i] = ordersWsReply("cancel_order", `{"order_id":"`+id+`"}`)
		}
		ex := newOrdersWsTestExchange(t,
			ordersWsRoundTrip{request: ordersWsRequest("cancel_order", `"order_id":`+string(firstChunk)), replies: replies},
			ordersWsRoundTrip{
				request: ordersWsRequest("cancel_order", `"order_id":["`+ids[50]+`"]`),
				replies: []string{ordersWsRejection("cancel_order", "EOrder:Unknown order")},
			},
		)
		status := make(map[string]string)
		require.NoError(t, ex.cancelSpotOrders(t.Context(), ids, status), "cancelSpotOrders must not error")
		exp := make(map[string]string, len(ids))
		for _, id := range ids[:50] {
			exp[id] = order.Cancelled.String()
		}
		exp[ids[50]] = "API error response: EOrder:Unknown order"
		assert.Equal(t, exp, status, "cancelSpotOrders should record each order's outcome")
	})

	t.Run("request failing over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/private/GetWebSocketsToken", r.URL.Path, "only the websocket token should be requested")
			_, _ = w.Write([]byte(`{"error":["EAPI:Invalid key"]}`))
		})
		useOrdersTestWebsocket(t, ex)
		ex.wsToken = ""
		status := make(map[string]string)
		err := ex.cancelSpotOrders(t.Context(), ids[:2], status)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "cancelSpotOrders must return the request's error")
		assert.Empty(t, status, "cancelSpotOrders should record no outcome for orders it could not ask Kraken to cancel")
	})

	t.Run("in chunks of 50 over REST", func(t *testing.T) {
		t.Parallel()
		var (
			mu    sync.Mutex
			paths []string
		)
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			body := tradingRequestBody(t, r)
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			switch r.URL.Path {
			case "/0/private/CancelOrderBatch":
				assert.JSONEq(t, `{"orders":`+string(firstChunk)+`}`, body, "Cancel Order Batch should cancel the first 50 orders")
				_, _ = w.Write([]byte(`{"error":[],"result":{"count":50}}`))
			case "/0/private/CancelOrder":
				assert.JSONEq(t, `{"txid":"`+ids[50]+`"}`, body, "Cancel Order should cancel the order left over")
				_, _ = w.Write([]byte(`{"error":[],"result":{"count":1}}`))
			}
		})
		status := make(map[string]string)
		require.NoError(t, ex.cancelSpotOrders(t.Context(), ids, status), "cancelSpotOrders must not error")
		exp := make(map[string]string, len(ids))
		for _, id := range ids {
			exp[id] = order.Cancelled.String()
		}
		assert.Equal(t, exp, status, "cancelSpotOrders should record each order as cancelled")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/0/private/CancelOrderBatch", "/0/private/CancelOrder"}, paths, "cancelSpotOrders should cancel the second chunk's only order on its own")
	})

	t.Run("batch cancelling none of its orders over REST", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			_ = tradingRequestBody(t, r)
			switch r.URL.Path {
			case "/0/private/CancelOrderBatch":
				_, _ = w.Write([]byte(`{"error":[],"result":{"count":0}}`))
			case "/0/private/CancelOrder":
				_, _ = w.Write([]byte(`{"error":["EOrder:Unknown order"]}`))
			case "/0/private/QueryOrders":
				_, _ = w.Write([]byte(`{"error":[],"result":{}}`))
			default:
				assert.Failf(t, "only Cancel Order Batch, Cancel Order and Query Orders Info should be called", "%s", r.URL.Path)
			}
		})
		status := make(map[string]string)
		require.NoError(t, ex.cancelSpotOrders(t.Context(), ids[:2], status), "cancelSpotOrders must not error")
		exp := map[string]string{ids[0]: "API error response: EOrder:Unknown order", ids[1]: "API error response: EOrder:Unknown order"}
		assert.Equal(t, exp, status, "cancelSpotOrders should record Kraken's error for each order no request cancelled")
	})

	t.Run("Cancel Order Batch failing", func(t *testing.T) {
		t.Parallel()
		ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/private/CancelOrderBatch", r.URL.Path, "cancelSpotOrders should call Cancel Order Batch")
			_, _ = w.Write([]byte(`{"error":["EGeneral:Invalid arguments"]}`))
		})
		status := make(map[string]string)
		err := ex.cancelSpotOrders(t.Context(), ids[:2], status)
		require.ErrorIs(t, err, errAPIResponse, "cancelSpotOrders must return Cancel Order Batch's error")
		assert.Empty(t, status, "cancelSpotOrders should record no outcome when the batch request fails")
	})

	if !mockTests {
		return
	}
	t.Run("order Kraken does not know cancelled on its own over REST", func(t *testing.T) {
		t.Parallel()
		status := make(map[string]string)
		require.NoError(t, e.cancelSpotOrders(t.Context(), []string{"OZJW0T-MIQ3C-5NATAU"}, status), "cancelSpotOrders must not error")
		assert.Equal(t, map[string]string{"OZJW0T-MIQ3C-5NATAU": "API error response: EOrder:Unknown order"}, status, "cancelSpotOrders should record Kraken's error")
	})
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancellation")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Exchange: e.Name, AssetType: asset.Margin})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an unsupported asset")
	failing := newOrdersFailingExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err = failing.CancelAllOrders(t.Context(), &order.Cancel{Exchange: failing.Name, AssetType: a})
		assert.ErrorIsf(t, err, errAPIResponse, "CancelAllOrders should return the %s request's error", a)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
		exp    map[string]string
	}{
		{
			name:   "every spot order",
			cancel: &order.Cancel{AssetType: asset.Spot},
			exp:    map[string]string{"OQCLML-BW3P3-BUCMWZ": order.Cancelled.String(), "OHYO67-6LP66-HMQ437": order.Cancelled.String()},
		},
		{
			name:   "a spot pair's orders",
			cancel: &order.Cancel{AssetType: asset.Spot, Pair: spotTestPair},
			exp:    map[string]string{"OQCLML-BW3P3-BUCMWZ": order.Cancelled.String()},
		},
		{
			name:   "every futures order",
			cancel: &order.Cancel{AssetType: asset.Futures},
			exp: map[string]string{
				"d6e7f8a9-b0c1-4d2e-9f3a-4b5c6d7e8f90": order.Cancelled.String(),
				"a3b4c5d6-e7f8-4a1b-8c2d-3e4f5a6b7c8d": order.Cancelled.String(),
			},
		},
		{
			name:   "a futures pair without open orders",
			cancel: &order.Cancel{AssetType: asset.Futures, Pair: futuresTestPair},
			exp:    map[string]string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.cancel.Exchange = e.Name
			resp, err := e.CancelAllOrders(t.Context(), tc.cancel)
			require.NoError(t, err, "CancelAllOrders must not error")
			if !mockTests {
				assert.NotNil(t, resp.Status, "CancelAllOrders should report the cancelled orders")
				return
			}
			assert.Equal(t, order.CancelAllResponse{Status: tc.exp}, resp, "CancelAllOrders should report each cancelled order")
		})
	}
	if !mockTests {
		return
	}
	t.Run("a spot pair's orders over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{
			request: ordersWsRequest("cancel_order", `"order_id":["OQCLML-BW3P3-BUCMWZ"]`),
			replies: []string{ordersWsReply("cancel_order", `{"order_id":"OQCLML-BW3P3-BUCMWZ"}`)},
		})
		resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{Exchange: ex.Name, AssetType: asset.Spot, Pair: spotTestPair})
		require.NoError(t, err, "CancelAllOrders must not error")
		assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"OQCLML-BW3P3-BUCMWZ": order.Cancelled.String()}}, resp, "CancelAllOrders should cancel the pair's open orders over the private websocket")
	})
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", spotTestPair, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must reject an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "OQCLML-BW3P3-BUCMWZ", spotTestPair, asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an unsupported asset")
	failing := newOrdersFailingExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err = failing.GetOrderInfo(t.Context(), "OQCLML-BW3P3-BUCMWZ", currency.EMPTYPAIR, a)
		assert.ErrorIsf(t, err, errAPIResponse, "GetOrderInfo should return the %s request's error", a)
	}
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/0/private/QueryTrades" {
			_, _ = w.Write([]byte(`{"error":["EAPI:Rate limit exceeded"]}`))
			return
		}
		_, _ = w.Write([]byte(`{"error":[],"result":{"OAOAF2-9FPNH-WJR3S9":{"cl_ord_id":null,"closetm":1791480299.1204,"cost":"612.50000","descr":{"aclass":"forex","close":"","leverage":"none","order":"sell 500.00000000 WAVESUSD @ limit 1.2250","ordertype":"limit","pair":"WAVESUSD","price":"1.2250","price2":"0","type":"sell"},"expiretm":0,"fee":"1.59250","limitprice":"0.00000","misc":"","oflags":"fcib","opentm":1791480200.3381,"price":"1.2250","reason":null,"refid":null,"sender_sub_id":null,"starttm":0,"status":"closed","stopprice":"0.00000","time_in_force":"gtc","trades":["TXWQSD-V0G15-21EON1"],"userref":null,"vol":"500.00000000","vol_exec":"500.00000000"},` +
			`"O6KIO9-Y73RV-6LFAIX":{"cl_ord_id":"gct-history-1","closetm":1791421301.0127,"cost":"3250.00000","descr":{"aclass":"forex","close":"","leverage":"none","order":"buy 0.04000000 XBTUSD @ limit 81250.0","ordertype":"limit","pair":"XBTUSD","price":"81250.0","price2":"0","type":"buy"},"expiretm":0,"fee":"8.45000","limitprice":"0.00000","misc":"","oflags":"fciq","opentm":1791421234.5512,"price":"81250.0","reason":null,"refid":null,"sender_sub_id":null,"starttm":0,"status":"closed","stopprice":"0.00000","time_in_force":"gtc","trades":["TVDNBH-CIQGX-SO4L12"],"userref":null,"vol":"0.04000000","vol_exec":"0.04000000"}}}`))
	})
	_, err = ex.GetOrderInfo(t.Context(), "OAOAF2-9FPNH-WJR3S9", currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "GetOrderInfo should reject a spot order on a pair no longer listed")
	_, err = ex.GetOrderInfo(t.Context(), "O6KIO9-Y73RV-6LFAIX", currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, errAPIResponse, "GetOrderInfo should return the error of the spot order's trades request")

	ex = newOrdersFillsExchange(t, "", ordersFuturesFills, newHistoryListing([]string{}))
	got, err := ex.GetOrderInfo(t.Context(), ordersFillsOrderID, currency.EMPTYPAIR, asset.Futures)
	require.NoError(t, err, "GetOrderInfo must not error for a futures order known by its fills alone")
	exp := &order.Detail{
		Exchange:      ex.Name,
		AssetType:     asset.Futures,
		OrderID:       ordersFillsOrderID,
		ClientOrderID: "gct-fills-1",
		Pair:          futuresTestPair,
		Side:          order.Buy,
		LastUpdated:   time.Date(2026, 10, 9, 8, 41, 7, 335000000, time.UTC),
		Trades:        ordersFuturesFillTrades(ex.Name),
	}
	assert.Equal(t, exp, got, "GetOrderInfo should return a futures order no longer open or cached, whose order events do not show how it ended, as its trades alone")
	// The order was of 2, partly filled; its fills alone must not finish it or shrink it to what they hold
	placed := time.Date(2026, 10, 9, 8, 40, 0, 50000000, time.UTC)
	stored := order.Detail{
		Exchange:        ex.Name,
		AssetType:       asset.Futures,
		OrderID:         ordersFillsOrderID,
		ClientOrderID:   "gct-fills-1",
		Pair:            futuresTestPair,
		Side:            order.Buy,
		Type:            order.Limit,
		Status:          order.PartiallyFilled,
		Price:           81450,
		Amount:          2,
		ExecutedAmount:  0.5,
		RemainingAmount: 1.5,
		Date:            placed,
		LastUpdated:     time.Date(2026, 10, 9, 8, 40, 12, 118000000, time.UTC),
	}
	expStored := stored
	expStored.LastUpdated, expStored.Trades = exp.LastUpdated, exp.Trades
	require.NoError(t, stored.UpdateOrderFromDetail(got), "UpdateOrderFromDetail must not error")
	assert.Equal(t, expStored, stored, "an order store merging the order should keep its size, status and amounts, adding its trades")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		for _, a := range e.GetAssetTypes(false) {
			orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
			if len(orders) == 0 {
				continue
			}
			d, err := e.GetOrderInfo(t.Context(), orders[0].OrderID, orders[0].Pair, a)
			require.NoErrorf(t, err, "GetOrderInfo must not error for %s", a)
			assert.Equalf(t, orders[0].OrderID, d.OrderID, "GetOrderInfo should return the %s order requested", a)
		}
		return
	}
	spotOrder := ordersStopLossLimitDetail(e.Name)
	spotOrder.Trades = []order.TradeHistory{
		{
			TID:       "TCCCTY-WE2O6-P3NB37",
			Price:     30012.5,
			Amount:    0.25,
			Fee:       19.50813,
			FeeAsset:  "USD",
			Exchange:  e.Name,
			Type:      order.StopLimit,
			Side:      order.Buy,
			Timestamp: time.UnixMicro(1780582261442100),
			Total:     7503.125,
		},
		{
			TID:       "TJUW2K-FLX2N-AR2FLU",
			Price:     30012.5,
			Amount:    0.125,
			Fee:       9.75406,
			FeeAsset:  "USD",
			Exchange:  e.Name,
			Type:      order.StopLimit,
			Side:      order.Buy,
			Timestamp: time.UnixMicro(1780582262015300),
			IsMaker:   true,
			Total:     3751.5625,
		},
	}
	futuresOrder := ordersFuturesLimitDetail(e.Name)
	for _, tc := range []struct {
		name    string
		orderID string
		a       asset.Item
		exp     *order.Detail
	}{
		{"spot order with its trades", "OQCLML-BW3P3-BUCMWZ", asset.Spot, &spotOrder},
		{"open futures order", "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b", asset.Futures, &futuresOrder},
		{
			name:    "futures order from its status",
			orderID: "1a2b3c4d-5e6f-4a7b-9c8d-0e1f2a3b4c5d",
			a:       asset.Futures,
			exp: &order.Detail{
				Exchange:        e.Name,
				AssetType:       asset.Futures,
				OrderID:         "1a2b3c4d-5e6f-4a7b-9c8d-0e1f2a3b4c5d",
				ClientOrderID:   "gct-trailing-3",
				Pair:            futuresTestPair,
				Side:            order.Sell,
				Status:          order.Rejected,
				ReduceOnly:      true,
				Price:           80400,
				TriggerPrice:    80650,
				Amount:          0.0005,
				ExecutedAmount:  0.0001,
				RemainingAmount: 0.0004,
				Date:            time.Date(2026, 10, 9, 8, 12, 31, 448000000, time.UTC),
				LastUpdated:     time.Date(2026, 10, 9, 8, 27, 41, 302000000, time.UTC),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderInfo(t.Context(), tc.orderID, currency.EMPTYPAIR, tc.a)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, tc.exp, got, "GetOrderInfo should return the order")
		})
	}
	for _, tc := range []struct {
		orderID string
		a       asset.Item
	}{
		{"O7VICR-5EPZ6-VHTP7B", asset.Spot},
		{"89408aa3-845e-4a0f-bd6a-b44dcf4361a5", asset.Futures},
	} {
		_, err = e.GetOrderInfo(t.Context(), tc.orderID, currency.EMPTYPAIR, tc.a)
		assert.ErrorIsf(t, err, order.ErrOrderNotFound, "GetOrderInfo should report a %s order Kraken does not return as not found", tc.a)
	}
}

func TestSpotOrderDetail(t *testing.T) {
	t.Parallel()
	openTime, closeTime := time.UnixMicro(1791421234551200), time.UnixMicro(1791421301012700)
	for _, tc := range []struct {
		name string
		info *OrderInfo
		exp  *order.Detail
	}{
		{
			name: "partly filled open post only order",
			info: &OrderInfo{
				ClientOrderID:  "gct-detail-1",
				Status:         "open",
				OpenTime:       types.Time(openTime),
				Description:    OrderDescription{Pair: "XBTUSD", Side: "buy", OrderType: "limit", Price: OrderPrice{Value: 81250}},
				TimeInForce:    "gtc",
				Volume:         0.04,
				VolumeExecuted: 0.01,
				Cost:           812.5,
				Fee:            1.3,
				AveragePrice:   81250,
				OrderFlags:     "post,fciq",
			},
			exp: &order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Spot,
				OrderID:              "O6KIO9-Y73RV-6LFAIX",
				ClientOrderID:        "gct-detail-1",
				Pair:                 wsTestPair,
				Side:                 order.Buy,
				Type:                 order.Limit,
				Status:               order.PartiallyFilled,
				TimeInForce:          order.GoodTillCancel | order.PostOnly,
				Price:                81250,
				AverageExecutedPrice: 81250,
				Amount:               0.04,
				ExecutedAmount:       0.01,
				RemainingAmount:      0.03,
				Cost:                 812.5,
				CostAsset:            currency.USD,
				Fee:                  1.3,
				FeeAsset:             currency.USD,
				Date:                 openTime,
				LastUpdated:          openTime,
			},
		},
		{
			name: "partly filled cancelled stop loss order without a time in force",
			info: &OrderInfo{
				Status:         "canceled",
				OpenTime:       types.Time(openTime),
				CloseTime:      types.Time(closeTime),
				Description:    OrderDescription{Pair: "XBTUSD", Side: "sell", OrderType: "stop-loss", Price: OrderPrice{Value: 75000}},
				Volume:         0.5,
				VolumeExecuted: 0.25,
				Cost:           18750,
				Fee:            48.75,
				AveragePrice:   75000,
				ReduceOnly:     true,
			},
			exp: &order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Spot,
				OrderID:              "O6KIO9-Y73RV-6LFAIX",
				Pair:                 wsTestPair,
				Side:                 order.Sell,
				Type:                 order.Stop,
				Status:               order.PartiallyFilledCancelled,
				ReduceOnly:           true,
				TriggerPrice:         75000,
				AverageExecutedPrice: 75000,
				Amount:               0.5,
				ExecutedAmount:       0.25,
				RemainingAmount:      0.25,
				Cost:                 18750,
				CostAsset:            currency.USD,
				Fee:                  48.75,
				FeeAsset:             currency.USD,
				Date:                 openTime,
				CloseTime:            closeTime,
				LastUpdated:          closeTime,
			},
		},
		{
			name: "closed take profit limit order",
			info: &OrderInfo{
				Status:         "closed",
				OpenTime:       types.Time(openTime),
				CloseTime:      types.Time(closeTime),
				Description:    OrderDescription{Pair: "XBTGBP", Side: "buy", OrderType: "take-profit-limit", Price: OrderPrice{Value: 60000}, SecondaryPrice: OrderPrice{Value: 60100}},
				TimeInForce:    "ioc",
				Volume:         0.5,
				VolumeExecuted: 0.5,
				Cost:           30050,
				Fee:            78.13,
				AveragePrice:   60100,
				OrderFlags:     "fcib",
			},
			exp: &order.Detail{
				Exchange:             e.Name,
				AssetType:            asset.Spot,
				OrderID:              "O6KIO9-Y73RV-6LFAIX",
				Pair:                 ordersXBTGBPPair,
				Side:                 order.Buy,
				Type:                 order.TakeProfit | order.Limit,
				Status:               order.Filled,
				TimeInForce:          order.ImmediateOrCancel,
				Price:                60100,
				TriggerPrice:         60000,
				AverageExecutedPrice: 60100,
				Amount:               0.5,
				ExecutedAmount:       0.5,
				Cost:                 30050,
				CostAsset:            currency.GBP,
				Fee:                  78.13,
				FeeAsset:             currency.GBP,
				Date:                 openTime,
				CloseTime:            closeTime,
				LastUpdated:          closeTime,
			},
		},
	} {
		got, err := e.spotOrderDetail("O6KIO9-Y73RV-6LFAIX", tc.info)
		require.NoErrorf(t, err, "spotOrderDetail must not error for a %s", tc.name)
		assert.Equalf(t, tc.exp, got, "spotOrderDetail should convert a %s", tc.name)
	}

	// A trailing stop's description prices are offsets from the market, as an open trailing stop Kraken returned shows,
	// so its trigger and limit prices are those it holds now
	xrpEUR := currency.NewPairWithDelimiter("XRP", "EUR", "_")
	trailingOpenTime := time.Unix(1706893367, 465664900)
	for _, tc := range []struct {
		name, orderType, price, secondaryPrice string
		exp                                    *order.Detail
	}{
		{
			name:      "trailing stop",
			orderType: "trailing-stop",
			price:     "+50.0000%",
			exp:       &order.Detail{Type: order.TrailingStop, TriggerPrice: 0.23424},
		},
		{
			name:           "trailing stop limit",
			orderType:      "trailing-stop-limit",
			price:          "+50.0000%",
			secondaryPrice: "-0.0100",
			exp:            &order.Detail{Type: order.TrailingStopLimit, TriggerPrice: 0.23424, Price: 0.46847},
		},
	} {
		var info OrderInfo
		require.NoErrorf(t, json.Unmarshal([]byte(`{"refid":null,"userref":0,"status":"open","opentm":1706893367.4656649,"starttm":0,"expiretm":0,`+
			`"descr":{"pair":"XRPEUR","type":"sell","ordertype":"`+tc.orderType+`","price":"`+tc.price+`","price2":"`+cmp.Or(tc.secondaryPrice, "0")+`","leverage":"none",`+
			`"order":"sell 10.00000000 XRPEUR @ trailing stop `+tc.price+`","close":""},"vol":"10.00000000","vol_exec":"0.00000000","cost":"0.00000000",`+
			`"fee":"0.00000000","price":"0.00000000","stopprice":"0.23424000","limitprice":"0.46847000","misc":"","oflags":"fciq","trigger":"index"}`), &info),
			"Unmarshal must not error for a %s", tc.name)
		got, err := e.spotOrderDetail("OKKB2W-DFVLB-P3NNO6", &info)
		require.NoErrorf(t, err, "spotOrderDetail must not error for a %s", tc.name)
		exp := &order.Detail{
			Exchange:        e.Name,
			AssetType:       asset.Spot,
			OrderID:         "OKKB2W-DFVLB-P3NNO6",
			Pair:            xrpEUR,
			Side:            order.Sell,
			Type:            tc.exp.Type,
			Status:          order.Open,
			Price:           tc.exp.Price,
			TriggerPrice:    tc.exp.TriggerPrice,
			Amount:          10,
			RemainingAmount: 10,
			CostAsset:       currency.EUR,
			FeeAsset:        currency.EUR,
			Date:            trailingOpenTime,
			LastUpdated:     trailingOpenTime,
		}
		assert.Equalf(t, exp, got, "spotOrderDetail should convert a %s", tc.name)
	}

	valid := OrderInfo{Status: "open", Description: OrderDescription{Pair: "XBTUSD", Side: "buy", OrderType: "limit", Price: OrderPrice{Value: 81250}}, TimeInForce: "gtc"}
	for _, tc := range []struct {
		name   string
		modify func(*OrderInfo)
		err    error
	}{
		{"an order on a pair no longer listed", func(o *OrderInfo) { o.Description.Pair = "WAVESUSD" }, currency.ErrPairNotFound},
		{"an order without a side", func(o *OrderInfo) { o.Description.Side = "" }, order.ErrSideIsInvalid},
		{"an unknown order type", func(o *OrderInfo) { o.Description.OrderType = "moon" }, errUnknownOrderType},
		{"an unknown status", func(o *OrderInfo) { o.Status = "moon" }, errUnknownOrderStatus},
		{"an unknown time in force", func(o *OrderInfo) { o.TimeInForce = "gtx" }, order.ErrInvalidTimeInForce},
	} {
		info := valid
		tc.modify(&info)
		_, err := e.spotOrderDetail("O6KIO9-Y73RV-6LFAIX", &info)
		assert.ErrorIsf(t, err, tc.err, "spotOrderDetail should reject %s", tc.name)
	}
}

func TestSpotOrderTrades(t *testing.T) {
	t.Parallel()
	ids := make([]string, 21)
	for i := range ids {
		ids[i] = "TNPSC" + string(rune('A'+i)) + "-HV7VA-9AD78E"
	}
	// The reply leaves this trade out, which spotOrderTrades skips
	missing := ids[6]
	var (
		mu        sync.Mutex
		requested []string
	)
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/0/private/QueryTrades", r.URL.Path, "spotOrderTrades should call Query Trades Info")
		var req struct {
			TradeIDs string `json:"txid"`
		}
		assert.NoError(t, json.Unmarshal([]byte(tradingRequestBody(t, r)), &req), "Unmarshal should not error")
		mu.Lock()
		requested = append(requested, req.TradeIDs)
		mu.Unlock()
		trades := make(map[string]any)
		for id := range strings.SplitSeq(req.TradeIDs, ",") {
			if id == missing {
				continue
			}
			i := slices.Index(ids, id)
			trades[id] = map[string]any{
				"aclass": "forex", "cost": "810.00000", "ext_exec_id": "EXEC-" + strconv.Itoa(56000+i), "fee": "2.10600", "maker": i%2 == 0,
				"misc": "", "ordertxid": "O6KIO9-Y73RV-6LFAIX", "ordertype": "limit", "pair": "XXBTZUSD", "price": "81000.00000",
				"time": float64(1791530000+i) + 0.5, "trade_id": 40300000 + i, "tradeordertype": "limit", "type": "buy", "vol": "0.01000000",
			}
		}
		resp, err := json.Marshal(map[string]any{"error": []string{}, "result": trades})
		assert.NoError(t, err, "Marshal should not error")
		_, _ = w.Write(resp)
	})
	d := &order.Detail{Type: order.Limit, Side: order.Buy, FeeAsset: currency.USD}
	got, err := ex.spotOrderTrades(t.Context(), &OrderInfo{TradeIDs: ids}, d)
	require.NoError(t, err, "spotOrderTrades must not error")
	exp := make([]order.TradeHistory, 0, len(ids)-1)
	for i, id := range ids {
		if id == missing {
			continue
		}
		exp = append(exp, order.TradeHistory{
			TID:       id,
			Price:     81000,
			Amount:    0.01,
			Fee:       2.106,
			FeeAsset:  "USD",
			Exchange:  ex.Name,
			Type:      order.Limit,
			Side:      order.Buy,
			Timestamp: time.UnixMilli(int64(1791530000+i)*1000 + 500),
			IsMaker:   i%2 == 0,
			Total:     810,
		})
	}
	assert.Equal(t, exp, got, "spotOrderTrades should return each trade Kraken returned, in the order's order")
	mu.Lock()
	assert.Equal(t, []string{strings.Join(ids[:20], ","), ids[20]}, requested, "spotOrderTrades should query the trades 20 at a time")
	mu.Unlock()

	ex = newHTTPTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":["EGeneral:Invalid arguments"]}`))
	})
	_, err = ex.spotOrderTrades(t.Context(), &OrderInfo{TradeIDs: ids[:1]}, d)
	assert.ErrorIs(t, err, errAPIResponse, "spotOrderTrades should return Query Trades Info's error")
}

// ordersFillsOrderID is the futures order whose fills ordersFuturesFills holds, which is neither open nor still
// cached by Get Specific Orders' Status
const ordersFillsOrderID = "bd229873-78e4-45f7-ad1e-5b84a8a2c87a"

// ordersFuturesFills is Get fills' reply holding two fills of ordersFillsOrderID, newest first, around another order's
const ordersFuturesFills = `{"fills":[` +
	`{"cliOrdId":"gct-fills-1","fillTime":"2026-10-09T08:41:07.335Z","fillType":"taker","fill_id":"5c0e9a1f-2b7d-4e36-8f41-9d2a6c3b8e70","order_id":"` + ordersFillsOrderID +
	`","price":81447,"realized_pnl":0,"sequence_id":"57","side":"buy","size":0.25,"symbol":"PF_XBTUSD"},` +
	`{"cliOrdId":null,"fillTime":"2026-10-09T08:40:51.902Z","fillType":"maker","fill_id":"8a4f2d6b-1c9e-4b7a-a3f5-0e6d8c2b4a91","order_id":"d6e7f8a9-b0c1-4d2e-9f3a-4b5c6d7e8f90",` +
	`"price":81460,"realized_pnl":0.0012,"sequence_id":"55","side":"sell","size":0.0004,"symbol":"PF_XBTUSD"},` +
	`{"cliOrdId":"gct-fills-1","fillTime":"2026-10-09T08:40:12.118Z","fillType":"maker","fill_id":"3e7b1c5d-9a2f-4d68-b0e4-7f1a3c9d5b26","order_id":"` + ordersFillsOrderID +
	`","price":81450,"realized_pnl":0,"sequence_id":"51","side":"buy","size":0.5,"symbol":"PF_XBTUSD"}` +
	`],"result":"success","serverTime":"2026-10-09T08:42:00.004Z"}`

// ordersFuturesFillTrades returns the trades of ordersFillsOrderID that ordersFuturesFills holds, newest first
func ordersFuturesFillTrades(exchangeName string) []order.TradeHistory {
	return []order.TradeHistory{
		{TID: "5c0e9a1f-2b7d-4e36-8f41-9d2a6c3b8e70", Price: 81447, Amount: 0.25, Exchange: exchangeName, Side: order.Buy, Timestamp: time.Date(2026, 10, 9, 8, 41, 7, 335000000, time.UTC)},
		{TID: "3e7b1c5d-9a2f-4d68-b0e4-7f1a3c9d5b26", Price: 81450, Amount: 0.5, Exchange: exchangeName, Side: order.Buy, Timestamp: time.Date(2026, 10, 9, 8, 40, 12, 118000000, time.UTC), IsMaker: true},
	}
}

// historyListing is a Derivatives history listing serving its pages by the continuation token requesting them, "" for
// the first, which records each request's query. A token requested twice, or one it holds no page for, is refused with
// a 400, so a pager that does not advance fails instead of looping
type historyListing struct {
	pages     map[string]string
	mu        sync.Mutex
	requested []url.Values
}

// newHistoryListing returns a listing whose pages hold these events, each page but the last continued to the next
func newHistoryListing(pages ...[]string) *historyListing {
	l := &historyListing{pages: make(map[string]string, len(pages))}
	var token string
	for i, elements := range pages {
		var next string
		if i < len(pages)-1 {
			next = "page" + strconv.Itoa(i+2)
		}
		l.pages[token] = futuresHistoryPage(next, elements...)
		token = next
	}
	return l
}

// serve answers a request with the page of its continuation token
func (l *historyListing) serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	token := q.Get("continuation_token")
	l.mu.Lock()
	repeated := slices.ContainsFunc(l.requested, func(v url.Values) bool { return v.Get("continuation_token") == token })
	l.requested = append(l.requested, q)
	l.mu.Unlock()
	page, ok := l.pages[token]
	if repeated || !ok {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"bad_request","reason":"The continuation token was requested before or never given."}`))
		return
	}
	_, _ = w.Write([]byte(page))
}

// queries returns the encoded queries of the requests made so far
func (l *historyListing) queries() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	queries := make([]string, len(l.requested))
	for i := range l.requested {
		queries[i] = l.requested[i].Encode()
	}
	return queries
}

// futuresHistoryPage returns a page of a Derivatives account history listing holding elements, continued by next when
// it is set
func futuresHistoryPage(next string, elements ...string) string {
	page := `{"accountUid":"` + futuresHistoryTestAccount + `","len":` + strconv.Itoa(len(elements)) + `,"serverTime":"2026-10-09T08:42:00.010Z","elements":[` + strings.Join(elements, ",") + `]`
	if next != "" {
		page += `,"continuationToken":"` + next + `"`
	}
	return page + `}`
}

// futuresHistoryElement returns an account history event of kind holding body, which happened at
func futuresHistoryElement(uid string, at time.Time, kind, body string) string {
	return `{"uid":"` + uid + `","timestamp":` + strconv.FormatInt(at.UnixMilli(), 10) + `,"event":{"` + kind + `":` + body + `}}`
}

// futuresHistoryOrderJSON returns a PF_XBTUSD order as order events report it
func futuresHistoryOrderJSON(id, clientID, direction, orderType, quantity, filled string, placed, updated time.Time) string {
	return `{"uid":"` + id + `","accountUid":"` + futuresHistoryTestAccount + `","tradeable":"PF_XBTUSD","direction":"` + direction + `","quantity":"` + quantity +
		`","filled":"` + filled + `","timestamp":` + strconv.FormatInt(placed.UnixMilli(), 10) + `,"limitPrice":"81450","orderType":"` + orderType +
		`","clientId":"` + clientID + `","reduceOnly":false,"lastUpdateTimestamp":` + strconv.FormatInt(updated.UnixMilli(), 10) + `}`
}

// newOrdersFillsExchange returns an exchange named after the test whose futures endpoints hold no open orders, no
// cached status of ordersFillsOrderID, fills and the order events orderEvents serves, and refuse the request to
// failingPath
func newOrdersFillsExchange(t *testing.T, failingPath, fills string, orderEvents *historyListing) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == failingPath {
			_, _ = w.Write([]byte(`{"result":"error","error":"apiLimitExceeded","serverTime":"2026-10-09T08:42:00.001Z"}`))
			return
		}
		switch r.URL.Path {
		case "/derivatives/api/v3/openorders":
			_, _ = w.Write([]byte(`{"openOrders":[],"result":"success","serverTime":"2026-10-09T08:42:00.001Z"}`))
		case "/derivatives/api/v3/orders/status":
			assert.Equal(t, "orderIds="+ordersFillsOrderID, r.URL.RawQuery, "the order's status should be asked for")
			_, _ = w.Write([]byte(`{"orders":[],"result":"success","serverTime":"2026-10-09T08:42:00.002Z"}`))
		case "/derivatives/api/v3/fills":
			_, _ = w.Write([]byte(fills))
		case "/api/history/v3/orders":
			orderEvents.serve(w, r)
		default:
			assert.Failf(t, "only the open orders, order status, fills and order events endpoints should be called", "%s", r.URL.Path)
		}
	})
}

func TestFuturesOrderInfo(t *testing.T) {
	t.Parallel()
	ex := newOrdersFillsExchange(t, "", strings.ReplaceAll(ordersFuturesFills, `"PF_XBTUSD"`, `"PF_ZORAUSD"`), newHistoryListing([]string{}))
	_, err := ex.futuresOrderInfo(t.Context(), ordersFillsOrderID)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "futuresOrderInfo should reject fills on a contract no longer listed")
	for _, path := range []string{"/derivatives/api/v3/openorders", "/derivatives/api/v3/orders/status", "/derivatives/api/v3/fills"} {
		ex = newOrdersFillsExchange(t, path, ordersFuturesFills, newHistoryListing([]string{}))
		_, err = ex.futuresOrderInfo(t.Context(), ordersFillsOrderID)
		assert.ErrorIsf(t, err, errAPIResponse, "futuresOrderInfo should return the error of %s", path)
	}
	// The order events only add how the order ended, so its fills are returned when they cannot be read
	ex = newOrdersFillsExchange(t, "/api/history/v3/orders", ordersFuturesFills, newHistoryListing([]string{}))
	fillsOnly, err := ex.futuresOrderInfo(t.Context(), ordersFillsOrderID)
	require.NoError(t, err, "futuresOrderInfo must not error when the order events cannot be read")
	assert.Equal(t, order.UnknownStatus, fillsOnly.Status, "futuresOrderInfo should return no status when the order events cannot be read")
	assert.Equal(t, ordersFuturesFillTrades(ex.Name), fillsOnly.Trades, "futuresOrderInfo should return the order's fills when the order events cannot be read")

	// Order events are timed in milliseconds, which decode to local times, at the times of the order's fills
	placed, firstFill, lastFill, cancelled := time.UnixMilli(1791535200050), time.UnixMilli(1791535212118), time.UnixMilli(1791535267335), time.UnixMilli(1791535290000)
	afterFirstFill := futuresHistoryOrderJSON(ordersFillsOrderID, "gct-fills-1", "Buy", "Limit", "2", "0.5", placed, firstFill)
	afterLastFill := futuresHistoryOrderJSON(ordersFillsOrderID, "gct-fills-1", "Buy", "Limit", "2", "0.75", placed, lastFill)
	lastFillEvent := futuresHistoryElement("e1a2b3c4-0001-4000-8000-000000000001", lastFill, "OrderUpdated",
		`{"oldOrder":`+afterFirstFill+`,"newOrder":`+afterLastFill+`,"reason":"partial_fill","reducedQuantity":"0"}`)
	fullFillEvent := futuresHistoryElement("e1a2b3c4-0001-4000-8000-000000000002", lastFill, "OrderUpdated",
		`{"oldOrder":`+strings.Replace(afterFirstFill, `"quantity":"2"`, `"quantity":"0.75"`, 1)+
			`,"newOrder":`+strings.Replace(afterLastFill, `"quantity":"2"`, `"quantity":"0.75"`, 1)+`,"reason":"full_fill","reducedQuantity":"0"}`)
	cancelEvent := futuresHistoryElement("e1a2b3c4-0001-4000-8000-000000000003", cancelled, "OrderCancelled", `{"order":`+afterLastFill+`,"reason":"cancelled_by_user"}`)
	// Events of other orders on the market, including one closing and one Kraken sent with a type it could not name
	otherOrder := futuresHistoryOrderJSON("d6e7f8a9-b0c1-4d2e-9f3a-4b5c6d7e8f90", "", "Sell", "Post", "0.5", "0.0004", placed, lastFill)
	otherCancelEvent := futuresHistoryElement("e1a2b3c4-0001-4000-8000-000000000004", lastFill.Add(time.Second), "OrderCancelled", `{"order":`+otherOrder+`,"reason":"cancelled_by_user"}`)
	undecodedEvent := futuresHistoryElement("e1a2b3c4-0001-4000-8000-000000000005", lastFill.Add(2*time.Second), "OrderPlaced",
		`{"order":`+futuresHistoryOrderJSON("1f2e3d4c-5b6a-4978-8695-a4b3c2d1e0f9", "", "Buy", "Moon", "1", "0", lastFill, lastFill)+`,"reason":"new_user_order","reducedQuantity":""}`)
	// The order's final event cannot come before its last fill, a second before which the search starts
	query := "count=1000&since=1791535266335&sort=asc&tradeable=PF_XBTUSD"
	for _, tc := range []struct {
		name    string
		pages   [][]string
		exp     func(exchangeName string) *order.Detail
		queries []string
	}{
		{
			name:  "cancelled after its last fill",
			pages: [][]string{{lastFillEvent, otherCancelEvent, cancelEvent}},
			exp: func(exchangeName string) *order.Detail {
				return &order.Detail{
					Exchange:        exchangeName,
					AssetType:       asset.Futures,
					OrderID:         ordersFillsOrderID,
					ClientOrderID:   "gct-fills-1",
					Pair:            futuresTestPair,
					Side:            order.Buy,
					Type:            order.Limit,
					Status:          order.PartiallyFilledCancelled,
					Price:           81450,
					Amount:          2,
					ExecutedAmount:  0.75,
					RemainingAmount: 1.25,
					Date:            placed,
					LastUpdated:     cancelled,
					Trades:          ordersFuturesFillTrades(exchangeName),
				}
			},
			queries: []string{query},
		},
		{
			name:  "filled by its last fill",
			pages: [][]string{{fullFillEvent}},
			exp: func(exchangeName string) *order.Detail {
				return &order.Detail{
					Exchange:       exchangeName,
					AssetType:      asset.Futures,
					OrderID:        ordersFillsOrderID,
					ClientOrderID:  "gct-fills-1",
					Pair:           futuresTestPair,
					Side:           order.Buy,
					Type:           order.Limit,
					Status:         order.Filled,
					Price:          81450,
					Amount:         0.75,
					ExecutedAmount: 0.75,
					Date:           placed,
					LastUpdated:    lastFill,
					Trades:         ordersFuturesFillTrades(exchangeName),
				}
			},
			queries: []string{query},
		},
		{
			name:    "order events showing no end",
			pages:   [][]string{{lastFillEvent, otherCancelEvent}},
			queries: []string{query},
		},
		{
			name:  "ending on the last page searched",
			pages: [][]string{{lastFillEvent}, {otherCancelEvent}, {undecodedEvent, cancelEvent}},
			exp: func(exchangeName string) *order.Detail {
				return &order.Detail{
					Exchange:        exchangeName,
					AssetType:       asset.Futures,
					OrderID:         ordersFillsOrderID,
					ClientOrderID:   "gct-fills-1",
					Pair:            futuresTestPair,
					Side:            order.Buy,
					Type:            order.Limit,
					Status:          order.PartiallyFilledCancelled,
					Price:           81450,
					Amount:          2,
					ExecutedAmount:  0.75,
					RemainingAmount: 1.25,
					Date:            placed,
					LastUpdated:     cancelled,
					Trades:          ordersFuturesFillTrades(exchangeName),
				}
			},
			queries: []string{query, "continuation_token=page2&" + query, "continuation_token=page3&" + query},
		},
		{
			name:    "ending past the pages searched",
			pages:   [][]string{{lastFillEvent}, {otherCancelEvent}, {undecodedEvent}, {cancelEvent}},
			queries: []string{query, "continuation_token=page2&" + query, "continuation_token=page3&" + query},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			events := newHistoryListing(tc.pages...)
			ex := newOrdersFillsExchange(t, "", ordersFuturesFills, events)
			got, err := ex.futuresOrderInfo(t.Context(), ordersFillsOrderID)
			require.NoError(t, err, "futuresOrderInfo must not error")
			assert.Equal(t, tc.queries, events.queries(), "futuresOrderInfo should search the order's market from its last fill, a bounded number of pages")
			if tc.exp != nil {
				assert.Equal(t, tc.exp(ex.Name), got, "futuresOrderInfo should return the order as its final event left it, with its trades")
				return
			}
			assert.Equal(t, &order.Detail{
				Exchange:      ex.Name,
				AssetType:     asset.Futures,
				OrderID:       ordersFillsOrderID,
				ClientOrderID: "gct-fills-1",
				Pair:          futuresTestPair,
				Side:          order.Buy,
				LastUpdated:   time.Date(2026, 10, 9, 8, 41, 7, 335000000, time.UTC),
				Trades:        ordersFuturesFillTrades(ex.Name),
			}, got, "futuresOrderInfo should return an order whose order events do not show how it ended as its trades alone")
		})
	}
}

func TestFuturesOpenOrderDetail(t *testing.T) {
	t.Parallel()
	received, updated := time.Date(2026, 10, 9, 8, 24, 59, 122000000, time.UTC), time.Date(2026, 10, 9, 8, 26, 15, 4000000, time.UTC)
	for _, tc := range []struct {
		name  string
		order *FuturesOpenOrder
		exp   *order.Detail
	}{
		{
			name: "untouched take profit order",
			order: &FuturesOpenOrder{
				OrderID:        "cd1adc0e-2cbe-4c9c-9b0c-7526b5c7c567",
				Symbol:         "pf_xbtusd",
				Side:           "sell",
				OrderType:      "take_profit",
				Status:         "untouched",
				StopPrice:      86000,
				UnfilledSize:   0.001,
				TriggerSignal:  "last",
				ReceivedTime:   received,
				LastUpdateTime: received,
			},
			exp: &order.Detail{
				Exchange:        e.Name,
				AssetType:       asset.Futures,
				OrderID:         "cd1adc0e-2cbe-4c9c-9b0c-7526b5c7c567",
				Pair:            futuresTestPair,
				Side:            order.Sell,
				Type:            order.TakeProfit,
				Status:          order.Open,
				TriggerPrice:    86000,
				Amount:          0.001,
				RemainingAmount: 0.001,
				Date:            received,
				LastUpdated:     received,
			},
		},
		{
			name: "untouched take profit limit order",
			order: &FuturesOpenOrder{
				OrderID:        "4fd39d86-4af5-4b7a-9bc5-7495be737079",
				Symbol:         "PF_XBTUSD",
				Side:           "sell",
				OrderType:      "take_profit",
				Status:         "untouched",
				LimitPrice:     85900,
				StopPrice:      86000,
				UnfilledSize:   0.001,
				TriggerSignal:  "index",
				ReceivedTime:   received,
				LastUpdateTime: received,
			},
			exp: &order.Detail{
				Exchange:        e.Name,
				AssetType:       asset.Futures,
				OrderID:         "4fd39d86-4af5-4b7a-9bc5-7495be737079",
				Pair:            futuresTestPair,
				Side:            order.Sell,
				Type:            order.TakeProfit | order.Limit,
				Status:          order.Open,
				Price:           85900,
				TriggerPrice:    86000,
				Amount:          0.001,
				RemainingAmount: 0.001,
				Date:            received,
				LastUpdated:     received,
			},
		},
		{
			name: "partly filled stop order",
			order: &FuturesOpenOrder{
				OrderID:        "f8a9b0c1-d2e3-4f4a-9b5c-6d7e8f9a0b1c",
				ClientOrderID:  "gct-stop-3",
				Symbol:         "PF_XBTUSD",
				Side:           "sell",
				OrderType:      "stp",
				Status:         "partiallyFilled",
				LimitPrice:     79000,
				StopPrice:      79100,
				FilledSize:     0.0002,
				UnfilledSize:   0.0003,
				ReduceOnly:     true,
				TriggerSignal:  "mark",
				ReceivedTime:   received,
				LastUpdateTime: updated,
			},
			exp: &order.Detail{
				Exchange:        e.Name,
				AssetType:       asset.Futures,
				OrderID:         "f8a9b0c1-d2e3-4f4a-9b5c-6d7e8f9a0b1c",
				ClientOrderID:   "gct-stop-3",
				Pair:            futuresTestPair,
				Side:            order.Sell,
				Type:            order.StopLimit,
				Status:          order.PartiallyFilled,
				ReduceOnly:      true,
				Price:           79000,
				TriggerPrice:    79100,
				Amount:          0.0005,
				ExecutedAmount:  0.0002,
				RemainingAmount: 0.0003,
				Date:            received,
				LastUpdated:     updated,
			},
		},
	} {
		got, err := e.futuresOpenOrderDetail(tc.order)
		require.NoErrorf(t, err, "futuresOpenOrderDetail must not error for a %s", tc.name)
		assert.Equalf(t, tc.exp, got, "futuresOpenOrderDetail should convert a %s", tc.name)
	}
	valid := FuturesOpenOrder{OrderID: "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b", Symbol: "PF_XBTUSD", Side: "buy", OrderType: "lmt", LimitPrice: 81450, UnfilledSize: 0.0004}
	for _, tc := range []struct {
		name   string
		modify func(*FuturesOpenOrder)
		err    error
	}{
		{"an order on a contract no longer listed", func(o *FuturesOpenOrder) { o.Symbol = "PF_ZORAUSD" }, currency.ErrPairNotFound},
		{"an order without a side", func(o *FuturesOpenOrder) { o.Side = "" }, order.ErrSideIsInvalid},
		{"an unknown order type", func(o *FuturesOpenOrder) { o.OrderType = "moon" }, errUnknownOrderType},
	} {
		o := valid
		tc.modify(&o)
		_, err := e.futuresOpenOrderDetail(&o)
		assert.ErrorIsf(t, err, tc.err, "futuresOpenOrderDetail should reject %s", tc.name)
	}
}

func TestFuturesOrderStatusDetail(t *testing.T) {
	t.Parallel()
	placed, updated := time.Date(2026, 10, 9, 8, 24, 59, 120000000, time.UTC), time.Date(2026, 10, 9, 8, 26, 14, 982000000, time.UTC)
	cached := FuturesCachedOrder{
		Type:           "ORDER",
		OrderID:        "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b",
		ClientOrderID:  "gct-batch-1",
		Symbol:         "PF_XBTUSD",
		Side:           "buy",
		Quantity:       0.0007,
		LimitPrice:     81450,
		ReduceOnly:     true,
		PlacedTime:     placed,
		LastUpdateTime: updated,
	}
	expected := func(status order.Status, filled, remaining, triggerPrice float64) *order.Detail {
		return &order.Detail{
			Exchange:        e.Name,
			AssetType:       asset.Futures,
			OrderID:         "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b",
			ClientOrderID:   "gct-batch-1",
			Pair:            futuresTestPair,
			Side:            order.Buy,
			Status:          status,
			ReduceOnly:      true,
			Price:           81450,
			TriggerPrice:    triggerPrice,
			Amount:          0.0007,
			ExecutedAmount:  filled,
			RemainingAmount: remaining,
			Date:            placed,
			LastUpdated:     updated,
		}
	}
	trigger := &FuturesPriceTriggerOptions{TriggerPrice: 81500, TriggerSide: "TRIGGER_ABOVE", TriggerSignal: "MARK_PRICE"}
	for _, tc := range []struct {
		status  string
		filled  float64
		trigger *FuturesPriceTriggerOptions
		exp     *order.Detail
	}{
		{"ENTERED_BOOK", 0, nil, expected(order.Open, 0, 0.0007, 0)},
		{"ENTERED_BOOK", 0.0003, nil, expected(order.PartiallyFilled, 0.0003, 0.0004, 0)},
		{"TRIGGER_PLACED", 0, trigger, expected(order.Open, 0, 0.0007, 81500)},
		{"FULLY_EXECUTED", 0.0007, nil, expected(order.Filled, 0.0007, 0, 0)},
		{"REJECTED", 0, nil, expected(order.Rejected, 0, 0.0007, 0)},
		{"TRIGGER_ACTIVATION_FAILURE", 0, trigger, expected(order.Rejected, 0, 0.0007, 81500)},
		{"CANCELLED", 0, nil, expected(order.Cancelled, 0, 0.0007, 0)},
		{"CANCELLED", 0.0003, nil, expected(order.PartiallyFilledCancelled, 0.0003, 0.0004, 0)},
	} {
		o := cached
		o.FilledQuantity, o.PriceTriggerOptions = tc.filled, tc.trigger
		got, err := e.futuresOrderStatusDetail(&FuturesOrderStatusDetails{Order: o, Status: tc.status})
		require.NoErrorf(t, err, "futuresOrderStatusDetail must not error for %s with %v filled", tc.status, tc.filled)
		assert.Equalf(t, tc.exp, got, "futuresOrderStatusDetail should convert %s with %v filled", tc.status, tc.filled)
	}
	for _, tc := range []struct {
		name   string
		modify func(*FuturesOrderStatusDetails)
		err    error
	}{
		{"an order on a contract no longer listed", func(s *FuturesOrderStatusDetails) { s.Order.Symbol = "PF_ZORAUSD" }, currency.ErrPairNotFound},
		{"an order without a side", func(s *FuturesOrderStatusDetails) { s.Order.Side = "" }, order.ErrSideIsInvalid},
		{"an unknown status", func(s *FuturesOrderStatusDetails) { s.Status = "PENDING" }, errUnknownOrderStatus},
	} {
		s := FuturesOrderStatusDetails{Order: cached, Status: "ENTERED_BOOK"}
		tc.modify(&s)
		_, err := e.futuresOrderStatusDetail(&s)
		assert.ErrorIsf(t, err, tc.err, "futuresOrderStatusDetail should reject %s", tc.name)
	}
}

func TestFuturesOrderTypeFromString(t *testing.T) {
	t.Parallel()
	for s, exp := range map[string]struct {
		orderType order.Type
		tif       order.TimeInForce
	}{
		"lmt":                    {order.Limit, order.UnknownTIF},
		"Limit":                  {order.Limit, order.UnknownTIF},
		"post":                   {order.Limit, order.PostOnly},
		"Post":                   {order.Limit, order.PostOnly},
		"ioc":                    {order.Limit, order.ImmediateOrCancel},
		"IoC":                    {order.Limit, order.ImmediateOrCancel},
		"HedgeImmediateOrCancel": {order.Limit, order.ImmediateOrCancel},
		"fok":                    {order.Limit, order.FillOrKill},
		"FillOrKill":             {order.Limit, order.FillOrKill},
		"mkt":                    {order.Market, order.UnknownTIF},
		"Market":                 {order.Market, order.UnknownTIF},
		"stp":                    {order.Stop, order.UnknownTIF},
		"stop":                   {order.Stop, order.UnknownTIF},
		"Stop":                   {order.Stop, order.UnknownTIF},
		"take_profit":            {order.TakeProfit, order.UnknownTIF},
		"trailing_stop":          {order.TrailingStop, order.UnknownTIF},
		"liquidation":            {order.Liquidation, order.UnknownTIF},
		"PartialLiquidation":     {order.Liquidation, order.UnknownTIF},
		"CoveredLiquidation":     {order.Liquidation, order.UnknownTIF},
		"Assignment":             {order.UnknownType, order.UnknownTIF},
		"HedgeAssignment":        {order.UnknownType, order.UnknownTIF},
		"Unwind":                 {order.UnknownType, order.UnknownTIF},
		"Block":                  {order.UnknownType, order.UnknownTIF},
		"Rfq":                    {order.UnknownType, order.UnknownTIF},
		"Unknown":                {order.UnknownType, order.UnknownTIF},
	} {
		orderType, tif, err := futuresOrderTypeFromString(s)
		require.NoErrorf(t, err, "futuresOrderTypeFromString must not error for %s", s)
		assert.Equalf(t, exp.orderType, orderType, "futuresOrderTypeFromString should convert %s's order type", s)
		assert.Equalf(t, exp.tif, tif, "futuresOrderTypeFromString should convert %s's time in force", s)
	}
	_, _, err := futuresOrderTypeFromString("moon")
	assert.ErrorIs(t, err, errUnknownOrderType, "futuresOrderTypeFromString should reject an unknown order type")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an unsupported asset")
	failing := newOrdersFailingExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err = failing.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
		assert.ErrorIsf(t, err, errAPIResponse, "GetActiveOrders should return the %s request's error", a)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		for _, a := range e.GetAssetTypes(false) {
			_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			assert.NoErrorf(t, err, "GetActiveOrders should not error for %s", a)
		}
		return
	}
	spotStopLossLimit, futuresLimit := ordersStopLossLimitDetail(e.Name), ordersFuturesLimitDetail(e.Name)
	openTime := time.UnixMicro(1780582311512449)
	spotIceberg := order.Detail{
		Exchange:             e.Name,
		AssetType:            asset.Spot,
		OrderID:              "OHYO67-6LP66-HMQ437",
		ClientOrderID:        "6d1b345e-2821-40e2-ad83-4ecb18a06876",
		Pair:                 ordersETHUSDPair,
		Side:                 order.Sell,
		Type:                 order.Limit,
		Status:               order.PartiallyFilled,
		TimeInForce:          order.GoodTillCancel | order.PostOnly,
		Price:                2650.5,
		AverageExecutedPrice: 2650.7,
		Amount:               10,
		ExecutedAmount:       2,
		RemainingAmount:      8,
		Cost:                 5301.4,
		CostAsset:            currency.USD,
		Fee:                  8.48224,
		FeeAsset:             currency.USD,
		Date:                 openTime,
		LastUpdated:          openTime,
	}
	futuresStop := order.Detail{
		Exchange:        e.Name,
		AssetType:       asset.Futures,
		OrderID:         "f8a9b0c1-d2e3-4f4a-9b5c-6d7e8f9a0b1c",
		ClientOrderID:   "gct-stop-3",
		Pair:            futuresTestPair,
		Side:            order.Sell,
		Type:            order.StopLimit,
		Status:          order.PartiallyFilled,
		Price:           79000,
		TriggerPrice:    79100,
		Amount:          0.0005,
		ExecutedAmount:  0.0002,
		RemainingAmount: 0.0003,
		Date:            time.Date(2026, 10, 9, 8, 24, 59, 122000000, time.UTC),
		LastUpdated:     time.Date(2026, 10, 9, 8, 26, 15, 4000000, time.UTC),
	}
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  order.FilteredOrders
	}{
		{"spot orders", &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}, order.FilteredOrders{spotIceberg, spotStopLossLimit}},
		{"a spot pair's orders", &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{spotTestPair}, Side: order.AnySide, Type: order.AnyType}, order.FilteredOrders{spotStopLossLimit}},
		{"futures orders", &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType}, order.FilteredOrders{futuresLimit, futuresStop}},
		{"futures sell orders", &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.Sell, Type: order.AnyType}, order.FilteredOrders{futuresStop}},
		{
			name: "a futures pair without open orders",
			req:  &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{currency.NewPairWithDelimiter("PF", "ETHUSD", "_")}, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetActiveOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetActiveOrders must not error")
			// Get Open Orders keys spot orders by ID, so they come in no particular order
			slices.SortFunc(got, func(a, b order.Detail) int { return strings.Compare(a.OrderID, b.OrderID) })
			assert.Equal(t, tc.exp, got, "GetActiveOrders should return the open orders requested")
		})
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an unsupported asset")
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: a,
			Side:      order.AnySide,
			Type:      order.AnyType,
			StartTime: time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		})
		require.ErrorIsf(t, err, common.ErrStartAfterEnd, "GetOrderHistory must reject a %s window ending before it starts", a)
	}
	failing := newOrdersFailingExchange(t)
	_, err = failing.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errAPIResponse, "GetOrderHistory should return Get Closed Orders' error")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		for _, a := range e.GetAssetTypes(false) {
			_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now()})
			assert.NoErrorf(t, err, "GetOrderHistory should not error for %s", a)
		}
		return
	}
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  order.FilteredOrders
	}{
		{
			name: "spot orders across two pages, leaving out an order on a pair no longer listed",
			req: &order.MultiOrderRequest{
				AssetType: asset.Spot,
				Side:      order.AnySide,
				Type:      order.AnyType,
				StartTime: time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC),
			},
			exp: order.FilteredOrders{
				{
					Exchange:             e.Name,
					AssetType:            asset.Spot,
					OrderID:              "O6KIO9-Y73RV-6LFAIX",
					ClientOrderID:        "gct-history-1",
					Pair:                 wsTestPair,
					Side:                 order.Buy,
					Type:                 order.Limit,
					Status:               order.Filled,
					TimeInForce:          order.GoodTillCancel,
					Price:                81250,
					AverageExecutedPrice: 81250,
					Amount:               0.04,
					ExecutedAmount:       0.04,
					Cost:                 3250,
					CostAsset:            currency.USD,
					Fee:                  8.45,
					FeeAsset:             currency.USD,
					Date:                 time.UnixMicro(1791421234551200),
					CloseTime:            time.UnixMicro(1791421301012700),
					LastUpdated:          time.UnixMicro(1791421301012700),
				},
				{
					Exchange:             e.Name,
					AssetType:            asset.Spot,
					OrderID:              "OONG6B-IWSS3-9K3G5D",
					Pair:                 wsTestPair,
					Side:                 order.Sell,
					Type:                 order.Market,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					AverageExecutedPrice: 81330,
					Amount:               0.015,
					ExecutedAmount:       0.015,
					Cost:                 1219.95,
					CostAsset:            currency.USD,
					Fee:                  3.17187,
					FeeAsset:             currency.USD,
					Date:                 time.UnixMicro(1791505810660300),
					CloseTime:            time.UnixMicro(1791505810663100),
					LastUpdated:          time.UnixMicro(1791505810663100),
				},
				{
					Exchange:             e.Name,
					AssetType:            asset.Spot,
					OrderID:              "OXVZKC-2J609-ZTQOBP",
					Pair:                 ordersXBTGBPPair,
					Side:                 order.Sell,
					Type:                 order.TakeProfit | order.Limit,
					Status:               order.PartiallyFilledCancelled,
					TimeInForce:          order.GoodTillCancel,
					Price:                61900,
					TriggerPrice:         62000,
					AverageExecutedPrice: 61900,
					Amount:               0.5,
					ExecutedAmount:       0.25,
					RemainingAmount:      0.25,
					Cost:                 15475,
					CostAsset:            currency.GBP,
					Fee:                  40.235,
					FeeAsset:             currency.GBP,
					Date:                 time.UnixMicro(1791450011002100),
					CloseTime:            time.UnixMicro(1791463950771800),
					LastUpdated:          time.UnixMicro(1791463950771800),
				},
				{
					Exchange:        e.Name,
					AssetType:       asset.Spot,
					OrderID:         "OYGVH6-M8JNT-0PI3GN",
					Pair:            ordersETHUSDPair,
					Side:            order.Sell,
					Type:            order.Limit,
					Status:          order.Expired,
					TimeInForce:     order.GoodTillTime,
					Price:           2900,
					Amount:          1.5,
					RemainingAmount: 1.5,
					CostAsset:       currency.USD,
					FeeAsset:        currency.USD,
					Date:            time.UnixMilli(1791490000250),
					CloseTime:       time.UnixMicro(1791504000000400),
					LastUpdated:     time.UnixMicro(1791504000000400),
				},
			},
		},
		{
			name: "futures orders as their last events across two pages left them, leaving out an order on a contract no longer listed",
			req: &order.MultiOrderRequest{
				AssetType: asset.Futures,
				Side:      order.AnySide,
				Type:      order.AnyType,
				StartTime: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC),
			},
			exp: order.FilteredOrders{
				{
					Exchange:        e.Name,
					AssetType:       asset.Futures,
					OrderID:         "7a1b2c3d-4e5f-4a6b-9c7d-8e9f0a1b2c3d",
					ClientOrderID:   "gct-order-1",
					Pair:            futuresTestPair,
					Side:            order.Buy,
					Type:            order.Limit,
					TimeInForce:     order.PostOnly,
					Status:          order.Open,
					Price:           81050,
					Amount:          0.4,
					RemainingAmount: 0.4,
					Date:            time.UnixMilli(1791504010000),
					LastUpdated:     time.UnixMilli(1791504020000),
				},
				{
					Exchange:        e.Name,
					AssetType:       asset.Futures,
					OrderID:         "a2f0c1d2-3e4f-4a5b-8c6d-7e8f9a0b1c3e",
					ClientOrderID:   "gct-history-3",
					Pair:            futuresTestPair,
					Side:            order.Sell,
					Type:            order.Limit,
					TimeInForce:     order.PostOnly,
					Status:          order.Rejected,
					ReduceOnly:      true,
					Price:           81700,
					Amount:          0.75,
					RemainingAmount: 0.75,
					Date:            time.UnixMilli(1791504090000),
					LastUpdated:     time.UnixMilli(1791504090001),
				},
				{
					Exchange:        e.Name,
					AssetType:       asset.Futures,
					OrderID:         "44e6100d-b5b9-4b44-9b1b-adf08794d53f",
					ClientOrderID:   "gct-history-2",
					Pair:            futuresTestPair,
					Side:            order.Sell,
					Type:            order.Limit,
					Status:          order.PartiallyFilled,
					ReduceOnly:      true,
					Price:           81950,
					Amount:          1.5,
					ExecutedAmount:  0.25,
					RemainingAmount: 1.25,
					Date:            time.UnixMilli(1791504100000),
					LastUpdated:     time.UnixMilli(1791504160000),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetOrderHistory must not error")
			if tc.req.AssetType == asset.Spot {
				// Get Closed Orders keys orders by ID, so they come in no particular order
				slices.SortFunc(got, func(a, b order.Detail) int { return strings.Compare(a.OrderID, b.OrderID) })
			}
			assert.Equal(t, tc.exp, got, "GetOrderHistory should return the orders in the window")
		})
	}
}

func TestFuturesOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.futuresOrderHistory(t.Context(), time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC), time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "futuresOrderHistory must reject a window ending before it starts")
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/history/v3/orders", r.URL.Path, "futuresOrderHistory should list order events")
		assert.Equal(t, "before=1791507600000&since=1791504000000&sort=asc", r.URL.RawQuery, "futuresOrderHistory should list the window's events oldest first")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"unauthorized","reason":"You are not authorized to access this endpoint."}`))
	})
	_, err = ex.futuresOrderHistory(t.Context(), time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC))
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "futuresOrderHistory should return Get order events' error")

	var mu sync.Mutex
	var paths []string
	ex = newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		assert.Equal(t, "before=1791507600000&since=1791504000000&sort=asc", r.URL.RawQuery, "futuresOrderHistory should list the window's events oldest first")
		if r.URL.Path == "/api/history/v3/orders" {
			_, _ = w.Write([]byte(futuresHistoryPage("")))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"unauthorized","reason":"You are not authorized to access this endpoint."}`))
	})
	_, err = ex.futuresOrderHistory(t.Context(), time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC))
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "futuresOrderHistory should return Get trigger events' error")
	mu.Lock()
	assert.Equal(t, []string{"/api/history/v3/orders", "/api/history/v3/triggers"}, paths, "futuresOrderHistory should list the window's trigger events after its order events")
	mu.Unlock()
}

// newFuturesHistoryExchange returns an exchange named after the test whose order and trigger events are the listings
// orders and triggers
func newFuturesHistoryExchange(t *testing.T, orders, triggers *historyListing) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/history/v3/orders":
			orders.serve(w, r)
		case "/api/history/v3/triggers":
			triggers.serve(w, r)
		default:
			assert.Failf(t, "only the order and trigger events endpoints should be called", "%s", r.URL.Path)
		}
	})
}

// futuresHistoryTriggerJSON returns a reduce-only sell stop, triggered below 80500 by the mark price for a limit order
// at 80000, as trigger events report it
func futuresHistoryTriggerJSON(id, tradeable, clientID, quantity string, placed, updated time.Time) string {
	return `{"uid":"` + id + `","accountId":4815162342,"accountUid":"` + futuresHistoryTestAccount + `","tradeable":"` + tradeable + `","direction":"Sell","quantity":"` + quantity +
		`","timestamp":` + strconv.FormatInt(placed.UnixMilli(), 10) + `,"limitPrice":"80000","orderType":"Limit","clientId":"` + clientID +
		`","reduceOnly":true,"lastUpdateTimestamp":` + strconv.FormatInt(updated.UnixMilli(), 10) +
		`,"triggerOptions":{"triggerPrice":"80500","triggerSignal":"MarkPrice","triggerSide":"Below","trailingStopOptions":{"maxDeviation":"0","unit":"Percent"},` +
		`"limitPriceOffset":{"priceOffset":"0","unit":"QuoteCurrency"}}}`
}

func TestGetOrderHistoryFuturesTriggerOrders(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1791518400000)
	at := start.Add
	const (
		limitID              = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
		cancelledStopID      = "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e"
		triggeredStopID      = "2c3d4e5f-6a7b-4c8d-ae9f-1a2b3c4d5e6f"
		lateActivationID     = "3d4e5f6a-7b8c-4d9e-bfa0-2b3c4d5e6f70"
		bothListedID         = "4e5f6a7b-8c9d-4eaf-80b1-3c4d5e6f7081"
		tiedID               = "5f6a7b8c-9dae-4fb0-91c2-4d5e6f708192"
		delistedID           = "6a7b8c9d-aebf-40c1-a2d3-5e6f708192a3"
		triggeredElsewhereID = "7b8c9dae-bfc0-41d2-b3e4-6f708192a3b4"
		placedByTriggerID    = "8c9daebf-c0d1-42e3-c4f5-708192a3b4c5"
		editedStopID         = "9daebfc0-d1e2-43f4-8506-8192a3b4c5d6"
	)
	event := func(kind string, when time.Time, body string) string {
		return futuresHistoryElement("ev-"+strconv.FormatInt(when.UnixMilli(), 10)+"-"+kind, when, kind, body)
	}
	placedEvent := func(o string, when time.Time) string {
		return event("OrderPlaced", when, `{"order":`+o+`,"reason":"new_user_order","reducedQuantity":""}`)
	}
	updatedEvent := func(old, o string, when time.Time) string {
		return event("OrderUpdated", when, `{"oldOrder":`+old+`,"newOrder":`+o+`,"reason":"full_fill","reducedQuantity":"0"}`)
	}
	triggerEvent := func(kind, trigger string, when time.Time) string {
		reason := "new_user_order"
		if kind == "OrderTriggerCancelled" {
			reason = "cancelled_by_user"
		}
		return event(kind, when, `{"order":`+trigger+`,"reason":"`+reason+`"}`)
	}
	limitPlaced := futuresHistoryOrderJSON(limitID, "gct-limit-1", "Buy", "Limit", "0.5", "0", at(time.Minute), at(time.Minute))
	limitFilled := futuresHistoryOrderJSON(limitID, "gct-limit-1", "Buy", "Limit", "0.5", "0.25", at(time.Minute), at(2*time.Minute))
	triggeredPlaced := futuresHistoryOrderJSON(triggeredStopID, "gct-stop-2", "Sell", "Stop", "0.25", "0", at(6*time.Minute), at(6*time.Minute))
	triggeredFilled := futuresHistoryOrderJSON(triggeredStopID, "gct-stop-2", "Sell", "Stop", "0.25", "0.25", at(6*time.Minute), at(6*time.Minute))
	latePlaced := futuresHistoryOrderJSON(lateActivationID, "gct-stop-3", "Sell", "Stop", "0.125", "0", at(8*time.Minute), at(8*time.Minute))
	lateFilled := futuresHistoryOrderJSON(lateActivationID, "gct-stop-3", "Sell", "Stop", "0.125", "0.125", at(8*time.Minute), at(8*time.Minute+100*time.Millisecond))
	tiedPlaced := futuresHistoryOrderJSON(tiedID, "gct-stop-5", "Sell", "Stop", "0.5", "0", at(11*time.Minute), at(11*time.Minute))
	placedByTrigger := futuresHistoryOrderJSON(placedByTriggerID, "gct-stop-7", "Sell", "Stop", "0.25", "0", at(15*time.Minute), at(15*time.Minute))
	placedByTriggerFilled := futuresHistoryOrderJSON(placedByTriggerID, "gct-stop-7", "Sell", "Stop", "0.25", "0.25", at(15*time.Minute), at(15*time.Minute))
	orderPages := [][]string{
		{
			placedEvent(limitPlaced, at(time.Minute)),
			event("OrderUpdated", at(2*time.Minute), `{"oldOrder":`+limitPlaced+`,"newOrder":`+limitFilled+`,"reason":"partial_fill","reducedQuantity":"0"}`),
			placedEvent(triggeredPlaced, at(6*time.Minute)),
			updatedEvent(triggeredPlaced, triggeredFilled, at(6*time.Minute)),
		},
		{
			placedEvent(latePlaced, at(8*time.Minute)),
			updatedEvent(latePlaced, lateFilled, at(8*time.Minute+100*time.Millisecond)),
			placedEvent(futuresHistoryOrderJSON(bothListedID, "gct-stop-4", "Sell", "Stop", "0.5", "0", at(9*time.Minute), at(9*time.Minute)), at(9*time.Minute)),
			placedEvent(tiedPlaced, at(11*time.Minute)),
			event("OrderCancelled", at(12*time.Minute), `{"order":`+tiedPlaced+`,"reason":"cancelled_by_user"}`),
			placedEvent(placedByTrigger, at(15*time.Minute)),
			updatedEvent(placedByTrigger, placedByTriggerFilled, at(15*time.Minute)),
		},
	}
	cancelledStop := futuresHistoryTriggerJSON(cancelledStopID, "PF_XBTUSD", "gct-stop-1", "0.5", at(3*time.Minute), at(3*time.Minute))
	triggered := futuresHistoryTriggerJSON(triggeredStopID, "PF_XBTUSD", "gct-stop-2", "0.25", at(5*time.Minute), at(5*time.Minute))
	late := futuresHistoryTriggerJSON(lateActivationID, "PF_XBTUSD", "gct-stop-3", "0.125", at(7*time.Minute), at(7*time.Minute))
	bothListed := futuresHistoryTriggerJSON(bothListedID, "PF_XBTUSD", "gct-stop-4", "0.5", at(9*time.Minute), at(9*time.Minute))
	tied := futuresHistoryTriggerJSON(tiedID, "PF_XBTUSD", "gct-stop-5", "0.5", at(11*time.Minute), at(11*time.Minute))
	tiedEdited := futuresHistoryTriggerJSON(tiedID, "PF_XBTUSD", "gct-stop-5", "0.5", at(11*time.Minute), at(12*time.Minute))
	triggeredElsewhere := futuresHistoryTriggerJSON(triggeredElsewhereID, "PF_XBTUSD", "gct-stop-6", "0.25", at(14*time.Minute), at(14*time.Minute))
	edited := futuresHistoryTriggerJSON(editedStopID, "PF_XBTUSD", "gct-stop-9", "0.5", at(16*time.Minute), at(16*time.Minute))
	resized := futuresHistoryTriggerJSON(editedStopID, "PF_XBTUSD", "gct-stop-9", "0.75", at(16*time.Minute), at(17*time.Minute))
	triggerPages := [][]string{
		{
			triggerEvent("OrderTriggerPlaced", cancelledStop, at(3*time.Minute)),
			triggerEvent("OrderTriggerCancelled", cancelledStop, at(4*time.Minute)),
			triggerEvent("OrderTriggerPlaced", triggered, at(5*time.Minute)),
		},
		{
			// The stop placed on the page before triggers, as do others, each placing an order
			event("OrderTriggerActivated", at(6*time.Minute), `{"order":`+triggered+`}`),
			triggerEvent("OrderTriggerPlaced", late, at(7*time.Minute)),
			// Timed after the order this stop placed filled, which an activation must not override
			event("OrderTriggerActivated", at(8*time.Minute+500*time.Millisecond), `{"order":`+late+`}`),
			// Listed among order events as well, if Kraken lists a stop there before it triggers, where it was placed
			// before it was cancelled
			triggerEvent("OrderTriggerPlaced", bothListed, at(9*time.Minute)),
			triggerEvent("OrderTriggerCancelled", bothListed, at(10*time.Minute)),
			// Edited at the time the same order's order events cancel it
			triggerEvent("OrderTriggerPlaced", tied, at(11*time.Minute)),
			event("OrderTriggerUpdated", at(12*time.Minute), `{"oldOrderTrigger":`+tied+`,"newOrderTrigger":`+tiedEdited+`,"reason":"edited_by_user"}`),
			triggerEvent("OrderTriggerPlaced", futuresHistoryTriggerJSON(delistedID, "PF_ZORAUSD", "gct-stop-8", "0.5", at(13*time.Minute), at(13*time.Minute)), at(13*time.Minute)),
			// Triggered, placing an order under its own ID
			triggerEvent("OrderTriggerPlaced", triggeredElsewhere, at(14*time.Minute)),
			event("OrderTriggerActivated", at(15*time.Minute), `{"order":`+triggeredElsewhere+`}`),
			// Resized, and still waiting to trigger when the window ends
			triggerEvent("OrderTriggerPlaced", edited, at(16*time.Minute)),
			event("OrderTriggerUpdated", at(17*time.Minute), `{"oldOrderTrigger":`+edited+`,"newOrderTrigger":`+resized+`,"reason":"edited_by_user"}`),
		},
	}
	futuresOrder := func(exchangeName, id, clientID string, side order.Side, orderType order.Type, status order.Status, amount, executed float64, placed, updated time.Time) order.Detail {
		return order.Detail{
			Exchange:        exchangeName,
			AssetType:       asset.Futures,
			OrderID:         id,
			ClientOrderID:   clientID,
			Pair:            futuresTestPair,
			Side:            side,
			Type:            orderType,
			Status:          status,
			Price:           81450,
			Amount:          amount,
			ExecutedAmount:  executed,
			RemainingAmount: amount - executed,
			Date:            placed,
			LastUpdated:     updated,
		}
	}
	triggerOrder := func(exchangeName, id, clientID string, status order.Status, amount float64, placed, updated time.Time) order.Detail {
		return order.Detail{
			Exchange:        exchangeName,
			AssetType:       asset.Futures,
			OrderID:         id,
			ClientOrderID:   clientID,
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Status:          status,
			ReduceOnly:      true,
			Price:           80000,
			TriggerPrice:    80500,
			Amount:          amount,
			RemainingAmount: amount,
			Date:            placed,
			LastUpdated:     updated,
		}
	}
	// The orders only order events list, as order events alone report them
	orderEventsOnly := func(exchangeName string) []order.Detail {
		return []order.Detail{
			futuresOrder(exchangeName, limitID, "gct-limit-1", order.Buy, order.Limit, order.PartiallyFilled, 0.5, 0.25, at(time.Minute), at(2*time.Minute)),
			futuresOrder(exchangeName, triggeredStopID, "gct-stop-2", order.Sell, order.Stop, order.Filled, 0.25, 0.25, at(6*time.Minute), at(6*time.Minute)),
			futuresOrder(exchangeName, lateActivationID, "gct-stop-3", order.Sell, order.Stop, order.Filled, 0.125, 0.125, at(8*time.Minute), at(8*time.Minute+100*time.Millisecond)),
			futuresOrder(exchangeName, tiedID, "gct-stop-5", order.Sell, order.Stop, order.Cancelled, 0.5, 0, at(11*time.Minute), at(12*time.Minute)),
			futuresOrder(exchangeName, placedByTriggerID, "gct-stop-7", order.Sell, order.Stop, order.Filled, 0.25, 0.25, at(15*time.Minute), at(15*time.Minute)),
		}
	}
	query, nextQuery := "before=1791522000000&since=1791518400000&sort=asc", "before=1791522000000&continuation_token=page2&since=1791518400000&sort=asc"
	for _, tc := range []struct {
		name     string
		triggers [][]string
		exp      func(exchangeName string) order.FilteredOrders
	}{
		{
			name:     "trigger events",
			triggers: triggerPages,
			exp: func(exchangeName string) order.FilteredOrders {
				o := orderEventsOnly(exchangeName)
				return order.FilteredOrders{
					o[0], o[1], o[2],
					triggerOrder(exchangeName, bothListedID, "gct-stop-4", order.Cancelled, 0.5, at(9*time.Minute), at(10*time.Minute)),
					o[3], o[4],
					triggerOrder(exchangeName, cancelledStopID, "gct-stop-1", order.Cancelled, 0.5, at(3*time.Minute), at(4*time.Minute)),
					triggerOrder(exchangeName, editedStopID, "gct-stop-9", order.Open, 0.75, at(16*time.Minute), at(17*time.Minute)),
				}
			},
		},
		{
			name:     "no trigger events",
			triggers: [][]string{{}},
			exp: func(exchangeName string) order.FilteredOrders {
				o := orderEventsOnly(exchangeName)
				return order.FilteredOrders{
					o[0], o[1], o[2],
					futuresOrder(exchangeName, bothListedID, "gct-stop-4", order.Sell, order.Stop, order.Open, 0.5, 0, at(9*time.Minute), at(9*time.Minute)),
					o[3], o[4],
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orders, triggers := newHistoryListing(orderPages...), newHistoryListing(tc.triggers...)
			ex := newFuturesHistoryExchange(t, orders, triggers)
			got, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType, StartTime: start, EndTime: at(time.Hour)})
			require.NoError(t, err, "GetOrderHistory must not error")
			assert.Equal(t, tc.exp(ex.Name), got, "GetOrderHistory should merge each order's order and trigger events by order ID, the later event winning, leaving a stop that triggered to its order events")
			assert.Equal(t, []string{query, nextQuery}, orders.queries(), "GetOrderHistory should list every page of the window's order events")
			expTriggerQueries := []string{query}
			if len(tc.triggers) > 1 {
				expTriggerQueries = append(expTriggerQueries, nextQuery)
			}
			assert.Equal(t, expTriggerQueries, triggers.queries(), "GetOrderHistory should list every page of the window's trigger events")
		})
	}
}

func TestFuturesHistoryTriggerDetail(t *testing.T) {
	t.Parallel()
	placed, updated := time.UnixMilli(1791518580000), time.UnixMilli(1791518600000)
	trigger := FuturesHistoryTrigger{
		UID:            "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e",
		AccountID:      4815162342,
		AccountUID:     "4b2e9c1a-7f3d-4e8b-9a6c-2d1f5e8b7c30",
		Tradeable:      "PF_XBTUSD",
		Direction:      "Buy",
		Quantity:       0.75,
		Timestamp:      types.Time(placed),
		LimitPrice:     83000,
		OrderType:      "Limit",
		ClientOrderID:  "gct-stop-1",
		LastUpdateTime: types.Time(updated),
		TriggerOptions: FuturesHistoryTriggerOptions{TriggerPrice: 82900, TriggerSignal: "LastPrice", TriggerSide: "Above"},
	}
	exp := &order.Detail{
		Exchange:        e.Name,
		AssetType:       asset.Futures,
		OrderID:         "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e",
		ClientOrderID:   "gct-stop-1",
		Pair:            futuresTestPair,
		Side:            order.Buy,
		Status:          order.Cancelled,
		Price:           83000,
		TriggerPrice:    82900,
		Amount:          0.75,
		RemainingAmount: 0.75,
		Date:            placed,
		LastUpdated:     updated,
	}
	got, err := e.futuresHistoryTriggerDetail(&trigger, order.Cancelled)
	require.NoError(t, err, "futuresHistoryTriggerDetail must not error")
	assert.Equal(t, exp, got, "futuresHistoryTriggerDetail should convert a trigger order")

	undecoded := trigger
	undecoded.Direction = "Unknown"
	got, err = e.futuresHistoryTriggerDetail(&undecoded, order.Cancelled)
	require.NoError(t, err, "futuresHistoryTriggerDetail must not error for a direction Kraken could not decode")
	exp.Side = order.UnknownSide
	assert.Equal(t, exp, got, "futuresHistoryTriggerDetail should leave a direction Kraken could not decode unknown")

	for _, tc := range []struct {
		name   string
		modify func(*FuturesHistoryTrigger)
		err    error
	}{
		{"a trigger order on a contract no longer listed", func(o *FuturesHistoryTrigger) { o.Tradeable = "PF_ZORAUSD" }, currency.ErrPairNotFound},
		{"a trigger order without a direction", func(o *FuturesHistoryTrigger) { o.Direction = "" }, order.ErrSideIsInvalid},
	} {
		o := trigger
		tc.modify(&o)
		_, err := e.futuresHistoryTriggerDetail(&o, order.Open)
		assert.ErrorIsf(t, err, tc.err, "futuresHistoryTriggerDetail should reject %s", tc.name)
	}
}

func TestFuturesHistoryOrderDetail(t *testing.T) {
	t.Parallel()
	placed, updated := time.UnixMilli(1791504100000), time.UnixMilli(1791504160000)
	history := FuturesHistoryOrder{
		UID:            "44e6100d-b5b9-4b44-9b1b-adf08794d53f",
		AccountUID:     "4b2e9c1a-7f3d-4e8b-9a6c-2d1f5e8b7c30",
		Tradeable:      "PF_XBTUSD",
		Direction:      "Sell",
		Quantity:       1.5,
		Timestamp:      types.Time(placed),
		LimitPrice:     81950,
		OrderType:      "Post",
		ClientOrderID:  "gct-history-2",
		ReduceOnly:     true,
		LastUpdateTime: types.Time(updated),
	}
	for _, tc := range []struct {
		status            order.Status
		filled, remaining float64
		exp               order.Status
	}{
		{order.Open, 0, 1.5, order.Open},
		{order.Open, 0.25, 1.25, order.PartiallyFilled},
		{order.Open, 1.5, 0, order.Filled},
		{order.Cancelled, 0, 1.5, order.Cancelled},
		{order.Cancelled, 0.25, 1.25, order.PartiallyFilledCancelled},
		{order.Rejected, 0, 1.5, order.Rejected},
	} {
		o := history
		o.FilledQuantity = types.Number(tc.filled)
		got, err := e.futuresHistoryOrderDetail(&o, tc.status)
		require.NoErrorf(t, err, "futuresHistoryOrderDetail must not error for a %s event with %v filled", tc.status, tc.filled)
		exp := &order.Detail{
			Exchange:        e.Name,
			AssetType:       asset.Futures,
			OrderID:         "44e6100d-b5b9-4b44-9b1b-adf08794d53f",
			ClientOrderID:   "gct-history-2",
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Type:            order.Limit,
			TimeInForce:     order.PostOnly,
			Status:          tc.exp,
			ReduceOnly:      true,
			Price:           81950,
			Amount:          1.5,
			ExecutedAmount:  tc.filled,
			RemainingAmount: tc.remaining,
			Date:            placed,
			LastUpdated:     updated,
		}
		assert.Equalf(t, exp, got, "futuresHistoryOrderDetail should convert a %s event with %v filled", tc.status, tc.filled)
	}
	for _, tc := range []struct {
		name   string
		modify func(*FuturesHistoryOrder)
		err    error
	}{
		{"an order on a contract no longer listed", func(o *FuturesHistoryOrder) { o.Tradeable = "PF_ZORAUSD" }, currency.ErrPairNotFound},
		{"an order without a direction", func(o *FuturesHistoryOrder) { o.Direction = "" }, order.ErrSideIsInvalid},
		{"an unknown order type", func(o *FuturesHistoryOrder) { o.OrderType = "Moon" }, errUnknownOrderType},
	} {
		o := history
		tc.modify(&o)
		_, err := e.futuresHistoryOrderDetail(&o, order.Open)
		assert.ErrorIsf(t, err, tc.err, "futuresHistoryOrderDetail should reject %s", tc.name)
	}
}

// newOrdersRequestCheckExchange returns an exchange named after the test whose REST endpoints expect one request at
// path, carrying params: a spot private request's JSON body without its nonce, or a futures request's query string, and
// reply with reply
func newOrdersRequestCheckExchange(t *testing.T, path, params, reply string) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path, "the request should go to the expected endpoint")
		if strings.HasPrefix(path, "/derivatives/") {
			assert.Equal(t, http.MethodPost, r.Method, "Method should be POST")
			assert.Equal(t, params, r.URL.Query().Encode(), "the request should carry the expected parameters")
		} else {
			assert.JSONEq(t, params, tradingRequestBody(t, r), "the request should carry the expected parameters")
		}
		_, _ = w.Write([]byte(reply))
	})
}

func TestSubmitOrderQuoteSizedMarketBuy(t *testing.T) {
	t.Parallel()
	ex := newOrdersRequestCheckExchange(t, "/0/private/AddOrder",
		`{"oflags":"viqc","ordertype":"market","pair":"XBTUSD","type":"buy","volume":"100"}`,
		`{"error":[],"result":{"descr":{"order":"buy 100.00000000 XBTUSD @ market"},"txid":["ODV663-X7J9B-IUFH4O"]}}`)
	// Without round trips the mock fails the test on any websocket request
	useOrdersTestWebsocket(t, ex)
	submitted := time.Now()
	resp, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.Market, Side: order.Buy, QuoteAmount: 100})
	require.NoError(t, err, "SubmitOrder must not error")
	ordersClearSubmitTimes(t, resp, submitted)
	exp := &order.SubmitResponse{
		Exchange:    ex.Name,
		Type:        order.Market,
		Side:        order.Buy,
		Pair:        spotTestPair,
		AssetType:   asset.Spot,
		QuoteAmount: 100,
		Status:      order.Filled,
		OrderID:     "ODV663-X7J9B-IUFH4O",
	}
	assert.Equal(t, exp, resp, "SubmitOrder should place a quote-sized market buy over REST while the private websocket is connected")
}

func TestSubmitOrderUntriggeredOrders(t *testing.T) {
	t.Parallel()
	futuresPlaced := func(orderID, orderType, side string, quantity, limitPrice float64) string {
		order := `{"algoId":null,"cliOrdId":null,"filled":0,"lastUpdateTimestamp":"2026-10-09T08:33:01.104Z","limitPrice":` + strconv.FormatFloat(limitPrice, 'f', -1, 64) +
			`,"orderId":"` + orderID + `","quantity":` + strconv.FormatFloat(quantity, 'f', -1, 64) + `,"reduceOnly":false,"side":"` + side +
			`","symbol":"PF_XBTUSD","timestamp":"2026-10-09T08:33:01.104Z","type":"` + orderType + `"}`
		return `{"result":"success","sendStatus":{"order_id":"` + orderID + `","orderEvents":[{"order":` + order + `,"reducedQuantity":null,"type":"PLACE"}],` +
			`"receivedTime":"2026-10-09T08:33:01.104Z","status":"placed"},"serverTime":"2026-10-09T08:33:01.107Z"}`
	}
	futuresExecuted := func(orderID, clientOrderID, orderType, side string, quantity, limitPrice, price float64, reduceOnly bool) string {
		client := "null"
		if clientOrderID != "" {
			client = `"` + clientOrderID + `"`
		}
		order := `{"algoId":null,"cliOrdId":` + client + `,"filled":0,"lastUpdateTimestamp":"2026-10-09T08:34:12.551Z","limitPrice":` + strconv.FormatFloat(limitPrice, 'f', -1, 64) +
			`,"orderId":"` + orderID + `","quantity":` + strconv.FormatFloat(quantity, 'f', -1, 64) + `,"reduceOnly":` + strconv.FormatBool(reduceOnly) + `,"side":"` + side +
			`","symbol":"PF_XBTUSD","timestamp":"2026-10-09T08:34:12.551Z","type":"` + orderType + `"}`
		return `{"result":"success","sendStatus":{"order_id":"` + orderID + `","orderEvents":[{"amount":` + strconv.FormatFloat(quantity, 'f', -1, 64) +
			`,"executionId":"0c7d1e5a-3f2b-4a96-8e1d-6b4c9a2f7e30","orderPriorEdit":null,"orderPriorExecution":` + order + `,"price":` + strconv.FormatFloat(price, 'f', -1, 64) +
			`,"takerReducedQuantity":null,"type":"EXECUTION"}],"receivedTime":"2026-10-09T08:34:12.551Z","status":"placed"},"serverTime":"2026-10-09T08:34:12.554Z"}`
	}
	spotPlaced := func(txid, descr string) string {
		return `{"error":[],"result":{"descr":{"order":"` + descr + `"},"txid":["` + txid + `"]}}`
	}
	for _, tc := range []struct {
		name    string
		submit  *order.Submit
		path    string
		request string
		reply   string
		exp     *order.SubmitResponse
		err     error
	}{
		{
			name:    "spot limit order",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 80000, Amount: 0.5, TimeInForce: order.GoodTillCancel},
			path:    "/0/private/AddOrder",
			request: `{"ordertype":"limit","pair":"XBTUSD","price":"80000","type":"buy","volume":"0.5"}`,
			reply:   spotPlaced("OT83XF-1Y4TH-OTWZWB", "buy 0.50000000 XBTUSD @ limit 80000.0"),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, TimeInForce: order.GoodTillCancel, Price: 80000, Amount: 0.5, Status: order.New, OrderID: "OT83XF-1Y4TH-OTWZWB"},
		},
		{
			name:    "spot post only order with a client order ID",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Sell, Price: 90000, Amount: 0.5, TimeInForce: order.PostOnly, ClientOrderID: "gct-post-1"},
			path:    "/0/private/AddOrder",
			request: `{"cl_ord_id":"gct-post-1","oflags":"post","ordertype":"limit","pair":"XBTUSD","price":"90000","type":"sell","volume":"0.5"}`,
			reply:   spotPlaced("OIPJSD-XXGKZ-RTDNG1", "sell 0.50000000 XBTUSD @ limit 90000.0"),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Sell, TimeInForce: order.PostOnly, Price: 90000, Amount: 0.5, ClientOrderID: "gct-post-1", Status: order.New, OrderID: "OIPJSD-XXGKZ-RTDNG1"},
		},
		{
			name:    "spot immediate or cancel order",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 81000, Amount: 0.5, TimeInForce: order.ImmediateOrCancel},
			path:    "/0/private/AddOrder",
			request: `{"ordertype":"limit","pair":"XBTUSD","price":"81000","timeinforce":"IOC","type":"buy","volume":"0.5"}`,
			reply:   spotPlaced("O6ILXG-WV335-W1H72M", "buy 0.50000000 XBTUSD @ limit 81000.0"),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, TimeInForce: order.ImmediateOrCancel, Price: 81000, Amount: 0.5, Status: order.New, OrderID: "O6ILXG-WV335-W1H72M"},
		},
		{
			name:    "spot fill or kill order",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 81000, Amount: 0.5, TimeInForce: order.FillOrKill},
			path:    "/0/private/AddOrder",
			request: `{"ordertype":"limit","pair":"XBTUSD","price":"81000","timeinforce":"FOK","type":"buy","volume":"0.5"}`,
			reply:   spotPlaced("O6XQV8-FNDCO-0DF6F7", "buy 0.50000000 XBTUSD @ limit 81000.0"),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, TimeInForce: order.FillOrKill, Price: 81000, Amount: 0.5, Status: order.New, OrderID: "O6XQV8-FNDCO-0DF6F7"},
		},
		{
			name:    "spot good till time order",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Sell, Price: 90000, Amount: 0.5, TimeInForce: order.GoodTillTime, EndTime: ordersTestExpiry},
			path:    "/0/private/AddOrder",
			request: `{"expiretm":"1791590400","ordertype":"limit","pair":"XBTUSD","price":"90000","timeinforce":"GTD","type":"sell","volume":"0.5"}`,
			reply:   spotPlaced("OR0CEN-TWHE5-ZJOOFV", "sell 0.50000000 XBTUSD @ limit 90000.0"),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Sell, TimeInForce: order.GoodTillTime, Price: 90000, Amount: 0.5, Status: order.New, OrderID: "OR0CEN-TWHE5-ZJOOFV"},
		},
		{
			name:    "spot market order",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Amount: 0.25},
			path:    "/0/private/AddOrder",
			request: `{"ordertype":"market","pair":"XBTUSD","type":"sell","volume":"0.25"}`,
			reply:   spotPlaced("OZD33P-8CG44-9E2SMV", "sell 0.25000000 XBTUSD @ market"),
			exp:     &order.SubmitResponse{Type: order.Market, Side: order.Sell, Amount: 0.25, Status: order.Filled, OrderID: "OZD33P-8CG44-9E2SMV"},
		},
		{
			name:    "spot market buy sized in the quote currency",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Buy, QuoteAmount: 100},
			path:    "/0/private/AddOrder",
			request: `{"oflags":"viqc","ordertype":"market","pair":"XBTUSD","type":"buy","volume":"100"}`,
			reply:   spotPlaced("OFCPM1-37ISG-FU08H6", "buy 100.00000000 XBTUSD @ market"),
			exp:     &order.SubmitResponse{Type: order.Market, Side: order.Buy, QuoteAmount: 100, Status: order.Filled, OrderID: "OFCPM1-37ISG-FU08H6"},
		},
		{
			// Add Order takes a quote amount for market buys alone
			name:    "spot market sell carrying a quote amount",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Amount: 0.25, QuoteAmount: 20000},
			path:    "/0/private/AddOrder",
			request: `{"ordertype":"market","pair":"XBTUSD","type":"sell","volume":"0.25"}`,
			reply:   spotPlaced("OBHC5N-U3V9W-KDTAAS", "sell 0.25000000 XBTUSD @ market"),
			exp:     &order.SubmitResponse{Type: order.Market, Side: order.Sell, Amount: 0.25, QuoteAmount: 20000, Status: order.Filled, OrderID: "OBHC5N-U3V9W-KDTAAS"},
		},
		{
			name:    "futures limit order",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Limit, Side: order.Buy, Price: 81500, Amount: 0.0005},
			path:    "/derivatives/api/v3/sendorder",
			request: "limitPrice=81500&orderType=lmt&side=buy&size=0.0005&symbol=PF_XBTUSD",
			reply:   futuresPlaced("1c4bb72a-7afe-4b9b-9b9b-50e772c06e89", "lmt", "buy", 0.0005, 81500),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, Price: 81500, Amount: 0.0005, Status: order.New, OrderID: "1c4bb72a-7afe-4b9b-9b9b-50e772c06e89"},
		},
		{
			name:    "futures post only order",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Limit, Side: order.Sell, Price: 82500, Amount: 0.0005, TimeInForce: order.PostOnly},
			path:    "/derivatives/api/v3/sendorder",
			request: "limitPrice=82500&orderType=post&side=sell&size=0.0005&symbol=PF_XBTUSD",
			reply:   futuresPlaced("64e79f53-8422-4bbc-8a65-2f6a879ccff1", "post", "sell", 0.0005, 82500),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Sell, TimeInForce: order.PostOnly, Price: 82500, Amount: 0.0005, Status: order.New, OrderID: "64e79f53-8422-4bbc-8a65-2f6a879ccff1"},
		},
		{
			name:    "futures immediate or cancel order",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Limit, Side: order.Buy, Price: 81600, Amount: 0.0005, TimeInForce: order.ImmediateOrCancel},
			path:    "/derivatives/api/v3/sendorder",
			request: "limitPrice=81600&orderType=ioc&side=buy&size=0.0005&symbol=PF_XBTUSD",
			reply:   futuresExecuted("1cd7f466-fba0-4fbc-84e6-6c72fe725b6f", "", "ioc", "buy", 0.0005, 81600, 81588, false),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, TimeInForce: order.ImmediateOrCancel, Price: 81600, Amount: 0.0005, Status: order.New, OrderID: "1cd7f466-fba0-4fbc-84e6-6c72fe725b6f"},
		},
		{
			name:    "futures fill or kill order",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Limit, Side: order.Buy, Price: 81600, Amount: 0.0005, TimeInForce: order.FillOrKill},
			path:    "/derivatives/api/v3/sendorder",
			request: "limitPrice=81600&orderType=fok&side=buy&size=0.0005&symbol=PF_XBTUSD",
			reply:   futuresExecuted("25f378e1-7598-40f4-a5e0-0ac669661081", "", "fok", "buy", 0.0005, 81600, 81591, false),
			exp:     &order.SubmitResponse{Type: order.Limit, Side: order.Buy, TimeInForce: order.FillOrKill, Price: 81600, Amount: 0.0005, Status: order.New, OrderID: "25f378e1-7598-40f4-a5e0-0ac669661081"},
		},
		{
			name:    "futures reduce only market order with a client order ID",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Market, Side: order.Sell, Amount: 0.001, ReduceOnly: true, ClientOrderID: "gct-market-1"},
			path:    "/derivatives/api/v3/sendorder",
			request: "cliOrdId=gct-market-1&orderType=mkt&reduceOnly=true&side=sell&size=0.001&symbol=PF_XBTUSD",
			reply:   futuresExecuted("4a762a34-dffa-48db-b7f1-4cce43a94fa1", "gct-market-1", "ioc", "sell", 0.001, 80784, 81590, true),
			exp:     &order.SubmitResponse{Type: order.Market, Side: order.Sell, ReduceOnly: true, Amount: 0.001, ClientOrderID: "gct-market-1", Status: order.Filled, OrderID: "4a762a34-dffa-48db-b7f1-4cce43a94fa1"},
		},
		{
			name:    "futures post only order that would execute",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Limit, Side: order.Buy, Price: 81900, Amount: 0.0005, TimeInForce: order.PostOnly},
			path:    "/derivatives/api/v3/sendorder",
			request: "limitPrice=81900&orderType=post&side=buy&size=0.0005&symbol=PF_XBTUSD",
			reply: `{"result":"success","sendStatus":{"orderEvents":[{"order":{"algoId":null,"cliOrdId":null,"filled":0,"lastUpdateTimestamp":"2026-10-09T08:18:10.005Z",` +
				`"limitPrice":81900,"orderId":"c5d6e7f8-a9b0-4c3d-8e4f-5a6b7c8d9e0f","quantity":0.0005,"reduceOnly":false,"side":"buy","symbol":"PF_XBTUSD",` +
				`"timestamp":"2026-10-09T08:18:10.004Z","type":"post"},"reason":"POST_WOULD_EXECUTE","type":"REJECT","uid":"c5d6e7f8-a9b0-4c3d-8e4f-5a6b7c8d9e0f"}],` +
				`"order_id":"c5d6e7f8-a9b0-4c3d-8e4f-5a6b7c8d9e0f","receivedTime":"2026-10-09T08:18:10.004Z","status":"postWouldExecute"},"serverTime":"2026-10-09T08:18:10.006Z"}`,
			err: order.ErrPlaceFailed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newOrdersRequestCheckExchange(t, tc.path, tc.request, tc.reply)
			tc.submit.Exchange, tc.submit.Pair = ex.Name, spotTestPair
			if tc.submit.AssetType == asset.Futures {
				tc.submit.Pair = futuresTestPair
			}
			submitted := time.Now()
			resp, err := ex.SubmitOrder(t.Context(), tc.submit)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "SubmitOrder should return Kraken's refusal")
				return
			}
			require.NoError(t, err, "SubmitOrder must not error")
			ordersClearSubmitTimes(t, resp, submitted)
			tc.exp.Exchange, tc.exp.Pair, tc.exp.AssetType = ex.Name, tc.submit.Pair, tc.submit.AssetType
			assert.Equal(t, tc.exp, resp, "SubmitOrder should place the order without a trigger reference")
		})
	}
}

func TestSubmitOrderSpotTrailingStopTrackingMode(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Failf(t, "no order should be placed", "%s", r.URL.Path)
		_, _ = w.Write([]byte(`{"error":["EGeneral:Invalid arguments"]}`))
	})
	_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.TrailingStop, Side: order.Sell, Amount: 0.3, TrackingValue: 2.5})
	assert.ErrorIs(t, err, order.ErrUnknownTrackingMode, "SubmitOrder should reject a spot trailing stop without a tracking mode over REST")
	ws := newOrdersWsTestExchange(t)
	_, err = ws.SubmitOrder(t.Context(), &order.Submit{Exchange: ws.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.TrailingStop, Side: order.Sell, Amount: 0.3, TrackingValue: 2.5})
	assert.ErrorIs(t, err, order.ErrUnknownTrackingMode, "SubmitOrder should reject a spot trailing stop without a tracking mode over the private websocket")
}

func TestSubmitOrderPositionSides(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		submit  *order.Submit
		path    string
		request string
		reply   string
		orderID string
	}{
		{
			name:    "spot bid over REST",
			submit:  &order.Submit{AssetType: asset.Spot, Type: order.Stop, Side: order.Bid, TriggerPrice: 90000, Amount: 0.1},
			path:    "/0/private/AddOrder",
			request: `{"ordertype":"stop-loss","pair":"XBTUSD","price":"90000","trigger":"last","type":"buy","volume":"0.1"}`,
			reply:   `{"error":[],"result":{"descr":{"order":"buy 0.10000000 XBTUSD @ stop loss 90000.0"},"txid":["OSIQRP-BJZQK-3R5CPS"]}}`,
			orderID: "OSIQRP-BJZQK-3R5CPS",
		},
		{
			name:    "futures long",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Stop, Side: order.Long, TriggerPrice: 84000, Amount: 0.001},
			path:    "/derivatives/api/v3/sendorder",
			request: "orderType=stp&side=buy&size=0.001&stopPrice=84000&symbol=PF_XBTUSD&triggerSignal=last",
			reply: `{"result":"success","sendStatus":{"orderEvents":[{"orderTrigger":{"clientId":null,"lastUpdateTimestamp":"2026-10-09T08:35:02.441Z","limitPrice":null,` +
				`"quantity":0.001,"reduceOnly":false,"side":"buy","startTime":null,"symbol":"PF_XBTUSD","timestamp":"2026-10-09T08:35:02.441Z","triggerPrice":84000,` +
				`"triggerSide":"trigger_above","triggerSignal":"last_price","type":"lmt","uid":"7e2b9c41-5d8a-4f36-b1e7-3a9c0d6f2b58"},"type":"PLACE"}],` +
				`"order_id":"7e2b9c41-5d8a-4f36-b1e7-3a9c0d6f2b58","receivedTime":"2026-10-09T08:35:02.441Z","status":"placed"},"serverTime":"2026-10-09T08:35:02.444Z"}`,
			orderID: "7e2b9c41-5d8a-4f36-b1e7-3a9c0d6f2b58",
		},
		{
			name:    "futures short",
			submit:  &order.Submit{AssetType: asset.Futures, Type: order.Stop, Side: order.Short, TriggerPrice: 79000, Amount: 0.001},
			path:    "/derivatives/api/v3/sendorder",
			request: "orderType=stp&side=sell&size=0.001&stopPrice=79000&symbol=PF_XBTUSD&triggerSignal=last",
			reply: `{"result":"success","sendStatus":{"orderEvents":[{"orderTrigger":{"clientId":null,"lastUpdateTimestamp":"2026-10-09T08:35:14.902Z","limitPrice":null,` +
				`"quantity":0.001,"reduceOnly":false,"side":"sell","startTime":null,"symbol":"PF_XBTUSD","timestamp":"2026-10-09T08:35:14.902Z","triggerPrice":79000,` +
				`"triggerSide":"trigger_below","triggerSignal":"last_price","type":"lmt","uid":"9f4d2a7e-0b6c-4e15-a8d3-5c1e7b9f4a26"},"type":"PLACE"}],` +
				`"order_id":"9f4d2a7e-0b6c-4e15-a8d3-5c1e7b9f4a26","receivedTime":"2026-10-09T08:35:14.902Z","status":"placed"},"serverTime":"2026-10-09T08:35:14.905Z"}`,
			orderID: "9f4d2a7e-0b6c-4e15-a8d3-5c1e7b9f4a26",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newOrdersRequestCheckExchange(t, tc.path, tc.request, tc.reply)
			tc.submit.Exchange, tc.submit.Pair = ex.Name, spotTestPair
			if tc.submit.AssetType == asset.Futures {
				tc.submit.Pair = futuresTestPair
			}
			resp, err := ex.SubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "SubmitOrder must not error")
			assert.Equal(t, tc.orderID, resp.OrderID, "SubmitOrder should place the order")
		})
	}
	t.Run("spot ask over the private websocket", func(t *testing.T) {
		t.Parallel()
		ex := newOrdersWsTestExchange(t, ordersWsRoundTrip{
			request: ordersWsRequest("add_order", `"order_type":"limit","side":"sell","order_qty":0.5,"limit_price":90000,"symbol":"BTC/USD"`),
			replies: []string{ordersWsReply("add_order", `{"order_id":"OOSYBA-N7T1K-2E5PUZ"}`)},
		})
		resp, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spotTestPair, AssetType: asset.Spot, Type: order.Limit, Side: order.Ask, Price: 90000, Amount: 0.5})
		require.NoError(t, err, "SubmitOrder must not error")
		assert.Equal(t, "OOSYBA-N7T1K-2E5PUZ", resp.OrderID, "SubmitOrder should place the order")
	})
}

func TestSubmitOrderFuturesUnsupportedTimeInForce(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Failf(t, "no futures order should be placed", "%s", r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"result":"error","error":"invalidArgument","serverTime":"2026-10-09T08:42:00.001Z"}`))
	})
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
	}{
		{order.Limit, order.GoodTillTime},
		{order.Limit, order.GoodTillDay},
		{order.Limit, order.GoodTillCrossing},
		{order.Market, order.PostOnly},
		{order.Market, order.FillOrKill},
		{order.Stop, order.ImmediateOrCancel},
		{order.TakeProfit, order.PostOnly},
		{order.TrailingStop, order.FillOrKill},
	} {
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{
			Exchange:      ex.Name,
			Pair:          futuresTestPair,
			AssetType:     asset.Futures,
			Type:          tc.orderType,
			Side:          order.Buy,
			Price:         81500,
			TriggerPrice:  84000,
			Amount:        0.001,
			TrackingMode:  order.Percentage,
			TrackingValue: 1,
			TimeInForce:   tc.tif,
			EndTime:       ordersTestExpiry,
		})
		assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "SubmitOrder should reject a futures %s order good for %s", tc.orderType, tc.tif)
	}
}

func TestModifyOrderFuturesSize(t *testing.T) {
	t.Parallel()
	ex := newOrdersRequestCheckExchange(t, "/derivatives/api/v3/editorder",
		"limitPrice=81520&orderId=4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b&qtyMode=ABSOLUTE&size=0.0006",
		`{"editStatus":{"cliOrdId":"gct-limit-1","orderEvents":[{"new":{"algoId":"gct-algo-1","cliOrdId":"gct-limit-1","filled":0.0002,`+
			`"lastUpdateTimestamp":"2026-10-09T08:19:58.730Z","limitPrice":81520,"orderId":"4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b","quantity":0.0006,"reduceOnly":true,`+
			`"side":"buy","symbol":"PF_XBTUSD","timestamp":"2026-10-09T08:15:30.412Z","type":"lmt"},"old":{"algoId":"gct-algo-1","cliOrdId":"gct-limit-1","filled":0.0002,`+
			`"lastUpdateTimestamp":"2026-10-09T08:15:30.413Z","limitPrice":81500,"orderId":"4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b","quantity":0.0004,"reduceOnly":true,`+
			`"side":"buy","symbol":"PF_XBTUSD","timestamp":"2026-10-09T08:15:30.412Z","type":"lmt"},"reducedQuantity":null,"type":"EDIT"}],`+
			`"orderId":"4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b","receivedTime":"2026-10-09T08:19:58.729Z","status":"edited"},"result":"success","serverTime":"2026-10-09T08:19:58.735Z"}`)
	resp, err := ex.ModifyOrder(t.Context(), &order.Modify{
		Exchange:      ex.Name,
		OrderID:       "4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b",
		ClientOrderID: "gct-limit-1",
		Type:          order.Limit,
		Side:          order.Buy,
		AssetType:     asset.Futures,
		Pair:          futuresTestPair,
		Price:         81520,
		Amount:        0.0006,
	})
	require.NoError(t, err, "ModifyOrder must not error")
	exp := &order.ModifyResponse{
		Exchange:      ex.Name,
		OrderID:       "4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b",
		ClientOrderID: "gct-limit-1",
		Pair:          futuresTestPair,
		Type:          order.Limit,
		Side:          order.Buy,
		AssetType:     asset.Futures,
		Price:         81520,
		Amount:        0.0006,
	}
	assert.Equal(t, exp, resp, "ModifyOrder should set the futures order's total size, as a spot amend does")
}

func TestCancelBatchOrdersPartialSpotBatch(t *testing.T) {
	t.Parallel()
	const stopLoss, iceberg, unknown = "OQCLML-BW3P3-BUCMWZ", "OHYO67-6LP66-HMQ437", "OZJW0T-MIQ3C-5NATAU"
	var mu sync.Mutex
	// open holds whether each order Kraken knows is still open; Kraken does not know a cancelled order to cancel it again
	open := map[string]bool{stopLoss: true, iceberg: true}
	// orderInfo holds each order as Query Orders Info returns it, with STATUS standing for its status
	orderInfo := map[string]string{
		stopLoss: `{"cl_ord_id":null,"cost":"11254.68750","descr":{"aclass":"forex","close":"","leverage":"5:1","order":"buy 1.25000000 XBTUSD @ stop loss 30010.0 -> limit 30020.0 with 5:1 leverage",` +
			`"ordertype":"stop-loss-limit","pair":"XBTUSD","price":"30010.0","price2":"30020.0","type":"buy"},"expiretm":1780668660,"fee":"29.26219","limitprice":"30020.0",` +
			`"misc":"stopped,partial","oflags":"fciq","opentm":1780582233.729133,"price":"30012.5","reduce_only":true,"refid":null,"starttm":1780582260,"status":"STATUS",` +
			`"stopprice":"30010.0","time_in_force":"gtd","userref":45326,"vol":"1.25000000","vol_exec":"0.37500000"}`,
		iceberg: `{"cl_ord_id":"6d1b345e-2821-40e2-ad83-4ecb18a06876","cost":"5301.40","descr":{"aclass":"forex","close":"","leverage":"none","order":"sell 10.00000000 ETHUSD @ limit 2650.50",` +
			`"ordertype":"iceberg","pair":"ETHUSD","price":"2650.50","price2":"0","type":"sell"},"displayvol":"1.00000000","expiretm":0,"fee":"8.48224","limitprice":"0.00000",` +
			`"misc":"partial","oflags":"post,fcib","opentm":1780582311.512449,"price":"2650.70","refid":null,"starttm":0,"status":"STATUS","stopprice":"0.00000","time_in_force":"gtc",` +
			`"userref":null,"vol":"10.00000000","vol_exec":"2.00000000"}`,
	}
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Orders        []string `json:"orders"`
			TransactionID string   `json:"txid"`
		}
		assert.NoError(t, json.Unmarshal([]byte(tradingRequestBody(t, r)), &req), "Unmarshal should not error")
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/0/private/CancelOrderBatch":
			var count int
			for _, id := range req.Orders {
				if open[id] {
					open[id] = false
					count++
				}
			}
			_, _ = w.Write([]byte(`{"error":[],"result":{"count":` + strconv.Itoa(count) + `}}`))
		case "/0/private/CancelOrder":
			if !open[req.TransactionID] {
				_, _ = w.Write([]byte(`{"error":["EOrder:Unknown order"]}`))
				return
			}
			open[req.TransactionID] = false
			_, _ = w.Write([]byte(`{"error":[],"result":{"count":1,"pending":false}}`))
		case "/0/private/QueryOrders":
			result := make([]string, 0, len(open))
			for id := range strings.SplitSeq(req.TransactionID, ",") {
				isOpen, known := open[id]
				if !known {
					continue
				}
				status := "canceled"
				if isOpen {
					status = "open"
				}
				result = append(result, `"`+id+`":`+strings.Replace(orderInfo[id], "STATUS", status, 1))
			}
			_, _ = w.Write([]byte(`{"error":[],"result":{` + strings.Join(result, ",") + `}}`))
		default:
			assert.Failf(t, "only Cancel Order Batch, Cancel Order and Query Orders Info should be called", "%s", r.URL.Path)
		}
	})
	resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
		{Exchange: ex.Name, AssetType: asset.Spot, Pair: spotTestPair, OrderID: stopLoss},
		{Exchange: ex.Name, AssetType: asset.Spot, Pair: ordersETHUSDPair, OrderID: iceberg},
		{Exchange: ex.Name, AssetType: asset.Spot, Pair: spotTestPair, OrderID: unknown},
	})
	require.NoError(t, err, "CancelBatchOrders must not error")
	assert.NotEqual(t, order.Cancelled.String(), resp.Status[unknown], "CancelBatchOrders should not report an order Kraken does not know as cancelled")
	delete(resp.Status, unknown)
	assert.Equal(t, map[string]string{stopLoss: order.Cancelled.String(), iceberg: order.Cancelled.String()}, resp.Status, "CancelBatchOrders should report the orders the batch cancelled as cancelled")
}

func TestGetActiveOrdersUnlistedPairs(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/0/private/OpenOrders":
			_, _ = w.Write([]byte(ordersUnlistedOpenOrders))
		case "/derivatives/api/v3/openorders":
			_, _ = w.Write([]byte(`{"openOrders":[{"algoId":"gct-algo-1","cliOrdId":"gct-batch-1","filledSize":0.0003,"lastUpdateTime":"2026-10-09T08:26:14.982Z","limitPrice":81450,` +
				`"orderType":"lmt","order_id":"e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b","receivedTime":"2026-10-09T08:24:59.120Z","reduceOnly":true,"side":"buy",` +
				`"status":"partiallyFilled","symbol":"PF_XBTUSD","unfilledSize":0.0004},{"filledSize":0,"lastUpdateTime":"2026-10-09T08:43:20.611Z","limitPrice":0.0821,` +
				`"orderType":"lmt","order_id":"2398d7e0-ee86-4911-be2c-ef3a4a874153","receivedTime":"2026-10-09T08:43:20.611Z","reduceOnly":false,"side":"buy",` +
				`"status":"untouched","symbol":"PF_ZORAUSD","unfilledSize":2500}],"result":"success","serverTime":"2026-10-09T08:44:00.512Z"}`))
		}
	})
	got, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for spot")
	assert.Equal(t, order.FilteredOrders{ordersStopLossLimitDetail(ex.Name)}, got, "GetActiveOrders should return the spot orders on pairs it can name")
	got, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for futures")
	assert.Equal(t, order.FilteredOrders{ordersFuturesLimitDetail(ex.Name)}, got, "GetActiveOrders should return the futures orders on contracts it can name")
}

// ordersUnlistedOpenOrders is Get Open Orders' reply holding an order on WAVESUSD, which the available pairs lack, and
// the recorded open stop loss limit order
const ordersUnlistedOpenOrders = `{"error":[],"result":{"open":{` +
	`"O0IJNX-TWAD7-W21QA7":{"cl_ord_id":null,"cost":"0.00000","descr":{"aclass":"forex","close":"","leverage":"none","order":"sell 500.00000000 WAVESUSD @ limit 1.2250",` +
	`"ordertype":"limit","pair":"WAVESUSD","price":"1.2250","price2":"0","type":"sell"},"expiretm":0,"fee":"0.00000","limitprice":"0.00000","misc":"","oflags":"fcib",` +
	`"opentm":1791480200.3381,"price":"0.00000","refid":null,"starttm":0,"status":"open","stopprice":"0.00000","time_in_force":"gtc","userref":null,` +
	`"vol":"500.00000000","vol_exec":"0.00000000"},` +
	`"OQCLML-BW3P3-BUCMWZ":{"cl_ord_id":null,"cost":"11254.68750","descr":{"aclass":"forex","close":"close position @ stop loss 28000.0 -> limit 27900.0","leverage":"5:1",` +
	`"order":"buy 1.25000000 XBTUSD @ stop loss 30010.0 -> limit 30020.0 with 5:1 leverage","ordertype":"stop-loss-limit","pair":"XBTUSD","price":"30010.0",` +
	`"price2":"30020.0","type":"buy"},"expiretm":1780668660,"ext_ord_id":"EXT-8812-AB","fee":"29.26219","limitprice":"30020.0","link_id":"OK3SN7-GFU3V-6E3ZBM",` +
	`"margin":true,"misc":"stopped,partial,amended","oflags":"fciq","opentm":1780582233.729133,"price":"30012.5","reduce_only":true,"refid":"OB5VMB-B4U2U-DK2WRW",` +
	`"sender_sub_id":"desk-7","starttm":1780582260,"status":"open","stopprice":"30010.0","time_in_force":"gtd","trigger":"index","userref":45326,` +
	`"vol":"1.25000000","vol_exec":"0.37500000"}}}}`

func TestCancelAllOrdersUnlistedPairs(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/0/private/OpenOrders":
			_, _ = w.Write([]byte(ordersUnlistedOpenOrders))
		case "/0/private/CancelOrderBatch":
			var req struct {
				Orders []string `json:"orders"`
			}
			assert.NoError(t, json.Unmarshal([]byte(tradingRequestBody(t, r)), &req), "Unmarshal should not error")
			assert.ElementsMatch(t, []string{"O0IJNX-TWAD7-W21QA7", "OQCLML-BW3P3-BUCMWZ"}, req.Orders, "Cancel Order Batch should cancel every open order")
			_, _ = w.Write([]byte(`{"error":[],"result":{"count":2}}`))
		default:
			assert.Failf(t, "only Get Open Orders and Cancel Order Batch should be called", "%s", r.URL.Path)
		}
	})
	resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{Exchange: ex.Name, AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error")
	exp := order.CancelAllResponse{Status: map[string]string{"O0IJNX-TWAD7-W21QA7": order.Cancelled.String(), "OQCLML-BW3P3-BUCMWZ": order.Cancelled.String()}}
	assert.Equal(t, exp, resp, "CancelAllOrders should cancel every open spot order")
}

func TestGetOrderHistoryFuturesCancelledOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the cancelled orders are recorded responses")
	}
	got, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Futures,
		Side:      order.AnySide,
		Type:      order.AnyType,
		StartTime: time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err, "GetOrderHistory must not error")
	exp := order.FilteredOrders{
		{
			Exchange:        e.Name,
			AssetType:       asset.Futures,
			OrderID:         "1171b7d3-8ba0-449d-af73-f2ee5d34f5e6",
			ClientOrderID:   "gct-close-1",
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Type:            order.Limit,
			TimeInForce:     order.PostOnly,
			Status:          order.Cancelled,
			Price:           82018,
			Amount:          0.0575,
			RemainingAmount: 0.0575,
			Date:            time.UnixMilli(1791507801988),
			LastUpdated:     time.UnixMilli(1791507802196),
		},
		{
			Exchange:        e.Name,
			AssetType:       asset.Futures,
			OrderID:         "72e04257-ca24-4034-b4f8-de017c57e33b",
			ClientOrderID:   "gct-close-2",
			Pair:            futuresTestPair,
			Side:            order.Buy,
			Type:            order.Limit,
			Status:          order.PartiallyFilledCancelled,
			Price:           81400,
			Amount:          0.5,
			ExecutedAmount:  0.25,
			RemainingAmount: 0.25,
			Date:            time.UnixMilli(1791508000000),
			LastUpdated:     time.UnixMilli(1791508100250),
		},
	}
	assert.Equal(t, exp, got, "GetOrderHistory should report a cancelled order as last updated when it was cancelled")
}

func TestGetOrderHistoryFuturesUndecodedValues(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the undecoded orders are recorded responses")
	}
	got, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.Futures,
		Side:      order.AnySide,
		Type:      order.AnyType,
		StartTime: time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 10, 9, 3, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err, "GetOrderHistory must not error")
	exp := order.FilteredOrders{
		{
			Exchange:        e.Name,
			AssetType:       asset.Futures,
			OrderID:         "eadab10e-d9f0-4af9-ae2d-f85ea491ab7a",
			ClientOrderID:   "gct-undecoded-1",
			Pair:            futuresTestPair,
			Side:            order.Buy,
			Status:          order.Open,
			Price:           81620,
			Amount:          0.2,
			RemainingAmount: 0.2,
			Date:            time.UnixMilli(1791511300000),
			LastUpdated:     time.UnixMilli(1791511300000),
		},
		{
			Exchange:        e.Name,
			AssetType:       asset.Futures,
			OrderID:         "a62ee6b7-ad14-47d6-ab3a-f9151713f756",
			ClientOrderID:   "gct-undecoded-2",
			Pair:            futuresTestPair,
			Type:            order.Limit,
			Status:          order.Open,
			Price:           81700,
			Amount:          0.3,
			RemainingAmount: 0.3,
			Date:            time.UnixMilli(1791511400000),
			LastUpdated:     time.UnixMilli(1791511400000),
		},
		{
			Exchange:        e.Name,
			AssetType:       asset.Futures,
			OrderID:         "4a7d2506-5dde-45a6-acb9-3c8080bd9ca9",
			ClientOrderID:   "gct-plain-1",
			Pair:            futuresTestPair,
			Side:            order.Sell,
			Type:            order.Limit,
			Status:          order.Open,
			Price:           82200,
			Amount:          0.1,
			RemainingAmount: 0.1,
			Date:            time.UnixMilli(1791511500000),
			LastUpdated:     time.UnixMilli(1791511500000),
		},
	}
	assert.Equal(t, exp, got, "GetOrderHistory should keep orders whose order type or direction Kraken could not decode, as unknown")
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, "Kraken", ex.Name, "SetDefaults should name the exchange")
	assert.True(t, ex.Enabled, "SetDefaults should enable the exchange")
	assert.True(t, ex.API.CredentialsValidator.RequiresBase64DecodeSecret, "SetDefaults should require a base64 secret")
	for a, delimiter := range map[asset.Item]string{asset.Spot: "", asset.Futures: currency.UnderscoreDelimiter} {
		format, err := ex.GetPairFormat(a, true)
		require.NoErrorf(t, err, "GetPairFormat must not error for %s", a)
		assert.Equalf(t, currency.PairFormat{Uppercase: true, Delimiter: delimiter}, format, "SetDefaults should set the %s request format", a)
	}
	for u, exp := range map[exchange.URL]string{
		exchange.RestSpot:                   spotAPIURL,
		exchange.RestFutures:                futuresAPIURL,
		exchange.RestFuturesSupplementary:   futuresSupplementaryAPIURL,
		exchange.WebsocketSpot:              wsPublicURL,
		exchange.WebsocketSpotSupplementary: wsPrivateURL,
		exchange.WebsocketPrivate:           wsLevel3URL,
		exchange.WebsocketFutures:           wsFuturesURL,
	} {
		got, err := ex.API.Endpoints.GetURL(u)
		require.NoErrorf(t, err, "GetURL must not error for %s", u)
		assert.Equalf(t, exp, got, "SetDefaults should set the %s endpoint", u)
	}
	assert.Equal(t, defaultSubscriptions.Clone(), ex.Features.Subscriptions, "SetDefaults should set the default subscriptions")
	assert.NotNil(t, ex.Websocket, "SetDefaults should create the websocket manager")
	assert.NotNil(t, ex.Requester, "SetDefaults should create the requester")
	assert.Equal(t, exchange.DefaultWebsocketResponseMaxLimit, ex.WebsocketResponseMaxLimit, "SetDefaults should set the websocket response limit")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Error(t, ex.Setup(nil), "Setup should reject a nil config")
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.Enabled = false
	require.NoError(t, ex.Setup(cfg), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should disable an exchange its config disables")

	ex = new(Exchange)
	ex.SetDefaults()
	cfg, err = ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	require.NoError(t, ex.Setup(cfg), "Setup must not error")
	assert.True(t, ex.IsEnabled(), "Setup should keep an enabled exchange enabled")
	assert.True(t, ex.Websocket.IsInitialised(), "Setup should set up the websocket")
	assert.Error(t, ex.Setup(cfg), "Setup should reject setting up the websocket twice")
}

func TestCancelSpotOrder(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "ReadAll should not error")
		reply := `{"error":["EOrder:Unknown order"]}`
		if strings.Contains(string(body), "OQCLML-BW3P3-BUCMWZ") {
			reply = `{"error":[],"result":{"count":1}}`
		}
		_, err = w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	})
	assert.Equal(t, order.Cancelled.String(), ex.cancelSpotOrder(t.Context(), "OQCLML-BW3P3-BUCMWZ"), "cancelSpotOrder should report a cancelled order")
	assert.Contains(t, ex.cancelSpotOrder(t.Context(), "OUNKNO-WN000-000000"), "EOrder:Unknown order", "cancelSpotOrder should report the error Kraken replies with")
}

func TestFuturesCandles(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(1791504000, 0), time.Unix(1791511200, 0)
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/charts/v1/trade/PF_XBTUSD/1h"), "futuresCandles should request the market's trade candles")
		assert.Equal(t, url.Values{"from": {"1791504000"}, "to": {"1791511199"}}, r.URL.Query(), "futuresCandles should end the range a second before end")
		_, err := w.Write([]byte(`{"candles":[{"time":1791504000000,"open":"82000.5","high":"82100","low":"81900","close":"82050","volume":"12.5"}],"more_candles":false}`))
		assert.NoError(t, err, "Write should not error")
	})
	candles, err := ex.futuresCandles(t.Context(), futuresTestPair, kline.OneHour, start, end)
	require.NoError(t, err, "futuresCandles must not error")
	assert.Equal(t, []kline.Candle{{Time: start, Open: 82000.5, High: 82100, Low: 81900, Close: 82050, Volume: 12.5}}, candles, "futuresCandles should convert each candle")
	_, err = ex.futuresCandles(t.Context(), currency.EMPTYPAIR, kline.OneHour, start, end)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "futuresCandles should return the request's error")
}

func TestIsPerpetual(t *testing.T) {
	t.Parallel()
	assert.True(t, e.isPerpetual(futuresTestPair), "isPerpetual should report a linear perpetual")
	assert.True(t, e.isPerpetual(piXBTUSDPair), "isPerpetual should report an inverse perpetual")
	assert.False(t, e.isPerpetual(currency.NewPairWithDelimiter("FI", "XBTUSD_261225", currency.UnderscoreDelimiter)), "isPerpetual should not report a fixed maturity future")
}

func TestSpotTrades(t *testing.T) {
	t.Parallel()
	trades := &RecentTradesResponse{Trades: map[string][]RecentTrade{"XXBTZUSD": {
		{Price: 81720.1, Volume: 0.00029992, Time: types.Time(time.Unix(1791504601, 92063400)), Side: "b", OrderType: "l", TradeID: 110760685},
		{Price: 81723.4, Volume: 0.00005006, Time: types.Time(time.Unix(1791504605, 893529400)), Side: "s", OrderType: "m", TradeID: 110760687},
	}}}
	exp := []trade.Data{
		{TID: "110760685", Exchange: e.Name, CurrencyPair: spotTestPair, AssetType: asset.Spot, Side: order.Buy, Price: 81720.1, Amount: 0.00029992, Timestamp: time.Unix(1791504601, 92063400)},
		{TID: "110760687", Exchange: e.Name, CurrencyPair: spotTestPair, AssetType: asset.Spot, Side: order.Sell, Price: 81723.4, Amount: 0.00005006, Timestamp: time.Unix(1791504605, 893529400)},
	}
	assert.Equal(t, exp, e.spotTrades(trades, spotTestPair), "spotTrades should convert each trade with its side")
	assert.Empty(t, e.spotTrades(&RecentTradesResponse{}, spotTestPair), "spotTrades should return nothing without trades")
}

func TestOrderSideString(t *testing.T) {
	t.Parallel()
	for side, exp := range map[order.Side]string{order.Buy: "buy", order.Bid: "buy", order.Long: "buy", order.Sell: "sell", order.Ask: "sell", order.Short: "sell"} {
		got, err := orderSideString(side)
		require.NoErrorf(t, err, "orderSideString must not error for %s", side)
		assert.Equalf(t, exp, got, "orderSideString should return the side Kraken takes for %s", side)
	}
	for _, side := range []order.Side{order.UnknownSide, order.AnySide} {
		_, err := orderSideString(side)
		assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "orderSideString should reject %s", side)
	}
}
