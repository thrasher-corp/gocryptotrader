package hyperliquid

import (
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	officialSigningAddress = "0x14791697260e4c9a71f18484c9f997b308e59325"
	testVaultAddress       = "0x1719884eb866cb12b2287399b15f7db5e7d775ea"
	testOtherAddress       = "0x2222222222222222222222222222222222222222"
	validClientOrderID     = "0x00000000000000000000000000000001"
	testExchangeEndpoint   = "/exchange"
	testInfoEndpoint       = "/info"
	testUserRoleInfoType   = "userRole"
	testUserRoleResponse   = `{"role":"user"}`
)

func setTestCredentials(ex *Exchange, credentials *accounts.Credentials) {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(credentials)
}

func setCachedTestAuthority(t *testing.T, ex *Exchange, credentials *accounts.Credentials) {
	t.Helper()
	setTestCredentials(ex, credentials)
	accountAddress, _, err := normaliseAddress(credentials.Key)
	require.NoError(t, err, "Normalising the cached test account must not error")
	vaultAddress := ""
	if credentials.SubAccount != "" {
		vaultAddress, _, err = normaliseAddress(credentials.SubAccount)
		require.NoError(t, err, "Normalising the cached test subaccount must not error")
	}
	signerAddress := ""
	if credentials.Secret != "" {
		privateKey, err := parsePrivateKey(credentials.Secret)
		require.NoError(t, err, "Parsing the cached test signer must not error")
		signerAddress = privateKeyAddress(privateKey)
		privateKey.Zero()
	}
	ex.authorityValidationMu.Lock()
	ex.authorityValidationKey = authorityValidationKey{
		accountAddress: accountAddress,
		vaultAddress:   vaultAddress,
		signerAddress:  signerAddress,
		mainnet:        ex.isMainnetEnvironment(),
	}
	ex.authorityValidated = true
	ex.authorityValidationMu.Unlock()
}

func TestValidateClientOrderID(t *testing.T) {
	for _, clientOrderID := range []string{
		validClientOrderID,
		strings.ToUpper(validClientOrderID),
		"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		require.NoError(t, validateClientOrderID(clientOrderID), "Valid client order ID must not error")
	}
	for _, clientOrderID := range []string{"", "0x01", "0000000000000000000000000000000001", "0x0000000000000000000000000000000z"} {
		require.ErrorIs(t, validateClientOrderID(clientOrderID), errClientOrderIDInvalid, "Invalid client order ID must return the expected error")
	}
}

func TestGetWatchAddress(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()

	_, err := ex.getWatchAddress(t.Context())
	require.Error(t, err, "Getting a watch address without credentials must error")

	setTestCredentials(ex, &accounts.Credentials{Key: strings.ToUpper(officialSigningAddress)})
	address, err := ex.getWatchAddress(t.Context())
	require.NoError(t, err, "Getting a valid account watch address must not error")
	assert.Equal(t, officialSigningAddress, address, "Account watch address should be normalised")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, SubAccount: strings.ToUpper(testVaultAddress)})
	address, err = ex.getWatchAddress(t.Context())
	require.NoError(t, err, "Getting a valid subaccount watch address must not error")
	assert.Equal(t, testVaultAddress, address, "Subaccount watch address should be normalised")

	setTestCredentials(ex, &accounts.Credentials{Key: "invalid"})
	_, err = ex.getWatchAddress(t.Context())
	require.ErrorIs(t, err, errInvalidAddress, "Invalid account address must return the expected error")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, SubAccount: "invalid"})
	_, err = ex.getWatchAddress(t.Context())
	require.ErrorIs(t, err, errInvalidAddress, "Invalid subaccount address must return the expected error")
}

func TestGetSigningCredentials(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()

	_, _, err := ex.getSigningCredentials(t.Context())
	require.Error(t, err, "Getting signing credentials without credentials must error")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress})
	_, _, err = ex.getSigningCredentials(t.Context())
	require.ErrorIs(t, err, errPrivateKeyRequired, "Missing private key must return the expected error")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Missing private key must classify as an authentication failure")

	setTestCredentials(ex, &accounts.Credentials{Key: "invalid", Secret: officialSigningTestKey})
	_, _, err = ex.getSigningCredentials(t.Context())
	require.ErrorIs(t, err, errInvalidAddress, "Invalid account address must return the expected error")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	credentials, vault, err := ex.getSigningCredentials(t.Context())
	require.NoError(t, err, "Getting valid main-account signing credentials must not error")
	assert.Equal(t, officialSigningTestKey, credentials.Secret, "Signing secret should be returned")
	assert.Empty(t, vault, "Main-account signing should not set a vault address")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: strings.ToUpper(testVaultAddress)})
	_, vault, err = ex.getSigningCredentials(t.Context())
	require.NoError(t, err, "Getting valid vault signing credentials must not error")
	assert.Equal(t, testVaultAddress, vault, "Vault signing address should be normalised")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: "invalid"})
	_, _, err = ex.getSigningCredentials(t.Context())
	require.ErrorIs(t, err, errInvalidAddress, "Invalid vault address must return the expected error")
}

