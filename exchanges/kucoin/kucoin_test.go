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
	require.NoError(t, err)
	require.NotEmpty(t, symbols, "should return all available spot/margin symbols")
	// Using market string reduces the scope of what is returned.
	symbols, err = ku.GetSymbols(context.Background(), "ETF")
	assert.NoError(t, err)
	assert.NotEmpty(t, symbols, "should return all available ETF symbols")
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTicker(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetAllTickers(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTickers(context.Background())
	assert.NoError(t, err)
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
	assert.NoError(t, err)
}

func TestGetMarketList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarketList(context.Background())
	assert.NoError(t, err)
}

func TestGetPartOrderbook20(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPartOrderbook20(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetPartOrderbook100(t *testing.T) {
	t.Parallel()
	_, err := ku.GetPartOrderbook100(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetOrderbook(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetTradeHistory(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := ku.GetKlines(context.Background(), "BTC-USDT", "1week", time.Time{}, time.Time{})
	require.NoError(t, err)
	_, err = ku.GetKlines(context.Background(), "BTC-USDT", "5min", time.Now().Add(-time.Hour*1), time.Now())
	assert.NoError(t, err)
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrencies(context.Background())
	assert.NoError(t, err)
}

func TestGetCurrenciesV3(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrenciesV3(context.Background())
	assert.NoError(t, err)
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrencyDetailV2(context.Background(), "BTC", "")
	require.NoError(t, err)

	_, err = ku.GetCurrencyDetailV2(context.Background(), "BTC", "ETH")
	assert.NoError(t, err)
}

func TestGetCurrencyV3(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrencyDetailV3(context.Background(), "BTC", "")
	require.NoError(t, err)

	_, err = ku.GetCurrencyDetailV3(context.Background(), "BTC", "ETH")
	assert.NoError(t, err)
}

func TestGetFiatPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFiatPrice(context.Background(), "", "")
	require.NoError(t, err)

	_, err = ku.GetFiatPrice(context.Background(), "EUR", "ETH,BTC")
	assert.NoError(t, err)
}

func TestGetLeveragedTokenInfo(t *testing.T) {
	t.Parallel()
	_, err := ku.GetLeveragedTokenInfo(context.Background(), "BTC")
	assert.NoError(t, err)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarkPrice(context.Background(), "USDT-BTC")
	assert.NoError(t, err)
}

func TestGetMarginConfiguration(t *testing.T) {
	t.Parallel()
	_, err := ku.GetMarginConfiguration(context.Background())
	assert.NoError(t, err)
}

func TestGetMarginAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetMarginAccount(context.Background())
	assert.NoError(t, err)
}

func TestGetMarginRiskLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetMarginRiskLimit(context.Background(), "cross")
	require.NoError(t, err)

	_, err = ku.GetMarginRiskLimit(context.Background(), "isolated")
	assert.NoError(t, err)
}

func TestPostBorrowOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostMarginBorrowOrder(context.Background(),
		&MarginBorrowParam{
			Currency:    currency.USDT,
			TimeInForce: "FOK", Size: 0})
	require.NoError(t, err)
	_, err = ku.PostMarginBorrowOrder(context.Background(),
		&MarginBorrowParam{
			Currency:    currency.USDT,
			TimeInForce: "IOC",
			Size:        0.05,
		})
	assert.NoError(t, err)
}

func TestGetMarginBorrowingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetMarginBorrowingHistory(context.Background(), currency.BTC, true, currency.EMPTYPAIR, "", time.Time{}, time.Now().Add(-time.Hour*80), 0, 10)
	assert.NoError(t, err)
}

func TestPostRepayment(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostRepayment(context.Background(), &RepayParam{
		Currency: currency.USDT,
		Size:     0.05})
	require.NoError(t, err)
}

func TestGetRepaymentHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetRepaymentHistory(context.Background(), currency.BTC, true, currency.EMPTYPAIR, "", time.Time{}, time.Now().Add(-time.Hour*80), 0, 10)
	assert.NoError(t, err)
}

const borrowOrderJSON = `{"orderId": "a2111213","currency": "USDT","size": "1.009","filled": 1.009,"matchList": [{"currency": "USDT","dailyIntRate": "0.001","size": "12.9","term": 7,"timestamp": "1544657947759","tradeId": "1212331"}],"status": "DONE"}`

func TestGetBorrowOrder(t *testing.T) {
	t.Parallel()
	var resp *BorrowOrder
	err := json.Unmarshal([]byte(borrowOrderJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetBorrowOrder(context.Background(), "orderID")
	assert.NoError(t, err)
}

func TestGetIsolatedMarginPairConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetIsolatedMarginPairConfig(context.Background())
	assert.NoError(t, err)
}

func TestGetIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetIsolatedMarginAccountInfo(context.Background(), "")
	require.NoError(t, err)
	_, err = ku.GetIsolatedMarginAccountInfo(context.Background(), "USDT")
	assert.NoError(t, err)
}

func TestGetSingleIsolatedMarginAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetSingleIsolatedMarginAccountInfo(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	_, err := ku.GetCurrentServerTime(context.Background())
	assert.NoError(t, err)
}

func TestGetServiceStatus(t *testing.T) {
	t.Parallel()
	_, err := ku.GetServiceStatus(context.Background())
	assert.NoError(t, err)
}

func TestPostOrder(t *testing.T) {
	t.Parallel()

	// default order type is limit
	_, err := ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: ""})
	require.ErrorIs(t, err, errInvalidClientOrderID)

	customID, err := uuid.NewV4()
	require.NoError(t, err)

	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair,
		OrderType: ""})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565})

	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Side: "buy",
		Symbol:    spotTradablePair,
		OrderType: "limit", Size: 0.1})
	require.ErrorIs(t, err, errInvalidPrice)
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(), Symbol: spotTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565})
	require.ErrorIs(t, err, errInvalidSize)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err = ku.PostOrder(context.Background(), &SpotOrderParam{
		ClientOrderID: customID.String(),
		Side:          "buy",
		Symbol:        spotTradablePair,
		OrderType:     "limit",
		Size:          0.005,
		Price:         1000})
	assert.NoError(t, err)
}

