package dydx

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/crypto/sha3"
)

// Onboarding onboard a user so they can begin using dYdX V3 API. This will generate a user, account and derive a key, passphrase and secret from the signature.
func (dy *DYDX) Onboarding(ctx context.Context, arg *OnboardingParam, privateKey string) (*OnboardingResponse, error) {
	if arg == nil {
		return nil, fmt.Errorf("%w, nil argument", common.ErrNilPointer)
	}
	var resp OnboardingResponse
	if arg.StarkXCoordinate == "" {
		return nil, errors.New("missing Stark Key X-Coordinate")
	}
	if arg.StarkYCoordinate == "" {
		return nil, errors.New("missing Stark Key Y-Coordinate")
	}
	if arg.EthereumAddress == "" {
		return nil, errMissingEthereumAddress
	}
	if arg.Country == "" {
		return nil, errors.New("country is required")
	}
	return &resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, arg.EthereumAddress, privateKey, http.MethodPost, onboarding, true, &arg, &resp)
}

// RecoverStarkKeyQuoteBalanceAndOpenPosition if you can't recover your starkKey or apiKey and need an additional way to get your starkKey and balance on our exchange, both of which are needed to call the L1 solidity function needed to recover your funds.
func (dy *DYDX) RecoverStarkKeyQuoteBalanceAndOpenPosition(ctx context.Context, ethereumAddress, privateKey string) (*RecoverAPIKeysResponse, error) {
	var resp *RecoverAPIKeysResponse
	return resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, privateKey, http.MethodGet, recovery, false, nil, &resp)
}

// GetRegistration gets the dYdX provided Ethereum signature required to send a registration transaction to the Starkware smart contract.
func (dy *DYDX) GetRegistration(ctx context.Context, ethereumAddress, privateKey string) (*SignatureResponse, error) {
	var resp *SignatureResponse
	return resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, privateKey, http.MethodGet, registration, false, nil, &resp)
}

// RegisterAPIKey create new API key credentials for a user.
func (dy *DYDX) RegisterAPIKey(ctx context.Context, ethereumAddress, privateKey string) (*APIKeyCredentials, error) {
	resp := &struct {
		APIKeys APIKeyCredentials `json:"apiKey"`
	}{}
	return &resp.APIKeys, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, privateKey, http.MethodPost, apiKeys, false, nil, &resp)
}

// GetAPIKeys gets all api keys associated with an Ethereum address.
// It returns an array of apiKey strings corresponding to the ethereumAddress in the request.
func (dy *DYDX) GetAPIKeys(ctx context.Context, ethereumAddress, privateKey string) ([]string, error) {
	if ethereumAddress == "" {
		return nil, errMissingEthereumAddress
	}
	resp := &struct {
		APIKeys []string `json:"apiKeys"`
	}{}
	return resp.APIKeys, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, privateKey, http.MethodGet, apiKeys, false, nil, &resp)
}

// DeleteAPIKeys delete an api key by key and Ethereum address.
// It requires piblic API key and ethereum address the api key is associated with.
func (dy *DYDX) DeleteAPIKeys(ctx context.Context, publicKey, ethereumAddress, privateKey string) (string, error) {
	if publicKey == "" {
		return "", errMissingPublicKey
	}
	if ethereumAddress == "" {
		return "", errMissingEthereumAddress
	}
	params := url.Values{}
	params.Set("apiKey", publicKey)
	params.Set("ethereumAdddress", ethereumAddress)
	resp := struct {
		PublicAPIKey string `json:"apiKey"`
	}{}
	return resp.PublicAPIKey, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, privateKey, http.MethodDelete, common.EncodeURLValues(apiKeys, params), false, nil, &resp)
}

// SendEthereumSignedRequest sends an http request with onboarding and ethereum signed headers to the server.
func (dy *DYDX) SendEthereumSignedRequest(ctx context.Context, endpoint exchange.URL, ethereumAddress, privateKey, method, path string, onboarding bool, data, result interface{}) error {
	if ethereumAddress == "" {
		return errors.New("ethereum Address is required")
	}
	if privateKey == "" {
		return errors.New("private key is required")
	}
	urlPath, err := dy.API.Endpoints.GetURL(endpoint)
	if err != nil {
		return err
	}
	var dataString string
	if data != nil {
		var value []byte
		value, err = json.Marshal(data)
		if err != nil {
			return err
		}
		dataString = string(value)
	}
	var creds *account.Credentials
	creds, err = dy.GetCredentials(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	var payload []byte
	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
	}
	if err != nil {
		return err
	}
	newRequest := func() (*request.Item, error) {
		timestamp := time.Now().UTC().Format(timeFormat)
		var signature string
		privateKeyECDSA, err := crypto.HexToECDSA(strings.Replace(privateKey, "0x", "", -1))
		if err != nil {
			return nil, err
		}
		if onboarding {
			signature, err = generateOnboardingEIP712(privateKeyECDSA)
			if err != nil {
				return nil, err
			}
		} else {
			signature, err = generateAPIKeyEIP712(privateKeyECDSA, method, "/"+dydxAPIVersion+path, dataString, timestamp)
			if err != nil {
				return nil, err
			}
		}
		headers := make(map[string]string)
		headers["DYDX-SIGNATURE"] = signature
		headers["DYDX-ETHEREUM-ADDRESS"] = ethereumAddress
		if !onboarding {
			headers["DYDX-TIMESTAMP"] = timestamp
		}
		headers["DYDX-API-KEY"] = creds.Key
		headers["DYDX-PASSPHRASE"] = creds.PEMKey
		headers["Content-Type"] = "application/json"
		return &request.Item{
			Method:        method,
			Path:          urlPath + path,
			Headers:       headers,
			Body:          body,
			Result:        result,
			AuthRequest:   true,
			Verbose:       dy.Verbose,
			HTTPDebugging: dy.HTTPDebugging,
			HTTPRecording: dy.HTTPRecording,
		}, nil
	}
	return dy.SendPayload(ctx, request.Unset, newRequest)
}

