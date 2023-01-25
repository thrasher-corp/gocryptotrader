package kucoin

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passPhrase              = ""
	canManipulateRealOrders = false

	cantManipulateRealOrdersOrKeysNotSet = "either API keys are missing or canManipulateRealOrders not enabled"
	credentialsNotSet                    = "credentials not set" // "skipping test message: api keys not set"
)

var ku Kucoin

func TestMain(m *testing.M) {
	ku.SetDefaults()
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
	exchCfg.API.Credentials.ClientID = passPhrase

	ku.SetDefaults()
	ku.Websocket = sharedtestvalues.NewTestWebsocket()
	ku.Websocket.Orderbook = buffer.Orderbook{}
	err = ku.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	request.MaxRequestJobs = 100
	ku.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	ku.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return ku.ValidateAPICredentials(ku.GetDefaultCredentials()) == nil
}

// Spot asset test cases starts from here
func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := ku.GetSymbols(context.Background(), "")
	if err != nil {
		t.Error("GetSymbols() error", err)
	}
	_, err = ku.GetSymbols(context.Background(), currency.BTC.String())
	if err != nil {
		t.Error("GetSymbols() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTicker(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetAllTickers(t *testing.T) {
	t.Parallel()
	_, err := ku.GetAllTickers(context.Background())
	if err != nil {
		t.Error("GetAllTickers() error", err)
	}
}

func TestGet24hrStats(t *testing.T) {
	t.Parallel()
	_, err := ku.Get24hrStats(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("Get24hrStats() error", err)
	}
}

func TestGetMarketList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarketList(context.Background())
	if err != nil {
		t.Error("GetMarketList() error", err)
	}
}

func TestGetPartOrderbook20(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPartOrderbook20(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetPartOrderbook20() error", err)
	}
}

func TestGetPartOrderbook100(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPartOrderbook100(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetPartOrderbook100() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetOrderbook(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTradeHistory(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetTradeHistory() error", err)
	}
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := ku.GetKlines(context.Background(), "BTC-USDT", "1week", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetKlines() error", err)
	}
	_, err = ku.GetKlines(context.Background(), "BTC-USDT", "5min", time.Now().Add(-time.Hour*1), time.Now())
	if err != nil {
		t.Error("GetKlines() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrencies(context.Background())
	if err != nil {
		t.Error("GetCurrencies() error", err)
	}
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrencyDetail(context.Background(), "BTC", "")
	if err != nil {
		t.Error("GetCurrency() error", err)
	}

	_, err = ku.GetCurrencyDetail(context.Background(), "BTC", "ETH")
	if err != nil {
		t.Error("GetCurrency() error", err)
	}
}

func TestGetFiatPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFiatPrice(context.Background(), "", "")
	if err != nil {
		t.Error("GetFiatPrice() error", err)
	}

	_, err = ku.GetFiatPrice(context.Background(), "EUR", "ETH,BTC")
	if err != nil {
		t.Error("GetFiatPrice() error", err)
	}
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarkPrice(context.Background(), "USDT-BTC")
	if err != nil {
		t.Error("GetMarkPrice() error", err)
	}
}

func TestGetMarginConfiguration(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarginConfiguration(context.Background())
	if err != nil {
		t.Error("GetMarginConfiguration() error", err)
	}
}

func TestGetMarginAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetMarginAccount(context.Background())
	if err != nil {
		t.Error("GetMarginAccount() error", err)
	}
}

func TestGetMarginRiskLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetMarginRiskLimit(context.Background(), "cross")
	if err != nil {
		t.Error("GetMarginRiskLimit() error", err)
	}

	_, err = ku.GetMarginRiskLimit(context.Background(), "isolated")
	if err != nil {
		t.Error("GetMarginRiskLimit() error", err)
	}
}

func TestPostBorrowOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.PostBorrowOrder(context.Background(), "USDT", "FOK", "", 10, 0)
	if err != nil {
		t.Error("PostBorrowOrder() error", err)
	}

	_, err = ku.PostBorrowOrder(context.Background(), "USDT", "IOC", "7,14,28", 10, 10)
	if err != nil {
		t.Error("PostBorrowOrder() error", err)
	}
}

func TestGetBorrowOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetBorrowOrder(context.Background(), "orderID")
	if err != nil && !strings.Contains(err.Error(), "Not Found") {
		t.Error("GetBorrowOrder() error", err)
	}
}

func TestGetOutstandingRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetOutstandingRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetOutstandingRecord() error", err)
	}
}

func TestGetRepaidRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	ku.Verbose = true
	_, err := ku.GetRepaidRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetRepaidRecord() error", err)
	}
}

func TestOneClickRepayment(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.OneClickRepayment(context.Background(), "BTC", "RECENTLY_EXPIRE_FIRST", 2.5)
	if err != nil && err.Error() != "Balance insufficient" {
		t.Error("OneClickRepayment() error", err)
	}
}

func TestSingleOrderRepayment(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.SingleOrderRepayment(context.Background(), "BTC", "fa3e34c980062c10dad74016", 2.5)
	if err != nil && err.Error() != "Balance insufficient" {
		t.Error("SingleOrderRepayment() error", err)
	}
}

func TestPostLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.PostLendOrder(context.Background(), "BTC", 0.001, 5, 7)
	if err != nil && err.Error() != "Balance insufficient" {
		t.Error("PostLendOrder() error", err)
	}
}

func TestCancelLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.CancelLendOrder(context.Background(), "OrderID")
	if err != nil && err.Error() != "order not exist" {
		t.Error("CancelLendOrder() error", err)
	}
}

func TestSetAutoLend(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.SetAutoLend(context.Background(), "BTC", 0.002, 0.005, 7, true)
	if err != nil {
		t.Error("SetAutoLend() error", err)
	}
}

func TestGetActiveOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetActiveOrder(context.Background(), "")
	if err != nil {
		t.Error("GetActiveOrder() error", err)
	}

	_, err = ku.GetActiveOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetActiveOrder() error", err)
	}
}

func TestGetLendHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetLendHistory(context.Background(), "")
	if err != nil {
		t.Error("GetLendHistory() error", err)
	}

	_, err = ku.GetLendHistory(context.Background(), "BTC")
	if err != nil {
		t.Error("GetLendHistory() error", err)
	}
}

func TestGetUnsettleLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetUnsettleLendOrder(context.Background(), "")
	if err != nil {
		t.Error("GetUnsettleLendOrder() error", err)
	}

	_, err = ku.GetUnsettleLendOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetUnsettleLendOrder() error", err)
	}
}

func TestGetSettleLendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetSettleLendOrder(context.Background(), "")
	if err != nil {
		t.Error("GetSettleLendOrder() error", err)
	}

	_, err = ku.GetSettleLendOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetSettleLendOrder() error", err)
	}
}

func TestGetAccountLendRecord(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetAccountLendRecord(context.Background(), "")
	if err != nil {
		t.Error("GetAccountLendRecord() error", err)
	}

	_, err = ku.GetAccountLendRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetAccountLendRecord() error", err)
	}
}

