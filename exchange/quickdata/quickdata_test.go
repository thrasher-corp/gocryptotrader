package quickdata

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// these are here to help a user test
// modifying them and decrying that tests fail will get you thrown in gaol
var (
	exchangeName     = "gateio"
	assetType        = asset.Spot
	pair             = currency.NewBTCUSDT()
	apiKey           = "abc"
	apiSecret        = "123"
	futuresAssetType = asset.USDTMarginedFutures
)

func mustQuickData(t *testing.T, ft FocusType) *QuickData {
	t.Helper()
	ftd := NewFocusData(ft, false, false, time.Second)
	ftd.Init()
	k := key.NewExchangeAssetPair(exchangeName, assetType, pair)
	qs, err := NewQuickData(&k, []*FocusData{ftd})
	require.NoError(t, err)
	require.NotNil(t, qs)
	return qs
}

func mustQuickDataAllFocuses(t *testing.T) *QuickData {
	t.Helper()
	focuses := make([]*FocusData, 0, len(allFocusList))
	for _, ft := range allFocusList {
		ftd := NewFocusData(ft, false, false, time.Second)
		focuses = append(focuses, ftd)
	}
	k := key.NewExchangeAssetPair(exchangeName, futuresAssetType, pair)
	qs, err := NewQuickData(&k, focuses)
	require.NoError(t, err)
	require.NotNil(t, qs)
	return qs
}

func TestNewQuickData(t *testing.T) {
	t.Parallel()
	_, err := NewQuickData(nil, nil)
	require.ErrorIs(t, err, errNoKey)

	_, err = NewQuickData(&key.ExchangeAssetPair{}, nil)
	require.ErrorIs(t, err, errNoFocus)

	_, err = NewQuickData(&key.ExchangeAssetPair{}, []*FocusData{{}})
	require.ErrorIs(t, err, ErrUnsupportedFocusType)

	_, err = NewQuickData(&key.ExchangeAssetPair{}, []*FocusData{{focusType: OrderBookFocusType, restPollTime: -1}})
	require.ErrorIs(t, err, ErrInvalidRESTPollTime)

	k := key.NewExchangeAssetPair(exchangeName, asset.Binary, pair)
	_, err = NewQuickData(&k, []*FocusData{{focusType: OpenInterestFocusType, restPollTime: 10}})
	require.ErrorIs(t, err, ErrInvalidAssetForFocusType)

	k = key.NewExchangeAssetPair(exchangeName, assetType, pair)
	qs, err := NewQuickData(&k, []*FocusData{{focusType: AccountHoldingsFocusType, restPollTime: 10}})
	require.NoError(t, err)
	require.NotNil(t, qs)
}

func TestAnyRequiresWebsocket(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)
	q.focuses.Upsert(TickerFocusType, &FocusData{focusType: TickerFocusType, restPollTime: time.Second, useWebsocket: true})
	require.True(t, q.AnyRequiresWebsocket())

	q.focuses.Upsert(TickerFocusType, &FocusData{focusType: TickerFocusType, restPollTime: time.Second, useWebsocket: false})
	require.False(t, q.AnyRequiresWebsocket())
}

func TestAnyRequiresAuth(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)
	require.False(t, q.AnyRequiresAuth())

	q.focuses.Upsert(AccountHoldingsFocusType, &FocusData{focusType: AccountHoldingsFocusType, restPollTime: time.Second})
	require.True(t, q.AnyRequiresAuth())
}

func TestGetFocusByKey(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)

	_, err := q.GetFocusByKey(UnsetFocusType)
	require.ErrorIs(t, err, errKeyNotFound)

	f, err := q.GetFocusByKey(TickerFocusType)
	require.NoError(t, err)
	require.NotNil(t, f)
}

func TestSetupExchange(t *testing.T) {
	t.Parallel()
	k := key.NewExchangeAssetPair(exchangeName, assetType, pair)
	q := &QuickData{
		key:                &k,
		dataHandlerChannel: make(chan any),
		wg:                 sync.WaitGroup{},
		focuses:            NewFocusStore(),
	}
	err := q.setupExchange()
	require.NoError(t, err)

	q = &QuickData{
		key:     &k,
		focuses: NewFocusStore(),
	}
	k.Exchange = "butts"
	err = q.setupExchange()
	require.ErrorIs(t, err, engine.ErrExchangeNotFound)
}

func TestSetupExchangeDefaults(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.key.Exchange)
	require.NoError(t, err)
	b := e.GetBase()

	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)
}

