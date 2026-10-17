package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// mockRefreshToken is the refresh token the mock token exchange and fork requests send
	mockRefreshToken = "mock-refresh-token"
	// mockSubjectID is the subaccount the mock token exchange switches to
	mockSubjectID = 10
	// mockSessionName is the session the mock token fork names
	mockSessionName = "forked_session_name"
)

// testAccessToken is the documented token response the mock token requests return
func testAccessToken(scope string) *AccessTokenResponse {
	return &AccessTokenResponse{
		AccessToken:     "1582628593469.1MbQ-J_4.CBP-OqOwm_FBdMYj4cRK2dMXyHPfBtXGpzLxhWg31nHu3H_Q60FpE5_vqUBEQGSiMrIGzw3nC37NMb9d1tpBNqBOM_Ql9pXOmgtV9Yj3Pq1c6BqC6dU6eTxHMFO67x8GpJxqw_QcKP5IepwGBD-gfKSHfAv9AEnLJkNu3JkMJBdLToY1lrBnuedF3dU_uARm",
		TokenType:       "bearer",
		ExpiresIn:       31536000,
		RefreshToken:    "1582628593469.1GP4rQd0.A9Wa78o5kFRIUP49mScaD1CqHgiK50HOl2VA6kCtWa8BQZU5Dr03BhcbXPNvEh3I_MVixKZXnyoBeKJwLl8LXnfo180ckAiPj3zOclcUu4zkXuF3NNP3sTPcDf1B3C1CwMKkJ1NOcf1yPmRbsrd7hbgQ-hLa40tfx6Oa-85ymm_3Z65LZcnCeLrqlj_A9jM",
		Scope:           scope,
		EnabledFeatures: []string{},
	}
}

// testAuthenticateRequest returns a client credentials request for the test credentials, or, live, the configured
// credentials
func testAuthenticateRequest(t *testing.T) *AuthenticateRequest {
	t.Helper()
	if mockTests {
		return &AuthenticateRequest{GrantType: GrantTypeClientCredentials, ClientID: testClientID, ClientSecret: testClientSecret}
	}
	creds, err := e.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	return &AuthenticateRequest{GrantType: GrantTypeClientCredentials, ClientID: creds.Key, ClientSecret: creds.Secret}
}

func TestAuthenticate(t *testing.T) {
	t.Parallel()
	_, err := e.Authenticate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Authenticate must reject a nil request")
	_, err = e.Authenticate(t.Context(), &AuthenticateRequest{GrantType: "password"})
	require.ErrorIs(t, err, errInvalidGrantType, "Authenticate must reject an undocumented grant type")

	skipLiveWithoutCredentials(t)
	result, err := e.Authenticate(t.Context(), testAuthenticateRequest(t))
	require.NoError(t, err, "Authenticate must not error")
	if mockTests {
		assert.Equal(t, testAccessToken("connection mainaccount"), result, "Authenticate should decode the token")
	} else {
		assert.NotEmpty(t, result.RefreshToken, "Authenticate should return a refresh token")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"access_token":""}}`).Authenticate(t.Context(), testAuthenticateRequest(t))
	assert.ErrorIs(t, err, errNoAccessToken, "Authenticate should reject a result without an access token")
	_, err = newUnavailableServerExchange(t).Authenticate(t.Context(), testAuthenticateRequest(t))
	assert.ErrorIs(t, err, request.ErrBadStatus, "Authenticate should return the request failure")
}

func TestAuthenticateRequestValidate(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, tc := range []struct {
		name string
		arg  *AuthenticateRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "undocumented grant type", arg: &AuthenticateRequest{GrantType: "password"}, err: errInvalidGrantType},
		{name: "credentials without client ID", arg: &AuthenticateRequest{GrantType: GrantTypeClientCredentials, ClientSecret: "secret"}, err: errClientIDRequired},
		{name: "credentials without secret", arg: &AuthenticateRequest{GrantType: GrantTypeClientCredentials, ClientID: "id"}, err: errClientSecretRequired},
		{name: "credentials", arg: &AuthenticateRequest{GrantType: GrantTypeClientCredentials, ClientID: "id", ClientSecret: "secret"}},
		{name: "signature without client ID", arg: &AuthenticateRequest{GrantType: GrantTypeClientSignature, Timestamp: now, Signature: "sig"}, err: errClientIDRequired},
		{name: "signature without timestamp", arg: &AuthenticateRequest{GrantType: GrantTypeClientSignature, ClientID: "id", Signature: "sig"}, err: errTimestampRequired},
		{name: "signature without signature", arg: &AuthenticateRequest{GrantType: GrantTypeClientSignature, ClientID: "id", Timestamp: now}, err: errSignatureRequired},
		{name: "signature", arg: &AuthenticateRequest{GrantType: GrantTypeClientSignature, ClientID: "id", Timestamp: now, Signature: "sig"}},
		{name: "refresh without token", arg: &AuthenticateRequest{GrantType: GrantTypeRefreshToken}, err: errRefreshTokenRequired},
		{name: "refresh", arg: &AuthenticateRequest{GrantType: GrantTypeRefreshToken, RefreshToken: "token"}},
	} {
		assert.ErrorIsf(t, tc.arg.validate(), tc.err, "validate should return the expected error for %s", tc.name)
	}
}

func TestNewClientSignatureRequest(t *testing.T) {
	t.Parallel()
	creds := &accounts.Credentials{Key: testClientID, Secret: testClientSecret}
	timestamp := time.UnixMilli(1700000000000)
	// The signatures were computed independently with Python's hmac module from Deribit's documented formula
	result, err := NewClientSignatureRequest(creds, timestamp, "abc123", "")
	require.NoError(t, err, "NewClientSignatureRequest must not error")
	exp := &AuthenticateRequest{
		GrantType: GrantTypeClientSignature,
		ClientID:  testClientID,
		Timestamp: timestamp,
		Signature: "19d6e4709fd2be7a114744057afbbd2d40cee88fe897e0fc678a28b0fd116171",
		Nonce:     "abc123",
	}
	assert.Equal(t, exp, result, "NewClientSignatureRequest should sign the timestamp and nonce")

	result, err = NewClientSignatureRequest(creds, timestamp, "abc123", "data")
	require.NoError(t, err, "NewClientSignatureRequest must not error with data")
	assert.Equal(t, "7be07198f2c7c1e36d8a8b94dab0b4d85f5cf0b141c8f2df9fd1ba53b3c663d7", result.Signature, "NewClientSignatureRequest should sign the data")
	assert.Equal(t, "data", result.Data, "NewClientSignatureRequest should send the data")

	for _, creds := range []*accounts.Credentials{nil, {Secret: testClientSecret}, {Key: testClientID}} {
		_, err = NewClientSignatureRequest(creds, timestamp, "abc123", "")
		assert.ErrorIs(t, err, errSignatureCredentials, "NewClientSignatureRequest should need a client ID and secret")
	}
}

func TestAuthenticateParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(&authenticateParams{
		AuthenticateRequest: &AuthenticateRequest{GrantType: GrantTypeClientSignature, ClientID: "id", Timestamp: time.UnixMilli(1700000000000), Signature: "sig", Nonce: "nonce"},
		Timestamp:           1700000000000,
	})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"grant_type":"client_signature","client_id":"id","timestamp":1700000000000,"signature":"sig","nonce":"nonce"}`, string(encoded), "authenticateParams should send the timestamp in Unix milliseconds once")
}

