package dydx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	solsha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/thrasher-corp/gocryptotrader/common"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/umbracle/ethgo/jsonrpc"
)

// GenerateOnboardingSignature generates an onboarding signature to be used as DYDX-SIGNATURE using the ethereumAddress
func (dy *DYDX) GenerateOnboardingSignature(signerAddress string) (string, error) {
	message := map[string]interface{}{}
	message["action"] = offChainOnboardingAction
	message["onlySignOn"] = dydxOnlySignOnDomainMainnet
	eip712Message := dy.getEIP712Message(message)
	client, err := jsonrpc.NewClient(web3ProviderURL)
	if err != nil {
		return "", err
	}
	var out []byte
	if err := client.Call(ethSignMethod, &out, signerAddress, eip712Message); err != nil {
		return "", err
	}
	return createTypedSignature(string(out), 0)
}

// GenerateEtheriumKeyPrivateEndpointsSignature sends a request to Onboarging or or Etherium private endpoints.
func (dy *DYDX) GenerateEtheriumKeyPrivateEndpointsSignature(signerAddress, method, requestPath, body, timestamp string) (string, error) {
	message := map[string]interface{}{}
	message["method"] = strings.ToUpper(method)
	message["requestPath"] = requestPath
	message["body"] = body
	message["timestamp"] = timestamp
	eip712Message := dy.getEIP712Message(message)
	var err error
	var out string
	var client *jsonrpc.Client
	client, err = jsonrpc.NewClient(web3ProviderURL)
	if err != nil {
		return "", err
	}
	if err := client.Call(ethSignMethod, &out, signerAddress, eip712Message); err != nil {
		return "", err
	}
	return createTypedSignature(out, 0)
}

// Onboarding onboard a user so they can begin using dYdX V3 API. This will generate a user, account and derive a key, passphrase and secret from the signature.
func (dy *DYDX) Onboarding(ctx context.Context, arg *OnboardingParam) (*OnboardingResponse, error) {
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
	return &resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, arg.EthereumAddress, http.MethodPost, onboarding, true, &arg, &resp)
}

// RecoverStarkKeyQuoteBalanceAndOpenPosition if you can't recover your starkKey or apiKey and need an additional way to get your starkKey and balance on our exchange, both of which are needed to call the L1 solidity function needed to recover your funds.
func (dy *DYDX) RecoverStarkKeyQuoteBalanceAndOpenPosition(ctx context.Context, ethereumAddress string) (*RecoverAPIKeysResponse, error) {
	var resp *RecoverAPIKeysResponse
	return resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, http.MethodGet, recovery, false, nil, &resp)
}

// GetRegistration gets the dYdX provided Ethereum signature required to send a registration transaction to the Starkware smart contract.
func (dy *DYDX) GetRegistration(ctx context.Context, ethereumAddress string) (*SignatureResponse, error) {
	var resp *SignatureResponse
	return resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, http.MethodGet, registration, false, nil, &resp)
}

// RegisterAPIKey create new API key credentials for a user.
func (dy *DYDX) RegisterAPIKey(ctx context.Context, ethereumAddress string) (*APIKeyCredentials, error) {
	resp := &struct {
		APIKeys APIKeyCredentials `json:"apiKey"`
	}{}
	return &resp.APIKeys, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, http.MethodPost, apiKeys, false, nil, &resp)
}

// GetAPIKeys gets all api keys associated with an Ethereum address.
// It returns an array of apiKey strings corresponding to the ethereumAddress in the request.
func (dy *DYDX) GetAPIKeys(ctx context.Context, ethereumAddress string) ([]string, error) {
	if ethereumAddress == "" {
		return nil, errMissingEthereumAddress
	}
	resp := &struct {
		APIKeys []string `json:"apiKeys"`
	}{}
	return resp.APIKeys, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, http.MethodGet, apiKeys, false, nil, &resp)
}

