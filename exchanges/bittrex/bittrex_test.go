package bittrex

import (
	"log"
	"os"
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

// Please supply you own test keys here to run better tests.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	currPair                = "USDT-BTC"
	curr                    = "BTC"
)

var b Bittrex

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	bConfig, err := cfg.GetExchangeConfig("Bittrex")
	if err != nil {
		log.Fatal(err)
	}
	bConfig.API.Credentials.Key = apiKey
	bConfig.API.Credentials.Secret = apiSecret
	bConfig.API.AuthenticatedSupport = true

	err = b.Setup(bConfig)
	if err != nil {
		log.Fatal(err)
	}

	if !b.IsEnabled() || !b.API.AuthenticatedSupport ||
		b.Verbose || len(b.BaseCurrencies) < 1 {
		log.Fatal("Bittrex Setup values not set correctly")
	}

	os.Exit(m.Run())
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := b.GetCurrencies()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(currPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketSummaries(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketSummaries()
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketSummary(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketSummary(currPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook(currPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetMarketHistory(currPair)
	if err != nil {
		t.Error(err)
	}
}

func TestPlaceBuyLimit(t *testing.T) {
	t.Parallel()

	_, err := b.PlaceBuyLimit(currPair, 1, 1)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestPlaceSellLimit(t *testing.T) {
	t.Parallel()

	_, err := b.PlaceSellLimit(currPair, 1, 1)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, err = b.GetOpenOrders(currPair)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	_, err := b.CancelExistingOrder("invalid-order")
	if err == nil {
		t.Error("Expected error")
	}
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalances()
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetAccountBalanceByCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalanceByCurrency(curr)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrder("0cb4c4e4-bdc7-4e13-8c13-430e587d2cc1")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, err = b.GetOrder("")
	if areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOrderHistoryForCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderHistoryForCurrency("")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, err = b.GetOrderHistoryForCurrency(currPair)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetwithdrawalHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetWithdrawalHistory("")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, err = b.GetWithdrawalHistory(curr)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetDepositHistory("")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, err = b.GetDepositHistory(currPair)
	if err == nil {
		t.Error("Expected error")
	}
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
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Error(err)
		t.Errorf("Expected: %f, Received: %f", float64(0.0025), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(2500) || err != nil {
		t.Errorf("Expected: %f, Received: %f", float64(2500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Errorf("Expected: %f, Received: %f", float64(0.0025), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0003) || err != nil {
		t.Errorf("Expected: %f, Received: %f", float64(0.0003), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	p, err := currency.NewPairFromString(currPair)
	if err != nil {
		t.Fatal(err)
	}

	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{p},
		AssetType: asset.Spot,
	}

	getOrdersRequest.Pairs[0].Delimiter = "-"

	_, err = b.GetActiveOrders(&getOrdersRequest)
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

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "-",
			Base:      currency.BTC,
			Quote:     currency.LTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := b.SubmitOrder(orderSubmission)
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

	err := b.CancelOrder(orderCancellation)
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

	resp, err := b.CancelAllOrders(orderCancellation)

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

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := b.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
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

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}

	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error(err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("error cannot be nil")
		}
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()

	tm, err := parseTime("2019-11-21T02:08:34.87")
	if err != nil {
		t.Fatal(err)
	}

	if tm.Year() != 2019 ||
		tm.Month() != 11 ||
		tm.Day() != 21 ||
		tm.Hour() != 2 ||
		tm.Minute() != 8 ||
		tm.Second() != 34 {
		t.Error("invalid time values")
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(currPair)
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
	currencyPair, err := currency.NewPairFromString(currPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Fatal(err)
	}
}
