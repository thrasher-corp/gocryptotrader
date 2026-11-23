package okx

import (
	"net/url"
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

func TestGetETHStakingProductInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetETHStakingProductInfo(contextGenerate())
	require.NoError(t, err, "GetETHStakingProductInfo must not error")
	if mockTests {
		exp := &ETHStakingProductInfoResponse{FastRedemptionDailyLimit: 100, Rate: 2.23, RedemptionDays: 8, MinimumAmount: 0.001}
		assert.Equal(t, exp, result, "GetETHStakingProductInfo should decode every field")
	} else {
		assert.NotNil(t, result, "GetETHStakingProductInfo should return the product")
	}
}

func TestPurchaseETHStaking(t *testing.T) {
	t.Parallel()
	err := e.PurchaseETHStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PurchaseETHStaking must reject a zero amount")
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/staking-defi/eth/purchase", `{"amt":"100"}`, []any{})
		require.NoError(t, ex.PurchaseETHStaking(t.Context(), 100), "PurchaseETHStaking must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.PurchaseETHStaking(contextGenerate(), 100), "PurchaseETHStaking must not error")
}

func TestRedeemETHStaking(t *testing.T) {
	t.Parallel()
	err := e.RedeemETHStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RedeemETHStaking must reject a zero amount")
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/staking-defi/eth/redeem", `{"amt":"100"}`, []any{})
		require.NoError(t, ex.RedeemETHStaking(t.Context(), 100), "RedeemETHStaking must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.RedeemETHStaking(contextGenerate(), 100), "RedeemETHStaking must not error")
}

func TestCancelETHStakingRedemption(t *testing.T) {
	t.Parallel()
	_, err := e.CancelETHStakingRedemption(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelETHStakingRedemption must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelETHStakingRedemption(contextGenerate(), "1234567890")
	require.NoError(t, err, "CancelETHStakingRedemption must not error")
	if mockTests {
		assert.Equal(t, &ETHStakingOrderIDResponse{OrderID: "1234567890"}, result, "CancelETHStakingRedemption should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "CancelETHStakingRedemption should return the order ID")
	}
}

func TestGetETHStakingBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetETHStakingBalance(contextGenerate())
	require.NoError(t, err, "GetETHStakingBalance must not error")
	if mockTests {
		exp := &ETHStakingBalanceResponse{
			Currency:              currency.BETH,
			Amount:                0.63926191,
			LatestInterestAccrual: 0.00006549,
			TotalInterestAccrual:  0.01490596,
			Timestamp:             types.Time(time.UnixMilli(1699257600000)),
		}
		assert.Equal(t, exp, result, "GetETHStakingBalance should decode every field")
	} else {
		assert.NotNil(t, result, "GetETHStakingBalance should return the balance")
	}
}

func TestGetETHStakingPurchaseRedeemHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetETHStakingPurchaseRedeemHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetETHStakingPurchaseRedeemHistory must reject a nil request")
	_, err = e.GetETHStakingPurchaseRedeemHistory(t.Context(), &StakingPurchaseRedeemHistoryRequest{After: time.UnixMilli(1683413170000), Before: time.UnixMilli(1683413172000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetETHStakingPurchaseRedeemHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		arg  *StakingPurchaseRedeemHistoryRequest
		exp  []ETHStakingPurchaseRedeemHistoryResponse
	}{
		{
			name: "pending purchases",
			arg:  &StakingPurchaseRedeemHistoryRequest{Type: "purchase", Status: "pending", Before: time.UnixMilli(1683413170000), Limit: 10},
			exp: []ETHStakingPurchaseRedeemHistoryResponse{
				{Type: "purchase", Amount: 0.6266663, Status: "pending", OrderID: "1234567891", RequestTime: types.Time(time.UnixMilli(1683413171000))},
			},
		},
		{
			name: "successful redemptions",
			arg:  &StakingPurchaseRedeemHistoryRequest{Type: "redeem", Status: "success", After: time.UnixMilli(1683413172000), Before: time.UnixMilli(1683413170000), Limit: 10},
			exp: []ETHStakingPurchaseRedeemHistoryResponse{
				{
					Type:                   "redeem",
					Amount:                 0.6266663,
					RedeemingAmount:        0.6266663,
					Status:                 "success",
					OrderID:                "1234567890",
					RequestTime:            types.Time(time.UnixMilli(1683413171000)),
					CompletedTime:          types.Time(time.UnixMilli(1683672371000)),
					EstimatedCompletedTime: types.Time(time.UnixMilli(1683672371000)),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetETHStakingPurchaseRedeemHistory(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetETHStakingPurchaseRedeemHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetETHStakingPurchaseRedeemHistory should decode every field")
			} else {
				assert.NotNil(t, result, "GetETHStakingPurchaseRedeemHistory should return records")
			}
		})
	}
}

func TestGetETHStakingAPYHistory(t *testing.T) {
	t.Parallel()
	for _, days := range []uint64{0, 366} {
		_, err := e.GetETHStakingAPYHistory(t.Context(), days)
		require.ErrorIsf(t, err, errAPYHistoryDaysOutOfRange, "GetETHStakingAPYHistory must reject %d days", days)
	}

	result, err := e.GetETHStakingAPYHistory(contextGenerate(), 34)
	require.NoError(t, err, "GetETHStakingAPYHistory must not error")
	if mockTests {
		exp := []APYHistoryResponse{
			{Rate: 0.02123893, Timestamp: types.Time(time.UnixMilli(1795398300000))},
			{Rate: 0.02079278, Timestamp: types.Time(time.UnixMilli(1795311900000))},
			{Rate: 0.02129801, Timestamp: types.Time(time.UnixMilli(1795225500000))},
		}
		assert.Equal(t, exp, result, "GetETHStakingAPYHistory should decode every field")
	} else {
		require.NotEmpty(t, result, "GetETHStakingAPYHistory must return history")
		assert.Positive(t, result[0].Rate.Float64(), "GetETHStakingAPYHistory should return a positive APY")
		assert.False(t, result[0].Timestamp.Time().IsZero(), "GetETHStakingAPYHistory should return a timestamp")
	}
}

func TestStakingPurchaseRedeemHistoryRequestValues(t *testing.T) {
	t.Parallel()
	var nilRequest *StakingPurchaseRedeemHistoryRequest
	_, err := nilRequest.values()
	require.ErrorIs(t, err, common.ErrNilPointer, "values must reject a nil request")
	_, err = (&StakingPurchaseRedeemHistoryRequest{After: time.UnixMilli(1683413170000), Before: time.UnixMilli(1683413172000)}).values()
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "values must reject a window that starts after it ends")
	for _, tc := range []struct {
		name string
		arg  *StakingPurchaseRedeemHistoryRequest
		exp  url.Values
	}{
		{name: "no filter", arg: &StakingPurchaseRedeemHistoryRequest{}, exp: url.Values{}},
		{name: "equal cursors", arg: &StakingPurchaseRedeemHistoryRequest{After: time.UnixMilli(1683413171000), Before: time.UnixMilli(1683413171000)}, exp: url.Values{"after": {"1683413171000"}, "before": {"1683413171000"}}},
		{
			name: "every filter",
			arg:  &StakingPurchaseRedeemHistoryRequest{Type: "redeem", Status: "success", After: time.UnixMilli(1683413172000), Before: time.UnixMilli(1683413170000), Limit: 10},
			exp:  url.Values{"type": {"redeem"}, "status": {"success"}, "after": {"1683413172000"}, "before": {"1683413170000"}, "limit": {"10"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := tc.arg.values()
			require.NoError(t, err, "values must not error")
			assert.Equal(t, tc.exp, params, "values should encode every set filter")
		})
	}
}