func TestGetLendingMarketData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetLendingMarketData(context.Background(), "BTC", 0)
	if err != nil {
		t.Error("GetLendingMarketData() error", err)
	}

	_, err = ku.GetLendingMarketData(context.Background(), "BTC", 7)
	if err != nil {
		t.Error("GetLendingMarketData() error", err)
	}
}

func TestGetMarginTradeData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetMarginTradeData(context.Background(), "BTC")
	if err != nil {
		t.Error("GetMarginTradeData() error", err)
	}
}

func TestGetIsolatedMarginPairConfig(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetIsolatedMarginPairConfig(context.Background())
	if err != nil {
		t.Error("GetIsolatedMarginPairConfig() error", err)
	}
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetIsolatedMarginAccountInfo(context.Background(), "")
	if err != nil {
		t.Error("GetIsolatedMarginAccountInfo() error", err)
	}
	_, err = ku.GetIsolatedMarginAccountInfo(context.Background(), "USDT")
	if err != nil {
		t.Error("GetIsolatedMarginAccountInfo() error", err)
	}
}

func TestGetSingleIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetSingleIsolatedMarginAccountInfo(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetSingleIsolatedMarginAccountInfo() error", err)
	}
}

func TestInitiateIsolateMarginBorrowing(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.InitiateIsolateMarginBorrowing(context.Background(), "BTC-USDT", "USDT", "FOK", "", 10, 0)
	if err != nil {
		t.Error("InitiateIsolateMarginBorrowing() error", err)
	}
}

func TestGetIsolatedOutstandingRepaymentRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetIsolatedOutstandingRepaymentRecords(context.Background(), "", "", 0, 0)
	if err != nil {
		t.Error("GetIsolatedOutstandingRepaymentRecords() error", err)
	}

	_, err = ku.GetIsolatedOutstandingRepaymentRecords(context.Background(), "BTC-USDT", "USDT", 0, 0)
	if err != nil {
		t.Error("GetIsolatedOutstandingRepaymentRecords() error", err)
	}
}

func TestGetIsolatedMarginRepaymentRecords(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetIsolatedMarginRepaymentRecords(context.Background(), "", "", 0, 0)
	if err != nil {
		t.Error("GetIsolatedMarginRepaymentRecords() error", err)
	}

	_, err = ku.GetIsolatedMarginRepaymentRecords(context.Background(), "BTC-USDT", "USDT", 0, 0)
	if err != nil {
		t.Error("GetIsolatedMarginRepaymentRecords() error", err)
	}
}

func TestInitiateIsolatedMarginQuickRepayment(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.InitiateIsolatedMarginQuickRepayment(context.Background(), "BTC-USDT", "USDT", "RECENTLY_EXPIRE_FIRST", 10)
	if err != nil {
		t.Error("InitiateIsolatedMarginQuickRepayment() error", err)
	}
}

func TestInitiateIsolatedMarginSingleRepayment(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.InitiateIsolatedMarginSingleRepayment(context.Background(), "BTC-USDT", "USDT", "628c570f7818320001d52b69", 10)
	if err != nil {
		t.Error("InitiateIsolatedMarginSingleRepayment() error", err)
	}
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()

	_, err := ku.GetCurrentServerTime(context.Background())
	if err != nil {
		t.Error("GetCurrentServerTime() error", err)
	}
}

func TestGetServiceStatus(t *testing.T) {
	t.Parallel()
	_, err := ku.GetServiceStatus(context.Background())
	if err != nil {
		t.Error("GetServiceStatus() error", err)
	}
}

func TestPostOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	// default order type is limit
	_, err := ku.PostOrder(context.Background(), "5bd6e9286d99522a52e458de", "buy", "BTC-USDT", "limit", "", "", "", 0.1, 0, 0, 0, 0, true, false, false)
	if err != nil && err.Error() != "Balance insufficient!" {
		t.Error("PostOrder() error", err)
	}

	// market order
	_, err = ku.PostOrder(context.Background(), "5bd6e9286d99522a52e458de", "buy", "BTC-USDT", "market", "remark", "", "", 0.1, 0, 0, 0, 0, true, false, false)
	if err != nil && err.Error() != "Balance insufficient!" {
		t.Error("PostOrder() error", err)
	}
}

func TestPostMarginOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	// default order type is limit and margin mode is cross
	_, err := ku.PostMarginOrder(context.Background(), "5bd6e9286d99522a52e458de", "buy", "BTC-USDT", "", "", "", "", "10000", 1000, 0.1, 0, 0, 0, true, false, false, false)
	if err != nil && err.Error() != "Balance insufficient!" {
		t.Error("PostMarginOrder() error", err)
	}

	// market isolated order
	_, err = ku.PostMarginOrder(context.Background(), "5bd6e9286d99522a52e458de", "buy", "BTC-USDT", "market", "remark", "", "isolated", "", 1000, 0.1, 0, 0, 5, true, false, false, true)
	if err != nil && err.Error() != "Balance insufficient!" {
		t.Error("PostMarginOrder() error", err)
	}
}

func TestPostBulkOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	req := []OrderRequest{
		{
			ClientOID: "3d07008668054da6b3cb12e432c2b13a",
			Side:      "buy",
			Type:      "limit",
			Price:     1000,
			Size:      0.01,
		},
		{
			ClientOID: "37245dbe6e134b5c97732bfb36cd4a9d",
			Side:      "buy",
			Type:      "limit",
			Price:     1000,
			Size:      0.01,
		},
	}

	_, err := ku.PostBulkOrder(context.Background(), "BTC-USDT", req)
	if err != nil {
		t.Error("PostBulkOrder() error", err)
	}
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelSingleOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil && err.Error() != "order_not_exist_or_not_allow_to_cancel" {
		t.Error("CancelSingleOrder() error", err)
	}
}

func TestCancelOrderByClientOID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelOrderByClientOID(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil && err.Error() != "order_not_exist_or_not_allow_to_cancel" {
		t.Error("CancelOrderByClientOID() error", err)
	}
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelAllOpenOrders(context.Background(), "", "")
	if err != nil {
		t.Error("CancelAllOpenOrders() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetOrders(context.Background(), "", "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetOrders() error", err)
	}
}

// TODO: ambiguity in doc. and API response
func TestGetRecentOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetRecentOrders(context.Background())
	if err != nil {
		t.Error("GetRecentOrders() error", err)
	}
}

// TODO: not sure of response after looking at doc.
func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetOrderByID(context.Background(), "5c35c02703aa673ceec2a168")
	if err != nil && !strings.Contains(err.Error(), "order not exist.") {
		t.Error("GetOrderByID() error", err)
	}
}

