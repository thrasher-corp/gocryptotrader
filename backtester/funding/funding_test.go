package funding

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	elite    = decimal.NewFromInt(1337)
	neg      = decimal.NewFromInt(-1)
	one      = decimal.NewFromInt(1)
	exchName = "exchname"
	a        = asset.Spot
	base     = currency.DOGE
	quote    = currency.XRP
	pair     = currency.NewPair(base, quote)
)

// fakeEvent implements common.EventHandler without
// caring about the response, or dealing with import cycles
type fakeEvent struct{}

func (f *fakeEvent) GetOffset() int64                         { return 0 }
func (f *fakeEvent) SetOffset(int64)                          {}
func (f *fakeEvent) IsEvent() bool                            { return true }
func (f *fakeEvent) GetTime() time.Time                       { return time.Now() }
func (f *fakeEvent) Pair() currency.Pair                      { return pair }
func (f *fakeEvent) GetExchange() string                      { return exchName }
func (f *fakeEvent) GetInterval() gctkline.Interval           { return gctkline.OneMin }
func (f *fakeEvent) GetAssetType() asset.Item                 { return asset.Spot }
func (f *fakeEvent) AppendReason(string)                      {}
func (f *fakeEvent) GetClosePrice() decimal.Decimal           { return elite }
func (f *fakeEvent) AppendReasonf(s string, i ...interface{}) {}
func (f *fakeEvent) GetBase() *event.Base                     { return &event.Base{} }
func (f *fakeEvent) GetUnderlyingPair() currency.Pair         { return pair }
func (f *fakeEvent) GetConcatReasons() string                 { return "" }
func (f *fakeEvent) GetReasons() []string                     { return nil }

