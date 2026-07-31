package exchange

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
)

func TestGetCredentials(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetCredentials(t.Context())
	require.ErrorIs(t, err, ErrCredentialsAreEmpty)

	b.API.CredentialsValidator.RequiresKey = true
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Secret: "wow"})
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errRequiresAPIKey)

	b.API.CredentialsValidator.RequiresSecret = true
	ctx = accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "wow"})
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errRequiresAPISecret)

	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	ctx = accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:    "meow",
		Secret: "invalidb64",
	})
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errBase64DecodeFailure)

	const expectedBase64DecodedOutput = "hello world"
	ctx = accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:    "meow",
		Secret: "aGVsbG8gd29ybGQ=",
	})
	creds, err := b.GetCredentials(ctx)
	require.NoError(t, err)

	if creds.Secret != expectedBase64DecodedOutput {
		t.Fatalf("received: %v but expected: %v", creds.Secret, expectedBase64DecodedOutput)
	}

	ctx = context.WithValue(t.Context(), accounts.ContextCredentialsFlag, "pewpew")
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	b.API.CredentialsValidator.RequiresBase64DecodeSecret = false
	fullCred := &accounts.Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		SubAccount:      "supersub",
		ClientID:        "superclient",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	ctx = accounts.DeployCredentialsToContext(t.Context(), fullCred)
	creds, err = b.GetCredentials(ctx)
	require.NoError(t, err)

	if creds.Key != "superkey" &&
		creds.Secret != "supersecret" &&
		creds.SubAccount != "supersub" &&
		creds.ClientID != "superclient" &&
		creds.PEMKey != "superpem" &&
		creds.OneTimePassword != "superOneTimePasssssss" {
		t.Fatal("unexpected values")
	}

	lonelyCred := &accounts.Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		SubAccount:      "supersub",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	ctx = accounts.DeployCredentialsToContext(t.Context(), lonelyCred)
	b.API.CredentialsValidator.RequiresClientID = true
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errRequiresAPIClientID)

	b.SetCredentials(&accounts.Credentials{Key: "hello", Secret: "sir", ClientID: "1337"})

	ctx = context.WithValue(t.Context(), accounts.ContextSubAccountFlag, "superaccount")
	overriddenSA, err := b.GetCredentials(ctx)
	require.NoError(t, err)

	assert.Equal(t, "hello", overriddenSA.Key, "Key should match")
	assert.Equal(t, "sir", overriddenSA.Secret, "Secret should match")
	assert.Equal(t, "1337", overriddenSA.ClientID, "ClientID should match")
	assert.Equal(t, "superaccount", overriddenSA.SubAccount, "SubAccount should match")

	notOverridden, err := b.GetCredentials(t.Context())
	require.NoError(t, err)

	assert.Equal(t, "hello", notOverridden.Key, "Key should match")
	assert.Equal(t, "sir", notOverridden.Secret, "Secret should match")
	assert.Equal(t, "1337", notOverridden.ClientID, "ClientID should match")
	assert.Empty(t, notOverridden.SubAccount, "SubAccount should be empty")
}

func TestAreCredentialsValid(t *testing.T) {
	t.Parallel()
	var b Base
	if b.AreCredentialsValid(t.Context()) {
		t.Fatal("should not be valid")
	}
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "hello"})
	if !b.AreCredentialsValid(ctx) {
		t.Fatal("should be valid")
	}
}

