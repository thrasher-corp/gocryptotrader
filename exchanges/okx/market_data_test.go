package okx

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTickers must reject a nil request")
	_, err = e.GetTickers(t.Context(), &TickersRequest{InstrumentFamily: "BTC-USD"})
	require.ErrorIs(t, err, errInvalidInstrumentType, "GetTickers must require an instrument type")

	testexch.UpdatePairsOnce(t, e)
	pairs, err := e.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, pairs, "GetAvailablePairs must not return empty pairs")
	instFamily, err := e.instrumentFamilyFromInstID(instTypeOption, pairs[0].String())
	require.NoError(t, err, "instrumentFamilyFromInstID must not error")

	result, err := e.GetTickers(contextGenerate(), &TickersRequest{InstrumentType: instTypeOption, InstrumentFamily: instFamily})
	require.NoError(t, err, "GetTickers must not error")
	if mockTests {
		exp := []TickerResponse{
			{
				InstrumentType:           "OPTION",
				InstrumentID:             currency.NewPairWithDelimiter("SOL", "USD_UM-261125-66-C", "-"),
				BestAskPrice:             5.73,
				BestAskSize:              248,
				BestBidPrice:             5.37,
				BestBidSize:              248,
				TickerDataGenerationTime: types.Time(time.UnixMilli(1795406755108)),
			},
			{
				InstrumentType:           "OPTION",
				InstrumentID:             currency.NewPairWithDelimiter("SOL", "USD_UM-261127-80-C", "-"),
				LastTradePrice:           0.68,
				LastTradeSize:            250,
				BestAskPrice:             0.37,
				BestAskSize:              4998,
				BestBidPrice:             0.24,
				BestBidSize:              4998,
				OpenPrice24Hour:          0.68,
				HighestPrice24Hour:       0.68,
				LowestPrice24Hour:        0.68,
				OpenPriceUTC0:            0.68,
				OpenPriceUTC8:            0.68,
				TickerDataGenerationTime: types.Time(time.UnixMilli(1795406755103)),
			},
		}
		assert.Equal(t, exp, result, "GetTickers should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetTickers should return the family's tickers")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetTicker must require an instrument ID")

	result, err := e.GetTicker(contextGenerate(), perpetualSwapPair.String())
	require.NoError(t, err, "GetTicker must not error")
	require.NotNil(t, result, "GetTicker must return a ticker")
	if mockTests {
		exp := &TickerResponse{
			InstrumentType:                "SWAP",
			InstrumentID:                  currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-"),
			LastTradePrice:                61207.8,
			LastTradeSize:                 0.3,
			BestAskPrice:                  61207.9,
			BestAskSize:                   276.89,
			BestBidPrice:                  61207.8,
			BestBidSize:                   786.51,
			OpenPrice24Hour:               59472.6,
			HighestPrice24Hour:            61943,
			LowestPrice24Hour:             58731,
			TradingVolume24HourInCurrency: 124103.2083,
			TradingVolume24HourInContract: 12410320.83,
			OpenPriceUTC0:                 61240,
			OpenPriceUTC8:                 60857.8,
			TickerDataGenerationTime:      types.Time(time.UnixMilli(1795406755412)),
		}
		assert.Equal(t, exp, result, "GetTicker should decode every field")
	} else {
		assert.Positive(t, result.LastTradePrice.Float64(), "GetTicker should return a last traded price")
	}
}

func TestOrderBookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("bare sequence ID", func(t *testing.T) {
		t.Parallel()
		var ob OrderBookResponse
		require.NoError(t, json.Unmarshal([]byte(`{"asks":[["42000","0.1","0","3"]],"bids":[],"seqId":3235851742,"ts":"1724751378980"}`), &ob), "the books row must decode")
		exp := OrderBookResponse{
			Asks:       []OrderBookLevel{{Price: 42000, Size: 0.1, OrderCount: 3}},
			Bids:       []OrderBookLevel{},
			SequenceID: uint64(3235851742),
			Timestamp:  types.Time(time.UnixMilli(1724751378980)),
		}
		assert.Equal(t, exp, ob, "the books row should decode with its bare seqId integer")
	})
	t.Run("levels", func(t *testing.T) {
		t.Parallel()
		var ob OrderBookResponse
		err := json.Unmarshal([]byte(`{"asks":[["74611.6","0.50582211","0","13"],["74612.5","0.10514904","0","2"]],"bids":[["74611.5","0.012","0","1"]],"ts":"1787276524807"}`), &ob)
		require.NoError(t, err, "Unmarshal must not error")
		require.Len(t, ob.Asks, 2, "Asks must decode")
		require.Len(t, ob.Bids, 1, "Bids must decode")
		assert.Equal(t, 74611.6, ob.Asks[0].Price.Float64(), "ask price should decode")
		assert.Equal(t, 74611.5, ob.Bids[0].Price.Float64(), "bid price should decode")
		assert.Equal(t, int64(1787276524807), ob.Timestamp.Time().UnixMilli(), "Timestamp should decode")
	})
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBookDepth(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBookDepth must reject a nil request")
	_, err = e.GetOrderBookDepth(t.Context(), &OrderBookRequest{Depth: 400})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetOrderBookDepth must require an instrument ID")

	result, err := e.GetOrderBookDepth(contextGenerate(), &OrderBookRequest{InstrumentID: mainPair.String(), Depth: 400})
	require.NoError(t, err, "GetOrderBookDepth must not error")
	require.NotNil(t, result, "GetOrderBookDepth must return an order book")
	if mockTests {
		exp := &OrderBookResponse{
			Asks: []OrderBookLevel{
				{
					Price:      61256.8,
					Size:       0.46269468,
					OrderCount: 7,
				},
				{
					Price:      61256.9,
					Size:       0.21597674,
					OrderCount: 1,
				},
				{
					Price:      61258.7,
					Size:       0.00320868,
					OrderCount: 1,
				},
				{
					Price:      61258.8,
					Size:       0.0000102,
					OrderCount: 1,
				},
			},
			Bids: []OrderBookLevel{
				{
					Price:      61256.7,
					Size:       0.22362398,
					OrderCount: 4,
				},
				{
					Price:      61256.6,
					Size:       0.28443965,
					OrderCount: 2,
				},
				{
					Price:      61256.5,
					Size:       0.23084313,
					OrderCount: 1,
				},
				{
					Price:      61253.5,
					Size:       0.40074797,
					OrderCount: 1,
				},
			},
			Timestamp:  types.Time(time.UnixMilli(1795406739809)),
			SequenceID: 84751328091,
		}
		assert.Equal(t, exp, result, "GetOrderBookDepth should decode every field")
	} else {
		assert.NotEmpty(t, result.Asks, "GetOrderBookDepth should return asks")
		assert.NotEmpty(t, result.Bids, "GetOrderBookDepth should return bids")
	}
}

func TestGetRPIOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetRPIOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetRPIOrderBook must reject a nil request")
	_, err = e.GetRPIOrderBook(t.Context(), &OrderBookRequest{Depth: 2})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetRPIOrderBook must require an instrument ID")

	result, err := e.GetRPIOrderBook(contextGenerate(), &OrderBookRequest{InstrumentID: perpetualSwapPair.String(), Depth: 2})
	require.NoError(t, err, "GetRPIOrderBook must not error")
	require.NotNil(t, result, "GetRPIOrderBook must return an order book")
	if mockTests {
		exp := &RPIOrderBookResponse{
			Asks: []RPIOrderBookLevel{
				{
					Price:      61230.3,
					TotalSize:  226.52,
					NonRPISize: 226.52,
					OrderCount: 33,
				},
				{
					Price:      61230.4,
					TotalSize:  0.05,
					NonRPISize: 0.05,
					OrderCount: 3,
				},
			},
			Bids: []RPIOrderBookLevel{
				{
					Price:      61230.2,
					TotalSize:  290.67,
					NonRPISize: 290.67,
					OrderCount: 31,
				},
				{
					Price:      61230.1,
					TotalSize:  0.07,
					NonRPISize: 0.07,
					OrderCount: 4,
				},
			},
			Timestamp:  types.Time(time.UnixMilli(1795406742009)),
			SequenceID: 356679512470,
		}
		assert.Equal(t, exp, result, "GetRPIOrderBook should decode every field")
	} else {
		assert.NotEmpty(t, result.Asks, "GetRPIOrderBook should return asks")
		assert.NotEmpty(t, result.Bids, "GetRPIOrderBook should return bids")
	}
}

func TestGetFullOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFullOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFullOrderBook must reject a nil request")
	_, err = e.GetFullOrderBook(t.Context(), &OrderBookRequest{Depth: 2})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetFullOrderBook must require an instrument ID")

	result, err := e.GetFullOrderBook(contextGenerate(), &OrderBookRequest{InstrumentID: mainPair.String(), Depth: 2})
	require.NoError(t, err, "GetFullOrderBook must not error")
	require.NotNil(t, result, "GetFullOrderBook must return an order book")
	if mockTests {
		exp := &FullOrderBookResponse{
			Asks: []FullOrderBookLevel{
				{
					Price:      61256.8,
					Size:       0.32351,
					OrderCount: 5,
				},
				{
					Price:      61256.9,
					Size:       0.21597674,
					OrderCount: 1,
				},
			},
			Bids: []FullOrderBookLevel{
				{
					Price:      61256.7,
					Size:       0.24262398,
					OrderCount: 5,
				},
				{
					Price:      61256.6,
					Size:       0.28443965,
					OrderCount: 2,
				},
			},
			Timestamp:  types.Time(time.UnixMilli(1795406741009)),
			SequenceID: 84751328532,
		}
		assert.Equal(t, exp, result, "GetFullOrderBook should decode every field")
	} else {
		assert.NotEmpty(t, result.Asks, "GetFullOrderBook should return asks")
		assert.NotEmpty(t, result.Bids, "GetFullOrderBook should return bids")
	}
}

func TestOrderBookParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OrderBookRequest
		exp  url.Values
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &OrderBookRequest{Depth: 5}, err: errMissingInstrumentID},
		{name: "instrument only", arg: &OrderBookRequest{InstrumentID: mainPair.String()}, exp: url.Values{"instId": {"BTC-USDT"}}},
		{name: "depth", arg: &OrderBookRequest{InstrumentID: mainPair.String(), Depth: 400}, exp: url.Values{"instId": {"BTC-USDT"}, "sz": {"400"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := orderBookParams(tc.arg)
			require.ErrorIs(t, err, tc.err, "orderBookParams must return the expected error")
			assert.Equal(t, tc.exp, params, "orderBookParams should encode the instrument and any depth")
		})
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CandlesticksRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &CandlesticksRequest{Bar: kline.OneHour, Limit: 2}, err: errMissingInstrumentID},
		{name: "start after end", arg: &CandlesticksRequest{InstrumentID: mainPair.String(), Before: time.UnixMilli(1795402800000), After: time.UnixMilli(1795392000000)}, err: common.ErrStartAfterEnd},
		{name: "unsupported bar", arg: &CandlesticksRequest{InstrumentID: mainPair.String(), Bar: kline.TenMin}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCandlesticks(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetCandlesticks should reject an invalid request")
		})
	}

	arg := &CandlesticksRequest{InstrumentID: mainPair.String(), Bar: kline.OneHour, Before: time.UnixMilli(1795392000000), After: time.UnixMilli(1795402800000), Limit: 2}
	if !mockTests {
		arg.After = time.Now()
		arg.Before = arg.After.Add(-3 * time.Hour)
	}
	result, err := e.GetCandlesticks(contextGenerate(), arg)
	require.NoError(t, err, "GetCandlesticks must not error")
	if mockTests {
		exp := []CandlestickResponse{
			{
				OpenTime:            types.Time(time.UnixMilli(1795399200000)),
				OpenPrice:           60869.3,
				HighestPrice:        60918.2,
				LowestPrice:         60380.8,
				ClosePrice:          60611,
				Volume:              511.5680745,
				VolumeCurrency:      31055216.215715157,
				VolumeQuoteCurrency: 31055216.215715157,
				Confirmed:           true,
			},
			{
				OpenTime:            types.Time(time.UnixMilli(1795395600000)),
				OpenPrice:           61343.2,
				HighestPrice:        61356,
				LowestPrice:         60798,
				ClosePrice:          60869.3,
				Volume:              387.38473782,
				VolumeCurrency:      23618053.762056758,
				VolumeQuoteCurrency: 23618053.762056758,
				Confirmed:           true,
			},
		}
		assert.Equal(t, exp, result, "GetCandlesticks should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetCandlesticks should return candlesticks")
	}

	t.Run("getCandlesticks sends every parameter to the caller's path", func(t *testing.T) {
		t.Parallel()
		ex := newQueryPinExchange(t, "/market/history-candles",
			url.Values{"instId": {"BTC-USDT-SWAP"}, "bar": {"1H"}, "after": {"1795402800000"}, "before": {"1795392000000"}, "limit": {"2"}, "adjust": {"forward"}},
			[][]string{{"1795399200000", "60869.3", "60918.2", "60380.8", "60611", "511.5680745", "31055216.215715157", "31055216.215715157", "1"}})
		result, err := ex.getCandlesticks(t.Context(), "market/history-candles", getCandlestickHistoryEPL, &CandlesticksRequest{
			InstrumentID: perpetualSwapPair.String(), Bar: kline.OneHour, After: time.UnixMilli(1795402800000), Before: time.UnixMilli(1795392000000), Limit: 2, Adjust: "forward",
		})
		require.NoError(t, err, "getCandlesticks must not error")
		exp := []CandlestickResponse{{
			OpenTime: types.Time(time.UnixMilli(1795399200000)), OpenPrice: 60869.3, HighestPrice: 60918.2, LowestPrice: 60380.8, ClosePrice: 60611,
			Volume: 511.5680745, VolumeCurrency: 31055216.215715157, VolumeQuoteCurrency: 31055216.215715157, Confirmed: true,
		}}
		assert.Equal(t, exp, result, "getCandlesticks should decode every candlestick")
	})
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCandlesticksHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCandlesticksHistory must reject a nil request")
	_, err = e.GetCandlesticksHistory(t.Context(), &CandlesticksRequest{Bar: kline.OneHour, Limit: 3})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetCandlesticksHistory must require an instrument ID")

	result, err := e.GetCandlesticksHistory(contextGenerate(), &CandlesticksRequest{InstrumentID: mainPair.String(), Bar: kline.OneHour, Before: time.UnixMilli(1793491200000), After: time.UnixMilli(1793502000000), Limit: 3})
	require.NoError(t, err, "GetCandlesticksHistory must not error")
	if mockTests {
		exp := []CandlestickResponse{
			{
				OpenTime:            types.Time(time.UnixMilli(1793498400000)),
				OpenPrice:           77693.5,
				HighestPrice:        77849.1,
				LowestPrice:         77627.4,
				ClosePrice:          77700.2,
				Volume:              104.1499066,
				VolumeCurrency:      8094916.132596957,
				VolumeQuoteCurrency: 8094916.132596957,
				Confirmed:           true,
			},
			{
				OpenTime:            types.Time(time.UnixMilli(1793494800000)),
				OpenPrice:           77641.4,
				HighestPrice:        77734.9,
				LowestPrice:         77611.9,
				ClosePrice:          77693.3,
				Volume:              124.13457521,
				VolumeCurrency:      9644419.342725701,
				VolumeQuoteCurrency: 9644419.342725701,
				Confirmed:           true,
			},
		}
		assert.Equal(t, exp, result, "GetCandlesticksHistory should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetCandlesticksHistory should return candlesticks")
	}
}

