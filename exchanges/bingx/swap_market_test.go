package bingx

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// swapMarketTradFiPair is a non-crypto perpetual contract, whose display name differs from its symbol
var swapMarketTradFiPair = currency.NewPair(currency.NewCode("NCSKMSFT2USD"), currency.USDT)

func TestGetSwapContracts(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetSwapContracts(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapContracts must return the symbol formatting error")

	btcUSDT := SwapContractResponse{
		ContractID:        "100",
		Symbol:            "BTC-USDT",
		Size:              0.0001,
		QuantityPrecision: 4,
		PricePrecision:    1,
		FeeRate:           0.0005,
		MakerFeeRate:      0.0002,
		TakerFeeRate:      0.0005,
		TradeMinQuantity:  0.0001,
		TradeMinUSDT:      2,
		Currency:          currency.USDT,
		Asset:             currency.BTC,
		Status:            SwapContractStatusOnline,
		APIStateOpen:      true,
		APIStateClose:     true,
		EnsureTrigger:     true,
		TriggerFeeRate:    0.0005,
		LaunchTime:        types.Time(time.UnixMilli(1586275200000)),
		DisplayName:       "BTC-USDT",
	}
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []SwapContractResponse
	}{
		{name: "USDT-settled contract", symbol: currency.NewBTCUSDT(), exp: []SwapContractResponse{btcUSDT}},
		{
			name:   "USDC-settled contract",
			symbol: currency.NewPair(currency.BTC, currency.USDC),
			exp: []SwapContractResponse{
				{
					ContractID:        "100410",
					Symbol:            "BTC-USDC",
					Size:              0.0001,
					QuantityPrecision: 4,
					PricePrecision:    1,
					FeeRate:           0.0005,
					MakerFeeRate:      0.0002,
					TakerFeeRate:      0.0005,
					TradeMinQuantity:  0.0001,
					TradeMinUSDT:      2,
					Currency:          currency.USDC,
					Asset:             currency.BTC,
					Status:            SwapContractStatusOnline,
					APIStateOpen:      true,
					APIStateClose:     true,
					EnsureTrigger:     true,
					TriggerFeeRate:    0.0006,
					LaunchTime:        types.Time(time.UnixMilli(1719134881000)),
					DisplayName:       "BTC-USDC",
				},
			},
		},
		{
			name:   "TradFi contract",
			symbol: swapMarketTradFiPair,
			exp: []SwapContractResponse{
				{
					ContractID:        "101885",
					Symbol:            "NCSKMSFT2USD-USDT",
					Size:              0.001,
					QuantityPrecision: 3,
					PricePrecision:    2,
					FeeRate:           0.0005,
					MakerFeeRate:      0.0002,
					TakerFeeRate:      0.0005,
					TradeMinQuantity:  0.004,
					TradeMinUSDT:      2,
					Currency:          currency.USDT,
					Asset:             currency.NewCode("NCSKMSFT2USD"),
					Status:            SwapContractStatusOnline,
					APIStateOpen:      true,
					APIStateClose:     true,
					TriggerFeeRate:    0.002,
					LaunchTime:        types.Time(time.UnixMilli(1762161300000)),
					DisplayName:       "MSFT-USDT",
				},
			},
		},
		{
			name:   "every contract",
			symbol: currency.EMPTYPAIR,
			exp: []SwapContractResponse{
				btcUSDT,
				{
					ContractID:        "101",
					Symbol:            "ETH-USDT",
					Size:              0.01,
					QuantityPrecision: 2,
					PricePrecision:    2,
					FeeRate:           0.0005,
					MakerFeeRate:      0.0002,
					TakerFeeRate:      0.0005,
					TradeMinQuantity:  0.01,
					TradeMinUSDT:      2,
					Currency:          currency.USDT,
					Asset:             currency.ETH,
					Status:            SwapContractStatusOnline,
					APIStateOpen:      true,
					APIStateClose:     true,
					EnsureTrigger:     true,
					TriggerFeeRate:    0.0005,
					LaunchTime:        types.Time(time.UnixMilli(1586275200000)),
					DisplayName:       "ETH-USDT",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSwapContracts(t.Context(), tc.symbol)
			require.NoError(t, err, "GetSwapContracts must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSwapContracts should decode every field")
				return
			}
			if tc.symbol.IsEmpty() {
				assert.Greater(t, len(result), len(tc.exp), "GetSwapContracts should return every contract")
				return
			}
			require.Len(t, result, 1, "GetSwapContracts must return the requested contract")
			assert.Equal(t, tc.exp[0].Symbol, result[0].Symbol, "Symbol should be the requested symbol")
			assert.Equal(t, tc.exp[0].Currency, result[0].Currency, "Currency should be the contract's settlement currency")
			assert.Equal(t, tc.exp[0].DisplayName, result[0].DisplayName, "DisplayName should be the platform's name for the contract")
		})
	}
}

func TestGetSwapOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapOrderBook(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapOrderBook must reject an empty symbol")
	_, err = new(Exchange).GetSwapOrderBook(t.Context(), currency.NewBTCUSDT(), 5)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapOrderBook must return the symbol formatting error")

	result, err := e.GetSwapOrderBook(t.Context(), currency.NewBTCUSDT(), 5)
	require.NoError(t, err, "GetSwapOrderBook must not error")
	require.NotNil(t, result, "GetSwapOrderBook must return an order book")
	if mockTests {
		bids := orderbook.LevelsArrayPriceAmount{
			{Price: 74476.3, Amount: 5.5749},
			{Price: 74476.2, Amount: 0.0248},
			{Price: 74475.9, Amount: 0.0165},
			{Price: 74475.6, Amount: 0.0899},
			{Price: 74475.5, Amount: 0.0183},
		}
		asks := orderbook.LevelsArrayPriceAmount{
			{Price: 74476.5, Amount: 1.3373},
			{Price: 74476.6, Amount: 0.0211},
			{Price: 74476.9, Amount: 0.0884},
			{Price: 74477.0, Amount: 0.0245},
			{Price: 74477.1, Amount: 0.0079},
		}
		exp := &SwapOrderBookResponse{
			Time:     types.Time(time.UnixMilli(1794423579205)),
			Bids:     bids,
			Asks:     asks,
			BidsCoin: bids,
			AsksCoin: asks,
		}
		assert.Equal(t, exp, result, "GetSwapOrderBook should decode every field")
		return
	}
	assert.Len(t, result.Bids, 5, "GetSwapOrderBook should return the requested number of bids")
	assert.Len(t, result.Asks, 5, "GetSwapOrderBook should return the requested number of asks")
}

func TestGetSwapRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapRecentTrades(t.Context(), currency.EMPTYPAIR, 3)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapRecentTrades must reject an empty symbol")
	_, err = new(Exchange).GetSwapRecentTrades(t.Context(), currency.NewBTCUSDT(), 3)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapRecentTrades must return the symbol formatting error")

	result, err := e.GetSwapRecentTrades(t.Context(), currency.NewBTCUSDT(), 3)
	require.NoError(t, err, "GetSwapRecentTrades must not error")
	if mockTests {
		exp := []SwapRecentTradeResponse{
			{
				Time:          types.Time(time.UnixMilli(1794423409678)),
				IsBuyerMaker:  true,
				Price:         74429.8,
				Quantity:      0.0152,
				QuoteQuantity: 1131.33,
				FillID:        "759281606",
				Timestamp:     types.Time(time.UnixMilli(1794423409678)),
			},
			{
				Time:          types.Time(time.UnixMilli(1794423409734)),
				Price:         74429.9,
				Quantity:      0.0011,
				QuoteQuantity: 81.87,
				FillID:        "759281607",
				Timestamp:     types.Time(time.UnixMilli(1794423409734)),
			},
			{
				Time:          types.Time(time.UnixMilli(1794423409933)),
				Price:         74429.9,
				Quantity:      0.0007,
				QuoteQuantity: 52.10,
				FillID:        "759281608",
				Timestamp:     types.Time(time.UnixMilli(1794423409933)),
			},
		}
		assert.Equal(t, exp, result, "GetSwapRecentTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetSwapRecentTrades should return trades")
}

