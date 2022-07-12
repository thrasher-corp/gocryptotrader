package kucoin

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passPhrase              = ""
	canManipulateRealOrders = false
)

var k Kucoin

func TestMain(m *testing.M) {
	k.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Kucoin")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.OTPSecret = passPhrase
	err = k.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Kucoin); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return k.ValidateAPICredentials(k.GetDefaultCredentials()) == nil
}

// Spot asset test cases starts from here
func TestGetSymbols(t *testing.T) {
	t.Parallel()

	_, err := k.GetSymbols(context.Background(), "")
	if err != nil {
		t.Error("GetSymbols() error", err)
	}

	_, err = k.GetSymbols(context.Background(), currency.BTC.String())
	if err != nil {
		t.Error("GetSymbols() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()

	_, err := k.GetTicker(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetAllTickers(t *testing.T) {
	t.Parallel()

	_, err := k.GetAllTickers(context.Background())
	if err != nil {
		t.Error("GetAllTickers() error", err)
	}
}

func TestGet24hrStats(t *testing.T) {
	t.Parallel()

	_, err := k.Get24hrStats(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("Get24hrStats() error", err)
	}
}

func TestGetMarketList(t *testing.T) {
	t.Parallel()

	_, err := k.GetMarketList(context.Background())
	if err != nil {
		t.Error("GetMarketList() error", err)
	}
}

func TestGetPartOrderbook20(t *testing.T) {
	t.Parallel()

	_, err := k.GetPartOrderbook20(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetPartOrderbook20() error", err)
	}
}

func TestGetPartOrderbook100(t *testing.T) {
	t.Parallel()

	_, err := k.GetPartOrderbook100(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetPartOrderbook100() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetOrderbook(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()

	_, err := k.GetTradeHistory(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetTradeHistory() error", err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()

	_, err := k.GetKlines(context.Background(), "BTC-USDT", "1week", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetKlines() error", err)
	}

	_, err = k.GetKlines(context.Background(), "BTC-USDT", "5min", time.Now().Add(-time.Hour*1), time.Now())
	if err != nil {
		t.Error("GetKlines() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()

	_, err := k.GetCurrencies(context.Background())
	if err != nil {
		t.Error("GetCurrencies() error", err)
	}
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()

	_, err := k.GetCurrency(context.Background(), "BTC", "")
	if err != nil {
		t.Error("GetCurrency() error", err)
	}

	_, err = k.GetCurrency(context.Background(), "BTC", "ETH")
	if err != nil {
		t.Error("GetCurrency() error", err)
	}
}

func TestGetFiatPrice(t *testing.T) {
	t.Parallel()

	_, err := k.GetFiatPrice(context.Background(), "", "")
	if err != nil {
		t.Error("GetFiatPrice() error", err)
	}

	_, err = k.GetFiatPrice(context.Background(), "EUR", "ETH,BTC")
	if err != nil {
		t.Error("GetFiatPrice() error", err)
	}
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()

	_, err := k.GetMarkPrice(context.Background(), "USDT-BTC")
	if err != nil {
		t.Error("GetMarkPrice() error", err)
	}
}

func TestGetMarginConfiguration(t *testing.T) {
	t.Parallel()

	_, err := k.GetMarginConfiguration(context.Background())
	if err != nil {
		t.Error("GetMarginConfiguration() error", err)
	}
}

func TestGetMarginAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetMarginAccount(context.Background())
	if err != nil {
		t.Error("GetMarginAccount() error", err)
	}
}

func TestGetMarginRiskLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetMarginRiskLimit(context.Background(), "cross")
	if err != nil {
		t.Error("GetMarginRiskLimit() error", err)
	}

	_, err = k.GetMarginRiskLimit(context.Background(), "isolated")
	if err != nil {
		t.Error("GetMarginRiskLimit() error", err)
	}
}

func TestPostBorrowOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := k.PostBorrowOrder(context.Background(), "USDT", "FOK", "", 10, 0)
	if err != nil {
		t.Error("PostBorrowOrder() error", err)
	}

	_, err = k.PostBorrowOrder(context.Background(), "USDT", "IOC", "7,14,28", 10, 10)
	if err != nil {
		t.Error("PostBorrowOrder() error", err)
	}
}

func TestGetBorrowOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetBorrowOrder(context.Background(), "orderID")
	if err != nil && err.Error() != "Not Found" {
		t.Error("GetBorrowOrder() error", err)
	}
}

func TestGetOutstandingRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetOutstandingRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetOutstandingRecord() error", err)
	}
}

func TestGetRepaidRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetRepaidRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetRepaidRecord() error", err)
	}
}

func TestOneClickRepayment(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := k.OneClickRepayment(context.Background(), "BTC", "RECENTLY_EXPIRE_FIRST", 2.5)
	if err != nil && err.Error() != "Balance insufficient" {
		t.Error("OneClickRepayment() error", err)
	}
}

func TestSingleOrderRepayment(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := k.SingleOrderRepayment(context.Background(), "BTC", "fa3e34c980062c10dad74016", 2.5)
	if err != nil && err.Error() != "Balance insufficient" {
		t.Error("SingleOrderRepayment() error", err)
	}
}

func TestPostLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := k.PostLendOrder(context.Background(), "BTC", 0.001, 5, 7)
	if err != nil && err.Error() != "Balance insufficient" {
		t.Error("PostLendOrder() error", err)
	}
}

func TestCancelLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := k.CancelLendOrder(context.Background(), "OrderID")
	if err != nil && err.Error() != "order not exist" {
		t.Error("CancelLendOrder() error", err)
	}
}

func TestSetAutoLend(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := k.SetAutoLend(context.Background(), "BTC", 0.002, 0.005, 7, true)
	if err != nil {
		t.Error("SetAutoLend() error", err)
	}
}

func TestGetActiveOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetActiveOrder(context.Background(), "")
	if err != nil {
		t.Error("GetActiveOrder() error", err)
	}

	_, err = k.GetActiveOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetActiveOrder() error", err)
	}
}

func TestGetLendHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetLendHistory(context.Background(), "")
	if err != nil {
		t.Error("GetLendHistory() error", err)
	}

	_, err = k.GetLendHistory(context.Background(), "BTC")
	if err != nil {
		t.Error("GetLendHistory() error", err)
	}
}

func TestGetUnsettleLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetUnsettleLendOrder(context.Background(), "")
	if err != nil {
		t.Error("GetUnsettleLendOrder() error", err)
	}

	_, err = k.GetUnsettleLendOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetUnsettleLendOrder() error", err)
	}
}

func TestGetSettleLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetSettleLendOrder(context.Background(), "")
	if err != nil {
		t.Error("GetSettleLendOrder() error", err)
	}

	_, err = k.GetSettleLendOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetSettleLendOrder() error", err)
	}
}

func TestGetAccountLendRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetAccountLendRecord(context.Background(), "")
	if err != nil {
		t.Error("GetAccountLendRecord() error", err)
	}

	_, err = k.GetAccountLendRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetAccountLendRecord() error", err)
	}
}

func TestGetLendingMarketData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetLendingMarketData(context.Background(), "BTC", 0)
	if err != nil {
		t.Error("GetLendingMarketData() error", err)
	}

	_, err = k.GetLendingMarketData(context.Background(), "BTC", 7)
	if err != nil {
		t.Error("GetLendingMarketData() error", err)
	}
}

func TestGetMarginTradeData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}

	_, err := k.GetMarginTradeData(context.Background(), "BTC")
	if err != nil {
		t.Error("GetMarginTradeData() error", err)
	}
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()

	_, err := k.GetCurrentServerTime(context.Background())
	if err != nil {
		t.Error("GetCurrentServerTime() error", err)
	}
}

func TestGetServiceStatus(t *testing.T) {
	t.Parallel()

	_, _, err := k.GetServiceStatus(context.Background())
	if err != nil {
		t.Error("GetServiceStatus() error", err)
	}
}
