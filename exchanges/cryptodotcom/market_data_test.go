package cryptodotcom

import (
	"net/http"
	"net/url"
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

// newQueryCheckExchange returns an exchange whose test server checks each request is a GET of the method with exactly
// the params expected, and answers it with result
func newQueryCheckExchange(t *testing.T, method string, exp url.Values, result string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "the request should be a GET")
		assert.Equal(t, "/exchange/v1/"+method, r.URL.Path, "the request should be for the method")
		assert.Equal(t, exp, r.URL.Query(), "the request should send only the params set")
		_, err := w.Write([]byte(`{"id":-1,"method":"` + method + `","code":0,"result":` + result + `}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

func TestGetRiskParameters(t *testing.T) {
	t.Parallel()
	result, err := e.GetRiskParameters(t.Context())
	require.NoError(t, err, "GetRiskParameters must not error")
	if mockTests {
		exp := &RiskParametersResponse{
			DefaultMaxProductLeverageForSpot:             1,
			DefaultMaxProductLeverageForPerpetuals:       20,
			DefaultMaxProductLeverageForFutures:          20,
			DefaultUnitMarginRateMultiplierForSpot:       1,
			DefaultUnitMarginRateMultiplierForPerpetuals: 1,
			DefaultUnitMarginRateMultiplierForFutures:    2,
			DefaultLongPositionLimitPerpetuals:           -1,
			DefaultShortPositionLimitPerpetuals:          -1,
			DefaultLongPositionLimitFutures:              -1,
			DefaultShortPositionLimitFutures:             -1,
			DefaultUnitMarginRate:                        0.05,
			UpdateTime:                                   types.Time(time.UnixMilli(1792728159827)),
			BaseCurrencyConfig: []BaseCurrencyRiskConfig{
				{
					InstrumentName:                  "BTC",
					CollateralCapNotional:           20000000,
					MinimumHaircut:                  0.0625,
					MaxProductLeverageForSpot:       16,
					MaxProductLeverageForPerpetuals: 50,
					UnitMarginRate:                  0.006,
					MaxShortSellLimit:               200,
					OrderLimit:                      1000000,
					MaxOrderNotionalUSD:             1000000,
					MinOrderNotionalUSD:             1,
					LongPositionLimitPerpetuals:     9700,
					ShortPositionLimitPerpetuals:    9700,
					LongPositionLimitFutures:        512,
					ShortPositionLimitFutures:       512,
				},
				{
					InstrumentName:                  "BASECAT",
					MaxProductLeverageForPerpetuals: 10,
					UnitMarginRate:                  0.0002,
					DailyNotionalLimit:              100000,
					OrderLimit:                      10000,
					MaxOrderNotionalUSD:             10000,
					MinOrderNotionalUSD:             1,
					LongPositionLimitPerpetuals:     25000000,
					ShortPositionLimitPerpetuals:    25000000,
				},
				{
					InstrumentName:        "USD_Stable_Coin",
					CollateralCapNotional: -1,
					MaxShortSellLimit:     4000000,
					OrderLimit:            5000000,
					MaxOrderNotionalUSD:   5000000,
					MinOrderNotionalUSD:   1,
				},
			},
			PerpetualSwapConfig: []BaseCurrencyOrderLimit{{InstrumentName: "1INCH", MaxOrderNotionalUSD: 500000}},
			FuturesConfig:       []BaseCurrencyOrderLimit{{InstrumentName: "1INCH", MaxOrderNotionalUSD: 2500000}},
		}
		assert.Equal(t, exp, result, "GetRiskParameters should decode every field")
	} else {
		assert.NotEmpty(t, result.BaseCurrencyConfig, "GetRiskParameters should return the base currencies' parameters")
	}

	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-risk-parameters","code":0,"result":null}`).GetRiskParameters(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetRiskParameters should error for a null result")
	_, err = newUnavailableServerExchange(t).GetRiskParameters(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetRiskParameters should return a request failure")
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	result, err := e.GetInstruments(t.Context())
	require.NoError(t, err, "GetInstruments must not error")
	if mockTests {
		exp := []InstrumentResponse{
			{
				Symbol:            "BTC_USD",
				InstrumentType:    InstrumentTypeCurrencyPair,
				DisplayName:       "BTC/USD",
				BaseCurrency:      currency.BTC,
				QuoteCurrency:     currency.USD,
				QuoteDecimals:     2,
				QuantityDecimals:  5,
				PriceTickSize:     0.01,
				QuantityTickSize:  0.00001,
				MaxLeverage:       50,
				Tradable:          true,
				ProductType:       ProductTypeDigitalCurrencies,
				MarginBuyEnabled:  true,
				MarginSellEnabled: true,
			},
			{
				Symbol:           "BTCUSD-PERP",
				InstrumentType:   InstrumentTypePerpetualSwap,
				DisplayName:      "BTCUSD Perpetual",
				BaseCurrency:     currency.BTC,
				QuoteCurrency:    currency.USD,
				QuoteDecimals:    1,
				QuantityDecimals: 4,
				PriceTickSize:    0.1,
				QuantityTickSize: 0.0001,
				MaxLeverage:      100,
				Tradable:         true,
				UnderlyingSymbol: "BTCUSD-INDEX",
				ProductType:      ProductTypeDigitalCurrencies,
				ContractSize:     1,
			},
			{
				Symbol:           "BTCUSD-261030",
				InstrumentType:   InstrumentTypeFuture,
				DisplayName:      "BTCUSD Futures 20261030",
				BaseCurrency:     currency.BTC,
				QuoteCurrency:    currency.USD,
				QuoteDecimals:    1,
				QuantityDecimals: 4,
				PriceTickSize:    0.1,
				QuantityTickSize: 0.0001,
				MaxLeverage:      100,
				Tradable:         true,
				ExpiryTime:       types.Time(time.UnixMilli(1793347200000)),
				UnderlyingSymbol: "BTCUSD-INDEX",
				ProductType:      ProductTypeDigitalCurrencies,
				ContractSize:     1,
			},
			{
				Symbol:            "SHIB_BTC@OEX_HK",
				InstrumentType:    InstrumentTypeCurrencyPair,
				DisplayName:       "SHIB/BTC",
				BaseCurrency:      currency.SHIB,
				QuoteCurrency:     currency.BTC,
				QuoteDecimals:     14,
				PriceTickSize:     0.00000000000001,
				QuantityTickSize:  10000,
				MaxLeverage:       50,
				Tradable:          true,
				MarginBuyEnabled:  true,
				MarginSellEnabled: true,
			},
		}
		assert.Equal(t, exp, result, "GetInstruments should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetInstruments should return instruments")
	}

	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-instruments","code":0,"result":null}`).GetInstruments(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetInstruments should error for a null result")
	_, err = newUnavailableServerExchange(t).GetInstruments(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetInstruments should return a request failure")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderbook must refuse a nil request")
	_, err = e.GetOrderbook(t.Context(), &OrderbookRequest{Depth: 3})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetOrderbook must refuse a request without an instrument name")

	for _, tc := range []struct {
		req *OrderbookRequest
		exp *OrderbookResponse
	}{
		{
			req: &OrderbookRequest{InstrumentName: "BTC_USD", Depth: 3},
			exp: &OrderbookResponse{
				Depth:          3,
				InstrumentName: "BTC_USD",
				Data: []OrderbookSnapshot{
					{
						Bids: []OrderbookLevel{
							{Price: 80815.15, Quantity: 0.47123, OrderCount: 11},
							{Price: 80814.77, Quantity: 0.03094, OrderCount: 1},
							{Price: 80814.46, Quantity: 0.0028, OrderCount: 1},
						},
						Asks: []OrderbookLevel{
							{Price: 80815.16, Quantity: 0.43305, OrderCount: 7},
							{Price: 80815.22, Quantity: 0.03094, OrderCount: 1},
							{Price: 80815.97, Quantity: 0.24, OrderCount: 2},
						},
						Timestamp: types.Time(time.UnixMilli(1792893935228)),
					},
				},
			},
		},
		{
			req: &OrderbookRequest{InstrumentName: "BTCUSD-PERP", Depth: 3},
			exp: &OrderbookResponse{
				Depth:          3,
				InstrumentName: "BTCUSD-PERP",
				Data: []OrderbookSnapshot{
					{
						Bids: []OrderbookLevel{
							{Price: 80817.5, Quantity: 0.0027, OrderCount: 2},
							{Price: 80816.4, Quantity: 0.0712, OrderCount: 2},
							{Price: 80816.3, Quantity: 0.0775, OrderCount: 1},
						},
						Asks: []OrderbookLevel{
							{Price: 80817.6, Quantity: 0.0301, OrderCount: 2},
							{Price: 80818.1, Quantity: 0.003, OrderCount: 1},
							{Price: 80821.6, Quantity: 0.0001, OrderCount: 1},
						},
						Timestamp: types.Time(time.UnixMilli(1792893936026)),
					},
				},
			},
		},
	} {
		result, err := e.GetOrderbook(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", tc.req.InstrumentName)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetOrderbook should decode every field for %s", tc.req.InstrumentName)
		} else {
			require.Lenf(t, result.Data, 1, "GetOrderbook must return one book for %s", tc.req.InstrumentName)
			assert.Lenf(t, result.Data[0].Bids, 3, "GetOrderbook should return the depth of bids requested for %s", tc.req.InstrumentName)
			assert.Lenf(t, result.Data[0].Asks, 3, "GetOrderbook should return the depth of asks requested for %s", tc.req.InstrumentName)
		}
	}

	ex := newQueryCheckExchange(t, "public/get-book", url.Values{"instrument_name": {"BTC_USD"}}, `{"depth":150,"data":[],"instrument_name":"BTC_USD"}`)
	_, err = ex.GetOrderbook(t.Context(), &OrderbookRequest{InstrumentName: "BTC_USD"})
	require.NoError(t, err, "GetOrderbook must not error without a depth")
	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-book","code":0,"result":null}`).GetOrderbook(t.Context(), &OrderbookRequest{InstrumentName: "BTC_USD"})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderbook should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOrderbook(t.Context(), &OrderbookRequest{InstrumentName: "BTC_USD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderbook should return a request failure")
}

func TestGetCandlestick(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	_, err := e.GetCandlestick(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCandlestick must refuse a nil request")
	_, err = e.GetCandlestick(t.Context(), &CandlestickRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetCandlestick must refuse a request without an instrument name")
	_, err = e.GetCandlestick(t.Context(), &CandlestickRequest{InstrumentName: "BTC_USD", Interval: kline.ThreeMin})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetCandlestick must refuse an interval Crypto.com has no timeframe for")
	_, err = e.GetCandlestick(t.Context(), &CandlestickRequest{InstrumentName: "BTC_USD", StartTime: start.Add(time.Hour), EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCandlestick must refuse a start after the end")
	_, err = e.GetCandlestick(t.Context(), &CandlestickRequest{InstrumentName: "BTC_USD", StartTime: time.UnixMilli(-1)})
	require.ErrorIs(t, err, errTimeBeforeUnixEpoch, "GetCandlestick must refuse a time before the Unix epoch")

	for _, tc := range []struct {
		req *CandlestickRequest
		exp *CandlestickResponse
	}{
		{
			req: &CandlestickRequest{InstrumentName: "BTC_USD", Interval: kline.OneHour, Count: 3, StartTime: start, EndTime: start.Add(3 * time.Hour)},
			exp: &CandlestickResponse{
				Interval:       "1h",
				InstrumentName: "BTC_USD",
				Data: []Candle{
					{Open: 81097, High: 81120.26, Low: 80508.11, Close: 80535.23, Volume: 202.44773, StartTime: types.Time(time.UnixMilli(1792800000000))},
					{Open: 80533.86, High: 80886.45, Low: 80524.08, Close: 80837.08, Volume: 138.48729, StartTime: types.Time(time.UnixMilli(1792803600000))},
					{Open: 80837.08, High: 80907.2, Low: 80744, Close: 80905.17, Volume: 55.85107, StartTime: types.Time(time.UnixMilli(1792807200000))},
				},
			},
		},
		{
			req: &CandlestickRequest{InstrumentName: "BTCUSD-PERP", Interval: kline.OneDay, Count: 2, EndTime: start},
			exp: &CandlestickResponse{
				Interval:       "1D",
				InstrumentName: "BTCUSD-PERP",
				Data: []Candle{
					{Open: 84030.9, High: 84307.8, Low: 80829.1, Close: 81322.9, Volume: 4043.4965, StartTime: types.Time(time.UnixMilli(1792627200000))},
					{Open: 81323.8, High: 81964.8, Low: 80413.9, Close: 81089.6, Volume: 2882.9591, StartTime: types.Time(time.UnixMilli(1792713600000))},
				},
			},
		},
	} {
		result, err := e.GetCandlestick(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetCandlestick must not error for %s", tc.req.InstrumentName)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetCandlestick should decode every field for %s", tc.req.InstrumentName)
		} else {
			assert.Lenf(t, result.Data, len(tc.exp.Data), "GetCandlestick should return the candles of the window for %s", tc.req.InstrumentName)
			assert.Equalf(t, tc.exp.Interval, result.Interval, "GetCandlestick should return the timeframe requested for %s", tc.req.InstrumentName)
		}
	}

	ex := newQueryCheckExchange(t, "public/get-candlestick", url.Values{"instrument_name": {"BTC_USD"}}, `{"interval":"1m","data":[],"instrument_name":"BTC_USD"}`)
	_, err = ex.GetCandlestick(t.Context(), &CandlestickRequest{InstrumentName: "BTC_USD"})
	require.NoError(t, err, "GetCandlestick must not error without optional params")
	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-candlestick","code":0,"result":null}`).GetCandlestick(t.Context(), &CandlestickRequest{InstrumentName: "BTC_USD"})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCandlestick should error for a null result")
	_, err = newUnavailableServerExchange(t).GetCandlestick(t.Context(), &CandlestickRequest{InstrumentName: "BTC_USD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCandlestick should return a request failure")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 25, 1, 46, 26, 0, time.UTC)
	_, err := e.GetTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTrades must refuse a nil request")
	_, err = e.GetTrades(t.Context(), &TradesRequest{Count: 3})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetTrades must refuse a request without an instrument name")
	_, err = e.GetTrades(t.Context(), &TradesRequest{InstrumentName: "BTC_USD", StartTime: start.Add(time.Second), EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTrades must refuse a start after the end")
	_, err = e.GetTrades(t.Context(), &TradesRequest{InstrumentName: "BTC_USD", EndTime: time.Unix(0, -1)})
	require.ErrorIs(t, err, errTimeBeforeUnixEpoch, "GetTrades must refuse a time before the Unix epoch")

	for _, tc := range []struct {
		req *TradesRequest
		exp []TradeResponse
	}{
		{
			// The window holds one taker order's three fills, which share a nanosecond time and a match ID
			req: &TradesRequest{InstrumentName: "BTC_USD", Count: 3, StartTime: start, EndTime: start.Add(time.Second)},
			exp: []TradeResponse{
				{TradeID: "1792892786513932606", TradeTime: types.Time(time.UnixMilli(1792892786513)), TradeTimeNano: types.Time(time.Unix(0, 1792892786513932604)), Quantity: 0.02992, Price: 80778, Side: "buy", InstrumentName: "BTC_USD", MatchID: "4611686018750829219"},
				{TradeID: "1792892786513932605", TradeTime: types.Time(time.UnixMilli(1792892786513)), TradeTimeNano: types.Time(time.Unix(0, 1792892786513932604)), Quantity: 0.00003, Price: 80778, Side: "buy", InstrumentName: "BTC_USD", MatchID: "4611686018750829219"},
				{TradeID: "1792892786513932604", TradeTime: types.Time(time.UnixMilli(1792892786513)), TradeTimeNano: types.Time(time.Unix(0, 1792892786513932604)), Quantity: 0.00005, Price: 80778, Side: "buy", InstrumentName: "BTC_USD", MatchID: "4611686018750829219"},
			},
		},
		{
			req: &TradesRequest{InstrumentName: "BTCUSD-PERP", Count: 3},
			exp: []TradeResponse{
				{TradeID: "1792893935765996075", TradeTime: types.Time(time.UnixMilli(1792893935765)), TradeTimeNano: types.Time(time.Unix(0, 1792893935765996075)), Quantity: 0.0025, Price: 80816.4, Side: "buy", InstrumentName: "BTCUSD-PERP", MatchID: "4611686018798094314"},
				{TradeID: "1792893935765994309", TradeTime: types.Time(time.UnixMilli(1792893935765)), TradeTimeNano: types.Time(time.Unix(0, 1792893935765994309)), Quantity: 0.0001, Price: 80815.4, Side: "buy", InstrumentName: "BTCUSD-PERP", MatchID: "4611686018798094313"},
				{TradeID: "1792893933495739253", TradeTime: types.Time(time.UnixMilli(1792893933495)), TradeTimeNano: types.Time(time.Unix(0, 1792893933495739252)), Quantity: 0.0001, Price: 80810.2, Side: "buy", InstrumentName: "BTCUSD-PERP", MatchID: "4611686018798094312"},
			},
		},
	} {
		result, err := e.GetTrades(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetTrades must not error for %s", tc.req.InstrumentName)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetTrades should decode every field for %s", tc.req.InstrumentName)
		} else {
			assert.Lenf(t, result, len(tc.exp), "GetTrades should return the trades requested for %s", tc.req.InstrumentName)
		}
	}

	ex := newQueryCheckExchange(t, "public/get-trades", url.Values{"instrument_name": {"BTC_USD"}}, `{"data":[]}`)
	_, err = ex.GetTrades(t.Context(), &TradesRequest{InstrumentName: "BTC_USD"})
	require.NoError(t, err, "GetTrades must not error without optional params")
	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-trades","code":0,"result":null}`).GetTrades(t.Context(), &TradesRequest{InstrumentName: "BTC_USD"})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTrades should error for a null result")
	_, err = newUnavailableServerExchange(t).GetTrades(t.Context(), &TradesRequest{InstrumentName: "BTC_USD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTrades should return a request failure")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTickers must refuse a nil request")

	for _, tc := range []struct {
		req *TickersRequest
		exp []TickerResponse
	}{
		{
			// The last ticker has no bids and no open interest, which Crypto.com sends as null
			req: &TickersRequest{},
			exp: []TickerResponse{
				{InstrumentName: "ACH_USD", High24Hour: 0.005229, Low24Hour: 0.004969, LastPrice: 0.005159, Volume24Hour: 479210, USDVolume24Hour: 2470.19, PriceChange24Hour: -0.0079, BestBidPrice: 0.005159, BestAskPrice: 0.005179, Timestamp: types.Time(time.UnixMilli(1792893939837))},
				{InstrumentName: "SHAZUSD-PERP", High24Hour: 23.09, Low24Hour: 22.38, LastPrice: 22.44, Volume24Hour: 16.48, USDVolume24Hour: 371.92, PriceChange24Hour: -0.0282, BestBidPrice: 22.76, BestAskPrice: 22.8, OpenInterest: 5.6, Timestamp: types.Time(time.UnixMilli(1792893900501))},
				{InstrumentName: "SHIB_BTC@OEX_HK", High24Hour: 0.000000000252, Low24Hour: 0.000000000252, LastPrice: 0.000000000252, BestAskPrice: 0.0000000003, Timestamp: types.Time(time.UnixMilli(1792893900131))},
			},
		},
		{
			req: &TickersRequest{InstrumentName: "BTC_USD"},
			exp: []TickerResponse{
				{InstrumentName: "BTC_USD", High24Hour: 81007.04, Low24Hour: 79128.21, LastPrice: 80815.22, Volume24Hour: 2201.6316, USDVolume24Hour: 176643650.4, PriceChange24Hour: -0.0002, BestBidPrice: 80816.57, BestAskPrice: 80816.58, Timestamp: types.Time(time.UnixMilli(1792893939611))},
			},
		},
		{
			req: &TickersRequest{InstrumentName: "BTCUSD-PERP"},
			exp: []TickerResponse{
				{InstrumentName: "BTCUSD-PERP", High24Hour: 81004.7, Low24Hour: 79117.7, LastPrice: 80816.4, Volume24Hour: 2497.9661, USDVolume24Hour: 200360781.94, PriceChange24Hour: -0.0001, BestBidPrice: 80818.1, BestAskPrice: 80818.2, OpenInterest: 6180.2299, Timestamp: types.Time(time.UnixMilli(1792893941098))},
			},
		},
	} {
		result, err := e.GetTickers(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetTickers must not error for %q", tc.req.InstrumentName)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetTickers should decode every field for %q", tc.req.InstrumentName)
			continue
		}
		if tc.req.InstrumentName == "" {
			assert.NotEmpty(t, result, "GetTickers should return every instrument's ticker")
			continue
		}
		require.Lenf(t, result, 1, "GetTickers must return one ticker for %s", tc.req.InstrumentName)
		assert.Equalf(t, tc.req.InstrumentName, result[0].InstrumentName, "GetTickers should return the ticker of %s", tc.req.InstrumentName)
	}

	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-tickers","code":0,"result":null}`).GetTickers(t.Context(), &TickersRequest{})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTickers should error for a null result")
	_, err = newUnavailableServerExchange(t).GetTickers(t.Context(), &TickersRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTickers should return a request failure")
}

func TestGetValuations(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	_, err := e.GetValuations(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetValuations must refuse a nil request")
	_, err = e.GetValuations(t.Context(), &ValuationsRequest{ValuationType: ValuationTypeIndexPrice})
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetValuations must refuse a request without an instrument name")
	_, err = e.GetValuations(t.Context(), &ValuationsRequest{InstrumentName: "BTCUSD-INDEX"})
	require.ErrorIs(t, err, errValuationTypeRequired, "GetValuations must refuse a request without a valuation type")
	_, err = e.GetValuations(t.Context(), &ValuationsRequest{InstrumentName: "BTCUSD-PERP", ValuationType: ValuationTypeFundingHistory, StartTime: start.Add(time.Hour), EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetValuations must refuse a start after the end")
	_, err = e.GetValuations(t.Context(), &ValuationsRequest{InstrumentName: "BTCUSD-PERP", ValuationType: ValuationTypeFundingHistory, StartTime: time.UnixMilli(-1)})
	require.ErrorIs(t, err, errTimeBeforeUnixEpoch, "GetValuations must refuse a time before the Unix epoch")

	for _, tc := range []struct {
		req *ValuationsRequest
		exp *ValuationsResponse
	}{
		{
			req: &ValuationsRequest{InstrumentName: "BTCUSD-INDEX", ValuationType: ValuationTypeIndexPrice, Count: 3},
			exp: &ValuationsResponse{
				InstrumentName: "BTCUSD-INDEX",
				Data: []Valuation{
					{Value: 80799, Timestamp: types.Time(time.UnixMilli(1792893906000))},
					{Value: 80796.52, Timestamp: types.Time(time.UnixMilli(1792893886000))},
					{Value: 80796.52, Timestamp: types.Time(time.UnixMilli(1792893885000))},
				},
			},
		},
		{
			req: &ValuationsRequest{InstrumentName: "BTC_USD", ValuationType: ValuationTypeMarkPrice, Count: 2},
			exp: &ValuationsResponse{
				InstrumentName: "BTC_USD",
				Data: []Valuation{
					{Value: 80727.6, Timestamp: types.Time(time.UnixMilli(1792894626000))},
					{Value: 80744.33, Timestamp: types.Time(time.UnixMilli(1792894605000))},
				},
			},
		},
		{
			req: &ValuationsRequest{InstrumentName: "BTCUSD-PERP", ValuationType: ValuationTypeFundingHistory, Count: 3, StartTime: start, EndTime: start.Add(3 * time.Hour)},
			exp: &ValuationsResponse{
				InstrumentName: "BTCUSD-PERP",
				Data: []Valuation{
					{Value: 0.000008003, Timestamp: types.Time(time.UnixMilli(1792807200000))},
					{Value: 0.000008003, Timestamp: types.Time(time.UnixMilli(1792803600000))},
					{Value: 0.000007063, Timestamp: types.Time(time.UnixMilli(1792800000000))},
				},
			},
		},
	} {
		result, err := e.GetValuations(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetValuations must not error for %s %s", tc.req.InstrumentName, tc.req.ValuationType)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetValuations should decode every field for %s %s", tc.req.InstrumentName, tc.req.ValuationType)
		} else {
			assert.Lenf(t, result.Data, len(tc.exp.Data), "GetValuations should return the values requested for %s %s", tc.req.InstrumentName, tc.req.ValuationType)
		}
	}

	ex := newQueryCheckExchange(t, "public/get-valuations", url.Values{"instrument_name": {"BTCUSD-PERP"}, "valuation_type": {ValuationTypeMarkPrice}}, `{"data":[],"instrument_name":"BTCUSD-PERP"}`)
	_, err = ex.GetValuations(t.Context(), &ValuationsRequest{InstrumentName: "BTCUSD-PERP", ValuationType: ValuationTypeMarkPrice})
	require.NoError(t, err, "GetValuations must not error without optional params")
	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-valuations","code":0,"result":null}`).GetValuations(t.Context(), &ValuationsRequest{InstrumentName: "BTCUSD-PERP", ValuationType: ValuationTypeMarkPrice})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetValuations should error for a null result")
	_, err = newUnavailableServerExchange(t).GetValuations(t.Context(), &ValuationsRequest{InstrumentName: "BTCUSD-PERP", ValuationType: ValuationTypeMarkPrice})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetValuations should return a request failure")
}

func TestGetExpiredSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetExpiredSettlementPrice(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExpiredSettlementPrice must refuse a nil request")
	_, err = e.GetExpiredSettlementPrice(t.Context(), &ExpiredSettlementPriceRequest{Page: 1})
	require.ErrorIs(t, err, errInstrumentTypeRequired, "GetExpiredSettlementPrice must refuse a request without an instrument type")

	for _, tc := range []struct {
		req *ExpiredSettlementPriceRequest
		exp []ExpiredSettlementPriceResponse
	}{
		{
			req: &ExpiredSettlementPriceRequest{InstrumentType: InstrumentTypeFuture},
			exp: []ExpiredSettlementPriceResponse{
				{InstrumentName: "ETHUSD-260925", ExpiryTime: types.Time(time.UnixMilli(1790323200000)), SettlementPrice: 2672.67, Timestamp: types.Time(time.UnixMilli(1790323199500))},
				{InstrumentName: "BTCUSD-260925", ExpiryTime: types.Time(time.UnixMilli(1790323200000)), SettlementPrice: 83934.3, Timestamp: types.Time(time.UnixMilli(1790323199500))},
			},
		},
		{
			req: &ExpiredSettlementPriceRequest{InstrumentType: InstrumentTypeFuture, Page: 2},
			exp: []ExpiredSettlementPriceResponse{
				{InstrumentName: "BTCUSD-260424", ExpiryTime: types.Time(time.UnixMilli(1777017600000)), SettlementPrice: 77732.1, Timestamp: types.Time(time.UnixMilli(1777017599500))},
				{InstrumentName: "ETHUSD-260424", ExpiryTime: types.Time(time.UnixMilli(1777017600000)), SettlementPrice: 2313.94, Timestamp: types.Time(time.UnixMilli(1777017599500))},
			},
		},
	} {
		result, err := e.GetExpiredSettlementPrice(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetExpiredSettlementPrice must not error for page %d", tc.req.Page)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetExpiredSettlementPrice should decode every field for page %d", tc.req.Page)
		} else {
			assert.NotEmptyf(t, result, "GetExpiredSettlementPrice should return settlement prices for page %d", tc.req.Page)
		}
	}

	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-expired-settlement-price","code":0,"result":null}`).GetExpiredSettlementPrice(t.Context(), &ExpiredSettlementPriceRequest{InstrumentType: InstrumentTypeFuture})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetExpiredSettlementPrice should error for a null result")
	_, err = newUnavailableServerExchange(t).GetExpiredSettlementPrice(t.Context(), &ExpiredSettlementPriceRequest{InstrumentType: InstrumentTypeFuture})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetExpiredSettlementPrice should return a request failure")
}

func TestGetDepositInstruments(t *testing.T) {
	t.Parallel()
	result, err := e.GetDepositInstruments(t.Context())
	require.NoError(t, err, "GetDepositInstruments must not error")
	if mockTests {
		exp := []DepositInstrumentResponse{
			{
				Symbol:             "BTC",
				InstrumentType:     InstrumentTypeCurrency,
				DisplayName:        "BTC",
				BaseCurrency:       currency.BTC,
				QuoteCurrency:      currency.BTC,
				QuoteDecimals:      8,
				QuantityDecimals:   8,
				Haircut:            0.0625,
				CollateralEligible: true,
				PriceTickSize:      0.00000001,
				QuantityTickSize:   0.00000001,
				MaxLeverage:        1,
				ConversionFee:      0.4,
			},
			{
				Symbol:           "SOL.staked-COLLATERAL",
				InstrumentType:   InstrumentTypeCollateral,
				DisplayName:      "SOL.staked-COLLATERAL",
				BaseCurrency:     currency.SOL,
				QuoteCurrency:    currency.SOL,
				QuoteDecimals:    8,
				QuantityDecimals: 8,
				PriceTickSize:    0.00000001,
				QuantityTickSize: 0.00000001,
				MaxLeverage:      1,
				ConversionFee:    0.4,
				UnderlyingSymbol: "SOL.staked",
			},
			{
				Symbol:           "HYPE",
				InstrumentType:   InstrumentTypeCurrency,
				DisplayName:      "HYPE",
				BaseCurrency:     currency.HYPE,
				QuoteCurrency:    currency.HYPE,
				QuoteDecimals:    8,
				QuantityDecimals: 8,
				Haircut:          1,
				PriceTickSize:    0.00000001,
				QuantityTickSize: 0.00000001,
				MaxLeverage:      1,
				ConversionFee:    0.4,
				ProductType:      ProductTypeDigitalCurrencies,
			},
		}
		assert.Equal(t, exp, result, "GetDepositInstruments should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetDepositInstruments should return deposit instruments")
	}

	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-deposit-instruments","code":0,"result":null}`).GetDepositInstruments(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDepositInstruments should error for a null result")
	_, err = newUnavailableServerExchange(t).GetDepositInstruments(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDepositInstruments should return a request failure")
}

func TestGetInsurance(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	_, err := e.GetInsurance(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInsurance must refuse a nil request")
	_, err = e.GetInsurance(t.Context(), &InsuranceRequest{Count: 3})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetInsurance must refuse a request without a currency")
	_, err = e.GetInsurance(t.Context(), &InsuranceRequest{Currency: currency.USD, StartTime: start.Add(time.Hour), EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetInsurance must refuse a start after the end")
	_, err = e.GetInsurance(t.Context(), &InsuranceRequest{Currency: currency.USD, EndTime: time.UnixMilli(-1)})
	require.ErrorIs(t, err, errTimeBeforeUnixEpoch, "GetInsurance must refuse a time before the Unix epoch")

	for _, tc := range []struct {
		req *InsuranceRequest
		exp *InsuranceResponse
	}{
		{
			req: &InsuranceRequest{Currency: currency.USD, Count: 3, StartTime: start, EndTime: start.Add(3 * time.Hour)},
			exp: &InsuranceResponse{
				Currency: currency.USD,
				Data: []InsuranceFundBalance{
					{Balance: 7351059.39615527389065578, Timestamp: types.Time(time.UnixMilli(1792807200028))},
					{Balance: 7351060.35128467012890248, Timestamp: types.Time(time.UnixMilli(1792803600028))},
					{Balance: 7351060.35128467012890248, Timestamp: types.Time(time.UnixMilli(1792800001961))},
				},
			},
		},
		{
			// Crypto.com keeps no insurance fund in BTC, so it returns an empty list
			req: &InsuranceRequest{Currency: currency.BTC, Count: 3},
			exp: &InsuranceResponse{Currency: currency.BTC, Data: []InsuranceFundBalance{}},
		},
	} {
		result, err := e.GetInsurance(t.Context(), tc.req)
		require.NoErrorf(t, err, "GetInsurance must not error for %s", tc.req.Currency)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetInsurance should decode every field for %s", tc.req.Currency)
		} else {
			assert.Equalf(t, tc.req.Currency, result.Currency, "GetInsurance should return the fund of %s", tc.req.Currency)
		}
	}

	ex := newQueryCheckExchange(t, "public/get-insurance", url.Values{"instrument_name": {"USD"}}, `{"data":[],"instrument_name":"USD"}`)
	_, err = ex.GetInsurance(t.Context(), &InsuranceRequest{Currency: currency.USD.Lower()})
	require.NoError(t, err, "GetInsurance must not error without optional params")
	_, err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-insurance","code":0,"result":null}`).GetInsurance(t.Context(), &InsuranceRequest{Currency: currency.USD})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetInsurance should error for a null result")
	_, err = newUnavailableServerExchange(t).GetInsurance(t.Context(), &InsuranceRequest{Currency: currency.USD})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetInsurance should return a request failure")
}

func TestSetTimeWindowParams(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 24, 0, 0, 0, 123456789, time.UTC)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		toParam    func(time.Time) (uint64, error)
		exp        url.Values
		err        error
	}{
		{name: "no bounds", toParam: unixMilliParam, exp: url.Values{}},
		{name: "start only", start: start, toParam: unixMilliParam, exp: url.Values{"start_ts": {"1792800000123"}}},
		{name: "end only", end: start, toParam: unixNanoParam, exp: url.Values{"end_ts": {"1792800000123456789"}}},
		{name: "both in milliseconds", start: start, end: start.Add(time.Hour), toParam: unixMilliParam, exp: url.Values{"start_ts": {"1792800000123"}, "end_ts": {"1792803600123"}}},
		{name: "both in nanoseconds", start: start, end: start.Add(time.Hour), toParam: unixNanoParam, exp: url.Values{"start_ts": {"1792800000123456789"}, "end_ts": {"1792803600123456789"}}},
		{name: "start after end", start: start.Add(time.Nanosecond), end: start, toParam: unixNanoParam, exp: url.Values{}, err: common.ErrStartAfterEnd},
		{name: "start equals end", start: start, end: start, toParam: unixNanoParam, exp: url.Values{}, err: common.ErrStartEqualsEnd},
		{name: "start before the epoch", start: time.Unix(0, -1), toParam: unixNanoParam, exp: url.Values{}, err: errTimeBeforeUnixEpoch},
		{name: "end before the epoch", end: time.UnixMilli(-1), toParam: unixMilliParam, exp: url.Values{}, err: errTimeBeforeUnixEpoch},
	} {
		params := url.Values{}
		err := setTimeWindowParams(params, tc.start, tc.end, tc.toParam)
		require.ErrorIsf(t, err, tc.err, "setTimeWindowParams must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.exp, params, "setTimeWindowParams should set the bounds expected for %s", tc.name)
	}
}

// candlestickTimeframes are the candlestick timeframes Crypto.com documents, with their intervals
var candlestickTimeframes = []struct {
	interval  kline.Interval
	timeframe string
}{
	{kline.OneMin, "1m"},
	{kline.FiveMin, "5m"},
	{kline.FifteenMin, "15m"},
	{kline.ThirtyMin, "30m"},
	{kline.OneHour, "1h"},
	{kline.TwoHour, "2h"},
	{kline.FourHour, "4h"},
	{kline.TwelveHour, "12h"},
	{kline.OneDay, "1D"},
	{kline.SevenDay, "7D"},
	{kline.TwoWeek, "14D"},
	{kline.OneMonth, "1M"},
}

func TestIntervalToTimeframe(t *testing.T) {
	t.Parallel()
	for _, tc := range candlestickTimeframes {
		timeframe, err := intervalToTimeframe(tc.interval)
		require.NoErrorf(t, err, "intervalToTimeframe must not error for %s", tc.interval)
		assert.Equalf(t, tc.timeframe, timeframe, "intervalToTimeframe should name %s as Crypto.com does", tc.interval)
	}
	// Crypto.com also serves 6h candles, which its docs leave out
	for _, interval := range []kline.Interval{0, kline.ThreeMin, kline.SixHour, kline.ThreeDay} {
		_, err := intervalToTimeframe(interval)
		assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "intervalToTimeframe should refuse %s", interval)
	}
}

func TestTimeframeToInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range candlestickTimeframes {
		interval, err := timeframeToInterval(tc.timeframe)
		require.NoErrorf(t, err, "timeframeToInterval must not error for %s", tc.timeframe)
		assert.Equalf(t, tc.interval, interval, "timeframeToInterval should give the interval of %s", tc.timeframe)
	}
	// Crypto.com echoes the timeframe a request names, and also accepts its older names, such as M5, and 1d
	for _, timeframe := range []string{"", "6h", "M5", "1d", "1H", "1w"} {
		_, err := timeframeToInterval(timeframe)
		assert.ErrorIsf(t, err, kline.ErrInvalidInterval, "timeframeToInterval should refuse %q", timeframe)
	}
}

func TestOrderbookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := OrderbookLevel{Price: 80694.94, Quantity: 0.03103, OrderCount: 3}
	for _, data := range []string{`["80694.94","0.03103","3"]`, `[80694.94,0.03103,3]`} {
		var level OrderbookLevel
		require.NoErrorf(t, json.Unmarshal([]byte(data), &level), "Unmarshal must not error for %s", data)
		assert.Equalf(t, exp, level, "Unmarshal should decode the price, quantity and number of orders of %s", data)
	}
	for _, tc := range []struct {
		data, reason string
	}{
		{data: `{"price":"1"}`, reason: "malformed order book level"},
		{data: `["1","2"]`, reason: "2 values"},
		{data: `["1","2","3","4"]`, reason: "4 values"},
		{data: `[true,"2","3"]`, reason: "price"},
		{data: `["1",{},"3"]`, reason: "quantity"},
		{data: `["1","2","x"]`, reason: "number of orders"},
		{data: `["1","2","-1"]`, reason: "number of orders"},
		{data: `["1","2","1.5"]`, reason: "number of orders"},
	} {
		var level OrderbookLevel
		err := level.UnmarshalJSON([]byte(tc.data))
		assert.ErrorIsf(t, err, errMalformedOrderbookLevel, "UnmarshalJSON should refuse %s", tc.data)
		assert.ErrorContainsf(t, err, tc.reason, "UnmarshalJSON should say why it refuses %s", tc.data)
	}
}
