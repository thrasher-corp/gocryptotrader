package okx

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

func TestGetOnChainEarnOffers(t *testing.T) {
	t.Parallel()
	_, err := e.GetOnChainEarnOffers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOnChainEarnOffers must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnOffersRequest
		exp  []OnChainEarnOfferResponse
	}{
		{
			name: "every offer",
			arg:  &OnChainEarnOffersRequest{},
			exp: []OnChainEarnOfferResponse{
				{
					Currency:     currency.DOT,
					ProductID:    "101",
					Protocol:     "Polkadot",
					ProtocolType: "defi",
					APY:          0.1767,
					InvestData:   []OnChainEarnOfferInvestment{{Currency: currency.DOT, MinimumAmount: 2}},
					EarningData:  []OnChainEarnOfferEarning{{Currency: currency.DOT, EarningType: "0"}},
					State:        "purchasable",
					RedeemPeriod: []string{"28D", "28D"},
				},
			},
		},
		{
			name: "every filter",
			arg:  &OnChainEarnOffersRequest{ProductID: "101", ProtocolType: "defi", Currency: currency.DOT},
			exp: []OnChainEarnOfferResponse{
				{
					Currency:                 currency.DOT,
					ProductID:                "101",
					Protocol:                 "Polkadot",
					ProtocolType:             "defi",
					Term:                     28,
					APY:                      0.1767,
					EarlyRedeem:              true,
					InvestData:               []OnChainEarnOfferInvestment{{Currency: currency.DOT, Balance: 10.5, MinimumAmount: 2, MaximumAmount: 1000}},
					EarningData:              []OnChainEarnOfferEarning{{Currency: currency.DOT, EarningType: "0"}},
					State:                    "purchasable",
					RedeemPeriod:             []string{"28D", "28D"},
					FastRedemptionDailyLimit: 100,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOnChainEarnOffers(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetOnChainEarnOffers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOnChainEarnOffers should decode every field")
			} else {
				assert.NotNil(t, result, "GetOnChainEarnOffers should return offers")
			}
		})
	}
}

