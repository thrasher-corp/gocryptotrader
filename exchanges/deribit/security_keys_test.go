package deribit

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

const (
	// testSecurityKeyChallenge and testSecurityKeyCode are the challenge Deribit's security key documentation sends and
	// the two-factor authentication code it confirms the challenge with
	testSecurityKeyChallenge = "+Di4SKN9VykrSoHlZO2KF3LEyEZF4ih9CZXVuudQiKQ="
	testSecurityKeyCode      = "602051"
	// testSecurityKeyChallengeResult is the result Deribit's documentation sends in place of a challenged request's
	testSecurityKeyChallengeResult = `{"security_keys":[{"type":"tfa","name":"tfa"}],"security_key_authorization_required":true,"rp_id":"test.deribit.com","challenge":"` + testSecurityKeyChallenge + `"}`
	// mockChallengedAPIKeyName names the API key whose mock creation Deribit challenges
	mockChallengedAPIKeyName = "challenged_key"
)

// testSecurityKeyChallengeError returns the documented security key challenge
func testSecurityKeyChallengeError() *SecurityKeyChallengeError {
	return &SecurityKeyChallengeError{
		AuthorisationRequired: true,
		Challenge:             testSecurityKeyChallenge,
		RelyingPartyID:        "test.deribit.com",
		SecurityKeys:          []SecurityKey{{Type: "tfa", Name: "tfa"}},
	}
}

func TestSecurityKeyChallengeErrorError(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "security key authorisation required: tfa", testSecurityKeyChallengeError().Error(), "Error should report the security key types")
	assert.Equal(t, "security key authorisation required", new(SecurityKeyChallengeError).Error(), "Error should report a challenge without security keys")
}

func TestSecurityKeyChallengeErrorUnwrap(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, testSecurityKeyChallengeError(), ErrSecurityKeyAuthorisationRequired, "a challenge should wrap ErrSecurityKeyAuthorisationRequired")
}

func TestWithSecurityKeyAuthorisation(t *testing.T) {
	t.Parallel()
	ctx := WithSecurityKeyAuthorisation(t.Context(), testSecurityKeyChallenge, testSecurityKeyCode)
	exp := &securityKeyAuthorisation{AuthorisationData: testSecurityKeyCode, Challenge: testSecurityKeyChallenge}
	assert.Equal(t, exp, ctx.Value(securityKeyAuthorisationKey{}), "WithSecurityKeyAuthorisation should hold the authorisation in the context")
}

func TestAuthorisedParams(t *testing.T) {
	t.Parallel()
	params := &instrumentNameParams{InstrumentName: "BTC-PERPETUAL"}
	got, err := authorisedParams(t.Context(), params)
	require.NoError(t, err, "authorisedParams must not error without an authorisation")
	assert.Same(t, params, got, "authorisedParams should leave params unchanged without an authorisation")

	ctx := WithSecurityKeyAuthorisation(t.Context(), testSecurityKeyChallenge, testSecurityKeyCode)
	authorisation := `"authorization_data":"` + testSecurityKeyCode + `","challenge":"` + testSecurityKeyChallenge + `"`
	for _, tc := range []struct {
		name   string
		params any
		exp    string
	}{
		{name: "no params", exp: `{` + authorisation + `}`},
		{name: "a nil request", params: (*instrumentNameParams)(nil), exp: `{` + authorisation + `}`},
		{name: "a request", params: params, exp: `{"instrument_name":"BTC-PERPETUAL",` + authorisation + `}`},
		{name: "a map", params: map[string]any{"currency": "BTC"}, exp: `{"currency":"BTC",` + authorisation + `}`},
	} {
		got, err := authorisedParams(ctx, tc.params)
		require.NoErrorf(t, err, "authorisedParams must not error for %s", tc.name)
		encoded, err := json.Marshal(got)
		require.NoErrorf(t, err, "Marshal must not error for %s", tc.name)
		assert.JSONEqf(t, tc.exp, string(encoded), "authorisedParams should add the authorisation to %s", tc.name)
	}

	_, err = authorisedParams(ctx, []string{"BTC"})
	assert.ErrorIs(t, err, errSecurityKeyParams, "authorisedParams should reject params that are not an object")
	_, err = authorisedParams(ctx, make(chan int))
	assert.Error(t, err, "authorisedParams should error for params that cannot be marshalled")
}

func TestPrivateResultUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result map[string]any
	decoded := privateResult{result: &result}
	require.NoError(t, json.Unmarshal([]byte(testSecurityKeyChallengeResult), &decoded), "Unmarshal must not error for a challenge")
	assert.Equal(t, testSecurityKeyChallengeError(), decoded.challenge, "Unmarshal should decode a challenge")
	assert.Nil(t, result, "Unmarshal should not decode a challenge as the result")

	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "a result without the field", data: `{"currency":"BTC"}`},
		{name: "a result not requiring authorisation", data: `{"security_key_authorization_required":false,"currency":"BTC"}`},
		{name: "a result nesting the field", data: `{"nested":{"security_key_authorization_required":true},"currency":"BTC"}`},
	} {
		var fields map[string]any
		fieldsResult := privateResult{result: &fields}
		require.NoErrorf(t, json.Unmarshal([]byte(tc.data), &fieldsResult), "Unmarshal must not error for %s", tc.name)
		assert.Nilf(t, fieldsResult.challenge, "Unmarshal should not decode %s as a challenge", tc.name)
		assert.Equalf(t, "BTC", fields["currency"], "Unmarshal should decode %s", tc.name)
	}

	var list []map[string]any
	decoded = privateResult{result: &list}
	require.NoError(t, json.Unmarshal([]byte(`[`+testSecurityKeyChallengeResult+`]`), &decoded), "Unmarshal must not error for a list")
	assert.Nil(t, decoded.challenge, "Unmarshal should not decode a list as a challenge")
	assert.Len(t, list, 1, "Unmarshal should decode a list as the result")

	assert.NoError(t, json.Unmarshal([]byte(`{"currency":"BTC"}`), new(privateResult)), "Unmarshal should discard a result without a destination")
	assert.Error(t, json.Unmarshal([]byte(`"BTC"`), &privateResult{result: &result}), "Unmarshal should error for a result that does not match its destination")
}

func TestSendAuthenticatedHTTPRequestSecurityKeyChallenge(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "Reading the request body should not error") {
			return
		}
		var req websocketVenueRequest
		if !assert.NoError(t, json.Unmarshal(body, &req), "Unmarshal should not error for the request body") {
			return
		}
		result := testSecurityKeyChallengeResult
		if string(req.Params) != `{"currency":"BTC"}` {
			assert.JSONEq(t, `{"currency":"BTC","authorization_data":"`+testSecurityKeyCode+`","challenge":"`+testSecurityKeyChallenge+`"}`, string(req.Params), "SendAuthenticatedHTTPRequest should add the authorisation to the params")
			assertHTTPSignature(t, r.Header.Get("Authorization"), r.URL.Path, body)
			result = `{"currency":"BTC"}`
		}
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":` + result + `}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result map[string]string
	err := ex.SendAuthenticatedHTTPRequest(t.Context(), nonMatchingEPL, "private/list_api_keys", map[string]string{"currency": "BTC"}, &result)
	var challenge *SecurityKeyChallengeError
	require.ErrorAs(t, err, &challenge, "SendAuthenticatedHTTPRequest must return the security key challenge")
	assert.Equal(t, testSecurityKeyChallengeError(), challenge, "SendAuthenticatedHTTPRequest should decode the challenge")
	assert.Nil(t, result, "SendAuthenticatedHTTPRequest should not decode the challenge as the result")

	ctx := WithSecurityKeyAuthorisation(t.Context(), challenge.Challenge, testSecurityKeyCode)
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(ctx, nonMatchingEPL, "private/list_api_keys", map[string]string{"currency": "BTC"}, &result), "SendAuthenticatedHTTPRequest must not error once the challenge is confirmed")
	assert.Equal(t, map[string]string{"currency": "BTC"}, result, "SendAuthenticatedHTTPRequest should decode the confirmed request's result")

	err = ex.SendAuthenticatedHTTPRequest(ctx, nonMatchingEPL, "private/list_api_keys", []string{"BTC"}, &result)
	assert.ErrorIs(t, err, errSecurityKeyParams, "SendAuthenticatedHTTPRequest should reject params that cannot carry the authorisation")
}

func TestSendHTTPRequestIgnoresSecurityKeyAuthorisation(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"jsonrpc":"2.0","method":"public/get_time"}`, string(body), "SendHTTPRequest should not add the authorisation to a public method")
		}
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":1792105300181}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	ctx := WithSecurityKeyAuthorisation(t.Context(), testSecurityKeyChallenge, testSecurityKeyCode)
	var result int64
	require.NoError(t, ex.SendHTTPRequest(ctx, nonMatchingEPL, "public/get_time", nil, &result), "SendHTTPRequest must not error")
	assert.Equal(t, int64(1792105300181), result, "SendHTTPRequest should decode the result")
}

func TestSecurityKeyAuthorisation(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("confirming a security key challenge needs an account with two-factor authentication and its secret")
	}
	testSecurityKeyAuthorisation(t, newTestExchange(t).CreateAPIKey, "CreateAPIKey")
}

// testSecurityKeyAuthorisation checks that create returns Deribit's security key challenge for the challenged API key,
// and creates the key once WithSecurityKeyAuthorisation confirms the challenge
func testSecurityKeyAuthorisation(t *testing.T, create func(context.Context, *CreateAPIKeyRequest) (*APIKey, error), name string) {
	t.Helper()
	req := &CreateAPIKeyRequest{MaxScope: APIKeyScopes{"account:read"}, Name: mockChallengedAPIKeyName}
	_, err := create(t.Context(), req)
	var challenge *SecurityKeyChallengeError
	require.ErrorAsf(t, err, &challenge, "%s must return the security key challenge", name)
	assert.Equalf(t, testSecurityKeyChallengeError(), challenge, "%s should decode the challenge", name)

	ctx := WithSecurityKeyAuthorisation(t.Context(), challenge.Challenge, testSecurityKeyCode)
	result, err := create(ctx, req)
	require.NoErrorf(t, err, "%s must not error once the challenge is confirmed", name)
	assert.Equalf(t, testAPIKey(), result, "%s should decode the confirmed request's result", name)
}
