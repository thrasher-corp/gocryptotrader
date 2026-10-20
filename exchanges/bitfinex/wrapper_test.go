package bitfinex

import (
	"cmp"
	"encoding/base64"
	"errors"
	"io"
	"math"
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

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must refuse an asset Bitfinex does not trade")
	derivative := func(base string) currency.Pair {
		return currency.NewPair(currency.NewCode(base), currency.NewCode("USTF0"))
	}
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
			continue
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

func TestUpdateOrderExecutionLimits(t *testing.T) {
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
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
		l, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, a, p))
		require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", a, p)
		if mockTests {
			assert.Equalf(t, 4.0, l.MinimumBaseAmount, "the %s minimum order should be Bitfinex's", a)
			assert.Equalf(t, 250000.0, l.MaximumBaseAmount, "the %s maximum order should be Bitfinex's", a)
			continue
		}
		assert.Positivef(t, l.MinimumBaseAmount, "the %s minimum order should be set", a)
		assert.Greaterf(t, l.MaximumBaseAmount, l.MinimumBaseAmount, "the %s maximum order should exceed the minimum", a)
	}
	for _, tc := range []struct {
		a             asset.Item
		p, other      currency.Pair
		minimum, most float64
	}{
		{a: asset.USDTMarginedFutures, p: testDerivativePair, other: testCoinMarginedPair, minimum: 0.00004, most: 100},
		{a: asset.CoinMarginedFutures, p: testCoinMarginedPair, other: testDerivativePair, minimum: 0.0008, most: 100},
	} {
		require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.a), "UpdateOrderExecutionLimits must not error for %s", tc.a)
		l, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, tc.a, tc.p))
		require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", tc.a, tc.p)
		_, err = limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, tc.a, tc.other))
		assert.ErrorIsf(t, err, limits.ErrOrderLimitNotFound, "%s should hold no limits for %s, which another asset trades", tc.a, tc.other)
		if mockTests {
			assert.Equalf(t, tc.minimum, l.MinimumBaseAmount, "the %s minimum order should be Bitfinex's", tc.a)
			assert.Equalf(t, tc.most, l.MaximumBaseAmount, "the %s maximum order should be Bitfinex's", tc.a)
			continue
		}
		assert.Positivef(t, l.MinimumBaseAmount, "the %s minimum order should be set", tc.a)
		assert.Greaterf(t, l.MaximumBaseAmount, l.MinimumBaseAmount, "the %s maximum order should exceed the minimum", tc.a)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers must refuse an asset Bitfinex does not trade")
	for a, pairs := range map[asset.Item]currency.Pairs{
		asset.Spot:                {currency.NewBTCUSD(), testETHUSD},
		asset.Margin:              {currency.NewBTCUSD()},
		asset.MarginFunding:       {testFundingUSD},
		asset.USDTMarginedFutures: {testDerivativePair},
		asset.CoinMarginedFutures: {testCoinMarginedPair},
	} {
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
		for _, p := range pairs {
			got, err := ticker.GetTicker(ex.Name, p, a)
			require.NoErrorf(t, err, "UpdateTickers must store %s %s's ticker", a, p)
			assert.Positivef(t, got.Last, "the %s %s ticker should have a last price", a, p)
			assert.WithinDurationf(t, time.Now(), got.LastUpdated, time.Minute, "the %s %s ticker should be stamped when received, not with its first trade", a, p)
		}
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

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must refuse an empty pair")
	_, err = ex.UpdateTicker(t.Context(), currency.NewBTCUSD(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must refuse an asset Bitfinex has no symbol for")
	_, err = ex.UpdateTicker(t.Context(), currency.NewBTCUSD(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errPairNotOfAsset, "UpdateTicker must refuse a pair the asset does not trade")
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		got, err := ex.UpdateTicker(t.Context(), p, a)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s", a)
		assert.Positivef(t, got.Last, "the %s ticker should have a last price", a)
		assert.Equalf(t, a, got.AssetType, "the %s ticker should be stored for its asset", a)
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

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must refuse an empty pair")
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		book, err := ex.UpdateOrderbook(t.Context(), p, a)
		require.NoErrorf(t, err, "UpdateOrderbook must not error for %s", a)
		require.NotEmptyf(t, book.Bids, "the %s book must hold bids", a)
		require.NotEmptyf(t, book.Asks, "the %s book must hold asks", a)
		assert.Equalf(t, a == asset.MarginFunding, book.IsFundingRate, "the %s book should be marked a funding book only for funding", a)
		assert.NotZerof(t, book.Bids[0].ID, "the %s book should hold orders by ID", a)
		if a == asset.MarginFunding {
			assert.NotZero(t, book.Bids[0].Period, "the funding book should hold each offer's period")
		}
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must refuse an asset Bitfinex has no wallet for")
	skipLiveWithoutCredentials(t)
	ex := newWrapperTestExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.MarginFunding, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		subAccts, err := ex.UpdateAccountBalances(t.Context(), a)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
		require.Lenf(t, subAccts, 1, "UpdateAccountBalances must return %s's wallet", a)
		assert.Equalf(t, a, subAccts[0].AssetType, "UpdateAccountBalances should return the %s wallet", a)
		if !mockTests {
			continue
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
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	assert.NoError(t, newWrapperTestExchange(t).ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	history, err := e.GetAccountFundingHistory(t.Context())
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

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	for _, c := range []currency.Code{currency.ETH, currency.EMPTYCODE} {
		history, err := e.GetWithdrawalsHistory(t.Context(), c, asset.Spot)
		require.NoErrorf(t, err, "GetWithdrawalsHistory must not error for currency %q", c)
		for i := range history {
			assert.Equal(t, "withdrawal", history[i].TransferType, "GetWithdrawalsHistory should return withdrawals alone")
		}
		if mockTests {
			require.Lenf(t, history, 1, "GetWithdrawalsHistory must leave out the deposit for currency %q", c)
			assert.Equal(t, 0.26300954, history[0].Amount, "a withdrawal's amount should be positive")
		}
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
	for c, path := range map[currency.Code]string{currency.EMPTYCODE: "/v2/auth/r/movements/hist", currency.ETH: "/v2/auth/r/movements/ETH/hist"} {
		got, err := ex.getMovementsHistory(t.Context(), c)
		require.NoErrorf(t, err, "getMovementsHistory must not error for currency %q", c)
		require.Lenf(t, got, 1, "getMovementsHistory must return the movement for currency %q", c)
		assert.Equalf(t, uint64(13293039), got[0].ID, "getMovementsHistory should return the movement for currency %q", c)
		mu.Lock()
		assert.Equalf(t, path, paths[len(paths)-1], "getMovementsHistory should read the movements of currency %q", c)
		mu.Unlock()
	}
	_, err := newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).getMovementsHistory(t.Context(), currency.ETH)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "getMovementsHistory should return the error Bitfinex reports")
}

func TestMovementChain(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tetheruse", movementChain(&MovementResponse{Method: "TETHERUSE"}), "movementChain should give the method in lower case, as GetAvailableTransferChains does")
}

func TestMovementType(t *testing.T) {
	t.Parallel()
	for amount, exp := range map[types.Number]string{-0.24: "withdrawal", 0.26: "deposit"} {
		assert.Equalf(t, exp, movementType(&MovementResponse{Amount: amount}), "movementType should call a movement of %v a %s", amount, exp)
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
	// serve returns a fetch serving history as Bitfinex does, newest first, records up to the end and from the start
	serve := func(history []record, start time.Time, limit int, ends *[]time.Time) func(time.Time) ([]record, error) {
		return func(end time.Time) ([]record, error) {
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
		return history[:3], nil
	}
	_, err = pageBack(time.Time{}, time.Time{}, 3, ignoringEnd, id, at)
	assert.ErrorIs(t, err, errHistoryPageFull, "pageBack should error for a server ignoring the end")
	assert.Equal(t, 2, calls, "pageBack should stop paging a server ignoring the end at its second page")

	errPageRefused := errors.New("page refused")
	_, err = pageBack(time.Time{}, time.Time{}, 3, func(time.Time) ([]record, error) { return nil, errPageRefused }, id, at)
	assert.ErrorIs(t, err, errPageRefused, "pageBack should return a page's error")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		trades, err := ex.GetRecentTrades(t.Context(), p, a)
		require.NoErrorf(t, err, "GetRecentTrades must not error for %s", a)
		require.NotEmptyf(t, trades, "GetRecentTrades must return %s trades", a)
		assert.Falsef(t, trades[0].Timestamp.After(trades[len(trades)-1].Timestamp), "GetRecentTrades should return %s trades oldest first", a)
		for i := range trades {
			assert.Positivef(t, trades[i].Amount, "a %s trade's amount should be positive, its side giving its sign", a)
		}
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := historyWindow(time.Hour)
	_, err := ex.GetHistoricTrades(t.Context(), currency.NewBTCUSD(), asset.Spot, end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades must refuse a window ending before it starts")
	trades, err := ex.GetHistoricTrades(t.Context(), currency.NewBTCUSD(), asset.Spot, start, end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	require.NotEmpty(t, trades, "GetHistoricTrades must return trades")
	for i := range trades {
		assert.Falsef(t, trades[i].Timestamp.Before(start) || trades[i].Timestamp.After(end), "trade %s should be within the window", trades[i].TID)
	}
}

func TestPlaceOrderRequest(t *testing.T) {
	t.Parallel()
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

func TestSubmitOrder(t *testing.T) {
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

	// Bitfinex places each order with its price at 90: a stop's and a stop limit's trigger, and a trailing stop's stop
	placed := newResponseServerExchange(t, `[1678987199,"on-req",null,null,[[1747566431,null,1678987199446,"tBTCUSD",1678987199446,1678987199448,-1,-1,"EXCHANGE STOP",null,null,null,0,"ACTIVE",null,null,90,0,0,0,null,null,null,0,0,null,null,null,"API>BFX",null,null,null]],null,"SUCCESS","Submitting exchange stop sell order for 1 BTC."]`)
	for _, tc := range []struct {
		s                   order.Submit
		price, triggerPrice float64
	}{
		{s: order.Submit{Type: order.Stop, Price: 89, TriggerPrice: 90}, triggerPrice: 90},
		{s: order.Submit{Type: order.StopLimit, Price: 89, TriggerPrice: 90}, price: 89, triggerPrice: 90},
		{s: order.Submit{Type: order.TrailingStop, Price: 89, TriggerPrice: 70, TrackingMode: order.Distance, TrackingValue: 5}, triggerPrice: 90},
	} {
		tc.s.Exchange, tc.s.AssetType, tc.s.Pair, tc.s.Side, tc.s.Amount = placed.Name, asset.Spot, currency.NewBTCUSD(), order.Sell, 1
		resp, err := placed.SubmitOrder(t.Context(), &tc.s)
		require.NoErrorf(t, err, "SubmitOrder must not error for a %s order", tc.s.Type)
		assert.Equalf(t, tc.price, resp.Price, "SubmitOrder should report a %s order's limit price, which neither stop has", tc.s.Type)
		assert.Equalf(t, tc.triggerPrice, resp.TriggerPrice, "SubmitOrder should report a %s order's trigger price, a trailing stop's as Bitfinex placed it", tc.s.Type)
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

func TestSetDefaultsTransferChains(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	f := ex.Features.Supports.RESTCapabilities
	assert.True(t, f.MultiChainDeposits, "deposits should be supported over each of a currency's methods")
	assert.True(t, f.MultiChainWithdrawals, "withdrawals should be supported over each of a currency's methods")
	assert.True(t, f.MultiChainDepositRequiresChainSet, "a deposit address should need its method, as a currency with several has no default")
}

func TestModifyOrder(t *testing.T) {
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
		for _, tc := range []struct {
			side   order.Side
			amount float64
			exp    map[string]any
		}{
			{side: order.Buy, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "0.25", "price": "25000"}},
			{side: order.Bid, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "0.25", "price": "25000"}},
			{side: order.Long, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "0.25", "price": "25000"}},
			{side: order.Sell, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "-0.25", "price": "25000"}},
			{side: order.Ask, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "-0.25", "price": "25000"}},
			{side: order.Short, amount: 0.25, exp: map[string]any{"id": 1.0, "amount": "-0.25", "price": "25000"}},
			{side: order.Sell, exp: map[string]any{"id": 1.0, "price": "25000"}},
			{side: order.UnknownSide, exp: map[string]any{"id": 1.0, "price": "25000"}},
		} {
			_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "1", AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: tc.side, Amount: tc.amount, Price: 25000})
			require.NoErrorf(t, err, "ModifyOrder must not error for a %s modification of %v", tc.side, tc.amount)
			sent := updates()
			require.NotEmptyf(t, sent, "ModifyOrder must send the %s modification", tc.side)
			assert.Equalf(t, tc.exp, sent[len(sent)-1], "ModifyOrder should send a %s amount of %v signed by its side, a sell's negative, and none when only the price changes", tc.side, tc.amount)
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
			tc.m.AssetType, tc.m.Pair = asset.Spot, currency.NewBTCUSD()
			resp, err := ex.ModifyOrder(t.Context(), &tc.m)
			require.NoErrorf(t, err, "ModifyOrder must not error for %s", tc.name)
			sent := updates()
			require.NotEmptyf(t, sent, "ModifyOrder must send %s", tc.name)
			assert.Equalf(t, tc.exp, sent[len(sent)-1], "ModifyOrder should send %s as Bitfinex takes it for the order's type", tc.name)
			assert.Equalf(t, tc.price, resp.Price, "ModifyOrder should report the price it sent for %s, and none for a stop", tc.name)
			assert.Equalf(t, tc.m.TriggerPrice, resp.TriggerPrice, "ModifyOrder should report the trigger price it sent for %s", tc.name)
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
		)
		// The order manager fills a modification's amount from the stored order, its whole amount
		resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 1, Price: 101})
		require.NoError(t, err, "ModifyOrder must not error for a price change")
		require.Len(t, updates(), 1, "ModifyOrder must send the price change")
		assert.Equal(t, map[string]any{"id": 7.0, "price": "101"}, updates()[0], "ModifyOrder should send no amount for a partly filled order whose whole amount is unchanged, leaving what remains to Bitfinex")
		assert.Equal(t, 0.6, resp.RemainingAmount, "ModifyOrder should report what remains of the order")
		assert.Equal(t, 1.0, resp.Amount, "ModifyOrder should report the order's whole amount")

		_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 0.5})
		require.NoError(t, err, "ModifyOrder must not error for an amount change")
		require.Len(t, updates(), 2, "ModifyOrder must send the amount change")
		assert.Equal(t, map[string]any{"id": 7.0, "amount": "0.1"}, updates()[1], "ModifyOrder should send what remains of the new amount, worked out without rounding error")

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
		for _, tc := range []struct {
			name       string
			amount     float64
			exp        map[string]any
			tradeReads int
		}{
			{name: "the whole amount the order is reported with, as the order manager sends", amount: 1, exp: map[string]any{"id": 18.0, "price": "101"}},
			{name: "the whole amount the order is reported with, carried as a float", amount: math.Nextafter(1, 2), exp: map[string]any{"id": 18.0, "price": "101"}},
			{name: "the whole amount the order has, by its trades", amount: 0.8, exp: map[string]any{"id": 18.0, "price": "101"}, tradeReads: 1},
			{name: "a new whole amount, less what its trades executed", amount: 1.2, exp: map[string]any{"id": 18.0, "amount": "0.7", "price": "101"}, tradeReads: 2},
		} {
			_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "18", AssetType: asset.Margin, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: tc.amount, Price: 101})
			require.NoErrorf(t, err, "ModifyOrder must not error for %s", tc.name)
			mu.Lock()
			assert.Equalf(t, tc.exp, sent[len(sent)-1], "ModifyOrder should send what remains of %s, unsent when unchanged", tc.name)
			assert.Equalf(t, tc.tradeReads, tradeReads, "ModifyOrder should read the order's trades only for an amount it changes, for %s", tc.name)
			mu.Unlock()
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
		// The order manager fills a modification's time in force from the stored order
		for _, orderType := range []order.Type{order.UnknownType, order.AnyType, order.Limit} {
			resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "16", AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: orderType, TimeInForce: order.GoodTillCancel, Price: 101})
			require.NoErrorf(t, err, "ModifyOrder must not error for a modification of type %s", orderType)
			assert.Equalf(t, order.Limit, resp.Type, "ModifyOrder should report the order's type for a modification of type %s", orderType)
			assert.Equalf(t, order.PostOnly, resp.TimeInForce, "ModifyOrder should report the order's time in force, which Bitfinex does not update, for a modification of type %s", orderType)
		}
		require.Len(t, updates(), 3, "ModifyOrder must send each modification of the order's type")
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "16", AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Type: order.Stop, Price: 95})
		assert.ErrorIs(t, err, errOrderTypeUnchangeable, "ModifyOrder should refuse to make the order another type")
		assert.Len(t, updates(), 3, "ModifyOrder should send nothing for a modification it refuses")
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
			tc.m.AssetType, tc.m.Pair = asset.Spot, currency.NewBTCUSD()
			_, err := ex.ModifyOrder(t.Context(), &tc.m)
			assert.ErrorIsf(t, err, tc.err, "ModifyOrder should refuse %s", tc.name)
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

	wsEx, venue := newWebsocketTestExchange(t, subscription.List{})
	require.True(t, wsEx.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the wrapper must be able to use the authenticated websocket")
	_, err = wsEx.ModifyOrder(t.Context(), &order.Modify{OrderID: testWrapperOrderID, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Amount: 0.25, Price: 25000})
	require.NoError(t, err, "ModifyOrder must not error through the websocket")
	inputs := venue.inputs(wsOrderUpdate)
	require.Len(t, inputs, 1, "ModifyOrder must send one update through the websocket")
	assert.JSONEq(t, `{"id":1747566428,"amount":"0.25","price":"25000"}`, string(inputs[0]), "ModifyOrder should send the update through the websocket")
	_, err = wsEx.ModifyOrder(t.Context(), &order.Modify{OrderID: testWrapperOrderID, AssetType: asset.Spot, Pair: currency.NewBTCUSD(), Side: order.Buy, Price: -1})
	assert.ErrorIs(t, err, errInvalidPrice, "ModifyOrder should return the websocket update's error")
}

func TestSameAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b float64
		exp  bool
	}{
		{a: 1, b: 1, exp: true},
		{a: 0.949681784542512, b: 0.9496817845425121, exp: true},
		{a: 1.918933389737914, b: 1.9189333897379142, exp: true},
		{a: 0.6, b: 0.60000001},
		{a: 10000.12345678, b: 10000.12345679},
		{a: 10000000.12345678, b: 10000000.12345679},
		{a: 1, b: -1},
		{a: math.NaN(), b: math.NaN()},
		{a: math.Inf(1), b: math.Inf(1)},
	} {
		assert.Equalf(t, tc.exp, sameAmount(tc.a, tc.b), "sameAmount should report whether %v and %v are the same amount to Bitfinex's 8 decimal places", tc.a, tc.b)
	}
}

func TestSetUpdatePrices(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		m         order.Modify
		exp       UpdateOrderRequest
		err       error
	}{
		{orderType: order.Limit, m: order.Modify{Price: 100}, exp: UpdateOrderRequest{Price: 100}},
		{orderType: order.Market, m: order.Modify{Price: 100}, exp: UpdateOrderRequest{Price: 100}},
		{orderType: order.Limit, m: order.Modify{Price: 100, TriggerPrice: 90}, err: errTriggerPriceUnsupported},
		{orderType: order.Stop, m: order.Modify{Price: 100, TriggerPrice: 90}, exp: UpdateOrderRequest{Price: 90}},
		{orderType: order.Stop, m: order.Modify{TriggerPrice: 90}, exp: UpdateOrderRequest{Price: 90}},
		{orderType: order.Stop, m: order.Modify{Price: 100}, err: errTriggerPriceRequired},
		{orderType: order.Stop},
		{orderType: order.StopLimit, m: order.Modify{Price: 89}, exp: UpdateOrderRequest{AuxiliaryLimitPrice: 89}},
		{orderType: order.StopLimit, m: order.Modify{Price: 89, TriggerPrice: 90}, exp: UpdateOrderRequest{Price: 90, AuxiliaryLimitPrice: 89}},
		{orderType: order.TrailingStop},
		{orderType: order.TrailingStop, m: order.Modify{Price: 95}, err: errPriceUnsupported},
		{orderType: order.TrailingStop, m: order.Modify{TriggerPrice: 95}, err: errPriceUnsupported},
	} {
		var got UpdateOrderRequest
		err := setUpdatePrices(&got, tc.orderType, &tc.m)
		require.ErrorIsf(t, err, tc.err, "setUpdatePrices must return the expected error for a %s order", tc.orderType)
		assert.Equalf(t, tc.exp, got, "setUpdatePrices should set the prices a %s order is updated by", tc.orderType)
	}
}

