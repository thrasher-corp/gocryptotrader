package kucoin

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
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
)

var (
	ku                                                        = &Kucoin{}
	spotTradablePair, marginTradablePair, futuresTradablePair currency.Pair
)

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
	if apiKey != "" && apiSecret != "" && passPhrase != "" {
		ku.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

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
	setupWS()
	getFirstTradablePairOfAssets()
	os.Exit(m.Run())
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
	_, err := ku.GetTickers(context.Background())
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetMarginAccount(context.Background())
	if err != nil {
		t.Error("GetMarginAccount() error", err)
	}
}

func TestGetMarginRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostBorrowOrder(context.Background(), "USDT", "FOK", "", 10, 0)
	if err != nil {
		t.Error("PostBorrowOrder() error", err)
	}

	_, err = ku.PostBorrowOrder(context.Background(), "USDT", "IOC", "7,14,28", 10, 0.05)
	if err != nil {
		t.Error("PostBorrowOrder() error", err)
	}
}

const borrowOrderJSON = `{"orderId": "a2111213","currency": "USDT","size": "1.009","filled": 1.009,"matchList": [{"currency": "USDT","dailyIntRate": "0.001","size": "12.9","term": 7,"timestamp": "1544657947759","tradeId": "1212331"}],"status": "DONE"}`

func TestGetBorrowOrder(t *testing.T) {
	t.Parallel()
	var resp *BorrowOrder
	err := json.Unmarshal([]byte(borrowOrderJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetBorrowOrder(context.Background(), "orderID")
	if err != nil {
		t.Error("GetBorrowOrder() error", err)
	}
}

const outstandingRecordResponseJSON = `{"currentPage": 0, "pageSize": 0, "totalNum": 0, "totalPage": 0, "items": [ { "tradeId": "1231141", "currency": "USDT", "accruedInterest": "0.22121", "dailyIntRate": "0.0021", "liability": "1.32121", "maturityTime": "1544657947759", "principal": "1.22121", "repaidSize": "0", "term": 7, "createdAt": "1544657947759" } ] }`

func TestGetOutstandingRecord(t *testing.T) {
	t.Parallel()
	var resp *OutstandingRecordResponse
	err := json.Unmarshal([]byte(outstandingRecordResponseJSON), &resp)
	if err != nil {
		t.Error(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetOutstandingRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetOutstandingRecord() error", err)
	}
}

const repaidRecordJSON = `{"pageSize": 0, "totalNum": 0, "totalPage": 0, "currentPage": 0, "items": [ { "tradeId": "1231141", "currency": "USDT", "dailyIntRate": "0.0021", "interest": "0.22121", "principal": "1.22121", "repaidSize": "0", "repayTime": "1544657947759", "term": 7 } ] }`

func TestGetRepaidRecord(t *testing.T) {
	t.Parallel()
	var resp *RepaidRecordsResponse
	err := json.Unmarshal([]byte(repaidRecordJSON), &resp)
	if err != nil {
		t.Error(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetRepaidRecord(context.Background(), "BTC")
	if err != nil {
		t.Error("GetRepaidRecord() error", err)
	}
}

func TestOneClickRepayment(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.OneClickRepayment(context.Background(), "BTC", "RECENTLY_EXPIRE_FIRST", 2.5)
	if err != nil {
		t.Error("OneClickRepayment() error", err)
	}
}

func TestSingleOrderRepayment(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.SingleOrderRepayment(context.Background(), "BTC", "fa3e34c980062c10dad74016", 2.5)
	if err != nil {
		t.Error("SingleOrderRepayment() error", err)
	}
}

func TestPostLendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostLendOrder(context.Background(), "BTC", 0.0001, 5, 7)
	if err != nil {
		t.Error("PostLendOrder() error", err)
	}
}

func TestCancelLendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.CancelLendOrder(context.Background(), "OrderID")
	if err != nil {
		t.Error("CancelLendOrder() error", err)
	}
}

func TestSetAutoLend(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.SetAutoLend(context.Background(), "BTC", 0.0002, 0.005, 7, true)
	if err != nil {
		t.Error("SetAutoLend() error", err)
	}
}

const activeOrderResponseJSON = `[ { "orderId": "5da59f5ef943c033b2b643e4", "currency": "BTC", "size": "0.51", "filledSize": "0", "dailyIntRate": "0.0001", "term": 7, "createdAt": 1571135326913 } ]`

func TestGetActiveOrder(t *testing.T) {
	t.Parallel()
	var resp []LendOrder
	err := json.Unmarshal([]byte(activeOrderResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetActiveOrder(context.Background(), "")
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetLendHistory(context.Background(), "")
	if err != nil {
		t.Error("GetLendHistory() error", err)
	}
	_, err = ku.GetLendHistory(context.Background(), "BTC")
	if err != nil {
		t.Error("GetLendHistory() error", err)
	}
}

const activeLentOrderResponseJSON = `[ { "tradeId": "5da6dba0f943c0c81f5d5db5", "currency": "BTC", "size": "0.51", "accruedInterest": "0", "repaid": "0.10999968", "dailyIntRate": "0.0001", "term": 14, "maturityTime": 1572425888958 } ]`

func TestGetUnsettleLendOrder(t *testing.T) {
	t.Parallel()
	var resp []UnsettleLendOrder
	err := json.Unmarshal([]byte(activeLentOrderResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetUnsettledLendOrder(context.Background(), "")
	if err != nil {
		t.Error("GetUnsettledLendOrder() error", err)
	}

	_, err = ku.GetUnsettledLendOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetUnsettledLendOrder() error", err)
	}
}

const settledLendOrderResponseJSON = `[{ "tradeId": "5da59fe6f943c033b2b6440b", "currency": "BTC", "size": "0.51", "interest": "0.00004899", "repaid": "0.510041641", "dailyIntRate": "0.0001", "term": 7, "settledAt": 1571216254767, "note": "The account of the borrowers reached a negative balance, and the system has supplemented the loss via the insurance fund. Deposit funds: 0.51." } ]`

func TestGetSettleLendOrder(t *testing.T) {
	t.Parallel()
	var resp []SettleLendOrder
	err := json.Unmarshal([]byte(settledLendOrderResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetSettledLendOrder(context.Background(), "")
	if err != nil {
		t.Error("GetSettledLendOrder() error", err)
	}
	_, err = ku.GetSettledLendOrder(context.Background(), "BTC")
	if err != nil {
		t.Error("GetSettledLendOrder() error", err)
	}
}

func TestGetAccountLendRecord(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetMarginTradeData(context.Background(), "BTC")
	if err != nil {
		t.Error("GetMarginTradeData() error", err)
	}
}

func TestGetIsolatedMarginPairConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetIsolatedMarginPairConfig(context.Background())
	if err != nil {
		t.Error("GetIsolatedMarginPairConfig() error", err)
	}
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetSingleIsolatedMarginAccountInfo(context.Background(), "BTC-USDT")
	if err != nil {
		t.Error("GetSingleIsolatedMarginAccountInfo() error", err)
	}
}

func TestInitiateIsolateMarginBorrowing(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.InitiateIsolatedMarginBorrowing(context.Background(), "BTC-USDT", "USDT", "FOK", "", 10, 0)
	if err != nil {
		t.Error("InitiateIsolateMarginBorrowing() error", err)
	}
}

func TestGetIsolatedOutstandingRepaymentRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.InitiateIsolatedMarginQuickRepayment(context.Background(), "BTC-USDT", "USDT", "RECENTLY_EXPIRE_FIRST", 10)
	if err != nil {
		t.Error("InitiateIsolatedMarginQuickRepayment() error", err)
	}
}

func TestInitiateIsolatedMarginSingleRepayment(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	// default order type is limit
	_, err := ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: ""})
	if !errors.Is(err, errInvalidClientOrderID) {
		t.Errorf("PostOrder() expected %v, but found %v", errInvalidClientOrderID, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: spotTradablePair,
		OrderType: ""})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("PostOrder() expected %v, but found %v", order.ErrSideIsInvalid, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("PostOrder() expected %v, but found %v", currency.ErrCurrencyPairEmpty, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: 0.1})
	if !errors.Is(err, errInvalidPrice) {
		t.Errorf("PostOrder() expected %v, but found %v", errInvalidPrice, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: spotTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565})
	if !errors.Is(err, errInvalidSize) {
		t.Errorf("PostOrder() expected %v, but found %v", errInvalidSize, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy",
		Symbol: spotTradablePair, OrderType: "limit", Size: 0.1, Price: 234565})
	if err != nil {
		t.Error("PostOrder() error", err)
	}

	// market order
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "market", Remark: "remark", Size: 0.1})
	if err != nil {
		t.Error("PostOrder() error", err)
	}
}

func TestPostMarginOrder(t *testing.T) {
	t.Parallel()
	// default order type is limit
	_, err := ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: ""})
	if !errors.Is(err, errInvalidClientOrderID) {
		t.Errorf("PostMarginOrder() expected %v, but found %v", errInvalidClientOrderID, err)
	}
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair,
		OrderType: ""})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("PostMarginOrder() expected %v, but found %v", order.ErrSideIsInvalid, err)
	}
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("PostMarginOrder() expected %v, but found %v", currency.ErrCurrencyPairEmpty, err)
	}
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy",
		Symbol:    marginTradablePair,
		OrderType: "limit", Size: 0.1})
	if !errors.Is(err, errInvalidPrice) {
		t.Errorf("PostMarginOrder() expected %v, but found %v", errInvalidPrice, err)
	}
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565})
	if !errors.Is(err, errInvalidSize) {
		t.Errorf("PostMarginOrder() expected %v, but found %v", errInvalidSize, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	// default order type is limit and margin mode is cross
	_, err = ku.PostMarginOrder(context.Background(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			Price: 1000, Size: 0.1, PostOnly: true})
	if err != nil {
		t.Error("PostMarginOrder() error", err)
	}

	// market isolated order
	_, err = ku.PostMarginOrder(context.Background(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			OrderType: "market", Funds: 1234,
			Remark: "remark", MarginMode: "cross", Price: 1000, PostOnly: true, AutoBorrow: true})
	if err != nil {
		t.Error("PostMarginOrder() error", err)
	}
}

func TestPostBulkOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelSingleOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("CancelSingleOrder() error", err)
	}
}

