package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestAccountBalanceResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	balance := AccountBalance{Asset: currency.BTC, TotalWalletBalance: 1.5, UpdateTime: types.Time(time.UnixMilli(1617939110373))}
	for _, tc := range []struct {
		name  string
		input string
		exp   AccountBalanceResponse
	}{
		{name: "array", input: `[{"asset":"BTC","totalWalletBalance":"1.5","updateTime":1617939110373},{"asset":"ETH"}]`, exp: AccountBalanceResponse{balance, {Asset: currency.ETH}}},
		{name: "object", input: `{"asset":"BTC","totalWalletBalance":"1.5","updateTime":1617939110373}`, exp: AccountBalanceResponse{balance}},
		{name: "null", input: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var resp AccountBalanceResponse
			require.NoError(t, json.Unmarshal([]byte(tc.input), &resp), "Unmarshal must not error")
			assert.Equal(t, tc.exp, resp, "Unmarshal should decode every balance")
		})
	}
	for _, input := range []string{`"oops"`, `[{"totalWalletBalance":true}]`, `{"totalWalletBalance":true}`} {
		var resp AccountBalanceResponse
		assert.Errorf(t, json.Unmarshal([]byte(input), &resp), "Unmarshal should error for %s", input)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		ccy  currency.Code
		exp  AccountBalanceResponse
	}{
		{name: "allAssets", ccy: currency.EMPTYCODE, exp: AccountBalanceResponse{
			{
				Asset:               currency.USDT,
				TotalWalletBalance:  139.22469206,
				CrossMarginAsset:    103,
				CrossMarginBorrowed: 10,
				CrossMarginFree:     100,
				CrossMarginInterest: 0.72469206,
				CrossMarginLocked:   3,
				UMWalletBalance:     12.5,
				UMUnrealizedPNL:     23.72469206,
				CMWalletBalance:     23.72469206,
				CMUnrealizedPNL:     -1.25,
				UpdateTime:          types.Time(time.UnixMilli(1617939110373)),
				NegativeBalance:     0.5,
			},
		}},
		{name: "asset", ccy: currency.BTC, exp: AccountBalanceResponse{
			{
				Asset:               currency.BTC,
				TotalWalletBalance:  0.31,
				CrossMarginBorrowed: 0.01,
				CrossMarginFree:     0.2,
				CrossMarginInterest: 0.0001,
				CrossMarginLocked:   0.05,
				UMWalletBalance:     0.04,
				UMUnrealizedPNL:     0.002,
				CMWalletBalance:     0.02,
				CMUnrealizedPNL:     -0.001,
				UpdateTime:          types.Time(time.UnixMilli(1617939110373)),
				NegativeBalance:     0.003,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAccountBalance(t.Context(), tc.ccy)
			require.NoError(t, err, "GetAccountBalance must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAccountBalance should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetAccountBalance should return balances")
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
		exp := &PMAccountInformationResponse{
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

func TestBNBTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.BNBTransfer(t.Context(), 0, "TO_UM")
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "BNBTransfer must reject a zero amount")
	_, err = e.BNBTransfer(t.Context(), 0.0001, "")
	require.ErrorIs(t, err, errTransferSideRequired, "BNBTransfer must reject an empty transfer side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BNBTransfer(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err, "BNBTransfer must not error")
	if mockTests {
		exp := &PMTransactionResponse{
			TransactionID: 100000001,
		}
		assert.Equal(t, exp, result, "BNBTransfer should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "BNBTransfer should return a transaction ID")
}

func TestGetAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAutoRepayFuturesStatus(t.Context())
	require.NoError(t, err, "GetAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PMAutoRepayFuturesStatusResponse{
			AutoRepay: true,
		}
		assert.Equal(t, exp, result, "GetAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAutoRepayFuturesStatus should return a status")
}

func TestChangeAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangeAutoRepayFuturesStatus(t.Context(), false)
	require.NoError(t, err, "ChangeAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangeAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangeAutoRepayFuturesStatus should report success")
}

func TestChangeCMInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeCMInitialLeverage(t.Context(), currency.EMPTYPAIR, 29)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangeCMInitialLeverage must reject an empty symbol")
	_, err = e.ChangeCMInitialLeverage(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0)
	require.ErrorIs(t, err, errLeverageRequired, "ChangeCMInitialLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.ChangeCMInitialLeverage(t.Context(), coinmTradablePair, 29)
	require.NoError(t, err, "ChangeCMInitialLeverage must not error")
	if mockTests {
		exp := &CMInitialLeverageResponse{
			Leverage:    29,
			MaxQuantity: 1000,
			Symbol:      "BTCUSD_PERP",
		}
		assert.Equal(t, exp, result, "ChangeCMInitialLeverage should decode every field")
		return
	}
	assert.Equal(t, uint64(29), result.Leverage, "ChangeCMInitialLeverage should return the new leverage")
}

func TestGetCMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCMCurrentPositionMode(t.Context())
	require.NoError(t, err, "GetCMCurrentPositionMode must not error")
	if mockTests {
		exp := &PMPositionModeResponse{
			DualSidePosition: true,
		}
		assert.Equal(t, exp, result, "GetCMCurrentPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCMCurrentPositionMode should return a position mode")
}

func TestChangeCMPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangeCMPositionMode(t.Context(), true)
	require.NoError(t, err, "ChangeCMPositionMode must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangeCMPositionMode should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangeCMPositionMode should report success")
}

func TestChangeUMInitialLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeUMInitialLeverage(t.Context(), currency.EMPTYPAIR, 29)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangeUMInitialLeverage must reject an empty symbol")
	_, err = e.ChangeUMInitialLeverage(t.Context(), currency.NewBTCUSDT(), 0)
	require.ErrorIs(t, err, errLeverageRequired, "ChangeUMInitialLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.ChangeUMInitialLeverage(t.Context(), usdtmTradablePair, 29)
	require.NoError(t, err, "ChangeUMInitialLeverage must not error")
	if mockTests {
		exp := &UMInitialLeverageResponse{
			Leverage:         29,
			MaxNotionalValue: 1000000,
			Symbol:           "BTCUSDT",
		}
		assert.Equal(t, exp, result, "ChangeUMInitialLeverage should decode every field")
		return
	}
	assert.Equal(t, uint64(29), result.Leverage, "ChangeUMInitialLeverage should return the new leverage")
}

func TestGetUMCurrentPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUMCurrentPositionMode(t.Context())
	require.NoError(t, err, "GetUMCurrentPositionMode must not error")
	if mockTests {
		exp := &PMPositionModeResponse{
			DualSidePosition: true,
		}
		assert.Equal(t, exp, result, "GetUMCurrentPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUMCurrentPositionMode should return a position mode")
}

func TestChangeUMPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangeUMPositionMode(t.Context(), true)
	require.NoError(t, err, "ChangeUMPositionMode must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangeUMPositionMode should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangeUMPositionMode should report success")
}

func TestGetCMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCMNotionalAndLeverageBrackets(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetCMNotionalAndLeverageBrackets must not error")
	if mockTests {
		exp := []CMNotionalAndLeverage{
			{
				Symbol: "BTCUSD_PERP",
				Brackets: []CMNotionalBracket{
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
		assert.Equal(t, exp, result, "GetCMNotionalAndLeverageBrackets should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCMNotionalAndLeverageBrackets should return brackets")
}

func TestFundAutoCollection(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FundAutoCollection(t.Context())
	require.NoError(t, err, "FundAutoCollection must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "FundAutoCollection should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "FundAutoCollection should report success")
}

func TestFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.FundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FundCollectionByAsset must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.FundCollectionByAsset(t.Context(), currency.ETH)
	require.NoError(t, err, "FundCollectionByAsset must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "FundCollectionByAsset should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "FundCollectionByAsset should report success")
}

func TestGetCMAccountDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCMAccountDetail(t.Context())
	require.NoError(t, err, "GetCMAccountDetail must not error")
	if mockTests {
		exp := &CMAccountDetailResponse{
			Assets: []PMAccountDetailAsset{
				{
					Asset:                  currency.BTC,
					CrossWalletBalance:     0.00241969,
					CrossUnrealizedPNL:     0.00000123,
					MaintenanceMargin:      0.00000456,
					InitialMargin:          0.0005,
					PositionInitialMargin:  0.0004,
					OpenOrderInitialMargin: 0.0001,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []CMAccountDetailPosition{
				{
					Symbol:                 "BTCUSD_PERP",
					PositionAmount:         1,
					InitialMargin:          0.0005,
					MaintenanceMargin:      0.00000456,
					UnrealizedProfit:       0.00000123,
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
		assert.Equal(t, exp, result, "GetCMAccountDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCMAccountDetail should return account details")
}

func TestGetCMIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCMIncomeHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetCMIncomeHistory(t.Context(), &PMIncomeHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCMIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCMIncomeHistory(t.Context(), &PMIncomeHistoryRequest{Symbol: coinmTradablePair, IncomeType: "REALIZED_PNL", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 10})
	require.NoError(t, err, "GetCMIncomeHistory must not error")
	if mockTests {
		exp := []IncomeItem{
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
		assert.Equal(t, exp, result, "GetCMIncomeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCMIncomeHistory should return income records")
}

func TestGetMarginBorrowOrLoanInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginBorrowOrLoanInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginBorrowOrLoanInterestHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetMarginBorrowOrLoanInterestHistory(t.Context(), &PMMarginInterestHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginBorrowOrLoanInterestHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginBorrowOrLoanInterestHistory(t.Context(), &PMMarginInterestHistoryRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Current: 10, Size: 1, Archived: true})
	require.NoError(t, err, "GetMarginBorrowOrLoanInterestHistory must not error")
	if mockTests {
		exp := &PMMarginInterestHistoryResponse{
			Rows: []PMMarginInterest{
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
		assert.Equal(t, exp, result, "GetMarginBorrowOrLoanInterestHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginBorrowOrLoanInterestHistory should return interest records")
}

func TestGetUMAccountDetail(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUMAccountDetail(t.Context())
	require.NoError(t, err, "GetUMAccountDetail must not error")
	if mockTests {
		exp := &UMAccountDetailResponse{
			Assets: []PMAccountDetailAsset{
				{
					Asset:                  currency.USDT,
					CrossWalletBalance:     23.72469206,
					CrossUnrealizedPNL:     1.25,
					MaintenanceMargin:      0.42,
					InitialMargin:          10.5,
					PositionInitialMargin:  10,
					OpenOrderInitialMargin: 0.5,
					UpdateTime:             types.Time(time.UnixMilli(1625474304765)),
				},
			},
			Positions: []UMAccountDetailPosition{
				{
					Symbol:                 "BTCUSDT",
					InitialMargin:          10.5,
					MaintenanceMargin:      0.42,
					UnrealizedProfit:       1.25,
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
		assert.Equal(t, exp, result, "GetUMAccountDetail should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUMAccountDetail should return account details")
}

func TestGetUMIncomeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMIncomeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUMIncomeHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUMIncomeHistory(t.Context(), &PMIncomeHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUMIncomeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUMIncomeHistory(t.Context(), &PMIncomeHistoryRequest{Symbol: usdtmTradablePair, IncomeType: "COMMISSION", StartTime: startTime, EndTime: endTime, Page: 1, Limit: 10})
	require.NoError(t, err, "GetUMIncomeHistory must not error")
	if mockTests {
		exp := []IncomeItem{
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
		assert.Equal(t, exp, result, "GetUMIncomeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUMIncomeHistory should return income records")
}

func TestGetCMUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMUserCommissionRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCMUserCommissionRate must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCMUserCommissionRate(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetCMUserCommissionRate must not error")
	if mockTests {
		exp := &PMCommissionRateResponse{
			Symbol:              "BTCUSD_PERP",
			MakerCommissionRate: 0.00015,
			TakerCommissionRate: 0.0004,
		}
		assert.Equal(t, exp, result, "GetCMUserCommissionRate should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbol, "GetCMUserCommissionRate should return the symbol")
}

func TestGetUMUserCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMUserCommissionRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUMUserCommissionRate must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUMUserCommissionRate(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetUMUserCommissionRate must not error")
	if mockTests {
		exp := &PMCommissionRateResponse{
			Symbol:              "BTCUSDT",
			MakerCommissionRate: 0.0002,
			TakerCommissionRate: 0.0004,
		}
		assert.Equal(t, exp, result, "GetUMUserCommissionRate should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbol, "GetUMUserCommissionRate should return the symbol")
}

func TestGetPMMarginMaxBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMMarginMaxBorrow(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetPMMarginMaxBorrow must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPMMarginMaxBorrow(t.Context(), currency.ETH)
	require.NoError(t, err, "GetPMMarginMaxBorrow must not error")
	if mockTests {
		exp := &PMMarginMaxBorrowResponse{
			Amount:      1.69248805,
			BorrowLimit: 60,
		}
		assert.Equal(t, exp, result, "GetPMMarginMaxBorrow should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPMMarginMaxBorrow should return borrow limits")
}

func TestGetPortfolioMarginUMTradingQuantitativeRulesIndicator(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginUMTradingQuantitativeRulesIndicator(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetPortfolioMarginUMTradingQuantitativeRulesIndicator must not error")
	if mockTests {
		exp := &PMTradingQuantitativeRulesIndicatorsResponse{
			Indicators: map[string][]PMTradingIndicator{
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
		assert.Equal(t, exp, result, "GetPortfolioMarginUMTradingQuantitativeRulesIndicator should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginUMTradingQuantitativeRulesIndicator should return indicators")
}

func TestGetCMPositionInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name        string
		marginAsset currency.Code
		pair        currency.Code
		exp         []CMPositionInformation
	}{
		{name: "marginAsset", marginAsset: currency.ETH, exp: []CMPositionInformation{
			{
				Symbol:           "ETHUSD_PERP",
				PositionAmount:   10,
				EntryPrice:       1580.5,
				MarkPrice:        1590.25,
				UnrealizedProfit: 0.00038801,
				LiquidationPrice: 1201.75,
				Leverage:         20,
				PositionSide:     "LONG",
				UpdateTime:       types.Time(time.UnixMilli(1627026881327)),
				MaxQuantity:      1000,
				NotionalValue:    0.06288319,
			},
		}},
		{name: "pair", pair: currency.NewCode("BTCUSD"), exp: []CMPositionInformation{
			{
				Symbol:           "BTCUSD_201225",
				PositionAmount:   1,
				EntryPrice:       11707.70000003,
				MarkPrice:        11788.66626667,
				UnrealizedProfit: 0.00005866,
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
			result, err := e.GetCMPositionInformation(t.Context(), tc.marginAsset, tc.pair)
			require.NoError(t, err, "GetCMPositionInformation must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCMPositionInformation should decode every field")
				return
			}
			assert.NotNil(t, result, "GetCMPositionInformation should return positions")
		})
	}
}

func TestGetMarginLoanRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginLoanRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginLoanRecord must reject a nil request")
	_, err = e.GetMarginLoanRecord(t.Context(), &PMMarginLoanRepayRecordRequest{TransactionID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMarginLoanRecord must reject an empty asset")
	_, err = e.GetMarginLoanRecord(t.Context(), &PMMarginLoanRepayRecordRequest{Asset: currency.ETH})
	require.ErrorIs(t, err, errStartTimeRequired, "GetMarginLoanRecord must reject a request without a transaction ID or start time")
	startTime, endTime := getTime()
	_, err = e.GetMarginLoanRecord(t.Context(), &PMMarginLoanRepayRecordRequest{Asset: currency.ETH, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginLoanRecord must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginLoanRecord(t.Context(), &PMMarginLoanRepayRecordRequest{Asset: currency.ETH, TransactionID: 12807067523, StartTime: startTime, EndTime: endTime, Current: 10, Size: 1, Archived: true})
	require.NoError(t, err, "GetMarginLoanRecord must not error")
	if mockTests {
		exp := &PMMarginLoanRecordResponse{
			Rows: []PMMarginLoanRecord{
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
		assert.Equal(t, exp, result, "GetMarginLoanRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginLoanRecord should return loan records")
}

func TestGetMarginMaxWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginMaxWithdrawal(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMarginMaxWithdrawal must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginMaxWithdrawal(t.Context(), currency.BTC)
	require.NoError(t, err, "GetMarginMaxWithdrawal must not error")
	if mockTests {
		exp := &PMMarginMaxWithdrawResponse{
			Amount: 60,
		}
		assert.Equal(t, exp, result, "GetMarginMaxWithdrawal should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginMaxWithdrawal should return an amount")
}

func TestGetMarginRepayRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginRepayRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarginRepayRecord must reject a nil request")
	_, err = e.GetMarginRepayRecord(t.Context(), &PMMarginLoanRepayRecordRequest{TransactionID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetMarginRepayRecord must reject an empty asset")
	_, err = e.GetMarginRepayRecord(t.Context(), &PMMarginLoanRepayRecordRequest{Asset: currency.ETH})
	require.ErrorIs(t, err, errStartTimeRequired, "GetMarginRepayRecord must reject a request without a transaction ID or start time")
	startTime, endTime := getTime()
	_, err = e.GetMarginRepayRecord(t.Context(), &PMMarginLoanRepayRecordRequest{Asset: currency.ETH, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMarginRepayRecord must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginRepayRecord(t.Context(), &PMMarginLoanRepayRecordRequest{Asset: currency.ETH, TransactionID: 2970933056, StartTime: startTime, EndTime: endTime, Current: 10, Size: 1, Archived: true})
	require.NoError(t, err, "GetMarginRepayRecord must not error")
	if mockTests {
		exp := &PMMarginRepayRecordResponse{
			Rows: []PMMarginRepayRecord{
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
		assert.Equal(t, exp, result, "GetMarginRepayRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginRepayRecord should return repay records")
}

func TestGetPortfolioMarginNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginNegativeBalanceInterestHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), &PMNegativeBalanceInterestHistoryRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginNegativeBalanceInterestHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginNegativeBalanceInterestHistory(t.Context(), &PMNegativeBalanceInterestHistoryRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Size: 10})
	require.NoError(t, err, "GetPortfolioMarginNegativeBalanceInterestHistory must not error")
	if mockTests {
		exp := []PortfolioMarginNegativeBalanceInterest{
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

func TestGetUMPositionInformation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUMPositionInformation(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetUMPositionInformation must not error")
	if mockTests {
		exp := []UMPositionInformation{
			{
				EntryPrice:       62000.5,
				Leverage:         10,
				MarkPrice:        62500.1,
				MaxNotionalValue: 20000000,
				PositionAmount:   0.01,
				Notional:         625.001,
				Symbol:           "BTCUSDT",
				UnrealizedProfit: 4.996,
				LiquidationPrice: 6170.20509059,
				PositionSide:     "BOTH",
				UpdateTime:       types.Time(time.UnixMilli(1625474304765)),
			},
		}
		assert.Equal(t, exp, result, "GetUMPositionInformation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUMPositionInformation should return positions")
}

func TestGetUserNegativeBalanceAutoExchangeRecord(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), time.Time{}, endTime)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserNegativeBalanceAutoExchangeRecord must reject an unset start time")
	_, err = e.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), startTime, time.Time{})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUserNegativeBalanceAutoExchangeRecord must reject an unset end time")
	_, err = e.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUserNegativeBalanceAutoExchangeRecord must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserNegativeBalanceAutoExchangeRecord(t.Context(), startTime, endTime)
	require.NoError(t, err, "GetUserNegativeBalanceAutoExchangeRecord must not error")
	if mockTests {
		exp := &PMNegativeBalanceExchangeRecordResponse{
			Total: 1,
			Rows: []PMNegativeBalanceExchangeRecord{
				{
					StartTime: types.Time(time.Unix(1744110000, 0)),
					EndTime:   types.Time(time.UnixMilli(1744110201338)),
					Details: []PMNegativeBalanceExchangeDetail{
						{
							Asset:                currency.ETH,
							NegativeBalance:      1.10264488,
							NegativeMaxThreshold: 5,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetUserNegativeBalanceAutoExchangeRecord should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUserNegativeBalanceAutoExchangeRecord should return records")
}

func TestGetUserRateLimits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUserRateLimits(t.Context())
	require.NoError(t, err, "GetUserRateLimits must not error")
	if mockTests {
		exp := []PMRateLimit{
			{
				RateLimitType:  "ORDERS",
				Interval:       "MINUTE",
				IntervalNumber: 1,
				Limit:          1200,
			},
		}
		assert.Equal(t, exp, result, "GetUserRateLimits should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUserRateLimits should return rate limits")
}

func TestRepayFuturesNegativeBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayFuturesNegativeBalance(t.Context())
	require.NoError(t, err, "RepayFuturesNegativeBalance must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "RepayFuturesNegativeBalance should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "RepayFuturesNegativeBalance should report success")
}

func TestGetUMNotionalAndLeverageBrackets(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUMNotionalAndLeverageBrackets(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetUMNotionalAndLeverageBrackets must not error")
	if mockTests {
		exp := []UMNotionalAndLeverage{
			{
				Symbol:              "BTCUSDT",
				NotionalCoefficient: 1.5,
				Brackets: []UMNotionalBracket{
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
		assert.Equal(t, exp, result, "GetUMNotionalAndLeverageBrackets should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUMNotionalAndLeverageBrackets should return brackets")
}

func TestCancelAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllCMOpenConditionalOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllCMOpenConditionalOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllCMOpenConditionalOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "CancelAllCMOpenConditionalOrders must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "The operation of cancel all conditional open order is done.",
		}
		assert.Equal(t, exp, result, "CancelAllCMOpenConditionalOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result.Message, "CancelAllCMOpenConditionalOrders should return a message")
}

func TestCancelAllCMOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllCMOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllCMOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllCMOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "CancelAllCMOrders must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "The operation of cancel all open order is done.",
		}
		assert.Equal(t, exp, result, "CancelAllCMOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result.Message, "CancelAllCMOrders should return a message")
}

func TestCancelAllUMOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllUMOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllUMOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllUMOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "CancelAllUMOrders must not error")
	if mockTests {
		exp := &SuccessResponse{
			Code:    200,
			Message: "The operation of cancel all open order is done.",
		}
		assert.Equal(t, exp, result, "CancelAllUMOrders should decode every field")
		return
	}
	assert.NotEmpty(t, result.Message, "CancelAllUMOrders should return a message")
}

func TestNewCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewCMConditionalOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewCMConditionalOrder must reject a nil request")
	arg := &CMConditionalOrderRequest{PositionSide: "LONG"}
	_, err = e.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewCMConditionalOrder must reject an empty symbol")
	arg.Symbol = currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	_, err = e.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewCMConditionalOrder must reject an invalid side")
	arg.Side = order.Buy
	_, err = e.NewCMConditionalOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStrategyTypeRequired, "NewCMConditionalOrder must reject an empty strategy type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.NewCMConditionalOrder(t.Context(), &CMConditionalOrderRequest{
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
	require.NoError(t, err, "NewCMConditionalOrder must not error")
	if mockTests {
		exp := &NewCMConditionalOrderResponse{
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
		assert.Equal(t, exp, result, "NewCMConditionalOrder should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "NewCMConditionalOrder should return a strategy ID")
}

func TestCancelCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCMConditionalOrder(t.Context(), currency.EMPTYPAIR, 1231231, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelCMConditionalOrder must reject an empty symbol")
	_, err = e.CancelCMConditionalOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelCMConditionalOrder must reject a request without a strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelCMConditionalOrder(t.Context(), coinmTradablePair, 1231231, "myOrder1")
	require.NoError(t, err, "CancelCMConditionalOrder must not error")
	if mockTests {
		exp := &CancelCMConditionalOrderResponse{
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
		assert.Equal(t, exp, result, "CancelCMConditionalOrder should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "CancelCMConditionalOrder should return the strategy ID")
}

func TestGetCMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCMOrder must reject an empty symbol")
	_, err = e.GetCMOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetCMOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCMOrder(t.Context(), coinmTradablePair, 1234, "")
	require.NoError(t, err, "GetCMOrder must not error")
	if mockTests {
		exp := &CMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetCMOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetCMOrder should return the order ID")
}

func TestNewCMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewCMOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewCMOrder must reject a nil request")
	arg := &CMOrderRequest{ReduceOnly: true}
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewCMOrder must reject an empty symbol")
	arg.Symbol = currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewCMOrder must reject an invalid side")
	arg.Side = order.Buy
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewCMOrder must reject an empty order type")
	arg.OrderType = "LIMIT"
	arg.Price = 9500
	arg.PriceMatch = "OPPONENT"
	_, err = e.NewCMOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "NewCMOrder must reject a price with priceMatch")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *CMOrderRequest
		exp  *CMOrderResponse
	}{
		{
			name: "price",
			arg:  &CMOrderRequest{Symbol: coinmTradablePair, Side: order.Buy, PositionSide: "BOTH", OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 2, ReduceOnly: true, Price: 9500, NewClientOrderID: "testOrder", NewOrderRespType: "RESULT"},
			exp: &CMOrderResponse{
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
			arg:  &CMOrderRequest{Symbol: coinmTradablePair, Side: order.Sell, OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 2, PriceMatch: "OPPONENT"},
			exp: &CMOrderResponse{
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
			result, err := e.NewCMOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewCMOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewCMOrder should decode every field")
				return
			}
			assert.NotZero(t, result.OrderID, "NewCMOrder should return an order ID")
		})
	}
}

func TestCancelCMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCMOrder(t.Context(), currency.EMPTYPAIR, 21321312, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelCMOrder must reject an empty symbol")
	_, err = e.CancelCMOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelCMOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelCMOrder(t.Context(), coinmTradablePair, 21321312, "myOrder1")
	require.NoError(t, err, "CancelCMOrder must not error")
	if mockTests {
		exp := &CMOrderResponse{
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
		assert.Equal(t, exp, result, "CancelCMOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "CancelCMOrder should return the order ID")
}

func TestCancelAllMarginOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllMarginOpenOrdersBySymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllMarginOpenOrdersBySymbol must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllMarginOpenOrdersBySymbol(t.Context(), marginTradablePair)
	require.NoError(t, err, "CancelAllMarginOpenOrdersBySymbol must not error")
	if mockTests {
		exp := []PMCancelledMarginOrder{
			{
				Symbol:                   "BTCUSDT",
				OriginalClientOrderID:    "E6APeyTJvkMvLMYMqu1KQ4",
				OrderID:                  11,
				OrderListID:              -1,
				ClientOrderID:            "pXLV6Hz6mprAcVYpVMTGgx",
				Price:                    61000,
				OriginalQuantity:         0.02,
				ExecutedQuantity:         0.005,
				CummulativeQuoteQuantity: 305,
				Status:                   "CANCELED",
				TimeInForce:              "GTC",
				Type:                     "LIMIT",
				Side:                     "BUY",
			},
			{
				Symbol:            "BTCUSDT",
				OrderListID:       1929,
				ContingencyType:   "OCO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "2inzWQdDvZLHbbAmAozX2N",
				TransactionTime:   types.Time(time.UnixMilli(1585230948299)),
				Orders: []PMMarginOCOOrderLeg{
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
				OrderReports: []PMCancelledMarginOrderReport{
					{
						Symbol:                   "BTCUSDT",
						OriginalClientOrderID:    "CwOOIPHSmYywx6jZX77TdL",
						OrderID:                  20,
						OrderListID:              1929,
						ClientOrderID:            "pXLV6Hz6mprAcVYpVMTGgy",
						Price:                    59000,
						OriginalQuantity:         0.02,
						ExecutedQuantity:         0.001,
						CummulativeQuoteQuantity: 59,
						Status:                   "CANCELED",
						TimeInForce:              "GTC",
						Type:                     "STOP_LOSS_LIMIT",
						Side:                     "SELL",
						StopPrice:                59100,
						IcebergQuantity:          0.01,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "CancelAllMarginOpenOrdersBySymbol should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllMarginOpenOrdersBySymbol should return the cancelled orders")
}

func TestGetMarginAccountOCO(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountOCO(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMarginAccountOCO must reject a request without an order list ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountOCO(t.Context(), 0, "123421-abcde")
	require.NoError(t, err, "GetMarginAccountOCO must not error")
	if mockTests {
		exp := &PMMarginOCOOrderResponse{
			OrderListID:       27,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
			TransactionTime:   types.Time(time.UnixMilli(1565245656253)),
			Symbol:            "BTCUSDT",
			Orders: []PMMarginOCOOrderLeg{
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
		assert.Equal(t, exp, result, "GetMarginAccountOCO should decode every field")
		return
	}
	assert.NotZero(t, result.OrderListID, "GetMarginAccountOCO should return the order list ID")
}

func TestCancelMarginAccountOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMarginAccountOCOOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelMarginAccountOCOOrders must reject a nil request")
	_, err = e.CancelMarginAccountOCOOrders(t.Context(), &PMCancelMarginOCOOrderRequest{OrderListID: 1929})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelMarginAccountOCOOrders must reject an empty symbol")
	_, err = e.CancelMarginAccountOCOOrders(t.Context(), &PMCancelMarginOCOOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelMarginAccountOCOOrders must reject a request without an order list ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelMarginAccountOCOOrders(t.Context(), &PMCancelMarginOCOOrderRequest{Symbol: marginTradablePair, OrderListID: 1929, ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN", NewClientOrderID: "cancelMyList"})
	require.NoError(t, err, "CancelMarginAccountOCOOrders must not error")
	if mockTests {
		exp := &PMCancelMarginOCOOrderResponse{
			OrderListID:       1929,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN",
			TransactionTime:   types.Time(time.UnixMilli(1574040868128)),
			Symbol:            "BTCUSDT",
			Orders: []PMMarginOCOOrderLeg{
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
			OrderReports: []PMCancelMarginOCOOrderReport{
				{
					Symbol:                   "BTCUSDT",
					OriginalClientOrderID:    "pO9ufTiFGg3nw2fOdgeOXa",
					OrderID:                  2,
					OrderListID:              1929,
					ClientOrderID:            "unfWT8ig8i0uj6lPuYLez6",
					Price:                    59000,
					OriginalQuantity:         0.02,
					ExecutedQuantity:         0.002,
					CummulativeQuoteQuantity: 118,
					Status:                   "CANCELED",
					TimeInForce:              "GTC",
					Type:                     "STOP_LOSS_LIMIT",
					Side:                     "SELL",
					StopPrice:                59100,
				},
			},
		}
		assert.Equal(t, exp, result, "CancelMarginAccountOCOOrders should decode every field")
		return
	}
	assert.NotZero(t, result.OrderListID, "CancelMarginAccountOCOOrders should return the order list ID")
}

func TestGetMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginAccountOrder(t.Context(), currency.EMPTYPAIR, 12434, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarginAccountOrder must reject an empty symbol")
	_, err = e.GetMarginAccountOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetMarginAccountOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetMarginAccountOrder(t.Context(), marginTradablePair, 12434, "")
	require.NoError(t, err, "GetMarginAccountOrder must not error")
	if mockTests {
		exp := &PMMarginOrderResponse{
			ClientOrderID:            "ZwfQzuDIGpceVhKW5DvCmO",
			CummulativeQuoteQuantity: 1500.5,
			ExecutedQuantity:         0.025,
			IcebergQuantity:          0.01,
			IsWorking:                true,
			OrderID:                  12434,
			OriginalQuantity:         0.3,
			Price:                    60020,
			Side:                     "SELL",
			Status:                   "PARTIALLY_FILLED",
			StopPrice:                60000,
			Symbol:                   "BTCUSDT",
			Time:                     types.Time(time.UnixMilli(1562133008725)),
			TimeInForce:              "GTC",
			Type:                     "STOP_LOSS_LIMIT",
			UpdateTime:               types.Time(time.UnixMilli(1562133008725)),
			AccountID:                152950866,
			SelfTradePreventionMode:  "EXPIRE_TAKER",
			PreventedMatchID:         1,
			PreventedQuantity:        0.002,
		}
		assert.Equal(t, exp, result, "GetMarginAccountOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetMarginAccountOrder should return the order ID")
}

func TestNewMarginOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewMarginOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewMarginOrder must reject a nil request")
	arg := &PMMarginOrderRequest{TimeInForce: "GTC"}
	_, err = e.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewMarginOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewMarginOrder must reject an invalid side")
	arg.Side = order.Sell
	_, err = e.NewMarginOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewMarginOrder must reject an empty order type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	autoRepayAtCancel := false
	for _, tc := range []struct {
		name string
		arg  *PMMarginOrderRequest
		exp  *PMNewMarginOrderResponse
	}{
		{
			name: "stopLossLimit",
			arg:  &PMMarginOrderRequest{Symbol: marginTradablePair, Side: order.Sell, OrderType: "STOP_LOSS_LIMIT", Quantity: 0.001, Price: 60000, StopPrice: 60100, NewClientOrderID: "6gCrw2kRUAF9CvJDGP16IP", NewOrderRespType: "FULL", IcebergQuantity: 0.0005, SideEffectType: "MARGIN_BUY", TimeInForce: "GTC", SelfTradePreventionMode: "EXPIRE_TAKER", AutoRepayAtCancel: &autoRepayAtCancel},
			exp: &PMNewMarginOrderResponse{
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
				Fills:                 []PMMarginOrderFill{},
			},
		},
		{
			name: "marketQuoteQuantity",
			arg:  &PMMarginOrderRequest{Symbol: marginTradablePair, Side: order.Buy, OrderType: "MARKET", QuoteOrderQuantity: 100, NewOrderRespType: "FULL"},
			exp: &PMNewMarginOrderResponse{
				Symbol:                   "BTCUSDT",
				OrderID:                  29,
				ClientOrderID:            "8xH4F9NkTRjQ1Rz2bR6l3P",
				TransactTime:             types.Time(time.UnixMilli(1507725176596)),
				OriginalQuantity:         0.00166,
				ExecutedQuantity:         0.00166,
				CummulativeQuoteQuantity: 100,
				Status:                   "FILLED",
				TimeInForce:              "GTC",
				Type:                     "MARKET",
				Side:                     "BUY",
				Fills: []PMMarginOrderFill{
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
			result, err := e.NewMarginOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewMarginOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewMarginOrder should decode every field")
				return
			}
			assert.NotZero(t, result.OrderID, "NewMarginOrder should return an order ID")
		})
	}
}

func TestPMCancelMarginAccountOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PMCancelMarginAccountOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PMCancelMarginAccountOrder must reject a nil request")
	_, err = e.PMCancelMarginAccountOrder(t.Context(), &PMCancelMarginOrderRequest{OrderID: 12314})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PMCancelMarginAccountOrder must reject an empty symbol")
	_, err = e.PMCancelMarginAccountOrder(t.Context(), &PMCancelMarginOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "PMCancelMarginAccountOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.PMCancelMarginAccountOrder(t.Context(), &PMCancelMarginOrderRequest{Symbol: marginTradablePair, OrderID: 12314, NewClientOrderID: "cancelMyOrder1"})
	require.NoError(t, err, "PMCancelMarginAccountOrder must not error")
	if mockTests {
		exp := &PMCancelMarginOrderResponse{
			Symbol:                   "BTCUSDT",
			OrderID:                  12314,
			OriginalClientOrderID:    "myOrder1",
			ClientOrderID:            "cancelMyOrder1",
			Price:                    60000,
			OriginalQuantity:         0.01,
			ExecutedQuantity:         0.008,
			CummulativeQuoteQuantity: 480,
			Status:                   "CANCELED",
			TimeInForce:              "GTC",
			Type:                     "LIMIT",
			Side:                     "SELL",
		}
		assert.Equal(t, exp, result, "PMCancelMarginAccountOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "PMCancelMarginAccountOrder should return the order ID")
}

func TestGetUMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUMOrder must reject an empty symbol")
	_, err = e.GetUMOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetUMOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUMOrder(t.Context(), usdtmTradablePair, 1234, "")
	require.NoError(t, err, "GetUMOrder must not error")
	if mockTests {
		exp := &UMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetUMOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetUMOrder should return the order ID")
}

func TestNewUMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewUMOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewUMOrder must reject a nil request")
	arg := &UMOrderRequest{ReduceOnly: true}
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewUMOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewUMOrder must reject an invalid side")
	arg.Side = order.Buy
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewUMOrder must reject an empty order type")
	arg.OrderType = "LIMIT"
	arg.Price = 62000.5
	arg.PriceMatch = "QUEUE"
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "NewUMOrder must reject a price with priceMatch")
	arg.PriceMatch = ""
	arg.TimeInForce = "GTD"
	_, err = e.NewUMOrder(t.Context(), arg)
	require.ErrorIs(t, err, errGoodTillDateRequired, "NewUMOrder must reject a GTD order without a good till date")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *UMOrderRequest
		exp  *UMOrderResponse
	}{
		{
			name: "goodTillDate",
			arg:  &UMOrderRequest{Symbol: usdtmTradablePair, Side: order.Buy, PositionSide: "BOTH", OrderType: "LIMIT", TimeInForce: "GTD", Quantity: 0.02, ReduceOnly: true, Price: 62000.5, NewClientOrderID: "testOrder", NewOrderRespType: "RESULT", SelfTradePreventionMode: "EXPIRE_MAKER", GoodTillDate: time.UnixMilli(1693207680000)},
			exp: &UMOrderResponse{
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
			arg:  &UMOrderRequest{Symbol: usdtmTradablePair, Side: order.Sell, OrderType: "LIMIT", TimeInForce: "GTC", Quantity: 0.02, PriceMatch: "QUEUE"},
			exp: &UMOrderResponse{
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
			result, err := e.NewUMOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewUMOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewUMOrder should decode every field")
				return
			}
			assert.NotZero(t, result.OrderID, "NewUMOrder should return an order ID")
		})
	}
}

func TestCancelUMOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelUMOrder(t.Context(), currency.EMPTYPAIR, 1234132, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelUMOrder must reject an empty symbol")
	_, err = e.CancelUMOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelUMOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelUMOrder(t.Context(), usdtmTradablePair, 1234132, "myOrder1")
	require.NoError(t, err, "CancelUMOrder must not error")
	if mockTests {
		exp := &UMOrderResponse{
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
		assert.Equal(t, exp, result, "CancelUMOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "CancelUMOrder should return the order ID")
}

func TestGetCMAccountTradeList(t *testing.T) {
	t.Parallel()
	cmSymbol := currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	startTime, endTime := getTime()
	_, err := e.GetCMAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCMAccountTradeList must reject a nil request")
	_, err = e.GetCMAccountTradeList(t.Context(), &CMAccountTradeListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCMAccountTradeList must reject a request without a symbol or pair")
	_, err = e.GetCMAccountTradeList(t.Context(), &CMAccountTradeListRequest{Symbol: cmSymbol, Pair: currency.NewCode("BTCUSD")})
	require.ErrorIs(t, err, errSymbolWithPair, "GetCMAccountTradeList must reject a symbol with a pair")
	_, err = e.GetCMAccountTradeList(t.Context(), &CMAccountTradeListRequest{Pair: currency.NewCode("BTCUSD"), FromID: 6})
	require.ErrorIs(t, err, errFromIDWithPair, "GetCMAccountTradeList must reject fromId with a pair")
	_, err = e.GetCMAccountTradeList(t.Context(), &CMAccountTradeListRequest{Symbol: cmSymbol, FromID: 6, StartTime: startTime})
	require.ErrorIs(t, err, errFromIDWithTimeRange, "GetCMAccountTradeList must reject fromId with a time range")
	_, err = e.GetCMAccountTradeList(t.Context(), &CMAccountTradeListRequest{Symbol: cmSymbol, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCMAccountTradeList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *CMAccountTradeListRequest
		exp  []CMAccountTrade
	}{
		{name: "symbol", arg: &CMAccountTradeListRequest{Symbol: coinmTradablePair, StartTime: startTime, EndTime: endTime, Limit: 10}, exp: []CMAccountTrade{
			{
				Symbol:          "BTCUSD_PERP",
				ID:              6,
				OrderID:         28,
				Pair:            "BTCUSD",
				Side:            "BUY",
				Price:           8800,
				Quantity:        1,
				RealizedPNL:     0.0001,
				MarginAsset:     currency.BTC,
				BaseQuantity:    0.01136364,
				Commission:      0.00000454,
				CommissionAsset: currency.BTC,
				Time:            types.Time(time.UnixMilli(1590743483586)),
				PositionSide:    "BOTH",
				Buyer:           true,
				Maker:           true,
			},
		}},
		{name: "pair", arg: &CMAccountTradeListRequest{Pair: currency.NewCode("BTCUSD")}, exp: []CMAccountTrade{
			{
				Symbol:          "BTCUSD_250926",
				ID:              7,
				OrderID:         29,
				Pair:            "BTCUSD",
				Side:            "BUY",
				Price:           8800,
				Quantity:        1,
				RealizedPNL:     0.0001,
				MarginAsset:     currency.BTC,
				BaseQuantity:    0.01136364,
				Commission:      0.00000454,
				CommissionAsset: currency.BTC,
				Time:            types.Time(time.UnixMilli(1590743483586)),
				PositionSide:    "BOTH",
				Buyer:           true,
				Maker:           true,
			},
		}},
		{name: "fromID", arg: &CMAccountTradeListRequest{Symbol: coinmTradablePair, FromID: 6}, exp: []CMAccountTrade{
			{
				Symbol:          "BTCUSD_PERP",
				ID:              6,
				OrderID:         28,
				Pair:            "BTCUSD",
				Side:            "BUY",
				Price:           8800,
				Quantity:        1,
				RealizedPNL:     0.0001,
				MarginAsset:     currency.BTC,
				BaseQuantity:    0.01136364,
				Commission:      0.00000454,
				CommissionAsset: currency.BTC,
				Time:            types.Time(time.UnixMilli(1590743483586)),
				PositionSide:    "BOTH",
				Buyer:           true,
				Maker:           true,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCMAccountTradeList(t.Context(), tc.arg)
			require.NoError(t, err, "GetCMAccountTradeList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCMAccountTradeList should decode every field")
				return
			}
			assert.NotNil(t, result, "GetCMAccountTradeList should return trades")
		})
	}
}

func TestGetCMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCMPositionADLQuantileEstimation(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetCMPositionADLQuantileEstimation must not error")
	if mockTests {
		exp := []CMADLQuantileEstimation{
			{
				Symbol: "BTCUSD_PERP",
				ADLQuantile: CMADLQuantile{
					Long:  3,
					Short: 3,
					Hedge: 1,
				},
			},
			{
				Symbol: "BTCUSD_250926",
				ADLQuantile: CMADLQuantile{
					Long:  1,
					Short: 2,
					Both:  4,
				},
			},
		}
		assert.Equal(t, exp, result, "GetCMPositionADLQuantileEstimation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCMPositionADLQuantileEstimation should return quantiles")
}

func TestMarginAccountBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountBorrow(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "MarginAccountBorrow must reject an empty asset")
	_, err = e.MarginAccountBorrow(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "MarginAccountBorrow must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.MarginAccountBorrow(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err, "MarginAccountBorrow must not error")
	if mockTests {
		exp := &PMTransactionResponse{
			TransactionID: 100000001,
		}
		assert.Equal(t, exp, result, "MarginAccountBorrow should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "MarginAccountBorrow should return a transaction ID")
}

func TestMarginAccountNewOCO(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountNewOCO(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "MarginAccountNewOCO must reject a nil request")
	arg := &PMMarginOCOOrderRequest{ListClientOrderID: "JYVpp3F0f5CAG15DhtrqLp"}
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "MarginAccountNewOCO must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "MarginAccountNewOCO must reject an invalid side")
	arg.Side = order.Sell
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "MarginAccountNewOCO must reject a zero quantity")
	arg.Quantity = 0.1
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "MarginAccountNewOCO must reject a zero price")
	arg.Price = 70000
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "MarginAccountNewOCO must reject a zero stop price")
	arg.StopPrice = 60000
	arg.StopLimitPrice = 59900
	_, err = e.MarginAccountNewOCO(t.Context(), arg)
	require.ErrorIs(t, err, errStopLimitTimeInForceRequired, "MarginAccountNewOCO must reject a stop limit price without a time in force")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.MarginAccountNewOCO(t.Context(), &PMMarginOCOOrderRequest{
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
		NewOrderRespType:     "RESULT",
		SideEffectType:       "MARGIN_BUY",
	})
	require.NoError(t, err, "MarginAccountNewOCO must not error")
	if mockTests {
		exp := &PMNewMarginOCOOrderResponse{
			OrderListID:       1,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "JYVpp3F0f5CAG15DhtrqLp",
			TransactionTime:   types.Time(time.UnixMilli(1563417480525)),
			Symbol:            "BTCUSDT",
			Orders: []PMMarginOCOOrderLeg{
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
			OrderReports: []PMNewMarginOCOOrderReport{
				{
					Symbol:                   "BTCUSDT",
					OrderID:                  2,
					OrderListID:              1,
					ClientOrderID:            "xTXKaGYd4bluPVp78IVRvl",
					TransactTime:             types.Time(time.UnixMilli(1563417480525)),
					Price:                    59900,
					OriginalQuantity:         0.1,
					ExecutedQuantity:         0.01,
					CummulativeQuoteQuantity: 599,
					Status:                   "PARTIALLY_FILLED",
					TimeInForce:              "GTC",
					Type:                     "STOP_LOSS_LIMIT",
					Side:                     "SELL",
					StopPrice:                60000,
				},
			},
		}
		assert.Equal(t, exp, result, "MarginAccountNewOCO should decode every field")
		return
	}
	assert.NotZero(t, result.OrderListID, "MarginAccountNewOCO should return an order list ID")
}

func TestMarginAccountRepay(t *testing.T) {
	t.Parallel()
	_, err := e.MarginAccountRepay(t.Context(), currency.EMPTYCODE, 0.001)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "MarginAccountRepay must reject an empty asset")
	_, err = e.MarginAccountRepay(t.Context(), currency.USDT, 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "MarginAccountRepay must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.MarginAccountRepay(t.Context(), currency.USDT, 0.001)
	require.NoError(t, err, "MarginAccountRepay must not error")
	if mockTests {
		exp := &PMTransactionResponse{
			TransactionID: 100000002,
		}
		assert.Equal(t, exp, result, "MarginAccountRepay should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "MarginAccountRepay should return a transaction ID")
}

func TestGetPMMarginAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMMarginAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPMMarginAccountTradeList must reject a nil request")
	_, err = e.GetPMMarginAccountTradeList(t.Context(), &PMMarginTradeListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPMMarginAccountTradeList must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetPMMarginAccountTradeList(t.Context(), &PMMarginTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPMMarginAccountTradeList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPMMarginAccountTradeList(t.Context(), &PMMarginTradeListRequest{Symbol: marginTradablePair, OrderID: 39324, StartTime: startTime, EndTime: endTime, FromID: 1, Limit: 10})
	require.NoError(t, err, "GetPMMarginAccountTradeList must not error")
	if mockTests {
		exp := []PMMarginTrade{
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
		assert.Equal(t, exp, result, "GetPMMarginAccountTradeList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPMMarginAccountTradeList should return trades")
}

func TestGetAllCMConditionalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllCMConditionalOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllCMConditionalOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetAllCMConditionalOrders(t.Context(), &CMConditionalOrdersRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllCMConditionalOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllCMConditionalOrders(t.Context(), &CMConditionalOrdersRequest{Symbol: coinmTradablePair, StrategyID: 123445, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetAllCMConditionalOrders must not error")
	if mockTests {
		exp := []CMConditionalOrderRecord{
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
		assert.Equal(t, exp, result, "GetAllCMConditionalOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllCMConditionalOrders should return orders")
}

func TestGetAllCMOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllCMOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllCMOrders must reject a nil request")
	_, err = e.GetAllCMOrders(t.Context(), &CMOrdersRequest{Limit: 20})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAllCMOrders must reject a request without a symbol or pair")
	startTime, endTime := getTime()
	_, err = e.GetAllCMOrders(t.Context(), &CMOrdersRequest{Pair: currency.NewCode("BTCUSD"), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllCMOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllCMOrders(t.Context(), &CMOrdersRequest{Symbol: coinmTradablePair, Pair: currency.NewCode("BTCUSD"), StartTime: startTime, EndTime: endTime, Limit: 20})
	require.NoError(t, err, "GetAllCMOrders must not error")
	if mockTests {
		exp := []CMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetAllCMOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllCMOrders should return orders")
}

func TestGetAllCMOpenConditionalOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllCMOpenConditionalOrders(t.Context(), coinmTradablePair)
	require.NoError(t, err, "GetAllCMOpenConditionalOrders must not error")
	if mockTests {
		exp := []CMConditionalOrderResponse{
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
		assert.Equal(t, exp, result, "GetAllCMOpenConditionalOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllCMOpenConditionalOrders should return orders")
}

func TestGetAllCMOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllCMOpenOrders(t.Context(), coinmTradablePair, currency.NewCode("BTCUSD"))
	require.NoError(t, err, "GetAllCMOpenOrders must not error")
	if mockTests {
		exp := []CMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetAllCMOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllCMOpenOrders should return orders")
}

func TestGetAllUMOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllUMOpenOrders(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetAllUMOpenOrders must not error")
	if mockTests {
		exp := []UMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetAllUMOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllUMOpenOrders should return orders")
}

func TestGetAllMarginAccountOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllMarginAccountOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllMarginAccountOrders must reject a nil request")
	_, err = e.GetAllMarginAccountOrders(t.Context(), &PMMarginOrdersRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAllMarginAccountOrders must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetAllMarginAccountOrders(t.Context(), &PMMarginOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllMarginAccountOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllMarginAccountOrders(t.Context(), &PMMarginOrdersRequest{Symbol: marginTradablePair, OrderID: 1234, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetAllMarginAccountOrders must not error")
	if mockTests {
		exp := []PMMarginOrderResponse{
			{
				ClientOrderID:            "ZwfQzuDIGpceVhKW5DvCmO",
				CummulativeQuoteQuantity: 1500.5,
				ExecutedQuantity:         0.025,
				IcebergQuantity:          0.01,
				IsWorking:                true,
				OrderID:                  12434,
				OriginalQuantity:         0.3,
				Price:                    60020,
				Side:                     "SELL",
				Status:                   "PARTIALLY_FILLED",
				StopPrice:                60000,
				Symbol:                   "BTCUSDT",
				Time:                     types.Time(time.UnixMilli(1562133008725)),
				TimeInForce:              "GTC",
				Type:                     "STOP_LOSS_LIMIT",
				UpdateTime:               types.Time(time.UnixMilli(1562133008725)),
				AccountID:                152950866,
				SelfTradePreventionMode:  "EXPIRE_TAKER",
				PreventedMatchID:         1,
				PreventedQuantity:        0.002,
			},
		}
		assert.Equal(t, exp, result, "GetAllMarginAccountOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllMarginAccountOrders should return orders")
}

func TestGetAllUMOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllUMOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllUMOrders must reject a nil request")
	_, err = e.GetAllUMOrders(t.Context(), &UMOrdersRequest{Limit: 100})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAllUMOrders must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetAllUMOrders(t.Context(), &UMOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllUMOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllUMOrders(t.Context(), &UMOrdersRequest{Symbol: usdtmTradablePair, OrderID: 1, StartTime: startTime, EndTime: endTime, Limit: 100})
	require.NoError(t, err, "GetAllUMOrders must not error")
	if mockTests {
		exp := []UMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetAllUMOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllUMOrders should return orders")
}

func TestGetCMConditionalOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMConditionalOrderHistory(t.Context(), currency.EMPTYPAIR, 123432423, "abc")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCMConditionalOrderHistory must reject an empty symbol")
	_, err = e.GetCMConditionalOrderHistory(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetCMConditionalOrderHistory must reject a request without a strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCMConditionalOrderHistory(t.Context(), coinmTradablePair, 123432423, "abc")
	require.NoError(t, err, "GetCMConditionalOrderHistory must not error")
	if mockTests {
		exp := &CMConditionalOrderHistoryResponse{
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
		assert.Equal(t, exp, result, "GetCMConditionalOrderHistory should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "GetCMConditionalOrderHistory should return the strategy ID")
}

func TestGetOpenCMConditionalOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenCMConditionalOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOpenCMConditionalOrder must reject an empty symbol")
	_, err = e.GetOpenCMConditionalOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOpenCMConditionalOrder must reject a request without a strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetOpenCMConditionalOrder(t.Context(), coinmTradablePair, 1234, "")
	require.NoError(t, err, "GetOpenCMConditionalOrder must not error")
	if mockTests {
		exp := &CMConditionalOrderResponse{
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
		assert.Equal(t, exp, result, "GetOpenCMConditionalOrder should decode every field")
		return
	}
	assert.NotZero(t, result.StrategyID, "GetOpenCMConditionalOrder should return the strategy ID")
}

func TestGetCMOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetCMOpenOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCMOpenOrder must reject an empty symbol")
	_, err = e.GetCMOpenOrder(t.Context(), currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetCMOpenOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCMOpenOrder(t.Context(), coinmTradablePair, 1234, "")
	require.NoError(t, err, "GetCMOpenOrder must not error")
	if mockTests {
		exp := []CMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetCMOpenOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCMOpenOrder should return the order")
}

func TestGetCurrentMarginOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCurrentMarginOpenOrders(t.Context(), marginTradablePair)
	require.NoError(t, err, "GetCurrentMarginOpenOrders must not error")
	if mockTests {
		exp := []PMMarginOrderResponse{
			{
				ClientOrderID:            "ZwfQzuDIGpceVhKW5DvCmO",
				CummulativeQuoteQuantity: 1500.5,
				ExecutedQuantity:         0.025,
				IcebergQuantity:          0.01,
				IsWorking:                true,
				OrderID:                  12434,
				OriginalQuantity:         0.3,
				Price:                    60020,
				Side:                     "SELL",
				Status:                   "PARTIALLY_FILLED",
				StopPrice:                60000,
				Symbol:                   "BTCUSDT",
				Time:                     types.Time(time.UnixMilli(1562133008725)),
				TimeInForce:              "GTC",
				Type:                     "STOP_LOSS_LIMIT",
				UpdateTime:               types.Time(time.UnixMilli(1562133008725)),
				AccountID:                152950866,
				SelfTradePreventionMode:  "EXPIRE_TAKER",
				PreventedMatchID:         1,
				PreventedQuantity:        0.002,
			},
		}
		assert.Equal(t, exp, result, "GetCurrentMarginOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrentMarginOpenOrders should return orders")
}

func TestGetUMOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMOpenOrder(t.Context(), currency.EMPTYPAIR, 1234, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUMOpenOrder must reject an empty symbol")
	_, err = e.GetUMOpenOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetUMOpenOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUMOpenOrder(t.Context(), usdtmTradablePair, 1234, "")
	require.NoError(t, err, "GetUMOpenOrder must not error")
	if mockTests {
		exp := &UMOrderDetailResponse{
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
		assert.Equal(t, exp, result, "GetUMOpenOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "GetUMOpenOrder should return the order ID")
}

func TestGetPMMarginAccountAllOCO(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMMarginAccountAllOCO(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPMMarginAccountAllOCO must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPMMarginAccountAllOCO(t.Context(), &PMMarginAllOCORequest{StartTime: endTime, EndTime: startTime, FromID: 1, Limit: 100})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPMMarginAccountAllOCO must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPMMarginAccountAllOCO(t.Context(), &PMMarginAllOCORequest{StartTime: startTime, EndTime: endTime, FromID: 1, Limit: 100})
	require.NoError(t, err, "GetPMMarginAccountAllOCO must not error")
	if mockTests {
		exp := []PMMarginOCOOrderResponse{
			{
				OrderListID:       27,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
				TransactionTime:   types.Time(time.UnixMilli(1565245656253)),
				Symbol:            "BTCUSDT",
				Orders: []PMMarginOCOOrderLeg{
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
		assert.Equal(t, exp, result, "GetPMMarginAccountAllOCO should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPMMarginAccountAllOCO should return order lists")
}

func TestGetMarginAccountsOpenOCO(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMarginAccountsOpenOCO(t.Context())
	require.NoError(t, err, "GetMarginAccountsOpenOCO must not error")
	if mockTests {
		exp := []PMMarginOCOOrderResponse{
			{
				OrderListID:       31,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
				TransactionTime:   types.Time(time.UnixMilli(1565245656253)),
				Symbol:            "BTCUSDT",
				Orders: []PMMarginOCOOrderLeg{
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
		assert.Equal(t, exp, result, "GetMarginAccountsOpenOCO should decode every field")
		return
	}
	assert.NotNil(t, result, "GetMarginAccountsOpenOCO should return order lists")
}

func TestGetUsersCMForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersCMForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUsersCMForceOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUsersCMForceOrders(t.Context(), &PMForceOrdersRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUsersCMForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUsersCMForceOrders(t.Context(), &PMForceOrdersRequest{Symbol: coinmTradablePair, AutoCloseType: "ADL", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUsersCMForceOrders must not error")
	if mockTests {
		exp := []CMForceOrder{
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
		assert.Equal(t, exp, result, "GetUsersCMForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersCMForceOrders should return orders")
}

func TestGetUsersMarginForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersMarginForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUsersMarginForceOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUsersMarginForceOrders(t.Context(), &PMMarginForceOrdersRequest{StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUsersMarginForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUsersMarginForceOrders(t.Context(), &PMMarginForceOrdersRequest{StartTime: startTime, EndTime: endTime, Current: 1, Size: 5})
	require.NoError(t, err, "GetUsersMarginForceOrders must not error")
	if mockTests {
		exp := &PMMarginForceOrderResponse{
			Rows: []PMMarginForceOrder{
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
		assert.Equal(t, exp, result, "GetUsersMarginForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersMarginForceOrders should return orders")
}

func TestGetUsersUMForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUsersUMForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUsersUMForceOrders must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUsersUMForceOrders(t.Context(), &PMForceOrdersRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUsersUMForceOrders must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUsersUMForceOrders(t.Context(), &PMForceOrdersRequest{Symbol: usdtmTradablePair, AutoCloseType: "LIQUIDATION", StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUsersUMForceOrders must not error")
	if mockTests {
		exp := []UMForceOrder{
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
		assert.Equal(t, exp, result, "GetUsersUMForceOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUsersUMForceOrders should return orders")
}

func TestGetUMAccountTradeList(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetUMAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUMAccountTradeList must reject a nil request")
	_, err = e.GetUMAccountTradeList(t.Context(), &UMAccountTradeListRequest{StartTime: startTime, EndTime: endTime})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUMAccountTradeList must reject an empty symbol")
	_, err = e.GetUMAccountTradeList(t.Context(), &UMAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), FromID: 1, EndTime: endTime})
	require.ErrorIs(t, err, errFromIDWithTimeRange, "GetUMAccountTradeList must reject fromId with a time range")
	_, err = e.GetUMAccountTradeList(t.Context(), &UMAccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUMAccountTradeList must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *UMAccountTradeListRequest
		exp  []UMAccountTrade
	}{
		{name: "timeRange", arg: &UMAccountTradeListRequest{Symbol: usdtmTradablePair, StartTime: startTime, EndTime: endTime, Limit: 10}, exp: []UMAccountTrade{
			{
				Symbol:          "BTCUSDT",
				ID:              67880589,
				OrderID:         270093109,
				Side:            "BUY",
				Price:           28511,
				Quantity:        0.01,
				RealizedPNL:     2.585,
				QuoteQuantity:   285.11,
				Commission:      0.114044,
				CommissionAsset: currency.USDT,
				Time:            types.Time(time.UnixMilli(1680688557875)),
				Buyer:           true,
				Maker:           true,
				PositionSide:    "BOTH",
			},
		}},
		{name: "fromID", arg: &UMAccountTradeListRequest{Symbol: usdtmTradablePair, FromID: 67880589}, exp: []UMAccountTrade{
			{
				Symbol:          "BTCUSDT",
				ID:              67880589,
				OrderID:         270093109,
				Side:            "BUY",
				Price:           28511,
				Quantity:        0.01,
				RealizedPNL:     2.585,
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
			result, err := e.GetUMAccountTradeList(t.Context(), tc.arg)
			require.NoError(t, err, "GetUMAccountTradeList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUMAccountTradeList should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUMAccountTradeList should return trades")
		})
	}
}

func TestGetUMPositionADLQuantileEstimation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetUMPositionADLQuantileEstimation(t.Context(), usdtmTradablePair)
	require.NoError(t, err, "GetUMPositionADLQuantileEstimation must not error")
	if mockTests {
		exp := []UMADLQuantileEstimation{
			{
				Symbol: "BTCUSDT",
				ADLQuantile: UMADLQuantile{
					Long:  3,
					Short: 3,
					Both:  2,
				},
			},
		}
		assert.Equal(t, exp, result, "GetUMPositionADLQuantileEstimation should decode every field")
		return
	}
	assert.NotNil(t, result, "GetUMPositionADLQuantileEstimation should return quantiles")
}

func TestNewUMAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewUMAlgoOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewUMAlgoOrder must reject a nil request")
	arg := &UMAlgoOrderRequest{TriggerPrice: 74000}
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewUMAlgoOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewUMAlgoOrder must reject an invalid side")
	arg.Side = order.Sell
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewUMAlgoOrder must reject an empty order type")
	arg.OrderType = "TAKE_PROFIT"
	arg.Price = 75000
	arg.PriceMatch = "QUEUE"
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPriceMatchWithPrice, "NewUMAlgoOrder must reject a price with priceMatch")
	arg.PriceMatch = ""
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errAlgoTypeRequired, "NewUMAlgoOrder must reject an empty algo type")
	arg.AlgoType = "CONDITIONAL"
	arg.TimeInForce = "GTD"
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errGoodTillDateRequired, "NewUMAlgoOrder must reject a GTD order without a good till date")
	arg.TimeInForce = ""
	arg.ClosePosition = true
	arg.Quantity = 0.01
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionConflict, "NewUMAlgoOrder must reject closePosition with a quantity")
	arg.Quantity = 0
	arg.ReduceOnly = true
	_, err = e.NewUMAlgoOrder(t.Context(), arg)
	require.ErrorIs(t, err, errClosePositionConflict, "NewUMAlgoOrder must reject closePosition with reduceOnly")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		name string
		arg  *UMAlgoOrderRequest
		exp  *UMAlgoOrderResponse
	}{
		{
			name: "takeProfit",
			arg:  &UMAlgoOrderRequest{AlgoType: "CONDITIONAL", Symbol: usdtmTradablePair, Side: order.Sell, PositionSide: "BOTH", OrderType: "TAKE_PROFIT", TimeInForce: "GTD", Quantity: 0.01, Price: 75000, TriggerPrice: 74000, WorkingType: "MARK_PRICE", PriceProtect: true, ReduceOnly: true, ClientAlgoID: "tpOrder1", NewOrderRespType: "ACK", SelfTradePreventionMode: "EXPIRE_MAKER", GoodTillDate: time.UnixMilli(1750500000000)},
			exp: &UMAlgoOrderResponse{
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
			arg:  &UMAlgoOrderRequest{AlgoType: "CONDITIONAL", Symbol: usdtmTradablePair, Side: order.Sell, OrderType: "TRAILING_STOP_MARKET", Quantity: 0.01, ActivatePrice: 76000, CallbackRate: 1},
			exp: &UMAlgoOrderResponse{
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
			arg:  &UMAlgoOrderRequest{AlgoType: "CONDITIONAL", Symbol: usdtmTradablePair, Side: order.Sell, OrderType: "STOP_MARKET", TriggerPrice: 60000, ClosePosition: true},
			exp: &UMAlgoOrderResponse{
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
			result, err := e.NewUMAlgoOrder(t.Context(), tc.arg)
			require.NoError(t, err, "NewUMAlgoOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewUMAlgoOrder should decode every field")
				return
			}
			assert.NotZero(t, result.AlgoID, "NewUMAlgoOrder should return an algo ID")
		})
	}
}

func TestCancelUMAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelUMAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelUMAlgoOrder must reject a request without an algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelUMAlgoOrder(t.Context(), 2146760, "tpOrder1")
	require.NoError(t, err, "CancelUMAlgoOrder must not error")
	if mockTests {
		exp := &CancelUMAlgoOrderResponse{
			Complete: true,
		}
		assert.Equal(t, exp, result, "CancelUMAlgoOrder should decode every field")
		return
	}
	assert.True(t, result.Complete, "CancelUMAlgoOrder should complete")
}

func TestCancelAllUMAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllUMAlgoOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllUMAlgoOpenOrders must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllUMAlgoOpenOrders(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "CancelAllUMAlgoOpenOrders must not error")
	if mockTests {
		assert.Equal(t, &SuccessResponse{Code: 200, Message: "The operation of cancel all open order is done."}, result, "CancelAllUMAlgoOpenOrders should decode every field")
	}
}

// umAlgoOrderDetail is the UM algo order the algo order fixtures return
func umAlgoOrderDetail() UMAlgoOrderDetailResponse {
	return UMAlgoOrderDetailResponse{
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

func TestGetUMAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetUMAlgoOrder must reject a request without an algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUMAlgoOrder(t.Context(), 2146760, "")
	require.NoError(t, err, "GetUMAlgoOrder must not error")
	if mockTests {
		exp := umAlgoOrderDetail()
		assert.Equal(t, &exp, result, "GetUMAlgoOrder should decode every field")
	}
}

func TestGetUMOpenAlgoOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUMOpenAlgoOrders(t.Context(), currency.NewBTCUSDT(), "CONDITIONAL", 2146760)
	require.NoError(t, err, "GetUMOpenAlgoOrders must not error")
	if mockTests {
		exp := []UMAlgoOrder{{
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
		assert.Equal(t, exp, result, "GetUMOpenAlgoOrders should decode every field")
	}
}

func TestGetUMAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUMAlgoOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUMAlgoOrderHistory must reject a nil request")
	_, err = e.GetUMAlgoOrderHistory(t.Context(), &UMAlgoOrderHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetUMAlgoOrderHistory must reject an empty symbol")
	startTime, endTime := getTime()
	_, err = e.GetUMAlgoOrderHistory(t.Context(), &UMAlgoOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUMAlgoOrderHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUMAlgoOrderHistory(t.Context(), &UMAlgoOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), AlgoID: 2146760, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetUMAlgoOrderHistory must not error")
	if mockTests {
		assert.Equal(t, []UMAlgoOrderDetailResponse{umAlgoOrderDetail()}, result, "GetUMAlgoOrderHistory should decode every field")
	}
}