func TestRemainingOrderAmount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		amount   float64
		executed string
		exp      float64
		err      error
	}{
		{amount: 1, executed: "0", exp: 1},
		{amount: 1, executed: "0.4", exp: 0.6},
		{amount: 0.3, executed: "0.1", exp: 0.2},
		{amount: 0.3, executed: "0.3", err: errInvalidAmount},
		{amount: 0.25, executed: "0.4", err: errInvalidAmount},
		{amount: -1, executed: "0", err: errInvalidAmount},
		{amount: math.NaN(), executed: "0", err: errInvalidAmount},
		{amount: math.Inf(1), executed: "0", err: errInvalidAmount},
	} {
		got, err := remainingOrderAmount(tc.amount, 1, decimal.MustFromString(tc.executed))
		require.ErrorIsf(t, err, tc.err, "remainingOrderAmount must return the expected error for %v of an order that executed %s", tc.amount, tc.executed)
		assert.Equalf(t, tc.exp, got, "remainingOrderAmount should leave what an order of %v that executed %s has yet to execute", tc.amount, tc.executed)
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
		_, err := orderLeverage(&OrderResponse{ID: 1, Meta: json.RawMessage(tc.meta)})
		assert.ErrorIsf(t, err, tc.err, "orderLeverage should return the expected error for meta data %s", tc.meta)
	}
}

