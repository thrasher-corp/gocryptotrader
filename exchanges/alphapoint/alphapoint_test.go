package alphapoint

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	onlineTest = false

	testAPIKey    = ""
	testAPISecret = ""
)

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	SetDefaults := Alphapoint{}

	SetDefaults.SetDefaults()
	if SetDefaults.APIUrl != "https://sim3.alphapoint.com:8400" {
		t.Error("Test Failed - SetDefaults: String Incorrect -", SetDefaults.APIUrl)
	}
	if SetDefaults.WebsocketURL != "wss://sim3.alphapoint.com:8401/v1/GetTicker/" {
		t.Error("Test Failed - SetDefaults: String Incorrect -", SetDefaults.WebsocketURL)
	}
}

func testSetAPIKey(a *Alphapoint) {
	a.APIKey = testAPIKey
	a.APISecret = testAPISecret
	a.AuthenticatedAPISupport = true
}

func testIsAPIKeysSet(a *Alphapoint) bool {
	if testAPIKey != "" && testAPISecret != "" && a.AuthenticatedAPISupport {
		return true
	}
	return false
}
func TestGetTicker(t *testing.T) {
	alpha := Alphapoint{}
	alpha.SetDefaults()

	var ticker Ticker
	var err error

	if onlineTest {
		ticker, err = alpha.GetTicker("BTCUSD")
		if err != nil {
			t.Fatal("Test Failed - Alphapoint GetTicker init error: ", err)
		}

		_, err = alpha.GetTicker("wigwham")
		if err == nil {
			t.Error("Test Failed - Alphapoint GetTicker error")
		}
	} else {
		mockResp := []byte(
			string(`{"high":253.101,"last":249.76,"bid":248.8901,"volume":5.813354,"low":231.21,"ask":248.9012,"Total24HrQtyTraded":52.654968,"Total24HrProduct2Traded":569.05762,"Total24HrNumTrades":4,"sellOrderCount":7,"buyOrderCount":11,"numOfCreateOrders":0,"isAccepted":true}`),
		)

		err = common.JSONDecode(mockResp, &ticker)
		if err != nil {
			t.Fatal("Test Failed - Alphapoint GetTicker unmarshalling error: ", err)
		}

		if ticker.Last != 249.76 {
			t.Error("Test failed - Alphapoint GetTicker expected last = 249.76")
		}
	}

	if ticker.Last < 0 {
		t.Error("Test failed - Alphapoint GetTicker last < 0")
	}
}

func TestGetTrades(t *testing.T) {
	alpha := Alphapoint{}
	alpha.SetDefaults()

	var trades Trades
	var err error

	if onlineTest {
		trades, err = alpha.GetTrades("BTCUSD", 0, 10)
		if err != nil {
			t.Fatalf("Test Failed - Init error: %s", err)
		}

		_, err = alpha.GetTrades("wigwham", 0, 10)
		if err == nil {
			t.Fatal("Test Failed - GetTrades error")
		}
	} else {
		mockResp := []byte(
			string(`{"isAccepted":true,"dateTimeUtc":635507981548085938,"ins":"BTCUSD","startIndex":0,"count":10,"trades":[{"tid":0,"px":231.8379,"qty":4.913,"unixtime":1399951989,"utcticks":635355487898355234,"incomingOrderSide":0,"incomingServerOrderId":2598,"bookServerOrderId":2588},{"tid":1,"px":7895.1487,"qty":0.25,"unixtime":1403143708,"utcticks":635387405087297421,"incomingOrderSide":0,"incomingServerOrderId":284241,"bookServerOrderId":284235},{"tid":2,"px":7935.058,"qty":0.25,"unixtime":1403195348,"utcticks":635387921488684140,"incomingOrderSide":0,"incomingServerOrderId":575845,"bookServerOrderId":574078},{"tid":3,"px":7935.0448,"qty":0.25,"unixtime":1403195378,"utcticks":635387921780090390,"incomingOrderSide":0,"incomingServerOrderId":576028,"bookServerOrderId":575946},{"tid":4,"px":7933.9566,"qty":0.1168,"unixtime":1403195510,"utcticks":635387923108371640,"incomingOrderSide":0,"incomingServerOrderId":576974,"bookServerOrderId":576947},{"tid":5,"px":7961.0856,"qty":0.25,"unixtime":1403202307,"utcticks":635387991073850156,"incomingOrderSide":0,"incomingServerOrderId":600547,"bookServerOrderId":600338},{"tid":6,"px":7961.1388,"qty":0.011,"unixtime":1403202307,"utcticks":635387991073850156,"incomingOrderSide":0,"incomingServerOrderId":600547,"bookServerOrderId":600418},{"tid":7,"px":7961.2451,"qty":0.02,"unixtime":1403202307,"utcticks":635387991073850156,"incomingOrderSide":0,"incomingServerOrderId":600547,"bookServerOrderId":600428},{"tid":8,"px":7947.1437,"qty":0.09,"unixtime":1403202749,"utcticks":635387995498225156,"incomingOrderSide":0,"incomingServerOrderId":602183,"bookServerOrderId":601745},{"tid":9,"px":7818.5073,"qty":0.25,"unixtime":1403219720,"utcticks":635388165206506406,"incomingOrderSide":0,"incomingServerOrderId":661909,"bookServerOrderId":661620}]}`),
		)

		err = common.JSONDecode(mockResp, &trades)
		if err != nil {
			t.Fatal("Test Failed - GetTrades unmarshalling error: ", err)
		}
	}

	if !trades.IsAccepted {
		t.Error("Test Failed - GetTrades IsAccepted failed")
	}

	if trades.Count <= 0 {
		t.Error("Test failed - GetTrades trades count is <= 0")
	}

	if trades.Instrument != "BTCUSD" {
		t.Error("Test failed - GetTrades instrument is != BTCUSD")
	}
}