func TestCandlestickParams(t *testing.T) {
	t.Parallel()
	older, newer := time.UnixMilli(1795392000000), time.UnixMilli(1795402800000)
	for _, tc := range []struct {
		name          string
		instrumentID  string
		bar           kline.Interval
		after, before time.Time
		limit         uint64
		exp           url.Values
		err           error
	}{
		{name: "missing instrument ID", bar: kline.OneHour, err: errMissingInstrumentID},
		{name: "before bound newer than the after bound", instrumentID: mainPair.String(), after: older, before: newer, err: common.ErrStartAfterEnd},
		{name: "unsupported bar", instrumentID: mainPair.String(), bar: kline.TenMin, err: kline.ErrUnsupportedInterval},
		{name: "instrument only", instrumentID: mainPair.String(), exp: url.Values{"instId": {"BTC-USDT"}}},
		{name: "hourly bar", instrumentID: mainPair.String(), bar: kline.OneHour, exp: url.Values{"instId": {"BTC-USDT"}, "bar": {"1H"}}},
		{name: "daily bar opens at UTC", instrumentID: mainPair.String(), bar: kline.OneDay, exp: url.Values{"instId": {"BTC-USDT"}, "bar": {"1Dutc"}}},
		{name: "after bound only", instrumentID: mainPair.String(), after: newer, exp: url.Values{"instId": {"BTC-USDT"}, "after": {"1795402800000"}}},
		{name: "before bound only", instrumentID: mainPair.String(), before: older, exp: url.Values{"instId": {"BTC-USDT"}, "before": {"1795392000000"}}},
		{name: "equal bounds", instrumentID: mainPair.String(), after: older, before: older, exp: url.Values{"instId": {"BTC-USDT"}, "after": {"1795392000000"}, "before": {"1795392000000"}}},
		{name: "limit", instrumentID: mainPair.String(), limit: 100, exp: url.Values{"instId": {"BTC-USDT"}, "limit": {"100"}}},
		{
			name: "every parameter", instrumentID: mainPair.String(), bar: kline.FiveMin, after: newer, before: older, limit: 2,
			exp: url.Values{"instId": {"BTC-USDT"}, "bar": {"5m"}, "after": {"1795402800000"}, "before": {"1795392000000"}, "limit": {"2"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := candlestickParams(tc.instrumentID, tc.bar, tc.after, tc.before, tc.limit)
			require.ErrorIs(t, err, tc.err, "candlestickParams must return the expected error")
			assert.Equal(t, tc.exp, params, "candlestickParams should encode every set parameter")
		})
	}
}

func TestIntervalFromString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Description string
		Interval    kline.Interval
		Expected    string
		AppendUTC   bool
	}{
		{Description: "zero interval", Expected: "", AppendUTC: true},
		{Description: "kline.ThousandMilliseconds", Interval: kline.ThousandMilliseconds, Expected: "1s"},
		{Description: "kline.OneMin", Interval: kline.OneMin, Expected: "1m"},
		{Description: "kline.ThreeMin", Interval: kline.ThreeMin, Expected: "3m"},
		{Description: "kline.FiveMin", Interval: kline.FiveMin, Expected: "5m"},
		{Description: "kline.FifteenMin", Interval: kline.FifteenMin, Expected: "15m"},
		{Description: "kline.ThirtyMin", Interval: kline.ThirtyMin, Expected: "30m"},
		{Description: "kline.OneHour", Interval: kline.OneHour, Expected: "1H"},
		{Description: "kline.TwoHour", Interval: kline.TwoHour, Expected: "2H"},
		{Description: "kline.FourHour", Interval: kline.FourHour, Expected: "4H"},
		{Description: "kline.EightHour", Interval: kline.EightHour, Expected: "8H"},
		{Description: "kline.EightHour + UTC", Interval: kline.EightHour, Expected: "8H", AppendUTC: true},
		{Description: "kline.SixHour", Interval: kline.SixHour, Expected: "6H"},
		{Description: "kline.TwelveHour", Interval: kline.TwelveHour, Expected: "12H"},
		{Description: "kline.OneDay", Interval: kline.OneDay, Expected: "1D"},
		{Description: "kline.TwoDay", Interval: kline.TwoDay, Expected: "2D"},
		{Description: "kline.ThreeDay", Interval: kline.ThreeDay, Expected: "3D"},
		{Description: "kline.OneWeek", Interval: kline.OneWeek, Expected: "1W"},
		{Description: "kline.FiveDay", Interval: kline.FiveDay, Expected: "5D"},
		{Description: "kline.OneMonth", Interval: kline.OneMonth, Expected: "1M"},
		{Description: "kline.ThreeMonth", Interval: kline.ThreeMonth, Expected: "3M"},
		{Description: "kline.SixMonth", Interval: kline.SixMonth, Expected: "6M"},
		{Description: "kline.OneYear", Interval: kline.OneYear, Expected: "1Y"},
		{Description: "kline.SixHour + UTC", Interval: kline.SixHour, Expected: "6Hutc", AppendUTC: true},
		{Description: "kline.TwelveHour + UTC", Interval: kline.TwelveHour, Expected: "12Hutc", AppendUTC: true},
		{Description: "kline.OneDay + UTC", Interval: kline.OneDay, Expected: "1Dutc", AppendUTC: true},
		{Description: "kline.TwoDay + UTC", Interval: kline.TwoDay, Expected: "2Dutc", AppendUTC: true},
		{Description: "kline.ThreeDay + UTC", Interval: kline.ThreeDay, Expected: "3Dutc", AppendUTC: true},
		{Description: "kline.FiveDay + UTC", Interval: kline.FiveDay, Expected: "5Dutc", AppendUTC: true},
		{Description: "kline.OneWeek + UTC", Interval: kline.OneWeek, Expected: "1Wutc", AppendUTC: true},
		{Description: "kline.OneMonth + UTC", Interval: kline.OneMonth, Expected: "1Mutc", AppendUTC: true},
		{Description: "kline.ThreeMonth + UTC", Interval: kline.ThreeMonth, Expected: "3Mutc", AppendUTC: true},
		{Description: "kline.SixMonth + UTC", Interval: kline.SixMonth, Expected: "6Mutc", AppendUTC: true},
		{Description: "kline.OneYear + UTC", Interval: kline.OneYear, Expected: "1Yutc", AppendUTC: true},
		{Description: "unsupported interval", Interval: kline.TenMin, Expected: "", AppendUTC: true},
	}
	for _, tt := range tests {
		t.Run(tt.Description, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.Expected, IntervalFromString(tt.Interval, tt.AppendUTC), "IntervalFromString should return the expected bar")
		})
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTrades must reject a nil request")
	_, err = e.GetTrades(t.Context(), &TradesRequest{Limit: 3})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetTrades must require an instrument ID")

	result, err := e.GetTrades(contextGenerate(), &TradesRequest{InstrumentID: mainPair.String(), Limit: 3})
	require.NoError(t, err, "GetTrades must not error")
	if mockTests {
		exp := []TradeResponse{
			{
				InstrumentID: "BTC-USDT",
				TradeID:      "1089308333",
				Price:        61232.7,
				Size:         0.00001,
				Side:         order.Buy,
				Source:       "0",
				Timestamp:    types.Time(time.UnixMilli(1795406755975)),
			},
			{
				InstrumentID: "BTC-USDT",
				TradeID:      "1089308332",
				Price:        61231.4,
				Size:         0.00001,
				Side:         order.Buy,
				Source:       "0",
				Timestamp:    types.Time(time.UnixMilli(1795406755520)),
			},
			{
				InstrumentID: "BTC-USDT",
				TradeID:      "1089308331",
				Price:        61231.4,
				Size:         0.00029999,
				Side:         order.Buy,
				Source:       "0",
				Timestamp:    types.Time(time.UnixMilli(1795406755043)),
			},
		}
		assert.Equal(t, exp, result, "GetTrades should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetTrades should return trades")
	}
}

