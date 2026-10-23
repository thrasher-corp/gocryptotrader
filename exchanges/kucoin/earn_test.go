package kucoin

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

// earnProductsTestCase is the currency filter of an Earn product listing and the products it should return
type earnProductsTestCase struct {
	ccy currency.Code
	exp []EarnProductResponse
}

func TestSubscribeToEarnFixedIncomeProduct(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeToEarnFixedIncomeProduct(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubscribeToEarnFixedIncomeProduct must reject a nil request")
	_, err = e.SubscribeToEarnFixedIncomeProduct(t.Context(), &EarnPurchaseRequest{AccountType: "MAIN", Amount: 12.2})
	require.ErrorIs(t, err, errProductIDMissing, "SubscribeToEarnFixedIncomeProduct must reject an empty product ID")
	_, err = e.SubscribeToEarnFixedIncomeProduct(t.Context(), &EarnPurchaseRequest{ProductID: "1232412", AccountType: "MAIN"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubscribeToEarnFixedIncomeProduct must reject a zero amount")
	_, err = e.SubscribeToEarnFixedIncomeProduct(t.Context(), &EarnPurchaseRequest{ProductID: "1232412", Amount: 12.2})
	require.ErrorIs(t, err, errAccountTypeMissing, "SubscribeToEarnFixedIncomeProduct must reject an empty account type")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.SubscribeToEarnFixedIncomeProduct(t.Context(), &EarnPurchaseRequest{ProductID: "1232412", AccountType: "MAIN", Amount: 12.2})
		return err
	})
	assert.JSONEq(t, `{"productId":"1232412","amount":"12.2","accountType":"MAIN"}`, sent.body, "SubscribeToEarnFixedIncomeProduct should send the amount as the string KuCoin documents")

	if !mockTests {
		t.Skip("the product ID is the docs' example, and subscribing to a real product locks the account's funds")
	}
	result, err := e.SubscribeToEarnFixedIncomeProduct(t.Context(), &EarnPurchaseRequest{ProductID: "1232412", AccountType: "MAIN", Amount: 12.2})
	require.NoError(t, err, "SubscribeToEarnFixedIncomeProduct must not error")
	exp := &EarnPurchaseResponse{
		OrderID:            "2767291",
		OrderTransactionID: "6603694",
	}
	assert.Equal(t, exp, result, "SubscribeToEarnFixedIncomeProduct should decode every field")
}

func TestGetEarnRedeemPreviewByHoldingID(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnRedeemPreviewByHoldingID(t.Context(), "", "MAIN")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetEarnRedeemPreviewByHoldingID must reject an empty holding ID")

	holdingID := "12345"
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		holdings, err := e.GetEarnFixedIncomeCurrentHoldings(t.Context(), &EarnAccountHoldingRequest{})
		require.NoError(t, err, "GetEarnFixedIncomeCurrentHoldings must not error")
		require.NotNil(t, holdings, "GetEarnFixedIncomeCurrentHoldings must return a page")
		if len(holdings.Items) == 0 {
			t.Skip("the account has no Earn holdings to preview a redemption of")
		}
		holdingID = holdings.Items[0].OrderID
	}
	result, err := e.GetEarnRedeemPreviewByHoldingID(t.Context(), holdingID, "MAIN")
	require.NoError(t, err, "GetEarnRedeemPreviewByHoldingID must not error")
	if mockTests {
		exp := &EarnRedeemPreviewResponse{
			Currency:              currency.KCS,
			RedeemAmount:          1,
			PenaltyInterestAmount: 0.00012345,
			RedeemPeriod:          3,
			DeliverTime:           types.Time(time.UnixMilli(1729518951000)),
			ManualRedeemable:      true,
			RedeemAll:             true,
		}
		assert.Equal(t, exp, result, "GetEarnRedeemPreviewByHoldingID should decode every field")
		return
	}
	require.NotNil(t, result, "GetEarnRedeemPreviewByHoldingID must return a preview")
	assert.False(t, result.Currency.IsEmpty(), "GetEarnRedeemPreviewByHoldingID should return the holding's currency")
}