func TestCancelOrderByClientOID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelOrderByClientOID(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("CancelOrderByClientOID() error", err)
	}
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelAllOpenOrders(context.Background(), "", "")
	if err != nil {
		t.Error("CancelAllOpenOrders() error", err)
	}
}

const ordersListResponseJSON = `{"currentPage": 1, "pageSize": 1, "totalNum": 153408, "totalPage": 153408, "items": [ { "id": "5c35c02703aa673ceec2a168", "symbol": "BTC-USDT", "opType": "DEAL", "type": "limit", "side": "buy", "price": "10", "size": "2", "funds": "0", "dealFunds": "0.166", "dealSize": "2", "fee": "0", "feeCurrency": "USDT", "stp": "", "stop": "", "stopTriggered": false, "stopPrice": "0", "timeInForce": "GTC", "postOnly": false, "hidden": false, "iceberg": false, "visibleSize": "0", "cancelAfter": 0, "channel": "IOS", "clientOid": "", "remark": "", "tags": "", "isActive": false, "cancelExist": false, "createdAt": 1547026471000, "tradeType": "TRADE" } ] }`

func TestGetOrders(t *testing.T) {
	t.Parallel()
	var resp *OrdersListResponse
	err := json.Unmarshal([]byte(ordersListResponseJSON), &resp)
	if err != nil {
		t.Error(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err = ku.ListOrders(context.Background(), "", "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetOrders() error", err)
	}
}

func TestGetRecentOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetRecentOrders(context.Background())
	if err != nil {
		t.Error("GetRecentOrders() error", err)
	}
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetOrderByID(context.Background(), "5c35c02703aa673ceec2a168")
	if err != nil {
		t.Error("GetOrderByID() error", err)
	}
}

func TestGetOrderByClientOID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetOrderByClientSuppliedOrderID(context.Background(), "6d539dc614db312")
	if err != nil {
		t.Error("GetOrderByClientOID() error", err)
	}
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFills(context.Background(), "", "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFills() error", err)
	}
	_, err = ku.GetFills(context.Background(), "5c35c02703aa673ceec2a168", "BTC-USDT", "buy", "limit", "TRADE", time.Now().Add(-time.Hour*12), time.Now())
	if err != nil {
		t.Error("GetFills() error", err)
	}
}

const limitFillsResponseJSON = `[{ "counterOrderId":"5db7ee769797cf0008e3beea", "createdAt":1572335233000, "fee":"0.946357371456", "feeCurrency":"USDT", "feeRate":"0.001", "forceTaker":true, "funds":"946.357371456", "liquidity":"taker", "orderId":"5db7ee805d53620008dce1ba", "price":"9466.8", "side":"buy", "size":"0.09996592", "stop":"", "symbol":"BTC-USDT", "tradeId":"5db7ee8054c05c0008069e21", "tradeType":"MARGIN_TRADE", "type":"market" }, { "counterOrderId":"5db7ee4b5d53620008dcde8e", "createdAt":1572335207000, "fee":"0.94625", "feeCurrency":"USDT", "feeRate":"0.001", "forceTaker":true, "funds":"946.25", "liquidity":"taker", "orderId":"5db7ee675d53620008dce01e", "price":"9462.5", "side":"sell", "size":"0.1", "stop":"", "symbol":"BTC-USDT", "tradeId":"5db7ee6754c05c0008069e03", "tradeType":"MARGIN_TRADE", "type":"market" }]`

func TestGetRecentFills(t *testing.T) {
	t.Parallel()
	var resp []Fill
	err := json.Unmarshal([]byte(limitFillsResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetRecentFills(context.Background())
	if err != nil {
		t.Error("GetRecentFills() error", err)
	}
}

func TestPostStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostStopOrder(context.Background(), "5bd6e9286d99522a52e458de", "buy", "BTC-USDT", "", "", "entry", "CO", "TRADE", "", 0.1, 1, 10, 0, 0, 0, true, false, false)
	if err != nil {
		t.Error("PostStopOrder() error", err)
	}
}

func TestCancelStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelStopOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("CancelStopOrder() error", err)
	}
}

func TestCancelAllStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelStopOrders(context.Background(), "", "", "")
	if err != nil {
		t.Error("CancelAllStopOrder() error", err)
	}
}

const stopOrderResponseJSON = `{"id": "vs8hoo8q2ceshiue003b67c0", "symbol": "KCS-USDT", "userId": "60fe4956c43cbc0006562c2c", "status": "NEW", "type": "limit", "side": "buy", "price": "0.01000000000000000000", "size": "0.01000000000000000000", "funds": null, "stp": null, "timeInForce": "GTC", "cancelAfter": -1, "postOnly": false, "hidden": false, "iceberg": false, "visibleSize": null, "channel": "API", "clientOid": "40e0eb9efe6311eb8e58acde48001122", "remark": null, "tags": null, "orderTime": 1629098781127530345, "domainId": "kucoin", "tradeSource": "USER", "tradeType": "TRADE", "feeCurrency": "USDT", "takerFeeRate": "0.00200000000000000000", "makerFeeRate": "0.00200000000000000000", "createdAt": 1629098781128, "stop": "loss", "stopTriggerTime": null, "stopPrice": "10.00000000000000000000" }`

func TestGetStopOrder(t *testing.T) {
	t.Parallel()
	var resp *StopOrder
	err := json.Unmarshal([]byte(stopOrderResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetStopOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("GetStopOrder() error", err)
	}
}

func TestGetAllStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.ListStopOrders(context.Background(), "", "", "", "", "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error("GetAllStopOrder() error", err)
	}
}

func TestGetStopOrderByClientID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetStopOrderByClientID(context.Background(), "", "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("GetStopOrderByClientID() error", err)
	}
}

func TestCancelStopOrderByClientID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelStopOrderByClientID(context.Background(), "", "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("CancelStopOrderByClientID() error", err)
	}
}

func TestGetAllAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetAllAccounts(context.Background(), "", "")
	if err != nil {
		t.Error("GetAllAccounts() error", err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetAccount(context.Background(), "62fcd1969474ea0001fd20e4")
	if err != nil {
		t.Error("GetAccount() error", err)
	}
}

const accountLedgerResponseJSON = `{"currentPage": 1, "pageSize": 50, "totalNum": 2, "totalPage": 1, "items": [ { "id": "611a1e7c6a053300067a88d9", "currency": "USDT", "amount": "10.00059547", "fee": "0", "balance": "0", "accountType": "MAIN", "bizType": "Loans Repaid", "direction": "in", "createdAt": 1629101692950, "context": "{\"borrowerUserId\":\"601ad03e50dc810006d242ea\",\"loanRepayDetailNo\":\"611a1e7cc913d000066cf7ec\"}" }, { "id": "611a18bc6a0533000671e1bf", "currency": "USDT", "amount": "10.00059547", "fee": "0", "balance": "0", "accountType": "MAIN", "bizType": "Loans Repaid", "direction": "in", "createdAt": 1629100220843, "context": "{\"borrowerUserId\":\"5e3f4623dbf52d000800292f\",\"loanRepayDetailNo\":\"611a18bc7255c200063ea545\"}" } ] }`

func TestGetAccountLedgers(t *testing.T) {
	t.Parallel()
	var resp *AccountLedgerResponse
	err := json.Unmarshal([]byte(accountLedgerResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetAccountLedgers(context.Background(), "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetAccountLedgers() error", err)
	}
}

func TestGetAccountSummaryInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetAccountSummaryInformation(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetSubAccountBalance(context.Background(), "62fcd1969474ea0001fd20e4", false)
	if err != nil {
		t.Error("GetSubAccountBalance() error", err)
	}
}

func TestGetAggregatedSubAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAggregatedSubAccountBalance(context.Background())
	if err != nil {
		t.Error("GetAggregatedSubAccountBalance() error", err)
	}
}

func TestGetPaginatedSubAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetPaginatedSubAccountInformation(context.Background(), 0, 10)
	if err != nil {
		t.Error("GetPaginatedSubAccountInformation() error", err)
	}
}

func TestGetTransferableBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetTransferableBalance(context.Background(), "BTC", "MAIN", "")
	if err != nil {
		t.Error("GetTransferableBalance() error", err)
	}
}

func TestTransferMainToSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.TransferMainToSubAccount(context.Background(), "62fcd1969474ea0001fd20e4", "BTC", "1", "OUT", "", "", "5caefba7d9575a0688f83c45")
	if err != nil {
		t.Error("TransferMainToSubAccount() error", err)
	}
}

func TestMakeInnerTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.MakeInnerTransfer(context.Background(), "62fcd1969474ea0001fd20e4", "BTC", "trade", "main", "1", "", "")
	if err != nil {
		t.Error("MakeInnerTransfer() error", err)
	}
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetDepositAddressesV2(context.Background(), "BTC")
	if err != nil {
		t.Error("GetDepositAddressV2() error", err)
	}
}

func TestGetDepositAddressesV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetDepositAddressV1(context.Background(), "BTC", "")
	if err != nil {
		t.Error("GetDepositAddressV1() error", err)
	}
}

const depositResponseJSON = `{"currentPage": 1, "pageSize": 50, "totalNum": 1, "totalPage": 1, "items": [ { "currency": "XRP", "chain": "xrp", "status": "SUCCESS", "address": "rNFugeoj3ZN8Wv6xhuLegUBBPXKCyWLRkB", "memo": "1919537769", "isInner": false, "amount": "20.50000000", "fee": "0.00000000", "walletTxId": "2C24A6D5B3E7D5B6AA6534025B9B107AC910309A98825BF5581E25BEC94AD83B@e8902757998fc352e6c9d8890d18a71c", "createdAt": 1666600519000, "updatedAt": 1666600549000, "remark": "Deposit" } ] }`

func TestGetDepositList(t *testing.T) {
	t.Parallel()
	var resp DepositResponse
	err := json.Unmarshal([]byte(depositResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetDepositList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetDepositList() error", err)
	}
}

const historicalDepositResponseJSON = `{"currentPage":1, "pageSize":1, "totalNum":9, "totalPage":9, "items":[ { "currency":"BTC", "createAt":1528536998, "amount":"0.03266638", "walletTxId":"55c643bc2c68d6f17266383ac1be9e454038864b929ae7cee0bc408cc5c869e8@12ffGWmMMD1zA1WbFm7Ho3JZ1w6NYXjpFk@234", "isInner":false, "status":"SUCCESS" } ] }`

func TestGetHistoricalDepositList(t *testing.T) {
	t.Parallel()
	var resp *HistoricalDepositWithdrawalResponse
	err := json.Unmarshal([]byte(historicalDepositResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetHistoricalDepositList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetHistoricalDepositList() error", err)
	}
}

func TestGetWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetWithdrawalList() error", err)
	}
}

func TestGetHistoricalWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetHistoricalWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error("GetHistoricalWithdrawalList() error", err)
	}
}

func TestGetWithdrawalQuotas(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetWithdrawalQuotas(context.Background(), "BTC", "")
	if err != nil {
		t.Error("GetWithdrawalQuotas() error", err)
	}
}

func TestApplyWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.ApplyWithdrawal(context.Background(), "ETH", "0x597873884BC3a6C10cB6Eb7C69172028Fa85B25A", "", "", "", "", false, 1)
	if err != nil {
		t.Error("ApplyWithdrawal() error", err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.CancelWithdrawal(context.Background(), "5bffb63303aa675e8bbe18f9")
	if err != nil {
		t.Error("CancelWithdrawal() error", err)
	}
}

func TestGetBasicFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetBasicFee(context.Background(), "1")
	if err != nil {
		t.Error("GetBasicFee() error", err)
	}
}

func TestGetTradingFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

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
	_, err := ku.GetFuturesOrderbook(context.Background(), futuresTradablePair.String())
	if err != nil {
		t.Error("GetFuturesOrderbook() error", err)
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
	_, err := ku.GetFuturesIndexList(context.Background(), futuresTradablePair.String(), time.Time{}, time.Time{}, false, false, 0, 10)
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
	_, err := ku.GetFuturesServiceStatus(context.Background())
	if err != nil {
		t.Error("GetFuturesServiceStatus() error", err)
	}
}

func TestGetFuturesKline(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesKline(context.Background(), int64(kline.ThirtyMin.Duration().Minutes()), "XBTUSDTM", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesKline() error", err)
	}
}

func TestPostFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de"})
	if !errors.Is(err, errInvalidLeverage) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidLeverage, err)
	}
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{Side: "buy", Leverage: 0.02})
	if !errors.Is(err, errInvalidClientOrderID) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidClientOrderID, err)
	}
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Leverage: 0.02})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", order.ErrSideIsInvalid, err)
	}
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Leverage: 0.02})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", currency.ErrCurrencyPairEmpty, err)
	}

	// With Stop order configuration
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 0.02, VisibleSize: 0})
	if !errors.Is(err, errInvalidStopPriceType) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidStopPriceType, err)
	}

	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 0.02, VisibleSize: 0})
	if !errors.Is(err, errInvalidPrice) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidPrice, err)
	}

	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", StopPrice: 123456, TimeInForce: "", Size: 1, Price: 1000, Leverage: 0.02, VisibleSize: 0})
	if err != nil {
		t.Errorf("PostFuturesOrder expected %v", err)
	}

	// Limit Orders
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "limit", Remark: "10", Leverage: 0.02})
	if !errors.Is(err, errInvalidPrice) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidPrice, err)
	}
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10", Price: 1000, Leverage: 0.02, VisibleSize: 0})
	if !errors.Is(err, errInvalidSize) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidSize, err)
	}
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Size: 1, Price: 1000, Leverage: 0.02, VisibleSize: 0})
	if err != nil {
		t.Error(err)
	}

	// Market Orders
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "market", Remark: "10", Leverage: 0.02})
	if !errors.Is(err, errInvalidSize) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidSize, err)
	}
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "market", Remark: "10",
		Size: 1, Leverage: 0.02, VisibleSize: 0})
	if !errors.Is(err, errInvalidSize) {
		t.Errorf("PostFuturesOrder expected %v, but found %v", errInvalidSize, err)
	}

	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de",
		Side:          "buy",
		Symbol:        futuresTradablePair,
		OrderType:     "limit",
		Remark:        "10",
		Stop:          "",
		StopPriceType: "",
		TimeInForce:   "",
		Size:          1,
		Price:         1000,
		StopPrice:     0,
		Leverage:      0.02,
		VisibleSize:   0})
	if err != nil {
		t.Error("PostFuturesOrder() error", err)
	}
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelFuturesOrder(context.Background(), "5bd6e9286d99522a52e458de")
	if err != nil {
		t.Error("CancelFuturesOrder() error", err)
	}
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelAllFuturesOpenOrders(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("CancelAllFuturesOpenOrders() error", err)
	}
}

func TestCancelAllFuturesStopOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelAllFuturesStopOrders(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("CancelAllFuturesStopOrders() error", err)
	}
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOrders(context.Background(), "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesOrders() error", err)
	}
}

func TestGetUntriggeredFuturesStopOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetUntriggeredFuturesStopOrders(context.Background(), "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetUntriggeredFuturesStopOrders() error", err)
	}
}

func TestGetFuturesRecentCompletedOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesRecentCompletedOrders(context.Background())
	if err != nil {
		t.Error("GetFuturesRecentCompletedOrders() error", err)
	}
}

func TestGetFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOrderDetails(context.Background(), "5cdfc138b21023a909e5ad55")
	if err != nil {
		t.Error("GetFuturesOrderDetails() error", err)
	}
}

func TestGetFuturesOrderDetailsByClientID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOrderDetailsByClientID(context.Background(), "eresc138b21023a909e5ad59")
	if err != nil {
		t.Error("GetFuturesOrderDetailsByClientID() error", err)
	}
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesFills(context.Background(), "", "", "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesFills() error", err)
	}
}

func TestGetFuturesRecentFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesRecentFills(context.Background())
	if err != nil {
		t.Error("GetFuturesRecentFills() error", err)
	}
}

func TestGetFuturesOpenOrderStats(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOpenOrderStats(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("GetFuturesOpenOrderStats() error", err)
	}
}

func TestGetFuturesPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesPosition(context.Background(), "XBTUSDM")
	if err != nil {
		t.Error("GetFuturesPosition() error", err)
	}
}

func TestGetFuturesPositionList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesPositionList(context.Background())
	if err != nil {
		t.Error("GetFuturesPositionList() error", err)
	}
}

func TestSetAutoDepositMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SetAutoDepositMargin(context.Background(), "ADAUSDTM", true)
	if err != nil {
		t.Error("SetAutoDepositMargin() error", err)
	}
}

func TestAddMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.AddMargin(context.Background(), "XBTUSDTM", "6200c9b83aecfb000152dasfdee", 1)
	if err != nil {
		t.Error("AddMargin() error", err)
	}
}

func TestGetFuturesRiskLimitLevel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetFuturesRiskLimitLevel(context.Background(), "ADAUSDTM")
	if err != nil {
		t.Error("GetFuturesRiskLimitLevel() error", err)
	}
}

func TestUpdateRiskLmitLevel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.FuturesUpdateRiskLmitLevel(context.Background(), "ADASUDTM", 2)
	if err != nil {
		t.Error("UpdateRiskLmitLevel() error", err)
	}
}

func TestGetFuturesFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesFundingHistory(context.Background(), futuresTradablePair.String(), 0, 0, true, true, time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesFundingHistory() error", err)
	}
}

func TestGetFuturesAccountOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesAccountOverview(context.Background(), "")
	if err != nil {
		t.Error("GetFuturesAccountOverview() error", err)
	}
}

func TestGetFuturesTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesTransactionHistory(context.Background(), "", "", 0, 0, true, time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesTransactionHistory() error", err)
	}
}

func TestCreateFuturesSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CreateFuturesSubAccountAPIKey(context.Background(), "", "passphrase", "", "remark", "subAccName")
	if err != nil {
		t.Error("CreateFuturesSubAccountAPIKey() error", err)
	}
}

func TestGetFuturesDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesDepositAddress(context.Background(), "XBT")
	if err != nil {
		t.Error("GetFuturesDepositAddress() error", err)
	}
}

func TestGetFuturesDepositsList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesDepositsList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesDepositsList() error", err)
	}
}

func TestGetFuturesWithdrawalLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesWithdrawalLimit(context.Background(), "XBT")
	if err != nil {
		t.Error("GetFuturesWithdrawalLimit() error", err)
	}
}

func TestGetFuturesWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesWithdrawalList() error", err)
	}
}

func TestCancelFuturesWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelFuturesWithdrawal(context.Background(), "5cda659603aa67131f305f7e")
	if err != nil {
		t.Error("CancelFuturesWithdrawal() error", err)
	}
}

func TestTransferFuturesFundsToMainAccount(t *testing.T) {
	t.Parallel()
	var resp *TransferRes
	err := json.Unmarshal([]byte(transferFuturesFundsResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err = ku.TransferFuturesFundsToMainAccount(context.Background(), 1, "USDT", "MAIN")
	if err != nil {
		t.Error("TransferFuturesFundsToMainAccount() error", err)
	}
}

func TestTransferFundsToFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.TransferFundsToFuturesAccount(context.Background(), 1, "USDT", "MAIN")
	if err != nil {
		t.Error("TransferFundsToFuturesAccount() error", err)
	}
}

func TestGetFuturesTransferOutList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesTransferOutList(context.Background(), "USDT", "", time.Time{}, time.Time{})
	if err != nil {
		t.Error("GetFuturesTransferOutList() error", err)
	}
}

func TestCancelFuturesTransferOut(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
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
	_, err = ku.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = ku.FetchTradablePairs(context.Background(), asset.Margin)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := ku.UpdateOrderbook(context.Background(), futuresTradablePair, asset.Futures); err != nil {
		t.Error(err)
	}
	if _, err := ku.UpdateOrderbook(context.Background(), marginTradablePair, asset.Margin); err != nil {
		t.Error(err)
	}
	if _, err := ku.UpdateOrderbook(context.Background(), spotTradablePair, asset.Spot); err != nil {
		t.Error(err)
	}
}
func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := ku.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	err = ku.UpdateTickers(context.Background(), asset.Margin)
	if err != nil {
		t.Fatal(err)
	}
	err = ku.UpdateTickers(context.Background(), asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	err = ku.UpdateTickers(context.Background(), asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatal(err)
	}
}
func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	var err error
	_, err = ku.UpdateTicker(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.UpdateTicker(context.Background(), marginTradablePair, asset.Margin)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.UpdateTicker(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := ku.FetchTicker(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ku.FetchTicker(context.Background(), marginTradablePair, asset.Margin); err != nil {
		t.Error(err)
	}
	if _, err = ku.FetchTicker(context.Background(), futuresTradablePair, asset.Futures); err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := ku.FetchOrderbook(context.Background(), spotTradablePair, asset.Spot); err != nil {
		t.Error(err)
	}
	if _, err := ku.FetchOrderbook(context.Background(), marginTradablePair, asset.Margin); err != nil {
		t.Error(err)
	}
	if _, err := ku.FetchOrderbook(context.Background(), futuresTradablePair, asset.Futures); err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 48)
	endTime := time.Now().Add(-time.Hour * 3)
	var err error
	_, err = ku.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.OneHour, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.OneHour, startTime, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetHistoricCandles(context.Background(), marginTradablePair, asset.Margin, kline.OneHour, startTime, time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 48)
	endTime := time.Now().Add(-time.Hour * 1)
	var err error
	_, err = ku.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.OneHour, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	if err != nil {
		t.Error(err)
	}
	_, err = ku.GetHistoricCandlesExtended(context.Background(), marginTradablePair, asset.Margin, kline.OneHour, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, startTime, endTime)
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
	_, err = ku.GetServerTime(context.Background(), asset.Margin)
	if err != nil {
		t.Error(err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := ku.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	if err != nil {
		t.Error(err)
	}
	_, err = ku.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = ku.GetRecentTrades(context.Background(), marginTradablePair, asset.Margin)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var enabledPairs currency.Pairs
	var getOrdersRequest order.MultiOrderRequest
	var err error
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     append([]currency.Pair{currency.NewPair(currency.BTC, currency.USDT)}, enabledPairs[:3]...),
		AssetType: asset.Futures,
		Side:      order.AnySide,
	}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     []currency.Pair{spotTradablePair},
		AssetType: asset.Spot,
		Side:      order.Sell,
	}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.AssetType = asset.Margin
	getOrdersRequest.Pairs = currency.Pairs{marginTradablePair}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var getOrdersRequest order.MultiOrderRequest
	var enabledPairs currency.Pairs
	var err error
	enabledPairs, err = ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     enabledPairs,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Pairs = []currency.Pair{}
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Type = order.Market
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Type = order.OCO
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); !errors.Is(err, order.ErrUnsupportedOrderType) {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	enabledPairs, err = ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     enabledPairs,
		AssetType: asset.Margin,
		Side:      order.Buy,
	}
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Pairs = []currency.Pair{}
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Type = order.Market
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Type = order.OCO
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); !errors.Is(err, order.ErrUnsupportedOrderType) {
		t.Errorf("expected %v, but found %v", order.ErrUnsupportedOrderType, err)
	}
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	getOrdersRequest = order.MultiOrderRequest{
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
	getOrdersRequest.Type = order.StopLimit
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Kucoin GetActiveOrders() error", err)
	}
	getOrdersRequest.Type = order.OCO
	if _, err = ku.GetActiveOrders(context.Background(), &getOrdersRequest); !errors.Is(err, order.ErrUnsupportedOrderType) {
		t.Errorf("expected %v, but found %v", order.ErrUnsupportedOrderType, err)
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}); err != nil {
		t.Error(err)
	}
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	assetTypes := ku.CurrencyPairs.GetAssetTypes(true)
	for _, at := range assetTypes {
		if err := ku.ValidateCredentials(context.Background(), at); err != nil {
			t.Error(err)
		}
	}
}