func TestGetUserSigningCredentials(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, _, err := ex.getUserSigningCredentials(t.Context())
	require.Error(t, err, "Getting user-signing credentials without credentials must error")

	setTestCredentials(ex, &accounts.Credentials{
		Key:        strings.ToUpper(officialSigningAddress),
		Secret:     officialSigningTestKey,
		SubAccount: strings.ToUpper(testVaultAddress),
	})
	credentials, subAccount, err := ex.getUserSigningCredentials(t.Context())
	require.NoError(t, err, "Getting matching master-key credentials must not error")
	assert.Equal(t, officialSigningTestKey, credentials.Secret, "Master private key should be returned")
	assert.Equal(t, testVaultAddress, subAccount, "Configured subaccount should be normalised")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: "invalid"})
	_, _, err = ex.getUserSigningCredentials(t.Context())
	require.ErrorIs(t, err, errInvalidPrivateKey, "Invalid user-signing key must return the expected error")

	setTestCredentials(ex, &accounts.Credentials{
		Key:    officialSigningAddress,
		Secret: "0x1123456789012345678901234567890123456789012345678901234567890123",
	})
	_, _, err = ex.getUserSigningCredentials(t.Context())
	require.ErrorIs(t, err, errUserSignedMasterRequired, "API-wallet signing key must be rejected for user-signed actions")
}

func TestValidateCachedAuthority(t *testing.T) {
	_, err := (*Exchange)(nil).validateCachedAuthority(t.Context(), &accounts.Credentials{}, false)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil exchange authority validation must return the expected error")

	ex := new(Exchange)
	ex.SetDefaults()
	_, err = ex.validateCachedAuthority(t.Context(), nil, false)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil authority credentials must return the expected error")
	_, err = ex.validateCachedAuthority(t.Context(), &accounts.Credentials{Key: "invalid"}, false)
	require.ErrorIs(t, err, errInvalidAddress, "Invalid cached account address must return the expected error")
	_, err = ex.validateCachedAuthority(t.Context(), &accounts.Credentials{Key: officialSigningAddress, SubAccount: "invalid"}, false)
	require.ErrorIs(t, err, errInvalidAddress, "Invalid cached subaccount address must return the expected error")
	_, err = ex.validateCachedAuthority(t.Context(), &accounts.Credentials{Key: officialSigningAddress, Secret: "invalid"}, false)
	require.ErrorIs(t, err, errInvalidPrivateKey, "Invalid cached signer must return the expected error")

	var roleCalls atomic.Int32
	valid := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "Decoding a cached role request should not error") {
			return
		}
		assert.Equal(t, testUserRoleInfoType, payload.Type, "Cached authority validation should request the account role")
		assert.Equal(t, officialSigningAddress, payload.User, "Cached authority validation should request the configured account")
		roleCalls.Add(1)
		_, writeErr := w.Write([]byte(testUserRoleResponse))
		assert.NoError(t, writeErr, "Writing a cached role response should not error")
	}))
	credentials := &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey}
	setTestCredentials(valid, credentials)
	validationKey, err := valid.validateCachedAuthority(t.Context(), credentials, false)
	require.NoError(t, err, "Validating a fresh authority tuple must not error")
	assert.Equal(t, officialSigningAddress, validationKey.accountAddress, "Cached authority account should match")
	assert.Equal(t, officialSigningAddress, validationKey.signerAddress, "Cached authority signer should match")
	assert.Empty(t, validationKey.vaultAddress, "Main-account authority should not cache a vault")
	assert.True(t, validationKey.mainnet, "Default authority should use the mainnet environment")
	assert.Equal(t, int32(1), roleCalls.Load(), "Fresh authority validation should perform one role lookup")

	cachedKey, err := valid.validateCachedAuthority(t.Context(), credentials, false)
	require.NoError(t, err, "Reusing an unchanged authority tuple must not error")
	assert.Equal(t, validationKey, cachedKey, "Cached authority tuple should remain unchanged")
	assert.Equal(t, int32(1), roleCalls.Load(), "Cached authority validation should not repeat the role lookup")

	_, err = valid.validateCachedAuthority(t.Context(), credentials, true)
	require.NoError(t, err, "Forcing authority revalidation must not error")
	assert.Equal(t, int32(2), roleCalls.Load(), "Forced authority validation should repeat the role lookup")
}

