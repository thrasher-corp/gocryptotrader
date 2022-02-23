package exchange

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"google.golang.org/grpc/metadata"
)

// contextCredential is a string flag for use with context values when setting
// credentials internally or via gRPC.
type contextCredential string

const (
	contextCrendentialsFlag contextCredential = "apicredentials"

	_Key             = "key"
	_Secret          = "secret"
	_Subaccount      = "subaccount"
	_ClientID        = "clientid"
	_PEMKey          = "pemkey"
	_OneTimePassword = "otp"
)

var (
	errRequiresAPIKey                  = errors.New("requires API key but default/empty one set")
	errRequiresAPISecret               = errors.New("requires API secret but default/empty one set")
	errRequiresAPIPEMKey               = errors.New("requires API PEM key but default/empty one set")
	errRequiresAPIClientID             = errors.New("requires API Client ID but default/empty one set")
	errBase64DecodeFailure             = errors.New("base64 decode has failed")
	errAuthenticationSupportNotEnabled = errors.New("REST or Websocket authentication support is not enabled")
	errMissingInfo                     = errors.New("cannot parse meta data missing information in key value pair")
	errInvalidCredentialMetaData       = errors.New("invalid meta data to process credentials")
	errContextCredentialsFailure       = errors.New("context credentials type assertion failure")
	errMetaDataIsNil                   = errors.New("meta data is nil")
	errCredentialsAreEmpty             = errors.New("credentials are empty")
)

// ParseCredentialsMetadata intercepts and converts credentials metadata to a
// static type for authentication processing and protection.
func ParseCredentialsMetadata(ctx context.Context, md metadata.MD) (context.Context, error) {
	if md == nil {
		return ctx, errMetaDataIsNil
	}

	credMD, ok := md[string(contextCrendentialsFlag)]
	if !ok {
		return ctx, nil
	}

	if len(credMD) != 1 {
		return ctx, errInvalidCredentialMetaData
	}

	segregatedCreds := strings.Split(credMD[0], ",")
	var ctxCreds Credentials
	for x := range segregatedCreds {
		keyvals := strings.Split(segregatedCreds[x], ":")
		if len(keyvals) != 2 {
			return ctx, errMissingInfo
		}
		switch keyvals[0] {
		case _Key:
			ctxCreds.Key = keyvals[1]
		case _Secret:
			ctxCreds.Secret = keyvals[1]
		case _Subaccount:
			ctxCreds.Subaccount = keyvals[1]
		case _ClientID:
			ctxCreds.ClientID = keyvals[1]
		case _PEMKey:
			ctxCreds.PEMKey = keyvals[1]
		case _OneTimePassword:
			ctxCreds.OneTimePassword = keyvals[1]
		}
	}
	return DeployCredentialsToContext(ctx, ctxCreds), nil
}

// DeployCredentialsToContext sets credentials for internal use to context which
// can override default credential values.
func DeployCredentialsToContext(ctx context.Context, creds Credentials) context.Context {
	flag, store := creds.getInternal()
	return context.WithValue(ctx, flag, store)
}

// Credentials define parameters that allow for an authenticated request.
type Credentials struct {
	Key             string
	Secret          string
	ClientID        string
	PEMKey          string
	Subaccount      string
	OneTimePassword string
}

// GetMetaData returns the credentials for metadata context deployment
func (c Credentials) GetMetaData() (flag, values string) {
	vals := make([]string, 0, 6)
	if c.Key != "" {
		vals = append(vals, _Key+":"+c.Key)
	}
	if c.Secret != "" {
		vals = append(vals, _Secret+":"+c.Secret)
	}
	if c.Subaccount != "" {
		vals = append(vals, _Subaccount+":"+c.Subaccount)
	}
	if c.ClientID != "" {
		vals = append(vals, _ClientID+":"+c.ClientID)
	}
	if c.PEMKey != "" {
		vals = append(vals, _PEMKey+":"+c.PEMKey)
	}
	if c.OneTimePassword != "" {
		vals = append(vals, _OneTimePassword+":"+c.OneTimePassword)
	}
	return string(contextCrendentialsFlag), strings.Join(vals, ",")
}

// IsEmpty return true if the underlying credentials type has not been filled
// with atleast one item.
func (c Credentials) IsEmpty() bool {
	return c.ClientID == "" && c.Key == "" && c.OneTimePassword == "" &&
		c.PEMKey == "" && c.Secret == "" && c.Subaccount == ""
}

// getInternal returns the values for assignment to an internal context
func (c Credentials) getInternal() (contextCredential, *contextCredentialsStore) {
	if c.IsEmpty() {
		return "", nil
	}
	store := &contextCredentialsStore{}
	store.Load(c)
	return contextCrendentialsFlag, store
}

// CheckCredentials checks to see if the required fields have been set before
// sending an authenticated API request
func (b *Base) CheckCredentials(creds Credentials, isContext bool) error {
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
		return fmt.Errorf("%s %w", b.Name, errAuthenticationSupportNotEnabled)
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

// GetCredentials checks and validates current credentials, context credentials
// overide default credentials, if no credentials found, will return an error.
func (b *Base) GetCredentials(ctx context.Context) (Credentials, error) {
	value := ctx.Value(contextCrendentialsFlag)
	if value != nil {
		ctxCredStore, ok := value.(*contextCredentialsStore)
		if !ok {
			return Credentials{}, errContextCredentialsFailure
		}

		creds := ctxCredStore.Get()
		if err := b.CheckCredentials(creds, true); err != nil {
			return Credentials{}, fmt.Errorf("context credentials issue: %w", err)
		}
		return creds, nil
	}
	return b.API.credentials, b.CheckCredentials(b.API.credentials, false)
}

// ValidateAPICredentials validates the exchanges API credentials
func (b *Base) ValidateAPICredentials(creds Credentials) error {
	if creds.IsEmpty() {
		return fmt.Errorf("%s %w", b.Name, errCredentialsAreEmpty)
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

type contextCredentialsStore struct {
	creds Credentials
	mu    sync.RWMutex
}

func (c *contextCredentialsStore) Load(creds Credentials) {
	c.mu.Lock()
	c.creds = creds
	c.mu.Unlock()
}

func (c *contextCredentialsStore) Get() Credentials {
	c.mu.RLock()
	creds := c.creds
	c.mu.RUnlock()
	return creds
}

// SetKey sets new key for the default credentials
func (a *API) SetKey(key string) {
	a.credentials.Key = key
}

// SetSecret sets new secret for the default credentials
func (a *API) SetSecret(secret string) {
	a.credentials.Secret = secret
}

// SetClientID sets new clientID for the default credentials
func (a *API) SetClientID(clientID string) {
	a.credentials.ClientID = clientID
}

// SetPEMKey sets pem key for the default credentials
func (a *API) SetPEMKey(pem string) {
	a.credentials.PEMKey = pem
}

// SetSubaccount sets sub account for the default credentials
func (a *API) SetSubaccount(sub string) {
	a.credentials.Subaccount = sub
}
