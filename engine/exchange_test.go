package engine

import (
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
)

// addPassingFakeExchange adds an exchange to engine tests where all funcs return a positive result
func addPassingFakeExchange() {
	testExch := GetExchangeByName(testExchange)
	base := testExch.GetBase()
	Bot.Config.Exchanges = append(Bot.Config.Exchanges, config.ExchangeConfig{
		Name:    fakePassExchange,
		Enabled: true,
		Verbose: false,
	})

	Bot.exchangeManager.add(&FakePassingExchange{
		Base: exchange.Base{
			Name:                          fakePassExchange,
			Enabled:                       true,
			LoadedByConfig:                true,
			SkipAuthCheck:                 true,
			API:                           base.API,
			Features:                      base.Features,
			HTTPTimeout:                   base.HTTPTimeout,
			HTTPUserAgent:                 base.HTTPUserAgent,
			HTTPRecording:                 base.HTTPRecording,
			HTTPDebugging:                 base.HTTPDebugging,
			WebsocketResponseCheckTimeout: base.WebsocketResponseCheckTimeout,
			WebsocketResponseMaxLimit:     base.WebsocketResponseMaxLimit,
			WebsocketOrderbookBufferLimit: base.WebsocketOrderbookBufferLimit,
			Websocket:                     base.Websocket,
			Requester:                     base.Requester,
			Config:                        base.Config,
		},
	})
}

func CleanupTest(t *testing.T) {
	if GetExchangeByName(testExchange) != nil {
		err := UnloadExchange(testExchange)
		if err != nil {
			t.Fatalf("CleanupTest: Failed to unload exchange: %s",
				err)
		}
	}
	if GetExchangeByName(fakePassExchange) != nil {
		err := UnloadExchange(fakePassExchange)
		if err != nil {
			t.Fatalf("CleanupTest: Failed to unload exchange: %s",
				err)
		}
	}
}

func TestExchangeManagerAdd(t *testing.T) {
	t.Parallel()
	var e exchangeManager
	bitfinex := new(bitfinex.Bitfinex)
	bitfinex.SetDefaults()
	e.add(bitfinex)
	if exch := e.getExchanges(); exch[0].GetName() != "Bitfinex" {
		t.Error("unexpected exchange name")
	}
}

func TestExchangeManagerGetExchanges(t *testing.T) {
	t.Parallel()
	var e exchangeManager
	if exchanges := e.getExchanges(); exchanges != nil {
		t.Error("unexpected value")
	}
	bitfinex := new(bitfinex.Bitfinex)
	bitfinex.SetDefaults()
	e.add(bitfinex)
	if exch := e.getExchanges(); exch[0].GetName() != "Bitfinex" {
		t.Error("unexpected exchange name")
	}
}

func TestExchangeManagerRemoveExchange(t *testing.T) {
	t.Parallel()
	var e exchangeManager
	if err := e.removeExchange("Bitfinex"); err != ErrNoExchangesLoaded {
		t.Error("no exchanges should be loaded")
	}
	bitfinex := new(bitfinex.Bitfinex)
	bitfinex.SetDefaults()
	e.add(bitfinex)
	if err := e.removeExchange(testExchange); err != ErrExchangeNotFound {
		t.Error("Bitstamp exchange should return an error")
	}
	if err := e.removeExchange("BiTFiNeX"); err != nil {
		t.Error("exchange should have been removed")
	}
	if e.Len() != 0 {
		t.Error("exchange manager len should be 0")
	}
}

func TestCheckExchangeExists(t *testing.T) {
	SetupTestHelpers(t)

	if GetExchangeByName(testExchange) == nil {
		t.Errorf("TestGetExchangeExists: Unable to find exchange")
	}

	if GetExchangeByName("Asdsad") != nil {
		t.Errorf("TestGetExchangeExists: Non-existent exchange found")
	}

	CleanupTest(t)
}