func TestSendSignedAction(t *testing.T) {
	action := cancelAction{Type: "cancel", Cancels: []cancelWire{{AssetID: 1, OrderID: 2}}}
	var response exchangeActionResponse

	require.ErrorIs(t, (*Exchange)(nil).sendSignedAction(t.Context(), action, 1, &response), common.ErrNilPointer, "Nil exchange must return the expected error")
	ex := new(Exchange)
	ex.SetDefaults()
	require.ErrorIs(t, ex.sendSignedAction(t.Context(), action, 1, nil), common.ErrNilPointer, "Nil response must return the expected error")
	ex.Requester = nil
	require.ErrorIs(t, ex.sendSignedAction(t.Context(), action, 1, &response), common.ErrNilPointer, "Nil requester must return the expected error")

	ex.SetDefaults()
	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	err := ex.sendSignedAction(t.Context(), action, maximumActionBatchSize+1, &response)
	require.ErrorIs(t, err, errActionBatchTooLarge, "Oversized action batch must return the expected error")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress})
	err = ex.sendSignedAction(t.Context(), action, 1, &response)
	require.ErrorIs(t, err, errPrivateKeyRequired, "Action without a private key must return the expected error")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Action without a private key must classify as an authentication failure")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: "invalid"})
	err = ex.sendSignedAction(t.Context(), action, 1, &response)
	require.ErrorIs(t, err, errInvalidPrivateKey, "Action with an invalid private key must return the expected error")

	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	err = ex.sendSignedAction(t.Context(), make(chan int), 1, &response)
	require.Error(t, err, "Action with an unsupported signing payload must error")

	ex.API.Endpoints = ex.NewEndpoints()
	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	err = ex.sendSignedAction(t.Context(), action, 1, &response)
	require.Error(t, err, "Action without a configured endpoint must error")

	var captured signedActionRequest
	successExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testInfoEndpoint:
			var request infoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding a signed-action authority request should not error") {
				return
			}
			response := testUserRoleResponse
			if request.User == testVaultAddress {
				response = `{"role":"subAccount","data":{"master":"` + officialSigningAddress + `"}}`
			}
			_, writeErr := w.Write([]byte(response))
			assert.NoError(t, writeErr, "Writing a signed-action authority response should not error")
		case testExchangeEndpoint:
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&captured), "Decoding a signed action request should not error") {
				return
			}
			_, writeErr := w.Write([]byte(`{"status":"ok","response":{"type":"cancel","data":{"statuses":["success"]}}}`))
			assert.NoError(t, writeErr, "Writing a signed action response should not error")
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	setTestCredentials(successExchange, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress})
	err = successExchange.sendSignedAction(t.Context(), action, 1, &response)
	require.NoError(t, err, "Sending a successful signed action must not error")
	assert.Equal(t, testVaultAddress, captured.VaultAddress, "Signed action should include the configured vault")
	assert.NotZero(t, captured.Nonce, "Signed action should include a nonce")
	assert.NotEmpty(t, captured.Signature.R, "Signed action should include signature R")
	assert.NotEmpty(t, captured.Signature.S, "Signed action should include signature S")
	assert.Contains(t, []uint8{27, 28}, captured.Signature.V, "Signed action should include a valid recovery ID")

	agentSecret := "0x1123456789012345678901234567890123456789012345678901234567890123"
	agentKey, err := parsePrivateKey(agentSecret)
	require.NoError(t, err, "Parsing unauthorised API-wallet test key must not error")
	agentAddress := privateKeyAddress(agentKey)
	agentKey.Zero()
	var unauthorisedExchangeCalls atomic.Int32
	unauthorisedSigner := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testInfoEndpoint:
			var request infoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding an unauthorised signer role request should not error") {
				return
			}
			response := testUserRoleResponse
			if request.User == agentAddress {
				response = `{"role":"agent","data":{"user":"` + testOtherAddress + `"}}`
			}
			_, writeErr := w.Write([]byte(response))
			assert.NoError(t, writeErr, "Writing an unauthorised signer role response should not error")
		case testExchangeEndpoint:
			unauthorisedExchangeCalls.Add(1)
			http.Error(w, "must not be reached", http.StatusInternalServerError)
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	setTestCredentials(unauthorisedSigner, &accounts.Credentials{Key: officialSigningAddress, Secret: agentSecret})
	err = unauthorisedSigner.sendSignedAction(t.Context(), action, 1, &response)
	require.ErrorIs(t, err, errSignerNotAuthorised, "Unapproved API-wallet action must fail authority validation")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Unapproved API-wallet action must classify as an authentication failure")
	assert.Zero(t, unauthorisedExchangeCalls.Load(), "Unapproved API-wallet action should not reach the exchange endpoint")
	assert.Zero(t, unauthorisedSigner.lastNonce.Load(), "Unapproved API-wallet action should not consume a nonce")

	var unauthorisedVaultExchangeCalls atomic.Int32
	unauthorisedVault := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testInfoEndpoint:
			var request infoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding an unauthorised vault role request should not error") {
				return
			}
			response := testUserRoleResponse
			if request.User == testVaultAddress {
				response = `{"role":"subAccount","data":{"master":"` + testOtherAddress + `"}}`
			}
			_, writeErr := w.Write([]byte(response))
			assert.NoError(t, writeErr, "Writing an unauthorised vault role response should not error")
		case testExchangeEndpoint:
			unauthorisedVaultExchangeCalls.Add(1)
			http.Error(w, "must not be reached", http.StatusInternalServerError)
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	setTestCredentials(unauthorisedVault, &accounts.Credentials{
		Key:        officialSigningAddress,
		Secret:     officialSigningTestKey,
		SubAccount: testVaultAddress,
	})
	err = unauthorisedVault.sendSignedAction(t.Context(), action, 1, &response)
	require.ErrorIs(t, err, errVaultNotAuthorised, "Unowned vault action must fail authority validation")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Unowned vault action must classify as an authentication failure")
	assert.Zero(t, unauthorisedVaultExchangeCalls.Load(), "Unowned vault action should not reach the exchange endpoint")
	assert.Zero(t, unauthorisedVault.lastNonce.Load(), "Unowned vault action should not consume a nonce")

	jsonFailureExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != testInfoEndpoint {
			t.Error("JSON encoding failure must occur before an exchange request")
			return
		}
		_, writeErr := w.Write([]byte(testUserRoleResponse))
		assert.NoError(t, writeErr, "Writing a JSON-failure authority response should not error")
	}))
	setTestCredentials(jsonFailureExchange, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	err = jsonFailureExchange.sendSignedAction(t.Context(), struct {
		Type  string  `json:"type"  msgpack:"type"`
		Value float64 `json:"value" msgpack:"value"`
	}{Type: "test", Value: math.Inf(1)}, 1, &response)
	require.Error(t, err, "JSON-unsupported signed action must error")
	err = jsonFailureExchange.sendSignedAction(t.Context(), make(chan int), 1, &response)
	require.Error(t, err, "Msgpack-unsupported signed action must error after authority validation")

	for _, tc := range []struct {
		name        string
		replacement *accounts.Credentials
		expectedIs  error
	}{
		{name: "credentials removed", expectedIs: request.ErrAuthRequestFailed},
		{
			name:        "credentials changed",
			replacement: &accounts.Credentials{Key: testOtherAddress, Secret: officialSigningTestKey},
			expectedIs:  errCredentialsChanged,
		},
	} {
		t.Run(tc.name+" during authority validation", func(t *testing.T) {
			var (
				mutatingExchange *Exchange
				exchangeCalls    atomic.Int32
			)
			mutatingExchange = newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case testInfoEndpoint:
					setTestCredentials(mutatingExchange, tc.replacement)
					_, writeErr := w.Write([]byte(testUserRoleResponse))
					assert.NoError(t, writeErr, "Writing a mutating authority response should not error")
				case testExchangeEndpoint:
					exchangeCalls.Add(1)
					http.Error(w, "must not be reached", http.StatusInternalServerError)
				default:
					http.Error(w, "unexpected path", http.StatusNotFound)
				}
			}))
			setTestCredentials(mutatingExchange, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
			var result exchangeActionResponse
			err := mutatingExchange.sendSignedAction(t.Context(), action, 1, &result)
			require.ErrorIs(t, err, tc.expectedIs, "Credentials mutated during validation must return the expected error")
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Credentials mutated during validation must classify as an authentication failure")
			assert.Zero(t, exchangeCalls.Load(), "Credentials mutated during validation should not reach the exchange endpoint")
			assert.Zero(t, mutatingExchange.lastNonce.Load(), "Credentials mutated during validation should not consume a nonce")
		})
	}

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(errorExchange, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	err = errorExchange.sendSignedAction(t.Context(), action, 1, &response)
	require.Error(t, err, "HTTP failure must be returned")

	for _, tc := range []struct {
		name     string
		response string
		contains string
	}{
		{name: "message", response: `{"status":"err","response":"bad action"}`, contains: "bad action"},
		{name: "escaped message", response: `{"status":"err","response":"bad \"size\""}`, contains: `bad "size"`},
		{name: "structured response", response: `{"status":"err","response":{"error":"bad action"}}`, contains: `"error":"bad action"`},
		{name: "null response", response: `{"status":"err","response":null}`, contains: "err"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actionExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == testInfoEndpoint {
					_, writeErr := w.Write([]byte(testUserRoleResponse))
					assert.NoError(t, writeErr, "Writing an action-error authority response should not error")
					return
				}
				_, writeErr := w.Write([]byte(tc.response))
				assert.NoError(t, writeErr, "Writing an action error response should not error")
			}))
			setTestCredentials(actionExchange, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
			var result exchangeActionResponse
			err := actionExchange.sendSignedAction(t.Context(), action, 1, &result)
			require.ErrorIs(t, err, errActionResponse, "Failed action must return the expected error")
			assert.ErrorContains(t, err, tc.contains, "Action error should include the server message")
		})
	}
}

