package huobihadax

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var h HUOBIHADAX

// getDefaultConfig returns a default hadax config
func getDefaultConfig() config.ExchangeConfig {
	exchCfg := config.ExchangeConfig{
		Name:    "Huobi",
		Enabled: true,
		Verbose: true,

		Features: &config.FeaturesConfig{
			Supports: config.FeaturesSupportedConfig{
				REST:      true,
				Websocket: false,

				RESTCapabilities: config.ProtocolFeaturesConfig{
					AutoPairUpdates: false,
				},
			},
			Enabled: config.FeaturesEnabledConfig{
				AutoPairUpdates: false,
				Websocket:       false,
			},
		},

		API: config.APIConfig{
			AuthenticatedSupport: false,
		},

		HTTPTimeout: 15000000000,
		CurrencyPairs: &currency.PairsManager{
			AssetTypes: assets.AssetTypes{
				assets.AssetTypeSpot,
			},
			UseGlobalFormat: true,
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
			RequestFormat: &currency.PairFormat{
				Uppercase: false,
			},
		},

		BaseCurrencies: currency.NewCurrenciesFromStringArray([]string{"USD"}),
	}

	exchCfg.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTC-USDT", "BCH-USDT"}),
		false)
	exchCfg.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTC-USDT"}),
		true)

	return exchCfg
}

func TestSetDefaults(t *testing.T) {
	h.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	hadaxConfig, err := cfg.GetExchangeConfig("HuobiHadax")
	if err != nil {
		t.Error("Test Failed - HuobiHadax Setup() init error")
	}
	hadaxConfig.API.AuthenticatedSupport = true
	hadaxConfig.API.Credentials.Key = apiKey
	hadaxConfig.API.Credentials.Secret = apiSecret

	h.Setup(hadaxConfig)
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := h.GetSpotKline(KlinesRequestParams{
		Symbol: "hptusdt",
		Period: TimeIntervalHour,
		Size:   0,
	})
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetSpotKline: %s", err)
	}
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetailMerged("hptusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarketDetailMerged: %s", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := h.GetDepth("hptusdt", "step1")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetDepth: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := h.GetTrades("hptusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetTrades: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := h.GetLatestSpotPrice("hptusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi GetLatestSpotPrice: %s", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := h.GetTradeHistory("hptusdt", "50")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetTradeHistory: %s", err)
	}
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetail("hptusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetTradeHistory: %s", err)
	}
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := h.GetSymbols()
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetSymbols: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := h.GetCurrencies()
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetCurrencies: %s", err)
	}
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	_, err := h.GetTimestamp()
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetTimestamp: %s", err)
	}
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.GetAccounts()
	if err != nil {
		t.Errorf("Test failed - Huobi GetAccounts: %s", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	result, err := h.GetAccounts()
	if err != nil {
		t.Errorf("Test failed - Huobi GetAccounts: %s", err)
	}

	userID := strconv.FormatInt(result[0].ID, 10)
	_, err = h.GetAccountBalance(userID)
	if err != nil {
		t.Errorf("Test failed - Huobi GetAccountBalance: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	arg := SpotNewOrderRequestParams{
		Symbol:    "hptusdt",
		AccountID: 000000,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	newOrderID, err := h.SpotNewOrder(arg)
	if err != nil {
		t.Errorf("Test failed - Huobi SpotNewOrder: %s", err)
	} else {
		fmt.Println(newOrderID)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.CancelExistingOrder(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelExistingOrder: Invalid orderID returned true")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.GetOrder(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelOrder: Invalid orderID returned true")
	}
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.GetMarginLoanOrders("hptusdt", "", "", "", "", "", "", "")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarginLoanOrders: %s", err)
	}
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.GetMarginAccountBalance("hptusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarginAccountBalance: %s", err)
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.CancelWithdraw(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelWithdraw: Invalid withdraw-ID was valid")
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
	h.GetFeeByType(feeBuilder)
	if apiKey == "" || apiSecret == "" {
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
	if resp, err := h.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := h.GetFee(feeBuilder); resp != float64(2000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := h.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	h.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText

	withdrawPermissions := h.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	h.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType:  exchange.AnyOrderType,
		Currencies: []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
	}

	_, err := h.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	h.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType:  exchange.AnyOrderType,
		Currencies: []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
	}

	_, err := h.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return h.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	h.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.BTC,
		Quote:     currency.USDT,
	}

	accounts, err := h.GetAccounts()
	if err != nil {
		t.Fatalf("Failed to get accounts. Err: %s", err)
	}

	response, err := h.SubmitOrder(p, exchange.BuyOrderSide, exchange.LimitOrderType, 1, 10, strconv.FormatInt(accounts[0].ID, 10))
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	h.SetDefaults()
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

	err := h.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	h.SetDefaults()
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

	resp, err := h.CancelAllOrders(orderCancellation)

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

func TestGetAccountInfo(t *testing.T) {
	if apiKey == "" || apiSecret == "" {
		_, err := h.GetAccountInfo()
		if err == nil {
			t.Error("Test Failed - GetAccountInfo() error")
		}
	} else {
		_, err := h.GetAccountInfo()
		if err != nil {
			t.Error("Test Failed - GetAccountInfo() error", err)
		}
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := h.ModifyOrder(&exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	h.SetDefaults()
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

	_, err := h.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	h.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := h.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	h.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := h.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := h.GetDepositAddress(currency.BTC, "")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress() error cannot be nil")
	}
}
