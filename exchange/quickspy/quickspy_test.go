package quickspy

import (
	"context"
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
	exchangeName     = "binance"
	assetType        = asset.Spot
	currencyPair     = currency.NewBTCUSDT()
	apiKey           = "abc"
	apiSecret        = "123"
	futuresAssetType = asset.USDTMarginedFutures // used in TestDumpAndCurrentPayload
)

func mustQuickSpy(t *testing.T, ft FocusType) *QuickSpy {
	t.Helper()
	ftd := NewFocusData(ft, false, true, time.Second)
	ftd.Init()
	qs, err := NewQuickSpy(
		t.Context(),
		&CredentialsKey{
			ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair),
			Credentials: &account.Credentials{
				Key:    apiKey,
				Secret: apiSecret,
			},
		},
		[]*FocusData{ftd})
	require.NoError(t, err)
	require.NotNil(t, qs)
	return qs
}

func mustQuickSpyAllFocuses(t *testing.T) *QuickSpy {
	t.Helper()
	focuses := make([]*FocusData, 0, len(allFocusList))
	for _, ft := range allFocusList {
		ftd := NewFocusData(ft, false, false, time.Second)
		focuses = append(focuses, ftd)
	}
	qs, err := NewQuickSpy(
		t.Context(),
		&CredentialsKey{
			ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, futuresAssetType, currencyPair),
			Credentials: &account.Credentials{
				Key:    apiKey,
				Secret: apiSecret,
			},
		},
		focuses)
	require.NoError(t, err)
	require.NotNil(t, qs)
	return qs
}

func TestNewQuickSpy(t *testing.T) {
	t.Parallel()
	_, err := NewQuickSpy(nil, nil, nil) //nolint:staticcheck // testing nil context
	require.ErrorIs(t, err, errNoKey)

	_, err = NewQuickSpy(t.Context(), &CredentialsKey{}, nil)
	require.ErrorIs(t, err, errNoFocus)

	_, err = NewQuickSpy(t.Context(), &CredentialsKey{}, []*FocusData{{}})
	require.ErrorIs(t, err, ErrUnsupportedFocusType)

	_, err = NewQuickSpy(t.Context(), &CredentialsKey{}, []*FocusData{{focusType: OrderBookFocusType, restPollTime: -1}})
	require.ErrorIs(t, err, ErrInvalidRESTPollTime)

	_, err = NewQuickSpy(t.Context(), &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, asset.Binary, currency.NewBTCUSD())}, []*FocusData{{focusType: OpenInterestFocusType, restPollTime: 10}})
	require.ErrorIs(t, err, ErrInvalidAssetForFocusType)

	_, err = NewQuickSpy(t.Context(), &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, asset.Futures, currencyPair)}, []*FocusData{{focusType: AccountHoldingsFocusType, restPollTime: 10}})
	require.ErrorIs(t, err, ErrCredentialsRequiredForFocusType)

	qs, err := NewQuickSpy(t.Context(), &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair), Credentials: &account.Credentials{
		Key:    apiKey,
		Secret: apiSecret,
	}}, []*FocusData{{focusType: AccountHoldingsFocusType, restPollTime: 10}})
	require.NoError(t, err)
	require.NotNil(t, qs)

	qs, err = NewQuickSpy(t.Context(), &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair), Credentials: &account.Credentials{
		Key:    apiKey,
		Secret: apiSecret,
	}}, []*FocusData{{focusType: AccountHoldingsFocusType, restPollTime: 10}})
	require.NoError(t, err)
	require.NotNil(t, qs)
	assert.NotEmpty(t, qs.credContext.Value(account.ContextCredentialsFlag), "credentials should be popultated in context")
}

func TestAnyRequiresWebsocket(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	require.True(t, q.AnyRequiresWebsocket())

	q.focuses.Upsert(TickerFocusType, &FocusData{focusType: TickerFocusType, restPollTime: time.Second, useWebsocket: false})
	require.False(t, q.AnyRequiresWebsocket())
}

