package hyperliquid

import (
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func newTransferTestExchange(
	t *testing.T,
	credentials *accounts.Credentials,
	responses map[string]string,
	captured *signedActionRequest,
) *Exchange {
	t.Helper()
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/info":
			var request infoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding transfer validation request should not error") {
				return
			}
			key := request.Type
			if request.DEX != "" {
				key += ":" + request.DEX
			}
			response, ok := responses[key]
			if request.Type == testUserRoleInfoType {
				if specific, exists := responses[key+":"+strings.ToLower(request.User)]; exists {
					response, ok = specific, true
				}
			}
			if !ok && request.Type == testUserRoleInfoType && credentials != nil && strings.EqualFold(request.User, credentials.Key) {
				response, ok = testUserRoleResponse, true
			}
			if !ok {
				http.Error(w, "unexpected info request", http.StatusBadRequest)
				return
			}
			_, err := w.Write([]byte(response))
			assert.NoError(t, err, "Writing transfer validation response should not error")
		case "/exchange":
			if captured != nil {
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(captured), "Decoding signed transfer action should not error") {
					return
				}
			}
			_, err := w.Write([]byte(`{"status":"ok","response":null}`))
			assert.NoError(t, err, "Writing transfer action response should not error")
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	if credentials != nil {
		setTestCredentials(ex, credentials)
	}
	return ex
}

func getCapturedAction(t *testing.T, captured *signedActionRequest) map[string]any {
	t.Helper()
	action, ok := captured.Action.(map[string]any)
	require.True(t, ok, "Captured action must decode as an object")
	return action
}

func TestGetBridgeChain(t *testing.T) {
	mainnet := new(Exchange)
	mainnet.Config = new(config.Exchange)
	assert.Equal(t, "Arbitrum", mainnet.getBridgeChain(), "Mainnet bridge should use Arbitrum")
	sandbox := new(Exchange)
	sandbox.Config = &config.Exchange{UseSandbox: true}
	assert.Equal(t, "Arbitrum Sepolia", sandbox.getBridgeChain(), "Sandbox bridge should use Arbitrum Sepolia")
}

func TestFormatTransferAmount(t *testing.T) {
	for _, tc := range []struct {
		name       string
		amount     float64
		expected   string
		expectedIs error
	}{
		{name: "integer", amount: 1, expected: "1"},
		{name: "fraction", amount: 1.23, expected: "1.23"},
		{name: "zero", expectedIs: errTransferAmountInvalid},
		{name: "negative", amount: -1, expectedIs: errTransferAmountInvalid},
		{name: "too precise", amount: 0.000000001, expectedIs: errTransferAmountInvalid},
		{name: "nan", amount: math.NaN(), expectedIs: errTransferAmountInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := formatTransferAmount(tc.amount)
			require.ErrorIs(t, err, tc.expectedIs, "Formatting a transfer amount must return the expected error")
			assert.Equal(t, tc.expected, result, "Formatted transfer amount should match")
		})
	}
}

func TestValidateUserSignedSubAccount(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, ex.validateUserSignedSubAccount(t.Context(), nil, ""), "Empty subaccount must not require validation")
	require.ErrorIs(t, ex.validateUserSignedSubAccount(t.Context(), nil, testVaultAddress), common.ErrNilPointer, "Nil credentials must return the expected error")

	failing := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	err := failing.validateUserSignedSubAccount(t.Context(), &accounts.Credentials{Key: officialSigningAddress}, testVaultAddress)
	require.Error(t, err, "Subaccount role lookup failure must be returned")

	for _, tc := range []struct {
		name       string
		role       string
		account    string
		expectedIs error
	}{
		{name: "wrong role", role: `{"role":"vault"}`, account: officialSigningAddress, expectedIs: errTransferSubAccountInvalid},
		{name: "invalid account", role: `{"role":"subAccount","data":{"master":"` + officialSigningAddress + `"}}`, account: "invalid", expectedIs: errInvalidAddress},
		{name: "invalid master", role: `{"role":"subAccount","data":{"master":"invalid"}}`, account: officialSigningAddress, expectedIs: errTransferSubAccountInvalid},
		{name: "wrong master", role: `{"role":"subAccount","data":{"master":"` + testOtherAddress + `"}}`, account: officialSigningAddress, expectedIs: errTransferSubAccountInvalid},
		{name: "owned", role: `{"role":"subAccount","data":{"master":"` + officialSigningAddress + `"}}`, account: officialSigningAddress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roleExchange := newStaticInfoExchange(t, map[string]string{testUserRoleInfoType: tc.role})
			err := roleExchange.validateUserSignedSubAccount(
				t.Context(),
				&accounts.Credentials{Key: tc.account},
				testVaultAddress)
			require.ErrorIs(t, err, tc.expectedIs, "Subaccount validation must return the expected error")
		})
	}
}

