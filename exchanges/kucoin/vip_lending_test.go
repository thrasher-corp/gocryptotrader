package kucoin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetVIPLendingCollateralRatio(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetVIPLendingCollateralRatio(t.Context())
	require.NoError(t, err, "GetVIPLendingCollateralRatio must not error")
	if mockTests {
		exp := []VIPLendingCollateralRatioResponse{
			{
				Currency: currency.BTC,
				USDTLevels: []VIPLendingCollateralRatioLevel{
					{
						Right:        20000000,
						DiscountRate: 1,
					},
					{
						Left:         20000000,
						Right:        50000000,
						DiscountRate: 0.95,
					},
					{
						Left:         50000000,
						Right:        100000000,
						DiscountRate: 0.9,
					},
				},
			},
			{
				Currency: currency.USDT,
				USDTLevels: []VIPLendingCollateralRatioLevel{
					{
						Right:        99999999999,
						DiscountRate: 1,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetVIPLendingCollateralRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetVIPLendingCollateralRatio should return collateral ratios")
}

func TestGetInformationOnOffExchangeFundingAndLoans(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInformationOnOffExchangeFundingAndLoans(t.Context())
	require.NoError(t, err, "GetInformationOnOffExchangeFundingAndLoans must not error")
	if mockTests {
		exp := &VIPLendingLoanInfoResponse{
			ParentUID: "100000001",
			Orders: []VIPLendingLoanOrder{
				{
					OrderID:   "671a2be815f4140007a588e1",
					Principal: 100,
					Interest:  0.0274,
					Currency:  currency.USDT,
				},
			},
			LTV: VIPLendingLTV{
				TransferLTV:           0.6,
				OnlyClosePositionLTV:  0.75,
				DelayedLiquidationLTV: 0.75,
				InstantLiquidationLTV: 0.8,
				CurrentLTV:            0.1111,
			},
			TotalMarginAmount:    900,
			TransferMarginAmount: 166.66666666,
			Margins: []VIPLendingMargin{
				{
					MarginCurrency: currency.USDT,
					MarginQuantity: 1000,
					MarginFactor:   0.9,
				},
			},
		}
		assert.Equal(t, exp, result, "GetInformationOnOffExchangeFundingAndLoans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInformationOnOffExchangeFundingAndLoans should return loan information")
}

func TestGetInformationOnAccountInvolvedInOffExchangeLoans(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetInformationOnAccountInvolvedInOffExchangeLoans(t.Context())
	require.NoError(t, err, "GetInformationOnAccountInvolvedInOffExchangeLoans must not error")
	if mockTests {
		exp := []VIPLendingAccountResponse{
			{
				UID:            "100000001",
				MarginCurrency: currency.USDT,
				MarginQuantity: 900,
				MarginFactor:   0.9,
				AccountType:    "TRADE",
				IsParent:       true,
			},
			{
				UID:            "100000002",
				MarginCurrency: currency.BTC,
				MarginQuantity: 0.475,
				MarginFactor:   0.95,
				AccountType:    "CONTRACT",
			},
		}
		assert.Equal(t, exp, result, "GetInformationOnAccountInvolvedInOffExchangeLoans should decode every field")
		return
	}
	assert.NotNil(t, result, "GetInformationOnAccountInvolvedInOffExchangeLoans should return an account list")
}