func TestPurchaseOnChainEarn(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnPurchaseRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing product ID", arg: &OnChainEarnPurchaseRequest{Term: 2}, err: errMissingRequiredParameter},
		{name: "missing invest data", arg: &OnChainEarnPurchaseRequest{ProductID: "1234", Term: 2}, err: errMissingRequiredParameter},
		{name: "missing invest currency", arg: &OnChainEarnPurchaseRequest{ProductID: "1234", Term: 2, InvestData: []OnChainEarnCurrencyAmount{{Amount: 1}}}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing invest amount", arg: &OnChainEarnPurchaseRequest{ProductID: "1234", Term: 2, InvestData: []OnChainEarnCurrencyAmount{{Currency: currency.USDT}}}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.PurchaseOnChainEarn(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "PurchaseOnChainEarn should reject an invalid request")
		})
	}
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnPurchaseRequest
		body string
	}{
		{
			name: "every optional parameter",
			arg:  &OnChainEarnPurchaseRequest{ProductID: "1234", InvestData: []OnChainEarnCurrencyAmount{{Currency: currency.ZIL, Amount: 100}}, Term: 30, Tag: "gctEarn01"},
			body: `{"productId":"1234","investData":[{"ccy":"ZIL","amt":"100"}],"term":"30","tag":"gctEarn01"}`,
		},
		{
			name: "required parameters only",
			arg:  &OnChainEarnPurchaseRequest{ProductID: "4005", InvestData: []OnChainEarnCurrencyAmount{{Currency: currency.USDT, Amount: 100}}},
			body: `{"productId":"4005","investData":[{"ccy":"USDT","amt":"100"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/staking-defi/purchase", tc.body, []map[string]string{{"ordId": "754147", "tag": tc.arg.Tag}})
			_, err := ex.PurchaseOnChainEarn(t.Context(), tc.arg)
			require.NoError(t, err, "PurchaseOnChainEarn must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PurchaseOnChainEarn(contextGenerate(), &OnChainEarnPurchaseRequest{
		ProductID: "1234",
		InvestData: []OnChainEarnCurrencyAmount{
			{Currency: currency.BTC, Amount: 100},
			{Currency: currency.ETH, Amount: 100},
		},
		Term: 30,
		Tag:  "gctEarn01",
	})
	require.NoError(t, err, "PurchaseOnChainEarn must not error")
	if mockTests {
		exp := &OnChainEarnOrderIDResponse{OrderID: "754147", Tag: "gctEarn01"}
		assert.Equal(t, exp, result, "PurchaseOnChainEarn should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "PurchaseOnChainEarn should return an order ID")
	}
}

func TestRedeemOnChainEarn(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnRedeemRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", arg: &OnChainEarnRedeemRequest{}, err: order.ErrOrderIDNotSet},
		{name: "missing order ID", arg: &OnChainEarnRedeemRequest{AllowEarlyRedeem: true}, err: order.ErrOrderIDNotSet},
		{name: "missing protocol type", arg: &OnChainEarnRedeemRequest{OrderID: "754147"}, err: errMissingRequiredParameter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RedeemOnChainEarn(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "RedeemOnChainEarn should reject an invalid request")
		})
	}
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnRedeemRequest
		body string
	}{
		{name: "every optional parameter", arg: &OnChainEarnRedeemRequest{OrderID: "754147", ProtocolType: "defi", AllowEarlyRedeem: true}, body: `{"ordId":"754147","protocolType":"defi","allowEarlyRedeem":true}`},
		{name: "required parameters only", arg: &OnChainEarnRedeemRequest{OrderID: "754147", ProtocolType: "defi"}, body: `{"ordId":"754147","protocolType":"defi"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/staking-defi/redeem", tc.body, []map[string]string{{"ordId": "754147", "tag": ""}})
			_, err := ex.RedeemOnChainEarn(t.Context(), tc.arg)
			require.NoError(t, err, "RedeemOnChainEarn must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemOnChainEarn(contextGenerate(), &OnChainEarnRedeemRequest{OrderID: "754147", ProtocolType: "defi", AllowEarlyRedeem: true})
	require.NoError(t, err, "RedeemOnChainEarn must not error")
	if mockTests {
		exp := &OnChainEarnOrderIDResponse{OrderID: "754147", Tag: "gctEarn01"}
		assert.Equal(t, exp, result, "RedeemOnChainEarn should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "RedeemOnChainEarn should return an order ID")
	}
}

func TestCancelOnChainEarnPurchaseOrRedemption(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnCancelRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", arg: &OnChainEarnCancelRequest{}, err: order.ErrOrderIDNotSet},
		{name: "missing order ID", arg: &OnChainEarnCancelRequest{ProtocolType: "defi"}, err: order.ErrOrderIDNotSet},
		{name: "missing protocol type", arg: &OnChainEarnCancelRequest{OrderID: "754147"}, err: errMissingRequiredParameter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelOnChainEarnPurchaseOrRedemption(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "CancelOnChainEarnPurchaseOrRedemption should reject an invalid request")
		})
	}
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/staking-defi/cancel", `{"ordId":"754147","protocolType":"defi"}`, []map[string]string{{"ordId": "754147", "tag": ""}})
		_, err := ex.CancelOnChainEarnPurchaseOrRedemption(t.Context(), &OnChainEarnCancelRequest{OrderID: "754147", ProtocolType: "defi"})
		require.NoError(t, err, "CancelOnChainEarnPurchaseOrRedemption must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOnChainEarnPurchaseOrRedemption(contextGenerate(), &OnChainEarnCancelRequest{OrderID: "754147", ProtocolType: "defi"})
	require.NoError(t, err, "CancelOnChainEarnPurchaseOrRedemption must not error")
	if mockTests {
		exp := &OnChainEarnOrderIDResponse{OrderID: "754147", Tag: "gctEarn01"}
		assert.Equal(t, exp, result, "CancelOnChainEarnPurchaseOrRedemption should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "CancelOnChainEarnPurchaseOrRedemption should return an order ID")
	}
}

func TestGetOnChainEarnActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOnChainEarnActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOnChainEarnActiveOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnActiveOrdersRequest
		exp  []OnChainEarnActiveOrderResponse
	}{
		{
			name: "every order",
			arg:  &OnChainEarnActiveOrdersRequest{},
			exp: []OnChainEarnActiveOrderResponse{
				{
					Currency:           currency.DOT,
					OrderID:            "2413499",
					ProductID:          "101",
					State:              "1",
					Protocol:           "Polkadot",
					ProtocolType:       "defi",
					APY:                0.1014,
					InvestData:         []OnChainEarnInvestment{{Currency: currency.DOT, Amount: 2}},
					EarningData:        []OnChainEarnOrderEarning{{Currency: currency.DOT, EarningType: "0", Earnings: 0.10615025}},
					FastRedemptionData: []OnChainEarnFastRedemption{},
					PurchasedTime:      types.Time(time.UnixMilli(1729839328000)),
				},
			},
		},
		{
			name: "every filter",
			arg:  &OnChainEarnActiveOrdersRequest{ProductID: "4005", ProtocolType: "defi", Currency: currency.USDT, State: "2"},
			exp: []OnChainEarnActiveOrderResponse{
				{
					Currency:     currency.USDT,
					OrderID:      "2213257",
					ProductID:    "4005",
					State:        "2",
					Protocol:     "On-Chain Defi",
					ProtocolType: "defi",
					Term:         30,
					APY:          0.0323,
					InvestData:   []OnChainEarnInvestment{{Currency: currency.USDT, Amount: 1}},
					EarningData: []OnChainEarnOrderEarning{
						{Currency: currency.USDT, EarningType: "0", Earnings: 0.02886582},
						{Currency: currency.COMP, EarningType: "1", Earnings: 0.0000627},
					},
					FastRedemptionData:       []OnChainEarnFastRedemption{{Currency: currency.USDT, RedeemingAmount: 1}},
					PurchasedTime:            types.Time(time.UnixMilli(1725345790000)),
					EstimatedSettlementTime:  types.Time(time.UnixMilli(1725432190000)),
					CancelRedemptionDeadline: types.Time(time.UnixMilli(1725428590000)),
					Tag:                      "usdtEarn01",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOnChainEarnActiveOrders(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetOnChainEarnActiveOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOnChainEarnActiveOrders should decode every field")
			} else {
				assert.NotNil(t, result, "GetOnChainEarnActiveOrders should return orders")
			}
		})
	}
}

func TestGetOnChainEarnOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOnChainEarnOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOnChainEarnOrderHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		arg  *OnChainEarnOrderHistoryRequest
		exp  []OnChainEarnOrderHistoryResponse
	}{
		{
			name: "latest order",
			arg:  &OnChainEarnOrderHistoryRequest{Limit: 1},
			exp: []OnChainEarnOrderHistoryResponse{
				{
					Currency:      currency.DOT,
					OrderID:       "1579252",
					ProductID:     "101",
					State:         "3",
					Protocol:      "Polkadot",
					ProtocolType:  "defi",
					APY:           0.1704,
					InvestData:    []OnChainEarnInvestment{{Currency: currency.DOT, Amount: 2}},
					EarningData:   []OnChainEarnOrderHistoryEarning{{Currency: currency.DOT, EarningType: "0"}},
					PurchasedTime: types.Time(time.UnixMilli(1712908001000)),
					RedeemedTime:  types.Time(time.UnixMilli(1712914294000)),
				},
			},
		},
		{
			name: "every filter",
			arg:  &OnChainEarnOrderHistoryRequest{ProductID: "101", ProtocolType: "defi", Currency: currency.DOT, After: "1579253", Before: "1579251", Limit: 1},
			exp: []OnChainEarnOrderHistoryResponse{
				{
					Currency:      currency.DOT,
					OrderID:       "1579252",
					ProductID:     "101",
					State:         "3",
					Protocol:      "Polkadot",
					ProtocolType:  "defi",
					Term:          28,
					APY:           0.1704,
					InvestData:    []OnChainEarnInvestment{{Currency: currency.DOT, Amount: 2}},
					EarningData:   []OnChainEarnOrderHistoryEarning{{Currency: currency.DOT, EarningType: "0", RealisedEarnings: 0.0125}},
					PurchasedTime: types.Time(time.UnixMilli(1712908001000)),
					RedeemedTime:  types.Time(time.UnixMilli(1712914294000)),
					Tag:           "dotEarn01",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOnChainEarnOrderHistory(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetOnChainEarnOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOnChainEarnOrderHistory should decode every field")
			} else {
				assert.NotNil(t, result, "GetOnChainEarnOrderHistory should return orders")
			}
		})
	}
}