// SignEIP712EthereumKey creates an Ethereum Improvement Proposal(EIP) ethereum.
func (dy *DYDX) SignEIP712EthereumKey(method, requestPath, body, timestamp, privKey string) (string, error) {
	eipMessage := fmt.Sprintf(ethereumSigningTemplate, method, requestPath, body, timestamp)
	encodedMessage := encodeEIP712Message(eipMessage)
	hash := getHashOfTheMessage(encodedMessage)
	privateKeyECDSA, err := crypto.HexToECDSA(strings.Replace(privKey, "0x", "", -1))
	if err != nil {
		return "", err
	}
	r, s, err := ecdsa.Sign(rand.Reader, privateKeyECDSA, hash)
	if err != nil {
		return "", err
	}

	// Marshal the r and s value into a byte array
	bytes := append(r.Bytes(), s.Bytes()...)
	sig := hex.EncodeToString(bytes)
	return sig[:44], nil
}

// encodeEIP712Message the EIP-712 message to a byte array
// Ethereum Improvement Proposal(EIP)
func encodeEIP712Message(msg string) []byte {
	if msg == "" {
		return []byte{}
	}
	hashedMessage := big.NewInt(0).SetBytes(getHashOfTheMessage([]byte(msg)))
	bz := append(make([]byte, 32), hashedMessage.Bytes()...)
	return bz
}

// Generate a hash of the message
func getHashOfTheMessage(m []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(m)
	return hash.Sum(nil)
}

// EIP712Domain - type for EIP712 domain
type EIP712Domain struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	ChainID uint64 `json:"chainId"`
}

// Message - type for message
type Message struct {
	Action     string `json:"action"`
	OnlySignOn string `json:"onlySignOn"`
}

func generateOnboardingEIP712(privateKey *ecdsa.PrivateKey) (string, error) {
	domain := EIP712Domain{
		Name:    "dydx",
		Version: "1.0",
		ChainID: 1,
	}
	types := []interface{}{
		map[string]string{"name": "name", "type": "string"},
		map[string]string{"name": "version", "type": "string"},
		map[string]string{"name": "chainId", "type": "uint64"},
		map[string]string{"name": "action", "type": "string"},
		map[string]string{"name": "onlySignOn", "type": "string"},
	}
	domainFields := []interface{}{
		domain.Name, domain.Version, domain.ChainID,
	}
	messageFields := []interface{}{
		"DYDX-ONBOARDING", "https://trade.dydx.exchange",
	}
	data := map[string]interface{}{
		"primaryType": "dYdX",
		"types": map[string]interface{}{
			"EIP712Domain": types[0:3],
			"Message":      types[3:],
		},
		"domain":  domainFields,
		"message": messageFields,
	}
	eipMessage, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	encodedMessage := encodeEIP712Message(string(eipMessage))
	hash := getHashOfTheMessage(encodedMessage)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash)
	if err != nil {
		return "", err
	}
	// Marshal the r and s value into a byte array
	bytes := append(r.Bytes(), s.Bytes()...)
	sig := hex.EncodeToString(bytes)
	return sig, nil
}

// generateAPIKeyEIP712 generated an EIP712 API key signature using private key
func generateAPIKeyEIP712(privateKey *ecdsa.PrivateKey, method, requestPath, body, timestamp string) (string, error) {
	domain := EIP712Domain{
		Name:    "dydx",
		Version: "1.0",
		ChainID: 1,
	}
	types := []interface{}{
		map[string]string{"name": "name", "type": "string"},
		map[string]string{"name": "version", "type": "string"},
		map[string]string{"name": "chainId", "type": "uint64"},
		map[string]string{"name": "method", "type": "string"},
		map[string]string{"name": "requestPath", "type": "string"},
		map[string]string{"name": "body", "type": "string"},
		map[string]string{"name": "timestamp", "type": "string"},
	}
	domainFields := []interface{}{
		domain.Name, domain.Version, domain.ChainID,
	}
	messageFields := []interface{}{method, requestPath, body, timestamp}
	data := map[string]interface{}{
		"primaryType": "dYdX",
		"types": map[string]interface{}{
			"EIP712Domain": types[0:3],
			"Message":      types[3:],
		},
		"domain":  domainFields,
		"message": messageFields,
	}

	eipMessage, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	encodedMessage := encodeEIP712Message(string(eipMessage))
	hash := getHashOfTheMessage(encodedMessage)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash)
	if err != nil {
		return "", err
	}
	bytes := append(r.Bytes(), s.Bytes()...)
	sig := hex.EncodeToString(bytes)
	return sig[:44], nil
}
