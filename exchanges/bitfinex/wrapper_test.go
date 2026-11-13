package bitfinex

import (
	"cmp"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// testWrapperOrderID is the order the wrapper's mock order tests act on, the documented Submit Order example's, and
// testDerivativeOrderID a derivative order following it
const (
	testWrapperOrderID    = "1747566428"
	testDerivativeOrderID = "1747566430"
)

// testDerivativeNextFunding is the next funding event of the derivatives' recorded status
var testDerivativeNextFunding = time.UnixMilli(1792368000000)

// testHistoryStart starts the window the mock history tests ask for, which their fixtures were recorded with
var testHistoryStart = time.UnixMilli(1792022400000).UTC()

// newWrapperTestExchange returns an exchange named after the test, trading BTCUSD and ETHUSD on spot, BTCUSD on
// margin, funding USD, and trading the perpetual contracts BTCF0:USTF0, margined by USDT, and ETHF0:BTCF0, by bitcoin
func newWrapperTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newTestExchange(t)
	for a, pairs := range map[asset.Item]currency.Pairs{
		asset.Spot:                {currency.NewBTCUSD(), testETHUSD},
		asset.Margin:              {currency.NewBTCUSD()},
		asset.MarginFunding:       {testFundingUSD},
		asset.USDTMarginedFutures: {testDerivativePair},
		asset.CoinMarginedFutures: {testCoinMarginedPair},
	} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for %s available pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for %s enabled pairs", a)
	}
	return ex
}

// historyWindow returns the window a history test asks for: the mock's recorded one, or the last hour live
func historyWindow(d time.Duration) (start, end time.Time) {
	if mockTests {
		return testHistoryStart, testHistoryStart.Add(d)
	}
	end = time.Now().UTC().Truncate(time.Minute)
	return end.Add(-d), end
}

// testOrderRecord is an order a test server holds, which it sends as Bitfinex sends an order
type testOrderRecord struct {
	id                                      uint64
	symbol, orderType, status, meta         string
	amount, original, price, auxiliaryLimit float64
	// createdAt and updatedAt are the order's times as sent, 1678987199446 and 1678987199448 when not given
	createdAt, updatedAt string
}

// String returns the order as Bitfinex sends it, its meta data null when it has none
func (o *testOrderRecord) String() string {
	meta := o.meta
	if meta == "" {
		meta = "null"
	}
	createdAt, updatedAt := cmp.Or(o.createdAt, "1678987199446"), cmp.Or(o.updatedAt, "1678987199448")
	number := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	return `[` + strconv.FormatUint(o.id, 10) + `,null,null,"` + o.symbol + `",` + createdAt + `,` + updatedAt + `,` + number(o.amount) + `,` + number(o.original) +
		`,"` + o.orderType + `",null,null,null,0,"` + o.status + `",null,null,` + number(o.price) + `,0,0,` + number(o.auxiliaryLimit) +
		`,null,null,null,0,0,null,null,null,"API>BFX",null,null,` + meta + `]`
}

// testOrderRecords returns orders as a list Bitfinex sends
func testOrderRecords(orders []testOrderRecord) string {
	records := make([]string, len(orders))
	for i := range orders {
		records[i] = orders[i].String()
	}
	return `[` + strings.Join(records, ",") + `]`
}

// newOrderUpdateTestExchange returns an exchange whose open orders are those given, which it looks up by ID, and the
// order updates it sends, decoded, in the order sent
func newOrderUpdateTestExchange(t *testing.T, open ...testOrderRecord) (ex *Exchange, updates func() []map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var sent []map[string]any
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "Decoding the request should not error") {
			http.Error(w, "undecodable", http.StatusBadRequest)
			return
		}
		body := `[]`
		switch r.URL.Path {
		case "/v2/auth/r/orders":
			ids, ok := req["id"].([]any)
			if !assert.True(t, ok && len(ids) == 1, "an order lookup should name one order") {
				http.Error(w, "no order named", http.StatusBadRequest)
				return
			}
			for i := range open {
				if ids[0] == float64(open[i].id) {
					body = `[` + open[i].String() + `]`
				}
			}
		case "/v2/auth/w/order/update":
			mu.Lock()
			sent = append(sent, req)
			mu.Unlock()
			body = `[1678988263842,"ou-req",null,null,[1747566428],null,"SUCCESS","Submitting update"]`
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
			assert.Failf(t, "the test server should receive only the requests expected", "%s %s", r.Method, r.URL.Path)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	return ex, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(sent)
	}
}

func TestFetchTradablePairs(t *testing.T) { //nolint:tparallel // The cases share e in order
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must refuse an asset Bitfinex does not trade")
	derivative := func(base string) currency.Pair {
		return currency.NewPair(currency.NewCode(base), currency.NewCode("USTF0"))
	}
	// The cases share e in order, so each also checks that a call keeps nothing from the last
	for a, exp := range map[asset.Item]currency.Pairs{
		asset.Spot: {
			currency.NewPair(currency.AAVE, currency.USD), currency.NewPair(currency.AAVE, currency.UST), currency.NewPair(currency.ADA, currency.BTC),
			currency.NewPair(currency.ADA, currency.USD), currency.NewPair(currency.ADA, currency.UST),
		},
		asset.Margin: {
			currency.NewPair(currency.ADA, currency.BTC), currency.NewPair(currency.ADA, currency.USD), currency.NewPair(currency.ADA, currency.UST),
			currency.NewPair(currency.NewCode("ALG"), currency.USD), currency.NewPair(currency.APT, currency.USD),
		},
		asset.MarginFunding:       {testFundingUSD, {Base: currency.BTC}},
		asset.USDTMarginedFutures: {derivative("AAVEF0"), derivative("ADAF0"), derivative("ALGF0"), derivative("ALLOF0"), derivative("APEF0"), testDerivativePair},
		asset.CoinMarginedFutures: {testCoinMarginedPair},
	} {
		t.Run(a.String(), func(t *testing.T) {
			pairs, err := e.FetchTradablePairs(t.Context(), a)
			require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
			require.NotEmptyf(t, pairs, "FetchTradablePairs must return %s pairs", a)
			for _, p := range pairs {
				assert.NotContainsf(t, p.String(), testListingMarker, "FetchTradablePairs should leave out %s test pairs", a)
				if isDerivativeAsset(a) {
					assert.Equalf(t, a, derivativeAsset(p.Quote), "FetchTradablePairs should return only the derivative pairs %s margins, got %s", a, p)
				}
			}
			if mockTests {
				assert.Truef(t, exp.Equal(pairs), "FetchTradablePairs should return the %s pairs Bitfinex lists, got %s", a, pairs)
				return
			}
			want := map[asset.Item]currency.Pair{
				asset.MarginFunding:       testFundingUSD,
				asset.USDTMarginedFutures: testDerivativePair,
				asset.CoinMarginedFutures: testCoinMarginedPair,
			}[a]
			if want.IsEmpty() {
				want = currency.NewBTCUSD()
			}
			assert.Truef(t, pairs.Contains(want, true), "FetchTradablePairs should return %s for %s", want, a)
		})
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	for _, a := range ex.GetAssetTypes(false) {
		available, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		assert.NotEmptyf(t, available, "UpdateTradablePairs should store %s's available pairs", a)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) { //nolint:tparallel // The cases share ex in order
	t.Parallel()
	ex := newWrapperTestExchange(t)
	assert.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.MarginFunding), common.ErrNotYetImplemented, "UpdateOrderExecutionLimits should not load funding limits")
	assert.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateOrderExecutionLimits should refuse an asset Bitfinex does not trade")
	var apiErr *APIError
	assert.ErrorAs(t, newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).UpdateOrderExecutionLimits(t.Context(), asset.Spot), &apiErr, "UpdateOrderExecutionLimits should return the error Bitfinex reports")
	unparsable := newPathResponseServerExchange(t, map[string]string{"/v2/conf/pub:info:pair": `[[["BTC",[1596710520048,null,null,"4.0","250000.0",null,null,null,0.3,0.15,null,null]]]]`})
	assert.ErrorIs(t, unparsable.UpdateOrderExecutionLimits(t.Context(), asset.Spot), errSymbolUnparsable, "UpdateOrderExecutionLimits should refuse a pair it cannot read")
	p := currency.NewBTCUSD()
	if mockTests {
		p = currency.NewPair(currency.ADA, currency.USD)
	}
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run(a.String(), func(t *testing.T) {
			require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
			l, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, a, p))
			require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", a, p)
			if mockTests {
				assert.Equalf(t, 4.0, l.MinimumBaseAmount, "the %s minimum order should be Bitfinex's", a)
				assert.Equalf(t, 250000.0, l.MaximumBaseAmount, "the %s maximum order should be Bitfinex's", a)
				return
			}
			assert.Positivef(t, l.MinimumBaseAmount, "the %s minimum order should be set", a)
			assert.Greaterf(t, l.MaximumBaseAmount, l.MinimumBaseAmount, "the %s maximum order should exceed the minimum", a)
		})
	}
	// The cases share ex in order after those above, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		a             asset.Item
		p, other      currency.Pair
		minimum, most float64
	}{
		{a: asset.USDTMarginedFutures, p: testDerivativePair, other: testCoinMarginedPair, minimum: 0.00004, most: 100},
		{a: asset.CoinMarginedFutures, p: testCoinMarginedPair, other: testDerivativePair, minimum: 0.0008, most: 100},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.a), "UpdateOrderExecutionLimits must not error for %s", tc.a)
			l, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, tc.a, tc.p))
			require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", tc.a, tc.p)
			_, err = limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, tc.a, tc.other))
			assert.ErrorIsf(t, err, limits.ErrOrderLimitNotFound, "%s should hold no limits for %s, which another asset trades", tc.a, tc.other)
			if mockTests {
				assert.Equalf(t, tc.minimum, l.MinimumBaseAmount, "the %s minimum order should be Bitfinex's", tc.a)
				assert.Equalf(t, tc.most, l.MaximumBaseAmount, "the %s maximum order should be Bitfinex's", tc.a)
				return
			}
			assert.Positivef(t, l.MinimumBaseAmount, "the %s minimum order should be set", tc.a)
			assert.Greaterf(t, l.MaximumBaseAmount, l.MinimumBaseAmount, "the %s maximum order should exceed the minimum", tc.a)
		})
	}
}

func TestUpdateTickers(t *testing.T) { //nolint:tparallel // The cases share ex in order
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers must refuse an asset Bitfinex does not trade")
	// The cases share ex in order, and the check after them reads the ticker the spot case stores
	for a, pairs := range map[asset.Item]currency.Pairs{
		asset.Spot:                {currency.NewBTCUSD(), testETHUSD},
		asset.Margin:              {currency.NewBTCUSD()},
		asset.MarginFunding:       {testFundingUSD},
		asset.USDTMarginedFutures: {testDerivativePair},
		asset.CoinMarginedFutures: {testCoinMarginedPair},
	} {
		t.Run(a.String(), func(t *testing.T) {
			before := time.Now()
			require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
			after := time.Now()
			for _, p := range pairs {
				got, err := ticker.GetTicker(ex.Name, p, a)
				require.NoErrorf(t, err, "UpdateTickers must store %s %s's ticker", a, p)
				assert.Positivef(t, got.Last, "the %s %s ticker should have a last price", a, p)
				assert.WithinRangef(t, got.LastUpdated, before, after, "the %s %s ticker should be stamped when received, not with its first trade", a, p)
			}
		})
	}
	if mockTests {
		got, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
		require.NoError(t, err, "GetTicker must not error")
		got.LastUpdated = time.Time{}
		dailyChangeRelative := 0.01367597
		assert.Equal(t, ticker.Price{
			Last: 82126, High: 82641, Low: 79821, Bid: 82109, BidSize: 1.49200629, Ask: 82121, AskSize: 1.90605581, BaseVolume: 505.79793186,
			Open24Hour: 82126 - 1108, PercentChange24Hour: dailyChangeRelative * 100, Pair: currency.NewBTCUSD(), ExchangeName: ex.Name, AssetType: asset.Spot,
		}, *got, "the stored spot ticker should be Bitfinex's, its open the last price less the daily change")
	}
}

func TestUpdateTicker(t *testing.T) { //nolint:tparallel // The cases share ex in order
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must refuse an empty pair")
	_, err = ex.UpdateTicker(t.Context(), currency.NewBTCUSD(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must refuse an asset Bitfinex has no symbol for")
	_, err = ex.UpdateTicker(t.Context(), currency.NewBTCUSD(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errPairNotOfAsset, "UpdateTicker must refuse a pair the asset does not trade")
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		t.Run(a.String(), func(t *testing.T) {
			got, err := ex.UpdateTicker(t.Context(), p, a)
			require.NoErrorf(t, err, "UpdateTicker must not error for %s", a)
			assert.Positivef(t, got.Last, "the %s ticker should have a last price", a)
			assert.Equalf(t, a, got.AssetType, "the %s ticker should be stored for its asset", a)
		})
	}
}

func TestTickerPrice(t *testing.T) {
	t.Parallel()
	got := e.tickerPrice(&TickerResponse{
		Bid: 1, BidSize: 2, Ask: 3, AskSize: 4, DailyChange: 5, DailyChangeRelative: 0.25, LastPrice: 107, Volume: 8, High: 109, Low: 100,
		FirstTradeTime: types.Time(time.UnixMilli(1358182043000)),
	}, currency.NewBTCUSD(), asset.Spot)
	assert.Equal(t, &ticker.Price{
		ExchangeName: e.Name, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Bid: 1, BidSize: 2, Ask: 3, AskSize: 4, Last: 107, BaseVolume: 8, High: 109,
		Low: 100, Open24Hour: 102, PercentChange24Hour: 25,
	}, got, "tickerPrice should convert the ticker, its open the last price less the daily change, and leave its first trade time out")
}

func TestFundingTickerPrice(t *testing.T) {
	t.Parallel()
	got := e.fundingTickerPrice(&FundingTickerResponse{
		FlashReturnRate: 0.0002, Bid: 0.0001, BidPeriod: 30, BidSize: 1000, Ask: 0.0003, AskPeriod: 2, AskSize: 500, DailyChange: 0.00001,
		DailyChangeRelative: 0.05, LastRate: 0.00025, Volume: 9, High: 0.0004, Low: 0.00005, FlashReturnRateAmountAvailable: 7,
		FirstTradeTime: types.Time(time.UnixMilli(1358182043000)),
	}, testFundingUSD, asset.MarginFunding)
	assert.Equal(t, &ticker.Price{
		ExchangeName: e.Name, Pair: testFundingUSD, AssetType: asset.MarginFunding, FlashReturnRate: 0.0002, Bid: 0.0001, BidPeriod: 30, BidSize: 1000,
		Ask: 0.0003, AskPeriod: 2, AskSize: 500, Last: 0.00025, BaseVolume: 9, High: 0.0004, Low: 0.00005, FlashReturnRateAmount: 7,
	}, got, "fundingTickerPrice should convert the funding ticker, its rates as prices")
}

func TestUpdateOrderbook(t *testing.T) { //nolint:tparallel // The cases share ex in order
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must refuse an empty pair")
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		t.Run(a.String(), func(t *testing.T) {
			book, err := ex.UpdateOrderbook(t.Context(), p, a)
			require.NoErrorf(t, err, "UpdateOrderbook must not error for %s", a)
			require.NotEmptyf(t, book.Bids, "the %s book must hold bids", a)
			require.NotEmptyf(t, book.Asks, "the %s book must hold asks", a)
			assert.Equalf(t, a == asset.MarginFunding, book.IsFundingRate, "the %s book should be marked a funding book only for funding", a)
			assert.NotZerof(t, book.Bids[0].ID, "the %s book should hold orders by ID", a)
			if a == asset.MarginFunding {
				assert.NotZero(t, book.Bids[0].Period, "the funding book should hold each offer's period")
			}
		})
	}
}

func TestUpdateAccountBalances(t *testing.T) { //nolint:tparallel // The cases share ex in order
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must refuse an asset Bitfinex has no wallet for")
	_, err = newRefusingServerExchange(t, `["error",10100,"apikey: invalid"]`).UpdateAccountBalances(t.Context(), asset.Spot)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "UpdateAccountBalances should return the error Bitfinex reports")
	unsaved := newResponseServerExchange(t, `[["exchange","BTC",1,0,1]]`)
	unsaved.Accounts.Exchange = failedCredentialLookup{Exchange: unsaved}
	_, err = unsaved.UpdateAccountBalances(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, errCredentialLookup, "UpdateAccountBalances should return the error saving the balances")
	skipLiveWithoutCredentials(t)
	ex := newWrapperTestExchange(t)
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.MarginFunding, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			subAccts, err := ex.UpdateAccountBalances(t.Context(), a)
			require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
			require.Lenf(t, subAccts, 1, "UpdateAccountBalances must return %s's wallet", a)
			assert.Equalf(t, a, subAccts[0].AssetType, "UpdateAccountBalances should return the %s wallet", a)
			if !mockTests {
				return
			}
			switch a {
			case asset.Spot:
				assert.Equal(t, accounts.CurrencyBalances{currency.UST: {Currency: currency.UST, Total: 19788.6529257, Free: 19788.6529257, UpdatedAt: subAccts[0].Balances[currency.UST].UpdatedAt}}, subAccts[0].Balances, "the exchange wallet should hold its balances")
			case asset.Margin:
				assert.Empty(t, subAccts[0].Balances, "the margin wallet should hold nothing beside the derivatives' collateral")
			case asset.MarginFunding:
				b := subAccts[0].Balances[currency.BTC]
				assert.Equal(t, 0.5, b.Total, "the funding wallet should hold its balance")
				assert.Zero(t, b.Free, "a balance whose available amount Bitfinex has not calculated should not be free")
				assert.Equal(t, 0.5, b.Hold, "a balance whose available amount Bitfinex has not calculated should be held")
			case asset.USDTMarginedFutures:
				c := testDerivativePair.Quote
				assert.Equal(t, accounts.CurrencyBalances{c: {Currency: c, Total: 500, Free: 300, Hold: 200, UpdatedAt: subAccts[0].Balances[c].UpdatedAt}}, subAccts[0].Balances, "the derivatives wallet should fund USDT margined futures with its USTF0")
			case asset.CoinMarginedFutures:
				c := testCoinMarginedPair.Quote
				assert.Equal(t, accounts.CurrencyBalances{c: {Currency: c, Total: 0.01, Free: 0.01, UpdatedAt: subAccts[0].Balances[c].UpdatedAt}}, subAccts[0].Balances, "the derivatives wallet should fund coin margined futures with its BTCF0")
			}
		})
	}
}

func TestUpdateAccountBalancesStoresEveryWallet(t *testing.T) { //nolint:tparallel // The cases read ex's account store in order
	t.Parallel()
	replies := []string{
		// Led by a wallet of a type Bitfinex may yet add, which funds no asset GCT knows, with a zero balance, a negative
		// available balance, and interest the funding wallet has yet to settle, which is no part of its balance
		`[["novel","XRP",4,0,4],["exchange","BTC",1,0,1],["exchange","ETH",0,0,0],["margin","ETH",2,0,2],["margin","UST",1,0,-0.5],["funding","USD",3,0.5,3],["margin","USTF0",500,0,300],["margin","BTCF0",0.01,0,0.01]]`,
		// The ETH has since moved from the margin wallet to the exchange wallet, beside the same unknown wallet
		`[["exchange","BTC",1,0,1],["exchange","ETH",2,0,2],["novel","XRP",4,0,4],["funding","USD",3,0,3],["margin","USTF0",500,0,300],["margin","BTCF0",0.01,0,0.01]]`,
		// The exchange wallet's BTC has since grown, behind a wallet of each known type without a currency
		`[["exchange","",1,0,1],["margin","",1,0,1],["funding","",1,0,1],["exchange","BTC",5,0,5]]`,
		// And every wallet has since been emptied
		`[]`,
	}
	var requests int
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(replies[min(requests, len(replies)-1)]))
		requests++
		assert.NoError(t, err, "Writing the response should not error")
	}))
	counter := &credentialLookupCounter{Exchange: ex}
	ex.Accounts.Exchange = counter
	creds := &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret}
	// The engine's portfolio manager updates spot alone
	subAccts, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, subAccts, 1, "UpdateAccountBalances must return one wallet")
	assert.Equal(t, asset.Spot, subAccts[0].AssetType, "UpdateAccountBalances should return the spot wallet")
	assert.Equal(t, 1, requests, "UpdateAccountBalances should read the wallets once")
	assert.Equal(t, 1, counter.lookups, "UpdateAccountBalances should store every wallet in one save")
	stored, err := ex.Accounts.SubAccounts(creds, asset.Spot)
	require.NoError(t, err, "SubAccounts must not error")
	assert.Equal(t, stored, subAccts, "UpdateAccountBalances should return the spot wallet as stored")
	assertWalletBalances(t, ex, map[asset.Item]map[currency.Code][3]float64{
		asset.Spot:                {currency.BTC: {1, 1, 0}, currency.ETH: {0, 0, 0}},
		asset.Margin:              {currency.ETH: {2, 2, 0}, currency.UST: {1, -0.5, 1.5}},
		asset.MarginFunding:       {currency.USD: {3, 3, 0}},
		asset.USDTMarginedFutures: {testDerivativePair.Quote: {500, 300, 200}},
		asset.CoinMarginedFutures: {testCoinMarginedPair.Quote: {0.01, 0.01, 0}},
	})
	for _, a := range walletAssets {
		_, err = ex.Accounts.GetBalance("", creds, a, currency.XRP)
		assert.ErrorIsf(t, err, accounts.ErrNoBalances, "UpdateAccountBalances should store no %s balance from a wallet of an unknown type", a)
	}

	_, err = ex.UpdateAccountBalances(t.Context(), asset.MarginFunding)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	assert.Equal(t, 2, requests, "UpdateAccountBalances should read the wallets once an update")
	assert.Equal(t, 2, counter.lookups, "UpdateAccountBalances should store every wallet in one save an update")
	assertWalletBalances(t, ex, map[asset.Item]map[currency.Code][3]float64{asset.Spot: {currency.BTC: {1, 1, 0}, currency.ETH: {2, 2, 0}}})
	_, err = ex.Accounts.GetBalance("", creds, asset.Margin, currency.ETH)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "UpdateAccountBalances should remove a balance a wallet no longer holds, whichever asset it updates")

	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateAccountBalances should report a wallet without a currency")
	// The read cannot be taken to list every balance, so it removes none
	assertWalletBalances(t, ex, map[asset.Item]map[currency.Code][3]float64{
		asset.Spot:          {currency.BTC: {5, 5, 0}, currency.ETH: {2, 2, 0}},
		asset.MarginFunding: {currency.USD: {3, 3, 0}},
	})
	_, err = ex.Accounts.CurrencyBalances(nil, asset.All)
	assert.NoError(t, err, "the stored balances should collate, as UpdateAccountBalances should store none without a currency")

	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	// The cases read ex's account store and assign the test's err, as the checks above do, so they run in order
	for a, c := range map[asset.Item]currency.Code{asset.Spot: currency.BTC, asset.MarginFunding: currency.USD, asset.USDTMarginedFutures: testDerivativePair.Quote} {
		t.Run(a.String(), func(t *testing.T) {
			_, err = ex.Accounts.GetBalance("", creds, a, c)
			assert.ErrorIsf(t, err, accounts.ErrNoBalances, "UpdateAccountBalances should remove the %s %s balance once no wallet is listed", a, c)
		})
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	assert.NoError(t, newWrapperTestExchange(t).ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := newRefusingServerExchange(t, `["error",10100,"apikey: invalid"]`).GetAccountFundingHistory(t.Context())
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "GetAccountFundingHistory should return the error Bitfinex reports")
	skipLiveWithoutCredentials(t)
	history, err := e.GetAccountFundingHistory(mockRequestContext(t))
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		return
	}
	require.Len(t, history, 2, "GetAccountFundingHistory must return every movement")
	assert.Equal(t, exchange.FundingHistory{
		ExchangeName: e.Name, Status: "COMPLETED", TransferID: "13105603", Timestamp: time.UnixMilli(1569348774000), Currency: "ETH", Amount: 0.26300954,
		Fee: 0.00135, TransferType: "withdrawal", CryptoToAddress: "DESTINATION_ADDRESS", CryptoTxID: "TRANSACTION_ID", CryptoChain: "ethereum",
	}, history[0], "a withdrawal should be reported with its positive amount and fee, and the method it moved over")
	assert.Equal(t, "deposit", history[1].TransferType, "a positive movement should be a deposit")
}

func TestGetWithdrawalsHistory(t *testing.T) { //nolint:tparallel // The cases share e in order
	t.Parallel()
	_, err := newRefusingServerExchange(t, `["error",10100,"apikey: invalid"]`).GetWithdrawalsHistory(t.Context(), currency.ETH, asset.Spot)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "GetWithdrawalsHistory should return the error Bitfinex reports")
	skipLiveWithoutCredentials(t)
	ctx := mockRequestContext(t)
	// The cases share e and ctx in order, so each also checks that a call keeps nothing from the last
	for _, c := range []currency.Code{currency.ETH, currency.EMPTYCODE} {
		t.Run(subtestName(c.String()), func(t *testing.T) {
			history, err := e.GetWithdrawalsHistory(ctx, c, asset.Spot)
			require.NoErrorf(t, err, "GetWithdrawalsHistory must not error for currency %q", c)
			for i := range history {
				assert.Equal(t, "withdrawal", history[i].TransferType, "GetWithdrawalsHistory should return withdrawals alone")
			}
			if mockTests {
				require.Lenf(t, history, 1, "GetWithdrawalsHistory must leave out the deposit for currency %q", c)
				assert.Equal(t, 0.26300954, history[0].Amount, "a withdrawal's amount should be positive")
			}
		})
	}
}