func TestGetExchangeByName(t *testing.T) {
	SetupTestHelpers(t)

	exch := GetExchangeByName(testExchange)
	if exch == nil {
		t.Errorf("TestGetExchangeByName: Failed to get exchange")
	}

	if !exch.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	exch.SetEnabled(false)
	bfx := GetExchangeByName(testExchange)
	if bfx.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	if exch.GetName() != testExchange {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	exch = GetExchangeByName("Asdasd")
	if exch != nil {
		t.Errorf("TestGetExchangeByName: Non-existent exchange found")
	}

	CleanupTest(t)
}

func TestUnloadExchange(t *testing.T) {
	SetupTestHelpers(t)

	err := UnloadExchange("asdf")
	if err.Error() != "exchange asdf not found" {
		t.Errorf("TestUnloadExchange: Incorrect result: %s",
			err)
	}

	err = UnloadExchange(testExchange)
	if err != nil {
		t.Errorf("TestUnloadExchange: Failed to get exchange. %s",
			err)
	}

	err = UnloadExchange(fakePassExchange)
	if err != nil {
		t.Errorf("TestUnloadExchange: Failed to get exchange. %s",
			err)
	}

	err = UnloadExchange(testExchange)
	if err != ErrNoExchangesLoaded {
		t.Errorf("TestUnloadExchange: Incorrect result: %s",
			err)
	}

	CleanupTest(t)
}

func TestDryRunParamInteraction(t *testing.T) {
	SetupTestHelpers(t)

	// Load bot as per normal, dry run and verbose for Bitfinex should be
	// disabled
	exchCfg, err := Bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}

	if Bot.Settings.EnableDryRun ||
		exchCfg.Verbose {
		t.Error("dryrun and verbose should have been disabled")
	}

	// Simulate overiding default settings and ensure that enabling exchange
	// verbose mode will be set on Bitfinex
	if err = UnloadExchange(testExchange); err != nil {
		t.Error(err)
	}

	Bot.Settings.CheckParamInteraction = true
	Bot.Settings.EnableExchangeVerbose = true
	if err = LoadExchange(testExchange, false, nil); err != nil {
		t.Error(err)
	}

	exchCfg, err = Bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}

	if !Bot.Settings.EnableDryRun ||
		!exchCfg.Verbose {
		t.Error("dryrun and verbose should have been enabled")
	}

	if err = UnloadExchange(testExchange); err != nil {
		t.Error(err)
	}

	// Now set dryrun mode to false (via flagset and the previously enabled
	// setting), enable exchange verbose mode and verify that verbose mode
	// will be set on Bitfinex
	Bot.Settings.EnableDryRun = false
	Bot.Settings.CheckParamInteraction = true
	Bot.Settings.EnableExchangeVerbose = true
	flagSet["dryrun"] = true
	if err = LoadExchange(testExchange, false, nil); err != nil {
		t.Error(err)
	}

	exchCfg, err = Bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}

	if Bot.Settings.EnableDryRun ||
		!exchCfg.Verbose {
		t.Error("dryrun should be false and verbose should be true")
	}
}

// FakePassingExchange is used to override IBotExchange responses in tests
// In this context, we don't care what FakePassingExchange does as we're testing
// the engine package
type FakePassingExchange struct {
	exchange.Base
}

func (h *FakePassingExchange) Setup(_ *config.ExchangeConfig) error { return nil }
func (h *FakePassingExchange) Start(_ *sync.WaitGroup)              {}
func (h *FakePassingExchange) SetDefaults()                         {}
func (h *FakePassingExchange) GetName() string                      { return fakePassExchange }
func (h *FakePassingExchange) IsEnabled() bool                      { return true }
func (h *FakePassingExchange) SetEnabled(bool)                      {}
func (h *FakePassingExchange) ValidateCredentials() error           { return nil }

func (h *FakePassingExchange) FetchTicker(_ currency.Pair, _ asset.Item) (*ticker.Price, error) {
	return nil, nil
}
func (h *FakePassingExchange) UpdateTicker(_ currency.Pair, _ asset.Item) (*ticker.Price, error) {
	return nil, nil
}
func (h *FakePassingExchange) FetchOrderbook(_ currency.Pair, _ asset.Item) (*orderbook.Base, error) {
	return nil, nil
}
func (h *FakePassingExchange) UpdateOrderbook(_ currency.Pair, _ asset.Item) (*orderbook.Base, error) {
	return nil, nil
}
func (h *FakePassingExchange) FetchTradablePairs(_ asset.Item) ([]string, error) {
	return nil, nil
}
func (h *FakePassingExchange) UpdateTradablePairs(_ bool) error { return nil }

func (h *FakePassingExchange) GetEnabledPairs(_ asset.Item) currency.Pairs {
	return currency.Pairs{}
}
func (h *FakePassingExchange) GetAvailablePairs(_ asset.Item) currency.Pairs {
	return currency.Pairs{}
}
func (h *FakePassingExchange) FetchAccountInfo() (account.Holdings, error) {
	return account.Holdings{}, nil
}