func TestGetInstanceServers(t *testing.T) {
	t.Parallel()
	if _, err := ku.GetInstanceServers(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetAuthenticatedServersInstances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAuthenticatedInstanceServers(context.Background())
	if err != nil {
		t.Error(err)
	}
}

var websocketPushDatas = map[string]string{
	"SymbolTickerPushDataJSON":                                   `{"type": "message","topic": "/market/ticker:FET-BTC","subject": "trade.ticker","data": {"bestAsk": "0.000018679","bestAskSize": "258.4609","bestBid": "0.000018622","bestBidSize": "68.5961","price": "0.000018628","sequence": "38509148","size": "8.943","time": 1677321643926}}`,
	"AllSymbolsTickerPushDataJSON":                               `{"type": "message","topic": "/market/ticker:all","subject": "FTM-ETH","data": {"bestAsk": "0.0002901","bestAskSize": "3514.4978","bestBid": "0.0002894","bestBidSize": "65.536","price": "0.0002894","sequence": "186911324","size": "150","time": 1677320967673}}`,
	"MarketTradeSnapshotPushDataJSON":                            `{"type": "message","topic": "/market/snapshot:BTC","subject": "trade.snapshot","data": {"sequence": "5701753771","data": {"averagePrice": 21736.73225440,"baseCurrency": "BTC","board": 1,"buy": 21423,"changePrice": -556.80000000000000000000,"changeRate": -0.0253,"close": 21423.1,"datetime": 1676310802092,"high": 22030.70000000000000000000,"lastTradedPrice": 21423.1,"low": 21407.00000000000000000000,"makerCoefficient": 1.000000,"makerFeeRate": 0.001,"marginTrade": true,"mark": 0,"market": "USDS","markets": ["USDS"],"open": 21979.90000000000000000000,"quoteCurrency": "USDT","sell": 21423.1,"sort": 100,"symbol": "BTC-USDT","symbolCode": "BTC-USDT","takerCoefficient": 1.000000,"takerFeeRate": 0.001,"trading": true,"vol": 6179.80570155000000000000,"volValue": 133988049.45570351500000000000}}}`,
	"Orderbook Level 2 PushDataJSON":                             `{"type": "message","topic": "/spotMarket/level2Depth5:ETH-USDT","subject": "level2","data": {"asks": [[	"21612.7",	"0.32307467"],[	"21613.1",	"0.1581911"],[	"21613.2",	"1.37156153"],[	"21613.3",	"2.58327302"],[	"21613.4",	"0.00302088"]],"bids": [[	"21612.6",	"2.34316818"],[	"21612.3",	"0.5771615"],[	"21612.2",	"0.21605964"],[	"21612.1",	"0.22894841"],[	"21611.6",	"0.29251003"]],"timestamp": 1676319909635}}`,
	"TradeCandlesUpdatePushDataJSON":                             `{"type":"message","topic":"/market/candles:BTC-USDT_1hour","subject":"trade.candles.update","data":{"symbol":"BTC-USDT","candles":["1589968800","9786.9","9740.8","9806.1","9732","27.45649579","268280.09830877"],"time":1589970010253893337}}`,
	"SymbolSnapshotPushDataJSON":                                 `{"type": "message","topic": "/market/snapshot:KCS-BTC","subject": "trade.snapshot","data": {"sequence": "1545896669291","data": {"trading": true,"symbol": "KCS-BTC","buy": 0.00011,"sell": 0.00012,            "sort": 100,            "volValue": 3.13851792584,            "baseCurrency": "KCS",            "market": "BTC",            "quoteCurrency": "BTC",            "symbolCode": "KCS-BTC",            "datetime": 1548388122031,            "high": 0.00013,            "vol": 27514.34842,            "low": 0.0001,            "changePrice": -1.0e-5,            "changeRate": -0.0769,            "lastTradedPrice": 0.00012,            "board": 0,            "mark": 0        }    }}`,
	"MatchExecutionPushDataJSON":                                 `{"type":"message","topic":"/market/match:BTC-USDT","subject":"trade.l3match","data":{"sequence":"1545896669145","type":"match","symbol":"BTC-USDT","side":"buy","price":"0.08200000000000000000","size":"0.01022222000000000000","tradeId":"5c24c5da03aa673885cd67aa","takerOrderId":"5c24c5d903aa6772d55b371e","makerOrderId":"5c2187d003aa677bd09d5c93","time":"1545913818099033203"}}`,
	"IndexPricePushDataJSON":                                     `{"id":"","type":"message","topic":"/indicator/index:USDT-BTC","subject":"tick","data":{"symbol": "USDT-BTC","granularity": 5000,"timestamp": 1551770400000,"value": 0.0001092}}`,
	"MarkPricePushDataJSON":                                      `{"type":"message","topic":"/indicator/markPrice:USDT-BTC","subject":"tick","data":{"symbol": "USDT-BTC","granularity": 5000,"timestamp": 1551770400000,"value": 0.0001093}}`,
	"Orderbook ChangePushDataJSON":                               `{"type":"message","topic":"/margin/fundingBook:USDT","subject":"funding.update","data":{"annualIntRate":0.0547,"currency":"USDT","dailyIntRate":0.00015,"sequence":87611418,"side":"lend","size":25040,"term":7,"ts":1671005721087508735}}`,
	"Order ChangeStateOpenPushDataJSON":                          `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"buy","orderId":"5efab07953bdea00089965d2","type":"open","orderTime":1593487481683297666,"size":"0.1","filledSize":"0","price":"0.937","clientOid":"1593487481000906","remainSize":"0.1","status":"open","ts":1593487481683297666}}`,
	"Order ChangeStateMatchPushDataJSON":                         `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"sell","orderId":"5efab07953bdea00089965fa","liquidity":"taker","type":"match","orderTime":1593487482038606180,"size":"0.1","filledSize":"0.1","price":"0.938","matchPrice":"0.96738","matchSize":"0.1","tradeId":"5efab07a4ee4c7000a82d6d9","clientOid":"1593487481000313","remainSize":"0","status":"match","ts":1593487482038606180}}`,
	"Order ChangeStateFilledPushDataJSON":                        `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"sell","orderId":"5efab07953bdea00089965fa","type":"filled","orderTime":1593487482038606180,"size":"0.1","filledSize":"0.1","price":"0.938","clientOid":"1593487481000313","remainSize":"0","status":"done","ts":1593487482038606180}}`,
	"Order ChangeStateCancelledPushDataJSON":                     `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"buy","orderId":"5efab07953bdea00089965d2","type":"canceled","orderTime":1593487481683297666,"size":"0.1","filledSize":"0","price":"0.937","clientOid":"1593487481000906","remainSize":"0","status":"done","ts":1593487481893140844}}`,
	"Order ChangeStateUpdatePushDataJSON":                        `{"type":"message","topic":"/spotMarket/tradeOrders","subject":"orderChange","channelType":"private","data":{"symbol":"KCS-USDT","orderType":"limit","side":"buy","orderId":"5efab13f53bdea00089971df","type":"update","oldSize":"0.1","orderTime":1593487679693183319,"size":"0.06","filledSize":"0","price":"0.937","clientOid":"1593487679000249","remainSize":"0.06","status":"open","ts":1593487682916117521}}`,
	"AccountBalanceNoticePushDataJSON":                           `{"type": "message","topic": "/account/balance","subject": "account.balance","channelType":"private","data": {"total": "88","available": "88","availableChange": "88","currency": "KCS","hold": "0","holdChange": "0","relationEvent": "trade.hold","relationEventId": "5c21e80303aa677bd09d7dff","relationContext": {"symbol":"BTC-USDT","tradeId":"5e6a5dca9e16882a7d83b7a4","orderId":"5ea10479415e2f0009949d54"},"time": "1545743136994"}}`,
	"DebtRatioChangePushDataJSON":                                `{"type":"message","topic":"/margin/position","subject":"debt.ratio","channelType":"private","data": {"debtRatio": 0.7505,"totalDebt": "21.7505","debtList": {"BTC": "1.21","USDT": "2121.2121","EOS": "0"},"timestamp": 1553846081210}}`,
	"PositionStatusChangeEventPushDataJSON":                      `{"type":"message","topic":"/margin/position","subject":"position.status","channelType":"private","data": {"type": "FROZEN_FL","timestamp": 1553846081210}}`,
	"MarginTradeOrderEntersEventPushDataJSON":                    `{"type": "message","topic": "/margin/loan:BTC","subject": "order.open","channelType":"private","data": {    "currency": "BTC",    "orderId": "ac928c66ca53498f9c13a127a60e8",    "dailyIntRate": 0.0001,    "term": 7,    "size": 1,        "side": "lend",    "ts": 1553846081210004941}}`,
	"MarginTradeOrderUpdateEventPushDataJSON":                    `{"type": "message","topic": "/margin/loan:BTC","subject": "order.update","channelType":"private","data": {    "currency": "BTC",    "orderId": "ac928c66ca53498f9c13a127a60e8",    "dailyIntRate": 0.0001,    "term": 7,    "size": 1,    "lentSize": 0.5,    "side": "lend",    "ts": 1553846081210004941}}`,
	"MarginTradeOrderDoneEventPushDataJSON":                      `{"type": "message","topic": "/margin/loan:BTC","subject": "order.done","channelType":"private","data": {    "currency": "BTC",    "orderId": "ac928c66ca53498f9c13a127a60e8",    "reason": "filled",    "side": "lend",    "ts": 1553846081210004941  }}`,
	"StopOrderEventPushDataJSON":                                 `{"type":"message","topic":"/spotMarket/advancedOrders","subject":"stopOrder","channelType":"private","data":{"createdAt":1589789942337,"orderId":"5ec244f6a8a75e0009958237","orderPrice":"0.00062","orderType":"stop","side":"sell","size":"1","stop":"entry","stopPrice":"0.00062","symbol":"KCS-BTC","tradeType":"TRADE","triggerSuccess":true,"ts":1589790121382281286,"type":"triggered"}}`,
	"Public Futures TickerPushDataJSON":                          `{"subject": "tickerV2","topic": "/contractMarket/tickerV2:ETHUSDCM","data": {"symbol": "ETHUSDCM","bestBidSize": 795,"bestBidPrice": 3200.00,"bestAskPrice": 3600.00,"bestAskSize": 284,"ts": 1553846081210004941}}`,
	"Public Futures TickerV1PushDataJSON":                        `{"subject": "ticker","topic": "/contractMarket/ticker:ETHUSDCM","data": {"symbol": "ETHUSDCM","sequence": 45,"side": "sell","price": 3600.00,"size": 16,"tradeId": "5c9dcf4170744d6f5a3d32fb","bestBidSize": 795,"bestBidPrice": 3200.00,"bestAskPrice": 3600.00,"bestAskSize": 284,"ts": 1553846081210004941}}`,
	"Public Futures Level2OrderbookPushDataJSON":                 `{"subject": "level2",  "topic": "/contractMarket/level2:ETHUSDCM",  "type": "message",  "data": {    "sequence": 18,    "change": "5000.0,sell,83","timestamp": 1551770400000}}`,
	"Public Futures ExecutionDataJSON":                           `{"type": "message","topic": "/contractMarket/execution:ETHUSDCM","subject": "match","data": {"makerUserId": "6287c3015c27f000017d0c2f","symbol": "ETHUSDCM","sequence": 31443494,"side": "buy","size": 35,"price": 23083.00000000,"takerOrderId": "63f94040839d00000193264b","makerOrderId": "63f94036839d0000019310c3","takerUserId": "6133f817230d8d000607b941","tradeId": "63f940400000650065f4996f","ts": 1677279296134648869}}`,
	"PublicFuturesOrderbookWithDepth5PushDataJSON":               `{ "type": "message", "topic": "/contractMarket/level2Depth5:ETHUSDCM", "subject": "level2", "data": { "sequence": 1672332328701, "asks": [[	23149,	13703],[	23150,	1460],[	23151.00000000,	941],[	23152,	4591],[	23153,	4107] ], "bids": [[	23148.00000000,	22801],[23147.0,4766],[	23146,	1388],[	23145.00000000,	2593],[	23144.00000000,	6286] ], "ts": 1677280435684, "timestamp": 1677280435684 }}`,
	"Private PositionSettlementPushDataJSON":                     `{"userId": "xbc453tg732eba53a88ggyt8c","topic": "/contract/position:ETHUSDCM","subject": "position.settlement","data": {"fundingTime": 1551770400000,"qty": 100,"markPrice": 3610.85,"fundingRate": -0.002966,"fundingFee": -296,"ts": 1547697294838004923,"settleCurrency": "XBT"}}`,
	"Futures PositionChangePushDataJSON":                         `{ "userId": "5cd3f1a7b7ebc19ae9558591","topic": "/contract/position:ETHUSDCM",  "subject": "position.change", "data": {"markPrice": 7947.83,"markValue": 0.00251640,"maintMargin": 0.00252044,"realLeverage": 10.06,"unrealisedPnl": -0.00014735,"unrealisedRoePcnt": -0.0553,"unrealisedPnlPcnt": -0.0553,"delevPercentage": 0.52,"currentTimestamp": 1558087175068,"settleCurrency": "XBT"}}`,
	"Futures PositionChangeWithChangeReasonPushDataJSON":         `{ "type": "message","userId": "5c32d69203aa676ce4b543c7","channelType": "private","topic": "/contract/position:ETHUSDCM",  "subject": "position.change", "data": {"realisedGrossPnl": 0E-8,"symbol":"ETHUSDCM","crossMode": false,"liquidationPrice": 1000000.0,"posLoss": 0E-8,"avgEntryPrice": 7508.22,"unrealisedPnl": -0.00014735,"markPrice": 7947.83,"posMargin": 0.00266779,"autoDeposit": false,"riskLimit": 100000,"unrealisedCost": 0.00266375,"posComm": 0.00000392,"posMaint": 0.00001724,"posCost": 0.00266375,"maintMarginReq": 0.005,"bankruptPrice": 1000000.0,"realisedCost": 0.00000271,"markValue": 0.00251640,"posInit": 0.00266375,"realisedPnl": -0.00000253,"maintMargin": 0.00252044,"realLeverage": 1.06,"changeReason": "positionChange","currentCost": 0.00266375,"openingTimestamp": 1558433191000,"currentQty": -20,"delevPercentage": 0.52,"currentComm": 0.00000271,"realisedGrossCost": 0E-8,"isOpen": true,"posCross": 1.2E-7,"currentTimestamp": 1558506060394,"unrealisedRoePcnt": -0.0553,"unrealisedPnlPcnt": -0.0553,"settleCurrency": "XBT"}}`,
	"Futures WithdrawalAmountTransferOutAmountEventPushDataJSON": `{ "userId": "xbc453tg732eba53a88ggyt8c","topic": "/contractAccount/wallet","subject": "withdrawHold.change","data": {"withdrawHold": 5923,"currency":"USDT","timestamp": 1553842862614}}`,
	"Futures AvailableBalanceChangePushData":                     `{ "userId": "xbc453tg732eba53a88ggyt8c","topic": "/contractAccount/wallet","subject": "availableBalance.change","data": {"availableBalance": 5923,"holdBalance": 2312,"currency":"USDT","timestamp": 1553842862614}}`,
	"Futures OrderMarginChangePushDataJSON":                      `{ "userId": "xbc453tg732eba53a88ggyt8c","topic": "/contractAccount/wallet","subject": "orderMargin.change","data": {"orderMargin": 5923,"currency":"USDT","timestamp": 1553842862614}}`,
	"Futures StopOrderPushDataJSON":                              `{"userId": "5cd3f1a7b7ebc19ae9558591","topic": "/contractMarket/advancedOrders", "subject": "stopOrder","data": {"orderId": "5cdfc138b21023a909e5ad55","symbol": "ETHUSDCM","type": "open","orderType":"stop","side":"buy","size":"1000","orderPrice":"9000","stop":"up","stopPrice":"9100","stopPriceType":"TP","triggerSuccess": true,"error": "error.createOrder.accountBalanceInsufficient","createdAt": 1558074652423,"ts":1558074652423004000}}`,
	"Futures TradeOrdersPushDataJSON":                            `{"type": "message","topic": "/contractMarket/tradeOrders","subject": "orderChange","channelType": "private","data": {"orderId": "5cdfc138b21023a909e5ad55","symbol": "ETHUSDCM","type": "match","status": "open","matchSize": "","matchPrice": "","orderType": "limit","side": "buy","price": "3600","size": "20000","remainSize": "20001","filledSize":"20000","canceledSize": "0","tradeId": "5ce24c16b210233c36eexxxx","clientOid": "5ce24c16b210233c36ee321d","orderTime": 1545914149935808589,"oldSize ": "15000","liquidity": "maker","ts": 1545914149935808589}}`,
	"TransactionStaticsPushDataJSON":                             `{ "topic": "/contractMarket/snapshot:ETHUSDCM","subject": "snapshot.24h","data": {"volume": 30449670,      "turnover": 845169919063,"lastPrice": 3551,       "priceChgPct": 0.0043,   "ts": 1547697294838004923}  }`,
	"Futures EndFundingFeeSettlementPushDataJSON":                `{ "type":"message","topic": "/contract/announcement","subject": "funding.end","data": {"symbol": "ETHUSDCM",         "fundingTime": 1551770400000,"fundingRate": -0.002966,    "timestamp": 1551770410000          }}`,
	"Futures StartFundingFeeSettlementPushDataJSON":              `{ "topic": "/contract/announcement","subject": "funding.begin","data": {"symbol": "ETHUSDCM","fundingTime": 1551770400000,"fundingRate": -0.002966,"timestamp": 1551770400000}}`,
	"Futures FundingRatePushDataJSON":                            `{ "topic": "/contract/instrument:ETHUSDCM","subject": "funding.rate","data": {"granularity": 60000,"fundingRate": -0.002966,"timestamp": 1551770400000}}`,
	"Futures MarkIndexPricePushDataJSON":                         `{ "topic": "/contract/instrument:ETHUSDCM","subject": "mark.index.price","data": {"granularity": 1000,"indexPrice": 4000.23,"markPrice": 4010.52,"timestamp": 1551770400000}}`,
	"Orderbook Market Level2":                                    `{ "type": "message", "topic": "/market/level2:BTC-USDT", "subject": "trade.l2update", "data": { "changes": { "asks": [ [ "18906", "0.00331", "14103845" ], [ "18907.3", "0.58751503", "14103844" ] ], "bids": [ [ "18891.9", "0.15688", "14103847" ] ] }, "sequenceEnd": 14103847, "sequenceStart": 14103844, "symbol": "BTC-USDT", "time": 1663747970273 } }`,
}

func TestPushData(t *testing.T) {
	for key, val := range websocketPushDatas {
		err := ku.wsHandleData([]byte(val))
		if err != nil {
			t.Errorf("%s: %v", key, err)
		}
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetAvailableTransferChains(context.Background(), currency.BTC); err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Futures); err != nil {
		t.Error(err)
	}
	if _, err := ku.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot); err != nil {
		t.Error(err)
	}
	if _, err := ku.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Margin); !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var err error
	_, err = ku.GetOrderInfo(context.Background(), "123", futuresTradablePair, asset.Futures)
	if err != nil {
		t.Errorf("expected %s, but found %v", "Order does not exist", err)
	}
	_, err = ku.GetOrderInfo(context.Background(), "123", futuresTradablePair, asset.Spot)
	if err != nil {
		t.Errorf("expected %s, but found %v", "Order does not exist", err)
	}
	_, err = ku.GetOrderInfo(context.Background(), "123", futuresTradablePair, asset.Margin)
	if err != nil {
		t.Errorf("expected %s, but found %v", "Order does not exist", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetDepositAddress(context.Background(), currency.BTC, "", ""); err != nil && !errors.Is(err, errNoDepositAddress) {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	withdrawCryptoRequest := withdraw.Request{
		Exchange: ku.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}
	if _, err := ku.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest); err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	orderSubmission := &order.Submit{
		Pair:          spotTradablePair,
		Exchange:      ku.Name,
		Side:          order.Bid,
		Type:          order.Limit,
		Price:         1,
		Amount:        100000,
		ClientOrderID: "myOrder",
		AssetType:     asset.Spot,
	}
	_, err := ku.SubmitOrder(context.Background(), orderSubmission)
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("expected %v, but found %v", asset.ErrNotSupported, err)
	}
	orderSubmission.Side = order.Buy
	orderSubmission.AssetType = asset.Options
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("expected %v, but found %v", asset.ErrNotSupported, err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	orderSubmission.AssetType = asset.Spot
	orderSubmission.Side = order.Buy
	orderSubmission.Pair = spotTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	if err != order.ErrTypeIsInvalid {
		t.Errorf("expected %v, but found %v", order.ErrTypeIsInvalid, err)
	}
	orderSubmission.AssetType = asset.Spot
	orderSubmission.Pair = spotTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
	orderSubmission.AssetType = asset.Margin
	orderSubmission.Pair = marginTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
	orderSubmission.AssetType = asset.Margin
	orderSubmission.Pair = marginTradablePair
	orderSubmission.MarginType = margin.Isolated
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
	orderSubmission.AssetType = asset.Futures
	orderSubmission.Pair = futuresTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	if !errors.Is(err, errInvalidLeverage) {
		t.Error(err)
	}
	orderSubmission.Leverage = 0.01
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          spotTradablePair,
		AssetType:     asset.Spot,
	}
	if err := ku.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = marginTradablePair
	orderCancellation.AssetType = asset.Margin
	if err := ku.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
	orderCancellation.Pair = futuresTradablePair
	orderCancellation.AssetType = asset.Futures
	if err := ku.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	if _, err := ku.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType:  asset.Futures,
		MarginType: margin.Isolated,
	}); err != nil {
		t.Error(err)
	}
	if _, err := ku.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType:  asset.Margin,
		MarginType: margin.Isolated,
	}); err != nil {
		t.Error(err)
	}
	if _, err := ku.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType:  asset.Spot,
		MarginType: margin.Isolated,
	}); err != nil {
		t.Error(err)
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
		t.Error("derived payload is not same as generated channel subscription instances")
	}
}

