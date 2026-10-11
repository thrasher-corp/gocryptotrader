package hyperliquid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const (
	// testBuilderAddress and testValidatorAddress are the placeholder builder and validator of the mock responses
	testBuilderAddress   = "0x0000000000000000000000000000000000000015"
	testValidatorAddress = "0x0000000000000000000000000000000000000016"
	// testPriorityClientOrderID is the client order ID of the mock responses' priority order
	testPriorityClientOrderID = "0x000000000000000000065d27f23d74a0"
	defaultActionResponse     = `{"status":"ok","response":{"type":"default"}}`
	rejectedActionResponse    = `{"status":"err","response":"Insufficient margin to place order."}`
)

// newActionServerExchange returns an exchange with the given credentials whose exchange requests are decoded into
// captured and answered with response; info requests receive a user role, or an owned subaccount role for
// testSubAccountAddress, the xyz DEX registry, USDC collateral metadata and the USDC and HYPE spot tokens
func newActionServerExchange(t *testing.T, credentials *accounts.Credentials, captured *SignedActionRequest, response string) *Exchange {
	t.Helper()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := response
		switch r.URL.Path {
		case "/info":
			var infoRequest InfoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&infoRequest), "Decoding the info request should not error") {
				return
			}
			switch infoRequest.Type {
			case "userRole":
				body = `{"role":"user"}`
				if infoRequest.User == testSubAccountAddress {
					body = `{"role":"subAccount","data":{"master":"` + testAccountAddress + `"}}`
				}
			case "perpDexs":
				body = `[null,{"name":"xyz"}]`
			case "meta":
				body = `{"universe":[],"collateralToken":0}`
			case "spotMeta":
				body = `{"universe":[],"tokens":[{"name":"USDC","index":0,"tokenId":"0x6d1e7cde53ba9467b783cb7c530ce054"},{"name":"HYPE","index":150,"tokenId":"0x0d01dc56dcaaca66ad901c959b4011ec"}]}`
			default:
				http.Error(w, "unexpected info request "+infoRequest.Type, http.StatusBadRequest)
				return
			}
		case "/exchange":
			*captured = SignedActionRequest{}
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(captured), "Decoding the exchange request should not error") {
				return
			}
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	ex.Name = t.Name()
	setTestCredentials(ex, credentials)
	return ex
}

// signingCredentials returns credentials holding the mock signing account's own key
func signingCredentials() *accounts.Credentials {
	return &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey}
}

// subAccountCredentials returns signing credentials that act for testSubAccountAddress
func subAccountCredentials() *accounts.Credentials {
	return &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey, SubAccount: testSubAccountAddress}
}

// unownedSubAccountCredentials returns signing credentials that act for a subaccount the account does not own, which
// newActionServerExchange reports as a user
func unownedSubAccountCredentials() *accounts.Credentials {
	return &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey, SubAccount: testVaultAddress}
}

// agentCredentials returns credentials holding an API wallet's key, which cannot sign user-signed actions
func agentCredentials() *accounts.Credentials {
	return &accounts.Credentials{Key: testAccountAddress, Secret: testAgentPrivateKey}
}

// capturedAction returns a captured request's action as JSON
func capturedAction(t *testing.T, captured *SignedActionRequest) string {
	t.Helper()
	encoded, err := json.Marshal(captured.Action)
	require.NoError(t, err, "Marshal must not error")
	return string(encoded)
}

// assertUserSignedNonce checks the nonce a user-signed action returned: the recorded one in mock mode
func assertUserSignedNonce(t *testing.T, nonce uint64) {
	t.Helper()
	if mockTests {
		assert.Equal(t, uint64(mockUserSignedNonce), nonce, "The action should return the recorded nonce")
		return
	}
	assert.NotZero(t, nonce, "The action should return its nonce")
}

// skipLiveWithoutPrivateKey skips a live test of a signed action when the live credentials hold no private key, which
// signing needs
func skipLiveWithoutPrivateKey(t *testing.T) {
	t.Helper()
	if _, _, err := e.getSigningCredentials(t.Context()); errors.Is(err, errPrivateKeyRequired) {
		t.Skip("Live signed action tests need a private key in apiCredentials")
	}
}

// skipLiveUnlessAccountSigner skips a live test of a user-signed action unless the live credentials hold the account's
// own private key and configure no subaccount or vault, as the live tests move the account's own funds and change its
// own settings
func skipLiveUnlessAccountSigner(t *testing.T) {
	t.Helper()
	_, err := e.getAccountSigningCredentials(t.Context())
	if errors.Is(err, errPrivateKeyRequired) || errors.Is(err, errUserSignedMasterRequired) || errors.Is(err, errTransferSubAccountUnsupported) {
		t.Skip("Live user-signed action tests need the account's own private key in apiCredentials, and no subaccount or vault")
	}
	require.NoError(t, err, "getAccountSigningCredentials must not error")
}

// skipLiveWithoutTransferDestination skips a live transfer test unless liveTransferDestination is set, as live
// transfers must reach an address the tester controls
func skipLiveWithoutTransferDestination(t *testing.T) {
	t.Helper()
	if liveTransferDestination == "" {
		t.Skip("Set liveTransferDestination to an address you control to run live transfer tests")
	}
}

// liveAccountAddress returns the live credentials' account address
func liveAccountAddress(t *testing.T) string {
	t.Helper()
	credentials, err := e.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	address, _, err := normaliseAddress(credentials.Key)
	require.NoError(t, err, "normaliseAddress must not error for the live account")
	return address
}

// liveBTCPosition returns the BTC position of the account live trading tests act for, or nil when it has none
func liveBTCPosition(t *testing.T) *Position {
	t.Helper()
	address, err := e.getWatchAddress(t.Context())
	require.NoError(t, err, "getWatchAddress must not error")
	state, err := e.GetClearinghouseState(t.Context(), address, "")
	require.NoError(t, err, "GetClearinghouseState must not error")
	for i := range state.AssetPositions {
		if state.AssetPositions[i].Position.Coin == "BTC" {
			return &state.AssetPositions[i].Position
		}
	}
	return nil
}

// liveBTCMidPrice returns the live BTC mid price, which live order tests price their orders well away from
func liveBTCMidPrice(t *testing.T) float64 {
	t.Helper()
	mids, err := e.GetAllMids(t.Context(), "")
	require.NoError(t, err, "GetAllMids must not error")
	mid := mids["BTC"].Float64()
	require.Positive(t, mid, "The BTC mid price must be positive")
	return mid
}

// livePrice returns a fraction of a BTC price as a whole number of at most five significant figures, which Hyperliquid
// accepts as a BTC price
func livePrice(price, fraction float64) float64 {
	price *= fraction
	step := max(1, math.Pow(10, math.Floor(math.Log10(price))-4))
	return math.Round(price/step) * step
}

// liveSize returns the smallest BTC size, in its five size decimals, worth at least a notional value at a price
func liveSize(price, notional float64) float64 {
	return math.Ceil(notional/price*1e5) / 1e5
}

// newClientOrderID returns a random client order ID, as live orders must not reuse one
func newClientOrderID(t *testing.T) string {
	t.Helper()
	var id [16]byte
	_, err := rand.Read(id[:])
	require.NoError(t, err, "rand.Read must not error")
	return "0x" + hex.EncodeToString(id[:])
}

// newAgentAddress returns the address of a freshly generated private key, which nobody holds once it is discarded
func newAgentAddress(t *testing.T) string {
	t.Helper()
	key, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err, "GeneratePrivateKey must not error")
	defer key.Zero()
	return privateKeyAddress(key)
}

// liveRestingOrder returns a minimal BTC buy at half the live mid price, which rests without matching, under a fresh
// client order ID that is cancelled when the test ends
func liveRestingOrder(t *testing.T) OrderRequest {
	t.Helper()
	price := livePrice(liveBTCMidPrice(t), 0.5)
	clientOrderID := newClientOrderID(t)
	ctx := context.WithoutCancel(t.Context())
	t.Cleanup(func() {
		// Cancelling an order that has already gone reports a status rather than an error
		_, err := e.CancelOrdersByClientOrderID(ctx, &CancelOrdersByClientOrderIDRequest{Cancels: []CancelByClientOrderIDRequest{{Asset: 0, ClientOrderID: clientOrderID}}})
		assert.NoError(t, err, "CancelOrdersByClientOrderID should cancel the live test order")
	})
	// Hyperliquid's minimum order value is $10
	return OrderRequest{Asset: 0, IsBuy: true, Price: price, Size: liveSize(price, 11), Limit: &LimitOrderType{TimeInForce: TimeInForceALO}, ClientOrderID: clientOrderID}
}

// placeLiveRestingOrder places liveRestingOrder and returns it with its order ID
func placeLiveRestingOrder(t *testing.T) (resting OrderRequest, orderID uint64) {
	t.Helper()
	resting = liveRestingOrder(t)
	statuses, err := e.PlaceOrders(t.Context(), &PlaceOrdersRequest{Orders: []OrderRequest{resting}})
	require.NoError(t, err, "PlaceOrders must not error")
	require.Len(t, statuses, 1, "PlaceOrders must return the live test order's status")
	require.NotNilf(t, statuses[0].Resting, "The live test order must rest, got %+v", statuses[0])
	return resting, statuses[0].Resting.OrderID
}

// liveIsolatedOnlyTopUp returns a top-up of the live account's first position on an isolated-only market to its
// current leverage, which leaves its margin as it is, skipping the test when the account has no such position
func liveIsolatedOnlyTopUp(t *testing.T) *TopUpIsolatedOnlyMarginRequest {
	t.Helper()
	address, err := e.getWatchAddress(t.Context())
	require.NoError(t, err, "getWatchAddress must not error")
	state, err := e.GetClearinghouseState(t.Context(), address, "")
	require.NoError(t, err, "GetClearinghouseState must not error")
	metadata, err := e.GetPerpetualMetadata(t.Context(), "")
	require.NoError(t, err, "GetPerpetualMetadata must not error")
	for i := range state.AssetPositions {
		position := &state.AssetPositions[i].Position
		index := slices.IndexFunc(metadata.Universe, func(asset PerpetualAssetMetadata) bool { return asset.Name == position.Coin })
		if index < 0 || position.MarginUsed <= 0 {
			continue
		}
		if asset := &metadata.Universe[index]; asset.OnlyIsolated || asset.MarginMode == "strictIsolated" || asset.MarginMode == "noCross" {
			return &TopUpIsolatedOnlyMarginRequest{Asset: uint64(index), Leverage: math.Round(position.PositionValue.Float64()/position.MarginUsed.Float64()*100) / 100}
		}
	}
	t.Skip("Live TopUpIsolatedOnlyMargin needs a position on an isolated-only market")
	return nil
}

// liveTWAPOrderRequest returns a reduce-only BTC TWAP order worth just over the $100 minimum, on the side that cannot
// reduce the live account's position, so none of its suborders can fill
func liveTWAPOrderRequest(t *testing.T) *TWAPOrderRequest {
	t.Helper()
	var signedSize float64
	if position := liveBTCPosition(t); position != nil {
		signedSize = position.SignedSize.Float64()
	}
	// A reduce-only buy can only reduce a short position, and a reduce-only sell only a long one
	return &TWAPOrderRequest{Asset: 0, IsBuy: signedSize >= 0, Size: liveSize(liveBTCMidPrice(t), 110), ReduceOnly: true, Minutes: 30}
}

// liveApprovedBuilderFee returns an approval of the live account's first approved builder at its current maximum fee,
// skipping the test when the account has approved none
func liveApprovedBuilderFee(t *testing.T) *ApproveBuilderFeeRequest {
	t.Helper()
	account := liveAccountAddress(t)
	builders, err := e.GetApprovedBuilders(t.Context(), account)
	require.NoError(t, err, "GetApprovedBuilders must not error")
	if len(builders) == 0 {
		t.Skip("Live ApproveBuilderFee approves an approved builder's current maximum fee again, and the live account has approved none")
	}
	maximumFee, err := e.GetMaxBuilderFee(t.Context(), account, builders[0])
	require.NoError(t, err, "GetMaxBuilderFee must not error")
	// Maximum builder fees are in tenths of a basis point, which are thousandths of a percent
	return &ApproveBuilderFeeRequest{Builder: builders[0], MaxFeeRate: float64(maximumFee) / 1000}
}

// liveDEXAbstraction returns whether the live account has HIP-3 DEX abstraction enabled, or nil when it never set it
func liveDEXAbstraction(t *testing.T) *bool {
	t.Helper()
	enabled, err := e.GetUserDEXAbstraction(t.Context(), liveAccountAddress(t))
	require.NoError(t, err, "GetUserDEXAbstraction must not error")
	return enabled
}

// liveSettableAbstraction returns the live account's abstraction mode, skipping the test unless an abstraction action
// can set that mode back
func liveSettableAbstraction(t *testing.T) AccountAbstraction {
	t.Helper()
	abstraction, err := e.GetUserAbstraction(t.Context(), liveAccountAddress(t))
	require.NoError(t, err, "GetUserAbstraction must not error")
	switch abstraction {
	case AccountAbstractionDisabled, AccountAbstractionUnifiedAccount, AccountAbstractionPortfolioMargin:
		return abstraction
	default:
		t.Skipf("Live abstraction tests set the account's current mode back, which an abstraction action cannot set to %s", abstraction)
		return ""
	}
}

// liveOutcomeQuestion returns an active question without settled outcomes, and its first named outcome
func liveOutcomeQuestion(t *testing.T) (question, outcome uint64) {
	t.Helper()
	metadata, err := e.GetOutcomeMetadata(t.Context())
	require.NoError(t, err, "GetOutcomeMetadata must not error")
	for i := range metadata.Questions {
		if q := &metadata.Questions[i]; len(q.NamedOutcomes) != 0 && len(q.SettledNamedOutcomes) == 0 {
			return q.Question, q.NamedOutcomes[0]
		}
	}
	t.Skip("Live outcome tests need an active outcome question")
	return 0, 0
}