func (h *FakePassingExchange) UpdateAccountInfo() (account.Holdings, error) {
	return account.Holdings{}, nil
}
func (h *FakePassingExchange) GetAuthenticatedAPISupport(_ uint8) bool { return true }
func (h *FakePassingExchange) SetPairs(_ currency.Pairs, _ asset.Item, _ bool) error {
	return nil
}
func (h *FakePassingExchange) GetAssetTypes() asset.Items { return asset.Items{asset.Spot} }
func (h *FakePassingExchange) GetExchangeHistory(_ currency.Pair, _ asset.Item) ([]exchange.TradeHistory, error) {
	return nil, nil
}
func (h *FakePassingExchange) SupportsAutoPairUpdates() bool        { return true }
func (h *FakePassingExchange) SupportsRESTTickerBatchUpdates() bool { return true }
func (h *FakePassingExchange) GetFeeByType(_ *exchange.FeeBuilder) (float64, error) {
	return 0, nil
}
func (h *FakePassingExchange) GetLastPairsUpdateTime() int64                      { return 0 }
func (h *FakePassingExchange) GetWithdrawPermissions() uint32                     { return 0 }
func (h *FakePassingExchange) FormatWithdrawPermissions() string                  { return "" }
func (h *FakePassingExchange) SupportsWithdrawPermissions(_ uint32) bool          { return true }
func (h *FakePassingExchange) GetFundingHistory() ([]exchange.FundHistory, error) { return nil, nil }
func (h *FakePassingExchange) SubmitOrder(_ *order.Submit) (order.SubmitResponse, error) {
	return order.SubmitResponse{
		IsOrderPlaced: true,
		FullyMatched:  true,
		OrderID:       "FakePassingExchangeOrder",
	}, nil
}
func (h *FakePassingExchange) ModifyOrder(_ *order.Modify) (string, error) { return "", nil }
func (h *FakePassingExchange) CancelOrder(_ *order.Cancel) error           { return nil }
func (h *FakePassingExchange) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, nil
}
func (h *FakePassingExchange) GetOrderInfo(_ string) (order.Detail, error) {
	return order.Detail{}, nil
}
func (h *FakePassingExchange) GetDepositAddress(_ currency.Code, _ string) (string, error) {
	return "", nil
}
func (h *FakePassingExchange) GetOrderHistory(_ *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, nil
}
func (h *FakePassingExchange) GetActiveOrders(_ *order.GetOrdersRequest) ([]order.Detail, error) {
	return []order.Detail{
		{
			Price:     1337,
			Amount:    1337,
			Exchange:  fakePassExchange,
			ID:        "fakeOrder",
			Type:      order.Market,
			Side:      order.Buy,
			Status:    order.Active,
			AssetType: asset.Spot,
			Date:      time.Now(),
			Pair:      currency.NewPairFromString("BTCUSD"),
		},
	}, nil
}
func (h *FakePassingExchange) WithdrawCryptocurrencyFunds(_ *withdraw.CryptoRequest) (string, error) {
	return "", nil
}
func (h *FakePassingExchange) WithdrawFiatFunds(_ *withdraw.FiatRequest) (string, error) {
	return "", nil
}
func (h *FakePassingExchange) WithdrawFiatFundsToInternationalBank(_ *withdraw.FiatRequest) (string, error) {
	return "", nil
}
func (h *FakePassingExchange) SetHTTPClientUserAgent(_ string)             {}
func (h *FakePassingExchange) GetHTTPClientUserAgent() string              { return "" }
func (h *FakePassingExchange) SetClientProxyAddress(_ string) error        { return nil }
func (h *FakePassingExchange) SupportsWebsocket() bool                     { return true }
func (h *FakePassingExchange) SupportsREST() bool                          { return true }
func (h *FakePassingExchange) IsWebsocketEnabled() bool                    { return true }
func (h *FakePassingExchange) GetWebsocket() (*wshandler.Websocket, error) { return nil, nil }
func (h *FakePassingExchange) SubscribeToWebsocketChannels(_ []wshandler.WebsocketChannelSubscription) error {
	return nil
}
func (h *FakePassingExchange) UnsubscribeToWebsocketChannels(_ []wshandler.WebsocketChannelSubscription) error {
	return nil
}
func (h *FakePassingExchange) AuthenticateWebsocket() error { return nil }
func (h *FakePassingExchange) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, nil
}
func (h *FakePassingExchange) GetDefaultConfig() (*config.ExchangeConfig, error) { return nil, nil }
func (h *FakePassingExchange) GetBase() *exchange.Base                           { return nil }
func (h *FakePassingExchange) SupportsAsset(_ asset.Item) bool                   { return true }
func (h *FakePassingExchange) GetHistoricCandles(_ currency.Pair, _, _ int64) ([]exchange.Candle, error) {
	return []exchange.Candle{}, nil
}
func (h *FakePassingExchange) DisableRateLimiter() error { return nil }
func (h *FakePassingExchange) EnableRateLimiter() error  { return nil }
