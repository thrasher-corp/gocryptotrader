package hyperliquid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errActionBatchTooLarge      = errors.New("signed action batch is too large")
	errActionResponse           = errors.New("exchange action failed")
	errClientOrderIDInvalid     = errors.New("client order ID must be a 16-byte hexadecimal value")
	errConfiguredAccountMissing = errors.New("configured account does not exist")
	errCredentialsChanged       = errors.New("credentials changed during authority validation")
	errSignerNotAuthorised      = errors.New("private key is not authorised for the configured account")
	errUserSignedActionInvalid  = errors.New("invalid user-signed action")
	errUserSignedMasterRequired = errors.New("user-signed actions require the configured account's master private key")
	errVaultNotAuthorised       = errors.New("vault or subaccount is not controlled by the configured account")
)

func validateClientOrderID(clientOrderID string) error {
	if len(clientOrderID) != 34 || !strings.EqualFold(clientOrderID[:2], "0x") {
		return errClientOrderIDInvalid
	}
	for _, char := range clientOrderID[2:] {
		switch {
		case char >= '0' && char <= '9':
		case char >= 'a' && char <= 'f':
		case char >= 'A' && char <= 'F':
		default:
			return errClientOrderIDInvalid
		}
	}
	return nil
}

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

func (e *Exchange) getSigningCredentials(ctx context.Context) (*accounts.Credentials, string, error) {
	credentials, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(credentials.Secret) == "" {
		return nil, "", fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, errPrivateKeyRequired)
	}
	if _, _, err := normaliseAddress(credentials.Key); err != nil {
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

func (e *Exchange) validateCachedAuthority(ctx context.Context, credentials *accounts.Credentials, force bool) (authorityValidationKey, error) {
	if e == nil || credentials == nil {
		return authorityValidationKey{}, common.ErrNilPointer
	}
	accountAddress, _, err := normaliseAddress(credentials.Key)
	if err != nil {
		return authorityValidationKey{}, err
	}
	vaultAddress := ""
	if credentials.SubAccount != "" {
		vaultAddress, _, err = normaliseAddress(credentials.SubAccount)
		if err != nil {
			return authorityValidationKey{}, err
		}
	}
	signerAddress := ""
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
	if !force && e.authorityValidated && e.authorityValidationKey == validationKey {
		return validationKey, nil
	}
	e.authorityValidated = false
	if err := e.validateCredentials(ctx); err != nil {
		return authorityValidationKey{}, err
	}
	currentCredentials, err := e.GetCredentials(ctx)
	if err != nil {
		return authorityValidationKey{}, err
	}
	if *currentCredentials != *credentials {
		return authorityValidationKey{}, errCredentialsChanged
	}
	e.authorityValidationKey = validationKey
	e.authorityValidated = true
	return validationKey, nil
}

func (e *Exchange) getUserSigningCredentials(ctx context.Context) (*accounts.Credentials, string, error) {
	credentials, subAccount, err := e.getSigningCredentials(ctx)
	if err != nil {
		return nil, "", err
	}
	accountAddress := strings.ToLower(strings.TrimSpace(credentials.Key)) // getSigningCredentials validated and normalised the address format.
	key, err := parsePrivateKey(credentials.Secret)
	if err != nil {
		return nil, "", err
	}
	signerAddress := privateKeyAddress(key)
	key.Zero()
	if signerAddress != accountAddress {
		return nil, "", fmt.Errorf("%w: signer %s does not match account %s", errUserSignedMasterRequired, signerAddress, accountAddress)
	}
	return credentials, subAccount, nil
}

func (e *Exchange) sendSignedAction(ctx context.Context, action any, batchLength int, result *exchangeActionResponse) error {
	if e == nil || e.Requester == nil || e.API.Endpoints == nil || result == nil {
		return common.ErrNilPointer
	}
	if batchLength > maximumActionBatchSize {
		return fmt.Errorf("%w: maximum %d, got %d", errActionBatchTooLarge, maximumActionBatchSize, batchLength)
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	credentials, vaultAddress, err := e.getSigningCredentials(ctx)
	if err != nil {
		return err
	}
	validationKey, err := e.validateCachedAuthority(ctx, credentials, false)
	if err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	nonce := e.nextNonce()
	signature, err := signL1Action(credentials.Secret, action, vaultAddress, nonce, nil, e.isMainnetEnvironment())
	if err != nil {
		return err
	}
	payload := &signedActionRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    signature,
		VaultAddress: vaultAddress,
	}
	err = e.SendPayload(ctx, exchangeActionEndpointLimit(batchLength), func() (*request.Item, error) {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + "/exchange",
			Headers:                map[string]string{"Content-Type": "application/json"},
			Body:                   bytes.NewReader(body),
			Result:                 result,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		e.authorityValidationMu.Lock()
		if e.authorityValidationKey == validationKey {
			e.authorityValidated = false
		}
		e.authorityValidationMu.Unlock()
		return err
	}
	if result.Status != "ok" {
		e.authorityValidationMu.Lock()
		if e.authorityValidationKey == validationKey {
			e.authorityValidated = false
		}
		e.authorityValidationMu.Unlock()
		var message string
		if err := json.Unmarshal(result.Response, &message); err != nil {
			message = strings.TrimSpace(string(result.Response))
		}
		if message == "" || message == "null" {
			message = result.Status
		}
		return fmt.Errorf("%w: %s", errActionResponse, message)
	}
	return nil
}

func (e *Exchange) sendUserSignedAction(
	ctx context.Context,
	credentials *accounts.Credentials,
	actionType,
	primaryType,
	nonceField string,
	fields []eip712Field,
	result *exchangeActionResponse,
) (uint64, error) {
	if e == nil || e.Requester == nil || e.API.Endpoints == nil || credentials == nil || result == nil {
		return 0, common.ErrNilPointer
	}
	actionType = strings.TrimSpace(actionType)
	primaryType = strings.TrimSpace(primaryType)
	nonceField = strings.TrimSpace(nonceField)
	if actionType == "" || primaryType == "" || (nonceField != "nonce" && nonceField != "time") {
		return 0, errUserSignedActionInvalid
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return 0, err
	}
	currentCredentials, _, err := e.getUserSigningCredentials(ctx)
	if err != nil {
		return 0, err
	}
	if *currentCredentials != *credentials {
		return 0, fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, errCredentialsChanged)
	}
	if _, err := e.validateCachedAuthority(ctx, currentCredentials, false); err != nil {
		return 0, fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	nonce := e.nextNonce()
	chain := "Testnet"
	if e.isMainnetEnvironment() {
		chain = "Mainnet"
	}
	action := map[string]any{
		"type":             actionType,
		"signatureChainId": userSignedChainIDHex,
		"hyperliquidChain": chain,
		nonceField:         nonce,
	}
	signingFields := make([]eip712Field, 0, len(fields)+2)
	signingFields = append(signingFields, eip712Field{Name: "hyperliquidChain", Type: "string", Value: chain})
	seen := map[string]struct{}{
		"type":             {},
		"signatureChainId": {},
		"hyperliquidChain": {},
		nonceField:         {},
	}
	for i := range fields {
		if _, ok := seen[fields[i].Name]; ok || strings.TrimSpace(fields[i].Name) == "" {
			return 0, fmt.Errorf("%w: duplicate or reserved field %q", errUserSignedActionInvalid, fields[i].Name)
		}
		seen[fields[i].Name] = struct{}{}
		action[fields[i].Name] = fields[i].Value
		signingFields = append(signingFields, fields[i])
	}
	signingFields = append(signingFields, eip712Field{Name: nonceField, Type: "uint64", Value: nonce})
	signature, err := signUserSignedAction(credentials.Secret, primaryType, signingFields)
	if err != nil {
		return 0, err
	}
	payload := &signedActionRequest{
		Action:    action,
		Nonce:     nonce,
		Signature: signature,
	}
	body, _ := json.Marshal(payload) // EIP-712 field validation limits action values to JSON-safe strings, booleans, and uint64s.
	err = e.SendPayload(ctx, exchangeActionEndpointLimit(1), func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + "/exchange",
			Headers:                map[string]string{"Content-Type": "application/json"},
			Body:                   bytes.NewReader(body),
			Result:                 result,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		e.authorityValidationMu.Lock()
		e.authorityValidated = false
		e.authorityValidationMu.Unlock()
		return 0, err
	}
	if result.Status != "ok" {
		e.authorityValidationMu.Lock()
		e.authorityValidated = false
		e.authorityValidationMu.Unlock()
		var message string
		if err := json.Unmarshal(result.Response, &message); err != nil {
			message = strings.TrimSpace(string(result.Response))
		}
		if message == "" || message == "null" {
			message = result.Status
		}
		return 0, fmt.Errorf("%w: %s", errActionResponse, message)
	}
	return nonce, nil
}

func (e *Exchange) isMainnetEnvironment() bool {
	return e == nil || e.Config == nil || !e.Config.UseSandbox
}

func (e *Exchange) validateCredentials(ctx context.Context) error {
	credentials, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	accountAddress, _, err := normaliseAddress(credentials.Key)
	if err != nil {
		return err
	}
	accountRole, err := e.GetUserRole(ctx, accountAddress)
	if err != nil {
		return err
	}
	if accountRole.Role == "missing" {
		return errConfiguredAccountMissing
	}
	if accountRole.Role != "user" {
		return fmt.Errorf("%w: configured account role is %s, expected user", errConfiguredAccountMissing, accountRole.Role)
	}
	if credentials.SubAccount != "" {
		vaultAddress, _, err := normaliseAddress(credentials.SubAccount)
		if err != nil {
			return err
		}
		vaultRole, err := e.GetUserRole(ctx, vaultAddress)
		if err != nil {
			return err
		}
		switch vaultRole.Role {
		case "vault":
			details, err := e.GetVaultDetails(ctx, vaultAddress, accountAddress)
			if err != nil {
				return err
			}
			leader, _, err := normaliseAddress(details.Leader)
			if err != nil || leader != accountAddress {
				return errVaultNotAuthorised
			}
		case "subAccount":
			master, _, err := normaliseAddress(vaultRole.Data.Master)
			if err != nil || master != accountAddress {
				return errVaultNotAuthorised
			}
		default:
			return errVaultNotAuthorised
		}
	}
	if strings.TrimSpace(credentials.Secret) == "" {
		return nil
	}
	key, err := parsePrivateKey(credentials.Secret)
	if err != nil {
		return err
	}
	signerAddress := privateKeyAddress(key)
	key.Zero()
	if signerAddress == accountAddress {
		return nil
	}
	signerRole, err := e.GetUserRole(ctx, signerAddress)
	if err != nil {
		return err
	}
	authorisedUser, _, err := normaliseAddress(signerRole.Data.User)
	if err != nil ||
		signerRole.Role != "agent" ||
		authorisedUser != accountAddress {
		return errSignerNotAuthorised
	}
	return nil
}