func TestPlaceOrders(t *testing.T) {
	t.Parallel()
	limit := OrderRequest{Asset: 0, IsBuy: true, Price: 85000, Size: 0.001, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, ClientOrderID: testClientOrderID}
	immediate := OrderRequest{Asset: 0, IsBuy: true, Price: 86000, Size: 0.002, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}}
	addLiquidity := OrderRequest{Asset: 0, IsBuy: true, Price: 84000, Size: 0.002, Limit: &LimitOrderType{TimeInForce: TimeInForceALO}}
	for _, tc := range []struct {
		name string
		arg  *PlaceOrdersRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no orders", arg: &PlaceOrdersRequest{}, err: errNoActionItems},
		{name: "unknown grouping", arg: &PlaceOrdersRequest{Orders: []OrderRequest{limit}, Grouping: "other"}, err: errInvalidGrouping},
		{name: "invalid order", arg: &PlaceOrdersRequest{Orders: []OrderRequest{{Price: 1}}}, err: order.ErrAmountIsInvalid},
		{name: "priority rate above the maximum", arg: &PlaceOrdersRequest{Orders: []OrderRequest{immediate}, PriorityRate: maximumPriorityRate + 1}, err: errPriorityRateInvalid},
		{name: "priority with TP/SL grouping", arg: &PlaceOrdersRequest{Orders: []OrderRequest{immediate}, Grouping: GroupingNormalTPSL, PriorityRate: 1}, err: errPriorityGroupingInvalid},
		{name: "priority on an outcome", arg: &PlaceOrdersRequest{Orders: []OrderRequest{{Asset: outcomeAssetIDBase, Price: 0.5, Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}}}, PriorityRate: 1}, err: errPriorityGroupingInvalid},
		{name: "priority on a trigger order", arg: &PlaceOrdersRequest{Orders: []OrderRequest{{Price: 1, Size: 1, Trigger: &TriggerOrderType{TriggerPrice: 1, TakeProfitStopLoss: TriggerStopLoss}}}, PriorityRate: 1}, err: errPriorityGroupingInvalid},
		{name: "priority on a GTC order", arg: &PlaceOrdersRequest{Orders: []OrderRequest{limit}, PriorityRate: 1}, err: errPriorityGroupingInvalid},
		{name: "priority on a reduce-only ALO order", arg: &PlaceOrdersRequest{Orders: []OrderRequest{{Price: 1, Size: 1, ReduceOnly: true, Limit: &LimitOrderType{TimeInForce: TimeInForceALO}}}, PriorityRate: 1}, err: errPriorityGroupingInvalid},
		{name: "priority on mixed IOC and ALO orders", arg: &PlaceOrdersRequest{Orders: []OrderRequest{immediate, addLiquidity}, PriorityRate: 1}, err: errPriorityGroupingInvalid},
		{name: "invalid builder", arg: &PlaceOrdersRequest{Orders: []OrderRequest{limit}, Builder: &BuilderFee{Builder: "invalid"}}, err: errInvalidAddress},
		{name: "the zero address as builder", arg: &PlaceOrdersRequest{Orders: []OrderRequest{limit}, Builder: &BuilderFee{Builder: testZeroAddress}}, err: errInvalidAddress},
		{name: "perpetual builder fee above the maximum", arg: &PlaceOrdersRequest{Orders: []OrderRequest{limit}, Builder: &BuilderFee{Builder: testBuilderAddress, Fee: maximumPerpetualBuilderFee + 1}}, err: errBuilderFeeInvalid},
		{name: "expired action", arg: &PlaceOrdersRequest{Orders: []OrderRequest{limit}, ExpiresAfter: time.Now().Add(-time.Second)}, err: errExpiresAfterPassed},
	} {
		_, err := e.PlaceOrders(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "PlaceOrders should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, statusesResponse("order", `{"resting":{"oid":1}}`))
	expiresAfter := time.Now().Add(time.Minute)
	spot := OrderRequest{Asset: 10107, Price: 95.5, Size: 1.5, Limit: &LimitOrderType{TimeInForce: TimeInForceALO}}
	_, err := ex.PlaceOrders(t.Context(), &PlaceOrdersRequest{Orders: []OrderRequest{spot}, PriorityRate: 5, Builder: &BuilderFee{Builder: strings.ToUpper(testBuilderAddress[2:]), Fee: maximumSpotBuilderFee}, ExpiresAfter: expiresAfter})
	require.ErrorIs(t, err, errInvalidAddress, "PlaceOrders must reject a builder address without the 0x prefix")
	statuses, err := ex.PlaceOrders(t.Context(), &PlaceOrdersRequest{Orders: []OrderRequest{spot}, PriorityRate: 5, Builder: &BuilderFee{Builder: "0X" + strings.ToUpper(testBuilderAddress[2:]), Fee: maximumSpotBuilderFee}, ExpiresAfter: expiresAfter})
	require.NoError(t, err, "PlaceOrders must not error for a spot order with the spot builder fee maximum")
	assert.Equal(t, []OrderActionStatus{{Resting: &RestingOrderStatus{OrderID: 1}}}, statuses, "PlaceOrders should return the order's status")
	assert.JSONEq(t, `{"type":"order","orders":[{"a":10107,"b":false,"p":"95.5","s":"1.5","r":false,"t":{"limit":{"tif":"Alo"}}}],"grouping":{"p":5},"builder":{"b":"`+testBuilderAddress+`","f":1000}}`, capturedAction(t, &captured), "PlaceOrders should send a priority grouping and the lower-case builder")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "PlaceOrders should act for the configured subaccount")
	require.NotNil(t, captured.ExpiresAfter, "PlaceOrders must send the expiry")
	assert.Equal(t, unixMilli(expiresAfter), *captured.ExpiresAfter, "PlaceOrders should send the expiry")

	bracket := []OrderRequest{
		limit,
		{Asset: 0, Price: 81000, Size: 0.001, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 90000, TakeProfitStopLoss: TriggerTakeProfit}},
		{Asset: 0, Price: 79000, Size: 0.001, ReduceOnly: true, Trigger: &TriggerOrderType{TriggerPrice: 80000, TakeProfitStopLoss: TriggerStopLoss}},
	}
	priority := []OrderRequest{
		{Asset: 0, IsBuy: true, Price: 86000, Size: 0.002, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: testPriorityClientOrderID},
		{Asset: 1, Price: 2700, Size: 0.05, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}},
	}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The parent rests far below the market, and cancelling it unfilled when the test ends also cancels its children;
		// the take profit triggers above the mid price and the stop loss below the parent, as Hyperliquid requires
		parent := liveRestingOrder(t)
		bracket = []OrderRequest{
			parent,
			{Asset: 0, Price: livePrice(parent.Price, 2), Size: parent.Size, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: livePrice(parent.Price, 2.2), TakeProfitStopLoss: TriggerTakeProfit}},
			{Asset: 0, Price: livePrice(parent.Price, 0.88), Size: parent.Size, ReduceOnly: true, Trigger: &TriggerOrderType{TriggerPrice: livePrice(parent.Price, 0.9), TakeProfitStopLoss: TriggerStopLoss}},
		}
		// An IOC buy at half the mid price cannot fill, so it pays no priority fee
		priority = []OrderRequest{{Asset: 0, IsBuy: true, Price: parent.Price, Size: parent.Size, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: newClientOrderID(t)}}
	}
	statuses, err = e.PlaceOrders(t.Context(), &PlaceOrdersRequest{Orders: bracket, Grouping: GroupingNormalTPSL})
	require.NoError(t, err, "PlaceOrders must not error for a bracket order")
	if mockTests {
		exp := []OrderActionStatus{
			{Resting: &RestingOrderStatus{OrderID: testOrderID, ClientOrderID: testClientOrderID}},
			{Waiting: orderStatusWaitingForFill},
			{Waiting: orderStatusWaitingForFill},
		}
		assert.Equal(t, exp, statuses, "PlaceOrders should return the parent's resting status and its children's waiting statuses")
	} else {
		require.Len(t, statuses, len(bracket), "PlaceOrders must return a status for each bracket order")
		assert.NotNilf(t, statuses[0].Resting, "PlaceOrders should rest the live parent order, got %+v", statuses[0])
	}

	if !mockTests {
		// Hyperliquid charges priority fees to the undelegated staking balance, and rejects priority orders without one
		summary, summaryErr := e.GetDelegatorSummary(t.Context(), liveAccountAddress(t))
		require.NoError(t, summaryErr, "GetDelegatorSummary must not error")
		if summary.Undelegated <= 0 {
			t.Skip("Live priority orders need an undelegated staking balance")
		}
	}
	statuses, err = e.PlaceOrders(t.Context(), &PlaceOrdersRequest{Orders: priority, PriorityRate: 1000})
	require.NoError(t, err, "PlaceOrders must not error for priority orders")
	if mockTests {
		exp := []OrderActionStatus{
			{Filled: &FilledOrderStatus{TotalSize: 0.002, AveragePrice: 85813.5, OrderID: 566563303545, ClientOrderID: testPriorityClientOrderID}},
			{Error: "Order could not immediately match against any resting orders. asset=1"},
		}
		assert.Equal(t, exp, statuses, "PlaceOrders should return each priority order's status")
	} else {
		assert.Len(t, statuses, len(priority), "PlaceOrders should return a status for each priority order")
	}
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelOrdersRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no cancels", arg: &CancelOrdersRequest{}, err: errNoActionItems},
		{name: "a zero order ID", arg: &CancelOrdersRequest{Cancels: []CancelRequest{{Asset: 1}}}, err: order.ErrOrderIDNotSet},
	} {
		_, err := e.CancelOrders(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "CancelOrders should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, statusesResponse("cancel", `"success"`))
	_, err := ex.CancelOrders(t.Context(), &CancelOrdersRequest{Cancels: []CancelRequest{{Asset: 1, OrderID: 2}}})
	require.NoError(t, err, "CancelOrders must not error")
	assert.JSONEq(t, `{"type":"cancel","cancels":[{"a":1,"o":2}]}`, capturedAction(t, &captured), "CancelOrders should omit the fast flag unless it is set")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "CancelOrders should act for the configured subaccount")
	_, err = newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), rejectedActionResponse).CancelOrders(t.Context(), &CancelOrdersRequest{Cancels: []CancelRequest{{Asset: 1, OrderID: 2}}})
	assert.ErrorIs(t, err, errActionResponse, "CancelOrders should return a rejected action")

	orderID := uint64(testOrderID)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The recorded order only exists in the mock responses
		_, orderID = placeLiveRestingOrder(t)
	}
	statuses, err := e.CancelOrders(t.Context(), &CancelOrdersRequest{Cancels: []CancelRequest{{Asset: 0, OrderID: orderID}, {Asset: 1, OrderID: 566563303543}}, Fast: true})
	require.NoError(t, err, "CancelOrders must not error")
	if mockTests {
		assert.Equal(t, []CancelActionStatus{{}, {Error: "Order was never placed, already canceled, or filled. asset=1"}}, statuses, "CancelOrders should return each cancel's status")
		return
	}
	require.Len(t, statuses, 2, "CancelOrders must return each cancel's status")
	assert.Empty(t, statuses[0].Error, "CancelOrders should cancel the live test order")
}

func TestCancelOrdersByClientOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *CancelOrdersByClientOrderIDRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no cancels", arg: &CancelOrdersByClientOrderIDRequest{}, err: errNoActionItems},
		{name: "an invalid client order ID", arg: &CancelOrdersByClientOrderIDRequest{Cancels: []CancelByClientOrderIDRequest{{ClientOrderID: "0x1"}}}, err: errClientOrderIDInvalid},
	} {
		_, err := e.CancelOrdersByClientOrderID(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "CancelOrdersByClientOrderID should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, statusesResponse("cancel", `"success"`))
	_, err := ex.CancelOrdersByClientOrderID(t.Context(), &CancelOrdersByClientOrderIDRequest{Cancels: []CancelByClientOrderIDRequest{{Asset: 1, ClientOrderID: strings.ToUpper(testClientOrderID)}}, Fast: true})
	require.NoError(t, err, "CancelOrdersByClientOrderID must not error")
	assert.JSONEq(t, `{"type":"cancelByCloid","cancels":[{"asset":1,"cloid":"`+testClientOrderID+`"}],"f":true}`, capturedAction(t, &captured), "CancelOrdersByClientOrderID should send lower-case client order IDs and the fast flag")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "CancelOrdersByClientOrderID should act for the configured subaccount")
	cancelAction := CancelByClientOrderIDAction{Type: "cancelByCloid", Cancels: []CancelByClientOrderIDRequest{{Asset: 1, ClientOrderID: testClientOrderID}}, Fast: true}
	exp, err := signL1Action(testPrivateKey, cancelAction, testSubAccountAddress, captured.Nonce, nil, true)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, exp, captured.Signature, "CancelOrdersByClientOrderID should sign for the configured subaccount")
	_, err = newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), rejectedActionResponse).CancelOrdersByClientOrderID(t.Context(), &CancelOrdersByClientOrderIDRequest{Cancels: []CancelByClientOrderIDRequest{{ClientOrderID: testClientOrderID}}})
	assert.ErrorIs(t, err, errActionResponse, "CancelOrdersByClientOrderID should return a rejected action")

	clientOrderID := testClientOrderID
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The recorded order only exists in the mock responses
		resting, _ := placeLiveRestingOrder(t)
		clientOrderID = resting.ClientOrderID
	}
	statuses, err := e.CancelOrdersByClientOrderID(t.Context(), &CancelOrdersByClientOrderIDRequest{Cancels: []CancelByClientOrderIDRequest{{Asset: 0, ClientOrderID: clientOrderID}}, Fast: true})
	require.NoError(t, err, "CancelOrdersByClientOrderID must not error")
	if mockTests {
		assert.Equal(t, []CancelActionStatus{{}}, statuses, "CancelOrdersByClientOrderID should return the cancel's status")
		return
	}
	require.Len(t, statuses, 1, "CancelOrdersByClientOrderID must return the cancel's status")
	assert.Empty(t, statuses[0].Error, "CancelOrdersByClientOrderID should cancel the live test order")
}

func TestScheduleCancel(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ScheduleCancel(t.Context(), nil), common.ErrNilPointer, "ScheduleCancel must reject a nil request")
	require.ErrorIs(t, e.ScheduleCancel(t.Context(), &ScheduleCancelRequest{Time: time.Now().Add(time.Second)}), errScheduleCancelTimeInvalid, "ScheduleCancel must reject a time less than 5 seconds ahead")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	cancelTime := time.Now().Add(time.Minute)
	require.NoError(t, ex.ScheduleCancel(t.Context(), &ScheduleCancelRequest{Time: cancelTime}), "ScheduleCancel must not error")
	assert.JSONEq(t, `{"type":"scheduleCancel","time":`+strconv.FormatUint(unixMilli(cancelTime), 10)+`}`, capturedAction(t, &captured), "ScheduleCancel should send the cancel time")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "ScheduleCancel should act for the configured subaccount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
	}
	assert.NoError(t, e.ScheduleCancel(t.Context(), &ScheduleCancelRequest{}), "ScheduleCancel should not error when clearing the schedule")
}

func TestModifySingleOrder(t *testing.T) {
	t.Parallel()
	replacement := OrderRequest{Asset: 0, IsBuy: true, Price: 84500, Size: 0.001, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, ClientOrderID: testClientOrderID}
	for _, tc := range []struct {
		name string
		arg  *ModifyOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no identifier", arg: &ModifyOrderRequest{Order: replacement}, err: order.ErrOrderIDNotSet},
		{name: "both identifiers", arg: &ModifyOrderRequest{OrderID: 1, ClientOrderID: testClientOrderID, Order: replacement}, err: errOrderIdentifiersConflict},
		{name: "an invalid client order ID", arg: &ModifyOrderRequest{ClientOrderID: "0x1", Order: replacement}, err: errClientOrderIDInvalid},
		{name: "an invalid replacement", arg: &ModifyOrderRequest{OrderID: 1, Order: OrderRequest{Price: 1, Size: 1}}, err: errOrderTypeRequired},
	} {
		assert.ErrorIsf(t, e.ModifySingleOrder(t.Context(), tc.arg), tc.err, "ModifySingleOrder should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	require.NoError(t, ex.ModifySingleOrder(t.Context(), &ModifyOrderRequest{ClientOrderID: strings.ToUpper(testClientOrderID), Order: replacement}), "ModifySingleOrder must not error")
	assert.JSONEq(t, `{"type":"modify","oid":"`+testClientOrderID+`","order":{"a":0,"b":true,"p":"84500","s":"0.001","r":false,"t":{"limit":{"tif":"Gtc"}},"c":"`+testClientOrderID+`"}}`, capturedAction(t, &captured), "ModifySingleOrder should omit always place unless it is set")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "ModifySingleOrder should act for the configured subaccount")
	modifyAction := ModifyAction{Type: "modify", OrderID: testClientOrderID, Order: OrderWire{Asset: 0, IsBuy: true, Price: "84500", Size: "0.001", Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceGTC}}, ClientOrderID: testClientOrderID}}
	exp, err := signL1Action(testPrivateKey, modifyAction, testSubAccountAddress, captured.Nonce, nil, true)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, exp, captured.Signature, "ModifySingleOrder should sign for the configured subaccount")

	arg := &ModifyOrderRequest{OrderID: testOrderID, Order: replacement, AlwaysPlace: true}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The replacement stays far below the market and keeps the client order ID, which the test's cleanup cancels by
		resting, orderID := placeLiveRestingOrder(t)
		resting.Price = livePrice(resting.Price, 1.1)
		arg = &ModifyOrderRequest{OrderID: orderID, Order: resting, AlwaysPlace: true}
	}
	assert.NoError(t, e.ModifySingleOrder(t.Context(), arg), "ModifySingleOrder should not error")
}

func TestModifyOrders(t *testing.T) {
	t.Parallel()
	replacement := OrderRequest{Asset: 0, Price: 79500, Size: 0.001, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 80500, TakeProfitStopLoss: TriggerStopLoss}}
	for _, tc := range []struct {
		name string
		arg  *ModifyOrdersRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no modifies", arg: &ModifyOrdersRequest{}, err: errNoActionItems},
		{name: "both identifiers", arg: &ModifyOrdersRequest{Modifies: []OrderModification{{OrderID: 1, ClientOrderID: testClientOrderID, Order: replacement}}}, err: errOrderIdentifiersConflict},
		{name: "an invalid replacement", arg: &ModifyOrdersRequest{Modifies: []OrderModification{{OrderID: 1}}}, err: order.ErrAmountIsInvalid},
	} {
		_, err := e.ModifyOrders(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "ModifyOrders should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, statusesResponse("order", `{"resting":{"oid":2}}`))
	_, err := ex.ModifyOrders(t.Context(), &ModifyOrdersRequest{Modifies: []OrderModification{{OrderID: 1, Order: replacement}}, AlwaysPlace: true})
	require.NoError(t, err, "ModifyOrders must not error")
	assert.JSONEq(t, `{"type":"batchModify","modifies":[{"oid":1,"order":{"a":0,"b":false,"p":"79500","s":"0.001","r":true,"t":{"trigger":{"isMarket":true,"triggerPx":"80500","tpsl":"sl"}}}}],"a":true}`, capturedAction(t, &captured), "ModifyOrders should send always place")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "ModifyOrders should act for the configured subaccount")
	_, err = newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), rejectedActionResponse).ModifyOrders(t.Context(), &ModifyOrdersRequest{Modifies: []OrderModification{{OrderID: 1, Order: replacement}}})
	assert.ErrorIs(t, err, errActionResponse, "ModifyOrders should return a rejected action")

	arg := &ModifyOrdersRequest{Modifies: []OrderModification{{ClientOrderID: testClientOrderID, Order: replacement}}, AlwaysPlace: true}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The replacement stays far below the market and keeps the client order ID, which the test's cleanup cancels by
		resting, _ := placeLiveRestingOrder(t)
		resting.Price = livePrice(resting.Price, 1.1)
		arg = &ModifyOrdersRequest{Modifies: []OrderModification{{ClientOrderID: resting.ClientOrderID, Order: resting}}, AlwaysPlace: true}
	}
	statuses, err := e.ModifyOrders(t.Context(), arg)
	require.NoError(t, err, "ModifyOrders must not error")
	if mockTests {
		assert.Equal(t, []OrderActionStatus{{Resting: &RestingOrderStatus{OrderID: 566563303546}}}, statuses, "ModifyOrders should return the replacement's status")
		return
	}
	require.Len(t, statuses, 1, "ModifyOrders must return the replacement's status")
	assert.NotNilf(t, statuses[0].Resting, "ModifyOrders should rest the live replacement, got %+v", statuses[0])
}

func TestUpdateLeverage(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateLeverage(t.Context(), nil), common.ErrNilPointer, "UpdateLeverage must reject a nil request")
	require.ErrorIs(t, e.UpdateLeverage(t.Context(), &UpdateLeverageRequest{}), errInvalidLeverage, "UpdateLeverage must reject zero leverage")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	require.NoError(t, ex.UpdateLeverage(t.Context(), &UpdateLeverageRequest{Asset: 1, Leverage: 5, ExpiresAfter: time.Now().Add(time.Minute)}), "UpdateLeverage must not error")
	assert.JSONEq(t, `{"type":"updateLeverage","asset":1,"isCross":false,"leverage":5}`, capturedAction(t, &captured), "UpdateLeverage should send the leverage")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "UpdateLeverage should act for the configured subaccount")
	assert.NotNil(t, captured.ExpiresAfter, "UpdateLeverage should send the expiry")

	arg := &UpdateLeverageRequest{Asset: 0, IsCross: true, Leverage: 20}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// Setting BTC's current margin mode and leverage leaves the account as it was
		address, err := e.getWatchAddress(t.Context())
		require.NoError(t, err, "getWatchAddress must not error")
		data, err := e.GetActiveAssetData(t.Context(), address, "BTC")
		require.NoError(t, err, "GetActiveAssetData must not error")
		arg = &UpdateLeverageRequest{Asset: 0, IsCross: data.Leverage.Type == "cross", Leverage: data.Leverage.Value}
	}
	assert.NoError(t, e.UpdateLeverage(t.Context(), arg), "UpdateLeverage should not error")
}

