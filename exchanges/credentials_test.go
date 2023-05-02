package exchange

import (
	"context"
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
)

func TestGetCredentials(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetCredentials(context.Background())
	if !errors.Is(err, ErrCredentialsAreEmpty) {
		t.Fatalf("received: %v but expected: %v", err, ErrCredentialsAreEmpty)
	}

	b.API.CredentialsValidator.RequiresKey = true
	ctx := account.DeployCredentialsToContext(context.Background(), &account.Credentials{Secret: "wow"})
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errRequiresAPIKey) {
		t.Fatalf("received: %v but expected: %v", err, errRequiresAPIKey)
	}

	b.API.CredentialsValidator.RequiresSecret = true
	ctx = account.DeployCredentialsToContext(context.Background(), &account.Credentials{Key: "wow"})
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errRequiresAPISecret) {
		t.Fatalf("received: %v but expected: %v", err, errRequiresAPISecret)
	}

	ctx = context.WithValue(context.Background(), account.ContextCredentialsFlag, "pewpew")
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errContextCredentialsFailure) {
		t.Fatalf("received: %v but expected: %v", err, errContextCredentialsFailure)
	}

	fullCred := &account.Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		SubAccount:      "supersub",
		ClientID:        "superclient",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	ctx = account.DeployCredentialsToContext(context.Background(), fullCred)
	creds, err := b.GetCredentials(ctx)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

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

	ctx = account.DeployCredentialsToContext(context.Background(), lonelyCred)
	b.API.CredentialsValidator.RequiresClientID = true
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errRequiresAPIClientID) {
		t.Fatalf("received: %v but expected: %v", err, errRequiresAPIClientID)
	}

	b.API.SetKey("hello")
	b.API.SetSecret("sir")
	b.API.SetClientID("1337")

	ctx = context.WithValue(context.Background(), account.ContextSubAccountFlag, "superaccount")
	overridedSA, err := b.GetCredentials(ctx)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if overridedSA.Key != "hello" &&
		overridedSA.Secret != "sir" &&
		overridedSA.ClientID != "1337" &&
		overridedSA.SubAccount != "superaccount" {
		t.Fatal("unexpected values")
	}

	notOverrided, err := b.GetCredentials(context.Background())
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

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
	if b.AreCredentialsValid(context.Background()) {
		t.Fatal("should not be valid")
	}
	ctx := account.DeployCredentialsToContext(context.Background(), &account.Credentials{Key: "hello"})
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
		Expected                   error
	}

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
		{RequiresBase64DecodeSecret: true, Secret: "aGVsbG8gd29ybGQ="},
	}

	setupBase := func(tData *tester) *Base {
		b := &Base{}
		b.API.SetKey(tData.Key)
		b.API.SetSecret(tData.Secret)
		b.API.SetClientID(tData.ClientID)
		b.API.SetPEMKey(tData.PEMKey)
		b.API.CredentialsValidator.RequiresKey = tData.RequiresKey
		b.API.CredentialsValidator.RequiresSecret = tData.RequiresSecret
		b.API.CredentialsValidator.RequiresPEM = tData.RequiresPEM
		b.API.CredentialsValidator.RequiresClientID = tData.RequiresClientID
		b.API.CredentialsValidator.RequiresBase64DecodeSecret = tData.RequiresBase64DecodeSecret
		return b
	}

	for x := range testCases {
		testData := &testCases[x]
		x := x
		t.Run("", func(t *testing.T) {
			t.Parallel()
			b := setupBase(testData)
			if err := b.VerifyAPICredentials(b.API.credentials); !errors.Is(err, testData.Expected) {
				t.Errorf("Test %d: expected: %v: got %v", x+1, testData.Expected, err)
			}
		})
	}
}

func TestCheckCredentials(t *testing.T) {
	t.Parallel()

	b := Base{
		SkipAuthCheck: true,
		API:           API{credentials: &account.Credentials{}},
	}

	// Test SkipAuthCheck
	err := b.CheckCredentials(&account.Credentials{}, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	// Test credentials failure
	b.SkipAuthCheck = false
	b.API.CredentialsValidator.RequiresKey = true
	b.API.credentials.OneTimePassword = "wow"
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, errRequiresAPIKey) {
		t.Errorf("received '%v' expected '%v'", err, errRequiresAPIKey)
	}
	b.API.credentials.OneTimePassword = ""

	// Test bot usage with authenticated API support disabled, but with
	// valid credentials
	b.LoadedByConfig = true
	b.API.credentials.Key = "k3y"
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, ErrAuthenticationSupportNotEnabled) {
		t.Errorf("received '%v' expected '%v'", err, ErrAuthenticationSupportNotEnabled)
	}

	// Test enabled authenticated API support and loaded by config
	// but invalid credentials
	b.API.AuthenticatedSupport = true
	b.API.credentials.Key = ""
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, ErrCredentialsAreEmpty) {
		t.Errorf("received '%v' expected '%v'", err, ErrCredentialsAreEmpty)
	}

	// Finally a valid one
	b.API.credentials.Key = "k3y"
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
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
