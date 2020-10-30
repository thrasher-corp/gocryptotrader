package bitstamp

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please add your private keys and customerID for better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	customerID              = "" // This is the customer id you use to log in
	canManipulateRealOrders = false
)

var b Bitstamp

func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

	var feeBuilder = setFeeBuilder()
	b.GetFeeByType(feeBuilder)
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

	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Error(err)
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || (areTestAPIKeysSet() && err != nil) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(0),
			resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(7.5) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(7.5),
			resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(15) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f",
			float64(15),
			resp)
		t.Error(err)
	}
}

func TestCalculateTradingFee(t *testing.T) {
	t.Parallel()

	newBalance := make(Balances)
	newBalance["BTC"] = Balance{
		USDFee: 1,
		EURFee: 0,
	}

	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 0, 0, newBalance); resp != 0 {
		t.Error("GetFee() error")
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.USD, 2, 2, newBalance); resp != float64(4) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(4), resp)
	}
	if resp := b.CalculateTradingFee(currency.BTC, currency.EUR, 2, 2, newBalance); resp != float64(0) {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
	}

	dummy1, dummy2 := currency.NewCode(""), currency.NewCode("")
	if resp := b.CalculateTradingFee(dummy1, dummy2, 0, 0, newBalance); resp != 0 {
		t.Error("GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()

	_, err := b.GetTicker(currency.BTC.String()+currency.USD.String(), false)
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook(currency.BTC.String() + currency.USD.String())
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTradingPairs(t *testing.T) {
	t.Parallel()

	_, err := b.GetTradingPairs()
	if err != nil {
		t.Error("GetTradingPairs() error", err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	_, err := b.GetTransactions(currency.BTC.String()+currency.USD.String(), "hour")
	if err != nil {
		t.Error("GetTransactions() error", err)
	}
}

func TestGetEURUSDConversionRate(t *testing.T) {
	t.Parallel()

	_, err := b.GetEURUSDConversionRate()
	if err != nil {
		t.Error("GetEURUSDConversionRate() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()

	_, err := b.GetBalance()
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetBalance() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetBalance() error", err)
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()

	_, err := b.GetUserTransactions("btcusd")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetUserTransactions() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetUserTransactions() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("btcusd")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetOpenOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetOpenOrders() error", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetOrderStatus() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Expecting an error until a QA pass can be completed")
	}
}

func TestGetWithdrawalRequests(t *testing.T) {
	t.Parallel()

	_, err := b.GetWithdrawalRequests(0)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetWithdrawalRequests() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetWithdrawalRequests() error", err)
	}
}

func TestGetUnconfirmedBitcoinDeposits(t *testing.T) {
	t.Parallel()

	_, err := b.GetUnconfirmedBitcoinDeposits()
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetUnconfirmedBitcoinDeposits() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("GetUnconfirmedBitcoinDeposits() error", err)
	}
}

func TestTransferAccountBalance(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() && !mockTests {
		t.Skip()
	}

	err := b.TransferAccountBalance(0.01, "btc", "testAccount", true)
	if !mockTests && err != nil {
		t.Error("TransferAccountBalance() error", err)
	}
	if mockTests && err == nil {
		t.Error("Expecting an error until a QA pass can be completed")
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	expectedResult := exchange.AutoWithdrawCryptoText +
		" & " +
		exchange.AutoWithdrawFiatText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		Type: order.AnyType,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
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
		Type: order.AnyType,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get order history: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get order history: %s", err)
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
			Base:  currency.BTC,
			Quote: currency.USD,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := b.SubmitOrder(orderSubmission)
	switch {
	case areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) && !mockTests:
		t.Errorf("Order failed to be placed: %v", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	orderCancellation := &order.Cancel{
		ID:        "1234",
		AssetType: asset.Spot,
	}
	err := b.CancelOrder(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	resp, err := b.CancelAllOrders(&order.Cancel{AssetType: asset.Spot})
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err != nil:
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()

	_, err := b.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{
		Fiat: withdraw.FiatRequest{
			Bank: banking.Account{
				AccountName:    "Satoshi Nakamoto",
				AccountNumber:  "12345",
				BankAddress:    "123 Fake St",
				BankPostalCity: "Tarry Town",
				BankCountry:    "AU",
				BankName:       "Federal Reserve Bank",
				SWIFTCode:      "CTBAAU2S",
				BankPostalCode: "2088",
				IBAN:           "IT60X0542811101000000123456",
			},
			WireCurrency:             currency.USD.String(),
			RequiresIntermediaryBank: false,
			IsExpressWire:            false,
		},
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{
		Fiat: withdraw.FiatRequest{
			Bank: banking.Account{
				AccountName:    "Satoshi Nakamoto",
				AccountNumber:  "12345",
				BankAddress:    "123 Fake St",
				BankPostalCity: "Tarry Town",
				BankCountry:    "AU",
				BankName:       "Federal Reserve Bank",
				SWIFTCode:      "CTBAAU2S",
				BankPostalCode: "2088",
				IBAN:           "IT60X0542811101000000123456",
			},
			WireCurrency:                  currency.USD.String(),
			RequiresIntermediaryBank:      false,
			IsExpressWire:                 false,
			IntermediaryBankAccountNumber: 12345,
			IntermediaryBankAddress:       "123 Fake St",
			IntermediaryBankCity:          "Tarry Town",
			IntermediaryBankCountry:       "AU",
			IntermediaryBankName:          "Federal Reserve Bank",
			IntermediaryBankPostalCode:    "2088",
		},
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass is completed")
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositAddress(currency.BTC, "")
	switch {
	case areTestAPIKeysSet() && customerID != "" && err != nil && !mockTests:
		t.Error("GetDepositAddress error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetDepositAddress error cannot be nil")
	case mockTests && err != nil:
		t.Error("GetDepositAddress error", err)
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()

	tm, err := parseTime("2019-10-18 01:55:14")
	if err != nil {
		t.Error(err)
	}

	if tm.Year() != 2019 ||
		tm.Month() != 10 ||
		tm.Day() != 18 ||
		tm.Hour() != 1 ||
		tm.Minute() != 55 ||
		tm.Second() != 14 {
		t.Error("invalid time values")
	}
}

func TestWsSubscription(t *testing.T) {
	pressXToJSON := []byte(`{
		"event": "bts:subscribe",
		"data": {
			"channel": "[channel_name]"
		}
	}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUnsubscribe(t *testing.T) {
	pressXToJSON := []byte(`{
		"event": "bts:subscribe",
		"data": {
			"channel": "[channel_name]"
		}
	}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(`{"data": {"microtimestamp": "1580336751488517", "amount": 0.00598803, "buy_order_id": 4621328909, "sell_order_id": 4621329035, "amount_str": "0.00598803", "price_str": "9334.73", "timestamp": "1580336751", "price": 9334.73, "type": 1, "id": 104007706}, "event": "trade", "channel": "live_trades_btcusd"}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{"data": {"timestamp": "1580336834", "microtimestamp": "1580336834607546", "bids": [["9328.28", "0.05925332"], ["9327.34", "0.43120000"], ["9327.29", "0.63470860"], ["9326.59", "0.41114619"], ["9326.38", "1.06910000"], ["9323.91", "2.67930000"], ["9322.69", "0.80000000"], ["9322.57", "0.03000000"], ["9322.31", "1.36010820"], ["9319.54", "0.03090000"], ["9318.97", "0.28000000"], ["9317.61", "0.02910000"], ["9316.39", "1.08000000"], ["9316.20", "2.00000000"], ["9315.48", "1.00000000"], ["9314.72", "0.11197459"], ["9314.47", "0.32207398"], ["9312.53", "0.03961501"], ["9312.29", "1.00000000"], ["9311.78", "0.03060000"], ["9311.69", "0.32217221"], ["9310.98", "3.29000000"], ["9310.18", "0.01304192"], ["9310.13", "0.02500000"], ["9309.04", "1.00000000"], ["9309.00", "0.05000000"], ["9308.96", "0.03030000"], ["9308.91", "0.32227154"], ["9307.52", "0.32191362"], ["9307.25", "2.44280000"], ["9305.92", "3.00000000"], ["9305.62", "2.37600000"], ["9305.60", "0.21815312"], ["9305.54", "2.80000000"], ["9305.13", "0.05000000"], ["9305.02", "2.90917302"], ["9303.68", "0.02316372"], ["9303.53", "12.55000000"], ["9303.00", "0.02191430"], ["9302.94", "2.38250000"], ["9302.37", "0.01000000"], ["9301.85", "2.50000000"], ["9300.89", "0.02000000"], ["9300.40", "4.10000000"], ["9300.00", "0.33936139"], ["9298.48", "1.45200000"], ["9297.80", "0.42380000"], ["9295.44", "4.54689328"], ["9295.43", "3.20000000"], ["9295.00", "0.28669566"], ["9291.66", "14.09931321"], ["9290.13", "2.87254900"], ["9290.00", "0.67530840"], ["9285.37", "0.38033002"], ["9285.15", "5.37993528"], ["9285.00", "0.09419278"], ["9283.71", "0.15679830"], ["9280.33", "12.55000000"], ["9280.13", "3.20310000"], ["9280.00", "1.36477909"], ["9276.01", "0.00707488"], ["9275.75", "0.56974291"], ["9275.00", "5.88000000"], ["9274.00", "0.00754205"], ["9271.68", "0.01400000"], ["9271.11", "15.37188500"], ["9270.00", "0.06674325"], ["9268.79", "24.54320000"], ["9257.18", "12.55000000"], ["9256.30", "0.17876365"], ["9255.71", "13.82642967"], ["9254.79", "0.96329407"], ["9250.00", "0.78214958"], ["9245.34", "4.90200000"], ["9245.13", "0.10000000"], ["9240.00", "0.44383459"], ["9238.84", "13.16615207"], ["9234.11", "0.43317656"], ["9234.10", "12.55000000"], ["9231.28", "11.79290000"], ["9230.09", "4.15059441"], ["9227.69", "0.00791097"], ["9225.00", "0.44768346"], ["9224.49", "0.85857203"], ["9223.50", "5.61001041"], ["9216.01", "0.03222653"], ["9216.00", "0.05000000"], ["9213.54", "0.71253866"], ["9212.50", "2.86768195"], ["9211.07", "12.55000000"], ["9210.00", "0.54288817"], ["9208.00", "1.00000000"], ["9206.06", "2.62587578"], ["9205.98", "15.40000000"], ["9205.52", "0.01710603"], ["9205.37", "0.03524953"], ["9205.11", "0.15000000"], ["9205.00", "0.01534763"], ["9204.76", "7.00600000"], ["9203.00", "0.01090000"]], "asks": [["9337.10", "0.03000000"], ["9340.85", "2.67820000"], ["9340.95", "0.02900000"], ["9341.17", "1.00000000"], ["9341.41", "2.13966390"], ["9341.61", "0.20000000"], ["9341.97", "0.11199911"], ["9341.98", "3.00000000"], ["9342.26", "0.32112762"], ["9343.87", "1.00000000"], ["9344.17", "3.57250000"], ["9345.04", "0.32103450"], ["9345.41", "4.90000000"], ["9345.69", "1.03000000"], ["9345.80", "0.03000000"], ["9346.00", "0.10200000"], ["9346.69", "0.02397394"], ["9347.41", "1.00000000"], ["9347.82", "0.32094177"], ["9348.23", "0.02880000"], ["9348.62", "11.96287551"], ["9349.31", "2.44270000"], ["9349.47", "0.96000000"], ["9349.86", "4.50000000"], ["9350.37", "0.03300000"], ["9350.57", "0.34682266"], ["9350.60", "0.32085527"], ["9351.45", "0.31147923"], ["9352.31", "0.28000000"], ["9352.86", "9.80000000"], ["9353.73", "0.02360739"], ["9354.00", "0.45000000"], ["9354.12", "0.03000000"], ["9354.29", "3.82446861"], ["9356.20", "0.64000000"], ["9356.90", "0.02316372"], ["9357.30", "2.50000000"], ["9357.70", "2.38240000"], ["9358.92", "6.00000000"], ["9359.97", "0.34898075"], ["9359.98", "2.30000000"], ["9362.56", "2.37600000"], ["9365.00", "0.64000000"], ["9365.16", "1.70030306"], ["9365.27", "3.03000000"], ["9369.99", "2.47102665"], ["9370.00", "3.15688574"], ["9370.21", "2.32720000"], ["9371.78", "13.20000000"], ["9371.89", "0.96293482"], ["9375.08", "4.74762500"], ["9384.34", "1.45200000"], ["9384.49", "16.42310000"], ["9385.66", "0.34382112"], ["9388.19", "0.00268265"], ["9392.20", "0.20980000"], ["9392.40", "0.10320000"], ["9393.00", "0.20980000"], ["9395.40", "0.40000000"], ["9398.86", "24.54310000"], ["9400.00", "0.05489988"], ["9400.33", "0.00495100"], ["9400.45", "0.00484700"], ["9402.92", "17.20000000"], ["9404.18", "10.00000000"], ["9418.89", "16.38000000"], ["9419.41", "3.06700000"], ["9420.40", "12.50000000"], ["9421.11", "0.10500000"], ["9434.47", "0.03215805"], ["9434.48", "0.28285714"], ["9434.49", "15.83000000"], ["9435.13", "0.15000000"], ["9438.93", "0.00368800"], ["9439.19", "0.69343985"], ["9442.86", "0.10000000"], ["9443.96", "12.50000000"], ["9444.00", "0.06004471"], ["9444.97", "0.01494896"], ["9447.00", "0.01234000"], ["9448.97", "0.14500000"], ["9449.00", "0.05000000"], ["9450.00", "11.13426018"], ["9451.87", "15.90000000"], ["9452.00", "0.20000000"], ["9454.25", "0.01100000"], ["9454.51", "0.02409062"], ["9455.05", "0.00600063"], ["9456.00", "0.27965118"], ["9456.10", "0.17000000"], ["9459.00", "0.00320000"], ["9459.98", "0.02460685"], ["9459.99", "8.11000000"], ["9460.00", "0.08500000"], ["9464.36", "0.56957951"], ["9464.54", "0.69158059"], ["9465.00", "21.00002015"], ["9467.57", "12.50000000"], ["9468.00", "0.08800000"], ["9469.09", "13.94000000"]]}, "event": "data", "channel": "order_book_btcusd"}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderbook2(t *testing.T) {
	pressXToJSON := []byte(`{"data": {"timestamp": "1580336904", "microtimestamp": "1580336904228758", "bids": [["9317.09", "2.67910000"], ["9296.14", "0.00000000"], ["9294.36", "0.04967421"]], "asks": [["9333.85", "0.00000000"], ["9339.20", "0.20000000"], ["9339.66", "0.00000000"], ["9342.63", "4.90000000"], ["9343.66", "0.00000000"], ["9343.76", "2.87275947"]]}, "event": "data", "channel": "diff_order_book_btcusd"}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderUpdate(t *testing.T) {
	pressXToJSON := []byte(`{"data": {"microtimestamp": "1580336940972599", "amount": 0.6347086, "order_type": 0, "amount_str": "0.63470860", "price_str": "9350.49", "price": 9350.49, "id": 4621332237, "datetime": "1580336940"}, "event": "order_created", "channel": "live_orders_btcusd"}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsRequestReconnect(t *testing.T) {
	pressXToJSON := []byte(`{
		"event": "bts:request_reconnect",
		"channel": "",
		"data": ""
	}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestBitstamp_OHLC(t *testing.T) {
	start := time.Unix(1546300800, 0)
	end := time.Unix(1577836799, 0)
	_, err := b.OHLC("btcusd", start, end, "60", "10")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBitstamp_GetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Unix(1546300800, 0)
	end := time.Unix(1577836799, 0)

	_, err = b.GetHistoricCandles(currencyPair, asset.Spot, start, end, kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBitstamp_GetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Unix(1546300800, 0)
	end := time.Unix(1577836799, 0)
	_, err = b.GetHistoricCandlesExtended(currencyPair, asset.Spot, start, end, kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("LTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("LTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}