func TestGetTradesByDate(t *testing.T) {
	alpha := Alphapoint{}
	alpha.SetDefaults()

	var trades Trades
	var err error

	if onlineTest {
		trades, err = alpha.GetTradesByDate("BTCUSD", 1414799400, 1414800000)
		if err != nil {
			t.Errorf("Test Failed - Init error: %s", err)
		}
		_, err = alpha.GetTradesByDate("wigwham", 1414799400, 1414800000)
		if err == nil {
			t.Error("Test Failed - GetTradesByDate error")
		}
	} else {
		mockResp := []byte(
			string(`{"isAccepted":true,"dateTimeUtc":635504540880633671,"ins":"BTCUSD","startDate":1414799400,"endDate":1414800000,"trades":[{"tid":11505,"px":334.669,"qty":0.1211,"unixtime":1414799403,"utcticks":635503962032459843,"incomingOrderSide":1,"incomingServerOrderId":5185651,"bookServerOrderId":5162440},{"tid":11506,"px":334.669,"qty":0.1211,"unixtime":1414799405,"utcticks":635503962058446171,"incomingOrderSide":1,"incomingServerOrderId":5186245,"bookServerOrderId":5162440},{"tid":11507,"px":336.498,"qty":0.011,"unixtime":1414799407,"utcticks":635503962072967656,"incomingOrderSide":0,"incomingServerOrderId":5186530,"bookServerOrderId":5178944},{"tid":11508,"px":335.948,"qty":0.011,"unixtime":1414799410,"utcticks":635503962108055546,"incomingOrderSide":0,"incomingServerOrderId":5187260,"bookServerOrderId":5186531}]}`),
		)

		err = common.JSONDecode(mockResp, &trades)
		if err != nil {
			t.Fatal("Test Failed - GetTradesByDate unmarshalling error: ", err)
		}
	}

	if trades.DateTimeUTC < 0 {
		t.Error("Test Failed - Alphapoint trades.Count value is negative")
	}
	if trades.EndDate < 0 {
		t.Error("Test Failed - Alphapoint trades.DateTimeUTC value is negative")
	}
	if trades.Instrument != "BTCUSD" {
		t.Error("Test Failed - Alphapoint trades.Instrument value is incorrect")
	}
	if trades.IsAccepted != true {
		t.Error("Test Failed - Alphapoint trades.IsAccepted value is true")
	}
	if len(trades.RejectReason) > 0 {
		t.Error("Test Failed - Alphapoint trades.IsAccepted value has been returned")
	}
	if trades.StartDate < 0 {
		t.Error("Test Failed - Alphapoint trades.StartIndex value is negative")
	}
}

func TestGetOrderbook(t *testing.T) {
	alpha := Alphapoint{}
	alpha.SetDefaults()

	var orderBook Orderbook
	var err error

	if onlineTest {
		orderBook, err = alpha.GetOrderbook("BTCUSD")
		if err != nil {
			t.Errorf("Test Failed - Init error: %s", err)
		}

		_, err = alpha.GetOrderbook("wigwham")
		if err == nil {
			t.Error("Test Failed - GetOrderbook() error")
		}
	} else {
		mockResp := []byte(
			string(`{"bids":[{"qty":725,"px":66},{"qty":1289,"px":65},{"qty":1266,"px":64}],"asks":[{"qty":1,"px":67},{"qty":1,"px":69},{"qty":2,"px":70}],"isAccepted":true}`),
		)

		err = common.JSONDecode(mockResp, &orderBook)
		if err != nil {
			t.Fatal("Test Failed - TestGetOrderbook unmarshalling error: ", err)
		}

		if orderBook.Bids[0].Quantity != 725 {
			t.Error("Test Failed - TestGetOrderbook Bids[0].Quantity != 725")
		}
	}

	if !orderBook.IsAccepted {
		t.Error("Test Failed - Alphapoint orderBook.IsAccepted value is negative")
	}

	if len(orderBook.Asks) == 0 {
		t.Error("Test Failed - Alphapoint orderBook.Asks has len 0")
	}

	if len(orderBook.Bids) == 0 {
		t.Error("Test Failed - Alphapoint orderBook.Bids has len 0")
	}
}

