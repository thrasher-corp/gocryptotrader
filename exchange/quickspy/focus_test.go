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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

func TestNewFocusDataAndInit(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, false, time.Second)
	if fd == nil {
		t.Fatal("NewFocusData returned nil")
	}
	if fd.Type != TickerFocusType || fd.UseWebsocket || fd.IsOnceOff || fd.RESTPollTime != time.Second {
		t.Fatalf("unexpected FocusData fields: %+v", fd)
	}
	if fd.m == nil {
		t.Fatal("mutex not initialised")
	}
	if fd.HasBeenSuccessfulChan == nil || fd.Stream == nil {
		t.Fatal("channels not initialised")
	}
	select {
	case <-fd.HasBeenSuccessfulChan:
		t.Fatal("HasBeenSuccessfulChan should not be closed initially")
	default:
	}

	// Re-init should recreate channels and reset success flag
	fd.SetSuccessful()
	// channel must be closed now
	select {
	case <-fd.HasBeenSuccessfulChan:
	default:
		t.Fatal("HasBeenSuccessfulChan should be closed after SetSuccessful")
	}
	oldChan := fd.HasBeenSuccessfulChan
	fd.Init()
	if fd.m == nil || fd.HasBeenSuccessfulChan == nil || fd.Stream == nil {
		t.Fatal("Init did not re-initialise state")
	}
	if oldChan == fd.HasBeenSuccessfulChan {
		t.Fatal("Init should create a new HasBeenSuccessfulChan")
	}
	select {
	case <-fd.HasBeenSuccessfulChan:
		t.Fatal("HasBeenSuccessfulChan should not be closed after re-Init")
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
			fd.SetSuccessful()
		}()
	}
	wg.Wait()

	// channel should be closed and reads should not block
	select {
	case <-fd.HasBeenSuccessfulChan:
	default:
		t.Fatal("HasBeenSuccessfulChan should be closed and readable")
	}
	// multiple reads on a closed channel should still proceed immediately
	select {
	case <-fd.HasBeenSuccessfulChan:
	default:
		t.Fatal("HasBeenSuccessfulChan should remain closed and readable")
	}
}

func TestRequiresWebsocket(t *testing.T) {
	t.Parallel()
	fd := NewFocusData(TickerFocusType, false, true, 0)
	if !fd.RequiresWebsocket() {
		t.Fatal("expected RequiresWebsocket to be true")
	}
	fd.UseWebsocket = false
	if fd.RequiresWebsocket() {
		t.Fatal("expected RequiresWebsocket to be false")
	}
}

func TestRequiresAuth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ft       FocusType
		expected bool
	}{
		{AccountHoldingsFocusType, true},
		{ActiveOrdersFocusType, true},
		{OrderPlacementFocusType, true},
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
			if got := fd.RequiresAuth(); got != tc.expected {
				t.Fatalf("RequiresAuth(%v) = %v, expected %v", tc.ft, got, tc.expected)
			}
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
		OrderPlacementFocusType:  "OrderPlacementFocusType",
		KlineFocusType:           "KlineFocusType",
		ContractFocusType:        "ContractFocusType",
		OrderLimitsFocusType:     "OrderLimitsFocusType",
		URLFocusType:             "URLFocusType",
		FocusType(999):           "Unset/Unknown FocusType",
	}
	for in, exp := range cases {
		t.Run(in.String(), func(t *testing.T) {
			t.Parallel()
			if got := in.String(); got != exp {
				t.Fatalf("FocusType(%d).String() = %q, expected %q", in, got, exp)
			}
		})

	}
}

func TestFocusToSubMap(t *testing.T) {
	t.Parallel()
	if s, ok := focusToSub[OrderBookFocusType]; !ok || s != subscription.OrderbookChannel {
		t.Fatalf("focusToSub[OrderBookFocusType] = %q, ok=%v", s, ok)
	}
	if s, ok := focusToSub[TickerFocusType]; !ok || s != subscription.TickerChannel {
		t.Fatalf("focusToSub[TickerFocusType] = %q, ok=%v", s, ok)
	}
	if s, ok := focusToSub[KlineFocusType]; !ok || s != subscription.CandlesChannel {
		t.Fatalf("focusToSub[KlineFocusType] = %q, ok=%v", s, ok)
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
	fd := &FocusData{Type: TickerFocusType, UseWebsocket: false, RESTPollTime: time.Second}
	k := makeCredKey(t, asset.Spot, nil)
	require.NoError(t, fd.Validate(k))
	// Futures-specific type allowed on futures asset with websocket
	fd = &FocusData{Type: OpenInterestFocusType, UseWebsocket: true}
	k = makeCredKey(t, asset.Futures, nil)
	require.NoError(t, fd.Validate(k))
	// Futures-specific type fails on spot asset
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), ErrInvalidAssetForFocusType)
	// Auth-required type passes when credentials are provided
	fd = &FocusData{Type: AccountHoldingsFocusType, UseWebsocket: false, RESTPollTime: time.Second}
	k = makeCredKey(t, asset.Spot, &account.Credentials{})
	require.ErrorIs(t, fd.Validate(k), ErrNoCredentials)
	// OrderPlacementFocusType does not require credentials in Validate
	fd = &FocusData{Type: OrderPlacementFocusType, UseWebsocket: false, RESTPollTime: time.Second}
	k = makeCredKey(t, asset.Spot, nil)
	require.NoError(t, fd.Validate(k), ErrNoCredentials)
	// invalid REST poll time
	fd = &FocusData{Type: TickerFocusType, UseWebsocket: false, RESTPollTime: 0}
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), ErrInvalidRESTPollTime)
	fd = &FocusData{Type: UnsetFocusType, UseWebsocket: true}
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), ErrUnsetFocusType)
	// nil stuff
	fd = nil
	k = makeCredKey(t, asset.Spot, nil)
	require.ErrorIs(t, fd.Validate(k), common.ErrNilPointer)
}
