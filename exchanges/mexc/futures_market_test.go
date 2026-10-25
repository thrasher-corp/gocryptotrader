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
	"github.com/thrasher-corp/gocryptotrader/types"
)

// futuresWindow returns a fixed window of futures market history. Mock responses are matched on the full request
// URL, so the window must not move, and the venue serves candles of any age
func futuresWindow() (start, end time.Time) {
	return time.Unix(1792800000, 0), time.Unix(1792821600, 0)
}

// futuresSymbols returns the symbols of a list of futures records, in order
func futuresSymbols[T any](records []T, symbol func(T) string) []string {
	symbols := make([]string, len(records))
	for i := range records {
		symbols[i] = symbol(records[i])
	}
	return symbols
}

func TestGetFuturesServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesServerTime(t.Context())
	require.NoError(t, err, "GetFuturesServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1792843031176), result, "GetFuturesServerTime should decode the venue's time")
	} else {
		assert.False(t, result.IsZero(), "GetFuturesServerTime should return the venue's time")
	}
}

func TestGetFuturesContracts(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesContracts(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesContracts must not error")
	require.Len(t, result, 1, "a single contract must be returned for a symbol")
	if mockTests {
		exp := &FuturesContractResponse{
			Symbol:                           "BTC_USDT",
			DisplayName:                      "BTC_USDT永续",
			DisplayNameEnglish:               "BTC_USDT PERPETUAL",
			PositionOpenType:                 3,
			BaseCoin:                         currency.BTC,
			QuoteCoin:                        currency.USDT,
			BaseCoinName:                     "BTC",
			QuoteCoinName:                    "USDT",
			FutureType:                       1,
			SettleCoin:                       currency.USDT,
			ContractSize:                     0.0001,
			MinLeverage:                      1,
			MaxLeverage:                      500,
			CountryConfigContractMaxLeverage: 200,
			PriceScale:                       1,
			AmountScale:                      4,
			PriceUnit:                        0.1,
			VolumeUnit:                       1,
			MinVolume:                        1,
			MaxVolume:                        400000,
			BidLimitPriceRate:                0.1,
			AskLimitPriceRate:                0.1,
			TakerFeeRate:                     0.0002,
			MaintenanceMarginRate:            0.001,
			InitialMarginRate:                0.002,
			RiskBaseVolume:                   19000000,
			RiskLevelLimit:                   1,
			PriceCoefficientVariation:        0.004,
			IndexOrigin:                      []string{"BITGET", "BYBIT", "BINANCE", "HTX", "UPBIT", "OKX", "MEXC", "KUCOIN"},
			ConceptPlate:                     []string{"mc-trade-zone-mainly", "mc-trade-zone-layer2", "mc-trade-zone-pow"},
			ConceptPlateID:                   []int64{50, 5, 12},
			RiskLimitType:                    "BY_VOLUME",
			MaxNumberOfOrders:                []uint64{200, 50},
			MarketOrderMaxLevel:              20,
			MarketOrderPriceLimitRate1:       0.1,
			MarketOrderPriceLimitRate2:       0.005,
			TriggerProtect:                   0.1,
			APIAllowed:                       true,
			DepthStepList:                    []types.Number{0.1, 1, 10, 100},
			LimitMaxVolume:                   2500000,
			BaseCoinIconURL:                  "https://public.mocortech.com/coin/F20250612182226438Ba037qttKoGcrm.png",
			ID:                               10,
			SettleCoinID:                     "128f589271cb4951b03e71e6323eb7be",
			BaseCoinID:                       "febc9973be4d4d53bb374476239eb219",
			CreateTime:                       types.Time(time.Unix(1591242684, 0)),
			OpeningCountdownOption:           1,
			ShowBeforeOpen:                   true,
			IsMaxLeverage:                    true,
			RiskLimitMode:                    "CUSTOM",
			RiskLimitCustom: []FuturesRiskLimitTier{
				{Level: 1, MaxVolume: 120000, MaintenanceMarginRate: 0.001, InitialMarginRate: 0.002, MaxLeverage: 500},
				{Level: 2, MaxVolume: 310000, MaintenanceMarginRate: 0.004, InitialMarginRate: 0.005, MaxLeverage: 200},
				{Level: 3, MaxVolume: 480000, MaintenanceMarginRate: 0.005, InitialMarginRate: 0.01, MaxLeverage: 100},
				{Level: 4, MaxVolume: 2800000, MaintenanceMarginRate: 0.01, InitialMarginRate: 0.02, MaxLeverage: 50},
				{Level: 5, MaxVolume: 17500000, MaintenanceMarginRate: 0.02, InitialMarginRate: 0.05, MaxLeverage: 20},
			},
			LiquidationFeeRate: 0.0004,
			FeeRateMode:        "NORMAL",
			TieredFeeRates:     []FuturesTieredFeeRate{},
			PairType:           1,
			FullName:           "BTC_USDT永续",
			FeeRateType:        "BASE",
			TagIDList:          []uint64{5, 8},
			RegularMaxLeverage: 200,
		}
		assert.Equal(t, exp, result[0], "GetFuturesContracts should decode every field")
	} else {
		assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")
		assert.Positive(t, result[0].ContractSize, "a contract should carry its size")
		assert.False(t, result[0].IsInverse(), "BTC_USDT should be a linear contract")
	}

	all, err := e.GetFuturesContracts(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesContracts must not error")
	require.NotEmpty(t, all, "every contract must be returned without a symbol")
	if mockTests {
		symbols := futuresSymbols(all, func(c *FuturesContractResponse) string { return c.Symbol })
		assert.Equal(t, []string{"BTC_USDT", "ETH_USDT", "BTC_USD", "HOOKED_USDT"}, symbols, "every recorded contract should be returned")
		assert.Equal(t, result[0], all[0], "a contract should decode the same in a list")
	}
}

func TestGetFuturesTransferableCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesTransferableCurrencies(t.Context())
	require.NoError(t, err, "GetFuturesTransferableCurrencies must not error")
	require.NotEmpty(t, result, "GetFuturesTransferableCurrencies must return currencies")
	if mockTests {
		require.Len(t, result, 45, "every recorded currency must be returned")
		exp := []currency.Code{currency.NewCode("STETH"), currency.NewCode("CRO"), currency.NewCode("MXSOL"), currency.NewCode("TAO"), currency.NewCode("PAXG")}
		assert.Equal(t, exp, result[:len(exp)], "the currencies should be decoded in order")
	}
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesOrderbook(t.Context(), futuresTradablePair, 5)
	require.NoError(t, err, "GetFuturesOrderbook must not error")
	if mockTests {
		exp := &FuturesOrderbookResponse{
			Asks: FuturesOrderbookLevels{
				{Price: 80564.3, Amount: 738, OrderCount: 2},
				{Price: 80564.5, Amount: 184, OrderCount: 1},
				{Price: 80564.8, Amount: 306, OrderCount: 1},
				{Price: 80565.1, Amount: 10000, OrderCount: 1},
				{Price: 80565.2, Amount: 306, OrderCount: 1},
			},
			Bids: FuturesOrderbookLevels{
				{Price: 80564.2, Amount: 14875, OrderCount: 6},
				{Price: 80564.1, Amount: 184, OrderCount: 1},
				{Price: 80564, Amount: 184, OrderCount: 1},
				{Price: 80563.8, Amount: 184, OrderCount: 1},
				{Price: 80562.8, Amount: 306, OrderCount: 1},
			},
			Version:   42904905215,
			Timestamp: types.Time(time.UnixMilli(1792843037369)),
		}
		assert.Equal(t, exp, result, "GetFuturesOrderbook should decode every level")
	} else {
		require.NotEmpty(t, result.Bids, "the book must have bids")
		require.NotEmpty(t, result.Asks, "the book must have asks")
		assert.Less(t, result.Bids[0].Price, result.Asks[0].Price, "the best bid should be below the best ask")
		assert.Positive(t, result.Version, "the book should carry its version")
		assert.False(t, result.Timestamp.Time().IsZero(), "the book should carry its time")
	}
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
	if mockTests {
		// A level of no contracts removes the price from the book
		exp := []*FuturesDepthSnapshotResponse{
			{Asks: FuturesOrderbookLevels{}, Bids: FuturesOrderbookLevels{{Price: 80560.5, Amount: 4137, OrderCount: 1}}, Version: 42904905213},
			{Asks: FuturesOrderbookLevels{}, Bids: FuturesOrderbookLevels{{Price: 80563}}, Version: 42904905212},
			{Asks: FuturesOrderbookLevels{{Price: 80565.9, Amount: 8159, OrderCount: 1}}, Bids: FuturesOrderbookLevels{}, Version: 42904905211},
		}
		assert.Equal(t, exp, result, "GetFuturesDepthSnapshots should decode every change")
	} else {
		require.NotEmpty(t, result, "GetFuturesDepthSnapshots must return changes")
		assert.Positive(t, result[0].Version, "a change should carry its version")
	}
}

func TestGetFuturesIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesIndexPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesIndexPrice(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesIndexPrice must not error")
	if mockTests {
		exp := &FuturesIndexPriceResponse{Symbol: "BTC_USDT", IndexPrice: 80602.5, Timestamp: types.Time(time.UnixMilli(1792843036520))}
		assert.Equal(t, exp, result, "GetFuturesIndexPrice should decode every field")
	} else {
		assert.Equal(t, futuresTradablePair.String(), result.Symbol, "the contract asked for should be returned")
		assert.Positive(t, result.IndexPrice, "the index price should be positive")
	}
}

func TestGetFuturesFairPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFairPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)

	result, err := e.GetFuturesFairPrice(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesFairPrice must not error")
	if mockTests {
		exp := &FuturesFairPriceResponse{Symbol: "BTC_USDT", FairPrice: 80564.2, Timestamp: types.Time(time.UnixMilli(1792843037422))}
		assert.Equal(t, exp, result, "GetFuturesFairPrice should decode every field")
	} else {
		assert.Equal(t, futuresTradablePair.String(), result.Symbol, "the contract asked for should be returned")
		assert.Positive(t, result.FairPrice, "the fair price should be positive")
	}
}

