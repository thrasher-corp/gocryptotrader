package kucoin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// utaVIPLendingAccountTypes are the account types the UTA VIP lending tests query: the classic account, which the
// endpoints default to, and the unified account
var utaVIPLendingAccountTypes = []struct {
	name, accountType string
}{
	{name: "classic"},
	{name: "unified", accountType: "UNIFIED"},
}

func TestGetUTAVIPLendingCollateralRatio(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []UTAVIPLendingCollateralRatioResponse{
		{
			Currency: currency.NewCode("VET"),
			USDTLevels: []UTAVIPLendingCollateralLevel{
				{
					Right:        100000,
					DiscountRate: 1,
				},
				{
					Left:         100000,
					Right:        200000,
					DiscountRate: 0.7,
				},
			},
		},
		{
			Currency: currency.USDT,
			USDTLevels: []UTAVIPLendingCollateralLevel{
				{
					Right:        999999999999,
					DiscountRate: 1,
				},
			},
		},
	}
	for _, tc := range utaVIPLendingAccountTypes {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAVIPLendingCollateralRatio(t.Context(), tc.accountType)
			require.NoError(t, err, "GetUTAVIPLendingCollateralRatio must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAVIPLendingCollateralRatio should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetUTAVIPLendingCollateralRatio should return currencies")
		})
	}
}

func TestGetUTAVIPLendingLoanInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := &UTAVIPLendingLoanInfoResponse{
		ParentUID: "100000001",
		Orders: []UTAVIPLendingLoanOrder{
			{
				OrderID:   "47",
				Principal: 1000,
				Interest:  0.5,
				Currency:  currency.USDT,
			},
		},
		LTV: UTAVIPLendingLTV{
			TransferLTV:           0.8,
			OnlyClosePositionLTV:  0.85,
			DelayedLiquidationLTV: 0.85,
			InstantLiquidationLTV: 0.9,
			CurrentLTV:            0.3349,
		},
		TotalMarginAmount:    2986.24604,
		TransferMarginAmount: 1250.00000012,
		Margins: []UTAVIPLendingMargin{
			{
				MarginCurrency: currency.USDT,
				MarginQuantity: 2986.24604,
				MarginFactor:   1,
			},
		},
	}
	for _, tc := range utaVIPLendingAccountTypes {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAVIPLendingLoanInfo(t.Context(), tc.accountType)
			require.NoError(t, err, "GetUTAVIPLendingLoanInfo must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAVIPLendingLoanInfo should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAVIPLendingLoanInfo should return the loan")
		})
	}
}

func TestGetUTAVIPLendingAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []UTAVIPLendingAccountResponse{
		{
			UID:            "100000001",
			MarginCurrency: currency.USDT,
			MarginQuantity: 2986.24604,
			MarginFactor:   1,
			AccountType:    "UNIFIED",
			IsParent:       true,
		},
	}
	for _, tc := range utaVIPLendingAccountTypes {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAVIPLendingAccounts(t.Context(), tc.accountType)
			require.NoError(t, err, "GetUTAVIPLendingAccounts must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetUTAVIPLendingAccounts should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetUTAVIPLendingAccounts should return accounts")
		})
	}
}

func TestUTAVIPLendingPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/ua/v2/otc-loan/loan", utaVIPLendingPath("/ua/v2/otc-loan/loan", ""), "utaVIPLendingPath should leave the path alone without an account type")
	assert.Equal(t, "/ua/v2/otc-loan/loan?accountType=UNIFIED", utaVIPLendingPath("/ua/v2/otc-loan/loan", "UNIFIED"), "utaVIPLendingPath should add the account type")
}
