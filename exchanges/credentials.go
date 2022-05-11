package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"google.golang.org/grpc/metadata"
)

// contextCredential is a string flag for use with context values when setting
// credentials internally or via gRPC.
type contextCredential string

const (
	contextCredentialsFlag contextCredential = "apicredentials"
	contextSubAccountFlag  contextCredential = "subaccountoverride"

	key             = "key"
	secret          = "secret"
	subAccount      = "subaccount"
	clientID        = "clientid"
	oneTimePassword = "otp"
	_PEMKey         = "pemkey"
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

	errRequiresAPIKey                  = errors.New("requires API key but default/empty one set")
	errRequiresAPISecret               = errors.New("requires API secret but default/empty one set")
	errRequiresAPIPEMKey               = errors.New("requires API PEM key but default/empty one set")
	errRequiresAPIClientID             = errors.New("requires API Client ID but default/empty one set")
	errBase64DecodeFailure             = errors.New("base64 decode has failed")
	errMissingInfo                     = errors.New("cannot parse meta data missing information in key value pair")
	errInvalidCredentialMetaDataLength = errors.New("invalid meta data to process credentials")
	errContextCredentialsFailure       = errors.New("context credentials type assertion failure")
	errMetaDataIsNil                   = errors.New("meta data is nil")
)

// ParseCredentialsMetadata intercepts and converts credentials metadata to a
// static type for authentication processing and protection.
func ParseCredentialsMetadata(ctx context.Context, md metadata.MD) (context.Context, error) {
	if md == nil {
		return ctx, errMetaDataIsNil
	}

	credMD, ok := md[string(contextCredentialsFlag)]
	if !ok || len(credMD) == 0 {
		return ctx, nil
	}

	if len(credMD) != 1 {
		return ctx, errInvalidCredentialMetaDataLength
	}

	segregatedCreds := strings.Split(credMD[0], ",")
	var ctxCreds Credentials
	var subAccountHere string
	for x := range segregatedCreds {
		keyVals := strings.Split(segregatedCreds[x], ":")
		if len(keyVals) != 2 {
			return ctx, fmt.Errorf("%w received %v fields, expected 2 contains: %s",
				errMissingInfo,
				len(keyVals),
				keyVals)
		}
		switch keyVals[0] {
		case key:
			ctxCreds.Key = keyVals[1]
		case secret:
			ctxCreds.Secret = keyVals[1]
		case subAccount:
			// Capture sub account as this can override if other values are
			// not included in metadata.
			subAccountHere = keyVals[1]
		case clientID:
			ctxCreds.ClientID = keyVals[1]
		case _PEMKey:
			ctxCreds.PEMKey = keyVals[1]
		case oneTimePassword:
			ctxCreds.OneTimePassword = keyVals[1]
		}
	}
	if ctxCreds.IsEmpty() && subAccountHere != "" {
		// This will override default sub account details if needed.
		return deploySubAccountOverrideToContext(ctx, subAccountHere), nil
	}
	// merge sub account to main context credentials
	ctxCreds.SubAccount = subAccountHere
	return DeployCredentialsToContext(ctx, &ctxCreds), nil
}

// Credentials define parameters that allow for an authenticated request.
type Credentials struct {
	Key             string
	Secret          string
	ClientID        string
	PEMKey          string
	SubAccount      string
	OneTimePassword string
}

// DeployCredentialsToContext sets credentials for internal use to context which
// can override default credential values.
func DeployCredentialsToContext(ctx context.Context, creds *Credentials) context.Context {
	flag, store := creds.getInternal()
	return context.WithValue(ctx, flag, store)
}

// deploySubAccountOverrideToContext sets subaccount as override to credentials
// as a separate flag.
func deploySubAccountOverrideToContext(ctx context.Context, subAccount string) context.Context {
	return context.WithValue(ctx, contextSubAccountFlag, subAccount)
}

