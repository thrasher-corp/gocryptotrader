package binance

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestAdjustCrossMarginMaxLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.AdjustCrossMarginMaxLeverage(t.Context(), 0)
	require.ErrorIs(t, err, errMaxLeverageRequired, "AdjustCrossMarginMaxLeverage must reject a zero max leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AdjustCrossMarginMaxLeverage(t.Context(), 5)
	require.NoError(t, err, "AdjustCrossMarginMaxLeverage must not error")
	if mockTests {
		exp := &AdjustCrossMarginMaxLeverageResponse{
			Success: true,
		}
		assert.Equal(t, exp, result, "AdjustCrossMarginMaxLeverage should decode every field")
		return
	}
	assert.True(t, result.Success, "AdjustCrossMarginMaxLeverage should succeed")
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginAccountInfo(t.Context(), currency.Pairs{currency.NewBTCUSDT()})
	require.NoError(t, err, "GetIsolatedMarginAccountInfo must not error")
	if mockTests {
		exp := &IsolatedMarginAccountInfoResponse{
			Assets: []IsolatedMarginAccountAsset{
				{
					BaseAsset: IsolatedMarginAssetBalance{
						Asset:         currency.BTC,
						BorrowEnabled: true,
						Borrowed:      0.0012,
						Free:          0.015,
						Interest:      0.00000081,
						Locked:        0.002,
						NetAsset:      0.01579919,
						NetAssetOfBTC: 0.01579919,
						RepayEnabled:  true,
						TotalAsset:    0.017,
					},
					QuoteAsset: IsolatedMarginAssetBalance{
						Asset:         currency.USDT,
						BorrowEnabled: true,
						Borrowed:      150,
						Free:          820.5,
						Interest:      0.0125,
						Locked:        25,
						NetAsset:      695.4875,
						NetAssetOfBTC: 0.01041214,
						RepayEnabled:  true,
						TotalAsset:    845.5,
					},
					Symbol:            "BTCUSDT",
					IsolatedCreated:   true,
					Enabled:           true,
					MarginLevel:       5.8621315,
					MarginLevelStatus: "EXCESSIVE",
					MarginRatio:       10,
					IndexPrice:        66798.52,
					LiquidatePrice:    33012.73,
					LiquidateRate:     55.123,
					TradeEnabled:      true,
				},
			},
			TotalAssetOfBTC:     0.0296579,
			TotalLiabilityOfBTC: 0.00344637,
			TotalNetAssetOfBTC:  0.02621133,
		}
		assert.Equal(t, exp, result, "GetIsolatedMarginAccountInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetIsolatedMarginAccountInfo should return account info")
}

func TestEnableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := e.EnableIsolatedMarginAccount(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "EnableIsolatedMarginAccount must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.EnableIsolatedMarginAccount(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "EnableIsolatedMarginAccount must not error")
	if mockTests {
		exp := &IsolatedMarginAccountToggleResponse{
			Success: true,
			Symbol:  "BTCUSDT",
		}
		assert.Equal(t, exp, result, "EnableIsolatedMarginAccount should decode every field")
		return
	}
	assert.True(t, result.Success, "EnableIsolatedMarginAccount should succeed")
}

func TestDisableIsolatedMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := e.DisableIsolatedMarginAccount(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "DisableIsolatedMarginAccount must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.DisableIsolatedMarginAccount(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "DisableIsolatedMarginAccount must not error")
	if mockTests {
		exp := &IsolatedMarginAccountToggleResponse{
			Success: true,
			Symbol:  "BTCUSDT",
		}
		assert.Equal(t, exp, result, "DisableIsolatedMarginAccount should decode every field")
		return
	}
	assert.True(t, result.Success, "DisableIsolatedMarginAccount should succeed")
}

func TestGetBNBBurnStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBNBBurnStatus(t.Context())
	require.NoError(t, err, "GetBNBBurnStatus must not error")
	if mockTests {
		exp := &BNBBurnStatusResponse{
			SpotBNBBurn:     true,
			InterestBNBBurn: true,
		}
		assert.Equal(t, exp, result, "GetBNBBurnStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetBNBBurnStatus should return the burn status")
}

func TestGetSummaryOfMarginAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSummaryOfMarginAccount(t.Context())
	require.NoError(t, err, "GetSummaryOfMarginAccount must not error")
	if mockTests {
		exp := &MarginAccountSummaryResponse{
			NormalBar:           1.5,
			MarginCallBar:       1.3,
			ForceLiquidationBar: 1.1,
		}
		assert.Equal(t, exp, result, "GetSummaryOfMarginAccount should decode every field")
		return
	}
	assert.Positive(t, result.NormalBar.Float64(), "GetSummaryOfMarginAccount should return a normal bar")
}

