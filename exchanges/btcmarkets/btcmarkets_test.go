package btcmarkets

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var b BTCMarkets

// Please supply your own keys here to do better tests
const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("BTC Markets")
	if err != nil {
		t.Error("Test Failed - BTC Markets Setup() init error")
	}

	if apiKey != "" && apiSecret != "" {
		bConfig.APIKey = apiKey
		bConfig.APISecret = apiSecret
		bConfig.AuthenticatedAPISupport = true
	}

	b.Setup(bConfig)
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := bm.GetMarkets()
	if err != nil {
		t.Error("Test failed - GetMarkets() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("BTC", "AUD")
	if err != nil {
		t.Error("Test failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook("BTC", "AUD")
	if err != nil {
		t.Error("Test failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades("BTC", "AUD", nil)
	if err != nil {
		t.Error("Test failed - GetTrades() error", err)
	}

	val := url.Values{}
	val.Set("since", "0")
	_, err = b.GetTrades("BTC", "AUD", val)
	if err != nil {
		t.Error("Test failed - GetTrades() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.NewOrder("AUD", "BTC", 0, 0, "Bid", "limit", "testTest")
	if err == nil {
		t.Error("Test failed - NewOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelOrder([]int64{1337})
	if err == nil {
		t.Error("Test failed - CancelOrder() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrders("AUD", "BTC", 10, 0, false)
	if err == nil {
		t.Error("Test failed - GetOrders() error", err)
	}
	_, err = b.GetOrders("AUD", "BTC", 10, 0, true)
	if err == nil {
		t.Error("Test failed - GetOrders() error", err)
	}
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderDetail([]int64{1337})
	if err == nil {
		t.Error("Test failed - GetOrderDetail() error", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := b.GetAccountBalance()
	if err == nil {
		t.Error("Test failed - GetAccountBalance() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawCrypto(0, "BTC", "LOLOLOL")
	if err == nil {
		t.Error("Test failed - WithdrawCrypto() error", err)
	}
}

func TestWithdrawAUD(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawAUD("BLA", "1337", "blawest", "1336", 10000000)
	if err == nil {
		t.Error("Test failed - WithdrawAUD() error", err)
	}
}

func TestGetExchangeAccountInfo(t *testing.T) {
	_, err := b.GetExchangeAccountInfo()
	if err == nil {
		t.Error("Test failed - GetExchangeAccountInfo() error", err)
	}
}

func TestGetExchangeFundTransferHistory(t *testing.T) {
	_, err := b.GetExchangeFundTransferHistory()
	if err == nil {
		t.Error("Test failed - GetExchangeAccountInfo() error", err)
	}
}

func TestSubmitExchangeOrder(t *testing.T) {
	p := pair.NewCurrencyPair("LTC", "AUD")
	_, err := b.SubmitExchangeOrder(p, exchange.OrderSideSell(), exchange.OrderTypeMarket(), 0, 0.0, "testID001")
	if err == nil {
		t.Error("Test failed - SubmitExchangeOrder() error", err)
	}
}

func TestModifyExchangeOrder(t *testing.T) {
	_, err := b.ModifyExchangeOrder(1337, exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyExchangeOrder() error", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	err := b.CancelExchangeOrder(1337)
	if err == nil {
		t.Error("Test failed - CancelExchangeOrder() error", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	err := b.CancelAllExchangeOrders()
	if err == nil {
		t.Error("Test failed - CancelAllExchangeOrders() error", err)
	}
}

func TestGetExchangeOrderInfo(t *testing.T) {
	_, err := b.GetExchangeOrderInfo(1337)
	if err == nil {
		t.Error("Test failed - GetExchangeOrderInfo() error", err)
	}
}

func TestWithdrawCryptoExchangeFunds(t *testing.T) {
	_, err := b.WithdrawCryptoExchangeFunds("someaddress", "ltc", 0)
	if err == nil {
		t.Error("Test failed - WithdrawExchangeFunds() error", err)
	}
}

func TestWithdrawFiatExchangeFunds(t *testing.T) {
	_, err := b.WithdrawFiatExchangeFunds("AUD", 0)
	if err == nil {
		t.Error("Test failed - WithdrawFiatExchangeFunds() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:         1,
		Delimiter:      "",
		FeeType:        exchange.CryptocurrencyTradeFee,
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
		IsMaker:        false,
		IsTaker:        false,
		PurchasePrice:  1,
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	b.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()

	if apiKey != "" || apiSecret != "" {
		// CryptocurrencyTradeFee Basic
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Error(err)
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsTaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsTaker = true
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.02), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.01), resp)
			t.Error(err)
		}

		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
			t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
			t.Error(err)
		}
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.CurrencyItem = symbol.AUD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.AUD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}
