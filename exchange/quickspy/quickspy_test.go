package quickspy

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestNewQuickSpy(t *testing.T) {
	t.Parallel()
	_, err := NewQuickSpy(nil, nil, false)
	require.ErrorIs(t, err, errNoKey)

	_, err = NewQuickSpy(&CredentialsKey{}, nil, false)
	require.ErrorIs(t, err, errNoFocus)

	_, err = NewQuickSpy(&CredentialsKey{}, []FocusData{{}}, false)
	require.ErrorIs(t, err, ErrUnsetFocusType)

	_, err = NewQuickSpy(&CredentialsKey{}, []FocusData{{Type: OrderBookFocusType, RESTPollTime: -1}}, false)
	require.ErrorIs(t, err, ErrInvalidRESTPollTime)

	_, err = NewQuickSpy(&CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("hi", asset.Binary, currency.NewBTCUSD())}, []FocusData{{Type: OpenInterestFocusType, RESTPollTime: 10}}, false)
	require.ErrorIs(t, err, ErrInvalidAssetForFocusType)

	_, err = NewQuickSpy(&CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("hi", asset.Futures, currency.NewBTCUSD())}, []FocusData{{Type: AccountHoldingsFocusType, RESTPollTime: 10}}, false)
	require.ErrorIs(t, err, ErrCredentialsRequiredForFocusType)

	qs, err := NewQuickSpy(&CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("Binance", asset.Spot, currency.NewBTCUSDT()), Credentials: &account.Credentials{
		Key:    "abc",
		Secret: "123",
	}}, []FocusData{{Type: AccountHoldingsFocusType, RESTPollTime: 10}}, false)
	require.NoError(t, err)
	require.NotNil(t, qs)
}

func mustQuickSpy(t *testing.T, data *FocusData) *QuickSpy {
	t.Helper()
	qs, err := NewQuickSpy(
		&CredentialsKey{
			ExchangeAssetPair: key.NewExchangeAssetPair("Binance", asset.Spot, currency.NewBTCUSDT()),
			Credentials: &account.Credentials{
				Key:    "abc",
				Secret: "123",
			}},
		[]FocusData{*data}, false)
	require.NoError(t, err)
	require.NotNil(t, qs)
	return qs
}

func TestAnyRequiresWebsocket(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10, UseWebsocket: false})
	require.False(t, q.AnyRequiresWebsocket())
	q.Focuses.s[TickerFocusType] = &FocusData{Type: TickerFocusType, RESTPollTime: 10, UseWebsocket: true}
	require.True(t, q.AnyRequiresWebsocket())
}

func TestAnyRequiresAuth(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10})
	require.False(t, q.AnyRequiresAuth())
	q.Focuses.s[AccountHoldingsFocusType] = &FocusData{Type: AccountHoldingsFocusType, RESTPollTime: 10}
	require.True(t, q.AnyRequiresAuth())
}

func TestFocusTypeRequiresWebsocket(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10, UseWebsocket: false})
	require.False(t, q.FocusTypeRequiresWebsocket(TickerFocusType))
	q.Focuses.s[TickerFocusType] = &FocusData{Type: TickerFocusType, RESTPollTime: 10, UseWebsocket: true}
	require.True(t, q.FocusTypeRequiresWebsocket(TickerFocusType))
	require.False(t, q.FocusTypeRequiresWebsocket(OrderBookFocusType))
}

func TestGetAndWaitForFocusByKey(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10})

	_, err := q.GetAndWaitForFocusByKey(TickerFocusType)
	require.ErrorIs(t, err, errFocusDataTimeout)

	var wg sync.WaitGroup
	wg.Go(func() {
		f, err := q.GetFocusByKey(TickerFocusType)
		require.NoError(t, err)
		close(f.HasBeenSuccessfulChan)
	})
	wg.Wait()
	_, err = q.GetAndWaitForFocusByKey(TickerFocusType)
	require.NoError(t, err)

	_, err = q.GetAndWaitForFocusByKey(OrderPlacementFocusType)
	require.ErrorIs(t, err, errKeyNotFound)
}

func TestGetFocusByKey(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10})

	_, err := q.GetFocusByKey(OrderPlacementFocusType)
	require.ErrorIs(t, err, errKeyNotFound)

	f, err := q.GetFocusByKey(TickerFocusType)
	require.NoError(t, err)
	require.NotNil(t, f)
}

func TestSetupExchange(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10})
	err := q.setupExchange()
	require.NoError(t, err)

	q = &QuickSpy{
		Key:                &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("butts", asset.Spot, currency.NewBTCUSDT())},
		shutdown:           make(chan any),
		dataHandlerChannel: make(chan any),
		m:                  new(sync.RWMutex),
		wg:                 sync.WaitGroup{},
		alert:              alert.Notice{},
	}
	err = q.setupExchange()
	require.ErrorIs(t, err, engine.ErrExchangeNotFound)
}

func TestSetupExchangeDefaults(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10})
	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()

	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)

	q.Key.ExchangeAssetPair.Exchange = "butts"
	err = q.setupExchangeDefaults(e, b)
	require.Error(t, err)
}

func TestSetupCurrencyPairs(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10})
	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()
	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)

	err = q.setupCurrencyPairs(b)
	require.NoError(t, err)
	require.NotNil(t, b.CurrencyPairs.Pairs[asset.Spot])
	require.Nil(t, b.CurrencyPairs.Pairs[asset.Futures])

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = b.CurrencyPairs.Pairs[asset.Spot].RequestFormat
	b.CurrencyPairs.ConfigFormat = b.CurrencyPairs.Pairs[asset.Spot].ConfigFormat
	err = q.setupCurrencyPairs(b)
	require.NoError(t, err)
	require.NotNil(t, b.CurrencyPairs.Pairs[asset.Spot])
}

func TestCheckRateLimits(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10})
	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()
	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)

	err = q.checkRateLimits(b)
	require.NoError(t, err)

	b.Requester = nil
	err = q.checkRateLimits(b)
	require.ErrorIs(t, err, errNoRateLimits)
}

func TestSetupWebsocket(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, &FocusData{Type: TickerFocusType, RESTPollTime: 10, UseWebsocket: true})
	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()
	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)

	err = q.setupWebsocket(e, b)
	require.NoError(t, err)
	require.NoError(t, b.Websocket.Shutdown())

	q.Focuses.Upsert(OrderBookFocusType, &FocusData{Type: OrderBookFocusType, RESTPollTime: 10, UseWebsocket: true})
	q.Focuses.Upsert(TickerFocusType, &FocusData{Type: TickerFocusType, RESTPollTime: 10, UseWebsocket: true})
	q.Focuses.Upsert(TradesFocusType, &FocusData{Type: TradesFocusType, RESTPollTime: 10, UseWebsocket: true})
	q.Focuses.Upsert(AccountHoldingsFocusType, &FocusData{Type: AccountHoldingsFocusType, RESTPollTime: 10, UseWebsocket: true})
	err = q.setupWebsocket(e, b)
	require.NoError(t, err)
	require.NoError(t, b.Websocket.Shutdown())

	q.Focuses.Upsert(OrderPlacementFocusType, &FocusData{Type: OrderPlacementFocusType, RESTPollTime: 10, UseWebsocket: true})
	err = q.setupWebsocket(e, b)
	require.ErrorIs(t, err, errNoWebsocketSupportForFocusType)

	b.Websocket = nil
	err = q.setupWebsocket(e, b)
	require.ErrorIs(t, err, common.ErrNilPointer)
}