func TestSendSignedActionAuthorityCache(t *testing.T) {
	action := cancelAction{Type: "cancel", Cancels: []cancelWire{{AssetID: 1, OrderID: 2}}}
	agentSecret := "0x1123456789012345678901234567890123456789012345678901234567890123"
	agentKey, err := parsePrivateKey(agentSecret)
	require.NoError(t, err, "Parsing API-wallet test key must not error")
	agentAddress := privateKeyAddress(agentKey)
	agentKey.Zero()

	var (
		infoCalls     atomic.Int32
		exchangeCalls atomic.Int32
		failAction    atomic.Bool
	)
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testInfoEndpoint:
			infoCalls.Add(1)
			var request infoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding an authority request should not error") {
				return
			}
			response := testUserRoleResponse
			if request.User == agentAddress {
				response = `{"role":"agent","data":{"user":"` + officialSigningAddress + `"}}`
			}
			_, writeErr := w.Write([]byte(response))
			assert.NoError(t, writeErr, "Writing an authority response should not error")
		case testExchangeEndpoint:
			exchangeCalls.Add(1)
			response := `{"status":"ok","response":{"type":"cancel","data":{"statuses":["success"]}}}`
			if failAction.Swap(false) {
				response = `{"status":"err","response":"signer is no longer authorised"}`
			}
			_, writeErr := w.Write([]byte(response))
			assert.NoError(t, writeErr, "Writing a signed-action response should not error")
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress, Secret: agentSecret})

	var response exchangeActionResponse
	require.NoError(t, ex.sendSignedAction(t.Context(), action, 1, &response), "The first API-wallet action must validate and succeed")
	assert.Equal(t, int32(2), infoCalls.Load(), "The first API-wallet action should validate the account and signer")
	require.NoError(t, ex.sendSignedAction(t.Context(), action, 1, &response), "A cached API-wallet action must succeed")
	assert.Equal(t, int32(2), infoCalls.Load(), "An unchanged authority tuple should not be revalidated")

	require.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Empty), "Explicit credential validation must succeed")
	assert.Equal(t, int32(4), infoCalls.Load(), "Explicit credential validation should force a fresh account and signer check")
	require.NoError(t, ex.sendSignedAction(t.Context(), action, 1, &response), "An action after explicit validation must succeed")
	assert.Equal(t, int32(4), infoCalls.Load(), "A successful explicit validation should refresh the cached authority tuple")

	failAction.Store(true)
	require.ErrorIs(t, ex.sendSignedAction(t.Context(), action, 1, &response), errActionResponse, "An exchange action failure must be returned")
	assert.Equal(t, int32(4), infoCalls.Load(), "The failed action should use the previously validated authority tuple")
	require.NoError(t, ex.sendSignedAction(t.Context(), action, 1, &response), "An action after a server rejection must revalidate and succeed")
	assert.Equal(t, int32(6), infoCalls.Load(), "A server rejection should invalidate the cached authority tuple")
	assert.Equal(t, int32(5), exchangeCalls.Load(), "Every locally authorised action should reach the exchange endpoint")
}

