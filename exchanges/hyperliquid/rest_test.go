package hyperliquid

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Live tests of actions that need an operator's own value skip unless it is set; please supply your own
const (
	// liveTransferDestination is an address you control, which receives the funds of live transfer and withdrawal tests
	liveTransferDestination = ""
	// liveGossipPriorityIP is your node's public IP address, which live gossip priority bids are placed for
	liveGossipPriorityIP = ""
	// liveEVMDataRecipient is a HyperEVM address you control, which live sendToEvmWithData tests send 1 USDC to through
	// the coreReceiveWithData function of USDC's linked contract
	liveEVMDataRecipient = ""
	// liveVaultDepositAddress is a vault you accept leaving 5 USDC in, which live vault transfer tests deposit into;
	// Hyperliquid locks a deposit for the vault's lock-up period from the latest deposit, 4 days for HLP
	liveVaultDepositAddress = ""
)

const (
	// canManipulateRealOrders lets live tests change the account the credentials act for. They place and cancel orders
	// that cannot fill (small limits far from the market, a reduce-only TWAP and a trailing stop that never activates),
	// buy about $11 of BTC at market and sell it back, paying taker fees, cancel the test pair's open orders only when
	// the account holds none of its own, clear any scheduled cancel, set the account's current leverage, margin and
	// abstraction settings again, and move funds between its own balances and positions, outcome tokens and, 1000 quote
	// tokens at a time, a HIP-3 backstop liquidator. Some actions last: 0.01 HYPE staked, 0.01 HYPE unstaked through the
	// 7 day queue, 0.01 HYPE delegated to a validator for at least a day, a 1 hour API wallet for a discarded key, 0.0005
	// USDC of request weight, any unclaimed referral rewards claimed, DEX abstraction enabled for good if it was never
	// set, and a 2 USDC bridge withdrawal to the account's own Arbitrum address, less the 1 USDC fee. Funds only reach
	// other addresses and vaults, and gossip priority is only bid for, when the live constants above are set
	canManipulateRealOrders = false
	// testPrivateKey is the Python SDK's public signing test key, which mock mode signs with; testAccountAddress is its
	// account address
	testPrivateKey     = "0x0123456789012345678901234567890123456789012345678901234567890123"
	testAccountAddress = "0x14791697260e4c9a71f18484c9f997b308e59325"
	// testVaultAddress is the HLP vault and testTraderAddress its first child vault; both are public vaults used by the
	// documentation's examples, so live tests can query their account data without credentials
	testVaultAddress  = "0xdfc24b077bc1425ad1dea75bcb6f8158e10df303"
	testTraderAddress = "0x010461c14e146ac35fe42271bdc1134ee31c703a"
	// testVaultLeaderAddress leads the HLP vault, and testFoundationValidatorAddress is the Hyper Foundation 1 validator;
	// live tests query their vault equities, referral, portfolio and staking data, which the mock account lacks on the
	// venue
	testVaultLeaderAddress         = "0x677d831aef5328190852e24f13c46cac05f984e7"
	testFoundationValidatorAddress = "0x5ac99df645f3414876c816caa18b2d234024b487"
	// testAgentAddress is the address of testAgentPrivateKey, an API wallet of testAccountAddress in the mock responses
	testAgentAddress  = "0xed2ff3513c0f08c3ff181b1732745e256c97ce13"
	testClientOrderID = "0x000000000000000000065d27f23d749f"
	testOrderID       = 566563303535
	// Fixed mock time range, which the recorded requests use; live tests use the hours before now
	mockStartTime = 1791252000000
	mockEndTime   = 1791273600000
	// mockUserSignedNonce is the nonce of the recorded user-signed actions, which sign their nonce inside the action
	mockUserSignedNonce = 4102444800000
)

var (
	e             *Exchange
	perpetualPair = currency.NewBTCUSDC()
	spotPair      = currency.NewPair(currency.HYPE, currency.USDC)
)

// getTime returns the mock time range, or the six whole hours before now when testing live
func getTime() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(mockStartTime), time.UnixMilli(mockEndTime)
	}
	end = time.Now().Truncate(time.Hour)
	return end.Add(-6 * time.Hour), end
}

// milli converts a Unix millisecond timestamp, as the info endpoint sends them, into a types.Time
func milli(timestamp int64) types.Time {
	return types.Time(time.UnixMilli(timestamp))
}

// newTestServerExchange returns an exchange whose REST requests the handler serves, for responses the mock recording
// cannot express, such as malformed payloads and errors
func newTestServerExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	server := httptest.NewTestServer(t, handler)
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	return ex
}

// newResponseServerExchange returns an exchange whose REST requests all receive one fixed response body
func newResponseServerExchange(t *testing.T, body string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

// newUnavailableServerExchange returns an exchange signing as the mock signing account whose requests all fail
func newUnavailableServerExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
	return ex
}

// setTestCredentials enables authenticated support with the given credentials
func setTestCredentials(ex *Exchange, credentials *accounts.Credentials) {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(credentials)
}

// testOpenOrder is the open order the mock responses return for testTraderAddress
func testOpenOrder() FrontendOpenOrder {
	return FrontendOpenOrder{
		Coin:             "BTC",
		Side:             "B",
		LimitPrice:       85000,
		Size:             0.001,
		OrderID:          testOrderID,
		Timestamp:        milli(1791275926947),
		OriginalSize:     0.002,
		ClientOrderID:    testClientOrderID,
		TriggerCondition: "N/A",
		Children: []FrontendOpenOrder{{
			Coin:             "BTC",
			Side:             "A",
			LimitPrice:       90000,
			Size:             0.001,
			OrderID:          566563303536,
			Timestamp:        milli(1791275926947),
			OriginalSize:     0.001,
			TriggerCondition: "Price above 90000",
			IsTrigger:        true,
			TriggerPrice:     90000,
			Children:         []FrontendOpenOrder{},
			ReduceOnly:       true,
			OrderType:        "Take Profit Market",
		}},
		OrderType:   "Limit",
		TimeInForce: TimeInForceGTC,
	}
}

// testPositionTPSLOrder is the position TP/SL order the mock responses return for testTraderAddress, which Hyperliquid
// lists without a size, as it closes the whole position
func testPositionTPSLOrder() FrontendOpenOrder {
	return FrontendOpenOrder{
		Coin:             "BTC",
		Side:             "A",
		LimitPrice:       72000,
		OrderID:          566563303544,
		Timestamp:        milli(1791275926947),
		TriggerCondition: "Price below 80000",
		IsTrigger:        true,
		TriggerPrice:     80000,
		Children:         []FrontendOpenOrder{},
		IsPositionTPSL:   true,
		ReduceOnly:       true,
		OrderType:        "Stop Market",
	}
}

// testTakerFill is the mock responses' fill with every optional field set
func testTakerFill() Fill {
	return Fill{
		Coin:          "HYPE",
		Price:         92.618,
		Size:          0.14,
		Side:          "B",
		Time:          milli(1791263098221),
		StartPosition: -126.54,
		Direction:     "Close Short",
		ClosedPNL:     0.263368,
		Hash:          "0x278d87a9293512d629070445f50a3502060d0101c43831a8cb5632fbe838ecc0",
		OrderID:       566598358854,
		Crossed:       true,
		Fee:           0.018567,
		TradeID:       803925706272831,
		ClientOrderID: "0x59a57a0cd0915cb026713c7c048e511d",
		Liquidation:   &FillLiquidation{LiquidatedUser: "0x0000000000000000000000000000000000000015", MarkPrice: 92.6, Method: "backstop"},
		FeeToken:      currency.USDC,
		BuilderFee:    0.012966,
		TWAPID:        2288797,
	}
}

