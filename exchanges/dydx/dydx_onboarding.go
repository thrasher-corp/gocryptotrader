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

	"github.com/ethereum/go-ethereum/common/hexutil"
	mytypes "github.com/ethereum/go-ethereum/core/types"
)

// Onboarding onboard a user so they can begin using dYdX V3 API. This will generate a user, account and derive a key, passphrase and secret from the signature.
func (dy *DYDX) Onboarding(ctx context.Context, arg *OnboardingParam, privateKey string) (*OnboardingResponse, error) {
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
	if ethereumAddress == "" || privateKey == "" {
		return errors.New("both ethereum Address and private key are required")
	}
	urlPath, err := dy.API.Endpoints.GetURL(endpoint)
	if err != nil {
		return err
	}
	var dataString string
	if data == nil {
		dataString = ""
	} else {
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
	newRequest := func() (*request.Item, error) {
		var body io.Reader
		var payload []byte
		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		if err != nil {
			return nil, err
		}
		timestamp := time.Now().UTC().Format(timeFormat)
		var signature string
		if onboarding {
			signature, err = dy.SignEIP712Onboarding(privateKey)
			if err != nil {
				println("Onboarding signature error: ", err.Error())
				return nil, err
			}
		} else {
			signature, err = dy.SignEIP712EthereumKey(method, "/"+dydxAPIVersion+path, dataString, timestamp, privateKey)
			if err != nil {
				println("Ethereum Key signature error: ", err.Error())
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
		println(urlPath + path)
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

// SignEIP712Onboarding creates an EIP ethereum.
func (dy *DYDX) SignEIP712Onboarding(privKey string) (string, error) {
	eipMessage := fmt.Sprintf(`
	{
		"types": {
			"EIP712Domain": [
			{"name": "name", "type": "string"},
			{"name": "version", "type": "string"},
			{"name": "chainId", "type": "uint64"}
			],
			"Message": [
			{"name": "action", "type": "string"},
			{"name": "onlySignOn", "type": "string"}
			]
		},
		"primaryType": "dYdX",
		"domain": {
			"name": "dydx",
			"version": "1.0",
			"chainId": 1
		},
		"message": {
			"action": "DYDX-ONBOARDING",
			"onlySignOn": "https://trade.dydx.exchange"
		}
	}`, "DYDX-ONBOARDING", dydxOnlySignOnDomainMainnet)
	privateKeyECDSA, err := crypto.HexToECDSA(strings.Replace(privKey, "0x", "", -1))
	if err != nil {
		return "", err
	}

	// Sign the hash with the private key
	r, s, err := ecdsa.Sign(rand.Reader, privateKeyECDSA, []byte(eipMessage))
	if err != nil {
		return "", err
	}

	rb := r.Bytes()
	sb := s.Bytes()

	// Pad the signatures to ensure each part is padded up to length 32
	for len(rb) < 32 {
		rb = append([]byte{0}, rb...)
	}
	for len(sb) < 32 {
		sb = append([]byte{0}, sb...)
	}
	signature := append(rb, sb...)
	sig := hex.EncodeToString(signature)
	if len(sig) > 132 {
		return sig[:132], nil
	}
	return sig, nil
}

// SignEIP712EthereumKey creates an EIP ethereum.
func (dy *DYDX) SignEIP712EthereumKey(method, requestPath, body string, timestamp string, privKey string) (string, error) {
	eipMessage := fmt.Sprintf(`
	{
		"types": {
			"EIP712Domain": [
			  {"name": "name", "type": "string"},
			  {"name": "version", "type": "string"},
			  {"name": "chainId", "type": "uint64"}
			],
			"Message": [
			  {"name": "method", "type": "string"},
			  {"name": "requestPath", "type": "string"},
			  {"name": "body", "type": "string"},
			  {"name": "timestamp", "type": "string"}
			]
		  },
		  "primaryType": "dYdX",
		  "domain": {
			"name": "dydx",
			"version": "1.0",
			"chainId": 1
		  },
		  "message": {
			"method": "%s",
			"requestPath": "%s",
			"body": "%s",
			"timestamp": %s
		  }
	}`, method, requestPath, body, timestamp)

	// Encode the EIP-712 message
	encodedMessage := EncodeEIP712Message(eipMessage)

	// Generate a hash of the encoded message
	hash := getHashOfTheMessage(encodedMessage)

	privateKeyECDSA, err := crypto.HexToECDSA(strings.Replace(privKey, "0x", "", -1))
	if err != nil {
		return "", err
	}

	// Sign the hash with the private key
	r, s, err := ecdsa.Sign(rand.Reader, privateKeyECDSA, hash)
	if err != nil {
		return "", err
	}

	// Marshal the r and s value into a byte array
	bytes := append(r.Bytes(), s.Bytes()...)
	sig := hex.EncodeToString(bytes)
	return sig[:44], nil
}

// Encode the EIP-712 message
func EncodeEIP712Message(msg string) []byte {
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

// generate onboarding EIP712
func generateEIP712(key *ecdsa.PrivateKey) (string, error) {

	// set domain struct
	domain := EIP712Domain{
		Name:    "dydx",
		Version: "1.0",
		ChainID: 1,
	}

	// set message struct
	message := Message{
		Action:     "DYDX-ONBOARDING",
		OnlySignOn: "https://trade.dydx.exchange",
	}

	// get types
	types := []interface{}{
		map[string]string{"name": "name", "type": "string"},
		map[string]string{"name": "version", "type": "string"},
		map[string]string{"name": "chainId", "type": "uint64"},
		map[string]string{"name": "action", "type": "string"},
		map[string]string{"name": "onlySignOn", "type": "string"},
	}

	// get domain fields
	domainFields := []interface{}{
		domain.Name, domain.Version, domain.ChainID,
	}

	// get message fields
	messageFields := []interface{}{
		message.Action, message.OnlySignOn,
	}

	// generate payload
	data := map[string]interface{}{
		"primaryType": "dYdX",
		"types": map[string]interface{}{
			"EIP712Domain": types[0:3],
			"Message":      types[3:],
		},
		"domain":  domainFields,
		"message": messageFields,
	}

	// get signer
	signer := mytypes.MakeSigner(key)

	// generate signature
	payload, err := signer.Hash(data)
	if err != nil {
		return "", err
	}

	// generate signature from payload
	signature, err := crypto.Sign(payload, key)
	if err != nil {
		return "", err
	}

	sigBytes, err := hex.DecodeString(hexutil.Encode(signature))
	if err != nil {
		return "", err
	}

	length := len(sigBytes)
	// make sure signature length is 132 characters
	if length != 132 {
		return "", fmt.Errorf("signature length is unexpected: %v != 132", length)
	}

	// format signature
	signatureHex := strings.ToLower(hexutil.Encode(sigBytes))
	return signatureHex, nil
}

func main() {
	// Generate a new random account
	key, _ := crypto.GenerateKey()

	// generate onboarding EIP712
	signature, err := generateEIP712(key)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Signature:", signature)
}

/*package main

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	var data interface{}

	// Parse the json data string into an interface
	err := json.Unmarshal([]byte(`{
		"types": {
			"EIP712Domain": [
			{"name": "name", "type": "string"},
			{"name": "version", "type": "string"},
			{"name": "chainId", "type": "uint64"}
			],
			"Message": [
			{"name": "action", "type": "string"},
			{"name": "onlySignOn", "type": "string"}
			]
		},
		"primaryType": "dYdX",
		"domain": {
			"name": "dydx",
			"version": "1.0",
			"chainId": 1
		},
		"message": {
			"action": "DYDX-ONBOARDING",
			"onlySignOn": "https://trade.dydx.exchange"
		}
	}`), &data)
	if err != nil {
		log.Fatal(err)
	}

	// Generate a new EIP712 private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	// Create an auth with the EIP712 private key
	auth := bind.NewKeyedTransactor(privateKey)

	// Sign the data using the eip712hash function
	hash := crypto.Keccak256Hash(data)
	sig, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		log.Fatal(err)
	}

	// Recover the public key from the signature
	publicKey, err := crypto.SigToPub(hash[:], sig)
	if err != nil {
		log.Fatal(err)
	}

	// Get the address of the public key
	address := crypto.PubkeyToAddress(*publicKey)

	// Print the EIP712 private key
	fmt.Println("EIP712 private key:", fmt.Sprintf("0x%x", privateKey.D.Bytes()))
	fmt.Println("Public key:", fmt.Sprintf("0x%x", publicKey.X.Bytes()))
	fmt.Println("Signature:", fmt.Sprintf("0x%x", sig))
	fmt.Println("Address:", address.Hex())
}


*/
