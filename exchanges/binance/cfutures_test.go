package binance

import (
	"net/http"
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

func TestGetCoinMarginedFuturesAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoinMarginedFuturesAccountInfo(t.Context())
	require.NoError(t, err, "GetCoinMarginedFuturesAccountInfo must not error")
	if mockTests {
		exp := &CoinMarginedFuturesAccountInformationResponse{
			Assets: []CoinMarginedFuturesAccountAsset{
				{
					Asset:                  currency.BTC,
					WalletBalance:          0.00241969,
					UnrealisedProfit:       -0.00000081,
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
			Positions: []CoinMarginedFuturesAccountInformationPosition{
				{
					Symbol:                 "BTCUSD_PERP",
					PositionAmount:         3,
					InitialMargin:          0.0000441,
					MaintenanceMargin:      0.00001102,
					UnrealisedProfit:       -0.00000081,
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesAccountInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCoinMarginedFuturesAccountInfo should return account information")
}

func TestGetCoinMarginedFuturesAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoinMarginedFuturesAccountBalance(t.Context())
	require.NoError(t, err, "GetCoinMarginedFuturesAccountBalance must not error")
	if mockTests {
		exp := []CoinMarginedFuturesAccountBalanceResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesAccountBalance should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesAccountBalance should return balances")
}

func TestCoinMarginedFuturesIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesIncomeHistory must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.CoinMarginedFuturesIncomeHistory(t.Context(), &CoinMarginedFuturesIncomeHistoryRequest{IncomeType: "COMMISSION", StartTime: endTime, EndTime: startTime, Limit: 5})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CoinMarginedFuturesIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	result, err := e.CoinMarginedFuturesIncomeHistory(t.Context(), &CoinMarginedFuturesIncomeHistoryRequest{Symbol: coinmTradablePair, IncomeType: "COMMISSION", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 5})
	require.NoError(t, err, "CoinMarginedFuturesIncomeHistory must not error")
	if mockTests {
		exp := []CoinMarginedFuturesIncomeHistoryResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesIncomeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesIncomeHistory should return income records")
}

func TestCoinMarginedFuturesNotionalBracket(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	btcusd := NotionalBracketResponse{
		Pair: "BTCUSD",
		Brackets: []NotionalBracket{
			{Bracket: 1, InitialLeverage: 125, QuantityCap: 5, MaintenanceMarginRatio: 0.004},
			{Bracket: 2, InitialLeverage: 100, QuantityCap: 10, QuantityFloor: 5, MaintenanceMarginRatio: 0.005, Cumulative: 0.005},
		},
	}
	for _, tc := range []struct {
		name string
		pair currency.Code
		exp  []NotionalBracketResponse
	}{
		{"pair", currency.NewCode("BTCUSD"), []NotionalBracketResponse{btcusd}},
		{"every pair", currency.EMPTYCODE, []NotionalBracketResponse{btcusd, {
			Pair: "ETHUSD",
			Brackets: []NotionalBracket{
				{Bracket: 1, InitialLeverage: 75, QuantityCap: 150, MaintenanceMarginRatio: 0.0065},
				{Bracket: 2, InitialLeverage: 50, QuantityCap: 300, QuantityFloor: 150, MaintenanceMarginRatio: 0.01, Cumulative: 0.525},
			},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CoinMarginedFuturesNotionalBracket(t.Context(), tc.pair)
			require.NoError(t, err, "CoinMarginedFuturesNotionalBracket must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CoinMarginedFuturesNotionalBracket should decode every field")
				return
			}
			assert.NotEmpty(t, result, "CoinMarginedFuturesNotionalBracket should return brackets")
		})
	}
}

func TestGetCoinMarginedFuturesBasisData(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesBasisData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedFuturesBasisData must reject a nil request")
	_, err = e.GetCoinMarginedFuturesBasisData(t.Context(), &CoinMarginedFuturesStatisticsRequest{ContractType: "CURRENT_QUARTER", Period: kline.FiveMin})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesBasisData must reject an empty pair")
	_, err = e.GetCoinMarginedFuturesBasisData(t.Context(), &CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), Period: kline.FiveMin})
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetCoinMarginedFuturesBasisData must reject an empty contract type")
	_, err = e.GetCoinMarginedFuturesBasisData(t.Context(), &CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER"})
	require.ErrorIs(t, err, errInvalidPeriodOrInterval, "GetCoinMarginedFuturesBasisData must reject an empty period")

	startTime, endTime := getTime()
	_, err = e.GetCoinMarginedFuturesBasisData(t.Context(), &CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Period: kline.FiveMin, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedFuturesBasisData must reject a start time after the end time")

	result, err := e.GetCoinMarginedFuturesBasisData(t.Context(), &CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Period: kline.FiveMin, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetCoinMarginedFuturesBasisData must not error")
	if mockTests {
		exp := []CoinMarginedFuturesBasisResponse{
			{
				IndexPrice:          85913.95737453,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0112,
				FuturesPrice:        86872.4,
				AnnualisedBasisRate: 0.0511,
				Basis:               958.44262547,
				Pair:                "BTCUSD",
				Timestamp:           types.Time(time.Unix(1791278100, 0)),
			},
			{
				IndexPrice:          85988.84769358,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0111,
				FuturesPrice:        86942.1,
				AnnualisedBasisRate: 0.0507,
				Basis:               953.25230642,
				Pair:                "BTCUSD",
				Timestamp:           types.Time(time.Unix(1791278400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesBasisData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesBasisData should return basis records")
}

func TestCoinMarginedFuturesServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.CoinMarginedFuturesServerTime(t.Context())
	require.NoError(t, err, "CoinMarginedFuturesServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1791298531317), result, "CoinMarginedFuturesServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "CoinMarginedFuturesServerTime should return the current time")
}

func TestGetCoinMarginedFuturesAggregatedTradesList(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesAggregatedTradesList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedFuturesAggregatedTradesList must reject a nil request")
	_, err = e.GetCoinMarginedFuturesAggregatedTradesList(t.Context(), &CoinMarginedFuturesAggregatedTradesRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesAggregatedTradesList must reject an empty symbol")

	ensureTradablePairs(t)
	_, endTime := getTime()
	// Binance only serves the last 48 hours of aggregate trades, in windows shorter than an hour
	startTime := endTime.Add(-30 * time.Minute)
	_, err = e.GetCoinMarginedFuturesAggregatedTradesList(t.Context(), &CoinMarginedFuturesAggregatedTradesRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedFuturesAggregatedTradesList must reject a start time after the end time")

	result, err := e.GetCoinMarginedFuturesAggregatedTradesList(t.Context(), &CoinMarginedFuturesAggregatedTradesRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetCoinMarginedFuturesAggregatedTradesList must not error")
	if mockTests {
		exp := []CoinMarginedFuturesAggregatedTradeResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesAggregatedTradesList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCoinMarginedFuturesAggregatedTradesList should return aggregate trades")
}

func TestGetCoinMarginedFuturesContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesContinuousKlineData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedFuturesContinuousKlineData must reject a nil request")
	_, err = e.GetCoinMarginedFuturesContinuousKlineData(t.Context(), &CoinMarginedFuturesContinuousKlineRequest{ContractType: "CURRENT_QUARTER", Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesContinuousKlineData must reject an empty pair")
	_, err = e.GetCoinMarginedFuturesContinuousKlineData(t.Context(), &CoinMarginedFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), Interval: kline.OneHour})
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetCoinMarginedFuturesContinuousKlineData must reject an empty contract type")
	_, err = e.GetCoinMarginedFuturesContinuousKlineData(t.Context(), &CoinMarginedFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER"})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetCoinMarginedFuturesContinuousKlineData must reject an empty interval")

	startTime, endTime := getTime()
	_, err = e.GetCoinMarginedFuturesContinuousKlineData(t.Context(), &CoinMarginedFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Interval: kline.OneHour, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedFuturesContinuousKlineData must reject a start time after the end time")

	result, err := e.GetCoinMarginedFuturesContinuousKlineData(t.Context(), &CoinMarginedFuturesContinuousKlineRequest{Pair: currency.NewCode("BTCUSD"), ContractType: "CURRENT_QUARTER", Interval: kline.OneHour, StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "GetCoinMarginedFuturesContinuousKlineData must not error")
	if mockTests {
		exp := []CoinMarginedFuturesCandleStickResponse{
			{
				OpenTime:                  types.Time(time.Unix(1744106400, 0)),
				Open:                      79836.5,
				High:                      80120.1,
				Low:                       79745.5,
				Close:                     79890.3,
				VolumeInContracts:         15708,
				CloseTime:                 types.Time(time.UnixMilli(1744109999999)),
				BaseAssetVolume:           19.64641546,
				NumberOfTrades:            507,
				TakerBuyVolumeInContracts: 8474,
				TakerBuyBaseAssetVolume:   10.60066335,
			},
			{
				OpenTime:                  types.Time(time.Unix(1744110000, 0)),
				Open:                      79954.9,
				High:                      80999.2,
				Low:                       79927,
				Close:                     80916.5,
				VolumeInContracts:         24560,
				CloseTime:                 types.Time(time.UnixMilli(1744113599999)),
				BaseAssetVolume:           30.511258,
				NumberOfTrades:            827,
				TakerBuyVolumeInContracts: 14297,
				TakerBuyBaseAssetVolume:   17.75126477,
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesContinuousKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesContinuousKlineData should return klines")
}

func TestCoinMarginedFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.CoinMarginedFuturesExchangeInfo(t.Context())
	require.NoError(t, err, "CoinMarginedFuturesExchangeInfo must not error")
	if mockTests {
		exp := &CoinMarginedFuturesExchangeInfoResponse{
			ExchangeFilters: []string{
				"EXCHANGE_MAX_NUM_ORDERS",
			},
			RateLimits: []RateLimitInfoResponse{
				{
					RateLimitType:  "REQUEST_WEIGHT",
					Interval:       "MINUTE",
					IntervalNumber: 1,
					Limit:          2400,
				},
				{
					RateLimitType:  "ORDERS",
					Interval:       "MINUTE",
					IntervalNumber: 1,
					Limit:          1200,
				},
			},
			ServerTime: types.Time(time.UnixMilli(1791278101111)),
			Symbols: []CoinMarginedFuturesSymbolInfo{
				{
					Filters: []CoinMarginedFuturesSymbolFilter{
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
					Filters: []CoinMarginedFuturesSymbolFilter{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesExchangeInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbols, "CoinMarginedFuturesExchangeInfo should return symbols")
}

func TestCoinMarginedFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesGetFundingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesGetFundingHistory must reject a nil request")
	_, err = e.CoinMarginedFuturesGetFundingHistory(t.Context(), &CoinMarginedFuturesFundingRateHistoryRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesGetFundingHistory must reject an empty symbol")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.CoinMarginedFuturesGetFundingHistory(t.Context(), &CoinMarginedFuturesFundingRateHistoryRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CoinMarginedFuturesGetFundingHistory must reject a start time after the end time")

	result, err := e.CoinMarginedFuturesGetFundingHistory(t.Context(), &CoinMarginedFuturesFundingRateHistoryRequest{Symbol: coinmTradablePair, Limit: 5, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "CoinMarginedFuturesGetFundingHistory must not error")
	if mockTests {
		exp := []CoinMarginedFuturesFundingRateHistoryResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesGetFundingHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesGetFundingHistory should return funding rates")
}

func TestGetCoinMarginedFuturesFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetCoinMarginedFuturesFundingRateInfo(t.Context())
	require.NoError(t, err, "GetCoinMarginedFuturesFundingRateInfo must not error")
	if mockTests {
		exp := []CoinMarginedFuturesFundingRateInfoResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesFundingRateInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCoinMarginedFuturesFundingRateInfo should return a list, which may be empty")
}

func TestGetCoinMarginedFuturesIndexAndMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.GetCoinMarginedFuturesIndexAndMarkPrice(t.Context(), currency.EMPTYPAIR, currency.NewCode("BTCUSD"))
	require.NoError(t, err, "GetCoinMarginedFuturesIndexAndMarkPrice must not error")
	if mockTests {
		exp := []IndexMarkPriceResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesIndexAndMarkPrice should decode every field and keep only the pair's symbols")
		return
	}
	require.NotEmpty(t, result, "GetCoinMarginedFuturesIndexAndMarkPrice must return prices")
	for i := range result {
		assert.Equalf(t, "BTCUSD", result[i].Pair, "GetCoinMarginedFuturesIndexAndMarkPrice should return only the pair's symbols, not %s", result[i].Symbol)
	}
}

func TestGetCoinMarginedFuturesIndexPriceKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesIndexPriceKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedFuturesIndexPriceKlines must reject a nil request")
	_, err = e.GetCoinMarginedFuturesIndexPriceKlines(t.Context(), &CoinMarginedFuturesIndexPriceKlineRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesIndexPriceKlines must reject an empty pair")
	_, err = e.GetCoinMarginedFuturesIndexPriceKlines(t.Context(), &CoinMarginedFuturesIndexPriceKlineRequest{Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetCoinMarginedFuturesIndexPriceKlines must reject an empty interval")

	startTime, endTime := getTime()
	_, err = e.GetCoinMarginedFuturesIndexPriceKlines(t.Context(), &CoinMarginedFuturesIndexPriceKlineRequest{Pair: currency.NewCode("BTCUSD"), Interval: kline.OneHour, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedFuturesIndexPriceKlines must reject a start time after the end time")

	result, err := e.GetCoinMarginedFuturesIndexPriceKlines(t.Context(), &CoinMarginedFuturesIndexPriceKlineRequest{Pair: currency.NewCode("BTCUSD"), Interval: kline.OneHour, StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "GetCoinMarginedFuturesIndexPriceKlines must not error")
	if mockTests {
		exp := []CoinMarginedFuturesPriceCandleStickResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesIndexPriceKlines should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesIndexPriceKlines should return klines")
}

// symbolKlineTests checks the request validation shared by the coin margined futures kline endpoints that take a
// symbol, and returns the request their success case sends
func symbolKlineTests(t *testing.T, fn func(*CoinMarginedFuturesKlineRequest) error) *CoinMarginedFuturesKlineRequest {
	t.Helper()
	require.ErrorIs(t, fn(nil), common.ErrNilPointer, "a nil request must be rejected")
	require.ErrorIs(t, fn(&CoinMarginedFuturesKlineRequest{Interval: kline.OneHour}), currency.ErrCurrencyPairEmpty, "an empty symbol must be rejected")
	ensureTradablePairs(t)
	require.ErrorIs(t, fn(&CoinMarginedFuturesKlineRequest{Symbol: coinmTradablePair}), kline.ErrInvalidInterval, "an empty interval must be rejected")
	startTime, endTime := getTime()
	require.ErrorIs(t, fn(&CoinMarginedFuturesKlineRequest{Symbol: coinmTradablePair, Interval: kline.OneHour, StartTime: endTime, EndTime: startTime}), common.ErrStartAfterEnd, "a start time after the end time must be rejected")
	return &CoinMarginedFuturesKlineRequest{Symbol: coinmTradablePair, Interval: kline.OneHour, StartTime: startTime, EndTime: endTime, Limit: 5}
}

func TestGetCoinMarginedFuturesKlineData(t *testing.T) {
	t.Parallel()
	req := symbolKlineTests(t, func(r *CoinMarginedFuturesKlineRequest) error {
		_, err := e.GetCoinMarginedFuturesKlineData(t.Context(), r)
		return err
	})
	result, err := e.GetCoinMarginedFuturesKlineData(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesKlineData must not error")
	if mockTests {
		exp := []CoinMarginedFuturesCandleStickResponse{
			{
				OpenTime:                  types.Time(time.Unix(1744106400, 0)),
				Open:                      78844,
				High:                      79120,
				Low:                       78765,
				Close:                     78950,
				VolumeInContracts:         180128,
				CloseTime:                 types.Time(time.UnixMilli(1744109999999)),
				BaseAssetVolume:           228.21023657,
				NumberOfTrades:            8942,
				TakerBuyVolumeInContracts: 84148,
				TakerBuyBaseAssetVolume:   106.61711606,
			},
			{
				OpenTime:                  types.Time(time.Unix(1744110000, 0)),
				Open:                      78951.1,
				High:                      80000,
				Low:                       78910.1,
				Close:                     79916.1,
				VolumeInContracts:         818611,
				CloseTime:                 types.Time(time.UnixMilli(1744113599999)),
				BaseAssetVolume:           1028.54016988,
				NumberOfTrades:            23993,
				TakerBuyVolumeInContracts: 561352,
				TakerBuyBaseAssetVolume:   705.11758224,
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesKlineData should return klines")
}

// futuresStatisticsTests checks the request validation shared by the coin margined futures statistics endpoints, and
// returns the request their success case sends
func futuresStatisticsTests(t *testing.T, contractType string, fn func(*CoinMarginedFuturesStatisticsRequest) error) *CoinMarginedFuturesStatisticsRequest {
	t.Helper()
	require.ErrorIs(t, fn(nil), common.ErrNilPointer, "a nil request must be rejected")
	require.ErrorIs(t, fn(&CoinMarginedFuturesStatisticsRequest{ContractType: contractType, Period: kline.FiveMin}), currency.ErrCurrencyPairEmpty, "an empty pair must be rejected")
	require.ErrorIs(t, fn(&CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: contractType}), errInvalidPeriodOrInterval, "an empty period must be rejected")
	startTime, endTime := getTime()
	require.ErrorIs(t, fn(&CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: contractType, Period: kline.FiveMin, StartTime: endTime, EndTime: startTime}), common.ErrStartAfterEnd, "a start time after the end time must be rejected")
	return &CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), ContractType: contractType, Period: kline.FiveMin, StartTime: startTime, EndTime: endTime}
}

func TestGetCoinMarginedFuturesMarketRatio(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "", func(r *CoinMarginedFuturesStatisticsRequest) error {
		_, err := e.GetCoinMarginedFuturesMarketRatio(t.Context(), r)
		return err
	})
	result, err := e.GetCoinMarginedFuturesMarketRatio(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesMarketRatio must not error")
	if mockTests {
		exp := []CoinMarginedFuturesLongShortAccountRatioResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesMarketRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesMarketRatio should return ratios")
}

func TestGetCoinMarginedFuturesMarkPriceKline(t *testing.T) {
	t.Parallel()
	req := symbolKlineTests(t, func(r *CoinMarginedFuturesKlineRequest) error {
		_, err := e.GetCoinMarginedFuturesMarkPriceKline(t.Context(), r)
		return err
	})
	result, err := e.GetCoinMarginedFuturesMarkPriceKline(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesMarkPriceKline must not error")
	if mockTests {
		exp := []CoinMarginedFuturesPriceCandleStickResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesMarkPriceKline should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesMarkPriceKline should return klines")
}

func TestGetCoinMarginedFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesHistoricalTrades(t.Context(), currency.EMPTYPAIR, 0, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesHistoricalTrades must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	result, err := e.GetCoinMarginedFuturesHistoricalTrades(t.Context(), coinmTradablePair, 0, 5)
	require.NoError(t, err, "GetCoinMarginedFuturesHistoricalTrades must not error")
	if mockTests {
		exp := []CoinMarginedFuturesPublicTradesResponse{
			{
				ID:                  1157458431,
				Price:               86118.7,
				QuantityInContracts: 1,
				BaseQuantity:        0.00116118,
				Time:                types.Time(time.UnixMilli(1791285176377)),
			},
			{
				ID:                  1157458432,
				Price:               86118.7,
				QuantityInContracts: 180,
				BaseQuantity:        0.20901383,
				Time:                types.Time(time.UnixMilli(1791285176377)),
				IsBuyerMaker:        true,
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesHistoricalTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesHistoricalTrades should return trades")
}

func TestCoinMarginedFuturesOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesOpenInterest(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesOpenInterest must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.CoinMarginedFuturesOpenInterest(t.Context(), coinmTradablePair)
	require.NoError(t, err, "CoinMarginedFuturesOpenInterest must not error")
	if mockTests {
		exp := &CoinMarginedFuturesOpenInterestResponse{
			Symbol:       "BTCUSD_PERP",
			Pair:         "BTCUSD",
			OpenInterest: 12669867,
			ContractType: "PERPETUAL",
			Time:         types.Time(time.UnixMilli(1791282919818)),
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesOpenInterest should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesOpenInterest should return open interest")
}

func TestGetCoinMarginedFuturesOpenInterestStats(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "CURRENT_QUARTER", func(r *CoinMarginedFuturesStatisticsRequest) error {
		_, err := e.GetCoinMarginedFuturesOpenInterestStats(t.Context(), r)
		return err
	})
	result, err := e.GetCoinMarginedFuturesOpenInterestStats(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesOpenInterestStats must not error")
	if mockTests {
		exp := []OpenInterestStatsResponse{
			{
				Pair:                            "BTCUSD",
				ContractType:                    "CURRENT_QUARTER",
				SumOpenInterestInContracts:      212983,
				SumOpenInterestValueInBaseAsset: 245.17046161,
				Timestamp:                       types.Time(time.Unix(1791278100, 0)),
			},
			{
				Pair:                            "BTCUSD",
				ContractType:                    "CURRENT_QUARTER",
				SumOpenInterestInContracts:      212974,
				SumOpenInterestValueInBaseAsset: 244.95863332,
				Timestamp:                       types.Time(time.Unix(1791278400, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesOpenInterestStats should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesOpenInterestStats should return statistics")
}

func TestGetCoinMarginedFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesOrderbook(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesOrderbook must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.GetCoinMarginedFuturesOrderbook(t.Context(), coinmTradablePair, 1000)
	require.NoError(t, err, "GetCoinMarginedFuturesOrderbook must not error")
	if mockTests {
		exp := &CoinMarginedFuturesOrderBookResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetCoinMarginedFuturesOrderbook should return bids")
}

func TestGetCoinMarginedFuturesPremiumIndexKlineData(t *testing.T) {
	t.Parallel()
	req := symbolKlineTests(t, func(r *CoinMarginedFuturesKlineRequest) error {
		_, err := e.GetCoinMarginedFuturesPremiumIndexKlineData(t.Context(), r)
		return err
	})
	result, err := e.GetCoinMarginedFuturesPremiumIndexKlineData(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesPremiumIndexKlineData must not error")
	if mockTests {
		exp := []CoinMarginedFuturesPremiumIndexCandleStickResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesPremiumIndexKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesPremiumIndexKlineData should return klines")
}

func TestGetCoinMarginedFuturesIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesIndexPriceConstituents(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesIndexPriceConstituents must reject an empty symbol")

	result, err := e.GetCoinMarginedFuturesIndexPriceConstituents(t.Context(), currency.NewCode("BTCUSD"))
	require.NoError(t, err, "GetCoinMarginedFuturesIndexPriceConstituents must not error")
	if mockTests {
		exp := &CoinMarginedFuturesIndexPriceConstituentsResponse{
			Symbol: "BTCUSD",
			Time:   types.Time(time.UnixMilli(1791282887128)),
			Constituents: []CoinMarginedFuturesIndexPriceConstituent{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesIndexPriceConstituents should decode every field")
		return
	}
	assert.NotEmpty(t, result.Constituents, "GetCoinMarginedFuturesIndexPriceConstituents should return constituents")
}

func TestGetCoinMarginedFuturesPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesPublicTrades(t.Context(), currency.EMPTYPAIR, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesPublicTrades must reject an empty symbol")

	ensureTradablePairs(t)
	result, err := e.GetCoinMarginedFuturesPublicTrades(t.Context(), coinmTradablePair, 5)
	require.NoError(t, err, "GetCoinMarginedFuturesPublicTrades must not error")
	if mockTests {
		exp := []CoinMarginedFuturesPublicTradesResponse{
			{
				ID:                  1157458431,
				Price:               86118.7,
				QuantityInContracts: 1,
				BaseQuantity:        0.00116118,
				Time:                types.Time(time.UnixMilli(1791285176377)),
			},
			{
				ID:                  1157458434,
				Price:               86118.7,
				QuantityInContracts: 20,
				BaseQuantity:        0.02322375,
				Time:                types.Time(time.UnixMilli(1791285176377)),
				IsBuyerMaker:        true,
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesPublicTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesPublicTrades should return trades")
}

func TestGetCoinMarginedFuturesOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesOrderbookTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), currency.NewCode("BTCUSD"))
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetCoinMarginedFuturesOrderbookTicker must reject a symbol and a pair together")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []SymbolOrderBookTickerResponse
	}{
		{"symbol", coinmTradablePair, []SymbolOrderBookTickerResponse{
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
		{"every symbol", currency.EMPTYPAIR, []SymbolOrderBookTickerResponse{
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
			result, err := e.GetCoinMarginedFuturesOrderbookTicker(t.Context(), tc.symbol, currency.EMPTYCODE)
			require.NoError(t, err, "GetCoinMarginedFuturesOrderbookTicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCoinMarginedFuturesOrderbookTicker should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetCoinMarginedFuturesOrderbookTicker should return tickers")
		})
	}
}

func TestGetCoinMarginedFuturesSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesSymbolPriceTicker(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), currency.NewCode("BTCUSD"))
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetCoinMarginedFuturesSymbolPriceTicker must reject a symbol and a pair together")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		pair   currency.Code
		exp    []SymbolPriceTickerResponse
	}{
		{"symbol", coinmTradablePair, currency.EMPTYCODE, []SymbolPriceTickerResponse{
			{Symbol: "BTCUSD_PERP", Pair: "BTCUSD", Price: 86002.8, Time: types.Time(time.UnixMilli(1791282919817))},
		}},
		{"pair", currency.EMPTYPAIR, currency.NewCode("BTCUSD"), []SymbolPriceTickerResponse{
			{Symbol: "BTCUSD_PERP", Pair: "BTCUSD", Price: 86110.7, Time: types.Time(time.UnixMilli(1791285170154))},
			{Symbol: "BTCUSD_270326", Pair: "BTCUSD", Price: 88215.3, Time: types.Time(time.UnixMilli(1791285054739))},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCoinMarginedFuturesSymbolPriceTicker(t.Context(), tc.symbol, tc.pair)
			require.NoError(t, err, "GetCoinMarginedFuturesSymbolPriceTicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCoinMarginedFuturesSymbolPriceTicker should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetCoinMarginedFuturesSymbolPriceTicker should return tickers")
		})
	}
}

func TestGetCoinMarginedFuturesTakerVolume(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "ALL", func(r *CoinMarginedFuturesStatisticsRequest) error {
		_, err := e.GetCoinMarginedFuturesTakerVolume(t.Context(), r)
		return err
	})
	_, err := e.GetCoinMarginedFuturesTakerVolume(t.Context(), &CoinMarginedFuturesStatisticsRequest{Pair: currency.NewCode("BTCUSD"), Period: kline.FiveMin})
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetCoinMarginedFuturesTakerVolume must reject an empty contract type")

	result, err := e.GetCoinMarginedFuturesTakerVolume(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesTakerVolume must not error")
	if mockTests {
		exp := []TakerBuySellVolumeResponse{
			{
				Pair:                            "BTCUSD",
				ContractType:                    "ALL",
				TakerBuyVolumeInContracts:       11049,
				TakerSellVolumeInContracts:      24801,
				TakerBuyVolumeValueInBaseAsset:  0.1286,
				TakerSellVolumeValueInBaseAsset: 0.2888,
				Timestamp:                       types.Time(time.Unix(1791277800, 0)),
			},
			{
				Pair:                            "BTCUSD",
				ContractType:                    "ALL",
				TakerBuyVolumeInContracts:       8673,
				TakerSellVolumeInContracts:      2939,
				TakerBuyVolumeValueInBaseAsset:  0.1009,
				TakerSellVolumeValueInBaseAsset: 0.0342,
				Timestamp:                       types.Time(time.Unix(1791278100, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesTakerVolume should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesTakerVolume should return volumes")
}

func TestGetCoinMarginedFuturesSwapTickerChangeStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesSwapTickerChangeStats(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), currency.NewCode("BTCUSD"))
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetCoinMarginedFuturesSwapTickerChangeStats must reject a symbol and a pair together")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    []CoinMarginedFuturesPriceChangeStatsResponse
	}{
		{"symbol", coinmTradablePair, []CoinMarginedFuturesPriceChangeStatsResponse{
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
				VolumeInContracts:    7343590,
				BaseVolume:           8571.2,
				OpenTime:             types.Time(time.Unix(1791196500, 0)),
				CloseTime:            types.Time(time.UnixMilli(1791282919746)),
				FirstID:              1157292594,
				LastID:               1157454183,
				Count:                161590,
			},
		}},
		{"every symbol", currency.EMPTYPAIR, []CoinMarginedFuturesPriceChangeStatsResponse{
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
				VolumeInContracts:    7427276,
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
				VolumeInContracts:    35914142,
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
			result, err := e.GetCoinMarginedFuturesSwapTickerChangeStats(t.Context(), tc.symbol, currency.EMPTYCODE)
			require.NoError(t, err, "GetCoinMarginedFuturesSwapTickerChangeStats must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCoinMarginedFuturesSwapTickerChangeStats should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetCoinMarginedFuturesSwapTickerChangeStats should return statistics")
		})
	}
}

func TestGetCoinMarginedFuturesTraderAccountRatio(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "", func(r *CoinMarginedFuturesStatisticsRequest) error {
		_, err := e.GetCoinMarginedFuturesTraderAccountRatio(t.Context(), r)
		return err
	})
	result, err := e.GetCoinMarginedFuturesTraderAccountRatio(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesTraderAccountRatio must not error")
	if mockTests {
		exp := []CoinMarginedFuturesLongShortAccountRatioResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesTraderAccountRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesTraderAccountRatio should return ratios")
}

func TestGetCoinMarginedFuturesTraderPositionsRatio(t *testing.T) {
	t.Parallel()
	req := futuresStatisticsTests(t, "", func(r *CoinMarginedFuturesStatisticsRequest) error {
		_, err := e.GetCoinMarginedFuturesTraderPositionsRatio(t.Context(), r)
		return err
	})
	result, err := e.GetCoinMarginedFuturesTraderPositionsRatio(t.Context(), req)
	require.NoError(t, err, "GetCoinMarginedFuturesTraderPositionsRatio must not error")
	if mockTests {
		exp := []TopTraderPositionRatioResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesTraderPositionsRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCoinMarginedFuturesTraderPositionsRatio should return ratios")
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

func TestCoinMarginedFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesTradeHistory must reject a nil request")
	_, err = e.CoinMarginedFuturesTradeHistory(t.Context(), &CoinMarginedFuturesAccountTradeListRequest{})
	require.ErrorIs(t, err, errSymbolOrPairRequired, "CoinMarginedFuturesTradeHistory must require a symbol or a pair")
	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesTradeHistory(t.Context(), &CoinMarginedFuturesAccountTradeListRequest{Symbol: coinmTradablePair, Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "CoinMarginedFuturesTradeHistory must reject a symbol and a pair together")
	_, err = e.CoinMarginedFuturesTradeHistory(t.Context(), &CoinMarginedFuturesAccountTradeListRequest{Pair: currency.NewCode("BTCUSD"), FromID: 1})
	require.ErrorIs(t, err, errFromIDMutuallyExclusive, "CoinMarginedFuturesTradeHistory must reject a from ID with a pair")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.CoinMarginedFuturesTradeHistory(t.Context(), &CoinMarginedFuturesAccountTradeListRequest{Symbol: coinmTradablePair, StartTime: startTime, FromID: 1})
	require.ErrorIs(t, err, errFromIDMutuallyExclusive, "CoinMarginedFuturesTradeHistory must reject a from ID with a time window")
	_, err = e.CoinMarginedFuturesTradeHistory(t.Context(), &CoinMarginedFuturesAccountTradeListRequest{Symbol: coinmTradablePair, StartTime: endTime, EndTime: startTime, Limit: 5})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CoinMarginedFuturesTradeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// The endpoint also returns USDⓈ-M trades since the integration, which carry a quote quantity instead of a base one
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		pair   currency.Code
		exp    []CoinMarginedFuturesAccountTradeListResponse
	}{
		{"coin margined symbol", coinmTradablePair, currency.EMPTYCODE, []CoinMarginedFuturesAccountTradeListResponse{
			{
				Symbol:          "BTCUSD_PERP",
				ID:              1157458432,
				OrderID:         28,
				Pair:            "BTCUSD",
				Side:            "SELL",
				Price:           86118.7,
				Quantity:        180,
				RealisedPNL:     0.00001574,
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
		{"USDⓈ-M pair", currency.EMPTYPAIR, currency.NewCode("BTCUSDT"), []CoinMarginedFuturesAccountTradeListResponse{
			{
				Symbol:          "BTCUSDT",
				ID:              6096753380,
				OrderID:         8886774,
				Pair:            "BTCUSDT",
				Side:            "BUY",
				Price:           78950.1,
				Quantity:        0.002,
				RealisedPNL:     1.20532,
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
			result, err := e.CoinMarginedFuturesTradeHistory(t.Context(), &CoinMarginedFuturesAccountTradeListRequest{Symbol: tc.symbol, Pair: tc.pair, StartTime: startTime, EndTime: endTime, Limit: 5})
			require.NoError(t, err, "CoinMarginedFuturesTradeHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CoinMarginedFuturesTradeHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "CoinMarginedFuturesTradeHistory should return trades")
		})
	}
}

func TestGetAllCoinMarginedFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllCoinMarginedFuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllCoinMarginedFuturesOrders must reject a nil request")
	_, err = e.GetAllCoinMarginedFuturesOrders(t.Context(), &CoinMarginedFuturesAllOrdersRequest{Pair: currency.NewCode("BTCUSD"), OrderID: 1})
	require.ErrorIs(t, err, errOrderIDRequiresSymbol, "GetAllCoinMarginedFuturesOrders must reject an order ID with a pair")
	ensureTradablePairs(t)
	_, err = e.GetAllCoinMarginedFuturesOrders(t.Context(), &CoinMarginedFuturesAllOrdersRequest{Symbol: coinmTradablePair, Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, errSymbolAndPairMutuallyExclusive, "GetAllCoinMarginedFuturesOrders must reject a symbol and a pair together")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.GetAllCoinMarginedFuturesOrders(t.Context(), &CoinMarginedFuturesAllOrdersRequest{Symbol: coinmTradablePair, StartTime: endTime, EndTime: startTime, Limit: 2})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllCoinMarginedFuturesOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllCoinMarginedFuturesOrders(t.Context(), &CoinMarginedFuturesAllOrdersRequest{Symbol: coinmTradablePair, StartTime: startTime, EndTime: endTime, Limit: 2})
	require.NoError(t, err, "GetAllCoinMarginedFuturesOrders must not error")
	if mockTests {
		exp := []CoinMarginedFuturesAllOrdersResponse{
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
		assert.Equal(t, exp, result, "GetAllCoinMarginedFuturesOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllCoinMarginedFuturesOrders should return orders")
}

func TestCoinMarginedFuturesAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesAutoCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR, 30*time.Second)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesAutoCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.CoinMarginedFuturesAutoCancelAllOpenOrders(t.Context(), coinmTradablePair, 30*time.Second)
	require.NoError(t, err, "CoinMarginedFuturesAutoCancelAllOpenOrders must not error")
	if mockTests {
		exp := &CoinMarginedFuturesAutoCancelAllOpenOrdersResponse{
			Symbol:                      "BTCUSD_PERP",
			CountdownTimeInMilliseconds: 30000,
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesAutoCancelAllOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesAutoCancelAllOpenOrders should return the countdown")
}

func TestCoinMarginedFuturesCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.CoinMarginedFuturesCancelAllOpenOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "CoinMarginedFuturesCancelAllOpenOrders must not error")
	if mockTests {
		exp := &CoinMarginedFuturesStatusResponse{
			Code:    200,
			Message: "The operation of cancel all open order is done.",
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesCancelAllOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesCancelAllOpenOrders should return the status")
}

func TestFuturesOrderParams(t *testing.T) {
	t.Parallel()
	_, err := e.futuresOrderParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "futuresOrderParams must reject a nil request")
	_, err = e.futuresOrderParams(&CoinMarginedFuturesNewOrderRequest{Side: "BUY", OrderType: "LIMIT"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "futuresOrderParams must reject an empty symbol")
	symbol := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	_, err = e.futuresOrderParams(&CoinMarginedFuturesNewOrderRequest{Symbol: symbol, OrderType: "LIMIT"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "futuresOrderParams must reject an empty side")
	_, err = e.futuresOrderParams(&CoinMarginedFuturesNewOrderRequest{Symbol: symbol, Side: "BUY"})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "futuresOrderParams must reject an empty order type")

	params, err := e.futuresOrderParams(&CoinMarginedFuturesNewOrderRequest{
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
		NewOrderResponseType:    "RESULT",
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

	params, err = e.futuresOrderParams(&CoinMarginedFuturesNewOrderRequest{Symbol: symbol, Side: "BUY", OrderType: "MARKET"})
	require.NoError(t, err, "futuresOrderParams must not error")
	assert.Equal(t, url.Values{"symbol": {"BTCUSD_PERP"}, "side": {"BUY"}, "type": {"MARKET"}}, params, "futuresOrderParams should omit unset parameters")
}

func TestCoinMarginedFuturesModifyMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesModifyMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "CoinMarginedFuturesModifyMultipleOrders must reject no orders")
	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesModifyMultipleOrders(t.Context(), []CoinMarginedFuturesModifyOrderRequest{{Symbol: coinmTradablePair, OrderID: 1, Side: order.Buy.String(), Quantity: 1, PriceMatch: "QUEUE"}})
	require.ErrorIs(t, err, errBatchModifyPriceMatch, "CoinMarginedFuturesModifyMultipleOrders must reject a price match")
	_, err = e.CoinMarginedFuturesModifyMultipleOrders(t.Context(), []CoinMarginedFuturesModifyOrderRequest{{Symbol: coinmTradablePair, OrderID: 1, Side: order.Buy.String()}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CoinMarginedFuturesModifyMultipleOrders must reject an order without a quantity")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CoinMarginedFuturesModifyMultipleOrders(t.Context(), []CoinMarginedFuturesModifyOrderRequest{
		{Symbol: coinmTradablePair, OrderID: 20072994037, Side: order.Buy.String(), Quantity: 2, Price: 30005, ModifyID: 1},
		{Symbol: coinmTradablePair, OrigClientOrderID: "myOrder1", Side: order.Sell.String(), Quantity: 1, Price: 30006},
	})
	require.NoError(t, err, "CoinMarginedFuturesModifyMultipleOrders must not error")
	if mockTests {
		exp := []CoinMarginedFuturesModifyBatchOrderResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesModifyMultipleOrders should decode every field and each rejection")
	}
}

func TestCoinMarginedFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "CoinMarginedFuturesBatchOrder must reject an empty batch")
	_, err = e.CoinMarginedFuturesBatchOrder(t.Context(), []CoinMarginedFuturesNewOrderRequest{{Side: "SELL", OrderType: "STOP_MARKET", StopPrice: 60000, ClosePosition: true}})
	require.ErrorIs(t, err, errBatchClosePosition, "CoinMarginedFuturesBatchOrder must reject a close position order")
	_, err = e.CoinMarginedFuturesBatchOrder(t.Context(), []CoinMarginedFuturesNewOrderRequest{{Side: "BUY", OrderType: "LIMIT"}})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CoinMarginedFuturesBatchOrder must reject an order without a quantity")
	_, err = e.CoinMarginedFuturesBatchOrder(t.Context(), []CoinMarginedFuturesNewOrderRequest{{Side: "BUY", OrderType: "LIMIT", Quantity: 1}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesBatchOrder must reject an order without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.CoinMarginedFuturesBatchOrder(t.Context(), []CoinMarginedFuturesNewOrderRequest{
		{Symbol: coinmTradablePair, Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 1, Price: 10001},
		{Symbol: coinmTradablePair, Side: "SELL", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 1, Price: 9000, ReduceOnly: true},
	})
	require.NoError(t, err, "CoinMarginedFuturesBatchOrder must not error")
	if mockTests {
		exp := []CoinMarginedFuturesBatchOrderResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesBatchOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CoinMarginedFuturesBatchOrder should return an entry per order")
}

func TestCoinMarginedFuturesBatchCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesBatchCancelOrders(t.Context(), currency.EMPTYPAIR, []uint64{283194212}, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesBatchCancelOrders must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesBatchCancelOrders(t.Context(), coinmTradablePair, nil, nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CoinMarginedFuturesBatchCancelOrders must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CoinMarginedFuturesBatchCancelOrders(t.Context(), coinmTradablePair, []uint64{283194212}, []string{"myOrder2"})
	require.NoError(t, err, "CoinMarginedFuturesBatchCancelOrders must not error")
	if mockTests {
		exp := []CoinMarginedFuturesBatchOrderResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesBatchCancelOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CoinMarginedFuturesBatchCancelOrders should return an entry per order")
}

func TestCoinMarginedFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesGetOrderData(t.Context(), currency.EMPTYPAIR, 1573346959, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesGetOrderData must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesGetOrderData(t.Context(), coinmTradablePair, 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CoinMarginedFuturesGetOrderData must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CoinMarginedFuturesGetOrderData(t.Context(), coinmTradablePair, 1573346959, "")
	require.NoError(t, err, "CoinMarginedFuturesGetOrderData must not error")
	if mockTests {
		exp := &CoinMarginedFuturesOrderDetailResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesGetOrderData should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesGetOrderData should return the order")
}

func TestCoinMarginedFuturesModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesModifyOrder must reject a nil request")
	req := &CoinMarginedFuturesModifyOrderRequest{}
	_, err = e.CoinMarginedFuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesModifyOrder must reject an empty symbol")
	ensureTradablePairs(t)
	req.Symbol = coinmTradablePair
	_, err = e.CoinMarginedFuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CoinMarginedFuturesModifyOrder must reject a missing order ID")
	req.OrderID = 20072994037
	_, err = e.CoinMarginedFuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CoinMarginedFuturesModifyOrder must reject an empty side")
	req.Side = order.Buy.String()
	_, err = e.CoinMarginedFuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "CoinMarginedFuturesModifyOrder must reject an empty quantity")
	req.Quantity = 2
	_, err = e.CoinMarginedFuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "CoinMarginedFuturesModifyOrder must reject an empty price without a price match")
	req.Price = 30005
	req.PriceMatch = "QUEUE"
	_, err = e.CoinMarginedFuturesModifyOrder(t.Context(), req)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "CoinMarginedFuturesModifyOrder must reject a price match with a price")
	req.PriceMatch = ""

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.ModifyID = 1
	result, err := e.CoinMarginedFuturesModifyOrder(t.Context(), req)
	require.NoError(t, err, "CoinMarginedFuturesModifyOrder must not error")
	if mockTests {
		exp := &CoinMarginedFuturesModifyOrderResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesModifyOrder should decode every field")
	}
}

func TestCoinMarginedFuturesNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesNewOrder must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	result, err := e.CoinMarginedFuturesNewOrder(t.Context(), &CoinMarginedFuturesNewOrderRequest{Symbol: coinmTradablePair, Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 1, Price: 1, NewClientOrderID: "testOrder"})
	require.NoError(t, err, "CoinMarginedFuturesNewOrder must not error")
	if mockTests {
		exp := &CoinMarginedFuturesOrderResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesNewOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesNewOrder should return the order")
}

func TestCoinMarginedFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesCancelOrder(t.Context(), currency.EMPTYPAIR, 283194212, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesCancelOrder must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesCancelOrder(t.Context(), coinmTradablePair, 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CoinMarginedFuturesCancelOrder must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CoinMarginedFuturesCancelOrder(t.Context(), coinmTradablePair, 283194212, "myOrder1")
	require.NoError(t, err, "CoinMarginedFuturesCancelOrder must not error")
	if mockTests {
		exp := &CoinMarginedFuturesOrderResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesCancelOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesCancelOrder should return the order")
}

func TestCoinMarginedFuturesChangeInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesChangeInitialLeverage(t.Context(), currency.EMPTYPAIR, 21)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesChangeInitialLeverage must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesChangeInitialLeverage(t.Context(), coinmTradablePair, 0)
	require.ErrorIs(t, err, errLeverageRequired, "CoinMarginedFuturesChangeInitialLeverage must reject an empty leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CoinMarginedFuturesChangeInitialLeverage(t.Context(), coinmTradablePair, 21)
	require.NoError(t, err, "CoinMarginedFuturesChangeInitialLeverage must not error")
	if mockTests {
		exp := &CoinMarginedFuturesChangeLeverageResponse{
			Leverage:    21,
			MaxQuantity: 1000,
			Symbol:      "BTCUSD_PERP",
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesChangeInitialLeverage should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesChangeInitialLeverage should return the leverage")
}

func TestCoinMarginedFuturesChangeMarginType(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesChangeMarginType(t.Context(), currency.EMPTYPAIR, "ISOLATED")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesChangeMarginType must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesChangeMarginType(t.Context(), coinmTradablePair, "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "CoinMarginedFuturesChangeMarginType must reject an empty margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CoinMarginedFuturesChangeMarginType(t.Context(), coinmTradablePair, "ISOLATED")
	require.NoError(t, err, "CoinMarginedFuturesChangeMarginType must not error")
	if mockTests {
		exp := &CoinMarginedFuturesStatusResponse{
			Code:    200,
			Message: "success",
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesChangeMarginType should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesChangeMarginType should return the status")
}

func TestFuturesAlgoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesNewAlgoOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesNewAlgoOrder must reject a nil request")
	_, err = e.CoinMarginedFuturesNewAlgoOrder(t.Context(), &USDTMarginedFuturesNewAlgoOrderRequest{AlgoType: "CONDITIONAL"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesNewAlgoOrder must reject an empty symbol")
	_, err = e.CoinMarginedFuturesQueryAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CoinMarginedFuturesQueryAlgoOrder must require an algo ID or a client algo ID")
	_, err = e.CoinMarginedFuturesCancelAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CoinMarginedFuturesCancelAlgoOrder must require an algo ID or a client algo ID")
	err = e.CoinMarginedFuturesCancelAllAlgoOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesCancelAllAlgoOpenOrders must reject an empty symbol")
	_, err = e.CoinMarginedFuturesQueryAllAlgoOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesQueryAllAlgoOrders must reject a nil request")
	_, err = e.CoinMarginedFuturesQueryAllAlgoOrders(t.Context(), &USDTMarginedFuturesAllAlgoOrdersRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesQueryAllAlgoOrders must reject an empty symbol")

	// Binance gives COIN-M the USDⓈ-M algo order endpoints, so the requests and replies match USDⓈ-M's on the COIN-M host
	const algoOrder = `{"algoId":3100001,"clientAlgoId":"gct-cm-stop","algoType":"CONDITIONAL","orderType":"STOP_MARKET","symbol":"BTCUSD_PERP","side":"SELL","positionSide":"BOTH","timeInForce":"GTE_GTC","quantity":"1","algoStatus":"NEW","triggerPrice":"70000","price":"0","icebergQuantity":"null","selfTradePreventionMode":"EXPIRE_MAKER","workingType":"MARK_PRICE","priceMatch":"NONE","closePosition":false,"priceProtect":false,"reduceOnly":true,"createTime":1744120800000,"updateTime":1744120800000,"triggerTime":0,"goodTillDate":0}`
	pair := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	ex, sent := newCancelStepsExchange(t, []cancelStep{
		{http.MethodPost, "/dapi/v1/algoOrder", map[string]string{"algoType": "CONDITIONAL", "symbol": "BTCUSD_PERP", "side": "SELL", "type": "STOP_MARKET", "quantity": "1", "triggerPrice": "70000", "workingType": "MARK_PRICE", "reduceOnly": "true", "clientAlgoId": "gct-cm-stop"}, algoOrder},
		{http.MethodGet, "/dapi/v1/algoOrder", map[string]string{"algoId": "3100001", "clientAlgoId": ""}, algoOrder},
		{http.MethodGet, "/dapi/v1/openAlgoOrders", map[string]string{"symbol": "BTCUSD_PERP", "algoType": "CONDITIONAL"}, "[" + algoOrder + "]"},
		{http.MethodGet, "/dapi/v1/openAlgoOrders", map[string]string{"symbol": ""}, "[" + algoOrder + "]"},
		{http.MethodGet, "/dapi/v1/allAlgoOrders", map[string]string{"symbol": "BTCUSD_PERP", "startTime": "1744103854944", "endTime": "1744190254944", "limit": "100"}, "[" + algoOrder + "]"},
		{http.MethodDelete, "/dapi/v1/algoOrder", map[string]string{"clientAlgoId": "gct-cm-stop", "algoId": ""}, `{"algoId":3100001,"clientAlgoId":"gct-cm-stop","code":"200","msg":"success"}`},
		{http.MethodDelete, "/dapi/v1/algoOpenOrders", map[string]string{"symbol": "BTCUSD_PERP"}, `{"code":200,"msg":"The operation of cancel all open order is done."}`},
	})
	base := USDTMarginedFuturesAlgoOrderBase{
		AlgoID: 3100001, ClientAlgoID: "gct-cm-stop", AlgoType: "CONDITIONAL", OrderType: "STOP_MARKET", Symbol: "BTCUSD_PERP", Side: "SELL",
		PositionSide: "BOTH", TimeInForce: "GTE_GTC", Quantity: 1, AlgoStatus: "NEW", TriggerPrice: 70000, IcebergQuantity: "null",
		SelfTradePreventionMode: "EXPIRE_MAKER", WorkingType: "MARK_PRICE", PriceMatch: "NONE", ReduceOnly: true,
		CreateTime: types.Time(time.UnixMilli(1744120800000)), UpdateTime: types.Time(time.UnixMilli(1744120800000)),
	}
	placed, err := ex.CoinMarginedFuturesNewAlgoOrder(t.Context(), &USDTMarginedFuturesNewAlgoOrderRequest{AlgoType: "CONDITIONAL", Symbol: pair, Side: "SELL", OrderType: "STOP_MARKET", Quantity: 1, TriggerPrice: 70000, WorkingType: "MARK_PRICE", ReduceOnly: true, ClientAlgoID: "gct-cm-stop"})
	require.NoError(t, err, "CoinMarginedFuturesNewAlgoOrder must not error")
	assert.Equal(t, &USDTMarginedFuturesNewAlgoOrderResponse{USDTMarginedFuturesAlgoOrderBase: base}, placed, "CoinMarginedFuturesNewAlgoOrder should decode the placed algo order")
	queried, err := ex.CoinMarginedFuturesQueryAlgoOrder(t.Context(), 3100001, "gct-cm-stop")
	require.NoError(t, err, "CoinMarginedFuturesQueryAlgoOrder must not error")
	assert.Equal(t, &USDTMarginedFuturesAlgoOrderResponse{USDTMarginedFuturesAlgoOrderBase: base}, queried, "CoinMarginedFuturesQueryAlgoOrder should decode the algo order")
	open, err := ex.CoinMarginedFuturesCurrentAllAlgoOpenOrders(t.Context(), "CONDITIONAL", pair, 0)
	require.NoError(t, err, "CoinMarginedFuturesCurrentAllAlgoOpenOrders must not error")
	assert.Equal(t, []USDTMarginedFuturesAlgoOrderResponse{{USDTMarginedFuturesAlgoOrderBase: base}}, open, "CoinMarginedFuturesCurrentAllAlgoOpenOrders should decode the open algo orders")
	_, err = ex.CoinMarginedFuturesCurrentAllAlgoOpenOrders(t.Context(), "", currency.EMPTYPAIR, 0)
	require.NoError(t, err, "CoinMarginedFuturesCurrentAllAlgoOpenOrders must list every symbol's open algo orders")
	history, err := ex.CoinMarginedFuturesQueryAllAlgoOrders(t.Context(), &USDTMarginedFuturesAllAlgoOrdersRequest{Symbol: pair, StartTime: time.UnixMilli(1744103854944), EndTime: time.UnixMilli(1744190254944), Limit: 100})
	require.NoError(t, err, "CoinMarginedFuturesQueryAllAlgoOrders must not error")
	assert.Equal(t, []USDTMarginedFuturesAlgoOrderResponse{{USDTMarginedFuturesAlgoOrderBase: base}}, history, "CoinMarginedFuturesQueryAllAlgoOrders should decode the algo orders")
	cancelled, err := ex.CoinMarginedFuturesCancelAlgoOrder(t.Context(), 0, "gct-cm-stop")
	require.NoError(t, err, "CoinMarginedFuturesCancelAlgoOrder must not error")
	assert.Equal(t, &USDTMarginedFuturesCancelAlgoOrderResponse{AlgoID: 3100001, ClientAlgoID: "gct-cm-stop", Code: 200, Message: "success"}, cancelled, "CoinMarginedFuturesCancelAlgoOrder should decode the cancellation")
	require.NoError(t, ex.CoinMarginedFuturesCancelAllAlgoOpenOrders(t.Context(), pair), "CoinMarginedFuturesCancelAllAlgoOpenOrders must not error")
	assert.Equal(t, 7, sent(), "the COIN-M algo order endpoints should send every expected request")
}

func TestGetCoinMarginedFuturesAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	result, err := e.GetCoinMarginedFuturesAllOpenOrders(t.Context(), coinmTradablePair, currency.EMPTYCODE)
	require.NoError(t, err, "GetCoinMarginedFuturesAllOpenOrders must not error")
	if mockTests {
		exp := []CoinMarginedFuturesOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesAllOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCoinMarginedFuturesAllOpenOrders should return orders")
}

func TestGetCoinMarginedFuturesOrderModifyHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCoinMarginedFuturesOrderModifyHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCoinMarginedFuturesOrderModifyHistory must reject a nil request")
	_, err = e.GetCoinMarginedFuturesOrderModifyHistory(t.Context(), &CoinMarginedFuturesOrderModifyHistoryRequest{OrderID: 20072994037})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCoinMarginedFuturesOrderModifyHistory must reject an empty symbol")
	ensureTradablePairs(t)
	_, err = e.GetCoinMarginedFuturesOrderModifyHistory(t.Context(), &CoinMarginedFuturesOrderModifyHistoryRequest{Symbol: coinmTradablePair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetCoinMarginedFuturesOrderModifyHistory must reject a missing order ID")
	startTime, endTime := getTime()
	_, err = e.GetCoinMarginedFuturesOrderModifyHistory(t.Context(), &CoinMarginedFuturesOrderModifyHistoryRequest{Symbol: coinmTradablePair, OrderID: 20072994037, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCoinMarginedFuturesOrderModifyHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCoinMarginedFuturesOrderModifyHistory(t.Context(), &CoinMarginedFuturesOrderModifyHistoryRequest{Symbol: coinmTradablePair, OrderID: 20072994037, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetCoinMarginedFuturesOrderModifyHistory must not error")
	if mockTests {
		exp := []FuturesOrderAmendmentResponse{
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
		assert.Equal(t, exp, result, "GetCoinMarginedFuturesOrderModifyHistory should decode every field")
	}
}

func TestCoinMarginedFuturesMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesMarginChangeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesMarginChangeHistory must reject a nil request")
	_, err = e.CoinMarginedFuturesMarginChangeHistory(t.Context(), &CoinMarginedFuturesPositionMarginHistoryRequest{Type: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesMarginChangeHistory must reject an empty symbol")

	ensureTradablePairs(t)
	startTime, endTime := getTime()
	_, err = e.CoinMarginedFuturesMarginChangeHistory(t.Context(), &CoinMarginedFuturesPositionMarginHistoryRequest{Symbol: coinmTradablePair, Type: 1, StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CoinMarginedFuturesMarginChangeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CoinMarginedFuturesMarginChangeHistory(t.Context(), &CoinMarginedFuturesPositionMarginHistoryRequest{Symbol: coinmTradablePair, Type: 1, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "CoinMarginedFuturesMarginChangeHistory must not error")
	if mockTests {
		exp := []GetPositionMarginChangeHistoryResponse{
			{
				Amount:       23.36332311,
				Asset:        currency.BTC,
				Symbol:       "BTCUSD_PERP",
				Time:         types.Time(time.Unix(1744120800, 0)),
				Type:         1,
				PositionSide: "BOTH",
			},
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesMarginChangeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesMarginChangeHistory should return margin changes")
}

func TestCoinMarginedFuturesModifyIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesModifyIsolatedPositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesModifyIsolatedPositionMargin must reject a nil request")
	_, err = e.CoinMarginedFuturesModifyIsolatedPositionMargin(t.Context(), &CoinMarginedFuturesModifyIsolatedPositionMarginRequest{Amount: 100, Type: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesModifyIsolatedPositionMargin must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesModifyIsolatedPositionMargin(t.Context(), &CoinMarginedFuturesModifyIsolatedPositionMarginRequest{Symbol: coinmTradablePair, Type: 1})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CoinMarginedFuturesModifyIsolatedPositionMargin must reject an empty amount")
	_, err = e.CoinMarginedFuturesModifyIsolatedPositionMargin(t.Context(), &CoinMarginedFuturesModifyIsolatedPositionMarginRequest{Symbol: coinmTradablePair, Amount: 100})
	require.ErrorIs(t, err, errMarginChangeTypeInvalid, "CoinMarginedFuturesModifyIsolatedPositionMargin must reject an empty change type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CoinMarginedFuturesModifyIsolatedPositionMargin(t.Context(), &CoinMarginedFuturesModifyIsolatedPositionMarginRequest{Symbol: coinmTradablePair, PositionSide: "BOTH", Amount: 100, Type: 1})
	require.NoError(t, err, "CoinMarginedFuturesModifyIsolatedPositionMargin must not error")
	if mockTests {
		exp := &CoinMarginedFuturesMarginUpdatedResponse{
			Amount:  100,
			Code:    200,
			Message: "Successfully modify position margin.",
			Type:    1,
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesModifyIsolatedPositionMargin should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesModifyIsolatedPositionMargin should return the result")
}

func TestCoinMarginedFuturesPositionsAutoDeleveragingEstimate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CoinMarginedFuturesPositionsAutoDeleveragingEstimate(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "CoinMarginedFuturesPositionsAutoDeleveragingEstimate must not error")
	if mockTests {
		exp := []AutoDeleveragingEstimateResponse{
			{
				Symbol: "BTCUSD_PERP",
				AutoDeleveragingQuantile: CoinMarginedFuturesAutoDeleveragingQuantile{
					Long:  2,
					Short: 2,
					Hedge: 2,
				},
			},
			{
				Symbol: "ETHUSD_PERP",
				AutoDeleveragingQuantile: CoinMarginedFuturesAutoDeleveragingQuantile{
					Both: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesPositionsAutoDeleveragingEstimate should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesPositionsAutoDeleveragingEstimate should return quantiles")
}

func TestCoinMarginedFuturesPositionsInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CoinMarginedFuturesPositionsInfo(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.NoError(t, err, "CoinMarginedFuturesPositionsInfo must not error")
	if mockTests {
		exp := []CoinMarginedFuturesPositionInformationResponse{
			{
				Symbol:           "BTCUSD_PERP",
				PositionAmount:   3,
				EntryPrice:       85642.5,
				BreakEvenPrice:   85676.8,
				MarkPrice:        86068.2,
				UnrealisedProfit: -0.00000081,
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesPositionsInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesPositionsInfo should return positions")
}

func TestCoinMarginedFuturesOpenOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesOpenOrderData(t.Context(), currency.EMPTYPAIR, 1917641, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CoinMarginedFuturesOpenOrderData must reject an empty symbol")

	ensureTradablePairs(t)
	_, err = e.CoinMarginedFuturesOpenOrderData(t.Context(), coinmTradablePair, 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CoinMarginedFuturesOpenOrderData must require an order ID or a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CoinMarginedFuturesOpenOrderData(t.Context(), coinmTradablePair, 1917641, "")
	require.NoError(t, err, "CoinMarginedFuturesOpenOrderData must not error")
	if mockTests {
		exp := &CoinMarginedFuturesOrderDetailResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesOpenOrderData should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesOpenOrderData should return the order")
}

func TestCoinMarginedFuturesForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CoinMarginedFuturesForceOrders must reject a nil request")

	startTime, endTime := getTime()
	_, err = e.CoinMarginedFuturesForceOrders(t.Context(), &CoinMarginedFuturesForceOrdersRequest{AutoCloseType: "ADL", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "CoinMarginedFuturesForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CoinMarginedFuturesForceOrders(t.Context(), &CoinMarginedFuturesForceOrdersRequest{AutoCloseType: "ADL", StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "CoinMarginedFuturesForceOrders must not error")
	if mockTests {
		exp := []ForcedOrdersResponse{
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
		assert.Equal(t, exp, result, "CoinMarginedFuturesForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CoinMarginedFuturesForceOrders should return orders")
}

func TestCoinMarginedFuturesQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.CoinMarginedFuturesQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CoinMarginedFuturesQuarterlyContractSettlementPrice must reject an empty pair")

	result, err := e.CoinMarginedFuturesQuarterlyContractSettlementPrice(t.Context(), currency.NewCode("BTCUSD"))
	require.NoError(t, err, "CoinMarginedFuturesQuarterlyContractSettlementPrice must not error")
	if mockTests {
		exp := []SettlementPriceResponse{
			{
				DeliveryTime:  types.Time(time.Unix(1782432000, 0)),
				DeliveryPrice: 60510.3,
			},
			{
				DeliveryTime:  types.Time(time.Unix(1774569600, 0)),
				DeliveryPrice: 68597.7,
			},
		}
		assert.Equal(t, exp, result, "CoinMarginedFuturesQuarterlyContractSettlementPrice should decode every field")
		return
	}
	assert.NotEmpty(t, result, "CoinMarginedFuturesQuarterlyContractSettlementPrice should return settlement prices")
}

func TestKeepaliveCoinMarginedFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		_, err := e.StartCoinMarginedFuturesUserDataStream(t.Context())
		require.NoError(t, err, "StartCoinMarginedFuturesUserDataStream must not error")
	}
	result, err := e.KeepaliveCoinMarginedFuturesUserDataStream(t.Context())
	require.NoError(t, err, "KeepaliveCoinMarginedFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "KeepaliveCoinMarginedFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "KeepaliveCoinMarginedFuturesUserDataStream should return the listen key")
}

func TestStartCoinMarginedFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.StartCoinMarginedFuturesUserDataStream(t.Context())
	require.NoError(t, err, "StartCoinMarginedFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "StartCoinMarginedFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "StartCoinMarginedFuturesUserDataStream should return a listen key")
}

func TestCloseCoinMarginedFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		// Closing invalidates the listen key every client of the account shares
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CloseCoinMarginedFuturesUserDataStream(t.Context()), "CloseCoinMarginedFuturesUserDataStream should not error")
}
