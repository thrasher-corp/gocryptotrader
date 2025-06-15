package binancecashandcarry

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	eventkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const testExchange = "binance"

func TestName(t *testing.T) {
	t.Parallel()
	d := Strategy{}
	if n := d.Name(); n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestDescription(t *testing.T) {
	t.Parallel()
	d := Strategy{}
	if n := d.Description(); n != description {
		t.Errorf("expected %v", description)
	}
}

func TestSupportsSimultaneousProcessing(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	if !s.SupportsSimultaneousProcessing() {
		t.Error("expected true")
	}
}

func TestSetCustomSettings(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	err := s.SetCustomSettings(nil)
	assert.NoError(t, err)

	float14 := float64(14)
	mappalopalous := make(map[string]any)
	mappalopalous[openShortDistancePercentageString] = float14
	mappalopalous[closeShortDistancePercentageString] = float14

	err = s.SetCustomSettings(mappalopalous)
	assert.NoError(t, err)

	mappalopalous[openShortDistancePercentageString] = "14"
	err = s.SetCustomSettings(mappalopalous)
	assert.ErrorIs(t, err, base.ErrInvalidCustomSettings)

	mappalopalous[closeShortDistancePercentageString] = float14
	mappalopalous[openShortDistancePercentageString] = "14"
	err = s.SetCustomSettings(mappalopalous)
	assert.ErrorIs(t, err, base.ErrInvalidCustomSettings)

	mappalopalous[closeShortDistancePercentageString] = float14
	mappalopalous["lol"] = float14
	err = s.SetCustomSettings(mappalopalous)
	assert.ErrorIs(t, err, base.ErrInvalidCustomSettings)
}

func TestOnSignal(t *testing.T) {
	t.Parallel()
	s := Strategy{
		openShortDistancePercentage: decimal.NewFromInt(14),
	}
	_, err := s.OnSignal(nil, nil, nil)
	assert.ErrorIs(t, err, base.ErrSimultaneousProcessingOnly)
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	s.SetDefaults()
	if !s.openShortDistancePercentage.Equal(decimal.NewFromInt(0)) {
		t.Errorf("expected 5, received %v", s.openShortDistancePercentage)
	}
	if !s.closeShortDistancePercentage.Equal(decimal.NewFromInt(0)) {
		t.Errorf("expected 5, received %v", s.closeShortDistancePercentage)
	}
}

func TestSortSignals(t *testing.T) {
	t.Parallel()
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := &data.Base{}
	err := d.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         dInsert,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	da := &datakline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	_, err = sortSignals([]data.Handler{da})
	assert.ErrorIs(t, err, errNotSetup)

	d2 := &data.Base{}
	err = d2.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Exchange:       exch,
			Time:           dInsert,
			Interval:       gctkline.OneDay,
			CurrencyPair:   currency.NewPair(currency.DOGE, currency.XRP),
			AssetType:      asset.Futures,
			UnderlyingPair: p,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	assert.NoError(t, err)

	_, err = d2.Next()
	assert.NoError(t, err)

	da2 := &datakline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d2,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	_, err = sortSignals([]data.Handler{da, da2})
	assert.NoError(t, err)
}

func TestCreateSignals(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.createSignals(nil, nil, nil, decimal.Zero, false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	spotSignal := &signal.Signal{
		Base: &event.Base{AssetType: asset.Spot},
	}
	_, err = s.createSignals(nil, spotSignal, nil, decimal.Zero, false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	// targeting first case
	futuresSignal := &signal.Signal{
		Base: &event.Base{AssetType: asset.Futures},
	}
	resp, err := s.createSignals(nil, spotSignal, futuresSignal, decimal.Zero, false)
	require.NoError(t, err, "createSignals must not error")
	require.Len(t, resp, 1, "createSignals must return one signal")
	assert.Equal(t, asset.Spot, resp[0].GetAssetType())

	// targeting second case:
	pos := []futures.Position{
		{
			Status: gctorder.Open,
		},
	}
	resp, err = s.createSignals(pos, spotSignal, futuresSignal, decimal.Zero, false)
	require.NoError(t, err, "createSignals must not error")
	require.Len(t, resp, 2, "createSignals must return two signals")
	caseTested := false
	for i := range resp {
		if resp[i].GetAssetType().IsFutures() {
			assert.Equal(t, gctorder.ClosePosition, resp[i].GetDirection())
			caseTested = true
			break
		}
	}
	require.True(t, caseTested, "Unhandled issue in test scenario")

	// targeting third case
	resp, err = s.createSignals(pos, spotSignal, futuresSignal, decimal.Zero, true)
	require.NoError(t, err, "createSignals must not error")
	require.Len(t, resp, 2, "createSignals must return two signals")

	caseTested = false
	for i := range resp {
		if resp[i].GetAssetType().IsFutures() {
			assert.Equal(t, gctorder.ClosePosition, resp[i].GetDirection())
			caseTested = true
			break
		}
	}
	require.True(t, caseTested, "Unhandled issue in test scenario")

	// targeting first case after a cash and carry is completed, have a new one opened
	pos[0].Status = gctorder.Closed
	resp, err = s.createSignals(pos, spotSignal, futuresSignal, decimal.NewFromInt(1337), true)
	require.NoError(t, err, "createSignals must not error")
	require.Len(t, resp, 1, "createSignals must return one signal")
	assert.Equal(t, asset.Spot, resp[0].GetAssetType())
	assert.Equal(t, gctorder.Buy, resp[0].GetDirection())
	assert.NotNil(t, resp[0].GetFillDependentEvent(), "GetFillDependentEvent should not return nil")

	// targeting default case
	pos[0].Status = gctorder.UnknownStatus
	resp, err = s.createSignals(pos, spotSignal, futuresSignal, decimal.NewFromInt(1337), true)
	require.NoError(t, err, "createSignals must not error")
	assert.Len(t, resp, 2, "createSignals should return two signals")
}

// fakeFunds overrides default implementation
type fakeFunds struct {
	funding.FundManager
	hasBeenLiquidated bool
}

// HasExchangeBeenLiquidated overrides default implementation
func (f fakeFunds) HasExchangeBeenLiquidated(_ common.Event) bool {
	return f.hasBeenLiquidated
}

// portfolerino overrides default implementation
type portfolerino struct {
	portfolio.Portfolio
}

// GetPositions overrides default implementation
func (p portfolerino) GetPositions(common.Event) ([]futures.Position, error) {
	return []futures.Position{
		{
			Exchange:           exchangeName,
			Asset:              asset.Spot,
			Pair:               currency.NewBTCUSD(),
			Underlying:         currency.BTC,
			CollateralCurrency: currency.USD,
		},
	}, nil
}

func TestOnSimultaneousSignals(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.OnSimultaneousSignals(nil, nil, nil)
	assert.ErrorIs(t, err, base.ErrNoDataToProcess)

	cp := currency.NewBTCUSD()
	d := &datakline.DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange:       exchangeName,
			Asset:          asset.Spot,
			Pair:           cp,
			UnderlyingPair: currency.NewBTCUSD(),
		},
	}
	tt := time.Now()
	err = d.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Exchange:     exchangeName,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: cp,
			AssetType:    asset.Spot,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	signals := []data.Handler{
		d,
	}
	f := &fakeFunds{}
	_, err = s.OnSimultaneousSignals(signals, f, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	p := &portfolerino{}
	_, err = s.OnSimultaneousSignals(signals, f, p)
	assert.ErrorIs(t, err, errNotSetup)

	d2 := &datakline.DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange:       exchangeName,
			Asset:          asset.Futures,
			Pair:           cp,
			UnderlyingPair: cp,
		},
	}
	err = d2.SetStream([]data.Event{&eventkline.Kline{
		Base: &event.Base{
			Exchange:       exchangeName,
			Time:           tt,
			Interval:       gctkline.OneDay,
			CurrencyPair:   cp,
			AssetType:      asset.Futures,
			UnderlyingPair: cp,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	assert.NoError(t, err)

	_, err = d2.Next()
	assert.NoError(t, err)

	signals = []data.Handler{
		d,
		d2,
	}
	resp, err := s.OnSimultaneousSignals(signals, f, p)
	assert.NoError(t, err)

	if len(resp) != 2 {
		t.Errorf("received '%v' expected '%v", len(resp), 2)
	}

	f.hasBeenLiquidated = true
	resp, err = s.OnSimultaneousSignals(signals, f, p)
	assert.NoError(t, err)

	if len(resp) != 2 {
		t.Fatalf("received '%v' expected '%v", len(resp), 2)
	}
	if resp[0].GetDirection() != gctorder.DoNothing {
		t.Errorf("received '%v' expected '%v", resp[0].GetDirection(), gctorder.DoNothing)
	}
}

func TestCloseAllPositions(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.CloseAllPositions(nil, nil)
	assert.NoError(t, err)

	leet := decimal.NewFromInt(1337)
	cp := currency.NewBTCUSD()
	h := []holdings.Holding{
		{
			Offset:   1,
			Item:     cp.Base,
			Pair:     cp,
			Asset:    asset.Spot,
			Exchange: testExchange,
		},
		{
			Offset:   1,
			Item:     cp.Base,
			Pair:     cp,
			Asset:    asset.Futures,
			Exchange: testExchange,
		},
	}

	p := []data.Event{
		&signal.Signal{
			Base: &event.Base{
				Offset:         1,
				Exchange:       testExchange,
				Time:           time.Now(),
				Interval:       gctkline.OneDay,
				CurrencyPair:   cp,
				UnderlyingPair: cp,
				AssetType:      asset.Spot,
			},
			OpenPrice:  leet,
			HighPrice:  leet,
			LowPrice:   leet,
			ClosePrice: leet,
			Volume:     leet,
			BuyLimit:   leet,
			SellLimit:  leet,
			Amount:     leet,
			Direction:  gctorder.Buy,
		},
		&signal.Signal{
			Base: &event.Base{
				Offset:         1,
				Exchange:       testExchange,
				Time:           time.Now(),
				Interval:       gctkline.OneDay,
				CurrencyPair:   cp,
				UnderlyingPair: cp,
				AssetType:      asset.Futures,
			},
			OpenPrice:  leet,
			HighPrice:  leet,
			LowPrice:   leet,
			ClosePrice: leet,
			Volume:     leet,
			BuyLimit:   leet,
			SellLimit:  leet,
			Amount:     leet,
			Direction:  gctorder.Buy,
		},
	}
	positionsToClose, err := s.CloseAllPositions(h, p)
	assert.NoError(t, err)

	if len(positionsToClose) != 2 {
		t.Errorf("received '%v' expected '%v", len(positionsToClose), 2)
	}
	if !positionsToClose[0].GetAssetType().IsFutures() {
		t.Errorf("received '%v' expected '%v", positionsToClose[0].GetAssetType(), asset.Futures)
	}
}