func TestSetupCurrencyPairs(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.key.Exchange)
	require.NoError(t, err)
	b := e.GetBase()
	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)

	err = q.setupCurrencyPairs(b)
	require.NoError(t, err)
	require.NotNil(t, b.CurrencyPairs.Pairs[assetType])
	require.Nil(t, b.CurrencyPairs.Pairs[asset.Binary])

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = b.CurrencyPairs.Pairs[assetType].RequestFormat
	b.CurrencyPairs.ConfigFormat = b.CurrencyPairs.Pairs[assetType].ConfigFormat
	err = q.setupCurrencyPairs(b)
	require.NoError(t, err)
	require.NotNil(t, b.CurrencyPairs.Pairs[assetType])
}

func TestCheckRateLimits(t *testing.T) {
	t.Parallel()
	q := mustQuickData(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.key.Exchange)
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

func TestFocusDataValidateAndInit(t *testing.T) {
	t.Parallel()
	var f *FocusData
	require.ErrorIs(t, f.Validate(nil), common.ErrNilPointer)

	f = &FocusData{}
	require.ErrorIs(t, f.Validate(&key.ExchangeAssetPair{}), ErrUnsetFocusType)

	f = &FocusData{focusType: TickerFocusType}
	require.ErrorIs(t, f.Validate(&key.ExchangeAssetPair{}), ErrInvalidRESTPollTime)

	f = &FocusData{focusType: OpenInterestFocusType, restPollTime: time.Second}
	k := key.NewExchangeAssetPair(exchangeName, assetType, pair)
	require.ErrorIs(t, f.Validate(&k), ErrInvalidAssetForFocusType)

	k = key.NewExchangeAssetPair(exchangeName, assetType, pair)
	f = &FocusData{focusType: TickerFocusType, restPollTime: time.Second, useWebsocket: true}
	require.NoError(t, f.Validate(&k))

	f.Init()
	go f.setSuccessful()
	go f.setSuccessful()
	select {
	case <-f.hasBeenSuccessfulChan:
		// closed as expected
	case <-time.After(time.Second):
		require.FailNow(t, "hasBeenSuccessfulChan must be closed")
	}

	for _, ft := range allFocusList {
		fd := &FocusData{focusType: ft, restPollTime: time.Second}
		fd.Init()
		if slices.Contains(authFocusList, ft) {
			assert.Truef(t, RequiresAuth(fd.focusType), "expected %v to require auth", ft)
		} else {
			assert.Falsef(t, RequiresAuth(fd.focusType), "expected %v to not require auth", ft)
		}
	}
	fd := &FocusData{focusType: TickerFocusType, restPollTime: time.Second}
	fd.Init()
	fd.useWebsocket = true
	assert.True(t, fd.UseWebsocket())
	fd.useWebsocket = false
	assert.False(t, fd.UseWebsocket())
}

func TestLatestData(t *testing.T) {
	t.Parallel()
	t.Run("errKeyNotFound", func(t *testing.T) {
		q := mustQuickData(t, OrderBookFocusType)
		_, err := q.LatestData(TickerFocusType)
		require.ErrorIs(t, err, errKeyNotFound)
	})

	q := mustQuickDataAllFocuses(t)
	l := q.focuses.List()
	for i := range l {
		t.Run(l[i].focusType.String(), func(t *testing.T) {
			t.Parallel()
			_, err := q.LatestData(l[i].focusType)
			if err != nil {
				// quickData automatically runs, so if the test is too fast we may not have data yet
				require.ErrorIs(t, err, errNoDataYet)
			}
			l[i].setSuccessful()
			_, err = q.LatestData(l[i].focusType)
			require.NoError(t, err)
		})
	}

	t.Run("illegal Focus default scenario", func(t *testing.T) {
		t.Parallel()
		q := mustQuickData(t, TickerFocusType)
		_, err := q.LatestData(111)
		require.ErrorIs(t, err, errKeyNotFound)
	})
}

func TestWaitForInitialDataWithTimeout(t *testing.T) {
	t.Parallel()
	k := key.NewExchangeAssetPair(exchangeName, assetType, pair)
	qs := &QuickData{
		key:     &k,
		focuses: NewFocusStore(),
	}
	f := &FocusData{focusType: TickerFocusType, restPollTime: time.Millisecond * 10}
	f.Init()
	qs.focuses.Upsert(TickerFocusType, f)
	ctx, cancel := context.WithCancel(t.Context())
	require.Error(t, qs.WaitForInitialDataWithTimeout(ctx, TickerFocusType, 1))
	cancel()
	require.ErrorIs(t, qs.WaitForInitialDataWithTimeout(ctx, TickerFocusType, time.Second), context.Canceled)

	f.setSuccessful()
	require.NoError(t, qs.WaitForInitialDataWithTimeout(t.Context(), TickerFocusType, time.Second))
	require.Error(t, qs.WaitForInitialDataWithTimeout(t.Context(), OrderBookFocusType, 1))

	require.ErrorIs(t, qs.WaitForInitialDataWithTimeout(t.Context(), TickerFocusType, 0), errTimerNotSet)
}

func TestWaitForInitialData(t *testing.T) {
	t.Parallel()
	k := key.NewExchangeAssetPair(exchangeName, assetType, pair)
	qs := &QuickData{
		key:     &k,
		focuses: NewFocusStore(),
	}
	f := &FocusData{focusType: TickerFocusType, restPollTime: time.Millisecond * 10}
	f.Init()
	qs.focuses.Upsert(TickerFocusType, f)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, qs.WaitForInitialData(ctx, TickerFocusType), context.Canceled)

	f.setSuccessful()
	require.NoError(t, qs.WaitForInitialData(t.Context(), TickerFocusType))
	require.Error(t, qs.WaitForInitialData(t.Context(), OrderBookFocusType))
}

