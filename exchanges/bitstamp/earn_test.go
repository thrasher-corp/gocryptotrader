package bitstamp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestEarnSubscription(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *EarnSubscriptionRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &EarnSubscriptionRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{req: &EarnSubscriptionRequest{Currency: currency.ETH}, err: errEarnTypeRequired},
		{req: &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking}, err: errEarnTermRequired},
		{req: &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible}, err: order.ErrAmountIsInvalid},
	} {
		_, err := earnSubscription(tc.req)
		assert.ErrorIs(t, err, tc.err, "earnSubscription should return the correct error")
	}
	body, err := earnSubscription(&EarnSubscriptionRequest{Currency: currency.NewCode("eth"), EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10})
	require.NoError(t, err, "earnSubscription must not error")
	assert.Equal(t, &earnSubscriptionBody{Currency: "ETH", EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10}, body, "earnSubscription should upper case the currency")
}

func TestEarnSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *EarnSubscriptionRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &EarnSubscriptionRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{req: &EarnSubscriptionRequest{Currency: currency.ETH}, err: errEarnTypeRequired},
		{req: &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking}, err: errEarnTermRequired},
		{req: &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible}, err: order.ErrAmountIsInvalid},
	} {
		assert.ErrorIs(t, e.EarnSubscribe(t.Context(), tc.req), tc.err, "EarnSubscribe should return the correct error")
		assert.ErrorIs(t, e.EarnUnsubscribe(t.Context(), tc.req), tc.err, "EarnUnsubscribe should return the correct error")
	}

	if !mockTests {
		t.Skip("Earn subscriptions are not tested live to avoid moving funds")
	}
	req := &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10}
	assert.NoError(t, e.EarnSubscribe(t.Context(), req), "EarnSubscribe should not error")
	assert.NoError(t, e.EarnUnsubscribe(t.Context(), req), "EarnUnsubscribe should not error")
}

func TestSetEarnSubscriptionSetting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *EarnSubscriptionSettingRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &EarnSubscriptionSettingRequest{}, err: errEarnSettingRequired},
		{req: &EarnSubscriptionSettingRequest{Setting: EarnSettingOptIn}, err: currency.ErrCurrencyCodeEmpty},
		{req: &EarnSubscriptionSettingRequest{Setting: EarnSettingOptIn, Currency: currency.ADA}, err: errEarnTypeRequired},
	} {
		assert.ErrorIs(t, e.SetEarnSubscriptionSetting(t.Context(), tc.req), tc.err, "SetEarnSubscriptionSetting should return the correct error")
	}

	if !mockTests {
		t.Skip("SetEarnSubscriptionSetting is not tested live to avoid changing settings")
	}
	err := e.SetEarnSubscriptionSetting(t.Context(), &EarnSubscriptionSettingRequest{Setting: EarnSettingOptIn, Currency: currency.ADA, EarnType: EarnTypeStaking})
	assert.NoError(t, err, "SetEarnSubscriptionSetting should not error")
}

func TestGetEarnTransactions(t *testing.T) {
	t.Parallel()
	_, err := e.GetEarnTransactions(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetEarnTransactions should error on a nil request")
	_, err = e.GetEarnTransactions(t.Context(), &EarnTransactionsRequest{Limit: 1001})
	assert.ErrorIs(t, err, errInvalidLimit, "GetEarnTransactions should error on an invalid limit")

	skipIfLiveWithoutCredentials(t)
	txs, err := e.GetEarnTransactions(t.Context(), &EarnTransactionsRequest{Currency: currency.ETH, Limit: 10})
	require.NoError(t, err, "GetEarnTransactions must not error")
	if mockTests {
		exp := []EarnTransactionResponse{{
			DateTime:      types.DateTime(time.Date(2022, 1, 31, 14, 43, 15, 796000000, time.UTC)),
			Type:          "SUBSCRIBE",
			Amount:        10,
			Currency:      currency.ETH,
			Value:         20000,
			QuoteCurrency: currency.USD,
			Status:        "COMPLETED",
		}}
		assert.Equal(t, exp, txs, "GetEarnTransactions should return the transactions")
	}
}

func TestGetEarnSubscriptions(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	subs, err := e.GetEarnSubscriptions(t.Context())
	require.NoError(t, err, "GetEarnSubscriptions must not error")
	if mockTests {
		exp := []EarnSubscriptionResponse{{
			Currency:             currency.ETH,
			Type:                 EarnTypeStaking,
			Term:                 EarnTermFlexible,
			EstimatedAnnualYield: 3.5,
			DistributionPeriod:   "WEEKLY",
			ActivationPeriod:     "24 hours",
			Amount:               10,
			AvailableAmount:      9.5,
			AmountEarned:         0.12,
		}}
		assert.Equal(t, exp, subs, "GetEarnSubscriptions should return the subscriptions")
	}
}