func TestUserSignedTransfersRejectAgentAccount(t *testing.T) {
	agentSecret := "0x1123456789012345678901234567890123456789012345678901234567890123"
	agentKey, err := parsePrivateKey(agentSecret)
	require.NoError(t, err, "Parsing the agent test key must not error")
	agentAddress := privateKeyAddress(agentKey)
	agentKey.Zero()

	for _, tc := range []struct {
		name string
		run  func(*Exchange) error
	}{
		{
			name: "class transfer",
			run: func(ex *Exchange) error {
				_, err := ex.TransferUSDCBetweenSpotAndPerp(t.Context(), 1, true)
				return err
			},
		},
		{
			name: "asset transfer",
			run: func(ex *Exchange) error {
				_, err := ex.SendAsset(t.Context(), &SendAssetRequest{
					Destination:    testOtherAddress,
					SourceDEX:      "spot",
					DestinationDEX: "spot",
					Token:          "USDC",
					Amount:         1,
				})
				return err
			},
		},
		{
			name: "Core USDC send",
			run: func(ex *Exchange) error {
				_, err := ex.SendCoreUSDC(t.Context(), testOtherAddress, 1)
				return err
			},
		},
		{
			name: "Core spot send",
			run: func(ex *Exchange) error {
				_, err := ex.SendCoreSpot(t.Context(), testOtherAddress, "USDC", 1)
				return err
			},
		},
		{
			name: "bridge withdrawal",
			run: func(ex *Exchange) error {
				_, err := ex.WithdrawFromBridge(t.Context(), testOtherAddress, 1)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var captured signedActionRequest
			ex := newTransferTestExchange(t, &accounts.Credentials{
				Key:    agentAddress,
				Secret: agentSecret,
			}, map[string]string{
				testUserRoleInfoType: `{"role":"agent","data":{"user":"` + officialSigningAddress + `"}}`,
				"spotMeta":           spotMetadataJSON,
			}, &captured)
			err := tc.run(ex)
			require.ErrorIs(t, err, errConfiguredAccountMissing, "An API-wallet account must be rejected")
			require.ErrorIs(t, err, request.ErrAuthRequestFailed, "An API-wallet account must classify as an authentication failure")
			assert.Zero(t, ex.lastNonce.Load(), "A rejected API-wallet account should not consume a nonce")
			assert.Nil(t, captured.Action, "A rejected API-wallet account should not reach the exchange endpoint")
		})
	}
}

func TestResolveTransferToken(t *testing.T) {
	ex := new(Exchange)
	_, _, err := ex.resolveTransferToken(t.Context(), "")
	require.ErrorIs(t, err, errTransferTokenInvalid, "Blank token must return the expected error")

	failing := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, _, err = failing.resolveTransferToken(t.Context(), "USDC")
	require.Error(t, err, "Spot metadata failure must be returned")

	valid := newStaticInfoExchange(t, map[string]string{"spotMeta": spotMetadataJSON})
	token, canonical, err := valid.resolveTransferToken(t.Context(), "USDC")
	require.NoError(t, err, "Resolving USDC must not error")
	assert.Equal(t, uint64(0), token.Index, "USDC should retain its token index")
	assert.Equal(t, "USDC:0x0", canonical, "USDC should use its full spot-transfer identifier")

	token, canonical, err = valid.resolveTransferToken(t.Context(), "HYPE:0X96")
	require.NoError(t, err, "Resolving an exact spot token must not error")
	assert.Equal(t, uint64(150), token.Index, "Spot token should retain its token index")
	assert.Equal(t, "HYPE:0x96", canonical, "Spot token identifier should use metadata casing")

	for _, invalid := range []string{"HYPE", ":0x96", "HYPE:", "MISSING:0x1"} {
		_, _, err := valid.resolveTransferToken(t.Context(), invalid)
		require.ErrorIs(t, err, errTransferTokenInvalid, "Invalid token identifier must return the expected error")
	}

	missingUSDC := newStaticInfoExchange(t, map[string]string{"spotMeta": `{"tokens":[{"name":"HYPE","index":150}]}`})
	_, _, err = missingUSDC.resolveTransferToken(t.Context(), "USDC")
	require.ErrorIs(t, err, errTransferTokenInvalid, "Missing USDC metadata must return the expected error")
}