func TestPostMarginOrder(t *testing.T) {
	t.Parallel()
	// default order type is limit
	_, err := ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: ""})
	require.ErrorIs(t, err, errInvalidClientOrderID)
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair,
		OrderType: ""})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: currency.EMPTYPAIR,
		Size: 0.1, Side: "buy", Price: 234565})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy",
		Symbol:    marginTradablePair,
		OrderType: "limit", Size: 0.1})
	require.ErrorIs(t, err, errInvalidPrice)
	_, err = ku.PostMarginOrder(context.Background(), &MarginOrderParam{
		ClientOrderID: "5bd6e9286d99522a52e458de", Symbol: marginTradablePair, Side: "buy",
		OrderType: "limit", Price: 234565})
	require.ErrorIs(t, err, errInvalidSize)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	// default order type is limit and margin mode is cross
	_, err = ku.PostMarginOrder(context.Background(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			Price: 1000, Size: 0.1, PostOnly: true})
	require.NoError(t, err)

	// market isolated order
	_, err = ku.PostMarginOrder(context.Background(),
		&MarginOrderParam{
			ClientOrderID: "5bd6e9286d99522a52e458de",
			Side:          "buy", Symbol: marginTradablePair,
			OrderType: "market", Funds: 1234,
			Remark: "remark", MarginModel: "cross", Price: 1000, PostOnly: true, AutoBorrow: true})
	assert.NoError(t, err)
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
	assert.NoError(t, err)
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelSingleOrder(context.Background(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
}

func TestCancelOrderByClientOID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelOrderByClientOID(context.Background(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelAllOpenOrders(context.Background(), "", "")
	assert.NoError(t, err)
}

const ordersListResponseJSON = `{"currentPage": 1, "pageSize": 1, "totalNum": 153408, "totalPage": 153408, "items": [ { "id": "5c35c02703aa673ceec2a168", "symbol": "BTC-USDT", "opType": "DEAL", "type": "limit", "side": "buy", "price": "10", "size": "2", "funds": "0", "dealFunds": "0.166", "dealSize": "2", "fee": "0", "feeCurrency": "USDT", "stp": "", "stop": "", "stopTriggered": false, "stopPrice": "0", "timeInForce": "GTC", "postOnly": false, "hidden": false, "iceberg": false, "visibleSize": "0", "cancelAfter": 0, "channel": "IOS", "clientOid": "", "remark": "", "tags": "", "isActive": false, "cancelExist": false, "createdAt": 1547026471000, "tradeType": "TRADE" } ] }`

func TestGetOrders(t *testing.T) {
	t.Parallel()
	var resp *OrdersListResponse
	err := json.Unmarshal([]byte(ordersListResponseJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err = ku.ListOrders(context.Background(), "", "", "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetRecentOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetRecentOrders(context.Background())
	assert.NoError(t, err)
}

func TestGetOrderByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetOrderByID(context.Background(), "5c35c02703aa673ceec2a168")
	assert.NoError(t, err)
}

func TestGetOrderByClientOID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetOrderByClientSuppliedOrderID(context.Background(), "6d539dc614db312")
	assert.NoError(t, err)
}

func TestGetFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFills(context.Background(), "", "", "", "", "", time.Time{}, time.Time{})
	require.NoError(t, err)
	_, err = ku.GetFills(context.Background(), "5c35c02703aa673ceec2a168", "BTC-USDT", "buy", "limit", "TRADE", time.Now().Add(-time.Hour*12), time.Now())
	assert.NoError(t, err)
}

const limitFillsResponseJSON = `[{ "counterOrderId":"5db7ee769797cf0008e3beea", "createdAt":1572335233000, "fee":"0.946357371456", "feeCurrency":"USDT", "feeRate":"0.001", "forceTaker":true, "funds":"946.357371456", "liquidity":"taker", "orderId":"5db7ee805d53620008dce1ba", "price":"9466.8", "side":"buy", "size":"0.09996592", "stop":"", "symbol":"BTC-USDT", "tradeId":"5db7ee8054c05c0008069e21", "tradeType":"MARGIN_TRADE", "type":"market" }, { "counterOrderId":"5db7ee4b5d53620008dcde8e", "createdAt":1572335207000, "fee":"0.94625", "feeCurrency":"USDT", "feeRate":"0.001", "forceTaker":true, "funds":"946.25", "liquidity":"taker", "orderId":"5db7ee675d53620008dce01e", "price":"9462.5", "side":"sell", "size":"0.1", "stop":"", "symbol":"BTC-USDT", "tradeId":"5db7ee6754c05c0008069e03", "tradeType":"MARGIN_TRADE", "type":"market" }]`

func TestGetRecentFills(t *testing.T) {
	t.Parallel()
	var resp []Fill
	err := json.Unmarshal([]byte(limitFillsResponseJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetRecentFills(context.Background())
	assert.NoError(t, err)
}

func TestPostStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostStopOrder(context.Background(), "5bd6e9286d99522a52e458de", "buy", "BTC-USDT", "", "", "entry", "CO", "TRADE", "", 0.1, 1, 10, 0, 0, 0, true, false, false)
	assert.NoError(t, err)
}

func TestCancelStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelStopOrder(context.Background(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
}

func TestCancelAllStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelStopOrders(context.Background(), "", "", "")
	assert.NoError(t, err)
}

const stopOrderResponseJSON = `{"id": "vs8hoo8q2ceshiue003b67c0", "symbol": "KCS-USDT", "userId": "60fe4956c43cbc0006562c2c", "status": "NEW", "type": "limit", "side": "buy", "price": "0.01000000000000000000", "size": "0.01000000000000000000", "funds": null, "stp": null, "timeInForce": "GTC", "cancelAfter": -1, "postOnly": false, "hidden": false, "iceberg": false, "visibleSize": null, "channel": "API", "clientOid": "40e0eb9efe6311eb8e58acde48001122", "remark": null, "tags": null, "orderTime": 1629098781127530345, "domainId": "kucoin", "tradeSource": "USER", "tradeType": "TRADE", "feeCurrency": "USDT", "takerFeeRate": "0.00200000000000000000", "makerFeeRate": "0.00200000000000000000", "createdAt": 1629098781128, "stop": "loss", "stopTriggerTime": null, "stopPrice": "10.00000000000000000000" }`

func TestGetStopOrder(t *testing.T) {
	t.Parallel()
	var resp *StopOrder
	err := json.Unmarshal([]byte(stopOrderResponseJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetStopOrder(context.Background(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
}

func TestGetAllStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.ListStopOrders(context.Background(), "", "", "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.NoError(t, err)
}

func TestGetStopOrderByClientID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetStopOrderByClientID(context.Background(), "", "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
}

func TestCancelStopOrderByClientID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelStopOrderByClientID(context.Background(), "", "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
}

func TestGetAllAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetAllAccounts(context.Background(), "", "")
	assert.NoError(t, err)
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAccountDetail(context.Background(), "")
	assert.ErrorIs(t, err, errAccountIDMissing)
	_, err = ku.GetAccountDetail(context.Background(), "62fcd1969474ea0001fd20e4")
	assert.NoError(t, err)
}

func TestGetCrossMarginAccountsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetCrossMarginAccountsDetail(context.Background(), "KCS", "MARGIN_V2")
	assert.NoError(t, err)
}

func TestGetIsolatedMarginAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetIsolatedMarginAccountDetail(context.Background(), "BTCUSDT", "BTC", "")
	assert.NoError(t, err)
}

func TestGetFuturesAccountDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesAccountDetail(context.Background(), "XBT")
	assert.NoError(t, err)
}

func TestGetSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetSubAccounts(context.Background(), "5caefba7d9575a0688f83c45", false)
	assert.NoError(t, err)
}

func TestGetAllFuturesSubAccountBalances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAllFuturesSubAccountBalances(context.Background(), "BTC")
	assert.NoError(t, err)
}

const accountLedgerResponseJSON = `{"currentPage": 1, "pageSize": 50, "totalNum": 2, "totalPage": 1, "items": [ { "id": "611a1e7c6a053300067a88d9", "currency": "USDT", "amount": "10.00059547", "fee": "0", "balance": "0", "accountType": "MAIN", "bizType": "Loans Repaid", "direction": "in", "createdAt": 1629101692950, "context": "{\"borrowerUserId\":\"601ad03e50dc810006d242ea\",\"loanRepayDetailNo\":\"611a1e7cc913d000066cf7ec\"}" }, { "id": "611a18bc6a0533000671e1bf", "currency": "USDT", "amount": "10.00059547", "fee": "0", "balance": "0", "accountType": "MAIN", "bizType": "Loans Repaid", "direction": "in", "createdAt": 1629100220843, "context": "{\"borrowerUserId\":\"5e3f4623dbf52d000800292f\",\"loanRepayDetailNo\":\"611a18bc7255c200063ea545\"}" } ] }`

func TestGetAccountLedgers(t *testing.T) {
	t.Parallel()
	var resp *AccountLedgerResponse
	err := json.Unmarshal([]byte(accountLedgerResponseJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetAccountLedgers(context.Background(), "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetAccountLedgersHFTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAccountLedgersHFTrade(context.Background(), "BTC", "", "", 0, 10, time.Time{}, time.Now())
	require.NoError(t, err)
}

func TestGetAccountLedgerHFMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAccountLedgerHFMargin(context.Background(), "BTC", "", "", 0, 0, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesAccountLedgers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesAccountLedgers(context.Background(), "BTC", true, time.Time{}, time.Now(), 0, 100)
	assert.NoError(t, err)
}

func TestGetAllSubAccountsInfoV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAllSubAccountsInfoV1(context.Background())
	require.NoError(t, err)
}

func TestGetAllSubAccountsInfoV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAllSubAccountsInfoV2(context.Background(), 0, 30)
	require.NoError(t, err)
}

func TestGetAccountSummaryInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAccountSummaryInformation(context.Background())
	assert.NoError(t, err)
}

func TestGetAggregatedSubAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAggregatedSubAccountBalance(context.Background())
	assert.NoError(t, err)
}

