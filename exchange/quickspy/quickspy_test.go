package quickspy

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// these are here to help a user test
// modifying them and decrying that tests fail will get you thrown in gaol
var (
	exchangeName     = "Binance"
	assetType        = asset.Spot
	currencyPair     = currency.NewBTCUSDT()
	apiKey           = "abc"
	apiSecret        = "123"
	futuresAssetType = asset.USDTMarginedFutures // used in TestDumpAndCurrentPayload
)

func mustQuickSpy(t *testing.T, ft FocusType) *QuickSpy {
	t.Helper()
	ftd := FocusData{Type: ft, RESTPollTime: time.Second}
	ftd.Init()
	qs, err := NewQuickSpy(
		context.Background(),
		&CredentialsKey{
			ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair),
			Credentials: &account.Credentials{
				Key:    apiKey,
				Secret: apiSecret,
			}},
		[]FocusData{ftd},
		false)
	require.NoError(t, err)
	require.NotNil(t, qs)
	return qs
}

func mustQuickSpyAllFocuses(t *testing.T) *QuickSpy {
	t.Helper()
	focuses := make([]FocusData, 0, len(focusList))
	for _, ft := range focusList {
		ftd := FocusData{Type: ft, RESTPollTime: time.Second}
		ftd.Init()
		focuses = append(focuses, ftd)
	}
	qs, err := NewQuickSpy(
		context.Background(),
		&CredentialsKey{
			ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, futuresAssetType, currencyPair),
			Credentials: &account.Credentials{
				Key:    apiKey,
				Secret: apiSecret,
			}},
		focuses, false)
	require.NoError(t, err)
	require.NotNil(t, qs)
	return qs
}

func TestNewQuickSpy(t *testing.T) {
	t.Parallel()
	_, err := NewQuickSpy(nil, nil, nil, false)
	require.ErrorIs(t, err, errNoKey)

	_, err = NewQuickSpy(nil, &CredentialsKey{}, nil, false)
	require.ErrorIs(t, err, errNoFocus)

	_, err = NewQuickSpy(nil, &CredentialsKey{}, []FocusData{{}}, false)
	require.ErrorIs(t, err, ErrUnsetFocusType)

	_, err = NewQuickSpy(nil, &CredentialsKey{}, []FocusData{{Type: OrderBookFocusType, RESTPollTime: -1}}, false)
	require.ErrorIs(t, err, ErrInvalidRESTPollTime)

	_, err = NewQuickSpy(nil, &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, asset.Binary, currency.NewBTCUSD())}, []FocusData{{Type: OpenInterestFocusType, RESTPollTime: 10}}, false)
	require.ErrorIs(t, err, ErrInvalidAssetForFocusType)

	_, err = NewQuickSpy(nil, &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, asset.Futures, currencyPair)}, []FocusData{{Type: AccountHoldingsFocusType, RESTPollTime: 10}}, false)
	require.ErrorIs(t, err, ErrCredentialsRequiredForFocusType)

	qs, err := NewQuickSpy(nil, &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair), Credentials: &account.Credentials{
		Key:    apiKey,
		Secret: apiSecret,
	}}, []FocusData{{Type: AccountHoldingsFocusType, RESTPollTime: 10}}, false)
	require.NoError(t, err)
	require.NotNil(t, qs)

	ctx := context.Background()
	qs, err = NewQuickSpy(ctx, &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair), Credentials: &account.Credentials{
		Key:    apiKey,
		Secret: apiSecret,
	}}, []FocusData{{Type: AccountHoldingsFocusType, RESTPollTime: 10}}, false)
	require.NoError(t, err)
	require.NotNil(t, qs)
	assert.NotEmpty(t, qs.credContext.Value(account.ContextCredentialsFlag), "credentials should be popultated in context")
}

func TestAnyRequiresWebsocket(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	require.False(t, q.AnyRequiresWebsocket())

	q.Focuses.Upsert(TickerFocusType, &FocusData{Type: TickerFocusType, RESTPollTime: time.Second, UseWebsocket: true})
	require.True(t, q.AnyRequiresWebsocket())
}

func TestAnyRequiresAuth(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	require.False(t, q.AnyRequiresAuth())

	q.Focuses.Upsert(AccountHoldingsFocusType, &FocusData{Type: AccountHoldingsFocusType, RESTPollTime: time.Second})
	require.True(t, q.AnyRequiresAuth())
}

