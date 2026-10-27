package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPortfolioMarginProBNBTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginProBNBTransfer(t.Context(), 0, "TO_UM")
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PortfolioMarginProBNBTransfer must reject a zero amount")
	_, err = e.PortfolioMarginProBNBTransfer(t.Context(), 0.0001, "")
	require.ErrorIs(t, err, errTransferSideRequired, "PortfolioMarginProBNBTransfer must reject an empty transfer side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginProBNBTransfer(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err, "PortfolioMarginProBNBTransfer must not error")
	if mockTests {
		exp := &PortfolioMarginTransactionResponse{
			TransactionID: 100000001,
		}
		assert.Equal(t, exp, result, "PortfolioMarginProBNBTransfer should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "PortfolioMarginProBNBTransfer should return a transaction ID")
}

func TestGetPortfolioMarginProAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginProAutoRepayFuturesStatus(t.Context())
	require.NoError(t, err, "GetPortfolioMarginProAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PortfolioMarginAutoRepayFuturesStatusResponse{
			AutoRepay: true,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginProAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginProAutoRepayFuturesStatus should return a status")
}

func TestChangePortfolioMarginProAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangePortfolioMarginProAutoRepayFuturesStatus(t.Context(), false)
	require.NoError(t, err, "ChangePortfolioMarginProAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangePortfolioMarginProAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangePortfolioMarginProAutoRepayFuturesStatus should report success")
}

func TestPortfolioMarginProFundAutoCollection(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginProFundAutoCollection(t.Context())
	require.NoError(t, err, "PortfolioMarginProFundAutoCollection must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "PortfolioMarginProFundAutoCollection should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "PortfolioMarginProFundAutoCollection should report success")
}

func TestPortfolioMarginProFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.PortfolioMarginProFundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PortfolioMarginProFundCollectionByAsset must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PortfolioMarginProFundCollectionByAsset(t.Context(), currency.LTC)
	require.NoError(t, err, "PortfolioMarginProFundCollectionByAsset must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "PortfolioMarginProFundCollectionByAsset should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "PortfolioMarginProFundCollectionByAsset should report success")
}

func TestGetPortfolioMarginProAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginProAccountInfo(t.Context())
	require.NoError(t, err, "GetPortfolioMarginProAccountInfo must not error")
	if mockTests {
		exp := &PortfolioMarginProAccountInfoResponse{
			UniMMR:                   5167.92171923,
			AccountEquity:            122607.35137903,
			ActualEquity:             142607.35137903,
			AccountMaintenanceMargin: 23.72469206,
			AccountInitialMargin:     47.44938412,
			TotalAvailableBalance:    "122,559.90199491",
			AccountStatus:            "NORMAL",
			AccountType:              "PM_1",
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginProAccountInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.AccountType, "GetPortfolioMarginProAccountInfo should return the account type")
}

func TestRepayPortfolioMarginProBankruptcyLoan(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayPortfolioMarginProBankruptcyLoan(t.Context(), "SPOT")
	require.NoError(t, err, "RepayPortfolioMarginProBankruptcyLoan must not error")
	if mockTests {
		exp := &PortfolioMarginTransactionResponse{
			TransactionID: 58203331886213500,
		}
		assert.Equal(t, exp, result, "RepayPortfolioMarginProBankruptcyLoan should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "RepayPortfolioMarginProBankruptcyLoan should return a transaction ID")
}

func TestGetPortfolioMarginProBankruptcyLoanAmount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginProBankruptcyLoanAmount(t.Context())
	require.NoError(t, err, "GetPortfolioMarginProBankruptcyLoanAmount must not error")
	if mockTests {
		exp := &PortfolioMarginProBankruptcyLoanAmountResponse{
			Asset:  currency.BUSD,
			Amount: 579.45,
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginProBankruptcyLoanAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginProBankruptcyLoanAmount should return an amount")
}

func TestGetPortfolioMarginProNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolioMarginProNegativeBalanceInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPortfolioMarginProNegativeBalanceInterestHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPortfolioMarginProNegativeBalanceInterestHistory(t.Context(), &PortfolioMarginNegativeBalanceInterestHistoryRequest{Asset: currency.ETH, StartTime: endTime, EndTime: startTime, Size: 100})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPortfolioMarginProNegativeBalanceInterestHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginProNegativeBalanceInterestHistory(t.Context(), &PortfolioMarginNegativeBalanceInterestHistoryRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Size: 100})
	require.NoError(t, err, "GetPortfolioMarginProNegativeBalanceInterestHistory must not error")
	if mockTests {
		exp := []PortfolioMarginProNegativeBalanceInterestResponse{
			{
				Asset:               currency.ETH,
				Interest:            24.444,
				InterestAccruedTime: types.Time(time.Unix(1744110000, 0)),
				InterestRate:        0.0001164,
				Principal:           210000,
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginProNegativeBalanceInterestHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPortfolioMarginProNegativeBalanceInterestHistory should return interest records")
}

func TestRepayPortfolioMarginProFuturesNegativeBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayPortfolioMarginProFuturesNegativeBalance(t.Context(), "MARGIN")
	require.NoError(t, err, "RepayPortfolioMarginProFuturesNegativeBalance must not error")
	if mockTests {
		exp := &PortfolioMarginMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "RepayPortfolioMarginProFuturesNegativeBalance should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "RepayPortfolioMarginProFuturesNegativeBalance should report success")
}

func TestGetPortfolioMarginAssetLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginAssetLeverage(t.Context())
	require.NoError(t, err, "GetPortfolioMarginAssetLeverage must not error")
	if mockTests {
		exp := []PortfolioMarginAssetLeverageResponse{
			{
				Asset:    currency.USDC,
				Leverage: 10,
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginAssetLeverage should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPortfolioMarginAssetLeverage should return leverages")
}

func TestGetPortfolioMarginCollateralRate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginCollateralRate(t.Context())
	require.NoError(t, err, "GetPortfolioMarginCollateralRate must not error")
	if mockTests {
		exp := []PortfolioMarginCollateralRateResponse{
			{
				Asset:          currency.USDC,
				CollateralRate: 1,
			},
			{
				Asset:          currency.BTC,
				CollateralRate: 0.95,
			},
		}
		assert.Equal(t, exp, result, "GetPortfolioMarginCollateralRate should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetPortfolioMarginCollateralRate should return collateral rates")
}

func TestGetPortfolioMarginAssetIndexPrice(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		ccy  currency.Code
		exp  []PortfolioMarginAssetIndexPriceResponse
	}{
		{name: "asset", ccy: currency.BTC, exp: []PortfolioMarginAssetIndexPriceResponse{
			{
				Asset:                currency.BTC,
				AssetIndexPriceInUSD: 28251.9136906,
				Time:                 types.Time(time.UnixMilli(1683518338121)),
			},
		}},
		{name: "allAssets", ccy: currency.EMPTYCODE, exp: []PortfolioMarginAssetIndexPriceResponse{
			{
				Asset:                currency.BTC,
				AssetIndexPriceInUSD: 28251.9136906,
				Time:                 types.Time(time.UnixMilli(1683518338121)),
			},
			{
				Asset:                currency.ETH,
				AssetIndexPriceInUSD: 1890.12345678,
				Time:                 types.Time(time.UnixMilli(1683518338121)),
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPortfolioMarginAssetIndexPrice(t.Context(), tc.ccy)
			require.NoError(t, err, "GetPortfolioMarginAssetIndexPrice must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPortfolioMarginAssetIndexPrice should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetPortfolioMarginAssetIndexPrice should return index prices")
		})
	}
}