func TestResolveTransferDEX(t *testing.T) {
	ex := new(Exchange)
	dex, err := ex.resolveTransferDEX(t.Context(), " ")
	require.NoError(t, err, "Resolving the default DEX must not error")
	assert.Empty(t, dex, "Default DEX should remain empty")
	dex, err = ex.resolveTransferDEX(t.Context(), " SPOT ")
	require.NoError(t, err, "Resolving the spot balance must not error")
	assert.Equal(t, "spot", dex, "Spot DEX should be canonicalised")

	failing := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = failing.resolveTransferDEX(t.Context(), "xyz")
	require.Error(t, err, "DEX registry failure must be returned")

	valid := newStaticInfoExchange(t, map[string]string{infoTypePerpetualDEXs: `[null,null,{"name":"xyz"}]`})
	dex, err = valid.resolveTransferDEX(t.Context(), "xyz")
	require.NoError(t, err, "Resolving a registered builder DEX must not error")
	assert.Equal(t, "xyz", dex, "Builder DEX name should be retained")
	_, err = valid.resolveTransferDEX(t.Context(), "missing")
	require.ErrorIs(t, err, errTransferDEXInvalid, "Unknown builder DEX must return the expected error")
}

func TestValidateSendAssetRoute(t *testing.T) {
	badRegistry := newStaticInfoExchange(t, map[string]string{infoTypePerpetualDEXs: `[null]`})
	sourceDEX, destinationDEX, token, err := badRegistry.validateSendAssetRoute(t.Context(), "missing", "spot", "USDC")
	require.ErrorIs(t, err, errTransferDEXInvalid, "Invalid source DEX must return the expected error")
	assert.Empty(t, sourceDEX, "Invalid source route should not return a source DEX")
	assert.Empty(t, destinationDEX, "Invalid source route should not return a destination DEX")
	assert.Empty(t, token, "Invalid source route should not return a token")

	sourceDEX, destinationDEX, token, err = badRegistry.validateSendAssetRoute(t.Context(), "", "missing", "USDC")
	require.ErrorIs(t, err, errTransferDEXInvalid, "Invalid destination DEX must return the expected error")
	assert.Empty(t, sourceDEX, "Invalid destination route should not return a source DEX")
	assert.Empty(t, destinationDEX, "Invalid destination route should not return a destination DEX")
	assert.Empty(t, token, "Invalid destination route should not return a token")

	missingToken := newStaticInfoExchange(t, map[string]string{"spotMeta": `{"tokens":[]}`})
	sourceDEX, destinationDEX, token, err = missingToken.validateSendAssetRoute(t.Context(), "spot", "spot", "USDC")
	require.ErrorIs(t, err, errTransferTokenInvalid, "Invalid transfer token must return the expected error")
	assert.Empty(t, sourceDEX, "Invalid-token route should not return a source DEX")
	assert.Empty(t, destinationDEX, "Invalid-token route should not return a destination DEX")
	assert.Empty(t, token, "Invalid-token route should not return a token")

	metadataFailure := newTransferTestExchange(t, nil, map[string]string{
		"spotMeta": spotMetadataJSON,
		"meta":     `null`,
	}, nil)
	sourceDEX, destinationDEX, token, err = metadataFailure.validateSendAssetRoute(t.Context(), "", "spot", "USDC")
	require.ErrorIs(t, err, common.ErrNilPointer, "Perpetual metadata failure must be returned")
	assert.Empty(t, sourceDEX, "Metadata-failure route should not return a source DEX")
	assert.Empty(t, destinationDEX, "Metadata-failure route should not return a destination DEX")
	assert.Empty(t, token, "Metadata-failure route should not return a token")

	wrongCollateral := newTransferTestExchange(t, nil, map[string]string{
		"spotMeta": spotMetadataJSON,
		"meta":     `{"collateralToken":150}`,
	}, nil)
	sourceDEX, destinationDEX, token, err = wrongCollateral.validateSendAssetRoute(t.Context(), "spot", "", "USDC")
	require.ErrorIs(t, err, errTransferTokenInvalid, "Non-collateral token must return the expected error")
	assert.Empty(t, sourceDEX, "Wrong-collateral route should not return a source DEX")
	assert.Empty(t, destinationDEX, "Wrong-collateral route should not return a destination DEX")
	assert.Empty(t, token, "Wrong-collateral route should not return a token")

	valid := newTransferTestExchange(t, nil, map[string]string{
		infoTypePerpetualDEXs: `[null,{"name":"xyz"}]`,
		"spotMeta":            spotMetadataJSON,
		"meta":                `{"collateralToken":0}`,
		"meta:xyz":            `{"collateralToken":0}`,
	}, nil)
	sourceDEX, destinationDEX, token, err = valid.validateSendAssetRoute(t.Context(), "", "xyz", "USDC")
	require.NoError(t, err, "Valid default-to-builder collateral route must not error")
	assert.Empty(t, sourceDEX, "Default DEX should remain empty")
	assert.Equal(t, "xyz", destinationDEX, "Destination builder DEX should be retained")
	assert.Equal(t, "USDC", token, "Validated collateral token should be retained")

	sourceDEX, destinationDEX, token, err = valid.validateSendAssetRoute(t.Context(), "spot", "spot", "HYPE:0x96")
	require.NoError(t, err, "Valid spot-to-spot route must not error")
	assert.Equal(t, "spot", sourceDEX, "Spot source should be retained")
	assert.Equal(t, "spot", destinationDEX, "Spot destination should be retained")
	assert.Equal(t, "HYPE:0x96", token, "Validated spot token should be retained")
}