const (
	subUserResponseJSON              = `{"userId":"635002438793b80001dcc8b3", "uid":62356, "subName":"margin01", "status":2, "type":4, "access":"Margin", "createdAt":1666187844000, "remarks":null }`
	positionSettlementPushData       = `{"userId": "xbc453tg732eba53a88ggyt8c", "topic": "/contract/position:XBTUSDM", "subject": "position.settlement", "data": { "fundingTime": 1551770400000, "qty": 100, "markPrice": 3610.85, "fundingRate": -0.002966, "fundingFee": -296, "ts": 1547697294838004923, "settleCurrency": "XBT" } }`
	transferFuturesFundsResponseJSON = `{"applyId": "620a0bbefeaa6a000110e833", "bizNo": "620a0bbefeaa6a000110e832", "payAccountType": "CONTRACT", "payTag": "DEFAULT", "remark": "", "recAccountType": "MAIN", "recTag": "DEFAULT", "recRemark": "", "recSystem": "KUCOIN", "status": "PROCESSING", "currency": "USDT", "amount": "0.001", "fee": "0", "sn": 889048787670001, "reason": "", "createdAt": 1644825534000, "updatedAt": 1644825534000 }`
	modifySubAccountSpotAPIs         = `{"subName": "AAAAAAAAAA0007", "remark": "remark", "apiKey": "630325e0e750870001829864", "apiSecret": "110f31fc-61c5-4baf-a29f-3f19a62bbf5d", "passphrase": "passphrase", "permission": "General", "ipWhitelist": "", "createdAt": 1661150688000 }`
)