func TestGetOrderByClientOID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetOrderByClientSuppliedOrderID(context.Background(), "6d539dc614db312")
	if err != nil && !strings.Contains(err.Error(), "400100") {
		t.Error("GetOrderByClientOID() error", err)
	}
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetFills(context.Background(), "", "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFills() error", err)
	}

	_, err = ku.GetFills(context.Background(), "5c35c02703aa673ceec2a168", "BTC-USDT", "buy", "limit", "TRADE", time.Now().Add(-time.Hour*12), time.Now())
	if err != nil {
		t.Error("GetFills() error", err)
	}
}

func TestGetRecentFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetRecentFills(context.Background())
	if err != nil {
		t.Error("GetRecentFills() error", err)
	}
}

func TestPostStopOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := ku.PostStopOrder(context.Background(), "5bd6e9286d99522a52e458de", "buy", "BTC-USDT", "", "", "entry", "10000", "11000", "", 0.1, 1, 0, 0, 0, 0, true, false, false)
	if err != nil {
		t.Error("PostStopOrder() error", err)
	}
}

func TestCancelStopOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelStopOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil && err.Error() != "order_not_exist_or_not_allow_to_cancel" {
		t.Error("CancelStopOrder() error", err)
	}
}

func TestCancelAllStopOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelAllStopOrder(context.Background(), "", "", "")
	if err != nil {
		t.Error("CancelAllStopOrder() error", err)
	}
}

func TestGetStopOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetStopOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("GetStopOrder() error", err)
	}
}

func TestGetAllStopOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetAllStopOrder(context.Background(), "", "", "", "", "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error("GetAllStopOrder() error", err)
	}
}

func TestGetStopOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetStopOrderByClientID(context.Background(), "", "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("GetStopOrderByClientID() error", err)
	}
}

func TestCancelStopOrderByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.CancelStopOrderByClientID(context.Background(), "", "5bd6e9286d99522a52e458de")
	if err != nil && err.Error() != "order_not_exist_or_not_allow_to_cancel" {
		t.Error("CancelStopOrderByClientID() error", err)
	}
}

func TestGetAllAccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetAllAccounts(context.Background(), "", "")
	if err != nil {
		t.Error("GetAllAccounts() error", err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetAccount(context.Background(), "62fcd1969474ea0001fd20e4")
	if err != nil && err.Error() != "account not exist" {
		t.Error("GetAccount() error", err)
	}
}

func TestGetAccountLedgers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetAccountLedgers(context.Background(), "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetAccountLedgers() error", err)
	}
}

func TestGetAccountSummaryInformation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetAccountSummaryInformation(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetSubAccountBalance(context.Background(), "62fcd1969474ea0001fd20e4", false)
	if err != nil && err.Error() != "User not found." {
		t.Error("GetSubAccountBalance() error", err)
	}
}

func TestGetAggregatedSubAccountBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetAggregatedSubAccountBalance(context.Background())
	if err != nil {
		t.Error("GetAggregatedSubAccountBalance() error", err)
	}
}

func TestGetPaginatedSubAccountInformation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetPaginatedSubAccountInformation(context.Background(), 0, 10)
	if err != nil {
		t.Error("GetPaginatedSubAccountInformation() error", err)
	}
}

func TestGetTransferableBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetTransferableBalance(context.Background(), "BTC", "MAIN", "")
	if err != nil {
		t.Error("GetTransferableBalance() error", err)
	}
}

func TestTransferMainToSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.TransferMainToSubAccount(context.Background(), "62fcd1969474ea0001fd20e4", "BTC", "1", "OUT", "", "", "5caefba7d9575a0688f83c45")
	if err != nil {
		t.Error("TransferMainToSubAccount() error", err)
	}
}

func TestMakeInnerTransfer(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.MakeInnerTransfer(context.Background(), "62fcd1969474ea0001fd20e4", "BTC", "trade", "main", "1", "", "")
	if err != nil {
		t.Error("MakeInnerTransfer() error", err)
	}
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CreateDepositAddress(context.Background(), "BTC", "")
	if err != nil {
		t.Error("CreateDepositAddress() error", err)
	}

	_, err = ku.CreateDepositAddress(context.Background(), "USDT", "TRC20")
	if err != nil {
		t.Error("CreateDepositAddress() error", err)
	}
}

func TestGetDepositAddressV2(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetDepositAddressesV2(context.Background(), "BTC")
	if err != nil {
		t.Error("GetDepositAddressV2() error", err)
	}
}

func TestGetDepositList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetDepositList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetDepositList() error", err)
	}
}

func TestGetHistoricalDepositList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetHistoricalDepositList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetHistoricalDepositList() error", err)
	}
}

func TestGetWithdrawalList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetWithdrawalList() error", err)
	}
}

func TestGetHistoricalWithdrawalList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetHistoricalWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error("GetHistoricalWithdrawalList() error", err)
	}
}

func TestGetWithdrawalQuotas(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetWithdrawalQuotas(context.Background(), "BTC", "")
	if err != nil {
		t.Error("GetWithdrawalQuotas() error", err)
	}
}

func TestApplyWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.ApplyWithdrawal(context.Background(), "ETH", "0x597873884BC3a6C10cB6Eb7C69172028Fa85B25A", "", "", "", "", false, 1)
	if err != nil {
		t.Error("ApplyWithdrawal() error", err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.CancelWithdrawal(context.Background(), "5bffb63303aa675e8bbe18f9")
	if err != nil {
		t.Error("CancelWithdrawal() error", err)
	}
}

func TestGetBasicFee(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetBasicFee(context.Background(), "1")
	if err != nil {
		t.Error("GetBasicFee() error", err)
	}
}

func TestGetTradingFee(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetTradingFee(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetTradingFee() error", err)
	}
}

// futures
func TestGetFuturesOpenContracts(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesOpenContracts(context.Background())
	if err != nil {
		t.Error("GetFuturesOpenContracts() error", err)
	}
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesContract(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesContract() error", err)
	}
}

func TestGetFuturesRealTimeTicker(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesRealTimeTicker(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesRealTimeTicker() error", err)
	}
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	pairs, err := ku.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Skip(err)
	}
	_, err = ku.GetFuturesOrderbook(context.Background(), pairs[0].String())
	if err != nil {
		t.Error("GetFuturesOrderbook() error", err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	if err := ku.UpdateTradablePairs(context.Background(), false); err != nil {
		t.Error(err)
	}
}

func TestGetFuturesPartOrderbook20(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPartOrderbook20(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesPartOrderbook20() error", err)
	}
}

func TestGetFuturesPartOrderbook100(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPartOrderbook100(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesPartOrderbook100() error", err)
	}
}

func TestGetFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesTradeHistory(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesTradeHistory() error", err)
	}
}

func TestGetFuturesInterestRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesInterestRate(context.Background(), "XBTUSDTM", time.Time{}, time.Time{}, false, false, 0, 0)
	if err != nil {
		t.Error("GetFuturesInterestRate() error", err)
	}
}

