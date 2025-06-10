package account

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestIsEmpty(t *testing.T) {
	t.Parallel()
	var c *Credentials
	if !c.IsEmpty() {
		t.Fatalf("expected: %v but received: %v", true, c.IsEmpty())
	}
	c = new(Credentials)
	if !c.IsEmpty() {
		t.Fatalf("expected: %v but received: %v", true, c.IsEmpty())
	}

	c.SubAccount = "woow"
	if c.IsEmpty() {
		t.Fatalf("expected: %v but received: %v", false, c.IsEmpty())
	}
}

func TestParseCredentialsMetadata(t *testing.T) {
	t.Parallel()
	_, err := ParseCredentialsMetadata(t.Context(), nil)
	require.ErrorIs(t, err, errMetaDataIsNil)

	_, err = ParseCredentialsMetadata(t.Context(), metadata.MD{})
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(t.Context(),
		string(ContextCredentialsFlag), "wow", string(ContextCredentialsFlag), "wow2")
	nortyMD, _ := metadata.FromOutgoingContext(ctx)

	_, err = ParseCredentialsMetadata(t.Context(), nortyMD)
	require.ErrorIs(t, err, errInvalidCredentialMetaDataLength)

	ctx = metadata.AppendToOutgoingContext(t.Context(),
		string(ContextCredentialsFlag), "brokenstring")
	nortyMD, _ = metadata.FromOutgoingContext(ctx)

	_, err = ParseCredentialsMetadata(t.Context(), nortyMD)
	require.ErrorIs(t, err, errMissingInfo)

	beforeCreds := Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		SubAccount:      "supersub",
		ClientID:        "superclient",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	flag, outGoing := beforeCreds.GetMetaData()
	ctx = metadata.AppendToOutgoingContext(t.Context(), flag, outGoing)
	lovelyMD, _ := metadata.FromOutgoingContext(ctx)

	ctx, err = ParseCredentialsMetadata(t.Context(), lovelyMD)
	require.NoError(t, err)

	store, ok := ctx.Value(ContextCredentialsFlag).(*ContextCredentialsStore)
	if !ok {
		t.Fatal("should have processed")
	}

	afterCreds := store.Get()

	if afterCreds.Key != "superkey" &&
		afterCreds.Secret != "supersecret" &&
		afterCreds.SubAccount != "supersub" &&
		afterCreds.ClientID != "superclient" &&
		afterCreds.PEMKey != "superpem" &&
		afterCreds.OneTimePassword != "superOneTimePasssssss" {
		t.Fatal("unexpected values")
	}

	// subaccount override
	subaccount := Credentials{
		SubAccount: "supersub",
	}

	flag, outGoing = subaccount.GetMetaData()
	ctx = metadata.AppendToOutgoingContext(t.Context(), flag, outGoing)
	lovelyMD, _ = metadata.FromOutgoingContext(ctx)

	ctx, err = ParseCredentialsMetadata(t.Context(), lovelyMD)
	require.NoError(t, err)

	sa, ok := ctx.Value(ContextSubAccountFlag).(string)
	if !ok {
		t.Fatal("should have processed")
	}

	if sa != "supersub" {
		t.Fatal("unexpected value")
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
	if flag != ContextCredentialsFlag {
		t.Fatal("unexpected value")
	}
	if store == nil {
		t.Fatal("unexpected value")
	}
	if store.Get().Key != "wow" {
		t.Fatal("unexpected value")
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	creds := Credentials{}
	if s := creds.String(); s != "Key:[...] SubAccount:[] ClientID:[]" {
		t.Fatal("unexpected value")
	}

	creds.Key = "12345678910111234"
	creds.SubAccount = "sub"
	creds.ClientID = "client"

	if s := creds.String(); s != "Key:[1234567891011123...] SubAccount:[sub] ClientID:[client]" {
		t.Fatal("unexpected value")
	}
}

func TestCredentialsEqual(t *testing.T) {
	t.Parallel()
	var this, that *Credentials
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this = &Credentials{}
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that = &Credentials{Key: "1337"}
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.Key = "1337"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.ClientID = "1337"
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that.ClientID = "1337"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.SubAccount = "someSub"
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that.SubAccount = "someSub"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
}

func TestProtectedString(t *testing.T) {
	t.Parallel()
	p := Protected{}
	if s := p.String(); s != "Key:[...] SubAccount:[] ClientID:[]" {
		t.Fatal("unexpected value")
	}

	p.creds.Key = "12345678910111234"
	p.creds.SubAccount = "sub"
	p.creds.ClientID = "client"

	if s := p.creds.String(); s != "Key:[1234567891011123...] SubAccount:[sub] ClientID:[client]" {
		t.Fatal("unexpected value")
	}
}

func TestProtectedCredentialsEqual(t *testing.T) {
	t.Parallel()
	var this Protected
	var that *Credentials
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.creds = Credentials{}
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that = &Credentials{Key: "1337"}
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.creds.Key = "1337"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.creds.ClientID = "1337"
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that.ClientID = "1337"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.creds.SubAccount = "someSub"
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that.SubAccount = "someSub"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
}