func TestGetSwapMarkPriceAndFundingRate(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetSwapMarkPriceAndFundingRate(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapMarkPriceAndFundingRate must return the symbol formatting error")
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []SwapMarkPriceAndFundingRateResponse
	}{
		{
			name:   "one contract",
			symbol: currency.NewBTCUSDT(),
			exp: []SwapMarkPriceAndFundingRateResponse{
				{
					Symbol:               "BTC-USDT",
					MarkPrice:            74475.8,
					IndexPrice:           74512.4,
					LastFundingRate:      0.000056,
					NextFundingTime:      types.Time(time.UnixMilli(1794441600000)),
					FundingIntervalHours: 8,
					MinFundingRate:       -0.003,
					MaxFundingRate:       0.003,
					UpdateTime:           types.Time(time.UnixMilli(1794412800000)),
				},
			},
		},
		{
			name:   "every contract",
			symbol: currency.EMPTYPAIR,
			exp: []SwapMarkPriceAndFundingRateResponse{
				{
					Symbol:               "BTC-USDT",
					MarkPrice:            74475.7,
					IndexPrice:           74512.4,
					LastFundingRate:      0.000056,
					NextFundingTime:      types.Time(time.UnixMilli(1794441600000)),
					FundingIntervalHours: 8,
					MinFundingRate:       -0.003,
					MaxFundingRate:       0.003,
					UpdateTime:           types.Time(time.UnixMilli(1794412800000)),
				},
				{
					Symbol:               "ETH-USDT",
					MarkPrice:            2083.12,
					IndexPrice:           2084.15,
					LastFundingRate:      0.0001,
					NextFundingTime:      types.Time(time.UnixMilli(1794441600000)),
					FundingIntervalHours: 8,
					MinFundingRate:       -0.003,
					MaxFundingRate:       0.003,
					UpdateTime:           types.Time(time.UnixMilli(1794412800000)),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSwapMarkPriceAndFundingRate(t.Context(), tc.symbol)
			require.NoError(t, err, "GetSwapMarkPriceAndFundingRate must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSwapMarkPriceAndFundingRate should decode every field")
				return
			}
			if tc.symbol.IsEmpty() {
				assert.Greater(t, len(result), 1, "GetSwapMarkPriceAndFundingRate should return every contract")
				return
			}
			require.Len(t, result, 1, "GetSwapMarkPriceAndFundingRate must return the requested contract")
			assert.Equal(t, "BTC-USDT", result[0].Symbol, "Symbol should be the requested symbol")
			assert.Positive(t, result[0].MarkPrice.Float64(), "MarkPrice should be positive")
		})
	}
}

func TestGetSwapFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapFundingRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapFundingRateHistory must reject a nil request")
	_, err = e.GetSwapFundingRateHistory(t.Context(), &SwapFundingRateHistoryRequest{Limit: 3})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapFundingRateHistory must reject an empty symbol")
	_, err = new(Exchange).GetSwapFundingRateHistory(t.Context(), &SwapFundingRateHistoryRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapFundingRateHistory must return the symbol formatting error")
	startTime, endTime := time.UnixMilli(1794211200000), time.UnixMilli(1794384000000)
	_, err = e.GetSwapFundingRateHistory(t.Context(), &SwapFundingRateHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapFundingRateHistory must reject a start time after the end time")

	t.Run("range without settlements", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ name, body string }{{"null data", `{"code":0,"msg":"","data":null}`}, {"an empty list", `{"code":0,"msg":"","data":[]}`}} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v2/quote/fundingRate": {tc.body}})
				result, err := ex.GetSwapFundingRateHistory(t.Context(), &SwapFundingRateHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: startTime.Add(time.Hour)})
				require.NoErrorf(t, err, "GetSwapFundingRateHistory must not error for %s", tc.name)
				assert.Emptyf(t, result, "GetSwapFundingRateHistory should return no settlements for %s", tc.name)
				assert.Lenf(t, requests(), 1, "GetSwapFundingRateHistory should make one request for %s", tc.name)
			})
		}
		for _, tc := range []struct {
			name, body string
			err        error
		}{
			{"an error", `{"code":100500,"msg":"busy"}`, errAPIResponse},
			{"a null body", `null`, common.ErrNoResponse},
			{"an empty body", ``, nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v2/quote/fundingRate": {tc.body}})
				_, err := ex.GetSwapFundingRateHistory(t.Context(), &SwapFundingRateHistoryRequest{Symbol: currency.NewBTCUSDT()})
				require.Errorf(t, err, "GetSwapFundingRateHistory must return the error of a reply with %s", tc.name)
				if tc.err != nil {
					assert.ErrorIsf(t, err, tc.err, "GetSwapFundingRateHistory should return the error of a reply with %s", tc.name)
				}
			})
		}
	})

	result, err := e.GetSwapFundingRateHistory(t.Context(), &SwapFundingRateHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 3})
	require.NoError(t, err, "GetSwapFundingRateHistory must not error")
	if mockTests {
		exp := []SwapFundingRateHistoryResponse{
			{Symbol: "BTC-USDT", FundingRate: 0.000097, FundingTime: types.Time(time.UnixMilli(1794384000000)), MarkPrice: 74062.1},
			{Symbol: "BTC-USDT", FundingRate: 0.0001, FundingTime: types.Time(time.UnixMilli(1794355200000)), MarkPrice: 73767.1},
			{Symbol: "BTC-USDT", FundingRate: 0.0001, FundingTime: types.Time(time.UnixMilli(1794326400000)), MarkPrice: 73185.9},
		}
		assert.Equal(t, exp, result, "GetSwapFundingRateHistory should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetSwapFundingRateHistory must return funding settlements")
	assert.LessOrEqual(t, len(result), 3, "GetSwapFundingRateHistory should return at most the limit")
	for _, r := range result {
		assert.Falsef(t, r.FundingTime.Time().Before(startTime) || r.FundingTime.Time().After(endTime), "FundingTime %s should be within the requested range", r.FundingTime)
	}
}

func TestGetSwapKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapKlines must reject a nil request")
	_, err = e.GetSwapKlines(t.Context(), &SwapKlinesRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapKlines must reject an empty symbol")
	_, err = e.GetSwapKlines(t.Context(), &SwapKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.TenMin})
	require.ErrorIs(t, err, errUnsupportedInterval, "GetSwapKlines must reject an unsupported interval")
	startTime, endTime := time.UnixMilli(1794326400000), time.UnixMilli(1794340800000)
	_, err = e.GetSwapKlines(t.Context(), &SwapKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneHour, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapKlines must reject a start time after the end time")

	for _, tc := range []struct {
		name string
		arg  *SwapKlinesRequest
		exp  []SwapKlineResponse
	}{
		{
			name: "time range",
			arg:  &SwapKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneHour, StartTime: startTime, EndTime: endTime},
			exp: []SwapKlineResponse{
				{Open: 73455.5, Close: 73440.2, High: 73566.6, Low: 73292.5, Volume: 296.1055, Time: types.Time(time.UnixMilli(1794337200000))},
				{Open: 73257.6, Close: 73455.5, High: 73645.1, Low: 73078.0, Volume: 807.2174, Time: types.Time(time.UnixMilli(1794333600000))},
				{Open: 72850.7, Close: 73257.6, High: 73262.4, Low: 72751.9, Volume: 855.8179, Time: types.Time(time.UnixMilli(1794330000000))},
				{Open: 73180.1, Close: 72850.7, High: 73532.9, Low: 72705.9, Volume: 1768.5875, Time: types.Time(time.UnixMilli(1794326400000))},
			},
		},
		{
			name: "time zone and limit",
			arg:  &SwapKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneDay, TimeZone: 8, Limit: 2},
			exp: []SwapKlineResponse{
				{Open: 73953.1, Close: 74476.4, High: 74748.0, Low: 73885.2, Volume: 1222.8595, Time: types.Time(time.UnixMilli(1794412800000))},
				{Open: 73180.1, Close: 73953.1, High: 74449.0, Low: 72705.9, Volume: 12276.0978, Time: types.Time(time.UnixMilli(1794326400000))},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSwapKlines(t.Context(), tc.arg)
			require.NoError(t, err, "GetSwapKlines must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSwapKlines should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetSwapKlines must return candles")
			for _, k := range result {
				if !tc.arg.StartTime.IsZero() {
					assert.Falsef(t, k.Time.Time().Before(tc.arg.StartTime) || !k.Time.Time().Before(tc.arg.EndTime), "Time %s should be within the requested range", k.Time)
				}
			}
		})
	}
}

func TestGetSwapOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapOpenInterest(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapOpenInterest must reject an empty symbol")
	_, err = new(Exchange).GetSwapOpenInterest(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapOpenInterest must return the symbol formatting error")

	result, err := e.GetSwapOpenInterest(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetSwapOpenInterest must not error")
	require.NotNil(t, result, "GetSwapOpenInterest must return open interest")
	if mockTests {
		exp := &SwapOpenInterestResponse{OpenInterest: 634048539.6, Symbol: "BTC-USDT", Time: types.Time(time.UnixMilli(1794423586343))}
		assert.Equal(t, exp, result, "GetSwapOpenInterest should decode every field")
		return
	}
	assert.Equal(t, "BTC-USDT", result.Symbol, "Symbol should be the requested symbol")
	assert.Positive(t, result.OpenInterest.Float64(), "OpenInterest should be positive")
}

func TestGetSwap24HourTicker(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetSwap24HourTicker(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwap24HourTicker must return the symbol formatting error")
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []Swap24HourTickerResponse
	}{
		{
			name:   "one contract",
			symbol: currency.NewBTCUSDT(),
			exp: []Swap24HourTickerResponse{
				{
					Symbol:             "BTC-USDT",
					PriceChange:        934.9,
					PriceChangePercent: 1.27,
					LastPrice:          74466.8,
					LastQuantity:       0.0009,
					HighPrice:          74748.0,
					LowPrice:           73187.5,
					Volume:             10092.81,
					QuoteVolume:        745120024.59,
					OpenPrice:          73531.9,
					OpenTime:           types.Time(time.UnixMilli(1794337185000)),
					CloseTime:          types.Time(time.UnixMilli(1794423585000)),
					BidPrice:           74467.2,
					BidQuantity:        15.9549,
					AskPrice:           74467.3,
					AskQuantity:        23.41,
				},
			},
		},
		{
			name:   "every contract",
			symbol: currency.EMPTYPAIR,
			exp: []Swap24HourTickerResponse{
				{
					Symbol:             "NCCOGOLD2USD-USDT",
					PriceChange:        -40.54,
					PriceChangePercent: -1,
					LastPrice:          3994.73,
					LastQuantity:       0.0019,
					HighPrice:          4073.86,
					LowPrice:           3993.78,
					Volume:             213983.59,
					QuoteVolume:        863995029.39,
					OpenPrice:          4035.27,
					OpenTime:           types.Time(time.UnixMilli(1794337187000)),
					CloseTime:          types.Time(time.UnixMilli(1794423587000)),
					BidPrice:           3994.72,
					BidQuantity:        84.0512,
					AskPrice:           3994.73,
					AskQuantity:        266.1873,
				},
				{
					Symbol:             "BTC-USDT",
					PriceChange:        934.9,
					PriceChangePercent: 1.27,
					LastPrice:          74466.8,
					LastQuantity:       0.0002,
					HighPrice:          74748.0,
					LowPrice:           73187.5,
					Volume:             10092.88,
					QuoteVolume:        745124902.80,
					OpenPrice:          73531.9,
					OpenTime:           types.Time(time.UnixMilli(1794337187000)),
					CloseTime:          types.Time(time.UnixMilli(1794423587000)),
					BidPrice:           74469.5,
					BidQuantity:        43.7142,
					AskPrice:           74469.6,
					AskQuantity:        9.217,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSwap24HourTicker(t.Context(), tc.symbol)
			require.NoError(t, err, "GetSwap24HourTicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSwap24HourTicker should decode every field")
				return
			}
			if tc.symbol.IsEmpty() {
				assert.Greater(t, len(result), 1, "GetSwap24HourTicker should return every contract")
				return
			}
			require.Len(t, result, 1, "GetSwap24HourTicker must return the requested contract")
			assert.Equal(t, "BTC-USDT", result[0].Symbol, "Symbol should be the requested symbol")
			assert.Positive(t, result[0].LastPrice.Float64(), "LastPrice should be positive")
		})
	}
}

func TestGetSwapHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapHistoricalTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapHistoricalTrades must reject a nil request")
	_, err = e.GetSwapHistoricalTrades(t.Context(), &SwapHistoricalTradesRequest{Limit: 3})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapHistoricalTrades must reject an empty symbol")
	_, err = new(Exchange).GetSwapHistoricalTrades(t.Context(), &SwapHistoricalTradesRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapHistoricalTrades must return the symbol formatting error")

	t.Run("request parameters", func(t *testing.T) {
		t.Parallel()
		var path string
		var query url.Values
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			path, query = r.URL.Path, r.URL.Query()
			_, err := io.WriteString(w, `{"code":0,"msg":"","data":[]}`)
			assert.NoError(t, err, "WriteString should not error")
		})
		_, err := ex.GetSwapHistoricalTrades(t.Context(), &SwapHistoricalTradesRequest{Symbol: currency.NewBTCUSDT(), FromID: 759282370, Limit: 100})
		require.NoError(t, err, "GetSwapHistoricalTrades must not error")
		assert.Equal(t, "/openApi/swap/v1/market/historicalTrades", path, "GetSwapHistoricalTrades should request its path")
		_, err = strconv.ParseInt(query.Get("timestamp"), 10, 64)
		assert.NoError(t, err, "timestamp should be sent as an integer")
		query.Del("timestamp")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "fromId": {"759282370"}, "limit": {"100"}}, query, "GetSwapHistoricalTrades should send the trade ID to start from and the limit")
	})

	t.Run("latest trades", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSwapHistoricalTrades(t.Context(), &SwapHistoricalTradesRequest{Symbol: currency.NewBTCUSDT(), Limit: 3})
		require.NoError(t, err, "GetSwapHistoricalTrades must not error")
		if mockTests {
			exp := []SwapHistoricalTradeResponse{
				{Time: types.Time(time.UnixMilli(1794423539030)), Price: 74491.6, Quantity: 0.0001, QuoteQuantity: 7.45, ID: "759282370"},
				{Time: types.Time(time.UnixMilli(1794423539110)), Price: 74491.6, Quantity: 0.1341, QuoteQuantity: 9989.32, ID: "759282371"},
				{Time: types.Time(time.UnixMilli(1794423539133)), IsBuyerMaker: true, Price: 74491.5, Quantity: 0.0007, QuoteQuantity: 52.14, ID: "759282372"},
			}
			assert.Equal(t, exp, result, "GetSwapHistoricalTrades should decode every field")
			return
		}
		assert.NotEmpty(t, result, "GetSwapHistoricalTrades should return trades")
	})
}

func TestGetSwapOrderBookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapOrderBookTicker(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapOrderBookTicker must reject an empty symbol")
	_, err = new(Exchange).GetSwapOrderBookTicker(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapOrderBookTicker must return the symbol formatting error")

	t.Run("missing book ticker", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, err := io.WriteString(w, `{"code":0,"msg":"","data":{}}`)
			assert.NoError(t, err, "WriteString should not error")
		})
		_, err := ex.GetSwapOrderBookTicker(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetSwapOrderBookTicker should reject a reply without a book ticker")
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, err := io.WriteString(w, `{"code":109425,"msg":"symbol does not exist","data":{}}`)
			assert.NoError(t, err, "WriteString should not error")
		})
		_, err := ex.GetSwapOrderBookTicker(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, errAPIResponse, "GetSwapOrderBookTicker should return the API error")
	})

	t.Run("best bid and ask", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetSwapOrderBookTicker(t.Context(), currency.NewBTCUSDT())
		require.NoError(t, err, "GetSwapOrderBookTicker must not error")
		require.NotNil(t, result, "GetSwapOrderBookTicker must return a result")
		if mockTests {
			exp := &SwapOrderBookTickerResponse{
				BookTicker: &SwapOrderBookTicker{
					Symbol:       "BTC-USDT",
					BidPrice:     74466.8,
					BidQuantity:  6.6915,
					AskPrice:     74466.9,
					AskQuantity:  27.1585,
					LastUpdateID: 759282669,
					Time:         types.Time(time.UnixMilli(1794423591333)),
				},
			}
			assert.Equal(t, exp, result, "GetSwapOrderBookTicker should decode every field")
			return
		}
		assert.Equal(t, "BTC-USDT", result.BookTicker.Symbol, "Symbol should be the requested symbol")
		assert.Positive(t, result.BookTicker.BidPrice, "BidPrice should be positive")
	})
}

func TestGetSwapMarkPriceKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarkPriceKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapMarkPriceKlines must reject a nil request")
	_, err = e.GetSwapMarkPriceKlines(t.Context(), &SwapMarkPriceKlinesRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapMarkPriceKlines must reject an empty symbol")
	_, err = e.GetSwapMarkPriceKlines(t.Context(), &SwapMarkPriceKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.TenMin})
	require.ErrorIs(t, err, errUnsupportedInterval, "GetSwapMarkPriceKlines must reject an unsupported interval")
	startTime, endTime := time.UnixMilli(1794326400000), time.UnixMilli(1794340800000)
	_, err = e.GetSwapMarkPriceKlines(t.Context(), &SwapMarkPriceKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneHour, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapMarkPriceKlines must reject a start time after the end time")

	result, err := e.GetSwapMarkPriceKlines(t.Context(), &SwapMarkPriceKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneHour, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetSwapMarkPriceKlines must not error")
	if mockTests {
		exp := []SwapMarkPriceKlineResponse{
			{Open: 73439.2, Close: 73754.2, High: 73845.1, Low: 73281.6, OpenTime: types.Time(time.UnixMilli(1794337200000)), CloseTime: types.Time(time.UnixMilli(1794340800000))},
			{Open: 73455.7, Close: 73439.2, High: 73566.7, Low: 73296.4, OpenTime: types.Time(time.UnixMilli(1794333600000)), CloseTime: types.Time(time.UnixMilli(1794337200000))},
			{Open: 73257.6, Close: 73455.7, High: 73644.9, Low: 73079.9, OpenTime: types.Time(time.UnixMilli(1794330000000)), CloseTime: types.Time(time.UnixMilli(1794333600000))},
			{Open: 72856.4, Close: 73257.6, High: 73262.6, Low: 72754.3, OpenTime: types.Time(time.UnixMilli(1794326400000)), CloseTime: types.Time(time.UnixMilli(1794330000000))},
			{Open: 73185.2, Close: 72856.4, High: 73532.9, Low: 72715.6, OpenTime: types.Time(time.UnixMilli(1794322800000)), CloseTime: types.Time(time.UnixMilli(1794326400000))},
		}
		assert.Equal(t, exp, result, "GetSwapMarkPriceKlines should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetSwapMarkPriceKlines must return candles")
	for _, k := range result {
		assert.Falsef(t, k.CloseTime.Time().Before(startTime) || k.CloseTime.Time().After(endTime), "CloseTime %s should be within the requested range", k.CloseTime)
	}
}

func TestGetSwapPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetSwapPriceTicker(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapPriceTicker must return the symbol formatting error")
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []SwapPriceTickerResponse
	}{
		{
			name:   "one contract",
			symbol: currency.NewBTCUSDT(),
			exp:    []SwapPriceTickerResponse{{Symbol: "BTC-USDT", Price: 74457.1, Time: types.Time(time.UnixMilli(1794423593133))}},
		},
		{
			name:   "every contract",
			symbol: currency.EMPTYPAIR,
			exp: []SwapPriceTickerResponse{
				{Symbol: "0G-USDT", Price: 0.1543, Time: types.Time(time.UnixMilli(1794423592444))},
				{Symbol: "1000000BABYDOGE-USDT", Price: 0.0002945, Time: types.Time(time.UnixMilli(1794423593881))},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSwapPriceTicker(t.Context(), tc.symbol)
			require.NoError(t, err, "GetSwapPriceTicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSwapPriceTicker should decode every field")
				return
			}
			if tc.symbol.IsEmpty() {
				assert.Greater(t, len(result), 1, "GetSwapPriceTicker should return every contract")
				return
			}
			require.Len(t, result, 1, "GetSwapPriceTicker must return the requested contract")
			assert.Equal(t, "BTC-USDT", result[0].Symbol, "Symbol should be the requested symbol")
			assert.Positive(t, result[0].Price.Float64(), "Price should be positive")
		})
	}
}