func TestGetCrossOrIsolatedMarginCapitalFlow(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossOrIsolatedMarginCapitalFlow(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossOrIsolatedMarginCapitalFlow must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetCrossOrIsolatedMarginCapitalFlow(t.Context(), &MarginCapitalFlowRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCrossOrIsolatedMarginCapitalFlow must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossOrIsolatedMarginCapitalFlow(t.Context(), &MarginCapitalFlowRequest{
		Asset:     currency.ETH,
		Symbol:    currency.NewPair(currency.ETH, currency.USDT),
		Type:      "BORROW",
		StartTime: startTime,
		EndTime:   endTime,
		FromID:    10,
		Limit:     20,
	})
	require.NoError(t, err, "GetCrossOrIsolatedMarginCapitalFlow must not error")
	if mockTests {
		exp := []MarginCapitalFlow{
			{
				ID:            123456,
				TransactionID: 123123,
				Timestamp:     types.Time(time.UnixMilli(1744150000000)),
				Asset:         currency.ETH,
				Symbol:        "ETHUSDT",
				Type:          "BORROW",
				Amount:        1.5,
				Note:          "INSTITUTIONAL_LOAN_BORROW",
			},
		}
		assert.Equal(t, exp, result, "GetCrossOrIsolatedMarginCapitalFlow should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossOrIsolatedMarginCapitalFlow should return a capital flow list")
}

func TestGetCrossMarginAccountDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginAccountDetail(t.Context())
	require.NoError(t, err, "GetCrossMarginAccountDetail must not error")
	if mockTests {
		exp := &CrossMarginAccountDetailResponse{
			Created:                    true,
			BorrowEnabled:              true,
			MarginLevel:                11.64405625,
			CollateralMarginLevel:      3.2,
			TotalAssetOfBTC:            6.82728457,
			TotalLiabilityOfBTC:        0.5864556,
			TotalNetAssetOfBTC:         6.24082897,
			TotalCollateralValueInUSDT: 5.82728457,
			TradeEnabled:               true,
			TransferInEnabled:          true,
			TransferOutEnabled:         true,
			AccountType:                "MARGIN_1",
			UserAssets: []CrossMarginUserAsset{
				{
					Asset:    currency.BTC,
					Borrowed: 0.58633215,
					Free:     6.81728457,
					Interest: 0.00012345,
					Locked:   0.01,
					NetAsset: 6.24082897,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCrossMarginAccountDetail should decode every field")
		return
	}
	assert.NotEmpty(t, result.AccountType, "GetCrossMarginAccountDetail should return the account type")
}

// vipLevelZeroExchange serves path, checking each request asks for VIP level 0, which a zero value could not express
func vipLevelZeroExchange(t *testing.T, path string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path, "the request should be for the expected path")
		assert.Equal(t, "0", r.URL.Query().Get("vipLevel"), "the request should ask for VIP level 0")
		_, err := w.Write([]byte("[]"))
		assert.NoError(t, err, "Write should not error")
	})
}

func TestGetCrossMarginFeeData(t *testing.T) {
	t.Parallel()
	_, err := vipLevelZeroExchange(t, "/sapi/v1/margin/crossMarginData").GetCrossMarginFeeData(t.Context(), new(uint64(0)), currency.BTC)
	require.NoError(t, err, "GetCrossMarginFeeData must not error for VIP level 0")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginFeeData(t.Context(), nil, currency.BTC)
	require.NoError(t, err, "GetCrossMarginFeeData must not error")
	if mockTests {
		exp := []CrossMarginFeeData{
			{
				VIPLevel:       1,
				Coin:           currency.BTC,
				TransferIn:     true,
				Borrowable:     true,
				DailyInterest:  0.00026125,
				YearlyInterest: 0.0953,
				BorrowLimit:    180,
				MarginablePairs: []string{
					"BTCUSDT",
					"ETHBTC",
				},
			},
		}
		assert.Equal(t, exp, result, "GetCrossMarginFeeData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCrossMarginFeeData should return fee data")
}

func TestGetEnabledIsolatedMarginAccountLimit(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEnabledIsolatedMarginAccountLimit(t.Context())
	require.NoError(t, err, "GetEnabledIsolatedMarginAccountLimit must not error")
	if mockTests {
		exp := &IsolatedMarginAccountLimitResponse{
			EnabledAccount: 5,
			MaxAccount:     20,
		}
		assert.Equal(t, exp, result, "GetEnabledIsolatedMarginAccountLimit should decode every field")
		return
	}
	assert.Positive(t, result.MaxAccount, "GetEnabledIsolatedMarginAccountLimit should return the maximum")
}

func TestGetIsolatedMarginFeeData(t *testing.T) {
	t.Parallel()
	_, err := vipLevelZeroExchange(t, "/sapi/v1/margin/isolatedMarginData").GetIsolatedMarginFeeData(t.Context(), new(uint64(0)), currency.NewBTCUSDT())
	require.NoError(t, err, "GetIsolatedMarginFeeData must not error for VIP level 0")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginFeeData(t.Context(), new(uint64(1)), currency.NewBTCUSDT())
	require.NoError(t, err, "GetIsolatedMarginFeeData must not error")
	if mockTests {
		exp := []IsolatedMarginFeeData{
			{
				VIPLevel: 1,
				Symbol:   "BTCUSDT",
				Leverage: 10,
				Data: []IsolatedMarginFeeCoin{
					{
						Coin:          currency.BTC,
						DailyInterest: 0.00026125,
						BorrowLimit:   270,
					},
					{
						Coin:          currency.USDT,
						DailyInterest: 0.0003,
						BorrowLimit:   2000000,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetIsolatedMarginFeeData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetIsolatedMarginFeeData should return fee data")
}

func TestGetFutureHourlyInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFutureHourlyInterestRate(t.Context(), nil, true)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "GetFutureHourlyInterestRate must reject empty assets")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFutureHourlyInterestRate(t.Context(), []currency.Code{currency.BTC, currency.ETH}, true)
	require.NoError(t, err, "GetFutureHourlyInterestRate must not error")
	if mockTests {
		exp := []HourlyInterestRate{
			{
				Asset:                  currency.BTC,
				NextHourlyInterestRate: 0.00000571,
			},
			{
				Asset:                  currency.ETH,
				NextHourlyInterestRate: 0.00000578,
			},
		}
		assert.Equal(t, exp, result, "GetFutureHourlyInterestRate should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFutureHourlyInterestRate should return rates")
}

func TestGetUserMarginInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserMarginInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUserMarginInterestHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUserMarginInterestHistory(t.Context(), &UserMarginInterestHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserMarginInterestHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// Binance returns rawAsset for cross margin rows only and isolatedSymbol for isolated margin rows only
	for _, tc := range []struct {
		name           string
		isolatedSymbol currency.Pair
		exp            *UserMarginInterestHistoryResponse
	}{
		{
			name: "cross",
			exp: &UserMarginInterestHistoryResponse{
				Rows: []UserMarginInterestHistory{
					{
						TransactionID:       1352286576452864800,
						InterestAccuredTime: types.Time(time.UnixMilli(1744120800000)),
						Asset:               currency.USDT,
						RawAsset:            currency.USDT,
						Principal:           45.3313,
						Interest:            0.00024995,
						InterestRate:        0.00013233,
						Type:                "PERIODIC",
					},
				},
				Total: 1,
			},
		},
		{
			name:           "isolated",
			isolatedSymbol: currency.NewBTCUSDT(),
			exp: &UserMarginInterestHistoryResponse{
				Rows: []UserMarginInterestHistory{
					{
						TransactionID:       1352286576452864801,
						InterestAccuredTime: types.Time(time.UnixMilli(1744124400000)),
						Asset:               currency.USDT,
						Principal:           150,
						Interest:            0.00082708,
						InterestRate:        0.00000551,
						Type:                "ON_BORROW",
						IsolatedSymbol:      "BTCUSDT",
					},
				},
				Total: 1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUserMarginInterestHistory(t.Context(), &UserMarginInterestHistoryRequest{
				Asset:          currency.USDT,
				IsolatedSymbol: tc.isolatedSymbol,
				StartTime:      startTime,
				EndTime:        endTime,
				Current:        1,
				Size:           10,
			})
			require.NoError(t, err, "GetUserMarginInterestHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUserMarginInterestHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUserMarginInterestHistory should return a page")
		})
	}
}

func TestGetBorrowOrRepayRecordsInMarginAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowOrRepayRecordsInMarginAccount(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBorrowOrRepayRecordsInMarginAccount must reject a nil request")
	_, err = e.GetBorrowOrRepayRecordsInMarginAccount(t.Context(), &MarginBorrowRepayRecordsRequest{Asset: currency.LTC})
	require.ErrorIs(t, err, errLendingTypeRequired, "GetBorrowOrRepayRecordsInMarginAccount must reject an empty type")
	startTime, endTime := getTime()
	_, err = e.GetBorrowOrRepayRecordsInMarginAccount(t.Context(), &MarginBorrowRepayRecordsRequest{Type: "REPAY", StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetBorrowOrRepayRecordsInMarginAccount must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// Binance returns isolatedSymbol for isolated margin rows only
	for _, tc := range []struct {
		name string
		req  *MarginBorrowRepayRecordsRequest
		exp  *MarginBorrowRepayRecordsResponse
	}{
		{
			name: "cross",
			req:  &MarginBorrowRepayRecordsRequest{Type: "REPAY", Asset: currency.LTC, StartTime: startTime, EndTime: endTime, Current: 1, Size: 10},
			exp: &MarginBorrowRepayRecordsResponse{
				Rows: []MarginBorrowRepayRecord{
					{
						Type:          "BNB_AUTO_REPAY",
						Amount:        14,
						Asset:         currency.LTC,
						Interest:      0.01866667,
						Principal:     13.98133333,
						Status:        "CONFIRMED",
						Timestamp:     types.Time(time.UnixMilli(1744150000000)),
						TransactionID: 2970933056,
					},
				},
				Total: 1,
			},
		},
		{
			name: "isolated",
			req:  &MarginBorrowRepayRecordsRequest{Type: "REPAY", Asset: currency.BTC, IsolatedSymbol: currency.NewBTCUSDT(), TransactionID: 2970933057},
			exp: &MarginBorrowRepayRecordsResponse{
				Rows: []MarginBorrowRepayRecord{
					{
						Type:           "MANUAL",
						IsolatedSymbol: "BTCUSDT",
						Amount:         0.001,
						Asset:          currency.BTC,
						Interest:       0.00000012,
						Principal:      0.00099988,
						Status:         "CONFIRMED",
						Timestamp:      types.Time(time.UnixMilli(1744160000000)),
						TransactionID:  2970933057,
					},
				},
				Total: 1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetBorrowOrRepayRecordsInMarginAccount(t.Context(), tc.req)
			require.NoError(t, err, "GetBorrowOrRepayRecordsInMarginAccount must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetBorrowOrRepayRecordsInMarginAccount should decode every field")
				return
			}
			assert.NotNil(t, result, "GetBorrowOrRepayRecordsInMarginAccount should return a page")
		})
	}
}

func TestMarginAccountBorrowRepay(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountBorrowRepay(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MarginAccountBorrowRepay must reject a nil request")
	req := &MarginAccountBorrowRepayRequest{IsIsolated: true}
	_, err = e.MarginAccountBorrowRepay(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "MarginAccountBorrowRepay must reject an empty asset")
	req.Asset = currency.BTC
	_, err = e.MarginAccountBorrowRepay(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "MarginAccountBorrowRepay must reject isolated margin without a symbol")
	req.Symbol = currency.NewBTCUSDT()
	_, err = e.MarginAccountBorrowRepay(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "MarginAccountBorrowRepay must reject a zero amount")
	req.Amount = 0.001
	_, err = e.MarginAccountBorrowRepay(t.Context(), req)
	require.ErrorIs(t, err, errLendingTypeRequired, "MarginAccountBorrowRepay must reject an empty type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *MarginAccountBorrowRepayRequest
		exp  *MarginAccountBorrowRepayResponse
	}{
		{
			name: "cross",
			req:  &MarginAccountBorrowRepayRequest{Asset: currency.ETH, Amount: 0.1234, Type: "BORROW"},
			exp:  &MarginAccountBorrowRepayResponse{TransactionID: 100000001},
		},
		{
			name: "isolated",
			req:  &MarginAccountBorrowRepayRequest{Asset: currency.BTC, IsIsolated: true, Symbol: currency.NewBTCUSDT(), Amount: 0.001, Type: "BORROW"},
			exp:  &MarginAccountBorrowRepayResponse{TransactionID: 100000002},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.MarginAccountBorrowRepay(t.Context(), tc.req)
			require.NoError(t, err, "MarginAccountBorrowRepay must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "MarginAccountBorrowRepay should decode every field")
				return
			}
			assert.NotZero(t, result.TransactionID, "MarginAccountBorrowRepay should return a transaction ID")
		})
	}
}

func TestGetMarginInterestRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginInterestRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginInterestRateHistory must reject a nil request")
	_, err = e.GetMarginInterestRateHistory(t.Context(), &MarginInterestRateHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMarginInterestRateHistory must reject an empty asset")
	startTime, endTime := getTime()
	_, err = e.GetMarginInterestRateHistory(t.Context(), &MarginInterestRateHistoryRequest{Asset: currency.ETH, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginInterestRateHistory must reject a start after the end")
	_, err = vipLevelZeroExchange(t, "/sapi/v1/margin/interestRateHistory").GetMarginInterestRateHistory(t.Context(), &MarginInterestRateHistoryRequest{Asset: currency.ETH, VIPLevel: new(uint64(0))})
	require.NoError(t, err, "GetMarginInterestRateHistory must not error for VIP level 0")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginInterestRateHistory(t.Context(), &MarginInterestRateHistoryRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetMarginInterestRateHistory must not error")
	if mockTests {
		exp := []MarginInterestRate{
			{
				Asset:             currency.ETH,
				DailyInterestRate: 0.00025,
				Timestamp:         types.Time(time.UnixMilli(1744156800000)),
				VIPLevel:          1,
			},
		}
		assert.Equal(t, exp, result, "GetMarginInterestRateHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetMarginInterestRateHistory should return rates")
}

func TestGetMaxBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxBorrow(t.Context(), currency.EMPTYCODE, currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxBorrow must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxBorrow(t.Context(), currency.BTC, currency.NewBTCUSDT())
	require.NoError(t, err, "GetMaxBorrow must not error")
	if mockTests {
		exp := &MarginMaxBorrowableResponse{
			Amount:      1.69248805,
			BorrowLimit: 60,
		}
		assert.Equal(t, exp, result, "GetMaxBorrow should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMaxBorrow should return the maximum")
}

func TestGetCrossMarginCollateralRatio(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginCollateralRatio(t.Context())
	require.NoError(t, err, "GetCrossMarginCollateralRatio must not error")
	if mockTests {
		exp := []CrossMarginCollateralRatio{
			{
				Collaterals: []CrossMarginCollateral{
					{
						MaxUSDValue:  13000000,
						DiscountRate: 1,
					},
					{
						MinUSDValue:  13000000,
						MaxUSDValue:  20000000,
						DiscountRate: 0.975,
					},
				},
				AssetNames: []currency.Code{
					currency.NewCode("BNX"),
				},
			},
		}
		assert.Equal(t, exp, result, "GetCrossMarginCollateralRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCrossMarginCollateralRatio should return ratios")
}

func TestGetAllCrossMarginPairs(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllCrossMarginPairs(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetAllCrossMarginPairs must not error")
	if mockTests {
		exp := []CrossMarginPair{
			{
				Base:          currency.BTC,
				ID:            351637150141315840,
				IsBuyAllowed:  true,
				IsMarginTrade: true,
				IsSellAllowed: true,
				Quote:         currency.USDT,
				Symbol:        "BTCUSDT",
				DelistTime:    types.Time(time.Unix(1893456000, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetAllCrossMarginPairs should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAllCrossMarginPairs should return pairs")
}

func TestGetAllIsolatedMarginSymbols(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllIsolatedMarginSymbols(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetAllIsolatedMarginSymbols must not error")
	if mockTests {
		exp := []IsolatedMarginSymbol{
			{
				Base:          currency.BNB,
				IsBuyAllowed:  true,
				IsMarginTrade: true,
				IsSellAllowed: true,
				Quote:         currency.BTC,
				Symbol:        "BNBBTC",
			},
			{
				Base:          currency.BTC,
				IsBuyAllowed:  true,
				IsMarginTrade: true,
				IsSellAllowed: true,
				Quote:         currency.USDT,
				Symbol:        "BTCUSDT",
			},
		}
		assert.Equal(t, exp, result, "GetAllIsolatedMarginSymbols should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAllIsolatedMarginSymbols should return symbols")
}

func TestGetAllMarginAssets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllMarginAssets(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAllMarginAssets must not error")
	if mockTests {
		exp := []MarginAssetInfo{
			{
				AssetFullName:  "Bitcoin",
				AssetName:      currency.BTC,
				IsBorrowable:   true,
				IsMortgageable: true,
				UserMinBorrow:  0.00001,
				UserMinRepay:   0.00001,
				DelistTime:     types.Time(time.Unix(1893456000, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetAllMarginAssets should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAllMarginAssets should return assets")
}

func TestGetTokensOrSymbolsDelistSchedule(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTokensOrSymbolsDelistSchedule(t.Context())
	require.NoError(t, err, "GetTokensOrSymbolsDelistSchedule must not error")
	if mockTests {
		exp := []MarginDelistSchedule{
			{
				DelistTime: types.Time(time.UnixMilli(1759276800000)),
				CrossMarginAssets: []currency.Code{
					currency.NewCode("ANT"),
					currency.NewCode("MDT"),
				},
				IsolatedMarginSymbols: []string{
					"ANTUSDT",
					"MDTUSDT",
				},
			},
		}
		assert.Equal(t, exp, result, "GetTokensOrSymbolsDelistSchedule should decode every field")
		return
	}
	assert.NotNil(t, result, "GetTokensOrSymbolsDelistSchedule should return a schedule")
}

func TestGetIsolatedMarginTierData(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedMarginTierData(t.Context(), currency.EMPTYPAIR, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetIsolatedMarginTierData must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetIsolatedMarginTierData(t.Context(), currency.NewBTCUSDT(), 10)
	require.NoError(t, err, "GetIsolatedMarginTierData must not error")
	if mockTests {
		exp := []IsolatedMarginTier{
			{
				Symbol:                  "BTCUSDT",
				Tier:                    10,
				EffectiveMultiple:       2,
				InitialRiskRatio:        2,
				LiquidationRiskRatio:    1.9,
				BaseAssetMaxBorrowable:  400,
				QuoteAssetMaxBorrowable: 30000000,
			},
		}
		assert.Equal(t, exp, result, "GetIsolatedMarginTierData should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetIsolatedMarginTierData should return tiers")
}

func TestGetMarginAvailableInventory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAvailableInventory(t.Context(), "")
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "GetMarginAvailableInventory must reject an empty margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAvailableInventory(t.Context(), "ISOLATED")
	require.NoError(t, err, "GetMarginAvailableInventory must not error")
	if mockTests {
		exp := &MarginAvailableInventoryResponse{
			Assets: map[currency.Code]types.Number{
				currency.BTC:  2.5,
				currency.USDT: 1500000,
			},
			UpdateTime: types.Time(time.Unix(1744190254, 0)),
		}
		assert.Equal(t, exp, result, "GetMarginAvailableInventory should decode every field")
		return
	}
	assert.NotEmpty(t, result.Assets, "GetMarginAvailableInventory should return assets")
}

func TestGetMarginPriceIndex(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginPriceIndex(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarginPriceIndex must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginPriceIndex(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetMarginPriceIndex must not error")
	if mockTests {
		exp := &MarginPriceIndexResponse{
			CalculationTime: types.Time(time.UnixMilli(1744190254000)),
			Price:           76312.21,
			Symbol:          "BTCUSDT",
		}
		assert.Equal(t, exp, result, "GetMarginPriceIndex should decode every field")
		return
	}
	assert.Positive(t, result.Price.Float64(), "GetMarginPriceIndex should return a price")
}

func TestGetForceLiquidationRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetForceLiquidationRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetForceLiquidationRecord must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetForceLiquidationRecord(t.Context(), &ForceLiquidationRecordRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetForceLiquidationRecord must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetForceLiquidationRecord(t.Context(), &ForceLiquidationRecordRequest{StartTime: startTime, EndTime: endTime, IsolatedSymbol: currency.NewBTCUSDT(), Size: 12})
	require.NoError(t, err, "GetForceLiquidationRecord must not error")
	if mockTests {
		exp := &ForceLiquidationRecordResponse{
			Rows: []ForceLiquidationRecord{
				{
					AveragePrice:     76000.5,
					ExecutedQuantity: 0.005,
					OrderID:          180015097,
					Price:            75900,
					Quantity:         0.005,
					Side:             "SELL",
					Symbol:           "BTCUSDT",
					TimeInForce:      "IOC",
					IsIsolated:       true,
					UpdatedTime:      types.Time(time.UnixMilli(1744150000000)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetForceLiquidationRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetForceLiquidationRecord should return a page")
}

func TestGetSmallLiabilityExchangeCoinList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSmallLiabilityExchangeCoinList(t.Context())
	require.NoError(t, err, "GetSmallLiabilityExchangeCoinList must not error")
	if mockTests {
		exp := []SmallLiabilityExchangeCoin{
			{
				Asset:             currency.ETH,
				Interest:          0.00083334,
				Principal:         0.001,
				LiabilityAsset:    currency.USDT,
				LiabilityQuantity: 0.3552,
			},
		}
		assert.Equal(t, exp, result, "GetSmallLiabilityExchangeCoinList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSmallLiabilityExchangeCoinList should return a list")
}

func TestMarginSmallLiabilityExchange(t *testing.T) {
	t.Parallel()
	err := e.MarginSmallLiabilityExchange(t.Context(), nil)
	require.ErrorIs(t, err, errEmptyCurrencyCodes, "MarginSmallLiabilityExchange must reject empty asset names")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.MarginSmallLiabilityExchange(t.Context(), []currency.Code{currency.BTC, currency.ETH})
	assert.NoError(t, err, "MarginSmallLiabilityExchange should not error")
}

func TestGetSmallLiabilityExchangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSmallLiabilityExchangeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSmallLiabilityExchangeHistory must reject a nil request")
	_, err = e.GetSmallLiabilityExchangeHistory(t.Context(), &SmallLiabilityExchangeHistoryRequest{Size: 10})
	require.ErrorIs(t, err, errPageNumberRequired, "GetSmallLiabilityExchangeHistory must reject a zero page number")
	_, err = e.GetSmallLiabilityExchangeHistory(t.Context(), &SmallLiabilityExchangeHistoryRequest{Current: 1})
	require.ErrorIs(t, err, errPageSizeRequired, "GetSmallLiabilityExchangeHistory must reject a zero page size")
	startTime, endTime := getTime()
	_, err = e.GetSmallLiabilityExchangeHistory(t.Context(), &SmallLiabilityExchangeHistoryRequest{Current: 1, Size: 10, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSmallLiabilityExchangeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSmallLiabilityExchangeHistory(t.Context(), &SmallLiabilityExchangeHistoryRequest{Current: 1, Size: 10, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetSmallLiabilityExchangeHistory must not error")
	if mockTests {
		exp := &SmallLiabilityExchangeHistoryResponse{
			Total: 1,
			Rows: []SmallLiabilityExchangeRecord{
				{
					Asset:        currency.ETH,
					Amount:       0.00083434,
					TargetAsset:  currency.USDT,
					TargetAmount: 1.37576819,
					BusinessType: "EXCHANGE_SMALL_LIABILITY",
					Timestamp:    types.Time(time.UnixMilli(1744150000000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetSmallLiabilityExchangeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSmallLiabilityExchangeHistory should return a page")
}

func TestGetMarginAccountsOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountsOpenOrders(t.Context(), currency.EMPTYPAIR, true)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarginAccountsOpenOrders must reject isolated margin without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountsOpenOrders(t.Context(), currency.NewBTCUSDT(), true)
	require.NoError(t, err, "GetMarginAccountsOpenOrders must not error")
	if mockTests {
		exp := []MarginTradeOrderResponse{
			{
				ClientOrderID:            "qhcZw71gAkCCTv0t0k8LUK",
				CummulativeQuoteQuantity: 7.6005,
				ExecutedQuantity:         0.0001,
				IcebergQuantity:          0.0001,
				IsWorking:                true,
				OrderID:                  211842552,
				OriginalQuantity:         0.0003,
				Price:                    76005,
				Side:                     "SELL",
				Status:                   "PARTIALLY_FILLED",
				StopPrice:                76000,
				Symbol:                   "BTCUSDT",
				IsIsolated:               true,
				Time:                     types.Time(time.UnixMilli(1744150000000)),
				TimeInForce:              "GTC",
				Type:                     "TAKE_PROFIT_LIMIT",
				SelfTradePreventionMode:  "EXPIRE_MAKER",
				UpdateTime:               types.Time(time.UnixMilli(1744150100000)),
			},
		}
		assert.Equal(t, exp, result, "GetMarginAccountsOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountsOpenOrders should return a list")
}

func TestCancelAllOpenMarginAccountOrdersOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenMarginAccountOrdersOnSymbol(t.Context(), currency.EMPTYPAIR, true)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOpenMarginAccountOrdersOnSymbol must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllOpenMarginAccountOrdersOnSymbol(t.Context(), currency.NewBTCUSDT(), true)
	require.NoError(t, err, "CancelAllOpenMarginAccountOrdersOnSymbol must not error")
	if mockTests {
		exp := []MarginCancelledOrder{
			{
				Symbol:                   "BTCUSDT",
				IsIsolated:               true,
				OriginalClientOrderID:    "E6APeyTJvkMvLMYMqu1KQ4",
				OrderID:                  11,
				OrderListID:              -1,
				ClientOrderID:            "pXLV6Hz6mprAcVYpVMTGgx",
				Price:                    70000,
				OriginalQuantity:         0.001,
				ExecutedQuantity:         0.0002,
				CummulativeQuoteQuantity: 14,
				Status:                   "CANCELED",
				TimeInForce:              "GTC",
				Type:                     "LIMIT",
				Side:                     "BUY",
				SelfTradePreventionMode:  "NONE",
			},
			{
				Symbol:            "BTCUSDT",
				IsIsolated:        true,
				OrderListID:       1929,
				ContingencyType:   "OCO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "2inzWQdDvZLHbbAmAozX2N",
				TransactionTime:   types.Time(time.UnixMilli(1744150000000)),
				Orders: []MarginOrderListOrder{
					{
						Symbol:        "BTCUSDT",
						OrderID:       20,
						ClientOrderID: "CwOOIPHSmYywx6jZX77TdL",
					},
					{
						Symbol:        "BTCUSDT",
						OrderID:       21,
						ClientOrderID: "461cPg51vQjV3zIMOXNz39",
					},
				},
				OrderReports: []MarginCancelledOrderReport{
					{
						Symbol:                   "BTCUSDT",
						OriginalClientOrderID:    "CwOOIPHSmYywx6jZX77TdL",
						OrderID:                  20,
						OrderListID:              1929,
						ClientOrderID:            "g6gwSwpWa6GQVmZ7iQqfWS",
						Price:                    80000,
						OriginalQuantity:         0.001,
						ExecutedQuantity:         0.0001,
						CummulativeQuoteQuantity: 8,
						Status:                   "CANCELED",
						TimeInForce:              "GTC",
						Type:                     "STOP_LOSS_LIMIT",
						Side:                     "BUY",
						StopPrice:                79900,
						IcebergQuantity:          0.0005,
					},
					{
						Symbol:                "BTCUSDT",
						OriginalClientOrderID: "461cPg51vQjV3zIMOXNz39",
						OrderID:               21,
						OrderListID:           1929,
						ClientOrderID:         "g6gwSwpWa6GQVmZ7iQqfWS",
						Price:                 70000,
						OriginalQuantity:      0.001,
						Status:                "CANCELED",
						TimeInForce:           "GTC",
						Type:                  "LIMIT_MAKER",
						Side:                  "BUY",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "CancelAllOpenMarginAccountOrdersOnSymbol should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllOpenMarginAccountOrdersOnSymbol should return the cancelled orders")
}

func TestGetMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountOCOOrder(t.Context(), currency.NewBTCUSDT(), true, 0, "12345")
	require.NoError(t, err, "GetMarginAccountOCOOrder must not error")
	if mockTests {
		exp := &MarginOrderListResponse{
			OrderListID:       27,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "12345",
			TransactionTime:   types.Time(time.UnixMilli(1744150000000)),
			Symbol:            "BTCUSDT",
			IsIsolated:        true,
			Orders: []MarginOrderListOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       4,
					ClientOrderID: "qD1gy3kc3Gx0rihm9Y3xwS",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       5,
					ClientOrderID: "ARzZ9I00CPM8i3NhmU9Ega",
				},
			},
		}
		assert.Equal(t, exp, result, "GetMarginAccountOCOOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountOCOOrder should return an order list")
}

func TestCancelMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginAccountOCOOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelMarginAccountOCOOrder must reject a nil request")
	_, err = e.CancelMarginAccountOCOOrder(t.Context(), &MarginOCOCancelRequest{ListClientOrderID: "12345678"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelMarginAccountOCOOrder must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginAccountOCOOrder(t.Context(), &MarginOCOCancelRequest{Symbol: currency.NewBTCUSDT(), IsIsolated: true, ListClientOrderID: "12345678"})
	require.NoError(t, err, "CancelMarginAccountOCOOrder must not error")
	if mockTests {
		exp := &MarginOCOCancelResponse{
			OrderListID:       1,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "12345678",
			TransactionTime:   types.Time(time.UnixMilli(1744150000000)),
			Symbol:            "BTCUSDT",
			IsIsolated:        true,
			Orders: []MarginOrderListOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       2,
					ClientOrderID: "pO9ufTiFGg3nw2fOdgeOXa",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       3,
					ClientOrderID: "TXOvglzXuaubXAaENpaRCB",
				},
			},
			OrderReports: []MarginOCOCancelReport{
				{
					Symbol:                   "BTCUSDT",
					OriginalClientOrderID:    "pO9ufTiFGg3nw2fOdgeOXa",
					OrderID:                  2,
					OrderListID:              1,
					ClientOrderID:            "unfWT8ig8i0uj6lPuYLez6",
					Price:                    60000,
					OriginalQuantity:         0.001,
					ExecutedQuantity:         0.0002,
					CummulativeQuoteQuantity: 12,
					Status:                   "CANCELED",
					TimeInForce:              "GTC",
					Type:                     "STOP_LOSS_LIMIT",
					Side:                     "SELL",
					StopPrice:                60100,
					SelfTradePreventionMode:  "NONE",
				},
				{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "TXOvglzXuaubXAaENpaRCB",
					OrderID:                 3,
					OrderListID:             1,
					ClientOrderID:           "unfWT8ig8i0uj6lPuYLez6",
					Price:                   70000,
					OriginalQuantity:        0.001,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "LIMIT_MAKER",
					Side:                    "SELL",
					SelfTradePreventionMode: "NONE",
				},
			},
		}
		assert.Equal(t, exp, result, "CancelMarginAccountOCOOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMarginAccountOCOOrder should return the cancelled order list")
}

func TestGetMarginAccountsOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountsOrder(t.Context(), currency.EMPTYPAIR, true, 112233424, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarginAccountsOrder must reject an empty symbol")
	_, err = e.GetMarginAccountsOrder(t.Context(), currency.NewBTCUSDT(), true, 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMarginAccountsOrder must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountsOrder(t.Context(), currency.NewBTCUSDT(), true, 112233424, "")
	require.NoError(t, err, "GetMarginAccountsOrder must not error")
	if mockTests {
		exp := &MarginTradeOrderResponse{
			ClientOrderID:            "ZwfQzuDIGpceVhKW5DvCmO",
			CummulativeQuoteQuantity: 15.16,
			ExecutedQuantity:         0.0002,
			IcebergQuantity:          0.0001,
			IsWorking:                true,
			OrderID:                  112233424,
			OriginalQuantity:         0.0005,
			Price:                    75800,
			Side:                     "SELL",
			Status:                   "PARTIALLY_FILLED",
			StopPrice:                75900,
			Symbol:                   "BTCUSDT",
			IsIsolated:               true,
			Time:                     types.Time(time.UnixMilli(1744150000000)),
			TimeInForce:              "GTC",
			Type:                     "STOP_LOSS_LIMIT",
			SelfTradePreventionMode:  "EXPIRE_TAKER",
			UpdateTime:               types.Time(time.UnixMilli(1744150500000)),
		}
		assert.Equal(t, exp, result, "GetMarginAccountsOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountsOrder should return the order")
}

func TestPostMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PostMarginAccountOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PostMarginAccountOrder must reject a nil request")
	req := &MarginAccountOrderRequest{}
	_, err = e.PostMarginAccountOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PostMarginAccountOrder must reject an empty symbol")
	req.Symbol = currency.NewBTCUSDT()
	_, err = e.PostMarginAccountOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PostMarginAccountOrder must reject an empty side")
	req.Side = order.Buy.String()
	_, err = e.PostMarginAccountOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "PostMarginAccountOrder must reject an empty order type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PostMarginAccountOrder(t.Context(), &MarginAccountOrderRequest{
		Symbol:                  currency.NewBTCUSDT(),
		IsIsolated:              true,
		Side:                    order.Buy.String(),
		OrderType:               order.Limit.String(),
		Quantity:                0.001,
		Price:                   65982.53,
		NewClientOrderID:        "E156O3KP4gOif65bjuUK5V",
		NewOrderResponseType:    "FULL",
		SideEffectType:          "MARGIN_BUY",
		TimeInForce:             order.GoodTillCancel.String(),
		SelfTradePreventionMode: "EXPIRE_MAKER",
		AutoRepayAtCancel:       new(bool),
	})
	require.NoError(t, err, "PostMarginAccountOrder must not error")
	if mockTests {
		exp := &MarginAccountNewOrderResponse{
			Symbol:                   "BTCUSDT",
			OrderID:                  26769564559,
			ClientOrderID:            "E156O3KP4gOif65bjuUK5V",
			IsIsolated:               true,
			TransactTime:             types.Time(time.UnixMilli(1744150000000)),
			Price:                    65982.53,
			OriginalQuantity:         0.001,
			ExecutedQuantity:         0.001,
			CummulativeQuoteQuantity: 65.98253,
			Status:                   "FILLED",
			TimeInForce:              "GTC",
			Type:                     "LIMIT",
			Side:                     "BUY",
			SelfTradePreventionMode:  "EXPIRE_MAKER",
			MarginBuyBorrowAmount:    5,
			MarginBuyBorrowAsset:     currency.USDT,
			Fills: []MarginOrderFill{
				{
					Price:           65982.53,
					Quantity:        0.001,
					Commission:      0.000001,
					CommissionAsset: currency.BTC,
					TradeID:         3570680726,
				},
			},
		}
		assert.Equal(t, exp, result, "PostMarginAccountOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "PostMarginAccountOrder should return an order ID")
}

func TestCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginAccountOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelMarginAccountOrder must reject a nil request")
	_, err = e.CancelMarginAccountOrder(t.Context(), &MarginOrderCancelRequest{OrderID: 12314234})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelMarginAccountOrder must reject an empty symbol")
	_, err = e.CancelMarginAccountOrder(t.Context(), &MarginOrderCancelRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelMarginAccountOrder must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMarginAccountOrder(t.Context(), &MarginOrderCancelRequest{Symbol: currency.NewBTCUSDT(), IsIsolated: true, OrderID: 12314234})
	require.NoError(t, err, "CancelMarginAccountOrder must not error")
	if mockTests {
		exp := &MarginOrderCancelResponse{
			Symbol:                   "BTCUSDT",
			OrderID:                  12314234,
			OriginalClientOrderID:    "myOrder1",
			ClientOrderID:            "cancelMyOrder1",
			Price:                    76000,
			OriginalQuantity:         0.001,
			ExecutedQuantity:         0.0008,
			CummulativeQuoteQuantity: 60.8,
			Status:                   "CANCELED",
			TimeInForce:              "GTC",
			Type:                     "LIMIT",
			Side:                     "SELL",
			IsIsolated:               true,
		}
		assert.Equal(t, exp, result, "CancelMarginAccountOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelMarginAccountOrder should return the cancelled order")
}

func TestNewMarginAccountOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewMarginAccountOCOOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewMarginAccountOCOOrder must reject a nil request")
	req := &MarginOCOOrderRequest{IsIsolated: true}
	_, err = e.NewMarginAccountOCOOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewMarginAccountOCOOrder must reject an empty symbol")
	req.Symbol = currency.NewBTCUSDT()
	_, err = e.NewMarginAccountOCOOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewMarginAccountOCOOrder must reject an empty side")
	req.Side = order.Sell.String()
	_, err = e.NewMarginAccountOCOOrder(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "NewMarginAccountOCOOrder must reject a zero quantity")
	req.Quantity = 0.001
	_, err = e.NewMarginAccountOCOOrder(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "NewMarginAccountOCOOrder must reject a zero price")
	req.Price = 70000
	_, err = e.NewMarginAccountOCOOrder(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "NewMarginAccountOCOOrder must reject a zero stop price")
	req.StopPrice = 60000
	req.StopLimitPrice = 59900
	_, err = e.NewMarginAccountOCOOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrInvalidTimeInForce, "NewMarginAccountOCOOrder must reject a stop limit price without a time in force")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.NewMarginAccountOCOOrder(t.Context(), &MarginOCOOrderRequest{
		Symbol:               currency.NewBTCUSDT(),
		IsIsolated:           true,
		ListClientOrderID:    "JYVpp3F0f5CAG15DhtrqLp",
		Side:                 order.Sell.String(),
		Quantity:             0.001,
		Price:                70000,
		StopPrice:            60000,
		StopLimitPrice:       59900,
		StopLimitTimeInForce: order.GoodTillCancel.String(),
		NewOrderResponseType: "FULL",
		SideEffectType:       "MARGIN_BUY",
	})
	require.NoError(t, err, "NewMarginAccountOCOOrder must not error")
	if mockTests {
		exp := &MarginOCOOrderResponse{
			OrderListID:           2,
			ContingencyType:       "OCO",
			ListStatusType:        "EXEC_STARTED",
			ListOrderStatus:       "EXECUTING",
			ListClientOrderID:     "JYVpp3F0f5CAG15DhtrqLp",
			TransactionTime:       types.Time(time.UnixMilli(1744150000000)),
			Symbol:                "BTCUSDT",
			MarginBuyBorrowAmount: 0.001,
			MarginBuyBorrowAsset:  currency.BTC,
			IsIsolated:            true,
			Orders: []MarginOrderListOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       4,
					ClientOrderID: "Kk7sqHb9J6mJWTMDVW7Vos",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       5,
					ClientOrderID: "xTXKaGYd4bluPVp78IVRvl",
				},
			},
			OrderReports: []MarginOCOOrderReport{
				{
					Symbol:                  "BTCUSDT",
					OrderID:                 4,
					OrderListID:             2,
					ClientOrderID:           "Kk7sqHb9J6mJWTMDVW7Vos",
					TransactTime:            types.Time(time.UnixMilli(1744150000000)),
					Price:                   59900,
					OriginalQuantity:        0.001,
					Status:                  "NEW",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					StopPrice:               60000,
					SelfTradePreventionMode: "NONE",
				},
				{
					Symbol:                   "BTCUSDT",
					OrderID:                  5,
					OrderListID:              2,
					ClientOrderID:            "xTXKaGYd4bluPVp78IVRvl",
					TransactTime:             types.Time(time.UnixMilli(1744150000000)),
					Price:                    70000,
					OriginalQuantity:         0.001,
					ExecutedQuantity:         0.0001,
					CummulativeQuoteQuantity: 7,
					Status:                   "PARTIALLY_FILLED",
					TimeInForce:              "GTC",
					Type:                     "LIMIT_MAKER",
					Side:                     "SELL",
					SelfTradePreventionMode:  "NONE",
				},
			},
		}
		assert.Equal(t, exp, result, "NewMarginAccountOCOOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "NewMarginAccountOCOOrder should return the order list")
}

func TestMarginManualLiquidation(t *testing.T) {
	t.Parallel()
	_, err := e.MarginManualLiquidation(t.Context(), "", currency.EMPTYPAIR)
	require.ErrorIs(t, err, margin.ErrInvalidMarginType, "MarginManualLiquidation must reject an empty margin type")
	_, err = e.MarginManualLiquidation(t.Context(), "ISOLATED", currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "MarginManualLiquidation must reject isolated margin without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		marginType string
		symbol     currency.Pair
		exp        *MarginManualLiquidationResponse
	}{
		{
			marginType: "MARGIN",
			exp: &MarginManualLiquidationResponse{
				Asset:             currency.ETH,
				Interest:          0.00083334,
				Principal:         0.001,
				LiabilityAsset:    currency.USDT,
				LiabilityQuantity: 0.3552,
			},
		},
		{
			marginType: "ISOLATED",
			symbol:     currency.NewBTCUSDT(),
			exp: &MarginManualLiquidationResponse{
				Asset:             currency.BTC,
				Interest:          0.00000123,
				Principal:         0.0001,
				LiabilityAsset:    currency.USDT,
				LiabilityQuantity: 7.6,
			},
		},
	} {
		t.Run(tc.marginType, func(t *testing.T) {
			t.Parallel()
			result, err := e.MarginManualLiquidation(t.Context(), tc.marginType, tc.symbol)
			require.NoError(t, err, "MarginManualLiquidation must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "MarginManualLiquidation should decode every field")
				return
			}
			assert.NotNil(t, result, "MarginManualLiquidation should return the repaid liability")
		})
	}
}

func TestGetCurrentMarginOrderCountUsage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentMarginOrderCountUsage(t.Context(), currency.NewBTCUSDT(), true)
	require.NoError(t, err, "GetCurrentMarginOrderCountUsage must not error")
	if mockTests {
		exp := []MarginOrderCountUsage{
			{
				RateLimitType:  "ORDERS",
				Interval:       "SECOND",
				IntervalNumber: 10,
				Limit:          10000,
				Count:          2,
			},
			{
				RateLimitType:  "ORDERS",
				Interval:       "DAY",
				IntervalNumber: 1,
				Limit:          20000,
				Count:          15,
			},
		}
		assert.Equal(t, exp, result, "GetCurrentMarginOrderCountUsage should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCurrentMarginOrderCountUsage should return usage")
}

func TestGetMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountAllOCO(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginAccountAllOCO must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetMarginAccountAllOCO(t.Context(), &MarginAccountAllOCORequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginAccountAllOCO must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountAllOCO(t.Context(), &MarginAccountAllOCORequest{IsIsolated: true, Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: endTime, Limit: 12})
	require.NoError(t, err, "GetMarginAccountAllOCO must not error")
	if mockTests {
		exp := []MarginOrderListResponse{
			{
				OrderListID:       29,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "amEEAXryFzFwYF1FeRpUoZ",
				TransactionTime:   types.Time(time.UnixMilli(1744150000000)),
				Symbol:            "BTCUSDT",
				IsIsolated:        true,
				Orders: []MarginOrderListOrder{
					{
						Symbol:        "BTCUSDT",
						OrderID:       4,
						ClientOrderID: "oD7aesZqjEGlZrbtRpy5zB",
					},
					{
						Symbol:        "BTCUSDT",
						OrderID:       5,
						ClientOrderID: "Jr1h6xirOxgeJOUuYQS7V3",
					},
				},
			},
			{
				OrderListID:       30,
				ContingencyType:   "OTO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "q2Ve2RiVBk5DxGkcVQJTDK",
				TransactionTime:   types.Time(time.UnixMilli(1744160000000)),
				Symbol:            "BTCUSDT",
				IsIsolated:        true,
				Orders: []MarginOrderListOrder{
					{
						Symbol:        "BTCUSDT",
						OrderID:       6,
						ClientOrderID: "oCQCzKJvlUm89ShKDQWRle",
					},
					{
						Symbol:        "BTCUSDT",
						OrderID:       7,
						ClientOrderID: "UK3Fuz8RkZjzaePPvbOm4i",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetMarginAccountAllOCO should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountAllOCO should return a list")
}

func TestGetMarginAccountAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginAccountAllOrders must reject a nil request")
	_, err = e.GetMarginAccountAllOrders(t.Context(), &MarginAccountAllOrdersRequest{IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarginAccountAllOrders must reject an empty symbol")
	// Binance requires windows shorter than 24 hours
	_, endTime := getTime()
	startTime := endTime.Add(-12 * time.Hour)
	_, err = e.GetMarginAccountAllOrders(t.Context(), &MarginAccountAllOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginAccountAllOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountAllOrders(t.Context(), &MarginAccountAllOrdersRequest{Symbol: currency.NewBTCUSDT(), IsIsolated: true, OrderID: 1, StartTime: startTime, EndTime: endTime, Limit: 20})
	require.NoError(t, err, "GetMarginAccountAllOrders must not error")
	if mockTests {
		exp := []MarginTradeOrderResponse{
			{
				ClientOrderID:            "D2KDy4DIeS56PvkM13f8cP",
				CummulativeQuoteQuantity: 38,
				ExecutedQuantity:         0.0005,
				IcebergQuantity:          0.0001,
				IsWorking:                true,
				OrderID:                  41295,
				OriginalQuantity:         0.0005,
				Price:                    76000,
				Side:                     "SELL",
				Status:                   "FILLED",
				StopPrice:                75900,
				Symbol:                   "BTCUSDT",
				IsIsolated:               true,
				Time:                     types.Time(time.UnixMilli(1744150000000)),
				TimeInForce:              "GTC",
				Type:                     "TAKE_PROFIT_LIMIT",
				SelfTradePreventionMode:  "NONE",
				UpdateTime:               types.Time(time.UnixMilli(1744150342148)),
			},
		}
		assert.Equal(t, exp, result, "GetMarginAccountAllOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountAllOrders should return a list")
}

func TestGetMarginAccountsOpenOCOOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountsOpenOCOOrder(t.Context(), currency.NewBTCUSDT(), true)
	require.NoError(t, err, "GetMarginAccountsOpenOCOOrder must not error")
	if mockTests {
		exp := []MarginOrderListResponse{
			{
				OrderListID:       31,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "wuB13fmulKj3YjdqWEcsnp",
				TransactionTime:   types.Time(time.UnixMilli(1744150000000)),
				Symbol:            "BTCUSDT",
				IsIsolated:        true,
				Orders: []MarginOrderListOrder{
					{
						Symbol:        "BTCUSDT",
						OrderID:       4,
						ClientOrderID: "r3EH2N76dHfLoSZWIUw1bT",
					},
					{
						Symbol:        "BTCUSDT",
						OrderID:       5,
						ClientOrderID: "Cv1SnyPD3qhqpbjpYEHbd2",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetMarginAccountsOpenOCOOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountsOpenOCOOrder should return a list")
}

func TestGetMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginAccountTradeList must reject a nil request")
	_, err = e.GetMarginAccountTradeList(t.Context(), &MarginAccountTradeListRequest{IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarginAccountTradeList must reject an empty symbol")
	// Binance requires windows shorter than 24 hours
	_, endTime := getTime()
	startTime := endTime.Add(-12 * time.Hour)
	_, err = e.GetMarginAccountTradeList(t.Context(), &MarginAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginAccountTradeList must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountTradeList(t.Context(), &MarginAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), IsIsolated: true, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetMarginAccountTradeList must not error")
	if mockTests {
		exp := []MarginAccountTrade{
			{
				Commission:      0.0000005,
				CommissionAsset: currency.BTC,
				ID:              34,
				IsBestMatch:     true,
				IsBuyer:         true,
				IsMaker:         true,
				OrderID:         39324,
				Price:           76000,
				Quantity:        0.0005,
				Symbol:          "BTCUSDT",
				IsIsolated:      true,
				Time:            types.Time(time.UnixMilli(1744150000000)),
			},
		}
		assert.Equal(t, exp, result, "GetMarginAccountTradeList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountTradeList should return a list")
}

func TestGetCrossMarginTransferHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossMarginTransferHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCrossMarginTransferHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetCrossMarginTransferHistory(t.Context(), &CrossMarginTransferHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCrossMarginTransferHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCrossMarginTransferHistory(t.Context(), &CrossMarginTransferHistoryRequest{
		Asset:          currency.ETH,
		Type:           "ROLL_IN",
		StartTime:      startTime,
		EndTime:        endTime,
		Current:        10,
		Size:           30,
		IsolatedSymbol: currency.NewPair(currency.ETH, currency.USDT),
	})
	require.NoError(t, err, "GetCrossMarginTransferHistory must not error")
	if mockTests {
		exp := &CrossMarginTransferHistoryResponse{
			Rows: []CrossMarginTransfer{
				{
					Amount:        0.1,
					Asset:         currency.ETH,
					Status:        "CONFIRMED",
					Timestamp:     types.Time(time.UnixMilli(1744150000000)),
					TransactionID: 5240372201,
					Type:          "ROLL_IN",
					TransferFrom:  "SPOT",
					TransferTo:    "ISOLATED_MARGIN",
					ToSymbol:      "ETHUSDT",
				},
				{
					Amount:        0.05,
					Asset:         currency.ETH,
					Status:        "CONFIRMED",
					Timestamp:     types.Time(time.UnixMilli(1744140000000)),
					TransactionID: 5240372100,
					Type:          "ROLL_IN",
					TransferFrom:  "ISOLATED_MARGIN",
					TransferTo:    "ISOLATED_MARGIN",
					FromSymbol:    "ETHBTC",
					ToSymbol:      "ETHUSDT",
				},
			},
			Total: 272,
		}
		assert.Equal(t, exp, result, "GetCrossMarginTransferHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCrossMarginTransferHistory should return a page")
}

func TestGetMaxTransferOutAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxTransferOutAmount(t.Context(), currency.EMPTYCODE, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMaxTransferOutAmount must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaxTransferOutAmount(t.Context(), currency.ETH, currency.EMPTYPAIR)
	require.NoError(t, err, "GetMaxTransferOutAmount must not error")
	if mockTests {
		exp := &MaxTransferOutAmountResponse{
			Amount: 3.59498107,
		}
		assert.Equal(t, exp, result, "GetMaxTransferOutAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMaxTransferOutAmount should return the amount")
}

func TestCreateMarginListenToken(t *testing.T) {
	t.Parallel()
	_, err := e.CreateMarginListenToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateMarginListenToken must reject a nil request")
	_, err = e.CreateMarginListenToken(t.Context(), &MarginListenTokenRequest{IsIsolated: true})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CreateMarginListenToken must reject an isolated request without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CreateMarginListenToken(t.Context(), &MarginListenTokenRequest{})
	require.NoError(t, err, "CreateMarginListenToken must not error for the cross margin account")
	if mockTests {
		exp := &MarginListenTokenResponse{
			Token:          "6xXxePXwZRjVSHKhzUCCGnmN3fkvMTXru+pYJS8RwijXk9Vcyr3rkwfVOTcP2OkONqciYA",
			ExpirationTime: types.Time(time.UnixMilli(1758792204196)),
		}
		assert.Equal(t, exp, result, "CreateMarginListenToken should return the cross margin listen token")
	} else {
		assert.NotEmpty(t, result.Token, "CreateMarginListenToken should return a token")
	}

	ensureTradablePairs(t)
	result, err = e.CreateMarginListenToken(t.Context(), &MarginListenTokenRequest{Symbol: marginTradablePair, IsIsolated: true, Validity: time.Hour})
	require.NoError(t, err, "CreateMarginListenToken must not error for an isolated margin account")
	if mockTests {
		exp := &MarginListenTokenResponse{
			Token:          "Kp3vQw7ZtRb2NcYx9LmHs4DfGjUaE8WoXi1TeVkPn6BqzCyJr5AgMd0OuSlFh",
			ExpirationTime: types.Time(time.UnixMilli(1758709404196)),
		}
		assert.Equal(t, exp, result, "CreateMarginListenToken should return the isolated margin listen token")
	} else {
		assert.NotEmpty(t, result.Token, "CreateMarginListenToken should return a token")
	}
}