func TestGetMovementsHistory(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var paths []string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		_, err := w.Write([]byte(`[[13293039,"ETH","ETHEREUM",null,null,1574175052000,1574181326000,null,null,"COMPLETED",null,null,"-0.24","-0.00135",null,null,"DESTINATION_ADDRESS",null,null,null,"TRANSACTION_ID",null]]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	// The cases share ex, and each checks the last path it recorded, so they run in order
	for c, path := range map[currency.Code]string{currency.EMPTYCODE: "/v2/auth/r/movements/hist", currency.ETH: "/v2/auth/r/movements/ETH/hist"} {
		t.Run(subtestName(c.String()), func(t *testing.T) {
			got, err := ex.getMovementsHistory(t.Context(), c)
			require.NoErrorf(t, err, "getMovementsHistory must not error for currency %q", c)
			require.Lenf(t, got, 1, "getMovementsHistory must return the movement for currency %q", c)
			assert.Equalf(t, uint64(13293039), got[0].ID, "getMovementsHistory should return the movement for currency %q", c)
			mu.Lock()
			assert.Equalf(t, path, paths[len(paths)-1], "getMovementsHistory should read the movements of currency %q", c)
			mu.Unlock()
		})
	}
	_, err := newRefusingServerExchange(t, `["error",10020,"symbol: invalid"]`).getMovementsHistory(t.Context(), currency.ETH)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "getMovementsHistory should return the error Bitfinex reports")

	// Deposits and withdrawals alternate, served newest first, two to a millisecond so that a page can end inside one.
	// Those milliseconds are an hour apart and each movement started 10 minutes before its last update, by which the
	// pager steps back, so a bound taken from when a movement started leaves older ones out. A request gets those last
	// updated within its start and end, at most its limit or 25 without one, and a limit past the 1,000 Bitfinex
	// documents is refused, as Bitfinex refuses one past a public history's maximum
	for held, exp := range map[int][]uint64{999: {1000}, 1002: {1000, 1000}} {
		t.Run(strconv.Itoa(held)+" movements", func(t *testing.T) {
			t.Parallel()
			var requestLimits []uint64
			paged := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Start int64  `json:"start"`
					End   int64  `json:"end"`
					Limit uint64 `json:"limit"`
				}
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "Decoding the request should not error") {
					http.Error(w, "undecodable", http.StatusBadRequest)
					return
				}
				mu.Lock()
				requestLimits = append(requestLimits, req.Limit)
				mu.Unlock()
				reply := `["error",10020,"limit: invalid"]`
				if req.Limit > 1000 {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					rows := make([]string, 0, cmp.Or(req.Limit, 25))
					for id := held; id > 0 && len(rows) < cap(rows); id-- {
						if at := 1574181326000 + int64(id/2)*3600000; at >= req.Start && (req.End == 0 || at <= req.End) {
							amount, fee, address := "-0.24", "-0.00135", "DESTINATION_ADDRESS"
							if id%2 == 0 {
								amount, fee, address = "0.24", "0", "DEPOSIT_ADDRESS"
							}
							started, updated := strconv.FormatInt(at-600000, 10), strconv.FormatInt(at, 10)
							rows = append(rows, `[`+strconv.Itoa(id)+`,"ETH","ETHEREUM",null,null,`+started+`,`+updated+`,null,null,"COMPLETED",null,null,"`+amount+`","`+fee+`",null,null,"`+address+`",null,null,null,"TRANSACTION_ID",null]`)
						}
					}
					reply = `[` + strings.Join(rows, ",") + `]`
				}
				_, err := w.Write([]byte(reply))
				assert.NoError(t, err, "Writing the response should not error")
			}))
			got, err := paged.getMovementsHistory(t.Context(), currency.ETH)
			require.NoErrorf(t, err, "getMovementsHistory must not error for %d movements", held)
			assert.Lenf(t, got, held, "getMovementsHistory should return %d movements", held)
			mu.Lock()
			assert.Equalf(t, exp, requestLimits, "getMovementsHistory should ask for 1,000 movements a page, the most Bitfinex documents, for %d movements", held)
			mu.Unlock()
		})
	}
}

func TestMovementChain(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tetheruse", movementChain(&MovementResponse{Method: "TETHERUSE"}), "movementChain should give the method in lower case, as GetAvailableTransferChains does")
}

func TestMovementType(t *testing.T) {
	t.Parallel()
	for amount, exp := range map[types.Number]string{-0.24: "withdrawal", 0.26: "deposit"} {
		t.Run(exp, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, movementType(&MovementResponse{Amount: amount}), "movementType should call a movement of %v a %s", amount, exp)
		})
	}
}

func TestPageBack(t *testing.T) {
	t.Parallel()
	type record struct {
		id uint64
		at time.Time
	}
	ms := time.UnixMilli
	id := func(r *record) uint64 { return r.id }
	at := func(r *record) time.Time { return r.at }
	// errPagerStalled refuses a page past one per record, which only a pager no longer advancing asks for, so it fails
	// the test rather than hangs it
	errPagerStalled := errors.New("pager stalled")
	// serve returns a fetch serving history as Bitfinex does, newest first, records up to the end and from the start
	serve := func(history []record, start time.Time, limit int, ends *[]time.Time) func(time.Time) ([]record, error) {
		var pages int
		return func(end time.Time) ([]record, error) {
			pages++
			if pages > len(history)+1 {
				return nil, errPagerStalled
			}
			*ends = append(*ends, end)
			var page []record
			for _, r := range history {
				if (end.IsZero() || !r.at.After(end)) && !r.at.Before(start) && len(page) < limit {
					page = append(page, r)
				}
			}
			return page, nil
		}
	}
	history := []record{{9, ms(50)}, {8, ms(40)}, {7, ms(40)}, {6, ms(30)}, {5, ms(30)}, {4, ms(20)}, {3, ms(10)}}
	var ends []time.Time
	got, err := pageBack(time.Time{}, time.Time{}, 3, serve(history, time.Time{}, 3, &ends), id, at)
	require.NoError(t, err, "pageBack must not error")
	assert.Equal(t, history, got, "pageBack should return each record once, newest first")
	assert.Equal(t, []time.Time{{}, ms(40), ms(30), ms(20)}, ends, "each page should end at its predecessor's oldest record")

	ends = nil
	got, err = pageBack(ms(30), time.Time{}, 3, serve(history, ms(30), 3, &ends), id, at)
	require.NoError(t, err, "pageBack must not error")
	assert.Equal(t, history[:5], got, "pageBack should return the records from the start")
	assert.Equal(t, []time.Time{{}, ms(40), ms(30)}, ends, "pageBack should page no further than the start")

	ends = nil
	got, err = pageBack(ms(35), time.Time{}, 3, serve(history, time.Time{}, 3, &ends), id, at)
	require.NoError(t, err, "pageBack must not error")
	assert.Equal(t, history[:4], got, "pageBack should stop at a page reaching before the start")
	assert.Equal(t, []time.Time{{}, ms(40)}, ends, "pageBack should not fetch past a page reaching before the start")

	crowded := []record{{4, ms(20)}, {3, ms(20)}, {2, ms(20)}, {1, ms(20)}, {0, ms(10)}}
	_, err = pageBack(time.Time{}, time.Time{}, 3, serve(crowded, time.Time{}, 3, &ends), id, at)
	assert.ErrorIs(t, err, errHistoryPageFull, "pageBack should error for a millisecond holding a page of records")

	var calls int
	ignoringEnd := func(time.Time) ([]record, error) {
		calls++
		if calls > len(history) {
			return nil, errPagerStalled
		}
		return history[:3], nil
	}
	_, err = pageBack(time.Time{}, time.Time{}, 3, ignoringEnd, id, at)
	assert.ErrorIs(t, err, errHistoryPageFull, "pageBack should error for a server ignoring the end")
	assert.Equal(t, 2, calls, "pageBack should stop paging a server ignoring the end at its second page")

	errPageRefused := errors.New("page refused")
	var refusals int
	// Pages after the refusal are empty, so a pager retrying it returns without the error rather than hangs
	refusedOnce := func(time.Time) ([]record, error) {
		refusals++
		if refusals > 1 {
			return nil, nil
		}
		return nil, errPageRefused
	}
	_, err = pageBack(time.Time{}, time.Time{}, 3, refusedOnce, id, at)
	assert.ErrorIs(t, err, errPageRefused, "pageBack should return a page's error")
	assert.Equal(t, 1, refusals, "pageBack should not fetch again after a page is refused")
}

func TestGetRecentTrades(t *testing.T) { //nolint:tparallel // The cases share ex in order
	t.Parallel()
	ex := newWrapperTestExchange(t)
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		t.Run(a.String(), func(t *testing.T) {
			trades, err := ex.GetRecentTrades(t.Context(), p, a)
			require.NoErrorf(t, err, "GetRecentTrades must not error for %s", a)
			require.NotEmptyf(t, trades, "GetRecentTrades must return %s trades", a)
			assert.Falsef(t, trades[0].Timestamp.After(trades[len(trades)-1].Timestamp), "GetRecentTrades should return %s trades oldest first", a)
			for i := range trades {
				assert.Positivef(t, trades[i].Amount, "a %s trade's amount should be positive, its side giving its sign", a)
			}
		})
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := historyWindow(time.Hour)
	_, err := ex.GetHistoricTrades(t.Context(), currency.NewBTCUSD(), asset.Spot, end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades must refuse a window ending before it starts")
	trades, err := ex.GetHistoricTrades(mockRequestContext(t), currency.NewBTCUSD(), asset.Spot, start, end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	require.NotEmpty(t, trades, "GetHistoricTrades must return trades")
	for i := range trades {
		assert.Falsef(t, trades[i].Timestamp.Before(start) || trades[i].Timestamp.After(end), "trade %s should be within the window", trades[i].TID)
	}
}

func TestPlaceOrderRequest(t *testing.T) {
	t.Parallel()
	t.Run("margin type and quote amount refusals", func(t *testing.T) {
		t.Parallel()
		pairs := map[asset.Item]currency.Pair{
			asset.Spot:                currency.NewBTCUSD(),
			asset.Margin:              currency.NewBTCUSD(),
			asset.USDTMarginedFutures: testDerivativePair,
			asset.CoinMarginedFutures: testCoinMarginedPair,
		}
		// Every order type placeOrderRequest takes, and every name for a side, as neither refusal depends on them
		orders := []order.Submit{
			{Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100},
			{Type: order.Market, Side: order.Sell, Amount: 1},
			{Type: order.Stop, Side: order.Bid, Amount: 1, TriggerPrice: 100},
			{Type: order.StopLimit, Side: order.Ask, Amount: 1, Price: 90, TriggerPrice: 95},
			{Type: order.TrailingStop, Side: order.Long, Amount: 1, TrackingValue: 10},
			{Type: order.Limit, Side: order.Short, Amount: 1, Price: 100},
		}
		// The cases and those within them run in order, as placeOrderRequest is called again after them
		for _, tc := range []struct {
			a       asset.Item
			applied margin.Type
		}{
			{a: asset.Spot, applied: margin.NoMargin},
			{a: asset.Margin, applied: margin.Multi},
			{a: asset.USDTMarginedFutures, applied: margin.Isolated},
			{a: asset.CoinMarginedFutures, applied: margin.Isolated},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				for _, s := range orders {
					t.Run(s.Side.String()+" "+s.Type.String(), func(t *testing.T) {
						s.AssetType, s.Pair = tc.a, pairs[tc.a]
						exp, err := placeOrderRequest(&s)
						require.NoErrorf(t, err, "placeOrderRequest must not error for a %s %s %s order without a margin type", tc.a, s.Side, s.Type)
						// The cases set the margin type of s in turn, and the quote amount check after them reuses s, so they run in order
						for _, mt := range []struct {
							name       string
							marginType margin.Type
						}{
							{name: "unset", marginType: margin.Unset},
							{name: "isolated", marginType: margin.Isolated},
							{name: "multi", marginType: margin.Multi},
							{name: "no margin", marginType: margin.NoMargin},
							{name: "spot isolated", marginType: margin.SpotIsolated},
							{name: "unknown", marginType: margin.Unknown},
							{name: "isolated, multi and no margin at once", marginType: margin.Isolated | margin.Multi | margin.NoMargin},
						} {
							t.Run(mt.name, func(t *testing.T) {
								s.MarginType = mt.marginType
								got, err := placeOrderRequest(&s)
								if mt.marginType == margin.Unset || mt.marginType == tc.applied {
									require.NoErrorf(t, err, "placeOrderRequest must not error for a %s %s %s order of margin type %s", tc.a, s.Side, s.Type, mt.marginType)
									assert.Equalf(t, exp, got, "a %s %s %s order of margin type %s should send the same request", tc.a, s.Side, s.Type, mt.marginType)
									return
								}
								assert.ErrorIsf(t, err, margin.ErrMarginTypeUnsupported, "placeOrderRequest should refuse a %s %s %s order of margin type %s", tc.a, s.Side, s.Type, mt.marginType)
								assert.Nilf(t, got, "placeOrderRequest should return no request for a %s %s %s order of margin type %s", tc.a, s.Side, s.Type, mt.marginType)
							})
						}
						s.MarginType, s.QuoteAmount = margin.Unset, 0.5
						got, err := placeOrderRequest(&s)
						assert.ErrorIsf(t, err, errQuoteAmountUnsupported, "placeOrderRequest should refuse a quote amount for a %s %s %s order", tc.a, s.Side, s.Type)
						assert.Nilf(t, got, "placeOrderRequest should return no request for a %s %s %s order with a quote amount", tc.a, s.Side, s.Type)
					})
				}
			})
		}
		got, err := placeOrderRequest(&order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Market, Side: order.Buy, QuoteAmount: 50})
		assert.ErrorIs(t, err, errQuoteAmountUnsupported, "placeOrderRequest should refuse a quote amount given in place of an amount")
		assert.Nil(t, got, "placeOrderRequest should return no request for a quote amount given in place of an amount")
	})

	t.Run("take profit, stop loss and trigger price type refusals", func(t *testing.T) {
		t.Parallel()
		pairs := map[asset.Item]currency.Pair{
			asset.Spot:                currency.NewBTCUSD(),
			asset.Margin:              currency.NewBTCUSD(),
			asset.USDTMarginedFutures: testDerivativePair,
			asset.CoinMarginedFutures: testCoinMarginedPair,
		}
		// Every order type placeOrderRequest takes, every name for a side and every time in force it takes, as no refusal
		// of either depends on them
		orders := []order.Submit{
			{Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100},
			{Type: order.Market, Side: order.Sell, Amount: 1, TimeInForce: order.ImmediateOrCancel},
			{Type: order.Stop, Side: order.Bid, Amount: 1, TriggerPrice: 100, TimeInForce: order.PostOnly},
			{Type: order.StopLimit, Side: order.Ask, Amount: 1, Price: 90, TriggerPrice: 95},
			{Type: order.TrailingStop, Side: order.Long, Amount: 1, TrackingValue: 10, ReduceOnly: true},
			{Type: order.Limit, Side: order.Short, Amount: 1, Price: 100, TimeInForce: order.FillOrKill},
			{Type: order.StopLimit, Side: order.Sell, Amount: 1, Price: 90, TriggerPrice: 95, TimeInForce: order.GoodTillTime, EndTime: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)},
			{Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillCrossing},
			{Type: order.Limit, Side: order.Sell, Amount: 1, Price: 100, TimeInForce: order.GoodTillCancel},
			{Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillDay | order.GoodTillTime, EndTime: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)},
		}
		options := orderOptionCases()
		for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			for _, s := range orders {
				s.AssetType, s.Pair = a, pairs[a]
				name := a.String() + " " + s.Side.String() + " " + s.Type.String()
				if s.TimeInForce != order.UnknownTIF {
					name += " " + s.TimeInForce.String()
				}
				for _, option := range options {
					t.Run(name+" with "+option.name, func(t *testing.T) {
						t.Parallel()
						_, err := placeOrderRequest(&s)
						require.NoErrorf(t, err, "placeOrderRequest must place a %s order", name)
						with := s
						with.RiskManagementModes, with.TriggerPriceType = option.risk, option.triggerPriceType
						got, err := placeOrderRequest(&with)
						assert.ErrorIsf(t, err, option.err, "placeOrderRequest should refuse a %s order with %s", name, option.name)
						assert.Nilf(t, got, "placeOrderRequest should return no request for a %s order with %s", name, option.name)
					})
				}
			}
		}
	})

	until := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		s    *order.Submit
		exp  *PlaceOrderRequest
		err  error
	}{
		{
			name: "spot limit buy",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, ClientOrderID: "12"},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE LIMIT", Symbol: "tBTCUSD", Amount: 1, Price: 100, ClientOrderID: 12},
		},
		{
			name: "margin limit sell, post only and reduce only",
			s:    &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Sell, Amount: 1, Price: 100, TimeInForce: order.PostOnly, ReduceOnly: true},
			exp:  &PlaceOrderRequest{Type: "LIMIT", Symbol: "tBTCUSD", Amount: -1, Price: 100, Flags: OrderFlagPostOnly | OrderFlagReduceOnly},
		},
		{
			name: "fill or kill",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.FillOrKill},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE FOK", Symbol: "tBTCUSD", Amount: 1, Price: 100},
		},
		{
			name: "immediate or cancel",
			s:    &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Short, Amount: 1, Price: 100, TimeInForce: order.ImmediateOrCancel},
			exp:  &PlaceOrderRequest{Type: "IOC", Symbol: "tBTCUSD", Amount: -1, Price: 100},
		},
		{
			name: "good till time",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillTime, EndTime: until},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE LIMIT", Symbol: "tBTCUSD", Amount: 1, Price: 100, TimeInForce: until},
		},
		{
			name: "market",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Market, Side: order.Sell, Amount: 1, Price: 100},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE MARKET", Symbol: "tBTCUSD", Amount: -1},
		},
		{
			name: "stop",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Stop, Side: order.Sell, Amount: 1, TriggerPrice: 90},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE STOP", Symbol: "tBTCUSD", Amount: -1, Price: 90},
		},
		{
			name: "stop limit",
			s:    &order.Submit{AssetType: asset.Margin, Pair: currency.NewPair(currency.AVAX, currency.USD), Type: order.StopLimit, Side: order.Buy, Amount: 1, Price: 105, TriggerPrice: 110},
			exp:  &PlaceOrderRequest{Type: "STOP LIMIT", Symbol: "tAVAX:USD", Amount: 1, Price: 110, AuxiliaryLimitPrice: 105},
		},
		{
			name: "trailing stop",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingValue: 50},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE TRAILING STOP", Symbol: "tBTCUSD", Amount: -1, TrailingPrice: 50},
		},
		{
			name: "USDT margined derivative limit buy with leverage",
			s:    &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: testDerivativePair, Type: order.Limit, Side: order.Buy, Amount: 0.001, Price: 70000, Leverage: 20},
			exp:  &PlaceOrderRequest{Type: "LIMIT", Symbol: "tBTCF0:USTF0", Amount: 0.001, Price: 70000, Leverage: 20},
		},
		{
			name: "coin margined derivative market sell, reduce only, at Bitfinex's default leverage",
			s:    &order.Submit{AssetType: asset.CoinMarginedFutures, Pair: testCoinMarginedPair, Type: order.Market, Side: order.Sell, Amount: 1, ReduceOnly: true},
			exp:  &PlaceOrderRequest{Type: "MARKET", Symbol: "tETHF0:BTCF0", Amount: -1, Flags: OrderFlagReduceOnly},
		},
		{
			name: "margin order leverage, which Bitfinex takes for derivatives only",
			s:    &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, Leverage: 3},
			exp:  &PlaceOrderRequest{Type: "LIMIT", Symbol: "tBTCUSD", Amount: 1, Price: 100},
		},
		{name: "fractional leverage", s: &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: testDerivativePair, Type: order.Limit, Leverage: 2.5}, err: errInvalidLeverage},
		{name: "leverage above 100", s: &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: testDerivativePair, Type: order.Limit, Leverage: 101}, err: errInvalidLeverage},
		{name: "leverage below 1", s: &order.Submit{AssetType: asset.CoinMarginedFutures, Pair: testCoinMarginedPair, Type: order.Limit, Leverage: 0.5}, err: errInvalidLeverage},
		{name: "spot pair as a derivative", s: &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSD(), Type: order.Limit}, err: errPairNotOfAsset},
		{name: "derivative pair on margin", s: &order.Submit{AssetType: asset.Margin, Pair: testDerivativePair, Type: order.Limit}, err: errPairNotOfAsset},
		{name: "derivative pair of the other collateral", s: &order.Submit{AssetType: asset.CoinMarginedFutures, Pair: testDerivativePair, Type: order.Limit}, err: errPairNotOfAsset},
		{name: "funding", s: &order.Submit{AssetType: asset.MarginFunding, Pair: testFundingUSD, Type: order.Limit}, err: asset.ErrNotSupported},
		{name: "unsupported type", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.OCO}, err: errUnsupportedOrderType},
		{name: "stop without a trigger price", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Stop, Side: order.Sell, Amount: 1, Price: 90}, err: errTriggerPriceRequired},
		{name: "stop limit without a trigger price", s: &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.StopLimit, Side: order.Buy, Amount: 1, Price: 105}, err: errTriggerPriceRequired},
		{name: "stop limit without a limit price", s: &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.StopLimit, Side: order.Buy, Amount: 1, TriggerPrice: 110}, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "limit with a trigger price", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TriggerPrice: 90}, err: errTriggerPriceUnsupported},
		{name: "limit with a negative trigger price", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Sell, Amount: 1, Price: 100, TriggerPrice: -90}, err: errTriggerPriceUnsupported},
		{name: "market with a negative trigger price", s: &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.Market, Side: order.Sell, Amount: 1, TriggerPrice: -90}, err: errTriggerPriceUnsupported},
		{
			name: "fill or kill derivative limit with a trigger price",
			s:    &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: testDerivativePair, Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TriggerPrice: 90, TimeInForce: order.FillOrKill},
			err:  errTriggerPriceUnsupported,
		},
		{name: "market with a trigger price", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Market, Side: order.Buy, Amount: 1, TriggerPrice: 90}, err: errTriggerPriceUnsupported},
		{name: "limit with a NaN trigger price", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TriggerPrice: math.NaN()}, err: errTriggerPriceUnsupported},
		{
			name: "immediate or cancel limit with a trigger price",
			s:    &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Sell, Amount: 1, Price: 100, TriggerPrice: 110, TimeInForce: order.ImmediateOrCancel},
			err:  errTriggerPriceUnsupported,
		},
		{
			name: "coin margined derivative market with a trigger price",
			s:    &order.Submit{AssetType: asset.CoinMarginedFutures, Pair: testCoinMarginedPair, Type: order.Market, Side: order.Sell, Amount: 1, TriggerPrice: 90},
			err:  errTriggerPriceUnsupported,
		},
		{name: "limit with a fractional trigger price", s: &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Sell, Amount: 1, Price: 100, TriggerPrice: 0.5}, err: errTriggerPriceUnsupported},
		{
			name: "post only limit with a trigger price",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TriggerPrice: 90, TimeInForce: order.PostOnly},
			err:  errTriggerPriceUnsupported,
		},
		{
			name: "good till time limit with a trigger price",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TriggerPrice: 90, TimeInForce: order.GoodTillTime, EndTime: until},
			err:  errTriggerPriceUnsupported,
		},
		{
			name: "good till crossing limit with a trigger price",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TriggerPrice: 90, TimeInForce: order.GoodTillCrossing},
			err:  errTriggerPriceUnsupported,
		},
		{name: "good till day", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
		{name: "good till time without an end time", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillTime}, err: order.ErrUnsupportedTimeInForce},
		{
			name: "good till day with an end time",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillDay | order.GoodTillTime, EndTime: until},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE LIMIT", Symbol: "tBTCUSD", Amount: 1, Price: 100, TimeInForce: until},
		},
		{
			name: "good till crossing, which is post only",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillCrossing},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE LIMIT", Symbol: "tBTCUSD", Amount: 1, Price: 100, Flags: OrderFlagPostOnly},
		},
		{
			name: "good till crossing and post only",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 100, TimeInForce: order.GoodTillCrossing | order.PostOnly},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE LIMIT", Symbol: "tBTCUSD", Amount: 1, Price: 100, Flags: OrderFlagPostOnly},
		},
		{name: "good till crossing market", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Market, Side: order.Buy, Amount: 1, TimeInForce: order.GoodTillCrossing}, err: order.ErrUnsupportedTimeInForce},
		{name: "good till crossing stop", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Stop, Side: order.Sell, Amount: 1, TriggerPrice: 90, TimeInForce: order.GoodTillCrossing}, err: order.ErrUnsupportedTimeInForce},
		{name: "stop or reduce", s: &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Sell, Amount: 1, Price: 100, TimeInForce: order.StopOrReduce}, err: order.ErrUnsupportedTimeInForce},
		{name: "fill or kill market", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Market, Side: order.Buy, Amount: 1, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{name: "immediate or cancel stop", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Stop, Side: order.Sell, Amount: 1, TriggerPrice: 90, TimeInForce: order.ImmediateOrCancel}, err: order.ErrUnsupportedTimeInForce},
		{name: "fill or kill stop limit", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.StopLimit, Side: order.Sell, Amount: 1, Price: 89, TriggerPrice: 90, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{
			name: "immediate or cancel market, which a market order already is",
			s:    &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Market, Side: order.Sell, Amount: 1, TimeInForce: order.ImmediateOrCancel},
			exp:  &PlaceOrderRequest{Type: "EXCHANGE MARKET", Symbol: "tBTCUSD", Amount: -1},
		},
		{name: "percentage trailing stop", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.TrailingStop, Side: order.Sell, Amount: 1, TrackingMode: order.Percentage, TrackingValue: 5}, err: errTrackingModeUnsupported},
		{
			name: "distance trailing stop",
			s:    &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Type: order.TrailingStop, Side: order.Buy, Amount: 1, TrackingMode: order.Distance, TrackingValue: 50},
			exp:  &PlaceOrderRequest{Type: "TRAILING STOP", Symbol: "tBTCUSD", Amount: 1, TrailingPrice: 50},
		},
		{name: "client order ID", s: &order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, ClientOrderID: "abc"}, err: errInvalidClientOrderID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := placeOrderRequest(tc.s)
			require.ErrorIs(t, err, tc.err, "placeOrderRequest must return the expected error")
			assert.Equal(t, tc.exp, got, "placeOrderRequest should map the order to Bitfinex's request")
		})
	}
}

// orderOptionCase is an option of an order or a modification that Bitfinex takes no field for, set alone, and the error
// refusing it
type orderOptionCase struct {
	name             string
	risk             order.RiskManagementModes
	triggerPriceType order.PriceType
	err              error
}

// orderOptionCases returns each take profit and stop loss mode, each field of a take profit, stop loss and stop entry set
// alone, as wrappers read them by price or by flag, and each price type for a trigger but the last price
func orderOptionCases() []orderOptionCase {
	legs := []struct {
		name string
		of   func(*order.RiskManagementModes) *order.RiskManagement
	}{
		{"take profit", func(r *order.RiskManagementModes) *order.RiskManagement { return &r.TakeProfit }},
		{"stop loss", func(r *order.RiskManagementModes) *order.RiskManagement { return &r.StopLoss }},
		{"stop entry", func(r *order.RiskManagementModes) *order.RiskManagement { return &r.StopEntry }},
	}
	fields := []struct {
		name string
		set  func(*order.RiskManagement)
	}{
		{"flag", func(m *order.RiskManagement) { m.Enabled = true }},
		{"mark price trigger", func(m *order.RiskManagement) { m.TriggerPriceType = order.MarkPrice }},
		{"index price trigger", func(m *order.RiskManagement) { m.TriggerPriceType = order.IndexPrice }},
		{"unknown price type trigger", func(m *order.RiskManagement) { m.TriggerPriceType = order.UnknownPriceType }},
		{"price", func(m *order.RiskManagement) { m.Price = 120 }},
		{"negative price", func(m *order.RiskManagement) { m.Price = -120 }},
		{"price that is not a number", func(m *order.RiskManagement) { m.Price = math.NaN() }},
		{"limit price", func(m *order.RiskManagement) { m.LimitPrice = 121 }},
		{"order type", func(m *order.RiskManagement) { m.OrderType = order.Limit }},
	}
	cases := make([]orderOptionCase, 0, 5+len(legs)*len(fields))
	cases = append(cases,
		orderOptionCase{name: "a full take profit and stop loss mode", risk: order.RiskManagementModes{Mode: "full"}, err: errRiskManagementUnsupported},
		orderOptionCase{name: "a partial take profit and stop loss mode", risk: order.RiskManagementModes{Mode: "partial"}, err: errRiskManagementUnsupported},
		orderOptionCase{name: "a trigger on the index price", triggerPriceType: order.IndexPrice, err: errTriggerPriceTypeUnsupported},
		orderOptionCase{name: "a trigger on the mark price", triggerPriceType: order.MarkPrice, err: errTriggerPriceTypeUnsupported},
		orderOptionCase{name: "a trigger on an unknown price type", triggerPriceType: order.UnknownPriceType, err: errTriggerPriceTypeUnsupported},
	)
	for _, leg := range legs {
		for _, field := range fields {
			c := orderOptionCase{name: "a " + leg.name + "'s " + field.name, err: errRiskManagementUnsupported}
			field.set(leg.of(&c.risk))
			cases = append(cases, c)
		}
	}
	return cases
}

func TestCheckOrderOptions(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkOrderOptions(new(order.RiskManagementModes), order.LastPrice), "checkOrderOptions should take a trigger on the last price, a price type's zero value, as none")
	for _, tc := range orderOptionCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, checkOrderOptions(&tc.risk, tc.triggerPriceType), tc.err, "checkOrderOptions should refuse %s", tc.name)
		})
	}
}

func TestSubmitOrder(t *testing.T) { //nolint:tparallel // The cases share placed in order
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must refuse a nil submission")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, AssetType: asset.MarginFunding, Pair: testFundingUSD, Type: order.Limit, Side: order.Buy, Amount: 1, Price: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder must refuse funding orders")
	limit := order.Submit{AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 1, Price: 1}
	refused := newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`)
	limit.Exchange = refused.Name
	_, err = refused.SubmitOrder(t.Context(), &limit)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SubmitOrder must return the error Bitfinex reports")
	unanswered := newResponseServerExchange(t, `[1678987199,"on-req",null,null,[],null,"SUCCESS","Submitting 0 orders."]`)
	limit.Exchange = unanswered.Name
	_, err = unanswered.SubmitOrder(t.Context(), &limit)
	require.ErrorIs(t, err, common.ErrNoResponse, "SubmitOrder must report a reply without the order placed")
	unsent, bodies := newRequestBodyExchange(t, "/v2/auth/w/order/submit", `[1678987199,"on-req",null,null,[],null,"SUCCESS","Submitting 0 orders."]`)
	limit.Exchange, limit.RiskManagementModes = unsent.Name, order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 0.9}}
	_, err = unsent.SubmitOrder(t.Context(), &limit)
	assert.ErrorIs(t, err, errRiskManagementUnsupported, "SubmitOrder should refuse a stop loss, which Bitfinex takes no field for")
	limit.RiskManagementModes, limit.TriggerPriceType = order.RiskManagementModes{}, order.MarkPrice
	_, err = unsent.SubmitOrder(t.Context(), &limit)
	assert.ErrorIs(t, err, errTriggerPriceTypeUnsupported, "SubmitOrder should refuse a trigger price type, which Bitfinex takes no field for")
	assert.Empty(t, bodies, "SubmitOrder should send no order it refuses")

	// Bitfinex places each order with its price at 90: a stop's and a stop limit's trigger, and a trailing stop's stop
	placed := newResponseServerExchange(t, `[1678987199,"on-req",null,null,[[1747566431,null,1678987199446,"tBTCUSD",1678987199446,1678987199448,-1,-1,"EXCHANGE STOP",null,null,null,0,"ACTIVE",null,null,90,0,0,0,null,null,null,0,0,null,null,null,"API>BFX",null,null,null]],null,"SUCCESS","Submitting exchange stop sell order for 1 BTC."]`)
	// The cases share placed in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		s                   order.Submit
		price, triggerPrice float64
	}{
		{s: order.Submit{Type: order.Stop, Price: 89, TriggerPrice: 90}, triggerPrice: 90},
		{s: order.Submit{Type: order.StopLimit, Price: 89, TriggerPrice: 90}, price: 89, triggerPrice: 90},
		{s: order.Submit{Type: order.TrailingStop, Price: 89, TriggerPrice: 70, TrackingMode: order.Distance, TrackingValue: 5}, triggerPrice: 90},
	} {
		t.Run(tc.s.Type.String(), func(t *testing.T) {
			tc.s.Exchange, tc.s.AssetType, tc.s.Pair, tc.s.Side, tc.s.Amount = placed.Name, asset.Spot, currency.NewBTCUSD(), order.Sell, 1
			resp, err := placed.SubmitOrder(t.Context(), &tc.s)
			require.NoErrorf(t, err, "SubmitOrder must not error for a %s order", tc.s.Type)
			assert.Equalf(t, tc.price, resp.Price, "SubmitOrder should report a %s order's limit price, which neither stop has", tc.s.Type)
			assert.Equalf(t, tc.triggerPrice, resp.TriggerPrice, "SubmitOrder should report a %s order's trigger price, a trailing stop's as Bitfinex placed it", tc.s.Type)
		})
	}
	skipLiveUnlessCanManipulate(t)
	ex := newWrapperTestExchange(t)
	s := &order.Submit{Exchange: ex.Name, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: 0.1, Price: 10000, ClientOrderID: "1678987199446"}
	if !mockTests {
		req := liveTestOrderRequest(t)
		s.Amount, s.Price, s.ClientOrderID = req.Amount, req.Price, ""
	}
	resp, err := ex.SubmitOrder(t.Context(), s)
	require.NoError(t, err, "SubmitOrder must not error")
	if !mockTests {
		id, err := parseOrderID(resp.OrderID)
		require.NoError(t, err, "SubmitOrder must return the order's ID")
		cancelLiveTestOrderOnCleanup(t, id)
	} else {
		assert.Equal(t, testWrapperOrderID, resp.OrderID, "SubmitOrder should return the order's ID")
	}
	assert.Equal(t, order.Active, resp.Status, "SubmitOrder should return the order's status")
	if !mockTests {
		return
	}
	resp, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, AssetType: asset.USDTMarginedFutures, Pair: testDerivativePair, Type: order.Limit, Side: order.Buy, Amount: 0.001, Price: 70000, Leverage: 20})
	require.NoError(t, err, "SubmitOrder must not error for a derivative order")
	assert.Equal(t, testDerivativeOrderID, resp.OrderID, "SubmitOrder should return the derivative order's ID")
	assert.Equal(t, asset.USDTMarginedFutures, resp.AssetType, "SubmitOrder should return the derivative order's asset")
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	t.Run("transfer chains", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		f := ex.Features.Supports.RESTCapabilities
		assert.True(t, f.MultiChainDeposits, "deposits should be supported over each of a currency's methods")
		assert.True(t, f.MultiChainWithdrawals, "withdrawals should be supported over each of a currency's methods")
		assert.True(t, f.MultiChainDepositRequiresChainSet, "a deposit address should need its method, as a currency with several has no default")
	})
	t.Run("derivatives", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		ex.SetDefaults()
		// The cases share ex, so they run in order
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			t.Run(a.String(), func(t *testing.T) {
				assert.NoErrorf(t, ex.CurrencyPairs.IsAssetEnabled(a), "SetDefaults should enable %s", a)
			})
		}
		f := ex.Features.Supports.FuturesCapabilities
		assert.True(t, f.FundingRates, "funding rates should be supported")
		assert.Equal(t, map[kline.Interval]bool{kline.EightHour: true}, f.SupportedFundingRateFrequencies, "funding should be set every 8 hours")
		assert.Equal(t, map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true}, f.FundingRateBatching, "every pair's funding should come in one request")
		assert.True(t, f.Positions, "positions should be supported")
		assert.True(t, f.Leverage, "leverage should be supported")
		assert.Equal(t, exchange.OpenInterestSupport{Supported: true, SupportsRestBatch: true}, f.OpenInterest, "open interest should come in one request")
	})
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	t.Run("sends once", func(t *testing.T) {
		t.Parallel()
		o := testOrderRecord{id: 21, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
		for _, tc := range []struct {
			name   string
			status int
			reply  string
			err    error
		}{
			{name: "a rate limit refusal", status: http.StatusTooManyRequests, reply: `{"message":"ratelimit"}`, err: request.ErrAuthRequestFailed},
			{name: "a refused update", status: http.StatusOK, reply: `[1678988263842,"ou-req",null,null,[21],null,"ERROR","Update refused"]`, err: errNotificationFailed},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var updates atomic.Int64
				ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/v2/auth/r/orders" {
						_, err := w.Write([]byte(`[` + o.String() + `]`))
						assert.NoError(t, err, "Writing the response should not error")
						return
					}
					assert.Equal(t, "/v2/auth/w/order/update", r.URL.Path, "the test server should receive only the order's lookup and its update")
					// Only the first update is refused, so a repeat of it would succeed
					if updates.Add(1) > 1 {
						_, err := w.Write([]byte(`[1678988263842,"ou-req",null,null,[21],null,"SUCCESS","Submitting update"]`))
						assert.NoError(t, err, "Writing the response should not error")
						return
					}
					w.WriteHeader(tc.status)
					_, err := w.Write([]byte(tc.reply))
					assert.NoError(t, err, "Writing the response should not error")
				}))
				_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "21", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 2})
				assert.ErrorIsf(t, err, tc.err, "ModifyOrder should return %s", tc.name)
				assert.Equalf(t, int64(1), updates.Load(), "ModifyOrder should send an update once for %s, as a repeat of its amount would put back a fill made since", tc.name)
			})
		}
	})

	// Beside the scenario above, as when it was a test of its own
	t.Run("modifications", func(t *testing.T) {
		t.Parallel()
		_, err := e.ModifyOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")
		_, err = e.ModifyOrder(t.Context(), &order.Modify{OrderID: "x", AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Amount: 1})
		require.ErrorIs(t, err, errOrderIDRequired, "ModifyOrder must need a numeric order ID")
		_, err = e.ModifyOrder(t.Context(), &order.Modify{OrderID: testWrapperOrderID, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Amount: 1})
		require.ErrorIs(t, err, order.ErrSideIsInvalid, "ModifyOrder must need the side of an order whose amount it changes")

		t.Run("amount signed by side", func(t *testing.T) {
			t.Parallel()
			ex, updates := newOrderUpdateTestExchange(t, testOrderRecord{id: 1, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 0.4, original: 0.4, price: 24000})
			// The cases share ex in order, each checking the last update it recorded
			for _, tc := range []struct {
				name   string
				side   order.Side
				amount float64
				exp    map[string]any
			}{
				{name: "buy", side: order.Buy, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "0.25", "price": "25000"}},
				{name: "bid", side: order.Bid, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "0.25", "price": "25000"}},
				{name: "long", side: order.Long, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "0.25", "price": "25000"}},
				{name: "sell", side: order.Sell, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "-0.25", "price": "25000"}},
				{name: "ask", side: order.Ask, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "-0.25", "price": "25000"}},
				{name: "short", side: order.Short, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "-0.25", "price": "25000"}},
				{name: "sell, price only", side: order.Sell, exp: map[string]any{"id": 1.0, "price": "25000"}},
				{name: "unknown side, price only", side: order.UnknownSide, exp: map[string]any{"id": 1.0, "price": "25000"}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "1", AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: tc.side, Amount: tc.amount, Price: 25000})
					require.NoErrorf(t, err, "ModifyOrder must not error for a %s modification of %v", tc.side, tc.amount)
					sent := updates()
					require.NotEmptyf(t, sent, "ModifyOrder must send the %s modification", tc.side)
					assert.Equalf(t, tc.exp, sent[len(sent)-1], "ModifyOrder should send a %s amount of %v signed by its side, a sell's negative, and none when only the price changes", tc.side, tc.amount)
				})
			}
		})

		t.Run("prices by type", func(t *testing.T) {
			t.Parallel()
			ex, updates := newOrderUpdateTestExchange(t,
				testOrderRecord{id: 2, symbol: "tBTCUSD", orderType: "EXCHANGE STOP LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 90, auxiliaryLimit: 89},
				testOrderRecord{id: 3, symbol: "tBTCUSD", orderType: "EXCHANGE STOP", status: "ACTIVE", amount: -1, original: -1, price: 90},
				testOrderRecord{id: 4, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100},
				testOrderRecord{id: 5, symbol: "tBTCF0:USTF0", orderType: "STOP LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 70000, auxiliaryLimit: 70500, meta: `{"$F33":10}`},
				testOrderRecord{id: 6, symbol: "tBTCUSD", orderType: "EXCHANGE TRAILING STOP", status: "ACTIVE", amount: -1, original: -1, price: 95},
			)
			// The cases share ex in order, each checking the last update it recorded, and the refusal after them counts them
			for _, tc := range []struct {
				name  string
				m     order.Modify
				exp   map[string]any
				price float64
			}{
				{
					// The order manager fills a modification's price from the stored order, whose price is a stop limit's limit
					name:  "a stop limit's amount alone",
					m:     order.Modify{OrderID: "2", Side: order.Sell, Amount: 2, Price: 89},
					exp:   map[string]any{"id": 2.0, "amount": "-2", "price_aux_limit": "89"},
					price: 89,
				},
				{
					name:  "a stop limit's trigger price",
					m:     order.Modify{OrderID: "2", Side: order.Sell, Amount: 2, Price: 89, TriggerPrice: 92},
					exp:   map[string]any{"id": 2.0, "amount": "-2", "price": "92", "price_aux_limit": "89"},
					price: 89,
				},
				{name: "a stop's trigger price", m: order.Modify{OrderID: "3", Side: order.Sell, Amount: 1, TriggerPrice: 92}, exp: map[string]any{"id": 3.0, "price": "92"}},
				{name: "a stop's price, which it has none of, with its trigger price", m: order.Modify{OrderID: "3", Side: order.Sell, Price: 91, TriggerPrice: 93}, exp: map[string]any{"id": 3.0, "price": "93"}},
				{
					name:  "a derivative stop limit, modified as another asset, at its own leverage",
					m:     order.Modify{OrderID: "5", Side: order.Buy, Price: 71000, TriggerPrice: 72000},
					exp:   map[string]any{"id": 5.0, "price": "72000", "price_aux_limit": "71000", "lev": 10.0},
					price: 71000,
				},
				{name: "a trailing stop's amount", m: order.Modify{OrderID: "6", Side: order.Sell, Amount: 1}, exp: map[string]any{"id": 6.0}},
				{name: "a trailing stop's new amount", m: order.Modify{OrderID: "6", Side: order.Sell, Amount: 2}, exp: map[string]any{"id": 6.0, "amount": "-2"}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					tc.m.AssetType, tc.m.Pair = asset.Spot, currency.NewBTCUSD()
					resp, err := ex.ModifyOrder(t.Context(), &tc.m)
					require.NoErrorf(t, err, "ModifyOrder must not error for %s", tc.name)
					sent := updates()
					require.NotEmptyf(t, sent, "ModifyOrder must send %s", tc.name)
					assert.Equalf(t, tc.exp, sent[len(sent)-1], "ModifyOrder should send %s as Bitfinex takes it for the order's type", tc.name)
					assert.Equalf(t, tc.price, resp.Price, "ModifyOrder should report the price it sent for %s, and none for a stop", tc.name)
					assert.Equalf(t, tc.m.TriggerPrice, resp.TriggerPrice, "ModifyOrder should report the trigger price it sent for %s", tc.name)
				})
			}
			sent := len(updates())
			_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "4", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, TriggerPrice: 92})
			assert.ErrorIs(t, err, errTriggerPriceUnsupported, "ModifyOrder should refuse a trigger price for a limit order, rather than report one it never sent")
			assert.Len(t, updates(), sent, "ModifyOrder should send nothing for a modification it refuses")
		})

		t.Run("what remains of a partly filled order", func(t *testing.T) {
			t.Parallel()
			ex, updates := newOrderUpdateTestExchange(t,
				testOrderRecord{id: 7, symbol: "tBTCUSD", orderType: "LIMIT", status: "PARTIALLY FILLED @ 100.0(0.4)", amount: 0.6, original: 1, price: 100},
				testOrderRecord{id: 14, symbol: "tBTCUSD", orderType: "LIMIT", status: "PARTIALLY FILLED @ 100.0(-0.4)", amount: -0.6, original: -1, price: 100},
				testOrderRecord{id: 20, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: 10000000.12345678, original: 10000000.12345678, price: 100},
				testOrderRecord{id: 22, symbol: "tBTCUSD", orderType: "LIMIT", status: "PARTIALLY FILLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.3)", amount: 0.3, original: 1, price: 100},
			)
			// The order manager fills a modification's amount from the stored order, its whole amount
			resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 1, Price: 101})
			require.NoError(t, err, "ModifyOrder must not error for a price change")
			require.Len(t, updates(), 1, "ModifyOrder must send the price change")
			assert.Equal(t, map[string]any{"id": 7.0, "price": "101"}, updates()[0], "ModifyOrder should send no amount for a partly filled order whose whole amount is unchanged, leaving what remains to Bitfinex")
			assert.Zero(t, resp.RemainingAmount, "ModifyOrder should report no remainder for a price change, as the one it looked up can be older than a fill the order store holds")
			assert.Equal(t, 1.0, resp.Amount, "ModifyOrder should report the order's whole amount")

			resp, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 0.5})
			require.NoError(t, err, "ModifyOrder must not error for an amount change")
			require.Len(t, updates(), 2, "ModifyOrder must send the amount change")
			assert.Equal(t, map[string]any{"id": 7.0, "amount": "0.1"}, updates()[1], "ModifyOrder should send what remains of the new amount, worked out without rounding error")
			assert.Equal(t, 0.1, resp.RemainingAmount, "ModifyOrder should report what remains of the amount it sends")

			_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 0.4})
			assert.ErrorIs(t, err, errInvalidAmount, "ModifyOrder should refuse an amount no more than the order has executed")
			assert.Len(t, updates(), 2, "ModifyOrder should send nothing for an amount it refuses")

			_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 0.99999999})
			require.NoError(t, err, "ModifyOrder must not error for an amount change of the smallest step")
			require.Len(t, updates(), 3, "ModifyOrder must send an amount change of the smallest step")
			assert.Equal(t, map[string]any{"id": 7.0, "amount": "0.59999999"}, updates()[2], "ModifyOrder should send an amount the smallest step from what remains")

			_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "14", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Sell, Amount: 1, Price: 99})
			require.NoError(t, err, "ModifyOrder must not error for a sell's price change")
			require.Len(t, updates(), 4, "ModifyOrder must send the sell's price change")
			assert.Equal(t, map[string]any{"id": 14.0, "price": "99"}, updates()[3], "ModifyOrder should send no amount for a partly filled sell whose whole amount is unchanged")

			_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "20", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 10000000.12345679})
			require.NoError(t, err, "ModifyOrder must not error for a change of a large amount by its smallest step")
			require.Len(t, updates(), 5, "ModifyOrder must send a change of a large amount by its smallest step")
			assert.Equal(t, map[string]any{"id": 20.0, "amount": "10000000.12345679"}, updates()[4], "ModifyOrder should send a large amount changed by its smallest step")

			resp, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "14", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Sell, Amount: 0.5})
			require.NoError(t, err, "ModifyOrder must not error for a sell's amount change")
			require.Len(t, updates(), 6, "ModifyOrder must send the sell's amount change")
			assert.Equal(t, map[string]any{"id": 14.0, "amount": "-0.1"}, updates()[5], "ModifyOrder should send what remains of a sell as negative")
			assert.Equal(t, 0.1, resp.RemainingAmount, "ModifyOrder should report what remains of a sell as positive")

			// A new whole amount can equal what remains of the old one
			resp, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 0.6})
			require.NoError(t, err, "ModifyOrder must not error for a new amount equal to what remains")
			require.Len(t, updates(), 7, "ModifyOrder must send a new amount equal to what remains")
			assert.Equal(t, map[string]any{"id": 7.0, "amount": "0.2"}, updates()[6], "ModifyOrder should send what remains of a new amount equal to what remains")
			assert.Equal(t, 0.2, resp.RemainingAmount, "ModifyOrder should report what remains of a new amount equal to what remains")

			// Bitfinex keeps an order's original amount from before any update, so its original amount less what remains, 0.7, is more than it executed
			resp, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "22", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 1.2})
			require.NoError(t, err, "ModifyOrder must not error for an amount change of an order updated before")
			require.Len(t, updates(), 8, "ModifyOrder must send an amount change of an order updated before")
			assert.Equal(t, map[string]any{"id": 22.0, "amount": "0.7"}, updates()[7], "ModifyOrder should send the new amount less the fills the order's status lists")
			assert.Equal(t, 0.7, resp.RemainingAmount, "ModifyOrder should report the new amount less the fills the order's status lists")
		})

		t.Run("an order whose status was cut short", func(t *testing.T) {
			t.Parallel()
			// The order, placed for 1 and lowered to 0.8, executed 0.5, of which its cut status lists 0.4, so its original
			// amount less what remains, 0.7, is more than it executed
			ms := func(at time.Time) string { return strconv.FormatInt(at.UnixMilli(), 10) }
			read := time.Now().Add(-time.Hour)
			o := testOrderRecord{
				id: 18, symbol: "tBTCUSD", orderType: "LIMIT", status: "... 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.2)",
				amount: 0.3, original: 1, price: 100, createdAt: ms(read.Add(-time.Hour)), updatedAt: ms(read),
			}
			trade := func(id, amount string) string {
				return `[` + id + `,"tBTCUSD",` + ms(read.Add(-time.Minute)) + `,18,` + amount + `,100,"LIMIT",100,1,-0.01,"USD",0]`
			}
			var mu sync.Mutex
			var sent []map[string]any
			var tradeReads int
			newExchange := func(o testOrderRecord, trades string) *Exchange {
				return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					body := `[` + o.String() + `]`
					switch r.URL.Path {
					case "/v2/auth/r/orders":
					case "/v2/auth/r/order/" + o.symbol + ":" + strconv.FormatUint(o.id, 10) + "/trades":
						tradeReads++
						body = trades
					case "/v2/auth/w/order/update":
						var req map[string]any
						assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "Decoding the update should not error")
						sent = append(sent, req)
						body = `[1678988263842,"ou-req",null,null,[18],null,"SUCCESS","Submitting update"]`
					default:
						http.Error(w, "unexpected path", http.StatusNotFound)
						assert.Failf(t, "the test server should receive only the requests expected", "%s %s", r.Method, r.URL.Path)
						return
					}
					_, err := w.Write([]byte(body))
					assert.NoError(t, err, "Writing the response should not error")
				}))
			}
			ex := newExchange(o, `[`+trade("1", "0.1")+`,`+trade("2", "0.2")+`,`+trade("3", "0.2")+`]`)
			// The cases share ex in order, each checking the last update recorded and the running count of trade reads
			for _, tc := range []struct {
				name       string
				amount     float64
				exp        map[string]any
				tradeReads int
				remaining  float64
			}{
				{name: "the whole amount the order is reported with, as the order manager sends", amount: 1, exp: map[string]any{"id": 18.0, "price": "101"}},
				{name: "the whole amount the order is reported with, carried as a float", amount: math.Nextafter(1, 2), exp: map[string]any{"id": 18.0, "price": "101"}},
				{name: "the whole amount the order has, by its trades", amount: 0.8, exp: map[string]any{"id": 18.0, "price": "101"}, tradeReads: 1},
				{name: "a new whole amount, less what its trades executed", amount: 1.2, exp: map[string]any{"id": 18.0, "amount": "0.7", "price": "101"}, tradeReads: 2, remaining: 0.7},
			} {
				t.Run(tc.name, func(t *testing.T) {
					resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "18", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: tc.amount, Price: 101})
					require.NoErrorf(t, err, "ModifyOrder must not error for %s", tc.name)
					assert.Equalf(t, tc.remaining, resp.RemainingAmount, "ModifyOrder should report what remains only of an amount it sends, for %s", tc.name)
					mu.Lock()
					assert.Equalf(t, tc.exp, sent[len(sent)-1], "ModifyOrder should send what remains of %s, unsent when unchanged", tc.name)
					assert.Equalf(t, tc.tradeReads, tradeReads, "ModifyOrder should read the order's trades only for an amount it changes, for %s", tc.name)
					mu.Unlock()
				})
			}
			_, err := newExchange(o, `["error",10020,"symbol: invalid"]`).ModifyOrder(t.Context(), &order.Modify{OrderID: "18", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 1.2})
			var apiErr *APIError
			assert.ErrorAs(t, err, &apiErr, "ModifyOrder should return the error Bitfinex reports reading the order's trades")

			// The order store holds the whole amount GetOrderInfo reported from the order's trades, rounded to a float
			precise := o
			precise.id, precise.amount = 19, 0.949681784542512
			ex = newExchange(precise, `[[1,"tBTCUSD",`+ms(read.Add(-time.Minute))+`,19,0.9692516051954019,100,"LIMIT",100,1,-0.01,"USD",0]]`)
			_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "19", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 1.918933389737914, Price: 101})
			require.NoError(t, err, "ModifyOrder must not error for a price change of an order whose amounts fill a float")
			mu.Lock()
			assert.Equal(t, map[string]any{"id": 19.0, "price": "101"}, sent[len(sent)-1], "ModifyOrder should send no amount where what remains differs from the order's only past a float's precision")
			mu.Unlock()
		})

		t.Run("the order's type and time in force", func(t *testing.T) {
			t.Parallel()
			ex, updates := newOrderUpdateTestExchange(t, testOrderRecord{id: 16, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100, meta: `{"$F7":1}`})
			// The order manager fills a modification's time in force from the stored order. The cases share ex in order, and
			// the checks after them count the updates they sent
			for _, orderType := range []order.Type{order.UnknownType, order.AnyType, order.Limit} {
				t.Run(orderType.String(), func(t *testing.T) {
					resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "16", AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: orderType, TimeInForce: order.GoodTillCancel, Price: 101})
					require.NoErrorf(t, err, "ModifyOrder must not error for a modification of type %s", orderType)
					assert.Equalf(t, order.Limit, resp.Type, "ModifyOrder should report the order's type for a modification of type %s", orderType)
					assert.Equalf(t, order.PostOnly, resp.TimeInForce, "ModifyOrder should report the order's time in force, which it leaves unchanged, for a modification of type %s", orderType)
				})
			}
			require.Len(t, updates(), 3, "ModifyOrder must send each modification of the order's type")
			exp := map[string]any{"id": 16.0, "price": "101"}
			assert.Equal(t, []map[string]any{exp, exp, exp}, updates(), "ModifyOrder should send each modification's price alone, leaving the order's expiry and flags, post only among them, as they are")
			_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "16", AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: order.Stop, Price: 95})
			assert.ErrorIs(t, err, errOrderTypeUnchangeable, "ModifyOrder should refuse to make the order another type")
			assert.Len(t, updates(), 3, "ModifyOrder should send nothing for a modification it refuses")
		})

		t.Run("take profit, stop loss and trigger price type refusals", func(t *testing.T) {
			t.Parallel()
			ex, updates := newOrderUpdateTestExchange(t, testOrderRecord{id: 11, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100})
			pairs := map[asset.Item]currency.Pair{
				asset.Spot:                currency.NewBTCUSD(),
				asset.Margin:              currency.NewBTCUSD(),
				asset.USDTMarginedFutures: testDerivativePair,
				asset.CoinMarginedFutures: testCoinMarginedPair,
			}
			// No side, any side and every name for one, no type, any type and every type Bitfinex places, every time in force,
			// and new prices, amounts and trigger prices, as no refusal depends on them
			modifications := []struct {
				name string
				m    order.Modify
			}{
				{name: "a buy's new price", m: order.Modify{Side: order.Buy, Price: 101}},
				{name: "an ask limit's new amount", m: order.Modify{Side: order.Ask, Type: order.Limit, Amount: 2}},
				{name: "a post only long's new price", m: order.Modify{Side: order.Long, Price: 101, TimeInForce: order.PostOnly}},
				{name: "a sell's new amount", m: order.Modify{Side: order.Sell, Amount: 2}},
				{name: "a short stop's new trigger price", m: order.Modify{Side: order.Short, Type: order.Stop, TriggerPrice: 92}},
				{name: "a good till time bid stop limit's new prices", m: order.Modify{Side: order.Bid, Type: order.StopLimit, Price: 99, TriggerPrice: 92, TimeInForce: order.GoodTillTime}},
				{name: "a good till cancelled sell limit's new price", m: order.Modify{Side: order.Sell, Type: order.Limit, Price: 101, TimeInForce: order.GoodTillCancel}},
				{name: "an immediate or cancel buy market's new amount", m: order.Modify{Side: order.Buy, Type: order.Market, Amount: 2, TimeInForce: order.ImmediateOrCancel}},
				{name: "a fill or kill short limit's new amount", m: order.Modify{Side: order.Short, Type: order.Limit, Amount: 2, TimeInForce: order.FillOrKill}},
				{name: "a good till crossing bid limit's new price", m: order.Modify{Side: order.Bid, Type: order.Limit, Price: 101, TimeInForce: order.GoodTillCrossing}},
				{name: "a long trailing stop's new amount", m: order.Modify{Side: order.Long, Type: order.TrailingStop, Amount: 2}},
				{name: "a new price without a side", m: order.Modify{Price: 101}},
				{name: "a new price for any side", m: order.Modify{Side: order.AnySide, Price: 101}},
				{name: "a buy's new price for any type", m: order.Modify{Side: order.Buy, Type: order.AnyType, Price: 101}},
				{name: "a good till day sell's new price", m: order.Modify{Side: order.Sell, Price: 101, TimeInForce: order.GoodTillDay}},
				{name: "a stop or reduce short's new amount", m: order.Modify{Side: order.Short, Amount: 2, TimeInForce: order.StopOrReduce}},
			}
			options := orderOptionCases()
			// The cases share ex in order, and the check after them expects none of them to have sent an update
			for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
				for _, mod := range modifications {
					for _, option := range options {
						t.Run(a.String()+" "+mod.name+" with "+option.name, func(t *testing.T) {
							m := mod.m
							m.OrderID, m.AssetType, m.Pair = "11", a, pairs[a]
							m.RiskManagementModes, m.TriggerPriceType = option.risk, option.triggerPriceType
							_, err := ex.ModifyOrder(t.Context(), &m)
							assert.ErrorIsf(t, err, option.err, "ModifyOrder should refuse %s with %s for a %s order", mod.name, option.name, a)
						})
					}
				}
			}
			assert.Empty(t, updates(), "ModifyOrder should send no update it refuses")
		})

		t.Run("refusals", func(t *testing.T) {
			t.Parallel()
			ex, updates := newOrderUpdateTestExchange(t,
				testOrderRecord{id: 9, symbol: "tBTC", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100},
				testOrderRecord{id: 10, symbol: "tBTCF0:USTF0", orderType: "LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 70000},
				testOrderRecord{id: 11, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100},
				testOrderRecord{id: 12, symbol: "tBTCUSD", orderType: "NOVEL", status: "ACTIVE", amount: 1, original: 1, price: 100},
				testOrderRecord{id: 13, symbol: "tBTCUSD", orderType: "EXCHANGE STOP LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 90, auxiliaryLimit: 89},
				testOrderRecord{id: 15, symbol: "tBTCUSD", orderType: "EXCHANGE STOP", status: "ACTIVE", amount: -1, original: -1, price: 90},
				testOrderRecord{id: 17, symbol: "tBTCUSD", orderType: "EXCHANGE TRAILING STOP", status: "ACTIVE", amount: -1, original: -1, price: 95},
			)
			// The cases share ex in order, and the check after them expects none of them to have sent an update
			for _, tc := range []struct {
				name string
				m    order.Modify
				err  error
			}{
				{name: "a stop's price without its trigger price, as the RPC sends", m: order.Modify{OrderID: "15", Side: order.Sell, Amount: 1, Price: 95}, err: errTriggerPriceRequired},
				{name: "a trailing stop's price", m: order.Modify{OrderID: "17", Side: order.Sell, Amount: 1, Price: 95}, err: errPriceUnsupported},
				{name: "an order that is not open", m: order.Modify{OrderID: "8", Side: order.Buy, Amount: 1}, err: order.ErrOrderNotFound},
				{name: "an order on a symbol that is not a pair", m: order.Modify{OrderID: "9", Side: order.Buy, Amount: 1}, err: errSymbolUnparsable},
				{name: "a derivative order without leverage", m: order.Modify{OrderID: "10", Side: order.Buy, Amount: 0.2}, err: errOrderLeverageUnknown},
				{name: "a negative price", m: order.Modify{OrderID: "11", Side: order.Buy, Price: -1}, err: errInvalidPrice},
				{name: "an order of an unknown type", m: order.Modify{OrderID: "12", Side: order.Buy, Price: 101}, err: errUnknownOrderType},
				{name: "a stop limit's negative limit price", m: order.Modify{OrderID: "13", Side: order.Sell, Price: -1, TriggerPrice: 92}, err: errInvalidPrice},
				{name: "a stop limit's trigger price that is not a number", m: order.Modify{OrderID: "13", Side: order.Sell, TriggerPrice: math.NaN()}, err: errInvalidPrice},
			} {
				t.Run(tc.name, func(t *testing.T) {
					tc.m.AssetType, tc.m.Pair = asset.Spot, currency.NewBTCUSD()
					_, err := ex.ModifyOrder(t.Context(), &tc.m)
					assert.ErrorIsf(t, err, tc.err, "ModifyOrder should refuse %s", tc.name)
				})
			}
			assert.Empty(t, updates(), "ModifyOrder should send no update it refuses")
		})
		skipLiveUnlessCanManipulate(t)
		ex := newWrapperTestExchange(t)
		m := &order.Modify{OrderID: testWrapperOrderID, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 0.25, Price: 25000}
		if !mockTests {
			placed := placeLiveTestOrder(t)
			m.OrderID, m.Amount, m.Price = strconv.FormatUint(placed.ID, 10), placed.Amount, placed.Price-1000
		}
		resp, err := ex.ModifyOrder(t.Context(), m)
		require.NoError(t, err, "ModifyOrder must not error")
		assert.Equal(t, m.OrderID, resp.OrderID, "ModifyOrder should return the order modified")
		if !mockTests {
			return
		}
		// The recorded update matches only a request sending the order's own leverage, which Bitfinex needs to update it
		m = &order.Modify{OrderID: testDerivativeOrderID, AssetType: asset.USDTMarginedFutures, Pair: testDerivativePair, Side: order.Buy, Amount: 0.002, Price: 75000}
		resp, err = ex.ModifyOrder(t.Context(), m)
		require.NoError(t, err, "ModifyOrder must not error for a derivative order")
		assert.Equal(t, testDerivativeOrderID, resp.OrderID, "ModifyOrder should return the derivative order modified")

		wsEx, updates := newOrderUpdateTestExchange(t,
			testOrderRecord{id: 1747566428, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 0.4, original: 0.4, price: 24000},
			testOrderRecord{id: 1747566429, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: -0.4, original: -0.4, price: 26000},
		)
		venue := connectWebsocketTestExchange(t, wsEx, subscription.List{})
		require.True(t, wsEx.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the wrapper must be able to use the authenticated websocket")
		// The cases share wsEx in order, each checking its update by its index, and assign the test's resp and err
		for i, tc := range []struct {
			id   string
			side order.Side
			exp  map[string]any
		}{
			{id: "1747566428", side: order.Buy, exp: map[string]any{"id": 1747566428.0, "amount": "0.25", "price": "25000"}},
			{id: "1747566429", side: order.Sell, exp: map[string]any{"id": 1747566429.0, "amount": "-0.25", "price": "25000"}},
		} {
			t.Run(tc.side.String()+" with the websocket connected", func(t *testing.T) {
				resp, err = wsEx.ModifyOrder(t.Context(), &order.Modify{OrderID: tc.id, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: tc.side, Amount: 0.25, Price: 25000})
				require.NoErrorf(t, err, "ModifyOrder must not error for a %s with the websocket connected", tc.side)
				require.Lenf(t, updates(), i+1, "ModifyOrder must send the update of a %s over REST with the websocket connected", tc.side)
				assert.Equalf(t, tc.exp, updates()[i], "ModifyOrder should send a %s's update over REST with the websocket connected", tc.side)
				assert.Zerof(t, resp.RemainingAmount, "ModifyOrder should leave what remains of a %s to the websocket's order events, which can reach the order store ahead of its reply", tc.side)
			})
		}
		assert.Empty(t, venue.inputs(wsOrderUpdate), "ModifyOrder should send nothing through the websocket, where a late notification would answer the order's next update")
		_, err = wsEx.ModifyOrder(t.Context(), &order.Modify{OrderID: testWrapperOrderID, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Price: -1})
		assert.ErrorIs(t, err, errInvalidPrice, "ModifyOrder should return the update's error with the websocket connected")
	})
}

func TestSameAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		a, b float64
		exp  bool
	}{
		{name: "equal", a: 1, b: 1, exp: true},
		{name: "differing past 8 decimal places", a: 0.949681784542512, b: 0.9496817845425121, exp: true},
		{name: "above one, differing past 8 decimal places", a: 1.918933389737914, b: 1.9189333897379142, exp: true},
		{name: "differing in the 8th decimal place", a: 0.6, b: 0.60000001},
		{name: "ten thousand, differing in the 8th decimal place", a: 10000.12345678, b: 10000.12345679},
		{name: "ten million, differing in the 8th decimal place", a: 10000000.12345678, b: 10000000.12345679},
		{name: "opposite signs", a: 1, b: -1},
		{name: "not a number", a: math.NaN(), b: math.NaN()},
		{name: "infinity", a: math.Inf(1), b: math.Inf(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, sameAmount(tc.a, tc.b), "sameAmount should report whether %v and %v are the same amount to Bitfinex's 8 decimal places", tc.a, tc.b)
		})
	}
}

func TestSetUpdatePrices(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		orderType order.Type
		m         order.Modify
		exp       UpdateOrderRequest
		err       error
	}{
		{name: "limit price", orderType: order.Limit, m: order.Modify{Price: 100}, exp: UpdateOrderRequest{Price: 100}},
		{name: "market price", orderType: order.Market, m: order.Modify{Price: 100}, exp: UpdateOrderRequest{Price: 100}},
		{name: "limit trigger price", orderType: order.Limit, m: order.Modify{Price: 100, TriggerPrice: 90}, err: errTriggerPriceUnsupported},
		{name: "stop price and trigger price", orderType: order.Stop, m: order.Modify{Price: 100, TriggerPrice: 90}, exp: UpdateOrderRequest{Price: 90}},
		{name: "stop trigger price", orderType: order.Stop, m: order.Modify{TriggerPrice: 90}, exp: UpdateOrderRequest{Price: 90}},
		{name: "stop price without a trigger price", orderType: order.Stop, m: order.Modify{Price: 100}, err: errTriggerPriceRequired},
		{name: "stop without prices", orderType: order.Stop},
		{name: "stop limit price", orderType: order.StopLimit, m: order.Modify{Price: 89}, exp: UpdateOrderRequest{AuxiliaryLimitPrice: 89}},
		{name: "stop limit price and trigger price", orderType: order.StopLimit, m: order.Modify{Price: 89, TriggerPrice: 90}, exp: UpdateOrderRequest{Price: 90, AuxiliaryLimitPrice: 89}},
		{name: "trailing stop without prices", orderType: order.TrailingStop},
		{name: "trailing stop price", orderType: order.TrailingStop, m: order.Modify{Price: 95}, err: errPriceUnsupported},
		{name: "trailing stop trigger price", orderType: order.TrailingStop, m: order.Modify{TriggerPrice: 95}, err: errPriceUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got UpdateOrderRequest
			err := setUpdatePrices(&got, tc.orderType, &tc.m)
			require.ErrorIsf(t, err, tc.err, "setUpdatePrices must return the expected error for a %s order", tc.orderType)
			assert.Equalf(t, tc.exp, got, "setUpdatePrices should set the prices a %s order is updated by", tc.orderType)
		})
	}
}

func TestRemainingOrderAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		amount   float64
		executed string
		exp      float64
		err      error
	}{
		{name: "nothing executed", amount: 1, executed: "0", exp: 1},
		{name: "part executed", amount: 1, executed: "0.4", exp: 0.6},
		{name: "part executed, without rounding error", amount: 0.3, executed: "0.1", exp: 0.2},
		{name: "all executed", amount: 0.3, executed: "0.3", err: errInvalidAmount},
		{name: "more than the amount executed", amount: 0.25, executed: "0.4", err: errInvalidAmount},
		{name: "a negative amount", amount: -1, executed: "0", err: errInvalidAmount},
		{name: "an amount that is not a number", amount: math.NaN(), executed: "0", err: errInvalidAmount},
		{name: "an infinite amount", amount: math.Inf(1), executed: "0", err: errInvalidAmount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := remainingOrderAmount(tc.amount, 1, decimal.MustFromString(tc.executed))
			require.ErrorIsf(t, err, tc.err, "remainingOrderAmount must return the expected error for %v of an order that executed %s", tc.amount, tc.executed)
			assert.Equalf(t, tc.exp, got, "remainingOrderAmount should leave what an order of %v that executed %s has yet to execute", tc.amount, tc.executed)
		})
	}
}

func TestOpenOrder(t *testing.T) {
	t.Parallel()
	ex, _ := newOrderUpdateTestExchange(t, testOrderRecord{id: 5, symbol: "tBTCF0:USTF0", orderType: "LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 70000, meta: `{"$F33":10}`})
	got, err := ex.openOrder(t.Context(), 5)
	require.NoError(t, err, "openOrder must not error")
	assert.Equal(t, &OrderResponse{
		ID: 5, Symbol: "tBTCF0:USTF0", CreatedAt: types.Time(time.UnixMilli(1678987199446)), UpdatedAt: types.Time(time.UnixMilli(1678987199448)), Amount: 0.1,
		OriginalAmount: 0.1, Type: OrderTypeLimit, Status: "ACTIVE", Price: 70000, Routing: "API>BFX", Meta: json.RawMessage(`{"$F33":10}`),
	}, got, "openOrder should return the open order")
	_, err = ex.openOrder(t.Context(), 6)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "openOrder should report an order that is not open")
	_, err = newResponseServerExchange(t, `["error",10100,"apikey: invalid"]`).openOrder(t.Context(), 5)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "openOrder should return the error Bitfinex reports")
}

func TestOrderLeverage(t *testing.T) {
	t.Parallel()
	got, err := orderLeverage(&OrderResponse{ID: 1, Meta: json.RawMessage(`{"$F33":20}`)})
	require.NoError(t, err, "orderLeverage must not error")
	assert.Equal(t, uint64(20), got, "orderLeverage should return the leverage the order's meta data holds")
	for _, tc := range []struct {
		meta string
		err  error
	}{
		{meta: `null`, err: errOrderLeverageUnknown},
		{meta: `{"$F7":1}`, err: errOrderLeverageUnknown},
		{meta: `{"$F33":2.5}`, err: errInvalidLeverage},
		{meta: `{"$F33":101}`, err: errInvalidLeverage},
	} {
		t.Run(tc.meta, func(t *testing.T) {
			t.Parallel()
			_, err := orderLeverage(&OrderResponse{ID: 1, Meta: json.RawMessage(tc.meta)})
			assert.ErrorIsf(t, err, tc.err, "orderLeverage should return the expected error for meta data %s", tc.meta)
		})
	}
}

func TestDerivativeLeverage(t *testing.T) {
	t.Parallel()
	// Serial, as derivativeLeverage is called again after these cases
	for _, leverage := range []float64{1, 20, 100} {
		t.Run(strconv.FormatFloat(leverage, 'f', -1, 64), func(t *testing.T) {
			got, err := derivativeLeverage(leverage)
			require.NoErrorf(t, err, "derivativeLeverage must not error for %v", leverage)
			assert.Equalf(t, uint64(leverage), got, "derivativeLeverage should take leverage %v as it is", leverage)
		})
	}
	for _, leverage := range []float64{0, 0.5, 2.5, 101, -1} {
		t.Run(strconv.FormatFloat(leverage, 'f', -1, 64), func(t *testing.T) {
			t.Parallel()
			_, err := derivativeLeverage(leverage)
			assert.ErrorIsf(t, err, errInvalidLeverage, "derivativeLeverage should refuse leverage %v", leverage)
		})
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil request")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot}), order.ErrOrderIDNotSet, "CancelOrder must need an order ID")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: "x", AssetType: asset.Spot}), errOrderIDRequired, "CancelOrder must need a numeric order ID")
	skipLiveUnlessCanManipulate(t)
	ex := newWrapperTestExchange(t)
	id := testWrapperOrderID
	if !mockTests {
		id = strconv.FormatUint(placeLiveTestOrder(t).ID, 10)
	}
	assert.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: id, AssetType: asset.Spot, Pair: currency.NewBTCUSD()}), "CancelOrder should not error")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must need orders")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "x"}})
	require.ErrorIs(t, err, errOrderIDRequired, "CancelBatchOrders must need numeric order IDs")
	_, err = newResponseServerExchange(t, `["error",10001,"cancel: invalid"]`).CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1"}})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "CancelBatchOrders must return the error Bitfinex reports")
	skipLiveUnlessCanManipulate(t)
	ex := newWrapperTestExchange(t)
	ids := []string{testWrapperOrderID, "1747566429"}
	if !mockTests {
		ids = []string{strconv.FormatUint(placeLiveTestOrder(t).ID, 10)}
	}
	cancels := make([]order.Cancel, len(ids))
	for i := range ids {
		cancels[i] = order.Cancel{OrderID: ids[i], AssetType: asset.Spot, Pair: currency.NewBTCUSD()}
	}
	resp, err := ex.CancelBatchOrders(t.Context(), cancels)
	require.NoError(t, err, "CancelBatchOrders must not error")
	assert.Equal(t, order.PendingCancel.String(), resp.Status[ids[0]], "an order Bitfinex accepted for cancellation should be pending cancellation until it closes")
	if mockTests {
		assert.Equal(t, order.UnknownStatus.String(), resp.Status["1747566429"], "an order Bitfinex did not report should be unknown")
	}
}

// newCancelTestExchange returns an exchange whose open orders are those given, and whose cancellations Bitfinex
// answers naming those replied, and the cancellations it sends, decoded, in the order sent
func newCancelTestExchange(t *testing.T, open, replied []testOrderRecord) (ex *Exchange, cancellations func() []map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var sent []map[string]any
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch {
		case r.URL.Path == "/v2/auth/r/orders":
			body = testOrderRecords(open)
		case strings.HasPrefix(r.URL.Path, "/v2/auth/r/orders/"):
			symbol := strings.TrimPrefix(r.URL.Path, "/v2/auth/r/orders/")
			body = testOrderRecords(slices.DeleteFunc(slices.Clone(open), func(o testOrderRecord) bool { return o.symbol != symbol }))
		case r.URL.Path == "/v2/auth/w/order/cancel/multi":
			var req map[string]any
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "Decoding the cancellation should not error") {
				http.Error(w, "undecodable", http.StatusBadRequest)
				return
			}
			mu.Lock()
			sent = append(sent, req)
			mu.Unlock()
			body = `[1678987199,"oc_multi-req",null,null,` + testOrderRecords(replied) + `,null,"SUCCESS","Submitting order cancellations."]`
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
			assert.Failf(t, "the test server should receive only the requests expected", "%s %s", r.Method, r.URL.Path)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	return ex, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(sent)
	}
}