func TestGetFuturesIndexList(t *testing.T) {
	t.Parallel()
	pairs, err := ku.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetFuturesIndexList(context.Background(), pairs[0].String(), time.Time{}, time.Time{}, false, false, 0, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesCurrentMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesCurrentMarkPrice(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesCurrentMarkPrice() error", err)
	}
}

func TestGetFuturesPremiumIndex(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPremiumIndex(context.Background(), "XBTUSDTM", time.Time{}, time.Time{}, false, false, 0, 0)
	if err != nil {
		t.Error("GetFuturesPremiumIndex() error", err)
	}
}

func TestGetFuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesCurrentFundingRate(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesCurrentFundingRate() error", err)
	}
}

func TestGetFuturesServerTime(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesServerTime(context.Background())
	if err != nil {
		t.Error("GetFuturesServerTime() error", err)
	}
}

func TestGetFuturesServiceStatus(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesServiceStatus(context.Background(), "XBTUSDTM")
	if err != nil {
		t.Error("GetFuturesServiceStatus() error", err)
	}
}

func TestGetFuturesKline(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesKline(context.Background(), "30", "XBTUSDTM", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesKline() error", err)
	}
}

func TestPostFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := ku.PostFuturesOrder(context.Background(), "5bd6e9286d99522a52e458de",
		"buy", "XBTUSDM", "", "10", "", "", "", 1, 1000, 0, 0.02,
		0, false, false, false, false, false, false)
	if err != nil {
		t.Error("PostFuturesOrder() error", err)
	}
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelFuturesOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("CancelFuturesOrder() error", err)
	}
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelAllFuturesOpenOrders(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("CancelAllFuturesOpenOrders() error", err)
	}
}

func TestCancelAllFuturesStopOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}
	_, err := ku.CancelAllFuturesStopOrders(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("CancelAllFuturesStopOrders() error", err)
	}
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetFuturesOrders(context.Background(), "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesOrders() error", err)
	}
}

func TestGetUntriggeredFuturesStopOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetUntriggeredFuturesStopOrders(context.Background(), "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetUntriggeredFuturesStopOrders() error", err)
	}
}

func TestGetFuturesRecentCompletedOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesRecentCompletedOrders(context.Background())
	if err != nil {
		t.Error("GetFuturesRecentCompletedOrders() error", err)
	}
}

func TestGetFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetFuturesOrderDetails(context.Background(), "5cdfc138b21023a909e5ad55")
	if err != nil && !strings.Contains(err.Error(), "error.getOrder.orderNotExist") {
		t.Error("GetFuturesOrderDetails() error", err)
	}
}

func TestGetFuturesOrderDetailsByClientID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesOrderDetailsByClientID(context.Background(), "eresc138b21023a909e5ad59")
	if err != nil && !strings.Contains(err.Error(), "error.getOrder.orderNotExist") {
		t.Error("GetFuturesOrderDetailsByClientID() error", err)
	}
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesFills(context.Background(), "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesFills() error", err)
	}
}

func TestGetFuturesRecentFills(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesRecentFills(context.Background())
	if err != nil {
		t.Error("GetFuturesRecentFills() error", err)
	}
}

func TestGetFuturesOpenOrderStats(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesOpenOrderStats(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("GetFuturesOpenOrderStats() error", err)
	}
}

func TestGetFuturesPosition(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesPosition(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("GetFuturesPosition() error", err)
	}
}

func TestGetFuturesPositionList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesPositionList(context.Background())
	if err != nil {
		t.Error("GetFuturesPositionList() error", err)
	}
}

func TestSetAutoDepositMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.SetAutoDepositMargin(context.Background(), "ADAUSDTM", true)
	if err != nil {
		t.Error("SetAutoDepositMargin() error", err)
	}
}

func TestAddMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.AddMargin(context.Background(), "XBTUSDTM", "6200c9b83aecfb000152dasfdee", 1)
	if err != nil {
		t.Error("AddMargin() error", err)
	}
}

func TestGetFuturesRiskLimitLevel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesRiskLimitLevel(context.Background(), "ADAUSDTM")
	if err != nil {
		t.Error("GetFuturesRiskLimitLevel() error", err)
	}
}

func TestUpdateRiskLmitLevel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.UpdateRiskLmitLevel(context.Background(), "ADASUDTM", 2)
	if err != nil {
		t.Error("UpdateRiskLmitLevel() error", err)
	}
}

func TestGetFuturesFundingHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	ku.Verbose = true
	_, err := ku.GetFuturesFundingHistory(context.Background(), "XBTUSDM", 0, 0, true, true, time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesFundingHistory() error", err)
	}
}

func TestGetFuturesAccountOverview(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesAccountOverview(context.Background(), "")
	if err != nil {
		t.Error("GetFuturesAccountOverview() error", err)
	}
}

func TestGetFuturesTransactionHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesTransactionHistory(context.Background(), "", "", 0, 0, true, time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesTransactionHistory() error", err)
	}
}

func TestCreateFuturesSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CreateFuturesSubAccountAPIKey(context.Background(), "", "passphrase", "", "remark", "subAccName")
	if err != nil {
		t.Error("CreateFuturesSubAccountAPIKey() error", err)
	}
}

func TestGetFuturesDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesDepositAddress(context.Background(), "XBT")
	if err != nil {
		t.Error("GetFuturesDepositAddress() error", err)
	}
}

func TestGetFuturesDepositsList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesDepositsList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesDepositsList() error", err)
	}
}

func TestGetFuturesWithdrawalLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesWithdrawalLimit(context.Background(), "XBT")
	if err != nil {
		t.Error("GetFuturesWithdrawalLimit() error", err)
	}
}

func TestGetFuturesWithdrawalList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetFuturesWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesWithdrawalList() error", err)
	}
}

func TestCancelFuturesWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.CancelFuturesWithdrawal(context.Background(), "5cda659603aa67131f305f7e")
	if err != nil {
		t.Error("CancelFuturesWithdrawal() error", err)
	}
}

func TestTransferFuturesFundsToMainAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	_, err := ku.TransferFuturesFundsToMainAccount(context.Background(), 1, "USDT", "MAIN")
	if err != nil {
		t.Error("TransferFuturesFundsToMainAccount() error", err)
	}
}

func TestTransferFundsToFuturesAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.TransferFundsToFuturesAccount(context.Background(), 1, "USDT", "MAIN")
	if err != nil {
		t.Error("TransferFundsToFuturesAccount() error", err)
	}
}

func TestGetFuturesTransferOutList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}

	_, err := ku.GetFuturesTransferOutList(context.Background(), "USDT", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesTransferOutList() error", err)
	}
}