func TestRedeemByEarnHoldingID(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemByEarnHoldingID(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RedeemByEarnHoldingID must reject a nil request")
	_, err = e.RedeemByEarnHoldingID(t.Context(), &EarnRedeemRequest{Amount: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RedeemByEarnHoldingID must reject an empty holding ID")
	_, err = e.RedeemByEarnHoldingID(t.Context(), &EarnRedeemRequest{OrderID: "123231"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RedeemByEarnHoldingID must reject a zero amount")

	if !mockTests {
		t.Skip("the holding ID is the docs' example, and redeeming a real holding early can cost the account a penalty")
	}
	result, err := e.RedeemByEarnHoldingID(t.Context(), &EarnRedeemRequest{OrderID: "123231", Amount: 1, FromAccountType: "TRADE", ConfirmPunishRedeem: true})
	require.NoError(t, err, "RedeemByEarnHoldingID must not error")
	exp := &EarnRedeemResponse{
		OrderTransactionID: "6603700",
		DeliverTime:        types.Time(time.UnixMilli(1729517805000)),
		Status:             "PENDING",
		Amount:             1,
	}
	assert.Equal(t, exp, result, "RedeemByEarnHoldingID should decode every field")
}

func TestGetEarnSavingsProducts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []EarnProductResponse{
		{
			ID:                  "2172",
			Currency:            currency.BTC,
			Category:            "DEMAND",
			Type:                "DEMAND",
			Precision:           8,
			ProductUpperLimit:   480,
			UserUpperLimit:      20,
			UserLowerLimit:      0.01,
			LockStartTime:       types.Time(time.UnixMilli(1644807600000)),
			ApplyStartTime:      types.Time(time.UnixMilli(1644807600000)),
			ReturnRate:          0.00047208,
			IncomeCurrency:      currency.BTC,
			ProductRemainAmount: 132.36153083,
			Status:              "ONGOING",
			RedeemType:          "MANUAL",
			IncomeReleaseType:   "DAILY",
			InterestDate:        types.Time(time.UnixMilli(1729267200000)),
		},
		{
			ID:                   "2899",
			Currency:             currency.USDT,
			Category:             "DEMAND",
			Type:                 "TIME",
			Precision:            8,
			ProductUpperLimit:    5000000,
			UserUpperLimit:       100000,
			UserLowerLimit:       10,
			RedeemPeriod:         1,
			LockStartTime:        types.Time(time.UnixMilli(1729267200000)),
			LockEndTime:          types.Time(time.UnixMilli(1731859200000)),
			ApplyStartTime:       types.Time(time.UnixMilli(1729180800000)),
			ApplyEndTime:         types.Time(time.UnixMilli(1729267200000)),
			ReturnRate:           0.06,
			IncomeCurrency:       currency.USDT,
			EarlyRedeemSupported: true,
			ProductRemainAmount:  1250000.5,
			Status:               "ONGOING",
			RedeemType:           "AUTO",
			IncomeReleaseType:    "AFTER",
			InterestDate:         types.Time(time.UnixMilli(1729267200000)),
			Duration:             30,
			NewUserOnly:          true,
		},
	}
	for name, tc := range map[string]earnProductsTestCase{
		"every currency": {ccy: currency.EMPTYCODE, exp: exp},
		"one currency":   {ccy: currency.BTC, exp: exp[:1]},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEarnSavingsProducts(t.Context(), tc.ccy)
			require.NoError(t, err, "GetEarnSavingsProducts must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEarnSavingsProducts should decode every field")
				return
			}
			assert.NotNil(t, result, "GetEarnSavingsProducts should return a product list")
		})
	}
}

func TestGetLimitedTimePromotionProducts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []EarnProductResponse{
		{
			ID:                  "2685",
			Currency:            currency.BTC,
			Category:            "ACTIVITY",
			Type:                "TIME",
			Precision:           8,
			ProductUpperLimit:   50,
			UserUpperLimit:      1,
			UserLowerLimit:      0.001,
			LockStartTime:       types.Time(time.UnixMilli(1702371601000)),
			LockEndTime:         types.Time(time.UnixMilli(1729858405000)),
			ApplyStartTime:      types.Time(time.UnixMilli(1702371600000)),
			ReturnRate:          0.03,
			IncomeCurrency:      currency.BTC,
			ProductRemainAmount: 49.78203998,
			Status:              "ONGOING",
			RedeemType:          "TRANS_DEMAND",
			IncomeReleaseType:   "DAILY",
			InterestDate:        types.Time(time.UnixMilli(1729253605000)),
			Duration:            7,
			NewUserOnly:         true,
		},
		{
			ID:                   "2731",
			Currency:             currency.USDT,
			Category:             "ACTIVITY",
			Type:                 "TIME",
			Precision:            6,
			ProductUpperLimit:    2000000,
			UserUpperLimit:       5000,
			UserLowerLimit:       100,
			RedeemPeriod:         1,
			LockStartTime:        types.Time(time.UnixMilli(1729267200000)),
			LockEndTime:          types.Time(time.UnixMilli(1729872000000)),
			ApplyStartTime:       types.Time(time.UnixMilli(1729180800000)),
			ApplyEndTime:         types.Time(time.UnixMilli(1729872000000)),
			ReturnRate:           0.12,
			IncomeCurrency:       currency.USDT,
			EarlyRedeemSupported: true,
			ProductRemainAmount:  1532000.25,
			Status:               "ONGOING",
			RedeemType:           "AUTO",
			IncomeReleaseType:    "AFTER",
			InterestDate:         types.Time(time.UnixMilli(1729267200000)),
			Duration:             7,
		},
	}
	for name, tc := range map[string]earnProductsTestCase{
		"every currency": {ccy: currency.EMPTYCODE, exp: exp},
		"one currency":   {ccy: currency.BTC, exp: exp[:1]},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLimitedTimePromotionProducts(t.Context(), tc.ccy)
			require.NoError(t, err, "GetLimitedTimePromotionProducts must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetLimitedTimePromotionProducts should decode every field")
				return
			}
			assert.NotNil(t, result, "GetLimitedTimePromotionProducts should return a product list")
		})
	}
}