func TestSubmitOrderCancellations(t *testing.T) {
	t.Parallel()
	spot := testOrderRecord{id: 1747566428, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 10000}
	late := testOrderRecord{id: 1747566431, symbol: "tETHUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 2000}
	// Bitfinex replies with the orders it accepted for cancellation still active, one listed for it left out and one
	// placed since the listing included
	ex, cancellations := newCancelTestExchange(t, nil, []testOrderRecord{spot, late})
	got, err := ex.submitOrderCancellations(t.Context(), &CancelOrdersRequest{All: true}, []uint64{1747566428, 1747566429})
	require.NoError(t, err, "submitOrderCancellations must not error")
	assert.Equal(t, map[string]string{
		"1747566428": order.PendingCancel.String(), "1747566429": order.UnknownStatus.String(), "1747566431": order.PendingCancel.String(),
	}, got, "submitOrderCancellations should report the orders Bitfinex accepted pending cancellation, and one it left out unknown")
	assert.Equal(t, []map[string]any{{"all": 1.0}}, cancellations(), "submitOrderCancellations should send the request it is given")

	_, err = newResponseServerExchange(t, `["error",10001,"cancel: invalid"]`).submitOrderCancellations(t.Context(), &CancelOrdersRequest{IDs: []uint64{1}}, []uint64{1})
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "submitOrderCancellations should return the error Bitfinex reports")

	var cancels atomic.Int64
	limited := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Only the first request is refused, so a repeat of it would succeed
		if cancels.Add(1) > 1 {
			_, err := w.Write([]byte(`[1678987199,"oc_multi-req",null,null,[],null,"SUCCESS","Submitting order cancellations."]`))
			assert.NoError(t, err, "Writing the response should not error")
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	_, err = limited.submitOrderCancellations(t.Context(), &CancelOrdersRequest{All: true}, nil)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "submitOrderCancellations should return a rate limit refusal")
	assert.Equal(t, int64(1), cancels.Load(), "submitOrderCancellations should send a request once, as a repeat of one cancelling every order would reach an order placed since")

	if !mockTests {
		return
	}
	wsEx, wsCancellations := newCancelTestExchange(t, nil, []testOrderRecord{spot})
	venue := connectWebsocketTestExchange(t, wsEx, subscription.List{})
	require.True(t, wsEx.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the wrapper must be able to use the authenticated websocket")
	got, err = wsEx.submitOrderCancellations(t.Context(), &CancelOrdersRequest{All: true}, []uint64{1747566428, 1747566429})
	require.NoError(t, err, "submitOrderCancellations must not error with the websocket connected")
	assert.Equal(t, map[string]string{"1747566428": order.PendingCancel.String(), "1747566429": order.UnknownStatus.String()}, got, "submitOrderCancellations should report by Bitfinex's reply with the websocket connected")
	assert.Equal(t, []map[string]any{{"all": 1.0}}, wsCancellations(), "submitOrderCancellations should send the request over REST with the websocket connected")
	assert.Empty(t, venue.inputs(wsOrderCancelMulti), "submitOrderCancellations should send nothing through the websocket, where a late notification would answer the next request")
	_, err = wsEx.submitOrderCancellations(t.Context(), &CancelOrdersRequest{}, nil)
	assert.ErrorIs(t, err, errOrdersToCancelRequired, "submitOrderCancellations should return the cancellation's error with the websocket connected")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	t.Run("by side and type", func(t *testing.T) {
		t.Parallel()
		buy := testOrderRecord{id: 101, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
		sell := testOrderRecord{id: 102, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 120}
		stop := testOrderRecord{id: 103, symbol: "tBTCUSD", orderType: "EXCHANGE STOP", status: "ACTIVE", amount: 1, original: 1, price: 130}
		margined := testOrderRecord{id: 104, symbol: "tETHUSD", orderType: "LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 200}
		stopLimit := testOrderRecord{id: 105, symbol: "tBTCUSD", orderType: "EXCHANGE STOP LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 90, auxiliaryLimit: 85}
		spotPair := currency.NewBTCUSD()
		for _, tc := range []struct {
			name string
			c    order.Cancel
			exp  []testOrderRecord
		}{
			{name: "buys", c: order.Cancel{AssetType: asset.Spot, Side: order.Buy}, exp: []testOrderRecord{buy, stop}},
			{name: "bids", c: order.Cancel{AssetType: asset.Spot, Side: order.Bid}, exp: []testOrderRecord{buy, stop}},
			{name: "longs", c: order.Cancel{AssetType: asset.Spot, Side: order.Long}, exp: []testOrderRecord{buy, stop}},
			{name: "sells", c: order.Cancel{AssetType: asset.Spot, Side: order.Sell}, exp: []testOrderRecord{sell, stopLimit}},
			{name: "asks", c: order.Cancel{AssetType: asset.Spot, Side: order.Ask}, exp: []testOrderRecord{sell, stopLimit}},
			{name: "shorts", c: order.Cancel{AssetType: asset.Spot, Side: order.Short}, exp: []testOrderRecord{sell, stopLimit}},
			{name: "any side", c: order.Cancel{AssetType: asset.Spot, Side: order.AnySide}, exp: []testOrderRecord{buy, sell, stop, stopLimit}},
			{name: "limit orders", c: order.Cancel{AssetType: asset.Spot, Type: order.Limit}, exp: []testOrderRecord{buy, sell}},
			{name: "stop orders", c: order.Cancel{AssetType: asset.Spot, Type: order.Stop}, exp: []testOrderRecord{stop}},
			{name: "stop limit orders", c: order.Cancel{AssetType: asset.Spot, Type: order.StopLimit}, exp: []testOrderRecord{stopLimit}},
			{name: "any type", c: order.Cancel{AssetType: asset.Spot, Type: order.AnyType}, exp: []testOrderRecord{buy, sell, stop, stopLimit}},
			{name: "a side and type", c: order.Cancel{AssetType: asset.Spot, Side: order.Buy, Type: order.Limit}, exp: []testOrderRecord{buy}},
			{name: "a side of a pair", c: order.Cancel{AssetType: asset.Spot, Pair: spotPair, Side: order.Sell}, exp: []testOrderRecord{sell, stopLimit}},
			{name: "a type of a pair", c: order.Cancel{AssetType: asset.Spot, Pair: spotPair, Type: order.Stop}, exp: []testOrderRecord{stop}},
			{name: "a side of every asset", c: order.Cancel{AssetType: asset.All, Side: order.Sell}, exp: []testOrderRecord{sell, margined, stopLimit}},
			{name: "a side with no asset", c: order.Cancel{Side: order.Sell}, exp: []testOrderRecord{sell, margined, stopLimit}},
			{name: "a type of every asset", c: order.Cancel{AssetType: asset.All, Type: order.Limit}, exp: []testOrderRecord{buy, sell, margined}},
			{name: "a type with no asset", c: order.Cancel{Type: order.Stop}, exp: []testOrderRecord{stop}},
			{name: "a side and type matching nothing", c: order.Cancel{AssetType: asset.Spot, Side: order.Sell, Type: order.Stop}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, cancellations := newCancelTestExchange(t, []testOrderRecord{buy, sell, stop, margined, stopLimit}, tc.exp)
				resp, err := ex.CancelAllOrders(t.Context(), &tc.c)
				require.NoError(t, err, "CancelAllOrders must not error")
				status := map[string]string{}
				var ids []any
				for i := range tc.exp {
					status[strconv.FormatUint(tc.exp[i].id, 10)] = order.PendingCancel.String()
					ids = append(ids, float64(tc.exp[i].id))
				}
				assert.Equal(t, status, resp.Status, "CancelAllOrders should report the orders of the side and type asked for")
				if len(ids) == 0 {
					assert.Empty(t, cancellations(), "CancelAllOrders should send no cancellation when nothing matches")
					return
				}
				assert.Equal(t, []map[string]any{{"id": ids}}, cancellations(), "CancelAllOrders should cancel only the orders of the side and type asked for, by their IDs")
			})
		}
		refused, cancellations := newCancelTestExchange(t, []testOrderRecord{buy}, []testOrderRecord{buy})
		_, err := refused.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Side: order.ClosePosition})
		assert.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelAllOrders should refuse a side no order has")
		assert.Empty(t, cancellations(), "CancelAllOrders should cancel nothing for a side it refuses")
	})

	// Beside the scenario above, as when it was a test of its own
	t.Run("cancellations", func(t *testing.T) {
		t.Parallel()
		spot := testOrderRecord{id: 1747566428, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 10000}
		derivative := testOrderRecord{id: 1747566430, symbol: "tBTCF0:USTF0", orderType: "LIMIT", status: "ACTIVE", amount: 0.002, original: 0.002, price: 70000, meta: `{"$F33":20}`}
		late := testOrderRecord{id: 1747566431, symbol: "tETHUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 2000}
		pending, unknown := order.PendingCancel.String(), order.UnknownStatus.String()
		for _, tc := range []struct {
			name          string
			c             *order.Cancel
			open, replied []testOrderRecord
			exp           map[string]string
			expSent       []map[string]any
		}{
			{
				// The cancel all orders RPC sends no request; an order placed since the listing is cancelled too
				name: "nil", open: []testOrderRecord{spot, derivative}, replied: []testOrderRecord{spot, derivative, late},
				exp: map[string]string{"1747566428": pending, "1747566430": pending, "1747566431": pending}, expSent: []map[string]any{{"all": 1.0}},
			},
			{
				name: "empty, an order left out of the reply", c: &order.Cancel{}, open: []testOrderRecord{spot, derivative}, replied: []testOrderRecord{spot},
				exp: map[string]string{"1747566428": pending, "1747566430": unknown}, expSent: []map[string]any{{"all": 1.0}},
			},
			{name: "empty, nothing open", c: &order.Cancel{}, exp: map[string]string{}, expSent: []map[string]any{{"all": 1.0}}},
			{
				name: "every asset", c: &order.Cancel{AssetType: asset.All}, open: []testOrderRecord{spot, derivative}, replied: []testOrderRecord{spot, derivative},
				exp: map[string]string{"1747566428": pending, "1747566430": pending}, expSent: []map[string]any{{"all": 1.0}},
			},
			{
				name: "an asset", c: &order.Cancel{AssetType: asset.Spot}, open: []testOrderRecord{spot, derivative}, replied: []testOrderRecord{spot},
				exp: map[string]string{"1747566428": pending}, expSent: []map[string]any{{"id": []any{1747566428.0}}},
			},
			{
				name: "a pair", c: &order.Cancel{AssetType: asset.USDTMarginedFutures, Pair: testDerivativePair}, open: []testOrderRecord{spot, derivative},
				replied: []testOrderRecord{derivative}, exp: map[string]string{"1747566430": pending}, expSent: []map[string]any{{"id": []any{1747566430.0}}},
			},
			{
				name: "an asset, the reply naming none of its orders", c: &order.Cancel{AssetType: asset.Spot}, open: []testOrderRecord{spot, derivative},
				exp: map[string]string{"1747566428": unknown}, expSent: []map[string]any{{"id": []any{1747566428.0}}},
			},
			{name: "an asset with nothing open", c: &order.Cancel{AssetType: asset.Margin}, open: []testOrderRecord{spot, derivative}, exp: map[string]string{}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, cancellations := newCancelTestExchange(t, tc.open, tc.replied)
				resp, err := ex.CancelAllOrders(t.Context(), tc.c)
				require.NoError(t, err, "CancelAllOrders must not error")
				assert.Equal(t, tc.exp, resp.Status, "CancelAllOrders should report the orders Bitfinex accepted pending cancellation, and any it left out unknown")
				assert.Equal(t, tc.expSent, cancellations(), "CancelAllOrders should cancel every open order through Bitfinex's own cancellation of them all, and an asset's or pair's by their IDs")
			})
		}
		_, err := newResponseServerExchange(t, `["error",10001,"orders: invalid"]`).CancelAllOrders(t.Context(), nil)
		var apiErr *APIError
		assert.ErrorAs(t, err, &apiErr, "CancelAllOrders should return the error Bitfinex reports listing the open orders")
		unnumbered, _ := newCancelTestExchange(t, []testOrderRecord{{symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 10000}}, nil)
		_, err = unnumbered.CancelAllOrders(t.Context(), nil)
		assert.ErrorIs(t, err, errOrderIDRequired, "CancelAllOrders should refuse an open order without an ID")
		refused, cancellations := newCancelTestExchange(t, []testOrderRecord{spot}, []testOrderRecord{spot})
		_, err = refused.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.All + 1})
		assert.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders should refuse an asset that is not one, rather than cancel every order")
		assert.Empty(t, cancellations(), "CancelAllOrders should cancel nothing for an asset it refuses")

		skipLiveUnlessCanManipulate(t)
		ex := newWrapperTestExchange(t)
		if !mockTests {
			placeLiveTestOrder(t)
		}
		resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSD()})
		require.NoError(t, err, "CancelAllOrders must not error")
		require.NotEmpty(t, resp.Status, "CancelAllOrders must cancel the open spot order")
		for id, status := range resp.Status {
			assert.Equalf(t, order.PendingCancel.String(), status, "CancelAllOrders should report order %s pending cancellation until it closes", id)
		}
	})

	t.Run("a pair of every asset", func(t *testing.T) {
		t.Parallel()
		testListingPair := currency.NewPair(currency.NewCode("TESTBTCF0"), currency.NewCode("TESTUSDTF0"))
		fourLetterPair := currency.NewPair(currency.DOGE, currency.USD)
		// Each asset a pair trades has a buy and a sell of each type, one of them twice side by side, read in an order its IDs
		// do not sort to, so no order of an asset, side or type goes unread, uncancelled or unfiltered, and none is merged
		// with or reordered past another; ETH-USD's order is on another symbol. Each order's pair, asset, side and type are
		// given beside it rather than read from it, as the wrapper reads them
		open := []struct {
			record testOrderRecord
			pair   currency.Pair
			a      asset.Item
			side   order.Side
			typ    order.Type
		}{
			{testOrderRecord{id: 201, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}, currency.NewBTCUSD(), asset.Spot, order.Buy, order.Limit},
			{testOrderRecord{id: 230, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 3, original: 3, price: 95}, currency.NewBTCUSD(), asset.Spot, order.Buy, order.Limit},
			{testOrderRecord{id: 208, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: -2, original: -2, price: 130}, currency.NewBTCUSD(), asset.Spot, order.Sell, order.Limit},
			{testOrderRecord{id: 202, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 120}, currency.NewBTCUSD(), asset.Margin, order.Sell, order.Limit},
			{testOrderRecord{id: 231, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: -3, original: -3, price: 125}, currency.NewBTCUSD(), asset.Margin, order.Sell, order.Limit},
			{testOrderRecord{id: 210, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 90}, currency.NewBTCUSD(), asset.Margin, order.Buy, order.Limit},
			{testOrderRecord{id: 209, symbol: "tBTCUSD", orderType: "EXCHANGE STOP", status: "ACTIVE", amount: -1, original: -1, price: 80}, currency.NewBTCUSD(), asset.Spot, order.Sell, order.Stop},
			{testOrderRecord{id: 211, symbol: "tBTCUSD", orderType: "STOP", status: "ACTIVE", amount: 1, original: 1, price: 150}, currency.NewBTCUSD(), asset.Margin, order.Buy, order.Stop},
			{testOrderRecord{id: 232, symbol: "tBTCUSD", orderType: "EXCHANGE STOP", status: "ACTIVE", amount: 1, original: 1, price: 160}, currency.NewBTCUSD(), asset.Spot, order.Buy, order.Stop},
			{testOrderRecord{id: 233, symbol: "tBTCUSD", orderType: "STOP", status: "ACTIVE", amount: -1, original: -1, price: 70}, currency.NewBTCUSD(), asset.Margin, order.Sell, order.Stop},
			{testOrderRecord{id: 203, symbol: "tETHUSD", orderType: "LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 200}, currency.NewPair(currency.ETH, currency.USD), asset.Margin, order.Sell, order.Limit},
			{testOrderRecord{id: 234, symbol: "tBTCF0:USTF0", orderType: "LIMIT", status: "ACTIVE", amount: 0.3, original: 0.3, price: 69000, meta: `{"$F33":10}`}, testDerivativePair, asset.USDTMarginedFutures, order.Buy, order.Limit},
			{testOrderRecord{id: 204, symbol: "tBTCF0:USTF0", orderType: "LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 70000, meta: `{"$F33":10}`}, testDerivativePair, asset.USDTMarginedFutures, order.Buy, order.Limit},
			{testOrderRecord{id: 212, symbol: "tBTCF0:USTF0", orderType: "LIMIT", status: "ACTIVE", amount: -0.2, original: -0.2, price: 75000, meta: `{"$F33":10}`}, testDerivativePair, asset.USDTMarginedFutures, order.Sell, order.Limit},
			{testOrderRecord{id: 213, symbol: "tBTCF0:USTF0", orderType: "STOP", status: "ACTIVE", amount: -0.1, original: -0.1, price: 65000, meta: `{"$F33":10}`}, testDerivativePair, asset.USDTMarginedFutures, order.Sell, order.Stop},
			{testOrderRecord{id: 235, symbol: "tBTCF0:USTF0", orderType: "STOP", status: "ACTIVE", amount: 0.1, original: 0.1, price: 80000, meta: `{"$F33":10}`}, testDerivativePair, asset.USDTMarginedFutures, order.Buy, order.Stop},
			{testOrderRecord{id: 236, symbol: "tETHF0:BTCF0", orderType: "LIMIT", status: "ACTIVE", amount: 2, original: 2, price: 0.025, meta: `{"$F33":10}`}, testCoinMarginedPair, asset.CoinMarginedFutures, order.Buy, order.Limit},
			{testOrderRecord{id: 205, symbol: "tETHF0:BTCF0", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 0.03, meta: `{"$F33":10}`}, testCoinMarginedPair, asset.CoinMarginedFutures, order.Buy, order.Limit},
			{testOrderRecord{id: 214, symbol: "tETHF0:BTCF0", orderType: "LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 0.04, meta: `{"$F33":10}`}, testCoinMarginedPair, asset.CoinMarginedFutures, order.Sell, order.Limit},
			{testOrderRecord{id: 215, symbol: "tETHF0:BTCF0", orderType: "STOP", status: "ACTIVE", amount: 1, original: 1, price: 0.05, meta: `{"$F33":10}`}, testCoinMarginedPair, asset.CoinMarginedFutures, order.Buy, order.Stop},
			{testOrderRecord{id: 237, symbol: "tETHF0:BTCF0", orderType: "STOP", status: "ACTIVE", amount: -1, original: -1, price: 0.02, meta: `{"$F33":10}`}, testCoinMarginedPair, asset.CoinMarginedFutures, order.Sell, order.Stop},
			{testOrderRecord{id: 238, symbol: "tTESTBTCF0:TESTUSDTF0", orderType: "LIMIT", status: "ACTIVE", amount: 0.2, original: 0.2, price: 69000, meta: `{"$F33":10}`}, testListingPair, asset.USDTMarginedFutures, order.Buy, order.Limit},
			{testOrderRecord{id: 206, symbol: "tTESTBTCF0:TESTUSDTF0", orderType: "LIMIT", status: "ACTIVE", amount: 0.1, original: 0.1, price: 70000, meta: `{"$F33":10}`}, testListingPair, asset.USDTMarginedFutures, order.Buy, order.Limit},
			{testOrderRecord{id: 239, symbol: "tTESTBTCF0:TESTUSDTF0", orderType: "LIMIT", status: "ACTIVE", amount: -0.1, original: -0.1, price: 75000, meta: `{"$F33":10}`}, testListingPair, asset.USDTMarginedFutures, order.Sell, order.Limit},
			{testOrderRecord{id: 216, symbol: "tTESTBTCF0:TESTUSDTF0", orderType: "STOP", status: "ACTIVE", amount: -0.1, original: -0.1, price: 65000, meta: `{"$F33":10}`}, testListingPair, asset.USDTMarginedFutures, order.Sell, order.Stop},
			{testOrderRecord{id: 240, symbol: "tTESTBTCF0:TESTUSDTF0", orderType: "STOP", status: "ACTIVE", amount: 0.1, original: 0.1, price: 80000, meta: `{"$F33":10}`}, testListingPair, asset.USDTMarginedFutures, order.Buy, order.Stop},
			{testOrderRecord{id: 241, symbol: "tDOGE:USD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 200, original: 200, price: 0.19}, fourLetterPair, asset.Spot, order.Buy, order.Limit},
			{testOrderRecord{id: 207, symbol: "tDOGE:USD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 100, original: 100, price: 0.2}, fourLetterPair, asset.Spot, order.Buy, order.Limit},
			{testOrderRecord{id: 242, symbol: "tDOGE:USD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: -100, original: -100, price: 0.3}, fourLetterPair, asset.Spot, order.Sell, order.Limit},
			{testOrderRecord{id: 217, symbol: "tDOGE:USD", orderType: "EXCHANGE STOP", status: "ACTIVE", amount: -100, original: -100, price: 0.1}, fourLetterPair, asset.Spot, order.Sell, order.Stop},
			{testOrderRecord{id: 243, symbol: "tDOGE:USD", orderType: "EXCHANGE STOP", status: "ACTIVE", amount: 100, original: 100, price: 0.4}, fourLetterPair, asset.Spot, order.Buy, order.Stop},
			{testOrderRecord{id: 245, symbol: "tDOGE:USD", orderType: "LIMIT", status: "ACTIVE", amount: 100, original: 100, price: 0.18}, fourLetterPair, asset.Margin, order.Buy, order.Limit},
			{testOrderRecord{id: 244, symbol: "tDOGE:USD", orderType: "LIMIT", status: "ACTIVE", amount: 50, original: 50, price: 0.17}, fourLetterPair, asset.Margin, order.Buy, order.Limit},
			{testOrderRecord{id: 246, symbol: "tDOGE:USD", orderType: "LIMIT", status: "ACTIVE", amount: -100, original: -100, price: 0.31}, fourLetterPair, asset.Margin, order.Sell, order.Limit},
			{testOrderRecord{id: 247, symbol: "tDOGE:USD", orderType: "STOP", status: "ACTIVE", amount: 100, original: 100, price: 0.41}, fourLetterPair, asset.Margin, order.Buy, order.Stop},
			{testOrderRecord{id: 248, symbol: "tDOGE:USD", orderType: "STOP", status: "ACTIVE", amount: -100, original: -100, price: 0.09}, fourLetterPair, asset.Margin, order.Sell, order.Stop},
		}
		records := make([]testOrderRecord, len(open))
		for i := range open {
			records[i] = open[i].record
		}
		for _, p := range []struct {
			name   string
			pair   currency.Pair
			assets []asset.Item
		}{
			{name: "a spot pair", pair: currency.NewBTCUSD(), assets: []asset.Item{asset.Spot, asset.Margin}},
			{name: "a USDT margined pair", pair: testDerivativePair, assets: []asset.Item{asset.USDTMarginedFutures}},
			{name: "a coin margined pair", pair: testCoinMarginedPair, assets: []asset.Item{asset.CoinMarginedFutures}},
			{name: "a test listing's pair", pair: testListingPair, assets: []asset.Item{asset.USDTMarginedFutures}},
			{name: "a four letter currency's pair", pair: fourLetterPair, assets: []asset.Item{asset.Spot, asset.Margin}},
		} {
			for _, a := range append([]asset.Item{asset.All, asset.Empty}, p.assets...) {
				assets := "of " + a.String()
				switch a {
				case asset.All:
					assets = "of every asset"
				case asset.Empty:
					assets = "with no asset"
				}
				// side is the side asked for, by every name for one, and matches the side of the orders it names; no order is a
				// market order
				for _, filter := range []struct {
					name    string
					side    order.Side
					matches order.Side
					typ     order.Type
				}{
					{name: ""},
					{name: ", buys", side: order.Buy, matches: order.Buy},
					{name: ", bids", side: order.Bid, matches: order.Buy},
					{name: ", longs", side: order.Long, matches: order.Buy},
					{name: ", sells", side: order.Sell, matches: order.Sell},
					{name: ", asks", side: order.Ask, matches: order.Sell},
					{name: ", shorts", side: order.Short, matches: order.Sell},
					{name: ", limit orders", typ: order.Limit},
					{name: ", stops", typ: order.Stop},
					{name: ", limit buys", side: order.Buy, matches: order.Buy, typ: order.Limit},
					{name: ", sell stops", side: order.Sell, matches: order.Sell, typ: order.Stop},
					{name: ", market orders", typ: order.Market},
				} {
					var exp []testOrderRecord
					for i := range open {
						o := &open[i]
						if o.pair.Equal(p.pair) && (a == asset.All || a == asset.Empty || o.a == a) &&
							(filter.matches == order.UnknownSide || o.side == filter.matches) && (filter.typ == order.UnknownType || o.typ == filter.typ) {
							exp = append(exp, o.record)
						}
					}
					t.Run(p.name+" "+assets+filter.name, func(t *testing.T) {
						t.Parallel()
						ex, cancellations := newCancelTestExchange(t, records, exp)
						resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: a, Pair: p.pair, Side: filter.side, Type: filter.typ})
						require.NoError(t, err, "CancelAllOrders must not error")
						status := make(map[string]string, len(exp))
						ids := make([]any, len(exp))
						for i := range exp {
							status[strconv.FormatUint(exp[i].id, 10)] = order.PendingCancel.String()
							ids[i] = float64(exp[i].id)
						}
						assert.Equal(t, status, resp.Status, "CancelAllOrders should report the pair's orders of the assets, side and type asked for")
						if len(exp) == 0 {
							assert.Empty(t, cancellations(), "CancelAllOrders should send no cancellation when none of the pair's orders matches")
							return
						}
						assert.Equal(t, []map[string]any{{"id": ids}}, cancellations(), "CancelAllOrders should cancel only the pair's orders of the assets, side and type asked for, by their IDs")
					})
				}
			}
		}
		t.Run("a derivative pair of the other collateral", func(t *testing.T) {
			t.Parallel()
			ex, cancellations := newCancelTestExchange(t, records, nil)
			_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.CoinMarginedFutures, Pair: testDerivativePair})
			assert.ErrorIs(t, err, errPairNotOfAsset, "CancelAllOrders should refuse a pair of another asset than the one asked for")
			assert.Empty(t, cancellations(), "CancelAllOrders should cancel nothing for a pair it refuses")
		})
	})
}

// newPathResponseServerExchange returns an exchange whose REST requests each receive the body given for their path, or
// an empty list
func newPathResponseServerExchange(t *testing.T, bodies map[string]string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path]
		if !ok {
			body = `[]`
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

// newOrderInfoTestExchange returns an exchange whose open orders and order history are those given, which it looks up
// by ID, and whose orders' trades are those given by their paths
func newOrderInfoTestExchange(t *testing.T, open, history []testOrderRecord, trades map[string]string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := trades[r.URL.Path]
		switch {
		case ok:
		case r.URL.Path == "/v2/auth/r/orders", r.URL.Path == "/v2/auth/r/orders/hist":
			var req struct {
				IDs []uint64 `json:"id"`
			}
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "Decoding the order lookup should not error") ||
				!assert.Len(t, req.IDs, 1, "GetOrderInfo should look the order up by its ID") {
				http.Error(w, "unexpected lookup", http.StatusBadRequest)
				return
			}
			orders := open
			if r.URL.Path == "/v2/auth/r/orders/hist" {
				orders = history
			}
			body = testOrderRecords(slices.DeleteFunc(slices.Clone(orders), func(o testOrderRecord) bool { return o.id != req.IDs[0] }))
		default:
			body = `[]`
			if strings.HasSuffix(r.URL.Path, "/trades") {
				break
			}
			http.Error(w, "unexpected path", http.StatusNotFound)
			assert.Failf(t, "the test server should receive only the requests expected", "%s %s", r.Method, r.URL.Path)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

// newOrderReadsTestExchange returns an exchange whose open orders, order history and order trades change from one read
// to the next, each read's body given in turn and the last repeated, and how often the open orders, the history and the
// trades were read
func newOrderReadsTestExchange(t *testing.T, open, history []string, trades ...string) (ex *Exchange, reads func() [3]int) {
	t.Helper()
	var mu sync.Mutex
	var openReads, historyReads, tradeReads int
	next := func(bodies []string, n *int) string {
		*n++
		if len(bodies) == 0 {
			return `[]`
		}
		return bodies[min(*n, len(bodies))-1]
	}
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body string
		switch {
		case r.URL.Path == "/v2/auth/r/orders":
			body = next(open, &openReads)
		case r.URL.Path == "/v2/auth/r/orders/hist":
			body = next(history, &historyReads)
		case strings.HasSuffix(r.URL.Path, "/trades"):
			body = next(trades, &tradeReads)
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
			assert.Failf(t, "the test server should receive only the requests expected", "%s %s", r.Method, r.URL.Path)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	return ex, func() [3]int {
		mu.Lock()
		defer mu.Unlock()
		return [3]int{openReads, historyReads, tradeReads}
	}
}

func TestGetOrderInfo(t *testing.T) { //nolint:tparallel // serial subtests share their exchange or precede later GetOrderInfo calls
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "x", currency.EMPTYPAIR, asset.Empty)
	require.ErrorIs(t, err, errOrderIDRequired, "GetOrderInfo must need a numeric order ID")

	// A derivative sell from the history whose remainder was cancelled after a taker fill charged a fee and a maker fill
	// paid a rebate, which the history reports PARTIALLY FILLED, and a spot market buy charged its fee in the currency it
	// bought
	derivative := testOrderRecord{
		id: 33963608932, symbol: "tBTCF0:USTF0", orderType: "LIMIT", status: "PARTIALLY FILLED @ 69600.0(-0.5), PARTIALLY FILLED @ 69000.0(-0.25)",
		amount: -0.25, original: -1, price: 69000, meta: `{"$F33":20}`,
	}
	spot := testOrderRecord{id: 33963608933, symbol: "tBTCUSD", orderType: "EXCHANGE MARKET", status: "EXECUTED @ 69010.0(0.5)", original: 0.5, price: 69000}
	ex := newOrderInfoTestExchange(t, nil, []testOrderRecord{derivative, spot}, map[string]string{
		"/v2/auth/r/order/tBTCF0:USTF0:33963608932/trades": `[[399251013,"tBTCF0:USTF0",1573485492500,33963608932,-0.5,69600,"LIMIT",null,-1,-2.5,"USTF0",1573485491000],` +
			`[399251014,"tBTCF0:USTF0",1573485493500,33963608932,-0.25,69000,"LIMIT",null,1,0.625,"USTF0",1573485491000]]`,
		"/v2/auth/r/order/tBTCUSD:33963608933/trades": `[[399251015,"tBTCUSD",1573485495500,33963608933,0.5,69010,"EXCHANGE MARKET",69000,-1,-0.001,"BTC",1573485491001]]`,
	})
	created, updated := time.UnixMilli(1678987199446), time.UnixMilli(1678987199448)
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for _, exp := range []*order.Detail{
		{
			Exchange: ex.Name, OrderID: "33963608932", Pair: testDerivativePair, AssetType: asset.USDTMarginedFutures, Side: order.Sell, Type: order.Limit,
			Status: order.PartiallyFilledCancelled, Price: 69000, Amount: 1, RemainingAmount: 0.25, ExecutedAmount: 0.75, Leverage: 20, Fee: 1.875,
			Date: created, LastUpdated: updated,
			Trades: []order.TradeHistory{
				{Price: 69600, Amount: 0.5, Fee: 2.5, FeeAsset: "USTF0", Exchange: ex.Name, TID: "399251013", Type: order.Limit, Side: order.Sell, Timestamp: time.UnixMilli(1573485492500)},
				{Price: 69000, Amount: 0.25, Fee: -0.625, FeeAsset: "USTF0", Exchange: ex.Name, TID: "399251014", Type: order.Limit, Side: order.Sell, Timestamp: time.UnixMilli(1573485493500), IsMaker: true},
			},
		},
		{
			Exchange: ex.Name, OrderID: "33963608933", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Buy, Type: order.Market,
			Status: order.Filled, Price: 69000, Amount: 0.5, ExecutedAmount: 0.5, Fee: 0.001, Date: created, LastUpdated: updated,
			Trades: []order.TradeHistory{
				{Price: 69010, Amount: 0.5, Fee: 0.001, FeeAsset: "BTC", Exchange: ex.Name, TID: "399251015", Type: order.Market, Side: order.Buy, Timestamp: time.UnixMilli(1573485495500)},
			},
		},
	} {
		t.Run(exp.AssetType.String(), func(t *testing.T) {
			got, err := ex.GetOrderInfo(t.Context(), exp.OrderID, exp.Pair, exp.AssetType)
			require.NoErrorf(t, err, "GetOrderInfo must not error for order %s in the history", exp.OrderID)
			assert.Equalf(t, exp, got, "GetOrderInfo should return order %s from the history, closed, with each trade's fee as charged, a rebate negative, its currency and whether it was made", exp.OrderID)
		})
	}

	// An open order's trades, reaching back 10 days, can fall short of the fills its status lists
	beyondTrades := testOrderRecord{
		id: 33963608934, symbol: "tETHUSD", orderType: "EXCHANGE LIMIT", status: "PARTIALLY FILLED @ 100.0(0.25), PARTIALLY FILLED @ 100.0(0.5)",
		amount: 0.25, original: 1, price: 100,
	}
	ex = newOrderInfoTestExchange(t, []testOrderRecord{beyondTrades}, nil, map[string]string{
		"/v2/auth/r/order/tETHUSD:33963608934/trades": `[[399251016,"tETHUSD",1573485495500,33963608934,0.5,100,"EXCHANGE LIMIT",100,1,-0.05,"ETH",0]]`,
	})
	got, err := ex.GetOrderInfo(t.Context(), "33963608934", currency.EMPTYPAIR, asset.Empty)
	require.NoError(t, err, "GetOrderInfo must not error for an open order")
	assert.Equal(t, order.PartiallyFilled, got.Status, "GetOrderInfo should report the open order partially filled")
	assert.Equal(t, [3]float64{1, 0.75, 0.25}, [3]float64{got.Amount, got.ExecutedAmount, got.RemainingAmount},
		"GetOrderInfo should report the open order's whole amount, what it executed by its status and what remains")

	// The order, created at 1678987198000, before its fills, was read at 1678987199448 unless given another update time
	read := func(status string, remaining, original float64, updatedAt string) string {
		return `[` + (&testOrderRecord{
			id: 33963608936, symbol: "tETHUSD", orderType: "EXCHANGE LIMIT", status: status, amount: remaining, original: original, price: 100,
			createdAt: "1678987198000", updatedAt: updatedAt,
		}).String() + `]`
	}
	trade := func(id, at, amount string) string {
		return `[` + id + `,"tETHUSD",` + at + `,33963608936,` + amount + `,100,"EXCHANGE LIMIT",100,1,-0.01,"ETH",0]`
	}
	// Serial, as GetOrderInfo is called again after these cases
	for _, tc := range []struct {
		name                        string
		open, history               []string
		trades                      string
		laterTrades                 []string
		status                      order.Status
		amount, executed, remaining float64
		lastUpdated                 time.Time
		tids                        []string
		openReads, historyReads     int
	}{
		{
			name: "no fill since the order was read", open: []string{read("PARTIALLY FILLED @ 100.0(0.4)", 0.6, 1, "")}, trades: `[` + trade("1", "1678987199000", "0.4") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.4, remaining: 0.6, lastUpdated: updated, tids: []string{"1"}, openReads: 1,
		},
		{
			name:   "a fill since the order was read, read again",
			open:   []string{read("PARTIALLY FILLED @ 100.0(0.4)", 0.6, 1, ""), read("PARTIALLY FILLED @ 100.0(0.4), PARTIALLY FILLED @ 100.0(0.1)", 0.5, 1, "1678987200000")},
			trades: `[` + trade("1", "1678987199000", "0.4") + `,` + trade("2", "1678987200000", "0.1") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.5, remaining: 0.5, lastUpdated: time.UnixMilli(1678987200000), tids: []string{"1", "2"}, openReads: 2,
		},
		{
			name:        "fills since the order was read and since its trades were, read again with its trades",
			open:        []string{read("ACTIVE", 1, 1, ""), read("PARTIALLY FILLED @ 100.0(0.25), PARTIALLY FILLED @ 100.0(0.5)", 0.25, 1, "1678987200001")},
			trades:      `[` + trade("1", "1678987200000", "0.25") + `]`,
			laterTrades: []string{`[` + trade("1", "1678987200000", "0.25") + `,` + trade("2", "1678987200001", "0.5") + `]`},
			status:      order.PartiallyFilled, amount: 1, executed: 0.75, remaining: 0.25, lastUpdated: time.UnixMilli(1678987200001), tids: []string{"1", "2"}, openReads: 2,
		},
		{
			name:        "fills since the order was read and since its trades were, read again with its trades listed newest first",
			open:        []string{read("ACTIVE", 1, 1, ""), read("PARTIALLY FILLED @ 100.0(0.25), PARTIALLY FILLED @ 100.0(0.5)", 0.25, 1, "1678987200001")},
			trades:      `[` + trade("1", "1678987200000", "0.25") + `]`,
			laterTrades: []string{`[` + trade("2", "1678987200001", "0.5") + `,` + trade("1", "1678987200000", "0.25") + `]`},
			status:      order.PartiallyFilled, amount: 1, executed: 0.75, remaining: 0.25, lastUpdated: time.UnixMilli(1678987200001), tids: []string{"1", "2"}, openReads: 2,
		},
		{
			name:    "the order filled since it was read, read again from the history",
			open:    []string{read("ACTIVE", 1, 1, ""), `[]`},
			history: []string{read("EXECUTED @ 100.0(1)", 0, 1, "1678987200000")},
			trades:  `[` + trade("1", "1678987200000", "1") + `]`,
			status:  order.Filled, amount: 1, executed: 1, lastUpdated: time.UnixMilli(1678987200000), tids: []string{"1"}, openReads: 2, historyReads: 1,
		},
		{
			name:    "the order partly filled and cancelled since it was read, read again from the history",
			open:    []string{read("ACTIVE", 1, 1, ""), `[]`},
			history: []string{read("PARTIALLY FILLED @ 100.0(0.25)", 0.75, 1, "1678987200000")},
			trades:  `[` + trade("1", "1678987200000", "0.25") + `]`,
			status:  order.PartiallyFilledCancelled, amount: 1, executed: 0.25, remaining: 0.75, lastUpdated: time.UnixMilli(1678987200000), tids: []string{"1"}, openReads: 2, historyReads: 1,
		},
		{
			name:    "a sell partly filled and cancelled since it was read, read again from the history",
			open:    []string{read("ACTIVE", -1, -1, ""), `[]`},
			history: []string{read("PARTIALLY FILLED @ 100.0(-0.25)", -0.75, -1, "1678987200000")},
			trades:  `[` + trade("1", "1678987200000", "-0.25") + `]`,
			status:  order.PartiallyFilledCancelled, amount: 1, executed: 0.25, remaining: 0.75, lastUpdated: time.UnixMilli(1678987200000), tids: []string{"1"}, openReads: 2, historyReads: 1,
		},
		{
			name:    "the order partly filled and cancelled in the millisecond it was read, read again from the history",
			open:    []string{read("ACTIVE", 1, 1, ""), `[]`},
			history: []string{read("PARTIALLY FILLED @ 100.0(0.25)", 0.75, 1, "")},
			trades:  `[` + trade("1", "1678987199448", "0.25") + `]`,
			status:  order.PartiallyFilledCancelled, amount: 1, executed: 0.25, remaining: 0.75, lastUpdated: updated, tids: []string{"1"}, openReads: 2, historyReads: 1,
		},
		{
			name:    "the order, its status cut, filled further and cancelled since it was read, read again from the history",
			open:    []string{read("... 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.2)", 0.5, 1, ""), `[]`},
			history: []string{read("...LLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.25)", 0.25, 1, "1678987200000")},
			trades: `[` + trade("1", "1678987199446", "0.1") + `,` + trade("2", "1678987199447", "0.2") + `,` + trade("3", "1678987199448", "0.2") + `,` +
				trade("4", "1678987200000", "0.25") + `]`,
			status: order.PartiallyFilledCancelled, amount: 1, executed: 0.75, remaining: 0.25, lastUpdated: time.UnixMilli(1678987200000), tids: []string{"1", "2", "3", "4"}, openReads: 2, historyReads: 1,
		},
		{
			name: "a fill its second read predates too, of an order whose status was cut, left to its next read",
			open: []string{read("... 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.2)", 0.3, 0.8, "")},
			trades: `[` + trade("1", "1678987199446", "0.1") + `,` + trade("2", "1678987199447", "0.2") + `,` + trade("3", "1678987199448", "0.2") + `,` +
				trade("4", "1678987200000", "0.3") + `]`,
			status: order.PartiallyFilled, amount: 0.8, executed: 0.5, remaining: 0.3, lastUpdated: updated, tids: []string{"1", "2", "3", "4"}, openReads: 2,
		},
		{
			name: "a fill its second read predates too, left to its next read", open: []string{read("ACTIVE", 1, 1, "")}, trades: `[` + trade("1", "1678987200000", "0.25") + `]`,
			status: order.Active, amount: 1, remaining: 1, lastUpdated: updated, tids: []string{"1"}, openReads: 2,
		},
		{
			name: "a fill made in the millisecond the order was read, which its status does not list, read again",
			open: []string{
				read("PARTIALLY FILLED @ 100.0(0.25)", 0.75, 1, ""),
				read("PARTIALLY FILLED @ 100.0(0.25), PARTIALLY FILLED @ 100.0(0.5)", 0.25, 1, ""),
			},
			trades: `[` + trade("1", "1678987199000", "0.25") + `,` + trade("2", "1678987199448", "0.5") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.75, remaining: 0.25, lastUpdated: updated, tids: []string{"1", "2"}, openReads: 2,
		},
		{
			name: "a fill its status lists in neither read, left to its next read", open: []string{read("PARTIALLY FILLED @ 100.0(0.25)", 0.75, 1, "")},
			trades: `[` + trade("1", "1678987199000", "0.25") + `,` + trade("2", "1678987199448", "0.5") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.25, remaining: 0.75, lastUpdated: updated, tids: []string{"1", "2"}, openReads: 2,
		},
		{
			name: "an order without its update time, whose trades cannot be told from its read", open: []string{read("PARTIALLY FILLED @ 100.0(0.4)", 0.6, 1, "null")},
			trades: `[` + trade("1", "1678987199000", "0.4") + `,` + trade("2", "1678987200000", "0.1") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.4, remaining: 0.6, tids: []string{"1", "2"}, openReads: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reread, reads := newOrderReadsTestExchange(t, tc.open, tc.history, append([]string{tc.trades}, tc.laterTrades...)...)
			info, err := reread.GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
			require.NoErrorf(t, err, "GetOrderInfo must not error for %s", tc.name)
			assert.Equalf(t, tc.status, info.Status, "GetOrderInfo should report the status of %s", tc.name)
			assert.Equalf(t, [3]float64{tc.amount, tc.executed, tc.remaining}, [3]float64{info.Amount, info.ExecutedAmount, info.RemainingAmount},
				"GetOrderInfo should report the whole amount, what was executed and what remains of %s", tc.name)
			assert.Truef(t, tc.lastUpdated.Equal(info.LastUpdated), "GetOrderInfo should report %s last updated at %s, as last read, not %s", tc.name, tc.lastUpdated, info.LastUpdated)
			tids := make([]string, len(info.Trades))
			for i := range info.Trades {
				tids[i] = info.Trades[i].TID
			}
			assert.ElementsMatchf(t, tc.tids, tids, "GetOrderInfo should return the trades of %s as last read", tc.name)
			assert.Equalf(t, [3]int{tc.openReads, tc.historyReads, tc.openReads}, reads(), "GetOrderInfo should read the order as often as %s needs, and its trades after each read", tc.name)
		})
	}
	ex, _ = newOrderReadsTestExchange(t, []string{read("ACTIVE", 1, 1, ""), `["error",10020,"symbol: invalid"]`}, nil, `[`+trade("1", "1678987200000", "0.25")+`]`)
	_, err = ex.GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "GetOrderInfo should return the error Bitfinex reports when reading the order again")
	ex, _ = newOrderReadsTestExchange(t, []string{read("ACTIVE", 1, 1, "")}, nil, `[`+trade("1", "1678987200000", "0.25")+`]`, `["error",10020,"symbol: invalid"]`)
	_, err = ex.GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
	assert.ErrorAs(t, err, &apiErr, "GetOrderInfo should return the error Bitfinex reports when reading the order's trades again")

	// A cut status lists only the latest fills: 0.1 was cut short, so the status lists 0.4 in full. The order, placed for
	// 1, was lowered to 0.8, which Bitfinex's original amount from before the update does not show
	cut := "... 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.2)"
	ago := func(d time.Duration) string { return strconv.FormatInt(time.Now().Add(-d).UnixMilli(), 10) }
	// Serial, as GetOrderInfo is called again after these cases
	for _, tc := range []struct {
		name                        string
		createdAt, trades           string
		closed                      bool
		status                      order.Status
		amount, executed, remaining float64
	}{
		{
			name: "a cut status of an order within its trades' reach", createdAt: ago(2 * time.Hour), status: order.PartiallyFilled, amount: 0.8, executed: 0.5, remaining: 0.3,
			trades: `[` + trade("1", ago(100*time.Minute), "0.1") + `,` + trade("2", ago(90*time.Minute), "0.2") + `,` + trade("3", ago(80*time.Minute), "0.2") + `]`,
		},
		{
			name: "a cut status of an order beyond its trades' reach, whose original amount less what remains is taken", createdAt: ago(11 * 24 * time.Hour),
			status: order.PartiallyFilled, amount: 1, executed: 0.7, remaining: 0.3,
			trades: `[` + trade("2", ago(90*time.Minute), "0.2") + `,` + trade("3", ago(80*time.Minute), "0.2") + `]`,
		},
		{
			name: "a cut status of an order within its trades' reach, from the history", createdAt: ago(2 * time.Hour), closed: true,
			status: order.PartiallyFilledCancelled, amount: 0.8, executed: 0.5, remaining: 0.3,
			trades: `[` + trade("1", ago(100*time.Minute), "0.1") + `,` + trade("2", ago(90*time.Minute), "0.2") + `,` + trade("3", ago(80*time.Minute), "0.2") + `]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := testOrderRecord{id: 33963608936, symbol: "tETHUSD", orderType: "EXCHANGE LIMIT", status: cut, amount: 0.3, original: 1, price: 100, createdAt: tc.createdAt, updatedAt: ago(time.Hour)}
			open, history, historyReads := []string{`[` + o.String() + `]`}, []string(nil), 0
			if tc.closed {
				open, history, historyReads = nil, open, 1
			}
			cutEx, reads := newOrderReadsTestExchange(t, open, history, tc.trades)
			info, err := cutEx.GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
			require.NoErrorf(t, err, "GetOrderInfo must not error for %s", tc.name)
			assert.Equalf(t, tc.status, info.Status, "GetOrderInfo should report the status of %s", tc.name)
			assert.Equalf(t, [3]float64{tc.amount, tc.executed, tc.remaining}, [3]float64{info.Amount, info.ExecutedAmount, info.RemainingAmount},
				"GetOrderInfo should report the whole amount, what was executed and what remains of %s", tc.name)
			assert.Equalf(t, [3]int{1, historyReads, 1}, reads(), "GetOrderInfo should read %s and its trades once, with no fill since", tc.name)
		})
	}

	const refused = `["error",10020,"symbol: invalid"]`
	open := `[` + (&testOrderRecord{id: 33963608936, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}).String() + `]`
	// Serial, as GetOrderInfo is called again after these cases
	for _, tc := range []struct {
		name    string
		bodies  map[string]string
		refused bool
		err     error
	}{
		{name: "the open orders refused", bodies: map[string]string{"/v2/auth/r/orders": refused}, refused: true},
		{name: "the history refused", bodies: map[string]string{"/v2/auth/r/orders/hist": refused}, refused: true},
		{name: "an order of an unknown type", bodies: map[string]string{"/v2/auth/r/orders": `[[33963608936,null,null,"tBTCUSD",null,null,1,1,"NOVEL"]]`}, err: errUnknownOrderType},
		{name: "the trades refused", bodies: map[string]string{"/v2/auth/r/orders": open, "/v2/auth/r/order/tBTCUSD:33963608936/trades": refused}, refused: true},
		{
			name:   "a trade without an amount",
			bodies: map[string]string{"/v2/auth/r/orders": open, "/v2/auth/r/order/tBTCUSD:33963608936/trades": `[[1,"tBTCUSD",1573485495500,33963608936,0,100,"EXCHANGE LIMIT",null,1,0,"USD",0]]`},
			err:    errOrderSideUnresolved,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newPathResponseServerExchange(t, tc.bodies).GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
			if tc.refused {
				var apiErr *APIError
				assert.ErrorAsf(t, err, &apiErr, "GetOrderInfo should return the error Bitfinex reports with %s", tc.name)
				return
			}
			assert.ErrorIsf(t, err, tc.err, "GetOrderInfo should return the expected error for %s", tc.name)
		})
	}

	skipLiveWithoutCredentials(t)
	id, p := "33950998275", currency.NewPair(currency.ETH, currency.USD)
	if !mockTests {
		orders, err := e.GetOrdersHistory(t.Context(), &OrdersHistoryRequest{Limit: 1})
		require.NoError(t, err, "GetOrdersHistory must not error")
		if len(orders) == 0 {
			t.Skip("The account has no recent orders whose information could be returned")
		}
		id = strconv.FormatUint(orders[0].ID, 10)
		p, err = pairFromSymbol(orders[0].Symbol)
		require.NoError(t, err, "the order's symbol must be a pair")
	}
	got, err = e.GetOrderInfo(t.Context(), id, p, asset.Empty)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Equal(t, id, got.OrderID, "GetOrderInfo should return the order")
	if mockTests {
		assert.Equal(t, asset.Margin, got.AssetType, "the documented LIMIT order should be margin")
		require.Len(t, got.Trades, 1, "GetOrderInfo must return the order's trades")
		assert.InDelta(t, 0.09868591, got.Fee, 1e-12, "GetOrderInfo should total the order's trading fees as positive")
		_, err = e.GetOrderInfo(t.Context(), id, currency.NewBTCUSD(), asset.Empty)
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should refuse an order on another pair")
		// The cases share e, so each also checks that a call keeps nothing from the last
		for name, pair := range map[string]currency.Pair{"its pair": p, "no pair": currency.EMPTYPAIR} {
			t.Run(name, func(t *testing.T) {
				_, err = e.GetOrderInfo(t.Context(), id, pair, asset.Spot)
				assert.ErrorIsf(t, err, order.ErrOrderNotFound, "GetOrderInfo should refuse an order of another asset given %s", name)
			})
		}
		_, err = e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Empty)
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report an order neither open nor in the history")
	}
}

func TestIsOrderReadBehindTrades(t *testing.T) {
	t.Parallel()
	read := time.UnixMilli(1678987199448)
	trade := func(at time.Time, amount float64) AccountTradeResponse {
		return AccountTradeResponse{ExecutedAt: types.Time(at), ExecutedAmount: amount}
	}
	cut := "... 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2)"
	phraseCut := "...LLED @ 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2)"
	for _, tc := range []struct {
		name   string
		o      *OrderResponse
		trades []AccountTradeResponse
		exp    bool
	}{
		{name: "no trades", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "ACTIVE"}},
		{name: "a fill its status lists", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(-0.25)"}, trades: []AccountTradeResponse{trade(read, -0.25)}},
		{name: "a fill made after the read", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "ACTIVE"}, trades: []AccountTradeResponse{trade(read.Add(time.Millisecond), 0.25)}, exp: true},
		{name: "a fill in the read's millisecond its status does not list", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "ACTIVE"}, trades: []AccountTradeResponse{trade(read, -0.25)}, exp: true},
		{name: "more fills than a cut status lists", o: &OrderResponse{UpdatedAt: types.Time(read), Status: cut}, trades: []AccountTradeResponse{trade(read, 0.1), trade(read, 0.1), trade(read, 0.2)}},
		{name: "a fill an hour after the read of an order whose status was cut", o: &OrderResponse{UpdatedAt: types.Time(read), Status: cut}, trades: []AccountTradeResponse{trade(read.Add(time.Hour), 0.2)}, exp: true},
		{name: "a fill after the read, listed after older ones, of an order whose status was cut", o: &OrderResponse{UpdatedAt: types.Time(read), Status: cut}, trades: []AccountTradeResponse{trade(read, 0.1), trade(read, 0.2), trade(read.Add(time.Millisecond), 0.2)}, exp: true},
		{
			name: "a sell after the read, listed before an older one, of an order whose status was cut", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "... 100.0(-0.1), PARTIALLY FILLED @ 100.0(-0.2)"},
			trades: []AccountTradeResponse{trade(read.Add(time.Millisecond), -0.2), trade(read, -0.2)}, exp: true,
		},
		{
			name: "a fill after the read of an order whose listed fills its trades no longer reach", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(0.25)"},
			trades: []AccountTradeResponse{trade(read.Add(time.Millisecond), 0.25)}, exp: true,
		},
		{
			name: "a fill its status lists, made since the order was created", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(0.25)"},
			trades: []AccountTradeResponse{trade(read.Add(-time.Second), 0.25)},
		},
		{
			name: "a fill of 0.00000001 in the read's millisecond, after one its status lists", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(0.25)"},
			trades: []AccountTradeResponse{trade(read.Add(-time.Second), 0.25), trade(read, 0.00000001)}, exp: true,
		},
		{name: "a fill after the read, listed between older ones, of an order whose status was cut", o: &OrderResponse{UpdatedAt: types.Time(read), Status: cut}, trades: []AccountTradeResponse{trade(read, 0.1), trade(read.Add(time.Millisecond), 0.2), trade(read, 0.2)}, exp: true},
		{name: "a fill a day after the read of an order whose status was cut", o: &OrderResponse{UpdatedAt: types.Time(read), Status: cut}, trades: []AccountTradeResponse{trade(read.Add(24*time.Hour), 0.2)}, exp: true},
		{name: "a fill after the read of an order whose status was cut inside a fill's phrase", o: &OrderResponse{UpdatedAt: types.Time(read), Status: phraseCut}, trades: []AccountTradeResponse{trade(read.Add(time.Millisecond), 0.2)}, exp: true},
		{name: "more fills than a status cut inside a fill's phrase lists", o: &OrderResponse{UpdatedAt: types.Time(read), Status: phraseCut}, trades: []AccountTradeResponse{trade(read, 0.1), trade(read, 0.1), trade(read, 0.2)}},
		{
			name: "fills its status lists of 0.1 and 0.2, whose sum as floats is more than 0.3", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2)"},
			trades: []AccountTradeResponse{trade(read.Add(-time.Second), 0.1), trade(read, 0.2)},
		},
		{
			name: "fewer fills than its status lists, the others beyond its trades' reach", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(0.25), PARTIALLY FILLED @ 100.0(0.5)"},
			trades: []AccountTradeResponse{trade(read, 0.5)},
		},
		{
			name: "a sell in the read's millisecond its status does not list, after one it does", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(-0.25)"},
			trades: []AccountTradeResponse{trade(read.Add(-time.Second), -0.25), trade(read, -0.25)}, exp: true,
		},
		{
			name: "a fill of 0.00000001 in the read's millisecond, after one of 100 its status lists", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "PARTIALLY FILLED @ 100.0(100)"},
			trades: []AccountTradeResponse{trade(read.Add(-time.Second), 100), trade(read, 0.00000001)}, exp: true,
		},
		{name: "more fills than a status cut without a mark lists", o: &OrderResponse{UpdatedAt: types.Time(read), Status: "(0.1), PARTIALLY FILLED @ 100.0(0.2)"}, trades: []AccountTradeResponse{trade(read, 0.1), trade(read, 0.1), trade(read, 0.2)}},
		{name: "a fill after the read of an order without its update time", o: &OrderResponse{Status: "ACTIVE"}, trades: []AccountTradeResponse{trade(read, 0.25)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, isOrderReadBehindTrades(tc.o, tc.trades), "isOrderReadBehindTrades should report whether %s leaves the read behind", tc.name)
		})
	}
}

func TestCutOrderExecutedAmount(t *testing.T) { //nolint:tparallel // serial subtests share trades in order
	t.Parallel()
	read := time.Now().Add(-time.Hour)
	trades := []AccountTradeResponse{
		{ExecutedAt: types.Time(read.Add(-time.Minute)), ExecutedAmount: 0.1},
		{ExecutedAt: types.Time(read.Add(-time.Second)), ExecutedAmount: 0.2},
		{ExecutedAt: types.Time(read), ExecutedAmount: 0.2},
		{ExecutedAt: types.Time(read.Add(time.Second)), ExecutedAmount: 0.3},
	}
	// The order, placed for 1, was lowered to 0.8, so its original amount less what remains, 0.7, is more than it executed
	cut := "... 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2), PARTIALLY FILLED @ 100.0(0.2)"

	// The cases share trades in order, so each also checks that cutOrderExecutedAmount leaves them as they were
	for _, tc := range []struct {
		name   string
		o      *OrderResponse
		trades []AccountTradeResponse
		exp    string
	}{
		{name: "an order within its trades' reach", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.5"},
		{name: "an order created nine and a half days before its read, within its trades' reach", o: &OrderResponse{CreatedAt: types.Time(read.Add(-228 * time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.5"},
		{
			name: "a sell within its trades' reach", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Amount: -0.3, OriginalAmount: -1, Status: "... 100.0(-0.1), PARTIALLY FILLED @ 100.0(-0.2), PARTIALLY FILLED @ 100.0(-0.2)"},
			trades: []AccountTradeResponse{
				{ExecutedAt: types.Time(read.Add(-time.Minute)), ExecutedAmount: -0.1},
				{ExecutedAt: types.Time(read.Add(-time.Second)), ExecutedAmount: -0.2},
				{ExecutedAt: types.Time(read), ExecutedAmount: -0.2},
				{ExecutedAt: types.Time(read.Add(time.Second)), ExecutedAmount: -0.3},
			},
			exp: "0.5",
		},
		{
			name: "an order within its trades' reach whose newest trade is listed first", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut},
			trades: []AccountTradeResponse{trades[3], trades[2], trades[1], trades[0]}, exp: "0.5",
		},
		{
			name: "an order within its trades' reach whose fills of 0.1 and 0.2 add up exactly", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: "... 100.0(0.1), PARTIALLY FILLED @ 100.0(0.2)"},
			trades: trades[:2], exp: "0.3",
		},
		{name: "an order beyond its trades' reach", o: &OrderResponse{CreatedAt: types.Time(read.Add(-11 * 24 * time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.7"},
		{name: "an order beyond its trades' reach whose trades count more than its estimate", o: &OrderResponse{UpdatedAt: types.Time(read), Amount: 0.9, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.5"},
		{name: "an order within its trades' reach whose trades count less than its status lists", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades[:1], exp: "0.7"},
		{name: "an order without its update time", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := cutOrderExecutedAmount(tc.o, tc.trades)
			assert.Truef(t, decimal.MustFromString(tc.exp).Equal(got), "cutOrderExecutedAmount should report %s executed for %s, got %s", tc.exp, tc.name, got)
		})
	}
}

func TestOrderByID(t *testing.T) {
	t.Parallel()
	open := testOrderRecord{id: 1, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	other := testOrderRecord{id: 2, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	closed := testOrderRecord{id: 3, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "EXECUTED @ 100.0(1)", original: 1, price: 100}
	// Serial, as orderByID is called again after these cases
	for _, tc := range []struct {
		name          string
		open, history []string
		id            uint64
		closed        bool
		err           error
	}{
		{name: "an open order", open: []string{testOrderRecords([]testOrderRecord{other, open})}, id: 1},
		{name: "an order from the history", open: []string{testOrderRecords([]testOrderRecord{other})}, history: []string{testOrderRecords([]testOrderRecord{closed})}, id: 3, closed: true},
		{name: "an order neither open nor in the history", history: []string{testOrderRecords([]testOrderRecord{closed})}, id: 4, err: order.ErrOrderNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, _ := newOrderReadsTestExchange(t, tc.open, tc.history, `[]`)
			got, gotClosed, err := ex.orderByID(t.Context(), tc.id)
			require.ErrorIsf(t, err, tc.err, "orderByID must return the expected error for %s", tc.name)
			if tc.err != nil {
				return
			}
			assert.Equalf(t, tc.id, got.ID, "orderByID should return %s", tc.name)
			assert.Equalf(t, tc.closed, gotClosed, "orderByID should report whether %s is from the history", tc.name)
		})
	}
	const refused = `["error",10020,"symbol: invalid"]`
	for _, tc := range []struct {
		name          string
		open, history []string
	}{
		{name: "the open orders", open: []string{refused}},
		{name: "the history", history: []string{refused}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, _ := newOrderReadsTestExchange(t, tc.open, tc.history, `[]`)
			_, _, err := ex.orderByID(t.Context(), 1)
			var apiErr *APIError
			assert.ErrorAsf(t, err, &apiErr, "orderByID should return the error Bitfinex reports reading %s", tc.name)
		})
	}
}

func TestOpenOrderDetails(t *testing.T) { //nolint:tparallel // serial subtests share ex and its recorded paths in order
	t.Parallel()
	spot := testOrderRecord{id: 1, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	marginOrder := testOrderRecord{id: 2, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	var mu sync.Mutex
	var paths []string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		_, err := w.Write([]byte(testOrderRecords([]testOrderRecord{spot, marginOrder})))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	// The cases share ex and its recorded paths in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		p    currency.Pair
		a    asset.Item
		path string
		ids  []string
	}{
		{a: asset.Spot, path: "/v2/auth/r/orders", ids: []string{"1"}},
		{p: currency.NewBTCUSD(), a: asset.Margin, path: "/v2/auth/r/orders/tBTCUSD", ids: []string{"2"}},
		{a: asset.Empty, path: "/v2/auth/r/orders", ids: []string{"1", "2"}},
	} {
		t.Run(subtestName(tc.a.String()), func(t *testing.T) {
			got, err := ex.openOrderDetails(t.Context(), tc.p, tc.a)
			require.NoErrorf(t, err, "openOrderDetails must not error for %s %s", tc.a, tc.p)
			ids := make([]string, len(got))
			for i := range got {
				ids[i] = got[i].OrderID
			}
			assert.Equalf(t, tc.ids, ids, "openOrderDetails should return the open %s orders on %s", tc.a, tc.p)
			mu.Lock()
			assert.Equalf(t, tc.path, paths[len(paths)-1], "openOrderDetails should read the open %s orders on %s", tc.a, tc.p)
			mu.Unlock()
		})
	}
	_, err := ex.openOrderDetails(t.Context(), testFundingUSD, asset.MarginFunding)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "openOrderDetails should refuse funding, which takes offers rather than orders")
	_, err = newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).openOrderDetails(t.Context(), currency.EMPTYPAIR, asset.Spot)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "openOrderDetails should return the error Bitfinex reports")
}

func TestOrderDetails(t *testing.T) { //nolint:tparallel // serial subtests share e
	t.Parallel()
	orders := []OrderResponse{
		{ID: 1, Symbol: "tBTCUSD", Type: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "ACTIVE", Amount: 1, OriginalAmount: 1},
		{ID: 2, Symbol: "tBTCUSD", Type: OrderTypeLimit, Status: "ACTIVE", Amount: -1, OriginalAmount: -1},
	}
	// The cases share e and orders, so each also checks that a call keeps nothing from the last
	for a, ids := range map[asset.Item][]string{asset.Spot: {"1"}, asset.Margin: {"2"}, asset.Empty: {"1", "2"}} {
		t.Run(subtestName(a.String()), func(t *testing.T) {
			got, err := e.orderDetails(orders, a)
			require.NoErrorf(t, err, "orderDetails must not error for %s", a)
			gotIDs := make([]string, len(got))
			for i := range got {
				gotIDs[i] = got[i].OrderID
			}
			assert.Equalf(t, ids, gotIDs, "orderDetails should return the %s orders alone, or every order without an asset", a)
		})
	}
	_, err := e.orderDetails([]OrderResponse{{ID: 3, Symbol: "tBTCUSD", Type: "NOVEL", OriginalAmount: 1}}, asset.Spot)
	assert.ErrorIs(t, err, errUnknownOrderType, "orderDetails should return the error of an order it cannot convert")
}

func TestGetActiveOrders(t *testing.T) { //nolint:tparallel // serial subtests share ex or e in order
	t.Parallel()
	// Bitfinex returns every open order, so the request's pairs, side and type select from them
	ethBuy := testOrderRecord{id: 1, symbol: "tETHUSD", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	btcSell := testOrderRecord{id: 2, symbol: "tBTCUSD", orderType: "LIMIT", status: "ACTIVE", amount: -1, original: -1, price: 70000}
	btcStop := testOrderRecord{id: 3, symbol: "tBTCUSD", orderType: "STOP", status: "ACTIVE", amount: 1, original: 1, price: 80000}
	ex := newResponseServerExchange(t, testOrderRecords([]testOrderRecord{ethBuy, btcSell, btcStop}))
	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		name string
		req  order.MultiOrderRequest
		ids  []string
	}{
		{name: "a pair", req: order.MultiOrderRequest{AssetType: asset.Margin, Pairs: currency.Pairs{currency.NewBTCUSD()}, Side: order.AnySide, Type: order.AnyType}, ids: []string{"2", "3"}},
		{name: "a side", req: order.MultiOrderRequest{AssetType: asset.Margin, Side: order.Sell, Type: order.AnyType}, ids: []string{"2"}},
		{name: "a type", req: order.MultiOrderRequest{AssetType: asset.Margin, Side: order.AnySide, Type: order.Stop}, ids: []string{"3"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			orders, err := ex.GetActiveOrders(t.Context(), &tc.req)
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s", tc.name)
			ids := make([]string, len(orders))
			for i := range orders {
				ids[i] = orders[i].OrderID
			}
			assert.ElementsMatchf(t, tc.ids, ids, "GetActiveOrders should return the open orders of %s alone", tc.name)
		})
	}
	_, err := newRefusingServerExchange(t, `["error",10100,"apikey: invalid"]`).GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Side: order.AnySide, Type: order.AnyType})
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "GetActiveOrders should return the error Bitfinex reports")
	_, err = e.GetActiveOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders should reject a nil request")
	unreadable := testOrderRecord{id: 1, symbol: "tX", orderType: "LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	_, err = newResponseServerExchange(t, testOrderRecords([]testOrderRecord{unreadable})).GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errSymbolUnparsable, "GetActiveOrders should report an order it cannot read")

	skipLiveWithoutCredentials(t)
	// The cases share e in order, so each also checks that a call keeps nothing from the last
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		t.Run(a.String(), func(t *testing.T) {
			orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
			for i := range orders {
				assert.Equalf(t, a, orders[i].AssetType, "GetActiveOrders should return %s orders alone", a)
			}
			if mockTests && a == asset.Margin {
				require.Len(t, orders, 1, "GetActiveOrders must return the open margin order")
				assert.Equal(t, "33950998275", orders[0].OrderID, "GetActiveOrders should return the open order")
			}
		})
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("pages back through every order", func(t *testing.T) {
		t.Parallel()
		// Bitfinex bounds the history by when its orders were created, so each page ends at its predecessor's oldest
		// creation. These orders were updated an hour after they were created, so a page ending at an update time would
		// return the same orders again rather than reach the oldest
		const createdFrom = 1678987199446
		records := make([]testOrderRecord, ordersHistoryPageLimit+1)
		for i := range records {
			createdAt := createdFrom + int64(i)
			records[i] = testOrderRecord{
				id: uint64(i) + 1, symbol: "tETHUSD", orderType: "EXCHANGE LIMIT", status: "CANCELED", amount: 1, original: 1, price: 100,
				createdAt: strconv.FormatInt(createdAt, 10), updatedAt: strconv.FormatInt(createdAt+time.Hour.Milliseconds(), 10),
			}
		}
		// Serial, as the cases share the records slice
		for path, pairs := range map[string]currency.Pairs{
			"/v2/auth/r/orders/hist":         nil,
			"/v2/auth/r/orders/tETHUSD/hist": {currency.NewPair(currency.ETH, currency.USD)},
		} {
			t.Run(subtestName(pairs.Join()), func(t *testing.T) {
				ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req map[string]any
					if !assert.Equalf(t, path, r.URL.Path, "GetOrderHistory should read the history for pairs %s", pairs) ||
						!assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "Decoding the request should not error") {
						http.Error(w, "unexpected request", http.StatusBadRequest)
						return
					}
					end, _ := req["end"].(float64)
					limit, _ := req["limit"].(float64)
					var page []testOrderRecord
					for i := len(records) - 1; i >= 0 && float64(len(page)) < limit; i-- {
						if end == 0 || float64(createdFrom+int64(i)) <= end {
							page = append(page, records[i])
						}
					}
					_, err := w.Write([]byte(testOrderRecords(page)))
					assert.NoError(t, err, "Writing the response should not error")
				}))
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: pairs, Side: order.AnySide, Type: order.AnyType})
				require.NoErrorf(t, err, "GetOrderHistory must not error for pairs %s", pairs)
				assert.Lenf(t, history, len(records), "GetOrderHistory should page back through every order for pairs %s", pairs)
			})
		}
	})

	// Beside the scenario above, as when it was a test of its own
	t.Run("history", func(t *testing.T) {
		t.Parallel()
		// The history reports an order whose remainder was cancelled PARTIALLY FILLED
		partial := testOrderRecord{id: 33963608932, symbol: "tETHUSD", orderType: "EXCHANGE LIMIT", status: "PARTIALLY FILLED @ 100.0(0.25)", amount: 0.75, original: 1, price: 100}
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !assert.Containsf(t, []string{"/v2/auth/r/orders/hist", "/v2/auth/r/orders/tETHUSD/hist"}, r.URL.Path, "GetOrderHistory should read the history, not %s", r.URL.Path) {
				http.Error(w, "unexpected path", http.StatusNotFound)
				return
			}
			_, err := w.Write([]byte(`[` + partial.String() + `]`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		// The cases share ex in order, so each also checks that a call keeps nothing from the last
		for _, pairs := range []currency.Pairs{nil, {currency.NewPair(currency.ETH, currency.USD)}} {
			t.Run(subtestName(pairs.Join()), func(t *testing.T) {
				history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: pairs, Side: order.AnySide, Type: order.AnyType})
				require.NoErrorf(t, err, "GetOrderHistory must not error for pairs %s", pairs)
				require.Lenf(t, history, 1, "GetOrderHistory must return the order for pairs %s", pairs)
				assert.Equalf(t, order.PartiallyFilledCancelled, history[0].Status, "GetOrderHistory should report a partially filled order from the history cancelled, its remainder no longer in the book, for pairs %s", pairs)
			})
		}
		_, err := ex.GetOrderHistory(t.Context(), nil)
		assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory should refuse a nil request")
		_, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{testDerivativePair}, Side: order.AnySide, Type: order.AnyType})
		assert.ErrorIs(t, err, errPairNotOfAsset, "GetOrderHistory should refuse a pair the asset does not trade")
		unknownType := newResponseServerExchange(t, `[[33963608937,null,null,"tETHUSD",null,null,1,1,"NOVEL"]]`)
		// The cases share unknownType in order, so each also checks that a call keeps nothing from the last
		for _, pairs := range []currency.Pairs{nil, {currency.NewPair(currency.ETH, currency.USD)}} {
			t.Run("errors for "+subtestName(pairs.Join()), func(t *testing.T) {
				req := &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: pairs, Side: order.AnySide, Type: order.AnyType}
				_, err = newRefusingServerExchange(t, `["error",10020,"symbol: invalid"]`).GetOrderHistory(t.Context(), req)
				var apiErr *APIError
				assert.ErrorAsf(t, err, &apiErr, "GetOrderHistory should return the error Bitfinex reports for pairs %s", pairs)
				_, err = unknownType.GetOrderHistory(t.Context(), req)
				assert.ErrorIsf(t, err, errUnknownOrderType, "GetOrderHistory should refuse an order of an unknown type for pairs %s", pairs)
			})
		}

		skipLiveWithoutCredentials(t)
		ctx := mockRequestContext(t)
		// The cases share e in order, so each also checks that a call keeps nothing from the last
		for _, pairs := range []currency.Pairs{nil, {currency.NewBTCUSD()}} {
			t.Run("account history for "+subtestName(pairs.Join()), func(t *testing.T) {
				orders, err := e.GetOrderHistory(ctx, &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: pairs, Side: order.AnySide, Type: order.AnyType})
				require.NoErrorf(t, err, "GetOrderHistory must not error for pairs %s", pairs)
				for i := range orders {
					assert.Equal(t, asset.Spot, orders[i].AssetType, "GetOrderHistory should return spot orders alone")
				}
			})
		}
	})
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must need a currency")
	skipLiveWithoutCredentials(t)
	got, err := e.GetDepositAddress(t.Context(), currency.BTC, WalletTypeExchange, "")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.NotEmpty(t, got.Address, "GetDepositAddress should return an address")
	assert.Equal(t, "bitcoin", got.Chain, "GetDepositAddress should return the method the address deposits over")
}

func TestGetDepositAddressTagsAndChains(t *testing.T) {
	t.Parallel()
	var deposits []string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/v2/conf/pub:map:tx:method":
			body = `[[["BITCOIN",["BTC"]],["RIPPLE",["XRP"]],["TETHERUSE",["UST"]],["TETHERUSX",["UST"]]]]`
		case "/v2/auth/w/deposit/address":
			raw, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "Reading the request should not error")
			deposits = append(deposits, string(raw))
			body = `[1568738594687,"acc_dep",null,null,[null,"RIPPLE","XRP",null,"2011546049","rLW9gnQo7BQhU6igk5keqYnH3TVrCxGRzm"],null,"SUCCESS","success"]`
		default:
			assert.Failf(t, "the test server should receive only the requests expected", "%s %s", r.Method, r.URL.Path)
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	got, err := ex.GetDepositAddress(t.Context(), currency.XRP, "", "")
	require.NoError(t, err, "GetDepositAddress must not error for a currency with a tag")
	assert.Equal(t, &deposit.Address{Address: "rLW9gnQo7BQhU6igk5keqYnH3TVrCxGRzm", Tag: "2011546049", Chain: "ripple"}, got, "a tagged currency's address should be its pool address, with its tag")
	assert.Equal(t, []string{`{"wallet":"funding","method":"ripple"}`}, deposits, "GetDepositAddress should deposit into the funding wallet by default")

	_, err = ex.GetDepositAddress(t.Context(), currency.USDT, "", "")
	assert.ErrorIs(t, err, errTransferMethodNeeded, "GetDepositAddress should need a chain for a currency with several")
	chains, err := ex.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err, "GetAvailableTransferChains must not error for USDT")
	assert.Equal(t, []string{"tetheruse", "tetherusx"}, chains, "USDT, which Bitfinex names UST, should have both its chains")
	_, err = ex.GetDepositAddress(t.Context(), currency.BTC, "", "dogecoin")
	assert.ErrorIs(t, err, errTransferMethodUnknown, "GetDepositAddress should refuse a chain the currency has not")
	got, err = ex.GetDepositAddress(t.Context(), currency.USDT, "", "TetherUSX")
	require.NoError(t, err, "GetDepositAddress must not error for a chain the currency has")
	assert.Equal(t, "tetherusx", got.Chain, "GetDepositAddress should return the chain the address deposits over")
	assert.JSONEq(t, `{"wallet":"funding","method":"tetherusx"}`, deposits[len(deposits)-1], "GetDepositAddress should deposit over the chain named, as Bitfinex names it")

	var apiErr *APIError
	_, err = newPathResponseServerExchange(t, map[string]string{
		"/v2/conf/pub:map:tx:method": `[[["RIPPLE",["XRP"]]]]`,
		"/v2/auth/w/deposit/address": `["error",10100,"apikey: invalid"]`,
	}).GetDepositAddress(t.Context(), currency.XRP, "", "")
	assert.ErrorAs(t, err, &apiErr, "GetDepositAddress should return the error Bitfinex reports")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must need a currency")
	chains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	assert.Contains(t, chains, "bitcoin", "BTC should be transferred over bitcoin")
	_, err = e.GetAvailableTransferChains(t.Context(), currency.NewCode("NOPE"))
	assert.ErrorIs(t, err, errTransferMethodUnknown, "GetAvailableTransferChains should report a currency without a method")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) { //nolint:tparallel // the folded scenarios run together before the test's own call
	t.Parallel()
	// The scenarios run together, as they could when they were tests of their own, and before this test's own call
	t.Run("scenarios", func(t *testing.T) {
		t.Run("Lightning invoice", func(t *testing.T) {
			t.Parallel()
			var sent []string
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := `[[["LNX",["LNX"]]]]`
				if r.URL.Path == "/v2/auth/w/withdraw" {
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "Reading the withdrawal's body should not error")
					sent = append(sent, string(body))
					response = `[1568742390999,"acc_wd-req",null,null,[13080092,null,"lnx",null,"exchange",0.001,null,null,0],null,"SUCCESS","Your withdrawal request has been successfully submitted."]`
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err, "Writing the response should not error")
			}))
			req := &withdraw.Request{Exchange: ex.Name, Currency: currency.NewCode("LNX"), Amount: 0.001, Type: withdraw.Crypto, Crypto: withdraw.CryptoRequest{Address: "lnbc1invoice"}}
			resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), req)
			require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error for a Lightning withdrawal")
			assert.Equal(t, "13080092", resp.ID, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")
			require.Len(t, sent, 1, "WithdrawCryptocurrencyFunds must send one withdrawal")
			assert.JSONEq(t, `{"wallet":"exchange","method":"lnx","invoice":"lnbc1invoice"}`, sent[0], "a Lightning withdrawal should send its invoice in place of an address and an amount")
		})

		t.Run("travel rule", func(t *testing.T) {
			t.Parallel()
			var sent []byte
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := `[[["ETHEREUM",["ETH"]]]]`
				if r.URL.Path == "/v2/auth/w/withdraw" {
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "Reading the withdrawal's body should not error")
					sent = body
					response = `[1568742390999,"acc_wd-req",null,null,[13080092,null,"ethereum",null,"exchange",0.01,null,null,0.00135],null,"SUCCESS","Your withdrawal request has been successfully submitted."]`
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err, "Writing the response should not error")
			}))
			w := &withdraw.Request{
				Exchange: ex.Name, Currency: currency.ETH, Amount: 0.01, Description: "test", Type: withdraw.Crypto,
				Crypto: withdraw.CryptoRequest{Address: "0x6C20eBD2222CE0F80971A5D2C4Da29e70a0B9b23"},
				Travel: withdraw.TravelRule{BeneficiaryName: "Mary Ann Smith", BeneficiaryFinancialInstitution: "Strawb Exchange Ltd"},
			}
			_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), w)
			require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
			assert.JSONEq(t, `{"wallet":"exchange","method":"ethereum","amount":"0.01","address":"0x6C20eBD2222CE0F80971A5D2C4Da29e70a0B9b23","note":"test","travel_rule_tos":true,"vasp_name":"Strawb Exchange Ltd","dest_firstname":"Mary Ann","dest_lastname":"Smith"}`, string(sent), "a withdrawal should send its beneficiary's names and provider")
		})

		t.Run("named chain", func(t *testing.T) {
			t.Parallel()
			var sent []string
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := `[[["TETHERUSE",["UST"]],["TETHERUSX",["UST"]]]]`
				if r.URL.Path == "/v2/auth/w/withdraw" {
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "Reading the withdrawal's body should not error")
					sent = append(sent, string(body))
					response = `[1568742390999,"acc_wd-req",null,null,[13080092,null,"tetherusx",null,"exchange",1,null,null,0.00135],null,"SUCCESS","Your withdrawal request has been successfully submitted."]`
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err, "Writing the response should not error")
			}))
			req := &withdraw.Request{Exchange: ex.Name, Currency: currency.USDT, Amount: 1, Type: withdraw.Crypto, Crypto: withdraw.CryptoRequest{Address: "TQ1", Chain: "TetherUSX"}}
			resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), req)
			require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error for a chain the currency has")
			assert.Equal(t, "13080092", resp.ID, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")
			require.Len(t, sent, 1, "WithdrawCryptocurrencyFunds must send one withdrawal")
			assert.JSONEq(t, `{"wallet":"exchange","method":"tetherusx","amount":"1","address":"TQ1"}`, sent[0], "a withdrawal should go over the chain named, as Bitfinex names it")
		})

		t.Run("failures", func(t *testing.T) {
			t.Parallel()
			_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
			require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must reject a nil request")
			ex := newPathResponseServerExchange(t, map[string]string{
				"/v2/conf/pub:map:tx:method": `[[["TETHERUSE",["UST"]],["TETHERUSX",["UST"]]]]`,
				"/v2/auth/w/withdraw":        `["error",10100,"apikey: invalid"]`,
			})
			req := withdraw.Request{Exchange: ex.Name, Currency: currency.USDT, Amount: 1, Type: withdraw.Crypto, Crypto: withdraw.CryptoRequest{Address: "TQ1", Chain: "dogecoin"}}
			_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &req)
			assert.ErrorIs(t, err, errTransferMethodUnknown, "WithdrawCryptocurrencyFunds should refuse a chain the currency has not")
			req.Crypto.Chain = "tetherusx"
			req.Travel.BeneficiaryName = "Satoshi"
			_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &req)
			assert.ErrorIs(t, err, errTravelRuleNameIncomplete, "WithdrawCryptocurrencyFunds should refuse a beneficiary without a last name")
			req.Travel.BeneficiaryName = ""
			var apiErr *APIError
			_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &req)
			assert.ErrorAs(t, err, &apiErr, "WithdrawCryptocurrencyFunds should return the error Bitfinex reports")
		})
	})

	if !mockTests {
		t.Skip("A withdrawal cannot be undone")
	}
	resp, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name, Currency: currency.ETH, Amount: 0.01, Description: "test", Type: withdraw.Crypto,
		Crypto: withdraw.CryptoRequest{Address: "0x6C20eBD2222CE0F80971A5D2C4Da29e70a0B9b23"},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, &withdraw.ExchangeResponse{ID: "13080092", Status: NotificationStatusSuccess}, resp, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")
}

func TestWithdrawalTravelRule(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		travel withdraw.TravelRule
		exp    *TravelRuleDetails
		err    error
	}{
		{name: "none"},
		{name: "self", travel: withdraw.TravelRule{IsSelf: true, BeneficiaryName: "John Doe"}, exp: &TravelRuleDetails{BeneficiarySelf: true}},
		{name: "provider", travel: withdraw.TravelRule{BeneficiaryFinancialInstitution: "Strawb Exchange Ltd"}, exp: &TravelRuleDetails{VirtualAssetServiceProviderName: "Strawb Exchange Ltd"}},
		{name: "person", travel: withdraw.TravelRule{BeneficiaryName: "  John   Doe "}, exp: &TravelRuleDetails{FirstName: "John", LastName: "Doe"}},
		{name: "one name", travel: withdraw.TravelRule{BeneficiaryName: "Satoshi"}, err: errTravelRuleNameIncomplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := withdrawalTravelRule(&tc.travel)
			require.ErrorIsf(t, err, tc.err, "withdrawalTravelRule must return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "withdrawalTravelRule should return the travel rule details of %s", tc.name)
		})
	}
}

func TestRequestFiatWithdrawalPayload(t *testing.T) { //nolint:tparallel // serial subtests share ex and the payload it records
	t.Parallel()
	var payload map[string]any
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/withdraw", r.URL.Path, "a wire should be sent to v1's withdraw endpoint")
		assert.Equal(t, testAPIKey, r.Header.Get("X-BFX-APIKEY"), "a v1 request should send the API key")
		encoded := r.Header.Get("X-BFX-PAYLOAD")
		signature, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(encoded), []byte(testAPISecret))
		if assert.NoError(t, err, "GetHMAC should not error") {
			assert.Equal(t, hex.EncodeToString(signature), r.Header.Get("X-BFX-SIGNATURE"), "a v1 request should sign its base64 encoded payload")
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		assert.NoError(t, err, "a v1 request's payload should be base64 encoded")
		payload = nil
		assert.NoError(t, json.Unmarshal(decoded, &payload), "a v1 request's payload should be JSON")
		assert.Equal(t, "/v1/withdraw", payload["request"], "a v1 request's payload should name its path")
		nonceText, _ := payload["nonce"].(string)
		nonce, err := strconv.ParseInt(nonceText, 10, 64)
		// v1 and v2 requests take their nonces from one counter, started at the time in nanoseconds
		if assert.NoError(t, err, "a v1 request's nonce should be an integer") {
			assert.Greater(t, nonce, time.Now().Add(-time.Hour).UnixNano(), "a v1 request should send a nanosecond nonce")
		}
		_, err = w.Write([]byte(`[{"status":"success","message":"Your withdrawal request has been successfully submitted.","withdrawal_id":586829}]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	newWithdrawal := func() *withdraw.Request {
		return &withdraw.Request{
			Exchange: ex.Name, Currency: currency.USD, Amount: 1000, Type: withdraw.Fiat,
			Fiat: withdraw.FiatRequest{
				Bank: banking.Account{
					Enabled: true, AccountName: "Satoshi Nakamoto", AccountNumber: "12345", BankName: "Federal Reserve Bank", BankAddress: "33 Liberty Street",
					BankPostalCity: "New York", BankCountry: "United States", SWIFTCode: "FRNYUS33", SupportedCurrencies: "USD", SupportedExchanges: ex.Name,
				},
				IsExpressWire:            true,
				RequiresIntermediaryBank: true, IntermediaryBankName: "Intermediary Bank", IntermediaryBankAccountNumber: 123456789,
			},
		}
	}
	w := newWithdrawal()
	_, err := ex.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange, Withdrawal: w})
	require.NoError(t, err, "RequestFiatWithdrawal must not error")
	assert.NotContains(t, payload, "account_address", "a wire should send no account holder's address, which GCT does not hold")
	assert.Equal(t, "33 Liberty Street", payload["bank_address"], "a wire should send the bank's address")
	assert.Equal(t, 1.0, payload["expressWire"], "an express wire should be flagged 1, as the docs ask")
	assert.Equal(t, "123456789", payload["intermediary_bank_account"], "a wire should send its intermediary bank's account number exactly")
	assert.Equal(t, WalletTypeExchange, payload["walletselected"], "a wire should name the exchange wallet as v1 does")
	// The cases share ex and the payload it records, so each also checks that a call keeps nothing from the last
	for wallet, exp := range map[string]string{WalletTypeMargin: "trading", WalletTypeFunding: "deposit"} {
		t.Run(wallet, func(t *testing.T) {
			_, err = ex.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: wallet, Withdrawal: newWithdrawal()})
			require.NoErrorf(t, err, "RequestFiatWithdrawal must not error for the %s wallet", wallet)
			assert.Equalf(t, exp, payload["walletselected"], "a wire should name the %s wallet as v1 does", wallet)
		})
	}

	w = newWithdrawal()
	w.Fiat.IsExpressWire = false
	w.Fiat.IntermediaryIBAN = "DE89370400440532013000"
	_, err = ex.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange, Withdrawal: w})
	require.NoError(t, err, "RequestFiatWithdrawal must not error")
	assert.NotContains(t, payload, "expressWire", "a normal wire should leave the express flag out")
	assert.Equal(t, "DE89370400440532013000", payload["intermediary_bank_account"], "a wire should send its intermediary bank's IBAN as its account")

	// The cases share ex in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		account float64
		err     error
	}{
		{account: 0, err: errIntermediaryBankAccountRequired},
		{account: 123456789012345678, err: errIntermediaryBankAccountInexact},
		{account: 1 << 53, err: errIntermediaryBankAccountInexact},
		{account: 1.5, err: errIntermediaryBankAccountInexact},
		{account: -1, err: errIntermediaryBankAccountInexact},
	} {
		t.Run(strconv.FormatFloat(tc.account, 'f', -1, 64), func(t *testing.T) {
			w = newWithdrawal()
			w.Fiat.IntermediaryBankAccountNumber = tc.account
			_, err = ex.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange, Withdrawal: w})
			assert.ErrorIsf(t, err, tc.err, "RequestFiatWithdrawal should refuse intermediary account number %v", tc.account)
		})
	}
}

