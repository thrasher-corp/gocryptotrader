package exchange

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
)

func TestGetCredentials(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetCredentials(t.Context())
	require.ErrorIs(t, err, ErrCredentialsAreEmpty)

	b.API.CredentialsValidator.RequiresKey = true
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{Secret: "wow"})
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errRequiresAPIKey)

	b.API.CredentialsValidator.RequiresSecret = true
	ctx = account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "wow"})
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errRequiresAPISecret)

	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	ctx = account.DeployCredentialsToContext(t.Context(), &account.Credentials{
		Key:    "meow",
		Secret: "invalidb64",
	})
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errBase64DecodeFailure)

	const expectedBase64DecodedOutput = "hello world"
	ctx = account.DeployCredentialsToContext(t.Context(), &account.Credentials{
		Key:    "meow",
		Secret: "aGVsbG8gd29ybGQ=",
	})
	creds, err := b.GetCredentials(ctx)
	require.NoError(t, err)

	if creds.Secret != expectedBase64DecodedOutput {
		t.Fatalf("received: %v but expected: %v", creds.Secret, expectedBase64DecodedOutput)
	}

	ctx = context.WithValue(t.Context(), account.ContextCredentialsFlag, "pewpew")
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, common.ErrTypeAssertFailure)

	b.API.CredentialsValidator.RequiresBase64DecodeSecret = false
	fullCred := &account.Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		SubAccount:      "supersub",
		ClientID:        "superclient",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	ctx = account.DeployCredentialsToContext(t.Context(), fullCred)
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

	lonelyCred := &account.Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		SubAccount:      "supersub",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	ctx = account.DeployCredentialsToContext(t.Context(), lonelyCred)
	b.API.CredentialsValidator.RequiresClientID = true
	_, err = b.GetCredentials(ctx)
	require.ErrorIs(t, err, errRequiresAPIClientID)

	b.API.SetKey("hello")
	b.API.SetSecret("sir")
	b.API.SetClientID("1337")

	ctx = context.WithValue(t.Context(), account.ContextSubAccountFlag, "superaccount")
	overridedSA, err := b.GetCredentials(ctx)
	require.NoError(t, err)

	if overridedSA.Key != "hello" &&
		overridedSA.Secret != "sir" &&
		overridedSA.ClientID != "1337" &&
		overridedSA.SubAccount != "superaccount" {
		t.Fatal("unexpected values")
	}

	notOverrided, err := b.GetCredentials(t.Context())
	require.NoError(t, err)

	if notOverrided.Key != "hello" &&
		notOverrided.Secret != "sir" &&
		notOverrided.ClientID != "1337" &&
		notOverrided.SubAccount != "" {
		t.Fatal("unexpected values")
	}
}

func TestAreCredentialsValid(t *testing.T) {
	t.Parallel()
	var b Base
	if b.AreCredentialsValid(t.Context()) {
		t.Fatal("should not be valid")
	}
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{Key: "hello"})
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
		UseSetCredentials          bool
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
		{RequiresBase64DecodeSecret: true, Secret: "aGVsbG8gd29ybGQ=", UseSetCredentials: true, CheckBase64DecodedOutput: true},
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
		if tData.UseSetCredentials {
			b.SetCredentials(tData.Key, tData.Secret, tData.ClientID, "", tData.PEMKey, "")
		} else {
			b.API.SetKey(tData.Key)
			b.API.SetSecret(tData.Secret)
			b.API.SetClientID(tData.ClientID)
			b.API.SetPEMKey(tData.PEMKey)
		}
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
					credentials:          account.Credentials{OneTimePassword: "wow"},
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
					credentials:          account.Credentials{Key: "k3y"},
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
					credentials:          account.Credentials{},
				},
			},
			expectedErr: ErrCredentialsAreEmpty,
		},
		{
			name: "Test base64 decoded invalid credentials",
			base: &Base{
				API: API{
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresBase64DecodeSecret: true},
					credentials:          account.Credentials{Secret: "invalid"},
				},
			},
			expectedErr: errBase64DecodeFailure,
		},
		{
			name: "Test base64 decoded valid credentials",
			base: &Base{
				API: API{
					CredentialsValidator: config.APICredentialsValidatorConfig{RequiresBase64DecodeSecret: true},
					credentials:          account.Credentials{Secret: "aGVsbG8gd29ybGQ="},
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
					credentials:          account.Credentials{Key: "k3y"},
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

func TestAPISetters(t *testing.T) {
	t.Parallel()
	api := API{}
	api.SetKey(account.Key)
	if api.credentials.Key != account.Key {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetSecret(account.Secret)
	if api.credentials.Secret != account.Secret {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetClientID(account.ClientID)
	if api.credentials.ClientID != account.ClientID {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetPEMKey(account.PEMKey)
	if api.credentials.PEMKey != account.PEMKey {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetSubAccount(account.SubAccountSTR)
	if api.credentials.SubAccount != account.SubAccountSTR {
		t.Fatal("unexpected value")
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

	b.SetCredentials("RocketMan", "Digereedoo", "007", "", "", "")
	if b.API.credentials.Key != "RocketMan" &&
		b.API.credentials.Secret != "Digereedoo" &&
		b.API.credentials.ClientID != "007" {
		t.Error("invalid API credentials")
	}

	// Invalid secret
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.AuthenticatedSupport = true
	b.SetCredentials("RocketMan", "%%", "007", "", "", "")
	if b.API.AuthenticatedSupport || b.API.AuthenticatedWebsocketSupport {
		t.Error("invalid secret should disable authenticated API support")
	}

	// valid secret
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.AuthenticatedSupport = true
	b.SetCredentials("RocketMan", "aGVsbG8gd29ybGQ=", "007", "", "", "")
	if !b.API.AuthenticatedSupport && b.API.credentials.Secret != "hello world" {
		t.Error("invalid secret should disable authenticated API support")
	}
}

func TestGetDefaultCredentials(t *testing.T) {
	var b Base
	if b.GetDefaultCredentials() != nil {
		t.Fatal("unexpected return")
	}
	b.SetCredentials("test", "", "", "", "", "")
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

// TestGetAuthenticatedAPISupport logic test
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