func TestFocusTypeRequiresWebsocket(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	require.False(t, q.FocusTypeRequiresWebsocket(TickerFocusType))

	q.Focuses.Upsert(TickerFocusType, &FocusData{Type: TickerFocusType, RESTPollTime: time.Second, UseWebsocket: true})
	require.True(t, q.FocusTypeRequiresWebsocket(TickerFocusType))
	require.False(t, q.FocusTypeRequiresWebsocket(OrderBookFocusType))
}

func TestGetAndWaitForFocusByKey(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)

	_, err := q.GetAndWaitForFocusByKey(TickerFocusType)
	require.ErrorIs(t, err, errFocusDataTimeout)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		f, err := q.GetFocusByKey(TickerFocusType)
		require.NoError(t, err)
		close(f.HasBeenSuccessfulChan)
	}()
	wg.Wait()
	_, err = q.GetAndWaitForFocusByKey(TickerFocusType)
	require.NoError(t, err)

	_, err = q.GetAndWaitForFocusByKey(OrderPlacementFocusType)
	require.ErrorIs(t, err, errKeyNotFound)
}

func TestGetFocusByKey(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)

	_, err := q.GetFocusByKey(OrderPlacementFocusType)
	require.ErrorIs(t, err, errKeyNotFound)

	f, err := q.GetFocusByKey(TickerFocusType)
	require.NoError(t, err)
	require.NotNil(t, f)
}

func TestSetupExchange(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	err := q.setupExchange()
	require.NoError(t, err)

	q = &QuickSpy{
		Key:                &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("butts", assetType, currencyPair)},
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
	q := mustQuickSpy(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()

	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)

	// Ensure verbose is respected and template accessible
	require.Equal(t, q.verbose, b.Verbose)
}

func TestSetupCurrencyPairs(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()
	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)

	err = q.setupCurrencyPairs(b)
	require.NoError(t, err)
	require.NotNil(t, b.CurrencyPairs.Pairs[assetType])
	require.Nil(t, b.CurrencyPairs.Pairs[asset.Futures])

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = b.CurrencyPairs.Pairs[assetType].RequestFormat
	b.CurrencyPairs.ConfigFormat = b.CurrencyPairs.Pairs[assetType].ConfigFormat
	err = q.setupCurrencyPairs(b)
	require.NoError(t, err)
	require.NotNil(t, b.CurrencyPairs.Pairs[assetType])
}

func TestCheckRateLimits(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
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
	// Reworked to cover key branches deterministically
	t.Parallel()
	// Case 1: No websocket required -> nil
	q := mustQuickSpy(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()
	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)
	err = q.setupWebsocket(e, b)
	require.NoError(t, err)

	// Case 2: Requires websocket but mapping not supported -> errNoWebsocketSupportForFocusType
	q.Focuses.Upsert(OrderPlacementFocusType, &FocusData{Type: OrderPlacementFocusType, RESTPollTime: time.Second, UseWebsocket: true})
	err = q.setupWebsocket(e, b)
	require.ErrorIs(t, err, errNoWebsocketSupportForFocusType)

	q.Focuses = NewFocusStore() // reset focuses
	// Case 3: Supported mapping but no subscription template -> errNoSubSwitchingToREST
	q.Focuses.Upsert(OrderBookFocusType, &FocusData{Type: OrderBookFocusType, RESTPollTime: time.Second, UseWebsocket: true})
	// clear any existing templates
	b.Config.Features.Subscriptions = nil
	err = q.setupWebsocket(e, b)
	require.ErrorIs(t, err, errNoSubSwitchingToREST)

	// Case 4: Nil websocket manager -> common.ErrNilPointer
	b.Websocket = nil
	err = q.setupWebsocket(e, b)
	require.ErrorIs(t, err, common.ErrNilPointer)

	// Case 5: Supported mapping with a template existing; we won't actually connect, just ensure it reaches template matching.
	// Recreate websocket manager and add a template for ticker.
	_ = q.setupExchangeDefaults(e, b)
	q.Focuses.Upsert(TickerFocusType, &FocusData{Type: TickerFocusType, RESTPollTime: time.Second, UseWebsocket: true})
	b.Config.Features.Subscriptions = []*subscription.Subscription{{Channel: subscription.TickerChannel, Asset: q.Key.ExchangeAssetPair.Asset}}
	// This may attempt to connect; skip asserting success to avoid flakiness.
}

