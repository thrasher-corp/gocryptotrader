package exchange

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestParseCredentialsMetadata(t *testing.T) {
	t.Parallel()
	_, err := ParseCredentialsMetadata(context.Background(), nil)
	if !errors.Is(err, errMetaDataIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMetaDataIsNil)
	}

	_, err = ParseCredentialsMetadata(context.Background(), metadata.MD{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(),
		string(contextCrendentialsFlag), "wow", string(contextCrendentialsFlag), "wow2")
	nortyMD, _ := metadata.FromOutgoingContext(ctx)

	_, err = ParseCredentialsMetadata(context.Background(), nortyMD)
	if !errors.Is(err, errInvalidCredentialMetaData) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidCredentialMetaData)
	}

	ctx = metadata.AppendToOutgoingContext(context.Background(),
		string(contextCrendentialsFlag), "poopy")
	nortyMD, _ = metadata.FromOutgoingContext(ctx)

	_, err = ParseCredentialsMetadata(context.Background(), nortyMD)
	if !errors.Is(err, errMissingInfo) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMissingInfo)
	}

	beforeCreds := Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		Subaccount:      "supersub",
		ClientID:        "superclient",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	flag, outgoingbruh := beforeCreds.GetMetaData()
	ctx = metadata.AppendToOutgoingContext(context.Background(), flag, outgoingbruh)
	lovelyMD, _ := metadata.FromOutgoingContext(ctx)

	ctx, err = ParseCredentialsMetadata(context.Background(), lovelyMD)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	store, ok := ctx.Value(contextCrendentialsFlag).(*contextCredentialsStore)
	if !ok {
		t.Fatal("should have processed")
	}

	afterCreds := store.Get()

	if afterCreds.Key != "superkey" &&
		afterCreds.Secret != "supersecret" &&
		afterCreds.Subaccount != "supersub" &&
		afterCreds.ClientID != "superclient" &&
		afterCreds.PEMKey != "superpem" &&
		afterCreds.OneTimePassword != "superOneTimePasssssss" {
		t.Fatal("unexpected values")
	}
}

func TestGetCredentials(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetCredentials(context.Background())
	if !errors.Is(err, errCredentialsAreEmpty) {
		t.Fatalf("received: %v but expected: %v", err, errCredentialsAreEmpty)
	}

	b.API.CredentialsValidator.RequiresKey = true
	ctx := DeployCredentialsToContext(context.Background(), &Credentials{Secret: "wow"})
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errRequiresAPIKey) {
		t.Fatalf("received: %v but expected: %v", err, errRequiresAPIKey)
	}

	b.API.CredentialsValidator.RequiresSecret = true
	ctx = DeployCredentialsToContext(context.Background(), &Credentials{Key: "wow"})
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errRequiresAPISecret) {
		t.Fatalf("received: %v but expected: %v", err, errRequiresAPISecret)
	}

	ctx = context.WithValue(context.Background(), contextCrendentialsFlag, "pewpew")
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errContextCredentialsFailure) {
		t.Fatalf("received: %v but expected: %v", err, errContextCredentialsFailure)
	}

	fullCred := Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		Subaccount:      "supersub",
		ClientID:        "superclient",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	flag, store := fullCred.getInternal()

	ctx = context.WithValue(context.Background(), flag, store)
	creds, err := b.GetCredentials(ctx)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	if creds.Key != "superkey" &&
		creds.Secret != "supersecret" &&
		creds.Subaccount != "supersub" &&
		creds.ClientID != "superclient" &&
		creds.PEMKey != "superpem" &&
		creds.OneTimePassword != "superOneTimePasssssss" {
		t.Fatal("unexpected values")
	}

	lonelyCred := Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		Subaccount:      "supersub",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	flag, store = lonelyCred.getInternal()

	ctx = context.WithValue(context.Background(), flag, store)
	b.API.CredentialsValidator.RequiresClientID = true
	_, err = b.GetCredentials(ctx)
	if !errors.Is(err, errRequiresAPIClientID) {
		t.Fatalf("received: %v but expected: %v", err, errRequiresAPIClientID)
	}
}

