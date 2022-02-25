package bittrex

import (
	"context"
	"errors"
	"log"
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

// Please supply you own test keys here to run better tests.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	currPair                = "BTC-USDT"
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

func TestStart(t *testing.T) {
	t.Parallel()
	err := b.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = b.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := b.GetCurrencies(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(context.Background(), currPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketSummaries(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketSummaries(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketSummary(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketSummary(context.Background(), currPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, _, err := b.GetOrderbook(context.Background(), currPair, 500)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarketHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetMarketHistory(context.Background(), currPair)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRecentCandles(t *testing.T) {
	t.Parallel()

	_, err := b.GetRecentCandles(context.Background(),
		currPair, "HOUR_1", "MIDPOINT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricalCandles(t *testing.T) {
	t.Parallel()

	_, err := b.GetHistoricalCandles(context.Background(),
		currPair, "MINUTE_5", "MIDPOINT", 2020, 12, 31)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetHistoricalCandles(context.Background(),
		currPair, "MINUTE_5", "MIDPOINT", 2020, 12, 32)
	if err == nil {
		t.Error("invalid date should give an error")
	}
}

func TestOrder(t *testing.T) {
	t.Parallel()

	_, err := b.Order(context.Background(),
		currPair, order.Buy.String(), order.Limit.String(), "", 1, 1, 0.0)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, _, err := b.GetOpenOrders(context.Background(), "")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, _, err = b.GetOpenOrders(context.Background(), currPair)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	_, err := b.CancelExistingOrder(context.Background(), "invalid-order")
	if err == nil {
		t.Error("Expected error")
	}
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()

	_, err := b.GetBalances(context.Background())
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetAccountBalanceByCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalanceByCurrency(context.Background(), curr)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrder(context.Background(), "0cb4c4e4-bdc7-4e13-8c13-430e587d2cc1")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, err = b.GetOrder(context.Background(), "")
	if areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOrderHistoryForCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderHistoryForCurrency(context.Background(), "")
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
	_, err = b.GetOrderHistoryForCurrency(context.Background(), currPair)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetClosedWithdrawals(t *testing.T) {
	t.Parallel()

	_, err := b.GetClosedWithdrawals(context.Background())
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetClosedWithdrawalsForCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetClosedWithdrawalsForCurrency(context.Background(), curr)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOpenWithdrawals(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenWithdrawals(context.Background())
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetCryptoDepositAddresses(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() {
		t.Skip("api keys not set")
	}
	_, err := b.GetCryptoDepositAddresses(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestProvisionNewDepositAddress(t *testing.T) {
	t.Parallel()

	if !areTestAPIKeysSet() {
		t.Skip("api keys not set")
	}
	_, err := b.ProvisionNewDepositAddress(context.Background(), currency.XRP.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetClosedDeposits(t *testing.T) {
	t.Parallel()

	_, err := b.GetClosedDeposits(context.Background())
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetClosedDepositsForCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetClosedDepositsForCurrency(context.Background(), curr)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetClosedDepositsPaginated(t *testing.T) {
	t.Parallel()

	_, err := b.GetClosedDepositsPaginated(context.Background(), 100)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOpenDeposits(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenDeposits(context.Background())
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestGetOpenDepositsForCurrency(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenDepositsForCurrency(context.Background(), curr)
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or canManipulateRealOrders isnt set correctly")
	}
	_, err := b.Withdraw(context.Background(),
		curr, "", core.BitcoinDonationAddress, 0.0009)
	if err != nil {
		t.Error(err)
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
	_, err := b.GetFeeByType(context.Background(), feeBuilder)
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
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(context.Background(), feeBuilder); err != nil {
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

	getOrdersRequest.Pairs[0].Delimiter = currency.DashDelimiter

	_, err = b.GetActiveOrders(context.Background(), &getOrdersRequest)
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

	_, err := b.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err == nil {
		t.Error("Expected: 'At least one currency is required to fetch order history'. received nil")
	}

	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.BTC, currency.USDT),
	}

	_, err = b.GetOrderHistory(context.Background(), &getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials(b.GetDefaultCredentials()) == nil
}

func TestSubmitOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: currency.DashDelimiter,
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
	response, err := b.SubmitOrder(context.Background(), orderSubmission)
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

	err := b.CancelOrder(context.Background(), orderCancellation)
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

	resp, err := b.CancelAllOrders(context.Background(), orderCancellation)

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
	_, err := b.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("Expected error")
	}
}

func WithdrawCryptocurrencyFunds(t *testing.T) {
	t.Helper()
	withdrawCryptoRequest := withdraw.Request{
		Exchange:    b.Name,
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

	_, err := b.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
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

	_, err := b.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}

	_, err := b.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(context.Background(), currency.XRP, "", "")
		if err != nil {
			t.Error(err)
		}
	} else {
		_, err := b.GetDepositAddress(context.Background(), currency.BTC, "", "")
		if err == nil {
			t.Error("error cannot be nil")
		}
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(currPair)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
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
	_, err = b.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Fatal(err)
	}
}
