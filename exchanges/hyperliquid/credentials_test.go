package hyperliquid

import (
	"cmp"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const (
	// testAgentPrivateKey signs as an API wallet in authority tests
	testAgentPrivateKey = "0x1123456789012345678901234567890123456789012345678901234567890123"
	// testSubAccountAddress is the vault address of the Python SDK's signing vectors, used as a subaccount or vault
	testSubAccountAddress = "0x1719884eb866cb12b2287399b15f7db5e7d775ea"
	testOtherAddress      = "0x2222222222222222222222222222222222222222"
)

// roleServerUnavailable, as a role or leader given to newRoleServerExchange, fails that address's lookup with HTTP 503
const roleServerUnavailable = "unavailable"

// newRoleServerExchange serves userRole requests from roles and vaultDetails requests from leaders, both keyed by
// address; an exchange endpoint request fails the test unless actions handles it, and every role lookup is counted
func newRoleServerExchange(t *testing.T, roles, leaders map[string]string, actions http.HandlerFunc) (*Exchange, *atomic.Int32) {
	t.Helper()
	var roleLookups atomic.Int32
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/info":
			var body InfoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
				return
			}
			var response string
			switch body.Type {
			case "userRole":
				roleLookups.Add(1)
				response = cmp.Or(roles[body.User], `{"role":"missing"}`)
			case "vaultDetails":
				response = leaders[body.VaultAddress]
				if response != roleServerUnavailable {
					response = `{"vaultAddress":"` + body.VaultAddress + `","leader":"` + response + `"}`
				}
			default:
				http.Error(w, "unexpected info request "+body.Type, http.StatusBadRequest)
				return
			}
			if response == roleServerUnavailable {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
				return
			}
			_, err := w.Write([]byte(response))
			assert.NoError(t, err, "Writing the info response should not error")
		case "/exchange":
			if actions == nil {
				t.Error("The exchange endpoint should not be reached")
				http.Error(w, "unexpected exchange request", http.StatusInternalServerError)
				return
			}
			actions(w, r)
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	})), &roleLookups
}

func testKeyAddress(t *testing.T, secret string) string {
	t.Helper()
	key, err := parsePrivateKey(secret)
	require.NoError(t, err, "parsePrivateKey must not error")
	defer key.Zero()
	return privateKeyAddress(key)
}

// configuredAuthorityKey returns the authority validation key of an exchange's configured credentials
func configuredAuthorityKey(t *testing.T, ex *Exchange) authorityValidationKey {
	t.Helper()
	credentials, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	key := authorityValidationKey{mainnet: ex.isMainnetEnvironment()}
	key.accountAddress, _, err = normaliseAddress(credentials.Key)
	require.NoError(t, err, "normaliseAddress must not error for the configured account")
	if credentials.SubAccount != "" {
		key.vaultAddress, _, err = normaliseAddress(credentials.SubAccount)
		require.NoError(t, err, "normaliseAddress must not error for the configured subaccount")
	}
	if strings.TrimSpace(credentials.Secret) != "" {
		key.signerAddress = testKeyAddress(t, credentials.Secret)
	}
	return key
}

// authorityCached reports whether the authority of an exchange's configured credentials is cached
func authorityCached(t *testing.T, ex *Exchange) bool {
	t.Helper()
	key := configuredAuthorityKey(t, ex)
	ex.authorityValidationMu.Lock()
	defer ex.authorityValidationMu.Unlock()
	_, cached := ex.authorityValidations[key]
	return cached
}

func TestGetWatchAddress(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.getWatchAddress(t.Context())
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "getWatchAddress should error without credentials")

	for _, tc := range []struct {
		credentials *accounts.Credentials
		exp         string
		err         error
	}{
		{credentials: &accounts.Credentials{Key: strings.ToUpper(testAccountAddress)}, exp: testAccountAddress},
		{credentials: &accounts.Credentials{Key: testAccountAddress, SubAccount: strings.ToUpper(testSubAccountAddress)}, exp: testSubAccountAddress},
		{credentials: &accounts.Credentials{Key: "invalid"}, err: errInvalidAddress},
		{credentials: &accounts.Credentials{Key: testAccountAddress, SubAccount: "invalid"}, err: errInvalidAddress},
	} {
		setTestCredentials(ex, tc.credentials)
		address, err := ex.getWatchAddress(t.Context())
		require.ErrorIsf(t, err, tc.err, "getWatchAddress must return the expected error for %+v", tc.credentials)
		assert.Equalf(t, tc.exp, address, "getWatchAddress should return the watched address for %+v", tc.credentials)
	}
}

