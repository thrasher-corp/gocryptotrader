package account

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestIsEmpty(t *testing.T) {
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
	_, err := ParseCredentialsMetadata(context.Background(), nil)
	if !errors.Is(err, errMetaDataIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMetaDataIsNil)
	}

	_, err = ParseCredentialsMetadata(context.Background(), metadata.MD{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(),
		string(ContextCredentialsFlag), "wow", string(ContextCredentialsFlag), "wow2")
	nortyMD, _ := metadata.FromOutgoingContext(ctx)

	_, err = ParseCredentialsMetadata(context.Background(), nortyMD)
	if !errors.Is(err, errInvalidCredentialMetaDataLength) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidCredentialMetaDataLength)
	}

	ctx = metadata.AppendToOutgoingContext(context.Background(),
		string(ContextCredentialsFlag), "brokenstring")
	nortyMD, _ = metadata.FromOutgoingContext(ctx)

	_, err = ParseCredentialsMetadata(context.Background(), nortyMD)
	if !errors.Is(err, errMissingInfo) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errMissingInfo)
	}

	beforeCreds := Credentials{
		Key:             "superkey",
		Secret:          "supersecret",
		SubAccount:      "supersub",
		ClientID:        "superclient",
		PEMKey:          "superpem",
		OneTimePassword: "superOneTimePasssssss",
	}

	flag, outGoing := beforeCreds.GetMetaData()
	ctx = metadata.AppendToOutgoingContext(context.Background(), flag, outGoing)
	lovelyMD, _ := metadata.FromOutgoingContext(ctx)

	ctx, err = ParseCredentialsMetadata(context.Background(), lovelyMD)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

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
	ctx = metadata.AppendToOutgoingContext(context.Background(), flag, outGoing)
	lovelyMD, _ = metadata.FromOutgoingContext(ctx)

	ctx, err = ParseCredentialsMetadata(context.Background(), lovelyMD)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	sa, ok := ctx.Value(ContextSubAccountFlag).(string)
	if !ok {
		t.Fatal("should have processed")
	}

	if sa != "supersub" {
		t.Fatal("unexpected value")
	}
}