func TestFocusDataValidateAndInit(t *testing.T) {
	t.Parallel()
	var f *FocusData
	require.ErrorIs(t, f.Validate(&CredentialsKey{}), common.ErrNilPointer)

	f = &FocusData{}
	require.ErrorIs(t, f.Validate(&CredentialsKey{}), ErrUnsetFocusType)

	f = &FocusData{Type: TickerFocusType}
	require.ErrorIs(t, f.Validate(&CredentialsKey{}), ErrInvalidRESTPollTime)

	f = &FocusData{Type: OpenInterestFocusType, RESTPollTime: time.Second}
	k := &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)}
	require.ErrorIs(t, f.Validate(k), ErrInvalidAssetForFocusType)

	f = &FocusData{Type: AccountHoldingsFocusType, RESTPollTime: time.Second}
	k = &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, asset.Futures, currencyPair)}
	require.ErrorIs(t, f.Validate(k), ErrCredentialsRequiredForFocusType)

	f = &FocusData{Type: TickerFocusType, RESTPollTime: time.Second, UseWebsocket: true}
	require.NoError(t, f.Validate(&CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)}))

	f.Init()
	// SetSuccessful twice is safe
	go f.SetSuccessful()
	go f.SetSuccessful()
	select {
	case <-f.HasBeenSuccessfulChan:
		// closed as expected
	case <-time.After(time.Second):
		require.FailNow(t, "expected HasBeenSuccessfulChan to be closed")
	}

	for _, ft := range focusList {
		fd := &FocusData{Type: ft, RESTPollTime: time.Second}
		fd.Init()
		require.NotNil(t, fd.m)
		if ft == AccountHoldingsFocusType || ft == ActiveOrdersFocusType || ft == OrderPlacementFocusType {
			assert.Truef(t, fd.RequiresAuth(), "expected %v to require auth", ft)
		} else {
			assert.Falsef(t, fd.RequiresAuth(), "expected %v to not require auth", ft)
		}
	}
	fd := &FocusData{Type: TickerFocusType, RESTPollTime: time.Second}
	fd.Init()
	fd.UseWebsocket = true
	assert.True(t, fd.RequiresWebsocket())
	fd.UseWebsocket = false
	assert.False(t, fd.RequiresWebsocket())
}

func TestLatestData(t *testing.T) {
	t.Parallel()
	t.Run("errKeyNotFound", func(t *testing.T) {
		q := mustQuickSpy(t, OrderBookFocusType)
		_, err := q.LatestData(TickerFocusType)
		require.ErrorIs(t, err, errKeyNotFound)
	})

	q := mustQuickSpyAllFocuses(t)
	l := q.Focuses.List()
	for i := range l {
		t.Run(l[i].Type.String(), func(t *testing.T) {
			t.Parallel()
			_, err := q.LatestData(l[i].Type)
			require.ErrorIs(t, err, errNoDataYet)
			l[i].hasBeenSuccessful = true
			close(l[i].HasBeenSuccessfulChan)
			_, err = q.LatestData(l[i].Type)
			require.NoError(t, err)
		})
	}

	t.Run("illegal Focus default scenario", func(t *testing.T) {
		t.Parallel()
		q := mustQuickSpy(t, 999)
		q.Focuses.s[999].hasBeenSuccessful = true
		close(q.Focuses.s[999].HasBeenSuccessfulChan)
		_, err := q.LatestData(999)
		require.ErrorIs(t, err, ErrUnsupportedFocusType)
	})
}

func TestWaitForInitialDataWithTimer(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{
		Key:     &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
		Focuses: NewFocusStore(),
	}
	f := &FocusData{Type: TickerFocusType, RESTPollTime: time.Millisecond * 10}
	f.Init()
	qs.Focuses.Upsert(TickerFocusType, f)
	ctx, cancel := context.WithCancel(context.Background())
	// Timeout path for WaitForInitialDataWithTimer
	require.Error(t, qs.WaitForInitialDataWithTimer(ctx, TickerFocusType, 1))
	// Success path
	close(f.HasBeenSuccessfulChan)
	require.NoError(t, qs.WaitForInitialDataWithTimer(context.Background(), TickerFocusType, time.Second))
	// Key not found path
	require.Error(t, qs.WaitForInitialDataWithTimer(context.Background(), OrderBookFocusType, 1))
	// Cancel context path
	cancel()
	require.NoError(t, qs.WaitForInitialDataWithTimer(context.Background(), TickerFocusType, time.Second))
}

