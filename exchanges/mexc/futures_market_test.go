package mexc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// futuresWindow returns a fixed window of futures market history. Mock responses are matched on the full request
// URL, so the window must not move, and the venue serves candles of any age
func futuresWindow() (start, end time.Time) {
	return time.Unix(1792800000, 0), time.Unix(1792821600, 0)
}

func TestGetFuturesServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesServerTime(t.Context())
	require.NoError(t, err, "GetFuturesServerTime must not error")
	assert.False(t, result.IsZero(), "GetFuturesServerTime should return the venue's time")
}

func TestGetFuturesContracts(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesContracts(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesContracts must not error")
	require.Len(t, result, 1, "a single contract must be returned for a symbol")
	assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")
	assert.Positive(t, result[0].ContractSize, "a contract should carry its size")
	assert.False(t, result[0].IsInverse(), "BTC_USDT should be a linear contract")

	result, err = e.GetFuturesContracts(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesContracts must not error")
	assert.NotEmpty(t, result, "every contract should be returned without a symbol")
}

func TestGetFuturesTransferableCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesTransferableCurrencies(t.Context())
	require.NoError(t, err, "GetFuturesTransferableCurrencies must not error")
	assert.NotEmpty(t, result, "GetFuturesTransferableCurrencies should return currencies")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesOrderbook(t.Context(), futuresTradablePair, 5)
	require.NoError(t, err, "GetFuturesOrderbook must not error")
	require.NotEmpty(t, result.Bids, "the book must have bids")
	require.NotEmpty(t, result.Asks, "the book must have asks")
	assert.Less(t, result.Bids[0].Price, result.Asks[0].Price, "the best bid should be below the best ask")
	assert.Positive(t, result.Version, "the book should carry its version")
	assert.False(t, result.Timestamp.Time().IsZero(), "the book should carry its time")
}

func TestGetFuturesDepthSnapshots(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesDepthSnapshots(t.Context(), currency.EMPTYPAIR, 3)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesDepthSnapshots(t.Context(), futuresTradablePair, 0)
	require.ErrorIs(t, err, errInvalidLimit)
	_, err = e.GetFuturesDepthSnapshots(t.Context(), futuresTradablePair, futuresDepthSnapshotsLimit+1)
	require.ErrorIs(t, err, errInvalidLimit)

	result, err := e.GetFuturesDepthSnapshots(t.Context(), futuresTradablePair, 3)
	require.NoError(t, err, "GetFuturesDepthSnapshots must not error")
	require.NotEmpty(t, result, "GetFuturesDepthSnapshots must return changes")
	assert.Positive(t, result[0].Version, "a change should carry its version")
}

func TestGetFuturesIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesIndexPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesIndexPrice(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesIndexPrice must not error")
	assert.Equal(t, futuresTradablePair.String(), result.Symbol, "the contract asked for should be returned")
	assert.Positive(t, result.IndexPrice, "the index price should be positive")
}

func TestGetFuturesFairPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFairPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesFairPrice(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesFairPrice must not error")
	assert.Equal(t, futuresTradablePair.String(), result.Symbol, "the contract asked for should be returned")
	assert.Positive(t, result.FairPrice, "the fair price should be positive")
}

func TestGetFuturesFundingRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesFundingRates(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesFundingRates must not error")
	require.Len(t, result, 1, "a single rate must be returned for a symbol")
	assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")
	assert.Positive(t, result[0].CollectCycle, "the rate should carry its funding interval")
	assert.False(t, result[0].NextSettleTime.Time().IsZero(), "the rate should carry its settlement time")

	result, err = e.GetFuturesFundingRates(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesFundingRates must not error")
	assert.NotEmpty(t, result, "every contract's rate should be returned without a symbol")
}

func TestFuturesIntervals(t *testing.T) {
	t.Parallel()
	for _, i := range futuresIntervals {
		value, err := futuresIntervalString(i.interval)
		require.NoErrorf(t, err, "futuresIntervalString must not error for %s", i.interval)
		interval, err := futuresIntervalFromString(value)
		require.NoErrorf(t, err, "futuresIntervalFromString must not error for %s", value)
		assert.Equalf(t, i.interval, interval, "%s should round trip", value)
	}
	_, err := futuresIntervalString(kline.TenMin)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval)
	_, err = futuresIntervalString(kline.HundredMilliseconds)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "a spot websocket interval should not be a futures interval")
	_, err = futuresIntervalFromString("Min3")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval)
}

