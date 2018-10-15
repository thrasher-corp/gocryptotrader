package gateio

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey    = ""
	apiSecret = ""
)

var g Gateio

func TestSetDefaults(t *testing.T) {
	g.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	gateioConfig, err := cfg.GetExchangeConfig("GateIO")
	if err != nil {
		t.Error("Test Failed - GateIO Setup() init error")
	}

	gateioConfig.AuthenticatedAPISupport = true
	gateioConfig.APIKey = apiKey
	gateioConfig.APISecret = apiSecret

	g.Setup(gateioConfig)
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols()
	if err != nil {
		t.Errorf("Test failed - Gateio TestGetSymbols: %s", err)
	}
}

func TestGetMarketInfo(t *testing.T) {
	t.Parallel()
	_, err := g.GetMarketInfo()
	if err != nil {
		t.Errorf("Test failed - Gateio GetMarketInfo: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := g.SpotNewOrder(SpotNewOrderRequestParams{
		Symbol: "btc_usdt",
		Amount: 1.1,
		Price:  10.1,
		Type:   SpotNewOrderRequestParamsTypeSell,
	})
	if err != nil {
		t.Errorf("Test failed - Gateio SpotNewOrder: %s", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := g.CancelOrder(917591554, "btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio CancelOrder: %s", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()

	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := g.GetBalances()
	if err != nil {
		t.Errorf("Test failed - Gateio GetBalances: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestSpotPrice("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := g.GetTickers()
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := g.GetOrderbook("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	_, err := g.GetSpotKline(KlinesRequestParams{
		Symbol:   "btc_usdt",
		GroupSec: TimeIntervalFiveMinutes, // 5 minutes or less
		HourSize: 1,                       // 1 hour data
	})

	if err != nil {
		t.Errorf("Test failed - Gateio GetSpotKline: %s", err)
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
	g.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()
	if apiKey != "" && apiSecret != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(20) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(20), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsTaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsTaker = true
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := g.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := g.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}