func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradesHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradesHistory must reject a nil request")
	_, err = e.GetTradesHistory(t.Context(), &TradesHistoryRequest{Limit: 2})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetTradesHistory must require an instrument ID")

	t.Run("latest", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetTradesHistory(contextGenerate(), &TradesHistoryRequest{InstrumentID: mainPair.String(), Limit: 2})
		require.NoError(t, err, "GetTradesHistory must not error")
		if mockTests {
			exp := []TradeResponse{
				{
					InstrumentID: "BTC-USDT",
					TradeID:      "1089308301",
					Price:        61247.8,
					Size:         0.0000102,
					Side:         order.Sell,
					Source:       "0",
					Timestamp:    types.Time(time.UnixMilli(1795406748909)),
				},
				{
					InstrumentID: "BTC-USDT",
					TradeID:      "1089308300",
					Price:        61249.7,
					Size:         0.0000102,
					Side:         order.Sell,
					Source:       "0",
					Timestamp:    types.Time(time.UnixMilli(1795406748909)),
				},
			}
			assert.Equal(t, exp, result, "GetTradesHistory should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetTradesHistory should return trades")
		}
	})
	t.Run("paginated by timestamp", func(t *testing.T) {
		t.Parallel()
		after := "1795392000000"
		if !mockTests {
			after = strconv.FormatInt(time.Now().Add(-time.Hour).UnixMilli(), 10)
		}
		result, err := e.GetTradesHistory(contextGenerate(), &TradesHistoryRequest{InstrumentID: mainPair.String(), PaginationType: "2", After: after, Limit: 2})
		require.NoError(t, err, "GetTradesHistory must not error")
		if mockTests {
			exp := []TradeResponse{
				{
					InstrumentID: "BTC-USDT",
					TradeID:      "1089197701",
					Price:        61268.8,
					Size:         0.0000102,
					Side:         order.Sell,
					Source:       "0",
					Timestamp:    types.Time(time.UnixMilli(1795391998732)),
				},
				{
					InstrumentID: "BTC-USDT",
					TradeID:      "1089197700",
					Price:        61269.5,
					Size:         0.018,
					Side:         order.Sell,
					Source:       "0",
					Timestamp:    types.Time(time.UnixMilli(1795391998731)),
				},
			}
			assert.Equal(t, exp, result, "GetTradesHistory should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetTradesHistory should return trades")
		}
	})
}

