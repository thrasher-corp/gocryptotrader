package bitfinex

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply your own keys here to do better tests
const (
	testAPIKey    = ""
	testAPISecret = ""
)

var b Bitfinex

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()

	if b.Name != "Bitfinex" || b.Enabled != false ||
		b.Verbose != false || b.Websocket != false ||
		b.RESTPollingDelay != 10 {
		t.Error("Test Failed - Bitfinex SetDefaults values not set correctly")
	}
}

func TestSetup(t *testing.T) {
	setup := Bitfinex{}
	setup.Name = "Bitfinex"
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bfxConfig, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		t.Error("Test Failed - Bitfinex Setup() init error")
	}
	setup.Setup(bfxConfig)

	b.SetDefaults()
	b.Setup(bfxConfig)

	if !b.Enabled || b.AuthenticatedAPISupport || b.RESTPollingDelay != time.Duration(10) ||
		b.Verbose || b.Websocket || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - Bitfinex Setup values not set correctly")
	}
}

func TestGetPlatformStatus(t *testing.T) {
	t.Parallel()

	result, err := b.GetPlatformStatus()
	if err != nil {
		t.Errorf("TestGetPlatformStatus error: %s", err)
	}

	if result != bitfinexOperativeMode && result != bitfinexMaintenanceMode {
		t.Errorf("TestGetPlatformStatus unexpected response code")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetTicker init error: ", err)
	}

	_, err = b.GetTicker("wigwham", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetTicker() error")
	}
}

func TestGetTickerV2(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickerV2("tBTCUSD")
	if err != nil {
		t.Errorf("GetTickerV2 error: %s", err)
	}

	_, err = b.GetTickerV2("fUSD")
	if err != nil {
		t.Errorf("GetTickerV2 error: %s", err)
	}
}

func TestGetTickersV2(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickersV2("tBTCUSD,fUSD")
	if err != nil {
		t.Errorf("GetTickersV2 error: %s", err)
	}
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetStats("BTCUSD")
	if err != nil {
		t.Error("BitfinexGetStatsTest init error: ", err)
	}

	_, err = b.GetStats("wigwham")
	if err == nil {
		t.Error("Test Failed - GetStats() error")
	}
}