func TestGetProductPairs(t *testing.T) {
	alpha := Alphapoint{}
	alpha.SetDefaults()

	var products ProductPairs
	var err error

	if onlineTest {
		products, err = alpha.GetProductPairs()
		if err != nil {
			t.Errorf("Test Failed - Init error: %s", err)
		}
	} else {
		mockResp := []byte(
			string(`{"productPairs":[{"name":"LTCUSD","productPairCode":100,"product1Label":"LTC","product1DecimalPlaces":8,"product2Label":"USD","product2DecimalPlaces":6}, {"name":"BTCUSD","productPairCode":99,"product1Label":"BTC","product1DecimalPlaces":8,"product2Label":"USD","product2DecimalPlaces":6}],"isAccepted":true}`),
		)

		err = common.JSONDecode(mockResp, &products)
		if err != nil {
			t.Fatal("Test Failed - TestGetProductPairs unmarshalling error: ", err)
		}

		if products.ProductPairs[0].Name != "LTCUSD" {
			t.Error("Test Failed - Alphapoint ProductPairs 0 != LTCUSD")
		}

		if products.ProductPairs[1].Product1Label != "BTC" {
			t.Error("Test Failed - Alphapoint ProductPairs 1 != BTC")
		}
	}

	if !products.IsAccepted {
		t.Error("Test Failed - Alphapoint ProductPairs.IsAccepted value is negative")
	}

	if len(products.ProductPairs) == 0 {
		t.Error("Test Failed - Alphapoint ProductPairs len is 0")
	}
}

func TestGetProducts(t *testing.T) {
	alpha := Alphapoint{}
	alpha.SetDefaults()

	var products Products
	var err error

	if onlineTest {
		products, err = alpha.GetProducts()
		if err != nil {
			t.Errorf("Test Failed - Init error: %s", err)
		}
	} else {
		mockResp := []byte(
			string(`{"products": [{"name": "USD","isDigital": false,"productCode": 0,"decimalPlaces": 4,"fullName": "US Dollar"},{"name": "BTC","isDigital": true,"productCode": 1,"decimalPlaces": 6,"fullName": "Bitcoin"}],"isAccepted": true}`),
		)

		err = common.JSONDecode(mockResp, &products)
		if err != nil {
			t.Fatal("Test Failed - TestGetProducts unmarshalling error: ", err)
		}

		if products.Products[0].Name != "USD" {
			t.Error("Test Failed - Alphapoint Products 0 != USD")
		}

		if products.Products[1].ProductCode != 1 {
			t.Error("Test Failed - Alphapoint Products 1 product code != 1")
		}
	}

	if !products.IsAccepted {
		t.Error("Test Failed - Alphapoint Products.IsAccepted value is negative")
	}

	if len(products.Products) == 0 {
		t.Error("Test Failed - Alphapoint Products len is 0")
	}
}

func TestCreateAccount(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	err := a.CreateAccount("test", "account", "something@something.com", "0292383745", "lolcat123")
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
	err = a.CreateAccount("test", "account", "something@something.com", "0292383745", "bla")
	if err == nil {
		t.Errorf("Test Failed - CreateAccount() error")
	}
	err = a.CreateAccount("", "", "", "", "lolcat123")
	if err == nil {
		t.Errorf("Test Failed - CreateAccount() error")
	}
}

func TestGetUserInfo(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.GetUserInfo()
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestSetUserInfo(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.SetUserInfo("bla", "bla", "1", "meh", true, true)
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestGetAccountInfo(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.GetAccountInfo()
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestGetAccountTrades(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.GetAccountTrades("", 1, 2)
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestGetDepositAddresses(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.GetDepositAddresses()
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestWithdrawCoins(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	err := a.WithdrawCoins("", "", "", 0.01)
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestCreateOrder(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.CreateOrder("", "", 1, 0.01, 0)
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestModifyOrder(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.ModifyOrder("", 1, 1)
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestCancelOrder(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.CancelOrder("", 1)
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	err := a.CancelAllOrders("")
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestGetOrders(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.GetOrders()
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}

func TestGetOrderFee(t *testing.T) {
	a := &Alphapoint{}
	a.SetDefaults()
	testSetAPIKey(a)

	if !testIsAPIKeysSet(a) {
		return
	}

	_, err := a.GetOrderFee("", "", 1, 1)
	if err == nil {
		t.Error("Test Failed - GetUserInfo() error")
	}
}
