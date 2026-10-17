package bitfinex

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// testInvalidSymbol is a trading pair symbol Bitfinex does not list, whose error response is recorded
const testInvalidSymbol = "tNOPEUSD"

// The windows the history fixtures record, in Unix milliseconds: 2026-10-15 UTC, and for the weekly leaderboards
// 2026-10-01 to 2026-10-16 UTC
const (
	testPublicHistoryStartMilli     = 1792022400000
	testPublicHistoryEndMilli       = 1792108800000
	testPublicLeaderboardStartMilli = 1790812800000
	// testPublicHistoryLimit keeps the history fixtures short
	testPublicHistoryLimit = 3
)

// testPublicHistoryWindow returns the window a history test asks for: the day its fixture records, or, live, the day to now
func testPublicHistoryWindow() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(testPublicHistoryStartMilli), time.UnixMilli(testPublicHistoryEndMilli)
	}
	end = time.Now()
	return end.AddDate(0, 0, -1), end
}

func TestGetPlatformStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetPlatformStatus(t.Context())
	require.NoError(t, err, "GetPlatformStatus must not error")
	require.NotNil(t, result, "GetPlatformStatus must return the status")
	if mockTests {
		assert.Equal(t, &PlatformStatusResponse{Operative: true}, result, "GetPlatformStatus should decode the status")
	}

	_, err = newResponseServerExchange(t, `null`).GetPlatformStatus(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPlatformStatus should error for a null response")
	_, err = newUnavailableServerExchange(t).GetPlatformStatus(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPlatformStatus should return the request failure")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"", "t", "fUSD", "BTCUSD"} {
		_, err := e.GetTicker(t.Context(), symbol)
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetTicker should refuse symbol %q", symbol)
	}

	result, err := e.GetTicker(t.Context(), "tBTCUSD")
	require.NoError(t, err, "GetTicker must not error")
	if mockTests {
		exp := &Ticker{
			Bid:                 82402,
			BidSize:             1.65150279,
			Ask:                 82423,
			AskSize:             2.78348726,
			DailyChange:         1100,
			DailyChangeRelative: 0.01352614,
			LastPrice:           82424,
			Volume:              540.71148817,
			High:                82641,
			Low:                 79821,
			FirstTradeTime:      types.Time(time.UnixMilli(1358182043000)),
		}
		assert.Equal(t, exp, result, "GetTicker should decode every field")
	} else {
		assert.Positive(t, result.LastPrice, "GetTicker should return the last price")
	}

	_, err = e.GetTicker(t.Context(), testInvalidSymbol)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "GetTicker must return the error Bitfinex reports for an unlisted symbol")
	assert.Equal(t, &APIError{Code: 10020, Message: "symbol: invalid"}, apiErr, "GetTicker should decode the reported error")
	_, err = newResponseServerExchange(t, `null`).GetTicker(t.Context(), "tBTCUSD")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTicker should error for a null response")
	_, err = newUnavailableServerExchange(t).GetTicker(t.Context(), "tBTCUSD")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTicker should return the request failure")
}

func TestGetFundingTicker(t *testing.T) {
	t.Parallel()
	for _, symbol := range []string{"", "f", "tBTCUSD", "USD"} {
		_, err := e.GetFundingTicker(t.Context(), symbol)
		assert.ErrorIsf(t, err, errFundingSymbolRequired, "GetFundingTicker should refuse symbol %q", symbol)
	}

	result, err := e.GetFundingTicker(t.Context(), "fUSD")
	require.NoError(t, err, "GetFundingTicker must not error")
	if mockTests {
		exp := &FundingTicker{
			FlashReturnRate:                0.00030070958904109587,
			Bid:                            0.0001712,
			BidPeriod:                      120,
			BidSize:                        18938194.36843404,
			Ask:                            0.0001258336,
			AskPeriod:                      2,
			AskSize:                        40993.865045,
			DailyChange:                    -0.000075,
			DailyChangeRelative:            -0.375,
			LastRate:                       0.000125,
			Volume:                         95383739.75889896,
			High:                           0.0002,
			Low:                            0.000001,
			FlashReturnRateAmountAvailable: 220861838.82944623,
			FirstTradeTime:                 types.Time(time.UnixMilli(1469734163000)),
		}
		assert.Equal(t, exp, result, "GetFundingTicker should decode every field")
	} else {
		assert.Positive(t, result.FlashReturnRate, "GetFundingTicker should return the flash return rate")
	}

	_, err = newResponseServerExchange(t, `null`).GetFundingTicker(t.Context(), "fUSD")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFundingTicker should error for a null response")
	_, err = newUnavailableServerExchange(t).GetFundingTicker(t.Context(), "fUSD")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingTicker should return the request failure")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetTickers(t.Context(), []string{"tBTCUSD", "tTESTBTC:TESTUSD", "fUSD", "fBTC"})
	require.NoError(t, err, "GetTickers must not error")
	if mockTests {
		exp := &TickersResponse{
			Trading: []TickerEntry{
				{Symbol: "tBTCUSD", Ticker: Ticker{Bid: 82402, BidSize: 1.65150279, Ask: 82423, AskSize: 2.78348726, DailyChange: 1100, DailyChangeRelative: 0.01352614, LastPrice: 82424, Volume: 540.71148817, High: 82641, Low: 79821, FirstTradeTime: types.Time(time.UnixMilli(1358182043000))}},
				{Symbol: "tTESTBTC:TESTUSD", Ticker: Ticker{Bid: 82345, BidSize: 39.43474446, Ask: 82427, AskSize: 93.60610432, DailyChange: 1065, DailyChangeRelative: 0.01309625, LastPrice: 82386, Volume: 146.73022923, High: 82644, Low: 79845, FirstTradeTime: types.Time(time.UnixMilli(1592991523941))}},
			},
			Funding: []FundingTickerEntry{
				{Symbol: "fUSD", Ticker: FundingTicker{FlashReturnRate: 0.00030070958904109587, Bid: 0.0001712, BidPeriod: 120, BidSize: 18938194.36843404, Ask: 0.0001258336, AskPeriod: 2, AskSize: 40993.865045, DailyChange: -0.000075, DailyChangeRelative: -0.375, LastRate: 0.000125, Volume: 95383739.75889896, High: 0.0002, Low: 0.000001, FlashReturnRateAmountAvailable: 220861838.82944623, FirstTradeTime: types.Time(time.UnixMilli(1469734163000))}},
				{Symbol: "fBTC", Ticker: FundingTicker{FlashReturnRate: 2.1917808219178083e-08, Bid: 0.002, BidPeriod: 2, BidSize: 0.202, Ask: 1e-08, AskPeriod: 2, AskSize: 842.19826519, LastRate: 2e-08, Volume: 101.13070526, High: 2e-08, Low: 1e-08, FlashReturnRateAmountAvailable: 841.40737511, FirstTradeTime: types.Time(time.UnixMilli(1469734183000))}},
			},
		}
		assert.Equal(t, exp, result, "GetTickers should decode every ticker")
	} else {
		assert.Len(t, result.Trading, 2, "GetTickers should return both trading pairs' tickers")
		assert.Len(t, result.Funding, 2, "GetTickers should return both funding currencies' tickers")
	}

	_, err = newResponseServerExchange(t, `null`).GetTickers(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTickers should error for a null response")
	_, err = newUnavailableServerExchange(t).GetTickers(t.Context(), nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTickers should return the request failure")
}

func TestTickersResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp TickersResponse
	require.NoError(t, resp.UnmarshalJSON([]byte(`[]`)), "UnmarshalJSON must not error for no tickers")
	assert.Empty(t, resp, "UnmarshalJSON should decode no tickers from an empty list")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`[["xBTCUSD",1]]`)), errUnknownSymbolType, "UnmarshalJSON should refuse a symbol of neither kind")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`[[1,2]]`)), errTupleElementType, "UnmarshalJSON should refuse a ticker without a symbol")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`[["tBTCUSD","x"]]`)), errTupleElementType, "UnmarshalJSON should refuse a malformed trading pair ticker")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`[["fUSD","x"]]`)), errTupleElementType, "UnmarshalJSON should refuse a malformed funding currency ticker")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`{}`)), errTupleNotArray, "UnmarshalJSON should refuse an object")
}

func TestGetTickersHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickersHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTickersHistory must reject a nil request")
	_, err = e.GetTickersHistory(t.Context(), &TickersHistoryRequest{Symbols: []string{"tBTCUSD", "fUSD"}})
	require.ErrorIs(t, err, errTradingSymbolRequired, "GetTickersHistory must refuse a funding currency")
	_, err = e.GetTickersHistory(t.Context(), &TickersHistoryRequest{Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTickersHistory must refuse a reversed window")

	start, end := testPublicHistoryWindow()
	result, err := e.GetTickersHistory(t.Context(), &TickersHistoryRequest{Symbols: []string{"tBTCUSD", "tETHUSD"}, Start: start, End: end, Limit: testPublicHistoryLimit, Sort: 1})
	require.NoError(t, err, "GetTickersHistory must not error")
	if mockTests {
		exp := []HistoricTicker{
			{Symbol: "tETHUSD", Bid: 2423.1, Ask: 2423.5, Time: types.Time(time.UnixMilli(1792022415000))},
			{Symbol: "tBTCUSD", Bid: 81700, Ask: 81715, Time: types.Time(time.UnixMilli(1792022415000))},
			{Symbol: "tETHUSD", Bid: 2420.1, Ask: 2420.2, Time: types.Time(time.UnixMilli(1792026015000))},
		}
		assert.Equal(t, exp, result, "GetTickersHistory should decode every ticker")
	} else {
		assert.NotEmpty(t, result, "GetTickersHistory should return the pairs' history")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "ALL", r.URL.Query().Get("symbols"), "GetTickersHistory should ask for every pair when given none")
		_, err := w.Write([]byte(`[]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetTickersHistory(t.Context(), &TickersHistoryRequest{})
	assert.NoError(t, err, "GetTickersHistory should not error for every pair")
	_, err = newUnavailableServerExchange(t).GetTickersHistory(t.Context(), &TickersHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTickersHistory should return the request failure")
}

func TestGetPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicTrades must reject a nil request")
	for _, symbol := range []string{"", "t", "fUSD", "BTCUSD"} {
		_, err = e.GetPublicTrades(t.Context(), &PublicTradesRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "GetPublicTrades must refuse symbol %q", symbol)
	}
	_, err = e.GetPublicTrades(t.Context(), &PublicTradesRequest{Symbol: "tBTCUSD", Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPublicTrades must refuse a reversed window")

	start, end := testPublicHistoryWindow()
	result, err := e.GetPublicTrades(t.Context(), &PublicTradesRequest{Symbol: "tBTCUSD", Start: start, End: end, Limit: testPublicHistoryLimit, Sort: 1})
	require.NoError(t, err, "GetPublicTrades must not error")
	if mockTests {
		exp := []PublicTrade{
			{ID: 1993816156, Time: types.Time(time.UnixMilli(1792022401421)), Amount: -0.00020198, Price: 81678},
			{ID: 1993816158, Time: types.Time(time.UnixMilli(1792022402063)), Amount: 0.00000329, Price: 81688},
			{ID: 1993816159, Time: types.Time(time.UnixMilli(1792022402064)), Amount: 0.00003671, Price: 81688},
		}
		assert.Equal(t, exp, result, "GetPublicTrades should decode every trade")
	} else {
		assert.NotEmpty(t, result, "GetPublicTrades should return the pair's trades")
	}

	result, err = e.GetPublicTrades(t.Context(), &PublicTradesRequest{Symbol: testInvalidSymbol})
	require.NoError(t, err, "GetPublicTrades must not error for a pair Bitfinex does not list")
	assert.Empty(t, result, "GetPublicTrades should return no trades for a pair Bitfinex does not list")
	_, err = newUnavailableServerExchange(t).GetPublicTrades(t.Context(), &PublicTradesRequest{Symbol: "tBTCUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPublicTrades should return the request failure")
}

func TestGetPublicFundingTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicFundingTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicFundingTrades must reject a nil request")
	for _, symbol := range []string{"", "f", "tBTCUSD", "USD"} {
		_, err = e.GetPublicFundingTrades(t.Context(), &PublicTradesRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errFundingSymbolRequired, "GetPublicFundingTrades must refuse symbol %q", symbol)
	}
	_, err = e.GetPublicFundingTrades(t.Context(), &PublicTradesRequest{Symbol: "fUSD", Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPublicFundingTrades must refuse a reversed window")

	start, end := testPublicHistoryWindow()
	result, err := e.GetPublicFundingTrades(t.Context(), &PublicTradesRequest{Symbol: "fUSD", Start: start, End: end, Limit: testPublicHistoryLimit, Sort: 1})
	require.NoError(t, err, "GetPublicFundingTrades must not error")
	if mockTests {
		exp := []PublicFundingTrade{
			{ID: 434057819, Time: types.Time(time.UnixMilli(1792022451000)), Amount: 773.566735, Rate: 0.00026938, Period: 120},
			{ID: 434058034, Time: types.Time(time.UnixMilli(1792023032000)), Amount: -599.18397559, Rate: 0.00016127, Period: 2},
			{ID: 434058030, Time: types.Time(time.UnixMilli(1792023032000)), Amount: -500, Rate: 0.00016127, Period: 2},
		}
		assert.Equal(t, exp, result, "GetPublicFundingTrades should decode every trade")
	} else {
		assert.NotEmpty(t, result, "GetPublicFundingTrades should return the currency's trades")
	}

	_, err = newUnavailableServerExchange(t).GetPublicFundingTrades(t.Context(), &PublicTradesRequest{Symbol: "fUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPublicFundingTrades should return the request failure")
}

func TestGetBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBook must reject a nil request")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.GetBook(t.Context(), &BookRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "GetBook must refuse symbol %q", symbol)
	}
	_, err = e.GetBook(t.Context(), &BookRequest{Symbol: "tBTCUSD", Precision: 5})
	require.ErrorIs(t, err, errInvalidBookPrecision, "GetBook must refuse a precision beyond P4")
	_, err = e.GetBook(t.Context(), &BookRequest{Symbol: "tBTCUSD", Length: 50})
	require.ErrorIs(t, err, errInvalidBookLength, "GetBook must refuse a length Bitfinex does not offer")

	result, err := e.GetBook(t.Context(), &BookRequest{Symbol: "tBTCUSD", Length: 1})
	require.NoError(t, err, "GetBook must not error")
	if mockTests {
		exp := []BookLevel{{Price: 82159, Count: 2, Amount: 0.00216011}, {Price: 82171, Count: 1, Amount: -0.00022}}
		assert.Equal(t, exp, result, "GetBook should decode every level")
	} else {
		assert.Len(t, result, 2, "GetBook should return a level a side")
	}

	_, err = e.GetBook(t.Context(), &BookRequest{Symbol: testInvalidSymbol, Length: 1})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "GetBook must return the error Bitfinex reports for an unlisted symbol")
	assert.Equal(t, &APIError{Code: 10020, Message: "symbol: invalid"}, apiErr, "GetBook should decode the reported error")
	_, err = newUnavailableServerExchange(t).GetBook(t.Context(), &BookRequest{Symbol: "tBTCUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBook should return the request failure")
}

func TestGetFundingBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingBook must reject a nil request")
	for _, symbol := range []string{"", "tBTCUSD"} {
		_, err = e.GetFundingBook(t.Context(), &BookRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errFundingSymbolRequired, "GetFundingBook must refuse symbol %q", symbol)
	}
	_, err = e.GetFundingBook(t.Context(), &BookRequest{Symbol: "fUSD", Precision: 5})
	require.ErrorIs(t, err, errInvalidBookPrecision, "GetFundingBook must refuse a precision beyond P4")
	_, err = e.GetFundingBook(t.Context(), &BookRequest{Symbol: "fUSD", Length: 50})
	require.ErrorIs(t, err, errInvalidBookLength, "GetFundingBook must refuse a length Bitfinex does not offer")

	result, err := e.GetFundingBook(t.Context(), &BookRequest{Symbol: "fUSD", Precision: 1, Length: 1})
	require.NoError(t, err, "GetFundingBook must not error")
	if mockTests {
		exp := []FundingBookLevel{{Rate: 0.0001712, Period: 120, Count: 1, Amount: -20000}, {Rate: 0.0001252, Period: 2, Count: 1, Amount: 500}}
		assert.Equal(t, exp, result, "GetFundingBook should decode every level")
	} else {
		assert.Len(t, result, 2, "GetFundingBook should return a level a side")
	}

	_, err = newUnavailableServerExchange(t).GetFundingBook(t.Context(), &BookRequest{Symbol: "fUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingBook should return the request failure")
}

func TestGetRawBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetRawBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRawBook must reject a nil request")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.GetRawBook(t.Context(), &RawBookRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "GetRawBook must refuse symbol %q", symbol)
	}
	_, err = e.GetRawBook(t.Context(), &RawBookRequest{Symbol: "tBTCUSD", Length: 50})
	require.ErrorIs(t, err, errInvalidBookLength, "GetRawBook must refuse a length Bitfinex does not offer")

	result, err := e.GetRawBook(t.Context(), &RawBookRequest{Symbol: "tBTCUSD", Length: 1})
	require.NoError(t, err, "GetRawBook must not error")
	if mockTests {
		exp := []RawBookLevel{{OrderID: 246603709470, Price: 82149, Amount: 0.00009406}, {OrderID: 246646973663, Price: 82156, Amount: -0.06086039}}
		assert.Equal(t, exp, result, "GetRawBook should decode every order")
	} else {
		assert.Len(t, result, 2, "GetRawBook should return an order a side")
	}

	_, err = newUnavailableServerExchange(t).GetRawBook(t.Context(), &RawBookRequest{Symbol: "tBTCUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetRawBook should return the request failure")
}

func TestGetRawFundingBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetRawFundingBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRawFundingBook must reject a nil request")
	for _, symbol := range []string{"", "tBTCUSD"} {
		_, err = e.GetRawFundingBook(t.Context(), &RawBookRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errFundingSymbolRequired, "GetRawFundingBook must refuse symbol %q", symbol)
	}
	_, err = e.GetRawFundingBook(t.Context(), &RawBookRequest{Symbol: "fUSD", Length: 50})
	require.ErrorIs(t, err, errInvalidBookLength, "GetRawFundingBook must refuse a length Bitfinex does not offer")

	result, err := e.GetRawFundingBook(t.Context(), &RawBookRequest{Symbol: "fUSD", Length: 1})
	require.NoError(t, err, "GetRawFundingBook must not error")
	if mockTests {
		exp := []RawFundingBookLevel{{OfferID: 5038297953, Period: 120, Rate: 0.0001712, Amount: -20000}, {OfferID: 5149012082, Period: 2, Rate: 0.000125120548, Amount: 500}}
		assert.Equal(t, exp, result, "GetRawFundingBook should decode every offer")
	} else {
		assert.Len(t, result, 2, "GetRawFundingBook should return an offer a side")
	}

	_, err = newUnavailableServerExchange(t).GetRawFundingBook(t.Context(), &RawBookRequest{Symbol: "fUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetRawFundingBook should return the request failure")
}

func TestBookPrecision(t *testing.T) {
	t.Parallel()
	for precision, exp := range map[uint64]string{0: "P0", 1: "P1", 2: "P2", 3: "P3", 4: "P4"} {
		got, err := bookPrecision(precision)
		require.NoErrorf(t, err, "bookPrecision must not error for %d", precision)
		assert.Equalf(t, exp, got, "bookPrecision should format %d", precision)
	}
	_, err := bookPrecision(5)
	assert.ErrorIs(t, err, errInvalidBookPrecision, "bookPrecision should refuse a precision beyond P4")
}

func TestBookParams(t *testing.T) {
	t.Parallel()
	params, err := bookParams(0)
	require.NoError(t, err, "bookParams must not error for an unset length")
	assert.Empty(t, params, "bookParams should leave an unset length to Bitfinex's default")
	for _, length := range []uint64{1, 25, 100, 250} {
		params, err = bookParams(length)
		require.NoErrorf(t, err, "bookParams must not error for length %d", length)
		assert.Equalf(t, url.Values{"len": {strconv.FormatUint(length, 10)}}, params, "bookParams should send length %d", length)
	}
	for _, length := range []uint64{2, 50, 251} {
		_, err = bookParams(length)
		assert.ErrorIsf(t, err, errInvalidBookLength, "bookParams should refuse length %d", length)
	}
}

func TestGetLastStat(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastStat(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastStat must reject a nil request")
	_, err = e.GetLastStat(t.Context(), &StatsRequest{Key: "nope", Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errInvalidStatsKey, "GetLastStat must refuse an unknown key")

	result, err := e.GetLastStat(t.Context(), &StatsRequest{Key: StatsKeyPositionSize, Symbol: "tBTCUSD", Side: StatsSideLong})
	require.NoError(t, err, "GetLastStat must not error")
	if mockTests {
		exp := &Stat{Time: types.Time(time.UnixMilli(1792129680000)), Value: 89621.97322647}
		assert.Equal(t, exp, result, "GetLastStat should decode the statistic")
	} else {
		assert.Positive(t, result.Value, "GetLastStat should return the size of the pair's long positions")
	}

	_, err = e.GetLastStat(t.Context(), &StatsRequest{Key: StatsKeyPositionSize, Symbol: testInvalidSymbol, Side: StatsSideLong})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastStat should error for the null Bitfinex sends for an unlisted pair")
	_, err = newUnavailableServerExchange(t).GetLastStat(t.Context(), &StatsRequest{Key: StatsKeyPositionSize, Symbol: "tBTCUSD", Side: StatsSideLong})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastStat should return the request failure")
}

func TestGetStatsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetStatsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStatsHistory must reject a nil request")
	_, err = e.GetStatsHistory(t.Context(), &StatsHistoryRequest{Key: "nope", Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errInvalidStatsKey, "GetStatsHistory must refuse an unknown key")
	_, err = e.GetStatsHistory(t.Context(), &StatsHistoryRequest{Key: StatsKeyFundingSize, Symbol: "fUSD", Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStatsHistory must refuse a reversed window")

	start, end := testPublicHistoryWindow()
	result, err := e.GetStatsHistory(t.Context(), &StatsHistoryRequest{Key: StatsKeyCreditsSizeByPair, Symbol: "fUSD", Pair: "tBTCUSD", Start: start, End: end, Limit: testPublicHistoryLimit, Sort: 1})
	require.NoError(t, err, "GetStatsHistory must not error")
	if mockTests {
		exp := []Stat{
			{Time: types.Time(time.UnixMilli(1792022400000)), Value: 5600850891.1279125},
			{Time: types.Time(time.UnixMilli(1792022460000)), Value: 5600850891.1279125},
			{Time: types.Time(time.UnixMilli(1792022520000)), Value: 5600850891.1279125},
		}
		assert.Equal(t, exp, result, "GetStatsHistory should decode every value")
	} else {
		assert.NotEmpty(t, result, "GetStatsHistory should return the statistic's values")
	}

	_, err = newUnavailableServerExchange(t).GetStatsHistory(t.Context(), &StatsHistoryRequest{Key: StatsKeyFundingSize, Symbol: "fUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStatsHistory should return the request failure")
}

func TestStatsKey(t *testing.T) {
	t.Parallel()
	for exp, req := range map[string]*StatsRequest{
		"pos.size:1m:tBTCUSD:long":         {Key: StatsKeyPositionSize, Symbol: "tBTCUSD", Side: StatsSideLong},
		"pos.size:1m:tBTCUSD:short":        {Key: StatsKeyPositionSize, Symbol: "tBTCUSD", Side: StatsSideShort},
		"funding.size:1m:fUSD":             {Key: StatsKeyFundingSize, Symbol: "fUSD"},
		"credits.size:1m:fUSD":             {Key: StatsKeyCreditsSize, Symbol: "fUSD"},
		"credits.size.sym:1m:fUSD:tBTCUSD": {Key: StatsKeyCreditsSizeByPair, Symbol: "fUSD", Pair: "tBTCUSD"},
		"vol.1d:30m:BFX":                   {Key: StatsKeyVolume1Day, Symbol: StatsPlatform},
		"vol.7d:30m:BFX":                   {Key: StatsKeyVolume7Day, Symbol: StatsPlatform},
		"vol.30d:30m:BFX":                  {Key: StatsKeyVolume30Day, Symbol: StatsPlatform},
		"vwap:1d:tBTCUSD":                  {Key: StatsKeyVolumeWeightedAveragePrice, Symbol: "tBTCUSD"},
	} {
		got, err := statsKey(req)
		require.NoErrorf(t, err, "statsKey must not error for %s", exp)
		assert.Equalf(t, exp, got, "statsKey should build %s", exp)
	}
	for _, tc := range []struct {
		req *StatsRequest
		err error
	}{
		{req: &StatsRequest{Symbol: "tBTCUSD"}, err: errInvalidStatsKey},
		{req: &StatsRequest{Key: "pos.size:1m", Symbol: "tBTCUSD", Side: StatsSideLong}, err: errInvalidStatsKey},
		{req: &StatsRequest{Key: StatsKeyPositionSize, Symbol: "fUSD", Side: StatsSideLong}, err: errTradingSymbolRequired},
		{req: &StatsRequest{Key: StatsKeyVolumeWeightedAveragePrice}, err: errTradingSymbolRequired},
		{req: &StatsRequest{Key: StatsKeyFundingSize, Symbol: "tBTCUSD"}, err: errFundingSymbolRequired},
		{req: &StatsRequest{Key: StatsKeyCreditsSize}, err: errFundingSymbolRequired},
		{req: &StatsRequest{Key: StatsKeyCreditsSizeByPair, Symbol: "tBTCUSD", Pair: "tBTCUSD"}, err: errFundingSymbolRequired},
		{req: &StatsRequest{Key: StatsKeyVolume1Day}, err: errInvalidStatsSymbol},
		{req: &StatsRequest{Key: StatsKeyVolume30Day, Symbol: "tBTCUSD"}, err: errInvalidStatsSymbol},
		{req: &StatsRequest{Key: StatsKeyPositionSize, Symbol: "tBTCUSD"}, err: errInvalidStatsSide},
		{req: &StatsRequest{Key: StatsKeyPositionSize, Symbol: "tBTCUSD", Side: "LONG"}, err: errInvalidStatsSide},
		{req: &StatsRequest{Key: StatsKeyFundingSize, Symbol: "fUSD", Side: StatsSideLong}, err: errInvalidStatsSide},
		{req: &StatsRequest{Key: StatsKeyCreditsSizeByPair, Symbol: "fUSD"}, err: errTradingSymbolRequired},
		{req: &StatsRequest{Key: StatsKeyCreditsSizeByPair, Symbol: "fUSD", Pair: "fBTC"}, err: errTradingSymbolRequired},
		{req: &StatsRequest{Key: StatsKeyCreditsSize, Symbol: "fUSD", Pair: "tBTCUSD"}, err: errInvalidStatsPair},
		{req: &StatsRequest{Key: StatsKeyPositionSize, Symbol: "tBTCUSD", Side: StatsSideLong, Pair: "tBTCUSD"}, err: errInvalidStatsPair},
	} {
		_, err := statsKey(tc.req)
		assert.ErrorIsf(t, err, tc.err, "statsKey should refuse %+v", tc.req)
	}
}

func TestGetLastCandle(t *testing.T) {
	t.Parallel()
	_, err := e.GetLastCandle(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLastCandle must reject a nil request")
	_, err = e.GetLastCandle(t.Context(), &CandlesRequest{Symbol: "tBTCUSD", Interval: kline.TwoHour})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetLastCandle must refuse an interval Bitfinex does not offer")

	result, err := e.GetLastCandle(t.Context(), &CandlesRequest{Symbol: "tBTCUSD", Interval: kline.OneDay})
	require.NoError(t, err, "GetLastCandle must not error")
	if mockTests {
		exp := &Candle{Time: types.Time(time.UnixMilli(1792108800000)), Open: 82233, Close: 82179, High: 82547, Low: 81692, Volume: 33.39572295}
		assert.Equal(t, exp, result, "GetLastCandle should decode the candle")
	} else {
		assert.Positive(t, result.Close, "GetLastCandle should return the pair's latest candle")
	}

	result, err = e.GetLastCandle(t.Context(), &CandlesRequest{Symbol: "fUSD", Interval: kline.OneDay, Period: 30})
	require.NoError(t, err, "GetLastCandle must not error for a funding currency")
	if mockTests {
		exp := &Candle{Time: types.Time(time.UnixMilli(1792108800000)), Open: 0.00016, Close: 0.000132, High: 0.00016, Low: 0.00013, Volume: 19052.79981827}
		assert.Equal(t, exp, result, "GetLastCandle should decode a funding currency's candle")
	} else {
		assert.Positive(t, result.Close, "GetLastCandle should return the currency's latest candle")
	}

	_, err = e.GetLastCandle(t.Context(), &CandlesRequest{Symbol: testInvalidSymbol, Interval: kline.OneDay})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetLastCandle should error for the null Bitfinex sends for an unlisted pair")
	_, err = newUnavailableServerExchange(t).GetLastCandle(t.Context(), &CandlesRequest{Symbol: "tBTCUSD", Interval: kline.OneDay})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLastCandle should return the request failure")
}

func TestGetCandlesHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCandlesHistory must reject a nil request")
	_, err = e.GetCandlesHistory(t.Context(), &CandlesHistoryRequest{Symbol: "fUSD", Interval: kline.OneHour})
	require.ErrorIs(t, err, errCandlesPeriodRequired, "GetCandlesHistory must refuse a funding currency without a period")
	_, err = e.GetCandlesHistory(t.Context(), &CandlesHistoryRequest{Symbol: "tBTCUSD", Interval: kline.OneHour, Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCandlesHistory must refuse a reversed window")

	start, end := testPublicHistoryWindow()
	result, err := e.GetCandlesHistory(t.Context(), &CandlesHistoryRequest{Symbol: "tBTCUSD", Interval: kline.OneHour, Start: start, End: end, Limit: testPublicHistoryLimit, Sort: 1})
	require.NoError(t, err, "GetCandlesHistory must not error")
	if mockTests {
		exp := []Candle{
			{Time: types.Time(time.UnixMilli(1792022400000)), Open: 81678, Close: 81659, High: 81845, Low: 81368, Volume: 7.16225303},
			{Time: types.Time(time.UnixMilli(1792026000000)), Open: 81675, Close: 81663, High: 81930, Low: 81504, Volume: 6.00131924},
			{Time: types.Time(time.UnixMilli(1792029600000)), Open: 81640, Close: 81557, High: 81808, Low: 80210, Volume: 53.38772928},
		}
		assert.Equal(t, exp, result, "GetCandlesHistory should decode every candle")
	} else {
		assert.NotEmpty(t, result, "GetCandlesHistory should return the pair's candles")
	}

	result, err = e.GetCandlesHistory(t.Context(), &CandlesHistoryRequest{Symbol: "fUSD", Interval: kline.OneHour, Aggregation: 30, PeriodStart: 2, PeriodEnd: 30, Start: start, End: end, Limit: testPublicHistoryLimit, Sort: 1})
	require.NoError(t, err, "GetCandlesHistory must not error for aggregated funding candles")
	if mockTests {
		exp := []Candle{
			{Time: types.Time(time.UnixMilli(1792022400000)), Open: 0.000133, Close: 0.00014384, High: 0.00017987, Low: 0.00012051, Volume: 3074226.6222474203},
			{Time: types.Time(time.UnixMilli(1792026000000)), Open: 0.00013346, Close: 0.00014, High: 0.00017087, Low: 0.00012281, Volume: 880889.47972334},
			{Time: types.Time(time.UnixMilli(1792029600000)), Open: 0.000133, Close: 0.00014, High: 0.0001784, Low: 0.0001244, Volume: 4190168.7710503098},
		}
		assert.Equal(t, exp, result, "GetCandlesHistory should decode every aggregated funding candle")
	} else {
		assert.NotEmpty(t, result, "GetCandlesHistory should return the currency's aggregated candles")
	}

	_, err = newUnavailableServerExchange(t).GetCandlesHistory(t.Context(), &CandlesHistoryRequest{Symbol: "tBTCUSD", Interval: kline.OneHour})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCandlesHistory should return the request failure")
}

func TestCandleKey(t *testing.T) {
	t.Parallel()
	for interval, timeFrame := range map[kline.Interval]string{
		kline.OneMin: "1m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m", kline.OneHour: "1h", kline.ThreeHour: "3h",
		kline.SixHour: "6h", kline.TwelveHour: "12h", kline.OneDay: "1D", kline.OneWeek: "1W", kline.TwoWeek: "14D", kline.OneMonth: "1M",
	} {
		got, err := candleKey(&CandlesRequest{Symbol: "tBTCUSD", Interval: interval})
		require.NoErrorf(t, err, "candleKey must not error for interval %s", interval)
		assert.Equalf(t, "trade:"+timeFrame+":tBTCUSD", got, "candleKey should name interval %s %s", interval, timeFrame)
	}
	for exp, req := range map[string]*CandlesRequest{
		"trade:1D:fUSD:p30":        {Symbol: "fUSD", Interval: kline.OneDay, Period: 30},
		"trade:1m:fUSD:a10:p2:p10": {Symbol: "fUSD", Interval: kline.OneMin, Aggregation: 10, PeriodStart: 2, PeriodEnd: 10},
		"trade:1h:fUSD:a30:p2:p30": {Symbol: "fUSD", Interval: kline.OneHour, Aggregation: 30, PeriodStart: 2, PeriodEnd: 30},
	} {
		got, err := candleKey(req)
		require.NoErrorf(t, err, "candleKey must not error for %s", exp)
		assert.Equalf(t, exp, got, "candleKey should build %s", exp)
	}
	for _, tc := range []struct {
		req *CandlesRequest
		err error
	}{
		{req: &CandlesRequest{Symbol: "tBTCUSD"}, err: kline.ErrUnsupportedInterval},
		{req: &CandlesRequest{Symbol: "tBTCUSD", Interval: kline.TwoHour}, err: kline.ErrUnsupportedInterval},
		{req: &CandlesRequest{Interval: kline.OneMin}, err: errSymbolRequired},
		{req: &CandlesRequest{Symbol: "BTCUSD", Interval: kline.OneMin}, err: errSymbolRequired},
		{req: &CandlesRequest{Symbol: "tBTCUSD", Interval: kline.OneMin, Period: 30}, err: errCandlesPeriodUnsupported},
		{req: &CandlesRequest{Symbol: "tBTCUSD", Interval: kline.OneMin, Aggregation: 30, PeriodStart: 2, PeriodEnd: 30}, err: errCandlesPeriodUnsupported},
		{req: &CandlesRequest{Symbol: "tBTCUSD", Interval: kline.OneMin, PeriodEnd: 30}, err: errCandlesPeriodUnsupported},
		{req: &CandlesRequest{Symbol: "fUSD", Interval: kline.OneMin}, err: errCandlesPeriodRequired},
		{req: &CandlesRequest{Symbol: "fUSD", Interval: kline.OneMin, PeriodStart: 2, PeriodEnd: 30}, err: errInvalidCandlesAggregation},
		{req: &CandlesRequest{Symbol: "fUSD", Interval: kline.OneMin, Aggregation: 20, PeriodStart: 2, PeriodEnd: 30}, err: errInvalidCandlesAggregation},
		{req: &CandlesRequest{Symbol: "fUSD", Interval: kline.OneMin, Aggregation: 30, Period: 30, PeriodStart: 2, PeriodEnd: 30}, err: errCandlesPeriodUnsupported},
		{req: &CandlesRequest{Symbol: "fUSD", Interval: kline.OneMin, Aggregation: 30, PeriodStart: 2}, err: errCandlesPeriodRequired},
		{req: &CandlesRequest{Symbol: "fUSD", Interval: kline.OneMin, Aggregation: 30, PeriodEnd: 30}, err: errCandlesPeriodRequired},
	} {
		_, err := candleKey(tc.req)
		assert.ErrorIsf(t, err, tc.err, "candleKey should refuse %+v", tc.req)
	}
}

func TestGetDerivativesStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetDerivativesStatus(t.Context(), []string{"tBTCF0:USTF0", "tETHF0:USTF0"})
	require.NoError(t, err, "GetDerivativesStatus must not error")
	if mockTests {
		exp := []DerivativeStatusEntry{
			{Key: "tBTCF0:USTF0", Status: DerivativeStatus{
				Time: types.Time(time.UnixMilli(1792129748000)), DerivativePrice: 82189.30837992, SpotPrice: 82135, InsuranceFundBalance: 61340784.36117272,
				NextFundingTime: types.Time(time.UnixMilli(1792137600000)), NextFundingAccrued: 0.00042072, NextFundingStep: 6688, CurrentFunding: 0.00010884,
				MarkPrice: 82139.1843, OpenInterest: 9254.32427008, ClampMinimum: 0.0005, ClampMaximum: 0.0025,
			}},
			{Key: "tETHF0:USTF0", Status: DerivativeStatus{
				Time: types.Time(time.UnixMilli(1792129748000)), DerivativePrice: 2448.245538685, SpotPrice: 2446.8, InsuranceFundBalance: 61340784.36117272,
				NextFundingTime: types.Time(time.UnixMilli(1792137600000)), NextFundingAccrued: 0.00036349, NextFundingStep: 6688, CurrentFunding: 0.00004981,
				MarkPrice: 2447.0178, OpenInterest: 10858.12719944, ClampMinimum: 0.0005, ClampMaximum: 0.0025,
			}},
		}
		assert.Equal(t, exp, result, "GetDerivativesStatus should decode every pair's status")
	} else {
		assert.Len(t, result, 2, "GetDerivativesStatus should return both pairs' status")
	}

	// The recorded response to keys=ALL is trimmed to its first pairs and the derivative pairs the wrapper tests use, so
	// only those pairs' keys are compared
	result, err = e.GetDerivativesStatus(t.Context(), nil)
	require.NoError(t, err, "GetDerivativesStatus must not error for every pair")
	keys := make([]string, len(result))
	for i := range result {
		keys[i] = result[i].Key
	}
	if mockTests {
		exp := []string{"tTESTXAUTF0:TESTUSDTF0", "tTESTETHF0:TESTUSDTF0", "tTESTAVAXF0:TESTUSDTF0", "tTESTDOGEF0:TESTUSDTF0", "tTESTALGOF0:TESTUSDTF0", "tBTCF0:USTF0", "tETHF0:BTCF0"}
		assert.Equal(t, exp, keys, "GetDerivativesStatus should return every pair's status")
	} else {
		assert.Contains(t, keys, "tBTCF0:USTF0", "GetDerivativesStatus should return every pair's status")
	}

	_, err = newUnavailableServerExchange(t).GetDerivativesStatus(t.Context(), nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDerivativesStatus should return the request failure")
}

func TestGetDerivativesStatusHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDerivativesStatusHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDerivativesStatusHistory must reject a nil request")
	for _, key := range []string{"", "BTCF0:USTF0", "fUSD"} {
		_, err = e.GetDerivativesStatusHistory(t.Context(), &DerivativesStatusHistoryRequest{Key: key})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "GetDerivativesStatusHistory must refuse key %q", key)
	}
	_, err = e.GetDerivativesStatusHistory(t.Context(), &DerivativesStatusHistoryRequest{Key: "tBTCF0:USTF0", Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDerivativesStatusHistory must refuse a reversed window")

	start, end := testPublicHistoryWindow()
	result, err := e.GetDerivativesStatusHistory(t.Context(), &DerivativesStatusHistoryRequest{Key: "tBTCF0:USTF0", Start: start, End: end, Limit: testPublicHistoryLimit, Sort: 1})
	require.NoError(t, err, "GetDerivativesStatusHistory must not error")
	if mockTests {
		exp := []DerivativeStatus{
			{
				Time: types.Time(time.UnixMilli(1792022457000)), DerivativePrice: 81735.99909449, SpotPrice: 81690, InsuranceFundBalance: 61384568.96803488,
				NextFundingTime: types.Time(time.UnixMilli(1792051200000)), NextFundingAccrued: 0.00000111, NextFundingStep: 17, CurrentFunding: 0.0001331,
				MarkPrice: 81689.5573, OpenInterest: 9235.50464195, ClampMinimum: 0.0005, ClampMaximum: 0.0025,
			},
			{
				Time: types.Time(time.UnixMilli(1792022516000)), DerivativePrice: 81715.14533084, SpotPrice: 81665.5, InsuranceFundBalance: 61384568.96803488,
				NextFundingTime: types.Time(time.UnixMilli(1792051200000)), NextFundingAccrued: 0.00000236, NextFundingStep: 36, CurrentFunding: 0.0001331,
				MarkPrice: 81660.9393, OpenInterest: 9235.4417504, ClampMinimum: 0.0005, ClampMaximum: 0.0025,
			},
			{
				Time: types.Time(time.UnixMilli(1792022576000)), DerivativePrice: 81790.141911965, SpotPrice: 81733.5, InsuranceFundBalance: 61384568.96803488,
				NextFundingTime: types.Time(time.UnixMilli(1792051200000)), NextFundingAccrued: 0.00000391, NextFundingStep: 55, CurrentFunding: 0.0001331,
				MarkPrice: 81730.472, OpenInterest: 9235.40582521, ClampMinimum: 0.0005, ClampMaximum: 0.0025,
			},
		}
		assert.Equal(t, exp, result, "GetDerivativesStatusHistory should decode every status")
	} else {
		assert.NotEmpty(t, result, "GetDerivativesStatusHistory should return the pair's status history")
	}

	_, err = newUnavailableServerExchange(t).GetDerivativesStatusHistory(t.Context(), &DerivativesStatusHistoryRequest{Key: "tBTCF0:USTF0"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDerivativesStatusHistory should return the request failure")
}

func TestGetLiquidations(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidations(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLiquidations must reject a nil request")
	_, err = e.GetLiquidations(t.Context(), &LiquidationsRequest{Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLiquidations must refuse a reversed window")

	// Live, the latest liquidations are asked for, as an hour can pass without one
	req := &LiquidationsRequest{Limit: testPublicHistoryLimit}
	if mockTests {
		req = &LiquidationsRequest{Start: time.UnixMilli(testPublicHistoryStartMilli), End: time.UnixMilli(testPublicHistoryEndMilli), Limit: testPublicHistoryLimit, Sort: 1}
	}
	result, err := e.GetLiquidations(t.Context(), req)
	require.NoError(t, err, "GetLiquidations must not error")
	if mockTests {
		exp := []Liquidation{
			{Type: "pos", PositionID: 194460875, Time: types.Time(time.UnixMilli(1792030515850)), Symbol: "tDEXEF0:USTF0", Amount: 52.3842728, BasePrice: 1.8957, IsMarketSold: true},
			{Type: "pos", PositionID: 194460875, Time: types.Time(time.UnixMilli(1792030515852)), Symbol: "tDEXEF0:USTF0", Amount: 52.3842728, BasePrice: 1.8957, IsMatch: true, IsMarketSold: true, PriceAcquired: 1.7488},
			{Type: "pos", PositionID: 194320977, Time: types.Time(time.UnixMilli(1792030726016)), Symbol: "tZECUSD", Amount: 2.00001238, BasePrice: 1276.159368523508, IsMarketSold: true},
		}
		assert.Equal(t, exp, result, "GetLiquidations should decode every liquidation")
	} else {
		assert.NotEmpty(t, result, "GetLiquidations should return the latest liquidations")
	}

	result, err = newResponseServerExchange(t, `null`).GetLiquidations(t.Context(), &LiquidationsRequest{})
	require.NoError(t, err, "GetLiquidations must not error for a null response")
	assert.Empty(t, result, "GetLiquidations should return no liquidations for a null response")
	_, err = newUnavailableServerExchange(t).GetLiquidations(t.Context(), &LiquidationsRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLiquidations should return the request failure")
}

func TestLiquidationsResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	resp := liquidationsResponse{{Type: "kept"}}
	require.NoError(t, resp.UnmarshalJSON([]byte(`[]`)), "UnmarshalJSON must not error for no liquidations")
	assert.Empty(t, resp, "UnmarshalJSON should decode no liquidations from an empty list, dropping earlier ones")

	require.NoError(t, resp.UnmarshalJSON([]byte(`[[["pos",1,1700000000002,null,"tBTCF0:USTF0",1.5]],null,[null],[["pos",2],["pos",3]]]`)), "UnmarshalJSON must not error")
	exp := liquidationsResponse{
		{Type: "pos", PositionID: 1, Time: types.Time(time.UnixMilli(1700000000002)), Symbol: "tBTCF0:USTF0", Amount: 1.5},
		{Type: "pos", PositionID: 2},
		{Type: "pos", PositionID: 3},
	}
	assert.Equal(t, exp, resp, "UnmarshalJSON should decode each wrapped liquidation, skipping nulls and keeping any more an array holds")
	// The json package passes null to a slice type's UnmarshalJSON rather than leaving the slice unset
	require.NoError(t, json.Unmarshal([]byte(` null `), &resp), "Unmarshal must not error for null")
	assert.Empty(t, resp, "Unmarshal should decode no liquidations from null, dropping earlier ones")

	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`[["pos",1]]`)), errTupleNotArray, "UnmarshalJSON should refuse a liquidation sent unwrapped")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`[1]`)), errTupleNotArray, "UnmarshalJSON should refuse an element that is not an array")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`[[["pos","x"]]]`)), errTupleElementType, "UnmarshalJSON should refuse a malformed liquidation")
	assert.ErrorIs(t, resp.UnmarshalJSON([]byte(`{}`)), errTupleNotArray, "UnmarshalJSON should refuse an object")
}

func TestGetLeaderboards(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeaderboards(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLeaderboards must reject a nil request")
	_, err = e.GetLeaderboards(t.Context(), &LeaderboardsRequest{Key: "nope", TimeFrame: kline.OneWeek, Symbol: LeaderboardSymbolGlobal})
	require.ErrorIs(t, err, errInvalidLeaderboardKey, "GetLeaderboards must refuse an unknown key")
	_, err = e.GetLeaderboards(t.Context(), &LeaderboardsRequest{Key: LeaderboardKeyVolume, TimeFrame: kline.OneDay, Symbol: LeaderboardSymbolGlobal})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetLeaderboards must refuse a time frame Bitfinex does not rank over")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.GetLeaderboards(t.Context(), &LeaderboardsRequest{Key: LeaderboardKeyVolume, TimeFrame: kline.OneWeek, Symbol: symbol})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "GetLeaderboards must refuse symbol %q", symbol)
	}
	_, err = e.GetLeaderboards(t.Context(), &LeaderboardsRequest{Key: LeaderboardKeyVolume, TimeFrame: kline.OneWeek, Symbol: LeaderboardSymbolGlobal, Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLeaderboards must refuse a reversed window")

	// Live, the latest standings are asked for, as the weekly leaderboard ranks once a week
	req := &LeaderboardsRequest{Key: LeaderboardKeyUnrealisedProfitPeriodDelta, TimeFrame: kline.OneWeek, Symbol: LeaderboardSymbolGlobal, Limit: testPublicHistoryLimit}
	if mockTests {
		req.Start, req.End = time.UnixMilli(testPublicLeaderboardStartMilli), time.UnixMilli(testPublicHistoryEndMilli)
	}
	result, err := e.GetLeaderboards(t.Context(), req)
	require.NoError(t, err, "GetLeaderboards must not error")
	if mockTests {
		exp := []LeaderboardStanding{
			{Time: types.Time(time.UnixMilli(1791763200000)), Username: "countyourmoney", Ranking: 1, Value: 4234382.58},
			{Time: types.Time(time.UnixMilli(1791763200000)), Username: "Bobby-Big-Size", Ranking: 2, Value: 4066612.79},
			{Time: types.Time(time.UnixMilli(1791763200000)), Username: "EGAF", Ranking: 3, Value: 2585259.28},
		}
		assert.Equal(t, exp, result, "GetLeaderboards should decode every standing")
	} else {
		assert.NotEmpty(t, result, "GetLeaderboards should return the leaderboard's standings")
	}

	for timeFrame, name := range map[kline.Interval]string{kline.ThreeHour: "3h", kline.OneWeek: "1w", kline.OneMonth: "1M"} {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equalf(t, "/v2/rankings/vol:"+name+":tBTCUSD/hist", r.URL.Path, "GetLeaderboards should name time frame %s %s", timeFrame, name)
			_, err := w.Write([]byte(`[]`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		_, err = ex.GetLeaderboards(t.Context(), &LeaderboardsRequest{Key: LeaderboardKeyVolume, TimeFrame: timeFrame, Symbol: "tBTCUSD"})
		assert.NoErrorf(t, err, "GetLeaderboards should not error for time frame %s", timeFrame)
	}
	_, err = newUnavailableServerExchange(t).GetLeaderboards(t.Context(), &LeaderboardsRequest{Key: LeaderboardKeyVolume, TimeFrame: kline.OneWeek, Symbol: LeaderboardSymbolGlobal})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLeaderboards should return the request failure")
}

func TestGetFundingStatistics(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingStatistics(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingStatistics must reject a nil request")
	for _, symbol := range []string{"", "f", "tBTCUSD"} {
		_, err = e.GetFundingStatistics(t.Context(), &FundingStatisticsRequest{Symbol: symbol})
		require.ErrorIsf(t, err, errFundingSymbolRequired, "GetFundingStatistics must refuse symbol %q", symbol)
	}
	_, err = e.GetFundingStatistics(t.Context(), &FundingStatisticsRequest{Symbol: "fUSD", Start: time.UnixMilli(2), End: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFundingStatistics must refuse a reversed window")

	start, end := testPublicHistoryWindow()
	result, err := e.GetFundingStatistics(t.Context(), &FundingStatisticsRequest{Symbol: "fUSD", Start: start, End: end, Limit: testPublicHistoryLimit})
	require.NoError(t, err, "GetFundingStatistics must not error")
	if mockTests {
		exp := []FundingStatistic{
			{Time: types.Time(time.UnixMilli(1792105500000)), FlashReturnRate: 0.00000082, AveragePeriod: 82.01, FundingAmount: 5950334298.556745, FundingAmountUsed: 5758174671.332418, FundingBelowThreshold: 402445978.86773586},
			{Time: types.Time(time.UnixMilli(1792101900000)), FlashReturnRate: 0.00000082, AveragePeriod: 81.16, FundingAmount: 5942068272.108026, FundingAmountUsed: 5758506652.673497, FundingBelowThreshold: 410051733.793576},
			{Time: types.Time(time.UnixMilli(1792098300000)), FlashReturnRate: 0.00000082, AveragePeriod: 81.62, FundingAmount: 5941551577.016957, FundingAmountUsed: 5758004731.183161, FundingBelowThreshold: 413009916.16160125},
		}
		assert.Equal(t, exp, result, "GetFundingStatistics should decode every statistic")
	} else {
		assert.NotEmpty(t, result, "GetFundingStatistics should return the currency's statistics")
	}

	_, err = newUnavailableServerExchange(t).GetFundingStatistics(t.Context(), &FundingStatisticsRequest{Symbol: "fUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingStatistics should return the request failure")
}

// testPublicConfig checks a config method decodes its recorded config, or, live, returns one, and that it errors for a config
// Bitfinex has no value for and for a failed request
func testPublicConfig[T any](t *testing.T, name string, get func(*Exchange) (T, error), exp T) {
	t.Helper()
	result, err := get(e)
	require.NoErrorf(t, err, "%s must not error", name)
	if mockTests {
		assert.Equalf(t, exp, result, "%s should decode every field of the config", name)
	} else {
		assert.NotEmptyf(t, result, "%s should return the config", name)
	}
	_, err = get(newResponseServerExchange(t, `[null]`))
	assert.ErrorIsf(t, err, common.ErrNoResponse, "%s should error for a config Bitfinex has no value for", name)
	_, err = get(newUnavailableServerExchange(t))
	assert.ErrorIsf(t, err, request.ErrBadStatus, "%s should return the request failure", name)
}

func TestGetCurrencySymbols(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencySymbols", func(ex *Exchange) ([]CurrencySymbol, error) { return ex.GetCurrencySymbols(t.Context()) }, []CurrencySymbol{
		{Currency: currency.NewCode("ALG"), Symbol: "ALGO"},
		{Currency: currency.NewCode("ATO"), Symbol: "ATOM"},
		{Currency: currency.NewCode("AUSDT"), Symbol: "aUSDT"},
		{Currency: currency.NewCode("BTCF0"), Symbol: "BTC"},
		{Currency: currency.NewCode("CHZF0"), Symbol: "CHZ"},
	})
}

func TestGetCurrencyLabels(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencyLabels", func(ex *Exchange) ([]CurrencyLabel, error) { return ex.GetCurrencyLabels(t.Context()) }, []CurrencyLabel{
		{Currency: currency.AAVE, Label: "Aave"},
		{Currency: currency.ADA, Label: "Cardano ADA"},
		{Currency: currency.NewCode("ALG"), Label: "Algorand"},
		{Currency: currency.NewCode("ALKN"), Label: "ALKN"},
		{Currency: currency.NewCode("ALT11M2507"), Label: "ALT11M2507"},
	})
}

func TestGetCurrencyUnits(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencyUnits", func(ex *Exchange) ([]CurrencyUnit, error) { return ex.GetCurrencyUnits(t.Context()) }, []CurrencyUnit{
		{Currency: currency.IOT, Unit: "Mi|MegaIOTA"},
	})
}

func TestGetCurrencyUnderlyings(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencyUnderlyings", func(ex *Exchange) ([]CurrencyUnderlying, error) { return ex.GetCurrencyUnderlyings(t.Context()) }, []CurrencyUnderlying{
		{Currency: currency.NewCode("AAVEF0"), Underlying: currency.AAVE},
		{Currency: currency.NewCode("ADAF0"), Underlying: currency.ADA},
		{Currency: currency.NewCode("ALGF0"), Underlying: currency.NewCode("ALG")},
		{Currency: currency.NewCode("APEF0"), Underlying: currency.APE},
		{Currency: currency.NewCode("APTF0"), Underlying: currency.APT},
	})
}

func TestGetCurrencyPools(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencyPools", func(ex *Exchange) ([]CurrencyPool, error) { return ex.GetCurrencyPools(t.Context()) }, []CurrencyPool{
		{Currency: currency.AAVE, Pool: currency.ETH},
		{Currency: currency.NewCode("ALKN"), Pool: currency.ETH},
		{Currency: currency.ARB, Pool: currency.NewCode("ARBETH")},
		{Currency: currency.NewCode("AUSDT"), Pool: currency.ETH},
		{Currency: currency.NewCode("B2M"), Pool: currency.ETH},
	})
}

func TestGetCurrencyExplorers(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencyExplorers", func(ex *Exchange) ([]CurrencyExplorer, error) { return ex.GetCurrencyExplorers(t.Context()) }, []CurrencyExplorer{
		{Currency: currency.ADA, Explorer: BlockExplorer{URL: "https://adastat.net", AddressURL: "https://adastat.net/addresses/VAL", TransactionURL: "https://adastat.net/transactions/VAL"}},
		// Bitfinex sends this address URL with its trailing space
		{Currency: currency.NewCode("ALG"), Explorer: BlockExplorer{URL: "https://allo.info", AddressURL: "https://allo.info/account/VAL ", TransactionURL: "https://allo.info/tx/VAL"}},
		{Currency: currency.APT, Explorer: BlockExplorer{URL: "https://explorer.aptoslabs.com", AddressURL: "https://explorer.aptoslabs.com/account/VAL", TransactionURL: "https://explorer.aptoslabs.com/txn/VAL"}},
		{Currency: currency.NewCode("ARBETH"), Explorer: BlockExplorer{URL: "https://arbiscan.io", AddressURL: "https://arbiscan.io/address/VAL", TransactionURL: "https://arbiscan.io/tx/VAL"}},
		{Currency: currency.NewCode("ATO"), Explorer: BlockExplorer{URL: "https://www.mintscan.io/cosmos", AddressURL: "https://www.mintscan.io/cosmos/account/VAL", TransactionURL: "https://www.mintscan.io/cosmos/txs/VAL"}},
	})
}

func TestGetCurrencyWithdrawalFees(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencyWithdrawalFees", func(ex *Exchange) ([]CurrencyWithdrawalFee, error) { return ex.GetCurrencyWithdrawalFees(t.Context()) }, []CurrencyWithdrawalFee{
		{Currency: currency.AAVE, Fee: WithdrawalFee{Amount: 0.018825}},
		{Currency: currency.ADA, Fee: WithdrawalFee{Amount: 0.3}},
		{Currency: currency.NewCode("ALG")},
		{Currency: currency.NewCode("ALKN")},
		{Currency: currency.NewCode("ALT11M2507")},
	})
}

func TestGetTransferMethods(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetTransferMethods", func(ex *Exchange) ([]TransferMethod, error) { return ex.GetTransferMethods(t.Context()) }, []TransferMethod{
		{Method: "BITCOIN", Currencies: []currency.Code{currency.BTC}},
		{Method: "LITECOIN", Currencies: []currency.Code{currency.LTC}},
		{Method: "ETHEREUM", Currencies: []currency.Code{currency.ETH}},
		{Method: "ETHEREUMC", Currencies: []currency.Code{currency.ETC}},
		{Method: "ZCASH", Currencies: []currency.Code{currency.ZEC}},
	})
}

func TestGetSpotPairs(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetSpotPairs", func(ex *Exchange) ([]string, error) { return ex.GetSpotPairs(t.Context()) }, []string{"AAVE:USD", "AAVE:UST", "ADABTC", "ADAUSD", "ADAUST"})
}

func TestGetMarginPairs(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetMarginPairs", func(ex *Exchange) ([]string, error) { return ex.GetMarginPairs(t.Context()) }, []string{"ADABTC", "ADAUSD", "ADAUST", "ALGUSD", "APTUSD"})
}

func TestGetFuturesPairs(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetFuturesPairs", func(ex *Exchange) ([]string, error) { return ex.GetFuturesPairs(t.Context()) }, []string{
		"AAVEF0:USTF0", "ADAF0:USTF0", "ALGF0:USTF0", "ALLOF0:USTF0", "APEF0:USTF0", "BTCF0:USTF0", "ETHF0:BTCF0", "TESTBTCF0:TESTUSDTF0",
	})
}

func TestGetSecuritiesPairs(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetSecuritiesPairs", func(ex *Exchange) ([]string, error) { return ex.GetSecuritiesPairs(t.Context()) }, []string{"ALKN:USD", "ALT2612:USD", "ALT2612:UST", "BMN2:BTC", "BMN2:USD"})
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetCurrencies", func(ex *Exchange) ([]currency.Code, error) { return ex.GetCurrencies(t.Context()) }, []currency.Code{
		currency.AAVE, currency.NewCode("AAVEF0"), currency.ADA, currency.NewCode("ADAF0"), currency.NewCode("ALG"),
	})
}

func TestGetMarginCurrencies(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetMarginCurrencies", func(ex *Exchange) ([]currency.Code, error) { return ex.GetMarginCurrencies(t.Context()) }, []currency.Code{
		currency.ADA, currency.NewCode("ALG"), currency.APT, currency.AVAX, currency.NewCode("BCHN"),
	})
}

func TestGetLeaderboardCompetitions(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetLeaderboardCompetitions", func(ex *Exchange) ([]string, error) { return ex.GetLeaderboardCompetitions(t.Context()) }, []string{
		"plu_diff:1w:tGLOBAL:USD", "plu_diff:1M:tGLOBAL:USD", "plu:1w:tGLOBAL:USD", "plu:3h:tGLOBAL:USD", "vol:1w:tGLOBAL:USD",
	})
}

func TestGetPairInfo(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetPairInfo", func(ex *Exchange) ([]PairInfoEntry, error) { return ex.GetPairInfo(t.Context()) }, []PairInfoEntry{
		{Pair: "AAVE:USD", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1612517298684)), MinimumOrderSize: 0.02, MaximumOrderSize: 5000}},
		{Pair: "AAVE:UST", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1612518415535)), MinimumOrderSize: 0.02, MaximumOrderSize: 5000}},
		{Pair: "ADABTC", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1596710520048)), MinimumOrderSize: 4, MaximumOrderSize: 250000, InitialMargin: 0.3, MinimumMargin: 0.15}},
		{Pair: "ADAUSD", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1596708663880)), MinimumOrderSize: 4, MaximumOrderSize: 250000, InitialMargin: 0.3, MinimumMargin: 0.15}},
		{Pair: "ADAUST", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1596708684324)), MinimumOrderSize: 4, MaximumOrderSize: 250000, InitialMargin: 0.3, MinimumMargin: 0.15}},
	})
}

func TestGetFuturesPairInfo(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetFuturesPairInfo", func(ex *Exchange) ([]PairInfoEntry, error) { return ex.GetFuturesPairInfo(t.Context()) }, []PairInfoEntry{
		{Pair: "AAVEF0:USTF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1633350178515)), MinimumOrderSize: 0.02, MaximumOrderSize: 5000, InitialMargin: 0.01, MinimumMargin: 0.005}},
		{Pair: "ADAF0:USTF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1612351549219)), MinimumOrderSize: 4, MaximumOrderSize: 250000, InitialMargin: 0.01, MinimumMargin: 0.005}},
		{Pair: "ALGF0:USTF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1631890038807)), MinimumOrderSize: 8, MaximumOrderSize: 250000, InitialMargin: 0.01, MinimumMargin: 0.005}},
		{Pair: "ALLOF0:USTF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1787576315252)), MinimumOrderSize: 0.001, MaximumOrderSize: 500000, InitialMargin: 0.05, MinimumMargin: 0.025}},
		{Pair: "APEF0:USTF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1648035593066)), MinimumOrderSize: 0.8, MaximumOrderSize: 50000, InitialMargin: 0.01, MinimumMargin: 0.005}},
		{Pair: "BTCF0:USTF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1562164542332)), MinimumOrderSize: 0.00004, MaximumOrderSize: 100, InitialMargin: 0.01, MinimumMargin: 0.005}},
		{Pair: "ETHF0:BTCF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1608633542930)), MinimumOrderSize: 0.0008, MaximumOrderSize: 100, InitialMargin: 0.01, MinimumMargin: 0.005}},
		{Pair: "TESTBTCF0:TESTUSDTF0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1594021206659)), MinimumOrderSize: 0.00004, MaximumOrderSize: 1000, InitialMargin: 0.01, MinimumMargin: 0.005}},
	})
}

func TestGetTransferMethodStatuses(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetTransferMethodStatuses", func(ex *Exchange) ([]TransferMethodStatus, error) { return ex.GetTransferMethodStatuses(t.Context()) }, []TransferMethodStatus{
		{Method: "BITCOIN", DepositActive: true, WithdrawalActive: true, DepositConfirmationsRequired: 3},
		{Method: "LITECOIN", DepositActive: true, WithdrawalActive: true, DepositConfirmationsRequired: 24},
		{Method: "ETHEREUM", DepositActive: true, WithdrawalActive: true, DepositConfirmationsRequired: 5},
		{Method: "ETHEREUMC", DepositActive: true, WithdrawalActive: true, DepositConfirmationsRequired: 5},
		{Method: "ZCASH", DepositConfirmationsRequired: 15},
	})
}

func TestGetMarginSpecification(t *testing.T) {
	t.Parallel()
	// The recording keeps the first keys of each object
	testPublicConfig(t, "GetMarginSpecification", func(ex *Exchange) (*MarginSpecificationResponse, error) {
		return ex.GetMarginSpecification(t.Context())
	}, &MarginSpecificationResponse{
		Configuration: map[string]float64{
			"base_xmarg_mul": 1, "base_xmarg_step_mul": 1, "base_size_thres": 0, "base_size_incr": 1, "base_step": 0, "base_haircut": 1,
			"base_tradable_penalty": 0.05, "XRP_haircut": 0.5,
		},
		Initial: map[string]float64{"base": 0.3, "base_cap": 1, "BTCUSD": 0.1, "LTCUSD": 0.2, "LTCBTC": 0.3},
		Minimum: map[string]float64{"base": 0.15, "base_cap": 1, "BTCUSD": 0.05, "LTCUSD": 0.1, "LTCBTC": 0.15},
	})
}

func TestGetDerivativesFees(t *testing.T) {
	t.Parallel()
	testPublicConfig(t, "GetDerivativesFees", func(ex *Exchange) (*DerivativesFeesResponse, error) { return ex.GetDerivativesFees(t.Context()) }, &DerivativesFeesResponse{
		Perpetual: PerpetualContractFees{CorrectionClampMinimum: 0.0025, CorrectionClampInsurance: 0.0015, CorrectionSlope: 0.2},
	})
}

func TestGetPublicConfig(t *testing.T) {
	t.Parallel()
	result, err := getPublicConfig[[]string](t.Context(), newResponseServerExchange(t, `[["BTCUSD","ETHUSD"]]`), "conf/pub:list:pair:exchange")
	require.NoError(t, err, "getPublicConfig must not error")
	assert.Equal(t, []string{"BTCUSD", "ETHUSD"}, result, "getPublicConfig should decode the config")
	result, err = getPublicConfig[[]string](t.Context(), newResponseServerExchange(t, `[[]]`), "conf/pub:list:pair:exchange")
	require.NoError(t, err, "getPublicConfig must not error for a config with no entries")
	assert.Empty(t, result, "getPublicConfig should decode a config with no entries")
	for _, body := range []string{`[null]`, `[]`, `null`} {
		_, err = getPublicConfig[[]string](t.Context(), newResponseServerExchange(t, body), "conf/pub:list:pair:exchange")
		assert.ErrorIsf(t, err, common.ErrNoResponse, "getPublicConfig should error for response %s", body)
	}
	_, err = getPublicConfig[[]string](t.Context(), newResponseServerExchange(t, `[1]`), "conf/pub:list:pair:exchange")
	assert.Error(t, err, "getPublicConfig should return a decoding error")
	_, err = getPublicConfig[[]string](t.Context(), newUnavailableServerExchange(t), "conf/pub:list:pair:exchange")
	assert.ErrorIs(t, err, request.ErrBadStatus, "getPublicConfig should return the request failure")
}

func TestGetVirtualAssetServiceProviders(t *testing.T) {
	t.Parallel()
	result, err := e.GetVirtualAssetServiceProviders(t.Context())
	require.NoError(t, err, "GetVirtualAssetServiceProviders must not error")
	if mockTests {
		exp := []VirtualAssetServiceProvider{
			{ID: "did:ethr:0xf2bd35dc59cbbe3efb15a5bd411ac84e7c86d25a", Name: "100s Holdings Limited (IM)"},
			{ID: "did:ethr:0xdf41ef7dcb92ab5aed72e284db3766ace5026948", Name: "1Money (US)"},
			{ID: "did:ethr:0xfeb4f99829a9acdf52979abee87e83addf22a7e1", Name: "1xBet"},
			{ID: "did:ethr:0x5d598bbc15fd01dce280ac65d8d3a9b8b748d202", Name: "1xSlots"},
			{ID: "did:ethr:0xc6423fb22b31148b0de37051a98241ba2487269e", Name: "21bitcoin (AT)"},
		}
		assert.Equal(t, exp, result, "GetVirtualAssetServiceProviders should decode every provider")
	} else {
		assert.NotEmpty(t, result, "GetVirtualAssetServiceProviders should return the providers")
	}

	_, err = newUnavailableServerExchange(t).GetVirtualAssetServiceProviders(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetVirtualAssetServiceProviders should return the request failure")
}

func TestGetMarketAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketAveragePrice(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarketAveragePrice must reject a nil request")
	for _, symbol := range []string{"", "fUSD"} {
		_, err = e.GetMarketAveragePrice(t.Context(), &MarketAveragePriceRequest{Symbol: symbol, Amount: 1})
		require.ErrorIsf(t, err, errTradingSymbolRequired, "GetMarketAveragePrice must refuse symbol %q", symbol)
	}
	_, err = e.GetMarketAveragePrice(t.Context(), &MarketAveragePriceRequest{Symbol: "tBTCUSD"})
	require.ErrorIs(t, err, errInvalidAmount, "GetMarketAveragePrice must refuse a zero amount")

	result, err := e.GetMarketAveragePrice(t.Context(), &MarketAveragePriceRequest{Symbol: "tBTCUSD", Amount: 1.123})
	require.NoError(t, err, "GetMarketAveragePrice must not error")
	if mockTests {
		exp := &MarketAveragePriceResponse{AveragePrice: 82203.72844229743, Amount: 1.123}
		assert.Equal(t, exp, result, "GetMarketAveragePrice should decode the average price")
	} else {
		assert.Positive(t, result.AveragePrice, "GetMarketAveragePrice should return the average price")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "GetMarketAveragePrice should send a POST request")
		assert.Equal(t, url.Values{"symbol": {"tBTCUSD"}, "amount": {"-0.00001"}, "rate_limit": {"82000.5"}}, r.URL.Query(), "GetMarketAveragePrice should send its parameters in the query")
		assert.Empty(t, r.Header.Get("Content-Type"), "GetMarketAveragePrice should send no body")
		_, err := w.Write([]byte(`[1.5,-0.00001]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetMarketAveragePrice(t.Context(), &MarketAveragePriceRequest{Symbol: "tBTCUSD", Amount: -0.00001, PriceLimit: 82000.5})
	require.NoError(t, err, "GetMarketAveragePrice must not error with a price limit")
	_, err = newResponseServerExchange(t, `null`).GetMarketAveragePrice(t.Context(), &MarketAveragePriceRequest{Symbol: "tBTCUSD", Amount: 1})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMarketAveragePrice should error for a null response")
	_, err = newUnavailableServerExchange(t).GetMarketAveragePrice(t.Context(), &MarketAveragePriceRequest{Symbol: "tBTCUSD", Amount: 1})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMarketAveragePrice should return the request failure")
}

func TestGetFundingMarketAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingMarketAveragePrice(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingMarketAveragePrice must reject a nil request")
	for _, symbol := range []string{"", "tBTCUSD"} {
		_, err = e.GetFundingMarketAveragePrice(t.Context(), &FundingMarketAveragePriceRequest{Symbol: symbol, Amount: 1, Period: 2})
		require.ErrorIsf(t, err, errFundingSymbolRequired, "GetFundingMarketAveragePrice must refuse symbol %q", symbol)
	}
	_, err = e.GetFundingMarketAveragePrice(t.Context(), &FundingMarketAveragePriceRequest{Symbol: "fUSD", Period: 2})
	require.ErrorIs(t, err, errInvalidAmount, "GetFundingMarketAveragePrice must refuse a zero amount")
	_, err = e.GetFundingMarketAveragePrice(t.Context(), &FundingMarketAveragePriceRequest{Symbol: "fUSD", Amount: 1})
	require.ErrorIs(t, err, errMarketAveragePeriodRequired, "GetFundingMarketAveragePrice must refuse a missing period")

	result, err := e.GetFundingMarketAveragePrice(t.Context(), &FundingMarketAveragePriceRequest{Symbol: "fUSD", Amount: -1.123, Period: 2})
	require.NoError(t, err, "GetFundingMarketAveragePrice must not error")
	if mockTests {
		exp := &FundingMarketAveragePriceResponse{AverageRate: 0.0001712, Amount: -1.123}
		assert.Equal(t, exp, result, "GetFundingMarketAveragePrice should decode the average rate")
	} else {
		assert.Positive(t, result.AverageRate, "GetFundingMarketAveragePrice should return the average rate")
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "GetFundingMarketAveragePrice should send a POST request")
		assert.Equal(t, url.Values{"symbol": {"fUSD"}, "amount": {"500"}, "period": {"30"}, "rate_limit": {"0.0002"}}, r.URL.Query(), "GetFundingMarketAveragePrice should send its parameters in the query")
		assert.Empty(t, r.Header.Get("Content-Type"), "GetFundingMarketAveragePrice should send no body")
		_, err := w.Write([]byte(`[0.0001,500]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetFundingMarketAveragePrice(t.Context(), &FundingMarketAveragePriceRequest{Symbol: "fUSD", Amount: 500, Period: 30, RateLimit: 0.0002})
	require.NoError(t, err, "GetFundingMarketAveragePrice must not error with a rate limit")
	_, err = newResponseServerExchange(t, `null`).GetFundingMarketAveragePrice(t.Context(), &FundingMarketAveragePriceRequest{Symbol: "fUSD", Amount: 1, Period: 2})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFundingMarketAveragePrice should error for a null response")
	_, err = newUnavailableServerExchange(t).GetFundingMarketAveragePrice(t.Context(), &FundingMarketAveragePriceRequest{Symbol: "fUSD", Amount: 1, Period: 2})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFundingMarketAveragePrice should return the request failure")
}

func TestGetForeignExchangeRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetForeignExchangeRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetForeignExchangeRate must reject a nil request")
	_, err = e.GetForeignExchangeRate(t.Context(), &ForeignExchangeRateRequest{QuoteCurrency: currency.USD})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetForeignExchangeRate must refuse an empty base currency")
	_, err = e.GetForeignExchangeRate(t.Context(), &ForeignExchangeRateRequest{BaseCurrency: currency.BTC})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetForeignExchangeRate must refuse an empty quote currency")

	result, err := e.GetForeignExchangeRate(t.Context(), &ForeignExchangeRateRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USD})
	require.NoError(t, err, "GetForeignExchangeRate must not error")
	if mockTests {
		exp := &ForeignExchangeRateResponse{CurrentRate: 82172.5}
		assert.Equal(t, exp, result, "GetForeignExchangeRate should decode the rate")
	} else {
		assert.Positive(t, result.CurrentRate, "GetForeignExchangeRate should return the rate")
	}

	_, err = e.GetForeignExchangeRate(t.Context(), &ForeignExchangeRateRequest{BaseCurrency: currency.NewCode("NOPE"), QuoteCurrency: currency.USD})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetForeignExchangeRate should error for the [null] Bitfinex sends for a currency it has no rate for")
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"ccy1":"BTC","ccy2":"USD"}`, string(body), "GetForeignExchangeRate should send the currencies in upper case")
		}
		_, err = w.Write([]byte(`[1.5]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = ex.GetForeignExchangeRate(t.Context(), &ForeignExchangeRateRequest{BaseCurrency: currency.NewCode("btc"), QuoteCurrency: currency.NewCode("usd")})
	require.NoError(t, err, "GetForeignExchangeRate must not error for lower case currencies")
	_, err = newUnavailableServerExchange(t).GetForeignExchangeRate(t.Context(), &ForeignExchangeRateRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USD})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetForeignExchangeRate should return the request failure")
}

// TestPublicRecordsEveryPosition decodes each public record from an array with a distinct value at every tagged
// position, so a field tagged with the wrong position or type cannot pass on a null or zero element
func TestPublicRecordsEveryPosition(t *testing.T) {
	t.Parallel()
	testTupleDecode(t, `[0.5,1.5,2.5,3.5,4.5,5.5,6.5,7.5,8.5,9.5,1700000000010]`,
		Ticker{Bid: 0.5, BidSize: 1.5, Ask: 2.5, AskSize: 3.5, DailyChange: 4.5, DailyChangeRelative: 5.5, LastPrice: 6.5, Volume: 7.5, High: 8.5, Low: 9.5, FirstTradeTime: types.Time(time.UnixMilli(1700000000010))})
	testTupleDecode(t, `[0.5,1.5,1002,3.5,4.5,1005,6.5,7.5,8.5,9.5,10.5,11.5,12.5,null,null,15.5,1700000000016]`,
		FundingTicker{FlashReturnRate: 0.5, Bid: 1.5, BidPeriod: 1002, BidSize: 3.5, Ask: 4.5, AskPeriod: 1005, AskSize: 6.5, DailyChange: 7.5, DailyChangeRelative: 8.5, LastRate: 9.5, Volume: 10.5, High: 11.5, Low: 12.5, FlashReturnRateAmountAvailable: 15.5, FirstTradeTime: types.Time(time.UnixMilli(1700000000016))})
	testTupleDecode(t, `["s0",1.5,2.5,3.5,4.5,5.5,6.5,7.5,8.5,9.5,10.5,1700000000011]`,
		TickerEntry{Symbol: "s0", Ticker: Ticker{Bid: 1.5, BidSize: 2.5, Ask: 3.5, AskSize: 4.5, DailyChange: 5.5, DailyChangeRelative: 6.5, LastPrice: 7.5, Volume: 8.5, High: 9.5, Low: 10.5, FirstTradeTime: types.Time(time.UnixMilli(1700000000011))}})
	testTupleDecode(t, `["s0",1.5,2.5,1003,4.5,5.5,1006,7.5,8.5,9.5,10.5,11.5,12.5,13.5,null,null,16.5,1700000000017]`,
		FundingTickerEntry{Symbol: "s0", Ticker: FundingTicker{FlashReturnRate: 1.5, Bid: 2.5, BidPeriod: 1003, BidSize: 4.5, Ask: 5.5, AskPeriod: 1006, AskSize: 7.5, DailyChange: 8.5, DailyChangeRelative: 9.5, LastRate: 10.5, Volume: 11.5, High: 12.5, Low: 13.5, FlashReturnRateAmountAvailable: 16.5, FirstTradeTime: types.Time(time.UnixMilli(1700000000017))}})
	testTupleDecode(t, `[1000,1700000000001,2.5,3.5]`,
		PublicTrade{ID: 1000, Time: types.Time(time.UnixMilli(1700000000001)), Amount: 2.5, Price: 3.5})
	testTupleDecode(t, `[1000,1700000000001,2.5,3.5,1004]`,
		PublicFundingTrade{ID: 1000, Time: types.Time(time.UnixMilli(1700000000001)), Amount: 2.5, Rate: 3.5, Period: 1004})
	testTupleDecode(t, `[0.5,1001,2.5]`,
		BookLevel{Price: 0.5, Count: 1001, Amount: 2.5})
	testTupleDecode(t, `[0.5,1001,1002,3.5]`,
		FundingBookLevel{Rate: 0.5, Period: 1001, Count: 1002, Amount: 3.5})
	testTupleDecode(t, `[1000,1.5,2.5]`,
		RawBookLevel{OrderID: 1000, Price: 1.5, Amount: 2.5})
	testTupleDecode(t, `[1000,1001,2.5,3.5]`,
		RawFundingBookLevel{OfferID: 1000, Period: 1001, Rate: 2.5, Amount: 3.5})
	testTupleDecode(t, `[1700000000000,1.5,2.5,3.5,4.5,5.5]`,
		Candle{Time: types.Time(time.UnixMilli(1700000000000)), Open: 1.5, Close: 2.5, High: 3.5, Low: 4.5, Volume: 5.5})
	testTupleDecode(t, `[1700000000000,null,2.5,3.5,null,5.5,null,1700000000007,8.5,1009,null,11.5,null,null,14.5,null,null,17.5,null,null,null,21.5,22.5]`,
		DerivativeStatus{Time: types.Time(time.UnixMilli(1700000000000)), DerivativePrice: 2.5, SpotPrice: 3.5, InsuranceFundBalance: 5.5, NextFundingTime: types.Time(time.UnixMilli(1700000000007)), NextFundingAccrued: 8.5, NextFundingStep: 1009, CurrentFunding: 11.5, MarkPrice: 14.5, OpenInterest: 17.5, ClampMinimum: 21.5, ClampMaximum: 22.5})
	testTupleDecode(t, `["s0",1700000000001,null,3.5,4.5,null,6.5,null,1700000000008,9.5,1010,null,12.5,null,null,15.5,null,null,18.5,null,null,null,22.5,23.5]`,
		DerivativeStatusEntry{Key: "s0", Status: DerivativeStatus{Time: types.Time(time.UnixMilli(1700000000001)), DerivativePrice: 3.5, SpotPrice: 4.5, InsuranceFundBalance: 6.5, NextFundingTime: types.Time(time.UnixMilli(1700000000008)), NextFundingAccrued: 9.5, NextFundingStep: 1010, CurrentFunding: 12.5, MarkPrice: 15.5, OpenInterest: 18.5, ClampMinimum: 22.5, ClampMaximum: 23.5}})
	testTupleDecode(t, `["s0",1001,1700000000002,null,"s4",5.5,6.5,null,1,1,null,11.5]`,
		Liquidation{Type: "s0", PositionID: 1001, Time: types.Time(time.UnixMilli(1700000000002)), Symbol: "s4", Amount: 5.5, BasePrice: 6.5, IsMatch: true, IsMarketSold: true, PriceAcquired: 11.5})
	testTupleDecode(t, `[1]`,
		PlatformStatusResponse{Operative: true})
	testTupleDecode(t, `["s0",1.5,null,3.5,null,null,null,null,null,null,null,null,1700000000012]`,
		HistoricTicker{Symbol: "s0", Bid: 1.5, Ask: 3.5, Time: types.Time(time.UnixMilli(1700000000012))})
	testTupleDecode(t, `[1700000000000,1.5]`,
		Stat{Time: types.Time(time.UnixMilli(1700000000000)), Value: 1.5})
	testTupleDecode(t, `[1700000000000,null,"s2",1003,null,null,6.5,null,null,"s9"]`,
		LeaderboardStanding{Time: types.Time(time.UnixMilli(1700000000000)), Username: "s2", Ranking: 1003, Value: 6.5, TwitterHandle: "s9"})
	testTupleDecode(t, `[1700000000000,null,null,3.5,4.5,null,null,7.5,8.5,null,null,11.5]`,
		FundingStatistic{Time: types.Time(time.UnixMilli(1700000000000)), FlashReturnRate: 3.5, AveragePeriod: 4.5, FundingAmount: 7.5, FundingAmountUsed: 8.5, FundingBelowThreshold: 11.5})
	testTupleDecode(t, `["UST","s1"]`,
		CurrencySymbol{Currency: currency.UST, Symbol: "s1"})
	testTupleDecode(t, `["UST","s1"]`,
		CurrencyLabel{Currency: currency.UST, Label: "s1"})
	testTupleDecode(t, `["UST","s1"]`,
		CurrencyUnit{Currency: currency.UST, Unit: "s1"})
	testTupleDecode(t, `["UST","BTC"]`,
		CurrencyUnderlying{Currency: currency.UST, Underlying: currency.BTC})
	testTupleDecode(t, `["UST","BTC"]`,
		CurrencyPool{Currency: currency.UST, Pool: currency.BTC})
	testTupleDecode(t, `["UST",["s0","s1","s2"]]`,
		CurrencyExplorer{Currency: currency.UST, Explorer: BlockExplorer{URL: "s0", AddressURL: "s1", TransactionURL: "s2"}})
	testTupleDecode(t, `["s0","s1","s2"]`,
		BlockExplorer{URL: "s0", AddressURL: "s1", TransactionURL: "s2"})
	testTupleDecode(t, `["UST",[null,1.5]]`,
		CurrencyWithdrawalFee{Currency: currency.UST, Fee: WithdrawalFee{Amount: 1.5}})
	testTupleDecode(t, `[null,1.5]`,
		WithdrawalFee{Amount: 1.5})
	testTupleDecode(t, `["s0",["UST","BTC"]]`,
		TransferMethod{Method: "s0", Currencies: []currency.Code{currency.UST, currency.BTC}})
	testTupleDecode(t, `["s0",[1700000000000,null,null,"3.5","4.5",null,null,null,8.5,9.5]]`,
		PairInfoEntry{Pair: "s0", Info: PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1700000000000)), MinimumOrderSize: 3.5, MaximumOrderSize: 4.5, InitialMargin: 8.5, MinimumMargin: 9.5}})
	testTupleDecode(t, `[1700000000000,null,null,"3.5","4.5",null,null,null,8.5,9.5]`,
		PairInfo{FirstTradeTime: types.Time(time.UnixMilli(1700000000000)), MinimumOrderSize: 3.5, MaximumOrderSize: 4.5, InitialMargin: 8.5, MinimumMargin: 9.5})
	testTupleDecode(t, `["s0",1,1,null,null,null,null,1,1,null,null,1011]`,
		TransferMethodStatus{Method: "s0", DepositActive: true, WithdrawalActive: true, DepositPaymentIDAllowed: true, WithdrawalPaymentIDAllowed: true, DepositConfirmationsRequired: 1011})
	// Mixed flags, so a flag tagged with another's position cannot pass on both decodes
	testTupleDecode(t, `["s0",1,0,null,null,null,null,0,1]`,
		TransferMethodStatus{Method: "s0", DepositActive: true, WithdrawalPaymentIDAllowed: true})
	testTupleDecode(t, `[0.5,1.5]`,
		MarketAveragePriceResponse{AveragePrice: 0.5, Amount: 1.5})
	testTupleDecode(t, `[0.5,1.5]`,
		FundingMarketAveragePriceResponse{AverageRate: 0.5, Amount: 1.5})
	testTupleDecode(t, `[0.5]`,
		ForeignExchangeRateResponse{CurrentRate: 0.5})
}

// TestPublicRecordsDocumentedExamples decodes the docs' examples of each public record, the websocket's being the same
func TestPublicRecordsDocumentedExamples(t *testing.T) {
	t.Parallel()
	testTupleDecode(t, `[388063448,1567526214876,1.918524,10682]`,
		PublicTrade{ID: 388063448, Time: types.Time(time.UnixMilli(1567526214876)), Amount: 1.918524, Price: 10682})
	testTupleDecode(t, `[124486873,1567526287066,-210.69675707,0.00034369,2]`,
		PublicFundingTrade{ID: 124486873, Time: types.Time(time.UnixMilli(1567526287066)), Amount: -210.69675707, Rate: 0.00034369, Period: 2})
	testTupleDecode(t, `[8744.9,2,0.45603413]`, BookLevel{Price: 8744.9, Count: 2, Amount: 0.45603413})
	testTupleDecode(t, `[0.0003301,30,1,-3862.874]`, FundingBookLevel{Rate: 0.0003301, Period: 30, Count: 1, Amount: -3862.874})
	testTupleDecode(t, `[34006738527,8744.9,0.25603413]`, RawBookLevel{OrderID: 34006738527, Price: 8744.9, Amount: 0.25603413})
	testTupleDecode(t, `[645902785,30,0.0003301,-3862.874]`, RawFundingBookLevel{OfferID: 645902785, Period: 30, Rate: 0.0003301, Amount: -3862.874})
	testTupleDecode(t, `[1678465320000,20097,20094,20097,20094,0.07870586]`,
		Candle{Time: types.Time(time.UnixMilli(1678465320000)), Open: 20097, Close: 20094, High: 20097, Low: 20094, Volume: 0.07870586})
	testTupleDecode(t, `[1570578776000,null,8194.95,8196.95,null,101120.80328179,null,1570579200000,0.00021817,9379,null,0,null,null,8192.823333333334,null,null,5174.01444474,null,null,null,0.0005,0.0025]`,
		DerivativeStatus{Time: types.Time(time.UnixMilli(1570578776000)), DerivativePrice: 8194.95, SpotPrice: 8196.95, InsuranceFundBalance: 101120.80328179, NextFundingTime: types.Time(time.UnixMilli(1570579200000)), NextFundingAccrued: 0.00021817, NextFundingStep: 9379, MarkPrice: 8192.823333333334, OpenInterest: 5174.01444474, ClampMinimum: 0.0005, ClampMaximum: 0.0025})
	testTupleDecode(t, `["tAMPF0:USTF0",1596124822000,null,0.896,0.771995,null,1396531.67460709,null,1596153600000,0.0001056,6,null,-0.01381344,null,null,0.7664,null,null,246502.09001551,null,null,null,0.0005,0.0025]`,
		DerivativeStatusEntry{Key: "tAMPF0:USTF0", Status: DerivativeStatus{Time: types.Time(time.UnixMilli(1596124822000)), DerivativePrice: 0.896, SpotPrice: 0.771995, InsuranceFundBalance: 1396531.67460709, NextFundingTime: types.Time(time.UnixMilli(1596153600000)), NextFundingAccrued: 0.0001056, NextFundingStep: 6, CurrentFunding: -0.01381344, MarkPrice: 0.7664, OpenInterest: 246502.09001551, ClampMinimum: 0.0005, ClampMaximum: 0.0025}})
	testTupleDecode(t, `["pos",158793100,1678465351520,null,"tAMPF0:USTF0",1e-08,1.0093,null,0,1,null,1.0098]`,
		Liquidation{Type: "pos", PositionID: 158793100, Time: types.Time(time.UnixMilli(1678465351520)), Symbol: "tAMPF0:USTF0", Amount: 1e-08, BasePrice: 1.0093, IsMarketSold: true, PriceAcquired: 1.0098})
	// GET liquidations/hist, which sends each liquidation in an array of its own
	testTupleDecode(t, `[[["pos",158793100,1678465351520,null,"tAMPF0:USTF0",1e-8,1.0093,null,0,1,null,1.0098]]]`,
		liquidationsResponse{{Type: "pos", PositionID: 158793100, Time: types.Time(time.UnixMilli(1678465351520)), Symbol: "tAMPF0:USTF0", Amount: 1e-08, BasePrice: 1.0093, IsMarketSold: true, PriceAcquired: 1.0098}})
	testTupleDecode(t, `["tMKRF0:USTF0",643.57,null,648.82,null,null,null,null,null,null,null,null,1674061204000]`,
		HistoricTicker{Symbol: "tMKRF0:USTF0", Bid: 643.57, Ask: 648.82, Time: types.Time(time.UnixMilli(1674061204000))})
	testTupleDecode(t, `[1573554000000,25957.94278561]`,
		Stat{Time: types.Time(time.UnixMilli(1573554000000)), Value: 25957.94278561})
	testTupleDecode(t, `[1575201600000,null,"USERNAME",1,null,null,15504971.51,null,0,"TWITTER_HANDLE"]`,
		LeaderboardStanding{Time: types.Time(time.UnixMilli(1575201600000)), Username: "USERNAME", Ranking: 1, Value: 15504971.51, TwitterHandle: "TWITTER_HANDLE"})
	testTupleDecode(t, `[1678467900000,null,null,0.00000187,68.88,null,null,3230660371.4429536,2988430006.4623938,null,null,1447938775.5708046]`,
		FundingStatistic{Time: types.Time(time.UnixMilli(1678467900000)), FlashReturnRate: 0.00000187, AveragePeriod: 68.88, FundingAmount: 3230660371.4429536, FundingAmountUsed: 2988430006.4623938, FundingBelowThreshold: 1447938775.5708046})

	// GET conf/pub:{action}:{object}:{detail}, whose entries the configs send in a list alone in an array
	testTupleDecode(t, `["DSH","DASH"]`,
		CurrencySymbol{Currency: currency.NewCode("DSH"), Symbol: "DASH"})
	testTupleDecode(t, `["IOT","Mi|MegaIOTA"]`,
		CurrencyUnit{Currency: currency.IOT, Unit: "Mi|MegaIOTA"})
	testTupleDecode(t, `["BTCF0","BTC"]`,
		CurrencyUnderlying{Currency: currency.NewCode("BTCF0"), Underlying: currency.BTC})
	testTupleDecode(t, `["AIO",["https://mainnet.aion.network","https://mainnet.aion.network/#/account/VAL","https://mainnet.aion.network/#/transaction/VAL"]]`,
		CurrencyExplorer{Currency: currency.NewCode("AIO"), Explorer: BlockExplorer{URL: "https://mainnet.aion.network", AddressURL: "https://mainnet.aion.network/#/account/VAL", TransactionURL: "https://mainnet.aion.network/#/transaction/VAL"}})
	testTupleDecode(t, `["BTC",[0,0.0004]]`,
		CurrencyWithdrawalFee{Currency: currency.BTC, Fee: WithdrawalFee{Amount: 0.0004}})
	testTupleDecode(t, `["WIRE",["USD","GBP","HKD","EUR","SGD","TWD","CHF","JPY","CNH"]]`,
		TransferMethod{Method: "WIRE", Currencies: []currency.Code{currency.USD, currency.GBP, currency.HKD, currency.EUR, currency.SGD, currency.TWD, currency.CHF, currency.JPY, currency.NewCode("CNH")}})
	testTupleDecode(t, `["1INCH:USD",[null,null,null,"4.0","100000.0",null,null,null,null,null,null,null]]`,
		PairInfoEntry{Pair: "1INCH:USD", Info: PairInfo{MinimumOrderSize: 4, MaximumOrderSize: 100000}})
	testTupleDecode(t, `["AAVEF0:USTF0",[null,null,null,"0.02","5000.0",null,null,null,0.01,0.005]]`,
		PairInfoEntry{Pair: "AAVEF0:USTF0", Info: PairInfo{MinimumOrderSize: 0.02, MaximumOrderSize: 5000, InitialMargin: 0.01, MinimumMargin: 0.005}})
	// The docs' example ends before the deposit confirmations
	testTupleDecode(t, `["RIPPLE",1,1,null,null,null,null,1,1]`,
		TransferMethodStatus{Method: "RIPPLE", DepositActive: true, WithdrawalActive: true, DepositPaymentIDAllowed: true, WithdrawalPaymentIDAllowed: true})

	// POST calc/trade/avg, for a trading pair and for a funding currency, and POST calc/fx
	testTupleDecode(t, `[24187.614862350845,1.123]`,
		MarketAveragePriceResponse{AveragePrice: 24187.614862350845, Amount: 1.123})
	testTupleDecode(t, `[0.00033317434,1.123]`,
		FundingMarketAveragePriceResponse{AverageRate: 0.00033317434, Amount: 1.123})
	testTupleDecode(t, `[8668.65]`,
		ForeignExchangeRateResponse{CurrentRate: 8668.65})
}