func TestTransferUSDCBetweenSpotAndPerp(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.TransferUSDCBetweenSpotAndPerp(t.Context(), 0, true)
	require.ErrorIs(t, err, errTransferAmountInvalid, "Invalid class-transfer amount must return the expected error")
	_, err = ex.TransferUSDCBetweenSpotAndPerp(t.Context(), 1, true)
	require.Error(t, err, "Class transfer without credentials must error")

	invalidSubAccount := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress,
	}, map[string]string{"userRole:" + testVaultAddress: `{"role":"vault"}`}, nil)
	_, err = invalidSubAccount.TransferUSDCBetweenSpotAndPerp(t.Context(), 1, true)
	require.ErrorIs(t, err, errTransferSubAccountInvalid, "Class transfer from a vault must be rejected")

	var captured signedActionRequest
	success := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress,
	}, map[string]string{
		"userRole:" + testVaultAddress: `{"role":"subAccount","data":{"master":"` + officialSigningAddress + `"}}`,
	}, &captured)
	nonce, err := success.TransferUSDCBetweenSpotAndPerp(t.Context(), 1.23, true)
	require.NoError(t, err, "Owned-subaccount class transfer must not error")
	action := getCapturedAction(t, &captured)
	assert.Equal(t, "usdClassTransfer", action["type"], "Class-transfer action type should match")
	assert.Equal(t, "1.23 subaccount:"+testVaultAddress, action["amount"], "Class-transfer amount should identify the subaccount")
	assert.Equal(t, true, action["toPerp"], "Class-transfer direction should be retained")
	assert.Equal(t, float64(nonce), action["nonce"], "Class-transfer action nonce should match the outer nonce")
}

