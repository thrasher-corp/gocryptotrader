package alphapoint

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	onlineTest              = false
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var a Alphapoint

func TestMain(m *testing.M) {
	a.SetDefaults()
	a.SetCredentials(apiKey, apiSecret, "", "", "", "")
	a.API.AuthenticatedSupport = true
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return a.ValidateAPICredentials(a.GetDefaultCredentials()) == nil
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	var ticker Ticker
	var err error
	if onlineTest {
		ticker, err = a.GetTicker(context.Background(), "BTCUSD")
		if err != nil {
			t.Fatal("Alphapoint GetTicker init error: ", err)
		}

		_, err = a.GetTicker(context.Background(), "wigwham")
		if err == nil {
			t.Error("Alphapoint GetTicker Expected error")
		}
	} else {
		mockResp := []byte(
			string(`{"high":253.101,"last":249.76,"bid":248.8901,"volume":5.813354,"low":231.21,"ask":248.9012,"Total24HrQtyTraded":52.654968,"Total24HrProduct2Traded":569.05762,"Total24HrNumTrades":4,"sellOrderCount":7,"buyOrderCount":11,"numOfCreateOrders":0,"isAccepted":true}`),
		)

		err = json.Unmarshal(mockResp, &ticker)
		if err != nil {
			t.Fatal("Alphapoint GetTicker unmarshalling error: ", err)
		}

		if ticker.Last != 249.76 {
			t.Error("Alphapoint GetTicker expected last = 249.76")
		}
	}

	if ticker.Last < 0 {
		t.Error("Alphapoint GetTicker last < 0")
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	var trades Trades
	var err error
	if onlineTest {
		trades, err = a.GetTrades(context.Background(), "BTCUSD", 0, 10)
		if err != nil {
			t.Fatalf("Init error: %s", err)
		}

		_, err = a.GetTrades(context.Background(), "wigwham", 0, 10)
		if err == nil {
			t.Fatal("GetTrades Expected error")
		}
	} else {
		mockResp := []byte(
			string(`{"isAccepted":true,"dateTimeUtc":635507981548085938,"ins":"BTCUSD","startIndex":0,"count":10,"trades":[{"tid":0,"px":231.8379,"qty":4.913,"unixtime":1399951989,"utcticks":635355487898355234,"incomingOrderSide":0,"incomingServerOrderId":2598,"bookServerOrderId":2588},{"tid":1,"px":7895.1487,"qty":0.25,"unixtime":1403143708,"utcticks":635387405087297421,"incomingOrderSide":0,"incomingServerOrderId":284241,"bookServerOrderId":284235},{"tid":2,"px":7935.058,"qty":0.25,"unixtime":1403195348,"utcticks":635387921488684140,"incomingOrderSide":0,"incomingServerOrderId":575845,"bookServerOrderId":574078},{"tid":3,"px":7935.0448,"qty":0.25,"unixtime":1403195378,"utcticks":635387921780090390,"incomingOrderSide":0,"incomingServerOrderId":576028,"bookServerOrderId":575946},{"tid":4,"px":7933.9566,"qty":0.1168,"unixtime":1403195510,"utcticks":635387923108371640,"incomingOrderSide":0,"incomingServerOrderId":576974,"bookServerOrderId":576947},{"tid":5,"px":7961.0856,"qty":0.25,"unixtime":1403202307,"utcticks":635387991073850156,"incomingOrderSide":0,"incomingServerOrderId":600547,"bookServerOrderId":600338},{"tid":6,"px":7961.1388,"qty":0.011,"unixtime":1403202307,"utcticks":635387991073850156,"incomingOrderSide":0,"incomingServerOrderId":600547,"bookServerOrderId":600418},{"tid":7,"px":7961.2451,"qty":0.02,"unixtime":1403202307,"utcticks":635387991073850156,"incomingOrderSide":0,"incomingServerOrderId":600547,"bookServerOrderId":600428},{"tid":8,"px":7947.1437,"qty":0.09,"unixtime":1403202749,"utcticks":635387995498225156,"incomingOrderSide":0,"incomingServerOrderId":602183,"bookServerOrderId":601745},{"tid":9,"px":7818.5073,"qty":0.25,"unixtime":1403219720,"utcticks":635388165206506406,"incomingOrderSide":0,"incomingServerOrderId":661909,"bookServerOrderId":661620}]}`),
		)

		err = json.Unmarshal(mockResp, &trades)
		if err != nil {
			t.Fatal("GetTrades unmarshalling error: ", err)
		}
	}

	if !trades.IsAccepted {
		t.Error("GetTrades IsAccepted failed")
	}

	if trades.Count <= 0 {
		t.Error("GetTrades trades count is <= 0")
	}

	if trades.Instrument != "BTCUSD" {
		t.Error("GetTrades instrument is != BTCUSD")
	}
}

func TestGetTradesByDate(t *testing.T) {
	t.Parallel()
	var trades Trades
	var err error
	if onlineTest {
		trades, err = a.GetTradesByDate(context.Background(),
			"BTCUSD", 1414799400, 1414800000)
		if err != nil {
			t.Errorf("Init error: %s", err)
		}
		_, err = a.GetTradesByDate(context.Background(),
			"wigwham", 1414799400, 1414800000)
		if err == nil {
			t.Error("GetTradesByDate Expected error")
		}
	} else {
		mockResp := []byte(
			string(`{"isAccepted":true,"dateTimeUtc":635504540880633671,"ins":"BTCUSD","startDate":1414799400,"endDate":1414800000,"trades":[{"tid":11505,"px":334.669,"qty":0.1211,"unixtime":1414799403,"utcticks":635503962032459843,"incomingOrderSide":1,"incomingServerOrderId":5185651,"bookServerOrderId":5162440},{"tid":11506,"px":334.669,"qty":0.1211,"unixtime":1414799405,"utcticks":635503962058446171,"incomingOrderSide":1,"incomingServerOrderId":5186245,"bookServerOrderId":5162440},{"tid":11507,"px":336.498,"qty":0.011,"unixtime":1414799407,"utcticks":635503962072967656,"incomingOrderSide":0,"incomingServerOrderId":5186530,"bookServerOrderId":5178944},{"tid":11508,"px":335.948,"qty":0.011,"unixtime":1414799410,"utcticks":635503962108055546,"incomingOrderSide":0,"incomingServerOrderId":5187260,"bookServerOrderId":5186531}]}`),
		)

		err = json.Unmarshal(mockResp, &trades)
		if err != nil {
			t.Fatal("GetTradesByDate unmarshalling error: ", err)
		}
	}

	if trades.DateTimeUTC < 0 {
		t.Error("Alphapoint trades.Count value is negative")
	}
	if trades.EndDate < 0 {
		t.Error("Alphapoint trades.DateTimeUTC value is negative")
	}
	if trades.Instrument != "BTCUSD" {
		t.Error("Alphapoint trades.Instrument value is incorrect")
	}
	if !trades.IsAccepted {
		t.Error("Alphapoint trades.IsAccepted value is true")
	}
	if len(trades.RejectReason) > 0 {
		t.Error("Alphapoint trades.IsAccepted value has been returned")
	}
	if trades.StartDate < 0 {
		t.Error("Alphapoint trades.StartIndex value is negative")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	var orderBook Orderbook
	var err error
	if onlineTest {
		orderBook, err = a.GetOrderbook(context.Background(), "BTCUSD")
		if err != nil {
			t.Errorf("Init error: %s", err)
		}

		_, err = a.GetOrderbook(context.Background(), "wigwham")
		if err == nil {
			t.Error("GetOrderbook() Expected error")
		}
	} else {
		mockResp := []byte(
			string(`{"bids":[{"qty":725,"px":66},{"qty":1289,"px":65},{"qty":1266,"px":64}],"asks":[{"qty":1,"px":67},{"qty":1,"px":69},{"qty":2,"px":70}],"isAccepted":true}`),
		)

		err = json.Unmarshal(mockResp, &orderBook)
		if err != nil {
			t.Fatal("TestGetOrderbook unmarshalling error: ", err)
		}

		if orderBook.Bids[0].Quantity != 725 {
			t.Error("TestGetOrderbook Bids[0].Quantity != 725")
		}
	}

	if !orderBook.IsAccepted {
		t.Error("Alphapoint orderBook.IsAccepted value is negative")
	}

	if len(orderBook.Asks) == 0 {
		t.Error("Alphapoint orderBook.Asks has len 0")
	}

	if len(orderBook.Bids) == 0 {
		t.Error("Alphapoint orderBook.Bids has len 0")
	}
}

func TestGetProductPairs(t *testing.T) {
	t.Parallel()
	var products ProductPairs
	var err error

	if onlineTest {
		products, err = a.GetProductPairs(context.Background())
		if err != nil {
			t.Errorf("Init error: %s", err)
		}
	} else {
		mockResp := []byte(
			string(`{"productPairs":[{"name":"LTCUSD","productPairCode":100,"product1Label":"LTC","product1DecimalPlaces":8,"product2Label":"USD","product2DecimalPlaces":6}, {"name":"BTCUSD","productPairCode":99,"product1Label":"BTC","product1DecimalPlaces":8,"product2Label":"USD","product2DecimalPlaces":6}],"isAccepted":true}`),
		)

		err = json.Unmarshal(mockResp, &products)
		if err != nil {
			t.Fatal("TestGetProductPairs unmarshalling error: ", err)
		}

		if products.ProductPairs[0].Name != "LTCUSD" {
			t.Error("Alphapoint ProductPairs 0 != LTCUSD")
		}

		if products.ProductPairs[1].Product1Label != "BTC" {
			t.Error("Alphapoint ProductPairs 1 != BTC")
		}
	}

	if !products.IsAccepted {
		t.Error("Alphapoint ProductPairs.IsAccepted value is negative")
	}

	if len(products.ProductPairs) == 0 {
		t.Error("Alphapoint ProductPairs len is 0")
	}
}

func TestGetProducts(t *testing.T) {
	t.Parallel()
	var products Products
	var err error

	if onlineTest {
		products, err = a.GetProducts(context.Background())
		if err != nil {
			t.Errorf("Init error: %s", err)
		}
	} else {
		mockResp := []byte(
			string(`{"products": [{"name": "USD","isDigital": false,"productCode": 0,"decimalPlaces": 4,"fullName": "US Dollar"},{"name": "BTC","isDigital": true,"productCode": 1,"decimalPlaces": 6,"fullName": "Bitcoin"}],"isAccepted": true}`),
		)

		err = json.Unmarshal(mockResp, &products)
		if err != nil {
			t.Fatal("TestGetProducts unmarshalling error: ", err)
		}

		if products.Products[0].Name != "USD" {
			t.Error("Alphapoint Products 0 != USD")
		}

		if products.Products[1].ProductCode != 1 {
			t.Error("Alphapoint Products 1 product code != 1")
		}
	}

	if !products.IsAccepted {
		t.Error("Alphapoint Products.IsAccepted value is negative")
	}

	if len(products.Products) == 0 {
		t.Error("Alphapoint Products len is 0")
	}
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	err := a.CreateAccount(context.Background(),
		"test", "account", "something@something.com", "0292383745", "lolcat123")
	if err != nil {
		t.Errorf("Init error: %s", err)
	}
	err = a.CreateAccount(context.Background(),
		"test", "account", "something@something.com", "0292383745", "bla")
	if err == nil {
		t.Errorf("CreateAccount() Expected error")
	}
	err = a.CreateAccount(context.Background(), "", "", "", "", "lolcat123")
	if err == nil {
		t.Errorf("CreateAccount() Expected error")
	}
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.GetUserInfo(context.Background())
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestSetUserInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.SetUserInfo(context.Background(),
		"bla", "bla", "1", "meh", true, true)
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.UpdateAccountInfo(context.Background(), asset.Spot)
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestGetAccountTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.GetAccountTrades(context.Background(), "", 1, 2)
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestGetDepositAddresses(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.GetDepositAddresses(context.Background())
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestWithdrawCoins(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	err := a.WithdrawCoins(context.Background(), "", "", "", 0.01)
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.CreateOrder(context.Background(),
		"", "", order.Limit.String(), 0.01, 0)
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestModifyExistingOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.ModifyExistingOrder(context.Background(), "", 1, 1)
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestCancelAllExistingOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	err := a.CancelAllExistingOrders(context.Background(), "")
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.GetOrders(context.Background())
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestGetOrderFee(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping")
	}

	_, err := a.GetOrderFee(context.Background(), "", "", 1, 1)
	if err == nil {
		t.Error("GetUserInfo() Expected error")
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.WithdrawCryptoWith2FAText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := a.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := a.GetActiveOrders(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := a.GetOrderHistory(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}

	response, err := a.SubmitOrder(context.Background(), orderSubmission)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)

		if !response.IsOrderPlaced {
			t.Errorf("Order failed to be placed: %v", err)
		}
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.BTC, currency.LTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := a.CancelOrder(context.Background(), orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.BTC, currency.LTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := a.CancelAllOrders(context.Background(), orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := a.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	_, err := a.WithdrawCryptocurrencyFunds(context.Background(),
		&withdraw.Request{})
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected 'Not implemented', received %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := a.WithdrawFiatFunds(context.Background(),
		&withdraw.Request{})
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := a.WithdrawFiatFundsToInternationalBank(&withdraw.Request{})
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', received: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("btc_usdt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil && err != common.ErrNotYetImplemented {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("btc_usdt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrNotYetImplemented {
		t.Error(err)
	}
}
