package itbit

import (
	"context"
	"errors"
	"log"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var i ItBit

// Please provide your own keys to do proper testing
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	i.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Itbit load config error", err)
	}
	itbitConfig, err := cfg.GetExchangeConfig("ITBIT")
	if err != nil {
		log.Fatal("Itbit Setup() init error")
	}
	itbitConfig.API.AuthenticatedSupport = true
	itbitConfig.API.Credentials.Key = apiKey
	itbitConfig.API.Credentials.Secret = apiSecret
	itbitConfig.API.Credentials.ClientID = clientID

	err = i.Setup(itbitConfig)
	if err != nil {
		log.Fatal("Itbit setup error", err)
	}

	os.Exit(m.Run())
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := i.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = i.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := i.GetTicker(context.Background(), "XBTUSD")
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := i.GetOrderbook(context.Background(), "XBTUSD")
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := i.GetTradeHistory(context.Background(), "XBTUSD", "0")
	if err != nil {
		t.Error("GetTradeHistory() error", err)
	}
}

func TestGetWallets(t *testing.T) {
	_, err := i.GetWallets(context.Background(), url.Values{})
	if err == nil {
		t.Error("GetWallets() Expected error")
	}
}

func TestCreateWallet(t *testing.T) {
	_, err := i.CreateWallet(context.Background(), "test")
	if err == nil {
		t.Error("CreateWallet() Expected error")
	}
}

func TestGetWallet(t *testing.T) {
	_, err := i.GetWallet(context.Background(), "1337")
	if err == nil {
		t.Error("GetWallet() Expected error")
	}
}

func TestGetWalletBalance(t *testing.T) {
	_, err := i.GetWalletBalance(context.Background(), "1337", "XRT")
	if err == nil {
		t.Error("GetWalletBalance() Expected error")
	}
}

func TestGetWalletTrades(t *testing.T) {
	_, err := i.GetWalletTrades(context.Background(), "1337", url.Values{})
	if err == nil {
		t.Error("GetWalletTrades() Expected error")
	}
}

func TestGetFundingHistory(t *testing.T) {
	_, err := i.GetFundingHistoryForWallet(context.Background(), "1337", url.Values{})
	if err == nil {
		t.Error("GetFundingHistory() Expected error")
	}
}

func TestPlaceOrder(t *testing.T) {
	_, err := i.PlaceOrder(context.Background(),
		"1337", order.Buy.Lower(),
		order.Limit.Lower(), "USD", 1, 0.2, "banjo",
		"sauce")
	if err == nil {
		t.Error("PlaceOrder() Expected error")
	}
}

func TestGetOrder(t *testing.T) {
	_, err := i.GetOrder(context.Background(), "1337", url.Values{})
	if err == nil {
		t.Error("GetOrder() Expected error")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Skip()
	err := i.CancelExistingOrder(context.Background(), "1337", "1337order")
	if err == nil {
		t.Error("CancelOrder() Expected error")
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	_, err := i.GetCryptoDepositAddress(context.Background(), "1337", "AUD")
	if err == nil {
		t.Error("GetCryptoDepositAddress() Expected error")
	}
}

func TestWalletTransfer(t *testing.T) {
	_, err := i.WalletTransfer(context.Background(),
		"1337", "mywallet", "anotherwallet", 200, "USD")
	if err == nil {
		t.Error("WalletTransfer() Expected error")
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
	var feeBuilder = setFeeBuilder()
	_, err := i.GetFeeByType(context.Background(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := i.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.WithdrawCryptoViaWebsiteOnlyText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := i.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := i.GetActiveOrders(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
	}

	_, err := i.GetOrderHistory(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return i.ValidateAPICredentials(i.GetDefaultCredentials()) == nil
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
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
	response, err := i.SubmitOrder(context.Background(), orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := i.CancelOrder(context.Background(), orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := i.CancelAllOrders(context.Background(), orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := i.UpdateAccountInfo(context.Background(), asset.Spot)
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := i.ModifyOrder(context.Background(), &order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    i.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := i.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected 'Not supported', received %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := i.WithdrawFiatFunds(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := i.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := i.GetDepositAddress(context.Background(), currency.BTC, "", "")
	if err == nil {
		t.Error("GetDepositAddress() error cannot be nil")
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("XBTUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = i.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("XBTUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = i.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}