func TestUpdateIsolatedMargin(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateIsolatedMargin(t.Context(), nil), common.ErrNilPointer, "UpdateIsolatedMargin must reject a nil request")
	require.ErrorIs(t, e.UpdateIsolatedMargin(t.Context(), &UpdateIsolatedMarginRequest{}), errMarginChangeInvalid, "UpdateIsolatedMargin must reject a zero margin change")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	require.NoError(t, ex.UpdateIsolatedMargin(t.Context(), &UpdateIsolatedMarginRequest{Asset: 1, SignedNotional: -1000000}), "UpdateIsolatedMargin must not error")
	assert.JSONEq(t, `{"type":"updateIsolatedMargin","asset":1,"isBuy":true,"ntli":-1000000}`, capturedAction(t, &captured), "UpdateIsolatedMargin should send a signed margin change")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "UpdateIsolatedMargin should act for the configured subaccount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		if position := liveBTCPosition(t); position == nil || position.Leverage.Type != "isolated" {
			t.Skip("Live UpdateIsolatedMargin needs an isolated BTC position")
		}
	}
	require.NoError(t, e.UpdateIsolatedMargin(t.Context(), &UpdateIsolatedMarginRequest{Asset: 0, SignedNotional: 2000000}), "UpdateIsolatedMargin must not error")
	if !mockTests {
		assert.NoError(t, e.UpdateIsolatedMargin(t.Context(), &UpdateIsolatedMarginRequest{Asset: 0, SignedNotional: -2000000}), "UpdateIsolatedMargin should remove the margin the live test added")
	}
}

func TestTopUpIsolatedOnlyMargin(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.TopUpIsolatedOnlyMargin(t.Context(), nil), common.ErrNilPointer, "TopUpIsolatedOnlyMargin must reject a nil request")
	for _, leverage := range []float64{0, -1, math.NaN()} {
		assert.ErrorIsf(t, e.TopUpIsolatedOnlyMargin(t.Context(), &TopUpIsolatedOnlyMarginRequest{Leverage: leverage}), errTargetLeverageInvalid, "TopUpIsolatedOnlyMargin should reject leverage %v", leverage)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	require.NoError(t, ex.TopUpIsolatedOnlyMargin(t.Context(), &TopUpIsolatedOnlyMarginRequest{Asset: 3, Leverage: 0.5}), "TopUpIsolatedOnlyMargin must not error")
	assert.JSONEq(t, `{"type":"topUpIsolatedOnlyMargin","asset":3,"leverage":"0.5"}`, capturedAction(t, &captured), "TopUpIsolatedOnlyMargin should send the target leverage as a decimal string")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "TopUpIsolatedOnlyMargin should act for the configured subaccount")

	arg := &TopUpIsolatedOnlyMarginRequest{Asset: 3, Leverage: 1.5}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		arg = liveIsolatedOnlyTopUp(t)
	}
	assert.NoError(t, e.TopUpIsolatedOnlyMargin(t.Context(), arg), "TopUpIsolatedOnlyMargin should not error")
}

func TestSendAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SendAssetRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid destination", arg: &SendAssetRequest{Destination: "invalid", Amount: 1}, err: errInvalidAddress},
		{name: "the zero address as destination", arg: &SendAssetRequest{Destination: testZeroAddress, Amount: 1}, err: errInvalidAddress},
		{name: "no amount", arg: &SendAssetRequest{Destination: testWithdrawalAddress}, err: errTransferAmountInvalid},
		{name: "no token", arg: &SendAssetRequest{Destination: testWithdrawalAddress, Amount: 1}, err: errTransferTokenInvalid},
	} {
		_, err := e.SendAsset(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "SendAsset should reject a request with %s", tc.name)
	}
	valid := &SendAssetRequest{Destination: testWithdrawalAddress, Token: currency.USDC, Amount: 1}
	_, err := newActionServerExchange(t, agentCredentials(), new(SignedActionRequest), defaultActionResponse).SendAsset(t.Context(), valid)
	assert.ErrorIs(t, err, errUserSignedMasterRequired, "SendAsset should require the account's own key")
	_, err = newActionServerExchange(t, unownedSubAccountCredentials(), new(SignedActionRequest), defaultActionResponse).SendAsset(t.Context(), valid)
	assert.ErrorIs(t, err, errTransferSubAccountInvalid, "SendAsset should reject a configured subaccount the account does not own")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	for _, tc := range []struct {
		name string
		arg  *SendAssetRequest
		err  error
	}{
		{name: "an unregistered DEX", arg: &SendAssetRequest{Destination: testWithdrawalAddress, SourceDEX: "abc", Token: currency.USDC, Amount: 1}, err: errTransferDEXInvalid},
		{name: "a token that is not the DEX's collateral", arg: &SendAssetRequest{Destination: testWithdrawalAddress, DestinationDEX: "xyz", Token: currency.HYPE, Amount: 1}, err: errTransferTokenInvalid},
	} {
		_, err := ex.SendAsset(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "SendAsset should reject a request with %s", tc.name)
	}
	nonce, err := ex.SendAsset(t.Context(), &SendAssetRequest{Destination: testWithdrawalAddress, SourceDEX: "SPOT", DestinationDEX: "xyz", Token: currency.USDC, Amount: 1.5})
	require.NoError(t, err, "SendAsset must not error")
	exp := `{"type":"sendAsset","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","destination":"` + testWithdrawalAddress + `","sourceDex":"spot","destinationDex":"xyz","token":"USDC","amount":"1.5","fromSubAccount":"` + testSubAccountAddress + `","nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "SendAsset should send from the configured subaccount")
	assert.Empty(t, captured.VaultAddress, "SendAsset should not send a vault address")

	arg := &SendAssetRequest{Destination: testWithdrawalAddress, DestinationDEX: "spot", Token: currency.USDC, Amount: 10}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// Funds sent to the mock responses' placeholder would be lost, so the live test moves 1 USDC between the account's
		// own perpetual and spot balances
		arg.Destination, arg.Amount = liveAccountAddress(t), 1
	}
	nonce, err = newUserSignedTestExchange(t).SendAsset(t.Context(), arg)
	require.NoError(t, err, "SendAsset must not error")
	assertUserSignedNonce(t, nonce)
}

func TestAgentSendAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *AgentSendAssetRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid destination", arg: &AgentSendAssetRequest{Destination: "invalid", Amount: 1}, err: errInvalidAddress},
		{name: "the zero address as destination", arg: &AgentSendAssetRequest{Destination: testZeroAddress, Amount: 1}, err: errInvalidAddress},
		{name: "no amount", arg: &AgentSendAssetRequest{Destination: testWithdrawalAddress}, err: errTransferAmountInvalid},
		{name: "no token", arg: &AgentSendAssetRequest{Destination: testWithdrawalAddress, Amount: 1}, err: errTransferTokenInvalid},
		{name: "a passed expiry", arg: &AgentSendAssetRequest{Destination: testWithdrawalAddress, Token: currency.USDC, Amount: 1, ExpiresAfter: time.Now().Add(-time.Second)}, err: errExpiresAfterPassed},
	} {
		_, err := e.AgentSendAsset(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "AgentSendAsset should reject a request with %s", tc.name)
	}
	valid := &AgentSendAssetRequest{Destination: testWithdrawalAddress, Token: currency.USDC, Amount: 1}
	_, err := newActionServerExchange(t, &accounts.Credentials{Key: testAccountAddress}, new(SignedActionRequest), defaultActionResponse).AgentSendAsset(t.Context(), valid)
	assert.ErrorIs(t, err, errPrivateKeyRequired, "AgentSendAsset should require a private key")
	_, err = newActionServerExchange(t, unownedSubAccountCredentials(), new(SignedActionRequest), defaultActionResponse).AgentSendAsset(t.Context(), valid)
	assert.ErrorIs(t, err, errTransferSubAccountInvalid, "AgentSendAsset should reject a configured subaccount the account does not own")
	_, err = newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), rejectedActionResponse).AgentSendAsset(t.Context(), valid)
	assert.ErrorIs(t, err, errActionResponse, "AgentSendAsset should return a rejected action")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	_, err = ex.AgentSendAsset(t.Context(), &AgentSendAssetRequest{Destination: testWithdrawalAddress, SourceDEX: "xyz", Token: currency.HYPE, Amount: 1})
	assert.ErrorIs(t, err, errTransferTokenInvalid, "AgentSendAsset should reject a token that is not the DEX's collateral")
	expiresAfter := time.Now().Add(time.Minute)
	nonce, err := ex.AgentSendAsset(t.Context(), &AgentSendAssetRequest{Destination: testWithdrawalAddress, SourceDEX: "xyz", DestinationDEX: "spot", Token: currency.USDC, Amount: 2, ExpiresAfter: expiresAfter})
	require.NoError(t, err, "AgentSendAsset must not error")
	exp := `{"type":"agentSendAsset","destination":"` + testWithdrawalAddress + `","sourceDex":"xyz","destinationDex":"spot","token":"USDC","amount":"2","fromSubAccount":"` + testSubAccountAddress + `","nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "AgentSendAsset should sign the request nonce inside the action")
	assert.Equal(t, nonce, captured.Nonce, "AgentSendAsset should send the action's nonce as the request nonce")
	assert.Empty(t, captured.VaultAddress, "AgentSendAsset should not act for the vault")
	assert.Equal(t, new(unixMilli(expiresAfter)), captured.ExpiresAfter, "AgentSendAsset should send the expiry")
	action := AgentSendAssetAction{Type: "agentSendAsset", Destination: testWithdrawalAddress, SourceDEX: "xyz", DestinationDEX: "spot", Token: "USDC", Amount: "2", FromSubAccount: testSubAccountAddress, Nonce: nonce}
	signature, err := signL1Action(testPrivateKey, action, "", nonce, new(unixMilli(expiresAfter)), true)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, signature, captured.Signature, "AgentSendAsset should sign the action holding its nonce, with the expiry and without the vault")

	arg := &AgentSendAssetRequest{Destination: testWithdrawalAddress, SourceDEX: "spot", Token: currency.USDC, Amount: 10}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		credentials, credentialsErr := e.GetCredentials(t.Context())
		require.NoError(t, credentialsErr, "GetCredentials must not error")
		if credentials.SubAccount != "" {
			t.Skip("Live AgentSendAsset moves the account's own funds, so it needs apiCredentials without a subaccount or vault")
		}
		// Hyperliquid requires the destination to be the sending account, so the live test moves 1 USDC between the
		// account's own spot and perpetual balances
		arg.Destination, arg.Amount = liveAccountAddress(t), 1
	}
	nonce, err = newUserSignedTestExchange(t).AgentSendAsset(t.Context(), arg)
	require.NoError(t, err, "AgentSendAsset must not error")
	assertUserSignedNonce(t, nonce)
}

func TestSendToEVMWithData(t *testing.T) {
	t.Parallel()
	valid := SendToEVMWithDataRequest{Token: currency.USDC, Amount: 5, SourceDEX: "spot", DestinationRecipient: testWithdrawalAddress, AddressEncoding: AddressEncodingHex, DestinationChainID: 999, GasLimit: 200000}
	for _, tc := range []struct {
		name   string
		mutate func(*SendToEVMWithDataRequest)
		err    error
	}{
		{name: "no amount", mutate: func(r *SendToEVMWithDataRequest) { r.Amount = 0 }, err: errTransferAmountInvalid},
		{name: "no token", mutate: func(r *SendToEVMWithDataRequest) { r.Token = currency.EMPTYCODE }, err: errTransferTokenInvalid},
		{name: "an unknown address encoding", mutate: func(r *SendToEVMWithDataRequest) { r.AddressEncoding = "bech32" }, err: errAddressEncodingInvalid},
		{name: "an unprefixed hex recipient", mutate: func(r *SendToEVMWithDataRequest) { r.DestinationRecipient = testWithdrawalAddress[2:] }, err: errDestinationRecipientInvalid},
		{name: "an invalid hex recipient", mutate: func(r *SendToEVMWithDataRequest) { r.DestinationRecipient = "0xzz" }, err: errDestinationRecipientInvalid},
		{name: "no base58 recipient", mutate: func(r *SendToEVMWithDataRequest) {
			r.AddressEncoding = AddressEncodingBase58
			r.DestinationRecipient = " "
		}, err: errDestinationRecipientInvalid},
	} {
		arg := valid
		tc.mutate(&arg)
		_, err := e.SendToEVMWithData(t.Context(), &arg)
		assert.ErrorIsf(t, err, tc.err, "SendToEVMWithData should reject a request with %s", tc.name)
	}
	_, err := e.SendToEVMWithData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SendToEVMWithData must reject a nil request")
	_, err = newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).SendToEVMWithData(t.Context(), &valid)
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "SendToEVMWithData must reject a configured subaccount")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse)
	for _, tc := range []struct {
		name   string
		mutate func(*SendToEVMWithDataRequest)
		err    error
	}{
		{name: "an unregistered DEX", mutate: func(r *SendToEVMWithDataRequest) { r.SourceDEX = "abc" }, err: errTransferDEXInvalid},
		{name: "an unknown token", mutate: func(r *SendToEVMWithDataRequest) { r.Token = currency.BTC }, err: errTransferTokenInvalid},
	} {
		arg := valid
		tc.mutate(&arg)
		_, err := ex.SendToEVMWithData(t.Context(), &arg)
		assert.ErrorIsf(t, err, tc.err, "SendToEVMWithData should reject a request with %s", tc.name)
	}
	base58 := valid
	base58.AddressEncoding = AddressEncodingBase58
	base58.DestinationRecipient = "GCTRecipient11111111111111111111111111111111"
	nonce, err := ex.SendToEVMWithData(t.Context(), &base58)
	require.NoError(t, err, "SendToEVMWithData must not error for a base58 recipient")
	exp := `{"type":"sendToEvmWithData","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","token":"USDC:0x6d1e7cde53ba9467b783cb7c530ce054","amount":"5","sourceDex":"spot","destinationRecipient":"GCTRecipient11111111111111111111111111111111","addressEncoding":"base58","destinationChainId":999,"gasLimit":200000,"data":"0x","nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "SendToEVMWithData should send empty data as 0x and keep a base58 recipient")

	upperHex := valid
	upperHex.DestinationRecipient = "0X" + strings.ToUpper(testAccountAddress[2:])
	nonce, err = ex.SendToEVMWithData(t.Context(), &upperHex)
	require.NoError(t, err, "SendToEVMWithData must not error for an upper-case hex recipient")
	exp = `{"type":"sendToEvmWithData","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","token":"USDC:0x6d1e7cde53ba9467b783cb7c530ce054","amount":"5","sourceDex":"spot","destinationRecipient":"` + testAccountAddress + `","addressEncoding":"hex","destinationChainId":999,"gasLimit":200000,"data":"0x","nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "SendToEVMWithData should send a hex recipient in lower case")

	arg := valid
	arg.Data = []byte{0xde, 0xad, 0xbe, 0xef}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// Hyperliquid passes the recipient to coreReceiveWithData on the token's linked contract, so only the tester can
		// choose a recipient that is safe to receive the funds
		if liveEVMDataRecipient == "" {
			t.Skip("Set liveEVMDataRecipient to a HyperEVM address you control to run live sendToEvmWithData tests")
		}
		arg.DestinationRecipient, arg.Amount = liveEVMDataRecipient, 1
	}
	nonce, err = newUserSignedTestExchange(t).SendToEVMWithData(t.Context(), &arg)
	require.NoError(t, err, "SendToEVMWithData must not error")
	assertUserSignedNonce(t, nonce)
}

func TestSendCoreUSDC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SendCoreUSDCRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid destination", arg: &SendCoreUSDCRequest{Destination: "invalid", Amount: 1}, err: errInvalidAddress},
		{name: "the zero address as destination", arg: &SendCoreUSDCRequest{Destination: testZeroAddress, Amount: 1}, err: errInvalidAddress},
		{name: "no amount", arg: &SendCoreUSDCRequest{Destination: testWithdrawalAddress}, err: errTransferAmountInvalid},
	} {
		_, err := e.SendCoreUSDC(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "SendCoreUSDC should reject a request with %s", tc.name)
	}
	_, err := newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).SendCoreUSDC(t.Context(), &SendCoreUSDCRequest{Destination: testWithdrawalAddress, Amount: 1})
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "SendCoreUSDC must reject a configured subaccount")

	var captured SignedActionRequest
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).SendCoreUSDC(t.Context(), &SendCoreUSDCRequest{Destination: testWithdrawalAddress, Amount: 1.5})
	require.NoError(t, err, "SendCoreUSDC must not error")
	exp := `{"type":"usdSend","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","destination":"` + testWithdrawalAddress + `","amount":"1.5","time":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "SendCoreUSDC should send the transfer with a time nonce")

	arg := &SendCoreUSDCRequest{Destination: testWithdrawalAddress, Amount: 25}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		skipLiveWithoutTransferDestination(t)
		arg = &SendCoreUSDCRequest{Destination: liveTransferDestination, Amount: 1}
	}
	nonce, err = newUserSignedTestExchange(t).SendCoreUSDC(t.Context(), arg)
	require.NoError(t, err, "SendCoreUSDC must not error")
	assertUserSignedNonce(t, nonce)
}

func TestSendCoreSpot(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SendCoreSpotRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid destination", arg: &SendCoreSpotRequest{Destination: "invalid", Token: currency.HYPE, Amount: 1}, err: errInvalidAddress},
		{name: "the zero address as destination", arg: &SendCoreSpotRequest{Destination: testZeroAddress, Token: currency.HYPE, Amount: 1}, err: errInvalidAddress},
		{name: "no amount", arg: &SendCoreSpotRequest{Destination: testWithdrawalAddress, Token: currency.HYPE}, err: errTransferAmountInvalid},
	} {
		_, err := e.SendCoreSpot(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "SendCoreSpot should reject a request with %s", tc.name)
	}
	_, err := e.SendCoreSpot(t.Context(), &SendCoreSpotRequest{Destination: testWithdrawalAddress, Amount: 1})
	require.ErrorIs(t, err, errTransferTokenInvalid, "SendCoreSpot must reject an empty token")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SendCoreSpot must report the empty token")
	_, err = newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).SendCoreSpot(t.Context(), &SendCoreSpotRequest{Destination: testWithdrawalAddress, Token: currency.HYPE, Amount: 1})
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "SendCoreSpot must reject a configured subaccount")
	_, err = newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), defaultActionResponse).SendCoreSpot(t.Context(), &SendCoreSpotRequest{Destination: testWithdrawalAddress, Token: currency.BTC, Amount: 1})
	require.ErrorIs(t, err, errTransferTokenInvalid, "SendCoreSpot must reject an unknown token")

	var captured SignedActionRequest
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).SendCoreSpot(t.Context(), &SendCoreSpotRequest{Destination: testWithdrawalAddress, Token: currency.HYPE, Amount: 1.5})
	require.NoError(t, err, "SendCoreSpot must not error")
	exp := `{"type":"spotSend","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","destination":"` + testWithdrawalAddress + `","token":"HYPE:0x0d01dc56dcaaca66ad901c959b4011ec","amount":"1.5","time":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "SendCoreSpot should send the token by its NAME:TOKEN_ID identifier with a time nonce")

	arg := &SendCoreSpotRequest{Destination: testWithdrawalAddress, Token: currency.HYPE, Amount: 2.5}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		skipLiveWithoutTransferDestination(t)
		arg = &SendCoreSpotRequest{Destination: liveTransferDestination, Token: currency.HYPE, Amount: 0.01}
	}
	nonce, err = newUserSignedTestExchange(t).SendCoreSpot(t.Context(), arg)
	require.NoError(t, err, "SendCoreSpot must not error")
	assertUserSignedNonce(t, nonce)
}

