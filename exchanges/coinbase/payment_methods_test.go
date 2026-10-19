package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetPaymentMethod(t *testing.T) {
	t.Parallel()
	_, err := e.GetPaymentMethod(t.Context(), "")
	require.ErrorIs(t, err, errPaymentMethodIDRequired, "GetPaymentMethod must require a payment method ID")

	skipLiveWithoutCredentials(t)
	// The docs' example payment method
	paymentMethodID := "8bfc20d7-f7c6-4422-bf07-8243ca4169fe"
	if !mockTests {
		methods, err := e.ListPaymentMethods(t.Context())
		require.NoError(t, err, "ListPaymentMethods must not error")
		if len(methods) == 0 {
			t.Skip("the account has no payment methods")
		}
		paymentMethodID = methods[0].ID
	}
	resp, err := e.GetPaymentMethod(t.Context(), paymentMethodID)
	require.NoError(t, err, "GetPaymentMethod must not error")
	if mockTests {
		exp := &PaymentMethod{
			ID:            "8bfc20d7-f7c6-4422-bf07-8243ca4169fe",
			Type:          "ACH",
			Name:          "ALLY BANK ******1234",
			Currency:      currency.USD,
			Verified:      true,
			AllowBuy:      true,
			AllowSell:     true,
			AllowDeposit:  true,
			AllowWithdraw: true,
			CreatedAt:     time.Date(2021, 5, 31, 9, 59, 59, 0, time.UTC),
			UpdatedAt:     time.Date(2021, 5, 31, 9, 59, 59, 0, time.UTC),
		}
		assert.Equal(t, exp, resp, "GetPaymentMethod should decode every field")
	} else {
		assert.Equal(t, paymentMethodID, resp.ID, "GetPaymentMethod should return the payment method")
	}

	_, err = newJWTCheckingServerExchange(t, `null`).GetPaymentMethod(t.Context(), paymentMethodID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPaymentMethod should error for a null response")
	_, err = newJWTCheckingServerExchange(t, `{"payment_method":null}`).GetPaymentMethod(t.Context(), paymentMethodID)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetPaymentMethod should error for a null payment method")
	_, err = newUnavailableServerExchange(t).GetPaymentMethod(t.Context(), paymentMethodID)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetPaymentMethod should return the request failure")
}

func TestListPaymentMethods(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.ListPaymentMethods(t.Context())
	require.NoError(t, err, "ListPaymentMethods must not error")
	if mockTests {
		exp := []PaymentMethod{
			{
				ID:            "8bfc20d7-f7c6-4422-bf07-8243ca4169fe",
				Type:          "ACH",
				Name:          "ALLY BANK ******1234",
				Currency:      currency.USD,
				Verified:      true,
				AllowBuy:      true,
				AllowSell:     true,
				AllowDeposit:  true,
				AllowWithdraw: true,
				CreatedAt:     time.Date(2021, 5, 31, 9, 59, 59, 0, time.UTC),
				UpdatedAt:     time.Date(2021, 5, 31, 9, 59, 59, 0, time.UTC),
			},
		}
		assert.Equal(t, exp, resp, "ListPaymentMethods should decode every field")
	} else {
		for i := range resp {
			assert.NotEmptyf(t, resp[i].ID, "ListPaymentMethods should return payment method %d's ID", i)
		}
	}

	resp, err = newJWTCheckingServerExchange(t, `{"payment_methods":[]}`).ListPaymentMethods(t.Context())
	require.NoError(t, err, "ListPaymentMethods must not error for an empty response")
	assert.Empty(t, resp, "ListPaymentMethods should return no payment methods for an empty response")
	_, err = newUnavailableServerExchange(t).ListPaymentMethods(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "ListPaymentMethods should return the request failure")
}