// GetMetaData returns the credentials for metadata context deployment
func (c *Credentials) GetMetaData() (flag, values string) {
	vals := make([]string, 0, 6)
	if c.Key != "" {
		vals = append(vals, key+":"+c.Key)
	}
	if c.Secret != "" {
		vals = append(vals, secret+":"+c.Secret)
	}
	if c.SubAccount != "" {
		vals = append(vals, subAccount+":"+c.SubAccount)
	}
	if c.ClientID != "" {
		vals = append(vals, clientID+":"+c.ClientID)
	}
	if c.PEMKey != "" {
		vals = append(vals, _PEMKey+":"+c.PEMKey)
	}
	if c.OneTimePassword != "" {
		vals = append(vals, oneTimePassword+":"+c.OneTimePassword)
	}
	return string(contextCredentialsFlag), strings.Join(vals, ",")
}

// IsEmpty return true if the underlying credentials type has not been filled
// with at least one item.
func (c *Credentials) IsEmpty() bool {
	return c == nil || c.ClientID == "" &&
		c.Key == "" &&
		c.OneTimePassword == "" &&
		c.PEMKey == "" &&
		c.Secret == "" &&
		c.SubAccount == ""
}

// contextCredentialsStore protects the stored credentials for use in a context
type contextCredentialsStore struct {
	creds *Credentials
	mu    sync.RWMutex
}

// getInternal returns the values for assignment to an internal context
func (c *Credentials) getInternal() (contextCredential, *contextCredentialsStore) {
	if c.IsEmpty() {
		return "", nil
	}
	store := &contextCredentialsStore{}
	store.Load(c)
	return contextCredentialsFlag, store
}

// Load stores provided credentials
func (c *contextCredentialsStore) Load(creds *Credentials) {
	// Segregate from external call
	cpy := *creds
	c.mu.Lock()
	c.creds = &cpy
	c.mu.Unlock()
}

// Get returns the full credentials from the store
func (c *contextCredentialsStore) Get() *Credentials {
	c.mu.RLock()
	creds := *c.creds
	c.mu.RUnlock()
	return &creds
}

// SetKey sets new key for the default credentials
func (a *API) SetKey(key string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	if a.credentials == nil {
		a.credentials = &Credentials{}
	}
	a.credentials.Key = key
}

// SetSecret sets new secret for the default credentials
func (a *API) SetSecret(secret string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	if a.credentials == nil {
		a.credentials = &Credentials{}
	}
	a.credentials.Secret = secret
}

// SetClientID sets new clientID for the default credentials
func (a *API) SetClientID(clientID string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	if a.credentials == nil {
		a.credentials = &Credentials{}
	}
	a.credentials.ClientID = clientID
}

// SetPEMKey sets pem key for the default credentials
func (a *API) SetPEMKey(pem string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	if a.credentials == nil {
		a.credentials = &Credentials{}
	}
	a.credentials.PEMKey = pem
}

// SetSubAccount sets sub account for the default credentials
func (a *API) SetSubAccount(sub string) {
	a.credMu.Lock()
	defer a.credMu.Unlock()
	if a.credentials == nil {
		a.credentials = &Credentials{}
	}
	a.credentials.SubAccount = sub
}

// CheckCredentials checks to see if the required fields have been set before
// sending an authenticated API request
func (b *Base) CheckCredentials(creds *Credentials, isContext bool) error {
	if b.SkipAuthCheck {
		return nil
	}

	// Individual package usage, allow request if API credentials are valid a
	// and without needing to set AuthenticatedSupport to true
	if !b.LoadedByConfig {
		return b.ValidateAPICredentials(creds)
	}

	// Bot usage, AuthenticatedSupport can be disabled by user if desired, so
	// don't allow authenticated requests. Context credentials set will override
	// default credentials and supported checks.
	if !b.API.AuthenticatedSupport && !b.API.AuthenticatedWebsocketSupport && !isContext {
		return fmt.Errorf("%s %w", b.Name, ErrAuthenticationSupportNotEnabled)
	}

	// Check to see if the user has enabled AuthenticatedSupport, but has
	// invalid API credentials set and loaded by config
	return b.ValidateAPICredentials(creds)
}

// AreCredentialsValid returns if the supplied credentials are valid.
func (b *Base) AreCredentialsValid(ctx context.Context) bool {
	creds, err := b.GetCredentials(ctx)
	return err == nil && b.ValidateAPICredentials(creds) == nil
}

// GetDefaultCredentials returns the exchange.Base api credentials loaded by
// config.json
func (b *Base) GetDefaultCredentials() *Credentials {
	b.API.credMu.RLock()
	defer b.API.credMu.RUnlock()
	if b.API.credentials == nil {
		return nil
	}
	creds := *b.API.credentials
	return &creds
}