func TestGetFundingBook(t *testing.T) {
	t.Parallel()
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
	t.Parallel()

	_, err := b.GetLendbook("BTCUSD", url.Values{})
	if err != nil {
		t.Error("Testing Failed - GetLendbook() error: ", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbook("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetOrderbook init error: ", err)
	}
}

func TestGetOrderbookV2(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderbookV2("tBTCUSD", "P0", url.Values{})
	if err != nil {
		t.Errorf("GetOrderbookV2 error: %s", err)
	}

	_, err = b.GetOrderbookV2("fUSD", "P0", url.Values{})
	if err != nil {
		t.Errorf("GetOrderbookV2 error: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetTrades("BTCUSD", url.Values{})
	if err != nil {
		t.Error("BitfinexGetTrades init error: ", err)
	}
}

func TestGetTradesv2(t *testing.T) {
	t.Parallel()

	_, err := b.GetTradesV2("tBTCUSD", 0, 0, true)
	if err != nil {
		t.Error("BitfinexGetTrades init error: ", err)
	}
}

func TestGetLends(t *testing.T) {
	t.Parallel()

	_, err := b.GetLends("BTC", url.Values{})
	if err != nil {
		t.Error("BitfinexGetLends init error: ", err)
	}
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()

	symbols, err := b.GetSymbols()
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
			if common.StringDataCompare(expectedCurrencies, explicitSymbol) {
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
	t.Parallel()

	_, err := b.GetSymbolsDetails()
	if err != nil {
		t.Error("BitfinexGetSymbolsDetails init error: ", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountInfo()
	if err == nil {
		t.Error("Test Failed - GetAccountInfo error")
	}
}

func TestGetAccountFees(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountFees()
	if err == nil {
		t.Error("Test Failed - GetAccountFees error")
	}
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountSummary()
	if err == nil {
		t.Error("Test Failed - GetAccountSummary() error:")
	}
}

func TestNewDeposit(t *testing.T) {
	t.Parallel()

	_, err := b.NewDeposit("blabla", "testwallet", 1)
	if err == nil {
		t.Error("Test Failed - NewDeposit() error:", err)
	}
}

func TestGetKeyPermissions(t *testing.T) {
	t.Parallel()

	_, err := b.GetKeyPermissions()
	if err == nil {
		t.Error("Test Failed - GetKeyPermissions() error:")
	}
}

func TestGetMarginInfo(t *testing.T) {
	t.Parallel()

	_, err := b.GetMarginInfo()
	if err == nil {
		t.Error("Test Failed - GetMarginInfo() error")
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()

	_, err := b.GetAccountBalance()
	if err == nil {
		t.Error("Test Failed - GetAccountBalance() error")
	}
}

func TestWalletTransfer(t *testing.T) {
	t.Parallel()

	_, err := b.WalletTransfer(0.01, "bla", "bla", "bla")
	if err == nil {
		t.Error("Test Failed - WalletTransfer() error")
	}
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()

	_, err := b.Withdrawal("LITECOIN", "deposit", "1000", 0.01)
	if err == nil {
		t.Error("Test Failed - Withdrawal() error")
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()

	_, err := b.NewOrder("BTCUSD", 1, 2, true, "market", false)
	if err == nil {
		t.Error("Test Failed - NewOrder() error")
	}
}

func TestNewOrderMulti(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	_, err := b.CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error")
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()

	_, err := b.CancelMultipleOrders([]int64{1337, 1336})
	if err == nil {
		t.Error("Test Failed - CancelMultipleOrders() error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()

	_, err := b.CancelAllOrders()
	if err == nil {
		t.Error("Test Failed - CancelAllOrders() error")
	}
}

func TestReplaceOrder(t *testing.T) {
	t.Parallel()

	_, err := b.ReplaceOrder(1337, "BTCUSD", 1, 1, true, "market", false)
	if err == nil {
		t.Error("Test Failed - ReplaceOrder() error")
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOrderStatus() error")
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetActiveOrders()
	if err == nil {
		t.Error("Test Failed - GetActiveOrders() error")
	}
}

func TestGetActivePositions(t *testing.T) {
	t.Parallel()

	_, err := b.GetActivePositions()
	if err == nil {
		t.Error("Test Failed - GetActivePositions() error")
	}
}

func TestClaimPosition(t *testing.T) {
	t.Parallel()

	_, err := b.ClaimPosition(1337)
	if err == nil {
		t.Error("Test Failed - ClaimPosition() error")
	}
}

func TestGetBalanceHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetBalanceHistory("USD", time.Time{}, time.Time{}, 1, "deposit")
	if err == nil {
		t.Error("Test Failed - GetBalanceHistory() error")
	}
}

func TestGetMovementHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetMovementHistory("USD", "bitcoin", time.Time{}, time.Time{}, 1)
	if err == nil {
		t.Error("Test Failed - GetMovementHistory() error")
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()

	_, err := b.GetTradeHistory("BTCUSD", time.Time{}, time.Time{}, 1, 0)
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error")
	}
}

func TestNewOffer(t *testing.T) {
	t.Parallel()

	_, err := b.NewOffer("BTC", 1, 1, 1, "loan")
	if err == nil {
		t.Error("Test Failed - NewOffer() error")
	}
}

func TestCancelOffer(t *testing.T) {
	t.Parallel()

	_, err := b.CancelOffer(1337)
	if err == nil {
		t.Error("Test Failed - CancelOffer() error")
	}
}

func TestGetOfferStatus(t *testing.T) {
	t.Parallel()

	_, err := b.GetOfferStatus(1337)
	if err == nil {
		t.Error("Test Failed - NewOffer() error")
	}
}

func TestGetActiveCredits(t *testing.T) {
	t.Parallel()

	_, err := b.GetActiveCredits()
	if err == nil {
		t.Error("Test Failed - GetActiveCredits() error", err)
	}
}

func TestGetActiveOffers(t *testing.T) {
	t.Parallel()

	_, err := b.GetActiveOffers()
	if err == nil {
		t.Error("Test Failed - GetActiveOffers() error", err)
	}
}

func TestGetActiveMarginFunding(t *testing.T) {
	t.Parallel()

	_, err := b.GetActiveMarginFunding()
	if err == nil {
		t.Error("Test Failed - GetActiveMarginFunding() error", err)
	}
}

func TestGetUnusedMarginFunds(t *testing.T) {
	t.Parallel()

	_, err := b.GetUnusedMarginFunds()
	if err == nil {
		t.Error("Test Failed - GetUnusedMarginFunds() error", err)
	}
}

func TestGetMarginTotalTakenFunds(t *testing.T) {
	t.Parallel()

	_, err := b.GetMarginTotalTakenFunds()
	if err == nil {
		t.Error("Test Failed - GetMarginTotalTakenFunds() error", err)
	}
}

func TestCloseMarginFunding(t *testing.T) {
	t.Parallel()

	_, err := b.CloseMarginFunding(1337)
	if err == nil {
		t.Error("Test Failed - CloseMarginFunding() error")
	}
}
