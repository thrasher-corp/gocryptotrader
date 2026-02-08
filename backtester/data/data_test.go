package data

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	exch = "binance"
	a    = asset.Spot
)

var p = currency.NewBTCUSD()

type fakeEvent struct {
	secretID int64
	*event.Base
}

type fakeHandler struct{}

func TestSetDataForCurrency(t *testing.T) {
	t.Parallel()
	d := HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, nil)
	assert.NoError(t, err)

	if d.data == nil {
		t.Error("expected not nil")
	}
	if d.data[key.NewExchangeAssetPair(exch, a, p)] != nil {
		t.Error("expected nil")
	}
}

func TestGetAllData(t *testing.T) {
	t.Parallel()
	d := HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, nil)
	assert.NoError(t, err)

	err = d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	assert.NoError(t, err)

	result, err := d.GetAllData()
	require.NoError(t, err)
	assert.Len(t, result, 2, "GetAllData should return 2 items")
}

func TestGetDataForCurrency(t *testing.T) {
	t.Parallel()
	d := HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, &fakeHandler{})
	assert.NoError(t, err)

	err = d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	assert.NoError(t, err)

	_, err = d.GetDataForCurrency(nil)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	_, err = d.GetDataForCurrency(&fakeEvent{Base: &event.Base{
		Exchange:     "lol",
		AssetType:    asset.USDTMarginedFutures,
		CurrencyPair: currency.NewPair(currency.EMB, currency.DOGE),
	}})
	assert.ErrorIs(t, err, ErrHandlerNotFound)

	_, err = d.GetDataForCurrency(&fakeEvent{Base: &event.Base{
		Exchange:     exch,
		AssetType:    a,
		CurrencyPair: p,
	}})
	assert.NoError(t, err)
}

