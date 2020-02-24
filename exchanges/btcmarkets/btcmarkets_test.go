package btcmarkets

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var b BTCMarkets

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	BTCAUD                  = "BTC-AUD"
	LTCAUD                  = "LTC-AUD"
	ETHAUD                  = "ETH-AUD"
	fakePair                = "Fake-USDT"
	bid                     = "bid"
)

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	bConfig, err := cfg.GetExchangeConfig("BTC Markets")
	if err != nil {
		log.Fatal(err)
	}
	bConfig.API.Credentials.Key = apiKey
	bConfig.API.Credentials.Secret = apiSecret
	bConfig.API.AuthenticatedSupport = true

	err = b.Setup(bConfig)
	if err != nil {
		log.Fatal(err)
	}
	b.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	b.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()

	err = b.ValidateCredentials()
	if err != nil {
		fmt.Println("API credentials are invalid:", err)
		b.API.AuthenticatedSupport = false
		b.API.AuthenticatedWebsocketSupport = false
	}

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return b.AllowAuthenticatedRequest()
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets()
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(BTCAUD)
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades(BTCAUD, 0, 0, 5)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook(BTCAUD, 2)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetMarketCandles(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketCandles(BTCAUD, "", "", "", 0, 0, 5)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	temp := currency.NewPairsFromStrings([]string{LTCAUD, BTCAUD})
	_, err := b.GetTickers(temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMultipleOrderbooks(t *testing.T) {
	t.Parallel()
	temp := []string{BTCAUD, LTCAUD, ETHAUD}
	_, err := b.GetMultipleOrderbooks(temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime()
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetAccountBalance()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradingFees(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTradingFees()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTradeHistory(ETHAUD, "", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTradeHistory(BTCAUD, "", -1, -1, 1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTradeHistory(fakePair, "", -1, -1, -1)
	if err == nil {
		t.Error("expected an error due to invalid trading pair")
	}
}

func TestGetTradeByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTradeByID("4712043732")
	if err != nil {
		t.Error(err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := b.NewOrder(BTCAUD, 100, 1, limit, bid, 0, 0, "", true, "", "")
	if err != nil {
		t.Error(err)
	}
	_, err = b.NewOrder(BTCAUD, 100, 1, "invalid", bid, 0, 0, "", true, "", "")
	if err == nil {
		t.Error("expected an error due to invalid ordertype")
	}
	_, err = b.NewOrder(BTCAUD, 100, 1, limit, "invalid", 0, 0, "", true, "", "")
	if err == nil {
		t.Error("expected an error due to invalid orderside")
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetOrders("", -1, -1, 2, false)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrders(LTCAUD, -1, -1, -1, true)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	temp := []string{BTCAUD, LTCAUD}
	_, err := b.CancelAllOpenOrdersByPairs(temp)
	if err != nil {
		t.Error(err)
	}
	temp = []string{BTCAUD, fakePair}
	_, err = b.CancelAllOpenOrdersByPairs(temp)
	if err == nil {
		t.Error("expected an error due to invalid marketID")
	}
}

func TestFetchOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.FetchOrder("4477045999")
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchOrder("696969")
	if err == nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := b.RemoveOrder("")
	if err != nil {
		t.Error(err)
	}
}

func TestListWithdrawals(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListWithdrawals(-1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetWithdrawal("4477381751")
	if err != nil {
		t.Error(err)
	}
}

func TestListDeposits(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListDeposits(-1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeposit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetDeposit("4476769607")
	if err != nil {
		t.Error(err)
	}
}

func TestListTransfers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListTransfers(-1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTransfer("4476769607")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTransfer("6969696")
	if err == nil {
		t.Error("expected an error due to invalid transferID")
	}
}

func TestFetchDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.FetchDepositAddress("LTC", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchDepositAddress(fakePair, -1, -1, -1)
	if err != nil {
		t.Error("expected an error due to invalid assetID")
	}
}

func TestGetWithdrawalFees(t *testing.T) {
	t.Parallel()
	_, err := b.GetWithdrawalFees()
	if err != nil {
		t.Error(err)
	}
}

func TestListAssets(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListAssets()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTransactions("", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewReport(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.CreateNewReport("TransactionReport", "json")
	if err != nil {
		t.Error(err)
	}
}

func TestGetReport(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetReport("1kv38epne5v7lek9f18m60idg6")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestWithdaw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := b.RequestWithdraw("BTC", 1, "sdjflajdslfjld", "", "", "", "")
	if err == nil {
		t.Error("expected an error due to invalid toAddress")
	}
}

func TestBatchPlaceCancelOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	var temp []PlaceBatch
	o := PlaceBatch{
		MarketID:  BTCAUD,
		Amount:    11000,
		Price:     1,
		OrderType: order.Limit.String(),
		Side:      bid,
	}
	_, err := b.BatchPlaceCancelOrders(nil, append(temp, o))
	if err != nil {
		t.Error(err)
	}
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	temp := []string{"4477045999", "4477381751", "4476769607"}
	_, err := b.GetBatchTrades(temp)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	temp := []string{"4477045999", "4477381751", "4477381751"}
	_, err := b.CancelBatchOrders(temp)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.FetchAccountInfo()
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}

	_, err := b.GetOrderHistory(&order.GetOrdersRequest{
		Side: order.Buy,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.AUD.String(), "-")
	_, err := b.UpdateOrderbook(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.AUD.String(), "-")
	_, err := b.UpdateTicker(cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}

	_, err := b.GetActiveOrders(&order.GetOrdersRequest{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsOrderNotification(t *testing.T) {
	status := []string{"ORDER_INSERTED", "ORDER_CANCELLED", "TRIGGER_INSERTED", "ORDER_FULL_TRANSACTED", "ORDER_PARTIALLY_TRANSACTED", "INSUFFICIENT_BALANCE", "TRIGGER_ACTIVATED", "MARKET_UNAVAILABLE"}
	for i := range status {
		pressXToJSON := []byte(`{"topic": "notificationApi","data": [{"symbol": "BTC-USD","orderID": "1234","orderMode": "MODE_BUY","orderType": "TYPE_LIMIT","price": "1","size": "1","status": "` + status[i] + `","timestamp": "1580349090693","type": "STOP","triggerPrice": "1"}]}`)
		err := b.wsHandleData(pressXToJSON)
		if err != nil {
			t.Error(err)
		}
	}
}
