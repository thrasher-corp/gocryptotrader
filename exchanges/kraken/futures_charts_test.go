package kraken

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFuturesTickTypes(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesTickTypes(t.Context())
	require.NoError(t, err, "GetFuturesTickTypes must not error")
	if mockTests {
		assert.Equal(t, []string{"mark", "spot", "trade"}, result, "GetFuturesTickTypes should decode every tick type")
		return
	}
	assert.Contains(t, result, "trade", "GetFuturesTickTypes should list the trade tick type")
}

func TestGetFuturesChartMarkets(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesChartMarkets(t.Context(), "")
	require.ErrorIs(t, err, errTickTypeEmpty, "GetFuturesChartMarkets must reject an empty tick type")

	result, err := e.GetFuturesChartMarkets(t.Context(), "trade")
	require.NoError(t, err, "GetFuturesChartMarkets must not error")
	if mockTests {
		assert.Equal(t, []string{"PF_XBTUSD", "PI_XBTUSD", "FF_XBTUSD_261225"}, result, "GetFuturesChartMarkets should decode every market")
		return
	}
	assert.Contains(t, result, "PF_XBTUSD", "GetFuturesChartMarkets should list PF_XBTUSD")
}

func TestGetFuturesChartResolutions(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesChartResolutions(t.Context(), "", futuresTestPair)
	require.ErrorIs(t, err, errTickTypeEmpty, "GetFuturesChartResolutions must reject an empty tick type")
	_, err = e.GetFuturesChartResolutions(t.Context(), "trade", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesChartResolutions must reject an empty pair")

	result, err := e.GetFuturesChartResolutions(t.Context(), "trade", futuresTestPair)
	require.NoError(t, err, "GetFuturesChartResolutions must not error")
	supported := slices.Collect(maps.Values(futuresChartIntervals))
	for _, resolution := range result {
		assert.Containsf(t, supported, resolution, "futuresChartIntervals should map an interval to resolution %s", resolution)
	}
	if mockTests {
		assert.Equal(t, []string{"15m", "1d", "30m", "4h", "12h", "1h", "1m", "5m", "1w"}, result, "GetFuturesChartResolutions should decode every resolution")
		return
	}
	assert.Contains(t, result, "1h", "GetFuturesChartResolutions should list the hourly resolution")
}

func TestGetFuturesCandles(t *testing.T) {
	t.Parallel()
	from := time.Unix(1791489600, 0)
	to := time.Unix(1791500400, 0)
	_, err := e.GetFuturesCandles(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesCandles must reject a nil request")
	_, err = e.GetFuturesCandles(t.Context(), &FuturesCandlesRequest{Pair: futuresTestPair, Interval: kline.OneHour})
	require.ErrorIs(t, err, errTickTypeEmpty, "GetFuturesCandles must reject an empty tick type")
	_, err = e.GetFuturesCandles(t.Context(), &FuturesCandlesRequest{TickType: "trade", Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesCandles must reject an empty pair")
	_, err = e.GetFuturesCandles(t.Context(), &FuturesCandlesRequest{TickType: "trade", Pair: futuresTestPair, Interval: kline.TwoHour})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetFuturesCandles must reject an unsupported interval")
	_, err = e.GetFuturesCandles(t.Context(), &FuturesCandlesRequest{TickType: "trade", Pair: futuresTestPair, Interval: kline.OneHour, From: to, To: from})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesCandles must reject a reversed range")

	result, err := e.GetFuturesCandles(t.Context(), &FuturesCandlesRequest{TickType: "trade", Pair: futuresTestPair, Interval: kline.OneHour})
	require.NoError(t, err, "GetFuturesCandles must not error")
	if mockTests {
		exp := &FuturesCandlesResponse{
			Candles: []FuturesCandle{
				{Time: types.Time(time.UnixMilli(1791500400000)), Open: 81840, High: 81886, Low: 81639, Close: 81688, Volume: 56.1316},
				{Time: types.Time(time.UnixMilli(1791504000000)), Open: 81688, High: 81830, Low: 81640, Close: 81738, Volume: 60.7758},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesCandles should decode every field")
	} else {
		assert.NotEmpty(t, result.Candles, "GetFuturesCandles should return candles")
	}

	result, err = e.GetFuturesCandles(t.Context(), &FuturesCandlesRequest{TickType: "trade", Pair: futuresTestPair, Interval: kline.OneHour, From: from, To: to, Count: 2})
	require.NoError(t, err, "GetFuturesCandles must not error with every parameter")
	if mockTests {
		exp := &FuturesCandlesResponse{
			Candles: []FuturesCandle{
				{Time: types.Time(time.UnixMilli(1791489600000)), Open: 81726, High: 81849, Low: 81640, Close: 81773, Volume: 195.9643},
				{Time: types.Time(time.UnixMilli(1791493200000)), Open: 81773, High: 81780, Low: 81561, Close: 81652, Volume: 82.449},
			},
			MoreCandles: true,
		}
		assert.Equal(t, exp, result, "GetFuturesCandles should decode every field with every parameter")
		return
	}
	require.Len(t, result.Candles, 2, "GetFuturesCandles must return the requested number of candles")
	assert.True(t, from.Equal(result.Candles[0].Time.Time()), "GetFuturesCandles should start at From")
	assert.True(t, result.MoreCandles, "GetFuturesCandles should report the range's remaining candles")
}

func TestGetFuturesLiquidityPoolStatistic(t *testing.T) {
	t.Parallel()
	since := time.Unix(1791493200, 0)
	_, err := e.GetFuturesLiquidityPoolStatistic(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesLiquidityPoolStatistic must reject a nil request")
	_, err = e.GetFuturesLiquidityPoolStatistic(t.Context(), &FuturesLiquidityPoolStatisticRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFuturesLiquidityPoolStatistic must reject a missing since")
	_, err = e.GetFuturesLiquidityPoolStatistic(t.Context(), &FuturesLiquidityPoolStatisticRequest{Since: since, Interval: kline.TwoHour})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetFuturesLiquidityPoolStatistic must reject an unsupported interval")
	_, err = e.GetFuturesLiquidityPoolStatistic(t.Context(), &FuturesLiquidityPoolStatisticRequest{Since: since, To: since.Add(-time.Hour), Interval: kline.OneHour})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesLiquidityPoolStatistic must reject a reversed range")

	result, err := e.GetFuturesLiquidityPoolStatistic(t.Context(), &FuturesLiquidityPoolStatisticRequest{Since: since, Interval: kline.OneHour})
	require.NoError(t, err, "GetFuturesLiquidityPoolStatistic must not error")
	if mockTests {
		exp := &FuturesAnalyticsResponse{
			Timestamps: []types.Time{types.Time(since), types.Time(time.Unix(1791496800, 0)), types.Time(time.Unix(1791500400, 0))},
			Data:       FuturesAnalyticsData{USDValue: []types.Number{28898863.81, 28901245.17, 28903377.02}},
			More:       true,
		}
		assert.Equal(t, exp, result, "GetFuturesLiquidityPoolStatistic should decode every field")
	} else {
		assert.NotEmpty(t, result.Data.USDValue, "GetFuturesLiquidityPoolStatistic should return the pool's value")
	}

	to := time.Unix(1791496800, 0)
	result, err = e.GetFuturesLiquidityPoolStatistic(t.Context(), &FuturesLiquidityPoolStatisticRequest{Since: since, To: to, Interval: kline.OneHour})
	require.NoError(t, err, "GetFuturesLiquidityPoolStatistic must not error with every parameter")
	if mockTests {
		exp := &FuturesAnalyticsResponse{
			Timestamps: []types.Time{types.Time(since), types.Time(to)},
			Data:       FuturesAnalyticsData{USDValue: []types.Number{28898863.81, 28901245.17}},
		}
		assert.Equal(t, exp, result, "GetFuturesLiquidityPoolStatistic should decode every field with every parameter")
		return
	}
	assert.LessOrEqual(t, len(result.Timestamps), 2, "GetFuturesLiquidityPoolStatistic should return at most the requested buckets")
}

func TestGetFuturesMarketAnalytics(t *testing.T) {
	t.Parallel()
	since := time.Unix(1791493200, 0)
	to := time.Unix(1791496800, 0)
	_, err := e.GetFuturesMarketAnalytics(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesMarketAnalytics must reject a nil request")
	_, err = e.GetFuturesMarketAnalytics(t.Context(), &FuturesMarketAnalyticsRequest{AnalyticsType: "open-interest", Since: since, Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesMarketAnalytics must reject an empty pair")
	_, err = e.GetFuturesMarketAnalytics(t.Context(), &FuturesMarketAnalyticsRequest{Pair: futuresTestPair, Since: since, Interval: kline.OneHour})
	require.ErrorIs(t, err, errAnalyticsTypeEmpty, "GetFuturesMarketAnalytics must reject an empty analytics type")
	_, err = e.GetFuturesMarketAnalytics(t.Context(), &FuturesMarketAnalyticsRequest{Pair: futuresTestPair, AnalyticsType: "open-interest", Interval: kline.OneHour})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFuturesMarketAnalytics must reject a missing since")
	_, err = e.GetFuturesMarketAnalytics(t.Context(), &FuturesMarketAnalyticsRequest{Pair: futuresTestPair, AnalyticsType: "open-interest", Since: since, Interval: kline.TwoHour})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetFuturesMarketAnalytics must reject an unsupported interval")
	_, err = e.GetFuturesMarketAnalytics(t.Context(), &FuturesMarketAnalyticsRequest{Pair: futuresTestPair, AnalyticsType: "open-interest", Since: to, To: since, Interval: kline.OneHour})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesMarketAnalytics must reject a reversed range")

	bid := FuturesAnalyticsBookSide{
		BestPrice:           []types.Number{81681, 81802},
		Liquidity005:        []types.Number{517.6019, 438.4528},
		Liquidity01:         []types.Number{611.8567, 533.7758},
		Liquidity025:        []types.Number{749.1745, 664.0817},
		Liquidity05:         []types.Number{782.7155, 704.2306},
		Liquidity10:         []types.Number{953.1215, 873.34},
		Liquidity100:        []types.Number{1435.5449, 1354.9931},
		Slippage1Thousand:   []types.Number{81680.95, 81801.97},
		Slippage10Thousand:  []types.Number{81680.102445837, 81801.3423600281},
		Slippage100Thousand: []types.Number{81674.9071907578, 81795.8204016833},
		Slippage1Million:    []types.Number{81669.2786121538, 0},
	}
	ask := FuturesAnalyticsBookSide{
		BestPrice:           []types.Number{81682, 81803},
		Liquidity005:        []types.Number{516.624, 360.9437},
		Liquidity01:         []types.Number{580.676, 494.2428},
		Liquidity025:        []types.Number{697.3085, 589.3401},
		Liquidity05:         []types.Number{729.4568, 619.2972},
		Liquidity10:         []types.Number{866.9697, 755.9774},
		Liquidity100:        []types.Number{908.9167, 797.8923},
		Slippage1Thousand:   []types.Number{81682.04, 81803.03},
		Slippage10Thousand:  []types.Number{81686.1836215012, 81804.8529673064},
		Slippage100Thousand: []types.Number{81694.3047614819, 81815.3889436772},
		Slippage1Million:    []types.Number{81703.5509287372, 81823.7621736731},
	}
	liquidity := func(s FuturesAnalyticsBookSide) FuturesAnalyticsBookSide {
		return FuturesAnalyticsBookSide{
			Liquidity005: s.Liquidity005, Liquidity01: s.Liquidity01, Liquidity025: s.Liquidity025,
			Liquidity05: s.Liquidity05, Liquidity10: s.Liquidity10, Liquidity100: s.Liquidity100,
		}
	}
	slippage := func(s FuturesAnalyticsBookSide) FuturesAnalyticsBookSide {
		return FuturesAnalyticsBookSide{
			Slippage1Thousand: s.Slippage1Thousand, Slippage10Thousand: s.Slippage10Thousand,
			Slippage100Thousand: s.Slippage100Thousand, Slippage1Million: s.Slippage1Million,
		}
	}
	for _, tc := range []struct {
		analyticsType string
		exp           FuturesAnalyticsData
	}{
		{
			analyticsType: "open-interest",
			exp: FuturesAnalyticsData{OHLC: []FuturesAnalyticsOHLC{
				{Open: 2346.7189, High: 2362.41, Low: 2346.505, Close: 2360.6244},
				{Open: 2360.6244, High: 2370.435, Low: 2358.0432, Close: 2370.1928},
			}},
		},
		{analyticsType: "trade-volume", exp: FuturesAnalyticsData{Values: []types.Number{82.449, 83.9653}}},
		{analyticsType: "trade-count", exp: FuturesAnalyticsData{Values: []types.Number{5204, 4866}}},
		{
			analyticsType: "long-short-info",
			exp: FuturesAnalyticsData{
				LongCount:    []uint64{3663, 3672},
				ShortCount:   []uint64{1829, 1824},
				LongPercent:  []types.Number{66.69, 66.81},
				ShortPercent: []types.Number{33.31, 33.19},
				Ratio:        []types.Number{0.66, 0.67},
			},
		},
		{
			analyticsType: "cvd",
			exp: FuturesAnalyticsData{
				BuyVolume:             []types.Number{49.5412, 58.9854},
				SellVolume:            []types.Number{32.9078, 24.9799},
				CumulativeVolumeDelta: []types.Number{16.6334, 50.6389},
			},
		},
		{
			analyticsType: "top-traders",
			exp: FuturesAnalyticsData{Top20Percent: FuturesAnalyticsTopTraders{
				OpenInterest: []types.Number{2263.2416, 2271.542},
				LongCount:    []uint64{683, 688},
				ShortCount:   []uint64{415, 410},
				LongPercent:  []types.Number{62.2, 62.65},
				ShortPercent: []types.Number{37.8, 37.35},
				Ratio:        []types.Number{0.62, 0.63},
			}},
		},
		{analyticsType: "orderbook", exp: FuturesAnalyticsData{Bid: bid, Ask: ask}},
		{
			analyticsType: "spreads",
			exp:           FuturesAnalyticsData{Bid: FuturesAnalyticsBookSide{BestPrice: bid.BestPrice}, Ask: FuturesAnalyticsBookSide{BestPrice: ask.BestPrice}},
		},
		{analyticsType: "liquidity", exp: FuturesAnalyticsData{Bid: liquidity(bid), Ask: liquidity(ask)}},
		{analyticsType: "slippage", exp: FuturesAnalyticsData{Bid: slippage(bid), Ask: slippage(ask)}},
		{analyticsType: "future-basis", exp: FuturesAnalyticsData{Basis: []types.Number{0.0001, 0.0002}}},
		{
			analyticsType: "funding",
			exp: FuturesAnalyticsData{
				FundingRate: []FuturesAnalyticsOHLC{
					{Open: 1.4873250817605, High: 1.764903990041861, Low: 1.4801, Close: 1.7512},
					{Open: 1.7512, High: 1.7689, Low: 1.741186912535, Close: 1.7436},
				},
				RelativeFundingRate: []FuturesAnalyticsOHLC{
					{Open: 0.00001820255, High: 0.000021587529166667, Low: 0.0000181, Close: 0.0000214},
					{Open: 0.0000214, High: 0.0000216, Low: 0.00002133005, Close: 0.00002134},
				},
			},
		},
	} {
		t.Run(tc.analyticsType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesMarketAnalytics(t.Context(), &FuturesMarketAnalyticsRequest{Pair: futuresTestPair, AnalyticsType: tc.analyticsType, Since: since, To: to, Interval: kline.OneHour})
			require.NoError(t, err, "GetFuturesMarketAnalytics must not error")
			if mockTests {
				exp := &FuturesAnalyticsResponse{Timestamps: []types.Time{types.Time(since), types.Time(to)}, Data: tc.exp}
				assert.Equal(t, exp, result, "GetFuturesMarketAnalytics should decode every field")
				return
			}
			assert.LessOrEqual(t, len(result.Timestamps), 2, "GetFuturesMarketAnalytics should return at most the requested buckets")
		})
	}
}

func TestFuturesAnalytics(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/charts/v1/analytics/PF_XBTUSD/error":
			_, _ = w.Write([]byte(`{"result":null,"errors":[{"severity":"E","error_class":"General","type":"Invalid arguments","msg":null,"value":"61","field":"interval"},{"severity":"E","error_class":"General","type":"Unknown method","msg":"nope","value":null,"field":null}]}`))
		case "/api/charts/v1/analytics/PF_XBTUSD/warning":
			_, _ = w.Write([]byte(`{"result":{"timestamp":[1791493200],"data":["1.5"],"more":false},"errors":[{"severity":"W","error_class":"General","type":"Partial data","msg":null,"value":null,"field":"to"}]}`))
		case "/api/charts/v1/analytics/PF_XBTUSD/empty":
			_, _ = w.Write([]byte(`{"result":null,"errors":[]}`))
		case "/api/charts/v1/analytics/PF_XBTUSD/status":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"result":null,"errors":[{"severity":"E","error_class":"General","type":"Invalid arguments","msg":null,"value":null,"field":"interval"}]}`))
		}
	})
	analytics := func(analyticsType string) (*FuturesAnalyticsResponse, error) {
		return ex.GetFuturesMarketAnalytics(t.Context(), &FuturesMarketAnalyticsRequest{Pair: futuresTestPair, AnalyticsType: analyticsType, Since: time.Unix(1791493200, 0), Interval: kline.OneHour})
	}

	_, err := analytics("error")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "futuresAnalytics must return an APIError for the errors a response reports")
	assert.Equal(t, []string{"EGeneral:Invalid arguments:interval:61", "EGeneral:Unknown method:nope"}, apiErr.Errors, "APIError should hold each error formatted as Spot REST's")
	assert.ErrorIs(t, err, errAPIResponse, "futuresAnalytics should match errAPIResponse")

	result, err := analytics("warning")
	require.NoError(t, err, "futuresAnalytics must not error when only warned")
	exp := &FuturesAnalyticsResponse{Timestamps: []types.Time{types.Time(time.Unix(1791493200, 0))}, Data: FuturesAnalyticsData{Values: []types.Number{1.5}}}
	assert.Equal(t, exp, result, "futuresAnalytics should return the result sent with a warning")

	_, err = analytics("empty")
	assert.ErrorIs(t, err, common.ErrNoResponse, "futuresAnalytics should reject a null result")

	_, err = analytics("status")
	assert.ErrorIs(t, err, request.ErrBadStatus, "futuresAnalytics should return the status error of an error status")
}

func TestFuturesAnalyticsDataUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  FuturesAnalyticsData
	}{
		{name: "numbers and strings", data: `[1,"2.5"]`, exp: FuturesAnalyticsData{Values: []types.Number{1, 2.5}}},
		{name: "no buckets", data: `[]`, exp: FuturesAnalyticsData{Values: []types.Number{}}},
		{
			name: "documented cvd names",
			data: `{"buyVolume":["1.5"],"sellVolume":["2.5"],"cvd":["-1"]}`,
			exp:  FuturesAnalyticsData{BuyVolume: []types.Number{1.5}, SellVolume: []types.Number{2.5}, CumulativeVolumeDelta: []types.Number{-1}},
		},
		{name: "unknown series", data: `{"unknown":[1],"basis":["0.1"]}`, exp: FuturesAnalyticsData{Basis: []types.Number{0.1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var data FuturesAnalyticsData
			require.NoError(t, data.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error")
			assert.Equal(t, tc.exp, data, "UnmarshalJSON should decode the series")
		})
	}
	var data FuturesAnalyticsData
	assert.ErrorIs(t, data.UnmarshalJSON([]byte(`[["1","2"]]`)), errUnexpectedLength, "UnmarshalJSON should reject a short OHLC array")
	assert.Error(t, data.UnmarshalJSON([]byte(`{"basis":{}}`)), "UnmarshalJSON should reject a series that is not an array")
	assert.Error(t, data.UnmarshalJSON([]byte(`[`)), "UnmarshalJSON should reject malformed JSON")
	assert.Error(t, data.UnmarshalJSON([]byte(`"1"`)), "UnmarshalJSON should reject data that is neither an array nor an object")
}

func TestFuturesAnalyticsOHLCUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var ohlc FuturesAnalyticsOHLC
	require.NoError(t, ohlc.UnmarshalJSON([]byte(`["81683.8",81771.5,"81656.4","81723.4"]`)), "UnmarshalJSON must not error for numbers and strings")
	assert.Equal(t, FuturesAnalyticsOHLC{Open: 81683.8, High: 81771.5, Low: 81656.4, Close: 81723.4}, ohlc, "UnmarshalJSON should decode the open, high, low and close in order")
	assert.ErrorIs(t, ohlc.UnmarshalJSON([]byte(`["1","2","3"]`)), errUnexpectedLength, "UnmarshalJSON should reject a short array")
	assert.ErrorIs(t, ohlc.UnmarshalJSON([]byte(`["1","2","3","4","5"]`)), errUnexpectedLength, "UnmarshalJSON should reject a long array")
	assert.Error(t, ohlc.UnmarshalJSON([]byte(`{"open":"1"}`)), "UnmarshalJSON should reject an object")
}

func TestFuturesAnalyticsBookSideUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := FuturesAnalyticsBookSide{
		BestPrice:           []types.Number{1},
		Liquidity005:        []types.Number{2},
		Liquidity01:         []types.Number{3},
		Liquidity025:        []types.Number{4},
		Liquidity05:         []types.Number{5},
		Liquidity10:         []types.Number{6},
		Liquidity100:        []types.Number{7},
		Slippage1Thousand:   []types.Number{8},
		Slippage10Thousand:  []types.Number{9},
		Slippage100Thousand: []types.Number{10},
		Slippage1Million:    []types.Number{11},
	}
	for name, data := range map[string]string{
		"snake case": `{"best_price":["1"],"liquidity_005":["2"],"liquidity_01":["3"],"liquidity_025":["4"],"liquidity_05":["5"],"liquidity_10":["6"],"liquidity_100":["7"],"slippage_1k":["8"],"slippage_10k":["9"],"slippage_100k":["10"],"slippage_1m":["11"]}`,
		"camel case": `{"bestPrice":["1"],"liquidity005":["2"],"liquidity01":["3"],"liquidity025":["4"],"liquidity05":["5"],"liquidity10":["6"],"liquidity100":["7"],"slippage1k":["8"],"slippage10k":["9"],"slippage100k":["10"],"slippage1m":["11"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var side FuturesAnalyticsBookSide
			require.NoError(t, side.UnmarshalJSON([]byte(data)), "UnmarshalJSON must not error")
			assert.Equal(t, exp, side, "UnmarshalJSON should decode every series")
		})
	}
	var side FuturesAnalyticsBookSide
	assert.Error(t, side.UnmarshalJSON([]byte(`{"best_price":"1"}`)), "UnmarshalJSON should reject a series that is not an array")
}

func TestFuturesAnalyticsParams(t *testing.T) {
	t.Parallel()
	since, to := time.Unix(1791504000, 0), time.Unix(1791590400, 0)
	_, err := futuresAnalyticsParams(time.Time{}, to, kline.OneHour)
	assert.ErrorIs(t, err, common.ErrDateUnset, "futuresAnalyticsParams should reject an unset since")
	_, err = futuresAnalyticsParams(since, to, kline.TwoHour)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "futuresAnalyticsParams should reject an interval Kraken does not chart")
	_, err = futuresAnalyticsParams(to, since, kline.OneHour)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "futuresAnalyticsParams should reject a range that ends before it starts")
	params, err := futuresAnalyticsParams(since, to, kline.OneHour)
	require.NoError(t, err, "futuresAnalyticsParams must not error")
	assert.Equal(t, url.Values{"since": {"1791504000"}, "interval": {"3600"}, "to": {"1791590400"}}, params, "futuresAnalyticsParams should set since, the interval in seconds and to")
	params, err = futuresAnalyticsParams(since, time.Time{}, kline.OneDay)
	require.NoError(t, err, "futuresAnalyticsParams must not error without to")
	assert.Equal(t, url.Values{"since": {"1791504000"}, "interval": {"86400"}}, params, "futuresAnalyticsParams should leave out an unset to")
}

func TestFuturesAnalyticsEnvelope(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		var reply string
		switch {
		case strings.HasSuffix(r.URL.Path, "/warning"):
			reply = `{"result":{"timestamp":[1791504000],"data":[1],"more":true},"errors":[{"severity":"W","error_class":"General","type":"Deprecated","field":"interval"}]}`
		case strings.HasSuffix(r.URL.Path, "/error"):
			reply = `{"result":null,"errors":[{"severity":"W","error_class":"General","type":"Deprecated"},{"severity":"E","error_class":"General","type":"Invalid arguments","field":"interval"},{"severity":"E","error_class":"Query","type":"Unknown","field":"symbol","value":"PF_MOONUSD"}]}`
		default:
			reply = `{"errors":[]}`
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	})
	resp, err := ex.futuresAnalytics(t.Context(), "/charts/v1/analytics/warning")
	require.NoError(t, err, "futuresAnalytics must not error for a result with a warning")
	exp := &FuturesAnalyticsResponse{Timestamps: []types.Time{types.Time(time.Unix(1791504000, 0))}, Data: FuturesAnalyticsData{Values: []types.Number{1}}, More: true}
	assert.Equal(t, exp, resp, "futuresAnalytics should return the result beside a warning")

	_, err = ex.futuresAnalytics(t.Context(), "/charts/v1/analytics/error")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "futuresAnalytics must return the errors as an API error")
	assert.Equal(t, []string{"EGeneral:Invalid arguments:interval", "EQuery:Unknown:symbol:PF_MOONUSD"}, apiErr.Errors, "futuresAnalytics should report each error, leaving out warnings")

	_, err = ex.futuresAnalytics(t.Context(), "/charts/v1/analytics/empty")
	assert.ErrorIs(t, err, common.ErrNoResponse, "futuresAnalytics should reject a reply without a result")
}

func TestFuturesAnalyticsErrorMessage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		problem futuresAnalyticsError
		exp     string
	}{
		{problem: futuresAnalyticsError{Severity: "E", ErrorClass: "General", Type: "Invalid arguments"}, exp: "EGeneral:Invalid arguments"},
		{problem: futuresAnalyticsError{Severity: "E", ErrorClass: "General", Type: "Invalid arguments", Field: "interval"}, exp: "EGeneral:Invalid arguments:interval"},
		{problem: futuresAnalyticsError{Severity: "W", ErrorClass: "Query", Type: "Unknown", Field: "symbol", Message: "not listed", Value: "PF_MOONUSD"}, exp: "WQuery:Unknown:symbol:not listed:PF_MOONUSD"},
	} {
		assert.Equalf(t, tc.exp, tc.problem.message(), "message should format %+v as a Spot REST error", tc.problem)
	}
}