func TestLegacyWallet(t *testing.T) {
	t.Parallel()
	for wallet, exp := range map[string]string{WalletTypeExchange: "exchange", WalletTypeMargin: "trading", WalletTypeFunding: "deposit"} {
		t.Run(wallet, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, legacyWallet(wallet), "legacyWallet should name the %s wallet as v1 does", wallet)
		})
	}
}

func TestIntermediaryBankAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		f    withdraw.FiatRequest
		exp  string
		err  error
	}{
		{name: "an IBAN and an account number", f: withdraw.FiatRequest{IntermediaryIBAN: "DE89370400440532013000", IntermediaryBankAccountNumber: 123456789}, exp: "DE89370400440532013000"},
		{name: "an account number", f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 123456789}, exp: "123456789"},
		{name: "2 to the 53 less 1", f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 1<<53 - 1}, exp: "9007199254740991"},
		// 2^53 is held exactly, but so is the 2^53+1 that rounds to it
		{name: "2 to the 53", f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 1 << 53}, err: errIntermediaryBankAccountInexact},
		{name: "2 to the 53 plus 1", f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 9007199254740993}, err: errIntermediaryBankAccountInexact},
		{name: "a fraction", f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 1.5}, err: errIntermediaryBankAccountInexact},
		{name: "a negative number", f: withdraw.FiatRequest{IntermediaryBankAccountNumber: -1}, err: errIntermediaryBankAccountInexact},
		{name: "none", f: withdraw.FiatRequest{}, err: errIntermediaryBankAccountRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := intermediaryBankAccount(&tc.f)
			require.ErrorIsf(t, err, tc.err, "intermediaryBankAccount must return the expected error for %+v", tc.f)
			assert.Equalf(t, tc.exp, got, "intermediaryBankAccount should return the account a wire goes through for %+v", tc.f)
		})
	}
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.RequestFiatWithdrawal(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestFiatWithdrawal must reject a nil request")
	_, err = e.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: "novel", Withdrawal: &withdraw.Request{}})
	require.ErrorIs(t, err, errInvalidAccountWallet, "RequestFiatWithdrawal must need a wallet")
	_, err = e.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange, Withdrawal: &withdraw.Request{}})
	require.ErrorIs(t, err, errInvalidAmount, "RequestFiatWithdrawal must need an amount")
	if !mockTests {
		t.Skip("A withdrawal cannot be undone")
	}
	w := &withdraw.Request{
		Exchange: e.Name, Currency: currency.USD, Amount: 1000, Description: "test", Type: withdraw.Fiat,
		Fiat: withdraw.FiatRequest{Bank: banking.Account{
			Enabled: true, AccountName: "Satoshi Nakamoto", AccountNumber: "12345", BankName: "Federal Reserve Bank", BankAddress: "33 Liberty Street",
			BankPostalCity: "New York", BankCountry: "United States", SWIFTCode: "FRNYUS33", SupportedCurrencies: "USD", SupportedExchanges: e.Name,
		}},
	}
	resp, err := e.WithdrawFiatFunds(t.Context(), w)
	require.NoError(t, err, "WithdrawFiatFunds must not error")
	assert.Equal(t, &withdraw.ExchangeResponse{ID: "586829", Status: "success"}, resp, "WithdrawFiatFunds should return the withdrawal's ID")
	resp, err = e.WithdrawFiatFundsToInternationalBank(t.Context(), w)
	require.NoError(t, err, "WithdrawFiatFundsToInternationalBank must not error")
	assert.Equal(t, "586829", resp.ID, "WithdrawFiatFundsToInternationalBank should withdraw by wire")
}