func TestWithdrawFromBridge(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *WithdrawFromBridgeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid destination", arg: &WithdrawFromBridgeRequest{Destination: "invalid", Amount: 1}, err: errInvalidAddress},
		{name: "the zero address as destination", arg: &WithdrawFromBridgeRequest{Destination: testZeroAddress, Amount: 1}, err: errInvalidAddress},
		{name: "a negative amount", arg: &WithdrawFromBridgeRequest{Destination: testWithdrawalAddress, Amount: -1}, err: errTransferAmountInvalid},
	} {
		_, err := e.WithdrawFromBridge(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "WithdrawFromBridge should reject a request with %s", tc.name)
	}
	_, err := newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).WithdrawFromBridge(t.Context(), &WithdrawFromBridgeRequest{Destination: testWithdrawalAddress, Amount: 1})
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "WithdrawFromBridge must reject a configured subaccount")

	var captured SignedActionRequest
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).WithdrawFromBridge(t.Context(), &WithdrawFromBridgeRequest{Destination: testWithdrawalAddress, Amount: 1.5})
	require.NoError(t, err, "WithdrawFromBridge must not error")
	exp := `{"type":"withdraw3","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","destination":"` + testWithdrawalAddress + `","amount":"1.5","time":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "WithdrawFromBridge should send the withdrawal with a time nonce")

	arg := &WithdrawFromBridgeRequest{Destination: testWithdrawalAddress, Amount: 25}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		skipLiveWithoutTransferDestination(t)
		// Hyperliquid deducts its $1 withdrawal fee from the amount
		arg = &WithdrawFromBridgeRequest{Destination: liveTransferDestination, Amount: 5}
	}
	nonce, err = newUserSignedTestExchange(t).WithdrawFromBridge(t.Context(), arg)
	require.NoError(t, err, "WithdrawFromBridge must not error")
	assertUserSignedNonce(t, nonce)
}

func TestTransferUSDCBetweenSpotAndPerp(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *TransferUSDCBetweenSpotAndPerpRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no amount", arg: &TransferUSDCBetweenSpotAndPerpRequest{ToPerp: true}, err: errTransferAmountInvalid},
	} {
		_, err := e.TransferUSDCBetweenSpotAndPerp(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "TransferUSDCBetweenSpotAndPerp should reject a request with %s", tc.name)
	}
	valid := &TransferUSDCBetweenSpotAndPerpRequest{Amount: 1}
	_, err := newActionServerExchange(t, agentCredentials(), new(SignedActionRequest), defaultActionResponse).TransferUSDCBetweenSpotAndPerp(t.Context(), valid)
	assert.ErrorIs(t, err, errUserSignedMasterRequired, "TransferUSDCBetweenSpotAndPerp should require the account's own key")
	_, err = newActionServerExchange(t, unownedSubAccountCredentials(), new(SignedActionRequest), defaultActionResponse).TransferUSDCBetweenSpotAndPerp(t.Context(), valid)
	assert.ErrorIs(t, err, errTransferSubAccountInvalid, "TransferUSDCBetweenSpotAndPerp should reject a configured subaccount the account does not own")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	nonce, err := ex.TransferUSDCBetweenSpotAndPerp(t.Context(), &TransferUSDCBetweenSpotAndPerpRequest{Amount: 1, ToPerp: false})
	require.NoError(t, err, "TransferUSDCBetweenSpotAndPerp must not error for a subaccount")
	exp := `{"type":"usdClassTransfer","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","amount":"1 subaccount:` + testSubAccountAddress + `","toPerp":false,"nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "TransferUSDCBetweenSpotAndPerp should name the subaccount in the amount")

	arg := &TransferUSDCBetweenSpotAndPerpRequest{Amount: 100, ToPerp: true}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// The live test moves 1 USDC between the account's own perpetual and spot balances
		arg = &TransferUSDCBetweenSpotAndPerpRequest{Amount: 1}
	}
	nonce, err = newUserSignedTestExchange(t).TransferUSDCBetweenSpotAndPerp(t.Context(), arg)
	require.NoError(t, err, "TransferUSDCBetweenSpotAndPerp must not error")
	assertUserSignedNonce(t, nonce)
}

func TestDepositIntoStaking(t *testing.T) {
	t.Parallel()
	_, err := e.DepositIntoStaking(t.Context(), 0)
	require.ErrorIs(t, err, errStakeAmountInvalid, "DepositIntoStaking must reject a zero amount")
	_, err = newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).DepositIntoStaking(t.Context(), 1)
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "DepositIntoStaking must reject a configured subaccount")

	var captured SignedActionRequest
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).DepositIntoStaking(t.Context(), 1)
	require.NoError(t, err, "DepositIntoStaking must not error")
	assert.JSONEq(t, `{"type":"cDeposit","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","wei":1,"nonce":`+strconv.FormatUint(nonce, 10)+`}`, capturedAction(t, &captured), "DepositIntoStaking should send the amount in wei")

	wei := uint64(100000000)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		wei = 1000000
	}
	nonce, err = newUserSignedTestExchange(t).DepositIntoStaking(t.Context(), wei)
	require.NoError(t, err, "DepositIntoStaking must not error")
	assertUserSignedNonce(t, nonce)
}

func TestWithdrawFromStaking(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFromStaking(t.Context(), 0)
	require.ErrorIs(t, err, errStakeAmountInvalid, "WithdrawFromStaking must reject a zero amount")

	wei := uint64(50000000)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// Withdrawn HYPE passes through a 7 day unstaking queue before it reaches spot
		wei = 1000000
	}
	nonce, err := newUserSignedTestExchange(t).WithdrawFromStaking(t.Context(), wei)
	require.NoError(t, err, "WithdrawFromStaking must not error")
	assertUserSignedNonce(t, nonce)
}

func TestDelegateStake(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DelegateStakeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid validator", arg: &DelegateStakeRequest{Validator: "invalid", Wei: 1}, err: errInvalidAddress},
		{name: "the zero address as validator", arg: &DelegateStakeRequest{Validator: testZeroAddress, Wei: 1}, err: errInvalidAddress},
		{name: "no amount", arg: &DelegateStakeRequest{Validator: testValidatorAddress}, err: errStakeAmountInvalid},
	} {
		_, err := e.DelegateStake(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "DelegateStake should reject a request with %s", tc.name)
	}
	_, err := newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).DelegateStake(t.Context(), &DelegateStakeRequest{Validator: testValidatorAddress, Wei: 1})
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "DelegateStake must reject a configured subaccount")

	var captured SignedActionRequest
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).DelegateStake(t.Context(), &DelegateStakeRequest{Validator: "0X" + strings.ToUpper(testValidatorAddress[2:]), Wei: 2, IsUndelegate: true})
	require.NoError(t, err, "DelegateStake must not error")
	exp := `{"type":"tokenDelegate","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","validator":"` + testValidatorAddress + `","wei":2,"isUndelegate":true,"nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "DelegateStake should send the lower-case validator")

	arg := &DelegateStakeRequest{Validator: testValidatorAddress, Wei: 100000000}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// The mock validator is a placeholder, so the live test delegates a little HYPE to a real validator, where it stays
		// the account's own stake, locked for a day
		arg = &DelegateStakeRequest{Validator: testFoundationValidatorAddress, Wei: 1000000}
	}
	nonce, err = newUserSignedTestExchange(t).DelegateStake(t.Context(), arg)
	require.NoError(t, err, "DelegateStake must not error")
	assertUserSignedNonce(t, nonce)
}

func TestTransferVaultUSDC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *VaultTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid vault", arg: &VaultTransferRequest{VaultAddress: "invalid", USD: 1}, err: errInvalidAddress},
		{name: "the zero address as vault", arg: &VaultTransferRequest{VaultAddress: testZeroAddress, USD: 1}, err: errInvalidAddress},
		{name: "no amount", arg: &VaultTransferRequest{VaultAddress: testVaultAddress}, err: errTransferAmountInvalid},
	} {
		assert.ErrorIsf(t, e.TransferVaultUSDC(t.Context(), tc.arg), tc.err, "TransferVaultUSDC should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	require.NoError(t, ex.TransferVaultUSDC(t.Context(), &VaultTransferRequest{VaultAddress: testVaultAddress, USD: 1000000}), "TransferVaultUSDC must not error")
	assert.JSONEq(t, `{"type":"vaultTransfer","vaultAddress":"`+testVaultAddress+`","isDeposit":false,"usd":1000000}`, capturedAction(t, &captured), "TransferVaultUSDC should send the vault inside the action")
	assert.Empty(t, captured.VaultAddress, "TransferVaultUSDC should not act for the configured subaccount")

	arg := &VaultTransferRequest{VaultAddress: testVaultAddress, IsDeposit: true, USD: 5000000}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		if liveVaultDepositAddress == "" {
			t.Skip("Set liveVaultDepositAddress to a vault you accept leaving 5 USDC in to run live vault transfer tests")
		}
		arg.VaultAddress = liveVaultDepositAddress
	}
	assert.NoError(t, e.TransferVaultUSDC(t.Context(), arg), "TransferVaultUSDC should not error")
}

func TestTransferHIP3BackstopLiquidator(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *HIP3LiquidatorTransferRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no amount", arg: &HIP3LiquidatorTransferRequest{DEX: "xyz"}, err: errLiquidatorTransferInvalid},
		{name: "a part of 1000 quote tokens", arg: &HIP3LiquidatorTransferRequest{DEX: "xyz", Notional: liquidatorTransferStep + 1}, err: errLiquidatorTransferInvalid},
		{name: "the default DEX", arg: &HIP3LiquidatorTransferRequest{Notional: liquidatorTransferStep}, err: errTransferDEXInvalid},
		{name: "spot", arg: &HIP3LiquidatorTransferRequest{DEX: "Spot", Notional: liquidatorTransferStep}, err: errTransferDEXInvalid},
		{name: "a passed expiry", arg: &HIP3LiquidatorTransferRequest{DEX: "xyz", Notional: liquidatorTransferStep, ExpiresAfter: time.Now().Add(-time.Second)}, err: errExpiresAfterPassed},
	} {
		assert.ErrorIsf(t, e.TransferHIP3BackstopLiquidator(t.Context(), tc.arg), tc.err, "TransferHIP3BackstopLiquidator should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	require.ErrorIs(t, ex.TransferHIP3BackstopLiquidator(t.Context(), &HIP3LiquidatorTransferRequest{DEX: "abc", Notional: liquidatorTransferStep}), errTransferDEXInvalid, "TransferHIP3BackstopLiquidator must reject an unregistered DEX")
	require.NoError(t, ex.TransferHIP3BackstopLiquidator(t.Context(), &HIP3LiquidatorTransferRequest{DEX: "xyz", Notional: 2 * liquidatorTransferStep}), "TransferHIP3BackstopLiquidator must not error")
	assert.JSONEq(t, `{"type":"hip3LiquidatorTransfer","dex":"xyz","ntl":2000000000,"isDeposit":false}`, capturedAction(t, &captured), "TransferHIP3BackstopLiquidator should send the transfer")
	assert.Empty(t, captured.VaultAddress, "TransferHIP3BackstopLiquidator should not act for the configured subaccount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
	}
	require.NoError(t, e.TransferHIP3BackstopLiquidator(t.Context(), &HIP3LiquidatorTransferRequest{DEX: "xyz", Notional: liquidatorTransferStep, IsDeposit: true}), "TransferHIP3BackstopLiquidator must not error")
	if !mockTests {
		// Only the deposited principal can be withdrawn, so the live test takes back exactly what it deposited
		assert.NoError(t, e.TransferHIP3BackstopLiquidator(t.Context(), &HIP3LiquidatorTransferRequest{DEX: "xyz", Notional: liquidatorTransferStep}), "TransferHIP3BackstopLiquidator should withdraw the live test's deposit")
	}
}

func TestApproveAgent(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ApproveAgentRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid agent", arg: &ApproveAgentRequest{AgentAddress: "invalid"}, err: errInvalidAddress},
		{name: "the zero address as agent", arg: &ApproveAgentRequest{AgentAddress: testZeroAddress}, err: errInvalidAddress},
		{name: "a long name", arg: &ApproveAgentRequest{AgentAddress: testAgentAddress, AgentName: strings.Repeat("a", maximumAgentNameLength+1)}, err: errAgentNameInvalid},
		{name: "an expiry without a name", arg: &ApproveAgentRequest{AgentAddress: testAgentAddress, ValidUntil: time.Now().Add(time.Hour)}, err: errAgentNameInvalid},
		{name: "a passed expiry", arg: &ApproveAgentRequest{AgentAddress: testAgentAddress, AgentName: "gct", ValidUntil: time.Now().Add(-time.Hour)}, err: errAgentValidUntilInvalid},
		{name: "an expiry beyond 180 days", arg: &ApproveAgentRequest{AgentAddress: testAgentAddress, AgentName: "gct", ValidUntil: time.Now().Add(maximumAgentValidity + time.Hour)}, err: errAgentValidUntilInvalid},
	} {
		_, err := e.ApproveAgent(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "ApproveAgent should reject a request with %s", tc.name)
	}
	_, err := newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).ApproveAgent(t.Context(), &ApproveAgentRequest{AgentAddress: testAgentAddress})
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "ApproveAgent must reject a configured subaccount")

	var captured SignedActionRequest
	validUntil := time.Now().Add(time.Hour)
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).ApproveAgent(t.Context(), &ApproveAgentRequest{AgentAddress: testAgentAddress, AgentName: strings.Repeat("é", maximumAgentNameLength), ValidUntil: validUntil})
	require.NoError(t, err, "ApproveAgent must not error for an expiring agent")
	exp := `{"type":"approveAgent","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","agentAddress":"` + testAgentAddress + `","agentName":"` + strings.Repeat("é", maximumAgentNameLength) + ` valid_until ` + strconv.FormatUint(unixMilli(validUntil), 10) + `","nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "ApproveAgent should append the expiry to the agent name")

	arg := &ApproveAgentRequest{AgentAddress: testAgentAddress, AgentName: "gct"}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// testAgentAddress's private key is published, so the live test approves a fresh key nobody holds, under a fresh
		// name and with a short expiry, which leaves the account's own API wallets in place
		agent := newAgentAddress(t)
		arg = &ApproveAgentRequest{AgentAddress: agent, AgentName: "gct" + agent[2:10], ValidUntil: time.Now().Add(time.Hour)}
	}
	nonce, err = newUserSignedTestExchange(t).ApproveAgent(t.Context(), arg)
	require.NoError(t, err, "ApproveAgent must not error")
	assertUserSignedNonce(t, nonce)
}

