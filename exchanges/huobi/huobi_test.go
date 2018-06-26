package huobi

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

var h HUOBI

// getDefaultConfig 获取默认配置
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "huobi",
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
	h.Setup(getDefaultConfig())
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if h.GetFee() != 0 {
		t.Errorf("test failed - Huobi GetFee() error")
	}
}

func TestGetKline(t *testing.T) {
	t.Parallel()
	_, err := h.GetKline(KlinesRequestParams{
		Symbol: "btcusdt",
		Period: TimeIntervalHour,
		Size:   0,
	})
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetKline: %s", err)
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

	_, err := h.GetAccounts()
	if err != nil {
		t.Errorf("Test failed - Huobi GetAccounts: %s", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()

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

func TestPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := h.GetAccounts()
	if err != nil {
		t.Errorf("Test failed - Huobi GetAccounts: %s", err)
	}

	/*
		userID := strconv.FormatInt(result[0].ID, 10)
		_, err = h.PlaceOrder("ethusdt", "api", userID, "buy-limit", 10.1, 100.1)
		if err != nil {
			t.Errorf("Test failed - Huobi TestPlaceOrder: %s", err)
		}
	*/
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	_, err := h.CancelOrder(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelOrder: Invalid orderID returned true")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()

	_, err := h.GetOrder(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelOrder: Invalid orderID returned true")
	}
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()

	_, err := h.GetMarginLoanOrders("btcusdt", "", "", "", "", "", "", "")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarginLoanOrders: %s", err)
	}
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()

	_, err := h.GetMarginAccountBalance("btcusdt")
	if err != nil {
		t.Errorf("Test failed - Huobi TestGetMarginAccountBalance: %s", err)
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()

	_, err := h.CancelWithdraw(1337)
	if err == nil {
		t.Error("Test failed - Huobi TestCancelWithdraw: Invalid withdraw-ID was valid")
	}
}

func TestPEMLoadAndSign(t *testing.T) {
	t.Parallel()

	pemKey := strings.NewReader(h.APIAuthPEMKey)
	pemBytes, err := ioutil.ReadAll(pemKey)
	if err != nil {
		t.Fatalf("Test Failed. TestPEMLoadAndSign Unable to ioutil.ReadAll PEM key: %s", err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		t.Fatalf("Test Failed. TestPEMLoadAndSign Block is nil")
	}

	x509Encoded := block.Bytes
	privKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		t.Fatalf("Test Failed. TestPEMLoadAndSign Unable to ParseECPrivKey: %s", err)
	}

	_, _, err = ecdsa.Sign(rand.Reader, privKey, common.GetSHA256([]byte("test")))
	if err != nil {
		t.Fatalf("Test Failed. TestPEMLoadAndSign Unable to sign: %s", err)
	}
}