func TestSendSignedActionRejectsAPIWalletAccount(t *testing.T) {
	action := cancelAction{Type: "cancel", Cancels: []cancelWire{{AssetID: 1, OrderID: 2}}}
	agentSecret := "0x1123456789012345678901234567890123456789012345678901234567890123"
	agentKey, err := parsePrivateKey(agentSecret)
	require.NoError(t, err, "Parsing API-wallet test key must not error")
	agentAddress := privateKeyAddress(agentKey)
	agentKey.Zero()

	var exchangeCalls atomic.Int32
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testInfoEndpoint:
			_, writeErr := w.Write([]byte(`{"role":"agent","data":{"user":"` + officialSigningAddress + `"}}`))
			assert.NoError(t, writeErr, "Writing an API-wallet role response should not error")
		case testExchangeEndpoint:
			exchangeCalls.Add(1)
			_, writeErr := w.Write([]byte(`{"status":"ok","response":{"type":"cancel","data":{"statuses":["success"]}}}`))
			assert.NoError(t, writeErr, "Writing a signed-action response should not error")
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	setTestCredentials(ex, &accounts.Credentials{Key: agentAddress, Secret: agentSecret})

	var response exchangeActionResponse
	err = ex.sendSignedAction(t.Context(), action, 1, &response)
	require.ErrorIs(t, err, errConfiguredAccountMissing, "An API wallet configured as the account must fail master-account validation")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "An API wallet configured as the account must classify as an authentication failure")
	assert.Zero(t, exchangeCalls.Load(), "A misconfigured API-wallet account should not reach the exchange endpoint")
	assert.Zero(t, ex.lastNonce.Load(), "A misconfigured API-wallet account should not consume a nonce")
}