func TestApproveBuilderFee(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ApproveBuilderFeeRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an invalid builder", arg: &ApproveBuilderFeeRequest{Builder: "invalid"}, err: errInvalidAddress},
		{name: "the zero address as builder", arg: &ApproveBuilderFeeRequest{Builder: testZeroAddress}, err: errInvalidAddress},
		{name: "a negative rate", arg: &ApproveBuilderFeeRequest{Builder: testBuilderAddress, MaxFeeRate: -0.001}, err: errBuilderFeeRateInvalid},
		{name: "a rate that is not a number", arg: &ApproveBuilderFeeRequest{Builder: testBuilderAddress, MaxFeeRate: math.NaN()}, err: errBuilderFeeRateInvalid},
		{name: "a rate that rounds to zero", arg: &ApproveBuilderFeeRequest{Builder: testBuilderAddress, MaxFeeRate: 1e-13}, err: errBuilderFeeRateInvalid},
	} {
		_, err := e.ApproveBuilderFee(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "ApproveBuilderFee should reject a request with %s", tc.name)
	}
	_, err := newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).ApproveBuilderFee(t.Context(), &ApproveBuilderFeeRequest{Builder: testBuilderAddress})
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "ApproveBuilderFee must reject a configured subaccount")

	var captured SignedActionRequest
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).ApproveBuilderFee(t.Context(), &ApproveBuilderFeeRequest{Builder: testBuilderAddress, MaxFeeRate: 0.001})
	require.NoError(t, err, "ApproveBuilderFee must not error")
	exp := `{"type":"approveBuilderFee","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","maxFeeRate":"0.001%","builder":"` + testBuilderAddress + `","nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "ApproveBuilderFee should send the rate as a percentage")

	arg := &ApproveBuilderFeeRequest{Builder: testBuilderAddress, MaxFeeRate: 0.01}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		arg = liveApprovedBuilderFee(t)
	}
	nonce, err = newUserSignedTestExchange(t).ApproveBuilderFee(t.Context(), arg)
	require.NoError(t, err, "ApproveBuilderFee must not error")
	assertUserSignedNonce(t, nonce)
}

func TestPlaceTWAPOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *TWAPOrderRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "too few minutes", arg: &TWAPOrderRequest{Size: 1, Minutes: minimumTWAPMinutes - 1}, err: errTWAPMinutesInvalid},
		{name: "too many minutes", arg: &TWAPOrderRequest{Size: 1, Minutes: maximumTWAPMinutes + 1}, err: errTWAPMinutesInvalid},
		{name: "no size", arg: &TWAPOrderRequest{Minutes: 30}, err: order.ErrAmountIsInvalid},
		{name: "a size beyond 8 decimals", arg: &TWAPOrderRequest{Size: 0.000000001, Minutes: 30}, err: errWireNumberRounding},
	} {
		_, err := e.PlaceTWAPOrder(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "PlaceTWAPOrder should reject a request with %s", tc.name)
	}
	for response, exp := range map[string]error{
		rejectedActionResponse: errActionResponse,
		`{"status":"ok","response":{"type":"twapOrder","data":{"status":{"error":"Invalid TWAP duration: 1 min(s)"}}}}`: errActionResponse,
		`{"status":"ok","response":{"type":"twapOrder","data":{"status":{}}}}`:                                          errActionStatusMalformed,
		`{"status":"ok","response":{"type":"twapOrder","data":{}}}`:                                                     errActionStatusMalformed,
	} {
		_, err := newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), response).PlaceTWAPOrder(t.Context(), &TWAPOrderRequest{Size: 1, Minutes: 30})
		assert.ErrorIsf(t, err, exp, "PlaceTWAPOrder should reject the response %s", response)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, `{"status":"ok","response":{"type":"twapOrder","data":{"status":{"running":{"twapId":1}}}}}`)
	_, err := ex.PlaceTWAPOrder(t.Context(), &TWAPOrderRequest{Asset: 1, Size: 2, ReduceOnly: true, Minutes: 5})
	require.NoError(t, err, "PlaceTWAPOrder must not error")
	assert.JSONEq(t, `{"type":"twapOrder","twap":{"a":1,"b":false,"s":"2","r":true,"m":5,"t":false}}`, capturedAction(t, &captured), "PlaceTWAPOrder should send the TWAP order")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "PlaceTWAPOrder should act for the configured subaccount")

	arg := &TWAPOrderRequest{Asset: 0, IsBuy: true, Size: 0.01, Minutes: 30, Randomise: true}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		arg = liveTWAPOrderRequest(t)
	}
	status, err := e.PlaceTWAPOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceTWAPOrder must not error")
	if mockTests {
		assert.Equal(t, &TWAPOrderStatus{Running: &TWAPRunningStatus{TWAPID: 77738308}}, status, "PlaceTWAPOrder should return the running TWAP order")
		return
	}
	require.NotNil(t, status.Running, "PlaceTWAPOrder must return the running TWAP order")
	assert.NoError(t, e.CancelTWAPOrder(t.Context(), &TWAPCancelRequest{Asset: 0, TWAPID: status.Running.TWAPID}), "CancelTWAPOrder should cancel the live test's TWAP order")
}

func TestCancelTWAPOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelTWAPOrder(t.Context(), nil), common.ErrNilPointer, "CancelTWAPOrder must reject a nil request")
	require.ErrorIs(t, e.CancelTWAPOrder(t.Context(), &TWAPCancelRequest{}), errTWAPIDRequired, "CancelTWAPOrder must require a TWAP order ID")
	for response, exp := range map[string]error{
		rejectedActionResponse: errActionResponse,
		`{"status":"ok","response":{"type":"twapCancel","data":{"status":{"error":"TWAP was never placed, already canceled, or filled."}}}}`: errActionResponse,
		`{"status":"ok","response":{"type":"twapCancel","data":{"status":"unknown"}}}`:                                                       errUnknownCancelStatus,
	} {
		ex := newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), response)
		assert.ErrorIsf(t, ex.CancelTWAPOrder(t.Context(), &TWAPCancelRequest{TWAPID: 1}), exp, "CancelTWAPOrder should reject the response %s", response)
	}

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, `{"status":"ok","response":{"type":"twapCancel","data":{"status":"success"}}}`)
	require.NoError(t, ex.CancelTWAPOrder(t.Context(), &TWAPCancelRequest{Asset: 1, TWAPID: 2}), "CancelTWAPOrder must not error")
	assert.JSONEq(t, `{"type":"twapCancel","a":1,"t":2}`, capturedAction(t, &captured), "CancelTWAPOrder should send the asset and TWAP order ID")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "CancelTWAPOrder should act for the configured subaccount")
	exp, err := signL1Action(testPrivateKey, TWAPCancelAction{Type: "twapCancel", Asset: 1, TWAPID: 2}, testSubAccountAddress, captured.Nonce, nil, true)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, exp, captured.Signature, "CancelTWAPOrder should sign for the configured subaccount")

	arg := &TWAPCancelRequest{Asset: 0, TWAPID: 77738308}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The recorded TWAP order only exists in the mock responses
		status, err := e.PlaceTWAPOrder(t.Context(), liveTWAPOrderRequest(t))
		require.NoError(t, err, "PlaceTWAPOrder must not error")
		require.NotNil(t, status.Running, "PlaceTWAPOrder must return the running TWAP order")
		arg.TWAPID = status.Running.TWAPID
	}
	assert.NoError(t, e.CancelTWAPOrder(t.Context(), arg), "CancelTWAPOrder should not error")
}

func TestPlaceTrailingStopOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *TrailingStopRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no size", arg: &TrailingStopRequest{RetracementPercent: 1}, err: order.ErrAmountIsInvalid},
		{name: "a size beyond 8 decimals", arg: &TrailingStopRequest{Size: 0.000000001, RetracementPercent: 1}, err: errWireNumberRounding},
		{name: "no retracement", arg: &TrailingStopRequest{Size: 1}, err: errRetracementInvalid},
		{name: "both retracements", arg: &TrailingStopRequest{Size: 1, RetracementPercent: 1, RetracementPrice: 1}, err: errRetracementInvalid},
		{name: "a retracement percentage beyond 8 decimals", arg: &TrailingStopRequest{Size: 1, RetracementPercent: 0.000000001}, err: errRetracementInvalid},
		{name: "a retracement price beyond 8 decimals", arg: &TrailingStopRequest{Size: 1, RetracementPrice: 0.000000001}, err: errRetracementInvalid},
		{name: "a negative activation price", arg: &TrailingStopRequest{Size: 1, RetracementPrice: 1, ActivationPrice: -1}, err: errActivationPriceInvalid},
		{name: "an activation price that is not a number", arg: &TrailingStopRequest{Size: 1, RetracementPrice: 1, ActivationPrice: math.NaN()}, err: errActivationPriceInvalid},
	} {
		_, err := e.PlaceTrailingStopOrder(t.Context(), tc.arg)
		assert.ErrorIsf(t, err, tc.err, "PlaceTrailingStopOrder should reject a request with %s", tc.name)
	}
	for response, exp := range map[string]error{
		rejectedActionResponse:                                         errActionResponse,
		`{"status":"ok","response":null}`:                              errActionStatusMalformed,
		`{"status":"ok","response":{"type":"trailingStop","data":{}}}`: errActionStatusMalformed,
	} {
		_, err := newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), response).PlaceTrailingStopOrder(t.Context(), &TrailingStopRequest{Size: 1, RetracementPrice: 1})
		assert.ErrorIsf(t, err, exp, "PlaceTrailingStopOrder should reject the response %s", response)
	}
	_, err := newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), `{"status":"ok","response":[]}`).PlaceTrailingStopOrder(t.Context(), &TrailingStopRequest{Size: 1, RetracementPrice: 1})
	assert.ErrorContains(t, err, "decoding trailing stop response", "PlaceTrailingStopOrder should report an undecodable response")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, `{"status":"ok","response":{"type":"trailingStop","data":{"oid":1}}}`)
	orderID, err := ex.PlaceTrailingStopOrder(t.Context(), &TrailingStopRequest{Asset: 1, IsBuy: true, Size: 2, RetracementPrice: 500})
	require.NoError(t, err, "PlaceTrailingStopOrder must not error")
	assert.Equal(t, uint64(1), orderID, "PlaceTrailingStopOrder should return the order ID")
	assert.JSONEq(t, `{"type":"trailingStop","asset":1,"isBuy":true,"sz":"2","reduceOnly":false,"retracement":{"px":"500"},"activationPx":null}`, capturedAction(t, &captured), "PlaceTrailingStopOrder should send a null activation price when tracking starts immediately")
	assert.Equal(t, testSubAccountAddress, captured.VaultAddress, "PlaceTrailingStopOrder should act for the configured subaccount")

	arg := &TrailingStopRequest{Asset: 0, Size: 0.001, ReduceOnly: true, RetracementPercent: 1.5, ActivationPrice: 90000}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// A sell only starts tracking once the mark price rises to its activation price, so activating at one and a half
		// times the mid price keeps the reduce-only order from triggering before the test cancels it
		mid := liveBTCMidPrice(t)
		arg.Size, arg.ActivationPrice = liveSize(mid, 11), livePrice(mid, 1.5)
	}
	orderID, err = e.PlaceTrailingStopOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceTrailingStopOrder must not error")
	if mockTests {
		assert.Equal(t, uint64(566563303544), orderID, "PlaceTrailingStopOrder should return the order ID")
		return
	}
	statuses, err := e.CancelOrders(t.Context(), &CancelOrdersRequest{Cancels: []CancelRequest{{Asset: 0, OrderID: orderID}}})
	require.NoError(t, err, "CancelOrders must not error")
	require.Len(t, statuses, 1, "CancelOrders must return the cancel's status")
	assert.Empty(t, statuses[0].Error, "CancelOrders should cancel the live test's trailing stop order")
}

func TestReserveRequestWeight(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *ReserveRequestWeightRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no weight", arg: &ReserveRequestWeightRequest{}, err: errRequestWeightInvalid},
		{name: "an invalid destination", arg: &ReserveRequestWeightRequest{Weight: 1, Destination: "invalid"}, err: errInvalidAddress},
		{name: "the zero address as destination", arg: &ReserveRequestWeightRequest{Weight: 1, Destination: testZeroAddress}, err: errInvalidAddress},
	} {
		assert.ErrorIsf(t, e.ReserveRequestWeight(t.Context(), tc.arg), tc.err, "ReserveRequestWeight should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).ReserveRequestWeight(t.Context(), &ReserveRequestWeightRequest{Weight: 10}), "ReserveRequestWeight must not error")
	assert.JSONEq(t, `{"type":"reserveRequestWeight","weight":10}`, capturedAction(t, &captured), "ReserveRequestWeight should omit an unset destination")
	assert.Empty(t, captured.VaultAddress, "ReserveRequestWeight should not act for the configured subaccount")

	arg := &ReserveRequestWeightRequest{Weight: 1000, Destination: testWithdrawalAddress}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// A destination must be an existing user, so the live test reserves one request, costing 0.0005 USDC, for the
		// signer itself
		arg = &ReserveRequestWeightRequest{Weight: 1}
	}
	assert.NoError(t, e.ReserveRequestWeight(t.Context(), arg), "ReserveRequestWeight should not error")
}

func TestInvalidatePendingNonce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *InvalidatePendingNonceRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "no nonce", arg: &InvalidatePendingNonceRequest{}, err: errNonceRequired},
		{name: "a passed expiry", arg: &InvalidatePendingNonceRequest{Nonce: vectorNonce, ExpiresAfter: time.Now().Add(-time.Second)}, err: errExpiresAfterPassed},
	} {
		assert.ErrorIsf(t, e.InvalidatePendingNonce(t.Context(), tc.arg), tc.err, "InvalidatePendingNonce should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	expiresAfter := time.Now().Add(time.Minute)
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).InvalidatePendingNonce(t.Context(), &InvalidatePendingNonceRequest{Nonce: vectorNonce, ExpiresAfter: expiresAfter}), "InvalidatePendingNonce must not error")
	assert.JSONEq(t, `{"type":"noop"}`, capturedAction(t, &captured), "InvalidatePendingNonce should send a noop")
	assert.Equal(t, uint64(vectorNonce), captured.Nonce, "InvalidatePendingNonce should sign the pending nonce")
	assert.Empty(t, captured.VaultAddress, "InvalidatePendingNonce should not act for the configured subaccount")
	assert.Equal(t, new(unixMilli(expiresAfter)), captured.ExpiresAfter, "InvalidatePendingNonce should send the expiry")
	signature, err := signL1Action(testPrivateKey, EmptyAction{Type: "noop"}, "", vectorNonce, new(unixMilli(expiresAfter)), true)
	require.NoError(t, err, "signL1Action must not error")
	assert.Equal(t, signature, captured.Signature, "InvalidatePendingNonce should sign the noop with the pending nonce and the expiry")

	nonce := uint64(vectorNonce)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// Hyperliquid only accepts nonces within a day or two of its block time, so the live test invalidates the next
		// nonce, which no other action can then take
		nonce = e.nextNonce()
	}
	assert.NoError(t, e.InvalidatePendingNonce(t.Context(), &InvalidatePendingNonceRequest{Nonce: nonce}), "InvalidatePendingNonce should not error")
}

func TestSetUserDEXAbstraction(t *testing.T) {
	t.Parallel()
	_, err := e.SetUserDEXAbstraction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetUserDEXAbstraction must reject a nil request")
	_, err = newActionServerExchange(t, agentCredentials(), new(SignedActionRequest), defaultActionResponse).SetUserDEXAbstraction(t.Context(), &UserDEXAbstractionRequest{})
	require.ErrorIs(t, err, errUserSignedMasterRequired, "SetUserDEXAbstraction must require the account's own key")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	_, err = ex.SetUserDEXAbstraction(t.Context(), &UserDEXAbstractionRequest{User: "invalid"})
	require.ErrorIs(t, err, errInvalidAddress, "SetUserDEXAbstraction must reject an invalid user")
	nonce, err := ex.SetUserDEXAbstraction(t.Context(), &UserDEXAbstractionRequest{})
	require.NoError(t, err, "SetUserDEXAbstraction must not error for the configured subaccount")
	exp := `{"type":"userDexAbstraction","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","user":"` + testSubAccountAddress + `","enabled":false,"nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "SetUserDEXAbstraction should default to the configured subaccount")

	arg := &UserDEXAbstractionRequest{Enabled: true}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// Setting the account's current value leaves it as it was
		enabled := liveDEXAbstraction(t)
		if enabled == nil {
			t.Skip("Live SetUserDEXAbstraction sets the account's current value again, and the live account has never set one")
		}
		arg.Enabled = *enabled
	}
	nonce, err = newUserSignedTestExchange(t).SetUserDEXAbstraction(t.Context(), arg)
	require.NoError(t, err, "SetUserDEXAbstraction must not error")
	assertUserSignedNonce(t, nonce)
}

func TestAgentEnableDEXAbstraction(t *testing.T) {
	t.Parallel()
	var captured SignedActionRequest
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).AgentEnableDEXAbstraction(t.Context()), "AgentEnableDEXAbstraction must not error")
	assert.JSONEq(t, `{"type":"agentEnableDexAbstraction"}`, capturedAction(t, &captured), "AgentEnableDEXAbstraction should send the action")
	assert.Empty(t, captured.VaultAddress, "AgentEnableDEXAbstraction should not act for the configured subaccount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		if liveDEXAbstraction(t) != nil {
			t.Skip("AgentEnableDEXAbstraction only enables a DEX abstraction that was never set, and the live account has set it")
		}
	}
	assert.NoError(t, e.AgentEnableDEXAbstraction(t.Context()), "AgentEnableDEXAbstraction should not error")
}

