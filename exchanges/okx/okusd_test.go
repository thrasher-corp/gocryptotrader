package okx

import (
	"net/url"
	"strings"
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

func TestGetOKUSDLimits(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOKUSDLimits(contextGenerate())
	require.NoError(t, err, "GetOKUSDLimits must not error")
	if mockTests {
		exp := &OKUSDLimitsResponse{
			SubscriptionLimit: OKUSDSubscriptionLimit{
				MaximumSubscriptionAmount: 45000000,
				PersonalDailyLimit:        5000000,
				PersonalUsedAmount:        500000,
				PlatformDailyLimit:        50000000,
				PlatformUsedAmount:        5000000,
			},
			FastRedemptionLimit:     OKUSDRedemptionLimit{PersonalDailyLimit: 10000, PersonalUsedAmount: 2500, PlatformDailyLimit: 5000000, PlatformUsedAmount: 1000000, FeeRate: 0.001},
			StandardRedemptionLimit: OKUSDRedemptionLimit{PersonalDailyLimit: 1000000, PlatformDailyLimit: 40000000, FeeRate: 0.00025},
			Timestamp:               types.Time(time.UnixMilli(1718500000000)),
		}
		assert.Equal(t, exp, result, "GetOKUSDLimits should decode every field")
	} else {
		assert.NotNil(t, result, "GetOKUSDLimits should return the limits")
	}
}

func TestSubscribeOKUSD(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OKUSDSubscribeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing amount", arg: &OKUSDSubscribeRequest{ClientOrderID: "my-sub-001"}, err: limits.ErrAmountBelowMin},
		{name: "negative amount", arg: &OKUSDSubscribeRequest{Amount: -1000, ClientOrderID: "my-sub-001"}, err: limits.ErrAmountBelowMin},
		{name: "amount below the minimum", arg: &OKUSDSubscribeRequest{Amount: 0.99999999, ClientOrderID: "my-sub-001"}, err: limits.ErrAmountBelowMin},
		{name: "amount beyond eight decimal places", arg: &OKUSDSubscribeRequest{Amount: 1.123456789, ClientOrderID: "my-sub-001"}, err: limits.ErrAmountExceedsStep},
		{name: "missing client order ID", arg: &OKUSDSubscribeRequest{Amount: 1000}, err: order.ErrClientOrderIDMustBeSet},
		{name: "client order ID beyond 32 characters", arg: &OKUSDSubscribeRequest{Amount: 1000, ClientOrderID: strings.Repeat("a", 33)}, err: errInvalidOKUSDClientOrderID},
		{name: "client order ID with an unsupported character", arg: &OKUSDSubscribeRequest{Amount: 1000, ClientOrderID: "my.sub.001"}, err: errInvalidOKUSDClientOrderID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SubscribeOKUSD(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "SubscribeOKUSD should reject an invalid request")
		})
	}
	for _, tc := range []struct {
		name, body string
		arg        *OKUSDSubscribeRequest
	}{
		{name: "request body", arg: &OKUSDSubscribeRequest{Amount: 1000.5, ClientOrderID: "my-sub-001"}, body: `{"amt":"1000.5","clOrdId":"my-sub-001"}`},
		{name: "minimum amount and longest client order ID", arg: &OKUSDSubscribeRequest{Amount: 1, ClientOrderID: "OKUSD_sub-2026-11-23_abcdefghijk"}, body: `{"amt":"1","clOrdId":"OKUSD_sub-2026-11-23_abcdefghijk"}`},
		{name: "eight decimal places", arg: &OKUSDSubscribeRequest{Amount: 1.12345678, ClientOrderID: "my-sub-001"}, body: `{"amt":"1.12345678","clOrdId":"my-sub-001"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/okusd/subscribe", tc.body, []map[string]string{{"ordId": "680012345678901234"}})
			_, err := ex.SubscribeOKUSD(t.Context(), tc.arg)
			require.NoError(t, err, "SubscribeOKUSD must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubscribeOKUSD(contextGenerate(), &OKUSDSubscribeRequest{Amount: 1000, ClientOrderID: "my-sub-001"})
	require.NoError(t, err, "SubscribeOKUSD must not error")
	if mockTests {
		exp := &OKUSDSubscribeResponse{
			OrderID:       "680012345678901234",
			ClientOrderID: "my-sub-001",
			Currency:      currency.USDT,
			Amount:        1000,
			OKUSDAmount:   1000,
			State:         "success",
			Timestamp:     types.Time(time.UnixMilli(1718500000000)),
		}
		assert.Equal(t, exp, result, "SubscribeOKUSD should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "SubscribeOKUSD should return an order ID")
	}
}

func TestRedeemOKUSD(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OKUSDRedeemRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing amount", arg: &OKUSDRedeemRequest{RedeemType: "1", ClientOrderID: "my-redeem-001"}, err: limits.ErrAmountBelowMin},
		{name: "amount below the minimum", arg: &OKUSDRedeemRequest{Amount: 0.5, RedeemType: "1", ClientOrderID: "my-redeem-001"}, err: limits.ErrAmountBelowMin},
		{name: "amount beyond eight decimal places", arg: &OKUSDRedeemRequest{Amount: 1.123456789, RedeemType: "1", ClientOrderID: "my-redeem-001"}, err: limits.ErrAmountExceedsStep},
		{name: "missing redeem type", arg: &OKUSDRedeemRequest{Amount: 1000, ClientOrderID: "my-redeem-001"}, err: errMissingRequiredParameter},
		{name: "redemption history type name", arg: &OKUSDRedeemRequest{Amount: 1000, RedeemType: "fast", ClientOrderID: "my-redeem-001"}, err: errInvalidOKUSDRedeemType},
		{name: "undocumented redeem type", arg: &OKUSDRedeemRequest{Amount: 1000, RedeemType: "3", ClientOrderID: "my-redeem-001"}, err: errInvalidOKUSDRedeemType},
		{name: "missing client order ID", arg: &OKUSDRedeemRequest{Amount: 1000, RedeemType: "1"}, err: order.ErrClientOrderIDMustBeSet},
		{name: "client order ID beyond 32 characters", arg: &OKUSDRedeemRequest{Amount: 1000, RedeemType: "1", ClientOrderID: strings.Repeat("a", 33)}, err: errInvalidOKUSDClientOrderID},
		{name: "client order ID with an unsupported character", arg: &OKUSDRedeemRequest{Amount: 1000, RedeemType: "1", ClientOrderID: "my redeem 001"}, err: errInvalidOKUSDClientOrderID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RedeemOKUSD(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "RedeemOKUSD should reject an invalid request")
		})
	}
	for _, tc := range []struct {
		name, body string
		arg        *OKUSDRedeemRequest
	}{
		{name: "request body", arg: &OKUSDRedeemRequest{Amount: 50000, RedeemType: "2", ClientOrderID: "my-redeem-002"}, body: `{"amt":"50000","redeemType":"2","clOrdId":"my-redeem-002"}`},
		{name: "minimum amount and longest client order ID", arg: &OKUSDRedeemRequest{Amount: 1, RedeemType: "1", ClientOrderID: "OKUSD_rdm-2026-11-23_ABCDEFGHIJK"}, body: `{"amt":"1","redeemType":"1","clOrdId":"OKUSD_rdm-2026-11-23_ABCDEFGHIJK"}`},
		{name: "eight decimal places", arg: &OKUSDRedeemRequest{Amount: 1.12345678, RedeemType: "2", ClientOrderID: "my-redeem-002"}, body: `{"amt":"1.12345678","redeemType":"2","clOrdId":"my-redeem-002"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/okusd/redeem", tc.body, []map[string]string{{"ordId": "680012345678906789"}})
			_, err := ex.RedeemOKUSD(t.Context(), tc.arg)
			require.NoError(t, err, "RedeemOKUSD must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemOKUSD(contextGenerate(), &OKUSDRedeemRequest{Amount: 1000, RedeemType: "1", ClientOrderID: "my-redeem-001"})
	require.NoError(t, err, "RedeemOKUSD must not error")
	if mockTests {
		exp := &OKUSDRedeemResponse{
			OrderID:                 "680012345678905678",
			ClientOrderID:           "my-redeem-001",
			Currency:                currency.NewCode("OKUSD"),
			Amount:                  1000,
			Fee:                     1,
			USDTAmount:              999,
			RedeemType:              "1",
			State:                   "success",
			EstimatedSettlementTime: types.Time(time.UnixMilli(1718500010000)),
			Timestamp:               types.Time(time.UnixMilli(1718500000000)),
		}
		assert.Equal(t, exp, result, "RedeemOKUSD should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "RedeemOKUSD should return an order ID")
	}
}

func TestGetOKUSDAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOKUSDAccount(contextGenerate())
	require.NoError(t, err, "GetOKUSDAccount must not error")
	if mockTests {
		exp := &OKUSDAccountResponse{Currency: currency.NewCode("OKUSD"), Amount: 10000, TotalEarnAccrual: 123.456789, Timestamp: types.Time(time.UnixMilli(1718500000000))}
		assert.Equal(t, exp, result, "GetOKUSDAccount should decode every field")
	} else {
		assert.NotNil(t, result, "GetOKUSDAccount should return the account")
	}
}

func TestGetOKUSDSubscriptionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOKUSDSubscriptionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOKUSDSubscriptionHistory must reject a nil request")
	_, err = e.GetOKUSDSubscriptionHistory(t.Context(), &OKUSDHistoryRequest{Begin: time.UnixMilli(1718600000000), End: time.UnixMilli(1718400000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOKUSDSubscriptionHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOKUSDSubscriptionHistory(contextGenerate(), &OKUSDHistoryRequest{Limit: 2, Begin: time.UnixMilli(1718400000000), End: time.UnixMilli(1718600000000)})
	require.NoError(t, err, "GetOKUSDSubscriptionHistory must not error")
	if mockTests {
		exp := []OKUSDSubscriptionHistoryResponse{
			{
				OrderID:                  "680012345678901234",
				ClientOrderID:            "my-sub-001",
				Currency:                 currency.USDT,
				Amount:                   1000,
				SettlementCurrency:       currency.NewCode("OKUSD"),
				SettlementCurrencyAmount: 1000,
				Status:                   "success",
				Timestamp:                types.Time(time.UnixMilli(1718500000000)),
			},
		}
		assert.Equal(t, exp, result, "GetOKUSDSubscriptionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetOKUSDSubscriptionHistory should return subscriptions")
	}
}

func TestGetOKUSDRedemptionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOKUSDRedemptionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOKUSDRedemptionHistory must reject a nil request")
	_, err = e.GetOKUSDRedemptionHistory(t.Context(), &OKUSDRedemptionHistoryRequest{Begin: time.UnixMilli(1718600000000), End: time.UnixMilli(1718400000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOKUSDRedemptionHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOKUSDRedemptionHistory(contextGenerate(), &OKUSDRedemptionHistoryRequest{Limit: 1, Begin: time.UnixMilli(1718400000000), End: time.UnixMilli(1718600000000), Type: "fast"})
	require.NoError(t, err, "GetOKUSDRedemptionHistory must not error")
	if mockTests {
		exp := []OKUSDRedemptionHistoryResponse{
			{
				OrderID:                  "680012345678905678",
				ClientOrderID:            "my-rdm-fast-001",
				Currency:                 currency.NewCode("OKUSD"),
				Amount:                   1000,
				Fee:                      1,
				SettlementCurrency:       currency.USDT,
				SettlementCurrencyAmount: 999,
				Type:                     "fast",
				Status:                   "success",
				EstimatedSettlementTime:  types.Time(time.UnixMilli(1718500010000)),
				Timestamp:                types.Time(time.UnixMilli(1718500000000)),
			},
		}
		assert.Equal(t, exp, result, "GetOKUSDRedemptionHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetOKUSDRedemptionHistory should return redemptions")
	}
}

func TestGetOKUSDRewardsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOKUSDRewardsHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOKUSDRewardsHistory must reject a nil request")
	_, err = e.GetOKUSDRewardsHistory(t.Context(), &OKUSDHistoryRequest{Begin: time.UnixMilli(1718600000000), End: time.UnixMilli(1718400000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOKUSDRewardsHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOKUSDRewardsHistory(contextGenerate(), &OKUSDHistoryRequest{Limit: 7, Begin: time.UnixMilli(1718400000000), End: time.UnixMilli(1718600000000)})
	require.NoError(t, err, "GetOKUSDRewardsHistory must not error")
	if mockTests {
		exp := []OKUSDRewardsHistoryResponse{
			{Currency: currency.USDT, EarnedAmount: 1.14246575, Amount: 10000, APR: 0.0418, Timestamp: types.Time(time.UnixMilli(1718500000000))},
		}
		assert.Equal(t, exp, result, "GetOKUSDRewardsHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetOKUSDRewardsHistory should return rewards")
	}
}

func TestGetOKUSDAPRHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOKUSDAPRHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOKUSDAPRHistory must reject a nil request")
	_, err = e.GetOKUSDAPRHistory(t.Context(), &OKUSDHistoryRequest{Begin: time.UnixMilli(1718600000000), End: time.UnixMilli(1718400000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOKUSDAPRHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOKUSDAPRHistory(contextGenerate(), &OKUSDHistoryRequest{Limit: 10, Begin: time.UnixMilli(1718400000000), End: time.UnixMilli(1718600000000)})
	require.NoError(t, err, "GetOKUSDAPRHistory must not error")
	if mockTests {
		exp := []OKUSDAPRHistoryResponse{
			{APR: 0.0418, Timestamp: types.Time(time.UnixMilli(1718500000000))},
			{APR: 0.0395, Timestamp: types.Time(time.UnixMilli(1718413600000))},
		}
		assert.Equal(t, exp, result, "GetOKUSDAPRHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetOKUSDAPRHistory should return APR snapshots")
	}
}

func TestOKUSDHistoryValues(t *testing.T) {
	t.Parallel()
	_, err := okusdHistoryValues(0, time.UnixMilli(1718600000000), time.UnixMilli(1718400000000))
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "okusdHistoryValues must reject a window that starts after it ends")
	for _, tc := range []struct {
		name       string
		limit      uint64
		begin, end time.Time
		exp        url.Values
	}{
		{name: "no filter", exp: url.Values{}},
		{name: "single instant", begin: time.UnixMilli(1718500000000), end: time.UnixMilli(1718500000000), exp: url.Values{"begin": {"1718500000000"}, "end": {"1718500000000"}}},
		{name: "begin only", begin: time.UnixMilli(1718400000000), exp: url.Values{"begin": {"1718400000000"}}},
		{
			name:  "every filter",
			limit: 100,
			begin: time.UnixMilli(1718400000000),
			end:   time.UnixMilli(1718600000000),
			exp:   url.Values{"limit": {"100"}, "begin": {"1718400000000"}, "end": {"1718600000000"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := okusdHistoryValues(tc.limit, tc.begin, tc.end)
			require.NoError(t, err, "okusdHistoryValues must not error")
			assert.Equal(t, tc.exp, params, "okusdHistoryValues should encode every set filter")
		})
	}
}
