package huobihadax

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey    = ""
	apiSecret = ""
)

var h HUOBIHADAX

// getDefaultConfig returns a default huobi config
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "huobihadax",
		Enabled:                 true,
		Verbose:                 true,
		Websocket:               false,
		UseSandbox:              false,
		RESTPollingDelay:        10,
		HTTPTimeout:             15000000000,
		AuthenticatedAPISupport: true,
		APIKey:                  "",
		APISecret:               "",
		ClientID:                "",
		AvailablePairs:          "BTC-USDT,BCH-USDT",
		EnabledPairs:            "BTC-USDT",
		BaseCurrencies:          "USD",
		AssetTypes:              "SPOT",
		SupportsAutoPairUpdates: false,
		ConfigCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: true,
			Delimiter: "-",
		},
		RequestCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: false,
		},
	}
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

	hadaxConfig.AuthenticatedAPISupport = true
	hadaxConfig.APIKey = apiKey
	hadaxConfig.APISecret = apiSecret

	h.Setup(hadaxConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if h.GetFee() != 0 {
		t.Errorf("test failed - Huobi GetFee() error")
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := h.GetSpotKline(KlinesRequestParams{
		Symbol: "btcusdt",
		Period: TimeIntervalHour,
		Size:   0,
	})
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetSpotKline: %s", err)
	}
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetailMerged("btcusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarketDetailMerged: %s", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := h.GetDepth("btcusdt", "step1")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetDepth: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := h.GetTrades("btcusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetTrades: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := h.GetLatestSpotPrice("btcusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi GetLatestSpotPrice: %s", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := h.GetTradeHistory("btcusdt", "50")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetTradeHistory: %s", err)
	}
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetail("btcusdt")
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

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
		t.Skip()
	}

	_, err := h.GetAccounts()
	if err != nil {
		t.Errorf("Test failed - Huobi GetAccounts: %s", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
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

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
		t.Skip()
	}

	arg := SpotNewOrderRequestParams{
		Symbol:    "btcusdt",
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

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
		t.Skip()
	}

	_, err := h.CancelOrder(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelOrder: Invalid orderID returned true")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
		t.Skip()
	}

	_, err := h.GetOrder(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelOrder: Invalid orderID returned true")
	}
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
		t.Skip()
	}

	_, err := h.GetMarginLoanOrders("btcusdt", "", "", "", "", "", "", "")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarginLoanOrders: %s", err)
	}
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
		t.Skip()
	}

	_, err := h.GetMarginAccountBalance("btcusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarginAccountBalance: %s", err)
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()

	if h.APIKey == "" || h.APISecret == "" || h.APIAuthPEMKey == "" {
		t.Skip()
	}

	_, err := h.CancelWithdraw(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelWithdraw: Invalid withdraw-ID was valid")
	}
}
