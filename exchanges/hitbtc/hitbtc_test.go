package hitbtc

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var h HitBTC

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	h.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	hitbtcConfig, err := cfg.GetExchangeConfig("HitBTC")
	if err != nil {
		t.Error("Test Failed - HitBTC Setup() init error")
	}

	hitbtcConfig.AuthenticatedAPISupport = true
	hitbtcConfig.APIKey = apiKey
	hitbtcConfig.APISecret = apiSecret

	h.Setup(hitbtcConfig)
}

func TestGetOrderbook(t *testing.T) {
	_, err := h.GetOrderbook("BTCUSD", 50)
	if err != nil {
		t.Error("Test faild - HitBTC GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := h.GetTrades("BTCUSD", "", "", "", "", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetTradeHistory() error", err)
	}
}

func TestGetChartCandles(t *testing.T) {
	_, err := h.GetCandles("BTCUSD", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := h.GetCurrencies()
	if err != nil {
		t.Error("Test faild - HitBTC GetCurrencies() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "-",
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
	h.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()
	if apiKey != "" && apiSecret != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsTaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsTaker = true
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if resp, err := h.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
			t.Error(err)
		}

		// CryptocurrencyWithdrawalFee Invalid currency
		feeBuilder = setFeeBuilder()
		feeBuilder.FirstCurrency = "hello"
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err == nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0.0006) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0006), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}
