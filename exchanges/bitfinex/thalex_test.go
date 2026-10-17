package bitfinex

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// testThalexCurrency is USDt on Ethereum, which the docs' Thalex examples transfer
var testThalexCurrency = currency.NewCode("USE")

// testThalexTwoFactorToken returns the docs' example one time password, which the recorded Thalex requests hold
func testThalexTwoFactorToken() *TwoFactorToken {
	return &TwoFactorToken{Method: TwoFactorMethodOneTimePassword, Token: "123456"}
}

// newThalexErrorServerExchange returns an exchange whose requests Bitfinex refuses as the Thalex docs show, with an
// error array and an internal server error status
func newThalexErrorServerExchange(t *testing.T) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`["error",null,"ERR_WRONG_AMOUNT"]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

func TestRequestThalexDeposit(t *testing.T) {
	t.Parallel()
	_, err := e.RequestThalexDeposit(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestThalexDeposit must reject a nil request")
	_, err = e.RequestThalexDeposit(t.Context(), &ThalexTransferRequest{Currency: currency.USDT, Amount: 1, TwoFactorToken: testThalexTwoFactorToken()})
	require.ErrorIs(t, err, errInvalidThalexCurrency, "RequestThalexDeposit must refuse a currency Thalex does not take")

	req := &ThalexTransferRequest{Currency: testThalexCurrency, Amount: 1000, TwoFactorToken: testThalexTwoFactorToken()}
	_, err = newResponseServerExchange(t, `null`).RequestThalexDeposit(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "RequestThalexDeposit should error for a null response")
	_, err = newUnavailableServerExchange(t).RequestThalexDeposit(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "RequestThalexDeposit should return the request failure")
	_, err = newThalexErrorServerExchange(t).RequestThalexDeposit(t.Context(), req)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "RequestThalexDeposit must return the error Bitfinex reports")
	assert.Equal(t, &APIError{Message: "ERR_WRONG_AMOUNT"}, apiErr, "RequestThalexDeposit should decode the reported error")

	if !mockTests {
		t.Skip("Thalex transfers need a live one time password, which a test cannot supply")
	}
	result, err := e.RequestThalexDeposit(t.Context(), req)
	require.NoError(t, err, "RequestThalexDeposit must not error")
	exp := &ThalexTransfer{
		Type:               "deposit",
		DestinationAddress: "0x1d341cR1512C6s1984da95C15BfF2B9F3A29345e1",
		Amount:             1000,
		Currency:           testThalexCurrency,
		CreatedAt:          types.Time(time.UnixMilli(1732125293487)),
		UpdatedAt:          types.Time(time.UnixMilli(1732125295060)),
		ID:                 "f1e86378-d5fb-420f-b429-716b88c64c23",
		Status:             "PENDING",
	}
	assert.Equal(t, exp, result, "RequestThalexDeposit should decode every field")
}

func TestRequestThalexWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.RequestThalexWithdrawal(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestThalexWithdrawal must reject a nil request")
	_, err = e.RequestThalexWithdrawal(t.Context(), &ThalexTransferRequest{Currency: testThalexCurrency, Amount: 90})
	require.ErrorIs(t, err, errTwoFactorTokenRequired, "RequestThalexWithdrawal must require a two factor token")

	req := &ThalexTransferRequest{Currency: testThalexCurrency, Amount: 90, TwoFactorToken: testThalexTwoFactorToken()}
	_, err = newResponseServerExchange(t, `null`).RequestThalexWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "RequestThalexWithdrawal should error for a null response")
	_, err = newUnavailableServerExchange(t).RequestThalexWithdrawal(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "RequestThalexWithdrawal should return the request failure")
	_, err = newThalexErrorServerExchange(t).RequestThalexWithdrawal(t.Context(), req)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "RequestThalexWithdrawal must return the error Bitfinex reports")
	assert.Equal(t, &APIError{Message: "ERR_WRONG_AMOUNT"}, apiErr, "RequestThalexWithdrawal should decode the reported error")

	if !mockTests {
		t.Skip("Thalex transfers need a live one time password, which a test cannot supply")
	}
	result, err := e.RequestThalexWithdrawal(t.Context(), req)
	require.NoError(t, err, "RequestThalexWithdrawal must not error")
	exp := &ThalexTransfer{
		Type:               "withdraw",
		DestinationAddress: "0x1d341cR1512C6s1984da95C15BfF2B9F3A29345e1",
		Amount:             90,
		Currency:           testThalexCurrency,
		CreatedAt:          types.Time(time.UnixMilli(1732126385718)),
		UpdatedAt:          types.Time(time.UnixMilli(1732126385718)),
		ID:                 "f70a4286-6f7c-4af8-931f-3e89f74ebdf3",
		Status:             "PENDING",
	}
	assert.Equal(t, exp, result, "RequestThalexWithdrawal should decode every field")
}

func TestGetThalexFreeTransferCount(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetThalexFreeTransferCount(t.Context())
	var apiErr *APIError
	if !mockTests && errors.As(err, &apiErr) && (apiErr.Message == "ERR_NOT_FOUND_USER" || apiErr.Message == "ERR_NOT_ALLOWED") {
		t.Skip("The account has not completed Thalex onboarding")
	}
	require.NoError(t, err, "GetThalexFreeTransferCount must not error")
	if mockTests {
		exp := &ThalexFreeTransferCountResponse{
			Deposits:    ThalexFreeTransfers{Available: 1, ResetsAt: types.Time(time.UnixMilli(1732147200000))},
			Withdrawals: ThalexFreeTransfers{},
		}
		assert.Equal(t, exp, result, "GetThalexFreeTransferCount should decode every field")
	}

	_, err = newResponseServerExchange(t, `null`).GetThalexFreeTransferCount(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetThalexFreeTransferCount should error for a null response")
	_, err = newUnavailableServerExchange(t).GetThalexFreeTransferCount(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetThalexFreeTransferCount should return the request failure")
}

func TestNewThalexTransferBody(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *ThalexTransferRequest
		err  error
	}{
		{"no currency", &ThalexTransferRequest{Amount: 1, TwoFactorToken: testThalexTwoFactorToken()}, currency.ErrCurrencyCodeEmpty},
		{"USDT, which Thalex takes as USE", &ThalexTransferRequest{Currency: currency.USDT, Amount: 1, TwoFactorToken: testThalexTwoFactorToken()}, errInvalidThalexCurrency},
		{"no amount", &ThalexTransferRequest{Currency: currency.BTC, TwoFactorToken: testThalexTwoFactorToken()}, errInvalidAmount},
		{"a negative amount", &ThalexTransferRequest{Currency: currency.BTC, Amount: -1, TwoFactorToken: testThalexTwoFactorToken()}, errInvalidAmount},
		{"no two factor token", &ThalexTransferRequest{Currency: currency.BTC, Amount: 1}, errTwoFactorTokenRequired},
		{"an invalid two factor token", &ThalexTransferRequest{Currency: currency.BTC, Amount: 1, TwoFactorToken: &TwoFactorToken{Method: TwoFactorMethodU2F}}, errTwoFactorAssertionRequired},
	} {
		_, err := newThalexTransferBody(tc.req)
		assert.ErrorIsf(t, err, tc.err, "newThalexTransferBody should refuse %s", tc.name)
	}

	token := &TwoFactorToken{
		Method: TwoFactorMethodU2F,
		Response: &WebAuthnAssertion{
			ID:       "id1",
			RawID:    "raw1",
			Response: &WebAuthnAssertionResponse{ClientDataJSON: "client", AttestationObject: "attestation", Signature: "signature", UserHandle: "user"},
		},
	}
	body, err := newThalexTransferBody(&ThalexTransferRequest{Currency: currency.NewCode("eth"), Amount: 0.5, TwoFactorToken: token})
	require.NoError(t, err, "newThalexTransferBody must not error")
	assert.Equal(t, &thalexTransferBody{Provider: thalexProvider, Amount: "0.5", Currency: "ETH", TwoFactorToken: token}, body, "newThalexTransferBody should send the currency in upper case")
	encoded, err := json.Marshal(body)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"provider":"thalex","amount":"0.5","ccy":"ETH","tfaToken":{"method":"u2f","response":{"id":"id1","rawId":"raw1","response":{"clientDataJSON":"client","attestationObject":"attestation","signature":"signature","userHandle":"user"}}}}`, string(encoded), "A Thalex transfer body should hold the docs' keys")
}
