package binance

import (
	"context"
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

func TestUAccountInformationV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UAccountInformationV2(t.Context())
	require.NoError(t, err, "UAccountInformationV2 must not error")
	if mockTests {
		exp := &UAccountInformationV2Response{
			FeeTier:                     1,
			FeeBurn:                     true,
			CanTrade:                    true,
			CanDeposit:                  true,
			CanWithdraw:                 true,
			UpdateTime:                  types.Time(time.UnixMilli(1625474304765)),
			MultiAssetsMargin:           true,
			TradeGroupID:                -1,
			TotalInitialMargin:          0.33683,
			TotalMaintenanceMargin:      0.02695,
			TotalWalletBalance:          126.72469206,
			TotalUnrealizedProfit:       0.00211,
			TotalMarginBalance:          126.72680206,
			TotalPositionInitialMargin:  0.33683,
			TotalOpenOrderInitialMargin: 0.125,
			TotalCrossWalletBalance:     126.72469206,
			TotalCrossUnrealizedPNL:     0.00211,
			AvailableBalance:            126.26497206,
			MaxWithdrawAmount:           126.26497206,
			Assets: []UAsset{
				{
					Asset:                  currency.USDT,
					WalletBalance:          23.72469206,
					UnrealizedProfit:       0.00211,
					MarginBalance:          23.72680206,
					MaintenanceMargin:      0.02695,
					InitialMargin:          0.46183,
					PositionInitialMargin:  0.33683,
					OpenOrderInitialMargin: 0.125,
					CrossWalletBalance:     23.72469206,
					CrossUnrealizedPNL:     0.00211,
					AvailableBalance:       23.26497206,
					MaxWithdrawAmount:      23.26497206,
					MarginAvailable:        true,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []UPosition{
				{
					Symbol:                 "BTCUSDT",
					InitialMargin:          0.33683,
					MaintenanceMargin:      0.02695,
					UnrealizedProfit:       0.00211,
					PositionInitialMargin:  0.33683,
					OpenOrderInitialMargin: 0.125,
					Leverage:               100,
					Isolated:               true,
					EntryPrice:             33681.1,
					MaxNotional:            250000,
					BidNotional:            12.5,
					AskNotional:            6.25,
					PositionSide:           "BOTH",
					PositionAmount:         0.001,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
		}
		assert.Equal(t, exp, result, "UAccountInformationV2 should decode every field")
		return
	}
	assert.NotNil(t, result, "UAccountInformationV2 should return account information")
}

func TestUAccountInformationV3(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UAccountInformationV3(t.Context())
	require.NoError(t, err, "UAccountInformationV3 must not error")
	if mockTests {
		exp := &UAccountInformationV3Response{
			TotalInitialMargin:          0.33683,
			TotalMaintenanceMargin:      0.02695,
			TotalWalletBalance:          126.72469206,
			TotalUnrealizedProfit:       0.00211,
			TotalMarginBalance:          126.72680206,
			TotalPositionInitialMargin:  0.33683,
			TotalOpenOrderInitialMargin: 0.125,
			TotalCrossWalletBalance:     126.72469206,
			TotalCrossUnrealizedPNL:     0.00211,
			AvailableBalance:            126.26497206,
			MaxWithdrawAmount:           126.26497206,
			Assets: []UAssetV3{
				{
					Asset:                  currency.USDT,
					WalletBalance:          23.72469206,
					UnrealizedProfit:       0.00211,
					MarginBalance:          23.72680206,
					MaintenanceMargin:      0.02695,
					InitialMargin:          0.46183,
					PositionInitialMargin:  0.33683,
					OpenOrderInitialMargin: 0.125,
					CrossWalletBalance:     23.72469206,
					CrossUnrealizedPNL:     0.00211,
					AvailableBalance:       23.26497206,
					MaxWithdrawAmount:      23.26497206,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []UPositionV3{
				{
					Symbol:            "BTCUSDT",
					PositionSide:      "BOTH",
					PositionAmount:    0.001,
					UnrealizedProfit:  0.00211,
					IsolatedMargin:    0.33894,
					Notional:          33.68321,
					IsolatedWallet:    0.33683,
					InitialMargin:     0.33683,
					MaintenanceMargin: 0.02695,
					UpdateTime:        types.Time(time.UnixMilli(1625474304765)),
				},
			},
		}
		assert.Equal(t, exp, result, "UAccountInformationV3 should decode every field")
		return
	}
	assert.NotNil(t, result, "UAccountInformationV3 should return account information")
}

func TestUAccountBalanceV3(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UAccountBalanceV3(t.Context())
	require.NoError(t, err, "UAccountBalanceV3 must not error")
	if mockTests {
		exp := []UAccountBalance{
			{
				AccountAlias:       "SgsR",
				Asset:              currency.USDT,
				Balance:            122607.35137903,
				CrossWalletBalance: 23.72469206,
				CrossUnrealizedPNL: -0.00211,
				AvailableBalance:   23.72469206,
				MaxWithdrawAmount:  23.72469206,
				MarginAvailable:    true,
				UpdateTime:         types.Time(time.UnixMilli(1617939110373)),
			},
		}
		assert.Equal(t, exp, result, "UAccountBalanceV3 should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UAccountBalanceV3 should return balances")
}

func TestUFuturesTradingQuantitativeRulesIndicators(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UFuturesTradingQuantitativeRulesIndicators(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UFuturesTradingQuantitativeRulesIndicators must not error for a symbol")
	if mockTests {
		exp := &TradingQuantitativeRulesIndicatorsResponse{
			Indicators: map[string][]TradingQuantitativeRulesIndicator{
				"ACCOUNT": {
					{
						IsLocked:           true,
						PlannedRecoverTime: types.Time(time.Unix(1644919865, 0)),
						Indicator:          "TMV",
						Value:              10,
						TriggerValue:       1,
					},
				},
				"BTCUSDT": {
					{
						IsLocked:           true,
						PlannedRecoverTime: types.Time(time.Unix(1545741270, 0)),
						Indicator:          "UFR",
						Value:              0.05,
						TriggerValue:       0.995,
					},
				},
			},
			UpdateTime: types.Time(time.UnixMilli(1644913304748)),
		}
		assert.Equal(t, exp, result, "UFuturesTradingQuantitativeRulesIndicators should decode every field for a symbol")
	}
	result, err = e.UFuturesTradingQuantitativeRulesIndicators(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "UFuturesTradingQuantitativeRulesIndicators must not error for every symbol")
	if mockTests {
		exp := &TradingQuantitativeRulesIndicatorsResponse{
			Indicators: map[string][]TradingQuantitativeRulesIndicator{
				"ACCOUNT": {
					{
						IsLocked:           true,
						PlannedRecoverTime: types.Time(time.Unix(1644919865, 0)),
						Indicator:          "TMV",
						Value:              10,
						TriggerValue:       1,
					},
				},
				"BTCUSDT": {
					{
						IsLocked:           true,
						PlannedRecoverTime: types.Time(time.Unix(1545741270, 0)),
						Indicator:          "UFR",
						Value:              0.05,
						TriggerValue:       0.995,
					},
				},
				"ETHUSDT": {
					{
						IsLocked:           true,
						PlannedRecoverTime: types.Time(time.Unix(1545741270, 0)),
						Indicator:          "IFER",
						Value:              0.99,
						TriggerValue:       0.99,
					},
				},
			},
			UpdateTime: types.Time(time.UnixMilli(1644913304748)),
		}
		assert.Equal(t, exp, result, "UFuturesTradingQuantitativeRulesIndicators should decode every field for every symbol")
		return
	}
	assert.NotNil(t, result, "UFuturesTradingQuantitativeRulesIndicators should return indicators")
}

func TestGetAssetsMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAssetsMode(t.Context())
	require.NoError(t, err, "GetAssetsMode must not error")
	if mockTests {
		assert.True(t, result, "GetAssetsMode should report Multi-Assets Mode")
	}
}

func TestGetCurrentPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentPositionMode(t.Context())
	require.NoError(t, err, "GetCurrentPositionMode must not error")
	if mockTests {
		exp := &UPositionModeResponse{
			DualSidePosition: true,
		}
		assert.Equal(t, exp, result, "GetCurrentPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrentPositionMode should return the position mode")
}

func TestUFuturesDownloadIDs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		fn   func(*testing.T, time.Time, time.Time) (*UDownloadIDResponse, error)
	}{
		{"UFuturesOrderHistoryDownloadID", func(t *testing.T, start, end time.Time) (*UDownloadIDResponse, error) {
			t.Helper()
			return e.UFuturesOrderHistoryDownloadID(t.Context(), start, end)
		}},
		{"UFuturesTradeHistoryDownloadID", func(t *testing.T, start, end time.Time) (*UDownloadIDResponse, error) {
			t.Helper()
			return e.UFuturesTradeHistoryDownloadID(t.Context(), start, end)
		}},
		{"UFuturesTransactionHistoryDownloadID", func(t *testing.T, start, end time.Time) (*UDownloadIDResponse, error) {
			t.Helper()
			return e.UFuturesTransactionHistoryDownloadID(t.Context(), start, end)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			startTime, endTime := getTime()
			_, err := tc.fn(t, time.Time{}, endTime)
			require.ErrorIs(t, err, common.ErrDateUnset, "must reject a missing start time")
			_, err = tc.fn(t, endTime, startTime)
			require.ErrorIs(t, err, common.ErrStartAfterEnd, "must reject a start time after the end time")

			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := tc.fn(t, startTime, endTime)
			require.NoError(t, err, "must not error")
			if mockTests {
				exp := &UDownloadIDResponse{
					AverageCostTimestampOfLast30Days: 7241837,
					DownloadID:                       "546975389218332672",
				}
				assert.Equal(t, exp, result, "should decode every field")
				return
			}
			assert.NotEmpty(t, result.DownloadID, "should return a download ID")
		})
	}
}

func TestUFuturesDownloadLinks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		fn   func(*testing.T, string) (*UDownloadLinkResponse, error)
	}{
		{"UFuturesOrderHistoryDownloadLink", func(t *testing.T, id string) (*UDownloadLinkResponse, error) {
			t.Helper()
			return e.UFuturesOrderHistoryDownloadLink(t.Context(), id)
		}},
		{"UFuturesTradeHistoryDownloadLink", func(t *testing.T, id string) (*UDownloadLinkResponse, error) {
			t.Helper()
			return e.UFuturesTradeHistoryDownloadLink(t.Context(), id)
		}},
		{"UFuturesTransactionHistoryDownloadLink", func(t *testing.T, id string) (*UDownloadLinkResponse, error) {
			t.Helper()
			return e.UFuturesTransactionHistoryDownloadLink(t.Context(), id)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.fn(t, "")
			require.ErrorIs(t, err, errDownloadIDRequired, "must reject an empty download ID")

			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := tc.fn(t, "546975389218332672")
			require.NoError(t, err, "must not error")
			if mockTests {
				exp := &UDownloadLinkResponse{
					DownloadID:          "546975389218332672",
					Status:              "completed",
					URL:                 "https://bin-prod-user-rebate-bucket.s3.amazonaws.com/future-order-history/example.csv",
					Notified:            true,
					ExpirationTimestamp: 1645009771000,
					IsExpired:           true,
				}
				assert.Equal(t, exp, result, "should decode every field")
				return
			}
			assert.NotEmpty(t, result.Status, "should return the download status")
		})
	}
}

func TestUAccountIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.UAccountIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UAccountIncomeHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.UAccountIncomeHistory(t.Context(), &UIncomeHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UAccountIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UAccountIncomeHistory(t.Context(), &UIncomeHistoryRequest{
		Symbol:     currency.NewBTCUSDT(),
		IncomeType: "REALIZED_PNL",
		StartTime:  startTime,
		EndTime:    endTime,
		Page:       1,
		Limit:      5,
	})
	require.NoError(t, err, "UAccountIncomeHistory must not error")
	if mockTests {
		exp := []UAccountIncomeHistory{
			{
				Symbol:        "BTCUSDT",
				IncomeType:    "REALIZED_PNL",
				Income:        -0.375,
				Asset:         currency.USDT,
				Info:          "REALIZED_PNL",
				Time:          types.Time(time.Unix(1744118400, 0)),
				TransactionID: 9689322392,
				TradeID:       "2059192",
			},
		}
		assert.Equal(t, exp, result, "UAccountIncomeHistory should decode every field")
	}
}

func TestUGetNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UGetNotionalAndLeverageBrackets(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UGetNotionalAndLeverageBrackets must not error for a symbol")
	if mockTests {
		exp := []UNotionalLeverageAndBrackets{
			{
				Symbol:              "BTCUSDT",
				NotionalCoefficient: 1.5,
				Brackets: []UNotionalBracket{
					{
						Bracket:                1,
						InitialLeverage:        75,
						NotionalCap:            10000,
						NotionalFloor:          5000,
						MaintenanceMarginRatio: 0.0065,
						Cumulative:             0.5,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "UGetNotionalAndLeverageBrackets should decode the single symbol object")
	}
	result, err = e.UGetNotionalAndLeverageBrackets(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "UGetNotionalAndLeverageBrackets must not error for every symbol")
	if mockTests {
		exp := []UNotionalLeverageAndBrackets{
			{
				Symbol:              "BTCUSDT",
				NotionalCoefficient: 1.5,
				Brackets: []UNotionalBracket{
					{
						Bracket:                1,
						InitialLeverage:        75,
						NotionalCap:            10000,
						NotionalFloor:          5000,
						MaintenanceMarginRatio: 0.0065,
						Cumulative:             0.5,
					},
				},
			},
			{
				Symbol:              "ETHUSDT",
				NotionalCoefficient: 1.5,
				Brackets: []UNotionalBracket{
					{
						Bracket:                2,
						InitialLeverage:        50,
						NotionalCap:            50000,
						NotionalFloor:          10000,
						MaintenanceMarginRatio: 0.01,
						Cumulative:             7.5,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "UGetNotionalAndLeverageBrackets should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "UGetNotionalAndLeverageBrackets should return brackets")
}

func TestGetUSDTUserRateLimits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUSDTUserRateLimits(t.Context())
	require.NoError(t, err, "GetUSDTUserRateLimits must not error")
	if mockTests {
		exp := []RateLimitInfo{
			{
				RateLimitType: "ORDERS",
				Interval:      "SECOND",
				IntervalNum:   10,
				Limit:         300,
			},
			{
				RateLimitType: "ORDERS",
				Interval:      "MINUTE",
				IntervalNum:   1,
				Limit:         1200,
			},
		}
		assert.Equal(t, exp, result, "GetUSDTUserRateLimits should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTUserRateLimits should return rate limits")
}

func TestUGetCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := e.UGetCommissionRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UGetCommissionRates must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UGetCommissionRates(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UGetCommissionRates must not error")
	if mockTests {
		exp := &UCommissionRateResponse{
			Symbol:              "BTCUSDT",
			MakerCommissionRate: 0.0002,
			TakerCommissionRate: 0.0004,
			RPICommissionRate:   0.00005,
		}
		assert.Equal(t, exp, result, "UGetCommissionRates should decode every field")
		return
	}
	assert.Equal(t, "BTCUSDT", result.Symbol, "UGetCommissionRates should return the requested symbol")
}

func TestGetBasis(t *testing.T) {
	t.Parallel()
	_, err := e.GetBasis(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBasis must reject a nil request")
	arg := &UBasisRequest{ContractType: "CURRENT_QUARTER", Period: "15m"}
	_, err = e.GetBasis(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetBasis must reject an empty pair")
	arg.Pair = currency.NewBTCUSDT()
	arg.ContractType = ""
	_, err = e.GetBasis(t.Context(), arg)
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetBasis must reject an empty contract type")
	arg.ContractType = "CURRENT_QUARTER"
	arg.Period = ""
	_, err = e.GetBasis(t.Context(), arg)
	require.ErrorIs(t, err, errPeriodRequired, "GetBasis must reject an empty period")
	arg.Period = "15m"
	startTime, endTime := getTime()
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetBasis(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBasis must reject a start time after the end time")

	arg.StartTime, arg.EndTime, arg.Limit = startTime, endTime, 20
	result, err := e.GetBasis(t.Context(), arg)
	require.NoError(t, err, "GetBasis must not error")
	if mockTests {
		exp := []BasisInfo{
			{
				IndexPrice:          86002.56086957,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0108,
				FuturesPrice:        86928.7,
				AnnualizedBasisRate: 0.0493,
				Basis:               926.13913043,
				Pair:                "BTCUSDT",
				Timestamp:           types.Time(time.Unix(1791280800, 0)),
			},
			{
				IndexPrice:          86159.96913043,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0108,
				FuturesPrice:        87087.9,
				AnnualizedBasisRate: 0.0493,
				Basis:               927.93086957,
				Pair:                "BTCUSDT",
				Timestamp:           types.Time(time.Unix(1791281700, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetBasis should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetBasis should return basis records")
}

func TestUServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.UServerTime(t.Context())
	require.NoError(t, err, "UServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1791283479731), result, "UServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "UServerTime should return the current time")
}

func TestUCompositeIndexInfo(t *testing.T) {
	t.Parallel()
	result, err := e.UCompositeIndexInfo(t.Context(), currency.NewPair(currency.NewCode("SMALL"), currency.USDT))
	require.NoError(t, err, "UCompositeIndexInfo must not error for a symbol")
	if mockTests {
		exp := []UCompositeIndex{
			{
				Symbol:    "SMALLUSDT",
				Time:      types.Time(time.UnixMilli(1791283544001)),
				Component: "baseAsset",
				BaseAssetList: []UCompositeIndexBaseAsset{
					{
						BaseAsset:          currency.NewCode("1000CAT"),
						QuoteAsset:         currency.USDT,
						WeightInQuantity:   5.42858919,
						WeightInPercentage: 0.01,
					},
					{
						BaseAsset:          currency.NewCode("4"),
						QuoteAsset:         currency.USDT,
						WeightInQuantity:   0.6687129,
						WeightInPercentage: 0.01,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "UCompositeIndexInfo should decode the single symbol object")
	}
	result, err = e.UCompositeIndexInfo(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "UCompositeIndexInfo must not error for every symbol")
	if mockTests {
		exp := []UCompositeIndex{
			{
				Symbol:    "ALLUSDT",
				Time:      types.Time(time.Unix(1791283499, 0)),
				Component: "baseAsset",
				BaseAssetList: []UCompositeIndexBaseAsset{
					{
						BaseAsset:          currency.NewCode("0G"),
						QuoteAsset:         currency.USDT,
						WeightInQuantity:   0.0043106,
						WeightInPercentage: 0.00191571,
					},
					{
						BaseAsset:          currency.NewCode("1000000MOG"),
						QuoteAsset:         currency.USDT,
						WeightInQuantity:   0.01101495,
						WeightInPercentage: 0.00191571,
					},
				},
			},
			{
				Symbol:    "BTCDOMUSDT",
				Time:      types.Time(time.Unix(1791283499, 0)),
				Component: "quoteAsset",
				BaseAssetList: []UCompositeIndexBaseAsset{
					{
						BaseAsset:          currency.BTC,
						QuoteAsset:         currency.NewCode("AAVE"),
						WeightInQuantity:   0.03389094,
						WeightInPercentage: 0.003359,
					},
					{
						BaseAsset:          currency.BTC,
						QuoteAsset:         currency.ADA,
						WeightInQuantity:   0.0001888,
						WeightInPercentage: 0.012384,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "UCompositeIndexInfo should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "UCompositeIndexInfo should return composite indexes")
}

func TestUCompressedTrades(t *testing.T) {
	t.Parallel()
	_, err := e.UCompressedTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UCompressedTrades must reject a nil request")
	_, err = e.UCompressedTrades(t.Context(), &UCompressedTradesRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UCompressedTrades must reject an empty symbol")
	startTime, endTime := getTime(true)
	_, err = e.UCompressedTrades(t.Context(), &UCompressedTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UCompressedTrades must reject a start time after the end time")

	if !mockTests {
		// Binance only serves the last 48 hours, and a window must be shorter than an hour
		endTime = time.Now()
		startTime = endTime.Add(-30 * time.Minute)
	}
	result, err := e.UCompressedTrades(t.Context(), &UCompressedTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "UCompressedTrades must not error for a time window")
	require.NotEmpty(t, result, "UCompressedTrades must return trades")
	if mockTests {
		exp := []UCompressedTradeData{
			{
				AggregateTradeID: 3476467420,
				Price:            86163.1,
				Quantity:         0.009,
				NormalQuantity:   0.009,
				FirstTradeID:     8149539168,
				LastTradeID:      8149539168,
				Timestamp:        types.Time(time.UnixMilli(1791283482175)),
			},
			{
				AggregateTradeID: 3476467421,
				Price:            86163.1,
				Quantity:         0.35,
				NormalQuantity:   0.35,
				FirstTradeID:     8149539169,
				LastTradeID:      8149539175,
				Timestamp:        types.Time(time.UnixMilli(1791283482359)),
				IsBuyerMaker:     true,
			},
		}
		assert.Equal(t, exp, result, "UCompressedTrades should decode every field")
	}
	_, err = e.UCompressedTrades(t.Context(), &UCompressedTradesRequest{Symbol: currency.NewBTCUSDT(), FromID: result[0].AggregateTradeID, Limit: 5})
	assert.NoError(t, err, "UCompressedTrades should not error from a trade ID")
}

func TestGetUFuturesContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetUFuturesContinuousKlineData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUFuturesContinuousKlineData must reject a nil request")
	arg := &UContinuousKlineRequest{ContractType: "PERPETUAL", Interval: "1d"}
	_, err = e.GetUFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUFuturesContinuousKlineData must reject an empty pair")
	arg.Pair = currency.NewBTCUSDT()
	arg.ContractType = ""
	_, err = e.GetUFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetUFuturesContinuousKlineData must reject an empty contract type")
	arg.ContractType = "PERPETUAL"
	arg.Interval = ""
	_, err = e.GetUFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetUFuturesContinuousKlineData must reject an empty interval")
	arg.Interval = "1d"
	startTime, endTime := getTime()
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetUFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUFuturesContinuousKlineData must reject a start time after the end time")

	arg.StartTime, arg.EndTime, arg.Limit = startTime, endTime, 10
	result, err := e.GetUFuturesContinuousKlineData(t.Context(), arg)
	require.NoError(t, err, "GetUFuturesContinuousKlineData must not error")
	if mockTests {
		exp := []UKline{
			{
				OpenTime:                 types.Time(time.Unix(1744156800, 0)),
				Open:                     76298,
				High:                     83554.9,
				Low:                      74578.5,
				Close:                    82588,
				Volume:                   560649.862,
				CloseTime:                types.Time(time.UnixMilli(1744243199999)),
				QuoteAssetVolume:         44072322884.0975,
				NumberOfTrades:           8799174,
				TakerBuyBaseAssetVolume:  279060.371,
				TakerBuyQuoteAssetVolume: 21941548090.1517,
			},
		}
		assert.Equal(t, exp, result, "GetUFuturesContinuousKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUFuturesContinuousKlineData should return klines")
}

func TestUExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.UExchangeInfo(t.Context())
	require.NoError(t, err, "UExchangeInfo must not error")
	if mockTests {
		exp := &UExchangeInfoResponse{
			Timezone:    "UTC",
			ServerTime:  types.Time(time.UnixMilli(1791278101115)),
			FuturesType: "U_MARGINED",
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
				{
					RateLimitType: "ORDERS",
					Interval:      "SECOND",
					IntervalNum:   10,
					Limit:         300,
				},
			},
			ExchangeFilters: []string{},
			Assets: []UExchangeInfoAsset{
				{
					Asset:             currency.USDT,
					MarginAvailable:   true,
					AutoAssetExchange: -10000,
				},
				{
					Asset:             currency.BTC,
					MarginAvailable:   true,
					AutoAssetExchange: -0.1,
				},
			},
			Symbols: []UFuturesSymbolInfo{
				{
					Symbol:                   "BTCUSDT",
					Pair:                     "BTCUSDT",
					ContractType:             "PERPETUAL",
					DeliveryDate:             types.Time(time.Unix(4133404800, 0)),
					OnboardDate:              types.Time(time.Unix(1567965300, 0)),
					Status:                   "TRADING",
					MaintenanceMarginPercent: 2.5,
					RequiredMarginPercent:    5,
					BaseAsset:                currency.BTC,
					QuoteAsset:               currency.USDT,
					MarginAsset:              currency.USDT,
					PricePrecision:           2,
					QuantityPrecision:        3,
					BaseAssetPrecision:       8,
					QuotePrecision:           8,
					UnderlyingType:           "COIN",
					UnderlyingSubType: []string{
						"PoW",
						"Crypto",
					},
					TriggerProtect:    0.05,
					LiquidationFee:    0.0125,
					MarketTakeBound:   0.05,
					MaxMoveOrderLimit: 10000,
					Filters: []UFuturesSymbolFilter{
						{
							FilterType: "PRICE_FILTER",
							MinPrice:   556.8,
							MaxPrice:   4529764,
							TickSize:   0.1,
						},
						{
							FilterType:  "LOT_SIZE",
							MinQuantity: 0.001,
							MaxQuantity: 1000,
							StepSize:    0.001,
						},
						{
							FilterType:  "MARKET_LOT_SIZE",
							MinQuantity: 0.001,
							MaxQuantity: 120,
							StepSize:    0.001,
						},
						{
							FilterType: "MAX_NUM_ORDERS",
							Limit:      200,
						},
						{
							FilterType: "MIN_NOTIONAL",
							Notional:   50,
						},
						{
							FilterType:        "PERCENT_PRICE",
							MultiplierUp:      1.05,
							MultiplierDown:    0.95,
							MultiplierDecimal: 4,
						},
						{
							FilterType:          "POSITION_RISK_CONTROL",
							PositionControlSide: "BOTH",
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
						"GTD",
					},
					PermissionSets: []string{
						"GRID",
						"COPY",
						"DCA",
						"PSB",
					},
				},
				{
					Symbol:                   "BTCUSDT_261225",
					Pair:                     "BTCUSDT",
					ContractType:             "CURRENT_QUARTER",
					DeliveryDate:             types.Time(time.Unix(1798185600, 0)),
					OnboardDate:              types.Time(time.Unix(1782460800, 0)),
					Status:                   "TRADING",
					MaintenanceMarginPercent: 2.5,
					RequiredMarginPercent:    5,
					BaseAsset:                currency.BTC,
					QuoteAsset:               currency.USDT,
					MarginAsset:              currency.USDT,
					PricePrecision:           1,
					QuantityPrecision:        3,
					BaseAssetPrecision:       8,
					QuotePrecision:           8,
					UnderlyingType:           "COIN",
					UnderlyingSubType:        []string{},
					TriggerProtect:           0.05,
					LiquidationFee:           0.0125,
					MarketTakeBound:          0.05,
					MaxMoveOrderLimit:        10000,
					Filters: []UFuturesSymbolFilter{
						{
							FilterType: "PRICE_FILTER",
							MinPrice:   576.3,
							MaxPrice:   1000000,
							TickSize:   0.1,
						},
						{
							FilterType:  "LOT_SIZE",
							MinQuantity: 0.001,
							MaxQuantity: 500,
							StepSize:    0.001,
						},
						{
							FilterType:  "MARKET_LOT_SIZE",
							MinQuantity: 0.001,
							MaxQuantity: 1,
							StepSize:    0.001,
						},
						{
							FilterType: "MAX_NUM_ORDERS",
							Limit:      200,
						},
						{
							FilterType: "MIN_NOTIONAL",
							Notional:   5,
						},
						{
							FilterType:        "PERCENT_PRICE",
							MultiplierUp:      1.05,
							MultiplierDown:    0.95,
							MultiplierDecimal: 4,
						},
						{
							FilterType:          "POSITION_RISK_CONTROL",
							PositionControlSide: "BOTH",
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
						"GTD",
					},
					PermissionSets: []string{
						"GRID",
						"COPY",
						"DCA",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "UExchangeInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbols, "UExchangeInfo should return symbols")
}

func TestUGetFundingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.UGetFundingHistory(t.Context(), currency.NewBTCUSDT(), endTime, startTime, 1000)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UGetFundingHistory must reject a start time after the end time")

	result, err := e.UGetFundingHistory(t.Context(), currency.NewBTCUSDT(), startTime, endTime, 1000)
	require.NoError(t, err, "UGetFundingHistory must not error")
	if mockTests {
		exp := []FundingRateHistory{
			{
				Symbol:      "BTCUSDT",
				FundingRate: 0.00003134,
				FundingTime: types.Time(time.Unix(1744128000, 0)),
				MarkPrice:   78462.8292963,
				RateType:    "Regular",
			},
			{
				Symbol:      "BTCUSDT",
				FundingRate: 0.00009568,
				FundingTime: types.Time(time.Unix(1744156800, 0)),
				MarkPrice:   76297.9,
				RateType:    "Regular",
			},
		}
		assert.Equal(t, exp, result, "UGetFundingHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UGetFundingHistory should return funding rates")
}

func TestUGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := e.UGetFundingRateInfo(t.Context())
	require.NoError(t, err, "UGetFundingRateInfo must not error")
	if mockTests {
		exp := []FundingRateInfoResponse{
			{
				Symbol:                   "GTCUSDT",
				AdjustedFundingRateCap:   0.02,
				AdjustedFundingRateFloor: -0.02,
				FundingIntervalHours:     8,
				UpdateTime:               types.Time(time.UnixMilli(1758377721362)),
			},
			{
				Symbol:                   "TRXUSDT",
				AdjustedFundingRateCap:   0.004875,
				AdjustedFundingRateFloor: -0.004875,
				FundingIntervalHours:     8,
				Disclaimer:               true,
				UpdateTime:               types.Time(time.UnixMilli(1758377721362)),
			},
		}
		assert.Equal(t, exp, result, "UGetFundingRateInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UGetFundingRateInfo should return funding rate info")
}

func TestGetIndexPriceKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceKlineCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexPriceKlineCandlesticks must reject a nil request")
	arg := &UIndexPriceKlineRequest{Interval: "1d"}
	_, err = e.GetIndexPriceKlineCandlesticks(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIndexPriceKlineCandlesticks must reject an empty pair")
	arg.Pair = currency.NewBTCUSDT()
	arg.Interval = ""
	_, err = e.GetIndexPriceKlineCandlesticks(t.Context(), arg)
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetIndexPriceKlineCandlesticks must reject an empty interval")
	arg.Interval = "1d"
	startTime, endTime := getTime()
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetIndexPriceKlineCandlesticks(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetIndexPriceKlineCandlesticks must reject a start time after the end time")

	arg.StartTime, arg.EndTime, arg.Limit = startTime, endTime, 100
	result, err := e.GetIndexPriceKlineCandlesticks(t.Context(), arg)
	require.NoError(t, err, "GetIndexPriceKlineCandlesticks must not error")
	if mockTests {
		exp := []UPriceKline{
			{
				OpenTime:  types.Time(time.Unix(1744156800, 0)),
				Open:      76322.968,
				High:      83567.47466667,
				Low:       74626.60222222,
				Close:     82611.44511111,
				CloseTime: types.Time(time.UnixMilli(1744243199999)),
			},
		}
		assert.Equal(t, exp, result, "GetIndexPriceKlineCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetIndexPriceKlineCandlesticks should return klines")
}

func TestUKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.UKlineData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UKlineData must reject a nil request")
	_, err = e.UKlineData(t.Context(), &UKlineRequest{Interval: "1d", Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UKlineData must reject an empty symbol")
	_, err = e.UKlineData(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Limit: 5})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "UKlineData must reject an empty interval")
	startTime, endTime := getTime()
	_, err = e.UKlineData(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: "5m", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UKlineData must reject a start time after the end time")

	result, err := e.UKlineData(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: "1d", StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "UKlineData must not error for a limit")
	if mockTests {
		exp := []UKline{
			{
				OpenTime:                 types.Time(time.Unix(1744156800, 0)),
				Open:                     76298,
				High:                     83554.9,
				Low:                      74578.5,
				Close:                    82588,
				Volume:                   560649.862,
				CloseTime:                types.Time(time.UnixMilli(1744243199999)),
				QuoteAssetVolume:         44072322884.0975,
				NumberOfTrades:           8799174,
				TakerBuyBaseAssetVolume:  279060.371,
				TakerBuyQuoteAssetVolume: 21941548090.1517,
			},
		}
		assert.Equal(t, exp, result, "UKlineData should decode every field")
	}
	result, err = e.UKlineData(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: "5m", StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "UKlineData must not error without a limit")
	if mockTests {
		exp := []UKline{
			{
				OpenTime:                 types.Time(time.Unix(1744104000, 0)),
				Open:                     78966.3,
				High:                     79061.6,
				Low:                      78964.1,
				Close:                    79014.4,
				Volume:                   324.734,
				CloseTime:                types.Time(time.UnixMilli(1744104299999)),
				QuoteAssetVolume:         25659455.7241,
				NumberOfTrades:           9663,
				TakerBuyBaseAssetVolume:  167.358,
				TakerBuyQuoteAssetVolume: 13223691.0953,
			},
			{
				OpenTime:                 types.Time(time.Unix(1744104300, 0)),
				Open:                     79014.3,
				High:                     79014.3,
				Low:                      78841.6,
				Close:                    78848.6,
				Volume:                   731.011,
				CloseTime:                types.Time(time.UnixMilli(1744104599999)),
				QuoteAssetVolume:         57669955.9255,
				NumberOfTrades:           13569,
				TakerBuyBaseAssetVolume:  299.749,
				TakerBuyQuoteAssetVolume: 23646539.9461,
			},
		}
		assert.Equal(t, exp, result, "UKlineData should decode every field without a limit")
		return
	}
	assert.NotEmpty(t, result, "UKlineData should return klines")
}

func TestUGlobalLongShortRatio(t *testing.T) {
	t.Parallel()
	testUFuturesDataValidation(t, e.UGlobalLongShortRatio)
	startTime, endTime := getTime()
	result, err := e.UGlobalLongShortRatio(t.Context(), &UFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: "5m", StartTime: startTime, EndTime: endTime, Limit: 3})
	require.NoError(t, err, "UGlobalLongShortRatio must not error")
	if mockTests {
		exp := []ULongShortRatio{
			{
				Symbol:         "BTCUSDT",
				LongShortRatio: 1.0593,
				LongAccount:    0.5144,
				ShortAccount:   0.4856,
				Timestamp:      types.Time(time.Unix(1791282900, 0)),
			},
			{
				Symbol:         "BTCUSDT",
				LongShortRatio: 1.0563,
				LongAccount:    0.5137,
				ShortAccount:   0.4863,
				Timestamp:      types.Time(time.Unix(1791283200, 0)),
			},
		}
		assert.Equal(t, exp, result, "UGlobalLongShortRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UGlobalLongShortRatio should return ratios")
}

// testUFuturesDataValidation checks the request validation the /futures/data statistics endpoints share
func testUFuturesDataValidation[T any](t *testing.T, fn func(context.Context, *UFuturesDataRequest) ([]T, error)) {
	t.Helper()
	_, err := fn(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "must reject a nil request")
	_, err = fn(t.Context(), &UFuturesDataRequest{Period: "5m"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "must reject an empty symbol")
	_, err = fn(t.Context(), &UFuturesDataRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, errPeriodRequired, "must reject an empty period")
	startTime, endTime := getTime()
	_, err = fn(t.Context(), &UFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: "5m", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "must reject a start time after the end time")
}

func TestUGetMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.UGetMarkPrice(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UGetMarkPrice must not error for a symbol")
	if mockTests {
		exp := []UMarkPrice{
			{
				Symbol:               "BTCUSDT",
				MarkPrice:            86163,
				IndexPrice:           86197.90478261,
				EstimatedSettlePrice: 86090.15774908,
				LastFundingRate:      -0.0000349,
				InterestRate:         0.0001,
				NextFundingTime:      types.Time(time.Unix(1791302400, 0)),
				Time:                 types.Time(time.Unix(1791283486, 0)),
			},
		}
		assert.Equal(t, exp, result, "UGetMarkPrice should decode the single symbol object")
	}
	result, err = e.UGetMarkPrice(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "UGetMarkPrice must not error for every symbol")
	if mockTests {
		exp := []UMarkPrice{
			{
				Symbol:               "BTCUSDT",
				MarkPrice:            86163,
				IndexPrice:           86197.90478261,
				EstimatedSettlePrice: 86090.11913599,
				LastFundingRate:      -0.0000349,
				InterestRate:         0.0001,
				NextFundingTime:      types.Time(time.Unix(1791302400, 0)),
				Time:                 types.Time(time.Unix(1791283487, 0)),
			},
			{
				Symbol:               "ETHUSDT",
				MarkPrice:            2716.49264341,
				IndexPrice:           2717.38325581,
				EstimatedSettlePrice: 2714.68348075,
				LastFundingRate:      -0.00001222,
				InterestRate:         0.0001,
				NextFundingTime:      types.Time(time.Unix(1791302400, 0)),
				Time:                 types.Time(time.Unix(1791283487, 0)),
			},
		}
		assert.Equal(t, exp, result, "UGetMarkPrice should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "UGetMarkPrice should return mark prices")
}

func TestGetMarkPriceKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceKlineCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarkPriceKlineCandlesticks must reject a nil request")
	_, err = e.GetMarkPriceKlineCandlesticks(t.Context(), &UKlineRequest{Interval: "1d"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarkPriceKlineCandlesticks must reject an empty symbol")
	_, err = e.GetMarkPriceKlineCandlesticks(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetMarkPriceKlineCandlesticks must reject an empty interval")
	startTime, endTime := getTime()
	_, err = e.GetMarkPriceKlineCandlesticks(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: "1d", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarkPriceKlineCandlesticks must reject a start time after the end time")

	result, err := e.GetMarkPriceKlineCandlesticks(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: "1d", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetMarkPriceKlineCandlesticks must not error")
	if mockTests {
		exp := []UPriceKline{
			{
				OpenTime:  types.Time(time.Unix(1744156800, 0)),
				Open:      76297.9,
				High:      83553.4,
				Low:       74588.1,
				Close:     82587.9,
				CloseTime: types.Time(time.UnixMilli(1744243199999)),
			},
		}
		assert.Equal(t, exp, result, "GetMarkPriceKlineCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetMarkPriceKlineCandlesticks should return klines")
}

func TestGetMultiAssetModeAssetIndex(t *testing.T) {
	t.Parallel()
	result, err := e.GetMultiAssetModeAssetIndex(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "GetMultiAssetModeAssetIndex must not error for a symbol")
	if mockTests {
		exp := []AssetIndex{
			{
				Symbol:                "BTCUSD",
				Time:                  types.Time(time.UnixMilli(1791283500001)),
				Index:                 86168.37311999,
				BidBuffer:             0.05,
				AskBuffer:             0.05,
				BidRate:               81859.95446399,
				AskRate:               90476.79177599,
				AutoExchangeBidBuffer: 0.025,
				AutoExchangeAskBuffer: 0.025,
				AutoExchangeBidRate:   84014.16379199,
				AutoExchangeAskRate:   88322.58244799,
			},
		}
		assert.Equal(t, exp, result, "GetMultiAssetModeAssetIndex should decode the single symbol object")
	}
	result, err = e.GetMultiAssetModeAssetIndex(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetMultiAssetModeAssetIndex must not error for every symbol")
	if mockTests {
		exp := []AssetIndex{
			{
				Symbol:                "BTCUSD",
				Time:                  types.Time(time.UnixMilli(1791283500001)),
				Index:                 86168.37311999,
				BidBuffer:             0.05,
				AskBuffer:             0.05,
				BidRate:               81859.95446399,
				AskRate:               90476.79177599,
				AutoExchangeBidBuffer: 0.025,
				AutoExchangeAskBuffer: 0.025,
				AutoExchangeBidRate:   84014.16379199,
				AutoExchangeAskRate:   88322.58244799,
			},
			{
				Symbol:              "SUIUSD",
				Time:                types.Time(time.UnixMilli(1791283500001)),
				Index:               1.20147365,
				BidRate:             1.20147365,
				AskRate:             1.20147365,
				AutoExchangeBidRate: 1.20147365,
				AutoExchangeAskRate: 1.20147365,
			},
		}
		assert.Equal(t, exp, result, "GetMultiAssetModeAssetIndex should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "GetMultiAssetModeAssetIndex should return asset indexes")
}

func TestUFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.UFuturesHistoricalTrades(t.Context(), currency.EMPTYPAIR, 0, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UFuturesHistoricalTrades must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, fromID := range []uint64{0, 8149534980} {
		result, err := e.UFuturesHistoricalTrades(t.Context(), currency.NewBTCUSDT(), fromID, 5)
		require.NoErrorf(t, err, "UFuturesHistoricalTrades must not error from trade ID %d", fromID)
		if mockTests {
			exp := []UPublicTradesData{
				{
					ID:            8149534980,
					Price:         86065.5,
					Quantity:      0.001,
					QuoteQuantity: 86.06,
					Time:          types.Time(time.UnixMilli(1791283323016)),
					IsBuyerMaker:  true,
				},
				{
					ID:            8149534981,
					Price:         86065.4,
					Quantity:      0.059,
					QuoteQuantity: 5077.86,
					Time:          types.Time(time.UnixMilli(1791283323999)),
					IsRPITrade:    true,
				},
			}
			assert.Equal(t, exp, result, "UFuturesHistoricalTrades should decode every field")
			continue
		}
		assert.NotEmptyf(t, result, "UFuturesHistoricalTrades should return trades from trade ID %d", fromID)
	}
}

func TestUOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.UOpenInterest(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UOpenInterest must reject an empty symbol")

	result, err := e.UOpenInterest(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UOpenInterest must not error")
	if mockTests {
		exp := &UOpenInterestResponse{
			OpenInterest: 94598.71,
			Symbol:       "BTCUSDT",
			Time:         types.Time(time.UnixMilli(1791283485477)),
		}
		assert.Equal(t, exp, result, "UOpenInterest should decode every field")
		return
	}
	assert.Positive(t, result.OpenInterest.Float64(), "UOpenInterest should return open interest")
}

func TestUOpenInterestStats(t *testing.T) {
	t.Parallel()
	testUFuturesDataValidation(t, e.UOpenInterestStats)
	startTime, endTime := getTime()
	result, err := e.UOpenInterestStats(t.Context(), &UFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: "5m", StartTime: startTime, EndTime: endTime, Limit: 1})
	require.NoError(t, err, "UOpenInterestStats must not error")
	if mockTests {
		exp := []UOpenInterestStats{
			{
				Symbol:               "BTCUSDT",
				SumOpenInterest:      94576.939,
				SumOpenInterestValue: 8134776100.048762,
				CMCCirculatingSupply: 20094037,
				Timestamp:            types.Time(time.Unix(1791282900, 0)),
			},
		}
		assert.Equal(t, exp, result, "UOpenInterestStats should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UOpenInterestStats should return statistics")
}

func TestUFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UFuturesOrderbook(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UFuturesOrderbook must reject an empty symbol")

	result, err := e.UFuturesOrderbook(t.Context(), currency.NewBTCUSDT(), 1000)
	require.NoError(t, err, "UFuturesOrderbook must not error")
	if mockTests {
		exp := &UOrderBookResponse{
			LastUpdateID:      11747025852863,
			MessageOutputTime: types.Time(time.UnixMilli(1791283481760)),
			TransactionTime:   types.Time(time.UnixMilli(1791283481757)),
			Bids: []orderbook.Level{
				{
					Amount: 13.582,
					Price:  86163,
				},
				{
					Amount: 0.021,
					Price:  86162.9,
				},
				{
					Amount: 0.034,
					Price:  86162.8,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 5.38,
					Price:  86163.1,
				},
				{
					Amount: 0.957,
					Price:  86163.2,
				},
				{
					Amount: 0.041,
					Price:  86163.3,
				},
			},
		}
		assert.Equal(t, exp, result, "UFuturesOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "UFuturesOrderbook should return bids")
}

func TestGetPremiumIndexKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexKlineCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPremiumIndexKlineCandlesticks must reject a nil request")
	_, err = e.GetPremiumIndexKlineCandlesticks(t.Context(), &UKlineRequest{Interval: "1d"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPremiumIndexKlineCandlesticks must reject an empty symbol")
	_, err = e.GetPremiumIndexKlineCandlesticks(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetPremiumIndexKlineCandlesticks must reject an empty interval")
	startTime, endTime := getTime()
	_, err = e.GetPremiumIndexKlineCandlesticks(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: "1d", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPremiumIndexKlineCandlesticks must reject a start time after the end time")

	result, err := e.GetPremiumIndexKlineCandlesticks(t.Context(), &UKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: "1d", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetPremiumIndexKlineCandlesticks must not error")
	if mockTests {
		exp := []UPriceKline{
			{
				OpenTime:  types.Time(time.Unix(1744156800, 0)),
				Open:      -0.00032714,
				High:      0.00180625,
				Low:       -0.0025365,
				Close:     -0.00028604,
				CloseTime: types.Time(time.UnixMilli(1744243199999)),
			},
		}
		assert.Equal(t, exp, result, "GetPremiumIndexKlineCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPremiumIndexKlineCandlesticks should return klines")
}

func TestGetQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetQuarterlyContractSettlementPrice must reject an empty pair")

	result, err := e.GetQuarterlyContractSettlementPrice(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetQuarterlyContractSettlementPrice must not error")
	if mockTests {
		exp := []SettlementPrice{
			{
				DeliveryTime:  types.Time(time.Unix(1782432000, 0)),
				DeliveryPrice: 60608.3,
			},
			{
				DeliveryTime:  types.Time(time.Unix(1774569600, 0)),
				DeliveryPrice: 68656.7,
			},
		}
		assert.Equal(t, exp, result, "GetQuarterlyContractSettlementPrice should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetQuarterlyContractSettlementPrice should return settlement prices")
}

func TestGetIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceConstituents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIndexPriceConstituents must reject an empty symbol")

	result, err := e.GetIndexPriceConstituents(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetIndexPriceConstituents must not error")
	if mockTests {
		exp := &UIndexPriceConstituentsResponse{
			Symbol: "BTCUSDT",
			Time:   types.Time(time.UnixMilli(1791283487198)),
			Constituents: []UIndexPriceConstituent{
				{
					Exchange: "binance",
					Symbol:   "BTCUSDT",
					Price:    86199.99,
					Weight:   0.43478261,
				},
				{
					Exchange: "okex",
					Symbol:   "BTC-USDT",
					Price:    86207.6,
					Weight:   0.13043478,
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexPriceConstituents should decode every field")
		return
	}
	assert.NotEmpty(t, result.Constituents, "GetIndexPriceConstituents should return constituents")
}

func TestURecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.URecentTrades(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "URecentTrades must reject an empty symbol")

	result, err := e.URecentTrades(t.Context(), currency.NewBTCUSDT(), 1000)
	require.NoError(t, err, "URecentTrades must not error")
	if mockTests {
		exp := []UPublicTradesData{
			{
				ID:            8149534980,
				Price:         86065.5,
				Quantity:      0.001,
				QuoteQuantity: 86.06,
				Time:          types.Time(time.UnixMilli(1791283323016)),
				IsBuyerMaker:  true,
			},
			{
				ID:            8149534982,
				Price:         86065.6,
				Quantity:      0.001,
				QuoteQuantity: 86.06,
				Time:          types.Time(time.UnixMilli(1791283324072)),
				IsRPITrade:    true,
			},
		}
		assert.Equal(t, exp, result, "URecentTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "URecentTrades should return trades")
}

func TestGetURPIOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetURPIOrderbook(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetURPIOrderbook must reject an empty symbol")

	result, err := e.GetURPIOrderbook(t.Context(), currency.NewBTCUSDT(), 1000)
	require.NoError(t, err, "GetURPIOrderbook must not error")
	if mockTests {
		exp := &UOrderBookResponse{
			LastUpdateID:      11747025896572,
			MessageOutputTime: types.Time(time.UnixMilli(1791283482433)),
			TransactionTime:   types.Time(time.UnixMilli(1791283482430)),
			Bids: []orderbook.Level{
				{
					Amount: 13.315,
					Price:  86163,
				},
				{
					Amount: 0.021,
					Price:  86162.9,
				},
				{
					Amount: 0.034,
					Price:  86162.8,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 6.399,
					Price:  86163.1,
				},
				{
					Amount: 0.957,
					Price:  86163.2,
				},
				{
					Amount: 0.013,
					Price:  86163.3,
				},
			},
		}
		assert.Equal(t, exp, result, "GetURPIOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetURPIOrderbook should return bids")
}

func TestUSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := e.USymbolOrderbookTicker(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USymbolOrderbookTicker must not error for a symbol")
	if mockTests {
		exp := []USymbolOrderbookTicker{
			{
				Symbol:       "BTCUSDT",
				BidPrice:     86163,
				BidQuantity:  8.219,
				AskPrice:     86163.1,
				AskQuantity:  6.55,
				Time:         types.Time(time.UnixMilli(1791283492295)),
				LastUpdateID: 11747026548280,
			},
		}
		assert.Equal(t, exp, result, "USymbolOrderbookTicker should decode the single symbol object")
	}
	result, err = e.USymbolOrderbookTicker(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USymbolOrderbookTicker must not error for every symbol")
	if mockTests {
		exp := []USymbolOrderbookTicker{
			{
				Symbol:       "BTCUSDT",
				BidPrice:     86163,
				BidQuantity:  10.241,
				AskPrice:     86163.1,
				AskQuantity:  6.343,
				Time:         types.Time(time.UnixMilli(1791283492916)),
				LastUpdateID: 11747026591165,
			},
			{
				Symbol:       "ETHUSDT",
				BidPrice:     2716.46,
				BidQuantity:  184.555,
				AskPrice:     2716.47,
				AskQuantity:  257.508,
				Time:         types.Time(time.UnixMilli(1791283493059)),
				LastUpdateID: 11747026600753,
			},
		}
		assert.Equal(t, exp, result, "USymbolOrderbookTicker should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USymbolOrderbookTicker should return tickers")
}

func TestUSymbolPriceTickerV2(t *testing.T) {
	t.Parallel()
	result, err := e.USymbolPriceTickerV2(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USymbolPriceTickerV2 must not error for a symbol")
	if mockTests {
		exp := []USymbolPriceTicker{
			{
				Symbol: "BTCUSDT",
				Price:  86163,
				Time:   types.Time(time.UnixMilli(1791283491172)),
			},
		}
		assert.Equal(t, exp, result, "USymbolPriceTickerV2 should decode the single symbol object")
	}
	result, err = e.USymbolPriceTickerV2(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USymbolPriceTickerV2 must not error for every symbol")
	if mockTests {
		exp := []USymbolPriceTicker{
			{
				Symbol: "HUSDT",
				Price:  0.06884,
				Time:   types.Time(time.UnixMilli(1791283460503)),
			},
			{
				Symbol: "VTHOUSDT",
				Price:  0.0007297,
				Time:   types.Time(time.UnixMilli(1791283491196)),
			},
		}
		assert.Equal(t, exp, result, "USymbolPriceTickerV2 should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USymbolPriceTickerV2 should return prices")
}

func TestUTakerBuySellVol(t *testing.T) {
	t.Parallel()
	testUFuturesDataValidation(t, e.UTakerBuySellVol)
	startTime, endTime := getTime()
	result, err := e.UTakerBuySellVol(t.Context(), &UFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: "5m", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "UTakerBuySellVol must not error")
	if mockTests {
		exp := []UTakerVolumeData{
			{
				BuySellRatio: 0.5069,
				BuyVolume:    63.747,
				SellVolume:   125.768,
				Timestamp:    types.Time(time.Unix(1791282600, 0)),
			},
			{
				BuySellRatio: 0.742,
				BuyVolume:    81.352,
				SellVolume:   109.638,
				Timestamp:    types.Time(time.Unix(1791282900, 0)),
			},
		}
		assert.Equal(t, exp, result, "UTakerBuySellVol should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UTakerBuySellVol should return volumes")
}

func TestU24HourTickerPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := e.U24HourTickerPriceChangeStats(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "U24HourTickerPriceChangeStats must not error for a symbol")
	if mockTests {
		exp := []U24HourPriceChangeStats{
			{
				Symbol:               "BTCUSDT",
				PriceChange:          169.2,
				PriceChangePercent:   0.197,
				WeightedAveragePrice: 85686.16,
				LastPrice:            86163,
				LastQuantity:         0.013,
				OpenPrice:            85993.8,
				HighPrice:            86683.9,
				LowPrice:             84910,
				Volume:               119574.386,
				QuoteVolume:          10245870317.88,
				OpenTime:             types.Time(time.Unix(1791197040, 0)),
				CloseTime:            types.Time(time.UnixMilli(1791283483051)),
				FirstID:              8146769003,
				LastID:               8149539182,
				Count:                2758000,
			},
		}
		assert.Equal(t, exp, result, "U24HourTickerPriceChangeStats should decode the single symbol object")
	}
	result, err = e.U24HourTickerPriceChangeStats(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "U24HourTickerPriceChangeStats must not error for every symbol")
	if mockTests {
		exp := []U24HourPriceChangeStats{
			{
				Symbol:               "BTCUSDT",
				PriceChange:          169.3,
				PriceChangePercent:   0.197,
				WeightedAveragePrice: 85686.17,
				LastPrice:            86163.1,
				LastQuantity:         0.11,
				OpenPrice:            85993.8,
				HighPrice:            86683.9,
				LowPrice:             84910,
				Volume:               119575.491,
				QuoteVolume:          10245965528.02,
				OpenTime:             types.Time(time.Unix(1791197040, 0)),
				CloseTime:            types.Time(time.UnixMilli(1791283486974)),
				FirstID:              8146769003,
				LastID:               8149539204,
				Count:                2758022,
			},
			{
				Symbol:               "ETHUSDT",
				PriceChange:          2.72,
				PriceChangePercent:   0.1,
				WeightedAveragePrice: 2704.81,
				LastPrice:            2716.47,
				LastQuantity:         0.156,
				OpenPrice:            2713.75,
				HighPrice:            2729.89,
				LowPrice:             2676.97,
				Volume:               2409044.784,
				QuoteVolume:          6516006786.4,
				OpenTime:             types.Time(time.Unix(1791197040, 0)),
				CloseTime:            types.Time(time.UnixMilli(1791283487484)),
				FirstID:              8820450476,
				LastID:               8823224129,
				Count:                2763052,
			},
		}
		assert.Equal(t, exp, result, "U24HourTickerPriceChangeStats should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "U24HourTickerPriceChangeStats should return statistics")
}

func TestUTopAccountsLongShortRatio(t *testing.T) {
	t.Parallel()
	testUFuturesDataValidation(t, e.UTopAccountsLongShortRatio)
	startTime, endTime := getTime()
	result, err := e.UTopAccountsLongShortRatio(t.Context(), &UFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: "5m", StartTime: startTime, EndTime: endTime, Limit: 2})
	require.NoError(t, err, "UTopAccountsLongShortRatio must not error")
	if mockTests {
		exp := []ULongShortRatio{
			{
				Symbol:         "BTCUSDT",
				LongShortRatio: 1.124,
				LongAccount:    0.5292,
				ShortAccount:   0.4708,
				Timestamp:      types.Time(time.Unix(1791282900, 0)),
			},
			{
				Symbol:         "BTCUSDT",
				LongShortRatio: 1.1222,
				LongAccount:    0.5288,
				ShortAccount:   0.4712,
				Timestamp:      types.Time(time.Unix(1791283200, 0)),
			},
		}
		assert.Equal(t, exp, result, "UTopAccountsLongShortRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UTopAccountsLongShortRatio should return ratios")
}

func TestUTopPositionsLongShortRatio(t *testing.T) {
	t.Parallel()
	testUFuturesDataValidation(t, e.UTopPositionsLongShortRatio)
	startTime, endTime := getTime()
	result, err := e.UTopPositionsLongShortRatio(t.Context(), &UFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: "5m", StartTime: startTime, EndTime: endTime, Limit: 3})
	require.NoError(t, err, "UTopPositionsLongShortRatio must not error")
	if mockTests {
		exp := []ULongShortRatio{
			{
				Symbol:         "BTCUSDT",
				LongShortRatio: 1.6782,
				LongAccount:    0.6266,
				ShortAccount:   0.3734,
				Timestamp:      types.Time(time.Unix(1791282900, 0)),
			},
			{
				Symbol:         "BTCUSDT",
				LongShortRatio: 1.6768,
				LongAccount:    0.6264,
				ShortAccount:   0.3736,
				Timestamp:      types.Time(time.Unix(1791283200, 0)),
			},
		}
		assert.Equal(t, exp, result, "UTopPositionsLongShortRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "UTopPositionsLongShortRatio should return ratios")
}

func TestSetAssetsMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.SetAssetsMode(t.Context(), true), "SetAssetsMode should not error")
}

func TestChangePositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.ChangePositionMode(t.Context(), false), "ChangePositionMode should not error")
}

func TestUAccountTradesHistory(t *testing.T) {
	t.Parallel()
	_, err := e.UAccountTradesHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UAccountTradesHistory must reject a nil request")
	_, err = e.UAccountTradesHistory(t.Context(), &UAccountTradeListRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UAccountTradesHistory must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.UAccountTradesHistory(t.Context(), &UAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), FromID: 698759, StartTime: startTime})
	require.ErrorIs(t, err, errFromIDWithTimeWindow, "UAccountTradesHistory must reject a from ID with a start time")
	_, err = e.UAccountTradesHistory(t.Context(), &UAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UAccountTradesHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UAccountTradesHistory(t.Context(), &UAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "UAccountTradesHistory must not error")
	if mockTests {
		exp := []UAccountTradeHistory{
			{
				Buyer:           true,
				Commission:      -0.0781901,
				CommissionAsset: currency.USDT,
				ID:              698759,
				Maker:           true,
				OrderID:         25851813,
				Price:           7819.01,
				Quantity:        0.002,
				QuoteQuantity:   15.63802,
				BaseQuantity:    0.00025578,
				MarginAsset:     currency.USDT,
				RealizedPNL:     -0.91539999,
				Side:            "BUY",
				PositionSide:    "SHORT",
				Symbol:          "BTCUSDT",
				Pair:            "BTCUSDT",
				Time:            types.Time(time.Unix(1744118400, 0)),
			},
		}
		assert.Equal(t, exp, result, "UAccountTradesHistory should decode every field")
	}
}

func TestUAllAccountOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UAllAccountOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UAllAccountOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.UAllAccountOrders(t.Context(), &UAllOrdersRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UAllAccountOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []*UAllOrdersRequest{
		{StartTime: startTime, EndTime: endTime},
		{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 5},
	} {
		result, err := e.UAllAccountOrders(t.Context(), tc)
		require.NoErrorf(t, err, "UAllAccountOrders must not error for symbol %q", tc.Symbol)
		if mockTests {
			exp := []UFuturesOrderData{
				{
					ClientOrderID:           "testOrder",
					ExecutedQuantity:        0.001,
					OrderID:                 22542179,
					OriginalQuantity:        0.002,
					Price:                   80000,
					ReduceOnly:              true,
					Side:                    "BUY",
					PositionSide:            "LONG",
					Status:                  "PARTIALLY_FILLED",
					StopPrice:               79000,
					ClosePosition:           true,
					Symbol:                  "BTCUSDT",
					TimeInForce:             "GTD",
					Type:                    "LIMIT",
					OriginalType:            "TAKE_PROFIT",
					UpdateTime:              types.Time(time.Unix(1744118400, 0)),
					WorkingType:             "MARK_PRICE",
					PriceProtect:            true,
					PriceMatch:              "QUEUE",
					SelfTradePreventionMode: "EXPIRE_MAKER",
					GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
					AveragePrice:            80000,
					CumulativeQuote:         80,
					Time:                    types.Time(time.Unix(1744118000, 0)),
					ActivatePrice:           79500,
					PriceRate:               0.3,
					CumulativeBase:          0.001,
					Pair:                    "BTCUSDT",
				},
			}
			assert.Equal(t, exp, result, "UAllAccountOrders should decode every field")
		}
	}
}

func TestUAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UAutoCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR, 30*time.Second)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UAutoCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UAutoCancelAllOpenOrders(t.Context(), currency.NewBTCUSDT(), 30*time.Second)
	require.NoError(t, err, "UAutoCancelAllOpenOrders must not error")
	if mockTests {
		exp := &UAutoCancelAllOpenOrdersResponse{
			Symbol:        "BTCUSDT",
			CountdownTime: 30000,
		}
		assert.Equal(t, exp, result, "UAutoCancelAllOpenOrders should decode every field")
	}
}

func TestUQueryAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UQueryAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UQueryAlgoOrder must reject a missing algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UQueryAlgoOrder(t.Context(), 2146760, "")
	require.NoError(t, err, "UQueryAlgoOrder must not error")
	if mockTests {
		exp := &UAlgoOrderResponse{
			AlgoID:                  2146760,
			ClientAlgoID:            "6B2I9XVcJpCjqPAJ4YoFX7",
			AlgoType:                "CONDITIONAL",
			OrderType:               "TAKE_PROFIT",
			Symbol:                  "BTCUSDT",
			Side:                    "SELL",
			PositionSide:            "LONG",
			TimeInForce:             "GTC",
			Quantity:                0.01,
			AlgoStatus:              "FINISHED",
			TriggerPrice:            90000,
			Price:                   90000,
			IcebergQuantity:         "null",
			SelfTradePreventionMode: "EXPIRE_MAKER",
			WorkingType:             "MARK_PRICE",
			PriceMatch:              "QUEUE",
			ClosePosition:           true,
			PriceProtect:            true,
			ReduceOnly:              true,
			CreateTime:              types.Time(time.UnixMilli(1750485492076)),
			UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
			TriggerTime:             types.Time(time.UnixMilli(1750514545091)),
			GoodTillDate:            types.Time(time.Unix(1750550400, 0)),
			ActualOrderID:           "20072994037",
			ActualPrice:             90000,
			ActualType:              "LIMIT",
			ActualQuantity:          0.01,
			TakeProfitOrderType:     "LIMIT",
		}
		assert.Equal(t, exp, result, "UQueryAlgoOrder should decode every field")
	}
}

func TestUNewAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UNewAlgoOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UNewAlgoOrder must reject a nil request")
	arg := &UNewAlgoOrderRequest{}
	_, err = e.UNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errAlgoTypeRequired, "UNewAlgoOrder must reject an empty algo type")
	arg.AlgoType = "CONDITIONAL"
	_, err = e.UNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UNewAlgoOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.UNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "UNewAlgoOrder must reject an empty side")
	arg.Side = order.Sell.String()
	_, err = e.UNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "UNewAlgoOrder must reject an empty order type")
	arg.OrderType = "STOP"
	arg.Price = 80000
	arg.PriceMatch = "QUEUE"
	_, err = e.UNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "UNewAlgoOrder must reject a price match with a price")
	arg.PriceMatch = ""
	arg.ClosePosition = true
	arg.Quantity = 0.01
	_, err = e.UNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionWithQuantity, "UNewAlgoOrder must reject a close position with a quantity")
	arg.Quantity = 0
	arg.ReduceOnly = true
	_, err = e.UNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionWithQuantity, "UNewAlgoOrder must reject a close position with reduce only")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []*UNewAlgoOrderRequest{
		{
			AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", PositionSide: "LONG", OrderType: "TAKE_PROFIT",
			TimeInForce: "GTD", Quantity: 0.01, TriggerPrice: 90000, WorkingType: "MARK_PRICE", PriceMatch: "QUEUE",
			PriceProtect: true, ClientAlgoID: "6B2I9XVcJpCjqPAJ4YoFX7", NewOrderRespType: "RESULT",
			SelfTradePreventionMode: "EXPIRE_MAKER", GoodTillDate: time.UnixMilli(1750550400000),
		},
		{
			AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "TRAILING_STOP_MARKET",
			Quantity: 0.01, ReduceOnly: true, ActivatePrice: 89500, CallbackRate: 0.5,
		},
		{AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "STOP_MARKET", TriggerPrice: 80000, ClosePosition: true},
		{AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "STOP", Quantity: 0.01, Price: 80000, TriggerPrice: 80500},
	} {
		result, err := e.UNewAlgoOrder(t.Context(), tc)
		require.NoErrorf(t, err, "UNewAlgoOrder must not error for order type %s", tc.OrderType)
		if mockTests {
			exp := &UNewAlgoOrderResponse{
				AlgoID:                  2146760,
				ClientAlgoID:            "6B2I9XVcJpCjqPAJ4YoFX7",
				AlgoType:                "CONDITIONAL",
				OrderType:               "TAKE_PROFIT",
				Symbol:                  "BTCUSDT",
				Side:                    "SELL",
				PositionSide:            "LONG",
				TimeInForce:             "GTD",
				Quantity:                0.01,
				AlgoStatus:              "NEW",
				TriggerPrice:            90000,
				Price:                   90000,
				IcebergQuantity:         "null",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				WorkingType:             "MARK_PRICE",
				PriceMatch:              "QUEUE",
				ClosePosition:           true,
				PriceProtect:            true,
				ReduceOnly:              true,
				CreateTime:              types.Time(time.UnixMilli(1750485492076)),
				UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
				TriggerTime:             types.Time(time.UnixMilli(1750514545091)),
				GoodTillDate:            types.Time(time.Unix(1750550400, 0)),
				ActivatePrice:           89500,
				CallbackRate:            0.5,
			}
			assert.Equal(t, exp, result, "UNewAlgoOrder should decode every field")
		}
	}
}

func TestUCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UCancelAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UCancelAlgoOrder must reject a missing algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UCancelAlgoOrder(t.Context(), 2146760, "")
	require.NoError(t, err, "UCancelAlgoOrder must not error")
	if mockTests {
		exp := &UCancelAlgoOrderResponse{
			AlgoID:       2146760,
			ClientAlgoID: "6B2I9XVcJpCjqPAJ4YoFX7",
			Code:         200,
			Message:      "success",
		}
		assert.Equal(t, exp, result, "UCancelAlgoOrder should decode every field")
	}
}

func TestUCancelAllAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	err := e.UCancelAllAlgoOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UCancelAllAlgoOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.UCancelAllAlgoOpenOrders(t.Context(), currency.NewBTCUSDT()), "UCancelAllAlgoOpenOrders should not error")
}

func TestUCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	err := e.UCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.UCancelAllOpenOrders(t.Context(), currency.NewBTCUSDT()), "UCancelAllOpenOrders should not error")
}

func TestUModifyMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UModifyMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "UModifyMultipleOrders must reject an empty batch")
	_, err = e.UModifyMultipleOrders(t.Context(), []UModifyOrderRequest{{}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UModifyMultipleOrders must validate every order")
	_, err = e.UModifyMultipleOrders(t.Context(), []UModifyOrderRequest{{OrderID: 1, ReduceOnly: true}})
	require.ErrorIs(t, err, errBatchModifyReduceOnly, "UModifyMultipleOrders must reject reduce only")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UModifyMultipleOrders(t.Context(), []UModifyOrderRequest{
		{OrderID: 20072994037, Symbol: currency.NewBTCUSDT(), Side: "BUY", Quantity: 1, Price: 30005, ModifyID: 1},
		{OrigClientOrderID: "myOrder1", Symbol: currency.NewBTCUSDT(), Side: "SELL", Quantity: 0.5, PriceMatch: "QUEUE"},
	})
	require.NoError(t, err, "UModifyMultipleOrders must not error")
	if mockTests {
		exp := []UModifyBatchOrderResponse{
			{
				ClientOrderID:           "LJ9R4QZDihCaS8UAOOLpgW",
				ExecutedQuantity:        0.5,
				OrderID:                 20072994037,
				OriginalQuantity:        1,
				Price:                   30005,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "LONG",
				Status:                  "PARTIALLY_FILLED",
				StopPrice:               79000,
				ClosePosition:           true,
				Symbol:                  "BTCUSDT",
				TimeInForce:             "GTD",
				Type:                    "LIMIT",
				OriginalType:            "TAKE_PROFIT",
				UpdateTime:              types.Time(time.Unix(1744118400, 0)),
				WorkingType:             "MARK_PRICE",
				PriceProtect:            true,
				PriceMatch:              "QUEUE",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
				CumulativeQuantity:      0.5,
				Pair:                    "BTCUSDT",
				ModifyID:                1,
			},
			{
				Code:    -2022,
				Message: "ReduceOnly Order is rejected.",
			},
		}
		assert.Equal(t, exp, result, "UModifyMultipleOrders should decode every field")
	}
}

func TestUPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UPlaceBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "UPlaceBatchOrders must reject an empty batch")
	_, err = e.UPlaceBatchOrders(t.Context(), []UFuturesNewOrderRequest{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UPlaceBatchOrders must validate every order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UPlaceBatchOrders(t.Context(), []UFuturesNewOrderRequest{
		{Symbol: currency.NewBTCUSDT(), Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 0.002, Price: 80000, NewClientOrderID: "testOrder"},
		{Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "MARKET", Quantity: 0.002, ReduceOnly: true},
	})
	require.NoError(t, err, "UPlaceBatchOrders must not error")
	if mockTests {
		exp := []UPlaceBatchOrderResponse{
			{
				ClientOrderID:           "testOrder",
				ExecutedQuantity:        0.001,
				OrderID:                 22542179,
				OriginalQuantity:        0.002,
				Price:                   80000,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "LONG",
				Status:                  "PARTIALLY_FILLED",
				StopPrice:               79000,
				ClosePosition:           true,
				Symbol:                  "BTCUSDT",
				TimeInForce:             "GTD",
				Type:                    "LIMIT",
				OriginalType:            "TAKE_PROFIT",
				UpdateTime:              types.Time(time.Unix(1744118400, 0)),
				WorkingType:             "MARK_PRICE",
				PriceProtect:            true,
				PriceMatch:              "QUEUE",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
				CumulativeQuantity:      0.001,
			},
			{
				Code:    -2022,
				Message: "ReduceOnly Order is rejected.",
			},
		}
		assert.Equal(t, exp, result, "UPlaceBatchOrders should decode every field")
	}
}

func TestUCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UCancelBatchOrders(t.Context(), currency.EMPTYPAIR, []uint64{283194212}, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UCancelBatchOrders must reject an empty symbol")
	_, err = e.UCancelBatchOrders(t.Context(), currency.NewBTCUSDT(), nil, nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UCancelBatchOrders must reject an empty ID list")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UCancelBatchOrders(t.Context(), currency.NewBTCUSDT(), []uint64{283194212}, []string{"myOrder2"})
	require.NoError(t, err, "UCancelBatchOrders must not error")
	if mockTests {
		exp := []UCancelBatchOrderResponse{
			{
				ClientOrderID:           "myOrder1",
				ExecutedQuantity:        0.001,
				OrderID:                 283194212,
				OriginalQuantity:        0.002,
				Price:                   80000,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "LONG",
				Status:                  "CANCELED",
				StopPrice:               79000,
				ClosePosition:           true,
				Symbol:                  "BTCUSDT",
				TimeInForce:             "GTD",
				Type:                    "LIMIT",
				OriginalType:            "TAKE_PROFIT",
				UpdateTime:              types.Time(time.Unix(1744118400, 0)),
				WorkingType:             "MARK_PRICE",
				PriceProtect:            true,
				PriceMatch:              "QUEUE",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
				CumulativeQuantity:      0.001,
				ActivatePrice:           79500,
				PriceRate:               0.3,
			},
			{
				Code:    -2011,
				Message: "Unknown order sent.",
			},
		}
		assert.Equal(t, exp, result, "UCancelBatchOrders should decode every field")
	}
}

func TestUGetOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.UGetOrderData(t.Context(), currency.EMPTYPAIR, 123, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UGetOrderData must reject an empty symbol")
	_, err = e.UGetOrderData(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UGetOrderData must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UGetOrderData(t.Context(), currency.NewBTCUSDT(), 123, "")
	require.NoError(t, err, "UGetOrderData must not error")
	if mockTests {
		exp := &UOrderResponse{
			ClientOrderID:           "testOrder",
			ExecutedQuantity:        0.001,
			OrderID:                 123,
			OriginalQuantity:        0.002,
			Price:                   80000,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "LONG",
			Status:                  "PARTIALLY_FILLED",
			StopPrice:               79000,
			ClosePosition:           true,
			Symbol:                  "BTCUSDT",
			TimeInForce:             "GTD",
			Type:                    "LIMIT",
			OriginalType:            "TAKE_PROFIT",
			UpdateTime:              types.Time(time.Unix(1744118400, 0)),
			WorkingType:             "MARK_PRICE",
			PriceProtect:            true,
			PriceMatch:              "QUEUE",
			SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
			AveragePrice:            80000,
			CumulativeQuote:         80,
			Time:                    types.Time(time.Unix(1744118000, 0)),
			ActivatePrice:           79500,
			PriceRate:               0.3,
		}
		assert.Equal(t, exp, result, "UGetOrderData should decode every field")
	}
}

func TestUModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UModifyOrder must reject a nil request")
	arg := &UModifyOrderRequest{}
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UModifyOrder must reject a missing order ID")
	arg.OrderID = 20072994037
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UModifyOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "UModifyOrder must reject an empty side")
	arg.Side = order.Buy.String()
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "UModifyOrder must reject an empty quantity")
	arg.Quantity = 1
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "UModifyOrder must reject an empty price without a price match")
	arg.Price = 30005
	arg.PriceMatch = "QUEUE"
	_, err = e.UModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "UModifyOrder must reject a price match with a price")
	arg.PriceMatch = ""

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.ModifyID = 1
	arg.ReduceOnly = true
	result, err := e.UModifyOrder(t.Context(), arg)
	require.NoError(t, err, "UModifyOrder must not error")
	if mockTests {
		exp := &UModifyOrderResponse{
			ClientOrderID:           "LJ9R4QZDihCaS8UAOOLpgW",
			ExecutedQuantity:        0.5,
			OrderID:                 20072994037,
			OriginalQuantity:        1,
			Price:                   30005,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "LONG",
			Status:                  "PARTIALLY_FILLED",
			StopPrice:               79000,
			ClosePosition:           true,
			Symbol:                  "BTCUSDT",
			TimeInForce:             "GTD",
			Type:                    "LIMIT",
			OriginalType:            "TAKE_PROFIT",
			UpdateTime:              types.Time(time.Unix(1744118400, 0)),
			WorkingType:             "MARK_PRICE",
			PriceProtect:            true,
			PriceMatch:              "QUEUE",
			SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
			CumulativeQuantity:      0.5,
			Pair:                    "BTCUSDT",
			ModifyID:                1,
		}
		assert.Equal(t, exp, result, "UModifyOrder should decode every field")
	}
}

func TestUFuturesNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UFuturesNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UFuturesNewOrder must reject a nil request")
	arg := &UFuturesNewOrderRequest{}
	_, err = e.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UFuturesNewOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "UFuturesNewOrder must reject an empty side")
	arg.Side = order.Buy.String()
	_, err = e.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "UFuturesNewOrder must reject an empty order type")
	arg.OrderType = order.Limit.String()
	arg.Price = 80000
	arg.PriceMatch = "QUEUE"
	_, err = e.UFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "UFuturesNewOrder must reject a price match with a price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []*UFuturesNewOrderRequest{
		{
			Symbol: currency.NewBTCUSDT(), Side: "BUY", PositionSide: "LONG", OrderType: "LIMIT", TimeInForce: "GTD", Quantity: 0.002,
			Price: 80000, NewClientOrderID: "testOrder", NewOrderRespType: "RESULT", SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate: time.UnixMilli(1744300800000),
		},
		{Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "LIMIT", TimeInForce: "GTC", ReduceOnly: true, Quantity: 0.002, PriceMatch: "QUEUE"},
	} {
		result, err := e.UFuturesNewOrder(t.Context(), tc)
		require.NoErrorf(t, err, "UFuturesNewOrder must not error for side %s", tc.Side)
		if mockTests {
			exp := &UNewOrderResponse{
				ClientOrderID:           "testOrder",
				ExecutedQuantity:        0.001,
				OrderID:                 22542179,
				OriginalQuantity:        0.002,
				Price:                   80000,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "LONG",
				Status:                  "PARTIALLY_FILLED",
				StopPrice:               79000,
				ClosePosition:           true,
				Symbol:                  "BTCUSDT",
				TimeInForce:             "GTD",
				Type:                    "LIMIT",
				OriginalType:            "TAKE_PROFIT",
				UpdateTime:              types.Time(time.Unix(1744118400, 0)),
				WorkingType:             "MARK_PRICE",
				PriceProtect:            true,
				PriceMatch:              "QUEUE",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
				CumulativeQuantity:      0.001,
			}
			assert.Equal(t, exp, result, "UFuturesNewOrder should decode every field")
		}
	}
}

func TestUCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UCancelOrder(t.Context(), currency.EMPTYPAIR, 123, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UCancelOrder must reject an empty symbol")
	_, err = e.UCancelOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UCancelOrder must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UCancelOrder(t.Context(), currency.NewBTCUSDT(), 123, "")
	require.NoError(t, err, "UCancelOrder must not error")
	if mockTests {
		exp := &UCancelOrderResponse{
			ClientOrderID:           "myOrder1",
			ExecutedQuantity:        0.001,
			OrderID:                 123,
			OriginalQuantity:        0.002,
			Price:                   80000,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "LONG",
			Status:                  "CANCELED",
			StopPrice:               79000,
			ClosePosition:           true,
			Symbol:                  "BTCUSDT",
			TimeInForce:             "GTD",
			Type:                    "LIMIT",
			OriginalType:            "TAKE_PROFIT",
			UpdateTime:              types.Time(time.Unix(1744118400, 0)),
			WorkingType:             "MARK_PRICE",
			PriceProtect:            true,
			PriceMatch:              "QUEUE",
			SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
			CumulativeQuantity:      0.001,
			ActivatePrice:           79500,
			PriceRate:               0.3,
		}
		assert.Equal(t, exp, result, "UCancelOrder should decode every field")
	}
}

func TestUChangeInitialLeverageRequest(t *testing.T) {
	t.Parallel()
	_, err := e.UChangeInitialLeverageRequest(t.Context(), currency.EMPTYPAIR, 2)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UChangeInitialLeverageRequest must reject an empty symbol")
	_, err = e.UChangeInitialLeverageRequest(t.Context(), currency.NewBTCUSDT(), 0)
	require.ErrorIs(t, err, errInvalidLeverage, "UChangeInitialLeverageRequest must reject zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UChangeInitialLeverageRequest(t.Context(), currency.NewBTCUSDT(), 2)
	require.NoError(t, err, "UChangeInitialLeverageRequest must not error")
	if mockTests {
		exp := &UChangeInitialLeverageResponse{
			Leverage:         2,
			MaxNotionalValue: 1000000,
			Symbol:           "BTCUSDT",
		}
		assert.Equal(t, exp, result, "UChangeInitialLeverageRequest should decode every field")
	}
}

func TestUChangeInitialMarginType(t *testing.T) {
	t.Parallel()
	err := e.UChangeInitialMarginType(t.Context(), currency.EMPTYPAIR, "ISOLATED")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UChangeInitialMarginType must reject an empty symbol")
	err = e.UChangeInitialMarginType(t.Context(), currency.NewBTCUSDT(), "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "UChangeInitialMarginType must reject an empty margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.UChangeInitialMarginType(t.Context(), currency.NewBTCUSDT(), "ISOLATED"), "UChangeInitialMarginType should not error")
}

func TestUCurrentAllAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UCurrentAllAlgoOpenOrders(t.Context(), "CONDITIONAL", currency.NewBTCUSDT(), 2146760)
	require.NoError(t, err, "UCurrentAllAlgoOpenOrders must not error for a symbol")
	if mockTests {
		exp := []UAlgoOrder{
			{
				AlgoID:                  2146760,
				ClientAlgoID:            "6B2I9XVcJpCjqPAJ4YoFX7",
				AlgoType:                "CONDITIONAL",
				OrderType:               "TAKE_PROFIT",
				Symbol:                  "BTCUSDT",
				Side:                    "SELL",
				PositionSide:            "LONG",
				TimeInForce:             "GTC",
				Quantity:                0.01,
				AlgoStatus:              "NEW",
				TriggerPrice:            90000,
				Price:                   90000,
				IcebergQuantity:         "null",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				WorkingType:             "MARK_PRICE",
				PriceMatch:              "QUEUE",
				ClosePosition:           true,
				PriceProtect:            true,
				ReduceOnly:              true,
				CreateTime:              types.Time(time.UnixMilli(1750485492076)),
				UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
				TriggerTime:             types.Time(time.UnixMilli(1750514545091)),
				GoodTillDate:            types.Time(time.Unix(1750550400, 0)),
				ActualOrderID:           "20072994037",
				ActualPrice:             90000,
				TakeProfitTriggerPrice:  91000,
				TakeProfitPrice:         91000,
				StopLossTriggerPrice:    85000,
				StopLossPrice:           85000,
				TakeProfitOrderType:     "LIMIT",
			},
		}
		assert.Equal(t, exp, result, "UCurrentAllAlgoOpenOrders should decode every field")
	}
	result, err = e.UCurrentAllAlgoOpenOrders(t.Context(), "", currency.EMPTYPAIR, 0)
	require.NoError(t, err, "UCurrentAllAlgoOpenOrders must not error for every symbol")
	if mockTests {
		exp := []UAlgoOrder{
			{
				AlgoID:                  2146760,
				ClientAlgoID:            "6B2I9XVcJpCjqPAJ4YoFX7",
				AlgoType:                "CONDITIONAL",
				OrderType:               "TAKE_PROFIT",
				Symbol:                  "BTCUSDT",
				Side:                    "SELL",
				PositionSide:            "LONG",
				TimeInForce:             "GTC",
				Quantity:                0.01,
				AlgoStatus:              "NEW",
				TriggerPrice:            90000,
				Price:                   90000,
				IcebergQuantity:         "null",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				WorkingType:             "MARK_PRICE",
				PriceMatch:              "QUEUE",
				ClosePosition:           true,
				PriceProtect:            true,
				ReduceOnly:              true,
				CreateTime:              types.Time(time.UnixMilli(1750485492076)),
				UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
				TriggerTime:             types.Time(time.UnixMilli(1750514545091)),
				GoodTillDate:            types.Time(time.Unix(1750550400, 0)),
				ActualOrderID:           "20072994037",
				ActualPrice:             90000,
				TakeProfitTriggerPrice:  91000,
				TakeProfitPrice:         91000,
				StopLossTriggerPrice:    85000,
				StopLossPrice:           85000,
				TakeProfitOrderType:     "LIMIT",
			},
		}
		assert.Equal(t, exp, result, "UCurrentAllAlgoOpenOrders should decode every field for every symbol")
	}
}

func TestUAllAccountOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, symbol := range []currency.Pair{currency.NewBTCUSDT(), currency.EMPTYPAIR} {
		result, err := e.UAllAccountOpenOrders(t.Context(), symbol)
		require.NoErrorf(t, err, "UAllAccountOpenOrders must not error for symbol %q", symbol)
		if mockTests {
			exp := []UOrderResponse{
				{
					ClientOrderID:           "testOrder",
					ExecutedQuantity:        0.001,
					OrderID:                 22542179,
					OriginalQuantity:        0.002,
					Price:                   80000,
					ReduceOnly:              true,
					Side:                    "BUY",
					PositionSide:            "LONG",
					Status:                  "PARTIALLY_FILLED",
					StopPrice:               79000,
					ClosePosition:           true,
					Symbol:                  "BTCUSDT",
					TimeInForce:             "GTD",
					Type:                    "LIMIT",
					OriginalType:            "TAKE_PROFIT",
					UpdateTime:              types.Time(time.Unix(1744118400, 0)),
					WorkingType:             "MARK_PRICE",
					PriceProtect:            true,
					PriceMatch:              "QUEUE",
					SelfTradePreventionMode: "EXPIRE_MAKER",
					GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
					AveragePrice:            80000,
					CumulativeQuote:         80,
					Time:                    types.Time(time.Unix(1744118000, 0)),
					ActivatePrice:           79500,
					PriceRate:               0.3,
				},
			}
			assert.Equal(t, exp, result, "UAllAccountOpenOrders should decode every field")
		}
	}
}

func TestGetUSDTOrderModifyHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTOrderModifyHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDTOrderModifyHistory must reject a nil request")
	_, err = e.GetUSDTOrderModifyHistory(t.Context(), &UOrderModifyHistoryRequest{OrderID: 20072994037})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTOrderModifyHistory must reject an empty symbol")
	_, err = e.GetUSDTOrderModifyHistory(t.Context(), &UOrderModifyHistoryRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetUSDTOrderModifyHistory must reject a missing order ID")
	startTime, endTime := getTime()
	_, err = e.GetUSDTOrderModifyHistory(t.Context(), &UOrderModifyHistoryRequest{Symbol: currency.NewBTCUSDT(), OrderID: 20072994037, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDTOrderModifyHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUSDTOrderModifyHistory(t.Context(), &UOrderModifyHistoryRequest{Symbol: currency.NewBTCUSDT(), OrderID: 20072994037, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUSDTOrderModifyHistory must not error")
	if mockTests {
		exp := []FuturesOrderAmendment{
			{
				AmendmentID:   5363,
				Symbol:        "BTCUSDT",
				Pair:          "BTCUSDT",
				OrderID:       20072994037,
				ClientOrderID: "LJ9R4QZDihCaS8UAOOLpgW",
				Time:          types.Time(time.Unix(1744118400, 0)),
				Amendment: OrderAmendment{
					Price: AmendmentChange{
						Before: 30004,
						After:  30003.2,
					},
					OriginalQuantity: AmendmentChange{
						Before: 1,
						After:  2,
					},
					Count:    3,
					ModifyID: 123,
				},
			},
		}
		assert.Equal(t, exp, result, "GetUSDTOrderModifyHistory should decode every field")
	}
}

func TestUPositionMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.UPositionMarginChangeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UPositionMarginChangeHistory must reject a nil request")
	_, err = e.UPositionMarginChangeHistory(t.Context(), &UPositionMarginChangeHistoryRequest{ChangeType: "add"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UPositionMarginChangeHistory must reject an empty symbol")
	_, err = e.UPositionMarginChangeHistory(t.Context(), &UPositionMarginChangeHistoryRequest{Symbol: currency.NewBTCUSDT(), ChangeType: "remove"})
	require.ErrorIs(t, err, errMarginChangeTypeInvalid, "UPositionMarginChangeHistory must reject an unknown change type")
	startTime, endTime := getTime()
	_, err = e.UPositionMarginChangeHistory(t.Context(), &UPositionMarginChangeHistoryRequest{Symbol: currency.NewBTCUSDT(), ChangeType: "add", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UPositionMarginChangeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UPositionMarginChangeHistory(t.Context(), &UPositionMarginChangeHistoryRequest{Symbol: currency.NewBTCUSDT(), ChangeType: "add", StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "UPositionMarginChangeHistory must not error")
	if mockTests {
		exp := []UPositionMarginChange{
			{
				Symbol:       "BTCUSDT",
				Type:         1,
				DeltaType:    "USER_ADJUST",
				Amount:       23.36332311,
				Asset:        currency.USDT,
				Time:         types.Time(time.Unix(1744118400, 0)),
				PositionSide: "LONG",
			},
		}
		assert.Equal(t, exp, result, "UPositionMarginChangeHistory should decode every field")
	}
}

func TestUModifyIsolatedPositionMarginReq(t *testing.T) {
	t.Parallel()
	_, err := e.UModifyIsolatedPositionMarginReq(t.Context(), currency.EMPTYPAIR, "LONG", "add", 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UModifyIsolatedPositionMarginReq must reject an empty symbol")
	_, err = e.UModifyIsolatedPositionMarginReq(t.Context(), currency.NewBTCUSDT(), "LONG", "", 5)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid, "UModifyIsolatedPositionMarginReq must reject an empty change type")
	_, err = e.UModifyIsolatedPositionMarginReq(t.Context(), currency.NewBTCUSDT(), "LONG", "add", 0)
	require.ErrorIs(t, err, errMarginAmountRequired, "UModifyIsolatedPositionMarginReq must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UModifyIsolatedPositionMarginReq(t.Context(), currency.NewBTCUSDT(), "LONG", "add", 5)
	require.NoError(t, err, "UModifyIsolatedPositionMarginReq must not error")
	if mockTests {
		exp := &UModifyIsolatedPositionMarginResponse{
			Amount:  5,
			Code:    200,
			Message: "Successfully modify position margin.",
			Type:    1,
		}
		assert.Equal(t, exp, result, "UModifyIsolatedPositionMarginReq should decode every field")
	}
}

func TestUPositionsADLEstimate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UPositionsADLEstimate(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UPositionsADLEstimate must not error")
	if mockTests {
		exp := []UPositionADLQuantile{
			{
				Symbol: "BTCUSDT",
				ADLQuantile: UADLQuantile{
					Long:  3,
					Short: 2,
					Hedge: 1,
					Both:  4,
				},
			},
		}
		assert.Equal(t, exp, result, "UPositionsADLEstimate should decode every field")
	}
}

func TestUPositionsInfoV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UPositionsInfoV2(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UPositionsInfoV2 must not error")
	if mockTests {
		exp := []UPositionInformationV2{
			{
				Symbol:           "BTCUSDT",
				PositionAmount:   0.001,
				EntryPrice:       80000,
				BreakEvenPrice:   80032,
				MarkPrice:        81000,
				UnrealizedProfit: 1,
				LiquidationPrice: 72400.5,
				Leverage:         10,
				MaxNotionalValue: 20000000,
				MarginType:       "isolated",
				IsolatedMargin:   8.1,
				IsAutoAddMargin:  true,
				PositionSide:     "BOTH",
				Notional:         81,
				IsolatedWallet:   8,
				UpdateTime:       types.Time(time.UnixMilli(1625474304765)),
			},
		}
		assert.Equal(t, exp, result, "UPositionsInfoV2 should decode every field")
	}
}

func TestUPositionsInfoV3(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UPositionsInfoV3(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "UPositionsInfoV3 must not error")
	if mockTests {
		exp := []UPositionInformationV3{
			{
				Symbol:                 "BTCUSDT",
				PositionSide:           "BOTH",
				PositionAmount:         0.001,
				EntryPrice:             80000,
				BreakEvenPrice:         80032,
				MarkPrice:              81000,
				UnrealizedProfit:       1,
				LiquidationPrice:       72400.5,
				IsolatedMargin:         8.1,
				Notional:               81,
				MarginAsset:            currency.USDT,
				IsolatedWallet:         8,
				InitialMargin:          8.1,
				MaintenanceMargin:      0.405,
				PositionInitialMargin:  8.1,
				OpenOrderInitialMargin: 1.25,
				ADL:                    2,
				BidNotional:            12.5,
				AskNotional:            6.25,
				UpdateTime:             types.Time(time.UnixMilli(1720736417660)),
			},
		}
		assert.Equal(t, exp, result, "UPositionsInfoV3 should decode every field")
	}
}

func TestUQueryAllAlgoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UQueryAllAlgoOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UQueryAllAlgoOrders must reject a nil request")
	_, err = e.UQueryAllAlgoOrders(t.Context(), &UAllAlgoOrdersRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UQueryAllAlgoOrders must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.UQueryAllAlgoOrders(t.Context(), &UAllAlgoOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UQueryAllAlgoOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UQueryAllAlgoOrders(t.Context(), &UAllAlgoOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "UQueryAllAlgoOrders must not error")
	if mockTests {
		exp := []UAlgoOrder{
			{
				AlgoID:                  2146760,
				ClientAlgoID:            "6B2I9XVcJpCjqPAJ4YoFX7",
				AlgoType:                "CONDITIONAL",
				OrderType:               "TAKE_PROFIT",
				Symbol:                  "BTCUSDT",
				Side:                    "SELL",
				PositionSide:            "LONG",
				TimeInForce:             "GTC",
				Quantity:                0.01,
				AlgoStatus:              "NEW",
				TriggerPrice:            90000,
				Price:                   90000,
				IcebergQuantity:         "null",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				WorkingType:             "MARK_PRICE",
				PriceMatch:              "QUEUE",
				ClosePosition:           true,
				PriceProtect:            true,
				ReduceOnly:              true,
				CreateTime:              types.Time(time.UnixMilli(1750485492076)),
				UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
				TriggerTime:             types.Time(time.UnixMilli(1750514545091)),
				GoodTillDate:            types.Time(time.Unix(1750550400, 0)),
				ActualOrderID:           "20072994037",
				ActualPrice:             90000,
				TakeProfitTriggerPrice:  91000,
				TakeProfitPrice:         91000,
				StopLossTriggerPrice:    85000,
				StopLossPrice:           85000,
				TakeProfitOrderType:     "LIMIT",
			},
		}
		assert.Equal(t, exp, result, "UQueryAllAlgoOrders should decode every field")
	}
}

func TestUFetchOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UFetchOpenOrder(t.Context(), currency.EMPTYPAIR, 123, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UFetchOpenOrder must reject an empty symbol")
	_, err = e.UFetchOpenOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "UFetchOpenOrder must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UFetchOpenOrder(t.Context(), currency.NewBTCUSDT(), 123, "")
	require.NoError(t, err, "UFetchOpenOrder must not error")
	if mockTests {
		exp := &UOrderResponse{
			ClientOrderID:           "testOrder",
			ExecutedQuantity:        0.001,
			OrderID:                 123,
			OriginalQuantity:        0.002,
			Price:                   80000,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "LONG",
			Status:                  "PARTIALLY_FILLED",
			StopPrice:               79000,
			ClosePosition:           true,
			Symbol:                  "BTCUSDT",
			TimeInForce:             "GTD",
			Type:                    "LIMIT",
			OriginalType:            "TAKE_PROFIT",
			UpdateTime:              types.Time(time.Unix(1744118400, 0)),
			WorkingType:             "MARK_PRICE",
			PriceProtect:            true,
			PriceMatch:              "QUEUE",
			SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate:            types.Time(time.Unix(1744300800, 0)),
			AveragePrice:            80000,
			CumulativeQuote:         80,
			Time:                    types.Time(time.Unix(1744118000, 0)),
			ActivatePrice:           79500,
			PriceRate:               0.3,
		}
		assert.Equal(t, exp, result, "UFetchOpenOrder should decode every field")
	}
}

func TestUAccountForcedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.UAccountForcedOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UAccountForcedOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.UAccountForcedOrders(t.Context(), &UForceOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "UAccountForcedOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.UAccountForcedOrders(t.Context(), &UForceOrdersRequest{Symbol: currency.NewBTCUSDT(), AutoCloseType: "ADL", StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "UAccountForcedOrders must not error")
	if mockTests {
		exp := []UForceOrder{
			{
				OrderID:          6071832819,
				Symbol:           "BTCUSDT",
				Pair:             "BTCUSDT",
				Status:           "FILLED",
				ClientOrderID:    "autoclose-1596107620040000020",
				Price:            10871.09,
				AveragePrice:     10913.21,
				OriginalQuantity: 0.001,
				ExecutedQuantity: 0.001,
				CumulativeQuote:  10.91321,
				CumulativeBase:   0.001,
				TimeInForce:      "IOC",
				Type:             "LIMIT",
				ReduceOnly:       true,
				ClosePosition:    true,
				Side:             "SELL",
				PositionSide:     "BOTH",
				StopPrice:        10800,
				WorkingType:      "CONTRACT_PRICE",
				OriginalType:     "LIMIT",
				Time:             types.Time(time.UnixMilli(1744118400044)),
				UpdateTime:       types.Time(time.UnixMilli(1744118400087)),
			},
		}
		assert.Equal(t, exp, result, "UAccountForcedOrders should decode every field")
	}
}

func TestKeepaliveUFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		_, err := e.StartUFuturesUserDataStream(t.Context())
		require.NoError(t, err, "StartUFuturesUserDataStream must not error")
	}
	result, err := e.KeepaliveUFuturesUserDataStream(t.Context())
	require.NoError(t, err, "KeepaliveUFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "KeepaliveUFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "KeepaliveUFuturesUserDataStream should return the listen key")
}

func TestStartUFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.StartUFuturesUserDataStream(t.Context())
	require.NoError(t, err, "StartUFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "StartUFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "StartUFuturesUserDataStream should return a listen key")
}

func TestCloseUFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		// Closing invalidates the listen key every client of the account shares
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CloseUFuturesUserDataStream(t.Context()), "CloseUFuturesUserDataStream should not error")
}