func TestCancelFuturesTransferOut(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test: api keys not set or canManipulateRealOrders set to false")
	}

	err := ku.CancelFuturesTransferOut(context.Background(), "5cd53be30c19fc3754b60928")
	if err != nil {
		t.Error("CancelFuturesTransferOut() error", err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := ku.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	enabledPairs, err := ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ku.UpdateOrderbook(context.Background(), enabledPairs[0], asset.Futures); err != nil {
		t.Error(err)
	}
}
func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := ku.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
}
func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	newP, err := currency.NewPairFromString("MKR-USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = ku.UpdateTicker(context.Background(), newP, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	tradablePairs, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if _, err := ku.FetchTicker(context.Background(), tradablePairs[len(tradablePairs)-1], asset.Spot); err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	enabledPair, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if _, err := ku.FetchOrderbook(context.Background(), enabledPair[0], asset.Spot); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	enabledPairs, err := ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Error(err)
	}
	startTime := time.Now().Add(-time.Hour * 4)
	endTime := time.Now().Add(-time.Hour * 3)
	_, err = ku.GetHistoricCandles(context.Background(), enabledPairs[0], asset.Futures, kline.OneHour, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}
	enabledPairs, err = ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = ku.GetHistoricCandles(context.Background(), enabledPairs[len(enabledPairs)-1], asset.Spot, kline.OneHour, startTime, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	enabledPairs, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 5)
	endTime := time.Now().Add(-time.Hour * 2)
	_, err = ku.GetHistoricCandlesExtended(context.Background(), enabledPairs[0], asset.Spot, kline.OneHour, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetHistoricCandlesExtended(context.Background(), enabledPairs[0], asset.Spot, kline.FiveMin, startTime, endTime)
	if err != nil {
		t.Error(err)
	}
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetHistoricCandlesExtended(context.Background(), enabledPairs[0], asset.Futures, kline.OneHour, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetHistoricCandlesExtended(context.Background(), enabledPairs[0], asset.Futures, kline.FiveMin, startTime, endTime)
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := ku.GetServerTime(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = ku.GetServerTime(context.Background(), asset.Futures)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	enabledPairs, err := ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetRecentTrades(context.Background(), enabledPairs[0], asset.Futures)
	if err != nil {
		t.Error(err)
	}
	enabledPairs, err = ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetRecentTrades(context.Background(), enabledPairs[0], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.Limit,
		Pairs:     []currency.Pair{currency.NewPair(currency.LTC, currency.BTC), currency.NewPair(currency.MHC, currency.ETH), currency.NewPair(currency.MHC, currency.BTC), currency.NewPair(currency.OXEN, currency.BTC)},
		AssetType: asset.Futures,
		Side:      order.Sell,
	}
	_, err := ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	enabledPairs, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.Limit,
		Pairs:     enabledPairs,
		AssetType: asset.Futures,
		Side:      order.Buy,
	}
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Pairs = []currency.Pair{}
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}); err != nil {
		t.Errorf("%s GetFeeByType() error %v", ku.Name, err)
	}
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if ku.ValidateCredentials(context.Background(), asset.Spot) != ku.ValidateCredentials(context.Background(), asset.Futures) {
		t.Errorf("%s ValidateCredentials() error", ku.Name)
	}
}

