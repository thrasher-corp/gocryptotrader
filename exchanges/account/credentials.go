package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc/metadata"
)

// contextCredential is a string flag for use with context values when setting
// credentials internally or via gRPC.
type contextCredential string

const (
	// ContextCredentialsFlag used for retrieving api credentials from context
	ContextCredentialsFlag contextCredential = "apicredentials"
	// ContextSubAccountFlag used for retrieving just the sub account from
	// context, when the default config credentials sub account needs to be
	// changed while the same keys can be used.
	ContextSubAccountFlag contextCredential = "subaccountoverride"

	apiKeyDisplaySize = 16
)

// Default credential values
const (
	Key             = "key"
	Secret          = "secret"
	SubAccountSTR   = "subaccount"
	ClientID        = "clientid"
	OneTimePassword = "otp"
	PEMKey          = "pemkey"
)

var (
	errMetaDataIsNil                   = errors.New("meta data is nil")
	errInvalidCredentialMetaDataLength = errors.New("invalid meta data to process credentials")
	errMissingInfo                     = errors.New("cannot parse meta data missing information in key value pair")
)

// Credentials define parameters that allow for an authenticated request.
type Credentials struct {
	Key                 string
	Secret              string
	ClientID            string // TODO: Implement with exchange orders functionality
	PEMKey              string
	SubAccount          string
	OneTimePassword     string
	SecretBase64Decoded bool
	// TODO: Add AccessControl uint8 for READ/WRITE/Withdraw capabilities.
}

// GetMetaData returns the credentials for metadata context deployment
func (c *Credentials) GetMetaData() (flag, values string) {
	vals := make([]string, 0, 6)
	if c.Key != "" {
		vals = append(vals, Key+":"+c.Key)
	}
	if c.Secret != "" {
		vals = append(vals, Secret+":"+c.Secret)
	}
	if c.SubAccount != "" {
		vals = append(vals, SubAccountSTR+":"+c.SubAccount)
	}
	if c.ClientID != "" {
		vals = append(vals, ClientID+":"+c.ClientID)
	}
	if c.PEMKey != "" {
		vals = append(vals, PEMKey+":"+c.PEMKey)
	}
	if c.OneTimePassword != "" {
		vals = append(vals, OneTimePassword+":"+c.OneTimePassword)
	}
	return string(ContextCredentialsFlag), strings.Join(vals, ",")
}

// String prints out basic credential info (obfuscated) to track key instances
// associated with exchanges.
func (c *Credentials) String() string {
	obfuscated := c.Key
	if len(obfuscated) > apiKeyDisplaySize {
		obfuscated = obfuscated[:apiKeyDisplaySize]
	}
	return fmt.Sprintf("Key:[%s...] SubAccount:[%s] ClientID:[%s]",
		obfuscated,
		c.SubAccount,
		c.ClientID)
}

// getInternal returns the values for assignment to an internal context
func (c *Credentials) getInternal() (contextCredential, *ContextCredentialsStore) {
	if c.IsEmpty() {
		return "", nil
	}
	store := &ContextCredentialsStore{}
	store.Load(c)
	return ContextCredentialsFlag, store
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

// Equal determines if the keys are the same.
// OTP omitted because it's generated per request.
// PEMKey and Secret omitted because of direct correlation with api key.
func (c *Credentials) Equal(other *Credentials) bool {
	return c != nil &&
		other != nil &&
		c.Key == other.Key &&
		c.ClientID == other.ClientID &&
		(c.SubAccount == other.SubAccount || c.SubAccount == "" && other.SubAccount == "main" || c.SubAccount == "main" && other.SubAccount == "")
}

// ContextCredentialsStore protects the stored credentials for use in a context
type ContextCredentialsStore struct {
	creds *Credentials
	mu    sync.RWMutex
}

// Load stores provided credentials
func (c *ContextCredentialsStore) Load(creds *Credentials) {
	// Segregate from external call
	cpy := *creds
	c.mu.Lock()
	c.creds = &cpy
	c.mu.Unlock()
}

// Get returns the full credentials from the store
func (c *ContextCredentialsStore) Get() *Credentials {
	c.mu.RLock()
	creds := *c.creds
	c.mu.RUnlock()
	return &creds
}

// ParseCredentialsMetadata intercepts and converts credentials metadata to a
// static type for authentication processing and protection.
func ParseCredentialsMetadata(ctx context.Context, md metadata.MD) (context.Context, error) {
	if md == nil {
		return ctx, errMetaDataIsNil
	}

	credMD, ok := md[string(ContextCredentialsFlag)]
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
		case Key:
			ctxCreds.Key = keyVals[1]
		case Secret:
			ctxCreds.Secret = keyVals[1]
		case SubAccountSTR:
			// Capture sub account as this can override if other values are
			// not included in metadata.
			subAccountHere = keyVals[1]
		case ClientID:
			ctxCreds.ClientID = keyVals[1]
		case PEMKey:
			ctxCreds.PEMKey = keyVals[1]
		case OneTimePassword:
			ctxCreds.OneTimePassword = keyVals[1]
		}
	}
	if ctxCreds.IsEmpty() && subAccountHere != "" {
		// This will override default sub account details if needed.
		return DeploySubAccountOverrideToContext(ctx, subAccountHere), nil
	}
	// merge sub account to main context credentials
	ctxCreds.SubAccount = subAccountHere
	return DeployCredentialsToContext(ctx, &ctxCreds), nil
}

// DeployCredentialsToContext sets credentials for internal use to context which
// can override default credential values.
func DeployCredentialsToContext(ctx context.Context, creds *Credentials) context.Context {
	flag, store := creds.getInternal()
	return context.WithValue(ctx, flag, store)
}

// DeploySubAccountOverrideToContext sets subaccount as override to credentials
// as a separate flag.
func DeploySubAccountOverrideToContext(ctx context.Context, subAccount string) context.Context {
	return context.WithValue(ctx, ContextSubAccountFlag, subAccount)
}

// String strings the credentials in a protected way.
func (p *Protected) String() string {
	return p.creds.String()
}

// Equal determines if the keys are the same
func (p *Protected) Equal(other *Credentials) bool {
	return p.creds.Equal(other)
}