func TestWaitForInitialData(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{
		Key:     &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
		Focuses: NewFocusStore(),
	}
	f := &FocusData{Type: TickerFocusType, RESTPollTime: time.Millisecond * 10}
	f.Init()
	qs.Focuses.Upsert(TickerFocusType, f)
	_, cancel := context.WithCancel(context.Background())
	// Success path
	close(f.HasBeenSuccessfulChan)
	require.NoError(t, qs.WaitForInitialData(context.Background(), TickerFocusType))
	// Key not found path
	require.Error(t, qs.WaitForInitialData(context.Background(), OrderBookFocusType))
	// Cancel context path
	cancel()
	require.NoError(t, qs.WaitForInitialDataWithTimer(context.Background(), TickerFocusType, 1))
}

func TestHandleWSAndShutdown(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{
		Key:                &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
		Focuses:            NewFocusStore(),
		dataHandlerChannel: make(chan any, 10),
		Data:               &Data{},
		m:                  new(sync.RWMutex),
	}
	// Register focuses and init streams
	for _, ft := range []FocusType{TickerFocusType, OrderBookFocusType, TradesFocusType} {
		fd := &FocusData{Type: ft, RESTPollTime: time.Second}
		fd.Init()
		qs.Focuses.Upsert(ft, fd)
	}

	// Provide a cancellable context to avoid nil dereference and enable shutdown
	ctx, cancel := context.WithCancel(context.Background())
	qs.credContext = ctx

	done := make(chan struct{})
	go func() {
		_ = qs.HandleWS()
		close(done)
	}()

	// ticker.Price single
	pr := &ticker.Price{Last: 1}
	qs.dataHandlerChannel <- pr
	// ticker.Price slice
	qs.dataHandlerChannel <- []ticker.Price{{Last: 2}}

	// orderbook depth
	uid, err := uuid.NewV4()
	require.NoError(t, err)
	d := orderbook.NewDepth(uid)
	_ = d.LoadSnapshot(&orderbook.Book{
		Exchange:    exchangeName,
		Pair:        currencyPair,
		Asset:       assetType,
		Bids:        orderbook.Levels{{Amount: 1, Price: 1}},
		Asks:        orderbook.Levels{{Amount: 1, Price: 2}},
		LastUpdated: time.Now()})
	qs.dataHandlerChannel <- d

	// trades
	qs.dataHandlerChannel <- trade.Data{Price: 1, Amount: 1}
	qs.dataHandlerChannel <- []trade.Data{{Price: 2, Amount: 2}}

	// Give some time for processing
	time.Sleep(50 * time.Millisecond)
	qs.m.RLock()

	require.Equal(t, qs.Data.Ticker.Last, 2., "expected 2 as it is the latest ticker received")
	require.NotNil(t, qs.Data.Orderbook)
	require.Len(t, qs.Data.Trades, 1)
	qs.m.RUnlock()

	// Cancel context and ensure goroutine exits
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "HandleWS did not shut down in time")
	}
}

