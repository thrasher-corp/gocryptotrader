package coinut

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var (
	e          *Exchange
	wsSetupRan bool
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	clientID                = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Coinut Setup error: %s", err)
	}

	if apiKey != "" && clientID != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, clientID, "", "", "", "")
	}

	if err := e.SeedInstruments(context.Background()); err != nil {
		log.Fatalf("Coinut SeedInstruments error: %s", err)
	}

	os.Exit(m.Run())
}

func setupWSTestAuth(t *testing.T) {
	t.Helper()
	if wsSetupRan {
		return
	}

	if !e.Websocket.IsEnabled() && !e.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(e) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	if sharedtestvalues.AreAPICredentialsSet(e) {
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(t.Context(), &dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go e.wsReadData(t.Context())
	err = e.wsAuthenticate(t.Context())
	if err != nil {
		t.Error(err)
	}
	wsSetupRan = true
	_, err = e.WsGetInstruments(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstruments(t *testing.T) {
	_, err := e.GetInstruments(t.Context())
	if err != nil {
		t.Error("GetInstruments() error", err)
	}
}

func TestSeedInstruments(t *testing.T) {
	err := e.SeedInstruments(t.Context())
	if err != nil {
		// No point checking the next condition
		t.Fatal(err)
	}

	if len(e.instrumentMap.GetInstrumentIDs()) == 0 {
		t.Error("instrument map hasn't been seeded")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if apiKey == "" {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.EUR
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.SGD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.CAD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.SGD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.CAD
	if _, err := e.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.WithdrawCryptoViaWebsiteOnlyText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := e.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}
	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	}
}

func TestGetOrderHistoryWrapper(t *testing.T) {
	t.Parallel()
	setupWSTestAuth(t)
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{currency.NewBTCUSD()},
		Side:      order.AnySide,
	}

	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "123",
		AssetType: asset.Spot,
	}
	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(e) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewBTCUSD()
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := e.CancelAllOrders(t.Context(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if apiKey != "" || clientID != "" {
		_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
	} else {
		_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
		require.Error(t, err, "UpdateAccountBalances must error")
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	_, err := e.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected 'Not supported', received %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
	if err == nil {
		t.Error("GetDepositAddress() function unsupported cannot be nil")
	}
}

func TestWsAuthGetAccountBalance(t *testing.T) {
	setupWSTestAuth(t)
	if _, err := e.wsGetAccountBalance(t.Context()); err != nil {
		t.Error(err)
	}
}

func TestWsAuthSubmitOrder(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	ord := WsSubmitOrderParameters{
		Amount:   1,
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
		Price:    1,
		Side:     order.Buy,
	}
	if _, err := e.wsSubmitOrder(t.Context(), &ord); err != nil {
		t.Error(err)
	}
}

func TestWsAuthSubmitOrders(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	order1 := WsSubmitOrderParameters{
		Amount:   1,
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
		Price:    1,
		Side:     order.Buy,
	}
	order2 := WsSubmitOrderParameters{
		Amount:   3,
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  2,
		Price:    2,
		Side:     order.Buy,
	}
	_, err := e.wsSubmitOrders(t.Context(), []WsSubmitOrderParameters{order1, order2})
	if err != nil {
		t.Error(err)
	}
}

func TestWsAuthCancelOrders(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	ord := WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
	}
	order2 := WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  2,
	}
	resp, err := e.wsCancelOrders(t.Context(), []WsCancelOrderParameters{ord, order2})
	if err != nil {
		t.Error(err)
	}
	if resp.Status[0] != "OK" {
		t.Error("Order failed to cancel")
	}
}

func TestWsAuthCancelOrdersWrapper(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	orderDetails := order.Cancel{
		Pair: currency.NewPair(currency.LTC, currency.BTC),
	}
	_, err := e.CancelAllOrders(t.Context(), &orderDetails)
	if err != nil {
		t.Error(err)
	}
}

func TestWsAuthCancelOrder(t *testing.T) {
	setupWSTestAuth(t)
	if !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	ord := &WsCancelOrderParameters{
		Currency: currency.NewPair(currency.LTC, currency.BTC),
		OrderID:  1,
	}
	resp, err := e.wsCancelOrder(t.Context(), ord)
	if err != nil {
		t.Error(err)
	}
	if len(resp.Status) >= 1 && resp.Status[0] != "OK" {
		t.Errorf("Failed to cancel order")
	}
}

func TestWsAuthGetOpenOrders(t *testing.T) {
	setupWSTestAuth(t)
	_, err := e.wsGetOpenOrders(t.Context(), currency.NewPair(currency.LTC, currency.BTC).String())
	if err != nil {
		t.Error(err)
	}
}

func TestCurrencyMapIsLoaded(t *testing.T) {
	t.Parallel()
	var i instrumentMap
	if l := i.IsLoaded(); l {
		t.Error("unexpected result")
	}

	i.Seed("BTCUSD", 1337)
	if l := i.IsLoaded(); !l {
		t.Error("unexpected result")
	}
}

func TestCurrencyMapSeed(t *testing.T) {
	t.Parallel()
	var i instrumentMap
	// Test non-seeded lookups
	if id := i.LookupInstrument(1234); id != "" {
		t.Error("unexpected result")
	}
	if id := i.LookupID("BLAH"); id != 0 {
		t.Error("unexpected result")
	}

	// Test seeded lookups
	i.Seed("BTCUSD", 1337)
	if id := i.LookupID("BTCUSD"); id != 1337 {
		t.Error("unexpected result")
	}
	if id := i.LookupInstrument(1337); id != "BTCUSD" {
		t.Error("unexpected result")
	}

	// Test invalid lookups
	if id := i.LookupInstrument(1234); id != "" {
		t.Error("unexpected result")
	}
	if id := i.LookupID("BLAH"); id != 0 {
		t.Error("unexpected result")
	}

	// Test seeding existing item
	i.Seed("BTCUSD", 1234)
	if id := i.LookupID("BTCUSD"); id != 1337 {
		t.Error("unexpected result")
	}
	if id := i.LookupInstrument(1337); id != "BTCUSD" {
		t.Error("unexpected result")
	}
}

func TestCurrencyMapInstrumentIDs(t *testing.T) {
	t.Parallel()

	var i instrumentMap
	assert.Empty(t, i.GetInstrumentIDs())

	// Seed the instrument map
	i.Seed("BTCUSD", 1234)
	i.Seed("LTCUSD", 1337)

	// Test 2 valid instruments and one invalid
	ids := i.GetInstrumentIDs()
	assert.Contains(t, ids, int64(1234))
	assert.Contains(t, ids, int64(1337))
	assert.NotContains(t, ids, int64(4321))
}

func TestGetNonce(t *testing.T) {
	result := getNonce()
	for range 100000 {
		if result <= 0 || result > coinutMaxNonce {
			t.Fatal("invalid nonce value")
		}
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{
  "buy":
   [ { "count": 1, "price": "751.34500000", "qty": "0.01000000" },
   { "count": 1, "price": "751.00000000", "qty": "0.01000000" },
   { "count": 7, "price": "750.00000000", "qty": "0.07000000" } ],
  "sell":
   [ { "count": 6, "price": "750.58100000", "qty": "0.06000000" },
     { "count": 1, "price": "750.58200000", "qty": "0.01000000" },
     { "count": 1, "price": "750.58300000", "qty": "0.01000000" } ],
  "inst_id": 1,
  "nonce": 704114,
  "total_buy": "67.52345000",
  "total_sell": "0.08000000",
  "reply": "inst_order_book",
  "status": [ "OK" ]
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{ "count": 7,
  "inst_id": 1,
  "price": "750.58100000",
  "qty": "0.07000000",
  "total_buy": "120.06412000",
  "reply": "inst_order_book_update",
  "side": "BUY",
  "trans_id": 169384
}`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
  "highest_buy": "750.58100000",
  "inst_id": 1,
  "last": "752.00000000",
  "lowest_sell": "752.00000000",
  "reply": "inst_tick",
  "timestamp": 1481355058109705,
  "trans_id": 170064,
  "volume": "0.07650000",
  "volume24": "56.07650000"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsGetInstruments(t *testing.T) {
	pressXToJSON := []byte(`{
   "SPOT":{
      "LTCBTC":[
         {
            "base":"LTC",
            "inst_id":1,
            "decimal_places":5,
            "quote":"BTC"
         }
      ],
      "ETHBTC":[
         {
            "quote":"BTC",
            "base":"ETH",
            "decimal_places":5,
            "inst_id":2
         }
      ]
   },
   "nonce":39116,
   "reply":"inst_list",
   "status":[
      "OK"
   ]
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	if e.instrumentMap.LookupID("ETHBTC") != 2 {
		t.Error("Expected id to load")
	}
}

func TestWsTrades(t *testing.T) {
	pressXToJSON := []byte(`{
  "inst_id": 1,
  "nonce": 450319,
  "reply": "inst_trade",
  "status": [
    "OK"
  ],
  "trades": [
    {
      "price": "750.00000000",
      "qty": "0.01000000",
      "side": "BUY",
      "timestamp": 1481193563288963,
      "trans_id": 169514
    },
    {
      "price": "750.00000000",
      "qty": "0.01000000",
      "side": "BUY",
      "timestamp": 1481193345279104,
      "trans_id": 169510
    },
    {
      "price": "750.00000000",
      "qty": "0.01000000",
      "side": "BUY",
      "timestamp": 1481193333272230,
      "trans_id": 169506
    },
    {
      "price": "750.00000000",
      "qty": "0.01000000",
      "side": "BUY",
      "timestamp": 1481193007342874,
      "trans_id": 169502
    }]
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
  "inst_id": 1,
  "price": "750.58300000",
  "reply": "inst_trade_update",
  "side": "BUY",
  "timestamp": 0,
  "trans_id": 169478
}`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsLogin(t *testing.T) {
	pressXToJSON := []byte(`{
   "api_key":"b46e658f-d4c4-433c-b032-093423b1aaa4",
   "country":"NA",
   "email":"tester@test.com",
   "failed_times":0,
   "lang":"en_US",
   "nonce":829055,
   "otp_enabled":false,
   "products_enabled":[
      "SPOT",
      "FUTURE",
      "BINARY_OPTION",
      "OPTION"
   ],
   "reply":"login",
   "session_id":"f8833081-af69-4266-904d-eea088cdcc52",
   "status":[
      "OK"
   ],
   "timezone":"Asia/Singapore",
   "unverified_email":"",
   "username":"test"
}`)
	ctx := accounts.DeployCredentialsToContext(t.Context(),
		&accounts.Credentials{Key: "b46e658f-d4c4-433c-b032-093423b1aaa4", ClientID: "dummy"})
	err := e.wsHandleData(ctx, pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsAccountBalance(t *testing.T) {
	pressXToJSON := []byte(`{
  "nonce": 306254,
  "status": [
    "OK"
  ],
  "BTC": "192.46630415",
  "LTC": "6000.00000000",
  "ETC": "800.00000000",
  "ETH": "496.99938000",
  "floating_pl": "0.00000000",
  "initial_margin": "0.00000000",
  "realized_pl": "0.00000000",
  "maintenance_margin": "0.00000000",
  "equity": "192.46630415",
  "reply": "user_balance",
  "trans_id": 15159032
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrder(t *testing.T) {
	pressXToJSON := []byte(`{
      "nonce":956475,
      "status":[
         "OK"
      ],
      "order_id":1,
      "open_qty": "0.01",
      "inst_id": 490590,
      "qty":"0.01",
      "client_ord_id": 1345,
      "order_price":"750.581",
      "reply":"order_accepted",
      "side":"SELL",
      "trans_id":127303
   }`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(` {
    "commission": {
      "amount": "0.00799000",
      "currency": "USD"
    },
    "fill_price": "799.00000000",
    "fill_qty": "0.01000000",
    "nonce": 956475,
    "order": {
      "client_ord_id": 12345,
      "inst_id": 490590,
      "open_qty": "0.00000000",
      "order_id": 721923,
      "price": "748.00000000",
      "qty": "0.01000000",
      "side": "SELL",
      "timestamp": 1482903034617491
    },
    "reply": "order_filled",
    "status": [
      "OK"
    ],
    "timestamp": 1482903034617491,
    "trans_id": 20859252
  }`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(` {
    "nonce": 275825,
    "status": [
        "OK"
    ],
    "order_id": 7171,
    "open_qty": "100000.00000000",
    "price": "750.60000000",
    "inst_id": 490590,
    "reasons": [
        "NOT_ENOUGH_BALANCE"
    ],
    "client_ord_id": 4,
    "timestamp": 1482080535098689,
    "reply": "order_rejected",
    "qty": "100000.00000000",
    "side": "BUY",
    "trans_id": 3282993
}`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err == nil {
		t.Error("Expected not enough balance error")
	}
}

func TestWsOrders(t *testing.T) {
	pressXToJSON := []byte(`[
  {
    "nonce": 621701,
    "status": [
      "OK"
    ],
    "order_id": 331,
    "open_qty": "0.01000000",
    "price": "750.58100000",
    "inst_id": 490590,
    "client_ord_id": 1345,
    "timestamp": 1490713990542441,
    "reply": "order_accepted",
    "qty": "0.01000000",
    "side": "SELL",
    "trans_id": 15155495
  },
  {
    "nonce": 621701,
    "status": [
      "OK"
    ],
    "order_id": 332,
    "open_qty": "0.01000000",
    "price": "750.32100000",
    "inst_id": 490590,
    "client_ord_id": 50001346,
    "timestamp": 1490713990542441,
    "reply": "order_accepted",
    "qty": "0.01000000",
    "side": "BUY",
    "trans_id": 15155497
  }
]`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOpenOrders(t *testing.T) {
	pressXToJSON := []byte(`{
    "nonce": 1234,
    "reply": "user_open_orders",
    "status": [
        "OK"
    ],
    "orders": [
        {
            "order_id": 35,
            "open_qty": "0.01000000",
            "price": "750.58200000",
            "inst_id": 490590,
            "client_ord_id": 4,
            "timestamp": 1481138766081720,
            "qty": "0.01000000",
            "side": "BUY"
        },
        {
            "order_id": 30,
            "open_qty": "0.01000000",
            "price": "750.58100000",
            "inst_id": 490590,
            "client_ord_id": 5,
            "timestamp": 1481137697919617,
            "qty": "0.01000000",
            "side": "BUY"
        }
    ]
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsCancelOrder(t *testing.T) {
	pressXToJSON := []byte(` {
    "nonce": 547201,
    "reply": "cancel_order",
    "order_id": 1,
    "client_ord_id": 13556,
    "status": [
      "OK"
    ]
  }`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsCancelOrders(t *testing.T) {
	pressXToJSON := []byte(`{
  "nonce": 547201,
  "reply": "cancel_orders",
  "status": [
    "OK"
  ],
  "results": [
    {
      "order_id": 329,
      "status": "OK",
      "inst_id": 490590,
      "client_ord_id": 13561
    },
    {
      "order_id": 332,
      "status": "OK",
      "inst_id": 490590,
      "client_ord_id": 13562
    }
  ],
  "trans_id": 15166063
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderHistory(t *testing.T) {
	pressXToJSON := []byte(`{
  "nonce": 326181,
  "reply": "trade_history",
  "status": [
    "OK"
  ],
  "total_number": 261,
  "trades": [
    {
      "commission": {
        "amount": "0.00000100",
        "currency": "BTC"
      },
      "order": {
        "client_ord_id": 297125564,
        "inst_id": 490590,
        "open_qty": "0.00000000",
        "order_id": 721327,
        "price": "1.00000000",
        "qty": "0.00100000",
        "side": "SELL",
        "timestamp": 1482490337560987
      },
      "fill_price": "1.00000000",
      "fill_qty": "0.00100000",
      "timestamp": 1482490337560987,
      "trans_id": 10020695
    },
    {
      "commission": {
        "amount": "0.00000100",
        "currency": "BTC"
      },
      "order": {
        "client_ord_id": 297118937,
        "inst_id": 490590,
        "open_qty": "0.00000000",
        "order_id": 721326,
        "price": "1.00000000",
        "qty": "0.00100000",
        "side": "SELL",
        "timestamp": 1482490330557949
      },
      "fill_price": "1.00000000",
      "fill_qty": "0.00100000",
      "timestamp": 1482490330557949,
      "trans_id": 10020514
    }
  ]
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestStringToStatus(t *testing.T) {
	type TestCases struct {
		Case     string
		Quantity float64
		Result   order.Status
	}
	testCases := []TestCases{
		{Case: "order_accepted", Result: order.Active},
		{Case: "order_filled", Quantity: 1, Result: order.PartiallyFilled},
		{Case: "order_rejected", Result: order.Rejected},
		{Case: "order_filled", Result: order.Filled},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case, testCases[i].Quantity)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("LTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewBTCUSD(),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}