func TestGetAllSubAccountsBalanceV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAllSubAccountsBalanceV2(context.Background())
	assert.NoError(t, err)
}

func TestGetPaginatedSubAccountInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetPaginatedSubAccountInformation(context.Background(), 0, 10)
	assert.NoError(t, err)
}

func TestGetTransferableBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetTransferableBalance(context.Background(), "BTC", "MAIN", "")
	assert.NoError(t, err)
}

func TestGetUniversalTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.GetUniversalTransfer(context.Background(), &UniversalTransferParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.GetUniversalTransfer(context.Background(), &UniversalTransferParam{
		ClientSuppliedOrderID: "64ccc0f164781800010d8c09",
		TransferType:          "INTERNAL",
		Currency:              currency.BTC,
		Amount:                1,
		FromAccountType:       "TRADE",
		ToAccountType:         "CONTRACT",
	})
	assert.NoError(t, err)
	_, err = ku.GetUniversalTransfer(context.Background(), &UniversalTransferParam{
		ClientSuppliedOrderID: "64ccc0f164781800010d8c09",
		TransferType:          "PARENT_TO_SUB",
		Currency:              currency.BTC,
		Amount:                1,
		FromAccountType:       "TRADE",
		ToUserID:              "62f5f5d4d72aaf000122707e",
		ToAccountType:         "CONTRACT",
	})
	require.NoError(t, err)
}

func TestTransferMainToSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.TransferMainToSubAccount(context.Background(), "62fcd1969474ea0001fd20e4", "BTC", "1", "OUT", "", "", "5caefba7d9575a0688f83c45")
	assert.NoError(t, err)
}

func TestMakeInnerTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.MakeInnerTransfer(context.Background(), "62fcd1969474ea0001fd20e4", "BTC", "trade", "main", "1", "", "")
	assert.NoError(t, err)
}

func TestTransferToMainOrTradeAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.TransferToMainOrTradeAccount(context.Background(), &FundTransferFuturesParam{})
	assert.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.TransferToMainOrTradeAccount(context.Background(), &FundTransferFuturesParam{
		Amount:             1,
		Currency:           currency.USDT,
		RecieveAccountType: "MAIN",
	})
	assert.NoError(t, err)
}

func TestTransferToFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.TransferToFuturesAccount(context.Background(), &FundTransferToFuturesParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.TransferToFuturesAccount(context.Background(), &FundTransferToFuturesParam{
		Amount:             1,
		Currency:           currency.USDT,
		PaymentAccountType: "MAIN",
	})
	assert.NoError(t, err)
}

func TestGetFuturesTransferOutRequestRecords(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesTransferOutRequestRecords(context.Background(), time.Time{}, time.Now(), "", "", "BTC", 0, 10)
	assert.NoError(t, err)
}

func TestCreateDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CreateDepositAddress(context.Background(), &DepositAddressParams{
		Currency: currency.BTC,
	})
	assert.NoError(t, err)
	_, err = ku.CreateDepositAddress(context.Background(),
		&DepositAddressParams{
			Currency: currency.USDT,
			Chain:    "TRC20"})
	assert.NoError(t, err)
}

func TestGetDepositAddressV2(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetDepositAddressesV2(context.Background(), "BTC")
	assert.NoError(t, err)
}

func TestGetDepositAddressesV1(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetDepositAddressV1(context.Background(), "BTC", "")
	assert.NoError(t, err)
}

const depositResponseJSON = `{"currentPage": 1, "pageSize": 50, "totalNum": 1, "totalPage": 1, "items": [ { "currency": "XRP", "chain": "xrp", "status": "SUCCESS", "address": "rNFugeoj3ZN8Wv6xhuLegUBBPXKCyWLRkB", "memo": "1919537769", "isInner": false, "amount": "20.50000000", "fee": "0.00000000", "walletTxId": "2C24A6D5B3E7D5B6AA6534025B9B107AC910309A98825BF5581E25BEC94AD83B@e8902757998fc352e6c9d8890d18a71c", "createdAt": 1666600519000, "updatedAt": 1666600549000, "remark": "Deposit" } ] }`

