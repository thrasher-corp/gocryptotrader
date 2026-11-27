package sharedtestvalues

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func TestSetTestCredentials(t *testing.T) {
	t.Parallel()
	held := accounts.Credentials{Key: "old key", Secret: "old secret", ClientID: "old client", PEMKey: "old pem", SubAccount: "old sub account", OneTimePassword: "old password"}
	for _, tc := range []struct {
		name                       string
		requiresBase64DecodeSecret bool
		creds                      *accounts.Credentials
		exp                        *accounts.Credentials
		authenticated              bool
	}{
		{"credentials", false, &accounts.Credentials{Key: "key", Secret: "secret"}, &accounts.Credentials{Key: "key", Secret: "secret"}, true},
		{"credentials without a key", false, &accounts.Credentials{Secret: "secret", SubAccount: "sub account"}, &accounts.Credentials{Secret: "secret", SubAccount: "sub account"}, true},
		{"credentials with spaces around values", false, &accounts.Credentials{Key: " key ", Secret: "secret", SubAccount: " sub account "}, &accounts.Credentials{Key: " key ", Secret: "secret", SubAccount: " sub account "}, true},
		{
			"every credential field", false,
			&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "client", PEMKey: "pem", SubAccount: "sub account", OneTimePassword: "password", SecretBase64Decoded: true},
			&accounts.Credentials{Key: "key", Secret: "secret", ClientID: "client", PEMKey: "pem", SubAccount: "sub account", OneTimePassword: "password", SecretBase64Decoded: true},
			true,
		},
		{"no credentials", false, nil, nil, true},
		{"no credentials for an exchange that decodes its secret", true, nil, nil, true},
		{"empty credentials for an exchange that decodes its secret", true, &accounts.Credentials{}, &accounts.Credentials{SecretBase64Decoded: true}, true},
		{"a secret the exchange decodes", true, &accounts.Credentials{Key: "key", Secret: "c2VjcmV0"}, &accounts.Credentials{Key: "key", Secret: "secret", SecretBase64Decoded: true}, true},
		{"a secret the exchange cannot decode", true, &accounts.Credentials{Key: "key", Secret: "!"}, &accounts.Credentials{Key: "key", Secret: "!"}, false},
	} {
		for _, flags := range []struct {
			name      string
			rest      bool
			websocket bool
		}{
			{"with authentication off", false, false},
			{"with REST authentication on", true, false},
			{"with websocket authentication on", false, true},
			{"with authentication on", true, true},
		} {
			t.Run(tc.name+" "+flags.name, func(t *testing.T) {
				t.Parallel()
				b := new(exchange.Base)
				previous := held
				b.SetCredentials(&previous)
				b.API.AuthenticatedSupport, b.API.AuthenticatedWebsocketSupport = flags.rest, flags.websocket
				b.API.CredentialsValidator.RequiresBase64DecodeSecret = tc.requiresBase64DecodeSecret
				var given *accounts.Credentials
				if tc.creds != nil {
					given = new(*tc.creds)
				}
				SetTestCredentials(b, given)
				assert.Equal(t, tc.authenticated, b.API.AuthenticatedSupport, "SetTestCredentials should leave authenticated support on unless the secret cannot be decoded")
				assert.Equal(t, tc.authenticated, b.API.AuthenticatedWebsocketSupport, "SetTestCredentials should leave authenticated websocket support on unless the secret cannot be decoded")
				assert.Equal(t, tc.exp, b.GetDefaultCredentials(), "SetTestCredentials should replace the credentials held with those given")
				assert.Equal(t, tc.creds, given, "SetTestCredentials should leave the credentials given as they were")
			})
		}
	}
}