func TestAnyRequiresAuth(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	require.False(t, q.AnyRequiresAuth())

	q.focuses.Upsert(AccountHoldingsFocusType, &FocusData{focusType: AccountHoldingsFocusType, restPollTime: time.Second})
	require.True(t, q.AnyRequiresAuth())
}

func TestGetFocusByKey(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)

	_, err := q.GetFocusByKey(UnsetFocusType)
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
		key:                &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair("butts", assetType, currencyPair)},
		dataHandlerChannel: make(chan any),
		m:                  new(sync.RWMutex),
		wg:                 sync.WaitGroup{},
	}
	err = q.setupExchange()
	require.ErrorIs(t, err, engine.ErrExchangeNotFound)
}

func TestSetupExchangeDefaults(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.key.ExchangeAssetPair.Exchange)
	require.NoError(t, err)
	b := e.GetBase()

	err = q.setupExchangeDefaults(e, b)
	require.NoError(t, err)
}

func TestSetupCurrencyPairs(t *testing.T) {
	t.Parallel()
	q := mustQuickSpy(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.key.ExchangeAssetPair.Exchange)
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
	q := mustQuickSpy(t, TickerFocusType)
	e, err := engine.NewSupportedExchangeByName(q.key.ExchangeAssetPair.Exchange)
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
	require.ErrorIs(t, f.Validate(&CredentialsKey{}), common.ErrNilPointer)

	f = &FocusData{}
	require.ErrorIs(t, f.Validate(&CredentialsKey{}), ErrUnsetFocusType)

	f = &FocusData{focusType: TickerFocusType}
	require.ErrorIs(t, f.Validate(&CredentialsKey{}), ErrInvalidRESTPollTime)

	f = &FocusData{focusType: OpenInterestFocusType, restPollTime: time.Second}
	k := &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)}
	require.ErrorIs(t, f.Validate(k), ErrInvalidAssetForFocusType)

	f = &FocusData{focusType: AccountHoldingsFocusType, restPollTime: time.Second}
	k = &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, asset.Futures, currencyPair)}
	require.ErrorIs(t, f.Validate(k), ErrCredentialsRequiredForFocusType)

	f = &FocusData{focusType: TickerFocusType, restPollTime: time.Second, useWebsocket: true}
	require.NoError(t, f.Validate(&CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)}))

	f.Init()
	go f.setSuccessful()
	go f.setSuccessful()
	select {
	case <-f.hasBeenSuccessfulChan:
		// closed as expected
	case <-time.After(time.Second):
		require.FailNow(t, "expected hasBeenSuccessfulChan to be closed")
	}

	for _, ft := range allFocusList {
		fd := &FocusData{focusType: ft, restPollTime: time.Second}
		fd.Init()
		require.NotNil(t, fd.m)
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
		q := mustQuickSpy(t, OrderBookFocusType)
		_, err := q.LatestData(TickerFocusType)
		require.ErrorIs(t, err, errKeyNotFound)
	})

	q := mustQuickSpyAllFocuses(t)
	l := q.focuses.List()
	for i := range l {
		t.Run(l[i].focusType.String(), func(t *testing.T) {
			t.Parallel()
			_, err := q.LatestData(l[i].focusType)
			if err != nil {
				// quickspy automatically runs, so if the test is too fast we may not have data yet
				require.ErrorIs(t, err, errNoDataYet)
			}
			l[i].setSuccessful()
			_, err = q.LatestData(l[i].focusType)
			require.NoError(t, err)
		})
	}

	t.Run("illegal Focus default scenario", func(t *testing.T) {
		t.Parallel()
		q := mustQuickSpy(t, TickerFocusType)
		_, err := q.LatestData(999)
		require.ErrorIs(t, err, errKeyNotFound)
	})
}

func TestWaitForInitialDataWithTimer(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{
		key:     &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
		focuses: NewFocusStore(),
	}
	f := &FocusData{focusType: TickerFocusType, restPollTime: time.Millisecond * 10}
	f.Init()
	qs.focuses.Upsert(TickerFocusType, f)
	ctx, cancel := context.WithCancel(t.Context())
	require.Error(t, qs.WaitForInitialDataWithTimer(ctx, TickerFocusType, 1))

	f.setSuccessful()
	require.NoError(t, qs.WaitForInitialDataWithTimer(t.Context(), TickerFocusType, time.Second))
	require.Error(t, qs.WaitForInitialDataWithTimer(t.Context(), OrderBookFocusType, 1))
	cancel()

	require.NoError(t, qs.WaitForInitialDataWithTimer(t.Context(), TickerFocusType, time.Second))
}

