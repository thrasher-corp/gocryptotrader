package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
)

var oManager orderManager
var setupRan bool

func OrdersSetup(t *testing.T) {
	if !setupRan {
		SetupTest(t)
		if oManager.Started() {
			t.Fatal("Order manager already started")
		}
		err := oManager.Start()
		if !oManager.Started() {
			t.Fatal("Order manager not started")
		}
		if err != nil {
			t.Fatal(err)
		}
		setupRan = true
	}
}

func TestOrdersGet(t *testing.T) {
	OrdersSetup(t)
	if oManager.orderStore.get() == nil {
		t.Error("orderStore not established")
	}
}

func TestOrdersAdd(t *testing.T) {
	OrdersSetup(t)
	err := oManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err != nil {
		t.Error(err)
	}
	err = oManager.orderStore.Add(&order.Detail{
		Exchange: "testTest",
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error from non existent exchange")
	}

	err = oManager.orderStore.Add(nil)
	if err == nil {
		t.Error("Expected error from nil order")
	}

	err = oManager.orderStore.Add(&order.Detail{
		Exchange: testExchange,
		ID:       "TestOrdersAdd",
	})
	if err == nil {
		t.Error("Expected error re-adding order")
	}
}

func TestGetByInternalOrderID(t *testing.T) {
	OrdersSetup(t)
	err := oManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByInternalOrderID",
		InternalOrderID: "internalTest",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := oManager.orderStore.GetByInternalOrderID("internalTest")
	if err != nil {
		t.Error(err)
	}
	if o == nil {
		t.Fatal("Expected a matching order")
	}
	if o.ID != "TestGetByInternalOrderID" {
		t.Error("Expected to retrieve order")
	}

	_, err = oManager.orderStore.GetByInternalOrderID("NoOrder")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}
}

func TestGetByExchangeAndID(t *testing.T) {
	OrdersSetup(t)
	err := oManager.orderStore.Add(&order.Detail{
		Exchange:        testExchange,
		ID:              "TestGetByExchangeAndID",
		InternalOrderID: "internalTest",
	})
	if err != nil {
		t.Error(err)
	}

	o, err := oManager.orderStore.GetByExchangeAndID(testExchange, "TestGetByExchangeAndID")
	if err != nil {
		t.Error(err)
	}
	if o.ID != "TestGetByExchangeAndID" {
		t.Error("Expected to retrieve order")
	}

	o, err = oManager.orderStore.GetByExchangeAndID("", "TestGetByExchangeAndID")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}

	o, err = oManager.orderStore.GetByExchangeAndID(testExchange, "")
	if err != ErrOrderFourOhFour {
		t.Error(err)
	}
}

func TestExistsWithLock(t *testing.T) {
	OrdersSetup(t)
	oManager.orderStore.exists(nil)
	oManager.orderStore.existsWithLock(nil)
	o := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExistsWithLock",
	}
	err := oManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}
	b := oManager.orderStore.existsWithLock(o)
	if !b {
		t.Error("Expected true")
	}
	o2 := &order.Detail{
		Exchange: testExchange,
		ID:       "TestExistsWithLock2",
	}
	go oManager.orderStore.existsWithLock(o)
	go oManager.orderStore.Add(o2)
	go oManager.orderStore.existsWithLock(o)
}

func TestCancelOrder(t *testing.T) {
	OrdersSetup(t)
	exch := GetExchangeByName(testExchange)
	exch = &FakeExchange{
		Exchange: testExchange,
	}
	Bot.Exchanges[0] = exch
	o := &order.Detail{
		Exchange:        testExchange,
		ID:              "TestCancelOrder",
		InternalOrderID: "internalTest",
		Status:          order.New,
	}
	err := oManager.orderStore.Add(o)
	if err != nil {
		t.Error(err)
	}
	cancel := &order.Cancel{
		Exchange:  testExchange,
		ID:        "TestCancelOrder",
		Side:      order.Sell,
		Status:    order.New,
		AssetType: asset.Spot,
		Date:      time.Now(),
		Pair:      currency.NewPairFromString("BTCUSD"),
	}
	err = oManager.Cancel(cancel)
	if err != nil {
		t.Error(err)
	}

	if o.Status != order.Cancelled {
		t.Error("Failed to cancel")
	}
}

// FakeExchange is used to override IBotExchange responses in tests
// In this context, we don't care what FakeExchange does as we're testing
// the engine package
type FakeExchange struct {
	Exchange string
}