func TestSendUserSignedAction(t *testing.T) {
	credentials := &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey}
	fields := []eip712Field{
		{Name: "destination", Type: "string", Value: testOtherAddress},
		{Name: "amount", Type: "string", Value: "1"},
	}
	var response exchangeActionResponse

	_, err := (*Exchange)(nil).sendUserSignedAction(
		t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
	require.ErrorIs(t, err,
		common.ErrNilPointer,
		"Nil exchange must return the expected error")
	ex := new(Exchange)
	ex.SetDefaults()
	_, err = ex.sendUserSignedAction(
		t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, nil)
	require.ErrorIs(t, err,
		common.ErrNilPointer,
		"Nil response must return the expected error")
	_, err = ex.sendUserSignedAction(
		t.Context(), nil, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
	require.ErrorIs(t, err,
		common.ErrNilPointer,
		"Nil credentials must return the expected error")
	ex.Requester = nil
	_, err = ex.sendUserSignedAction(
		t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
	require.ErrorIs(t, err,
		common.ErrNilPointer,
		"Nil requester must return the expected error")

	ex.SetDefaults()
	for _, tc := range []struct {
		name        string
		actionType  string
		primaryType string
		nonceField  string
	}{
		{name: "blank action type", primaryType: "HyperliquidTransaction:UsdSend", nonceField: "time"},
		{name: "blank primary type", actionType: "usdSend", nonceField: "time"},
		{name: "invalid nonce field", actionType: "usdSend", primaryType: "HyperliquidTransaction:UsdSend", nonceField: "id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ex.sendUserSignedAction(
				t.Context(), credentials, tc.actionType, tc.primaryType, tc.nonceField, fields, &response)
			require.ErrorIs(t, err, errUserSignedActionInvalid, "Malformed action metadata must return the expected error")
		})
	}

	ex.API.Endpoints = ex.NewEndpoints()
	_, err = ex.sendUserSignedAction(
		t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
	require.Error(t, err, "User-signed action without a configured endpoint must error")

	ex.SetDefaults()
	setCachedTestAuthority(t, ex, credentials)
	staleCredentials := *credentials
	staleCredentials.ClientID = "changed"
	nonceBeforeStaleCredentials := ex.lastNonce.Load()
	_, err = ex.sendUserSignedAction(
		t.Context(), &staleCredentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
	require.ErrorIs(t, err, errCredentialsChanged, "Stale user-signing credentials must return the expected error")
	assert.Equal(t, nonceBeforeStaleCredentials, ex.lastNonce.Load(), "Stale user-signing credentials should not consume a nonce")

	for _, invalidFields := range [][]eip712Field{
		{{Name: "", Type: "string", Value: "value"}},
		{{Name: "type", Type: "string", Value: "value"}},
		{{Name: "amount", Type: "string", Value: "1"}, {Name: "amount", Type: "string", Value: "2"}},
	} {
		_, err := ex.sendUserSignedAction(
			t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", invalidFields, &response)
		require.ErrorIs(t, err, errUserSignedActionInvalid, "Reserved or duplicate action fields must return the expected error")
	}
	_, err = ex.sendUserSignedAction(
		t.Context(),
		credentials,
		"usdSend",
		"HyperliquidTransaction:UsdSend",
		"time",
		[]eip712Field{{Name: "destination", Type: "address", Value: testOtherAddress}},
		&response)
	require.ErrorIs(t, err, errEIP712Field, "Unsupported EIP-712 field type must return the expected error")
	invalidCredentials := &accounts.Credentials{Key: officialSigningAddress, Secret: "invalid"}
	setTestCredentials(ex, invalidCredentials)
	ex.authorityValidationMu.Lock()
	ex.authorityValidated = false
	ex.authorityValidationMu.Unlock()
	_, err = ex.sendUserSignedAction(
		t.Context(),
		invalidCredentials,
		"usdSend",
		"HyperliquidTransaction:UsdSend",
		"time",
		fields,
		&response)
	require.ErrorIs(t, err, errInvalidPrivateKey, "Invalid user-signing key must return the expected error")

	var captured struct {
		Action struct {
			Type             string `json:"type"`
			SignatureChainID string `json:"signatureChainId"`
			HyperliquidChain string `json:"hyperliquidChain"`
			Destination      string `json:"destination"`
			Amount           string `json:"amount"`
			Time             uint64 `json:"time"`
		} `json:"action"`
		Nonce        uint64      `json:"nonce"`
		Signature    l1Signature `json:"signature"`
		VaultAddress string      `json:"vaultAddress"`
	}
	successExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testExchangeEndpoint, r.URL.Path, "User-signed action should use the exchange endpoint")
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&captured), "Decoding a user-signed action request should not error") {
			return
		}
		_, writeErr := w.Write([]byte(`{"status":"ok","response":null}`))
		assert.NoError(t, writeErr, "Writing a successful user action response should not error")
	}))
	successExchange.Config.UseSandbox = true
	setCachedTestAuthority(t, successExchange, credentials)
	nonce, err := successExchange.sendUserSignedAction(
		t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
	require.NoError(t, err, "Sending a successful user-signed action must not error")
	assert.True(t, successExchange.authorityValidated, "Successful user-signed action should retain cached authority")
	assert.Equal(t, nonce, captured.Nonce, "Returned nonce should match the outer request nonce")
	assert.Equal(t, nonce, captured.Action.Time, "Action time should match the outer request nonce")
	assert.Equal(t, "usdSend", captured.Action.Type, "Action type should be retained")
	assert.Equal(t, userSignedChainIDHex, captured.Action.SignatureChainID, "Signature chain ID should use the SDK value")
	assert.Equal(t, "Testnet", captured.Action.HyperliquidChain, "Sandbox action should be domain-separated to testnet")
	assert.Equal(t, testOtherAddress, captured.Action.Destination, "Destination should be retained")
	assert.Equal(t, "1", captured.Action.Amount, "Amount should be retained")
	assert.Empty(t, captured.VaultAddress, "VaultAddress should be omitted from a user-signed action")
	assert.NotEmpty(t, captured.Signature.R, "User-signed action should include signature R")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setCachedTestAuthority(t, errorExchange, credentials)
	_, err = errorExchange.sendUserSignedAction(
		t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
	require.Error(t, err, "User-signed action HTTP failure must be returned")
	assert.False(t, errorExchange.authorityValidated, "User-signed HTTP failure should invalidate cached authority")

	for _, tc := range []struct {
		name     string
		response string
		contains string
	}{
		{name: "message", response: `{"status":"err","response":"bad action"}`, contains: "bad action"},
		{name: "structured response", response: `{"status":"err","response":{"error":"bad action"}}`, contains: `"error":"bad action"`},
		{name: "null response", response: `{"status":"err","response":null}`, contains: "err"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actionExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, writeErr := w.Write([]byte(tc.response))
				assert.NoError(t, writeErr, "Writing a user action error response should not error")
			}))
			setCachedTestAuthority(t, actionExchange, credentials)
			_, err := actionExchange.sendUserSignedAction(
				t.Context(), credentials, "usdSend", "HyperliquidTransaction:UsdSend", "time", fields, &response)
			require.ErrorIs(t, err, errActionResponse, "Failed user action must return the expected error")
			assert.ErrorContains(t, err, tc.contains, "User action error should include the server message")
			assert.False(t, actionExchange.authorityValidated, "Rejected user action should invalidate cached authority")
		})
	}
}

func TestIsMainnetEnvironment(t *testing.T) {
	mainnet := new(Exchange)
	mainnet.Config = new(config.Exchange)
	sandbox := new(Exchange)
	sandbox.Config = &config.Exchange{UseSandbox: true}
	for _, tc := range []struct {
		name     string
		exchange *Exchange
		expected bool
	}{
		{name: "nil exchange", expected: true},
		{name: "without config", exchange: new(Exchange), expected: true},
		{name: "mainnet", exchange: mainnet, expected: true},
		{name: "sandbox", exchange: sandbox},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.exchange.isMainnetEnvironment(), "Signing environment should match explicit sandbox configuration")
		})
	}
}

func newRoleTestExchange(t *testing.T, credentials *accounts.Credentials, roles map[string]string, vaultDetails string) *Exchange {
	t.Helper()
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding role request should not error") {
			return
		}
		var response string
		switch request.Type {
		case testUserRoleInfoType:
			response = roles[request.User]
		case "vaultDetails":
			response = vaultDetails
		default:
			t.Errorf("Unexpected role validation request type %q", request.Type)
		}
		if response == "" {
			response = `{"role":"missing"}`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing role response should not error")
	}))
	setTestCredentials(ex, credentials)
	return ex
}