func TestWaitForInitialData(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{
		key:     &CredentialsKey{ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, currencyPair)},
		focuses: NewFocusStore(),
	}
	f := &FocusData{focusType: TickerFocusType, restPollTime: time.Millisecond * 10}
	f.Init()
	qs.focuses.Upsert(TickerFocusType, f)
	_, cancel := context.WithCancel(t.Context())

	f.setSuccessful()
	require.NoError(t, qs.WaitForInitialData(t.Context(), TickerFocusType))

	require.Error(t, qs.WaitForInitialData(t.Context(), OrderBookFocusType))

	cancel()
	require.NoError(t, qs.WaitForInitialData(t.Context(), TickerFocusType))
}

func TestHandleFocusType(t *testing.T) {
	t.Parallel()
	q := mustQuickSpyAllFocuses(t)
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
			assert.NoError(t, q.handleFocusType(tc.ft, fd))
			assert.NotEmpty(t, <-fd.Stream)
		})
	}
}

func TestDump(t *testing.T) {
	t.Parallel()
	q := mustQuickSpyAllFocuses(t)
	fl := q.focuses.List()
	for _, fd := range fl {
		if slices.Contains(authFocusList, fd.focusType) && apiKey == "abc" && apiSecret == "123" {
			continue
		}
		require.NoError(t, q.handleFocusType(fd.focusType, fd))
	}
	d, err := q.DumpJSON()
	require.NoError(t, err)
	require.NotEmpty(t, d)
}

func TestWaitForInitialDataWithTimer_Zero(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{focuses: NewFocusStore()}
	require.Error(t, qs.WaitForInitialDataWithTimer(t.Context(), TickerFocusType, 0))
}

func TestShutdown(t *testing.T) {
	t.Parallel()
	qs := &QuickSpy{credContext: t.Context()}
	require.NotPanics(t, func() { qs.Shutdown() }, "shutdown with set context should not panic")
	qs.credContext = nil
	require.Panics(t, func() { qs.Shutdown() }, "shutdown with nil context should panic")
}

func TestGetAndWaitForFocusByKey(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		qs := mustQuickSpy(t, TickerFocusType)
		f, err := qs.GetAndWaitForFocusByKey(t.Context(), TickerFocusType, time.Second)
		require.NoError(t, err)
		require.NotNil(t, f)
	})
	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		qs := mustQuickSpy(t, TickerFocusType)
		f, err := qs.GetAndWaitForFocusByKey(t.Context(), TickerFocusType, 0)
		require.ErrorIs(t, err, errFocusDataTimeout)
		require.Nil(t, f)
	})
	t.Run("context cancelled", func(t *testing.T) {
		t.Parallel()
		qs := mustQuickSpy(t, TickerFocusType)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		f, err := qs.GetAndWaitForFocusByKey(ctx, TickerFocusType, time.Hour)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, f)
	})
}

func TestNewQuickerSpy(t *testing.T) {
	t.Parallel()
	_, err := NewQuickerSpy(nil, nil, -1) //nolint:staticcheck // testing nil context
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = NewQuickerSpy(t.Context(), &key.ExchangeAssetPair{}, -1)
	require.ErrorIs(t, err, ErrUnsupportedFocusType)

	_, err = NewQuickerSpy(t.Context(), &key.ExchangeAssetPair{}, TickerFocusType)
	require.ErrorIs(t, err, engine.ErrExchangeNotFound)

	k := &key.ExchangeAssetPair{
		Exchange: exchangeName,
		Asset:    futuresAssetType,
		Base:     currencyPair.Base.Item,
		Quote:    currencyPair.Quote.Item,
	}
	qs, err := NewQuickerSpy(t.Context(), k, TickerFocusType)
	require.NoError(t, err)
	require.NotNil(t, qs)
	ts := func() bool {
		hasBeen, _ := qs.HasBeenSuccessful(TickerFocusType)
		return hasBeen
	}
	assert.Eventually(t, ts, time.Second*5, time.Millisecond*100, "expected Ticker focus to have been successful within 5 seconds")
}