func TestHandleFocusType(t *testing.T) {
	t.Parallel()
	q := mustQuickDataAllFocuses(t)
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{
		Key:    apiKey,
		Secret: apiSecret,
	})
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
		{"ActiveOrders", ActiveOrdersFocusType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if slices.Contains(authFocusList, tc.ft) && apiKey == "abc" && apiSecret == "123" {
				t.Skip("API credentials not set; skipping test that requires them")
			}
			fd := &FocusData{focusType: tc.ft, restPollTime: time.Second}
			fd.Init()
			assert.NoError(t, q.handleFocusType(ctx, tc.ft, fd))
			assert.NotEmpty(t, <-fd.Stream)
		})
	}
}

func TestDump(t *testing.T) {
	t.Parallel()
	q := mustQuickDataAllFocuses(t)
	fl := q.focuses.List()
	ctx := account.DeployCredentialsToContext(t.Context(), &account.Credentials{
		Key:    apiKey,
		Secret: apiSecret,
	})
	for _, fd := range fl {
		if slices.Contains(authFocusList, fd.focusType) && apiKey == "abc" && apiSecret == "123" {
			continue
		}
		require.NoError(t, q.handleFocusType(ctx, fd.focusType, fd))
	}
	d, err := q.DumpJSON()
	require.NoError(t, err)
	require.NotEmpty(t, d)
}

func TestGetAndWaitForFocusByKey(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		qs := mustQuickData(t, TickerFocusType)
		err := qs.Run(t.Context())
		require.NoError(t, err, "Run must not error")
		f, err := qs.GetAndWaitForFocusByKey(t.Context(), TickerFocusType, time.Second*5)
		require.NoError(t, err)
		require.NotNil(t, f)
	})
	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		qs := mustQuickData(t, TickerFocusType)
		f, err := qs.GetAndWaitForFocusByKey(t.Context(), TickerFocusType, 0)
		require.ErrorIs(t, err, errFocusDataTimeout)
		require.Nil(t, f)
	})
	t.Run("context cancelled", func(t *testing.T) {
		t.Parallel()
		qs := mustQuickData(t, TickerFocusType)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		f, err := qs.GetAndWaitForFocusByKey(ctx, TickerFocusType, time.Hour)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, f)
	})
}

func TestNewQuickerData(t *testing.T) {
	t.Parallel()
	_, err := NewQuickerData(nil, 111)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = NewQuickerData(&key.ExchangeAssetPair{}, 111)
	require.ErrorIs(t, err, ErrUnsupportedFocusType)

	_, err = NewQuickerData(&key.ExchangeAssetPair{}, TickerFocusType)
	require.ErrorIs(t, err, engine.ErrExchangeNotFound)

	k := &key.ExchangeAssetPair{
		Exchange: exchangeName,
		Asset:    futuresAssetType,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
	}
	qs, err := NewQuickerData(k, TickerFocusType)
	require.NoError(t, err)
	require.NotNil(t, qs)
	require.NoError(t, qs.Run(t.Context()), "Run must not error")
	ts := func() bool {
		hasBeen, _ := qs.HasBeenSuccessful(TickerFocusType)
		return hasBeen
	}
	assert.Eventually(t, ts, time.Second*5, time.Millisecond*100, "expected Ticker focus to have been successful within 5 seconds")
}

func TestNewQuickestData(t *testing.T) {
	t.Parallel()
	_, err := NewQuickestData(t.Context(), nil, 111)
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = NewQuickestData(t.Context(), &key.ExchangeAssetPair{}, 111)
	require.ErrorIs(t, err, ErrUnsupportedFocusType)

	_, err = NewQuickestData(t.Context(), &key.ExchangeAssetPair{}, TickerFocusType)
	require.ErrorIs(t, err, engine.ErrExchangeNotFound)

	k := &key.ExchangeAssetPair{
		Exchange: exchangeName,
		Asset:    futuresAssetType,
		Base:     pair.Base.Item,
		Quote:    pair.Quote.Item,
	}
	c, err := NewQuickestData(t.Context(), k, TickerFocusType)
	require.NoError(t, err)
	ts := func() bool {
		<-c
		return true
	}
	assert.Eventually(t, ts, time.Second*5, time.Millisecond*100, "expected Ticker focus to have been successful within 5 seconds")
}

