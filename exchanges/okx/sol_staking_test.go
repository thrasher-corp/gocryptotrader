package okx

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

func TestGetSOLStakingProductInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSOLStakingProductInfo(contextGenerate())
	require.NoError(t, err, "GetSOLStakingProductInfo must not error")
	if mockTests {
		exp := &SOLStakingProductInfoResponse{FastRedemptionDailyLimit: 240, FastRedemptionAvailable: 240, Rate: 5.57, RedemptionDays: 2, MinimumAmount: 0.01}
		assert.Equal(t, exp, result, "GetSOLStakingProductInfo should decode every field")
	} else {
		assert.NotNil(t, result, "GetSOLStakingProductInfo should return the product")
	}
}

func TestPurchaseSOLStaking(t *testing.T) {
	t.Parallel()
	err := e.PurchaseSOLStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PurchaseSOLStaking must reject a zero amount")
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/staking-defi/sol/purchase", `{"amt":"100"}`, []any{})
		require.NoError(t, ex.PurchaseSOLStaking(t.Context(), 100), "PurchaseSOLStaking must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.PurchaseSOLStaking(contextGenerate(), 100), "PurchaseSOLStaking must not error")
}

func TestRedeemSOLStaking(t *testing.T) {
	t.Parallel()
	err := e.RedeemSOLStaking(t.Context(), 0)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RedeemSOLStaking must reject a zero amount")
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/staking-defi/sol/redeem", `{"amt":"10"}`, []any{})
		require.NoError(t, ex.RedeemSOLStaking(t.Context(), 10), "RedeemSOLStaking must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	require.NoError(t, e.RedeemSOLStaking(contextGenerate(), 10), "RedeemSOLStaking must not error")
}

func TestGetSOLStakingBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSOLStakingBalance(contextGenerate())
	require.NoError(t, err, "GetSOLStakingBalance must not error")
	if mockTests {
		exp := &SOLStakingBalanceResponse{Currency: currency.NewCode("OKSOL"), Amount: 0.01100012, LatestInterestAccrual: 0.00000012, TotalInterestAccrual: 0.00000012}
		assert.Equal(t, exp, result, "GetSOLStakingBalance should decode every field")
	} else {
		assert.NotNil(t, result, "GetSOLStakingBalance should return the balance")
	}
}

func TestGetSOLStakingPurchaseRedeemHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSOLStakingPurchaseRedeemHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSOLStakingPurchaseRedeemHistory must reject a nil request")
	_, err = e.GetSOLStakingPurchaseRedeemHistory(t.Context(), &StakingPurchaseRedeemHistoryRequest{After: time.UnixMilli(1683413170000), Before: time.UnixMilli(1683413172000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSOLStakingPurchaseRedeemHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSOLStakingPurchaseRedeemHistory(contextGenerate(), &StakingPurchaseRedeemHistoryRequest{Type: "redeem", Status: "success", After: time.UnixMilli(1683413172000), Before: time.UnixMilli(1683413170000), Limit: 10})
	require.NoError(t, err, "GetSOLStakingPurchaseRedeemHistory must not error")
	if mockTests {
		exp := []SOLStakingPurchaseRedeemHistoryResponse{
			{
				Type:                   "redeem",
				Amount:                 0.6266663,
				RedeemingAmount:        0.6266663,
				Status:                 "success",
				RequestTime:            types.Time(time.UnixMilli(1683413171000)),
				CompletedTime:          types.Time(time.UnixMilli(1683585971000)),
				EstimatedCompletedTime: types.Time(time.UnixMilli(1683585971000)),
			},
		}
		assert.Equal(t, exp, result, "GetSOLStakingPurchaseRedeemHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetSOLStakingPurchaseRedeemHistory should return records")
	}
}

func TestGetSOLStakingAPYHistory(t *testing.T) {
	t.Parallel()
	for _, days := range []uint64{0, 366} {
		_, err := e.GetSOLStakingAPYHistory(t.Context(), days)
		require.ErrorIsf(t, err, errAPYHistoryDaysOutOfRange, "GetSOLStakingAPYHistory must reject %d days", days)
	}

	result, err := e.GetSOLStakingAPYHistory(contextGenerate(), 7)
	require.NoError(t, err, "GetSOLStakingAPYHistory must not error")
	if mockTests {
		exp := []APYHistoryResponse{
			{Rate: 0.0473, Timestamp: types.Time(time.UnixMilli(1795276800000))},
			{Rate: 0.0488, Timestamp: types.Time(time.UnixMilli(1795190400000))},
			{Rate: 0.0488, Timestamp: types.Time(time.UnixMilli(1795183200000))},
		}
		assert.Equal(t, exp, result, "GetSOLStakingAPYHistory should decode every field")
	} else {
		require.NotEmpty(t, result, "GetSOLStakingAPYHistory must return history")
		assert.Positive(t, result[0].Rate.Float64(), "GetSOLStakingAPYHistory should return a positive APY")
		assert.False(t, result[0].Timestamp.Time().IsZero(), "GetSOLStakingAPYHistory should return a timestamp")
	}
}