func TestNewQuickestSpy(t *testing.T) {
	t.Parallel()
	_, err := NewQuickestSpy(nil, nil, -1) //nolint:staticcheck // testing nil context
	require.ErrorIs(t, err, common.ErrNilPointer)

	_, err = NewQuickestSpy(t.Context(), &key.ExchangeAssetPair{}, -1)
	require.ErrorIs(t, err, ErrUnsupportedFocusType)

	_, err = NewQuickestSpy(t.Context(), &key.ExchangeAssetPair{}, TickerFocusType)
	require.ErrorIs(t, err, engine.ErrExchangeNotFound)

	k := &key.ExchangeAssetPair{
		Exchange: exchangeName,
		Asset:    futuresAssetType,
		Base:     currencyPair.Base.Item,
		Quote:    currencyPair.Quote.Item,
	}
	c, err := NewQuickestSpy(t.Context(), k, TickerFocusType)
	require.NoError(t, err)
	ts := func() bool {
		<-c
		return true
	}
	assert.Eventually(t, ts, time.Second*5, time.Millisecond*100, "expected Ticker focus to have been successful within 5 seconds")
}

func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	qs := mustQuickSpy(t, TickerFocusType)

	assert.ErrorIs(t, qs.validateSubscriptions(nil), errNoSubSwitchingToREST)

	assert.ErrorIs(t, qs.validateSubscriptions([]*subscription.Subscription{{}, {}}), errNoSubSwitchingToREST)

	assert.ErrorIs(t, qs.validateSubscriptions([]*subscription.Subscription{{
		Enabled: true,
		Channel: subscription.TickerChannel,
		Pairs:   []currency.Pair{currency.NewPair(currency.BTC, currency.USD)},
		Asset:   asset.Binary,
	}}), errNoSubSwitchingToREST)

	assert.ErrorIs(t, qs.validateSubscriptions([]*subscription.Subscription{{
		Enabled: true,
		Channel: subscription.TickerChannel,
		Pairs:   []currency.Pair{currency.NewPair(currency.BTC, currency.USD)},
		Asset:   futuresAssetType,
	}}), errNoSubSwitchingToREST)

	qs = mustQuickSpy(t, TickerFocusType)
	assert.NoError(t, qs.validateSubscriptions([]*subscription.Subscription{{
		Enabled: true,
		Channel: subscription.TickerChannel,
		Pairs:   []currency.Pair{currencyPair},
		Asset:   futuresAssetType,
	}}))
}

func TestData(t *testing.T) {
	t.Parallel()
	qs := mustQuickSpyAllFocuses(t)
	assert.NotNil(t, qs.Data())
	assert.Equal(t, qs.Data(), qs.data)
}

func TestProcessRESTFocus(t *testing.T) {
	t.Parallel()
	qs := mustQuickSpy(t, TickerFocusType)
	f := qs.focuses.GetByFocusType(TickerFocusType)
	require.NoError(t, qs.processRESTFocus(f))

	fd := NewFocusData(999, false, false, time.Second)
	fd.Init()
	fd.FailureTolerance = 2
	require.NoError(t, qs.processRESTFocus(fd))
	require.ErrorIs(t, qs.processRESTFocus(fd), errOverMaxFailures)

	fd = NewFocusData(999, false, false, time.Second)
	fd.Init()
	fd.setSuccessful()
	require.NoError(t, qs.processRESTFocus(fd))
	sErr := <-fd.Stream
	err, ok := sErr.(error)
	require.True(t, ok)
	require.ErrorIs(t, err, ErrUnsupportedFocusType)
}

func TestHandleWSData(t *testing.T) {
	t.Parallel()
	qs := mustQuickSpyAllFocuses(t)
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
