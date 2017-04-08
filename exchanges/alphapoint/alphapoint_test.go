package alphapoint

import (
	"reflect"
	"testing"
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

func TestGetTicker(t *testing.T) {
	GetTicker := Alphapoint{}
	GetTicker.SetDefaults()

	response, err := GetTicker.GetTicker("BTCUSD")
	if err != nil {
		t.Error("Test Failed - Alphapoint GetTicker init error: ", err)
	}
	if reflect.ValueOf(response).NumField() != 13 {
		t.Error("Test Failed - Alphapoint GetTicker struct change/or updated")
	}
	if reflect.TypeOf(response.Ask).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.Ask value is not a float64")
	}
	if reflect.TypeOf(response.Bid).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.Bid value is not a float64")
	}
	if reflect.TypeOf(response.BuyOrderCount).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.BuyOrderCount value is not a float64")
	}
	if reflect.TypeOf(response.High).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.High value is not a float64")
	}
	if reflect.TypeOf(response.IsAccepted).String() != "bool" {
		t.Error("Test Failed - Alphapoint GetTicker.IsAccepted value is not a bool")
	}
	if reflect.TypeOf(response.Last).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.Last value is not a float64")
	}
	if reflect.TypeOf(response.Low).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.Low value is not a float64")
	}
	if reflect.TypeOf(response.NumOfCreateOrders).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.NumOfCreateOrders value is not a float64")
	}
	if reflect.TypeOf(response.RejectReason).String() != "string" {
		t.Error("Test Failed - Alphapoint GetTicker.RejectReason value is not a string")
	}
	if reflect.TypeOf(response.SellOrderCount).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.SellOrderCount value is not a float64")
	}
	if reflect.TypeOf(response.Total24HrNumTrades).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.Total24HrNumTrades value is not a float64")
	}
	if reflect.TypeOf(response.Total24HrQtyTraded).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.Total24HrQtyTraded value is not a float64")
	}
	if reflect.TypeOf(response.Volume).String() != "float64" {
		t.Error("Test Failed - Alphapoint GetTicker.Volume value is not a float64")
	}

	if response.Ask < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.Ask value is negative")
	}
	if response.Bid < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.Bid value is negative")
	}
	if response.BuyOrderCount < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.High value is negative")
	}
	if response.High < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.Last value is negative")
	}
	if response.Last < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.Low value is negative")
	}
	if response.Low < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.Mid value is negative")
	}
	if response.NumOfCreateOrders < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.ask value is negative")
	}
	if response.SellOrderCount < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.ask value is negative")
	}
	if response.Total24HrNumTrades < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.ask value is negative")
	}
	if response.Total24HrQtyTraded < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.ask value is negative")
	}
	if response.Volume < 0 {
		t.Error("Test Failed - Alphapoint GetTicker.ask value is negative")
	}
}

