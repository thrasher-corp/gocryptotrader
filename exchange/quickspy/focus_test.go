package quickspy

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestNewFocusDataAndInit(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, false, time.Second)
	require.NotNil(t, fd, "NewFocusData returned nil")
	require.Equal(t, TickerFocusType, fd.focusType)
	require.False(t, fd.useWebsocket)
	require.False(t, fd.isOnceOff)
	require.Equal(t, time.Second, fd.restPollTime)
	require.NotNil(t, fd.m, "mutex not initialised")
	require.NotNil(t, fd.hasBeenSuccessfulChan, "hasBeenSuccessfulChan not initialised")
	require.NotNil(t, fd.Stream, "Stream channel not initialised")
	select {
	case <-fd.hasBeenSuccessfulChan:
		require.FailNow(t, "hasBeenSuccessfulChan should not be closed initially")
	default:
	}

	// Re-init should recreate channels and reset success flag
	fd.setSuccessful()
	// channel must be closed now
	select {
	case <-fd.hasBeenSuccessfulChan:
		// ok
	default:
		require.FailNow(t, "hasBeenSuccessfulChan should be closed after setSuccessful")
	}
	oldChan := fd.hasBeenSuccessfulChan
	fd.Init()
	require.NotNil(t, fd.m)
	require.NotNil(t, fd.hasBeenSuccessfulChan)
	require.NotNil(t, fd.Stream)
	require.NotEqual(t, oldChan, fd.hasBeenSuccessfulChan, "Init should create a new hasBeenSuccessfulChan")
	select {
	case <-fd.hasBeenSuccessfulChan:
		require.FailNow(t, "hasBeenSuccessfulChan should not be closed after re-Init")
	default:
	}
}

func TestSetSuccessful(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, false, time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fd.setSuccessful()
		}()
	}
	wg.Wait()

	// channel should be closed and reads should not block
	select {
	case <-fd.hasBeenSuccessfulChan:
		// ok
	default:
		require.FailNow(t, "hasBeenSuccessfulChan should be closed and readable")
	}
	// multiple reads on a closed channel should still proceed immediately
	select {
	case <-fd.hasBeenSuccessfulChan:
		// ok
	default:
		require.FailNow(t, "hasBeenSuccessfulChan should remain closed and readable")
	}
}

func TestRequiresWebsocket(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, true, 0)
	require.True(t, fd.RequiresWebsocket())
	fd.useWebsocket = false
	require.False(t, fd.RequiresWebsocket())
}

func TestRequiresAuth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ft       FocusType
		expected bool
	}{
		{AccountHoldingsFocusType, true},
		{ActiveOrdersFocusType, true},
		{TickerFocusType, false},
		{OrderBookFocusType, false},
		{FundingRateFocusType, false},
		{TradesFocusType, false},
		{KlineFocusType, false},
		{ContractFocusType, false},
		{OpenInterestFocusType, false},
		{OrderLimitsFocusType, false},
		{URLFocusType, false},
	}
	for _, tc := range cases {
		t.Run(tc.ft.String(), func(t *testing.T) {
			t.Parallel()
			fd := NewFocusData(tc.ft, false, false, time.Second)
			require.Equalf(t, tc.expected, fd.RequiresAuth(), "RequiresAuth(%v) mismatch", tc.ft)
		})
	}
}

func TestFocusType_String(t *testing.T) {
	t.Parallel()
	cases := map[FocusType]string{
		UnsetFocusType:           "Unset/Unknown FocusType",
		OpenInterestFocusType:    "OpenInterestFocusType",
		TickerFocusType:          "TickerFocusType",
		OrderBookFocusType:       "OrderBookFocusType",
		FundingRateFocusType:     "FundingRateFocusType",
		TradesFocusType:          "TradesFocusType",
		AccountHoldingsFocusType: "AccountHoldingsFocusType",
		ActiveOrdersFocusType:    "ActiveOrdersFocusType",
		KlineFocusType:           "KlineFocusType",
		ContractFocusType:        "ContractFocusType",
		OrderLimitsFocusType:     "OrderLimitsFocusType",
		URLFocusType:             "URLFocusType",
		FocusType(999):           "Unset/Unknown FocusType",
	}
	for in, exp := range cases {
		t.Run(in.String(), func(t *testing.T) {
			t.Parallel()
			require.Equalf(t, exp, in.String(), "FocusType(%d).String() mismatch", in)
		})

	}
}

// helper to build a CredentialsKey with provided asset and creds
func makeCredKey(tb testing.TB, a asset.Item, creds *account.Credentials) *CredentialsKey {
	tb.Helper()
	k := key.NewExchangeAssetPair("Binance", a, currency.NewBTCUSD())
	return &CredentialsKey{Credentials: creds, ExchangeAssetPair: k}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	// Spot ticker via REST
	fd := &FocusData{focusType: TickerFocusType, useWebsocket: false, restPollTime: time.Second}
	k := makeCredKey(t, asset.Spot, nil)
	require.NoError(t, fd.Validate(k))
	// Futures-specific type allowed on futures asset with websocket
	fd = &FocusData{focusType: OpenInterestFocusType, useWebsocket: true}
	k = makeCredKey(t, asset.Futures, nil)
	require.NoError(t, fd.Validate(k))
	// Futures-specific type fails on spot asset
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), ErrInvalidAssetForFocusType)
	// Auth-required type passes when credentials are provided
	fd = &FocusData{focusType: AccountHoldingsFocusType, useWebsocket: false, restPollTime: time.Second}
	k = makeCredKey(t, asset.Spot, &account.Credentials{})
	require.ErrorIs(t, fd.Validate(k), ErrNoCredentials)
	// invalid REST poll time
	fd = &FocusData{focusType: TickerFocusType, useWebsocket: false, restPollTime: 0}
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), ErrInvalidRESTPollTime)
	fd = &FocusData{focusType: UnsetFocusType, useWebsocket: true}
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), ErrUnsetFocusType)
	// nil stuff
	fd = nil
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), common.ErrNilPointer)
}
