package exchange

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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

// CheckCredentials checks to see if the required fields have been set before
// sending an authenticated API request
func (b *Base) CheckCredentials(creds *accounts.Credentials, isContext bool) error {
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
func (b *Base) GetDefaultCredentials() *accounts.Credentials {
	b.API.credMu.RLock()
	defer b.API.credMu.RUnlock()
	if b.API.credentials == (accounts.Credentials{}) {
		return nil
	}
	creds := b.API.credentials
	return &creds
}

// GetCredentials checks and validates current credentials, context credentials
// override default credentials, if no credentials found, will return an error.
func (b *Base) GetCredentials(ctx context.Context) (*accounts.Credentials, error) {
	value := ctx.Value(accounts.ContextCredentialsFlag)
	if value != nil {
		ctxCredStore, ok := value.(*accounts.ContextCredentialsStore)
		if !ok {
			return nil, common.GetTypeAssertError("*accounts.ContextCredentialsStore", value)
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

	if subAccountOverride, ok := ctx.Value(accounts.ContextSubAccountFlag).(string); ok {
		creds.SubAccount = subAccountOverride
	}

	return &creds, nil
}

// VerifyAPICredentials verifies the exchanges API credentials
func (b *Base) VerifyAPICredentials(creds *accounts.Credentials) error {
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

// SetCredentials sets the exchange's default API credentials, copying the
// supplied credentials so later caller mutation has no effect. When the
// exchange requires a base64-decoded secret, the secret is decoded here and
// authenticated support is disabled if decoding fails.
func (b *Base) SetCredentials(creds *accounts.Credentials) {
	b.API.credMu.Lock()
	defer b.API.credMu.Unlock()
	if creds == nil {
		b.API.credentials = accounts.Credentials{}
		return
	}
	b.API.credentials = *creds

	if b.API.CredentialsValidator.RequiresBase64DecodeSecret && !b.API.credentials.SecretBase64Decoded {
		result, err := base64.StdEncoding.DecodeString(b.API.credentials.Secret)
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
