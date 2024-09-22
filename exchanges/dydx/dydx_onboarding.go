package dydx

/*
// RecoverStarkKeyQuoteBalanceAndOpenPosition if you can't recover your starkKey or apiKey and need an additional way to get your starkKey and balance on our exchange, both of which are needed to call the L1 solidity function needed to recover your funds.
func (dy *DYDX) RecoverStarkKeyQuoteBalanceAndOpenPosition(ctx context.Context) (*RecoverAPIKeysResponse, error) {
	var resp *RecoverAPIKeysResponse
	return resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, http.MethodGet, recovery, false, nil, &resp)
}

// GetRegistration gets the dYdX provided Ethereum signature required to send a registration transaction to the Starkware smart contract.
func (dy *DYDX) GetRegistration(ctx context.Context) (*SignatureResponse, error) {
	var resp *SignatureResponse
	return resp, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, http.MethodGet, registration, false, nil, &resp)
}

// RegisterAPIKey create new API key credentials for a user.
func (dy *DYDX) RegisterAPIKey(ctx context.Context) (*APIKeyCredentials, error) {
	resp := &struct {
		APIKeys APIKeyCredentials `json:"apiKey"`
	}{}
	return &resp.APIKeys, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, http.MethodPost, apiKeys, false, nil, &resp)
}

// GetAPIKeys gets all api keys associated with an Ethereum address.
// It returns an array of apiKey strings corresponding to the ethereumAddress in the request.
func (dy *DYDX) GetAPIKeys(ctx context.Context) ([]string, error) {
	resp := &struct {
		APIKeys []string `json:"apiKeys"`
	}{}
	return resp.APIKeys, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, http.MethodGet, apiKeys, false, nil, &resp)
}

// DeleteAPIKeys delete an api key by key and Ethereum address.
// It requires piblic API key and ethereum address the api key is associated with.
func (dy *DYDX) DeleteAPIKeys(ctx context.Context, apiKey string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("%w, api key to be deleted", errMissingPublicKey)
	}
	params := url.Values{}
	params.Set("apiKey", apiKey)
	creds, err := dy.GetCredentials(ctx)
	if err != nil {
		return "", err
	}
	_, address, err := GeneratePublicKeyAndAddress(creds.PrivateKey)
	if err != nil {
		return "", err
	}
	params.Set("ethereumAdddress", address)
	resp := struct {
		PublicAPIKey string `json:"apiKey"`
	}{}
	return resp.PublicAPIKey, dy.SendEthereumSignedRequest(ctx, exchange.RestSpot, http.MethodDelete, common.EncodeURLValues(apiKeys, params), false, nil, &resp)
}

// SendEthereumSignedRequest sends an http request with onboarding and ethereum signed headers to the server.
func (dy *DYDX) SendEthereumSignedRequest(ctx context.Context, endpoint exchange.URL, method, path string, onboarding bool, data, result interface{}) error {
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
	privateKeyECDSA, err := crypto.HexToECDSA(strings.Replace(creds.PrivateKey, "0x", "", 1))
	if err != nil {
		return err
	}
	_, address, err := GeneratePublicKeyAndAddress(creds.PrivateKey)
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
		headers["DYDX-ETHEREUM-ADDRESS"] = address
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
			Verbose:       dy.Verbose,
			HTTPDebugging: dy.HTTPDebugging,
			HTTPRecording: dy.HTTPRecording,
		}, nil
	}
	return dy.SendPayload(ctx, request.Unset, newRequest, request.AuthenticatedRequest)
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
	messageTypes := []apitypes.Type{
		{Name: "action", Type: "string"},
		{Name: "onlySignOn", Type: "string"},
	}
	messageDatas := apitypes.TypedDataMessage{
		"action":     "DYDX-ONBOARDING",
		"onlySignOn": "https://trade.dydx.exchange",
	}

	data := apitypes.TypedData{
		Types: apitypes.Types{
			"dYdX": messageTypes,
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "dYdX",
		Domain: apitypes.TypedDataDomain{
			Name:    "dydx",
			Version: "1.0",
			ChainId: math.NewHexOrDecimal256(1),
		},
		Message: messageDatas,
	}
	domainSeparator, err := data.HashStruct("EIP712Domain", data.Domain.Map())
	if err != nil {
		return "", err
	}
	typedDataHash, err := data.HashStruct(data.PrimaryType, data.Message)
	if err != nil {
		return "", err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	dataHash := crypto.Keccak256(rawData)

	signature, err := crypto.Sign(dataHash, privateKey)
	if err != nil {
		return "", err
	}
	if signature[64] < 27 {
		signature[64] += 27
	}
	return hexutil.Encode(signature), nil
}

// generateAPIKeyEIP712 generated an EIP712 API key signature using private key
func generateAPIKeyEIP712(privateKey *ecdsa.PrivateKey, method, requestPath, body, timestamp string) (string, error) {
	messageTypes := []apitypes.Type{
		{Name: "method", Type: "string"},
		{Name: "requestPath", Type: "string"},
		{Name: "timestamp", Type: "string"},
	}
	messageDatas := apitypes.TypedDataMessage{
		"method":      method,
		"requestPath": requestPath,
		"timestamp":   timestamp,
	}
	if body != "" {
		messageTypes = append(messageTypes, apitypes.Type{Name: "body", Type: "string"})
		messageDatas["body"] = body
	}
	data := apitypes.TypedData{
		Types: apitypes.Types{
			"dYdX": messageTypes,
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "dYdX",
		Domain: apitypes.TypedDataDomain{
			Name:    "dydx",
			Version: "1.0",
			ChainId: math.NewHexOrDecimal256(1),
		},
		Message: messageDatas,
	}
	domainSeparator, err := data.HashStruct("EIP712Domain", data.Domain.Map())
	if err != nil {
		return "", err
	}
	typedDataHash, err := data.HashStruct(data.PrimaryType, data.Message)
	if err != nil {
		return "", err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	dataHash := crypto.Keccak256(rawData)
	signature, err := crypto.Sign(dataHash, privateKey)
	if err != nil {
		return "", err
	}
	if signature[64] < 27 {
		signature[64] += 27
	}
	if len(signature) > 44 {
		return hexutil.Encode(signature)[:44], nil
	}
	return hexutil.Encode(signature), nil
}
*/