func TestValidateCredentials(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	require.Error(t, ex.validateCredentials(t.Context()), "Validating absent credentials must error")

	setTestCredentials(ex, &accounts.Credentials{Key: "invalid"})
	require.ErrorIs(t, ex.validateCredentials(t.Context()), errInvalidAddress, "Invalid account address must return the expected error")

	missing := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, nil, "")
	require.ErrorIs(t, missing.validateCredentials(t.Context()), errConfiguredAccountMissing, "Missing account must return the expected error")

	agentAccount := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, map[string]string{
		officialSigningAddress: `{"role":"agent","data":{"user":"` + testOtherAddress + `"}}`,
	}, "")
	require.ErrorIs(t, agentAccount.validateCredentials(t.Context()), errConfiguredAccountMissing, "Non-user account role must return the expected error")

	userRoles := map[string]string{officialSigningAddress: testUserRoleResponse}
	keyOnly := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, userRoles, "")
	require.NoError(t, keyOnly.validateCredentials(t.Context()), "Valid key-only monitoring credentials must pass")

	masterSigner := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey}, userRoles, "")
	require.NoError(t, masterSigner.validateCredentials(t.Context()), "Matching master signer must pass")

	invalidSecret := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, Secret: "invalid"}, userRoles, "")
	require.ErrorIs(t, invalidSecret.validateCredentials(t.Context()), errInvalidPrivateKey, "Invalid signer key must return the expected error")

	agentSecret := "0x1123456789012345678901234567890123456789012345678901234567890123"
	agentKey, err := parsePrivateKey(agentSecret)
	require.NoError(t, err, "Parsing agent test key must not error")
	agentAddress := privateKeyAddress(agentKey)
	agentKey.Zero()
	agentRoles := map[string]string{
		officialSigningAddress: testUserRoleResponse,
		agentAddress:           `{"role":"agent","data":{"user":"` + officialSigningAddress + `"}}`,
	}
	agentSigner := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, Secret: agentSecret}, agentRoles, "")
	require.NoError(t, agentSigner.validateCredentials(t.Context()), "Approved API-wallet signer must pass")

	unauthorisedRoles := map[string]string{
		officialSigningAddress: testUserRoleResponse,
		agentAddress:           `{"role":"agent","data":{"user":"` + testOtherAddress + `"}}`,
	}
	unauthorisedSigner := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, Secret: agentSecret}, unauthorisedRoles, "")
	require.ErrorIs(t, unauthorisedSigner.validateCredentials(t.Context()), errSignerNotAuthorised, "Unapproved API-wallet signer must return the expected error")

	invalidAgentUserRoles := map[string]string{
		officialSigningAddress: testUserRoleResponse,
		agentAddress:           `{"role":"agent","data":{"user":"invalid"}}`,
	}
	invalidAgentUser := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, Secret: agentSecret}, invalidAgentUserRoles, "")
	require.ErrorIs(t, invalidAgentUser.validateCredentials(t.Context()), errSignerNotAuthorised, "Agent with malformed user must return the expected error")

	invalidVault := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: "invalid"}, userRoles, "")
	require.ErrorIs(t, invalidVault.validateCredentials(t.Context()), errInvalidAddress, "Invalid vault address must return the expected error")

	subaccountRoles := map[string]string{
		officialSigningAddress: testUserRoleResponse,
		testVaultAddress:       `{"role":"subAccount","data":{"master":"` + officialSigningAddress + `"}}`,
	}
	subaccount := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress}, subaccountRoles, "")
	require.NoError(t, subaccount.validateCredentials(t.Context()), "Owned subaccount must pass")

	unownedSubaccountRoles := map[string]string{
		officialSigningAddress: testUserRoleResponse,
		testVaultAddress:       `{"role":"subAccount","data":{"master":"` + testOtherAddress + `"}}`,
	}
	unownedSubaccount := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress}, unownedSubaccountRoles, "")
	require.ErrorIs(t, unownedSubaccount.validateCredentials(t.Context()), errVaultNotAuthorised, "Unowned subaccount must return the expected error")

	invalidSubaccountMasterRoles := map[string]string{
		officialSigningAddress: testUserRoleResponse,
		testVaultAddress:       `{"role":"subAccount","data":{"master":"invalid"}}`,
	}
	invalidSubaccountMaster := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress}, invalidSubaccountMasterRoles, "")
	require.ErrorIs(t, invalidSubaccountMaster.validateCredentials(t.Context()), errVaultNotAuthorised, "Malformed subaccount master must return the expected error")

	vaultRoles := map[string]string{
		officialSigningAddress: testUserRoleResponse,
		testVaultAddress:       `{"role":"vault"}`,
	}
	vault := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress}, vaultRoles,
		`{"vaultAddress":"`+testVaultAddress+`","leader":"`+officialSigningAddress+`"}`)
	require.NoError(t, vault.validateCredentials(t.Context()), "Led vault must pass")

	unownedVault := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress}, vaultRoles,
		`{"vaultAddress":"`+testVaultAddress+`","leader":"`+testOtherAddress+`"}`)
	require.ErrorIs(t, unownedVault.validateCredentials(t.Context()), errVaultNotAuthorised, "Vault led by another account must return the expected error")

	invalidVaultLeader := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress}, vaultRoles,
		`{"vaultAddress":"`+testVaultAddress+`","leader":"invalid"}`)
	require.ErrorIs(t, invalidVaultLeader.validateCredentials(t.Context()), errVaultNotAuthorised, "Vault with malformed leader must return the expected error")

	unknownVaultRole := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress}, map[string]string{
		officialSigningAddress: testUserRoleResponse,
		testVaultAddress:       `{"role":"missing"}`,
	}, "")
	require.ErrorIs(t, unknownVaultRole.validateCredentials(t.Context()), errVaultNotAuthorised, "Unsupported vault role must return the expected error")

	failing := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failing, &accounts.Credentials{Key: officialSigningAddress})
	require.Error(t, failing.validateCredentials(t.Context()), "Role lookup failure must be returned")

	for _, tc := range []struct {
		name        string
		credentials *accounts.Credentials
		failureType string
		failureUser string
	}{
		{
			name:        "vault role lookup",
			credentials: &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress},
			failureType: testUserRoleInfoType,
			failureUser: testVaultAddress,
		},
		{
			name:        "vault details lookup",
			credentials: &accounts.Credentials{Key: officialSigningAddress, SubAccount: testVaultAddress},
			failureType: "vaultDetails",
		},
		{
			name:        "signer role lookup",
			credentials: &accounts.Credentials{Key: officialSigningAddress, Secret: agentSecret},
			failureType: testUserRoleInfoType,
			failureUser: agentAddress,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failedLookup := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload infoRequest
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "Decoding failed-role fixture request should not error") {
					return
				}
				if payload.Type == tc.failureType && (tc.failureUser == "" || payload.User == tc.failureUser) {
					http.Error(w, "unavailable", http.StatusServiceUnavailable)
					return
				}
				switch {
				case payload.Type == testUserRoleInfoType && payload.User == officialSigningAddress:
					_, err := w.Write([]byte(testUserRoleResponse))
					assert.NoError(t, err, "Writing the account role should not error")
				case payload.Type == testUserRoleInfoType && payload.User == testVaultAddress:
					_, err := w.Write([]byte(`{"role":"vault"}`))
					assert.NoError(t, err, "Writing the vault role should not error")
				default:
					t.Errorf("Unexpected failed-role fixture request: type=%q user=%q", payload.Type, payload.User)
				}
			}))
			setTestCredentials(failedLookup, tc.credentials)
			require.Error(t, failedLookup.validateCredentials(t.Context()), "Credential validation must return the nested API failure")
		})
	}
}

