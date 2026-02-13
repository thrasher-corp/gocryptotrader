package gemini

import (
	"net/url"
	"strings"
	"testing"
	"time"

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

var e *Exchange

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbols(t.Context())
	assert.NoError(t, err, "GetSymbols should not error")
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	pairs, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.True(t, pairs.Contains(currency.NewPair(currency.STORJ, currency.USD), false), "tradable pairs should contain STORJ-USD")
	assert.True(t, pairs.Contains(currency.NewBTCUSD(), false), "tradable pairs should contain BTC-USD")
	assert.True(t, pairs.Contains(currency.NewPair(currency.AAVE, currency.USD), false), "tradable pairs should contain AAVE-USD")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), "BTCUSD")
	assert.NoError(t, err, "GetTicker should not error")
	_, err = e.GetTicker(t.Context(), "bla")
	assert.Error(t, err, "GetTicker should error for invalid symbol")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), testCurrency, url.Values{})
	assert.NoError(t, err, "GetOrderbook should not error")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), testCurrency, 0, 0, false)
	assert.NoError(t, err, "GetTrades should not error")
}

func TestGetNotionalVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetNotionalVolume(t.Context())
	if err != nil && mockTests {
		assert.NoError(t, err, "GetNotionalVolume should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "GetNotionalVolume should error when credentials are unset")
	}
}

func TestGetAuction(t *testing.T) {
	t.Parallel()
	_, err := e.GetAuction(t.Context(), testCurrency)
	assert.NoError(t, err, "GetAuction should not error")
}

func TestGetAuctionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetAuctionHistory(t.Context(), testCurrency, url.Values{})
	assert.NoError(t, err, "GetAuctionHistory should not error")
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrder(t.Context(),
		testCurrency,
		1,
		9000000,
		order.Sell.Lower(),
		"exchange limit")
	if err != nil && mockTests {
		assert.NoError(t, err, "NewOrder should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "NewOrder should error when credentials are unset")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrder(t.Context(), 265555413)
	if err != nil && mockTests {
		assert.NoError(t, err, "CancelExistingOrder should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "CancelExistingOrder should error when credentials are unset")
	}
}

func TestCancelExistingOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrders(t.Context(), false)
	if err != nil && mockTests {
		assert.NoError(t, err, "CancelExistingOrders should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "CancelExistingOrders should error when credentials are unset")
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderStatus(t.Context(), 265563260)
	if err != nil && mockTests {
		assert.NoError(t, err, "GetOrderStatus should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "GetOrderStatus should error when credentials are unset")
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrders(t.Context())
	if err != nil && mockTests {
		assert.NoError(t, err, "GetOrders should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "GetOrders should error when credentials are unset")
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeHistory(t.Context(), testCurrency, 0)
	if err != nil && mockTests {
		assert.NoError(t, err, "GetTradeHistory should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "GetTradeHistory should error when credentials are unset")
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeVolume(t.Context())
	if err != nil && mockTests {
		assert.NoError(t, err, "GetTradeVolume should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "GetTradeVolume should error when credentials are unset")
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	_, err := e.GetBalances(t.Context())
	if err != nil && mockTests {
		assert.NoError(t, err, "GetBalances should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "GetBalances should error when credentials are unset")
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetCryptoDepositAddress(t.Context(), "LOL123", "btc")
	assert.Error(t, err, "GetCryptoDepositAddress should error for invalid account")
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCrypto(t.Context(), "LOL123", "btc", 1)
	assert.Error(t, err, "WithdrawCrypto should error for invalid account")
}

func TestPostHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := e.PostHeartbeat(t.Context())
	if err != nil && mockTests {
		assert.NoError(t, err, "PostHeartbeat should not error in mock mode")
	} else if err == nil && !mockTests {
		assert.Error(t, err, "PostHeartbeat should error when credentials are unset")
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

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err)
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType)
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()
	if sharedtestvalues.AreAPICredentialsSet(e) || mockTests {
		// CryptocurrencyTradeFee Basic
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			assert.NoError(t, err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			assert.NoError(t, err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			assert.NoError(t, err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			assert.NoError(t, err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		assert.NoError(t, err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		assert.NoError(t, err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		assert.NoError(t, err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		assert.NoError(t, err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		assert.NoError(t, err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText +
		" & " +
		exchange.AutoWithdrawCryptoWithSetupText +
		" & " +
		exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := e.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions)
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

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil && !mockTests:
		assert.NoError(t, err, "GetActiveOrders should not error")
	case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
		assert.Error(t, err, "GetActiveOrders should error when no keys are set")
	case mockTests && err != nil:
		assert.NoError(t, err, "GetActiveOrders should not error")
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

	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		assert.NoError(t, err, "GetOrderHistory should not error")
	case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
		assert.Error(t, err, "GetOrderHistory should error when no keys are set")
	case err != nil && mockTests:
		assert.NoError(t, err, "GetOrderHistory should not error")
	}
}

// TestSubmitOrder and below can impact your orders on the exchange. Enable canManipulateRealOrders to run them
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	orderSubmission := &order.Submit{
		Exchange: e.Name,
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

	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	switch {
	case sharedtestvalues.AreAPICredentialsSet(e) && (err != nil || response.Status != order.New):
		assert.NoError(t, err, "SubmitOrder should not error")
		assert.Equal(t, order.New, response.Status, "SubmitOrder should return order.New status")
	case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
		assert.Error(t, err, "SubmitOrder should error when no keys are set")
	case mockTests && err != nil:
		assert.NoError(t, err, "SubmitOrder should not error")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
	orderCancellation := &order.Cancel{
		OrderID:   "266029865",
		AssetType: asset.Spot,
		Pair:      currency.NewBTCUSDT(),
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
		assert.Error(t, err, "CancelOrder should error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		assert.NoError(t, err, "CancelOrder should not error")
	case err != nil && mockTests:
		assert.NoError(t, err, "CancelOrder should not error")
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := e.CancelAllOrders(t.Context(), orderCancellation)
	switch {
	case !sharedtestvalues.AreAPICredentialsSet(e) && err == nil && !mockTests:
		assert.Error(t, err, "CancelAllOrders should error when no keys are set")
	case sharedtestvalues.AreAPICredentialsSet(e) && err != nil:
		assert.NoError(t, err, "CancelAllOrders should not error")
	case mockTests && err != nil:
		assert.NoError(t, err, "CancelAllOrders should not error")
	}

	assert.Empty(t, resp.Status, "CancelAllOrders should return zero failed statuses")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	_, err := e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	assert.Error(t, err, "ModifyOrder should error for incomplete request")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdraw.Request{
			Exchange:    e.Name,
			Amount:      -1,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
			Crypto: withdraw.CryptoRequest{
				Address: core.BitcoinDonationAddress,
			},
		})
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		assert.Error(t, err, "Withdraw should error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil && !mockTests {
		assert.NoError(t, err, "Withdraw should not error")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err == nil && mockTests {
		assert.Error(t, err, "Withdraw should error in mock mode with credentials")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
	assert.Error(t, err, "GetDepositAddress should error when account details are missing")
}

func TestWsAuth(t *testing.T) {
	if !e.Websocket.IsEnabled() &&
		!e.API.AuthenticatedWebsocketSupport ||
		!sharedtestvalues.AreAPICredentialsSet(e) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	if !e.Websocket.IsConnected() {
		if err := e.Websocket.Connect(t.Context()); err != nil {
			require.NoError(t, err)
		}
	}

	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	defer timer.Stop()

	for {
		select {
		case resp := <-e.Websocket.DataHandler.C:
			subAck, ok := resp.Data.(WsSubscriptionAcknowledgementResponse)
			if !ok {
				continue
			}
			if subAck.Type != "subscription_ack" {
				continue
			}
			return
		case <-timer.C:
			require.FailNow(t, "Auth websocket subscription ack must be received before timeout")
		}
	}
}

func TestWsMissingRole(t *testing.T) {
	pressXToJSON := []byte(`{
		"result":"error",
		"reason":"MissingRole",
		"message":"To access this endpoint, you need to log in to the website and go to the settings page to assign one of these roles [FundManager] to API key wujB3szN54gtJ4QDhqRJ which currently has roles [Trader]"
	}`)
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err == nil {
		assert.Error(t, err, "wsHandleData should return an error")
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
	err := e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	err = e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	err = e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	err = e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	err = e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	err := e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
	err := e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	err = e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
	err = e.wsHandleData(t.Context(), nil, pressXToJSON)
	if err != nil {
		assert.NoError(t, err)
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
		err := e.wsHandleData(t.Context(), nil, tt[x].Data)
		if tt[x].ErrorExpected && err != nil && !strings.Contains(err.Error(), tt[x].ErrorShouldContain) {
			assert.Contains(t, err.Error(), tt[x].ErrorShouldContain, "error should contain expected substring")
		} else if !tt[x].ErrorExpected && err != nil {
			assert.NoError(t, err, "wsHandleData should not error for this payload")
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
	if err := e.wsHandleData(t.Context(), nil, pressXToJSON); err != nil {
		assert.NoError(t, err)
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
		assert.Equal(t, testCases[i].Result, result)
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
		assert.Equal(t, testCases[i].Result, result)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testCurrency)
	if err != nil {
		require.NoError(t, err)
	}
	_, err = e.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		assert.NoError(t, err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testCurrency)
	if err != nil {
		require.NoError(t, err)
	}
	tStart := time.Date(2020, 6, 6, 0, 0, 0, 0, time.UTC)
	tEnd := time.Date(2020, 6, 7, 0, 0, 0, 0, time.UTC)
	if !mockTests {
		tStart = time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		tEnd = time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 15, 0, 0, time.UTC)
	}
	_, err = e.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, tStart, tEnd)
	if err != nil {
		assert.NoError(t, err)
	}
}

func TestTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.Transfers(t.Context(), currency.BTC, time.Time{}, 100, "", true)
	if err != nil {
		assert.NoError(t, err)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetAccountFundingHistory(t.Context())
	if err != nil {
		assert.NoError(t, err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		assert.NoError(t, err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetOrderInfo(t.Context(), "1234", currency.EMPTYPAIR, asset.Empty)
	if err != nil {
		assert.NoError(t, err)
	}
}

func TestGetSymbolDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbolDetails(t.Context(), "all")
	if err != nil {
		assert.NoError(t, err)
	}
	_, err = e.GetSymbolDetails(t.Context(), "btcusd")
	if err != nil {
		assert.NoError(t, err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, false)
			require.NoError(t, err, "GetPairs must not error")
			l, err := e.GetOrderExecutionLimits(a, pairs[0])
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "GetPairs must not error for asset %s", a)
		require.NotEmptyf(t, pairs, "pairs must not be empty for asset %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", ""), currency.NewPairWithDelimiter("ETH", "BTC", "")}
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, false))
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, true))
	subs, err := e.generateSubscriptions()
	require.NoError(t, err)
	exp := subscription.List{
		{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: p, QualifiedChannel: "candles_1d", Interval: kline.OneDay},
		{Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: p, QualifiedChannel: "l2"},
	}
	testsubs.EqualLists(t, exp, subs)

	for _, i := range []kline.Interval{kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.SixHour} {
		subs, err = subscription.List{{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: p, Interval: i}}.ExpandTemplates(e)
		assert.NoErrorf(t, err, "ExpandTemplates should not error on interval %s", i)
		require.NotEmpty(t, subs)
		assert.Equal(t, "candles_"+i.Short(), subs[0].QualifiedChannel)
	}
	_, err = subscription.List{{Asset: asset.Spot, Channel: subscription.CandlesChannel, Pairs: p, Interval: kline.FourHour}}.ExpandTemplates(e)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "ExpandTemplates should error on invalid interval")

	assert.PanicsWithError(t,
		"subscription channel not supported: wibble",
		func() { channelName(&subscription.Subscription{Channel: "wibble"}) },
		"should panic on invalid channel",
	)
}