func TestSetUserAbstraction(t *testing.T) {
	t.Parallel()
	_, err := e.SetUserAbstraction(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetUserAbstraction must reject a nil request")
	for _, abstraction := range []AccountAbstraction{"", AccountAbstractionDefault, AccountAbstractionDEXAbstraction} {
		_, err := e.SetUserAbstraction(t.Context(), &SetUserAbstractionRequest{Abstraction: abstraction})
		assert.ErrorIsf(t, err, errAccountAbstractionInvalid, "SetUserAbstraction should reject abstraction %q", abstraction)
	}
	_, err = newActionServerExchange(t, agentCredentials(), new(SignedActionRequest), defaultActionResponse).SetUserAbstraction(t.Context(), &SetUserAbstractionRequest{Abstraction: AccountAbstractionDisabled})
	require.ErrorIs(t, err, errUserSignedMasterRequired, "SetUserAbstraction must require the account's own key")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	_, err = ex.SetUserAbstraction(t.Context(), &SetUserAbstractionRequest{User: "invalid", Abstraction: AccountAbstractionDisabled})
	require.ErrorIs(t, err, errInvalidAddress, "SetUserAbstraction must reject an invalid user")
	nonce, err := ex.SetUserAbstraction(t.Context(), &SetUserAbstractionRequest{User: testAccountAddress, Abstraction: AccountAbstractionPortfolioMargin})
	require.NoError(t, err, "SetUserAbstraction must not error")
	exp := `{"type":"userSetAbstraction","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","user":"` + testAccountAddress + `","abstraction":"portfolioMargin","nonce":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "SetUserAbstraction should apply to the requested user")

	arg := &SetUserAbstractionRequest{Abstraction: AccountAbstractionUnifiedAccount}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		// Setting the account's current mode leaves it as it was
		arg.Abstraction = liveSettableAbstraction(t)
	}
	nonce, err = newUserSignedTestExchange(t).SetUserAbstraction(t.Context(), arg)
	require.NoError(t, err, "SetUserAbstraction must not error")
	assertUserSignedNonce(t, nonce)
}

func TestAgentSetUserAbstraction(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.AgentSetUserAbstraction(t.Context(), AccountAbstractionDEXAbstraction), errAccountAbstractionInvalid, "AgentSetUserAbstraction must reject an abstraction it cannot set")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse)
	for abstraction, code := range map[AccountAbstraction]string{AccountAbstractionDisabled: "i", AccountAbstractionUnifiedAccount: "u", AccountAbstractionPortfolioMargin: "p"} {
		require.NoErrorf(t, ex.AgentSetUserAbstraction(t.Context(), abstraction), "AgentSetUserAbstraction must not error for %s", abstraction)
		assert.JSONEqf(t, `{"type":"agentSetAbstraction","abstraction":"`+code+`"}`, capturedAction(t, &captured), "AgentSetUserAbstraction should send %s as %s", abstraction, code)
		assert.Emptyf(t, captured.VaultAddress, "AgentSetUserAbstraction should not act for the configured subaccount for %s", abstraction)
	}

	abstraction := AccountAbstractionPortfolioMargin
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// Setting the account's current mode leaves it as it was; portfolio margin also needs a large account
		abstraction = liveSettableAbstraction(t)
	}
	assert.NoError(t, e.AgentSetUserAbstraction(t.Context(), abstraction), "AgentSetUserAbstraction should not error")
}

func TestSplitOutcome(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.SplitOutcome(t.Context(), nil), common.ErrNilPointer, "SplitOutcome must reject a nil request")
	for _, amount := range []float64{0, -1, math.Inf(1)} {
		assert.ErrorIsf(t, e.SplitOutcome(t.Context(), &SplitOutcomeRequest{Amount: amount}), errOutcomeAmountInvalid, "SplitOutcome should reject amount %v", amount)
	}

	var captured SignedActionRequest
	expiresAfter := time.Now().Add(time.Minute)
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).SplitOutcome(t.Context(), &SplitOutcomeRequest{Outcome: 3, Amount: 4, ExpiresAfter: expiresAfter}), "SplitOutcome must not error")
	assert.JSONEq(t, `{"type":"userOutcome","splitOutcome":{"outcome":3,"amount":"4"}}`, capturedAction(t, &captured), "SplitOutcome should send the split")
	assert.Empty(t, captured.VaultAddress, "SplitOutcome should not act for the configured subaccount")
	assert.Equal(t, new(unixMilli(expiresAfter)), captured.ExpiresAfter, "SplitOutcome should send the expiry")

	outcome := uint64(9015)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The recorded outcome may no longer be active
		_, outcome = liveOutcomeQuestion(t)
	}
	require.NoError(t, e.SplitOutcome(t.Context(), &SplitOutcomeRequest{Outcome: outcome, Amount: 10}), "SplitOutcome must not error")
	if !mockTests {
		assert.NoError(t, e.MergeOutcome(t.Context(), &MergeOutcomeRequest{Outcome: outcome, Amount: 10}), "MergeOutcome should merge the live test's split back into quote tokens")
	}
}

func TestMergeOutcome(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.MergeOutcome(t.Context(), nil), common.ErrNilPointer, "MergeOutcome must reject a nil request")
	assert.ErrorIs(t, e.MergeOutcome(t.Context(), &MergeOutcomeRequest{Amount: -1}), errOutcomeAmountInvalid, "MergeOutcome should reject a negative amount")

	var captured SignedActionRequest
	expiresAfter := time.Now().Add(time.Minute)
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).MergeOutcome(t.Context(), &MergeOutcomeRequest{Outcome: 1, ExpiresAfter: expiresAfter}), "MergeOutcome must not error")
	assert.JSONEq(t, `{"type":"userOutcome","mergeOutcome":{"outcome":1,"amount":null}}`, capturedAction(t, &captured), "MergeOutcome should send a null amount to merge the maximum")
	assert.Empty(t, captured.VaultAddress, "MergeOutcome should not act for the configured subaccount")
	assert.Equal(t, new(unixMilli(expiresAfter)), captured.ExpiresAfter, "MergeOutcome should send the expiry")

	arg := &MergeOutcomeRequest{Outcome: 9015}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// Merging takes both sides, so the live test first splits the quote tokens it merges back
		_, outcome := liveOutcomeQuestion(t)
		require.NoError(t, e.SplitOutcome(t.Context(), &SplitOutcomeRequest{Outcome: outcome, Amount: 10}), "SplitOutcome must not error")
		arg = &MergeOutcomeRequest{Outcome: outcome, Amount: 10}
	}
	assert.NoError(t, e.MergeOutcome(t.Context(), arg), "MergeOutcome should not error")
}

func TestMergeQuestion(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.MergeQuestion(t.Context(), nil), common.ErrNilPointer, "MergeQuestion must reject a nil request")
	assert.ErrorIs(t, e.MergeQuestion(t.Context(), &MergeQuestionRequest{Amount: -1}), errOutcomeAmountInvalid, "MergeQuestion should reject a negative amount")

	var captured SignedActionRequest
	expiresAfter := time.Now().Add(time.Minute)
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).MergeQuestion(t.Context(), &MergeQuestionRequest{Question: 2, Amount: 3, ExpiresAfter: expiresAfter}), "MergeQuestion must not error")
	assert.JSONEq(t, `{"type":"userOutcome","mergeQuestion":{"question":2,"amount":"3"}}`, capturedAction(t, &captured), "MergeQuestion should send the merge")
	assert.Empty(t, captured.VaultAddress, "MergeQuestion should not act for the configured subaccount")
	assert.Equal(t, new(unixMilli(expiresAfter)), captured.ExpiresAfter, "MergeQuestion should send the expiry")

	arg := &MergeQuestionRequest{Question: 198, Amount: 5}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// Splitting one outcome and negating its side 1 tokens yields side 0 tokens of every outcome of the question,
		// which the merge turns back into the quote tokens split
		question, outcome := liveOutcomeQuestion(t)
		require.NoError(t, e.SplitOutcome(t.Context(), &SplitOutcomeRequest{Outcome: outcome, Amount: 5}), "SplitOutcome must not error")
		require.NoError(t, e.NegateOutcome(t.Context(), &NegateOutcomeRequest{Question: question, Outcome: outcome, Amount: 5}), "NegateOutcome must not error")
		arg.Question = question
	}
	assert.NoError(t, e.MergeQuestion(t.Context(), arg), "MergeQuestion should not error")
}

func TestNegateOutcome(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.NegateOutcome(t.Context(), nil), common.ErrNilPointer, "NegateOutcome must reject a nil request")
	for _, amount := range []float64{0, -1, math.Inf(1)} {
		assert.ErrorIsf(t, e.NegateOutcome(t.Context(), &NegateOutcomeRequest{Amount: amount}), errOutcomeAmountInvalid, "NegateOutcome should reject amount %v", amount)
	}

	var captured SignedActionRequest
	expiresAfter := time.Now().Add(time.Minute)
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).NegateOutcome(t.Context(), &NegateOutcomeRequest{Question: 5, Outcome: 6, Amount: 7, ExpiresAfter: expiresAfter}), "NegateOutcome must not error")
	assert.JSONEq(t, `{"type":"userOutcome","negateOutcome":{"question":5,"outcome":6,"amount":"7"}}`, capturedAction(t, &captured), "NegateOutcome should send the negation")
	assert.Empty(t, captured.VaultAddress, "NegateOutcome should not act for the configured subaccount")
	assert.Equal(t, new(unixMilli(expiresAfter)), captured.ExpiresAfter, "NegateOutcome should send the expiry")

	arg := &NegateOutcomeRequest{Question: 198, Outcome: 1473, Amount: 2}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// Negating takes side 1 tokens, which a split provides; merging the question afterwards returns the quote tokens
		question, outcome := liveOutcomeQuestion(t)
		require.NoError(t, e.SplitOutcome(t.Context(), &SplitOutcomeRequest{Outcome: outcome, Amount: 2}), "SplitOutcome must not error")
		arg.Question, arg.Outcome = question, outcome
	}
	require.NoError(t, e.NegateOutcome(t.Context(), arg), "NegateOutcome must not error")
	if !mockTests {
		assert.NoError(t, e.MergeQuestion(t.Context(), &MergeQuestionRequest{Question: arg.Question, Amount: 2}), "MergeQuestion should merge the live test's tokens back into quote tokens")
	}
}

func TestClaimRewards(t *testing.T) {
	t.Parallel()
	var captured SignedActionRequest
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).ClaimRewards(t.Context()), "ClaimRewards must not error")
	assert.JSONEq(t, `{"type":"claimRewards"}`, capturedAction(t, &captured), "ClaimRewards should send the action")
	assert.Empty(t, captured.VaultAddress, "ClaimRewards should not act for the configured subaccount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		referral, err := e.GetReferral(t.Context(), liveAccountAddress(t))
		require.NoError(t, err, "GetReferral must not error")
		if referral.UnclaimedRewards <= 0 {
			t.Skip("Live ClaimRewards needs unclaimed rewards, as Hyperliquid rejects a claim without any")
		}
	}
	assert.NoError(t, e.ClaimRewards(t.Context()), "ClaimRewards should not error")
}

func TestBidGossipPriority(t *testing.T) {
	t.Parallel()
	ip := netip.MustParseAddr("203.0.113.7")
	for _, tc := range []struct {
		name string
		arg  *GossipPriorityBidRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "an unknown slot", arg: &GossipPriorityBidRequest{SlotID: 2, IP: ip, MaxGas: 1}, err: errGossipSlotInvalid},
		{name: "no IP address", arg: &GossipPriorityBidRequest{MaxGas: 1}, err: errGossipIPInvalid},
		{name: "no maximum gas", arg: &GossipPriorityBidRequest{IP: ip}, err: errMaxGasInvalid},
	} {
		assert.ErrorIsf(t, e.BidGossipPriority(t.Context(), tc.arg), tc.err, "BidGossipPriority should reject a request with %s", tc.name)
	}

	var captured SignedActionRequest
	require.NoError(t, newActionServerExchange(t, subAccountCredentials(), &captured, defaultActionResponse).BidGossipPriority(t.Context(), &GossipPriorityBidRequest{IP: netip.MustParseAddr("::ffff:1.2.3.4"), MaxGas: 100000000}), "BidGossipPriority must not error")
	assert.JSONEq(t, `{"type":"gossipPriorityBid","slotId":0,"ip":"1.2.3.4","maxGas":100000000}`, capturedAction(t, &captured), "BidGossipPriority should send an IPv4-mapped address as IPv4")
	assert.Empty(t, captured.VaultAddress, "BidGossipPriority should not act for the configured subaccount")

	// 10000000 wei is the auction's 0.1 HYPE minimum price
	arg := &GossipPriorityBidRequest{SlotID: 1, IP: ip, MaxGas: 10000000}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The mock IP address is a documentation address, while priority only helps the IP address peers see
		if liveGossipPriorityIP == "" {
			t.Skip("Set liveGossipPriorityIP to your node's public IP address to run live gossip priority bids")
		}
		liveIP, err := netip.ParseAddr(liveGossipPriorityIP)
		require.NoError(t, err, "liveGossipPriorityIP must be an IP address")
		status, err := e.GetGossipPriorityAuctionStatus(t.Context())
		require.NoError(t, err, "GetGossipPriorityAuctionStatus must not error")
		require.Len(t, status.Auctions, 2, "GetGossipPriorityAuctionStatus must return both slots' auctions")
		// A winning bid is burned, so the live test only bids the minimum, which can win once the falling price reaches it
		if price := status.Auctions[1].CurrentGas.Float64(); price <= 0 || price > 0.1 {
			t.Skipf("The slot 1 auction price of %v HYPE is not open to the live test's 0.1 HYPE bid", price)
		}
		arg.IP = liveIP
	}
	assert.NoError(t, e.BidGossipPriority(t.Context(), arg), "BidGossipPriority should not error")
}

func TestOrderRequestToWire(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  OrderRequest
		err  error
	}{
		{name: "no size", arg: OrderRequest{Price: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}}, err: order.ErrAmountIsInvalid},
		{name: "no price", arg: OrderRequest{Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}}, err: errOrderPriceInvalid},
		{name: "a size beyond 8 decimals", arg: OrderRequest{Price: 1, Size: 0.000000001, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}}, err: errWireNumberRounding},
		{name: "a size that rounds to zero", arg: OrderRequest{Price: 1, Size: 1e-13, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}}, err: errWireNumberRounding},
		{name: "a price beyond 8 decimals", arg: OrderRequest{Price: 0.000000001, Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}}, err: errWireNumberRounding},
		{name: "a price that rounds to zero", arg: OrderRequest{Price: 1e-13, Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}}, err: errWireNumberRounding},
		{name: "an invalid client order ID", arg: OrderRequest{Price: 1, Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, ClientOrderID: "0x1"}, err: errClientOrderIDInvalid},
		{name: "an unknown time in force", arg: OrderRequest{Price: 1, Size: 1, Limit: &LimitOrderType{TimeInForce: "Fok"}}, err: order.ErrUnsupportedTimeInForce},
		{name: "an unknown trigger kind", arg: OrderRequest{Price: 1, Size: 1, Trigger: &TriggerOrderType{TriggerPrice: 1}}, err: errInvalidTriggerKind},
		{name: "no trigger price", arg: OrderRequest{Price: 1, Size: 1, Trigger: &TriggerOrderType{TakeProfitStopLoss: TriggerStopLoss}}, err: errTriggerPriceRequired},
		{name: "a trigger price beyond 8 decimals", arg: OrderRequest{Price: 1, Size: 1, Trigger: &TriggerOrderType{TriggerPrice: 0.000000001, TakeProfitStopLoss: TriggerStopLoss}}, err: errWireNumberRounding},
		{name: "a trigger price that rounds to zero", arg: OrderRequest{Price: 1, Size: 1, Trigger: &TriggerOrderType{TriggerPrice: 1e-13, TakeProfitStopLoss: TriggerStopLoss}}, err: errWireNumberRounding},
		{name: "no order type", arg: OrderRequest{Price: 1, Size: 1}, err: errOrderTypeRequired},
		{name: "both order types", arg: OrderRequest{Price: 1, Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, Trigger: &TriggerOrderType{TriggerPrice: 1, TakeProfitStopLoss: TriggerStopLoss}}, err: errOrderTypeRequired},
	} {
		_, err := tc.arg.ToWire()
		assert.ErrorIsf(t, err, tc.err, "ToWire should reject an order with %s", tc.name)
	}

	limit := OrderRequest{Asset: 10107, IsBuy: true, Price: 95.5, Size: 1.5, ReduceOnly: true, Limit: &LimitOrderType{TimeInForce: TimeInForceALO}, ClientOrderID: strings.ToUpper(testClientOrderID)}
	wire, err := limit.ToWire()
	require.NoError(t, err, "ToWire must not error for a limit order")
	exp := OrderWire{Asset: 10107, IsBuy: true, Price: "95.5", Size: "1.5", ReduceOnly: true, Type: OrderTypeWire{Limit: &LimitOrderTypeWire{TimeInForce: TimeInForceALO}}, ClientOrderID: testClientOrderID}
	assert.Equal(t, exp, wire, "ToWire should format the limit order with a lower-case client order ID")

	trigger := OrderRequest{Asset: 0, Price: 79000, Size: 0.001, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 80000, TakeProfitStopLoss: TriggerStopLoss}}
	wire, err = trigger.ToWire()
	require.NoError(t, err, "ToWire must not error for a trigger order")
	exp = OrderWire{Price: "79000", Size: "0.001", ReduceOnly: true, Type: OrderTypeWire{Trigger: &TriggerOrderTypeWire{IsMarket: true, TriggerPrice: "80000", TakeProfitStopLoss: TriggerStopLoss}}}
	assert.Equal(t, exp, wire, "ToWire should format the trigger order")
}

func TestValidatePriorityGrouping(t *testing.T) {
	t.Parallel()
	immediate := OrderRequest{Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}}
	addLiquidity := OrderRequest{Limit: &LimitOrderType{TimeInForce: TimeInForceALO}}
	for _, tc := range []struct {
		name string
		arg  *PlaceOrdersRequest
		err  error
	}{
		{name: "IOC orders", arg: &PlaceOrdersRequest{Orders: []OrderRequest{immediate, immediate}, PriorityRate: maximumPriorityRate}},
		{name: "ALO orders", arg: &PlaceOrdersRequest{Orders: []OrderRequest{addLiquidity}, Grouping: GroupingNone, PriorityRate: 1}},
		{name: "a rate above the maximum", arg: &PlaceOrdersRequest{Orders: []OrderRequest{immediate}, PriorityRate: maximumPriorityRate + 1}, err: errPriorityRateInvalid},
		{name: "a position TP/SL grouping", arg: &PlaceOrdersRequest{Orders: []OrderRequest{immediate}, Grouping: GroupingPositionTPSL, PriorityRate: 1}, err: errPriorityGroupingInvalid},
		{name: "mixed IOC and ALO orders", arg: &PlaceOrdersRequest{Orders: []OrderRequest{addLiquidity, immediate}, PriorityRate: 1}, err: errPriorityGroupingInvalid},
	} {
		assert.ErrorIsf(t, validatePriorityGrouping(tc.arg), tc.err, "validatePriorityGrouping should return the expected error for %s", tc.name)
	}
}

func TestBuilderFeeWire(t *testing.T) {
	t.Parallel()
	spot := OrderRequest{Asset: 10107}
	perpetual := OrderRequest{Asset: 0}
	builder := OrderRequest{Asset: 110000}
	for _, tc := range []struct {
		name   string
		orders []OrderRequest
		fee    uint64
		err    error
	}{
		{name: "spot orders at the spot maximum", orders: []OrderRequest{spot}, fee: maximumSpotBuilderFee},
		{name: "spot orders above the spot maximum", orders: []OrderRequest{spot}, fee: maximumSpotBuilderFee + 1, err: errBuilderFeeInvalid},
		{name: "a perpetual order above the perpetual maximum", orders: []OrderRequest{spot, perpetual}, fee: maximumPerpetualBuilderFee + 1, err: errBuilderFeeInvalid},
		{name: "a builder DEX order above the perpetual maximum", orders: []OrderRequest{builder}, fee: maximumPerpetualBuilderFee + 1, err: errBuilderFeeInvalid},
		{name: "a perpetual order at the perpetual maximum", orders: []OrderRequest{perpetual}, fee: maximumPerpetualBuilderFee},
	} {
		wire, err := builderFeeWire(&BuilderFee{Builder: testBuilderAddress, Fee: tc.fee}, tc.orders)
		require.ErrorIsf(t, err, tc.err, "builderFeeWire must return the expected error for %s", tc.name)
		if tc.err == nil {
			assert.Equalf(t, BuilderFeeWire{Builder: testBuilderAddress, Fee: tc.fee}, wire, "builderFeeWire should return the fee for %s", tc.name)
		}
	}
	for _, address := range []string{"invalid", testZeroAddress} {
		_, err := builderFeeWire(&BuilderFee{Builder: address}, []OrderRequest{spot})
		assert.ErrorIsf(t, err, errInvalidAddress, "builderFeeWire should reject builder %q", address)
	}
}

func TestModifyOrderID(t *testing.T) {
	t.Parallel()
	orderID, err := modifyOrderID(7, "")
	require.NoError(t, err, "modifyOrderID must not error for an order ID")
	assert.Equal(t, uint64(7), orderID, "modifyOrderID should return the order ID")
	orderID, err = modifyOrderID(0, strings.ToUpper(testClientOrderID))
	require.NoError(t, err, "modifyOrderID must not error for a client order ID")
	assert.Equal(t, testClientOrderID, orderID, "modifyOrderID should return the lower-case client order ID")
	_, err = modifyOrderID(7, testClientOrderID)
	assert.ErrorIs(t, err, errOrderIdentifiersConflict, "modifyOrderID should reject both identifiers")
	_, err = modifyOrderID(0, "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "modifyOrderID should require an identifier")
	_, err = modifyOrderID(0, "0x1")
	assert.ErrorIs(t, err, errClientOrderIDInvalid, "modifyOrderID should reject an invalid client order ID")
}

func TestParseActionStatuses(t *testing.T) {
	t.Parallel()
	_, err := parseActionStatuses[CancelActionStatus](json.RawMessage(`[`), 1)
	assert.ErrorContains(t, err, "decoding action statuses", "parseActionStatuses should report invalid JSON")
	_, err = parseActionStatuses[CancelActionStatus](json.RawMessage(`null`), 1)
	assert.ErrorIs(t, err, errActionStatusMalformed, "parseActionStatuses should reject a missing response")
	_, err = parseActionStatuses[CancelActionStatus](json.RawMessage(`{"data":{"statuses":["success","success"]}}`), 3)
	assert.ErrorIs(t, err, errActionStatusCount, "parseActionStatuses should reject a status count that does not match the batch")
	statuses, err := parseActionStatuses[CancelActionStatus](json.RawMessage(`{"type":"cancel","data":{"statuses":[{"error":"Invalid nonce"}]}}`), 2)
	require.NoError(t, err, "parseActionStatuses must not error for a lone status")
	assert.Equal(t, []CancelActionStatus{{Error: "Invalid nonce"}, {Error: "Invalid nonce"}}, statuses, "parseActionStatuses should apply a lone status to every item")
}

func TestParseActionStatus(t *testing.T) {
	t.Parallel()
	_, err := parseActionStatus[CancelActionStatus](json.RawMessage(`[`))
	assert.ErrorContains(t, err, "decoding action status response", "parseActionStatus should report invalid JSON")
	for _, response := range []string{`null`, `{"data":{}}`, `{"data":{"status":null}}`} {
		_, err = parseActionStatus[CancelActionStatus](json.RawMessage(response))
		assert.ErrorIsf(t, err, errActionStatusMalformed, "parseActionStatus should reject %s", response)
	}
	_, err = parseActionStatus[CancelActionStatus](json.RawMessage(`{"data":{"status":"unknown"}}`))
	assert.ErrorIs(t, err, errUnknownCancelStatus, "parseActionStatus should return the status's decoding error")
	_, err = parseActionStatus[TWAPOrderStatus](json.RawMessage(`{"data":{"status":[]}}`))
	assert.ErrorContains(t, err, "decoding action status: ", "parseActionStatus should report an undecodable status")
	status, err := parseActionStatus[TWAPOrderStatus](json.RawMessage(`{"type":"twapOrder","data":{"status":{"running":{"twapId":2}}}}`))
	require.NoError(t, err, "parseActionStatus must not error")
	assert.Equal(t, &TWAPOrderStatus{Running: &TWAPRunningStatus{TWAPID: 2}}, status, "parseActionStatus should decode the status")
}

func TestOrderActionStatusUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]OrderActionStatus{
		`"waitingForFill"`:    {Waiting: orderStatusWaitingForFill},
		`"waitingForTrigger"`: {Waiting: orderStatusWaitingForTrigger},
		`{"resting":{"oid":1,"cloid":"` + testClientOrderID + `"}}`:                                  {Resting: &RestingOrderStatus{OrderID: 1, ClientOrderID: testClientOrderID}},
		`{"filled":{"totalSz":"0.02","avgPx":"1891.4","oid":2,"cloid":"` + testClientOrderID + `"}}`: {Filled: &FilledOrderStatus{TotalSize: 0.02, AveragePrice: 1891.4, OrderID: 2, ClientOrderID: testClientOrderID}},
		`{"error":"Order must have minimum value of $10."}`:                                          {Error: "Order must have minimum value of $10."},
	} {
		var status OrderActionStatus
		require.NoErrorf(t, json.Unmarshal([]byte(data), &status), "Unmarshal must not error for %s", data)
		assert.Equalf(t, exp, status, "OrderActionStatus should decode %s", data)
	}
	for _, data := range []string{`"unknown"`, `{}`, `{"resting":{"oid":0}}`, `{"filled":{"oid":0}}`, `{"resting":{"oid":1},"error":"e"}`} {
		var status OrderActionStatus
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &status), errActionStatusMalformed, "OrderActionStatus should reject %s", data)
	}
	// Unmarshal rejects an unterminated string before calling the method, so the method is called directly
	assert.Error(t, new(OrderActionStatus).UnmarshalJSON([]byte(`"`)), "OrderActionStatus should reject an unterminated status string")
	var status OrderActionStatus
	assert.Error(t, json.Unmarshal([]byte(`{"resting":1}`), &status), "OrderActionStatus should reject a mistyped resting status")
}

func TestCancelActionStatusUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var status CancelActionStatus
	require.NoError(t, json.Unmarshal([]byte(`"success"`), &status), "Unmarshal must not error for success")
	assert.Equal(t, CancelActionStatus{}, status, "CancelActionStatus should decode success without an error")
	require.NoError(t, json.Unmarshal([]byte(`{"error":"Order was never placed, already canceled, or filled."}`), &status), "Unmarshal must not error for an error")
	assert.Equal(t, CancelActionStatus{Error: "Order was never placed, already canceled, or filled."}, status, "CancelActionStatus should decode the error")
	assert.ErrorIs(t, json.Unmarshal([]byte(`"unknown"`), &status), errUnknownCancelStatus, "CancelActionStatus should reject an unknown status")
	assert.ErrorIs(t, json.Unmarshal([]byte(`{}`), &status), errActionStatusMalformed, "CancelActionStatus should reject an object without an error")
	assert.Error(t, json.Unmarshal([]byte(`{"error":1}`), &status), "CancelActionStatus should reject a mistyped error")
	// Unmarshal rejects an unterminated string before calling the method, so the method is called directly
	assert.Error(t, new(CancelActionStatus).UnmarshalJSON([]byte(`"`)), "CancelActionStatus should reject an unterminated string")
}

func TestTWAPOrderStatusUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]TWAPOrderStatus{
		`{"running":{"twapId":77738308}}`:             {Running: &TWAPRunningStatus{TWAPID: 77738308}},
		`{"error":"Invalid TWAP duration: 1 min(s)"}`: {Error: "Invalid TWAP duration: 1 min(s)"},
	} {
		var status TWAPOrderStatus
		require.NoErrorf(t, json.Unmarshal([]byte(data), &status), "Unmarshal must not error for %s", data)
		assert.Equalf(t, exp, status, "TWAPOrderStatus should decode %s", data)
	}
	for _, data := range []string{`{}`, `{"running":{"twapId":0}}`, `{"running":{"twapId":1},"error":"e"}`} {
		var status TWAPOrderStatus
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &status), errActionStatusMalformed, "TWAPOrderStatus should reject %s", data)
	}
	var status TWAPOrderStatus
	assert.Error(t, json.Unmarshal([]byte(`[]`), &status), "TWAPOrderStatus should reject an array")
}