func TestWithdrawFiatFundsFailures(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawFiatFunds must reject a nil request")
	_, err = e.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange})
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestFiatWithdrawal must reject a request without its withdrawal")
	req := &withdraw.Request{
		Exchange: e.Name, Currency: currency.USD, Amount: 1000, Description: "test", Type: withdraw.Fiat,
		Fiat: withdraw.FiatRequest{Bank: banking.Account{
			Enabled: true, AccountName: "Satoshi Nakamoto", AccountNumber: "12345", BankName: "Federal Reserve Bank", BankAddress: "33 Liberty Street",
			BankPostalCity: "New York", BankCountry: "United States", SWIFTCode: "FRNYUS33", SupportedCurrencies: "USD", SupportedExchanges: e.Name,
		}},
	}
	var apiErr *APIError
	_, err = newResponseServerExchange(t, `["error",10100,"apikey: invalid"]`).WithdrawFiatFunds(t.Context(), req)
	assert.ErrorAs(t, err, &apiErr, "WithdrawFiatFunds should return the error Bitfinex reports")
	_, err = newResponseServerExchange(t, `[]`).WithdrawFiatFunds(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "WithdrawFiatFunds should report a reply without the withdrawal")
	_, err = newResponseServerExchange(t, `[{"status":"error","message":"Invalid bank details","withdrawal_id":0}]`).WithdrawFiatFunds(t.Context(), req)
	assert.ErrorIs(t, err, errFiatWithdrawalFailed, "WithdrawFiatFunds should report a withdrawal Bitfinex did not make")
	uncredentialed := newResponseServerExchange(t, `[]`)
	uncredentialed.SetCredentials(&accounts.Credentials{})
	uncredentialed.SkipAuthCheck = false
	_, err = uncredentialed.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange, Withdrawal: req})
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "RequestFiatWithdrawal should need credentials")
}

func TestGetFeeByType(t *testing.T) { //nolint:tparallel // serial subtests share their exchange in order
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")
	wires := newTestExchange(t)
	// The cases share wires in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		name    string
		feeType exchange.FeeType
		c       currency.Code
		amount  float64
		exp     float64
	}{
		{name: "USD deposit", feeType: exchange.InternationalBankDepositFee, c: currency.USD, amount: 10000, exp: 60},
		{name: "EUR deposit", feeType: exchange.InternationalBankDepositFee, c: currency.EUR, amount: 100000, exp: 100},
		{name: "USD withdrawal", feeType: exchange.InternationalBankWithdrawalFee, c: currency.USD, amount: 10000, exp: 100},
		{name: "EUR withdrawal", feeType: exchange.InternationalBankWithdrawalFee, c: currency.EUR, amount: 200000, exp: 200},
		{name: "GBP withdrawal", feeType: exchange.InternationalBankWithdrawalFee, c: currency.GBP, amount: 1000, exp: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fee, err := wires.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: tc.feeType, FiatCurrency: tc.c, Amount: tc.amount})
			require.NoErrorf(t, err, "GetFeeByType must not error for a %s wire", tc.c)
			assert.Equalf(t, tc.exp, fee, "a %s wire of %v should cost a tenth of a percent, and at least Bitfinex's minimum for dollars and euros", tc.c, tc.amount)
		})
	}
	offline := newTestExchange(t)
	offline.SetCredentials(&accounts.Credentials{})
	fee, err := offline.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, PurchasePrice: 100, Amount: 1})
	require.NoError(t, err, "GetFeeByType must not error without credentials")
	assert.Zero(t, fee, "a trade without credentials should be estimated free, as Bitfinex charges no trading fees")

	fee, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.ADA, currency.USD)})
	require.NoError(t, err, "GetFeeByType must not error for a withdrawal fee")
	if mockTests {
		assert.Equal(t, 0.3, fee, "the withdrawal fee should be Bitfinex's")
	}
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewPair(currency.NewCode("NOPE"), currency.USD)})
	assert.ErrorIs(t, err, currency.ErrCurrencyNotSupported, "GetFeeByType should report a currency without a withdrawal fee")

	// Bitfinex prices Tether's tokens by chain, each under its method less the TETHER prefix. These are its fee and method
	// maps, trimmed to the rows used
	withdrawalConfig := func(methods string) *Exchange {
		return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reply, ok := map[string]string{
				"/v2/conf/pub:map:currency:tx:fee": `[[["ADA",[0,0.3]],["CNHTE",[0,35.817]],["CNHTX",[0,0]],["EUE",[0,0.001]],["MXNTE",[0,7.0351]],["USE",[0,2.1479]],["USX",[0,0]],["XAUTE",[0,0.00046028]]]]`,
				"/v2/conf/pub:map:tx:method":       methods,
			}[r.URL.Path]
			if !assert.Truef(t, ok, "GetFeeByType should read only the fee and method maps, not %s", r.URL.Path) {
				http.Error(w, "unexpected request", http.StatusBadRequest)
				return
			}
			_, err := w.Write([]byte(reply))
			assert.NoError(t, err, "Writing the response should not error")
		}))
	}
	tether := withdrawalConfig(`[[["WIRE",["USD","GBP","HKD","EUR","SGD","TWD","CHF"]],["TETHERUSE",["UST"]],["TETHEREUE",["EUT"]],["TETHERUSX",["UST"]],["TETHERCNHTE",["CNHT"]],["TETHERXAUTE",["XAUT"]],["TETHERMXNTE",["MXNT"]],["TETHERCNHTX",["CNHT"]]]]`)
	// The cases share tether in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		c   currency.Code
		exp float64
		err error
	}{
		{c: currency.ADA, exp: 0.3},
		{c: currency.NewCode("EUT"), exp: 0.001},
		{c: currency.XAUT, exp: 0.00046028},
		{c: currency.USDT, err: errTransferMethodNeeded},
		{c: currency.USD, err: currency.ErrCurrencyNotSupported},
		{c: currency.NewCode("NOPE"), err: currency.ErrCurrencyNotSupported},
	} {
		t.Run(tc.c.String(), func(t *testing.T) {
			fee, err = tether.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.Pair{Base: tc.c}})
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "GetFeeByType should refuse a withdrawal fee for %s", tc.c)
				return
			}
			require.NoErrorf(t, err, "GetFeeByType must not error for a withdrawal fee for %s", tc.c)
			assert.Equalf(t, tc.exp, fee, "a withdrawal of %s should cost the fee Bitfinex sets for it", tc.c)
		})
	}
	_, err = withdrawalConfig(`["error",10020,"symbol: invalid"]`).GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.Pair{Base: currency.XAUT}})
	var methodsErr *APIError
	assert.ErrorAs(t, err, &methodsErr, "GetFeeByType should return the error Bitfinex reports for its method map")

	refused := newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`)
	// The cases share refused in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		name    string
		feeType exchange.FeeType
	}{
		{name: "trade", feeType: exchange.CryptocurrencyTradeFee},
		{name: "withdrawal", feeType: exchange.CryptocurrencyWithdrawalFee},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err = refused.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: tc.feeType, Pair: currency.NewBTCUSD(), PurchasePrice: 1, Amount: 1})
			var apiErr *APIError
			assert.ErrorAsf(t, err, &apiErr, "GetFeeByType should return the error Bitfinex reports for fee type %v", tc.feeType)
		})
	}

	skipLiveWithoutCredentials(t)
	fee, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSD(), PurchasePrice: 1000, Amount: 1})
	require.NoError(t, err, "GetFeeByType must not error for a trade")
	if mockTests {
		assert.InDelta(t, 2.0, fee, 1e-9, "a taker trade quoted in fiat should be charged the account's taker fiat rate")
	}
	fee, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testDerivativePair, PurchasePrice: 1000, Amount: 1})
	require.NoError(t, err, "GetFeeByType must not error for a derivative trade")
	if mockTests {
		assert.InDelta(t, 6.5, fee, 1e-9, "a derivative taker trade should be charged the account's derivative taker rate")
	}
	fee, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testDerivativePair, PurchasePrice: 1000, Amount: 1, IsMaker: true})
	require.NoError(t, err, "GetFeeByType must not error for a derivative maker trade")
	assert.Zero(t, fee, "a derivative maker trade should be charged nothing, as Bitfinex pays its rebate")
}

func TestTradingFeeRate(t *testing.T) { //nolint:tparallel // serial subtests share rates in order
	t.Parallel()
	rates := &SummaryFeeRates{
		Maker: SummaryMakerFeeRates{ToCrypto: 0.1, ToStable: 0.2, ToFiat: 0.3, DerivativeRebate: 0.7},
		Taker: SummaryTakerFeeRates{ToCrypto: 0.4, ToStable: 0.5, ToFiat: 0.6, Derivative: 0.8},
	}
	// The cases share rates in order, so each also checks that tradingFeeRate leaves them as they were
	for _, tc := range []struct {
		name  string
		p     currency.Pair
		maker bool
		exp   float64
	}{
		{name: "maker crypto", p: currency.NewPair(currency.ETH, currency.BTC), maker: true, exp: 0.1},
		{name: "maker UST", p: currency.NewPair(currency.ETH, currency.UST), maker: true, exp: 0.2},
		{name: "maker USDT", p: currency.NewPair(currency.ETH, currency.USDT), maker: true, exp: 0.2},
		{name: "maker fiat", p: currency.NewPair(currency.ETH, currency.USD), maker: true, exp: 0.3},
		{name: "taker crypto", p: currency.NewPair(currency.ETH, currency.BTC), exp: 0.4},
		{name: "taker UST", p: currency.NewPair(currency.ETH, currency.UST), exp: 0.5},
		{name: "taker fiat", p: currency.NewPair(currency.ETH, currency.EUR), exp: 0.6},
		{name: "maker derivative", p: testDerivativePair, maker: true, exp: -0.7},
		{name: "taker derivative", p: testCoinMarginedPair, exp: 0.8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equalf(t, tc.exp, tradingFeeRate(rates, tc.p, tc.maker), "tradingFeeRate should give %s's rate, maker %t", tc.p, tc.maker)
		})
	}
}

func TestExchangeCurrency(t *testing.T) {
	t.Parallel()
	for c, exp := range map[currency.Code]currency.Code{
		currency.USDT: currency.UST,
		currency.UST:  currency.UST,
		currency.BTC:  currency.BTC,
		currency.USD:  currency.USD,
	} {
		t.Run(c.String(), func(t *testing.T) {
			t.Parallel()
			assert.Truef(t, exp.Equal(exchangeCurrency(c)), "exchangeCurrency should name %s as %s", c, exp)
		})
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	// Serial, so its two-week requests run before the hourly requests below, as they could when it was a test of its
	// own, while the current two weeks' cases run after them
	t.Run("two weeks", func(t *testing.T) {
		// Bitfinex's 14D candles open on Thursdays, where GCT's two weeks open on Mondays, so two-week candles are made from
		// weekly ones
		var paths []string
		ex := newWeeklyCandlesExchange(t, &paths)
		start := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
		item, err := ex.GetHistoricCandles(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.AddDate(0, 0, 42))
		require.NoError(t, err, "GetHistoricCandles must not error")
		assert.Equal(t, []string{"/v2/candles/trade:1W:tBTCUSD/hist"}, paths, "GetHistoricCandles should read weekly candles")
		assert.Equal(t, kline.TwoWeek, item.Interval, "GetHistoricCandles should return two-week candles")
		exp := twoWeekCandlesFromWeekly(start)
		require.Len(t, item.Candles, len(exp), "GetHistoricCandles must return each two weeks' candle")
		for i := range exp {
			assert.Truef(t, exp[i].Time.Equal(item.Candles[i].Time), "GetHistoricCandles candle %d should open at %s, not %s", i, exp[i].Time, item.Candles[i].Time)
			assert.InDeltaf(t, exp[i].Volume, item.Candles[i].Volume, 1e-8, "GetHistoricCandles candle %d should total its weeks' volume", i)
			exp[i].Time, exp[i].Volume = item.Candles[i].Time, item.Candles[i].Volume
			assert.Equalf(t, exp[i], item.Candles[i], "GetHistoricCandles candle %d should open with its first week, close with its second, and span both", i)
		}

		// A request returns at most its limit of weekly candles, two to a two-week candle
		ex.Features.Enabled.Kline.GlobalResultLimit = 5
		paths = nil
		_, err = ex.GetHistoricCandles(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.AddDate(0, 0, 42))
		assert.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "GetHistoricCandles should refuse two-week candles needing more weekly ones than a request returns")
		assert.Empty(t, paths, "GetHistoricCandles should send no request it cannot fill")

		// The weekly candles' own limit applies, should it differ from the two-week candles'
		ex.Features.Enabled.Kline.GlobalResultLimit = 10000
		ex.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneWeek, Capacity: 5}, kline.IntervalCapacity{Interval: kline.TwoWeek})
		_, err = ex.GetHistoricCandles(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.AddDate(0, 0, 42))
		assert.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "GetHistoricCandles should refuse more weekly candles than their limit")
		assert.Empty(t, paths, "GetHistoricCandles should send no request beyond the weekly candles' limit")
	})

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
				ex := newResponseServerExchange(t, tc.reply)
				require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, true), "StorePairs must not error")
				item, err := ex.GetHistoricCandles(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.Add(kline.TwoWeek.Duration()))
				skipIfTwoWeeksEnded(t, start)
				if tc.err != nil {
					assert.ErrorIs(t, err, tc.err, "GetHistoricCandles should error with too few weeks for a two-week candle")
					assert.Nil(t, item, "GetHistoricCandles should return no candles")
					return
				}
				require.NoError(t, err, "GetHistoricCandles must not error")
				require.Len(t, item.Candles, 1, "GetHistoricCandles must return the current two weeks' candle")
				exp := kline.Candle{Time: start, Open: 100, High: 104, Low: 98, Close: 103, Volume: 8, ValidationIssues: kline.PartialCandle}
				assert.Truef(t, exp.Time.Equal(item.Candles[0].Time), "GetHistoricCandles candle should open at %s, not %s", exp.Time, item.Candles[0].Time)
				exp.Time = item.Candles[0].Time
				assert.Equal(t, exp, item.Candles[0], "GetHistoricCandles should return the two weeks' candle, marked partial")
			})
		}
	})

	ex := newWrapperTestExchange(t)
	start, end := historyWindow(24 * time.Hour)
	// The cases share ex, so each also checks that a call keeps nothing from the last
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		t.Run(a.String(), func(t *testing.T) {
			item, err := ex.GetHistoricCandles(t.Context(), p, a, kline.OneHour, start, end)
			require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", a)
			require.NotEmptyf(t, item.Candles, "GetHistoricCandles must return %s candles", a)
			assert.Truef(t, item.Candles[0].Time.Before(item.Candles[len(item.Candles)-1].Time), "GetHistoricCandles should return %s candles oldest first", a)
		})
	}
}

func TestHistoryEnd(t *testing.T) {
	t.Parallel()
	past := time.Now().Add(-time.Hour)
	assert.Equal(t, past, historyEnd(past), "historyEnd should keep a window ending in the past")
	before := time.Now()
	end := historyEnd(before.Add(25 * time.Hour))
	assert.WithinRange(t, end, before, time.Now(), "historyEnd should end a window ending in the future now")
}

func TestHistoryRequestsEndNoLaterThanNow(t *testing.T) {
	t.Parallel()
	var ends []int64
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		end, err := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
		assert.NoErrorf(t, err, "%s should have an end", r.URL.Path)
		ends = append(ends, end)
		_, err = w.Write([]byte(`[]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, true), "StorePairs must not error")
	// A candle request ends at the next candle boundary, the next UTC midnight for daily candles, which Bitfinex refuses
	_, err := ex.GetHistoricCandles(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.OneDay, time.Now().Add(-72*time.Hour), time.Now())
	require.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "GetHistoricCandles must find no candles in an empty history")
	_, err = ex.GetHistoricTrades(t.Context(), currency.NewBTCUSD(), asset.Spot, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, err, "GetHistoricTrades must not error for an empty history")
	require.Len(t, ends, 2, "GetHistoricCandles and GetHistoricTrades must each request one page")
	for _, end := range ends {
		assert.LessOrEqual(t, end, time.Now().UnixMilli(), "a history request should end no later than now")
	}
}

