package lbank

import (
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey          = ""
	testAPISecret       = ""
	canManipulateOrders = false
)

var l Lbank
var setupRan bool

func TestSetup(t *testing.T) {
	if setupRan {
		return
	}
	setupRan = true

	t.Parallel()
	l.SetDefaults()
	l.APIKey = testAPIKey
	l.APISecret = testAPISecret
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json")
	if err != nil {
		t.Errorf("Test Failed - Lbank Setup() init error:, %v", err)
	}
	lbankConfig, err := cfg.GetExchangeConfig("Lbank")
	if err != nil {
		t.Errorf("Test Failed - Lbank Setup() init error: %v", err)
	}
	lbankConfig.Websocket = true
	l.Setup(&lbankConfig)
}

func areTestAPIKeysSet() bool {
	if l.APIKey != "" && l.APIKey != "Key" &&
		l.APISecret != "" && l.APISecret != "Secret" {
		return true
	}
	return false
}

func TestGetTicker(t *testing.T) {
	TestSetup(t)
	_, err := l.GetTicker("btc_usdt")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyPairs(t *testing.T) {
	TestSetup(t)
	_, err := l.GetCurrencyPairs()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketDepths(t *testing.T) {
	TestSetup(t)
	_, err := l.GetMarketDepths("btc_usdt", "60", "1")
	if err != nil {
		t.Errorf("GetMarketDepth failed: %v", err)
	}
	a, _ := l.GetMarketDepths("btc_usdt", "60", "0")
	if len(a.Asks) != 60 {
		t.Errorf("length requested doesnt match the output")
	}
	_, err = l.GetMarketDepths("btc_usdt", "61", "0")
	if err == nil {
		t.Errorf("size is greater than the maximum allowed")
	}
}

func TestGetTrades(t *testing.T) {
	TestSetup(t)
	_, err := l.GetTrades("btc_usdt", "600", fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		t.Error(err)
	}
	a, err := l.GetTrades("btc_usdt", "600", "0")
	if len(a) != 600 && err != nil {
		t.Error(err)
	}
}

func TestGetKlines(t *testing.T) {
	TestSetup(t)
	_, err := l.GetKlines("btc_usdt", "600", "minute1", fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	TestSetup(t)
	p := currency.Pair{
		Delimiter: "_",
		Base:      currency.ETH,
		Quote:     currency.BTC}

	_, err := l.UpdateOrderbook(p.Lower(), "spot")
	if err != nil {
		t.Errorf("Update for orderbook failed: %v", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.GetUserInfo()
	if err != nil {
		t.Error("invalid key or sign", err)
	}
}

func TestCreateOrder(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.CreateOrder(cp.Lower().String(), "what", 1231, 12314)
	if err == nil {
		t.Error("Test Failed - CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(cp.Lower().String(), "buy", 0, 0)
	if err == nil {
		t.Error("Test Failed - CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(cp.Lower().String(), "sell", 1231, 0)
	if err == nil {
		t.Error("Test Failed - CreateOrder error cannot be nil")
	}
	_, err = l.CreateOrder(cp.Lower().String(), "buy", 58, 681)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRemoveOrder(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	_, err := l.RemoveOrder(cp.Lower().String(), "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23")
	if err != nil {
		t.Errorf("unable to remove order: %v", err)
	}
}

func TestQueryOrder(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrder(cp.Lower().String(), "1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestQueryOrderHistory(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.QueryOrderHistory(cp.Lower().String(), "1", "50")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPairInfo(t *testing.T) {
	TestSetup(t)
	_, err := l.GetPairInfo()
	if err != nil {
		t.Errorf("couldnt get pair info: %v", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.GetOpenOrders(cp.Lower().String(), 1, 50)
	if err != nil {
		t.Error("unexpected error", err)
	}
}

func TestUSD2RMBRate(t *testing.T) {
	TestSetup(t)
	_, err := l.USD2RMBRate()
	if err != nil {
		t.Error("unable to acquire the rate")
	}
}

func TestGetWithdrawConfig(t *testing.T) {
	TestSetup(t)
	_, err := l.GetWithdrawConfig("eth")
	if err != nil {
		t.Errorf("unable to get withdraw config: %v", err)
	}
}

func TestWithdraw(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.Withdraw("", "", "", "", "")
	if err != nil {
		t.Errorf("unable to withdraw: %v", err)
	}
}

func TestGetWithdrawRecords(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	_, err := l.GetWithdrawalRecords("eth", "0", "1", "20")
	if err != nil {
		t.Errorf("unable to get withdrawal records: %v", err)
	}
}

func TestLoadPrivKey(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	err := l.loadPrivKey()
	if err != nil {
		t.Error(err)
	}
	l.privKeyLoaded = false
	l.APISecret = "errortest"
	err = l.loadPrivKey()
	if err == nil {
		t.Errorf("expected error due to pemblock nil, got err: %v", err)
	}
}

func TestSign(t *testing.T) {
	TestSetup(t)
	areTestAPIKeysSet()

	l.APISecret = testAPISecret
	l.loadPrivKey()
	_, err := l.sign("hello123")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	_, err := l.SubmitOrder(cp.Lower(), "BUY", "ANY", 2, 1312, "")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	areTestAPIKeysSet()
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.ETH.String(), currency.BTC.String(), "_")
	var a exchange.OrderCancellation
	a.CurrencyPair = cp
	a.OrderID = "24f7ce27-af1d-4dca-a8c1-ef1cbeec1b23"
	err := l.CancelOrder(&a)
	if err != nil {
		t.Error(err)
	}
}

// func TestGetOrderInfo(t *testing.T) {
// 	areTestAPIKeysSet()
// 	TestSetup(t)
// 	_, err := l.GetOrderInfo("9ead39f5-701a-400b-b635-d7349eb0f6b")
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestGetAllOpenOrderID(t *testing.T) {
// 	areTestAPIKeysSet()
// 	TestSetup(t)
// 	_, err := l.GetAllOpenOrderID()
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestGetFeeByType(t *testing.T) {
	TestSetup(t)
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "_")
	var input exchange.FeeBuilder
	input.Amount = 2
	input.FeeType = exchange.CryptocurrencyWithdrawalFee
	input.Pair = cp
	a, err := l.GetFeeByType(&input)
	if err != nil {
		t.Error(err)
	}
	if a != 0.0005 {
		t.Errorf("testGetFeeByType failed. Expected: 0.0005, Received: %v", a)
	}
}

// func TestSomething(t *testing.T) {
// 	var resp exchange.CancelAllOrdersResponse
// 	orderIDs, err := l.GetAllOpenOrderID()
// 	if err != nil {
// 		return resp, nil
// 	}
// 	for key := range orderIDs {
// 		if key != orders.CurrencyPair.String() {
// 			continue
// 		}
// 		var x, y int64
// 		x = 0
// 		y = 0
// 		var tempSlice []string
// 		tempSlice = append(tempSlice, orderIDs[key][x])
// 		for orderIDs[key][x] != "" {
// 			x++
// 			for y != x {
// 				tempSlice = append(tempSlice, orderIDs[key][y])
// 				if y%3 == 0 {
// 					input := strings.Join(tempSlice, ",")
// 					CancelResponse, err2 := l.RemoveOrder(key, input)
// 					if err2 != nil {
// 						return resp, err2
// 					}
// 					tempStringSuccess := strings.Split(CancelResponse.Success, ",")
// 					for k := range tempStringSuccess {
// 						resp.OrderStatus[tempStringSuccess[k]] = "Cancelled"
// 					}
// 					tempStringError := strings.Split(CancelResponse.Error, ",")
// 					for l := range tempStringError {
// 						resp.OrderStatus[tempStringError[l]] = "Failed"
// 					}
// 					tempSlice = tempSlice[:0]
// 					y++
// 				}
// 			y++
// 			}
// 		}
// 	}
// 	return resp, nil
// }

func TestSomething(t *testing.T) {
	var temp OpenOrderResponse
	temp.PageLength = 200
	temp.PageNumber = 1
	temp.Total = "3"
	temp.Result = true
	var temp2 OrderResponse
	temp2.Symbol = "eth_btc"
	temp2.Amount = 5.00
	temp2.CreateTime = 12472345454
	temp2.Price = 6666.00
	temp2.AvgPrice = 0.00
	temp2.Type = "sell"
	temp2.OrderID = "a"
	temp2.DealAmount = 10.00
	temp2.Status = 2
	var temp3 OrderResponse
	temp3.Symbol = "eth_btc"
	temp3.Amount = 5.00
	temp3.CreateTime = 12472345454
	temp3.Price = 6666.00
	temp3.AvgPrice = 0.00
	temp3.Type = "sell"
	temp3.OrderID = "b"
	temp3.DealAmount = 10.00
	temp3.Status = 2
	var temp4 OrderResponse
	temp4.Symbol = "eth_btc"
	temp4.Amount = 5.00
	temp4.CreateTime = 12472345454
	temp4.Price = 6666.00
	temp4.AvgPrice = 0.00
	temp4.Type = "sell"
	temp4.OrderID = "c"
	temp4.DealAmount = 10.00
	temp4.Status = 2
	temp.Orders = append(temp.Orders, temp2)
	t.Log(temp)
	temp.Orders = append(temp.Orders, temp3)
	t.Log(temp)
	temp.Orders = append(temp.Orders, temp4)
	t.Log(temp)
	var resp []GetAllOpenIDResp

	b := int64(1)
	var x int64
	tempData, err := strconv.ParseInt(temp.Total, 10, 64)
	if err != nil {
		t.Log(err)
	}
	if tempData%200 != 0 {
		tempData = tempData - (tempData % 200)
		x = tempData/200 + 1
	} else {
		x = tempData / 200
	}
	d, err := strconv.ParseInt(temp.Total, 10, 64)
	if err != nil {
		t.Log(err)
	}
	log.Println("HELLO MATE")
	for ; b <= x; b++ {
		log.Println("HELLO DUDE")

		for c := int64(0); c < d; c++ {
			resp = append(resp, GetAllOpenIDResp{
				CurrencyPair: "eth_btc",
				OrderID:      temp.Orders[c].OrderID})
		}
	}
	t.Log(resp)
}
