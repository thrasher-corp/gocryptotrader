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
	for _, tc := range earnSubscriptionRequestErrors {
		// Serial and in order, as the valid request below goes to the same function
		t.Run(tc.name, func(t *testing.T) {
			_, err := earnSubscription(tc.req)
			assert.ErrorIs(t, err, tc.err, "earnSubscription should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		body, err := earnSubscription(&EarnSubscriptionRequest{Currency: currency.NewCode("eth"), EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10})
		require.NoError(t, err, "earnSubscription must not error")
		assert.Equal(t, &earnSubscriptionBody{Currency: "ETH", EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10}, body, "earnSubscription should upper case the currency")
	})
}

// earnSubscriptionRequestErrors are the Earn subscription requests refused before they are sent
var earnSubscriptionRequestErrors = []struct {
	name string
	req  *EarnSubscriptionRequest
	err  error
}{
	{name: "nil request", req: nil, err: common.ErrNilPointer},
	{name: "without a currency", req: &EarnSubscriptionRequest{}, err: currency.ErrCurrencyCodeEmpty},
	{name: "without an earn type", req: &EarnSubscriptionRequest{Currency: currency.ETH}, err: errEarnTypeRequired},
	{name: "without an earn term", req: &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking}, err: errEarnTermRequired},
	{name: "without an amount", req: &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible}, err: order.ErrAmountIsInvalid},
}

func TestEarnSubscribe(t *testing.T) {
	t.Parallel()
	for _, tc := range earnSubscriptionRequestErrors {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, e.EarnSubscribe(t.Context(), tc.req), tc.err, "EarnSubscribe should return the correct error")
		})
	}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("The mocked subscription is not sent live, to avoid moving funds")
		}
		assert.NoError(t, e.EarnSubscribe(t.Context(), &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10}), "EarnSubscribe should not error")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		err := e.EarnSubscribe(t.Context(), &EarnSubscriptionRequest{Currency: unlistedCurrency, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10})
		requireRefusedOnMerits(t, err)
	})
}

func TestEarnUnsubscribe(t *testing.T) {
	t.Parallel()
	for _, tc := range earnSubscriptionRequestErrors {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, e.EarnUnsubscribe(t.Context(), tc.req), tc.err, "EarnUnsubscribe should return the correct error")
		})
	}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("The mocked unsubscription is not sent live, to avoid moving funds")
		}
		assert.NoError(t, e.EarnUnsubscribe(t.Context(), &EarnSubscriptionRequest{Currency: currency.ETH, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10}), "EarnUnsubscribe should not error")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		err := e.EarnUnsubscribe(t.Context(), &EarnSubscriptionRequest{Currency: unlistedCurrency, EarnType: EarnTypeStaking, EarnTerm: EarnTermFlexible, Amount: 10})
		requireRefusedOnMerits(t, err)
	})
}

func TestSetEarnSubscriptionSetting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *EarnSubscriptionSettingRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a setting", req: &EarnSubscriptionSettingRequest{}, err: errEarnSettingRequired},
		{name: "without a currency", req: &EarnSubscriptionSettingRequest{Setting: EarnSettingOptIn}, err: currency.ErrCurrencyCodeEmpty},
		{name: "without an earn type", req: &EarnSubscriptionSettingRequest{Setting: EarnSettingOptIn, Currency: currency.ADA}, err: errEarnTypeRequired},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, e.SetEarnSubscriptionSetting(t.Context(), tc.req), tc.err, "SetEarnSubscriptionSetting should return the correct error")
		})
	}
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("The mocked setting is not sent live, to avoid changing the account's settings")
		}
		err := e.SetEarnSubscriptionSetting(t.Context(), &EarnSubscriptionSettingRequest{Setting: EarnSettingOptIn, Currency: currency.ADA, EarnType: EarnTypeStaking})
		assert.NoError(t, err, "SetEarnSubscriptionSetting should not error")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		err := e.SetEarnSubscriptionSetting(t.Context(), &EarnSubscriptionSettingRequest{Setting: EarnSettingOptIn, Currency: unlistedCurrency, EarnType: EarnTypeStaking})
		requireRefusedOnMerits(t, err)
	})
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