func TestGetOptionTradesByInstrumentFamily(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionTradesByInstrumentFamily(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentFamilyRequired, "GetOptionTradesByInstrumentFamily must require an instrument family")

	result, err := e.GetOptionTradesByInstrumentFamily(contextGenerate(), optionsPair.String())
	require.NoError(t, err, "GetOptionTradesByInstrumentFamily must not error")
	if mockTests {
		exp := []OptionFamilyTradesResponse{
			{
				Volume24Hour: 1036664,
				OptionType:   "C",
				TradeInfo: []OptionFamilyTrade{
					{
						InstrumentID: "BTC-USD-261225-78000-C",
						TradeID:      "414",
						Price:        0.003,
						Size:         500,
						Side:         order.Sell,
						Timestamp:    types.Time(time.UnixMilli(1795406653757)),
					},
					{
						InstrumentID: "BTC-USD-270129-88000-C",
						TradeID:      "467",
						Price:        0.0037,
						Size:         3,
						Side:         order.Buy,
						Timestamp:    types.Time(time.UnixMilli(1795406652569)),
					},
				},
			},
			{
				Volume24Hour: 1124227,
				OptionType:   "P",
				TradeInfo: []OptionFamilyTrade{
					{
						InstrumentID: "BTC-USD-270924-60000-P",
						TradeID:      "97",
						Price:        0.1355,
						Size:         106,
						Side:         order.Sell,
						Timestamp:    types.Time(time.UnixMilli(1795406644934)),
					},
					{
						InstrumentID: "BTC-USD-261204-51000-P",
						TradeID:      "52",
						Price:        0.0034,
						Size:         569,
						Side:         order.Sell,
						Timestamp:    types.Time(time.UnixMilli(1795406631773)),
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetOptionTradesByInstrumentFamily should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetOptionTradesByInstrumentFamily should return trades")
	}
}

func TestGetOptionTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOptionTrades must reject a nil request")
	_, err = e.GetOptionTrades(t.Context(), &OptionTradesRequest{OptionType: "C"})
	require.ErrorIs(t, err, errInstrumentIDOrFamilyRequired, "GetOptionTrades must require an instrument ID or family")

	result, err := e.GetOptionTrades(contextGenerate(), &OptionTradesRequest{InstrumentFamily: optionsPair.String(), OptionType: "C"})
	require.NoError(t, err, "GetOptionTrades must not error")
	if mockTests {
		exp := []OptionTradeResponse{
			{
				InstrumentID:     "BTC-USD-261123-62000-C",
				InstrumentFamily: "BTC-USD",
				TradeID:          "425",
				Price:            0.0004,
				Size:             15,
				Side:             order.Sell,
				OptionType:       "C",
				FillVolatility:   0.4882902734375,
				ForwardPrice:     61135.49860166181,
				IndexPrice:       61127.6,
				MarkPrice:        0.0004236326566336,
				Timestamp:        types.Time(time.UnixMilli(1795406810056)),
			},
			{
				InstrumentID:     "BTC-USD-261123-62000-C",
				InstrumentFamily: "BTC-USD",
				TradeID:          "424",
				Price:            0.0004,
				Size:             19,
				Side:             order.Sell,
				OptionType:       "C",
				FillVolatility:   0.4882902734375,
				ForwardPrice:     61138.09860166181,
				IndexPrice:       61130.6,
				MarkPrice:        0.0004286873587287,
				Timestamp:        types.Time(time.UnixMilli(1795406808569)),
			},
		}
		assert.Equal(t, exp, result, "GetOptionTrades should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetOptionTrades should return trades")
	}
}

func TestGet24HourTotalVolume(t *testing.T) {
	t.Parallel()
	result, err := e.Get24HourTotalVolume(contextGenerate())
	require.NoError(t, err, "Get24HourTotalVolume must not error")
	require.NotNil(t, result, "Get24HourTotalVolume must return the volume")
	if mockTests {
		exp := &TotalVolume24HourResponse{
			VolumeUSD: 21612974422,
			VolumeCNY: 144028861548,
			Timestamp: types.Time(time.UnixMilli(1795406520105)),
		}
		assert.Equal(t, exp, result, "Get24HourTotalVolume should decode every field")
	} else {
		assert.Positive(t, result.VolumeUSD.Float64(), "Get24HourTotalVolume should return a USD volume")
	}
}

func TestGetCallAuctionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetCallAuctionDetails(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetCallAuctionDetails must require an instrument ID")

	// CT-USDT listed through a call auction, so its recorded details carry matched and unmatched sizes
	instrumentID := "CT-USDT"
	if !mockTests {
		instrumentID = mainPair.String()
	}
	result, err := e.GetCallAuctionDetails(contextGenerate(), instrumentID)
	require.NoError(t, err, "GetCallAuctionDetails must not error")
	require.NotNil(t, result, "GetCallAuctionDetails must return details")
	if mockTests {
		exp := &CallAuctionDetailsResponse{
			InstrumentID:     "CT-USDT",
			EquilibriumPrice: 0.292,
			MatchedSize:      274349.3,
			UnmatchedSize:    20541915.8,
			AuctionEndTime:   types.Time(time.UnixMilli(1790762400000)),
			State:            "continuous_trading",
			Timestamp:        types.Time(time.UnixMilli(1790762400008)),
		}
		assert.Equal(t, exp, result, "GetCallAuctionDetails should decode every field")
	} else {
		assert.Equal(t, instrumentID, result.InstrumentID, "GetCallAuctionDetails should return the instrument's details")
	}
}