func TestGetSigningCredentials(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	_, _, err := ex.getSigningCredentials(t.Context())
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "getSigningCredentials should error without credentials")

	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress})
	_, _, err = ex.getSigningCredentials(t.Context())
	assert.ErrorIs(t, err, errPrivateKeyRequired, "getSigningCredentials should require a private key")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "getSigningCredentials should report a missing private key as an authentication failure")

	setTestCredentials(ex, &accounts.Credentials{Key: "invalid", Secret: testPrivateKey})
	_, _, err = ex.getSigningCredentials(t.Context())
	assert.ErrorIs(t, err, errInvalidAddress, "getSigningCredentials should reject an invalid account address")

	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey, SubAccount: "invalid"})
	_, _, err = ex.getSigningCredentials(t.Context())
	assert.ErrorIs(t, err, errInvalidAddress, "getSigningCredentials should reject an invalid subaccount address")

	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
	credentials, vaultAddress, err := ex.getSigningCredentials(t.Context())
	require.NoError(t, err, "getSigningCredentials must not error for an account")
	assert.Equal(t, testPrivateKey, credentials.Secret, "getSigningCredentials should return the private key")
	assert.Empty(t, vaultAddress, "getSigningCredentials should not set a vault for an account")

	setTestCredentials(ex, &accounts.Credentials{Key: strings.ToUpper(testAccountAddress), Secret: testPrivateKey, SubAccount: strings.ToUpper(testSubAccountAddress)})
	credentials, vaultAddress, err = ex.getSigningCredentials(t.Context())
	require.NoError(t, err, "getSigningCredentials must not error for a subaccount")
	assert.Equal(t, testAccountAddress, credentials.Key, "getSigningCredentials should normalise the account address")
	assert.Equal(t, testSubAccountAddress, vaultAddress, "getSigningCredentials should normalise the subaccount address")
	stored, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	assert.Equal(t, strings.ToUpper(testAccountAddress), stored.Key, "getSigningCredentials should leave the stored account address as configured")
}

func TestGetUserSigningCredentials(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	_, _, err := ex.getUserSigningCredentials(t.Context())
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "getUserSigningCredentials should error without credentials")

	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: "invalid"})
	_, _, err = ex.getUserSigningCredentials(t.Context())
	assert.ErrorIs(t, err, errInvalidPrivateKey, "getUserSigningCredentials should reject an invalid private key")

	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: testAgentPrivateKey})
	_, _, err = ex.getUserSigningCredentials(t.Context())
	assert.ErrorIs(t, err, errUserSignedMasterRequired, "getUserSigningCredentials should reject an API wallet key")

	setTestCredentials(ex, &accounts.Credentials{Key: strings.ToUpper(testAccountAddress), Secret: testPrivateKey, SubAccount: strings.ToUpper(testSubAccountAddress)})
	credentials, subAccount, err := ex.getUserSigningCredentials(t.Context())
	require.NoError(t, err, "getUserSigningCredentials must not error for the account's own key")
	assert.Equal(t, testPrivateKey, credentials.Secret, "getUserSigningCredentials should return the private key")
	assert.Equal(t, testSubAccountAddress, subAccount, "getUserSigningCredentials should normalise the subaccount address")
}

