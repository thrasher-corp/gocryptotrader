package binance

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
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

func TestUSDTMarginedFuturesAccountInformationV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesAccountInformationV2(t.Context())
	require.NoError(t, err, "USDTMarginedFuturesAccountInformationV2 must not error")
	if mockTests {
		exp := &USDTMarginedFuturesAccountInformationV2Response{
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
			TotalUnrealisedProfit:       0.00211,
			TotalMarginBalance:          126.72680206,
			TotalPositionInitialMargin:  0.33683,
			TotalOpenOrderInitialMargin: 0.125,
			TotalCrossWalletBalance:     126.72469206,
			TotalCrossUnrealisedPNL:     0.00211,
			AvailableBalance:            126.26497206,
			MaxWithdrawAmount:           126.26497206,
			Assets: []USDTMarginedFuturesAsset{
				{
					Asset:                  currency.USDT,
					WalletBalance:          23.72469206,
					UnrealisedProfit:       0.00211,
					MarginBalance:          23.72680206,
					MaintenanceMargin:      0.02695,
					InitialMargin:          0.46183,
					PositionInitialMargin:  0.33683,
					OpenOrderInitialMargin: 0.125,
					CrossWalletBalance:     23.72469206,
					CrossUnrealisedPNL:     0.00211,
					AvailableBalance:       23.26497206,
					MaxWithdrawAmount:      23.26497206,
					MarginAvailable:        true,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []USDTMarginedFuturesPosition{
				{
					Symbol:                 "BTCUSDT",
					InitialMargin:          0.33683,
					MaintenanceMargin:      0.02695,
					UnrealisedProfit:       0.00211,
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesAccountInformationV2 should decode every field")
		return
	}
	assert.NotNil(t, result, "USDTMarginedFuturesAccountInformationV2 should return account information")
}

func TestUSDTMarginedFuturesAccountInformationV3(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesAccountInformationV3(t.Context())
	require.NoError(t, err, "USDTMarginedFuturesAccountInformationV3 must not error")
	if mockTests {
		exp := &USDTMarginedFuturesAccountInformationV3Response{
			TotalInitialMargin:          0.33683,
			TotalMaintenanceMargin:      0.02695,
			TotalWalletBalance:          126.72469206,
			TotalUnrealisedProfit:       0.00211,
			TotalMarginBalance:          126.72680206,
			TotalPositionInitialMargin:  0.33683,
			TotalOpenOrderInitialMargin: 0.125,
			TotalCrossWalletBalance:     126.72469206,
			TotalCrossUnrealisedPNL:     0.00211,
			AvailableBalance:            126.26497206,
			MaxWithdrawAmount:           126.26497206,
			Assets: []USDTMarginedFuturesAssetV3{
				{
					Asset:                  currency.USDT,
					WalletBalance:          23.72469206,
					UnrealisedProfit:       0.00211,
					MarginBalance:          23.72680206,
					MaintenanceMargin:      0.02695,
					InitialMargin:          0.46183,
					PositionInitialMargin:  0.33683,
					OpenOrderInitialMargin: 0.125,
					CrossWalletBalance:     23.72469206,
					CrossUnrealisedPNL:     0.00211,
					AvailableBalance:       23.26497206,
					MaxWithdrawAmount:      23.26497206,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []USDTMarginedFuturesPositionV3{
				{
					Symbol:            "BTCUSDT",
					PositionSide:      "BOTH",
					PositionAmount:    0.001,
					UnrealisedProfit:  0.00211,
					IsolatedMargin:    0.33894,
					Notional:          33.68321,
					IsolatedWallet:    0.33683,
					InitialMargin:     0.33683,
					MaintenanceMargin: 0.02695,
					UpdateTime:        types.Time(time.UnixMilli(1625474304765)),
				},
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesAccountInformationV3 should decode every field")
		return
	}
	assert.NotNil(t, result, "USDTMarginedFuturesAccountInformationV3 should return account information")
}

func TestUSDTMarginedFuturesAccountBalanceV3(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesAccountBalanceV3(t.Context())
	require.NoError(t, err, "USDTMarginedFuturesAccountBalanceV3 must not error")
	if mockTests {
		exp := []USDTMarginedFuturesAccountBalanceResponse{
			{
				AccountAlias:       "SgsR",
				Asset:              currency.USDT,
				Balance:            122607.35137903,
				CrossWalletBalance: 23.72469206,
				CrossUnrealisedPNL: -0.00211,
				AvailableBalance:   23.72469206,
				MaxWithdrawAmount:  23.72469206,
				MarginAvailable:    true,
				UpdateTime:         types.Time(time.UnixMilli(1617939110373)),
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesAccountBalanceV3 should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesAccountBalanceV3 should return balances")
}

func TestUSDTMarginedFuturesTradingQuantitativeRulesIndicators(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesTradingQuantitativeRulesIndicators(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesTradingQuantitativeRulesIndicators must not error for a symbol")
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesTradingQuantitativeRulesIndicators should decode every field for a symbol")
	}
	result, err = e.USDTMarginedFuturesTradingQuantitativeRulesIndicators(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USDTMarginedFuturesTradingQuantitativeRulesIndicators must not error for every symbol")
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesTradingQuantitativeRulesIndicators should decode every field for every symbol")
		return
	}
	assert.NotNil(t, result, "USDTMarginedFuturesTradingQuantitativeRulesIndicators should return indicators")
}

func TestGetUSDTMarginedFuturesAssetsMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUSDTMarginedFuturesAssetsMode(t.Context())
	require.NoError(t, err, "GetUSDTMarginedFuturesAssetsMode must not error")
	if mockTests {
		assert.True(t, result, "GetUSDTMarginedFuturesAssetsMode should report Multi-Assets Mode")
	}
}

func TestGetUSDTMarginedFuturesCurrentPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUSDTMarginedFuturesCurrentPositionMode(t.Context())
	require.NoError(t, err, "GetUSDTMarginedFuturesCurrentPositionMode must not error")
	if mockTests {
		exp := &USDTMarginedFuturesPositionModeResponse{
			DualSidePosition: true,
		}
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesCurrentPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUSDTMarginedFuturesCurrentPositionMode should return the position mode")
}

func TestUSDTMarginedFuturesDownloadIDs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		fn   func(*testing.T, time.Time, time.Time) (*USDTMarginedFuturesDownloadIDResponse, error)
	}{
		{"USDTMarginedFuturesOrderHistoryDownloadID", func(t *testing.T, start, end time.Time) (*USDTMarginedFuturesDownloadIDResponse, error) {
			t.Helper()
			return e.USDTMarginedFuturesOrderHistoryDownloadID(t.Context(), start, end)
		}},
		{"USDTMarginedFuturesTradeHistoryDownloadID", func(t *testing.T, start, end time.Time) (*USDTMarginedFuturesDownloadIDResponse, error) {
			t.Helper()
			return e.USDTMarginedFuturesTradeHistoryDownloadID(t.Context(), start, end)
		}},
		{"USDTMarginedFuturesTransactionHistoryDownloadID", func(t *testing.T, start, end time.Time) (*USDTMarginedFuturesDownloadIDResponse, error) {
			t.Helper()
			return e.USDTMarginedFuturesTransactionHistoryDownloadID(t.Context(), start, end)
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
				exp := &USDTMarginedFuturesDownloadIDResponse{
					AverageDownloadMillisecondsOfLast30Days: 7241837,
					DownloadID:                              "546975389218332672",
				}
				assert.Equal(t, exp, result, "should decode every field")
				return
			}
			assert.NotEmpty(t, result.DownloadID, "should return a download ID")
		})
	}
}

func TestUSDTMarginedFuturesDownloadLinks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		fn   func(*testing.T, string) (*USDTMarginedFuturesDownloadLinkResponse, error)
	}{
		{"USDTMarginedFuturesOrderHistoryDownloadLink", func(t *testing.T, id string) (*USDTMarginedFuturesDownloadLinkResponse, error) {
			t.Helper()
			return e.USDTMarginedFuturesOrderHistoryDownloadLink(t.Context(), id)
		}},
		{"USDTMarginedFuturesTradeHistoryDownloadLink", func(t *testing.T, id string) (*USDTMarginedFuturesDownloadLinkResponse, error) {
			t.Helper()
			return e.USDTMarginedFuturesTradeHistoryDownloadLink(t.Context(), id)
		}},
		{"USDTMarginedFuturesTransactionHistoryDownloadLink", func(t *testing.T, id string) (*USDTMarginedFuturesDownloadLinkResponse, error) {
			t.Helper()
			return e.USDTMarginedFuturesTransactionHistoryDownloadLink(t.Context(), id)
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
				exp := &USDTMarginedFuturesDownloadLinkResponse{
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

func TestUSDTMarginedFuturesAccountIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesAccountIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesAccountIncomeHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.USDTMarginedFuturesAccountIncomeHistory(t.Context(), &USDTMarginedFuturesIncomeHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesAccountIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesAccountIncomeHistory(t.Context(), &USDTMarginedFuturesIncomeHistoryRequest{
		Symbol:     currency.NewBTCUSDT(),
		IncomeType: "REALIZED_PNL",
		StartTime:  startTime,
		EndTime:    endTime,
		Page:       1,
		Limit:      5,
	})
	require.NoError(t, err, "USDTMarginedFuturesAccountIncomeHistory must not error")
	if mockTests {
		exp := []USDTMarginedFuturesAccountIncomeHistoryResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesAccountIncomeHistory should decode every field")
	}
}

func TestUSDTMarginedFuturesGetNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesGetNotionalAndLeverageBrackets(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesGetNotionalAndLeverageBrackets must not error for a symbol")
	if mockTests {
		exp := []USDTMarginedFuturesNotionalLeverageAndBracketsResponse{
			{
				Symbol:              "BTCUSDT",
				NotionalCoefficient: 1.5,
				Brackets: []USDTMarginedFuturesNotionalBracket{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetNotionalAndLeverageBrackets should decode the single symbol object")
	}
	result, err = e.USDTMarginedFuturesGetNotionalAndLeverageBrackets(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USDTMarginedFuturesGetNotionalAndLeverageBrackets must not error for every symbol")
	if mockTests {
		exp := []USDTMarginedFuturesNotionalLeverageAndBracketsResponse{
			{
				Symbol:              "BTCUSDT",
				NotionalCoefficient: 1.5,
				Brackets: []USDTMarginedFuturesNotionalBracket{
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
				Brackets: []USDTMarginedFuturesNotionalBracket{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetNotionalAndLeverageBrackets should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesGetNotionalAndLeverageBrackets should return brackets")
}

func TestGetUSDTMarginedFuturesUserRateLimits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUSDTMarginedFuturesUserRateLimits(t.Context())
	require.NoError(t, err, "GetUSDTMarginedFuturesUserRateLimits must not error")
	if mockTests {
		exp := []RateLimitInfoResponse{
			{
				RateLimitType:  "ORDERS",
				Interval:       "SECOND",
				IntervalNumber: 10,
				Limit:          300,
			},
			{
				RateLimitType:  "ORDERS",
				Interval:       "MINUTE",
				IntervalNumber: 1,
				Limit:          1200,
			},
		}
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesUserRateLimits should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesUserRateLimits should return rate limits")
}

func TestUSDTMarginedFuturesGetCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesGetCommissionRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesGetCommissionRates must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesGetCommissionRates(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesGetCommissionRates must not error")
	if mockTests {
		exp := &USDTMarginedFuturesCommissionRateResponse{
			Symbol:                               "BTCUSDT",
			MakerCommissionRate:                  0.0002,
			TakerCommissionRate:                  0.0004,
			RetailPriceImprovementCommissionRate: 0.00005,
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetCommissionRates should decode every field")
		return
	}
	assert.Equal(t, "BTCUSDT", result.Symbol, "USDTMarginedFuturesGetCommissionRates should return the requested symbol")
}

func TestGetUSDTMarginedFuturesBasis(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesBasis(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDTMarginedFuturesBasis must reject a nil request")
	arg := &USDTMarginedFuturesBasisRequest{ContractType: "CURRENT_QUARTER", Period: kline.FifteenMin}
	_, err = e.GetUSDTMarginedFuturesBasis(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesBasis must reject an empty pair")
	arg.Pair = currency.NewBTCUSDT()
	arg.ContractType = ""
	_, err = e.GetUSDTMarginedFuturesBasis(t.Context(), arg)
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetUSDTMarginedFuturesBasis must reject an empty contract type")
	arg.ContractType = "CURRENT_QUARTER"
	arg.Period = 0
	_, err = e.GetUSDTMarginedFuturesBasis(t.Context(), arg)
	require.ErrorIs(t, err, errPeriodRequired, "GetUSDTMarginedFuturesBasis must reject an empty period")
	arg.Period = kline.FifteenMin
	startTime, endTime := getTime()
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetUSDTMarginedFuturesBasis(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDTMarginedFuturesBasis must reject a start time after the end time")
	_, err = new(Exchange).GetUSDTMarginedFuturesBasis(t.Context(), &USDTMarginedFuturesBasisRequest{Pair: currency.NewBTCUSDT(), ContractType: "CURRENT_QUARTER", Period: kline.FifteenMin})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetUSDTMarginedFuturesBasis must return the error formatting the pair")

	arg.StartTime, arg.EndTime, arg.Limit = startTime, endTime, 20
	result, err := e.GetUSDTMarginedFuturesBasis(t.Context(), arg)
	require.NoError(t, err, "GetUSDTMarginedFuturesBasis must not error")
	if mockTests {
		exp := []BasisInfoResponse{
			{
				IndexPrice:          86002.56086957,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0108,
				FuturesPrice:        86928.7,
				AnnualisedBasisRate: 0.0493,
				Basis:               926.13913043,
				Pair:                "BTCUSDT",
				Timestamp:           types.Time(time.Unix(1791280800, 0)),
			},
			{
				IndexPrice:          86159.96913043,
				ContractType:        "CURRENT_QUARTER",
				BasisRate:           0.0108,
				FuturesPrice:        87087.9,
				AnnualisedBasisRate: 0.0493,
				Basis:               927.93086957,
				Pair:                "BTCUSDT",
				Timestamp:           types.Time(time.Unix(1791281700, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesBasis should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesBasis should return basis records")
}

func TestGetUSDTMarginedFuturesBasisPeriod(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		period kline.Interval
		exp    string
	}{
		{kline.FiveMin, "5m"},
		{kline.FifteenMin, "15m"},
		{kline.ThirtyMin, "30m"},
		{kline.OneHour, "1h"},
		{kline.TwoHour, "2h"},
		{kline.FourHour, "4h"},
		{kline.SixHour, "6h"},
		{kline.TwelveHour, "12h"},
		{kline.OneDay, "1d"},
	} {
		t.Run(tc.exp, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/futures/data/basis", r.URL.Path, "GetUSDTMarginedFuturesBasis should request the basis path")
				assert.Equal(t, url.Values{"pair": {"BTCUSDT"}, "contractType": {"PERPETUAL"}, "period": {tc.exp}}, r.URL.Query(), "GetUSDTMarginedFuturesBasis should send the period as Binance writes it")
				_, err := w.Write([]byte(`[]`))
				assert.NoError(t, err, "Write should not error")
			})
			_, err := ex.GetUSDTMarginedFuturesBasis(t.Context(), &USDTMarginedFuturesBasisRequest{Pair: currency.NewBTCUSDT(), ContractType: ContractTypePerpetual, Period: tc.period})
			assert.NoError(t, err, "GetUSDTMarginedFuturesBasis should not error")
		})
	}
}

func TestUSDTMarginedFuturesServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFuturesServerTime(t.Context())
	require.NoError(t, err, "USDTMarginedFuturesServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1791283479731), result, "USDTMarginedFuturesServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "USDTMarginedFuturesServerTime should return the current time")
}

func TestUSDTMarginedFuturesCompositeIndexInfo(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFuturesCompositeIndexInfo(t.Context(), currency.NewPair(currency.NewCode("SMALL"), currency.USDT))
	require.NoError(t, err, "USDTMarginedFuturesCompositeIndexInfo must not error for a symbol")
	if mockTests {
		exp := []USDTMarginedFuturesCompositeIndexResponse{
			{
				Symbol:    "SMALLUSDT",
				Time:      types.Time(time.UnixMilli(1791283544001)),
				Component: "baseAsset",
				BaseAssetList: []USDTMarginedFuturesCompositeIndexBaseAsset{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesCompositeIndexInfo should decode the single symbol object")
	}
	result, err = e.USDTMarginedFuturesCompositeIndexInfo(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USDTMarginedFuturesCompositeIndexInfo must not error for every symbol")
	if mockTests {
		exp := []USDTMarginedFuturesCompositeIndexResponse{
			{
				Symbol:    "ALLUSDT",
				Time:      types.Time(time.Unix(1791283499, 0)),
				Component: "baseAsset",
				BaseAssetList: []USDTMarginedFuturesCompositeIndexBaseAsset{
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
				BaseAssetList: []USDTMarginedFuturesCompositeIndexBaseAsset{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesCompositeIndexInfo should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesCompositeIndexInfo should return composite indexes")
}

func TestUSDTMarginedFuturesCompressedTrades(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesCompressedTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesCompressedTrades must reject a nil request")
	_, err = e.USDTMarginedFuturesCompressedTrades(t.Context(), &USDTMarginedFuturesCompressedTradesRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesCompressedTrades must reject an empty symbol")
	startTime, endTime := getTime(true)
	_, err = e.USDTMarginedFuturesCompressedTrades(t.Context(), &USDTMarginedFuturesCompressedTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesCompressedTrades must reject a start time after the end time")

	if !mockTests {
		// Binance only serves the last 48 hours, and a window must be shorter than an hour
		endTime = time.Now()
		startTime = endTime.Add(-30 * time.Minute)
	}
	result, err := e.USDTMarginedFuturesCompressedTrades(t.Context(), &USDTMarginedFuturesCompressedTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "USDTMarginedFuturesCompressedTrades must not error for a time window")
	require.NotEmpty(t, result, "USDTMarginedFuturesCompressedTrades must return trades")
	if mockTests {
		exp := []USDTMarginedFuturesCompressedTradeResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesCompressedTrades should decode every field")
	}
	_, err = e.USDTMarginedFuturesCompressedTrades(t.Context(), &USDTMarginedFuturesCompressedTradesRequest{Symbol: currency.NewBTCUSDT(), FromID: result[0].AggregateTradeID, Limit: 5})
	assert.NoError(t, err, "USDTMarginedFuturesCompressedTrades should not error from a trade ID")
}

func TestGetUSDTMarginedFuturesContinuousKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesContinuousKlineData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDTMarginedFuturesContinuousKlineData must reject a nil request")
	arg := &USDTMarginedFuturesContinuousKlineRequest{ContractType: "PERPETUAL", Interval: kline.OneDay}
	_, err = e.GetUSDTMarginedFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesContinuousKlineData must reject an empty pair")
	arg.Pair = currency.NewBTCUSDT()
	arg.ContractType = ""
	_, err = e.GetUSDTMarginedFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, errContractTypeIsRequired, "GetUSDTMarginedFuturesContinuousKlineData must reject an empty contract type")
	arg.ContractType = "PERPETUAL"
	arg.Interval = 0
	_, err = e.GetUSDTMarginedFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetUSDTMarginedFuturesContinuousKlineData must reject an empty interval")
	arg.Interval = kline.OneDay
	startTime, endTime := getTime()
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetUSDTMarginedFuturesContinuousKlineData(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDTMarginedFuturesContinuousKlineData must reject a start time after the end time")

	arg.StartTime, arg.EndTime, arg.Limit = startTime, endTime, 10
	result, err := e.GetUSDTMarginedFuturesContinuousKlineData(t.Context(), arg)
	require.NoError(t, err, "GetUSDTMarginedFuturesContinuousKlineData must not error")
	if mockTests {
		exp := []USDTMarginedFuturesKlineResponse{
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
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesContinuousKlineData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesContinuousKlineData should return klines")
}

func TestUSDTMarginedFuturesExchangeInfo(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFuturesExchangeInfo(t.Context())
	require.NoError(t, err, "USDTMarginedFuturesExchangeInfo must not error")
	if mockTests {
		exp := &USDTMarginedFuturesExchangeInfoResponse{
			Timezone:    "UTC",
			ServerTime:  types.Time(time.UnixMilli(1791278101115)),
			FuturesType: "U_MARGINED",
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
				{
					RateLimitType:  "ORDERS",
					Interval:       "SECOND",
					IntervalNumber: 10,
					Limit:          300,
				},
			},
			ExchangeFilters: []string{},
			Assets: []USDTMarginedFuturesExchangeInfoAsset{
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
			Symbols: []USDTMarginedFuturesSymbolInfo{
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
					Filters: []USDTMarginedFuturesSymbolFilter{
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
					OrderTypes: []OrderType{
						"LIMIT",
						"MARKET",
						"STOP",
						"STOP_MARKET",
						"TAKE_PROFIT",
						"TAKE_PROFIT_MARKET",
						"TRAILING_STOP_MARKET",
					},
					TimeInForce: []TimeInForce{
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
					Filters: []USDTMarginedFuturesSymbolFilter{
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
					OrderTypes: []OrderType{
						"LIMIT",
						"MARKET",
						"STOP",
						"STOP_MARKET",
						"TAKE_PROFIT",
						"TAKE_PROFIT_MARKET",
						"TRAILING_STOP_MARKET",
					},
					TimeInForce: []TimeInForce{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesExchangeInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbols, "USDTMarginedFuturesExchangeInfo should return symbols")
}

func TestUSDTMarginedFuturesGetFundingHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.USDTMarginedFuturesGetFundingHistory(t.Context(), currency.NewBTCUSDT(), endTime, startTime, 1000)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesGetFundingHistory must reject a start time after the end time")

	result, err := e.USDTMarginedFuturesGetFundingHistory(t.Context(), currency.NewBTCUSDT(), startTime, endTime, 1000)
	require.NoError(t, err, "USDTMarginedFuturesGetFundingHistory must not error")
	if mockTests {
		exp := []FundingRateHistoryResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetFundingHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesGetFundingHistory should return funding rates")
}

func TestUSDTMarginedFuturesGetFundingRateInfo(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFuturesGetFundingRateInfo(t.Context())
	require.NoError(t, err, "USDTMarginedFuturesGetFundingRateInfo must not error")
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetFundingRateInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesGetFundingRateInfo should return funding rate info")
}

func TestGetUSDTMarginedFuturesIndexPriceKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesIndexPriceKlineCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDTMarginedFuturesIndexPriceKlineCandlesticks must reject a nil request")
	arg := &USDTMarginedFuturesIndexPriceKlineRequest{Interval: kline.OneDay}
	_, err = e.GetUSDTMarginedFuturesIndexPriceKlineCandlesticks(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesIndexPriceKlineCandlesticks must reject an empty pair")
	arg.Pair = currency.NewBTCUSDT()
	arg.Interval = 0
	_, err = e.GetUSDTMarginedFuturesIndexPriceKlineCandlesticks(t.Context(), arg)
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetUSDTMarginedFuturesIndexPriceKlineCandlesticks must reject an empty interval")
	arg.Interval = kline.OneDay
	startTime, endTime := getTime()
	arg.StartTime, arg.EndTime = endTime, startTime
	_, err = e.GetUSDTMarginedFuturesIndexPriceKlineCandlesticks(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDTMarginedFuturesIndexPriceKlineCandlesticks must reject a start time after the end time")

	arg.StartTime, arg.EndTime, arg.Limit = startTime, endTime, 100
	result, err := e.GetUSDTMarginedFuturesIndexPriceKlineCandlesticks(t.Context(), arg)
	require.NoError(t, err, "GetUSDTMarginedFuturesIndexPriceKlineCandlesticks must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPriceKlineResponse{
			{
				OpenTime:  types.Time(time.Unix(1744156800, 0)),
				Open:      76322.968,
				High:      83567.47466667,
				Low:       74626.60222222,
				Close:     82611.44511111,
				CloseTime: types.Time(time.UnixMilli(1744243199999)),
			},
		}
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesIndexPriceKlineCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesIndexPriceKlineCandlesticks should return klines")
}

func TestUSDTMarginedFuturesKlineData(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesKlineData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesKlineData must reject a nil request")
	_, err = e.USDTMarginedFuturesKlineData(t.Context(), &USDTMarginedFuturesKlineRequest{Interval: kline.OneDay, Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesKlineData must reject an empty symbol")
	_, err = e.USDTMarginedFuturesKlineData(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Limit: 5})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "USDTMarginedFuturesKlineData must reject an empty interval")
	startTime, endTime := getTime()
	_, err = e.USDTMarginedFuturesKlineData(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.FiveMin, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesKlineData must reject a start time after the end time")

	result, err := e.USDTMarginedFuturesKlineData(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneDay, StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "USDTMarginedFuturesKlineData must not error for a limit")
	if mockTests {
		exp := []USDTMarginedFuturesKlineResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesKlineData should decode every field")
	}
	result, err = e.USDTMarginedFuturesKlineData(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.FiveMin, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "USDTMarginedFuturesKlineData must not error without a limit")
	if mockTests {
		exp := []USDTMarginedFuturesKlineResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesKlineData should decode every field without a limit")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesKlineData should return klines")
}

func TestUSDTMarginedFuturesKlineDataRequest(t *testing.T) {
	t.Parallel()
	var resp []any
	err := e.usdtMarginedFuturesKlineData(t.Context(), "/fapi/v1/klines", url.Values{"symbol": {"BTCUSDT"}}, 0, time.Time{}, time.Time{}, 0, &resp)
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "usdtMarginedFuturesKlineData must reject an empty interval")
	startTime, endTime := getTime()
	err = e.usdtMarginedFuturesKlineData(t.Context(), "/fapi/v1/klines", url.Values{"symbol": {"BTCUSDT"}}, kline.OneMin, endTime, startTime, 0, &resp)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "usdtMarginedFuturesKlineData must reject a start time after the end time")

	for _, tc := range []struct {
		name     string
		interval kline.Interval
		limit    uint64
		exp      url.Values
	}{
		{"one month with a limit", kline.OneMonth, 3, url.Values{"symbol": {"BTCUSDT"}, "interval": {"1M"}, "limit": {"3"}}},
		{"one minute without a limit", kline.OneMin, 0, url.Values{"symbol": {"BTCUSDT"}, "interval": {"1m"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/fapi/v1/klines", r.URL.Path, "usdtMarginedFuturesKlineData should request the given path")
				assert.Equal(t, tc.exp, r.URL.Query(), "usdtMarginedFuturesKlineData should send the interval as Binance writes it, and any limit")
				_, err := w.Write([]byte(`[]`))
				assert.NoError(t, err, "Write should not error")
			})
			var result []any
			require.NoError(t, ex.usdtMarginedFuturesKlineData(t.Context(), "/fapi/v1/klines", url.Values{"symbol": {"BTCUSDT"}}, tc.interval, time.Time{}, time.Time{}, tc.limit, &result), "usdtMarginedFuturesKlineData must not error")
		})
	}
}

func TestUSDTMarginedFuturesGlobalLongShortRatio(t *testing.T) {
	t.Parallel()
	testUSDTMarginedFuturesDataValidation(t, e.USDTMarginedFuturesGlobalLongShortRatio)
	startTime, endTime := getTime()
	result, err := e.USDTMarginedFuturesGlobalLongShortRatio(t.Context(), &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin, StartTime: startTime, EndTime: endTime, Limit: 3})
	require.NoError(t, err, "USDTMarginedFuturesGlobalLongShortRatio must not error")
	if mockTests {
		exp := []USDTMarginedFuturesLongShortRatioResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGlobalLongShortRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesGlobalLongShortRatio should return ratios")
}

func TestUSDTMarginedFuturesData(t *testing.T) {
	t.Parallel()
	var resp []any
	err := e.usdtMarginedFuturesData(t.Context(), "/futures/data/openInterestHist", nil, &resp)
	require.ErrorIs(t, err, common.ErrNilPointer, "usdtMarginedFuturesData must reject a nil request")
	err = e.usdtMarginedFuturesData(t.Context(), "/futures/data/openInterestHist", &USDTMarginedFuturesDataRequest{Period: kline.FiveMin}, &resp)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "usdtMarginedFuturesData must reject an empty symbol")
	err = e.usdtMarginedFuturesData(t.Context(), "/futures/data/openInterestHist", &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT()}, &resp)
	require.ErrorIs(t, err, errPeriodRequired, "usdtMarginedFuturesData must reject an empty period")
	startTime, endTime := getTime()
	err = e.usdtMarginedFuturesData(t.Context(), "/futures/data/openInterestHist", &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin, StartTime: endTime, EndTime: startTime}, &resp)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "usdtMarginedFuturesData must reject a start time after the end time")
	err = new(Exchange).usdtMarginedFuturesData(t.Context(), "/futures/data/openInterestHist", &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin}, &resp)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "usdtMarginedFuturesData must return the error formatting the symbol")

	for _, tc := range []struct {
		name   string
		period kline.Interval
		limit  uint64
		exp    url.Values
	}{
		{"five minutes with a limit", kline.FiveMin, 3, url.Values{"symbol": {"BTCUSDT"}, "period": {"5m"}, "limit": {"3"}}},
		{"one day without a limit", kline.OneDay, 0, url.Values{"symbol": {"BTCUSDT"}, "period": {"1d"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/futures/data/openInterestHist", r.URL.Path, "usdtMarginedFuturesData should request the given path")
				assert.Equal(t, tc.exp, r.URL.Query(), "usdtMarginedFuturesData should send the symbol, the period as Binance writes it and any limit")
				_, err := w.Write([]byte(`[]`))
				assert.NoError(t, err, "Write should not error")
			})
			var result []any
			require.NoError(t, ex.usdtMarginedFuturesData(t.Context(), "/futures/data/openInterestHist", &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: tc.period, Limit: tc.limit}, &result), "usdtMarginedFuturesData must not error")
		})
	}
}

// testUSDTMarginedFuturesDataValidation checks the request validation the /futures/data statistics endpoints share
func testUSDTMarginedFuturesDataValidation[T any](t *testing.T, fn func(context.Context, *USDTMarginedFuturesDataRequest) ([]T, error)) {
	t.Helper()
	_, err := fn(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "must reject a nil request")
	_, err = fn(t.Context(), &USDTMarginedFuturesDataRequest{Period: kline.FiveMin})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "must reject an empty symbol")
	_, err = fn(t.Context(), &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, errPeriodRequired, "must reject an empty period")
	startTime, endTime := getTime()
	_, err = fn(t.Context(), &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "must reject a start time after the end time")
}

func TestUSDTMarginedFuturesGetMarkPrice(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFuturesGetMarkPrice(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesGetMarkPrice must not error for a symbol")
	if mockTests {
		exp := []USDTMarginedFuturesMarkPriceResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetMarkPrice should decode the single symbol object")
	}
	result, err = e.USDTMarginedFuturesGetMarkPrice(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USDTMarginedFuturesGetMarkPrice must not error for every symbol")
	if mockTests {
		exp := []USDTMarginedFuturesMarkPriceResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetMarkPrice should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesGetMarkPrice should return mark prices")
}

func TestGetUSDTMarginedFuturesMarkPriceKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesMarkPriceKlineCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDTMarginedFuturesMarkPriceKlineCandlesticks must reject a nil request")
	_, err = e.GetUSDTMarginedFuturesMarkPriceKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Interval: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesMarkPriceKlineCandlesticks must reject an empty symbol")
	_, err = e.GetUSDTMarginedFuturesMarkPriceKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetUSDTMarginedFuturesMarkPriceKlineCandlesticks must reject an empty interval")
	startTime, endTime := getTime()
	_, err = e.GetUSDTMarginedFuturesMarkPriceKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneDay, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDTMarginedFuturesMarkPriceKlineCandlesticks must reject a start time after the end time")

	result, err := e.GetUSDTMarginedFuturesMarkPriceKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneDay, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUSDTMarginedFuturesMarkPriceKlineCandlesticks must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPriceKlineResponse{
			{
				OpenTime:  types.Time(time.Unix(1744156800, 0)),
				Open:      76297.9,
				High:      83553.4,
				Low:       74588.1,
				Close:     82587.9,
				CloseTime: types.Time(time.UnixMilli(1744243199999)),
			},
		}
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesMarkPriceKlineCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesMarkPriceKlineCandlesticks should return klines")
}

func TestGetUSDTMarginedFuturesMultiAssetModeAssetIndex(t *testing.T) {
	t.Parallel()
	result, err := e.GetUSDTMarginedFuturesMultiAssetModeAssetIndex(t.Context(), currency.NewBTCUSD())
	require.NoError(t, err, "GetUSDTMarginedFuturesMultiAssetModeAssetIndex must not error for a symbol")
	if mockTests {
		exp := []AssetIndexResponse{
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
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesMultiAssetModeAssetIndex should decode the single symbol object")
	}
	result, err = e.GetUSDTMarginedFuturesMultiAssetModeAssetIndex(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetUSDTMarginedFuturesMultiAssetModeAssetIndex must not error for every symbol")
	if mockTests {
		exp := []AssetIndexResponse{
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
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesMultiAssetModeAssetIndex should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesMultiAssetModeAssetIndex should return asset indexes")
}

func TestUSDTMarginedFuturesHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesHistoricalTrades(t.Context(), currency.EMPTYPAIR, 0, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesHistoricalTrades must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, fromID := range []uint64{0, 8149534980} {
		result, err := e.USDTMarginedFuturesHistoricalTrades(t.Context(), currency.NewBTCUSDT(), fromID, 5)
		require.NoErrorf(t, err, "USDTMarginedFuturesHistoricalTrades must not error from trade ID %d", fromID)
		if mockTests {
			exp := []USDTMarginedFuturesPublicTradesResponse{
				{
					ID:            8149534980,
					Price:         86065.5,
					Quantity:      0.001,
					QuoteQuantity: 86.06,
					Time:          types.Time(time.UnixMilli(1791283323016)),
					IsBuyerMaker:  true,
				},
				{
					ID:                            8149534981,
					Price:                         86065.4,
					Quantity:                      0.059,
					QuoteQuantity:                 5077.86,
					Time:                          types.Time(time.UnixMilli(1791283323999)),
					IsRetailPriceImprovementTrade: true,
				},
			}
			assert.Equal(t, exp, result, "USDTMarginedFuturesHistoricalTrades should decode every field")
			continue
		}
		assert.NotEmptyf(t, result, "USDTMarginedFuturesHistoricalTrades should return trades from trade ID %d", fromID)
	}
}

func TestUSDTMarginedFuturesOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesOpenInterest(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesOpenInterest must reject an empty symbol")

	result, err := e.USDTMarginedFuturesOpenInterest(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesOpenInterest must not error")
	if mockTests {
		exp := &USDTMarginedFuturesOpenInterestResponse{
			OpenInterest: 94598.71,
			Symbol:       "BTCUSDT",
			Time:         types.Time(time.UnixMilli(1791283485477)),
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesOpenInterest should decode every field")
		return
	}
	assert.Positive(t, result.OpenInterest.Float64(), "USDTMarginedFuturesOpenInterest should return open interest")
}

func TestUSDTMarginedFuturesOpenInterestStats(t *testing.T) {
	t.Parallel()
	testUSDTMarginedFuturesDataValidation(t, e.USDTMarginedFuturesOpenInterestStats)
	startTime, endTime := getTime()
	result, err := e.USDTMarginedFuturesOpenInterestStats(t.Context(), &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin, StartTime: startTime, EndTime: endTime, Limit: 1})
	require.NoError(t, err, "USDTMarginedFuturesOpenInterestStats must not error")
	if mockTests {
		exp := []USDTMarginedFuturesOpenInterestStatsResponse{
			{
				Symbol:               "BTCUSDT",
				SumOpenInterest:      94576.939,
				SumOpenInterestValue: 8134776100.048762,
				CMCCirculatingSupply: 20094037,
				Timestamp:            types.Time(time.Unix(1791282900, 0)),
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesOpenInterestStats should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesOpenInterestStats should return statistics")
}

func TestUSDTMarginedFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesOrderbook(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesOrderbook must reject an empty symbol")

	result, err := e.USDTMarginedFuturesOrderbook(t.Context(), currency.NewBTCUSDT(), 1000)
	require.NoError(t, err, "USDTMarginedFuturesOrderbook must not error")
	if mockTests {
		exp := &USDTMarginedFuturesOrderBookResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "USDTMarginedFuturesOrderbook should return bids")
}

func TestGetUSDTMarginedFuturesPremiumIndexKlineCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks must reject a nil request")
	_, err = e.GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Interval: kline.OneDay})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks must reject an empty symbol")
	_, err = e.GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks must reject an empty interval")
	startTime, endTime := getTime()
	_, err = e.GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneDay, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks must reject a start time after the end time")

	result, err := e.GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks(t.Context(), &USDTMarginedFuturesKlineRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneDay, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPriceKlineResponse{
			{
				OpenTime:  types.Time(time.Unix(1744156800, 0)),
				Open:      -0.00032714,
				High:      0.00180625,
				Low:       -0.0025365,
				Close:     -0.00028604,
				CloseTime: types.Time(time.UnixMilli(1744243199999)),
			},
		}
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks should return klines")
}

func TestGetUSDTMarginedFuturesQuarterlyContractSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesQuarterlyContractSettlementPrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesQuarterlyContractSettlementPrice must reject an empty pair")

	result, err := e.GetUSDTMarginedFuturesQuarterlyContractSettlementPrice(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetUSDTMarginedFuturesQuarterlyContractSettlementPrice must not error")
	if mockTests {
		exp := []SettlementPriceResponse{
			{
				DeliveryTime:  types.Time(time.Unix(1782432000, 0)),
				DeliveryPrice: 60608.3,
			},
			{
				DeliveryTime:  types.Time(time.Unix(1774569600, 0)),
				DeliveryPrice: 68656.7,
			},
		}
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesQuarterlyContractSettlementPrice should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUSDTMarginedFuturesQuarterlyContractSettlementPrice should return settlement prices")
}

func TestGetUSDTMarginedFuturesIndexPriceConstituents(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesIndexPriceConstituents(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesIndexPriceConstituents must reject an empty symbol")

	result, err := e.GetUSDTMarginedFuturesIndexPriceConstituents(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetUSDTMarginedFuturesIndexPriceConstituents must not error")
	if mockTests {
		exp := &USDTMarginedFuturesIndexPriceConstituentsResponse{
			Symbol: "BTCUSDT",
			Time:   types.Time(time.UnixMilli(1791283487198)),
			Constituents: []USDTMarginedFuturesIndexPriceConstituent{
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
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesIndexPriceConstituents should decode every field")
		return
	}
	assert.NotEmpty(t, result.Constituents, "GetUSDTMarginedFuturesIndexPriceConstituents should return constituents")
}

func TestUSDTMarginedFuturesRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesRecentTrades(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesRecentTrades must reject an empty symbol")

	result, err := e.USDTMarginedFuturesRecentTrades(t.Context(), currency.NewBTCUSDT(), 1000)
	require.NoError(t, err, "USDTMarginedFuturesRecentTrades must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPublicTradesResponse{
			{
				ID:            8149534980,
				Price:         86065.5,
				Quantity:      0.001,
				QuoteQuantity: 86.06,
				Time:          types.Time(time.UnixMilli(1791283323016)),
				IsBuyerMaker:  true,
			},
			{
				ID:                            8149534982,
				Price:                         86065.6,
				Quantity:                      0.001,
				QuoteQuantity:                 86.06,
				Time:                          types.Time(time.UnixMilli(1791283324072)),
				IsRetailPriceImprovementTrade: true,
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesRecentTrades should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesRecentTrades should return trades")
}

func TestGetUSDTMarginedFuturesRetailPriceImprovementOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesRetailPriceImprovementOrderbook(t.Context(), currency.EMPTYPAIR, 1000)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesRetailPriceImprovementOrderbook must reject an empty symbol")

	result, err := e.GetUSDTMarginedFuturesRetailPriceImprovementOrderbook(t.Context(), currency.NewBTCUSDT(), 1000)
	require.NoError(t, err, "GetUSDTMarginedFuturesRetailPriceImprovementOrderbook must not error")
	if mockTests {
		exp := &USDTMarginedFuturesOrderBookResponse{
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
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesRetailPriceImprovementOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetUSDTMarginedFuturesRetailPriceImprovementOrderbook should return bids")
}

func TestUSDTMarginedFuturesSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFuturesSymbolOrderbookTicker(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesSymbolOrderbookTicker must not error for a symbol")
	if mockTests {
		exp := []USDTMarginedFuturesSymbolOrderbookTickerResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesSymbolOrderbookTicker should decode the single symbol object")
	}
	result, err = e.USDTMarginedFuturesSymbolOrderbookTicker(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USDTMarginedFuturesSymbolOrderbookTicker must not error for every symbol")
	if mockTests {
		exp := []USDTMarginedFuturesSymbolOrderbookTickerResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesSymbolOrderbookTicker should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesSymbolOrderbookTicker should return tickers")
}

func TestUSDTMarginedFuturesSymbolPriceTickerV2(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFuturesSymbolPriceTickerV2(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesSymbolPriceTickerV2 must not error for a symbol")
	if mockTests {
		exp := []USDTMarginedFuturesSymbolPriceTickerResponse{
			{
				Symbol: "BTCUSDT",
				Price:  86163,
				Time:   types.Time(time.UnixMilli(1791283491172)),
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesSymbolPriceTickerV2 should decode the single symbol object")
	}
	result, err = e.USDTMarginedFuturesSymbolPriceTickerV2(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USDTMarginedFuturesSymbolPriceTickerV2 must not error for every symbol")
	if mockTests {
		exp := []USDTMarginedFuturesSymbolPriceTickerResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesSymbolPriceTickerV2 should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesSymbolPriceTickerV2 should return prices")
}

func TestUSDTMarginedFuturesTakerBuySellVolume(t *testing.T) {
	t.Parallel()
	testUSDTMarginedFuturesDataValidation(t, e.USDTMarginedFuturesTakerBuySellVolume)
	startTime, endTime := getTime()
	result, err := e.USDTMarginedFuturesTakerBuySellVolume(t.Context(), &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "USDTMarginedFuturesTakerBuySellVolume must not error")
	if mockTests {
		exp := []USDTMarginedFuturesTakerVolumeResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesTakerBuySellVolume should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesTakerBuySellVolume should return volumes")
}

func TestUSDTMarginedFutures24HourTickerPriceChangeStats(t *testing.T) {
	t.Parallel()
	result, err := e.USDTMarginedFutures24HourTickerPriceChangeStats(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFutures24HourTickerPriceChangeStats must not error for a symbol")
	if mockTests {
		exp := []USDTMarginedFutures24HourPriceChangeStatsResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFutures24HourTickerPriceChangeStats should decode the single symbol object")
	}
	result, err = e.USDTMarginedFutures24HourTickerPriceChangeStats(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "USDTMarginedFutures24HourTickerPriceChangeStats must not error for every symbol")
	if mockTests {
		exp := []USDTMarginedFutures24HourPriceChangeStatsResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFutures24HourTickerPriceChangeStats should decode every symbol")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFutures24HourTickerPriceChangeStats should return statistics")
}

func TestUSDTMarginedFuturesTopAccountsLongShortRatio(t *testing.T) {
	t.Parallel()
	testUSDTMarginedFuturesDataValidation(t, e.USDTMarginedFuturesTopAccountsLongShortRatio)
	startTime, endTime := getTime()
	result, err := e.USDTMarginedFuturesTopAccountsLongShortRatio(t.Context(), &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin, StartTime: startTime, EndTime: endTime, Limit: 2})
	require.NoError(t, err, "USDTMarginedFuturesTopAccountsLongShortRatio must not error")
	if mockTests {
		exp := []USDTMarginedFuturesLongShortRatioResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesTopAccountsLongShortRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesTopAccountsLongShortRatio should return ratios")
}

func TestUSDTMarginedFuturesTopPositionsLongShortRatio(t *testing.T) {
	t.Parallel()
	testUSDTMarginedFuturesDataValidation(t, e.USDTMarginedFuturesTopPositionsLongShortRatio)
	startTime, endTime := getTime()
	result, err := e.USDTMarginedFuturesTopPositionsLongShortRatio(t.Context(), &USDTMarginedFuturesDataRequest{Symbol: currency.NewBTCUSDT(), Period: kline.FiveMin, StartTime: startTime, EndTime: endTime, Limit: 3})
	require.NoError(t, err, "USDTMarginedFuturesTopPositionsLongShortRatio must not error")
	if mockTests {
		exp := []USDTMarginedFuturesLongShortRatioResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesTopPositionsLongShortRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "USDTMarginedFuturesTopPositionsLongShortRatio should return ratios")
}

func TestUSDTMarginedFuturesSetAssetsMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.USDTMarginedFuturesSetAssetsMode(t.Context(), true), "USDTMarginedFuturesSetAssetsMode should not error")
}

func TestUSDTMarginedFuturesChangePositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.USDTMarginedFuturesChangePositionMode(t.Context(), false), "USDTMarginedFuturesChangePositionMode should not error")
}

func TestUSDTMarginedFuturesAccountTradesHistory(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesAccountTradesHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesAccountTradesHistory must reject a nil request")
	_, err = e.USDTMarginedFuturesAccountTradesHistory(t.Context(), &USDTMarginedFuturesAccountTradeListRequest{Limit: 5})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesAccountTradesHistory must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.USDTMarginedFuturesAccountTradesHistory(t.Context(), &USDTMarginedFuturesAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), FromID: 698759, StartTime: startTime})
	require.ErrorIs(t, err, errFromIDWithTimeWindow, "USDTMarginedFuturesAccountTradesHistory must reject a from ID with a start time")
	_, err = e.USDTMarginedFuturesAccountTradesHistory(t.Context(), &USDTMarginedFuturesAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesAccountTradesHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesAccountTradesHistory(t.Context(), &USDTMarginedFuturesAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "USDTMarginedFuturesAccountTradesHistory must not error")
	if mockTests {
		exp := []USDTMarginedFuturesAccountTradeHistoryResponse{
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
				RealisedPNL:     -0.91539999,
				Side:            "BUY",
				PositionSide:    "SHORT",
				Symbol:          "BTCUSDT",
				Pair:            "BTCUSDT",
				Time:            types.Time(time.Unix(1744118400, 0)),
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesAccountTradesHistory should decode every field")
	}
}

func TestUSDTMarginedFuturesAllAccountOrders(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesAllAccountOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesAllAccountOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.USDTMarginedFuturesAllAccountOrders(t.Context(), &USDTMarginedFuturesAllOrdersRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesAllAccountOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []*USDTMarginedFuturesAllOrdersRequest{
		{StartTime: startTime, EndTime: endTime},
		{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 5},
	} {
		result, err := e.USDTMarginedFuturesAllAccountOrders(t.Context(), tc)
		require.NoErrorf(t, err, "USDTMarginedFuturesAllAccountOrders must not error for symbol %q", tc.Symbol)
		if mockTests {
			exp := []USDTMarginedFuturesAllOrdersResponse{
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
			assert.Equal(t, exp, result, "USDTMarginedFuturesAllAccountOrders should decode every field")
		}
	}
}

func TestUSDTMarginedFuturesAutoCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesAutoCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR, 30*time.Second)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesAutoCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesAutoCancelAllOpenOrders(t.Context(), currency.NewBTCUSDT(), 30*time.Second)
	require.NoError(t, err, "USDTMarginedFuturesAutoCancelAllOpenOrders must not error")
	if mockTests {
		exp := &USDTMarginedFuturesAutoCancelAllOpenOrdersResponse{
			Symbol:                      "BTCUSDT",
			CountdownTimeInMilliseconds: 30000,
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesAutoCancelAllOpenOrders should decode every field")
	}
}

func TestUSDTMarginedFuturesQueryAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesQueryAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesQueryAlgoOrder must reject a missing algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesQueryAlgoOrder(t.Context(), 2146760, "")
	require.NoError(t, err, "USDTMarginedFuturesQueryAlgoOrder must not error")
	if mockTests {
		exp := &USDTMarginedFuturesAlgoOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesQueryAlgoOrder should decode every field")
	}
}

func TestUSDTMarginedFuturesNewAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesNewAlgoOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesNewAlgoOrder must reject a nil request")
	arg := &USDTMarginedFuturesNewAlgoOrderRequest{}
	_, err = e.USDTMarginedFuturesNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errAlgoTypeRequired, "USDTMarginedFuturesNewAlgoOrder must reject an empty algo type")
	arg.AlgoType = "CONDITIONAL"
	_, err = e.USDTMarginedFuturesNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesNewAlgoOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.USDTMarginedFuturesNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "USDTMarginedFuturesNewAlgoOrder must reject an empty side")
	arg.Side = SideSell
	_, err = e.USDTMarginedFuturesNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "USDTMarginedFuturesNewAlgoOrder must reject an empty order type")
	arg.OrderType = "STOP"
	arg.Price = 80000
	arg.PriceMatch = "QUEUE"
	_, err = e.USDTMarginedFuturesNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "USDTMarginedFuturesNewAlgoOrder must reject a price match with a price")
	arg.PriceMatch = ""
	arg.ClosePosition = true
	arg.Quantity = 0.01
	_, err = e.USDTMarginedFuturesNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionWithQuantity, "USDTMarginedFuturesNewAlgoOrder must reject a close position with a quantity")
	arg.Quantity = 0
	arg.ReduceOnly = true
	_, err = e.USDTMarginedFuturesNewAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionWithQuantity, "USDTMarginedFuturesNewAlgoOrder must reject a close position with reduce only")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []*USDTMarginedFuturesNewAlgoOrderRequest{
		{
			AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", PositionSide: "LONG", OrderType: "TAKE_PROFIT",
			TimeInForce: "GTD", Quantity: 0.01, TriggerPrice: 90000, WorkingType: "MARK_PRICE", PriceMatch: "QUEUE",
			PriceProtect: true, ClientAlgoID: "6B2I9XVcJpCjqPAJ4YoFX7", NewOrderResponseType: "RESULT",
			SelfTradePreventionMode: "EXPIRE_MAKER", GoodTillDate: time.UnixMilli(1750550400000),
		},
		{
			AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "TRAILING_STOP_MARKET",
			Quantity: 0.01, ReduceOnly: true, ActivatePrice: 89500, CallbackRate: 0.5,
		},
		{AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "STOP_MARKET", TriggerPrice: 80000, ClosePosition: true},
		{AlgoType: "CONDITIONAL", Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "STOP", Quantity: 0.01, Price: 80000, TriggerPrice: 80500},
	} {
		result, err := e.USDTMarginedFuturesNewAlgoOrder(t.Context(), tc)
		require.NoErrorf(t, err, "USDTMarginedFuturesNewAlgoOrder must not error for order type %s", tc.OrderType)
		if mockTests {
			exp := &USDTMarginedFuturesNewAlgoOrderResponse{
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
			assert.Equal(t, exp, result, "USDTMarginedFuturesNewAlgoOrder should decode every field")
		}
	}
}

func TestUSDTMarginedFuturesCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesCancelAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesCancelAlgoOrder must reject a missing algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesCancelAlgoOrder(t.Context(), 2146760, "")
	require.NoError(t, err, "USDTMarginedFuturesCancelAlgoOrder must not error")
	if mockTests {
		exp := &USDTMarginedFuturesCancelAlgoOrderResponse{
			AlgoID:       2146760,
			ClientAlgoID: "6B2I9XVcJpCjqPAJ4YoFX7",
			Code:         200,
			Message:      "success",
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesCancelAlgoOrder should decode every field")
	}
}

func TestUSDTMarginedFuturesCancelAllAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	err := e.USDTMarginedFuturesCancelAllAlgoOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesCancelAllAlgoOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.USDTMarginedFuturesCancelAllAlgoOpenOrders(t.Context(), currency.NewBTCUSDT()), "USDTMarginedFuturesCancelAllAlgoOpenOrders should not error")
}

func TestUSDTMarginedFuturesCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	err := e.USDTMarginedFuturesCancelAllOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesCancelAllOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.USDTMarginedFuturesCancelAllOpenOrders(t.Context(), currency.NewBTCUSDT()), "USDTMarginedFuturesCancelAllOpenOrders should not error")
}

func TestUSDTMarginedFuturesModifyMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesModifyMultipleOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "USDTMarginedFuturesModifyMultipleOrders must reject an empty batch")
	_, err = e.USDTMarginedFuturesModifyMultipleOrders(t.Context(), []USDTMarginedFuturesModifyOrderRequest{{}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesModifyMultipleOrders must validate every order")
	_, err = e.USDTMarginedFuturesModifyMultipleOrders(t.Context(), []USDTMarginedFuturesModifyOrderRequest{{OrderID: 1, ReduceOnly: true}})
	require.ErrorIs(t, err, errBatchModifyReduceOnly, "USDTMarginedFuturesModifyMultipleOrders must reject reduce only")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesModifyMultipleOrders(t.Context(), []USDTMarginedFuturesModifyOrderRequest{
		{OrderID: 20072994037, Symbol: currency.NewBTCUSDT(), Side: "BUY", Quantity: 1, Price: 30005, ModifyID: 1},
		{OrigClientOrderID: "myOrder1", Symbol: currency.NewBTCUSDT(), Side: "SELL", Quantity: 0.5, PriceMatch: "QUEUE"},
	})
	require.NoError(t, err, "USDTMarginedFuturesModifyMultipleOrders must not error")
	if mockTests {
		exp := []USDTMarginedFuturesModifyBatchOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesModifyMultipleOrders should decode every field")
	}
}

func TestUSDTMarginedFuturesPlaceBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesPlaceBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "USDTMarginedFuturesPlaceBatchOrders must reject an empty batch")
	_, err = e.USDTMarginedFuturesPlaceBatchOrders(t.Context(), []USDTMarginedFuturesNewOrderRequest{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesPlaceBatchOrders must validate every order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesPlaceBatchOrders(t.Context(), []USDTMarginedFuturesNewOrderRequest{
		{Symbol: currency.NewBTCUSDT(), Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 0.002, Price: 80000, NewClientOrderID: "testOrder"},
		{Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "MARKET", Quantity: 0.002, ReduceOnly: true},
	})
	require.NoError(t, err, "USDTMarginedFuturesPlaceBatchOrders must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPlaceBatchOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesPlaceBatchOrders should decode every field")
	}
}

func TestUSDTMarginedFuturesCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesCancelBatchOrders(t.Context(), currency.EMPTYPAIR, []uint64{283194212}, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesCancelBatchOrders must reject an empty symbol")
	_, err = e.USDTMarginedFuturesCancelBatchOrders(t.Context(), currency.NewBTCUSDT(), nil, nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesCancelBatchOrders must reject an empty ID list")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesCancelBatchOrders(t.Context(), currency.NewBTCUSDT(), []uint64{283194212}, []string{"myOrder2"})
	require.NoError(t, err, "USDTMarginedFuturesCancelBatchOrders must not error")
	if mockTests {
		exp := []USDTMarginedFuturesCancelBatchOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesCancelBatchOrders should decode every field")
	}
}

func TestUSDTMarginedFuturesGetOrderData(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesGetOrderData(t.Context(), currency.EMPTYPAIR, 123, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesGetOrderData must reject an empty symbol")
	_, err = e.USDTMarginedFuturesGetOrderData(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesGetOrderData must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesGetOrderData(t.Context(), currency.NewBTCUSDT(), 123, "")
	require.NoError(t, err, "USDTMarginedFuturesGetOrderData must not error")
	if mockTests {
		exp := &USDTMarginedFuturesOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesGetOrderData should decode every field")
	}
}

func TestUSDTMarginedFuturesModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesModifyOrder must reject a nil request")
	arg := &USDTMarginedFuturesModifyOrderRequest{}
	_, err = e.USDTMarginedFuturesModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesModifyOrder must reject a missing order ID")
	arg.OrderID = 20072994037
	_, err = e.USDTMarginedFuturesModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesModifyOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.USDTMarginedFuturesModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "USDTMarginedFuturesModifyOrder must reject an empty side")
	arg.Side = SideBuy
	_, err = e.USDTMarginedFuturesModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "USDTMarginedFuturesModifyOrder must reject an empty quantity")
	arg.Quantity = 1
	_, err = e.USDTMarginedFuturesModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "USDTMarginedFuturesModifyOrder must reject an empty price without a price match")
	arg.Price = 30005
	arg.PriceMatch = "QUEUE"
	_, err = e.USDTMarginedFuturesModifyOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "USDTMarginedFuturesModifyOrder must reject a price match with a price")
	arg.PriceMatch = ""

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.ModifyID = 1
	arg.ReduceOnly = true
	result, err := e.USDTMarginedFuturesModifyOrder(t.Context(), arg)
	require.NoError(t, err, "USDTMarginedFuturesModifyOrder must not error")
	if mockTests {
		exp := &USDTMarginedFuturesModifyOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesModifyOrder should decode every field")
	}
}

func TestUSDTMarginedFuturesNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesNewOrder must reject a nil request")
	arg := &USDTMarginedFuturesNewOrderRequest{}
	_, err = e.USDTMarginedFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesNewOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.USDTMarginedFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "USDTMarginedFuturesNewOrder must reject an empty side")
	arg.Side = SideBuy
	_, err = e.USDTMarginedFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "USDTMarginedFuturesNewOrder must reject an empty order type")
	arg.OrderType = OrderTypeLimit
	arg.Price = 80000
	arg.PriceMatch = "QUEUE"
	_, err = e.USDTMarginedFuturesNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "USDTMarginedFuturesNewOrder must reject a price match with a price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []*USDTMarginedFuturesNewOrderRequest{
		{
			Symbol: currency.NewBTCUSDT(), Side: "BUY", PositionSide: "LONG", OrderType: "LIMIT", TimeInForce: "GTD", Quantity: 0.002,
			Price: 80000, NewClientOrderID: "testOrder", NewOrderResponseType: "RESULT", SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate: time.UnixMilli(1744300800000),
		},
		{Symbol: currency.NewBTCUSDT(), Side: "SELL", OrderType: "LIMIT", TimeInForce: "GTC", ReduceOnly: true, Quantity: 0.002, PriceMatch: "QUEUE"},
	} {
		result, err := e.USDTMarginedFuturesNewOrder(t.Context(), tc)
		require.NoErrorf(t, err, "USDTMarginedFuturesNewOrder must not error for side %s", tc.Side)
		if mockTests {
			exp := &USDTMarginedFuturesNewOrderResponse{
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
			assert.Equal(t, exp, result, "USDTMarginedFuturesNewOrder should decode every field")
		}
	}
}

func TestUSDTMarginedFuturesCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesCancelOrder(t.Context(), currency.EMPTYPAIR, 123, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesCancelOrder must reject an empty symbol")
	_, err = e.USDTMarginedFuturesCancelOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesCancelOrder must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesCancelOrder(t.Context(), currency.NewBTCUSDT(), 123, "")
	require.NoError(t, err, "USDTMarginedFuturesCancelOrder must not error")
	if mockTests {
		exp := &USDTMarginedFuturesCancelOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesCancelOrder should decode every field")
	}
}

func TestUSDTMarginedFuturesChangeInitialLeverageRequest(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesChangeInitialLeverageRequest(t.Context(), currency.EMPTYPAIR, 2)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesChangeInitialLeverageRequest must reject an empty symbol")
	_, err = e.USDTMarginedFuturesChangeInitialLeverageRequest(t.Context(), currency.NewBTCUSDT(), 0)
	require.ErrorIs(t, err, errInvalidLeverage, "USDTMarginedFuturesChangeInitialLeverageRequest must reject zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesChangeInitialLeverageRequest(t.Context(), currency.NewBTCUSDT(), 2)
	require.NoError(t, err, "USDTMarginedFuturesChangeInitialLeverageRequest must not error")
	if mockTests {
		exp := &USDTMarginedFuturesChangeInitialLeverageResponse{
			Leverage:         2,
			MaxNotionalValue: 1000000,
			Symbol:           "BTCUSDT",
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesChangeInitialLeverageRequest should decode every field")
	}
}

func TestUSDTMarginedFuturesChangeInitialMarginType(t *testing.T) {
	t.Parallel()
	err := e.USDTMarginedFuturesChangeInitialMarginType(t.Context(), currency.EMPTYPAIR, "ISOLATED")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesChangeInitialMarginType must reject an empty symbol")
	err = e.USDTMarginedFuturesChangeInitialMarginType(t.Context(), currency.NewBTCUSDT(), "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "USDTMarginedFuturesChangeInitialMarginType must reject an empty margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.USDTMarginedFuturesChangeInitialMarginType(t.Context(), currency.NewBTCUSDT(), "ISOLATED"), "USDTMarginedFuturesChangeInitialMarginType should not error")
}

func TestUSDTMarginedFuturesCurrentAllAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesCurrentAllAlgoOpenOrders(t.Context(), "CONDITIONAL", currency.NewBTCUSDT(), 2146760)
	require.NoError(t, err, "USDTMarginedFuturesCurrentAllAlgoOpenOrders must not error for a symbol")
	if mockTests {
		exp := []USDTMarginedFuturesAlgoOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesCurrentAllAlgoOpenOrders should decode every field")
	}
	result, err = e.USDTMarginedFuturesCurrentAllAlgoOpenOrders(t.Context(), "", currency.EMPTYPAIR, 0)
	require.NoError(t, err, "USDTMarginedFuturesCurrentAllAlgoOpenOrders must not error for every symbol")
	if mockTests {
		exp := []USDTMarginedFuturesAlgoOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesCurrentAllAlgoOpenOrders should decode every field for every symbol")
	}
}

func TestUSDTMarginedFuturesAllAccountOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, symbol := range []currency.Pair{currency.NewBTCUSDT(), currency.EMPTYPAIR} {
		result, err := e.USDTMarginedFuturesAllAccountOpenOrders(t.Context(), symbol)
		require.NoErrorf(t, err, "USDTMarginedFuturesAllAccountOpenOrders must not error for symbol %q", symbol)
		if mockTests {
			exp := []USDTMarginedFuturesOrderResponse{
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
			assert.Equal(t, exp, result, "USDTMarginedFuturesAllAccountOpenOrders should decode every field")
		}
	}
}

func TestGetUSDTMarginedFuturesOrderModifyHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTMarginedFuturesOrderModifyHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUSDTMarginedFuturesOrderModifyHistory must reject a nil request")
	_, err = e.GetUSDTMarginedFuturesOrderModifyHistory(t.Context(), &USDTMarginedFuturesOrderModifyHistoryRequest{OrderID: 20072994037})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUSDTMarginedFuturesOrderModifyHistory must reject an empty symbol")
	_, err = e.GetUSDTMarginedFuturesOrderModifyHistory(t.Context(), &USDTMarginedFuturesOrderModifyHistoryRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetUSDTMarginedFuturesOrderModifyHistory must reject a missing order ID")
	startTime, endTime := getTime()
	_, err = e.GetUSDTMarginedFuturesOrderModifyHistory(t.Context(), &USDTMarginedFuturesOrderModifyHistoryRequest{Symbol: currency.NewBTCUSDT(), OrderID: 20072994037, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUSDTMarginedFuturesOrderModifyHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUSDTMarginedFuturesOrderModifyHistory(t.Context(), &USDTMarginedFuturesOrderModifyHistoryRequest{Symbol: currency.NewBTCUSDT(), OrderID: 20072994037, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUSDTMarginedFuturesOrderModifyHistory must not error")
	if mockTests {
		exp := []FuturesOrderAmendmentResponse{
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
		assert.Equal(t, exp, result, "GetUSDTMarginedFuturesOrderModifyHistory should decode every field")
	}
}

func TestUSDTMarginedFuturesPositionMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesPositionMarginChangeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesPositionMarginChangeHistory must reject a nil request")
	_, err = e.USDTMarginedFuturesPositionMarginChangeHistory(t.Context(), &USDTMarginedFuturesPositionMarginChangeHistoryRequest{Type: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesPositionMarginChangeHistory must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.USDTMarginedFuturesPositionMarginChangeHistory(t.Context(), &USDTMarginedFuturesPositionMarginChangeHistoryRequest{Symbol: currency.NewBTCUSDT(), Type: 1, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesPositionMarginChangeHistory must reject a start time after the end time")
	_, err = new(Exchange).USDTMarginedFuturesPositionMarginChangeHistory(t.Context(), &USDTMarginedFuturesPositionMarginChangeHistoryRequest{Symbol: currency.NewBTCUSDT(), Type: 1})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "USDTMarginedFuturesPositionMarginChangeHistory must return the error formatting the symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesPositionMarginChangeHistory(t.Context(), &USDTMarginedFuturesPositionMarginChangeHistoryRequest{Symbol: currency.NewBTCUSDT(), Type: 1, StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "USDTMarginedFuturesPositionMarginChangeHistory must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPositionMarginChangeResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesPositionMarginChangeHistory should decode every field")
	}
}

func TestUSDTMarginedFuturesPositionMarginChangeHistoryType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		changeType uint64
		exp        url.Values
	}{
		{"both directions", 0, url.Values{"symbol": {"BTCUSDT"}}},
		{"added", 1, url.Values{"symbol": {"BTCUSDT"}, "type": {"1"}}},
		{"reduced", 2, url.Values{"symbol": {"BTCUSDT"}, "type": {"2"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/fapi/v1/positionMargin/history", r.URL.Path, "USDTMarginedFuturesPositionMarginChangeHistory should request the history path")
				query := r.URL.Query()
				for _, signed := range []string{"recvWindow", "signature", "timestamp"} {
					query.Del(signed)
				}
				assert.Equal(t, tc.exp, query, "USDTMarginedFuturesPositionMarginChangeHistory should send the type only when it is set")
				_, err := w.Write([]byte(`[]`))
				assert.NoError(t, err, "Write should not error")
			})
			_, err := ex.USDTMarginedFuturesPositionMarginChangeHistory(t.Context(), &USDTMarginedFuturesPositionMarginChangeHistoryRequest{Symbol: currency.NewBTCUSDT(), Type: tc.changeType})
			require.NoError(t, err, "USDTMarginedFuturesPositionMarginChangeHistory must not error")
		})
	}
}

func TestUSDTMarginedFuturesModifyIsolatedPositionMarginRequest(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesModifyIsolatedPositionMarginRequest(t.Context(), currency.EMPTYPAIR, "LONG", 1, 5)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest must reject an empty symbol")
	_, err = e.USDTMarginedFuturesModifyIsolatedPositionMarginRequest(t.Context(), currency.NewBTCUSDT(), "LONG", 0, 5)
	require.ErrorIs(t, err, errMarginChangeTypeInvalid, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest must reject an empty change type")
	_, err = e.USDTMarginedFuturesModifyIsolatedPositionMarginRequest(t.Context(), currency.NewBTCUSDT(), "LONG", 1, 0)
	require.ErrorIs(t, err, errMarginAmountRequired, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest must reject a zero amount")
	_, err = new(Exchange).USDTMarginedFuturesModifyIsolatedPositionMarginRequest(t.Context(), currency.NewBTCUSDT(), "LONG", 1, 5)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest must return the error formatting the symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.USDTMarginedFuturesModifyIsolatedPositionMarginRequest(t.Context(), currency.NewBTCUSDT(), "LONG", 1, 5)
	require.NoError(t, err, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest must not error")
	if mockTests {
		exp := &USDTMarginedFuturesModifyIsolatedPositionMarginResponse{
			Amount:  5,
			Code:    200,
			Message: "Successfully modify position margin.",
			Type:    1,
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest should decode every field")
	}
}

func TestUSDTMarginedFuturesModifyIsolatedPositionMarginRequestType(t *testing.T) {
	t.Parallel()
	for _, changeType := range []uint64{1, 2} {
		t.Run(strconv.FormatUint(changeType, 10), func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/fapi/v1/positionMargin", r.URL.Path, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest should request the position margin path")
				assert.NoError(t, r.ParseForm(), "ParseForm should not error")
				assert.Equal(t, strconv.FormatUint(changeType, 10), r.Form.Get("type"), "USDTMarginedFuturesModifyIsolatedPositionMarginRequest should send the type as given")
				_, err := w.Write([]byte(`{"amount":5,"code":200,"msg":"Successfully modify position margin.","type":` + strconv.FormatUint(changeType, 10) + `}`))
				assert.NoError(t, err, "Write should not error")
			})
			_, err := ex.USDTMarginedFuturesModifyIsolatedPositionMarginRequest(t.Context(), currency.NewBTCUSDT(), "LONG", changeType, 5)
			require.NoError(t, err, "USDTMarginedFuturesModifyIsolatedPositionMarginRequest must not error")
		})
	}
}

func TestUSDTMarginedFuturesPositionsAutoDeleveragingEstimate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesPositionsAutoDeleveragingEstimate(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesPositionsAutoDeleveragingEstimate must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPositionAutoDeleveragingQuantileResponse{
			{
				Symbol: "BTCUSDT",
				AutoDeleveragingQuantile: USDTMarginedFuturesAutoDeleveragingQuantile{
					Long:  3,
					Short: 2,
					Hedge: 1,
					Both:  4,
				},
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesPositionsAutoDeleveragingEstimate should decode every field")
	}
}

func TestUSDTMarginedFuturesPositionsInfoV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesPositionsInfoV2(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesPositionsInfoV2 must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPositionInformationV2Response{
			{
				Symbol:           "BTCUSDT",
				PositionAmount:   0.001,
				EntryPrice:       80000,
				BreakEvenPrice:   80032,
				MarkPrice:        81000,
				UnrealisedProfit: 1,
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesPositionsInfoV2 should decode every field")
	}
}

func TestUSDTMarginedFuturesPositionsInfoV3(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesPositionsInfoV3(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "USDTMarginedFuturesPositionsInfoV3 must not error")
	if mockTests {
		exp := []USDTMarginedFuturesPositionInformationV3Response{
			{
				Symbol:                  "BTCUSDT",
				PositionSide:            "BOTH",
				PositionAmount:          0.001,
				EntryPrice:              80000,
				BreakEvenPrice:          80032,
				MarkPrice:               81000,
				UnrealisedProfit:        1,
				LiquidationPrice:        72400.5,
				IsolatedMargin:          8.1,
				Notional:                81,
				MarginAsset:             currency.USDT,
				IsolatedWallet:          8,
				InitialMargin:           8.1,
				MaintenanceMargin:       0.405,
				PositionInitialMargin:   8.1,
				OpenOrderInitialMargin:  1.25,
				AutoDeleveragingRanking: 2,
				BidNotional:             12.5,
				AskNotional:             6.25,
				UpdateTime:              types.Time(time.UnixMilli(1720736417660)),
			},
		}
		assert.Equal(t, exp, result, "USDTMarginedFuturesPositionsInfoV3 should decode every field")
	}
}

func TestUSDTMarginedFuturesQueryAllAlgoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesQueryAllAlgoOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesQueryAllAlgoOrders must reject a nil request")
	_, err = e.USDTMarginedFuturesQueryAllAlgoOrders(t.Context(), &USDTMarginedFuturesAllAlgoOrdersRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesQueryAllAlgoOrders must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.USDTMarginedFuturesQueryAllAlgoOrders(t.Context(), &USDTMarginedFuturesAllAlgoOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesQueryAllAlgoOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesQueryAllAlgoOrders(t.Context(), &USDTMarginedFuturesAllAlgoOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "USDTMarginedFuturesQueryAllAlgoOrders must not error")
	if mockTests {
		exp := []USDTMarginedFuturesAlgoOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesQueryAllAlgoOrders should decode every field")
	}
}

func TestUSDTMarginedFuturesFetchOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesFetchOpenOrder(t.Context(), currency.EMPTYPAIR, 123, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "USDTMarginedFuturesFetchOpenOrder must reject an empty symbol")
	_, err = e.USDTMarginedFuturesFetchOpenOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "USDTMarginedFuturesFetchOpenOrder must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesFetchOpenOrder(t.Context(), currency.NewBTCUSDT(), 123, "")
	require.NoError(t, err, "USDTMarginedFuturesFetchOpenOrder must not error")
	if mockTests {
		exp := &USDTMarginedFuturesOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesFetchOpenOrder should decode every field")
	}
}

func TestUSDTMarginedFuturesAccountForcedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.USDTMarginedFuturesAccountForcedOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "USDTMarginedFuturesAccountForcedOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.USDTMarginedFuturesAccountForcedOrders(t.Context(), &USDTMarginedFuturesForceOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "USDTMarginedFuturesAccountForcedOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.USDTMarginedFuturesAccountForcedOrders(t.Context(), &USDTMarginedFuturesForceOrdersRequest{Symbol: currency.NewBTCUSDT(), AutoCloseType: "ADL", StartTime: startTime, EndTime: endTime, Limit: 5})
	require.NoError(t, err, "USDTMarginedFuturesAccountForcedOrders must not error")
	if mockTests {
		exp := []USDTMarginedFuturesForceOrderResponse{
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
		assert.Equal(t, exp, result, "USDTMarginedFuturesAccountForcedOrders should decode every field")
	}
}

func TestKeepaliveUSDTMarginedFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		_, err := e.StartUSDTMarginedFuturesUserDataStream(t.Context())
		require.NoError(t, err, "StartUSDTMarginedFuturesUserDataStream must not error")
	}
	result, err := e.KeepaliveUSDTMarginedFuturesUserDataStream(t.Context())
	require.NoError(t, err, "KeepaliveUSDTMarginedFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "KeepaliveUSDTMarginedFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "KeepaliveUSDTMarginedFuturesUserDataStream should return the listen key")
}

func TestStartUSDTMarginedFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.StartUSDTMarginedFuturesUserDataStream(t.Context())
	require.NoError(t, err, "StartUSDTMarginedFuturesUserDataStream must not error")
	if mockTests {
		exp := &FuturesListenKeyResponse{ListenKey: "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}
		assert.Equal(t, exp, result, "StartUSDTMarginedFuturesUserDataStream should decode every field")
		return
	}
	assert.NotEmpty(t, result.ListenKey, "StartUSDTMarginedFuturesUserDataStream should return a listen key")
}

func TestCloseUSDTMarginedFuturesUserDataStream(t *testing.T) {
	t.Parallel()
	if !mockTests {
		// Closing invalidates the listen key every client of the account shares
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.CloseUSDTMarginedFuturesUserDataStream(t.Context()), "CloseUSDTMarginedFuturesUserDataStream should not error")
}