func TestAreCredentialsValid(t *testing.T) {
	t.Parallel()
	var b Base
	if b.AreCredentialsValid(context.Background()) {
		t.Fatal("should not be valid")
	}
	ctx := DeployCredentialsToContext(context.Background(), &Credentials{Key: "hello"})
	if !b.AreCredentialsValid(ctx) {
		t.Fatal("should be valid")
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()

	var b Base
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

	tests := []tester{
		// Empty credentials
		{Expected: errCredentialsAreEmpty},
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

	for x := range tests {
		setupBase := func(b *Base, tData tester) {
			b.API.SetKey(tData.Key)
			b.API.SetSecret(tData.Secret)
			b.API.SetClientID(tData.ClientID)
			b.API.SetPEMKey(tData.PEMKey)
			b.API.CredentialsValidator.RequiresKey = tData.RequiresKey
			b.API.CredentialsValidator.RequiresSecret = tData.RequiresSecret
			b.API.CredentialsValidator.RequiresPEM = tData.RequiresPEM
			b.API.CredentialsValidator.RequiresClientID = tData.RequiresClientID
			b.API.CredentialsValidator.RequiresBase64DecodeSecret = tData.RequiresBase64DecodeSecret
		}

		setupBase(&b, tests[x])
		if err := b.ValidateAPICredentials(b.API.credentials); !errors.Is(err, tests[x].Expected) {
			t.Errorf("Test %d: expected: %v: got %v", x+1, tests[x].Expected, err)
		}
	}
}

func TestCheckCredentials(t *testing.T) {
	t.Parallel()

	b := Base{
		SkipAuthCheck: true,
		API:           API{credentials: &Credentials{}},
	}

	// Test SkipAuthCheck
	err := b.CheckCredentials(&Credentials{}, false)
	if !errors.Is(err, nil) {
		t.Error("skip auth check should allow authenticated requests")
	}

	// Test credentials failure
	b.SkipAuthCheck = false
	b.API.CredentialsValidator.RequiresKey = true
	b.API.credentials.OneTimePassword = "wow"
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, errRequiresAPIKey) {
		t.Error("should fail with an empty key")
	}
	b.API.credentials.OneTimePassword = ""

	// Test bot usage with authenticated API support disabled, but with
	// valid credentials
	b.LoadedByConfig = true
	b.API.credentials.Key = "k3y"
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, errAuthenticationSupportNotEnabled) {
		t.Error("should fail when authenticated support is disabled")
	}

	// Test enabled authenticated API support and loaded by config
	// but invalid credentials
	b.API.AuthenticatedSupport = true
	b.API.credentials.Key = ""
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, errCredentialsAreEmpty) {
		t.Error("should fail with invalid credentials")
	}

	// Finally a valid one
	b.API.credentials.Key = "k3y"
	err = b.CheckCredentials(b.API.credentials, false)
	if !errors.Is(err, nil) {
		t.Error("show allow an authenticated request")
	}
}

func TestGetInternal(t *testing.T) {
	t.Parallel()
	flag, store := (&Credentials{}).getInternal()
	if flag != "" {
		t.Fatal("unexpected value")
	}
	if store != nil {
		t.Fatal("unexpected value")
	}
	flag, store = (&Credentials{Key: "wow"}).getInternal()
	if flag != contextCrendentialsFlag {
		t.Fatal("unexpected value")
	}
	if store == nil {
		t.Fatal("unexpected value")
	}
	if store.Get().Key != "wow" {
		t.Fatal("unexpected value")
	}
}

func TestAPISetters(t *testing.T) {
	t.Parallel()
	api := API{}
	api.SetKey(_Key)
	if api.credentials.Key != _Key {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetSecret(_Secret)
	if api.credentials.Secret != _Secret {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetClientID((_ClientID))
	if api.credentials.ClientID != _ClientID {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetPEMKey(_PEMKey)
	if api.credentials.PEMKey != _PEMKey {
		t.Fatal("unexpected value")
	}

	api = API{}
	api.SetSubaccount(_Subaccount)
	if api.credentials.Subaccount != _Subaccount {
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