func TestDerivativeLeverage(t *testing.T) {
	t.Parallel()
	for _, leverage := range []float64{1, 20, 100} {
		got, err := derivativeLeverage(leverage)
		require.NoErrorf(t, err, "derivativeLeverage must not error for %v", leverage)
		assert.Equalf(t, uint64(leverage), got, "derivativeLeverage should take leverage %v as it is", leverage)
	}
	for _, leverage := range []float64{0, 0.5, 2.5, 101, -1} {
		_, err := derivativeLeverage(leverage)
		assert.ErrorIsf(t, err, errInvalidLeverage, "derivativeLeverage should refuse leverage %v", leverage)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
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

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("Cancelling every order through the websocket would reach the account's own")
	}
	wsEx, venue := newWebsocketTestExchange(t, subscription.List{})
	got, err = wsEx.submitOrderCancellations(t.Context(), &CancelOrdersRequest{All: true}, []uint64{1747566428, 1747566429})
	require.NoError(t, err, "submitOrderCancellations must not error through the websocket")
	assert.Equal(t, map[string]string{"1747566428": order.PendingCancel.String(), "1747566429": order.PendingCancel.String()}, got, "submitOrderCancellations should report each order listed pending cancellation through the websocket, whose notification names none")
	inputs := venue.inputs(wsOrderCancelMulti)
	require.Len(t, inputs, 1, "submitOrderCancellations must send one cancellation through the websocket")
	assert.JSONEq(t, `{"all":1}`, string(inputs[0]), "submitOrderCancellations should send the request it is given through the websocket")
	_, err = wsEx.submitOrderCancellations(t.Context(), &CancelOrdersRequest{}, nil)
	assert.ErrorIs(t, err, errOrdersToCancelRequired, "submitOrderCancellations should return the websocket cancellation's error")
}

