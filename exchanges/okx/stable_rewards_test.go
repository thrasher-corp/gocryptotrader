package okx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetStableRewardsProductInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetStableRewardsProductInfo(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetStableRewardsProductInfo must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStableRewardsProductInfo(contextGenerate(), currency.USDG)
	require.NoError(t, err, "GetStableRewardsProductInfo must not error")
	if mockTests {
		exp := &StableRewardsProductInfoResponse{
			Details: []StableRewardsProductDetail{
				{
					Currency:                   currency.USDG,
					SettlementCurrency:         currency.USDC,
					SubscriptionFeeRate:        0.0003,
					RedemptionFeeRate:          0.0001,
					MinimumSubscriptionAmount:  1,
					MinimumRedemptionAmount:    0.0000001,
					RemainingSubscriptionQuota: 1000000,
					RemainingRedemptionQuota:   500000,
					CanRedeem:                  true,
				},
				{
					Currency:                   currency.USDG,
					SettlementCurrency:         currency.USDT,
					SubscriptionFeeRate:        0.0003,
					MinimumSubscriptionAmount:  1,
					RemainingSubscriptionQuota: -1,
				},
			},
			Timestamp: types.Time(time.UnixMilli(1718035200000)),
		}
		assert.Equal(t, exp, result, "GetStableRewardsProductInfo should decode every field")
	} else {
		assert.NotNil(t, result, "GetStableRewardsProductInfo should return the product")
	}
}

func TestGetStableRewardsBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStableRewardsBalance(contextGenerate(), currency.USDG)
	require.NoError(t, err, "GetStableRewardsBalance must not error")
	if mockTests {
		exp := &StableRewardsBalanceResponse{
			Details:   []StableRewardsBalanceDetail{{Currency: currency.USDG, Amount: 100, TotalEarnAccrual: 0.0003, State: "earning"}},
			Timestamp: types.Time(time.UnixMilli(1718035200000)),
		}
		assert.Equal(t, exp, result, "GetStableRewardsBalance should decode every field")
	} else {
		assert.NotNil(t, result, "GetStableRewardsBalance should return the balance")
	}
}

func TestGetStableRewardsAPYHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetStableRewardsAPYHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStableRewardsAPYHistory must reject a nil request")
	_, err = e.GetStableRewardsAPYHistory(t.Context(), &StableRewardsAPYHistoryRequest{Days: 30})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetStableRewardsAPYHistory must reject an empty currency")
	_, err = e.GetStableRewardsAPYHistory(t.Context(), &StableRewardsAPYHistoryRequest{Currency: currency.USDG, Days: 101})
	require.ErrorIs(t, err, errAPYHistoryDaysOutOfRange, "GetStableRewardsAPYHistory must reject more than 100 days")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStableRewardsAPYHistory(contextGenerate(), &StableRewardsAPYHistoryRequest{Currency: currency.USDG, Days: 30})
	require.NoError(t, err, "GetStableRewardsAPYHistory must not error")
	if mockTests {
		exp := []APYHistoryResponse{{Rate: 0.004, Timestamp: types.Time(time.UnixMilli(1718035200000))}}
		assert.Equal(t, exp, result, "GetStableRewardsAPYHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetStableRewardsAPYHistory should return history")
	}
}