// testMakerFill is the mock responses' fill without optional fields, which earns a rebate
func testMakerFill() Fill {
	return Fill{
		Coin:          "BTC",
		Price:         85791.2407407407,
		Size:          0.00054,
		Side:          "A",
		Time:          milli(1791263098100),
		StartPosition: 0.12,
		Direction:     "Close Long",
		ClosedPNL:     -0.012,
		Hash:          "0x0000000000000000000000000000000000000000000000000000000000000000",
		OrderID:       566598358700,
		Fee:           -0.000463,
		TradeID:       803925706272800,
		FeeToken:      currency.USDC,
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "SendHTTPRequest should send a POST request")
		assert.Equal(t, "/info", r.URL.Path, "SendHTTPRequest should request the info endpoint")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "SendHTTPRequest should send a JSON body")
		var body InfoRequest
		if assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			assert.Equal(t, InfoRequest{Type: "allMids", DEX: "xyz"}, body, "SendHTTPRequest should send the payload")
		}
		_, err := w.Write([]byte(`{"xyz:XYZ100":"31122.5","xyz:TSLA":"380.815"}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result map[string]types.Number
	require.NoError(t, ex.SendHTTPRequest(t.Context(), infoLightEPL, &InfoRequest{Type: "allMids", DEX: "xyz"}, &result), "SendHTTPRequest must not error")
	assert.Equal(t, map[string]types.Number{"xyz:XYZ100": 31122.5, "xyz:TSLA": 380.815}, result, "SendHTTPRequest should decode the response")

	assert.Error(t, e.SendHTTPRequest(t.Context(), infoStandardEPL, make(chan int), nil), "SendHTTPRequest should error for a payload that cannot be marshalled")

	ex = new(Exchange)
	ex.SetDefaults()
	ex.API.Endpoints = ex.NewEndpoints()
	err := ex.SendHTTPRequest(t.Context(), infoStandardEPL, &InfoRequest{Type: "meta"}, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error without an endpoint")
}

func TestSendExchangeRequest(t *testing.T) {
	t.Parallel()
	payload := &SignedActionRequest{Action: CancelAction{Type: "cancel"}, Nonce: 1}
	for _, tc := range []struct {
		body    string
		exp     json.RawMessage
		err     error
		message string
	}{
		{body: `{"status":"ok","response":{"type":"default"}}`, exp: json.RawMessage(`{"type":"default"}`)},
		{body: `{"status":"err","response":"User or API Wallet does not exist."}`, err: errActionResponse, message: "User or API Wallet does not exist."},
		{body: `{"status":"err","response":{"reason":"rejected"}}`, err: errActionResponse, message: `{"reason":"rejected"}`},
		{body: `{"status":"err","response":null}`, err: errActionResponse, message: "exchange action failed: err"},
		{body: `null`, err: common.ErrNoResponse},
	} {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method, "sendExchangeRequest should send a POST request")
			assert.Equal(t, "/exchange", r.URL.Path, "sendExchangeRequest should request the exchange endpoint")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "sendExchangeRequest should send a JSON body")
			_, err := w.Write([]byte(tc.body))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		response, err := ex.sendExchangeRequest(t.Context(), exchangeActionEndpointLimit(1), payload)
		require.ErrorIsf(t, err, tc.err, "sendExchangeRequest must return the expected error for %s", tc.body)
		if tc.message != "" {
			assert.ErrorContainsf(t, err, tc.message, "sendExchangeRequest should report the rejection for %s", tc.body)
		}
		assert.Equalf(t, tc.exp, response, "sendExchangeRequest should return the response payload for %s", tc.body)
	}

	_, err := newUnavailableServerExchange(t).sendExchangeRequest(t.Context(), exchangeActionEndpointLimit(1), payload)
	assert.ErrorIs(t, err, request.ErrBadStatus, "sendExchangeRequest should return a request failure")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "sendExchangeRequest should report a request failure as an authentication failure")

	_, err = e.sendExchangeRequest(t.Context(), exchangeActionEndpointLimit(1), &SignedActionRequest{Action: make(chan int)})
	assert.Error(t, err, "sendExchangeRequest should error for a payload that cannot be marshalled")
	ex := new(Exchange)
	ex.SetDefaults()
	ex.API.Endpoints = ex.NewEndpoints()
	_, err = ex.sendExchangeRequest(t.Context(), exchangeActionEndpointLimit(1), payload)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "sendExchangeRequest should error without an endpoint")
}

func TestActionResponseData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		body           string
		exp            json.RawMessage
		err            error
		signerRejected bool
		message        string
	}{
		{body: `null`, err: common.ErrNoResponse},
		{body: `{"status":"ok","response":{"type":"cancel","data":{"statuses":["success"]}}}`, exp: json.RawMessage(`{"type":"cancel","data":{"statuses":["success"]}}`)},
		{body: `{"status":"err","response":"Order has invalid size."}`, err: errActionResponse, message: "exchange action failed: Order has invalid size."},
		{body: `{"status":"err","response":{"reason":"rejected"}}`, err: errActionResponse, message: `exchange action failed: {"reason":"rejected"}`},
		{body: `{"status":"err","response":null}`, err: errActionResponse, message: "exchange action failed: err"},
		{body: `{"status":"err","response":""}`, err: errActionResponse, message: "exchange action failed: err"},
		{body: `{"status":"err","response":"null"}`, err: errActionResponse, message: "exchange action failed: err"},
		{body: `{"status":"err"}`, err: errActionResponse, message: "exchange action failed: err"},
		{
			body:           `{"status":"err","response":"L1 error: User or API Wallet 0x0123456789abcdef0123456789abcdef01234567 does not exist."}`,
			err:            errActionResponse,
			signerRejected: true,
			message:        "exchange action failed: exchange rejected the signer: L1 error: User or API Wallet 0x0123456789abcdef0123456789abcdef01234567 does not exist.",
		},
		{
			body:           `{"status":"err","response":"Must deposit before performing actions. User: 0x0123456789abcdef0123456789abcdef01234567"}`,
			err:            errActionResponse,
			signerRejected: true,
			message:        "exchange action failed: exchange rejected the signer: Must deposit before performing actions. User: 0x0123456789abcdef0123456789abcdef01234567",
		},
	} {
		var resp *ExchangeActionResponse
		require.NoErrorf(t, json.Unmarshal([]byte(tc.body), &resp), "Unmarshal must not error for %s", tc.body)
		data, err := actionResponseData(resp)
		require.ErrorIsf(t, err, tc.err, "actionResponseData must return the expected error for %s", tc.body)
		if tc.signerRejected {
			assert.ErrorIsf(t, err, errSignerRejected, "actionResponseData should identify the signer rejection %s", tc.body)
		} else {
			assert.NotErrorIsf(t, err, errSignerRejected, "actionResponseData should not treat %s as a signer rejection", tc.body)
		}
		if tc.message != "" {
			assert.EqualErrorf(t, err, tc.message, "actionResponseData should report the rejection of %s", tc.body)
		}
		assert.Equalf(t, tc.exp, data, "actionResponseData should return the accepted action's data for %s", tc.body)
	}
}

func TestSendSignedAction(t *testing.T) {
	t.Parallel()
	action := CancelAction{Type: "cancel", Cancels: []CancelRequest{{Asset: 1, OrderID: 2}}}

	ex := new(Exchange)
	ex.SetDefaults()
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
	_, err := ex.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: maximumActionBatchSize + 1})
	assert.ErrorIs(t, err, errActionBatchTooLarge, "SendSignedAction should reject an oversized batch")
	_, err = ex.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1, ExpiresAfter: time.Now().Add(-time.Second)})
	assert.ErrorIs(t, err, errExpiresAfterPassed, "SendSignedAction should reject a passed expiry before signing")
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress})
	_, err = ex.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1})
	assert.ErrorIs(t, err, errPrivateKeyRequired, "SendSignedAction should require a private key")

	missing, _ := newRoleServerExchange(t, nil, nil, nil)
	setTestCredentials(missing, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
	_, err = missing.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1})
	assert.ErrorIs(t, err, errConfiguredAccountMissing, "SendSignedAction should validate the account before signing")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendSignedAction should report failed authority as an authentication failure")
	assert.Zero(t, missing.lastNonce.Load(), "SendSignedAction should not consume a nonce when authority fails")

	var captured SignedActionRequest
	roles := map[string]string{
		testAccountAddress:    `{"role":"user"}`,
		testSubAccountAddress: `{"role":"subAccount","data":{"master":"` + testAccountAddress + `"}}`,
	}
	accepted, _ := newRoleServerExchange(t, roles, nil, func(w http.ResponseWriter, r *http.Request) {
		captured = SignedActionRequest{}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&captured), "Decoding the signed action should not error") {
			return
		}
		_, err := w.Write([]byte(`{"status":"ok","response":{"type":"cancel","data":{"statuses":["success"]}}}`))
		assert.NoError(t, err, "Writing the action response should not error")
	})
	setTestCredentials(accepted, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey, SubAccount: testSubAccountAddress})
	expiresAfter := time.Now().Add(time.Minute)
	response, err := accepted.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1, ActForVault: true, ExpiresAfter: expiresAfter})
	require.NoError(t, err, "SendSignedAction must not error for an accepted action")
	assert.JSONEq(t, `{"type":"cancel","data":{"statuses":["success"]}}`, string(response), "SendSignedAction should return the response payload")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "SendSignedAction should act for the configured subaccount")
	assert.Equal(t, accepted.lastNonce.Load(), captured.Nonce, "SendSignedAction should send the nonce it consumed")
	require.NotNil(t, captured.ExpiresAfter, "SendSignedAction must send the expiry")
	assert.Equal(t, uint64(expiresAfter.UnixMilli()), *captured.ExpiresAfter, "SendSignedAction should send the expiry in Unix milliseconds")
	exp, err := signL1Action(testPrivateKey, action, testSubAccountAddress, captured.Nonce, captured.ExpiresAfter, true)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, exp, captured.Signature, "SendSignedAction should sign the vault and expiry it sends")

	var boundNonce uint64
	_, err = accepted.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1, SetNonce: func(nonce uint64) { boundNonce = nonce }})
	require.NoError(t, err, "SendSignedAction must not error without the vault")
	assert.Empty(t, captured.VaultAddress, "SendSignedAction should not act for the vault unless the action accepts one")
	assert.Nil(t, captured.ExpiresAfter, "SendSignedAction should not send an unset expiry")
	assert.Equal(t, captured.Nonce, boundNonce, "SendSignedAction should pass the action its nonce before signing")
	exp, err = signL1Action(testPrivateKey, action, "", captured.Nonce, nil, true)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, exp, captured.Signature, "SendSignedAction should sign without a vault")

	lastNonce := accepted.lastNonce.Load()
	_, err = accepted.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1, Nonce: 1700000000000})
	require.NoError(t, err, "SendSignedAction must not error with an explicit nonce")
	assert.Equal(t, uint64(1700000000000), captured.Nonce, "SendSignedAction should sign with the explicit nonce")
	assert.Equal(t, lastNonce, accepted.lastNonce.Load(), "SendSignedAction should not consume a nonce when given one")

	accepted.Config.UseSandbox = true
	_, err = accepted.SendSignedAction(t.Context(), action, nil)
	require.NoError(t, err, "SendSignedAction must not error in sandbox mode")
	exp, err = signL1Action(testPrivateKey, action, "", captured.Nonce, nil, false)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, exp, captured.Signature, "SendSignedAction should sign for testnet in sandbox mode")

	captured = SignedActionRequest{}
	_, err = accepted.SendSignedAction(t.Context(), make(chan int), nil)
	assert.Error(t, err, "SendSignedAction should error for an action that cannot be signed")
	assert.Zero(t, captured.Nonce, "SendSignedAction should not send an action that cannot be signed")

	accepted.Websocket.Conn = nil
	_, err = accepted.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1, Websocket: true})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "SendSignedAction should post the action over the websocket when asked")
	assert.Zero(t, captured.Nonce, "SendSignedAction should not send a websocket action to the exchange endpoint")

	for _, tc := range []struct {
		response            string
		signerRejected      bool
		expAuthorityRetains bool
	}{
		{response: `{"status":"err","response":"Insufficient margin."}`, expAuthorityRetains: true},
		{response: `{"status":"err","response":"L1 error: User or API Wallet 0x0123456789abcdef0123456789abcdef01234567 does not exist."}`, signerRejected: true},
		{response: `{"status":"err","response":"Must deposit before performing actions. User: 0x0123456789abcdef0123456789abcdef01234567"}`, signerRejected: true},
	} {
		var actionRequests atomic.Int32
		rejected, _ := newRoleServerExchange(t, map[string]string{testAccountAddress: `{"role":"user"}`}, nil, func(w http.ResponseWriter, _ *http.Request) {
			actionRequests.Add(1)
			_, err := w.Write([]byte(tc.response))
			assert.NoError(t, err, "Writing the action response should not error")
		})
		setTestCredentials(rejected, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
		_, err = rejected.SendSignedAction(t.Context(), action, &L1ActionOptions{BatchLength: 1})
		assert.ErrorIsf(t, err, errActionResponse, "SendSignedAction should report the rejected action %s", tc.response)
		if tc.signerRejected {
			assert.ErrorIsf(t, err, errSignerRejected, "SendSignedAction should identify the signer rejection %s", tc.response)
		} else {
			assert.NotErrorIsf(t, err, errSignerRejected, "SendSignedAction should not treat %s as a signer rejection", tc.response)
		}
		assert.Equalf(t, tc.expAuthorityRetains, authorityCached(t, rejected), "SendSignedAction should revalidate authority only after a signer rejection, not after %s", tc.response)
		assert.Equalf(t, int32(1), actionRequests.Load(), "SendSignedAction should send the action once for %s", tc.response)
	}
}

func TestGetAllMids(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllMids(t.Context(), "")
	require.NoError(t, err, "GetAllMids must not error")
	if mockTests {
		exp := map[string]types.Number{"BTC": 85812.5, "ETH": 2713.25, "HPOS": 0.04515, "PURR/USDC": 0.150495, "@107": 93.141, "@230": 1, "#10": 0.515}
		assert.Equal(t, exp, result, "GetAllMids should decode every mid price")
	} else {
		assert.NotZero(t, result["BTC"], "GetAllMids should include BTC")
	}

	result, err = e.GetAllMids(t.Context(), "xyz")
	require.NoError(t, err, "GetAllMids must not error for a builder DEX")
	if mockTests {
		assert.Equal(t, map[string]types.Number{"xyz:XYZ100": 31122.5, "xyz:TSLA": 380.815}, result, "GetAllMids should decode the builder DEX's mid prices")
	} else {
		assert.NotEmpty(t, result, "GetAllMids should return the builder DEX's mid prices")
	}

	_, err = newResponseServerExchange(t, `null`).GetAllMids(t.Context(), "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAllMids should reject a null response")
	_, err = newUnavailableServerExchange(t).GetAllMids(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAllMids should return a request failure")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), "invalid", "")
	require.ErrorIs(t, err, errInvalidAddress, "GetOpenOrders must reject an invalid address")

	for _, tc := range []struct {
		dex string
		exp []BasicOrder
	}{
		{
			exp: []BasicOrder{
				{Coin: "PURR/USDC", Side: "A", LimitPrice: 0.14922, Size: 131, OrderID: 566598803191, Timestamp: milli(1791279137528), OriginalSize: 1171, ClientOrderID: "0x00000000000000000b831bfccf6ef32d"},
				{Coin: "BTC", Side: "B", LimitPrice: 85700, Size: 0.05, OrderID: 566598803180, Timestamp: milli(1791279137001), OriginalSize: 0.05},
			},
		},
		{
			dex: "xyz",
			exp: []BasicOrder{{Coin: "xyz:XYZ100", Side: "B", LimitPrice: 31100, Size: 0.25, OrderID: 566598803100, Timestamp: milli(1791279130000), OriginalSize: 0.3, ClientOrderID: "0x00000000000000000b831bfccf6ef300"}},
		},
	} {
		result, err := e.GetOpenOrders(t.Context(), testTraderAddress, tc.dex)
		require.NoErrorf(t, err, "GetOpenOrders must not error for DEX %q", tc.dex)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetOpenOrders should decode every order field for DEX %q", tc.dex)
		} else if len(result) > 0 {
			assert.NotZerof(t, result[0].OrderID, "GetOpenOrders should return order IDs for DEX %q", tc.dex)
		}
	}
}

func TestGetFrontendOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFrontendOpenOrders(t.Context(), "invalid", "")
	require.ErrorIs(t, err, errInvalidAddress, "GetFrontendOpenOrders must reject an invalid address")

	result, err := e.GetFrontendOpenOrders(t.Context(), testTraderAddress, "")
	require.NoError(t, err, "GetFrontendOpenOrders must not error")
	if mockTests {
		assert.Equal(t, []FrontendOpenOrder{testOpenOrder(), testPositionTPSLOrder()}, result, "GetFrontendOpenOrders should decode every order field")
	} else if len(result) > 0 {
		assert.NotZero(t, result[0].OrderID, "GetFrontendOpenOrders should return order IDs")
	}
}

func TestGetUserFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserFills(t.Context(), "invalid", false)
	require.ErrorIs(t, err, errInvalidAddress, "GetUserFills must reject an invalid address")

	for _, tc := range []struct {
		aggregateByTime bool
		exp             []Fill
	}{
		{exp: []Fill{testTakerFill(), testMakerFill()}},
		{aggregateByTime: true, exp: []Fill{testTakerFill()}},
	} {
		result, err := e.GetUserFills(t.Context(), testTraderAddress, tc.aggregateByTime)
		require.NoErrorf(t, err, "GetUserFills must not error when aggregating by time is %t", tc.aggregateByTime)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetUserFills should decode every fill field when aggregating by time is %t", tc.aggregateByTime)
		} else {
			assert.NotEmptyf(t, result, "GetUserFills should return the trader's fills when aggregating by time is %t", tc.aggregateByTime)
		}
	}
}

func TestGetUserFillsByTime(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	for _, tc := range []struct {
		arg *UserFillsByTimeRequest
		err error
	}{
		{err: common.ErrNilPointer},
		{arg: &UserFillsByTimeRequest{User: "invalid", StartTime: start}, err: errInvalidAddress},
		{arg: &UserFillsByTimeRequest{User: testTraderAddress}, err: common.ErrDateUnset},
		{arg: &UserFillsByTimeRequest{User: testTraderAddress, StartTime: time.Unix(0, 0)}, err: common.ErrDateUnset},
		{arg: &UserFillsByTimeRequest{User: testTraderAddress, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetUserFillsByTime(t.Context(), tc.arg)
		require.ErrorIsf(t, err, tc.err, "GetUserFillsByTime must reject %+v", tc.arg)
	}

	for _, arg := range []*UserFillsByTimeRequest{
		{User: testTraderAddress, StartTime: start, EndTime: end, AggregateByTime: true},
		{User: testTraderAddress, StartTime: start},
	} {
		result, err := e.GetUserFillsByTime(t.Context(), arg)
		require.NoErrorf(t, err, "GetUserFillsByTime must not error for %+v", arg)
		if mockTests {
			assert.Equalf(t, []Fill{testMakerFill(), testTakerFill()}, result, "GetUserFillsByTime should decode every fill, oldest first, for %+v", arg)
		} else if len(result) > 1 {
			assert.Falsef(t, result[1].Time.Time().Before(result[0].Time.Time()), "GetUserFillsByTime should return fills oldest first for %+v", arg)
		}
	}
}

func TestGetUserRateLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserRateLimit(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserRateLimit must reject an invalid address")

	result, err := e.GetUserRateLimit(t.Context(), testTraderAddress)
	require.NoError(t, err, "GetUserRateLimit must not error")
	if mockTests {
		exp := &UserRateLimitResponse{CumulativeVolume: 193560424863.1700134277, NumberOfRequestsUsed: 59044382253, NumberOfRequestsCap: 193560434863, NumberOfRequestsSurplus: 5000}
		assert.Equal(t, exp, result, "GetUserRateLimit should decode every field")
	} else {
		assert.GreaterOrEqual(t, result.NumberOfRequestsCap, uint64(10000), "GetUserRateLimit should include the initial request buffer")
	}

	_, err = newResponseServerExchange(t, `null`).GetUserRateLimit(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserRateLimit should reject a null response")
	_, err = newUnavailableServerExchange(t).GetUserRateLimit(t.Context(), testTraderAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserRateLimit should return a request failure")
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		arg *OrderStatusRequest
		err error
	}{
		{err: common.ErrNilPointer},
		{arg: &OrderStatusRequest{User: "invalid", OrderID: 1}, err: errInvalidAddress},
		{arg: &OrderStatusRequest{User: testTraderAddress}, err: order.ErrOrderIDNotSet},
		{arg: &OrderStatusRequest{User: testTraderAddress, OrderID: 1, ClientOrderID: testClientOrderID}, err: errOrderIdentifiersConflict},
		{arg: &OrderStatusRequest{User: testTraderAddress, ClientOrderID: "0x1"}, err: errClientOrderIDInvalid},
	} {
		_, err := e.GetOrderStatus(t.Context(), tc.arg)
		require.ErrorIsf(t, err, tc.err, "GetOrderStatus must reject %+v", tc.arg)
	}

	for _, arg := range []*OrderStatusRequest{
		{User: testTraderAddress, OrderID: testOrderID},
		{User: testTraderAddress, ClientOrderID: testClientOrderID},
	} {
		result, err := e.GetOrderStatus(t.Context(), arg)
		require.NoErrorf(t, err, "GetOrderStatus must not error for %+v", arg)
		if mockTests {
			exp := &OrderStatusResponse{Status: "order", Order: &HistoricalOrder{Order: testOpenOrder(), Status: "open", StatusTimestamp: milli(1791275930650)}}
			assert.Equalf(t, exp, result, "GetOrderStatus should decode the order for %+v", arg)
		} else {
			assert.Containsf(t, []string{"order", "unknownOid"}, result.Status, "GetOrderStatus should return a known status for %+v", arg)
		}
	}

	result, err := e.GetOrderStatus(t.Context(), &OrderStatusRequest{User: testTraderAddress, OrderID: 1})
	require.NoError(t, err, "GetOrderStatus must not error for an unknown order")
	assert.Equal(t, &OrderStatusResponse{Status: "unknownOid"}, result, "GetOrderStatus should report an unknown order")

	_, err = newResponseServerExchange(t, `null`).GetOrderStatus(t.Context(), &OrderStatusRequest{User: testTraderAddress, OrderID: 1})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderStatus should reject a null response")
	_, err = newUnavailableServerExchange(t).GetOrderStatus(t.Context(), &OrderStatusRequest{User: testTraderAddress, OrderID: 1})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderStatus should return a request failure")
}

func TestGetL2Book(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		arg *L2BookRequest
		err error
	}{
		{err: common.ErrNilPointer},
		{arg: &L2BookRequest{Coin: " "}, err: errCoinRequired},
		{arg: &L2BookRequest{Coin: "BTC", NumberOfSignificantFigures: 1}, err: errInvalidSignificantFigures},
		{arg: &L2BookRequest{Coin: "BTC", NumberOfSignificantFigures: 4, Mantissa: 2}, err: errInvalidMantissa},
		{arg: &L2BookRequest{Coin: "BTC", NumberOfSignificantFigures: 5, Mantissa: 1}, err: errInvalidMantissa},
		{arg: &L2BookRequest{Coin: "BTC", NumberOfSignificantFigures: 5, Mantissa: 3}, err: errInvalidMantissa},
	} {
		_, err := e.GetL2Book(t.Context(), tc.arg)
		require.ErrorIsf(t, err, tc.err, "GetL2Book must reject %+v", tc.arg)
	}

	result, err := e.GetL2Book(t.Context(), &L2BookRequest{Coin: "BTC"})
	require.NoError(t, err, "GetL2Book must not error")
	if mockTests {
		exp := &L2BookResponse{
			Coin: "BTC",
			Time: milli(1791275808695),
			Levels: [][]L2Level{
				{{Price: 85812, Size: 0.41043, NumberOfOrders: 4}, {Price: 85811, Size: 1.62719, NumberOfOrders: 7}},
				{{Price: 85813, Size: 4.71207, NumberOfOrders: 19}, {Price: 85814, Size: 0.0035, NumberOfOrders: 1}},
			},
		}
		assert.Equal(t, exp, result, "GetL2Book should decode every level")
	} else {
		assert.NotEmpty(t, result.Levels[0], "GetL2Book should return bids")
	}

	result, err = e.GetL2Book(t.Context(), &L2BookRequest{Coin: "BTC", NumberOfSignificantFigures: 5, Mantissa: 2})
	require.NoError(t, err, "GetL2Book must not error for an aggregated book")
	if mockTests {
		exp := &L2BookResponse{
			Coin:   "BTC",
			Time:   milli(1791275810375),
			Levels: [][]L2Level{{{Price: 85812, Size: 0.09813, NumberOfOrders: 4}}, {{Price: 85814, Size: 4.7155, NumberOfOrders: 20}}},
			Spread: 1,
		}
		assert.Equal(t, exp, result, "GetL2Book should decode the aggregated book's spread")
	} else {
		assert.NotZero(t, result.Spread, "GetL2Book should return the aggregated book's spread")
	}

	_, err = newResponseServerExchange(t, `{"coin":"BTC","levels":[[]]}`).GetL2Book(t.Context(), &L2BookRequest{Coin: "BTC"})
	assert.ErrorIs(t, err, errInvalidBookLevelCount, "GetL2Book should reject a book without both sides")
	_, err = newResponseServerExchange(t, `null`).GetL2Book(t.Context(), &L2BookRequest{Coin: "BTC"})
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetL2Book should reject a null response")
	_, err = newUnavailableServerExchange(t).GetL2Book(t.Context(), &L2BookRequest{Coin: "BTC"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetL2Book should return a request failure")
}

func TestValidateBookAggregation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		significantFigures uint64
		mantissa           uint64
		err                error
	}{
		{},
		{significantFigures: 2},
		{significantFigures: 3},
		{significantFigures: 4},
		{significantFigures: 5},
		{significantFigures: 5, mantissa: 2},
		{significantFigures: 5, mantissa: 5},
		{significantFigures: 1, err: errInvalidSignificantFigures},
		{significantFigures: 6, err: errInvalidSignificantFigures},
		{significantFigures: 6, mantissa: 5, err: errInvalidSignificantFigures},
		{mantissa: 2, err: errInvalidMantissa},
		{significantFigures: 4, mantissa: 5, err: errInvalidMantissa},
		{significantFigures: 5, mantissa: 1, err: errInvalidMantissa},
		{significantFigures: 5, mantissa: 3, err: errInvalidMantissa},
		{significantFigures: 3, mantissa: 1, err: errInvalidMantissa},
	} {
		err := validateBookAggregation(tc.significantFigures, tc.mantissa)
		assert.ErrorIsf(t, err, tc.err, "validateBookAggregation should return the expected error for %d significant figures and mantissa %d", tc.significantFigures, tc.mantissa)
	}
}

func TestGetCandleSnapshot(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	for _, tc := range []struct {
		arg *CandleSnapshotRequest
		err error
	}{
		{err: common.ErrNilPointer},
		{arg: &CandleSnapshotRequest{Interval: kline.OneHour, StartTime: start, EndTime: end}, err: errCoinRequired},
		{arg: &CandleSnapshotRequest{Coin: "BTC", Interval: kline.TenMin, StartTime: start, EndTime: end}, err: kline.ErrUnsupportedInterval},
		{arg: &CandleSnapshotRequest{Coin: "BTC", Interval: kline.OneHour, StartTime: time.Unix(0, 0), EndTime: end}, err: common.ErrDateUnset},
		{arg: &CandleSnapshotRequest{Coin: "BTC", Interval: kline.OneHour, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetCandleSnapshot(t.Context(), tc.arg)
		require.ErrorIsf(t, err, tc.err, "GetCandleSnapshot must reject %+v", tc.arg)
	}

	// Only the weight of the candles from the start up to now is defined, so weighing the request any other way fails it
	var requestBody []byte
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		requestBody, err = io.ReadAll(r.Body)
		assert.NoError(t, err, "Reading the request body should not error")
		_, err = w.Write([]byte(`[]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	limited := new(Exchange)
	require.NoError(t, testexch.Setup(limited), "Setup must not error")
	var err error
	limited.Requester, err = request.New(limited.Name, server.Client(), request.WithLimiter(request.RateLimitDefinitions{
		candleEndpointLimit(90): request.NewRateLimitWithWeight(restRateLimitInterval, restRateLimitWeight, 1),
	}))
	require.NoError(t, err, "request.New must not error")
	require.NoError(t, limited.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	recentStart := time.Now().Add(-90 * time.Hour)
	_, err = limited.GetCandleSnapshot(t.Context(), &CandleSnapshotRequest{Coin: "BTC", Interval: kline.OneHour, StartTime: recentStart})
	require.NoError(t, err, "GetCandleSnapshot must weigh a request without an end time by the candles up to now")
	assert.JSONEq(t, `{"type":"candleSnapshot","req":{"coin":"BTC","interval":"1h","startTime":`+strconv.FormatInt(recentStart.UnixMilli(), 10)+`}}`, string(requestBody), "GetCandleSnapshot should omit a zero end time")

	exp := []Candle{
		{OpenTime: milli(1791252000000), CloseTime: milli(1791255599999), Coin: "BTC", Interval: "1h", Open: 85641, Close: 85513, High: 85739, Low: 85433, Volume: 526.02537, NumberOfTrades: 7830},
		{OpenTime: milli(1791255600000), CloseTime: milli(1791259199999), Coin: "BTC", Interval: "1h", Open: 85513, Close: 85690, High: 85702, Low: 85390, Volume: 611.8724, NumberOfTrades: 8104},
	}
	for _, arg := range []*CandleSnapshotRequest{
		{Coin: "BTC", Interval: kline.OneHour, StartTime: start, EndTime: end},
		// Without an end time Hyperliquid returns the candles up to now
		{Coin: "BTC", Interval: kline.OneHour, StartTime: start},
	} {
		result, err := e.GetCandleSnapshot(t.Context(), arg)
		require.NoErrorf(t, err, "GetCandleSnapshot must not error for %+v", arg)
		if mockTests {
			assert.Equalf(t, exp, result, "GetCandleSnapshot should decode every candle field for %+v", arg)
		} else {
			assert.NotEmptyf(t, result, "GetCandleSnapshot should return candles for %+v", arg)
		}
	}
}

func TestGetMaxBuilderFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxBuilderFee(t.Context(), "invalid", "0x0000000000000000000000000000000000000016")
	require.ErrorIs(t, err, errInvalidAddress, "GetMaxBuilderFee must reject an invalid user address")
	_, err = e.GetMaxBuilderFee(t.Context(), testTraderAddress, "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetMaxBuilderFee must reject an invalid builder address")

	result, err := e.GetMaxBuilderFee(t.Context(), testTraderAddress, "0x0000000000000000000000000000000000000016")
	require.NoError(t, err, "GetMaxBuilderFee must not error")
	if mockTests {
		assert.Equal(t, uint64(1000), result, "GetMaxBuilderFee should return the approved fee in tenths of a basis point")
	}

	_, err = newResponseServerExchange(t, `null`).GetMaxBuilderFee(t.Context(), testTraderAddress, "0x0000000000000000000000000000000000000016")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetMaxBuilderFee should reject a null response")
	_, err = newUnavailableServerExchange(t).GetMaxBuilderFee(t.Context(), testTraderAddress, "0x0000000000000000000000000000000000000016")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetMaxBuilderFee should return a request failure")
}

func TestGetHistoricalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalOrders(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetHistoricalOrders must reject an invalid address")

	result, err := e.GetHistoricalOrders(t.Context(), testTraderAddress)
	require.NoError(t, err, "GetHistoricalOrders must not error")
	if mockTests {
		assert.Equal(t, []HistoricalOrder{{Order: testOpenOrder(), Status: "open", StatusTimestamp: milli(1791275930650)}}, result, "GetHistoricalOrders should decode every order field")
	} else {
		assert.NotEmpty(t, result, "GetHistoricalOrders should return the trader's orders")
	}
}

func TestGetUserTWAPSliceFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserTWAPSliceFills(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserTWAPSliceFills must reject an invalid address")

	result, err := e.GetUserTWAPSliceFills(t.Context(), testTraderAddress)
	require.NoError(t, err, "GetUserTWAPSliceFills must not error")
	if mockTests {
		exp := []TWAPSliceFill{{
			Fill: Fill{
				Coin:          "NEAR",
				Price:         5.2062,
				Size:          34.7,
				Side:          "B",
				Time:          milli(1791278901040),
				StartPosition: -269420.5,
				Direction:     "Close Short",
				ClosedPNL:     -27.370319,
				Hash:          "0x0000000000000000000000000000000000000000000000000000000000000000",
				OrderID:       566595262718,
				Crossed:       true,
				Fee:           0.065035,
				TradeID:       86235454845785,
				FeeToken:      currency.USDC,
			},
			TWAPID: 2288797,
		}}
		assert.Equal(t, exp, result, "GetUserTWAPSliceFills should decode the slice fill and its TWAP ID")
	}
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubAccounts(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetSubAccounts must reject an invalid address")

	// No public organisation is known to have subaccounts, so live tests only check that the response decodes
	result, err := e.GetSubAccounts(t.Context(), testAccountAddress)
	require.NoError(t, err, "GetSubAccounts must not error")
	if mockTests {
		exp := []SubAccount{{
			Name:           "pr2_12",
			SubAccountUser: "0x0000000000000000000000000000000000000017",
			Master:         testAccountAddress,
			ClearinghouseState: ClearinghouseStateResponse{
				MarginSummary:              MarginSummary{AccountValue: 16754.533684, TotalNotionalPosition: 46230.3741, TotalRawUSD: 62984.907784, TotalMarginUsed: 4623.03741},
				CrossMarginSummary:         MarginSummary{AccountValue: 15554.033684, TotalNotionalPosition: 40227.8741, TotalRawUSD: 55781.907784, TotalMarginUsed: 4022.78741},
				CrossMaintenanceMarginUsed: 577.879676,
				Withdrawable:               12131.496274,
				AssetPositions: []AssetPosition{{
					Type: "oneWay",
					Position: Position{
						Coin:              "ETH",
						SignedSize:        -2.25,
						Leverage:          Leverage{Type: "isolated", Value: 5, RawUSD: 7203},
						EntryPrice:        2667.5,
						PositionValue:     6002.5,
						UnrealisedPNL:     -0.625,
						ReturnOnEquity:    -0.0005206,
						LiquidationPrice:  3165.3090145624,
						MarginUsed:        1200.5,
						MaxLeverage:       25,
						CumulativeFunding: CumulativeFunding{AllTime: -316.183281, SinceOpen: -1.179902, SinceChange: -0.577939},
					},
				}},
				Time: milli(1791278778100),
			},
			SpotState: SpotClearinghouseStateResponse{Balances: []SpotBalance{{Coin: currency.HYPE, Token: 150, Total: 367.68703816, Hold: 120, EntryNotional: 34398.42600243}}},
		}}
		assert.Equal(t, exp, result, "GetSubAccounts should decode every subaccount field")
	}

	result, err = newResponseServerExchange(t, `null`).GetSubAccounts(t.Context(), testAccountAddress)
	require.NoError(t, err, "GetSubAccounts must not error for an account without subaccounts")
	assert.Empty(t, result, "GetSubAccounts should return no subaccounts for a null response")
}

func TestGetVaultDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetVaultDetails(t.Context(), "invalid", "")
	require.ErrorIs(t, err, errInvalidAddress, "GetVaultDetails must reject an invalid vault address")
	_, err = e.GetVaultDetails(t.Context(), testVaultAddress, "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetVaultDetails must reject an invalid user address")

	follower := &VaultFollower{
		User:           "0x0000000000000000000000000000000000000010",
		VaultEquity:    604894.1647215381,
		PNL:            80759.3609095381,
		AllTimePNL:     156784.7214235381,
		DaysFollowing:  655,
		VaultEntryTime: milli(1734650428028),
		LockupUntil:    milli(1770680880422),
	}
	exp := &VaultDetailsResponse{
		Name:         "Hyperliquidity Provider (HLP)",
		VaultAddress: testVaultAddress,
		Leader:       "0x677d831aef5328190852e24f13c46cac05f984e7",
		Description:  "This community-owned vault provides liquidity to Hyperliquid through multiple market making strategies, performs liquidations, and accrues platform fees.",
		Portfolio: []PortfolioPeriod{
			{Period: "day", History: PortfolioHistory{
				AccountValueHistory: []PortfolioValue{{Time: milli(1791189344472), Value: 181448095.7272160053}},
				PNLHistory:          []PortfolioValue{{Time: milli(1791189344472), Value: 10342.6522}},
				Volume:              152843,
			}},
			{Period: "allTime", History: PortfolioHistory{
				AccountValueHistory: []PortfolioValue{{Time: milli(1734650428028), Value: 155362400}},
				PNLHistory:          []PortfolioValue{{Time: milli(1734650428028), Value: 91224556.29}},
				Volume:              60119255511.06,
			}},
		},
		APR:                   0.036051779465087176,
		FollowerState:         follower,
		LeaderFraction:        0.0019940895074468785,
		LeaderCommission:      0.1,
		Followers:             []VaultFollower{*follower},
		MaxDistributable:      41851855.944809,
		MaxWithdrawable:       742557.680863,
		IsClosed:              true,
		Relationship:          VaultRelationship{Type: "parent", Data: VaultRelationshipData{ChildAddresses: []string{testTraderAddress}}},
		AllowDeposits:         true,
		AlwaysCloseOnWithdraw: true,
	}
	for _, user := range []string{"", testTraderAddress} {
		result, err := e.GetVaultDetails(t.Context(), testVaultAddress, user)
		require.NoErrorf(t, err, "GetVaultDetails must not error for user %q", user)
		if mockTests {
			assert.Equalf(t, exp, result, "GetVaultDetails should decode every field for user %q", user)
		} else {
			assert.Equalf(t, testVaultAddress, result.VaultAddress, "GetVaultDetails should return the vault for user %q", user)
		}
	}

	_, err = newResponseServerExchange(t, `null`).GetVaultDetails(t.Context(), testVaultAddress, "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetVaultDetails should reject a null response")
	_, err = newUnavailableServerExchange(t).GetVaultDetails(t.Context(), testVaultAddress, "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetVaultDetails should return a request failure")
}

func TestGetUserVaultEquities(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserVaultEquities(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserVaultEquities must reject an invalid address")

	result, err := e.GetUserVaultEquities(t.Context(), testVaultLeaderAddress)
	require.NoError(t, err, "GetUserVaultEquities must not error")
	if mockTests {
		exp := []UserVaultEquity{{VaultAddress: testVaultAddress, Equity: 604897.465556, LockedUntilTimestamp: milli(1770680880422)}}
		assert.Equal(t, exp, result, "GetUserVaultEquities should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetUserVaultEquities should return the leader's vault equity")
	}
}

func TestGetUserRole(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserRole(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserRole must reject an invalid address")

	result, err := e.GetUserRole(t.Context(), testVaultAddress)
	require.NoError(t, err, "GetUserRole must not error")
	assert.Equal(t, &UserRoleResponse{Role: "vault"}, result, "GetUserRole should identify the vault")
	// The test agent and subaccount are only linked to the test account in the mock responses
	for _, tc := range []struct {
		user string
		exp  *UserRoleResponse
	}{
		{user: testAgentAddress, exp: &UserRoleResponse{Role: "agent", Data: UserRoleData{User: testAccountAddress}}},
		{user: testSubAccountAddress, exp: &UserRoleResponse{Role: "subAccount", Data: UserRoleData{Master: testAccountAddress}}},
	} {
		result, err := e.GetUserRole(t.Context(), tc.user)
		require.NoErrorf(t, err, "GetUserRole must not error for %s", tc.user)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetUserRole should decode the linked account of %s", tc.user)
		} else {
			assert.Containsf(t, []string{"missing", "user", "agent", "vault", "subAccount"}, result.Role, "GetUserRole should return a known role for %s", tc.user)
		}
	}

	_, err = newResponseServerExchange(t, `null`).GetUserRole(t.Context(), testVaultAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserRole should reject a null response")
	_, err = newUnavailableServerExchange(t).GetUserRole(t.Context(), testVaultAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserRole should return a request failure")
}

func TestGetPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.GetPortfolio(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetPortfolio must reject an invalid address")

	result, err := e.GetPortfolio(t.Context(), testVaultLeaderAddress)
	require.NoError(t, err, "GetPortfolio must not error")
	if mockTests {
		exp := []PortfolioPeriod{
			{Period: "month", History: PortfolioHistory{
				AccountValueHistory: []PortfolioValue{{Time: milli(1734565200011), Value: 50551.370822}},
				PNLHistory:          []PortfolioValue{{Time: milli(1734565200011), Value: 476.765822}},
				Volume:              4396031.2199999997,
			}},
			{Period: "perpDay", History: PortfolioHistory{
				AccountValueHistory: []PortfolioValue{{Time: milli(1791193200017), Value: 52011.58}},
				PNLHistory:          []PortfolioValue{{Time: milli(1791193200017), Value: -12.5}},
				Volume:              1503.25,
			}},
		}
		assert.Equal(t, exp, result, "GetPortfolio should decode every period")
	} else {
		require.NotEmpty(t, result, "GetPortfolio must return the portfolio periods")
		assert.NotEmpty(t, result[0].History.AccountValueHistory, "GetPortfolio should return the first period's account value history")
	}
}

func TestGetReferral(t *testing.T) {
	t.Parallel()
	_, err := e.GetReferral(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetReferral must reject an invalid address")

	usdcTotals := ReferralTokenState{CumulativeVolume: 22552.31, UnclaimedRewards: 403.80768, ClaimedRewards: 19055.21949223, BuilderRewards: 18949.2614927}
	referredUSDCTotals := ReferralStateTokenState{CumulativeVolume: 2663707.7000000002, CumulativeRewardedFeesSinceReferred: 3864.243222, CumulativeFeesRewardedToReferrer: 115.73759}
	for _, tc := range []struct {
		user string
		exp  *ReferralResponse
	}{
		{
			user: testVaultLeaderAddress,
			exp: &ReferralResponse{
				ReferralTokenState: usdcTotals,
				ReferredBy:         &ReferredBy{Referrer: "0x0000000000000000000000000000000000000018", Code: "SEABOND"},
				ReferrerState: ReferrerState{
					Stage: "ready",
					Data: ReferrerStateData{
						Code:              "GCTTEST",
						NumberOfReferrals: 1,
						ReferralStates: []ReferralState{{
							ReferralStateTokenState: referredUSDCTotals,
							TimeJoined:              milli(1785734755702),
							User:                    "0x0000000000000000000000000000000000000019",
							TokenToState: []ReferralStateTokenEntry{
								{State: referredUSDCTotals},
								{Token: 360, State: ReferralStateTokenState{CumulativeVolume: 1520.5, CumulativeRewardedFeesSinceReferred: 0.684225, CumulativeFeesRewardedToReferrer: 0.0684225}},
							},
						}},
					},
				},
				RewardHistory: []ReferralReward{{Earned: 0.24564, Volume: 3541.88913, ReferralVolume: 12.5, Time: milli(1686181525632)}},
				TokenToState: []ReferralTokenStateEntry{
					{State: usdcTotals},
					{Token: 360, State: ReferralTokenState{CumulativeVolume: 1500, UnclaimedRewards: 1.25, ClaimedRewards: 3.5, BuilderRewards: 0.75}},
				},
			},
		},
		{
			user: testTraderAddress,
			exp: &ReferralResponse{
				CumulativeVolume: 1.9,
				ReferrerState:    ReferrerState{Stage: "needToTrade", Data: ReferrerStateData{Required: 9998.1}},
				RewardHistory:    []ReferralReward{},
				TokenToState:     []ReferralTokenStateEntry{{State: ReferralTokenState{CumulativeVolume: 1.9}}},
			},
		},
	} {
		result, err := e.GetReferral(t.Context(), tc.user)
		require.NoErrorf(t, err, "GetReferral must not error for %s", tc.user)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetReferral should decode every field for %s", tc.user)
			continue
		}
		assert.NotEmptyf(t, result.ReferrerState.Stage, "GetReferral should return the referrer stage for %s", tc.user)
		if tc.user == testVaultLeaderAddress {
			assert.NotEmpty(t, result.ReferrerState.Data.ReferralStates, "GetReferral should return the accounts the leader referred")
			assert.NotEmpty(t, result.RewardHistory, "GetReferral should return the leader's legacy rewards")
		}
	}

	_, err = newResponseServerExchange(t, `null`).GetReferral(t.Context(), testVaultLeaderAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetReferral should reject a null response")
	_, err = newUnavailableServerExchange(t).GetReferral(t.Context(), testVaultLeaderAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetReferral should return a request failure")
}

func TestGetUserFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserFees(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserFees must reject an invalid address")

	result, err := e.GetUserFees(t.Context(), testVaultAddress)
	require.NoError(t, err, "GetUserFees must not error")
	if mockTests {
		exp := &UserFeesResponse{
			DailyUserVolume: []DailyUserVolume{{Date: "2026-10-05", UserCross: 15201.6, UserAdd: 9821.33, Exchange: 6311325373.4300003052}},
			FeeSchedule: FeeSchedule{
				Cross:     0.00045,
				Add:       0.00015,
				SpotCross: 0.0007,
				SpotAdd:   0.0004,
				Tiers: FeeTiers{
					VIP:         []VIPFeeTier{{NotionalCutoff: 5000000, Cross: 0.0004, Add: 0.00012, SpotCross: 0.0006, SpotAdd: 0.0003}},
					MarketMaker: []MarketMakerFeeTier{{MakerFractionCutoff: 0.005, Add: -0.00001}},
				},
				ReferralDiscount:     0.04,
				StakingDiscountTiers: []StakingDiscountTier{{BPSOfMaxSupply: 0.0001, Discount: 0.05}},
			},
			UserCrossRate:               0.000315,
			UserAddRate:                 0.000105,
			UserSpotCrossRate:           0.00049,
			UserSpotAddRate:             0.00028,
			ActiveReferralDiscount:      0.04,
			Trial:                       json.RawMessage(`null`),
			FeeTrialEscrow:              1.5,
			NextTrialAvailableTimestamp: milli(1791360000000),
			StakingLink:                 &StakingLink{Type: "tradingUser", StakingUser: "0x0000000000000000000000000000000000000011"},
			ActiveStakingDiscount:       StakingDiscountTier{BPSOfMaxSupply: 4.7577998927, Discount: 0.3},
		}
		assert.Equal(t, exp, result, "GetUserFees should decode every field")
	} else {
		assert.NotZero(t, result.UserCrossRate, "GetUserFees should return the taker rate")
	}

	_, err = newResponseServerExchange(t, `null`).GetUserFees(t.Context(), testVaultAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserFees should reject a null response")
	_, err = newUnavailableServerExchange(t).GetUserFees(t.Context(), testVaultAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserFees should return a request failure")
}

func TestGetDelegations(t *testing.T) {
	t.Parallel()
	_, err := e.GetDelegations(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetDelegations must reject an invalid address")

	result, err := e.GetDelegations(t.Context(), testFoundationValidatorAddress)
	require.NoError(t, err, "GetDelegations must not error")
	if mockTests {
		exp := []Delegation{{Validator: "0x000000000000000000000000000000000000001a", Amount: 10740.27855992, LockedUntilTimestamp: milli(1746044474341)}}
		assert.Equal(t, exp, result, "GetDelegations should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetDelegations should return the validator's self-delegation")
	}
}

func TestGetDelegatorSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetDelegatorSummary(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetDelegatorSummary must reject an invalid address")

	result, err := e.GetDelegatorSummary(t.Context(), testFoundationValidatorAddress)
	require.NoError(t, err, "GetDelegatorSummary must not error")
	if mockTests {
		exp := &DelegatorSummaryResponse{Delegated: 10740.27855992, Undelegated: 44.29025178, TotalPendingWithdrawal: 900, NumberOfPendingWithdrawals: 1}
		assert.Equal(t, exp, result, "GetDelegatorSummary should decode every field")
	} else {
		assert.NotZero(t, result.Delegated, "GetDelegatorSummary should return the validator's delegated stake")
	}

	_, err = newResponseServerExchange(t, `null`).GetDelegatorSummary(t.Context(), testFoundationValidatorAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetDelegatorSummary should reject a null response")
	_, err = newUnavailableServerExchange(t).GetDelegatorSummary(t.Context(), testFoundationValidatorAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetDelegatorSummary should return a request failure")
}

func TestGetDelegatorHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDelegatorHistory(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetDelegatorHistory must reject an invalid address")

	result, err := e.GetDelegatorHistory(t.Context(), testFoundationValidatorAddress)
	require.NoError(t, err, "GetDelegatorHistory must not error")
	if mockTests {
		exp := []DelegatorUpdate{
			{
				Time:  milli(1787860095825),
				Hash:  "0x1dd5a477b2ec41601f4f04431c75e7020573005d4def6032c19e4fca71e01b4a",
				Delta: DelegatorDelta{Withdrawal: &WithdrawalDelta{Amount: 900, Phase: "initiated"}},
			},
			{
				Time:  milli(1787860085543),
				Hash:  "0xbe5a6f43f7e6fb41bfd404431c754f020b3b002992ea1a1362231a96b6ead52c",
				Delta: DelegatorDelta{Delegate: &DelegateDelta{Validator: "0x000000000000000000000000000000000000001a", Amount: 900, IsUndelegate: true}},
			},
			{
				Time:  milli(1745955948735),
				Hash:  "0x2d6cb9002f3ec074bc0404227ee01401a90075a294f67195f3734cc428a99ddb",
				Delta: DelegatorDelta{CDeposit: &CDepositDelta{Amount: 10001}},
			},
		}
		assert.Equal(t, exp, result, "GetDelegatorHistory should decode every staking change")
	} else {
		assert.NotEmpty(t, result, "GetDelegatorHistory should return the validator's staking history")
	}
}

func TestGetDelegatorRewards(t *testing.T) {
	t.Parallel()
	_, err := e.GetDelegatorRewards(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetDelegatorRewards must reject an invalid address")

	result, err := e.GetDelegatorRewards(t.Context(), testFoundationValidatorAddress)
	require.NoError(t, err, "GetDelegatorRewards must not error")
	if mockTests {
		exp := []DelegatorReward{
			{Time: milli(1791244800027), Source: "delegation", TotalAmount: 0.63878098},
			{Time: milli(1791244800027), Source: "commission", TotalAmount: 17.51099385},
		}
		assert.Equal(t, exp, result, "GetDelegatorRewards should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetDelegatorRewards should return the validator's staking rewards")
	}
}

func TestGetUserDEXAbstraction(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserDEXAbstraction(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserDEXAbstraction must reject an invalid address")

	result, err := e.GetUserDEXAbstraction(t.Context(), testAccountAddress)
	require.NoError(t, err, "GetUserDEXAbstraction must not error")
	if mockTests {
		require.NotNil(t, result, "GetUserDEXAbstraction must return a state once set")
		assert.True(t, *result, "GetUserDEXAbstraction should return an enabled state")
	}

	result, err = newResponseServerExchange(t, `false`).GetUserDEXAbstraction(t.Context(), testAccountAddress)
	require.NoError(t, err, "GetUserDEXAbstraction must not error for a disabled state")
	require.NotNil(t, result, "GetUserDEXAbstraction must return a disabled state")
	assert.False(t, *result, "GetUserDEXAbstraction should return a disabled state")
	result, err = newResponseServerExchange(t, `null`).GetUserDEXAbstraction(t.Context(), testAccountAddress)
	require.NoError(t, err, "GetUserDEXAbstraction must not error for a state never set")
	assert.Nil(t, result, "GetUserDEXAbstraction should return no state when it has never been set")
}

func TestGetUserAbstraction(t *testing.T) {
	t.Parallel()
	_, err := e.GetUserAbstraction(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetUserAbstraction must reject an invalid address")

	result, err := e.GetUserAbstraction(t.Context(), testVaultAddress)
	require.NoError(t, err, "GetUserAbstraction must not error")
	assert.Equal(t, AccountAbstractionDefault, result, "GetUserAbstraction should return the vault's mode")

	for _, mode := range []AccountAbstraction{AccountAbstractionDefault, AccountAbstractionDisabled, AccountAbstractionDEXAbstraction, AccountAbstractionUnifiedAccount, AccountAbstractionPortfolioMargin} {
		result, err := newResponseServerExchange(t, `"`+string(mode)+`"`).GetUserAbstraction(t.Context(), testVaultAddress)
		require.NoErrorf(t, err, "GetUserAbstraction must not error for %s", mode)
		assert.Equalf(t, mode, result, "GetUserAbstraction should return %s", mode)
	}
	_, err = newResponseServerExchange(t, `"unknown"`).GetUserAbstraction(t.Context(), testVaultAddress)
	assert.ErrorIs(t, err, errAccountAbstractionInvalid, "GetUserAbstraction should reject an unknown mode")
	_, err = newUnavailableServerExchange(t).GetUserAbstraction(t.Context(), testVaultAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserAbstraction should return a request failure")
}

func TestGetBorrowLendUserState(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowLendUserState(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetBorrowLendUserState must reject an invalid address")

	result, err := e.GetBorrowLendUserState(t.Context(), testAccountAddress)
	require.NoError(t, err, "GetBorrowLendUserState must not error")
	if mockTests {
		exp := &BorrowLendUserStateResponse{
			TokenToState: []BorrowLendTokenStateEntry{{
				Token: 150,
				State: BorrowLendTokenState{Borrow: BorrowLendBalance{Basis: 12.5, Value: 12.51}, Supply: BorrowLendBalance{Basis: 44.69295862, Value: 44.69692314}},
			}},
			Health:       "healthy",
			HealthFactor: 2.5,
		}
		assert.Equal(t, exp, result, "GetBorrowLendUserState should decode every field")
	} else {
		assert.NotEmpty(t, result.Health, "GetBorrowLendUserState should return the account's health")
	}

	_, err = newResponseServerExchange(t, `null`).GetBorrowLendUserState(t.Context(), testAccountAddress)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBorrowLendUserState should reject a null response")
	_, err = newUnavailableServerExchange(t).GetBorrowLendUserState(t.Context(), testAccountAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBorrowLendUserState should return a request failure")
}

func TestGetBorrowLendReserveState(t *testing.T) {
	t.Parallel()
	result, err := e.GetBorrowLendReserveState(t.Context(), 0)
	require.NoError(t, err, "GetBorrowLendReserveState must not error for USDC")
	if mockTests {
		exp := &BorrowLendReserveStateResponse{
			BorrowYearlyRate: 0.05,
			SupplyYearlyRate: 0.0346721009,
			Balance:          124148247.9570675939,
			Utilisation:      0.7704911304,
			OraclePrice:      1,
			LoanToValue:      0.65,
			TotalSupplied:    540929789.4645597935,
			TotalBorrowed:    416781604.9600116611,
		}
		assert.Equal(t, exp, result, "GetBorrowLendReserveState should decode every field")
	} else {
		assert.Positive(t, result.TotalSupplied.Float64(), "GetBorrowLendReserveState should return USDC's supply")
	}

	var body map[string]any
	captured := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		_, err := w.Write([]byte(`null`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	_, err = captured.GetBorrowLendReserveState(t.Context(), 0)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetBorrowLendReserveState should reject a null response")
	assert.Equal(t, map[string]any{"type": "borrowLendReserveState", "token": float64(0)}, body, "GetBorrowLendReserveState should send token 0 rather than omit it")
	_, err = newUnavailableServerExchange(t).GetBorrowLendReserveState(t.Context(), 0)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetBorrowLendReserveState should return a request failure")
}

func TestGetAllBorrowLendReserveStates(t *testing.T) {
	t.Parallel()
	result, err := e.GetAllBorrowLendReserveStates(t.Context())
	require.NoError(t, err, "GetAllBorrowLendReserveStates must not error")
	if mockTests {
		exp := []BorrowLendReserveStateEntry{
			{Token: 150, State: BorrowLendReserveStateResponse{BorrowYearlyRate: 0.05, Balance: 13338366.2448045798, OraclePrice: 93.248, LoanToValue: 0.65, TotalSupplied: 13338366.2448045798}},
			{Token: 268, State: BorrowLendReserveStateResponse{
				BorrowYearlyRate: 0.05,
				SupplyYearlyRate: 0.0231915985,
				Balance:          1340936.4669496501,
				Utilisation:      0.5153688564,
				OraclePrice:      0.99965,
				TotalSupplied:    2766921.6155478801,
				TotalBorrowed:    1425985.2287287901,
			}},
		}
		assert.Equal(t, exp, result, "GetAllBorrowLendReserveStates should decode every reserve")
	} else {
		assert.NotEmpty(t, result, "GetAllBorrowLendReserveStates should return the reserves")
	}
}

func TestGetApprovedBuilders(t *testing.T) {
	t.Parallel()
	_, err := e.GetApprovedBuilders(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetApprovedBuilders must reject an invalid address")

	result, err := e.GetApprovedBuilders(t.Context(), testAccountAddress)
	require.NoError(t, err, "GetApprovedBuilders must not error")
	if mockTests {
		assert.Equal(t, []string{"0x0000000000000000000000000000000000000016"}, result, "GetApprovedBuilders should return the approved builder")
	}
}

func TestGetExchangeStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeStatus(t.Context())
	require.NoError(t, err, "GetExchangeStatus must not error")
	if mockTests {
		assert.Equal(t, &ExchangeStatusResponse{SpecialStatuses: "synthetic", Time: milli(1791592715097)}, result, "GetExchangeStatus should decode every field")
	} else {
		assert.WithinDuration(t, time.Now(), result.Time.Time(), time.Minute, "GetExchangeStatus should return the server time")
	}

	_, err = newResponseServerExchange(t, `null`).GetExchangeStatus(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetExchangeStatus should reject a null response")
	_, err = newUnavailableServerExchange(t).GetExchangeStatus(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetExchangeStatus should return a request failure")
}

func TestGetRecentTradesForCoin(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTradesForCoin(t.Context(), "")
	require.ErrorIs(t, err, errCoinRequired, "GetRecentTradesForCoin must require a coin")

	result, err := e.GetRecentTradesForCoin(t.Context(), "BTC")
	require.NoError(t, err, "GetRecentTradesForCoin must not error")
	if mockTests {
		exp := []RecentTrade{
			{Coin: "BTC", Side: "A", Price: 85812, Size: 0.00347, Time: milli(1791275814004), Hash: "0xe3c768699c49511ce5410445f45e660201be004f374c6fee879013bc5b4d2b07", TradeID: 284092582946010, Users: []string{"0x000000000000000000000000000000000000000a", "0x000000000000000000000000000000000000000b"}},
			{Coin: "BTC", Side: "B", Price: 85813, Size: 0.01757, Time: milli(1791275814806), Hash: "0x0000000000000000000000000000000000000000000000000000000000000000", TradeID: 744489298526402, Users: []string{"0x000000000000000000000000000000000000000c", "0x000000000000000000000000000000000000000d"}},
		}
		assert.Equal(t, exp, result, "GetRecentTradesForCoin should decode every trade field")
	} else {
		assert.NotEmpty(t, result, "GetRecentTradesForCoin should return trades")
	}
}

func TestGetTWAPHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTWAPHistory(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "GetTWAPHistory must reject an invalid address")

	// No public account is known to have TWAP history, so live tests only check that the response decodes
	result, err := e.GetTWAPHistory(t.Context(), testTraderAddress)
	require.NoError(t, err, "GetTWAPHistory must not error")
	if mockTests {
		exp := []TWAPHistoryEntry{
			{
				Time: types.Time(time.Unix(1791348375, 0)),
				State: TWAPState{
					Coin:             "xyz:SMSN",
					User:             testTraderAddress,
					Side:             "B",
					Size:             271.24,
					ExecutedSize:     135.62,
					ExecutedNotional: 27358.31,
					Minutes:          120,
					ReduceOnly:       true,
					Randomise:        true,
					Timestamp:        milli(1791348375494),
					Trigger:          &TWAPTrigger{Price: 201.5, Above: true},
					StopPrice:        215,
				},
				Status: TWAPStatus{Status: "activated"},
				TWAPID: 2292056,
			},
			{
				Time: types.Time(time.Unix(1784091798, 0)),
				State: TWAPState{
					Coin:      "xyz:SKHY",
					User:      testTraderAddress,
					Side:      "A",
					Size:      1400,
					Minutes:   1440,
					Timestamp: milli(1784091796275),
				},
				Status: TWAPStatus{Status: "error", Description: "Insufficient margin to place order."},
				TWAPID: 2031821,
			},
		}
		assert.Equal(t, exp, result, "GetTWAPHistory should decode every field")
	}
}

func TestGetUserTWAPSliceFillsByTime(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	for _, tc := range []struct {
		arg *UserTWAPSliceFillsByTimeRequest
		err error
	}{
		{err: common.ErrNilPointer},
		{arg: &UserTWAPSliceFillsByTimeRequest{User: "invalid", StartTime: start}, err: errInvalidAddress},
		{arg: &UserTWAPSliceFillsByTimeRequest{User: testTraderAddress}, err: common.ErrDateUnset},
		{arg: &UserTWAPSliceFillsByTimeRequest{User: testTraderAddress, StartTime: time.Unix(0, 0)}, err: common.ErrDateUnset},
		{arg: &UserTWAPSliceFillsByTimeRequest{User: testTraderAddress, StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetUserTWAPSliceFillsByTime(t.Context(), tc.arg)
		require.ErrorIsf(t, err, tc.err, "GetUserTWAPSliceFillsByTime must reject %+v", tc.arg)
	}

	// No public account is known to have TWAP slice fills, so live tests only check that the response decodes
	exp := []TWAPSliceFill{
		{
			Fill: Fill{
				Coin:          "BTC",
				Price:         85536,
				Size:          0.00347,
				Side:          "A",
				Time:          milli(1791263130059),
				StartPosition: 7.57725,
				Direction:     "Close Long",
				ClosedPNL:     7.072207,
				Hash:          "0x0000000000000000000000000000000000000000000000000000000000000000",
				OrderID:       567180581716,
				Crossed:       true,
				Fee:           0.106851,
				TradeID:       744181485982398,
				FeeToken:      currency.USDC,
			},
			TWAPID: 2288752,
		},
		{
			Fill: Fill{
				Coin:          "BTC",
				Price:         85536,
				Size:          0.00347,
				Side:          "A",
				Time:          milli(1791263136038),
				StartPosition: 7.57378,
				Direction:     "Close Long",
				ClosedPNL:     7.072207,
				Hash:          "0x0000000000000000000000000000000000000000000000000000000000000000",
				OrderID:       567180647975,
				Crossed:       true,
				Fee:           0.106851,
				TradeID:       347271508998491,
				FeeToken:      currency.USDC,
			},
			TWAPID: 2288753,
		},
	}
	for _, arg := range []*UserTWAPSliceFillsByTimeRequest{
		{User: testTraderAddress, StartTime: start, EndTime: end},
		{User: testTraderAddress, StartTime: start},
	} {
		result, err := e.GetUserTWAPSliceFillsByTime(t.Context(), arg)
		require.NoErrorf(t, err, "GetUserTWAPSliceFillsByTime must not error for %+v", arg)
		if mockTests {
			assert.Equalf(t, exp, result, "GetUserTWAPSliceFillsByTime should decode every slice fill, oldest first, for %+v", arg)
		}
	}
}

func TestGetGossipPriorityAuctionStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetGossipPriorityAuctionStatus(t.Context())
	require.NoError(t, err, "GetGossipPriorityAuctionStatus must not error")
	if mockTests {
		exp := &GossipPriorityAuctionStatusResponse{
			PreviousWinners: []string{"192.0.2.10", "198.51.100.20"},
			Auctions: []GasAuctionResponse{
				{StartTime: types.Time(time.Unix(1791279360, 0)), DurationSeconds: 180, StartGas: 14.4468832, EndGas: 1.45386754},
				{StartTime: types.Time(time.Unix(1791279360, 0)), DurationSeconds: 180, StartGas: 1.5771516, CurrentGas: 0.19290462},
			},
		}
		assert.Equal(t, exp, result, "GetGossipPriorityAuctionStatus should decode the winners and every auction field")
	} else {
		assert.NotEmpty(t, result.Auctions, "GetGossipPriorityAuctionStatus should return the slot auctions")
	}

	_, err = newResponseServerExchange(t, `null`).GetGossipPriorityAuctionStatus(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetGossipPriorityAuctionStatus should reject a null response")
	_, err = newUnavailableServerExchange(t).GetGossipPriorityAuctionStatus(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetGossipPriorityAuctionStatus should return a request failure")
}

func TestFormatInterval(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.OneMin:     "1m",
		kline.ThreeMin:   "3m",
		kline.FiveMin:    "5m",
		kline.FifteenMin: "15m",
		kline.ThirtyMin:  "30m",
		kline.OneHour:    "1h",
		kline.TwoHour:    "2h",
		kline.FourHour:   "4h",
		kline.EightHour:  "8h",
		kline.TwelveHour: "12h",
		kline.OneDay:     "1d",
		kline.ThreeDay:   "3d",
		kline.OneWeek:    "1w",
		kline.OneMonth:   "1M",
	} {
		formatted, err := formatInterval(interval)
		require.NoErrorf(t, err, "formatInterval must not error for %s", interval)
		assert.Equalf(t, exp, formatted, "formatInterval should format %s", interval)
	}
	_, err := formatInterval(kline.TenMin)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "formatInterval should reject an unsupported interval")
}

func TestParseInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval string
		exp      kline.Interval
		err      error
	}{
		{interval: "1m", exp: kline.OneMin},
		{interval: "3m", exp: kline.ThreeMin},
		{interval: "5m", exp: kline.FiveMin},
		{interval: "15m", exp: kline.FifteenMin},
		{interval: "30m", exp: kline.ThirtyMin},
		{interval: "1h", exp: kline.OneHour},
		{interval: "2h", exp: kline.TwoHour},
		{interval: "4h", exp: kline.FourHour},
		{interval: "8h", exp: kline.EightHour},
		{interval: "12h", exp: kline.TwelveHour},
		{interval: "1d", exp: kline.OneDay},
		{interval: "3d", exp: kline.ThreeDay},
		{interval: "1w", exp: kline.OneWeek},
		{interval: "1M", exp: kline.OneMonth},
		{interval: "10m", err: kline.ErrUnsupportedInterval},
		{interval: "bad", err: kline.ErrInvalidInterval},
	} {
		parsed, err := parseInterval(tc.interval)
		require.ErrorIsf(t, err, tc.err, "parseInterval must return the expected error for %q", tc.interval)
		assert.Equalf(t, tc.exp, parsed, "parseInterval should parse %q", tc.interval)
	}
}

func TestZonelessTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var z ZonelessTime
	require.NoError(t, json.Unmarshal([]byte(`"2025-11-23T17:37:10.033211662"`), &z), "Unmarshal must not error for a zone-less timestamp")
	assert.Equal(t, time.Date(2025, 11, 23, 17, 37, 10, 33211662, time.UTC), z.Time(), "ZonelessTime should decode as UTC")
	require.NoError(t, json.Unmarshal([]byte(`""`), &z), "Unmarshal must not error for an empty timestamp")
	assert.True(t, z.Time().IsZero(), "ZonelessTime should decode an empty timestamp as zero")
	assert.ErrorIs(t, json.Unmarshal([]byte(`1`), &z), errInvalidZonelessTime, "ZonelessTime should reject a number")
	assert.ErrorIs(t, json.Unmarshal([]byte(`"2025-11-23 17:37"`), &z), errInvalidZonelessTime, "ZonelessTime should reject another layout")
}

func TestZonelessTimeTime(t *testing.T) {
	t.Parallel()
	exp := time.Date(2025, 11, 23, 17, 37, 10, 0, time.UTC)
	assert.Equal(t, exp, ZonelessTime(exp).Time(), "Time should return the timestamp as a time.Time")
}

func TestIsHex128(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value string
		exp   bool
	}{
		{value: testClientOrderID, exp: true},
		{value: "0X" + strings.ToUpper(testClientOrderID[2:]), exp: true},
		{value: testClientOrderID[2:]},
		{value: testClientOrderID[:33]},
		{value: testClientOrderID + "0"},
		{value: "0x" + strings.Repeat("g", 32)},
		{value: "1x" + testClientOrderID[2:]},
	} {
		assert.Equalf(t, tc.exp, isHex128(tc.value), "isHex128 should report whether %q is a 16-byte hexadecimal value", tc.value)
	}
}

func TestUnmarshalTuple(t *testing.T) {
	t.Parallel()
	var value AssetValue
	require.NoError(t, unmarshalTuple([]byte(`["BTC","1.5"]`), &value.Coin, &value.Value), "unmarshalTuple must not error")
	assert.Equal(t, AssetValue{Coin: "BTC", Value: 1.5}, value, "unmarshalTuple should decode each element into its target")
	assert.ErrorIs(t, unmarshalTuple([]byte(`["BTC"]`), &value.Coin, &value.Value), errInvalidTupleLength, "unmarshalTuple should reject a short tuple")
	assert.ErrorIs(t, unmarshalTuple([]byte(`["BTC","1","2"]`), &value.Coin, &value.Value), errInvalidTupleLength, "unmarshalTuple should reject a long tuple")
	assert.Error(t, unmarshalTuple([]byte(`{}`), &value.Coin, &value.Value), "unmarshalTuple should reject an object")
	assert.Error(t, unmarshalTuple([]byte(`[1,"1"]`), &value.Coin, &value.Value), "unmarshalTuple should reject a mistyped element")
}

func TestPortfolioPeriodUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var period PortfolioPeriod
	require.NoError(t, json.Unmarshal([]byte(`["week",{"accountValueHistory":[[1791189344472,"1.5"]],"pnlHistory":[[1791189344472,"-0.5"]],"vlm":"2.0"}]`), &period), "Unmarshal must not error")
	exp := PortfolioPeriod{
		Period: "week",
		History: PortfolioHistory{
			AccountValueHistory: []PortfolioValue{{Time: milli(1791189344472), Value: 1.5}},
			PNLHistory:          []PortfolioValue{{Time: milli(1791189344472), Value: -0.5}},
			Volume:              2,
		},
	}
	assert.Equal(t, exp, period, "PortfolioPeriod should decode its period and history")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["week"]`), &period), errInvalidTupleLength, "PortfolioPeriod should reject a short tuple")
}

func TestPortfolioValueUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var value PortfolioValue
	require.NoError(t, json.Unmarshal([]byte(`[1791189344472,"181448095.72"]`), &value), "Unmarshal must not error")
	assert.Equal(t, PortfolioValue{Time: milli(1791189344472), Value: 181448095.72}, value, "PortfolioValue should decode its time and value")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[1791189344472]`), &value), errInvalidTupleLength, "PortfolioValue should reject a short tuple")
}

func TestReferralTokenStateEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var entry ReferralTokenStateEntry
	require.NoError(t, json.Unmarshal([]byte(`[360,{"cumVlm":"1.5","unclaimedRewards":"0.25","claimedRewards":"0.5","builderRewards":"0.125"}]`), &entry), "Unmarshal must not error")
	exp := ReferralTokenStateEntry{Token: 360, State: ReferralTokenState{CumulativeVolume: 1.5, UnclaimedRewards: 0.25, ClaimedRewards: 0.5, BuilderRewards: 0.125}}
	assert.Equal(t, exp, entry, "ReferralTokenStateEntry should decode its token and state")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[360]`), &entry), errInvalidTupleLength, "ReferralTokenStateEntry should reject a short tuple")
}

func TestReferralStateTokenEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var entry ReferralStateTokenEntry
	require.NoError(t, json.Unmarshal([]byte(`[360,{"cumVlm":"1.5","cumRewardedFeesSinceReferred":"0.25","cumFeesRewardedToReferrer":"0.025"}]`), &entry), "Unmarshal must not error")
	exp := ReferralStateTokenEntry{Token: 360, State: ReferralStateTokenState{CumulativeVolume: 1.5, CumulativeRewardedFeesSinceReferred: 0.25, CumulativeFeesRewardedToReferrer: 0.025}}
	assert.Equal(t, exp, entry, "ReferralStateTokenEntry should decode its token and state")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[360]`), &entry), errInvalidTupleLength, "ReferralStateTokenEntry should reject a short tuple")
}

func TestBorrowLendTokenStateEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var entry BorrowLendTokenStateEntry
	require.NoError(t, json.Unmarshal([]byte(`[150,{"borrow":{"basis":"1.0","value":"1.5"},"supply":{"basis":"2.0","value":"2.5"}}]`), &entry), "Unmarshal must not error")
	exp := BorrowLendTokenStateEntry{Token: 150, State: BorrowLendTokenState{Borrow: BorrowLendBalance{Basis: 1, Value: 1.5}, Supply: BorrowLendBalance{Basis: 2, Value: 2.5}}}
	assert.Equal(t, exp, entry, "BorrowLendTokenStateEntry should decode its token and state")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[150]`), &entry), errInvalidTupleLength, "BorrowLendTokenStateEntry should reject a short tuple")
}

func TestBorrowLendReserveStateEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var entry BorrowLendReserveStateEntry
	require.NoError(t, json.Unmarshal([]byte(`[0,{"borrowYearlyRate":"0.05","totalBorrowed":"0.5"}]`), &entry), "Unmarshal must not error")
	assert.Equal(t, BorrowLendReserveStateEntry{State: BorrowLendReserveStateResponse{BorrowYearlyRate: 0.05, TotalBorrowed: 0.5}}, entry, "BorrowLendReserveStateEntry should decode its token and state")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[0]`), &entry), errInvalidTupleLength, "BorrowLendReserveStateEntry should reject a short tuple")
}

func TestGossipPriorityAuctionStatusResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var status GossipPriorityAuctionStatusResponse
	require.NoError(t, json.Unmarshal([]byte(`[["192.0.2.10"],[{"startTimeSeconds":1791279360,"durationSeconds":180,"startGas":"2.0","currentGas":"1.0","endGas":null}]]`), &status), "Unmarshal must not error")
	exp := GossipPriorityAuctionStatusResponse{
		PreviousWinners: []string{"192.0.2.10"},
		Auctions:        []GasAuctionResponse{{StartTime: types.Time(time.Unix(1791279360, 0)), DurationSeconds: 180, StartGas: 2, CurrentGas: 1}},
	}
	assert.Equal(t, exp, status, "GossipPriorityAuctionStatusResponse should decode its winners and auctions")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[["192.0.2.10"]]`), &status), errInvalidTupleLength, "GossipPriorityAuctionStatusResponse should reject a short tuple")
}
