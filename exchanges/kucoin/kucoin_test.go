package kucoin

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
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
	ku.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	ku.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	setupWS()
	getFirstTradablePairOfAssets()
	os.Exit(m.Run())
}

// Spot asset test cases starts from here
func TestGetSymbols(t *testing.T) {
	t.Parallel()
	symbols, err := ku.GetSymbols(context.Background(), "")
	if err != nil {
		t.Error("GetSymbols() error", err)
	}
	assert.NotEmpty(t, symbols, "should return all available spot/margin symbols")
	// Using market string reduces the scope of what is returned.
	symbols, err = ku.GetSymbols(context.Background(), "ETF")
	if err != nil {
		t.Error("GetSymbols() error", err)
	}
	assert.NotEmpty(t, symbols, "should return all available ETF symbols")
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

func TestGetFuturesTickers(t *testing.T) {
	t.Parallel()
	tickers, err := ku.GetFuturesTickers(context.Background())
	assert.NoError(t, err, "GetFuturesTickers should not error")
	for i := range tickers {
		assert.Positive(t, tickers[i].Last, "Last should be positive")
		assert.Positive(t, tickers[i].Bid, "Bid should be positive")
		assert.Positive(t, tickers[i].Ask, "Ask should be positive")
		assert.NotEmpty(t, tickers[i].Pair, "Pair should not be empty")
		assert.NotEmpty(t, tickers[i].LastUpdated, "LastUpdated should not be empty")
		assert.Equal(t, ku.Name, tickers[i].ExchangeName, "Exchange name should be correct")
		assert.Equal(t, asset.Futures, tickers[i].AssetType, "Asset type should be correct")
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

	// default order type is limit
	_, err := ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: ""})
	if !errors.Is(err, errInvalidClientOrderID) {
		t.Errorf("PostOrder() expected %v, but found %v", errInvalidClientOrderID, err)
	}

	customID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}

	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair,
		OrderType: ""})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Errorf("PostOrder() expected %v, but found %v", order.ErrSideIsInvalid, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("PostOrder() expected %v, but found %v", currency.ErrCurrencyPairEmpty, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: 0.1})
	if !errors.Is(err, errInvalidPrice) {
		t.Errorf("PostOrder() expected %v, but found %v", errInvalidPrice, err)
	}
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565})
	if !errors.Is(err, errInvalidSize) {
		t.Errorf("PostOrder() expected %v, but found %v", errInvalidSize, err)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(),
		Side:          "buy",
		Symbol:        spotTradablePair,
		OrderType:     "limit",
		Size:          0.005,
		Price:         1000})
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

	_, err := ku.GetTradingFee(context.Background(), nil)
	if !errors.Is(err, currency.ErrCurrencyPairsEmpty) {
		t.Fatalf("received %v, expected %v", err, currency.ErrCurrencyPairsEmpty)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	avail, err := ku.GetAvailablePairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	pairs := currency.Pairs{avail[0]}
	btcusdTradingFee, err := ku.GetTradingFee(context.Background(), pairs)
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}

	if len(btcusdTradingFee) != 1 {
		t.Error("GetTradingFee() error, expected 1 pair")
	}

	// NOTE: Test below will error out from an external call as this will exceed
	// the allowed pairs. If this does not error then this endpoint will allow
	// more items to be requested.
	pairs = append(pairs, avail[1:11]...)
	_, err = ku.GetTradingFee(context.Background(), pairs)
	if errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, "code: 200000 message: symbols size invalid.")
	}

	got, err := ku.GetTradingFee(context.Background(), pairs[:10])
	if !errors.Is(err, nil) {
		t.Fatalf("received %v, expected %v", err, nil)
	}

	if len(got) != 10 {
		t.Error("GetTradingFee() error, expected 10 pairs")
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

func TestGetFuturesTicker(t *testing.T) {
	t.Parallel()
	tick, err := ku.GetFuturesTicker(context.Background(), "XBTUSDTM")
	if assert.NoError(t, err, "GetFuturesTicker should not error") {
		assert.Positive(t, tick.Sequence, "Sequence should be positive")
		assert.Equal(t, "XBTUSDTM", tick.Symbol, "Symbol should be correct")
		assert.Contains(t, []order.Side{order.Buy, order.Sell}, tick.Side, "Side should be a side")
		assert.Positive(t, tick.Size, "Size should be positive")
		assert.Positive(t, tick.Price.Float64(), "Price should be positive")
		assert.Positive(t, tick.BestBidPrice.Float64(), "BestBidPrice should be positive")
		assert.Positive(t, tick.BestBidSize, "BestBidSize should be positive")
		assert.Positive(t, tick.BestAskPrice.Float64(), "BestAskPrice should be positive")
		assert.Positive(t, tick.BestAskSize, "BestAskSize should be positive")
		assert.NotEmpty(t, tick.TradeID, "TradeID should not be empty")
		assert.WithinRange(t, tick.FilledTime.Time(), time.Now().Add(time.Hour*-24), time.Now(), "FilledTime should be within last 24 hours")
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
	for _, a := range ku.GetAssetTypes(true) {
		err := ku.UpdateTickers(context.Background(), a)
		assert.NoError(t, err, "UpdateTickers should not error")
		pairs, err := ku.GetEnabledPairs(a)
		assert.NoError(t, err, "GetEnabledPairs should not error")
		for _, p := range pairs {
			tick, err := ticker.GetTicker(ku.Name, p, a)
			if assert.NoError(t, err, "GetTicker %s %s should not error", a, p) {
				assert.Positive(t, tick.Last, "%s %s Tick Last should be positive", a, p)
				assert.NotEmpty(t, tick.Pair, "%s %s Tick Pair should not be empty", a, p)
				assert.Equal(t, ku.Name, tick.ExchangeName, "ExchangeName should be correct")
				assert.Equal(t, a, tick.AssetType, "AssetType should be correct")
				assert.NotEmpty(t, tick.LastUpdated, "%s %s Tick LastUpdated should not be empty", a, p)
			}
		}
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

func TestPushData(t *testing.T) {
	n := new(Kucoin)
	sharedtestvalues.TestFixtureToDataHandler(t, ku, n, "testdata/wsHandleData.json", ku.wsHandleData)
}

func verifySubs(tb testing.TB, subs []subscription.Subscription, a asset.Item, prefix string, expected ...string) {
	tb.Helper()
	var sub *subscription.Subscription
	for i, s := range subs { //nolint:gocritic // prefer convenience over performance here for tests
		if s.Asset == a && strings.HasPrefix(s.Channel, prefix) {
			if len(expected) == 1 && !strings.Contains(s.Channel, expected[0]) {
				continue
			}
			if sub != nil {
				assert.Failf(tb, "Too many subs with prefix", "Asset %s; Prefix %s", a.String(), prefix)
				return
			}
			sub = &subs[i]
		}
	}
	if assert.NotNil(tb, sub, "Should find a sub for asset %s with prefix %s for %s", a.String(), prefix, strings.Join(expected, ", ")) {
		suffix := strings.TrimPrefix(sub.Channel, prefix)
		if len(expected) == 0 {
			assert.Empty(tb, suffix, "Sub for asset %s with prefix %s should have no symbol suffix", a.String(), prefix)
		} else {
			currs := strings.Split(suffix, ",")
			assert.ElementsMatch(tb, currs, expected, "Currencies should match in sub for asset %s with prefix %s", a.String(), prefix)
		}
	}
}

// Pairs for Subscription tests:
// Only in Spot: BTC-USDT, ETH-USDT
// In Both: ETH-BTC, LTC-USDT
// Only in Margin: TRX-BTC, SOL-USDC

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()

	subs, err := ku.GenerateDefaultSubscriptions()

	assert.NoError(t, err, "GenerateDefaultSubscriptions should not error")

	assert.Len(t, subs, 11, "Should generate the correct number of subs when not logged in")

	verifySubs(t, subs, asset.Spot, "/market/ticker:all") // This takes care of margin as well.

	verifySubs(t, subs, asset.Spot, "/market/match:", "BTC-USDT", "ETH-USDT", "LTC-USDT", "ETH-BTC")
	verifySubs(t, subs, asset.Margin, "/market/match:", "SOL-USDC", "TRX-BTC")

	verifySubs(t, subs, asset.Spot, "/spotMarket/level2Depth5:", "BTC-USDT", "ETH-USDT", "LTC-USDT", "ETH-BTC")
	verifySubs(t, subs, asset.Margin, "/spotMarket/level2Depth5:", "SOL-USDC", "TRX-BTC")

	for _, c := range []string{"ETHUSDCM", "XBTUSDCM", "SOLUSDTM"} {
		verifySubs(t, subs, asset.Futures, "/contractMarket/tickerV2:", c)
		verifySubs(t, subs, asset.Futures, "/contractMarket/level2Depth50:", c)
	}
}

func TestGenerateAuthSubscriptions(t *testing.T) {
	t.Parallel()

	// Create a parallel safe Kucoin to mess with
	nu := new(Kucoin)
	nu.Base.Features = ku.Base.Features
	assert.NoError(t, nu.CurrencyPairs.Load(&ku.CurrencyPairs), "Loading Pairs should not error")
	nu.Websocket = sharedtestvalues.NewTestWebsocket()
	nu.Websocket.SetCanUseAuthenticatedEndpoints(true)

	subs, err := nu.GenerateDefaultSubscriptions()
	assert.NoError(t, err, "GenerateDefaultSubscriptions with Auth should not error")
	assert.Len(t, subs, 24, "Should generate the correct number of subs when logged in")

	verifySubs(t, subs, asset.Spot, "/market/ticker:all") // This takes care of margin as well.

	verifySubs(t, subs, asset.Spot, "/market/match:", "BTC-USDT", "ETH-USDT", "LTC-USDT", "ETH-BTC")
	verifySubs(t, subs, asset.Margin, "/market/match:", "SOL-USDC", "TRX-BTC")

	verifySubs(t, subs, asset.Spot, "/spotMarket/level2Depth5:", "BTC-USDT", "ETH-USDT", "LTC-USDT", "ETH-BTC")
	verifySubs(t, subs, asset.Margin, "/spotMarket/level2Depth5:", "SOL-USDC", "TRX-BTC")

	for _, c := range []string{"ETHUSDCM", "XBTUSDCM", "SOLUSDTM"} {
		verifySubs(t, subs, asset.Futures, "/contractMarket/tickerV2:", c)
		verifySubs(t, subs, asset.Futures, "/contractMarket/level2Depth50:", c)
	}
	for _, c := range []string{"SOL", "BTC", "TRX", "LTC", "USDC", "USDT", "ETH"} {
		verifySubs(t, subs, asset.Margin, "/margin/loan:", c)
	}
	verifySubs(t, subs, asset.Spot, "/account/balance")
	verifySubs(t, subs, asset.Margin, "/margin/position")
	verifySubs(t, subs, asset.Margin, "/margin/fundingBook:", "SOL", "BTC", "TRX", "LTC", "USDT", "USDC", "ETH")
	verifySubs(t, subs, asset.Futures, "/contractAccount/wallet")
	verifySubs(t, subs, asset.Futures, "/contractMarket/advancedOrders")
	verifySubs(t, subs, asset.Futures, "/contractMarket/tradeOrders")
}

func TestGenerateCandleSubscription(t *testing.T) {
	t.Parallel()

	// Create a parallel safe Kucoin to mess with
	nu := new(Kucoin)
	nu.Base.Features = ku.Base.Features
	nu.Websocket = sharedtestvalues.NewTestWebsocket()
	assert.NoError(t, nu.CurrencyPairs.Load(&ku.CurrencyPairs), "Loading Pairs should not error")

	nu.Features.Subscriptions = []*subscription.Subscription{
		{Channel: subscription.CandlesChannel, Interval: kline.FourHour},
	}

	subs, err := nu.GenerateDefaultSubscriptions()
	assert.NoError(t, err, "GenerateDefaultSubscriptions with Candles should not error")

	assert.Len(t, subs, 6, "Should generate the correct number of subs for candles")
	for _, c := range []string{"BTC-USDT", "ETH-USDT", "LTC-USDT", "ETH-BTC"} {
		verifySubs(t, subs, asset.Spot, "/market/candles:", c+"_4hour")
	}
	for _, c := range []string{"SOL-USDC", "TRX-BTC"} {
		verifySubs(t, subs, asset.Margin, "/market/candles:", c+"_4hour")
	}
}

func TestGenerateMarketSubscription(t *testing.T) {
	t.Parallel()

	// Create a parallel safe Kucoin to mess with
	nu := new(Kucoin)
	nu.Base.Features = ku.Base.Features
	nu.Websocket = sharedtestvalues.NewTestWebsocket()
	assert.NoError(t, nu.CurrencyPairs.Load(&ku.CurrencyPairs), "Loading Pairs should not error")

	nu.Features.Subscriptions = []*subscription.Subscription{
		{Channel: marketSnapshotChannel},
	}

	subs, err := nu.GenerateDefaultSubscriptions()
	assert.NoError(t, err, "GenerateDefaultSubscriptions with MarketSnapshot should not error")

	assert.Len(t, subs, 7, "Should generate the correct number of subs for snapshot")
	for _, c := range []string{"BTC", "ETH", "LTC", "USDT"} {
		verifySubs(t, subs, asset.Spot, "/market/snapshot:", c)
	}
	for _, c := range []string{"SOL", "USDC", "TRX"} {
		verifySubs(t, subs, asset.Margin, "/market/snapshot:", c)
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
	orderSubmission.AssetType = asset.Options
	_, err := ku.SubmitOrder(context.Background(), orderSubmission)
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
	err = ku.processOrderbook([]byte(orderbookLevel5PushData), "BTC-USDT", "")
	if err != nil {
		t.Error(err)
	}
	err = ku.wsHandleData([]byte(orderbookLevel5PushData))
	if err != nil {
		t.Error(err)
	}
}

func TestProcessMarketSnapshot(t *testing.T) {
	t.Parallel()
	n := new(Kucoin)
	sharedtestvalues.TestFixtureToDataHandler(t, ku, n, "testdata/wsMarketSnapshot.json", n.wsHandleData)
	seen := 0
	seenAssetTypes := map[asset.Item]int{}
	for reading := true; reading; {
		select {
		default:
			reading = false
		case resp := <-n.GetBase().Websocket.DataHandler:
			seen++
			switch v := resp.(type) {
			case *ticker.Price:
				switch seen {
				case 1:
					assert.Equal(t, asset.Margin, v.AssetType, "AssetType")
					assert.Equal(t, time.UnixMilli(1700555342007), v.LastUpdated, "datetime")
					assert.Equal(t, 0.004445, v.High, "high")
					assert.Equal(t, 0.004415, v.Last, "lastTradedPrice")
					assert.Equal(t, 0.004191, v.Low, "low")
					assert.Equal(t, currency.NewPairWithDelimiter("TRX", "BTC", "-"), v.Pair, "symbol")
					assert.Equal(t, 13097.3357, v.Volume, "volume")
					assert.Equal(t, 57.44552981, v.QuoteVolume, "volValue")
				case 2, 3:
					assert.Equal(t, time.UnixMilli(1700555340197), v.LastUpdated, "datetime")
					assert.Contains(t, []asset.Item{asset.Spot, asset.Margin}, v.AssetType, "AssetType is Spot or Margin")
					seenAssetTypes[v.AssetType]++
					assert.Equal(t, 1, seenAssetTypes[v.AssetType], "Each Asset Type is sent only once per unique snapshot")
					assert.Equal(t, 0.054846, v.High, "high")
					assert.Equal(t, 0.053778, v.Last, "lastTradedPrice")
					assert.Equal(t, 0.05364, v.Low, "low")
					assert.Equal(t, currency.NewPairWithDelimiter("ETH", "BTC", "-"), v.Pair, "symbol")
					assert.Equal(t, 2958.3139116, v.Volume, "volume")
					assert.Equal(t, 160.7847672784213, v.QuoteVolume, "volValue")
				case 4:
					assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
					assert.Equal(t, time.UnixMilli(1700555342151), v.LastUpdated, "datetime")
					assert.Equal(t, 37750.0, v.High, "high")
					assert.Equal(t, 37366.8, v.Last, "lastTradedPrice")
					assert.Equal(t, 36700.0, v.Low, "low")
					assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "-"), v.Pair, "symbol")
					assert.Equal(t, 2900.37846402, v.Volume, "volume")
					assert.Equal(t, 108210331.34015164, v.QuoteVolume, "volValue")
				default:
					t.Errorf("Got an unexpected *ticker.Price: %v", v)
				}
			case error:
				t.Error(v)
			default:
				t.Errorf("Got unexpected data: %T %v", v, v)
			}
		}
	}
	assert.Equal(t, 4, seen, "Number of messages")
}

func TestSubscribeMarketSnapshot(t *testing.T) {
	t.Parallel()
	setupWS()
	err := ku.Subscribe([]subscription.Subscription{{Channel: marketSymbolSnapshotChannel, Pair: currency.Pair{Base: currency.BTC}}})
	assert.NoError(t, err, "Subscribe to MarketSnapshot should not error")
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)

	req.StartDate = req.EndDate.Add(-time.Hour * 24 * 30)
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.ErrorIs(t, err, futures.ErrOrderHistoryTooLarge)

	req.RespectOrderHistoryLimits = true
	_, err = ku.GetFuturesPositionOrders(context.Background(), req)
	assert.NoError(t, err)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	err := ku.UpdateOrderExecutionLimits(context.Background(), asset.Binary)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("Received %v, expected %v", err, asset.ErrNotSupported)
	}

	assets := []asset.Item{asset.Spot, asset.Futures, asset.Margin}
	for x := range assets {
		err = ku.UpdateOrderExecutionLimits(context.Background(), assets[x])
		if !errors.Is(err, nil) {
			t.Fatalf("received %v, expected %v", err, nil)
		}

		enabled, err := ku.GetEnabledPairs(assets[x])
		if err != nil {
			t.Fatal(err)
		}

		for y := range enabled {
			lim, err := ku.GetOrderExecutionLimits(assets[x], enabled[y])
			if err != nil {
				t.Fatalf("%v %s %v", err, enabled[y], assets[x])
			}
			assert.NotEmpty(t, lim, "limit cannot be empty")
		}
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()

	nu := new(Kucoin)
	require.NoError(t, testexch.TestInstance(nu), "TestInstance setup should not error")

	_, err := nu.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	resp, err := nu.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  futuresTradablePair.Base.Item,
		Quote: futuresTradablePair.Quote.Item,
		Asset: asset.Futures,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	cp1 := currency.NewPair(currency.ETH, currency.USDTM)
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, nu, asset.Futures, cp1)
	resp, err = nu.GetOpenInterest(context.Background(),
		key.PairAsset{
			Base:  futuresTradablePair.Base.Item,
			Quote: futuresTradablePair.Quote.Item,
			Asset: asset.Futures,
		},
		key.PairAsset{
			Base:  cp1.Base.Item,
			Quote: cp1.Quote.Item,
			Asset: asset.Futures,
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = nu.GetOpenInterest(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}