func (h *FakeExchange) Setup(exch *config.ExchangeConfig) error { return nil }
func (h *FakeExchange) Start(wg *sync.WaitGroup)                {}
func (h *FakeExchange) SetDefaults()                            {}
func (h *FakeExchange) GetName() string                         { return testExchange }
func (h *FakeExchange) IsEnabled() bool                         { return true }
func (h *FakeExchange) SetEnabled(bool)                         {}
func (h *FakeExchange) FetchTicker(currency currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	return nil, nil
}
func (h *FakeExchange) UpdateTicker(currency currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	return nil, nil
}
func (h *FakeExchange) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	return nil, nil
}
func (h *FakeExchange) UpdateOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	return nil, nil
}
func (h *FakeExchange) FetchTradablePairs(assetType asset.Item) ([]string, error) { return nil, nil }
func (h *FakeExchange) UpdateTradablePairs(forceUpdate bool) error                { return nil }
func (h *FakeExchange) GetEnabledPairs(assetType asset.Item) currency.Pairs       { return currency.Pairs{} }
func (h *FakeExchange) GetAvailablePairs(assetType asset.Item) currency.Pairs     { return currency.Pairs{} }
func (h *FakeExchange) GetAccountInfo() (exchange.AccountInfo, error) {
	return exchange.AccountInfo{}, nil
}
func (h *FakeExchange) GetAuthenticatedAPISupport(endpoint uint8) bool { return true }
func (h *FakeExchange) SetPairs(pairs currency.Pairs, assetType asset.Item, enabled bool) error {
	return nil
}
func (h *FakeExchange) GetAssetTypes() asset.Items { return asset.Items{asset.Spot} }
func (h *FakeExchange) GetExchangeHistory(currencyPair currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, nil
}
func (h *FakeExchange) SupportsAutoPairUpdates() bool                                 { return true }
func (h *FakeExchange) SupportsRESTTickerBatchUpdates() bool                          { return true }
func (h *FakeExchange) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) { return 0, nil }
func (h *FakeExchange) GetLastPairsUpdateTime() int64                                 { return 0 }
func (h *FakeExchange) GetWithdrawPermissions() uint32                                { return 0 }
func (h *FakeExchange) FormatWithdrawPermissions() string                             { return "" }
func (h *FakeExchange) SupportsWithdrawPermissions(permissions uint32) bool           { return true }
func (h *FakeExchange) GetFundingHistory() ([]exchange.FundHistory, error)            { return nil, nil }
func (h *FakeExchange) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	return order.SubmitResponse{}, nil
}
func (h *FakeExchange) ModifyOrder(action *order.Modify) (string, error) { return "", nil }
func (h *FakeExchange) CancelOrder(order *order.Cancel) error            { return nil }
func (h *FakeExchange) CancelAllOrders(orders *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, nil
}
func (h *FakeExchange) GetOrderInfo(orderID string) (order.Detail, error) { return order.Detail{}, nil }
func (h *FakeExchange) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", nil
}
func (h *FakeExchange) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, nil
}
func (h *FakeExchange) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, nil
}
func (h *FakeExchange) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.CryptoRequest) (string, error) {
	return "", nil
}
func (h *FakeExchange) WithdrawFiatFunds(withdrawRequest *withdraw.FiatRequest) (string, error) {
	return "", nil
}
func (h *FakeExchange) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.FiatRequest) (string, error) {
	return "", nil
}
func (h *FakeExchange) SetHTTPClientUserAgent(ua string)            {}
func (h *FakeExchange) GetHTTPClientUserAgent() string              { return "" }
func (h *FakeExchange) SetClientProxyAddress(addr string) error     { return nil }
func (h *FakeExchange) SupportsWebsocket() bool                     { return true }
func (h *FakeExchange) SupportsREST() bool                          { return true }
func (h *FakeExchange) IsWebsocketEnabled() bool                    { return true }
func (h *FakeExchange) GetWebsocket() (*wshandler.Websocket, error) { return nil, nil }
func (h *FakeExchange) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return nil
}
func (h *FakeExchange) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return nil
}
func (h *FakeExchange) AuthenticateWebsocket() error { return nil }
func (h *FakeExchange) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, nil
}
func (h *FakeExchange) GetDefaultConfig() (*config.ExchangeConfig, error) { return nil, nil }
func (h *FakeExchange) GetBase() *exchange.Base                           { return nil }
func (h *FakeExchange) SupportsAsset(assetType asset.Item) bool           { return true }