func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	qs := mustQuickData(t, TickerFocusType)

	assert.ErrorIs(t, qs.validateSubscriptions(nil), errNoSubSwitchingToREST)

	assert.ErrorIs(t, qs.validateSubscriptions([]*subscription.Subscription{{}, {}}), errNoSubSwitchingToREST)

	assert.ErrorIs(t, qs.validateSubscriptions([]*subscription.Subscription{{
		Enabled: true,
		Channel: subscription.TickerChannel,
		Pairs:   []currency.Pair{currency.NewBTCUSD()},
		Asset:   asset.Binary,
	}}), errNoSubSwitchingToREST)

	assert.ErrorIs(t, qs.validateSubscriptions([]*subscription.Subscription{{
		Enabled: true,
		Channel: subscription.TickerChannel,
		Pairs:   []currency.Pair{currency.NewBTCUSD()},
		Asset:   futuresAssetType,
	}}), errNoSubSwitchingToREST)

	ftd := NewFocusData(TickerFocusType, false, true, time.Second)
	ftd.Init()

	k := key.NewExchangeAssetPair(exchangeName, assetType, pair)
	qs, err := NewQuickData(&k, []*FocusData{ftd})
	require.NoError(t, err)
	require.NotNil(t, qs)
	assert.NoError(t, qs.validateSubscriptions([]*subscription.Subscription{{
		Enabled: true,
		Channel: subscription.TickerChannel,
		Pairs:   []currency.Pair{pair},
		Asset:   futuresAssetType,
	}}))
}

func TestData(t *testing.T) {
	t.Parallel()
	qs := mustQuickDataAllFocuses(t)
	assert.NotNil(t, qs.Data())
	assert.Equal(t, qs.Data(), qs.data)
}

func TestProcessRESTFocus(t *testing.T) {
	t.Parallel()
	qs := mustQuickData(t, TickerFocusType)
	f := qs.focuses.GetByFocusType(TickerFocusType)
	require.NoError(t, qs.processRESTFocus(t.Context(), f))

	fd := NewFocusData(111, false, false, time.Second)
	fd.Init()
	fd.FailureTolerance = 2
	require.NoError(t, qs.processRESTFocus(t.Context(), fd))
	require.ErrorIs(t, qs.processRESTFocus(t.Context(), fd), errOverMaxFailures)

	fd = NewFocusData(111, false, false, time.Second)
	fd.Init()
	fd.setSuccessful()
	require.NoError(t, qs.processRESTFocus(t.Context(), fd))
	sErr := <-fd.Stream
	err, ok := sErr.(error)
	require.True(t, ok)
	require.ErrorIs(t, err, ErrUnsupportedFocusType)
}

func TestHandleWSData(t *testing.T) {
	t.Parallel()
	qs := mustQuickDataAllFocuses(t)
	assert.ErrorIs(t, qs.handleWSData("butts"), errUnhandledWebsocketData)
	assert.NoError(t, qs.handleWSData(&ticker.Price{}))
	assert.NoError(t, qs.handleWSData([]ticker.Price{}))
	assert.NoError(t, qs.handleWSData(&orderbook.Depth{}))
	assert.NoError(t, qs.handleWSData(account.Change{}))
	assert.NoError(t, qs.handleWSData([]account.Change{}))
	assert.NoError(t, qs.handleWSData(&order.Detail{}))
	assert.NoError(t, qs.handleWSData([]order.Detail{}))
	assert.NoError(t, qs.handleWSData(trade.Data{}))
	assert.NoError(t, qs.handleWSData([]trade.Data{}))
}

func TestRun(t *testing.T) {
	t.Parallel()
	qs := mustQuickData(t, TickerFocusType)
	require.ErrorIs(t, qs.Run(context.Background()), ErrContextMustBeAbleToFinish)

	ctx, cancel := context.WithCancel(t.Context())
	require.NoError(t, qs.Run(ctx))
	go func() {
		cancel()
	}()
	qs.wg.Wait()
}

func TestHandleWS(t *testing.T) {
	t.Parallel()
	qs := mustQuickData(t, TickerFocusType)
	qs.dataHandlerChannel = make(chan any, 1)
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		qs.dataHandlerChannel <- "test"
		time.Sleep(time.Millisecond * 100)
		cancel()
	}()
	ts := func() bool {
		return errors.Is(qs.handleWS(ctx), context.Canceled)
	}
	assert.Eventually(t, ts, time.Second*2, time.Millisecond*100, "expected cancellation within 2 seconds")
}