func TestVerifyAPICredentials(t *testing.T) {
	t.Parallel()

	type tester struct {
		Key                        string
		Secret                     string
		ClientID                   string
		PEMKey                     string
		RequiresPEM                bool
		RequiresKey                bool
		RequiresSecret             bool
		RequiresClientID           bool
		RequiresBase64DecodeSecret bool
		CheckBase64DecodedOutput   bool
		Expected                   error
	}

	const expectedBase64DecodedOutput = "hello world"

	testCases := []tester{
		// Empty credentials
		{Expected: ErrCredentialsAreEmpty},
		// test key
		{RequiresKey: true, Expected: errRequiresAPIKey, Secret: "bruh"},
		{RequiresKey: true, Key: "k3y"},
		// test secret
		{RequiresSecret: true, Expected: errRequiresAPISecret, Key: "bruh"},
		{RequiresSecret: true, Secret: "s3cr3t"},
		// test pem
		{RequiresPEM: true, Expected: errRequiresAPIPEMKey, Key: "bruh"},
		{RequiresPEM: true, PEMKey: "p3mK3y"},
		// test clientID
		{RequiresClientID: true, Expected: errRequiresAPIClientID, Key: "bruh"},
		{RequiresClientID: true, ClientID: "cli3nt1D"},
		// test requires base64 decode secret
		{RequiresBase64DecodeSecret: true, RequiresSecret: true, Expected: errRequiresAPISecret, Key: "bruh"},
		{RequiresBase64DecodeSecret: true, Secret: "%%", Expected: errBase64DecodeFailure},
		{RequiresBase64DecodeSecret: true, Secret: "aGVsbG8gd29ybGQ=", CheckBase64DecodedOutput: true},
	}

	setupBase := func(tData *tester) *Base {
		b := &Base{
			API: API{
				CredentialsValidator: config.APICredentialsValidatorConfig{
					RequiresKey:                tData.RequiresKey,
					RequiresSecret:             tData.RequiresSecret,
					RequiresClientID:           tData.RequiresClientID,
					RequiresPEM:                tData.RequiresPEM,
					RequiresBase64DecodeSecret: tData.RequiresBase64DecodeSecret,
				},
			},
		}
		b.SetCredentials(&accounts.Credentials{
			Key:      tData.Key,
			Secret:   tData.Secret,
			ClientID: tData.ClientID,
			PEMKey:   tData.PEMKey,
		})
		return b
	}

	for x, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			b := setupBase(&tc)
			assert.ErrorIs(t, b.VerifyAPICredentials(&b.API.credentials), tc.Expected)

			if tc.CheckBase64DecodedOutput {
				if b.API.credentials.Secret != expectedBase64DecodedOutput {
					t.Errorf("Test %d: expected: %v: got %v", x+1, expectedBase64DecodedOutput, b.API.credentials.Secret)
				}
			}
		})
	}
}

func TestCheckCredentials(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		base              *Base
		checkBase64Output bool
		expectedErr       error
	}{
		{
			name: "Test SkipAuthCheck",
			base: &Base{
				SkipAuthCheck: true,
			},
			expectedErr: nil,
		},
		{
			name: "Test credentials failure",
			base: &Base{
				API: API{
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresKey: true},
					credentials:          accounts.Credentials{OneTimePassword: "wow"},
				},
			},
			expectedErr: errRequiresAPIKey,
		},
		{
			name: "Test exchange usage with authenticated API support disabled, but with valid credentials",
			base: &Base{
				LoadedByConfig: true,
				API: API{
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresKey: true},
					credentials:          accounts.Credentials{Key: "k3y"},
				},
			},
			expectedErr: ErrAuthenticationSupportNotEnabled,
		},
		{
			name: "Test enabled authenticated API support and loaded by config but invalid credentials",
			base: &Base{
				LoadedByConfig: true,
				API: API{
					AuthenticatedSupport: true,
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresKey: true},
					credentials:          accounts.Credentials{},
				},
			},
			expectedErr: ErrCredentialsAreEmpty,
		},
		{
			name: "Test base64 decoded invalid credentials",
			base: &Base{
				API: API{
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresBase64DecodeSecret: true},
					credentials:          accounts.Credentials{Secret: "invalid"},
				},
			},
			expectedErr: errBase64DecodeFailure,
		},
		{
			name: "Test base64 decoded valid credentials",
			base: &Base{
				API: API{
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresBase64DecodeSecret: true},
					credentials:          accounts.Credentials{Secret: "aGVsbG8gd29ybGQ="},
				},
			},
			checkBase64Output: true,
			expectedErr:       nil,
		},
		{
			name: "Test valid credentials",
			base: &Base{
				API: API{
					AuthenticatedSupport: true,
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresKey: true},
					credentials:          accounts.Credentials{Key: "k3y"},
				},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.base.CheckCredentials(&tc.base.API.credentials, false), tc.expectedErr)

			if tc.checkBase64Output {
				if tc.base.API.credentials.SecretBase64Decoded != true {
					t.Errorf("%s: expected secret to be base64 decoded", tc.name)
				}
				if tc.base.API.credentials.Secret != "hello world" {
					t.Errorf("%s: expected %q but received %q", "hello world", tc.name, tc.base.API.credentials.Secret)
				}
			}
		})
	}
}

