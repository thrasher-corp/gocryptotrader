package localbitcoins

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var l LocalBitcoins

func TestGetTicker(t *testing.T) {
	t.Parallel()

	_, err := l.GetTicker()
	if err != nil {
		t.Errorf("GetTicker() returned: %s", err)
	}
}

func TestGetTradableCurrencies(t *testing.T) {
	t.Parallel()

	_, err := l.GetTradableCurrencies()
	if err != nil {
		t.Errorf("GetTradableCurrencies() returned: %s", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := l.GetAccountInformation("", true)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get AccountInformation: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get AccountInformation: %s", err)
	}
}

func TestGetads(t *testing.T) {
	t.Parallel()
	_, err := l.Getads("")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get ads: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Expecting error until QA pass")
	}
}

func TestEditAd(t *testing.T) {
	t.Parallel()

	var edit AdEdit
	err := l.EditAd(&edit, "1337")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not edit order: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Expecting error until QA pass")
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

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := l.GetTrades("LTC", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	ob, err := l.GetOrderbook("AUD")
	if err != nil {
		t.Fatal(err)
	}
	if mockTests {
		if ob.Bids[0].Price != 88781.42 && ob.Bids[0].Amount != 10000.00 ||
			ob.Asks[0].Price != 14400.00 && ob.Asks[0].Amount != 0.77 {
			t.Error("incorrect orderbook values")
		}
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	l.GetFeeByType(feeBuilder)
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
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := l.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	expectedResult := exchange.AutoWithdrawCryptoText +
		" & " +
		exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := l.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s",
			expectedResult,
			withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := l.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not get active orders: %s", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Errorf("Could not get active orders: %s", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := l.GetOrderHistory(&getOrdersRequest)
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
func areTestAPIKeysSet() bool {
	return l.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.EUR,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := l.SubmitOrder(orderSubmission)
	switch {
	case areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) && !mockTests:
		t.Errorf("Order failed to be placed: %v", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}

	err := l.CancelOrder(orderCancellation)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Could not cancel orders: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass")
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}

	resp, err := l.CancelAllOrders(orderCancellation)
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

	_, err := l.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err != common.ErrFunctionNotSupported {
		t.Error("ModifyOrder() error", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := l.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Expecting an error when no keys are set")
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Errorf("Withdraw failed to be placed: %v", err)
	case mockTests && err == nil:
		t.Error("Expecting an error until QA pass")
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()

	var withdrawFiatRequest = withdraw.Request{}
	_, err := l.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()

	var withdrawFiatRequest = withdraw.Request{}
	_, err := l.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'",
			common.ErrFunctionNotSupported,
			err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := l.GetDepositAddress(currency.BTC, "")
	switch {
	case areTestAPIKeysSet() && err != nil && !mockTests:
		t.Error("GetDepositAddress() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetDepositAddress() expecting an error when no APIKeys are set")
	case mockTests && err != nil:
		t.Error("GetDepositAddress() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC-LTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC-LTC")
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}