func TestGetFuturesCandles(t *testing.T) {
	t.Parallel()
	start, end := futuresWindow()
	for name, get := range map[string]func(*FuturesCandlesRequest) (*FuturesCandlesResponse, error){
		"trade": func(r *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
			return e.GetFuturesCandles(t.Context(), r)
		},
		"index": func(r *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
			return e.GetFuturesIndexPriceCandles(t.Context(), r)
		},
		"fair": func(r *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
			return e.GetFuturesFairPriceCandles(t.Context(), r)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := get(nil)
			require.ErrorIs(t, err, common.ErrNilPointer)
			_, err = get(&FuturesCandlesRequest{Interval: kline.OneHour})
			require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
			_, err = get(&FuturesCandlesRequest{Symbol: futuresTradablePair, Interval: kline.TenMin})
			require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
			_, err = get(&FuturesCandlesRequest{Symbol: futuresTradablePair, Interval: kline.OneHour, StartTime: end, EndTime: start})
			require.ErrorIs(t, err, common.ErrStartAfterEnd)

			result, err := get(&FuturesCandlesRequest{Symbol: futuresTradablePair, Interval: kline.OneHour, StartTime: start, EndTime: end})
			require.NoError(t, err, "the candle request must not error")
			require.NotEmpty(t, result.Time, "candles must be returned")
			assert.Len(t, result.Open, len(result.Time), "every candle should carry an open")
			assert.Len(t, result.Close, len(result.Time), "every candle should carry a close")
			assert.False(t, result.Time[0].Time().Before(start), "no candle should open before the window")
			assert.False(t, result.Time[len(result.Time)-1].Time().After(end), "no candle should open after the window")
		})
	}
}

func TestGetFuturesRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRecentTrades(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesRecentTrades(t.Context(), futuresTradablePair, futuresRecentTradesLimit+1)
	require.ErrorIs(t, err, errInvalidLimit)

	result, err := e.GetFuturesRecentTrades(t.Context(), futuresTradablePair, 5)
	require.NoError(t, err, "GetFuturesRecentTrades must not error")
	require.NotEmpty(t, result, "trades must be returned")
	assert.NotEmpty(t, result[0].TradeID, "a trade should carry its ID")
	assert.Positive(t, result[0].Price, "a trade should carry its price")
	assert.Positive(t, result[0].Volume, "a trade should carry its volume")
}

func TestGetFuturesTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesTickers(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesTickers must not error")
	require.Len(t, result, 1, "a single ticker must be returned for a symbol")
	assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")
	assert.Positive(t, result[0].LastPrice, "the ticker should carry the last price")
	require.NotNil(t, result[0].PriceChangeRates, "the ticker must carry its price changes")
	assert.NotEmpty(t, result[0].PriceChangeRates.Zone, "the price changes should name their time zone")

	result, err = e.GetFuturesTickers(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesTickers must not error")
	assert.NotEmpty(t, result, "every contract's ticker should be returned without a symbol")
}

func TestGetFuturesInsuranceFundBalances(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesInsuranceFundBalances(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesInsuranceFundBalances must not error")
	require.Len(t, result, 1, "the venue answers a symbol with one balance, though its docs show a list")
	assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")

	result, err = e.GetFuturesInsuranceFundBalances(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesInsuranceFundBalances must not error")
	assert.NotEmpty(t, result, "every contract's balance should be returned without a symbol")
}

func TestGetFuturesInsuranceFundBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesInsuranceFundBalanceHistory(t.Context(), currency.EMPTYPAIR, nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesInsuranceFundBalanceHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageSize: futuresDefaultPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)

	result, err := e.GetFuturesInsuranceFundBalanceHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageNumber: 1, PageSize: 2})
	require.NoError(t, err, "GetFuturesInsuranceFundBalanceHistory must not error")
	require.NotEmpty(t, result.ResultList, "snapshots must be returned")
	assert.EqualValues(t, 1, result.CurrentPage, "the first page should be returned")
	assert.False(t, result.ResultList[0].SnapshotTime.Time().IsZero(), "a snapshot should carry its time")
}

func TestGetFuturesFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFundingRateHistory(t.Context(), currency.EMPTYPAIR, nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesFundingRateHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageSize: futuresFundingRateHistoryPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)

	result, err := e.GetFuturesFundingRateHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageNumber: 1, PageSize: 2})
	require.NoError(t, err, "GetFuturesFundingRateHistory must not error")
	require.Len(t, result.ResultList, 2, "the page size asked for must be returned")
	assert.True(t, result.ResultList[0].SettleTime.Time().After(result.ResultList[1].SettleTime.Time()), "the rates should be newest first")
}

func TestFuturesIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for input, exp := range map[string]FuturesID{
		`"739106551624717312"`: "739106551624717312",
		`739106551624717312`:   "739106551624717312",
		`18446744073709551615`: "18446744073709551615",
		`null`:                 "",
		`""`:                   "",
		`0`:                    "",
		`"0"`:                  "",
	} {
		var id FuturesID
		require.NoErrorf(t, json.Unmarshal([]byte(input), &id), "Unmarshal must not error for %s", input)
		assert.Equalf(t, exp, id, "%s should decode exactly", input)
	}
	for _, input := range []string{`-1`, `1.5`, `true`, `{}`} {
		var id FuturesID
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &id), errInvalidFuturesID, "%s should be refused", input)
	}
}

func TestOneOrManyUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var one oneOrMany[*FuturesIndexPriceResponse]
	require.NoError(t, json.Unmarshal([]byte(`{"symbol":"BTC_USDT","indexPrice":1.5}`), &one), "Unmarshal must not error for an object")
	require.Len(t, one, 1, "an object must decode as one item")
	assert.Equal(t, 1.5, one[0].IndexPrice, "the object's fields should be decoded")

	var many oneOrMany[*FuturesIndexPriceResponse]
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"A_USDT"},{"symbol":"B_USDT"}]`), &many), "Unmarshal must not error for a list")
	assert.Len(t, many, 2, "a list should decode as its items")
}

func TestPagedListUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var page pagedList[*FuturesFundingFeeResponse]
	require.NoError(t, json.Unmarshal([]byte(`{"pageSize":2,"totalCount":1,"totalPage":1,"currentPage":1,"resultList":[{"id":7}]}`), &page), "Unmarshal must not error for a page")
	require.Len(t, page, 1, "a page must decode as its result list")
	assert.EqualValues(t, 7, page[0].ID, "the result list's fields should be decoded")

	var list pagedList[*FuturesFundingFeeResponse]
	require.NoError(t, json.Unmarshal([]byte(`[{"id":8},{"id":9}]`), &list), "Unmarshal must not error for a list")
	assert.Len(t, list, 2, "a list should decode as its items")
}

func TestFuturesOrderbookLevelsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var levels FuturesOrderbookLevels
	require.NoError(t, json.Unmarshal([]byte(`[[80654.3,4963,1],[80654.4,7584]]`), &levels), "Unmarshal must not error")
	require.Len(t, levels, 2, "every level must be decoded")
	assert.Equal(t, 80654.3, levels[0].Price, "the price should be decoded")
	assert.Equal(t, 4963.0, levels[0].Amount, "the contracts should be decoded")
	assert.EqualValues(t, 1, levels[0].OrderCount, "the order count should be decoded")
	assert.Zero(t, levels[1].OrderCount, "a level without an order count should have none")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[[80654.3]]`), &levels), errInvalidOrderbookLevel, "a level without its contracts should be refused")
}