// newWeeklyCandlesExchange returns an exchange trading BTCUSD spot whose REST requests all receive Bitfinex's weekly
// candles for the six weeks from 31 August 2026, and the next, as its end is inclusive, recording each request's path
func newWeeklyCandlesExchange(t *testing.T, paths *[]string) *Exchange {
	t.Helper()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*paths = append(*paths, r.URL.Path)
		_, err := w.Write([]byte(`[[1788134400000,77683,80430,82320,76321,5293.42158328],[1788739200000,80450,76833,80542,76043,9418.91123549],[1789344000000,76881,81184,81922,74985,7700.72194816],[1789948800000,81184,84464,87392,80846,7312.9739249],[1790553600000,84451,86501,87197,82491,6971.19971307],[1791158400000,86504,83800,86969,80347,4323.98135151],[1791763200000,83777,82159,84378,79821,2889.43614902]]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, true), "StorePairs must not error")
	return ex
}

// twoWeekCandlesFromWeekly returns the two-week candles newWeeklyCandlesExchange's weekly candles make from start, 31
// August 2026
func twoWeekCandlesFromWeekly(start time.Time) []kline.Candle {
	return []kline.Candle{
		{Time: start, Open: 77683, Close: 76833, High: 82320, Low: 76043, Volume: 14712.33281877},
		{Time: start.AddDate(0, 0, 14), Open: 76881, Close: 84464, High: 87392, Low: 74985, Volume: 15013.69587306},
		{Time: start.AddDate(0, 0, 28), Open: 84451, Close: 83800, High: 87197, Low: 80347, Volume: 11295.18106458},
	}
}

// currentTwoWeeksReplies returns the start of the current two weeks and replies in Bitfinex's format for their weekly
// candles: the first week's alone, as before their second week trades, then both weeks'. Until their second week has a
// candle, it is dropped as an empty partial candle, leaving one week, too few to make the two-week candle
func currentTwoWeeksReplies() (start time.Time, firstWeek, bothWeeks string) {
	start = time.Now().UTC().Truncate(kline.TwoWeek.Duration())
	first := "[" + strconv.FormatInt(start.UnixMilli(), 10) + ",100,101,102,99,5]"
	second := "[" + strconv.FormatInt(start.AddDate(0, 0, 7).UnixMilli(), 10) + ",101,103,104,98,3]"
	return start, "[" + first + "]", "[" + first + "," + second + "]"
}

// skipIfTwoWeeksEnded skips a test whose request for the two weeks from start was made as or after they ended, when the
// request could have fallen in the next two weeks
func skipIfTwoWeeksEnded(t *testing.T, start time.Time) {
	t.Helper()
	if !time.Now().UTC().Truncate(kline.TwoWeek.Duration()).Equal(start) {
		t.Skip("The current two weeks ended during the test, so its request could have fallen in the next two weeks")
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	// Serial, so its two-week requests run before the hourly requests below, as they could when it was a test of its
	// own, while the current two weeks' cases run after them
	t.Run("two weeks", func(t *testing.T) {
		var paths []string
		ex := newWeeklyCandlesExchange(t, &paths)
		start := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
		item, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.AddDate(0, 0, 42))
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		assert.Equal(t, []string{"/v2/candles/trade:1W:tBTCUSD/hist"}, paths, "GetHistoricCandlesExtended should read weekly candles")
		assert.Equal(t, kline.TwoWeek, item.Interval, "GetHistoricCandlesExtended should return two-week candles")
		exp := twoWeekCandlesFromWeekly(start)
		require.Len(t, item.Candles, len(exp), "GetHistoricCandlesExtended must return each two weeks' candle")
		for i := range exp {
			assert.Truef(t, exp[i].Time.Equal(item.Candles[i].Time), "GetHistoricCandlesExtended candle %d should open at %s, not %s", i, exp[i].Time, item.Candles[i].Time)
			assert.InDeltaf(t, exp[i].Volume, item.Candles[i].Volume, 1e-8, "GetHistoricCandlesExtended candle %d should total its weeks' volume", i)
			exp[i].Time, exp[i].Volume = item.Candles[i].Time, item.Candles[i].Volume
			assert.Equalf(t, exp[i], item.Candles[i], "GetHistoricCandlesExtended candle %d should open with its first week, close with its second, and span both", i)
		}

		// A request returns at most its limit of weekly candles, two to a two-week candle, so they are paged
		ex.Features.Enabled.Kline.GlobalResultLimit = 5
		paths = nil
		item, err = ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.AddDate(0, 0, 42))
		require.NoError(t, err, "GetHistoricCandlesExtended must page the weekly candles")
		assert.Len(t, paths, 2, "GetHistoricCandlesExtended should read the weekly candles over two requests")
		assert.Len(t, item.Candles, 3, "GetHistoricCandlesExtended should return each two weeks' candle")

		// The weekly candles' own limit applies, should it differ from the two-week candles'
		ex.Features.Enabled.Kline.GlobalResultLimit = 10000
		ex.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneWeek, Capacity: 5}, kline.IntervalCapacity{Interval: kline.TwoWeek})
		paths = nil
		_, err = ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.AddDate(0, 0, 42))
		require.NoError(t, err, "GetHistoricCandlesExtended must page the weekly candles")
		assert.Len(t, paths, 2, "GetHistoricCandlesExtended should page the weekly candles by their limit")
	})

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
				ex := newResponseServerExchange(t, tc.reply)
				require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, true), "StorePairs must not error")
				item, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.TwoWeek, start, start.Add(kline.TwoWeek.Duration()))
				skipIfTwoWeeksEnded(t, start)
				if tc.err != nil {
					assert.ErrorIs(t, err, tc.err, "GetHistoricCandlesExtended should error with too few weeks for a two-week candle")
					assert.Nil(t, item, "GetHistoricCandlesExtended should return no candles")
					return
				}
				require.NoError(t, err, "GetHistoricCandlesExtended must not error")
				require.Len(t, item.Candles, 1, "GetHistoricCandlesExtended must return the current two weeks' candle")
				exp := kline.Candle{Time: start, Open: 100, High: 104, Low: 98, Close: 103, Volume: 8, ValidationIssues: kline.PartialCandle}
				assert.Truef(t, exp.Time.Equal(item.Candles[0].Time), "GetHistoricCandlesExtended candle should open at %s, not %s", exp.Time, item.Candles[0].Time)
				exp.Time = item.Candles[0].Time
				assert.Equal(t, exp, item.Candles[0], "GetHistoricCandlesExtended should return the two weeks' candle, marked partial")
			})
		}
	})

	ex := newWrapperTestExchange(t)
	start, end := historyWindow(24 * time.Hour)
	item, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	assert.NotEmpty(t, item.Candles, "GetHistoricCandlesExtended should return candles")
}

func TestGetCurrencyTradeURL(t *testing.T) { //nolint:tparallel // serial subtests share ex
	t.Parallel()
	ex := newWrapperTestExchange(t)
	// The cases share ex, so each also checks that a call keeps nothing from the last
	for a, exp := range map[asset.Item]string{
		asset.Spot:                tradeBaseURL + "/t/BTC:USD?type=exchange",
		asset.Margin:              tradeBaseURL + "/t/BTC:USD?type=margin",
		asset.MarginFunding:       tradeBaseURL + "/f/USD",
		asset.USDTMarginedFutures: tradeBaseURL + "/t/BTCF0:USTF0",
		asset.CoinMarginedFutures: tradeBaseURL + "/t/ETHF0:BTCF0",
	} {
		t.Run(a.String(), func(t *testing.T) {
			p := map[asset.Item]currency.Pair{
				asset.MarginFunding:       testFundingUSD,
				asset.USDTMarginedFutures: testDerivativePair,
				asset.CoinMarginedFutures: testCoinMarginedPair,
			}[a]
			if p.IsEmpty() {
				p = currency.NewBTCUSD()
			}
			got, err := ex.GetCurrencyTradeURL(t.Context(), a, p)
			require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
			assert.Equalf(t, exp, got, "GetCurrencyTradeURL should link %s's trade page", a)
		})
	}
	_, err := ex.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL should refuse an empty pair")
}

func TestUnsupportedWrapperMethods(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetServerTime should not be supported")
	err = e.SetLeverage(t.Context(), asset.USDTMarginedFutures, testDerivativePair, margin.Isolated, 10, order.Buy)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "SetLeverage should not be supported, as Bitfinex takes leverage with each order")
}

func TestCancelBatchOrdersWithWebsocketConnected(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, subscription.List{})
	require.True(t, ex.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the wrapper must be able to use the authenticated websocket")
	ids := []string{testWrapperOrderID, "1747566429"}
	if !mockTests {
		ids = []string{strconv.FormatUint(placeLiveTestOrder(t).ID, 10)}
	}
	cancels := make([]order.Cancel, len(ids))
	for i := range ids {
		cancels[i] = order.Cancel{OrderID: ids[i], AssetType: asset.Spot, Pair: currency.NewBTCUSD()}
	}
	resp, err := ex.CancelBatchOrders(t.Context(), cancels)
	require.NoError(t, err, "CancelBatchOrders must not error with the websocket connected")
	assert.Equal(t, order.PendingCancel.String(), resp.Status[ids[0]], "an order Bitfinex's reply names should be pending cancellation until its close event confirms it")
	if mockTests {
		assert.Equal(t, order.UnknownStatus.String(), resp.Status["1747566429"], "an order Bitfinex's reply leaves out should be unknown")
		assert.Empty(t, venue.inputs(wsOrderCancelMulti), "CancelBatchOrders should send nothing through the websocket, where a late notification would answer the next request")
	}
}

func TestWrapperOrdersThroughWebsocket(t *testing.T) {
	t.Parallel()
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t, subscription.List{})
	require.True(t, ex.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the wrapper must be able to use the authenticated websocket")
	req := wsTestOrderRequest(t)
	resp, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: req.Amount, Price: req.Price})
	require.NoError(t, err, "SubmitOrder must not error through the websocket")
	_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Stop, Side: order.Sell, Amount: req.Amount, TriggerPrice: math.Inf(1)})
	assert.ErrorIs(t, err, errInvalidPrice, "SubmitOrder should return the websocket order's error")
	_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Amount: req.Amount, Price: req.Price, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: req.Price / 2}}})
	assert.ErrorIs(t, err, errRiskManagementUnsupported, "SubmitOrder should refuse a stop loss through the websocket")
	_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Type: order.Stop, Side: order.Sell, Amount: req.Amount, TriggerPrice: req.Price / 2, TriggerPriceType: order.MarkPrice})
	assert.ErrorIs(t, err, errTriggerPriceTypeUnsupported, "SubmitOrder should refuse a trigger on the mark price through the websocket")
	if !mockTests {
		id, err := parseOrderID(resp.OrderID)
		require.NoError(t, err, "SubmitOrder must return the order's ID")
		cancelLiveTestOrderOnCleanup(t, id)
	}
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: resp.OrderID, AssetType: asset.Spot, Pair: currency.NewBTCUSD()}), "CancelOrder must not error through the websocket")
	if mockTests {
		assert.Len(t, venue.inputs(wsOrderNew), 1, "SubmitOrder should place the order through the websocket, and send none it refuses")
		assert.Len(t, venue.inputs(wsOrderCancel), 1, "CancelOrder should cancel the order through the websocket")
	}
}

func TestAuthenticateWebsocket(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, venue := newWebsocketTestExchange(t, nil)
	require.NoError(t, ex.AuthenticateWebsocket(t.Context()), "AuthenticateWebsocket must not error")
	if mockTests {
		assert.Len(t, venue.events("auth"), 2, "the account connection should authenticate on connecting and again when asked")
	}
}

// testStatusReplyAge allows for a live status reply taken before a funding event and read just after it, as the replies
// have been read up to 9 seconds old
const testStatusReplyAge = 30 * time.Second

func TestGetFuturesContractDetails(t *testing.T) { //nolint:tparallel // serial subtests share e or precede later GetFuturesContractDetails calls
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must refuse a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must refuse a futures asset Bitfinex does not trade")
	const refused = `["error",10020,"symbol: invalid"]`
	info := func(pair string) string {
		return `[[["` + pair + `",[1562164542332,null,null,"0.00004","100.0",null,null,null,0.01,0.005]]]]`
	}
	// Serial, as GetFuturesContractDetails is called again after these cases
	for name, bodies := range map[string]map[string]string{
		"the pair information refused": {"/v2/conf/pub:info:pair:futures": refused},
		"the statuses refused":         {"/v2/conf/pub:info:pair:futures": info("BTCF0:USTF0"), "/v2/status/deriv": refused},
	} {
		t.Run(name, func(t *testing.T) {
			_, err = newPathResponseServerExchange(t, bodies).GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
			var apiErr *APIError
			assert.ErrorAsf(t, err, &apiErr, "GetFuturesContractDetails should return the error Bitfinex reports with %s", name)
		})
	}
	_, err = newPathResponseServerExchange(t, map[string]string{"/v2/conf/pub:info:pair:futures": info("BTCF0")}).GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, errSymbolUnparsable, "GetFuturesContractDetails should refuse a contract it cannot read")
	// The cases share e, so each also checks that a call keeps nothing from the last
	for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: testDerivativePair, asset.CoinMarginedFutures: testCoinMarginedPair} {
		t.Run(a.String(), func(t *testing.T) {
			before := time.Now()
			contracts, err := e.GetFuturesContractDetails(t.Context(), a)
			require.NoErrorf(t, err, "GetFuturesContractDetails must not error for %s", a)
			i := slices.IndexFunc(contracts, func(c futures.Contract) bool { return c.Name.Equal(p) })
			require.NotEqualf(t, -1, i, "GetFuturesContractDetails must return %s's %s contract", a, p)
			for j := range contracts {
				assert.Equalf(t, a, derivativeAsset(contracts[j].Name.Quote), "GetFuturesContractDetails should return only the contracts %s margins, got %s", a, contracts[j].Name)
				assert.NotContainsf(t, contracts[j].Name.String(), testListingMarker, "GetFuturesContractDetails should leave out %s test contracts", a)
			}
			if !mockTests {
				c := contracts[i]
				assert.Equalf(t, futures.Perpetual, c.Type, "%s should be perpetual", p)
				assert.Equalf(t, p.Quote.Upper(), c.SettlementCurrency, "%s should settle in its collateral", p)
				assert.Positivef(t, c.MaxLeverage, "%s should have a maximum leverage", p)
				assert.WithinDurationf(t, before, c.LatestRate.Time, derivativeFundingInterval+testStatusReplyAge, "%s's latest funding rate should be from its last funding event", p)
				return
			}
			collateral := p.Quote.Upper()
			exp := futures.Contract{
				Exchange: e.Name, Name: p, Asset: a, IsActive: true, Type: futures.Perpetual, SettlementType: futures.Linear, SettlementCurrency: collateral,
				MarginCurrency: collateral, Multiplier: 1, MaxLeverage: 100, FundingRateFloor: decimal.MustFromFloat(-0.0025), FundingRateCeiling: decimal.MustFromFloat(0.0025),
			}
			if a == asset.USDTMarginedFutures {
				exp.Underlying, exp.StartDate = currency.NewPair(currency.BTC, currency.UST), time.UnixMilli(1562164542332)
				exp.LatestRate = fundingrate.Rate{Time: testDerivativeNextFunding.Add(-8 * time.Hour), Rate: decimal.MustFromFloat(0.00003859)}
			} else {
				exp.Underlying, exp.StartDate = currency.NewPair(currency.ETH, currency.BTC), time.UnixMilli(1608633542930)
				exp.LatestRate = fundingrate.Rate{Time: testDerivativeNextFunding.Add(-8 * time.Hour), Rate: decimal.MustFromFloat(0)}
			}
			assert.Equalf(t, exp, contracts[i], "GetFuturesContractDetails should describe the %s contract", p)
		})
	}
	if mockTests {
		contracts, err := e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
		require.NoError(t, err, "GetFuturesContractDetails must not error")
		assert.Len(t, contracts, 6, "GetFuturesContractDetails should return every USDT margined contract recorded")
	}
}

func TestLatestFundingRate(t *testing.T) {
	t.Parallel()
	next := time.UnixMilli(1792137600000)
	assert.Equal(t, fundingrate.Rate{Time: next.Add(-derivativeFundingInterval), Rate: decimal.MustFromFloat(0.00010884)},
		latestFundingRate(&DerivativeStatusResponse{CurrentFunding: 0.00010884, NextFundingTime: types.Time(next)}), "latestFundingRate should be the current funding, set a funding interval before the next event")
	assert.Equal(t, fundingrate.Rate{Rate: decimal.MustFromFloat(0.00010884)}, latestFundingRate(&DerivativeStatusResponse{CurrentFunding: 0.00010884}),
		"latestFundingRate should leave the time unset without the next funding event's")
}

func TestGetLatestFundingRates(t *testing.T) { //nolint:tparallel // serial subtests share ex
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Margin, Pair: currency.NewBTCUSD()})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates must refuse an asset trading no perpetual contracts")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates must refuse a predicted rate, which Bitfinex does not give")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: testCoinMarginedPair})
	require.ErrorIs(t, err, errPairNotOfAsset, "GetLatestFundingRates must refuse a pair the asset does not trade")
	var apiErr *APIError
	_, err = newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair})
	require.ErrorAs(t, err, &apiErr, "GetLatestFundingRates must return the error Bitfinex reports")
	_, err = newResponseServerExchange(t, `[]`).GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair})
	require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetLatestFundingRates must report a contract Bitfinex has no status for")
	disabled := newWrapperTestExchange(t)
	require.NoError(t, disabled.CurrencyPairs.SetAssetEnabled(asset.CoinMarginedFutures, false), "SetAssetEnabled must not error")
	_, err = disabled.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures})
	require.ErrorIs(t, err, asset.ErrNotEnabled, "GetLatestFundingRates must refuse the enabled pairs of an asset not enabled")
	none := newWrapperTestExchange(t)
	require.NoError(t, none.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, nil, true), "StorePairs must not error")
	_, err = none.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetLatestFundingRates must refuse an asset without enabled pairs")
	ex := newWrapperTestExchange(t)
	// The cases share ex, so each also checks that a call keeps nothing from the last
	for a, req := range map[asset.Item]*fundingrate.LatestRateRequest{
		asset.USDTMarginedFutures: {Asset: asset.USDTMarginedFutures, Pair: testDerivativePair},
		// The asset's enabled pairs, without a pair
		asset.CoinMarginedFutures: {Asset: asset.CoinMarginedFutures},
	} {
		t.Run(a.String(), func(t *testing.T) {
			before := time.Now()
			rates, err := ex.GetLatestFundingRates(t.Context(), req)
			after := time.Now()
			require.NoErrorf(t, err, "GetLatestFundingRates must not error for %s", a)
			require.Lenf(t, rates, 1, "GetLatestFundingRates must return %s's rate", a)
			assert.WithinRangef(t, rates[0].TimeChecked, before, after, "the %s rate should be checked now", a)
			rates[0].TimeChecked = time.Time{}
			if !mockTests {
				assert.Equalf(t, derivativeFundingInterval, rates[0].TimeOfNextRate.Sub(rates[0].LatestRate.Time), "the %s rate should be set a funding interval before the next", a)
				assert.Truef(t, rates[0].TimeOfNextRate.After(before.Add(-testStatusReplyAge)), "the %s rate's next funding event should be to come", a)
				return
			}
			exp := fundingrate.LatestRateResponse{Exchange: ex.Name, Asset: a, Pair: testDerivativePair, TimeOfNextRate: testDerivativeNextFunding}
			exp.LatestRate = fundingrate.Rate{Time: testDerivativeNextFunding.Add(-8 * time.Hour), Rate: decimal.MustFromFloat(0.00003859)}
			if a == asset.CoinMarginedFutures {
				enabled, err := ex.GetEnabledPairs(a)
				require.NoError(t, err, "GetEnabledPairs must not error")
				exp.Pair, exp.LatestRate.Rate = enabled[0], decimal.MustFromFloat(0)
			}
			assert.Equalf(t, exp, rates[0], "GetLatestFundingRates should return the rate Bitfinex set at %s's latest funding event", exp.Pair)
		})
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	t.Run("window end", func(t *testing.T) {
		t.Parallel()
		var ends []int64
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			end, err := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
			assert.NoError(t, err, "the status history request should have an end")
			ends = append(ends, end)
			_, err = w.Write([]byte(`[]`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		past := time.UnixMilli(1792022400000)
		// The cases share ex and the ends it records in order, so each also checks that a call keeps nothing from the last
		for _, tc := range []struct {
			name       string
			start, end time.Time
			// exp is the end the request should give, or now when unset
			exp int64
		}{
			// The snapshots a funding interval after the window carry the rates of its last events
			{name: "a past window", start: past, end: past.Add(16 * time.Hour), exp: past.Add(24 * time.Hour).UnixMilli()},
			// Bitfinex refuses a window ending after the current UTC day, so one ending now goes no later
			{name: "a window ending now", start: time.Now().Add(-16 * time.Hour), end: time.Now()},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ends = nil
				earliest := time.Now().UnixMilli()
				_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: tc.start, EndDate: tc.end})
				latest := time.Now().UnixMilli()
				require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates must find no rates in an empty history")
				require.Len(t, ends, 1, "GetHistoricalFundingRates must request one page of the status history")
				if tc.exp != 0 {
					earliest, latest = tc.exp, tc.exp
				}
				assert.GreaterOrEqualf(t, ends[0], earliest, "a window ending at %s should request the status history up to the window's end and a funding interval, but not past now", tc.end)
				assert.LessOrEqualf(t, ends[0], latest, "a window ending at %s should request the status history up to the window's end and a funding interval, but not past now", tc.end)
			})
		}
	})

	// Beside the scenario above, as when it was a test of its own
	t.Run("rates over a window", func(t *testing.T) {
		t.Parallel()
		start, end := historyWindow(16 * time.Hour)
		_, err := e.GetHistoricalFundingRates(t.Context(), nil)
		require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")
		// The cases share e in order, so each also checks that a call keeps nothing from the last
		for _, tc := range []struct {
			name string
			r    fundingrate.HistoricalRatesRequest
			err  error
		}{
			{name: "spot", r: fundingrate.HistoricalRatesRequest{Asset: asset.Spot, Pair: currency.NewBTCUSD(), StartDate: start, EndDate: end}, err: futures.ErrNotPerpetualFuture},
			{name: "payments", r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end, IncludePayments: true}, err: common.ErrFunctionNotSupported},
			{name: "predicted rate", r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end, IncludePredictedRate: true}, err: common.ErrFunctionNotSupported},
			{name: "start after end", r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: end, EndDate: start}, err: common.ErrStartAfterEnd},
			{name: "no pair", r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, StartDate: start, EndDate: end}, err: currency.ErrCurrencyPairEmpty},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := e.GetHistoricalFundingRates(t.Context(), &tc.r)
				assert.ErrorIsf(t, err, tc.err, "GetHistoricalFundingRates should refuse %+v", tc.r)
			})
		}
		_, err = newRefusingServerExchange(t, `["error",10020,"symbol: invalid"]`).GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end})
		var apiErr *APIError
		assert.ErrorAs(t, err, &apiErr, "GetHistoricalFundingRates should return the error Bitfinex reports")
		rates, err := e.GetHistoricalFundingRates(mockRequestContext(t), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end})
		require.NoError(t, err, "GetHistoricalFundingRates must not error")
		if !mockTests {
			require.NotEmpty(t, rates.FundingRates, "GetHistoricalFundingRates must return the window's rates")
			for i := range rates.FundingRates {
				set := rates.FundingRates[i].Time
				assert.Falsef(t, set.Before(start) || set.After(end), "the rate set at %s should be within the window", set)
				if i > 0 {
					assert.Equal(t, derivativeFundingInterval, set.Sub(rates.FundingRates[i-1].Time), "rates should be set a funding interval apart, oldest first")
				}
			}
			assert.Equal(t, rates.FundingRates[len(rates.FundingRates)-1], rates.LatestRate, "the latest rate should be the window's last")
			return
		}
		// The window covers three funding events, whose rates the snapshots after each report
		set := func(hours int, rate float64) fundingrate.Rate {
			return fundingrate.Rate{Time: time.UnixMilli(testHistoryStart.UnixMilli()).Add(time.Duration(hours) * time.Hour), Rate: decimal.MustFromFloat(rate)}
		}
		exp := []fundingrate.Rate{set(0, 0.0001331), set(8, 0.00003394), set(16, 0.0000682)}
		assert.Equal(t, &fundingrate.HistoricalRates{
			Exchange: e.Name, Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: exp[0].Time, EndDate: exp[2].Time, LatestRate: exp[2], FundingRates: exp,
		}, rates, "GetHistoricalFundingRates should return the rate set at each of the window's funding events")

		empty := newResponseServerExchange(t, `[]`)
		_, err = empty.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end})
		assert.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates should report a window without funding rates")
	})
}

func TestIsPerpetualFutureCurrency(t *testing.T) { //nolint:tparallel // serial subtests share e
	t.Parallel()
	// The cases share e, so each also checks that a call keeps nothing from the last
	for a, exp := range map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true, asset.Spot: false, asset.Margin: false, asset.Futures: false} {
		t.Run(a.String(), func(t *testing.T) {
			got, err := e.IsPerpetualFutureCurrency(a, testDerivativePair)
			require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
			assert.Equalf(t, exp, got, "IsPerpetualFutureCurrency should report whether %s trades perpetual contracts", a)
		})
	}
}

func TestGetCollateralCurrencyForContract(t *testing.T) { //nolint:tparallel // serial subtests share e
	t.Parallel()
	// The cases share e, so each also checks that a call keeps nothing from the last
	for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: testDerivativePair, asset.CoinMarginedFutures: currency.NewPairWithDelimiter("ETHF0", "btcf0", "-")} {
		t.Run(a.String(), func(t *testing.T) {
			c, ca, err := e.GetCollateralCurrencyForContract(a, p)
			require.NoErrorf(t, err, "GetCollateralCurrencyForContract must not error for %s", a)
			assert.Truef(t, p.Quote.Equal(c), "%s's collateral should be its quote currency", p)
			assert.Equalf(t, a, ca, "%s's collateral should be held for its asset", p)
		})
	}
	_, _, err := e.GetCollateralCurrencyForContract(asset.Margin, currency.NewBTCUSD())
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetCollateralCurrencyForContract should refuse an asset trading no perpetual contracts")
	_, _, err = e.GetCollateralCurrencyForContract(asset.USDTMarginedFutures, testCoinMarginedPair)
	assert.ErrorIs(t, err, errPairNotOfAsset, "GetCollateralCurrencyForContract should refuse a pair the asset does not trade")
}

func TestGetOpenInterest(t *testing.T) { //nolint:tparallel // serial subtests share e in order
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.BTC.Item, Quote: currency.USD.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must refuse an asset trading no perpetual contracts")
	keys := []key.PairAsset{
		{Base: testDerivativePair.Base.Item, Quote: testDerivativePair.Quote.Item, Asset: asset.USDTMarginedFutures},
		{Base: testCoinMarginedPair.Base.Item, Quote: testCoinMarginedPair.Quote.Item, Asset: asset.CoinMarginedFutures},
	}
	// The cases share e in order, so each also checks that a call keeps nothing from the last
	for _, requested := range [][]key.PairAsset{keys, nil} {
		t.Run(strconv.Itoa(len(requested))+" keys", func(t *testing.T) {
			got, err := e.GetOpenInterest(t.Context(), requested...)
			require.NoErrorf(t, err, "GetOpenInterest must not error for %d keys", len(requested))
			for i := range got {
				assert.NotContainsf(t, got[i].Key.Base.Symbol, testListingMarker, "GetOpenInterest should leave out test contracts")
				assert.Equalf(t, derivativeAsset(got[i].Key.Pair().Quote), got[i].Key.Asset, "%s's open interest should be its asset's", got[i].Key.Pair())
			}
			if !mockTests {
				if len(requested) != 0 {
					assert.Len(t, got, len(requested), "GetOpenInterest should return each contract's open interest")
				}
				return
			}
			assert.Equalf(t, []futures.OpenInterest{
				{Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, testDerivativePair), OpenInterest: 9268.71813464},
				{Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, testCoinMarginedPair), OpenInterest: 6061.94360798},
			}, got, "GetOpenInterest should return each contract's open interest for %d keys", len(requested))
		})
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.USDTMarginedFutures, testDerivativePair, margin.Multi, order.Buy)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "GetLeverage must refuse cross margin, as each derivative position holds its own collateral")
	_, err = e.GetLeverage(t.Context(), asset.Margin, currency.NewBTCUSD(), margin.Isolated, order.Buy)
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLeverage must refuse an asset trading no perpetual contracts")
	skipLiveWithoutCredentials(t)
	leverage, err := e.GetLeverage(t.Context(), asset.USDTMarginedFutures, testDerivativePair, margin.Isolated, order.Buy)
	if !mockTests {
		if err != nil {
			require.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetLeverage must error only for want of a position")
		}
		return
	}
	require.NoError(t, err, "GetLeverage must not error")
	assert.Equal(t, 80.0, leverage, "GetLeverage should return the derivative position's leverage")
	_, err = e.GetLeverage(t.Context(), asset.CoinMarginedFutures, testCoinMarginedPair, margin.Isolated, order.Buy)
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetLeverage should report a contract without a position")
}

func TestDerivativePosition(t *testing.T) {
	t.Parallel()
	_, err := e.derivativePosition(t.Context(), asset.Margin, currency.NewBTCUSD())
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "derivativePosition must refuse an asset trading no perpetual contracts")
	_, err = e.derivativePosition(t.Context(), asset.USDTMarginedFutures, testCoinMarginedPair)
	require.ErrorIs(t, err, errPairNotOfAsset, "derivativePosition must refuse a pair the asset does not trade")
	ex := newResponseServerExchange(t, `[["tBTCF0:USTF0","ACTIVE",0.2,80000,0,0,-12.5,-0.0625,79400,80,null,142355653,1792022400000,1792026000000,null,1,null,200,160,null]]`)
	got, err := ex.derivativePosition(t.Context(), asset.USDTMarginedFutures, testDerivativePair)
	require.NoError(t, err, "derivativePosition must not error")
	assert.Equal(t, uint64(142355653), got.ID, "derivativePosition should return the contract's active position")
	_, err = ex.derivativePosition(t.Context(), asset.CoinMarginedFutures, testCoinMarginedPair)
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "derivativePosition should report a contract without a position")
	_, err = newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).derivativePosition(t.Context(), asset.USDTMarginedFutures, testDerivativePair)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "derivativePosition should return the error Bitfinex reports")
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary must reject a nil request")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Margin, Pair: currency.NewBTCUSD()})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetFuturesPositionSummary must refuse an asset trading no perpetual contracts")
	skipLiveWithoutCredentials(t)
	summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair})
	if !mockTests {
		if err != nil {
			require.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetFuturesPositionSummary must error only for want of a position")
		}
		return
	}
	require.NoError(t, err, "GetFuturesPositionSummary must not error")
	assert.Equal(t, &futures.PositionSummary{
		Pair: testDerivativePair, Asset: asset.USDTMarginedFutures, MarginType: margin.Isolated, Currency: testDerivativePair.Quote.Upper(),
		StartDate: time.UnixMilli(1792022400000), IsolatedMargin: decimal.MustFromFloat(200), CollateralUsed: decimal.MustFromFloat(200), Leverage: decimal.MustFromFloat(80),
		EstimatedLiquidationPrice: decimal.MustFromFloat(79400), CurrentSize: decimal.MustFromFloat(0.2), ContractSettlementType: futures.Linear,
		AverageOpenPrice: decimal.MustFromFloat(80000), UnrealisedPNL: decimal.MustFromFloat(-12.5),
	}, summary, "GetFuturesPositionSummary should summarise the derivative position")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: testCoinMarginedPair})
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetFuturesPositionSummary should report a contract without a position")
}

func TestGetFuturesPositionOrders(t *testing.T) { //nolint:tparallel // serial subtests share e in order
	t.Parallel()
	_, err := e.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionOrders must reject a nil request")
	now := time.Now().Truncate(time.Millisecond)
	// The cases share e in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		name string
		r    futures.PositionsRequest
		err  error
	}{
		{name: "margin", r: futures.PositionsRequest{Asset: asset.Margin, Pairs: currency.Pairs{currency.NewBTCUSD()}, StartDate: now.Add(-time.Hour), EndDate: now}, err: futures.ErrNotPerpetualFuture},
		{name: "no pairs", r: futures.PositionsRequest{Asset: asset.USDTMarginedFutures, StartDate: now.Add(-time.Hour), EndDate: now}, err: currency.ErrCurrencyPairsEmpty},
		{name: "start after end", r: futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now, EndDate: now.Add(-time.Hour)}, err: common.ErrStartAfterEnd},
		{name: "too long a window", r: futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now.AddDate(0, 0, -20), EndDate: now}, err: futures.ErrOrderHistoryTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetFuturesPositionOrders(t.Context(), &tc.r)
			assert.ErrorIsf(t, err, tc.err, "GetFuturesPositionOrders should refuse %+v", tc.r)
		})
	}
	orderArray := func(id uint64, created time.Time, status string) string {
		return `[` + strconv.FormatUint(id, 10) + `,null,null,"tBTCF0:USTF0",` + strconv.FormatInt(created.UnixMilli(), 10) + `,` + strconv.FormatInt(created.UnixMilli(), 10) +
			`,0,0.1,"LIMIT",null,null,null,0,"` + status + `",null,null,80000,80000,0,0,null,null,null,0,0,null,null,null,"API>BFX",null,null,{"$F33":10}]`
	}
	var historyBodies []map[string]any
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/v2/auth/r/orders/tBTCF0:USTF0/hist":
			var req map[string]any
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "Decoding the order history request should not error")
			historyBodies = append(historyBodies, req)
			body = `[` + orderArray(2, now.Add(-30*time.Minute), "EXECUTED @ 80000.0(0.1)") + `]`
		case "/v2/auth/r/orders/tBTCF0:USTF0":
			body = `[` + orderArray(3, now.Add(-10*time.Minute), "ACTIVE") + `,` + orderArray(4, now.AddDate(0, 0, -15), "ACTIVE") + `]`
		default:
			assert.Failf(t, "the test server should receive only the requests expected", "%s %s", r.Method, r.URL.Path)
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	got, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now.Add(-time.Hour), EndDate: now})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	require.Len(t, got, 1, "GetFuturesPositionOrders must return the pair's orders")
	assert.Equal(t, testDerivativePair, got[0].Pair, "GetFuturesPositionOrders should return the pair's orders")
	assert.Equal(t, futures.Linear, got[0].ContractSettlementType, "Bitfinex's derivatives should settle linearly")
	ids := make([]string, len(got[0].Orders))
	for i := range got[0].Orders {
		ids[i] = got[0].Orders[i].OrderID
		assert.Equal(t, asset.USDTMarginedFutures, got[0].Orders[i].AssetType, "each order should be the derivative's")
	}
	assert.Equal(t, []string{"3", "2"}, ids, "GetFuturesPositionOrders should return the window's open and closed orders")
	require.Len(t, historyBodies, 1, "GetFuturesPositionOrders must read one page of the order history")
	assert.Equal(t, map[string]any{"start": float64(now.Add(-time.Hour).UnixMilli()), "end": float64(now.UnixMilli()), "limit": float64(ordersHistoryPageLimit)}, historyBodies[0], "the order history should be read for the window")

	historyBodies = nil
	earliest := time.Now().Add(-orderHistoryReach).UnixMilli()
	_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now.AddDate(0, 0, -20), EndDate: now, RespectOrderHistoryLimits: true})
	latest := time.Now().Add(-orderHistoryReach).UnixMilli()
	require.NoError(t, err, "GetFuturesPositionOrders must not error for a window starting before the order history")
	require.Len(t, historyBodies, 1, "GetFuturesPositionOrders must read the order history")
	historyStart, ok := historyBodies[0]["start"].(float64)
	require.True(t, ok, "the order history request must have a start")
	assert.GreaterOrEqual(t, historyStart, float64(earliest), "a window starting before the order history should start as far back as it reaches")
	assert.LessOrEqual(t, historyStart, float64(latest), "a window starting before the order history should start as far back as it reaches")

	req := &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now.Add(-time.Hour), EndDate: now}
	var apiErr *APIError
	_, err = newRefusingServerExchange(t, `["error",10100,"apikey: invalid"]`).GetFuturesPositionOrders(t.Context(), req)
	assert.ErrorAs(t, err, &apiErr, "GetFuturesPositionOrders should return the order history's refusal")
	apiErr = nil
	_, err = newPathResponseServerExchange(t, map[string]string{"/v2/auth/r/orders/tBTCF0:USTF0": `["error",10100,"apikey: invalid"]`}).GetFuturesPositionOrders(t.Context(), req)
	assert.ErrorAs(t, err, &apiErr, "GetFuturesPositionOrders should return the open orders' refusal")
}

func TestChangePositionMargin(t *testing.T) { //nolint:tparallel // serial subtests share e in order
	t.Parallel()
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must reject a nil request")
	// The cases share e in order, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		name string
		r    margin.PositionChangeRequest
		err  error
	}{
		{name: "margin", r: margin.PositionChangeRequest{Asset: asset.Margin, Pair: currency.NewBTCUSD(), MarginType: margin.Isolated, NewAllocatedMargin: 1}, err: futures.ErrNotPerpetualFuture},
		{name: "multi margin", r: margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Multi, NewAllocatedMargin: 1}, err: margin.ErrMarginTypeUnsupported},
		{name: "no allocated margin", r: margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Isolated}, err: margin.ErrNewAllocatedMarginRequired},
		{name: "another asset's pair", r: margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testCoinMarginedPair, MarginType: margin.Isolated, NewAllocatedMargin: 1}, err: errPairNotOfAsset},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.ChangePositionMargin(t.Context(), &tc.r)
			assert.ErrorIsf(t, err, tc.err, "ChangePositionMargin should refuse %+v", tc.r)
		})
	}
	_, err = newResponseServerExchange(t, `[[0]]`).ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Isolated, NewAllocatedMargin: 1})
	assert.ErrorIs(t, err, errPositionCollateralNotSet, "ChangePositionMargin should report collateral Bitfinex did not set")
	var apiErr *APIError
	_, err = newResponseServerExchange(t, `["error",10100,"apikey: invalid"]`).ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Isolated, NewAllocatedMargin: 1})
	assert.ErrorAs(t, err, &apiErr, "ChangePositionMargin should return the error Bitfinex reports")
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Setting a live position's collateral needs a position, which the tests do not open")
	}
	resp, err := e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Exchange: e.Name, Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Isolated, NewAllocatedMargin: 1150.61})
	require.NoError(t, err, "ChangePositionMargin must not error")
	assert.Equal(t, &margin.PositionChangeResponse{
		Exchange: e.Name, Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Isolated, AllocatedMargin: 1150.61,
	}, resp, "ChangePositionMargin should return the position's new collateral")
}
