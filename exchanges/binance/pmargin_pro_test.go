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

func TestPMProBNBTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.PMProBNBTransfer(t.Context(), 0, "TO_UM")
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PMProBNBTransfer must reject a zero amount")
	_, err = e.PMProBNBTransfer(t.Context(), 0.0001, "")
	require.ErrorIs(t, err, errTransferSideRequired, "PMProBNBTransfer must reject an empty transfer side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PMProBNBTransfer(t.Context(), 0.0001, "TO_UM")
	require.NoError(t, err, "PMProBNBTransfer must not error")
	if mockTests {
		exp := &PMTransactionResponse{
			TransactionID: 100000001,
		}
		assert.Equal(t, exp, result, "PMProBNBTransfer should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "PMProBNBTransfer should return a transaction ID")
}

func TestGetPMProAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPMProAutoRepayFuturesStatus(t.Context())
	require.NoError(t, err, "GetPMProAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PMAutoRepayFuturesStatusResponse{
			AutoRepay: true,
		}
		assert.Equal(t, exp, result, "GetPMProAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPMProAutoRepayFuturesStatus should return a status")
}

func TestChangePMProAutoRepayFuturesStatus(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangePMProAutoRepayFuturesStatus(t.Context(), false)
	require.NoError(t, err, "ChangePMProAutoRepayFuturesStatus must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "ChangePMProAutoRepayFuturesStatus should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "ChangePMProAutoRepayFuturesStatus should report success")
}

func TestPMProFundAutoCollection(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PMProFundAutoCollection(t.Context())
	require.NoError(t, err, "PMProFundAutoCollection must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "PMProFundAutoCollection should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "PMProFundAutoCollection should report success")
}

func TestPMProFundCollectionByAsset(t *testing.T) {
	t.Parallel()
	_, err := e.PMProFundCollectionByAsset(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PMProFundCollectionByAsset must reject an empty asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PMProFundCollectionByAsset(t.Context(), currency.LTC)
	require.NoError(t, err, "PMProFundCollectionByAsset must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "PMProFundCollectionByAsset should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "PMProFundCollectionByAsset should report success")
}

func TestGetPMProAccountInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPMProAccountInfo(t.Context())
	require.NoError(t, err, "GetPMProAccountInfo must not error")
	if mockTests {
		exp := &PMProAccountInfoResponse{
			UniMMR:                   5167.92171923,
			AccountEquity:            122607.35137903,
			ActualEquity:             142607.35137903,
			AccountMaintenanceMargin: 23.72469206,
			AccountInitialMargin:     47.44938412,
			TotalAvailableBalance:    "122,559.90199491",
			AccountStatus:            "NORMAL",
			AccountType:              "PM_1",
		}
		assert.Equal(t, exp, result, "GetPMProAccountInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.AccountType, "GetPMProAccountInfo should return the account type")
}

func TestRepayPMProBankruptcyLoan(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayPMProBankruptcyLoan(t.Context(), "SPOT")
	require.NoError(t, err, "RepayPMProBankruptcyLoan must not error")
	if mockTests {
		exp := &PMTransactionResponse{
			TransactionID: 58203331886213500,
		}
		assert.Equal(t, exp, result, "RepayPMProBankruptcyLoan should decode every field")
		return
	}
	assert.NotZero(t, result.TransactionID, "RepayPMProBankruptcyLoan should return a transaction ID")
}

func TestGetPMProBankruptcyLoanAmount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPMProBankruptcyLoanAmount(t.Context())
	require.NoError(t, err, "GetPMProBankruptcyLoanAmount must not error")
	if mockTests {
		exp := &PMProBankruptcyLoanAmountResponse{
			Asset:  currency.BUSD,
			Amount: 579.45,
		}
		assert.Equal(t, exp, result, "GetPMProBankruptcyLoanAmount should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPMProBankruptcyLoanAmount should return an amount")
}

func TestGetPMProNegativeBalanceInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPMProNegativeBalanceInterestHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPMProNegativeBalanceInterestHistory must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetPMProNegativeBalanceInterestHistory(t.Context(), &PMNegativeBalanceInterestHistoryRequest{Asset: currency.ETH, StartTime: endTime, EndTime: startTime, Size: 100})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPMProNegativeBalanceInterestHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPMProNegativeBalanceInterestHistory(t.Context(), &PMNegativeBalanceInterestHistoryRequest{Asset: currency.ETH, StartTime: startTime, EndTime: endTime, Size: 100})
	require.NoError(t, err, "GetPMProNegativeBalanceInterestHistory must not error")
	if mockTests {
		exp := []PMProNegativeBalanceInterest{
			{
				Asset:               currency.ETH,
				Interest:            24.444,
				InterestAccruedTime: types.Time(time.Unix(1744110000, 0)),
				InterestRate:        0.0001164,
				Principal:           210000,
			},
		}
		assert.Equal(t, exp, result, "GetPMProNegativeBalanceInterestHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPMProNegativeBalanceInterestHistory should return interest records")
}

func TestRepayPMProFuturesNegativeBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RepayPMProFuturesNegativeBalance(t.Context(), "MARGIN")
	require.NoError(t, err, "RepayPMProFuturesNegativeBalance must not error")
	if mockTests {
		exp := &PMMessageResponse{
			Message: "success",
		}
		assert.Equal(t, exp, result, "RepayPMProFuturesNegativeBalance should decode every field")
		return
	}
	assert.Equal(t, "success", result.Message, "RepayPMProFuturesNegativeBalance should report success")
}

func TestGetPortfolioMarginAssetLeverage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPortfolioMarginAssetLeverage(t.Context())
	require.NoError(t, err, "GetPortfolioMarginAssetLeverage must not error")
	if mockTests {
		exp := []PMAssetLeverage{
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
		exp := []PMCollateralRate{
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
		exp  []PortfolioMarginAssetIndexPrice
	}{
		{name: "asset", ccy: currency.BTC, exp: []PortfolioMarginAssetIndexPrice{
			{
				Asset:           currency.BTC,
				AssetIndexPrice: 28251.9136906,
				Time:            types.Time(time.UnixMilli(1683518338121)),
			},
		}},
		{name: "allAssets", ccy: currency.EMPTYCODE, exp: []PortfolioMarginAssetIndexPrice{
			{
				Asset:           currency.BTC,
				AssetIndexPrice: 28251.9136906,
				Time:            types.Time(time.UnixMilli(1683518338121)),
			},
			{
				Asset:           currency.ETH,
				AssetIndexPrice: 1890.12345678,
				Time:            types.Time(time.UnixMilli(1683518338121)),
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