func TestReset(t *testing.T) {
	t.Parallel()
	d := &HandlerHolder{}
	err := d.SetDataForCurrency(exch, a, p, nil)
	assert.NoError(t, err)

	err = d.SetDataForCurrency(exch, a, currency.NewPair(currency.BTC, currency.DOGE), nil)
	assert.NoError(t, err)

	err = d.Reset()
	require.NoError(t, err)

	assert.NotNil(t, d.data, "Reset should initialise the data map")
	d = nil
	err = d.Reset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestBaseReset(t *testing.T) {
	t.Parallel()
	b := &Base{offset: 1}
	err := b.Reset()
	require.NoError(t, err)
	assert.Zero(t, b.offset, "offset should be reset")
	b = nil
	err = b.Reset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestGetStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	resp, err := b.GetStream()
	require.NoError(t, err)
	assert.Empty(t, resp, "GetStream should return an empty slice")
	b.stream = []Event{
		&fakeEvent{
			Base: &event.Base{
				Offset: 2048,
				Time:   time.Now(),
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset: 1337,
				Time:   time.Now().Add(-time.Hour),
			},
		},
	}
	resp, err = b.GetStream()
	require.NoError(t, err)
	assert.Len(t, resp, 2, "GetStream should return 2 items")

	b = nil
	_, err = b.GetStream()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestOffset(t *testing.T) {
	t.Parallel()
	b := &Base{}
	o, err := b.Offset()
	require.NoError(t, err)
	assert.Zero(t, o, "offset should be zero when not set")

	b.offset = 1337
	o, err = b.Offset()
	require.NoError(t, err)

	assert.Equal(t, int64(1337), o, "offset value should be correct")

	b = nil
	_, err = b.Offset()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestSetStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(nil)
	require.NoError(t, err)
	assert.Empty(t, b.stream, "SetStream should not error with nil slice and stream should be empty")

	cp := currency.NewBTCUSD()
	err = b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	require.NoError(t, err)
	assert.Len(t, b.stream, 2, "stream elements should be set correctly")
	assert.Equal(t, int64(1), b.stream[0].GetOffset(), "GetOffset should return the correct value")
	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
		},
	}
	err = b.SetStream([]Event{misMatchEvent})
	require.ErrorIs(t, err, ErrInvalidEventSupplied)

	misMatchEvent.Time = time.Now()
	err = b.SetStream([]Event{misMatchEvent})
	require.ErrorIs(t, err, errMismatchedEvent)

	err = b.SetStream([]Event{nil})
	require.ErrorIs(t, err, gctcommon.ErrNilPointer)

	b = nil
	err = b.SetStream(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestNext(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	require.NoError(t, err)

	resp, err := b.Next()
	require.NoError(t, err)
	assert.Equal(t, b.stream[0], resp, "Next should return the first event in the stream")
	assert.Equal(t, int64(1), b.offset, "offset should be correct")

	_, err = b.Next()
	require.NoError(t, err)

	resp, err = b.Next()
	require.ErrorIs(t, err, ErrEndOfData)
	assert.Nil(t, resp, "Expected nil response after end of data")

	b = nil
	_, err = b.Next()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestHistory(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	require.NoError(t, err)

	resp, err := b.History()
	require.NoError(t, err)
	assert.Empty(t, resp, "History should return an empty slice when no events have been processed")

	_, err = b.Next()
	require.NoError(t, err)

	resp, err = b.History()
	require.NoError(t, err)
	assert.Len(t, resp, 1, "History should return the first event after one Next call")

	b = nil
	_, err = b.History()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestLatest(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	require.NoError(t, err)

	resp, err := b.Latest()
	require.NoError(t, err)

	assert.Equal(t, b.stream[0], resp, "Latest should return the first event in the stream")

	_, err = b.Next()
	require.NoError(t, err)

	resp, err = b.Latest()
	require.NoError(t, err)
	assert.Equal(t, b.stream[0], resp, "Latest should return the first event after one Next call")

	_, err = b.Next()
	require.NoError(t, err)

	resp, err = b.Latest()
	require.NoError(t, err)
	assert.Equal(t, b.stream[1], resp, "Latest should return the second event after two Next calls")

	b = nil
	_, err = b.Latest()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestList(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	require.NoError(t, err)

	list, err := b.List()
	require.NoError(t, err)
	assert.Len(t, list, 2, "List should return all events in the stream")

	b = nil
	_, err = b.List()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestIsLastEvent(t *testing.T) {
	t.Parallel()
	b := &Base{}
	cp := currency.NewBTCUSD()
	err := b.SetStream([]Event{
		&fakeEvent{
			Base: &event.Base{
				Offset:       2048,
				Time:         time.Now(),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset:       1337,
				Time:         time.Now().Add(-time.Hour),
				Exchange:     "test",
				AssetType:    asset.Spot,
				CurrencyPair: cp,
			},
		},
	})
	require.NoError(t, err)

	b.latest = b.stream[0]
	b.offset = b.stream[0].GetOffset()
	isLastEvent, err := b.IsLastEvent()
	require.NoError(t, err)
	assert.False(t, isLastEvent, "isLastEvent should return false when not at the last event")

	b.isLiveData = true
	isLastEvent, err = b.IsLastEvent()
	require.NoError(t, err)
	assert.False(t, isLastEvent, "isLastEvent should return false when live data is set")

	b = nil
	_, err = b.IsLastEvent()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestIsLive(t *testing.T) {
	t.Parallel()
	b := &Base{}
	isLive, err := b.IsLive()
	assert.NoError(t, err)

	if isLive {
		t.Error("expected false")
	}
	b.isLiveData = true
	isLive, err = b.IsLive()
	assert.NoError(t, err)

	if !isLive {
		t.Error("expected true")
	}

	b = nil
	_, err = b.IsLive()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestSetLive(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetLive(true)
	assert.NoError(t, err)

	if !b.isLiveData {
		t.Error("expected true")
	}

	err = b.SetLive(false)
	assert.NoError(t, err)

	if b.isLiveData {
		t.Error("expected false")
	}

	b = nil
	err = b.SetLive(false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestAppendStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	e := &fakeEvent{
		Base: &event.Base{},
	}
	err := b.AppendStream(e)
	assert.ErrorIs(t, err, ErrInvalidEventSupplied)

	if len(b.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 0)
	}
	tt := time.Now().Add(-time.Hour)
	cp := currency.NewBTCUSD()
	e.Exchange = "b"
	e.AssetType = asset.Spot
	e.CurrencyPair = cp
	err = b.AppendStream(e)
	require.ErrorIs(t, err, ErrInvalidEventSupplied)

	e.Time = tt
	err = b.AppendStream(e, e)
	require.NoError(t, err)

	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	err = b.AppendStream(e)
	require.NoError(t, err)

	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	err = b.AppendStream(&fakeEvent{
		Base: &event.Base{
			Exchange:     "b",
			AssetType:    asset.Spot,
			CurrencyPair: cp,
			Time:         time.Now(),
		},
	})
	require.NoError(t, err)

	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	misMatchEvent := &fakeEvent{
		Base: &event.Base{
			Exchange:     "mismatch",
			CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
			AssetType:    asset.Futures,
			Time:         tt,
		},
	}
	err = b.AppendStream(misMatchEvent)
	require.ErrorIs(t, err, errMismatchedEvent)

	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	err = b.AppendStream(nil)
	require.ErrorIs(t, err, gctcommon.ErrNilPointer)

	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	err = b.AppendStream()
	require.ErrorIs(t, err, errNothingToAdd)

	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	b = nil
	err = b.AppendStream()
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestFirst(t *testing.T) {
	t.Parallel()
	var id1 int64 = 1
	var id2 int64 = 2
	var id3 int64 = 3
	e := Events{
		fakeEvent{secretID: id1},
		fakeEvent{secretID: id2},
		fakeEvent{secretID: id3},
	}

	first, err := e.First()
	require.NoError(t, err)
	assert.Equal(t, id1, first.GetOffset())
}

func TestLast(t *testing.T) {
	t.Parallel()
	var id1 int64 = 1
	var id2 int64 = 2
	var id3 int64 = 3
	e := Events{
		fakeEvent{secretID: id1},
		fakeEvent{secretID: id2},
		fakeEvent{secretID: id3},
	}

	last, err := e.Last()
	require.NoError(t, err)
	assert.Equal(t, id3, last.GetOffset())
}

func (f fakeEvent) GetOffset() int64 {
	if f.secretID > 0 {
		return f.secretID
	}
	return f.Offset
}

func (f fakeEvent) SetOffset(o int64) {
	f.Offset = o
}

func (f fakeEvent) IsEvent() bool {
	return false
}

func (f fakeEvent) GetTime() time.Time {
	return f.Base.Time
}

func (f fakeEvent) Pair() currency.Pair {
	return currency.NewBTCUSD()
}

func (f fakeEvent) GetExchange() string {
	return f.Exchange
}

func (f fakeEvent) GetInterval() gctkline.Interval {
	return gctkline.Interval(time.Minute)
}

func (f fakeEvent) GetAssetType() asset.Item {
	return f.AssetType
}

func (f fakeEvent) GetReason() string {
	return strings.Join(f.Reasons, ",")
}

func (f fakeEvent) AppendReason(string) {
}

func (f fakeEvent) GetClosePrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetHighPrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetLowPrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetOpenPrice() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetVolume() decimal.Decimal {
	return decimal.Zero
}

func (f fakeEvent) GetUnderlyingPair() currency.Pair {
	return f.Pair()
}

func (f fakeEvent) AppendReasonf(string, ...any) {}

func (f fakeEvent) GetBase() *event.Base {
	return &event.Base{}
}

func (f fakeEvent) GetConcatReasons() string {
	return ""
}

func (f fakeEvent) GetReasons() []string {
	return nil
}

func (f fakeHandler) Load() error {
	return nil
}

func (f fakeHandler) AppendStream(...Event) error {
	return nil
}

func (f fakeHandler) GetBase() Base {
	return Base{}
}

func (f fakeHandler) Next() (Event, error) {
	return nil, nil
}

func (f fakeHandler) GetStream() (Events, error) {
	return nil, nil
}

func (f fakeHandler) History() (Events, error) {
	return nil, nil
}

func (f fakeHandler) Latest() (Event, error) {
	return nil, nil
}

func (f fakeHandler) List() (Events, error) {
	return nil, nil
}

func (f fakeHandler) IsLastEvent() (bool, error) {
	return false, nil
}

func (f fakeHandler) Offset() (int64, error) {
	return 0, nil
}

func (f fakeHandler) StreamOpen() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamHigh() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamLow() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamClose() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) StreamVol() ([]decimal.Decimal, error) {
	return nil, nil
}

func (f fakeHandler) HasDataAtTime(time.Time) (bool, error) {
	return false, nil
}

func (f fakeHandler) Reset() error {
	return nil
}

func (f fakeHandler) GetDetails() (string, asset.Item, currency.Pair, error) {
	return "", asset.Empty, currency.EMPTYPAIR, nil
}
