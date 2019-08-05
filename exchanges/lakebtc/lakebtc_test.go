package lakebtc

import (
	"fmt"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ws/monitor"
)

var l LakeBTC

// Please add your own APIkeys to do correct due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestSetDefaults(t *testing.T) {
	l.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	lakebtcConfig, err := cfg.GetExchangeConfig("LakeBTC")
	if err != nil {
		t.Error("Test Failed - LakeBTC Setup() init error")
	}
	lakebtcConfig.AuthenticatedAPISupport = true
	lakebtcConfig.APIKey = apiKey
	lakebtcConfig.APISecret = apiSecret
	lakebtcConfig.Websocket = true
	l.Setup(&lakebtcConfig)
	l.WebsocketURL = lakeBTCWSURL
}

func TestGetTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := l.GetTradablePairs()
	if err != nil {
		t.Fatalf("Test failed. GetTradablePairs err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := l.GetTicker()
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := l.GetOrderBook("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetOrderBook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := l.GetTradeHistory("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestTrade(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	_, err := l.Trade(false, 0, 0, "USD")
	if err == nil {
		t.Error("Test Failed - Trade() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	_, err := l.GetOpenOrders()
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	_, err := l.GetOrders([]int64{1, 2})
	if err == nil {
		t.Error("Test Failed - GetOrders() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	err := l.CancelExistingOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrder() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	_, err := l.GetTrades(1337)
	if err == nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetExternalAccounts(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	_, err := l.GetExternalAccounts()
	if err == nil {
		t.Error("Test Failed - GetExternalAccounts() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.LTC.String(),
			"_"),
		IsMaker:             false,
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	l.GetFeeByType(feeBuilder)
	if apiKey == "" || apiSecret == "" {
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
	// CryptocurrencyTradeFee Basic
	if resp, err := l.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := l.GetFee(feeBuilder); resp != float64(2000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := l.GetFee(feeBuilder); resp != float64(0.0015) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0015), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	l.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText

	withdrawPermissions := l.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := l.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := l.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if l.APIKey != "" && l.APIKey != "Key" &&
		l.APISecret != "" && l.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.BTC,
		Quote:     currency.EUR,
	}
	response, err := l.SubmitOrder(p, exchange.BuyOrderSide, exchange.MarketOrderType, 1, 10, "hi")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := l.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := l.CancelAllOrders(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := l.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "7860767916",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := l.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := l.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := l.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := l.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error", err)
		}
	} else {
		_, err := l.GetDepositAddress(currency.DASH, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error cannot be nil")
		}
	}
}

// TestWsConn websocket connection test
func TestWsConn(t *testing.T) {
	TestSetDefaults(t)
	TestSetup(t)
	if !l.Websocket.IsEnabled() {
		t.Skip(monitor.WebsocketNotEnabled)
	}
	l.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	l.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	err := l.WsConnect()
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsOrderbookProcessing logic test
func TestWsOrderbookProcessing(t *testing.T) {
	TestSetDefaults(t)
	TestSetup(t)
	json := `{"asks":[["11905.66","0.0019"],["11905.73","0.0015"],["11906.43","0.0013"],["11906.62","0.0019"],["11907.25","11.087"],["11907.66","0.0006"],["11907.73","0.3113"],["11907.84","0.0006"],["11908.37","0.0016"],["11908.86","10.3786"],["11909.54","4.2955"],["11910.15","0.0012"],["11910.56","13.5505"],["11911.06","0.0011"],["11911.37","0.0023"]],"bids":[["11905.55","0.0171"],["11904.43","0.0225"],["11903.31","0.0223"],["11902.2","0.0027"],["11901.92","1.002"],["11901.6","0.0015"],["11901.49","0.0012"],["11901.08","0.0227"],["11900.93","0.0009"],["11900.53","1.662"],["11900.08","0.001"],["11900.01","3.6745"],["11899.96","0.003"],["11899.91","0.0006"],["11899.44","0.0013"]]}`
	err := l.processOrderbook(json, "market-btcusd-global")
	if err != nil {
		t.Error(err)
	}
}

// TestWsTradeProcessing logic test
func TestWsTradeProcessing(t *testing.T) {
	TestSetDefaults(t)
	TestSetup(t)
	json := `{"trades":[{"type":"sell","date":1564985787,"price":"11913.02","amount":"0.49"}]}`
	err := l.processTrades(json, "market-btcusd-global")
	if err != nil {
		t.Error(err)
	}
}

// TestWsTickerProcessing logic test
func TestWsTickerProcessing(t *testing.T) {
	TestSetDefaults(t)
	TestSetup(t)
	l.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	l.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	json := `{"btcusd":{"low":"10990.05","high":"11966.24","last":"11903.29","volume":"1803.967079","sell":"11912.39","buy":"11902.2"},"btceur":{"low":"9886.87","high":"10732.72","last":"10691.44","volume":"87.994478","sell":"10711.62","buy":"10691.44"},"btchkd":{"low":null,"high":null,"last":"51776.98","volume":null,"sell":"93307.37","buy":"93177.56"},"btcjpy":{"low":"1176039.0","high":"1272246.0","last":"1265680.0","volume":"129.021421","sell":"1266764.0","buy":"1265680.0"},"btcgbp":{"low":"9157.12","high":"9953.43","last":"9941.28","volume":"10.4997","sell":"10007.89","buy":"9941.28"},"btcaud":{"low":"16102.57","high":"17594.22","last":"17548.16","volume":"7.338316","sell":"17616.67","buy":"17549.69"},"btccad":{"low":"14541.69","high":"15834.87","last":"15763.54","volume":"30.480309","sell":"15793.45","buy":"15756.13"},"btcsgd":{"low":"15133.82","high":"16501.62","last":"16455.53","volume":"4.044026","sell":"16484.37","buy":"16462.18"},"btcchf":{"low":"10800.58","high":"11526.24","last":"11526.24","volume":"0.1765","sell":"11675.34","buy":"11632.02"},"btcnzd":{"low":null,"high":null,"last":"8340.98","volume":null,"sell":"18315.49","buy":"18221.37"},"btcngn":{"low":null,"high":null,"last":"600000.0","volume":null,"sell":null,"buy":null},"eurusd":{"low":"1.1088","high":"1.1138","last":"1.1125","volume":"2680.105249","sell":"1.1142","buy":"1.1121"},"gbpusd":{"low":"1.1934","high":"1.1958","last":"1.1934","volume":"1493.923823","sell":"1.1979","buy":"1.1903"},"usdjpy":{"low":"105.26","high":"107.25","last":"106.33","volume":"114490.2179","sell":"106.34","buy":"106.27"},"usdhkd":{"low":null,"high":null,"last":"7.851","volume":null,"sell":"7.8328","buy":"7.8286"},"usdcad":{"low":"1.3225","high":"1.3272","last":"1.3255","volume":"11033.9877","sell":"1.3258","buy":"1.3238"},"usdsgd":{"low":"1.3776","high":"1.3839","last":"1.3838","volume":"2523.75","sell":"1.3838","buy":"1.3819"},"audusd":{"low":"0.6764","high":"0.6853","last":"0.6771","volume":"5442.608321","sell":"0.6782","buy":"0.6762"},"nzdusd":{"low":null,"high":null,"last":"0.6758","volume":null,"sell":"0.6532","buy":"0.6504"},"usdchf":{"low":"0.9838","high":"0.9838","last":"0.9838","volume":"108.3352","sell":"0.9801","buy":"0.9773"},"usdngn":{"low":null,"high":null,"last":"200.0","volume":null,"sell":null,"buy":null},"ethbtc":{"low":"0.0205","high":"0.025","last":"0.0205","volume":null,"sell":"0.03","buy":"0.0194"},"ltcbtc":{"low":null,"high":null,"last":"0.0114","volume":null,"sell":"0.009","buy":"0.0073"},"bchbtc":{"low":null,"high":null,"last":"0.0544","volume":null,"sell":"0.0322","buy":"0.0274"},"xrpbtc":{"low":"0.000042","high":"0.000042","last":"0.000042","volume":null,"sell":"0.000037","buy":"0.000022"},"baceth":{"low":"0.000035","high":"0.000035","last":"0.000035","volume":null,"sell":"0.0015","buy":null}}`
	err := l.processTicker(json)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyFromChannel(t *testing.T) {
	curr := currency.NewPair(currency.LTC, currency.BTC)
	result := l.getCurrencyFromChannel(fmt.Sprintf("%v%v%v", marketSubstring, curr, globalSubstring))
	if curr != result {
		t.Errorf("currency result is not equal. Expected  %v", curr)
	}
}