func TestCancelAllOrders(t *testing.T) {
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

func TestGetOrderInfo(t *testing.T) {
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
		got, err := ex.GetOrderInfo(t.Context(), exp.OrderID, exp.Pair, exp.AssetType)
		require.NoErrorf(t, err, "GetOrderInfo must not error for order %s in the history", exp.OrderID)
		assert.Equalf(t, exp, got, "GetOrderInfo should return order %s from the history, closed, with each trade's fee as charged, a rebate negative, its currency and whether it was made", exp.OrderID)
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

	// The order was read at 1678987199448 unless given another update time
	read := func(status string, remaining, original float64, updatedAt string) string {
		return `[` + (&testOrderRecord{
			id: 33963608936, symbol: "tETHUSD", orderType: "EXCHANGE LIMIT", status: status, amount: remaining, original: original, price: 100, updatedAt: updatedAt,
		}).String() + `]`
	}
	trade := func(id, at, amount string) string {
		return `[` + id + `,"tETHUSD",` + at + `,33963608936,` + amount + `,100,"EXCHANGE LIMIT",100,1,-0.01,"ETH",0]`
	}
	for _, tc := range []struct {
		name                        string
		open, history               []string
		trades                      string
		status                      order.Status
		amount, executed, remaining float64
		openReads, historyReads     int
	}{
		{
			name: "no fill since the order was read", open: []string{read("PARTIALLY FILLED @ 100.0(0.4)", 0.6, 1, "")}, trades: `[` + trade("1", "1678987199000", "0.4") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.4, remaining: 0.6, openReads: 1,
		},
		{
			name:   "a fill since the order was read, read again",
			open:   []string{read("PARTIALLY FILLED @ 100.0(0.4)", 0.6, 1, ""), read("PARTIALLY FILLED @ 100.0(0.4), PARTIALLY FILLED @ 100.0(0.1)", 0.5, 1, "1678987200000")},
			trades: `[` + trade("1", "1678987199000", "0.4") + `,` + trade("2", "1678987200000", "0.1") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.5, remaining: 0.5, openReads: 2,
		},
		{
			name:    "the order filled since it was read, read again from the history",
			open:    []string{read("ACTIVE", 1, 1, ""), `[]`},
			history: []string{read("EXECUTED @ 100.0(1)", 0, 1, "1678987200000")},
			trades:  `[` + trade("1", "1678987200000", "1") + `]`,
			status:  order.Filled, amount: 1, executed: 1, openReads: 2, historyReads: 1,
		},
		{
			name: "a fill its second read predates too, left to its next read", open: []string{read("ACTIVE", 1, 1, "")}, trades: `[` + trade("1", "1678987200000", "0.25") + `]`,
			status: order.Active, amount: 1, remaining: 1, openReads: 2,
		},
		{
			name: "a fill made in the millisecond the order was read, which its status does not list, read again",
			open: []string{
				read("PARTIALLY FILLED @ 100.0(0.25)", 0.75, 1, ""),
				read("PARTIALLY FILLED @ 100.0(0.25), PARTIALLY FILLED @ 100.0(0.5)", 0.25, 1, ""),
			},
			trades: `[` + trade("1", "1678987199000", "0.25") + `,` + trade("2", "1678987199448", "0.5") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.75, remaining: 0.25, openReads: 2,
		},
		{
			name: "a fill its status lists in neither read, left to its next read", open: []string{read("PARTIALLY FILLED @ 100.0(0.25)", 0.75, 1, "")},
			trades: `[` + trade("1", "1678987199000", "0.25") + `,` + trade("2", "1678987199448", "0.5") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.25, remaining: 0.75, openReads: 2,
		},
		{
			name: "an order without its update time, whose trades cannot be told from its read", open: []string{read("PARTIALLY FILLED @ 100.0(0.4)", 0.6, 1, "null")},
			trades: `[` + trade("1", "1678987199000", "0.4") + `,` + trade("2", "1678987200000", "0.1") + `]`,
			status: order.PartiallyFilled, amount: 1, executed: 0.4, remaining: 0.6, openReads: 1,
		},
	} {
		reread, reads := newOrderReadsTestExchange(t, tc.open, tc.history, tc.trades)
		info, err := reread.GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
		require.NoErrorf(t, err, "GetOrderInfo must not error for %s", tc.name)
		assert.Equalf(t, tc.status, info.Status, "GetOrderInfo should report the status of %s", tc.name)
		assert.Equalf(t, [3]float64{tc.amount, tc.executed, tc.remaining}, [3]float64{info.Amount, info.ExecutedAmount, info.RemainingAmount},
			"GetOrderInfo should report the whole amount, what was executed and what remains of %s", tc.name)
		assert.Equalf(t, [3]int{tc.openReads, tc.historyReads, tc.openReads}, reads(), "GetOrderInfo should read the order as often as %s needs, and its trades after each read", tc.name)
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
	for _, tc := range []struct {
		name                        string
		createdAt, trades           string
		amount, executed, remaining float64
	}{
		{
			name: "a cut status of an order within its trades' reach", createdAt: ago(2 * time.Hour), amount: 0.8, executed: 0.5, remaining: 0.3,
			trades: `[` + trade("1", ago(100*time.Minute), "0.1") + `,` + trade("2", ago(90*time.Minute), "0.2") + `,` + trade("3", ago(80*time.Minute), "0.2") + `]`,
		},
		{
			name: "a cut status of an order beyond its trades' reach, whose original amount less what remains is taken", createdAt: ago(11 * 24 * time.Hour), amount: 1, executed: 0.7, remaining: 0.3,
			trades: `[` + trade("2", ago(90*time.Minute), "0.2") + `,` + trade("3", ago(80*time.Minute), "0.2") + `]`,
		},
	} {
		o := testOrderRecord{id: 33963608936, symbol: "tETHUSD", orderType: "EXCHANGE LIMIT", status: cut, amount: 0.3, original: 1, price: 100, createdAt: tc.createdAt, updatedAt: ago(time.Hour)}
		cutEx, _ := newOrderReadsTestExchange(t, []string{`[` + o.String() + `]`}, nil, tc.trades)
		info, err := cutEx.GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
		require.NoErrorf(t, err, "GetOrderInfo must not error for %s", tc.name)
		assert.Equalf(t, order.PartiallyFilled, info.Status, "GetOrderInfo should report %s by its last fill", tc.name)
		assert.Equalf(t, [3]float64{tc.amount, tc.executed, tc.remaining}, [3]float64{info.Amount, info.ExecutedAmount, info.RemainingAmount},
			"GetOrderInfo should report the whole amount, what was executed and what remains of %s", tc.name)
	}

	const refused = `["error",10020,"symbol: invalid"]`
	open := `[` + (&testOrderRecord{id: 33963608936, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}).String() + `]`
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
		_, err := newPathResponseServerExchange(t, tc.bodies).GetOrderInfo(t.Context(), "33963608936", currency.EMPTYPAIR, asset.Empty)
		if tc.refused {
			var apiErr *APIError
			assert.ErrorAsf(t, err, &apiErr, "GetOrderInfo should return the error Bitfinex reports with %s", tc.name)
			continue
		}
		assert.ErrorIsf(t, err, tc.err, "GetOrderInfo should return the expected error for %s", tc.name)
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
		{name: "a fill after the read of an order without its update time", o: &OrderResponse{Status: "ACTIVE"}, trades: []AccountTradeResponse{trade(read, 0.25)}},
	} {
		assert.Equalf(t, tc.exp, isOrderReadBehindTrades(tc.o, tc.trades), "isOrderReadBehindTrades should report whether %s leaves the read behind", tc.name)
	}
}

func TestCutOrderExecutedAmount(t *testing.T) {
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
	for _, tc := range []struct {
		name   string
		o      *OrderResponse
		trades []AccountTradeResponse
		exp    string
	}{
		{name: "an order within its trades' reach", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.5"},
		{name: "an order beyond its trades' reach", o: &OrderResponse{CreatedAt: types.Time(read.Add(-11 * 24 * time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.7"},
		{name: "an order beyond its trades' reach whose trades count more than its estimate", o: &OrderResponse{UpdatedAt: types.Time(read), Amount: 0.9, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.5"},
		{name: "an order within its trades' reach whose trades count less than its status lists", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), UpdatedAt: types.Time(read), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades[:1], exp: "0.7"},
		{name: "an order without its update time", o: &OrderResponse{CreatedAt: types.Time(read.Add(-time.Hour)), Amount: 0.3, OriginalAmount: 1, Status: cut}, trades: trades, exp: "0.7"},
	} {
		got := cutOrderExecutedAmount(tc.o, tc.trades)
		assert.Truef(t, decimal.MustFromString(tc.exp).Equal(got), "cutOrderExecutedAmount should report %s executed for %s, got %s", tc.exp, tc.name, got)
	}
}

func TestOrderByID(t *testing.T) {
	t.Parallel()
	open := testOrderRecord{id: 1, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	other := testOrderRecord{id: 2, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "ACTIVE", amount: 1, original: 1, price: 100}
	closed := testOrderRecord{id: 3, symbol: "tBTCUSD", orderType: "EXCHANGE LIMIT", status: "EXECUTED @ 100.0(1)", original: 1, price: 100}
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
		ex, _ := newOrderReadsTestExchange(t, tc.open, tc.history, `[]`)
		got, gotClosed, err := ex.orderByID(t.Context(), tc.id)
		require.ErrorIsf(t, err, tc.err, "orderByID must return the expected error for %s", tc.name)
		if tc.err != nil {
			continue
		}
		assert.Equalf(t, tc.id, got.ID, "orderByID should return %s", tc.name)
		assert.Equalf(t, tc.closed, gotClosed, "orderByID should report whether %s is from the history", tc.name)
	}
	const refused = `["error",10020,"symbol: invalid"]`
	for _, tc := range []struct {
		name          string
		open, history []string
	}{
		{name: "the open orders", open: []string{refused}},
		{name: "the history", history: []string{refused}},
	} {
		ex, _ := newOrderReadsTestExchange(t, tc.open, tc.history, `[]`)
		_, _, err := ex.orderByID(t.Context(), 1)
		var apiErr *APIError
		assert.ErrorAsf(t, err, &apiErr, "orderByID should return the error Bitfinex reports reading %s", tc.name)
	}
}

func TestOpenOrderDetails(t *testing.T) {
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
	}
	_, err := ex.openOrderDetails(t.Context(), testFundingUSD, asset.MarginFunding)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "openOrderDetails should refuse funding, which takes offers rather than orders")
	_, err = newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).openOrderDetails(t.Context(), currency.EMPTYPAIR, asset.Spot)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "openOrderDetails should return the error Bitfinex reports")
}

func TestOrderDetails(t *testing.T) {
	t.Parallel()
	orders := []OrderResponse{
		{ID: 1, Symbol: "tBTCUSD", Type: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "ACTIVE", Amount: 1, OriginalAmount: 1},
		{ID: 2, Symbol: "tBTCUSD", Type: OrderTypeLimit, Status: "ACTIVE", Amount: -1, OriginalAmount: -1},
	}
	for a, ids := range map[asset.Item][]string{asset.Spot: {"1"}, asset.Margin: {"2"}, asset.Empty: {"1", "2"}} {
		got, err := e.orderDetails(orders, a)
		require.NoErrorf(t, err, "orderDetails must not error for %s", a)
		gotIDs := make([]string, len(got))
		for i := range got {
			gotIDs[i] = got[i].OrderID
		}
		assert.Equalf(t, ids, gotIDs, "orderDetails should return the %s orders alone, or every order without an asset", a)
	}
	_, err := e.orderDetails([]OrderResponse{{ID: 3, Symbol: "tBTCUSD", Type: "NOVEL", OriginalAmount: 1}}, asset.Spot)
	assert.ErrorIs(t, err, errUnknownOrderType, "orderDetails should return the error of an order it cannot convert")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		orders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
		for i := range orders {
			assert.Equalf(t, a, orders[i].AssetType, "GetActiveOrders should return %s orders alone", a)
		}
		if mockTests && a == asset.Margin {
			require.Len(t, orders, 1, "GetActiveOrders must return the open margin order")
			assert.Equal(t, "33950998275", orders[0].OrderID, "GetActiveOrders should return the open order")
		}
	}
}

func TestGetOrderHistory(t *testing.T) {
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
	for _, pairs := range []currency.Pairs{nil, {currency.NewPair(currency.ETH, currency.USD)}} {
		history, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: pairs, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetOrderHistory must not error for pairs %s", pairs)
		require.Lenf(t, history, 1, "GetOrderHistory must return the order for pairs %s", pairs)
		assert.Equalf(t, order.PartiallyFilledCancelled, history[0].Status, "GetOrderHistory should report a partially filled order from the history cancelled, its remainder no longer in the book, for pairs %s", pairs)
	}
	_, err := ex.GetOrderHistory(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory should refuse a nil request")
	_, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{testDerivativePair}, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errPairNotOfAsset, "GetOrderHistory should refuse a pair the asset does not trade")
	refusing := newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`)
	unknownType := newResponseServerExchange(t, `[[33963608937,null,null,"tETHUSD",null,null,1,1,"NOVEL"]]`)
	for _, pairs := range []currency.Pairs{nil, {currency.NewPair(currency.ETH, currency.USD)}} {
		req := &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: pairs, Side: order.AnySide, Type: order.AnyType}
		_, err = refusing.GetOrderHistory(t.Context(), req)
		var apiErr *APIError
		assert.ErrorAsf(t, err, &apiErr, "GetOrderHistory should return the error Bitfinex reports for pairs %s", pairs)
		_, err = unknownType.GetOrderHistory(t.Context(), req)
		assert.ErrorIsf(t, err, errUnknownOrderType, "GetOrderHistory should refuse an order of an unknown type for pairs %s", pairs)
	}

	skipLiveWithoutCredentials(t)
	for _, pairs := range []currency.Pairs{nil, {currency.NewBTCUSD()}} {
		orders, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: pairs, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetOrderHistory must not error for pairs %s", pairs)
		for i := range orders {
			assert.Equal(t, asset.Spot, orders[i].AssetType, "GetOrderHistory should return spot orders alone")
		}
	}
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

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
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

func TestWithdrawCryptocurrencyFundsTravelRule(t *testing.T) {
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
		{name: "provider", travel: withdraw.TravelRule{BeneficiaryFinancialInstitution: "Strawb Exchange Ltd"}, exp: &TravelRuleDetails{VASPName: "Strawb Exchange Ltd"}},
		{name: "person", travel: withdraw.TravelRule{BeneficiaryName: "  John   Doe "}, exp: &TravelRuleDetails{FirstName: "John", LastName: "Doe"}},
		{name: "one name", travel: withdraw.TravelRule{BeneficiaryName: "Satoshi"}, err: errTravelRuleNameIncomplete},
	} {
		got, err := withdrawalTravelRule(&tc.travel)
		require.ErrorIsf(t, err, tc.err, "withdrawalTravelRule must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "withdrawalTravelRule should return the travel rule details of %s", tc.name)
	}
}

func TestRequestFiatWithdrawalPayload(t *testing.T) {
	t.Parallel()
	var payload map[string]any
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded, err := base64.StdEncoding.DecodeString(r.Header.Get("X-BFX-PAYLOAD"))
		assert.NoError(t, err, "a v1 request's payload should be base64 encoded")
		payload = nil
		assert.NoError(t, json.Unmarshal(decoded, &payload), "a v1 request's payload should be JSON")
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

	w = newWithdrawal()
	w.Fiat.IsExpressWire = false
	w.Fiat.IntermediaryIBAN = "DE89370400440532013000"
	_, err = ex.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange, Withdrawal: w})
	require.NoError(t, err, "RequestFiatWithdrawal must not error")
	assert.NotContains(t, payload, "expressWire", "a normal wire should leave the express flag out")
	assert.Equal(t, "DE89370400440532013000", payload["intermediary_bank_account"], "a wire should send its intermediary bank's IBAN as its account")

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
		w = newWithdrawal()
		w.Fiat.IntermediaryBankAccountNumber = tc.account
		_, err = ex.RequestFiatWithdrawal(t.Context(), &FiatWithdrawalRequest{Wallet: WalletTypeExchange, Withdrawal: w})
		assert.ErrorIsf(t, err, tc.err, "RequestFiatWithdrawal should refuse intermediary account number %v", tc.account)
	}
}

func TestIntermediaryBankAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		f   withdraw.FiatRequest
		exp string
		err error
	}{
		{f: withdraw.FiatRequest{IntermediaryIBAN: "DE89370400440532013000", IntermediaryBankAccountNumber: 123456789}, exp: "DE89370400440532013000"},
		{f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 123456789}, exp: "123456789"},
		{f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 1<<53 - 1}, exp: "9007199254740991"},
		// 2^53 is held exactly, but so is the 2^53+1 that rounds to it
		{f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 1 << 53}, err: errIntermediaryBankAccountInexact},
		{f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 9007199254740993}, err: errIntermediaryBankAccountInexact},
		{f: withdraw.FiatRequest{IntermediaryBankAccountNumber: 1.5}, err: errIntermediaryBankAccountInexact},
		{f: withdraw.FiatRequest{IntermediaryBankAccountNumber: -1}, err: errIntermediaryBankAccountInexact},
		{f: withdraw.FiatRequest{}, err: errIntermediaryBankAccountRequired},
	} {
		got, err := intermediaryBankAccount(&tc.f)
		require.ErrorIsf(t, err, tc.err, "intermediaryBankAccount must return the expected error for %+v", tc.f)
		assert.Equalf(t, tc.exp, got, "intermediaryBankAccount should return the account a wire goes through for %+v", tc.f)
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

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")
	wires := newTestExchange(t)
	for _, tc := range []struct {
		feeType exchange.FeeType
		c       currency.Code
		amount  float64
		exp     float64
	}{
		{feeType: exchange.InternationalBankDepositFee, c: currency.USD, amount: 10000, exp: 60},
		{feeType: exchange.InternationalBankDepositFee, c: currency.EUR, amount: 100000, exp: 100},
		{feeType: exchange.InternationalBankWithdrawalFee, c: currency.USD, amount: 10000, exp: 100},
		{feeType: exchange.InternationalBankWithdrawalFee, c: currency.EUR, amount: 200000, exp: 200},
		{feeType: exchange.InternationalBankWithdrawalFee, c: currency.GBP, amount: 1000, exp: 1},
	} {
		fee, err := wires.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: tc.feeType, FiatCurrency: tc.c, Amount: tc.amount})
		require.NoErrorf(t, err, "GetFeeByType must not error for a %s wire", tc.c)
		assert.Equalf(t, tc.exp, fee, "a %s wire of %v should cost a tenth of a percent, and at least Bitfinex's minimum for dollars and euros", tc.c, tc.amount)
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
	refused := newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`)
	for _, feeType := range []exchange.FeeType{exchange.CryptocurrencyTradeFee, exchange.CryptocurrencyWithdrawalFee} {
		_, err = refused.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: feeType, Pair: currency.NewBTCUSD(), PurchasePrice: 1, Amount: 1})
		var apiErr *APIError
		assert.ErrorAsf(t, err, &apiErr, "GetFeeByType should return the error Bitfinex reports for fee type %v", feeType)
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

func TestTradingFeeRate(t *testing.T) {
	t.Parallel()
	rates := &SummaryFeeRates{
		Maker: SummaryMakerFeeRates{ToCrypto: 0.1, ToStable: 0.2, ToFiat: 0.3, DerivativeRebate: 0.7},
		Taker: SummaryTakerFeeRates{ToCrypto: 0.4, ToStable: 0.5, ToFiat: 0.6, Derivative: 0.8},
	}
	for _, tc := range []struct {
		p     currency.Pair
		maker bool
		exp   float64
	}{
		{p: currency.NewPair(currency.ETH, currency.BTC), maker: true, exp: 0.1},
		{p: currency.NewPair(currency.ETH, currency.UST), maker: true, exp: 0.2},
		{p: currency.NewPair(currency.ETH, currency.USDT), maker: true, exp: 0.2},
		{p: currency.NewPair(currency.ETH, currency.USD), maker: true, exp: 0.3},
		{p: currency.NewPair(currency.ETH, currency.BTC), exp: 0.4},
		{p: currency.NewPair(currency.ETH, currency.UST), exp: 0.5},
		{p: currency.NewPair(currency.ETH, currency.EUR), exp: 0.6},
		{p: testDerivativePair, maker: true, exp: -0.7},
		{p: testCoinMarginedPair, exp: 0.8},
	} {
		assert.Equalf(t, tc.exp, tradingFeeRate(rates, tc.p, tc.maker), "tradingFeeRate should give %s's rate, maker %t", tc.p, tc.maker)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := historyWindow(24 * time.Hour)
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.MarginFunding: testFundingUSD, asset.USDTMarginedFutures: testDerivativePair} {
		item, err := ex.GetHistoricCandles(t.Context(), p, a, kline.OneHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", a)
		require.NotEmptyf(t, item.Candles, "GetHistoricCandles must return %s candles", a)
		assert.Truef(t, item.Candles[0].Time.Before(item.Candles[len(item.Candles)-1].Time), "GetHistoricCandles should return %s candles oldest first", a)
	}
}

func TestHistoryEnd(t *testing.T) {
	t.Parallel()
	past := time.Now().Add(-time.Hour)
	assert.Equal(t, past, historyEnd(past), "historyEnd should keep a window ending in the past")
	assert.WithinDuration(t, time.Now(), historyEnd(time.Now().Add(25*time.Hour)), time.Second, "historyEnd should end a window ending in the future now")
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

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := historyWindow(24 * time.Hour)
	item, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	assert.NotEmpty(t, item.Candles, "GetHistoricCandlesExtended should return candles")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	for a, exp := range map[asset.Item]string{
		asset.Spot:                tradeBaseURL + "/t/BTC:USD?type=exchange",
		asset.Margin:              tradeBaseURL + "/t/BTC:USD?type=margin",
		asset.MarginFunding:       tradeBaseURL + "/f/USD",
		asset.USDTMarginedFutures: tradeBaseURL + "/t/BTCF0:USTF0",
		asset.CoinMarginedFutures: tradeBaseURL + "/t/ETHF0:BTCF0",
	} {
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

func TestCancelBatchOrdersThroughWebsocket(t *testing.T) {
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
	require.NoError(t, err, "CancelBatchOrders must not error through the websocket")
	for _, id := range ids {
		assert.Equalf(t, order.PendingCancel.String(), resp.Status[id], "order %s should be pending cancellation until its close event confirms it", id)
	}
	if mockTests {
		assert.Len(t, venue.inputs(wsOrderCancelMulti), 1, "CancelBatchOrders should cancel the orders in one websocket request")
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
	if !mockTests {
		id, err := parseOrderID(resp.OrderID)
		require.NoError(t, err, "SubmitOrder must return the order's ID")
		cancelLiveTestOrderOnCleanup(t, id)
	}
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: resp.OrderID, AssetType: asset.Spot, Pair: currency.NewBTCUSD()}), "CancelOrder must not error through the websocket")
	if mockTests {
		assert.Len(t, venue.inputs(wsOrderNew), 1, "SubmitOrder should place the order through the websocket")
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

func TestSetDefaultsDerivatives(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		assert.NoErrorf(t, ex.CurrencyPairs.IsAssetEnabled(a), "SetDefaults should enable %s", a)
	}
	f := ex.Features.Supports.FuturesCapabilities
	assert.True(t, f.FundingRates, "funding rates should be supported")
	assert.Equal(t, map[kline.Interval]bool{kline.EightHour: true}, f.SupportedFundingRateFrequencies, "funding should be set every 8 hours")
	assert.Equal(t, map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true}, f.FundingRateBatching, "every pair's funding should come in one request")
	assert.True(t, f.Positions, "positions should be supported")
	assert.True(t, f.Leverage, "leverage should be supported")
	assert.Equal(t, exchange.OpenInterestSupport{Supported: true, SupportsRestBatch: true}, f.OpenInterest, "open interest should come in one request")
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must refuse a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must refuse a futures asset Bitfinex does not trade")
	const refused = `["error",10020,"symbol: invalid"]`
	info := func(pair string) string {
		return `[[["` + pair + `",[1562164542332,null,null,"0.00004","100.0",null,null,null,0.01,0.005]]]]`
	}
	for name, bodies := range map[string]map[string]string{
		"the pair information refused": {"/v2/conf/pub:info:pair:futures": refused},
		"the statuses refused":         {"/v2/conf/pub:info:pair:futures": info("BTCF0:USTF0"), "/v2/status/deriv": refused},
	} {
		_, err = newPathResponseServerExchange(t, bodies).GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
		var apiErr *APIError
		assert.ErrorAsf(t, err, &apiErr, "GetFuturesContractDetails should return the error Bitfinex reports with %s", name)
	}
	_, err = newPathResponseServerExchange(t, map[string]string{"/v2/conf/pub:info:pair:futures": info("BTCF0")}).GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, errSymbolUnparsable, "GetFuturesContractDetails should refuse a contract it cannot read")
	for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: testDerivativePair, asset.CoinMarginedFutures: testCoinMarginedPair} {
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
			assert.WithinDurationf(t, time.Now(), c.LatestRate.Time, derivativeFundingInterval, "%s's latest funding rate should be from its last funding event", p)
			continue
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

func TestGetLatestFundingRates(t *testing.T) {
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
	for a, req := range map[asset.Item]*fundingrate.LatestRateRequest{
		asset.USDTMarginedFutures: {Asset: asset.USDTMarginedFutures, Pair: testDerivativePair},
		// The asset's enabled pairs, without a pair
		asset.CoinMarginedFutures: {Asset: asset.CoinMarginedFutures},
	} {
		rates, err := ex.GetLatestFundingRates(t.Context(), req)
		require.NoErrorf(t, err, "GetLatestFundingRates must not error for %s", a)
		require.Lenf(t, rates, 1, "GetLatestFundingRates must return %s's rate", a)
		assert.WithinDurationf(t, time.Now(), rates[0].TimeChecked, time.Minute, "the %s rate should be checked now", a)
		rates[0].TimeChecked = time.Time{}
		if !mockTests {
			assert.Equalf(t, derivativeFundingInterval, rates[0].TimeOfNextRate.Sub(rates[0].LatestRate.Time), "the %s rate should be set a funding interval before the next", a)
			assert.Truef(t, rates[0].TimeOfNextRate.After(time.Now()), "the %s rate's next funding event should be to come", a)
			continue
		}
		exp := fundingrate.LatestRateResponse{Exchange: ex.Name, Asset: a, Pair: testDerivativePair, TimeOfNextRate: testDerivativeNextFunding}
		exp.LatestRate = fundingrate.Rate{Time: testDerivativeNextFunding.Add(-8 * time.Hour), Rate: decimal.MustFromFloat(0.00003859)}
		if a == asset.CoinMarginedFutures {
			enabled, err := ex.GetEnabledPairs(a)
			require.NoError(t, err, "GetEnabledPairs must not error")
			exp.Pair, exp.LatestRate.Rate = enabled[0], decimal.MustFromFloat(0)
		}
		assert.Equalf(t, exp, rates[0], "GetLatestFundingRates should return the rate Bitfinex set at %s's latest funding event", exp.Pair)
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	start, end := historyWindow(16 * time.Hour)
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")
	for _, tc := range []struct {
		r   fundingrate.HistoricalRatesRequest
		err error
	}{
		{r: fundingrate.HistoricalRatesRequest{Asset: asset.Spot, Pair: currency.NewBTCUSD(), StartDate: start, EndDate: end}, err: futures.ErrNotPerpetualFuture},
		{r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end, IncludePayments: true}, err: common.ErrFunctionNotSupported},
		{r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end, IncludePredictedRate: true}, err: common.ErrFunctionNotSupported},
		{r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: end, EndDate: start}, err: common.ErrStartAfterEnd},
		{r: fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, StartDate: start, EndDate: end}, err: currency.ErrCurrencyPairEmpty},
	} {
		_, err := e.GetHistoricalFundingRates(t.Context(), &tc.r)
		assert.ErrorIsf(t, err, tc.err, "GetHistoricalFundingRates should refuse %+v", tc.r)
	}
	_, err = newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end})
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr, "GetHistoricalFundingRates should return the error Bitfinex reports")
	rates, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: start, EndDate: end})
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
}