func TestSendAsset(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.SendAsset(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil asset transfer must return the expected error")
	_, err = ex.SendAsset(t.Context(), &SendAssetRequest{Destination: "invalid", Amount: 1})
	require.ErrorIs(t, err, errInvalidAddress, "Invalid asset-transfer destination must return the expected error")
	_, err = ex.SendAsset(t.Context(), &SendAssetRequest{Destination: testOtherAddress})
	require.ErrorIs(t, err, errTransferAmountInvalid, "Invalid asset-transfer amount must return the expected error")
	_, err = ex.SendAsset(t.Context(), &SendAssetRequest{Destination: testOtherAddress, Amount: 1})
	require.Error(t, err, "Asset transfer without credentials must error")

	invalidSubAccount := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress,
	}, map[string]string{"userRole:" + testVaultAddress: `{"role":"vault"}`}, nil)
	_, err = invalidSubAccount.SendAsset(t.Context(), &SendAssetRequest{
		Destination: testOtherAddress, SourceDEX: "spot", DestinationDEX: "spot", Token: "USDC", Amount: 1,
	})
	require.ErrorIs(t, err, errTransferSubAccountInvalid, "Asset transfer from a vault must be rejected")

	invalidRoute := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey,
	}, map[string]string{infoTypePerpetualDEXs: `[null]`}, nil)
	_, err = invalidRoute.SendAsset(t.Context(), &SendAssetRequest{
		Destination: testOtherAddress, SourceDEX: "missing", DestinationDEX: "spot", Token: "USDC", Amount: 1,
	})
	require.ErrorIs(t, err, errTransferDEXInvalid, "Invalid asset-transfer route must return the expected error")

	var captured signedActionRequest
	success := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress,
	}, map[string]string{
		"userRole:" + testVaultAddress: `{"role":"subAccount","data":{"master":"` + officialSigningAddress + `"}}`,
		"spotMeta":                     spotMetadataJSON,
	}, &captured)
	nonce, err := success.SendAsset(t.Context(), &SendAssetRequest{
		Destination: strings.ToUpper(testOtherAddress),
		SourceDEX:   "SPOT", DestinationDEX: "spot",
		Token: "HYPE:0X96", Amount: 1.25,
	})
	require.NoError(t, err, "Validated asset transfer must not error")
	action := getCapturedAction(t, &captured)
	assert.Equal(t, "sendAsset", action["type"], "Asset-transfer action type should match")
	assert.Equal(t, testOtherAddress, action["destination"], "Asset-transfer destination should be normalised")
	assert.Equal(t, "spot", action["sourceDex"], "Asset-transfer source should be canonicalised")
	assert.Equal(t, "spot", action["destinationDex"], "Asset-transfer destination DEX should be canonicalised")
	assert.Equal(t, "HYPE:0x96", action["token"], "Asset-transfer token should use metadata casing")
	assert.Equal(t, "1.25", action["amount"], "Asset-transfer amount should use wire formatting")
	assert.Equal(t, testVaultAddress, action["fromSubAccount"], "Asset transfer should identify its owned subaccount source")
	assert.Equal(t, float64(nonce), action["nonce"], "Asset-transfer action nonce should match the outer nonce")
}

func TestSendCoreUSDC(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.SendCoreUSDC(t.Context(), "invalid", 1)
	require.ErrorIs(t, err, errInvalidAddress, "Invalid USDC-send destination must return the expected error")
	_, err = ex.SendCoreUSDC(t.Context(), testOtherAddress, 0)
	require.ErrorIs(t, err, errTransferAmountInvalid, "Invalid USDC-send amount must return the expected error")
	_, err = ex.SendCoreUSDC(t.Context(), testOtherAddress, 1)
	require.Error(t, err, "USDC send without credentials must error")

	subAccount := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress,
	}, nil, nil)
	_, err = subAccount.SendCoreUSDC(t.Context(), testOtherAddress, 1)
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "USDC send with a configured subaccount must be rejected")

	var captured signedActionRequest
	success := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey,
	}, nil, &captured)
	nonce, err := success.SendCoreUSDC(t.Context(), strings.ToUpper(testOtherAddress), 1.5)
	require.NoError(t, err, "Valid Core USDC send must not error")
	action := getCapturedAction(t, &captured)
	assert.Equal(t, "usdSend", action["type"], "USDC-send action type should match")
	assert.Equal(t, testOtherAddress, action["destination"], "USDC-send destination should be normalised")
	assert.Equal(t, "1.5", action["amount"], "USDC-send amount should use wire formatting")
	assert.Equal(t, float64(nonce), action["time"], "USDC-send time should match the outer nonce")
}

