package gemini

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please enter sandbox API keys & assigned roles for better testing procedures

const (
	apiKey1           = ""
	apiSecret1        = ""
	apiKeyRole1       = ""
	sessionHeartBeat1 = false

	apiKey2           = ""
	apiSecret2        = ""
	apiKeyRole2       = ""
	sessionHeartBeat2 = false

	canManipulateRealOrders = false
)

func TestAddSession(t *testing.T) {
	var g1 Gemini
	if Session[1] == nil {
		err := AddSession(&g1, 1, apiKey1, apiSecret1, apiKeyRole1, true, true)
		if err != nil {
			t.Error("Test failed - AddSession() error", err)
		}
		err = AddSession(&g1, 1, apiKey1, apiSecret1, apiKeyRole1, true, true)
		if err == nil {
			t.Error("Test failed - AddSession() error", err)
		}
	}

	if len(Session) <= 1 {
		var g2 Gemini
		err := AddSession(&g2, 2, apiKey2, apiSecret2, apiKeyRole2, false, true)
		if err != nil {
			t.Error("Test failed - AddSession() error", err)
		}
	}
}

func TestSetDefaults(t *testing.T) {
	Session[1].SetDefaults()
	Session[2].SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	geminiConfig, err := cfg.GetExchangeConfig("Gemini")
	if err != nil {
		t.Error("Test Failed - Gemini Setup() init error")
	}

	geminiConfig.API.AuthenticatedSupport = true

	Session[1].Setup(geminiConfig)
	Session[2].Setup(geminiConfig)

	Session[1].API.Credentials.Key = apiKey1
	Session[1].API.Credentials.Secret = apiSecret1

	Session[2].API.Credentials.Key = apiKey2
	Session[2].API.Credentials.Secret = apiSecret2

	Session[1].API.Endpoints.URL = geminiSandboxAPIURL
	Session[2].API.Endpoints.URL = geminiSandboxAPIURL
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetSymbols()
	if err != nil {
		t.Error("Test Failed - GetSymbols() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTicker("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = Session[1].GetTicker("bla")
	if err == nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrderbook("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTrades("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetNotionalVolume(t *testing.T) {
	if apiKey2 != "" && apiSecret2 != "" {
		t.Parallel()
		_, err := Session[2].GetNotionalVolume()
		if err != nil {
			t.Error("Test Failed - GetNotionalVolume() error", err)
		}
	}
}

func TestGetAuction(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetAuction("btcusd")
	if err != nil {
		t.Error("Test Failed - GetAuction() error", err)
	}
}

func TestGetAuctionHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetAuctionHistory("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetAuctionHistory() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].NewOrder("btcusd", 1, 4500,
		exchange.BuyOrderSide.ToLower().ToString(), "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
	_, err = Session[2].NewOrder("btcusd", 1, 4500,
		exchange.BuyOrderSide.ToLower().ToString(), "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelExistingOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrder() error", err)
	}
}

func TestCancelExistingOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelExistingOrders(false)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrders() error", err)
	}
	_, err = Session[2].CancelExistingOrders(true)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrders() error", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOrderStatus() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrders()
	if err == nil {
		t.Error("Test Failed - GetOrders() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetTradeHistory("btcusd", 0)
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTradeVolume()
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetBalances()
	if err == nil {
		t.Error("Test Failed - GetBalances() error", err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetCryptoDepositAddress("LOL123", "btc")
	if err == nil {
		t.Error("Test Failed - GetCryptoDepositAddress() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := Session[1].WithdrawCrypto("LOL123", "btc", 1)
	if err == nil {
		t.Error("Test Failed - WithdrawCrypto() error", err)
	}
}

func TestPostHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := Session[2].PostHeartbeat()
	if err == nil {
		t.Error("Test Failed - PostHeartbeat() error", err)
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
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)
	var feeBuilder = setFeeBuilder()
	Session[1].GetFeeByType(feeBuilder)
	if apiKey1 == "" || apiSecret1 == "" {
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
	var feeBuilder = setFeeBuilder()
	if apiKey1 != "" && apiSecret1 != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0.01) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.01), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(100) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(100), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0.01) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.01), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)

	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText

	withdrawPermissions := Session[1].FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType:  exchange.AnyOrderType,
		Currencies: []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
	}

	_, err := Session[1].GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType:  exchange.AnyOrderType,
		Currencies: []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)},
	}

	_, err := Session[1].GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return Session[1].ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "_",
		Base:      currency.LTC,
		Quote:     currency.BTC,
	}
	response, err := Session[1].SubmitOrder(p, exchange.BuyOrderSide,
		exchange.LimitOrderType, 1, 10, "1234234")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
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

	err := Session[1].CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
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

	resp, err := Session[1].CancelAllOrders(orderCancellation)

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
	_, err := Session[1].ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := Session[1].WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := Session[1].WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	TestAddSession(t)
	TestSetDefaults(t)
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := Session[1].WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := Session[1].GetDepositAddress(currency.BTC, "")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress error cannot be nil")
	}
}