func TestGetEarnStakingProducts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []EarnProductResponse{
		{
			ID:                  "2535",
			Currency:            currency.STX,
			Category:            "STAKING",
			Type:                "DEMAND",
			Precision:           8,
			ProductUpperLimit:   1000000,
			UserUpperLimit:      10000,
			UserLowerLimit:      1,
			RedeemPeriod:        14,
			LockStartTime:       types.Time(time.UnixMilli(1688614514000)),
			ApplyStartTime:      types.Time(time.UnixMilli(1688614512000)),
			ReturnRate:          0.045,
			IncomeCurrency:      currency.BTC,
			ProductRemainAmount: 254032.90178701,
			Status:              "ONGOING",
			RedeemType:          "MANUAL",
			IncomeReleaseType:   "DAILY",
			InterestDate:        types.Time(time.UnixMilli(1729267200000)),
		},
		{
			ID:                   "2602",
			Currency:             currency.SOL,
			Category:             "STAKING",
			Type:                 "TIME",
			Precision:            8,
			ProductUpperLimit:    50000,
			UserUpperLimit:       500,
			UserLowerLimit:       0.1,
			RedeemPeriod:         2,
			LockStartTime:        types.Time(time.UnixMilli(1729267200000)),
			LockEndTime:          types.Time(time.UnixMilli(1734451200000)),
			ApplyStartTime:       types.Time(time.UnixMilli(1729180800000)),
			ApplyEndTime:         types.Time(time.UnixMilli(1729267200000)),
			ReturnRate:           0.075,
			IncomeCurrency:       currency.SOL,
			EarlyRedeemSupported: true,
			ProductRemainAmount:  12345.6789,
			Status:               "ONGOING",
			RedeemType:           "TRANS_DEMAND",
			IncomeReleaseType:    "DAILY",
			InterestDate:         types.Time(time.UnixMilli(1729267200000)),
			Duration:             60,
			NewUserOnly:          true,
		},
	}
	for name, tc := range map[string]earnProductsTestCase{
		"every currency": {ccy: currency.EMPTYCODE, exp: exp},
		"one currency":   {ccy: currency.STX, exp: exp[:1]},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEarnStakingProducts(t.Context(), tc.ccy)
			require.NoError(t, err, "GetEarnStakingProducts must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEarnStakingProducts should decode every field")
				return
			}
			assert.NotNil(t, result, "GetEarnStakingProducts should return a product list")
		})
	}
}