func TestSendCoreSpot(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.SendCoreSpot(t.Context(), "invalid", "HYPE:0x96", 1)
	require.ErrorIs(t, err, errInvalidAddress, "Invalid spot-send destination must return the expected error")
	_, err = ex.SendCoreSpot(t.Context(), testOtherAddress, "HYPE:0x96", 0)
	require.ErrorIs(t, err, errTransferAmountInvalid, "Invalid spot-send amount must return the expected error")
	_, err = ex.SendCoreSpot(t.Context(), testOtherAddress, "HYPE:0x96", 1)
	require.Error(t, err, "Spot send without credentials must error")

	subAccount := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress,
	}, nil, nil)
	_, err = subAccount.SendCoreSpot(t.Context(), testOtherAddress, "HYPE:0x96", 1)
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "Spot send with a configured subaccount must be rejected")

	missingToken := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey,
	}, map[string]string{"spotMeta": `{"tokens":[]}`}, nil)
	_, err = missingToken.SendCoreSpot(t.Context(), testOtherAddress, "HYPE:0x96", 1)
	require.ErrorIs(t, err, errTransferTokenInvalid, "Unknown spot-send token must return the expected error")

	var captured signedActionRequest
	success := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey,
	}, map[string]string{"spotMeta": spotMetadataJSON}, &captured)
	nonce, err := success.SendCoreSpot(t.Context(), testOtherAddress, "HYPE:0X96", 2)
	require.NoError(t, err, "Valid Core spot send must not error")
	action := getCapturedAction(t, &captured)
	assert.Equal(t, "spotSend", action["type"], "Spot-send action type should match")
	assert.Equal(t, "HYPE:0x96", action["token"], "Spot-send token should use metadata casing")
	assert.Equal(t, "2", action["amount"], "Spot-send amount should use wire formatting")
	assert.Equal(t, float64(nonce), action["time"], "Spot-send time should match the outer nonce")

	_, err = success.SendCoreSpot(t.Context(), testOtherAddress, "USDC", 1)
	require.NoError(t, err, "Valid Core spot USDC send must not error")
	assert.Equal(t, "USDC:0x0", getCapturedAction(t, &captured)["token"], "Core spot USDC should use its full token identifier")
}

func TestWithdrawFromBridge(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.WithdrawFromBridge(t.Context(), "invalid", 1)
	require.ErrorIs(t, err, errInvalidAddress, "Invalid bridge-withdrawal destination must return the expected error")
	_, err = ex.WithdrawFromBridge(t.Context(), testOtherAddress, 0)
	require.ErrorIs(t, err, errTransferAmountInvalid, "Invalid bridge-withdrawal amount must return the expected error")
	_, err = ex.WithdrawFromBridge(t.Context(), testOtherAddress, 1)
	require.Error(t, err, "Bridge withdrawal without credentials must error")

	subAccount := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey, SubAccount: testVaultAddress,
	}, nil, nil)
	_, err = subAccount.WithdrawFromBridge(t.Context(), testOtherAddress, 1)
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "Bridge withdrawal with a configured subaccount must be rejected")

	var captured signedActionRequest
	success := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey,
	}, nil, &captured)
	nonce, err := success.WithdrawFromBridge(t.Context(), strings.ToUpper(testOtherAddress), 2)
	require.NoError(t, err, "Valid bridge withdrawal must not error")
	action := getCapturedAction(t, &captured)
	assert.Equal(t, "withdraw3", action["type"], "Bridge-withdrawal action type should match")
	assert.Equal(t, testOtherAddress, action["destination"], "Bridge-withdrawal destination should be normalised")
	assert.Equal(t, "2", action["amount"], "Bridge-withdrawal amount should use wire formatting")
	assert.Equal(t, float64(nonce), action["time"], "Bridge-withdrawal time should match the outer nonce")
}