// GetCredentials checks and validates current credentials, context credentials
// override default credentials, if no credentials found, will return an error.
func (b *Base) GetCredentials(ctx context.Context) (*Credentials, error) {
	value := ctx.Value(contextCredentialsFlag)
	if value != nil {
		ctxCredStore, ok := value.(*contextCredentialsStore)
		if !ok {
			return &Credentials{}, errContextCredentialsFailure
		}

		creds := ctxCredStore.Get()
		if err := b.CheckCredentials(creds, true); err != nil {
			return creds, fmt.Errorf("context credentials issue: %w", err)
		}
		return creds, nil
	}

	err := b.CheckCredentials(b.API.credentials, false)
	if err != nil {
		return &Credentials{}, err
	}
	subAccountOverride, ok := ctx.Value(contextSubAccountFlag).(string)
	b.API.credMu.RLock()
	defer b.API.credMu.RUnlock()
	creds := *b.API.credentials
	if ok {
		creds.SubAccount = subAccountOverride
	}
	return &creds, nil
}

// ValidateAPICredentials validates the exchanges API credentials
func (b *Base) ValidateAPICredentials(creds *Credentials) error {
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

	if b.API.CredentialsValidator.RequiresBase64DecodeSecret && !b.LoadedByConfig {
		_, err := crypto.Base64Decode(creds.Secret)
		if err != nil {
			return fmt.Errorf("%s API secret %w: %s", b.Name, errBase64DecodeFailure, err)
		}
	}
	return nil
}

// SetCredentials is a method that sets the current API keys for the exchange
func (b *Base) SetCredentials(apiKey, apiSecret, clientID, subaccount, pemKey, oneTimePassword string) {
	b.API.credMu.Lock()
	defer b.API.credMu.Unlock()
	if b.API.credentials == nil {
		b.API.credentials = &Credentials{}
	}
	b.API.credentials.Key = apiKey
	b.API.credentials.ClientID = clientID
	b.API.credentials.SubAccount = subaccount
	b.API.credentials.PEMKey = pemKey
	b.API.credentials.OneTimePassword = oneTimePassword

	if b.API.CredentialsValidator.RequiresBase64DecodeSecret {
		result, err := crypto.Base64Decode(apiSecret)
		if err != nil {
			b.API.AuthenticatedSupport = false
			b.API.AuthenticatedWebsocketSupport = false
			log.Warnf(log.ExchangeSys,
				warningBase64DecryptSecretKeyFailed,
				b.Name)
			return
		}
		b.API.credentials.Secret = string(result)
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
	if b.Config.API.CredentialsValidator.RequiresKey != b.API.CredentialsValidator.RequiresKey {
		b.Config.API.CredentialsValidator.RequiresKey = b.API.CredentialsValidator.RequiresKey
	}

	if b.Config.API.CredentialsValidator.RequiresSecret != b.API.CredentialsValidator.RequiresSecret {
		b.Config.API.CredentialsValidator.RequiresSecret = b.API.CredentialsValidator.RequiresSecret
	}

	if b.Config.API.CredentialsValidator.RequiresBase64DecodeSecret != b.API.CredentialsValidator.RequiresBase64DecodeSecret {
		b.Config.API.CredentialsValidator.RequiresBase64DecodeSecret = b.API.CredentialsValidator.RequiresBase64DecodeSecret
	}

	if b.Config.API.CredentialsValidator.RequiresClientID != b.API.CredentialsValidator.RequiresClientID {
		b.Config.API.CredentialsValidator.RequiresClientID = b.API.CredentialsValidator.RequiresClientID
	}

	if b.Config.API.CredentialsValidator.RequiresPEM != b.API.CredentialsValidator.RequiresPEM {
		b.Config.API.CredentialsValidator.RequiresPEM = b.API.CredentialsValidator.RequiresPEM
	}
}

// GetAuthenticatedAPISupport returns whether the exchange supports
// authenticated API requests
func (b *Base) GetAuthenticatedAPISupport(endpoint uint8) bool {
	switch endpoint {
	case RestAuthentication:
		return b.API.AuthenticatedSupport
	case WebsocketAuthentication:
		return b.API.AuthenticatedWebsocketSupport
	}
	return false
}
