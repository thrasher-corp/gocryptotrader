package quickdata

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
)

func TestNewFocusDataAndInit(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, false, time.Second)
	require.NotNil(t, fd, "NewFocusData returned nil")
	require.Equal(t, TickerFocusType, fd.focusType)
	require.False(t, fd.useWebsocket)
	require.False(t, fd.isOnceOff)
	require.Equal(t, time.Second, fd.restPollTime)
	require.NotNil(t, fd.hasBeenSuccessfulChan, "hasBeenSuccessfulChan not initialised")
	require.NotNil(t, fd.Stream, "Stream channel not initialised")
	select {
	case <-fd.hasBeenSuccessfulChan:
		require.FailNow(t, "hasBeenSuccessfulChan must not be closed initially")
	default:
	}

	fd.setSuccessful()
	select {
	case <-fd.hasBeenSuccessfulChan:
		// ok
	default:
		require.FailNow(t, "hasBeenSuccessfulChan must be closed after setSuccessful")
	}
	oldChan := fd.hasBeenSuccessfulChan
	fd.Init()
	require.NotNil(t, fd.hasBeenSuccessfulChan)
	require.NotNil(t, fd.Stream)
	require.NotEqual(t, oldChan, fd.hasBeenSuccessfulChan, "Init must create a new hasBeenSuccessfulChan")
	select {
	case <-fd.hasBeenSuccessfulChan:
		require.FailNow(t, "hasBeenSuccessfulChan must not be closed after re-Init")
	default:
	}
}

func TestSetSuccessful(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, false, time.Second)

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			fd.setSuccessful()
		})
	}
	wg.Wait()

	select {
	case <-fd.hasBeenSuccessfulChan:
	default:
		require.FailNow(t, "hasBeenSuccessfulChan must be closed and readable")
	}

	select {
	case <-fd.hasBeenSuccessfulChan:
	default:
		require.FailNow(t, "hasBeenSuccessfulChan must remain closed and readable")
	}
}

func TestRequiresWebsocket(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, true, 0)
	require.True(t, fd.UseWebsocket())
	fd.useWebsocket = false
	require.False(t, fd.UseWebsocket())
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
			require.Equalf(t, tc.expected, RequiresAuth(fd.focusType), "RequiresAuth(%v) must match", tc.ft)
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
		FocusType(111):           "Unset/Unknown FocusType",
	}
	for in, exp := range cases {
		t.Run(in.String(), func(t *testing.T) {
			t.Parallel()
			require.Equalf(t, exp, in.String(), "FocusType(%d).String() must match", in)
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	// Spot ticker via REST
	fd := &FocusData{focusType: TickerFocusType, useWebsocket: false, restPollTime: time.Second}
	k := key.NewExchangeAssetPair(exchangeName, assetType, pair)
	require.NoError(t, fd.Validate(&k))
	fd = &FocusData{focusType: OpenInterestFocusType, useWebsocket: true}
	// Futures-specific type fails on spot asset
	require.ErrorIs(t, fd.Validate(&k), ErrInvalidAssetForFocusType)
	// Futures-specific type allowed on futures asset with websocket
	k = key.NewExchangeAssetPair(exchangeName, futuresAssetType, pair)
	require.NoError(t, fd.Validate(&k))
	// invalid REST poll time
	fd = &FocusData{focusType: TickerFocusType, useWebsocket: false, restPollTime: 0}
	require.ErrorIs(t, fd.Validate(&k), ErrInvalidRESTPollTime)
	fd = &FocusData{focusType: UnsetFocusType, useWebsocket: true}
	require.ErrorIs(t, fd.Validate(&k), ErrUnsetFocusType)
	// nil stuff
	fd = nil
	require.ErrorIs(t, fd.Validate(nil), common.ErrNilPointer)
}