func TestCreateSubUser(t *testing.T) {
	t.Parallel()
	var resp *SubAccount
	err := json.Unmarshal([]byte(subUserResponseJSON), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	if _, err := ku.CreateSubUser(context.Background(), "SamuaelTee1", "sdfajdlkad", "", ""); err != nil {
		t.Error(err)
	}
}

func TestGetSubAccountSpotAPIList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetSubAccountSpotAPIList(context.Background(), "sam", ""); err != nil {
		t.Error(err)
	}
}

func TestCreateSpotAPIsForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
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
	var resp SpotAPISubAccount
	err := json.Unmarshal([]byte(modifySubAccountSpotAPIs), &resp)
	if err != nil {
		t.Fatal(err)
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	if _, err := ku.ModifySubAccountSpotAPIs(context.Background(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		Remark:         "123456",
	}); err != nil {
		t.Error(err)
	}
}

func TestDeleteSubAccountSpotAPI(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	if _, err := ku.DeleteSubAccountSpotAPI(context.Background(), apiKey, "mysecretPassphrase123", "gocryptoTrader1"); err != nil {
		t.Error(err)
	}
}

func TestGetUserInfoOfAllSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetUserInfoOfAllSubAccounts(context.Background()); err != nil {
		t.Error(err)
	}
}

func TestGetPaginatedListOfSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	if _, err := ku.GetPaginatedListOfSubAccounts(context.Background(), 1, 100); err != nil {
		t.Error(err)
	}
}

