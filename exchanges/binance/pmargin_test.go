package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetPortfolioMarginAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		ccy  currency.Code
		exp  []AccountBalanceResponse
	}{
		{name: "allAssets", ccy: currency.EMPTYCODE, exp: []AccountBalanceResponse{
			{
				Asset:                     currency.USDT,
				TotalWalletBalance:        139.22469206,
				CrossMarginAsset:          103,
				CrossMarginBorrowed:       10,
				CrossMarginFree:           100,
				CrossMarginInterest:       0.72469206,
				CrossMarginLocked:         3,
				USDTMarginedWalletBalance: 12.5,
				USDTMarginedUnrealisedPNL: 23.72469206,
				CoinMarginedWalletBalance: 23.72469206,
				CoinMarginedUnrealisedPNL: -1.25,
				UpdateTime:                types.Time(time.UnixMilli(1617939110373)),
				NegativeBalance:           0.5,
			},
		}},
		{name: "asset", ccy: currency.BTC, exp: []AccountBalanceResponse{
			{
				Asset:                     currency.BTC,
				TotalWalletBalance:        0.31,
				CrossMarginBorrowed:       0.01,
				CrossMarginFree:           0.2,
				CrossMarginInterest:       0.0001,
				CrossMarginLocked:         0.05,
				USDTMarginedWalletBalance: 0.04,
				USDTMarginedUnrealisedPNL: 0.002,
				CoinMarginedWalletBalance: 0.02,
				CoinMarginedUnrealisedPNL: -0.001,
				UpdateTime:                types.Time(time.UnixMilli(1617939110373)),
				NegativeBalance:           0.003,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPortfolioMarginAccountBalance(t.Context(), tc.ccy)
			require.NoError(t, err, "GetPortfolioMarginAccountBalance must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPortfolioMarginAccountBalance should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetPortfolioMarginAccountBalance should return balances")
		})
	}
}

func TestGetPortfolioMarginAccountInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginAccountInformation(t.Context())
	require.NoError(t, err, "GetPortfolioMarginAccountInformation must not error")
	if mockTests {
		exp := &PortfolioMarginAccountInformationResponse{
			UniMMR:                   5167.92171923,
			AccountEquity:            73.47428058,
			ActualEquity:             122607.35137903,
			AccountInitialMargin:     23.72469206,
			AccountMaintenanceMargin: 23.72469206,
			AccountStatus:            "NORMAL",
			VirtualMaxWithdrawAmount: 1627523.32459208,
			TotalAvailableBalance:    49.75,
			TotalMarginOpenLoss:      1.5,
			UpdateTime:               types.Time(time.UnixMilli(1657707212154)),
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginAccountInformation should decode every field")
		return
	}
	assert.NotEmpty(t, result.AccountStatus, "GetPortfolioMarginAccountInformation should return an account status")
}

func TestPortfolioMarginBNBTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginBNBTransfer(t.Context(), 0, "TO_UM")
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PortfolioMarginBNBTransfer must reject a zero amount")
	_, err = e.PortfolioMarginBNBTransfer(t.Context(), 0.0001, "")
	require.ErrorIs(t, err, errTransferSideRequired, "PortfolioMarginBNBTransfer must reject an empty transfer side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginBNBTransfer(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err, "PortfolioMarginBNBTransfer must not error")
	if mockTests {
		exp := &PortfolioMarginTransactionResponse{
			TransactionID: 100000001,
		}
		assert.Equal(t, exp, result, "PortfolioMarginBNBTransfer should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "PortfolioMarginBNBTransfer should return a transaction ID")
}

func TestGetPortfolioMarginAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginAutoRepayFuturesStatus(t.Context())
	require.NoError(t, err, "GetPortfolioMarginAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PortfolioMarginAutoRepayFuturesStatusResponse{
			AutoRepay: true,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginAutoRepayFuturesStatus should return a status")
}

func TestChangePortfolioMarginAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangePortfolioMarginAutoRepayFuturesStatus(t.Context(), false)
	require.NoError(t, err, "ChangePortfolioMarginAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangePortfolioMarginAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangePortfolioMarginAutoRepayFuturesStatus should report success")
}

func TestChangePortfolioMarginCoinMarginedInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePortfolioMarginCoinMarginedInitialLeverage(t.Context(), currency.EMPTYPAIR, 29)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangePortfolioMarginCoinMarginedInitialLeverage must reject an empty symbol")
	_, err = e.ChangePortfolioMarginCoinMarginedInitialLeverage(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0)
	require.ErrorIs(t, err, errLeverageRequired, "ChangePortfolioMarginCoinMarginedInitialLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.ChangePortfolioMarginCoinMarginedInitialLeverage(t.Context(), coinmTradablePair, 29)
	require.NoError(t, err, "ChangePortfolioMarginCoinMarginedInitialLeverage must not error")
	if mockTests {
		exp := &PortfolioMarginCoinMarginedInitialLeverageResponse{
			Leverage:    29,
			MaxQuantity: 1000,
			Symbol:      "BTCUSD_PERP",
		}
		assert.Equal(t, exp, result, "ChangePortfolioMarginCoinMarginedInitialLeverage should decode every field")
		return
	}
	assert.Equal(t, uint64(29), result.Leverage, "ChangePortfolioMarginCoinMarginedInitialLeverage should return the new leverage")
}

func TestGetPortfolioMarginCoinMarginedCurrentPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginCoinMarginedCurrentPositionMode(t.Context())
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedCurrentPositionMode must not error")
	if mockTests {
		exp := &PortfolioMarginPositionModeResponse{
			DualSidePosition: true,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedCurrentPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginCoinMarginedCurrentPositionMode should return a position mode")
}

func TestChangePortfolioMarginCoinMarginedPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangePortfolioMarginCoinMarginedPositionMode(t.Context(), true)
	require.NoError(t, err, "ChangePortfolioMarginCoinMarginedPositionMode must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangePortfolioMarginCoinMarginedPositionMode should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangePortfolioMarginCoinMarginedPositionMode should report success")
}

func TestChangePortfolioMarginUSDTMarginedInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePortfolioMarginUSDTMarginedInitialLeverage(t.Context(), currency.EMPTYPAIR, 29)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangePortfolioMarginUSDTMarginedInitialLeverage must reject an empty symbol")
	_, err = e.ChangePortfolioMarginUSDTMarginedInitialLeverage(t.Context(), currency.NewBTCUSDT(), 0)
	require.ErrorIs(t, err, errLeverageRequired, "ChangePortfolioMarginUSDTMarginedInitialLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.ChangePortfolioMarginUSDTMarginedInitialLeverage(t.Context(), usdtmTradablePair, 29)
	require.NoError(t, err, "ChangePortfolioMarginUSDTMarginedInitialLeverage must not error")
	if mockTests {
		exp := &PortfolioMarginUSDTMarginedInitialLeverageResponse{
			Leverage:         29,
			MaxNotionalValue: 1000000,
			Symbol:           "BTCUSDT",
		}
		assert.Equal(t, exp, result, "ChangePortfolioMarginUSDTMarginedInitialLeverage should decode every field")
		return
	}
	assert.Equal(t, uint64(29), result.Leverage, "ChangePortfolioMarginUSDTMarginedInitialLeverage should return the new leverage")
}

func TestGetPortfolioMarginUSDTMarginedCurrentPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedCurrentPositionMode(t.Context())
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedCurrentPositionMode must not error")
	if mockTests {
		exp := &PortfolioMarginPositionModeResponse{
			DualSidePosition: true,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedCurrentPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUSDTMarginedCurrentPositionMode should return a position mode")
}

func TestChangePortfolioMarginUSDTMarginedPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangePortfolioMarginUSDTMarginedPositionMode(t.Context(), true)
	require.NoError(t, err, "ChangePortfolioMarginUSDTMarginedPositionMode must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangePortfolioMarginUSDTMarginedPositionMode should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangePortfolioMarginUSDTMarginedPositionMode should report success")
}

func TestGetPortfolioMarginCoinMarginedNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCoinMarginedNotionalAndLeverageBrackets(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedNotionalAndLeverageBrackets must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedNotionalAndLeverageResponse{
			{
				Symbol: "BTCUSD_PERP",
				Brackets: []PortfolioMarginCoinMarginedNotionalBracket{
					{
						Bracket:                1,
						InitialLeverage:        125,
						QuantityCap:            50,
						MaintenanceMarginRatio: 0.004,
					},
					{
						Bracket:                2,
						InitialLeverage:        100,
						QuantityCap:            100,
						QuantityFloor:          50,
						MaintenanceMarginRatio: 0.005,
						Cumulative:             0.05,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedNotionalAndLeverageBrackets should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPortfolioMarginCoinMarginedNotionalAndLeverageBrackets should return brackets")
}

func TestPortfolioMarginFundAutoCollection(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginFundAutoCollection(t.Context())
	require.NoError(t, err, "PortfolioMarginFundAutoCollection must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "PortfolioMarginFundAutoCollection should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "PortfolioMarginFundAutoCollection should report success")
}

func TestPortfolioMarginFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginFundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PortfolioMarginFundCollectionByAsset must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginFundCollectionByAsset(t.Context(), currency.ETH)
	require.NoError(t, err, "PortfolioMarginFundCollectionByAsset must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "PortfolioMarginFundCollectionByAsset should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "PortfolioMarginFundCollectionByAsset should report success")
}

func TestGetPortfolioMarginCoinMarginedAccountDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginCoinMarginedAccountDetail(t.Context())
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedAccountDetail must not error")
	if mockTests {
		exp := &PortfolioMarginCoinMarginedAccountDetailResponse{
			Assets: []PortfolioMarginAccountDetailAsset{
				{
					Asset:                  currency.BTC,
					CrossWalletBalance:     0.00241969,
					CrossUnrealisedPNL:     0.00000123,
					MaintenanceMargin:      0.00000456,
					InitialMargin:          0.0005,
					PositionInitialMargin:  0.0004,
					OpenOrderInitialMargin: 0.0001,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []PortfolioMarginCoinMarginedAccountDetailPosition{
				{
					Symbol:                 "BTCUSD_PERP",
					PositionAmount:         1,
					InitialMargin:          0.0005,
					MaintenanceMargin:      0.00000456,
					UnrealisedProfit:       0.00000123,
					PositionInitialMargin:  0.0004,
					OpenOrderInitialMargin: 0.0001,
					Leverage:               125,
					PositionSide:           "BOTH",
					EntryPrice:             9975.12,
					MaxQuantity:            50,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedAccountDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginCoinMarginedAccountDetail should return account details")
}

func TestGetPortfolioMarginCoinMarginedIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginCoinMarginedIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginCoinMarginedIncomeHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginCoinMarginedIncomeHistory(t.Context(), &PortfolioMarginIncomeHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginCoinMarginedIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCoinMarginedIncomeHistory(t.Context(), &PortfolioMarginIncomeHistoryRequest{Symbol: coinmTradablePair, IncomeType: "REALIZED_PNL", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 10})
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedIncomeHistory must not error")
	if mockTests {
		exp := []IncomeItemResponse{
			{
				Symbol:        "BTCUSD_PERP",
				IncomeType:    "REALIZED_PNL",
				Income:        -0.00000375,
				Asset:         currency.BTC,
				Info:          "REALIZED_PNL",
				Time:          types.Time(time.Unix(1744110000, 0)),
				TransactionID: "9689322392",
				TradeID:       "2059192",
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedIncomeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginCoinMarginedIncomeHistory should return income records")
}

func TestGetPortfolioMarginMarginBorrowOrLoanInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginBorrowOrLoanInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginMarginBorrowOrLoanInterestHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginMarginBorrowOrLoanInterestHistory(t.Context(), &PortfolioMarginMarginInterestHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginMarginBorrowOrLoanInterestHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginBorrowOrLoanInterestHistory(t.Context(), &PortfolioMarginMarginInterestHistoryRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Current: 10, Size: 1, Archived: true})
	require.NoError(t, err, "GetPortfolioMarginMarginBorrowOrLoanInterestHistory must not error")
	if mockTests {
		exp := &PortfolioMarginMarginInterestHistoryResponse{
			Rows: []PortfolioMarginMarginInterest{
				{
					TransactionID:       1352286576452864800,
					InterestAccuredTime: types.Time(time.Unix(1744110000, 0)),
					Asset:               currency.ETH,
					RawAsset:            currency.ETH,
					Principal:           45.3313,
					Interest:            0.00024995,
					InterestRate:        0.00013233,
					Type:                "ON_BORROW",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginBorrowOrLoanInterestHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginBorrowOrLoanInterestHistory should return interest records")
}

func TestGetPortfolioMarginUSDTMarginedAccountDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedAccountDetail(t.Context())
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedAccountDetail must not error")
	if mockTests {
		exp := &PortfolioMarginUSDTMarginedAccountDetailResponse{
			Assets: []PortfolioMarginAccountDetailAsset{
				{
					Asset:                  currency.USDT,
					CrossWalletBalance:     23.72469206,
					CrossUnrealisedPNL:     1.25,
					MaintenanceMargin:      0.42,
					InitialMargin:          10.5,
					PositionInitialMargin:  10,
					OpenOrderInitialMargin: 0.5,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []PortfolioMarginUSDTMarginedAccountDetailPosition{
				{
					Symbol:                 "BTCUSDT",
					InitialMargin:          10.5,
					MaintenanceMargin:      0.42,
					UnrealisedProfit:       1.25,
					PositionInitialMargin:  10,
					OpenOrderInitialMargin: 0.5,
					Leverage:               100,
					EntryPrice:             62000.5,
					MaxNotional:            250000,
					BidNotional:            50,
					AskNotional:            25,
					PositionSide:           "BOTH",
					PositionAmount:         0.017,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedAccountDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUSDTMarginedAccountDetail should return account details")
}

func TestGetPortfolioMarginUSDTMarginedIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginUSDTMarginedIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginUSDTMarginedIncomeHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginUSDTMarginedIncomeHistory(t.Context(), &PortfolioMarginIncomeHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginUSDTMarginedIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedIncomeHistory(t.Context(), &PortfolioMarginIncomeHistoryRequest{Symbol: usdtmTradablePair, IncomeType: "COMMISSION", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 10})
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedIncomeHistory must not error")
	if mockTests {
		exp := []IncomeItemResponse{
			{
				Symbol:        "BTCUSDT",
				IncomeType:    "COMMISSION",
				Income:        -0.01,
				Asset:         currency.USDT,
				Info:          "COMMISSION",
				Time:          types.Time(time.Unix(1744110000, 0)),
				TransactionID: "9689322393",
				TradeID:       "2059193",
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedIncomeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUSDTMarginedIncomeHistory should return income records")
}

func TestGetPortfolioMarginCoinMarginedUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginCoinMarginedUserCommissionRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginCoinMarginedUserCommissionRate must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCoinMarginedUserCommissionRate(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedUserCommissionRate must not error")
	if mockTests {
		exp := &PortfolioMarginCommissionRateResponse{
			Symbol:              "BTCUSD_PERP",
			MakerCommissionRate: 0.00015,
			TakerCommissionRate: 0.0004,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedUserCommissionRate should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbol, "GetPortfolioMarginCoinMarginedUserCommissionRate should return the symbol")
}

func TestGetPortfolioMarginUSDTMarginedUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginUSDTMarginedUserCommissionRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginUSDTMarginedUserCommissionRate must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedUserCommissionRate(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedUserCommissionRate must not error")
	if mockTests {
		exp := &PortfolioMarginCommissionRateResponse{
			Symbol:              "BTCUSDT",
			MakerCommissionRate: 0.0002,
			TakerCommissionRate: 0.0004,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedUserCommissionRate should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbol, "GetPortfolioMarginUSDTMarginedUserCommissionRate should return the symbol")
}

func TestGetPortfolioMarginMarginMaxBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginMaxBorrow(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPortfolioMarginMarginMaxBorrow must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginMaxBorrow(t.Context(), currency.ETH)
	require.NoError(t, err, "GetPortfolioMarginMarginMaxBorrow must not error")
	if mockTests {
		exp := &PortfolioMarginMarginMaxBorrowResponse{
			Amount:      1.69248805,
			BorrowLimit: 60,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginMaxBorrow should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginMaxBorrow should return borrow limits")
}

func TestGetPortfolioMarginUSDTMarginedTradingQuantitativeRulesIndicator(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedTradingQuantitativeRulesIndicator(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedTradingQuantitativeRulesIndicator must not error")
	if mockTests {
		exp := &PortfolioMarginTradingQuantitativeRulesIndicatorsResponse{
			Indicators: map[string][]PortfolioMarginTradingIndicator{
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
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedTradingQuantitativeRulesIndicator should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUSDTMarginedTradingQuantitativeRulesIndicator should return indicators")
}

func TestGetPortfolioMarginCoinMarginedPositionInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name        string
		marginAsset currency.Code
		pair        currency.Code
		exp         []PortfolioMarginCoinMarginedPositionInformationResponse
	}{
		{name: "marginAsset", marginAsset: currency.ETH, exp: []PortfolioMarginCoinMarginedPositionInformationResponse{
			{
				Symbol:           "ETHUSD_PERP",
				PositionAmount:   10,
				EntryPrice:       1580.5,
				MarkPrice:        1590.25,
				UnrealisedProfit: 0.00038801,
				LiquidationPrice: 1201.75,
				Leverage:         20,
				PositionSide:     "LONG",
				UpdateTime:       types.Time(time.UnixMilli(1627026881327)),
				MaxQuantity:      1000,
				NotionalValue:    0.06288319,
			},
		}},
		{name: "pair", pair: currency.NewCode("BTCUSD"), exp: []PortfolioMarginCoinMarginedPositionInformationResponse{
			{
				Symbol:           "BTCUSD_201225",
				PositionAmount:   1,
				EntryPrice:       11707.70000003,
				MarkPrice:        11788.66626667,
				UnrealisedProfit: 0.00005866,
				LiquidationPrice: 11667.63509587,
				Leverage:         125,
				PositionSide:     "BOTH",
				UpdateTime:       types.Time(time.UnixMilli(1627026881327)),
				MaxQuantity:      50,
				NotionalValue:    0.00084827,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPortfolioMarginCoinMarginedPositionInformation(t.Context(), tc.marginAsset, tc.pair)
			require.NoError(t, err, "GetPortfolioMarginCoinMarginedPositionInformation must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPortfolioMarginCoinMarginedPositionInformation should decode every field")
				return
			}
			assert.NotNil(t, result, "GetPortfolioMarginCoinMarginedPositionInformation should return positions")
		})
	}
}

func TestGetPortfolioMarginMarginLoanRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginLoanRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginMarginLoanRecord must reject a nil request")
	_, err = e.GetPortfolioMarginMarginLoanRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{TransactionID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPortfolioMarginMarginLoanRecord must reject an empty asset")
	_, err = e.GetPortfolioMarginMarginLoanRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{Asset: currency.ETH})
	require.ErrorIs(t, err, errStartTimeRequired, "GetPortfolioMarginMarginLoanRecord must reject a request without a transaction ID or start time")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginMarginLoanRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{Asset: currency.ETH, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginMarginLoanRecord must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginLoanRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{Asset: currency.ETH, TransactionID: 12807067523, StartTime: startTime, EndTime: endTime, Current: 10, Size: 1, Archived: true})
	require.NoError(t, err, "GetPortfolioMarginMarginLoanRecord must not error")
	if mockTests {
		exp := &PortfolioMarginMarginLoanRecordResponse{
			Rows: []PortfolioMarginMarginLoanRecord{
				{
					TransactionID: 12807067523,
					Asset:         currency.ETH,
					Principal:     0.84624403,
					Timestamp:     types.Time(time.Unix(1744110000, 0)),
					Status:        "CONFIRMED",
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginLoanRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginLoanRecord should return loan records")
}

func TestGetPortfolioMarginMarginMaxWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginMaxWithdrawal(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPortfolioMarginMarginMaxWithdrawal must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginMaxWithdrawal(t.Context(), currency.BTC)
	require.NoError(t, err, "GetPortfolioMarginMarginMaxWithdrawal must not error")
	if mockTests {
		exp := &PortfolioMarginMarginMaxWithdrawResponse{
			Amount: 60,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginMaxWithdrawal should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginMaxWithdrawal should return an amount")
}

func TestGetPortfolioMarginMarginRepayRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginRepayRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginMarginRepayRecord must reject a nil request")
	_, err = e.GetPortfolioMarginMarginRepayRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{TransactionID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPortfolioMarginMarginRepayRecord must reject an empty asset")
	_, err = e.GetPortfolioMarginMarginRepayRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{Asset: currency.ETH})
	require.ErrorIs(t, err, errStartTimeRequired, "GetPortfolioMarginMarginRepayRecord must reject a request without a transaction ID or start time")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginMarginRepayRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{Asset: currency.ETH, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginMarginRepayRecord must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginRepayRecord(t.Context(), &PortfolioMarginMarginLoanRepayRecordRequest{Asset: currency.ETH, TransactionID: 2970933056, StartTime: startTime, EndTime: endTime, Current: 10, Size: 1, Archived: true})
	require.NoError(t, err, "GetPortfolioMarginMarginRepayRecord must not error")
	if mockTests {
		exp := &PortfolioMarginMarginRepayRecordResponse{
			Rows: []PortfolioMarginMarginRepayRecord{
				{
					Amount:        14,
					Asset:         currency.ETH,
					Interest:      0.01866667,
					Principal:     13.98133333,
					Status:        "CONFIRMED",
					Timestamp:     types.Time(time.Unix(1744110000, 0)),
					TransactionID: 2970933056,
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginRepayRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginRepayRecord should return repay records")
}

func TestGetPortfolioMarginNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginNegativeBalanceInterestHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), &PortfolioMarginNegativeBalanceInterestHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginNegativeBalanceInterestHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), &PortfolioMarginNegativeBalanceInterestHistoryRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Size: 10})
	require.NoError(t, err, "GetPortfolioMarginNegativeBalanceInterestHistory must not error")
	if mockTests {
		exp := []PortfolioMarginNegativeBalanceInterestResponse{
			{
				Asset:               currency.ETH,
				Interest:            24.444,
				InterestAccuredTime: types.Time(time.Unix(1744110000, 0)),
				InterestRate:        0.0001164,
				Principal:           210000,
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginNegativeBalanceInterestHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginNegativeBalanceInterestHistory should return interest records")
}

func TestGetPortfolioMarginUSDTMarginedPositionInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedPositionInformation(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedPositionInformation must not error")
	if mockTests {
		exp := []PortfolioMarginUSDTMarginedPositionInformationResponse{
			{
				EntryPrice:       62000.5,
				Leverage:         10,
				MarkPrice:        62500.1,
				MaxNotionalValue: 20000000,
				PositionAmount:   0.01,
				Notional:         625.001,
				Symbol:           "BTCUSDT",
				UnrealisedProfit: 4.996,
				LiquidationPrice: 6170.20509059,
				PositionSide:     "BOTH",
				UpdateTime:       types.Time(time.UnixMilli(1625474304765)),
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedPositionInformation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUSDTMarginedPositionInformation should return positions")
}

func TestGetPortfolioMarginUserNegativeBalanceAutoExchangeRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord(t.Context(), time.Time{}, endTime)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord must reject an unset start time")
	_, err = e.GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord(t.Context(), startTime, time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord must reject an unset end time")
	_, err = e.GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord(t.Context(), startTime, endTime)
	require.NoError(t, err, "GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord must not error")
	if mockTests {
		exp := &PortfolioMarginNegativeBalanceExchangeRecordResponse{
			Total: 1,
			Rows: []PortfolioMarginNegativeBalanceExchangeRecord{
				{
					StartTime: types.Time(time.Unix(1744110000, 0)),
					EndTime:   types.Time(time.UnixMilli(1744110201338)),
					Details: []PortfolioMarginNegativeBalanceExchangeDetail{
						{
							Asset:                currency.ETH,
							NegativeBalance:      1.10264488,
							NegativeMaxThreshold: 5,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUserNegativeBalanceAutoExchangeRecord should return records")
}

func TestGetPortfolioMarginUserRateLimits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUserRateLimits(t.Context())
	require.NoError(t, err, "GetPortfolioMarginUserRateLimits must not error")
	if mockTests {
		exp := []PortfolioMarginRateLimitResponse{
			{
				RateLimitType:  "ORDERS",
				Interval:       "MINUTE",
				IntervalNumber: 1,
				Limit:          1200,
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUserRateLimits should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPortfolioMarginUserRateLimits should return rate limits")
}

func TestPortfolioMarginRepayFuturesNegativeBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginRepayFuturesNegativeBalance(t.Context())
	require.NoError(t, err, "PortfolioMarginRepayFuturesNegativeBalance must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "PortfolioMarginRepayFuturesNegativeBalance should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "PortfolioMarginRepayFuturesNegativeBalance should report success")
}

func TestGetPortfolioMarginUSDTMarginedNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedNotionalAndLeverageBrackets(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedNotionalAndLeverageBrackets must not error")
	if mockTests {
		exp := []PortfolioMarginUSDTMarginedNotionalAndLeverageResponse{
			{
				Symbol:              "BTCUSDT",
				NotionalCoefficient: 1.5,
				Brackets: []PortfolioMarginUSDTMarginedNotionalBracket{
					{
						Bracket:                1,
						InitialLeverage:        75,
						NotionalCap:            10000,
						MaintenanceMarginRatio: 0.0065,
					},
					{
						Bracket:                2,
						InitialLeverage:        50,
						NotionalCap:            50000,
						NotionalFloor:          10000,
						MaintenanceMarginRatio: 0.01,
						Cumulative:             35,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedNotionalAndLeverageBrackets should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPortfolioMarginUSDTMarginedNotionalAndLeverageBrackets should return brackets")
}

func TestCancelAllPortfolioMarginCoinMarginedOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "The operation of cancel all conditional open order is done.",
		}
		assert.Equal(t, exp, result, "CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result.Message, "CancelAllPortfolioMarginCoinMarginedOpenConditionalOrders should return a message")
}

func TestCancelAllPortfolioMarginCoinMarginedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllPortfolioMarginCoinMarginedOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllPortfolioMarginCoinMarginedOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllPortfolioMarginCoinMarginedOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "CancelAllPortfolioMarginCoinMarginedOrders must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "The operation of cancel all open order is done.",
		}
		assert.Equal(t, exp, result, "CancelAllPortfolioMarginCoinMarginedOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result.Message, "CancelAllPortfolioMarginCoinMarginedOrders should return a message")
}

func TestCancelAllPortfolioMarginUSDTMarginedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllPortfolioMarginUSDTMarginedOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllPortfolioMarginUSDTMarginedOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllPortfolioMarginUSDTMarginedOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "CancelAllPortfolioMarginUSDTMarginedOrders must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "The operation of cancel all open order is done.",
		}
		assert.Equal(t, exp, result, "CancelAllPortfolioMarginUSDTMarginedOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result.Message, "CancelAllPortfolioMarginUSDTMarginedOrders should return a message")
}

func TestNewPortfolioMarginCoinMarginedConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewPortfolioMarginCoinMarginedConditionalOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewPortfolioMarginCoinMarginedConditionalOrder must reject a nil request")
	arg := &PortfolioMarginCoinMarginedConditionalOrderRequest{PositionSide: "LONG"}
	_, err = e.NewPortfolioMarginCoinMarginedConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewPortfolioMarginCoinMarginedConditionalOrder must reject an empty symbol")
	arg.Symbol = currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	_, err = e.NewPortfolioMarginCoinMarginedConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewPortfolioMarginCoinMarginedConditionalOrder must reject an invalid side")
	arg.Side = order.Buy
	_, err = e.NewPortfolioMarginCoinMarginedConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStrategyTypeRequired, "NewPortfolioMarginCoinMarginedConditionalOrder must reject an empty strategy type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.NewPortfolioMarginCoinMarginedConditionalOrder(t.Context(), &PortfolioMarginCoinMarginedConditionalOrderRequest{
		Symbol:              coinmTradablePair,
		Side:                order.Buy,
		PositionSide:        "BOTH",
		StrategyType:        "TRAILING_STOP_MARKET",
		TimeInForce:         "GTC",
		Quantity:            1,
		ReduceOnly:          true,
		Price:               9500,
		WorkingType:         "MARK_PRICE",
		PriceProtect:        true,
		NewClientStrategyID: "testOrder",
		StopPrice:           9300,
		ActivationPrice:     9020,
		CallbackRate:        0.3,
	})
	require.NoError(t, err, "NewPortfolioMarginCoinMarginedConditionalOrder must not error")
	if mockTests {
		exp := &NewPortfolioMarginCoinMarginedConditionalOrderResponse{
			NewClientStrategyID: "testOrder",
			StrategyID:          123445,
			StrategyStatus:      "NEW",
			StrategyType:        "TRAILING_STOP_MARKET",
			OriginalQuantity:    1,
			Price:               9500,
			ReduceOnly:          true,
			Side:                "BUY",
			PositionSide:        "BOTH",
			StopPrice:           9300,
			Symbol:              "BTCUSD_PERP",
			TimeInForce:         "GTC",
			ActivatePrice:       9020,
			PriceRate:           0.3,
			BookTime:            types.Time(time.UnixMilli(1566818724710)),
			UpdateTime:          types.Time(time.UnixMilli(1566818724722)),
			Pair:                "BTCUSD",
			WorkingType:         "MARK_PRICE",
			PriceProtect:        true,
		}
		assert.Equal(t, exp, result, "NewPortfolioMarginCoinMarginedConditionalOrder should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "NewPortfolioMarginCoinMarginedConditionalOrder should return a strategy ID")
}

func TestCancelPortfolioMarginCoinMarginedConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPortfolioMarginCoinMarginedConditionalOrder(t.Context(), currency.EMPTYPAIR, 1231231, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelPortfolioMarginCoinMarginedConditionalOrder must reject an empty symbol")
	_, err = e.CancelPortfolioMarginCoinMarginedConditionalOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPortfolioMarginCoinMarginedConditionalOrder must reject a request without a strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelPortfolioMarginCoinMarginedConditionalOrder(t.Context(), coinmTradablePair, 1231231, "myOrder1")
	require.NoError(t, err, "CancelPortfolioMarginCoinMarginedConditionalOrder must not error")
	if mockTests {
		exp := &CancelPortfolioMarginCoinMarginedConditionalOrderResponse{
			NewClientStrategyID: "myOrder1",
			StrategyID:          1231231,
			StrategyStatus:      "CANCELED",
			StrategyType:        "TRAILING_STOP_MARKET",
			OriginalQuantity:    11,
			Price:               9500,
			ReduceOnly:          true,
			Side:                "BUY",
			PositionSide:        "BOTH",
			StopPrice:           9300,
			Symbol:              "BTCUSD_PERP",
			TimeInForce:         "GTC",
			ActivatePrice:       9020,
			PriceRate:           0.3,
			BookTime:            types.Time(time.UnixMilli(1566818724710)),
			UpdateTime:          types.Time(time.UnixMilli(1566818724722)),
			WorkingType:         "CONTRACT_PRICE",
			PriceProtect:        true,
		}
		assert.Equal(t, exp, result, "CancelPortfolioMarginCoinMarginedConditionalOrder should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "CancelPortfolioMarginCoinMarginedConditionalOrder should return the strategy ID")
}

func TestGetPortfolioMarginCoinMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginCoinMarginedOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginCoinMarginedOrder must reject an empty symbol")
	_, err = e.GetPortfolioMarginCoinMarginedOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginCoinMarginedOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCoinMarginedOrder(t.Context(), coinmTradablePair, 1234, "")
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedOrder must not error")
	if mockTests {
		exp := &PortfolioMarginCoinMarginedOrderDetailResponse{
			AveragePrice:     9500.5,
			ClientOrderID:    "abc",
			CumulativeBase:   0.01052576,
			ExecutedQuantity: 1,
			OrderID:          1234,
			OriginalQuantity: 2,
			OriginalType:     "LIMIT",
			Price:            9500.5,
			ReduceOnly:       true,
			Side:             "BUY",
			Status:           "PARTIALLY_FILLED",
			Symbol:           "BTCUSD_PERP",
			Pair:             "BTCUSD",
			PositionSide:     "BOTH",
			Time:             types.Time(time.UnixMilli(1579276756075)),
			TimeInForce:      "GTC",
			Type:             "LIMIT",
			UpdateTime:       types.Time(time.UnixMilli(1579276756075)),
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetPortfolioMarginCoinMarginedOrder should return the order ID")
}

func TestNewPortfolioMarginCoinMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewPortfolioMarginCoinMarginedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewPortfolioMarginCoinMarginedOrder must reject a nil request")
	arg := &PortfolioMarginCoinMarginedOrderRequest{ReduceOnly: true}
	_, err = e.NewPortfolioMarginCoinMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewPortfolioMarginCoinMarginedOrder must reject an empty symbol")
	arg.Symbol = currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	_, err = e.NewPortfolioMarginCoinMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewPortfolioMarginCoinMarginedOrder must reject an invalid side")
	arg.Side = order.Buy
	_, err = e.NewPortfolioMarginCoinMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewPortfolioMarginCoinMarginedOrder must reject an empty order type")
	arg.OrderType = "LIMIT"
	arg.Price = 9500
	arg.PriceMatch = "OPPONENT"
	_, err = e.NewPortfolioMarginCoinMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "NewPortfolioMarginCoinMarginedOrder must reject a price with priceMatch")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginCoinMarginedOrderRequest
		exp  *PortfolioMarginCoinMarginedOrderResponse
	}{
		{
			name: "price",
			arg:  &PortfolioMarginCoinMarginedOrderRequest{Symbol: coinmTradablePair, Side: order.Buy, PositionSide: "BOTH", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 2, ReduceOnly: true, Price: 9500, NewClientOrderID: "testOrder", NewOrderResponseType: "RESULT"},
			exp: &PortfolioMarginCoinMarginedOrderResponse{
				ClientOrderID:      "testOrder",
				CumulativeQuantity: 1,
				ExecutedQuantity:   1,
				OrderID:            22542179,
				OriginalQuantity:   2,
				Price:              9500,
				ReduceOnly:         true,
				Side:               "BUY",
				PositionSide:       "BOTH",
				Status:             "PARTIALLY_FILLED",
				Symbol:             "BTCUSD_PERP",
				Pair:               "BTCUSD",
				TimeInForce:        "GTC",
				Type:               "LIMIT",
				UpdateTime:         types.Time(time.UnixMilli(1566818724722)),
			},
		},
		{
			name: "priceMatch",
			arg:  &PortfolioMarginCoinMarginedOrderRequest{Symbol: coinmTradablePair, Side: order.Sell, OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 2, PriceMatch: "OPPONENT"},
			exp: &PortfolioMarginCoinMarginedOrderResponse{
				ClientOrderID:      "x-priceMatch",
				CumulativeQuantity: 1,
				ExecutedQuantity:   1,
				OrderID:            22542180,
				OriginalQuantity:   2,
				Price:              9499.9,
				Side:               "SELL",
				PositionSide:       "BOTH",
				Status:             "PARTIALLY_FILLED",
				Symbol:             "BTCUSD_PERP",
				Pair:               "BTCUSD",
				TimeInForce:        "GTC",
				Type:               "LIMIT",
				UpdateTime:         types.Time(time.UnixMilli(1566818724722)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.NewPortfolioMarginCoinMarginedOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewPortfolioMarginCoinMarginedOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewPortfolioMarginCoinMarginedOrder should decode every field")
				return
			}
			assert.NotZero(t, result.OrderID, "NewPortfolioMarginCoinMarginedOrder should return an order ID")
		})
	}
}

func TestCancelPortfolioMarginCoinMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPortfolioMarginCoinMarginedOrder(t.Context(), currency.EMPTYPAIR, 21321312, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelPortfolioMarginCoinMarginedOrder must reject an empty symbol")
	_, err = e.CancelPortfolioMarginCoinMarginedOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPortfolioMarginCoinMarginedOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelPortfolioMarginCoinMarginedOrder(t.Context(), coinmTradablePair, 21321312, "myOrder1")
	require.NoError(t, err, "CancelPortfolioMarginCoinMarginedOrder must not error")
	if mockTests {
		exp := &PortfolioMarginCoinMarginedOrderResponse{
			ClientOrderID:      "myOrder1",
			CumulativeQuantity: 1,
			ExecutedQuantity:   1,
			OrderID:            21321312,
			OriginalQuantity:   2,
			Price:              9500,
			ReduceOnly:         true,
			Side:               "BUY",
			PositionSide:       "BOTH",
			Status:             "CANCELED",
			Symbol:             "BTCUSD_PERP",
			Pair:               "BTCUSD",
			TimeInForce:        "GTC",
			Type:               "LIMIT",
			UpdateTime:         types.Time(time.UnixMilli(1571110484038)),
		}
		assert.Equal(t, exp, result, "CancelPortfolioMarginCoinMarginedOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "CancelPortfolioMarginCoinMarginedOrder should return the order ID")
}

func TestCancelAllPortfolioMarginMarginOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllPortfolioMarginMarginOpenOrdersBySymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllPortfolioMarginMarginOpenOrdersBySymbol must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllPortfolioMarginMarginOpenOrdersBySymbol(t.Context(), marginTradablePair)
	require.NoError(t, err, "CancelAllPortfolioMarginMarginOpenOrdersBySymbol must not error")
	if mockTests {
		exp := []PortfolioMarginCancelledMarginOrderResponse{
			{
				Symbol:                  "BTCUSDT",
				OriginalClientOrderID:   "E6APeyTJvkMvLMYMqu1KQ4",
				OrderID:                 11,
				OrderListID:             -1,
				ClientOrderID:           "pXLV6Hz6mprAcVYpVMTGgx",
				Price:                   61000,
				OriginalQuantity:        0.02,
				ExecutedQuantity:        0.005,
				CumulativeQuoteQuantity: 305,
				Status:                  "CANCELED",
				TimeInForce:             "GTC",
				Type:                    "LIMIT",
				Side:                    "BUY",
			},
			{
				Symbol:            "BTCUSDT",
				OrderListID:       1929,
				ContingencyType:   "OCO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "2inzWQdDvZLHbbAmAozX2N",
				TransactionTime:   types.Time(time.UnixMilli(1585230948299)),
				Orders: []PortfolioMarginMarginOneCancelsTheOtherOrderLeg{
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
				OrderReports: []PortfolioMarginCancelledMarginOrderReport{
					{
						Symbol:                  "BTCUSDT",
						OriginalClientOrderID:   "CwOOIPHSmYywx6jZX77TdL",
						OrderID:                 20,
						OrderListID:             1929,
						ClientOrderID:           "pXLV6Hz6mprAcVYpVMTGgy",
						Price:                   59000,
						OriginalQuantity:        0.02,
						ExecutedQuantity:        0.001,
						CumulativeQuoteQuantity: 59,
						Status:                  "CANCELED",
						TimeInForce:             "GTC",
						Type:                    "STOP_LOSS_LIMIT",
						Side:                    "SELL",
						StopPrice:               59100,
						IcebergQuantity:         0.01,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "CancelAllPortfolioMarginMarginOpenOrdersBySymbol should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllPortfolioMarginMarginOpenOrdersBySymbol should return the cancelled orders")
}

func TestGetPortfolioMarginMarginAccountOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder must reject a request without an order list ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder(t.Context(), 0, "123421-abcde")
	require.NoError(t, err, "GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder must not error")
	if mockTests {
		exp := &PortfolioMarginMarginOneCancelsTheOtherOrderResponse{
			OrderListID:       27,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
			TransactionTime:   types.Time(time.UnixMilli(1565245656253)),
			Symbol:            "BTCUSDT",
			Orders: []PortfolioMarginMarginOneCancelsTheOtherOrderLeg{
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
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderListID, "GetPortfolioMarginMarginAccountOneCancelsTheOtherOrder should return the order list ID")
}

func TestCancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders must reject a nil request")
	_, err = e.CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders(t.Context(), &PortfolioMarginCancelMarginOneCancelsTheOtherOrderRequest{OrderListID: 1929})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders must reject an empty symbol")
	_, err = e.CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders(t.Context(), &PortfolioMarginCancelMarginOneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders must reject a request without an order list ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders(t.Context(), &PortfolioMarginCancelMarginOneCancelsTheOtherOrderRequest{Symbol: marginTradablePair, OrderListID: 1929, ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN", NewClientOrderID: "cancelMyList"})
	require.NoError(t, err, "CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders must not error")
	if mockTests {
		exp := &PortfolioMarginCancelMarginOneCancelsTheOtherOrderResponse{
			OrderListID:       1929,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN",
			TransactionTime:   types.Time(time.UnixMilli(1574040868128)),
			Symbol:            "BTCUSDT",
			Orders: []PortfolioMarginMarginOneCancelsTheOtherOrderLeg{
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
			OrderReports: []PortfolioMarginCancelMarginOneCancelsTheOtherOrderReport{
				{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "pO9ufTiFGg3nw2fOdgeOXa",
					OrderID:                 2,
					OrderListID:             1929,
					ClientOrderID:           "unfWT8ig8i0uj6lPuYLez6",
					Price:                   59000,
					OriginalQuantity:        0.02,
					ExecutedQuantity:        0.002,
					CumulativeQuoteQuantity: 118,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					StopPrice:               59100,
				},
			},
		}
		assert.Equal(t, exp, result, "CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders should decode every field")
		return
	}
	assert.NotZero(t, result.OrderListID, "CancelPortfolioMarginMarginAccountOneCancelsTheOtherOrders should return the order list ID")
}

func TestGetPortfolioMarginMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginAccountOrder(t.Context(), currency.EMPTYPAIR, 12434, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginMarginAccountOrder must reject an empty symbol")
	_, err = e.GetPortfolioMarginMarginAccountOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginMarginAccountOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginMarginAccountOrder(t.Context(), marginTradablePair, 12434, "")
	require.NoError(t, err, "GetPortfolioMarginMarginAccountOrder must not error")
	if mockTests {
		exp := &PortfolioMarginMarginOrderResponse{
			ClientOrderID:           "ZwfQzuDIGpceVhKW5DvCmO",
			CumulativeQuoteQuantity: 1500.5,
			ExecutedQuantity:        0.025,
			IcebergQuantity:         0.01,
			IsWorking:               true,
			OrderID:                 12434,
			OriginalQuantity:        0.3,
			Price:                   60020,
			Side:                    "SELL",
			Status:                  "PARTIALLY_FILLED",
			StopPrice:               60000,
			Symbol:                  "BTCUSDT",
			Time:                    types.Time(time.UnixMilli(1562133008725)),
			TimeInForce:             "GTC",
			Type:                    "STOP_LOSS_LIMIT",
			UpdateTime:              types.Time(time.UnixMilli(1562133008725)),
			AccountID:               152950866,
			SelfTradePreventionMode: "EXPIRE_TAKER",
			PreventedMatchID:        1,
			PreventedQuantity:       0.002,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginAccountOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetPortfolioMarginMarginAccountOrder should return the order ID")
}

func TestNewPortfolioMarginMarginOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewPortfolioMarginMarginOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewPortfolioMarginMarginOrder must reject a nil request")
	arg := &PortfolioMarginMarginOrderRequest{TimeInForce: "GTC"}
	_, err = e.NewPortfolioMarginMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewPortfolioMarginMarginOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.NewPortfolioMarginMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewPortfolioMarginMarginOrder must reject an invalid side")
	arg.Side = order.Sell
	_, err = e.NewPortfolioMarginMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewPortfolioMarginMarginOrder must reject an empty order type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	autoRepayAtCancel := false
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginMarginOrderRequest
		exp  *PortfolioMarginNewMarginOrderResponse
	}{
		{
			name: "stopLossLimit",
			arg:  &PortfolioMarginMarginOrderRequest{Symbol: marginTradablePair, Side: order.Sell, OrderType: "STOP_LOSS_LIMIT", Quantity: 0.001, Price: 60000, StopPrice: 60100, NewClientOrderID: "6gCrw2kRUAF9CvJDGP16IP", NewOrderResponseType: "FULL", IcebergQuantity: 0.0005, SideEffectType: "MARGIN_BUY", TimeInForce: "GTC", SelfTradePreventionMode: "EXPIRE_TAKER", AutoRepayAtCancel: &autoRepayAtCancel},
			exp: &PortfolioMarginNewMarginOrderResponse{
				Symbol:                "BTCUSDT",
				OrderID:               28,
				ClientOrderID:         "6gCrw2kRUAF9CvJDGP16IP",
				TransactTime:          types.Time(time.UnixMilli(1507725176595)),
				Price:                 60000,
				OriginalQuantity:      0.001,
				Status:                "NEW",
				TimeInForce:           "GTC",
				Type:                  "STOP_LOSS_LIMIT",
				Side:                  "SELL",
				MarginBuyBorrowAmount: 0.001,
				MarginBuyBorrowAsset:  currency.BTC,
				Fills:                 []PortfolioMarginMarginOrderFill{},
			},
		},
		{
			name: "marketQuoteQuantity",
			arg:  &PortfolioMarginMarginOrderRequest{Symbol: marginTradablePair, Side: order.Buy, OrderType: "MARKET", QuoteOrderQuantity: 100, NewOrderResponseType: "FULL"},
			exp: &PortfolioMarginNewMarginOrderResponse{
				Symbol:                  "BTCUSDT",
				OrderID:                 29,
				ClientOrderID:           "8xH4F9NkTRjQ1Rz2bR6l3P",
				TransactTime:            types.Time(time.UnixMilli(1507725176596)),
				OriginalQuantity:        0.00166,
				ExecutedQuantity:        0.00166,
				CumulativeQuoteQuantity: 100,
				Status:                  "FILLED",
				TimeInForce:             "GTC",
				Type:                    "MARKET",
				Side:                    "BUY",
				Fills: []PortfolioMarginMarginOrderFill{
					{
						Price:           60240.96,
						Quantity:        0.00166,
						Commission:      0.00000166,
						CommissionAsset: currency.BTC,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.NewPortfolioMarginMarginOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewPortfolioMarginMarginOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewPortfolioMarginMarginOrder should decode every field")
				return
			}
			assert.NotZero(t, result.OrderID, "NewPortfolioMarginMarginOrder should return an order ID")
		})
	}
}

func TestPortfolioMarginCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginCancelMarginAccountOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PortfolioMarginCancelMarginAccountOrder must reject a nil request")
	_, err = e.PortfolioMarginCancelMarginAccountOrder(t.Context(), &PortfolioMarginCancelMarginOrderRequest{OrderID: 12314})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PortfolioMarginCancelMarginAccountOrder must reject an empty symbol")
	_, err = e.PortfolioMarginCancelMarginAccountOrder(t.Context(), &PortfolioMarginCancelMarginOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "PortfolioMarginCancelMarginAccountOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.PortfolioMarginCancelMarginAccountOrder(t.Context(), &PortfolioMarginCancelMarginOrderRequest{Symbol: marginTradablePair, OrderID: 12314, NewClientOrderID: "cancelMyOrder1"})
	require.NoError(t, err, "PortfolioMarginCancelMarginAccountOrder must not error")
	if mockTests {
		exp := &PortfolioMarginCancelMarginOrderResponse{
			Symbol:                  "BTCUSDT",
			OrderID:                 12314,
			OriginalClientOrderID:   "myOrder1",
			ClientOrderID:           "cancelMyOrder1",
			Price:                   60000,
			OriginalQuantity:        0.01,
			ExecutedQuantity:        0.008,
			CumulativeQuoteQuantity: 480,
			Status:                  "CANCELED",
			TimeInForce:             "GTC",
			Type:                    "LIMIT",
			Side:                    "SELL",
		}
		assert.Equal(t, exp, result, "PortfolioMarginCancelMarginAccountOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "PortfolioMarginCancelMarginAccountOrder should return the order ID")
}

func TestGetPortfolioMarginUSDTMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginUSDTMarginedOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginUSDTMarginedOrder must reject an empty symbol")
	_, err = e.GetPortfolioMarginUSDTMarginedOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginUSDTMarginedOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedOrder(t.Context(), usdtmTradablePair, 1234, "")
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedOrder must not error")
	if mockTests {
		exp := &PortfolioMarginUSDTMarginedOrderDetailResponse{
			AveragePrice:            62000.5,
			ClientOrderID:           "abc",
			CumulativeQuote:         620.005,
			ExecutedQuantity:        0.01,
			OrderID:                 1234,
			OriginalQuantity:        0.02,
			OriginalType:            "LIMIT",
			Price:                   62000.5,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "BOTH",
			Status:                  "PARTIALLY_FILLED",
			Symbol:                  "BTCUSDT",
			Time:                    types.Time(time.UnixMilli(1579276756075)),
			TimeInForce:             "GTD",
			Type:                    "LIMIT",
			UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
			SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate:            types.Time(time.Unix(1693207680, 0)),
			PriceMatch:              "QUEUE",
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetPortfolioMarginUSDTMarginedOrder should return the order ID")
}

func TestNewPortfolioMarginUSDTMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewPortfolioMarginUSDTMarginedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewPortfolioMarginUSDTMarginedOrder must reject a nil request")
	arg := &PortfolioMarginUSDTMarginedOrderRequest{ReduceOnly: true}
	_, err = e.NewPortfolioMarginUSDTMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewPortfolioMarginUSDTMarginedOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.NewPortfolioMarginUSDTMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewPortfolioMarginUSDTMarginedOrder must reject an invalid side")
	arg.Side = order.Buy
	_, err = e.NewPortfolioMarginUSDTMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewPortfolioMarginUSDTMarginedOrder must reject an empty order type")
	arg.OrderType = "LIMIT"
	arg.Price = 62000.5
	arg.PriceMatch = "QUEUE"
	_, err = e.NewPortfolioMarginUSDTMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "NewPortfolioMarginUSDTMarginedOrder must reject a price with priceMatch")
	arg.PriceMatch = ""
	arg.TimeInForce = "GTD"
	_, err = e.NewPortfolioMarginUSDTMarginedOrder(t.Context(), arg)
	require.ErrorIs(t, err, errGoodTillDateRequired, "NewPortfolioMarginUSDTMarginedOrder must reject a GTD order without a good till date")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginUSDTMarginedOrderRequest
		exp  *PortfolioMarginUSDTMarginedOrderResponse
	}{
		{
			name: "goodTillDate",
			arg:  &PortfolioMarginUSDTMarginedOrderRequest{Symbol: usdtmTradablePair, Side: order.Buy, PositionSide: "BOTH", OrderType: "LIMIT", TimeInForce: "GTD", Quantity: 0.02, ReduceOnly: true, Price: 62000.5, NewClientOrderID: "testOrder", NewOrderResponseType: "RESULT", SelfTradePreventionMode: "EXPIRE_MAKER", GoodTillDate: time.UnixMilli(1693207680000)},
			exp: &PortfolioMarginUSDTMarginedOrderResponse{
				ClientOrderID:           "testOrder",
				CumulativeQuantity:      0.01,
				ExecutedQuantity:        0.01,
				OrderID:                 22542179,
				OriginalQuantity:        0.02,
				Price:                   62000.5,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "BOTH",
				Status:                  "PARTIALLY_FILLED",
				Symbol:                  "BTCUSDT",
				TimeInForce:             "GTD",
				Type:                    "LIMIT",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				GoodTillDate:            types.Time(time.Unix(1693207680, 0)),
				UpdateTime:              types.Time(time.UnixMilli(1566818724722)),
				PriceMatch:              "NONE",
			},
		},
		{
			name: "priceMatch",
			arg:  &PortfolioMarginUSDTMarginedOrderRequest{Symbol: usdtmTradablePair, Side: order.Sell, OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 0.02, PriceMatch: "QUEUE"},
			exp: &PortfolioMarginUSDTMarginedOrderResponse{
				ClientOrderID:           "x-priceMatch",
				CumulativeQuantity:      0.01,
				ExecutedQuantity:        0.01,
				OrderID:                 22542180,
				OriginalQuantity:        0.02,
				Price:                   62001.1,
				ReduceOnly:              true,
				Side:                    "SELL",
				PositionSide:            "BOTH",
				Status:                  "PARTIALLY_FILLED",
				Symbol:                  "BTCUSDT",
				TimeInForce:             "GTC",
				Type:                    "LIMIT",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				UpdateTime:              types.Time(time.UnixMilli(1566818724722)),
				PriceMatch:              "QUEUE",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.NewPortfolioMarginUSDTMarginedOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewPortfolioMarginUSDTMarginedOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewPortfolioMarginUSDTMarginedOrder should decode every field")
				return
			}
			assert.NotZero(t, result.OrderID, "NewPortfolioMarginUSDTMarginedOrder should return an order ID")
		})
	}
}

func TestCancelPortfolioMarginUSDTMarginedOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPortfolioMarginUSDTMarginedOrder(t.Context(), currency.EMPTYPAIR, 1234132, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelPortfolioMarginUSDTMarginedOrder must reject an empty symbol")
	_, err = e.CancelPortfolioMarginUSDTMarginedOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPortfolioMarginUSDTMarginedOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelPortfolioMarginUSDTMarginedOrder(t.Context(), usdtmTradablePair, 1234132, "myOrder1")
	require.NoError(t, err, "CancelPortfolioMarginUSDTMarginedOrder must not error")
	if mockTests {
		exp := &PortfolioMarginUSDTMarginedOrderResponse{
			ClientOrderID:           "myOrder1",
			CumulativeQuantity:      0.01,
			ExecutedQuantity:        0.01,
			OrderID:                 1234132,
			OriginalQuantity:        0.02,
			Price:                   62000.5,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "BOTH",
			Status:                  "CANCELED",
			Symbol:                  "BTCUSDT",
			TimeInForce:             "GTD",
			Type:                    "LIMIT",
			SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate:            types.Time(time.Unix(1693207680, 0)),
			UpdateTime:              types.Time(time.UnixMilli(1571110484038)),
			PriceMatch:              "NONE",
		}
		assert.Equal(t, exp, result, "CancelPortfolioMarginUSDTMarginedOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "CancelPortfolioMarginUSDTMarginedOrder should return the order ID")
}

func TestGetPortfolioMarginCoinMarginedAccountTradeList(t *testing.T) {
	t.Parallel()
	cmSymbol := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	startTime, endTime := getTime()
	_, err := e.GetPortfolioMarginCoinMarginedAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginCoinMarginedAccountTradeList must reject a nil request")
	_, err = e.GetPortfolioMarginCoinMarginedAccountTradeList(t.Context(), &PortfolioMarginCoinMarginedAccountTradeListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginCoinMarginedAccountTradeList must reject a request without a symbol or pair")
	_, err = e.GetPortfolioMarginCoinMarginedAccountTradeList(t.Context(), &PortfolioMarginCoinMarginedAccountTradeListRequest{Symbol: cmSymbol, Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, errSymbolWithPair, "GetPortfolioMarginCoinMarginedAccountTradeList must reject a symbol with a pair")
	_, err = e.GetPortfolioMarginCoinMarginedAccountTradeList(t.Context(), &PortfolioMarginCoinMarginedAccountTradeListRequest{Pair: currency.NewCode("BTCUSD"), FromID: 6})
	require.ErrorIs(t, err, errFromIDWithPair, "GetPortfolioMarginCoinMarginedAccountTradeList must reject fromId with a pair")
	_, err = e.GetPortfolioMarginCoinMarginedAccountTradeList(t.Context(), &PortfolioMarginCoinMarginedAccountTradeListRequest{Symbol: cmSymbol, FromID: 6, StartTime: startTime})
	require.ErrorIs(t, err, errFromIDWithTimeRange, "GetPortfolioMarginCoinMarginedAccountTradeList must reject fromId with a time range")
	_, err = e.GetPortfolioMarginCoinMarginedAccountTradeList(t.Context(), &PortfolioMarginCoinMarginedAccountTradeListRequest{Symbol: cmSymbol, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginCoinMarginedAccountTradeList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginCoinMarginedAccountTradeListRequest
		exp  []PortfolioMarginCoinMarginedAccountTradeResponse
	}{
		{name: "symbol", arg: &PortfolioMarginCoinMarginedAccountTradeListRequest{Symbol: coinmTradablePair, StartTime: startTime, EndTime: endTime, Limit: 10}, exp: []PortfolioMarginCoinMarginedAccountTradeResponse{
			{
				Symbol:              "BTCUSD_PERP",
				ID:                  6,
				OrderID:             28,
				Pair:                "BTCUSD",
				Side:                "BUY",
				Price:               8800,
				QuantityInContracts: 1,
				RealisedPNL:         0.0001,
				MarginAsset:         currency.BTC,
				BaseQuantity:        0.01136364,
				Commission:          0.00000454,
				CommissionAsset:     currency.BTC,
				Time:                types.Time(time.UnixMilli(1590743483586)),
				PositionSide:        "BOTH",
				Buyer:               true,
				Maker:               true,
			},
		}},
		{name: "pair", arg: &PortfolioMarginCoinMarginedAccountTradeListRequest{Pair: currency.NewCode("BTCUSD")}, exp: []PortfolioMarginCoinMarginedAccountTradeResponse{
			{
				Symbol:              "BTCUSD_250926",
				ID:                  7,
				OrderID:             29,
				Pair:                "BTCUSD",
				Side:                "BUY",
				Price:               8800,
				QuantityInContracts: 1,
				RealisedPNL:         0.0001,
				MarginAsset:         currency.BTC,
				BaseQuantity:        0.01136364,
				Commission:          0.00000454,
				CommissionAsset:     currency.BTC,
				Time:                types.Time(time.UnixMilli(1590743483586)),
				PositionSide:        "BOTH",
				Buyer:               true,
				Maker:               true,
			},
		}},
		{name: "fromID", arg: &PortfolioMarginCoinMarginedAccountTradeListRequest{Symbol: coinmTradablePair, FromID: 6}, exp: []PortfolioMarginCoinMarginedAccountTradeResponse{
			{
				Symbol:              "BTCUSD_PERP",
				ID:                  6,
				OrderID:             28,
				Pair:                "BTCUSD",
				Side:                "BUY",
				Price:               8800,
				QuantityInContracts: 1,
				RealisedPNL:         0.0001,
				MarginAsset:         currency.BTC,
				BaseQuantity:        0.01136364,
				Commission:          0.00000454,
				CommissionAsset:     currency.BTC,
				Time:                types.Time(time.UnixMilli(1590743483586)),
				PositionSide:        "BOTH",
				Buyer:               true,
				Maker:               true,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPortfolioMarginCoinMarginedAccountTradeList(t.Context(), tc.arg)
			require.NoError(t, err, "GetPortfolioMarginCoinMarginedAccountTradeList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPortfolioMarginCoinMarginedAccountTradeList should decode every field")
				return
			}
			assert.NotNil(t, result, "GetPortfolioMarginCoinMarginedAccountTradeList should return trades")
		})
	}
}

func TestGetPortfolioMarginCoinMarginedPositionAutoDeleveragingQuantileEstimation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCoinMarginedPositionAutoDeleveragingQuantileEstimation(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedPositionAutoDeleveragingQuantileEstimation must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedAutoDeleveragingQuantileEstimationResponse{
			{
				Symbol: "BTCUSD_PERP",
				AutoDeleveragingQuantile: PortfolioMarginCoinMarginedAutoDeleveragingQuantile{
					Long:  3,
					Short: 3,
					Hedge: 1,
				},
			},
			{
				Symbol: "BTCUSD_250926",
				AutoDeleveragingQuantile: PortfolioMarginCoinMarginedAutoDeleveragingQuantile{
					Long:  1,
					Short: 2,
					Both:  4,
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedPositionAutoDeleveragingQuantileEstimation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginCoinMarginedPositionAutoDeleveragingQuantileEstimation should return quantiles")
}

func TestPortfolioMarginMarginAccountBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginMarginAccountBorrow(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PortfolioMarginMarginAccountBorrow must reject an empty asset")
	_, err = e.PortfolioMarginMarginAccountBorrow(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PortfolioMarginMarginAccountBorrow must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginMarginAccountBorrow(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err, "PortfolioMarginMarginAccountBorrow must not error")
	if mockTests {
		exp := &PortfolioMarginTransactionResponse{
			TransactionID: 100000001,
		}
		assert.Equal(t, exp, result, "PortfolioMarginMarginAccountBorrow should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "PortfolioMarginMarginAccountBorrow should return a transaction ID")
}

func TestPortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must reject a nil request")
	arg := &PortfolioMarginMarginOneCancelsTheOtherOrderRequest{ListClientOrderID: "JYVpp3F0f5CAG15DhtrqLp"}
	_, err = e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must reject an invalid side")
	arg.Side = order.Sell
	_, err = e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must reject a zero quantity")
	arg.Quantity = 0.1
	_, err = e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must reject a zero price")
	arg.Price = 70000
	_, err = e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must reject a zero stop price")
	arg.StopPrice = 60000
	arg.StopLimitPrice = 59900
	_, err = e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStopLimitTimeInForceRequired, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must reject a stop limit price without a time in force")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder(t.Context(), &PortfolioMarginMarginOneCancelsTheOtherOrderRequest{
		Symbol:               marginTradablePair,
		ListClientOrderID:    "JYVpp3F0f5CAG15DhtrqLp",
		Side:                 order.Sell,
		Quantity:             0.1,
		LimitClientOrderID:   "Kk7sqHb9J6mJWTMDVW7Vos",
		Price:                70000,
		LimitIcebergQuantity: 0.01,
		StopClientOrderID:    "xTXKaGYd4bluPVp78IVRvl",
		StopPrice:            60000,
		StopLimitPrice:       59900,
		StopIcebergQuantity:  0.01,
		StopLimitTimeInForce: "GTC",
		NewOrderResponseType: "RESULT",
		SideEffectType:       "MARGIN_BUY",
	})
	require.NoError(t, err, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder must not error")
	if mockTests {
		exp := &PortfolioMarginNewMarginOneCancelsTheOtherOrderResponse{
			OrderListID:       1,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "JYVpp3F0f5CAG15DhtrqLp",
			TransactionTime:   types.Time(time.UnixMilli(1563417480525)),
			Symbol:            "BTCUSDT",
			Orders: []PortfolioMarginMarginOneCancelsTheOtherOrderLeg{
				{
					Symbol:        "BTCUSDT",
					OrderID:       2,
					ClientOrderID: "xTXKaGYd4bluPVp78IVRvl",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       3,
					ClientOrderID: "Kk7sqHb9J6mJWTMDVW7Vos",
				},
			},
			MarginBuyBorrowAmount: 0.1,
			MarginBuyBorrowAsset:  currency.BTC,
			OrderReports: []PortfolioMarginNewMarginOneCancelsTheOtherOrderReport{
				{
					Symbol:                  "BTCUSDT",
					OrderID:                 2,
					OrderListID:             1,
					ClientOrderID:           "xTXKaGYd4bluPVp78IVRvl",
					TransactTime:            types.Time(time.UnixMilli(1563417480525)),
					Price:                   59900,
					OriginalQuantity:        0.1,
					ExecutedQuantity:        0.01,
					CumulativeQuoteQuantity: 599,
					Status:                  "PARTIALLY_FILLED",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					StopPrice:               60000,
				},
			},
		}
		assert.Equal(t, exp, result, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderListID, "PortfolioMarginMarginAccountNewOneCancelsTheOtherOrder should return an order list ID")
}

func TestPortfolioMarginMarginAccountRepay(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginMarginAccountRepay(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PortfolioMarginMarginAccountRepay must reject an empty asset")
	_, err = e.PortfolioMarginMarginAccountRepay(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PortfolioMarginMarginAccountRepay must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginMarginAccountRepay(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err, "PortfolioMarginMarginAccountRepay must not error")
	if mockTests {
		exp := &PortfolioMarginTransactionResponse{
			TransactionID: 100000002,
		}
		assert.Equal(t, exp, result, "PortfolioMarginMarginAccountRepay should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "PortfolioMarginMarginAccountRepay should return a transaction ID")
}

func TestGetPortfolioMarginMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginMarginAccountTradeList must reject a nil request")
	_, err = e.GetPortfolioMarginMarginAccountTradeList(t.Context(), &PortfolioMarginMarginTradeListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginMarginAccountTradeList must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginMarginAccountTradeList(t.Context(), &PortfolioMarginMarginTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginMarginAccountTradeList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginMarginAccountTradeList(t.Context(), &PortfolioMarginMarginTradeListRequest{Symbol: marginTradablePair, OrderID: 39324, StartTime: startTime, EndTime: endTime, FromID: 1, Limit: 10})
	require.NoError(t, err, "GetPortfolioMarginMarginAccountTradeList must not error")
	if mockTests {
		exp := []PortfolioMarginMarginTradeResponse{
			{
				Commission:      0.00006,
				CommissionAsset: currency.BTC,
				ID:              34,
				IsBestMatch:     true,
				IsBuyer:         true,
				IsMaker:         true,
				OrderID:         39324,
				Price:           62000.5,
				Quantity:        0.003,
				Symbol:          "BTCUSDT",
				Time:            types.Time(time.Unix(1744110000, 0)),
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginAccountTradeList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginAccountTradeList should return trades")
}

func TestGetAllPortfolioMarginCoinMarginedConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPortfolioMarginCoinMarginedConditionalOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPortfolioMarginCoinMarginedConditionalOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetAllPortfolioMarginCoinMarginedConditionalOrders(t.Context(), &PortfolioMarginCoinMarginedConditionalOrdersRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllPortfolioMarginCoinMarginedConditionalOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllPortfolioMarginCoinMarginedConditionalOrders(t.Context(), &PortfolioMarginCoinMarginedConditionalOrdersRequest{Symbol: coinmTradablePair, StrategyID: 123445, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetAllPortfolioMarginCoinMarginedConditionalOrders must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedConditionalOrderRecordResponse{
			{
				NewClientStrategyID: "abc",
				StrategyID:          123445,
				StrategyStatus:      "TRIGGERED",
				StrategyType:        "TRAILING_STOP_MARKET",
				OriginalQuantity:    0.4,
				Price:               9500,
				ReduceOnly:          true,
				Side:                "BUY",
				PositionSide:        "BOTH",
				StopPrice:           9300,
				Symbol:              "BTCUSD_PERP",
				TimeInForce:         "GTC",
				ActivatePrice:       9020,
				PriceRate:           0.3,
				BookTime:            types.Time(time.UnixMilli(1566818724710)),
				UpdateTime:          types.Time(time.UnixMilli(1566818724722)),
				OrderID:             12123343534,
				Status:              "NEW",
				TriggerTime:         types.Time(time.UnixMilli(1566818724750)),
				Type:                "MARKET",
			},
		}
		assert.Equal(t, exp, result, "GetAllPortfolioMarginCoinMarginedConditionalOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllPortfolioMarginCoinMarginedConditionalOrders should return orders")
}

func TestGetAllPortfolioMarginCoinMarginedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPortfolioMarginCoinMarginedOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPortfolioMarginCoinMarginedOrders must reject a nil request")
	_, err = e.GetAllPortfolioMarginCoinMarginedOrders(t.Context(), &PortfolioMarginCoinMarginedOrdersRequest{Limit: 20})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAllPortfolioMarginCoinMarginedOrders must reject a request without a symbol or pair")
	startTime, endTime := getTime()
	_, err = e.GetAllPortfolioMarginCoinMarginedOrders(t.Context(), &PortfolioMarginCoinMarginedOrdersRequest{Pair: currency.NewCode("BTCUSD"), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllPortfolioMarginCoinMarginedOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllPortfolioMarginCoinMarginedOrders(t.Context(), &PortfolioMarginCoinMarginedOrdersRequest{Symbol: coinmTradablePair, Pair: currency.NewCode("BTCUSD"), StartTime: startTime, EndTime: endTime, Limit: 20})
	require.NoError(t, err, "GetAllPortfolioMarginCoinMarginedOrders must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedOrderDetailResponse{
			{
				AveragePrice:     9500.5,
				ClientOrderID:    "abc",
				CumulativeBase:   0.01052576,
				ExecutedQuantity: 1,
				OrderID:          1234,
				OriginalQuantity: 2,
				OriginalType:     "LIMIT",
				Price:            9500.5,
				ReduceOnly:       true,
				Side:             "BUY",
				Status:           "PARTIALLY_FILLED",
				Symbol:           "BTCUSD_PERP",
				Pair:             "BTCUSD",
				PositionSide:     "BOTH",
				Time:             types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:      "GTC",
				Type:             "LIMIT",
				UpdateTime:       types.Time(time.UnixMilli(1579276756075)),
			},
		}
		assert.Equal(t, exp, result, "GetAllPortfolioMarginCoinMarginedOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllPortfolioMarginCoinMarginedOrders should return orders")
}

func TestGetAllPortfolioMarginCoinMarginedOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllPortfolioMarginCoinMarginedOpenConditionalOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetAllPortfolioMarginCoinMarginedOpenConditionalOrders must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedConditionalOrderResponse{
			{
				NewClientStrategyID: "abc",
				StrategyID:          123445,
				StrategyStatus:      "NEW",
				StrategyType:        "TRAILING_STOP_MARKET",
				OriginalQuantity:    0.4,
				Price:               9500,
				ReduceOnly:          true,
				Side:                "BUY",
				PositionSide:        "BOTH",
				StopPrice:           9300,
				Symbol:              "BTCUSD_PERP",
				TimeInForce:         "GTC",
				ActivatePrice:       9020,
				PriceRate:           0.3,
				BookTime:            types.Time(time.UnixMilli(1566818724710)),
				UpdateTime:          types.Time(time.UnixMilli(1566818724722)),
			},
		}
		assert.Equal(t, exp, result, "GetAllPortfolioMarginCoinMarginedOpenConditionalOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllPortfolioMarginCoinMarginedOpenConditionalOrders should return orders")
}

func TestGetAllPortfolioMarginCoinMarginedOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllPortfolioMarginCoinMarginedOpenOrders(t.Context(), coinmTradablePair, currency.NewCode("BTCUSD"))
	require.NoError(t, err, "GetAllPortfolioMarginCoinMarginedOpenOrders must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedOrderDetailResponse{
			{
				AveragePrice:     9500.5,
				ClientOrderID:    "abc",
				CumulativeBase:   0.01052576,
				ExecutedQuantity: 1,
				OrderID:          1234,
				OriginalQuantity: 2,
				OriginalType:     "LIMIT",
				Price:            9500.5,
				ReduceOnly:       true,
				Side:             "BUY",
				Status:           "PARTIALLY_FILLED",
				Symbol:           "BTCUSD_PERP",
				Pair:             "BTCUSD",
				PositionSide:     "BOTH",
				Time:             types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:      "GTC",
				Type:             "LIMIT",
				UpdateTime:       types.Time(time.UnixMilli(1579276756075)),
			},
		}
		assert.Equal(t, exp, result, "GetAllPortfolioMarginCoinMarginedOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllPortfolioMarginCoinMarginedOpenOrders should return orders")
}

func TestGetAllPortfolioMarginUSDTMarginedOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllPortfolioMarginUSDTMarginedOpenOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetAllPortfolioMarginUSDTMarginedOpenOrders must not error")
	if mockTests {
		exp := []PortfolioMarginUSDTMarginedOrderDetailResponse{
			{
				AveragePrice:            62000.5,
				ClientOrderID:           "abc",
				CumulativeQuote:         620.005,
				ExecutedQuantity:        0.01,
				OrderID:                 1234,
				OriginalQuantity:        0.02,
				OriginalType:            "LIMIT",
				Price:                   62000.5,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "BOTH",
				Status:                  "PARTIALLY_FILLED",
				Symbol:                  "BTCUSDT",
				Time:                    types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:             "GTD",
				Type:                    "LIMIT",
				UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
				SelfTradePreventionMode: "EXPIRE_MAKER",
				GoodTillDate:            types.Time(time.Unix(1693207680, 0)),
				PriceMatch:              "QUEUE",
			},
		}
		assert.Equal(t, exp, result, "GetAllPortfolioMarginUSDTMarginedOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllPortfolioMarginUSDTMarginedOpenOrders should return orders")
}

func TestGetAllPortfolioMarginMarginAccountOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPortfolioMarginMarginAccountOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPortfolioMarginMarginAccountOrders must reject a nil request")
	_, err = e.GetAllPortfolioMarginMarginAccountOrders(t.Context(), &PortfolioMarginMarginOrdersRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAllPortfolioMarginMarginAccountOrders must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetAllPortfolioMarginMarginAccountOrders(t.Context(), &PortfolioMarginMarginOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllPortfolioMarginMarginAccountOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllPortfolioMarginMarginAccountOrders(t.Context(), &PortfolioMarginMarginOrdersRequest{Symbol: marginTradablePair, OrderID: 1234, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetAllPortfolioMarginMarginAccountOrders must not error")
	if mockTests {
		exp := []PortfolioMarginMarginOrderResponse{
			{
				ClientOrderID:           "ZwfQzuDIGpceVhKW5DvCmO",
				CumulativeQuoteQuantity: 1500.5,
				ExecutedQuantity:        0.025,
				IcebergQuantity:         0.01,
				IsWorking:               true,
				OrderID:                 12434,
				OriginalQuantity:        0.3,
				Price:                   60020,
				Side:                    "SELL",
				Status:                  "PARTIALLY_FILLED",
				StopPrice:               60000,
				Symbol:                  "BTCUSDT",
				Time:                    types.Time(time.UnixMilli(1562133008725)),
				TimeInForce:             "GTC",
				Type:                    "STOP_LOSS_LIMIT",
				UpdateTime:              types.Time(time.UnixMilli(1562133008725)),
				AccountID:               152950866,
				SelfTradePreventionMode: "EXPIRE_TAKER",
				PreventedMatchID:        1,
				PreventedQuantity:       0.002,
			},
		}
		assert.Equal(t, exp, result, "GetAllPortfolioMarginMarginAccountOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllPortfolioMarginMarginAccountOrders should return orders")
}

func TestGetAllPortfolioMarginUSDTMarginedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPortfolioMarginUSDTMarginedOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllPortfolioMarginUSDTMarginedOrders must reject a nil request")
	_, err = e.GetAllPortfolioMarginUSDTMarginedOrders(t.Context(), &PortfolioMarginUSDTMarginedOrdersRequest{Limit: 100})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAllPortfolioMarginUSDTMarginedOrders must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetAllPortfolioMarginUSDTMarginedOrders(t.Context(), &PortfolioMarginUSDTMarginedOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllPortfolioMarginUSDTMarginedOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllPortfolioMarginUSDTMarginedOrders(t.Context(), &PortfolioMarginUSDTMarginedOrdersRequest{Symbol: usdtmTradablePair, OrderID: 1, StartTime: startTime, EndTime: endTime, Limit: 100})
	require.NoError(t, err, "GetAllPortfolioMarginUSDTMarginedOrders must not error")
	if mockTests {
		exp := []PortfolioMarginUSDTMarginedOrderDetailResponse{
			{
				AveragePrice:            62000.5,
				ClientOrderID:           "abc",
				CumulativeQuote:         620.005,
				ExecutedQuantity:        0.01,
				OrderID:                 1234,
				OriginalQuantity:        0.02,
				OriginalType:            "LIMIT",
				Price:                   62000.5,
				ReduceOnly:              true,
				Side:                    "BUY",
				PositionSide:            "BOTH",
				Status:                  "PARTIALLY_FILLED",
				Symbol:                  "BTCUSDT",
				Time:                    types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:             "GTD",
				Type:                    "LIMIT",
				UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
				SelfTradePreventionMode: "EXPIRE_MAKER",
				GoodTillDate:            types.Time(time.Unix(1693207680, 0)),
				PriceMatch:              "QUEUE",
			},
		}
		assert.Equal(t, exp, result, "GetAllPortfolioMarginUSDTMarginedOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllPortfolioMarginUSDTMarginedOrders should return orders")
}

func TestGetPortfolioMarginCoinMarginedConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginCoinMarginedConditionalOrderHistory(t.Context(), currency.EMPTYPAIR, 123432423, "abc")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginCoinMarginedConditionalOrderHistory must reject an empty symbol")
	_, err = e.GetPortfolioMarginCoinMarginedConditionalOrderHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginCoinMarginedConditionalOrderHistory must reject a request without a strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCoinMarginedConditionalOrderHistory(t.Context(), coinmTradablePair, 123432423, "abc")
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedConditionalOrderHistory must not error")
	if mockTests {
		exp := &PortfolioMarginCoinMarginedConditionalOrderHistoryResponse{
			NewClientStrategyID: "abc",
			StrategyID:          123432423,
			StrategyStatus:      "TRIGGERED",
			StrategyType:        "TRAILING_STOP_MARKET",
			OriginalQuantity:    0.4,
			Price:               9500,
			ReduceOnly:          true,
			Side:                "BUY",
			PositionSide:        "BOTH",
			StopPrice:           9300,
			Symbol:              "BTCUSD_PERP",
			TimeInForce:         "GTC",
			ActivatePrice:       9020,
			PriceRate:           0.3,
			BookTime:            types.Time(time.UnixMilli(1566818724710)),
			UpdateTime:          types.Time(time.UnixMilli(1566818724722)),
			OrderID:             12123343534,
			Status:              "NEW",
			TriggerTime:         types.Time(time.UnixMilli(1566818724750)),
			Type:                "MARKET",
			WorkingType:         "CONTRACT_PRICE",
			PriceProtect:        true,
			PriceMatch:          "NONE",
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedConditionalOrderHistory should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "GetPortfolioMarginCoinMarginedConditionalOrderHistory should return the strategy ID")
}

func TestGetOpenPortfolioMarginCoinMarginedConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenPortfolioMarginCoinMarginedConditionalOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOpenPortfolioMarginCoinMarginedConditionalOrder must reject an empty symbol")
	_, err = e.GetOpenPortfolioMarginCoinMarginedConditionalOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOpenPortfolioMarginCoinMarginedConditionalOrder must reject a request without a strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetOpenPortfolioMarginCoinMarginedConditionalOrder(t.Context(), coinmTradablePair, 1234, "")
	require.NoError(t, err, "GetOpenPortfolioMarginCoinMarginedConditionalOrder must not error")
	if mockTests {
		exp := &PortfolioMarginCoinMarginedConditionalOrderResponse{
			NewClientStrategyID: "abc",
			StrategyID:          1234,
			StrategyStatus:      "NEW",
			StrategyType:        "TRAILING_STOP_MARKET",
			OriginalQuantity:    0.4,
			Price:               9500,
			ReduceOnly:          true,
			Side:                "BUY",
			PositionSide:        "BOTH",
			StopPrice:           9300,
			Symbol:              "BTCUSD_PERP",
			TimeInForce:         "GTC",
			ActivatePrice:       9020,
			PriceRate:           0.3,
			BookTime:            types.Time(time.UnixMilli(1566818724710)),
			UpdateTime:          types.Time(time.UnixMilli(1566818724722)),
		}
		assert.Equal(t, exp, result, "GetOpenPortfolioMarginCoinMarginedConditionalOrder should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "GetOpenPortfolioMarginCoinMarginedConditionalOrder should return the strategy ID")
}

func TestGetPortfolioMarginCoinMarginedOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginCoinMarginedOpenOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginCoinMarginedOpenOrder must reject an empty symbol")
	_, err = e.GetPortfolioMarginCoinMarginedOpenOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginCoinMarginedOpenOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCoinMarginedOpenOrder(t.Context(), coinmTradablePair, 1234, "")
	require.NoError(t, err, "GetPortfolioMarginCoinMarginedOpenOrder must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedOrderDetailResponse{
			{
				AveragePrice:     9500.5,
				ClientOrderID:    "abc",
				CumulativeBase:   0.01052576,
				ExecutedQuantity: 1,
				OrderID:          1234,
				OriginalQuantity: 2,
				OriginalType:     "LIMIT",
				Price:            9500.5,
				ReduceOnly:       true,
				Side:             "BUY",
				Status:           "PARTIALLY_FILLED",
				Symbol:           "BTCUSD_PERP",
				Pair:             "BTCUSD",
				PositionSide:     "BOTH",
				Time:             types.Time(time.UnixMilli(1579276756075)),
				TimeInForce:      "GTC",
				Type:             "LIMIT",
				UpdateTime:       types.Time(time.UnixMilli(1579276756075)),
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCoinMarginedOpenOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPortfolioMarginCoinMarginedOpenOrder should return the order")
}

func TestGetPortfolioMarginCurrentMarginOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginCurrentMarginOpenOrders(t.Context(), marginTradablePair)
	require.NoError(t, err, "GetPortfolioMarginCurrentMarginOpenOrders must not error")
	if mockTests {
		exp := []PortfolioMarginMarginOrderResponse{
			{
				ClientOrderID:           "ZwfQzuDIGpceVhKW5DvCmO",
				CumulativeQuoteQuantity: 1500.5,
				ExecutedQuantity:        0.025,
				IcebergQuantity:         0.01,
				IsWorking:               true,
				OrderID:                 12434,
				OriginalQuantity:        0.3,
				Price:                   60020,
				Side:                    "SELL",
				Status:                  "PARTIALLY_FILLED",
				StopPrice:               60000,
				Symbol:                  "BTCUSDT",
				Time:                    types.Time(time.UnixMilli(1562133008725)),
				TimeInForce:             "GTC",
				Type:                    "STOP_LOSS_LIMIT",
				UpdateTime:              types.Time(time.UnixMilli(1562133008725)),
				AccountID:               152950866,
				SelfTradePreventionMode: "EXPIRE_TAKER",
				PreventedMatchID:        1,
				PreventedQuantity:       0.002,
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCurrentMarginOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginCurrentMarginOpenOrders should return orders")
}

func TestGetPortfolioMarginUSDTMarginedOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginUSDTMarginedOpenOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginUSDTMarginedOpenOrder must reject an empty symbol")
	_, err = e.GetPortfolioMarginUSDTMarginedOpenOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginUSDTMarginedOpenOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedOpenOrder(t.Context(), usdtmTradablePair, 1234, "")
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedOpenOrder must not error")
	if mockTests {
		exp := &PortfolioMarginUSDTMarginedOrderDetailResponse{
			AveragePrice:            62000.5,
			ClientOrderID:           "abc",
			CumulativeQuote:         620.005,
			ExecutedQuantity:        0.01,
			OrderID:                 1234,
			OriginalQuantity:        0.02,
			OriginalType:            "LIMIT",
			Price:                   62000.5,
			ReduceOnly:              true,
			Side:                    "BUY",
			PositionSide:            "BOTH",
			Status:                  "PARTIALLY_FILLED",
			Symbol:                  "BTCUSDT",
			Time:                    types.Time(time.UnixMilli(1579276756075)),
			TimeInForce:             "GTD",
			Type:                    "LIMIT",
			UpdateTime:              types.Time(time.UnixMilli(1579276756075)),
			SelfTradePreventionMode: "EXPIRE_MAKER",
			GoodTillDate:            types.Time(time.Unix(1693207680, 0)),
			PriceMatch:              "QUEUE",
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedOpenOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetPortfolioMarginUSDTMarginedOpenOrder should return the order ID")
}

func TestGetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders(t.Context(), &PortfolioMarginMarginAllOneCancelsTheOtherOrdersRequest{StartTime: endTime, EndTime: startTime, FromID: 1, Limit: 100})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders(t.Context(), &PortfolioMarginMarginAllOneCancelsTheOtherOrdersRequest{StartTime: startTime, EndTime: endTime, FromID: 1, Limit: 100})
	require.NoError(t, err, "GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders must not error")
	if mockTests {
		exp := []PortfolioMarginMarginOneCancelsTheOtherOrderResponse{
			{
				OrderListID:       27,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
				TransactionTime:   types.Time(time.UnixMilli(1565245656253)),
				Symbol:            "BTCUSDT",
				Orders: []PortfolioMarginMarginOneCancelsTheOtherOrderLeg{
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
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginAccountAllOneCancelsTheOtherOrders should return order lists")
}

func TestGetPortfolioMarginMarginAccountsOpenOneCancelsTheOtherOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginMarginAccountsOpenOneCancelsTheOtherOrders(t.Context())
	require.NoError(t, err, "GetPortfolioMarginMarginAccountsOpenOneCancelsTheOtherOrders must not error")
	if mockTests {
		exp := []PortfolioMarginMarginOneCancelsTheOtherOrderResponse{
			{
				OrderListID:       31,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
				TransactionTime:   types.Time(time.UnixMilli(1565245656253)),
				Symbol:            "BTCUSDT",
				Orders: []PortfolioMarginMarginOneCancelsTheOtherOrderLeg{
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
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginMarginAccountsOpenOneCancelsTheOtherOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginMarginAccountsOpenOneCancelsTheOtherOrders should return order lists")
}

func TestGetUsersPortfolioMarginCoinMarginedForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersPortfolioMarginCoinMarginedForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUsersPortfolioMarginCoinMarginedForceOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUsersPortfolioMarginCoinMarginedForceOrders(t.Context(), &PortfolioMarginForceOrdersRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUsersPortfolioMarginCoinMarginedForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUsersPortfolioMarginCoinMarginedForceOrders(t.Context(), &PortfolioMarginForceOrdersRequest{Symbol: coinmTradablePair, AutoCloseType: "ADL", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUsersPortfolioMarginCoinMarginedForceOrders must not error")
	if mockTests {
		exp := []PortfolioMarginCoinMarginedForceOrderResponse{
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
				TimeInForce:      "IOC",
				Type:             "LIMIT",
				ReduceOnly:       true,
				Side:             "SELL",
				PositionSide:     "BOTH",
				OriginalType:     "LIMIT",
				Time:             types.Time(time.UnixMilli(1596542005019)),
				UpdateTime:       types.Time(time.UnixMilli(1596542005050)),
			},
		}
		assert.Equal(t, exp, result, "GetUsersPortfolioMarginCoinMarginedForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersPortfolioMarginCoinMarginedForceOrders should return orders")
}

func TestGetUsersPortfolioMarginMarginForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersPortfolioMarginMarginForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUsersPortfolioMarginMarginForceOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUsersPortfolioMarginMarginForceOrders(t.Context(), &PortfolioMarginMarginForceOrdersRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUsersPortfolioMarginMarginForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersPortfolioMarginMarginForceOrders(t.Context(), &PortfolioMarginMarginForceOrdersRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 5})
	require.NoError(t, err, "GetUsersPortfolioMarginMarginForceOrders must not error")
	if mockTests {
		exp := &PortfolioMarginMarginForceOrderResponse{
			Rows: []PortfolioMarginMarginForceOrder{
				{
					AveragePrice:     0.00388359,
					ExecutedQuantity: 31.39,
					OrderID:          180015097,
					Price:            0.0038811,
					Quantity:         31.39,
					Side:             "SELL",
					Symbol:           "BNBBTC",
					TimeInForce:      "GTC",
					UpdatedTime:      types.Time(time.UnixMilli(1558941374745)),
				},
			},
			Total: 1,
		}
		assert.Equal(t, exp, result, "GetUsersPortfolioMarginMarginForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersPortfolioMarginMarginForceOrders should return orders")
}

func TestGetUsersPortfolioMarginUSDTMarginedForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersPortfolioMarginUSDTMarginedForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUsersPortfolioMarginUSDTMarginedForceOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUsersPortfolioMarginUSDTMarginedForceOrders(t.Context(), &PortfolioMarginForceOrdersRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUsersPortfolioMarginUSDTMarginedForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUsersPortfolioMarginUSDTMarginedForceOrders(t.Context(), &PortfolioMarginForceOrdersRequest{Symbol: usdtmTradablePair, AutoCloseType: "LIQUIDATION", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUsersPortfolioMarginUSDTMarginedForceOrders must not error")
	if mockTests {
		exp := []PortfolioMarginUSDTMarginedForceOrderResponse{
			{
				OrderID:          6071832819,
				Symbol:           "BTCUSDT",
				Status:           "FILLED",
				ClientOrderID:    "autoclose-1596107620040000020",
				Price:            10871.09,
				AveragePrice:     10913.21,
				OriginalQuantity: 0.001,
				ExecutedQuantity: 0.001,
				CumulativeQuote:  10.91321,
				TimeInForce:      "IOC",
				Type:             "LIMIT",
				ReduceOnly:       true,
				Side:             "SELL",
				PositionSide:     "BOTH",
				OriginalType:     "LIMIT",
				Time:             types.Time(time.UnixMilli(1596107620044)),
				UpdateTime:       types.Time(time.UnixMilli(1596107620087)),
			},
		}
		assert.Equal(t, exp, result, "GetUsersPortfolioMarginUSDTMarginedForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersPortfolioMarginUSDTMarginedForceOrders should return orders")
}

func TestGetPortfolioMarginUSDTMarginedAccountTradeList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetPortfolioMarginUSDTMarginedAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginUSDTMarginedAccountTradeList must reject a nil request")
	_, err = e.GetPortfolioMarginUSDTMarginedAccountTradeList(t.Context(), &PortfolioMarginUSDTMarginedAccountTradeListRequest{StartTime: startTime, EndTime: endTime})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginUSDTMarginedAccountTradeList must reject an empty symbol")
	_, err = e.GetPortfolioMarginUSDTMarginedAccountTradeList(t.Context(), &PortfolioMarginUSDTMarginedAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), FromID: 1, EndTime: endTime})
	require.ErrorIs(t, err, errFromIDWithTimeRange, "GetPortfolioMarginUSDTMarginedAccountTradeList must reject fromId with a time range")
	_, err = e.GetPortfolioMarginUSDTMarginedAccountTradeList(t.Context(), &PortfolioMarginUSDTMarginedAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginUSDTMarginedAccountTradeList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginUSDTMarginedAccountTradeListRequest
		exp  []PortfolioMarginUSDTMarginedAccountTradeResponse
	}{
		{name: "timeRange", arg: &PortfolioMarginUSDTMarginedAccountTradeListRequest{Symbol: usdtmTradablePair, StartTime: startTime, EndTime: endTime, Limit: 10}, exp: []PortfolioMarginUSDTMarginedAccountTradeResponse{
			{
				Symbol:          "BTCUSDT",
				ID:              67880589,
				OrderID:         270093109,
				Side:            "BUY",
				Price:           28511,
				Quantity:        0.01,
				RealisedPNL:     2.585,
				QuoteQuantity:   285.11,
				Commission:      0.114044,
				CommissionAsset: currency.USDT,
				Time:            types.Time(time.UnixMilli(1680688557875)),
				Buyer:           true,
				Maker:           true,
				PositionSide:    "BOTH",
			},
		}},
		{name: "fromID", arg: &PortfolioMarginUSDTMarginedAccountTradeListRequest{Symbol: usdtmTradablePair, FromID: 67880589}, exp: []PortfolioMarginUSDTMarginedAccountTradeResponse{
			{
				Symbol:          "BTCUSDT",
				ID:              67880589,
				OrderID:         270093109,
				Side:            "BUY",
				Price:           28511,
				Quantity:        0.01,
				RealisedPNL:     2.585,
				QuoteQuantity:   285.11,
				Commission:      0.114044,
				CommissionAsset: currency.USDT,
				Time:            types.Time(time.UnixMilli(1680688557875)),
				Buyer:           true,
				Maker:           true,
				PositionSide:    "BOTH",
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPortfolioMarginUSDTMarginedAccountTradeList(t.Context(), tc.arg)
			require.NoError(t, err, "GetPortfolioMarginUSDTMarginedAccountTradeList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPortfolioMarginUSDTMarginedAccountTradeList should decode every field")
				return
			}
			assert.NotNil(t, result, "GetPortfolioMarginUSDTMarginedAccountTradeList should return trades")
		})
	}
}

func TestGetPortfolioMarginUSDTMarginedPositionAutoDeleveragingQuantileEstimation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedPositionAutoDeleveragingQuantileEstimation(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedPositionAutoDeleveragingQuantileEstimation must not error")
	if mockTests {
		exp := []PortfolioMarginUSDTMarginedAutoDeleveragingQuantileEstimationResponse{
			{
				Symbol: "BTCUSDT",
				AutoDeleveragingQuantile: PortfolioMarginUSDTMarginedAutoDeleveragingQuantile{
					Long:  3,
					Short: 3,
					Both:  2,
				},
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedPositionAutoDeleveragingQuantileEstimation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUSDTMarginedPositionAutoDeleveragingQuantileEstimation should return quantiles")
}

func TestNewPortfolioMarginUSDTMarginedAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject a nil request")
	arg := &PortfolioMarginUSDTMarginedAlgoOrderRequest{TriggerPrice: 74000}
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject an invalid side")
	arg.Side = order.Sell
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject an empty order type")
	arg.OrderType = "TAKE_PROFIT"
	arg.Price = 75000
	arg.PriceMatch = "QUEUE"
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject a price with priceMatch")
	arg.PriceMatch = ""
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errAlgoTypeRequired, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject an empty algo type")
	arg.AlgoType = "CONDITIONAL"
	arg.TimeInForce = "GTD"
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errGoodTillDateRequired, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject a GTD order without a good till date")
	arg.TimeInForce = ""
	arg.ClosePosition = true
	arg.Quantity = 0.01
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionConflict, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject closePosition with a quantity")
	arg.Quantity = 0
	arg.ReduceOnly = true
	_, err = e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionConflict, "NewPortfolioMarginUSDTMarginedAlgoOrder must reject closePosition with reduceOnly")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *PortfolioMarginUSDTMarginedAlgoOrderRequest
		exp  *PortfolioMarginUSDTMarginedAlgoOrderResponse
	}{
		{
			name: "takeProfit",
			arg:  &PortfolioMarginUSDTMarginedAlgoOrderRequest{AlgoType: "CONDITIONAL", Symbol: usdtmTradablePair, Side: order.Sell, PositionSide: "BOTH", OrderType: "TAKE_PROFIT", TimeInForce: "GTD", Quantity: 0.01, Price: 75000, TriggerPrice: 74000, WorkingType: "MARK_PRICE", PriceProtect: true, ReduceOnly: true, ClientAlgoID: "tpOrder1", NewOrderResponseType: "ACK", SelfTradePreventionMode: "EXPIRE_MAKER", GoodTillDate: time.UnixMilli(1750500000000)},
			exp: &PortfolioMarginUSDTMarginedAlgoOrderResponse{
				AlgoID:                  2146760,
				ClientAlgoID:            "tpOrder1",
				AlgoType:                "CONDITIONAL",
				OrderType:               "TAKE_PROFIT",
				Symbol:                  "BTCUSDT",
				Side:                    "SELL",
				PositionSide:            "BOTH",
				TimeInForce:             "GTD",
				Quantity:                0.01,
				AlgoStatus:              "NEW",
				TriggerPrice:            74000,
				Price:                   75000,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				WorkingType:             "MARK_PRICE",
				PriceMatch:              "NONE",
				PriceProtect:            true,
				ReduceOnly:              true,
				CreateTime:              types.Time(time.UnixMilli(1750485492076)),
				UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
				GoodTillDate:            types.Time(time.Unix(1750500000, 0)),
			},
		},
		{
			name: "trailingStopMarket",
			arg:  &PortfolioMarginUSDTMarginedAlgoOrderRequest{AlgoType: "CONDITIONAL", Symbol: usdtmTradablePair, Side: order.Sell, OrderType: "TRAILING_STOP_MARKET", Quantity: 0.01, ActivatePrice: 76000, CallbackRate: 1},
			exp: &PortfolioMarginUSDTMarginedAlgoOrderResponse{
				AlgoID:                  2146761,
				ClientAlgoID:            "trailOrder1",
				AlgoType:                "CONDITIONAL",
				OrderType:               "TRAILING_STOP_MARKET",
				Symbol:                  "BTCUSDT",
				Side:                    "SELL",
				PositionSide:            "BOTH",
				TimeInForce:             "GTC",
				Quantity:                0.01,
				AlgoStatus:              "NEW",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				WorkingType:             "CONTRACT_PRICE",
				PriceMatch:              "NONE",
				ActivatePrice:           76000,
				CallbackRate:            1,
				CreateTime:              types.Time(time.UnixMilli(1750485492076)),
				UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
			},
		},
		{
			name: "closePosition",
			arg:  &PortfolioMarginUSDTMarginedAlgoOrderRequest{AlgoType: "CONDITIONAL", Symbol: usdtmTradablePair, Side: order.Sell, OrderType: "STOP_MARKET", TriggerPrice: 60000, ClosePosition: true},
			exp: &PortfolioMarginUSDTMarginedAlgoOrderResponse{
				AlgoID:                  2146762,
				ClientAlgoID:            "stopOrder1",
				AlgoType:                "CONDITIONAL",
				OrderType:               "STOP_MARKET",
				Symbol:                  "BTCUSDT",
				Side:                    "SELL",
				PositionSide:            "BOTH",
				TimeInForce:             "GTC",
				AlgoStatus:              "TRIGGERED",
				TriggerPrice:            60000,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				WorkingType:             "CONTRACT_PRICE",
				PriceMatch:              "NONE",
				ClosePosition:           true,
				CreateTime:              types.Time(time.UnixMilli(1750485492076)),
				UpdateTime:              types.Time(time.UnixMilli(1750485492076)),
				TriggerTime:             types.Time(time.UnixMilli(1750485492100)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.NewPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewPortfolioMarginUSDTMarginedAlgoOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewPortfolioMarginUSDTMarginedAlgoOrder should decode every field")
				return
			}
			assert.NotZero(t, result.AlgoID, "NewPortfolioMarginUSDTMarginedAlgoOrder should return an algo ID")
		})
	}
}

func TestCancelPortfolioMarginUSDTMarginedAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelPortfolioMarginUSDTMarginedAlgoOrder must reject a request without an algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), 2146760, "tpOrder1")
	require.NoError(t, err, "CancelPortfolioMarginUSDTMarginedAlgoOrder must not error")
	if mockTests {
		exp := &CancelPortfolioMarginUSDTMarginedAlgoOrderResponse{
			Complete: true,
		}
		assert.Equal(t, exp, result, "CancelPortfolioMarginUSDTMarginedAlgoOrder should decode every field")
		return
	}
	assert.True(t, result.Complete, "CancelPortfolioMarginUSDTMarginedAlgoOrder should complete")
}

func TestCancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "CancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders must not error")
	if mockTests {
		assert.Equal(t, &SuccessResponse{Code: 200, Message: "The operation of cancel all open order is done."}, result, "CancelAllPortfolioMarginUSDTMarginedAlgoOpenOrders should decode every field")
	}
}

// umAlgoOrderDetail is the UM algo order the algo order fixtures return
func umAlgoOrderDetail() PortfolioMarginUSDTMarginedAlgoOrderDetailResponse {
	return PortfolioMarginUSDTMarginedAlgoOrderDetailResponse{
		AlgoID:                  2146760,
		ClientAlgoID:            "6B2I9XVcJpCjqPAJ4YoFX7",
		AlgoType:                "CONDITIONAL",
		OrderType:               "TAKE_PROFIT",
		Symbol:                  "BTCUSDT",
		Side:                    "SELL",
		PositionSide:            "LONG",
		TimeInForce:             "GTD",
		Quantity:                0.01,
		AlgoStatus:              "TRIGGERED",
		TriggerPrice:            75000,
		Price:                   74990,
		SelfTradePreventionMode: "EXPIRE_MAKER",
		WorkingType:             "MARK_PRICE",
		PriceMatch:              "OPPONENT",
		ClosePosition:           true,
		PriceProtect:            true,
		ReduceOnly:              true,
		CreateTime:              types.Time(time.UnixMilli(1750485492076)),
		UpdateTime:              types.Time(time.UnixMilli(1750514545091)),
		TriggerTime:             types.Time(time.UnixMilli(1750514545000)),
		GoodTillDate:            types.Time(time.UnixMilli(1750600000000)),
		ActualOrderID:           "8389765564",
		ActualPrice:             75000.5,
	}
}

func TestGetPortfolioMarginUSDTMarginedAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPortfolioMarginUSDTMarginedAlgoOrder must reject a request without an algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedAlgoOrder(t.Context(), 2146760, "")
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedAlgoOrder must not error")
	if mockTests {
		exp := umAlgoOrderDetail()
		assert.Equal(t, &exp, result, "GetPortfolioMarginUSDTMarginedAlgoOrder should decode every field")
	}
}

func TestGetPortfolioMarginUSDTMarginedOpenAlgoOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedOpenAlgoOrders(t.Context(), currency.NewBTCUSDT(), "CONDITIONAL", 2146760)
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedOpenAlgoOrders must not error")
	if mockTests {
		exp := []PortfolioMarginUSDTMarginedAlgoOrderResponse{{
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
			TriggerPrice:            75000,
			Price:                   74990,
			SelfTradePreventionMode: "EXPIRE_MAKER",
			WorkingType:             "MARK_PRICE",
			PriceMatch:              "OPPONENT",
			ClosePosition:           true,
			PriceProtect:            true,
			ReduceOnly:              true,
			CreateTime:              types.Time(time.UnixMilli(1750485492076)),
			UpdateTime:              types.Time(time.UnixMilli(1750485492080)),
			TriggerTime:             types.Time(time.UnixMilli(1750514545000)),
			GoodTillDate:            types.Time(time.UnixMilli(1750600000000)),
		}}
		assert.Equal(t, exp, result, "GetPortfolioMarginUSDTMarginedOpenAlgoOrders should decode every field")
	}
}

func TestGetPortfolioMarginUSDTMarginedAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginUSDTMarginedAlgoOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginUSDTMarginedAlgoOrderHistory must reject a nil request")
	_, err = e.GetPortfolioMarginUSDTMarginedAlgoOrderHistory(t.Context(), &PortfolioMarginUSDTMarginedAlgoOrderHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPortfolioMarginUSDTMarginedAlgoOrderHistory must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginUSDTMarginedAlgoOrderHistory(t.Context(), &PortfolioMarginUSDTMarginedAlgoOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginUSDTMarginedAlgoOrderHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUSDTMarginedAlgoOrderHistory(t.Context(), &PortfolioMarginUSDTMarginedAlgoOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), AlgoID: 2146760, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetPortfolioMarginUSDTMarginedAlgoOrderHistory must not error")
	if mockTests {
		assert.Equal(t, []PortfolioMarginUSDTMarginedAlgoOrderDetailResponse{umAlgoOrderDetail()}, result, "GetPortfolioMarginUSDTMarginedAlgoOrderHistory should decode every field")
	}
}
