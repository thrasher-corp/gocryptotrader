package btse

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b BTSE

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	btseConfig, err := cfg.GetExchangeConfig("BTSE")
	if err != nil {
		log.Fatal("BTSE Setup() init error", err)
	}

	btseConfig.AuthenticatedAPISupport = true
	btseConfig.APIKey = apiKey
	btseConfig.APISecret = apiSecret

	b.Setup(&btseConfig)
	os.Exit(m.Run())
}

func TestGetMarketsSummary(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketsSummary()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.FetchOrderBook("BTC-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades("BTC-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("BTC-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketStatistics(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketStatistics("BTC-USD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetAccountBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetFills("", "BTC-USD", "", "", "", "")
	if err != nil {
		t.Error(err)
	}

}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.CreateOrder(4.5, 3.4, "buy", "limit", "BTC-USD", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	_, err := b.GetOrders("")
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys not set, skipping test")
	}
	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}
	_, err := b.GetOrderHistory(&getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
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
	t.Parallel()

	feeBuilder := &exchange.FeeBuilder{
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair:    currency.NewPair(currency.BTC, currency.USD),
		IsMaker: true,
		Amount:  1000,
	}

	if resp, err := b.GetFee(feeBuilder); resp != 0.00050 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 0.00050, resp)
		t.Error(err)
	}

	feeBuilder.IsMaker = false
	if resp, err := b.GetFee(feeBuilder); resp != 0.001000 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 0.001000, resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != 0.0005 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 0.0005, resp)
		t.Error(err)
	}

	feeBuilder.Pair.Base = currency.USDT
	if resp, err := b.GetFee(feeBuilder); resp != 1.080000 || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", 1.080000, resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(3) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(3), resp)
		t.Error(err)
	}

	feeBuilder.Amount = 1000000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(1000) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(1000), resp)
		t.Error(err)
	}

	feeBuilder.Amount = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(25) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(25), resp)
		t.Error(err)
	}
}

func TestParseOrderTime(t *testing.T) {
	expected := int64(1534794360)
	actual := parseOrderTime("2018-08-20 19:20:46").Unix()
	if expected != actual {
		t.Errorf("TestParseOrderTime expected: %d, got %d", expected, actual)
	}
}

func areTestAPIKeysSet() bool {
	if b.APIKey != "" && b.APIKey != "Key" &&
		b.APISecret != "" && b.APISecret != "Secret" {
		return true
	}
	return false
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
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
	t.Parallel()

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
	t.Parallel()

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

	if err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}