func TestValidateAPICredentials(t *testing.T) {
	missing := new(Exchange)
	missing.SetDefaults()
	require.ErrorIs(t, missing.ValidateAPICredentials(t.Context(), asset.Empty), request.ErrAuthRequestFailed, "Missing credentials must classify as an authentication failure")

	ex := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, map[string]string{
		officialSigningAddress: testUserRoleResponse,
	}, "")
	require.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Empty), "Valid credentials with an empty asset filter must pass")
	require.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), "Valid credentials with a supported asset must pass")
	require.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Options), asset.ErrNotSupported, "Unsupported validation asset must return the expected error")

	invalid := newRoleTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, map[string]string{
		officialSigningAddress: `{"role":"missing"}`,
	}, "")
	err := invalid.ValidateAPICredentials(t.Context(), asset.Empty)
	require.ErrorIs(t, err, errConfiguredAccountMissing, "Missing account must retain the credential validation error")
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Invalid credentials must classify as an authentication failure")

	for _, tc := range []struct {
		name        string
		credentials *accounts.Credentials
		expectedIs  error
	}{
		{name: "invalid account address", credentials: &accounts.Credentials{Key: "invalid"}, expectedIs: errInvalidAddress},
		{
			name:        "invalid vault address",
			credentials: &accounts.Credentials{Key: officialSigningAddress, SubAccount: "invalid"},
			expectedIs:  errInvalidAddress,
		},
		{
			name:        "invalid signer key",
			credentials: &accounts.Credentials{Key: officialSigningAddress, Secret: "invalid"},
			expectedIs:  errInvalidPrivateKey,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := newRoleTestExchange(t, tc.credentials, map[string]string{
				officialSigningAddress: testUserRoleResponse,
			}, "")
			err := invalid.ValidateAPICredentials(t.Context(), asset.Empty)
			require.ErrorIs(t, err, tc.expectedIs, "Invalid credential material must return the expected error")
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Invalid credential material must classify as an authentication failure")
		})
	}

	for _, tc := range []struct {
		name        string
		replacement *accounts.Credentials
		expectedIs  error
	}{
		{name: "credentials removed", expectedIs: request.ErrAuthRequestFailed},
		{
			name:        "credentials changed",
			replacement: &accounts.Credentials{Key: testOtherAddress},
			expectedIs:  errCredentialsChanged,
		},
	} {
		t.Run(tc.name+" during explicit validation", func(t *testing.T) {
			var mutatingExchange *Exchange
			mutatingExchange = newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				setTestCredentials(mutatingExchange, tc.replacement)
				_, writeErr := w.Write([]byte(testUserRoleResponse))
				assert.NoError(t, writeErr, "Writing a mutating credential-validation response should not error")
			}))
			setTestCredentials(mutatingExchange, &accounts.Credentials{Key: officialSigningAddress})
			err := mutatingExchange.ValidateAPICredentials(t.Context(), asset.Empty)
			require.ErrorIs(t, err, tc.expectedIs, "Credentials mutated during explicit validation must return the expected error")
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "Credentials mutated during explicit validation must classify as an authentication failure")
		})
	}
}