func TestExchangeToken(t *testing.T) {
	t.Parallel()
	_, err := e.ExchangeToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ExchangeToken must reject a nil request")
	_, err = e.ExchangeToken(t.Context(), &ExchangeTokenRequest{SubjectID: mockSubjectID})
	require.ErrorIs(t, err, errRefreshTokenRequired, "ExchangeToken must reject an empty refresh token")
	_, err = e.ExchangeToken(t.Context(), &ExchangeTokenRequest{RefreshToken: mockRefreshToken})
	require.ErrorIs(t, err, errSubjectIDRequired, "ExchangeToken must reject an empty subject ID")

	if !mockTests {
		t.Skip("ExchangeToken needs a refresh token and a subaccount, which live tests do not have")
	}
	result, err := e.ExchangeToken(t.Context(), &ExchangeTokenRequest{RefreshToken: mockRefreshToken, SubjectID: mockSubjectID})
	require.NoError(t, err, "ExchangeToken must not error")
	exp := testAccessToken("session:named_session mainaccount")
	exp.EnabledFeatures = nil
	assert.Equal(t, exp, result, "ExchangeToken should decode the token")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"access_token":""}}`).ExchangeToken(t.Context(), &ExchangeTokenRequest{RefreshToken: mockRefreshToken, SubjectID: mockSubjectID})
	assert.ErrorIs(t, err, errNoAccessToken, "ExchangeToken should reject a result without an access token")
	_, err = newUnavailableServerExchange(t).ExchangeToken(t.Context(), &ExchangeTokenRequest{RefreshToken: mockRefreshToken, SubjectID: mockSubjectID})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ExchangeToken should return the request failure")
}

func TestForkToken(t *testing.T) {
	t.Parallel()
	_, err := e.ForkToken(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ForkToken must reject a nil request")
	_, err = e.ForkToken(t.Context(), &ForkTokenRequest{SessionName: mockSessionName})
	require.ErrorIs(t, err, errRefreshTokenRequired, "ForkToken must reject an empty refresh token")
	_, err = e.ForkToken(t.Context(), &ForkTokenRequest{RefreshToken: mockRefreshToken})
	require.ErrorIs(t, err, errSessionNameRequired, "ForkToken must reject an empty session name")

	if !mockTests {
		t.Skip("ForkToken needs the refresh token of a session scoped token, which live tests do not have")
	}
	result, err := e.ForkToken(t.Context(), &ForkTokenRequest{RefreshToken: mockRefreshToken, SessionName: mockSessionName})
	require.NoError(t, err, "ForkToken must not error")
	exp := testAccessToken("session:named_session mainaccount")
	exp.EnabledFeatures = nil
	assert.Equal(t, exp, result, "ForkToken should decode the token")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).ForkToken(t.Context(), &ForkTokenRequest{RefreshToken: mockRefreshToken, SessionName: mockSessionName})
	assert.ErrorIs(t, err, errNoAccessToken, "ForkToken should reject a null result")
	_, err = newUnavailableServerExchange(t).ForkToken(t.Context(), &ForkTokenRequest{RefreshToken: mockRefreshToken, SessionName: mockSessionName})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ForkToken should return the request failure")
}
