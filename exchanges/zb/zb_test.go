package zb

import (
	"fmt"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey    = ""
	apiSecret = ""
)

var z ZB

func TestSetDefaults(t *testing.T) {
	z.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	zbConfig, err := cfg.GetExchangeConfig("ZB")
	if err != nil {
		t.Error("Test Failed - ZB Setup() init error")
	}

	zbConfig.AuthenticatedAPISupport = true
	zbConfig.APIKey = apiKey
	zbConfig.APISecret = apiSecret

	z.Setup(zbConfig)
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()

	if z.APIKey == "" || z.APISecret == "" {
		t.Skip()
	}

	arg := SpotNewOrderRequestParams{
		Symbol: "btc_usdt",
		Type:   SpotNewOrderRequestParamsTypeSell,
		Amount: 0.01,
		Price:  10246.1,
	}
	orderid, err := z.SpotNewOrder(arg)
	if err != nil {
		t.Errorf("Test failed - ZB SpotNewOrder: %s", err)
	} else {
		fmt.Println(orderid)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	if z.APIKey == "" || z.APISecret == "" {
		t.Skip()
	}

	err := z.CancelOrder(20180629145864850, "btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB CancelOrder: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := z.GetLatestSpotPrice("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := z.GetTicker("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB GetTicker: %s", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := z.GetTickers()
	if err != nil {
		t.Errorf("Test failed - ZB GetTicker: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := z.GetOrderbook("btc_usdt")
	if err != nil {
		t.Errorf("Test failed - ZB GetTicker: %s", err)
	}
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := z.GetMarkets()
	if err != nil {
		t.Errorf("Test failed - ZB GetMarkets: %s", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()

	if z.APIKey == "" || z.APISecret == "" {
		t.Skip()
	}

	_, err := z.GetAccountInfo()
	if err != nil {
		t.Errorf("Test failed - ZB GetAccountInfo: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()

	arg := KlinesRequestParams{
		Symbol: "btc_usdt",
		Type:   TimeIntervalFiveMinutes,
		Size:   10,
	}
	_, err := z.GetSpotKline(arg)
	if err != nil {
		t.Errorf("Test failed - ZB GetSpotKline: %s", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "-",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.LTC,
		SecondCurrency:      symbol.BTC,
		IsMaker:             false,
		IsTaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	z.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := z.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0015), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := z.GetFee(feeBuilder); resp != float64(2000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(2000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsTaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsTaker = true
	if resp, err := z.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := z.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0.005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.005), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := z.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}
