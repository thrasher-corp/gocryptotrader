package bithumb

import (
	"errors"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	testCurrency            = "btc"
)

var b Bithumb

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Bithumb load config error", err)
	}
	bitConfig, err := cfg.GetExchangeConfig("Bithumb")
	if err != nil {
		log.Fatal("Bithumb Setup() init error")
	}

	bitConfig.API.AuthenticatedSupport = true
	bitConfig.API.Credentials.Key = apiKey
	bitConfig.API.Credentials.Secret = apiSecret

	err = b.Setup(bitConfig)
	if err != nil {
		log.Fatal("Bithumb setup error", err)
	}

	os.Exit(m.Run())
}

func TestGetTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetTradablePairs()
	if err != nil {
		t.Error("Bithumb GetTradablePairs() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(testCurrency)
	if err != nil {
		t.Error("Bithumb GetTicker() error", err)
	}
}

func TestGetAllTickers(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllTickers()
	if err != nil {
		t.Error("Bithumb GetAllTickers() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook(testCurrency)
	if err != nil {
		t.Error("Bithumb GetOrderBook() error", err)
	}
}

func TestGetTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetTransactionHistory(testCurrency)
	if err != nil {
		t.Error("Bithumb GetTransactionHistory() error", err)
	}
}

func TestGetAccountInformation(t *testing.T) {
	t.Parallel()

	// Offline test
	_, err := b.GetAccountInformation("", "")
	if err == nil {
		t.Error("expected error when no order currency is specified")
	}

	if !areTestAPIKeysSet() {
		t.Skip()
	}

	_, err = b.GetAccountInformation(testCurrency, currency.KRW.String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}

	_, err := b.GetAccountBalance(testCurrency)
	if err == nil {
		t.Error("Bithumb GetAccountBalance() Expected error")
	}
}

func TestGetWalletAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}

	_, err := b.GetWalletAddress("")
	if err == nil {
		t.Error("Bithumb GetWalletAddress() Expected error")
	}
}

