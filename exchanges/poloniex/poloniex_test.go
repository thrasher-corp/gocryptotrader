package poloniex

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var p Poloniex

func areTestAPIKeysSet() bool {
	return p.ValidateAPICredentials()
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := p.GetTicker()
	if err != nil {
		t.Error("Poloniex GetTicker() error", err)
	}
}

func TestGetVolume(t *testing.T) {
	t.Parallel()
	_, err := p.GetVolume()
	if err != nil {
		t.Error("Test faild - Poloniex GetVolume() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := p.GetOrderbook("BTC_XMR", 50)
	if err != nil {
		t.Error("Test faild - Poloniex GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := p.GetTradeHistory("BTC_XMR", "", "")
	if err != nil {
		t.Error("Test faild - Poloniex GetTradeHistory() error", err)
	}
}

func TestGetChartData(t *testing.T) {
	t.Parallel()
	_, err := p.GetChartData("BTC_XMR", "1405699200", "1405699400", "300")
	if err != nil {
		t.Error("Test faild - Poloniex GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := p.GetCurrencies()
	if err != nil {
		t.Error("Test faild - Poloniex GetCurrencies() error", err)
	}
}

func TestGetLoanOrders(t *testing.T) {
	t.Parallel()
	_, err := p.GetLoanOrders("BTC")
	if err != nil {
		t.Error("Test faild - Poloniex GetLoanOrders() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.LTC.String(),
			currency.BTC.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()
	p.GetFeeByType(feeBuilder)
	if !areTestAPIKeysSet() {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.OfflineTradeFee,
				feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v",
				exchange.CryptocurrencyTradeFee,
				feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() || mockTests {
		// CryptocurrencyTradeFee Basic
		if resp, err := p.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f",
				float64(0.0025), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := p.GetFee(feeBuilder); resp != float64(2500) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f",
				float64(2500), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f",
				float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := p.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText +
		" & " +
		exchange.NoFiatWithdrawalsText
	withdrawPermissions := p.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := p.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() err", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := p.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not get order history: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not mock get order history: %s", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: delimiterUnderscore,
			Base:      currency.BTC,
			Quote:     currency.LTC,
		},
		OrderSide: order.Buy,
		OrderType: order.Market,
		Price:     10,
		Amount:    10000000,
		ClientID:  "hi",
	}

	response, err := p.SubmitOrder(orderSubmission)
	switch {
	case areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced):
		t.Errorf("Order failed to be placed: %v", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() err", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		CurrencyPair:  currency.NewPair(currency.LTC, currency.BTC),
	}

	err := p.CancelOrder(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() err", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := p.CancelAllOrders(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Error("Mock CancelAllExchangeOrders() err", err)
	}
	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := p.ModifyOrder(&order.Modify{OrderID: "1337", Price: 1337})
	switch {
	case areTestAPIKeysSet() && err != nil && mockTests:
		t.Error("ModifyOrder() error", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("ModifyOrder() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock ModifyOrder() err", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	withdrawCryptoRequest := withdraw.Request{
		Crypto: &withdraw.CryptoRequest{
			Address:   core.BitcoinDonationAddress,
			FeeAmount: 1,
		},
		Amount:        0,
		Currency:      currency.LTC,
		Description:   "WITHDRAW IT ALL",
		TradePassword: "Password",
	}
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := p.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock Withdraw() err", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest withdraw.Request
	_, err := p.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := p.GetDepositAddress(currency.DASH, "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetDepositAddress()", err)
	case !areTestAPIKeysSet() && !mockTests && err == nil:
		t.Error("GetDepositAddress() cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock GetDepositAddress() err", err)
	}
}

func TestWsHandleAccountData(t *testing.T) {
	t.Parallel()
	p.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	jsons := []string{
		`[["n",225,807230187,0,"1000.00000000","0.10000000","2018-11-07 16:42:42"],["b",267,"e","-0.10000000"]]`,
		`[["o",807230187,"0.00000000"],["b",267,"e","0.10000000"]]`,
		`[["t", 12345, "0.03000000", "0.50000000", "0.00250000", 0, 6083059, "0.00000375", "2018-09-08 05:54:09"]]`,
	}
	for i := range jsons {
		var result [][]interface{}
		err := json.Unmarshal([]byte(jsons[i]), &result)
		if err != nil {
			t.Error(err)
		}
		p.wsHandleAccountData(result)
	}
}

// TestWsAuth dials websocket, sends login request.
// Will receive a message only on failure
func TestWsAuth(t *testing.T) {
	t.Parallel()
	if !p.Websocket.IsEnabled() && !p.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(wshandler.WebsocketNotEnabled)
	}
	p.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         p.Name,
		URL:                  p.Websocket.GetWebsocketURL(),
		Verbose:              p.Verbose,
		ResponseMaxLimit:     exchange.DefaultWebsocketResponseMaxLimit,
		ResponseCheckTimeout: exchange.DefaultWebsocketResponseCheckTimeout,
	}
	var dialer websocket.Dialer
	err := p.WebsocketConn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	p.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	p.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	go p.WsHandleData()
	err = p.wsSendAuthorisedCommand("subscribe")
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
	select {
	case response := <-p.Websocket.DataHandler:
		t.Error(response)
	case <-timer.C:
	}
	timer.Stop()
}