func TestValidateClientOrderID(t *testing.T) {
	t.Parallel()
	for _, clientOrderID := range []string{testClientOrderID, strings.ToUpper(testClientOrderID)} {
		assert.NoErrorf(t, validateClientOrderID(clientOrderID), "validateClientOrderID should accept %s", clientOrderID)
	}
	for _, clientOrderID := range []string{"", "0x1", testClientOrderID[2:] + "00", "0x" + strings.Repeat("g", 32)} {
		assert.ErrorIsf(t, validateClientOrderID(clientOrderID), errClientOrderIDInvalid, "validateClientOrderID should reject %q", clientOrderID)
	}
}

func TestFormatOutcomeAmount(t *testing.T) {
	t.Parallel()
	amount, err := formatOutcomeAmount(10)
	require.NoError(t, err, "formatOutcomeAmount must not error")
	assert.Equal(t, "10", amount, "formatOutcomeAmount should format the amount")
	for _, value := range []float64{0, -1, math.NaN(), 1e-13} {
		_, err := formatOutcomeAmount(value)
		assert.ErrorIsf(t, err, errOutcomeAmountInvalid, "formatOutcomeAmount should reject %v", value)
	}
}

func TestFormatMergeAmount(t *testing.T) {
	t.Parallel()
	amount, err := formatMergeAmount(0)
	require.NoError(t, err, "formatMergeAmount must not error for zero")
	assert.Nil(t, amount, "formatMergeAmount should merge the maximum for zero")
	amount, err = formatMergeAmount(2.5)
	require.NoError(t, err, "formatMergeAmount must not error")
	assert.Equal(t, new("2.5"), amount, "formatMergeAmount should format the amount")
	_, err = formatMergeAmount(-1)
	assert.ErrorIs(t, err, errOutcomeAmountInvalid, "formatMergeAmount should reject a negative amount")
}

func TestExpiresAfterMilli(t *testing.T) {
	t.Parallel()
	expiresAfter, err := expiresAfterMilli(time.Time{})
	require.NoError(t, err, "expiresAfterMilli must not error for no expiry")
	assert.Nil(t, expiresAfter, "expiresAfterMilli should not set an expiry")
	_, err = expiresAfterMilli(time.Now().Add(-time.Millisecond))
	assert.ErrorIs(t, err, errExpiresAfterPassed, "expiresAfterMilli should reject a passed expiry")
	future := time.Now().Add(time.Minute)
	expiresAfter, err = expiresAfterMilli(future)
	require.NoError(t, err, "expiresAfterMilli must not error for a future expiry")
	assert.Equal(t, new(unixMilli(future)), expiresAfter, "expiresAfterMilli should convert the expiry to Unix milliseconds")
}

func TestUnixMilli(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(1700000000000), unixMilli(time.UnixMilli(1700000000000)), "unixMilli should return Unix milliseconds")
}

