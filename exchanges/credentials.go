package exchange

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// ErrAuthenticationSupportNotEnabled defines an error when
	// authenticatedSupport and authenticatedWebsocketApiSupport are set to
	// false in config.json
	ErrAuthenticationSupportNotEnabled = errors.New("REST or Websocket authentication support is not enabled")
	// ErrCredentialsAreEmpty defines an error for when the credentials are
	// completely empty but an attempt at retrieving credentials was made to
	// undertake an authenticated HTTP request.
	ErrCredentialsAreEmpty = errors.New("credentials are empty")
	// Errors related to API requirements and failures
	errRequiresAPIKey      = errors.New("requires API key but default/empty one set")
	errRequiresAPISecret   = errors.New("requires API secret but default/empty one set")
	errRequiresAPIPEMKey   = errors.New("requires API PEM key but default/empty one set")
	errRequiresAPIClientID = errors.New("requires API Client ID but default/empty one set")
	errBase64DecodeFailure = errors.New("base64 decode has failed")
)

// SetKey sets new key for the default credentials
func (a *API) SetKey(key string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	a.credentials.Key = key
}

// SetSecret sets new secret for the default credentials
func (a *API) SetSecret(secret string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	a.credentials.Secret = secret
}

// SetClientID sets new clientID for the default credentials
func (a *API) SetClientID(clientID string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	a.credentials.ClientID = clientID
}

// SetPEMKey sets pem key for the default credentials
func (a *API) SetPEMKey(pem string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	a.credentials.PEMKey = pem
}

// SetSubAccount sets sub account for the default credentials
func (a *API) SetSubAccount(sub string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	a.credentials.SubAccount = sub
}

// CheckCredentials checks to see if the required fields have been set before
// sending an authenticated API request
func (b *Base) CheckCredentials(creds *account.Credentials, isContext bool) error {
	if b.SkipAuthCheck {
		return nil
	}

	// Individual package usage, allow request if API credentials are valid a
	// and without needing to set AuthenticatedSupport to true
	if !b.LoadedByConfig {
		return b.VerifyAPICredentials(creds)
	}

	// Bot usage, AuthenticatedSupport can be disabled by user if desired, so
	// don't allow authenticated requests. Context credentials set will override
	// default credentials and supported checks.
	if !b.API.AuthenticatedSupport && !b.API.AuthenticatedWebsocketSupport && !isContext {
		return fmt.Errorf("%s %w", b.Name, ErrAuthenticationSupportNotEnabled)
	}

	// Check to see if the user has enabled AuthenticatedSupport, but has
	// invalid API credentials set and loaded by config
	return b.VerifyAPICredentials(creds)
}

// AreCredentialsValid returns if the supplied credentials are valid.
func (b *Base) AreCredentialsValid(ctx context.Context) bool {
	creds, err := b.GetCredentials(ctx)
	return err == nil && b.VerifyAPICredentials(creds) == nil
}

// GetDefaultCredentials returns the exchange.Base api credentials loaded by
// config.json
func (b *Base) GetDefaultCredentials() *account.Credentials {
	b.API.credMu.RLock()
	defer b.API.credMu.RUnlock()
	if b.API.credentials == (account.Credentials{}) {
		return nil
	}
	creds := b.API.credentials
	return &creds
}

// GetCredentials checks and validates current credentials, context credentials
// override default credentials, if no credentials found, will return an error.
func (b *Base) GetCredentials(ctx context.Context) (*account.Credentials, error) {
	value := ctx.Value(account.ContextCredentialsFlag)
	if value != nil {
		ctxCredStore, ok := value.(*account.ContextCredentialsStore)
		if !ok {
			return nil, common.GetTypeAssertError("*account.ContextCredentialsStore", value)
		}

		creds := ctxCredStore.Get()
		if err := b.CheckCredentials(creds, true); err != nil {
			return nil, fmt.Errorf("error checking credentials from context: %w", err)
		}
		return creds, nil
	}

	// Fallback to exchange loaded credentials
	b.API.credMu.RLock()
	creds := b.API.credentials
	b.API.credMu.RUnlock()
	if err := b.CheckCredentials(&creds, false); err != nil {
		return nil, fmt.Errorf("error checking credentials: %w", err)
	}

	if subAccountOverride, ok := ctx.Value(account.ContextSubAccountFlag).(string); ok {
		creds.SubAccount = subAccountOverride
	}

	return &creds, nil
}