func TestGetDepositList(t *testing.T) {
	t.Parallel()
	var resp DepositResponse
	err := json.Unmarshal([]byte(depositResponseJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetDepositList(context.Background(), "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

const historicalDepositResponseJSON = `{"currentPage":1, "pageSize":1, "totalNum":9, "totalPage":9, "items":[ { "currency":"BTC", "createAt":1528536998, "amount":"0.03266638", "walletTxId":"55c643bc2c68d6f17266383ac1be9e454038864b929ae7cee0bc408cc5c869e8@12ffGWmMMD1zA1WbFm7Ho3JZ1w6NYXjpFk@234", "isInner":false, "status":"SUCCESS" } ] }`

func TestGetHistoricalDepositList(t *testing.T) {
	t.Parallel()
	var resp *HistoricalDepositWithdrawalResponse
	err := json.Unmarshal([]byte(historicalDepositResponseJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err = ku.GetHistoricalDepositList(context.Background(), "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricalWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetHistoricalWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetWithdrawalQuotas(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetWithdrawalQuotas(context.Background(), "BTC", "")
	assert.NoError(t, err)
}

func TestApplyWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.ApplyWithdrawal(context.Background(), "ETH", "0x597873884BC3a6C10cB6Eb7C69172028Fa85B25A", "", "", "", "", false, 1)
	assert.NoError(t, err)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.CancelWithdrawal(context.Background(), "5bffb63303aa675e8bbe18f9")
	assert.NoError(t, err)
}

func TestGetBasicFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetBasicFee(context.Background(), "1")
	assert.NoError(t, err)
}

func TestGetTradingFee(t *testing.T) {
	t.Parallel()

	_, err := ku.GetTradingFee(context.Background(), nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	avail, err := ku.GetAvailablePairs(asset.Spot)
	require.NoError(t, err)

	pairs := currency.Pairs{avail[0]}
	btcusdTradingFee, err := ku.GetTradingFee(context.Background(), pairs)
	require.NoErrorf(t, err, "received %v, expected %v", err, nil)
	assert.Len(t, btcusdTradingFee, 1)

	// NOTE: Test below will error out from an external call as this will exceed
	// the allowed pairs. If this does not error then this endpoint will allow
	// more items to be requested.
	pairs = append(pairs, avail[1:11]...)
	_, err = ku.GetTradingFee(context.Background(), pairs)
	require.NoError(t, err)

	got, err := ku.GetTradingFee(context.Background(), pairs[:10])
	require.NoError(t, err)
	assert.Len(t, got, 10)
}

// futures
func TestGetFuturesOpenContracts(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesOpenContracts(context.Background())
	assert.NoError(t, err)
}

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesContract(context.Background(), "XBTUSDTM")
	assert.NoError(t, err)
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
	assert.NoError(t, err)
}

func TestGetFuturesPartOrderbook20(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPartOrderbook20(context.Background(), "XBTUSDTM")
	assert.NoError(t, err)
}

func TestGetFuturesPartOrderbook100(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPartOrderbook100(context.Background(), "XBTUSDTM")
	assert.NoError(t, err)
}

func TestGetFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesTradeHistory(context.Background(), "XBTUSDTM")
	assert.NoError(t, err)
}

func TestGetFuturesInterestRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesInterestRate(context.Background(), "XBTUSDTM", time.Time{}, time.Time{}, false, false, 0, 0)
	assert.NoError(t, err)
}

func TestGetFuturesIndexList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesIndexList(context.Background(), futuresTradablePair.String(), time.Time{}, time.Time{}, false, false, 0, 10)
	assert.NoError(t, err)
}

func TestGetFuturesCurrentMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesCurrentMarkPrice(context.Background(), "XBTUSDTM")
	assert.NoError(t, err)
}

func TestGetFuturesPremiumIndex(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesPremiumIndex(context.Background(), "XBTUSDTM", time.Time{}, time.Time{}, false, false, 0, 0)
	assert.NoError(t, err)
}

func TestGetFuturesCurrentFundingRate(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesCurrentFundingRate(context.Background(), "XBTUSDTM")
	assert.NoError(t, err)
}

func TestGetFuturesServerTime(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesServerTime(context.Background())
	assert.NoError(t, err)
}

func TestGetFuturesServiceStatus(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesServiceStatus(context.Background())
	assert.NoError(t, err)
}

func TestGetFuturesKline(t *testing.T) {
	t.Parallel()
	_, err := ku.GetFuturesKline(context.Background(), int64(kline.ThirtyMin.Duration().Minutes()), "XBTUSDTM", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestPostFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de"})
	assert.ErrorIs(t, err, errInvalidLeverage)
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{Side: "buy", Leverage: 0.02})
	assert.ErrorIs(t, err, errInvalidClientOrderID)
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Leverage: 0.02})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Leverage: 0.02})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	// With Stop order configuration
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 0.02, VisibleSize: 0})
	assert.ErrorIs(t, err, errInvalidStopPriceType)

	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", TimeInForce: "", Size: 1, Price: 1000, StopPrice: 0, Leverage: 0.02, VisibleSize: 0})
	assert.ErrorIs(t, err, errInvalidPrice)

	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Stop: "up", StopPriceType: "TP", StopPrice: 123456, TimeInForce: "", Size: 1, Price: 1000, Leverage: 0.02, VisibleSize: 0})
	assert.NoError(t, err)

	// Limit Orders
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "limit", Remark: "10", Leverage: 0.02})
	assert.ErrorIs(t, err, errInvalidPrice)
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10", Price: 1000, Leverage: 0.02, VisibleSize: 0})
	assert.ErrorIs(t, err, errInvalidSize)
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "limit", Remark: "10",
		Size: 1, Price: 1000, Leverage: 0.02, VisibleSize: 0})
	assert.NoError(t, err)

	// Market Orders
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair,
		OrderType: "market", Remark: "10", Leverage: 0.02})
	assert.ErrorIs(t, err, errInvalidSize)
	_, err = ku.PostFuturesOrder(context.Background(), &FuturesOrderParam{ClientOrderID: "5bd6e9286d99522a52e458de", Side: "buy", Symbol: futuresTradablePair, OrderType: "market", Remark: "10",
		Size: 1, Leverage: 0.02, VisibleSize: 0})
	assert.ErrorIs(t, err, errInvalidSize)

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
	assert.NoError(t, err)
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelFuturesOrder(context.Background(), "5bd6e9286d99522a52e458de")
	assert.NoError(t, err)
}

func TestCancelAllFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)

	_, err := ku.CancelAllFuturesOpenOrders(context.Background(), "XBTUSDM")
	assert.NoError(t, err)
}

func TestCancelAllFuturesStopOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelAllFuturesStopOrders(context.Background(), "XBTUSDM")
	assert.NoError(t, err)
}

func TestGetFuturesOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOrders(context.Background(), "", "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetUntriggeredFuturesStopOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetUntriggeredFuturesStopOrders(context.Background(), "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesRecentCompletedOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesRecentCompletedOrders(context.Background())
	assert.NoError(t, err)
}

func TestGetFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOrderDetails(context.Background(), "5cdfc138b21023a909e5ad55")
	assert.NoError(t, err)
}

func TestGetFuturesOrderDetailsByClientID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOrderDetailsByClientID(context.Background(), "eresc138b21023a909e5ad59")
	assert.NoError(t, err)
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesFills(context.Background(), "", "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesRecentFills(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesRecentFills(context.Background())
	assert.NoError(t, err)
}

func TestGetFuturesOpenOrderStats(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesOpenOrderStats(context.Background(), "XBTUSDM")
	assert.NoError(t, err)
}

func TestGetFuturesPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesPosition(context.Background(), "XBTUSDM")
	assert.NoError(t, err)
}

func TestGetFuturesPositionList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesPositionList(context.Background())
	assert.NoError(t, err)
}

func TestSetAutoDepositMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SetAutoDepositMargin(context.Background(), "ADAUSDTM", true)
	assert.NoError(t, err)
}

func TestAddMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.AddMargin(context.Background(), "XBTUSDTM", "6200c9b83aecfb000152dasfdee", 1)
	assert.NoError(t, err)
}

