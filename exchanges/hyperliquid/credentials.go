package hyperliquid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errConfiguredAccountMissing = errors.New("configured account does not exist")
	errConfiguredAccountNotUser = errors.New("configured account is not a user account")
	errSignerNotAuthorised      = errors.New("private key is not authorised for the configured account")
	errUserSignedMasterRequired = errors.New("user-signed actions require the configured account's master private key")
	errVaultNotAuthorised       = errors.New("vault or subaccount is not controlled by the configured account")
)

// authorityValidationKey identifies the account, vault, signer and environment a successful authority validation covers
type authorityValidationKey struct {
	accountAddress string
	vaultAddress   string
	signerAddress  string
	mainnet        bool
}

// getWatchAddress returns the address whose account data is queried: the configured vault or subaccount, else the account
func (e *Exchange) getWatchAddress(ctx context.Context) (string, error) {
	credentials, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	accountAddress, _, err := normaliseAddress(credentials.Key)
	if err != nil {
		return "", err
	}
	if credentials.SubAccount == "" {
		return accountAddress, nil
	}
	vaultAddress, _, err := normaliseAddress(credentials.SubAccount)
	if err != nil {
		return "", err
	}
	return vaultAddress, nil
}

// getSigningCredentials returns credentials holding a private key, with the account address normalised, and the
// normalised vault or subaccount address to act for
func (e *Exchange) getSigningCredentials(ctx context.Context) (*accounts.Credentials, string, error) {
	credentials, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(credentials.Secret) == "" {
		return nil, "", fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, errPrivateKeyRequired)
	}
	// GetCredentials returns a copy, so normalising its key leaves the stored credentials as configured
	if credentials.Key, _, err = normaliseAddress(credentials.Key); err != nil {
		return nil, "", err
	}
	if credentials.SubAccount == "" {
		return credentials, "", nil
	}
	vaultAddress, _, err := normaliseAddress(credentials.SubAccount)
	if err != nil {
		return nil, "", err
	}
	return credentials, vaultAddress, nil
}

// getUserSigningCredentials returns signing credentials whose private key belongs to the account itself
// User-signed actions move funds, so an approved API wallet must not sign them even though it may trade
func (e *Exchange) getUserSigningCredentials(ctx context.Context) (*accounts.Credentials, string, error) {
	credentials, subAccount, err := e.getSigningCredentials(ctx)
	if err != nil {
		return nil, "", err
	}
	key, err := parsePrivateKey(credentials.Secret)
	if err != nil {
		return nil, "", err
	}
	signerAddress := privateKeyAddress(key)
	key.Zero()
	if signerAddress != credentials.Key {
		return nil, "", fmt.Errorf("%w: signer %s does not match account %s", errUserSignedMasterRequired, signerAddress, credentials.Key)
	}
	return credentials, subAccount, nil
}

// validateCachedAuthority validates the credentials' on-chain authority unless the same account, vault, signer and
// environment were already validated; force always revalidates, and a failed validation leaves that authority uncached
func (e *Exchange) validateCachedAuthority(ctx context.Context, credentials *accounts.Credentials, force bool) (authorityValidationKey, error) {
	accountAddress, _, err := normaliseAddress(credentials.Key)
	if err != nil {
		return authorityValidationKey{}, err
	}
	var vaultAddress string
	if credentials.SubAccount != "" {
		if vaultAddress, _, err = normaliseAddress(credentials.SubAccount); err != nil {
			return authorityValidationKey{}, err
		}
	}
	var signerAddress string
	if strings.TrimSpace(credentials.Secret) != "" {
		privateKey, err := parsePrivateKey(credentials.Secret)
		if err != nil {
			return authorityValidationKey{}, err
		}
		signerAddress = privateKeyAddress(privateKey)
		privateKey.Zero()
	}
	validationKey := authorityValidationKey{
		accountAddress: accountAddress,
		vaultAddress:   vaultAddress,
		signerAddress:  signerAddress,
		mainnet:        e.isMainnetEnvironment(),
	}
	e.authorityValidationMu.Lock()
	defer e.authorityValidationMu.Unlock()
	if _, validated := e.authorityValidations[validationKey]; validated && !force {
		return validationKey, nil
	}
	delete(e.authorityValidations, validationKey)
	if err := e.validateAuthority(ctx, &validationKey); err != nil {
		return authorityValidationKey{}, err
	}
	if e.authorityValidations == nil {
		e.authorityValidations = make(map[authorityValidationKey]struct{})
	}
	e.authorityValidations[validationKey] = struct{}{}
	return validationKey, nil
}

// invalidateAuthority clears a cached validation, so a rejected action revalidates the account before signing again
func (e *Exchange) invalidateAuthority(validationKey *authorityValidationKey) {
	e.authorityValidationMu.Lock()
	delete(e.authorityValidations, *validationKey)
	e.authorityValidationMu.Unlock()
}

// validateAuthority checks that the account exists, that it controls any configured vault or subaccount, and that the
// signer is either the account or one of its approved API wallets
func (e *Exchange) validateAuthority(ctx context.Context, validationKey *authorityValidationKey) error {
	accountRole, err := e.GetUserRole(ctx, validationKey.accountAddress)
	if err != nil {
		return err
	}
	switch accountRole.Role {
	case "user":
	case "missing":
		return errConfiguredAccountMissing
	default:
		return fmt.Errorf("%w: role is %s", errConfiguredAccountNotUser, accountRole.Role)
	}
	if validationKey.vaultAddress != "" {
		vaultRole, err := e.GetUserRole(ctx, validationKey.vaultAddress)
		if err != nil {
			return err
		}
		switch vaultRole.Role {
		case "vault":
			details, err := e.GetVaultDetails(ctx, validationKey.vaultAddress, validationKey.accountAddress)
			if err != nil {
				return err
			}
			if leader, _, err := normaliseAddress(details.Leader); err != nil || leader != validationKey.accountAddress {
				return errVaultNotAuthorised
			}
		case "subAccount":
			if master, _, err := normaliseAddress(vaultRole.Data.Master); err != nil || master != validationKey.accountAddress {
				return errVaultNotAuthorised
			}
		default:
			return errVaultNotAuthorised
		}
	}
	if validationKey.signerAddress == "" || validationKey.signerAddress == validationKey.accountAddress {
		return nil
	}
	signerRole, err := e.GetUserRole(ctx, validationKey.signerAddress)
	if err != nil {
		return err
	}
	if authorisedUser, _, err := normaliseAddress(signerRole.Data.User); err != nil || signerRole.Role != "agent" || authorisedUser != validationKey.accountAddress {
		return errSignerNotAuthorised
	}
	return nil
}

// isMainnetEnvironment reports whether actions are signed for mainnet rather than testnet
func (e *Exchange) isMainnetEnvironment() bool {
	return e.Config == nil || !e.Config.UseSandbox
}

// nextNonce returns a strictly increasing millisecond nonce, because Hyperliquid rejects reused signer nonces
func (e *Exchange) nextNonce() uint64 {
	now := uint64(time.Now().UnixMilli())
	for {
		previous := e.lastNonce.Load()
		next := max(now, previous+1)
		if e.lastNonce.CompareAndSwap(previous, next) {
			return next
		}
	}
}