func TestGetInstanceServers(t *testing.T) {
	t.Parallel()
	if _, err := ku.GetInstanceServers(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestWSConnect(t *testing.T) {
	t.Parallel()
	err := ku.WsConnect()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAuthenticatedServersInstances(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	_, err := ku.GetAuthenticatedInstanceServers(context.Background())
	if err != nil {
		t.Error(err)
	}
}

var (
	symbolTickerPushDataJSON     = `{"type":"message","topic":"/market/ticker:BTC-USDT","subject":"trade.ticker","data":{"sequence":"1545896668986","price":"0.08","size":"0.011","bestAsk":"0.08","bestAskSize":"0.18","bestBid":"0.049","bestBidSize":"0.036"}}`
	allSymbolsTickerPushDataJSON = `{"type":"message","topic":"/market/ticker:all","subject":"BTC-USDT","data":{"sequence":"1545896668986","bestAsk":"0.08","size":"0.011","bestBidSize":"0.036","price":"0.08","bestAskSize":"0.18",        "bestBid":"0.049"    }}`
)

func TestSybmbolTicker(t *testing.T) {
	err := ku.wsHandleData([]byte(symbolTickerPushDataJSON))
	if err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(allSymbolsTickerPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var symbolSnapshotPushDataJSON = `{"type": "message","topic": "/market/snapshot:KCS-BTC","subject": "trade.snapshot","data": {"sequence": "1545896669291","data": {"trading": true,"symbol": "KCS-BTC","buy": 0.00011,"sell": 0.00012,            "sort": 100,            "volValue": 3.13851792584,            "baseCurrency": "KCS",            "market": "BTC",            "quoteCurrency": "BTC",            "symbolCode": "KCS-BTC",            "datetime": 1548388122031,            "high": 0.00013,            "vol": 27514.34842,            "low": 0.0001,            "changePrice": -1.0e-5,            "changeRate": -0.0769,            "lastTradedPrice": 0.00012,            "board": 0,            "mark": 0        }    }}`

func TestSymbolSnapshotPushData(t *testing.T) {
	if err := ku.wsHandleData([]byte(symbolSnapshotPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var marketTradeSnapshotPushDataJSON = `{"type": "message","topic": "/market/snapshot:BTC","subject": "trade.snapshot","data": {"sequence": "1545896669291","data": [{"trading": true,"symbol": "KCS-BTC","buy": 0.00011,"sell": 0.00012,"sort": 100,"volValue": 3.13851792584,"baseCurrency": "KCS","market": "BTC","quoteCurrency": "BTC","symbolCode": "KCS-BTC","datetime": 1548388122031,"high": 0.00013,"vol": 27514.34842,"low": 0.0001,"changePrice": -1.0e-5,"changeRate": -0.0769,"lastTradedPrice": 0.00012,"board": 0,"mark": 0}]}}`

func TestMarketTradeSnapshotPushData(t *testing.T) {
	if err := ku.wsHandleData([]byte(marketTradeSnapshotPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	orderbookLevel2PushDataJSON = `{"type": "message","topic": "/market/level2:BTC-USDT","subject": "trade.l2update","data": {"changes": {"asks": [["18906","0.00331","14103845"],["18907.3","0.58751503","14103844"]],"bids": [["18891.9","0.15688","14103847"]]},"sequenceEnd": 14103847,"sequenceStart": 14103844,"symbol": "BTC-USDT","time": 1663747970273}}`
	orderbookLevel5PushDataJSON = `{"type": "message","topic": "/spotMarket/level2Depth5:BTC-USDT","subject": "level2","data": {"asks":[["9989","8"],["9990","32"],["9991","47"],["9992","3"],["9993","3"]],"bids":[["9988","56"],["9987","15"],["9986","100"],["9985","10"],["9984","10"]],"timestamp": 1586948108193}}`
	orderbookLevel50PushData    = `{"type":"message","topic": "/spotMarket/level2Depth50:BTC-USDT","subject": "level2","data": {"asks":[["9989","8"],["9990","32"],["9991","47"],["9992","3"],["9993","3"]],"bids":[["9988","56"],["9987","15"],["9986","100"],["9985","10"],["9984","10"]],"timestamp": 1586948108193}}`
)

func TestOrderbookPushData(t *testing.T) {
	err := ku.wsHandleData([]byte(orderbookLevel2PushDataJSON))
	if err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(orderbookLevel5PushDataJSON)); err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(orderbookLevel50PushData)); err != nil {
		t.Error(err)
	}
}

var tradeCandlesUpdatePushDataJSON = `{"type":"message","topic":"/market/candles:BTC-USDT_1hour","subject":"trade.candles.update","data":{"symbol":"BTC-USDT","candles":["1589968800","9786.9","9740.8","9806.1","9732","27.45649579","268280.09830877"],"time":1589970010253893337}}`

func TestTradeCandlestickPushDataJSON(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(tradeCandlesUpdatePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var matchExecutionPushDataJSON = `{"type":"message","topic":"/market/match:BTC-USDT","subject":"trade.l3match","data":{"sequence":"1545896669145","type":"match","symbol":"BTC-USDT","side":"buy","price":"0.08200000000000000000","size":"0.01022222000000000000","tradeId":"5c24c5da03aa673885cd67aa","takerOrderId":"5c24c5d903aa6772d55b371e","makerOrderId":"5c2187d003aa677bd09d5c93","time":"1545913818099033203"}}`

func TestMatchExecutionPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(matchExecutionPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var indexPricePushDataJSON = `{"id":"","type":"message","topic":"/indicator/index:USDT-BTC","subject":"tick","data":{"symbol": "USDT-BTC","granularity": 5000,"timestamp": 1551770400000,"value": 0.0001092}}`

func TestIndexPricePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(indexPricePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var markPricePushDataJSON = `{"type":"message","topic":"/indicator/markPrice:USDT-BTC","subject":"tick","data":{"symbol": "USDT-BTC","granularity": 5000,"timestamp": 1551770400000,"value": 0.0001093}}`

func TestMarkPricePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(markPricePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var orderbookChangePushDataJSON = `{"type":"message","topic":"/margin/fundingBook:USDT","subject":"funding.update","data":{"annualIntRate":0.0547,"currency":"USDT","dailyIntRate":0.00015,"sequence":87611418,"side":"lend","size":25040,"term":7,"ts":1671005721087508735}}`

func TestOrderbookChangePushDataJSON(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(orderbookChangePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	orderChangeStateOpenPushDataJSON      = `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"buy","orderId":"5efab07953bdea00089965d2","type":"open","orderTime":1593487481683297666,"size":"0.1","filledSize":"0","price":"0.937","clientOid":"1593487481000906","remainSize":"0.1","status":"open","ts":1593487481683297666}}`
	orderChangeStateMatchPushDataJSON     = `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"sell","orderId":"5efab07953bdea00089965fa","liquidity":"taker","type":"match","orderTime":1593487482038606180,"size":"0.1","filledSize":"0.1","price":"0.938","matchPrice":"0.96738","matchSize":"0.1","tradeId":"5efab07a4ee4c7000a82d6d9","clientOid":"1593487481000313","remainSize":"0","status":"match","ts":1593487482038606180}}`
	orderChangeStateFilledPushDataJSON    = `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"sell","orderId":"5efab07953bdea00089965fa","type":"filled","orderTime":1593487482038606180,"size":"0.1","filledSize":"0.1","price":"0.938","clientOid":"1593487481000313","remainSize":"0","status":"done","ts":1593487482038606180}}`
	orderChangeStateCancelledPushDataJSON = `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"buy","orderId":"5efab07953bdea00089965d2","type":"canceled","orderTime":1593487481683297666,"size":"0.1","filledSize":"0","price":"0.937","clientOid":"1593487481000906","remainSize":"0","status":"done","ts":1593487481893140844}}`
	orderChangeStateUpdatePushDataJSON    = `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"buy","orderId":"5efab13f53bdea00089971df","type":"update","oldSize":"0.1","orderTime":1593487679693183319,"size":"0.06","filledSize":"0","price":"0.937","clientOid":"1593487679000249","remainSize":"0.06","status":"open","ts":1593487682916117521}}`
)

func TestOrderChangePushData(t *testing.T) {
	t.Parallel()
	err := ku.wsHandleData([]byte(orderChangeStateOpenPushDataJSON))
	if err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(orderChangeStateMatchPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(orderChangeStateFilledPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(orderChangeStateCancelledPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(orderChangeStateUpdatePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var accountBalanceNoticePushDataJSON = `{"type": "message","topic": "/account/balance","subject": "account.balance","channelType":"private","data": {"total": "88","available": "88","availableChange": "88","currency": "KCS","hold": "0","holdChange": "0","relationEvent": "trade.setted","relationEventId": "5c21e80303aa677bd09d7dff","relationContext": {"symbol":"BTC-USDT","tradeId":"5e6a5dca9e16882a7d83b7a4","orderId":"5ea10479415e2f0009949d54"},"time": "1545743136994"}}`

func TestAccountBalanceNotice(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(accountBalanceNoticePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var debtRatioChangePushDataJSON = `{"type":"message","topic":"/margin/position","subject":"debt.ratio",    "channelType":"private",    "data": {        "debtRatio": 0.7505,        "totalDebt": "21.7505",        "debtList": {"BTC": "1.21","USDT": "2121.2121","EOS": "0"},        "timestamp": 15538460812100               }}`

func TestDebtRatioChangePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(debtRatioChangePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var positionStatusChangeEventPushDataJSON = `{"type":"message","topic":"/margin/position","subject":"position.status","channelType":"private","data": {"type": "FROZEN_FL","timestamp": 15538460812100}}`

func TestPositionStatusChangeEvent(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(positionStatusChangeEventPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	marginTradeOrderEntersEventPushDataJSON = `{"type": "message","topic": "/margin/loan:BTC","subject": "order.open","channelType":"private","data": {    "currency": "BTC",    "orderId": "ac928c66ca53498f9c13a127a60e8",    "dailyIntRate": 0.0001,    "term": 7,    "size": 1,        "side": "lend",    "ts": 1553846081210004941}}`
	marginTradeOrderUpdateEventPushDataJSON = `{"type": "message","topic": "/margin/loan:BTC","subject": "order.update","channelType":"private","data": {    "currency": "BTC",    "orderId": "ac928c66ca53498f9c13a127a60e8",    "dailyIntRate": 0.0001,    "term": 7,    "size": 1,    "lentSize": 0.5,    "side": "lend",    "ts": 1553846081210004941}}`
	marginTradeOrderDoneEventPushDataJSON   = `{"type": "message","topic": "/margin/loan:BTC","subject": "order.done","channelType":"private","data": {    "currency": "BTC",    "orderId": "ac928c66ca53498f9c13a127a60e8",    "reason": "filled",    "side": "lend",    "ts": 1553846081210004941  }}`
)

func TestMarginTradeOrderPushData(t *testing.T) {
	t.Parallel()
	err := ku.wsHandleData([]byte(marginTradeOrderEntersEventPushDataJSON))
	if err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(marginTradeOrderUpdateEventPushDataJSON)); err != nil {
		t.Error(err)
	}
	if err = ku.wsHandleData([]byte(marginTradeOrderDoneEventPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var stopOrderEventPushDataJSON = `{"type":"message","topic":"/spotMarket/advancedOrders","subject":"stopOrder","channelType":"private","data":{"createdAt":1589789942337,"orderId":"5ec244f6a8a75e0009958237","orderPrice":"0.00062","orderType":"stop","side":"sell","size":"1","stop":"entry","stopPrice":"0.00062","symbol":"KCS-BTC","tradeType":"TRADE","triggerSuccess":true,"ts":1589790121382281286,"type":"triggered"}}`

func TestStopOrderEventPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(stopOrderEventPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var publicFuturesTickerPushDataJSON = `{"subject": "tickerV2","topic": "/contractMarket/tickerV2:XBTUSDM","data": {"symbol": "XBTUSDM","bestBidSize": 795,"bestBidPrice": 3200.00,"bestAskPrice": 3600.00,"bestAskSize": 284,"ts": 1553846081210004941}}`

func TestPublicFuturesTickerPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(publicFuturesTickerPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var publicFuturesTickerV1PushDataJSON = `{"subject": "ticker","topic": "/contractMarket/ticker:XBTUSDM","data": {"symbol": "XBTUSDM","sequence": 45,"side": "sell","price": 3600.00,"size": 16,"tradeId": "5c9dcf4170744d6f5a3d32fb","bestBidSize": 795,"bestBidPrice": 3200.00,"bestAskPrice": 3600.00,"bestAskSize": 284,"ts": 1553846081210004941}}`

func TestPublicFuturesTickerV2PushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(publicFuturesTickerV1PushDataJSON)); err != nil {
		t.Error(err)
	}
}

var publicFuturesLevel2OrderbookPushDataJSON = `{"subject": "level2",  "topic": "/contractMarket/level2:XBTUSDM",  "type": "message",  "data": {    "sequence": 18,    "change": "5000.0,sell,83","timestamp": 1551770400000}}`

func TestPublicFuturesMarketData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(publicFuturesLevel2OrderbookPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	publicFuturesExecutionDataJSON               = `{"topic": "/contractMarket/execution:XBTUSDM","subject": "match","data": {"symbol": "XBTUSDM","sequence": 36,     "side": "buy","matchSize": 1,      "size": 1,"price": 3200.00,"takerOrderId": "5c9dd00870744d71c43f5e25","time": 1553846281766256031,"makerOrderId": "5c9d852070744d0976909a0c","tradeId": "5c9dd00970744d6f5a3d32fc"      }}`
	publicFuturesOrderbookWithDepth5PushDataJSON = `{"type": "message","topic": "/contractMarket/level2Depth5:XBTUSDM","subject": "level2","data": {"sequence": 1668672185642,"asks": [[17222.00000000,10802],[17223,5691],[17224,8153],[17225,5051],[17226,20000]],"bids": [[17221.00000000,4690],[17220,3766],[17219.00000000,4857],[17218,750],[17217.00000000,2100]],    "ts": 1670571892868,    "timestamp": 1670571892868    }}`
)

func TestPublicFuturesExecutionData(t *testing.T) {
	t.Parallel()
	ku.Verbose = true
	if err := ku.wsHandleData([]byte(publicFuturesExecutionDataJSON)); err != nil {
		t.Error(err)
	}
	if err := ku.wsHandleData([]byte(publicFuturesOrderbookWithDepth5PushDataJSON)); err != nil {
		t.Error(err)
	}
}

var privatePositionSettlementPushDataJSON = `{"userId": "xbc453tg732eba53a88ggyt8c","topic": "/contract/position:XBTUSDM","subject": "position.settlement","data": {"fundingTime": 1551770400000,"qty": 100,"markPrice": 3610.85,"fundingRate": -0.002966,"fundingFee": -296,"ts": 1547697294838004923,"settleCurrency": "XBT"}}`

func TestFuturesPositionSettlementPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(privatePositionSettlementPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var (
	futuresPositionChangePushDataJSON                 = `{ "userId": "5cd3f1a7b7ebc19ae9558591","topic": "/contract/position:XBTUSDM",  "subject": "position.change", "data": {"markPrice": 7947.83,"markValue": 0.00251640,"maintMargin": 0.00252044,"realLeverage": 10.06,"unrealisedPnl": -0.00014735,"unrealisedRoePcnt": -0.0553,"unrealisedPnlPcnt": -0.0553,"delevPercentage": 0.52,"currentTimestamp": 1558087175068,"settleCurrency": "XBT"}}`
	futuresPositionChangeWithChangeReasonPushDataJSON = `{ "type": "message","userId": "5c32d69203aa676ce4b543c7","channelType": "private","topic": "/contract/position:XBTUSDM",  "subject": "position.change", "data": {"realisedGrossPnl": 0E-8,"symbol":"XBTUSDM","crossMode": false,"liquidationPrice": 1000000.0,"posLoss": 0E-8,"avgEntryPrice": 7508.22,"unrealisedPnl": -0.00014735,"markPrice": 7947.83,"posMargin": 0.00266779,"autoDeposit": false,"riskLimit": 100000,"unrealisedCost": 0.00266375,"posComm": 0.00000392,"posMaint": 0.00001724,"posCost": 0.00266375,"maintMarginReq": 0.005,"bankruptPrice": 1000000.0,"realisedCost": 0.00000271,"markValue": 0.00251640,"posInit": 0.00266375,"realisedPnl": -0.00000253,"maintMargin": 0.00252044,"realLeverage": 1.06,"changeReason": "positionChange","currentCost": 0.00266375,"openingTimestamp": 1558433191000,"currentQty": -20,"delevPercentage": 0.52,"currentComm": 0.00000271,"realisedGrossCost": 0E-8,"isOpen": true,"posCross": 1.2E-7,"currentTimestamp": 1558506060394,"unrealisedRoePcnt": -0.0553,"unrealisedPnlPcnt": -0.0553,"settleCurrency": "XBT"}}`
)

func TestFuturesPositionChangePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresPositionChangePushDataJSON)); err != nil {
		t.Error(err)
	}
	if err := ku.wsHandleData([]byte(futuresPositionChangeWithChangeReasonPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresWithdrawalAmountTransferOutAmountEventPushDataJSON = `{ "userId": "xbc453tg732eba53a88ggyt8c","topic": "/contractAccount/wallet","subject": "withdrawHold.change","data": {"withdrawHold": 5923,"currency":"USDT","timestamp": 1553842862614}}`

func TestFuturesWithdrawHoldChangePushDataJSON(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresWithdrawalAmountTransferOutAmountEventPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresAvailableBalanceChangePushData = `{ "userId": "xbc453tg732eba53a88ggyt8c","topic": "/contractAccount/wallet","subject": "availableBalance.change","data": {"availableBalance": 5923,"holdBalance": 2312,"currency":"USDT","timestamp": 1553842862614}}`

func TestFuturesAvailableBalanceChangePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresAvailableBalanceChangePushData)); err != nil {
		t.Error(err)
	}
}

var futuresOrderMarginChangePushDataJSON = `{ "userId": "xbc453tg732eba53a88ggyt8c","topic": "/contractAccount/wallet","subject": "orderMargin.change","data": {"orderMargin": 5923,"currency":"USDT","timestamp": 1553842862614}}`

func TestFuturesOrderMarginChangePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresOrderMarginChangePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresStopOrderPushDataJSON = `{"userId": "5cd3f1a7b7ebc19ae9558591","topic": "/contractMarket/advancedOrders", "subject": "stopOrder","data": {"orderId": "5cdfc138b21023a909e5ad55","symbol": "XBTUSDM","type": "open","orderType":"stop","side":"buy","size":"1000","orderPrice":"9000","stop":"up","stopPrice":"9100","stopPriceType":"TP","triggerSuccess": true,"error": "error.createOrder.accountBalanceInsufficient","createdAt": 1558074652423,"ts":1558074652423004000}}`

func TestFuturesStopOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresStopOrderPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresTradeOrdersPushDataJSON = `{"type": "message","topic": "/contractMarket/tradeOrders","subject": "orderChange","channelType": "private","data": {"orderId": "5cdfc138b21023a909e5ad55","symbol": "XBTUSDM","type": "match","status": "open","matchSize": "","matchPrice": "","orderType": "limit","side": "buy","price": "3600","size": "20000","remainSize": "20001","filledSize":"20000","canceledSize": "0","tradeId": "5ce24c16b210233c36eexxxx","clientOid": "5ce24c16b210233c36ee321d","orderTime": 1545914149935808589,"oldSize ": "15000","liquidity": "maker","ts": 1545914149935808589}}`

func TestFuturesTradeOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresTradeOrdersPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var transactionStaticsPushDataJSON = `{ "topic": "/contractMarket/snapshot:XBTUSDM","subject": "snapshot.24h","data": {"volume": 30449670,      "turnover": 845169919063,"lastPrice": 3551,       "priceChgPct": 0.0043,   "ts": 1547697294838004923}  }`

func TestFuturesTrasactionStaticsPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(transactionStaticsPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresEndFundingFeeSettlementPushDataJSON = `{ "type":"message","topic": "/contract/announcement","subject": "funding.end","data": {"symbol": "XBTUSDM",         "fundingTime": 1551770400000,"fundingRate": -0.002966,    "timestamp": 1551770410000          }}`

func TestFuturesEndFundingFeeSettlement(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresEndFundingFeeSettlementPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresStartFundingFeeSettlementPushDataJSON = `{ "topic": "/contract/announcement","subject": "funding.begin","data": {"symbol": "XBTUSDM","fundingTime": 1551770400000,"fundingRate": -0.002966,"timestamp": 1551770400000}}`

func TestFuturesStartFundingFeeSettlementPushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresStartFundingFeeSettlementPushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresFundingRatePushDataJSON = `{ "topic": "/contract/instrument:XBTUSDM","subject": "funding.rate","data": {"granularity": 60000,"fundingRate": -0.002966,"timestamp": 1551770400000}}`

func TestFuturesFundingRatePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresFundingRatePushDataJSON)); err != nil {
		t.Error(err)
	}
}

var futuresMarkIndexPricePushDataJSON = `{ "topic": "/contract/instrument:XBTUSDM","subject": "mark.index.price","data": {"granularity": 1000,"indexPrice": 4000.23,"markPrice": 4010.52,"timestamp": 1551770400000}}`

func TestFuturesMarkIndexPricePushData(t *testing.T) {
	t.Parallel()
	if err := ku.wsHandleData([]byte(futuresMarkIndexPricePushDataJSON)); err != nil {
		t.Error(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if _, err := ku.GenerateDefaultSubscriptions(); err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetAvailableTransferChains(context.Background(), currency.BTC); err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Futures); err != nil {
		t.Errorf("%s GetWithdrawalsHistory() error %v", ku.Name, err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("Kucoin GetOrderInfo() skipping test: api keys not set")
	}
	enabled, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error("couldn't find enabled tradable pairs")
	}
	if len(enabled) == 0 {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	_, err = ku.GetOrderInfo(context.Background(), "123", enabled[0], asset.Futures)
	if err != nil && !strings.Contains(err.Error(), "orderNotExist") {
		t.Errorf("Kucoin GetOrderInfo() expecting %s, but found %v", "Order does not exist", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetDepositAddress(context.Background(), currency.BTC, "", ""); err != nil && errors.Is(err, errNoDepositAddress) {
		t.Error("Kucoin GetDepositAddress() error", err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	withdrawCryptoRequest := withdraw.Request{
		Exchange: ku.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}
	if _, err := ku.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest); err != nil {
		t.Error("Kucoin WithdrawCryptoCurrencyFunds() error", err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  ku.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOrder",
		AssetType: asset.Spot,
	}
	_, err := ku.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error("Kucoin SubmitOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}
	if err := ku.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	if _, err := ku.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType:  asset.Margin,
		MarginMode: "isolated",
	}); err != nil {
		t.Errorf("%s CancelAllOrders() error: %v", ku.Name, err)
	}
}

func TestGeneratePayloads(t *testing.T) {
	t.Parallel()
	subscriptions, err := ku.GenerateDefaultSubscriptions()
	if err != nil {
		t.Error(err)
	}
	payload, err := ku.generatePayloads(subscriptions, "subscribe")
	if err != nil {
		t.Error(err)
	}
	if len(payload) != len(subscriptions) {
		t.Error(errors.New("derived payload is not same as generated channel subscription instances"))
	}
}

func TestCreateSubUser(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	if _, err := ku.CreateSubUser(context.Background(), "SamuaelTee1", "sdfajdlkad", "", ""); err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountSpotAPIList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetSubAccountSpotAPIList(context.Background(), "sam", ""); err != nil {
		t.Error(err)
	}
}

func TestCreateSpotAPIsForSubAccount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	if _, err := ku.CreateSpotAPIsForSubAccount(context.Background(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		Remark:         "123456",
	}); err != nil {
		t.Error(err)
	}
}

func TestModifySubAccountSpotAPIs(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	if _, err := ku.ModifySubAccountSpotAPIs(context.Background(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		Remark:         "123456",
	}); err != nil {
		t.Error(err)
	}
}

func TestDeleteSubAccountSpoAPI(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(cantManipulateRealOrdersOrKeysNotSet)
	}
	if _, err := ku.DeleteSubAccountSpotAPI(context.Background(), apiKey, "mysecretPassphrase123", "gocryptoTrader1"); err != nil {
		t.Error(err)
	}
}

func TestGetUserInfoOfAllSubAccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetUserInfoOfAllSubAccounts(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetPaginatedListOfSubAccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credentialsNotSet)
	}
	if _, err := ku.GetPaginatedListOfSubAccounts(context.Background(), 1, 100); err != nil {
		t.Error(err)
	}
}