func TestGetFuturesRiskLimitLevel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)

	_, err := ku.GetFuturesRiskLimitLevel(context.Background(), "ADAUSDTM")
	assert.NoError(t, err)
}

func TestUpdateRiskLmitLevel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.FuturesUpdateRiskLmitLevel(context.Background(), "ADASUDTM", 2)
	assert.NoError(t, err)
}

func TestGetFuturesFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesFundingHistory(context.Background(), futuresTradablePair.String(), 0, 0, true, true, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesAccountOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesAccountOverview(context.Background(), "")
	assert.NoError(t, err)
}

func TestGetFuturesTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesTransactionHistory(context.Background(), "", "", 0, 0, true, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestCreateFuturesSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CreateFuturesSubAccountAPIKey(context.Background(), "", "passphrase", "", "remark", "subAccName")
	assert.NoError(t, err)
}

func TestGetFuturesDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesDepositAddress(context.Background(), "XBT")
	assert.NoError(t, err)
}

func TestGetFuturesDepositsList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesDepositsList(context.Background(), "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesWithdrawalLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesWithdrawalLimit(context.Background(), "XBT")
	assert.NoError(t, err)
}

func TestGetFuturesWithdrawalList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesWithdrawalList(context.Background(), "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestCancelFuturesWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelFuturesWithdrawal(context.Background(), "5cda659603aa67131f305f7e")
	assert.NoError(t, err)
}

func TestTransferFuturesFundsToMainAccount(t *testing.T) {
	t.Parallel()
	var resp *TransferRes
	err := json.Unmarshal([]byte(transferFuturesFundsResponseJSON), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err = ku.TransferFuturesFundsToMainAccount(context.Background(), 1, "USDT", "MAIN")
	assert.NoError(t, err)
}

func TestTransferFundsToFuturesAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.TransferFundsToFuturesAccount(context.Background(), 1, "USDT", "MAIN")
	assert.NoError(t, err)
}