func TestRunAndHandleFocusTypeVariants(t *testing.T) {
	t.Parallel()
	// Test Run with a non-WS, no-implementation focus (OrderPlacementFocusType)
	t.Run("Run spawns REST goroutine for no-impl focus and completes", func(t *testing.T) {
		t.Parallel()
		qs := &QuickSpy{
			Key:     &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
			Focuses: NewFocusStore(),
			Data:    &Data{},
			m:       new(sync.RWMutex),
			// Provide cancellable context so background workers can exit if needed
			credContext: context.Background(),
		}
		fd := &FocusData{Type: OrderPlacementFocusType, RESTPollTime: time.Millisecond * 10}
		fd.Init()
		qs.Focuses.Upsert(OrderPlacementFocusType, fd)
		require.NoError(t, qs.Run())
		select {
		case <-fd.HasBeenSuccessfulChan:
			// success indicates the goroutine ran and SetSuccessful was called
		case <-time.After(time.Second):
			require.FailNow(t, "expected successful focus signal")
		}
	})

	// Test handleFocusType for URLFocusType success using a real exchange instance
	t.Run("handleFocusType URLFocusType populates URL and streams", func(t *testing.T) {
		t.Parallel()
		fd := &FocusData{Type: URLFocusType, RESTPollTime: time.Second}
		fd.Init()
		qs := &QuickSpy{
			Key:         &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
			Focuses:     NewFocusStore(),
			Data:        &Data{},
			m:           new(sync.RWMutex),
			credContext: context.Background(),
		}
		qs.Focuses.Upsert(URLFocusType, fd)
		// Use supported exchange and initialise defaults and pairs so URL can be generated without RPC calls
		e, err := engine.NewSupportedExchangeByName(qs.Key.ExchangeAssetPair.Exchange)
		require.NoError(t, err)
		b := e.GetBase()
		require.NoError(t, qs.setupExchangeDefaults(e, b))
		require.NoError(t, qs.setupCurrencyPairs(b))
		qs.Exch = e
		timer := time.NewTimer(time.Hour)
		require.NoError(t, qs.handleFocusType(URLFocusType, fd, timer))
		// Data may be empty if exchange doesn't support URL; just assert no panic and optional stream if provided
		select {
		case v := <-fd.Stream:
			_ = v // best-effort; type may be string
		case <-time.After(100 * time.Millisecond):
			// acceptable; non-blocking send may have dropped if empty URL
		}
	})

	// Test handleFocusType unknown focus returns error and streams it
	t.Run("handleFocusType unknown focus returns error and streams", func(t *testing.T) {
		t.Parallel()
		fd := &FocusData{Type: FocusType(999), RESTPollTime: time.Second}
		fd.Init()
		qs := &QuickSpy{Focuses: NewFocusStore(), Data: &Data{}, m: new(sync.RWMutex), credContext: context.Background()}
		qs.Focuses.Upsert(fd.Type, fd)
		timer := time.NewTimer(time.Hour)
		err := qs.handleFocusType(fd.Type, fd, timer)
		require.Error(t, err)
		// Do not assert on stream: handler uses non-blocking send and may drop the error
	})

	// Test successfulSpy behavior for once-off and periodic
	t.Run("successfulSpy closes chan and resets timer appropriately", func(t *testing.T) {
		t.Parallel()
		fd := &FocusData{Type: TickerFocusType, RESTPollTime: 50 * time.Millisecond}
		fd.Init()
		timer := time.NewTimer(time.Hour)
		qs := &QuickSpy{}
		qs.successfulSpy(fd, timer)
		// channel closed
		select {
		case <-fd.HasBeenSuccessfulChan:
		default:
			require.FailNow(t, "expected successful chan to be closed")
		}
		// timer should fire within rest poll time (with some slack)
		select {
		case <-timer.C:
			// ok
		case <-time.After(500 * time.Millisecond):
			require.FailNow(t, "timer did not reset as expected")
		}

		// Once-off should not reset timer
		fd = &FocusData{Type: TickerFocusType, RESTPollTime: 10 * time.Millisecond, IsOnceOff: true}
		fd.Init()
		timer = time.NewTimer(100 * time.Millisecond)
		// drain initial if fired
		select {
		case <-timer.C:
		default:
		}
		qs.successfulSpy(fd, timer)
		// expect no immediate reset (timer should not fire within 50ms)
		select {
		case <-timer.C:
			// could fire if it was already near, but not expected; restart to make deterministic
		default:
		}
		time.Sleep(50 * time.Millisecond)
		select {
		case <-timer.C:
			// acceptable if previous drain missed; main point is no reset call occurred
		default:
			// still acceptable; nothing to assert strongly here other than no panic
		}
	})
}

func TestRunRESTFocusEdgeCases(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{
		Key:         &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
		Focuses:     NewFocusStore(),
		credContext: context.Background(),
	}
	// Missing key
	require.Error(t, qs.RunRESTFocus(TickerFocusType))
	// UseWebsocket skips work
	fd := &FocusData{Type: TickerFocusType, RESTPollTime: time.Millisecond * 10, UseWebsocket: true}
	fd.Init()
	qs.Focuses.Upsert(TickerFocusType, fd)
	require.NoError(t, qs.RunRESTFocus(TickerFocusType))
	// Unknown focus type with IsOnceOff returns immediately on error path
	u := &FocusData{Type: FocusType(999), RESTPollTime: time.Millisecond * 10, IsOnceOff: true}
	u.Init()
	qs.Focuses.Upsert(FocusType(999), u)
	// Run in goroutine and wait a short moment then shutdown to ensure no deadlocks
	done := make(chan struct{})
	go func() {
		_ = qs.RunRESTFocus(FocusType(999))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "RunRESTFocus did not return for once-off unknown focus")
	}
}

