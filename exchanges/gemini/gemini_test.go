package gemini

import (
	"net/url"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please enter sandbox API keys & assigned roles for better testing procedures
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

const testCurrency = "btcusd"

var g = &Gemini{}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols(t.Context())
	if err != nil {
		t.Error("GetSymbols() error", err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	pairs, err := g.FetchTradablePairs(t.Context(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if !pairs.Contains(currency.NewPair(currency.STORJ, currency.USD), false) {
		t.Error("expected pair STORJ-USD")
	}
	if !pairs.Contains(currency.NewBTCUSD(), false) {
		t.Error("expected pair BTC-USD")
	}
	if !pairs.Contains(currency.NewPair(currency.AAVE, currency.USD), false) {
		t.Error("expected pair AAVE-BTC")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker(t.Context(), "BTCUSD")
	if err != nil {
		t.Error("GetTicker() error", err)
	}
	_, err = g.GetTicker(t.Context(), "bla")
	if err == nil {
		t.Error("GetTicker() Expected error")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook(t.Context(), testCurrency, url.Values{})
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := g.GetTrades(t.Context(), testCurrency, 0, 0, false)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetNotionalVolume(t *testing.T) {
	t.Parallel()
	_, err := g.GetNotionalVolume(t.Context())
	if err != nil && mockTests {
		t.Error("GetNotionalVolume() error", err)
	} else if err == nil && !mockTests {
		t.Error("GetNotionalVolume() error cannot be nil")
	}
}

func TestGetAuction(t *testing.T) {
	t.Parallel()
	_, err := g.GetAuction(t.Context(), testCurrency)
	if err != nil {
		t.Error("GetAuction() error", err)
	}
}

func TestGetAuctionHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetAuctionHistory(t.Context(), testCurrency, url.Values{})
	if err != nil {
		t.Error("GetAuctionHistory() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := g.NewOrder(t.Context(),
		testCurrency,
		1,
		9000000,
		order.Sell.Lower(),
		"exchange limit")
	if err != nil && mockTests {
		t.Error("NewOrder() error", err)
	} else if err == nil && !mockTests {
		t.Error("NewOrder() error cannot be nil")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := g.CancelExistingOrder(t.Context(), 265555413)
	if err != nil && mockTests {
		t.Error("CancelExistingOrder() error", err)
	} else if err == nil && !mockTests {
		t.Error("CancelExistingOrder() error cannot be nil")
	}
}

func TestCancelExistingOrders(t *testing.T) {
	t.Parallel()
	_, err := g.CancelExistingOrders(t.Context(), false)
	if err != nil && mockTests {
		t.Error("CancelExistingOrders() error", err)
	} else if err == nil && !mockTests {
		t.Error("CancelExistingOrders() error cannot be nil")
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderStatus(t.Context(), 265563260)
	if err != nil && mockTests {
		t.Error("GetOrderStatus() error", err)
	} else if err == nil && !mockTests {
		t.Error("GetOrderStatus() error cannot be nil")
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrders(t.Context())
	if err != nil && mockTests {
		t.Error("GetOrders() error", err)
	} else if err == nil && !mockTests {
		t.Error("GetOrders() error cannot be nil")
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := g.GetTradeHistory(t.Context(), testCurrency, 0)
	if err != nil && mockTests {
		t.Error("GetTradeHistory() error", err)
	} else if err == nil && !mockTests {
		t.Error("GetTradeHistory() error cannot be nil")
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := g.GetTradeVolume(t.Context())
	if err != nil && mockTests {
		t.Error("GetTradeVolume() error", err)
	} else if err == nil && !mockTests {
		t.Error("GetTradeVolume() error cannot be nil")
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	_, err := g.GetBalances(t.Context())
	if err != nil && mockTests {
		t.Error("GetBalances() error", err)
	} else if err == nil && !mockTests {
		t.Error("GetBalances() error cannot be nil")
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := g.GetCryptoDepositAddress(t.Context(), "LOL123", "btc")
	if err == nil {
		t.Error("GetCryptoDepositAddress() Expected error")
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := g.WithdrawCrypto(t.Context(), "LOL123", "btc", 1)
	if err == nil {
		t.Error("WithdrawCrypto() Expected error")
	}
}

func TestPostHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := g.PostHeartbeat(t.Context())
	if err != nil && mockTests {
		t.Error("PostHeartbeat() error", err)
	} else if err == nil && !mockTests {
		t.Error("PostHeartbeat() error cannot be nil")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.LTC.String(),
			"_"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	_, err := g.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(g) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.OfflineTradeFee,
				feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.CryptocurrencyTradeFee,
				feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	if sharedtestvalues.AreAPICredentialsSet(g) || mockTests {
		// CryptocurrencyTradeFee Basic
		if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := g.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText +
		" & " +
		exchange.AutoWithdrawCryptoWithSetupText +
		" & " +
		exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := g.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type: order.AnyType,
		Pairs: []currency.Pair{
			currency.NewPair(currency.LTC, currency.BTC),
		},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := g.GetActiveOrders(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(g) && err != nil && !mockTests:
		t.Errorf("Could not get open orders: %s", err)
	case !sharedtestvalues.AreAPICredentialsSet(g) && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get open orders: %s", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := g.GetOrderHistory(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(g) && err != nil:
		t.Errorf("Could not get order history: %s", err)
	case !sharedtestvalues.AreAPICredentialsSet(g) && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case err != nil && mockTests:
		t.Errorf("Could not get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, g, canManipulateRealOrders)
	}

	orderSubmission := &order.Submit{
		Exchange: g.Name,
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     10,
		Amount:    1,
		ClientID:  "1234234",
		AssetType: asset.Spot,
	}

	response, err := g.SubmitOrder(t.Context(), orderSubmission)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(g) && (err != nil || response.Status != order.New):
		t.Errorf("Order failed to be placed: %v", err)
	case !sharedtestvalues.AreAPICredentialsSet(g) && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, g, canManipulateRealOrders)
	}
	orderCancellation := &order.Cancel{
		OrderID:   "266029865",
		AssetType: asset.Spot,
		Pair:      currency.NewBTCUSDT(),
	}

	err := g.CancelOrder(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(g) && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(g) && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case err != nil && mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, g, canManipulateRealOrders)
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := g.CancelAllOrders(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(g) && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(g) && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, g, canManipulateRealOrders)

	_, err := g.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, g, canManipulateRealOrders)
	}

	_, err := g.WithdrawCryptocurrencyFunds(t.Context(),
		&withdraw.Request{
			Exchange:    g.Name,
			Amount:      -1,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
			Crypto: withdraw.CryptoRequest{
				Address: core.BitcoinDonationAddress,
			},
		})
	if !sharedtestvalues.AreAPICredentialsSet(g) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(g) && err != nil && !mockTests {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
	if sharedtestvalues.AreAPICredentialsSet(g) && err == nil && mockTests {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, g, canManipulateRealOrders)
	}

	withdrawFiatRequest := withdraw.Request{}
	_, err := g.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, g, canManipulateRealOrders)
	}

	withdrawFiatRequest := withdraw.Request{}
	_, err := g.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := g.GetDepositAddress(t.Context(), currency.BTC, "", "")
	if err == nil {
		t.Error("GetDepositAddress error cannot be nil")
	}
}

// TestWsAuth dials websocket, sends login request.
func TestWsAuth(t *testing.T) {
	t.Parallel()
	err := g.API.Endpoints.SetRunningURL(exchange.WebsocketSpot.String(), geminiWebsocketSandboxEndpoint)
	if err != nil {
		t.Error(err)
	}
	if !g.Websocket.IsEnabled() &&
		!g.API.AuthenticatedWebsocketSupport ||
		!sharedtestvalues.AreAPICredentialsSet(g) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	var dialer gws.Dialer
	go g.wsReadData()
	err = g.WsAuth(t.Context(), &dialer)
	if err != nil {
		t.Error(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case resp := <-g.Websocket.DataHandler:
		subAck, ok := resp.(WsSubscriptionAcknowledgementResponse)
		if !ok {
			t.Error("unable to type assert WsSubscriptionAcknowledgementResponse")
		}
		if subAck.Type != "subscription_ack" {
			t.Error("Login failed")
		}
	case <-timer.C:
		t.Error("Expected response")
	}
	timer.Stop()
}

func TestWsMissingRole(t *testing.T) {
	pressXToJSON := []byte(`{
		"result":"error",
		"reason":"MissingRole",
		"message":"To access this endpoint, you need to log in to the website and go to the settings page to assign one of these roles [FundManager] to API key wujB3szN54gtJ4QDhqRJ which currently has roles [Trader]"
	}`)
	if err := g.wsHandleData(pressXToJSON); err == nil {
		t.Error("Expected error")
	}
}

func TestWsOrderEventSubscriptionResponse(t *testing.T) {
	pressXToJSON := []byte(`[ {
  "type" : "accepted",
  "order_id" : "372456298",
  "event_id" : "372456299",
  "client_order_id": "20170208_example", 
  "api_session" : "AeRLptFXoYEqLaNiRwv8",
  "symbol" : "btcusd",
  "side" : "buy",
  "order_type" : "exchange limit",
  "timestamp" : "1478203017",
  "timestampms" : 1478203017455,
  "is_live" : true,
  "is_cancelled" : false,
  "is_hidden" : false,
  "avg_execution_price" : "0", 
  "original_amount" : "14.0296",
  "price" : "1059.54"
} ]`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`[{
    "type": "accepted",
    "order_id": "109535951",
    "event_id": "109535952",
    "api_session": "UI",
    "symbol": "btcusd",
    "side": "buy",
    "order_type": "exchange limit",
    "timestamp": "1547742904",
    "timestampms": 1547742904989,
    "is_live": true,
    "is_cancelled": false,
    "is_hidden": false,
    "original_amount": "1",
    "price": "3592.00",
    "socket_sequence": 13
}]`)
	err = g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`[{
    "type": "accepted",
    "order_id": "109964529",
    "event_id": "109964530",
    "api_session": "UI",
    "symbol": "btcusd",
    "side": "buy",
    "order_type": "market buy",
    "timestamp": "1547756076",
    "timestampms": 1547756076644,
    "is_live": false,
    "is_cancelled": false,
    "is_hidden": false,
    "total_spend": "200.00",
    "socket_sequence": 29
}]`)
	err = g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`[{
    "type": "accepted",
    "order_id": "109964616",
    "event_id": "109964617",
    "api_session": "UI",
    "symbol": "btcusd",
    "side": "sell",
    "order_type": "market sell",
    "timestamp": "1547756893",
    "timestampms": 1547756893937,
    "is_live": true,
    "is_cancelled": false,
    "is_hidden": false,
    "original_amount": "25",
    "socket_sequence": 26
}]`)
	err = g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`[ {
  "type" : "accepted",
  "order_id" : "6321",
  "event_id" : "6322",
  "api_session" : "UI",
  "symbol" : "btcusd",
  "side" : "sell",
  "order_type" : "block_trade",
  "timestamp" : "1478204198",
  "timestampms" : 1478204198989,
  "is_live" : true,
  "is_cancelled" : false,
  "is_hidden" : true,
  "avg_execution_price" : "0",
  "original_amount" : "500",
  "socket_sequence" : 32307
} ]`)
	err = g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsSubAck(t *testing.T) {
	pressXToJSON := []byte(`{
  "type": "subscription_ack",
  "accountId": 5365,
  "subscriptionId": "ws-order-events-5365-b8bk32clqeb13g9tk8p0",
  "symbolFilter": [
    "btcusd"
  ],
  "apiSessionFilter": [
    "UI"
  ],
  "eventTypeFilter": [
    "fill",
    "closed"
  ]
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWsHeartbeat(t *testing.T) {
	pressXToJSON := []byte(`{
  "type": "heartbeat",
  "timestampms": 1547742998508,
  "sequence": 31,
  "trace_id": "b8biknoqppr32kc7gfgg",
  "socket_sequence": 37
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWsUnsubscribe(t *testing.T) {
	pressXToJSON := []byte(`{
    "type": "unsubscribe",
    "subscriptions": [{
        "name": "l2",
        "symbols": [
            "BTCUSD",
            "ETHBTC"
        ]},
        {"name": "candles_1m",
        "symbols": [
            "BTCUSD",
            "ETHBTC"
        ]}
    ]
}`)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTradeData(t *testing.T) {
	pressXToJSON := []byte(`{
  "type": "update",
  "eventId": 5375547515,
  "timestamp": 1547760288,
  "timestampms": 1547760288001,
  "socket_sequence": 15,
  "events": [
    {
      "type": "trade",
      "tid": 5375547515,
      "price": "3632.54",
      "amount": "0.1362819142",
      "makerSide": "ask"
    }
  ]
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWsAuctionData(t *testing.T) {
	pressXToJSON := []byte(`{
    "eventId": 371469414,
    "socket_sequence":4009, 
    "timestamp":1486501200,
    "timestampms":1486501200000,
    "events": [
        {
            "amount": "1406",
            "makerSide": "auction",
            "price": "1048.75",
            "tid": 371469414,
            "type": "trade"
        },
        {
            "auction_price": "1048.75",
            "auction_quantity": "1406",
            "eid": 371469414,
            "highest_bid_price": "1050.98",
            "lowest_ask_price": "1050.99",
            "result": "success",
            "time_ms": 1486501200000,
            "type": "auction_result"
        }
    ],
    "type": "update"
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWsBlockTrade(t *testing.T) {
	pressXToJSON := []byte(`{
   "type":"update",
   "eventId":1111597035,
   "socket_sequence":8,
   "timestamp":1501175027,
   "timestampms":1501175027304,
   "events":[
      {
         "type":"block_trade",
         "tid":1111597035,
         "price":"10100.00",
         "amount":"1000"
      }
   ]
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWSTrade(t *testing.T) {
	pressXToJSON := []byte(`{
		"type": "trade",
		"symbol": "BTCUSD",
		"event_id": 3575573053,
		"timestamp": 151231241,
		"price": "9004.21000000",
		"quantity": "0.09110000",
		"side": "buy"
	}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWsCandles(t *testing.T) {
	pressXToJSON := []byte(`{
  "type": "candles_15m_updates",
  "symbol": "BTCUSD",
  "changes": [
    [
        1561054500000,
        9350.18,
        9358.35,
        9350.18,
        9355.51,
        2.07
    ],
    [
        1561053600000,
        9357.33,
        9357.33,
        9350.18,
        9350.18,
        1.5900161
    ]
  ]
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWsAuctions(t *testing.T) {
	pressXToJSON := []byte(`{
    "eventId": 372481811,
    "socket_sequence":23,
    "timestamp": 1486591200,
    "timestampms": 1486591200000,  
    "events": [
        {
            "auction_open_ms": 1486591200000,
            "auction_time_ms": 1486674000000,
            "first_indicative_ms": 1486673400000,
            "last_cancel_time_ms": 1486673985000,
            "type": "auction_open"
        }
    ],
    "type": "update"
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "update",
    "eventId": 2248762586,
    "timestamp": 1510865640,
    "timestampms": 1510865640122,
    "socket_sequence": 177,
    "events": [
        {
            "type": "auction_indicative",
            "eid": 2248762586,
            "result": "success",
            "time_ms": 1510865640000,
            "highest_bid_price": "7730.69",
            "lowest_ask_price": "7730.7",
            "collar_price": "7730.695",
            "indicative_price": "7750",
            "indicative_quantity": "45.43325086"
        }
    ]
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
    "type": "update",
    "eventId": 2248795680,
    "timestamp": 1510866000,
    "timestampms": 1510866000095,
    "socket_sequence": 2920,
    "events": [
        {
            "type": "trade",
            "tid": 2248795680,
            "price": "7763.23",
            "amount": "55.95",
            "makerSide": "auction"
        },
        {
            "type": "auction_result",
            "eid": 2248795680,
            "result": "success",
            "time_ms": 1510866000000,
            "highest_bid_price": "7769",
            "lowest_ask_price": "7769.01",
            "collar_price": "7769.005",
            "auction_price": "7763.23",
            "auction_quantity": "55.95"
        }
    ]
}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestWsMarketData(t *testing.T) {
	pressXToJSON := []byte(`{
  "type": "update",
  "eventId": 5375461993,
  "socket_sequence": 0,
  "events": [
    {
      "type": "change",
      "reason": "initial",
      "price": "3641.61",
      "delta": "0.83372051",
      "remaining": "0.83372051",
      "side": "bid"
    },
    {
      "type": "change",
      "reason": "initial",
      "price": "3641.62",
      "delta": "4.072",
      "remaining": "4.072",
      "side": "ask"
    }
  ]
}    `)
	err := g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
  "type": "update",
  "eventId": 5375461993,
  "socket_sequence": 0,
  "events": [
    {
      "type": "change",
      "reason": "initial",
      "price": "3641.61",
      "delta": "0.83372051",
      "remaining": "0.83372051",
      "side": "bid"
    },
    {
      "type": "change",
      "reason": "initial",
      "price": "3641.62",
      "delta": "4.072",
      "remaining": "4.072",
      "side": "ask"
    }
  ]
}    `)
	err = g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
  "type": "update",
  "eventId": 5375503736,
  "timestamp": 1547759964,
  "timestampms": 1547759964051,
  "socket_sequence": 2,
  "events": [
    {
      "type": "change",
      "side": "bid",
      "price": "3628.01",
      "remaining": "0",
      "delta": "-2",
      "reason": "cancel"
    }
  ]
}  `)
	err = g.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsError(t *testing.T) {
	tt := []struct {
		Data               []byte
		ErrorExpected      bool
		ErrorShouldContain string
	}{
		{
			Data:          []byte(`{"type": "test"}`),
			ErrorExpected: false,
		},
		{
			Data:          []byte(`{"result": "bla"}`),
			ErrorExpected: false,
		},
		{
			Data:               []byte(`{"result": "error"}`),
			ErrorExpected:      true,
			ErrorShouldContain: "Unhandled websocket error",
		},
		{
			Data:               []byte(`{"result": "error","reason": "InvalidJson"}`),
			ErrorExpected:      true,
			ErrorShouldContain: "InvalidJson",
		},
		{
			Data:               []byte(`{"result": "error","reason": "InvalidJson", "message": "WeAreGoingToTheMoonKirby"}`),
			ErrorExpected:      true,
			ErrorShouldContain: "InvalidJson - WeAreGoingToTheMoonKirby",
		},
	}

	for x := range tt {
		err := g.wsHandleData(tt[x].Data)
		if tt[x].ErrorExpected && err != nil && !strings.Contains(err.Error(), tt[x].ErrorShouldContain) {
			t.Errorf("expected error to contain: %s, got: %s",
				tt[x].ErrorShouldContain, err.Error(),
			)
		} else if !tt[x].ErrorExpected && err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	}
}

func TestWsLevel2Update(t *testing.T) {
	pressXToJSON := []byte(`{
		"type": "l2_updates",
		"symbol": "BTCUSD",
		"changes": [
			[
				"buy",
				"9122.04",
				"0.00121425"
			],
			[
				"sell",
				"9122.07",
				"0.98942292"
			]
		],
		"trades": [{
			"type": "trade",
			"symbol": "BTCUSD",
			"event_id": 169841458,
			"timestamp": 1560976400428,
			"price": "9122.04",
			"quantity": "0.0073173",
			"side": "sell"
		}],
		"auction_events": [{
				"type": "auction_result",
				"symbol": "BTCUSD",
				"time_ms": 1560974400000,
				"result": "success",
				"highest_bid_price": "9150.80",
				"lowest_ask_price": "9150.81",
				"collar_price": "9146.93",
				"auction_price": "9145.00",
				"auction_quantity": "470.10390845"
			},
			{
				"type": "auction_indicative",
				"symbol": "BTCUSD",
				"time_ms": 1560974385000,
				"result": "success",
				"highest_bid_price": "9150.80",
				"lowest_ask_price": "9150.81",
				"collar_price": "9146.84",
				"auction_price": "9134.04",
				"auction_quantity": "389.3094317"
			}
		]
	}`)
	if err := g.wsHandleData(pressXToJSON); err != nil {
		t.Error(err)
	}
}

func TestResponseToStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "accepted", Result: order.New},
		{Case: "booked", Result: order.Active},
		{Case: "fill", Result: order.Filled},
		{Case: "cancelled", Result: order.Cancelled},
		{Case: "cancel_rejected", Result: order.Rejected},
		{Case: "closed", Result: order.Filled},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestResponseToOrderType(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Type
	}
	testCases := []TestCases{
		{Case: "exchange limit", Result: order.Limit},
		{Case: "auction-only limit", Result: order.Limit},
		{Case: "indication-of-interest limit", Result: order.Limit},
		{Case: "market buy", Result: order.Market},
		{Case: "market sell", Result: order.Market},
		{Case: "block_trade", Result: order.Market},
		{Case: "LOL", Result: order.UnknownType},
	}
	for i := range testCases {
		result, _ := stringToOrderType(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Expected: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testCurrency)
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testCurrency)
	if err != nil {
		t.Fatal(err)
	}
	tStart := time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC)
	tEnd := time.Date(2020, 6, 7, 0, 0, 0, 0, time.UTC)
	if !mockTests {
		tStart = time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 15, 0, 0, time.UTC)
	}
	_, err = g.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, tStart, tEnd)
	if err != nil {
		t.Error(err)
	}
}

func TestTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)

	_, err := g.Transfers(t.Context(), currency.BTC, time.Time{}, 100, "", true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)

	_, err := g.GetAccountFundingHistory(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)

	_, err := g.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, g)

	_, err := g.GetOrderInfo(t.Context(), "1234", currency.EMPTYPAIR, asset.Empty)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSymbolDetails(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbolDetails(t.Context(), "all")
	if err != nil {
		t.Error(err)
	}
	_, err = g.GetSymbolDetails(t.Context(), "btcusd")
	if err != nil {
		t.Error(err)
	}
}

func TestSetExchangeOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := g.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	err = g.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	availPairs, err := g.GetAvailablePairs(asset.Spot)
	require.NoError(t, err)
	for x := range availPairs {
		var limit order.MinMaxLevel
		limit, err = g.GetOrderExecutionLimits(asset.Spot, availPairs[x])
		if err != nil {
			t.Fatal(err, availPairs[x])
		}
		if limit == (order.MinMaxLevel{}) {
			t.Fatal("exchange limit should be loaded")
		}
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, g)
	for _, a := range g.GetAssetTypes(false) {
		pairs, err := g.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := g.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	g := new(Gemini)
	require.NoError(t, testexch.Setup(g), "Test instance Setup must not error")
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", ""), currency.NewPairWithDelimiter("ETH", "BTC", "")}
	require.NoError(t, g.CurrencyPairs.StorePairs(asset.Spot, p, false))
	require.NoError(t, g.CurrencyPairs.StorePairs(asset.Spot, p, true))
	subs, err := g.generateSubscriptions()
	require.NoError(t, err)
	exp := subscription.List{
		{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: p, QualifiedChannel: "candles_1d", Interval: kline.OneDay},
		{Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: p, QualifiedChannel: "l2"},
	}
	testsubs.EqualLists(t, exp, subs)

	for _, i := range []kline.Interval{kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.SixHour} {
		subs, err = subscription.List{{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: p, Interval: i}}.ExpandTemplates(g)
		assert.NoErrorf(t, err, "ExpandTemplates should not error on interval %s", i)
		require.NotEmpty(t, subs)
		assert.Equal(t, "candles_"+i.Short(), subs[0].QualifiedChannel)
	}
	_, err = subscription.List{{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: p, Interval: kline.FourHour}}.ExpandTemplates(g)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "ExpandTemplates should error on invalid interval")

	assert.PanicsWithError(t,
		"subscription channel not supported: wibble",
		func() { channelName(&subscription.Subscription{Channel: "wibble"}) },
		"should panic on invalid channel",
	)
}