func TestGetFuturesTransferOutList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetFuturesTransferOutList(context.Background(), "USDT", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestCancelFuturesTransferOut(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	err := ku.CancelFuturesTransferOut(context.Background(), "5cd53be30c19fc3754b60928")
	assert.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := ku.FetchTradablePairs(context.Background(), asset.Futures)
	assert.NoError(t, err)
	_, err = ku.FetchTradablePairs(context.Background(), asset.Spot)
	assert.NoError(t, err)
	_, err = ku.FetchTradablePairs(context.Background(), asset.Margin)
	assert.NoError(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ku.UpdateOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	_, err = ku.UpdateOrderbook(context.Background(), marginTradablePair, asset.Margin)
	assert.NoError(t, err)
	_, err = ku.UpdateOrderbook(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
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
	require.NoError(t, err)
	_, err = ku.UpdateTicker(context.Background(), marginTradablePair, asset.Margin)
	require.NoError(t, err)
	_, err = ku.UpdateTicker(context.Background(), futuresTradablePair, asset.Futures)
	require.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := ku.FetchTicker(context.Background(), spotTradablePair, asset.Spot)
	require.NoError(t, err)
	_, err = ku.FetchTicker(context.Background(), marginTradablePair, asset.Margin)
	assert.NoError(t, err)
	_, err = ku.FetchTicker(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ku.FetchOrderbook(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	_, err = ku.FetchOrderbook(context.Background(), marginTradablePair, asset.Margin)
	assert.NoError(t, err)
	_, err = ku.FetchOrderbook(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 48)
	endTime := time.Now().Add(-time.Hour * 3)
	var err error
	_, err = ku.GetHistoricCandles(context.Background(), futuresTradablePair, asset.Futures, kline.OneHour, startTime, endTime)
	require.NoError(t, err)
	_, err = ku.GetHistoricCandles(context.Background(), spotTradablePair, asset.Spot, kline.OneHour, startTime, time.Now())
	require.NoError(t, err)
	_, err = ku.GetHistoricCandles(context.Background(), marginTradablePair, asset.Margin, kline.OneHour, startTime, time.Now())
	require.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	startTime := time.Now().Add(-time.Hour * 48)
	endTime := time.Now().Add(-time.Hour * 1)
	var err error
	_, err = ku.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.OneHour, startTime, endTime)
	require.NoError(t, err)
	_, err = ku.GetHistoricCandlesExtended(context.Background(), spotTradablePair, asset.Spot, kline.FiveMin, startTime, endTime)
	assert.NoError(t, err)
	_, err = ku.GetHistoricCandlesExtended(context.Background(), marginTradablePair, asset.Margin, kline.OneHour, startTime, endTime)
	require.NoError(t, err)
	_, err = ku.GetHistoricCandlesExtended(context.Background(), futuresTradablePair, asset.Futures, kline.FiveMin, startTime, endTime)
	assert.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := ku.GetServerTime(context.Background(), asset.Spot)
	assert.NoError(t, err)
	_, err = ku.GetServerTime(context.Background(), asset.Futures)
	assert.NoError(t, err)
	_, err = ku.GetServerTime(context.Background(), asset.Margin)
	assert.NoError(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := ku.GetRecentTrades(context.Background(), futuresTradablePair, asset.Futures)
	assert.NoError(t, err)
	_, err = ku.GetRecentTrades(context.Background(), spotTradablePair, asset.Spot)
	assert.NoError(t, err)
	_, err = ku.GetRecentTrades(context.Background(), marginTradablePair, asset.Margin)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var enabledPairs currency.Pairs
	var getOrdersRequest order.MultiOrderRequest
	var err error
	enabledPairs, err = ku.GetEnabledPairs(asset.Futures)
	require.NoError(t, err)
	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     append([]currency.Pair{currency.NewPair(currency.BTC, currency.USDT)}, enabledPairs[:3]...),
		AssetType: asset.Futures,
		Side:      order.AnySide,
	}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
	getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     []currency.Pair{spotTradablePair},
		AssetType: asset.Spot,
		Side:      order.Sell,
	}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
	getOrdersRequest.Pairs = []currency.Pair{}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
	getOrdersRequest.AssetType = asset.Margin
	getOrdersRequest.Pairs = currency.Pairs{marginTradablePair}
	_, err = ku.GetOrderHistory(context.Background(), &getOrdersRequest)
	assert.NoError(t, err)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var getOrdersRequest order.MultiOrderRequest
	var enabledPairs currency.Pairs
	var err error
	enabledPairs, err = ku.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	require.NoError(t, err)
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
	_, err := ku.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	})
	assert.NoError(t, err)
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	assetTypes := ku.CurrencyPairs.GetAssetTypes(true)
	for _, at := range assetTypes {
		err := ku.ValidateCredentials(context.Background(), at)
		require.NoError(t, err)
	}
}

func TestGetInstanceServers(t *testing.T) {
	t.Parallel()
	_, err := ku.GetInstanceServers(context.Background())
	assert.NoError(t, err)
}

func TestGetAuthenticatedServersInstances(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAuthenticatedInstanceServers(context.Background())
	assert.NoError(t, err)
}

func TestPushData(t *testing.T) {
	n := new(Kucoin)
	sharedtestvalues.TestFixtureToDataHandler(t, ku, n, "testdata/wsHandleData.json", ku.wsHandleData)
}

func verifySubs(tb testing.TB, subs []subscription.Subscription, a asset.Item, prefix string, expected ...string) {
	tb.Helper()
	var sub *subscription.Subscription
	for i, s := range subs {
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
// Only in Margin: XMR-BTC, SOL-USDC

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()

	subs, err := ku.GenerateDefaultSubscriptions()
	assert.NoError(t, err, "GenerateDefaultSubscriptions should not error")

	assert.Len(t, subs, 12, "Should generate the correct number of subs when not logged in")
	for _, p := range []string{"ticker", "match", "level2"} {
		verifySubs(t, subs, asset.Spot, "/market/"+p+":", "BTC-USDT", "ETH-USDT", "LTC-USDT", "ETH-BTC")
		verifySubs(t, subs, asset.Margin, "/market/"+p+":", "SOL-USDC", "XMR-BTC")
	}
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
	assert.Len(t, subs, 25, "Should generate the correct number of subs when logged in")
	for _, p := range []string{"ticker", "match", "level2"} {
		verifySubs(t, subs, asset.Spot, "/market/"+p+":", "BTC-USDT", "ETH-USDT", "LTC-USDT", "ETH-BTC")
		verifySubs(t, subs, asset.Margin, "/market/"+p+":", "SOL-USDC", "XMR-BTC")
	}
	for _, c := range []string{"ETHUSDCM", "XBTUSDCM", "SOLUSDTM"} {
		verifySubs(t, subs, asset.Futures, "/contractMarket/tickerV2:", c)
		verifySubs(t, subs, asset.Futures, "/contractMarket/level2Depth50:", c)
	}
	for _, c := range []string{"SOL", "BTC", "XMR", "LTC", "USDC", "USDT", "ETH"} {
		verifySubs(t, subs, asset.Margin, "/margin/loan:", c)
	}
	verifySubs(t, subs, asset.Spot, "/account/balance")
	verifySubs(t, subs, asset.Margin, "/margin/position")
	verifySubs(t, subs, asset.Margin, "/margin/fundingBook:", "SOL", "BTC", "XMR", "LTC", "USDT", "USDC", "ETH")
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
	for _, c := range []string{"SOL-USDC", "XMR-BTC"} {
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
	for _, c := range []string{"SOL", "USDC", "XMR"} {
		verifySubs(t, subs, asset.Margin, "/market/snapshot:", c)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetAvailableTransferChains(context.Background(), currency.BTC)
	assert.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Futures)
	assert.NoError(t, err)
	_, err = ku.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot)
	assert.NoError(t, err)
	_, err = ku.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Margin)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	var err error
	_, err = ku.GetOrderInfo(context.Background(), "123", futuresTradablePair, asset.Futures)
	assert.NoErrorf(t, err, "expected %s, but found %v", "Order does not exist", err)
	_, err = ku.GetOrderInfo(context.Background(), "123", futuresTradablePair, asset.Spot)
	assert.NoErrorf(t, err, "expected %s, but found %v", "Order does not exist", err)
	_, err = ku.GetOrderInfo(context.Background(), "123", futuresTradablePair, asset.Margin)
	assert.NoErrorf(t, err, "expected %s, but found %v", "Order does not exist", err)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetDepositAddress(context.Background(), currency.BTC, "", "")
	assert.True(t, err == nil || errors.Is(err, errNoDepositAddress), err)
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
	_, err := ku.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest)
	assert.NoError(t, err)
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
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	orderSubmission.Side = order.Buy
	orderSubmission.AssetType = asset.Options
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	orderSubmission.AssetType = asset.Spot
	orderSubmission.Side = order.Buy
	orderSubmission.Pair = spotTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid)
	orderSubmission.AssetType = asset.Spot
	orderSubmission.Pair = spotTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	assert.NoError(t, err)
	orderSubmission.AssetType = asset.Margin
	orderSubmission.Pair = marginTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	assert.NoError(t, err)
	orderSubmission.AssetType = asset.Margin
	orderSubmission.Pair = marginTradablePair
	orderSubmission.MarginType = margin.Isolated
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	assert.NoError(t, err)
	orderSubmission.AssetType = asset.Futures
	orderSubmission.Pair = futuresTradablePair
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	assert.ErrorIs(t, err, errInvalidLeverage)
	orderSubmission.Leverage = 0.01
	_, err = ku.SubmitOrder(context.Background(), orderSubmission)
	assert.NoError(t, err)
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
	err := ku.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
	orderCancellation.Pair = marginTradablePair
	orderCancellation.AssetType = asset.Margin
	err = ku.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
	orderCancellation.Pair = futuresTradablePair
	orderCancellation.AssetType = asset.Futures
	err = ku.CancelOrder(context.Background(), orderCancellation)
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType:  asset.Futures,
		MarginType: margin.Isolated,
	})
	assert.NoError(t, err)
	_, err = ku.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType:  asset.Margin,
		MarginType: margin.Isolated,
	})
	assert.NoError(t, err)
	_, err = ku.CancelAllOrders(context.Background(), &order.Cancel{
		AssetType:  asset.Spot,
		MarginType: margin.Isolated,
	})
	assert.NoError(t, err)
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
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err = ku.CreateSubUser(context.Background(), "SamuaelTee1", "sdfajdlkad", "", "")
	assert.NoError(t, err)
}

func TestGetSubAccountSpotAPIList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetSubAccountSpotAPIList(context.Background(), "sam", "")
	assert.NoError(t, err)
}

func TestCreateSpotAPIsForSubAccount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CreateSpotAPIsForSubAccount(context.Background(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		Remark:         "the-remark",
	})
	assert.NoError(t, err)
}

func TestModifySubAccountSpotAPIs(t *testing.T) {
	t.Parallel()
	var resp SpotAPISubAccount
	err := json.Unmarshal([]byte(modifySubAccountSpotAPIs), &resp)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err = ku.ModifySubAccountSpotAPIs(context.Background(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
	})
	require.ErrorIs(t, err, errAPIKeyRequired)
	_, err = ku.ModifySubAccountSpotAPIs(context.Background(), &SpotAPISubAccountParams{
		SubAccountName: "gocryptoTrader1",
		Passphrase:     "mysecretPassphrase123",
		APIKey:         apiKey,
	})
	assert.NoError(t, err)
}

func TestDeleteSubAccountSpotAPI(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.DeleteSubAccountSpotAPI(context.Background(), apiKey, "gocryptoTrader1", "the-passphrase")
	assert.NoError(t, err)
}

func TestGetUserInfoOfAllSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetUserInfoOfAllSubAccounts(context.Background())
	assert.NoError(t, err)
}

func TestGetPaginatedListOfSubAccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetPaginatedListOfSubAccounts(context.Background(), 1, 100)
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	require.NoError(t, err)
	_, err = ku.FetchAccountInfo(context.Background(), asset.Margin)
	require.NoError(t, err)
	_, err = ku.FetchAccountInfo(context.Background(), asset.Futures)
	require.NoError(t, err)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.UpdateAccountInfo(context.Background(), asset.Spot)
	assert.NoError(t, err)
	_, err = ku.UpdateAccountInfo(context.Background(), asset.Futures)
	assert.NoError(t, err)
	_, err = ku.UpdateAccountInfo(context.Background(), asset.Margin)
	assert.NoError(t, err)
}