func TestSendUserSignedAction(t *testing.T) {
	t.Parallel()
	missing, _ := newRoleServerExchange(t, nil, nil, nil)
	_, err := missing.sendUserSignedAction(t.Context(), signingCredentials(), &USDSendAction{Destination: testWithdrawalAddress, Amount: "1"})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "sendUserSignedAction must report failed authority as an authentication failure")

	var captured SignedActionRequest
	ex := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse)
	_, err = ex.sendUserSignedAction(t.Context(), signingCredentials(), &TokenDelegateAction{Validator: "invalid", Wei: 1})
	require.ErrorIs(t, err, errEIP712Field, "sendUserSignedAction must return a signing error")
	ex.Config.UseSandbox = true
	nonce, err := ex.sendUserSignedAction(t.Context(), signingCredentials(), &USDSendAction{Destination: testWithdrawalAddress, Amount: "1"})
	require.NoError(t, err, "sendUserSignedAction must not error")
	exp := `{"type":"usdSend","signatureChainId":"0x66eee","hyperliquidChain":"Testnet","destination":"` + testWithdrawalAddress + `","amount":"1","time":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "sendUserSignedAction should send the payload for testnet in sandbox mode")
	assert.Equal(t, nonce, captured.Nonce, "sendUserSignedAction should send the action's nonce as the request nonce")
	signature, err := signUserSignedAction(testPrivateKey, "HyperliquidTransaction:UsdSend", []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
		{Name: "destination", Type: "string", Value: testWithdrawalAddress},
		{Name: "amount", Type: "string", Value: "1"},
		{Name: "time", Type: "uint64", Value: nonce},
	})
	require.NoError(t, err, "signUserSignedAction must not error")
	assert.Equal(t, signature, captured.Signature, "sendUserSignedAction should sign the chain, fields and nonce it sends, in order")

	rejected := newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), `{"status":"err","response":"Insufficient balance."}`)
	_, err = rejected.sendUserSignedAction(t.Context(), signingCredentials(), &USDSendAction{Destination: testWithdrawalAddress, Amount: "1"})
	assert.ErrorIs(t, err, errActionResponse, "sendUserSignedAction should return a rejected action")
	assert.True(t, authorityCached(t, rejected), "sendUserSignedAction should keep the validated authority after a rejection unrelated to the signer")

	signerRejected := newActionServerExchange(t, signingCredentials(), new(SignedActionRequest), `{"status":"err","response":"L1 error: User or API Wallet 0x0123456789abcdef0123456789abcdef01234567 does not exist."}`)
	_, err = signerRejected.sendUserSignedAction(t.Context(), signingCredentials(), &USDSendAction{Destination: testWithdrawalAddress, Amount: "1"})
	assert.ErrorIs(t, err, errSignerRejected, "sendUserSignedAction should return a signer rejection")
	assert.False(t, authorityCached(t, signerRejected), "sendUserSignedAction should revalidate authority after a signer rejection")
}

// assertSignedFieldsSent checks a user-signed payload's JSON body carries its type, the signature chain ID and exactly
// the fields it signs, since Hyperliquid recovers the signer by hashing the fields it receives
func assertSignedFieldsSent(t *testing.T, payload userSignedPayload, actionType string, fields []eip712Field) {
	t.Helper()
	exp := map[string]any{"type": actionType, "signatureChainId": userSignedChainIDHex}
	for _, field := range fields {
		exp[field.Name] = field.Value
	}
	expJSON, err := json.Marshal(exp)
	require.NoError(t, err, "Marshal must not error for the expected body")
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err, "Marshal must not error for the payload")
	assert.JSONEq(t, string(expJSON), string(payloadJSON), "The payload should send its type, the signature chain ID and exactly the fields it signs")
}

func TestSendAssetActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &SendAssetAction{Destination: testWithdrawalAddress, Amount: "1.5"}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &SendAssetAction{Type: "sendAsset", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Destination: testWithdrawalAddress, Amount: "1.5", Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the sendAsset type, signature chain ID, chain and nonce")
}

func TestSendAssetActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &SendAssetAction{Type: "sendAsset", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Destination: testWithdrawalAddress, SourceDEX: "spot", DestinationDEX: "xyz", Token: "USDC", Amount: "1.5", FromSubAccount: testSubAccountAddress, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:SendAsset", primaryType, "signingFields should return the SendAsset primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "destination", Type: "string", Value: testWithdrawalAddress},
		{Name: "sourceDex", Type: "string", Value: "spot"},
		{Name: "destinationDex", Type: "string", Value: "xyz"},
		{Name: "token", Type: "string", Value: "USDC"},
		{Name: "amount", Type: "string", Value: "1.5"},
		{Name: "fromSubAccount", Type: "string", Value: testSubAccountAddress},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "sendAsset", fields)
}

func TestSendToEVMWithDataActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &SendToEVMWithDataAction{Token: "USDC:0x6d1e7cde53ba9467b783cb7c530ce054", Amount: "5"}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &SendToEVMWithDataAction{Type: "sendToEvmWithData", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Token: "USDC:0x6d1e7cde53ba9467b783cb7c530ce054", Amount: "5", Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the sendToEvmWithData type, signature chain ID, chain and nonce")
}

func TestSendToEVMWithDataActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &SendToEVMWithDataAction{
		Type:                 "sendToEvmWithData",
		SignatureChainID:     userSignedChainIDHex,
		HyperliquidChain:     "Mainnet",
		Token:                "USDC:0x6d1e7cde53ba9467b783cb7c530ce054",
		Amount:               "5",
		SourceDEX:            "spot",
		DestinationRecipient: testWithdrawalAddress,
		AddressEncoding:      AddressEncodingHex,
		DestinationChainID:   999,
		GasLimit:             200000,
		Data:                 "0xdeadbeef",
		Nonce:                vectorNonce,
	}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:SendToEvmWithData", primaryType, "signingFields should return the SendToEvmWithData primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "token", Type: "string", Value: "USDC:0x6d1e7cde53ba9467b783cb7c530ce054"},
		{Name: "amount", Type: "string", Value: "5"},
		{Name: "sourceDex", Type: "string", Value: "spot"},
		{Name: "destinationRecipient", Type: "string", Value: testWithdrawalAddress},
		{Name: "addressEncoding", Type: "string", Value: AddressEncodingHex},
		{Name: "destinationChainId", Type: "uint32", Value: uint32(999)},
		{Name: "gasLimit", Type: "uint64", Value: uint64(200000)},
		{Name: "data", Type: "bytes", Value: "0xdeadbeef"},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "sendToEvmWithData", fields)
}

func TestUSDSendActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &USDSendAction{Destination: testWithdrawalAddress, Amount: "1.5"}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &USDSendAction{Type: "usdSend", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Destination: testWithdrawalAddress, Amount: "1.5", Time: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the usdSend type, signature chain ID, chain and time nonce")
}

func TestUSDSendActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &USDSendAction{Type: "usdSend", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Destination: testWithdrawalAddress, Amount: "1.5", Time: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:UsdSend", primaryType, "signingFields should return the UsdSend primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "destination", Type: "string", Value: testWithdrawalAddress},
		{Name: "amount", Type: "string", Value: "1.5"},
		{Name: "time", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "usdSend", fields)
}

func TestSpotSendActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &SpotSendAction{Destination: testWithdrawalAddress, Token: "HYPE:0x0d01dc56dcaaca66ad901c959b4011ec"}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &SpotSendAction{Type: "spotSend", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Destination: testWithdrawalAddress, Token: "HYPE:0x0d01dc56dcaaca66ad901c959b4011ec", Time: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the spotSend type, signature chain ID, chain and time nonce")
}

func TestSpotSendActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &SpotSendAction{Type: "spotSend", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Destination: testWithdrawalAddress, Token: "HYPE:0x0d01dc56dcaaca66ad901c959b4011ec", Amount: "1.5", Time: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:SpotSend", primaryType, "signingFields should return the SpotSend primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "destination", Type: "string", Value: testWithdrawalAddress},
		{Name: "token", Type: "string", Value: "HYPE:0x0d01dc56dcaaca66ad901c959b4011ec"},
		{Name: "amount", Type: "string", Value: "1.5"},
		{Name: "time", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "spotSend", fields)
}

func TestWithdraw3ActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &Withdraw3Action{Destination: testWithdrawalAddress, Amount: "1.5"}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &Withdraw3Action{Type: "withdraw3", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Destination: testWithdrawalAddress, Amount: "1.5", Time: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the withdraw3 type, signature chain ID, chain and time nonce")
}

func TestWithdraw3ActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &Withdraw3Action{Type: "withdraw3", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Destination: testWithdrawalAddress, Amount: "1.5", Time: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:Withdraw", primaryType, "signingFields should return the Withdraw primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "destination", Type: "string", Value: testWithdrawalAddress},
		{Name: "amount", Type: "string", Value: "1.5"},
		{Name: "time", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "withdraw3", fields)
}

func TestUSDClassTransferActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &USDClassTransferAction{Amount: "1", ToPerp: true}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &USDClassTransferAction{Type: "usdClassTransfer", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Amount: "1", ToPerp: true, Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the usdClassTransfer type, signature chain ID, chain and nonce")
}

func TestUSDClassTransferActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &USDClassTransferAction{Type: "usdClassTransfer", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Amount: "1 subaccount:" + testSubAccountAddress, ToPerp: true, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:UsdClassTransfer", primaryType, "signingFields should return the UsdClassTransfer primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "amount", Type: "string", Value: "1 subaccount:" + testSubAccountAddress},
		{Name: "toPerp", Type: "bool", Value: true},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "usdClassTransfer", fields)
}

func TestCDepositActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &CDepositAction{Wei: 100000000}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &CDepositAction{Type: "cDeposit", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Wei: 100000000, Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the cDeposit type, signature chain ID, chain and nonce")
}

func TestCDepositActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &CDepositAction{Type: "cDeposit", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Wei: 100000000, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:CDeposit", primaryType, "signingFields should return the CDeposit primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "wei", Type: "uint64", Value: uint64(100000000)},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "cDeposit", fields)
}

func TestCWithdrawActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &CWithdrawAction{Wei: 50000000}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &CWithdrawAction{Type: "cWithdraw", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Wei: 50000000, Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the cWithdraw type, signature chain ID, chain and nonce")
}

func TestCWithdrawActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &CWithdrawAction{Type: "cWithdraw", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Wei: 50000000, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:CWithdraw", primaryType, "signingFields should return the CWithdraw primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "wei", Type: "uint64", Value: uint64(50000000)},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "cWithdraw", fields)
}

func TestTokenDelegateActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &TokenDelegateAction{Validator: testValidatorAddress, Wei: 2}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &TokenDelegateAction{Type: "tokenDelegate", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", Validator: testValidatorAddress, Wei: 2, Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the tokenDelegate type, signature chain ID, chain and nonce")
}

func TestTokenDelegateActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &TokenDelegateAction{Type: "tokenDelegate", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", Validator: testValidatorAddress, Wei: 2, IsUndelegate: true, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:TokenDelegate", primaryType, "signingFields should return the TokenDelegate primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "validator", Type: "address", Value: testValidatorAddress},
		{Name: "wei", Type: "uint64", Value: uint64(2)},
		{Name: "isUndelegate", Type: "bool", Value: true},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "tokenDelegate", fields)
}

func TestApproveAgentActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &ApproveAgentAction{AgentAddress: testAgentAddress, AgentName: "gct"}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &ApproveAgentAction{Type: "approveAgent", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", AgentAddress: testAgentAddress, AgentName: "gct", Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the approveAgent type, signature chain ID, chain and nonce")
}

func TestApproveAgentActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &ApproveAgentAction{Type: "approveAgent", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", AgentAddress: testAgentAddress, AgentName: "gct valid_until 1700000600000", Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:ApproveAgent", primaryType, "signingFields should return the ApproveAgent primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "agentAddress", Type: "address", Value: testAgentAddress},
		{Name: "agentName", Type: "string", Value: "gct valid_until 1700000600000"},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "approveAgent", fields)
}

func TestApproveBuilderFeeActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &ApproveBuilderFeeAction{MaxFeeRate: "0.001%", Builder: testBuilderAddress}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &ApproveBuilderFeeAction{Type: "approveBuilderFee", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", MaxFeeRate: "0.001%", Builder: testBuilderAddress, Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the approveBuilderFee type, signature chain ID, chain and nonce")
}

func TestApproveBuilderFeeActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &ApproveBuilderFeeAction{Type: "approveBuilderFee", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", MaxFeeRate: "0.001%", Builder: testBuilderAddress, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:ApproveBuilderFee", primaryType, "signingFields should return the ApproveBuilderFee primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "maxFeeRate", Type: "string", Value: "0.001%"},
		{Name: "builder", Type: "address", Value: testBuilderAddress},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "approveBuilderFee", fields)
}

func TestUserDEXAbstractionActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &UserDEXAbstractionAction{User: testSubAccountAddress, Enabled: true}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &UserDEXAbstractionAction{Type: "userDexAbstraction", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", User: testSubAccountAddress, Enabled: true, Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the userDexAbstraction type, signature chain ID, chain and nonce")
}

func TestUserDEXAbstractionActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &UserDEXAbstractionAction{Type: "userDexAbstraction", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", User: testSubAccountAddress, Enabled: true, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:UserDexAbstraction", primaryType, "signingFields should return the UserDexAbstraction primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "user", Type: "address", Value: testSubAccountAddress},
		{Name: "enabled", Type: "bool", Value: true},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "userDexAbstraction", fields)
}

func TestUserSetAbstractionActionSetEnvelope(t *testing.T) {
	t.Parallel()
	action := &UserSetAbstractionAction{User: testAccountAddress, Abstraction: AccountAbstractionPortfolioMargin}
	action.setEnvelope("Testnet", vectorNonce)
	exp := &UserSetAbstractionAction{Type: "userSetAbstraction", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Testnet", User: testAccountAddress, Abstraction: AccountAbstractionPortfolioMargin, Nonce: vectorNonce}
	assert.Equal(t, exp, action, "setEnvelope should set the userSetAbstraction type, signature chain ID, chain and nonce")
}

func TestUserSetAbstractionActionSigningFields(t *testing.T) {
	t.Parallel()
	action := &UserSetAbstractionAction{Type: "userSetAbstraction", SignatureChainID: userSignedChainIDHex, HyperliquidChain: "Mainnet", User: testAccountAddress, Abstraction: AccountAbstractionPortfolioMargin, Nonce: vectorNonce}
	primaryType, fields := action.signingFields()
	assert.Equal(t, "HyperliquidTransaction:UserSetAbstraction", primaryType, "signingFields should return the UserSetAbstraction primary type")
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Mainnet"},
		{Name: "user", Type: "address", Value: testAccountAddress},
		{Name: "abstraction", Type: "string", Value: "portfolioMargin"},
		{Name: "nonce", Type: "uint64", Value: uint64(vectorNonce)},
	}
	assert.Equal(t, exp, fields, "signingFields should list the fields in EIP-712 type order")
	assertSignedFieldsSent(t, action, "userSetAbstraction", fields)
}

func TestUserSignedFields(t *testing.T) {
	t.Parallel()
	exp := []eip712Field{
		{Name: "hyperliquidChain", Type: "string", Value: "Testnet"},
		{Name: "amount", Type: "string", Value: "1"},
		{Name: "time", Type: "uint64", Value: uint64(1700000000000)},
	}
	assert.Equal(t, exp, userSignedFields("Testnet", nonceFieldTime, 1700000000000, eip712Field{Name: "amount", Type: "string", Value: "1"}), "userSignedFields should list the chain, the fields and then the nonce")
}

func TestSendAccountSignedAction(t *testing.T) {
	t.Parallel()
	action := &USDSendAction{Destination: testWithdrawalAddress, Amount: "1"}
	_, err := newActionServerExchange(t, subAccountCredentials(), new(SignedActionRequest), defaultActionResponse).sendAccountSignedAction(t.Context(), action)
	require.ErrorIs(t, err, errTransferSubAccountUnsupported, "sendAccountSignedAction must reject a configured subaccount")
	_, err = newActionServerExchange(t, agentCredentials(), new(SignedActionRequest), defaultActionResponse).sendAccountSignedAction(t.Context(), action)
	require.ErrorIs(t, err, errUserSignedMasterRequired, "sendAccountSignedAction must require the account's own key")

	var captured SignedActionRequest
	nonce, err := newActionServerExchange(t, signingCredentials(), &captured, defaultActionResponse).sendAccountSignedAction(t.Context(), action)
	require.NoError(t, err, "sendAccountSignedAction must not error")
	exp := `{"type":"usdSend","signatureChainId":"0x66eee","hyperliquidChain":"Mainnet","destination":"` + testWithdrawalAddress + `","amount":"1","time":` + strconv.FormatUint(nonce, 10) + `}`
	assert.JSONEq(t, exp, capturedAction(t, &captured), "sendAccountSignedAction should send the action signed with the account's key")
	assert.Equal(t, nonce, captured.Nonce, "sendAccountSignedAction should send the action's nonce as the request nonce")
}

func TestGetAccountSigningCredentials(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	setTestCredentials(ex, signingCredentials())
	credentials, err := ex.getAccountSigningCredentials(t.Context())
	require.NoError(t, err, "getAccountSigningCredentials must not error")
	assert.Equal(t, testAccountAddress, credentials.Key, "getAccountSigningCredentials should return the account's credentials")
	setTestCredentials(ex, subAccountCredentials())
	_, err = ex.getAccountSigningCredentials(t.Context())
	assert.ErrorIs(t, err, errTransferSubAccountUnsupported, "getAccountSigningCredentials should reject a configured subaccount")
	setTestCredentials(ex, agentCredentials())
	_, err = ex.getAccountSigningCredentials(t.Context())
	assert.ErrorIs(t, err, errUserSignedMasterRequired, "getAccountSigningCredentials should require the account's own key")
}

func TestPrepareTransfer(t *testing.T) {
	t.Parallel()
	for _, address := range []string{"invalid", testZeroAddress} {
		_, _, err := prepareTransfer(address, 1)
		assert.ErrorIsf(t, err, errInvalidAddress, "prepareTransfer should reject destination %q", address)
	}
	_, _, err := prepareTransfer(testWithdrawalAddress, 0)
	assert.ErrorIs(t, err, errTransferAmountInvalid, "prepareTransfer should reject a zero amount")
	destination, amount, err := prepareTransfer("0X"+strings.ToUpper(testAccountAddress[2:]), 1.5)
	require.NoError(t, err, "prepareTransfer must not error")
	assert.Equal(t, testAccountAddress, destination, "prepareTransfer should normalise the destination to lower case")
	assert.Equal(t, "1.5", amount, "prepareTransfer should format the amount")
}

func TestAbstractionUser(t *testing.T) {
	t.Parallel()
	ex, _ := newRoleServerExchange(t, map[string]string{testSubAccountAddress: `{"role":"subAccount","data":{"master":"` + testAccountAddress + `"}}`}, nil, nil)
	credentials := signingCredentials()
	for _, tc := range []struct {
		subAccount, user string
		exp              string
		err              error
	}{
		{exp: testAccountAddress},
		{subAccount: testSubAccountAddress, exp: testSubAccountAddress},
		{subAccount: testSubAccountAddress, user: "0x" + strings.ToUpper(testSubAccountAddress[2:]), exp: testSubAccountAddress},
		{subAccount: testSubAccountAddress, user: testAccountAddress, exp: testAccountAddress},
		{user: "0x" + strings.ToUpper(testTraderAddress[2:]), exp: testTraderAddress},
		{subAccount: testSubAccountAddress, user: strings.ToUpper(testAccountAddress[2:]), err: errInvalidAddress},
		{subAccount: testVaultAddress, err: errTransferSubAccountInvalid},
		{subAccount: testVaultAddress, user: testVaultAddress, err: errTransferSubAccountInvalid},
	} {
		user, err := ex.abstractionUser(t.Context(), credentials, tc.subAccount, tc.user)
		require.ErrorIsf(t, err, tc.err, "abstractionUser must return the expected error for subaccount %q and user %q", tc.subAccount, tc.user)
		assert.Equalf(t, tc.exp, user, "abstractionUser should resolve subaccount %q and user %q", tc.subAccount, tc.user)
	}
}

func TestGetBridgeChain(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.Equal(t, "Arbitrum", ex.getBridgeChain(), "getBridgeChain should return Arbitrum without a configuration")
	ex.SetDefaults()
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.UseSandbox = true
	ex.Config = cfg
	assert.Equal(t, "Arbitrum Sepolia", ex.getBridgeChain(), "getBridgeChain should return Arbitrum Sepolia in sandbox mode")
}

func TestFormatTransferAmount(t *testing.T) {
	t.Parallel()
	amount, err := formatTransferAmount(1.23)
	require.NoError(t, err, "formatTransferAmount must not error")
	assert.Equal(t, "1.23", amount, "formatTransferAmount should format the amount")
	for _, value := range []float64{0, -1, math.NaN(), 0.000000001, 1e-13} {
		_, err := formatTransferAmount(value)
		assert.ErrorIsf(t, err, errTransferAmountInvalid, "formatTransferAmount should reject %v", value)
	}
}

func TestValidateUserSignedSubAccount(t *testing.T) {
	t.Parallel()
	roles := map[string]string{
		testSubAccountAddress: `{"role":"subAccount","data":{"master":"` + testAccountAddress + `"}}`,
		testVaultAddress:      `{"role":"vault"}`,
		testTraderAddress:     `{"role":"subAccount","data":{"master":"` + testOtherAddress + `"}}`,
	}
	ex, _ := newRoleServerExchange(t, roles, nil, nil)
	credentials := signingCredentials()
	assert.NoError(t, ex.validateUserSignedSubAccount(t.Context(), credentials, ""), "validateUserSignedSubAccount should accept no subaccount")
	assert.NoError(t, ex.validateUserSignedSubAccount(t.Context(), credentials, testSubAccountAddress), "validateUserSignedSubAccount should accept an owned subaccount")
	for _, subAccount := range []string{testVaultAddress, testTraderAddress, testOtherAddress} {
		assert.ErrorIsf(t, ex.validateUserSignedSubAccount(t.Context(), credentials, subAccount), errTransferSubAccountInvalid, "validateUserSignedSubAccount should reject %s", subAccount)
	}
	assert.ErrorIs(t, newUnavailableServerExchange(t).validateUserSignedSubAccount(t.Context(), credentials, testSubAccountAddress), request.ErrBadStatus, "validateUserSignedSubAccount should return a role failure")
}

func TestResolveTransferToken(t *testing.T) {
	t.Parallel()
	_, _, err := e.resolveTransferToken(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, errTransferTokenInvalid, "resolveTransferToken must reject an empty token")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "resolveTransferToken must report the empty token")

	token, identifier, err := e.resolveTransferToken(t.Context(), currency.HYPE)
	require.NoError(t, err, "resolveTransferToken must not error")
	if mockTests {
		exp := &SpotTokenMetadata{Name: currency.HYPE, SizeDecimals: 2, WeiDecimals: 8, Index: 150, TokenID: "0x0d01dc56dcaaca66ad901c959b4011ec", FullName: "Hyperliquid"}
		assert.Equal(t, exp, token, "resolveTransferToken should return the token's metadata")
		assert.Equal(t, "HYPE:0x0d01dc56dcaaca66ad901c959b4011ec", identifier, "resolveTransferToken should return the token's NAME:TOKEN_ID identifier")
	} else {
		assert.True(t, strings.HasPrefix(identifier, "HYPE:0x"), "resolveTransferToken should return the token's NAME:TOKEN_ID identifier")
	}
	_, _, err = e.resolveTransferToken(t.Context(), currency.NewCode("NOTATOKEN"))
	assert.ErrorIs(t, err, errTransferTokenInvalid, "resolveTransferToken should reject an unknown token")
	ambiguous := newInfoServerExchange(t, map[string]string{"spotMeta": `{"tokens":[{"name":"HYPE","index":150},{"name":"HYPE","index":151}]}`}, nil)
	_, _, err = ambiguous.resolveTransferToken(t.Context(), currency.HYPE)
	assert.ErrorIs(t, err, errTransferTokenInvalid, "resolveTransferToken should reject a name shared by two tokens")
	_, _, err = newUnavailableServerExchange(t).resolveTransferToken(t.Context(), currency.HYPE)
	assert.Error(t, err, "resolveTransferToken should return a metadata failure")
}

func TestResolveTransferDEX(t *testing.T) {
	t.Parallel()
	for dex, exp := range map[string]string{"": "", " ": "", "SPOT": "spot", "xyz": "xyz"} {
		resolved, err := e.resolveTransferDEX(t.Context(), dex)
		require.NoErrorf(t, err, "resolveTransferDEX must not error for %q", dex)
		assert.Equalf(t, exp, resolved, "resolveTransferDEX should resolve %q", dex)
	}
	_, err := e.resolveTransferDEX(t.Context(), "notadex")
	assert.ErrorIs(t, err, errTransferDEXInvalid, "resolveTransferDEX should reject an unregistered DEX")
	_, err = newUnavailableServerExchange(t).resolveTransferDEX(t.Context(), "xyz")
	assert.Error(t, err, "resolveTransferDEX should return a registry failure")
}

func TestValidateSendAssetRoute(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		source, destination string
		token               currency.Code
		exp                 [3]string
	}{
		{destination: "spot", token: currency.USDC, exp: [3]string{"", "spot", "USDC"}},
		{source: "spot", destination: "spot", token: currency.HYPE, exp: [3]string{"spot", "spot", "HYPE:0x0d01dc56dcaaca66ad901c959b4011ec"}},
		{source: "xyz", destination: "", token: currency.USDC, exp: [3]string{"xyz", "", "USDC"}},
	} {
		sourceDEX, destinationDEX, token, err := e.validateSendAssetRoute(t.Context(), tc.source, tc.destination, tc.token)
		require.NoErrorf(t, err, "validateSendAssetRoute must not error from %q to %q", tc.source, tc.destination)
		if mockTests {
			assert.Equalf(t, tc.exp, [3]string{sourceDEX, destinationDEX, token}, "validateSendAssetRoute should resolve the route from %q to %q", tc.source, tc.destination)
		}
	}
	for _, tc := range []struct {
		source, destination string
		token               currency.Code
		err                 error
	}{
		{source: "notadex", token: currency.USDC, err: errTransferDEXInvalid},
		{destination: "notadex", token: currency.USDC, err: errTransferDEXInvalid},
		{token: currency.NewCode("NOTATOKEN"), err: errTransferTokenInvalid},
		{destination: "xyz", token: currency.HYPE, err: errTransferTokenInvalid},
	} {
		sourceDEX, destinationDEX, token, err := e.validateSendAssetRoute(t.Context(), tc.source, tc.destination, tc.token)
		assert.ErrorIsf(t, err, tc.err, "validateSendAssetRoute should reject %s from %q to %q", tc.token, tc.source, tc.destination)
		assert.Equalf(t, [3]string{}, [3]string{sourceDEX, destinationDEX, token}, "validateSendAssetRoute should return no route for %s from %q to %q", tc.token, tc.source, tc.destination)
	}
	ex := newInfoServerExchange(t, map[string]string{"spotMeta": `{"tokens":[{"name":"USDC","index":0}]}`, "meta": `{`}, nil)
	sourceDEX, destinationDEX, token, err := ex.validateSendAssetRoute(t.Context(), "", "spot", currency.USDC)
	assert.Error(t, err, "validateSendAssetRoute should return a metadata failure")
	assert.Equal(t, [3]string{}, [3]string{sourceDEX, destinationDEX, token}, "validateSendAssetRoute should return no route after a metadata failure")
}