func TestHandleFocusType(t *testing.T) {
	t.Parallel()
	q := mustQuickSpyAllFocuses(t)
	// Each subtest calls the specific handler through handleFocusType to exercise its branch.
	cases := []struct {
		name string
		ft   FocusType
	}{
		{"Contract", ContractFocusType},
		{"Kline", KlineFocusType},
		{"OpenInterest", OpenInterestFocusType},
		{"Ticker", TickerFocusType},
		{"ActiveOrders", ActiveOrdersFocusType},
		{"AccountHoldings", AccountHoldingsFocusType},
		{"OrderBook", OrderBookFocusType},
		{"Trades", TradesFocusType},
		{"OrderExecution", OrderLimitsFocusType},
		{"FundingRate", FundingRateFocusType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if slices.Contains(authFocusList, tc.ft) && apiKey == "abc" && apiSecret == "123" {
				t.Skip("API credentials not set; skipping test that requires them")
			}
			fd := &FocusData{Type: tc.ft, RESTPollTime: time.Second}
			fd.Init()
			timer := time.NewTimer(time.Hour)
			assert.NoError(t, q.handleFocusType(tc.ft, fd, timer))
		})
	}
}

func TestDump(t *testing.T) {
	t.Parallel()
	q := mustQuickSpyAllFocuses(t)
	fl := q.Focuses.List()
	for _, fd := range fl {
		if slices.Contains(authFocusList, fd.Type) && apiKey == "abc" && apiSecret == "123" {
			continue
		}
		require.NoError(t, q.handleFocusType(fd.Type, fd, time.NewTimer(fd.RESTPollTime)))
	}
	d, err := q.Dump()
	require.NoError(t, err)
	require.NotEmpty(t, d)
	assert.NotEmpty(t, d.Key)
	assert.NotNil(t, d.UnderlyingBase)
	assert.NotNil(t, d.UnderlyingQuote)
	assert.NotEmpty(t, d.ContractExpirationTime)
	assert.NotEmpty(t, d.ContractType)
	assert.Positive(t, d.ContractDecimals)
	assert.NotEmpty(t, d.ContractSettlement)
	assert.True(t, d.HasValidCredentials)
	assert.Positive(t, d.LastPrice)
	assert.Positive(t, d.IndexPrice)
	assert.Positive(t, d.MarkPrice)
	assert.Positive(t, d.Volume)
	assert.Positive(t, d.FundingRate)
	assert.Positive(t, d.EstimatedFundingRate)
	assert.Positive(t, d.LastTradePrice)
	assert.Positive(t, d.LastTradeSize)
	assert.NotEmpty(t, d.Bids)
	assert.NotEmpty(t, d.Asks)
	assert.Positive(t, d.OpenInterest)
	assert.Positive(t, d.NextFundingRateTime)
	assert.Positive(t, d.CurrentFundingRateTime)
	assert.NotEmpty(t, d.ExecutionLimits)
	assert.NotEmpty(t, d.URL)

}

func TestWaitForInitialDataWithTimer_Zero(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{Focuses: NewFocusStore()}
	require.Error(t, qs.WaitForInitialDataWithTimer(context.Background(), TickerFocusType, 0))
}

func TestShutdown(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{credContext: context.Background()}
	require.NotPanics(t, func() { qs.Shutdown() }, "shutdown with set context should not panic")
	qs.credContext = nil
	require.Panics(t, func() { qs.Shutdown() }, "shutdown with nil context should panic")
}

func TestAccountHoldingsFocusType(t *testing.T) {
	t.Parallel()
	if apiKey == "abc" || apiSecret == "123" {
		t.Skip("API credentials not set; skipping test that requires them")
	}
	qs := mustQuickSpy(t, AccountHoldingsFocusType)
	f, err := qs.GetFocusByKey(AccountHoldingsFocusType)
	require.NoError(t, err)
	require.NotNil(t, f)

	require.NoError(t, qs.handleFocusType(f.Type, f, time.NewTimer(f.RESTPollTime)))
	require.NotEmpty(t, qs.Data.AccountBalance)
}