const (
	orderbookLevel5PushData = `{"type": "message","topic": "/spotMarket/level2Depth50:BTC-USDT","subject": "level2","data": {"asks": [["21621.7","3.03206193"],["21621.8","1.00048239"],["21621.9","0.29558803"],["21622","0.0049653"],["21622.4","0.06177582"],["21622.9","0.39664116"],["21623.7","0.00803466"],["21624.2","0.65405"],["21624.3","0.34661426"],["21624.6","0.00035589"],["21624.9","0.61282048"],["21625.2","0.16421424"],["21625.4","0.90107014"],["21625.5","0.73484442"],["21625.9","0.04"],["21626.2","0.28569324"],["21626.4","0.18403701"],["21627.1","0.06503999"],["21627.2","0.56105832"],["21627.7","0.10649999"],["21628.1","2.66459953"],["21628.2","0.32"],["21628.5","0.27605551"],["21628.6","1.59482596"],["21628.9","0.16"],["21629.8","0.08"],["21630","0.04"],["21631.6","0.1"],["21631.8","0.0920185"],["21633.6","0.00447983"],["21633.7","0.00015044"],["21634.3","0.32193346"],["21634.4","0.00004"],["21634.5","0.1"],["21634.6","0.0002865"],["21635.6","0.12069941"],["21635.8","0.00117158"],["21636","0.00072816"],["21636.5","0.98611492"],["21636.6","0.00007521"],["21637.2","0.00699999"],["21637.6","0.00017129"],["21638","0.00013035"],["21638.1","0.05"],["21638.5","0.92427"],["21639.2","1.84998696"],["21639.3","0.04827233"],["21640","0.56255996"],["21640.9","0.8"],["21641","0.12"]],"bids": [["21621.6","0.40949924"],["21621.5","0.27703279"],["21621.3","0.04"],["21621.1","0.0086"],["21621","0.6653104"],["21620.9","0.35435999"],["21620.8","0.37224309"],["21620.5","0.416184"],["21620.3","0.24"],["21619.6","0.13883999"],["21619.5","0.21053355"],["21618.7","0.2"],["21618.6","0.001"],["21618.5","0.2258151"],["21618.4","0.06503999"],["21618.3","0.00370056"],["21618","0.12067842"],["21617.7","0.34844131"],["21617.6","0.92845495"],["21617.5","0.66460535"],["21617","0.01"],["21616.7","0.0004624"],["21616.4","0.02"],["21615.6","0.04828251"],["21615","0.59065665"],["21614.4","0.00227"],["21614.3","0.1"],["21613","0.32193346"],["21612.9","0.0028638"],["21612.6","0.1"],["21612.5","0.92539"],["21610.7","0.08208616"],["21610.6","0.00967666"],["21610.3","0.12"],["21610.2","0.00611126"],["21609.9","0.00226344"],["21609.8","0.00315812"],["21609.1","0.00547218"],["21608.6","0.09793157"],["21608.5","0.00437793"],["21608.4","1.85013454"],["21608.1","0.00366647"],["21607.9","0.00611595"],["21607.7","0.83263561"],["21607.6","0.00368919"],["21607.5","0.00280702"],["21607.1","0.66610849"],["21606.8","0.00364164"],["21606.2","0.80351642"],["21605.7","0.075"]],"timestamp": 1676319280783}}`
	wsOrderbookData         = `{"changes":{"asks":[["21621.7","3.03206193",""],["21621.8","1.00048239",""],["21621.9","0.29558803",""],["21622","0.0049653",""],["21622.4","0.06177582",""],["21622.9","0.39664116",""],["21623.7","0.00803466",""],["21624.2","0.65405",""],["21624.3","0.34661426",""],["21624.6","0.00035589",""],["21624.9","0.61282048",""],["21625.2","0.16421424",""],["21625.4","0.90107014",""],["21625.5","0.73484442",""],["21625.9","0.04",""],["21626.2","0.28569324",""],["21626.4","0.18403701",""],["21627.1","0.06503999",""],["21627.2","0.56105832",""],["21627.7","0.10649999",""],["21628.1","2.66459953",""],["21628.2","0.32",""],["21628.5","0.27605551",""],["21628.6","1.59482596",""],["21628.9","0.16",""],["21629.8","0.08",""],["21630","0.04",""],["21631.6","0.1",""],["21631.8","0.0920185",""],["21633.6","0.00447983",""],["21633.7","0.00015044",""],["21634.3","0.32193346",""],["21634.4","0.00004",""],["21634.5","0.1",""],["21634.6","0.0002865",""],["21635.6","0.12069941",""],["21635.8","0.00117158",""],["21636","0.00072816",""],["21636.5","0.98611492",""],["21636.6","0.00007521",""],["21637.2","0.00699999",""],["21637.6","0.00017129",""],["21638","0.00013035",""],["21638.1","0.05",""],["21638.5","0.92427",""],["21639.2","1.84998696",""],["21639.3","0.04827233",""],["21640","0.56255996",""],["21640.9","0.8",""],["21641","0.12",""]],"bids":[["21621.6","0.40949924",""],["21621.5","0.27703279",""],["21621.3","0.04",""],["21621.1","0.0086",""],["21621","0.6653104",""],["21620.9","0.35435999",""],["21620.8","0.37224309",""],["21620.5","0.416184",""],["21620.3","0.24",""],["21619.6","0.13883999",""],["21619.5","0.21053355",""],["21618.7","0.2",""],["21618.6","0.001",""],["21618.5","0.2258151",""],["21618.4","0.06503999",""],["21618.3","0.00370056",""],["21618","0.12067842",""],["21617.7","0.34844131",""],["21617.6","0.92845495",""],["21617.5","0.66460535",""],["21617","0.01",""],["21616.7","0.0004624",""],["21616.4","0.02",""],["21615.6","0.04828251",""],["21615","0.59065665",""],["21614.4","0.00227",""],["21614.3","0.1",""],["21613","0.32193346",""],["21612.9","0.0028638",""],["21612.6","0.1",""],["21612.5","0.92539",""],["21610.7","0.08208616",""],["21610.6","0.00967666",""],["21610.3","0.12",""],["21610.2","0.00611126",""],["21609.9","0.00226344",""],["21609.8","0.00315812",""],["21609.1","0.00547218",""],["21608.6","0.09793157",""],["21608.5","0.00437793",""],["21608.4","1.85013454",""],["21608.1","0.00366647",""],["21607.9","0.00611595",""],["21607.7","0.83263561",""],["21607.6","0.00368919",""],["21607.5","0.00280702",""],["21607.1","0.66610849",""],["21606.8","0.00364164",""],["21606.2","0.80351642",""],["21605.7","0.075",""]]},"sequenceEnd":1676319280783,"sequenceStart":0,"symbol":"BTC-USDT","time":1676319280783}`
)