func TestGetFuturesFundingRates(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesFundingRates(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesFundingRates must not error")
	require.Len(t, result, 1, "a single rate must be returned for a symbol")
	if mockTests {
		exp := &FuturesFundingRateResponse{
			Symbol:         "BTC_USDT",
			FundingRate:    0.000034,
			MaxFundingRate: 0.0018,
			MinFundingRate: -0.0018,
			CollectCycle:   8,
			NextSettleTime: types.Time(time.Unix(1792857600, 0)),
			Timestamp:      types.Time(time.UnixMilli(1792843036306)),
			IndexPrice:     80602.5,
			FairPrice:      80563.1,
		}
		assert.Equal(t, exp, result[0], "GetFuturesFundingRates should decode every field")
	} else {
		assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")
		assert.Positive(t, result[0].CollectCycle, "the rate should carry its funding interval")
		assert.False(t, result[0].NextSettleTime.Time().IsZero(), "the rate should carry its settlement time")
	}

	result, err = e.GetFuturesFundingRates(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesFundingRates must not error")
	if mockTests {
		next, at := types.Time(time.Unix(1792857600, 0)), types.Time(time.UnixMilli(1792843116682))
		exp := []*FuturesFundingRateResponse{
			{Symbol: "BTC_USDT", FundingRate: 0.000034, MaxFundingRate: 0.0018, MinFundingRate: -0.0018, CollectCycle: 8, NextSettleTime: next, Timestamp: at, IndexPrice: 80602.6, FairPrice: 80563.9},
			{Symbol: "ETH_USDT", FundingRate: 0.000059, MaxFundingRate: 0.0018, MinFundingRate: -0.0018, CollectCycle: 8, NextSettleTime: next, Timestamp: at, IndexPrice: 2365.45, FairPrice: 2364.53},
			{Symbol: "BTC_USD", FundingRate: 0.000076, MaxFundingRate: 0.003, MinFundingRate: -0.003, CollectCycle: 8, NextSettleTime: next, Timestamp: at, IndexPrice: 80492.9, FairPrice: 80467.8},
		}
		assert.Equal(t, exp, result, "every contract's rate should be decoded without a symbol")
	} else {
		assert.NotEmpty(t, result, "every contract's rate should be returned without a symbol")
	}
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
	hours := make([]types.Time, 7)
	for i := range hours {
		hours[i] = types.Time(start.Add(time.Duration(i) * time.Hour))
	}
	none := make([]float64, len(hours))
	for name, tc := range map[string]struct {
		get func(*FuturesCandlesRequest) (*FuturesCandlesResponse, error)
		exp *FuturesCandlesResponse
	}{
		"trade": {
			get: func(r *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
				return e.GetFuturesCandles(t.Context(), r)
			},
			exp: &FuturesCandlesResponse{
				Time:      hours,
				Open:      []float64{81159.9, 80588.3, 80898.9, 80963.4, 80943.9, 80857.2, 80732.4},
				Close:     []float64{80588.3, 80898.9, 80963.4, 80943.9, 80857.2, 80732.4, 80746.6},
				High:      []float64{81188.2, 80935, 80965, 81068.6, 81069.4, 80859.8, 80793.9},
				Low:       []float64{80574.2, 80586.4, 80801.5, 80811, 80812, 80681.7, 80652},
				Volume:    []float64{16499022, 10006081, 4030439, 5675471, 5345322, 7363386, 5201999},
				Amount:    []float64{133289374.01748, 80819744.71053, 32607786.47078, 45927404.41558, 43252606.6287, 59455724.18861, 41996656.52525},
				RealOpen:  []float64{81160, 80588.3, 80899, 80963.5, 80943.9, 80857.1, 80732.3},
				RealClose: []float64{80588.3, 80898.9, 80963.4, 80943.9, 80857.2, 80732.4, 80746.6},
				RealHigh:  []float64{81188.2, 80935, 80965, 81068.6, 81069.4, 80859.8, 80793.9},
				RealLow:   []float64{80574.2, 80586.4, 80801.5, 80811, 80812, 80681.7, 80652},
			},
		},
		// The venue sends index and fair price candles without volumes, and its real prices as each close
		"index": {
			get: func(r *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
				return e.GetFuturesIndexPriceCandles(t.Context(), r)
			},
			exp: &FuturesCandlesResponse{
				Time:      hours,
				Open:      []float64{81189.1, 80623.5, 80934.2, 80994.7, 80975.5, 80881.2, 80774.1},
				Close:     []float64{80623.5, 80934.2, 80994.7, 80975.5, 80881.2, 80774.1, 80789.8},
				High:      []float64{81212.8, 80965.6, 80995, 81093.3, 81094.5, 80888.2, 80826.7},
				Low:       []float64{80603.8, 80616.6, 80845.9, 80848.1, 80843.9, 80722.6, 80695.5},
				Volume:    none,
				Amount:    none,
				RealOpen:  []float64{80623.5, 80934.2, 80994.7, 80975.5, 80881.2, 80774.1, 80789.8},
				RealClose: []float64{80623.5, 80934.2, 80994.7, 80975.5, 80881.2, 80774.1, 80789.8},
				RealHigh:  []float64{80623.5, 80934.2, 80994.7, 80975.5, 80881.2, 80774.1, 80789.8},
				RealLow:   []float64{80623.5, 80934.2, 80994.7, 80975.5, 80881.2, 80774.1, 80789.8},
			},
		},
		"fair": {
			get: func(r *FuturesCandlesRequest) (*FuturesCandlesResponse, error) {
				return e.GetFuturesFairPriceCandles(t.Context(), r)
			},
			exp: &FuturesCandlesResponse{
				Time:      hours,
				Open:      []float64{81158, 80591.8, 80899, 80963.5, 80944.8, 80857.1, 80732.4},
				Close:     []float64{80591.8, 80899, 80963.5, 80944.8, 80857.1, 80732.4, 80746.6},
				High:      []float64{81185.2, 80935.7, 80965, 81067.6, 81069.2, 80864.2, 80793.9},
				Low:       []float64{80577, 80586.7, 80808.1, 80813.3, 80812, 80683.5, 80652.1},
				Volume:    none,
				Amount:    none,
				RealOpen:  []float64{80591.8, 80899, 80963.5, 80944.8, 80857.1, 80732.4, 80746.6},
				RealClose: []float64{80591.8, 80899, 80963.5, 80944.8, 80857.1, 80732.4, 80746.6},
				RealHigh:  []float64{80591.8, 80899, 80963.5, 80944.8, 80857.1, 80732.4, 80746.6},
				RealLow:   []float64{80591.8, 80899, 80963.5, 80944.8, 80857.1, 80732.4, 80746.6},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.get(nil)
			require.ErrorIs(t, err, common.ErrNilPointer)
			_, err = tc.get(&FuturesCandlesRequest{Interval: kline.OneHour})
			require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
			_, err = tc.get(&FuturesCandlesRequest{Symbol: futuresTradablePair, Interval: kline.TenMin})
			require.ErrorIs(t, err, kline.ErrUnsupportedInterval)
			_, err = tc.get(&FuturesCandlesRequest{Symbol: futuresTradablePair, Interval: kline.OneHour, StartTime: end, EndTime: start})
			require.ErrorIs(t, err, common.ErrStartAfterEnd)

			result, err := tc.get(&FuturesCandlesRequest{Symbol: futuresTradablePair, Interval: kline.OneHour, StartTime: start, EndTime: end})
			require.NoError(t, err, "the candle request must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "every candle should be decoded")
				return
			}
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
	if mockTests {
		at := func(ms int64) types.Time { return types.Time(time.UnixMilli(ms)) }
		exp := []*FuturesTradeResponse{
			{Price: 80563.1, Volume: 72, Side: 1, OpenFlag: 1, SelfTrade: 2, Time: at(1792843035866), TradeID: "16745175758", MatchingEngineTime: at(1792843035866)},
			{Price: 80562.1, Volume: 12, Side: 2, OpenFlag: 1, SelfTrade: 2, Time: at(1792843035628), TradeID: "16745175737", MatchingEngineTime: at(1792843035628)},
			{Price: 80562.2, Volume: 9, Side: 1, OpenFlag: 3, SelfTrade: 1, Time: at(1792843035336), TradeID: "16745175717", MatchingEngineTime: at(1792843035336)},
			{Price: 80562.1, Volume: 289, Side: 2, OpenFlag: 3, SelfTrade: 2, Time: at(1792843034953), TradeID: "16745175628", MatchingEngineTime: at(1792843034953)},
			{Price: 80562.4, Volume: 184, Side: 2, OpenFlag: 3, SelfTrade: 2, Time: at(1792843034953), TradeID: "16745175627", MatchingEngineTime: at(1792843034953)},
		}
		assert.Equal(t, exp, result, "GetFuturesRecentTrades should decode every trade")
	} else {
		require.NotEmpty(t, result, "trades must be returned")
		assert.NotEmpty(t, result[0].TradeID, "a trade should carry its ID")
		assert.Positive(t, result[0].Price, "a trade should carry its price")
		assert.Positive(t, result[0].Volume, "a trade should carry its volume")
	}
}

func TestGetFuturesTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesTickers(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesTickers must not error")
	require.Len(t, result, 1, "a single ticker must be returned for a symbol")
	if mockTests {
		exp := &FuturesTickerResponse{
			ContractID:      10,
			Symbol:          "BTC_USDT",
			LastPrice:       80563,
			BestBidPrice:    80563,
			BestAskPrice:    80563.1,
			Volume24Hour:    363848365,
			Amount24Hour:    2955827156.60217,
			HoldVolume:      489894180,
			Low24Hour:       80409.4,
			High24Hour:      82032.7,
			PriceChangeRate: -0.0133,
			PriceChange:     -1088.2,
			IndexPrice:      80602.6,
			FairPrice:       80564.6,
			FundingRate:     0.000034,
			MaxBidPrice:     88662.8,
			MinAskPrice:     72542.3,
			Timestamp:       types.Time(time.UnixMilli(1792843033884)),
			PriceChangeRates: &FuturesPriceChangeRates{
				Zone:        "UTC+8",
				Rate:        -0.0133,
				Value:       -1088.2,
				Rate7Days:   -0.0093,
				Rate30Days:  -0.0404,
				Rate90Days:  0.2558,
				Rate180Days: 0.0327,
				Rate365Days: -0.2665,
			},
			PriceChangeRatesOfZones: []float64{-0.0133, -0.0073, -0.0133},
		}
		assert.Equal(t, exp, result[0], "GetFuturesTickers should decode every field")
	} else {
		assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")
		assert.Positive(t, result[0].LastPrice, "the ticker should carry the last price")
		require.NotNil(t, result[0].PriceChangeRates, "the ticker must carry its price changes")
		assert.NotEmpty(t, result[0].PriceChangeRates.Zone, "the price changes should name their time zone")
	}

	result, err = e.GetFuturesTickers(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesTickers must not error")
	require.NotEmpty(t, result, "every contract's ticker must be returned without a symbol")
	if mockTests {
		symbols := futuresSymbols(result, func(r *FuturesTickerResponse) string { return r.Symbol })
		assert.Equal(t, []string{"BTC_USDT", "ETH_USDT", "BTC_USD"}, symbols, "every recorded ticker should be returned")
	}
}

func TestGetFuturesInsuranceFundBalances(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesInsuranceFundBalances(t.Context(), futuresTradablePair)
	require.NoError(t, err, "GetFuturesInsuranceFundBalances must not error")
	require.Len(t, result, 1, "the venue answers a symbol with one balance, though its docs show a list")
	if mockTests {
		exp := &FuturesInsuranceFundResponse{Symbol: "BTC_USDT", Currency: currency.USDT, Available: 118554864.79198489, Timestamp: types.Time(time.UnixMilli(1792843039451))}
		assert.Equal(t, exp, result[0], "GetFuturesInsuranceFundBalances should decode every field")
	} else {
		assert.Equal(t, futuresTradablePair.String(), result[0].Symbol, "the contract asked for should be returned")
	}

	result, err = e.GetFuturesInsuranceFundBalances(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesInsuranceFundBalances must not error")
	if mockTests {
		at := types.Time(time.UnixMilli(1792843039606))
		exp := []*FuturesInsuranceFundResponse{
			{Symbol: "BTC_USDT", Currency: currency.USDT, Available: 118554864.79198489, Timestamp: at},
			{Symbol: "ETH_USDT", Currency: currency.USDT, Available: 68305101.75207666, Timestamp: at},
			{Symbol: "XAU_USDT", Currency: currency.USDT, Available: 190768.98725099856, Timestamp: at},
			{Symbol: "USOIL_USDT", Currency: currency.USDT, Available: 1265415.245677108, Timestamp: at},
			{Symbol: "IRYS_USDT", Currency: currency.USDT, Available: 128580.97390650162, Timestamp: at},
		}
		assert.Equal(t, exp, result, "every contract's balance should be decoded without a symbol")
	} else {
		assert.NotEmpty(t, result, "every contract's balance should be returned without a symbol")
	}
}

func TestGetFuturesInsuranceFundBalanceHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesInsuranceFundBalanceHistory(t.Context(), currency.EMPTYPAIR, nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesInsuranceFundBalanceHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageSize: futuresDefaultPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)

	result, err := e.GetFuturesInsuranceFundBalanceHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageNumber: 1, PageSize: 2})
	require.NoError(t, err, "GetFuturesInsuranceFundBalanceHistory must not error")
	if mockTests {
		exp := &FuturesPageResponse[FuturesInsuranceFundSnapshotResponse]{
			PageSize:    2,
			TotalCount:  42,
			TotalPage:   21,
			CurrentPage: 1,
			ResultList: []FuturesInsuranceFundSnapshotResponse{
				{Symbol: "BTC_USDT", Currency: currency.USDT, Available: 118553094.5310135, SnapshotTime: types.Time(time.Unix(1792828800, 0))},
				{Symbol: "BTC_USDT", Currency: currency.USDT, Available: 118552809.16805147, SnapshotTime: types.Time(time.Unix(1792814400, 0))},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesInsuranceFundBalanceHistory should decode the page")
	} else {
		require.NotEmpty(t, result.ResultList, "snapshots must be returned")
		assert.EqualValues(t, 1, result.CurrentPage, "the first page should be returned")
		assert.False(t, result.ResultList[0].SnapshotTime.Time().IsZero(), "a snapshot should carry its time")
	}
}

func TestGetFuturesFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFundingRateHistory(t.Context(), currency.EMPTYPAIR, nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = e.GetFuturesFundingRateHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageSize: futuresFundingRateHistoryPageSizeLimit + 1})
	require.ErrorIs(t, err, errInvalidLimit)

	result, err := e.GetFuturesFundingRateHistory(t.Context(), futuresTradablePair, &FuturesPageRequest{PageNumber: 1, PageSize: 2})
	require.NoError(t, err, "GetFuturesFundingRateHistory must not error")
	if mockTests {
		exp := &FuturesPageResponse[FuturesFundingRateHistoryResponse]{
			PageSize:    2,
			TotalCount:  1620,
			TotalPage:   810,
			CurrentPage: 1,
			ResultList: []FuturesFundingRateHistoryResponse{
				{Symbol: "BTC_USDT", FundingRate: 0.000035, SettleTime: types.Time(time.Unix(1792828800, 0)), CollectCycle: 8},
				{Symbol: "BTC_USDT", FundingRate: 0.000039, SettleTime: types.Time(time.Unix(1792800000, 0)), CollectCycle: 8},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesFundingRateHistory should decode the page, newest first")
	} else {
		require.Len(t, result.ResultList, 2, "the page size asked for must be returned")
		assert.True(t, result.ResultList[0].SettleTime.Time().After(result.ResultList[1].SettleTime.Time()), "the rates should be newest first")
	}
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
		id := FuturesID("1") // a receiver reused from an earlier decode
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
	assert.Equal(t, oneOrMany[*FuturesIndexPriceResponse]{{Symbol: "BTC_USDT", IndexPrice: 1.5}}, one, "an object should decode as one item")

	var many oneOrMany[*FuturesIndexPriceResponse]
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"A_USDT"},{"symbol":"B_USDT"}]`), &many), "Unmarshal must not error for a list")
	assert.Equal(t, oneOrMany[*FuturesIndexPriceResponse]{{Symbol: "A_USDT"}, {Symbol: "B_USDT"}}, many, "a list should decode as its items")
}

func TestPagedListUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var page pagedList[*FuturesFundingFeeResponse]
	require.NoError(t, json.Unmarshal([]byte(`{"pageSize":2,"totalCount":1,"totalPage":1,"currentPage":1,"resultList":[{"id":7}]}`), &page), "Unmarshal must not error for a page")
	assert.Equal(t, pagedList[*FuturesFundingFeeResponse]{{ID: 7}}, page, "a page should decode as its result list")

	var list pagedList[*FuturesFundingFeeResponse]
	require.NoError(t, json.Unmarshal([]byte(`[{"id":8},{"id":9}]`), &list), "Unmarshal must not error for a list")
	assert.Equal(t, pagedList[*FuturesFundingFeeResponse]{{ID: 8}, {ID: 9}}, list, "a list should decode as its items")
}

func TestFuturesOrderbookLevelsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var levels FuturesOrderbookLevels
	require.NoError(t, json.Unmarshal([]byte(`[[80654.3,4963,1],[80654.4,7584]]`), &levels), "Unmarshal must not error")
	assert.Equal(t, FuturesOrderbookLevels{{Price: 80654.3, Amount: 4963, OrderCount: 1}, {Price: 80654.4, Amount: 7584}}, levels,
		"every level should be decoded, with no order count where the venue sends none")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[[80654.3]]`), &levels), errInvalidOrderbookLevel, "a level without its contracts should be refused")
}