func TestSetupFundingManager(t *testing.T) {
	t.Parallel()
	f, err := SetupFundingManager(&engine.ExchangeManager{}, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", true, false)
	}
	if f.disableUSDTracking {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	f, err = SetupFundingManager(&engine.ExchangeManager{}, false, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	if !f.disableUSDTracking {
		t.Errorf("expected '%v received '%v'", true, false)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	f, err := SetupFundingManager(&engine.ExchangeManager{}, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(baseItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	f.Reset()
	if f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	if f.Exists(baseItem) {
		t.Errorf("expected '%v received '%v'", false, true)
	}
}

func TestIsUsingExchangeLevelFunding(t *testing.T) {
	t.Parallel()
	f, err := SetupFundingManager(&engine.ExchangeManager{}, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !f.IsUsingExchangeLevelFunding() {
		t.Errorf("expected '%v received '%v'", true, false)
	}
}

func TestTransfer(t *testing.T) {
	t.Parallel()
	f := FundManager{
		usingExchangeLevelFunding: false,
		items:                     nil,
	}
	err := f.Transfer(decimal.Zero, nil, nil, false)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}
	err = f.Transfer(decimal.Zero, &Item{}, nil, false)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}
	err = f.Transfer(decimal.Zero, &Item{}, &Item{}, false)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = f.Transfer(elite, &Item{}, &Item{}, false)
	if !errors.Is(err, errNotEnoughFunds) {
		t.Errorf("received '%v' expected '%v'", err, errNotEnoughFunds)
	}
	item1 := &Item{exchange: "hello", asset: a, currency: base, available: elite}
	err = f.Transfer(elite, item1, item1, false)
	if !errors.Is(err, errCannotTransferToSameFunds) {
		t.Errorf("received '%v' expected '%v'", err, errCannotTransferToSameFunds)
	}

	item2 := &Item{exchange: "hello", asset: a, currency: quote}
	err = f.Transfer(elite, item1, item2, false)
	if !errors.Is(err, errTransferMustBeSameCurrency) {
		t.Errorf("received '%v' expected '%v'", err, errTransferMustBeSameCurrency)
	}

	item2.exchange = "moto"
	item2.currency = base
	err = f.Transfer(elite, item1, item2, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !item2.available.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", item2.available, elite)
	}
	if !item1.available.IsZero() {
		t.Errorf("received '%v' expected '%v'", item1.available, decimal.Zero)
	}

	item2.transferFee = one
	err = f.Transfer(elite, item2, item1, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !item1.available.Equal(elite.Sub(item2.transferFee)) {
		t.Errorf("received '%v' expected '%v'", item2.available, elite.Sub(item2.transferFee))
	}
}

func TestAddItem(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.AddItem(nil)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	baseItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(baseItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = f.AddItem(baseItem)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("received '%v' expected '%v'", err, ErrAlreadyExists)
	}
}

func TestExists(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	if f.Exists(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	conflictingSingleItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(conflictingSingleItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !f.Exists(conflictingSingleItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
	baseItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exchName, a, quote, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	p, err := CreatePair(baseItem, quoteItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddPair(p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	_, err = f.getFundingForEAP(exchName, a, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = f.getFundingForEAP(exchName, a, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	// demonstration that you don't need the original *Items
	// to check for existence, just matching fields
	baseCopy := Item{
		exchange:          baseItem.exchange,
		asset:             baseItem.asset,
		currency:          baseItem.currency,
		initialFunds:      baseItem.initialFunds,
		available:         baseItem.available,
		reserved:          baseItem.reserved,
		transferFee:       baseItem.transferFee,
		pairedWith:        baseItem.pairedWith,
		trackingCandles:   baseItem.trackingCandles,
		snapshot:          baseItem.snapshot,
		isCollateral:      baseItem.isCollateral,
		collateralCandles: baseItem.collateralCandles,
	}
	quoteCopy := Item{
		exchange:          quoteItem.exchange,
		asset:             quoteItem.asset,
		currency:          quoteItem.currency,
		initialFunds:      quoteItem.initialFunds,
		available:         quoteItem.available,
		reserved:          quoteItem.reserved,
		transferFee:       quoteItem.transferFee,
		pairedWith:        quoteItem.pairedWith,
		trackingCandles:   quoteItem.trackingCandles,
		snapshot:          quoteItem.snapshot,
		isCollateral:      quoteItem.isCollateral,
		collateralCandles: quoteItem.collateralCandles,
	}
	quoteCopy.pairedWith = &baseCopy
	if !f.Exists(&baseCopy) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	currFunds, err := f.getFundingForEAC(exchName, a, base)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if currFunds.pairedWith != nil {
		t.Errorf("received '%v' expected '%v'", nil, currFunds.pairedWith)
	}
}

func TestAddPair(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	p, err := CreatePair(baseItem, quoteItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddPair(p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	_, err = f.getFundingForEAP(exchName, a, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	p, err = CreatePair(baseItem, quoteItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddPair(p)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("received '%v' expected '%v'", err, ErrAlreadyExists)
	}
}

func TestGetFundingForEvent(t *testing.T) {
	t.Parallel()
	e := &fakeEvent{}
	f := FundManager{}
	_, err := f.GetFundingForEvent(e)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	p, err := CreatePair(baseItem, quoteItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddPair(p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	_, err = f.GetFundingForEvent(e)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestGetFundingForEAC(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	_, err := f.getFundingForEAC(exchName, a, base)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(baseItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	fundo, err := f.getFundingForEAC(exchName, a, base)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if !baseItem.Equal(fundo) {
		t.Errorf("received '%v' expected '%v'", baseItem, fundo)
	}
}

func TestGetFundingForEAP(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	_, err := f.getFundingForEAP(exchName, a, pair)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	p, err := CreatePair(baseItem, quoteItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddPair(p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	_, err = f.getFundingForEAP(exchName, a, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = CreatePair(baseItem, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}
	_, err = CreatePair(nil, quoteItem)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}
	p, err = CreatePair(baseItem, quoteItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddPair(p)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("received '%v' expected '%v'", err, ErrAlreadyExists)
	}
}

func TestGenerateReport(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	report := f.GenerateReport()
	if report == nil { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Fatal("shouldn't be nil")
	}
	if len(report.Items) > 0 { //nolint:staticcheck,nolintlint // SA5011 Ignore the nil warnings
		t.Error("expected 0")
	}
	item := &Item{
		exchange:     exchName,
		initialFunds: decimal.NewFromInt(100),
		available:    decimal.NewFromInt(200),
		currency:     currency.BTC,
		asset:        a,
	}
	err := f.AddItem(item)
	if err != nil {
		t.Fatal(err)
	}
	report = f.GenerateReport()
	if len(report.Items) != 1 {
		t.Fatal("expected 1")
	}
	if report.Items[0].Exchange != item.exchange {
		t.Error("expected matching name")
	}

	f.usingExchangeLevelFunding = true
	err = f.AddItem(&Item{
		exchange:     exchName,
		initialFunds: decimal.NewFromInt(100),
		available:    decimal.NewFromInt(200),
		currency:     currency.USD,
		asset:        a,
	})
	if err != nil {
		t.Fatal(err)
	}

	dfk := &kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: exchName,
			Pair:     currency.NewPair(currency.BTC, currency.USD),
			Asset:    a,
			Interval: gctkline.OneHour,
			Candles: []gctkline.Candle{
				{
					Time: time.Now(),
				},
			},
		},
	}
	err = dfk.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddUSDTrackingData(dfk)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	f.items[0].trackingCandles = dfk
	f.CreateSnapshot(dfk.Item.Candles[0].Time)

	report = f.GenerateReport()
	if len(report.Items) != 2 {
		t.Fatal("expected 2")
	}
	if report.Items[0].Exchange != item.exchange {
		t.Error("expected matching name")
	}
	if !report.Items[1].FinalFunds.Equal(decimal.NewFromInt(200)) {
		t.Errorf("received %v expected %v", report.Items[1].FinalFunds, decimal.NewFromInt(200))
	}
}

func TestCreateSnapshot(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	f.CreateSnapshot(time.Time{})
	f.items = append(f.items, &Item{})
	f.CreateSnapshot(time.Time{})

	dfk := &kline.DataFromKline{
		Item: gctkline.Item{
			Candles: []gctkline.Candle{
				{
					Time: time.Now(),
				},
			},
		},
	}
	if err := dfk.Load(); err != nil {
		t.Error(err)
	}
	f.items = append(f.items, &Item{
		exchange:        "test",
		asset:           asset.Spot,
		currency:        currency.BTC,
		initialFunds:    decimal.NewFromInt(1337),
		available:       decimal.NewFromInt(1337),
		reserved:        decimal.NewFromInt(1337),
		transferFee:     decimal.NewFromInt(1337),
		trackingCandles: dfk,
	})
	f.CreateSnapshot(dfk.Item.Candles[0].Time)
}

func TestAddUSDTrackingData(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.AddUSDTrackingData(nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}

	err = f.AddUSDTrackingData(&kline.DataFromKline{})
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}

	dfk := &kline.DataFromKline{
		Item: gctkline.Item{
			Candles: []gctkline.Candle{
				{
					Time: time.Now(),
				},
			},
		},
	}
	err = dfk.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(quoteItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	f.disableUSDTracking = true
	err = f.AddUSDTrackingData(dfk)
	if !errors.Is(err, ErrUSDTrackingDisabled) {
		t.Errorf("received '%v' expected '%v'", err, ErrUSDTrackingDisabled)
	}

	f.disableUSDTracking = false
	err = f.AddUSDTrackingData(dfk)
	if !errors.Is(err, errCannotMatchTrackingToItem) {
		t.Errorf("received '%v' expected '%v'", err, errCannotMatchTrackingToItem)
	}

	dfk = &kline.DataFromKline{
		Item: gctkline.Item{
			Exchange: exchName,
			Pair:     currency.NewPair(pair.Quote, currency.USD),
			Asset:    a,
			Interval: gctkline.OneHour,
			Candles: []gctkline.Candle{
				{
					Time: time.Now(),
				},
			},
		},
	}
	err = dfk.Load()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddUSDTrackingData(dfk)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	usdtItem, err := CreateItem(exchName, a, currency.USDT, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(usdtItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddUSDTrackingData(dfk)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestUSDTrackingDisabled(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	if f.USDTrackingDisabled() {
		t.Error("received true, expected false")
	}
	f.disableUSDTracking = true
	if !f.USDTrackingDisabled() {
		t.Error("received false, expected true")
	}
}

func TestFundingLiquidate(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	f.Liquidate(nil)
	f.items = append(f.items, &Item{
		exchange:  "test",
		asset:     asset.Spot,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1337),
	})

	f.Liquidate(&signal.Signal{
		Base: &event.Base{
			Exchange:     "test",
			AssetType:    asset.Spot,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
	})
	if !f.items[0].available.IsZero() {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, "0")
	}
}

func TestHasExchangeBeenLiquidated(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	f.Liquidate(nil)
	f.items = append(f.items, &Item{
		exchange:  "test",
		asset:     asset.Spot,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1337),
	})
	ev := &signal.Signal{
		Base: &event.Base{
			Exchange:     "test",
			AssetType:    asset.Spot,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
	}
	f.Liquidate(ev)
	if !f.items[0].available.IsZero() {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, "0")
	}
	if has := f.HasExchangeBeenLiquidated(ev); !has {
		t.Errorf("received '%v' expected '%v'", has, true)
	}
}

func TestGetAllFunding(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	resp := f.GetAllFunding()
	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}

	f.items = append(f.items, &Item{
		exchange:  "test",
		asset:     asset.Spot,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1337),
	})

	resp = f.GetAllFunding()
	if len(resp) != 1 {
		t.Errorf("received '%v' expected '%v'", len(resp), 1)
	}
}

func TestHasFutures(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	if has := f.HasFutures(); has {
		t.Errorf("received '%v' expected '%v'", has, false)
	}

	f.items = append(f.items, &Item{
		exchange:  "test",
		asset:     asset.Futures,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1337),
	})
	if has := f.HasFutures(); !has {
		t.Errorf("received '%v' expected '%v'", has, true)
	}
}

func TestRealisePNL(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	f.items = append(f.items, &Item{
		exchange:     "test",
		asset:        asset.Futures,
		currency:     currency.BTC,
		available:    decimal.NewFromInt(1336),
		isCollateral: true,
	})

	var expectedError error
	err := f.RealisePNL("test", asset.Futures, currency.BTC, decimal.NewFromInt(1))
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}
	if !f.items[0].available.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("recevied '%v' expected '%v'", f.items[0].available, decimal.NewFromInt(1337))
	}

	expectedError = ErrFundsNotFound
	err = f.RealisePNL("test2", asset.Futures, currency.BTC, decimal.NewFromInt(1))
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}
}

func TestCreateCollateral(t *testing.T) {
	t.Parallel()
	collat := &Item{
		exchange:     "test",
		asset:        asset.Futures,
		currency:     currency.BTC,
		available:    decimal.NewFromInt(1336),
		isCollateral: true,
	}
	contract := &Item{
		exchange:  "test",
		asset:     asset.Futures,
		currency:  currency.DOGE,
		available: decimal.NewFromInt(1336),
	}

	var expectedError error
	_, err := CreateCollateral(collat, contract)
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}

	expectedError = common.ErrNilArguments
	_, err = CreateCollateral(nil, contract)
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}

	_, err = CreateCollateral(collat, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}
}

func TestUpdateCollateral(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	expectedError := common.ErrNilEvent
	err := f.UpdateCollateral(nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}

	ev := &signal.Signal{
		Base: &event.Base{
			Exchange:     "ftx",
			AssetType:    asset.Futures,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
	}
	f.items = append(f.items, &Item{
		exchange:  "ftx",
		asset:     asset.Spot,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1336),
	})
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName("ftx")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	cfg, err := exch.GetDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	err = exch.Setup(cfg)
	if err != nil {
		t.Fatal(err)
	}
	em.Add(exch)
	f.exchangeManager = em

	expectedError = ErrFundsNotFound
	err = f.UpdateCollateral(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}

	expectedError = nil
	f.items = append(f.items, &Item{
		exchange:     "ftx",
		asset:        asset.Futures,
		currency:     currency.USD,
		available:    decimal.NewFromInt(1336),
		isCollateral: true,
	})
	err = f.UpdateCollateral(ev)
	if !errors.Is(err, expectedError) {
		t.Errorf("recevied '%v' expected '%v'", err, expectedError)
	}
}

func TestCreateFuturesCurrencyCode(t *testing.T) {
	t.Parallel()
	if result := CreateFuturesCurrencyCode(currency.BTC, currency.USDT); result != currency.NewCode("BTC-USDT") {
		t.Errorf("received '%v', expected  '%v'", result, "BTC-USDT")
	}
}

func TestLinkCollateralCurrency(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.LinkCollateralCurrency(nil, currency.EMPTYCODE)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v', expected  '%v'", err, common.ErrNilArguments)
	}

	item := &Item{}
	err = f.LinkCollateralCurrency(item, currency.EMPTYCODE)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v', expected  '%v'", err, common.ErrNilArguments)
	}

	err = f.LinkCollateralCurrency(item, currency.BTC)
	if !errors.Is(err, errNotFutures) {
		t.Errorf("received '%v', expected  '%v'", err, errNotFutures)
	}

	item.asset = asset.Futures
	err = f.LinkCollateralCurrency(item, currency.BTC)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}
	if !item.pairedWith.currency.Equal(currency.BTC) {
		t.Errorf("received '%v', expected  '%v'", currency.BTC, item.pairedWith.currency)
	}

	err = f.LinkCollateralCurrency(item, currency.LTC)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("received '%v', expected  '%v'", err, ErrAlreadyExists)
	}

	f.items = append(f.items, item.pairedWith)
	item.pairedWith = nil
	err = f.LinkCollateralCurrency(item, currency.BTC)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}
}