func TestProcessOrderbook(t *testing.T) {
	t.Parallel()
	response := &WsOrderbook{}
	err := json.Unmarshal([]byte(wsOrderbookData), &response)
	assert.NoError(t, err)
	_, err = ku.UpdateLocalBuffer(response, asset.Spot)
	assert.NoError(t, err)
	err = ku.processOrderbook([]byte(orderbookLevel5PushData), "BTC-USDT")
	assert.NoError(t, err)
	err = ku.wsHandleData([]byte(orderbookLevel5PushData))
	assert.NoError(t, err)
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
					assert.Equal(t, currency.NewPairWithDelimiter("XMR", "BTC", "-"), v.Pair, "symbol")
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
	assert.NoError(t, err)
	err = ku.SeedLocalCache(context.Background(), pair, asset.Margin)
	assert.NoError(t, err)
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
	require.ErrorIs(t, err, asset.ErrNotSupported)
	assets := []asset.Item{asset.Spot, asset.Futures, asset.Margin}
	for x := range assets {
		err = ku.UpdateOrderExecutionLimits(context.Background(), assets[x])
		require.NoError(t, err)

		enabled, err := ku.GetEnabledPairs(assets[x])
		require.NoError(t, err)

		for y := range enabled {
			lim, err := ku.GetOrderExecutionLimits(assets[x], enabled[y])
			require.NoError(t, err, "%v %s %v", err, enabled[y], assets[x])
			assert.NotEmpty(t, lim, "limit cannot be empty")
		}
	}
}

func BenchmarkIntervalToString(b *testing.B) {
	for x := 0; x < b.N; x++ {
		_, err := ku.intervalToString(kline.OneWeek)
		require.NoError(b, err)
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

func TestSpotHFPlaceOrder(t *testing.T) {
	t.Parallel()
	// sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SpotHFPlaceOrder(context.Background(), &PlaceHFParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.SpotHFPlaceOrder(context.Background(), &PlaceHFParam{
		TimeInForce: "GTT",
		Symbol:      currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.BTC},
		OrderType:   "limit",
		Side:        order.Sell.String(),
		Price:       1234,
		Size:        1,
	})
	assert.NoError(t, err)
}

func TestSpotPlaceHFOrderTest(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SpotPlaceHFOrderTest(context.Background(), &PlaceHFParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.SpotPlaceHFOrderTest(context.Background(), &PlaceHFParam{
		TimeInForce: "GTT",
		Symbol:      currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.BTC},
		OrderType:   "limit",
		Side:        order.Sell.String(),
		Price:       1234,
		Size:        1,
	})
	assert.NoError(t, err)
}

func TestSyncPlaceHFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SyncPlaceHFOrder(context.Background(), &PlaceHFParam{})
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = ku.SyncPlaceHFOrder(context.Background(), &PlaceHFParam{
		TimeInForce: "GTT",
		Symbol:      currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.BTC},
		OrderType:   "limit",
		Side:        order.Sell.String(),
		Price:       1234,
		Size:        1,
	})
	assert.NoError(t, err)
}

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.PlaceMultipleOrders(context.Background(), []PlaceHFParam{
		{TimeInForce: "GTT",
			Symbol:    currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.BTC},
			OrderType: "limit",
			Side:      order.Sell.String(),
			Price:     1234,
			Size:      1},
		{
			ClientOrderID: "3d07008668054da6b3cb12e432c2b13a",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.USDT},
		},
		{
			ClientOrderID: "37245dbe6e134b5c97732bfb36cd4a9d",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.USDT},
		},
	})
	assert.NoError(t, err)
}

func TestSyncPlaceMultipleHFOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SyncPlaceMultipleHFOrders(context.Background(), []PlaceHFParam{
		{TimeInForce: "GTT",
			Symbol:    currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.BTC},
			OrderType: "limit",
			Side:      order.Sell.String(),
			Price:     1234,
			Size:      1},
		{
			ClientOrderID: "3d07008668054da6b3cb12e432c2b13a",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.USDT},
		},
		{
			ClientOrderID: "37245dbe6e134b5c97732bfb36cd4a9d",
			Side:          "buy",
			OrderType:     "limit",
			Price:         0.01,
			Size:          1,
			Symbol:        currency.Pair{Base: currency.ETH, Delimiter: "-", Quote: currency.USDT},
		},
	})
	assert.NoError(t, err)
}

func TestModifyHFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.ModifyHFOrder(context.Background(), &ModifyHFOrderParam{})
	assert.ErrorIs(t, err, common.ErrNilPointer)
}

func TestCancelHFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelHFOrder(context.Background(), "630625dbd9180300014c8d52", "BTC-USDT")
	assert.NoError(t, err)
}

func TestSyncCancelHFOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SyncCancelHFOrder(context.Background(), "641d67ea162d47000160bfb8", "BTC-USDT")
	assert.NoError(t, err)
}

func TestSyncCancelHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.SyncCancelHFOrderByClientOrderID(context.Background(), "client-order-id", "BTC-ETH")
	assert.NoError(t, err)
}

func TestCancelHFOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelHFOrderByClientOrderID(context.Background(), "client-order-id", "BTC-ETH")
	assert.NoError(t, err)
}

func TestCancelSpecifiedNumberHFOrdersByOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelSpecifiedNumberHFOrdersByOrderID(context.Background(), "1", "ETH-USDT", 10.0)
	assert.NoError(t, err)
}

func TestCancelAllHFOrdersBySymbol(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelAllHFOrdersBySymbol(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestCancelAllHFOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.CancelAllHFOrders(context.Background())
	assert.NoError(t, err)
}

func TestGetActiveHFOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetActiveHFOrders(context.Background(), "BTC-USDT")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty)
	_, err = ku.GetActiveHFOrders(context.Background(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetSymbolsWithActiveHFOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetSymbolsWithActiveHFOrderList(context.Background())
	assert.NoError(t, err)
}

func TestGetHFCompletedOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetHFCompletedOrderList(context.Background(), "BTC-ETH", "sell", "limit", "", time.Time{}, time.Now(), 0)
	assert.NoError(t, err)
}

func TestGetHFOrderDetailsByOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetHFOrderDetailsByOrderID(context.Background(), "1234567", "BTC-ETH")
	assert.NoError(t, err)
}

func TestGetHFOrderDetailsByClientOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.GetHFOrderDetailsByClientOrderID(context.Background(), "6d539dc614db312", "BTC-ETH")
	assert.NoError(t, err)
}

func TestAutoCancelHFOrderSetting(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku, canManipulateRealOrders)
	_, err := ku.AutoCancelHFOrderSetting(context.Background(), 450, []string{})
	assert.NoError(t, err)
}

func TestAutoCancelHFOrderSettingQuery(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ku)
	_, err := ku.AutoCancelHFOrderSettingQuery(context.Background())
	assert.NoError(t, err)
}

func TestGetHFFilledList(t *testing.T) {
	t.Parallel()
	_, err := ku.GetHFFilledList(context.Background(), "", "BTC-USDT", "sell", "market", "", time.Time{}, time.Now(), 0)
	assert.NoError(t, err)
}
