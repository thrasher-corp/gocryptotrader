package bitfinex

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
)

// Please supply your own keys here to do better tests
const (
	testAPIKey    = ""
	testAPISecret = ""
)

func TestSetDefaults(t *testing.T) {
	t.Parallel()

	setDefaults := Bitfinex{}
	setDefaults.SetDefaults()

	if setDefaults.Name != "Bitfinex" || setDefaults.Enabled != false ||
		setDefaults.Verbose != false || setDefaults.Websocket != false ||
		setDefaults.RESTPollingDelay != 10 {
		t.Error("Test Failed - Bitfinex SetDefaults values not set correctly")
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	testConfig := config.ExchangeConfig{
		Enabled:                 true,
		AuthenticatedAPISupport: true,
		APIKey:                  "lamb",
		APISecret:               "cutlets",
		RESTPollingDelay:        time.Duration(10),
		Verbose:                 true,
		Websocket:               true,
		BaseCurrencies:          currency.DefaultCurrencies,
		AvailablePairs:          currency.MakecurrencyPairs(currency.DefaultCurrencies),
		EnabledPairs:            currency.MakecurrencyPairs(currency.DefaultCurrencies),
	}
	setup := Bitfinex{}
	setup.Setup(testConfig)

	if !setup.Enabled || !setup.AuthenticatedAPISupport || setup.APIKey != "lamb" ||
		setup.APISecret != "cutlets" || setup.RESTPollingDelay != time.Duration(10) ||
		!setup.Verbose || !setup.Websocket || len(setup.BaseCurrencies) < 1 ||
		len(setup.AvailablePairs) < 1 || len(setup.EnabledPairs) < 1 {
		t.Error("Test Failed - Bitfinex Setup values not set correctly")
	}
	testConfig.Enabled = false
	setup.Setup(testConfig)
}

func TestGetTicker(t *testing.T) {
	bitfinex := Bitfinex{}
	_, err := bitfinex.GetTicker("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetTicker init error: ", err)
	}

	_, err = bitfinex.GetTicker("wigwham", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetTicker() error")
	}
}

func TestGetStats(t *testing.T) {
	BitfinexGetStatsTest := Bitfinex{}
	_, err := BitfinexGetStatsTest.GetStats("BTCUSD")
	if err != nil {
		t.Error("BitfinexGetStatsTest init error: ", err)
	}

	_, err = BitfinexGetStatsTest.GetStats("wigwham")
	if err == nil {
		t.Error("Test Failed - GetStats() error")
	}
}

func TestGetFundingBook(t *testing.T) {
	b := Bitfinex{}
	_, err := b.GetFundingBook("USD")
	if err != nil {
		t.Error("Testing Failed - GetFundingBook() error")
	}
	_, err = b.GetFundingBook("wigwham")
	if err == nil {
		t.Error("Testing Failed - GetFundingBook() error")
	}
}

func TestGetLendbook(t *testing.T) {
	BitfinexGetLendbook := Bitfinex{}
	_, err := BitfinexGetLendbook.GetLendbook("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetLendbook init error: ", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	BitfinexGetOrderbook := Bitfinex{}
	_, err := BitfinexGetOrderbook.GetOrderbook("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetOrderbook init error: ", err)
	}
}

func TestGetTrades(t *testing.T) {
	BitfinexGetTrades := Bitfinex{}
	_, err := BitfinexGetTrades.GetTrades("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetTrades init error: ", err)
	}
}

func TestGetLends(t *testing.T) {
	BitfinexGetLends := Bitfinex{}

	_, err := BitfinexGetLends.GetLends("BTC", url.Values{})
	if err != nil {
		t.Error("BitfinexGetLends init error: ", err)
	}
}

func TestGetSymbols(t *testing.T) {
	BitfinexGetSymbols := Bitfinex{}

	symbols, err := BitfinexGetSymbols.GetSymbols()
	if err != nil {
		t.Error("BitfinexGetSymbols init error: ", err)
	}
	if reflect.TypeOf(symbols[0]).String() != "string" {
		t.Error("Bitfinex GetSymbols is not a string")
	}

	expectedCurrencies := []string{
		"rrtbtc",
		"zecusd",
		"zecbtc",
		"xmrusd",
		"xmrbtc",
		"dshusd",
		"dshbtc",
		"bccbtc",
		"bcubtc",
		"bccusd",
		"bcuusd",
		"btcusd",
		"ltcusd",
		"ltcbtc",
		"ethusd",
		"ethbtc",
		"etcbtc",
		"etcusd",
		"bfxusd",
		"bfxbtc",
		"rrtusd",
	}
	if len(expectedCurrencies) <= len(symbols) {

		for _, explicitSymbol := range expectedCurrencies {
			if common.DataContains(expectedCurrencies, explicitSymbol) {
				break
			} else {
				t.Error("BitfinexGetSymbols currency mismatch with: ", explicitSymbol)
			}
		}
	} else {
		t.Error("BitfinexGetSymbols currency mismatch, Expected Currencies < Exchange Currencies")
	}
}