func TestGetLastTransaction(t *testing.T) {
	t.Parallel()
	_, err := b.GetLastTransaction()
	if err == nil {
		t.Error("Bithumb GetLastTransaction() Expected error")
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrders("1337", order.Bid.Lower(), "100", "", testCurrency)
	if err == nil {
		t.Error("Bithumb GetOrders() Expected error")
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()
	_, err := b.GetUserTransactions()
	if err == nil {
		t.Error("Bithumb GetUserTransactions() Expected error")
	}
}

func TestPlaceTrade(t *testing.T) {
	t.Parallel()
	_, err := b.PlaceTrade(testCurrency, order.Bid.Lower(), 0, 0)
	if err == nil {
		t.Error("Bithumb PlaceTrade() Expected error")
	}
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderDetails("1337", order.Bid.Lower(), testCurrency)
	if err == nil {
		t.Error("Bithumb GetOrderDetails() Expected error")
	}
}

func TestCancelTrade(t *testing.T) {
	t.Parallel()
	_, err := b.CancelTrade("", "", "")
	if err == nil {
		t.Error("Bithumb CancelTrade() Expected error")
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawCrypto("LQxiDhKU7idKiWQhx4ALKYkBx8xKEQVxJR", "", "ltc", 0)
	if err == nil {
		t.Error("Bithumb WithdrawCrypto() Expected error")
	}
}

func TestRequestKRWDepositDetails(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip()
	}
	_, err := b.RequestKRWDepositDetails()
	if err == nil {
		t.Error("Bithumb RequestKRWDepositDetails() Expected error")
	}
}

func TestRequestKRWWithdraw(t *testing.T) {
	t.Parallel()
	_, err := b.RequestKRWWithdraw("102_bank", "1337", 1000)
	if err == nil {
		t.Error("Bithumb RequestKRWWithdraw() Expected error")
	}
}

func TestMarketBuyOrder(t *testing.T) {
	t.Parallel()
	p := currency.NewPair(currency.BTC, currency.KRW)
	_, err := b.MarketBuyOrder(p, 0)
	if err == nil {
		t.Error("Bithumb MarketBuyOrder() Expected error")
	}
}

func TestMarketSellOrder(t *testing.T) {
	t.Parallel()
	p := currency.NewPair(currency.BTC, currency.KRW)
	_, err := b.MarketSellOrder(p, 0)
	if err == nil {
		t.Error("Bithumb MarketSellOrder() Expected error")
	}
}

func TestUpdateTicker(t *testing.T) {
	cp := currency.NewPair(currency.QTUM, currency.KRW)
	_, err := b.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	cp = currency.NewPair(currency.BTC, currency.KRW)
	_, err = b.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
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
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if _, err := b.GetFee(feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.AutoWithdrawFiatText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		Side:      order.Sell,
		AssetType: asset.Spot,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.LTC,
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
	t.Parallel()
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
		t.Errorf("Could not cancel order: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
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
		t.Errorf("Could not cancel order: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() {
		_, err := b.UpdateAccountInfo(asset.Spot)
		if err != nil {
			t.Error("Bithumb GetAccountInfo() error", err)
		}
	} else {
		_, err := b.UpdateAccountInfo(asset.Spot)
		if err == nil {
			t.Error("Bithumb GetAccountInfo() Expected error")
		}
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	curr, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.ModifyOrder(&order.Modify{
		ID:        "1337",
		Price:     100,
		Amount:    1000,
		Side:      order.Sell,
		Pair:      curr,
		AssetType: asset.Spot,
	})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
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
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{
		Fiat: withdraw.FiatRequest{
			WireCurrency:             currency.KRW.String(),
			RequiresIntermediaryBank: false,
			IsExpressWire:            false,
		},
		Amount:      -1,
		Currency:    currency.USD,
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	if areTestAPIKeysSet() {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("GetDepositAddress() error cannot be nil")
		}
	}
}

func TestGetCandleStick(t *testing.T) {
	_, err := b.GetCandleStick("BTC_KRW", "1m")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTCKRW")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	_, err = b.GetHistoricCandles(currencyPair, asset.Spot, startTime, time.Now(), kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTCKRW")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 24)
	_, err = b.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, time.Now(), kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC_KRW")
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
	currencyPair, err := currency.NewPairFromString("BTC_KRW")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	err := b.UpdateOrderExecutionLimits("")
	if err != nil {
		t.Fatal(err)
	}
	cp := currency.NewPair(currency.BTC, currency.KRW)
	limit, err := b.GetOrderExecutionLimits(asset.Spot, cp)
	if err != nil {
		t.Fatal(err)
	}

	err = limit.Conforms(46241000, 0.00001, order.Limit)
	if !errors.Is(err, order.ErrAmountBelowMin) {
		t.Fatalf("expected error %v but received %v",
			order.ErrAmountBelowMin,
			err)
	}

	err = limit.Conforms(46241000, 0.0001, order.Limit)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error %v but received %v",
			nil,
			err)
	}
}

func TestGetAmountMinimum(t *testing.T) {
	testCases := []struct {
		name      string
		unitprice float64
		expected  float64
	}{
		{
			name:      "ETH-KRW",
			unitprice: 2638000.0,
			expected:  0.0002,
		},
		{
			name:      "DOGE-KRW",
			unitprice: 236.5,
			expected:  2.1142,
		},
		{
			name:      "XRP-KRW",
			unitprice: 818.8,
			expected:  0.6107,
		},
		{
			name:      "LTC-KRW",
			unitprice: 160100,
			expected:  0.0032,
		},
		{
			name:      "BTC-KRW",
			unitprice: 46079000,
			expected:  0.0001,
		},
		{
			name:      "nonsense",
			unitprice: 0,
			expected:  0,
		},
	}

	for i := range testCases {
		tt := &testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			minAmount := getAmountMinimum(tt.unitprice)
			if minAmount != tt.expected {
				t.Fatalf("expected: %f but received: %f for unit price: %f",
					tt.expected,
					minAmount,
					tt.unitprice)
			}
		})
	}
}