func TestSetCredentials(t *testing.T) {
	t.Parallel()

	b := Base{
		Name:    "TESTNAME",
		Enabled: false,
		API: API{
			AuthenticatedSupport:          false,
			AuthenticatedWebsocketSupport: false,
		},
	}

	b.SetCredentials(&accounts.Credentials{Key: "RocketMan", Secret: "Digereedoo", ClientID: "007"})
	if b.API.credentials.Key != "RocketMan" &&
		b.API.credentials.Secret != "Digereedoo" &&
		b.API.credentials.ClientID != "007" {
		t.Error("invalid API credentials")
	}

	// Invalid secret
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.AuthenticatedSupport = true
	b.SetCredentials(&accounts.Credentials{Key: "RocketMan", Secret: "%%", ClientID: "007"})
	if b.API.AuthenticatedSupport || b.API.AuthenticatedWebsocketSupport {
		t.Error("invalid secret should disable authenticated API support")
	}

	// valid secret
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.AuthenticatedSupport = true
	b.SetCredentials(&accounts.Credentials{Key: "RocketMan", Secret: "aGVsbG8gd29ybGQ=", ClientID: "007"})
	require.True(t, b.API.AuthenticatedSupport, "authenticated support must remain enabled")
	require.Equal(t, "hello world", b.API.credentials.Secret, "secret must be decoded")

	// Unchanged decoded secret
	creds := b.GetDefaultCredentials()
	b.SetCredentials(creds)
	require.True(t, b.API.AuthenticatedSupport, "authenticated support must remain enabled")
	require.Equal(t, "hello world", b.API.credentials.Secret, "secret must not be decoded again")

	// Rotated secret with stale derived state
	creds.Secret = "Z29vZGJ5ZSB3b3JsZA=="
	b.SetCredentials(creds)
	require.Equal(t, "goodbye world", b.API.credentials.Secret, "rotated secret must be decoded")
	require.True(t, b.API.credentials.SecretBase64Decoded, "rotated secret must be marked as decoded")

	// Invalid rotated secret with stale derived state
	creds = b.GetDefaultCredentials()
	creds.Secret = "%%"
	b.SetCredentials(creds)
	require.False(t, b.API.credentials.SecretBase64Decoded, "invalid secret must not be marked as decoded")
	require.ErrorIs(t, b.VerifyAPICredentials(b.GetDefaultCredentials()), errBase64DecodeFailure, "invalid secret must fail verification")
}

func TestGetDefaultCredentials(t *testing.T) {
	var b Base
	if b.GetDefaultCredentials() != nil {
		t.Fatal("unexpected return")
	}
	b.SetCredentials(&accounts.Credentials{Key: "test"})
	if b.GetDefaultCredentials() == nil {
		t.Fatal("unexpected return")
	}
}

func TestSetAPICredentialDefaults(t *testing.T) {
	t.Parallel()

	b := Base{
		Config: &config.Exchange{},
	}
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.CredentialsValidator.RequiresClientID = true
	b.API.CredentialsValidator.RequiresPEM = true
	b.SetAPICredentialDefaults()

	if !b.Config.API.CredentialsValidator.RequiresKey ||
		!b.Config.API.CredentialsValidator.RequiresSecret ||
		!b.Config.API.CredentialsValidator.RequiresBase64DecodeSecret ||
		!b.Config.API.CredentialsValidator.RequiresClientID ||
		!b.Config.API.CredentialsValidator.RequiresPEM {
		t.Error("incorrect values")
	}
}

func TestGetAuthenticatedAPISupport(t *testing.T) {
	t.Parallel()

	base := Base{
		API: API{
			AuthenticatedSupport:          true,
			AuthenticatedWebsocketSupport: false,
		},
	}

	if !base.IsRESTAuthenticationSupported() {
		t.Fatal("Expected RestAuthentication to return true")
	}
	base.API.AuthenticatedSupport = false
	if base.IsRESTAuthenticationSupported() {
		t.Fatal("Expected RestAuthentication to return false")
	}
	if base.IsWebsocketAuthenticationSupported() {
		t.Fatal("Expected WebsocketAuthentication to return false")
	}
	base.API.AuthenticatedWebsocketSupport = true
	if !base.IsWebsocketAuthenticationSupported() {
		t.Fatal("Expected WebsocketAuthentication to return true")
	}
}