func TestGetSwapTradingRules(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapTradingRules(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapTradingRules must reject an empty symbol")
	_, err = new(Exchange).GetSwapTradingRules(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapTradingRules must return the symbol formatting error")

	result, err := e.GetSwapTradingRules(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetSwapTradingRules must not error")
	require.NotNil(t, result, "GetSwapTradingRules must return trading rules")
	if mockTests {
		exp := &SwapTradingRulesResponse{
			MinSizeCoin:         0.0001,
			MinSizeUSD:          2,
			MaxNumberOrder:      200,
			ProtectionThreshold: 0.05,
			BuyMaxPrice:         1.1,
			BuyMinPrice:         0.2,
			SellMaxPrice:        5,
			SellMinPrice:        0.9,
			MarketRatio:         0.005,
		}
		assert.Equal(t, exp, result, "GetSwapTradingRules should decode every field")
		return
	}
	assert.Positive(t, result.MinSizeCoin.Float64(), "MinSizeCoin should be positive")
	assert.Positive(t, result.MaxNumberOrder.Float64(), "MaxNumberOrder should be positive")
}

func TestGetSwapTradingSchedule(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetSwapTradingSchedule(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapTradingSchedule must return the symbol formatting error")
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []SwapTradingScheduleResponse
	}{
		{
			name:   "one contract",
			symbol: swapMarketTradFiPair,
			exp: []SwapTradingScheduleResponse{
				{
					Symbol:       "NCSKMSFT2USD-USDT",
					ContractType: NonCryptoContractTypeStock,
					MarketStatus: MarketStatusOpen,
					TimeZone:     "UTC+08:00",
					UpdateTime:   types.Time(time.UnixMilli(1792045615000)),
					Sessions:     []SwapTradingSession{{StartTime: types.Time(time.UnixMilli(1794412800000)), EndTime: types.Time(time.UnixMilli(1795017600000)), Type: TradingSessionTypeRegular}},
				},
			},
		},
		{
			name:   "every contract",
			symbol: currency.EMPTYPAIR,
			exp: []SwapTradingScheduleResponse{
				{
					Symbol:       "NCCO1OILBRENT2USD-USDT",
					ContractType: NonCryptoContractTypeCommodity,
					MarketStatus: MarketStatusOpen,
					TimeZone:     "UTC+08:00",
					UpdateTime:   types.Time(time.UnixMilli(1792124235000)),
					Sessions:     []SwapTradingSession{{StartTime: types.Time(time.UnixMilli(1794412800000)), EndTime: types.Time(time.UnixMilli(1795017600000)), Type: TradingSessionTypeRegular}},
				},
				{
					Symbol:       "NCCO1OILWTI2USD-USDT",
					ContractType: NonCryptoContractTypeCommodity,
					MarketStatus: MarketStatusOpen,
					TimeZone:     "UTC+08:00",
					UpdateTime:   types.Time(time.UnixMilli(1792124220000)),
					Sessions:     []SwapTradingSession{{StartTime: types.Time(time.UnixMilli(1794412800000)), EndTime: types.Time(time.UnixMilli(1795017600000)), Type: TradingSessionTypeRegular}},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSwapTradingSchedule(t.Context(), tc.symbol)
			require.NoError(t, err, "GetSwapTradingSchedule must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSwapTradingSchedule should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetSwapTradingSchedule must return schedules")
			if !tc.symbol.IsEmpty() {
				require.Len(t, result, 1, "GetSwapTradingSchedule must return the requested contract")
				assert.Equal(t, "NCSKMSFT2USD-USDT", result[0].Symbol, "Symbol should be the requested symbol")
			}
			assert.NotEmpty(t, result[0].Sessions, "Sessions should cover the schedule's days")
		})
	}
}

func TestSwapMarketSymbolParams(t *testing.T) {
	t.Parallel()
	params, err := e.swapMarketSymbolParams(currency.EMPTYPAIR)
	require.NoError(t, err, "swapMarketSymbolParams must not error for an empty pair")
	assert.Equal(t, url.Values{}, params, "swapMarketSymbolParams should return no parameters for an empty pair")

	params, err = e.swapMarketSymbolParams(currency.NewPair(currency.NewCode("btc"), currency.NewCode("usdt")))
	require.NoError(t, err, "swapMarketSymbolParams must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}}, params, "swapMarketSymbolParams should send the pair as an upper case perpetual symbol")

	_, err = new(Exchange).swapMarketSymbolParams(currency.NewBTCUSDT())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "swapMarketSymbolParams should return the symbol formatting error")
}

func TestSwapMarketKlineParams(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.UnixMilli(1794326400000), time.UnixMilli(1794340800000)
	_, err := e.swapMarketKlineParams(currency.EMPTYPAIR, kline.OneHour, startTime, endTime, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "swapMarketKlineParams must reject an empty symbol")
	_, err = e.swapMarketKlineParams(currency.NewBTCUSDT(), 0, startTime, endTime, 5)
	require.ErrorIs(t, err, errUnsupportedInterval, "swapMarketKlineParams must reject an empty interval")
	_, err = e.swapMarketKlineParams(currency.NewBTCUSDT(), kline.OneHour, endTime, startTime, 5)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "swapMarketKlineParams must reject a start time after the end time")
	_, err = new(Exchange).swapMarketKlineParams(currency.NewBTCUSDT(), kline.OneHour, startTime, endTime, 5)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "swapMarketKlineParams must return the symbol formatting error")

	params, err := e.swapMarketKlineParams(currency.NewBTCUSDT(), kline.OneMonth, startTime, endTime, 1440)
	require.NoError(t, err, "swapMarketKlineParams must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "interval": {"1M"}, "startTime": {"1794326400000"}, "endTime": {"1794340800000"}, "limit": {"1440"}}, params, "swapMarketKlineParams should send every parameter set")

	params, err = e.swapMarketKlineParams(currency.NewBTCUSDT(), kline.OneMin, time.Time{}, time.Time{}, 0)
	require.NoError(t, err, "swapMarketKlineParams must not error without optional parameters")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "interval": {"1m"}}, params, "swapMarketKlineParams should leave unset parameters out")
}
