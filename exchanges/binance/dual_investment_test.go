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

func TestGetDualInvestmentProductList(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentProductList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDualInvestmentProductList must reject a nil request")
	_, err = e.GetDualInvestmentProductList(t.Context(), &DualInvestmentProductListRequest{ExercisedCoin: currency.USDT, InvestCoin: currency.BNB})
	require.ErrorIs(t, err, errOptionTypeRequired, "GetDualInvestmentProductList must reject an empty option type")
	_, err = e.GetDualInvestmentProductList(t.Context(), &DualInvestmentProductListRequest{OptionType: "CALL", InvestCoin: currency.BNB})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDualInvestmentProductList must reject an empty exercised coin")
	_, err = e.GetDualInvestmentProductList(t.Context(), &DualInvestmentProductListRequest{OptionType: "CALL", ExercisedCoin: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDualInvestmentProductList must reject an empty invest coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentProductList(t.Context(), &DualInvestmentProductListRequest{
		OptionType:    "CALL",
		ExercisedCoin: currency.USDT,
		InvestCoin:    currency.BNB,
		PageSize:      10,
		PageIndex:     1,
	})
	require.NoError(t, err, "GetDualInvestmentProductList must not error")
	if mockTests {
		exp := &DualInvestmentProductListResponse{
			Total: 1,
			List: []DualInvestmentProduct{
				{
					ID:                   "741590",
					InvestCoin:           currency.BNB,
					ExercisedCoin:        currency.USDT,
					StrikePrice:          380,
					Duration:             4,
					SettleDate:           types.Time(time.Unix(1709020800, 0)),
					PurchaseDecimal:      8,
					PurchaseEndTime:      types.Time(time.Unix(1708934400, 0)),
					CanPurchase:          true,
					APR:                  0.6076,
					OrderID:              8257205859,
					MinimumAmount:        0.1,
					MaximumAmount:        25265.7,
					CreateTimestamp:      types.Time(time.Unix(1708560084, 0)),
					OptionType:           "CALL",
					IsAutoCompoundEnable: true,
					AutoCompoundPlanList: []string{
						"STANDARD",
						"ADVANCED",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetDualInvestmentProductList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualInvestmentProductList should return a result")
}

func TestChangeAutoCompoundStatus(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeAutoCompoundStatus(t.Context(), "", "STANDARD")
	require.ErrorIs(t, err, errPositionIDRequired, "ChangeAutoCompoundStatus must reject an empty position ID")
	_, err = e.ChangeAutoCompoundStatus(t.Context(), "123456789", "")
	require.ErrorIs(t, err, errAutoCompoundPlanRequired, "ChangeAutoCompoundStatus must reject an empty plan")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ChangeAutoCompoundStatus(t.Context(), "123456789", "STANDARD")
	require.NoError(t, err, "ChangeAutoCompoundStatus must not error")
	if mockTests {
		exp := &DualInvestmentAutoCompoundStatusResponse{
			PositionID:       "123456789",
			AutoCompoundPlan: "STANDARD",
		}
		assert.Equal(t, exp, result, "ChangeAutoCompoundStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result.PositionID, "ChangeAutoCompoundStatus should return the position ID")
}

func TestCheckDualInvestmentAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.CheckDualInvestmentAccounts(t.Context())
	require.NoError(t, err, "CheckDualInvestmentAccounts must not error")
	if mockTests {
		exp := &DualInvestmentAccountsResponse{
			TotalAmountInBTC:  0.01067982,
			TotalAmountInUSDT: 77.1328923,
		}
		assert.Equal(t, exp, result, "CheckDualInvestmentAccounts should decode every field")
		return
	}
	assert.NotNil(t, result, "CheckDualInvestmentAccounts should return a result")
}

func TestGetDualInvestmentPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentPositions(t.Context(), "PURCHASE_SUCCESS", 10, 1)
	require.NoError(t, err, "GetDualInvestmentPositions must not error")
	if mockTests {
		exp := &DualInvestmentPositionsResponse{
			Total: 1,
			List: []DualInvestmentPosition{
				{
					ID:                 "10160533",
					InvestCoin:         currency.USDT,
					ExercisedCoin:      currency.BNB,
					SubscriptionAmount: 0.5,
					StrikePrice:        330,
					Duration:           4,
					SettleDate:         types.Time(time.Unix(1708416000, 0)),
					PurchaseStatus:     "PURCHASE_SUCCESS",
					APR:                0.0365,
					OrderID:            7973677530,
					PurchaseEndTime:    types.Time(time.Unix(1708329600, 0)),
					OptionType:         "PUT",
					AutoCompoundPlan:   "STANDARD",
					SubscriptionTime:   types.Time(time.Unix(1708243200, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetDualInvestmentPositions should decode every field")
		return
	}
	assert.NotNil(t, result, "GetDualInvestmentPositions should return a result")
}

func TestSubscribeDualInvestmentProducts(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeDualInvestmentProducts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubscribeDualInvestmentProducts must reject a nil request")
	_, err = e.SubscribeDualInvestmentProducts(t.Context(), &SubscribeDualInvestmentProductsRequest{OrderID: 8257205859, DepositAmount: 0.1, AutoCompoundPlan: "STANDARD"})
	require.ErrorIs(t, err, errProductIDRequired, "SubscribeDualInvestmentProducts must reject an empty product ID")
	_, err = e.SubscribeDualInvestmentProducts(t.Context(), &SubscribeDualInvestmentProductsRequest{ID: "741590", DepositAmount: 0.1, AutoCompoundPlan: "STANDARD"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "SubscribeDualInvestmentProducts must reject an empty order ID")
	_, err = e.SubscribeDualInvestmentProducts(t.Context(), &SubscribeDualInvestmentProductsRequest{ID: "741590", OrderID: 8257205859, AutoCompoundPlan: "STANDARD"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SubscribeDualInvestmentProducts must reject a zero deposit amount")
	_, err = e.SubscribeDualInvestmentProducts(t.Context(), &SubscribeDualInvestmentProductsRequest{ID: "741590", OrderID: 8257205859, DepositAmount: 0.1})
	require.ErrorIs(t, err, errAutoCompoundPlanRequired, "SubscribeDualInvestmentProducts must reject an empty plan")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubscribeDualInvestmentProducts(t.Context(), &SubscribeDualInvestmentProductsRequest{
		ID:               "741590",
		OrderID:          8257205859,
		DepositAmount:    0.1,
		AutoCompoundPlan: "STANDARD",
	})
	require.NoError(t, err, "SubscribeDualInvestmentProducts must not error")
	if mockTests {
		exp := &DualInvestmentSubscriptionResponse{
			PositionID:         10208824,
			InvestCoin:         currency.BNB,
			ExercisedCoin:      currency.USDT,
			SubscriptionAmount: 0.1,
			Duration:           4,
			AutoCompoundPlan:   "STANDARD",
			StrikePrice:        380,
			SettleDate:         types.Time(time.Unix(1709020800, 0)),
			PurchaseStatus:     "PURCHASE_SUCCESS",
			APR:                0.6076,
			OrderID:            8257205859,
			PurchaseTime:       types.Time(time.UnixMilli(1708677583874)),
			OptionType:         "CALL",
		}
		assert.Equal(t, exp, result, "SubscribeDualInvestmentProducts should decode every field")
		return
	}
	assert.NotZero(t, result.PositionID, "SubscribeDualInvestmentProducts should return a position ID")
}