func TestValidateCachedAuthority(t *testing.T) {
	t.Parallel()
	ex, roleLookups := newRoleServerExchange(t, map[string]string{testAccountAddress: `{"role":"user"}`}, nil, nil)
	for _, tc := range []struct {
		credentials *accounts.Credentials
		err         error
	}{
		{credentials: &accounts.Credentials{Key: "invalid"}, err: errInvalidAddress},
		{credentials: &accounts.Credentials{Key: testAccountAddress, SubAccount: "invalid"}, err: errInvalidAddress},
		{credentials: &accounts.Credentials{Key: testAccountAddress, Secret: "invalid"}, err: errInvalidPrivateKey},
	} {
		_, err := ex.validateCachedAuthority(t.Context(), tc.credentials, false)
		assert.ErrorIsf(t, err, tc.err, "validateCachedAuthority should reject %+v", tc.credentials)
	}
	assert.Zero(t, roleLookups.Load(), "validateCachedAuthority should not look up roles for invalid credentials")

	credentials := &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey}
	validationKey, err := ex.validateCachedAuthority(t.Context(), credentials, false)
	require.NoError(t, err, "validateCachedAuthority must not error for the account's own key")
	exp := authorityValidationKey{accountAddress: testAccountAddress, signerAddress: testAccountAddress, mainnet: true}
	assert.Equal(t, exp, validationKey, "validateCachedAuthority should key the account, signer and environment")
	assert.Equal(t, int32(1), roleLookups.Load(), "validateCachedAuthority should look up the account role once")

	cachedKey, err := ex.validateCachedAuthority(t.Context(), credentials, false)
	require.NoError(t, err, "validateCachedAuthority must not error for cached authority")
	assert.Equal(t, validationKey, cachedKey, "validateCachedAuthority should return the cached key")
	assert.Equal(t, int32(1), roleLookups.Load(), "validateCachedAuthority should not repeat a cached validation")

	_, err = ex.validateCachedAuthority(t.Context(), credentials, true)
	require.NoError(t, err, "validateCachedAuthority must not error when forced")
	assert.Equal(t, int32(2), roleLookups.Load(), "validateCachedAuthority should revalidate when forced")

	watchOnly := &accounts.Credentials{Key: testAccountAddress}
	_, err = ex.validateCachedAuthority(t.Context(), watchOnly, false)
	require.NoError(t, err, "validateCachedAuthority must not error for a watch-only account")
	assert.Equal(t, int32(3), roleLookups.Load(), "validateCachedAuthority should revalidate a different signer")

	for _, alternate := range []*accounts.Credentials{credentials, watchOnly} {
		_, err = ex.validateCachedAuthority(t.Context(), alternate, false)
		require.NoError(t, err, "validateCachedAuthority must not error for alternating credentials")
		assert.Equal(t, int32(3), roleLookups.Load(), "validateCachedAuthority should keep a validation while other credentials are validated")
	}

	ex.SetDefaults()
	assert.Nil(t, ex.authorityValidations, "SetDefaults should clear the cached validations")

	subAccounts, subAccountLookups := newRoleServerExchange(t, map[string]string{
		testAccountAddress:    `{"role":"user"}`,
		testSubAccountAddress: `{"role":"subAccount","data":{"master":"` + testAccountAddress + `"}}`,
		testOtherAddress:      `{"role":"subAccount","data":{"master":"` + testAccountAddress + `"}}`,
	}, nil, nil)
	for range 2 {
		for _, subAccount := range []string{testSubAccountAddress, testOtherAddress} {
			_, err = subAccounts.validateCachedAuthority(t.Context(), &accounts.Credentials{Key: testAccountAddress, SubAccount: subAccount}, false)
			require.NoErrorf(t, err, "validateCachedAuthority must not error for subaccount %s", subAccount)
		}
	}
	assert.Equal(t, int32(4), subAccountLookups.Load(), "validateCachedAuthority should validate each alternating subaccount once")

	// The role server reads roles on each request, so the account can lose its role between validations
	roles := map[string]string{testAccountAddress: `{"role":"user"}`}
	revoked, revokedLookups := newRoleServerExchange(t, roles, nil, nil)
	_, err = revoked.validateCachedAuthority(t.Context(), credentials, false)
	require.NoError(t, err, "validateCachedAuthority must not error before the account loses its role")
	roles[testAccountAddress] = `{"role":"missing"}`
	_, err = revoked.validateCachedAuthority(t.Context(), credentials, true)
	require.ErrorIs(t, err, errConfiguredAccountMissing, "validateCachedAuthority must return a failed forced revalidation")
	_, err = revoked.validateCachedAuthority(t.Context(), credentials, false)
	assert.ErrorIs(t, err, errConfiguredAccountMissing, "validateCachedAuthority should not keep a validation that a forced revalidation failed")
	assert.Equal(t, int32(3), revokedLookups.Load(), "validateCachedAuthority should look the account up again after a failed forced revalidation")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	configured, err := e.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	validationKey, err = e.validateCachedAuthority(t.Context(), configured, true)
	require.NoError(t, err, "validateCachedAuthority must not error for the configured credentials")
	if mockTests {
		assert.Equal(t, exp, validationKey, "validateCachedAuthority should key the mock account, its own signer and mainnet")
	} else {
		assert.Equal(t, configuredAuthorityKey(t, e), validationKey, "validateCachedAuthority should key the configured account, vault, signer and environment")
	}
}