func TestGetSymbolsDetails(t *testing.T) {
	BitfinexGetSymbolsDetails := Bitfinex{}
	_, err := BitfinexGetSymbolsDetails.GetSymbolsDetails()
	if err != nil {
		t.Error("BitfinexGetSymbolsDetails init error: ", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetAccountInfo()
	if err == nil {
		t.Error("Test Failed - GetAccountInfo error")
	}
}

func TestGetAccountFees(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetAccountFees()
	if err == nil {
		t.Error("Test Failed - GetAccountFees error")
	}
}

func TestGetAccountSummary(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetAccountSummary()
	if err == nil {
		t.Error("Test Failed - GetAccountSummary() error:")
	}
}

func TestNewDeposit(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.NewDeposit("blabla", "testwallet", 1)
	if err == nil {
		t.Error("Test Failed - NewDeposit() error:", err)
	}
}

func TestGetKeyPermissions(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetKeyPermissions()
	if err == nil {
		t.Error("Test Failed - GetKeyPermissions() error:")
	}
}

func TestGetMarginInfo(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetMarginInfo()
	if err == nil {
		t.Error("Test Failed - GetMarginInfo() error")
	}
}

func TestGetAccountBalance(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetAccountBalance()
	if err == nil {
		t.Error("Test Failed - GetAccountBalance() error")
	}
}

func TestWalletTransfer(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.WalletTransfer(0.01, "bla", "bla", "bla")
	if err == nil {
		t.Error("Test Failed - WalletTransfer() error")
	}
}

func TestWithdrawal(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.Withdrawal("LITECOIN", "deposit", "1000", 0.01)
	if err == nil {
		t.Error("Test Failed - Withdrawal() error")
	}
}

func TestNewOrder(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.NewOrder("BTCUSD", 1, 2, true, "market", false)
	if err == nil {
		t.Error("Test Failed - NewOrder() error")
	}
}

func TestNewOrderMulti(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret
	newOrder := []PlaceOrder{
		{
			Symbol:   "BTCUSD",
			Amount:   1,
			Price:    1,
			Exchange: "bitfinex",
			Side:     "buy",
			Type:     "market",
		},
	}

	_, err := b.NewOrderMulti(newOrder)
	if err == nil {
		t.Error("Test Failed - NewOrderMulti() error")
	}
}

func TestCancelOrder(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error")
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.CancelMultipleOrders([]int64{1337, 1336})
	if err == nil {
		t.Error("Test Failed - CancelMultipleOrders() error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.CancelAllOrders()
	if err == nil {
		t.Error("Test Failed - CancelAllOrders() error")
	}
}

func TestReplaceOrder(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.ReplaceOrder(1337, "BTCUSD", 1, 1, true, "market", false)
	if err == nil {
		t.Error("Test Failed - ReplaceOrder() error")
	}
}

func TestGetOrderStatus(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOrderStatus() error")
	}
}

func TestGetActiveOrders(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetActiveOrders()
	if err == nil {
		t.Error("Test Failed - GetActiveOrders() error")
	}
}

func TestGetActivePositions(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetActivePositions()
	if err == nil {
		t.Error("Test Failed - GetActivePositions() error")
	}
}

func TestClaimPosition(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.ClaimPosition(1337)
	if err == nil {
		t.Error("Test Failed - ClaimPosition() error")
	}
}

func TestGetBalanceHistory(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetBalanceHistory("USD", time.Time{}, time.Time{}, 1, "deposit")
	if err == nil {
		t.Error("Test Failed - GetBalanceHistory() error")
	}
}

func TestGetMovementHistory(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetMovementHistory("USD", "bitcoin", time.Time{}, time.Time{}, 1)
	if err == nil {
		t.Error("Test Failed - GetMovementHistory() error")
	}
}

func TestGetTradeHistory(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetTradeHistory("BTCUSD", time.Time{}, time.Time{}, 1, 0)
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error")
	}
}

func TestNewOffer(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.NewOffer("BTC", 1, 1, 1, "loan")
	if err == nil {
		t.Error("Test Failed - NewOffer() error")
	}
}

func TestCancelOffer(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.CancelOffer(1337)
	if err == nil {
		t.Error("Test Failed - CancelOffer() error")
	}
}

func TestGetOfferStatus(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetOfferStatus(1337)
	if err == nil {
		t.Error("Test Failed - NewOffer() error")
	}
}

func TestGetActiveCredits(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetActiveCredits()
	if err == nil {
		t.Error("Test Failed - GetActiveCredits() error", err)
	}
}

func TestGetActiveOffers(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetActiveOffers()
	if err == nil {
		t.Error("Test Failed - GetActiveOffers() error", err)
	}
}

func TestGetActiveMarginFunding(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetActiveMarginFunding()
	if err == nil {
		t.Error("Test Failed - GetActiveMarginFunding() error", err)
	}
}

func TestGetUnusedMarginFunds(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetUnusedMarginFunds()
	if err == nil {
		t.Error("Test Failed - GetUnusedMarginFunds() error", err)
	}
}

func TestGetMarginTotalTakenFunds(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.GetMarginTotalTakenFunds()
	if err == nil {
		t.Error("Test Failed - GetMarginTotalTakenFunds() error", err)
	}
}

func TestCloseMarginFunding(t *testing.T) {
	b := Bitfinex{}
	b.APIKey = testAPIKey
	b.APISecret = testAPISecret

	_, err := b.CloseMarginFunding(1337)
	if err == nil {
		t.Error("Test Failed - CloseMarginFunding() error")
	}
}