// DeleteAPIKeys delete an api key by key and Ethereum address.
// It requires piblic API key and ethereum address the api key is associated with.
func (dy *DYDX) DeleteAPIKeys(ctx context.Context, publicKey, ethereumAddress string) (string, error) {
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
	return resp.PublicAPIKey, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, ethereumAddress, http.MethodDelete, common.EncodeURLValues(apiKeys, params), false, nil, &resp)
}

// SendEthereumSignedRequest represents a
func (dy *DYDX) SendEthereumSignedRequest(ctx context.Context, endpoint exchange.URL, ethererumAddress, method, path string, onboarding bool, data, result interface{}) error {
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
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
		var signature string
		if onboarding {
			signature, err = dy.GenerateOnboardingSignature(ethererumAddress)
			if err != nil {
				return nil, err
			}
		} else {
			signature, err = dy.GenerateEtheriumKeyPrivateEndpointsSignature(ethererumAddress, method, "/"+dydxAPIVersion+path, dataString, timestamp)
			if err != nil {
				return nil, err
			}
		}
		headers := make(map[string]string)
		headers["DYDX-SIGNATURE"] = signature
		headers["DYDX-ETHEREUM-ADDRESS"] = ethererumAddress
		if !onboarding {
			headers["DYDX-TIMESTAMP"] = timestamp
		}
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

func createTypedSignature(signature string, sigType int) (string, error) {
	fixRawSig, err := fixRawSignature(signature)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s0%s", fixRawSig, strconv.Itoa(sigType)), nil
}

func fixRawSignature(signature string) (string, error) {
	stripped := strings.TrimPrefix(signature, "0x")
	if len(stripped) != 130 {
		panic(fmt.Sprintf("Invalid raw signature: %s", signature))
	}
	rs := stripped[:128]
	v := stripped[128:130]
	if v == "00" {
		return "0x" + rs + "1b", nil
	}
	if v == "01" {
		return "0x" + rs + "1c", nil
	}
	if v == "1b" || v == "1c" {
		return "0x" + stripped, nil
	}
	return "", fmt.Errorf("invalid v value: %s", v)
}

func (dy *DYDX) getEIP712Message(message map[string]interface{}) map[string]interface{} {
	eip712Message := map[string]interface{}{
		"types": map[string]interface{}{
			"EIP712Domain": []map[string]string{
				{
					"name": "name",
					"type": "string",
				},
				{
					"name": "version",
					"type": "string",
				},
				{
					"name": "chainId",
					"type": "uint256",
				},
			},
			eip712StructName: eip712OnboardingActionsStruct,
		},
		"domain": map[string]interface{}{
			"name":    domain,
			"version": version,
			"chainId": 1,
		},
		"primaryType": eip712StructName,
		"message":     message,
	}
	return eip712Message
}

func (dy *DYDX) getHash(action string) string {
	eip712StructStr := eip712OnboardingActionStructString
	data := [][]string{
		{"bytes32", "bytes32"},
		{hashString(eip712StructStr), hashString(action)},
	}
	data[0] = append(data[0], "bytes32")
	data[1] = append(data[1], hashString(onlySignOnDomainMainnet))
	structHash := solsha3.SoliditySHA3(data[0], data[1])
	return dy.getEip712Hash(hexutil.Encode(structHash))
}

func (dy *DYDX) getEip712Hash(structHash string) string {
	fact := solsha3.SoliditySHA3(
		[]string{"bytes2", "bytes32", "bytes32"},
		[]interface{}{"0x1901", getDomainHash(), structHash},
	)
	return fmt.Sprintf("0x%x", fact)
}

func getDomainHash() string {
	fact := solsha3.SoliditySHA3(
		[]string{"bytes32", "bytes32", "bytes32", "uint256"},
		[]interface{}{hashString(eip712DomainStringNoContract), hashString(domain), hashString(version), 1},
	)
	return fmt.Sprintf("0x%x", fact)
}

func hashString(input string) string {
	return hexutil.Encode(solsha3.SoliditySHA3([]string{"string"}, input))
}