func TestInvalidateAuthority(t *testing.T) {
	t.Parallel()
	validationKey := authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress, signerAddress: testAccountAddress, mainnet: true}
	otherSubAccountKey := authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testOtherAddress, signerAddress: testAccountAddress, mainnet: true}
	for _, tc := range []struct {
		key          authorityValidationKey
		expValidated bool
	}{
		{key: validationKey},
		{key: authorityValidationKey{accountAddress: testOtherAddress, vaultAddress: testSubAccountAddress, signerAddress: testAccountAddress, mainnet: true}, expValidated: true},
		{key: authorityValidationKey{accountAddress: testAccountAddress, signerAddress: testAccountAddress, mainnet: true}, expValidated: true},
		{key: authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress, mainnet: true}, expValidated: true},
		{key: authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress, signerAddress: testAccountAddress}, expValidated: true},
	} {
		ex := &Exchange{authorityValidations: map[authorityValidationKey]struct{}{validationKey: {}, otherSubAccountKey: {}}}
		ex.invalidateAuthority(&tc.key)
		_, validated := ex.authorityValidations[validationKey]
		assert.Equalf(t, tc.expValidated, validated, "invalidateAuthority should clear only a matching validation, given %+v", tc.key)
		assert.Containsf(t, ex.authorityValidations, otherSubAccountKey, "invalidateAuthority should keep other validations, given %+v", tc.key)
	}
}

