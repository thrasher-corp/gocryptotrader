package okx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// TestOptionInstrumentFamilies pins the family extraction behind the option
// open interest queries: every family of the tick bands table survives in
// first-seen order, familyless rows and duplicates do not, and the
// no-response sentinel is directly covered.
func TestOptionInstrumentFamilies(t *testing.T) {
	t.Parallel()

	tickBandsTable := func(rows ...string) string {
		return `{"code":"0","msg":"","data":[` + strings.Join(rows, ",") + `]}`
	}
	tickBandRow := func(instFamily string) string {
		return fmt.Sprintf(`{"instType":%q,"instFamily":%q,"tickBand":[{"minPx":"0","maxPx":"100","tickSz":"0.1"}]}`, instTypeOption, instFamily)
	}

	for _, tc := range []struct {
		name      string
		response  string
		cancelled bool
		families  []string
		err       error
		errText   string
	}{
		{
			// An underlying spans several families (BTC-USD spans BTC-USD
			// and BTC-USD_UM), so every family of the table must survive;
			// duplicates and familyless rows must not.
			name: "each family once, in first-seen order",
			response: tickBandsTable(
				tickBandRow("BTC-USD"),
				tickBandRow(""),
				tickBandRow("ETH-USD_UM"),
				tickBandRow("BTC-USD"),
				tickBandRow("SOL-USD_UM"),
			),
			families: []string{"BTC-USD", "ETH-USD_UM", "SOL-USD_UM"},
		},
		{
			name:     "single family",
			response: tickBandsTable(tickBandRow("BTC-USD")),
			families: []string{"BTC-USD"},
		},
		{
			name:      "cancelled context",
			response:  tickBandsTable(tickBandRow("BTC-USD")),
			cancelled: true,
			err:       context.Canceled,
		},
		{
			name:     "empty tick bands table",
			response: tickBandsTable(),
			err:      common.ErrNoResponse,
		},
		{
			// The tick bands response documents instFamily:"" rows for other
			// instrument types, so a table of familyless rows must not
			// masquerade as option coverage.
			name:     "table without any family",
			response: tickBandsTable(tickBandRow(""), tickBandRow("")),
			err:      common.ErrNoResponse,
		},
		{
			name:     "error response",
			response: `{"code":"1","msg":"mock: tick bands request failed","data":[]}`,
			errText:  "error code: `1`",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := new(Exchange)
			require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

			var mu sync.Mutex
			var gotPath string
			var gotQuery url.Values
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotPath = r.URL.Path
				gotQuery = r.URL.Query()
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.response))
			}))

			b := e.GetBase()
			b.SkipAuthCheck = true
			require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
			for k := range b.API.Endpoints.GetURLMap() {
				require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
			}

			ctx := t.Context()
			if tc.cancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			families, err := e.optionInstrumentFamilies(ctx)
			if tc.err != nil || tc.errText != "" {
				require.Error(t, err, "optionInstrumentFamilies must error")
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err, "optionInstrumentFamilies must wrap the expected sentinel")
				}
				if tc.errText != "" {
					require.ErrorContains(t, err, tc.errText, "optionInstrumentFamilies must surface the response error")
				}
				return
			}
			mu.Lock()
			path, query := gotPath, gotQuery
			mu.Unlock()
			require.NoError(t, err, "optionInstrumentFamilies must not error")
			assert.Equal(t, tc.families, families, "optionInstrumentFamilies should return the deduplicated families in first-seen order")
			assert.Equal(t, "/public/instrument-tick-bands", path, "optionInstrumentFamilies should query the documented tick bands endpoint")
			assert.Equal(t, instTypeOption, query.Get("instType"), "the tick bands query should be scoped to OPTION")
			assert.Empty(t, query.Get("instFamily"), "the tick bands query should not pre-filter a family")
		})
	}
}

const placeOrderArgs = `[{"side": "buy","instId": "BTC-USDT","tdMode": "cash","ordType": "market","sz": "100"},{"side": "buy","instId": "LTC-USDT","tdMode": "cash","ordType": "market","sz": "1"}]`

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.Options, asset.PerpetualSwap, asset.Futures, asset.Spot, asset.Spread} {
		result, err := e.FetchTradablePairs(contextGenerate(), a)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.GetAvailablePairs(a)
			require.NoError(t, err, "GetAvailablePairs must not error")
			require.NotEmpty(t, pairs, "GetAvailablePairs must return the listed instruments")
			for _, p := range pairs {
				l, err := e.GetOrderExecutionLimits(a, p)
				require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", p)
				assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
				assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
			}
		})
	}
	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported)
	})
	t.Run("maps a spread's tick, minimum and lot sizes", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/sprd/spreads", r.URL.Path, "the limits should come from the public spreads")
			assert.Equal(t, url.Values{"state": {instrumentStateLive}}, r.URL.Query(), "the request should ask for live spreads")
			writeOKXData(t, w, []map[string]string{{
				"sprdId": "BTC-USDT_BTC-USDT-SWAP", "sprdType": "linear", "state": instrumentStateLive, "baseCcy": "BTC", "szCcy": "BTC",
				"quoteCcy": "USDT", "tickSz": "0.1", "minSz": "0.002", "lotSz": "0.0001", "listTime": "1606468572000", "expTime": "", "uTime": "1794451805535",
			}})
		}))
		ex.Name = t.Name()
		require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spread), "UpdateOrderExecutionLimits must not error")
		got, err := ex.GetOrderExecutionLimits(asset.Spread, spreadPair)
		require.NoError(t, err, "GetOrderExecutionLimits must not error")
		assert.WithinDuration(t, time.Now(), got.UpdatedAt, time.Minute, "the limits should record when they were loaded")
		exp := limits.MinMaxLevel{
			UpdatedAt:               got.UpdatedAt,
			Key:                     key.NewExchangeAssetPair(ex.Name, asset.Spread, spreadPair),
			PriceStepIncrementSize:  0.1,
			MinimumBaseAmount:       0.002,
			AmountStepIncrementSize: 0.0001,
		}
		assert.Equal(t, exp, got, "the spread's tick size should step the price, its minimum and lot sizes the amount")
	})
	t.Run("a listing answered after a newer one leaves the newer listing's limits", func(t *testing.T) {
		t.Parallel()
		row := func(minimum string, code uint64) map[string]any {
			r, _ := instrumentRow(t, instTypeSpot, "BTC-USDT", "", code)
			r["state"], r["minSz"], r["lotSz"], r["tickSz"] = instrumentStateLive, minimum, "1", "0.1"
			return r
		}
		ex, entered, release := newOverlappingListingsExchange(t, []map[string]any{row("1", 31)}, []map[string]any{row("10", 72)})
		ex.Name = t.Name()
		older := make(chan error, 1)
		go func() { older <- ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot) }()
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the older listing must be requested")
		}
		require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")
		release()
		select {
		case err := <-older:
			require.NoError(t, err, "UpdateOrderExecutionLimits must not error for the older listing")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the older listing must return")
		}
		got, err := ex.GetOrderExecutionLimits(asset.Spot, mainPair)
		require.NoError(t, err, "GetOrderExecutionLimits must not error")
		exp := limits.MinMaxLevel{
			UpdatedAt:               got.UpdatedAt,
			Key:                     key.NewExchangeAssetPair(ex.Name, asset.Spot, mainPair),
			PriceStepIncrementSize:  0.1,
			MinimumBaseAmount:       10,
			AmountStepIncrementSize: 1,
		}
		assert.Equal(t, exp, got, "the limits should stay the newer listing's")
	})
}

func TestLoadInstrumentOrderExecutionLimits(t *testing.T) {
	exch := &Exchange{}
	exch.SetDefaults()
	exch.Name = t.Name()

	livePair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	inactivePair := currency.NewPairWithDelimiter("ETH", "USDT", "-")
	require.NoError(t, exch.loadInstrumentOrderExecutionLimits(asset.Futures, []InstrumentResponse{
		{
			InstrumentID:     livePair,
			State:            instrumentStateLive,
			TickSize:         types.Number(0.1),
			MinimumOrderSize: types.Number(1),
			LotSize:          types.Number(0.01),
		},
		{
			InstrumentID:     inactivePair,
			State:            "preopen",
			TickSize:         types.Number(0.01),
			MinimumOrderSize: types.Number(2),
		},
		{
			InstrumentID:     currency.EMPTYPAIR,
			State:            instrumentStateLive,
			TickSize:         types.Number(0.01),
			MinimumOrderSize: types.Number(2),
		},
	}), "loadInstrumentOrderExecutionLimits must load live instruments and skip inactive or empty instruments")

	loadedLimit, err := exch.GetOrderExecutionLimits(asset.Futures, livePair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for live instrument")
	assert.Equal(t, 0.1, loadedLimit.PriceStepIncrementSize, "PriceStepIncrementSize should be set from live instrument")
	assert.Equal(t, 1.0, loadedLimit.MinimumBaseAmount, "MinimumBaseAmount should be set from live instrument")
	assert.Equal(t, 0.01, loadedLimit.AmountStepIncrementSize, "AmountStepIncrementSize should be set from live instrument")

	_, err = exch.GetOrderExecutionLimits(asset.Futures, inactivePair)
	require.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "inactive instruments must not be loaded")

	require.ErrorIs(t, exch.loadInstrumentOrderExecutionLimits(asset.Futures, []InstrumentResponse{
		{InstrumentID: inactivePair, State: "preopen"},
		{InstrumentID: currency.EMPTYPAIR, State: instrumentStateLive},
	}), common.ErrInvalidResponse, "all filtered instruments must return invalid response")

	require.ErrorIs(t, exch.loadInstrumentOrderExecutionLimits(asset.Futures, nil),
		common.ErrNoResponse, "empty instrument slice must return no response")

	// One OKX contract is ctVal times ctMult: ctMult is 1 on swaps and futures
	// but 0.01 on a BTC-USD option, so one option contract is 0.01 BTC.
	require.NoError(t, exch.loadInstrumentOrderExecutionLimits(asset.Options, []InstrumentResponse{
		{InstrumentID: optionsPair, State: instrumentStateLive, ContractValue: types.Number(1), ContractMultiplier: types.Number(0.01)},
	}), "loadInstrumentOrderExecutionLimits must load a live option")
	optionLimit, err := exch.GetOrderExecutionLimits(asset.Options, optionsPair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for a live option")
	assert.Equal(t, 0.01, optionLimit.MultiplierDecimal, "MultiplierDecimal should be the size of one option contract")
}

func TestLoadStoredInstrumentOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.Name = t.Name()
	ex.instrumentsInfoMap = map[string][]InstrumentResponse{
		instTypeSwap: {{InstrumentID: perpetualSwapPair, State: instrumentStateLive, TickSize: types.Number(0.1), MinimumOrderSize: types.Number(10), LotSize: types.Number(1), ContractValue: types.Number(0.01), ContractMultiplier: types.Number(1)}},
		instTypeSpot: {{InstrumentID: mainPair, State: instrumentStateLive, TickSize: types.Number(0.5), MinimumOrderSize: types.Number(3), LotSize: types.Number(2)}},
	}
	require.ErrorIs(t, ex.loadStoredInstrumentOrderExecutionLimits(asset.Futures), common.ErrNoResponse, "an asset without a stored list must have no limits to load")
	requireWaitsForLock(t, &ex.instrumentsInfoMapLock, func() {
		assert.NoError(t, ex.loadStoredInstrumentOrderExecutionLimits(asset.PerpetualSwap), "loadStoredInstrumentOrderExecutionLimits should load the stored list's limits")
	})
	got, err := ex.GetOrderExecutionLimits(asset.PerpetualSwap, perpetualSwapPair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error")
	exp := limits.MinMaxLevel{
		UpdatedAt:               got.UpdatedAt,
		Key:                     key.NewExchangeAssetPair(ex.Name, asset.PerpetualSwap, perpetualSwapPair),
		PriceStepIncrementSize:  0.1,
		MinimumBaseAmount:       10,
		AmountStepIncrementSize: 1,
		MultiplierDecimal:       0.01,
	}
	assert.Equal(t, exp, got, "the stored swap list's limits should load")
	_, err = ex.GetOrderExecutionLimits(asset.Spot, mainPair)
	assert.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "another instrument type's list should not load")
}

func TestTickerVolumes(t *testing.T) {
	t.Parallel()
	tk := &TickerResponse{TradingVolume24HourInContract: 1000, TradingVolume24HourInCurrency: 10}
	for _, tc := range []struct {
		a           asset.Item
		base, quote float64
	}{
		{asset.Spot, 1000, 10},
		{asset.Margin, 1000, 10},
		{asset.PerpetualSwap, 10, 0},
		{asset.Futures, 10, 0},
		{asset.Options, 10, 0},
		{asset.Spread, 0, 0},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			base, quote := tickerVolumes(tk, tc.a)
			assert.Equalf(t, tc.base, base, "tickerVolumes should return the base volume for %s", tc.a)
			assert.Equalf(t, tc.quote, quote, "tickerVolumes should return the quote volume for %s", tc.a)
		})
	}
}

// newIsolatedExchange returns an exchange of its own, answered by the recorded fixtures in mock mode and by OKX
// otherwise, for tests that change state, such as the enabled pairs, which the package's e shares with every test
func newIsolatedExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	if mockTests {
		require.NoError(t, testexch.MockHTTPInstance(ex, "api/v5/"), "MockHTTPInstance must not error")
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	}
	return ex
}

// btcUSDSwapPair is the inverse BTC perpetual swap, which the recorded instrument lists carry
var btcUSDSwapPair = currency.NewPairWithDelimiter("BTC", "USD-SWAP", currency.DashDelimiter)

// mockFuturesContract is a dated futures contract the recorded instrument lists carry, used in mock mode where live
// runs take an enabled contract, since the configured contract has expired
var mockFuturesContract = currency.NewPairWithDelimiter("BTC", "USD_UM-261225", currency.DashDelimiter)

func TestUpdateTicker(t *testing.T) {
	t.Parallel()

	t.Run("rejects a malformed ticker reply", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, data string
		}{
			{"null", "null"},
			{"null row", "[null]"},
			{"empty row", "[{}]"},
			{"row without an instrument", `[{"instType":"SPOT","last":"60000","bidPx":"59999","askPx":"60001","ts":"1700000000000"}]`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/market/ticker", r.URL.Path, "the ticker should come from the instrument's ticker")
					_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				ex.Name = t.Name()
				var err error
				require.NotPanics(t, func() { _, err = ex.UpdateTicker(t.Context(), mainPair, asset.Spot) }, "UpdateTicker must not panic on a malformed reply")
				assert.ErrorIs(t, err, common.ErrInvalidResponse, "UpdateTicker should reject a success reply without a ticker")
			})
		}
	})

	e := newIsolatedExchange(t)
	// Renamed before UpdatePairsOnce, which caches by name, so the package's e still loads its own instruments
	e.Name = t.Name()
	testexch.UpdatePairsOnce(t, e)

	_, err := e.UpdateTicker(contextGenerate(), currency.Pair{}, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	// The recorded tickers of the first listed instrument of each asset
	spotTicker := ticker.Price{
		Last: 61723.2, High: 61964.8, Low: 58830, Bid: 61723.1, BidSize: 0.08109077, Ask: 61723.2, AskSize: 1.05305333,
		BaseVolume: 8873.34876322, QuoteVolume: 538894867.157978826, Open: 58874, Pair: mainPair, ExchangeName: e.Name,
		AssetType: asset.Spot, LastUpdated: time.UnixMilli(1795412938772),
	}
	marginTicker := spotTicker
	marginTicker.AssetType = asset.Margin
	mockTickers := map[asset.Item]ticker.Price{
		asset.Spot:   spotTicker,
		asset.Margin: marginTicker,
		asset.PerpetualSwap: {
			Last: 61607.7, High: 61840.6, Low: 58675.1, Bid: 61593, BidSize: 152.7, Ask: 61593.1, AskSize: 46.4,
			BaseVolume: 7357.1661, Open: 58717, Pair: btcUSDSwapPair,
			ExchangeName: e.Name, AssetType: asset.PerpetualSwap, LastUpdated: time.UnixMilli(1795412940567),
		},
		asset.Futures: {
			Last: 61648.8, High: 61837.1, Low: 58721.1, Bid: 61636.6, BidSize: 0.2, Ask: 61669, AskSize: 0.2,
			BaseVolume: 28.9871, Open: 58744.7, Pair: currency.NewPairWithDelimiter("BTC", "USD-261127", currency.DashDelimiter),
			ExchangeName: e.Name, AssetType: asset.Futures, LastUpdated: time.UnixMilli(1795412939671),
		},
		asset.Options: {
			Pair:         currency.NewPairWithDelimiter("SOL", "USD_UM-261123-54-C", currency.DashDelimiter),
			ExchangeName: e.Name, AssetType: asset.Options, LastUpdated: time.UnixMilli(1795412940122),
		},
		// The spread's bid and ask come from its book, which was empty, and its time is the book's, being newer
		asset.Spread: {
			Last: 2285, High: 2285, Low: 2285, Open: 2285,
			Pair:         currency.NewPairWithDelimiter("BTC", "USD-261225_BTC-USD-270625", currency.DashDelimiter),
			ExchangeName: e.Name, AssetType: asset.Spread, LastUpdated: time.UnixMilli(1795412978495),
		},
	}
	for _, a := range e.GetAssetTypes(false) {
		p, err := e.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs for asset %s must not error", a)
		require.NotEmptyf(t, p, "GetAvailablePairs for asset %s must not return empty pairs", a)
		result, err := e.UpdateTicker(contextGenerate(), p[0], a)
		require.NoErrorf(t, err, "UpdateTicker for asset %s and pair %s must not error", a, p[0])
		require.NotNilf(t, result, "UpdateTicker for asset %s and pair %s must not return nil", a, p[0])
		if mockTests {
			exp, ok := mockTickers[a]
			require.Truef(t, ok, "a recorded ticker must be expected for asset %s", a)
			assert.Equalf(t, &exp, result, "UpdateTicker should map the recorded %s ticker", a)
		}
	}

	ticks, err := e.GetTickers(contextGenerate(), &TickersRequest{InstrumentType: instTypeSwap})
	require.NoError(t, err, "GetTickers must not error")
	var checked bool
	for i := range ticks {
		if ticks[i].TradingVolume24HourInCurrency.Float64() == 0 || ticks[i].TradingVolume24HourInCurrency.Float64() == ticks[i].TradingVolume24HourInContract.Float64() {
			continue
		}
		got, err := e.UpdateTicker(contextGenerate(), ticks[i].InstrumentID, asset.PerpetualSwap)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s", ticks[i].InstrumentID)
		assert.InEpsilonf(t, ticks[i].TradingVolume24HourInCurrency.Float64(), got.BaseVolume, 0.05, "UpdateTicker should take the base volume for %s from volCcy24h", ticks[i].InstrumentID)
		assert.Zerof(t, got.QuoteVolume, "UpdateTicker should report no quote volume for swap %s, since vol24h counts contracts", ticks[i].InstrumentID)
		checked = true
		break
	}
	require.True(t, checked, "must find a swap whose base volume and contract count differ")

	// InEpsilon errors outright on a zero expected value, so pin this to a pair that always
	// trades rather than whichever happens to sit first in the enabled list
	spotTick, err := e.GetTicker(contextGenerate(), mainPair.String())
	require.NoError(t, err, "GetTicker must not error")
	require.Positive(t, spotTick.TradingVolume24HourInContract.Float64(), "vol24h for the spot pair under test must be non-zero")
	spotGot, err := e.UpdateTicker(contextGenerate(), mainPair, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.InEpsilonf(t, spotTick.TradingVolume24HourInContract.Float64(), spotGot.BaseVolume, 0.05, "UpdateTicker should take the base volume for spot %s from vol24h", mainPair)
	assert.InEpsilonf(t, spotTick.TradingVolume24HourInCurrency.Float64(), spotGot.QuoteVolume, 0.05, "UpdateTicker should take the quote volume for spot %s from volCcy24h", mainPair)
	// OKX stamps a market ticker's ts at response time, so a staleness bound cannot tell it from
	// the time.Now() that ProcessTicker substitutes for a zero. The monotonic clock reading can:
	// time.Now() carries one, a timestamp decoded through time.UnixMilli does not, and Round(0)
	// strips it. Round(0) alone would still admit a time.Now().UTC(), so pin the granularity as
	// well: ts is milliseconds, where a wall clock reading is nanoseconds
	assert.Equalf(t, spotGot.LastUpdated.Round(0).String(), spotGot.LastUpdated.String(),
		"UpdateTicker should record the exchange timestamp for spot %s, not the time of the call", mainPair)
	assert.Zerof(t, spotGot.LastUpdated.UnixNano()%int64(time.Millisecond),
		"UpdateTicker should record the exchange timestamp for spot %s to the millisecond it was sent in", mainPair)

	t.Run("a pre-open instrument's crossed ticker", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/market/ticker", r.URL.Path, "UpdateTicker should request the instrument's ticker")
			writeOKXData(t, w, []map[string]string{{"instType": "SPOT", "instId": "BTC-USDT", "bidPx": "101", "askPx": "99", "ts": "1700000001000"}})
		}))
		ex.Name = t.Name()
		ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 31, State: instrumentStatePreOpen}})
		result, err := ex.UpdateTicker(t.Context(), mainPair, asset.Spot)
		require.NoError(t, err, "UpdateTicker must accept a pre-open instrument's crossed ticker")
		exp := &ticker.Price{Bid: 101, Ask: 99, Pair: mainPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1700000001000), PreOpen: true}
		assert.Equal(t, exp, result, "UpdateTicker should store the pre-open ticker")
	})
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()

	e := newIsolatedExchange(t)
	// Renamed before UpdatePairsOnce, which caches by name, so the package's e still loads its own instruments
	e.Name = t.Name()
	testexch.UpdatePairsOnce(t, e)
	spotPair := mainPair
	if mockTests {
		// The recorded ticker lists keep only their first entries, so the checked pairs come from those. The spread is
		// pinned since UpdatePairsOnce enables a random listed spread in place of the expired configured one
		spotPair = currency.NewPairWithDelimiter("PEOPLE", "EUR", currency.DashDelimiter)
		sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.Spot, spotPair)
		sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.PerpetualSwap, currency.NewPairWithDelimiter("EDGE", "USDT-SWAP", currency.DashDelimiter))
		require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spread, currency.Pairs{spreadPair}, true), "StorePairs must not error")
	}

	for _, a := range e.GetAssetTypes(false) {
		err := e.UpdateTickers(contextGenerate(), a)
		require.NoErrorf(t, err, "UpdateTickers for asset %s must not error", a)
	}

	// around half the swaps report both figures identically and cannot tell the fields apart
	pairs, err := e.GetEnabledPairs(asset.PerpetualSwap)
	require.NoError(t, err, "GetEnabledPairs must not error")
	ticks, err := e.GetTickers(contextGenerate(), &TickersRequest{InstrumentType: instTypeSwap})
	require.NoError(t, err, "GetTickers must not error")
	byInstrument := make(map[string]TickerResponse, len(ticks))
	for i := range ticks {
		byInstrument[ticks[i].InstrumentID.String()] = ticks[i]
	}
	var swapStored *ticker.Price
	for _, p := range pairs {
		tick, ok := byInstrument[p.String()]
		if !ok || tick.TradingVolume24HourInCurrency.Float64() == 0 || tick.TradingVolume24HourInCurrency.Float64() == tick.TradingVolume24HourInContract.Float64() {
			continue
		}
		swapStored, err = ticker.GetTicker(e.Name, p, asset.PerpetualSwap)
		require.NoErrorf(t, err, "GetTicker must not error for %s", p)
		assert.InEpsilonf(t, tick.TradingVolume24HourInCurrency.Float64(), swapStored.BaseVolume, 0.05, "UpdateTickers should take the base volume for %s from volCcy24h", p)
		assert.Zerof(t, swapStored.QuoteVolume, "UpdateTickers should report no quote volume for swap %s, since vol24h counts contracts", p)
		break
	}
	require.NotNil(t, swapStored, "must find an enabled swap whose base volume and contract count differ")

	// spot takes the opposite mapping. InEpsilon errors outright on a zero expected value, so pin
	// this to a pair that always trades rather than whichever sits first in the enabled list
	spotTick, err := e.GetTicker(contextGenerate(), spotPair.String())
	require.NoError(t, err, "GetTicker must not error")
	require.Positive(t, spotTick.TradingVolume24HourInContract.Float64(), "vol24h for the spot pair under test must be non-zero")
	spotStored, err := ticker.GetTicker(e.Name, spotPair, asset.Spot)
	require.NoError(t, err, "ticker.GetTicker must not error")
	assert.InEpsilonf(t, spotTick.TradingVolume24HourInContract.Float64(), spotStored.BaseVolume, 0.05,
		"UpdateTickers should take the base volume for spot %s from vol24h", spotPair)
	assert.InEpsilonf(t, spotTick.TradingVolume24HourInCurrency.Float64(), spotStored.QuoteVolume, 0.05,
		"UpdateTickers should take the quote volume for spot %s from volCcy24h", spotPair)
	assert.Equalf(t, spotStored.LastUpdated.Round(0).String(), spotStored.LastUpdated.String(),
		"UpdateTickers should record the exchange timestamp for spot %s, not the time of the call", spotPair)
	assert.Zerof(t, spotStored.LastUpdated.UnixNano()%int64(time.Millisecond),
		"UpdateTickers should record the exchange timestamp for spot %s to the millisecond it was sent in", spotPair)

	// the spread book can move between the store write and a verifying re-fetch, so the same
	// monotonic check settles this without racing the exchange
	spreadPairs, err := e.GetEnabledPairs(asset.Spread)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.NotEmpty(t, spreadPairs, "GetEnabledPairs must return spread pairs")
	spreadStored, err := ticker.GetTicker(e.Name, spreadPairs[0], asset.Spread)
	require.NoErrorf(t, err, "ticker.GetTicker must not error for spread %s", spreadPairs[0])
	assert.Equalf(t, spreadStored.LastUpdated.Round(0).String(), spreadStored.LastUpdated.String(),
		"UpdateTickers should record the exchange timestamp for spread %s, not the time of the call", spreadPairs[0])
	assert.Zerof(t, spreadStored.LastUpdated.UnixNano()%int64(time.Millisecond),
		"UpdateTickers should record the exchange timestamp for spread %s to the millisecond it was sent in", spreadPairs[0])

	if mockTests {
		// The spread's bid, ask and time come from its book, which is newer than the ticker's cached snapshot
		exp := []*ticker.Price{
			{
				Last: 0.2532, High: 0.2555, Low: 0.2332, Bid: 0.2531, BidSize: 12, Ask: 0.2533, AskSize: 84, BaseVolume: 5082620,
				Open: 0.2334, Pair: currency.NewPairWithDelimiter("EDGE", "USDT-SWAP", currency.DashDelimiter), ExchangeName: e.Name,
				AssetType: asset.PerpetualSwap, LastUpdated: time.UnixMilli(1795412935369),
			},
			{
				Last: 0.004327, High: 0.004335, Low: 0.004316, Bid: 0.004393, BidSize: 155749.440638, Ask: 0.004408, AskSize: 155219.440273,
				BaseVolume: 196776.508803, QuoteVolume: 851.782144058309, Open: 0.004316, Pair: spotPair, ExchangeName: e.Name,
				AssetType: asset.Spot, LastUpdated: time.UnixMilli(1795412933366),
			},
			{
				Last: -30.5, High: -30.5, Low: -30.5, Bid: -63.9, BidSize: 0.2297, Ask: -8.7, AskSize: 2.7457, Open: -30.5,
				Pair: spreadPair, ExchangeName: e.Name, AssetType: asset.Spread, LastUpdated: time.UnixMilli(1795406573867),
			},
		}
		assert.Equal(t, exp, []*ticker.Price{swapStored, spotStored, spreadStored}, "UpdateTickers should store the recorded tickers")
	}

	t.Run("a pre-open instrument's crossed ticker", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/market/tickers", r.URL.Path, "UpdateTickers should request the asset's tickers")
			writeOKXData(t, w, []map[string]string{{"instType": "SPOT", "instId": "BTC-USDT", "bidPx": "101", "askPx": "99", "ts": "1700000001000"}})
		}))
		ex.Name = t.Name()
		ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 31, State: instrumentStatePreOpen}})
		require.NoError(t, ex.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers must accept a pre-open instrument's crossed ticker")
		stored, err := ticker.GetTicker(ex.Name, mainPair, asset.Spot)
		require.NoError(t, err, "GetTicker must not error")
		exp := &ticker.Price{Bid: 101, Ask: 99, Pair: mainPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1700000001000), PreOpen: true}
		assert.Equal(t, exp, stored, "UpdateTickers should store the pre-open ticker")
	})
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	t.Run("rejects a malformed book reply", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, data string
		}{
			{"null", "null"},
			{"null row", "[null]"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/market/books", r.URL.Path, "the book should come from the instrument's order book")
					_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				ex.Name = t.Name()
				var err error
				require.NotPanics(t, func() { _, err = ex.UpdateOrderbook(t.Context(), mainPair, asset.Spot) }, "UpdateOrderbook must not panic on a malformed reply")
				assert.ErrorIs(t, err, common.ErrInvalidResponse, "UpdateOrderbook should reject a success reply without a book")
			})
		}
	})
	t.Run("keeps a spread book's zero-priced levels", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			bids, asks [][3]string
			exp        orderbook.Book
		}{
			{
				name: "zero bid",
				bids: [][3]string{{"0", "1", "1"}, {"-1.5", "2", "3"}}, asks: [][3]string{{"0.5", "4", "1"}},
				exp: orderbook.Book{
					Bids: orderbook.Levels{{Price: 0, Amount: 1, OrderCount: 1}, {Price: -1.5, Amount: 2, OrderCount: 3}},
					Asks: orderbook.Levels{{Price: 0.5, Amount: 4, OrderCount: 1}},
				},
			},
			{
				name: "zero ask",
				bids: [][3]string{{"-0.5", "1", "1"}}, asks: [][3]string{{"0", "2", "2"}, {"1.5", "3", "1"}},
				exp: orderbook.Book{
					Bids: orderbook.Levels{{Price: -0.5, Amount: 1, OrderCount: 1}},
					Asks: orderbook.Levels{{Price: 0, Amount: 2, OrderCount: 2}, {Price: 1.5, Amount: 3, OrderCount: 1}},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/sprd/books", r.URL.Path, "the book should come from the spread order book")
					assert.Equal(t, url.Values{"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}, "sz": {"50"}}, r.URL.Query(), "the request should name the spread")
					writeOKXData(t, w, []map[string]any{{"bids": tc.bids, "asks": tc.asks, "ts": "1700000000000"}})
				}))
				ex.Name = t.Name()
				book, err := ex.UpdateOrderbook(t.Context(), spreadPair, asset.Spread)
				require.NoError(t, err, "UpdateOrderbook must not error")
				assert.Equal(t, tc.exp.Bids, book.Bids, "UpdateOrderbook should keep every bid, zero-priced ones included")
				assert.Equal(t, tc.exp.Asks, book.Asks, "UpdateOrderbook should keep every ask, zero-priced ones included")
			})
		}
	})
	testexch.UpdatePairsOnce(t, e)
	spotBook := orderbook.Book{
		Exchange:          e.Name,
		Pair:              mainPair,
		Asset:             asset.Spot,
		ValidateOrderbook: true,
		RestSnapshot:      true,
		LastUpdated:       time.UnixMilli(1795406739809),
		Bids:              orderbook.Levels{{Price: 61256.7, Amount: 0.22362398, OrderCount: 4}, {Price: 61256.6, Amount: 0.28443965, OrderCount: 2}, {Price: 61256.5, Amount: 0.23084313, OrderCount: 1}, {Price: 61253.5, Amount: 0.40074797, OrderCount: 1}},
		Asks:              orderbook.Levels{{Price: 61256.8, Amount: 0.46269468, OrderCount: 7}, {Price: 61256.9, Amount: 0.21597674, OrderCount: 1}, {Price: 61258.7, Amount: 0.00320868, OrderCount: 1}, {Price: 61258.8, Amount: 0.0000102, OrderCount: 1}},
	}
	marginBook := spotBook
	marginBook.Asset = asset.Margin
	mockBooks := map[asset.Item]orderbook.Book{
		asset.Spot:   spotBook,
		asset.Margin: marginBook,
		asset.PerpetualSwap: {
			Exchange:          e.Name,
			Pair:              btcUSDSwapPair,
			Asset:             asset.PerpetualSwap,
			ValidateOrderbook: true,
			RestSnapshot:      true,
			LastUpdated:       time.UnixMilli(1795412226953),
			Bids:              orderbook.Levels{{Price: 61374.5, Amount: 0.1, OrderCount: 1}, {Price: 61368.9, Amount: 1, OrderCount: 1}, {Price: 61366.2, Amount: 1, OrderCount: 1}, {Price: 61363.4, Amount: 1, OrderCount: 1}},
			Asks:              orderbook.Levels{{Price: 61374.6, Amount: 1213.6, OrderCount: 14}, {Price: 61374.7, Amount: 79.1, OrderCount: 1}, {Price: 61374.8, Amount: 0.2, OrderCount: 1}, {Price: 61378.4, Amount: 103.8, OrderCount: 1}},
		},
		asset.Futures: {
			Exchange:          e.Name,
			Pair:              currency.NewPairWithDelimiter("BTC", "USD-261127", currency.DashDelimiter),
			Asset:             asset.Futures,
			ValidateOrderbook: true,
			RestSnapshot:      true,
			LastUpdated:       time.UnixMilli(1795412227953),
			Bids:              orderbook.Levels{{Price: 61417, Amount: 0.2, OrderCount: 1}, {Price: 61411.1, Amount: 3.4, OrderCount: 1}, {Price: 61411, Amount: 105, OrderCount: 1}, {Price: 61410.9, Amount: 0.2, OrderCount: 1}},
			Asks:              orderbook.Levels{{Price: 61436.8, Amount: 105, OrderCount: 1}, {Price: 61447.4, Amount: 0.2, OrderCount: 1}, {Price: 61453.5, Amount: 0.2, OrderCount: 1}, {Price: 61459.6, Amount: 0.2, OrderCount: 1}},
		},
		asset.Options: {
			Exchange:          e.Name,
			Pair:              currency.NewPairWithDelimiter("SOL", "USD_UM-261123-54-C", currency.DashDelimiter),
			Asset:             asset.Options,
			ValidateOrderbook: true,
			RestSnapshot:      true,
			LastUpdated:       time.UnixMilli(1795412225908),
			Bids:              orderbook.Levels{},
			Asks:              orderbook.Levels{},
		},
		asset.Spread: {
			Exchange:          e.Name,
			Pair:              currency.NewPairWithDelimiter("BTC", "USD-261225_BTC-USD-270625", currency.DashDelimiter),
			Asset:             asset.Spread,
			ValidateOrderbook: true,
			RestSnapshot:      true,
			LastUpdated:       time.UnixMilli(1795407774077),
			Bids:              orderbook.Levels{},
			Asks:              orderbook.Levels{},
		},
	}
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			p, err := e.GetAvailablePairs(a)
			require.NoErrorf(t, err, "GetAvailablePairs for asset %s must not error", a)
			require.NotEmptyf(t, p, "GetAvailablePairs for asset %s must not return empty pairs", a)
			result, err := e.UpdateOrderbook(contextGenerate(), p[0], a)
			require.NoErrorf(t, err, "UpdateOrderbook for asset %s and pair %s must not error", a, p[0])
			require.NotNilf(t, result, "UpdateOrderbook for asset %s and pair %s must not return nil", a, p[0])
			if mockTests {
				exp, ok := mockBooks[a]
				require.Truef(t, ok, "a recorded book must be expected for asset %s", a)
				exp.InsertedAt = result.InsertedAt
				assert.Equalf(t, &exp, result, "UpdateOrderbook should map the recorded %s book", a)
			}
		})
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UpdateAccountBalances(contextGenerate(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	if mockTests {
		require.Len(t, result, 1, "UpdateAccountBalances must return one sub-account")
		exp := accounts.SubAccounts{{
			AssetType: asset.Spot,
			Balances: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 0.8034, Hold: 0.05, Free: 0.7512, UpdatedAt: time.UnixMilli(1795409800000)},
				currency.USDT: {Currency: currency.USDT, Total: 4992.890093622894, Hold: 158.573, Free: 4834.317093622894, Borrowed: 21625.64, UpdatedAt: time.UnixMilli(1705449605015)},
				currency.USDC: {Currency: currency.USDC, Total: 1521.75, Hold: 20, Free: 1500.25, UpdatedAt: time.UnixMilli(1795409800000)},
			},
		}}
		assert.Equal(t, exp, result, "UpdateAccountBalances should map every currency's balance and update time")
	} else {
		assert.NotNil(t, result, "UpdateAccountBalances should return the account's balances")
	}

	creds := &accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"}
	t.Run("maps a currency's liabilities as borrowed", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/account/balance", r.URL.Path, "the balances should come from the trading account")
			writeOKXData(t, w, []map[string]any{{"uTime": "1795400000000", "details": []map[string]string{
				{"ccy": "USDT", "eq": "100", "cashBal": "75", "availBal": "60", "frozenBal": "40", "liab": "25", "uTime": "1795400000000"},
			}}})
		}))
		ex.SetCredentials(creds)
		got, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		usdt := accounts.Balance{Currency: currency.USDT, Total: 100, Hold: 40, Free: 60, Borrowed: 25, UpdatedAt: time.UnixMilli(1795400000000)}
		exp := accounts.SubAccounts{{AssetType: asset.Margin, Balances: accounts.CurrencyBalances{currency.USDT: usdt}}}
		assert.Equal(t, exp, got, "UpdateAccountBalances should report the currency's liabilities as borrowed")
		stored, err := ex.Accounts.GetBalance("", creds, asset.Margin, currency.USDT)
		require.NoError(t, err, "GetBalance must find the saved balance")
		assert.Equal(t, usdt, stored, "UpdateAccountBalances should save the currency's liabilities as borrowed")
	})

	t.Run("replaces the stored balances only with an account", func(t *testing.T) {
		t.Parallel()
		btc := accounts.Balance{Currency: currency.BTC, Total: 1, Free: 1, UpdatedAt: time.UnixMilli(1795400000000)}
		for _, tc := range []struct {
			name   string
			data   any
			expErr error
			exp    accounts.CurrencyBalances
		}{
			{name: "no account", data: []any{}, expErr: common.ErrInvalidResponse, exp: accounts.CurrencyBalances{currency.BTC: btc}},
			{name: "a null account", data: []any{nil}, expErr: common.ErrInvalidResponse, exp: accounts.CurrencyBalances{currency.BTC: btc}},
			{name: "an account holding nothing", data: []map[string]any{{"uTime": "1795400001000", "details": []any{}}}, exp: accounts.CurrencyBalances{}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/account/balance", r.URL.Path, "the balances should come from the trading account")
					writeOKXData(t, w, tc.data)
				}))
				ex.SetCredentials(creds)
				held := accounts.NewSubAccount(asset.Spot, "")
				held.Balances.Set(currency.BTC, btc)
				require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{held}, true), "Save must store the earlier balances")
				_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
				require.ErrorIs(t, err, tc.expErr, "UpdateAccountBalances must return the expected error")
				stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
				require.NoError(t, err, "SubAccounts must not error")
				exp := accounts.SubAccounts{{AssetType: asset.Spot, Balances: tc.exp}}
				assert.Equal(t, exp, stored, "UpdateAccountBalances should replace the stored balances with an account's, and only an account's")
			})
		}
	})
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAccountFundingHistory(contextGenerate())
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetWithdrawalsHistory(contextGenerate(), currency.BTC, asset.Spot)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	result, err := e.GetRecentTrades(contextGenerate(), perpetualSwapPair, asset.PerpetualSwap)
	require.NoError(t, err, "GetRecentTrades must not error for a perpetual swap")
	if mockTests {
		exp := []trade.Data{
			{TID: "3120383456", Exchange: e.Name, CurrencyPair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Price: 61509.7, Amount: 1, Timestamp: time.UnixMilli(1795415254356)},
			{TID: "3120383455", Exchange: e.Name, CurrencyPair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Price: 61509.7, Amount: 31.86, Timestamp: time.UnixMilli(1795415254356)},
			{TID: "3120383454", Exchange: e.Name, CurrencyPair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Price: 61509.7, Amount: 18, Timestamp: time.UnixMilli(1795415254356)},
		}
		assert.Equal(t, exp, result, "GetRecentTrades should map every recorded trade in the order OKX sends them")
	} else {
		assert.NotEmpty(t, result, "GetRecentTrades should return trades for a perpetual swap")
	}

	result, err = e.GetRecentTrades(contextGenerate(), spreadPair, asset.Spread)
	require.NoError(t, err, "GetRecentTrades must not error for a spread")
	assert.NotNil(t, result, "GetRecentTrades should return trades for a spread")

	t.Run("a margin pair's trades are its spot instrument's", func(t *testing.T) {
		t.Parallel()
		trades, err := e.GetRecentTrades(contextGenerate(), mainPair, asset.Margin)
		require.NoError(t, err, "GetRecentTrades must not error for a margin pair")
		if mockTests {
			exp := []trade.Data{
				{TID: "1089358024", Exchange: e.Name, CurrencyPair: mainPair, AssetType: asset.Margin, Side: order.Sell, Price: 61534.7, Amount: 0.00001, Timestamp: time.UnixMilli(1795417636441)},
				{TID: "1089358025", Exchange: e.Name, CurrencyPair: mainPair, AssetType: asset.Margin, Side: order.Sell, Price: 61531.2, Amount: 0.000011, Timestamp: time.UnixMilli(1795417636528)},
				{TID: "1089358026", Exchange: e.Name, CurrencyPair: mainPair, AssetType: asset.Margin, Side: order.Sell, Price: 61531.2, Amount: 0.00001, Timestamp: time.UnixMilli(1795417636749)},
			}
			assert.Equal(t, exp, trades, "GetRecentTrades should map the spot instrument's trades as the margin pair's, oldest first")
		} else {
			require.NotEmpty(t, trades, "GetRecentTrades must return trades for a margin pair")
			assert.Equal(t, asset.Margin, trades[0].AssetType, "GetRecentTrades should report the trades under the margin asset")
		}
	})

	t.Run("requests the documented maximum of the instrument's trades", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/market/trades", r.URL.Path, "the trades should come from the instrument's trades")
			assert.Equal(t, url.Values{"instId": {perpetualSwapPair.String()}, "limit": {"500"}}, r.URL.Query(), "the request should ask for the documented maximum of the instrument's trades")
			writeOKXData(t, w, []map[string]string{{"instId": perpetualSwapPair.String(), "tradeId": "1", "px": "60000", "sz": "2", "side": "buy", "ts": "1700000000000"}})
		}))
		trades, err := ex.GetRecentTrades(t.Context(), perpetualSwapPair, asset.PerpetualSwap)
		require.NoError(t, err, "GetRecentTrades must not error")
		exp := []trade.Data{{
			TID: "1", Exchange: ex.Name, CurrencyPair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy,
			Price: 60000, Amount: 2, Timestamp: time.UnixMilli(1700000000000),
		}}
		assert.Equal(t, exp, trades, "GetRecentTrades should map the instrument's trades")
	})

	t.Run("requests the spread's trades", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/sprd/public-trades", r.URL.Path, "the trades should come from the public spread trades")
			assert.Equal(t, url.Values{"sprdId": {"BTC-USDT_BTC-USDT-SWAP"}}, r.URL.Query(), "the request should name the spread")
			writeOKXData(t, w, []map[string]string{{"sprdId": "BTC-USDT_BTC-USDT-SWAP", "tradeId": "2", "px": "-63.9", "sz": "0.5", "side": "sell", "ts": "1700000000000"}})
		}))
		trades, err := ex.GetRecentTrades(t.Context(), spreadPair, asset.Spread)
		require.NoError(t, err, "GetRecentTrades must not error")
		exp := []trade.Data{{
			TID: "2", Exchange: ex.Name, CurrencyPair: spreadPair, AssetType: asset.Spread, Side: order.Sell,
			Price: -63.9, Amount: 0.5, Timestamp: time.UnixMilli(1700000000000),
		}}
		assert.Equal(t, exp, trades, "GetRecentTrades should map the spread's trades")
	})

	t.Run("rejects a spread without a pair", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}))
		_, err := ex.GetRecentTrades(t.Context(), currency.EMPTYPAIR, asset.Spread)
		require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetRecentTrades must not request every spread's trades for an empty pair")
	})
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	var resp []PlaceOrderRequest
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	require.NoError(t, err)

	_, err = e.SubmitOrder(contextGenerate(), nil)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder should error for a nil submission")

	arg := &order.Submit{
		Exchange:  e.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		ClientID:  "yeneOrder",
		AssetType: asset.Binary,
	}
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Amount = 1000000000
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Pair = mainPair
	arg.AssetType = asset.Futures
	arg.Leverage = -1
	_, err = e.SubmitOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported)

	t.Run("only a spot market order is sized by tgtCcy", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name        string
			assetType   asset.Item
			marginType  margin.Type
			orderType   order.Type
			tif         order.TimeInForce
			side        order.Side
			quoteAmount float64
			expType     string
			expSize     string
			expTarget   string // empty when no tgtCcy is sent
			expPrice    string // empty when no px is sent
		}{
			{"quote sized market buy", asset.Spot, margin.Unset, order.Market, order.UnknownTIF, order.Buy, 100, orderMarket, "100", targetCurrencyQuote, ""},
			{"base sized market buy", asset.Spot, margin.Unset, order.Market, order.UnknownTIF, order.Buy, 0, orderMarket, "0.002", targetCurrencyBase, ""},
			{"quote sized market sell", asset.Spot, margin.Unset, order.Market, order.UnknownTIF, order.Sell, 100, orderMarket, "100", targetCurrencyQuote, ""},
			{"quote sized market buy in cross mode", asset.Spot, margin.Multi, order.Market, order.UnknownTIF, order.Buy, 100, orderMarket, "100", "", ""},
			{"quote sized market buy in isolated mode", asset.Spot, margin.Isolated, order.Market, order.UnknownTIF, order.Buy, 100, orderMarket, "100", "", ""},
			{"market sell in cross mode", asset.Spot, margin.Multi, order.Market, order.UnknownTIF, order.Sell, 100, orderMarket, "0.002", "", ""},
			{"quote sized market buy in spot lead trading", asset.Spot, margin.SpotIsolated, order.Market, order.UnknownTIF, order.Buy, 100, orderMarket, "100", targetCurrencyQuote, ""},
			{"market buy with immediate or cancel", asset.Spot, margin.Unset, order.Market, order.ImmediateOrCancel, order.Buy, 100, orderIOC, "0.002", "", "50000"},
			{"market sell with immediate or cancel", asset.Spot, margin.Unset, order.Market, order.ImmediateOrCancel, order.Sell, 100, orderIOC, "0.002", "", "50000"},
			{"market buy with fill or kill", asset.Spot, margin.Unset, order.Market, order.FillOrKill, order.Buy, 100, orderFOK, "0.002", "", "50000"},
			{"limit buy with immediate or cancel", asset.Spot, margin.Unset, order.Limit, order.ImmediateOrCancel, order.Buy, 100, orderIOC, "0.002", "", "50000"},
			{"margin market buy", asset.Margin, margin.Unset, order.Market, order.UnknownTIF, order.Buy, 100, orderMarket, "100", "", ""},
			{"margin market sell", asset.Margin, margin.Unset, order.Market, order.UnknownTIF, order.Sell, 100, orderMarket, "0.002", "", ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "3", "posMode": positionModeNet}})
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the place order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: mainPair, AssetType: tc.assetType, MarginType: tc.marginType, Side: tc.side,
					Type: tc.orderType, TimeInForce: tc.tif, Amount: 0.002, QuoteAmount: tc.quoteAmount, Price: 50000,
				})
				require.NoError(t, err, "SubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
				assert.Equal(t, tc.expType, sent["ordType"], "ordType should follow the order type and time in force")
				assert.Equal(t, tc.expSize, sent["sz"], "sz should be stated in the currency tgtCcy names")
				if tc.expTarget == "" {
					assert.NotContains(t, sent, "tgtCcy", "only a spot market order should send tgtCcy")
				} else {
					assert.Equal(t, tc.expTarget, sent["tgtCcy"], "tgtCcy should name the size's currency")
				}
				if tc.expPrice == "" {
					assert.NotContains(t, sent, "px", "a market order should send no price")
				} else {
					assert.Equal(t, tc.expPrice, sent["px"], "an ioc or fok order should send its price")
				}
			})
		}
	})

	t.Run("sizes an at-market algo buy in the currency OKX reads", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name        string
			assetType   asset.Item
			marginType  margin.Type
			orderType   order.Type
			side        order.Side
			price       float64
			takeProfit  float64
			stopLoss    float64
			quoteAmount float64
			expSize     string
			expTarget   string // empty when no tgtCcy is sent
			expErr      error
		}{
			{name: "spot trigger at market", assetType: asset.Spot, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, expErr: order.ErrAmountMustBeSet},
			{name: "spot trigger at market in cross mode", assetType: asset.Spot, marginType: margin.Multi, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, expErr: order.ErrAmountMustBeSet},
			{name: "spot conditional at market in isolated mode", assetType: asset.Spot, marginType: margin.Isolated, orderType: order.ConditionalStop, side: order.Buy, price: marketOrderPrice, expErr: order.ErrAmountMustBeSet},
			{name: "spot oco with a market stop loss in cross mode", assetType: asset.Spot, marginType: margin.Multi, orderType: order.OCO, side: order.Buy, takeProfit: 41000, stopLoss: marketOrderPrice, expErr: order.ErrUnknownSubmissionAmountType},
			{name: "spot trigger at market in cash mode", assetType: asset.Spot, marginType: margin.NoMargin, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, expErr: order.ErrAmountMustBeSet},
			{name: "spot trigger sell at market", assetType: asset.Spot, orderType: order.Trigger, side: order.Sell, price: marketOrderPrice, expSize: "0.002"},
			{name: "spot trigger at a limit price", assetType: asset.Spot, orderType: order.Trigger, side: order.Buy, price: 42000, expSize: "0.002"},
			{name: "spot conditional at market", assetType: asset.Spot, orderType: order.ConditionalStop, side: order.Buy, price: marketOrderPrice, expSize: "0.002", expTarget: targetCurrencyBase},
			{name: "spot conditional sell at market", assetType: asset.Spot, orderType: order.ConditionalStop, side: order.Sell, price: marketOrderPrice, expSize: "0.002"},
			{name: "spot conditional at a limit price", assetType: asset.Spot, orderType: order.ConditionalStop, side: order.Buy, price: 42000, expSize: "0.002"},
			{name: "margin trigger at market", assetType: asset.Margin, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, expErr: order.ErrAmountMustBeSet},
			{name: "margin conditional at market", assetType: asset.Margin, orderType: order.ConditionalStop, side: order.Buy, price: marketOrderPrice, expErr: order.ErrAmountMustBeSet},
			{name: "oco with a market take profit", assetType: asset.Spot, orderType: order.OCO, side: order.Buy, takeProfit: marketOrderPrice, stopLoss: 42000, expErr: order.ErrUnknownSubmissionAmountType},
			{name: "oco with a market stop loss", assetType: asset.Spot, orderType: order.OCO, side: order.Buy, takeProfit: 41000, stopLoss: marketOrderPrice, expErr: order.ErrUnknownSubmissionAmountType},
			{name: "oco sell with a market stop loss", assetType: asset.Spot, orderType: order.OCO, side: order.Sell, takeProfit: 41000, stopLoss: marketOrderPrice, expSize: "0.002"},
			{name: "oco sell with a market take profit", assetType: asset.Spot, orderType: order.OCO, side: order.Sell, takeProfit: marketOrderPrice, stopLoss: 42000, expSize: "0.002"},
			{name: "margin oco with a market stop loss", assetType: asset.Margin, orderType: order.OCO, side: order.Buy, takeProfit: 41000, stopLoss: marketOrderPrice, expErr: order.ErrUnknownSubmissionAmountType},
			{name: "margin oco with a market take profit", assetType: asset.Margin, orderType: order.OCO, side: order.Buy, takeProfit: marketOrderPrice, stopLoss: 42000, expErr: order.ErrUnknownSubmissionAmountType},
			{name: "oco with both legs limited", assetType: asset.Spot, orderType: order.OCO, side: order.Buy, takeProfit: 41000, stopLoss: 43000, expSize: "0.002"},
			{name: "spot trigger at market sized by a quote amount", assetType: asset.Spot, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, quoteAmount: 80, expSize: "80"},
			{name: "spot trigger at market in cross mode sized by a quote amount", assetType: asset.Spot, marginType: margin.Multi, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, quoteAmount: 80, expSize: "80"},
			{name: "spot conditional at market sized by a quote amount", assetType: asset.Spot, orderType: order.ConditionalStop, side: order.Buy, price: marketOrderPrice, quoteAmount: 80, expSize: "0.002", expTarget: targetCurrencyBase},
			{name: "spot conditional at market in isolated mode sized by a quote amount", assetType: asset.Spot, marginType: margin.Isolated, orderType: order.ConditionalStop, side: order.Buy, price: marketOrderPrice, quoteAmount: 80, expSize: "80"},
			{name: "margin trigger at market sized by a quote amount", assetType: asset.Margin, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, quoteAmount: 80, expSize: "80"},
			{name: "margin conditional at market sized by a quote amount", assetType: asset.Margin, orderType: order.ConditionalStop, side: order.Buy, price: marketOrderPrice, quoteAmount: 80, expSize: "80"},
			{name: "oco with both legs at market sized by a quote amount", assetType: asset.Spot, orderType: order.OCO, side: order.Buy, takeProfit: marketOrderPrice, stopLoss: marketOrderPrice, quoteAmount: 80, expSize: "80"},
			{name: "oco with both legs at market", assetType: asset.Spot, orderType: order.OCO, side: order.Buy, takeProfit: marketOrderPrice, stopLoss: marketOrderPrice, expErr: order.ErrAmountMustBeSet},
			{name: "margin oco with both legs at market sized by a quote amount", assetType: asset.Margin, orderType: order.OCO, side: order.Buy, takeProfit: marketOrderPrice, stopLoss: marketOrderPrice, quoteAmount: 80, expSize: "80"},
			{name: "spot trigger at a limit price beside a quote amount", assetType: asset.Spot, orderType: order.Trigger, side: order.Buy, price: 42000, quoteAmount: 80, expSize: "0.002"},
			{name: "perpetual swap trigger at market", assetType: asset.PerpetualSwap, orderType: order.Trigger, side: order.Buy, price: marketOrderPrice, expSize: "0.002"},
			{name: "perpetual swap oco with a market stop loss", assetType: asset.PerpetualSwap, orderType: order.OCO, side: order.Buy, takeProfit: 41000, stopLoss: marketOrderPrice, expSize: "0.002"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "3", "posMode": positionModeNet}})
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the place algo order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"algoId": "ALGO-1", "sCode": "0"}})
				}))
				pair := mainPair
				if tc.assetType == asset.PerpetualSwap {
					pair = perpetualSwapPair
				}
				s := &order.Submit{
					Exchange: ex.Name, Pair: pair, AssetType: tc.assetType, MarginType: tc.marginType, Side: tc.side,
					Type: tc.orderType, Amount: 0.002, QuoteAmount: tc.quoteAmount, Price: tc.price, TriggerPrice: 41000,
				}
				if tc.orderType == order.OCO {
					// OKX takes an OCO leg's trigger price, and its order price
					// decides whether that leg executes at market.
					s.RiskManagementModes = order.RiskManagementModes{
						TakeProfit: order.RiskManagement{Price: 45000, LimitPrice: tc.takeProfit},
						StopLoss:   order.RiskManagement{Price: 35000, LimitPrice: tc.stopLoss},
					}
				}
				resp, err := ex.SubmitOrder(t.Context(), s)
				mu.Lock()
				defer mu.Unlock()
				if tc.expErr != nil {
					require.ErrorIs(t, err, tc.expErr, "an order whose size OKX would read in another currency must be refused")
					assert.Nil(t, body, "a refused order should not reach OKX")
					return
				}
				require.NoError(t, err, "SubmitOrder must not error")
				assert.Equal(t, "ALGO-1", resp.OrderID, "the algo order should return its ID")
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the place algo order request body must decode")
				assert.Equal(t, tc.expSize, sent["sz"], "sz should be stated in the currency OKX reads it in")
				if tc.expTarget == "" {
					assert.NotContains(t, sent, "tgtCcy", "only a spot conditional buy at market should send tgtCcy")
				} else {
					assert.Equal(t, tc.expTarget, sent["tgtCcy"], "a spot conditional buy at market should name its base currency size")
				}
			})
		}
	})

	t.Run("checks each OCO leg as an attached leg is checked", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name                 string
			side                 order.Side
			amount, quoteAmount  float64
			takeProfit, stopLoss order.RiskManagement
			expBody              string
			expErr               error
		}{
			{
				name: "a quote sized buy with limit legs missing their limit prices", side: order.Buy, quoteAmount: 100,
				takeProfit: order.RiskManagement{Enabled: true, OrderType: order.Limit, Price: 95}, stopLoss: order.RiskManagement{Enabled: true, OrderType: order.Limit, Price: 110},
				expErr: limits.ErrPriceBelowMin,
			},
			{
				name: "a sell with a limit take profit missing its limit price", side: order.Sell, amount: 1,
				takeProfit: order.RiskManagement{OrderType: order.Limit, Price: 110}, stopLoss: order.RiskManagement{Price: 90},
				expErr: limits.ErrPriceBelowMin,
			},
			{
				name: "a market leg with a limit price", side: order.Sell, amount: 1,
				takeProfit: order.RiskManagement{OrderType: order.Market, Price: 110, LimitPrice: 111}, stopLoss: order.RiskManagement{Price: 90},
				expErr: order.ErrUnsupportedOrderType,
			},
			{
				name: "a leg of an order type OKX does not place", side: order.Sell, amount: 1,
				takeProfit: order.RiskManagement{Price: 110}, stopLoss: order.RiskManagement{OrderType: order.Stop, Price: 90},
				expErr: order.ErrUnsupportedOrderType,
			},
			{
				name: "a leg with a negative limit price", side: order.Sell, amount: 1,
				takeProfit: order.RiskManagement{Price: 110, LimitPrice: -2}, stopLoss: order.RiskManagement{Price: 90},
				expErr: limits.ErrPriceBelowMin,
			},
			{
				name: "a quote sized buy with market legs", side: order.Buy, quoteAmount: 100,
				takeProfit: order.RiskManagement{OrderType: order.Market, Price: 95}, stopLoss: order.RiskManagement{OrderType: order.Market, Price: 110},
				expBody: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"oco","sz":"100","algoClOrdId":"clientOne","tpTriggerPx":"95","tpTriggerPxType":"last","tpOrdPx":"-1","slTriggerPx":"110","slTriggerPxType":"last","slOrdPx":"-1"}`,
			},
			{
				name: "a buy with limit legs at their limit prices", side: order.Buy, amount: 1,
				takeProfit: order.RiskManagement{OrderType: order.Limit, Price: 95, LimitPrice: 94}, stopLoss: order.RiskManagement{OrderType: order.Limit, Price: 110, LimitPrice: 111},
				expBody: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"oco","sz":"1","algoClOrdId":"clientOne","tpTriggerPx":"95","tpTriggerPxType":"last","tpOrdPx":"94","slTriggerPx":"110","slTriggerPxType":"last","slOrdPx":"111"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var bodies []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/account/config":
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
					case "/trade/order-algo":
						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err, "reading the place algo order body should not error")
						mu.Lock()
						bodies = append(bodies, string(b))
						mu.Unlock()
						writeOKXData(t, w, []map[string]string{{"algoId": "1001", "algoClOrdId": "clientOne", "sCode": "0"}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: mainPair, AssetType: asset.Spot, Side: tc.side, Type: order.OCO, Amount: tc.amount, QuoteAmount: tc.quoteAmount,
					ClientOrderID: "clientOne", RiskManagementModes: order.RiskManagementModes{TakeProfit: tc.takeProfit, StopLoss: tc.stopLoss},
				})
				mu.Lock()
				defer mu.Unlock()
				if tc.expErr != nil {
					require.ErrorIs(t, err, tc.expErr, "SubmitOrder must refuse a leg it would not attach to an order")
					assert.Empty(t, bodies, "a refused OCO order should not reach OKX")
					return
				}
				require.NoError(t, err, "SubmitOrder must not error")
				assert.Equal(t, "1001", resp.OrderID, "the OCO order should return its algo ID")
				require.Len(t, bodies, 1, "one OCO order must be placed")
				assert.JSONEq(t, tc.expBody, bodies[0], "each leg should be placed at the price its order type takes")
			})
		}
	})

	t.Run("live", func(t *testing.T) {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		arg = &order.Submit{
			Pair: currency.Pair{
				Base:  currency.LTC,
				Quote: currency.BTC,
			},
			Exchange:  e.Name,
			Side:      order.Sell,
			Type:      order.Limit,
			Price:     120000,
			Amount:    1000000000,
			ClientID:  "yeneOrder",
			AssetType: asset.Spot,
		}
		result, err := e.SubmitOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.Trigger
		arg.TriggerPrice = 11999
		arg.TriggerPriceType = order.LastPrice
		result, err = e.SubmitOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.ConditionalStop
		arg.TriggerPrice = 11999
		arg.TriggerPriceType = order.IndexPrice
		result, err = e.SubmitOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.Chase
		arg.TrackingMode = order.Percentage
		arg.TrackingValue = .5
		_, err = e.SubmitOrder(contextGenerate(), arg)
		assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "a spot chase order should be refused, as OKX places chase orders on futures and perpetual swaps only")

		arg.Type = order.TWAP
		result, err = e.SubmitOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.TrailingStop
		result, err = e.SubmitOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.OCO
		_, err = e.SubmitOrder(contextGenerate(), arg)
		require.ErrorIs(t, err, limits.ErrPriceBelowMin)

		arg.RiskManagementModes = order.RiskManagementModes{
			TakeProfit: order.RiskManagement{
				Price:      11999,
				LimitPrice: 12000,
			},
			StopLoss: order.RiskManagement{
				Price:      10999,
				LimitPrice: 11000,
			},
		}
		result, err = e.SubmitOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		cp, err := currency.NewPairFromString("BTC-USDT-230630")
		require.NoError(t, err)

		arg = &order.Submit{
			Pair:       cp,
			Exchange:   e.Name,
			Side:       order.Long,
			Type:       order.Market,
			Amount:     1,
			ClientID:   "hellomoto",
			AssetType:  asset.Futures,
			MarginType: margin.Multi,
		}
		result, err = e.SubmitOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		pair, err := currency.NewPairFromString("BTC-USDT-SWAP_BTC-USDT-250328")
		require.NoError(t, err)

		result, err = e.SubmitOrder(contextGenerate(), &order.Submit{
			Pair:       pair,
			Exchange:   e.Name,
			Side:       order.Sell,
			Type:       order.Limit,
			Price:      120000,
			Amount:     1,
			ClientID:   "hellomoto",
			AssetType:  asset.Spread,
			MarginType: margin.Multi,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	// guards the OCO submit path: the stop loss leg
	// triggers at the stop loss price, not the take profit price.
	t.Run("oco stop-loss trigger price", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/order-algo" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the algo order request body should not error")
			mu.Lock()
			body = b
			mu.Unlock()
			writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0", "sMsg": ""})
		}))
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{
			Exchange:   ex.Name,
			Pair:       mainPair,
			AssetType:  asset.Spot,
			Side:       order.Sell,
			Type:       order.OCO,
			Amount:     1,
			MarginType: margin.NoMargin,
			RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110},
				StopLoss:   order.RiskManagement{Price: 90},
			},
		})
		require.NoError(t, err, "SubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		var sent PlaceAlgoOrderRequest
		require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
		assert.Equal(t, 110.0, sent.TakeProfitTriggerPrice, "the take profit leg should trigger at the take profit price")
		assert.Equal(t, 90.0, sent.StopLossTriggerPrice, "the stop loss leg should trigger at the stop loss price")
	})
	// guards the ordType a limit order
	// sends for each time in force: post_only, fok and ioc must map to their
	// documented placement ordTypes instead of a plain resting limit.
	t.Run("limit time in forces request documented ordTypes", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			orderType order.Type
			tif       order.TimeInForce
			exp       string
		}{
			{"plain limit", asset.Spot, order.Limit, order.UnknownTIF, orderLimit},
			{"post only", asset.Spot, order.Limit, order.PostOnly, orderPostOnly},
			{"good till cancel post only", asset.Spot, order.Limit, order.GoodTillCancel | order.PostOnly, orderPostOnly},
			{"fill or kill", asset.Spot, order.Limit, order.FillOrKill, orderFOK},
			{"immediate or cancel", asset.Spot, order.Limit, order.ImmediateOrCancel, orderIOC},
			// OKX takes MMP orders for options only.
			{"market maker protection", asset.Options, order.MarketMakerProtection, order.UnknownTIF, orderMarketMakerProtection},
			{"market maker protection post only", asset.Options, order.MarketMakerProtection, order.PostOnly, orderMarketMakerProtectionAndPostOnly},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/account/config":
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
						return
					case "/trade/order":
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the place order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange:    ex.Name,
					Pair:        mainPair,
					AssetType:   tc.assetType,
					Side:        order.Buy,
					Type:        tc.orderType,
					TimeInForce: tc.tif,
					Amount:      1,
					Price:       1,
				})
				require.NoError(t, err, "SubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
				assert.Equal(t, tc.exp, sent["ordType"], "the submitted limit order should send the documented ordType")
				assert.Equal(t, "1", sent["px"], "the submitted order should send its price")
			})
		}
	})
	// guards the contract order
	// body: OKX requires side for every instrument, and reduceOnly applies to
	// MARGIN and FUTURES/SWAP orders only.
	t.Run("contract sends side and reduce only", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name          string
			assetType     asset.Item
			side          order.Side
			reduceOnly    bool
			expSide       string
			expPosSide    any // nil when the field must be absent
			expReduceOnly any // nil when the field must be absent
		}{
			{"perpetual swap reduce only sell", asset.PerpetualSwap, order.Sell, true, "sell", positionSideNet, true},
			{"futures long", asset.Futures, order.Long, false, "buy", positionSideNet, nil},
			{"margin reduce only", asset.Margin, order.Sell, true, "sell", nil, true},
			{"options send side", asset.Options, order.Buy, false, "buy", nil, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "3", "posMode": positionModeNet}})
						return
					}
					if r.URL.Path != "/trade/order" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the place order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange:    ex.Name,
					Pair:        mainPair,
					AssetType:   tc.assetType,
					Side:        tc.side,
					Type:        order.Limit,
					TimeInForce: order.GoodTillCancel,
					Amount:      1,
					Price:       1,
					MarginType:  margin.Multi,
					ReduceOnly:  tc.reduceOnly,
				})
				require.NoError(t, err, "SubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
				assert.Equal(t, tc.expSide, sent["side"], "the order should send OKX's required side")
				assert.Equal(t, tc.expPosSide, sent["posSide"], "the order should send the position side")
				assert.Equal(t, tc.expReduceOnly, sent["reduceOnly"], "reduceOnly should match the documented applicability")
			})
		}
	})
	// verifies that algo
	// order placement suppresses reduceOnly for FUTURES/SWAP in long/short mode,
	// where OKX ignores the flag and the position side carries the close intent.
	t.Run("algo contract guards reduce only by position mode", func(t *testing.T) {
		t.Parallel()
		// Each algo type needs its own fields to place.
		algoExtras := map[order.Type]func(s *order.Submit){
			order.Trigger: func(s *order.Submit) {
				s.TriggerPrice = 100
				s.Price = -1
			},
			order.ConditionalStop: func(s *order.Submit) {
				s.TriggerPrice = 100
				s.Price = -1
			},
			order.Chase: func(s *order.Submit) {
				s.TrackingMode = order.Percentage
				s.TrackingValue = 0.1
			},
			order.TrailingStop: func(s *order.Submit) {
				s.TrackingMode = order.Percentage
				s.TrackingValue = 0.05
			},
			order.TWAP: func(s *order.Submit) {
				s.TrackingMode = order.Percentage
				s.TrackingValue = 0.01
				s.Price = 100
			},
			order.OCO: func(s *order.Submit) {
				s.RiskManagementModes.TakeProfit.Price = 110
				s.RiskManagementModes.TakeProfit.LimitPrice = -1
				s.RiskManagementModes.StopLoss.Price = 90
				s.RiskManagementModes.StopLoss.LimitPrice = -1
			},
		}
		for _, tc := range []struct {
			name          string
			orderType     order.Type
			posMode       string
			assetType     asset.Item
			pair          currency.Pair
			expReduceOnly any // nil when the field must be absent
		}{
			{"net mode passes reduceOnly", order.Trigger, positionModeNet, asset.PerpetualSwap, mainPair, true},
			{"long/short mode suppresses reduceOnly", order.Trigger, positionModeLongShort, asset.PerpetualSwap, mainPair, nil},
			{"margin passes reduceOnly regardless of mode", order.Trigger, positionModeLongShort, asset.Margin, mainPair, true},
			{"long/short mode suppresses reduceOnly", order.ConditionalStop, positionModeLongShort, asset.PerpetualSwap, perpetualSwapPair, nil},
			{"long/short mode suppresses reduceOnly", order.Chase, positionModeLongShort, asset.PerpetualSwap, perpetualSwapPair, nil},
			{"long/short mode suppresses reduceOnly", order.TrailingStop, positionModeLongShort, asset.PerpetualSwap, perpetualSwapPair, nil},
			{"long/short mode suppresses reduceOnly", order.TWAP, positionModeLongShort, asset.PerpetualSwap, perpetualSwapPair, nil},
			{"long/short mode suppresses reduceOnly", order.OCO, positionModeLongShort, asset.PerpetualSwap, perpetualSwapPair, nil},
		} {
			t.Run(tc.orderType.String()+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "3", "posMode": tc.posMode}})
						return
					}
					if r.URL.Path != "/trade/order-algo" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the algo order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0"})
				}))
				sub := &order.Submit{
					Exchange:   ex.Name,
					Pair:       tc.pair,
					AssetType:  tc.assetType,
					Side:       order.Sell,
					Type:       tc.orderType,
					Amount:     1,
					MarginType: margin.Multi,
					ReduceOnly: true,
				}
				algoExtras[tc.orderType](sub)
				_, err := ex.SubmitOrder(t.Context(), sub)
				require.NoError(t, err, "SubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
				assert.Equal(t, tc.expReduceOnly, sent["reduceOnly"], "reduceOnly should follow the position mode guard")
			})
		}
	})
	// guards the placement of perpetual
	// swap orders against the account's position mode: net mode pairs reduceOnly
	// with posSide net, and long/short mode pairs the side with the position
	// side so that a reduce-only sell closes the long position.
	t.Run("perpetual swap position mode follows account config", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name          string
			mode          string
			side          order.Side
			reduceOnly    bool
			expPosSide    string
			expReduceOnly any // nil when the field must be absent
		}{
			{"net mode open long", positionModeNet, order.Buy, false, positionSideNet, nil},
			{"net mode close long", positionModeNet, order.Sell, true, positionSideNet, true},
			{"long short open long", positionModeLongShort, order.Buy, false, positionSideLong, nil},
			{"long short open short", positionModeLongShort, order.Sell, false, positionSideShort, nil},
			{"long short close long", positionModeLongShort, order.Sell, true, positionSideLong, nil},
			{"long short close short", positionModeLongShort, order.Buy, true, positionSideShort, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"posMode": tc.mode}})
						return
					}
					if r.URL.Path != "/trade/order" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the place order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange:   ex.Name,
					Pair:       mainPair,
					AssetType:  asset.PerpetualSwap,
					Side:       tc.side,
					Type:       order.Limit,
					Amount:     1,
					Price:      1,
					MarginType: margin.Multi,
					ReduceOnly: tc.reduceOnly,
				})
				require.NoError(t, err, "SubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
				assert.Equal(t, tc.expPosSide, sent["posSide"], "the order should send the position side the account's mode requires")
				assert.Equal(t, tc.expReduceOnly, sent["reduceOnly"], "reduceOnly should follow the position mode")
			})
		}
		t.Run("unknown mode", func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					writeOKXData(t, w, []map[string]string{{"posMode": "weird_mode"}})
					return
				}
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			_, err := ex.SubmitOrder(t.Context(), &order.Submit{
				Exchange:   ex.Name,
				Pair:       mainPair,
				AssetType:  asset.PerpetualSwap,
				Side:       order.Buy,
				Type:       order.Limit,
				Amount:     1,
				Price:      1,
				MarginType: margin.Multi,
			})
			require.ErrorIs(t, err, errInvalidPositionMode, "an undocumented position mode must stop the submit")
		})
		t.Run("fetch failure", func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					http.Error(w, "account config unavailable", http.StatusInternalServerError)
					return
				}
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			_, err := ex.SubmitOrder(t.Context(), &order.Submit{
				Exchange:   ex.Name,
				Pair:       mainPair,
				AssetType:  asset.PerpetualSwap,
				Side:       order.Buy,
				Type:       order.Limit,
				Amount:     1,
				Price:      1,
				MarginType: margin.Multi,
			})
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "a failed position mode fetch must stop the submit")
		})
		t.Run("empty reply", func(t *testing.T) {
			t.Parallel()
			for _, data := range []any{nil, []any{nil}} {
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, data)
						return
					}
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange:   ex.Name,
					Pair:       mainPair,
					AssetType:  asset.PerpetualSwap,
					Side:       order.Buy,
					Type:       order.Limit,
					Amount:     1,
					Price:      1,
					MarginType: margin.Multi,
				})
				require.ErrorIsf(t, err, common.ErrNoResponse, "a %v account configuration reply must stop the submit", data)
			}
		})
	})
	// guards the tdMode default: OKX requires tdMode on every order, and the
	// zero margin type takes the trade mode OKX documents for the account's
	// mode: spot orders and option buyers trade in cash in Spot mode, spot
	// orders in cash and the rest in cross in Futures mode, and every order in
	// cross in the margin modes. An explicit margin type is sent as it is, but
	// a margin order cannot trade in a non-margin mode.
	t.Run("defaults the trade mode", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name         string
			accountLevel string
			assetType    asset.Item
			side         order.Side
			marginType   margin.Type
			exp          string
			expErr       error
		}{
			{name: "spot defaults to cross in multi-currency margin mode", accountLevel: "3", assetType: asset.Spot, exp: TradeModeCross},
			{name: "margin defaults to cross", accountLevel: "3", assetType: asset.Margin, exp: TradeModeCross},
			{name: "futures defaults to cross", accountLevel: "3", assetType: asset.Futures, exp: TradeModeCross},
			{name: "perpetual swap defaults to cross", accountLevel: "3", assetType: asset.PerpetualSwap, exp: TradeModeCross},
			{name: "options defaults to cross", accountLevel: "3", assetType: asset.Options, exp: TradeModeCross},
			{name: "spot defaults to cash in spot mode", accountLevel: "1", assetType: asset.Spot, exp: TradeModeCash},
			{name: "option buyer defaults to cash in spot mode", accountLevel: "1", assetType: asset.Options, exp: TradeModeCash},
			{name: "option seller defaults to cash in spot mode", accountLevel: "1", assetType: asset.Options, side: order.Sell, exp: TradeModeCash},
			{name: "spot defaults to cash in futures mode", accountLevel: "2", assetType: asset.Spot, exp: TradeModeCash},
			{name: "options defaults to cross in futures mode", accountLevel: "2", assetType: asset.Options, exp: TradeModeCross},
			{name: "margin defaults to cross in futures mode", accountLevel: "2", assetType: asset.Margin, exp: TradeModeCross},
			{name: "perpetual swap defaults to cross in futures mode", accountLevel: "2", assetType: asset.PerpetualSwap, exp: TradeModeCross},
			{name: "spot defaults to cross in portfolio margin mode", accountLevel: "4", assetType: asset.Spot, exp: TradeModeCross},
			{name: "options defaults to cross in portfolio margin mode", accountLevel: "4", assetType: asset.Options, exp: TradeModeCross},
			{name: "explicit cash spot in multi-currency margin mode", accountLevel: "3", assetType: asset.Spot, marginType: margin.NoMargin, exp: TradeModeCash},
			{name: "explicit cross option in spot mode", accountLevel: "1", assetType: asset.Options, marginType: margin.Multi, exp: TradeModeCross},
			{name: "explicit isolated margin in multi-currency margin mode", accountLevel: "3", assetType: asset.Margin, marginType: margin.Isolated, exp: TradeModeIsolated},
			{name: "margin in cash mode", accountLevel: "3", assetType: asset.Margin, marginType: margin.NoMargin, expErr: margin.ErrMarginTypeUnsupported},
			{name: "margin in spot lead trading mode", accountLevel: "3", assetType: asset.Margin, marginType: margin.SpotIsolated, expErr: margin.ErrMarginTypeUnsupported},
			{name: "spot in an undocumented account mode", accountLevel: "5", assetType: asset.Spot, expErr: errInvalidAccountLevel},
			{name: "option without an account mode", accountLevel: "", assetType: asset.Options, expErr: errInvalidAccountLevel},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": tc.accountLevel, "posMode": positionModeNet}})
						return
					}
					if r.URL.Path != "/trade/order" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the place order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
				}))
				side := order.Buy
				if tc.side != order.UnknownSide {
					side = tc.side
				}
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange:   ex.Name,
					Pair:       mainPair,
					AssetType:  tc.assetType,
					Side:       side,
					Type:       order.Limit,
					Amount:     1,
					Price:      1,
					MarginType: tc.marginType,
				})
				mu.Lock()
				defer mu.Unlock()
				if tc.expErr != nil {
					require.ErrorIs(t, err, tc.expErr, "SubmitOrder must refuse a trade mode it cannot place")
					assert.Nil(t, body, "a refused order should not reach OKX")
					return
				}
				require.NoError(t, err, "SubmitOrder must not error")
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
				assert.Equal(t, tc.exp, sent["tdMode"], "the order should send the documented default trade mode")
			})
		}
		t.Run("looks the account level up once per account", func(t *testing.T) {
			t.Parallel()
			var configFetches atomic.Int32
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/account/config" {
					configFetches.Add(1)
					writeOKXData(t, w, []map[string]string{{"acctLv": "3", "posMode": positionModeNet}})
					return
				}
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			submit := &order.Submit{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}
			for range 2 {
				_, err := ex.SubmitOrder(t.Context(), submit)
				require.NoError(t, err, "SubmitOrder must not error")
			}
			assert.Equal(t, int32(1), configFetches.Load(), "the account level should be fetched once for the exchange's own account")
			_, err := ex.SubmitOrder(otherAccountContext(t), submit)
			require.NoError(t, err, "SubmitOrder must not error for another account")
			assert.Equal(t, int32(2), configFetches.Load(), "another account's level should be fetched for it")
		})
	})
	// guards the market order body: px is only
	// applicable to the limit-style order types, so a market order carries no
	// price even when the submit set one.
	t.Run("market omits the price", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
				return
			}
			if r.URL.Path != "/trade/order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the place order body should not error")
			mu.Lock()
			body = b
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
		}))
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Side:      order.Buy,
			Type:      order.Market,
			Amount:    1,
			Price:     999,
		})
		require.NoError(t, err, "SubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		var sent map[string]any
		require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
		_, has := sent["px"]
		assert.False(t, has, "a market order should not send a price")
	})
	// guards the submit price check: an order type that
	// sends px is refused without a price before any request transmits, while
	// market and optimal_limit_ioc orders, which OKX places with no px, still go
	// out, the latter only on the futures and perpetual swaps OKX offers it on.
	t.Run("guards the price", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			pair      currency.Pair
			orderType order.Type
			price     float64
			expErr    error
			expType   string // the ordType sent when the order is placed
		}{
			{"limit", asset.Spot, mainPair, order.Limit, 0, limits.ErrPriceBelowMin, ""},
			{"limit with a negative price", asset.Spot, mainPair, order.Limit, -1, limits.ErrPriceBelowMin, ""},
			{"limit maker", asset.Spot, mainPair, order.LimitMaker, 0, limits.ErrPriceBelowMin, ""},
			{"market maker protection", asset.Options, optionsPair, order.MarketMakerProtection, 0, limits.ErrPriceBelowMin, ""},
			{"optimal limit on spot", asset.Spot, mainPair, order.OptimalLimit, 0, order.ErrUnsupportedOrderType, ""},
			{"market", asset.Spot, mainPair, order.Market, 0, nil, orderMarket},
			{"optimal limit", asset.PerpetualSwap, perpetualSwapPair, order.OptimalLimit, 0, nil, orderOptimalLimitIOC},
			{"optimal limit on futures", asset.Futures, futuresPair, order.OptimalLimit, 0, nil, orderOptimalLimitIOC},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/account/config":
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
					case "/trade/order":
						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err, "reading the place order body should not error")
						mu.Lock()
						body = b
						mu.Unlock()
						writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange:  ex.Name,
					Pair:      tc.pair,
					AssetType: tc.assetType,
					Side:      order.Buy,
					Type:      tc.orderType,
					Amount:    1,
					Price:     tc.price,
				})
				mu.Lock()
				defer mu.Unlock()
				if tc.expErr != nil {
					require.ErrorIs(t, err, tc.expErr, "an order OKX cannot place as given must be refused")
					assert.Nil(t, body, "a refused order should not reach OKX")
					return
				}
				require.NoError(t, err, "an order type OKX places with no px must not need a price")
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the place order request body must decode")
				assert.Equal(t, tc.expType, sent["ordType"], "the order should send its documented ordType")
				assert.NotContains(t, sent, "px", "the order should not send a price")
			})
		}
	})
	// guards the trigger order body: OKX
	// requires orderPx, which the wrapper must take from the submitted price
	// instead of dropping it.
	t.Run("trigger sends the order price", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
				return
			}
			if r.URL.Path != "/trade/order-algo" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the algo order body should not error")
			mu.Lock()
			body = b
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"algoId": "1", "sCode": "0"}})
		}))
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{
			Exchange:     ex.Name,
			Pair:         mainPair,
			AssetType:    asset.PerpetualSwap,
			Side:         order.Long,
			Type:         order.Trigger,
			Amount:       1,
			Price:        100,
			TriggerPrice: 110,
			MarginType:   margin.Multi,
		})
		require.NoError(t, err, "SubmitOrder must not error")
		// Both submits run before the body is read under the mutex, so a second
		// request cannot deadlock against the handler's own lock.
		_, err = ex.SubmitOrder(t.Context(), &order.Submit{
			Exchange:     ex.Name,
			Pair:         mainPair,
			AssetType:    asset.PerpetualSwap,
			Side:         order.Long,
			Type:         order.Trigger,
			Amount:       1,
			TriggerPrice: 110,
			MarginType:   margin.Multi,
		})
		assert.ErrorIs(t, err, limits.ErrPriceBelowMin, "a trigger order without an order price should be rejected before sending")

		mu.Lock()
		defer mu.Unlock()
		var sent map[string]any
		require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
		assert.Equal(t, "100", sent["orderPx"], "the trigger order should carry the submitted order price")
		assert.Equal(t, "buy", sent["side"], "the trigger order should send OKX's required side")
		assert.Equal(t, positionSideNet, sent["posSide"], "the trigger order should send the position side the account's mode requires")
	})
	// guards the TWAP interval transport:
	// OKX documents the field as timeInterval carrying seconds, not interval
	// carrying a duration code.
	t.Run("twap sends the time interval", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/order-algo" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the algo order body should not error")
			mu.Lock()
			body = b
			mu.Unlock()
			writeOKXData(t, w, []map[string]string{{"algoId": "1", "sCode": "0"}})
		}))
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{
			Exchange:      ex.Name,
			Pair:          mainPair,
			AssetType:     asset.Spot,
			Side:          order.Buy,
			Type:          order.TWAP,
			Amount:        1,
			Price:         100,
			TrackingMode:  order.Percentage,
			TrackingValue: 0.5,
			MarginType:    margin.NoMargin,
		})
		require.NoError(t, err, "SubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		var sent map[string]any
		require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
		assert.Equal(t, "900", sent["timeInterval"], "the TWAP order should send its interval as documented seconds")
		_, has := sent["interval"]
		assert.False(t, has, "the TWAP order should not send the undocumented interval field")
	})
	// guards the chase order mapping: the tracking value is the order's
	// distance from the best bid or ask, OKX's chaseType and chaseVal, which
	// name a ratio rather than the percentage the tracking mode string carries.
	// A zero distance tracks the best price itself, and with no tracking mode
	// OKX's defaults do the same. The maximum chase is an auto-cancel bound, not
	// the distance, and OKX places chase orders on futures and perpetual swaps
	// only.
	t.Run("chase sends its tracking distance as the chase type and value", func(t *testing.T) {
		t.Parallel()
		futuresContract := currency.NewPairWithDelimiter("BTC", "USD-261225", currency.DashDelimiter)
		for _, tc := range []struct {
			name          string
			assetType     asset.Item
			pair          currency.Pair
			trackingMode  order.TrackingMode
			trackingValue float64
			expRequest    string
			expErr        error
		}{
			{
				name: "percentage maps to ratio", assetType: asset.PerpetualSwap, pair: mainPair, trackingMode: order.Percentage, trackingValue: 0.5,
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","posSide":"net","ordType":"chase","sz":"1","chaseType":"ratio","chaseVal":"0.5"}`,
			},
			{
				name: "distance maps to distance", assetType: asset.PerpetualSwap, pair: mainPair, trackingMode: order.Distance, trackingValue: 3,
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","posSide":"net","ordType":"chase","sz":"1","chaseType":"distance","chaseVal":"3"}`,
			},
			{
				name: "a zero distance tracks the best price", assetType: asset.PerpetualSwap, pair: perpetualSwapPair, trackingMode: order.Distance,
				expRequest: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","posSide":"net","ordType":"chase","sz":"1","chaseType":"distance"}`,
			},
			{
				name: "no tracking mode takes OKX's default", assetType: asset.PerpetualSwap, pair: perpetualSwapPair,
				expRequest: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","posSide":"net","ordType":"chase","sz":"1"}`,
			},
			{
				name: "futures", assetType: asset.Futures, pair: futuresContract, trackingMode: order.Percentage, trackingValue: 0.01,
				expRequest: `{"instId":"BTC-USD-261225","tdMode":"cross","side":"buy","posSide":"net","ordType":"chase","sz":"1","chaseType":"ratio","chaseVal":"0.01"}`,
			},
			{name: "a tracking value without a tracking mode", assetType: asset.PerpetualSwap, pair: perpetualSwapPair, trackingValue: 3, expErr: order.ErrUnknownTrackingMode},
			{name: "an undefined tracking mode", assetType: asset.PerpetualSwap, pair: perpetualSwapPair, trackingMode: order.TrackingMode(99), expErr: order.ErrUnknownTrackingMode},
			{name: "a negative distance", assetType: asset.PerpetualSwap, pair: perpetualSwapPair, trackingMode: order.Distance, trackingValue: -1, expErr: limits.ErrAmountBelowMin},
			{name: "spot", assetType: asset.Spot, pair: mainPair, trackingMode: order.Distance, trackingValue: 3, expErr: order.ErrUnsupportedOrderType},
			{name: "margin", assetType: asset.Margin, pair: mainPair, trackingMode: order.Distance, trackingValue: 3, expErr: order.ErrUnsupportedOrderType},
			{name: "options", assetType: asset.Options, pair: currency.NewPairWithDelimiter("BTC", "USD-261225-100000-C", currency.DashDelimiter), trackingMode: order.Distance, trackingValue: 3, expErr: order.ErrUnsupportedOrderType},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
						return
					}
					if r.URL.Path != "/trade/order-algo" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the algo order body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"algoId": "1", "sCode": "0"}})
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange:      ex.Name,
					Pair:          tc.pair,
					AssetType:     tc.assetType,
					Side:          order.Long,
					Type:          order.Chase,
					Amount:        1,
					Price:         100,
					TrackingMode:  tc.trackingMode,
					TrackingValue: tc.trackingValue,
					MarginType:    margin.Multi,
				})
				mu.Lock()
				defer mu.Unlock()
				if tc.expErr != nil {
					require.ErrorIs(t, err, tc.expErr, "SubmitOrder must refuse a chase order OKX cannot place as given")
					assert.Nil(t, body, "a refused chase order should not reach OKX")
					return
				}
				require.NoError(t, err, "SubmitOrder must not error")
				assert.JSONEq(t, tc.expRequest, string(body), "the chase order should send its distance from the best price as its chase type and value")
			})
		}
	})
	// guards the position side of
	// futures and perpetual swap orders, plain and algo: OKX's position mode
	// covers FUTURES and SWAP alike, and in long/short mode a close is expressed
	// by the position side, since reduceOnly does not apply there.
	t.Run("contracts follow the position mode", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.Futures, asset.PerpetualSwap} {
			for _, oType := range []order.Type{order.Limit, order.ConditionalStop} {
				for _, tc := range []struct {
					mode          string
					side          order.Side
					reduceOnly    bool
					expPosSide    string
					expReduceOnly any // nil when the field must be absent
				}{
					{positionModeNet, order.Buy, false, positionSideNet, nil},
					{positionModeNet, order.Sell, false, positionSideNet, nil},
					{positionModeNet, order.Sell, true, positionSideNet, true},
					{positionModeNet, order.Buy, true, positionSideNet, true},
					{positionModeLongShort, order.Buy, false, positionSideLong, nil},
					{positionModeLongShort, order.Sell, false, positionSideShort, nil},
					{positionModeLongShort, order.Sell, true, positionSideLong, nil},
					{positionModeLongShort, order.Buy, true, positionSideShort, nil},
				} {
					t.Run(fmt.Sprintf("%s %s %s %s reduce only %t", a, oType, tc.mode, tc.side, tc.reduceOnly), func(t *testing.T) {
						t.Parallel()
						var mu sync.Mutex
						var body []byte
						ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							switch r.URL.Path {
							case "/account/config":
								writeOKXData(t, w, []map[string]string{{"posMode": tc.mode}})
								return
							case "/trade/order", "/trade/order-algo":
							default:
								t.Errorf("unexpected request path %s", r.URL.Path)
								http.NotFound(w, r)
								return
							}
							b, err := io.ReadAll(r.Body)
							assert.NoError(t, err, "reading the order body should not error")
							mu.Lock()
							body = b
							mu.Unlock()
							writeOKXData(t, w, []map[string]string{{"ordId": "1", "algoId": "1", "sCode": "0"}})
						}))
						_, err := ex.SubmitOrder(t.Context(), &order.Submit{
							Exchange:     ex.Name,
							Pair:         mainPair,
							AssetType:    a,
							Side:         tc.side,
							Type:         oType,
							Amount:       1,
							Price:        1,
							TriggerPrice: 2,
							MarginType:   margin.Multi,
							ReduceOnly:   tc.reduceOnly,
						})
						require.NoError(t, err, "SubmitOrder must not error")
						mu.Lock()
						defer mu.Unlock()
						var sent map[string]any
						require.NoError(t, json.Unmarshal(body, &sent), "the order request body must decode")
						assert.Equal(t, tc.expPosSide, sent["posSide"], "the order should send the position side the account's mode requires")
						if oType == order.Limit {
							assert.Equal(t, tc.expReduceOnly, sent["reduceOnly"], "reduceOnly should follow the position mode")
						}
					})
				}
			}
		}
	})
	// guards the trade mode default:
	// only the zero margin type defaults, so an explicit margin type OKX cannot
	// express fails before any request instead of trading in another mode.
	t.Run("rejects an unsupported margin type", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
			for _, mt := range []margin.Type{
				margin.Unknown,
				margin.Isolated | margin.Multi,
				margin.Isolated | margin.NoMargin,
				margin.Isolated | margin.SpotIsolated,
				margin.NoMargin | margin.SpotIsolated,
				margin.Isolated | margin.NoMargin | margin.SpotIsolated,
			} {
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: mainPair, AssetType: a, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: mt})
				assert.ErrorIsf(t, err, margin.ErrMarginTypeUnsupported, "SubmitOrder should reject margin type %d for %s", mt, a)
			}
		}
	})
	// guards every placement path against a
	// success reply without a usable row: data null or a null row leaves the
	// order's outcome unknown, so placement must report it rather than panic on
	// the missing response.
	t.Run("rejects null reply rows", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			path   string
			data   any
			submit *order.Submit
		}{
			{
				name: "spot reply with data null",
				path: "/trade/order",
				data: nil,
				submit: &order.Submit{
					Pair: mainPair, AssetType: asset.Spot,
					Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1,
				},
			},
			{
				name: "spot reply with a null row",
				path: "/trade/order",
				data: []any{nil},
				submit: &order.Submit{
					Pair: mainPair, AssetType: asset.Spot,
					Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1,
				},
			},
			{
				name: "spread reply with data null",
				path: "/sprd/order",
				data: nil,
				submit: &order.Submit{
					Pair: spreadPair, AssetType: asset.Spread,
					Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2,
				},
			},
			{
				name: "algo reply with a null row",
				path: "/trade/order-algo",
				data: []any{nil},
				submit: &order.Submit{
					Pair: mainPair, AssetType: asset.Spot,
					Side: order.Buy, Type: order.Trigger, Amount: 1, Price: 1, TriggerPrice: 1,
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
						return
					}
					if r.URL.Path != tc.path {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					writeOKXData(t, w, tc.data)
				}))
				var err error
				require.NotPanics(t, func() {
					_, err = ex.SubmitOrder(t.Context(), tc.submit)
				}, "SubmitOrder must not panic on a success reply without a row")
				require.ErrorIs(t, err, common.ErrInvalidResponse, "SubmitOrder must reject a success reply without a row")
			})
		}
	})
	t.Run("rejects a reply row which confirms no placement", func(t *testing.T) {
		t.Parallel()
		const insufficient = "Order failed. Insufficient USDT balance in account"
		limit := &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ClientOrderID: "clientTwo"}
		spread := &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2, ClientOrderID: "clientTwo"}
		conditional := &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.ConditionalStop, Amount: 1, Price: 100, TriggerPrice: 110, ClientOrderID: "clientTwo"}
		for _, tc := range []struct {
			name    string
			path    string
			submit  *order.Submit
			data    any
			expErr  error  // nil when expText names the rejected row's status message
			expText string // the rejected row's status message
		}{
			{name: "rejected row with an order ID", path: "/trade/order", submit: limit, data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}}, expText: insufficient},
			{name: "row without a status code", path: "/trade/order", submit: limit, data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo"}}, expErr: errMissingStatusCode},
			{name: "spread rejected row with an order ID", path: "/sprd/order", submit: spread, data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}}, expText: insufficient},
			{name: "spread row without a status code", path: "/sprd/order", submit: spread, data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo"}}, expErr: errMissingStatusCode},
			{name: "algo rejected row with an algo ID", path: "/trade/order-algo", submit: conditional, data: []map[string]string{{"algoId": "1002", "algoClOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}}, expText: insufficient},
			{name: "algo row without a status code", path: "/trade/order-algo", submit: conditional, data: []map[string]string{{"algoId": "1002", "algoClOrdId": "clientTwo"}}, expErr: errMissingStatusCode},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := newRecordingExchange(t, func(string) any { return tc.data })
				submit := *tc.submit
				submit.Exchange = ex.Name
				resp, err := ex.SubmitOrder(t.Context(), &submit)
				if tc.expErr != nil {
					assert.ErrorIs(t, err, tc.expErr, "SubmitOrder should return the sentinel for a row which confirms no placement")
				} else {
					assert.ErrorContains(t, err, tc.expText, "SubmitOrder should report the rejected row's status message")
				}
				assert.Nil(t, resp, "SubmitOrder should give no response for an order the reply does not confirm placed")
				reqs := requests()
				require.Len(t, reqs, 1, "one placement must be sent")
				assert.Equal(t, tc.path, reqs[0].path, "the placement should reach its endpoint")
			})
		}
	})
	// guards OCO placement reading
	// each leg's own trigger price type: OKX takes independent tpTriggerPxType
	// and slTriggerPxType parameters, and ModifyOrder already amends each leg by
	// its own type, so placement must agree with it.
	t.Run("oco per-leg trigger price types", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			modes order.RiskManagementModes
			expTP string
			expSL string
		}{
			{
				name: "per leg types",
				modes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 110, TriggerPriceType: order.IndexPrice},
					StopLoss:   order.RiskManagement{Price: 90, TriggerPriceType: order.MarkPrice},
				},
				expTP: "index",
				expSL: "mark",
			},
			{
				name: "unset legs default to last",
				modes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 110},
					StopLoss:   order.RiskManagement{Price: 90},
				},
				expTP: "last",
				expSL: "last",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/order-algo" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the algo order request body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0", "sMsg": ""})
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Pair: mainPair, AssetType: asset.Spot, Side: order.Sell,
					Type: order.OCO, Amount: 1, MarginType: margin.NoMargin,
					RiskManagementModes: tc.modes,
				})
				require.NoError(t, err, "SubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent PlaceAlgoOrderRequest
				require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
				assert.Equal(t, tc.expTP, sent.TakeProfitTriggerPriceType, "the take profit leg should use its own trigger price type")
				assert.Equal(t, tc.expSL, sent.StopLossTriggerPriceType, "the stop loss leg should use its own trigger price type")
			})
		}
	})
	// guards the algo placement round trip:
	// ModifyOrder amends and the detail queries look up by algoClOrdId, so
	// placement must register the client's order ID, while a submit without one
	// keeps the field off the wire.
	t.Run("algo sends the client order ID", func(t *testing.T) {
		t.Parallel()
		extras := func(t order.Type) *order.Submit {
			s := &order.Submit{
				Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Amount: 1,
				Type: t, MarginType: margin.NoMargin, ClientOrderID: "my-algo-cl-id",
			}
			switch t {
			case order.Trigger:
				s.TriggerPrice = 100
				s.Price = 100
			case order.ConditionalStop:
				s.TriggerPrice = 90
				s.Price = 90
			case order.Chase:
				// OKX places chase orders on futures and perpetual swaps only
				s.Pair, s.AssetType, s.MarginType = perpetualSwapPair, asset.PerpetualSwap, margin.Multi
				s.TrackingMode = order.Distance
				s.TrackingValue = 1
			case order.TrailingStop:
				s.TrackingMode = order.Distance
				s.TrackingValue = 1
				// OKX sizes a spot trailing stop buy in the quote currency
				s.QuoteAmount = 100
			case order.TWAP:
				s.TrackingMode = order.Distance
				s.TrackingValue = 1
				s.Price = 100
			case order.OCO:
				s.RiskManagementModes = order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 110},
					StopLoss:   order.RiskManagement{Price: 90},
				}
				// OKX sizes a spot buy whose legs execute at market in the quote currency
				s.QuoteAmount = 100
			}
			return s
		}
		for _, oType := range []order.Type{order.Trigger, order.ConditionalStop, order.Chase, order.TrailingStop, order.TWAP, order.OCO} {
			t.Run(oType.String(), func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
						return
					}
					if r.URL.Path != "/trade/order-algo" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the algo order request body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0", "sMsg": ""})
				}))
				_, err := ex.SubmitOrder(t.Context(), extras(oType))
				require.NoError(t, err, "SubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent PlaceAlgoOrderRequest
				require.NoError(t, json.Unmarshal(body, &sent), "the algo order request body must decode")
				assert.Equal(t, "my-algo-cl-id", sent.AlgoClientOrderID, "algo placement should register the client's order ID as algoClOrdId")
			})
		}
		t.Run("without a client order ID", func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/trade/order-algo" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the algo order request body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0", "sMsg": ""})
			}))
			s := extras(order.Trigger)
			s.ClientOrderID = ""
			_, err := ex.SubmitOrder(t.Context(), s)
			require.NoError(t, err, "SubmitOrder must not error")
			mu.Lock()
			defer mu.Unlock()
			var raw map[string]any
			require.NoError(t, json.Unmarshal(body, &raw), "the algo order request body must decode")
			assert.NotContains(t, raw, "algoClOrdId", "a submit without a client order ID should keep algoClOrdId off the wire")
		})
	})
	t.Run("spread orders", func(t *testing.T) {
		t.Parallel()

		t.Run("market price and type", func(t *testing.T) {
			t.Parallel()
			for _, tc := range []struct {
				name       string
				orderType  order.Type
				tif        order.TimeInForce
				price      float64
				expOrdType string
				expPX      string // empty when no px is sent
			}{
				{"market carries no price", order.Market, order.UnknownTIF, 0, orderMarket, ""},
				{"market given a price carries none", order.Market, order.UnknownTIF, 2, orderMarket, ""},
				{"limit follows the maker type", order.LimitMaker, order.UnknownTIF, 2, orderPostOnly, "2"},
				{"ioc follows the time in force", order.Limit, order.ImmediateOrCancel, 2, orderIOC, "2"},
				{"limit at a zero price", order.Limit, order.UnknownTIF, 0, orderLimit, "0"},
				{"post only at a zero price", order.LimitMaker, order.UnknownTIF, 0, orderPostOnly, "0"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					var mu sync.Mutex
					var body []byte
					e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path != "/sprd/order" {
							t.Errorf("unexpected request path %s", r.URL.Path)
							http.NotFound(w, r)
							return
						}
						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err, "reading the spread order body should not error")
						mu.Lock()
						body = b
						mu.Unlock()
						writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
					}))
					_, err := e.SubmitOrder(t.Context(), &order.Submit{
						Exchange:    e.Name,
						Pair:        spreadPair,
						AssetType:   asset.Spread,
						Side:        order.Buy,
						Type:        tc.orderType,
						TimeInForce: tc.tif,
						Amount:      1,
						Price:       tc.price,
					})
					require.NoError(t, err, "SubmitOrder must not error")
					mu.Lock()
					defer mu.Unlock()
					var sent map[string]any
					require.NoError(t, json.Unmarshal(body, &sent), "the spread order body must decode")
					assert.Equal(t, tc.expOrdType, sent["ordType"], "the spread order should send the documented ordType")
					if tc.expPX == "" {
						assert.NotContains(t, sent, "px", "a market spread order should send no price")
					} else {
						assert.Equal(t, tc.expPX, sent["px"], "a price-bearing spread order should send its price")
					}
				})
			}
			// fok is refused outright rather than silently downgraded to a resting
			// limit order, client-side, so nothing reaches OKX.
			for _, tc := range []struct {
				name   string
				tif    order.TimeInForce
				price  float64
				expErr error
			}{
				{"fok limit", order.FillOrKill, 2, order.ErrUnsupportedOrderType},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						t.Errorf("a rejected spread order should not reach OKX: %s", r.URL.Path)
						http.NotFound(w, r)
					}))
					_, err := e.SubmitOrder(t.Context(), &order.Submit{
						Exchange:    e.Name,
						Pair:        spreadPair,
						AssetType:   asset.Spread,
						Side:        order.Buy,
						Type:        order.Limit,
						TimeInForce: tc.tif,
						Amount:      1,
						Price:       tc.price,
					})
					require.ErrorIs(t, err, tc.expErr, "the rejected spread order must return its own error")
				})
			}
		})

		t.Run("reports the row error", func(t *testing.T) {
			t.Parallel()
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/sprd/order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":"1","msg":"All operations failed","data":[{"sCode":"51008","sMsg":"Order placement failed due to insufficient balance.","ordId":"","clOrdId":""}]}`))
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange: e.Name, Pair: spreadPair, AssetType: asset.Spread,
				Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2,
			})
			require.Error(t, err, "a failed spread row must error")
			assert.ErrorContains(t, err, "51008", "the error should report the row's sCode")
			assert.ErrorContains(t, err, "insufficient balance", "the error should report the row's sMsg")
		})

		t.Run("negative limit price", func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var body []byte
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/sprd/order" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the spread order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			}))
			_, err := e.SubmitOrder(t.Context(), &order.Submit{
				Exchange:  e.Name,
				Pair:      spreadPair,
				AssetType: asset.Spread,
				Side:      order.Buy,
				Type:      order.Limit,
				Amount:    1,
				Price:     -41.2,
			})
			require.NoError(t, err, "SubmitOrder must accept a negative spread limit price")
			mu.Lock()
			defer mu.Unlock()
			var sent map[string]any
			require.NoError(t, json.Unmarshal(body, &sent), "the spread order body must decode")
			assert.Equal(t, "-41.2", sent["px"], "the negative spread price should be sent as-is")
		})
	})

	t.Run("refuses a time in force OKX cannot honour", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			pair      currency.Pair
			orderType order.Type
			tif       order.TimeInForce
		}{
			{"good till time limit", asset.Spot, mainPair, order.Limit, order.GoodTillTime},
			{"good till time post only", asset.Spot, mainPair, order.Limit, order.GoodTillTime | order.PostOnly},
			{"good till time trigger", asset.Spot, mainPair, order.Trigger, order.GoodTillTime},
			{"good till time spread post only", asset.Spread, spreadPair, order.Limit, order.GoodTillTime | order.PostOnly},
			{"good till day limit", asset.Spot, mainPair, order.Limit, order.GoodTillDay},
			{"good till day post only", asset.Spot, mainPair, order.Limit, order.GoodTillDay | order.PostOnly},
			{"good till cancel and good till day", asset.Spot, mainPair, order.Limit, order.GoodTillCancel | order.GoodTillDay},
			{"good till day conditional stop", asset.Spot, mainPair, order.ConditionalStop, order.GoodTillDay},
			{"good till day spread limit", asset.Spread, spreadPair, order.Limit, order.GoodTillDay},
			{"stop or reduce limit", asset.Spot, mainPair, order.Limit, order.StopOrReduce},
			{"stop or reduce post only", asset.Spot, mainPair, order.Limit, order.StopOrReduce | order.PostOnly},
			{"stop or reduce perpetual swap limit", asset.PerpetualSwap, perpetualSwapPair, order.Limit, order.StopOrReduce},
			{"good till crossing limit", asset.Spot, mainPair, order.Limit, order.GoodTillCrossing},
			{"good till crossing market", asset.Spot, mainPair, order.Market, order.GoodTillCrossing},
			{"good till crossing spread limit", asset.Spread, spreadPair, order.Limit, order.GoodTillCrossing},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("a refused order should not reach OKX: %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: tc.pair, AssetType: tc.assetType, Side: order.Buy, Type: tc.orderType, TimeInForce: tc.tif,
					Amount: 1, Price: 100, TriggerPrice: 110, EndTime: time.UnixMilli(1800000000000),
				})
				assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "an order with a time in force OKX cannot honour should be refused")
			})
		}
	})

	t.Run("places good till crossing with post only as post_only", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			assetType  asset.Item
			pair       currency.Pair
			expPath    string
			expRequest string
		}{
			{"spot", asset.Spot, mainPair, "/trade/order", `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"post_only","sz":"1","px":"100"}`},
			{"spread", asset.Spread, spreadPair, "/sprd/order", `{"sprdId":"BTC-USDT_BTC-USDT-SWAP","side":"buy","ordType":"post_only","sz":"1","px":"100"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountPinExchange(t, "2", tc.expPath, tc.expRequest, orderActionReply)
				resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: tc.pair, AssetType: tc.assetType, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillCrossing | order.PostOnly, Amount: 1, Price: 100,
				})
				require.NoError(t, err, "SubmitOrder must place a post only order")
				assert.Equal(t, "1", resp.OrderID, "the placed order should report OKX's order ID")
			})
		}
	})

	t.Run("refuses order types the asset does not take", func(t *testing.T) {
		t.Parallel()
		optionPair := currency.NewPairWithDelimiter("BTC", "USD-261225-50000-C", "-")
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			pair      currency.Pair
			orderType order.Type
			tif       order.TimeInForce
		}{
			{"options market", asset.Options, optionPair, order.Market, order.UnknownTIF},
			{"options market good till cancel", asset.Options, optionPair, order.Market, order.GoodTillCancel},
			{"spot market maker protection", asset.Spot, mainPair, order.MarketMakerProtection, order.UnknownTIF},
			{"margin market maker protection post only", asset.Margin, mainPair, order.MarketMakerProtection, order.PostOnly},
			{"perpetual swap market maker protection", asset.PerpetualSwap, perpetualSwapPair, order.MarketMakerProtection, order.UnknownTIF},
			{"spread market maker protection", asset.Spread, spreadPair, order.MarketMakerProtection, order.UnknownTIF},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("a refused order should not reach OKX: %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: tc.pair, AssetType: tc.assetType, Side: order.Buy, Type: tc.orderType, TimeInForce: tc.tif, Amount: 1, Price: 100,
				})
				assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "an order type the asset does not take should be refused")
			})
		}
		t.Run("options market with immediate or cancel", func(t *testing.T) {
			t.Parallel()
			ex := newAccountPinExchange(t, "2", "/trade/order", `{"instId":"BTC-USD-261225-50000-C","tdMode":"cross","side":"buy","ordType":"ioc","sz":"1","px":"0.05"}`, orderActionReply)
			resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
				Exchange: ex.Name, Pair: optionPair, AssetType: asset.Options, Side: order.Buy, Type: order.Market, TimeInForce: order.ImmediateOrCancel, Amount: 1, Price: 0.05,
			})
			require.NoError(t, err, "a market order with immediate or cancel must place on options as ioc")
			assert.Equal(t, "1", resp.OrderID, "the placed order should report OKX's order ID")
		})
	})

	t.Run("an isolated margin order sends its margin currency", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name         string
			submit       *order.Submit
			accountLevel string // the acctLv a cross MARGIN order looks up, empty where none may be requested
			expPath      string
			expRequest   string
		}{
			{
				name:       "margin limit buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.Isolated, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"isolated","ccy":"USDT","side":"buy","ordType":"limit","sz":"1","px":"100"}`,
			},
			{
				name:       "margin limit sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.Isolated, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"isolated","ccy":"BTC","side":"sell","ordType":"limit","sz":"1","px":"100"}`,
			},
			{
				name:       "spot instrument in isolated mode",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Isolated, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"isolated","ccy":"USDT","side":"buy","ordType":"limit","sz":"1","px":"100"}`,
			},
			{
				name:       "margin trigger buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.Isolated, Side: order.Buy, Type: order.Trigger, Amount: 1, Price: 111, TriggerPrice: 110},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"isolated","ccy":"USDT","side":"buy","ordType":"trigger","sz":"1","triggerPx":"110","orderPx":"111","triggerPxType":"last"}`,
			},
			{
				name:       "margin conditional sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.Isolated, Side: order.Sell, Type: order.ConditionalStop, Amount: 1, Price: 89, TriggerPrice: 90},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"isolated","ccy":"BTC","side":"sell","ordType":"conditional","sz":"1","slTriggerPx":"90","slOrdPx":"89","slTriggerPxType":"last"}`,
			},
			{
				name:         "margin limit buy in cross mode",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.Multi, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				accountLevel: "3",
				expPath:      "/trade/order",
				expRequest:   `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"limit","sz":"1","px":"100"}`,
			},
			{
				name:       "spot limit buy in cash mode",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"limit","sz":"1","px":"100"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				reply := []map[string]string{{"ordId": "1", "algoId": "1", "sCode": "0"}}
				ex := newRequestPinExchange(t, tc.expPath, tc.expRequest, reply)
				if tc.accountLevel != "" {
					ex = newAccountPinExchange(t, tc.accountLevel, tc.expPath, tc.expRequest, reply)
				}
				tc.submit.Exchange = ex.Name
				resp, err := ex.SubmitOrder(t.Context(), tc.submit)
				require.NoError(t, err, "SubmitOrder must not error")
				assert.Equal(t, "1", resp.OrderID, "the placed order should report OKX's order ID")
			})
		}
		t.Run("isolated futures send none", func(t *testing.T) {
			t.Parallel()
			ex, requests := newRecordingExchange(t, func(path string) any {
				if path == "/account/config" {
					return []map[string]string{{"posMode": positionModeNet}}
				}
				return []map[string]string{{"ordId": "1", "sCode": "0"}}
			})
			_, err := ex.SubmitOrder(t.Context(), &order.Submit{
				Exchange: ex.Name, Pair: futuresPair, AssetType: asset.Futures, MarginType: margin.Isolated, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100,
			})
			require.NoError(t, err, "SubmitOrder must not error")
			reqs := requests()
			require.Len(t, reqs, 2, "the position mode and the order must be requested")
			assert.Equal(t, "/trade/order", reqs[1].path, "the order should be placed")
			assert.JSONEq(t, `{"instId":"BTC-USD","tdMode":"isolated","side":"buy","posSide":"net","ordType":"limit","sz":"1","px":"100"}`, reqs[1].body, "an isolated futures order should send no margin currency")
		})
	})

	t.Run("a cross margin order in Futures mode sends its margin currency", func(t *testing.T) {
		t.Parallel()
		orders := []struct {
			name       string
			submit     *order.Submit
			expPath    string
			expRequest string // the request in Futures mode; the other modes send it without its ccy
			ccy        string
		}{
			{
				name:       "margin limit buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","ccy":"USDT","side":"buy","ordType":"limit","sz":"1","px":"100"}`,
				ccy:        `"ccy":"USDT",`,
			},
			{
				name:       "margin market sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.Multi, Side: order.Sell, Type: order.Market, Amount: 0.5},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","ccy":"BTC","side":"sell","ordType":"market","sz":"0.5"}`,
				ccy:        `"ccy":"BTC",`,
			},
			{
				name:       "quote-only spot market buy in cross mode",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Multi, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","ccy":"USDT","side":"buy","ordType":"market","sz":"100"}`,
				ccy:        `"ccy":"USDT",`,
			},
			{
				name:       "margin trigger sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Sell, Type: order.Trigger, Amount: 1, Price: 89, TriggerPrice: 90},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","ccy":"BTC","side":"sell","ordType":"trigger","sz":"1","triggerPx":"90","orderPx":"89","triggerPxType":"last"}`,
				ccy:        `"ccy":"BTC",`,
			},
		}
		for _, level := range []struct {
			name, accountLevel string
			sendsCurrency      bool
		}{
			{"spot mode", "1", false},
			{"futures mode", "2", true},
			{"multi-currency margin", "3", false},
			{"portfolio margin", "4", false},
		} {
			for _, tc := range orders {
				t.Run(level.name+"/"+tc.name, func(t *testing.T) {
					t.Parallel()
					expRequest := tc.expRequest
					if !level.sendsCurrency {
						expRequest = strings.Replace(expRequest, tc.ccy, "", 1)
					}
					ex := newAccountPinExchange(t, level.accountLevel, tc.expPath, expRequest, []map[string]string{{"ordId": "1", "algoId": "1", "sCode": "0"}})
					submit := *tc.submit
					submit.Exchange = ex.Name
					resp, err := ex.SubmitOrder(t.Context(), &submit)
					require.NoError(t, err, "SubmitOrder must not error")
					assert.Equal(t, "1", resp.OrderID, "the placed order should report OKX's order ID")
				})
			}
		}
		t.Run("refuses an account configuration without its level", func(t *testing.T) {
			t.Parallel()
			for _, accountLevel := range []string{"", "0"} {
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/account/config" {
						t.Errorf("an order whose account level is unknown should not reach OKX: %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					writeOKXData(t, w, []map[string]string{{"acctLv": accountLevel, "posMode": positionModeNet}})
				}))
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100})
				assert.ErrorIsf(t, err, errInvalidAccountLevel, "SubmitOrder should refuse an account level of %q", accountLevel)
			}
		})
	})

	t.Run("spot lead trading sends spot_isolated", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			orderType  order.Type
			expPath    string
			expRequest string
		}{
			{"limit", order.Limit, "/trade/order", `{"instId":"BTC-USDT","tdMode":"spot_isolated","side":"buy","ordType":"limit","sz":"1","px":"100"}`},
			{"trigger", order.Trigger, "/trade/order-algo", `{"instId":"BTC-USDT","tdMode":"spot_isolated","side":"buy","ordType":"trigger","sz":"1","triggerPx":"110","orderPx":"100","triggerPxType":"last"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, tc.expPath, tc.expRequest, []map[string]string{{"ordId": "1", "algoId": "1", "sCode": "0"}})
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.SpotIsolated, Side: order.Buy, Type: tc.orderType, Amount: 1, Price: 100, TriggerPrice: 110,
				})
				require.NoError(t, err, "SubmitOrder must place a spot lead trading order")
			})
		}
	})

	t.Run("sizes orders in the currency OKX reads", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			submit     *order.Submit
			expPath    string
			expRequest string
		}{
			{
				name:       "quote-only spot market buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"market","sz":"100","tgtCcy":"quote_ccy"}`,
			},
			{
				name:       "quote-only spot market sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Market, QuoteAmount: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"sell","ordType":"market","sz":"100","tgtCcy":"quote_ccy"}`,
			},
			{
				name:       "quote-only margin market buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"market","sz":"100"}`,
			},
			{
				name:       "margin market sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Sell, Type: order.Market, Amount: 0.5, QuoteAmount: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"sell","ordType":"market","sz":"0.5"}`,
			},
			{
				name:       "spot trailing stop buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.TrailingStop, Amount: 0.5, QuoteAmount: 100, TrackingMode: order.Percentage, TrackingValue: 0.05},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"move_order_stop","sz":"100","callbackRatio":"0.05"}`,
			},
			{
				name:       "spot trailing stop sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.TrailingStop, Amount: 0.5, QuoteAmount: 100, TrackingMode: order.Percentage, TrackingValue: 0.05},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"sell","ordType":"move_order_stop","sz":"0.5","callbackRatio":"0.05"}`,
			},
			{
				name:       "margin trailing stop buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.TrailingStop, Amount: 0.5, QuoteAmount: 100, TrackingMode: order.Percentage, TrackingValue: 0.05},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"move_order_stop","sz":"100","callbackRatio":"0.05"}`,
			},
			{
				name:       "margin trailing stop sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Sell, Type: order.TrailingStop, Amount: 0.5, QuoteAmount: 100, TrackingMode: order.Percentage, TrackingValue: 0.05},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"sell","ordType":"move_order_stop","sz":"0.5","callbackRatio":"0.05"}`,
			},
			{
				name:       "margin market buy with immediate or cancel",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, TimeInForce: order.ImmediateOrCancel, Amount: 0.5, Price: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"ioc","sz":"0.5","px":"100"}`,
			},
			{
				name:       "quote-only spot market buy in cross mode",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Multi, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"market","sz":"100"}`,
			},
			{
				name:       "spot market sell in isolated mode",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Isolated, Side: order.Sell, Type: order.Market, Amount: 0.5, QuoteAmount: 100},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"isolated","ccy":"BTC","side":"sell","ordType":"market","sz":"0.5"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountPinExchange(t, "3", tc.expPath, tc.expRequest, []map[string]string{{"ordId": "1", "algoId": "1", "sCode": "0"}})
				tc.submit.Exchange = ex.Name
				_, err := ex.SubmitOrder(t.Context(), tc.submit)
				require.NoError(t, err, "SubmitOrder must place the order in the currency OKX reads")
			})
		}
		for _, tc := range []struct {
			name   string
			submit *order.Submit
			expErr error
		}{
			{"margin market buy sized in base", &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.5}, order.ErrAmountMustBeSet},
			{"spot market buy in cross mode sized in base", &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Multi, Side: order.Buy, Type: order.Market, Amount: 0.5}, order.ErrAmountMustBeSet},
			{"quote-only spot market sell in isolated mode", &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Isolated, Side: order.Sell, Type: order.Market, QuoteAmount: 100}, limits.ErrAmountBelowMin},
			{"spot trailing stop buy sized in base", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.TrailingStop, Amount: 0.5, TrackingMode: order.Percentage, TrackingValue: 0.05}, order.ErrAmountMustBeSet},
			{"margin trailing stop buy sized in base", &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.TrailingStop, Amount: 0.5, TrackingMode: order.Percentage, TrackingValue: 0.05}, order.ErrAmountMustBeSet},
			{"quote-only spot limit", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 100, QuoteAmount: 100}, limits.ErrAmountBelowMin},
			{"quote-only spot market with immediate or cancel", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, TimeInForce: order.ImmediateOrCancel, Price: 100, QuoteAmount: 100}, limits.ErrAmountBelowMin},
			{"quote-only margin market sell", &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Sell, Type: order.Market, QuoteAmount: 100}, limits.ErrAmountBelowMin},
			{"quote-only spread market", &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Market, QuoteAmount: 100}, limits.ErrAmountBelowMin},
			{"quote-only perpetual swap limit", &order.Submit{Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Limit, Price: 100, QuoteAmount: 100}, limits.ErrAmountBelowMin},
			{"quote-only twap", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.TWAP, Price: 100, QuoteAmount: 100, TrackingMode: order.Percentage, TrackingValue: 0.01}, limits.ErrAmountBelowMin},
			{"negative amount with a quote amount", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: -1, QuoteAmount: 100}, limits.ErrAmountBelowMin},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("a refused order should not reach OKX: %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				tc.submit.Exchange = ex.Name
				_, err := ex.SubmitOrder(t.Context(), tc.submit)
				assert.ErrorIs(t, err, tc.expErr, "an order whose amount OKX cannot read should be refused")
			})
		}
	})

	t.Run("a leg with only a trigger price executes at market", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			submit     *order.Submit
			expRequest string
		}{
			{
				name: "oco sell",
				submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.OCO, Amount: 1, RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90},
				}},
				expRequest: `{"instId":"BTC-USDT","tdMode":"cash","side":"sell","ordType":"oco","sz":"1","tpTriggerPx":"110","tpTriggerPxType":"last","tpOrdPx":"-1","slTriggerPx":"90","slTriggerPxType":"last","slOrdPx":"-1"}`,
			},
			{
				name: "oco buy with a limited take profit",
				submit: &order.Submit{Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.OCO, Amount: 1, RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 90, LimitPrice: 89}, StopLoss: order.RiskManagement{Price: 110},
				}},
				expRequest: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","posSide":"net","ordType":"oco","sz":"1","tpTriggerPx":"90","tpTriggerPxType":"last","tpOrdPx":"89","slTriggerPx":"110","slTriggerPxType":"last","slOrdPx":"-1"}`,
			},
			{
				name:       "conditional sell",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.ConditionalStop, Amount: 1, TriggerPrice: 90},
				expRequest: `{"instId":"BTC-USDT","tdMode":"cash","side":"sell","ordType":"conditional","sz":"1","slTriggerPx":"90","slTriggerPxType":"last","slOrdPx":"-1"}`,
			},
			{
				name:       "conditional buy",
				submit:     &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.ConditionalStop, Amount: 1, TriggerPrice: 110},
				expRequest: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"conditional","sz":"1","tgtCcy":"base_ccy","slTriggerPx":"110","slTriggerPxType":"last","slOrdPx":"-1"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountPinExchange(t, "2", "/trade/order-algo", tc.expRequest, []map[string]string{{"algoId": "1", "sCode": "0"}})
				tc.submit.Exchange = ex.Name
				_, err := ex.SubmitOrder(t.Context(), tc.submit)
				require.NoError(t, err, "SubmitOrder must place the algo order")
			})
		}
	})

	t.Run("a spot order traded on margin sends reduce only", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name         string
			accountLevel string
			assetType    asset.Item
			marginType   margin.Type
			expRequest   string
		}{
			{"cross margin", "3", asset.Spot, margin.Multi, `{"instId":"BTC-USDT","tdMode":"cross","side":"sell","ordType":"limit","sz":"1","px":"100","reduceOnly":true}`},
			{"cross margin in futures mode", "2", asset.Spot, margin.Multi, `{"instId":"BTC-USDT","tdMode":"cross","ccy":"BTC","side":"sell","ordType":"limit","sz":"1","px":"100","reduceOnly":true}`},
			{"isolated margin", "2", asset.Spot, margin.Isolated, `{"instId":"BTC-USDT","tdMode":"isolated","ccy":"BTC","side":"sell","ordType":"limit","sz":"1","px":"100","reduceOnly":true}`},
			{"default cross trade mode", "3", asset.Spot, margin.Unset, `{"instId":"BTC-USDT","tdMode":"cross","side":"sell","ordType":"limit","sz":"1","px":"100","reduceOnly":true}`},
			{"default cash trade mode", "2", asset.Spot, margin.Unset, `{"instId":"BTC-USDT","tdMode":"cash","side":"sell","ordType":"limit","sz":"1","px":"100"}`},
			{"cash", "3", asset.Spot, margin.NoMargin, `{"instId":"BTC-USDT","tdMode":"cash","side":"sell","ordType":"limit","sz":"1","px":"100"}`},
			{"spot lead trading", "3", asset.Spot, margin.SpotIsolated, `{"instId":"BTC-USDT","tdMode":"spot_isolated","side":"sell","ordType":"limit","sz":"1","px":"100"}`},
			{"margin asset", "3", asset.Margin, margin.Unset, `{"instId":"BTC-USDT","tdMode":"cross","side":"sell","ordType":"limit","sz":"1","px":"100","reduceOnly":true}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountPinExchange(t, tc.accountLevel, "/trade/order", tc.expRequest, orderActionReply)
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: mainPair, AssetType: tc.assetType, MarginType: tc.marginType, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 100, ReduceOnly: true,
				})
				require.NoError(t, err, "SubmitOrder must place the reduce only order")
			})
		}
	})

	t.Run("refuses a time in force the order type cannot honour", func(t *testing.T) {
		t.Parallel()
		optionPair := currency.NewPairWithDelimiter("BTC", "USD-261225-50000-C", "-")
		futuresContract := currency.NewPairWithDelimiter("BTC", "USD-261225", currency.DashDelimiter)
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			pair      currency.Pair
			orderType order.Type
			tif       order.TimeInForce
			price     float64
			expErr    error
		}{
			{"post only market", asset.Spot, mainPair, order.Market, order.PostOnly, 100, order.ErrUnsupportedTimeInForce},
			{"good till cancel post only market", asset.Spot, mainPair, order.Market, order.GoodTillCancel | order.PostOnly, 100, order.ErrUnsupportedTimeInForce},
			{"good till crossing post only market", asset.PerpetualSwap, perpetualSwapPair, order.Market, order.GoodTillCrossing | order.PostOnly, 100, order.ErrUnsupportedTimeInForce},
			{"post only optimal limit", asset.PerpetualSwap, perpetualSwapPair, order.OptimalLimit, order.PostOnly, 0, order.ErrUnsupportedTimeInForce},
			{"fill or kill optimal limit", asset.PerpetualSwap, perpetualSwapPair, order.OptimalLimit, order.FillOrKill, 0, order.ErrUnsupportedTimeInForce},
			{"good till cancel optimal limit", asset.Futures, futuresContract, order.OptimalLimit, order.GoodTillCancel, 0, order.ErrUnsupportedTimeInForce},
			{"immediate or cancel market maker protection", asset.Options, optionPair, order.MarketMakerProtection, order.ImmediateOrCancel, 0.05, order.ErrUnsupportedTimeInForce},
			{"fill or kill market maker protection", asset.Options, optionPair, order.MarketMakerProtection, order.FillOrKill, 0.05, order.ErrUnsupportedTimeInForce},
			{"post only trigger", asset.Spot, mainPair, order.Trigger, order.PostOnly, 100, order.ErrUnsupportedTimeInForce},
			{"fill or kill trigger at market", asset.Futures, futuresContract, order.Trigger, order.FillOrKill, marketOrderPrice, order.ErrUnsupportedTimeInForce},
			{"immediate or cancel trigger at market", asset.PerpetualSwap, perpetualSwapPair, order.Trigger, order.ImmediateOrCancel, marketOrderPrice, order.ErrUnsupportedTimeInForce},
			{"immediate or cancel chase", asset.PerpetualSwap, perpetualSwapPair, order.Chase, order.ImmediateOrCancel, 0, order.ErrUnsupportedTimeInForce},
			{"fill or kill twap", asset.Spot, mainPair, order.TWAP, order.FillOrKill, 100, order.ErrUnsupportedTimeInForce},
			{"immediate or cancel conditional", asset.Spot, mainPair, order.ConditionalStop, order.ImmediateOrCancel, 100, order.ErrUnsupportedTimeInForce},
			{"post only oco", asset.Spot, mainPair, order.OCO, order.PostOnly, 0, order.ErrUnsupportedTimeInForce},
			{"post only trailing stop", asset.PerpetualSwap, perpetualSwapPair, order.TrailingStop, order.PostOnly, 0, order.ErrUnsupportedTimeInForce},
			{"immediate or cancel post only limit", asset.Spot, mainPair, order.Limit, order.ImmediateOrCancel | order.PostOnly, 100, order.ErrInvalidTimeInForce},
			{"post only with an undefined flag", asset.Spot, mainPair, order.Limit, order.PostOnly | 1<<15, 100, order.ErrInvalidTimeInForce},
			{"any type", asset.Spot, mainPair, order.AnyType, order.ImmediateOrCancel, 100, order.ErrUnsupportedOrderType},
			{"unknown type", asset.Spot, mainPair, order.UnknownType, order.PostOnly, 100, order.ErrUnsupportedOrderType},
			{"type combination", asset.Spot, mainPair, order.TakeProfit | order.Limit, order.FillOrKill, 100, order.ErrUnsupportedOrderType},
			{"post only spread market", asset.Spread, spreadPair, order.Market, order.PostOnly, 0, order.ErrUnsupportedTimeInForce},
			{"good till crossing post only spread market", asset.Spread, spreadPair, order.Market, order.GoodTillCrossing | order.PostOnly, 0, order.ErrUnsupportedTimeInForce},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("a refused order should send no request: %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				s := &order.Submit{
					Exchange: ex.Name, Pair: tc.pair, AssetType: tc.assetType, Side: order.Buy, Type: tc.orderType, TimeInForce: tc.tif,
					Amount: 1, Price: tc.price, TriggerPrice: 110, TrackingMode: order.Distance, TrackingValue: 1,
				}
				if tc.orderType == order.OCO {
					s.RiskManagementModes = order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 120, LimitPrice: 121}, StopLoss: order.RiskManagement{Price: 90, LimitPrice: 89}}
				}
				_, err := ex.SubmitOrder(t.Context(), s)
				assert.ErrorIs(t, err, tc.expErr, "SubmitOrder should refuse an instruction the order cannot honour")
			})
		}
	})

	t.Run("a trigger order sends its time in force as its advance order type", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			assetType  asset.Item
			pair       currency.Pair
			tif        order.TimeInForce
			expRequest string
		}{
			{"fill or kill", asset.Spot, mainPair, order.FillOrKill, `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"trigger","sz":"1","triggerPx":"110","orderPx":"100","advanceOrdType":"fok","triggerPxType":"last"}`},
			{"immediate or cancel", asset.PerpetualSwap, perpetualSwapPair, order.ImmediateOrCancel, `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","posSide":"net","ordType":"trigger","sz":"1","triggerPx":"110","orderPx":"100","advanceOrdType":"ioc","triggerPxType":"last"}`},
			{"good till cancel", asset.Spot, mainPair, order.GoodTillCancel, `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"trigger","sz":"1","triggerPx":"110","orderPx":"100","triggerPxType":"last"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountPinExchange(t, "2", "/trade/order-algo", tc.expRequest, []map[string]string{{"algoId": "1", "sCode": "0"}})
				resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
					Exchange: ex.Name, Pair: tc.pair, AssetType: tc.assetType, Side: order.Buy, Type: order.Trigger, TimeInForce: tc.tif, Amount: 1, Price: 100, TriggerPrice: 110,
				})
				require.NoError(t, err, "SubmitOrder must place the trigger order")
				exp := &order.SubmitResponse{
					Exchange: ex.Name, Type: order.Trigger, Side: order.Buy, Pair: tc.pair, AssetType: tc.assetType, TimeInForce: tc.tif,
					Price: 100, Amount: 1, TriggerPrice: 110, Status: order.New, OrderID: "1",
				}
				resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
				assert.Equal(t, exp, resp, "SubmitOrder should report the time in force the trigger order was placed with")
			})
		}
	})

	t.Run("attaches the take profit and stop loss", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name         string
			accountLevel string
			submit       *order.Submit
			expPath      string
			expRequest   string
		}{
			{
				name: "limit buy with both legs", accountLevel: "2",
				submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Enabled: true, Price: 120, LimitPrice: 121, OrderType: order.Limit},
					StopLoss:   order.RiskManagement{Enabled: true, Price: 90, OrderType: order.Market, TriggerPriceType: order.MarkPrice},
				}},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"limit","sz":"1","px":"100","attachAlgoOrds":[{"tpTriggerPx":"120","tpOrdPx":"121","tpTriggerPxType":"last","slTriggerPx":"90","slOrdPx":"-1","slTriggerPxType":"mark"}]}`,
			},
			{
				name: "limit buy with a stop loss alone", accountLevel: "2",
				submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{
					StopLoss: order.RiskManagement{Price: 90, TriggerPriceType: order.IndexPrice},
				}},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"limit","sz":"1","px":"100","attachAlgoOrds":[{"slTriggerPx":"90","slOrdPx":"-1","slTriggerPxType":"index"}]}`,
			},
			{
				name: "perpetual swap sell with a take profit alone", accountLevel: "2",
				submit: &order.Submit{Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Sell, Type: order.Limit, TimeInForce: order.PostOnly, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 80, LimitPrice: 79},
				}},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"sell","posSide":"net","ordType":"post_only","sz":"1","px":"100","attachAlgoOrds":[{"tpTriggerPx":"80","tpOrdPx":"79","tpTriggerPxType":"last"}]}`,
			},
			{
				name: "quote sized spot market buy", accountLevel: "2",
				submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 100, RiskManagementModes: order.RiskManagementModes{
					StopLoss: order.RiskManagement{Price: 90},
				}},
				expPath:    "/trade/order",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"market","sz":"100","tgtCcy":"quote_ccy","attachAlgoOrds":[{"slTriggerPx":"90","slOrdPx":"-1","slTriggerPxType":"last"}]}`,
			},
			{
				name: "trigger buy with both legs", accountLevel: "3",
				submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Trigger, Amount: 1, Price: 100, TriggerPrice: 105, RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 120}, StopLoss: order.RiskManagement{Price: 90, LimitPrice: 89},
				}},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT","tdMode":"cross","side":"buy","ordType":"trigger","sz":"1","triggerPx":"105","orderPx":"100","triggerPxType":"last","attachAlgoOrds":[{"tpTriggerPx":"120","tpOrdPx":"-1","tpTriggerPxType":"last","slTriggerPx":"90","slOrdPx":"89","slTriggerPxType":"last"}]}`,
			},
			{
				name: "trigger sell with a stop loss alone in futures mode", accountLevel: "2",
				submit: &order.Submit{Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Sell, Type: order.Trigger, Amount: 1, Price: marketOrderPrice, TriggerPrice: 95, RiskManagementModes: order.RiskManagementModes{
					StopLoss: order.RiskManagement{Price: 110, OrderType: order.Market},
				}},
				expPath:    "/trade/order-algo",
				expRequest: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"sell","posSide":"net","ordType":"trigger","sz":"1","triggerPx":"95","orderPx":"-1","triggerPxType":"last","attachAlgoOrds":[{"slTriggerPx":"110","slOrdPx":"-1","slTriggerPxType":"last"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountPinExchange(t, tc.accountLevel, tc.expPath, tc.expRequest, []map[string]string{{"ordId": "1", "algoId": "1", "sCode": "0"}})
				tc.submit.Exchange = ex.Name
				resp, err := ex.SubmitOrder(t.Context(), tc.submit)
				require.NoError(t, err, "SubmitOrder must place the order with its attachments")
				assert.Equal(t, "1", resp.OrderID, "SubmitOrder should report the placed order's ID")
			})
		}
	})

	t.Run("refuses a take profit or stop loss the order cannot carry", func(t *testing.T) {
		t.Parallel()
		stopLoss := order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90}}
		for _, tc := range []struct {
			name         string
			accountLevel string
			submit       *order.Submit
			expErr       error
		}{
			{"trigger in spot mode", "1", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Trigger, Amount: 1, Price: 100, TriggerPrice: 105, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"base sized spot market buy", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: 1, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"quote sized spot market sell", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Market, QuoteAmount: 100, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"spread", "2", &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, Amount: 1, Price: -5, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"conditional", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.ConditionalStop, Amount: 1, TriggerPrice: 95, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"chase", "2", &order.Submit{Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Chase, Amount: 1, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"trailing stop", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.TrailingStop, Amount: 1, TrackingMode: order.Distance, TrackingValue: 1, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"twap", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.TWAP, Amount: 1, Price: 100, TrackingMode: order.Distance, TrackingValue: 1, RiskManagementModes: stopLoss}, errAttachedOrderUnsupported},
			{"stop entry", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 110}}}, errAttachedOrderUnsupported},
			{"take profit and stop loss mode", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{Mode: "partial", StopLoss: order.RiskManagement{Price: 90}}}, errAttachedOrderUnsupported},
			{"oco stop entry", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.OCO, Amount: 1, RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90}, StopEntry: order.RiskManagement{Price: 100},
			}}, errAttachedOrderUnsupported},
			{"leg without a trigger price", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true}}}, limits.ErrPriceBelowMin},
			{"limit leg without a limit price", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 120, OrderType: order.Limit}}}, limits.ErrPriceBelowMin},
			{"leg of a type OKX cannot attach", "2", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Price: 90, OrderType: order.StopLimit}}}, order.ErrUnsupportedOrderType},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": tc.accountLevel, "posMode": positionModeNet}})
						return
					}
					t.Errorf("a refused order should not reach OKX: %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				tc.submit.Exchange = ex.Name
				_, err := ex.SubmitOrder(t.Context(), tc.submit)
				assert.ErrorIs(t, err, tc.expErr, "SubmitOrder should refuse an attachment the order cannot carry")
			})
		}
	})

	t.Run("acts for one account when the exchange's credentials change mid-order", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var requests []string
		var body []byte
		var ex atomic.Pointer[Exchange]
		mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("OK-ACCESS-KEY")
			mu.Lock()
			requests = append(requests, r.URL.Path+" "+key)
			mu.Unlock()
			switch r.URL.Path {
			case "/account/config":
				ex.Load().SetCredentials(secondaryAccount())
				mode := positionModeLongShort
				if key == secondaryAccount().Key {
					mode = positionModeNet
				}
				writeOKXData(t, w, []map[string]string{{"posMode": mode, "acctLv": "2"}})
			case "/trade/order":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		ex.Store(mock)
		mock.SetCredentials(primaryAccount())
		_, err := mock.SubmitOrder(t.Context(), &order.Submit{
			Exchange: mock.Name, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, MarginType: margin.Multi,
			Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ReduceOnly: true,
		})
		require.NoError(t, err, "SubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/account/config primary-key", "/trade/order primary-key"}, requests, "the order should be placed for the account whose configuration it read")
		assert.JSONEq(t, `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","posSide":"short","ordType":"limit","sz":"1","px":"100"}`, string(body), "the order should follow its account's long/short mode")
	})

	t.Run("acts for one account when the context's credentials change mid-order", func(t *testing.T) {
		t.Parallel()
		ctx := accounts.DeployCredentialsToContext(t.Context(), primaryAccount())
		store, ok := ctx.Value(accounts.ContextCredentialsFlag).(*accounts.ContextCredentialsStore)
		require.True(t, ok, "the context must carry a credentials store")
		var mu sync.Mutex
		var requests []string
		var body []byte
		mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("OK-ACCESS-KEY")
			mu.Lock()
			requests = append(requests, r.URL.Path+" "+key)
			mu.Unlock()
			switch r.URL.Path {
			case "/account/config":
				store.Load(secondaryAccount())
				mode := positionModeLongShort
				if key == secondaryAccount().Key {
					mode = positionModeNet
				}
				writeOKXData(t, w, []map[string]string{{"posMode": mode, "acctLv": "2"}})
			case "/trade/order":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the order body should not error")
				mu.Lock()
				body = b
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		_, err := mock.SubmitOrder(ctx, &order.Submit{
			Exchange: mock.Name, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, MarginType: margin.Multi,
			Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ReduceOnly: true,
		})
		require.NoError(t, err, "SubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/account/config primary-key", "/trade/order primary-key"}, requests, "the order should be placed for the account whose configuration it read")
		assert.JSONEq(t, `{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","posSide":"short","ordType":"limit","sz":"1","px":"100"}`, string(body), "the order should follow its account's long/short mode")
	})
}

// newAccountPinExchange returns an exchange whose mock answers the account configuration with the account level given,
// in net position mode, and pins every other request as newRequestPinExchange does: a cross MARGIN order looks the
// account level up before it places
func newAccountPinExchange(t *testing.T, accountLevel, path, body string, data any) *Exchange {
	t.Helper()
	return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/config" {
			writeOKXData(t, w, []map[string]string{{"acctLv": accountLevel, "posMode": positionModeNet}})
			return
		}
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, path, r.URL.Path, "request path should match")
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		assert.JSONEq(t, body, string(b), "request body should match")
		writeOKXData(t, w, data)
	}))
}

// recordedRequest holds the path and body of a REST request a mock exchange
// received
type recordedRequest struct {
	path string
	body string
}

// newRecordingExchange returns a mock exchange answering each REST request with
// the data reply returns for its path, and a function returning the requests
// it has received
func newRecordingExchange(t *testing.T, reply func(path string) any) (ex *Exchange, requests func() []recordedRequest) {
	t.Helper()
	var mu sync.Mutex
	var received []recordedRequest
	ex = newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		mu.Lock()
		received = append(received, recordedRequest{path: r.URL.Path, body: string(body)})
		mu.Unlock()
		writeOKXData(t, w, reply(r.URL.Path))
	}))
	return ex, func() []recordedRequest {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.CancelOrder(contextGenerate(), nil), order.ErrCancelOrderIsNil, "CancelOrder should error for a nil cancellation")

	arg := &order.Cancel{
		AccountID: "1",
		AssetType: asset.Binary,
	}
	err := e.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	err = e.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	arg.Pair = mainPair
	err = e.CancelOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		err = e.CancelOrder(contextGenerate(), &order.Cancel{
			OrderID: "1", AccountID: "1", Pair: mainPair, AssetType: asset.Spot,
		})
		assert.NoError(t, err)

		err = e.CancelOrder(contextGenerate(), &order.Cancel{
			Type: order.OCO, OrderID: "1", AccountID: "1", Pair: mainPair, AssetType: asset.Spot,
		})
		assert.NoError(t, err)

		err = e.CancelOrder(contextGenerate(), &order.Cancel{
			OrderID: "1", AccountID: "1", Pair: spreadPair, AssetType: asset.Spread,
		})
		assert.NoError(t, err)
	})

	t.Run("spread replies", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			reply string
			err   error
		}{
			{"no data", `{"code":"0","msg":"","data":null}`, common.ErrNoResponse},
			{"no order ID", `{"code":"0","msg":"","data":[{"ordId":"","clOrdId":"S1","sCode":"0","sMsg":""}]}`, common.ErrInvalidResponse},
			{"cancel failed", `{"code":"1","msg":"All operations failed","data":[{"ordId":"","clOrdId":"S1","sCode":"51400","sMsg":"Order does not exist"}]}`, nil},
			{"failed status in a success reply", `{"code":"0","msg":"","data":[{"ordId":"SPRD-1","clOrdId":"S1","sCode":"51400","sMsg":"Order does not exist"}]}`, nil},
			{"another order's acknowledgement", `{"code":"0","msg":"","data":[{"ordId":"SPRD-2","clOrdId":"S2","sCode":"0","sMsg":""}]}`, errReplyOrderMismatch},
			{"reply echoing no client order ID", `{"code":"0","msg":"","data":[{"ordId":"SPRD-1","clOrdId":"","sCode":"0","sMsg":""}]}`, errReplyOrderMismatch},
			{"row without a status code", `{"code":"0","msg":"","data":[{"ordId":"SPRD-1","clOrdId":"S1"}]}`, errMissingStatusCode},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/sprd/cancel-order" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tc.reply))
				}))
				err := ex.CancelOrder(t.Context(), &order.Cancel{
					AssetType:     asset.Spread,
					Pair:          spreadPair,
					ClientOrderID: "S1",
				})
				if tc.err != nil {
					assert.ErrorIs(t, err, tc.err, "CancelOrder should return the sentinel for a reply that names no order")
				} else {
					assert.ErrorContains(t, err, "Order does not exist", "CancelOrder should report the failed cancel's status message")
				}
			})
		}
	})

	t.Run("spread replies to a cancel by order ID", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			data   any
			expErr error
		}{
			{"cancelled", []map[string]string{{"ordId": "1001", "clOrdId": "", "sCode": "0", "sMsg": ""}}, nil},
			{"another order's acknowledgement", []map[string]string{{"ordId": "9999", "clOrdId": "", "sCode": "0", "sMsg": ""}}, errReplyOrderMismatch},
			{"row without a status code", []map[string]string{{"ordId": "1001", "clOrdId": ""}}, errMissingStatusCode},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/sprd/cancel-order", `{"ordId":"1001"}`, tc.data)
				err := ex.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spread, Pair: spreadPair, OrderID: "1001"})
				assert.ErrorIs(t, err, tc.expErr, "CancelOrder should accept only an acknowledgement of the order it cancelled")
			})
		}
	})

	t.Run("algo empty reply", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/cancel-algos" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []any{})
		}))
		err := ex.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-1", Type: order.Trigger})
		assert.ErrorIs(t, err, common.ErrNoResponse, "an algo cancel reply without results should return ErrNoResponse")
	})

	t.Run("order replies", func(t *testing.T) {
		t.Parallel()
		const failed = "Order cancellation failed as the order has been filled, canceled or does not exist"
		for _, tc := range []struct {
			name, path, reply string
			cancel            *order.Cancel
			expErr            error  // nil with expText empty when the cancel succeeds
			expText           string // the failed cancel's status message
		}{
			{name: "cancelled", path: "/trade/cancel-order", reply: `[{"ordId":"1","clOrdId":"","sCode":"0","sMsg":""}]`},
			{name: "cancelled by client order ID", path: "/trade/cancel-order", reply: `[{"ordId":"1","clOrdId":"C1","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{ClientOrderID: "C1"}},
			{name: "no data", path: "/trade/cancel-order", reply: `null`, expErr: common.ErrNoResponse},
			{name: "null row", path: "/trade/cancel-order", reply: `[null]`, expErr: common.ErrNoResponse},
			{name: "empty row", path: "/trade/cancel-order", reply: `[{}]`, expErr: common.ErrInvalidResponse},
			{name: "row without an order ID", path: "/trade/cancel-order", reply: `[{"ordId":"","clOrdId":"C1","sCode":"0","sMsg":""}]`, expErr: common.ErrInvalidResponse},
			{name: "failed status in a success reply", path: "/trade/cancel-order", reply: `[{"ordId":"1","clOrdId":"","sCode":"51400","sMsg":"` + failed + `"}]`, expText: failed},
			{name: "failed status without an order ID", path: "/trade/cancel-order", reply: `[{"ordId":"","clOrdId":"C1","sCode":"51400","sMsg":"` + failed + `"}]`, cancel: &order.Cancel{ClientOrderID: "C1"}, expText: failed},
			{name: "another order's acknowledgement", path: "/trade/cancel-order", reply: `[{"ordId":"9999","clOrdId":"","sCode":"0","sMsg":""}]`, expErr: errReplyOrderMismatch},
			{name: "by client order ID, another order's acknowledgement", path: "/trade/cancel-order", reply: `[{"ordId":"9999","clOrdId":"C2","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{ClientOrderID: "C1"}, expErr: errReplyOrderMismatch},
			{name: "by client order ID, a reply echoing none", path: "/trade/cancel-order", reply: `[{"ordId":"1","clOrdId":"","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{ClientOrderID: "C1"}, expErr: errReplyOrderMismatch},
			{name: "row without a status code", path: "/trade/cancel-order", reply: `[{"ordId":"1","clOrdId":""}]`, expErr: errMissingStatusCode},
			{name: "algo cancelled", path: "/trade/cancel-algos", reply: `[{"algoId":"1","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{Type: order.Trigger, OrderID: "1"}},
			{name: "algo cancelled by client order ID", path: "/trade/cancel-algos", reply: `[{"algoId":"1","algoClOrdId":"C1","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{Type: order.ConditionalStop, ClientOrderID: "C1"}},
			{name: "algo cancelled by client order ID the reply leaves out", path: "/trade/cancel-algos", reply: `[{"algoId":"1","algoClOrdId":"","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{Type: order.ConditionalStop, ClientOrderID: "C1"}},
			{name: "algo by client order ID, another order's acknowledgement", path: "/trade/cancel-algos", reply: `[{"algoId":"9999","algoClOrdId":"C2","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{Type: order.ConditionalStop, ClientOrderID: "C1"}, expErr: errReplyOrderMismatch},
			{name: "algo another order's acknowledgement", path: "/trade/cancel-algos", reply: `[{"algoId":"9999","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{Type: order.Trigger, OrderID: "1"}, expErr: errReplyOrderMismatch},
			{name: "algo row without a status code", path: "/trade/cancel-algos", reply: `[{"algoId":"1"}]`, cancel: &order.Cancel{Type: order.Trigger, OrderID: "1"}, expErr: errMissingStatusCode},
			{name: "algo no data", path: "/trade/cancel-algos", reply: `null`, cancel: &order.Cancel{Type: order.OCO, OrderID: "1"}, expErr: common.ErrNoResponse},
			{name: "algo null row", path: "/trade/cancel-algos", reply: `[null]`, cancel: &order.Cancel{Type: order.TWAP, OrderID: "1"}, expErr: common.ErrInvalidResponse},
			{name: "algo empty row", path: "/trade/cancel-algos", reply: `[{}]`, cancel: &order.Cancel{Type: order.Trigger, OrderID: "1"}, expErr: common.ErrInvalidResponse},
			{name: "algo row without an algo ID", path: "/trade/cancel-algos", reply: `[{"algoId":"","algoClOrdId":"C1","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{Type: order.Trigger, ClientOrderID: "C1"}, expErr: common.ErrInvalidResponse},
			{name: "algo rows for other orders", path: "/trade/cancel-algos", reply: `[{"algoId":"1","sCode":"0","sMsg":""},{"algoId":"2","sCode":"0","sMsg":""}]`, cancel: &order.Cancel{Type: order.Chase, OrderID: "1"}, expErr: common.ErrInvalidResponse},
			{name: "algo failed status in a success reply", path: "/trade/cancel-algos", reply: `[{"algoId":"1","sCode":"51400","sMsg":"` + failed + `"}]`, cancel: &order.Cancel{Type: order.TrailingStop, OrderID: "1"}, expText: failed},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, tc.path, r.URL.Path, "the cancel should reach the endpoint for its order type")
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.reply + `}`))
				}))
				cancel := &order.Cancel{OrderID: "1"}
				if tc.cancel != nil {
					cancel = tc.cancel
				}
				cancel.AssetType, cancel.Pair = asset.Spot, mainPair
				err := ex.CancelOrder(t.Context(), cancel)
				switch {
				case tc.expErr != nil:
					assert.ErrorIs(t, err, tc.expErr, "CancelOrder should return the sentinel for a reply that confirms no cancel")
				case tc.expText != "":
					assert.ErrorContains(t, err, tc.expText, "CancelOrder should report the failed cancel's status message")
				default:
					assert.NoError(t, err, "CancelOrder should report the acknowledged cancel")
				}
			})
		}
	})

	t.Run("cancels an algo order by its client order ID", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			cancel     *order.Cancel
			expRequest string
		}{
			{"client order ID alone", &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, ClientOrderID: "client802"}, `[{"instId":"BTC-USDT","algoClOrdId":"client802"}]`},
			{"both IDs", &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.OCO, OrderID: "802", ClientOrderID: "client802"}, `[{"instId":"BTC-USDT","algoId":"802","algoClOrdId":"client802"}]`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/cancel-algos", tc.expRequest, []map[string]string{{"algoId": "802", "sCode": "0", "sMsg": ""}})
				require.NoError(t, ex.CancelOrder(t.Context(), tc.cancel), "CancelOrder must cancel the algo order")
			})
		}
	})
}

func TestCancelReplyError(t *testing.T) {
	t.Parallel()
	const failed = "status code: `51400` status message: \"Order does not exist\""
	byClientOrderID := &order.Cancel{ClientOrderID: "C1"}
	for _, tc := range []struct {
		name                string
		cancel              *order.Cancel // OrderID 1 and ClientOrderID C1 when nil
		row                 *orderOutcome
		echoesClientOrderID bool
		expErr              error
		expText             string
	}{
		{name: "acknowledged", row: &orderOutcome{orderID: "1", clientOrderID: "C1"}, echoesClientOrderID: true},
		{name: "no row", echoesClientOrderID: true, expErr: common.ErrNoResponse},
		{name: "no order ID", row: &orderOutcome{clientOrderID: "C1"}, echoesClientOrderID: true, expErr: common.ErrInvalidResponse},
		{name: "failed status", row: &orderOutcome{orderID: "1", status: getStatusError(51400, "Order does not exist")}, echoesClientOrderID: true, expText: failed},
		{name: "failed status without an order ID", row: &orderOutcome{status: getStatusError(51400, "Order does not exist")}, echoesClientOrderID: true, expText: failed},
		{name: "no status code", row: &orderOutcome{orderID: "1", clientOrderID: "C1", status: errMissingStatusCode}, echoesClientOrderID: true, expErr: errMissingStatusCode},
		{name: "another order's acknowledgement", row: &orderOutcome{orderID: "9999", clientOrderID: "C1"}, echoesClientOrderID: true, expErr: errReplyOrderMismatch},
		{name: "another client order ID beside the order ID", row: &orderOutcome{orderID: "1", clientOrderID: "C2"}, echoesClientOrderID: true},
		{name: "by client order ID", cancel: byClientOrderID, row: &orderOutcome{orderID: "1", clientOrderID: "C1"}, echoesClientOrderID: true},
		{name: "by client order ID echoing another", cancel: byClientOrderID, row: &orderOutcome{orderID: "1", clientOrderID: "C2"}, echoesClientOrderID: true, expErr: errReplyOrderMismatch},
		{name: "by client order ID echoing none", cancel: byClientOrderID, row: &orderOutcome{orderID: "1"}, echoesClientOrderID: true, expErr: errReplyOrderMismatch},
		{name: "by client order ID where the reply may echo none", cancel: byClientOrderID, row: &orderOutcome{orderID: "1"}},
		{name: "by client order ID where the reply echoes another", cancel: byClientOrderID, row: &orderOutcome{orderID: "1", clientOrderID: "C2"}, expErr: errReplyOrderMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cancel := tc.cancel
			if cancel == nil {
				cancel = &order.Cancel{OrderID: "1", ClientOrderID: "C1"}
			}
			err := cancelReplyError(tc.row, cancel, tc.echoesClientOrderID)
			switch {
			case tc.expErr != nil:
				assert.ErrorIs(t, err, tc.expErr, "cancelReplyError should return the sentinel for a row that confirms no cancel")
			case tc.expText != "":
				assert.EqualError(t, err, tc.expText, "cancelReplyError should report the row's status")
			default:
				assert.NoError(t, err, "cancelReplyError should accept an acknowledged cancel")
			}
		})
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(contextGenerate(), make([]order.Cancel, 21))
	require.ErrorIs(t, err, errExceedLimit)
	_, err = e.CancelBatchOrders(contextGenerate(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	arg := order.Cancel{
		AccountID: "1",
		AssetType: asset.Binary,
	}
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	arg.AssetType = asset.Spot
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	arg.Pair = mainPair
	arg.Type = order.Liquidation
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Trigger
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.Type = order.Limit
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	spreadCancel := order.Cancel{
		AssetType: asset.Spread,
		Pair:      spreadPair,
	}
	_, err = e.CancelBatchOrders(contextGenerate(), []order.Cancel{spreadCancel})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		orderCancellationParams := []order.Cancel{
			{
				OrderID:   "1",
				AccountID: "1",
				Pair:      mainPair,
				AssetType: asset.Spot,
			},
			{
				OrderID:   "1",
				AccountID: "1",
				Pair:      perpetualSwapPair,
				AssetType: asset.PerpetualSwap,
			},
			{
				OrderID:   "1",
				AccountID: "1",
				Type:      order.Trigger,
				Pair:      mainPair,
				AssetType: asset.Spot,
			},
		}
		result, err := e.CancelBatchOrders(contextGenerate(), orderCancellationParams)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("rest partial success", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			reply   string
			partial bool
			status  map[string]string
		}{
			{
				name:    "per-order results",
				reply:   `{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"51400","sMsg":"Order does not exist"}]}`,
				partial: true,
				status:  map[string]string{"OK-1": order.PendingCancel.String(), "FAIL-1": "Order does not exist"},
			},
			{
				name:    "null result",
				reply:   `{"code":"2","msg":"Bulk operation partially succeeded.","data":[null,{"ordId":"OK-1","sCode":"0","sMsg":""}]}`,
				partial: true,
				status:  map[string]string{"OK-1": order.PendingCancel.String()},
			},
			{
				name:   "undecodable result",
				reply:  `{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"bad","sMsg":"Order does not exist"}]}`,
				status: map[string]string{},
			},
			{
				name:   "undecodable result without a message",
				reply:  `{"code":"2","msg":"","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"bad","sMsg":"Order does not exist"}]}`,
				status: map[string]string{},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/cancel-batch-orders" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tc.reply))
				}))
				resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
					{AssetType: asset.Spot, Pair: mainPair, OrderID: "OK-1"},
					{AssetType: asset.Spot, Pair: mainPair, OrderID: "FAIL-1"},
				})
				require.Error(t, err, "a partially successful REST batch must report its error")
				assert.Equal(t, tc.partial, errors.Is(err, errPartialSuccess), "only results that decode should be marked as a partial success")
				require.NotNil(t, resp, "CancelBatchOrders must return the partial results")
				assert.Equal(t, tc.status, resp.Status, "a partially successful REST batch should report every decoded per-order result")
			})
		}
	})

	t.Run("spread guards", func(t *testing.T) {
		t.Parallel()
		const spotOrdID = "2341161427393388545"
		var mu sync.Mutex
		var requests int
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/cancel-batch-orders" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			mu.Lock()
			requests++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":"","ordId":"` + spotOrdID + `","clOrdId":"spot-cl"}]}`))
		}))
		requestsSent := func() int {
			mu.Lock()
			defer mu.Unlock()
			return requests
		}

		_, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{
			AssetType:     asset.Spread,
			Pair:          spreadPair,
			ClientOrderID: "spread-cl",
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread cancel should be refused: spread orders cancel one request at a time")
		assert.Zero(t, requestsSent(), "a refused batch should send no cancels")

		_, err = ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "spot-order"},
			{AssetType: asset.Spread, Pair: spreadPair, ClientOrderID: "spread-cl"},
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a spread entry should fail the batch")
		assert.Zero(t, requestsSent(), "no cancel should be sent before the whole batch validates")

		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{
			AssetType:     asset.Spot,
			Pair:          mainPair,
			ClientOrderID: "spot-cl",
		}})
		require.NoError(t, err, "CancelBatchOrders must accept a client-ID-only spot cancel")
		assert.Equal(t, map[string]string{spotOrdID: order.PendingCancel.String()}, resp.Status,
			"a client-ID-only spot cancel should report the status under the order ID OKX returns")
	})

	t.Run("reports algo results", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/cancel-algos" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[` +
				`{"algoId":"ALGO-OK","sCode":"0","sMsg":""},` +
				`{"algoId":"ALGO-BAD","sCode":"51000","sMsg":"The algo order does not exist"}]}`))
		}))

		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-OK", Type: order.Trigger},
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-BAD", Type: order.Trigger},
		})
		require.NoError(t, err, "CancelBatchOrders must not error when the batch is accepted")
		assert.Equal(t, order.PendingCancel.String(), resp.Status["ALGO-OK"], "an acknowledged algo cancel should report the pending state, as sCode 0 covers the request only")
		assert.Equal(t, "The algo order does not exist", resp.Status["ALGO-BAD"], "a failed algo cancel should report its status message")
	})

	t.Run("algo partial success", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/cancel-algos" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[null,` +
				`{"algoId":"ALGO-OK","sCode":"0","sMsg":""},` +
				`{"algoId":"ALGO-BAD","sCode":"51000","sMsg":"The algo order does not exist"}]}`))
		}))
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-OK", Type: order.Trigger},
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-BAD", Type: order.Trigger},
		})
		require.ErrorIs(t, err, errPartialSuccess, "a partially successful algo batch must report its error")
		require.NotNil(t, resp, "CancelBatchOrders must return the partial results")
		assert.Equal(t, map[string]string{
			"ALGO-OK":  order.PendingCancel.String(),
			"ALGO-BAD": "The algo order does not exist",
		}, resp.Status, "a partially successful algo batch should report every per-order result")
	})

	t.Run("algo limit", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var requests int
		var lastSize int
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/cancel-algos" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			raw, readErr := io.ReadAll(r.Body)
			assert.NoError(t, readErr, "reading the algo cancel body should not error")
			var sent []map[string]string
			if err := json.Unmarshal(raw, &sent); err != nil {
				t.Errorf("the algo cancel body should decode: %v", err)
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			rows := make([]map[string]string, 0, len(sent))
			for _, a := range sent {
				rows = append(rows, map[string]string{"algoId": a["algoId"], "sCode": "0", "sMsg": ""})
			}
			mu.Lock()
			requests++
			lastSize = len(rows)
			mu.Unlock()
			writeOKXData(t, w, rows)
		}))
		cancels := make([]order.Cancel, 0, maxCancelAlgosPerRequest+1)
		for i := range maxCancelAlgosPerRequest + 1 {
			cancels = append(cancels, order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "ALGO-" + strconv.Itoa(i), Type: order.Trigger})
		}
		_, err := ex.CancelBatchOrders(t.Context(), cancels)
		assert.ErrorIs(t, err, errExceedLimit, "more than ten algo orders in one batch should be refused")

		resp, err := ex.CancelBatchOrders(t.Context(), cancels[:maxCancelAlgosPerRequest])
		require.NoError(t, err, "CancelBatchOrders must accept ten algo orders")
		assert.Len(t, resp.Status, maxCancelAlgosPerRequest, "every cancelled algo order should appear in the status map")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 1, requests, "the algo cancels within the limit should be sent as one request")
		assert.Equal(t, maxCancelAlgosPerRequest, lastSize, "the single algo request should carry the whole batch")
	})

	t.Run("cancels algo orders by client order ID", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/trade/cancel-algos", `[{"instId":"BTC-USDT","algoClOrdId":"client802"},{"instId":"BTC-USDT","algoId":"803"}]`,
			[]map[string]string{{"algoId": "802", "sCode": "0", "sMsg": ""}, {"algoId": "803", "sCode": "0", "sMsg": ""}})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, ClientOrderID: "client802"},
			{AssetType: asset.Spot, Pair: mainPair, Type: order.OCO, OrderID: "803"},
		})
		require.NoError(t, err, "CancelBatchOrders must cancel algo orders by client order ID")
		assert.Equal(t, map[string]string{"802": order.PendingCancel.String(), "803": order.PendingCancel.String()}, resp.Status, "each algo cancel should be reported under its algo ID")
	})

	t.Run("acts for one account when the exchange's credentials change mid-batch", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var requests []string
		var ex atomic.Pointer[Exchange]
		mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requests = append(requests, r.URL.Path+" "+r.Header.Get("OK-ACCESS-KEY"))
			mu.Unlock()
			switch r.URL.Path {
			case "/trade/cancel-batch-orders":
				ex.Load().SetCredentials(secondaryAccount())
				writeOKXData(t, w, []map[string]string{{"ordId": "1001", "sCode": "0"}})
			case "/trade/cancel-algos":
				writeOKXData(t, w, []map[string]string{{"algoId": "2001", "sCode": "0"}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		ex.Store(mock)
		mock.SetCredentials(primaryAccount())
		resp, err := mock.CancelBatchOrders(t.Context(), []order.Cancel{
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "1001"},
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "2001", Type: order.Trigger},
		})
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, map[string]string{"1001": order.PendingCancel.String(), "2001": order.PendingCancel.String()}, resp.Status, "both orders should be cancelled")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/trade/cancel-batch-orders primary-key", "/trade/cancel-algos primary-key"}, requests, "both cancels should act for the account the batch started with")
	})

	t.Run("accounts for every requested order", func(t *testing.T) {
		t.Parallel()
		for _, algo := range []bool{false, true} {
			path, idKey, orderType := "/trade/cancel-batch-orders", "ordId", order.Limit
			if algo {
				path, idKey, orderType = "/trade/cancel-algos", "algoId", order.Trigger
			}
			for _, tc := range []struct {
				name string
				data string
				exp  map[string]string
				err  error
			}{
				{name: "no rows", data: `null`, exp: map[string]string{}, err: common.ErrInvalidResponse},
				{name: "a null row", data: `[null]`, exp: map[string]string{}, err: common.ErrInvalidResponse},
				{name: "an empty row", data: `[{}]`, exp: map[string]string{}, err: common.ErrInvalidResponse},
				{name: "an empty list", data: `[]`, exp: map[string]string{}, err: common.ErrInvalidResponse},
				{name: "one order's row", data: `[{"` + idKey + `":"1001","sCode":"0"}]`, exp: map[string]string{"1001": order.PendingCancel.String()}, err: common.ErrInvalidResponse},
				{name: "a row for another order", data: `[{"` + idKey + `":"9999","sCode":"0"},{"` + idKey + `":"1002","sCode":"0"}]`, exp: map[string]string{"1002": order.PendingCancel.String()}, err: common.ErrInvalidResponse},
				{name: "a repeated row", data: `[{"` + idKey + `":"1001","sCode":"0"},{"` + idKey + `":"1001","sCode":"0"}]`, exp: map[string]string{"1001": order.PendingCancel.String()}, err: common.ErrInvalidResponse},
				{name: "every order's row", data: `[{"` + idKey + `":"1002","sCode":"0"},{"` + idKey + `":"1001","sCode":"0"}]`, exp: map[string]string{"1001": order.PendingCancel.String(), "1002": order.PendingCancel.String()}},
				{name: "a row without a status code", data: `[{"` + idKey + `":"1001"},{"` + idKey + `":"1002","sCode":"0"}]`, exp: map[string]string{"1002": order.PendingCancel.String()}, err: errMissingStatusCode},
				{name: "a row with a null status code", data: `[{"` + idKey + `":"1001","sCode":null},{"` + idKey + `":"1002","sCode":"0"}]`, exp: map[string]string{"1002": order.PendingCancel.String()}, err: errMissingStatusCode},
			} {
				t.Run(fmt.Sprintf("algo %t %s", algo, tc.name), func(t *testing.T) {
					t.Parallel()
					ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path != path {
							t.Errorf("unexpected request path %s", r.URL.Path)
							http.NotFound(w, r)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
						assert.NoError(t, err, "writing the reply should not error")
					}))
					resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
						{AssetType: asset.Spot, Pair: mainPair, OrderID: "1001", Type: orderType},
						{AssetType: asset.Spot, Pair: mainPair, OrderID: "1002", Type: orderType},
					})
					require.ErrorIs(t, err, tc.err, "CancelBatchOrders must report every requested order the reply leaves unaccounted for")
					require.NotNil(t, resp, "CancelBatchOrders must return the verified results")
					assert.Equal(t, tc.exp, resp.Status, "CancelBatchOrders should report only the verified results")
				})
			}
		}
	})

	t.Run("keys results by exchange order ID only", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			data   string
			exp    map[string]string
			err    error
			reason string
		}{
			{name: "the order ID OKX resolved", data: `[{"clOrdId":"client1","ordId":"1001","sCode":"0"}]`, exp: map[string]string{"1001": order.PendingCancel.String()}},
			{name: "no order ID", data: `[{"clOrdId":"client1","ordId":"","sCode":"0"}]`, exp: map[string]string{}, err: common.ErrInvalidResponse},
			{name: "a failure without an order ID", data: `[{"clOrdId":"client1","ordId":"","sCode":"51400","sMsg":"Order cancellation failed as the order has been filled, canceled or does not exist"}]`, exp: map[string]string{}, reason: "51400"},
			{name: "another client order ID", data: `[{"clOrdId":"client2","ordId":"1002","sCode":"0"}]`, exp: map[string]string{}, err: common.ErrInvalidResponse},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot, Pair: mainPair, ClientOrderID: "client1"}})
				if tc.reason != "" {
					require.ErrorContains(t, err, tc.reason, "CancelBatchOrders must report the failure it cannot key by order ID")
				} else {
					require.ErrorIs(t, err, tc.err, "CancelBatchOrders must report a result it cannot key by order ID")
				}
				require.NotNil(t, resp, "CancelBatchOrders must return the verified results")
				assert.Equal(t, tc.exp, resp.Status, "CancelBatchOrders should key results by exchange order ID only")
			})
		}
	})

	t.Run("attributes an algo cancel result to a client order ID only when one remains", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, code, data string
			exp              map[string]string
			err              error
			errText          string
		}{
			{
				name: "a failure without an algo ID before the other order's acceptance", code: "2",
				data: `[{"algoId":"","sCode":"51400","sMsg":"order does not exist"},{"algoId":"1001","sCode":"0"}]`,
				exp:  map[string]string{"1001": order.PendingCancel.String()}, err: errPartialSuccess,
				errText: `cancelling one of client order ID "clientOne", client order ID "clientTwo": status code: ` + "`51400`",
			},
			{
				name: "one algo ID for both orders", code: "0",
				data: `[{"algoId":"1001","sCode":"0"},{"algoId":"1001","sCode":"0"}]`,
				exp:  map[string]string{}, err: errDuplicateReplyOrderID, errText: `order ID "1001"`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/cancel-algos" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"code":"` + tc.code + `","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
					{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, ClientOrderID: "clientOne"},
					{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, ClientOrderID: "clientTwo"},
				})
				require.ErrorIs(t, err, tc.err, "CancelBatchOrders must report a result it cannot attribute")
				assert.ErrorContains(t, err, tc.errText, "the error should name every order the result may answer")
				require.NotNil(t, resp, "CancelBatchOrders must return the verified results")
				assert.Equal(t, tc.exp, resp.Status, "CancelBatchOrders should record only the results it can verify")
			})
		}
	})

	t.Run("attempts the algo orders after the ordinary orders fail", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			ordinary func(http.ResponseWriter)
			exp      map[string]string
			err      error
		}{
			{
				name: "a partial success",
				ordinary: func(w http.ResponseWriter) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"code":"2","msg":"","data":[{"ordId":"1001","sCode":"0"},{"ordId":"1002","sCode":"51400","sMsg":"already cancelled"}]}`))
				},
				exp: map[string]string{"1001": order.PendingCancel.String(), "1002": "already cancelled", "2001": order.PendingCancel.String()},
				err: errPartialSuccess,
			},
			{
				name:     "a failed request",
				ordinary: func(w http.ResponseWriter) { http.Error(w, "unavailable", http.StatusBadGateway) },
				exp:      map[string]string{"2001": order.PendingCancel.String()},
				err:      request.ErrAuthRequestFailed,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var paths []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					paths = append(paths, r.URL.Path)
					mu.Unlock()
					switch r.URL.Path {
					case "/trade/cancel-batch-orders":
						tc.ordinary(w)
					case "/trade/cancel-algos":
						writeOKXData(t, w, []map[string]string{{"algoId": "2001", "sCode": "0"}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
					{AssetType: asset.Spot, Pair: mainPair, OrderID: "2001", Type: order.Trigger},
					{AssetType: asset.Spot, Pair: mainPair, OrderID: "1001"},
					{AssetType: asset.Spot, Pair: mainPair, OrderID: "1002"},
				})
				require.ErrorIs(t, err, tc.err, "CancelBatchOrders must report the ordinary orders' failure")
				require.NotNil(t, resp, "CancelBatchOrders must return both groups' results")
				assert.Equal(t, tc.exp, resp.Status, "CancelBatchOrders should report both groups' results")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, []string{"/trade/cancel-batch-orders", "/trade/cancel-algos"}, paths, "the algo orders should be cancelled after the ordinary orders fail")
			})
		}
	})

	t.Run("a dead context stops before the algo orders", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var mu sync.Mutex
		var paths []string
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			cancel()
			writeOKXData(t, w, []map[string]string{{"ordId": "1001", "sCode": "0"}})
		}))
		_, err := ex.CancelBatchOrders(ctx, []order.Cancel{
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "1001"},
			{AssetType: asset.Spot, Pair: mainPair, OrderID: "2001", Type: order.Trigger},
		})
		require.ErrorIs(t, err, context.Canceled, "CancelBatchOrders must report the dead context")
		assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()), "CancelBatchOrders should report the context error once")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/trade/cancel-batch-orders"}, paths, "no algo cancel should be sent once the context is dead")
	})
}

func TestUnreadOrders(t *testing.T) {
	t.Parallel()
	seen := make(common.Seen[string])
	first := []OrderDetailResponse{{OrderID: "1", ClientOrderID: "a"}, {OrderID: "1", ClientOrderID: "b"}, {OrderID: "2"}}
	assert.Equal(t, []OrderDetailResponse{{OrderID: "1", ClientOrderID: "a"}, {OrderID: "2"}}, unreadOrders(seen, first, orderDetailID), "unreadOrders should keep the first row of an order its page repeats")
	second := []OrderDetailResponse{{OrderID: "3"}, {OrderID: "2"}, {OrderID: "4"}}
	assert.Equal(t, []OrderDetailResponse{{OrderID: "3"}, {OrderID: "4"}}, unreadOrders(seen, second, orderDetailID), "unreadOrders should drop the orders an earlier page returned")
	assert.Equal(t, common.Seen[string]{"1": true, "2": true, "3": true, "4": true}, seen, "unreadOrders should record every order ID it was given")
	assert.Empty(t, unreadOrders(seen, []OrderDetailResponse{{OrderID: "4"}}, orderDetailID), "unreadOrders should return nothing for a page of read orders")
	spreads := []SpreadOrderDetailResponse{{OrderID: "4"}, {OrderID: "5"}}
	assert.Equal(t, []SpreadOrderDetailResponse{{OrderID: "5"}}, unreadOrders(seen, spreads, spreadOrderDetailID), "unreadOrders should read spread order IDs")

	t.Run("an order repeated within a page", func(t *testing.T) {
		t.Parallel()
		page := []OrderDetailResponse{{OrderID: "102"}, {OrderID: "99"}, {OrderID: "99"}, {OrderID: "98"}}
		assert.Equal(t, []OrderDetailResponse{{OrderID: "102"}, {OrderID: "99"}, {OrderID: "98"}}, unreadOrders(make(common.Seen[string]), page, orderDetailID), "unreadOrders should return an order its page repeats once")
	})

	t.Run("algo orders", func(t *testing.T) {
		t.Parallel()
		page := []AlgoOrderResponse{{AlgoID: "81"}, {AlgoID: "81", OrderID: "82"}, {AlgoID: "80"}}
		assert.Equal(t, []AlgoOrderResponse{{AlgoID: "81"}, {AlgoID: "80"}}, unreadOrders(make(common.Seen[string]), page, algoOrderID), "unreadOrders should read algo orders by their algo IDs")
	})
}

func TestAlgoOrderID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "81", algoOrderID(&AlgoOrderResponse{AlgoID: "81", OrderID: "82", AlgoClientOrderID: "client81"}), "algoOrderID should return the algo ID")
}

func TestOrderDetailID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *OrderDetailResponse
		exp  string
	}{
		{name: "order ID", o: &OrderDetailResponse{OrderID: "680800019749904384", ClientOrderID: "client1"}, exp: "680800019749904384"},
		{name: "client order ID only", o: &OrderDetailResponse{ClientOrderID: "client1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, orderDetailID(tc.o), "orderDetailID should return the order ID")
		})
	}
}

func TestSpreadOrderDetailID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *SpreadOrderDetailResponse
		exp  string
	}{
		{name: "order ID", o: &SpreadOrderDetailResponse{OrderID: "2510789768709120", ClientOrderID: "b16"}, exp: "2510789768709120"},
		{name: "client order ID only", o: &SpreadOrderDetailResponse{ClientOrderID: "b16"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, spreadOrderDetailID(tc.o), "spreadOrderDetailID should return the order ID")
		})
	}
}

func TestPendingOrdersToCancel(t *testing.T) {
	t.Parallel()
	row := func(i int) map[string]string {
		return map[string]string{
			"instId": mainPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i), "clOrdId": fmt.Sprintf("client-%03d", i), "ordType": orderLimit,
			"side": "buy", "state": "live", "sz": "1", "px": "1", "accFillSz": "0", "cTime": "1700000000000", "uTime": "1700000000000",
		}
	}
	for _, tc := range append(pagerCases(row), stalledPagerCases(row)...) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, pagerHandler(t, "/trade/orders-pending", "after", tc.pages))
			scope, err := ex.newCancelAllScope(&order.Cancel{AssetType: asset.Spot})
			require.NoError(t, err, "newCancelAllScope must not error")
			pending, err := ex.pendingOrdersToCancel(t.Context(), scope)
			if tc.incomplete {
				require.ErrorIs(t, err, errListingStalled, "pendingOrdersToCancel must report a listing which stops advancing")
			} else {
				require.NoError(t, err, "pendingOrdersToCancel must not error")
			}
			exp := make([]CancelOrderRequest, 0, len(tc.exp))
			for _, id := range tc.exp {
				exp = append(exp, CancelOrderRequest{InstrumentID: mainPair.String(), OrderID: id, ClientOrderID: "client-" + strings.TrimPrefix(id, "ORD-")})
			}
			assert.Equal(t, exp, pending, "pendingOrdersToCancel should select every order of every page once, in listing order")
		})
	}

	t.Run("leaves an order listed for another instrument type", func(t *testing.T) {
		t.Parallel()
		rows := []map[string]string{
			{"instType": instTypeSpot, "instId": mainPair.String(), "ordId": "1001", "ordType": orderLimit, "side": "buy"},
			{"instType": instTypeMargin, "instId": mainPair.String(), "ordId": "9002", "ordType": orderLimit, "side": "buy"},
			{"instType": "spot", "instId": mainPair.String(), "ordId": "1003", "ordType": orderLimit, "side": "buy"},
			{"instId": mainPair.String(), "ordId": "1004", "ordType": orderLimit, "side": "buy"},
		}
		for _, tc := range []struct {
			name   string
			cancel *order.Cancel
			exp    []string
			err    error
		}{
			{name: "every order of a spot pair", cancel: &order.Cancel{AssetType: asset.Spot, Pair: mainPair}, exp: []string{"1001", "1003", "1004"}, err: errListedOtherInstrumentType},
			{name: "the other instrument type's order by its ID", cancel: &order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "9002"}, exp: []string{}, err: errListedOtherInstrumentType},
			{name: "every order of every asset", cancel: &order.Cancel{AssetType: asset.All, Pair: mainPair}, exp: []string{"1001", "9002", "1003", "1004"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/orders-pending" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					writeOKXData(t, w, rows)
				}))
				scope, err := ex.newCancelAllScope(tc.cancel)
				require.NoError(t, err, "newCancelAllScope must not error")
				pending, err := ex.pendingOrdersToCancel(t.Context(), scope)
				require.ErrorIs(t, err, tc.err, "pendingOrdersToCancel must report an order of another instrument type than it listed")
				if tc.err != nil {
					assert.ErrorContains(t, err, `order "9002" is MARGIN`, "the error should name the order left alone")
				}
				exp := make([]CancelOrderRequest, 0, len(tc.exp))
				for _, id := range tc.exp {
					exp = append(exp, CancelOrderRequest{InstrumentID: mainPair.String(), OrderID: id})
				}
				assert.Equal(t, exp, pending, "pendingOrdersToCancel should select the orders of the requested instrument type alone")
			})
		}
	})
}

func TestRecordCancelOutcomes(t *testing.T) {
	t.Parallel()
	errFilled := errors.New("order filled")
	pending := order.PendingCancel.String()
	for _, tc := range []struct {
		name          string
		targets       []cancelTarget
		outcomes      []cancelOutcome
		byElimination bool
		exp           map[string]string
		err           error
		errNames      string
	}{
		{name: "by exchange order ID", targets: []cancelTarget{{orderID: "1001"}}, outcomes: []cancelOutcome{{orderID: "1001"}}, exp: map[string]string{"1001": pending}},
		{name: "a failed cancel's message", targets: []cancelTarget{{orderID: "1001"}}, outcomes: []cancelOutcome{{orderID: "1001", err: errFilled, message: "filled"}}, exp: map[string]string{"1001": "filled"}},
		{name: "by echoed client order ID", targets: []cancelTarget{{clientOrderID: "c1"}}, outcomes: []cancelOutcome{{orderID: "1001", clientOrderID: "c1"}}, exp: map[string]string{"1001": pending}},
		{
			name: "a client order ID echo for an order requested by exchange order ID", targets: []cancelTarget{{orderID: "1001", clientOrderID: "c1"}}, outcomes: []cancelOutcome{{clientOrderID: "c1"}},
			exp: map[string]string{}, err: common.ErrInvalidResponse, errNames: `order ID "1001"`,
		},
		{
			name: "by elimination", targets: []cancelTarget{{orderID: "1001"}, {clientOrderID: "c2"}}, outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "2002"}}, byElimination: true,
			exp: map[string]string{"1001": pending, "2002": pending},
		},
		{
			name: "by elimination of several", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c2"}, {clientOrderID: "c3"}},
			outcomes: []cancelOutcome{{orderID: "2001"}, {orderID: "2002", err: errFilled, message: "filled"}, {err: errFilled, message: "filled"}}, byElimination: true,
			exp: map[string]string{"2001": pending, "2002": "filled"}, err: errFilled, errNames: `cancelling one of client order ID "c1", client order ID "c2", client order ID "c3": order filled`,
		},
		{
			name: "a failure without an order ID before another order's acceptance", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{err: errFilled, message: "filled"}, {orderID: "2001"}}, byElimination: true,
			exp: map[string]string{"2001": pending}, err: errFilled, errNames: `cancelling one of client order ID "c1", client order ID "c2": order filled`,
		},
		{
			name: "a failure without an order ID by elimination of one", targets: []cancelTarget{{orderID: "1001"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "1001"}, {err: errFilled, message: "filled"}}, byElimination: true,
			exp: map[string]string{"1001": pending}, err: errFilled, errNames: `cancelling client order ID "c2": order filled`,
		},
		{
			name: "an acceptance without an order ID among several", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001"}, {}}, byElimination: true,
			exp: map[string]string{"2001": pending}, err: common.ErrInvalidResponse, errNames: `the cancel result for one of client order ID "c1", client order ID "c2" names no order ID`,
		},
		{
			name: "one order ID for two client order IDs by elimination", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001"}, {orderID: "2001"}}, byElimination: true,
			exp: map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "2001"`,
		},
		{
			name: "one order ID echoed for two client order IDs", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001", clientOrderID: "c1"}, {orderID: "2001", clientOrderID: "c2"}},
			exp:      map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "2001"`,
		},
		{
			name: "one order ID for an echoed client order ID and another by elimination", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001", clientOrderID: "c1"}, {orderID: "2001"}}, byElimination: true,
			exp: map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "2001"`,
		},
		{
			name: "one order ID for an echoed client order ID and its repeat by elimination", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001", clientOrderID: "c1"}, {orderID: "2001"}, {orderID: "3003"}}, byElimination: true,
			exp: map[string]string{"2001": pending, "3003": pending},
		},
		{
			name: "one order ID for a repeated client order ID", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}},
			outcomes: []cancelOutcome{{orderID: "2001"}, {orderID: "2001", err: errFilled, message: "filled"}}, byElimination: true,
			exp: map[string]string{"2001": pending},
		},
		{
			name: "one order ID for a repeated client order ID among several", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001"}, {orderID: "2001"}, {orderID: "3003"}}, byElimination: true,
			exp: map[string]string{"2001": pending, "3003": pending},
		},
		{
			name: "one order ID more often than any client order ID repeats", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001"}, {orderID: "2001"}, {orderID: "2001"}}, byElimination: true,
			exp: map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "2001"`,
		},
		{
			name: "one order ID for an order ID and a client order ID", targets: []cancelTarget{{orderID: "2001"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "2001"}, {orderID: "2001", clientOrderID: "c2"}},
			exp:      map[string]string{"2001": pending},
		},
		{
			name: "one order ID echoed for an order requested by its order ID and another client order ID", targets: []cancelTarget{{orderID: "1001", clientOrderID: "c1"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "1001", clientOrderID: "c1"}, {orderID: "1001", clientOrderID: "c2"}},
			exp:      map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "1001" answers the cancels of order ID "1001", client order ID "c2"`,
		},
		{
			name: "one order ID echoed for an order requested by its order ID and its client order ID", targets: []cancelTarget{{orderID: "1001", clientOrderID: "c1"}, {clientOrderID: "c1"}},
			outcomes: []cancelOutcome{{orderID: "1001", clientOrderID: "c1"}, {orderID: "1001", clientOrderID: "c1"}},
			exp:      map[string]string{"1001": pending},
		},
		{
			name: "one order ID echoed for an accepted cancel and another client order ID's failure", targets: []cancelTarget{{orderID: "1001"}, {clientOrderID: "c2"}},
			outcomes: []cancelOutcome{{orderID: "1001", clientOrderID: "c1"}, {orderID: "1001", clientOrderID: "c2", err: errFilled, message: "filled"}},
			exp:      map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "1001"`,
		},
		{
			name: "two order IDs each answering two requests where one client order ID repeats", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c2"}, {clientOrderID: "c3"}},
			outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "1001"}, {orderID: "2002"}, {orderID: "2002"}}, byElimination: true,
			exp: map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "2002" answers the cancels of client order ID "c1", client order ID "c2", client order ID "c3"`,
		},
		{
			name: "one order ID answering two requests where one client order ID repeats", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c2"}, {clientOrderID: "c3"}},
			outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "1001"}, {orderID: "2002"}, {orderID: "3003"}}, byElimination: true,
			exp: map[string]string{"1001": pending, "2002": pending, "3003": pending},
		},
		{
			name: "two order IDs for a repeated client order ID by elimination", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}},
			outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "2002"}}, byElimination: true,
			exp: map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "2002"`,
		},
		{
			name: "two order IDs needing the one client order ID requested often enough", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c2"}, {clientOrderID: "c3"}},
			outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "1001"}, {orderID: "1001"}, {orderID: "2002"}, {orderID: "2002"}}, byElimination: true,
			exp: map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "1001"`,
		},
		{
			name: "more order IDs than client order IDs beside one answering too many requests", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}, {clientOrderID: "c2"}, {clientOrderID: "c2"}, {clientOrderID: "c3"}, {clientOrderID: "c3"}},
			outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "1001"}, {orderID: "1001"}, {orderID: "2002"}, {orderID: "3003"}, {orderID: "4004"}}, byElimination: true,
			exp: map[string]string{}, err: errDuplicateReplyOrderID, errNames: `order ID "4004"`,
		},
		{
			name: "an order ID by elimination for a client order ID echoed with another", targets: []cancelTarget{{clientOrderID: "c1"}, {clientOrderID: "c1"}},
			outcomes: []cancelOutcome{{orderID: "1001", clientOrderID: "c1"}, {orderID: "2002"}}, byElimination: true,
			exp: map[string]string{"1001": pending}, err: errDuplicateReplyOrderID, errNames: `order ID "2002"`,
		},
		{
			name: "a row without a status code", targets: []cancelTarget{{orderID: "1001"}, {orderID: "1002"}},
			outcomes: []cancelOutcome{{orderID: "1001", err: errMissingStatusCode}, {orderID: "1002"}},
			exp:      map[string]string{"1002": pending}, err: errMissingStatusCode, errNames: `cancelling order ID "1001"`,
		},
		{
			name: "a row without a status code beside its message", targets: []cancelTarget{{orderID: "1001"}},
			outcomes: []cancelOutcome{{orderID: "1001", err: errMissingStatusCode, message: "cancelled"}},
			exp:      map[string]string{}, err: errMissingStatusCode, errNames: `cancelling order ID "1001"`,
		},
		{
			name: "a repeated request's acceptance beside a row without a status code", targets: []cancelTarget{{orderID: "1001"}, {orderID: "1001"}},
			outcomes: []cancelOutcome{{orderID: "1001", err: errMissingStatusCode}, {orderID: "1001"}},
			exp:      map[string]string{"1001": pending}, err: errMissingStatusCode, errNames: `cancelling order ID "1001"`,
		},
		{
			name: "no elimination without it", targets: []cancelTarget{{orderID: "1001"}, {clientOrderID: "c2"}}, outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "2002"}},
			exp: map[string]string{"1001": pending}, err: common.ErrInvalidResponse, errNames: `client order ID "c2"`,
		},
		{
			name: "no elimination when the counts differ", targets: []cancelTarget{{clientOrderID: "c1"}}, outcomes: []cancelOutcome{{orderID: "2001"}, {orderID: "2002"}}, byElimination: true,
			exp: map[string]string{}, err: common.ErrInvalidResponse, errNames: `client order ID "c1"`,
		},
		{
			name: "no elimination of a requested order's row", targets: []cancelTarget{{orderID: "1001"}, {clientOrderID: "c2"}}, outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "1001"}}, byElimination: true,
			exp: map[string]string{"1001": pending}, err: common.ErrInvalidResponse, errNames: `client order ID "c2"`,
		},
		{
			name: "no elimination of another client order ID's row", targets: []cancelTarget{{clientOrderID: "c1"}}, outcomes: []cancelOutcome{{orderID: "2002", clientOrderID: "c2"}}, byElimination: true,
			exp: map[string]string{}, err: common.ErrInvalidResponse, errNames: `client order ID "c1"`,
		},
		{
			name: "an accounted row without an exchange order ID", targets: []cancelTarget{{clientOrderID: "c1"}}, outcomes: []cancelOutcome{{}}, byElimination: true,
			exp: map[string]string{}, err: common.ErrInvalidResponse, errNames: `client order ID "c1"`,
		},
		{
			name: "a failure without an exchange order ID", targets: []cancelTarget{{clientOrderID: "c1"}}, outcomes: []cancelOutcome{{clientOrderID: "c1", err: errFilled, message: "filled"}},
			exp: map[string]string{}, err: errFilled, errNames: `client order ID "c1"`,
		},
		{
			name: "a repeated request's acceptance before its failure", targets: []cancelTarget{{orderID: "1001"}, {orderID: "1001"}}, outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "1001", err: errFilled, message: "filled"}},
			exp: map[string]string{"1001": pending},
		},
		{
			name: "a repeated request's acceptance after its failure", targets: []cancelTarget{{orderID: "1001"}, {orderID: "1001"}}, outcomes: []cancelOutcome{{orderID: "1001", err: errFilled, message: "filled"}, {orderID: "1001"}},
			exp: map[string]string{"1001": pending},
		},
		{
			name: "a repeated request's failures", targets: []cancelTarget{{orderID: "1001"}, {orderID: "1001"}}, outcomes: []cancelOutcome{{orderID: "1001", err: errFilled, message: "filled"}, {orderID: "1001", err: errFilled, message: "cancelled"}},
			exp: map[string]string{"1001": "filled"},
		},
		{name: "no rows", targets: []cancelTarget{{orderID: "1001"}}, exp: map[string]string{}, err: common.ErrInvalidResponse, errNames: `order ID "1001"`},
		{
			name: "an unrequested row", targets: []cancelTarget{{orderID: "1001"}}, outcomes: []cancelOutcome{{orderID: "1001"}, {orderID: "9999"}},
			exp: map[string]string{"1001": pending}, err: common.ErrInvalidResponse, errNames: `"9999"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status := map[string]string{}
			err := recordCancelOutcomes(status, tc.targets, tc.outcomes, tc.byElimination)
			require.ErrorIs(t, err, tc.err, "recordCancelOutcomes must report every requested order the rows do not account for")
			if tc.errNames != "" {
				assert.ErrorContains(t, err, tc.errNames, "the error should name what the rows do not account for")
			}
			assert.Equal(t, tc.exp, status, "recordCancelOutcomes should record each accounted order by its exchange order ID")
		})
	}
}

func TestRecordOrderCancels(t *testing.T) {
	t.Parallel()
	status := map[string]string{}
	err := recordOrderCancels(status,
		[]CancelOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1001"}, {InstrumentID: mainPair.String(), ClientOrderID: "c2"}, {InstrumentID: mainPair.String(), ClientOrderID: "c3"}},
		[]*OrderActionResponse{nil, {OrderID: "1002", ClientOrderID: "c2"}, {OrderID: "1001", StatusCode: 51400, StatusMessage: "already cancelled"}, {OrderID: "1003"}})
	require.ErrorIs(t, err, common.ErrInvalidResponse, "recordOrderCancels must report the order its rows do not account for")
	assert.ErrorContains(t, err, `client order ID "c3"`, "recordOrderCancels should not attribute a row echoing no client order ID")
	assert.Equal(t, map[string]string{"1001": "already cancelled", "1002": order.PendingCancel.String()}, status, "recordOrderCancels should record each accounted order's result by its order ID")

	t.Run("a row without a status code", func(t *testing.T) {
		t.Parallel()
		var rows []*OrderActionResponse
		require.NoError(t, json.Unmarshal([]byte(`[{"ordId":"1001","sMsg":""},{"ordId":"1002","sCode":"0"}]`), &rows), "the reply rows must decode")
		status := map[string]string{}
		err := recordOrderCancels(status, []CancelOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1001"}, {InstrumentID: mainPair.String(), OrderID: "1002"}}, rows)
		require.ErrorIs(t, err, errMissingStatusCode, "recordOrderCancels must report a row which states no outcome")
		assert.Equal(t, map[string]string{"1002": order.PendingCancel.String()}, status, "recordOrderCancels should record only the confirmed cancel")
	})

	t.Run("a failed row without a status message", func(t *testing.T) {
		t.Parallel()
		status := map[string]string{}
		err := recordOrderCancels(status, []CancelOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1001"}}, []*OrderActionResponse{{OrderID: "1001", StatusCode: 51400}})
		require.NoError(t, err, "recordOrderCancels must account for the order")
		assert.Equal(t, map[string]string{"1001": "status code: `51400` status message: \"\""}, status, "recordOrderCancels should record a failure's status code when it states no message")
	})
}

func TestRecordAlgoCancels(t *testing.T) {
	t.Parallel()
	status := map[string]string{}
	err := recordAlgoCancels(status,
		[]CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "2001"}, {InstrumentID: mainPair.String(), AlgoClientOrderID: "a2"}, {InstrumentID: mainPair.String(), AlgoClientOrderID: "a3"}},
		[]AlgoOrderActionResponse{{AlgoID: "2003"}, {AlgoID: "2002", AlgoClientOrderID: "a2"}, {AlgoID: "2001", StatusCode: 51000, StatusMessage: "the algo order does not exist"}})
	require.NoError(t, err, "recordAlgoCancels must account for every order")
	assert.Equal(t, map[string]string{"2001": "the algo order does not exist", "2002": order.PendingCancel.String(), "2003": order.PendingCancel.String()}, status, "recordAlgoCancels should record each order's result by its algo ID, attributing a row echoing no client order ID by elimination")

	t.Run("a row without a status code", func(t *testing.T) {
		t.Parallel()
		for _, reply := range []string{`[{"algoId":"2001"},{"algoId":"2002","sCode":"0"}]`, `[{"algoId":"2001","sCode":null},{"algoId":"2002","sCode":"0"}]`} {
			var rows []AlgoOrderActionResponse
			require.NoError(t, json.Unmarshal([]byte(reply), &rows), "the reply rows must decode")
			status := map[string]string{}
			err := recordAlgoCancels(status, []CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "2001"}, {InstrumentID: mainPair.String(), AlgoID: "2002"}}, rows)
			require.ErrorIsf(t, err, errMissingStatusCode, "recordAlgoCancels must report a row which states no outcome in %s", reply)
			assert.Equalf(t, map[string]string{"2002": order.PendingCancel.String()}, status, "recordAlgoCancels should record only the confirmed cancel in %s", reply)
		}
	})

	t.Run("one algo ID echoed for an order requested by its algo ID and another client order ID", func(t *testing.T) {
		t.Parallel()
		var rows []AlgoOrderActionResponse
		require.NoError(t, json.Unmarshal([]byte(`[{"algoId":"1001","algoClOrdId":"a1","sCode":"0"},{"algoId":"1001","algoClOrdId":"a2","sCode":"0"}]`), &rows), "the reply rows must decode")
		status := map[string]string{}
		err := recordAlgoCancels(status, []CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "1001", AlgoClientOrderID: "a1"}, {InstrumentID: mainPair.String(), AlgoClientOrderID: "a2"}}, rows)
		require.ErrorIs(t, err, errDuplicateReplyOrderID, "recordAlgoCancels must report an algo ID the rows echo for two client order IDs")
		assert.Empty(t, status, "recordAlgoCancels should record no cancel of an algo ID the rows give two client order IDs")
	})
}

func TestCancelTargetString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, `order ID "1001"`, cancelTarget{orderID: "1001", clientOrderID: "c1"}.String(), "a target with an order ID should be named by it")
	assert.Equal(t, `client order ID "c1"`, cancelTarget{clientOrderID: "c1"}.String(), "a target without an order ID should be named by its client order ID")
}

func TestWithContextError(t *testing.T) {
	t.Parallel()
	errBatch := errors.New("batch failed")
	live := t.Context()
	dead, cancel := context.WithCancel(t.Context())
	cancel()
	assert.NoError(t, withContextError(live, nil), "a live context should add no error")
	assert.Equal(t, errBatch, withContextError(live, errBatch), "a live context should leave the errors as they are")
	err := withContextError(dead, errBatch)
	assert.ErrorIs(t, err, errBatch, "a dead context should keep the errors")
	assert.ErrorIs(t, err, context.Canceled, "a dead context should add its error")
	err = withContextError(dead, fmt.Errorf("sending: %w", context.Canceled))
	assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()), "a dead context should add its error once")
}

func TestCancelAllScopeSelects(t *testing.T) {
	t.Parallel()
	type listed struct {
		instrumentID, orderID, clientOrderID string
		side                                 order.Side
		tradeMode, orderType                 string
	}
	match := listed{instrumentID: "BTC-USDT", orderID: "1001", clientOrderID: "c1", side: order.Buy, tradeMode: TradeModeCross, orderType: orderPostOnly}
	scope := &cancelAllScope{instrumentID: "BTC-USDT", orderID: "1001", clientOrderID: "c1", side: order.Buy, tradeMode: TradeModeCross}
	for _, tc := range []struct {
		name  string
		order func(*listed)
		types string
		exp   bool
	}{
		{name: "every filter", types: "limit,post_only", exp: true},
		{name: "no order type filter", exp: true},
		{name: "an instrument in another case", order: func(l *listed) { l.instrumentID = "btc-usdt" }, exp: true},
		{name: "another instrument", order: func(l *listed) { l.instrumentID = "ETH-USDT" }},
		{name: "another order ID", order: func(l *listed) { l.orderID = "1002" }},
		{name: "another client order ID", order: func(l *listed) { l.clientOrderID = "c2" }},
		{name: "another side", order: func(l *listed) { l.side = order.Sell }},
		{name: "another trade mode", order: func(l *listed) { l.tradeMode = TradeModeIsolated }},
		{name: "another order type", types: "limit,fok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := match
			if tc.order != nil {
				tc.order(&l)
			}
			assert.Equal(t, tc.exp, scope.selects(l.instrumentID, l.orderID, l.clientOrderID, l.side, l.tradeMode, l.orderType, tc.types), "selects should select an order only when every filter matches")
		})
	}
	assert.True(t, new(cancelAllScope).selects("ETH-USDT", "1002", "c2", order.Sell, TradeModeCash, orderLimit, ""), "an unfiltered scope should select every order")
}

func TestCancelAllScopeIDScoped(t *testing.T) {
	t.Parallel()
	assert.False(t, new(cancelAllScope).idScoped(), "a scope without IDs should not be ID scoped")
	assert.True(t, (&cancelAllScope{orderID: "1001"}).idScoped(), "an order ID should scope to one order")
	assert.True(t, (&cancelAllScope{clientOrderID: "c1"}).idScoped(), "a client order ID should scope to one order")
}

func TestCancelInBatches(t *testing.T) {
	t.Parallel()
	requests := func(n int) []CancelOrderRequest {
		params := make([]CancelOrderRequest, n)
		for i := range params {
			params[i] = CancelOrderRequest{InstrumentID: mainPair.String(), OrderID: strconv.Itoa(i + 1)}
		}
		return params
	}
	orderIDs := func(from, to int) []string {
		ids := make([]string, 0, to-from+1)
		for i := from; i <= to; i++ {
			ids = append(ids, strconv.Itoa(i))
		}
		return ids
	}
	pending := func(ids []string) map[string]string {
		statuses := make(map[string]string, len(ids))
		for _, id := range ids {
			statuses[id] = order.PendingCancel.String()
		}
		return statuses
	}
	accepted := func(batch []CancelOrderRequest) []*OrderActionResponse {
		rows := make([]*OrderActionResponse, len(batch))
		for i := range batch {
			rows[i] = &OrderActionResponse{OrderID: batch[i].OrderID}
		}
		return rows
	}
	errBatch := errors.New("batch failed")
	for _, tc := range []struct {
		name    string
		params  []CancelOrderRequest
		cancel  func(context.CancelFunc, int, []CancelOrderRequest) ([]*OrderActionResponse, error)
		batches [][]string
		exp     map[string]string
		err     error
	}{
		{name: "no orders", exp: map[string]string{}},
		{name: "one order", params: requests(1), batches: [][]string{{"1"}}, exp: pending([]string{"1"})},
		{name: "a full batch", params: requests(20), batches: [][]string{orderIDs(1, 20)}, exp: pending(orderIDs(1, 20))},
		{name: "one order past a full batch", params: requests(21), batches: [][]string{orderIDs(1, 20), {"21"}}, exp: pending(orderIDs(1, 21))},
		{name: "several batches", params: requests(41), batches: [][]string{orderIDs(1, 20), orderIDs(21, 40), {"41"}}, exp: pending(orderIDs(1, 41))},
		{
			name: "each order's result", params: requests(4), batches: [][]string{orderIDs(1, 4)},
			cancel: func(_ context.CancelFunc, _ int, _ []CancelOrderRequest) ([]*OrderActionResponse, error) {
				return []*OrderActionResponse{{OrderID: "1"}, {OrderID: "2", StatusCode: 51400, StatusMessage: "Cancellation failed as the order has been filled"}, nil, {StatusCode: 51400}}, nil
			},
			exp: map[string]string{"1": order.PendingCancel.String(), "2": "Cancellation failed as the order has been filled"},
			err: common.ErrInvalidResponse,
		},
		{
			name: "a failed batch does not stop the next", params: requests(25), batches: [][]string{orderIDs(1, 20), orderIDs(21, 25)},
			cancel: func(_ context.CancelFunc, call int, batch []CancelOrderRequest) ([]*OrderActionResponse, error) {
				if call == 0 {
					return accepted(batch), errBatch
				}
				return accepted(batch), nil
			},
			exp: pending(orderIDs(21, 25)),
			err: errBatch,
		},
		{
			name: "a partial success records the batch's results", params: requests(2), batches: [][]string{{"1", "2"}},
			cancel: func(_ context.CancelFunc, _ int, _ []CancelOrderRequest) ([]*OrderActionResponse, error) {
				return []*OrderActionResponse{{OrderID: "1"}, {OrderID: "2", StatusCode: 51400, StatusMessage: "Cancellation failed"}}, errPartialSuccess
			},
			exp: map[string]string{"1": order.PendingCancel.String(), "2": "Cancellation failed"},
			err: errPartialSuccess,
		},
		{
			name: "a cancelled context stops before the next batch", params: requests(25), batches: [][]string{orderIDs(1, 20)},
			cancel: func(cancel context.CancelFunc, _ int, batch []CancelOrderRequest) ([]*OrderActionResponse, error) {
				cancel()
				return accepted(batch), nil
			},
			exp: pending(orderIDs(1, 20)),
			err: context.Canceled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var batches [][]string
			resp, err := cancelInBatches(ctx, tc.params, func(_ context.Context, batch []CancelOrderRequest) ([]*OrderActionResponse, error) {
				ids := make([]string, len(batch))
				for i := range batch {
					ids[i] = batch[i].OrderID
				}
				batches = append(batches, ids)
				if tc.cancel == nil {
					return accepted(batch), nil
				}
				return tc.cancel(cancel, len(batches)-1, batch)
			})
			require.ErrorIs(t, err, tc.err, "cancelInBatches must return the expected error")
			assert.Equal(t, tc.batches, batches, "cancelInBatches should send the orders in order, at most 20 a batch")
			assert.Equal(t, order.CancelAllResponse{Status: tc.exp}, resp, "cancelInBatches should record each cancelled order's status")
		})
	}

	t.Run("a context cancelled before the first batch", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		resp, err := cancelInBatches(ctx, requests(2), func(context.Context, []CancelOrderRequest) ([]*OrderActionResponse, error) {
			assert.Fail(t, "no batch should be sent once the context is cancelled")
			return nil, nil
		})
		require.ErrorIs(t, err, context.Canceled, "cancelInBatches must return the context error")
		assert.Equal(t, order.CancelAllResponse{Status: map[string]string{}}, resp, "cancelInBatches should record no status")
	})

	t.Run("a batch failing with the context error reports it once", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		_, err := cancelInBatches(ctx, requests(25), func(ctx context.Context, _ []CancelOrderRequest) ([]*OrderActionResponse, error) {
			cancel()
			return nil, fmt.Errorf("cancelling the batch: %w", ctx.Err())
		})
		require.ErrorIs(t, err, context.Canceled, "cancelInBatches must return the context error")
		assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()), "cancelInBatches should report the context error once")
	})
}

// pendingBook is a mock venue's pending orders: it lists them with OKX's instType, instId and ordType filters, and
// records each listing and cancel it receives over REST or the websocket
type pendingBook struct {
	mu          sync.Mutex
	ordinary    []map[string]string
	algo        []map[string]string
	listings    []string
	cancelled   []string
	algoBatches [][]string
}

// selected returns the rows the listing query selects
func selected(rows []map[string]string, query url.Values) []map[string]string {
	out := []map[string]string{}
	for _, row := range rows {
		if (query.Get("instType") != "" && query.Get("instType") != row["instType"]) ||
			(query.Get("instId") != "" && query.Get("instId") != row["instId"]) ||
			(query.Get("ordType") != "" && !slices.Contains(strings.Split(query.Get("ordType"), ","), row["ordType"])) {
			continue
		}
		out = append(out, row)
	}
	return out
}

// rest answers the book's REST listings and cancels
func (b *pendingBook) rest(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		switch r.URL.Path {
		case "/trade/orders-pending":
			b.listings = append(b.listings, r.URL.Path+" "+r.URL.Query().Encode())
			writeOKXData(t, w, selected(b.ordinary, r.URL.Query()))
		case "/trade/orders-algo-pending":
			b.listings = append(b.listings, r.URL.Path+" "+r.URL.Query().Encode())
			writeOKXData(t, w, selected(b.algo, r.URL.Query()))
		case "/trade/cancel-batch-orders":
			var reqs []CancelOrderRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&reqs), "the cancel request body should decode") {
				return
			}
			rows := make([]map[string]string, 0, len(reqs))
			for x := range reqs {
				b.cancelled = append(b.cancelled, reqs[x].OrderID)
				rows = append(rows, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
			}
			writeOKXData(t, w, rows)
		case "/trade/cancel-algos":
			var reqs []CancelAlgoOrderRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&reqs), "the algo cancel request body should decode") {
				return
			}
			rows := make([]map[string]string, 0, len(reqs))
			batch := make([]string, 0, len(reqs))
			for x := range reqs {
				batch = append(batch, reqs[x].AlgoID)
				rows = append(rows, map[string]string{"algoId": reqs[x].AlgoID, "sCode": "0"})
			}
			b.algoBatches = append(b.algoBatches, batch)
			writeOKXData(t, w, rows)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}
}

// websocket answers the book's websocket batch cancels
func (b *pendingBook) websocket(t *testing.T) func(string, json.RawMessage) (string, any) {
	t.Helper()
	return func(op string, args json.RawMessage) (string, any) {
		if op != "batch-cancel-orders" {
			t.Errorf("unexpected websocket operation %s", op)
			return "60013", nil
		}
		var reqs []CancelOrderRequest
		if !assert.NoError(t, json.Unmarshal(args, &reqs), "the websocket cancel arguments should decode") {
			return "60013", nil
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		rows := make([]map[string]string, 0, len(reqs))
		for x := range reqs {
			b.cancelled = append(b.cancelled, reqs[x].OrderID)
			rows = append(rows, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
		}
		return "0", rows
	}
}

// sent returns and resets what the book received
func (b *pendingBook) sent() (listings, cancelled []string, algoBatches [][]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	listings, cancelled, algoBatches = b.listings, b.cancelled, b.algoBatches
	b.listings, b.cancelled, b.algoBatches = nil, nil, nil
	return listings, cancelled, algoBatches
}

func TestNewCancelAllScope(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, true), "margin must enable")
	spotAlgos := []string{"conditional,oco", "trigger", "move_order_stop", "twap", "iceberg", "smart_iceberg"}
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
		exp    *cancelAllScope
		err    error
	}{
		{
			name: "every order of a spot pair", cancel: &order.Cancel{AssetType: asset.Spot, Pair: mainPair},
			exp: &cancelAllScope{instrumentType: "SPOT", instrumentID: "BTC-USDT", ordinary: true, algoTypes: spotAlgos},
		},
		{
			name: "a differently delimited pair", cancel: &order.Cancel{AssetType: asset.Spot, Pair: currency.NewPairWithDelimiter("btc", "usdt", "/")},
			exp: &cancelAllScope{instrumentType: "SPOT", instrumentID: "BTC-USDT", ordinary: true, algoTypes: spotAlgos},
		},
		{
			name: "every order of a perpetual swap side", cancel: &order.Cancel{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Short},
			exp: &cancelAllScope{instrumentType: "SWAP", instrumentID: "BTC-USDT-SWAP", side: order.Sell, ordinary: true, algoTypes: []string{"conditional,oco", "trigger", "move_order_stop", "twap", "iceberg", "smart_iceberg", "chase"}},
		},
		{name: "every option order", cancel: &order.Cancel{AssetType: asset.Options}, exp: &cancelAllScope{instrumentType: "OPTION", ordinary: true}},
		{name: "every order of every asset", cancel: &order.Cancel{}, exp: &cancelAllScope{ordinary: true, algoTypes: []string{"conditional,oco", "trigger", "move_order_stop", "twap", "iceberg", "smart_iceberg", "chase"}}},
		{name: "every order of all assets", cancel: &order.Cancel{AssetType: asset.All, Pair: mainPair}, exp: &cancelAllScope{instrumentID: "BTC-USDT", ordinary: true, algoTypes: []string{"conditional,oco", "trigger", "move_order_stop", "twap", "iceberg", "smart_iceberg", "chase"}}},
		{name: "an algo order type", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.Trigger}, exp: &cancelAllScope{instrumentType: "SPOT", algoTypes: []string{"trigger"}}},
		{name: "conditional orders", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.ConditionalStop}, exp: &cancelAllScope{instrumentType: "SPOT", algoTypes: []string{"conditional"}}},
		{name: "trailing stop orders", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.TrailingStop}, exp: &cancelAllScope{instrumentType: "SPOT", algoTypes: []string{"move_order_stop"}}},
		{name: "an ordinary order type", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.Limit, TimeInForce: order.PostOnly}, exp: &cancelAllScope{instrumentType: "SPOT", ordinary: true, ordinaryTypes: "post_only"}},
		{name: "a time in force of any order type", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.AnyType, TimeInForce: order.PostOnly}, exp: &cancelAllScope{instrumentType: "SPOT", ordinary: true, ordinaryTypes: "post_only,mmp_and_post_only"}},
		{name: "a time in force without an order type", cancel: &order.Cancel{AssetType: asset.Spot, TimeInForce: order.ImmediateOrCancel}, exp: &cancelAllScope{instrumentType: "SPOT", ordinary: true, ordinaryTypes: "ioc,optimal_limit_ioc"}},
		{
			name: "a margin type", cancel: &order.Cancel{AssetType: asset.Margin, Pair: mainPair, MarginType: margin.Isolated},
			exp: &cancelAllScope{instrumentType: "MARGIN", instrumentID: "BTC-USDT", tradeMode: "isolated", ordinary: true, algoTypes: spotAlgos},
		},
		{
			name: "an order's IDs", cancel: &order.Cancel{AssetType: asset.Spot, OrderID: "1001", ClientOrderID: "c1"},
			exp: &cancelAllScope{instrumentType: "SPOT", orderID: "1001", clientOrderID: "c1", ordinary: true, algoTypes: spotAlgos},
		},
		{name: "a side it cannot filter by", cancel: &order.Cancel{AssetType: asset.Spot, Side: order.ClosePosition}, err: order.ErrSideIsInvalid},
		{name: "an invalid asset", cancel: &order.Cancel{AssetType: asset.Item(255), Pair: mainPair}, err: asset.ErrNotSupported},
		{name: "an unsupported asset", cancel: &order.Cancel{AssetType: asset.Binary}, err: asset.ErrNotSupported},
		{name: "a pair's base alone", cancel: &order.Cancel{AssetType: asset.Spot, Pair: currency.Pair{Base: currency.BTC}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "a pair's quote alone", cancel: &order.Cancel{AssetType: asset.Spot, Pair: currency.Pair{Quote: currency.USDT}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "an unknown margin type", cancel: &order.Cancel{AssetType: asset.Margin, MarginType: margin.Unknown}, err: margin.ErrMarginTypeUnsupported},
		{name: "several margin types", cancel: &order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated | margin.Multi}, err: margin.ErrMarginTypeUnsupported},
		{name: "a time in force no order reads back with", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.AnyType, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
		{name: "a time in force beside an algo order type", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.Trigger, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{name: "a time in force no order of the type reads back with", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.Limit, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
		{name: "a market order type with immediate or cancel", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.Market, TimeInForce: order.ImmediateOrCancel}, err: order.ErrUnsupportedTimeInForce},
		{name: "a limit maker order type", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.LimitMaker}, exp: &cancelAllScope{instrumentType: "SPOT", ordinary: true, ordinaryTypes: "post_only"}},
		{name: "an order type OKX does not list", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.Stop}, err: order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scope, err := ex.newCancelAllScope(tc.cancel)
			require.ErrorIs(t, err, tc.err, "newCancelAllScope must return the expected error")
			assert.Equal(t, tc.exp, scope, "newCancelAllScope should select what the cancellation names and nothing wider")
		})
	}
}

func TestCancelAllAlgoTypes(t *testing.T) {
	t.Parallel()
	spot := []string{"conditional,oco", "trigger", "move_order_stop", "twap", "iceberg", "smart_iceberg"}
	contracts := []string{"conditional,oco", "trigger", "move_order_stop", "twap", "iceberg", "smart_iceberg", "chase"}
	for _, tc := range []struct {
		a   asset.Item
		exp []string
	}{
		{asset.Spot, spot},
		{asset.Margin, spot},
		{asset.Futures, contracts},
		{asset.PerpetualSwap, contracts},
		{asset.Empty, contracts},
		{asset.All, contracts},
		{asset.Options, nil},
		{asset.Spread, nil},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, cancelAllAlgoTypes(tc.a), "cancelAllAlgoTypes should list every algo order type OKX lists on the asset")
		})
	}
}

func TestPendingAlgoOrdersToCancel(t *testing.T) {
	t.Parallel()
	row := func(i int) map[string]string {
		return map[string]string{
			"instType": "SPOT", "instId": mainPair.String(), "algoId": fmt.Sprintf("ORD-%03d", i), "algoClOrdId": fmt.Sprintf("client-%03d", i),
			"ordType": orderTrigger, "side": "buy", "state": "live", "sz": "1", "cTime": "1700000000000",
		}
	}
	for _, tc := range append(pagerCases(row), stalledPagerCases(row)...) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, pagerHandler(t, "/trade/orders-algo-pending", "after", tc.pages))
			scope, err := ex.newCancelAllScope(&order.Cancel{AssetType: asset.Spot, Type: order.Trigger})
			require.NoError(t, err, "newCancelAllScope must not error")
			pending, err := ex.pendingAlgoOrdersToCancel(t.Context(), scope)
			if tc.incomplete {
				require.ErrorIs(t, err, errListingStalled, "pendingAlgoOrdersToCancel must report a listing which stops advancing")
				assert.ErrorContains(t, err, "listing trigger algo orders", "the error should name the listing")
			} else {
				require.NoError(t, err, "pendingAlgoOrdersToCancel must not error")
			}
			var exp []CancelAlgoOrderRequest
			for _, id := range tc.exp {
				exp = append(exp, CancelAlgoOrderRequest{InstrumentID: mainPair.String(), AlgoID: id, AlgoClientOrderID: "client-" + strings.TrimPrefix(id, "ORD-")})
			}
			assert.Equal(t, exp, pending, "pendingAlgoOrdersToCancel should select every algo order of every page once, in listing order")
		})
	}

	t.Run("leaves an order listed for another instrument type", func(t *testing.T) {
		t.Parallel()
		rows := []map[string]string{
			{"instType": instTypeSpot, "instId": mainPair.String(), "algoId": "2001", "ordType": orderTrigger, "side": "buy"},
			{"instType": instTypeMargin, "instId": mainPair.String(), "algoId": "9002", "ordType": orderTrigger, "side": "buy"},
			{"instType": "spot", "instId": mainPair.String(), "algoId": "2003", "ordType": orderTrigger, "side": "buy"},
			{"instId": mainPair.String(), "algoId": "2004", "ordType": orderTrigger, "side": "buy"},
		}
		for _, tc := range []struct {
			name   string
			cancel *order.Cancel
			exp    []CancelAlgoOrderRequest
			err    error
		}{
			{
				name: "every trigger order of a spot pair", cancel: &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger}, err: errListedOtherInstrumentType,
				exp: []CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "2001"}, {InstrumentID: mainPair.String(), AlgoID: "2003"}, {InstrumentID: mainPair.String(), AlgoID: "2004"}},
			},
			{name: "the other instrument type's order by its ID", cancel: &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "9002"}, err: errListedOtherInstrumentType},
			{
				name: "every trigger order of every asset", cancel: &order.Cancel{AssetType: asset.All, Pair: mainPair, Type: order.Trigger},
				exp: []CancelAlgoOrderRequest{{InstrumentID: mainPair.String(), AlgoID: "2001"}, {InstrumentID: mainPair.String(), AlgoID: "9002"}, {InstrumentID: mainPair.String(), AlgoID: "2003"}, {InstrumentID: mainPair.String(), AlgoID: "2004"}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/orders-algo-pending" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					writeOKXData(t, w, rows)
				}))
				scope, err := ex.newCancelAllScope(tc.cancel)
				require.NoError(t, err, "newCancelAllScope must not error")
				pending, err := ex.pendingAlgoOrdersToCancel(t.Context(), scope)
				require.ErrorIs(t, err, tc.err, "pendingAlgoOrdersToCancel must report an order of another instrument type than it listed")
				if tc.err != nil {
					assert.ErrorContains(t, err, `order "9002" is MARGIN`, "the error should name the order left alone")
				}
				assert.Equal(t, tc.exp, pending, "pendingAlgoOrdersToCancel should select the orders of the requested instrument type alone")
			})
		}
	})

	t.Run("a failed listing does not stop the others", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch ordType := r.URL.Query().Get("ordType"); ordType {
			case orderTrigger:
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"code":"51000","msg":"Parameter ordType error","data":[]}`))
				assert.NoError(t, err, "writing the reply should not error")
			default:
				kind, _, _ := strings.Cut(ordType, ",")
				writeOKXData(t, w, []map[string]string{{"instType": "SPOT", "instId": mainPair.String(), "algoId": ordType + "-1", "ordType": kind, "side": "sell"}})
			}
		}))
		scope, err := ex.newCancelAllScope(&order.Cancel{AssetType: asset.Spot, Pair: mainPair})
		require.NoError(t, err, "newCancelAllScope must not error")
		pending, err := ex.pendingAlgoOrdersToCancel(t.Context(), scope)
		require.ErrorIs(t, err, ErrorCodes["51000"], "pendingAlgoOrdersToCancel must report the failed listing")
		assert.ErrorContains(t, err, "listing trigger algo orders", "the error should name the failed listing")
		exp := []CancelAlgoOrderRequest{
			{InstrumentID: mainPair.String(), AlgoID: "conditional,oco-1"},
			{InstrumentID: mainPair.String(), AlgoID: "move_order_stop-1"},
			{InstrumentID: mainPair.String(), AlgoID: "twap-1"},
			{InstrumentID: mainPair.String(), AlgoID: "iceberg-1"},
			{InstrumentID: mainPair.String(), AlgoID: "smart_iceberg-1"},
		}
		assert.Equal(t, exp, pending, "the other listings' orders should still be selected")
	})

	t.Run("a dead context stops the listing", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		var requests atomic.Int32
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			cancel()
			writeOKXData(t, w, []map[string]string{})
		}))
		scope, err := ex.newCancelAllScope(&order.Cancel{AssetType: asset.Spot})
		require.NoError(t, err, "newCancelAllScope must not error")
		_, err = ex.pendingAlgoOrdersToCancel(ctx, scope)
		require.ErrorIs(t, err, context.Canceled, "pendingAlgoOrdersToCancel must report the dead context")
		assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()), "pendingAlgoOrdersToCancel should report the context error once")
		assert.Equal(t, int32(1), requests.Load(), "no listing should be requested once the context is dead")
	})
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Binary})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		result, err := e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Spread})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		result, err = e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Futures})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		result, err = e.CancelAllOrders(contextGenerate(), &order.Cancel{AssetType: asset.Spot})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("matches only the requested orders", func(t *testing.T) {
		t.Parallel()
		// Subtests share the mock server's pending order queue, so they run
		// sequentially.
		var mu sync.Mutex
		var pending []map[string]string
		// pages, when set, serves the pending order list keyed by the after
		// pagination parameter, with "" naming the first page.
		var pages map[string][]map[string]string
		var cancelled []CancelOrderRequest
		var pendingQuery string
		var afterCursors []string

		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-pending":
				after := r.URL.Query().Get("after")
				mu.Lock()
				defer mu.Unlock()
				if slices.Contains(afterCursors, after) {
					// Nothing here fails a request, so a repeat means the cursor never advanced.
					t.Errorf("after %q requested again", after)
					http.Error(w, "repeated after", http.StatusBadRequest)
					return
				}
				afterCursors = append(afterCursors, after)
				pendingQuery = r.URL.RawQuery
				if pages != nil {
					writeOKXData(t, w, pages[after])
					return
				}
				writeOKXData(t, w, pending)
			case "/trade/orders-algo-pending":
				writeOKXData(t, w, []map[string]string{})
			case "/trade/cancel-batch-orders":
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("reading cancel request body should not error: %v", err)
					return
				}
				var reqs []CancelOrderRequest
				if err := json.Unmarshal(body, &reqs); err != nil {
					t.Errorf("decoding cancel request body should not error: %v", err)
					return
				}
				respData := make([]map[string]string, 0, len(reqs))
				for x := range reqs {
					respData = append(respData, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
				}
				mu.Lock()
				cancelled = append(cancelled, reqs...)
				mu.Unlock()
				writeOKXData(t, w, respData)
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))

		setPending := func(ords ...map[string]string) {
			mu.Lock()
			defer mu.Unlock()
			pending = ords
			pages = nil
		}
		setPages := func(p map[string][]map[string]string) {
			mu.Lock()
			defer mu.Unlock()
			pages = p
		}
		resetCancelled := func() {
			mu.Lock()
			defer mu.Unlock()
			cancelled = nil
			pendingQuery = ""
			afterCursors = nil
		}
		cancelledIDs := func() []string {
			mu.Lock()
			defer mu.Unlock()
			ids := make([]string, 0, len(cancelled))
			for x := range cancelled {
				ids = append(ids, cancelled[x].OrderID)
			}
			return ids
		}

		t.Run("by order ID", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "KEEP-ME"},
				map[string]string{"instId": "BTC-USDT", "ordId": "CANCEL-ME"},
			)
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType: asset.Spot,
				Pair:      mainPair,
				OrderID:   "CANCEL-ME",
			})
			require.NoError(t, err, "CancelAllOrders must not error when cancelling by order ID")
			assert.Equal(t, []string{"CANCEL-ME"}, cancelledIDs(), "cancelling by order ID should cancel only the requested order")
			assert.Contains(t, resp.Status, "CANCEL-ME", "the cancelled order should be reported")
		})

		t.Run("by client order ID", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "OTHER", "clOrdId": "other"},
				map[string]string{"instId": "BTC-USDT", "ordId": "TARGET", "clOrdId": "target"},
			)
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType:     asset.Spot,
				Pair:          mainPair,
				ClientOrderID: "target",
			})
			require.NoError(t, err, "CancelAllOrders must not error when cancelling by client order ID")
			assert.Equal(t, []string{"TARGET"}, cancelledIDs(), "cancelling by client order ID should cancel only the requested order")
		})

		t.Run("by both order ID and client order ID", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "MATCHES-CL-only", "clOrdId": "both"},
				map[string]string{"instId": "BTC-USDT", "ordId": "both", "clOrdId": "both"},
			)
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType:     asset.Spot,
				Pair:          mainPair,
				OrderID:       "both",
				ClientOrderID: "both",
			})
			require.NoError(t, err, "CancelAllOrders must not error when cancelling by both order ID and client order ID")
			assert.Equal(t, []string{"both"}, cancelledIDs(), "supplying both IDs should only cancel the order matching both")
		})

		t.Run("by both order ID and client order ID, matching the order ID only", func(t *testing.T) {
			resetCancelled()
			setPending(map[string]string{"instId": "BTC-USDT", "ordId": "both", "clOrdId": "MATCHES-ORD-only"})
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType:     asset.Spot,
				Pair:          mainPair,
				OrderID:       "both",
				ClientOrderID: "both",
			})
			require.NoError(t, err, "CancelAllOrders must not error when no order matches both IDs")
			assert.Empty(t, cancelledIDs(), "supplying both IDs should not cancel an order matching the order ID alone")
		})

		t.Run("order type filter parameter", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "LMT-1", "ordType": "limit"},
			)
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType: asset.Spot,
				Pair:      mainPair,
				Type:      order.Limit,
			})
			require.NoError(t, err, "CancelAllOrders must not error when filtering by order type")
			mu.Lock()
			query := pendingQuery
			mu.Unlock()
			require.NotEmpty(t, query, "the pending order list must have been requested")
			assert.Contains(t, query, "ordType=limit", "the order type filter should be sent as ordType")
			assert.NotContains(t, query, "orderType=", "OKX ignores an orderType parameter, so it should not be sent")
		})

		t.Run("formats the pair through the exchange's pair format", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "ANY"},
			)
			slashPair := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "/")
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType: asset.Spot,
				Pair:      slashPair,
			})
			require.NoError(t, err, "CancelAllOrders must not error for a differently delimited pair")
			mu.Lock()
			query := pendingQuery
			mu.Unlock()
			assert.Contains(t, query, "instId=BTC-USDT", "the instrument filter should use the exchange's pair format")
		})

		t.Run("by side", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "BUY-1", "side": "buy"},
				map[string]string{"instId": "BTC-USDT", "ordId": "SELL-1", "side": "sell"},
			)
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType: asset.Spot,
				Pair:      mainPair,
				Side:      order.Buy,
			})
			require.NoError(t, err, "CancelAllOrders must not error when cancelling by side")
			assert.Equal(t, []string{"BUY-1"}, cancelledIDs(), "cancelling buys should not cancel sell orders")
		})

		t.Run("all", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "BUY-1", "side": "buy"},
				map[string]string{"instId": "BTC-USDT", "ordId": "SELL-1", "side": "sell"},
			)
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType: asset.Spot,
				Pair:      mainPair,
			})
			require.NoError(t, err, "CancelAllOrders must not error when cancelling everything")
			assert.Equal(t, []string{"BUY-1", "SELL-1"}, cancelledIDs(), "an unfiltered cancel should cancel every open order")
		})

		t.Run("paginates the pending order list", func(t *testing.T) {
			resetCancelled()
			firstPage := make([]map[string]string, 0, orderListPageSize)
			for x := range orderListPageSize {
				firstPage = append(firstPage, map[string]string{"instId": "BTC-USDT", "ordId": fmt.Sprintf("PAGE1-%03d", x)})
			}
			lastOfFirstPage := firstPage[orderListPageSize-1]["ordId"]
			setPages(map[string][]map[string]string{
				"": firstPage,
				lastOfFirstPage: {
					{"instId": "BTC-USDT", "ordId": "PAGE2-0"},
					{"instId": "BTC-USDT", "ordId": "PAGE2-1"},
				},
			})
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType: asset.Spot,
				Pair:      mainPair,
			})
			require.NoError(t, err, "CancelAllOrders must not error when the pending order list spans pages")
			got := cancelledIDs()
			assert.Len(t, got, orderListPageSize+2, "a paginated cancel should cancel every open order across pages")
			assert.Contains(t, got, "PAGE2-1", "orders past the first page should be cancelled")
		})

		// The mock returns every pending order whatever the query, so these
		// pin the filters applied to the listing.
		for _, tc := range []struct {
			name    string
			pending map[string]string
			cancel  *order.Cancel
			exp     []string
		}{
			{"by order ID of another side", map[string]string{"instId": "BTC-USDT", "ordId": "401", "side": "sell", "ordType": "limit"}, &order.Cancel{Side: order.Buy, OrderID: "401"}, []string{}},
			{"by order ID and its own side", map[string]string{"instId": "BTC-USDT", "ordId": "401", "side": "sell", "ordType": "limit"}, &order.Cancel{Side: order.Sell, OrderID: "401"}, []string{"401"}},
			{"by client order ID of another side", map[string]string{"instId": "BTC-USDT", "ordId": "402", "clOrdId": "c402", "side": "sell", "ordType": "limit"}, &order.Cancel{Side: order.Buy, ClientOrderID: "c402"}, []string{}},
			{"by order ID and an order type it does not have", map[string]string{"instId": "BTC-USDT", "ordId": "401", "side": "sell", "ordType": "limit"}, &order.Cancel{Type: order.Market, OrderID: "401"}, []string{}},
			{"by order ID and its own order type", map[string]string{"instId": "BTC-USDT", "ordId": "401", "side": "sell", "ordType": "post_only"}, &order.Cancel{Type: order.Limit, OrderID: "401"}, []string{"401"}},
			{"by order ID on another instrument", map[string]string{"instId": "ETH-USDT", "ordId": "401", "side": "sell", "ordType": "limit"}, &order.Cancel{OrderID: "401"}, []string{}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resetCancelled()
				setPending(tc.pending)
				tc.cancel.AssetType = asset.Spot
				tc.cancel.Pair = mainPair
				_, err := ex.CancelAllOrders(t.Context(), tc.cancel)
				require.NoError(t, err, "CancelAllOrders must not error")
				assert.Equal(t, tc.exp, cancelledIDs(), "only the orders every supplied filter selects should be cancelled")
			})
		}

		for _, tc := range []struct {
			side order.Side
			exp  []string
		}{
			{order.Long, []string{"BUY-1"}},
			{order.Bid, []string{"BUY-1"}},
			{order.Short, []string{"SELL-1"}},
			{order.Ask, []string{"SELL-1"}},
			{order.AnySide, []string{"BUY-1", "SELL-1"}},
		} {
			t.Run("by side "+tc.side.String(), func(t *testing.T) {
				resetCancelled()
				setPending(
					map[string]string{"instId": "BTC-USDT", "ordId": "BUY-1", "side": "buy"},
					map[string]string{"instId": "BTC-USDT", "ordId": "SELL-1", "side": "sell"},
				)
				_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Side: tc.side})
				require.NoError(t, err, "CancelAllOrders must not error")
				assert.Equal(t, tc.exp, cancelledIDs(), "cancelling a side should cancel the orders on that side only")
			})
		}

		t.Run("by an order ID the listing repeats", func(t *testing.T) {
			resetCancelled()
			setPending(
				map[string]string{"instId": "BTC-USDT", "ordId": "401", "side": "sell"},
				map[string]string{"instId": "BTC-USDT", "ordId": "401", "side": "sell"},
			)
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "401"})
			require.NoError(t, err, "CancelAllOrders must not error")
			assert.Equal(t, []string{"401"}, cancelledIDs(), "an order ID should cancel its order once")
		})

		t.Run("refuses a side it cannot filter by", func(t *testing.T) {
			resetCancelled()
			setPending(map[string]string{"instId": "BTC-USDT", "ordId": "BUY-1", "side": "buy"})
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Side: order.ClosePosition})
			require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelAllOrders must refuse a side it cannot filter by")
			mu.Lock()
			query := pendingQuery
			mu.Unlock()
			assert.Empty(t, query, "a refused cancel should not list the pending orders")
			assert.Empty(t, cancelledIDs(), "a refused cancel should cancel nothing")
		})
	})

	t.Run("continues after a failed batch", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var cancelRequests int

		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-pending":
				// 22 orders exceed one batch of 20, forming a failing first batch
				// and a succeeding second batch.
				ords := make([]map[string]string, 0, 22)
				for x := range 22 {
					ords = append(ords, map[string]string{"instId": "BTC-USDT", "ordId": fmt.Sprintf("ORD-%02d", x)})
				}
				writeOKXData(t, w, ords)
			case "/trade/orders-algo-pending":
				writeOKXData(t, w, []map[string]string{})
			case "/trade/cancel-batch-orders":
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("reading cancel request body should not error: %v", err)
					return
				}
				var reqs []CancelOrderRequest
				if err := json.Unmarshal(body, &reqs); err != nil {
					t.Errorf("decoding cancel request body should not error: %v", err)
					return
				}
				mu.Lock()
				cancelRequests++
				first := cancelRequests == 1
				mu.Unlock()
				if first {
					http.Error(w, "batch one failed", http.StatusInternalServerError)
					return
				}
				respData := make([]map[string]string, 0, len(reqs))
				for x := range reqs {
					respData = append(respData, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
				}
				writeOKXData(t, w, respData)
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))

		resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
		require.Error(t, err, "CancelAllOrders must report the failed batch")
		mu.Lock()
		requests := cancelRequests
		mu.Unlock()
		assert.Equal(t, 2, requests, "a failed batch should not stop later batches from being sent")
		assert.Contains(t, resp.Status, "ORD-21", "orders from the later successful batch should still be cancelled")
		assert.NotContains(t, resp.Status, "ORD-00", "orders from the failed batch should not be reported as cancelled")
	})

	t.Run("rest partial success", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-pending":
				writeOKXData(t, w, []map[string]string{
					{"instId": "BTC-USDT", "ordId": "OK-1"},
					{"instId": "BTC-USDT", "ordId": "FAIL-1"},
				})
			case "/trade/orders-algo-pending":
				writeOKXData(t, w, []map[string]string{})
			case "/trade/cancel-batch-orders":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"OK-1","sCode":"0","sMsg":""},{"ordId":"FAIL-1","sCode":"51400","sMsg":"Order does not exist"}]}`))
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
		require.ErrorIs(t, err, errPartialSuccess, "a partially successful REST batch must report its error")
		assert.Equal(t, map[string]string{
			"OK-1":   order.PendingCancel.String(),
			"FAIL-1": "Order does not exist",
		}, resp.Status, "a partially successful REST batch should report every per-order result")
	})

	t.Run("scopes the spread mass cancel", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var massCancelBodies []string

		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/sprd/mass-cancel" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading mass-cancel body should not error: %v", err)
				return
			}
			mu.Lock()
			massCancelBodies = append(massCancelBodies, string(body))
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"result":true}]}`))
		}))

		// Subtests read the mock's recorded bodies, so they run sequentially.
		for _, tc := range []struct {
			name    string
			request order.Cancel
		}{
			{"order ID alone", order.Cancel{AssetType: asset.Spread, OrderID: "ONE"}},
			{"client order ID alone", order.Cancel{AssetType: asset.Spread, ClientOrderID: "mine"}},
			{"pair and order ID", order.Cancel{AssetType: asset.Spread, Pair: spreadPair, OrderID: "ONE"}},
			{"pair and side", order.Cancel{AssetType: asset.Spread, Pair: spreadPair, Side: order.Buy}},
			{"pair and order type", order.Cancel{AssetType: asset.Spread, Pair: spreadPair, Type: order.Limit}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				mu.Lock()
				before := len(massCancelBodies)
				mu.Unlock()
				_, err := ex.CancelAllOrders(t.Context(), &tc.request)
				assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a mass cancel scoped by a filter it cannot apply should be rejected")
				mu.Lock()
				defer mu.Unlock()
				assert.Empty(t, massCancelBodies[before:], "a rejected cancel should send no mass cancel request")
			})
		}

		t.Run("any side and any type cancel the spread's orders", func(t *testing.T) {
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spread, Pair: spreadPair, Side: order.AnySide, Type: order.AnyType})
			require.NoError(t, err, "CancelAllOrders must take any side and any type as no filter")
			mu.Lock()
			last := massCancelBodies[len(massCancelBodies)-1]
			mu.Unlock()
			assert.JSONEq(t, `{"sprdId":"BTC-USDT_BTC-USDT-SWAP"}`, last, "the mass cancel should be scoped by the pair alone")
			assert.Equal(t, map[string]string{"BTC-USDT_BTC-USDT-SWAP": "true"}, resp.Status, "the result should be reported under the spread it cancelled")
		})

		t.Run("pair scopes the mass cancel", func(t *testing.T) {
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{
				AssetType: asset.Spread,
				Pair:      spreadPair,
			})
			require.NoError(t, err, "CancelAllOrders must not error for a pair-scoped spread cancel")
			mu.Lock()
			last := massCancelBodies[len(massCancelBodies)-1]
			mu.Unlock()
			assert.Contains(t, last, `"sprdId":"BTC-USDT_BTC-USDT-SWAP"`, "a populated pair should scope the mass cancel to that spread")
			assert.Equal(t, map[string]string{"BTC-USDT_BTC-USDT-SWAP": "true"}, resp.Status, "the result should be reported under the spread it cancelled")
		})

		t.Run("no pair cancels every spread", func(t *testing.T) {
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spread})
			require.NoError(t, err, "CancelAllOrders must not error for an unscoped spread cancel")
			mu.Lock()
			last := massCancelBodies[len(massCancelBodies)-1]
			mu.Unlock()
			assert.Equal(t, "{}", last, "an unscoped cancel should omit sprdId so OKX cancels every spread order")
		})
	})

	t.Run("acts for one account when the exchange's credentials change mid-cancel", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var requests []string
		var ex atomic.Pointer[Exchange]
		mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requests = append(requests, r.URL.Path+" "+r.Header.Get("OK-ACCESS-KEY"))
			mu.Unlock()
			switch r.URL.Path {
			case "/trade/orders-pending":
				ex.Load().SetCredentials(secondaryAccount())
				writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "1001", "side": "buy", "ordType": orderLimit}})
			case "/trade/orders-algo-pending":
				writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "algoId": "2001", "side": "buy", "ordType": orderTrigger}})
			case "/trade/cancel-batch-orders":
				writeOKXData(t, w, []map[string]string{{"ordId": "1001", "sCode": "0"}})
			case "/trade/cancel-algos":
				writeOKXData(t, w, []map[string]string{{"algoId": "2001", "sCode": "0"}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		ex.Store(mock)
		mock.SetCredentials(primaryAccount())
		resp, err := mock.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
		require.NoError(t, err, "CancelAllOrders must not error")
		assert.Equal(t, map[string]string{"1001": order.PendingCancel.String(), "2001": order.PendingCancel.String()}, resp.Status, "both orders should be cancelled")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{
			"/trade/orders-pending primary-key", "/trade/cancel-batch-orders primary-key",
			"/trade/orders-algo-pending primary-key", "/trade/orders-algo-pending primary-key", "/trade/orders-algo-pending primary-key", "/trade/orders-algo-pending primary-key",
			"/trade/orders-algo-pending primary-key", "/trade/orders-algo-pending primary-key",
			"/trade/cancel-algos primary-key",
		}, requests, "every listing and cancel should act for the account the cancel-all started with")
	})

	t.Run("refuses a filter it cannot apply", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			t.Errorf("unexpected REST request %s", r.URL.Path)
			http.NotFound(w, r)
		}, func(op string, _ json.RawMessage) (string, any) {
			requests.Add(1)
			t.Errorf("unexpected websocket operation %s", op)
			return "60013", nil
		})
		for _, tc := range []struct {
			name   string
			cancel order.Cancel
			err    error
			wsErr  error
		}{
			{name: "a pair's base alone", cancel: order.Cancel{AssetType: asset.Spot, Pair: currency.Pair{Base: currency.BTC}}, err: currency.ErrCurrencyCodeEmpty},
			{name: "a pair's quote alone", cancel: order.Cancel{AssetType: asset.Spot, Pair: currency.Pair{Quote: currency.USDT}}, err: currency.ErrCurrencyCodeEmpty},
			{name: "an invalid asset", cancel: order.Cancel{AssetType: asset.Item(255), Pair: mainPair}, err: asset.ErrNotSupported},
			{name: "an unknown margin type", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, MarginType: margin.Unknown}, err: margin.ErrMarginTypeUnsupported},
			{name: "a time in force no order reads back with", cancel: order.Cancel{AssetType: asset.Spot, Type: order.AnyType, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
			{name: "a time in force beside an algo order type", cancel: order.Cancel{AssetType: asset.Spot, Type: order.Trigger, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce, wsErr: common.ErrFunctionNotSupported},
			{name: "a limit order type with good till day", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
			{name: "a limit order type with good till time", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, TimeInForce: order.GoodTillTime}, err: order.ErrUnsupportedTimeInForce},
			{name: "a limit order type with stop or reduce", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, TimeInForce: order.StopOrReduce}, err: order.ErrUnsupportedTimeInForce},
			{name: "a limit order type with good till cancel and stop or reduce", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, TimeInForce: order.GoodTillCancel | order.StopOrReduce}, err: order.ErrUnsupportedTimeInForce},
			{name: "a market order type with immediate or cancel", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Market, TimeInForce: order.ImmediateOrCancel}, err: order.ErrUnsupportedTimeInForce},
			{name: "a market order type with fill or kill", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Market, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
			{name: "a limit maker order type with good till day", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.LimitMaker, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
			{name: "a spread pair's base alone", cancel: order.Cancel{AssetType: asset.Spread, Pair: currency.Pair{Base: currency.BTC}}, err: currency.ErrCurrencyCodeEmpty},
			{name: "a spread time in force", cancel: order.Cancel{AssetType: asset.Spread, Pair: spreadPair, TimeInForce: order.PostOnly}, err: common.ErrFunctionNotSupported},
			{name: "a spread margin type", cancel: order.Cancel{AssetType: asset.Spread, Pair: spreadPair, MarginType: margin.Isolated}, err: common.ErrFunctionNotSupported},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := ex.CancelAllOrders(t.Context(), &tc.cancel)
				assert.ErrorIs(t, err, tc.err, "CancelAllOrders should refuse a filter it cannot apply")
				_, err = ex.WebsocketCancelAllOrders(t.Context(), &tc.cancel)
				assert.ErrorIs(t, err, cmp.Or(tc.wsErr, tc.err), "WebsocketCancelAllOrders should refuse a filter it cannot apply")
			})
		}
		t.Cleanup(func() { assert.Zero(t, requests.Load(), "a refused cancel-all should send nothing") })
	})

	t.Run("applies the time in force and margin type filters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			cancel   order.Cancel
			exp      []string
			listings []string
		}{
			{
				name: "post only orders of any type", cancel: order.Cancel{AssetType: asset.Spot, Type: order.AnyType, TimeInForce: order.PostOnly}, exp: []string{"1002", "1005"},
				listings: []string{"/trade/orders-pending instType=SPOT&ordType=post_only%2Cmmp_and_post_only"},
			},
			{name: "isolated margin orders", cancel: order.Cancel{AssetType: asset.Margin, Pair: mainPair, Type: order.Limit, MarginType: margin.Isolated}, exp: []string{"1003"}, listings: []string{"/trade/orders-pending instId=BTC-USDT&instType=MARGIN&ordType=limit%2Cpost_only%2Cfok%2Cioc%2Cop_fok%2Crpi"}},
			{name: "cross margin orders", cancel: order.Cancel{AssetType: asset.Margin, Pair: mainPair, Type: order.Limit, MarginType: margin.Multi}, exp: []string{"1004"}, listings: []string{"/trade/orders-pending instId=BTC-USDT&instType=MARGIN&ordType=limit%2Cpost_only%2Cfok%2Cioc%2Cop_fok%2Crpi"}},
			{name: "cash orders of a margin pair", cancel: order.Cancel{AssetType: asset.Margin, Pair: mainPair, Type: order.Limit, MarginType: margin.NoMargin}, listings: []string{"/trade/orders-pending instId=BTC-USDT&instType=MARGIN&ordType=limit%2Cpost_only%2Cfok%2Cioc%2Cop_fok%2Crpi"}},
		} {
			for _, websocket := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s websocket %t", tc.name, websocket), func(t *testing.T) {
					t.Parallel()
					book := &pendingBook{ordinary: []map[string]string{
						{"instType": "SPOT", "instId": "BTC-USDT", "ordId": "1001", "ordType": orderLimit, "tdMode": TradeModeCash, "side": "buy"},
						{"instType": "SPOT", "instId": "ETH-USDT", "ordId": "1002", "ordType": orderPostOnly, "tdMode": TradeModeCash, "side": "sell"},
						{"instType": "SPOT", "instId": "BTC-USDT", "ordId": "1005", "ordType": orderMarketMakerProtectionAndPostOnly, "tdMode": TradeModeCash, "side": "buy"},
						{"instType": "MARGIN", "instId": "BTC-USDT", "ordId": "1003", "ordType": orderLimit, "tdMode": TradeModeIsolated, "side": "buy"},
						{"instType": "MARGIN", "instId": "BTC-USDT", "ordId": "1004", "ordType": orderLimit, "tdMode": TradeModeCross, "side": "sell"},
					}}
					ex := newMockWebsocketExchange(t, book.rest(t), book.websocket(t))
					require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, true), "margin must enable")
					ex.instrumentIDCodeMap["ETH-USDT"] = 444
					cancelAll := ex.CancelAllOrders
					if websocket {
						cancelAll = ex.WebsocketCancelAllOrders
					}
					resp, err := cancelAll(t.Context(), &tc.cancel)
					require.NoError(t, err, "the cancel-all must not error")
					listings, cancelled, _ := book.sent()
					assert.Equal(t, tc.exp, cancelled, "only the orders every filter selects should be cancelled")
					assert.Len(t, resp.Status, len(tc.exp), "each cancelled order should be reported")
					assert.Equal(t, tc.listings, listings, "the listing should carry every filter OKX applies")
				})
			}
		}
	})

	t.Run("cancels the selected algo orders", func(t *testing.T) {
		t.Parallel()
		spotAlgoListings := func(query string) []string {
			return []string{
				"/trade/orders-algo-pending " + query + "&ordType=conditional%2Coco",
				"/trade/orders-algo-pending " + query + "&ordType=trigger",
				"/trade/orders-algo-pending " + query + "&ordType=move_order_stop",
				"/trade/orders-algo-pending " + query + "&ordType=twap",
				"/trade/orders-algo-pending " + query + "&ordType=iceberg",
				"/trade/orders-algo-pending " + query + "&ordType=smart_iceberg",
			}
		}
		for _, tc := range []struct {
			name        string
			cancel      order.Cancel
			cancelled   []string
			algoBatches [][]string
			listings    []string
		}{
			{
				name: "every order of a pair", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair},
				cancelled: []string{"1001"}, algoBatches: [][]string{{"2002", "2003", "2001", "2004", "2005", "2008", "2009"}},
				listings: append([]string{"/trade/orders-pending instId=BTC-USDT&instType=SPOT"}, spotAlgoListings("instId=BTC-USDT&instType=SPOT")...),
			},
			{
				name: "an algo order type alone", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger},
				algoBatches: [][]string{{"2001"}}, listings: []string{"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=trigger"},
			},
			{
				name: "an OCO order type alone", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.OCO},
				algoBatches: [][]string{{"2003"}}, listings: []string{"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=oco"},
			},
			{
				name: "a side", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell},
				algoBatches: [][]string{{"2002", "2004", "2009"}},
				listings:    append([]string{"/trade/orders-pending instId=BTC-USDT&instType=SPOT"}, spotAlgoListings("instId=BTC-USDT&instType=SPOT")...),
			},
			{
				name: "a margin type", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, MarginType: margin.NoMargin},
				cancelled: []string{"1001"}, algoBatches: [][]string{{"2003", "2001", "2005", "2008"}},
				listings: append([]string{"/trade/orders-pending instId=BTC-USDT&instType=SPOT"}, spotAlgoListings("instId=BTC-USDT&instType=SPOT")...),
			},
			{
				name: "an algo order's client order ID", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, ClientOrderID: "algo-client"},
				algoBatches: [][]string{{"2001"}},
				listings:    append([]string{"/trade/orders-pending instId=BTC-USDT&instType=SPOT"}, spotAlgoListings("instId=BTC-USDT&instType=SPOT")[:2]...),
			},
			{
				name: "an ordinary order's ID", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "1001"},
				cancelled: []string{"1001"}, listings: []string{"/trade/orders-pending instId=BTC-USDT&instType=SPOT"},
			},
			{
				name: "every order of a perpetual swap", cancel: order.Cancel{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair},
				algoBatches: [][]string{{"2007"}},
				listings: append(append([]string{"/trade/orders-pending instId=BTC-USDT-SWAP&instType=SWAP"}, spotAlgoListings("instId=BTC-USDT-SWAP&instType=SWAP")...),
					"/trade/orders-algo-pending instId=BTC-USDT-SWAP&instType=SWAP&ordType=chase"),
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				book := &pendingBook{
					ordinary: []map[string]string{{"instType": "SPOT", "instId": "BTC-USDT", "ordId": "1001", "ordType": orderLimit, "tdMode": TradeModeCash, "side": "buy"}},
					algo: []map[string]string{
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2001", "algoClOrdId": "algo-client", "ordType": orderTrigger, "tdMode": TradeModeCash, "side": "buy"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2002", "ordType": orderConditional, "tdMode": TradeModeCross, "side": "sell"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2003", "ordType": orderOCO, "tdMode": TradeModeCash, "side": "buy"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2004", "ordType": orderMoveOrderStop, "tdMode": TradeModeIsolated, "side": "sell"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2005", "ordType": orderTWAP, "tdMode": TradeModeCash, "side": "buy"},
						{"instType": "SPOT", "instId": "ETH-USDT", "algoId": "2006", "ordType": orderTrigger, "tdMode": TradeModeCash, "side": "buy"},
						{"instType": "SWAP", "instId": "BTC-USDT-SWAP", "algoId": "2007", "ordType": orderChase, "tdMode": TradeModeCross, "side": "buy"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2008", "ordType": "iceberg", "tdMode": TradeModeCash, "side": "buy"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2009", "ordType": orderSmartIceberg, "tdMode": TradeModeCross, "side": "sell"},
					},
				}
				ex := newMockExchange(t, book.rest(t))
				resp, err := ex.CancelAllOrders(t.Context(), &tc.cancel)
				require.NoError(t, err, "CancelAllOrders must not error")
				listings, cancelled, algoBatches := book.sent()
				assert.Equal(t, tc.cancelled, cancelled, "CancelAllOrders should cancel the selected ordinary orders")
				assert.Equal(t, tc.algoBatches, algoBatches, "CancelAllOrders should cancel the selected algo orders")
				assert.Equal(t, tc.listings, listings, "CancelAllOrders should list the orders the cancellation selects")
				exp := map[string]string{}
				for _, id := range slices.Concat(append(tc.algoBatches, tc.cancelled)...) {
					exp[id] = order.PendingCancel.String()
				}
				assert.Equal(t, exp, resp.Status, "CancelAllOrders should report every cancelled order")
			})
		}

		t.Run("ten algo orders a request", func(t *testing.T) {
			t.Parallel()
			book := new(pendingBook)
			for i := range 12 {
				book.algo = append(book.algo, map[string]string{"instType": "SPOT", "instId": "BTC-USDT", "algoId": strconv.Itoa(2001 + i), "ordType": orderTrigger, "side": "buy"})
			}
			ex := newMockExchange(t, book.rest(t))
			resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger})
			require.NoError(t, err, "CancelAllOrders must not error")
			_, _, algoBatches := book.sent()
			assert.Equal(t, [][]string{
				{"2001", "2002", "2003", "2004", "2005", "2006", "2007", "2008", "2009", "2010"},
				{"2011", "2012"},
			}, algoBatches, "the algo orders should be cancelled ten at a time, in listing order")
			assert.Len(t, resp.Status, 12, "every algo order should be reported")
		})
	})

	t.Run("leaves orders listed for another instrument type", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, listing, cancel, idKey string
			orderType                    order.Type
			kind                         string
		}{
			{name: "ordinary orders", listing: "/trade/orders-pending", cancel: "/trade/cancel-batch-orders", idKey: "ordId", orderType: order.Limit, kind: orderLimit},
			{name: "algo orders", listing: "/trade/orders-algo-pending", cancel: "/trade/cancel-algos", idKey: "algoId", orderType: order.Trigger, kind: orderTrigger},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var cancelled []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case tc.listing:
						writeOKXData(t, w, []map[string]string{
							{"instType": instTypeSpot, "instId": mainPair.String(), tc.idKey: "1001", "ordType": tc.kind, "side": "buy"},
							{"instType": instTypeMargin, "instId": mainPair.String(), tc.idKey: "9002", "ordType": tc.kind, "side": "buy"},
						})
					case tc.cancel:
						var reqs []map[string]string
						if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&reqs), "the cancel request body should decode") {
							return
						}
						rows := make([]map[string]string, 0, len(reqs))
						mu.Lock()
						for _, req := range reqs {
							cancelled = append(cancelled, req[tc.idKey])
							rows = append(rows, map[string]string{tc.idKey: req[tc.idKey], "sCode": "0"})
						}
						mu.Unlock()
						writeOKXData(t, w, rows)
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: tc.orderType})
				require.ErrorIs(t, err, errListedOtherInstrumentType, "CancelAllOrders must report an order of another instrument type than it listed")
				assert.ErrorContains(t, err, `order "9002" is MARGIN`, "the error should name the order left alone")
				assert.Equal(t, map[string]string{"1001": order.PendingCancel.String()}, resp.Status, "CancelAllOrders should cancel the requested instrument type's orders alone")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, []string{"1001"}, cancelled, "the other instrument type's order should not be cancelled")
			})
		}
	})

	t.Run("reports an algo order listing which stops advancing", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name              string
			orders, cancelled int
			page              func(after string, live map[string]bool) []string
			cursors           []string
		}{
			{
				name: "a full page repeated", orders: orderListPageSize + 1, cancelled: orderListPageSize, cursors: []string{"", "2"},
				page: func(_ string, live map[string]bool) []string { return liveOrderIDs(live, orderListPageSize+1) },
			},
			{
				name: "pages leading back to an earlier cursor", orders: 2 * orderListPageSize, cancelled: 2 * orderListPageSize, cursors: []string{"", "101", "1"},
				page: func(after string, live map[string]bool) []string {
					if after == "101" {
						return liveOrderIDs(live, orderListPageSize)
					}
					return liveOrderIDs(live, 2*orderListPageSize)
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				live := make(map[string]bool, tc.orders)
				for i := 1; i <= tc.orders; i++ {
					live[strconv.Itoa(i)] = true
				}
				var cursors []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					switch r.URL.Path {
					case "/trade/orders-algo-pending":
						after := r.URL.Query().Get("after")
						if slices.Contains(cursors, after) {
							t.Errorf("after %q requested again", after)
							http.Error(w, "repeated after", http.StatusBadRequest)
							return
						}
						cursors = append(cursors, after)
						ids := tc.page(after, live)
						rows := make([]map[string]string, 0, len(ids))
						for _, id := range ids {
							rows = append(rows, map[string]string{"instType": instTypeSpot, "instId": mainPair.String(), "algoId": id, "ordType": orderTrigger, "side": "buy", "state": "live"})
						}
						writeOKXData(t, w, rows)
					case "/trade/cancel-algos":
						var reqs []CancelAlgoOrderRequest
						if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&reqs), "the algo cancel request body should decode") {
							return
						}
						rows := make([]map[string]string, 0, len(reqs))
						for x := range reqs {
							delete(live, reqs[x].AlgoID)
							rows = append(rows, map[string]string{"algoId": reqs[x].AlgoID, "sCode": "0"})
						}
						writeOKXData(t, w, rows)
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger})
				require.ErrorIs(t, err, errListingStalled, "CancelAllOrders must report a listing which stops advancing")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.cursors, cursors, "the listing should stop rather than request a cursor again")
				exp := make(map[string]string, tc.cancelled)
				for i := tc.orders; i > tc.orders-tc.cancelled; i-- {
					exp[strconv.Itoa(i)] = order.PendingCancel.String()
				}
				assert.Equal(t, exp, resp.Status, "CancelAllOrders should cancel every order the listing returned")
			})
		}
	})

	t.Run("a dead context stops before the algo orders", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			cancelAt string
			paths    []string
		}{
			{name: "during the ordinary orders", cancelAt: "/trade/orders-pending", paths: []string{"/trade/orders-pending"}},
			{name: "during the algo listings", cancelAt: "/trade/orders-algo-pending " + orderTrigger, paths: []string{"/trade/orders-pending", "/trade/cancel-batch-orders", "/trade/orders-algo-pending", "/trade/orders-algo-pending"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				var mu sync.Mutex
				var paths []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					paths = append(paths, r.URL.Path)
					mu.Unlock()
					ordType := r.URL.Query().Get("ordType")
					if r.URL.Path == tc.cancelAt || r.URL.Path+" "+ordType == tc.cancelAt {
						cancel()
					}
					switch r.URL.Path {
					case "/trade/orders-pending":
						writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "1001", "ordType": orderLimit, "side": "buy"}})
					case "/trade/cancel-batch-orders":
						writeOKXData(t, w, []map[string]string{{"ordId": "1001", "sCode": "0"}})
					case "/trade/orders-algo-pending":
						kind, _, _ := strings.Cut(ordType, ",")
						writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "algoId": "2001-" + kind, "ordType": kind, "side": "buy"}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				_, err := ex.CancelAllOrders(ctx, &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
				require.ErrorIs(t, err, context.Canceled, "CancelAllOrders must report the dead context")
				assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()), "CancelAllOrders should report the context error once")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.paths, paths, "no request should be sent once the context is dead")
			})
		}
	})
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil)

	arg := &order.Modify{}
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	arg.Pair = mainPair
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrAssetNotSet)

	arg.AssetType = asset.Spot
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "1234"
	arg.Type = order.Liquidation
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType)

	arg.Type = order.Limit
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	arg.Type = order.Trigger
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	arg.Type = order.OCO
	_, err = e.ModifyOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin)

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		arg = &order.Modify{
			AssetType: asset.Spot,
			Pair:      mainPair,
			OrderID:   "1234",
			Price:     123456.44,
			Amount:    123,
		}
		result, err := e.ModifyOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.Limit
		result, err = e.ModifyOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.Trigger
		_, err = e.ModifyOrder(contextGenerate(), arg)
		require.ErrorIs(t, err, limits.ErrPriceBelowMin)

		arg.TriggerPrice = 12345678
		_, err = e.ModifyOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		arg.Type = order.OCO
		_, err = e.ModifyOrder(contextGenerate(), arg)
		require.ErrorIs(t, err, limits.ErrPriceBelowMin)

		arg.RiskManagementModes = order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 12345677},
			StopLoss:   order.RiskManagement{Price: 12345667},
		}
		result, err = e.ModifyOrder(contextGenerate(), arg)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		result, err = e.ModifyOrder(contextGenerate(),
			&order.Modify{
				AssetType: asset.Spread,
				Pair:      spreadPair,
				OrderID:   "1234",
				Price:     123456.44,
				Amount:    123,
			})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("price-only amend", func(t *testing.T) {
		t.Parallel()
		var amendBody []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/amend-order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			var err error
			amendBody, err = io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading amend request body should not error: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"123","sCode":"0"}]}`))
		}))

		_, err := ex.ModifyOrder(t.Context(), &order.Modify{
			Exchange:  ex.Name,
			AssetType: asset.Spot,
			Pair:      mainPair,
			OrderID:   "123",
			Type:      order.Limit,
			Price:     42000,
		})
		require.NoError(t, err, "ModifyOrder must accept a price-only amend")
		assert.Contains(t, string(amendBody), `"newPx":"42000"`, "a price-only amend should serialise the new price")
	})

	t.Run("fractional amount", func(t *testing.T) {
		t.Parallel()
		var amendBody []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/amend-order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			var err error
			amendBody, err = io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading amend request body should not error: %v", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"123","sCode":"0"}]}`))
		}))

		_, err := ex.ModifyOrder(t.Context(), &order.Modify{
			Exchange:  ex.Name,
			AssetType: asset.Spot,
			Pair:      mainPair,
			OrderID:   "123",
			Type:      order.Limit,
			Amount:    0.001,
			Price:     42000,
		})
		require.NoError(t, err, "ModifyOrder must accept a fractional amount")
		assert.JSONEq(t, `{"instId":"BTC-USDT","ordId":"123","newSz":"0.001","newPx":"42000"}`, string(amendBody), "a fractional amend should reach OKX with its new size and price")
	})

	t.Run("oco amends the stop-loss order price", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var bodies []string
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/amend-algos" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the algo amend request body should not error")
			mu.Lock()
			bodies = append(bodies, string(b))
			mu.Unlock()
			writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0", "sMsg": ""})
		}))
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{
			Exchange:  ex.Name,
			AssetType: asset.Spot,
			Pair:      mainPair,
			OrderID:   "1",
			Type:      order.OCO,
			RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110, LimitPrice: 111},
				StopLoss:   order.RiskManagement{Price: 90, LimitPrice: 89},
			},
		})
		require.NoError(t, err, "ModifyOrder must not error")
		_, err = ex.ModifyOrder(t.Context(), &order.Modify{
			Exchange:  ex.Name,
			AssetType: asset.Spot,
			Pair:      mainPair,
			OrderID:   "1",
			Type:      order.OCO,
			RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110},
				StopLoss:   order.RiskManagement{Price: 90},
			},
		})
		require.NoError(t, err, "ModifyOrder must not error without limit prices")
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, bodies, 2, "both amends must reach OKX")
		var sent AmendAlgoOrderRequest
		require.NoError(t, json.Unmarshal([]byte(bodies[0]), &sent), "the algo amend request body must decode")
		assert.Equal(t, 111.0, sent.NewTakeProfitOrderPrice, "the take profit leg should amend to its limit price")
		assert.Equal(t, 89.0, sent.NewStopLossOrderPrice, "the stop loss leg should amend to its limit price")
		assert.Equal(t, 90.0, sent.NewStopLossTriggerPrice, "the stop loss leg should trigger at the stop loss price")
		assert.NotContains(t, bodies[1], "newSlOrdPx", "an unset stop loss order price should be omitted, not sent as zero")
	})

	t.Run("trigger attach tpsl order prices", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var body []byte
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
				return
			}
			if r.URL.Path != "/trade/amend-algos" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the algo amend request body should not error")
			mu.Lock()
			body = b
			mu.Unlock()
			writeOKXData(t, w, map[string]string{"algoId": "1", "sCode": "0", "sMsg": ""})
		}))
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{
			Exchange:     ex.Name,
			AssetType:    asset.Spot,
			Pair:         mainPair,
			OrderID:      "1",
			Type:         order.Trigger,
			Price:        101,
			TriggerPrice: 100,
			RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110, LimitPrice: 111},
				StopLoss:   order.RiskManagement{Price: 90, LimitPrice: 89},
			},
		})
		require.NoError(t, err, "ModifyOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		var sent AmendAlgoOrderRequest
		require.NoError(t, json.Unmarshal(body, &sent), "the algo amend request body must decode")
		require.Len(t, sent.AttachAlgoOrders, 1, "the amend must attach the take profit and stop loss legs")
		assert.Equal(t, 111.0, sent.AttachAlgoOrders[0].NewTakeProfitOrderPrice, "the attached take profit leg should carry its limit price")
		assert.Equal(t, 89.0, sent.AttachAlgoOrders[0].NewStopLossOrderPrice, "the attached stop loss leg should carry the stop loss limit price, not its trigger price")
		assert.Equal(t, 90.0, sent.AttachAlgoOrders[0].NewStopLossTriggerPrice, "the attached stop loss leg should trigger at the stop loss price")
	})

	t.Run("reports the row error", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/amend-order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":"1","msg":"Request has failed","data":[{"ordId":"123","sCode":"51119","sMsg":"Order modification failed, the order state does not allow amendment"}]}`))
		}))
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{
			Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair,
			OrderID: "123", Type: order.Limit, Price: 42000,
		})
		require.Error(t, err, "a rejected amend must error")
		assert.ErrorContains(t, err, "51119", "the error should report the row's sCode")
		assert.ErrorContains(t, err, "does not allow amendment", "the error should report the row's sMsg")
	})

	t.Run("amends the size alone of a conditional or oco order", func(t *testing.T) {
		t.Parallel()
		for _, oType := range []order.Type{order.ConditionalStop, order.OCO} {
			t.Run(oType.String(), func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/amend-algos", `{"instId":"BTC-USDT","algoId":"803","newSz":"2"}`, []map[string]string{{"algoId": "803", "sCode": "0", "sMsg": ""}})
				resp, err := ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: oType, OrderID: "803", Amount: 2})
				require.NoError(t, err, "ModifyOrder must amend the size alone")
				exp := &order.ModifyResponse{Exchange: ex.Name, OrderID: "803", Pair: mainPair, Amount: 2}
				assert.Equal(t, exp, resp, "ModifyOrder should report the amended size alone")
			})
		}
		t.Run("conditional trigger and order prices", func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/trade/amend-algos", `{"instId":"BTC-USDT","algoId":"803","newSlTriggerPx":"90","newSlOrdPx":"89","newSlTriggerPxType":"mark"}`, []map[string]string{{"algoId": "803", "sCode": "0", "sMsg": ""}})
			resp, err := ex.ModifyOrder(t.Context(), &order.Modify{
				Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: order.ConditionalStop, OrderID: "803", TriggerPrice: 90, Price: 89, TriggerPriceType: order.MarkPrice,
			})
			require.NoError(t, err, "ModifyOrder must amend a conditional order's prices")
			exp := &order.ModifyResponse{Exchange: ex.Name, OrderID: "803", Pair: mainPair, Price: 89, TriggerPrice: 90}
			assert.Equal(t, exp, resp, "ModifyOrder should report the amended prices")
		})
		t.Run("conditional without a change", func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("a refused amendment should not reach OKX: %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			_, err := ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: order.ConditionalStop, OrderID: "803"})
			assert.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation, "an amendment changing nothing should be refused")
		})
	})

	t.Run("claims only what the amendment sent", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			modify *order.Modify
			path   string
			reply  []map[string]string
			exp    *order.ModifyResponse
		}{
			{
				name:   "limit",
				modify: &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, OrderID: "1301", TimeInForce: order.PostOnly, Price: 101, Amount: 1, TriggerPrice: 5},
				path:   "/trade/amend-order",
				reply:  []map[string]string{{"ordId": "1301", "sCode": "0", "sMsg": ""}},
				exp:    &order.ModifyResponse{OrderID: "1301", Pair: mainPair, Price: 101, Amount: 1},
			},
			{
				name:   "size of a limit order naming a side",
				modify: &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, Side: order.Sell, OrderID: "1001", Amount: 2},
				path:   "/trade/amend-order",
				reply:  []map[string]string{{"ordId": "1001", "sCode": "0", "sMsg": ""}},
				exp:    &order.ModifyResponse{OrderID: "1001", Pair: mainPair, Amount: 2},
			},
			{
				name:   "spread",
				modify: &order.Modify{AssetType: asset.Spread, Pair: spreadPair, Type: order.Limit, OrderID: "1302", TimeInForce: order.PostOnly, Price: -5, Amount: 1, TriggerPrice: 5},
				path:   "/sprd/amend-order",
				reply:  []map[string]string{{"ordId": "1302", "sCode": "0", "sMsg": ""}},
				exp:    &order.ModifyResponse{OrderID: "1302", Price: -5, Amount: 1},
			},
			{
				name:   "trigger",
				modify: &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "806", TimeInForce: order.ImmediateOrCancel, Price: 111, Amount: 2, TriggerPrice: 110},
				path:   "/trade/amend-algos",
				reply:  []map[string]string{{"algoId": "806", "sCode": "0", "sMsg": ""}},
				exp:    &order.ModifyResponse{OrderID: "806", Pair: mainPair, Price: 111, Amount: 2, TriggerPrice: 110},
			},
			{
				name: "oco",
				modify: &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.OCO, OrderID: "803", TimeInForce: order.PostOnly, Price: 7, Amount: 2, TriggerPrice: 5, RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90},
				}},
				path:  "/trade/amend-algos",
				reply: []map[string]string{{"algoId": "803", "sCode": "0", "sMsg": ""}},
				exp:   &order.ModifyResponse{OrderID: "803", Pair: mainPair, Amount: 2},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := newRecordingExchange(t, func(string) any { return tc.reply })
				tc.modify.Exchange = ex.Name
				tc.exp.Exchange = ex.Name
				resp, err := ex.ModifyOrder(t.Context(), tc.modify)
				require.NoError(t, err, "ModifyOrder must not error")
				reqs := requests()
				require.Len(t, reqs, 1, "one amendment must be sent")
				assert.Equal(t, tc.path, reqs[0].path, "the amendment should reach its endpoint")
				assert.Equal(t, tc.exp, resp, "ModifyOrder should claim only the changes it sent")
			})
		}
	})

	t.Run("takes the order ID from the reply", func(t *testing.T) {
		t.Parallel()
		limitByClientID := func() *order.Modify {
			return &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, ClientOrderID: "client1301", Amount: 1}
		}
		limitByID := func() *order.Modify {
			return &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, OrderID: "1301", Amount: 1}
		}
		spreadByID := func() *order.Modify {
			return &order.Modify{AssetType: asset.Spread, Pair: spreadPair, OrderID: "1302", Price: -5}
		}
		triggerByID := func() *order.Modify {
			return &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "806", TriggerPrice: 110, Price: -1}
		}
		for _, tc := range []struct {
			name   string
			modify *order.Modify
			reply  any
			expID  string
		}{
			{"limit by client order ID", limitByClientID(), []map[string]string{{"ordId": "1301", "clOrdId": "client1301", "sCode": "0", "sMsg": ""}}, "1301"},
			{"spread by client order ID", &order.Modify{AssetType: asset.Spread, Pair: spreadPair, ClientOrderID: "client1302", Price: -5}, []map[string]string{{"ordId": "1302", "clOrdId": "client1302", "sCode": "0", "sMsg": ""}}, "1302"},
			{"trigger by client order ID", &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, ClientOrderID: "algo806", TriggerPrice: 110, Price: -1}, []map[string]string{{"algoId": "806", "algoClOrdId": "algo806", "sCode": "0", "sMsg": ""}}, "806"},
			{"conditional by client order ID", &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.ConditionalStop, ClientOrderID: "algo803", Amount: 2}, []map[string]string{{"algoId": "803", "algoClOrdId": "algo803", "sCode": "0", "sMsg": ""}}, "803"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := newRecordingExchange(t, func(string) any { return tc.reply })
				tc.modify.Exchange = ex.Name
				resp, err := ex.ModifyOrder(t.Context(), tc.modify)
				require.NoError(t, err, "ModifyOrder must not error")
				require.Len(t, requests(), 1, "one amendment must be sent")
				assert.Equal(t, tc.expID, resp.OrderID, "the response should name the amended order")
			})
		}
		for _, tc := range []struct {
			name   string
			modify *order.Modify
			reply  any
			expErr error
		}{
			{"client order ID with null data", limitByClientID(), nil, common.ErrInvalidResponse},
			{"client order ID with null row", limitByClientID(), []any{nil}, common.ErrInvalidResponse},
			{"client order ID with row without an order ID", limitByClientID(), []map[string]string{{"sCode": "0", "sMsg": ""}}, common.ErrInvalidResponse},
			{"client order ID with a reply echoing another", limitByClientID(), []map[string]string{{"ordId": "9999", "clOrdId": "client9999", "sCode": "0", "sMsg": ""}}, errReplyOrderMismatch},
			{"limit by order ID with null data", limitByID(), nil, common.ErrInvalidResponse},
			{"limit by order ID with a null row", limitByID(), []any{nil}, common.ErrInvalidResponse},
			{"limit by order ID with an empty row", limitByID(), []map[string]string{{}}, errMissingStatusCode},
			{"limit by order ID with a row without a status code", limitByID(), []map[string]string{{"ordId": "1301"}}, errMissingStatusCode},
			{"limit by order ID with a reply naming another order", limitByID(), []map[string]string{{"ordId": "9999", "sCode": "0", "sMsg": ""}}, errReplyOrderMismatch},
			{"spread by order ID with a null row", spreadByID(), []any{nil}, common.ErrInvalidResponse},
			{"spread by order ID with an empty row", spreadByID(), []map[string]string{{}}, errMissingStatusCode},
			{"spread by order ID with a reply naming another order", spreadByID(), []map[string]string{{"ordId": "9999", "sCode": "0", "sMsg": ""}}, errReplyOrderMismatch},
			{"trigger by order ID with a null row", triggerByID(), []any{nil}, common.ErrInvalidResponse},
			{"trigger by order ID with an empty row", triggerByID(), []map[string]string{{}}, errMissingStatusCode},
			{"trigger by order ID with a reply naming another order", triggerByID(), []map[string]string{{"algoId": "9999", "sCode": "0", "sMsg": ""}}, errReplyOrderMismatch},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := newRecordingExchange(t, func(string) any { return tc.reply })
				tc.modify.Exchange = ex.Name
				resp, err := ex.ModifyOrder(t.Context(), tc.modify)
				require.Len(t, requests(), 1, "one amendment must be sent")
				assert.ErrorIs(t, err, tc.expErr, "a reply which confirms no amendment should not be reported as one")
				assert.Nil(t, resp, "a reply which confirms no amendment should give no response")
			})
		}
		t.Run("failed row in a success reply", func(t *testing.T) {
			t.Parallel()
			ex, requests := newRecordingExchange(t, func(string) any {
				return []map[string]string{{"ordId": "1301", "sCode": "51503", "sMsg": "Order modification failed as the order has been filled, canceled or does not exist."}}
			})
			_, err := ex.ModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, OrderID: "1301", Amount: 1})
			require.Len(t, requests(), 1, "one amendment must be sent")
			assert.ErrorContains(t, err, "51503", "a failed row should be reported rather than an amendment")
		})
	})

	t.Run("a trigger amendment states its order price", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			price      float64
			expRequest string
		}{
			{"limit price", 105, `{"instId":"BTC-USDT","algoId":"806","newTriggerPx":"100","newOrdPx":"105","newTriggerPxType":"last"}`},
			{"market", marketOrderPrice, `{"instId":"BTC-USDT","algoId":"806","newTriggerPx":"100","newOrdPx":"-1","newTriggerPxType":"last"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/amend-algos", tc.expRequest, []map[string]string{{"algoId": "806", "sCode": "0", "sMsg": ""}})
				_, err := ex.ModifyOrder(t.Context(), &order.Modify{
					Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "806", Price: tc.price, TriggerPrice: 100,
				})
				require.NoError(t, err, "ModifyOrder must amend the trigger order")
			})
		}
		t.Run("no order price", func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("a refused amendment should not reach OKX: %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			_, err := ex.ModifyOrder(t.Context(), &order.Modify{
				Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "806", TriggerPrice: 100,
			})
			assert.ErrorIs(t, err, limits.ErrPriceBelowMin, "ModifyOrder should refuse a trigger amendment without its order price")
		})
	})

	t.Run("attached legs with only a trigger price execute at market", func(t *testing.T) {
		t.Parallel()
		ex := newAccountPinExchange(t, "2", "/trade/amend-algos",
			`{"instId":"BTC-USDT","algoId":"806","newTriggerPx":"100","newOrdPx":"101","newTriggerPxType":"last","attachAlgoOrds":[{"newTpTriggerPx":"110","newTpOrdPx":"-1","newSlTriggerPx":"90","newSlOrdPx":"-1","newTpTriggerPxType":"last","newSlTriggerPxType":"last"}]}`,
			[]map[string]string{{"algoId": "806", "sCode": "0", "sMsg": ""}})
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{
			Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "806", Price: 101, TriggerPrice: 100,
			RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90}},
		})
		require.NoError(t, err, "ModifyOrder must amend the trigger order")
	})

	t.Run("amends one attached leg of a trigger order", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name       string
			modes      order.RiskManagementModes
			expRequest string
		}{
			{
				name:       "stop loss",
				modes:      order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90}},
				expRequest: `{"instId":"BTC-USDT","algoId":"806","newTriggerPx":"105","newOrdPx":"102","newTriggerPxType":"last","attachAlgoOrds":[{"newSlTriggerPx":"90","newSlOrdPx":"-1","newSlTriggerPxType":"last"}]}`,
			},
			{
				name:       "take profit",
				modes:      order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 120, LimitPrice: 119, TriggerPriceType: order.MarkPrice}},
				expRequest: `{"instId":"BTC-USDT","algoId":"806","newTriggerPx":"105","newOrdPx":"102","newTriggerPxType":"last","attachAlgoOrds":[{"newTpTriggerPx":"120","newTpOrdPx":"119","newTpTriggerPxType":"mark"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newAccountPinExchange(t, "3", "/trade/amend-algos", tc.expRequest, []map[string]string{{"algoId": "806", "sCode": "0", "sMsg": ""}})
				resp, err := ex.ModifyOrder(t.Context(), &order.Modify{
					Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "806", Price: 102, TriggerPrice: 105, RiskManagementModes: tc.modes,
				})
				require.NoError(t, err, "ModifyOrder must amend the trigger order's attached leg")
				exp := &order.ModifyResponse{Exchange: ex.Name, OrderID: "806", Pair: mainPair, Price: 102, TriggerPrice: 105}
				assert.Equal(t, exp, resp, "ModifyOrder should report the amended trigger order")
			})
		}
	})

	t.Run("refuses an attachment it cannot amend", func(t *testing.T) {
		t.Parallel()
		stopLoss := order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90}}
		for _, tc := range []struct {
			name         string
			accountLevel string
			modify       *order.Modify
		}{
			{"limit", "2", &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit, OrderID: "1301", Price: 101, RiskManagementModes: stopLoss}},
			{"order of unknown type", "2", &order.Modify{AssetType: asset.Spot, Pair: mainPair, OrderID: "1301", Amount: 2, RiskManagementModes: stopLoss}},
			{"trigger in spot mode", "1", &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.Trigger, OrderID: "806", Price: 102, TriggerPrice: 105, RiskManagementModes: stopLoss}},
			{"spread", "2", &order.Modify{AssetType: asset.Spread, Pair: spreadPair, Type: order.Limit, OrderID: "1302", Price: -5, RiskManagementModes: stopLoss}},
			{"conditional", "2", &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.ConditionalStop, OrderID: "803", Amount: 2, RiskManagementModes: stopLoss}},
			{"oco stop entry", "2", &order.Modify{AssetType: asset.Spot, Pair: mainPair, Type: order.OCO, OrderID: "803", Amount: 2, RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 100}}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": tc.accountLevel, "posMode": positionModeNet}})
						return
					}
					t.Errorf("a refused amendment should not reach OKX: %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				tc.modify.Exchange = ex.Name
				_, err := ex.ModifyOrder(t.Context(), tc.modify)
				assert.ErrorIs(t, err, errAttachedOrderUnsupported, "ModifyOrder should refuse an attachment it cannot send")
			})
		}
	})

	t.Run("acts for one account when the exchange's credentials change before a retry", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var keys []string
		var ex atomic.Pointer[Exchange]
		mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/amend-order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			mu.Lock()
			keys = append(keys, r.Header.Get("OK-ACCESS-KEY"))
			first := len(keys) == 1
			mu.Unlock()
			if first {
				ex.Load().SetCredentials(secondaryAccount())
				w.Header().Set("Retry-After", "0")
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
		}))
		ex.Store(mock)
		mock.SetCredentials(primaryAccount())
		_, err := mock.ModifyOrder(t.Context(), &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Amount: 2, Price: 1})
		require.NoError(t, err, "ModifyOrder must not error once its retry is accepted")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"primary-key", "primary-key"}, keys, "every attempt should act for the account the amendment started with")
	})
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		result, err := e.GetOrderInfo(contextGenerate(), "123", perpetualSwapPair, asset.PerpetualSwap)
		require.NoError(t, err, "GetOrderInfo must not error for a perpetual swap order")
		if mockTests {
			exp := &order.Detail{
				Amount: 2, Exchange: e.Name, OrderID: "123", ClientOrderID: "swap123", Side: order.Buy, Type: order.Limit,
				Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Status: order.PartiallyFilled, Price: 61200.5,
				AverageExecutedPrice: 61198.2, ExecutedAmount: 1.5, RemainingAmount: 0.5, Fee: 0.0459, FeeAsset: currency.USDT,
				Date: time.UnixMilli(1795409000000), LastUpdated: time.UnixMilli(1795409060000), Leverage: 10,
			}
			assert.Equal(t, exp, result, "GetOrderInfo should map the perpetual swap order")
		} else {
			assert.NotNil(t, result, "GetOrderInfo should return the perpetual swap order")
		}

		result, err = e.GetOrderInfo(contextGenerate(), "123", spreadPair, asset.Spread)
		require.NoError(t, err, "GetOrderInfo must not error for a spread order")
		if mockTests {
			exp := &order.Detail{
				Amount: 3, Exchange: e.Name, OrderID: "123", ClientOrderID: "b1", Side: order.Buy, Type: order.Limit, Pair: spreadPair,
				AssetType: asset.Spread, Status: order.Active, Price: -30.5, Date: time.UnixMilli(1795239700000),
				LastUpdated: time.UnixMilli(1795239700000), RemainingAmount: 3,
			}
			assert.Equal(t, exp, result, "GetOrderInfo should map the spread order")
		} else {
			assert.NotNil(t, result, "GetOrderInfo should return the spread order")
		}
	})
	t.Run("reports execution values", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			assert.Equal(t, "ORD-1", r.URL.Query().Get("ordId"), "the order detail request should identify the order")
			assert.Equal(t, mainPair.String(), r.URL.Query().Get("instId"), "the order detail request should identify the instrument")
			writeOKXData(t, w, map[string]string{
				"instId": mainPair.String(), "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": "filled",
				"sz": "2", "accFillSz": "1.5", "avgPx": "41000", "px": "40000", "fee": "-0.01", "feeCcy": "USDT",
				"cTime": "1700000000000", "uTime": "1700000000000",
			})
		}))
		detail, err := e.GetOrderInfo(t.Context(), "ORD-1", mainPair, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.Equal(t, 41000.0, detail.AverageExecutedPrice, "GetOrderInfo should read the average filled price")
		assert.Equal(t, 41000.0*1.5, detail.Cost, "the cost should be the average filled price times the accumulated fill, not the order price")
		assert.Equal(t, "USDT", detail.CostAsset.String(), "the cost asset should be the pair quote")
		assert.Equal(t, 0.01, detail.Fee, "a fee OKX charged should read as a positive cost, as the websocket stream reports it")
		assert.Equal(t, "USDT", detail.FeeAsset.String(), "the fee asset should come from the response")
	})
	t.Run("looks an unknown order ID up as an algo order", func(t *testing.T) {
		t.Parallel()
		algoRow := func(ordType string, fields map[string]string) map[string]string {
			row := map[string]string{
				"instType": instTypeSpot, "instId": mainPair.String(), "algoId": "ALGO-1", "algoClOrdId": "client-1", "ordType": ordType,
				"side": "buy", "sz": "2", "state": "live", "tgtCcy": "", "lever": "", "reduceOnly": "false",
				"cTime": "1700000000000", "uTime": "1700000001000",
			}
			maps.Copy(row, fields)
			return row
		}
		detail := func(oType order.Type, change func(*order.Detail)) *order.Detail {
			d := &order.Detail{
				Amount: 2, OrderID: "ALGO-1", ClientOrderID: "client-1", Type: oType, Side: order.Buy, Status: order.Active,
				AssetType: asset.Spot, Pair: mainPair, Date: time.UnixMilli(1700000000000), LastUpdated: time.UnixMilli(1700000001000),
			}
			if change != nil {
				change(d)
			}
			return d
		}
		for _, tc := range []struct {
			name   string
			pair   currency.Pair
			a      asset.Item
			row    map[string]string
			exp    *order.Detail
			expErr error
		}{
			{
				name: "trigger", pair: mainPair, a: asset.Spot,
				row: algoRow(orderTrigger, map[string]string{"triggerPx": "110", "ordPx": "111"}),
				exp: detail(order.Trigger, func(d *order.Detail) { d.Price, d.TriggerPrice = 111, 110 }),
			},
			{
				name: "conditional stop loss", pair: mainPair, a: asset.Spot,
				row: algoRow(orderConditional, map[string]string{"slTriggerPx": "90", "slOrdPx": "-1", "state": "effective"}),
				exp: detail(order.ConditionalStop, func(d *order.Detail) {
					d.Price, d.TriggerPrice, d.Status, d.Amount, d.QuoteAmount = -1, 90, order.Closed, 0, 2
				}),
			},
			{
				name: "quote sized conditional market buy", pair: mainPair, a: asset.Spot,
				row: algoRow(orderConditional, map[string]string{"slTriggerPx": "90", "slOrdPx": "-1", "sz": "100", "tgtCcy": targetCurrencyQuote}),
				exp: detail(order.ConditionalStop, func(d *order.Detail) { d.Price, d.TriggerPrice, d.Amount, d.QuoteAmount = -1, 90, 0, 100 }),
			},
			{
				name: "take profit only conditional sell", pair: mainPair, a: asset.Spot,
				row: algoRow(orderConditional, map[string]string{"side": "sell", "sz": "1", "tpTriggerPx": "60000", "tpOrdPx": "59900", "slTriggerPx": "", "slOrdPx": ""}),
				exp: detail(order.ConditionalStop, func(d *order.Detail) { d.Side, d.Amount, d.Price, d.TriggerPrice = order.Sell, 1, 59900, 60000 }),
			},
			{
				name: "one cancels the other", pair: mainPair, a: asset.Spot,
				row: algoRow(orderOCO, map[string]string{"tpTriggerPx": "120", "tpOrdPx": "-1", "slTriggerPx": "90", "slOrdPx": "-1", "state": "canceled"}),
				exp: detail(order.OCO, func(d *order.Detail) { d.Status, d.Amount, d.QuoteAmount = order.Cancelled, 0, 2 }),
			},
			{
				name: "trailing stop", pair: mainPair, a: asset.Spot,
				row: algoRow(orderMoveOrderStop, map[string]string{"activePx": "105", "callbackRatio": "0.05", "state": "pause"}),
				exp: detail(order.TrailingStop, func(d *order.Detail) { d.TriggerPrice, d.Amount, d.QuoteAmount = 105, 0, 2 }),
			},
			{
				name: "spot trailing stop buy sized in the quote currency", pair: mainPair, a: asset.Spot,
				row: algoRow(orderMoveOrderStop, map[string]string{"sz": "100", "callbackRatio": "0.01"}),
				exp: detail(order.TrailingStop, func(d *order.Detail) { d.Amount, d.QuoteAmount = 0, 100 }),
			},
			{
				name: "trailing stop sell", pair: mainPair, a: asset.Spot,
				row: algoRow(orderMoveOrderStop, map[string]string{"side": "sell", "sz": "0.002", "callbackRatio": "0.01"}),
				exp: detail(order.TrailingStop, func(d *order.Detail) { d.Side, d.Amount = order.Sell, 0.002 }),
			},
			{
				name: "margin trailing stop buy sized in the quote currency", pair: mainPair, a: asset.Margin,
				row: algoRow(orderMoveOrderStop, map[string]string{"instType": instTypeMargin, "sz": "100", "callbackRatio": "0.01"}),
				exp: detail(order.TrailingStop, func(d *order.Detail) { d.AssetType, d.Amount, d.QuoteAmount = asset.Margin, 0, 100 }),
			},
			{
				name: "triggered trigger order", pair: mainPair, a: asset.Spot,
				row: algoRow(orderTrigger, map[string]string{"state": "effective", "ordId": "22", "triggerPx": "50000", "ordPx": "49900", "actualSz": "2"}),
				exp: detail(order.Trigger, func(d *order.Detail) { d.Price, d.TriggerPrice, d.Status = 49900, 50000, order.Closed }),
			},
			{
				name: "time weighted", pair: mainPair, a: asset.Spot,
				row: algoRow(orderTWAP, map[string]string{"pxLimit": "100", "state": "partially_effective"}),
				exp: detail(order.TWAP, func(d *order.Detail) { d.Price, d.Status = 100, order.Open }),
			},
			{
				name: "reduce only perpetual swap chase", pair: perpetualSwapPair, a: asset.PerpetualSwap,
				row: algoRow(orderChase, map[string]string{"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "lever": "10", "reduceOnly": "true", "state": "order_failed"}),
				exp: detail(order.Chase, func(d *order.Detail) {
					d.AssetType, d.Pair, d.Leverage, d.ReduceOnly, d.Status = asset.PerpetualSwap, perpetualSwapPair, 10, true, order.Rejected
				}),
			},
			{
				name: "algo order of another instrument", pair: mainPair, a: asset.Spot,
				row:    algoRow(orderTrigger, map[string]string{"instId": "ETH-USDT", "triggerPx": "110", "ordPx": "111"}),
				expErr: order.ErrOrderNotFound,
			},
			{
				name: "unsupported algo order type", pair: mainPair, a: asset.Spot,
				row:    algoRow(orderSmartIceberg, nil),
				expErr: order.ErrTypeIsInvalid,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/trade/order":
						assert.Equal(t, url.Values{"instId": {tc.pair.String()}, "ordId": {"ALGO-1"}}, r.URL.Query(), "the order detail request should send the order ID")
						_, err := w.Write([]byte(`{"code":"51603","msg":"Order does not exist","data":[]}`))
						assert.NoError(t, err, "writing the reply should not error")
					case "/trade/order-algo":
						assert.Equal(t, url.Values{"algoId": {"ALGO-1"}}, r.URL.Query(), "the algo order detail request should send the order ID as the algo ID")
						writeOKXData(t, w, []map[string]string{tc.row})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				got, err := e.GetOrderInfo(t.Context(), "ALGO-1", tc.pair, tc.a)
				if tc.expErr != nil {
					require.ErrorIs(t, err, tc.expErr, "GetOrderInfo must return the expected error")
					return
				}
				require.NoError(t, err, "GetOrderInfo must find the algo order")
				tc.exp.Exchange = e.Name
				assert.Equal(t, tc.exp, got, "GetOrderInfo should map the algo order's details")
			})
		}
	})
	t.Run("rejects a malformed algo order reply", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, data string
			expErr     error
		}{
			{"null", "null", common.ErrInvalidResponse},
			{"null row", "[null]", common.ErrInvalidResponse},
			{"empty row", "[{}]", order.ErrOrderNotFound},
			{"no rows", "[]", nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					reply := `{"code":"51603","msg":"Order does not exist","data":[]}`
					if r.URL.Path == "/trade/order-algo" {
						reply = `{"code":"0","msg":"","data":` + tc.data + `}`
					}
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				var err error
				require.NotPanics(t, func() { _, err = e.GetOrderInfo(t.Context(), "ALGO-1", mainPair, asset.Spot) }, "GetOrderInfo must not panic on a malformed algo order reply")
				require.Error(t, err, "GetOrderInfo must error on a malformed algo order reply")
				if tc.expErr != nil {
					assert.ErrorIs(t, err, tc.expErr, "GetOrderInfo should return the expected error")
				}
			})
		}
	})
	t.Run("rejects a malformed order details reply", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, data string
		}{
			{"null", "null"},
			{"null row", "[null]"},
			{"empty row", "[{}]"},
			{"row without an order ID", `[{"instId":"BTC-USDT","ordType":"limit","side":"buy","state":"live","sz":"1","px":"60000","cTime":"1700000000000"}]`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/trade/order", r.URL.Path, "an order details reply should not be looked up as an algo order")
					_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				var err error
				require.NotPanics(t, func() { _, err = e.GetOrderInfo(t.Context(), "ORD-1", mainPair, asset.Spot) }, "GetOrderInfo must not panic on a malformed order details reply")
				assert.ErrorIs(t, err, common.ErrInvalidResponse, "GetOrderInfo should reject a success reply without an order")
			})
		}
	})
	t.Run("looks the algo order up only when the order does not exist", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, reply string
			code        int64
			algoLookup  bool
		}{
			{"order does not exist", `{"code":"51603","msg":"Order does not exist","data":[]}`, 51603, true},
			{"order does not exist without a message", `{"code":"51603","msg":"","data":[]}`, 51603, true},
			{"another error", `{"code":"50001","msg":"Service temporarily unavailable","data":[]}`, 50001, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var algoLookups atomic.Int32
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/trade/order":
						_, err := w.Write([]byte(tc.reply))
						assert.NoError(t, err, "writing the reply should not error")
					case "/trade/order-algo":
						algoLookups.Add(1)
						_, err := w.Write([]byte(`{"code":"51000","msg":"Parameter algoId error","data":[]}`))
						assert.NoError(t, err, "writing the reply should not error")
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				_, err := e.GetOrderInfo(t.Context(), "ORD-1", mainPair, asset.Spot)
				require.Error(t, err, "GetOrderInfo must error when neither listing knows the order")
				var respErr *ResponseError
				require.ErrorAs(t, err, &respErr, "GetOrderInfo must keep OKX's error code")
				assert.Equal(t, tc.code, respErr.Code, "GetOrderInfo should report the order lookup's error first")
				if tc.algoLookup {
					assert.EqualValues(t, 1, algoLookups.Load(), "GetOrderInfo should look an unknown order ID up as an algo ID")
					assert.ErrorContains(t, err, "Parameter algoId error", "GetOrderInfo should report the algo lookup's error too")
				} else {
					assert.Zero(t, algoLookups.Load(), "GetOrderInfo should not look the order up as an algo order after another error")
				}
			})
		}
	})
	t.Run("reports a maker rebate as a negative fee", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{makerRebateOrderRow("filled", "2")})
		}))
		detail, err := e.GetOrderInfo(t.Context(), "ORD-1", mainPair, asset.Spot)
		require.NoError(t, err, "GetOrderInfo must not error")
		exp := &order.Detail{
			Price: 100, Amount: 2, AverageExecutedPrice: 100, ExecutedAmount: 2, Cost: 200, CostAsset: currency.USDT,
			Fee: -0.5, FeeAsset: currency.USDT, Exchange: e.Name, OrderID: "ORD-1", ClientOrderID: "client-1",
			Type: order.Limit, Side: order.Buy, Status: order.Filled, AssetType: asset.Spot,
			Date: time.UnixMilli(1700000000000), LastUpdated: time.UnixMilli(1700000001000), Pair: mainPair,
		}
		assert.Equal(t, exp, detail, "GetOrderInfo should net the maker rebate into the fee")
	})
	t.Run("reports the leverage", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, map[string]string{
				"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "ordId": "ORD-1", "ordType": orderLimit, "side": "buy",
				"state": "live", "sz": "100", "px": "60000", "lever": "10", "cTime": "1700000000000", "uTime": "1700000000000",
			})
		}))
		detail, err := e.GetOrderInfo(t.Context(), "ORD-1", perpetualSwapPair, asset.PerpetualSwap)
		require.NoError(t, err, "GetOrderInfo must not error")
		assert.Equal(t, 10.0, detail.Leverage, "GetOrderInfo should report the order's leverage")
	})
	t.Run("rejects a spread reply without a row", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/sprd/order" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, nil)
		}))
		var err error
		require.NotPanics(t, func() {
			_, err = e.GetOrderInfo(t.Context(), "SPD-1", spreadPair, asset.Spread)
		}, "GetOrderInfo must not panic on a success reply without a row")
		require.ErrorIs(t, err, common.ErrInvalidResponse, "GetOrderInfo must reject a success reply without a row")
	})
	t.Run("spread order details", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name                string
			ordType             string
			state               string
			sz                  string
			accFillSz           string
			oType               order.Type
			tif                 order.TimeInForce
			executed, remaining float64
		}{
			{"limit", orderLimit, "live", "1", "0.3", order.Limit, order.UnknownTIF, 0.3, 0.7},
			{"post_only", orderPostOnly, "live", "1", "0.3", order.Limit, order.PostOnly, 0.3, 0.7},
			{"ioc", orderIOC, "live", "1", "0.3", order.Limit, order.ImmediateOrCancel, 0.3, 0.7},
			{"op_fok", orderOptionFOK, "live", "1", "0.3", order.Limit, order.FillOrKill, 0.3, 0.7},
			{"filled", orderLimit, "filled", "1", "1", order.Limit, order.UnknownTIF, 1, 0},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				spreadRow := map[string]string{
					"sprdId":    "BTC-USDT_BTC-USDT",
					"ordId":     "1",
					"ordType":   tc.ordType,
					"side":      "buy",
					"state":     tc.state,
					"cTime":     "1700000000000",
					"sz":        tc.sz,
					"accFillSz": tc.accFillSz,
					"px":        "100",
					"avgPx":     "90",
				}
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeOKXData(t, w, spreadRow)
				}))

				detail, err := e.GetOrderInfo(t.Context(), "1", currency.Pair{}, asset.Spread)
				require.NoError(t, err, "GetOrderInfo must not error for a spread order")
				format, err := e.GetPairFormat(asset.Spread, true)
				require.NoError(t, err, "GetPairFormat must not error for the spread asset")
				expPair, err := currency.NewPairDelimiter("BTC-USDT_BTC-USDT", format.Delimiter)
				require.NoError(t, err, "the expected spread pair must parse")
				assert.Equal(t, expPair, detail.Pair, "GetOrderInfo should parse the pair from the spread ID like the other spread listings")
				assert.Equal(t, tc.oType, detail.Type, "GetOrderInfo should read the order type")
				assert.Equal(t, tc.tif, detail.TimeInForce, "GetOrderInfo should preserve the time in force")
				assert.Equal(t, tc.executed, detail.ExecutedAmount, "GetOrderInfo should report the accumulated fill")
				assert.Equal(t, tc.remaining, detail.RemainingAmount, "GetOrderInfo should report the unfilled remainder")
				assert.Equal(t, 90.0, detail.AverageExecutedPrice, "GetOrderInfo should read the average filled price")
				assert.Zero(t, detail.Cost, "GetOrderInfo should leave a spread order's cost unset")
			})
		}
	})
	t.Run("reports each order as OKX states it", func(t *testing.T) {
		t.Parallel()
		for _, tc := range statedOrderCases("partially_filled", order.PartiallyFilled) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newQueryPinExchange(t, "/trade/order", url.Values{"instId": {tc.row["instId"]}, "ordId": {tc.row["ordId"]}}, []map[string]string{tc.row})
				got, err := ex.GetOrderInfo(t.Context(), tc.row["ordId"], tc.exp.Pair, tc.a)
				require.NoError(t, err, "GetOrderInfo must not error")
				exp := tc.exp
				exp.Exchange = ex.Name
				assert.Equal(t, &exp, got, "GetOrderInfo should report the order's sizes, trade currency and reduce only flag as OKX states them")
			})
		}
	})
	t.Run("reports the asset the reply's instrument type names", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			algo      bool
			requested asset.Item
			pair      currency.Pair
			instType  string
			exp       asset.Item
			err       error
		}{
			{name: "margin order looked up as spot", requested: asset.Spot, pair: mainPair, instType: instTypeMargin, exp: asset.Margin},
			{name: "spot order looked up as margin", requested: asset.Margin, pair: mainPair, instType: instTypeSpot, exp: asset.Spot},
			{name: "spot order looked up as spot", requested: asset.Spot, pair: mainPair, instType: instTypeSpot, exp: asset.Spot},
			{name: "futures order looked up as perpetual swap", requested: asset.PerpetualSwap, pair: perpetualSwapPair, instType: instTypeFutures, err: order.ErrOrderNotFound},
			{name: "margin algo order looked up as spot", algo: true, requested: asset.Spot, pair: mainPair, instType: instTypeMargin, exp: asset.Margin},
			{name: "spot algo order looked up as margin", algo: true, requested: asset.Margin, pair: mainPair, instType: instTypeSpot, exp: asset.Spot},
			{name: "futures algo order looked up as perpetual swap", algo: true, requested: asset.PerpetualSwap, pair: perpetualSwapPair, instType: instTypeFutures, err: order.ErrOrderNotFound},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch {
					case r.URL.Path == "/trade/order" && tc.algo:
						_, err := w.Write([]byte(`{"code":"51603","msg":"Order does not exist","data":[]}`))
						assert.NoError(t, err, "writing the reply should not error")
					case r.URL.Path == "/trade/order":
						writeOKXData(t, w, []map[string]string{{
							"instType": tc.instType, "instId": tc.pair.String(), "ordId": "84", "ordType": orderLimit, "side": "sell", "state": "live",
							"sz": "1", "px": "60000", "cTime": "1795409000000",
						}})
					case r.URL.Path == "/trade/order-algo" && tc.algo:
						writeOKXData(t, w, []map[string]string{{
							"instType": tc.instType, "instId": tc.pair.String(), "algoId": "84", "ordType": orderTrigger, "side": "sell", "state": "live",
							"sz": "1", "triggerPx": "61000", "ordPx": "60000", "cTime": "1795409000000",
						}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				got, err := ex.GetOrderInfo(t.Context(), "84", tc.pair, tc.requested)
				require.ErrorIs(t, err, tc.err, "GetOrderInfo must return the expected error")
				if tc.err != nil {
					return
				}
				exp := &order.Detail{
					Amount: 1, Exchange: ex.Name, OrderID: "84", Side: order.Sell, Type: order.Trigger, Pair: tc.pair, AssetType: tc.exp,
					Status: order.Active, Price: 60000, TriggerPrice: 61000, Date: time.UnixMilli(1795409000000),
				}
				if !tc.algo {
					exp.Type, exp.TriggerPrice, exp.RemainingAmount, exp.CostAsset = order.Limit, 0, 1, currency.USDT
				}
				assert.Equal(t, exp, got, "GetOrderInfo should report the order under the asset its instrument type names")
			})
		}
	})
	t.Run("reports the spread size still to fill", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			fields    map[string]string
			status    order.Status
			executed  float64
			remaining float64
		}{
			{name: "the documented example", status: order.Active, remaining: 2},
			{name: "an explicit zero", fields: map[string]string{"pendingFillSz": "0", "state": "canceled", "accFillSz": "1"}, status: order.Cancelled, executed: 1},
			{name: "no pending fill size", fields: map[string]string{"pendingFillSz": ""}, status: order.Active, remaining: 3},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				row := map[string]string{
					"sprdId": spreadPair.String(), "ordId": "312269865356374016", "clOrdId": "b1", "px": "999", "sz": "3", "ordType": orderLimit, "side": "buy",
					"accFillSz": "0", "pendingFillSz": "2", "pendingSettleSz": "1", "canceledSz": "1", "state": "live", "uTime": "1597026383085", "cTime": "1597026383085",
				}
				maps.Copy(row, tc.fields)
				ex := newQueryPinExchange(t, "/sprd/order", url.Values{"ordId": {"312269865356374016"}}, []map[string]string{row})
				got, err := ex.GetOrderInfo(t.Context(), "312269865356374016", spreadPair, asset.Spread)
				require.NoError(t, err, "GetOrderInfo must not error")
				exp := &order.Detail{
					Amount: 3, Exchange: ex.Name, OrderID: "312269865356374016", ClientOrderID: "b1", Side: order.Buy, Type: order.Limit, Pair: spreadPair,
					AssetType: asset.Spread, Status: tc.status, Price: 999, ExecutedAmount: tc.executed, RemainingAmount: tc.remaining,
					Date: time.UnixMilli(1597026383085), LastUpdated: time.UnixMilli(1597026383085),
				}
				assert.Equal(t, exp, got, "GetOrderInfo should report the size OKX states is still to fill")
			})
		}
	})
}

func TestAlgoOrderInfo(t *testing.T) {
	t.Parallel()
	created, updated := time.UnixMilli(1795231500000), time.UnixMilli(1795231800000)
	row := func(algoID string, edit func(map[string]string)) map[string]string {
		r := map[string]string{
			"algoId": algoID, "algoClOrdId": "algoClient", "instId": mainPair.String(), "ordType": orderTrigger, "side": "buy", "sz": "2", "state": "live",
			"lever": "3", "reduceOnly": "false", "ordPx": "100", "triggerPx": "95", "slOrdPx": "90", "slTriggerPx": "91", "activePx": "97", "pxLimit": "105",
			"cTime": "1795231500000", "uTime": "1795231800000",
		}
		if edit != nil {
			edit(r)
		}
		return r
	}
	detail := func(algoID string, edit func(*order.Detail)) *order.Detail {
		d := &order.Detail{
			Amount: 2, Exchange: "Okx", OrderID: algoID, ClientOrderID: "algoClient", Side: order.Buy, Type: order.Trigger, Pair: mainPair, AssetType: asset.Spot,
			Status: order.Active, Leverage: 3, Date: created, LastUpdated: updated,
		}
		edit(d)
		return d
	}
	for _, tc := range []struct {
		name string
		row  map[string]string
		exp  *order.Detail
		err  error
	}{
		{name: "trigger order", row: row("1", nil), exp: detail("1", func(d *order.Detail) { d.Price, d.TriggerPrice = 100, 95 })},
		{
			name: "conditional order", row: row("2", func(r map[string]string) { r["ordType"] = orderConditional }),
			exp: detail("2", func(d *order.Detail) { d.Type, d.Price, d.TriggerPrice = order.ConditionalStop, 90, 91 }),
		},
		{
			name: "trailing stop", row: row("3", func(r map[string]string) { r["ordType"] = orderMoveOrderStop }),
			exp: detail("3", func(d *order.Detail) { d.Type, d.TriggerPrice, d.Amount, d.QuoteAmount = order.TrailingStop, 97, 0, 2 }),
		},
		{
			name: "take profit only conditional order", row: row("12", func(r map[string]string) {
				r["ordType"], r["slOrdPx"], r["slTriggerPx"], r["tpOrdPx"], r["tpTriggerPx"] = orderConditional, "", "", "120", "121"
			}),
			exp: detail("12", func(d *order.Detail) { d.Type, d.Price, d.TriggerPrice = order.ConditionalStop, 120, 121 }),
		},
		{
			name: "take profit limit leg of a conditional order", row: row("13", func(r map[string]string) {
				r["ordType"], r["slOrdPx"], r["slTriggerPx"], r["tpOrdPx"] = orderConditional, "", "", "120"
			}),
			exp: detail("13", func(d *order.Detail) { d.Type, d.Price = order.ConditionalStop, 120 }),
		},
		{
			name: "conditional order with both legs", row: row("14", func(r map[string]string) {
				r["ordType"], r["tpOrdPx"], r["tpTriggerPx"] = orderConditional, "120", "121"
			}),
			exp: detail("14", func(d *order.Detail) { d.Type, d.Price, d.TriggerPrice = order.ConditionalStop, 90, 91 }),
		},
		{
			name: "conditional buy with its stop loss at market and its ignored take profit at a limit price", row: row("18", func(r map[string]string) {
				r["ordType"], r["slOrdPx"], r["tpOrdPx"], r["tpTriggerPx"] = orderConditional, "-1", "120", "121"
			}),
			exp: detail("18", func(d *order.Detail) {
				d.Type, d.Price, d.TriggerPrice, d.Amount, d.QuoteAmount = order.ConditionalStop, -1, 91, 0, 2
			}),
		},
		{
			name: "conditional buy with its stop loss at a limit price and its ignored take profit at market", row: row("19", func(r map[string]string) {
				r["ordType"], r["tpOrdPx"], r["tpTriggerPx"] = orderConditional, "-1", "121"
			}),
			exp: detail("19", func(d *order.Detail) { d.Type, d.Price, d.TriggerPrice = order.ConditionalStop, 90, 91 }),
		},
		{
			name: "trigger order placing an immediate or cancel order once triggered", row: row("20", func(r map[string]string) { r["advanceOrdType"] = orderIOC }),
			exp: detail("20", func(d *order.Detail) { d.Price, d.TriggerPrice, d.TimeInForce = 100, 95, order.ImmediateOrCancel }),
		},
		{
			name: "stop loss only conditional order at market", row: row("15", func(r map[string]string) {
				r["ordType"], r["slOrdPx"], r["tpOrdPx"], r["tpTriggerPx"] = orderConditional, "-1", "", ""
			}),
			exp: detail("15", func(d *order.Detail) {
				d.Type, d.Price, d.TriggerPrice, d.Amount, d.QuoteAmount = order.ConditionalStop, -1, 91, 0, 2
			}),
		},
		{
			name: "margin order", row: row("16", func(r map[string]string) { r["instType"] = instTypeMargin }),
			exp: detail("16", func(d *order.Detail) { d.AssetType, d.Price, d.TriggerPrice = asset.Margin, 100, 95 }),
		},
		{name: "another instrument type", row: row("17", func(r map[string]string) { r["instType"] = instTypeSwap }), err: order.ErrOrderNotFound},
		{name: "TWAP order", row: row("4", func(r map[string]string) { r["ordType"] = orderTWAP }), exp: detail("4", func(d *order.Detail) { d.Type, d.Price = order.TWAP, 105 })},
		{name: "OCO order", row: row("5", func(r map[string]string) { r["ordType"] = orderOCO }), exp: detail("5", func(d *order.Detail) { d.Type = order.OCO })},
		{name: "chase order", row: row("6", func(r map[string]string) { r["ordType"] = orderChase }), exp: detail("6", func(d *order.Detail) { d.Type = order.Chase })},
		{
			name: "sized in the quote currency", row: row("7", func(r map[string]string) { r["tgtCcy"] = targetCurrencyQuote }),
			exp: detail("7", func(d *order.Detail) { d.Amount, d.QuoteAmount, d.Price, d.TriggerPrice = 0, 2, 100, 95 }),
		},
		{
			name: "triggered reduce only sell", row: row("8", func(r map[string]string) { r["side"], r["state"], r["reduceOnly"] = "sell", "effective", "true" }),
			exp: detail("8", func(d *order.Detail) {
				d.Side, d.Status, d.ReduceOnly, d.Price, d.TriggerPrice = order.Sell, order.Closed, true, 100, 95
			}),
		},
		{name: "another instrument", row: row("9", func(r map[string]string) { r["instId"] = "ETH-USDT" }), err: order.ErrOrderNotFound},
		{name: "unknown order type", row: row("10", func(r map[string]string) { r["ordType"] = "iceberg" }), err: order.ErrTypeIsInvalid},
		{name: "unknown state", row: row("11", func(r map[string]string) { r["state"] = "unknown" }), err: order.ErrUnsupportedStatusType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newQueryPinExchange(t, "/trade/order-algo", url.Values{"algoId": {tc.row["algoId"]}}, []map[string]string{tc.row})
			d, err := ex.algoOrderInfo(t.Context(), tc.row["algoId"], mainPair.String(), mainPair, asset.Spot)
			require.ErrorIs(t, err, tc.err, "algoOrderInfo must return the expected error")
			assert.Equal(t, tc.exp, d, "algoOrderInfo should report the algo order where SubmitOrder places its prices")
		})
	}

	for _, tc := range []struct {
		name  string
		reply string
		err   error
	}{
		{name: "request error", reply: `{"code":"51603","msg":"","data":[]}`, err: ErrorCodes["51603"]},
		{name: "no reply row", reply: `{"code":"0","msg":"","data":null}`, err: common.ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte(tc.reply))
				assert.NoError(t, err, "writing the reply should not error")
			}))
			d, err := ex.algoOrderInfo(t.Context(), "1", mainPair.String(), mainPair, asset.Spot)
			require.ErrorIs(t, err, tc.err, "algoOrderInfo must return the expected error")
			assert.Nil(t, d, "algoOrderInfo should return no detail beside the error")
		})
	}
}

func TestAlgoOrderType(t *testing.T) {
	t.Parallel()
	for ordType, exp := range map[string]order.Type{
		orderTrigger:       order.Trigger,
		"CONDITIONAL":      order.ConditionalStop,
		orderOCO:           order.OCO,
		orderMoveOrderStop: order.TrailingStop,
		orderTWAP:          order.TWAP,
		orderChase:         order.Chase,
	} {
		t.Run(ordType, func(t *testing.T) {
			t.Parallel()
			oType, err := algoOrderType(ordType)
			require.NoError(t, err, "algoOrderType must not error")
			assert.Equal(t, exp, oType, "algoOrderType should return the order type SubmitOrder places it as")
		})
	}
	for _, ordType := range []string{orderSmartIceberg, orderLimit, ""} {
		t.Run("unsupported "+ordType, func(t *testing.T) {
			t.Parallel()
			_, err := algoOrderType(ordType)
			require.ErrorIs(t, err, order.ErrTypeIsInvalid, "algoOrderType must reject an order type SubmitOrder cannot place")
		})
	}
}

func TestAlgoOrderStatus(t *testing.T) {
	t.Parallel()
	for state, exp := range map[string]order.Status{
		"live":                order.Active,
		"pause":               order.Active,
		"partially_effective": order.Open,
		"EFFECTIVE":           order.Closed,
		"canceled":            order.Cancelled,
		"order_failed":        order.Rejected,
		"partially_failed":    order.PartiallyCancelled,
	} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			status, err := algoOrderStatus(state)
			require.NoError(t, err, "algoOrderStatus must not error")
			assert.Equal(t, exp, status, "algoOrderStatus should map the algo order state")
		})
	}
	t.Run("unknown state", func(t *testing.T) {
		t.Parallel()
		_, err := algoOrderStatus("filled")
		require.ErrorIs(t, err, order.ErrUnsupportedStatusType, "algoOrderStatus must reject a state algo orders do not report")
	})
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(contextGenerate(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetDepositAddress(contextGenerate(), currency.BTC, core.BitcoinDonationAddress, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		withdrawCryptoRequest := withdraw.Request{
			Exchange: e.Name,
			Amount:   -0.1,
			Currency: currency.BTC,
			Crypto: withdraw.CryptoRequest{
				Address: core.BitcoinDonationAddress,
			},
		}
		result, err := e.WithdrawCryptocurrencyFunds(contextGenerate(), &withdrawCryptoRequest)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	privateRecipient := withdraw.TravelRule{
		BeneficiaryWalletType: "private",
		IsSelf:                true,
		BeneficiaryName:       "Test Recipient",
		BeneficiaryAddress: withdraw.TravelAddress{
			Address1: "Level 2", Address3: "1 George Street", City: "Sydney", State: "NSW", Country: "AU", PostalCode: "2000",
		},
		BeneficiaryFinancialInstitution: "Test Bank",
		TransferPurpose:                 "Rent",
	}
	t.Run("sends the withdrawal method", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  withdraw.Request
			body string
		}{
			{
				name: "on-chain withdrawal",
				req:  withdraw.Request{Currency: currency.BTC, Amount: 0.1, Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress, Chain: "BTC-Bitcoin"}},
				body: `{"ccy":"BTC","amt":"0.1","dest":"4","toAddr":"` + core.BitcoinDonationAddress + `","chain":"BTC-Bitcoin"}`,
			},
			{
				name: "on-chain withdrawal with an address tag",
				req:  withdraw.Request{Currency: currency.XRP, Amount: 20, Crypto: withdraw.CryptoRequest{Address: "rDestination", AddressTag: "123456", Chain: "XRP-Ripple"}},
				body: `{"ccy":"XRP","amt":"20","dest":"4","toAddr":"rDestination:123456","chain":"XRP-Ripple"}`,
			},
			{
				name: "internal transfer",
				req:  withdraw.Request{Currency: currency.BTC, Amount: 0.1, InternalTransfer: true, Crypto: withdraw.CryptoRequest{Address: "recipient@example.com", AddressTag: "123456"}},
				body: `{"ccy":"BTC","amt":"0.1","dest":"3","toAddr":"recipient@example.com"}`,
			},
			{
				name: "on-chain withdrawal with a client ID and a private wallet recipient",
				req: withdraw.Request{
					Currency: currency.BTC, Amount: 0.1, ClientOrderID: "withdraw123",
					Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress, Chain: "BTC-Bitcoin"}, Travel: privateRecipient,
				},
				body: `{"ccy":"BTC","amt":"0.1","dest":"4","toAddr":"` + core.BitcoinDonationAddress + `","chain":"BTC-Bitcoin","clientId":"withdraw123",` +
					`"rcvrInfo":{"walletType":"private","rcvrCountry":"AU","rcvrCountrySubDivision":"NSW","rcvrTownName":"Sydney","rcvrStreetName":"Level 2, 1 George Street"}}`,
			},
			{
				name: "on-chain withdrawal to a private wallet named alone",
				req: withdraw.Request{
					Currency: currency.BTC, Amount: 0.1, Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress},
					Travel: withdraw.TravelRule{BeneficiaryWalletType: "private"},
				},
				body: `{"ccy":"BTC","amt":"0.1","dest":"4","toAddr":"` + core.BitcoinDonationAddress + `","rcvrInfo":{"walletType":"private"}}`,
			},
			{
				name: "internal transfer with a client ID, which takes no recipient",
				req: withdraw.Request{
					Currency: currency.BTC, Amount: 0.1, InternalTransfer: true, ClientOrderID: "transfer123",
					Crypto: withdraw.CryptoRequest{Address: "recipient@example.com"}, Travel: privateRecipient,
				},
				body: `{"ccy":"BTC","amt":"0.1","dest":"3","toAddr":"recipient@example.com","clientId":"transfer123"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method, "the withdrawal should be a POST")
					assert.Equal(t, "/asset/withdrawal", r.URL.Path, "the withdrawal should reach the withdrawal endpoint")
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the request body should not error")
					assert.JSONEq(t, tc.body, string(body), "the withdrawal should send the method its address takes, its client ID and its recipient")
					writeOKXData(t, w, []map[string]string{{"amt": "0.1", "wdId": "67485", "ccy": "BTC", "clientId": "", "chain": "BTC-Bitcoin"}})
				}))
				tc.req.Exchange = ex.Name
				resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &tc.req)
				require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
				assert.Equal(t, &withdraw.ExchangeResponse{ID: "67485"}, resp, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
			})
		}
	})

	t.Run("refuses a recipient it cannot send", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			travel withdraw.TravelRule
			expErr error
		}{
			{"exchange wallet", withdraw.TravelRule{BeneficiaryWalletType: "exchange", BeneficiaryName: "Bruce Wayne"}, common.ErrFunctionNotSupported},
			{"capitalised exchange wallet", withdraw.TravelRule{BeneficiaryWalletType: "Exchange", BeneficiaryName: "Bruce Wayne"}, common.ErrFunctionNotSupported},
			{"recipient without a wallet type", withdraw.TravelRule{BeneficiaryName: "Test Recipient", BeneficiaryAddress: withdraw.TravelAddress{Country: "AU"}}, errWalletTypeRequired},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Failf(t, "unexpected request", "a refused withdrawal should not reach %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
					Exchange: ex.Name, Currency: currency.BTC, Amount: 0.1, Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress}, Travel: tc.travel,
				})
				assert.ErrorIs(t, err, tc.expErr, "WithdrawCryptocurrencyFunds should refuse a recipient it cannot send")
			})
		}
	})

	t.Run("rejects a malformed withdrawal reply", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, data string
		}{
			{"null", "null"},
			{"null row", "[null]"},
			{"empty row", "[{}]"},
			{"row without a withdrawal ID", `[{"amt":"0.1","ccy":"BTC","clientId":"","chain":"BTC-Bitcoin"}]`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/asset/withdrawal", r.URL.Path, "the withdrawal should reach the withdrawal endpoint")
					_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				var err error
				require.NotPanics(t, func() {
					_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
						Exchange: ex.Name, Currency: currency.BTC, Amount: 0.1, Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress},
					})
				}, "WithdrawCryptocurrencyFunds must not panic on a malformed reply")
				assert.ErrorIs(t, err, common.ErrInvalidResponse, "WithdrawCryptocurrencyFunds should reject a success reply without a withdrawal ID")
			})
		}
	})
}

func TestWithdrawalRecipient(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		travel withdraw.TravelRule
		exp    *WithdrawalRecipientInformation
		expErr error
	}{
		{name: "no travel rule"},
		{
			name: "private wallet",
			travel: withdraw.TravelRule{
				BeneficiaryWalletType: "private", IsSelf: true, BeneficiaryName: "Test Recipient", TransferPurpose: "Rent", BeneficiaryFinancialInstitution: "Test Bank",
				BeneficiaryAddress: withdraw.TravelAddress{Address1: "1 George Street", Address2: "Level 2", Address3: "Suite 3", City: "Sydney", State: "NSW", Country: "AU", PostalCode: "2000"},
			},
			exp: &WithdrawalRecipientInformation{
				WalletType: "private", ReceiverCountry: "AU", ReceiverCountrySubDivision: "NSW", ReceiverTownName: "Sydney", ReceiverStreetName: "1 George Street, Level 2, Suite 3",
			},
		},
		{
			name:   "street lines with gaps",
			travel: withdraw.TravelRule{BeneficiaryWalletType: "private", BeneficiaryAddress: withdraw.TravelAddress{Address2: "Level 2"}},
			exp:    &WithdrawalRecipientInformation{WalletType: "private", ReceiverStreetName: "Level 2"},
		},
		{
			name:   "recipient without a wallet type",
			travel: withdraw.TravelRule{IsSelf: true},
			exp:    &WithdrawalRecipientInformation{},
		},
		{name: "exchange wallet", travel: withdraw.TravelRule{BeneficiaryWalletType: "exchange"}, expErr: common.ErrFunctionNotSupported},
		{name: "upper case exchange wallet", travel: withdraw.TravelRule{BeneficiaryWalletType: "EXCHANGE"}, expErr: common.ErrFunctionNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := withdrawalRecipient(&tc.travel)
			require.ErrorIs(t, err, tc.expErr, "withdrawalRecipient must return the expected error")
			assert.Equal(t, tc.exp, got, "withdrawalRecipient should map the recipient fields OKX takes")
		})
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}

		// The recorded listings also hold a sell and other instruments' orders, which the request filters out
		mockOrders := map[asset.Item]order.FilteredOrders{
			asset.Spot: {
				{
					Amount: 2, Pair: currency.NewPairWithDelimiter("LTC", "USDT", currency.DashDelimiter), Price: 72.5, CostAsset: currency.USDT,
					RemainingAmount: 2, FeeAsset: currency.LTC, Exchange: e.Name, OrderID: "1752588852617379850", ClientOrderID: "ltc1",
					Type: order.Limit, Side: order.Buy, Status: order.Active, AssetType: asset.Spot, Date: time.UnixMilli(1795409100000),
					LastUpdated: time.UnixMilli(1795409100000),
				},
				{
					Amount: 0.01, Pair: mainPair, Price: 61000.1, AverageExecutedPrice: 61000.1, Cost: 244.0004, CostAsset: currency.USDT,
					ExecutedAmount: 0.004, RemainingAmount: 0.006, Fee: 0.000004, FeeAsset: currency.BTC, Exchange: e.Name,
					OrderID: "1752588852617379851", ClientOrderID: "btc1", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled,
					AssetType: asset.Spot, Date: time.UnixMilli(1795409120000), LastUpdated: time.UnixMilli(1795409150000), TimeInForce: order.PostOnly,
				},
			},
			asset.Spread: {
				{
					Amount: 3, Pair: spreadPair, Price: -31.5, AverageExecutedPrice: -31.5, ExecutedAmount: 1, RemainingAmount: 2,
					Exchange: e.Name, OrderID: "312269865356374016", ClientOrderID: "b1", Type: order.Limit, Side: order.Buy,
					Status: order.PartiallyFilled, AssetType: asset.Spread, Date: time.UnixMilli(1795409200000), LastUpdated: time.UnixMilli(1795409260000),
				},
			},
		}
		for _, a := range []asset.Item{asset.Spot, asset.Spread} {
			pairs := []currency.Pair{currency.NewPair(currency.LTC, currency.USDT), mainPair}
			if a == asset.Spread {
				pairs = []currency.Pair{spreadPair}
			}

			result, err := e.GetActiveOrders(contextGenerate(), &order.MultiOrderRequest{
				Type:      order.Limit,
				Pairs:     pairs,
				AssetType: a,
				Side:      order.Buy,
			})
			require.NoErrorf(t, err, "GetActiveOrders for asset %s and pair %s must not error", a, pairs[0])
			if mockTests {
				assert.Equalf(t, mockOrders[a], result, "GetActiveOrders should map the requested %s orders", a)
			} else {
				assert.NotNilf(t, result, "GetActiveOrders should return the %s orders", a)
			}
		}
	})

	t.Run("paginates with the order ID cursor", func(t *testing.T) {
		t.Parallel()
		fillPage := func(first, count int) []map[string]string {
			ords := make([]map[string]string, 0, count)
			for x := range count {
				ords = append(ords, map[string]string{
					"instId":    "BTC-USDT",
					"ordId":     fmt.Sprintf("ORD-%03d", first+x),
					"cTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
					"uTime":     strconv.FormatInt(1700000000000+int64(first+x), 10),
					"state":     "live",
					"ordType":   "limit",
					"side":      "buy",
					"sz":        "1",
					"px":        "42000",
					"accFillSz": "0",
					"feeCcy":    "USDT",
					"fee":       "0",
				})
			}
			return ords
		}
		pages := map[string][]map[string]string{
			"":        fillPage(0, orderListPageSize),
			"ORD-099": fillPage(orderListPageSize, 50),
		}

		var mu sync.Mutex
		var afterCursors []string
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/orders-pending" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			after := r.URL.Query().Get("after")
			mu.Lock()
			repeated := slices.Contains(afterCursors, after)
			afterCursors = append(afterCursors, after)
			page := pages[after]
			mu.Unlock()
			if repeated {
				// Nothing here fails a request, so a repeat means the cursor never advanced.
				t.Errorf("after %q requested again", after)
				http.Error(w, "repeated after", http.StatusBadRequest)
				return
			}
			writeOKXData(t, w, page)
		}))

		resp, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot,
			Type:      order.AnyType,
			Side:      order.AnySide,
		})
		require.NoError(t, err, "GetActiveOrders must not error when the pending order list spans pages")

		mu.Lock()
		cursors := make([]string, len(afterCursors))
		copy(cursors, afterCursors)
		mu.Unlock()
		assert.Equal(t, []string{"", "ORD-099"}, cursors, "the second page should be requested with the first page's last order ID")
		assert.Len(t, resp, orderListPageSize+50, "a paginated crawl should return every open order")
		ids := make(map[string]struct{}, len(resp))
		for x := range resp {
			ids[resp[x].OrderID] = struct{}{}
		}
		assert.Len(t, ids, orderListPageSize+50, "a paginated crawl should not duplicate orders")
		assert.Contains(t, ids, "ORD-149", "orders past the first page should be returned")
	})

	t.Run("reports execution values", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/orders-pending" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{{
				"instId": "BTC-USDT", "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": "live",
				"sz": "2", "accFillSz": "0.5", "avgPx": "41000", "px": "40000", "fee": "-0.01", "feeCcy": "USDT",
				"cTime": "1700000000000", "uTime": "1700000000000",
			}})
		}))
		active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
		})
		require.NoError(t, err, "GetActiveOrders must not error")
		require.Len(t, active, 1, "the pending order must be returned")
		assert.Equal(t, 41000.0, active[0].AverageExecutedPrice, "GetActiveOrders should read the average filled price")
		assert.Equal(t, 41000.0*0.5, active[0].Cost, "the cost should be the average filled price times the accumulated fill")
		assert.Equal(t, "USDT", active[0].CostAsset.String(), "the cost asset should be the pair quote")
		assert.Equal(t, 0.01, active[0].Fee, "a fee OKX charged should read as a positive cost, as the websocket stream reports it")
		assert.Equal(t, "USDT", active[0].FeeAsset.String(), "the fee asset should come from the response")
	})

	t.Run("reports a maker rebate as a negative fee", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/orders-pending" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{makerRebateOrderRow("partially_filled", "1")})
		}))
		active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide})
		require.NoError(t, err, "GetActiveOrders must not error")
		exp := order.FilteredOrders{{
			Price: 100, Amount: 2, AverageExecutedPrice: 100, ExecutedAmount: 1, RemainingAmount: 1, Cost: 100, CostAsset: currency.USDT,
			Fee: -0.5, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "ORD-1", ClientOrderID: "client-1",
			Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.Spot,
			Date: time.UnixMilli(1700000000000), LastUpdated: time.UnixMilli(1700000001000), Pair: mainPair,
		}}
		assert.Equal(t, exp, active, "GetActiveOrders should net the maker rebate into the fee")
	})

	t.Run("reads each page before checking its cursor", func(t *testing.T) {
		t.Parallel()
		for _, listing := range []struct {
			name, path, cursor string
			a                  asset.Item
		}{
			{"standard", "/trade/orders-pending", "after", asset.Spot},
			{"spread", "/sprd/orders-pending", "endId", asset.Spread},
		} {
			row := func(i int) map[string]string {
				r := map[string]string{
					"ordId": fmt.Sprintf("ORD-%03d", i), "ordType": orderLimit, "side": "buy", "state": "live", "sz": "1", "px": "1",
					"accFillSz": "0", "cTime": "1700000000000", "uTime": "1700000000000",
				}
				if listing.a == asset.Spread {
					r["sprdId"] = "BTC-USDT_BTC-USDT-SWAP"
				} else {
					r["instId"] = mainPair.String()
				}
				return r
			}
			for _, tc := range pagerCases(row) {
				t.Run(listing.name+" "+tc.name, func(t *testing.T) {
					t.Parallel()
					ex := newMockExchange(t, withoutAlgoOrders(t, pagerHandler(t, listing.path, listing.cursor, tc.pages)))
					active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: listing.a, Type: order.AnyType, Side: order.AnySide})
					require.NoError(t, err, "GetActiveOrders must not error")
					ids := make([]string, 0, len(active))
					for i := range active {
						ids = append(ids, active[i].OrderID)
					}
					assert.Equal(t, tc.exp, ids, "GetActiveOrders should return every order of every page once, in listing order")
				})
			}
		}
	})

	t.Run("lists open orders whatever their age", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			age  time.Duration
		}{
			{"a start a minute past three months", kline.ThreeMonth.Duration() + time.Minute},
			{"a start a minute inside three months", kline.ThreeMonth.Duration() - time.Minute},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				now := timeNow()
				start := now.Add(-tc.age)
				created := time.UnixMilli(start.Add(time.Minute).UnixMilli())
				ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/trade/orders-pending", r.URL.Path, "GetActiveOrders should only request the pending orders")
					writeOKXData(t, w, []map[string]string{{
						"instType": instTypeSpot, "instId": mainPair.String(), "ordId": "126", "ordType": orderLimit, "side": "buy", "state": "live",
						"sz": "1", "px": "1", "cTime": strconv.FormatInt(created.UnixMilli(), 10),
					}})
				}))
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, StartTime: start, EndTime: now})
				require.NoError(t, err, "GetActiveOrders must not refuse a start before the order history's three months")
				exp := order.FilteredOrders{{
					Price: 1, Amount: 1, RemainingAmount: 1, CostAsset: currency.USDT, Exchange: ex.Name, OrderID: "126", Type: order.Limit,
					Side: order.Buy, Status: order.Active, AssetType: asset.Spot, Date: created, Pair: mainPair,
				}}
				assert.Equal(t, exp, active, "GetActiveOrders should list an open order created within the window")
			})
		}
	})

	t.Run("order lists include the start time", func(t *testing.T) {
		t.Parallel()
		start := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
		row := func(id string, created time.Time) map[string]string {
			return map[string]string{"instId": mainPair.String(), "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "1", "cTime": strconv.FormatInt(created.UnixMilli(), 10)}
		}
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, _ *http.Request) {
			writeOKXData(t, w, []map[string]string{row("NEWER", start.Add(time.Minute)), row("AT-START", start), row("OLDER", start.Add(-time.Minute))})
		}))
		active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, StartTime: start})
		require.NoError(t, err, "GetActiveOrders must not error")
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, StartTime: start, Pairs: currency.Pairs{mainPair}})
		require.NoError(t, err, "GetOrderHistory must not error")
		for name, orders := range map[string]order.FilteredOrders{"GetActiveOrders": active, "GetOrderHistory": history} {
			ids := make([]string, 0, len(orders))
			for i := range orders {
				ids = append(ids, orders[i].OrderID)
			}
			assert.ElementsMatchf(t, []string{"NEWER", "AT-START"}, ids, "%s should return the orders created from StartTime on", name)
		}
	})

	t.Run("starts from the requested order ID and reads a repeated order once", func(t *testing.T) {
		t.Parallel()
		row := func(id string) map[string]string {
			return map[string]string{"instId": mainPair.String(), "ordId": id, "ordType": orderLimit, "side": "buy", "state": "live", "sz": "1", "px": "1", "cTime": "1795409000000"}
		}
		var mu sync.Mutex
		var cursors []string
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/orders-pending" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			after := r.URL.Query().Get("after")
			mu.Lock()
			cursors = append(cursors, after)
			mu.Unlock()
			if after == "" {
				writeOKXData(t, w, []map[string]string{row("102"), row("99"), row("99")})
				return
			}
			writeOKXData(t, w, []map[string]string{row("99"), row("99"), row("98")})
		}))
		active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, FromOrderID: "100"})
		require.NoError(t, err, "GetActiveOrders must not error")
		assert.Equal(t, []string{"99", "98"}, detailOrderIDs(active), "GetActiveOrders should list each order before the starting order ID once")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"100"}, cursors, "GetActiveOrders should start the after cursor at the requested order ID")
	})

	t.Run("bounds creation times to the millisecond", func(t *testing.T) {
		t.Parallel()
		created := func(ms int) time.Time { return time.UnixMilli(1795410000000 + int64(ms)) }
		row := func(ms int) map[string]string {
			return map[string]string{
				"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": strconv.Itoa(ms), "ordType": orderLimit, "side": "buy", "state": "live",
				"sz": "1", "px": "1", "cTime": strconv.FormatInt(created(ms).UnixMilli(), 10),
			}
		}
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-pending", "/sprd/orders-pending":
				writeOKXData(t, w, []map[string]string{row(800), row(200), row(100), row(0)})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		for _, a := range []asset.Item{asset.Spot, asset.Spread} {
			for _, tc := range []struct {
				name       string
				start, end time.Time
				exp        []string
			}{
				{name: "an end within a millisecond", end: created(200).Add(500 * time.Microsecond), exp: []string{"200", "100", "0"}},
				{name: "an end at an order's creation", end: created(200), exp: []string{"200", "100", "0"}},
				{name: "an end before every order", end: created(0).Add(-time.Millisecond), exp: []string{}},
				{name: "a start within a millisecond", start: created(100).Add(500 * time.Microsecond), exp: []string{"800", "200", "100"}},
				{name: "a start and an end", start: created(100).Add(500 * time.Microsecond), end: created(200).Add(500 * time.Microsecond), exp: []string{"200", "100"}},
			} {
				t.Run(a.String()+" "+tc.name, func(t *testing.T) {
					t.Parallel()
					active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Type: order.AnyType, Side: order.AnySide, StartTime: tc.start, EndTime: tc.end})
					require.NoError(t, err, "GetActiveOrders must not error")
					assert.Equal(t, tc.exp, detailOrderIDs(active), "GetActiveOrders should list the orders created within each bound")
				})
			}
		}
	})

	t.Run("narrows any order type by the requested time in force", func(t *testing.T) {
		t.Parallel()
		row := func(id, ordType string) map[string]string {
			return map[string]string{"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": id, "ordType": ordType, "side": "buy", "state": "live", "sz": "1", "px": "1", "cTime": "1700000000000"}
		}
		rows := []map[string]string{
			row("LIMIT-1", orderLimit), row("POST-1", orderPostOnly), row("FOK-1", orderFOK), row("IOC-1", orderIOC), row("OPFOK-1", orderOptionFOK),
			row("MMPPO-1", orderMarketMakerProtectionAndPostOnly), row("OPTIMAL-1", orderOptimalLimitIOC), row("RPI-1", orderRPI),
		}
		trigger := func(id, advanceOrdType string) map[string]string {
			return map[string]string{"instId": mainPair.String(), "algoId": id, "ordType": orderTrigger, "advanceOrdType": advanceOrdType, "side": "buy", "state": "live", "sz": "1", "ordPx": "1", "triggerPx": "2", "cTime": "1700000000000"}
		}
		triggers := []map[string]string{trigger("TRIGGER-1", ""), trigger("TRIGGER-IOC-1", orderIOC), trigger("TRIGGER-FOK-1", orderFOK)}
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-pending", "/sprd/orders-pending":
				ordTypes := r.URL.Query().Get("ordType")
				listed := make([]map[string]string, 0, len(rows))
				for _, o := range rows {
					if ordTypes == "" || slices.Contains(strings.Split(ordTypes, ","), o["ordType"]) {
						listed = append(listed, o)
					}
				}
				writeOKXData(t, w, listed)
			case "/trade/orders-algo-pending":
				assert.Equal(t, orderTrigger, r.URL.Query().Get("ordType"), "only trigger orders should be listed by a time in force")
				writeOKXData(t, w, triggers)
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		for _, tc := range []struct {
			name string
			a    asset.Item
			tif  order.TimeInForce
			exp  []string
			err  error
		}{
			{"post only", asset.Spot, order.PostOnly, []string{"POST-1", "MMPPO-1"}, nil},
			{"immediate or cancel", asset.Spot, order.ImmediateOrCancel, []string{"IOC-1", "OPTIMAL-1", "TRIGGER-IOC-1"}, nil},
			{"fill or kill", asset.Spot, order.FillOrKill, []string{"FOK-1", "OPFOK-1", "TRIGGER-FOK-1"}, nil},
			{"fill or kill and immediate or cancel at once", asset.Spot, order.FillOrKill | order.ImmediateOrCancel, []string{}, order.ErrUnsupportedTimeInForce},
			{"good till cancel", asset.Spot, order.GoodTillCancel, []string{"LIMIT-1", "RPI-1"}, nil},
			{"spread post only", asset.Spread, order.PostOnly, []string{"POST-1"}, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Type: order.AnyType, TimeInForce: tc.tif, Side: order.AnySide})
				require.ErrorIs(t, err, tc.err, "GetActiveOrders must refuse a time in force no pending order reads back with")
				assert.ElementsMatch(t, tc.exp, detailOrderIDs(active), "GetActiveOrders should list only the orders read back with the time in force")
			})
		}
	})

	t.Run("refuses a market listing by a time in force no pending market order reads back with", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}))
		for _, tif := range []order.TimeInForce{order.ImmediateOrCancel, order.FillOrKill, order.PostOnly, order.GoodTillCancel} {
			t.Run(tif.String(), func(t *testing.T) {
				t.Parallel()
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Market, TimeInForce: tif, Side: order.AnySide})
				assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "GetActiveOrders should refuse the market listing before any request, as CancelAllOrders refuses one")
				assert.Empty(t, active, "GetActiveOrders should list nothing for the market listing")
			})
		}
	})

	t.Run("lists algo orders beside ordinary orders", func(t *testing.T) {
		t.Parallel()
		ordinary := map[string]string{"instType": instTypeSpot, "instId": mainPair.String(), "ordId": "80", "ordType": orderLimit, "side": "buy", "state": "live", "sz": "1", "px": "1", "cTime": "1795409000000"}
		trigger := map[string]string{
			"instType": instTypeSpot, "instId": mainPair.String(), "algoId": "81", "algoClOrdId": "client81", "ordType": orderTrigger, "side": "buy", "sz": "1",
			"state": "live", "triggerPx": "50000", "ordPx": "49000", "cTime": "1795409100000", "uTime": "1795409101000",
		}
		ordinaryDetail := order.Detail{
			Price: 1, Amount: 1, RemainingAmount: 1, CostAsset: currency.USDT, Exchange: "Okx", OrderID: "80", Type: order.Limit, Side: order.Buy,
			Status: order.Active, AssetType: asset.Spot, Date: time.UnixMilli(1795409000000), Pair: mainPair,
		}
		triggerDetail := order.Detail{
			Amount: 1, Exchange: "Okx", OrderID: "81", ClientOrderID: "client81", Side: order.Buy, Type: order.Trigger, Pair: mainPair, AssetType: asset.Spot,
			Status: order.Active, Price: 49000, TriggerPrice: 50000, Date: time.UnixMilli(1795409100000), LastUpdated: time.UnixMilli(1795409101000),
		}
		algoRequests := func(instType string, ordTypes ...string) []string {
			r := make([]string, 0, len(ordTypes))
			for _, o := range ordTypes {
				r = append(r, "/trade/orders-algo-pending "+instType+" "+o)
			}
			return r
		}
		for _, tc := range []struct {
			name      string
			a         asset.Item
			orderType order.Type
			requests  []string
			exp       order.FilteredOrders
		}{
			{
				name: "any order type", a: asset.Spot, orderType: order.AnyType,
				requests: append(algoRequests(instTypeSpot, "trigger", "conditional,oco", "move_order_stop", "twap"), "/trade/orders-pending SPOT "),
				exp:      order.FilteredOrders{triggerDetail, ordinaryDetail},
			},
			{name: "trigger orders", a: asset.Spot, orderType: order.Trigger, requests: algoRequests(instTypeSpot, "trigger"), exp: order.FilteredOrders{triggerDetail}},
			{
				name: "limit orders", a: asset.Spot, orderType: order.Limit, requests: []string{"/trade/orders-pending SPOT limit,post_only,fok,ioc,op_fok,rpi"},
				exp: order.FilteredOrders{ordinaryDetail},
			},
			{
				name: "any order type of perpetual swap", a: asset.PerpetualSwap, orderType: order.AnyType,
				requests: append(algoRequests(instTypeSwap, "trigger", "conditional,oco", "move_order_stop", "twap", "chase"), "/trade/orders-pending SWAP "),
				exp:      order.FilteredOrders{},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var requests []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					q := r.URL.Query()
					mu.Lock()
					requests = append(requests, r.URL.Path+" "+q.Get("instType")+" "+q.Get("ordType"))
					mu.Unlock()
					switch {
					case q.Get("instType") != instTypeSpot:
						writeOKXData(t, w, []any{})
					case r.URL.Path == "/trade/orders-pending":
						writeOKXData(t, w, []map[string]string{ordinary})
					case r.URL.Path == "/trade/orders-algo-pending" && q.Get("ordType") == orderTrigger:
						writeOKXData(t, w, []map[string]string{trigger})
					case r.URL.Path == "/trade/orders-algo-pending":
						writeOKXData(t, w, []any{})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Type: tc.orderType, Side: order.AnySide})
				require.NoError(t, err, "GetActiveOrders must not error")
				assert.Equal(t, tc.exp, active, "GetActiveOrders should list the algo orders and ordinary orders of the requested type")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.requests, requests, "GetActiveOrders should read the listings of the requested order type")
			})
		}
		for _, tc := range []struct {
			name      string
			a         asset.Item
			orderType order.Type
			tif       order.TimeInForce
			err       error
		}{
			{name: "chase orders on spot", a: asset.Spot, orderType: order.Chase, err: order.ErrUnsupportedOrderType},
			{name: "trigger orders on options", a: asset.Options, orderType: order.Trigger, err: order.ErrUnsupportedOrderType},
			{name: "trigger orders with a time in force none reads back with", a: asset.Spot, orderType: order.Trigger, tif: order.PostOnly, err: order.ErrUnsupportedTimeInForce},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide})
				assert.ErrorIs(t, err, tc.err, "GetActiveOrders should refuse an algo order listing OKX cannot serve")
			})
		}
	})

	t.Run("lists trigger orders by the time in force of the orders they place", func(t *testing.T) {
		t.Parallel()
		trigger := func(id, advanceOrdType string) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "algoId": id, "ordType": orderTrigger, "advanceOrdType": advanceOrdType, "side": "buy",
				"sz": "1", "state": "live", "triggerPx": "110", "ordPx": "100", "cTime": "1795409100000", "uTime": "1795409101000",
			}
		}
		ordinary := map[string]string{
			"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "ordId": "94", "ordType": orderIOC, "side": "sell", "state": "live", "sz": "2", "px": "90",
			"cTime": "1795409000000",
		}
		triggerDetail := func(id string, tif order.TimeInForce) order.Detail {
			return order.Detail{
				Amount: 1, Exchange: "Okx", OrderID: id, Side: order.Buy, Type: order.Trigger, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap,
				Status: order.Active, Price: 100, TriggerPrice: 110, Date: time.UnixMilli(1795409100000), LastUpdated: time.UnixMilli(1795409101000), TimeInForce: tif,
			}
		}
		ordinaryDetail := order.Detail{
			Amount: 2, Price: 90, RemainingAmount: 2, Exchange: "Okx", OrderID: "94", Type: order.Limit, Side: order.Sell, Status: order.Active,
			AssetType: asset.PerpetualSwap, Date: time.UnixMilli(1795409000000), Pair: perpetualSwapPair, TimeInForce: order.ImmediateOrCancel,
		}
		for _, tc := range []struct {
			name      string
			orderType order.Type
			tif       order.TimeInForce
			requests  []string
			exp       order.FilteredOrders
		}{
			{
				name: "immediate or cancel trigger orders", orderType: order.Trigger, tif: order.ImmediateOrCancel,
				requests: []string{"/trade/orders-algo-pending SWAP trigger"}, exp: order.FilteredOrders{triggerDetail("91", order.ImmediateOrCancel)},
			},
			{
				name: "fill or kill trigger orders", orderType: order.Trigger, tif: order.FillOrKill,
				requests: []string{"/trade/orders-algo-pending SWAP trigger"}, exp: order.FilteredOrders{triggerDetail("92", order.FillOrKill)},
			},
			{
				name: "immediate or cancel orders of any type", orderType: order.AnyType, tif: order.ImmediateOrCancel,
				requests: []string{"/trade/orders-algo-pending SWAP trigger", "/trade/orders-pending SWAP ioc,optimal_limit_ioc"},
				exp:      order.FilteredOrders{triggerDetail("91", order.ImmediateOrCancel), ordinaryDetail},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var requests []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					q := r.URL.Query()
					mu.Lock()
					requests = append(requests, r.URL.Path+" "+q.Get("instType")+" "+q.Get("ordType"))
					mu.Unlock()
					switch r.URL.Path {
					case "/trade/orders-algo-pending":
						writeOKXData(t, w, []map[string]string{trigger("91", orderIOC), trigger("92", orderFOK), trigger("93", "")})
					case "/trade/orders-pending":
						writeOKXData(t, w, []map[string]string{ordinary})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide})
				require.NoError(t, err, "GetActiveOrders must not error")
				assert.Equal(t, tc.exp, active, "GetActiveOrders should list the trigger orders read back with the requested time in force")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.requests, requests, "GetActiveOrders should read the trigger order listing for the requested time in force")
			})
		}
	})

	t.Run("lists the ordinary orders beside a trigger listing whose cursors cycle", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			tif  order.TimeInForce
			exp  []string
		}{
			{name: "immediate or cancel", tif: order.ImmediateOrCancel, exp: []string{"200", "800"}},
			{name: "fill or kill", tif: order.FillOrKill, exp: []string{"199", "801"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				triggers := pagerHandler(t, "/trade/orders-algo-pending", "after", cyclingTriggerPages("live"))
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/orders-pending" {
						triggers(w, r)
						return
					}
					writeOKXData(t, w, cyclingListingOrdinaryOrders("live", r.URL.Query().Get("ordType")))
				}))
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: order.AnyType, TimeInForce: tc.tif, Side: order.AnySide})
				require.ErrorIs(t, err, errListingStalled, "GetActiveOrders must report the trigger listing which stops advancing")
				assert.ErrorContains(t, err, "listing trigger algo orders", "the error should name the listing which stops advancing")
				assert.Equal(t, tc.exp, detailOrderIDs(active), "GetActiveOrders should return the trigger orders read and the ordinary listing's orders beside the error")
			})
		}
	})

	t.Run("reports each order as OKX states it", func(t *testing.T) {
		t.Parallel()
		for _, tc := range statedOrderCases("partially_filled", order.PartiallyFilled) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/trade/orders-pending", r.URL.Path, "GetActiveOrders should request the pending orders")
					writeOKXData(t, w, []map[string]string{tc.row})
				}))
				active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Type: order.AnyType, Side: order.AnySide})
				require.NoError(t, err, "GetActiveOrders must not error")
				exp := tc.exp
				exp.Exchange = ex.Name
				assert.Equal(t, order.FilteredOrders{exp}, active, "GetActiveOrders should report the order's sizes, trade currency and reduce only flag as OKX states them")
			})
		}
	})

	t.Run("reports the spread size still to fill", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/sprd/orders-pending", r.URL.Path, "GetActiveOrders should request the pending spread orders")
			writeOKXData(t, w, []map[string]string{
				{
					"sprdId": spreadPair.String(), "ordId": "312269865356374016", "clOrdId": "b1", "px": "999", "sz": "3", "ordType": orderLimit, "side": "buy",
					"accFillSz": "0", "pendingFillSz": "2", "pendingSettleSz": "1", "canceledSz": "1", "state": "live", "uTime": "1597026383085", "cTime": "1597026383085",
				},
				{
					"sprdId": spreadPair.String(), "ordId": "312269865356374015", "clOrdId": "b2", "px": "998", "sz": "3", "ordType": orderLimit, "side": "buy",
					"accFillSz": "1", "pendingFillSz": "0", "canceledSz": "2", "state": "partially_filled", "uTime": "1597026383084", "cTime": "1597026383084",
				},
			})
		}))
		active, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spread, Type: order.AnyType, Side: order.AnySide})
		require.NoError(t, err, "GetActiveOrders must not error")
		exp := order.FilteredOrders{
			{
				Amount: 3, Pair: spreadPair, Price: 999, RemainingAmount: 2, Exchange: ex.Name, OrderID: "312269865356374016", ClientOrderID: "b1",
				Type: order.Limit, Side: order.Buy, Status: order.Active, AssetType: asset.Spread, Date: time.UnixMilli(1597026383085), LastUpdated: time.UnixMilli(1597026383085),
			},
			{
				Amount: 3, Pair: spreadPair, Price: 998, ExecutedAmount: 1, Exchange: ex.Name, OrderID: "312269865356374015", ClientOrderID: "b2",
				Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.Spread, Date: time.UnixMilli(1597026383084), LastUpdated: time.UnixMilli(1597026383084),
			},
		}
		assert.Equal(t, exp, active, "GetActiveOrders should report the size OKX states is still to fill")
	})
}

// detailOrderIDs returns the order IDs of order details, in order
func detailOrderIDs(details []order.Detail) []string {
	ids := make([]string, 0, len(details))
	for i := range details {
		ids = append(ids, details[i].OrderID)
	}
	return ids
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		start := time.UnixMilli(1795410000000).Add(-kline.SevenDay.Duration())
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			start = time.Now().Add(-kline.SevenDay.Duration())
		}

		getOrdersRequest := order.MultiOrderRequest{
			Type:      order.AnyType,
			AssetType: asset.Spot,
			Side:      order.Buy,
		}
		// The history endpoints require only the instrument type, so empty pairs
		// queries every instrument within the window.
		getOrdersRequest.StartTime = start
		result, err := e.GetOrderHistory(contextGenerate(), &getOrdersRequest)
		require.NoError(t, err, "GetOrderHistory must accept empty pairs")
		// The archive ends at an order older than the window, holds a sell the request leaves out and repeats an order
		// the 7 day listing holds
		ltcBTCOrder := order.Detail{
			Price: 0.00118, AverageExecutedPrice: 0.00118, Amount: 1.5, ExecutedAmount: 1.5, Fee: 0.0015, FeeAsset: currency.LTC,
			Exchange: e.Name, OrderID: "680800019749904391", ClientOrderID: "ltcbtc1", Type: order.Limit, Side: order.Buy,
			Status: order.Filled, AssetType: asset.Spot, Date: time.UnixMilli(1795300000000), LastUpdated: time.UnixMilli(1795300060000),
			Pair: currency.NewPairWithDelimiter("LTC", "BTC", currency.DashDelimiter), Cost: 0.00177, CostAsset: currency.BTC,
		}
		if mockTests {
			exp := order.FilteredOrders{
				{
					QuoteAmount: 10, Exchange: e.Name, OrderID: "1753184812254216192", Side: order.Buy, Type: order.ConditionalStop, Pair: mainPair,
					AssetType: asset.Spot, Status: order.Closed, Price: -1, TriggerPrice: 10000, Date: time.UnixMilli(1795409500000),
					LastUpdated: time.UnixMilli(1795409600000),
				},
				{
					AverageExecutedPrice: 61728.4, Amount: 0.00162, ExecutedAmount: 0.00162, QuoteAmount: 100, Fee: 0.00000162,
					FeeAsset: currency.BTC, Exchange: e.Name, OrderID: "680800019749904390", Type: order.Market, Side: order.Buy,
					Status: order.Filled, AssetType: asset.Spot, Date: time.UnixMilli(1795409400000), LastUpdated: time.UnixMilli(1795409400001),
					Pair: mainPair, Cost: 100.000008, CostAsset: currency.USDT,
				},
				ltcBTCOrder,
				{
					Price: 61700, Amount: 0.002, RemainingAmount: 0.002, FeeAsset: currency.BTC, Exchange: e.Name, OrderID: "680800019749904395",
					ClientOrderID: "ioc1", Type: order.Limit, Side: order.Buy, Status: order.Cancelled, AssetType: asset.Spot,
					Date: time.UnixMilli(1795409900000), LastUpdated: time.UnixMilli(1795409900002), Pair: mainPair, CostAsset: currency.USDT,
					TimeInForce: order.ImmediateOrCancel,
				},
			}
			assert.Equal(t, exp, result, "GetOrderHistory should map the requested orders of both listings once each")
		} else {
			assert.NotNil(t, result, "GetOrderHistory should return orders")
		}

		getOrdersRequest.Pairs = []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)}
		result, err = e.GetOrderHistory(contextGenerate(), &getOrdersRequest)
		require.NoError(t, err, "GetOrderHistory must not error for a pair")
		if mockTests {
			assert.Equal(t, order.FilteredOrders{ltcBTCOrder}, result, "GetOrderHistory should return only the requested pair's orders")
		} else {
			assert.NotNil(t, result, "GetOrderHistory should return the pair's orders")
		}

		getOrdersRequest.AssetType = asset.Spread
		getOrdersRequest.Type = order.Market
		getOrdersRequest.Pairs = []currency.Pair{spreadPair}
		result, err = e.GetOrderHistory(contextGenerate(), &getOrdersRequest)
		require.NoError(t, err, "GetOrderHistory must not error for spread orders")
		if mockTests {
			exp := order.FilteredOrders{{
				AverageExecutedPrice: -30.8, Amount: 0.5, ExecutedAmount: 0.5, Exchange: e.Name, OrderID: "312269865356374020",
				ClientOrderID: "m1", Type: order.Market, Side: order.Buy, Status: order.Filled, AssetType: asset.Spread,
				Date: time.UnixMilli(1795400000000), LastUpdated: time.UnixMilli(1795400000150), Pair: spreadPair,
			}}
			assert.Equal(t, exp, result, "GetOrderHistory should map the spread market orders")
		} else {
			assert.NotNil(t, result, "GetOrderHistory should return spread orders")
		}
	})

	t.Run("judges the three month limit by the wrapper's clock", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			age  time.Duration
			err  error
		}{
			{"a start a minute past three months", kline.ThreeMonth.Duration() + time.Minute, errOnlyThreeMonthsSupported},
			{"a start a minute inside three months", kline.ThreeMonth.Duration() - time.Minute, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/orders-history" && r.URL.Path != "/trade/orders-history-archive" {
						t.Errorf("unexpected request path %s", r.URL.Path)
					}
					writeOKXData(t, w, []any{})
				}))
				now := timeNow()
				_, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, StartTime: now.Add(-tc.age), EndTime: now})
				assert.ErrorIs(t, err, tc.err, "GetOrderHistory should judge the start by the wrapper's clock")
			})
		}
	})

	t.Run("paginates with the after cursor", func(t *testing.T) {
		t.Parallel()
		base := time.Now().Add(-24 * time.Hour).Truncate(time.Millisecond)
		endTime := base.Add(128 * time.Millisecond)
		// ORD-099 and ORD-100 share one creation millisecond across the page
		// boundary: the after cursor must page past them without overlap.
		cTime := func(i int) time.Time {
			if i == 100 {
				return base.Add(99 * time.Millisecond)
			}
			return base.Add(time.Duration(i) * time.Millisecond)
		}
		row := func(i int) map[string]string {
			created := strconv.FormatInt(cTime(i).UnixMilli(), 10)
			return map[string]string{
				"instId": mainPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i),
				"cTime": created, "uTime": created, "state": "filled",
				"ordType": orderLimit, "side": "buy", "sz": "1",
				"px": "42000", "accFillSz": "1", "avgPx": "42000",
				"fee": "-0.01", "feeCcy": "USDT",
				// The rebate currency is unrelated to the pair, so the cost asset
				// must come from the pair quote, not this field.
				"rebateCcy": "DOGE",
			}
		}
		// The mock pages by the after order ID cursor: the first page holds the
		// 100 newest orders, and the second page holds the 30 older ones.
		page1 := make([]map[string]string, 0, orderListPageSize)
		for i := 129; i >= 30; i-- {
			page1 = append(page1, row(i))
		}
		page2 := make([]map[string]string, 0, 30)
		for i := 29; i >= 0; i-- {
			page2 = append(page2, row(i))
		}
		pages := map[string][]map[string]string{
			"":        page1,
			"ORD-030": page2,
		}
		var mu sync.Mutex
		var afterCursors []string
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/trade/orders-history" {
				// The window reaches the 7 day listing, which has nothing to add
				// to the archive's rows here.
				writeOKXData(t, w, []map[string]string{})
				return
			}
			if r.URL.Path != "/trade/orders-history-archive" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			after := r.URL.Query().Get("after")
			mu.Lock()
			repeated := slices.Contains(afterCursors, after)
			afterCursors = append(afterCursors, after)
			page := pages[after]
			mu.Unlock()
			if repeated {
				// A repeat means the after cursor never advanced.
				t.Errorf("after %q requested again", after)
				http.Error(w, "repeated after", http.StatusBadRequest)
				return
			}
			writeOKXData(t, w, page)
		}))
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
			StartTime: base.Add(-time.Second), EndTime: endTime,
			Pairs: currency.Pairs{mainPair},
		})
		require.NoError(t, err, "GetOrderHistory must not error when the archive spans pages")
		mu.Lock()
		cursors := slices.Clone(afterCursors)
		mu.Unlock()
		assert.Equal(t, []string{"", "ORD-030"}, cursors, "the second page should be requested with the first page's last order ID as the after cursor")
		require.Len(t, history, 130, "a paginated crawl must return every order")
		ids := make(map[string]struct{}, len(history))
		for x := range history {
			ids[history[x].OrderID] = struct{}{}
		}
		assert.Len(t, ids, len(history), "a paginated crawl should not duplicate orders")
		assert.Contains(t, ids, "ORD-100", "orders sharing a millisecond across the page boundary should all be returned")
		assert.Contains(t, ids, "ORD-000", "orders past the first page should be returned")
		assert.Equal(t, mainPair.Quote, history[0].CostAsset, "the cost asset should be the pair quote, not the order's rebate currency")
		assert.Equal(t, 0.01, history[0].Fee, "a fee OKX charged should read as a positive cost, as the websocket stream reports it")
		assert.Equal(t, "USDT", history[0].FeeAsset.String(), "the fee asset should come from the response")
	})

	t.Run("reports a maker rebate as a negative fee", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/orders-history-archive" && r.URL.Path != "/trade/orders-history" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{makerRebateOrderRow("filled", "2")})
		}))
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide})
		require.NoError(t, err, "GetOrderHistory must not error")
		exp := order.FilteredOrders{{
			Price: 100, Amount: 2, AverageExecutedPrice: 100, ExecutedAmount: 2, Cost: 200, CostAsset: currency.USDT,
			Fee: -0.5, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "ORD-1", ClientOrderID: "client-1",
			Type: order.Limit, Side: order.Buy, Status: order.Filled, AssetType: asset.Spot,
			Date: time.UnixMilli(1700000000000), LastUpdated: time.UnixMilli(1700000001000), Pair: mainPair,
		}}
		assert.Equal(t, exp, history, "GetOrderHistory should net the maker rebate into the fee")
	})

	t.Run("without pairs returns every instrument", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-24*time.Hour).Truncate(time.Millisecond).UnixMilli(), 10)
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/trade/orders-history" {
				writeOKXData(t, w, []map[string]string{})
				return
			}
			if r.URL.Path != "/trade/orders-history-archive" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{
				{
					"instId": mainPair.String(), "ordId": "ORD-A",
					"cTime": created, "uTime": created, "state": "filled",
					"ordType": orderLimit, "side": "buy", "sz": "1",
					"px": "42000", "accFillSz": "1", "avgPx": "42000",
				},
				{
					"instId": "ETH-USDT", "ordId": "ORD-B",
					"cTime": created, "uTime": created, "state": "filled",
					"ordType": orderLimit, "side": "sell", "sz": "2",
					"px": "3000", "accFillSz": "2", "avgPx": "3000",
				},
			})
		}))
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
			StartTime: time.Now().Add(-25 * time.Hour), EndTime: time.Now().Add(-23 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error when no pairs are set")
		require.Len(t, history, 2, "a request without pairs must return every instrument's orders")
		ids := make(map[string]struct{}, len(history))
		for x := range history {
			ids[history[x].OrderID] = struct{}{}
		}
		assert.Contains(t, ids, "ORD-A", "the first instrument's orders should be returned without pairs")
		assert.Contains(t, ids, "ORD-B", "the second instrument's orders should be returned without pairs")
	})

	t.Run("stops when a full page is seen", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-24*time.Hour).Truncate(time.Millisecond).UnixMilli(), 10)
		page := make([]map[string]string, 0, orderListPageSize)
		for i := range orderListPageSize {
			// Every order shares the one creation millisecond, and the mock keeps
			// returning the page the after cursor already passed.
			page = append(page, map[string]string{
				"instId": mainPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i),
				"cTime": created, "uTime": created, "state": "filled",
				"ordType": orderLimit, "side": "buy", "sz": "1",
				"px": "42000", "accFillSz": "1", "avgPx": "42000",
			})
		}
		var mu sync.Mutex
		var afterCursors []string
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/trade/orders-history" {
				// The window reaches the 7 day listing, which has nothing to add
				// to the archive's rows here.
				writeOKXData(t, w, []map[string]string{})
				return
			}
			if r.URL.Path != "/trade/orders-history-archive" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			after := r.URL.Query().Get("after")
			mu.Lock()
			repeated := slices.Contains(afterCursors, after)
			afterCursors = append(afterCursors, after)
			mu.Unlock()
			if repeated {
				// A repeat means the after cursor never advanced.
				t.Errorf("after %q requested again", after)
				http.Error(w, "repeated after", http.StatusBadRequest)
				return
			}
			writeOKXData(t, w, page)
		}))
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
			Pairs: currency.Pairs{mainPair},
		})
		require.NoError(t, err, "GetOrderHistory must stop when a full page holds no new order")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"", "ORD-099"}, afterCursors, "the crawl should stop when the after cursor stops advancing")
		assert.Len(t, history, orderListPageSize, "the first page's orders should be returned once")
	})

	t.Run("from order ID seeds the after cursor", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/trade/orders-history" {
				// The 7 day listing is crawled with the same seed and adds
				// nothing here.
				writeOKXData(t, w, []map[string]string{})
				return
			}
			if r.URL.Path != "/trade/orders-history-archive" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			assert.Equal(t, "SEED-1", r.URL.Query().Get("after"), "the first request should carry FromOrderID as the after cursor")
			writeOKXData(t, w, []map[string]string{})
		}))
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
			FromOrderID: "SEED-1",
			Pairs:       currency.Pairs{mainPair},
		})
		require.NoError(t, err, "GetOrderHistory must not error when FromOrderID seeds the crawl")
		assert.Empty(t, history, "an empty archive page should return no orders")
	})

	t.Run("pages the requested time window", func(t *testing.T) {
		t.Parallel()
		base := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
		created := func(id int64) time.Time { return base.Add(time.Duration(id) * time.Second) }
		// The archive holds orders 1 to 150 and the 7 day listing 151 to 300.
		listings := map[string][2]int64{"/trade/orders-history-archive": {1, 150}, "/trade/orders-history": {151, 300}}
		// venue serves one subtest, recording the cursors it has served in
		// requested.
		venue := func(t *testing.T, requested *sync.Map, w http.ResponseWriter, r *http.Request) {
			t.Helper()
			ids, ok := listings[r.URL.Path]
			if !ok {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			q := r.URL.Query()
			// This venue never fails a well-formed request, so a repeated cursor
			// means the crawl stopped advancing: failing it ends the crawl instead
			// of serving the same page until go test's timeout.
			if _, repeated := requested.LoadOrStore(r.URL.Path+"?after="+q.Get("after"), struct{}{}); repeated {
				t.Errorf("%s after %q requested again", r.URL.Path, q.Get("after"))
				http.Error(w, "repeated after", http.StatusBadRequest)
				return
			}
			param := func(k string) (int64, bool) {
				v, err := strconv.ParseInt(q.Get(k), 10, 64)
				return v, err == nil
			}
			begin, hasBegin := param("begin")
			end, hasEnd := param("end")
			after, hasAfter := param("after")
			var page []int64
			for id := ids[0]; id <= ids[1]; id++ {
				c := created(id).UnixMilli()
				if (hasBegin && c < begin) || (hasEnd && c > end) || (hasAfter && id >= after) {
					continue
				}
				page = append(page, id)
			}
			if len(page) > orderListPageSize {
				if hasBegin && !hasEnd && !hasAfter {
					page = page[:orderListPageSize]
				} else {
					page = page[len(page)-orderListPageSize:]
				}
			}
			rows := make([]map[string]string, 0, len(page))
			for _, id := range slices.Backward(page) {
				c := strconv.FormatInt(created(id).UnixMilli(), 10)
				rows = append(rows, map[string]string{
					"instId": mainPair.String(), "ordId": strconv.FormatInt(id, 10), "ordType": orderLimit, "side": "buy",
					"state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": c, "uTime": c,
				})
			}
			writeOKXData(t, w, rows)
		}
		for _, tc := range []struct {
			name       string
			start, end time.Time
			first      int64
			last       int64
		}{
			{"start only", created(31), time.Time{}, 31, 300},
			{"end only", time.Time{}, created(120), 1, 120},
			{"start and end", created(31), created(200), 31, 200},
			{"neither", time.Time{}, time.Time{}, 1, 300},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				requested := new(sync.Map)
				ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
					venue(t, requested, w, r)
				}))
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
					AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
					StartTime: tc.start, EndTime: tc.end, Pairs: currency.Pairs{mainPair},
				})
				require.NoError(t, err, "GetOrderHistory must not error")
				got := make([]int64, 0, len(history))
				for i := range history {
					id, err := strconv.ParseInt(history[i].OrderID, 10, 64)
					require.NoError(t, err, "order IDs must parse")
					got = append(got, id)
				}
				slices.Sort(got)
				exp := make([]int64, 0, tc.last-tc.first+1)
				for id := tc.first; id <= tc.last; id++ {
					exp = append(exp, id)
				}
				assert.Equal(t, exp, got, "GetOrderHistory should return exactly the orders in the window")
			})
		}
	})

	t.Run("rejects unmapped states", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Hour).UnixMilli(), 10)
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/orders-history-archive" && r.URL.Path != "/trade/orders-history" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{{
				"instId": mainPair.String(), "ordId": "ORD-1", "ordType": orderLimit, "side": "buy",
				"state": "bogus", "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": created,
			}})
		}))
		_, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{mainPair},
		})
		require.ErrorContains(t, err, "unrecognised order status", "an unmapped order state must fail the request")
	})

	t.Run("narrows by the requested time in force", func(t *testing.T) {
		t.Parallel()
		row := func(id, ordType string) map[string]string {
			return map[string]string{"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": id, "ordType": ordType, "side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": "1700000000000"}
		}
		rows := []map[string]string{
			row("LIMIT-1", orderLimit), row("POST-1", orderPostOnly), row("IOC-1", orderIOC), row("RPI-1", orderRPI), row("FOK-1", orderFOK),
			row("OPFOK-1", orderOptionFOK), row("MMP-1", orderMarketMakerProtection), row("MMPPO-1", orderMarketMakerProtectionAndPostOnly),
			// OKX sends ordType in lower case; the read-back accepts any case.
			row("POST-2", "POST_ONLY"),
		}
		trigger := func(id, advanceOrdType string) map[string]string {
			return map[string]string{"instId": mainPair.String(), "algoId": id, "ordType": orderTrigger, "advanceOrdType": advanceOrdType, "side": "buy", "state": "effective", "sz": "1", "ordPx": "1", "triggerPx": "2", "cTime": "1700000000000"}
		}
		triggers := []map[string]string{trigger("TRIGGER-1", ""), trigger("TRIGGER-IOC-1", orderIOC), trigger("TRIGGER-FOK-1", orderFOK)}
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-history-archive", "/trade/orders-history":
				writeOKXData(t, w, rows)
			case "/trade/orders-algo-history":
				assert.Equal(t, orderTrigger, r.URL.Query().Get("ordType"), "only trigger orders should be listed by a time in force")
				writeOKXData(t, w, triggers)
			case "/sprd/orders-history", "/sprd/orders-history-archive":
				// The spread listings filter on the single ordType they are sent.
				filtered := make([]map[string]string, 0, len(rows))
				for _, o := range rows {
					if ordType := r.URL.Query().Get("ordType"); ordType == "" || o["ordType"] == ordType {
						filtered = append(filtered, o)
					}
				}
				writeOKXData(t, w, filtered)
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			orderType order.Type
			tif       order.TimeInForce
			exp       []string
			err       error
		}{
			{"post only query returns post only orders", asset.Spot, order.Limit, order.PostOnly, []string{"POST-1", "POST-2"}, nil},
			{"good till cancel post only query returns post only orders", asset.Spot, order.Limit, order.GoodTillCancel | order.PostOnly, []string{"POST-1", "POST-2"}, nil},
			{"good till cancel post only spread query returns post only orders", asset.Spread, order.Limit, order.GoodTillCancel | order.PostOnly, []string{"POST-1"}, nil},
			{"immediate or cancel query returns ioc orders", asset.Spot, order.Limit, order.ImmediateOrCancel, []string{"IOC-1"}, nil},
			{"no time in force returns every limit order", asset.Spot, order.Limit, order.UnknownTIF, []string{"LIMIT-1", "POST-1", "IOC-1", "RPI-1", "FOK-1", "OPFOK-1", "POST-2"}, nil},
			{"fill or kill query returns both fill or kill types", asset.Spot, order.Limit, order.FillOrKill, []string{"FOK-1", "OPFOK-1"}, nil},
			{"good till cancel query returns resting limit orders", asset.Spot, order.Limit, order.GoodTillCancel, []string{"LIMIT-1", "RPI-1"}, nil},
			{"good till day query is refused", asset.Spot, order.Limit, order.GoodTillDay, []string{}, order.ErrUnsupportedTimeInForce},
			{"market immediate or cancel query is refused", asset.Spot, order.Market, order.ImmediateOrCancel, []string{}, order.ErrUnsupportedTimeInForce},
			{"market fill or kill query is refused", asset.Spot, order.Market, order.FillOrKill, []string{}, order.ErrUnsupportedTimeInForce},
			{"market post only query is refused", asset.Spot, order.Market, order.PostOnly, []string{}, order.ErrUnsupportedTimeInForce},
			{"good till cancel limit maker query returns post only orders", asset.Spot, order.LimitMaker, order.GoodTillCancel, []string{"POST-1", "POST-2"}, nil},
			{"post only market maker protection query returns its post only orders", asset.Spot, order.MarketMakerProtection, order.PostOnly, []string{"MMPPO-1"}, nil},
			{"any type post only query returns every post only order", asset.Spot, order.AnyType, order.PostOnly, []string{"POST-1", "MMPPO-1", "POST-2"}, nil},
			{"any type fill or kill query returns every fill or kill order", asset.Spot, order.AnyType, order.FillOrKill, []string{"FOK-1", "OPFOK-1", "TRIGGER-FOK-1"}, nil},
			{"any type immediate or cancel query returns ioc orders", asset.Spot, order.AnyType, order.ImmediateOrCancel, []string{"IOC-1", "TRIGGER-IOC-1"}, nil},
			{"any type good till cancel query returns resting orders", asset.Spot, order.AnyType, order.GoodTillCancel, []string{"LIMIT-1", "RPI-1", "MMP-1"}, nil},
			{"any type post only spread query returns post only orders", asset.Spread, order.AnyType, order.PostOnly, []string{"POST-1"}, nil},
			{"any type good till cancel spread query returns the orders it places", asset.Spread, order.AnyType, order.GoodTillCancel, []string{"LIMIT-1"}, nil},
			{"good till cancel spread query returns the orders it places", asset.Spread, order.Limit, order.GoodTillCancel, []string{"LIMIT-1"}, nil},
			{"spread query without a time in force returns every limit order", asset.Spread, order.Limit, order.UnknownTIF, []string{"LIMIT-1", "POST-1", "IOC-1", "RPI-1", "FOK-1", "OPFOK-1", "POST-2"}, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
					AssetType: tc.assetType, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide,
				})
				require.ErrorIs(t, err, tc.err, "GetOrderHistory must refuse a type and time in force no order reads back as")
				ids := make([]string, 0, len(history))
				for i := range history {
					ids = append(ids, history[i].OrderID)
				}
				assert.ElementsMatch(t, tc.exp, ids, "the history should return exactly the orders carrying the requested time in force")
			})
		}
		_, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.LimitMaker, TimeInForce: order.FillOrKill, Side: order.AnySide,
		})
		assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "a time in force the order type cannot take should be refused, as GetActiveOrders refuses it")
	})

	t.Run("keeps the start time millisecond", func(t *testing.T) {
		t.Parallel()
		created := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
		ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-history-archive", "/trade/orders-history":
				writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "1", "px": "1", "cTime": strconv.FormatInt(created.UnixMilli(), 10)}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, StartTime: created.Add(500 * time.Microsecond),
		})
		require.NoError(t, err, "GetOrderHistory must not error")
		require.Len(t, history, 1, "GetOrderHistory must keep an order created in the StartTime millisecond")
		assert.Equal(t, "ORD-1", history[0].OrderID, "the order created in the StartTime millisecond should be returned")
	})

	t.Run("includes zero-fill canceled from the seven-day listing", func(t *testing.T) {
		t.Parallel()

		start := time.Now().Add(-time.Hour).Truncate(time.Millisecond)
		filled := map[string]string{
			"instId": mainPair.String(), "ordId": "FILLED-1", "ordType": orderLimit, "side": "buy",
			"state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "42000", "px": "42000",
			"cTime": strconv.FormatInt(start.Add(time.Minute).UnixMilli(), 10),
		}
		zeroFillCanceled := map[string]string{
			"instId": mainPair.String(), "ordId": "CANCELED-1", "ordType": orderLimit, "side": "buy",
			"state": "canceled", "sz": "1", "accFillSz": "0", "avgPx": "0", "px": "42000",
			"cTime": strconv.FormatInt(start.Add(2*time.Minute).UnixMilli(), 10),
		}

		t.Run("the zero-fill canceled order comes from the 7 day listing", func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/trade/orders-history-archive":
					writeOKXData(t, w, []map[string]string{filled})
				case "/trade/orders-history":
					// Only the 7 day listing keeps canceled orders without fills.
					writeOKXData(t, w, []map[string]string{zeroFillCanceled, filled})
				default:
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
				StartTime: start, Pairs: currency.Pairs{mainPair},
			})
			require.NoError(t, err, "GetOrderHistory must not error when the 7 day listing adds the zero-fill canceled order")
			ids := make([]string, 0, len(history))
			for i := range history {
				ids = append(ids, history[i].OrderID)
			}
			assert.ElementsMatch(t, []string{"FILLED-1", "CANCELED-1"}, ids, "the crawl should keep the zero-fill canceled order beside the archive's rows without duplicating them")
		})

		t.Run("a window closed past the 7 day listing still reads it", func(t *testing.T) {
			t.Parallel()
			// OKX lists an order there for 7 days after it completes, so an older
			// order cancelled without fills within the last 2 hours is only there.
			oldCanceled := map[string]string{
				"instId": mainPair.String(), "ordId": "OLD-CANCELED", "ordType": orderLimit, "side": "buy",
				"state": "canceled", "sz": "1", "accFillSz": "0", "avgPx": "0", "px": "42000",
				"cTime": strconv.FormatInt(time.Now().Add(-10*24*time.Hour).UnixMilli(), 10),
			}
			ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/trade/orders-history-archive":
					writeOKXData(t, w, []map[string]string{})
				case "/trade/orders-history":
					writeOKXData(t, w, []map[string]string{oldCanceled})
				default:
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide,
				StartTime: time.Now().Add(-11 * 24 * time.Hour), EndTime: time.Now().Add(-9 * 24 * time.Hour),
				Pairs: currency.Pairs{mainPair},
			})
			require.NoError(t, err, "GetOrderHistory must not error when the window closes past the 7 day listing")
			require.Len(t, history, 1, "the 7 day listing must supply an older order cancelled without fills")
			assert.Equal(t, "OLD-CANCELED", history[0].OrderID, "the 7 day listing's order should be returned")
		})
	})

	t.Run("lists algo orders beside ordinary orders", func(t *testing.T) {
		t.Parallel()
		cancelled := map[string]string{
			"instType": instTypeSpot, "instId": mainPair.String(), "algoId": "82", "algoClOrdId": "client82", "ordType": orderTrigger, "side": "buy", "sz": "1",
			"state": "canceled", "triggerPx": "50000", "ordPx": "49000", "cTime": "1795409100000", "uTime": "1795409101000",
		}
		triggered := map[string]string{
			"instType": instTypeSpot, "instId": mainPair.String(), "algoId": "83", "algoClOrdId": "client83", "ordType": orderConditional, "side": "sell", "sz": "1",
			"state": "effective", "slTriggerPx": "45000", "slOrdPx": "-1", "cTime": "1795409200000", "uTime": "1795409201000",
		}
		filled := map[string]string{
			"instType": instTypeSpot, "instId": mainPair.String(), "ordId": "84", "ordType": orderLimit, "side": "buy", "state": "filled", "sz": "1", "px": "1",
			"accFillSz": "1", "avgPx": "1", "cTime": "1795409000000",
		}
		cancelledDetail := order.Detail{
			Amount: 1, Exchange: "Okx", OrderID: "82", ClientOrderID: "client82", Side: order.Buy, Type: order.Trigger, Pair: mainPair, AssetType: asset.Spot,
			Status: order.Cancelled, Price: 49000, TriggerPrice: 50000, Date: time.UnixMilli(1795409100000), LastUpdated: time.UnixMilli(1795409101000),
		}
		triggeredDetail := order.Detail{
			Amount: 1, Exchange: "Okx", OrderID: "83", ClientOrderID: "client83", Side: order.Sell, Type: order.ConditionalStop, Pair: mainPair, AssetType: asset.Spot,
			Status: order.Closed, Price: -1, TriggerPrice: 45000, Date: time.UnixMilli(1795409200000), LastUpdated: time.UnixMilli(1795409201000),
		}
		filledDetail := order.Detail{
			Price: 1, AverageExecutedPrice: 1, Amount: 1, ExecutedAmount: 1, Exchange: "Okx", OrderID: "84", Type: order.Limit, Side: order.Buy, Status: order.Filled,
			AssetType: asset.Spot, Date: time.UnixMilli(1795409000000), Pair: mainPair, Cost: 1, CostAsset: currency.USDT,
		}
		algoRequests := func(ordTypes ...string) []string {
			r := make([]string, 0, 3*len(ordTypes))
			for _, o := range ordTypes {
				for _, state := range []string{"effective", "canceled", "order_failed"} {
					r = append(r, "/trade/orders-algo-history SPOT "+o+" "+state)
				}
			}
			return r
		}
		for _, tc := range []struct {
			name      string
			orderType order.Type
			requests  []string
			exp       order.FilteredOrders
		}{
			{
				name: "any order type", orderType: order.AnyType,
				requests: append(algoRequests("trigger", "conditional,oco", "move_order_stop", "twap"), "/trade/orders-history-archive SPOT  ", "/trade/orders-history SPOT  "),
				exp:      order.FilteredOrders{cancelledDetail, triggeredDetail, filledDetail},
			},
			{name: "trigger orders", orderType: order.Trigger, requests: algoRequests("trigger"), exp: order.FilteredOrders{cancelledDetail}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var requests []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					q := r.URL.Query()
					mu.Lock()
					requests = append(requests, r.URL.Path+" "+q.Get("instType")+" "+q.Get("ordType")+" "+q.Get("state"))
					mu.Unlock()
					switch {
					case r.URL.Path == "/trade/orders-algo-history" && q.Get("ordType") == orderTrigger && q.Get("state") == "canceled":
						writeOKXData(t, w, []map[string]string{cancelled})
					case r.URL.Path == "/trade/orders-algo-history" && q.Get("ordType") == "conditional,oco" && q.Get("state") == "effective":
						writeOKXData(t, w, []map[string]string{triggered})
					case r.URL.Path == "/trade/orders-history-archive":
						writeOKXData(t, w, []map[string]string{filled})
					case r.URL.Path == "/trade/orders-algo-history", r.URL.Path == "/trade/orders-history":
						writeOKXData(t, w, []any{})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: tc.orderType, Side: order.AnySide})
				require.NoError(t, err, "GetOrderHistory must not error")
				assert.Equal(t, tc.exp, history, "GetOrderHistory should list the completed algo orders and ordinary orders of the requested type")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.requests, requests, "GetOrderHistory should read each algo order type's history in each state it lists")
			})
		}
	})

	t.Run("lists trigger orders by the time in force of the orders they place", func(t *testing.T) {
		t.Parallel()
		trigger := func(id, advanceOrdType, state string) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "algoId": id, "ordType": orderTrigger, "advanceOrdType": advanceOrdType, "side": "buy",
				"sz": "1", "state": state, "triggerPx": "110", "ordPx": "100", "cTime": "1795409100000", "uTime": "1795409101000",
			}
		}
		triggerDetail := func(id string, status order.Status, tif order.TimeInForce) order.Detail {
			return order.Detail{
				Amount: 1, Exchange: "Okx", OrderID: id, Side: order.Buy, Type: order.Trigger, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap,
				Status: status, Price: 100, TriggerPrice: 110, Date: time.UnixMilli(1795409100000), LastUpdated: time.UnixMilli(1795409101000), TimeInForce: tif,
			}
		}
		filled := map[string]string{
			"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "ordId": "94", "ordType": orderIOC, "side": "sell", "state": "filled", "sz": "2", "px": "90",
			"accFillSz": "2", "avgPx": "90", "cTime": "1795409000000",
		}
		filledDetail := order.Detail{
			Amount: 2, Price: 90, AverageExecutedPrice: 90, ExecutedAmount: 2, Exchange: "Okx", OrderID: "94", Type: order.Limit, Side: order.Sell,
			Status: order.Filled, AssetType: asset.PerpetualSwap, Date: time.UnixMilli(1795409000000), Pair: perpetualSwapPair, TimeInForce: order.ImmediateOrCancel,
		}
		triggerRequests := []string{
			"/trade/orders-algo-history SWAP trigger effective", "/trade/orders-algo-history SWAP trigger canceled", "/trade/orders-algo-history SWAP trigger order_failed",
		}
		for _, tc := range []struct {
			name      string
			orderType order.Type
			tif       order.TimeInForce
			requests  []string
			exp       order.FilteredOrders
		}{
			{
				name: "immediate or cancel trigger orders", orderType: order.Trigger, tif: order.ImmediateOrCancel, requests: triggerRequests,
				exp: order.FilteredOrders{triggerDetail("91", order.Closed, order.ImmediateOrCancel), triggerDetail("95", order.Cancelled, order.ImmediateOrCancel)},
			},
			{
				name: "fill or kill trigger orders", orderType: order.Trigger, tif: order.FillOrKill, requests: triggerRequests,
				exp: order.FilteredOrders{triggerDetail("92", order.Closed, order.FillOrKill)},
			},
			{
				name: "immediate or cancel orders of any type", orderType: order.AnyType, tif: order.ImmediateOrCancel,
				requests: append(slices.Clone(triggerRequests), "/trade/orders-history-archive SWAP  ", "/trade/orders-history SWAP  "),
				exp:      order.FilteredOrders{triggerDetail("91", order.Closed, order.ImmediateOrCancel), triggerDetail("95", order.Cancelled, order.ImmediateOrCancel), filledDetail},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var requests []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					q := r.URL.Query()
					mu.Lock()
					requests = append(requests, r.URL.Path+" "+q.Get("instType")+" "+q.Get("ordType")+" "+q.Get("state"))
					mu.Unlock()
					switch {
					case r.URL.Path == "/trade/orders-algo-history" && q.Get("state") == "effective":
						writeOKXData(t, w, []map[string]string{trigger("91", orderIOC, "effective"), trigger("92", orderFOK, "effective"), trigger("93", "", "effective")})
					case r.URL.Path == "/trade/orders-algo-history" && q.Get("state") == "canceled":
						writeOKXData(t, w, []map[string]string{trigger("95", orderIOC, "canceled")})
					case r.URL.Path == "/trade/orders-history-archive":
						writeOKXData(t, w, []map[string]string{filled})
					case r.URL.Path == "/trade/orders-algo-history", r.URL.Path == "/trade/orders-history":
						writeOKXData(t, w, []any{})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide})
				require.NoError(t, err, "GetOrderHistory must not error")
				assert.Equal(t, tc.exp, history, "GetOrderHistory should list the trigger orders read back with the requested time in force")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.requests, requests, "GetOrderHistory should read each state of the trigger order history for the requested time in force")
			})
		}
	})

	t.Run("lists the ordinary orders beside a trigger listing whose cursors cycle", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			tif  order.TimeInForce
			exp  []string
		}{
			{name: "immediate or cancel", tif: order.ImmediateOrCancel, exp: []string{"200", "800"}},
			{name: "fill or kill", tif: order.FillOrKill, exp: []string{"199", "801"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				triggers := pagerHandler(t, "/trade/orders-algo-history", "after", cyclingTriggerPages("effective"))
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch {
					case r.URL.Path == "/trade/orders-algo-history" && r.URL.Query().Get("state") == "effective":
						triggers(w, r)
					case r.URL.Path == "/trade/orders-algo-history":
						writeOKXData(t, w, []any{})
					case r.URL.Path == "/trade/orders-history-archive", r.URL.Path == "/trade/orders-history":
						writeOKXData(t, w, cyclingListingOrdinaryOrders("filled", r.URL.Query().Get("ordType")))
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: order.AnyType, TimeInForce: tc.tif, Side: order.AnySide})
				require.ErrorIs(t, err, errListingStalled, "GetOrderHistory must report the trigger listing which stops advancing")
				assert.ErrorContains(t, err, "listing trigger algo orders in the effective state", "the error should name the listing which stops advancing")
				assert.Equal(t, tc.exp, detailOrderIDs(history), "GetOrderHistory should return the trigger orders read and the ordinary history's orders beside the error")
			})
		}
	})

	t.Run("reports each order as OKX states it", func(t *testing.T) {
		t.Parallel()
		for _, tc := range statedOrderCases("canceled", order.Cancelled) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/trade/orders-history-archive":
						writeOKXData(t, w, []map[string]string{tc.row})
					case "/trade/orders-history":
						writeOKXData(t, w, []any{})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Type: order.AnyType, Side: order.AnySide})
				require.NoError(t, err, "GetOrderHistory must not error")
				exp := tc.exp
				exp.Exchange = ex.Name
				assert.Equal(t, order.FilteredOrders{exp}, history, "GetOrderHistory should report the order's sizes, trade currency and reduce only flag as OKX states them")
			})
		}
	})
}

func TestAsLimitMaker(t *testing.T) {
	t.Parallel()
	orders := func() []order.Detail {
		return []order.Detail{
			{OrderID: "1", Type: order.Limit, TimeInForce: order.PostOnly},
			{OrderID: "2", Type: order.Limit},
			{OrderID: "3", Type: order.Limit, TimeInForce: order.ImmediateOrCancel},
			{OrderID: "4", Type: order.MarketMakerProtection, TimeInForce: order.PostOnly},
			{OrderID: "5", Type: order.Market},
		}
	}
	limitMakers := orders()
	limitMakers[0].Type = order.LimitMaker
	for _, tc := range []struct {
		name    string
		reqType order.Type
		orders  []order.Detail
		exp     []order.Detail
	}{
		{name: "limit maker request", reqType: order.LimitMaker, orders: orders(), exp: limitMakers},
		{name: "limit request", reqType: order.Limit, orders: orders(), exp: orders()},
		{name: "any type request", reqType: order.AnyType, orders: orders(), exp: orders()},
		{name: "no orders", reqType: order.LimitMaker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			asLimitMaker(&order.MultiOrderRequest{Type: tc.reqType}, tc.orders)
			assert.Equal(t, tc.exp, tc.orders, "asLimitMaker should report post-only Limit orders as LimitMaker for a LimitMaker request only")
		})
	}
}

func TestSpreadOrderToDetail(t *testing.T) {
	t.Parallel()
	format := currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	created, updated := time.UnixMilli(1795231500000), time.UnixMilli(1795231800000)
	row := func(edit func(*SpreadOrderDetailResponse)) *SpreadOrderDetailResponse {
		so := &SpreadOrderDetailResponse{
			SpreadID: spreadPair.String(), OrderID: "2510789768709120", ClientOrderID: "b16", Price: -8.7, Size: 2, OrderType: orderLimit, Side: "buy",
			AccumulatedFillSize: 0.5, AveragePrice: -8.65, State: "partially_filled", CreationTime: types.Time(created), UpdateTime: types.Time(updated),
		}
		if edit != nil {
			edit(so)
		}
		return so
	}
	detail := func(edit func(*order.Detail)) order.Detail {
		d := order.Detail{
			Price: -8.7, AverageExecutedPrice: -8.65, Amount: 2, ExecutedAmount: 0.5, RemainingAmount: 1.5, Exchange: e.Name, OrderID: "2510789768709120",
			ClientOrderID: "b16", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.Spread, Date: created, LastUpdated: updated, Pair: spreadPair,
		}
		if edit != nil {
			edit(&d)
		}
		return d
	}
	for _, tc := range []struct {
		name string
		so   *SpreadOrderDetailResponse
		exp  order.Detail
	}{
		{name: "partially filled limit order", so: row(nil), exp: detail(nil)},
		{
			name: "filled order has nothing remaining",
			so:   row(func(so *SpreadOrderDetailResponse) { so.State, so.AccumulatedFillSize = "filled", 2 }),
			exp:  detail(func(d *order.Detail) { d.Status, d.ExecutedAmount, d.RemainingAmount = order.Filled, 2, 0 }),
		},
		{
			name: "filled order whose fills read short of its size has nothing remaining",
			so:   row(func(so *SpreadOrderDetailResponse) { so.State, so.AccumulatedFillSize = "filled", 1.5 }),
			exp:  detail(func(d *order.Detail) { d.Status, d.ExecutedAmount, d.RemainingAmount = order.Filled, 1.5, 0 }),
		},
		{
			name: "cancelled order reports its unfilled size",
			so:   row(func(so *SpreadOrderDetailResponse) { so.State = "canceled" }),
			exp:  detail(func(d *order.Detail) { d.Status = order.Cancelled }),
		},
		{
			name: "fills above the size leave nothing remaining",
			so:   row(func(so *SpreadOrderDetailResponse) { so.AccumulatedFillSize = 2.5 }),
			exp:  detail(func(d *order.Detail) { d.ExecutedAmount, d.RemainingAmount = 2.5, 0 }),
		},
		{
			name: "post only sell",
			so: row(func(so *SpreadOrderDetailResponse) {
				so.OrderType, so.Side, so.State, so.AccumulatedFillSize, so.AveragePrice = orderPostOnly, "sell", "live", 0, 0
			}),
			exp: detail(func(d *order.Detail) {
				d.TimeInForce, d.Side, d.Status, d.ExecutedAmount, d.AverageExecutedPrice, d.RemainingAmount = order.PostOnly, order.Sell, order.Active, 0, 0, 2
			}),
		},
		{
			name: "market order",
			so: row(func(so *SpreadOrderDetailResponse) {
				so.OrderType, so.Price, so.State, so.AccumulatedFillSize = orderMarket, 0, "filled", 2
			}),
			exp: detail(func(d *order.Detail) {
				d.Type, d.Price, d.Status, d.ExecutedAmount, d.RemainingAmount = order.Market, 0, order.Filled, 2, 0
			}),
		},
		{
			name: "size still to fill",
			so: row(func(so *SpreadOrderDetailResponse) {
				so.PendingFillSize, so.CancelledSize = OptionalNumber{Value: new(types.Number(1))}, 0.5
			}),
			exp: detail(func(d *order.Detail) { d.RemainingAmount = 1 }),
		},
		{
			name: "cancelled order stating nothing still to fill",
			so: row(func(so *SpreadOrderDetailResponse) {
				so.State, so.PendingFillSize, so.CancelledSize = "canceled", OptionalNumber{Value: new(types.Number(0))}, 1.5
			}),
			exp: detail(func(d *order.Detail) { d.Status, d.RemainingAmount = order.Cancelled, 0 }),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d, err := e.spreadOrderToDetail(tc.so, asset.Spread, format)
			require.NoError(t, err, "spreadOrderToDetail must not error")
			assert.Equal(t, tc.exp, d, "spreadOrderToDetail should convert every field")
		})
	}

	for _, tc := range []struct {
		name string
		so   *SpreadOrderDetailResponse
		err  error
		text string
	}{
		{name: "spread ID without the delimiter", so: row(func(so *SpreadOrderDetailResponse) { so.SpreadID = "BTCUSDT" }), text: "delimiter not found"},
		{name: "unknown order type", so: row(func(so *SpreadOrderDetailResponse) { so.OrderType = "unknown" }), err: order.ErrTypeIsInvalid},
		{name: "unknown side", so: row(func(so *SpreadOrderDetailResponse) { so.Side = "sideways" }), err: order.ErrSideIsInvalid},
		{name: "unknown state", so: row(func(so *SpreadOrderDetailResponse) { so.State = "unknown" }), text: "unrecognised order status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d, err := e.spreadOrderToDetail(tc.so, asset.Spread, format)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "spreadOrderToDetail must reject the row")
			} else {
				require.ErrorContains(t, err, tc.text, "spreadOrderToDetail must reject the row")
			}
			assert.Equal(t, order.Detail{}, d, "spreadOrderToDetail should return no detail beside the error")
		})
	}
}

// orderHistoryFake answers the order history listings with the page each path and after cursor selects, failing the
// path given as failing, and records every request it receives
type orderHistoryFake struct {
	mu       sync.Mutex
	requests []string
	pages    map[string][]map[string]string
	failing  string
}

func (f *orderHistoryFake) serve(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.URL.Path+"?"+r.URL.Query().Encode())
		f.mu.Unlock()
		switch r.URL.Path {
		case "/trade/orders-history-archive", "/trade/orders-history":
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == f.failing {
			_, err := w.Write([]byte(`{"code":"50011","msg":"","data":[]}`))
			assert.NoError(t, err, "writing the error reply should not error")
			return
		}
		page, ok := f.pages[r.URL.Path+"|"+r.URL.Query().Get("after")]
		if !ok {
			page = []map[string]string{}
		}
		writeOKXData(t, w, page)
	}
}

func (f *orderHistoryFake) received() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

func TestGetStandardOrderHistoryDetails(t *testing.T) {
	t.Parallel()
	const archive, sevenDays = "/trade/orders-history-archive", "/trade/orders-history"
	const created = int64(1795231500000)
	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)
	row := func(id, instID, ordType, state string, cTime int64) map[string]string {
		return map[string]string{
			"instId": instID, "ordId": id, "clOrdId": "client" + id, "ordType": ordType, "side": "buy", "px": "100", "sz": "2", "accFillSz": "2", "avgPx": "100",
			"state": state, "fee": "-0.002", "feeCcy": "BTC", "cTime": strconv.FormatInt(cTime, 10), "uTime": strconv.FormatInt(cTime+1000, 10),
		}
	}
	filled := func(id string, cTime int64) map[string]string {
		return row(id, mainPair.String(), orderLimit, "filled", cTime)
	}
	detail := func(id string, pair currency.Pair, a asset.Item, oType order.Type, tif order.TimeInForce, cTime int64) order.Detail {
		d := order.Detail{
			Price: 100, AverageExecutedPrice: 100, Amount: 2, ExecutedAmount: 2, Fee: 0.002, FeeAsset: currency.BTC, Exchange: "Okx", OrderID: id, ClientOrderID: "client" + id,
			Type: oType, Side: order.Buy, Status: order.Filled, AssetType: a, Date: time.UnixMilli(cTime), LastUpdated: time.UnixMilli(cTime + 1000), Pair: pair, TimeInForce: tif,
		}
		if a == asset.Spot || a == asset.Margin {
			d.Cost, d.CostAsset = 200, pair.Quote
		}
		return d
	}
	spotDetail := func(id string, cTime int64) order.Detail {
		return detail(id, mainPair, asset.Spot, order.Limit, order.UnknownTIF, cTime)
	}
	fullPage := func(newest int) ([]map[string]string, []order.Detail) {
		rows := make([]map[string]string, 0, orderListPageSize)
		details := make([]order.Detail, 0, orderListPageSize)
		for id := newest; id > newest-orderListPageSize; id-- {
			cTime := created - int64(newest-id)
			rows = append(rows, filled(strconv.Itoa(id), cTime))
			details = append(details, spotDetail(strconv.Itoa(id), cTime))
		}
		return rows, details
	}
	firstPage, firstPageDetails := fullPage(2000)
	spotQuery := "instType=SPOT"
	for _, tc := range []struct {
		name     string
		req      *order.MultiOrderRequest
		pages    map[string][]map[string]string
		failing  string
		exp      []order.Detail
		requests []string
		err      error
	}{
		{
			name: "both listings, de-duplicated", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages: map[string][]map[string]string{
				archive + "|":   {filled("3", created), filled("2", created-1)},
				sevenDays + "|": {filled("4", created+1), filled("2", created-1)},
			},
			exp:      []order.Detail{spotDetail("3", created), spotDetail("2", created-1), spotDetail("4", created+1)},
			requests: []string{archive + "?" + spotQuery, sevenDays + "?" + spotQuery},
		},
		{
			name: "a full page continues from its last order ID", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages:    map[string][]map[string]string{archive + "|": firstPage, archive + "|1901": {filled("1900", created-100)}},
			exp:      append(slices.Clone(firstPageDetails), spotDetail("1900", created-100)),
			requests: []string{archive + "?" + spotQuery, archive + "?after=1901&" + spotQuery, sevenDays + "?" + spotQuery},
		},
		{
			name: "a repeated page stops the crawl", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages:    map[string][]map[string]string{archive + "|": firstPage, archive + "|1901": firstPage},
			exp:      firstPageDetails,
			requests: []string{archive + "?" + spotQuery, archive + "?after=1901&" + spotQuery, sevenDays + "?" + spotQuery},
		},
		{
			name: "orders created before the start time end each listing",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, StartTime: time.UnixMilli(created).Add(500 * time.Microsecond)},
			pages: map[string][]map[string]string{
				archive + "|":   {filled("5", created+1), filled("4", created), filled("3", created-1), filled("2", created)},
				sevenDays + "|": {filled("6", created-1)},
			},
			exp:      []order.Detail{spotDetail("5", created+1), spotDetail("4", created)},
			requests: []string{archive + "?" + spotQuery, sevenDays + "?" + spotQuery},
		},
		{
			name: "the end time and the starting order ID", req: &order.MultiOrderRequest{AssetType: asset.Spot, EndTime: time.UnixMilli(created), FromOrderID: "10"},
			pages:    map[string][]map[string]string{archive + "|10": {filled("9", created)}},
			exp:      []order.Detail{spotDetail("9", created)},
			requests: []string{archive + "?after=10&end=1795231500000&" + spotQuery, sevenDays + "?after=10&end=1795231500000&" + spotQuery},
		},
		{
			name: "the requested pairs", req: &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{ethUSDT}},
			pages:    map[string][]map[string]string{archive + "|": {filled("2", created), row("1", "ETH-USDT", orderLimit, "filled", created-1)}},
			exp:      []order.Detail{detail("1", ethUSDT, asset.Spot, order.Limit, order.UnknownTIF, created-1)},
			requests: []string{archive + "?" + spotQuery, sevenDays + "?" + spotQuery},
		},
		{
			name: "a time in force keeps its order types", req: &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Limit, TimeInForce: order.PostOnly},
			pages: map[string][]map[string]string{
				archive + "|": {filled("3", created), row("2", mainPair.String(), orderPostOnly, "filled", created-1), row("1", mainPair.String(), orderIOC, "filled", created-2)},
			},
			exp:      []order.Detail{detail("2", mainPair, asset.Spot, order.Limit, order.PostOnly, created-1)},
			requests: []string{archive + "?" + spotQuery, sevenDays + "?" + spotQuery},
		},
		{
			name: "live orders are left out", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages:    map[string][]map[string]string{sevenDays + "|": {row("2", mainPair.String(), orderLimit, "live", created), filled("1", created-1)}},
			exp:      []order.Detail{spotDetail("1", created-1)},
			requests: []string{archive + "?" + spotQuery, sevenDays + "?" + spotQuery},
		},
		{
			name: "perpetual swap orders leave the cost unset", req: &order.MultiOrderRequest{AssetType: asset.PerpetualSwap},
			pages:    map[string][]map[string]string{archive + "|": {row("1", perpetualSwapPair.String(), orderMarket, "filled", created)}},
			exp:      []order.Detail{detail("1", perpetualSwapPair, asset.PerpetualSwap, order.Market, order.UnknownTIF, created)},
			requests: []string{archive + "?instType=SWAP", sevenDays + "?instType=SWAP"},
		},
		{name: "archive error", req: &order.MultiOrderRequest{AssetType: asset.Spot}, failing: archive, requests: []string{archive + "?" + spotQuery}, err: ErrorCodes["50011"]},
		{name: "seven day listing error", req: &order.MultiOrderRequest{AssetType: asset.Spot}, failing: sevenDays, requests: []string{archive + "?" + spotQuery, sevenDays + "?" + spotQuery}, err: ErrorCodes["50011"]},
		{
			name: "an instrument ID too short for a pair", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages: map[string][]map[string]string{archive + "|": {row("1", "", orderLimit, "filled", created)}}, requests: []string{archive + "?" + spotQuery}, err: currency.ErrCreatingPair,
		},
		{
			name: "an unknown order type", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages: map[string][]map[string]string{archive + "|": {row("1", mainPair.String(), "unknown", "filled", created)}}, requests: []string{archive + "?" + spotQuery}, err: order.ErrTypeIsInvalid,
		},
		{name: "an order type with no listing filter", req: &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Stop, TimeInForce: order.PostOnly}, err: order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &orderHistoryFake{pages: tc.pages, failing: tc.failing}
			ex := newMockExchange(t, f.serve(t))
			details, err := ex.getStandardOrderHistoryDetails(t.Context(), tc.req)
			require.ErrorIs(t, err, tc.err, "getStandardOrderHistoryDetails must return the expected error")
			assert.Equal(t, tc.exp, details, "getStandardOrderHistoryDetails should return every completed order the listings hold")
			assert.Equal(t, tc.requests, f.received(), "getStandardOrderHistoryDetails should crawl the archive then the 7 day listing")
		})
	}

	t.Run("an unknown state", func(t *testing.T) {
		t.Parallel()
		f := &orderHistoryFake{pages: map[string][]map[string]string{archive + "|": {row("1", mainPair.String(), orderLimit, "unknown", created)}}}
		_, err := newMockExchange(t, f.serve(t)).getStandardOrderHistoryDetails(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot})
		assert.ErrorContains(t, err, "unrecognised order status", "getStandardOrderHistoryDetails should reject an order state it cannot read")
	})

	t.Run("a cancelled context", func(t *testing.T) {
		t.Parallel()
		f := &orderHistoryFake{}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := newMockExchange(t, f.serve(t)).getStandardOrderHistoryDetails(ctx, &order.MultiOrderRequest{AssetType: asset.Spot})
		require.ErrorIs(t, err, context.Canceled, "getStandardOrderHistoryDetails must return the context error")
		assert.Empty(t, f.received(), "getStandardOrderHistoryDetails should send no request once the context is cancelled")
	})
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetFeeByType(contextGenerate(), &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	err := e.ValidateAPICredentials(contextGenerate(), asset.Spot)
	assert.NoError(t, err, "ValidateAPICredentials should not error")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	_, err := e.GetHistoricCandles(contextGenerate(), currency.Pair{}, asset.Binary, kline.OneDay, time.Now(), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported)

	startTime := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 0, 100)
	_, err = e.GetHistoricCandles(contextGenerate(), mainPair, asset.Spot, kline.Interval(time.Hour*4), startTime, endTime)
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits)

	ex := e
	start, end := time.Now().Add(-time.Hour), time.Now()
	mockPairs := map[asset.Item]currency.Pair{
		asset.Spot:          mainPair,
		asset.Margin:        mainPair,
		asset.PerpetualSwap: btcUSDSwapPair,
		asset.Futures:       mockFuturesContract,
		asset.Options:       currency.NewPairWithDelimiter("BTC", "USD-261123-46000-C", currency.DashDelimiter),
		asset.Spread:        spreadPair,
	}
	flatCandles := func(price float64) []kline.Candle {
		return []kline.Candle{
			{Time: time.UnixMilli(1795410000000), Open: price, High: price, Low: price, Close: price},
			{Time: time.UnixMilli(1795410060000), Open: price, High: price, Low: price, Close: price},
			{Time: time.UnixMilli(1795410120000), Open: price, High: price, Low: price, Close: price},
		}
	}
	mockCandles := map[asset.Item][]kline.Candle{
		asset.Spot:          mockSpotCandles,
		asset.Margin:        mockSpotCandles,
		asset.PerpetualSwap: mockInverseSwapCandles,
		asset.Futures:       flatCandles(60205.8),
		asset.Options:       flatCandles(0.2365),
		asset.Spread:        flatCandles(-30.5),
	}
	if mockTests {
		// The candle requests need enabled pairs, and enabling these on e would change the pairs the tests beside this
		// one read
		ex = newIsolatedExchange(t)
		start, end = mockCandlesStart, mockCandlesEnd
		for a, p := range mockPairs {
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, a, p)
		}
	} else {
		testexch.UpdatePairsOnce(t, e)
	}
	for _, a := range ex.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			p, ok := mockPairs[a]
			require.Truef(t, ok, "a pair must be chosen for asset %s", a)
			if !mockTests {
				pairs, err := e.GetEnabledPairs(a)
				require.NoErrorf(t, err, "GetEnabledPairs for asset %s must not error", a)
				require.NotEmptyf(t, pairs, "GetEnabledPairs for asset %s must not return empty pairs", a)
				p = pairs[0]
			}
			result, err := ex.GetHistoricCandles(contextGenerate(), p, a, kline.OneMin, start, end)
			if !mockTests && (a == asset.Spread || a == asset.Options || a == asset.Futures) && errors.Is(err, kline.ErrNoTimeSeriesDataToConvert) {
				return // These market types can legitimately return no candles for some windows.
			}
			require.NoErrorf(t, err, "GetHistoricCandles for asset %s and pair %s must not error", a, p)
			if mockTests {
				exp := &kline.Item{Exchange: ex.Name, Pair: p, Asset: a, Interval: kline.OneMin, Candles: mockCandles[a]}
				assert.Equalf(t, exp, result, "GetHistoricCandles should map the recorded %s candles", a)
			} else {
				assert.NotNilf(t, result, "GetHistoricCandles for asset %s and pair %s should not return nil", a, p)
			}
		})
	}
}

// mockCandlesStart and mockCandlesEnd bound the one minute candles recorded for the candle tests
var mockCandlesStart, mockCandlesEnd = time.UnixMilli(1795410000000), time.UnixMilli(1795410180000)

// mockSpotCandles are the BTC-USDT one minute candles recorded between mockCandlesStart and mockCandlesEnd
var mockSpotCandles = []kline.Candle{
	{Time: time.UnixMilli(1795410000000), Open: 61242.5, High: 61275, Low: 61219.8, Close: 61275, Volume: 1.96025751},
	{Time: time.UnixMilli(1795410060000), Open: 61275, High: 61352.8, Low: 61275, Close: 61319, Volume: 4.34452104},
	{Time: time.UnixMilli(1795410120000), Open: 61319.6, High: 61359.6, Low: 61319.2, Close: 61332.4, Volume: 1.11752764},
}

// mockInverseSwapCandles are the BTC-USD-SWAP one minute candles recorded between mockCandlesStart and
// mockCandlesEnd, whose volume is volCcy, in BTC, rather than vol's count of 100 USD contracts
var mockInverseSwapCandles = []kline.Candle{
	{Time: time.UnixMilli(1795410000000), Open: 61126.5, High: 61146.5, Low: 61110, Close: 61146.5, Volume: 0.8091},
	{Time: time.UnixMilli(1795410060000), Open: 61158.6, High: 61234.7, Low: 61158.6, Close: 61192.6, Volume: 1.1593},
	{Time: time.UnixMilli(1795410120000), Open: 61201.4, High: 61242, Low: 61201.4, Close: 61217.4, Volume: 1.1038},
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start, end := time.Now().Add(-time.Hour), time.Now()
	if mockTests {
		start, end = mockCandlesStart, mockCandlesEnd
	}
	for _, tc := range []struct {
		pair    currency.Pair
		a       asset.Item
		candles []kline.Candle
	}{
		{mainPair, asset.Spot, mockSpotCandles},
		{btcUSDSwapPair, asset.PerpetualSwap, mockInverseSwapCandles},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetHistoricCandlesExtended(contextGenerate(), tc.pair, tc.a, kline.OneMin, start, end)
			require.NoError(t, err, "GetHistoricCandlesExtended must not error")
			if mockTests {
				exp := &kline.Item{Exchange: e.Name, Pair: tc.pair, Asset: tc.a, Interval: kline.OneMin, Candles: tc.candles}
				assert.Equal(t, exp, result, "GetHistoricCandlesExtended should map the recorded candles with their base currency volume")
			} else {
				assert.NotNil(t, result, "GetHistoricCandlesExtended should return candles")
			}
		})
	}
}

func TestCandleVolume(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a   asset.Item
		exp float64
	}{
		{asset.Spot, 1000},
		{asset.Margin, 1000},
		{asset.PerpetualSwap, 10},
		{asset.Futures, 10},
		{asset.Options, 10},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, candleVolume(1000, 10, tc.a), "candleVolume should return the base currency volume: vol on spot and margin, volCcy on derivatives")
		})
	}
}

func TestOrderPushData(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsOrders.jsonl", func(ctx context.Context, b []byte) error { return e.wsHandleData(ctx, nil, b) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 4, "Should see 4 orders")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case *order.Detail:
			switch len(e.Websocket.DataHandler.C) {
			case 3:
				assert.Equal(t, "452197707845865472", v.OrderID, "OrderID")
				assert.Equal(t, "HamsterParty14", v.ClientOrderID, "ClientOrderID")
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
				assert.Equal(t, order.Sell, v.Side, "Side")
				assert.Equal(t, order.Filled, v.Status, "Status")
				assert.Equal(t, order.Limit, v.Type, "Type")
				assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "-"), v.Pair, "Pair")
				assert.Equal(t, 31527.1, v.AverageExecutedPrice, "AverageExecutedPrice")
				assert.Equal(t, time.UnixMilli(1654084334977), v.Date, "Date")
				assert.Equal(t, time.UnixMilli(1654084353263), v.CloseTime, "CloseTime")
				assert.Equal(t, 0.001, v.Amount, "Amount")
				assert.Equal(t, 0.001, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.000, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 31527.1, v.Price, "Price")
				assert.Equal(t, 0.02522168, v.Fee, "Fee")
				assert.Equal(t, currency.USDT, v.FeeAsset, "FeeAsset")
			case 2:
				assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
				assert.Equal(t, order.Market, v.Type, "Type")
				assert.Equal(t, order.Sell, v.Side, "Side")
				assert.Equal(t, order.Active, v.Status, "Status")
				assert.Equal(t, 0.0, v.Amount, "Amount should be 0 for a market sell")
				assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
			case 1:
				assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
				assert.Equal(t, 0.00038127046945832905, v.Amount, "Amount")
				assert.Equal(t, 0.010000249968, v.Fee, "Fee")
				assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, order.PartiallyFilled, v.Status, "Status")
			case 0:
				assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
				assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
				assert.Equal(t, 0.010000249968, v.Fee, "Fee")
				assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
				assert.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.00038128, v.Amount, "Amount should be derived because order filled")
				assert.Equal(t, order.Filled, v.Status, "Status")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Got unexpected data: %T %v", v, v)
		}
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(contextGenerate(), mainPair, asset.Spread, time.Now(), time.Now())
	require.ErrorIs(t, err, asset.ErrNotSupported)

	t.Run("judges the three month limit by the wrapper's clock", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			age  time.Duration
			err  error
		}{
			{"a start a minute past three months", kline.ThreeMonth.Duration() + time.Minute, errOnlyThreeMonthsSupported},
			{"a start a minute inside three months", kline.ThreeMonth.Duration() - time.Minute, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/market/history-trades", r.URL.Path, "GetHistoricTrades should only request the trade history")
					writeOKXData(t, w, []any{})
				}))
				now := timeNow()
				_, err := ex.GetHistoricTrades(t.Context(), mainPair, asset.Spot, now.Add(-tc.age), now)
				assert.ErrorIs(t, err, tc.err, "GetHistoricTrades should judge the start by the wrapper's clock")
			})
		}
	})

	start, end := time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2)
	if mockTests {
		// Inside the recorded page: its oldest trade ends the crawl and the end bound leaves out its newest
		start, end = time.UnixMilli(1795412424420), time.UnixMilli(1795412425800)
	}
	result, err := e.GetHistoricTrades(contextGenerate(), mainPair, asset.Spot, start, end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	if mockTests {
		exp := []trade.Data{
			{TID: "1089334512", Exchange: e.Name, CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Buy, Price: 61624.7, Amount: 0.00145071, Timestamp: time.UnixMilli(1795412424426)},
			{TID: "1089334513", Exchange: e.Name, CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Sell, Price: 61624.6, Amount: 0.00017919, Timestamp: time.UnixMilli(1795412425532)},
			{TID: "1089334514", Exchange: e.Name, CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Sell, Price: 61624.6, Amount: 0.00651595, Timestamp: time.UnixMilli(1795412425606)},
		}
		assert.Equal(t, exp, result, "GetHistoricTrades should map the trades inside the window, oldest first")
		return
	}
	require.NotEmpty(t, result, "GetHistoricTrades must return trades for a recent window")
	for _, tr := range result {
		assert.Equal(t, mainPair, tr.CurrencyPair, "trade should carry the requested pair")
		assert.Equal(t, asset.Spot, tr.AssetType, "trade should carry the requested asset")
		assert.WithinRange(t, tr.Timestamp, start, end, "trade should fall inside the requested window")
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetServerTime(contextGenerate(), asset.Empty)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetAvailableTransferChains(contextGenerate(), currency.BTC)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetLatestFundingRates(contextGenerate(), &fundingrate.LatestRateRequest{
			Asset:                asset.PerpetualSwap,
			Pair:                 perpetualSwapPair,
			IncludePredictedRate: true,
		})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
	for _, predicted := range []bool{false, true} {
		t.Run("observation "+strconv.FormatBool(predicted), func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			at := time.Now().UTC().Truncate(time.Millisecond)
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/public/funding-rate", r.URL.Path, "request should use the public endpoint")
				_, err := fmt.Fprintf(w, `{"code":"0","data":[{"instId":"BTC-USDT-SWAP","fundingRate":"0.0003","nextFundingRate":"0.9","ts":"%d","fundingTime":"%d","nextFundingTime":"%d"}]}`, at.UnixMilli(), at.Add(time.Hour).UnixMilli(), at.Add(9*time.Hour).UnixMilli())
				assert.NoError(t, err, "response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
			got, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, IncludePredictedRate: predicted})
			require.NoError(t, err, "funding observation must load")
			require.Len(t, got, 1, "one market must be returned")
			assert.Equal(t, at, got[0].LatestRate.Time.UTC(), "latest rate should use observation time")
			assert.Equal(t, "0.0003", got[0].LatestRate.Rate.String(), "latest rate should retain the indicative value")
			assert.Equal(t, at.Add(time.Hour), got[0].TimeOfNextRate.UTC(), "next rate should refer to imminent settlement")
			if predicted {
				assert.Equal(t, at.Add(time.Hour), got[0].PredictedUpcomingRate.Time.UTC(), "prediction should target imminent settlement")
				assert.Equal(t, "0.0003", got[0].PredictedUpcomingRate.Rate.String(), "prediction should use the current estimate")
			} else {
				assert.True(t, got[0].PredictedUpcomingRate.Time.IsZero(), "excluded prediction should remain unset")
			}
		})
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		request *fundingrate.HistoricalRatesRequest
		want    error
	}{
		{name: "nil request", want: common.ErrNilPointer},
		{name: "unsupported asset", request: &fundingrate.HistoricalRatesRequest{Asset: asset.Spot, Pair: perpetualSwapPair}, want: futures.ErrNotPerpetualFuture},
		{name: "empty pair", request: &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap}, want: currency.ErrCurrencyPairEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := new(Exchange).GetHistoricalFundingRates(t.Context(), tc.request)
			require.ErrorIs(t, err, tc.want, "invalid request must fail before accessing exchange state")
			assert.Nil(t, result, "invalid request should return no rates")
		})
	}

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		// Bound network delays and pagination regressions independently of the test suite timeout.
		ctx, cancel := context.WithTimeout(contextGenerate(), 30*time.Second)
		t.Cleanup(cancel)

		end := timeNow()
		if mockTests {
			end = time.UnixMilli(1795410000000)
		}
		r := &fundingrate.HistoricalRatesRequest{
			Asset:                asset.PerpetualSwap,
			Pair:                 perpetualSwapPair,
			PaymentCurrency:      currency.USDT,
			StartDate:            end.Add(-time.Hour * 24 * 2),
			EndDate:              end,
			IncludePredictedRate: true,
		}

		r.StartDate = end.Add(-time.Hour * 24 * 120)
		_, err := e.GetHistoricalFundingRates(ctx, r)
		require.ErrorIs(t, err, fundingrate.ErrFundingRateOutsideLimits)

		if sharedtestvalues.AreAPICredentialsSet(e) {
			r.IncludePayments = true
		}
		r.StartDate = end.Add(-time.Hour * 24 * 12)
		result, err := e.GetHistoricalFundingRates(ctx, r)
		require.NoError(t, err, "GetHistoricalFundingRates must not error")
		require.NotNil(t, result, "GetHistoricalFundingRates must return rates")
		var exp *fundingrate.HistoricalRates
		if mockTests {
			exp = &fundingrate.HistoricalRates{
				Exchange:              e.Name,
				Asset:                 asset.PerpetualSwap,
				Pair:                  perpetualSwapPair,
				StartDate:             time.UnixMilli(1794373200000),
				EndDate:               time.UnixMilli(1795410000000),
				LatestRate:            fundingrate.Rate{Time: time.UnixMilli(1795406714475), Rate: decimal.MustFromFloat(0.0000275120739438)},
				PredictedUpcomingRate: fundingrate.Rate{Time: time.UnixMilli(1795420800000), Rate: decimal.MustFromFloat(0.0000275120739438)},
				FundingRates: []fundingrate.Rate{
					{Time: time.UnixMilli(1795392000000), Rate: decimal.MustFromFloat(0.0000151578247325), Payment: decimal.MustFromFloat(-0.4642)},
					{Time: time.UnixMilli(1795363200000), Rate: decimal.MustFromFloat(0.0000227750962068), Payment: decimal.MustFromFloat(-0.697)},
					{Time: time.UnixMilli(1795334400000), Rate: decimal.MustFromFloat(0.0000340541459576)},
					{Time: time.UnixMilli(1795305600000), Rate: decimal.MustFromFloat(-0.0000376857530396), Payment: decimal.MustFromFloat(1.149)},
				},
				PaymentSum:      decimal.MustFromFloat(-0.0122),
				PaymentCurrency: currency.USDT,
				TimeOfNextRate:  time.UnixMilli(1795420800000),
			}
			assert.Equal(t, exp, result, "GetHistoricalFundingRates should map the recorded rates and payments")
		}

		r.RespectHistoryLimits = true
		result, err = e.GetHistoricalFundingRates(ctx, r)
		require.NoError(t, err, "GetHistoricalFundingRates must not error when respecting the history limits")
		if mockTests {
			assert.Equal(t, exp, result, "GetHistoricalFundingRates should return the same rates when the window is inside the limits")
		} else {
			assert.NotNil(t, result, "GetHistoricalFundingRates should return rates when respecting the history limits")
		}
	})
	for _, tc := range []struct {
		name, realised, want string
		missing              bool
	}{
		{"positive", `,"realizedRate":"0.0002"`, "0.0002", false},
		{"negative", `,"realizedRate":"-0.0002"`, "-0.0002", false},
		{"zero", `,"realizedRate":"0"`, "0", false},
		{"missing", "", "", true},
		{"empty", `,"realizedRate":""`, "", true},
		{"null", `,"realizedRate":null`, "", true},
	} {
		t.Run("settlement "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			require.NoError(t, ex.DisableRateLimiter(), "rate limiter must disable")
			at := timeNow().UTC().Truncate(time.Millisecond)
			server := httptest.NewTestServer(t, cappedHandler(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/public/funding-rate-history":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"fundingRate":"0.9","fundingTime":"%d"%s}]}`, at.Add(-time.Hour).UnixMilli(), tc.realised)
					assert.NoError(t, err, "history should write")
				case "/public/funding-rate":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"fundingRate":"0.0003","nextFundingRate":"0.9","ts":"%d","fundingTime":"%d","nextFundingTime":"%d"}]}`, at.UnixMilli(), at.Add(time.Hour).UnixMilli(), at.Add(9*time.Hour).UnixMilli())
					assert.NoError(t, err, "current funding should write")
				default:
					assert.Fail(t, "unexpected endpoint", "request should use funding endpoints")
				}
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
			got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, StartDate: at.Add(-2 * time.Hour), EndDate: at, IncludePredictedRate: true})
			if tc.missing {
				require.ErrorIs(t, err, common.ErrNoResponse, "unknown settlement must fail closed")
				return
			}
			require.NoError(t, err, "settlement must load")
			require.Len(t, got.FundingRates, 1, "one settled rate must load")
			assert.Equal(t, tc.want, got.FundingRates[0].Rate.String(), "history should use the realised value")
			assert.Equal(t, at, got.LatestRate.Time.UTC(), "latest rate should use observation time")
			assert.Equal(t, at.Add(time.Hour), got.TimeOfNextRate.UTC(), "next rate should target imminent settlement")
			assert.Equal(t, "0.0003", got.PredictedUpcomingRate.Rate.String(), "prediction should use the current estimate")
		})
	}
	for _, tc := range []struct {
		name, instrumentID, instruments string
		paymentCurrency, wantCurrency   currency.Code
		wantErr                         error
	}{
		{name: "USDT linear", instrumentID: "BTC-USDT-SWAP", instruments: `[{"instId":"BTC-USDT-SWAP","instType":"SWAP","settleCcy":"USDT"}]`, wantCurrency: currency.USDT},
		{name: "USDC linear", instrumentID: "BTC-USDC-SWAP", instruments: `[{"instId":"BTC-USDC-SWAP","instType":"SWAP","settleCcy":"USDC"}]`, wantCurrency: currency.USDC},
		{name: "BTC inverse", instrumentID: "BTC-USD-SWAP", instruments: `[{"instId":"BTC-USD-SWAP","instType":"SWAP","settleCcy":"BTC"}]`, wantCurrency: currency.BTC},
		{name: "explicit currency bypasses lookup", instrumentID: "BTC-USDT-SWAP", paymentCurrency: currency.USDT, wantCurrency: currency.USDT},
		{name: "absent instrument", instrumentID: "BTC-USDT-SWAP", instruments: `[]`, wantErr: common.ErrNoResponse},
		{name: "ambiguous instruments", instrumentID: "BTC-USDT-SWAP", instruments: `[{"instId":"BTC-USDT-SWAP","instType":"SWAP","settleCcy":"USDT"},{"instId":"BTC-USDT-SWAP","instType":"SWAP","settleCcy":"USDC"}]`, wantErr: common.ErrNoResponse},
		{name: "missing settlement currency", instrumentID: "BTC-USDT-SWAP", instruments: `[{"instId":"BTC-USDT-SWAP","instType":"SWAP"}]`, wantErr: common.ErrNoResponse},
		{name: "wrong instrument", instrumentID: "BTC-USDT-SWAP", instruments: `[{"instId":"ETH-USDT-SWAP","instType":"SWAP","settleCcy":"USDT"}]`, wantErr: common.ErrNoResponse},
		{name: "wrong instrument type", instrumentID: "BTC-USDT-SWAP", instruments: `[{"instId":"BTC-USDT-SWAP","instType":"FUTURES","settleCcy":"USDT"}]`, wantErr: common.ErrNoResponse},
	} {
		t.Run("payment currency "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			require.NoError(t, ex.DisableRateLimiter(), "rate limiter must disable")
			ex.SkipAuthCheck = true
			ex.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
			pair, err := currency.NewPairFromString(tc.instrumentID)
			require.NoError(t, err, "instrument pair must parse")
			at := timeNow().UTC().Truncate(time.Millisecond)
			server := httptest.NewTestServer(t, cappedHandler(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/public/funding-rate-history":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"realizedRate":"0.0001","fundingTime":"%d"}]}`, at.Add(-time.Hour).UnixMilli())
					assert.NoError(t, err, "funding history should write")
				case "/public/funding-rate":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"fundingRate":"0.0001","ts":"%d","fundingTime":"%d"}]}`, at.UnixMilli(), at.Add(time.Hour).UnixMilli())
					assert.NoError(t, err, "current rate should write")
				case "/public/instruments":
					assert.True(t, tc.paymentCurrency.IsEmpty(), "explicit payment currency should skip metadata lookup")
					assert.Equal(t, "SWAP", r.URL.Query().Get("instType"), "metadata request should scope perpetual swaps")
					assert.Equal(t, tc.instrumentID, r.URL.Query().Get("instId"), "metadata request should select the exact instrument")
					_, err := fmt.Fprintf(w, `{"code":"0","data":%s}`, tc.instruments)
					assert.NoError(t, err, "instruments should write")
				case "/account/bills":
					assert.NoError(t, tc.wantErr, "invalid settlement metadata should prevent bills retrieval")
					assert.Equal(t, tc.wantCurrency.String(), r.URL.Query().Get("ccy"), "bills should use the instrument settlement currency")
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"billId":"1","pnl":"2","ts":"%d"}]}`, at.Add(-time.Hour).UnixMilli())
					assert.NoError(t, err, "bill should write")
				default:
					assert.Fail(t, "unexpected endpoint", "funding payment lookup should only use expected endpoints")
				}
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
			got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: pair, PaymentCurrency: tc.paymentCurrency, StartDate: at.Add(-2 * time.Hour), EndDate: at, IncludePayments: true})
			require.ErrorIs(t, err, tc.wantErr, "payment currency resolution must return the expected outcome")
			if tc.wantErr != nil {
				assert.Nil(t, got, "invalid metadata should return no result")
				return
			}
			assert.Equal(t, tc.wantCurrency, got.PaymentCurrency, "response should report the currency used for bills")
			assert.Equal(t, "2", got.PaymentSum.String(), "payment in the resolved currency should be retained")
		})
	}
	t.Run("X-Perps future", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "exchange must initialise")
		require.NoError(t, ex.DisableRateLimiter(), "rate limiter must disable")
		ex.SkipAuthCheck = true
		ex.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
		pair, err := currency.NewPairFromString("BTC-USD_UM_XPERP-310404")
		require.NoError(t, err, "X-Perps pair must parse")
		at := timeNow().UTC().Truncate(time.Millisecond)
		server := httptest.NewTestServer(t, cappedHandler(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/funding-rate-history":
				assert.Equal(t, pair.String(), r.URL.Query().Get("instId"), "history should select the X-Perps future")
				_, err := fmt.Fprintf(w, `{"code":"0","data":[{"realizedRate":"0.0001","fundingTime":"%d"}]}`, at.Add(-time.Hour).UnixMilli())
				assert.NoError(t, err, "history should write")
			case "/public/funding-rate":
				_, err := fmt.Fprintf(w, `{"code":"0","data":[{"fundingRate":"0.0001","ts":"%d","fundingTime":"%d"}]}`, at.UnixMilli(), at.Add(time.Hour).UnixMilli())
				assert.NoError(t, err, "current rate should write")
			case "/public/instruments":
				assert.Equal(t, "FUTURES", r.URL.Query().Get("instType"), "metadata request should scope futures")
				_, err := fmt.Fprintf(w, `{"code":"0","data":[{"instId":%q,"instType":"FUTURES","settleCcy":"USD"}]}`, pair.String())
				assert.NoError(t, err, "instruments should write")
			case "/account/bills":
				assert.Equal(t, "FUTURES", r.URL.Query().Get("instType"), "bills should scope futures")
				assert.Equal(t, "USD", r.URL.Query().Get("ccy"), "bills should use the settlement currency")
				_, err := fmt.Fprintf(w, `{"code":"0","data":[{"billId":"1","pnl":"2","ts":"%d"}]}`, at.Add(-time.Hour).UnixMilli())
				assert.NoError(t, err, "bill should write")
			default:
				assert.Fail(t, "unexpected endpoint", "X-Perps funding should only use expected endpoints")
			}
		}))
		require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
		got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: at.Add(-2 * time.Hour), EndDate: at, IncludePayments: true})
		require.NoError(t, err, "X-Perps funding history must load")
		require.Len(t, got.FundingRates, 1, "one settled rate must load")
		assert.Equal(t, currency.USD, got.PaymentCurrency, "payment currency should come from the future's settlement currency")
		assert.Equal(t, "2", got.PaymentSum.String(), "the X-Perps funding bill should be retained")
	})
	for _, tc := range []struct {
		name       string
		repeatPage bool
	}{
		{name: "payments follow changing settlement intervals"},
		{name: "repeated bills cursor fails", repeatPage: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			require.NoError(t, ex.DisableRateLimiter(), "rate limiter must disable")
			ex.SkipAuthCheck = true
			ex.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
			now := timeNow().UTC().Truncate(time.Millisecond)
			var billCalls atomic.Int32
			server := httptest.NewTestServer(t, cappedHandler(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/public/funding-rate-history":
					rows := make([]string, 0, 4)
					// Deliberately unordered, spanning one-, two-, four- and eight-hour intervals.
					for _, hours := range []int{7, 1, 15, 3} {
						rows = append(rows, fmt.Sprintf(`{"fundingTime":"%d","realizedRate":"0.0001"}`, now.Add(-time.Duration(hours)*time.Hour).UnixMilli()))
					}
					_, err := fmt.Fprintf(w, `{"code":"0","data":[%s]}`, strings.Join(rows, ","))
					assert.NoError(t, err, "history should write")
				case "/public/funding-rate":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"fundingRate":"0.0001","ts":"%d","fundingTime":"%d"}]}`, now.UnixMilli(), now.Add(time.Hour).UnixMilli())
					assert.NoError(t, err, "current rate should write")
				case "/account/bills":
					calls := billCalls.Add(1)
					assert.Equal(t, perpetualSwapPair.String(), r.URL.Query().Get("instId"), "bills should be scoped to the requested instrument")
					assert.Equal(t, "USDT", r.URL.Query().Get("ccy"), "bills should be scoped to the payment currency")
					if calls > 2 {
						_, err := fmt.Fprint(w, `{"code":"1","msg":"unexpected additional page"}`)
						assert.NoError(t, err, "error should write")
						return
					}
					var rows []string
					for _, bill := range []struct {
						id, pnl string
						offset  time.Duration
					}{
						{"a", "-1", -time.Hour + time.Millisecond},
						{"b", "0.25", -time.Hour + time.Second},
						{"a", "-1", -time.Hour + time.Millisecond},
						{"c", "-2", -3*time.Hour + time.Minute},
						{"d", "-3", -7 * time.Hour},
						{"e", "4", -15*time.Hour + time.Second},
						{"before history", "100", -15*time.Hour - time.Millisecond},
						{"after end", "100", time.Millisecond},
						{"before start", "100", -17 * time.Hour},
					} {
						rows = append(rows, fmt.Sprintf(`{"billId":%q,"pnl":%q,"ts":"%d"}`, bill.id, bill.pnl, now.Add(bill.offset).UnixMilli()))
					}
					if tc.repeatPage {
						for len(rows) < 100 {
							rows = append(rows, rows[0])
						}
					}
					_, err := fmt.Fprintf(w, `{"code":"0","data":[%s]}`, strings.Join(rows, ","))
					assert.NoError(t, err, "bills should write")
				default:
					assert.Fail(t, "unexpected endpoint", "request should use funding or bills endpoints")
				}
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
			got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
				Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, PaymentCurrency: currency.USDT,
				StartDate: now.Add(-16 * time.Hour), EndDate: now, IncludePayments: true,
			})
			if tc.repeatPage {
				require.ErrorIs(t, err, common.ErrNoResponse, "a repeated cursor must fail instead of returning incomplete totals")
				assert.EqualValues(t, 2, billCalls.Load(), "pagination should stop on the repeated cursor")
				return
			}
			require.NoError(t, err, "payments must load")
			require.Len(t, got.FundingRates, 4, "all actual settlements must load")
			for i, want := range []string{"-0.75", "-2", "-3", "4"} {
				assert.Equal(t, want, got.FundingRates[i].Payment.String(), "each settlement should sum its distinct bills")
			}
			assert.Equal(t, "-1.75", got.PaymentSum.String(), "total should exclude duplicates and unmatched bills")
		})
	}
	t.Run("payments pagination", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

		var mu sync.Mutex
		var billsQueries []url.Values
		billsCalls := 0
		rateCalls := 0

		now := timeNow()
		start := now.Add(-24 * time.Hour)
		end := now.Add(-time.Hour)
		// Use an eight-hour settlement for this pagination fixture.
		fundingTime := now.Truncate(8 * time.Hour).Add(-8 * time.Hour)

		billRow := func(billID string, ts time.Time, pnl, instID string) string {
			return fmt.Sprintf(`{"billId":%q,"ccy":"USDT","type":"8","subType":"173","pnl":%q,"ts":"%d","instId":%q}`, billID, pnl, ts.UnixMilli(), instID)
		}
		billsPage := func(firstBillID string, count int, ts time.Time, pnl, instID string) string {
			bills := make([]string, 0, count)
			for i := range count {
				bills = append(bills, billRow(firstBillID+strconv.Itoa(i), ts, pnl, instID))
			}
			return `{"code":"0","msg":"","data":[` + strings.Join(bills, ",") + `]}`
		}

		srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/public/funding-rate-history":
				mu.Lock()
				rateCalls++
				calls := rateCalls
				mu.Unlock()
				if calls > 4 {
					// A short page ends the rates loop; refuse to feed the storm.
					_, _ = w.Write([]byte(`{"code":"1","msg":"mock: rate pagination did not stop after a short page"}`))
					return
				}
				_, _ = fmt.Fprintf(w, `{"code":"0","msg":"","data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","realizedRate":"0.0001","fundingTime":"%d"}]}`, fundingTime.UnixMilli())
			case "/public/funding-rate":
				_, _ = fmt.Fprintf(w, `{"code":"0","msg":"","data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","realizedRate":"0.0001","fundingTime":"%d","nextFundingRate":"0.0001","nextFundingTime":"%d"}]}`, fundingTime.UnixMilli(), fundingTime.Add(8*time.Hour).UnixMilli())
			case "/account/bills":
				mu.Lock()
				billsCalls++
				calls := billsCalls
				billsQueries = append(billsQueries, r.URL.Query())
				mu.Unlock()
				if calls > 4 {
					// The pagination failed to advance; refuse to feed the storm.
					_, _ = w.Write([]byte(`{"code":"1","msg":"mock: bills pagination did not advance past the first page"}`))
					return
				}
				// The bills endpoints scope by instrument: a sibling swap settling
				// funding in the same currency rides the unfiltered page and, being
				// last, would overwrite this pair's payment.
				sibling := r.URL.Query().Get("instId") == ""
				switch r.URL.Query().Get("after") {
				case "":
					if sibling {
						_, _ = w.Write([]byte(strings.TrimSuffix(billsPage("a", 99, fundingTime, "-1.5", "BTC-USDT-SWAP"), "]}") + "," + billRow("a99", fundingTime, "-150", "ETH-USDT-SWAP") + "]}"))
						return
					}
					_, _ = w.Write([]byte(billsPage("a", 100, fundingTime, "-1.5", "BTC-USDT-SWAP")))
				case "a99":
					// A short page strictly older than the cursor ends the loop.
					_, _ = w.Write([]byte(billsPage("b", 50, fundingTime.Add(-8*time.Hour), "0.5", "BTC-USDT-SWAP")))
				default:
					_, _ = w.Write([]byte(`{"code":"1","msg":"mock: unexpected bills cursor"}`))
				}
			default:
				_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
			}
		}))

		b := e.GetBase()
		b.SkipAuthCheck = true
		require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
		for k := range b.API.Endpoints.GetURLMap() {
			require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
		}
		e.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
		sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.PerpetualSwap, perpetualSwapPair)

		result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
			Asset:                asset.PerpetualSwap,
			Pair:                 perpetualSwapPair,
			PaymentCurrency:      currency.USDT,
			StartDate:            start,
			EndDate:              end,
			IncludePayments:      true,
			IncludePredictedRate: true,
		})
		require.NoError(t, err, "payments pagination must terminate without error")

		mu.Lock()
		queries := slices.Clone(billsQueries)
		mu.Unlock()
		require.Len(t, queries, 2, "the bills query must stop after the short page")
		// Both pages are pinned exactly: the documented filter names, the
		// requested window and this instrument's ID, with the second page keeping
		// the first page's filters and paging older via the last bill ID.
		firstPage := url.Values{
			"instType": {"SWAP"},
			"instId":   {perpetualSwapPair.String()},
			"ccy":      {"USDT"},
			"type":     {"8"},
			"begin":    {strconv.FormatInt(start.UnixMilli(), 10)},
			"end":      {strconv.FormatInt(end.UnixMilli(), 10)},
			"limit":    {"100"},
		}
		assert.Equal(t, firstPage, queries[0], "the first bills page should ask for this instrument's funding fee bills in the requested window")
		firstPage.Set("after", "a99")
		assert.Equal(t, firstPage, queries[1], "the second bills page should keep the first page's filters and page older via the last bill ID")

		require.Len(t, result.FundingRates, 1, "the funding rate history must keep its single entry")
		assert.Equal(t, "-150", result.FundingRates[0].Payment.String(), "the payment should sum all matching funding fee bills")
		assert.Equal(t, "-150", result.PaymentSum.String(), "the payment sum should only include in-window funding fee bills")
	})
	t.Run("rates pagination", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

		var mu sync.Mutex
		var rateQueries []url.Values
		rateCalls := 0
		billsCalls := 0

		end := timeNow().Truncate(8 * time.Hour)
		rateCount := 230
		rateTime := func(i int) time.Time {
			// 8h-aligned funding times, index 0 the newest, within the requested
			// window.
			return end.Add(-time.Duration(i+1) * 8 * time.Hour)
		}
		start := rateTime(rateCount - 1).Add(-8 * time.Hour)

		srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/public/funding-rate-history":
				mu.Lock()
				rateCalls++
				calls := rateCalls
				rateQueries = append(rateQueries, r.URL.Query())
				mu.Unlock()
				if calls > 4 {
					// The pagination failed to advance; refuse to feed the storm.
					_, _ = w.Write([]byte(`{"code":"1","msg":"mock: rate pagination did not advance past the first page"}`))
					return
				}
				q := r.URL.Query()
				before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
				after, _ := strconv.ParseInt(q.Get("after"), 10, 64)
				rows := make([]string, 0, 100)
				for i := range rateCount {
					ms := rateTime(i).UnixMilli()
					if before != 0 && ms <= before {
						continue // the endpoint returns records newer than before
					}
					if after != 0 && ms >= after {
						continue // the endpoint returns records earlier than after
					}
					if len(rows) == 100 {
						break
					}
					rows = append(rows, fmt.Sprintf(`{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","realizedRate":"0.0001","fundingTime":"%d"}`, ms))
				}
				if calls > 1 && after != 0 && len(rows) > 0 {
					// A boundary-inclusive server repeats the previous page's last
					// record; the wrapper must not duplicate the rate.
					keep := min(99, len(rows))
					rows = append([]string{fmt.Sprintf(`{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","realizedRate":"0.0001","fundingTime":"%d"}`, after)}, rows[:keep]...)
				}
				_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[` + strings.Join(rows, ",") + `]}`))
			case "/public/funding-rate":
				_, _ = fmt.Fprintf(w, `{"code":"0","msg":"","data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","realizedRate":"0.0001","fundingTime":"%d","nextFundingRate":"0.0001","nextFundingTime":"%d"}]}`, end.Add(7*time.Hour).UnixMilli(), end.Add(15*time.Hour).UnixMilli())
			case "/account/bills-archive":
				mu.Lock()
				billsCalls++
				calls := billsCalls
				mu.Unlock()
				if calls > 1 {
					// A short page ends the bills loop; refuse to feed the storm.
					_, _ = w.Write([]byte(`{"code":"1","msg":"mock: bills pagination did not stop after a short page"}`))
					return
				}
				// Funding fee bills, newest first, for the first, an inner and the
				// last rate of the pages; 99 and 198 are also the boundary records
				// the next page repeats
				bills := make([]string, 0, 5)
				for i, rate := range []int{0, 99, 150, 198, rateCount - 1} {
					bills = append(bills, fmt.Sprintf(`{"billId":"%d","ccy":"USDT","type":"8","pnl":"-%d","ts":"%d","instId":"BTC-USDT-SWAP"}`, 5-i, i+1, rateTime(rate).UnixMilli()))
				}
				_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[` + strings.Join(bills, ",") + `]}`))
			default:
				_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
			}
		}))

		b := e.GetBase()
		b.SkipAuthCheck = true
		require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
		for k := range b.API.Endpoints.GetURLMap() {
			require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
		}

		result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
			Asset:           asset.PerpetualSwap,
			Pair:            perpetualSwapPair,
			StartDate:       start,
			EndDate:         end,
			IncludePayments: false,
		})
		require.NoError(t, err, "rate pagination must terminate without error")

		mu.Lock()
		queries := slices.Clone(rateQueries)
		mu.Unlock()
		require.Len(t, queries, 3, "230 rates at 100 per page must take three requests")
		for i, after := range []string{
			strconv.FormatInt(end.UnixMilli(), 10),
			strconv.FormatInt(rateTime(99).UnixMilli(), 10),
			strconv.FormatInt(rateTime(198).UnixMilli(), 10),
		} {
			assert.Equal(t, after, queries[i].Get("after"), "each page should cursor after the previous page's oldest funding time")
			assert.Equal(t, strconv.FormatInt(start.UnixMilli(), 10), queries[i].Get("before"), "every page should keep the requested window floor")
			assert.Equal(t, "100", queries[i].Get("limit"), "every page should carry the request limit")
		}

		require.Len(t, result.FundingRates, rateCount, "every funding rate in the window must be kept exactly once")
		seen := make(common.Seen[int64], rateCount)
		for i := range result.FundingRates {
			ms := result.FundingRates[i].Time.UnixMilli()
			require.False(t, seen.CheckAndRecord(ms), "a repeated boundary record must not duplicate a rate")
		}
		assert.Equal(t, rateTime(0).UnixMilli(), result.FundingRates[0].Time.UnixMilli(), "the newest rate should come first")
		assert.Equal(t, rateTime(rateCount-1).UnixMilli(), result.FundingRates[rateCount-1].Time.UnixMilli(), "the oldest rate should be reached, not stranded behind repeated pages")

		// Payments index rates across pages, so repeat the run with them; the
		// first run stays as it is to cover the no-payments path
		mu.Lock()
		rateCalls = 0
		mu.Unlock()
		result, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
			Asset:           asset.PerpetualSwap,
			Pair:            perpetualSwapPair,
			PaymentCurrency: currency.USDT,
			StartDate:       start,
			EndDate:         end,
			IncludePayments: true,
		})
		require.NoError(t, err, "rate pagination with payments must terminate without error")
		require.Len(t, result.FundingRates, rateCount, "every funding rate in the window must be kept exactly once with payments")
		paid := make(map[int]string)
		for i := range result.FundingRates {
			if !result.FundingRates[i].Payment.IsZero() {
				paid[i] = result.FundingRates[i].Payment.String()
			}
		}
		assert.Equal(t, map[int]string{0: "-1", 99: "-2", 150: "-3", 198: "-4", rateCount - 1: "-5"}, paid, "each payment should land on its own rate, whichever page it arrived on")
	})
	t.Run("judges its windows by the wrapper's clock", func(t *testing.T) {
		t.Parallel()
		week := kline.OneWeek.Duration()
		for _, tc := range []struct {
			name     string
			age      time.Duration
			payments bool
			err      error
			bills    map[string]int
		}{
			{name: "a start a minute past the funding history limit", age: kline.ThreeMonth.Duration() + time.Minute, err: fundingrate.ErrFundingRateOutsideLimits, bills: map[string]int{}},
			{name: "a start a minute inside the funding history limit", age: kline.ThreeMonth.Duration() - time.Minute, bills: map[string]int{}},
			{name: "payments from a minute inside a week read the 7 day bills", age: week - time.Minute, payments: true, bills: map[string]int{"/account/bills": 1}},
			{name: "payments from a minute past a week read the bills archive", age: week + time.Minute, payments: true, bills: map[string]int{"/account/bills-archive": 1}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				now := timeNow()
				var mu sync.Mutex
				bills := make(map[string]int)
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/funding-rate-history":
						writeOKXData(t, w, []any{})
					case "/public/funding-rate":
						writeOKXData(t, w, []map[string]string{{"instId": perpetualSwapPair.String(), "fundingRate": "0.0001", "ts": strconv.FormatInt(now.UnixMilli(), 10), "fundingTime": strconv.FormatInt(now.Add(time.Hour).UnixMilli(), 10)}})
					case "/account/bills", "/account/bills-archive":
						mu.Lock()
						bills[r.URL.Path]++
						mu.Unlock()
						writeOKXData(t, w, []any{})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				ex.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
				_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
					Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, PaymentCurrency: currency.USDT,
					StartDate: now.Add(-tc.age), EndDate: now, IncludePayments: tc.payments,
				})
				assert.ErrorIs(t, err, tc.err, "GetHistoricalFundingRates should judge the start by the wrapper's clock")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.bills, bills, "GetHistoricalFundingRates should choose the bills listing by the wrapper's clock")
			})
		}
	})

	t.Run("zero end date", func(t *testing.T) {
		t.Parallel()
		for _, includePayments := range []bool{false, true} {
			t.Run(fmt.Sprintf("payments-%v", includePayments), func(t *testing.T) {
				t.Parallel()
				e := new(Exchange)
				require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

				var mu sync.Mutex
				rateCalls := 0
				billsCalls := 0
				var ratesAfter, billsEnd string

				now := timeNow()
				start := now.Add(-24 * time.Hour)
				fundingTime := now.Truncate(8 * time.Hour).Add(-8 * time.Hour)
				// The first page a before-only query returns: an old rate, not the
				// newest one.
				truncatedTime := start.Add(2 * time.Hour)

				srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/public/funding-rate-history":
						mu.Lock()
						rateCalls++
						calls := rateCalls
						if calls == 1 {
							ratesAfter = r.URL.Query().Get("after")
						}
						mu.Unlock()
						if calls > 4 {
							// A short page ends the rates loop; refuse to feed the storm.
							_, _ = w.Write([]byte(`{"code":"1","msg":"mock: rate pagination did not stop after a short page"}`))
							return
						}
						if r.URL.Query().Get("after") == "" {
							_, _ = fmt.Fprintf(w, `{"code":"0","msg":"","data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0002","realizedRate":"0.0002","fundingTime":"%d"}]}`, truncatedTime.UnixMilli())
							return
						}
						_, _ = fmt.Fprintf(w, `{"code":"0","msg":"","data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","realizedRate":"0.0001","fundingTime":"%d"}]}`, fundingTime.UnixMilli())
					case "/public/funding-rate":
						_, _ = fmt.Fprintf(w, `{"code":"0","msg":"","data":[{"instType":"SWAP","instId":"BTC-USDT-SWAP","fundingRate":"0.0001","realizedRate":"0.0001","fundingTime":"%d","nextFundingRate":"0.0001","nextFundingTime":"%d"}]}`, fundingTime.UnixMilli(), fundingTime.Add(8*time.Hour).UnixMilli())
					case "/account/bills":
						mu.Lock()
						billsCalls++
						calls := billsCalls
						billsEnd = r.URL.Query().Get("end")
						mu.Unlock()
						if calls > 4 {
							// A short page ends the bills loop; refuse to feed the storm.
							_, _ = w.Write([]byte(`{"code":"1","msg":"mock: bills pagination did not stop after a short page"}`))
							return
						}
						if r.URL.Query().Get("after") != "" {
							_, _ = w.Write([]byte(`{"code":"1","msg":"mock: unexpected bills cursor"}`))
							return
						}
						_, _ = fmt.Fprintf(w, `{"code":"0","msg":"","data":[{"billId":"a0","ccy":"USDT","type":"8","subType":"173","pnl":"-1.5","ts":"%d","instId":"BTC-USDT-SWAP"}]}`, fundingTime.UnixMilli())
					default:
						_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
					}
				}))

				b := e.GetBase()
				b.SkipAuthCheck = true
				require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
				for k := range b.API.Endpoints.GetURLMap() {
					require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
				}
				e.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
				sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.PerpetualSwap, perpetualSwapPair)

				called := timeNow()
				result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
					Asset:                asset.PerpetualSwap,
					Pair:                 perpetualSwapPair,
					PaymentCurrency:      currency.USDT,
					StartDate:            start,
					IncludePayments:      includePayments,
					IncludePredictedRate: true,
				})
				returned := timeNow()
				require.NoError(t, err, "an unset end date must be read as open-ended, not error")

				require.Len(t, result.FundingRates, 1, "the newest page must be returned")
				assert.Equal(t, fundingTime.UnixMilli(), result.FundingRates[0].Time.UnixMilli(), "the newest rate should come back, not the old page a before-only query returns")

				mu.Lock()
				bills, after, end := billsCalls, ratesAfter, billsEnd
				mu.Unlock()
				afterMS, err := strconv.ParseInt(after, 10, 64)
				require.NoError(t, err, "the first rates query must page from an after cursor")
				assert.WithinRange(t, time.UnixMilli(afterMS), called.Truncate(time.Millisecond), returned, "an unset end should read as the time of the call")
				if includePayments {
					require.Equal(t, 1, bills, "the bills loop must page against the same effective window")
					assert.Equal(t, after, end, "the bills query should end where the rates query does")
					assert.Equal(t, "-1.5", result.FundingRates[0].Payment.String(), "the payment should come from the matching funding fee bill")
				} else {
					assert.Zero(t, bills, "no bills request should be made without payments")
				}
			})
		}
	})
	for _, tc := range []struct {
		name                               string
		startMinutes, wantRates, wantCalls int
		wantErr                            error
	}{
		{name: "repeated full page", startMinutes: 200, wantCalls: 2, wantErr: common.ErrNoResponse},
		{name: "newer full page", startMinutes: 200, wantCalls: 2, wantErr: common.ErrNoResponse},
		{name: "full page reaches start", startMinutes: 100, wantRates: 100, wantCalls: 1},
		{name: "full page crosses start", startMinutes: 90, wantRates: 90, wantCalls: 1},
		{name: "short repeated boundary", startMinutes: 200, wantRates: 100, wantCalls: 2},
		{name: "empty page", startMinutes: 200, wantCalls: 1},
	} {
		t.Run("history cursor "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			require.NoError(t, ex.DisableRateLimiter(), "rate limiter must disable")
			end := timeNow().UTC().Truncate(time.Millisecond)
			var mu sync.Mutex
			calls := 0
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/public/funding-rate" {
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"fundingRate":"0","ts":"%d","fundingTime":"%d"}]}`, end.UnixMilli(), end.Add(time.Hour).UnixMilli())
					assert.NoError(t, err, "latest rate should write")
					return
				}
				assert.Equal(t, "/public/funding-rate-history", r.URL.Path, "request should retrieve history")
				mu.Lock()
				calls++
				count := calls
				mu.Unlock()
				if count > 2 {
					_, err := w.Write([]byte(`{"code":"1","msg":"unexpected third history request"}`))
					assert.NoError(t, err, "loop protection response should write")
					return
				}
				var rows []string
				switch {
				case tc.name == "empty page":
				case tc.name == "short repeated boundary" && count == 2:
					rows = append(rows, fmt.Sprintf(`{"realizedRate":"0.001","fundingTime":"%d"}`, end.Add(-100*time.Minute).UnixMilli()))
				default:
					shift := 0
					if tc.name == "newer full page" && count == 2 {
						shift = 10
					}
					for i := 1; i <= 100; i++ {
						rows = append(rows, fmt.Sprintf(`{"realizedRate":"0.001","fundingTime":"%d"}`, end.Add(time.Duration(shift-i)*time.Minute).UnixMilli()))
					}
				}
				_, err := fmt.Fprintf(w, `{"code":"0","data":[%s]}`, strings.Join(rows, ","))
				assert.NoError(t, err, "history page should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "in-memory HTTP client must initialise")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must use the test server")
			got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, StartDate: end.Add(-time.Duration(tc.startMinutes) * time.Minute), EndDate: end})
			require.ErrorIs(t, err, tc.wantErr, "history pagination must return the expected outcome")
			mu.Lock()
			count := calls
			mu.Unlock()
			assert.Equal(t, tc.wantCalls, count, "history pagination should stop at the expected page")
			if tc.wantErr != nil {
				assert.Nil(t, got, "incomplete history should not be returned as success")
				return
			}
			require.NotNil(t, got, "valid history must return a result")
			assert.Len(t, got.FundingRates, tc.wantRates, "history should retain only unique in-window rates")
		})
	}
	for _, tc := range []struct {
		name, id string
		outside  bool
		wantErr  error
	}{
		{name: "missing", wantErr: common.ErrNoResponse},
		{name: "empty", id: `,"billId":""`, wantErr: common.ErrNoResponse},
		{name: "null", id: `,"billId":null`, wantErr: common.ErrNoResponse},
		{name: "valid duplicate", id: `,"billId":"one"`},
		{name: "out-of-window missing", outside: true},
	} {
		t.Run("bill ID "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			require.NoError(t, ex.DisableRateLimiter(), "rate limiter must disable")
			ex.SkipAuthCheck = true
			ex.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
			end := timeNow().UTC().Truncate(time.Millisecond)
			settlement := end.Add(-time.Hour)
			server := httptest.NewTestServer(t, cappedHandler(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/public/funding-rate-history":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"realizedRate":"0.001","fundingTime":"%d"}]}`, settlement.UnixMilli())
					assert.NoError(t, err, "history should write")
				case "/public/funding-rate":
					_, err := fmt.Fprintf(w, `{"code":"0","data":[{"fundingRate":"0","ts":"%d","fundingTime":"%d"}]}`, end.UnixMilli(), end.Add(time.Hour).UnixMilli())
					assert.NoError(t, err, "latest rate should write")
				case "/account/bills":
					timestamp := settlement
					if tc.outside {
						timestamp = end.Add(-3 * time.Hour)
					}
					row := fmt.Sprintf(`{"pnl":"2","ts":"%d"%s}`, timestamp.UnixMilli(), tc.id)
					_, err := fmt.Fprintf(w, `{"code":"0","data":[%s,%s]}`, row, row)
					assert.NoError(t, err, "bills should write")
				default:
					assert.Fail(t, "unexpected endpoint", "payment lookup should use expected endpoints")
				}
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "in-memory HTTP client must initialise")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must use the test server")
			got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, PaymentCurrency: currency.USDT, StartDate: end.Add(-2 * time.Hour), EndDate: end, IncludePayments: true})
			require.ErrorIs(t, err, tc.wantErr, "bills must return the expected validation outcome")
			if tc.wantErr != nil {
				assert.Nil(t, got, "unidentifiable in-window bills should not return partial payments")
				return
			}
			want := "2"
			if tc.outside {
				want = "0"
			}
			assert.Equal(t, want, got.PaymentSum.String(), "valid duplicate bills should count once and out-of-window bills should be ignored")
		})
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Binary, mainPair)
	require.NoError(t, err)
	require.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.PerpetualSwap, perpetualSwapPair)
	require.NoError(t, err)
	assert.True(t, is, "expected true")
}

func TestGetAssetsFromInstrumentTypeOrID(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup must not error")

	_, err := e.getAssetsFromInstrumentID("")
	assert.ErrorIs(t, err, errMissingInstrumentID)

	for _, a := range []asset.Item{asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options} {
		assets, err2 := e.getAssetsFromInstrumentID(e.CurrencyPairs.Pairs[a].Enabled[0].String())
		require.NoErrorf(t, err2, "GetAssetsFromInstrumentTypeOrID must not error for asset: %s", a)
		switch a {
		case asset.Spot, asset.Margin:
			// spot and margin instruments are similar
			require.Len(t, assets, 2)
		default:
			require.Len(t, assets, 1)
		}
		assert.Containsf(t, assets, a, "Should contain asset: %s", a)
	}

	_, err = e.getAssetsFromInstrumentID("test")
	assert.ErrorIs(t, err, currency.ErrCurrencyNotSupported)
	_, err = e.getAssetsFromInstrumentID("test-test")
	assert.ErrorIs(t, err, asset.ErrNotEnabled)

	for _, a := range []asset.Item{asset.Margin, asset.Spot} {
		assets, err2 := e.getAssetsFromInstrumentID(e.CurrencyPairs.Pairs[a].Enabled[0].String())
		require.NoErrorf(t, err2, "GetAssetsFromInstrumentTypeOrID must not error for asset: %s", a)
		assert.Contains(t, assets, a)
	}
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	err := e.SetMarginType(contextGenerate(), asset.Spot, mainPair, margin.Isolated)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()

	t.Run("changes the position margin", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.ChangePositionMargin(contextGenerate(), &margin.PositionChangeRequest{
			Pair:                    mainPair,
			Asset:                   asset.Margin,
			MarginType:              margin.Isolated,
			OriginalAllocatedMargin: 0.2,
			NewAllocatedMargin:      0.5,
		})
		require.NoError(t, err, "ChangePositionMargin must not error")
		if mockTests {
			exp := &margin.PositionChangeResponse{Exchange: e.Name, Pair: mainPair, Asset: asset.Margin, AllocatedMargin: 0.5, MarginType: margin.Isolated}
			assert.Equal(t, exp, result, "ChangePositionMargin should report the new allocated margin")
		} else {
			assert.NotNil(t, result, "ChangePositionMargin should return the margin change")
		}
	})

	isolatedPosition := func(instID, posSide, mgnMode, ccy string) map[string]string {
		return map[string]string{"instType": instTypeMargin, "instId": instID, "posSide": posSide, "mgnMode": mgnMode, "ccy": ccy, "margin": "4", "pos": "0.0002", "posCcy": "BTC"}
	}
	// closedPosition is an isolated net BTC-USDT position holding no size, with the margin and debt given
	closedPosition := func(ccy, marginHeld, liabilities string) map[string]string {
		return map[string]string{"instType": instTypeMargin, "instId": "BTC-USDT", "posSide": positionSideNet, "mgnMode": TradeModeIsolated, "ccy": ccy, "margin": marginHeld, "pos": "0", "liab": liabilities, "posCcy": "USDT"}
	}
	// newPositionMarginExchange answers an isolated margin position lookup with positions, failing the test on a lookup
	// when positions is nil, and checks an adjustment carries the JSON body expRequest, failing the test on an
	// adjustment when expRequest is empty
	newPositionMarginExchange := func(t *testing.T, positions []map[string]string, expRequest, expType string) *Exchange {
		t.Helper()
		return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/account/positions":
				if !assert.NotNil(t, positions, "only an isolated margin position's adjustment should look the position up") {
					http.NotFound(w, r)
					return
				}
				assert.Equal(t, url.Values{"instType": {instTypeMargin}, "instId": {"BTC-USDT"}}, r.URL.Query(), "the lookup should name the position's instrument")
				writeOKXData(t, w, positions)
			case "/account/position/margin-balance":
				if !assert.NotEmpty(t, expRequest, "a refused adjustment should not reach OKX") {
					http.NotFound(w, r)
					return
				}
				assert.Equal(t, http.MethodPost, r.Method, "the adjustment should be a POST")
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				assert.JSONEq(t, expRequest, string(body), "the adjustment should carry the change in the position's margin currency")
				writeOKXData(t, w, []map[string]string{{"type": expType}})
			default:
				assert.Failf(t, "unexpected request", "a margin change should not reach %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
	}

	t.Run("sends the exact adjustment and reports the new allocated margin", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name                string
			req                 *margin.PositionChangeRequest
			positions           []map[string]string
			expRequest, expType string
		}{
			{
				name:       "margin increase",
				req:        &margin.PositionChangeRequest{Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: 4.0695, NewAllocatedMargin: 5},
				positions:  []map[string]string{isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "BTC")},
				expRequest: `{"instId":"BTC-USDT","posSide":"net","type":"add","amt":"0.9305","ccy":"BTC"}`,
				expType:    "add",
			},
			{
				name:       "margin decrease",
				req:        &margin.PositionChangeRequest{Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: 5, NewAllocatedMargin: 4.0695},
				positions:  []map[string]string{isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "BTC")},
				expRequest: `{"instId":"BTC-USDT","posSide":"net","type":"reduce","amt":"0.9305","ccy":"BTC"}`,
				expType:    "reduce",
			},
			{
				name:       "perpetual swap long position",
				req:        &margin.PositionChangeRequest{Pair: perpetualSwapPair, Asset: asset.PerpetualSwap, MarginType: margin.Isolated, MarginSide: positionSideLong, OriginalAllocatedMargin: 0.1, NewAllocatedMargin: 0.3},
				expRequest: `{"instId":"BTC-USDT-SWAP","posSide":"long","type":"add","amt":"0.2"}`,
				expType:    "add",
			},
			{
				name:       "margin held in the quote currency",
				req:        &margin.PositionChangeRequest{Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: 4, NewAllocatedMargin: 5},
				positions:  []map[string]string{isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "USDT")},
				expRequest: `{"instId":"BTC-USDT","posSide":"net","type":"add","amt":"1","ccy":"USDT"}`,
				expType:    "add",
			},
			{
				name: "the isolated position on the requested side",
				req:  &margin.PositionChangeRequest{Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, MarginSide: positionSideLong, OriginalAllocatedMargin: 4, NewAllocatedMargin: 3},
				positions: []map[string]string{
					isolatedPosition("BTC-USDT", positionSideLong, TradeModeCross, "BTC"),
					isolatedPosition("ETH-USDT", positionSideLong, TradeModeIsolated, "ETH"),
					isolatedPosition("BTC-USDT", positionSideShort, TradeModeIsolated, "BTC"),
					isolatedPosition("BTC-USDT", positionSideLong, TradeModeIsolated, "USDT"),
				},
				expRequest: `{"instId":"BTC-USDT","posSide":"long","type":"reduce","amt":"1","ccy":"USDT"}`,
				expType:    "reduce",
			},
			{
				name:       "a closed position beside the funded one",
				req:        &margin.PositionChangeRequest{Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: 4, NewAllocatedMargin: 5},
				positions:  []map[string]string{isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "USDT"), closedPosition("BTC", "0", "0")},
				expRequest: `{"instId":"BTC-USDT","posSide":"net","type":"add","amt":"1","ccy":"USDT"}`,
				expType:    "add",
			},
			{
				name:       "a closed position listed before the funded one",
				req:        &margin.PositionChangeRequest{Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: 4, NewAllocatedMargin: 5},
				positions:  []map[string]string{closedPosition("BTC", "0", "0"), isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "USDT")},
				expRequest: `{"instId":"BTC-USDT","posSide":"net","type":"add","amt":"1","ccy":"USDT"}`,
				expType:    "add",
			},
			{
				name:       "margin transferred in before the position opened",
				req:        &margin.PositionChangeRequest{Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: 4, NewAllocatedMargin: 5},
				positions:  []map[string]string{closedPosition("USDT", "4", "0")},
				expRequest: `{"instId":"BTC-USDT","posSide":"net","type":"add","amt":"1","ccy":"USDT"}`,
				expType:    "add",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newPositionMarginExchange(t, tc.positions, tc.expRequest, tc.expType)
				got, err := ex.ChangePositionMargin(t.Context(), tc.req)
				require.NoError(t, err, "ChangePositionMargin must not error")
				exp := &margin.PositionChangeResponse{Exchange: ex.Name, Pair: tc.req.Pair, Asset: tc.req.Asset, AllocatedMargin: tc.req.NewAllocatedMargin, MarginType: margin.Isolated}
				assert.Equal(t, exp, got, "ChangePositionMargin should report the new allocated margin")
			})
		}
	})

	t.Run("refuses an isolated margin position whose margin currency it cannot tell", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			positions []map[string]string
			expErr    error
		}{
			{name: "no position", positions: []map[string]string{}, expErr: futures.ErrPositionNotFound},
			{
				name:      "positions on other sides and margin modes only",
				positions: []map[string]string{isolatedPosition("BTC-USDT", positionSideLong, TradeModeIsolated, "BTC"), isolatedPosition("BTC-USDT", positionSideNet, TradeModeCross, "USDT")},
				expErr:    futures.ErrPositionNotFound,
			},
			{
				name:      "positions holding margin in either currency",
				positions: []map[string]string{isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "BTC"), isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "USDT")},
				expErr:    errAmbiguousPosition,
			},
			{
				name:      "a position without size holding margin in the other currency",
				positions: []map[string]string{isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "USDT"), closedPosition("BTC", "4", "0")},
				expErr:    errAmbiguousPosition,
			},
			{
				name:      "a position without size owing in the other currency",
				positions: []map[string]string{isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "USDT"), closedPosition("BTC", "0", "1")},
				expErr:    errAmbiguousPosition,
			},
			{
				name: "a position with size but no margin in the other currency",
				positions: []map[string]string{
					isolatedPosition("BTC-USDT", positionSideNet, TradeModeIsolated, "USDT"),
					{"instType": instTypeMargin, "instId": "BTC-USDT", "posSide": positionSideNet, "mgnMode": TradeModeIsolated, "ccy": "BTC", "margin": "0", "pos": "0.0002", "posCcy": "BTC"},
				},
				expErr: errAmbiguousPosition,
			},
			{name: "a closed position alone", positions: []map[string]string{closedPosition("BTC", "0", "0")}, expErr: futures.ErrPositionNotFound},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newPositionMarginExchange(t, tc.positions, "", "")
				_, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
					Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: 4, NewAllocatedMargin: 5,
				})
				assert.ErrorIs(t, err, tc.expErr, "ChangePositionMargin should refuse an adjustment it cannot name the margin currency of")
			})
		}
	})

	t.Run("refuses an asset that holds no isolated positions", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			a    asset.Item
			pair currency.Pair
		}{
			{asset.Spot, mainPair},
			{asset.Spread, spreadPair},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				ex := newPositionMarginExchange(t, nil, "", "")
				_, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
					Pair: tc.pair, Asset: tc.a, MarginType: margin.Isolated, OriginalAllocatedMargin: 1, NewAllocatedMargin: 2,
				})
				assert.ErrorIs(t, err, asset.ErrNotSupported, "ChangePositionMargin should refuse an asset without isolated positions")
			})
		}
	})

	t.Run("refuses a margin that is not a positive finite number", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name                string
			original, newMargin float64
			expErr              error
		}{
			{"new margin not a number", 1, math.NaN(), margin.ErrNewAllocatedMarginRequired},
			{"new margin infinite", 1, math.Inf(1), margin.ErrNewAllocatedMarginRequired},
			{"original margin not a number", math.NaN(), 1, margin.ErrOriginalPositionMarginRequired},
			{"original margin infinite", math.Inf(-1), 1, margin.ErrOriginalPositionMarginRequired},
			{"new margin zero", 1, 0, margin.ErrNewAllocatedMarginRequired},
			{"new margin negative", 1, -2, margin.ErrNewAllocatedMarginRequired},
			{"original margin zero", 0, 2, margin.ErrOriginalPositionMarginRequired},
			{"original margin negative", -1, 2, margin.ErrOriginalPositionMarginRequired},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("a refused margin change should not reach OKX: %s", r.URL.Path)
					http.NotFound(w, r)
				}))
				_, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{
					Pair: mainPair, Asset: asset.Margin, MarginType: margin.Isolated, OriginalAllocatedMargin: tc.original, NewAllocatedMargin: tc.newMargin,
				})
				assert.ErrorIs(t, err, tc.expErr, "ChangePositionMargin should refuse a margin it cannot express")
			})
		}
	})
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateralMode(contextGenerate(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	t.Run("rejects a malformed account configuration reply", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, data string
		}{
			{"null", "null"},
			{"null row", "[null]"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/account/config", r.URL.Path, "the collateral mode should come from the account configuration")
					_, err := w.Write([]byte(`{"code":"0","msg":"","data":` + tc.data + `}`))
					assert.NoError(t, err, "writing the reply should not error")
				}))
				var err error
				require.NotPanics(t, func() { _, err = ex.GetCollateralMode(t.Context(), asset.Spot) }, "GetCollateralMode must not panic on a malformed reply")
				assert.ErrorIs(t, err, common.ErrNoResponse, "GetCollateralMode should reject a success reply without a configuration")
			})
		}
	})

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetCollateralMode(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := e.SetCollateralMode(contextGenerate(), asset.Spot, collateral.SingleMode)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

// listedFuturesContract returns mockFuturesContract in mock mode, and otherwise an enabled contract of the listed ones,
// which the configured contract, having expired, is not
func listedFuturesContract(t *testing.T) currency.Pair {
	t.Helper()
	if mockTests {
		return mockFuturesContract
	}
	testexch.UpdatePairsOnce(t, e)
	pairs, err := e.GetEnabledPairs(asset.Futures)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.NotEmpty(t, pairs, "GetEnabledPairs must return a futures contract")
	return pairs[0]
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           mockFuturesContract,
		UnderlyingPair: mainPair,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesPositionSummary must reject a spot asset")

	swapPair := btcUSDSwapPair
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		pp, err := e.CurrencyPairs.GetPairs(asset.PerpetualSwap, true)
		require.NoError(t, err, "GetPairs must not error")
		swapPair = pp[0]
	}
	result, err := e.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.PerpetualSwap,
		Pair:           swapPair,
		UnderlyingPair: currency.EMPTYPAIR,
	})
	require.NoError(t, err, "GetFuturesPositionSummary must not error for a perpetual swap")
	dec := decimal.MustFromFloat
	if mockTests {
		exp := &futures.PositionSummary{
			Pair:                         swapPair,
			Asset:                        asset.PerpetualSwap,
			MarginType:                   margin.Isolated,
			CollateralMode:               collateral.SpotFuturesMode,
			Currency:                     currency.BTC,
			StartDate:                    time.UnixMilli(1795320000000),
			AvailableEquity:              dec(0.7408),
			CashBalance:                  dec(0.8012),
			DiscountEquity:               dec(49102.35),
			EquityUSD:                    dec(49352.1),
			IsolatedEquity:               dec(0.0021),
			IsolatedLiabilities:          dec(0.0003),
			IsolatedUPL:                  dec(-0.0002),
			NotionalLeverage:             dec(0.6),
			TotalEquity:                  dec(55837.43556134779),
			StrategyEquity:               dec(0.02),
			IsolatedMargin:               dec(0.00082987),
			NotionalSize:                 dec(500),
			Leverage:                     dec(10),
			MaintenanceMarginRequirement: dec(0.00003249),
			InitialMarginRequirement:     dec(0),
			EstimatedLiquidationPrice:    dec(54466.2),
			CollateralUsed:               dec(0.00082987),
			MarkPrice:                    dec(61552.1),
			CurrentSize:                  dec(500),
			ContractSize:                 dec(5),
			ContractMultiplier:           dec(100),
			ContractSettlementType:       futures.Inverse,
			AverageOpenPrice:             dec(60250.5),
			UnrealisedPNL:                dec(0.00017549),
			RealisedPNL:                  dec(-0.00000427),
			MaintenanceMarginFraction:    dec(30.94),
			FreeCollateral:               dec(0.7512),
			TotalCollateral:              dec(0.8012),
			FrozenBalance:                dec(0.05),
			EquityOfCurrency:             dec(0.8034),
		}
		assert.Equal(t, exp, result, "GetFuturesPositionSummary should map the perpetual swap position and its currency's balance")
	} else {
		assert.NotNil(t, result, "GetFuturesPositionSummary should return the perpetual swap position")
	}

	contract := listedFuturesContract(t)
	result, err = e.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Futures,
		Pair:           contract,
		UnderlyingPair: currency.EMPTYPAIR,
	})
	require.NoError(t, err, "GetFuturesPositionSummary must not error for a futures contract")
	if mockTests {
		// A long/short mode short reports a positive size, which the summary negates
		exp := &futures.PositionSummary{
			Pair:                         contract,
			Asset:                        asset.Futures,
			MarginType:                   margin.Multi,
			CollateralMode:               collateral.SpotFuturesMode,
			Currency:                     currency.USDC,
			StartDate:                    time.UnixMilli(1795330000000),
			AvailableEquity:              dec(1480.5),
			CashBalance:                  dec(1520.25),
			DiscountEquity:               dec(1519.9),
			EquityUSD:                    dec(1521.6),
			IsolatedEquity:               dec(0),
			IsolatedLiabilities:          dec(0),
			IsolatedUPL:                  dec(0),
			NotionalLeverage:             dec(10.02),
			TotalEquity:                  dec(55837.43556134779),
			StrategyEquity:               dec(5),
			IsolatedMargin:               dec(0),
			NotionalSize:                 dec(15255.15),
			Leverage:                     dec(20),
			MaintenanceMarginRequirement: dec(61.0206),
			InitialMarginRequirement:     dec(762.7575),
			EstimatedLiquidationPrice:    dec(88412.7),
			CollateralUsed:               dec(0),
			MarkPrice:                    dec(61020.6),
			CurrentSize:                  dec(-0.25),
			ContractSize:                 dec(-25),
			ContractMultiplier:           dec(0.01),
			ContractSettlementType:       futures.Linear,
			AverageOpenPrice:             dec(60310.4),
			UnrealisedPNL:                dec(-177.55),
			RealisedPNL:                  dec(-0.7539),
			MaintenanceMarginFraction:    dec(0),
			FreeCollateral:               dec(1500.25),
			TotalCollateral:              dec(1520.25),
			FrozenBalance:                dec(20),
			EquityOfCurrency:             dec(1521.75),
		}
		assert.Equal(t, exp, result, "GetFuturesPositionSummary should map the futures position and its currency's balance")
	} else {
		assert.NotNil(t, result, "GetFuturesPositionSummary should return the futures position")
	}
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
		Asset:     asset.Spot,
		Pairs:     []currency.Pair{mockFuturesContract},
		StartDate: time.Now().Add(time.Hour * 24 * -7),
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	end := time.UnixMilli(1795410000000)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		end = time.Now()
	}
	contract := listedFuturesContract(t)
	result, err := e.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
		Asset:     asset.Futures,
		Pairs:     []currency.Pair{contract},
		StartDate: end.Add(time.Hour * 24 * -7),
		EndDate:   end,
	})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	if mockTests {
		// Sizes are contracts of 0.01 BTC each, and the linear contract's cost is in its settlement currency. The
		// recorded cancelled order 1752810569801498801 executed nothing, so it made up no position
		exp := []futures.PositionResponse{{
			Pair:                   contract,
			Asset:                  asset.Futures,
			ContractSettlementType: futures.Linear,
			Orders: []order.Detail{
				{
					Price: 60310.4, AverageExecutedPrice: 60310.4, Amount: 0.25, ContractAmount: 25, ExecutedAmount: 0.25, Fee: 0.7539,
					FeeAsset: currency.USDC, Exchange: e.Name, OrderID: "1752810569801498800", ClientOrderID: "fut1", Type: order.Limit,
					Side: order.Sell, Status: order.Filled, AssetType: asset.Futures, Date: time.UnixMilli(1795330000000),
					LastUpdated: time.UnixMilli(1795330000123), Pair: contract, Cost: 15077.6, CostAsset: currency.USD,
				},
			},
		}}
		assert.Equal(t, exp, result, "GetFuturesPositionOrders should map the contract's orders")
	} else {
		assert.NotNil(t, result, "GetFuturesPositionOrders should return the contract's orders")
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(contextGenerate(), asset.Options, mockFuturesContract, margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.GetLeverage(contextGenerate(), asset.Futures, mockFuturesContract, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		_, err = e.GetLeverage(contextGenerate(), asset.Futures, mockFuturesContract, margin.Isolated, order.UnknownSide)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "without credentials the fetch must fail before any request is sent, as with them it would read the live account")
	}

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		contract := listedFuturesContract(t)
		result, err := e.GetLeverage(contextGenerate(), asset.Futures, contract, margin.Multi, order.UnknownSide)
		require.NoError(t, err, "GetLeverage must not error for cross margin")
		if mockTests {
			assert.Equal(t, 20.0, result, "GetLeverage should return the cross margin leverage")
		} else {
			assert.Positive(t, result, "GetLeverage should return the cross margin leverage")
		}

		result, err = e.GetLeverage(contextGenerate(), asset.Futures, contract, margin.Isolated, order.Long)
		require.NoError(t, err, "GetLeverage must not error for an isolated long")
		if mockTests {
			assert.Equal(t, 10.0, result, "GetLeverage should return the isolated long leverage")
		} else {
			assert.Positive(t, result, "GetLeverage should return the isolated long leverage")
		}

		result, err = e.GetLeverage(contextGenerate(), asset.Futures, contract, margin.Isolated, order.Short)
		require.NoError(t, err, "GetLeverage must not error for an isolated short")
		if mockTests {
			assert.Equal(t, 5.0, result, "GetLeverage should return the isolated short leverage")
		} else {
			assert.Positive(t, result, "GetLeverage should return the isolated short leverage")
		}
	})

	t.Run("without configured leverage", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/leverage-info" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{})
		}))
		_, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, order.AnySide)
		require.ErrorIs(t, err, futures.ErrPositionNotFound, "a query for an instrument without configured leverage must report the missing position")
	})

	t.Run("propagates request errors", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/leverage-info" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			http.Error(w, `{"code":"51000","data":[],"msg":"Parameter instId error"}`, http.StatusBadRequest)
		}))
		_, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, order.AnySide)
		require.ErrorContains(t, err, "51000", "upstream leverage-info errors must propagate to the caller")
	})

	t.Run("requires a matching position side row", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/account/config":
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
			case "/account/leverage-info":
				writeOKXData(t, w, []map[string]string{{
					"instId": "BTC-USDT-SWAP", "mgnMode": "isolated", "posSide": "short", "lever": "3",
				}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		_, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Isolated, order.Long)
		require.ErrorIs(t, err, futures.ErrPositionNotFound, "a per-side query without the requested side's row must not fall back to another side's leverage")
		lever, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Isolated, order.Short)
		require.NoError(t, err, "GetLeverage isolated short must not error")
		assert.Equal(t, 3.0, lever, "the short side should read its own leverage row")
	})

	t.Run("rejects unformattable pairs", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		err := ex.CurrencyPairs.Store(asset.PerpetualSwap, &currency.PairStore{AssetEnabled: true})
		// The mock setup shares a global pair format; a per-asset store with nil
		// formats must be authoritative for the failure to surface.
		ex.CurrencyPairs.UseGlobalFormat = false
		require.NoError(t, err, "storing a formatless pair store must not error")
		err = ex.SetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, 3, order.AnySide)
		require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "SetLeverage must surface the pair format failure")
		_, err = ex.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, margin.Multi, order.AnySide)
		require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetLeverage must surface the pair format failure")
	})

	t.Run("follows the contract position mode", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			mode      string // empty when no account configuration fetch is expected
			margin    margin.Type
			side      order.Side
			expPos    string // expected posSide in the set-leverage body; "" means absent
			expGet    float64
			expSetErr error
		}{
			{"long/short mode isolated long", positionModeLongShort, margin.Isolated, order.Long, positionSideLong, 5, nil},
			{"long/short mode isolated short", positionModeLongShort, margin.Isolated, order.Short, positionSideShort, 10, nil},
			{"long/short mode isolated without a side", positionModeLongShort, margin.Isolated, order.UnknownSide, "", 0, order.ErrSideIsInvalid},
			{"net mode isolated without a side", positionModeNet, margin.Isolated, order.UnknownSide, "", 3, nil},
			{"cross sends no position side and fetches no mode", "", margin.Multi, order.UnknownSide, "", 7, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/account/config":
						if tc.mode == "" {
							t.Errorf("unexpected account configuration fetch for %s margin", tc.margin)
						}
						writeOKXData(t, w, []map[string]string{{"posMode": tc.mode}})
					case "/account/set-leverage":
						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err, "reading the set leverage body should not error")
						mu.Lock()
						body = b
						mu.Unlock()
						writeOKXData(t, w, []map[string]string{{"lever": "5", "mgnMode": "isolated", "posSide": tc.expPos, "instId": perpetualSwapPair.String()}})
					case "/account/leverage-info":
						if r.URL.Query().Get("mgnMode") == TradeModeCross {
							writeOKXData(t, w, []map[string]string{
								{"instId": perpetualSwapPair.String(), "mgnMode": "cross", "posSide": "", "lever": "7"},
							})
							return
						}
						writeOKXData(t, w, []map[string]string{
							{"instId": perpetualSwapPair.String(), "mgnMode": "isolated", "posSide": positionSideLong, "lever": "5"},
							{"instId": perpetualSwapPair.String(), "mgnMode": "isolated", "posSide": positionSideShort, "lever": "10"},
							{"instId": perpetualSwapPair.String(), "mgnMode": "isolated", "posSide": positionSideNet, "lever": "3"},
						})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				err := ex.SetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, tc.margin, 5, tc.side)
				if tc.expSetErr != nil {
					require.ErrorIsf(t, err, tc.expSetErr, "SetLeverage must reject %s", tc.name)
					assert.Empty(t, body, "a rejected leverage setting should not reach OKX")
				} else {
					require.NoError(t, err, "SetLeverage must not error")
					mu.Lock()
					var sent map[string]any
					require.NoError(t, json.Unmarshal(body, &sent), "the set leverage body must decode")
					mu.Unlock()
					if tc.expPos == "" {
						_, has := sent["posSide"]
						assert.False(t, has, "posSide should be omitted")
					} else {
						assert.Equal(t, tc.expPos, sent["posSide"], "the leverage should target the position side the mode requires")
					}
				}

				got, err := ex.GetLeverage(t.Context(), asset.PerpetualSwap, perpetualSwapPair, tc.margin, tc.side)
				if tc.expSetErr != nil {
					require.ErrorIsf(t, err, tc.expSetErr, "GetLeverage must reject %s", tc.name)
				} else {
					require.NoError(t, err, "GetLeverage must not error")
					assert.Equalf(t, tc.expGet, got, "GetLeverage should return the row for %s", tc.name)
				}
			})
		}
		t.Run("options carry no leverage setting", func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}))
			err := ex.SetLeverage(t.Context(), asset.Options, mainPair, margin.Multi, 5, order.UnknownSide)
			assert.ErrorIs(t, err, asset.ErrNotSupported, "SetLeverage should reject options, which OKX gives no leverage setting")
			_, err = ex.GetLeverage(t.Context(), asset.Options, mainPair, margin.Multi, order.UnknownSide)
			assert.ErrorIs(t, err, asset.ErrNotSupported, "GetLeverage should reject options, which OKX gives no leverage setting")
		})
	})
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	pp, err := e.CurrencyPairs.GetPairs(asset.Futures, true)
	require.NoError(t, err)
	err = e.SetLeverage(contextGenerate(), asset.Options, pp[0], margin.Multi, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.SetLeverage(contextGenerate(), asset.Spot, pp[0], margin.Multi, 5, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Unset, 5, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.UnknownSide)
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "without credentials the fetch must fail before any request is sent, as with them it would change the live account's leverage")
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Multi, 5, order.UnknownSide)
	require.NoError(t, err)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Long)
	require.NoError(t, err)
	err = e.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Short)
	assert.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetFuturesContractDetails(contextGenerate(), asset.Spot)
		require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
		_, err = e.GetFuturesContractDetails(contextGenerate(), asset.USDTMarginedFutures)
		require.ErrorIs(t, err, asset.ErrNotSupported)

		for _, a := range []asset.Item{asset.Futures, asset.PerpetualSwap, asset.Spread} {
			result, err := e.GetFuturesContractDetails(contextGenerate(), a)
			require.NoError(t, err)
			require.NotNil(t, result)
		}
	})
	for _, tc := range []struct {
		name, instrumentType, underlying, state, settlementType, alias string
		item                                                           asset.Item
		currency                                                       currency.Code
		wantSettlement                                                 futures.ContractSettlementType
		wantType                                                       futures.ContractType
		wantErr                                                        error
	}{
		{name: "spot", item: asset.Spot, wantErr: futures.ErrNotFuturesAsset},
		{name: "unsupported futures", item: asset.USDTMarginedFutures, wantErr: asset.ErrNotSupported},
		{name: "wrong instrument type", item: asset.PerpetualSwap, instrumentType: "SPOT", wantErr: asset.ErrNotSupported},
		{name: "active missing underlying", item: asset.PerpetualSwap, state: "live", wantErr: currency.ErrCreatingPair},
		{name: "invalid settlement", item: asset.PerpetualSwap, underlying: "BTC-USD", settlementType: "unknown", wantErr: futures.ErrInvalidContractSettlementType},
		{name: "linear", item: asset.PerpetualSwap, instrumentType: "SWAP", underlying: "BTC-USDT", state: "live", settlementType: "linear", currency: currency.USDT, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "inverse empty base currency", item: asset.PerpetualSwap, instrumentType: "swap", underlying: "BTC-USD", state: "live", settlementType: "inverse", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Perpetual},
		{name: "inverse fallback", item: asset.PerpetualSwap, underlying: "BTC-USD", state: "live", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Perpetual},
		{name: "linear fallback", item: asset.PerpetualSwap, underlying: "BTC-USDT", state: "live", currency: currency.USDT, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "explicit type wins", item: asset.PerpetualSwap, underlying: "BTC-USD", state: "live", settlementType: "linear", currency: currency.BTC, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "suspended inverse", item: asset.PerpetualSwap, state: "suspend", settlementType: "inverse", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Perpetual},
		{name: "inactive preserves underlying", item: asset.PerpetualSwap, state: "preopen", underlying: "BTC-USDT", settlementType: "linear", currency: currency.USDT, wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "inactive empty metadata", item: asset.PerpetualSwap, state: "expired", wantSettlement: futures.Linear, wantType: futures.Perpetual},
		{name: "this week", item: asset.Futures, instrumentType: "FUTURES", state: "live", underlying: "BTC-USD", alias: "this_week", settlementType: "inverse", currency: currency.BTC, wantSettlement: futures.Inverse, wantType: futures.Weekly},
		{name: "next week", item: asset.Futures, underlying: "BTC-USD", alias: "next_week", wantSettlement: futures.Linear, wantType: futures.Weekly},
		{name: "quarter", item: asset.Futures, underlying: "BTC-USD", alias: "quarter", wantSettlement: futures.Linear, wantType: futures.Quarterly},
		{name: "next quarter", item: asset.Futures, underlying: "BTC-USD", alias: "next_quarter", wantSettlement: futures.Linear, wantType: futures.Quarterly},
		{name: "unknown alias", item: asset.Futures, underlying: "BTC-USD", alias: "other", wantSettlement: futures.Linear},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			started, expires := time.Unix(1700000000, 0).UTC(), time.Unix(1700100000, 0).UTC()
			instrumentID := "BTC-USD-SWAP"
			if tc.item == asset.Futures {
				instrumentID = "BTC-USD-231115"
			}
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/public/instruments", r.URL.Path, "wrapper should request instruments")
				assert.Equal(t, GetInstrumentTypeFromAssetItem(tc.item), r.URL.Query().Get("instType"), "request should use the selected asset")
				_, err := fmt.Fprintf(w, `{"code":"0","data":[{"instId":%q,"instType":%q,"uly":%q,"state":%q,"ctType":%q,"settleCcy":%q,"alias":%q,"ctVal":"100","lever":"20","listTime":"%d","expTime":"%d"}]}`, instrumentID, tc.instrumentType, tc.underlying, tc.state, tc.settlementType, tc.currency.String(), tc.alias, started.UnixMilli(), expires.UnixMilli())
				assert.NoError(t, err, "instrument response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
			contracts, err := ex.GetFuturesContractDetails(t.Context(), tc.item)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "invalid metadata must return the expected error")
				return
			}
			require.NoError(t, err, "valid metadata must produce contract details")
			require.Len(t, contracts, 1, "one instrument must produce one contract")
			got := contracts[0]
			assert.Equal(t, tc.wantSettlement, got.SettlementType, "settlement type should follow native metadata")
			assert.Equal(t, tc.wantType, got.Type, "contract type should follow the asset and alias")
			assert.Equal(t, tc.currency, got.MarginCurrency, "margin currency should survive in every state")
			assert.Equal(t, tc.currency, got.SettlementCurrency, "settlement currency should survive in every state")
			assert.Equal(t, tc.state == "live", got.IsActive, "activity should follow instrument state")
			assert.Equal(t, tc.state, got.Status, "status should remain available")
			assert.Equal(t, "Okx", got.Exchange, "exchange should be retained")
			assert.Equal(t, instrumentID, got.Name.String(), "instrument identity should be retained")
			assert.Equal(t, tc.item, got.Asset, "asset should be retained")
			assert.Equal(t, started, got.StartDate.UTC(), "listing time should be retained")
			assert.Equal(t, expires, got.EndDate.UTC(), "expiry should be retained")
			assert.Equal(t, float64(100), got.Multiplier, "contract value should be retained")
			assert.Equal(t, float64(20), got.MaxLeverage, "leverage should be retained")
			if tc.underlying != "" {
				assert.Equal(t, tc.underlying, got.Underlying.String(), "underlying should be retained in every state")
			}
		})
	}
	t.Run("reads the settlement type from ctType", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/public/instruments" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{
				{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"},
				{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"},
			})
		}))
		contracts, err := e.GetFuturesContractDetails(t.Context(), asset.PerpetualSwap)
		require.NoError(t, err, "GetFuturesContractDetails must not error")
		require.Len(t, contracts, 2, "GetFuturesContractDetails must return both contracts")
		byID := make(map[string]futures.Contract, len(contracts))
		for i := range contracts {
			byID[contracts[i].Name.String()] = contracts[i]
		}
		assert.Equal(t, futures.Linear, byID["BTC-USDT-SWAP"].SettlementType, "a ctType linear contract should report as linear")
		assert.Equal(t, futures.Inverse, byID["BTC-USD-SWAP"].SettlementType, "a ctType inverse contract should report as inverse, not linear")
	})
	t.Run("reads the spread settlement currency", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/sprd/spreads" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{
				{"sprdId": "BTC-USDT_BTC-USDT-SWAP", "sprdType": "linear", "state": "live", "baseCcy": "BTC", "szCcy": "BTC", "quoteCcy": "USDT"},
				{"sprdId": "BTC-USD-SWAP_BTC-USD-261225", "sprdType": "inverse", "state": "live", "baseCcy": "BTC", "szCcy": "USD", "quoteCcy": "USD"},
			})
		}))
		contracts, err := e.GetFuturesContractDetails(t.Context(), asset.Spread)
		require.NoError(t, err, "GetFuturesContractDetails must not error")
		require.Len(t, contracts, 2, "GetFuturesContractDetails must return both spreads")
		assert.Equal(t, "USDT", contracts[0].SettlementCurrency.String(), "a linear spread should settle in its quote currency")
		assert.Equal(t, "BTC", contracts[1].SettlementCurrency.String(), "an inverse spread should settle in its base currency")
	})
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(contextGenerate(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	// GetOpenInterest reports only enabled pairs, and enabling them on e would change the pairs the tests beside this
	// one read
	ex := newIsolatedExchange(t)
	usdSwapCode := currency.NewCode("USD-SWAP")
	cp1 := currency.NewPair(currency.DOGE, usdSwapCode)
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.PerpetualSwap, perpetualSwapPair, cp1)
	openInterest := func(p currency.Pair, oi float64) futures.OpenInterest {
		return futures.OpenInterest{Key: key.NewExchangeAssetPair(ex.Name, asset.PerpetualSwap, p), OpenInterest: oi}
	}

	resp, err := ex.GetOpenInterest(contextGenerate(), key.PairAsset{
		Base:  perpetualSwapPair.Base.Item,
		Quote: perpetualSwapPair.Quote.Item,
		Asset: asset.PerpetualSwap,
	})
	require.NoError(t, err, "GetOpenInterest must not error for one pair")
	if mockTests {
		exp := []futures.OpenInterest{openInterest(perpetualSwapPair, 3166850.88000001446)}
		assert.Equal(t, exp, resp, "GetOpenInterest should report the requested pair's open interest")
	} else {
		assert.NotEmpty(t, resp, "GetOpenInterest should report the requested pair's open interest")
	}

	resp, err = ex.GetOpenInterest(
		contextGenerate(),
		key.PairAsset{
			Base:  currency.BTC.Item,
			Quote: usdSwapCode.Item,
			Asset: asset.PerpetualSwap,
		},
		key.PairAsset{
			Base:  cp1.Base.Item,
			Quote: cp1.Quote.Item,
			Asset: asset.PerpetualSwap,
		},
	)
	require.NoError(t, err, "GetOpenInterest must not error for several pairs")
	if mockTests {
		exp := []futures.OpenInterest{
			openInterest(currency.NewPair(currency.BTC, usdSwapCode), 6573951.89999999923),
			openInterest(cp1, 1954073),
		}
		assert.Equal(t, exp, resp, "GetOpenInterest should report only the requested pairs")
	} else {
		assert.NotEmpty(t, resp, "GetOpenInterest should report the requested pairs")
	}

	resp, err = ex.GetOpenInterest(contextGenerate())
	require.NoError(t, err, "GetOpenInterest must not error for every enabled pair")
	if mockTests {
		exp := []futures.OpenInterest{
			openInterest(currency.NewPair(currency.BTC, usdSwapCode), 6573951.89999999923),
			openInterest(currency.NewPair(currency.ETH, usdSwapCode), 20685137),
			openInterest(cp1, 1954073),
		}
		assert.Equal(t, exp, resp, "GetOpenInterest should report every enabled pair")
	} else {
		assert.NotEmpty(t, resp, "GetOpenInterest should report the enabled pairs")
	}

	t.Run("single key of a pair that is not enabled", func(t *testing.T) {
		t.Parallel()
		ex := newIsolatedExchange(t)
		enabled, err := ex.IsPairEnabled(perpetualSwapPair, asset.PerpetualSwap)
		require.NoError(t, err, "IsPairEnabled must not error")
		require.False(t, enabled, "the requested pair must not be enabled")
		resp, err := ex.GetOpenInterest(contextGenerate(), key.PairAsset{Base: perpetualSwapPair.Base.Item, Quote: perpetualSwapPair.Quote.Item, Asset: asset.PerpetualSwap})
		require.NoError(t, err, "GetOpenInterest must not error for one pair")
		require.Len(t, resp, 1, "GetOpenInterest must report the requested pair's open interest")
		if mockTests {
			exp := []futures.OpenInterest{{Key: key.NewExchangeAssetPair(ex.Name, asset.PerpetualSwap, perpetualSwapPair), OpenInterest: 3166850.88000001446}}
			assert.Equal(t, exp, resp, "GetOpenInterest should report the requested pair's open interest whether or not it is enabled")
		} else {
			assert.Equal(t, key.NewExchangeAssetPair(ex.Name, asset.PerpetualSwap, perpetualSwapPair), resp[0].Key, "GetOpenInterest should key the open interest by the requested pair")
			assert.Positive(t, resp[0].OpenInterest, "GetOpenInterest should report the pair's open interest")
		}
	})

	t.Run("single key reply without the requested instrument", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			data []map[string]string
		}{
			{"no rows", []map[string]string{}},
			{"another instrument", []map[string]string{{"instType": instTypeSwap, "instId": "ETH-USDT-SWAP", "oi": "5"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/public/open-interest", r.URL.Path, "the open interest should come from the open interest endpoint")
					assert.Equal(t, url.Values{"instType": {instTypeSwap}, "instId": {"BTC-USDT-SWAP"}}, r.URL.Query(), "the request should name the instrument")
					writeOKXData(t, w, tc.data)
				}))
				_, err := ex.GetOpenInterest(t.Context(), key.PairAsset{Base: perpetualSwapPair.Base.Item, Quote: perpetualSwapPair.Quote.Item, Asset: asset.PerpetualSwap})
				assert.ErrorIs(t, err, common.ErrNoResponse, "GetOpenInterest should error rather than report an empty entry")
			})
		}
	})

	t.Run("a single options key queries the instrument's own family", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			instID    string
			tickBands string
			family    string
			err       error
		}{
			{
				name:      "plain family option, _UM family listed first",
				instID:    "BTC-USD-260928-79000-C",
				tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]}]`,
				family:    "BTC-USD",
			},
			{
				name:      "_UM option, plain family listed first",
				instID:    "BTC-USD_UM-260928-79000-C",
				tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]}]`,
				family:    "BTC-USD_UM",
			},
			{
				name:      "family missing from the tick bands",
				instID:    "ETH-USD-260928-2000-C",
				tickBands: `[{"instType":"OPTION","instFamily":"BTC-USD","tickBand":[]},{"instType":"OPTION","instFamily":"BTC-USD_UM","tickBand":[]}]`,
				err:       errInstrumentFamilyRequired,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := new(Exchange)
				require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

				var mu sync.Mutex
				var paths []string
				srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
					mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/public/instrument-tick-bands":
						_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.tickBands + `}`))
					case "/public/open-interest":
						if r.URL.Query().Get("instFamily") != tc.family {
							_, _ = w.Write([]byte(`{"code":"51002","msg":"Instrument ID doesn't match underlying index.","data":[]}`))
							return
						}
						_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"` + tc.instID + `","instType":"OPTION","oi":"100","oiCcy":"1","oiUsd":"10","ts":"1597026383085"}]}`))
					default:
						_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
					}
				}))
				b := e.GetBase()
				b.SkipAuthCheck = true
				require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
				for k := range b.API.Endpoints.GetURLMap() {
					require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
				}

				optionInstID, err := currency.NewPairFromString(tc.instID)
				require.NoError(t, err, "option instrument ID must parse")
				sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, e, asset.Options, optionInstID)

				resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
					Base:  optionInstID.Base.Item,
					Quote: optionInstID.Quote.Item,
					Asset: asset.Options,
				})
				mu.Lock()
				defer mu.Unlock()
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err, "GetOpenInterest must reject an option whose family isn't listed")
					assert.Equal(t, []string{"/public/instrument-tick-bands?instType=OPTION"}, paths, "no open interest query should be made without the family")
					return
				}
				require.NoError(t, err, "GetOpenInterest with an Options key must not error")
				require.Len(t, resp, 1, "GetOpenInterest with an Options key must return the requested instrument entry")
				assert.Equal(t, 100.0, resp[0].OpenInterest, "the entry should carry the mocked open interest")
				assert.Truef(t, resp[0].Key.Pair().Equal(optionInstID), "the entry should resolve to the requested pair, got %v", resp[0].Key.Pair())
				assert.Equal(t, []string{
					"/public/instrument-tick-bands?instType=OPTION",
					"/public/open-interest?instFamily=" + tc.family + "&instId=" + tc.instID + "&instType=OPTION",
				}, paths, "only the instrument's own family should be queried, with the instrument pinned by ID")
			})
		}
	})

	t.Run("several keys report their instruments from the markets they name", func(t *testing.T) {
		t.Parallel()
		btcSwap := currency.NewPairWithDelimiter("BTC", "USDT-SWAP", currency.DashDelimiter)
		ethSwap := currency.NewPairWithDelimiter("ETH", "USDT-SWAP", currency.DashDelimiter)
		xrpSwap := currency.NewPairWithDelimiter("XRP", "USDT-SWAP", currency.DashDelimiter)
		btcFutures := currency.NewPairWithDelimiter("BTC", "USD-261225", currency.DashDelimiter)
		btcOption := currency.NewPairWithDelimiter("BTC", "USD-260928-79000-C", currency.DashDelimiter)
		pairKey := func(a asset.Item, p currency.Pair) key.PairAsset {
			return key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}
		}
		for _, tc := range []struct {
			name     string
			keys     []key.PairAsset
			requests []string
			exp      []futures.OpenInterest
		}{
			{
				name:     "a swap not enabled beside an enabled one",
				keys:     []key.PairAsset{pairKey(asset.PerpetualSwap, btcSwap), pairKey(asset.PerpetualSwap, ethSwap)},
				requests: []string{"/public/open-interest?instType=SWAP"},
				exp: []futures.OpenInterest{
					{Key: key.NewExchangeAssetPair("Okx", asset.PerpetualSwap, btcSwap), OpenInterest: 100},
					{Key: key.NewExchangeAssetPair("Okx", asset.PerpetualSwap, ethSwap), OpenInterest: 200},
				},
			},
			{
				name:     "an option beside a swap",
				keys:     []key.PairAsset{pairKey(asset.PerpetualSwap, ethSwap), pairKey(asset.Options, btcOption)},
				requests: []string{"/public/instrument-tick-bands?instType=OPTION", "/public/open-interest?instType=SWAP", "/public/open-interest?instFamily=BTC-USD&instType=OPTION"},
				exp: []futures.OpenInterest{
					{Key: key.NewExchangeAssetPair("Okx", asset.PerpetualSwap, ethSwap), OpenInterest: 200},
					{Key: key.NewExchangeAssetPair("Okx", asset.Options, btcOption), OpenInterest: 400},
				},
			},
			{
				name:     "a futures contract twice beside a swap",
				keys:     []key.PairAsset{pairKey(asset.Futures, btcFutures), pairKey(asset.PerpetualSwap, ethSwap), pairKey(asset.Futures, btcFutures)},
				requests: []string{"/public/open-interest?instType=FUTURES", "/public/open-interest?instType=SWAP"},
				exp: []futures.OpenInterest{
					{Key: key.NewExchangeAssetPair("Okx", asset.Futures, btcFutures), OpenInterest: 300},
					{Key: key.NewExchangeAssetPair("Okx", asset.PerpetualSwap, ethSwap), OpenInterest: 200},
				},
			},
			{
				name:     "a swap OKX does not report",
				keys:     []key.PairAsset{pairKey(asset.PerpetualSwap, xrpSwap), pairKey(asset.PerpetualSwap, btcSwap)},
				requests: []string{"/public/open-interest?instType=SWAP"},
				exp:      []futures.OpenInterest{{Key: key.NewExchangeAssetPair("Okx", asset.PerpetualSwap, btcSwap), OpenInterest: 100}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var requests []string
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)
					mu.Unlock()
					switch r.URL.Path + "?" + r.URL.RawQuery {
					case "/public/instrument-tick-bands?instType=OPTION":
						writeOKXData(t, w, []map[string]any{{"instType": instTypeOption, "instFamily": "BTC-USD_UM", "tickBand": []any{}}, {"instType": instTypeOption, "instFamily": "BTC-USD", "tickBand": []any{}}})
					case "/public/open-interest?instType=SWAP":
						writeOKXData(t, w, []map[string]string{
							{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "oi": "100"},
							{"instType": instTypeSwap, "instId": "ETH-USDT-SWAP", "oi": "200"},
							{"instType": instTypeSwap, "instId": "SOL-USDT-SWAP", "oi": "500"},
						})
					case "/public/open-interest?instType=FUTURES":
						writeOKXData(t, w, []map[string]string{{"instType": instTypeFutures, "instId": "BTC-USD-261225", "oi": "300"}})
					case "/public/open-interest?instFamily=BTC-USD&instType=OPTION":
						writeOKXData(t, w, []map[string]string{{"instType": instTypeOption, "instId": "BTC-USD-260928-79000-C", "oi": "400"}})
					default:
						_, err := w.Write([]byte(`{"code":"51000","msg":"unexpected market","data":[]}`))
						assert.NoError(t, err, "writing the reply should not error")
					}
				}))
				require.NoError(t, ex.CurrencyPairs.StorePairs(asset.PerpetualSwap, currency.Pairs{btcSwap, ethSwap, xrpSwap}, false), "StorePairs must make the swaps available")
				require.NoError(t, ex.CurrencyPairs.StorePairs(asset.PerpetualSwap, currency.Pairs{ethSwap}, true), "StorePairs must enable the ETH swap alone")
				resp, err := ex.GetOpenInterest(t.Context(), tc.keys...)
				require.NoError(t, err, "GetOpenInterest must not error")
				assert.Equal(t, tc.exp, resp, "GetOpenInterest should report each requested instrument once, whether or not its pair is enabled")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.requests, requests, "GetOpenInterest should read only the markets the keys name, once each")
			})
		}
	})
}

func TestOptionInstrumentFamily(t *testing.T) {
	t.Parallel()
	umFirst := []string{"BTC-USD_UM", "BTC-USD", "ETH-USD"}
	plainFirst := []string{"BTC-USD", "BTC-USD_UM", "ETH-USD"}
	for _, tc := range []struct {
		name, instrumentID string
		families           []string
		exp                string
	}{
		{"plain family, _UM family listed first", "BTC-USD-260928-79000-C", umFirst, "BTC-USD"},
		{"_UM family, listed first", "BTC-USD_UM-260928-79000-C", umFirst, "BTC-USD_UM"},
		{"_UM family, plain family listed first", "BTC-USD_UM-260928-79000-C", plainFirst, "BTC-USD_UM"},
		{"another underlying", "ETH-USD-260928-2000-P", umFirst, "ETH-USD"},
		{"family not listed", "SOL-USD-260928-100-C", umFirst, ""},
		{"not an option", "BTC-USDT-SWAP", plainFirst, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, optionInstrumentFamily(tc.families, tc.instrumentID), "optionInstrumentFamily should return the family the instrument ID begins with")
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	// The first listed instrument of each asset in the recorded instrument lists
	mockURLs := map[asset.Item]string{
		asset.Spot:          baseURL + "trade-spot/btc-usdt",
		asset.Margin:        baseURL + "trade-margin/btc-usdt",
		asset.PerpetualSwap: baseURL + "trade-swap/btc-usd-swap",
		asset.Futures:       baseURL + "trade-futures/btc-usd-monthly",
		asset.Options:       baseURL + "trade-option/sol-usd",
		asset.Spread:        baseURL,
	}
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			pairs, err := e.CurrencyPairs.GetPairs(a, false)
			require.NoErrorf(t, err, "GetPairs must not error for asset %s", a)
			require.NotEmptyf(t, pairs, "GetPairs must return pairs for asset %s", a)

			resp, err := e.GetCurrencyTradeURL(contextGenerate(), a, pairs[0])
			require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for asset %s and pair %s", a, pairs[0])
			if mockTests {
				assert.Equalf(t, mockURLs[a], resp, "GetCurrencyTradeURL should return the trade page of %s %s", a, pairs[0])
			} else {
				assert.NotEmptyf(t, resp, "GetCurrencyTradeURL should return a URL for asset %s and pair %s", a, pairs[0])
			}
		})
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		feeBuilder := &exchange.FeeBuilder{Amount: 1, FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: mainPair, PurchasePrice: 1}
		_, err := e.GetFee(contextGenerate(), feeBuilder)
		require.ErrorIs(t, err, errFeeTypeUnsupported)
		feeBuilder.FeeType = exchange.OfflineTradeFee
		_, err = e.GetFee(contextGenerate(), feeBuilder)
		assert.NoError(t, err)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		feeBuilder.FeeType = exchange.CryptocurrencyTradeFee
		_, err = e.GetFee(contextGenerate(), feeBuilder)
		require.NoError(t, err)
	})
	t.Run("nil builder", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetFee(t.Context(), nil)
		require.ErrorIs(t, err, common.ErrNilPointer, "nil builder must fail safely")
	})
	for _, tc := range []struct {
		name, fees, instruments string
		quote                   currency.Code
		want, wantTaker         float64
		wantErr                 error
	}{
		{name: "legacy USDT", fees: `[{"maker":"-0.001","taker":"-0.001"}]`, want: 1},
		{name: "distinct legacy sides", fees: `[{"maker":"-0.001","taker":"-0.002"}]`, want: 1, wantTaker: 2},
		{name: "distinct modern sides", fees: `[{"feeGroup":[{"groupId":"1","maker":"-0.001","taker":"-0.002"}]}]`, want: 1, wantTaker: 2},
		{name: "legacy zero ignores derivatives", fees: `[{"maker":"0","taker":"0","makerU":"-0.1","takerU":"-0.1"}]`},
		{name: "legacy rebate", fees: `[{"maker":"0.001","taker":"0.001"}]`, want: -1},
		{name: "legacy USDC", quote: currency.USDC, fees: `[{"maker":"-0.1","taker":"-0.1","makerUSDC":"-0.002","takerUSDC":"-0.002"}]`, want: 2},
		{name: "legacy crypto quote", quote: currency.BTC, fees: `[{"maker":"-0.1","taker":"-0.1","makerUSDC":"-0.002","takerUSDC":"-0.002"}]`, want: 2},
		{name: "legacy USDC zero", quote: currency.USDC, fees: `[{"makerUSDC":"0","takerUSDC":"0"}]`},
		{name: "missing rates", fees: `[{}]`, wantErr: errFeeRateUnavailable},
		{name: "empty rates", fees: `[{"maker":"","taker":""}]`, wantErr: errFeeRateUnavailable},
		{name: "null rates", fees: `[{"maker":null,"taker":null}]`, wantErr: errFeeRateUnavailable},
		{name: "derivatives rates cannot price spot", fees: `[{"makerU":"-0.001","takerU":"-0.001"}]`, wantErr: errFeeRateUnavailable},
		{name: "missing legacy crypto rate", quote: currency.BTC, fees: `[{"maker":"-0.001","taker":"-0.001"}]`, wantErr: errFeeRateUnavailable},
		{name: "legacy fiat unsupported", quote: currency.AUD, fees: `[{"makerUSDC":"-0.001","takerUSDC":"-0.001"}]`, wantErr: errFeeRateUnavailable},
		{name: "empty response", fees: `[]`, wantErr: common.ErrNoResponse},
		{name: "multiple responses", fees: `[{},{}]`, wantErr: errFeeRateUnavailable},
		{name: "wrong asset", fees: `[{"instType":"MARGIN","maker":"-0.001","taker":"-0.001"}]`, wantErr: errFeeRateUnavailable},
		{name: "modern group wins", fees: `[{"instType":"SPOT","maker":"-0.1","taker":"-0.1","feeGroup":[{"groupId":"1","maker":"-0.003","taker":"-0.003"}]}]`, want: 3},
		{name: "modern zero", fees: `[{"feeGroup":[{"groupId":"1","maker":"0","taker":"0"}]}]`},
		{name: "modern rebate", fees: `[{"feeGroup":[{"groupId":"1","maker":"0.003","taker":"0.003"}]}]`, want: -3},
		{name: "modern fiat", quote: currency.AUD, fees: `[{"feeGroup":[{"groupId":"8","maker":"-0.003","taker":"-0.003"}]}]`, want: 3},
		{name: "missing modern rates do not fall back", fees: `[{"maker":"-0.1","taker":"-0.1","feeGroup":[{"groupId":"1"}]}]`, wantErr: errFeeRateUnavailable},
		{name: "missing modern group ID", fees: `[{"feeGroup":[{"maker":"-0.003","taker":"-0.003"}]}]`, wantErr: errFeeRateUnavailable},
		{name: "select matching group", fees: `[{"feeGroup":[{"groupId":"1","maker":"-0.1","taker":"-0.1"},{"groupId":"2","maker":"-0.003","taker":"-0.003"}]}]`, instruments: `[{"instType":"SPOT","instId":"ETH-USDT","groupId":"2"}]`, want: 3},
		{name: "missing instrument group", fees: `[{"feeGroup":[{"groupId":"1"},{"groupId":"2"}]}]`, instruments: `[{"instType":"SPOT","instId":"ETH-USDT"}]`, wantErr: errFeeRateUnavailable},
		{name: "unmatched instrument group", fees: `[{"feeGroup":[{"groupId":"1"},{"groupId":"2"}]}]`, instruments: `[{"instType":"SPOT","instId":"ETH-USDT","groupId":"3"}]`, wantErr: errFeeRateUnavailable},
		{name: "duplicate group", fees: `[{"feeGroup":[{"groupId":"2"},{"groupId":"2"}]}]`, instruments: `[{"instType":"SPOT","instId":"ETH-USDT","groupId":"2"}]`, wantErr: errFeeRateUnavailable},
	} {
		for _, maker := range []bool{true, false} {
			t.Run(tc.name+" maker="+strconv.FormatBool(maker), func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "exchange must initialise")
				ex.SetCredentials(&accounts.Credentials{Key: "test", Secret: "test", ClientID: "test"})
				ex.SkipAuthCheck = true
				quote := tc.quote
				if quote.IsEmpty() {
					quote = currency.USDT
				}
				pair := currency.NewPair(currency.ETH, quote)
				server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "SPOT", r.URL.Query().Get("instType"), "request should scope spot fees")
					assert.Equal(t, "ETH-"+quote.String(), r.URL.Query().Get("instId"), "request should retain the instrument for applicable rates")
					data := tc.fees
					if r.URL.Path == "/public/instruments" {
						assert.NotEmpty(t, tc.instruments, "instrument lookup should only resolve multiple groups")
						data = tc.instruments
					} else {
						assert.Equal(t, "/account/trade-fee", r.URL.Path, "request should query account fees")
					}
					_, err := fmt.Fprintf(w, `{"code":"0","data":%s}`, data)
					assert.NoError(t, err, "response should write")
				}))
				require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
				require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
				got, err := ex.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: pair, IsMaker: maker, Amount: 2, PurchasePrice: 500})
				require.ErrorIs(t, err, tc.wantErr, "fee selection must return the expected result")
				want := tc.want
				if !maker && tc.wantTaker != 0 {
					want = tc.wantTaker
				}
				assert.InDelta(t, want, got, 0.00000001, "fee should preserve selected rate and commission sign")
			})
		}
	}
}

func TestValidateSubmitPrelude(t *testing.T) {
	t.Parallel()
	newConfigExchange := func(accountLevel string) *Exchange {
		return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/config" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{{"acctLv": accountLevel, "posMode": positionModeLongShort}})
		}))
	}
	multiCurrencyMargin, futuresMode, noAccountLevel, spotMode := newConfigExchange("3"), newConfigExchange("2"), newConfigExchange(""), newConfigExchange("1")
	optionPair := currency.NewPairWithDelimiter("BTC", "USD-261225-100000-C", currency.DashDelimiter)
	futuresContract := currency.NewPairWithDelimiter("BTC", "USD-261225", currency.DashDelimiter)
	for _, tc := range []struct {
		name           string
		futuresMode    bool
		noAccountLevel bool
		spotMode       bool
		s              *order.Submit
		exp            *submitPrelude
		err            error
	}{
		{name: "nil submit", err: order.ErrSubmissionIsNil},
		{name: "unsupported asset", s: &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}, err: asset.ErrNotSupported},
		{name: "good till time", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillTime, Amount: 1, Price: 1}, err: order.ErrUnsupportedTimeInForce},
		{name: "good till day", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillDay, Amount: 1, Price: 1}, err: order.ErrUnsupportedTimeInForce},
		{name: "stop or reduce", s: &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Sell, Type: order.Limit, TimeInForce: order.StopOrReduce, Amount: 1, Price: 1}, err: order.ErrUnsupportedTimeInForce},
		{name: "good till crossing without post only", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillCrossing, Amount: 1, Price: 1}, err: order.ErrUnsupportedTimeInForce},
		{name: "spot cross margin market buy without a quote amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Market, MarginType: margin.Multi, Amount: 1}, err: order.ErrAmountMustBeSet},
		{name: "margin market sell sized by a quote amount", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Sell, Type: order.Market, QuoteAmount: 100}, err: limits.ErrAmountBelowMin},
		{name: "cross margin order without an account level", noAccountLevel: true, s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}, err: errInvalidAccountLevel},
		{name: "negative amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: -1, Price: 1}, err: limits.ErrAmountBelowMin},
		{name: "margin market buy without a quote amount", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Market, Amount: 1}, err: order.ErrAmountMustBeSet},
		{name: "spot trailing stop buy without a quote amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.TrailingStop, Amount: 1}, err: order.ErrAmountMustBeSet},
		{name: "limit without an amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Price: 1}, err: limits.ErrAmountBelowMin},
		{name: "limit sized by a quote amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, QuoteAmount: 100, Price: 1}, err: limits.ErrAmountBelowMin},
		{name: "spot market without an amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Market}, err: limits.ErrAmountBelowMin},
		{name: "optimal limit on spot", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.OptimalLimit, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{name: "market order on options", s: &order.Submit{AssetType: asset.Options, Pair: optionPair, Side: order.Buy, Type: order.Market, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{name: "market maker protection on spot", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.MarketMakerProtection, Amount: 1, Price: 1}, err: order.ErrUnsupportedOrderType},
		{name: "market maker protection without a price", s: &order.Submit{AssetType: asset.Options, Pair: optionPair, Side: order.Buy, Type: order.MarketMakerProtection, Amount: 1}, err: limits.ErrPriceBelowMin},
		{name: "limit without a price", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1}, err: limits.ErrPriceBelowMin},
		{name: "limit at a negative price", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: -1}, err: limits.ErrPriceBelowMin},
		{name: "empty pair", s: &order.Submit{AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}, err: currency.ErrCurrencyPairEmpty},
		{name: "futures leverage", s: &order.Submit{AssetType: asset.Futures, Pair: futuresContract, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, Leverage: 2}, err: order.ErrSubmitLeverageNotSupported},
		{name: "unknown margin type", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Unknown}, err: margin.ErrMarginTypeUnsupported},
		{name: "combined margin types", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Isolated | margin.Multi}, err: margin.ErrMarginTypeUnsupported},
		{name: "margin in cash mode", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.NoMargin}, err: margin.ErrMarginTypeUnsupported},
		{name: "margin in spot lead trading mode", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.SpotIsolated}, err: margin.ErrMarginTypeUnsupported},
		{name: "chase on spot", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Chase, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{name: "chase on margin", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Chase, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{name: "chase on options", s: &order.Submit{AssetType: asset.Options, Pair: optionPair, Side: order.Buy, Type: order.Chase, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{name: "chase on a spread", s: &order.Submit{AssetType: asset.Spread, Pair: spreadPair, Side: order.Buy, Type: order.Chase, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{name: "spot without an account level", noAccountLevel: true, s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}, err: errInvalidAccountLevel},
		{
			name: "spot limit defaults to cross in multi-currency margin mode", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY"},
		},
		{
			name: "spot limit defaults to cash in futures mode", futuresMode: true, s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCash, sideType: "BUY"},
		},
		{
			name: "pair in another format", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewPairWithDelimiter("btc", "usdt", "/"), Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "SELL"},
		},
		{
			name: "spot market sized by a quote amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY"},
		},
		{
			name: "spot trailing stop buy sized by a quote amount", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.TrailingStop, Amount: 1, QuoteAmount: 100},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY"},
		},
		{
			name: "trigger order takes no price", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, Type: order.Trigger, Amount: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "SELL"},
		},
		{
			name: "explicit cash", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.NoMargin},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCash, sideType: "BUY"},
		},
		{
			name: "spot isolated", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.SpotIsolated},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeSpotIsolated, sideType: "BUY"},
		},
		{
			name: "margin market buy sized by a quote amount defaults to cross", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY"},
		},
		{
			name: "good till crossing with post only", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillCrossing | order.PostOnly, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY"},
		},
		{
			name: "cross margin buy in Futures mode sends the quote currency", futuresMode: true, s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY", marginCurrency: currency.USDT},
		},
		{
			name: "cross margin sell in Futures mode sends the base currency", futuresMode: true, s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "SELL", marginCurrency: currency.BTC},
		},
		{
			name: "spot cross margin buy in Futures mode sends the quote currency", futuresMode: true, s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Market, MarginType: margin.Multi, QuoteAmount: 100},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY", marginCurrency: currency.USDT},
		},
		{
			name: "spot cash order in Futures mode sends no currency", futuresMode: true, s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCash, sideType: "BUY"},
		},
		{
			name: "isolated margin buy sends the quote currency", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Isolated},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeIsolated, sideType: "BUY", marginCurrency: currency.USDT},
		},
		{
			name: "isolated margin sell sends the base currency", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Isolated},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeIsolated, sideType: "SELL", marginCurrency: currency.BTC},
		},
		{
			name: "isolated spot sell sends the base currency", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Isolated},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeIsolated, sideType: "SELL", marginCurrency: currency.BTC},
		},
		{
			name: "option defaults to cross", s: &order.Submit{AssetType: asset.Options, Pair: optionPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 0.05},
			exp: &submitPrelude{pairString: "BTC-USD-261225-100000-C", tradeMode: TradeModeCross, sideType: "buy"},
		},
		{
			name: "option buyer defaults to cash in spot mode", spotMode: true, s: &order.Submit{AssetType: asset.Options, Pair: optionPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 0.05},
			exp: &submitPrelude{pairString: "BTC-USD-261225-100000-C", tradeMode: TradeModeCash, sideType: "buy"},
		},
		{
			name: "spot defaults to cash in spot mode", spotMode: true, s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCash, sideType: "SELL"},
		},
		{
			name: "explicit cross spot in spot mode", spotMode: true, s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Multi},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "SELL"},
		},
		{
			name: "perpetual swap chase", s: &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Buy, Type: order.Chase, Amount: 1},
			exp: &submitPrelude{pairString: "BTC-USDT-SWAP", tradeMode: TradeModeCross, sideType: "buy", positionMode: positionModeLongShort},
		},
		{
			name: "option market order placed as ioc", s: &order.Submit{AssetType: asset.Options, Pair: optionPair, Side: order.Sell, Type: order.Market, TimeInForce: order.ImmediateOrCancel, Amount: 1},
			exp: &submitPrelude{pairString: "BTC-USD-261225-100000-C", tradeMode: TradeModeCross, sideType: "sell"},
		},
		{
			name: "option market maker protection", s: &order.Submit{AssetType: asset.Options, Pair: optionPair, Side: order.Short, Type: order.MarketMakerProtection, Amount: 1, Price: 0.05},
			exp: &submitPrelude{pairString: "BTC-USD-261225-100000-C", tradeMode: TradeModeCross, sideType: "sell"},
		},
		{
			name: "spread limit at a negative price", s: &order.Submit{AssetType: asset.Spread, Pair: spreadPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: -8.7},
			exp: &submitPrelude{pairString: "BTC-USDT_BTC-USDT-SWAP", sideType: "BUY"},
		},
		{
			name: "spread limit at a zero price", s: &order.Submit{AssetType: asset.Spread, Pair: spreadPair, Side: order.Sell, Type: order.Limit, Amount: 1},
			exp: &submitPrelude{pairString: "BTC-USDT_BTC-USDT-SWAP", sideType: "SELL"},
		},
		{
			name: "perpetual swap takes the account's position mode", s: &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Long, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT-SWAP", tradeMode: TradeModeCross, sideType: "buy", positionMode: positionModeLongShort},
		},
		{
			name: "futures at a leverage of one", s: &order.Submit{AssetType: asset.Futures, Pair: futuresContract, Side: order.Sell, Type: order.OptimalLimit, Amount: 1, Leverage: 1, MarginType: margin.Isolated},
			exp: &submitPrelude{pairString: "BTC-USD-261225", tradeMode: TradeModeIsolated, sideType: "sell", positionMode: positionModeLongShort},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := multiCurrencyMargin
			switch {
			case tc.futuresMode:
				ex = futuresMode
			case tc.noAccountLevel:
				ex = noAccountLevel
			case tc.spotMode:
				ex = spotMode
			}
			p, err := ex.validateSubmitPrelude(t.Context(), tc.s)
			require.ErrorIs(t, err, tc.err, "validateSubmitPrelude must return the expected error")
			assert.Equal(t, tc.exp, p, "validateSubmitPrelude should return the formatted order values")
		})
	}

	t.Run("no request pair format", func(t *testing.T) {
		t.Parallel()
		unformatted := new(Exchange)
		unformatted.CurrencyPairs.UseGlobalFormat = true
		require.NoError(t, unformatted.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{mainPair}, true), "StorePairs must not error")
		_, err := unformatted.validateSubmitPrelude(t.Context(), &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
		assert.ErrorIs(t, err, currency.ErrPairFormatIsNil, "validateSubmitPrelude should report the missing pair format")
	})

	t.Run("position mode fetch error", func(t *testing.T) {
		t.Parallel()
		failing := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`{"code":"50011","msg":"","data":[]}`))
			assert.NoError(t, err, "writing the error reply should not error")
		}))
		_, err := failing.validateSubmitPrelude(t.Context(), &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
		assert.ErrorIs(t, err, ErrorCodes["50011"], "validateSubmitPrelude should return the position mode fetch's error")
	})
}

func TestCheckSubmitPrelude(t *testing.T) {
	t.Parallel()
	ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("checkSubmitPrelude should send no request: %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	for _, tc := range []struct {
		name string
		s    *order.Submit
		exp  *submitPrelude
		err  error
	}{
		{name: "nil submit", err: order.ErrSubmissionIsNil},
		{
			name: "spot leaves its default trade mode to the account", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT", sideType: "BUY"},
		},
		{
			name: "perpetual swap leaves its position mode to the account", s: &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 1},
			exp: &submitPrelude{pairString: "BTC-USDT-SWAP", sideType: "sell"},
		},
		{
			name: "cross margin leaves its margin currency to the account", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.Multi},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY"},
		},
		{name: "margin in cash mode", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, MarginType: margin.NoMargin}, err: margin.ErrMarginTypeUnsupported},
		{name: "spot chase", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Chase, Amount: 1}, err: order.ErrUnsupportedOrderType},
		{
			name: "quote sized oco buy with limit legs missing their limit prices", err: limits.ErrPriceBelowMin,
			s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.OCO, QuoteAmount: 100, RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{OrderType: order.Limit, Price: 95}, StopLoss: order.RiskManagement{OrderType: order.Limit, Price: 110},
			}},
		},
		{
			name: "quote sized oco buy with market legs", exp: &submitPrelude{pairString: "BTC-USDT", sideType: "BUY"},
			s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.OCO, QuoteAmount: 100, RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{OrderType: order.Market, Price: 95}, StopLoss: order.RiskManagement{OrderType: order.Market, Price: 110},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := ex.checkSubmitPrelude(tc.s)
			require.ErrorIs(t, err, tc.err, "checkSubmitPrelude must return the expected error")
			assert.Equal(t, tc.exp, p, "checkSubmitPrelude should return the values the submit decides alone")
		})
	}
}

func TestResolveSubmitAccount(t *testing.T) {
	t.Parallel()
	ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/config" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeLongShort}})
	}))
	for _, tc := range []struct {
		name string
		s    *order.Submit
		p    *submitPrelude
		exp  *submitPrelude
	}{
		{
			name: "spot takes the account's default trade mode", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy},
			p:   &submitPrelude{pairString: "BTC-USDT", sideType: "BUY"},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCash, sideType: "BUY"},
		},
		{
			name: "an explicit trade mode stays", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, MarginType: margin.Isolated},
			p:   &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeIsolated, sideType: "SELL"},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeIsolated, sideType: "SELL", marginCurrency: currency.BTC},
		},
		{
			name: "cross margin in futures mode takes its margin currency", s: &order.Submit{AssetType: asset.Margin, Pair: mainPair, Side: order.Buy},
			p:   &submitPrelude{pairString: "BTC-USDT", sideType: "BUY"},
			exp: &submitPrelude{pairString: "BTC-USDT", tradeMode: TradeModeCross, sideType: "BUY", marginCurrency: currency.USDT},
		},
		{
			name: "perpetual swap takes the account's position mode", s: &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Buy},
			p:   &submitPrelude{pairString: "BTC-USDT-SWAP", sideType: "buy"},
			exp: &submitPrelude{pairString: "BTC-USDT-SWAP", tradeMode: TradeModeCross, sideType: "buy", positionMode: positionModeLongShort},
		},
		{
			name: "spread sends no trade mode", s: &order.Submit{AssetType: asset.Spread, Pair: spreadPair, Side: order.Buy},
			p:   &submitPrelude{pairString: "BTC-USDT_BTC-USDT-SWAP", sideType: "BUY"},
			exp: &submitPrelude{pairString: "BTC-USDT_BTC-USDT-SWAP", sideType: "BUY"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ex.resolveSubmitAccount(t.Context(), tc.s, tc.p), "resolveSubmitAccount must not error")
			assert.Equal(t, tc.exp, tc.p, "resolveSubmitAccount should fill in the values the account decides")
		})
	}
	t.Run("account level fetch error", func(t *testing.T) {
		t.Parallel()
		failing := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`{"code":"50011","msg":"","data":[]}`))
			assert.NoError(t, err, "writing the error reply should not error")
		}))
		err := failing.resolveSubmitAccount(t.Context(), &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy}, &submitPrelude{pairString: "BTC-USDT", sideType: "BUY"})
		assert.ErrorIs(t, err, ErrorCodes["50011"], "resolveSubmitAccount should return the account level fetch's error")
	})
}

func TestDefaultTradeMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		accountLevel string
		assetType    asset.Item
		exp          string
		err          error
	}{
		{"1", asset.Spot, TradeModeCash, nil},
		{"1", asset.Options, TradeModeCash, nil},
		{"1", asset.Margin, TradeModeCross, nil},
		{"2", asset.Spot, TradeModeCash, nil},
		{"2", asset.Options, TradeModeCross, nil},
		{"2", asset.Futures, TradeModeCross, nil},
		{"3", asset.Spot, TradeModeCross, nil},
		{"3", asset.Options, TradeModeCross, nil},
		{"3", asset.PerpetualSwap, TradeModeCross, nil},
		{"4", asset.Spot, TradeModeCross, nil},
		{"4", asset.Options, TradeModeCross, nil},
		{"5", asset.Spot, "", errInvalidAccountLevel},
		{"5", asset.Margin, TradeModeCross, nil},
		{"", asset.Options, "", errInvalidAccountLevel},
		{"", asset.Spread, "", nil},
	} {
		t.Run(fmt.Sprintf("account level %q %s", tc.accountLevel, tc.assetType), func(t *testing.T) {
			t.Parallel()
			var fetches atomic.Int32
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fetches.Add(1)
				if r.URL.Path != "/account/config" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				writeOKXData(t, w, []map[string]string{{"acctLv": tc.accountLevel, "posMode": positionModeNet}})
			}))
			tradeMode, err := ex.defaultTradeMode(t.Context(), tc.assetType)
			require.ErrorIs(t, err, tc.err, "defaultTradeMode must return the expected error")
			assert.Equal(t, tc.exp, tradeMode, "defaultTradeMode should return the trade mode OKX documents for the account mode")
			var expFetches int32
			if tc.assetType == asset.Spot || tc.assetType == asset.Options {
				expFetches = 1
			}
			assert.Equal(t, expFetches, fetches.Load(), "defaultTradeMode should look the account level up for spot and option orders alone")
		})
	}
}

func TestDeriveSpreadOrderParam(t *testing.T) {
	t.Parallel()
	p := &submitPrelude{pairString: spreadPair.String(), sideType: order.Buy.String()}
	spreadOrder := func(orderType string, price float64) *PlaceSpreadOrderRequest {
		return &PlaceSpreadOrderRequest{SpreadID: "BTC-USDT_BTC-USDT-SWAP", ClientOrderID: "spread1", Side: "BUY", OrderType: orderType, Size: 2, Price: price}
	}
	for _, tc := range []struct {
		name string
		s    *order.Submit
		exp  *PlaceSpreadOrderRequest
		err  error
	}{
		{name: "limit", s: &order.Submit{Type: order.Limit, Amount: 2, Price: -8.7, ClientOrderID: "spread1"}, exp: spreadOrder(orderLimit, -8.7)},
		{name: "limit at a zero price", s: &order.Submit{Type: order.Limit, Amount: 2, ClientOrderID: "spread1"}, exp: spreadOrder(orderLimit, 0)},
		{name: "post only", s: &order.Submit{Type: order.Limit, TimeInForce: order.PostOnly, Amount: 2, Price: 8.7, ClientOrderID: "spread1"}, exp: spreadOrder(orderPostOnly, 8.7)},
		{name: "limit maker", s: &order.Submit{Type: order.LimitMaker, Amount: 2, Price: 8.7, ClientOrderID: "spread1"}, exp: spreadOrder(orderPostOnly, 8.7)},
		{name: "immediate or cancel", s: &order.Submit{Type: order.Limit, TimeInForce: order.ImmediateOrCancel, Amount: 2, Price: 8.7, ClientOrderID: "spread1"}, exp: spreadOrder(orderIOC, 8.7)},
		{name: "market", s: &order.Submit{Type: order.Market, Amount: 2, ClientOrderID: "spread1"}, exp: spreadOrder(orderMarket, 0)},
		{name: "fill or kill", s: &order.Submit{Type: order.Limit, TimeInForce: order.FillOrKill, Amount: 2, Price: 8.7}, err: order.ErrUnsupportedOrderType},
		{name: "order type the spread book does not list", s: &order.Submit{Type: order.Trigger, Amount: 2, Price: 8.7}, err: order.ErrUnsupportedOrderType},
		{name: "post only market", s: &order.Submit{Type: order.Market, TimeInForce: order.PostOnly, Amount: 2}, err: order.ErrUnsupportedTimeInForce},
		{name: "take profit", s: &order.Submit{Type: order.Limit, Amount: 2, Price: 8.7, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 9}}}, err: errAttachedOrderUnsupported},
		{name: "stop entry", s: &order.Submit{Type: order.Limit, Amount: 2, Price: 8.7, RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 9}}}, err: errAttachedOrderUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			param, err := deriveSpreadOrderParam(tc.s, p)
			require.ErrorIs(t, err, tc.err, "deriveSpreadOrderParam must return the expected error")
			assert.Equal(t, tc.exp, param, "deriveSpreadOrderParam should place the submit on the prelude's spread")
		})
	}
}

func TestPlacesMarketOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    bool
	}{
		{"market", &order.Submit{AssetType: asset.Margin, Type: order.Market}, true},
		{"market good till cancel", &order.Submit{AssetType: asset.Margin, Type: order.Market, TimeInForce: order.GoodTillCancel}, true},
		{"market immediate or cancel", &order.Submit{AssetType: asset.Margin, Type: order.Market, TimeInForce: order.ImmediateOrCancel}, false},
		{"market fill or kill", &order.Submit{AssetType: asset.Margin, Type: order.Market, TimeInForce: order.FillOrKill}, false},
		{"limit", &order.Submit{AssetType: asset.Margin, Type: order.Limit}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, placesMarketOrder(tc.submit), "placesMarketOrder should report whether the submit places an OKX market order")
		})
	}
}

func TestPlacesSpotMarketOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    bool
	}{
		{"spot market", &order.Submit{AssetType: asset.Spot, Type: order.Market}, true},
		{"spot market in cross mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi, Type: order.Market}, false},
		{"spot market in spot lead trading", &order.Submit{AssetType: asset.Spot, MarginType: margin.SpotIsolated, Type: order.Market}, true},
		{"spot market immediate or cancel", &order.Submit{AssetType: asset.Spot, Type: order.Market, TimeInForce: order.ImmediateOrCancel}, false},
		{"spot limit", &order.Submit{AssetType: asset.Spot, Type: order.Limit}, false},
		{"margin market", &order.Submit{AssetType: asset.Margin, Type: order.Market}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, placesSpotMarketOrder(tc.submit), "placesSpotMarketOrder should report whether the submit places a spot market order")
		})
	}
}

func TestSpotMarginOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    bool
	}{
		{"spot in cross mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi}, true},
		{"spot in isolated mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.Isolated}, true},
		{"spot without a margin type", &order.Submit{AssetType: asset.Spot}, false},
		{"spot in cash mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.NoMargin}, false},
		{"spot lead trading", &order.Submit{AssetType: asset.Spot, MarginType: margin.SpotIsolated}, false},
		{"spot with combined margin types", &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi | margin.Isolated}, false},
		{"margin asset in cross mode", &order.Submit{AssetType: asset.Margin, MarginType: margin.Multi}, false},
		{"perpetual swap in isolated mode", &order.Submit{AssetType: asset.PerpetualSwap, MarginType: margin.Isolated}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, spotMarginOrder(tc.submit), "spotMarginOrder should report whether a spot asset submit places a MARGIN order")
		})
	}
}

func TestQuoteSizedOnly(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    bool
	}{
		{"margin market buy", &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Buy}, true},
		{"margin market sell", &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Sell}, false},
		{"margin market buy immediate or cancel", &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Buy, TimeInForce: order.ImmediateOrCancel}, false},
		{"margin limit buy", &order.Submit{AssetType: asset.Margin, Type: order.Limit, Side: order.Buy}, false},
		{"margin trailing stop buy", &order.Submit{AssetType: asset.Margin, Type: order.TrailingStop, Side: order.Buy}, true},
		{"margin trailing stop sell", &order.Submit{AssetType: asset.Margin, Type: order.TrailingStop, Side: order.Sell}, false},
		{"spot trailing stop buy", &order.Submit{AssetType: asset.Spot, Type: order.TrailingStop, Side: order.Buy}, true},
		{"spot trailing stop sell", &order.Submit{AssetType: asset.Spot, Type: order.TrailingStop, Side: order.Sell}, false},
		{"spot market buy", &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Buy}, false},
		{"spot market buy in cross mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi, Type: order.Market, Side: order.Buy}, true},
		{"spot market sell in isolated mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.Isolated, Type: order.Market, Side: order.Sell}, false},
		{"spot limit buy in cross mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi, Type: order.Limit, Side: order.Buy}, false},
		{"perpetual swap market buy", &order.Submit{AssetType: asset.PerpetualSwap, Type: order.Market, Side: order.Buy}, false},
		{"spot trigger buy at market", &order.Submit{AssetType: asset.Spot, Type: order.Trigger, Side: order.Buy, Price: marketOrderPrice}, true},
		{"spot trigger long at market", &order.Submit{AssetType: asset.Spot, Type: order.Trigger, Side: order.Long, Price: marketOrderPrice}, true},
		{"spot trigger buy at a limit price", &order.Submit{AssetType: asset.Spot, Type: order.Trigger, Side: order.Buy, Price: 100}, false},
		{"spot trigger sell at market", &order.Submit{AssetType: asset.Spot, Type: order.Trigger, Side: order.Sell, Price: marketOrderPrice}, false},
		{"margin trigger buy at market", &order.Submit{AssetType: asset.Margin, Type: order.Trigger, Side: order.Buy, Price: marketOrderPrice}, true},
		{"perpetual swap trigger buy at market", &order.Submit{AssetType: asset.PerpetualSwap, Type: order.Trigger, Side: order.Buy, Price: marketOrderPrice}, false},
		{"spot conditional buy at market", &order.Submit{AssetType: asset.Spot, Type: order.ConditionalStop, Side: order.Buy, TriggerPrice: 110}, false},
		{"spot conditional buy at market in cross mode", &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi, Type: order.ConditionalStop, Side: order.Buy, TriggerPrice: 110}, true},
		{"margin conditional buy at market", &order.Submit{AssetType: asset.Margin, Type: order.ConditionalStop, Side: order.Buy, TriggerPrice: 110, Price: marketOrderPrice}, true},
		{"margin conditional buy at a limit price", &order.Submit{AssetType: asset.Margin, Type: order.ConditionalStop, Side: order.Buy, TriggerPrice: 110, Price: 111}, false},
		{"margin conditional sell at market", &order.Submit{AssetType: asset.Margin, Type: order.ConditionalStop, Side: order.Sell, TriggerPrice: 90}, false},
		{"spot oco buy with both legs at market", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Buy, RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 90}, StopLoss: order.RiskManagement{Price: 110, LimitPrice: marketOrderPrice},
		}}, true},
		{"spot oco buy with a leg at market", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Buy, RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 90, LimitPrice: 89}, StopLoss: order.RiskManagement{Price: 110},
		}}, false},
		{"margin oco buy with both legs at market", &order.Submit{AssetType: asset.Margin, Type: order.OCO, Side: order.Buy, RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 90}, StopLoss: order.RiskManagement{Price: 110},
		}}, true},
		{"spot oco sell with both legs at market", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Sell, RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90},
		}}, false},
		{"perpetual swap oco buy with both legs at market", &order.Submit{AssetType: asset.PerpetualSwap, Type: order.OCO, Side: order.Buy, RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 90}, StopLoss: order.RiskManagement{Price: 110},
		}}, false},
		{"spot oco buy with limit legs missing their limit prices", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Buy, RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 90, OrderType: order.Limit}, StopLoss: order.RiskManagement{Price: 110, OrderType: order.Limit},
		}}, false},
		{"spot oco buy with market legs", &order.Submit{AssetType: asset.Spot, Type: order.OCO, Side: order.Buy, RiskManagementModes: order.RiskManagementModes{
			TakeProfit: order.RiskManagement{Price: 90, OrderType: order.Market}, StopLoss: order.RiskManagement{Price: 110, OrderType: order.Market},
		}}, true},
		{"spot twap buy", &order.Submit{AssetType: asset.Spot, Type: order.TWAP, Side: order.Buy, Price: 100}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, quoteSizedOnly(tc.submit), "quoteSizedOnly should report whether OKX reads the size in the quote currency alone")
		})
	}
}

func TestOcoLegsAtMarket(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		modes                order.RiskManagementModes
		expTakeProfit, expSL bool
	}{
		{"legs with trigger prices alone", order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90}}, true, true},
		{"legs at market", order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110, LimitPrice: marketOrderPrice}, StopLoss: order.RiskManagement{Price: 90, LimitPrice: marketOrderPrice}}, true, true},
		{"a limited take profit", order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110, LimitPrice: 111}, StopLoss: order.RiskManagement{Price: 90}}, false, true},
		{"a limited stop loss", order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90, LimitPrice: 89}}, true, false},
		{"no legs", order.RiskManagementModes{}, false, false},
		{"market legs", order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110, OrderType: order.Market}, StopLoss: order.RiskManagement{Price: 90, OrderType: order.Market}}, true, true},
		{"limit legs without limit prices", order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110, OrderType: order.Limit}, StopLoss: order.RiskManagement{Price: 90, OrderType: order.Limit}}, false, false},
		{"a market take profit with a limit price", order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110, OrderType: order.Market, LimitPrice: 111}, StopLoss: order.RiskManagement{Price: 90}}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			takeProfit, stopLoss := ocoLegsAtMarket(&order.Submit{RiskManagementModes: tc.modes})
			assert.Equal(t, tc.expTakeProfit, takeProfit, "ocoLegsAtMarket should report whether the take profit leg executes at market")
			assert.Equal(t, tc.expSL, stopLoss, "ocoLegsAtMarket should report whether the stop loss leg executes at market")
		})
	}
}

func TestAlgoOrderSize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    float64
	}{
		{"spot trigger buy at market", &order.Submit{AssetType: asset.Spot, Type: order.Trigger, Side: order.Buy, Price: marketOrderPrice, Amount: 0.5, QuoteAmount: 100}, 100},
		{"spot trigger sell at market", &order.Submit{AssetType: asset.Spot, Type: order.Trigger, Side: order.Sell, Price: marketOrderPrice, Amount: 0.5, QuoteAmount: 100}, 0.5},
		{"spot trigger buy at a limit price", &order.Submit{AssetType: asset.Spot, Type: order.Trigger, Side: order.Buy, Price: 100, Amount: 0.5, QuoteAmount: 100}, 0.5},
		{"perpetual swap trigger buy at market", &order.Submit{AssetType: asset.PerpetualSwap, Type: order.Trigger, Side: order.Buy, Price: marketOrderPrice, Amount: 2, QuoteAmount: 100}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, algoOrderSize(tc.submit), "algoOrderSize should return the size in the currency OKX reads")
		})
	}
}

func TestDerivePlaceOrderRequest(t *testing.T) {
	t.Parallel()
	spot := &submitPrelude{pairString: mainPair.String(), tradeMode: TradeModeCash, sideType: "BUY"}
	spotSell := &submitPrelude{pairString: mainPair.String(), tradeMode: TradeModeCash, sideType: "SELL"}
	crossMargin := &submitPrelude{pairString: mainPair.String(), tradeMode: TradeModeCross, sideType: "BUY"}
	isolatedMargin := &submitPrelude{pairString: mainPair.String(), tradeMode: TradeModeIsolated, sideType: "SELL", marginCurrency: currency.BTC}
	netContract := &submitPrelude{pairString: perpetualSwapPair.String(), tradeMode: TradeModeCross, sideType: "sell", positionMode: positionModeNet}
	longShortContract := &submitPrelude{pairString: perpetualSwapPair.String(), tradeMode: TradeModeCross, sideType: "sell", positionMode: positionModeLongShort}
	option := &submitPrelude{pairString: "BTC-USD-261225-100000-C", tradeMode: TradeModeCross, sideType: "buy"}
	spotCrossSell := &submitPrelude{pairString: mainPair.String(), tradeMode: TradeModeCross, sideType: "SELL"}
	spotLeadTradingSell := &submitPrelude{pairString: mainPair.String(), tradeMode: TradeModeSpotIsolated, sideType: "SELL"}
	for _, tc := range []struct {
		name  string
		s     *order.Submit
		p     *submitPrelude
		oType string
		exp   *PlaceOrderRequest
	}{
		{
			name: "spot cross margin sell sends reduce only", s: &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi, Side: order.Sell, Type: order.Limit, Amount: 2, Price: 100, ReduceOnly: true}, p: spotCrossSell, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCross, Side: "SELL", OrderType: orderLimit, Size: 2, Price: 100, ReduceOnly: true},
		},
		{
			name: "spot isolated margin sell sends reduce only", s: &order.Submit{AssetType: asset.Spot, MarginType: margin.Isolated, Side: order.Sell, Type: order.Market, Amount: 2, ReduceOnly: true}, p: isolatedMargin, oType: orderMarket,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeIsolated, Currency: currency.BTC, Side: "SELL", OrderType: orderMarket, Size: 2, ReduceOnly: true},
		},
		{
			name: "spot lead trading sell ignores reduce only", s: &order.Submit{AssetType: asset.Spot, MarginType: margin.SpotIsolated, Side: order.Sell, Type: order.Limit, Amount: 2, Price: 100, ReduceOnly: true}, p: spotLeadTradingSell, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeSpotIsolated, Side: "SELL", OrderType: orderLimit, Size: 2, Price: 100},
		},
		{
			name: "spot cross margin sell without reduce only", s: &order.Submit{AssetType: asset.Spot, MarginType: margin.Multi, Side: order.Sell, Type: order.Limit, Amount: 2, Price: 100}, p: spotCrossSell, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCross, Side: "SELL", OrderType: orderLimit, Size: 2, Price: 100},
		},
		{
			name: "limit", s: &order.Submit{AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 2, Price: 100, ClientOrderID: "client1"}, p: spot, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "BUY", OrderType: orderLimit, Size: 2, Price: 100, ClientOrderID: "client1"},
		},
		{
			name: "post only", s: &order.Submit{AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.PostOnly, Amount: 2, Price: 100}, p: spot, oType: orderPostOnly,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "BUY", OrderType: orderPostOnly, Size: 2, Price: 100},
		},
		{
			name: "fill or kill", s: &order.Submit{AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.FillOrKill, Amount: 2, Price: 100}, p: spot, oType: orderFOK,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "BUY", OrderType: orderFOK, Size: 2, Price: 100},
		},
		{
			name: "immediate or cancel", s: &order.Submit{AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.ImmediateOrCancel, Amount: 2, Price: 100}, p: spot, oType: orderIOC,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "BUY", OrderType: orderIOC, Size: 2, Price: 100},
		},
		{
			name: "market maker protection", s: &order.Submit{AssetType: asset.Options, Side: order.Buy, Type: order.MarketMakerProtection, Amount: 1, Price: 0.05}, p: option, oType: orderMarketMakerProtection,
			exp: &PlaceOrderRequest{AssetType: asset.Options, InstrumentID: "BTC-USD-261225-100000-C", TradeMode: TradeModeCross, Side: "buy", OrderType: orderMarketMakerProtection, Size: 1, Price: 0.05},
		},
		{
			name: "market maker protection and post only", s: &order.Submit{AssetType: asset.Options, Side: order.Buy, Type: order.MarketMakerProtection, TimeInForce: order.PostOnly, Amount: 1, Price: 0.05}, p: option, oType: orderMarketMakerProtectionAndPostOnly,
			exp: &PlaceOrderRequest{AssetType: asset.Options, InstrumentID: "BTC-USD-261225-100000-C", TradeMode: TradeModeCross, Side: "buy", OrderType: orderMarketMakerProtectionAndPostOnly, Size: 1, Price: 0.05},
		},
		{
			name: "optimal limit takes no price", s: &order.Submit{AssetType: asset.PerpetualSwap, Side: order.Sell, Type: order.OptimalLimit, Amount: 1, Price: 95000}, p: netContract, oType: orderOptimalLimitIOC,
			exp: &PlaceOrderRequest{AssetType: asset.PerpetualSwap, InstrumentID: "BTC-USDT-SWAP", TradeMode: TradeModeCross, Side: "sell", PositionSide: positionSideNet, OrderType: orderOptimalLimitIOC, Size: 1},
		},
		{
			name: "spot market sized in the base currency takes no price", s: &order.Submit{AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: 2, Price: 100}, p: spot, oType: orderMarket,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "BUY", OrderType: orderMarket, Size: 2, TargetCurrency: targetCurrencyBase},
		},
		{
			name: "spot market sized in the quote currency", s: &order.Submit{AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 200}, p: spot, oType: orderMarket,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "BUY", OrderType: orderMarket, Size: 200, TargetCurrency: targetCurrencyQuote},
		},
		{
			name: "spot sell ignores reduce only", s: &order.Submit{AssetType: asset.Spot, Side: order.Sell, Type: order.Limit, Amount: 2, Price: 100, ReduceOnly: true}, p: spotSell, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", TradeMode: TradeModeCash, Side: "SELL", OrderType: orderLimit, Size: 2, Price: 100},
		},
		{
			name: "margin market buy sized in the quote currency", s: &order.Submit{AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 2, QuoteAmount: 200}, p: crossMargin, oType: orderMarket,
			exp: &PlaceOrderRequest{AssetType: asset.Margin, InstrumentID: "BTC-USDT", TradeMode: TradeModeCross, Side: "BUY", OrderType: orderMarket, Size: 200},
		},
		{
			name: "isolated margin sell sends its margin currency and reduce only", s: &order.Submit{AssetType: asset.Margin, Side: order.Sell, Type: order.Market, Amount: 2, ReduceOnly: true}, p: isolatedMargin, oType: orderMarket,
			exp: &PlaceOrderRequest{AssetType: asset.Margin, InstrumentID: "BTC-USDT", TradeMode: TradeModeIsolated, Currency: currency.BTC, Side: "SELL", OrderType: orderMarket, Size: 2, ReduceOnly: true},
		},
		{
			name: "net mode close", s: &order.Submit{AssetType: asset.PerpetualSwap, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 95000, ReduceOnly: true}, p: netContract, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.PerpetualSwap, InstrumentID: "BTC-USDT-SWAP", TradeMode: TradeModeCross, Side: "sell", PositionSide: positionSideNet, OrderType: orderLimit, Size: 1, Price: 95000, ReduceOnly: true},
		},
		{
			name: "net mode open", s: &order.Submit{AssetType: asset.Futures, Side: order.Sell, Type: order.Market, Amount: 1}, p: netContract, oType: orderMarket,
			exp: &PlaceOrderRequest{AssetType: asset.Futures, InstrumentID: "BTC-USDT-SWAP", TradeMode: TradeModeCross, Side: "sell", PositionSide: positionSideNet, OrderType: orderMarket, Size: 1},
		},
		{
			name: "long short mode close", s: &order.Submit{AssetType: asset.PerpetualSwap, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 95000, ReduceOnly: true}, p: longShortContract, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.PerpetualSwap, InstrumentID: "BTC-USDT-SWAP", TradeMode: TradeModeCross, Side: "sell", PositionSide: positionSideLong, OrderType: orderLimit, Size: 1, Price: 95000},
		},
		{
			name: "long short mode open", s: &order.Submit{AssetType: asset.Futures, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 95000}, p: longShortContract, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.Futures, InstrumentID: "BTC-USDT-SWAP", TradeMode: TradeModeCross, Side: "sell", PositionSide: positionSideShort, OrderType: orderLimit, Size: 1, Price: 95000},
		},
		{
			name: "option ignores reduce only", s: &order.Submit{AssetType: asset.Options, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 0.05, ReduceOnly: true}, p: option, oType: orderLimit,
			exp: &PlaceOrderRequest{AssetType: asset.Options, InstrumentID: "BTC-USD-261225-100000-C", TradeMode: TradeModeCross, Side: "buy", OrderType: orderLimit, Size: 1, Price: 0.05},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, derivePlaceOrderRequest(tc.s, tc.p, tc.oType), "derivePlaceOrderRequest should place the submit with the prelude's values")
		})
	}
}

func TestLegOrderPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                     string
		triggerPrice, limitPrice float64
		exp                      float64
	}{
		{"trigger price alone", 110, 0, marketOrderPrice},
		{"trigger and limit price", 110, 111, 111},
		{"trigger price at market", 110, marketOrderPrice, marketOrderPrice},
		{"limit price alone", 0, 111, 111},
		{"no prices", 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, legOrderPrice(tc.triggerPrice, tc.limitPrice), "legOrderPrice should return the order price the leg sends")
		})
	}
}

func TestSpotAlgoTargetCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		submit    *order.Submit
		marketLeg bool
		exp       string
	}{
		{name: "spot buy at market", submit: &order.Submit{AssetType: asset.Spot, Side: order.Buy}, marketLeg: true, exp: targetCurrencyBase},
		{name: "spot bid at market", submit: &order.Submit{AssetType: asset.Spot, Side: order.Bid}, marketLeg: true, exp: targetCurrencyBase},
		{name: "spot cash buy at market", submit: &order.Submit{AssetType: asset.Spot, Side: order.Buy, MarginType: margin.NoMargin}, marketLeg: true, exp: targetCurrencyBase},
		{name: "spot lead trading buy at market", submit: &order.Submit{AssetType: asset.Spot, Side: order.Buy, MarginType: margin.SpotIsolated}, marketLeg: true, exp: targetCurrencyBase},
		{name: "spot buy at a limit price", submit: &order.Submit{AssetType: asset.Spot, Side: order.Buy}},
		{name: "spot sell at market", submit: &order.Submit{AssetType: asset.Spot, Side: order.Sell}, marketLeg: true},
		{name: "spot cross margin buy at market", submit: &order.Submit{AssetType: asset.Spot, Side: order.Buy, MarginType: margin.Multi}, marketLeg: true},
		{name: "spot isolated margin buy at market", submit: &order.Submit{AssetType: asset.Spot, Side: order.Buy, MarginType: margin.Isolated}, marketLeg: true},
		{name: "margin buy at market", submit: &order.Submit{AssetType: asset.Margin, Side: order.Buy}, marketLeg: true},
		{name: "perpetual swap buy at market", submit: &order.Submit{AssetType: asset.PerpetualSwap, Side: order.Buy}, marketLeg: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, spotAlgoTargetCurrency(tc.submit, tc.marketLeg), "spotAlgoTargetCurrency should name base_ccy for a SPOT buy at market only")
		})
	}
}

func TestRiskManagementSet(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		modes order.RiskManagementModes
		exp   bool
	}{
		{"nothing set", order.RiskManagementModes{}, false},
		{"take profit enabled", order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true}}, true},
		{"stop loss trigger price", order.RiskManagementModes{StopLoss: order.RiskManagement{Price: 90}}, true},
		{"stop loss limit price", order.RiskManagementModes{StopLoss: order.RiskManagement{LimitPrice: 89}}, true},
		{"take profit order type", order.RiskManagementModes{TakeProfit: order.RiskManagement{OrderType: order.Market}}, true},
		{"take profit trigger price type", order.RiskManagementModes{TakeProfit: order.RiskManagement{TriggerPriceType: order.MarkPrice}}, true},
		{"stop entry", order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 100}}, true},
		{"mode", order.RiskManagementModes{Mode: "partial"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, riskManagementSet(&tc.modes), "riskManagementSet should report whether any value is set")
		})
	}
}

func TestCheckRiskManagementExtras(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkRiskManagementExtras(&order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90}}), "checkRiskManagementExtras should accept take profit and stop loss legs")
	assert.ErrorIs(t, checkRiskManagementExtras(&order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true}}), errAttachedOrderUnsupported, "checkRiskManagementExtras should refuse a stop entry order")
	assert.ErrorIs(t, checkRiskManagementExtras(&order.RiskManagementModes{Mode: "full"}), errAttachedOrderUnsupported, "checkRiskManagementExtras should refuse a take profit and stop loss mode")
}

func TestAttachedLegOrderPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		leg  order.RiskManagement
		exp  float64
		err  error
	}{
		{name: "trigger price alone", leg: order.RiskManagement{Price: 110}, exp: marketOrderPrice},
		{name: "limit price", leg: order.RiskManagement{Price: 110, LimitPrice: 111}, exp: 111},
		{name: "limit price at market", leg: order.RiskManagement{Price: 110, LimitPrice: marketOrderPrice}, exp: marketOrderPrice},
		{name: "market leg", leg: order.RiskManagement{Price: 110, OrderType: order.Market}, exp: marketOrderPrice},
		{name: "market leg at market", leg: order.RiskManagement{Price: 110, OrderType: order.Market, LimitPrice: marketOrderPrice}, exp: marketOrderPrice},
		{name: "limit leg", leg: order.RiskManagement{Price: 110, OrderType: order.Limit, LimitPrice: 111}, exp: 111},
		{name: "no trigger price", leg: order.RiskManagement{LimitPrice: 111}, err: limits.ErrPriceBelowMin},
		{name: "negative trigger price", leg: order.RiskManagement{Price: -1, LimitPrice: 111}, err: limits.ErrPriceBelowMin},
		{name: "market leg with a limit price", leg: order.RiskManagement{Price: 110, OrderType: order.Market, LimitPrice: 111}, err: order.ErrUnsupportedOrderType},
		{name: "limit leg without a limit price", leg: order.RiskManagement{Price: 110, OrderType: order.Limit}, err: limits.ErrPriceBelowMin},
		{name: "limit leg at market", leg: order.RiskManagement{Price: 110, OrderType: order.Limit, LimitPrice: marketOrderPrice}, err: limits.ErrPriceBelowMin},
		{name: "negative limit price", leg: order.RiskManagement{Price: 110, LimitPrice: -2}, err: limits.ErrPriceBelowMin},
		{name: "unsupported order type", leg: order.RiskManagement{Price: 110, OrderType: order.Stop}, err: order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orderPrice, err := attachedLegOrderPrice(&tc.leg)
			require.ErrorIs(t, err, tc.err, "attachedLegOrderPrice must return the expected error")
			assert.Equal(t, tc.exp, orderPrice, "attachedLegOrderPrice should return the price the leg is placed at")
		})
	}
}

func TestAttachedAlgoOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		modes order.RiskManagementModes
		exp   *AttachAlgoOrderRequest
		err   error
	}{
		{name: "nothing attached"},
		{
			name:  "both legs",
			modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 120, LimitPrice: 121, TriggerPriceType: order.IndexPrice}, StopLoss: order.RiskManagement{Enabled: true, Price: 90, OrderType: order.Market, TriggerPriceType: order.MarkPrice}},
			exp:   &AttachAlgoOrderRequest{TakeProfitTriggerPrice: 120, TakeProfitOrderPrice: 121, TakeProfitTriggerPriceType: "index", StopLossTriggerPrice: 90, StopLossOrderPrice: marketOrderPrice, StopLossTriggerPriceType: "mark"},
		},
		{
			name:  "take profit alone",
			modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 120}},
			exp:   &AttachAlgoOrderRequest{TakeProfitTriggerPrice: 120, TakeProfitOrderPrice: marketOrderPrice, TakeProfitTriggerPriceType: "last"},
		},
		{
			name:  "stop loss alone",
			modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Price: 90, OrderType: order.Limit, LimitPrice: 89}},
			exp:   &AttachAlgoOrderRequest{StopLossTriggerPrice: 90, StopLossOrderPrice: 89, StopLossTriggerPriceType: "last"},
		},
		{name: "take profit without a trigger price", modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true}, StopLoss: order.RiskManagement{Price: 90}}, err: limits.ErrPriceBelowMin},
		{name: "stop loss without a trigger price", modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 120}, StopLoss: order.RiskManagement{LimitPrice: 89}}, err: limits.ErrPriceBelowMin},
		{name: "stop entry", modes: order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 100}}, err: errAttachedOrderUnsupported},
		{name: "mode", modes: order.RiskManagementModes{Mode: "full", TakeProfit: order.RiskManagement{Price: 120}}, err: errAttachedOrderUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			attached, err := attachedAlgoOrder(&tc.modes)
			require.ErrorIs(t, err, tc.err, "attachedAlgoOrder must return the expected error")
			assert.Equal(t, tc.exp, attached, "attachedAlgoOrder should attach the legs set")
		})
	}
}

func TestAmendedAttachedAlgoOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		modes order.RiskManagementModes
		exp   *AmendAttachedAlgoOrderRequest
		err   error
	}{
		{name: "nothing amended"},
		{
			name:  "both legs",
			modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 120, LimitPrice: 121}, StopLoss: order.RiskManagement{Price: 90, TriggerPriceType: order.MarkPrice}},
			exp: &AmendAttachedAlgoOrderRequest{
				NewTakeProfitTriggerPrice: 120, NewTakeProfitOrderPrice: 121, NewTakeProfitTriggerPriceType: "last",
				NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: marketOrderPrice, NewStopLossTriggerPriceType: "mark",
			},
		},
		{
			name:  "stop loss alone",
			modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Price: 90}},
			exp:   &AmendAttachedAlgoOrderRequest{NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: marketOrderPrice, NewStopLossTriggerPriceType: "last"},
		},
		{name: "leg without a trigger price", modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true}}, err: limits.ErrPriceBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			attached, err := amendedAttachedAlgoOrder(&tc.modes)
			require.ErrorIs(t, err, tc.err, "amendedAttachedAlgoOrder must return the expected error")
			assert.Equal(t, tc.exp, attached, "amendedAttachedAlgoOrder should amend the legs set")
		})
	}
}

func TestCheckTriggerAttachmentAccountLevel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		accountLevel string
		err          error
	}{
		{"1", errAttachedOrderUnsupported},
		{"2", nil},
		{"3", nil},
		{"4", nil},
		{"", errInvalidAccountLevel},
	} {
		t.Run(fmt.Sprintf("account level %q", tc.accountLevel), func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/account/config" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				writeOKXData(t, w, []map[string]string{{"acctLv": tc.accountLevel, "posMode": positionModeNet}})
			}))
			assert.ErrorIs(t, ex.checkTriggerAttachmentAccountLevel(t.Context()), tc.err, "checkTriggerAttachmentAccountLevel should refuse a trigger order's attachments in Spot mode alone")
		})
	}
}

func TestRefuseRiskManagement(t *testing.T) {
	t.Parallel()
	assert.NoError(t, refuseRiskManagement(&order.RiskManagementModes{}, order.TWAP), "refuseRiskManagement should accept an order setting no take profit or stop loss")
	assert.ErrorIs(t, refuseRiskManagement(&order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110}}, order.TWAP), errAttachedOrderUnsupported, "refuseRiskManagement should refuse a take profit")
	assert.ErrorIs(t, refuseRiskManagement(&order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 100}}, order.Chase), errAttachedOrderUnsupported, "refuseRiskManagement should refuse a stop entry")
}

func TestDeriveAmendSpreadOrderRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		exp    *AmendSpreadOrderRequest
		err    error
	}{
		{
			name:   "price and size",
			modify: &order.Modify{OrderID: "1302", Price: -5, Amount: 2},
			exp:    &AmendSpreadOrderRequest{OrderID: "1302", NewSize: 2, NewPrice: new(-5.0)},
		},
		{
			name:   "size alone leaves the price",
			modify: &order.Modify{ClientOrderID: "client1302", Amount: 2},
			exp:    &AmendSpreadOrderRequest{ClientOrderID: "client1302", NewSize: 2},
		},
		{
			name:   "price alone",
			modify: &order.Modify{OrderID: "1302", Price: 3.5},
			exp:    &AmendSpreadOrderRequest{OrderID: "1302", NewPrice: new(3.5)},
		},
		{name: "stop loss", modify: &order.Modify{OrderID: "1302", Price: -5, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Price: -6}}}, err: errAttachedOrderUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			amendRequest, err := deriveAmendSpreadOrderRequest(tc.modify)
			require.ErrorIs(t, err, tc.err, "deriveAmendSpreadOrderRequest must return the expected error")
			assert.Equal(t, tc.exp, amendRequest, "deriveAmendSpreadOrderRequest should amend only what the modification sets")
		})
	}
}

func TestPositionSideForMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		side       order.Side
		reduceOnly bool
		mode       string
		exp        string
	}{
		{name: "net mode buy", side: order.Buy, mode: positionModeNet, exp: positionSideNet},
		{name: "net mode sell to close", side: order.Sell, reduceOnly: true, mode: positionModeNet, exp: positionSideNet},
		{name: "unknown mode", side: order.Buy, exp: positionSideNet},
		{name: "long short mode buy to open", side: order.Buy, mode: positionModeLongShort, exp: positionSideLong},
		{name: "long short mode sell to open", side: order.Sell, mode: positionModeLongShort, exp: positionSideShort},
		{name: "long short mode buy to close", side: order.Buy, reduceOnly: true, mode: positionModeLongShort, exp: positionSideShort},
		{name: "long short mode sell to close", side: order.Sell, reduceOnly: true, mode: positionModeLongShort, exp: positionSideLong},
		{name: "long short mode long to open", side: order.Long, mode: positionModeLongShort, exp: positionSideLong},
		{name: "long short mode short to close", side: order.Short, reduceOnly: true, mode: positionModeLongShort, exp: positionSideLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, positionSideForMode(tc.side, tc.reduceOnly, tc.mode), "positionSideForMode should return the posSide the position mode requires")
		})
	}
}

func TestAmendResponse(t *testing.T) {
	t.Parallel()
	_, err := amendResponse(&order.Modify{Type: order.Limit, ClientOrderID: "client1301"}, &orderOutcome{clientOrderID: "client1301"})
	require.ErrorIs(t, err, common.ErrInvalidResponse, "amendResponse must reject an amendment naming no order")
	_, err = amendResponse(&order.Modify{Type: order.Limit, OrderID: "1301"}, &orderOutcome{orderID: "9999"})
	require.ErrorIs(t, err, errReplyOrderMismatch, "amendResponse must reject a reply naming another order")

	for _, tc := range []struct {
		name   string
		modify *order.Modify
		row    *orderOutcome
		exp    *order.ModifyResponse
	}{
		{
			name:   "limit by order ID",
			modify: &order.Modify{Exchange: "Okx", Pair: mainPair, AssetType: asset.Spot, Type: order.Limit, Side: order.Sell, OrderID: "1301", TimeInForce: order.PostOnly, Price: 101, Amount: 1, TriggerPrice: 5},
			row:    &orderOutcome{orderID: "1301"},
			exp:    &order.ModifyResponse{Exchange: "Okx", OrderID: "1301", Pair: mainPair, Price: 101, Amount: 1},
		},
		{
			name:   "limit by client order ID",
			modify: &order.Modify{Exchange: "Okx", Pair: mainPair, AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, ClientOrderID: "client1301", Amount: 1},
			row:    &orderOutcome{orderID: "1301", clientOrderID: "client1301"},
			exp:    &order.ModifyResponse{Exchange: "Okx", OrderID: "1301", ClientOrderID: "client1301", Pair: mainPair, Amount: 1},
		},
		{
			name:   "conditional",
			modify: &order.Modify{Exchange: "Okx", Pair: mainPair, AssetType: asset.Margin, Type: order.ConditionalStop, Side: order.Sell, OrderID: "803", TimeInForce: order.PostOnly, Price: 89, Amount: 2, TriggerPrice: 90},
			row:    &orderOutcome{orderID: "803"},
			exp:    &order.ModifyResponse{Exchange: "Okx", OrderID: "803", Pair: mainPair, Price: 89, Amount: 2, TriggerPrice: 90},
		},
		{
			name:   "oco",
			modify: &order.Modify{Exchange: "Okx", Pair: mainPair, AssetType: asset.Spot, Type: order.OCO, Side: order.Buy, OrderID: "803", Price: 7, Amount: 2, TriggerPrice: 5},
			row:    &orderOutcome{orderID: "803"},
			exp:    &order.ModifyResponse{Exchange: "Okx", OrderID: "803", Pair: mainPair, Amount: 2},
		},
		{
			name:   "spread",
			modify: &order.Modify{Exchange: "Okx", Pair: spreadPair, AssetType: asset.Spread, Type: order.Limit, Side: order.Sell, OrderID: "1302", TimeInForce: order.PostOnly, Price: -5, Amount: 1, TriggerPrice: 5},
			row:    &orderOutcome{orderID: "1302"},
			exp:    &order.ModifyResponse{Exchange: "Okx", OrderID: "1302", Price: -5, Amount: 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := amendResponse(tc.modify, tc.row)
			require.NoError(t, err, "amendResponse must not error")
			assert.Equal(t, tc.exp, resp, "amendResponse should claim only what the amendment sent")
		})
	}
}

func TestOrderActionOutcome(t *testing.T) {
	t.Parallel()
	assert.Nil(t, orderActionOutcome(nil), "orderActionOutcome should report nothing for no row")
	for _, tc := range []struct {
		name string
		row  *OrderActionResponse
		exp  *orderOutcome
	}{
		{"succeeded", &OrderActionResponse{OrderID: "1301", ClientOrderID: "client1301", RequestID: "r1"}, &orderOutcome{orderID: "1301", clientOrderID: "client1301"}},
		{"failed", &OrderActionResponse{OrderID: "1301", StatusCode: 51503, StatusMessage: "failed"}, &orderOutcome{orderID: "1301", status: getStatusError(51503, "failed")}},
		{"no status code", &OrderActionResponse{OrderID: "1301", statusCodeMissing: true}, &orderOutcome{orderID: "1301", status: errMissingStatusCode}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, orderActionOutcome(tc.row), "orderActionOutcome should report the row's order and status")
		})
	}
}

func TestSpreadOrderOutcome(t *testing.T) {
	t.Parallel()
	assert.Nil(t, spreadOrderOutcome(nil), "spreadOrderOutcome should report nothing for no row")
	for _, tc := range []struct {
		name string
		row  *SpreadOrderResponse
		exp  *orderOutcome
	}{
		{"succeeded", &SpreadOrderResponse{OrderID: "1302", ClientOrderID: "client1302", RequestID: "r1"}, &orderOutcome{orderID: "1302", clientOrderID: "client1302"}},
		{"failed", &SpreadOrderResponse{OrderID: "1302", StatusCode: 51503, StatusMessage: "failed"}, &orderOutcome{orderID: "1302", status: getStatusError(51503, "failed")}},
		{"no status code", &SpreadOrderResponse{OrderID: "1302", statusCodeMissing: true}, &orderOutcome{orderID: "1302", status: errMissingStatusCode}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, spreadOrderOutcome(tc.row), "spreadOrderOutcome should report the row's order and status")
		})
	}
}

func TestAlgoOrderOutcome(t *testing.T) {
	t.Parallel()
	assert.Nil(t, algoOrderOutcome(nil), "algoOrderOutcome should report nothing for no row")
	for _, tc := range []struct {
		name string
		row  *AlgoOrderActionResponse
		exp  *orderOutcome
	}{
		{"succeeded", &AlgoOrderActionResponse{AlgoID: "806", AlgoClientOrderID: "algo806", ClientOrderID: "client806"}, &orderOutcome{orderID: "806", clientOrderID: "algo806"}},
		{"failed", &AlgoOrderActionResponse{AlgoID: "806", StatusCode: 51000, StatusMessage: "failed"}, &orderOutcome{orderID: "806", status: getStatusError(51000, "failed")}},
		{"no status code", &AlgoOrderActionResponse{AlgoID: "806", statusCodeMissing: true}, &orderOutcome{orderID: "806", status: errMissingStatusCode}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, algoOrderOutcome(tc.row), "algoOrderOutcome should report the row's algo order and status")
		})
	}
}

func TestAmendAlgoOrderOutcome(t *testing.T) {
	t.Parallel()
	assert.Nil(t, amendAlgoOrderOutcome(nil), "amendAlgoOrderOutcome should report nothing for no row")
	for _, tc := range []struct {
		name string
		row  *AmendAlgoOrderResponse
		exp  *orderOutcome
	}{
		{"succeeded", &AmendAlgoOrderResponse{AlgoID: "806", AlgoClientOrderID: "algo806", RequestID: "r1"}, &orderOutcome{orderID: "806", clientOrderID: "algo806"}},
		{"failed", &AmendAlgoOrderResponse{AlgoID: "806", StatusCode: 51000, StatusMessage: "failed"}, &orderOutcome{orderID: "806", status: getStatusError(51000, "failed")}},
		{"no status code", &AmendAlgoOrderResponse{AlgoID: "806", statusCodeMissing: true}, &orderOutcome{orderID: "806", status: errMissingStatusCode}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, amendAlgoOrderOutcome(tc.row), "amendAlgoOrderOutcome should report the row's algo order and status")
		})
	}
}

func TestPlacedOrderID(t *testing.T) {
	t.Parallel()
	const failed = "status code: `51008` status message: \"insufficient balance\""
	for _, tc := range []struct {
		name    string
		row     *orderOutcome
		exp     string
		expErr  error
		expText string
	}{
		{name: "placed", row: &orderOutcome{orderID: "1001", clientOrderID: "clientOne"}, exp: "1001"},
		{name: "no row", expErr: common.ErrInvalidResponse},
		{name: "no order ID", row: &orderOutcome{clientOrderID: "clientOne"}, expErr: common.ErrInvalidResponse},
		{name: "rejected with an order ID", row: &orderOutcome{orderID: "1002", status: getStatusError(51008, "insufficient balance")}, expText: failed},
		{name: "rejected without an order ID", row: &orderOutcome{status: getStatusError(51008, "insufficient balance")}, expText: failed},
		{name: "no status code", row: &orderOutcome{orderID: "1002", status: errMissingStatusCode}, expErr: errMissingStatusCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orderID, err := placedOrderID(tc.row)
			switch {
			case tc.expErr != nil:
				assert.ErrorIs(t, err, tc.expErr, "placedOrderID should return the sentinel for a row that confirms no placement")
			case tc.expText != "":
				assert.EqualError(t, err, tc.expText, "placedOrderID should report the row's status")
			default:
				assert.NoError(t, err, "placedOrderID should accept a placed order's row")
			}
			assert.Equal(t, tc.exp, orderID, "placedOrderID should name only an order the row confirms placed")
		})
	}
}

func TestAmendedOrderID(t *testing.T) {
	t.Parallel()
	const failed = "status code: `51503` status message: \"Order modification failed\""
	for _, tc := range []struct {
		name          string
		row           *orderOutcome
		orderID       string
		clientOrderID string
		exp           string
		expErr        error
		expText       string
	}{
		{name: "by order ID", row: &orderOutcome{orderID: "1301"}, orderID: "1301", exp: "1301"},
		{name: "by both IDs, decided by the order ID", row: &orderOutcome{orderID: "1301", clientOrderID: "other"}, orderID: "1301", clientOrderID: "client1301", exp: "1301"},
		{name: "by client order ID", row: &orderOutcome{orderID: "1301", clientOrderID: "client1301"}, clientOrderID: "client1301", exp: "1301"},
		{name: "no row", orderID: "1301", expErr: common.ErrInvalidResponse},
		{name: "failed", row: &orderOutcome{orderID: "1301", status: getStatusError(51503, "Order modification failed")}, orderID: "1301", expText: failed},
		{name: "no status code", row: &orderOutcome{orderID: "1301", status: errMissingStatusCode}, orderID: "1301", expErr: errMissingStatusCode},
		{name: "by order ID, naming another order", row: &orderOutcome{orderID: "9999"}, orderID: "1301", expErr: errReplyOrderMismatch},
		{name: "by order ID, naming no order", row: &orderOutcome{}, orderID: "1301", expErr: errReplyOrderMismatch},
		{name: "by client order ID, echoing another", row: &orderOutcome{orderID: "9999", clientOrderID: "other"}, clientOrderID: "client1301", expErr: errReplyOrderMismatch},
		{name: "by client order ID, echoing none", row: &orderOutcome{orderID: "1301"}, clientOrderID: "client1301", expErr: errReplyOrderMismatch},
		{name: "by client order ID, without an order ID", row: &orderOutcome{clientOrderID: "client1301"}, clientOrderID: "client1301", expErr: common.ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orderID, err := amendedOrderID(tc.row, tc.orderID, tc.clientOrderID)
			switch {
			case tc.expErr != nil:
				assert.ErrorIs(t, err, tc.expErr, "amendedOrderID should return the sentinel for a row that confirms no amendment")
			case tc.expText != "":
				assert.EqualError(t, err, tc.expText, "amendedOrderID should report the row's status")
			default:
				assert.NoError(t, err, "amendedOrderID should accept the amended order's row")
			}
			assert.Equal(t, tc.exp, orderID, "amendedOrderID should name only the order the row confirms amending")
		})
	}
	t.Run("an order without its ID is no mismatch", func(t *testing.T) {
		t.Parallel()
		_, err := amendedOrderID(&orderOutcome{clientOrderID: "client1301"}, "", "client1301")
		assert.NotErrorIs(t, err, errReplyOrderMismatch, "a row echoing the client order ID should not be reported as another order's")
	})
}

func TestBatchPlacementReplyRows(t *testing.T) {
	t.Parallel()
	rowA := &OrderActionResponse{OrderID: "1001", ClientOrderID: "clientOne"}
	rowB := &OrderActionResponse{OrderID: "2002", ClientOrderID: "clientTwo"}
	rowAAgain := &OrderActionResponse{OrderID: "2002", ClientOrderID: "clientOne"}
	plain := &OrderActionResponse{OrderID: "3003"}
	plainSecond := &OrderActionResponse{OrderID: "4004"}
	other := &OrderActionResponse{OrderID: "5005", ClientOrderID: "clientX"}
	failedA := &OrderActionResponse{ClientOrderID: "clientOne", StatusCode: 51008, StatusMessage: "insufficient balance"}
	failedB := &OrderActionResponse{ClientOrderID: "clientTwo", StatusCode: 51008, StatusMessage: "insufficient balance"}
	failedBNamingA := &OrderActionResponse{OrderID: "1001", ClientOrderID: "clientTwo", StatusCode: 51008, StatusMessage: "insufficient balance"}
	failedANamingB := &OrderActionResponse{OrderID: "2002", ClientOrderID: "clientOne", StatusCode: 51008, StatusMessage: "insufficient balance"}
	rowBThird := &OrderActionResponse{OrderID: "3003", ClientOrderID: "clientTwo"}
	rowCNamingB := &OrderActionResponse{OrderID: "2002", ClientOrderID: "clientThree"}
	otherNamingA := &OrderActionResponse{OrderID: "1001", ClientOrderID: "clientX"}
	for _, tc := range []struct {
		name           string
		clientOrderIDs []string
		rows           []*OrderActionResponse
		exp            []*OrderActionResponse
		expErr         error
		expText        string // what the error says of a row no order takes
	}{
		{name: "rows in order", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{rowA, rowB}, exp: []*OrderActionResponse{rowA, rowB}},
		{name: "rows reversed", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{rowB, rowA}, exp: []*OrderActionResponse{rowA, rowB}},
		{name: "orders without client order IDs", clientOrderIDs: []string{"", ""}, rows: []*OrderActionResponse{plain, plainSecond}, exp: []*OrderActionResponse{nil, nil}, expErr: errReplyOrderMismatch, expText: `order ID "4004"`},
		{name: "a client order ID echoed twice", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{rowA, rowAAgain}, exp: []*OrderActionResponse{nil, nil}, expErr: errReplyOrderMismatch},
		{name: "a client order ID echoed by no row", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{rowA, plain}, exp: []*OrderActionResponse{rowA, nil}, expErr: errReplyOrderMismatch, expText: `order ID "3003"`},
		{name: "a row echoing another client order ID", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{rowA, other}, exp: []*OrderActionResponse{rowA, nil}, expErr: errReplyOrderMismatch, expText: `order ID "5005"`},
		{name: "a null row for an order with a client order ID", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{nil, rowA}, exp: []*OrderActionResponse{rowA, nil}, expErr: errReplyOrderMismatch},
		{name: "a null row for an order without a client order ID", clientOrderIDs: []string{"clientOne", ""}, rows: []*OrderActionResponse{rowA, nil}, exp: []*OrderActionResponse{rowA, nil}, expErr: common.ErrInvalidResponse},
		{name: "a client order ID where an order without one sits", clientOrderIDs: []string{"clientOne", ""}, rows: []*OrderActionResponse{plain, rowA}, exp: []*OrderActionResponse{rowA, nil}, expErr: errReplyOrderMismatch, expText: `order ID "3003"`},
		{name: "an order without a client order ID after its rows", clientOrderIDs: []string{"clientOne", ""}, rows: []*OrderActionResponse{rowA, plain}, exp: []*OrderActionResponse{rowA, nil}, expErr: errReplyOrderMismatch, expText: `order ID "3003"`},
		{name: "fewer rows than orders", clientOrderIDs: []string{"", ""}, rows: []*OrderActionResponse{plain}, expErr: common.ErrInvalidResponse},
		{name: "one order ID for two orders", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{rowB, rowAAgain}, exp: []*OrderActionResponse{nil, nil}, expErr: errDuplicateReplyOrderID, expText: `order ID "2002"`},
		{name: "failures naming no order ID", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{failedB, failedA}, exp: []*OrderActionResponse{failedA, failedB}},
		{name: "a failure naming another order's order ID", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{failedBNamingA, rowA}, exp: []*OrderActionResponse{rowA, failedBNamingA}},
		{
			name: "an order ID the repeated rows of another order name", clientOrderIDs: []string{"clientOne", "clientTwo", "clientThree"}, rows: []*OrderActionResponse{rowAAgain, rowB, rowBThird},
			exp: []*OrderActionResponse{nil, nil, nil}, expErr: errDuplicateReplyOrderID, expText: `order ID "2002"`,
		},
		{
			name: "an order ID a row no order takes names", clientOrderIDs: []string{"clientOne", "clientTwo"}, rows: []*OrderActionResponse{rowA, otherNamingA},
			exp: []*OrderActionResponse{nil, nil}, expErr: errDuplicateReplyOrderID, expText: `order ID "1001"`,
		},
		{
			name: "a failure naming an order ID two other orders name", clientOrderIDs: []string{"clientOne", "clientTwo", "clientThree"}, rows: []*OrderActionResponse{failedANamingB, rowB, rowCNamingB},
			exp: []*OrderActionResponse{failedANamingB, nil, nil}, expErr: errDuplicateReplyOrderID, expText: `order ID "2002"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := make([]PlaceOrderRequest, len(tc.clientOrderIDs))
			for i, clientOrderID := range tc.clientOrderIDs {
				args[i] = PlaceOrderRequest{ClientOrderID: clientOrderID}
			}
			got, err := batchPlacementReplyRows(args, tc.rows)
			assert.ErrorIs(t, err, tc.expErr, "batchPlacementReplyRows should return the expected error")
			if tc.expText != "" {
				assert.ErrorContains(t, err, tc.expText, "batchPlacementReplyRows should name the order a row no order takes placed")
			}
			assert.Equal(t, tc.exp, got, "batchPlacementReplyRows should pair each order with its own row alone")
		})
	}
}

func TestDeriveAmendAlgoOrderRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		expErr error
	}{
		{"trigger without a trigger price", &order.Modify{Type: order.Trigger, OrderID: "806", Price: 101}, limits.ErrPriceBelowMin},
		{"trigger without an order price", &order.Modify{Type: order.Trigger, OrderID: "806", TriggerPrice: 100}, limits.ErrPriceBelowMin},
		{"conditional without a change", &order.Modify{Type: order.ConditionalStop, OrderID: "803"}, errInvalidNewSizeOrPriceInformation},
		{"oco without a change", &order.Modify{Type: order.OCO, OrderID: "803"}, limits.ErrPriceBelowMin},
		{"oco with one leg", &order.Modify{Type: order.OCO, OrderID: "803", Amount: 2, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110}}}, limits.ErrPriceBelowMin},
		{"oco with a stop entry", &order.Modify{Type: order.OCO, OrderID: "803", Amount: 2, RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 100}}}, errAttachedOrderUnsupported},
		{"conditional with a take profit", &order.Modify{Type: order.ConditionalStop, OrderID: "803", Amount: 2, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 110}}}, errAttachedOrderUnsupported},
		{"trigger with a leg lacking its trigger price", &order.Modify{Type: order.Trigger, OrderID: "806", Price: 101, TriggerPrice: 100, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true}}}, limits.ErrPriceBelowMin},
		{"trigger with a stop entry", &order.Modify{Type: order.Trigger, OrderID: "806", Price: 101, TriggerPrice: 100, RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Price: 99}}}, errAttachedOrderUnsupported},
		{"limit", &order.Modify{Type: order.Limit, OrderID: "1301", Amount: 2}, order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := deriveAmendAlgoOrderRequest(tc.modify)
			assert.ErrorIs(t, err, tc.expErr, "deriveAmendAlgoOrderRequest should refuse the amendment")
		})
	}
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		exp    *AmendAlgoOrderRequest
	}{
		{
			name:   "conditional size",
			modify: &order.Modify{Type: order.ConditionalStop, ClientOrderID: "algo803", Amount: 2, TriggerPriceType: order.MarkPrice},
			exp:    &AmendAlgoOrderRequest{AlgoClientOrderID: "algo803", NewSize: 2},
		},
		{
			name:   "conditional prices",
			modify: &order.Modify{Type: order.ConditionalStop, OrderID: "803", Price: 89, TriggerPrice: 90, TriggerPriceType: order.MarkPrice},
			exp:    &AmendAlgoOrderRequest{AlgoID: "803", NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: 89, NewStopLossTriggerPriceType: "mark"},
		},
		{
			name:   "oco size",
			modify: &order.Modify{Type: order.OCO, OrderID: "803", Amount: 2},
			exp:    &AmendAlgoOrderRequest{AlgoID: "803", NewSize: 2},
		},
		{
			name: "oco legs",
			modify: &order.Modify{Type: order.OCO, OrderID: "803", RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110, TriggerPriceType: order.IndexPrice}, StopLoss: order.RiskManagement{LimitPrice: 89},
			}},
			exp: &AmendAlgoOrderRequest{AlgoID: "803", NewTakeProfitTriggerPrice: 110, NewTakeProfitTriggerPriceType: "index", NewStopLossOrderPrice: 89},
		},
		{
			name:   "trigger at market",
			modify: &order.Modify{Type: order.Trigger, OrderID: "806", Price: marketOrderPrice, TriggerPrice: 100},
			exp:    &AmendAlgoOrderRequest{AlgoID: "806", NewTriggerPrice: 100, NewOrderPrice: marketOrderPrice, NewTriggerPriceType: "last"},
		},
		{
			name: "trigger with attached legs",
			modify: &order.Modify{Type: order.Trigger, OrderID: "806", Price: 101, TriggerPrice: 100, RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110}, StopLoss: order.RiskManagement{Price: 90, LimitPrice: 89},
			}},
			exp: &AmendAlgoOrderRequest{AlgoID: "806", NewTriggerPrice: 100, NewOrderPrice: 101, NewTriggerPriceType: "last", AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{
				NewTakeProfitTriggerPrice: 110, NewTakeProfitOrderPrice: marketOrderPrice, NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: 89,
				NewTakeProfitTriggerPriceType: "last", NewStopLossTriggerPriceType: "last",
			}}},
		},
		{
			name: "trigger with a stop loss alone",
			modify: &order.Modify{Type: order.Trigger, OrderID: "806", Price: 101, TriggerPrice: 100, RiskManagementModes: order.RiskManagementModes{
				StopLoss: order.RiskManagement{Price: 90},
			}},
			exp: &AmendAlgoOrderRequest{AlgoID: "806", NewTriggerPrice: 100, NewOrderPrice: 101, NewTriggerPriceType: "last", AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{
				NewStopLossTriggerPrice: 90, NewStopLossOrderPrice: marketOrderPrice, NewStopLossTriggerPriceType: "last",
			}}},
		},
		{
			name: "trigger with a take profit alone",
			modify: &order.Modify{Type: order.Trigger, OrderID: "806", Price: 101, TriggerPrice: 100, RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Price: 110, OrderType: order.Limit, LimitPrice: 109},
			}},
			exp: &AmendAlgoOrderRequest{AlgoID: "806", NewTriggerPrice: 100, NewOrderPrice: 101, NewTriggerPriceType: "last", AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{
				NewTakeProfitTriggerPrice: 110, NewTakeProfitOrderPrice: 109, NewTakeProfitTriggerPriceType: "last",
			}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req, err := deriveAmendAlgoOrderRequest(tc.modify)
			require.NoError(t, err, "deriveAmendAlgoOrderRequest must not error")
			assert.Equal(t, tc.exp, req, "deriveAmendAlgoOrderRequest should amend only what the modification sets")
		})
	}
}

func TestAmendedTriggerPriceType(t *testing.T) {
	t.Parallel()
	assert.Empty(t, amendedTriggerPriceType(0, order.MarkPrice), "a type without a trigger price should not be sent")
	assert.Equal(t, "mark", amendedTriggerPriceType(90, order.MarkPrice), "a type beside a trigger price should be sent")
	assert.Equal(t, "last", amendedTriggerPriceType(90, order.LastPrice), "the last price type beside a trigger price should be sent")
}

func TestCheckSpreadMassCancelFilters(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
		expErr error
	}{
		{"unfiltered", &order.Cancel{AssetType: asset.Spread}, nil},
		{"pair", &order.Cancel{AssetType: asset.Spread, Pair: spreadPair}, nil},
		{"any side and any type", &order.Cancel{AssetType: asset.Spread, Side: order.AnySide, Type: order.AnyType}, nil},
		{"order ID", &order.Cancel{AssetType: asset.Spread, OrderID: "ONE"}, common.ErrFunctionNotSupported},
		{"client order ID", &order.Cancel{AssetType: asset.Spread, ClientOrderID: "mine"}, common.ErrFunctionNotSupported},
		{"side", &order.Cancel{AssetType: asset.Spread, Side: order.Sell}, common.ErrFunctionNotSupported},
		{"order type", &order.Cancel{AssetType: asset.Spread, Type: order.Limit}, common.ErrFunctionNotSupported},
		{"time in force", &order.Cancel{AssetType: asset.Spread, Pair: spreadPair, TimeInForce: order.PostOnly}, common.ErrFunctionNotSupported},
		{"good till cancelled", &order.Cancel{AssetType: asset.Spread, TimeInForce: order.GoodTillCancel}, common.ErrFunctionNotSupported},
		{"margin type", &order.Cancel{AssetType: asset.Spread, Pair: spreadPair, MarginType: margin.Isolated}, common.ErrFunctionNotSupported},
		{"a pair's base alone", &order.Cancel{AssetType: asset.Spread, Pair: currency.Pair{Base: currency.BTC}}, currency.ErrCurrencyCodeEmpty},
		{"a pair's quote alone", &order.Cancel{AssetType: asset.Spread, Pair: currency.Pair{Quote: currency.USDT}}, currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, checkSpreadMassCancelFilters(tc.cancel), tc.expErr, "checkSpreadMassCancelFilters should refuse only a filter the mass cancel cannot apply")
		})
	}
}

func TestCheckWebsocketAccount(t *testing.T) {
	t.Parallel()
	login := accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"}
	ex := newWebsocketTestExchange(t, new(websocketTestServer), true, nil)
	withCredentials := func(edit func(*accounts.Credentials)) func(context.Context) context.Context {
		return func(ctx context.Context) context.Context {
			creds := login
			if edit != nil {
				edit(&creds)
			}
			return accounts.DeployCredentialsToContext(ctx, &creds)
		}
	}
	for _, tc := range []struct {
		name string
		ctx  func(context.Context) context.Context
		err  error
	}{
		{name: "no context credentials"},
		{name: "the login's credentials", ctx: withCredentials(nil)},
		{name: "a sub-account beside the login's key", ctx: withCredentials(func(c *accounts.Credentials) { c.SubAccount = "sub1" })},
		{name: "a one-time password beside the login's key", ctx: withCredentials(func(c *accounts.Credentials) { c.OneTimePassword = "123456" })},
		{name: "a sub-account override", ctx: func(ctx context.Context) context.Context {
			return accounts.DeploySubAccountOverrideToContext(ctx, "sub1")
		}},
		{name: "another key", ctx: withCredentials(func(c *accounts.Credentials) { c.Key = "other-key" }), err: errWebsocketOtherAccount},
		{name: "another secret", ctx: withCredentials(func(c *accounts.Credentials) { c.Secret = "other-secret" }), err: errWebsocketOtherAccount},
		{name: "another passphrase", ctx: withCredentials(func(c *accounts.Credentials) { c.ClientID = "other-passphrase" }), err: errWebsocketOtherAccount},
		{
			name: "unusable context credentials", err: common.ErrTypeAssertFailure,
			ctx: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, accounts.ContextCredentialsFlag, "unusable")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			if tc.ctx != nil {
				ctx = tc.ctx(ctx)
			}
			for _, connection := range []string{privateConnection, businessLoginConnection} {
				assert.ErrorIsf(t, ex.checkWebsocketAccount(ctx, connection), tc.err, "checkWebsocketAccount should return the expected error for the %s connection", connection)
			}
		})
	}

	t.Run("the exchange's credentials changed after the login", func(t *testing.T) {
		t.Parallel()
		rotated := newWebsocketTestExchange(t, new(websocketTestServer), true, nil)
		rotated.SetCredentials(&accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"})
		for _, connection := range []string{privateConnection, businessLoginConnection} {
			assert.ErrorIsf(t, rotated.checkWebsocketAccount(t.Context(), connection), errWebsocketOtherAccount, "checkWebsocketAccount should refuse the %s connection's login, which no longer acts for the exchange's credentials", connection)
		}
	})

	t.Run("a connection which never logged in", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, ex.checkWebsocketAccount(t.Context(), publicConnection), errWebsocketOtherAccount, "checkWebsocketAccount should refuse a connection without a login")
	})

	t.Run("an unavailable connection is left to the operation", func(t *testing.T) {
		t.Parallel()
		unconnected := new(Exchange)
		require.NoError(t, testexch.Setup(unconnected), "Setup must not error")
		assert.NoError(t, unconnected.checkWebsocketAccount(withCredentials(func(c *accounts.Credentials) { c.Key = "other-key" })(t.Context()), privateConnection), "checkWebsocketAccount should leave an unavailable connection for the operation to report")
	})
}

func TestWithAccountCredentials(t *testing.T) {
	t.Parallel()
	own := &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase"}
	other := &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(own)

	t.Run("carries the exchange's credentials through a later change", func(t *testing.T) {
		t.Parallel()
		rotated := new(Exchange)
		require.NoError(t, testexch.Setup(rotated), "Setup must not error")
		rotated.API.AuthenticatedSupport = true
		rotated.SetCredentials(own)
		ctx, err := rotated.withAccountCredentials(t.Context())
		require.NoError(t, err, "withAccountCredentials must not error")
		rotated.SetCredentials(other)
		creds, err := rotated.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		assert.Equal(t, own, creds, "the context should keep acting for the credentials the operation started with")
	})

	t.Run("keeps a sub-account override", func(t *testing.T) {
		t.Parallel()
		ctx, err := ex.withAccountCredentials(accounts.DeploySubAccountOverrideToContext(t.Context(), "sub1"))
		require.NoError(t, err, "withAccountCredentials must not error")
		creds, err := ex.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		assert.Equal(t, &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase", SubAccount: "sub1"}, creds, "the sub-account override should still apply")
	})

	t.Run("carries context credentials through a later change of their store", func(t *testing.T) {
		t.Parallel()
		caller := accounts.DeployCredentialsToContext(t.Context(), other)
		store, ok := caller.Value(accounts.ContextCredentialsFlag).(*accounts.ContextCredentialsStore)
		require.True(t, ok, "the caller's context must carry a credentials store")
		ctx, err := ex.withAccountCredentials(caller)
		require.NoError(t, err, "withAccountCredentials must not error")
		store.Load(own)
		creds, err := ex.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		assert.Equal(t, other, creds, "the context should keep acting for the credentials the operation started with")
	})

	t.Run("leaves disabled authentication to the requests", func(t *testing.T) {
		t.Parallel()
		disabled := new(Exchange)
		require.NoError(t, testexch.Setup(disabled), "Setup must not error")
		disabled.SetCredentials(own)
		disabled.API.AuthenticatedSupport = false
		disabled.API.AuthenticatedWebsocketSupport = false
		_, err := disabled.GetCredentials(t.Context())
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "the exchange's credentials must be refused while authentication is disabled")
		ctx, err := disabled.withAccountCredentials(t.Context())
		require.NoError(t, err, "withAccountCredentials must leave disabled authentication to the requests, which refuse it")
		assert.Nil(t, ctx.Value(accounts.ContextCredentialsFlag), "credentials refused for requests should not be carried, as context credentials bypass the authentication setting")
	})

	t.Run("returns the error of credentials which cannot be used", func(t *testing.T) {
		t.Parallel()
		incomplete := &accounts.Credentials{Key: "own-key", ClientID: "own-passphrase"}
		withoutSecret := new(Exchange)
		require.NoError(t, testexch.Setup(withoutSecret), "Setup must not error")
		withoutSecret.API.AuthenticatedSupport = true
		withoutSecret.SetCredentials(incomplete)
		for _, tc := range []struct {
			name string
			ex   *Exchange
			ctx  func(context.Context) context.Context
		}{
			{name: "the exchange's", ex: withoutSecret, ctx: func(ctx context.Context) context.Context { return ctx }},
			{name: "the context's", ex: ex, ctx: func(ctx context.Context) context.Context { return accounts.DeployCredentialsToContext(ctx, incomplete) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ctx := tc.ctx(t.Context())
				_, unusable := tc.ex.GetCredentials(ctx)
				require.Error(t, unusable, "the credentials must be unusable")
				_, err := tc.ex.withAccountCredentials(ctx)
				assert.EqualError(t, err, unusable.Error(), "withAccountCredentials should return the error of the credentials")
			})
		}
	})

	t.Run("an operation refuses credentials which cannot be used at its start rather than act for those they become", func(t *testing.T) {
		t.Parallel()
		spot := func(ex *Exchange) *order.Submit {
			return &order.Submit{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100}
		}
		cancel := func() *order.Cancel { return &order.Cancel{Pair: mainPair, AssetType: asset.Spot, OrderID: "1001"} }
		modify := func() *order.Modify {
			return &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1001", Amount: 2}
		}
		for _, tc := range []struct {
			name string
			call func(context.Context, *Exchange) error
		}{
			{name: "SubmitOrder", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.SubmitOrder(ctx, spot(ex))
				return err
			}},
			{name: "SubmitOrder of a spread", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.SubmitOrder(ctx, &order.Submit{Exchange: ex.Name, Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100})
				return err
			}},
			{name: "ModifyOrder", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.ModifyOrder(ctx, modify())
				return err
			}},
			{name: "CancelBatchOrders", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.CancelBatchOrders(ctx, []order.Cancel{*cancel()})
				return err
			}},
			{name: "CancelAllOrders", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.CancelAllOrders(ctx, &order.Cancel{Pair: mainPair, AssetType: asset.Spot})
				return err
			}},
			{name: "SetPositionMode", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.SetPositionMode(ctx, positionModeNet)
				return err
			}},
			{name: "SetAccountMode", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.SetAccountMode(ctx, "2")
				return err
			}},
			{name: "GetLeverage's position mode", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.GetLeverage(ctx, asset.PerpetualSwap, perpetualSwapPair, margin.Isolated, order.Long)
				return err
			}},
			{name: "the account level", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.accountLevel(ctx)
				return err
			}},
			{name: "WebsocketSubmitOrder", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.WebsocketSubmitOrder(ctx, spot(ex))
				return err
			}},
			{name: "WebsocketSubmitOrders", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.WebsocketSubmitOrders(ctx, []*order.Submit{spot(ex)})
				return err
			}},
			{name: "WebsocketModifyOrder", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.WebsocketModifyOrder(ctx, modify())
				return err
			}},
			{name: "WebsocketCancelOrder", call: func(ctx context.Context, ex *Exchange) error {
				return ex.WebsocketCancelOrder(ctx, cancel())
			}},
			{name: "WebsocketCancelBatchOrders", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.WebsocketCancelBatchOrders(ctx, []order.Cancel{*cancel()})
				return err
			}},
			{name: "WebsocketCancelAllOrders", call: func(ctx context.Context, ex *Exchange) error {
				_, err := ex.WebsocketCancelAllOrders(ctx, &order.Cancel{Pair: mainPair, AssetType: asset.Spot})
				return err
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				synctest.Test(t, func(t *testing.T) {
					var mu sync.Mutex
					var sent []string
					mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						mu.Lock()
						sent = append(sent, r.URL.Path+" "+r.Header.Get("OK-ACCESS-KEY"))
						mu.Unlock()
						http.Error(w, "unexpected request", http.StatusBadRequest)
					}))
					mock.SkipAuthCheck = false
					mock.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 111}
					ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: primaryAccount().Key, ClientID: primaryAccount().ClientID})
					store, ok := ctx.Value(accounts.ContextCredentialsFlag).(*accounts.ContextCredentialsStore)
					require.True(t, ok, "the context must carry a credentials store")
					_, unusable := mock.GetCredentials(ctx)
					require.Error(t, unusable, "the credentials must be unusable at the start")
					barrier, err := request.NewRateLimitBarrierContexts(ctx, 2)
					require.NoError(t, err, "NewRateLimitBarrierContexts must not error")
					result := make(chan error, 1)
					go func() {
						defer request.AbortRateLimitBarrier(barrier[0])
						result <- tc.call(barrier[0], mock)
					}()
					synctest.Wait()
					store.Load(secondaryAccount())
					assert.ErrorIs(t, request.WaitForRateLimitBarrier(barrier[1]), request.ErrRateLimitBarrierRejected, "the operation should end before any request waits for its rate limit")
					assert.ErrorContains(t, <-result, unusable.Error(), "the operation should return the error of the credentials it started with")
					mu.Lock()
					defer mu.Unlock()
					assert.Empty(t, sent, "the operation should send no request for the credentials they became")
				})
			})
		}
	})
}

func TestRequireWebsocketInstrumentIDCode(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345, "ETH-USDT": 0}
	for _, tc := range []struct {
		name   string
		instID string
		exp    uint64
		err    error
	}{
		{name: "cached code", instID: mainPair.String(), exp: 12345},
		{name: "uncached instrument", instID: "SOL-USDT", err: errMissingInstrumentIDCode},
		{name: "cached null code", instID: "ETH-USDT", err: errMissingInstrumentIDCode},
		{name: "empty instrument ID", err: errMissingInstrumentIDCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, err := ex.requireWebsocketInstrumentIDCode(tc.instID)
			require.ErrorIs(t, err, tc.err, "requireWebsocketInstrumentIDCode must return the expected error")
			if tc.err != nil {
				assert.EqualError(t, err, errMissingInstrumentIDCode.Error()+": "+tc.instID, "the error should name the instrument")
			}
			assert.Equal(t, tc.exp, code, "requireWebsocketInstrumentIDCode should return the cached code")
		})
	}
}

func TestPrepareWebsocketPlaceOrder(t *testing.T) {
	t.Parallel()
	ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/config" {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
	}))
	ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345, perpetualSwapPair.String(): 67890}
	for _, tc := range []struct {
		name string
		s    *order.Submit
		exp  *PlaceOrderRequest
		err  error
	}{
		{
			name: "spot limit", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ClientOrderID: "client1"},
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "BUY", OrderType: orderLimit, Size: 1, Price: 100, ClientOrderID: "client1"},
		},
		{
			name: "spot market", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Market, Amount: 1},
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "BUY", OrderType: orderMarket, Size: 1, TargetCurrency: targetCurrencyBase},
		},
		{
			name: "post only", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, Type: order.Limit, TimeInForce: order.PostOnly, Amount: 1, Price: 100},
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", InstrumentIDCode: 12345, TradeMode: TradeModeCash, Side: "SELL", OrderType: orderPostOnly, Size: 1, Price: 100},
		},
		{
			name: "perpetual swap close in net mode", s: &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 95000, ReduceOnly: true},
			exp: &PlaceOrderRequest{AssetType: asset.PerpetualSwap, InstrumentID: "BTC-USDT-SWAP", InstrumentIDCode: 67890, TradeMode: TradeModeCross, Side: "sell", PositionSide: positionSideNet, OrderType: orderLimit, Size: 1, Price: 95000, ReduceOnly: true},
		},
		{
			name: "optimal limit", s: &order.Submit{AssetType: asset.PerpetualSwap, Pair: perpetualSwapPair, Side: order.Buy, Type: order.OptimalLimit, Amount: 1},
			exp: &PlaceOrderRequest{AssetType: asset.PerpetualSwap, InstrumentID: "BTC-USDT-SWAP", InstrumentIDCode: 67890, TradeMode: TradeModeCross, Side: "buy", PositionSide: positionSideNet, OrderType: orderOptimalLimitIOC, Size: 1},
		},
		{name: "invalid submit", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1}, err: limits.ErrPriceBelowMin},
		{name: "spread order", s: &order.Submit{AssetType: asset.Spread, Pair: spreadPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}, err: errSpreadWebsocketBatchUnsupported},
		{name: "order type OKX does not place", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Stop, Amount: 1, Price: 100}, err: order.ErrUnsupportedOrderType},
		{name: "algo order", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Sell, Type: order.Trigger, Amount: 1}, err: common.ErrFunctionNotSupported},
		{
			name: "spot cross margin sell sends reduce only", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, MarginType: margin.Multi, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 100, ReduceOnly: true},
			exp: &PlaceOrderRequest{AssetType: asset.Spot, InstrumentID: "BTC-USDT", InstrumentIDCode: 12345, TradeMode: TradeModeCross, Currency: currency.BTC, Side: "SELL", OrderType: orderLimit, Size: 1, Price: 100, ReduceOnly: true},
		},
		{
			name: "take profit and stop loss", err: errAttachedOrderUnsupported,
			s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Price: 90}}},
		},
		{name: "post only market", s: &order.Submit{AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Market, TimeInForce: order.PostOnly, Amount: 1}, err: order.ErrUnsupportedTimeInForce},
		{
			name: "uncached instrument code",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter), Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
			err:  errMissingInstrumentIDCode,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req, err := ex.prepareWebsocketPlaceOrder(t.Context(), tc.s)
			require.ErrorIs(t, err, tc.err, "prepareWebsocketPlaceOrder must return the expected error")
			assert.Equal(t, tc.exp, req, "prepareWebsocketPlaceOrder should derive the order with its instrument ID code")
		})
	}
}

func TestContractPositionMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		key        string
		cached     map[string]string
		posMode    string
		reply      string
		switches   bool
		exp        string
		err        error
		expCached  map[string]string
		expFetches int32
	}{
		{name: "cached mode", cached: map[string]string{"": positionModeLongShort}, exp: positionModeLongShort, expCached: map[string]string{"": positionModeLongShort}},
		{name: "fetches and caches the mode", cached: map[string]string{}, posMode: positionModeNet, exp: positionModeNet, expCached: map[string]string{"": positionModeNet}, expFetches: 1},
		{
			name: "another account's cached mode is not used", key: "KEY-A", cached: map[string]string{"KEY-B": positionModeLongShort}, posMode: positionModeNet, exp: positionModeNet,
			expCached: map[string]string{"KEY-A": positionModeNet, "KEY-B": positionModeLongShort}, expFetches: 1,
		},
		{name: "fetch error", cached: map[string]string{}, reply: `{"code":"50011","msg":"","data":[]}`, err: ErrorCodes["50011"], expCached: map[string]string{}, expFetches: 1},
		{name: "no account configuration", cached: map[string]string{}, reply: `{"code":"0","msg":"","data":null}`, err: common.ErrNoResponse, expCached: map[string]string{}, expFetches: 1},
		{name: "unknown mode", cached: map[string]string{}, posMode: "both_mode", err: errInvalidPositionMode, expCached: map[string]string{}, expFetches: 1},
		{name: "a switch lands during the fetch", cached: map[string]string{}, posMode: positionModeNet, switches: true, err: errPositionModeSwitchedDuringFetch, expCached: map[string]string{}, expFetches: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var fetches atomic.Int32
			var fetching atomic.Pointer[Exchange]
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/account/config" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				fetches.Add(1)
				if tc.switches {
					ex := fetching.Load()
					ex.accountPositionModeMu.Lock()
					ex.accountPositionModeSwitches++
					ex.accountPositionModeMu.Unlock()
				}
				if tc.reply != "" {
					_, err := w.Write([]byte(tc.reply))
					assert.NoError(t, err, "writing the reply should not error")
					return
				}
				writeOKXData(t, w, []map[string]string{{"posMode": tc.posMode}})
			}))
			fetching.Store(ex)
			ex.accountPositionModes = tc.cached
			ctx := t.Context()
			if tc.key != "" {
				ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: tc.key, Secret: "secret", ClientID: "passphrase"})
			}
			mode, err := ex.contractPositionMode(ctx)
			require.ErrorIs(t, err, tc.err, "contractPositionMode must return the expected error")
			assert.Equal(t, tc.exp, mode, "contractPositionMode should return the account's position mode")
			assert.Equal(t, tc.expCached, ex.accountPositionModes, "contractPositionMode should cache a settled mode for the request's API key only")
			assert.Equal(t, tc.expFetches, fetches.Load(), "contractPositionMode should fetch the account configuration when no mode is cached")
		})
	}

	t.Run("a confirmed switch outlives an older in-flight fetch", func(t *testing.T) {
		t.Parallel()
		getStarted := make(chan struct{})
		startOnce := sync.OnceFunc(func() { close(getStarted) })
		release := make(chan struct{})
		releaseOnce := sync.OnceFunc(func() { close(release) })
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/account/config":
				startOnce()
				<-release
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
			case "/account/set-position-mode":
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		// Registered after the mock server's cleanup, so it runs first and frees
		// a handler still waiting on release.
		t.Cleanup(releaseOnce)
		type result struct {
			mode string
			err  error
		}
		fetched := make(chan result, 1)
		go func() {
			mode, err := e.contractPositionMode(t.Context())
			fetched <- result{mode, err}
		}()
		select {
		case <-getStarted:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the account configuration fetch must start")
		}
		_, err := e.SetPositionMode(t.Context(), positionModeLongShort)
		require.NoError(t, err, "SetPositionMode must not error")
		releaseOnce()
		select {
		case got := <-fetched:
			require.NoError(t, got.err, "the in-flight fetch must not error")
			assert.Equal(t, positionModeLongShort, got.mode, "the in-flight lookup should return the confirmed mode rather than its older fetch")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the in-flight fetch must return")
		}
		mode, err := e.contractPositionMode(t.Context())
		require.NoError(t, err, "contractPositionMode must not error")
		assert.Equal(t, positionModeLongShort, mode, "a confirmed switch should outlive an older fetch")
	})

	t.Run("each account follows its own mode", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		modes := map[string]string{"KEY-A": positionModeNet, "KEY-B": positionModeLongShort}
		var sent []string
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("OK-ACCESS-KEY")
			switch r.URL.Path {
			case "/account/config":
				mu.Lock()
				mode := modes[key]
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"posMode": mode}})
			case "/account/set-position-mode":
				var req SetPositionModeRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("decoding the position mode body should not error: %v", err)
					return
				}
				mu.Lock()
				modes[key] = req.PositionMode
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"posMode": req.PositionMode}})
			case "/trade/order":
				var req PlaceOrderRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("decoding the order body should not error: %v", err)
					return
				}
				mu.Lock()
				sent = append(sent, key+" "+req.PositionSide)
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		ctxFor := func(key string) context.Context {
			return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
		}
		submit := func(key string) {
			t.Helper()
			_, err := e.SubmitOrder(ctxFor(key), &order.Submit{Exchange: e.Name, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
			require.NoErrorf(t, err, "SubmitOrder must not error for %s", key)
		}
		submit("KEY-A")
		submit("KEY-B")
		_, err := e.SetPositionMode(ctxFor("KEY-B"), positionModeNet)
		require.NoError(t, err, "SetPositionMode must not error")
		submit("KEY-A")
		submit("KEY-B")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"KEY-A net", "KEY-B long", "KEY-A net", "KEY-B net"}, sent, "each account's orders should follow its own position mode")
	})

	t.Run("a switch through another key refetches the mode", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		accountMode := positionModeNet
		var sent []string
		var fetches int
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("OK-ACCESS-KEY")
			switch r.URL.Path {
			case "/account/config":
				mu.Lock()
				fetches++
				mode := accountMode
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"posMode": mode}})
			case "/account/set-position-mode":
				var req SetPositionModeRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("decoding the position mode body should not error: %v", err)
					return
				}
				mu.Lock()
				accountMode = req.PositionMode
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"posMode": req.PositionMode}})
			case "/trade/order":
				var req PlaceOrderRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("decoding the order body should not error: %v", err)
					return
				}
				mu.Lock()
				sent = append(sent, key+" "+req.PositionSide)
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		ctxFor := func(key string) context.Context {
			return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
		}
		submit := func(key string) {
			t.Helper()
			_, err := e.SubmitOrder(ctxFor(key), &order.Submit{Exchange: e.Name, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
			require.NoErrorf(t, err, "SubmitOrder must not error for %s", key)
		}
		submit("KEY-1")
		_, err := e.SetPositionMode(ctxFor("KEY-2"), positionModeLongShort)
		require.NoError(t, err, "SetPositionMode must not error")
		submit("KEY-1")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"KEY-1 net", "KEY-1 long"}, sent, "a key should follow a switch its account confirmed through another key")
		assert.Equal(t, 2, fetches, "the second order should fetch its own key's mode again rather than keep the stale one")
	})

	t.Run("a fetch overlapping another key's switch errors", func(t *testing.T) {
		t.Parallel()
		getStarted := make(chan struct{})
		startOnce := sync.OnceFunc(func() { close(getStarted) })
		release := make(chan struct{})
		releaseOnce := sync.OnceFunc(func() { close(release) })
		var mu sync.Mutex
		accountMode := positionModeNet
		var fetches int
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/account/config":
				mu.Lock()
				fetches++
				first := fetches == 1
				mode := accountMode
				mu.Unlock()
				if first {
					// The first fetch reads the mode before the switch and answers after it.
					startOnce()
					<-release
				}
				writeOKXData(t, w, []map[string]string{{"posMode": mode}})
			case "/account/set-position-mode":
				mu.Lock()
				accountMode = positionModeLongShort
				mu.Unlock()
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		// Registered after the mock server's cleanup, so it runs first and frees
		// a handler still waiting on release.
		t.Cleanup(releaseOnce)
		ctxFor := func(key string) context.Context {
			return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
		}
		type result struct {
			mode string
			err  error
		}
		fetched := make(chan result, 1)
		go func() {
			mode, err := e.contractPositionMode(ctxFor("KEY-1"))
			fetched <- result{mode, err}
		}()
		select {
		case <-getStarted:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the account configuration fetch must start")
		}
		_, err := e.SetPositionMode(ctxFor("KEY-2"), positionModeLongShort)
		require.NoError(t, err, "SetPositionMode must not error")
		releaseOnce()
		select {
		case got := <-fetched:
			require.ErrorIs(t, got.err, errPositionModeSwitchedDuringFetch, "the overlapped lookup must error rather than report a mode that predates the switch")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the in-flight lookup must return")
		}
		mode, err := e.contractPositionMode(ctxFor("KEY-1"))
		require.NoError(t, err, "contractPositionMode must not error once the switch settled")
		assert.Equal(t, positionModeLongShort, mode, "the next lookup should fetch the switched mode")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 2, fetches, "the overlapped fetch should be neither cached nor reused, so the next lookup fetches again")
	})

	t.Run("caches the mode of the account it fetched", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var keys []string
		mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/config" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			key := r.Header.Get("OK-ACCESS-KEY")
			mu.Lock()
			keys = append(keys, key)
			mu.Unlock()
			mode, level := positionModeLongShort, "2"
			if key == secondaryAccount().Key {
				mode, level = positionModeNet, "3"
			}
			writeOKXData(t, w, []map[string]string{{"posMode": mode, "acctLv": level}})
		}))
		mock.SetCredentials(primaryAccount())
		mode, err := mock.contractPositionMode(&credentialSwitchContext{Context: t.Context(), ex: mock, next: secondaryAccount()})
		require.NoError(t, err, "contractPositionMode must not error")
		assert.Equal(t, positionModeLongShort, mode, "the mode should be the account's the lookup started with")
		assert.Equal(t, map[string]string{"primary-key": positionModeLongShort}, mock.accountPositionModes, "the mode should be cached for the account it was fetched for")
		assert.Equal(t, map[string]uint64{"primary-key": 2}, mock.accountLevels, "the level should be cached for the account it was fetched for")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"primary-key"}, keys, "the account configuration should be fetched for the account the lookup started with")
	})
}

func TestCachePositionMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		cached             map[string]string
		switches, observed uint64
		exp                string
		settled            bool
		expCached          map[string]string
	}{
		{name: "caches a fetch no switch overlapped", cached: map[string]string{}, switches: 3, observed: 3, exp: positionModeNet, settled: true, expCached: map[string]string{"KEY-1": positionModeNet}},
		{
			name: "keeps other keys' modes", cached: map[string]string{"KEY-2": positionModeLongShort}, switches: 3, observed: 3, exp: positionModeNet, settled: true,
			expCached: map[string]string{"KEY-1": positionModeNet, "KEY-2": positionModeLongShort},
		},
		{name: "keeps the key's confirmed mode", cached: map[string]string{"KEY-1": positionModeLongShort}, switches: 3, observed: 3, exp: positionModeLongShort, settled: true, expCached: map[string]string{"KEY-1": positionModeLongShort}},
		{name: "a confirmed mode outlives an overlapped fetch", cached: map[string]string{"KEY-1": positionModeLongShort}, switches: 4, observed: 3, exp: positionModeLongShort, settled: true, expCached: map[string]string{"KEY-1": positionModeLongShort}},
		{name: "refuses a fetch a switch overlapped", cached: map[string]string{"KEY-2": positionModeLongShort}, switches: 4, observed: 3, expCached: map[string]string{"KEY-2": positionModeLongShort}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := &Exchange{accountPositionModes: tc.cached, accountPositionModeSwitches: tc.switches}
			mode, settled := ex.cachePositionMode("KEY-1", tc.observed, positionModeNet)
			assert.Equal(t, tc.exp, mode, "cachePositionMode should return the mode to place with")
			assert.Equal(t, tc.settled, settled, "cachePositionMode should report whether the mode is settled")
			assert.Equal(t, tc.expCached, ex.accountPositionModes, "cachePositionMode should cache only a settled fetch")
		})
	}

	t.Run("waits for the position mode lock", func(t *testing.T) {
		t.Parallel()
		ex := &Exchange{accountPositionModes: map[string]string{}}
		requireWaitsForLock(t, &ex.accountPositionModeMu, func() { ex.cachePositionMode("KEY-1", 0, positionModeNet) })
		assert.Equal(t, map[string]string{"KEY-1": positionModeNet}, ex.accountPositionModes, "cachePositionMode should cache the mode once it holds the lock")
	})
}

func TestAccountLevel(t *testing.T) {
	t.Parallel()
	ctxFor := func(key string) context.Context {
		return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: key, Secret: "secret", ClientID: "passphrase"})
	}
	t.Run("one fetch serves the level and the position mode", func(t *testing.T) {
		t.Parallel()
		ex, requests := newRecordingExchange(t, func(string) any {
			return []map[string]string{{"acctLv": "2", "posMode": positionModeLongShort}}
		})
		for range 2 {
			level, err := ex.accountLevel(t.Context())
			require.NoError(t, err, "accountLevel must not error")
			assert.Equal(t, uint64(accountLevelFutures), level, "accountLevel should report the account's level")
		}
		mode, err := ex.contractPositionMode(t.Context())
		require.NoError(t, err, "contractPositionMode must not error")
		assert.Equal(t, positionModeLongShort, mode, "contractPositionMode should report the mode the level's fetch cached")
		assert.Len(t, requests(), 1, "the account configuration should be fetched once")
	})

	t.Run("the position mode's fetch caches the level", func(t *testing.T) {
		t.Parallel()
		ex, requests := newRecordingExchange(t, func(string) any {
			return []map[string]string{{"acctLv": "4", "posMode": positionModeNet}}
		})
		_, err := ex.contractPositionMode(t.Context())
		require.NoError(t, err, "contractPositionMode must not error")
		level, err := ex.accountLevel(t.Context())
		require.NoError(t, err, "accountLevel must not error")
		assert.Equal(t, uint64(4), level, "accountLevel should report the level the position mode's fetch cached")
		assert.Len(t, requests(), 1, "the account configuration should be fetched once")
	})

	t.Run("each account has its own level", func(t *testing.T) {
		t.Parallel()
		levels := map[string]string{"KEY-A": "2", "KEY-B": "3"}
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/account/config", r.URL.Path, "the level should come from the account configuration")
			writeOKXData(t, w, []map[string]string{{"acctLv": levels[r.Header.Get("OK-ACCESS-KEY")], "posMode": positionModeNet}})
		}))
		for _, key := range []string{"KEY-A", "KEY-B", "KEY-A"} {
			level, err := ex.accountLevel(ctxFor(key))
			require.NoErrorf(t, err, "accountLevel must not error for %s", key)
			assert.Equalf(t, levels[key], strconv.FormatUint(level, 10), "accountLevel should report %s's own level", key)
		}
	})

	t.Run("an unusable reply is not cached", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, reply string
			expErr      error
		}{
			{"no level", `[{"acctLv":"","posMode":"net_mode"}]`, errInvalidAccountLevel},
			{"zero level", `[{"acctLv":"0","posMode":"net_mode"}]`, errInvalidAccountLevel},
			{"null data", `null`, common.ErrNoResponse},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var fetches atomic.Int32
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					fetches.Add(1)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"code":"0","msg":"","data":` + tc.reply + `}`))
				}))
				for range 2 {
					_, err := ex.accountLevel(t.Context())
					require.ErrorIs(t, err, tc.expErr, "accountLevel must refuse the reply")
				}
				assert.Equal(t, int32(2), fetches.Load(), "an unusable reply should not be cached")
				assert.Empty(t, ex.accountLevels, "no level should be cached")
			})
		}
	})

	// A switch confirmed through another key while the level is fetched may postdate the fetched level, which is
	// neither used nor cached, so the next lookup fetches again
	t.Run("a fetch overlapping a switch errors", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var fetches int
		var ex *Exchange
		ex = newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/account/config":
				mu.Lock()
				fetches++
				first := fetches == 1
				mu.Unlock()
				level := "2"
				if first {
					level = "3"
					_, err := ex.SetAccountMode(ctxFor("KEY-2"), "2")
					assert.NoError(t, err, "the overlapping switch should not error")
				}
				writeOKXData(t, w, []map[string]string{{"acctLv": level, "posMode": positionModeNet}})
			case "/account/set-account-level":
				writeOKXData(t, w, []map[string]string{{"acctLv": "2"}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		_, err := ex.accountLevel(ctxFor("KEY-1"))
		require.ErrorIs(t, err, errAccountConfigurationSwitchedDuringFetch, "the overlapped lookup must error rather than report a level that predates the switch")
		level, err := ex.accountLevel(ctxFor("KEY-1"))
		require.NoError(t, err, "accountLevel must not error once the switch settled")
		assert.Equal(t, uint64(accountLevelFutures), level, "the next lookup should fetch the switched level")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, 2, fetches, "the overlapped fetch should be neither cached nor reused")
	})

	t.Run("caches the level of the account it fetched", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var keys []string
		mock := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/account/config" {
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			key := r.Header.Get("OK-ACCESS-KEY")
			mu.Lock()
			keys = append(keys, key)
			mu.Unlock()
			mode, level := positionModeLongShort, "2"
			if key == secondaryAccount().Key {
				mode, level = positionModeNet, "3"
			}
			writeOKXData(t, w, []map[string]string{{"posMode": mode, "acctLv": level}})
		}))
		mock.SetCredentials(primaryAccount())
		level, err := mock.accountLevel(&credentialSwitchContext{Context: t.Context(), ex: mock, next: secondaryAccount()})
		require.NoError(t, err, "accountLevel must not error")
		assert.Equal(t, uint64(2), level, "the level should be the account's the lookup started with")
		assert.Equal(t, map[string]uint64{"primary-key": 2}, mock.accountLevels, "the level should be cached for the account it was fetched for")
		assert.Equal(t, map[string]string{"primary-key": positionModeLongShort}, mock.accountPositionModes, "the mode should be cached for the account it was fetched for")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"primary-key"}, keys, "the account configuration should be fetched for the account the lookup started with")
	})
}

func TestCacheAccountLevel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		cached     map[string]uint64
		switches   uint64 // the switches counted when the fetch began; the cache has counted one
		expLevel   uint64
		expSettled bool
		expCached  map[string]uint64
	}{
		{"caches the fetched level", map[string]uint64{}, 1, 2, true, map[string]uint64{"KEY-1": 2}},
		{"another key's level does not answer", map[string]uint64{"KEY-2": 4}, 1, 2, true, map[string]uint64{"KEY-1": 2, "KEY-2": 4}},
		{"keeps the level confirmed for the key", map[string]uint64{"KEY-1": 4}, 0, 4, true, map[string]uint64{"KEY-1": 4}},
		{"refuses a fetch overlapping a switch", map[string]uint64{}, 0, 0, false, map[string]uint64{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := &Exchange{accountLevels: tc.cached, accountPositionModeSwitches: 1}
			level, settled := ex.cacheAccountLevel("KEY-1", tc.switches, 2)
			assert.Equal(t, tc.expLevel, level, "cacheAccountLevel should report the level to use")
			assert.Equal(t, tc.expSettled, settled, "cacheAccountLevel should report whether the level is settled")
			assert.Equal(t, tc.expCached, ex.accountLevels, "cacheAccountLevel should cache only a settled fetched level")
		})
	}
}

func TestPositionModeKey(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetCredentials(&accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase"})
	for _, tc := range []struct {
		name string
		ex   *Exchange
		ctx  func(context.Context) context.Context
		exp  string
	}{
		{name: "the exchange's own credentials", ex: ex, exp: "own-key"},
		{
			name: "context credentials", ex: ex, exp: "other-key",
			ctx: func(ctx context.Context) context.Context {
				return accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"})
			},
		},
		{
			name: "a sub-account override keeps the key", ex: ex, exp: "own-key",
			ctx: func(ctx context.Context) context.Context {
				return accounts.DeploySubAccountOverrideToContext(ctx, "sub1")
			},
		},
		{name: "no credentials", ex: new(Exchange)},
		{
			name: "unusable context credentials", ex: ex,
			ctx: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, accounts.ContextCredentialsFlag, "unusable")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			if tc.ctx != nil {
				ctx = tc.ctx(ctx)
			}
			assert.Equal(t, tc.exp, tc.ex.positionModeKey(ctx), "positionModeKey should return the API key the request acts for")
		})
	}
}

func TestChaseTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		mode order.TrackingMode
		exp  string
	}{
		{name: "distance", mode: order.Distance, exp: "distance"},
		{name: "percentage", mode: order.Percentage, exp: "ratio"},
		{name: "unknown", mode: order.UnknownTrackingMode},
		{name: "undefined", mode: order.TrackingMode(99)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, chaseTypeString(tc.mode), "chaseTypeString should return OKX's chaseType for the tracking mode")
		})
	}
}

func TestPriceTypeString(t *testing.T) {
	t.Parallel()
	priceTypeToStringMap := map[order.PriceType]string{
		order.LastPrice:        "last",
		order.IndexPrice:       "index",
		order.MarkPrice:        "mark",
		order.UnknownPriceType: "",
	}
	var priceTString string
	for x := range priceTypeToStringMap {
		priceTString = priceTypeString(x)
		assert.Equal(t, priceTString, priceTypeToStringMap[x])
	}
}

func TestMarginTypeToString(t *testing.T) {
	t.Parallel()
	marginTypeToStringMap := map[margin.Type]string{
		margin.Isolated:     "isolated",
		margin.Multi:        "cross",
		margin.NoMargin:     "cash",
		margin.SpotIsolated: "spot_isolated",
		// Unset must map to an empty trade mode so the submit path can tell
		// the zero margin type, which takes a per-asset default, apart from
		// the types it refuses.
		margin.Unset: "",
		// An unsupported type falls through to an empty trade mode, which
		// the leverage wrappers reject with margin.ErrMarginTypeUnsupported.
		margin.Type(99): "",
	}
	var marginTypeString string
	for m := range marginTypeToStringMap {
		marginTypeString = e.marginTypeToString(m)
		assert.Equal(t, marginTypeString, marginTypeToStringMap[m])
	}
}

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := new(Exchange).MessageID()
	require.Len(t, id, 32, "Must return the correct length of message id")
	u, err := uuid.Parse(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	require.Equal(t, byte(7), u[6]>>4, "MessageID must return a V7 uuid") // RFC 9562 version nibble
	require.Len(t, u.String(), 36, "UUID v7 string representation must be 36 characters long")
}

func TestGetInstrumentIDCode(t *testing.T) {
	t.Parallel()

	// A detached instance keeps the seeded cache isolated from parallel tests
	// sharing the package-level exchange.
	fresh := new(Exchange)
	fresh.instrumentsInfoMap = make(map[string][]InstrumentResponse)
	fresh.instrumentIDCodeMap = map[string]uint64{
		"BTC-USDT":      12345,
		"BTC-USDT-SWAP": 67890,
	}

	testCases := []struct {
		instrumentID string
		expectedCode uint64
	}{
		{instrumentID: "BTC-USDT", expectedCode: 12345},
		{instrumentID: "BTC-USDT-SWAP", expectedCode: 67890},
		{instrumentID: "", expectedCode: 0},
		{instrumentID: "NOT-CACHED", expectedCode: 0},
	}
	for _, tc := range testCases {
		got := fresh.getInstrumentIDCode(tc.instrumentID)
		assert.Equalf(t, tc.expectedCode, got, "instrument ID code for %q should match the cached value", tc.instrumentID)
	}
}

func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

// instrumentFetchRecorder answers the underlying and instrument listings with the rows of each instrument type and
// underlying, failing the request given as failing, and records every request it receives
type instrumentFetchRecorder struct {
	mu          sync.Mutex
	requests    []string
	underlyings []string
	rows        map[string][]map[string]any
	failing     string
}

func (f *instrumentFetchRecorder) serve(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		requested := r.URL.Path + "?" + r.URL.Query().Encode()
		f.mu.Lock()
		f.requests = append(f.requests, requested)
		f.mu.Unlock()
		if requested == f.failing {
			_, err := w.Write([]byte(`{"code":"50011","msg":"","data":[]}`))
			assert.NoError(t, err, "writing the error reply should not error")
			return
		}
		switch r.URL.Path {
		case "/public/underlying":
			writeOKXData(t, w, [][]string{f.underlyings})
		case "/public/instruments":
			writeOKXData(t, w, f.rows[r.URL.Query().Get("instType")+r.URL.Query().Get("uly")])
		default:
			t.Errorf("unexpected request %s", requested)
			http.NotFound(w, r)
		}
	}
}

func (f *instrumentFetchRecorder) received() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

// instrumentRow returns an instrument listing row and the instrument it decodes to
func instrumentRow(t *testing.T, instType, instID, underlying string, code uint64) (map[string]any, InstrumentResponse) {
	t.Helper()
	pair, err := currency.NewPairDelimiter(instID, currency.DashDelimiter)
	require.NoError(t, err, "the instrument ID must parse")
	return map[string]any{"instType": instType, "instId": instID, "uly": underlying, "instIdCode": code},
		InstrumentResponse{InstrumentType: instType, InstrumentID: pair, Underlying: underlying, InstrumentIDCode: code}
}

func TestGetInstrumentsForOptions(t *testing.T) {
	t.Parallel()
	btcCallRow, btcCall := instrumentRow(t, instTypeOption, "BTC-USD-261225-100000-C", "BTC-USD", 101)
	btcPutRow, btcPut := instrumentRow(t, instTypeOption, "BTC-USD-261225-100000-P", "BTC-USD", 102)
	ethCallRow, ethCall := instrumentRow(t, instTypeOption, "ETH-USD-261225-4000-C", "ETH-USD", 201)
	rows := map[string][]map[string]any{instTypeOption + "BTC-USD": {btcCallRow, btcPutRow}, instTypeOption + "ETH-USD": {ethCallRow}}
	underlyingRequest := "/public/underlying?instType=OPTION"
	btcRequest, ethRequest := "/public/instruments?instType=OPTION&uly=BTC-USD", "/public/instruments?instType=OPTION&uly=ETH-USD"
	for _, tc := range []struct {
		name        string
		underlyings []string
		failing     string
		exp         []InstrumentResponse
		requests    []string
		err         error
	}{
		{name: "every underlying's instruments in order", underlyings: []string{"BTC-USD", "ETH-USD"}, exp: []InstrumentResponse{btcCall, btcPut, ethCall}, requests: []string{underlyingRequest, btcRequest, ethRequest}},
		{name: "no underlyings", underlyings: []string{}, requests: []string{underlyingRequest}},
		{name: "underlying listing error", underlyings: []string{"BTC-USD"}, failing: underlyingRequest, requests: []string{underlyingRequest}, err: ErrorCodes["50011"]},
		{name: "instrument listing error", underlyings: []string{"BTC-USD", "ETH-USD"}, failing: ethRequest, requests: []string{underlyingRequest, btcRequest, ethRequest}, err: ErrorCodes["50011"]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &instrumentFetchRecorder{underlyings: tc.underlyings, rows: rows, failing: tc.failing}
			ex := newMockExchange(t, f.serve(t))
			result, err := ex.getInstrumentsForOptions(t.Context())
			require.ErrorIs(t, err, tc.err, "getInstrumentsForOptions must return the expected error")
			assert.Equal(t, tc.exp, result, "getInstrumentsForOptions should return every underlying's instruments")
			assert.Equal(t, tc.requests, f.received(), "getInstrumentsForOptions should list each underlying's instruments in turn")
		})
	}

	t.Run("listed underlyings", func(t *testing.T) {
		t.Parallel()
		result, err := e.getInstrumentsForOptions(contextGenerate())
		require.NoError(t, err, "getInstrumentsForOptions must not error")
		require.NotEmpty(t, result, "getInstrumentsForOptions must return option instruments")
		var underlyings, instTypes []string
		for i := range result {
			if !slices.Contains(underlyings, result[i].Underlying) {
				underlyings = append(underlyings, result[i].Underlying)
			}
			if !slices.Contains(instTypes, result[i].InstrumentType) {
				instTypes = append(instTypes, result[i].InstrumentType)
			}
		}
		assert.Equal(t, []string{instTypeOption}, instTypes, "getInstrumentsForOptions should return option instruments only")
		if mockTests {
			assert.Equal(t, []string{"SOL-USD", "BTC-USD", "ETH-USD", "XAU-USD", "HYPE-USD"}, underlyings, "getInstrumentsForOptions should return the instruments of each listed underlying in turn")
		}
	})
}

func TestGetInstrumentsForAsset(t *testing.T) {
	t.Parallel()
	spotRow, spotInstrument := instrumentRow(t, instTypeSpot, "BTC-USDT", "", 1)
	marginRow, marginInstrument := instrumentRow(t, instTypeMargin, "BTC-USDT", "", 2)
	futuresRow, futuresInstrument := instrumentRow(t, instTypeFutures, "BTC-USD-261225", "BTC-USD", 3)
	swapRow, swapInstrument := instrumentRow(t, instTypeSwap, "BTC-USDT-SWAP", "BTC-USDT", 4)
	optionRow, optionInstrument := instrumentRow(t, instTypeOption, "BTC-USD-261225-100000-C", "BTC-USD", 5)
	rows := map[string][]map[string]any{
		instTypeSpot: {spotRow}, instTypeMargin: {marginRow}, instTypeFutures: {futuresRow}, instTypeSwap: {swapRow}, instTypeOption + "BTC-USD": {optionRow},
	}
	for _, tc := range []struct {
		name     string
		a        asset.Item
		failing  string
		exp      []InstrumentResponse
		cached   map[string][]InstrumentResponse
		codes    map[string]uint64
		requests []string
		err      error
	}{
		{
			name: "spot", a: asset.Spot, exp: []InstrumentResponse{spotInstrument}, cached: map[string][]InstrumentResponse{instTypeSpot: {spotInstrument}}, codes: map[string]uint64{"BTC-USDT": 1},
			requests: []string{"/public/instruments?instType=SPOT"},
		},
		{
			name: "margin", a: asset.Margin, exp: []InstrumentResponse{marginInstrument}, cached: map[string][]InstrumentResponse{instTypeMargin: {marginInstrument}}, codes: map[string]uint64{"BTC-USDT": 2},
			requests: []string{"/public/instruments?instType=MARGIN"},
		},
		{
			name: "futures", a: asset.Futures, exp: []InstrumentResponse{futuresInstrument}, cached: map[string][]InstrumentResponse{instTypeFutures: {futuresInstrument}}, codes: map[string]uint64{"BTC-USD-261225": 3},
			requests: []string{"/public/instruments?instType=FUTURES"},
		},
		{
			name: "perpetual swap", a: asset.PerpetualSwap, exp: []InstrumentResponse{swapInstrument}, cached: map[string][]InstrumentResponse{instTypeSwap: {swapInstrument}}, codes: map[string]uint64{"BTC-USDT-SWAP": 4},
			requests: []string{"/public/instruments?instType=SWAP"},
		},
		{
			name: "options", a: asset.Options, exp: []InstrumentResponse{optionInstrument}, cached: map[string][]InstrumentResponse{instTypeOption: {optionInstrument}}, codes: map[string]uint64{"BTC-USD-261225-100000-C": 5},
			requests: []string{"/public/underlying?instType=OPTION", "/public/instruments?instType=OPTION&uly=BTC-USD"},
		},
		{name: "unsupported asset", a: asset.USDTMarginedFutures, cached: map[string][]InstrumentResponse{}, codes: map[string]uint64{}, err: asset.ErrNotSupported},
		{name: "spread has no instrument type", a: asset.Spread, cached: map[string][]InstrumentResponse{}, codes: map[string]uint64{}, err: errInvalidInstrumentType},
		{
			name: "listing error caches nothing", a: asset.Spot, failing: "/public/instruments?instType=SPOT", cached: map[string][]InstrumentResponse{}, codes: map[string]uint64{},
			requests: []string{"/public/instruments?instType=SPOT"}, err: ErrorCodes["50011"],
		},
		{
			name: "option listing error caches nothing", a: asset.Options, failing: "/public/instruments?instType=OPTION&uly=BTC-USD", cached: map[string][]InstrumentResponse{}, codes: map[string]uint64{},
			requests: []string{"/public/underlying?instType=OPTION", "/public/instruments?instType=OPTION&uly=BTC-USD"}, err: ErrorCodes["50011"],
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &instrumentFetchRecorder{underlyings: []string{"BTC-USD"}, rows: rows, failing: tc.failing}
			ex := newMockExchange(t, f.serve(t))
			result, err := ex.getInstrumentsForAsset(t.Context(), tc.a)
			require.ErrorIs(t, err, tc.err, "getInstrumentsForAsset must return the expected error")
			assert.Equal(t, tc.exp, result, "getInstrumentsForAsset should return the asset's instruments")
			assert.Equal(t, tc.requests, f.received(), "getInstrumentsForAsset should list the asset's instrument type")
			assert.Equal(t, tc.cached, ex.instrumentsInfoMap, "getInstrumentsForAsset should cache the instruments under their instrument type")
			assert.Equal(t, tc.codes, ex.instrumentIDCodeMap, "getInstrumentsForAsset should cache the instruments' ID codes")
		})
	}

	t.Run("instruments pushed during the listing keep their codes", func(t *testing.T) {
		t.Parallel()
		push := func(rows string) []byte {
			return []byte(`{"arg":{"channel":"instruments","instType":"SPOT"},"data":[` + rows + `]}`)
		}
		btcRow, btc := instrumentRow(t, instTypeSpot, "BTC-USDT", "", 31)
		ethRow, eth := instrumentRow(t, instTypeSpot, "ETH-USDT", "", 41)
		solRow, sol := instrumentRow(t, instTypeSpot, "SOL-USDT", "", 51)
		var ex *Exchange
		ex = newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/public/instruments", r.URL.Path, "getInstrumentsForAsset should list the instruments")
			// While the listing is in flight, a push changes BTC-USDT's code and removes ETH-USDT's
			assert.NoError(t, ex.wsHandleData(r.Context(), nil, push(`{"instType":"SPOT","instId":"BTC-USDT","instIdCode":72},{"instType":"SPOT","instId":"ETH-USDT","instIdCode":null}`)), "the instruments push should not error")
			writeOKXData(t, w, []map[string]any{btcRow, ethRow, solRow})
		}))
		require.NoError(t, ex.wsHandleData(t.Context(), nil, push(`{"instType":"SPOT","instId":"ETH-USDT","instIdCode":40},{"instType":"SPOT","instId":"SOL-USDT","instIdCode":50}`)), "the instruments push before the listing must not error")
		result, err := ex.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "getInstrumentsForAsset must not error")
		assert.Equal(t, []InstrumentResponse{btc, eth, sol}, result, "getInstrumentsForAsset should return the listing")
		assert.Equal(t, map[string]uint64{"BTC-USDT": 72, "SOL-USDT": 51}, ex.instrumentIDCodeMap, "the listing should not restore codes pushed or removed after it was requested")
	})

	t.Run("a listing answered after a newer one keeps the newer list, codes and states", func(t *testing.T) {
		t.Parallel()
		olderRow, _ := instrumentRow(t, instTypeSpot, "BTC-USDT", "", 31)
		olderRow["state"] = instrumentStatePreOpen
		newerRow, newer := instrumentRow(t, instTypeSpot, "BTC-USDT", "", 72)
		newerRow["state"], newer.State = instrumentStateLive, instrumentStateLive
		ex, entered, release := newOverlappingListingsExchange(t, []map[string]any{olderRow}, []map[string]any{newerRow})
		require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(`{"arg":{"channel":"instruments","instType":"SPOT"},"data":[{"instType":"SPOT","instId":"BTC-USDT","state":"preopen","instIdCode":31}]}`)), "the instruments push before the listings must not error")
		older := make(chan error, 1)
		go func() {
			_, err := ex.getInstrumentsForAsset(t.Context(), asset.Spot)
			older <- err
		}()
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the older listing must be requested")
		}
		result, err := ex.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "the newer listing must not error")
		assert.Equal(t, []InstrumentResponse{newer}, result, "the newer listing should be returned")
		release()
		select {
		case err := <-older:
			require.NoError(t, err, "the older listing must not error")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the older listing must return")
		}
		assert.Equal(t, map[string][]InstrumentResponse{instTypeSpot: {newer}}, ex.instrumentsInfoMap, "the older listing should not replace the newer list")
		assert.Equal(t, map[string]uint64{"BTC-USDT": 72}, ex.instrumentIDCodeMap, "the older listing should not restore its code")
		assert.Empty(t, ex.instrumentTradingStates, "the older listing should not restore its pre-open state")
	})

	t.Run("a listing answered after a newer one omitting an instrument writes nothing of it and returns the newer list", func(t *testing.T) {
		t.Parallel()
		btcRow, _ := instrumentRow(t, instTypeSpot, "BTC-USDT", "", 31)
		btcRow["state"] = instrumentStatePreOpen
		ethRow, eth := instrumentRow(t, instTypeSpot, "ETH-USDT", "", 72)
		ethRow["state"], eth.State = instrumentStateLive, instrumentStateLive
		ex, entered, release := newOverlappingListingsExchange(t, []map[string]any{btcRow}, []map[string]any{ethRow})
		type listing struct {
			instruments []InstrumentResponse
			err         error
		}
		older := make(chan listing, 1)
		go func() {
			instruments, err := ex.getInstrumentsForAsset(t.Context(), asset.Spot)
			older <- listing{instruments, err}
		}()
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the older listing must be requested")
		}
		_, err := ex.getInstrumentsForAsset(t.Context(), asset.Spot)
		require.NoError(t, err, "the newer listing must not error")
		release()
		select {
		case got := <-older:
			require.NoError(t, got.err, "the older listing must not error")
			assert.Equal(t, []InstrumentResponse{eth}, got.instruments, "the older listing should return the newer list")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the older listing must return")
		}
		assert.Equal(t, map[string][]InstrumentResponse{instTypeSpot: {eth}}, ex.instrumentsInfoMap, "the older listing should not replace the newer list")
		assert.Equal(t, map[string]uint64{"ETH-USDT": 72}, ex.instrumentIDCodeMap, "the older listing should not add the code of an instrument the newer omits")
		assert.Empty(t, ex.instrumentTradingStates, "the older listing should not add the pre-open state of an instrument the newer omits")
		_, err = ex.prepareWebsocketPlaceOrder(t.Context(), &order.Submit{Exchange: ex.Name, AssetType: asset.Spot, Pair: mainPair, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1, MarginType: margin.NoMargin})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an order for the instrument the newer listing omits should have no code to send")
	})
}

// newOverlappingListingsExchange returns an exchange whose first instrument
// listing is answered with older once released, while later listings are
// answered with newer at once; entered closes once the first arrives
func newOverlappingListingsExchange(t *testing.T, older, newer []map[string]any) (ex *Exchange, entered <-chan struct{}, release func()) {
	t.Helper()
	first, released := make(chan struct{}), make(chan struct{})
	var calls atomic.Int64
	ex = newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/public/instruments", r.URL.Path, "the listings should request the instruments")
		if calls.Add(1) != 1 {
			writeOKXData(t, w, newer)
			return
		}
		close(first)
		select {
		case <-released:
		case <-r.Context().Done():
			return
		}
		writeOKXData(t, w, older)
	}))
	var once sync.Once
	release = func() { once.Do(func() { close(released) }) }
	t.Cleanup(release)
	return ex, first, release
}

func TestCacheInstruments(t *testing.T) {
	t.Parallel()
	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)
	preOpen := instrumentTradingState{preOpen: true}
	for _, tc := range []struct {
		name               string
		lists              map[string][]InstrumentResponse
		listGenerations    map[string]uint64
		codes              map[string]uint64
		states             map[string]instrumentTradingState
		generations        map[string]uint64
		generation         uint64
		instruments        []InstrumentResponse
		expLists           map[string][]InstrumentResponse
		expListGenerations map[string]uint64
		expCodes           map[string]uint64
		expStates          map[string]instrumentTradingState
		expGenerations     map[string]uint64
	}{
		{
			name:               "keeps the code and state of an instrument pushed after the listing was requested",
			lists:              map[string][]InstrumentResponse{},
			codes:              map[string]uint64{"BTC-USDT": 72, "ETH-USDT": 40},
			states:             map[string]instrumentTradingState{"BTC-USDT": preOpen, "ETH-USDT": preOpen},
			generations:        map[string]uint64{"BTC-USDT": 3, "ETH-USDT": 2},
			generation:         2,
			instruments:        []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 31, State: instrumentStateLive}, {InstrumentID: ethUSDT, InstrumentIDCode: 41, State: instrumentStateLive}},
			expLists:           map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 31, State: instrumentStateLive}, {InstrumentID: ethUSDT, InstrumentIDCode: 41, State: instrumentStateLive}}},
			expListGenerations: map[string]uint64{instTypeSpot: 2},
			expCodes:           map[string]uint64{"BTC-USDT": 72, "ETH-USDT": 41},
			expStates:          map[string]instrumentTradingState{"BTC-USDT": preOpen},
			expGenerations:     map[string]uint64{"BTC-USDT": 3, "ETH-USDT": 2},
		},
		{
			name:               "stores the list and indexes its codes",
			lists:              map[string][]InstrumentResponse{},
			codes:              map[string]uint64{},
			instruments:        []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 12345}, {InstrumentID: ethUSDT, InstrumentIDCode: 67890}},
			expLists:           map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 12345}, {InstrumentID: ethUSDT, InstrumentIDCode: 67890}}},
			expListGenerations: map[string]uint64{instTypeSpot: 0},
			expCodes:           map[string]uint64{"BTC-USDT": 12345, "ETH-USDT": 67890},
			expGenerations:     map[string]uint64{"BTC-USDT": 0, "ETH-USDT": 0},
		},
		{
			name:               "replaces the type's list and keeps other types",
			lists:              map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 1}}, instTypeSwap: {{InstrumentID: perpetualSwapPair, InstrumentIDCode: 3}}},
			codes:              map[string]uint64{"BTC-USDT": 1, "BTC-USDT-SWAP": 3},
			instruments:        []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 2}},
			expLists:           map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 2}}, instTypeSwap: {{InstrumentID: perpetualSwapPair, InstrumentIDCode: 3}}},
			expListGenerations: map[string]uint64{instTypeSpot: 0},
			expCodes:           map[string]uint64{"BTC-USDT": 2, "BTC-USDT-SWAP": 3},
			expGenerations:     map[string]uint64{"BTC-USDT": 0},
		},
		{
			name:               "a null code drops the cached code",
			lists:              map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 1}}},
			codes:              map[string]uint64{"BTC-USDT": 1},
			instruments:        []InstrumentResponse{{InstrumentID: mainPair}},
			expLists:           map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair}}},
			expListGenerations: map[string]uint64{instTypeSpot: 0},
			expCodes:           map[string]uint64{},
			expGenerations:     map[string]uint64{"BTC-USDT": 0},
		},
		{
			name:               "an older listing writes nothing over a newer listing",
			lists:              map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 72, State: instrumentStateLive}}},
			listGenerations:    map[string]uint64{instTypeSpot: 3},
			codes:              map[string]uint64{"BTC-USDT": 72},
			generations:        map[string]uint64{"BTC-USDT": 3},
			generation:         2,
			instruments:        []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 31, State: instrumentStatePreOpen}, {InstrumentID: ethUSDT, InstrumentIDCode: 41, State: instrumentStatePreOpen}},
			expLists:           map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 72, State: instrumentStateLive}}},
			expListGenerations: map[string]uint64{instTypeSpot: 3},
			expCodes:           map[string]uint64{"BTC-USDT": 72},
			expGenerations:     map[string]uint64{"BTC-USDT": 3},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.instrumentsInfoMap, ex.instrumentListGenerations, ex.instrumentIDCodeMap, ex.instrumentTradingStates, ex.instrumentGenerations = tc.lists, tc.listGenerations, tc.codes, tc.states, tc.generations
			assert.Equal(t, tc.expLists[instTypeSpot], ex.cacheInstruments(instTypeSpot, tc.instruments, tc.generation), "cacheInstruments should return the list stored for the instrument type")
			assert.Equal(t, tc.expLists, ex.instrumentsInfoMap, "cacheInstruments should store the list under its instrument type")
			assert.Equal(t, tc.expListGenerations, ex.instrumentListGenerations, "cacheInstruments should record the listing which stored each list")
			assert.Equal(t, tc.expCodes, ex.instrumentIDCodeMap, "cacheInstruments should index the list's instrument ID codes")
			assert.Equal(t, tc.expStates, ex.instrumentTradingStates, "cacheInstruments should index the list's pre-open instruments")
			assert.Equal(t, tc.expGenerations, ex.instrumentGenerations, "cacheInstruments should record the listing which set each instrument")
		})
	}

	t.Run("indexes ID codes beside instrument pushes", func(t *testing.T) {
		t.Parallel()
		fresh := new(Exchange)
		fresh.instrumentsInfoMap = make(map[string][]InstrumentResponse)
		fresh.instrumentIDCodeMap = make(map[string]uint64)

		solUSDT := currency.NewPairWithDelimiter(currency.SOL.String(), currency.USDT.String(), currency.DashDelimiter)

		fresh.cacheInstruments(instTypeSpot, []InstrumentResponse{
			{InstrumentID: mainPair, InstrumentIDCode: 12345},
			{InstrumentID: ethUSDT, InstrumentIDCode: 67890},
		}, 0)
		assert.EqualValues(t, 12345, fresh.getInstrumentIDCode("BTC-USDT"), "cached BTC-USDT should resolve its instrument ID code")
		assert.EqualValues(t, 67890, fresh.getInstrumentIDCode("ETH-USDT"), "cached ETH-USDT should resolve its instrument ID code")

		// Instruments channel pushes carry deltas, so they upsert codes without
		// replacing the full instrument lists.
		fresh.cacheInstrumentPush([]InstrumentResponse{
			{InstrumentID: solUSDT, InstrumentIDCode: 11111},
		})
		assert.EqualValues(t, 11111, fresh.getInstrumentIDCode("SOL-USDT"), "a pushed SOL-USDT should upsert its instrument ID code")
		assert.Len(t, fresh.instrumentsInfoMap[instTypeSpot], 2, "an instruments push should not replace the cached instrument list")

		// A push omitting instIdCode decodes to zero and drops the cached code, so
		// a relisted instrument's websocket orders fail rather than go out with its
		// replaced code.
		fresh.cacheInstrumentPush([]InstrumentResponse{
			{InstrumentID: mainPair},
			{InstrumentID: solUSDT, InstrumentIDCode: 22222},
		})
		assert.Zero(t, fresh.getInstrumentIDCode("BTC-USDT"), "a push without a code should drop the cached instrument ID code")
		assert.EqualValues(t, 22222, fresh.getInstrumentIDCode("SOL-USDT"), "a push with a code should still upsert it")

		assert.Zero(t, fresh.getInstrumentIDCode("DOGE-USDT"), "an uncached instrument ID should resolve a zero code")
	})

	t.Run("waits for the instruments lock", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.instrumentsInfoMap = make(map[string][]InstrumentResponse)
		ex.instrumentIDCodeMap = make(map[string]uint64)
		requireWaitsForLock(t, &ex.instrumentsInfoMapLock, func() {
			ex.cacheInstruments(instTypeSpot, []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 12345}}, 0)
		})
		assert.Equal(t, map[string]uint64{"BTC-USDT": 12345}, ex.instrumentIDCodeMap, "cacheInstruments should cache the code once it holds the lock")
	})
}

func TestNextInstrumentGeneration(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.instrumentIDCodeMap = make(map[string]uint64)
	assert.EqualValues(t, 1, ex.nextInstrumentGeneration(), "the first listing should be counted first")
	ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 1}})
	var generation uint64
	requireWaitsForLock(t, &ex.instrumentsInfoMapLock, func() { generation = ex.nextInstrumentGeneration() })
	assert.EqualValues(t, 3, generation, "nextInstrumentGeneration should count every listing and push")
}

func TestInstrumentTradingStatePreOpenAt(t *testing.T) {
	t.Parallel()
	switchTime := time.UnixMilli(1700000001000)
	for _, tc := range []struct {
		name  string
		state instrumentTradingState
		at    time.Time
		exp   bool
	}{
		{name: "pre-open without a switch time", state: instrumentTradingState{preOpen: true}, at: switchTime.Add(time.Hour), exp: true},
		{name: "live without a switch time", at: switchTime.Add(-time.Hour)},
		{name: "pre-open before its switch", state: instrumentTradingState{preOpen: true, continuousTradingSwitchTime: switchTime}, at: switchTime.Add(-time.Millisecond), exp: true},
		{name: "pre-open at its switch", state: instrumentTradingState{preOpen: true, continuousTradingSwitchTime: switchTime}, at: switchTime},
		{name: "pre-open after its switch", state: instrumentTradingState{preOpen: true, continuousTradingSwitchTime: switchTime}, at: switchTime.Add(time.Millisecond)},
		{name: "live before its switch", state: instrumentTradingState{continuousTradingSwitchTime: switchTime}, at: switchTime.Add(-time.Millisecond), exp: true},
		{name: "live after its switch", state: instrumentTradingState{continuousTradingSwitchTime: switchTime}, at: switchTime.Add(time.Millisecond)},
		{name: "pre-open at no time", state: instrumentTradingState{preOpen: true, continuousTradingSwitchTime: switchTime}, exp: true},
		{name: "live at no time", state: instrumentTradingState{continuousTradingSwitchTime: switchTime}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.state.preOpenAt(tc.at), "preOpenAt should judge the state by the switch time when known, and otherwise by the state")
		})
	}
}

func TestInstrumentPreOpen(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	at := time.UnixMilli(1700000000999)
	assert.False(t, ex.instrumentPreOpen("BTC-USDT", at), "an instrument without a cached state should not be pre-open")
	ex.instrumentTradingStates = map[string]instrumentTradingState{"BTC-USDT": {continuousTradingSwitchTime: time.UnixMilli(1700000001000)}}
	var preOpen bool
	requireWaitsForLock(t, &ex.instrumentsInfoMapLock, func() { preOpen = ex.instrumentPreOpen("BTC-USDT", at) })
	assert.True(t, preOpen, "an instrument before its continuous trading switch should be pre-open once the lock is free")
	assert.False(t, ex.instrumentPreOpen("BTC-USDT", at.Add(time.Millisecond)), "an instrument at its continuous trading switch should not be pre-open")
	assert.False(t, ex.instrumentPreOpen("ETH-USDT", at), "another instrument should not be pre-open")
}

// requireWaitsForLock holds lock while run runs, and checks run waits for the lock to be freed before it returns
func requireWaitsForLock(t *testing.T, lock sync.Locker, run func()) {
	t.Helper()
	lock.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		run()
	}()
	finished := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}
	assert.Never(t, finished, 50*time.Millisecond, 5*time.Millisecond, "the call should wait while another holds the lock")
	lock.Unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the call must return once the lock is free")
	}
}

func TestCacheInstrumentPush(t *testing.T) {
	t.Parallel()
	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)
	solUSDT := currency.NewPairWithDelimiter("SOL", "USDT", currency.DashDelimiter)

	t.Run("upserts codes and leaves the instrument lists alone", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.instrumentsInfoMap = map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 1}, {InstrumentID: ethUSDT, InstrumentIDCode: 2}}}
		ex.instrumentIDCodeMap = map[string]uint64{"BTC-USDT": 1, "ETH-USDT": 2}
		ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: solUSDT, InstrumentIDCode: 3}, {InstrumentID: mainPair}, {InstrumentID: ethUSDT, InstrumentIDCode: 4}})
		assert.Equal(t, map[string]uint64{"ETH-USDT": 4, "SOL-USDT": 3}, ex.instrumentIDCodeMap, "cacheInstrumentPush should upsert the pushed codes and drop null ones")
		exp := map[string][]InstrumentResponse{instTypeSpot: {{InstrumentID: mainPair, InstrumentIDCode: 1}, {InstrumentID: ethUSDT, InstrumentIDCode: 2}}}
		assert.Equal(t, exp, ex.instrumentsInfoMap, "cacheInstrumentPush should leave the instrument lists alone")
	})

	t.Run("counts the pushes and records which changed each instrument", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.instrumentIDCodeMap = make(map[string]uint64)
		ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 1, State: instrumentStatePreOpen}, {InstrumentID: ethUSDT, InstrumentIDCode: 2, State: instrumentStatePreOpen}})
		ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: ethUSDT, InstrumentIDCode: 2, State: instrumentStateLive}})
		assert.EqualValues(t, 2, ex.instrumentGeneration, "cacheInstrumentPush should count each push")
		assert.Equal(t, map[string]uint64{"BTC-USDT": 1, "ETH-USDT": 2}, ex.instrumentGenerations, "cacheInstrumentPush should record the push which last changed each instrument")
		assert.Equal(t, map[string]instrumentTradingState{"BTC-USDT": {preOpen: true}}, ex.instrumentTradingStates, "cacheInstrumentPush should track the pushed pre-open states")
	})

	t.Run("waits for the instruments lock", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.instrumentIDCodeMap = make(map[string]uint64)
		requireWaitsForLock(t, &ex.instrumentsInfoMapLock, func() { ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: solUSDT, InstrumentIDCode: 3}}) })
		assert.Equal(t, map[string]uint64{"SOL-USDT": 3}, ex.instrumentIDCodeMap, "cacheInstrumentPush should cache the code once it holds the lock")
	})
}

func TestCacheInstrumentsLocked(t *testing.T) {
	t.Parallel()
	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)
	solUSDT := currency.NewPairWithDelimiter("SOL", "USDT", currency.DashDelimiter)
	preOpen := instrumentTradingState{preOpen: true}
	switchTime := time.UnixMilli(1700000001000)
	for _, tc := range []struct {
		name           string
		cached         map[string]uint64
		states         map[string]instrumentTradingState
		generations    map[string]uint64
		generation     uint64
		instruments    []InstrumentResponse
		exp            map[string]uint64
		expStates      map[string]instrumentTradingState
		expGenerations map[string]uint64
	}{
		{name: "no instruments", cached: map[string]uint64{"BTC-USDT": 12345}, exp: map[string]uint64{"BTC-USDT": 12345}},
		{
			name: "new codes", cached: map[string]uint64{},
			instruments:    []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 12345}, {InstrumentID: ethUSDT, InstrumentIDCode: 67890}},
			exp:            map[string]uint64{"BTC-USDT": 12345, "ETH-USDT": 67890},
			expGenerations: map[string]uint64{"BTC-USDT": 0, "ETH-USDT": 0},
		},
		{name: "a changed code replaces the cached one", cached: map[string]uint64{"BTC-USDT": 12345}, instruments: []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 54321}}, exp: map[string]uint64{"BTC-USDT": 54321}, expGenerations: map[string]uint64{"BTC-USDT": 0}},
		{name: "a null code drops the cached one", cached: map[string]uint64{"BTC-USDT": 12345, "ETH-USDT": 67890}, instruments: []InstrumentResponse{{InstrumentID: mainPair}}, exp: map[string]uint64{"ETH-USDT": 67890}, expGenerations: map[string]uint64{"BTC-USDT": 0}},
		{name: "a null code for an uncached instrument", cached: map[string]uint64{"BTC-USDT": 12345}, instruments: []InstrumentResponse{{InstrumentID: solUSDT}}, exp: map[string]uint64{"BTC-USDT": 12345}, expGenerations: map[string]uint64{"SOL-USDT": 0}},
		{
			name: "a pre-open instrument is recorded", cached: map[string]uint64{},
			instruments: []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 1, State: instrumentStatePreOpen}},
			exp:         map[string]uint64{"BTC-USDT": 1}, expStates: map[string]instrumentTradingState{"BTC-USDT": preOpen}, expGenerations: map[string]uint64{"BTC-USDT": 0},
		},
		{
			name: "an instrument out of pre-open is dropped", cached: map[string]uint64{"BTC-USDT": 1}, states: map[string]instrumentTradingState{"BTC-USDT": preOpen, "ETH-USDT": preOpen},
			instruments: []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 1, State: instrumentStateLive}},
			exp:         map[string]uint64{"BTC-USDT": 1}, expStates: map[string]instrumentTradingState{"ETH-USDT": preOpen}, expGenerations: map[string]uint64{"BTC-USDT": 0},
		},
		{
			name: "a pre-open instrument's continuous trading switch is recorded", cached: map[string]uint64{},
			instruments: []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 1, State: instrumentStatePreOpen, ContinuousTradingSwitchTime: types.Time(switchTime)}},
			exp:         map[string]uint64{"BTC-USDT": 1}, expStates: map[string]instrumentTradingState{"BTC-USDT": {preOpen: true, continuousTradingSwitchTime: switchTime}}, expGenerations: map[string]uint64{"BTC-USDT": 0},
		},
		{
			name: "a live instrument keeps its continuous trading switch", cached: map[string]uint64{"BTC-USDT": 1}, states: map[string]instrumentTradingState{"BTC-USDT": {preOpen: true, continuousTradingSwitchTime: switchTime}},
			instruments: []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 1, State: instrumentStateLive, ContinuousTradingSwitchTime: types.Time(switchTime)}},
			exp:         map[string]uint64{"BTC-USDT": 1}, expStates: map[string]instrumentTradingState{"BTC-USDT": {continuousTradingSwitchTime: switchTime}}, expGenerations: map[string]uint64{"BTC-USDT": 0},
		},
		{
			name: "a futures contract's listing time is its switch to continuous trading", cached: map[string]uint64{},
			instruments: []InstrumentResponse{{InstrumentType: instTypeFutures, InstrumentID: websocketFuturesPair, InstrumentIDCode: 1, State: instrumentStatePreOpen, ListTime: types.Time(switchTime)}},
			exp:         map[string]uint64{"BTC-USD-261127": 1}, expStates: map[string]instrumentTradingState{"BTC-USD-261127": {preOpen: true, continuousTradingSwitchTime: switchTime}}, expGenerations: map[string]uint64{"BTC-USD-261127": 0},
		},
		{
			name: "a live swap keeps its listing time as its switch to continuous trading", cached: map[string]uint64{},
			instruments: []InstrumentResponse{{InstrumentType: instTypeSwap, InstrumentID: perpetualSwapPair, InstrumentIDCode: 1, State: instrumentStateLive, ListTime: types.Time(switchTime)}},
			exp:         map[string]uint64{"BTC-USDT-SWAP": 1}, expStates: map[string]instrumentTradingState{"BTC-USDT-SWAP": {continuousTradingSwitchTime: switchTime}}, expGenerations: map[string]uint64{"BTC-USDT-SWAP": 0},
		},
		{
			name: "a margin instrument's continuous trading switch follows its listing time", cached: map[string]uint64{},
			instruments: []InstrumentResponse{{InstrumentType: instTypeMargin, InstrumentID: mainPair, InstrumentIDCode: 1, State: instrumentStatePreOpen, ListTime: types.Time(switchTime.Add(-time.Hour)), ContinuousTradingSwitchTime: types.Time(switchTime)}},
			exp:         map[string]uint64{"BTC-USDT": 1}, expStates: map[string]instrumentTradingState{"BTC-USDT": {preOpen: true, continuousTradingSwitchTime: switchTime}}, expGenerations: map[string]uint64{"BTC-USDT": 0},
		},
		{
			name: "an option's listing time is not its switch to continuous trading", cached: map[string]uint64{},
			instruments: []InstrumentResponse{{InstrumentType: instTypeOption, InstrumentID: websocketOptionPair, InstrumentIDCode: 1, State: instrumentStatePreOpen, ListTime: types.Time(switchTime)}},
			exp:         map[string]uint64{"BTC-USD-261127-71000-P": 1}, expStates: map[string]instrumentTradingState{"BTC-USD-261127-71000-P": preOpen}, expGenerations: map[string]uint64{"BTC-USD-261127-71000-P": 0},
		},
		{
			name: "an instrument set since is left alone", cached: map[string]uint64{"BTC-USDT": 72}, states: map[string]instrumentTradingState{"BTC-USDT": preOpen}, generations: map[string]uint64{"BTC-USDT": 2}, generation: 1,
			instruments: []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 31, State: instrumentStateLive}},
			exp:         map[string]uint64{"BTC-USDT": 72}, expStates: map[string]instrumentTradingState{"BTC-USDT": preOpen}, expGenerations: map[string]uint64{"BTC-USDT": 2},
		},
		{
			name: "an instrument set before is replaced", cached: map[string]uint64{"BTC-USDT": 72}, states: map[string]instrumentTradingState{"BTC-USDT": preOpen}, generations: map[string]uint64{"BTC-USDT": 1}, generation: 1,
			instruments: []InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 31, State: instrumentStateLive}},
			exp:         map[string]uint64{"BTC-USDT": 31}, expStates: map[string]instrumentTradingState{}, expGenerations: map[string]uint64{"BTC-USDT": 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.instrumentIDCodeMap, ex.instrumentTradingStates, ex.instrumentGenerations = tc.cached, tc.states, tc.generations
			ex.cacheInstrumentsLocked(tc.instruments, tc.generation)
			assert.Equal(t, tc.exp, ex.instrumentIDCodeMap, "cacheInstrumentsLocked should upsert each code and drop null ones")
			assert.Equal(t, tc.expStates, ex.instrumentTradingStates, "cacheInstrumentsLocked should track each pre-open state")
			assert.Equal(t, tc.expGenerations, ex.instrumentGenerations, "cacheInstrumentsLocked should record the listing or push which set each instrument")
		})
	}
}

func TestSpreadOrderTypeFilterRequests(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		orderType order.Type
		tif       order.TimeInForce
		exp       string
		expErr    error
	}{
		{"limit spans no filter", order.Limit, order.UnknownTIF, "", nil},
		{"good till cancel limit", order.Limit, order.GoodTillCancel, orderLimit, nil},
		{"post only limit", order.Limit, order.PostOnly, orderPostOnly, nil},
		{"immediate or cancel limit", order.Limit, order.ImmediateOrCancel, orderIOC, nil},
		{"market", order.Market, order.UnknownTIF, orderMarket, nil},
		{"fill or kill market rejected", order.Market, order.FillOrKill, "", order.ErrUnsupportedOrderType},
		{"fill or kill limit rejected", order.Limit, order.FillOrKill, "", order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotOrdType string
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/sprd/orders-pending" {
					t.Errorf("unexpected request path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				gotOrdType = r.URL.Query().Get("ordType")
				writeOKXData(t, w, []map[string]string{{
					"sprdId": "BTC-USDT_BTC-USDT", "ordId": "1", "ordType": orderLimit,
					"side": "buy", "state": "live", "sz": "1", "accFillSz": "0", "px": "100",
				}})
			}))
			_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
				AssetType: asset.Spread, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide,
			})
			if tc.expErr != nil {
				assert.ErrorIs(t, err, tc.expErr, "GetActiveOrders should reject the type before sending")
				return
			}
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s %s", tc.orderType, tc.tif)
			assert.Equal(t, tc.exp, gotOrdType, "the spread query should send the documented ordType")
		})
	}
}

func TestPendingOrderTypeFilter(t *testing.T) {
	t.Parallel()
	pending := []map[string]string{
		{"instId": "BTC-USDT", "ordId": "LIMIT-1", "ordType": orderLimit, "side": "buy", "state": "live", "cTime": "1700000000008"},
		{"instId": "BTC-USDT", "ordId": "POST-1", "ordType": orderPostOnly, "side": "buy", "state": "live", "cTime": "1700000000007"},
		{"instId": "BTC-USDT", "ordId": "FOK-1", "ordType": orderFOK, "side": "buy", "state": "live", "cTime": "1700000000006"},
		{"instId": "BTC-USDT", "ordId": "IOC-1", "ordType": orderIOC, "side": "buy", "state": "live", "cTime": "1700000000005"},
		{"instId": "BTC-USDT", "ordId": "MMP-1", "ordType": orderMarketMakerProtection, "side": "buy", "state": "live", "cTime": "1700000000004"},
		{"instId": "BTC-USDT", "ordId": "MMP-PO-1", "ordType": orderMarketMakerProtectionAndPostOnly, "side": "buy", "state": "live", "cTime": "1700000000003"},
		{"instId": "BTC-USDT", "ordId": "OPTIMAL-1", "ordType": orderOptimalLimitIOC, "side": "buy", "state": "live", "cTime": "1700000000002"},
		{"instId": "BTC-USDT", "ordId": "RPI-1", "ordType": orderRPI, "side": "buy", "state": "live", "cTime": "1700000000001"},
		{"instId": "BTC-USDT", "ordId": "OPFOK-1", "ordType": orderOptionFOK, "side": "buy", "state": "live", "cTime": "1700000000000"},
	}
	var mu sync.Mutex
	var cancelled []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending":
			ordTypes := strings.Split(r.URL.Query().Get("ordType"), ",")
			ords := make([]map[string]string, 0, len(pending))
			for _, o := range pending {
				if ordTypes[0] == "" || slices.Contains(ordTypes, o["ordType"]) {
					ords = append(ords, o)
				}
			}
			writeOKXData(t, w, ords)
		case "/trade/cancel-batch-orders":
			var reqs []CancelOrderRequest
			if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
				t.Errorf("decoding cancel request body should not error: %v", err)
				return
			}
			respData := make([]map[string]string, 0, len(reqs))
			mu.Lock()
			for x := range reqs {
				cancelled = append(cancelled, reqs[x].OrderID)
				respData = append(respData, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
			}
			mu.Unlock()
			writeOKXData(t, w, respData)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	cancelAll := func(c *order.Cancel) ([]string, error) {
		mu.Lock()
		cancelled = nil
		mu.Unlock()
		_, err := e.CancelAllOrders(t.Context(), c)
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(cancelled), err
	}

	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       []string
		err       error
	}{
		{order.Limit, order.UnknownTIF, []string{"LIMIT-1", "POST-1", "FOK-1", "IOC-1", "RPI-1", "OPFOK-1"}, nil},
		{order.Limit, order.PostOnly, []string{"POST-1"}, nil},
		{order.Limit, order.FillOrKill, []string{"FOK-1", "OPFOK-1"}, nil},
		{order.Limit, order.ImmediateOrCancel, []string{"IOC-1"}, nil},
		{order.MarketMakerProtection, order.UnknownTIF, []string{"MMP-1", "MMP-PO-1"}, nil},
		{order.MarketMakerProtection, order.PostOnly, []string{"MMP-PO-1"}, nil},
		{order.Limit, order.GoodTillCancel | order.PostOnly, []string{"POST-1"}, nil},
		{order.MarketMakerProtection, order.GoodTillCancel | order.PostOnly, []string{"MMP-PO-1"}, nil},
		{order.OptimalLimit, order.ImmediateOrCancel, []string{"OPTIMAL-1"}, nil},
		{order.Limit, order.GoodTillDay, nil, order.ErrUnsupportedTimeInForce},
		{order.Limit, order.GoodTillTime, nil, order.ErrUnsupportedTimeInForce},
		{order.Limit, order.StopOrReduce, nil, order.ErrUnsupportedTimeInForce},
		{order.Limit, order.GoodTillCancel, []string{"LIMIT-1", "RPI-1"}, nil},
	} {
		resp, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: tc.orderType, TimeInForce: tc.tif, Side: order.AnySide})
		require.ErrorIsf(t, err, tc.err, "GetActiveOrders must refuse %s %s if no pending order reads back as it", tc.orderType, tc.tif)
		ids := make([]string, 0, len(resp))
		for x := range resp {
			ids = append(ids, resp[x].OrderID)
		}
		assert.ElementsMatchf(t, tc.exp, ids, "GetActiveOrders should return every order read back as %s %s", tc.orderType, tc.tif)

		got, err := cancelAll(&order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: tc.orderType, TimeInForce: tc.tif})
		require.ErrorIsf(t, err, tc.err, "CancelAllOrders must refuse %s %s if no pending order reads back as it", tc.orderType, tc.tif)
		assert.ElementsMatchf(t, tc.exp, got, "CancelAllOrders should cancel every order read back as %s %s, and nothing else", tc.orderType, tc.tif)
	}

	got, err := cancelAll(&order.Cancel{Type: order.Limit})
	require.NoError(t, err, "CancelAllOrders must not error for a cancel scoped by type alone")
	assert.ElementsMatch(t, []string{"LIMIT-1", "POST-1", "FOK-1", "IOC-1", "RPI-1", "OPFOK-1"}, got, "a cancel scoped by type alone should cancel every limit order")

	got, err = cancelAll(&order.Cancel{Type: order.Limit, OrderID: "POST-1"})
	require.NoError(t, err, "CancelAllOrders must not error for an order ID within the requested type")
	assert.Equal(t, []string{"POST-1"}, got, "an order ID within the requested type should be cancelled")

	got, err = cancelAll(&order.Cancel{Type: order.Limit, OrderID: "OPTIMAL-1"})
	require.NoError(t, err, "CancelAllOrders must not error for an order ID outside the requested type")
	assert.Empty(t, got, "an order ID outside the requested type should not be cancelled")

	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Stop, Side: order.AnySide})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "GetActiveOrders should reject an order type OKX does not list")
	_, err = cancelAll(&order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Stop})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "CancelAllOrders should reject an order type OKX does not list")
}

func TestWebsocketInstrumentIDCode(t *testing.T) {
	t.Parallel()
	fresh := new(Exchange)
	fresh.instrumentsInfoMap = make(map[string][]InstrumentResponse)
	fresh.instrumentIDCodeMap = map[string]uint64{"BTC-USDT": 12345}

	code, ok := fresh.websocketInstrumentIDCode("BTC-USDT")
	assert.True(t, ok, "a cached instrument should report availability")
	assert.EqualValues(t, 12345, code, "a cached instrument should report its code")

	code, ok = fresh.websocketInstrumentIDCode("ETH-USDT")
	assert.False(t, ok, "an uncached instrument should report unavailability")
	assert.Zero(t, code, "an uncached instrument should report a zero code")
}

func TestWebsocketTradingCapabilitiesDeclared(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	ws := e.Features.Supports.WebsocketCapabilities
	assert.True(t, ws.SubmitOrder, "WebsocketCapabilities should declare SubmitOrder")
	assert.True(t, ws.SubmitOrders, "WebsocketCapabilities should declare SubmitOrders")
	assert.True(t, ws.CancelOrder, "WebsocketCapabilities should declare CancelOrder")
	assert.True(t, ws.CancelOrders, "WebsocketCapabilities should declare CancelOrders")
	assert.True(t, ws.ModifyOrder, "WebsocketCapabilities should declare ModifyOrder")
}

// otherAccountContext returns a context carrying the credentials of an account other than the one
// newOwnAccountWebsocketExchange's connections log in as
func otherAccountContext(t *testing.T) context.Context {
	t.Helper()
	return accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"})
}

// primaryAccount and secondaryAccount return two accounts' credentials
func primaryAccount() *accounts.Credentials {
	return &accounts.Credentials{Key: "primary-key", Secret: "primary-secret", ClientID: "primary-passphrase"}
}

func secondaryAccount() *accounts.Credentials {
	return &accounts.Credentials{Key: "secondary-key", Secret: "secondary-secret", ClientID: "secondary-passphrase"}
}

// credentialSwitchContext changes an exchange's own credentials the first time a request reads it: a request reads
// the simulated trading flag just before its credentials, so an operation under it sees the exchange's credentials
// change after it chose its account and before it signed its first request
type credentialSwitchContext struct {
	context.Context //nolint:containedctx // credentialSwitchContext implements context.Context
	once            sync.Once
	ex              *Exchange
	next            *accounts.Credentials
}

// Value returns the parent's value for key, changing the exchange's credentials on the first read of the simulated
// trading flag
func (c *credentialSwitchContext) Value(k any) any {
	if k == testNetVal {
		c.once.Do(func() { c.ex.SetCredentials(c.next) })
	}
	return c.Context.Value(k)
}

// rotatedAccountContexts changes ex's own credentials to the account otherAccountContext carries after its connections
// logged in, returning the contexts an order for that account can carry: none, and the account's own credentials
func rotatedAccountContexts(t *testing.T, ex *Exchange) []context.Context {
	t.Helper()
	ex.SetCredentials(&accounts.Credentials{Key: "other-key", Secret: "other-secret", ClientID: "other-passphrase"})
	return []context.Context{t.Context(), otherAccountContext(t)}
}

// newOwnAccountWebsocketExchange returns a mock websocket exchange whose own credentials are own-key's, with a func
// returning the order operations its websocket received; any REST request but the account configuration, which a
// spot order's default trade mode follows, fails the test
func newOwnAccountWebsocketExchange(t *testing.T) (ex *Exchange, operations func() []string) {
	t.Helper()
	var mu sync.Mutex
	var ops []string
	ex = newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/account/config" {
			writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
			return
		}
		t.Errorf("unexpected REST request %s", r.URL)
		http.NotFound(w, r)
	}, func(op string, _ json.RawMessage) (string, any) {
		if op != operationLogin {
			mu.Lock()
			ops = append(ops, op)
			mu.Unlock()
		}
		return "0", []map[string]string{{"ordId": "1", "clOrdId": "", "sCode": "0", "sMsg": ""}}
	})
	ex.SetCredentials(&accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase"})
	loginMockWebsocket(t, ex)
	return ex, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(ops)
	}
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("refuses another account's context credentials", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		_, err := ex.WebsocketSubmitOrder(otherAccountContext(t), &order.Submit{Exchange: ex.Name, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spot})
		require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketSubmitOrder must refuse an order for another account than the connections act for")
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "the refusal should be a function not supported error")
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("refuses the exchange's credentials once they change from the login", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		for _, ctx := range rotatedAccountContexts(t, ex) {
			for _, s := range []*order.Submit{
				{Exchange: ex.Name, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spot},
				{Exchange: ex.Name, Pair: spreadPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spread},
			} {
				_, err := ex.WebsocketSubmitOrder(ctx, s)
				require.ErrorIsf(t, err, errWebsocketOtherAccount, "WebsocketSubmitOrder must refuse a %s order once the exchange's credentials differ from the connection's login", s.AssetType)
			}
		}
		assert.Empty(t, operations(), "no order operation should transmit")
	})
	t.Run("refuses its own key with another account's secret and passphrase", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		for _, creds := range []*accounts.Credentials{
			{Key: "own-key", Secret: "other-secret", ClientID: "own-passphrase"},
			{Key: "own-key", Secret: "own-secret", ClientID: "other-passphrase"},
		} {
			_, err := ex.WebsocketSubmitOrder(accounts.DeployCredentialsToContext(t.Context(), creds), &order.Submit{Exchange: ex.Name, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spot})
			require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketSubmitOrder must refuse credentials the connections did not log in with")
		}
		assert.Empty(t, operations(), "no order operation should transmit")
	})
	t.Run("acts for its own account under a sub-account override", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "own-key", Secret: "own-secret", ClientID: "own-passphrase", SubAccount: "sub", OneTimePassword: "123456"})
		_, err := ex.WebsocketSubmitOrder(ctx, &order.Submit{Exchange: ex.Name, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spot})
		require.NoError(t, err, "WebsocketSubmitOrder must place an order for the connections' own account")
		assert.Equal(t, []string{"order"}, operations(), "the order should transmit over the websocket")
	})

	t.Run("rejects an order before it transmits", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketSubmitOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrder must error for a nil submission")

		_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Trigger,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo order type should not transmit over the websocket")

		_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      spreadPair,
			Side:      order.Buy,
			Type:      order.OptimalLimit,
			Amount:    1,
			AssetType: asset.Spread,
		})
		assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "a spread order of a type OKX does not take for spreads should not transmit")

		_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")
	})
	t.Run("places a spread order on the business connection", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
			return "0", []map[string]string{{"clOrdId": "b15", "ordId": "12345689", "tag": "", "sCode": "0", "sMsg": ""}}
		}}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		resp, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:      ex.Name,
			Pair:          spreadPair,
			Side:          order.Buy,
			Type:          order.Limit,
			Amount:        1,
			Price:         -1.5,
			ClientOrderID: "b15",
			AssetType:     asset.Spread,
		})
		require.NoError(t, err, "WebsocketSubmitOrder must place a spread order")
		assert.Equal(t, "12345689", resp.OrderID, "WebsocketSubmitOrder should return the placed order's ID")
		frames := srv.requests("sprd-order")
		require.Len(t, frames, 1, "one spread order request must be sent")
		assert.Equal(t, "/ws/v5/business", frames[0].path, "the spread order should be sent on the business connection")
		assert.JSONEq(t, `[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","clOrdId":"b15","side":"buy","ordType":"limit","sz":"1","px":"-1.5"}]`, string(frames[0].arguments), "the spread order should carry its spread ID and differential price")
	})
	t.Run("rejects a spread reply without an order ID", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
			return "0", []map[string]string{{"clOrdId": "", "ordId": "", "tag": "", "sCode": "0", "sMsg": ""}}
		}}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spreadPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spread})
		assert.ErrorIs(t, err, common.ErrInvalidResponse, "a spread reply without an order ID should be rejected")
	})
	t.Run("reaches the websocket transport with a cached instrument code", func(t *testing.T) {
		t.Parallel()
		// The mock exchange below speaks REST only and fails the test on any
		// request but the account configuration a spot order's default trade
		// mode follows, so a pass proves the explicit websocket path neither
		// falls back to REST nor stops at instrument code resolution.
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
				return
			}
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		ex.instrumentsInfoMap = make(map[string][]InstrumentResponse)
		ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:  ex.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		})
		require.Error(t, err, "WebsocketSubmitOrder must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")
	})
	// Sends the ordType the REST placement path documents over the private
	// order frames.
	t.Run("orders send their documented ordType", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			assetType asset.Item
			orderType order.Type
			tif       order.TimeInForce
			exp       string
		}{
			{"plain limit", asset.Spot, order.Limit, order.UnknownTIF, orderLimit},
			{"post only", asset.Spot, order.Limit, order.PostOnly, orderPostOnly},
			{"good till cancel post only", asset.Spot, order.Limit, order.GoodTillCancel | order.PostOnly, orderPostOnly},
			{"fill or kill", asset.Spot, order.Limit, order.FillOrKill, orderFOK},
			{"immediate or cancel", asset.Spot, order.Limit, order.ImmediateOrCancel, orderIOC},
			// OKX takes MMP orders for options only.
			{"market maker protection", asset.Options, order.MarketMakerProtection, order.UnknownTIF, orderMarketMakerProtection},
			{"market maker protection post only", asset.Options, order.MarketMakerProtection, order.PostOnly, orderMarketMakerProtectionAndPostOnly},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var args string
				e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
						return
					}
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(op string, raw json.RawMessage) (string, any) {
					if op != "order" {
						t.Errorf("unexpected websocket operation %s", op)
						return "1", nil
					}
					mu.Lock()
					args = string(raw)
					mu.Unlock()
					return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
				})
				_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
					Exchange:    e.Name,
					Pair:        mainPair,
					AssetType:   tc.assetType,
					Side:        order.Buy,
					Type:        tc.orderType,
					TimeInForce: tc.tif,
					Amount:      1,
					Price:       1,
				})
				require.NoError(t, err, "WebsocketSubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				var sent []map[string]any
				require.NoError(t, json.Unmarshal([]byte(args), &sent), "the websocket order frame must decode")
				assert.Equal(t, tc.exp, sent[0]["ordType"], "the websocket limit order should send the documented ordType")
				assert.Equal(t, "1", sent[0]["px"], "the websocket order should send its price")
			})
		}
	})
	t.Run("a contract order sends its side and reduce-only flag", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var args string
		e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet}})
				return
			}
			t.Errorf("unexpected REST request %s", r.URL.Path)
			http.NotFound(w, r)
		}, func(op string, raw json.RawMessage) (string, any) {
			if op != "order" {
				t.Errorf("unexpected websocket operation %s", op)
				return "1", nil
			}
			mu.Lock()
			args = string(raw)
			mu.Unlock()
			return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
		})
		_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:   e.Name,
			Pair:       mainPair,
			AssetType:  asset.PerpetualSwap,
			Side:       order.Sell,
			Type:       order.Limit,
			Amount:     1,
			Price:      1,
			MarginType: margin.Multi,
			ReduceOnly: true,
		})
		require.NoError(t, err, "WebsocketSubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		var sent []map[string]any
		require.NoError(t, json.Unmarshal([]byte(args), &sent), "the websocket order frame must decode")
		assert.Equal(t, "sell", sent[0]["side"], "the websocket contract order should send OKX's required side")
		assert.Equal(t, positionSideNet, sent[0]["posSide"], "the websocket contract order should send the position side")
		assert.Equal(t, true, sent[0]["reduceOnly"], "the websocket contract order should carry the reduce-only flag")
	})
	// Places a perpetual swap with the side the account's confirmed position
	// mode requires, over the private order frames.
	t.Run("a perpetual swap follows the account position mode", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var args string
		e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"posMode": positionModeLongShort}})
				return
			}
			t.Errorf("unexpected REST request %s", r.URL.Path)
			http.NotFound(w, r)
		}, func(op string, raw json.RawMessage) (string, any) {
			if op != "order" {
				t.Errorf("unexpected websocket operation %s", op)
				return "1", nil
			}
			mu.Lock()
			args = string(raw)
			mu.Unlock()
			return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
		})
		_, err := e.WebsocketSubmitOrder(t.Context(), &order.Submit{
			Exchange:   e.Name,
			Pair:       mainPair,
			AssetType:  asset.PerpetualSwap,
			Side:       order.Sell,
			Type:       order.Limit,
			Amount:     1,
			Price:      1,
			MarginType: margin.Multi,
			ReduceOnly: true,
		})
		require.NoError(t, err, "WebsocketSubmitOrder must not error")
		mu.Lock()
		defer mu.Unlock()
		var sent []map[string]any
		require.NoError(t, json.Unmarshal([]byte(args), &sent), "the websocket order frame must decode")
		assert.Equal(t, "sell", sent[0]["side"], "the websocket contract order should send OKX's required side")
		assert.Equal(t, positionSideLong, sent[0]["posSide"], "a reduce-only sell should close the long position in long/short mode")
		_, has := sent[0]["reduceOnly"]
		assert.False(t, has, "long/short mode should not send reduceOnly")
	})
	t.Run("sends the order OKX documents", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name         string
			submit       *order.Submit
			accountLevel string // the acctLv a cross MARGIN order looks up, empty where none may be requested
			expFrame     string
		}{
			{
				name:     "isolated margin buy",
				submit:   &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.Isolated, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				expFrame: `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"isolated","ccy":"USDT","side":"buy","ordType":"limit","sz":"1","px":"100"}]`,
			},
			{
				name:     "spot lead trading",
				submit:   &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.SpotIsolated, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				expFrame: `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"spot_isolated","side":"buy","ordType":"limit","sz":"1","px":"100"}]`,
			},
			{
				name:         "quote-only spot market buy",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				accountLevel: "2",
				expFrame:     `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cash","side":"buy","ordType":"market","sz":"100","tgtCcy":"quote_ccy"}]`,
			},
			{
				name:         "margin market buy",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.5, QuoteAmount: 100},
				accountLevel: "3",
				expFrame:     `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cross","side":"buy","ordType":"market","sz":"100"}]`,
			},
			{
				name:         "margin market buy in futures mode",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.5, QuoteAmount: 100},
				accountLevel: "2",
				expFrame:     `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cross","ccy":"USDT","side":"buy","ordType":"market","sz":"100"}]`,
			},
			{
				name:         "quote-only spot market buy in cross mode",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Multi, Side: order.Buy, Type: order.Market, QuoteAmount: 100},
				accountLevel: "3",
				expFrame:     `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cross","side":"buy","ordType":"market","sz":"100"}]`,
			},
			{
				name:         "spot cross margin reduce only sell",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Multi, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 100, ReduceOnly: true},
				accountLevel: "3",
				expFrame:     `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cross","side":"sell","ordType":"limit","sz":"1","px":"100","reduceOnly":true}]`,
			},
			{
				name:     "spot isolated margin reduce only sell",
				submit:   &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.Isolated, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 100, ReduceOnly: true},
				expFrame: `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"isolated","ccy":"BTC","side":"sell","ordType":"limit","sz":"1","px":"100","reduceOnly":true}]`,
			},
			{
				name:     "spot cash reduce only sell",
				submit:   &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 100, ReduceOnly: true},
				expFrame: `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cash","side":"sell","ordType":"limit","sz":"1","px":"100"}]`,
			},
			{
				name:         "spot defaults to cross in multi-currency margin mode",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				accountLevel: "3",
				expFrame:     `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cross","side":"buy","ordType":"limit","sz":"1","px":"100"}]`,
			},
			{
				name:         "spot defaults to cash in spot mode",
				submit:       &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
				accountLevel: "1",
				expFrame:     `[{"instId":"BTC-USDT","instIdCode":12345,"tdMode":"cash","side":"buy","ordType":"limit","sz":"1","px":"100"}]`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var frames []string
				ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if tc.accountLevel != "" && r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": tc.accountLevel, "posMode": positionModeNet}})
						return
					}
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(op string, args json.RawMessage) (string, any) {
					assert.Equal(t, "order", op, "the order should be placed by the order operation")
					mu.Lock()
					frames = append(frames, string(args))
					mu.Unlock()
					return "0", []map[string]string{{"ordId": "1", "sCode": "0", "sMsg": ""}}
				})
				tc.submit.Exchange = ex.Name
				_, err := ex.WebsocketSubmitOrder(t.Context(), tc.submit)
				require.NoError(t, err, "WebsocketSubmitOrder must not error")
				mu.Lock()
				defer mu.Unlock()
				require.Len(t, frames, 1, "one order frame must be sent")
				assert.JSONEq(t, tc.expFrame, frames[0], "the order frame should carry the documented fields")
			})
		}
	})
	t.Run("refuses an order OKX cannot place as given", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			submit *order.Submit
			expErr error
		}{
			{"good till time", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillTime, Amount: 1, Price: 100, EndTime: time.UnixMilli(1800000000000)}, order.ErrUnsupportedTimeInForce},
			{"spread good till time", &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillTime | order.PostOnly, Amount: 1, Price: 1}, order.ErrUnsupportedTimeInForce},
			{"good till day", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillDay, Amount: 1, Price: 100}, order.ErrUnsupportedTimeInForce},
			{"stop or reduce", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.StopOrReduce, Amount: 1, Price: 100}, order.ErrUnsupportedTimeInForce},
			{"good till crossing", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillCrossing, Amount: 1, Price: 100}, order.ErrUnsupportedTimeInForce},
			{"spread good till day post only", &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillDay | order.PostOnly, Amount: 1, Price: 1}, order.ErrUnsupportedTimeInForce},
			{"options market", &order.Submit{Pair: currency.NewPairWithDelimiter("BTC", "USD-261225-50000-C", "-"), AssetType: asset.Options, Side: order.Buy, Type: order.Market, Amount: 1}, order.ErrUnsupportedOrderType},
			{"spot market maker protection", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.MarketMakerProtection, Amount: 1, Price: 100}, order.ErrUnsupportedOrderType},
			{"margin market buy sized in base", &order.Submit{Pair: mainPair, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.5}, order.ErrAmountMustBeSet},
			{"post only market", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, TimeInForce: order.PostOnly, Amount: 1}, order.ErrUnsupportedTimeInForce},
			{"post only optimal limit", &order.Submit{Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, Side: order.Buy, Type: order.OptimalLimit, TimeInForce: order.PostOnly, Amount: 1}, order.ErrUnsupportedTimeInForce},
			{"immediate or cancel market maker protection", &order.Submit{Pair: currency.NewPairWithDelimiter("BTC", "USD-261225-50000-C", "-"), AssetType: asset.Options, Side: order.Buy, Type: order.MarketMakerProtection, TimeInForce: order.ImmediateOrCancel, Amount: 1, Price: 0.05}, order.ErrUnsupportedTimeInForce},
			{"any type", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.AnyType, TimeInForce: order.ImmediateOrCancel, Amount: 1, Price: 100}, order.ErrUnsupportedOrderType},
			{"spread post only market", &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Market, TimeInForce: order.GoodTillCrossing | order.PostOnly, Amount: 1}, order.ErrUnsupportedTimeInForce},
			{"take profit and stop loss", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90}}}, errAttachedOrderUnsupported},
			{"spread take profit", &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, Amount: 1, Price: -5, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 1}}}, errAttachedOrderUnsupported},
			{"margin in cash mode", &order.Submit{Pair: mainPair, AssetType: asset.Margin, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100}, margin.ErrMarginTypeUnsupported},
			{"spot chase", &order.Submit{Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Chase, Amount: 1}, order.ErrUnsupportedOrderType},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(op string, _ json.RawMessage) (string, any) {
					t.Errorf("a refused order should not send the %s operation", op)
					return "1", nil
				})
				tc.submit.Exchange = ex.Name
				_, err := ex.WebsocketSubmitOrder(t.Context(), tc.submit)
				assert.ErrorIs(t, err, tc.expErr, "the order should be refused before it transmits")
			})
		}
	})
	t.Run("places a spread order at a zero price", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
			return "0", []map[string]string{{"clOrdId": "", "ordId": "12345689", "tag": "", "sCode": "0", "sMsg": ""}}
		}}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: spreadPair, Side: order.Buy, Type: order.Limit, Amount: 1, AssetType: asset.Spread})
		require.NoError(t, err, "WebsocketSubmitOrder must place a spread order at a zero price")
		frames := srv.requests("sprd-order")
		require.Len(t, frames, 1, "one spread order request must be sent")
		assert.JSONEq(t, `[{"sprdId":"BTC-USDT_BTC-USDT-SWAP","side":"buy","ordType":"limit","sz":"1","px":"0"}]`, string(frames[0].arguments), "a zero spread price should be sent rather than left out")
	})
	t.Run("rejects a reply row which confirms no placement", func(t *testing.T) {
		t.Parallel()
		const insufficient = "Order failed. Insufficient USDT balance in account"
		for _, tc := range []struct {
			name      string
			submit    *order.Submit
			operation string
			data      any
			expErr    error  // nil when expText names the rejected row's status message
			expText   string // the rejected row's status message
		}{
			{name: "rejected row with an order ID", submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ClientOrderID: "clientTwo"}, operation: "order", data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}}, expText: insufficient},
			{name: "row without a status code", submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ClientOrderID: "clientTwo"}, operation: "order", data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo"}}, expErr: errMissingStatusCode},
			{name: "row without an order ID", submit: &order.Submit{Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ClientOrderID: "clientTwo"}, operation: "order", data: []map[string]string{{"ordId": "", "clOrdId": "clientTwo", "sCode": "0", "sMsg": ""}}, expErr: common.ErrInvalidResponse},
			{name: "spread rejected row with an order ID", submit: &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2, ClientOrderID: "clientTwo"}, operation: "sprd-order", data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}}, expText: insufficient},
			{name: "spread row without a status code", submit: &order.Submit{Pair: spreadPair, AssetType: asset.Spread, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2, ClientOrderID: "clientTwo"}, operation: "sprd-order", data: []map[string]string{{"ordId": "1002", "clOrdId": "clientTwo"}}, expErr: errMissingStatusCode},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: websocketOperationReply("0", tc.data)}
				ex := newWebsocketTestExchange(t, srv, true, nil)
				ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}
				tc.submit.Exchange = ex.Name
				resp, err := ex.WebsocketSubmitOrder(t.Context(), tc.submit)
				if tc.expErr != nil {
					assert.ErrorIs(t, err, tc.expErr, "WebsocketSubmitOrder should return the sentinel for a row which confirms no placement")
				} else {
					assert.ErrorContains(t, err, tc.expText, "WebsocketSubmitOrder should report the rejected row's status message")
				}
				assert.Nil(t, resp, "WebsocketSubmitOrder should give no response for an order the reply does not confirm placed")
				assert.Len(t, srv.requests(tc.operation), 1, "one placement should be sent")
			})
		}
	})

	t.Run("keeps acting for its account when the exchange's credentials change mid-order", func(t *testing.T) {
		t.Parallel()
		var ex atomic.Pointer[Exchange]
		var mu sync.Mutex
		var keys, operations []string
		mock := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			keys = append(keys, r.URL.Path+" "+r.Header.Get("OK-ACCESS-KEY"))
			mu.Unlock()
			if r.URL.Path != "/account/config" {
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			ex.Load().SetCredentials(secondaryAccount())
			writeOKXData(t, w, []map[string]string{{"posMode": positionModeNet, "acctLv": "2"}})
		}, func(op string, _ json.RawMessage) (string, any) {
			mu.Lock()
			operations = append(operations, op)
			mu.Unlock()
			return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
		})
		ex.Store(mock)
		mock.instrumentIDCodeMap[perpetualSwapPair.String()] = 22222
		mock.SetCredentials(primaryAccount())
		loginMockWebsocket(t, mock)
		_, err := mock.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: mock.Name, Pair: perpetualSwapPair, AssetType: asset.PerpetualSwap, MarginType: margin.Multi, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100})
		require.NoError(t, err, "WebsocketSubmitOrder must keep acting for the account it started with")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/account/config primary-key"}, keys, "the position mode should be read for the connection's account")
		assert.Equal(t, []string{"order"}, operations, "the order should be sent over the websocket")
	})
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("refuses another account's context credentials", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		_, err := ex.WebsocketModifyOrder(otherAccountContext(t), &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Amount: 2, Price: 1})
		require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketModifyOrder must refuse an amendment for another account than the connections act for")
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("refuses the exchange's credentials once they change from the login", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		for _, ctx := range rotatedAccountContexts(t, ex) {
			for _, m := range []*order.Modify{
				{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Amount: 2, Price: 1},
				{Pair: spreadPair, AssetType: asset.Spread, OrderID: "1", Amount: 2, Price: 1},
			} {
				_, err := ex.WebsocketModifyOrder(ctx, m)
				require.ErrorIsf(t, err, errWebsocketOtherAccount, "WebsocketModifyOrder must refuse a %s amendment once the exchange's credentials differ from the connection's login", m.AssetType)
			}
		}
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("rejects an amend before it transmits", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketModifyOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "WebsocketModifyOrder must error for a nil modification")

		_, err = e.WebsocketModifyOrder(t.Context(), &order.Modify{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
			Amount:    1,
		})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")

		_, err = e.WebsocketModifyOrder(t.Context(), &order.Modify{
			Exchange:            e.Name,
			Pair:                mainPair,
			AssetType:           asset.Spot,
			OrderID:             "1234",
			Amount:              1,
			RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90}},
		})
		assert.ErrorIs(t, err, errAttachedOrderUnsupported, "a stop loss OKX's websocket amend operation cannot carry should fail before any request transmits")

		_, err = e.WebsocketModifyOrder(t.Context(), &order.Modify{
			Exchange:            e.Name,
			Pair:                spreadPair,
			AssetType:           asset.Spread,
			OrderID:             "1234",
			Price:               -2,
			RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: -1}},
		})
		assert.ErrorIs(t, err, errAttachedOrderUnsupported, "a spread order's take profit should fail before any request transmits")
	})

	t.Run("amends a spread order's size alone, leaving its price", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
			return "0", []map[string]string{{"clOrdId": "", "ordId": "1234", "reqId": "", "sCode": "0", "sMsg": ""}}
		}}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		_, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{Exchange: ex.Name, Pair: spreadPair, AssetType: asset.Spread, OrderID: "1234", Amount: 2})
		require.NoError(t, err, "WebsocketModifyOrder must amend a spread order's size")
		frames := srv.requests("sprd-amend-order")
		require.Len(t, frames, 1, "one spread amend request must be sent")
		assert.JSONEq(t, `[{"ordId":"1234","newSz":"2"}]`, string(frames[0].arguments), "a size-only spread amend should leave the price out")
	})

	t.Run("amends a spread order on the business connection", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
			return "0", []map[string]string{{"clOrdId": "", "ordId": "1234", "reqId": "", "sCode": "0", "sMsg": ""}}
		}}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		resp, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{
			Exchange:  ex.Name,
			Pair:      spreadPair,
			AssetType: asset.Spread,
			OrderID:   "1234",
			Amount:    1,
			Price:     -2,
		})
		require.NoError(t, err, "WebsocketModifyOrder must amend a spread order")
		assert.Equal(t, "1234", resp.OrderID, "WebsocketModifyOrder should return the amended order's ID")
		frames := srv.requests("sprd-amend-order")
		require.Len(t, frames, 1, "one spread amend request must be sent")
		assert.Equal(t, "/ws/v5/business", frames[0].path, "the spread amend should be sent on the business connection")
		assert.JSONEq(t, `[{"ordId":"1234","newSz":"1","newPx":"-2"}]`, string(frames[0].arguments), "the spread amend should carry its new size and price")
	})

	t.Run("responds with what the amendment sent", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			modify   *order.Modify
			expFrame string
			exp      *order.ModifyResponse
		}{
			{
				name:     "by order ID",
				modify:   &order.Modify{Pair: mainPair, AssetType: asset.Spot, Type: order.Limit, OrderID: "1301", TimeInForce: order.PostOnly, Price: 101, Amount: 1, TriggerPrice: 5},
				expFrame: `[{"instId":"BTC-USDT","instIdCode":12345,"ordId":"1301","newSz":"1","newPx":"101"}]`,
				exp:      &order.ModifyResponse{OrderID: "1301", Pair: mainPair, Price: 101, Amount: 1},
			},
			{
				name:     "by client order ID",
				modify:   &order.Modify{Pair: mainPair, AssetType: asset.Spot, Type: order.Limit, ClientOrderID: "client1301", Amount: 1},
				expFrame: `[{"instId":"BTC-USDT","instIdCode":12345,"clOrdId":"client1301","newSz":"1"}]`,
				exp:      &order.ModifyResponse{OrderID: "1301", ClientOrderID: "client1301", Pair: mainPair, Amount: 1},
			},
			{
				name:     "size naming a side",
				modify:   &order.Modify{Pair: mainPair, AssetType: asset.Spot, Type: order.Limit, Side: order.Sell, OrderID: "1301", Amount: 2},
				expFrame: `[{"instId":"BTC-USDT","instIdCode":12345,"ordId":"1301","newSz":"2"}]`,
				exp:      &order.ModifyResponse{OrderID: "1301", Pair: mainPair, Amount: 2},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var frames []string
				ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(op string, args json.RawMessage) (string, any) {
					assert.Equal(t, "amend-order", op, "the amendment should be sent by the amend-order operation")
					mu.Lock()
					frames = append(frames, string(args))
					mu.Unlock()
					return "0", []map[string]string{{"ordId": "1301", "clOrdId": "client1301", "sCode": "0", "sMsg": ""}}
				})
				tc.modify.Exchange = ex.Name
				tc.exp.Exchange = ex.Name
				resp, err := ex.WebsocketModifyOrder(t.Context(), tc.modify)
				require.NoError(t, err, "WebsocketModifyOrder must not error")
				assert.Equal(t, tc.exp, resp, "WebsocketModifyOrder should claim only the changes it sent")
				mu.Lock()
				defer mu.Unlock()
				require.Len(t, frames, 1, "one amendment frame must be sent")
				assert.JSONEq(t, tc.expFrame, frames[0], "the amendment frame should carry the documented fields")
			})
		}
		t.Run("spread by client order ID", func(t *testing.T) {
			t.Parallel()
			srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
				return "0", []map[string]string{{"clOrdId": "client1302", "ordId": "1302", "reqId": "", "sCode": "0", "sMsg": ""}}
			}}
			ex := newWebsocketTestExchange(t, srv, true, nil)
			resp, err := ex.WebsocketModifyOrder(t.Context(), &order.Modify{
				Exchange: ex.Name, Pair: spreadPair, AssetType: asset.Spread, ClientOrderID: "client1302", TimeInForce: order.PostOnly, Price: -2,
			})
			require.NoError(t, err, "WebsocketModifyOrder must amend a spread order")
			exp := &order.ModifyResponse{Exchange: ex.Name, OrderID: "1302", ClientOrderID: "client1302", Price: -2}
			assert.Equal(t, exp, resp, "WebsocketModifyOrder should name the amended order and claim only the changes it sent")
			frames := srv.requests("sprd-amend-order")
			require.Len(t, frames, 1, "one spread amend request must be sent")
			assert.JSONEq(t, `[{"clOrdId":"client1302","newPx":"-2"}]`, string(frames[0].arguments), "the spread amend should carry its client order ID and new price")
		})
	})

	t.Run("rejects a reply which confirms no amendment", func(t *testing.T) {
		t.Parallel()
		const failed = "Order modification failed as the order has been filled, canceled or does not exist."
		for _, tc := range []struct {
			name      string
			modify    *order.Modify
			operation string
			data      any
			expErr    error  // nil when expText names the failed row's status message
			expText   string // the failed row's status message
		}{
			{name: "null row", modify: &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1301", Amount: 2}, operation: "amend-order", data: []any{nil}, expErr: common.ErrInvalidResponse},
			{name: "empty row", modify: &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1301", Amount: 2}, operation: "amend-order", data: []map[string]string{{}}, expErr: errMissingStatusCode},
			{name: "row without a status code", modify: &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1301", Amount: 2}, operation: "amend-order", data: []map[string]string{{"ordId": "1301"}}, expErr: errMissingStatusCode},
			{name: "another order", modify: &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1301", Amount: 2}, operation: "amend-order", data: []map[string]string{{"ordId": "9999", "sCode": "0", "sMsg": ""}}, expErr: errReplyOrderMismatch},
			{name: "another client order ID", modify: &order.Modify{Pair: mainPair, AssetType: asset.Spot, ClientOrderID: "client1301", Amount: 2}, operation: "amend-order", data: []map[string]string{{"ordId": "9999", "clOrdId": "client9999", "sCode": "0", "sMsg": ""}}, expErr: errReplyOrderMismatch},
			{name: "failed row in a success reply", modify: &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1301", Amount: 2}, operation: "amend-order", data: []map[string]string{{"ordId": "1301", "sCode": "51503", "sMsg": failed}}, expText: failed},
			{name: "spread empty row", modify: &order.Modify{Pair: spreadPair, AssetType: asset.Spread, OrderID: "1302", Price: -2}, operation: "sprd-amend-order", data: []map[string]string{{}}, expErr: errMissingStatusCode},
			{name: "spread row without a status code", modify: &order.Modify{Pair: spreadPair, AssetType: asset.Spread, OrderID: "1302", Price: -2}, operation: "sprd-amend-order", data: []map[string]string{{"ordId": "1302"}}, expErr: errMissingStatusCode},
			{name: "spread another order", modify: &order.Modify{Pair: spreadPair, AssetType: asset.Spread, OrderID: "1302", Price: -2}, operation: "sprd-amend-order", data: []map[string]string{{"ordId": "9999", "sCode": "0", "sMsg": ""}}, expErr: errReplyOrderMismatch},
			{name: "spread another client order ID", modify: &order.Modify{Pair: spreadPair, AssetType: asset.Spread, ClientOrderID: "client1302", Price: -2}, operation: "sprd-amend-order", data: []map[string]string{{"ordId": "9999", "clOrdId": "client9999", "sCode": "0", "sMsg": ""}}, expErr: errReplyOrderMismatch},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: websocketOperationReply("0", tc.data)}
				ex := newWebsocketTestExchange(t, srv, true, nil)
				ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}
				tc.modify.Exchange = ex.Name
				resp, err := ex.WebsocketModifyOrder(t.Context(), tc.modify)
				if tc.expErr != nil {
					assert.ErrorIs(t, err, tc.expErr, "WebsocketModifyOrder should return the sentinel for a reply which confirms no amendment")
				} else {
					assert.ErrorContains(t, err, tc.expText, "WebsocketModifyOrder should report the failed amendment's status message")
				}
				assert.Nil(t, resp, "WebsocketModifyOrder should give no response for an amendment the reply does not confirm")
				assert.Len(t, srv.requests(tc.operation), 1, "one amendment should be sent")
			})
		}
	})
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("refuses another account's context credentials", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		err := ex.WebsocketCancelOrder(otherAccountContext(t), &order.Cancel{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"})
		require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketCancelOrder must refuse a cancel for another account than the connections act for")
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("refuses the exchange's credentials once they change from the login", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		for _, ctx := range rotatedAccountContexts(t, ex) {
			for _, c := range []*order.Cancel{
				{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"},
				{Pair: spreadPair, AssetType: asset.Spread, OrderID: "1"},
			} {
				err := ex.WebsocketCancelOrder(ctx, c)
				require.ErrorIsf(t, err, errWebsocketOtherAccount, "WebsocketCancelOrder must refuse a %s cancel once the exchange's credentials differ from the connection's login", c.AssetType)
			}
		}
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("rejects a cancel before it transmits", func(t *testing.T) {
		t.Parallel()
		err := e.WebsocketCancelOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "WebsocketCancelOrder must error for a nil cancellation")

		err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Type:      order.Trigger,
			OrderID:   "1234",
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo cancel should not transmit over the websocket")

		err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")
	})
	t.Run("cancels a spread order on the business connection", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			row    map[string]string
			expErr error
		}{
			{"cancelled", map[string]string{"clOrdId": "", "ordId": "1234", "sCode": "0", "sMsg": ""}, nil},
			{"reply without an order ID", map[string]string{"clOrdId": "", "ordId": "", "sCode": "0", "sMsg": ""}, common.ErrInvalidResponse},
			{"another order's acknowledgement", map[string]string{"clOrdId": "", "ordId": "9999", "sCode": "0", "sMsg": ""}, errReplyOrderMismatch},
			{"reply without a status code", map[string]string{"clOrdId": "", "ordId": "1234"}, errMissingStatusCode},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
					return "0", []map[string]string{tc.row}
				}}
				ex := newWebsocketTestExchange(t, srv, true, nil)
				err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{Exchange: ex.Name, Pair: spreadPair, AssetType: asset.Spread, OrderID: "1234"})
				require.ErrorIs(t, err, tc.expErr, "WebsocketCancelOrder must report the spread cancel's outcome")
				frames := srv.requests("sprd-cancel-order")
				require.Len(t, frames, 1, "one spread cancel request must be sent")
				assert.Equal(t, "/ws/v5/business", frames[0].path, "the spread cancel should be sent on the business connection")
				assert.JSONEq(t, `[{"ordId":"1234"}]`, string(frames[0].arguments), "the spread cancel should carry its order ID")
			})
		}
	})
	t.Run("checks the cancel's reply row", func(t *testing.T) {
		t.Parallel()
		const failed = "Order cancellation failed as the order has been filled, canceled or does not exist"
		for _, tc := range []struct {
			name    string
			data    any
			expErr  error  // nil with expText empty when the cancel succeeds
			expText string // the failed cancel's status message
		}{
			{name: "cancelled", data: []map[string]string{{"ordId": "1234", "clOrdId": "", "sCode": "0", "sMsg": ""}}},
			{name: "no rows", data: []map[string]string{}, expErr: common.ErrNoResponse},
			{name: "null row", data: []any{nil}, expErr: common.ErrInvalidResponse},
			{name: "empty row", data: []map[string]string{{}}, expErr: common.ErrInvalidResponse},
			{name: "row without an order ID", data: []map[string]string{{"ordId": "", "clOrdId": "C1", "sCode": "0", "sMsg": ""}}, expErr: common.ErrInvalidResponse},
			{name: "failed status in a success reply", data: []map[string]string{{"ordId": "1234", "sCode": "51400", "sMsg": failed}}, expText: failed},
			{name: "another order's acknowledgement", data: []map[string]string{{"ordId": "9999", "clOrdId": "", "sCode": "0", "sMsg": ""}}, expErr: errReplyOrderMismatch},
			{name: "row without a status code", data: []map[string]string{{"ordId": "1234", "clOrdId": ""}}, expErr: errMissingStatusCode},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var frames []string
				ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(op string, args json.RawMessage) (string, any) {
					assert.Equal(t, "cancel-order", op, "the cancel should be sent by the cancel order operation")
					mu.Lock()
					frames = append(frames, string(args))
					mu.Unlock()
					return "0", tc.data
				})
				err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Spot, OrderID: "1234"})
				switch {
				case tc.expErr != nil:
					assert.ErrorIs(t, err, tc.expErr, "WebsocketCancelOrder should return the sentinel for a reply that confirms no cancel")
				case tc.expText != "":
					assert.ErrorContains(t, err, tc.expText, "WebsocketCancelOrder should report the failed cancel's status message")
				default:
					assert.NoError(t, err, "WebsocketCancelOrder should report the acknowledged cancel")
				}
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, []string{`[{"instId":"BTC-USDT","instIdCode":12345,"ordId":"1234"}]`}, frames, "one cancel frame should be sent")
			})
		}
	})
	t.Run("matches a cancel by client order ID by its echo", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			cancel    *order.Cancel
			operation string
			row       map[string]string
			expErr    error
		}{
			{"echoed", &order.Cancel{Pair: mainPair, AssetType: asset.Spot, ClientOrderID: "C1"}, "cancel-order", map[string]string{"ordId": "1234", "clOrdId": "C1", "sCode": "0", "sMsg": ""}, nil},
			{"echoing none", &order.Cancel{Pair: mainPair, AssetType: asset.Spot, ClientOrderID: "C1"}, "cancel-order", map[string]string{"ordId": "1234", "clOrdId": "", "sCode": "0", "sMsg": ""}, errReplyOrderMismatch},
			{"echoing another", &order.Cancel{Pair: mainPair, AssetType: asset.Spot, ClientOrderID: "C1"}, "cancel-order", map[string]string{"ordId": "9999", "clOrdId": "C2", "sCode": "0", "sMsg": ""}, errReplyOrderMismatch},
			{"spread echoed", &order.Cancel{Pair: spreadPair, AssetType: asset.Spread, ClientOrderID: "S1"}, "sprd-cancel-order", map[string]string{"ordId": "1234", "clOrdId": "S1", "sCode": "0", "sMsg": ""}, nil},
			{"spread echoing none", &order.Cancel{Pair: spreadPair, AssetType: asset.Spread, ClientOrderID: "S1"}, "sprd-cancel-order", map[string]string{"ordId": "1234", "clOrdId": "", "sCode": "0", "sMsg": ""}, errReplyOrderMismatch},
			{"spread echoing another", &order.Cancel{Pair: spreadPair, AssetType: asset.Spread, ClientOrderID: "S1"}, "sprd-cancel-order", map[string]string{"ordId": "9999", "clOrdId": "S2", "sCode": "0", "sMsg": ""}, errReplyOrderMismatch},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: websocketOperationReply("0", []map[string]string{tc.row})}
				ex := newWebsocketTestExchange(t, srv, true, nil)
				ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}
				tc.cancel.Exchange = ex.Name
				assert.ErrorIs(t, ex.WebsocketCancelOrder(t.Context(), tc.cancel), tc.expErr, "WebsocketCancelOrder should confirm only the cancel its reply echoes")
				assert.Len(t, srv.requests(tc.operation), 1, "one cancel should be sent")
			})
		}
	})
	t.Run("reaches the websocket transport with a cached instrument code", func(t *testing.T) {
		t.Parallel()
		// The mock exchange below speaks REST only and fails the test on any
		// request, so a pass proves the explicit websocket path neither falls
		// back to REST nor stops at instrument code resolution.
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		ex.instrumentsInfoMap = make(map[string][]InstrumentResponse)
		ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		err := ex.WebsocketCancelOrder(t.Context(), &order.Cancel{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		})
		require.Error(t, err, "WebsocketCancelOrder must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")
	})
}

func TestCancelResultsUsable(t *testing.T) {
	t.Parallel()
	assert.True(t, cancelResultsUsable(nil), "a successful batch should report usable results")
	assert.True(t, cancelResultsUsable(errPartialSuccess), "a partial success should report usable results")
	assert.True(t, cancelResultsUsable(common.AppendError(errPartialSuccess, errors.New("order 3: does not exist"))), "a joined partial success should still report usable results")
	assert.False(t, cancelResultsUsable(errors.New("batch one failed")), "any other error should report unusable results")
}

func TestOrderMethodTransports(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var restPaths, wsOps []string
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		restPaths = append(restPaths, r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/account/config":
			writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
			return
		case "/trade/orders-pending":
			writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "1", "side": "buy", "ordType": orderLimit, "state": "live", "cTime": "1700000000000"}})
		case "/trade/orders-algo-pending":
			writeOKXData(t, w, []map[string]string{})
		default:
			writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
		}
	}, func(op string, _ json.RawMessage) (string, any) {
		mu.Lock()
		wsOps = append(wsOps, op)
		mu.Unlock()
		return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
	})
	sent := func() (rest, ws []string) {
		mu.Lock()
		defer mu.Unlock()
		rest, ws = restPaths, wsOps
		restPaths, wsOps = nil, nil
		return rest, ws
	}
	submit := &order.Submit{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1}
	modify := &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Amount: 2, Price: 1}
	cancel := &order.Cancel{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"}

	_, err := e.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "SubmitOrder must not error")
	_, err = e.ModifyOrder(t.Context(), modify)
	require.NoError(t, err, "ModifyOrder must not error")
	require.NoError(t, e.CancelOrder(t.Context(), cancel), "CancelOrder must not error")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{*cancel})
	require.NoError(t, err, "CancelBatchOrders must not error")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: mainPair, AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error")
	rest, ws := sent()
	assert.Equal(t, []string{
		"/account/config", "/trade/order", "/trade/amend-order", "/trade/cancel-order", "/trade/cancel-batch-orders", "/trade/orders-pending", "/trade/cancel-batch-orders",
		"/trade/orders-algo-pending", "/trade/orders-algo-pending", "/trade/orders-algo-pending", "/trade/orders-algo-pending", "/trade/orders-algo-pending", "/trade/orders-algo-pending",
	}, rest, "each standard order method should reach its REST endpoint")
	assert.Empty(t, ws, "no standard order method should send a websocket operation")

	_, err = e.WebsocketSubmitOrder(t.Context(), submit)
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	_, err = e.WebsocketModifyOrder(t.Context(), modify)
	require.NoError(t, err, "WebsocketModifyOrder must not error")
	require.NoError(t, e.WebsocketCancelOrder(t.Context(), cancel), "WebsocketCancelOrder must not error")
	rest, ws = sent()
	assert.Empty(t, rest, "no websocket order method should send a REST request")
	assert.Equal(t, []string{"order", "amend-order", "cancel-order"}, ws, "each websocket order method should send its websocket operation")
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("refuses another account's context credentials", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		_, err := ex.WebsocketSubmitOrders(otherAccountContext(t), []*order.Submit{{Exchange: ex.Name, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spot}})
		require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketSubmitOrders must refuse orders for another account than the connections act for")
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("refuses the exchange's credentials once they change from the login", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		for _, ctx := range rotatedAccountContexts(t, ex) {
			_, err := ex.WebsocketSubmitOrders(ctx, []*order.Submit{{Exchange: ex.Name, Pair: mainPair, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, AssetType: asset.Spot}})
			require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketSubmitOrders must refuse orders once the exchange's credentials differ from the connection's login")
		}
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("reports the replies", func(t *testing.T) {
		t.Parallel()
		first := map[string]string{"ordId": "1", "sCode": "0", "sMsg": ""}
		second := map[string]string{"ordId": "2", "sCode": "0", "sMsg": ""}
		failed := map[string]string{"ordId": "", "sCode": "51008", "sMsg": "Order failed. Insufficient USDT balance in account"}
		for _, tc := range []struct {
			name   string
			code   string
			data   []map[string]string
			exp    []string
			expErr error
		}{
			{"all placed", "0", []map[string]string{first, second}, []string{"1 BUY", "2 SELL"}, nil},
			{"partial success", "2", []map[string]string{first, failed}, []string{"1 BUY", ""}, errPartialSuccess},
			{"partial success with a rejected row carrying an order ID", "2", []map[string]string{first, {"ordId": "2", "sCode": "51008", "sMsg": "Order failed. Insufficient USDT balance in account"}}, []string{"1 BUY", ""}, errPartialSuccess},
			{"partial success with a successful row without an order ID", "2", []map[string]string{{"ordId": "", "sCode": "0", "sMsg": ""}, failed}, []string{"", ""}, common.ErrInvalidResponse},
			{"success with a row without a status code", "0", []map[string]string{first, {"ordId": "2"}}, []string{"1 BUY", ""}, errMissingStatusCode},
			{"success with a row echoing a client order ID", "0", []map[string]string{first, {"ordId": "2", "clOrdId": "clientX", "sCode": "0", "sMsg": ""}}, []string{"1 BUY", ""}, errReplyOrderMismatch},
			{"null row", "2", []map[string]string{nil, second}, []string{"", "2 SELL"}, errPartialSuccess},
			{"more rows than orders", "0", []map[string]string{first, second, second}, []string{}, common.ErrInvalidResponse},
			{"success without rows", "0", []map[string]string{}, []string{}, common.ErrInvalidResponse},
			{"success with a null row", "0", []map[string]string{nil, second}, []string{"", "2 SELL"}, common.ErrInvalidResponse},
			{"success with a row without an order ID", "0", []map[string]string{{"ordId": "", "sCode": "0", "sMsg": ""}, second}, []string{"", "2 SELL"}, common.ErrInvalidResponse},
			{"success without order IDs", "0", []map[string]string{nil, {"ordId": "", "sCode": "0", "sMsg": ""}}, []string{"", ""}, common.ErrInvalidResponse},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/account/config" {
						writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
						return
					}
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(_ string, args json.RawMessage) (string, any) {
					var reqs []PlaceOrderRequest
					if !assert.NoError(t, json.Unmarshal(args, &reqs), "the batch arguments should decode") {
						return "60013", nil
					}
					rows := make([]map[string]string, len(tc.data))
					for i, row := range tc.data {
						if row == nil {
							continue
						}
						rows[i] = maps.Clone(row)
						if _, echoes := row["clOrdId"]; !echoes && i < len(reqs) {
							rows[i]["clOrdId"] = reqs[i].ClientOrderID
						}
					}
					return tc.code, rows
				})
				resp, err := e.WebsocketSubmitOrders(t.Context(), []*order.Submit{
					{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1},
					{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 2},
				})
				if tc.expErr == nil {
					require.NoError(t, err, "WebsocketSubmitOrders must not error")
				} else {
					require.ErrorIs(t, err, tc.expErr, "WebsocketSubmitOrders must return the expected error")
				}
				got := make([]string, 0, len(resp))
				for _, r := range resp {
					if r == nil {
						got = append(got, "")
						continue
					}
					got = append(got, r.OrderID+" "+r.Side.String())
				}
				assert.Equal(t, tc.exp, got, "WebsocketSubmitOrders should report each placed order ID against its own order")
			})
		}
	})

	t.Run("correlates the replies by client order ID", func(t *testing.T) {
		t.Parallel()
		const insufficient = "Order failed. Insufficient USDT balance in account"
		buy := map[string]string{"ordId": "1001", "clOrdId": "clientOne", "sCode": "0", "sMsg": ""}
		sell := map[string]string{"ordId": "2002", "clOrdId": "clientTwo", "sCode": "0", "sMsg": ""}
		for _, tc := range []struct {
			name           string
			clientOrderIDs []string // of the buy and the sell
			code           string
			data           []map[string]string
			exp            []string
			expErr         error  // nil when expText names a failure without a sentinel
			expText        string // empty for success
		}{
			{name: "rows in order", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{buy, sell}, exp: []string{"1001 BUY", "2002 SELL"}},
			{name: "rows reversed", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{sell, buy}, exp: []string{"1001 BUY", "2002 SELL"}},
			{name: "a row echoed twice", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{buy, {"ordId": "2002", "clOrdId": "clientOne", "sCode": "0", "sMsg": ""}}, exp: []string{"", ""}, expErr: errReplyOrderMismatch},
			{name: "a row echoing no client order ID", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{buy, {"ordId": "2002", "sCode": "0", "sMsg": ""}}, exp: []string{"1001 BUY", ""}, expErr: errReplyOrderMismatch},
			{name: "a row echoing another client order ID", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{buy, {"ordId": "2002", "clOrdId": "clientX", "sCode": "0", "sMsg": ""}}, exp: []string{"1001 BUY", ""}, expErr: errReplyOrderMismatch},
			{name: "a null row", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "2", data: []map[string]string{nil, buy}, exp: []string{"1001 BUY", ""}, expErr: errReplyOrderMismatch},
			{name: "a rejected row carrying an order ID in a success reply", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{{"ordId": "2002", "clOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}, buy}, exp: []string{"1001 BUY", ""}, expText: insufficient},
			{name: "a rejected row carrying an order ID in a partial success", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "2", data: []map[string]string{{"ordId": "2002", "clOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}, buy}, exp: []string{"1001 BUY", ""}, expErr: errPartialSuccess},
			{name: "an order without a client order ID after its rows", clientOrderIDs: []string{"clientOne", ""}, code: "0", data: []map[string]string{buy, {"ordId": "3003", "sCode": "0", "sMsg": ""}}, exp: []string{"1001 BUY", ""}, expErr: errReplyOrderMismatch},
			{name: "an order without a client order ID given another's row", clientOrderIDs: []string{"clientOne", ""}, code: "0", data: []map[string]string{{"ordId": "3003", "sCode": "0", "sMsg": ""}, buy}, exp: []string{"1001 BUY", ""}, expErr: errReplyOrderMismatch},
			{name: "one order ID for both orders", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{{"ordId": "1001", "clOrdId": "clientTwo", "sCode": "0", "sMsg": ""}, buy}, exp: []string{"", ""}, expErr: errDuplicateReplyOrderID},
			{name: "a rejected row naming the accepted order's order ID", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "2", data: []map[string]string{{"ordId": "1001", "clOrdId": "clientTwo", "sCode": "51008", "sMsg": insufficient}, buy}, exp: []string{"1001 BUY", ""}, expErr: errPartialSuccess},
			{name: "an order ID a row echoing another client order ID names", clientOrderIDs: []string{"clientOne", "clientTwo"}, code: "0", data: []map[string]string{buy, {"ordId": "1001", "clOrdId": "clientX", "sCode": "0", "sMsg": ""}}, exp: []string{"", ""}, expErr: errDuplicateReplyOrderID},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var frames []string
				e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(_ string, args json.RawMessage) (string, any) {
					mu.Lock()
					frames = append(frames, string(args))
					mu.Unlock()
					return tc.code, tc.data
				})
				resp, err := e.WebsocketSubmitOrders(t.Context(), []*order.Submit{
					{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ClientOrderID: tc.clientOrderIDs[0]},
					{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 110, ClientOrderID: tc.clientOrderIDs[1]},
				})
				switch {
				case tc.expErr != nil:
					assert.ErrorIs(t, err, tc.expErr, "WebsocketSubmitOrders should return the expected error")
				case tc.expText != "":
					assert.ErrorContains(t, err, tc.expText, "WebsocketSubmitOrders should report the rejected row's status message")
				default:
					assert.NoError(t, err, "WebsocketSubmitOrders should accept rows it can pair with their orders")
				}
				got := make([]string, 0, len(resp))
				for _, r := range resp {
					if r == nil {
						got = append(got, "")
						continue
					}
					got = append(got, r.OrderID+" "+r.Side.String())
				}
				assert.Equal(t, tc.exp, got, "WebsocketSubmitOrders should report each placed order ID against its own order")
				mu.Lock()
				defer mu.Unlock()
				assert.Len(t, frames, 1, "one batch should be sent")
			})
		}
	})

	t.Run("reports no order an order ID another order's repeated rows name", func(t *testing.T) {
		t.Parallel()
		e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s", r.URL.Path)
			http.NotFound(w, r)
		}, func(string, json.RawMessage) (string, any) {
			return "0", []map[string]string{
				{"ordId": "2002", "clOrdId": "clientOne", "sCode": "0", "sMsg": ""},
				{"ordId": "2002", "clOrdId": "clientTwo", "sCode": "0", "sMsg": ""},
				{"ordId": "3003", "clOrdId": "clientTwo", "sCode": "0", "sMsg": ""},
			}
		})
		submit := func(clientOrderID string, price float64) *order.Submit {
			return &order.Submit{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: price, ClientOrderID: clientOrderID}
		}
		resp, err := e.WebsocketSubmitOrders(t.Context(), []*order.Submit{submit("clientOne", 100), submit("clientTwo", 101), submit("clientThree", 102)})
		require.ErrorIs(t, err, errDuplicateReplyOrderID, "WebsocketSubmitOrders must report an order ID the rows give two client order IDs")
		assert.ErrorContains(t, err, `order ID "2002"`, "the error should name the order ID")
		assert.Equal(t, []*order.SubmitResponse{nil, nil, nil}, resp, "WebsocketSubmitOrders should report no order the rows give another order's order ID")
	})

	t.Run("gives each order without a client order ID one of its own", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var sent []PlaceOrderRequest
		e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s", r.URL.Path)
			http.NotFound(w, r)
		}, func(_ string, args json.RawMessage) (string, any) {
			var reqs []PlaceOrderRequest
			if !assert.NoError(t, json.Unmarshal(args, &reqs), "the batch arguments should decode") {
				return "60013", nil
			}
			mu.Lock()
			sent = reqs
			mu.Unlock()
			rows := make([]map[string]string, 0, len(reqs))
			for i, req := range slices.Backward(reqs) {
				rows = append(rows, map[string]string{"ordId": strconv.Itoa(1001 + i), "clOrdId": req.ClientOrderID, "sCode": "0", "sMsg": ""})
			}
			return "0", rows
		})
		resp, err := e.WebsocketSubmitOrders(t.Context(), []*order.Submit{
			{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100},
			{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 110},
			{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 120, ClientOrderID: "clientThree"},
		})
		require.NoError(t, err, "WebsocketSubmitOrders must not error")
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, sent, 3, "every order must be sent")
		require.Len(t, resp, 3, "every order must have a response")
		assert.Equal(t, "clientThree", sent[2].ClientOrderID, "a client order ID given should be sent as it is")
		assert.NotEqual(t, sent[0].ClientOrderID, sent[1].ClientOrderID, "each order should be sent with a client order ID of its own")
		for i := range sent {
			assert.Regexpf(t, `^[0-9A-Za-z]{1,32}$`, sent[i].ClientOrderID, "order %d should be sent with a client order ID OKX accepts", i+1)
			require.NotNilf(t, resp[i], "order %d must be reported", i+1)
			assert.Equalf(t, strconv.Itoa(1001+i), resp[i].OrderID, "order %d should be reported with the order ID its own row names", i+1)
			assert.Equalf(t, sent[i].ClientOrderID, resp[i].ClientOrderID, "order %d should be reported with the client order ID it was sent with", i+1)
		}
	})

	t.Run("refuses a client order ID given twice before it transmits", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		_, err := ex.WebsocketSubmitOrders(t.Context(), []*order.Submit{
			{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 100, ClientOrderID: "clientOne"},
			{Exchange: ex.Name, Pair: mainPair, AssetType: asset.Spot, MarginType: margin.NoMargin, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 110, ClientOrderID: "clientOne"},
		})
		assert.ErrorIs(t, err, errDuplicateClientOrderID, "WebsocketSubmitOrders should refuse two orders sharing a client order ID, whose replies it could not tell apart")
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("rejects a batch before it transmits", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketSubmitOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrders must error for an empty batch")

		_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Trigger,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo order in the batch should stop the batch before transmission")

		_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
			Exchange:  e.Name,
			Pair:      spreadPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spread,
		}})
		assert.ErrorIs(t, err, errSpreadWebsocketBatchUnsupported, "a spread order should not join a websocket batch, which OKX cannot place spreads in")

		_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
			Exchange:    e.Name,
			Pair:        mainPair,
			Side:        order.Buy,
			Type:        order.Limit,
			TimeInForce: order.GoodTillTime,
			Amount:      1,
			Price:       1,
			AssetType:   asset.Spot,
			EndTime:     time.UnixMilli(1800000000000),
		}})
		assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "a good till time order should stop the batch before transmission")

		_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
			RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Enabled: true, Price: 2},
			},
		}})
		assert.ErrorIs(t, err, errAttachedOrderUnsupported, "a take profit OKX's websocket order operation cannot carry should stop the batch before transmission")
	})
	t.Run("sends nothing when one order in the batch is invalid", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/account/config" {
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
				return
			}
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		e.instrumentsInfoMap = make(map[string][]InstrumentResponse)
		e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := e.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Limit,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		}, {
			Exchange:  e.Name,
			Pair:      mainPair,
			Side:      order.Buy,
			Type:      order.Trigger,
			Amount:    1,
			Price:     1,
			AssetType: asset.Spot,
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an invalid second order should fail the whole batch before transmission")
	})
}

func TestWebsocketCancelBatchOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("refuses another account's context credentials", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		_, err := ex.WebsocketCancelBatchOrders(otherAccountContext(t), []order.Cancel{{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"}})
		require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketCancelBatchOrders must refuse cancels for another account than the connections act for")
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("refuses the exchange's credentials once they change from the login", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		for _, ctx := range rotatedAccountContexts(t, ex) {
			_, err := ex.WebsocketCancelBatchOrders(ctx, []order.Cancel{{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"}})
			require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketCancelBatchOrders must refuse cancels once the exchange's credentials differ from the connection's login")
		}
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("partial success", func(t *testing.T) {
		t.Parallel()
		e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s", r.URL.Path)
			http.NotFound(w, r)
		}, func(string, json.RawMessage) (string, any) {
			return "2", []map[string]string{
				{"ordId": "1", "sCode": "0", "sMsg": ""},
				{"ordId": "2", "sCode": "51400", "sMsg": "Order cancellation failed as the order has been filled, canceled or does not exist"},
			}
		})
		resp, err := e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{
			{Pair: mainPair, AssetType: asset.Spot, OrderID: "1"},
			{Pair: mainPair, AssetType: asset.Spot, OrderID: "2"},
		})
		require.ErrorIs(t, err, errPartialSuccess, "WebsocketCancelBatchOrders must return the partial success")
		require.NotNil(t, resp, "WebsocketCancelBatchOrders must return the results beside the error")
		assert.Equal(t, map[string]string{
			"1": order.PendingCancel.String(),
			"2": "Order cancellation failed as the order has been filled, canceled or does not exist",
		}, resp.Status, "each order should report its own outcome")
	})

	t.Run("rejects a cancel batch before it transmits", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketCancelBatchOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "WebsocketCancelBatchOrders must error for an empty batch")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), make([]order.Cancel, 21))
		require.ErrorIs(t, err, errExceedLimit, "WebsocketCancelBatchOrders must error for more than 20 orders")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      spreadPair,
			AssetType: asset.Spread,
			OrderID:   "1234",
		}})
		assert.ErrorIs(t, err, errSpreadBatchCancelUnsupported, "a spread cancel should not join a websocket batch, which OKX cannot cancel spreads in")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Type:      order.Trigger,
			OrderID:   "1234",
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo cancel should not transmit over the websocket")

		_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		}})
		assert.ErrorIs(t, err, errMissingInstrumentIDCode, "an uncached instrument code should fail before any request transmits")
	})
	t.Run("sends nothing when one cancel in the batch is invalid", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		e.instrumentsInfoMap = make(map[string][]InstrumentResponse)
		e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		}, {
			Exchange:  e.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			Type:      order.Trigger,
			OrderID:   "5678",
		}})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an invalid second order should fail the whole batch before transmission")
	})
	t.Run("reaches the websocket transport with a cached instrument code", func(t *testing.T) {
		t.Parallel()
		// The mock exchange below speaks REST only and fails the test on any
		// request, so a pass proves the explicit websocket path neither falls
		// back to REST nor stops at instrument code resolution.
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected REST request %s for an explicit websocket operation", r.URL.Path)
			http.NotFound(w, r)
		}))
		ex.instrumentsInfoMap = make(map[string][]InstrumentResponse)
		ex.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}

		_, err := ex.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{{
			Exchange:  ex.Name,
			Pair:      mainPair,
			AssetType: asset.Spot,
			OrderID:   "1234",
		}})
		require.Error(t, err, "WebsocketCancelBatchOrders must fail without a websocket connection")
		assert.NotErrorIs(t, err, errMissingInstrumentIDCode, "a cached instrument code should get past code resolution and fail at the connection")
	})

	t.Run("accounts for every requested order", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			data any
			exp  map[string]string
			err  error
		}{
			{name: "no rows", data: nil, exp: map[string]string{}, err: common.ErrInvalidResponse},
			{name: "a null row", data: []any{nil}, exp: map[string]string{}, err: common.ErrInvalidResponse},
			{name: "an empty row", data: []map[string]string{{}}, exp: map[string]string{}, err: common.ErrInvalidResponse},
			{name: "an empty list", data: []map[string]string{}, exp: map[string]string{}, err: common.ErrInvalidResponse},
			{name: "one order's row", data: []map[string]string{{"ordId": "1001", "sCode": "0"}}, exp: map[string]string{"1001": order.PendingCancel.String()}, err: common.ErrInvalidResponse},
			{name: "a row for another order", data: []map[string]string{{"ordId": "9999", "sCode": "0"}, {"clOrdId": "client2", "ordId": "1002", "sCode": "0"}}, exp: map[string]string{"1002": order.PendingCancel.String()}, err: common.ErrInvalidResponse},
			{name: "a row echoing no client order ID", data: []map[string]string{{"ordId": "1001", "sCode": "0"}, {"ordId": "1002", "sCode": "0"}}, exp: map[string]string{"1001": order.PendingCancel.String()}, err: common.ErrInvalidResponse},
			{name: "a client order ID's row without its order ID", data: []map[string]string{{"ordId": "1001", "sCode": "0"}, {"clOrdId": "client2", "sCode": "0"}}, exp: map[string]string{"1001": order.PendingCancel.String()}, err: common.ErrInvalidResponse},
			{name: "every order's row", data: []map[string]string{{"clOrdId": "client2", "ordId": "1002", "sCode": "0"}, {"ordId": "1001", "sCode": "0"}}, exp: map[string]string{"1001": order.PendingCancel.String(), "1002": order.PendingCancel.String()}},
			{name: "a row without a status code", data: []map[string]any{{"ordId": "1001", "sMsg": ""}, {"clOrdId": "client2", "ordId": "1002", "sCode": "0"}}, exp: map[string]string{"1002": order.PendingCancel.String()}, err: errMissingStatusCode},
			{name: "a row with a null status code", data: []map[string]any{{"ordId": "1001", "sCode": nil}, {"clOrdId": "client2", "ordId": "1002", "sCode": "0"}}, exp: map[string]string{"1002": order.PendingCancel.String()}, err: errMissingStatusCode},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected REST request %s", r.URL.Path)
					http.NotFound(w, r)
				}, func(string, json.RawMessage) (string, any) {
					return "0", tc.data
				})
				resp, err := ex.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{
					{Pair: mainPair, AssetType: asset.Spot, OrderID: "1001"},
					{Pair: mainPair, AssetType: asset.Spot, ClientOrderID: "client2"},
				})
				require.ErrorIs(t, err, tc.err, "WebsocketCancelBatchOrders must report every requested order the reply leaves unaccounted for")
				require.NotNil(t, resp, "WebsocketCancelBatchOrders must return the verified results")
				assert.Equal(t, tc.exp, resp.Status, "WebsocketCancelBatchOrders should report only the verified results, keyed by exchange order ID")
			})
		}
	})
}

func TestWebsocketCancelAllOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	t.Run("refuses another account's context credentials", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		_, err := ex.WebsocketCancelAllOrders(otherAccountContext(t), &order.Cancel{Pair: mainPair, AssetType: asset.Spot})
		require.ErrorIs(t, err, errWebsocketOtherAccount, "WebsocketCancelAllOrders must refuse cancels for another account than the connections act for")
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("refuses the exchange's credentials once they change from the login", func(t *testing.T) {
		t.Parallel()
		ex, operations := newOwnAccountWebsocketExchange(t)
		for _, ctx := range rotatedAccountContexts(t, ex) {
			for _, c := range []*order.Cancel{{Pair: mainPair, AssetType: asset.Spot}, {Pair: spreadPair, AssetType: asset.Spread}} {
				_, err := ex.WebsocketCancelAllOrders(ctx, c)
				require.ErrorIsf(t, err, errWebsocketOtherAccount, "WebsocketCancelAllOrders must refuse a %s cancel-all once the exchange's credentials differ from the connection's login", c.AssetType)
			}
		}
		assert.Empty(t, operations(), "no order operation should transmit")
	})

	t.Run("skips uncached instruments", func(t *testing.T) {
		t.Parallel()
		pending := make([]map[string]string, 0, 26)
		for i := range 25 {
			pending = append(pending, map[string]string{"instId": mainPair.String(), "ordId": strconv.Itoa(i), "side": "buy", "ordType": orderLimit, "state": "live", "cTime": "1700000000000"})
		}
		pending = append(pending, map[string]string{"instId": "XRP-USDT", "ordId": "UNCACHED-1", "side": "buy", "ordType": orderLimit, "state": "live", "cTime": "1700000000000"})
		var mu sync.Mutex
		var batches [][]CancelOrderRequest
		e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/orders-pending":
				writeOKXData(t, w, pending)
			case "/trade/orders-algo-pending":
				writeOKXData(t, w, []map[string]string{})
			default:
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}, func(op string, args json.RawMessage) (string, any) {
			var reqs []CancelOrderRequest
			if err := json.Unmarshal(args, &reqs); err != nil {
				t.Errorf("decoding %s arguments should not error: %v", op, err)
			}
			mu.Lock()
			batches = append(batches, reqs)
			mu.Unlock()
			rows := make([]map[string]string, 0, len(reqs))
			for x := range reqs {
				rows = append(rows, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0", "sMsg": ""})
			}
			return "0", rows
		})

		resp, err := e.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
		require.ErrorIs(t, err, errMissingInstrumentIDCode, "WebsocketCancelAllOrders must report the order it cannot address")
		assert.ErrorContains(t, err, "UNCACHED-1", "the error should name the order left open")
		assert.Len(t, resp.Status, 25, "every addressable order should be cancelled")
		assert.NotContains(t, resp.Status, "UNCACHED-1", "the unaddressable order should not be reported cancelled")

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, batches, 2, "25 orders must be cancelled in two batches")
		assert.Len(t, batches[0], 20, "the first batch should hold 20 orders")
		assert.Len(t, batches[1], 5, "the second batch should hold the other 5 orders")
		for _, b := range batches {
			for x := range b {
				assert.EqualValues(t, 12345, b[x].InstrumentIDCode, "each cancel should carry the cached instrument code")
			}
		}
	})

	t.Run("cancels spread orders on the business connection", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			pair      currency.Pair
			result    bool
			expArgs   string
			expStatus map[string]string
		}{
			{"one spread", spreadPair, true, `[{"sprdId":"BTC-USDT_BTC-USDT-SWAP"}]`, map[string]string{"BTC-USDT_BTC-USDT-SWAP": "true"}},
			{"every spread", currency.EMPTYPAIR, true, `[{}]`, map[string]string{"": "true"}},
			{"not cancelled", spreadPair, false, `[{"sprdId":"BTC-USDT_BTC-USDT-SWAP"}]`, map[string]string{"BTC-USDT_BTC-USDT-SWAP": "false"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
					return "0", []map[string]bool{{"result": tc.result}}
				}}
				ex := newWebsocketTestExchange(t, srv, true, nil)
				resp, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spread, Pair: tc.pair})
				require.NoError(t, err, "WebsocketCancelAllOrders must not error")
				assert.Equal(t, tc.expStatus, resp.Status, "WebsocketCancelAllOrders should key the result by the spread it scoped")
				frames := srv.requests("sprd-mass-cancel")
				require.Len(t, frames, 1, "one spread cancel-all request must be sent")
				assert.Equal(t, "/ws/v5/business", frames[0].path, "the spread cancel-all should be sent on the business connection")
				assert.JSONEq(t, tc.expArgs, string(frames[0].arguments), "the spread cancel-all should be scoped by the pair given")
			})
		}
	})

	t.Run("rejects a cancel-all before it transmits", func(t *testing.T) {
		t.Parallel()
		_, err := e.WebsocketCancelAllOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "WebsocketCancelAllOrders must error for a nil cancellation")

		_, err = e.WebsocketCancelAllOrders(t.Context(), &order.Cancel{
			AssetType: asset.Spot,
			Type:      order.Trigger,
		})
		assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "an algo cancel-all should not transmit over the websocket")
	})

	t.Run("refuses filters a spread mass cancel cannot apply", func(t *testing.T) {
		t.Parallel()
		srv := &websocketTestServer{reply: func(_, _ string, _ json.RawMessage) (string, any) {
			return "0", []map[string]bool{{"result": true}}
		}}
		ex := newWebsocketTestExchange(t, srv, true, nil)
		// The cases share the mock connection's recorded requests, so they run
		// sequentially.
		for _, tc := range []struct {
			name   string
			cancel *order.Cancel
		}{
			{"order ID alone", &order.Cancel{AssetType: asset.Spread, OrderID: "ONE"}},
			{"client order ID alone", &order.Cancel{AssetType: asset.Spread, ClientOrderID: "mine"}},
			{"pair and order ID", &order.Cancel{AssetType: asset.Spread, Pair: spreadPair, OrderID: "ONE"}},
			{"pair and side", &order.Cancel{AssetType: asset.Spread, Pair: spreadPair, Side: order.Sell}},
			{"pair and order type", &order.Cancel{AssetType: asset.Spread, Pair: spreadPair, Type: order.Limit}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				before := len(srv.requests("sprd-mass-cancel"))
				_, err := ex.WebsocketCancelAllOrders(t.Context(), tc.cancel)
				assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "a mass cancel scoped by a filter it cannot apply should be refused")
				assert.Len(t, srv.requests("sprd-mass-cancel"), before, "a refused cancel should send no mass cancel request")
			})
		}
	})

	t.Run("applies every filter to the pending orders", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			side order.Side
			exp  []string
		}{
			{"order ID of another side", order.Buy, nil},
			{"order ID and its own side", order.Sell, []string{"401"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var cancelled []string
				ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/trade/orders-pending":
						writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "401", "side": "sell", "ordType": orderLimit, "state": "live"}})
					case "/trade/orders-algo-pending":
						writeOKXData(t, w, []map[string]string{})
					default:
						t.Errorf("unexpected REST request %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}, func(op string, args json.RawMessage) (string, any) {
					var reqs []CancelOrderRequest
					if err := json.Unmarshal(args, &reqs); err != nil {
						t.Errorf("decoding %s arguments should not error: %v", op, err)
					}
					rows := make([]map[string]string, 0, len(reqs))
					mu.Lock()
					for x := range reqs {
						cancelled = append(cancelled, reqs[x].OrderID)
						rows = append(rows, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0", "sMsg": ""})
					}
					mu.Unlock()
					return "0", rows
				})
				_, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, OrderID: "401", Side: tc.side})
				require.NoError(t, err, "WebsocketCancelAllOrders must not error")
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, tc.exp, cancelled, "only an order every supplied filter selects should be cancelled")
			})
		}
	})

	t.Run("names the algo orders it cannot cancel", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			cancel   order.Cancel
			listings []string
			err      error
		}{
			{
				name: "every order of a pair", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair}, err: errWebsocketAlgoCancelUnsupported,
				listings: []string{
					"/trade/orders-pending instId=BTC-USDT&instType=SPOT",
					"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=conditional%2Coco",
					"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=trigger",
					"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=move_order_stop",
					"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=twap",
					"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=iceberg",
					"/trade/orders-algo-pending instId=BTC-USDT&instType=SPOT&ordType=smart_iceberg",
				},
			},
			{name: "an ordinary order type", cancel: order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit}, listings: []string{"/trade/orders-pending instId=BTC-USDT&instType=SPOT&ordType=limit%2Cpost_only%2Cfok%2Cioc%2Cop_fok%2Crpi"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				book := &pendingBook{
					ordinary: []map[string]string{{"instType": "SPOT", "instId": "BTC-USDT", "ordId": "1001", "ordType": orderLimit, "side": "buy"}},
					algo: []map[string]string{
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2001", "ordType": orderTrigger, "side": "buy"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2002", "ordType": orderConditional, "side": "sell"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2003", "ordType": "iceberg", "side": "buy"},
						{"instType": "SPOT", "instId": "BTC-USDT", "algoId": "2004", "ordType": orderSmartIceberg, "side": "sell"},
					},
				}
				ex := newMockWebsocketExchange(t, book.rest(t), book.websocket(t))
				resp, err := ex.WebsocketCancelAllOrders(t.Context(), &tc.cancel)
				require.ErrorIs(t, err, tc.err, "WebsocketCancelAllOrders must report the algo orders it leaves open")
				if tc.err != nil {
					assert.ErrorContains(t, err, "2002, 2001, 2003, 2004", "the error should name every algo order left open")
				}
				assert.Equal(t, map[string]string{"1001": order.PendingCancel.String()}, resp.Status, "the ordinary orders should still be cancelled")
				listings, cancelled, algoBatches := book.sent()
				assert.Equal(t, []string{"1001"}, cancelled, "the ordinary orders should be cancelled over the websocket")
				assert.Empty(t, algoBatches, "no algo order should be cancelled over REST")
				assert.Equal(t, tc.listings, listings, "WebsocketCancelAllOrders should list the orders the cancellation selects")
			})
		}
	})

	t.Run("leaves orders listed for another instrument type", func(t *testing.T) {
		t.Parallel()
		book := &pendingBook{}
		ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/trade/orders-pending" {
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			writeOKXData(t, w, []map[string]string{
				{"instType": instTypeSpot, "instId": mainPair.String(), "ordId": "1001", "ordType": orderLimit, "side": "buy"},
				{"instType": instTypeMargin, "instId": mainPair.String(), "ordId": "9002", "ordType": orderLimit, "side": "buy"},
			})
		}, book.websocket(t))
		resp, err := ex.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit})
		require.ErrorIs(t, err, errListedOtherInstrumentType, "WebsocketCancelAllOrders must report an order of another instrument type than it listed")
		assert.ErrorContains(t, err, `order "9002" is MARGIN`, "the error should name the order left alone")
		assert.Equal(t, map[string]string{"1001": order.PendingCancel.String()}, resp.Status, "WebsocketCancelAllOrders should cancel the requested instrument type's orders alone")
		_, cancelled, _ := book.sent()
		assert.Equal(t, []string{"1001"}, cancelled, "the other instrument type's order should not be cancelled")
	})

	t.Run("keeps acting for its account when the exchange's credentials change mid-cancel", func(t *testing.T) {
		t.Parallel()
		var ex atomic.Pointer[Exchange]
		var mu sync.Mutex
		var keys []string
		var operations []string
		mock := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			keys = append(keys, r.URL.Path+" "+r.Header.Get("OK-ACCESS-KEY"))
			mu.Unlock()
			if r.URL.Path != "/trade/orders-pending" {
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			ex.Load().SetCredentials(secondaryAccount())
			writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "1001", "ordType": orderLimit, "side": "buy"}})
		}, func(op string, _ json.RawMessage) (string, any) {
			mu.Lock()
			operations = append(operations, op)
			mu.Unlock()
			return "0", []map[string]string{{"ordId": "1001", "sCode": "0"}}
		})
		ex.Store(mock)
		mock.SetCredentials(primaryAccount())
		loginMockWebsocket(t, mock)
		resp, err := mock.WebsocketCancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: mainPair, Type: order.Limit})
		require.NoError(t, err, "WebsocketCancelAllOrders must keep acting for the account it started with")
		assert.Equal(t, map[string]string{"1001": order.PendingCancel.String()}, resp.Status, "the order should be cancelled")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/trade/orders-pending primary-key"}, keys, "the listing should be read for the connection's account")
		assert.Equal(t, []string{"batch-cancel-orders"}, operations, "the cancel should be sent over the websocket")
	})

	t.Run("a dead context stops before the algo listing", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var mu sync.Mutex
		var paths []string
		ex := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			if r.URL.Path != "/trade/orders-pending" {
				t.Errorf("unexpected REST request %s", r.URL.Path)
				http.NotFound(w, r)
				return
			}
			cancel()
			writeOKXData(t, w, []map[string]string{{"instId": mainPair.String(), "ordId": "1001", "ordType": orderLimit, "side": "buy"}})
		}, func(op string, _ json.RawMessage) (string, any) {
			t.Errorf("unexpected websocket operation %s", op)
			return "60013", nil
		})
		_, err := ex.WebsocketCancelAllOrders(ctx, &order.Cancel{AssetType: asset.Spot, Pair: mainPair})
		require.ErrorIs(t, err, context.Canceled, "WebsocketCancelAllOrders must report the dead context")
		assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()), "WebsocketCancelAllOrders should report the context error once")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"/trade/orders-pending"}, paths, "no algo listing should be requested once the context is dead")
	})
}

func TestOrderAmounts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                                     string
		row                                      map[string]string
		amount, executed, remaining, quoteAmount float64
	}{
		{
			"quote sized, filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderMarket, "side": "sell", "cTime": "1694153250532", "sz": "10", "tgtCcy": "quote_ccy", "accFillSz": "0.00038128", "avgPx": "26228.1", "state": "filled"},
			0.00038128, 0.00038128, 0, 10,
		},
		{
			"quote sized, partially filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderMarket, "side": "buy", "cTime": "1694153250532", "sz": "100", "tgtCcy": "quote_ccy", "accFillSz": "0.001", "avgPx": "25000", "state": "partially_filled"},
			0.004, 0.001, 0.003, 100,
		},
		{
			"quote sized, live",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderLimit, "side": "buy", "cTime": "1694153250532", "sz": "100", "tgtCcy": "quote_ccy", "accFillSz": "0", "avgPx": "", "state": "live"},
			0, 0, 0, 100,
		},
		{
			"quote sized, partially filled, zero avgPx",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderLimit, "side": "buy", "cTime": "1694153250532", "sz": "100", "tgtCcy": "quote_ccy", "accFillSz": "0", "avgPx": "", "state": "partially_filled"},
			0, 0, 0, 100,
		},
		{
			"base sized, partially filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderLimit, "side": "buy", "cTime": "1694153250532", "sz": "1", "tgtCcy": "base_ccy", "accFillSz": "0.5", "avgPx": "25000", "state": "partially_filled"},
			1, 0.5, 0.5, 0,
		},
		{
			"base sized, filled",
			map[string]string{"instId": mainPair.String(), "ordId": "1", "ordType": orderMarket, "side": "sell", "cTime": "1694153250532", "sz": "1", "tgtCcy": "base_ccy", "accFillSz": "1", "avgPx": "25000", "state": "filled"},
			1, 1, 0, 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeOKXData(t, w, []map[string]string{tc.row})
			}))
			detail, err := e.GetOrderInfo(t.Context(), "1", mainPair, asset.Spot)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, tc.amount, detail.Amount, "Amount should be in the base currency")
			assert.Equal(t, tc.executed, detail.ExecutedAmount, "ExecutedAmount should match accFillSz")
			assert.Equal(t, tc.remaining, detail.RemainingAmount, "RemainingAmount should be the unfilled base amount")
			assert.Equal(t, tc.quoteAmount, detail.QuoteAmount, "QuoteAmount should carry the quote-currency size when set")
		})
	}
}

func TestLimitMakerOrdersAmendAndCancelByType(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var restPaths, wsOps []string
	var submitBody []byte
	e := newMockWebsocketExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		restPaths = append(restPaths, r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/account/config":
			writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
			return
		case "/trade/order":
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the place order body should not error")
			mu.Lock()
			submitBody = b
			mu.Unlock()
		}
		writeOKXData(t, w, []map[string]string{{"ordId": "1", "sCode": "0"}})
	}, func(op string, _ json.RawMessage) (string, any) {
		mu.Lock()
		wsOps = append(wsOps, op)
		mu.Unlock()
		return "0", []map[string]string{{"ordId": "1", "sCode": "0"}}
	})
	sent := func() (rest, ws []string) {
		mu.Lock()
		defer mu.Unlock()
		rest, ws = restPaths, wsOps
		restPaths, wsOps = nil, nil
		return rest, ws
	}
	submit := &order.Submit{Exchange: e.Name, Pair: mainPair, AssetType: asset.Spot, Side: order.Buy, Type: order.LimitMaker, Amount: 1, Price: 1}
	modify := &order.Modify{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Type: order.LimitMaker, Amount: 2, Price: 1}
	cancel := &order.Cancel{Pair: mainPair, AssetType: asset.Spot, OrderID: "1", Type: order.LimitMaker}

	_, err := e.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "SubmitOrder must not error")
	_, err = e.ModifyOrder(t.Context(), modify)
	require.NoError(t, err, "ModifyOrder must accept a LimitMaker type")
	require.NoError(t, e.CancelOrder(t.Context(), cancel), "CancelOrder must accept a LimitMaker type")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{*cancel})
	require.NoError(t, err, "CancelBatchOrders must accept a LimitMaker type")
	rest, ws := sent()
	assert.Equal(t, []string{"/account/config", "/trade/order", "/trade/amend-order", "/trade/cancel-order", "/trade/cancel-batch-orders"}, rest, "each standard order method should reach its REST endpoint, after the account level a spot order's trade mode follows")
	assert.Empty(t, ws, "no standard order method should send a websocket operation")
	mu.Lock()
	var placed map[string]any
	require.NoError(t, json.Unmarshal(submitBody, &placed), "the place order request body must decode")
	mu.Unlock()
	assert.Equal(t, orderPostOnly, placed["ordType"], "a LimitMaker submit should place a post_only order")

	_, err = e.WebsocketModifyOrder(t.Context(), modify)
	require.NoError(t, err, "WebsocketModifyOrder must accept a LimitMaker type")
	require.NoError(t, e.WebsocketCancelOrder(t.Context(), cancel), "WebsocketCancelOrder must accept a LimitMaker type")
	_, err = e.WebsocketCancelBatchOrders(t.Context(), []order.Cancel{*cancel})
	require.NoError(t, err, "WebsocketCancelBatchOrders must accept a LimitMaker type")
	rest, ws = sent()
	assert.Empty(t, rest, "no websocket order method should send a REST request")
	assert.Equal(t, []string{"amend-order", "cancel-order", "batch-cancel-orders"}, ws, "each websocket order method should send its websocket operation")
}

func TestSpreadOrdersUseRESTWithAuthenticatedWebsocket(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var wsOps []string
	var restPaths []string
	var spreadIDs []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
				// The connections outlive the test, and OKX's keepalive ping is not JSON
				if string(msg) == "ping" {
					return nil
				}
				var req struct {
					ID        string `json:"id"`
					Operation string `json:"op"`
				}
				if err := json.Unmarshal(msg, &req); err != nil {
					return err
				}
				mu.Lock()
				wsOps = append(wsOps, req.Operation)
				mu.Unlock()
				// The error live OKX returns when a spread operation reaches the
				// private websocket connection.
				return c.WriteMessage(gws.TextMessage,
					[]byte(`{"id":"`+req.ID+`","op":"`+req.Operation+`","code":"60028","msg":"The current operation is not supported by this URL. Please use the correct WebSocket URL for the operation."}`))
			})
			return
		}
		mu.Lock()
		restPaths = append(restPaths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == "/sprd/order" {
			var body struct {
				SpreadID string `json:"sprdId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decoding spread order request: %v", err)
			}
			mu.Lock()
			spreadIDs = append(spreadIDs, body.SpreadID)
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"sCode":"0","sMsg":"","ordId":"SPRD-1","clOrdId":""}]}`))
	})

	e := testexch.MockWebsocketInstance[Exchange](t, handler)
	// MockWebsocketInstance points RestSpotURL at the server without a trailing
	// slash, which breaks the REST path join, so every endpoint is re-pointed
	// at the mock server; the websocket connections are already established.
	mockURL, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "the mock server URL must be resolvable")
	b := e.GetBase()
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, mockURL+"/"), "Setup must point endpoint %s at the mock server", k)
	}
	// A connected websocket with authenticated endpoints available is what the
	// broken code branches on before sending spread operations to the private
	// connection.
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the mock websocket must report availability for the wrapper")

	sub, err := e.SubmitOrder(t.Context(), &order.Submit{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		Side:      order.Buy,
		Type:      order.Limit,
		Amount:    1,
		Price:     100,
	})
	require.NoError(t, err, "SubmitOrder must place a spread order over REST")
	require.NotEmpty(t, sub.OrderID, "SubmitOrder must return the spread order ID")

	_, err = e.ModifyOrder(t.Context(), &order.Modify{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		OrderID:   "SPRD-1",
		Amount:    2,
		Price:     101,
	})
	require.NoError(t, err, "ModifyOrder must amend a spread order over REST")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		AssetType: asset.Spread,
		Pair:      spreadPair,
		OrderID:   "SPRD-1",
	})
	require.NoError(t, err, "CancelOrder must cancel a spread order over REST")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"/sprd/order", "/sprd/amend-order", "/sprd/cancel-order"}, restPaths,
		"spread submit, amend and cancel should reach their REST endpoints")
	assert.Equal(t, []string{"BTC-USDT_BTC-USDT-SWAP"}, spreadIDs,
		"SubmitOrder should send the spread ID OKX lists")
	assert.Empty(t, wsOps, "no spread operation should be sent over the websocket")
}

func TestPendingOrderPagersStopOnARepeatedPage(t *testing.T) {
	t.Parallel()
	page := make([]map[string]string, 0, orderListPageSize)
	algoPage := make([]map[string]string, 0, orderListPageSize)
	for i := range orderListPageSize {
		page = append(page, map[string]string{"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": fmt.Sprintf("ORD-%03d", i), "ordType": orderLimit, "side": "buy", "state": "live", "sz": "1", "px": "1", "cTime": "1700000000000"})
		algoPage = append(algoPage, map[string]string{"instType": instTypeSpot, "instId": mainPair.String(), "algoId": fmt.Sprintf("ALGO-%03d", i), "ordType": orderTrigger, "side": "buy", "state": "live", "sz": "1", "cTime": "1700000000000"})
	}
	var mu sync.Mutex
	listings := make(map[string]int)
	var cancelled, cancelledAlgos []string
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-pending", "/sprd/orders-pending", "/trade/orders-algo-pending":
			mu.Lock()
			listings[r.URL.Path]++
			mu.Unlock()
			if r.URL.Path == "/trade/orders-algo-pending" {
				writeOKXData(t, w, algoPage)
				return
			}
			writeOKXData(t, w, page)
		case "/trade/cancel-algos":
			var reqs []CancelAlgoOrderRequest
			if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
				t.Errorf("decoding algo cancel request body should not error: %v", err)
				return
			}
			rows := make([]map[string]string, 0, len(reqs))
			mu.Lock()
			for x := range reqs {
				cancelledAlgos = append(cancelledAlgos, reqs[x].AlgoID)
				rows = append(rows, map[string]string{"algoId": reqs[x].AlgoID, "sCode": "0"})
			}
			mu.Unlock()
			writeOKXData(t, w, rows)
		case "/trade/cancel-batch-orders":
			var reqs []CancelOrderRequest
			if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
				t.Errorf("decoding cancel request body should not error: %v", err)
				return
			}
			rows := make([]map[string]string, 0, len(reqs))
			mu.Lock()
			for x := range reqs {
				cancelled = append(cancelled, reqs[x].OrderID)
				rows = append(rows, map[string]string{"ordId": reqs[x].OrderID, "sCode": "0"})
			}
			mu.Unlock()
			writeOKXData(t, w, rows)
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	// A pager that never stops fails at this deadline instead of hanging.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	for _, tc := range []struct {
		a      asset.Item
		orders int
		err    error // the algo order listings report their repeated page
	}{
		{a: asset.Spot, orders: 2 * orderListPageSize, err: errListingStalled},
		{a: asset.Spread, orders: orderListPageSize},
	} {
		active, err := e.GetActiveOrders(ctx, &order.MultiOrderRequest{AssetType: tc.a, Type: order.AnyType, Side: order.AnySide})
		require.ErrorIsf(t, err, tc.err, "GetActiveOrders must stop at a repeated page for %s", tc.a)
		assert.Lenf(t, active, tc.orders, "GetActiveOrders should keep each repeated page once for %s", tc.a)
	}
	_, err := e.CancelAllOrders(ctx, &order.Cancel{AssetType: asset.Spot})
	require.ErrorIs(t, err, errListingStalled, "CancelAllOrders must stop at a repeated page and report the listing incomplete")
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, map[string]int{"/trade/orders-pending": 4, "/sprd/orders-pending": 2, "/trade/orders-algo-pending": 20}, listings, "each pager should stop at the first repeated page")
	assert.Len(t, cancelled, orderListPageSize, "cancel-all should cancel each order once")
	assert.Len(t, cancelledAlgos, orderListPageSize, "cancel-all should cancel each algo order once")
}

func TestListOrdersByLimitMakerType(t *testing.T) {
	t.Parallel()
	row := func(id, ordType, state string) map[string]string {
		return map[string]string{"instId": mainPair.String(), "sprdId": spreadPair.String(), "ordId": id, "ordType": ordType, "side": "buy", "state": state, "sz": "1", "px": "1", "cTime": "1700000000000"}
	}
	e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := "canceled"
		switch r.URL.Path {
		case "/trade/orders-pending", "/sprd/orders-pending":
			state = "live"
		case "/trade/orders-history-archive", "/trade/orders-history", "/sprd/orders-history", "/sprd/orders-history-archive":
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		writeOKXData(t, w, []map[string]string{row("LIMIT-1", orderLimit, state), row("POST-1", orderPostOnly, state), row("IOC-1", orderIOC, state)})
	}))
	for _, a := range []asset.Item{asset.Spot, asset.Spread} {
		req := &order.MultiOrderRequest{AssetType: a, Type: order.LimitMaker, Side: order.AnySide}
		active, err := e.GetActiveOrders(t.Context(), req)
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
		history, err := e.GetOrderHistory(t.Context(), req)
		require.NoErrorf(t, err, "GetOrderHistory must not error for %s", a)
		for _, got := range []order.FilteredOrders{active, history} {
			require.Lenf(t, got, 1, "a LimitMaker query must return the post-only order for %s", a)
			assert.Equalf(t, "POST-1", got[0].OrderID, "the post-only order should be returned for %s", a)
			assert.Equalf(t, order.LimitMaker, got[0].Type, "the order should report the requested type for %s", a)
		}
	}
}

func TestOrderCost(t *testing.T) {
	t.Parallel()
	o := &OrderDetailResponse{AveragePrice: 41000, AccumulatedFillSize: 1.5}
	for _, tc := range []struct {
		name               string
		assetType          asset.Item
		pair               currency.Pair
		tradeQuoteCurrency currency.Code
		cost               float64
		costAsset          currency.Code
	}{
		{"spot", asset.Spot, currency.NewBTCUSDT(), currency.EMPTYCODE, 61500, currency.USDT},
		{"margin", asset.Margin, currency.NewBTCUSDT(), currency.EMPTYCODE, 61500, currency.USDT},
		{"futures", asset.Futures, currency.NewBTCUSDT(), currency.EMPTYCODE, 0, currency.EMPTYCODE},
		{"perpetual swap", asset.PerpetualSwap, currency.NewBTCUSDT(), currency.EMPTYCODE, 0, currency.EMPTYCODE},
		{"options", asset.Options, currency.NewBTCUSDT(), currency.EMPTYCODE, 0, currency.EMPTYCODE},
		{"spot traded in another quote currency", asset.Spot, currency.NewBTCUSD(), currency.USDC, 61500, currency.USDC},
		{"margin traded in another quote currency", asset.Margin, currency.NewBTCUSD(), currency.USDC, 61500, currency.USDC},
		{"spot without a trade quote currency", asset.Spot, currency.NewBTCUSD(), currency.EMPTYCODE, 61500, currency.USD},
		{"futures with a trade quote currency", asset.Futures, currency.NewBTCUSD(), currency.USDC, 0, currency.EMPTYCODE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cost, costAsset := orderCost(tc.assetType, o.AveragePrice, o.AccumulatedFillSize, tc.pair, tc.tradeQuoteCurrency)
			assert.Equal(t, tc.cost, cost, "orderCost should report the expected cost")
			assert.Equal(t, tc.costAsset, costAsset, "orderCost should report the expected cost asset")
		})
	}
}

// makerRebateOrderRow is a spot maker buy of 2 BTC at 100 USDT, filled to accFillSz, which OKX charged no fee and
// paid a 0.5 USDT maker rebate
func makerRebateOrderRow(state, accFillSz string) map[string]string {
	return map[string]string{
		"instType": instTypeSpot, "instId": mainPair.String(), "ordId": "ORD-1", "clOrdId": "client-1", "ordType": orderLimit, "side": "buy",
		"state": state, "sz": "2", "accFillSz": accFillSz, "avgPx": "100", "px": "100", "fee": "0", "feeCcy": "USDT",
		"rebate": "0.5", "rebateCcy": "USDT", "tgtCcy": "", "cTime": "1700000000000", "uTime": "1700000001000",
	}
}

// statedOrderCase is an ordinary order part filled in the state it is listed in, as OKX states it, and the details
// every ordinary order reader reports for it
type statedOrderCase struct {
	name string
	a    asset.Item
	row  map[string]string
	exp  order.Detail
}

// statedOrderCases returns orders whose sizes, trade currency and reduce only flag OKX states beside their instrument,
// in state, which reads back as status: a margin market buy and a spot market buy sent without tgtCcy are sized in the
// quote currency, a BTC-USD order can trade against USDC, and a perpetual swap sell can only reduce a position
func statedOrderCases(state string, status order.Status) []statedOrderCase {
	btcUSD := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	row := func(instType, instID, ordID, ordType, side, sz string, fields map[string]string) map[string]string {
		r := map[string]string{
			"instType": instType, "instId": instID, "ordId": ordID, "clOrdId": "client" + ordID, "ordType": ordType, "side": side, "sz": sz,
			"accFillSz": "0.001", "avgPx": "50000", "state": state, "tgtCcy": "", "reduceOnly": "false", "cTime": "1795409000000", "uTime": "1795409001000",
		}
		maps.Copy(r, fields)
		return r
	}
	detail := func(a asset.Item, pair currency.Pair, ordID string, oType order.Type, side order.Side, edit func(*order.Detail)) order.Detail {
		d := order.Detail{
			AverageExecutedPrice: 50000, ExecutedAmount: 0.001, OrderID: ordID, ClientOrderID: "client" + ordID, Type: oType, Side: side,
			Status: status, AssetType: a, Pair: pair, Date: time.UnixMilli(1795409000000), LastUpdated: time.UnixMilli(1795409001000),
		}
		edit(&d)
		return d
	}
	return []statedOrderCase{
		{
			name: "margin market buy sized in the quote currency", a: asset.Margin,
			row: row(instTypeMargin, mainPair.String(), "11", orderMarket, "buy", "100", map[string]string{"tradeQuoteCcy": "USDT"}),
			exp: detail(asset.Margin, mainPair, "11", order.Market, order.Buy, func(d *order.Detail) {
				d.Amount, d.RemainingAmount, d.QuoteAmount, d.Cost, d.CostAsset = 0.002, 0.001, 100, 50, currency.USDT
			}),
		},
		{
			name: "margin market sell sized in the base currency", a: asset.Margin,
			row: row(instTypeMargin, mainPair.String(), "12", orderMarket, "sell", "0.002", nil),
			exp: detail(asset.Margin, mainPair, "12", order.Market, order.Sell, func(d *order.Detail) {
				d.Amount, d.RemainingAmount, d.Cost, d.CostAsset = 0.002, 0.001, 50, currency.USDT
			}),
		},
		{
			name: "spot market buy sent without a target currency", a: asset.Spot,
			row: row(instTypeSpot, mainPair.String(), "13", orderMarket, "buy", "100", nil),
			exp: detail(asset.Spot, mainPair, "13", order.Market, order.Buy, func(d *order.Detail) {
				d.Amount, d.RemainingAmount, d.QuoteAmount, d.Cost, d.CostAsset = 0.002, 0.001, 100, 50, currency.USDT
			}),
		},
		{
			name: "spot order traded against another quote currency", a: asset.Spot,
			row: row(instTypeSpot, btcUSD.String(), "14", orderLimit, "buy", "0.002", map[string]string{"px": "50000", "tradeQuoteCcy": "USDC"}),
			exp: detail(asset.Spot, btcUSD, "14", order.Limit, order.Buy, func(d *order.Detail) {
				d.Price, d.Amount, d.RemainingAmount, d.Cost, d.CostAsset = 50000, 0.002, 0.001, 50, currency.USDC
			}),
		},
		{
			name: "reduce only perpetual swap sell", a: asset.PerpetualSwap,
			row: row(instTypeSwap, perpetualSwapPair.String(), "15", orderLimit, "sell", "2", map[string]string{"px": "50000", "posSide": "net", "accFillSz": "1", "reduceOnly": "true"}),
			exp: detail(asset.PerpetualSwap, perpetualSwapPair, "15", order.Limit, order.Sell, func(d *order.Detail) {
				d.Price, d.Amount, d.ExecutedAmount, d.RemainingAmount, d.ReduceOnly = 50000, 2, 1, 1, true
			}),
		},
	}
}

// pagerCase is a listing a pager reads, as the pages its fake venue answers each cursor with, and the order IDs the
// pager must return
type pagerCase struct {
	name  string
	pages map[string][]map[string]string
	exp   []string
	// incomplete is set where the pages stop advancing before a last page, which the cancel-all pagers report
	incomplete bool
}

// pagerRows returns the rows row builds for the orders numbered first up to, but not including, end
func pagerRows(row func(int) map[string]string, first, end int) []map[string]string {
	r := make([]map[string]string, 0, end-first)
	for i := first; i < end; i++ {
		r = append(r, row(i))
	}
	return r
}

// pagerIDs returns the order IDs of the first count orders
func pagerIDs(count int) []string {
	r := make([]string, 0, count)
	for i := range count {
		r = append(r, fmt.Sprintf("ORD-%03d", i))
	}
	return r
}

// pagerCases returns the page boundaries a newest-first order ID pager must read through, its rows built by row:
// OKX's cursors are exclusive, but a page that ends at its own cursor still carries unseen orders before it
func pagerCases(row func(int) map[string]string) []pagerCase {
	rows := func(first, end int) []map[string]string { return pagerRows(row, first, end) }
	lastOfFirst := fmt.Sprintf("ORD-%03d", orderListPageSize-1)
	return []pagerCase{
		{
			name:  "single order on the final page",
			pages: map[string][]map[string]string{"": rows(0, orderListPageSize), lastOfFirst: rows(orderListPageSize, orderListPageSize+1)},
			exp:   pagerIDs(orderListPageSize + 1),
		},
		{
			name:  "full final page",
			pages: map[string][]map[string]string{"": rows(0, orderListPageSize), lastOfFirst: {}},
			exp:   pagerIDs(orderListPageSize),
		},
		{
			name: "page ending at its cursor",
			pages: map[string][]map[string]string{
				"":          rows(0, orderListPageSize),
				lastOfFirst: append(rows(orderListPageSize, 2*orderListPageSize-1), row(orderListPageSize-1)),
			},
			exp:        pagerIDs(2*orderListPageSize - 1),
			incomplete: true,
		},
		{
			name:  "no orders",
			pages: map[string][]map[string]string{"": {}},
			exp:   []string{},
		},
	}
}

// liveOrderIDs returns the live order IDs from head down, at most a page of them, as OKX lists orders newest first
func liveOrderIDs(live map[string]bool, head int) []string {
	ids := make([]string, 0, orderListPageSize)
	for i := head; i > 0 && len(ids) < orderListPageSize; i-- {
		if id := strconv.Itoa(i); live[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

// stalledPagerCases returns pages which stop advancing before a last page, its rows built by row: a full or short page
// holding only orders read already, and a full or short page whose cursor leads back to one requested before
func stalledPagerCases(row func(int) map[string]string) []pagerCase {
	firstPage := pagerRows(row, 0, orderListPageSize)
	readAgain := slices.Clone(firstPage)
	slices.Reverse(readAgain)
	lastOfFirst, lastOfSecond := fmt.Sprintf("ORD-%03d", orderListPageSize-1), fmt.Sprintf("ORD-%03d", 2*orderListPageSize-1)
	return []pagerCase{
		{
			name:       "a full page repeated",
			pages:      map[string][]map[string]string{"": firstPage, lastOfFirst: firstPage},
			exp:        pagerIDs(orderListPageSize),
			incomplete: true,
		},
		{
			name:       "a full page of orders read already",
			pages:      map[string][]map[string]string{"": firstPage, lastOfFirst: readAgain},
			exp:        pagerIDs(orderListPageSize),
			incomplete: true,
		},
		{
			name: "a page leading back to an earlier cursor",
			pages: map[string][]map[string]string{
				"":           firstPage,
				lastOfFirst:  pagerRows(row, orderListPageSize, 2*orderListPageSize),
				lastOfSecond: append(pagerRows(row, 2*orderListPageSize, 3*orderListPageSize-1), row(orderListPageSize-1)),
			},
			exp:        pagerIDs(3*orderListPageSize - 1),
			incomplete: true,
		},
		{
			name:       "a short page of its own cursor",
			pages:      map[string][]map[string]string{"": firstPage, lastOfFirst: {row(orderListPageSize - 1)}},
			exp:        pagerIDs(orderListPageSize),
			incomplete: true,
		},
		{
			name: "a short page leading back to an earlier cursor",
			pages: map[string][]map[string]string{
				"":           firstPage,
				lastOfFirst:  pagerRows(row, orderListPageSize, 2*orderListPageSize),
				lastOfSecond: {row(2 * orderListPageSize), row(orderListPageSize - 1)},
			},
			exp:        pagerIDs(2*orderListPageSize + 1),
			incomplete: true,
		},
		{
			name:       "a short page of orders read already",
			pages:      map[string][]map[string]string{"": firstPage, lastOfFirst: {row(orderListPageSize / 2)}},
			exp:        pagerIDs(orderListPageSize),
			incomplete: true,
		},
	}
}

// cyclingTriggerPages returns perpetual swap trigger order pages in state, keyed by the after cursor requesting each,
// whose cursors cycle: the newest page, orders 200 down to 101, leads to the oldest, 100 down to 1, which leads back to
// the newest. Orders 200 and 199 place an immediate or cancel and a fill or kill order once triggered.
func cyclingTriggerPages(state string) map[string][]map[string]string {
	trigger := func(id int, advanceOrdType string) map[string]string {
		return map[string]string{
			"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "algoId": strconv.Itoa(id), "ordType": orderTrigger, "advanceOrdType": advanceOrdType,
			"side": "buy", "sz": "1", "state": state, "triggerPx": "110", "ordPx": "100", "cTime": "1795409000000",
		}
	}
	page := func(head int) []map[string]string {
		rows := make([]map[string]string, 0, orderListPageSize)
		for id := head; id > head-orderListPageSize; id-- {
			rows = append(rows, trigger(id, ""))
		}
		return rows
	}
	newest := page(2 * orderListPageSize)
	newest[0], newest[1] = trigger(2*orderListPageSize, orderIOC), trigger(2*orderListPageSize-1, orderFOK)
	return map[string][]map[string]string{"": newest, strconv.Itoa(orderListPageSize + 1): page(orderListPageSize), "1": newest}
}

// cyclingListingOrdinaryOrders returns the perpetual swap ordinary orders in state the ordType filter selects beside
// cyclingTriggerPages: immediate or cancel order 800 and fill or kill order 801
func cyclingListingOrdinaryOrders(state, ordTypes string) []map[string]string {
	rows := make([]map[string]string, 0, 2)
	for _, o := range []struct{ id, ordType string }{{"800", orderIOC}, {"801", orderFOK}} {
		if ordTypes == "" || slices.Contains(strings.Split(ordTypes, ","), o.ordType) {
			rows = append(rows, map[string]string{
				"instType": instTypeSwap, "instId": perpetualSwapPair.String(), "ordId": o.id, "ordType": o.ordType, "side": "sell", "sz": "1", "px": "100",
				"state": state, "cTime": "1795409000000", "uTime": "1795409000000",
			})
		}
	}
	return rows
}

// pagerHandler answers path with the page its cursor query parameter names, and fails a cursor requested twice or
// with no page, so a pager that stops advancing fails rather than hangs
func pagerHandler(t *testing.T, path, cursor string, pages map[string][]map[string]string) http.HandlerFunc {
	t.Helper()
	var mu sync.Mutex
	requested := make(map[string]bool)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		at := r.URL.Query().Get(cursor)
		mu.Lock()
		repeated := requested[at]
		requested[at] = true
		mu.Unlock()
		page, ok := pages[at]
		if repeated || !ok {
			t.Errorf("%s %q should be requested once and only for a listed page", cursor, at)
			http.Error(w, "unexpected cursor", http.StatusBadRequest)
			return
		}
		writeOKXData(t, w, page)
	}
}

// withoutAlgoOrders answers the algo order listings, which a listing of any order type reads beside the ordinary
// ones, with no orders, and passes every other request to next
func withoutAlgoOrders(t *testing.T, next http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/orders-algo-pending", "/trade/orders-algo-history":
			writeOKXData(t, w, []any{})
		default:
			next(w, r)
		}
	}
}

func TestOrderFee(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              string
		fee, rebate       types.Number
		feeCcy, rebateCcy currency.Code
		expFee            float64
		expFeeAsset       currency.Code
	}{
		{name: "charged fee", fee: -0.25, feeCcy: currency.USDT, expFee: 0.25, expFeeAsset: currency.USDT},
		{name: "net rebate in the fee", fee: 0.25, feeCcy: currency.USDT, expFee: -0.25, expFeeAsset: currency.USDT},
		{name: "no fee", feeCcy: currency.USDT, rebateCcy: currency.BTC, expFeeAsset: currency.USDT},
		{name: "maker rebate in the fee currency", feeCcy: currency.USDT, rebate: 0.5, rebateCcy: currency.USDT, expFee: -0.5, expFeeAsset: currency.USDT},
		{name: "fee and rebate in one currency", fee: -0.25, feeCcy: currency.USDT, rebate: 0.5, rebateCcy: currency.USDT, expFee: -0.25, expFeeAsset: currency.USDT},
		{name: "maker rebate without a fee", feeCcy: currency.BTC, rebate: 0.5, rebateCcy: currency.USDT, expFee: -0.5, expFeeAsset: currency.USDT},
		{name: "fee and rebate in different currencies", fee: -0.25, feeCcy: currency.BTC, rebate: 0.5, rebateCcy: currency.USDT, expFee: 0.25, expFeeAsset: currency.BTC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fee, feeAsset := orderFee(tc.fee, tc.feeCcy, tc.rebate, tc.rebateCcy)
			assert.Equal(t, tc.expFee, fee, "orderFee should report the expected fee")
			assert.Equal(t, tc.expFeeAsset, feeAsset, "orderFee should report the expected fee asset")
		})
	}
}

func TestQuoteSized(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		a              asset.Item
		ordType        string
		side           order.Side
		targetCurrency string
		exp            bool
	}{
		{"spot market buy in the quote currency", asset.Spot, orderMarket, order.Buy, targetCurrencyQuote, true},
		{"spot market buy in the base currency", asset.Spot, orderMarket, order.Buy, targetCurrencyBase, false},
		{"spot market buy without a target currency", asset.Spot, orderMarket, order.Buy, "", true},
		{"spot market sell without a target currency", asset.Spot, orderMarket, order.Sell, "", false},
		{"spot market sell in the quote currency", asset.Spot, orderMarket, order.Sell, targetCurrencyQuote, true},
		{"margin market buy", asset.Margin, orderMarket, order.Buy, "", true},
		{"margin market buy stated in upper case", asset.Margin, "MARKET", order.Buy, "", true},
		{"margin market sell", asset.Margin, orderMarket, order.Sell, "", false},
		{"margin limit buy", asset.Margin, orderLimit, order.Buy, "", false},
		{"margin immediate or cancel buy", asset.Margin, orderIOC, order.Buy, "", false},
		{"spot conditional buy", asset.Spot, orderConditional, order.Buy, "", false},
		{"spot conditional buy in the quote currency", asset.Spot, orderConditional, order.Buy, targetCurrencyQuote, true},
		{"perpetual swap market buy", asset.PerpetualSwap, orderMarket, order.Buy, "", false},
		{"futures market buy", asset.Futures, orderMarket, order.Buy, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, quoteSized(tc.a, tc.ordType, tc.side, tc.targetCurrency), "quoteSized should report whether OKX states the size in the quote currency")
		})
	}
}

func TestAlgoQuoteSized(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		a    asset.Item
		algo *AlgoOrderResponse
		exp  bool
	}{
		{"spot trailing stop buy", asset.Spot, &AlgoOrderResponse{OrderType: orderMoveOrderStop, Side: order.Buy}, true},
		{"margin trailing stop buy", asset.Margin, &AlgoOrderResponse{OrderType: orderMoveOrderStop, Side: order.Buy}, true},
		{"spot trailing stop sell", asset.Spot, &AlgoOrderResponse{OrderType: orderMoveOrderStop, Side: order.Sell}, false},
		{"trailing stop buy stated in the base currency", asset.Spot, &AlgoOrderResponse{OrderType: orderMoveOrderStop, Side: order.Buy, TargetCurrency: targetCurrencyBase}, false},
		{"spot trigger buy at market", asset.Spot, &AlgoOrderResponse{OrderType: orderTrigger, Side: order.Buy, OrderPrice: -1}, true},
		{"margin trigger buy at market", asset.Margin, &AlgoOrderResponse{OrderType: orderTrigger, Side: order.Buy, OrderPrice: -1}, true},
		{"spot trigger buy at a limit price", asset.Spot, &AlgoOrderResponse{OrderType: orderTrigger, Side: order.Buy, OrderPrice: 100}, false},
		{"spot trigger sell at market", asset.Spot, &AlgoOrderResponse{OrderType: orderTrigger, Side: order.Sell, OrderPrice: -1}, false},
		{"spot conditional buy at market", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, StopLossTriggerPrice: 110, StopLossOrderPrice: -1}, true},
		{"spot conditional buy at market stated in the base currency", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, StopLossTriggerPrice: 110, StopLossOrderPrice: -1, TargetCurrency: targetCurrencyBase}, false},
		{"margin conditional buy at a limit price", asset.Margin, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, StopLossTriggerPrice: 110, StopLossOrderPrice: 111}, false},
		{"take profit only conditional buy at market", asset.Margin, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: -1}, true},
		{"take profit only conditional buy at a limit price", asset.Margin, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: 91}, false},
		{"conditional buy with its stop loss at market and its ignored take profit at a limit price", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: 89, StopLossTriggerPrice: 110, StopLossOrderPrice: -1}, true},
		{"conditional buy with its stop loss at a limit price and its ignored take profit at market", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: -1, StopLossTriggerPrice: 110, StopLossOrderPrice: 111}, false},
		{"conditional buy with both legs at market", asset.Margin, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: -1, StopLossTriggerPrice: 110, StopLossOrderPrice: -1}, true},
		{"conditional buy whose take profit at market triggered beside a stop loss at a limit price", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, ActualSide: "tp", TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: -1, StopLossTriggerPrice: 110, StopLossOrderPrice: 111}, true},
		{"conditional buy whose take profit at a limit price triggered beside a stop loss at market", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, ActualSide: "tp", TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: 89, StopLossTriggerPrice: 110, StopLossOrderPrice: -1}, false},
		{"conditional buy whose stop loss at market triggered beside a take profit at a limit price", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, ActualSide: "sl", TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: 89, StopLossTriggerPrice: 110, StopLossOrderPrice: -1}, true},
		{"conditional buy without legs", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy}, false},
		{"conditional buy whose stop loss sets only its market order price", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Buy, StopLossOrderPrice: -1}, false},
		{"conditional sell with its stop loss at market", asset.Spot, &AlgoOrderResponse{OrderType: orderConditional, Side: order.Sell, TakeProfitTriggerPrice: 110, TakeProfitOrderPrice: 111, StopLossTriggerPrice: 90, StopLossOrderPrice: -1}, false},
		{"oco buy with both legs at market", asset.Spot, &AlgoOrderResponse{OrderType: orderOCO, Side: order.Buy, TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: -1, StopLossTriggerPrice: 110, StopLossOrderPrice: -1}, true},
		{"oco buy with one leg at a limit price", asset.Spot, &AlgoOrderResponse{OrderType: orderOCO, Side: order.Buy, TakeProfitTriggerPrice: 90, TakeProfitOrderPrice: 89, StopLossTriggerPrice: 110, StopLossOrderPrice: -1}, false},
		{"oco buy without legs", asset.Spot, &AlgoOrderResponse{OrderType: orderOCO, Side: order.Buy}, false},
		{"perpetual swap trigger buy at market", asset.PerpetualSwap, &AlgoOrderResponse{OrderType: orderTrigger, Side: order.Buy, OrderPrice: -1}, false},
		{"twap buy", asset.Spot, &AlgoOrderResponse{OrderType: orderTWAP, Side: order.Buy}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, algoQuoteSized(tc.a, tc.algo), "algoQuoteSized should report whether OKX reads the algo order's size in the quote currency")
		})
	}
}

func TestConditionalLeg(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                      string
		algo                      *AlgoOrderResponse
		expPrice, expTriggerPrice float64
	}{
		{"stop loss alone", &AlgoOrderResponse{StopLossOrderPrice: -1, StopLossTriggerPrice: 110}, -1, 110},
		{"take profit alone", &AlgoOrderResponse{TakeProfitOrderPrice: 89, TakeProfitTriggerPrice: 90}, 89, 90},
		{"both legs", &AlgoOrderResponse{StopLossOrderPrice: 111, StopLossTriggerPrice: 110, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, 111, 110},
		{"a stop loss with its trigger price alone", &AlgoOrderResponse{StopLossTriggerPrice: 110, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, 0, 110},
		{"a stop loss with its order price alone", &AlgoOrderResponse{StopLossOrderPrice: 111, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, 111, 0},
		{"no legs", &AlgoOrderResponse{}, 0, 0},
		{"the take profit OKX states triggered", &AlgoOrderResponse{ActualSide: "tp", StopLossOrderPrice: 111, StopLossTriggerPrice: 110, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, -1, 90},
		{"the stop loss OKX states triggered", &AlgoOrderResponse{ActualSide: "sl", StopLossOrderPrice: 111, StopLossTriggerPrice: 110, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, 111, 110},
		{"the take profit OKX states triggered beside a stop loss with its order price alone", &AlgoOrderResponse{ActualSide: "tp", StopLossOrderPrice: 111, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, -1, 90},
		{"the trigger side in upper case", &AlgoOrderResponse{ActualSide: "TP", StopLossOrderPrice: 111, StopLossTriggerPrice: 110, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, -1, 90},
		{"a trigger side OKX does not document", &AlgoOrderResponse{ActualSide: "both", StopLossOrderPrice: 111, StopLossTriggerPrice: 110, TakeProfitOrderPrice: -1, TakeProfitTriggerPrice: 90}, 111, 110},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			price, triggerPrice := conditionalLeg(tc.algo)
			assert.Equal(t, tc.expPrice, price, "conditionalLeg should return the order price of the leg OKX executes")
			assert.Equal(t, tc.expTriggerPrice, triggerPrice, "conditionalLeg should return the trigger price of the leg OKX executes")
		})
	}
}

func TestReplyAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		instType  string
		requested asset.Item
		exp       asset.Item
		err       error
	}{
		{name: "spot reply to a spot lookup", instType: instTypeSpot, requested: asset.Spot, exp: asset.Spot},
		{name: "margin reply to a spot lookup", instType: instTypeMargin, requested: asset.Spot, exp: asset.Margin},
		{name: "spot reply to a margin lookup", instType: instTypeSpot, requested: asset.Margin, exp: asset.Spot},
		{name: "margin reply stated in lower case", instType: "margin", requested: asset.Spot, exp: asset.Margin},
		{name: "swap reply to a swap lookup", instType: instTypeSwap, requested: asset.PerpetualSwap, exp: asset.PerpetualSwap},
		{name: "no instrument type", requested: asset.PerpetualSwap, exp: asset.PerpetualSwap},
		{name: "swap reply to a spot lookup", instType: instTypeSwap, requested: asset.Spot, err: order.ErrOrderNotFound},
		{name: "spot reply to a swap lookup", instType: instTypeSpot, requested: asset.PerpetualSwap, err: order.ErrOrderNotFound},
		{name: "futures reply to a swap lookup", instType: instTypeFutures, requested: asset.PerpetualSwap, err: order.ErrOrderNotFound},
		{name: "unknown instrument type", instType: "EVENTS", requested: asset.Spot, err: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, err := replyAsset(tc.instType, tc.requested)
			require.ErrorIs(t, err, tc.err, "replyAsset must return the expected error")
			assert.Equal(t, tc.exp, a, "replyAsset should report the asset of the order the reply describes")
		})
	}
}

func TestSpreadRemainingSize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name            string
		pendingFillSize OptionalNumber
		size, executed  float64
		status          order.Status
		exp             float64
	}{
		{name: "pending fill size", pendingFillSize: OptionalNumber{Value: new(types.Number(2))}, size: 3, status: order.Active, exp: 2},
		{name: "explicit zero pending fill size", pendingFillSize: OptionalNumber{Value: new(types.Number(0))}, size: 3, executed: 1, status: order.PartiallyFilled},
		{name: "pending fill size of a filled order", pendingFillSize: OptionalNumber{Value: new(types.Number(0))}, size: 3, executed: 3, status: order.Filled},
		{name: "no pending fill size", size: 3, executed: 1, status: order.PartiallyFilled, exp: 2},
		{name: "no pending fill size on a filled order", size: 3, executed: 3, status: order.Filled},
		{name: "no pending fill size on an order filled before its fills read so", size: 3, executed: 1, status: order.Filled},
		{name: "no pending fill size and fills above the size", size: 3, executed: 4, status: order.Active},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, spreadRemainingSize(tc.pendingFillSize, tc.size, tc.executed, tc.status), "spreadRemainingSize should report the size still to fill")
		})
	}
}

func TestListingNarrowsOrderTypes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		orderType order.Type
		tif       order.TimeInForce
		exp       bool
	}{
		{"any order type", order.AnyType, order.UnknownTIF, false},
		{"any order type with a time in force", order.AnyType, order.PostOnly, true},
		{"an order type", order.Limit, order.UnknownTIF, true},
		{"an order type with a time in force", order.Limit, order.PostOnly, true},
		{"no order type", order.UnknownType, order.UnknownTIF, false},
		{"no order type with a time in force", order.UnknownType, order.PostOnly, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, listingNarrowsOrderTypes(&order.MultiOrderRequest{Type: tc.orderType, TimeInForce: tc.tif}), "listingNarrowsOrderTypes should report whether the request narrows the order types it lists")
		})
	}
}

func TestCreatedBeforeStart(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1795410000100).Add(500 * time.Microsecond)
	for _, tc := range []struct {
		name           string
		created, start time.Time
		exp            bool
	}{
		{"created a millisecond before the start's", time.UnixMilli(1795410000099), start, true},
		{"created within the start's millisecond", time.UnixMilli(1795410000100), start, false},
		{"created after the start", time.UnixMilli(1795410000101), start, false},
		{"no start", time.UnixMilli(1795410000099), time.Time{}, false},
		{"created at the epoch with a start within its millisecond", time.UnixMilli(0), time.Unix(0, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, createdBeforeStart(tc.created, tc.start), "createdBeforeStart should compare the creation time with the start's millisecond")
		})
	}
}

func TestCreatedAfterEnd(t *testing.T) {
	t.Parallel()
	end := time.UnixMilli(1795410000200).Add(500 * time.Microsecond)
	for _, tc := range []struct {
		name         string
		created, end time.Time
		exp          bool
	}{
		{"created after the end", time.UnixMilli(1795410000201), end, true},
		{"created within the end's millisecond", time.UnixMilli(1795410000200), end, false},
		{"created at the end", time.UnixMilli(1795410000200), time.UnixMilli(1795410000200), false},
		{"created before the end", time.UnixMilli(1795410000199), end, false},
		{"no end", time.UnixMilli(1795410000201), time.Time{}, false},
		{"an end just past the epoch", time.UnixMilli(1), time.Unix(0, 1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, createdAfterEnd(tc.created, tc.end), "createdAfterEnd should report a creation time after a set end")
		})
	}
}

func TestAlgoListingOrderTypes(t *testing.T) {
	t.Parallel()
	every := []string{orderTrigger, "conditional,oco", orderMoveOrderStop, orderTWAP}
	for _, tc := range []struct {
		name      string
		a         asset.Item
		orderType order.Type
		tif       order.TimeInForce
		exp       []string
		err       error
	}{
		{name: "any order type on spot", a: asset.Spot, orderType: order.AnyType, exp: every},
		{name: "any order type on margin", a: asset.Margin, orderType: order.AnyType, exp: every},
		{name: "any order type on futures", a: asset.Futures, orderType: order.AnyType, exp: append(slices.Clone(every), orderChase)},
		{name: "any order type on perpetual swap", a: asset.PerpetualSwap, orderType: order.AnyType, exp: append(slices.Clone(every), orderChase)},
		{name: "any order type on options", a: asset.Options, orderType: order.AnyType},
		{name: "any order type with a time in force", a: asset.Spot, orderType: order.AnyType, tif: order.PostOnly},
		{name: "trigger orders", a: asset.Spot, orderType: order.Trigger, exp: []string{orderTrigger}},
		{name: "conditional stop orders", a: asset.Margin, orderType: order.ConditionalStop, exp: []string{orderConditional}},
		{name: "one cancels the other orders", a: asset.Spot, orderType: order.OCO, exp: []string{orderOCO}},
		{name: "trailing stop orders", a: asset.Futures, orderType: order.TrailingStop, exp: []string{orderMoveOrderStop}},
		{name: "time weighted orders", a: asset.Spot, orderType: order.TWAP, exp: []string{orderTWAP}},
		{name: "chase orders on perpetual swap", a: asset.PerpetualSwap, orderType: order.Chase, exp: []string{orderChase}},
		{name: "chase orders on spot", a: asset.Spot, orderType: order.Chase, err: order.ErrUnsupportedOrderType},
		{name: "trigger orders on options", a: asset.Options, orderType: order.Trigger, err: order.ErrUnsupportedOrderType},
		{name: "trigger orders with a time in force", a: asset.Spot, orderType: order.Trigger, tif: order.ImmediateOrCancel, exp: []string{orderTrigger}},
		{name: "fill or kill trigger orders", a: asset.PerpetualSwap, orderType: order.Trigger, tif: order.FillOrKill, exp: []string{orderTrigger}},
		{name: "post only trigger orders", a: asset.Spot, orderType: order.Trigger, tif: order.PostOnly, err: order.ErrUnsupportedTimeInForce},
		{name: "good till cancel trigger orders", a: asset.Futures, orderType: order.Trigger, tif: order.GoodTillCancel, err: order.ErrUnsupportedTimeInForce},
		{name: "immediate or cancel trigger orders on options", a: asset.Options, orderType: order.Trigger, tif: order.ImmediateOrCancel, err: order.ErrUnsupportedOrderType},
		{name: "conditional stop orders with a time in force", a: asset.Spot, orderType: order.ConditionalStop, tif: order.ImmediateOrCancel, err: order.ErrUnsupportedTimeInForce},
		{name: "any order type immediate or cancel", a: asset.Spot, orderType: order.AnyType, tif: order.ImmediateOrCancel, exp: []string{orderTrigger}},
		{name: "any order type fill or kill", a: asset.Futures, orderType: order.AnyType, tif: order.FillOrKill, exp: []string{orderTrigger}},
		{name: "any order type immediate or cancel on options", a: asset.Options, orderType: order.AnyType, tif: order.ImmediateOrCancel},
		{name: "any order type good till cancel", a: asset.Spot, orderType: order.AnyType, tif: order.GoodTillCancel},
		{name: "limit orders", a: asset.Spot, orderType: order.Limit},
		{name: "limit orders with a time in force", a: asset.Spot, orderType: order.Limit, tif: order.PostOnly},
		{name: "market orders", a: asset.PerpetualSwap, orderType: order.Market},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ordTypes, err := algoListingOrderTypes(tc.a, tc.orderType, tc.tif)
			require.ErrorIs(t, err, tc.err, "algoListingOrderTypes must return the expected error")
			assert.Equal(t, tc.exp, ordTypes, "algoListingOrderTypes should return the algo order listings the request reads")
		})
	}
}

func TestAlgoOrderDetail(t *testing.T) {
	t.Parallel()
	algo := func(edit func(*AlgoOrderResponse)) *AlgoOrderResponse {
		a := &AlgoOrderResponse{
			InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), AlgoID: "1", AlgoClientOrderID: "client1", OrderType: orderTrigger, Side: order.Buy,
			Size: 2, State: "live", TriggerPrice: 50000, OrderPrice: 49000, Leverage: 3, ReduceOnly: "true",
			CreationTime: types.Time(time.UnixMilli(1795409000000)), UpdateTime: types.Time(time.UnixMilli(1795409001000)),
		}
		if edit != nil {
			edit(a)
		}
		return a
	}
	detail := func(edit func(*order.Detail)) *order.Detail {
		d := &order.Detail{
			Amount: 2, Exchange: e.Name, OrderID: "1", ClientOrderID: "client1", Side: order.Buy, Type: order.Trigger, Pair: mainPair, AssetType: asset.Margin,
			Status: order.Active, ReduceOnly: true, Leverage: 3, Price: 49000, TriggerPrice: 50000, Date: time.UnixMilli(1795409000000), LastUpdated: time.UnixMilli(1795409001000),
		}
		if edit != nil {
			edit(d)
		}
		return d
	}
	for _, tc := range []struct {
		name string
		algo *AlgoOrderResponse
		a    asset.Item
		exp  *order.Detail
		err  error
	}{
		{name: "the pair and asset it is given", algo: algo(nil), a: asset.Margin, exp: detail(nil)},
		{
			name: "spot trailing stop buy", algo: algo(func(a *AlgoOrderResponse) { a.OrderType, a.ActivePrice = orderMoveOrderStop, 51000 }), a: asset.Spot,
			exp: detail(func(d *order.Detail) {
				d.Type, d.AssetType, d.Amount, d.QuoteAmount, d.Price, d.TriggerPrice = order.TrailingStop, asset.Spot, 0, 2, 0, 51000
			}),
		},
		{
			name: "triggered order", algo: algo(func(a *AlgoOrderResponse) { a.State = "effective" }), a: asset.Margin,
			exp: detail(func(d *order.Detail) { d.Status = order.Closed }),
		},
		{
			name: "trigger order placing an immediate or cancel order once triggered", algo: algo(func(a *AlgoOrderResponse) { a.AdvanceOrderType = orderIOC }), a: asset.Margin,
			exp: detail(func(d *order.Detail) { d.TimeInForce = order.ImmediateOrCancel }),
		},
		{
			name: "trigger order placing a fill or kill order once triggered", algo: algo(func(a *AlgoOrderResponse) { a.AdvanceOrderType = orderFOK }), a: asset.Margin,
			exp: detail(func(d *order.Detail) { d.TimeInForce = order.FillOrKill }),
		},
		{
			name: "trigger order's advance order type in upper case", algo: algo(func(a *AlgoOrderResponse) { a.AdvanceOrderType = "FOK" }), a: asset.Margin,
			exp: detail(func(d *order.Detail) { d.TimeInForce = order.FillOrKill }),
		},
		{
			name: "trigger order placing a chase order once triggered", algo: algo(func(a *AlgoOrderResponse) { a.AdvanceOrderType = orderChase }), a: asset.Margin,
			exp: detail(nil),
		},
		{
			name: "spot conditional buy at market with an ignored take profit at a limit price", a: asset.Spot,
			algo: algo(func(a *AlgoOrderResponse) {
				a.OrderType, a.StopLossTriggerPrice, a.StopLossOrderPrice, a.TakeProfitTriggerPrice, a.TakeProfitOrderPrice = orderConditional, 55000, -1, 45000, 44900
			}),
			exp: detail(func(d *order.Detail) {
				d.Type, d.AssetType, d.Amount, d.QuoteAmount, d.Price, d.TriggerPrice = order.ConditionalStop, asset.Spot, 0, 2, -1, 55000
			}),
		},
		{
			name: "spot conditional buy whose take profit at market triggered beside a stop loss at a limit price", a: asset.Spot,
			algo: algo(func(a *AlgoOrderResponse) {
				a.OrderType, a.State, a.ActualSide = orderConditional, "effective", "tp"
				a.StopLossTriggerPrice, a.StopLossOrderPrice, a.TakeProfitTriggerPrice, a.TakeProfitOrderPrice = 55000, 55100, 45000, -1
			}),
			exp: detail(func(d *order.Detail) {
				d.Type, d.AssetType, d.Status, d.Amount, d.QuoteAmount, d.Price, d.TriggerPrice = order.ConditionalStop, asset.Spot, order.Closed, 0, 2, -1, 45000
			}),
		},
		{name: "unknown order type", algo: algo(func(a *AlgoOrderResponse) { a.OrderType = orderSmartIceberg }), a: asset.Spot, err: order.ErrTypeIsInvalid},
		{name: "unknown state", algo: algo(func(a *AlgoOrderResponse) { a.State = "filled" }), a: asset.Spot, err: order.ErrUnsupportedStatusType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d, err := e.algoOrderDetail(tc.algo, mainPair, tc.a)
			require.ErrorIs(t, err, tc.err, "algoOrderDetail must return the expected error")
			assert.Equal(t, tc.exp, d, "algoOrderDetail should report the algo order as SubmitOrder places it")
		})
	}
}

func TestAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	algoRow := func(id string, created int64, fields map[string]string) map[string]string {
		r := map[string]string{
			"instType": instTypeSpot, "instId": mainPair.String(), "algoId": id, "ordType": orderTrigger, "side": "buy", "sz": "1", "state": "live",
			"triggerPx": "50000", "ordPx": "49000", "cTime": strconv.FormatInt(created, 10),
		}
		maps.Copy(r, fields)
		return r
	}
	fetchFrom := func(ctx context.Context, ex *Exchange) func(string) ([]AlgoOrderResponse, error) {
		return func(after string) ([]AlgoOrderResponse, error) {
			return ex.GetAlgoOrderList(ctx, &AlgoOrderListRequest{OrderType: orderTrigger, After: after})
		}
	}

	t.Run("reads each page before checking its cursor", func(t *testing.T) {
		t.Parallel()
		for _, tc := range pagerCases(func(i int) map[string]string { return algoRow(fmt.Sprintf("ORD-%03d", i), 1795409000000, nil) }) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, pagerHandler(t, "/trade/orders-algo-pending", "after", tc.pages))
				details, err := ex.algoOrderDetails(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot}, make(common.Seen[string]), fetchFrom(t.Context(), ex))
				if tc.incomplete {
					require.ErrorIs(t, err, errListingStalled, "algoOrderDetails must report a listing which stops advancing")
				} else {
					require.NoError(t, err, "algoOrderDetails must not error")
				}
				assert.Equal(t, tc.exp, detailOrderIDs(details), "algoOrderDetails should return every order of every page once, in listing order")
			})
		}
	})

	t.Run("stops at a cursor requested before with the orders read", func(t *testing.T) {
		t.Parallel()
		row := func(i int) map[string]string { return algoRow(fmt.Sprintf("ORD-%03d", i), 1795409000000, nil) }
		first, second := pagerRows(row, 0, orderListPageSize), pagerRows(row, orderListPageSize, 2*orderListPageSize)
		lastOfFirst, lastOfSecond := fmt.Sprintf("ORD-%03d", orderListPageSize-1), fmt.Sprintf("ORD-%03d", 2*orderListPageSize-1)
		for _, tc := range []struct {
			name  string
			pages map[string][]map[string]string
			exp   []string
		}{
			{name: "cursors which cycle", pages: map[string][]map[string]string{"": first, lastOfFirst: second, lastOfSecond: first}, exp: pagerIDs(2 * orderListPageSize)},
			{name: "a short page of its own cursor", pages: map[string][]map[string]string{"": first, lastOfFirst: {row(orderListPageSize - 1)}}, exp: pagerIDs(orderListPageSize)},
			{
				name:  "a short page leading back to an earlier cursor",
				pages: map[string][]map[string]string{"": first, lastOfFirst: second, lastOfSecond: {row(2 * orderListPageSize), row(orderListPageSize - 1)}},
				exp:   pagerIDs(2*orderListPageSize + 1),
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, pagerHandler(t, "/trade/orders-algo-pending", "after", tc.pages))
				details, err := ex.algoOrderDetails(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot}, make(common.Seen[string]), fetchFrom(t.Context(), ex))
				require.ErrorIs(t, err, errListingStalled, "algoOrderDetails must report a cursor requested before rather than request it again")
				assert.Equal(t, tc.exp, detailOrderIDs(details), "algoOrderDetails should return the orders it read beside the error")
			})
		}
	})

	ethUSDT := currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter)
	for _, tc := range []struct {
		name  string
		req   *order.MultiOrderRequest
		seen  []string
		pages map[string][]map[string]string
		exp   []string
		err   error
	}{
		{
			name: "starts from the request's order ID", req: &order.MultiOrderRequest{AssetType: asset.Spot, FromOrderID: "90"},
			pages: map[string][]map[string]string{"90": {algoRow("89", 1795409000000, nil), algoRow("88", 1795409000000, nil)}},
			exp:   []string{"89", "88"},
		},
		{
			name: "stops at the first order created before the start", req: &order.MultiOrderRequest{AssetType: asset.Spot, StartTime: time.UnixMilli(1795409000100).Add(500 * time.Microsecond)},
			pages: map[string][]map[string]string{"": {
				algoRow("4", 1795409000200, nil), algoRow("3", 1795409000100, nil), algoRow("2", 1795409000099, nil), algoRow("1", 1795409000300, nil),
			}},
			exp: []string{"4", "3"},
		},
		{
			name: "leaves out the orders created after the end", req: &order.MultiOrderRequest{AssetType: asset.Spot, EndTime: time.UnixMilli(1795409000100)},
			pages: map[string][]map[string]string{"": {algoRow("3", 1795409000101, nil), algoRow("2", 1795409000100, nil), algoRow("1", 1795409000099, nil)}},
			exp:   []string{"2", "1"},
		},
		{
			name: "keeps the requested pairs", req: &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{ethUSDT}},
			pages: map[string][]map[string]string{"": {algoRow("2", 1795409000000, nil), algoRow("1", 1795409000000, map[string]string{"instId": ethUSDT.String()})}},
			exp:   []string{"1"},
		},
		{
			name: "an order an earlier listing returned", req: &order.MultiOrderRequest{AssetType: asset.Spot}, seen: []string{"2"},
			pages: map[string][]map[string]string{"": {algoRow("2", 1795409000000, nil), algoRow("1", 1795409000000, nil)}},
			exp:   []string{"1"},
		},
		{
			name: "an order type it cannot map", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages: map[string][]map[string]string{"": {algoRow("1", 1795409000000, map[string]string{"ordType": orderSmartIceberg})}},
			err:   order.ErrTypeIsInvalid,
		},
		{
			name: "an instrument ID too short for a pair", req: &order.MultiOrderRequest{AssetType: asset.Spot},
			pages: map[string][]map[string]string{"": {algoRow("1", 1795409000000, map[string]string{"instId": ""})}},
			err:   currency.ErrCreatingPair,
		},
		{name: "a listing error", req: &order.MultiOrderRequest{AssetType: asset.Spot}, err: ErrorCodes["50011"]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/trade/orders-algo-pending", r.URL.Path, "algoOrderDetails should read the listing it is given")
				if tc.pages == nil {
					_, err := w.Write([]byte(`{"code":"50011","msg":"Rate limit reached","data":[]}`))
					assert.NoError(t, err, "writing the reply should not error")
					return
				}
				writeOKXData(t, w, tc.pages[r.URL.Query().Get("after")])
			}))
			seen := make(common.Seen[string])
			for _, id := range tc.seen {
				seen[id] = true
			}
			details, err := ex.algoOrderDetails(t.Context(), tc.req, seen, fetchFrom(t.Context(), ex))
			require.ErrorIs(t, err, tc.err, "algoOrderDetails must return the expected error")
			if tc.err == nil {
				assert.Equal(t, tc.exp, detailOrderIDs(details), "algoOrderDetails should return the listed orders the request selects")
			}
		})
	}

	t.Run("a cancelled context", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := ex.algoOrderDetails(ctx, &order.MultiOrderRequest{AssetType: asset.Spot}, make(common.Seen[string]), fetchFrom(ctx, ex))
		assert.ErrorIs(t, err, context.Canceled, "algoOrderDetails should return the context error before any request")
	})
}

func TestGetAlgoOrderHistoryDetails(t *testing.T) {
	t.Parallel()
	algoRow := func(id, ordType, state string) map[string]string {
		return map[string]string{
			"instType": instTypeMargin, "instId": mainPair.String(), "algoId": id, "ordType": ordType, "side": "buy", "sz": "1", "state": state,
			"slTriggerPx": "45000", "slOrdPx": "-1", "triggerPx": "50000", "ordPx": "49000", "cTime": "1795409000000",
		}
	}
	listings := map[string][]map[string]string{
		"trigger effective":            {algoRow("3", orderTrigger, "effective")},
		"trigger canceled":             {algoRow("2", orderTrigger, "canceled"), algoRow("3", orderTrigger, "effective")},
		"conditional,oco order_failed": {algoRow("1", orderConditional, "order_failed")},
	}
	stalledPage := make([]map[string]string, 0, orderListPageSize)
	stalledIDs := make([]string, 0, orderListPageSize)
	for i := range orderListPageSize {
		id := fmt.Sprintf("S%03d", i)
		stalledPage = append(stalledPage, algoRow(id, orderTrigger, "canceled"))
		stalledIDs = append(stalledIDs, id)
	}
	for _, tc := range []struct {
		name     string
		ordTypes []string
		failing  string
		stalling string // answers every cursor with the same full page
		exp      []string
		requests []string
		err      error
	}{
		{
			name: "each state of each order type", ordTypes: []string{orderTrigger, "conditional,oco"}, exp: []string{"3", "2", "1"},
			requests: []string{
				"MARGIN trigger effective", "MARGIN trigger canceled", "MARGIN trigger order_failed",
				"MARGIN conditional,oco effective", "MARGIN conditional,oco canceled", "MARGIN conditional,oco order_failed",
			},
		},
		{name: "no order types", exp: []string{}},
		{
			name: "a listing error", ordTypes: []string{orderTrigger, "conditional,oco"}, failing: "trigger canceled", exp: []string{},
			requests: []string{"MARGIN trigger effective", "MARGIN trigger canceled"}, err: ErrorCodes["50011"],
		},
		{
			name: "a listing which stops advancing leaves the others to read", ordTypes: []string{orderTrigger, "conditional,oco"}, stalling: "trigger canceled",
			exp: slices.Concat([]string{"3"}, stalledIDs, []string{"1"}),
			requests: []string{
				"MARGIN trigger effective", "MARGIN trigger canceled", "MARGIN trigger canceled", "MARGIN trigger order_failed",
				"MARGIN conditional,oco effective", "MARGIN conditional,oco canceled", "MARGIN conditional,oco order_failed",
			},
			err: errListingStalled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var requests []string
			ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/trade/orders-algo-history", r.URL.Path, "getAlgoOrderHistoryDetails should read the algo order history")
				q := r.URL.Query()
				listing := q.Get("ordType") + " " + q.Get("state")
				mu.Lock()
				requests = append(requests, q.Get("instType")+" "+listing)
				mu.Unlock()
				switch listing {
				case tc.failing:
					_, err := w.Write([]byte(`{"code":"50011","msg":"Rate limit reached","data":[]}`))
					assert.NoError(t, err, "writing the reply should not error")
				case tc.stalling:
					writeOKXData(t, w, stalledPage)
				default:
					writeOKXData(t, w, listings[listing])
				}
			}))
			details, err := ex.getAlgoOrderHistoryDetails(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin}, tc.ordTypes)
			require.ErrorIs(t, err, tc.err, "getAlgoOrderHistoryDetails must return the expected error")
			if tc.stalling != "" {
				assert.ErrorContains(t, err, "listing trigger algo orders in the canceled state", "the error should name the listing which stops advancing")
			}
			assert.Equal(t, tc.exp, detailOrderIDs(details), "getAlgoOrderHistoryDetails should return each completed algo order once")
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tc.requests, requests, "getAlgoOrderHistoryDetails should read each order type's history in each state")
		})
	}
}

func TestContractOrdersLeaveCostUnset(t *testing.T) {
	t.Parallel()
	created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
	row := func(state string) map[string]string {
		return map[string]string{
			"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": state,
			"sz": "100", "accFillSz": "50", "avgPx": "60000", "px": "60000", "fee": "-3", "feeCcy": "USDT", "cTime": created, "uTime": created,
		}
	}
	e := newMockExchange(t, withoutAlgoOrders(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade/order", "/trade/orders-pending":
			writeOKXData(t, w, []map[string]string{row("partially_filled")})
		case "/trade/orders-history", "/trade/orders-history-archive":
			writeOKXData(t, w, []map[string]string{row("canceled")})
		default:
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	info, err := e.GetOrderInfo(t.Context(), "ORD-1", perpetualSwapPair, asset.PerpetualSwap)
	require.NoError(t, err, "GetOrderInfo must not error")
	active, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: order.AnyType, Side: order.AnySide})
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, active, 1, "GetActiveOrders must return the pending order")
	history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualSwap, Type: order.AnyType, Side: order.AnySide})
	require.NoError(t, err, "GetOrderHistory must not error")
	require.Len(t, history, 1, "GetOrderHistory must return the order")
	for name, d := range map[string]*order.Detail{"GetOrderInfo": info, "GetActiveOrders": &active[0], "GetOrderHistory": &history[0]} {
		assert.Zerof(t, d.Cost, "%s should leave a contract order's cost unset", name)
		assert.Truef(t, d.CostAsset.IsEmpty(), "%s should leave a contract order's cost asset unset", name)
		assert.Equalf(t, 60000.0, d.AverageExecutedPrice, "%s should still report the average price", name)
	}
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	t.Run("reads a charged fee as positive and a rebate as negative", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		row := func(id, fee string) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled",
				"sz": "100", "accFillSz": "100", "avgPx": "60000", "px": "60000", "fee": fee, "feeCcy": "USDT", "cTime": created, "uTime": created,
			}
		}
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/trade/orders-history":
				writeOKXData(t, w, []map[string]string{row("CHARGED", "-3"), row("REBATED", "0.5")})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, StartDate: timeNow().Add(-time.Hour), EndDate: timeNow(),
		})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
		fees := make(map[string]float64, len(positions[0].Orders))
		for i := range positions[0].Orders {
			fees[positions[0].Orders[i].OrderID] = positions[0].Orders[i].Fee
		}
		assert.Equal(t, map[string]float64{"CHARGED": 3, "REBATED": -0.5}, fees, "a fee OKX charged should read as a positive cost and a rebate as a negative one, as GetOrderHistory reports them")
	})
	t.Run("reports contract costs by settlement type", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		inverseSwapPair := currency.NewPairWithDelimiter("BTC", "USD-SWAP", "-")
		for _, tc := range []struct {
			name       string
			instrument map[string]string
			pair       currency.Pair
			state      string
			avgPx      string
			expType    futures.ContractSettlementType
			expCost    float64
			expAsset   currency.Code
		}{
			{
				name:       "linear",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"},
				pair:       perpetualSwapPair,
				state:      "filled",
				avgPx:      "60000",
				expType:    futures.Linear,
				expCost:    60000,
				expAsset:   currency.USDT,
			},
			{
				name:       "inverse",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"},
				pair:       inverseSwapPair,
				state:      "filled",
				avgPx:      "60000",
				expType:    futures.Inverse,
				expCost:    100.0 * 100.0 / 60000.0,
				expAsset:   currency.BTC,
			},
			{
				name:       "suspended inverse",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "suspend"},
				pair:       inverseSwapPair,
				state:      "filled",
				avgPx:      "60000",
				expType:    futures.Inverse,
				expCost:    100.0 * 100.0 / 60000.0,
				expAsset:   currency.BTC,
			},
			{
				name:       "inverse fill without an average price",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"},
				pair:       inverseSwapPair,
				state:      "filled",
				avgPx:      "0",
				expType:    futures.Inverse,
				expCost:    0,
				expAsset:   currency.BTC,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/instruments":
						writeOKXData(t, w, []map[string]string{tc.instrument})
					case "/trade/orders-history":
						writeOKXData(t, w, []map[string]string{{
							"instType": instTypeSwap, "instId": tc.instrument["instId"], "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": tc.state,
							"sz": "100", "accFillSz": map[string]string{"filled": "100", "live": "0"}[tc.state], "avgPx": tc.avgPx, "px": "60000", "cTime": created, "uTime": created,
						}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.PerpetualSwap, Pairs: currency.Pairs{tc.pair}, StartDate: timeNow().Add(-time.Hour), EndDate: timeNow(),
				})
				require.NoError(t, err, "GetFuturesPositionOrders must not error")
				require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
				assert.Equal(t, tc.expType, positions[0].ContractSettlementType, "the position should carry the contract's settlement type")
				require.Len(t, positions[0].Orders, 1, "GetFuturesPositionOrders must return the order")
				got := positions[0].Orders[0]
				assert.Equal(t, tc.expCost, got.Cost, "a contract fill's cost should follow its settlement type")
				assert.Equal(t, tc.expAsset, got.CostAsset, "the cost should be reported in the contract's settlement currency")
				assert.Equal(t, 100.0, got.ContractAmount, "the order amount should stay in contracts")
				assert.False(t, math.IsNaN(got.Cost), "a fill's cost should never be not-a-number")
			})
		}
	})
	t.Run("scales execution amounts by the contract value", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		for _, tc := range []struct {
			name                        string
			instrument                  map[string]string
			pair                        currency.Pair
			amount, executed, remaining float64
		}{
			{
				name:       "linear",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"},
				pair:       perpetualSwapPair,
				amount:     1, executed: 0.4, remaining: 0.6,
			},
			{
				name:       "inverse",
				instrument: map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"},
				pair:       currency.NewPairWithDelimiter("BTC", "USD-SWAP", "-"),
				amount:     10000, executed: 4000, remaining: 6000,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/instruments":
						writeOKXData(t, w, []map[string]string{tc.instrument})
					case "/trade/orders-history":
						writeOKXData(t, w, []map[string]string{{
							"instType": instTypeSwap, "instId": tc.instrument["instId"], "ordId": "ORD-1", "ordType": orderLimit, "side": "buy", "state": "live",
							"sz": "100", "accFillSz": "40", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
						}})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.PerpetualSwap, Pairs: currency.Pairs{tc.pair}, StartDate: timeNow().Add(-time.Hour), EndDate: timeNow(),
				})
				require.NoError(t, err, "GetFuturesPositionOrders must not error")
				require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
				require.Len(t, positions[0].Orders, 1, "GetFuturesPositionOrders must return the order")
				got := positions[0].Orders[0]
				assert.Equal(t, tc.amount, got.Amount, "the amount should be the contract count scaled by the contract value")
				assert.Equal(t, tc.executed, got.ExecutedAmount, "the executed amount should share the amount's unit")
				assert.Equal(t, tc.remaining, got.RemainingAmount, "the remaining amount should share the amount's unit")
				assert.Equal(t, 100.0, got.ContractAmount, "the contract amount should keep the contract count")
			})
		}
	})
	t.Run("reports a reduce only order", func(t *testing.T) {
		t.Parallel()
		now := timeNow()
		created := now.Add(-time.Minute).UnixMilli()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/trade/orders-history":
				writeOKXData(t, w, []map[string]string{{
					"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": "ORD-1", "ordType": orderLimit, "side": "sell", "posSide": "net", "state": "filled",
					"sz": "2", "accFillSz": "2", "avgPx": "60000", "px": "60000", "reduceOnly": "true", "cTime": strconv.FormatInt(created, 10), "uTime": strconv.FormatInt(created, 10),
				}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, StartDate: now.Add(-time.Hour), EndDate: now,
		})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
		exp := []order.Detail{{
			Price: 60000, AverageExecutedPrice: 60000, Amount: 0.02, ContractAmount: 2, ExecutedAmount: 0.02, Exchange: e.Name, OrderID: "ORD-1",
			Type: order.Limit, Side: order.Sell, Status: order.Filled, AssetType: asset.PerpetualSwap, ReduceOnly: true, Date: time.UnixMilli(created),
			LastUpdated: time.UnixMilli(created), Pair: perpetualSwapPair, Cost: 1200, CostAsset: currency.USDT,
		}}
		assert.Equal(t, exp, positions[0].Orders, "GetFuturesPositionOrders should report the order's reduce only flag")
	})
	t.Run("pages the history listing with the after cursor", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		row := func(id string) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled",
				"sz": "1", "accFillSz": "1", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
			}
		}
		first := make([]map[string]string, 0, orderListPageSize)
		for i := range orderListPageSize {
			first = append(first, row(fmt.Sprintf("ORD-%03d", i)))
		}
		last := make([]map[string]string, 0, 50)
		for i := range 50 {
			last = append(last, row(fmt.Sprintf("ORD-%03d", orderListPageSize+i)))
		}
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/trade/orders-history":
				after := r.URL.Query().Get("after")
				switch after {
				case "":
					writeOKXData(t, w, first)
				case "ORD-099":
					writeOKXData(t, w, last)
				default:
					t.Errorf("unexpected after cursor %q", after)
					http.Error(w, "unexpected cursor", http.StatusBadRequest)
				}
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, StartDate: timeNow().Add(-time.Hour), EndDate: timeNow(),
		})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
		require.Len(t, positions[0].Orders, 150, "GetFuturesPositionOrders must crawl every page of the instrument's window")
		assert.Equal(t, "ORD-000", positions[0].Orders[0].OrderID, "the first page's orders should come first")
		assert.Equal(t, "ORD-149", positions[0].Orders[149].OrderID, "the second page's orders should follow")
	})
	t.Run("reads the history listings the window reaches", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(timeNow().Add(-time.Minute).UnixMilli(), 10)
		row := func(id string) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled",
				"sz": "1", "accFillSz": "1", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
			}
		}
		first := make([]map[string]string, 0, orderListPageSize)
		for i := range orderListPageSize {
			first = append(first, row(fmt.Sprintf("ORD-%03d", i)))
		}
		const day = 24 * time.Hour
		for _, tc := range []struct {
			name       string
			start, end time.Duration
			listings   map[string]int
		}{
			{name: "window within 7 days", start: 2 * day, listings: map[string]int{"/trade/orders-history": 2}},
			{name: "window older than 7 days", start: 14 * day, listings: map[string]int{"/trade/orders-history": 2, "/trade/orders-history-archive": 2}},
			{name: "window ending before 7 days ago", start: 14 * day, end: 10 * day, listings: map[string]int{"/trade/orders-history": 2, "/trade/orders-history-archive": 2}},
			{name: "window starting a minute inside 7 days", start: 7*day - time.Minute, listings: map[string]int{"/trade/orders-history": 2}},
			{name: "window starting a minute past 7 days", start: 7*day + time.Minute, listings: map[string]int{"/trade/orders-history": 2, "/trade/orders-history-archive": 2}},
			{name: "window ending a minute inside 7 days", start: 14 * day, end: 7*day - time.Minute, listings: map[string]int{"/trade/orders-history": 2, "/trade/orders-history-archive": 2}},
			{name: "window ending a minute past 7 days", start: 14 * day, end: 7*day + time.Minute, listings: map[string]int{"/trade/orders-history": 2, "/trade/orders-history-archive": 2}},
			// StartDate turns 7 days old while the first page is served, so
			// either listing may be read, or both, but each to its last page.
			{name: "window turning 7 days old during the crawl", start: 7*day - 100*time.Millisecond},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				listings := make(map[string]int)
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/instruments":
						writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
					case "/trade/orders-history", "/trade/orders-history-archive":
						mu.Lock()
						listings[r.URL.Path]++
						mu.Unlock()
						if r.URL.Query().Get("after") == "" {
							// The first page outlasts the 100ms the last case's
							// StartDate has left before it is 7 days old.
							time.Sleep(200 * time.Millisecond)
							writeOKXData(t, w, first)
							return
						}
						writeOKXData(t, w, []map[string]string{row("ORD-100")})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair},
					StartDate: timeNow().Add(-tc.start), EndDate: timeNow().Add(-tc.end),
				})
				require.NoError(t, err, "GetFuturesPositionOrders must not error")
				require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
				assert.Len(t, positions[0].Orders, orderListPageSize+1, "GetFuturesPositionOrders should return both pages once")
				mu.Lock()
				defer mu.Unlock()
				if tc.listings != nil {
					assert.Equal(t, tc.listings, listings, "both pages of each listing the window reaches should be read")
					return
				}
				require.NotEmpty(t, listings, "a history listing must be read")
				for listing, pages := range listings {
					assert.Equalf(t, 2, pages, "both pages of %s should be read", listing)
				}
			})
		}
	})
	// A window starting before the 7 day mark and ending after it needs both
	// listings: the archive for its older orders, the 7 day listing for the
	// newest, which the archive lags, and for cancelled orders without fills.
	t.Run("judges the order history limit by the wrapper's clock", func(t *testing.T) {
		t.Parallel()
		limit := kline.OneDay.Duration() * 90
		for _, tc := range []struct {
			name    string
			age     time.Duration
			respect bool
			err     error
		}{
			{name: "a start a minute past the limit", age: limit + time.Minute, err: futures.ErrOrderHistoryTooLarge},
			{name: "a start a minute inside the limit", age: limit - time.Minute},
			{name: "a start past the limit moves to it when the limit is respected", age: limit + time.Hour, respect: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				begins := make(map[string]string)
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/instruments":
						writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
					case "/trade/orders-history", "/trade/orders-history-archive":
						mu.Lock()
						begins[r.URL.Path] = r.URL.Query().Get("begin")
						mu.Unlock()
						writeOKXData(t, w, []any{})
					default:
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				before := timeNow()
				_, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair},
					StartDate: before.Add(-tc.age), EndDate: before, RespectOrderHistoryLimits: tc.respect,
				})
				after := timeNow()
				require.ErrorIs(t, err, tc.err, "GetFuturesPositionOrders must judge the start by the wrapper's clock")
				if !tc.respect {
					return
				}
				mu.Lock()
				begin, err := strconv.ParseInt(begins["/trade/orders-history-archive"], 10, 64)
				mu.Unlock()
				require.NoError(t, err, "the archive must be asked for the moved start")
				assert.WithinRange(t, time.UnixMilli(begin), before.Add(-limit).Truncate(time.Millisecond), after.Add(-limit), "the start should move to the limit by the wrapper's clock")
			})
		}
	})

	t.Run("crawls both listings across the 7 day mark", func(t *testing.T) {
		t.Parallel()
		start, end := timeNow().Add(-14*24*time.Hour), timeNow()
		row := func(id string, age time.Duration) map[string]string {
			created := strconv.FormatInt(end.Add(-age).UnixMilli(), 10)
			return map[string]string{
				"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": id, "ordType": orderLimit, "side": "buy", "state": "filled",
				"sz": "1", "accFillSz": "1", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
			}
		}
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/trade/orders-history":
				assert.Equal(t, url.Values{
					"instType": {instTypeSwap}, "instId": {"BTC-USDT-SWAP"},
					"begin": {strconv.FormatInt(start.UnixMilli(), 10)}, "end": {strconv.FormatInt(end.UnixMilli(), 10)},
				}, r.URL.Query(), "the 7 day listing should be asked for the window")
				writeOKXData(t, w, []map[string]string{row("NEW-CANCELLED", time.Hour), row("SHARED", 2*24*time.Hour)})
			case "/trade/orders-history-archive":
				assert.Equal(t, url.Values{
					"instType": {instTypeSwap}, "instId": {"BTC-USDT-SWAP"},
					"begin": {strconv.FormatInt(start.UnixMilli(), 10)}, "end": {strconv.FormatInt(end.UnixMilli(), 10)},
				}, r.URL.Query(), "the archive should be asked for the window")
				writeOKXData(t, w, []map[string]string{row("SHARED", 2*24*time.Hour), row("OLD", 10*24*time.Hour)})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, StartDate: start, EndDate: end,
		})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
		ids := make([]string, 0, len(positions[0].Orders))
		for i := range positions[0].Orders {
			ids = append(ids, positions[0].Orders[i].OrderID)
		}
		assert.Equal(t, []string{"NEW-CANCELLED", "SHARED", "OLD"}, ids, "GetFuturesPositionOrders should return each listing's orders once, the 7 day listing's first")
	})
	t.Run("reads each page before checking its cursor", func(t *testing.T) {
		t.Parallel()
		created := strconv.FormatInt(time.Now().Add(-time.Minute).UnixMilli(), 10)
		row := func(i int) map[string]string {
			return map[string]string{
				"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "ordId": fmt.Sprintf("ORD-%03d", i), "ordType": orderLimit, "side": "buy",
				"state": "filled", "sz": "1", "accFillSz": "1", "avgPx": "60000", "px": "60000", "cTime": created, "uTime": created,
			}
		}
		for _, tc := range pagerCases(row) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				history := pagerHandler(t, "/trade/orders-history", "after", tc.pages)
				e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/public/instruments" {
						writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
						return
					}
					history(w, r)
				}))
				positions, err := e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, StartDate: timeNow().Add(-time.Hour), EndDate: timeNow(),
				})
				require.NoError(t, err, "GetFuturesPositionOrders must not error")
				require.Len(t, positions, 1, "GetFuturesPositionOrders must return the requested pair")
				ids := make([]string, 0, len(positions[0].Orders))
				for i := range positions[0].Orders {
					ids = append(ids, positions[0].Orders[i].OrderID)
				}
				assert.Equal(t, tc.exp, ids, "GetFuturesPositionOrders should return every order of every page once, in listing order")
			})
		}
	})

	inverseSwap := map[string]string{"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"}
	historyRow := func(id, state, accFillSz string, created, updated int64) map[string]string {
		return map[string]string{
			"instType": instTypeSwap, "instId": "BTC-USD-SWAP", "ordId": id, "clOrdId": "client-" + id, "ordType": orderLimit, "side": "buy", "state": state,
			"sz": "2", "accFillSz": accFillSz, "avgPx": "50000", "px": "50000", "fee": "-0.00001", "feeCcy": "BTC",
			"cTime": strconv.FormatInt(created, 10), "uTime": strconv.FormatInt(updated, 10),
		}
	}
	// historyDetail is the report of historyRow's order on BTC-USD-SWAP, whose contracts are worth 100 USD
	historyDetail := func(id string, status order.Status, executed float64, created, updated int64) order.Detail {
		remaining := 0.0
		if status != order.Filled {
			remaining = 2 - executed
		}
		return order.Detail{
			Price: 50000, AverageExecutedPrice: 50000, Amount: 200, ContractAmount: 2, ExecutedAmount: executed * 100, RemainingAmount: remaining * 100,
			Fee: 0.00001, FeeAsset: currency.BTC, Exchange: "Okx", OrderID: id, ClientOrderID: "client-" + id, Type: order.Limit, Side: order.Buy,
			Status: status, AssetType: asset.PerpetualSwap, Date: time.UnixMilli(created), LastUpdated: time.UnixMilli(updated), Pair: btcUSDSwapPair,
			Cost: 100 * executed / 50000, CostAsset: currency.BTC,
		}
	}

	t.Run("reads an order created before the last 7 days that completed in them", func(t *testing.T) {
		t.Parallel()
		created := timeNow().Add(-10 * 24 * time.Hour).UnixMilli()
		completed := timeNow().Add(-time.Minute).UnixMilli()
		start, end := time.UnixMilli(created).Add(-time.Hour), time.UnixMilli(created).Add(time.Hour)
		window := url.Values{
			"instType": {instTypeSwap}, "instId": {"BTC-USD-SWAP"},
			"begin": {strconv.FormatInt(start.UnixMilli(), 10)}, "end": {strconv.FormatInt(end.UnixMilli(), 10)},
		}
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{inverseSwap})
			case "/trade/orders-history":
				assert.Equal(t, window, r.URL.Query(), "the 7 day listing should be asked for the window's orders")
				writeOKXData(t, w, []map[string]string{historyRow("31", "filled", "2", created, completed)})
			case "/trade/orders-history-archive":
				assert.Equal(t, window, r.URL.Query(), "the archive should be asked for the window's orders")
				writeOKXData(t, w, []any{})
			default:
				assert.Failf(t, "unexpected request", "the position orders should not reach %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		positions, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.PerpetualSwap, Pairs: currency.Pairs{btcUSDSwapPair}, StartDate: start, EndDate: end,
		})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		exp := []futures.PositionResponse{{
			Pair: btcUSDSwapPair, Asset: asset.PerpetualSwap, ContractSettlementType: futures.Inverse,
			Orders: []order.Detail{historyDetail("31", order.Filled, 2, created, completed)},
		}}
		assert.Equal(t, exp, positions, "GetFuturesPositionOrders should report an order the window created which completed in the last 7 days")
	})

	t.Run("reports only the orders that executed something", func(t *testing.T) {
		t.Parallel()
		created := timeNow().Add(-time.Hour).UnixMilli()
		updated := timeNow().Add(-time.Minute).UnixMilli()
		partlyFilled := historyRow("42", "canceled", "1", created, updated)
		partlyFilled["ordType"] = orderPostOnly
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{inverseSwap})
			case "/trade/orders-history":
				writeOKXData(t, w, []map[string]string{
					historyRow("41", "canceled", "0", created, updated),
					partlyFilled,
					historyRow("43", "live", "0", created, updated),
					historyRow("44", "filled", "2", created, updated),
				})
			default:
				assert.Failf(t, "unexpected request", "the position orders should not reach %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		positions, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.PerpetualSwap, Pairs: currency.Pairs{btcUSDSwapPair}, StartDate: timeNow().Add(-2 * time.Hour), EndDate: timeNow(),
		})
		require.NoError(t, err, "GetFuturesPositionOrders must not error")
		partlyFilledDetail := historyDetail("42", order.Cancelled, 1, created, updated)
		partlyFilledDetail.TimeInForce = order.PostOnly
		exp := []futures.PositionResponse{{
			Pair: btcUSDSwapPair, Asset: asset.PerpetualSwap, ContractSettlementType: futures.Inverse,
			Orders: []order.Detail{partlyFilledDetail, historyDetail("44", order.Filled, 2, created, updated)},
		}}
		assert.Equal(t, exp, positions, "GetFuturesPositionOrders should leave out the orders that executed nothing, as they made up no position")
	})

	t.Run("refuses an unvalued contract's position orders", func(t *testing.T) {
		t.Parallel()
		expired := currency.NewPairWithDelimiter("BTC", "USD-261120", currency.DashDelimiter)
		listed := currency.NewPairWithDelimiter("BTC", "USD-261225", currency.DashDelimiter)
		created := timeNow().Add(-5 * 24 * time.Hour).UnixMilli()
		listedContract := map[string]string{"instType": instTypeFutures, "instId": listed.String(), "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "100", "state": "live"}
		malformedContract := func(contractValue string) map[string]string {
			return map[string]string{"instType": instTypeFutures, "instId": expired.String(), "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": contractValue, "state": "live"}
		}
		for _, tc := range []struct {
			name        string
			instruments []map[string]string
			pairs       currency.Pairs
		}{
			{name: "contract missing from the instrument list", instruments: []map[string]string{listedContract}, pairs: currency.Pairs{expired}},
			{
				name:        "contract without a contract value",
				instruments: []map[string]string{{"instType": instTypeFutures, "instId": expired.String(), "uly": "BTC-USD", "ctType": "inverse", "settleCcy": "BTC", "ctVal": "", "state": "live"}},
				pairs:       currency.Pairs{expired},
			},
			{name: "unvalued contract after a valued one", instruments: []map[string]string{listedContract}, pairs: currency.Pairs{listed, expired}},
			{name: "contract valued at not a number", instruments: []map[string]string{malformedContract("NaN")}, pairs: currency.Pairs{expired}},
			{name: "contract valued at infinity", instruments: []map[string]string{malformedContract("Inf")}, pairs: currency.Pairs{expired}},
			{name: "contract valued below zero", instruments: []map[string]string{malformedContract("-100")}, pairs: currency.Pairs{expired}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/public/instruments":
						writeOKXData(t, w, tc.instruments)
					case "/trade/orders-history":
						instrumentID := r.URL.Query().Get("instId")
						if !assert.Equal(t, listed.String(), instrumentID, "an unvalued contract's orders should not be read") {
							http.NotFound(w, r)
							return
						}
						row := historyRow("912", "filled", "2", created, created)
						row["instType"], row["instId"] = instTypeFutures, instrumentID
						writeOKXData(t, w, []map[string]string{row})
					default:
						assert.Failf(t, "unexpected request", "the position orders should not reach %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				positions, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
					Asset: asset.Futures, Pairs: tc.pairs, StartDate: timeNow().Add(-6 * 24 * time.Hour), EndDate: timeNow().Add(-4 * 24 * time.Hour),
				})
				require.ErrorIs(t, err, futures.ErrContractNotSupported, "GetFuturesPositionOrders must refuse a contract its instrument list does not value")
				assert.Nil(t, positions, "GetFuturesPositionOrders should report no position orders beside the refusal")
			})
		}
	})

	t.Run("judges a future start by the wrapper's clock", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Failf(t, "unexpected request", "a refused window should not reach %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		start := timeNow().Add(time.Minute)
		_, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.PerpetualSwap, Pairs: currency.Pairs{perpetualSwapPair}, StartDate: start, EndDate: start.Add(time.Hour),
		})
		assert.ErrorIs(t, err, common.ErrStartAfterTimeNow, "GetFuturesPositionOrders should refuse a window starting after the wrapper's clock")
	})

	t.Run("refuses a spread", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Failf(t, "unexpected request", "a refused spread should not reach %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		_, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{
			Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}, StartDate: timeNow().Add(-time.Hour), EndDate: timeNow(),
		})
		assert.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesPositionOrders should refuse a spread, whose positions are its legs'")
	})
}

func TestStartEndTimeCheckAt(t *testing.T) {
	t.Parallel()
	ahead := time.Now().Add(10 * 365 * 24 * time.Hour)
	for _, tc := range []struct {
		name            string
		start, end, now time.Time
		expErr          error
	}{
		{name: "window before a clock ahead of the wall clock", start: ahead.Add(-2 * time.Hour), end: ahead.Add(-time.Hour), now: ahead},
		{name: "window starting at the clock", start: ahead, end: ahead.Add(time.Hour), now: ahead},
		{name: "window starting after a clock behind the wall clock", start: time.Now().Add(-time.Hour), end: time.Now(), now: time.Now().Add(-2 * time.Hour), expErr: common.ErrStartAfterTimeNow},
		{name: "window starting just after the clock", start: ahead.Add(time.Nanosecond), end: ahead.Add(time.Hour), now: ahead, expErr: common.ErrStartAfterTimeNow},
		{name: "unset start", end: ahead, now: ahead, expErr: common.ErrDateUnset},
		{name: "unset end", start: ahead, now: ahead, expErr: common.ErrDateUnset},
		{name: "start after end", start: ahead.Add(-time.Hour), end: ahead.Add(-2 * time.Hour), now: ahead, expErr: common.ErrStartAfterEnd},
		{name: "start equal to end", start: ahead.Add(-time.Hour), end: ahead.Add(-time.Hour), now: ahead, expErr: common.ErrStartEqualsEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, startEndTimeCheckAt(tc.start, tc.end, tc.now), tc.expErr, "startEndTimeCheckAt should judge the window by the clock given")
		})
	}
}

func TestPositionHasDirection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		posSide   string
		size      types.Number
		direction order.Side
		exp       bool
	}{
		{positionSideLong, 2, order.UnknownSide, true},
		{positionSideLong, 2, order.Long, true},
		{positionSideLong, 2, order.Buy, true},
		{positionSideLong, 2, order.Short, false},
		{positionSideLong, 2, order.Sell, false},
		{positionSideLong, 2, order.AnySide, true},
		{positionSideShort, 3, order.UnknownSide, true},
		{positionSideShort, 3, order.Short, true},
		{positionSideShort, 3, order.Sell, true},
		{positionSideShort, 3, order.Long, false},
		{positionSideShort, 3, order.Buy, false},
		{positionSideNet, -1, order.Short, true},
		{positionSideNet, -1, order.Long, false},
		{positionSideNet, -1, order.UnknownSide, true},
		{positionSideNet, 4, order.Long, true},
		{positionSideNet, 4, order.Short, false},
		{"", 4, order.Long, true},
		{"", -4, order.Long, false},
	} {
		t.Run(fmt.Sprintf("posSide %q size %v for %s", tc.posSide, tc.size, tc.direction), func(t *testing.T) {
			t.Parallel()
			got := positionHasDirection(&AccountPositionResponse{PositionSide: tc.posSide, QuantityOfPosition: tc.size}, tc.direction)
			assert.Equal(t, tc.exp, got, "positionHasDirection should match a position on the side the direction names")
		})
	}
}

func TestValidContractValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value float64
		exp   bool
	}{
		{0.01, true},
		{100, true},
		{math.SmallestNonzeroFloat64, true},
		{math.MaxFloat64, true},
		{0, false},
		{-0.01, false},
		{math.NaN(), false},
		{math.Inf(1), false},
		{math.Inf(-1), false},
	} {
		t.Run(strconv.FormatFloat(tc.value, 'g', -1, 64), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, validContractValue(tc.value), "validContractValue should accept only a finite contract value above zero")
		})
	}
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	t.Run("signs a net mode short", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/account/positions":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "pos": "-1", "mgnMode": "cross", "ccy": "USDT"}})
			case "/account/balance":
				writeOKXData(t, w, []map[string]any{{"details": []map[string]string{{"ccy": "USDT", "eq": "100", "availEq": "100"}}}})
			case "/account/config":
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
			Asset: asset.PerpetualSwap, Pair: perpetualSwapPair,
		})
		require.NoError(t, err, "a net mode short must summarise rather than report no positions")
		assert.Equal(t, decimal.MustFromFloat(-0.01), summary.CurrentSize, "a short of one 0.01 contract should carry its negative size")
		assert.Equal(t, decimal.MustFromFloat(-1), summary.ContractSize, "the contract size should keep the signed position")
	})
	t.Run("signs a long/short mode short", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/account/positions":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "pos": "1", "posSide": positionSideShort, "mgnMode": "cross", "ccy": "USDT"}})
			case "/account/balance":
				writeOKXData(t, w, []map[string]any{{"details": []map[string]string{{"ccy": "USDT", "eq": "100", "availEq": "100"}}}})
			case "/account/config":
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeLongShort}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
			Asset: asset.PerpetualSwap, Pair: perpetualSwapPair,
		})
		require.NoError(t, err, "a long/short mode short must summarise rather than report no positions")
		assert.Equal(t, decimal.MustFromFloat(-0.01), summary.CurrentSize, "a posSide short of one 0.01 contract should read as a negative size")
		assert.Equal(t, decimal.MustFromFloat(-1), summary.ContractSize, "the contract size should carry the flipped short sign")
	})
	// The balance details document no total equity, so the summary takes the
	// account level total equity.
	t.Run("reports the account total equity", func(t *testing.T) {
		t.Parallel()
		e := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}})
			case "/account/positions":
				writeOKXData(t, w, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "pos": "1", "mgnMode": "cross", "ccy": "USDT"}})
			case "/account/balance":
				writeOKXData(t, w, []map[string]any{{"totalEq": "250.5", "details": []map[string]string{{"ccy": "USDT", "eq": "100", "eqUsd": "100.1"}}}})
			case "/account/config":
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeNet}})
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{
			Asset: asset.PerpetualSwap, Pair: perpetualSwapPair,
		})
		require.NoError(t, err, "GetFuturesPositionSummary must not error")
		assert.Equal(t, decimal.MustFromFloat(250.5), summary.TotalEquity, "the total equity should be the account level totalEq")
		assert.Equal(t, decimal.MustFromFloat(100.1), summary.EquityUSD, "the USD equity should come from the position currency's details")
	})

	linearSwap := map[string]string{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "uly": "BTC-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.01", "state": "live"}
	// newPositionSummaryExchange answers the summary's requests with the instrument list and positions given
	newPositionSummaryExchange := func(t *testing.T, instruments, positions []map[string]string) *Exchange {
		t.Helper()
		return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/instruments":
				writeOKXData(t, w, instruments)
			case "/account/positions":
				writeOKXData(t, w, positions)
			case "/account/balance":
				writeOKXData(t, w, []map[string]any{{"details": []map[string]string{{"ccy": "USDT", "eq": "100", "availBal": "100"}}}})
			case "/account/config":
				writeOKXData(t, w, []map[string]string{{"acctLv": "2", "posMode": positionModeLongShort}})
			default:
				assert.Failf(t, "unexpected request", "the summary should not reach %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
	}

	t.Run("selects the position on the requested side", func(t *testing.T) {
		t.Parallel()
		position := func(size, posSide, isolatedMargin string) map[string]string {
			return map[string]string{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "pos": size, "posSide": posSide, "mgnMode": TradeModeIsolated, "ccy": "USDT", "margin": isolatedMargin}
		}
		long, short := position("2", positionSideLong, "20"), position("3", positionSideShort, "30")
		netLong, netShort := position("4", positionSideNet, "40"), position("-1", positionSideNet, "10")
		dec := decimal.MustFromFloat
		for _, tc := range []struct {
			name             string
			positions        []map[string]string
			direction        order.Side
			expSize, expHeld decimal.Decimal
			expErr           error
		}{
			{name: "both sides without a direction", positions: []map[string]string{long, short}, expErr: errAmbiguousPosition},
			{name: "both sides in reverse without a direction", positions: []map[string]string{short, long}, expErr: errAmbiguousPosition},
			{name: "both sides for any side", positions: []map[string]string{long, short}, direction: order.AnySide, expErr: errAmbiguousPosition},
			{name: "both sides for a long", positions: []map[string]string{short, long}, direction: order.Long, expSize: dec(2), expHeld: dec(20)},
			{name: "both sides for a buy", positions: []map[string]string{short, long}, direction: order.Buy, expSize: dec(2), expHeld: dec(20)},
			{name: "both sides for a short", positions: []map[string]string{long, short}, direction: order.Short, expSize: dec(-3), expHeld: dec(30)},
			{name: "both sides for a sell", positions: []map[string]string{long, short}, direction: order.Sell, expSize: dec(-3), expHeld: dec(30)},
			{name: "a short alone for a long", positions: []map[string]string{short}, direction: order.Long, expErr: errOnlyOneResponseExpected},
			{name: "a net short without a direction", positions: []map[string]string{netShort}, expSize: dec(-1), expHeld: dec(10)},
			{name: "a net short for a short", positions: []map[string]string{netShort}, direction: order.Short, expSize: dec(-1), expHeld: dec(10)},
			{name: "a net short for a long", positions: []map[string]string{netShort}, direction: order.Long, expErr: errOnlyOneResponseExpected},
			{name: "a net long for a long", positions: []map[string]string{netLong}, direction: order.Long, expSize: dec(4), expHeld: dec(40)},
			{name: "a net long for a short", positions: []map[string]string{netLong}, direction: order.Short, expErr: errOnlyOneResponseExpected},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newPositionSummaryExchange(t, []map[string]string{linearSwap}, tc.positions)
				summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair, Direction: tc.direction})
				require.ErrorIs(t, err, tc.expErr, "GetFuturesPositionSummary must return the expected error")
				if tc.expErr != nil {
					return
				}
				assert.Equal(t, [2]decimal.Decimal{tc.expSize, tc.expHeld}, [2]decimal.Decimal{summary.ContractSize, summary.IsolatedMargin}, "GetFuturesPositionSummary should summarise the position on the requested side")
			})
		}
	})

	t.Run("requires the contract's value", func(t *testing.T) {
		t.Parallel()
		valued := func(contractValue string) []map[string]string {
			contract := maps.Clone(linearSwap)
			contract["ctVal"] = contractValue
			return []map[string]string{contract}
		}
		for _, tc := range []struct {
			name        string
			instruments []map[string]string
		}{
			{"contract missing from the instrument list", []map[string]string{{"instType": instTypeSwap, "instId": "ETH-USDT-SWAP", "uly": "ETH-USDT", "ctType": "linear", "settleCcy": "USDT", "ctVal": "0.1", "state": "live"}}},
			{"contract without a contract value", valued("")},
			{"contract valued at not a number", valued("NaN")},
			{"contract valued at infinity", valued("Inf")},
			{"contract valued below zero", valued("-0.01")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newPositionSummaryExchange(t, tc.instruments, []map[string]string{{"instType": instTypeSwap, "instId": "BTC-USDT-SWAP", "pos": "2", "posSide": positionSideNet, "mgnMode": TradeModeIsolated, "ccy": "USDT"}})
				_, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualSwap, Pair: perpetualSwapPair})
				assert.ErrorIs(t, err, futures.ErrContractNotSupported, "GetFuturesPositionSummary should refuse to size a position without its contract's value")
			})
		}
	})

	t.Run("refuses a spread", func(t *testing.T) {
		t.Parallel()
		ex := newPositionSummaryExchange(t, nil, nil)
		_, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spread, Pair: spreadPair})
		assert.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesPositionSummary should refuse a spread, whose positions are its legs'")
	})
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()
	t.Run("endpoint map", func(t *testing.T) {
		t.Parallel()
		defaults := ex.API.Endpoints.GetURLMap()
		require.Len(t, defaults, 4, "endpoint defaults must match the declared API scope")
		assert.Equal(t, "https://www.okx.com/api/v5/", defaults[exchange.RestSpot.String()], "REST default should target the v5 API")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/public", defaults[exchange.WebsocketSpot.String()], "public websocket default should use port 443")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/private", defaults[exchange.WebsocketPrivate.String()], "private websocket default should use port 443")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/business", defaults[exchange.WebsocketSpotSupplementary.String()], "business websocket default should use port 443")
		assert.Equal(t, "wss://ws.okx.com/ws/v5/", websocketURL, "websocket base should use port 443")
		for endpoint, url := range defaults {
			assert.NotContainsf(t, url, ":8443", "%s default should not use the retired port 8443", endpoint)
		}
	})
	for ep, want := range map[exchange.URL]string{
		exchange.WebsocketSpot:              "wss://ws.okx.com/ws/v5/public",
		exchange.WebsocketPrivate:           "wss://ws.okx.com/ws/v5/private",
		exchange.WebsocketSpotSupplementary: "wss://ws.okx.com/ws/v5/business",
	} {
		t.Run(ep.String(), func(t *testing.T) {
			t.Parallel()
			got, err := ex.API.Endpoints.GetURL(ep)
			require.NoErrorf(t, err, "GetURL must return the default %s endpoint", ep)
			assert.Equalf(t, want, got, "default %s endpoint should use port 443", ep)
		})
	}
}