func setupWS() {
	if !ku.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(ku) {
		ku.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := ku.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAccountFundingHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func getFirstTradablePairOfAssets() {
	if err := ku.UpdateTradablePairs(context.Background(), true); err != nil {
		log.Fatalf("Kucoin error while updating tradable pairs. %v", err)
	}
	enabledPairs, err := ku.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Fatalf("Kucoin %v, trying to get %v enabled pairs error", err, asset.Spot)
	}
	spotTradablePair = enabledPairs[0]
	enabledPairs, err = ku.GetEnabledPairs(asset.Margin)
	if err != nil {
		log.Fatalf("Kucoin %v, trying to get %v enabled pairs error", err, asset.Margin)
	}
	marginTradablePair = enabledPairs[0]
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
	if err != nil {
		log.Fatalf("Kucoin %v, trying to get %v enabled pairs error", err, asset.Futures)
	}
	futuresTradablePair = enabledPairs[0]
	futuresTradablePair.Delimiter = ""
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var err error
	_, err = ku.FetchAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.FetchAccountInfo(context.Background(), asset.Margin)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ku.FetchAccountInfo(context.Background(), asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKucoinNumberUnmarshal(t *testing.T) {
	t.Parallel()
	data := &struct {
		Number kucoinNumber `json:"number"`
	}{}
	data1 := `{"number": 123.33}`
	err := json.Unmarshal([]byte(data1), &data)
	if err != nil {
		t.Fatal(err)
	} else if data.Number.Float64() != 123.33 {
		t.Errorf("expecting %.2f, got %.2f", 123.33, data.Number)
	}
	data2 := `{"number": "123.33"}`
	err = json.Unmarshal([]byte(data2), &data)
	if err != nil {
		t.Fatal(err)
	} else if data.Number.Float64() != 123.33 {
		t.Errorf("expecting %.2f, got %.2f", 123.33, data.Number)
	}
	data3 := `{"number": ""}`
	err = json.Unmarshal([]byte(data3), &data)
	if err != nil {
		t.Fatal(err)
	} else if data.Number.Float64() != 0 {
		t.Errorf("expecting %d, got %.2f", 0, data.Number)
	}
	data4 := `{"number": "123"}`
	err = json.Unmarshal([]byte(data4), &data)
	if err != nil {
		t.Fatal(err)
	} else if data.Number.Float64() != 123 {
		t.Errorf("expecting %d, got %.2f", 123, data.Number)
	}
	data5 := `{"number": 0}`
	err = json.Unmarshal([]byte(data5), &data)
	if err != nil {
		t.Fatal(err)
	} else if data.Number.Float64() != 0 {
		t.Errorf("expecting %d, got %.2f", 0, data.Number)
	}
	data6 := `{"number": 123789}`
	err = json.Unmarshal([]byte(data6), &data)
	if err != nil {
		t.Fatal(err)
	} else if data.Number.Float64() != 123789 {
		t.Errorf("expecting %d, got %.2f", 123789, data.Number)
	}
	data7 := `{"number": 12321312312312312}`
	err = json.Unmarshal([]byte(data7), &data)
	if err != nil {
		t.Fatal(err)
	} else if data.Number.Float64() != float64(12321312312312312) {
		t.Errorf("expecting %.f, got %.2f", float64(12321312312312312), data.Number)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.UpdateAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error("Kucoin UpdateAccountInfo() error", err)
	}
	_, err = ku.UpdateAccountInfo(context.Background(), asset.Futures)
	if err != nil {
		t.Error("Kucoin UpdateAccountInfo() error", err)
	}
	_, err = ku.UpdateAccountInfo(context.Background(), asset.Margin)
	if err != nil {
		t.Error("Kucoin UpdateAccountInfo() error", err)
	}
}

const (
	orderbookLevel5PushData = `{"type": "message","topic": "/spotMarket/level2Depth50:BTC-USDT","subject": "level2","data": {"asks": [["21621.7","3.03206193"],["21621.8","1.00048239"],["21621.9","0.29558803"],["21622","0.0049653"],["21622.4","0.06177582"],["21622.9","0.39664116"],["21623.7","0.00803466"],["21624.2","0.65405"],["21624.3","0.34661426"],["21624.6","0.00035589"],["21624.9","0.61282048"],["21625.2","0.16421424"],["21625.4","0.90107014"],["21625.5","0.73484442"],["21625.9","0.04"],["21626.2","0.28569324"],["21626.4","0.18403701"],["21627.1","0.06503999"],["21627.2","0.56105832"],["21627.7","0.10649999"],["21628.1","2.66459953"],["21628.2","0.32"],["21628.5","0.27605551"],["21628.6","1.59482596"],["21628.9","0.16"],["21629.8","0.08"],["21630","0.04"],["21631.6","0.1"],["21631.8","0.0920185"],["21633.6","0.00447983"],["21633.7","0.00015044"],["21634.3","0.32193346"],["21634.4","0.00004"],["21634.5","0.1"],["21634.6","0.0002865"],["21635.6","0.12069941"],["21635.8","0.00117158"],["21636","0.00072816"],["21636.5","0.98611492"],["21636.6","0.00007521"],["21637.2","0.00699999"],["21637.6","0.00017129"],["21638","0.00013035"],["21638.1","0.05"],["21638.5","0.92427"],["21639.2","1.84998696"],["21639.3","0.04827233"],["21640","0.56255996"],["21640.9","0.8"],["21641","0.12"]],"bids": [["21621.6","0.40949924"],["21621.5","0.27703279"],["21621.3","0.04"],["21621.1","0.0086"],["21621","0.6653104"],["21620.9","0.35435999"],["21620.8","0.37224309"],["21620.5","0.416184"],["21620.3","0.24"],["21619.6","0.13883999"],["21619.5","0.21053355"],["21618.7","0.2"],["21618.6","0.001"],["21618.5","0.2258151"],["21618.4","0.06503999"],["21618.3","0.00370056"],["21618","0.12067842"],["21617.7","0.34844131"],["21617.6","0.92845495"],["21617.5","0.66460535"],["21617","0.01"],["21616.7","0.0004624"],["21616.4","0.02"],["21615.6","0.04828251"],["21615","0.59065665"],["21614.4","0.00227"],["21614.3","0.1"],["21613","0.32193346"],["21612.9","0.0028638"],["21612.6","0.1"],["21612.5","0.92539"],["21610.7","0.08208616"],["21610.6","0.00967666"],["21610.3","0.12"],["21610.2","0.00611126"],["21609.9","0.00226344"],["21609.8","0.00315812"],["21609.1","0.00547218"],["21608.6","0.09793157"],["21608.5","0.00437793"],["21608.4","1.85013454"],["21608.1","0.00366647"],["21607.9","0.00611595"],["21607.7","0.83263561"],["21607.6","0.00368919"],["21607.5","0.00280702"],["21607.1","0.66610849"],["21606.8","0.00364164"],["21606.2","0.80351642"],["21605.7","0.075"]],"timestamp": 1676319280783}}`
	wsOrderbookData         = `{"changes":{"asks":[["21621.7","3.03206193",""],["21621.8","1.00048239",""],["21621.9","0.29558803",""],["21622","0.0049653",""],["21622.4","0.06177582",""],["21622.9","0.39664116",""],["21623.7","0.00803466",""],["21624.2","0.65405",""],["21624.3","0.34661426",""],["21624.6","0.00035589",""],["21624.9","0.61282048",""],["21625.2","0.16421424",""],["21625.4","0.90107014",""],["21625.5","0.73484442",""],["21625.9","0.04",""],["21626.2","0.28569324",""],["21626.4","0.18403701",""],["21627.1","0.06503999",""],["21627.2","0.56105832",""],["21627.7","0.10649999",""],["21628.1","2.66459953",""],["21628.2","0.32",""],["21628.5","0.27605551",""],["21628.6","1.59482596",""],["21628.9","0.16",""],["21629.8","0.08",""],["21630","0.04",""],["21631.6","0.1",""],["21631.8","0.0920185",""],["21633.6","0.00447983",""],["21633.7","0.00015044",""],["21634.3","0.32193346",""],["21634.4","0.00004",""],["21634.5","0.1",""],["21634.6","0.0002865",""],["21635.6","0.12069941",""],["21635.8","0.00117158",""],["21636","0.00072816",""],["21636.5","0.98611492",""],["21636.6","0.00007521",""],["21637.2","0.00699999",""],["21637.6","0.00017129",""],["21638","0.00013035",""],["21638.1","0.05",""],["21638.5","0.92427",""],["21639.2","1.84998696",""],["21639.3","0.04827233",""],["21640","0.56255996",""],["21640.9","0.8",""],["21641","0.12",""]],"bids":[["21621.6","0.40949924",""],["21621.5","0.27703279",""],["21621.3","0.04",""],["21621.1","0.0086",""],["21621","0.6653104",""],["21620.9","0.35435999",""],["21620.8","0.37224309",""],["21620.5","0.416184",""],["21620.3","0.24",""],["21619.6","0.13883999",""],["21619.5","0.21053355",""],["21618.7","0.2",""],["21618.6","0.001",""],["21618.5","0.2258151",""],["21618.4","0.06503999",""],["21618.3","0.00370056",""],["21618","0.12067842",""],["21617.7","0.34844131",""],["21617.6","0.92845495",""],["21617.5","0.66460535",""],["21617","0.01",""],["21616.7","0.0004624",""],["21616.4","0.02",""],["21615.6","0.04828251",""],["21615","0.59065665",""],["21614.4","0.00227",""],["21614.3","0.1",""],["21613","0.32193346",""],["21612.9","0.0028638",""],["21612.6","0.1",""],["21612.5","0.92539",""],["21610.7","0.08208616",""],["21610.6","0.00967666",""],["21610.3","0.12",""],["21610.2","0.00611126",""],["21609.9","0.00226344",""],["21609.8","0.00315812",""],["21609.1","0.00547218",""],["21608.6","0.09793157",""],["21608.5","0.00437793",""],["21608.4","1.85013454",""],["21608.1","0.00366647",""],["21607.9","0.00611595",""],["21607.7","0.83263561",""],["21607.6","0.00368919",""],["21607.5","0.00280702",""],["21607.1","0.66610849",""],["21606.8","0.00364164",""],["21606.2","0.80351642",""],["21605.7","0.075",""]]},"sequenceEnd":1676319280783,"sequenceStart":0,"symbol":"BTC-USDT","time":1676319280783}`
)

func TestProcessOrderbook(t *testing.T) {
	t.Parallel()
	response := &WsOrderbook{}
	err := json.Unmarshal([]byte(wsOrderbookData), &response)
	if err != nil {
		t.Error(err)
	}
	_, err = ku.UpdateLocalBuffer(response, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	err = ku.processOrderbook([]byte(orderbookLevel5PushData), "BTC-USDT")
	if err != nil {
		t.Error(err)
	}
	err = ku.wsHandleData([]byte(orderbookLevel5PushData))
	if err != nil {
		t.Error(err)
	}
}

func TestSeedLocalCache(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("ETH-USDT")
	if err != nil {
		t.Error(err)
	}
	err = ku.SeedLocalCache(context.Background(), pair, asset.Margin)
	if err != nil {
		t.Error(err)
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesContractDetails(context.Background(), asset.Spot)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	_, err = ku.GetFuturesContractDetails(context.Background(), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = ku.GetFuturesContractDetails(context.Background(), asset.Futures)
	assert.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := ku.GetLatestFundingRates(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.BTC, currency.USD),
	}
	_, err = ku.GetLatestFundingRates(context.Background(), req)
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	req = &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.NewPair(currency.XBT, currency.USDTM),
	}
	resp, err := ku.GetLatestFundingRates(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, resp, 1)

	req = &fundingrate.LatestRateRequest{
		Asset: asset.Futures,
		Pair:  currency.EMPTYPAIR,
	}
	resp, err = ku.GetLatestFundingRates(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := ku.IsPerpetualFutureCurrency(asset.Spot, currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.False(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.EMPTYPAIR)
	assert.NoError(t, err)
	assert.False(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.NewPair(currency.XBT, currency.EOS))
	assert.NoError(t, err)
	assert.False(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.NewPair(currency.XBT, currency.USDTM))
	assert.NoError(t, err)
	assert.True(t, is)
	is, err = ku.IsPerpetualFutureCurrency(asset.Futures, currency.NewPair(currency.XBT, currency.USDM))
	assert.NoError(t, err)
	assert.True(t, is)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	_, err := ku.ChangePositionMargin(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &margin.PositionChangeRequest{}
	_, err = ku.ChangePositionMargin(context.Background(), req)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset)

	req.Asset = asset.Futures
	_, err = ku.ChangePositionMargin(context.Background(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pair = currency.NewPair(currency.XBT, currency.USDTM)
	_, err = ku.ChangePositionMargin(context.Background(), req)
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	req.MarginType = margin.Isolated
	_, err = ku.ChangePositionMargin(context.Background(), req)
	assert.Error(t, err)

	req.NewAllocatedMargin = 1337
	_, err = ku.ChangePositionMargin(context.Background(), req)
	assert.ErrorIs(t, err, nil)
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPositionSummary(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &futures.PositionSummaryRequest{}
	_, err = ku.GetFuturesPositionSummary(context.Background(), req)
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	req.Asset = asset.Futures
	_, err = ku.GetFuturesPositionSummary(context.Background(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	req.Pair = currency.NewPair(currency.XBT, currency.USDTM)
	_, err = ku.GetFuturesPositionSummary(context.Background(), req)
	assert.ErrorIs(t, err, nil)
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPositionOrders(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	req := &futures.PositionsRequest{}
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture)

	req.Asset = asset.Futures
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	req.Pairs = currency.Pairs{
		currency.NewPair(currency.XBT, currency.USDTM),
	}
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.ErrorIs(t, err, common.ErrDateUnset)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	req.EndDate = time.Now()
	req.StartDate = req.EndDate.Add(-time.Hour * 24 * 7)
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.ErrorIs(t, err, nil)

	req.StartDate = req.EndDate.Add(-time.Hour * 24 * 30)
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.ErrorIs(t, err, futures.ErrOrderHistoryTooLarge)

	req.RespectOrderHistoryLimits = true
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.ErrorIs(t, err, nil)
}