func TestGetEarnKCSStakingProducts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []EarnProductResponse{
		{
			ID:                  "2611",
			Currency:            currency.KCS,
			Category:            "KCS_STAKING",
			Type:                "DEMAND",
			Precision:           8,
			ProductUpperLimit:   100000000,
			UserUpperLimit:      100000000,
			UserLowerLimit:      1,
			RedeemPeriod:        3,
			LockStartTime:       types.Time(time.UnixMilli(1701252000000)),
			ApplyStartTime:      types.Time(time.UnixMilli(1701252000000)),
			ReturnRate:          0.03471727,
			IncomeCurrency:      currency.KCS,
			ProductRemainAmount: 58065850.54998251,
			Status:              "ONGOING",
			RedeemType:          "MANUAL",
			IncomeReleaseType:   "DAILY",
			InterestDate:        types.Time(time.UnixMilli(1729267200000)),
		},
		{
			ID:                   "2650",
			Currency:             currency.KCS,
			Category:             "KCS_STAKING",
			Type:                 "TIME",
			Precision:            8,
			ProductUpperLimit:    5000000,
			UserUpperLimit:       100000,
			UserLowerLimit:       10,
			RedeemPeriod:         1,
			LockStartTime:        types.Time(time.UnixMilli(1729267200000)),
			LockEndTime:          types.Time(time.UnixMilli(1737043200000)),
			ApplyStartTime:       types.Time(time.UnixMilli(1729180800000)),
			ApplyEndTime:         types.Time(time.UnixMilli(1729267200000)),
			ReturnRate:           0.05,
			IncomeCurrency:       currency.KCS,
			EarlyRedeemSupported: true,
			ProductRemainAmount:  4500000,
			Status:               "ONGOING",
			RedeemType:           "AUTO",
			IncomeReleaseType:    "AFTER",
			InterestDate:         types.Time(time.UnixMilli(1729267200000)),
			Duration:             90,
			NewUserOnly:          true,
		},
	}
	for name, tc := range map[string]earnProductsTestCase{
		"every currency": {ccy: currency.EMPTYCODE, exp: exp},
		"one currency":   {ccy: currency.KCS, exp: exp},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEarnKCSStakingProducts(t.Context(), tc.ccy)
			require.NoError(t, err, "GetEarnKCSStakingProducts must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEarnKCSStakingProducts should decode every field")
				return
			}
			assert.NotNil(t, result, "GetEarnKCSStakingProducts should return a product list")
		})
	}
}

func TestGetEarnETHStakingProducts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	exp := []EarnProductResponse{
		{
			ID:                   "ETH2",
			Currency:             currency.ETH,
			Category:             "ETH2",
			Type:                 "DEMAND",
			Precision:            8,
			ProductUpperLimit:    8557.3597075,
			UserUpperLimit:       8557.3597075,
			UserLowerLimit:       0.01,
			RedeemPeriod:         5,
			LockStartTime:        types.Time(time.UnixMilli(1729255485000)),
			LockEndTime:          types.Time(time.UnixMilli(1767225600000)),
			ApplyStartTime:       types.Time(time.UnixMilli(1729255485000)),
			ApplyEndTime:         types.Time(time.UnixMilli(1767225600000)),
			ReturnRate:           0.028,
			IncomeCurrency:       currency.ETH2,
			EarlyRedeemSupported: true,
			ProductRemainAmount:  8557.3597075,
			Status:               "ONGOING",
			RedeemType:           "MANUAL",
			IncomeReleaseType:    "DAILY",
			InterestDate:         types.Time(time.UnixMilli(1729267200000)),
			Duration:             30,
			NewUserOnly:          true,
		},
	}
	for name, tc := range map[string]earnProductsTestCase{
		"every currency": {ccy: currency.EMPTYCODE, exp: exp},
		"one currency":   {ccy: currency.ETH, exp: exp},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetEarnETHStakingProducts(t.Context(), tc.ccy)
			require.NoError(t, err, "GetEarnETHStakingProducts must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetEarnETHStakingProducts should decode every field")
				return
			}
			assert.NotNil(t, result, "GetEarnETHStakingProducts should return a product list")
		})
	}
}