// VerifyAPICredentials verifies the exchanges API credentials
func (b *Base) VerifyAPICredentials(creds *account.Credentials) error {
	b.API.credMu.RLock()
	defer b.API.credMu.RUnlock()
	if creds.IsEmpty() {
		return fmt.Errorf("%s %w", b.Name, ErrCredentialsAreEmpty)
	}
	if b.API.CredentialsValidator.RequiresKey &&
		(creds.Key == "" || creds.Key == config.DefaultAPIKey) {
		return fmt.Errorf("%s %w", b.Name, errRequiresAPIKey)
	}

	if b.API.CredentialsValidator.RequiresSecret &&
		(creds.Secret == "" || creds.Secret == config.DefaultAPISecret) {
		return fmt.Errorf("%s %w", b.Name, errRequiresAPISecret)
	}

	if b.API.CredentialsValidator.RequiresPEM &&
		(creds.PEMKey == "" || strings.Contains(creds.PEMKey, "JUSTADUMMY")) {
		return fmt.Errorf("%s %w", b.Name, errRequiresAPIPEMKey)
	}

	if b.API.CredentialsValidator.RequiresClientID &&
		(creds.ClientID == "" || creds.ClientID == config.DefaultAPIClientID) {
		return fmt.Errorf("%s %w", b.Name, errRequiresAPIClientID)
	}

	if b.API.CredentialsValidator.RequiresBase64DecodeSecret && !creds.SecretBase64Decoded {
		decodedResult, err := base64.StdEncoding.DecodeString(creds.Secret)
		if err != nil {
			return fmt.Errorf("%s API secret %w: %s", b.Name, errBase64DecodeFailure, err)
		}
		creds.Secret = string(decodedResult)
		creds.SecretBase64Decoded = true
	}

	return nil
}

// SetCredentials is a method that sets the current API keys for the exchange
func (b *Base) SetCredentials(apiKey, apiSecret, clientID, subaccount, pemKey, oneTimePassword string) {
	b.API.credMu.Lock()
	defer b.API.credMu.Unlock()
	b.API.credentials.Key = apiKey
	b.API.credentials.ClientID = clientID
	b.API.credentials.SubAccount = subaccount
	b.API.credentials.PEMKey = pemKey
	b.API.credentials.OneTimePassword = oneTimePassword

	if b.API.CredentialsValidator.RequiresBase64DecodeSecret {
		result, err := base64.StdEncoding.DecodeString(apiSecret)
		if err != nil {
			b.API.AuthenticatedSupport = false
			b.API.AuthenticatedWebsocketSupport = false
			log.Warnf(log.ExchangeSys,
				warningBase64DecryptSecretKeyFailed,
				b.Name)
			return
		}
		b.API.credentials.Secret = string(result)
		b.API.credentials.SecretBase64Decoded = true
	} else {
		b.API.credentials.Secret = apiSecret
	}
}

// SetAPICredentialDefaults sets the API Credential validator defaults
func (b *Base) SetAPICredentialDefaults() {
	b.API.credMu.Lock()
	defer b.API.credMu.Unlock()

	// Exchange hardcoded settings take precedence and overwrite the config settings
	if b.Config.API.CredentialsValidator == nil {
		b.Config.API.CredentialsValidator = new(config.APICredentialsValidatorConfig)
	}

	*b.Config.API.CredentialsValidator = b.API.CredentialsValidator
}

// IsWebsocketAuthenticationSupported returns whether the exchange supports
// websocket authenticated API requests
func (b *Base) IsWebsocketAuthenticationSupported() bool {
	return b.API.AuthenticatedWebsocketSupport
}

// IsRESTAuthenticationSupported returns whether the exchange supports REST authenticated
// API requests
func (b *Base) IsRESTAuthenticationSupported() bool {
	return b.API.AuthenticatedSupport
}
