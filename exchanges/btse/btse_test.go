package btse

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b BTSE

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	btseConfig, err := cfg.GetExchangeConfig("BTSE")
	if err != nil {
		t.Error("Test Failed - BTSE Setup() init error")
	}

	btseConfig.API.AuthenticatedSupport = true
	btseConfig.API.Credentials.Key = apiKey
	btseConfig.API.Credentials.Secret = apiSecret

	b.Setup(btseConfig)
}

func TestGetMarkets(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetMarkets()
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetTrades("BTC-USD")
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetTicker("BTC-USD")
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetMarketStatistics(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetMarketStatistics("BTC-USD")
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetServerTime(t *testing.T) {
	b.SetDefaults()
	_, err := b.GetServerTime()
	if err != nil {
		t.Fatalf("Test failed. Err: %s", err)
	}
}

func TestGetAccount(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	_, err := b.GetAccountBalance()
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get account balance: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetFills(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	_, err := b.GetFills("", "BTC-USD", "", "", "")
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get fills: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}

}

func TestGetActiveOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if err != common.ErrFunctionNotSupported {
		t.Fatal("Test failed. Expected different result")
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	b.SetDefaults()
	expected := exchange.NoAPIWithdrawalMethodsText
	actual := b.FormatWithdrawPermissions()
	if actual != expected {
		t.Errorf("Expected: %s, Received: %s", expected, actual)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := &exchange.FeeBuilder{
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair:    currency.NewPair(currency.BTC, currency.USD),
		IsMaker: true,
		Amount:  1000,
	}

	b.GetFeeByType(feeBuilder)
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
	b.SetDefaults()
	TestSetup(t)

	feeBuilder := &exchange.FeeBuilder{
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair:    currency.NewPair(currency.BTC, currency.USD),
		IsMaker: true,
		Amount:  1000,
	}

	if resp, err := b.GetFee(feeBuilder); resp != 0.00050 || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", 0.00050, resp)
		t.Error(err)
	}

	feeBuilder.IsMaker = false
	if resp, err := b.GetFee(feeBuilder); resp != 0.0015 || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", 0.0015, resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != 0.0005 || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", 0.0005, resp)
		t.Error(err)
	}

	feeBuilder.Pair.Base = currency.USDT
	if resp, err := b.GetFee(feeBuilder); resp != float64(5) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(5), resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(3) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(3), resp)
		t.Error(err)
	}

	feeBuilder.Amount = 1000000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(1000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(1000), resp)
		t.Error(err)
	}

	feeBuilder.Amount = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(25) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(25), resp)
		t.Error(err)
	}
}

func TestParseOrderTime(t *testing.T) {
	expected := int64(1534794360)
	actual := parseOrderTime("2018-08-20 19:20:46").Unix()
	if expected != actual {
		t.Errorf("Test Failed. TestParseOrderTime expected: %d, got %d", expected, actual)
	}
}

func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.BTC,
		Quote:     currency.USD,
	}
	response, err := b.SubmitOrder(p, exchange.SellOrderSide, exchange.LimitOrderType, 0.01, 1000000, "clientId")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPairWithDelimiter(currency.BTC.String(),
		currency.USD.String(),
		"-")

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "0b66ccaf-dfd4-4b9f-a30b-2380b9c7b66d",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := b.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPairWithDelimiter(currency.BTC.String(),
		currency.USD.String(),
		"-")

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := b.CancelAllOrders(orderCancellation)

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