func TestGetEarnFixedIncomeCurrentHoldings(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnFixedIncomeCurrentHoldings(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEarnFixedIncomeCurrentHoldings must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEarnFixedIncomeCurrentHoldings(t.Context(), &EarnAccountHoldingRequest{Currency: currency.KCS, ProductID: "2650", ProductCategory: "KCS_STAKING", CurrentPage: 1, PageSize: 10})
	require.NoError(t, err, "GetEarnFixedIncomeCurrentHoldings must not error")
	if mockTests {
		exp := &EarnAccountHoldingResponse{
			TotalNumber: 1,
			Items: []EarnAccountHolding{
				{
					OrderID:              "2767291",
					ProductID:            "2650",
					ProductCategory:      "KCS_STAKING",
					ProductType:          "TIME",
					Currency:             currency.KCS,
					IncomeCurrency:       currency.KCS,
					ReturnRate:           0.05,
					HoldAmount:           100,
					RedeemedAmount:       5,
					RedeemingAmount:      10,
					LockStartTime:        types.Time(time.UnixMilli(1729267200000)),
					LockEndTime:          types.Time(time.UnixMilli(1737043200000)),
					PurchaseTime:         types.Time(time.UnixMilli(1729257513000)),
					RedeemPeriod:         1,
					Status:               "REDEEMING",
					EarlyRedeemSupported: true,
				},
			},
			CurrentPage: 1,
			PageSize:    10,
			TotalPage:   1,
		}
		assert.Equal(t, exp, result, "GetEarnFixedIncomeCurrentHoldings should decode every field")
		return
	}
	assert.NotNil(t, result, "GetEarnFixedIncomeCurrentHoldings should return a page")
}

func TestPurchaseStructuredProduct(t *testing.T) {
	t.Parallel()
	_, err := e.PurchaseStructuredProduct(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PurchaseStructuredProduct must reject a nil request")
	_, err = e.PurchaseStructuredProduct(t.Context(), &StructuredProductPurchaseRequest{InvestCurrency: currency.USDT, InvestAmount: 10, AccountType: "TRADE"})
	require.ErrorIs(t, err, errProductIDMissing, "PurchaseStructuredProduct must reject an empty product ID")
	_, err = e.PurchaseStructuredProduct(t.Context(), &StructuredProductPurchaseRequest{ProductID: "1234", InvestAmount: 10, AccountType: "TRADE"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PurchaseStructuredProduct must reject an empty investment currency")
	_, err = e.PurchaseStructuredProduct(t.Context(), &StructuredProductPurchaseRequest{ProductID: "1234", InvestCurrency: currency.USDT, AccountType: "TRADE"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PurchaseStructuredProduct must reject a zero investment amount")
	_, err = e.PurchaseStructuredProduct(t.Context(), &StructuredProductPurchaseRequest{ProductID: "1234", InvestCurrency: currency.USDT, InvestAmount: 10})
	require.ErrorIs(t, err, errAccountTypeMissing, "PurchaseStructuredProduct must reject an empty account type")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.PurchaseStructuredProduct(t.Context(), &StructuredProductPurchaseRequest{ProductID: "1234", InvestCurrency: currency.USDT, InvestAmount: 10, AccountType: "TRADE"})
		return err
	})
	assert.JSONEq(t, `{"productId":"1234","investCurrency":"USDT","investAmount":"10","accountType":"TRADE"}`, sent.body, "PurchaseStructuredProduct should send the investment amount as the string KuCoin documents")

	if !mockTests {
		t.Skip("the product ID is an example, and buying a real structured product locks the account's funds until it settles")
	}
	result, err := e.PurchaseStructuredProduct(t.Context(), &StructuredProductPurchaseRequest{ProductID: "1234", InvestCurrency: currency.USDT, InvestAmount: 10, AccountType: "TRADE"})
	require.NoError(t, err, "PurchaseStructuredProduct must not error")
	exp := &StructuredProductPurchaseResponse{
		OrderID: "2767291",
	}
	assert.Equal(t, exp, result, "PurchaseStructuredProduct should decode every field")
}

func TestGetDualInvestmentProducts(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentProducts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDualInvestmentProducts must reject a nil request")
	_, err = e.GetDualInvestmentProducts(t.Context(), &DualInvestmentProductsRequest{Side: "PUT"})
	require.ErrorIs(t, err, errProductCategoryRequired, "GetDualInvestmentProducts must reject an empty category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentProducts(t.Context(), &DualInvestmentProductsRequest{Category: "DUAL_CLASSIC", StrikeCurrency: currency.BTC, InvestCurrency: currency.USDT, Side: "PUT"})
	require.NoError(t, err, "GetDualInvestmentProducts must not error")
	if mockTests {
		exp := []DualInvestmentProductResponse{
			{
				Category:         "DUAL_CLASSIC",
				ProductID:        "87839",
				SeriesNames:      "BTC-U",
				ProductName:      "BTC-U",
				TargetCurrency:   currency.BTC,
				QuoteCurrency:    currency.USDT,
				InvestCurrency:   currency.USDT,
				StrikeCurrency:   currency.BTC,
				StrikePrice:      118000,
				AnnualRate:       1.1937583349113794,
				ExpirationTime:   types.Time(time.UnixMilli(1753171200000)),
				Side:             "PUT",
				ExpectSettleTime: types.Time(time.UnixMilli(1753178400000)),
				Duration:         1,
				LowerLimit:       10,
				UpperLimit:       1568000,
				AvailableScale:   1568000,
				SoldStatus:       "AVAILABLE",
				Increment:        0.00000001,
			},
			{
				Category:             "DUAL_CLASSIC",
				ProductID:            "87840",
				SeriesNames:          "BTC-U",
				ProductName:          "BTC-U",
				TargetCurrency:       currency.BTC,
				QuoteCurrency:        currency.USDT,
				InvestCurrency:       currency.USDT,
				StrikeCurrency:       currency.BTC,
				StrikePrice:          115000,
				ProtectPrice:         110000,
				AnnualRate:           0.4825,
				ExpirationTime:       types.Time(time.UnixMilli(1753344000000)),
				Side:                 "PUT",
				ExpectSettleTime:     types.Time(time.UnixMilli(1753351200000)),
				Duration:             3,
				LowerLimit:           10,
				UpperLimit:           500000,
				UserAvailableBalance: 1250.5,
				SoldStatus:           "SOLD_OUT",
				Increment:            0.00000001,
			},
		}
		assert.Equal(t, exp, result, "GetDualInvestmentProducts should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualInvestmentProducts should return a product list")
}

func TestGetStructuredProductOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetStructuredProductOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStructuredProductOrders must reject a nil request")
	_, err = e.GetStructuredProductOrders(t.Context(), &StructuredProductOrdersRequest{InvestCurrency: currency.USDT})
	require.ErrorIs(t, err, errProductCategoryRequired, "GetStructuredProductOrders must reject empty categories")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStructuredProductOrders(t.Context(), &StructuredProductOrdersRequest{Categories: []string{"DUAL_CLASSIC", "DUAL_BOOSTER"}, OrderID: "687de2fce2f75e0008cb51ba", InvestCurrency: currency.USDT, CurrentPage: 1, PageSize: 10})
	require.NoError(t, err, "GetStructuredProductOrders must not error")
	if mockTests {
		exp := &StructuredProductOrdersResponse{
			CurrentPage: 1,
			PageSize:    10,
			TotalNumber: 1,
			TotalPage:   1,
			Items: []StructuredProductOrder{
				{
					OrderID:        "687de2fce2f75e0008cb51ba",
					Category:       "DUAL_CLASSIC",
					Side:           "PUT",
					Duration:       1,
					APR:            4.132676132342477,
					InvestCurrency: currency.USDT,
					StrikeCurrency: currency.ETH,
					InvestAmount:   10,
					SettleAmount:   10.00113224,
					SettleCurrency: currency.USDT,
					SettlePrice:    3801.5,
					TargetPrice:    3775,
					ExpirationTime: types.Time(time.UnixMilli(1753171200000)),
					Status:         "FINISHED",
				},
			},
		}
		assert.Equal(t, exp, result, "GetStructuredProductOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStructuredProductOrders should return a page")
}
