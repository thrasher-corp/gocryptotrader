package engine

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

func TestSetupSyncManager(t *testing.T) {
	t.Parallel()
	_, err := SetupSyncManager(nil, nil, nil, false)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	_, err = SetupSyncManager(&config.SyncManagerConfig{}, nil, nil, false)
	assert.ErrorIs(t, err, errNoSyncItemsEnabled)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, nil, nil, false)
	assert.ErrorIs(t, err, errNilExchangeManager)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, &ExchangeManager{}, nil, false)
	assert.ErrorIs(t, err, errNilConfig)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.BTC}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.ErrorIs(t, err, currency.ErrFiatDisplayCurrencyIsNotFiat)

	_, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	m, err := SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.NoError(t, err)

	if m == nil {
		t.Error("expected manager")
	}
}

func TestSyncManagerStart(t *testing.T) {
	t.Parallel()
	m, err := SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, &ExchangeManager{}, &config.RemoteControlConfig{}, true)
	assert.NoError(t, err)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m.exchangeManager = em
	m.config.SynchronizeContinuously = true
	err = m.Start()
	assert.NoError(t, err)

	err = m.Start()
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)

	m = nil
	err = m.Start()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestSyncManagerStop(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	err := m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start()
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)
}

func TestPrintCurrencyFormat(t *testing.T) {
	t.Parallel()
	c := printCurrencyFormat(1337, currency.BTC)
	if c == "" {
		t.Error("expected formatted currency")
	}
}

func TestPrintConvertCurrencyFormat(t *testing.T) {
	t.Parallel()
	c := printConvertCurrencyFormat(1337, currency.BTC, currency.USD)
	if c == "" {
		t.Error("expected formatted currency")
	}
}

func TestPrintTickerSummary(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	m.PrintTickerSummary(&ticker.Price{}, "REST", nil)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	assert.NoError(t, err)

	atomic.StoreInt32(&m.started, 1)
	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewBTCUSDT(),
	}, "REST", nil)
	m.fiatDisplayCurrency = currency.USD
	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.JPY
	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", errors.New("test"))

	m.PrintTickerSummary(&ticker.Price{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", common.ErrNotYetImplemented)
}

func TestPrintOrderbookSummary(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	m.PrintOrderbookSummary(nil, "REST", nil)

	em := NewExchangeManager()
	exch, err := em.NewExchangeByName("Bitstamp")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	m, err = SetupSyncManager(&config.SyncManagerConfig{SynchronizeTrades: true, SynchronizeContinuously: true, FiatDisplayCurrency: currency.USD, PairFormatDisplay: &currency.EMPTYFORMAT}, em, &config.RemoteControlConfig{}, false)
	assert.NoError(t, err)

	atomic.StoreInt32(&m.started, 1)
	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.USD
	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.fiatDisplayCurrency = currency.JPY
	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", nil)

	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", common.ErrNotYetImplemented)

	m.PrintOrderbookSummary(&orderbook.Book{
		Pair: currency.NewPair(currency.AUD, currency.USD),
	}, "REST", errors.New("test"))

	m.PrintOrderbookSummary(nil, "REST", errors.New("test"))
}

func TestWaitForInitialSync(t *testing.T) {
	var m *SyncManager
	err := m.WaitForInitialSync()
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = &SyncManager{}
	err = m.WaitForInitialSync()
	require.ErrorIs(t, err, ErrSubSystemNotStarted)

	m.started = 1
	err = m.WaitForInitialSync()
	require.NoError(t, err)
}

func TestSyncManagerWebsocketUpdate(t *testing.T) {
	t.Parallel()
	var m *SyncManager
	err := m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	require.ErrorIs(t, err, ErrNilSubsystem)

	m = &SyncManager{}
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	require.ErrorIs(t, err, ErrSubSystemNotStarted)

	m.started = 1
	// not started initial sync
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, 1, 47, nil)
	require.NoError(t, err)

	m.initSyncStarted = 1
	// orderbook not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, nil)
	require.NoError(t, err)

	m.config.SynchronizeOrderbook = true
	// ticker not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTicker, nil)
	require.NoError(t, err)

	m.config.SynchronizeTicker = true
	// trades not enabled
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTrade, nil)
	require.NoError(t, err)

	m.config.SynchronizeTrades = true
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, 1336, nil)
	require.ErrorIs(t, err, errUnknownSyncItem)

	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, nil)
	require.ErrorIs(t, err, errCouldNotSyncNewData)

	m.add(key.NewExchangeAssetPair("", asset.Spot, currency.EMPTYPAIR), syncBase{})
	m.initSyncWG.Add(3)
	// orderbook match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemOrderbook, errors.New("test"))
	require.NoError(t, err)

	// ticker match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTicker, errors.New("test"))
	require.NoError(t, err)

	// trades match
	err = m.WebsocketUpdate("", currency.EMPTYPAIR, asset.Spot, SyncItemTrade, errors.New("test"))
	require.NoError(t, err)
}
