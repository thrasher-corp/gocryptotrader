package gemini

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please enter sandbox API keys & assigned roles for better testing procedures

const (
	apiKey1           = ""
	apiSecret1        = ""
	apiKeyRole1       = ""
	sessionHeartBeat1 = false

	apiKey2           = ""
	apiSecret2        = ""
	apiKeyRole2       = ""
	sessionHeartBeat2 = false
)

func TestAddSession(t *testing.T) {
	var g1 Gemini
	err := AddSession(&g1, 1, apiKey1, apiSecret1, apiKeyRole1, true, false)
	if err != nil {
		t.Error("Test failed - AddSession() error")
	}
	err = AddSession(&g1, 1, apiKey1, apiSecret1, apiKeyRole1, true, false)
	if err == nil {
		t.Error("Test failed - AddSession() error")
	}
	var g2 Gemini
	err = AddSession(&g2, 2, apiKey2, apiSecret2, apiKeyRole2, false, true)
	if err != nil {
		t.Error("Test failed - AddSession() error")
	}
}

func TestSetDefaults(t *testing.T) {
	Session[1].SetDefaults()
	Session[2].SetDefaults()
}

func TestSetup(t *testing.T) {

	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	geminiConfig, err := cfg.GetExchangeConfig("Gemini")
	if err != nil {
		t.Error("Test Failed - Gemini Setup() init error")
	}

	geminiConfig.AuthenticatedAPISupport = true

	Session[1].Setup(geminiConfig)
	Session[2].Setup(geminiConfig)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetSymbols()
	if err != nil {
		t.Error("Test Failed - GetSymbols() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTicker("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = Session[1].GetTicker("bla")
	if err == nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrderbook("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTrades("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetNotionalVolume(t *testing.T) {
	if apiKey2 != "" && apiSecret2 != "" {
		t.Parallel()
		_, err := Session[2].GetNotionalVolume()
		if err != nil {
			t.Error("Test Failed - GetNotionalVolume() error", err)
		}
	}
}

func TestGetAuction(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetAuction("btcusd")
	if err != nil {
		t.Error("Test Failed - GetAuction() error", err)
	}
}

func TestGetAuctionHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetAuctionHistory("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetAuctionHistory() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].NewOrder("btcusd", 1, 4500, "buy", "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
	_, err = Session[2].NewOrder("btcusd", 1, 4500, "buy", "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelOrders(false)
	if err == nil {
		t.Error("Test Failed - CancelOrders() error", err)
	}
	_, err = Session[2].CancelOrders(true)
	if err == nil {
		t.Error("Test Failed - CancelOrders() error", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOrderStatus() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrders()
	if err == nil {
		t.Error("Test Failed - GetOrders() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetTradeHistory("btcusd", 0)
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTradeVolume()
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetBalances()
	if err == nil {
		t.Error("Test Failed - GetBalances() error", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetDepositAddress("LOL123", "btc")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := Session[1].WithdrawCrypto("LOL123", "btc", 1)
	if err == nil {
		t.Error("Test Failed - WithdrawCrypto() error", err)
	}
}

func TestPostHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := Session[2].PostHeartbeat()
	if err == nil {
		t.Error("Test Failed - PostHeartbeat() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "_",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.BTC,
		SecondCurrency:      symbol.LTC,
		IsMaker:             false,
		IsTaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {

	var feeBuilder = setFeeBuilder()
	if apiKey1 != "" && apiSecret1 != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0.01) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.01), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(100) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(100), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsTaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsTaker = true
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0.001) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0.01) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.01), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := Session[1].GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}
