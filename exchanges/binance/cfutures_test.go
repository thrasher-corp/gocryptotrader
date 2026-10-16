package binance

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFuturesAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAccountInfo(t.Context())
	require.NoError(t, err, "GetFuturesAccountInfo must not error")
	if mockTests {
		exp := &FuturesAccountInformationResponse{
			Assets: []FuturesAccountAsset{
				{
					Asset:                  currency.BTC,
					WalletBalance:          0.00241969,
					UnrealizedProfit:       -0.00000081,
					MarginBalance:          0.00241888,
					MaintenanceMargin:      0.00001102,
					InitialMargin:          0.00004533,
					PositionInitialMargin:  0.0000441,
					OpenOrderInitialMargin: 0.00000123,
					MaxWithdrawAmount:      0.00237355,
					CrossWalletBalance:     0.00241969,
					CrossUnPNL:             -0.00000081,
					AvailableBalance:       0.00237355,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []FuturesAccountInformationPosition{
				{
					Symbol:                 "BTCUSD_PERP",
					PositionAmount:         3,
					InitialMargin:          0.0000441,
					MaintenanceMargin:      0.00001102,
					UnrealizedProfit:       -0.00000081,
					PositionInitialMargin:  0.0000441,
					OpenOrderInitialMargin: 0.00000123,
					Leverage:               8,
					Isolated:               true,
					PositionSide:           "BOTH",
					EntryPrice:             85642.5,
					BreakEvenPrice:         85676.8,
					MaxQuantity:            500,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
					NotionalValue:          0.00352812,
					IsolatedWallet:         0.0000453,
				},
			},
			CanDeposit:  true,
			CanTrade:    true,
			CanWithdraw: true,
			FeeTier:     2,
			UpdateTime:  types.Time(time.UnixMilli(1625474304765)),
		}
		assert.Equal(t, exp, result, "GetFuturesAccountInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesAccountInfo should return account information")
}

func TestGetFuturesAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAccountBalance(t.Context())
	require.NoError(t, err, "GetFuturesAccountBalance must not error")
	if mockTests {
		exp := []FuturesAccountBalanceData{
			{
				AccountAlias:       "SgsR",
				Asset:              currency.BTC,
				Balance:            0.0025,
				WithdrawAvailable:  0.00237355,
				CrossWalletBalance: 0.00241969,
				CrossUnPNL:         -0.00000081,
				AvailableBalance:   0.00237355,
				UpdateTime:         types.Time(time.UnixMilli(1592468353979)),
			},
		}
		assert.Equal(t, exp, result, "GetFuturesAccountBalance should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesAccountBalance should return balances")
}

func TestFuturesIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesIncomeHistory must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.FuturesIncomeHistory(t.Context(), &CFuturesIncomeHistoryRequest{IncomeType: "COMMISSION", StartTime: endTime, EndTime: startTime, Limit: 5})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FuturesIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	result, err := e.FuturesIncomeHistory(t.Context(), &CFuturesIncomeHistoryRequest{Symbol: coinmTradablePair, IncomeType: "COMMISSION", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 5})
	require.NoError(t, err, "FuturesIncomeHistory must not error")
	if mockTests {
		exp := []FuturesIncomeHistoryData{
			{
				Symbol:        "BTCUSD_PERP",
				IncomeType:    "COMMISSION",
				Income:        -0.00000116,
				Asset:         currency.BTC,
				Info:          "COMMISSION",
				Time:          types.Time(time.Unix(1744120800, 0)),
				TransactionID: "9689322392",
				TradeID:       "1157458432",
			},
			{
				Symbol:        "BTCUSD_PERP",
				IncomeType:    "COMMISSION",
				Income:        -0.0001045,
				Asset:         currency.BTC,
				Info:          "COMMISSION",
				Time:          types.Time(time.Unix(1744124400, 0)),
				TransactionID: "9689322393",
				TradeID:       "1157458433",
			},
		}
		assert.Equal(t, exp, result, "FuturesIncomeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesIncomeHistory should return income records")
}

func TestFuturesNotionalBracket(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	btcusd := NotionalBracketData{
		Pair: "BTCUSD",
		Brackets: []NotionalBracket{
			{Bracket: 1, InitialLeverage: 125, QuantityCap: 5, MaintenanceMarginRatio: 0.004},
			{Bracket: 2, InitialLeverage: 100, QuantityCap: 10, QuantityFloor: 5, MaintenanceMarginRatio: 0.005, Cumulative: 0.005},
		},
	}
	for _, tc := range []struct {
		name string
		pair currency.Code
		exp  []NotionalBracketData
	}{
		{"pair", currency.NewCode("BTCUSD"), []NotionalBracketData{btcusd}},
		{"every pair", currency.EMPTYCODE, []NotionalBracketData{btcusd, {
			Pair: "ETHUSD",
			Brackets: []NotionalBracket{
				{Bracket: 1, InitialLeverage: 75, QuantityCap: 150, MaintenanceMarginRatio: 0.0065},
				{Bracket: 2, InitialLeverage: 50, QuantityCap: 300, QuantityFloor: 150, MaintenanceMarginRatio: 0.01, Cumulative: 0.525},
			},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.FuturesNotionalBracket(t.Context(), tc.pair)
			require.NoError(t, err, "FuturesNotionalBracket must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "FuturesNotionalBracket should decode every field")
				return
			}
			assert.NotEmpty(t, result, "FuturesNotionalBracket should return brackets")
		})
	}
}

func TestGetFuturesBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesBasisData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesBasisData must reject a nil request")
	_, err = e.GetFuturesBasisData(t.Context(), &CFuturesStatisticsRequest{ContractType: "CURRENT_QUARTER", Period: kline.FiveMin})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesBasisData must reject an empty pair")
	_, err = e.GetFuturesBasisData(t.Context(), &CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), Period: kline.FiveMin})
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetFuturesBasisData must reject an empty contract type")
	_, err = e.GetFuturesBasisData(t.Context(), &CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER"})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval, "GetFuturesBasisData must reject an empty period")

	startTime, endTime := getTime()
	_, err = e.GetFuturesBasisData(t.Context(), &CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Period: kline.FiveMin, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesBasisData must reject a start time after the end time")

	result, err := e.GetFuturesBasisData(t.Context(), &CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Period: kline.FiveMin, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetFuturesBasisData must not error")
	if mockTests {
		exp := []FuturesBasisData{
			{
				IndexPrice:          85913.95737453,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0112,
				FuturesPrice:        86872.4,
				AnnualizedBasisRate: 0.0511,
				Basis:               958.44262547,
				Pair:                "BTCUSD",
				Timestamp:           types.Time(time.Unix(1791278100, 0)),
			},
			{
				IndexPrice:          85988.84769358,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0111,
				FuturesPrice:        86942.1,
				AnnualizedBasisRate: 0.0507,
				Basis:               953.25230642,
				Pair:                "BTCUSD",
				Timestamp:           types.Time(time.Unix(1791278400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetFuturesBasisData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesBasisData should return basis records")
}

func TestCFuturesServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.CFuturesServerTime(t.Context())
	require.NoError(t, err, "CFuturesServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1791298531317), result, "CFuturesServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "CFuturesServerTime should return the current time")
}

func TestGetFuturesAggregatedTradesList(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAggregatedTradesList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesAggregatedTradesList must reject a nil request")
	_, err = e.GetFuturesAggregatedTradesList(t.Context(), &CFuturesAggregatedTradesRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesAggregatedTradesList must reject an empty symbol")

	ensureTradablePairs(t)
	_, endTime := getTime()
	// Binance only serves the last 48 hours of aggregate trades, in windows shorter than an hour
	startTime := endTime.Add(-30 * time.Minute)
	_, err = e.GetFuturesAggregatedTradesList(t.Context(), &CFuturesAggregatedTradesRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesAggregatedTradesList must reject a start time after the end time")

	result, err := e.GetFuturesAggregatedTradesList(t.Context(), &CFuturesAggregatedTradesRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetFuturesAggregatedTradesList must not error")
	if mockTests {
		exp := []CFuturesAggregatedTrade{
			{
				AggregateTradeID: 499297222,
				Price:            86060.2,
				Quantity:         46,
				FirstTradeID:     1157454712,
				LastTradeID:      1157454714,
				Timestamp:        types.Time(time.UnixMilli(1791283360152)),
			},
			{
				AggregateTradeID: 499297223,
				Price:            86060.4,
				Quantity:         1,
				FirstTradeID:     1157454715,
				LastTradeID:      1157454715,
				Timestamp:        types.Time(time.UnixMilli(1791283360296)),
			},
			{
				AggregateTradeID: 499297224,
				Price:            86060.6,
				Quantity:         40,
				FirstTradeID:     1157454716,
				LastTradeID:      1157454716,
				Timestamp:        types.Time(time.UnixMilli(1791283360296)),
				IsBuyerMaker:     true,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesAggregatedTradesList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesAggregatedTradesList should return aggregate trades")
}

func TestGetContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetContinuousKlineData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetContinuousKlineData must reject a nil request")
	_, err = e.GetContinuousKlineData(t.Context(), &CFuturesContinuousKlineRequest{ContractType: "CURRENT_QUARTER", Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetContinuousKlineData must reject an empty pair")
	_, err = e.GetContinuousKlineData(t.Context(), &CFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), Interval: kline.OneHour})
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetContinuousKlineData must reject an empty contract type")
	_, err = e.GetContinuousKlineData(t.Context(), &CFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER"})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetContinuousKlineData must reject an empty interval")

	startTime, endTime := getTime()
	_, err = e.GetContinuousKlineData(t.Context(), &CFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Interval: kline.OneHour, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetContinuousKlineData must reject a start time after the end time")

	result, err := e.GetContinuousKlineData(t.Context(), &CFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Interval: kline.OneHour, StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "GetContinuousKlineData must not error")
	if mockTests {
		exp := []CFuturesCandleStick{
			{
				OpenTime:                types.Time(time.Unix(1744106400, 0)),
				Open:                    79836.5,
				High:                    80120.1,
				Low:                     79745.5,
				Close:                   79890.3,
				Volume:                  15708,
				CloseTime:               types.Time(time.UnixMilli(1744109999999)),
				BaseAssetVolume:         19.64641546,
				NumberOfTrades:          507,
				TakerBuyVolume:          8474,
				TakerBuyBaseAssetVolume: 10.60066335,
			},
			{
				OpenTime:                types.Time(time.Unix(1744110000, 0)),
				Open:                    79954.9,
				High:                    80999.2,
				Low:                     79927,
				Close:                   80916.5,
				Volume:                  24560,
				CloseTime:               types.Time(time.UnixMilli(1744113599999)),
				BaseAssetVolume:         30.511258,
				NumberOfTrades:          827,
				TakerBuyVolume:          14297,
				TakerBuyBaseAssetVolume: 17.75126477,
			},
		}
		assert.Equal(t, exp, result, "GetContinuousKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetContinuousKlineData should return klines")
}

func TestFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.FuturesExchangeInfo(t.Context())
	require.NoError(t, err, "FuturesExchangeInfo must not error")
	if mockTests {
		exp := &CFuturesExchangeInfoResponse{
			ExchangeFilters: []string{
				"EXCHANGE_MAX_NUM_ORDERS",
			},
			RateLimits: []RateLimitInfo{
				{
					RateLimitType: "REQUEST_WEIGHT",
					Interval:      "MINUTE",
					IntervalNum:   1,
					Limit:         2400,
				},
				{
					RateLimitType: "ORDERS",
					Interval:      "MINUTE",
					IntervalNum:   1,
					Limit:         1200,
				},
			},
			ServerTime: types.Time(time.UnixMilli(1791278101111)),
			Symbols: []CFuturesSymbolInfo{
				{
					Filters: []CFuturesSymbolFilter{
						{
							FilterType: "PRICE_FILTER",
							MaxPrice:   4520958,
							MinPrice:   1000,
							TickSize:   0.1,
						},
						{
							FilterType:  "LOT_SIZE",
							MaxQuantity: 1000000,
							MinQuantity: 1,
							StepSize:    1,
						},
						{
							FilterType:  "MARKET_LOT_SIZE",
							MaxQuantity: 60000,
							MinQuantity: 1,
							StepSize:    1,
						},
						{
							FilterType: "MAX_NUM_ORDERS",
							Limit:      200,
						},
						{
							FilterType: "MAX_NUM_ALGO_ORDERS",
							Limit:      20,
						},
						{
							FilterType:        "PERCENT_PRICE",
							MultiplierUp:      1.05,
							MultiplierDown:    0.95,
							MultiplierDecimal: 4,
						},
					},
					OrderTypes: []string{
						"LIMIT",
						"MARKET",
						"STOP",
						"STOP_MARKET",
						"TAKE_PROFIT",
						"TAKE_PROFIT_MARKET",
						"TRAILING_STOP_MARKET",
					},
					TimeInForce: []string{
						"GTC",
						"IOC",
						"FOK",
						"GTX",
					},
					LiquidationFee:           0.015,
					MarketTakeBound:          0.05,
					Symbol:                   "BTCUSD_PERP",
					Pair:                     "BTCUSD",
					ContractType:             "PERPETUAL",
					DeliveryDate:             types.Time(time.Unix(4133404800, 0)),
					OnboardDate:              types.Time(time.Unix(1597042800, 0)),
					ContractStatus:           "TRADING",
					ContractSize:             100,
					QuoteAsset:               currency.USD,
					BaseAsset:                currency.BTC,
					MarginAsset:              currency.BTC,
					PricePrecision:           1,
					BaseAssetPrecision:       8,
					QuotePrecision:           8,
					EqualQuantityPrecision:   4,
					TriggerProtect:           0.05,
					MaintenanceMarginPercent: 2.5,
					RequiredMarginPercent:    5,
					UnderlyingType:           "COIN",
					UnderlyingSubType: []string{
						"PoW",
					},
					MaxMoveOrderLimit: 10000,
					PermissionSets: []string{
						"GRID",
					},
				},
				{
					Filters: []CFuturesSymbolFilter{
						{
							FilterType: "PRICE_FILTER",
							MaxPrice:   4671848,
							MinPrice:   1000,
							TickSize:   0.1,
						},
						{
							FilterType:  "LOT_SIZE",
							MaxQuantity: 1000000,
							MinQuantity: 1,
							StepSize:    1,
						},
						{
							FilterType:  "MARKET_LOT_SIZE",
							MaxQuantity: 1000,
							MinQuantity: 1,
							StepSize:    1,
						},
						{
							FilterType: "MAX_NUM_ORDERS",
							Limit:      200,
						},
						{
							FilterType: "MAX_NUM_ALGO_ORDERS",
							Limit:      20,
						},
						{
							FilterType:        "PERCENT_PRICE",
							MultiplierUp:      1.05,
							MultiplierDown:    0.95,
							MultiplierDecimal: 4,
						},
					},
					OrderTypes: []string{
						"LIMIT",
						"MARKET",
						"STOP",
						"STOP_MARKET",
						"TAKE_PROFIT",
						"TAKE_PROFIT_MARKET",
						"TRAILING_STOP_MARKET",
					},
					TimeInForce: []string{
						"GTC",
						"IOC",
						"FOK",
						"GTX",
					},
					LiquidationFee:           0.01,
					MarketTakeBound:          0.05,
					Symbol:                   "BTCUSD_261225",
					Pair:                     "BTCUSD",
					ContractType:             "CURRENT_QUARTER",
					DeliveryDate:             types.Time(time.Unix(1798185600, 0)),
					OnboardDate:              types.Time(time.Unix(1782460800, 0)),
					ContractStatus:           "TRADING",
					ContractSize:             100,
					QuoteAsset:               currency.USD,
					BaseAsset:                currency.BTC,
					MarginAsset:              currency.BTC,
					PricePrecision:           1,
					QuantityPrecision:        1,
					BaseAssetPrecision:       8,
					QuotePrecision:           8,
					EqualQuantityPrecision:   4,
					TriggerProtect:           0.05,
					MaintenanceMarginPercent: 2.5,
					RequiredMarginPercent:    5,
					UnderlyingType:           "COIN",
					UnderlyingSubType: []string{
						"PoW",
					},
					MaxMoveOrderLimit: 10000,
					PermissionSets: []string{
						"GRID",
					},
				},
			},
			Timezone: "UTC",
		}
		assert.Equal(t, exp, result, "FuturesExchangeInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbols, "FuturesExchangeInfo should return symbols")
}

func TestFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesGetFundingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesGetFundingHistory must reject a nil request")
	_, err = e.FuturesGetFundingHistory(t.Context(), &CFuturesFundingRateHistoryRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesGetFundingHistory must reject an empty symbol")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.FuturesGetFundingHistory(t.Context(), &CFuturesFundingRateHistoryRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FuturesGetFundingHistory must reject a start time after the end time")

	result, err := e.FuturesGetFundingHistory(t.Context(), &CFuturesFundingRateHistoryRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "FuturesGetFundingHistory must not error")
	if mockTests {
		exp := []CFuturesFundingRateHistory{
			{
				Symbol:      "BTCUSD_PERP",
				FundingTime: types.Time(time.Unix(1744128000, 0)),
				FundingRate: 0.00005596,
				MarkPrice:   78436.8,
				RateType:    "Regular",
			},
			{
				Symbol:      "BTCUSD_PERP",
				FundingTime: types.Time(time.UnixMilli(1744156800001)),
				FundingRate: 0.0001,
				MarkPrice:   76227.5,
				RateType:    "Regular",
			},
		}
		assert.Equal(t, exp, result, "FuturesGetFundingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesGetFundingHistory should return funding rates")
}

func TestGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetFundingRateInfo(t.Context())
	require.NoError(t, err, "GetFundingRateInfo must not error")
	if mockTests {
		exp := []CFuturesFundingRateInfo{
			{
				Symbol:                   "BTCUSD_PERP",
				AdjustedFundingRateCap:   0.025,
				AdjustedFundingRateFloor: -0.025,
				FundingIntervalHours:     8,
			},
			{
				Symbol:                   "SUIUSD_PERP",
				AdjustedFundingRateCap:   0.03,
				AdjustedFundingRateFloor: -0.03,
				FundingIntervalHours:     4,
				Disclaimer:               true,
			},
		}
		assert.Equal(t, exp, result, "GetFundingRateInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFundingRateInfo should return a list, which may be empty")
}

func TestGetIndexAndMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetIndexAndMarkPrice(t.Context(), currency.EMPTYPAIR, currency.NewCode("BTCUSD"))
	require.NoError(t, err, "GetIndexAndMarkPrice must not error")
	if mockTests {
		exp := []IndexMarkPrice{
			{
				Symbol:               "BTCUSD_PERP",
				Pair:                 "BTCUSD",
				MarkPrice:            86068.2,
				IndexPrice:           86099.38841172,
				EstimatedSettlePrice: 86149.1986673,
				LastFundingRate:      0.00007223,
				InterestRate:         0.0001,
				NextFundingTime:      types.Time(time.Unix(1791302400, 0)),
				Time:                 types.Time(time.Unix(1791285129, 0)),
			},
			{
				Symbol:               "BTCUSD_270326",
				Pair:                 "BTCUSD",
				MarkPrice:            88252.18329554,
				IndexPrice:           86099.38841172,
				EstimatedSettlePrice: 86149.1986673,
				Time:                 types.Time(time.Unix(1791285129, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetIndexAndMarkPrice should decode every field and keep only the pair's symbols")
		return
	}
	require.NotEmpty(t, result, "GetIndexAndMarkPrice must return prices")
	for i := range result {
		assert.Equalf(t, "BTCUSD", result[i].Pair, "GetIndexAndMarkPrice should return only the pair's symbols, not %s", result[i].Symbol)
	}
}

func TestGetIndexPriceKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexPriceKlines must reject a nil request")
	_, err = e.GetIndexPriceKlines(t.Context(), &CFuturesIndexPriceKlineRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIndexPriceKlines must reject an empty pair")
	_, err = e.GetIndexPriceKlines(t.Context(), &CFuturesIndexPriceKlineRequest{Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetIndexPriceKlines must reject an empty interval")

	startTime, endTime := getTime()
	_, err = e.GetIndexPriceKlines(t.Context(), &CFuturesIndexPriceKlineRequest{Pair: currency.NewCode("BTCUSD"), Interval: kline.OneHour, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetIndexPriceKlines must reject a start time after the end time")

	result, err := e.GetIndexPriceKlines(t.Context(), &CFuturesIndexPriceKlineRequest{Pair: currency.NewCode("BTCUSD"), Interval: kline.OneHour, StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "GetIndexPriceKlines must not error")
	if mockTests {
		exp := []CFuturesPriceCandleStick{
			{
				OpenTime:          types.Time(time.Unix(1744106400, 0)),
				Open:              78885.27746235,
				High:              79141.48690615,
				Low:               78804.59561981,
				Close:             78973.53978691,
				CloseTime:         types.Time(time.UnixMilli(1744109999999)),
				NumberOfBasicData: 3600,
			},
			{
				OpenTime:          types.Time(time.Unix(1744110000, 0)),
				Open:              78980.18168094,
				High:              80033.66369347,
				Low:               78954.84431711,
				Close:             79957.39937586,
				CloseTime:         types.Time(time.UnixMilli(1744113599999)),
				NumberOfBasicData: 3600,
			},
		}
		assert.Equal(t, exp, result, "GetIndexPriceKlines should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetIndexPriceKlines should return klines")
}

// symbolKlineTests checks the request validation shared by the coin margined futures kline endpoints that take a
// symbol, and returns the request their success case sends
func symbolKlineTests(t *testing.T, fn func(*CFuturesKlineRequest) error) *CFuturesKlineRequest {
	t.Helper()
	require.ErrorIs(t, fn(nil), common.ErrNilPointer, "a nil request must be rejected")
	require.ErrorIs(t, fn(&CFuturesKlineRequest{Interval: kline.OneHour}), currency.ErrCurrencyPairEmpty, "an empty symbol must be rejected")
	ensureTradablePairs(t)
	require.ErrorIs(t, fn(&CFuturesKlineRequest{Symbol: coinmTradablePair}), kline.ErrInvalidInterval, "an empty interval must be rejected")
	startTime, endTime := getTime()
	require.ErrorIs(t, fn(&CFuturesKlineRequest{Symbol: coinmTradablePair, Interval: kline.OneHour, StartTime: endTime, EndTime: startTime}), common.ErrStartAfterEnd, "a start time after the end time must be rejected")
	return &CFuturesKlineRequest{Symbol: coinmTradablePair, Interval: kline.OneHour, StartTime: startTime, EndTime: endTime, Limit: 5}
}

func TestGetFuturesKlineData(t *testing.T) {
	t.Parallel()
	req := symbolKlineTests(t, func(r *CFuturesKlineRequest) error {
		_, err := e.GetFuturesKlineData(t.Context(), r)
		return err
	})
	result, err := e.GetFuturesKlineData(t.Context(), req)
	require.NoError(t, err, "GetFuturesKlineData must not error")
	if mockTests {
		exp := []CFuturesCandleStick{
			{
				OpenTime:                types.Time(time.Unix(1744106400, 0)),
				Open:                    78844,
				High:                    79120,
				Low:                     78765,
				Close:                   78950,
				Volume:                  180128,
				CloseTime:               types.Time(time.UnixMilli(1744109999999)),
				BaseAssetVolume:         228.21023657,
				NumberOfTrades:          8942,
				TakerBuyVolume:          84148,
				TakerBuyBaseAssetVolume: 106.61711606,
			},
			{
				OpenTime:                types.Time(time.Unix(1744110000, 0)),
				Open:                    78951.1,
				High:                    80000,
				Low:                     78910.1,
				Close:                   79916.1,
				Volume:                  818611,
				CloseTime:               types.Time(time.UnixMilli(1744113599999)),
				BaseAssetVolume:         1028.54016988,
				NumberOfTrades:          23993,
				TakerBuyVolume:          561352,
				TakerBuyBaseAssetVolume: 705.11758224,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesKlineData should return klines")
}

// futuresStatisticsTests checks the request validation shared by the coin margined futures statistics endpoints, and
// returns the request their success case sends
func futuresStatisticsTests(t *testing.T, contractType string, fn func(*CFuturesStatisticsRequest) error) *CFuturesStatisticsRequest {
	t.Helper()
	require.ErrorIs(t, fn(nil), common.ErrNilPointer, "a nil request must be rejected")
	require.ErrorIs(t, fn(&CFuturesStatisticsRequest{ContractType: contractType, Period: kline.FiveMin}), currency.ErrCurrencyPairEmpty, "an empty pair must be rejected")
	require.ErrorIs(t, fn(&CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: contractType}), errInvalidPeriodOrInterval, "an empty period must be rejected")
	startTime, endTime := getTime()
	require.ErrorIs(t, fn(&CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: contractType, Period: kline.FiveMin, StartTime: endTime, EndTime: startTime}), common.ErrStartAfterEnd, "a start time after the end time must be rejected")
	return &CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: contractType, Period: kline.FiveMin, StartTime: startTime, EndTime: endTime}
}

func TestGetMarketRatio(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "", func(r *CFuturesStatisticsRequest) error {
		_, err := e.GetMarketRatio(t.Context(), r)
		return err
	})
	result, err := e.GetMarketRatio(t.Context(), req)
	require.NoError(t, err, "GetMarketRatio must not error")
	if mockTests {
		exp := []TopTraderAccountRatio{
			{
				Pair:           "BTCUSD",
				LongShortRatio: 3.662,
				LongAccount:    0.7844,
				ShortAccount:   0.2142,
				Timestamp:      types.Time(time.Unix(1791278100, 0)),
			},
			{
				Pair:           "BTCUSD",
				LongShortRatio: 3.6447,
				LongAccount:    0.7836,
				ShortAccount:   0.215,
				Timestamp:      types.Time(time.Unix(1791278400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetMarketRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetMarketRatio should return ratios")
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	req := symbolKlineTests(t, func(r *CFuturesKlineRequest) error {
		_, err := e.GetMarkPriceKline(t.Context(), r)
		return err
	})
	result, err := e.GetMarkPriceKline(t.Context(), req)
	require.NoError(t, err, "GetMarkPriceKline must not error")
	if mockTests {
		exp := []CFuturesPriceCandleStick{
			{
				OpenTime:          types.Time(time.Unix(1744106400, 0)),
				Open:              78844.32274145,
				High:              79119.9,
				Low:               78765,
				Close:             78940.2,
				CloseTime:         types.Time(time.UnixMilli(1744109999999)),
				NumberOfBasicData: 3600,
			},
			{
				OpenTime:          types.Time(time.Unix(1744110000, 0)),
				Open:              78946.81050573,
				High:              79999.9,
				Low:               78921.86379987,
				Close:             79916.1,
				CloseTime:         types.Time(time.UnixMilli(1744113599999)),
				NumberOfBasicData: 3600,
			},
		}
		assert.Equal(t, exp, result, "GetMarkPriceKline should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetMarkPriceKline should return klines")
}

func TestGetFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesHistoricalTrades(t.Context(), currency.EMPTYPAIR, 0, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesHistoricalTrades must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	result, err := e.GetFuturesHistoricalTrades(t.Context(), coinmTradablePair, 0, 5)
	require.NoError(t, err, "GetFuturesHistoricalTrades must not error")
	if mockTests {
		exp := []FuturesPublicTradesData{
			{
				ID:           1157458431,
				Price:        86118.7,
				Quantity:     1,
				BaseQuantity: 0.00116118,
				Time:         types.Time(time.UnixMilli(1791285176377)),
			},
			{
				ID:           1157458432,
				Price:        86118.7,
				Quantity:     180,
				BaseQuantity: 0.20901383,
				Time:         types.Time(time.UnixMilli(1791285176377)),
				IsBuyerMaker: true,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesHistoricalTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesHistoricalTrades should return trades")
}

func TestOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.OpenInterest(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "OpenInterest must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.OpenInterest(t.Context(), coinmTradablePair)
	require.NoError(t, err, "OpenInterest must not error")
	if mockTests {
		exp := &CFuturesOpenInterestResponse{
			Symbol:       "BTCUSD_PERP",
			Pair:         "BTCUSD",
			OpenInterest: 12669867,
			ContractType: "PERPETUAL",
			Time:         types.Time(time.UnixMilli(1791282919818)),
		}
		assert.Equal(t, exp, result, "OpenInterest should decode every field")
		return
	}
	assert.NotNil(t, result, "OpenInterest should return open interest")
}

func TestGetOpenInterestStats(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "CURRENT_QUARTER", func(r *CFuturesStatisticsRequest) error {
		_, err := e.GetOpenInterestStats(t.Context(), r)
		return err
	})
	result, err := e.GetOpenInterestStats(t.Context(), req)
	require.NoError(t, err, "GetOpenInterestStats must not error")
	if mockTests {
		exp := []OpenInterestStats{
			{
				Pair:                 "BTCUSD",
				ContractType:         "CURRENT_QUARTER",
				SumOpenInterest:      212983,
				SumOpenInterestValue: 245.17046161,
				Timestamp:            types.Time(time.Unix(1791278100, 0)),
			},
			{
				Pair:                 "BTCUSD",
				ContractType:         "CURRENT_QUARTER",
				SumOpenInterest:      212974,
				SumOpenInterestValue: 244.95863332,
				Timestamp:            types.Time(time.Unix(1791278400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetOpenInterestStats should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetOpenInterestStats should return statistics")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesOrderbook must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.GetFuturesOrderbook(t.Context(), coinmTradablePair, 1000)
	require.NoError(t, err, "GetFuturesOrderbook must not error")
	if mockTests {
		exp := &CFuturesOrderBookResponse{
			LastUpdateID:      11747177501814,
			Symbol:            "BTCUSD_PERP",
			Pair:              "BTCUSD",
			MessageOutputTime: types.Time(time.UnixMilli(1791285170291)),
			TransactionTime:   types.Time(time.UnixMilli(1791285170281)),
			Bids: []orderbook.Level{
				{
					Amount: 4217,
					Price:  86110.6,
				},
				{
					Amount: 13,
					Price:  86110.5,
				},
				{
					Amount: 1,
					Price:  86109.3,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 768,
					Price:  86110.7,
				},
				{
					Amount: 1,
					Price:  86111.4,
				},
				{
					Amount: 2,
					Price:  86112.6,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetFuturesOrderbook should return bids")
}

func TestGetPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	req := symbolKlineTests(t, func(r *CFuturesKlineRequest) error {
		_, err := e.GetPremiumIndexKlineData(t.Context(), r)
		return err
	})
	result, err := e.GetPremiumIndexKlineData(t.Context(), req)
	require.NoError(t, err, "GetPremiumIndexKlineData must not error")
	if mockTests {
		exp := []CFuturesPremiumIndexCandleStick{
			{
				OpenTime:  types.Time(time.Unix(1744106400, 0)),
				Open:      -0.00052326,
				High:      -0.00010773,
				Low:       -0.00074901,
				Close:     -0.00041092,
				CloseTime: types.Time(time.UnixMilli(1744109999999)),
			},
			{
				OpenTime:  types.Time(time.Unix(1744110000, 0)),
				Open:      -0.00028212,
				High:      0.00004144,
				Low:       -0.00073853,
				Close:     -0.00049728,
				CloseTime: types.Time(time.UnixMilli(1744113599999)),
			},
		}
		assert.Equal(t, exp, result, "GetPremiumIndexKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPremiumIndexKlineData should return klines")
}

func TestGetCFuturesIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := e.GetCFuturesIndexPriceConstituents(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCFuturesIndexPriceConstituents must reject an empty symbol")

	result, err := e.GetCFuturesIndexPriceConstituents(t.Context(), currency.NewCode("BTCUSD"))
	require.NoError(t, err, "GetCFuturesIndexPriceConstituents must not error")
	if mockTests {
		exp := &CFuturesIndexPriceConstituentsResponse{
			Symbol: "BTCUSD",
			Time:   types.Time(time.UnixMilli(1791282887128)),
			Constituents: []CFuturesIndexPriceConstituent{
				{
					Exchange: "binance_cross",
					Symbol:   "BTCUSDT*uindex(USDTUSD)",
					Price:    86029.70849087,
					Weight:   0.33333333,
				},
				{
					Exchange: "okex",
					Symbol:   "BTC-USDT*uindex(USDTUSD)",
					Price:    86027.99909296,
					Weight:   0.16666667,
				},
				{
					Exchange: "coinbase",
					Symbol:   "BTC-USD",
					Price:    86037.49,
					Weight:   0.33333333,
				},
				{
					Exchange: "bitstamp",
					Symbol:   "btcusd",
					Price:    86028.25,
					Weight:   0.16666667,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCFuturesIndexPriceConstituents should decode every field")
		return
	}
	assert.NotEmpty(t, result.Constituents, "GetCFuturesIndexPriceConstituents should return constituents")
}

func TestGetFuturesPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPublicTrades(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPublicTrades must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.GetFuturesPublicTrades(t.Context(), coinmTradablePair, 5)
	require.NoError(t, err, "GetFuturesPublicTrades must not error")
	if mockTests {
		exp := []FuturesPublicTradesData{
			{
				ID:           1157458431,
				Price:        86118.7,
				Quantity:     1,
				BaseQuantity: 0.00116118,
				Time:         types.Time(time.UnixMilli(1791285176377)),
			},
			{
				ID:           1157458434,
				Price:        86118.7,
				Quantity:     20,
				BaseQuantity: 0.02322375,
				Time:         types.Time(time.UnixMilli(1791285176377)),
				IsBuyerMaker: true,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesPublicTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesPublicTrades should return trades")
}

func TestGetFuturesOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbookTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), currency.NewCode("BTCUSD"))
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetFuturesOrderbookTicker must reject a symbol and a pair together")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []SymbolOrderBookTicker
	}{
		{"symbol", coinmTradablePair, []SymbolOrderBookTicker{
			{
				LastUpdateID: 11746981095251,
				Symbol:       "BTCUSD_PERP",
				Pair:         "BTCUSD",
				BidPrice:     86002.7,
				BidQuantity:  3129,
				AskPrice:     86002.8,
				AskQuantity:  2247,
				Time:         types.Time(time.UnixMilli(1791282926732)),
			},
		}},
		{"every symbol", currency.EMPTYPAIR, []SymbolOrderBookTicker{
			{
				LastUpdateID: 11747177394191,
				Symbol:       "BTCUSD_PERP",
				Pair:         "BTCUSD",
				BidPrice:     86110.6,
				BidQuantity:  4218,
				AskPrice:     86110.7,
				AskQuantity:  1013,
				Time:         types.Time(time.UnixMilli(1791285168243)),
			},
			{
				LastUpdateID: 11747177398717,
				Symbol:       "AAVEUSD_PERP",
				Pair:         "AAVEUSD",
				BidPrice:     180.84,
				BidQuantity:  63,
				AskPrice:     180.93,
				AskQuantity:  1,
				Time:         types.Time(time.UnixMilli(1791285168341)),
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesOrderbookTicker(t.Context(), tc.symbol, currency.EMPTYCODE)
			require.NoError(t, err, "GetFuturesOrderbookTicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesOrderbookTicker should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetFuturesOrderbookTicker should return tickers")
		})
	}
}

func TestGetFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesSymbolPriceTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), currency.NewCode("BTCUSD"))
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetFuturesSymbolPriceTicker must reject a symbol and a pair together")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		pair   currency.Code
		exp    []SymbolPriceTicker
	}{
		{"symbol", coinmTradablePair, currency.EMPTYCODE, []SymbolPriceTicker{
			{Symbol: "BTCUSD_PERP", Pair: "BTCUSD", Price: 86002.8, Time: types.Time(time.UnixMilli(1791282919817))},
		}},
		{"pair", currency.EMPTYPAIR, currency.NewCode("BTCUSD"), []SymbolPriceTicker{
			{Symbol: "BTCUSD_PERP", Pair: "BTCUSD", Price: 86110.7, Time: types.Time(time.UnixMilli(1791285170154))},
			{Symbol: "BTCUSD_270326", Pair: "BTCUSD", Price: 88215.3, Time: types.Time(time.UnixMilli(1791285054739))},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesSymbolPriceTicker(t.Context(), tc.symbol, tc.pair)
			require.NoError(t, err, "GetFuturesSymbolPriceTicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesSymbolPriceTicker should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetFuturesSymbolPriceTicker should return tickers")
		})
	}
}

func TestGetFuturesTakerVolume(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "ALL", func(r *CFuturesStatisticsRequest) error {
		_, err := e.GetFuturesTakerVolume(t.Context(), r)
		return err
	})
	_, err := e.GetFuturesTakerVolume(t.Context(), &CFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), Period: kline.FiveMin})
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetFuturesTakerVolume must reject an empty contract type")

	result, err := e.GetFuturesTakerVolume(t.Context(), req)
	require.NoError(t, err, "GetFuturesTakerVolume must not error")
	if mockTests {
		exp := []TakerBuySellVolume{
			{
				Pair:                 "BTCUSD",
				ContractType:         "ALL",
				TakerBuyVolume:       11049,
				TakerSellVolume:      24801,
				TakerBuyVolumeValue:  0.1286,
				TakerSellVolumeValue: 0.2888,
				Timestamp:            types.Time(time.Unix(1791277800, 0)),
			},
			{
				Pair:                 "BTCUSD",
				ContractType:         "ALL",
				TakerBuyVolume:       8673,
				TakerSellVolume:      2939,
				TakerBuyVolumeValue:  0.1009,
				TakerSellVolumeValue: 0.0342,
				Timestamp:            types.Time(time.Unix(1791278100, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetFuturesTakerVolume should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesTakerVolume should return volumes")
}

func TestGetFuturesSwapTickerChangeStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), currency.NewCode("BTCUSD"))
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetFuturesSwapTickerChangeStats must reject a symbol and a pair together")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []CFuturesPriceChangeStats
	}{
		{"symbol", coinmTradablePair, []CFuturesPriceChangeStats{
			{
				Symbol:               "BTCUSD_PERP",
				Pair:                 "BTCUSD",
				PriceChange:          -41,
				PriceChangePercent:   -0.048,
				WeightedAveragePrice: 85677.28191708,
				LastPrice:            85994.6,
				LastQuantity:         1,
				OpenPrice:            86035.6,
				HighPrice:            86664.7,
				LowPrice:             84911.4,
				Volume:               7343590,
				BaseVolume:           8571.2,
				OpenTime:             types.Time(time.Unix(1791196500, 0)),
				CloseTime:            types.Time(time.UnixMilli(1791282919746)),
				FirstID:              1157292594,
				LastID:               1157454183,
				Count:                161590,
			},
		}},
		{"every symbol", currency.EMPTYPAIR, []CFuturesPriceChangeStats{
			{
				Symbol:               "BTCUSD_PERP",
				Pair:                 "BTCUSD",
				PriceChange:          229.7,
				PriceChangePercent:   0.267,
				WeightedAveragePrice: 85683.47028219,
				LastPrice:            86110.7,
				LastQuantity:         104,
				OpenPrice:            85881,
				HighPrice:            86664.7,
				LowPrice:             84911.4,
				Volume:               7427276,
				BaseVolume:           8668.3,
				OpenTime:             types.Time(time.Unix(1791198720, 0)),
				CloseTime:            types.Time(time.UnixMilli(1791285170154)),
				FirstID:              1157295222,
				LastID:               1157458420,
				Count:                163199,
			},
			{
				Symbol:               "ETHUSD_PERP",
				Pair:                 "ETHUSD",
				PriceChange:          0.66,
				PriceChangePercent:   0.024,
				WeightedAveragePrice: 2704.01324921,
				LastPrice:            2710.63,
				LastQuantity:         5,
				OpenPrice:            2709.97,
				HighPrice:            2729.98,
				LowPrice:             2676.7,
				Volume:               35914142,
				BaseVolume:           132817.92,
				OpenTime:             types.Time(time.Unix(1791198720, 0)),
				CloseTime:            types.Time(time.UnixMilli(1791285160251)),
				FirstID:              1240782270,
				LastID:               1240881063,
				Count:                98794,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesSwapTickerChangeStats(t.Context(), tc.symbol, currency.EMPTYCODE)
			require.NoError(t, err, "GetFuturesSwapTickerChangeStats must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesSwapTickerChangeStats should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetFuturesSwapTickerChangeStats should return statistics")
		})
	}
}

func TestGetTraderFuturesAccountRatio(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "", func(r *CFuturesStatisticsRequest) error {
		_, err := e.GetTraderFuturesAccountRatio(t.Context(), r)
		return err
	})
	result, err := e.GetTraderFuturesAccountRatio(t.Context(), req)
	require.NoError(t, err, "GetTraderFuturesAccountRatio must not error")
	if mockTests {
		exp := []TopTraderAccountRatio{
			{
				Pair:           "BTCUSD",
				LongShortRatio: 3.946,
				LongAccount:    0.7967,
				ShortAccount:   0.2019,
				Timestamp:      types.Time(time.Unix(1791278100, 0)),
			},
			{
				Pair:           "BTCUSD",
				LongShortRatio: 3.9455,
				LongAccount:    0.7966,
				ShortAccount:   0.2019,
				Timestamp:      types.Time(time.Unix(1791278400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetTraderFuturesAccountRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetTraderFuturesAccountRatio should return ratios")
}

func TestGetTraderFuturesPositionsRatio(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "", func(r *CFuturesStatisticsRequest) error {
		_, err := e.GetTraderFuturesPositionsRatio(t.Context(), r)
		return err
	})
	result, err := e.GetTraderFuturesPositionsRatio(t.Context(), req)
	require.NoError(t, err, "GetTraderFuturesPositionsRatio must not error")
	if mockTests {
		exp := []TopTraderPositionRatio{
			{
				Pair:           "BTCUSD",
				LongShortRatio: 2.3599,
				LongPosition:   0.7024,
				ShortPosition:  0.2976,
				Timestamp:      types.Time(time.Unix(1791278100, 0)),
			},
			{
				Pair:           "BTCUSD",
				LongShortRatio: 2.3645,
				LongPosition:   0.7028,
				ShortPosition:  0.2972,
				Timestamp:      types.Time(time.Unix(1791278400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetTraderFuturesPositionsRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetTraderFuturesPositionsRatio should return ratios")
}

func TestCheckSymbolOrPair(t *testing.T) {
	t.Parallel()
	symbol := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name    string
		symbol  currency.Pair
		pair    currency.Code
		orderID uint64
		err     error
	}{
		{"neither", currency.EMPTYPAIR, currency.EMPTYCODE, 0, errSymbolOrPairRequired},
		{"order ID with pair", currency.EMPTYPAIR, currency.NewCode("BTCUSD"), 1, errOrderIDRequiresSymbol},
		{"order ID with symbol", symbol, currency.EMPTYCODE, 1, nil},
		{"pair", currency.EMPTYPAIR, currency.NewCode("BTCUSD"), 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, checkSymbolOrPair(tc.symbol, tc.pair, tc.orderID), tc.err, "checkSymbolOrPair should return the expected error")
		})
	}
}

func TestFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesTradeHistory must reject a nil request")
	_, err = e.FuturesTradeHistory(t.Context(), &CFuturesAccountTradeListRequest{})
	require.ErrorIs(t, err, errSymbolOrPairRequired, "FuturesTradeHistory must require a symbol or a pair")
	ensureTradablePairs(t)
	_, err = e.FuturesTradeHistory(t.Context(), &CFuturesAccountTradeListRequest{Symbol: coinmTradablePair, Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "FuturesTradeHistory must reject a symbol and a pair together")
	_, err = e.FuturesTradeHistory(t.Context(), &CFuturesAccountTradeListRequest{Pair: currency.NewCode("BTCUSD"), FromID: 1})
	require.ErrorIs(t, err, errFromIDMutuallyExclusive, "FuturesTradeHistory must reject a from ID with a pair")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.FuturesTradeHistory(t.Context(), &CFuturesAccountTradeListRequest{Symbol: coinmTradablePair, StartTime: startTime, FromID: 1})
	require.ErrorIs(t, err, errFromIDMutuallyExclusive, "FuturesTradeHistory must reject a from ID with a time window")
	_, err = e.FuturesTradeHistory(t.Context(), &CFuturesAccountTradeListRequest{Symbol: coinmTradablePair, StartTime: endTime, EndTime: startTime, Limit: 5})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FuturesTradeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// The endpoint also returns USDⓈ-M trades since the integration, which carry a quote quantity instead of a base one
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		pair   currency.Code
		exp    []FuturesAccountTradeList
	}{
		{"coin margined symbol", coinmTradablePair, currency.EMPTYCODE, []FuturesAccountTradeList{
			{
				Symbol:          "BTCUSD_PERP",
				ID:              1157458432,
				OrderID:         28,
				Pair:            "BTCUSD",
				Side:            "SELL",
				Price:           86118.7,
				Quantity:        180,
				RealizedPNL:     0.00001574,
				MarginAsset:     currency.BTC,
				BaseQuantity:    0.20901383,
				Commission:      0.0001045,
				CommissionAsset: currency.BTC,
				Time:            types.Time(time.Unix(1744120800, 0)),
				PositionSide:    "BOTH",
				Maker:           true,
			},
			{
				Symbol:          "BTCUSD_PERP",
				ID:              1157458433,
				OrderID:         29,
				Pair:            "BTCUSD",
				Side:            "BUY",
				Price:           86120.1,
				Quantity:        1,
				MarginAsset:     currency.BTC,
				BaseQuantity:    0.00116116,
				Commission:      0.00000058,
				CommissionAsset: currency.BTC,
				Time:            types.Time(time.Unix(1744120801, 0)),
				PositionSide:    "BOTH",
				Buyer:           true,
			},
		}},
		{"USDⓈ-M pair", currency.EMPTYPAIR, currency.NewCode("BTCUSDT"), []FuturesAccountTradeList{
			{
				Symbol:          "BTCUSDT",
				ID:              6096753380,
				OrderID:         8886774,
				Pair:            "BTCUSDT",
				Side:            "BUY",
				Price:           78950.1,
				Quantity:        0.002,
				RealizedPNL:     1.20532,
				MarginAsset:     currency.USDT,
				QuoteQuantity:   157.9002,
				Commission:      0.06316008,
				CommissionAsset: currency.USDT,
				Time:            types.Time(time.Unix(1744120800, 0)),
				PositionSide:    "BOTH",
				Buyer:           true,
				Maker:           true,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.FuturesTradeHistory(t.Context(), &CFuturesAccountTradeListRequest{Symbol: tc.symbol, Pair: tc.pair, StartTime: startTime, EndTime: endTime, Limit: 5})
			require.NoError(t, err, "FuturesTradeHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "FuturesTradeHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "FuturesTradeHistory should return trades")
		})
	}
}

func TestGetAllFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllFuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllFuturesOrders must reject a nil request")
	_, err = e.GetAllFuturesOrders(t.Context(), &CFuturesAllOrdersRequest{Pair: currency.NewCode("BTCUSD"), OrderID: 1})
	require.ErrorIs(t, err, errOrderIDRequiresSymbol, "GetAllFuturesOrders must reject an order ID with a pair")
	ensureTradablePairs(t)
	_, err = e.GetAllFuturesOrders(t.Context(), &CFuturesAllOrdersRequest{Symbol: coinmTradablePair, Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetAllFuturesOrders must reject a symbol and a pair together")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.GetAllFuturesOrders(t.Context(), &CFuturesAllOrdersRequest{Symbol: coinmTradablePair, StartTime: endTime, EndTime: startTime, Limit: 2})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllFuturesOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllFuturesOrders(t.Context(), &CFuturesAllOrdersRequest{Symbol: coinmTradablePair, StartTime: startTime, EndTime: endTime, Limit: 2})
	require.NoError(t, err, "GetAllFuturesOrders must not error")
	if mockTests {
		exp := []FuturesOrderData{
			{
				AveragePrice:            9022.5,
				ClientOrderID:           "abc",
				CumulativeBase:          0.04432225,
				ExecutedQuantity:        4,
				OrderID:                 1917641,
				OriginalQuantity:        10,
				OriginalType:            "TRAILING_STOP_MARKET",
				Price:                   9020.5,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "SHORT",
				Status:                  "PARTIALLY_FILLED",
				StopPrice:               9300,
				ClosePosition:           true,
				Symbol:                  "BTCUSD_PERP",
				Pair:                    "BTCUSD",
				Time:                    types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:             "GTD",
				OrderType:               "TRAILING_STOP_MARKET",
				ActivatePrice:           9020,
				PriceRate:               0.3,
				UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
				WorkingType:             "CONTRACT_PRICE",
				PriceProtect:            true,
				PriceMatch:              "OPPONENT",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				CumulativeQuote:         400,
				GoodTillDate:            types.Time(time.Unix(1579385751, 0)),
			},
			{
				AveragePrice:            9022.5,
				ClientOrderID:           "gct-cm-filled",
				CumulativeBase:          0.11080556,
				ExecutedQuantity:        10,
				OrderID:                 1917642,
				OriginalQuantity:        10,
				OriginalType:            "LIMIT",
				Price:                   9020.5,
				Side:                    "BUY",
				PositionSide:            "SHORT",
				Status:                  "FILLED",
				Symbol:                  "BTCUSD_PERP",
				Pair:                    "BTCUSD",
				Time:                    types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:             "GTC",
				OrderType:               "LIMIT",
				UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
				WorkingType:             "CONTRACT_PRICE",
				PriceMatch:              "NONE",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				CumulativeQuote:         1000,
			},
		}
		assert.Equal(t, exp, result, "GetAllFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllFuturesOrders should return orders")
}

func TestAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AutoCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR, 30*time.Second)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AutoCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.AutoCancelAllOpenOrders(t.Context(), coinmTradablePair, 30*time.Second)
	require.NoError(t, err, "AutoCancelAllOpenOrders must not error")
	if mockTests {
		exp := &CFuturesAutoCancelAllOpenOrdersResponse{
			Symbol:        "BTCUSD_PERP",
			CountdownTime: 30000,
		}
		assert.Equal(t, exp, result, "AutoCancelAllOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "AutoCancelAllOpenOrders should return the countdown")
}

func TestFuturesCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.FuturesCancelAllOpenOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "FuturesCancelAllOpenOrders must not error")
	if mockTests {
		exp := &CFuturesStatusResponse{
			Code:    200,
			Message: "The operation of cancel all open order is done.",
		}
		assert.Equal(t, exp, result, "FuturesCancelAllOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesCancelAllOpenOrders should return the status")
}

func TestFuturesOrderParams(t *testing.T) {
	t.Parallel()
	_, err := e.futuresOrderParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "futuresOrderParams must reject a nil request")
	_, err = e.futuresOrderParams(&FuturesNewOrderRequest{Side: "BUY", OrderType: "LIMIT"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "futuresOrderParams must reject an empty symbol")
	symbol := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	_, err = e.futuresOrderParams(&FuturesNewOrderRequest{Symbol: symbol, OrderType: "LIMIT"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "futuresOrderParams must reject an empty side")
	_, err = e.futuresOrderParams(&FuturesNewOrderRequest{Symbol: symbol, Side: "BUY"})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "futuresOrderParams must reject an empty order type")

	params, err := e.futuresOrderParams(&FuturesNewOrderRequest{
		Symbol:                  symbol,
		Side:                    "SELL",
		PositionSide:            "SHORT",
		OrderType:               "TRAILING_STOP_MARKET",
		TimeInForce:             "GTC",
		Quantity:                1000000,
		Price:                   0.00001,
		ReduceOnly:              true,
		NewClientOrderID:        "testOrder",
		StopPrice:               9300.5,
		ClosePosition:           true,
		ActivationPrice:         9020.25,
		CallbackRate:            0.3,
		WorkingType:             "MARK_PRICE",
		PriceProtect:            true,
		NewOrderRespType:        "RESULT",
		PriceMatch:              "OPPONENT",
		SelfTradePreventionMode: "EXPIRE_BOTH",
	})
	require.NoError(t, err, "futuresOrderParams must not error")
	exp := url.Values{
		"symbol":                  {"BTCUSD_PERP"},
		"side":                    {"SELL"},
		"positionSide":            {"SHORT"},
		"type":                    {"TRAILING_STOP_MARKET"},
		"timeInForce":             {"GTC"},
		"quantity":                {"1000000"},
		"price":                   {"0.00001"},
		"reduceOnly":              {"true"},
		"newClientOrderId":        {"testOrder"},
		"stopPrice":               {"9300.5"},
		"closePosition":           {"true"},
		"activationPrice":         {"9020.25"},
		"callbackRate":            {"0.3"},
		"workingType":             {"MARK_PRICE"},
		"priceProtect":            {"true"},
		"newOrderRespType":        {"RESULT"},
		"priceMatch":              {"OPPONENT"},
		"selfTradePreventionMode": {"EXPIRE_BOTH"},
	}
	assert.Equal(t, exp, params, "futuresOrderParams should send every parameter in plain decimal notation")

	params, err = e.futuresOrderParams(&FuturesNewOrderRequest{Symbol: symbol, Side: "BUY", OrderType: "MARKET"})
	require.NoError(t, err, "futuresOrderParams must not error")
	assert.Equal(t, url.Values{"symbol": {"BTCUSD_PERP"}, "side": {"BUY"}, "type": {"MARKET"}}, params, "futuresOrderParams should omit unset parameters")
}

func TestFuturesModifyMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesModifyMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "FuturesModifyMultipleOrders must reject no orders")
	ensureTradablePairs(t)
	_, err = e.FuturesModifyMultipleOrders(t.Context(), []CFuturesModifyOrderRequest{{Symbol: coinmTradablePair, OrderID: 1, Side: order.Buy.String(), Quantity: 1, PriceMatch: "QUEUE"}})
	require.ErrorIs(t, err, errBatchModifyPriceMatch, "FuturesModifyMultipleOrders must reject a price match")
	_, err = e.FuturesModifyMultipleOrders(t.Context(), []CFuturesModifyOrderRequest{{Symbol: coinmTradablePair, OrderID: 1, Side: order.Buy.String()}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FuturesModifyMultipleOrders must reject an order without a quantity")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FuturesModifyMultipleOrders(t.Context(), []CFuturesModifyOrderRequest{
		{Symbol: coinmTradablePair, OrderID: 20072994037, Side: order.Buy.String(), Quantity: 2, Price: 30005, ModifyID: 1},
		{Symbol: coinmTradablePair, OrigClientOrderID: "myOrder1", Side: order.Sell.String(), Quantity: 1, Price: 30006},
	})
	require.NoError(t, err, "FuturesModifyMultipleOrders must not error")
	if mockTests {
		exp := []CFuturesModifyBatchOrderResponse{
			{
				OrderID:                 20072994037,
				Symbol:                  "BTCUSD_PERP",
				Pair:                    "BTCUSD",
				Status:                  "PARTIALLY_FILLED",
				ClientOrderID:           "LJ9R4QZDihCaS8UAOOLpgW",
				ModifyID:                1,
				Price:                   30005,
				OriginalQuantity:        2,
				ExecutedQuantity:        1,
				CumulativeQuantity:      1,
				TimeInForce:             "GTX",
				OrderType:               "LIMIT",
				ReduceOnly:              true,
				ClosePosition:           true,
				Side:                    "BUY",
				PositionSide:            "LONG",
				StopPrice:               29000,
				WorkingType:             "MARK_PRICE",
				PriceProtect:            true,
				OriginalType:            "LIMIT",
				PriceMatch:              "QUEUE",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				UpdateTime:              types.Time(time.UnixMilli(1629182711600)),
			},
			{Code: -2022, Message: "ReduceOnly Order is rejected."},
		}
		assert.Equal(t, exp, result, "FuturesModifyMultipleOrders should decode every field and each rejection")
	}
}

func TestFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "FuturesBatchOrder must reject an empty batch")
	_, err = e.FuturesBatchOrder(t.Context(), []FuturesNewOrderRequest{{Side: "SELL", OrderType: "STOP_MARKET", StopPrice: 60000, ClosePosition: true}})
	require.ErrorIs(t, err, errBatchClosePosition, "FuturesBatchOrder must reject a close position order")
	_, err = e.FuturesBatchOrder(t.Context(), []FuturesNewOrderRequest{{Side: "BUY", OrderType: "LIMIT"}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FuturesBatchOrder must reject an order without a quantity")
	_, err = e.FuturesBatchOrder(t.Context(), []FuturesNewOrderRequest{{Side: "BUY", OrderType: "LIMIT", Quantity: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesBatchOrder must reject an order without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.FuturesBatchOrder(t.Context(), []FuturesNewOrderRequest{
		{Symbol: coinmTradablePair, Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 1, Price: 10001},
		{Symbol: coinmTradablePair, Side: "SELL", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 1, Price: 9000, ReduceOnly: true},
	})
	require.NoError(t, err, "FuturesBatchOrder must not error")
	if mockTests {
		exp := []CFuturesBatchOrderData{
			{
				ClientOrderID:           "testOrder1",
				CumulativeQuantity:      1,
				ExecutedQuantity:        1,
				OrderID:                 22542180,
				OriginalQuantity:        1,
				Price:                   10001,
				ReduceOnly:              true,
				ClosePosition:           true,
				Side:                    "BUY",
				PositionSide:            "BOTH",
				Status:                  "FILLED",
				StopPrice:               9300,
				Symbol:                  "BTCUSD_PERP",
				Pair:                    "BTCUSD",
				TimeInForce:             "GTC",
				OrderType:               "LIMIT",
				OriginalType:            "LIMIT",
				ActivatePrice:           9020,
				PriceRate:               0.3,
				UpdateTime:              types.Time(time.UnixMilli(1566818724722)),
				WorkingType:             "CONTRACT_PRICE",
				PriceProtect:            true,
				PriceMatch:              "OPPONENT",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
			{
				Code:    -2022,
				Message: "ReduceOnly Order is rejected.",
			},
		}
		assert.Equal(t, exp, result, "FuturesBatchOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result, "FuturesBatchOrder should return an entry per order")
}

func TestFuturesBatchCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesBatchCancelOrders(t.Context(), currency.EMPTYPAIR, []uint64{283194212}, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesBatchCancelOrders must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.FuturesBatchCancelOrders(t.Context(), coinmTradablePair, nil, nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "FuturesBatchCancelOrders must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FuturesBatchCancelOrders(t.Context(), coinmTradablePair, []uint64{283194212}, []string{"myOrder2"})
	require.NoError(t, err, "FuturesBatchCancelOrders must not error")
	if mockTests {
		exp := []CFuturesBatchOrderData{
			{
				ClientOrderID:           "myOrder1",
				CumulativeQuantity:      4,
				ExecutedQuantity:        4,
				OrderID:                 283194212,
				OriginalQuantity:        10,
				Price:                   9020.5,
				ReduceOnly:              true,
				ClosePosition:           true,
				Side:                    "SELL",
				PositionSide:            "SHORT",
				Status:                  "CANCELED",
				StopPrice:               9300,
				Symbol:                  "BTCUSD_PERP",
				Pair:                    "BTCUSD",
				TimeInForce:             "GTC",
				OrderType:               "TRAILING_STOP_MARKET",
				OriginalType:            "TRAILING_STOP_MARKET",
				ActivatePrice:           9020,
				PriceRate:               0.3,
				UpdateTime:              types.Time(time.UnixMilli(1571110484038)),
				WorkingType:             "CONTRACT_PRICE",
				PriceProtect:            true,
				PriceMatch:              "OPPONENT",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
			{
				Code:    -2011,
				Message: "Unknown order sent.",
			},
		}
		assert.Equal(t, exp, result, "FuturesBatchCancelOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "FuturesBatchCancelOrders should return an entry per order")
}

func TestFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesGetOrderData(t.Context(), currency.EMPTYPAIR, 1573346959, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesGetOrderData must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.FuturesGetOrderData(t.Context(), coinmTradablePair, 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "FuturesGetOrderData must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FuturesGetOrderData(t.Context(), coinmTradablePair, 1573346959, "")
	require.NoError(t, err, "FuturesGetOrderData must not error")
	if mockTests {
		exp := &FuturesOrderDetailResponse{
			AveragePrice:            9022.5,
			ClientOrderID:           "abc",
			CumulativeBase:          0.04432225,
			ExecutedQuantity:        4,
			OrderID:                 1573346959,
			OriginalQuantity:        10,
			OriginalType:            "TRAILING_STOP_MARKET",
			Price:                   9020.5,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "SHORT",
			Status:                  "PARTIALLY_FILLED",
			StopPrice:               9300,
			ClosePosition:           true,
			Symbol:                  "BTCUSD_PERP",
			Pair:                    "BTCUSD",
			Time:                    types.Time(time.UnixMilli(1579276756075)),
			TimeInForce:             "GTC",
			OrderType:               "TRAILING_STOP_MARKET",
			ActivatePrice:           9020,
			PriceRate:               0.3,
			UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
			WorkingType:             "CONTRACT_PRICE",
			PriceProtect:            true,
			PriceMatch:              "OPPONENT",
			SelfTradePreventionMode: "EXPIRE_MAKER",
		}
		assert.Equal(t, exp, result, "FuturesGetOrderData should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesGetOrderData should return the order")
}

func TestFuturesModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesModifyOrder must reject a nil request")
	req := &CFuturesModifyOrderRequest{}
	_, err = e.FuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesModifyOrder must reject an empty symbol")
	ensureTradablePairs(t)
	req.Symbol = coinmTradablePair
	_, err = e.FuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "FuturesModifyOrder must reject a missing order ID")
	req.OrderID = 20072994037
	_, err = e.FuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "FuturesModifyOrder must reject an empty side")
	req.Side = order.Buy.String()
	_, err = e.FuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FuturesModifyOrder must reject an empty quantity")
	req.Quantity = 2
	_, err = e.FuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "FuturesModifyOrder must reject an empty price without a price match")
	req.Price = 30005
	req.PriceMatch = "QUEUE"
	_, err = e.FuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "FuturesModifyOrder must reject a price match with a price")
	req.PriceMatch = ""

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.ModifyID = 1
	result, err := e.FuturesModifyOrder(t.Context(), req)
	require.NoError(t, err, "FuturesModifyOrder must not error")
	if mockTests {
		exp := &CFuturesModifyOrderResponse{
			OrderID:                 20072994037,
			Symbol:                  "BTCUSD_PERP",
			Pair:                    "BTCUSD",
			Status:                  "PARTIALLY_FILLED",
			ClientOrderID:           "LJ9R4QZDihCaS8UAOOLpgW",
			ModifyID:                1,
			Price:                   30005,
			OriginalQuantity:        2,
			ExecutedQuantity:        1,
			CumulativeQuantity:      1,
			TimeInForce:             "GTX",
			OrderType:               "LIMIT",
			ReduceOnly:              true,
			ClosePosition:           true,
			Side:                    "BUY",
			PositionSide:            "LONG",
			StopPrice:               29000,
			WorkingType:             "MARK_PRICE",
			PriceProtect:            true,
			OriginalType:            "LIMIT",
			PriceMatch:              "QUEUE",
			SelfTradePreventionMode: "EXPIRE_MAKER",
			UpdateTime:              types.Time(time.UnixMilli(1629182711600)),
		}
		assert.Equal(t, exp, result, "FuturesModifyOrder should decode every field")
	}
}

func TestFuturesNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesNewOrder must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.FuturesNewOrder(t.Context(), &FuturesNewOrderRequest{Symbol: coinmTradablePair, Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 1, Price: 1, NewClientOrderID: "testOrder"})
	require.NoError(t, err, "FuturesNewOrder must not error")
	if mockTests {
		exp := &FuturesOrderResponse{
			ClientOrderID:           "testOrder",
			CumulativeQuantity:      1,
			ExecutedQuantity:        1,
			OrderID:                 22542179,
			OriginalQuantity:        1,
			Price:                   1,
			ReduceOnly:              true,
			ClosePosition:           true,
			Side:                    "BUY",
			PositionSide:            "BOTH",
			Status:                  "FILLED",
			StopPrice:               9300,
			Symbol:                  "BTCUSD_PERP",
			Pair:                    "BTCUSD",
			TimeInForce:             "GTC",
			OrderType:               "LIMIT",
			OriginalType:            "LIMIT",
			ActivatePrice:           9020,
			PriceRate:               0.3,
			UpdateTime:              types.Time(time.UnixMilli(1566818724722)),
			WorkingType:             "CONTRACT_PRICE",
			PriceProtect:            true,
			PriceMatch:              "OPPONENT",
			SelfTradePreventionMode: "EXPIRE_MAKER",
		}
		assert.Equal(t, exp, result, "FuturesNewOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesNewOrder should return the order")
}

func TestFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesCancelOrder(t.Context(), currency.EMPTYPAIR, 283194212, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesCancelOrder must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.FuturesCancelOrder(t.Context(), coinmTradablePair, 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "FuturesCancelOrder must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FuturesCancelOrder(t.Context(), coinmTradablePair, 283194212, "myOrder1")
	require.NoError(t, err, "FuturesCancelOrder must not error")
	if mockTests {
		exp := &FuturesOrderResponse{
			ClientOrderID:           "myOrder1",
			CumulativeQuantity:      4,
			ExecutedQuantity:        4,
			OrderID:                 283194212,
			OriginalQuantity:        10,
			Price:                   9020.5,
			ReduceOnly:              true,
			ClosePosition:           true,
			Side:                    "SELL",
			PositionSide:            "SHORT",
			Status:                  "CANCELED",
			StopPrice:               9300,
			Symbol:                  "BTCUSD_PERP",
			Pair:                    "BTCUSD",
			TimeInForce:             "GTC",
			OrderType:               "TRAILING_STOP_MARKET",
			OriginalType:            "TRAILING_STOP_MARKET",
			ActivatePrice:           9020,
			PriceRate:               0.3,
			UpdateTime:              types.Time(time.UnixMilli(1571110484038)),
			WorkingType:             "CONTRACT_PRICE",
			PriceProtect:            true,
			PriceMatch:              "OPPONENT",
			SelfTradePreventionMode: "EXPIRE_MAKER",
		}
		assert.Equal(t, exp, result, "FuturesCancelOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesCancelOrder should return the order")
}

func TestFuturesChangeInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesChangeInitialLeverage(t.Context(), currency.EMPTYPAIR, 21)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesChangeInitialLeverage must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.FuturesChangeInitialLeverage(t.Context(), coinmTradablePair, 0)
	require.ErrorIs(t, err, errLeverageRequired, "FuturesChangeInitialLeverage must reject an empty leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FuturesChangeInitialLeverage(t.Context(), coinmTradablePair, 21)
	require.NoError(t, err, "FuturesChangeInitialLeverage must not error")
	if mockTests {
		exp := &CFuturesChangeLeverageResponse{
			Leverage:    21,
			MaxQuantity: 1000,
			Symbol:      "BTCUSD_PERP",
		}
		assert.Equal(t, exp, result, "FuturesChangeInitialLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesChangeInitialLeverage should return the leverage")
}

func TestFuturesChangeMarginType(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesChangeMarginType(t.Context(), currency.EMPTYPAIR, "ISOLATED")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesChangeMarginType must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.FuturesChangeMarginType(t.Context(), coinmTradablePair, "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "FuturesChangeMarginType must reject an empty margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FuturesChangeMarginType(t.Context(), coinmTradablePair, "ISOLATED")
	require.NoError(t, err, "FuturesChangeMarginType must not error")
	if mockTests {
		exp := &CFuturesStatusResponse{
			Code:    200,
			Message: "success",
		}
		assert.Equal(t, exp, result, "FuturesChangeMarginType should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesChangeMarginType should return the status")
}

func TestGetFuturesAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	result, err := e.GetFuturesAllOpenOrders(t.Context(), coinmTradablePair, currency.EMPTYCODE)
	require.NoError(t, err, "GetFuturesAllOpenOrders must not error")
	if mockTests {
		exp := []FuturesOrderDetailResponse{
			{
				AveragePrice:            9022.5,
				ClientOrderID:           "abc",
				CumulativeBase:          0.04432225,
				ExecutedQuantity:        4,
				OrderID:                 1917641,
				OriginalQuantity:        10,
				OriginalType:            "TRAILING_STOP_MARKET",
				Price:                   9020.5,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "SHORT",
				Status:                  "PARTIALLY_FILLED",
				StopPrice:               9300,
				ClosePosition:           true,
				Symbol:                  "BTCUSD_PERP",
				Pair:                    "BTCUSD",
				Time:                    types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:             "GTC",
				OrderType:               "TRAILING_STOP_MARKET",
				ActivatePrice:           9020,
				PriceRate:               0.3,
				UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
				WorkingType:             "CONTRACT_PRICE",
				PriceProtect:            true,
				PriceMatch:              "OPPONENT",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
			{
				ClientOrderID:           "abe",
				OrderID:                 1917643,
				OriginalQuantity:        10,
				OriginalType:            "LIMIT",
				Price:                   9020.5,
				Side:                    "BUY",
				PositionSide:            "SHORT",
				Status:                  "NEW",
				Symbol:                  "BTCUSD_PERP",
				Pair:                    "BTCUSD",
				Time:                    types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:             "GTX",
				OrderType:               "LIMIT",
				UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
				WorkingType:             "CONTRACT_PRICE",
				PriceMatch:              "NONE",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesAllOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesAllOpenOrders should return orders")
}

func TestGetFuturesOrderModifyHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderModifyHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesOrderModifyHistory must reject a nil request")
	_, err = e.GetFuturesOrderModifyHistory(t.Context(), &CFuturesOrderModifyHistoryRequest{OrderID: 20072994037})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesOrderModifyHistory must reject an empty symbol")
	ensureTradablePairs(t)
	_, err = e.GetFuturesOrderModifyHistory(t.Context(), &CFuturesOrderModifyHistoryRequest{Symbol: coinmTradablePair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFuturesOrderModifyHistory must reject a missing order ID")
	startTime, endTime := getTime()
	_, err = e.GetFuturesOrderModifyHistory(t.Context(), &CFuturesOrderModifyHistoryRequest{Symbol: coinmTradablePair, OrderID: 20072994037, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesOrderModifyHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrderModifyHistory(t.Context(), &CFuturesOrderModifyHistoryRequest{Symbol: coinmTradablePair, OrderID: 20072994037, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetFuturesOrderModifyHistory must not error")
	if mockTests {
		exp := []FuturesOrderAmendment{
			{
				AmendmentID:   5363,
				Symbol:        "BTCUSD_PERP",
				Pair:          "BTCUSD",
				OrderID:       20072994037,
				ClientOrderID: "LJ9R4QZDihCaS8UAOOLpgW",
				Time:          types.Time(time.UnixMilli(1629184560899)),
				Amendment: OrderAmendment{
					Price:            AmendmentChange{Before: 30004, After: 30003.2},
					OriginalQuantity: AmendmentChange{Before: 1, After: 2},
					Count:            3,
					ModifyID:         123,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesOrderModifyHistory should decode every field")
	}
}

func TestFuturesMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesMarginChangeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesMarginChangeHistory must reject a nil request")
	_, err = e.FuturesMarginChangeHistory(t.Context(), &CFuturesPositionMarginHistoryRequest{Type: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesMarginChangeHistory must reject an empty symbol")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.FuturesMarginChangeHistory(t.Context(), &CFuturesPositionMarginHistoryRequest{Symbol: coinmTradablePair, Type: 1, StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FuturesMarginChangeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FuturesMarginChangeHistory(t.Context(), &CFuturesPositionMarginHistoryRequest{Symbol: coinmTradablePair, Type: 1, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "FuturesMarginChangeHistory must not error")
	if mockTests {
		exp := []GetPositionMarginChangeHistoryData{
			{
				Amount:       23.36332311,
				Asset:        currency.BTC,
				Symbol:       "BTCUSD_PERP",
				Time:         types.Time(time.Unix(1744120800, 0)),
				Type:         1,
				PositionSide: "BOTH",
			},
		}
		assert.Equal(t, exp, result, "FuturesMarginChangeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesMarginChangeHistory should return margin changes")
}

func TestModifyIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyIsolatedPositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyIsolatedPositionMargin must reject a nil request")
	_, err = e.ModifyIsolatedPositionMargin(t.Context(), &CFuturesModifyIsolatedPositionMarginRequest{Amount: 100, Type: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ModifyIsolatedPositionMargin must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.ModifyIsolatedPositionMargin(t.Context(), &CFuturesModifyIsolatedPositionMarginRequest{Symbol: coinmTradablePair, Type: 1})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "ModifyIsolatedPositionMargin must reject an empty amount")
	_, err = e.ModifyIsolatedPositionMargin(t.Context(), &CFuturesModifyIsolatedPositionMarginRequest{Symbol: coinmTradablePair, Amount: 100})
	require.ErrorIs(t, err, errMarginChangeTypeInvalid, "ModifyIsolatedPositionMargin must reject an empty change type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyIsolatedPositionMargin(t.Context(), &CFuturesModifyIsolatedPositionMarginRequest{Symbol: coinmTradablePair, PositionSide: "BOTH", Amount: 100, Type: 1})
	require.NoError(t, err, "ModifyIsolatedPositionMargin must not error")
	if mockTests {
		exp := &FuturesMarginUpdatedResponse{
			Amount:  100,
			Code:    200,
			Message: "Successfully modify position margin.",
			Type:    1,
		}
		assert.Equal(t, exp, result, "ModifyIsolatedPositionMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "ModifyIsolatedPositionMargin should return the result")
}

func TestFuturesPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FuturesPositionsADLEstimate(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "FuturesPositionsADLEstimate must not error")
	if mockTests {
		exp := []ADLEstimateData{
			{
				Symbol: "BTCUSD_PERP",
				ADLQuantile: CFuturesADLQuantile{
					Long:  2,
					Short: 2,
					Hedge: 2,
				},
			},
			{
				Symbol: "ETHUSD_PERP",
				ADLQuantile: CFuturesADLQuantile{
					Both: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "FuturesPositionsADLEstimate should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesPositionsADLEstimate should return quantiles")
}

func TestFuturesPositionsInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FuturesPositionsInfo(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.NoError(t, err, "FuturesPositionsInfo must not error")
	if mockTests {
		exp := []FuturesPositionInformation{
			{
				Symbol:           "BTCUSD_PERP",
				PositionAmount:   3,
				EntryPrice:       85642.5,
				BreakEvenPrice:   85676.8,
				MarkPrice:        86068.2,
				UnrealizedProfit: -0.00000081,
				LiquidationPrice: 74502.8,
				Leverage:         8,
				MaxQuantity:      500,
				MarginType:       "isolated",
				IsolatedMargin:   0.00004449,
				IsAutoAddMargin:  true,
				PositionSide:     "BOTH",
				UpdateTime:       types.Time(time.Unix(1744120800, 0)),
				NotionalValue:    0.00348561,
				IsolatedWallet:   0.0000453,
			},
			{
				Symbol:       "BTCUSD_261225",
				MarkPrice:    87060.31088175,
				Leverage:     20,
				MaxQuantity:  250,
				MarginType:   "cross",
				PositionSide: "BOTH",
			},
		}
		assert.Equal(t, exp, result, "FuturesPositionsInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesPositionsInfo should return positions")
}

func TestFuturesOpenOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesOpenOrderData(t.Context(), currency.EMPTYPAIR, 1917641, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesOpenOrderData must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.FuturesOpenOrderData(t.Context(), coinmTradablePair, 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "FuturesOpenOrderData must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FuturesOpenOrderData(t.Context(), coinmTradablePair, 1917641, "")
	require.NoError(t, err, "FuturesOpenOrderData must not error")
	if mockTests {
		exp := &FuturesOrderDetailResponse{
			AveragePrice:            9022.5,
			ClientOrderID:           "abc",
			CumulativeBase:          0.04432225,
			ExecutedQuantity:        4,
			OrderID:                 1917641,
			OriginalQuantity:        10,
			OriginalType:            "TRAILING_STOP_MARKET",
			Price:                   9020.5,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "SHORT",
			Status:                  "PARTIALLY_FILLED",
			StopPrice:               9300,
			ClosePosition:           true,
			Symbol:                  "BTCUSD_PERP",
			Pair:                    "BTCUSD",
			Time:                    types.Time(time.UnixMilli(1579276756075)),
			TimeInForce:             "GTC",
			OrderType:               "TRAILING_STOP_MARKET",
			ActivatePrice:           9020,
			PriceRate:               0.3,
			UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
			WorkingType:             "CONTRACT_PRICE",
			PriceProtect:            true,
			PriceMatch:              "OPPONENT",
			SelfTradePreventionMode: "EXPIRE_MAKER",
		}
		assert.Equal(t, exp, result, "FuturesOpenOrderData should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesOpenOrderData should return the order")
}

func TestFuturesForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesForceOrders must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.FuturesForceOrders(t.Context(), &CFuturesForceOrdersRequest{AutoCloseType: "ADL", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "FuturesForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.FuturesForceOrders(t.Context(), &CFuturesForceOrdersRequest{AutoCloseType: "ADL", StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "FuturesForceOrders must not error")
	if mockTests {
		exp := []ForcedOrdersData{
			{
				OrderID:          165123080,
				Symbol:           "BTCUSD_PERP",
				Pair:             "BTCUSD",
				Status:           "FILLED",
				ClientOrderID:    "adl_autoclose",
				Price:            11326.9,
				AveragePrice:     11326.9,
				OriginalQuantity: 1,
				ExecutedQuantity: 1,
				CumulativeBase:   0.00882854,
				CumulativeQuote:  100,
				TimeInForce:      "IOC",
				OrderType:        "LIMIT",
				ReduceOnly:       true,
				ClosePosition:    true,
				Side:             "SELL",
				PositionSide:     "BOTH",
				StopPrice:        11300,
				WorkingType:      "CONTRACT_PRICE",
				PriceProtect:     true,
				OriginalType:     "LIMIT",
				Time:             types.Time(time.UnixMilli(1596542005019)),
				UpdateTime:       types.Time(time.UnixMilli(1596542005050)),
				GoodTillDate:     types.Time(time.UnixMilli(1596542005099)),
			},
		}
		assert.Equal(t, exp, result, "FuturesForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "FuturesForceOrders should return orders")
}

func TestCFuturesQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.CFuturesQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CFuturesQuarterlyContractSettlementPrice must reject an empty pair")

	result, err := e.CFuturesQuarterlyContractSettlementPrice(t.Context(), currency.NewCode("BTCUSD"))
	require.NoError(t, err, "CFuturesQuarterlyContractSettlementPrice must not error")
	if mockTests {
		exp := []SettlementPrice{
			{
				DeliveryTime:  types.Time(time.Unix(1782432000, 0)),
				DeliveryPrice: 60510.3,
			},
			{
				DeliveryTime:  types.Time(time.Unix(1774569600, 0)),
				DeliveryPrice: 68597.7,
			},
		}
		assert.Equal(t, exp, result, "CFuturesQuarterlyContractSettlementPrice should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CFuturesQuarterlyContractSettlementPrice should return settlement prices")
}

func TestKeepaliveCFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		_, err := e.StartCFuturesUserDataStream(t.Context())
		require.NoError(t, err, "StartCFuturesUserDataStream must not error")
	}
	result, err := e.KeepaliveCFuturesUserDataStream(t.Context())
	require.NoError(t, err, "KeepaliveCFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "KeepaliveCFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "KeepaliveCFuturesUserDataStream should return the listen key")
}

func TestStartCFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.StartCFuturesUserDataStream(t.Context())
	require.NoError(t, err, "StartCFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "StartCFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "StartCFuturesUserDataStream should return a listen key")
}

func TestCloseCFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		// Closing invalidates the listen key every client of the account shares
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CloseCFuturesUserDataStream(t.Context()), "CloseCFuturesUserDataStream should not error")
}
