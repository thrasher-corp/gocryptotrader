package anx

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var a ANX

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := a.GetCurrencies()
	if err != nil {
		t.Fatalf("TestGetCurrencies failed. Err: %s", err)
	}
}

func TestGetTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := a.FetchTradablePairs(asset.Spot)
	if err != nil {
		t.Fatalf("TestGetTradablePairs failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	ticker, err := a.GetTicker("BTCUSD")
	if err != nil {
		t.Errorf("ANX GetTicker() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("ANX GetTicker() unsuccessful")
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	depth, err := a.GetDepth("BTCUSD")
	if err != nil {
		t.Errorf("ANX GetDepth() error: %s", err)
	}
	if depth.Result != "success" {
		t.Error("ANX GetDepth() unsuccessful")
	}
}

func TestGetAPIKey(t *testing.T) {
	t.Parallel()
	apiKey, apiSecret, err := a.GetAPIKey("userName", "passWord", "", "1337")
	if err == nil {
		t.Error("ANX GetAPIKey() Expected error")
	}
	if apiKey != "" {
		t.Error("ANX GetAPIKey() Expected error")
	}
	if apiSecret != "" {
		t.Error("ANX GetAPIKey() Expected error")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		IsMaker:       false,
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	a.GetFeeByType(feeBuilder)
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

	// CryptocurrencyTradeFee Basic
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.02) || err != nil {
		t.Error(err)
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := a.GetFee(feeBuilder); resp != float64(20000) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(20000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.01) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.01), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := a.GetFee(feeBuilder); resp != float64(250.01) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(250.01), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawCryptoWith2FAText + " & " +
		exchange.WithdrawCryptoWithEmailText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := a.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := a.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get open orders: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get open orders: %s", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := a.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get order history: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetBalance() error", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func areTestAPIKeysSet() bool {
	return a.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.BTC,
			Quote:     currency.USD,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
	}
	response, err := a.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) && !mockTests {
		// TODO: QA Pass to submit order
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.BTC, currency.LTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := a.CancelOrder(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel order: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.BTC, currency.LTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := a.CancelAllOrders(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel order: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Errorf("QA pass needs to be completed and mock needs to be updated error cannot be nil")
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := a.GetAccountInfo()
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetAccountInfo() error:", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetAccountInfo() error")
	case mockTests && err != nil:
		t.Error("GetAccountInfo() error:", err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := a.ModifyOrder(&order.Modify{})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := exchange.CryptoWithdrawRequest{
		GenericWithdrawRequestInfo: exchange.GenericWithdrawRequestInfo{
			Amount:      -1,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
		},
		Address:    "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AddressTag: "0123456789",
	}

	_, err := a.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if areTestAPIKeysSet() && err != nil && !mockTests {
		t.Errorf("Withdraw failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil && mockTests {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	var withdrawFiatRequest = exchange.FiatWithdrawRequest{}
	_, err := a.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	var withdrawFiatRequest = exchange.FiatWithdrawRequest{}
	_, err := a.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := a.GetDepositAddress(currency.BTC, "")
	if areTestAPIKeysSet() && err != nil && !mockTests {
		t.Error("GetDepositAddress() error", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("GetDepositAddress() error cannot be nil")
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	q := currency.Pair{
		Delimiter: "_",
		Base:      currency.BTC,
		Quote:     currency.USD}

	_, err := a.UpdateOrderbook(q, "spot")
	if err == nil {
		t.Fatalf("error cannot be nil as the endpoint returns no orderbook information")
	}
}
