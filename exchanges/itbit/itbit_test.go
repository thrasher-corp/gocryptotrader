package itbit

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var i ItBit

// Please provide your own keys to do proper testing
const (
	apiKey    = ""
	apiSecret = ""
	clientID  = ""
)

func TestSetDefaults(t *testing.T) {
	i.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	itbitConfig, err := cfg.GetExchangeConfig("ITBIT")
	if err != nil {
		t.Error("Test Failed - Gemini Setup() init error")
	}

	itbitConfig.AuthenticatedAPISupport = true
	itbitConfig.APIKey = apiKey
	itbitConfig.APISecret = apiSecret
	itbitConfig.ClientID = clientID

	i.Setup(itbitConfig)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := i.GetTicker("XBTUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := i.GetOrderbook("XBTSGD")
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := i.GetTradeHistory("XBTUSD", "0")
	if err != nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestGetWallets(t *testing.T) {
	_, err := i.GetWallets(url.Values{})
	if err == nil {
		t.Error("Test Failed - GetWallets() error", err)
	}
}

func TestCreateWallet(t *testing.T) {
	_, err := i.CreateWallet("test")
	if err == nil {
		t.Error("Test Failed - CreateWallet() error", err)
	}
}

func TestGetWallet(t *testing.T) {
	_, err := i.GetWallet("1337")
	if err == nil {
		t.Error("Test Failed - GetWallet() error", err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	_, err := i.GetWalletBalance("1337", "XRT")
	if err == nil {
		t.Error("Test Failed - GetWalletBalance() error", err)
	}
}

func TestGetWalletTrades(t *testing.T) {
	_, err := i.GetWalletTrades("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetWalletTrades() error", err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	_, err := i.GetFundingHistory("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetFundingHistory() error", err)
	}
}

func TestPlaceOrder(t *testing.T) {
	_, err := i.PlaceOrder("1337", "buy", "limit", "USD", 1, 0.2, "banjo", "sauce")
	if err == nil {
		t.Error("Test Failed - PlaceOrder() error", err)
	}
}

func TestGetOrder(t *testing.T) {
	_, err := i.GetOrder("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Skip()
	err := i.CancelOrder("1337", "1337order")
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := i.GetDepositAddress("1337", "AUD")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress() error", err)
	}
}

func TestWalletTransfer(t *testing.T) {
	_, err := i.WalletTransfer("1337", "mywallet", "anotherwallet", 200, "USD")
	if err == nil {
		t.Error("Test Failed - WalletTransfer() error", err)
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
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := i.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := i.GetFee(feeBuilder); resp != float64(2500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(2500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsTaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsTaker = true
	if resp, err := i.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0025), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := i.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}