func TestValidateAuthority(t *testing.T) {
	t.Parallel()
	agentAddress := testKeyAddress(t, testAgentPrivateKey)
	user := `{"role":"user"}`
	for _, tc := range []struct {
		name    string
		key     authorityValidationKey
		roles   map[string]string
		leaders map[string]string
		err     error
	}{
		{name: "account", key: authorityValidationKey{accountAddress: testAccountAddress}, roles: map[string]string{testAccountAddress: user}},
		{name: "account role lookup fails", key: authorityValidationKey{accountAddress: testAccountAddress}, roles: map[string]string{testAccountAddress: roleServerUnavailable}, err: request.ErrBadStatus},
		{name: "missing account", key: authorityValidationKey{accountAddress: testAccountAddress}, err: errConfiguredAccountMissing},
		{name: "account is an agent", key: authorityValidationKey{accountAddress: testAccountAddress}, roles: map[string]string{testAccountAddress: `{"role":"agent","data":{"user":"` + testOtherAddress + `"}}`}, err: errConfiguredAccountNotUser},
		{name: "account is a subaccount", key: authorityValidationKey{accountAddress: testAccountAddress}, roles: map[string]string{testAccountAddress: `{"role":"subAccount","data":{"master":"` + testOtherAddress + `"}}`}, err: errConfiguredAccountNotUser},
		{name: "account is a vault", key: authorityValidationKey{accountAddress: testAccountAddress}, roles: map[string]string{testAccountAddress: `{"role":"vault"}`}, err: errConfiguredAccountNotUser},
		{
			name:  "owned subaccount",
			key:   authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress},
			roles: map[string]string{testAccountAddress: user, testSubAccountAddress: `{"role":"subAccount","data":{"master":"` + testAccountAddress + `"}}`},
		},
		{
			name:  "foreign subaccount",
			key:   authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress},
			roles: map[string]string{testAccountAddress: user, testSubAccountAddress: `{"role":"subAccount","data":{"master":"` + testOtherAddress + `"}}`},
			err:   errVaultNotAuthorised,
		},
		{
			name:  "vault role lookup fails",
			key:   authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress},
			roles: map[string]string{testAccountAddress: user, testSubAccountAddress: roleServerUnavailable},
			err:   request.ErrBadStatus,
		},
		{
			name:    "led vault",
			key:     authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress},
			roles:   map[string]string{testAccountAddress: user, testSubAccountAddress: `{"role":"vault"}`},
			leaders: map[string]string{testSubAccountAddress: strings.ToUpper(testAccountAddress)},
		},
		{
			name:    "foreign vault",
			key:     authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress},
			roles:   map[string]string{testAccountAddress: user, testSubAccountAddress: `{"role":"vault"}`},
			leaders: map[string]string{testSubAccountAddress: testOtherAddress},
			err:     errVaultNotAuthorised,
		},
		{
			name:    "vault details lookup fails",
			key:     authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress},
			roles:   map[string]string{testAccountAddress: user, testSubAccountAddress: `{"role":"vault"}`},
			leaders: map[string]string{testSubAccountAddress: roleServerUnavailable},
			err:     request.ErrBadStatus,
		},
		{
			name:  "vault address is a user",
			key:   authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress},
			roles: map[string]string{testAccountAddress: user, testSubAccountAddress: user},
			err:   errVaultNotAuthorised,
		},
		{
			name:  "account signs",
			key:   authorityValidationKey{accountAddress: testAccountAddress, signerAddress: testAccountAddress},
			roles: map[string]string{testAccountAddress: user},
		},
		{
			name:  "approved API wallet",
			key:   authorityValidationKey{accountAddress: testAccountAddress, signerAddress: agentAddress},
			roles: map[string]string{testAccountAddress: user, agentAddress: `{"role":"agent","data":{"user":"` + testAccountAddress + `"}}`},
		},
		{
			name:  "another account's API wallet",
			key:   authorityValidationKey{accountAddress: testAccountAddress, signerAddress: agentAddress},
			roles: map[string]string{testAccountAddress: user, agentAddress: `{"role":"agent","data":{"user":"` + testOtherAddress + `"}}`},
			err:   errSignerNotAuthorised,
		},
		{
			name:  "unapproved signer",
			key:   authorityValidationKey{accountAddress: testAccountAddress, signerAddress: agentAddress},
			roles: map[string]string{testAccountAddress: user, agentAddress: user},
			err:   errSignerNotAuthorised,
		},
		{
			name:  "signer role lookup fails",
			key:   authorityValidationKey{accountAddress: testAccountAddress, signerAddress: agentAddress},
			roles: map[string]string{testAccountAddress: user, agentAddress: roleServerUnavailable},
			err:   request.ErrBadStatus,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, _ := newRoleServerExchange(t, tc.roles, tc.leaders, nil)
			assert.ErrorIs(t, ex.validateAuthority(t.Context(), &tc.key), tc.err, "validateAuthority should return the expected error")
		})
	}

	t.Run("configured account", func(t *testing.T) {
		t.Parallel()
		// The mock responses link the test agent and subaccount to the mock account
		key := authorityValidationKey{accountAddress: testAccountAddress, vaultAddress: testSubAccountAddress, signerAddress: testAgentAddress, mainnet: true}
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			key = configuredAuthorityKey(t, e)
		}
		assert.NoError(t, e.validateAuthority(t.Context(), &key), "validateAuthority should accept the configured account, vault and signer")
	})
}

func TestIsMainnetEnvironment(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.True(t, ex.isMainnetEnvironment(), "isMainnetEnvironment should default to mainnet without a config")
	ex.Config = &config.Exchange{}
	assert.True(t, ex.isMainnetEnvironment(), "isMainnetEnvironment should be mainnet when sandbox is off")
	ex.Config.UseSandbox = true
	assert.False(t, ex.isMainnetEnvironment(), "isMainnetEnvironment should be testnet when sandbox is on")
}

func TestNextNonce(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	first := ex.nextNonce()
	assert.Greater(t, ex.nextNonce(), first, "nextNonce should strictly increase")

	ex.lastNonce.Store(1 << 62)
	assert.Equal(t, uint64(1<<62+1), ex.nextNonce(), "nextNonce should step past a nonce ahead of the clock")

	const goroutines = 32
	var mu sync.Mutex
	seen := make(map[uint64]struct{}, goroutines)
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			nonce := ex.nextNonce()
			mu.Lock()
			seen[nonce] = struct{}{}
			mu.Unlock()
		})
	}
	wg.Wait()
	assert.Len(t, seen, goroutines, "nextNonce should return a unique nonce to every concurrent caller")
}
