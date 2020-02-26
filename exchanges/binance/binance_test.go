package binance

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var b Binance

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

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := b.GetExchangeInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()

	_, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		t.Error("Binance FetchTradablePairs(asset asets.AssetType) error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: "BTCUSDT",
		Limit:  10,
	})

	if err != nil {
		t.Error("Binance GetOrderBook() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetRecentTrades(RecentTradeRequestParams{
		Symbol: "BTCUSDT",
		Limit:  15,
	})

	if err != nil {
		t.Error("Binance GetRecentTrades() error", err)
	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetHistoricalTrades("BTCUSDT", 5, 0)
	if !mockTests && err == nil {
		t.Error("Binance GetHistoricalTrades() expecting error")
	}
	if mockTests && err == nil {
		t.Error("Binance GetHistoricalTrades() error", err)
	}
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetAggregatedTrades("BTCUSDT", 5)
	if err != nil {
		t.Error("Binance GetAggregatedTrades() error", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	_, err := b.GetSpotKline(KlinesRequestParams{
		Symbol:   "BTCUSDT",
		Interval: TimeIntervalFiveMinutes,
		Limit:    24,
	})
	if err != nil {
		t.Error("Binance GetSpotKline() error", err)
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetAveragePrice("BTCUSDT")
	if err != nil {
		t.Error("Binance GetAveragePrice() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()

	_, err := b.GetPriceChangeStats("BTCUSDT")
	if err != nil {
		t.Error("Binance GetPriceChangeStats() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()

	_, err := b.GetTickers()
	if err != nil {
		t.Error("Binance TestGetTickers error", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetLatestSpotPrice("BTCUSDT")
	if err != nil {
		t.Error("Binance GetLatestSpotPrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()

	_, err := b.GetBestPrice("BTCUSDT")
	if err != nil {
		t.Error("Binance GetBestPrice() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()

	_, err := b.QueryOrder("BTCUSDT", "", 1337)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("QueryOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("QueryOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock QueryOrder() error", err)
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.OpenOrders("BTCUSDT")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("OpenOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("OpenOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock OpenOrders() error", err)
	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()

	_, err := b.AllOrders("BTCUSDT", "", "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("AllOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("AllOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock AllOrders() error", err)
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	var feeBuilder = setFeeBuilder()

	if areTestAPIKeysSet() || mockTests {
		// CryptocurrencyTradeFee Basic
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.1) || err != nil {
			t.Error(err)
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(100000) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(100000), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := b.GetFee(feeBuilder); resp != float64(0.1) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.1), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
			t.Error(err)
		}
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := b.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}
	_, err := b.GetActiveOrders(&getOrdersRequest)
	if err == nil {
		t.Error("Expected: 'At least one currency is required to fetch order history'. received nil")
	}

	getOrdersRequest.Currencies = []currency.Pair{
		currency.NewPair(currency.LTC, currency.BTC),
	}

	_, err = b.GetActiveOrders(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetActiveOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetActiveOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetActiveOrders() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()

	var getOrdersRequest = order.GetOrdersRequest{
		OrderType: order.AnyType,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if err == nil {
		t.Error("Expected: 'At least one currency is required to fetch order history'. received nil")
	}

	getOrdersRequest.Currencies = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC)}

	_, err = b.GetOrderHistory(&getOrdersRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetOrderHistory() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetOrderHistory() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetOrderHistory() error", err)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// -----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Delimiter: "_",
			Base:      currency.LTC,
			Quote:     currency.BTC,
		},
		OrderSide: order.Buy,
		OrderType: order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "meowOrder",
	}

	_, err := b.SubmitOrder(orderSubmission)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("SubmitOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("SubmitOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock SubmitOrder() error", err)
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

	err := b.CancelOrder(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("CancelExchangeOrder() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("CancelExchangeOrder() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock CancelExchangeOrder() error", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
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

	_, err := b.CancelAllOrders(orderCancellation)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("CancelAllExchangeOrders() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("CancelAllExchangeOrders() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock CancelAllExchangeOrders() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()

	_, err := b.UpdateAccountInfo()
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetAccountInfo() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetAccountInfo() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock GetAccountInfo() error", err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()

	_, err := b.ModifyOrder(&order.Modify{})
	if err == nil {
		t.Error("ModifyOrder() error cannot be nil")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	if areTestAPIKeysSet() && !canManipulateRealOrders && !mockTests {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	withdrawCryptoRequest := withdraw.Request{
		Amount:      0,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: &withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("Withdraw() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("Withdraw() expecting an error when no keys are set")
	case mockTests && err != nil:
		t.Error("Mock Withdraw() error", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()

	_, err := b.WithdrawFiatFunds(&withdraw.Request{})
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdraw.Request{})
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetDepositAddress(currency.BTC, "")
	switch {
	case areTestAPIKeysSet() && err != nil:
		t.Error("GetDepositAddress() error", err)
	case !areTestAPIKeysSet() && err == nil && !mockTests:
		t.Error("GetDepositAddress() error cannot be nil")
	case mockTests && err != nil:
		t.Error("Mock GetDepositAddress() error", err)
	}
}