func TestGetTrades(t *testing.T) {
	GetTrades := Alphapoint{}
	GetTrades.SetDefaults()

	trades, err := GetTrades.GetTrades("BTCUSD", 0, 10)
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
	if reflect.ValueOf(trades).NumField() != 7 {
		t.Error("Test Failed - Alphapoint AlphapointTrades struct updated/changed")
	}
	if len(trades.Trades) == 0 {
		t.Error("Test Failed - Alphapoint trades.Trades: Incorrect length")
	}
	if reflect.ValueOf(trades.Trades[0]).NumField() != 8 {
		t.Error("Test Failed - Alphapoint AlphapointTrades.Trades struct updated/changed")
	}
	if reflect.TypeOf(trades.Trades[0].BookServerOrderID).String() != "int" {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is not a int")
	}
	if reflect.TypeOf(trades.Trades[0].IncomingOrderSide).String() != "int" {
		t.Error("Test Failed - Alphapoint trades.Trades.IncomingOrderSide value is not a int")
	}
	if reflect.TypeOf(trades.Trades[0].IncomingServerOrderID).String() != "int" {
		t.Error("Test Failed - Alphapoint trades.Trades.IncomingServerOrderID value is not a int")
	}
	if reflect.TypeOf(trades.Trades[0].Price).String() != "float64" {
		t.Error("Test Failed - Alphapoint trades.Trades.Price value is not a float64")
	}
	if reflect.TypeOf(trades.Trades[0].Quantity).String() != "float64" {
		t.Error("Test Failed - Alphapoint trades.Trades.Quantity value is not a float64")
	}
	if reflect.TypeOf(trades.Trades[0].TID).String() != "int64" {
		t.Error("Test Failed - Alphapoint trades.Trades.TID value is not a int64")
	}
	if reflect.TypeOf(trades.Trades[0].UTCTicks).String() != "int64" {
		t.Error("Test Failed - Alphapoint trades.Trades.UTCTicks value is not a int64")
	}
	if reflect.TypeOf(trades.Trades[0].Unixtime).String() != "int" {
		t.Error("Test Failed - Alphapoint trades.Trades.Unixtime value is not a int")
	}
	if reflect.TypeOf(trades.Count).String() != "int" {
		t.Error("Test Failed - Alphapoint trades.Count value is not a int")
	}
	if reflect.TypeOf(trades.DateTimeUTC).String() != "int64" {
		t.Error("Test Failed - Alphapoint trades.DateTimeUTC value is not a int64")
	}
	if reflect.TypeOf(trades.Instrument).String() != "string" {
		t.Error("Test Failed - Alphapoint trades.Instrument value is not a string")
	}
	if reflect.TypeOf(trades.IsAccepted).String() != "bool" {
		t.Error("Test Failed - Alphapoint trades.IsAccepted value is not a bool")
	}
	if reflect.TypeOf(trades.RejectReason).String() != "string" {
		t.Error("Test Failed - Alphapoint trades.string value is not a string")
	}
	if reflect.TypeOf(trades.StartIndex).String() != "int" {
		t.Error("Test Failed - Alphapoint trades.Count value is not a int")
	}

	if trades.Count < 0 {
		t.Error("Test Failed - Alphapoint trades.Count value is negative")
	}
	if trades.DateTimeUTC <= 0 {
		t.Error("Test Failed - Alphapoint trades.DateTimeUTC value is negative or 0")
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
	if trades.StartIndex != 0 {
		t.Error("Test Failed - Alphapoint trades.StartIndex value is incorrect")
	}
	if trades.Trades[0].BookServerOrderID < 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
	if trades.Trades[0].IncomingOrderSide < 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
	if trades.Trades[0].IncomingServerOrderID < 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
	if trades.Trades[0].Price < 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
	if trades.Trades[0].Quantity < 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
	if trades.Trades[0].TID != 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
	if trades.Trades[0].UTCTicks < 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
	if trades.Trades[0].Unixtime < 0 {
		t.Error("Test Failed - Alphapoint trades.Trades.BookServerOrderID value is negative")
	}
}

func TestGetTradesByDate(t *testing.T) {
	GetTradesByDate := Alphapoint{}
	GetTradesByDate.SetDefaults()

	trades, err := GetTradesByDate.GetTradesByDate("BTCUSD", 1414799400, 1414800000)
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
	if reflect.ValueOf(trades).NumField() != 7 {
		t.Error("Test Failed - Alphapoint AlphapointTrades struct updated/changed")
	}
	if len(trades.Trades) != 0 {
		t.Error("Test Failed - Alphapoint trades.Trades: Incorrect length")
	}
	if reflect.TypeOf(trades.DateTimeUTC).String() != "int64" {
		t.Error("Test Failed - Alphapoint trades.Count value is not a int64")
	}
	if reflect.TypeOf(trades.EndDate).String() != "int64" {
		t.Error("Test Failed - Alphapoint trades.DateTimeUTC value is not a int64")
	}
	if reflect.TypeOf(trades.Instrument).String() != "string" {
		t.Error("Test Failed - Alphapoint trades.Instrument value is not a string")
	}
	if reflect.TypeOf(trades.IsAccepted).String() != "bool" {
		t.Error("Test Failed - Alphapoint trades.IsAccepted value is not a bool")
	}
	if reflect.TypeOf(trades.RejectReason).String() != "string" {
		t.Error("Test Failed - Alphapoint trades.string value is not a string")
	}
	if reflect.TypeOf(trades.StartDate).String() != "int64" {
		t.Error("Test Failed - Alphapoint trades.StartDate value is not a int64")
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
	GetOrderbook := Alphapoint{}
	GetOrderbook.SetDefaults()

	orderBook, err := GetOrderbook.GetOrderbook("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
	if reflect.ValueOf(orderBook).NumField() != 4 {
		t.Error("Test Failed - Alphapoint AlphapointOrderbook struct updated/changed")
	}
	if reflect.TypeOf(orderBook.IsAccepted).String() != "bool" {
		t.Error("Test Failed - Alphapoint orderBook.IsAccepted value is not a bool")
	}
	if reflect.TypeOf(orderBook.RejectReason).String() != "string" {
		t.Error("Test Failed - Alphapoint orderBook.RejectReason value is not a string")
	}
	if len(orderBook.Asks) < 1 {
		t.Error("Test Failed - Alphapoint orderBook.Asks does not contain anything.")
	}
	if len(orderBook.Bids) < 1 {
		t.Error("Test Failed - Alphapoint orderBook.Asks does not contain anything.")
	}
}

func TestGetProductPairs(t *testing.T) {
	GetProductPairs := Alphapoint{}
	GetProductPairs.SetDefaults()

	productPairs, err := GetProductPairs.GetProductPairs()
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
	if reflect.ValueOf(productPairs).NumField() != 3 {
		t.Error("Test Failed - Alphapoint GetProductPairs struct updated/changed")
	}
	if reflect.TypeOf(productPairs.IsAccepted).String() != "bool" {
		t.Error("Test Failed - Alphapoint productPairs.IsAccepted value is not a bool")
	}
	if reflect.TypeOf(productPairs.RejectReason).String() != "string" {
		t.Error("Test Failed - Alphapoint productPairs.RejectReason value is not a string")
	}

	if len(productPairs.ProductPairs) >= 1 {
		if reflect.ValueOf(productPairs.ProductPairs[0]).NumField() != 6 {
			t.Error("Test Failed - Alphapoint GetProductPairs.ProductPairs[] struct updated/changed")
		}
		if reflect.TypeOf(productPairs.ProductPairs[0].Name).String() != "string" {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Name value is not a string")
		}
		if reflect.TypeOf(productPairs.ProductPairs[0].Product1Decimalplaces).String() != "int" {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Product1Decimalplaces value is not a int")
		}
		if reflect.TypeOf(productPairs.ProductPairs[0].Product1Label).String() != "string" {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Product1Label value is not a string")
		}
		if reflect.TypeOf(productPairs.ProductPairs[0].Product2Decimalplaces).String() != "int" {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Product2Decimalplaces value is not a int")
		}
		if reflect.TypeOf(productPairs.ProductPairs[0].Product2Label).String() != "string" {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Product2Label value is not a string")
		}
		if reflect.TypeOf(productPairs.ProductPairs[0].Productpaircode).String() != "int" {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Productpaircode value is not a int")
		}

		if productPairs.ProductPairs[0].Product1Decimalplaces < 0 {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Product1Decimalplaces value is negative")
		}
		if productPairs.ProductPairs[0].Product2Decimalplaces < 0 {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Product2Decimalplaces value is negative")
		}
		if productPairs.ProductPairs[0].Productpaircode < 0 {
			t.Error("Test Failed - Alphapoint productPairs.ProductPairs.Productpaircode value is negative")
		}
	} else {
		t.Error("Test Failed - Alphapoint productPairs.ProductPairs no product pairs.")
	}
}

func TestGetProducts(t *testing.T) {
	GetProducts := Alphapoint{}
	GetProducts.SetDefaults()

	products, err := GetProducts.GetProducts()
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
	if reflect.ValueOf(products).NumField() != 3 {
		t.Error("Test Failed - Alphapoint GetProductPairs struct updated/changed")
	}
	if reflect.TypeOf(products.IsAccepted).String() != "bool" {
		t.Error("Test Failed - Alphapoint products.IsAccepted value is not a bool")
	}
	if reflect.TypeOf(products.RejectReason).String() != "string" {
		t.Error("Test Failed - Alphapoint products.RejectReason value is not a string")
	}

	if len(products.Products) >= 1 {
		if reflect.ValueOf(products.Products[0]).NumField() != 5 {
			t.Error("Test Failed - Alphapoint Getproducts.Products[] struct updated/changed")
		}
		if reflect.TypeOf(products.Products[0].DecimalPlaces).String() != "int" {
			t.Error("Test Failed - Alphapoint products.Products.DecimalPlaces value is not a int")
		}
		if reflect.TypeOf(products.Products[0].FullName).String() != "string" {
			t.Error("Test Failed - Alphapoint products.Products.FullName value is not a string")
		}
		if reflect.TypeOf(products.Products[0].IsDigital).String() != "bool" {
			t.Error("Test Failed - Alphapoint products.Products.IsDigital value is not a bool")
		}
		if reflect.TypeOf(products.Products[0].Name).String() != "string" {
			t.Error("Test Failed - Alphapoint products.Products.Name value is not a string")
		}
		if reflect.TypeOf(products.Products[0].ProductCode).String() != "int" {
			t.Error("Test Failed - Alphapoint products.Products.ProductCode value is not a int")
		}

		if products.Products[0].DecimalPlaces < 0 {
			t.Error("Test Failed - Alphapoint products.Products.DecimalPlaces value is negative")
		}
		if products.Products[0].ProductCode < 0 {
			t.Log(products.Products[0].ProductCode)
			t.Error("Test Failed - Alphapoint products.Products.ProductCode value is negative")
		}
	} else {
		t.Error("Test Failed - Alphapoint products.Products no product pairs.")
	}
}

func TestCreateAccount(t *testing.T) {
	CreateAccount := Alphapoint{}
	CreateAccount.SetDefaults()

	err := CreateAccount.CreateAccount("test", "account", "oharareid.ryan@gmail.com", "0433588258", "lolcat123")
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	GetUserInfo := Alphapoint{}
	GetUserInfo.SetDefaults()

	userInfo, err := GetUserInfo.GetUserInfo()
	if err != nil {
		t.Errorf("Test Failed - Init error: %s", err)
	}
	t.Log(userInfo)
}
