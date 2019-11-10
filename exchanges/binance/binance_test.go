package binance

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/idoall/TokenExchangeCommon/commonutils"
	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/config"
	"github.com/idoall/gocryptotrader/currency"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/binance"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Binance

func areTestAPIKeysSet() bool {
	if b.APIKey != "" && b.APIKey != "Key" &&
		b.APISecret != "" && b.APISecret != "Secret" {
		return true
	}
	return false
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

func TestGetExchangeValidCurrencyPairs(t *testing.T) {
	t.Parallel()

	_, err := b.GetExchangeValidCurrencyPairs()
	if err != nil {
		t.Error("Test Failed - Binance GetExchangeValidCurrencyPairs() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
<<<<<<< HEAD
	res, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: b.GetSymbol(),
=======

	_, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: "BTCUSDT",
>>>>>>> upstrem/master
		Limit:  10,
	})

	if err != nil {
		t.Error("Test Failed - Binance GetOrderBook() error", err)
	} else {
		fmt.Println("----------Bids-------")
		for _, v := range res.Bids {
			b, _ := json.Marshal(v)
			fmt.Println(string(b))
		}
		fmt.Println("----------Asks-------")
		for _, v := range res.Asks {
			b, _ := json.Marshal(v)
			fmt.Println(string(b))
		}

	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()

	list, err := b.GetRecentTrades(RecentTradeRequestParams{
		Symbol: b.GetSymbol(),
		Limit:  15,
	})

	if err != nil {
		t.Error("Test Failed - Binance GetRecentTrades() error", err)
	} else {
		for k, v := range list {
			b, _ := json.Marshal(v)
			fmt.Println(k, string(b))
		}

	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetHistoricalTrades("BTCUSDT", 5, 0)
	if !mockTests && err == nil {
		t.Error("Test Failed - Binance GetHistoricalTrades() expecting error")
	}
	if mockTests && err == nil {
		t.Error("Test Failed - Binance GetHistoricalTrades() error", err)
	}
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetAggregatedTrades("BTCUSDT", 5)
	if err != nil {
		t.Error("Test Failed - Binance GetAggregatedTrades() error", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	_, err := b.GetSpotKline(KlinesRequestParams{
		Symbol:   b.GetSymbol(),
		Interval: TimeIntervalFiveMinutes,
		Limit:    24,
	})
	if err != nil {
		t.Error("Test Failed - Binance GetSpotKline() error", err)
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetAveragePrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetAveragePrice() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()

	_, err := b.GetPriceChangeStats("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetPriceChangeStats() error", err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	toBeCharge := "2017-07-20 12:00:00" //待转化为时间戳的字符串 注意 这里的小时和分钟还要秒必须写 因为是跟着模板走的 修改模板的话也可以不写
	toEnCharge := "2019-07-20 12:00:00" //待转化为时间戳的字符串 注意 这里的小时和分钟还要秒必须写 因为是跟着模板走的 修改模板的话也可以不写

	timeLayout := "2006-01-02 15:04:05"  //时区格式化模板
	loc, _ := time.LoadLocation("Local") //重要：获取时区
	var startTime, endTime time.Time

	startTime, _ = time.ParseInLocation(timeLayout, toBeCharge, loc)
	endTime, _ = time.ParseInLocation(timeLayout, toEnCharge, loc)
	pair := currency.Pair{Base: currency.BTC, Quote: currency.USDT}

	klines, err := b.GetKlines(binance.KlinesRequestParams{
		Symbol:    pair.String(),
		Interval:  binance.TimeIntervalHour,
		Limit:     100,
		StartTime: commonutils.UnixNesc(startTime),
		EndTime:   commonutils.UnixNesc(endTime),
	})
	if err != nil {
		t.Error(err)
	} else {
		for _, v := range klines {
			fmt.Println(v)
		}
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()

	_, err := b.GetTickers()
	if err != nil {
		t.Error("Test Failed - Binance TestGetTickers error", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetLatestSpotPrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetLatestSpotPrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetBestPrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetBestPrice() error", err)
	}
}

<<<<<<< HEAD
func TestNewOrder(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}
	_, err := b.NewOrder(&NewOrderRequest{
		Symbol:      "BTCUSDT",
		Side:        BinanceRequestParamsSideSell,
		TradeType:   BinanceRequestParamsOrderLimit,
		TimeInForce: BinanceRequestParamsTimeGTC,
		Quantity:    0.01,
		Price:       1536.1,
	})
	if err == nil {
		t.Error("Test Failed - Binance NewOrder() error", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := b.CancelExistingOrder("BTCUSDT", 82584683, "")
	if err != nil {
		t.Error("Test Failed - Binance CancelExistingOrder() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	res, err := b.QueryOrder(b.GetSymbol(), "", 1337)
	if err != nil {
		t.Error("Test Failed - Binance QueryOrder() error", err)
	} else {
		//{"code":0,"msg":"","symbol":"BTCUSDT","orderId":131046063,"clientOrderId":"2t38MQXdRe9HvctyRdUbIT","price":"100000","origQty":"0.01","executedQty":"0","status":"NEW","timeInForce":"GTC","type":"LIMIT","side":"SELL","stopPrice":"0","icebergQty":"0","time":1531384312008,"isWorking":true}
		b, _ := json.Marshal(res)
		fmt.Println(string(b))
=======
func TestQueryOrder(t *testing.T) {
	t.Parallel()

	_, err := b.QueryOrder("BTCUSDT", "", 1337)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - QueryOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - QueryOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock QueryOrder() error", err)
>>>>>>> upstrem/master
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.OpenOrders("BTCUSDT")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - OpenOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - OpenOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock OpenOrders() error", err)
	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()

	_, err := b.AllOrders("BTCUSDT", "", "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - AllOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - AllOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock AllOrders() error", err)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()
	b.GetFeeByType(feeBuilder)
	if !areTestAPIKeysSet() {
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

	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() || mockTests {
		// CryptocurrencyTradeFee Basic
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.1) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(100000) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(100000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.1) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.1), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}

	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText

	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}
	_, err := b.GetActiveOrders(&getOrdersRequest)
	if err == nil {
		t.Error("Expected: 'At least one currency is required to fetch order history'. received nil")
	}

	getOrdersRequest.Currencies = []currency.Pair{
		currency.NewPair(currency.LTC, currency.BTC),
	}

	_, err = b.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - GetActiveOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - GetActiveOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock GetActiveOrders() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if err == nil {
		t.Error("Expected: 'At least one currency is required to fetch order history'. received nil")
	}

	getOrdersRequest.Currencies = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC)}

	_, err = b.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - GetOrderHistory() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - GetOrderHistory() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock GetOrderHistory() error", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := b.SubmitOrder(currency.NewPair(currency.LTC, currency.BTC),
		exchange.BuyOrderSide,
		exchange.MarketOrderType,
		1,
		1,
		"clientId")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - SubmitOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - SubmitOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock SubmitOrder() error", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currency.NewPair(currency.LTC, currency.BTC),
	}

	err := b.CancelOrder(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - CancelExchangeOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - CancelExchangeOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock CancelExchangeOrder() error", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currency.NewPair(currency.LTC, currency.BTC),
	}

	_, err := b.CancelAllOrders(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - CancelAllExchangeOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - CancelAllExchangeOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock CancelAllExchangeOrders() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountInfo()
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - GetAccountInfo() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - GetAccountInfo() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock GetAccountInfo() error", err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()

	_, err := b.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error cannot be nil")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      0,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - Withdraw() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - Withdraw() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock Withdraw() error", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()

	var withdrawFiatRequest exchange.WithdrawRequest
	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()

	var withdrawFiatRequest exchange.WithdrawRequest
	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositAddress(currency.BTC, "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Test Failed - GetDepositAddress() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Test Failed - GetDepositAddress() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Test Failed - Mock GetDepositAddress() error", err)
	}
}