func TestGetHistoricalFundingRatesWindowEnd(t *testing.T) {
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
	for _, tc := range []struct {
		start, end time.Time
		exp        func() int64
	}{
		// The snapshots a funding interval after the window carry the rates of its last events
		{start: past, end: past.Add(16 * time.Hour), exp: func() int64 { return past.Add(24 * time.Hour).UnixMilli() }},
		// Bitfinex refuses a window ending after the current UTC day, so one ending now goes no later
		{start: time.Now().Add(-16 * time.Hour), end: time.Now(), exp: func() int64 { return time.Now().UnixMilli() }},
	} {
		ends = nil
		_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, StartDate: tc.start, EndDate: tc.end})
		require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates must find no rates in an empty history")
		require.Len(t, ends, 1, "GetHistoricalFundingRates must request one page of the status history")
		assert.InDeltaf(t, tc.exp(), ends[0], float64(time.Minute.Milliseconds()), "a window ending at %s should request the status history up to the window's end and a funding interval, but not past now", tc.end)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true, asset.Spot: false, asset.Margin: false, asset.Futures: false} {
		got, err := e.IsPerpetualFutureCurrency(a, testDerivativePair)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Equalf(t, exp, got, "IsPerpetualFutureCurrency should report whether %s trades perpetual contracts", a)
	}
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: testDerivativePair, asset.CoinMarginedFutures: currency.NewPairWithDelimiter("ETHF0", "btcf0", "-")} {
		c, ca, err := e.GetCollateralCurrencyForContract(a, p)
		require.NoErrorf(t, err, "GetCollateralCurrencyForContract must not error for %s", a)
		assert.Truef(t, p.Quote.Equal(c), "%s's collateral should be its quote currency", p)
		assert.Equalf(t, a, ca, "%s's collateral should be held for its asset", p)
	}
	_, _, err := e.GetCollateralCurrencyForContract(asset.Margin, currency.NewBTCUSD())
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetCollateralCurrencyForContract should refuse an asset trading no perpetual contracts")
	_, _, err = e.GetCollateralCurrencyForContract(asset.USDTMarginedFutures, testCoinMarginedPair)
	assert.ErrorIs(t, err, errPairNotOfAsset, "GetCollateralCurrencyForContract should refuse a pair the asset does not trade")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.BTC.Item, Quote: currency.USD.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must refuse an asset trading no perpetual contracts")
	keys := []key.PairAsset{
		{Base: testDerivativePair.Base.Item, Quote: testDerivativePair.Quote.Item, Asset: asset.USDTMarginedFutures},
		{Base: testCoinMarginedPair.Base.Item, Quote: testCoinMarginedPair.Quote.Item, Asset: asset.CoinMarginedFutures},
	}
	for _, requested := range [][]key.PairAsset{keys, nil} {
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
			continue
		}
		assert.Equalf(t, []futures.OpenInterest{
			{Key: key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, testDerivativePair), OpenInterest: 9268.71813464},
			{Key: key.NewExchangeAssetPair(e.Name, asset.CoinMarginedFutures, testCoinMarginedPair), OpenInterest: 6061.94360798},
		}, got, "GetOpenInterest should return each contract's open interest for %d keys", len(requested))
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
		StartDate: time.UnixMilli(1792022400000), IsolatedMargin: decimal.MustFromFloat(200), Leverage: decimal.MustFromFloat(80),
		EstimatedLiquidationPrice: decimal.MustFromFloat(79400), CurrentSize: decimal.MustFromFloat(0.2), ContractSettlementType: futures.Linear,
		AverageOpenPrice: decimal.MustFromFloat(80000), UnrealisedPNL: decimal.MustFromFloat(-12.5),
	}, summary, "GetFuturesPositionSummary should summarise the derivative position")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: testCoinMarginedPair})
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetFuturesPositionSummary should report a contract without a position")
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionOrders must reject a nil request")
	now := time.Now().Truncate(time.Millisecond)
	for _, tc := range []struct {
		r   futures.PositionsRequest
		err error
	}{
		{r: futures.PositionsRequest{Asset: asset.Margin, Pairs: currency.Pairs{currency.NewBTCUSD()}, StartDate: now.Add(-time.Hour), EndDate: now}, err: futures.ErrNotPerpetualFuture},
		{r: futures.PositionsRequest{Asset: asset.USDTMarginedFutures, StartDate: now.Add(-time.Hour), EndDate: now}, err: currency.ErrCurrencyPairsEmpty},
		{r: futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now, EndDate: now.Add(-time.Hour)}, err: common.ErrStartAfterEnd},
		{r: futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now.AddDate(0, 0, -20), EndDate: now}, err: futures.ErrOrderHistoryTooLarge},
	} {
		_, err := e.GetFuturesPositionOrders(t.Context(), &tc.r)
		assert.ErrorIsf(t, err, tc.err, "GetFuturesPositionOrders should refuse %+v", tc.r)
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
	_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}, StartDate: now.AddDate(0, 0, -20), EndDate: now, RespectOrderHistoryLimits: true})
	require.NoError(t, err, "GetFuturesPositionOrders must not error for a window starting before the order history")
	require.Len(t, historyBodies, 1, "GetFuturesPositionOrders must read the order history")
	historyStart, ok := historyBodies[0]["start"].(float64)
	require.True(t, ok, "the order history request must have a start")
	assert.InDelta(t, float64(time.Now().Add(-orderHistoryReach).UnixMilli()), historyStart, float64(time.Minute.Milliseconds()), "a window starting before the order history should start as far back as it reaches")
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must reject a nil request")
	for _, tc := range []struct {
		r   margin.PositionChangeRequest
		err error
	}{
		{r: margin.PositionChangeRequest{Asset: asset.Margin, Pair: currency.NewBTCUSD(), MarginType: margin.Isolated, NewAllocatedMargin: 1}, err: futures.ErrNotPerpetualFuture},
		{r: margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Multi, NewAllocatedMargin: 1}, err: margin.ErrMarginTypeUnsupported},
		{r: margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Isolated}, err: margin.ErrNewAllocatedMarginRequired},
		{r: margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testCoinMarginedPair, MarginType: margin.Isolated, NewAllocatedMargin: 1}, err: errPairNotOfAsset},
	} {
		_, err := e.ChangePositionMargin(t.Context(), &tc.r)
		assert.ErrorIsf(t, err, tc.err, "ChangePositionMargin should refuse %+v", tc.r)
	}
	_, err = newResponseServerExchange(t, `[[0]]`).ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.USDTMarginedFutures, Pair: testDerivativePair, MarginType: margin.Isolated, NewAllocatedMargin: 1})
	assert.ErrorIs(t, err, errPositionCollateralNotSet, "ChangePositionMargin should report collateral Bitfinex did not set")
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
