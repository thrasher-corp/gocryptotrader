package coinbase

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// testDerivativesTransferDestination and testDerivativesTransferSource are the subaccounts of the documented
	// transfer
	testDerivativesTransferDestination = 20
	testDerivativesTransferSource      = 10
	// testDerivativesTransferNonce is the documented idempotency nonce
	testDerivativesTransferNonce = "bF1_gfgcsd"
)

// testDerivativesTransferRequest is the documented transfer between subaccounts, with the documented nonce
func testDerivativesTransferRequest() *DerivativesSubmitTransferBetweenSubaccountsRequest {
	return &DerivativesSubmitTransferBetweenSubaccountsRequest{
		Currency:    currency.ETH,
		Amount:      12.1234,
		Destination: testDerivativesTransferDestination,
		Source:      testDerivativesTransferSource,
		Nonce:       testDerivativesTransferNonce,
	}
}

// testDerivativesTransfer is the documented transfer, with the nonce it was requested with
func testDerivativesTransfer() *DerivativesTransfer {
	return &DerivativesTransfer{
		ID:               2,
		Type:             "user",
		Currency:         currency.BTC,
		Amount:           0.2,
		OtherSide:        "2MzyQc5Tkik61kJbEpJV5D5H9VfWHZK9Sgy",
		State:            "cancelled",
		Direction:        "payment",
		CreatedTimestamp: types.Time(time.UnixMilli(1550579457727)),
		UpdatedTimestamp: types.Time(time.UnixMilli(1550579457727)),
		Nonce:            "bF1_gfgcsd",
	}
}

func TestGlobalDerivativesSubmitTransferBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitTransferBetweenSubaccounts must reject a nil request")
	arg := testDerivativesTransferRequest()
	arg.Currency = currency.EMPTYCODE
	_, err = e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SubmitTransferBetweenSubaccounts must require a currency")
	arg.Currency = currency.SOL
	_, err = e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errDerivativesInvalidAccountCurrency, "SubmitTransferBetweenSubaccounts must refuse a currency the method does not document")
	arg = testDerivativesTransferRequest()
	arg.Amount = 0
	_, err = e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "SubmitTransferBetweenSubaccounts must refuse a zero amount")
	arg.Amount = -1
	_, err = e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidAmount, "SubmitTransferBetweenSubaccounts must refuse a negative amount")
	arg = testDerivativesTransferRequest()
	arg.Destination = 0
	_, err = e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errDerivativesTransferDestinationRequired, "SubmitTransferBetweenSubaccounts must require a destination")
	arg = testDerivativesTransferRequest()
	arg.Nonce = "short"
	_, err = e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), arg)
	require.ErrorIs(t, err, errDerivativesInvalidTransferNonce, "SubmitTransferBetweenSubaccounts must refuse a nonce under 8 characters")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("SubmitTransferBetweenSubaccounts needs subaccounts, which live tests do not have")
	}
	resp, err := e.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), testDerivativesTransferRequest())
	require.NoError(t, err, "SubmitTransferBetweenSubaccounts must not error")
	assert.Equal(t, testDerivativesTransfer(), resp, "SubmitTransferBetweenSubaccounts should decode every field")

	ex, sent := newDerivativesAccountParamsServer(t, `{"id":3}`)
	_, err = ex.GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), &DerivativesSubmitTransferBetweenSubaccountsRequest{Currency: currency.NewCode("usdc"), Amount: 50, Destination: testDerivativesAccountIsolatedSubaccountID})
	require.NoError(t, err, "SubmitTransferBetweenSubaccounts must not error from the requesting account")
	params := sent()
	require.Len(t, params, 1, "SubmitTransferBetweenSubaccounts must send one request")
	assert.JSONEq(t, `{"currency":"USDC","amount":50,"destination":123456}`, params[0], "SubmitTransferBetweenSubaccounts should leave the default source and an empty nonce out")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), testDerivativesTransferRequest())
	assert.ErrorIs(t, err, common.ErrNoResponse, "SubmitTransferBetweenSubaccounts should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().SubmitTransferBetweenSubaccounts(t.Context(), testDerivativesTransferRequest())
	assert.ErrorIs(t, err, request.ErrBadStatus, "SubmitTransferBetweenSubaccounts should return the request failure")
}

func TestValidateDerivativesTransferNonce(t *testing.T) {
	t.Parallel()
	for _, nonce := range []string{"", strings.Repeat("n", 8), strings.Repeat("n", 128), strings.Repeat("é", 8)} {
		assert.NoErrorf(t, validateDerivativesTransferNonce(nonce), "validateDerivativesTransferNonce should accept a nonce of %d characters", utf8.RuneCountInString(nonce))
	}
	for _, nonce := range []string{strings.Repeat("n", 7), strings.Repeat("n", 129), strings.Repeat("é", 7)} {
		assert.ErrorIsf(t, validateDerivativesTransferNonce(nonce), errDerivativesInvalidTransferNonce, "validateDerivativesTransferNonce should refuse a nonce of %d characters", utf8.RuneCountInString(nonce))
	}
}
