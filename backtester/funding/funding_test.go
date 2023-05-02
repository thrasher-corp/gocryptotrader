package funding

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	elite    = decimal.NewFromInt(1337)
	neg      = decimal.NewFromInt(-1)
	one      = decimal.NewFromInt(1)
	exchName = "binance"
	a        = asset.Spot
	base     = currency.DOGE
	quote    = currency.XRP
	pair     = currency.NewPair(base, quote)
)

func TestSetupFundingManager(t *testing.T) {
	t.Parallel()
	f, err := SetupFundingManager(&engine.ExchangeManager{}, true, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", true, false)
	}
	if f.disableUSDTracking {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	f, err = SetupFundingManager(&engine.ExchangeManager{}, false, true, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	if !f.disableUSDTracking {
		t.Errorf("expected '%v received '%v'", true, false)
	}
	if !f.verbose {
		t.Errorf("expected '%v received '%v'", true, false)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	f, err := SetupFundingManager(&engine.ExchangeManager{}, true, false, false)
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
	err = f.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	if f.Exists(baseItem) {
		t.Errorf("expected '%v received '%v'", false, true)
	}
}

func TestIsUsingExchangeLevelFunding(t *testing.T) {
	t.Parallel()
	f, err := SetupFundingManager(&engine.ExchangeManager{}, true, false, false)
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
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	err = f.Transfer(decimal.Zero, &Item{}, nil, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
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
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	_, err = CreatePair(nil, quoteItem)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
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
	report, err := f.GenerateReport()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	err = f.AddItem(item)
	if err != nil {
		t.Fatal(err)
	}
	report, err = f.GenerateReport()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
		currency:     currency.USDT,
		asset:        a,
	})
	if err != nil {
		t.Fatal(err)
	}

	dfk := &kline.DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange: exchName,
			Pair:     currency.NewPair(currency.BTC, currency.USDT),
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
	err = f.CreateSnapshot(dfk.Item.Candles[0].Time)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	report, err = f.GenerateReport()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	err := f.CreateSnapshot(time.Time{})
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrDateUnset)
	}

	f.items = append(f.items, &Item{})
	dfk := &kline.DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Candles: []gctkline.Candle{
				{
					Time: time.Now(),
				},
			},
		},
	}
	err = dfk.Load()
	if !errors.Is(err, data.ErrInvalidEventSupplied) {
		t.Errorf("received '%v' expected '%v'", err, nil)
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
	err = f.CreateSnapshot(dfk.Item.Candles[0].Time)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestAddUSDTrackingData(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.AddUSDTrackingData(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	err = f.AddUSDTrackingData(kline.NewDataFromKline())
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	dfk := &kline.DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Candles: []gctkline.Candle{
				{
					Time: time.Now(),
				},
			},
		},
	}
	err = dfk.Load()
	if !errors.Is(err, data.ErrInvalidEventSupplied) {
		t.Errorf("received '%v' expected '%v'", err, data.ErrInvalidEventSupplied)
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
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange: exchName,
			Pair:     currency.NewPair(pair.Quote, currency.USDT),
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
	err := f.Liquidate(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	f.items = append(f.items, &Item{
		exchange:  "test",
		asset:     asset.Spot,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1337),
	})

	err = f.Liquidate(&signal.Signal{
		Base: &event.Base{
			Exchange:     "test",
			AssetType:    asset.Spot,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !f.items[0].available.IsZero() {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, "0")
	}
}

func TestHasExchangeBeenLiquidated(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.Liquidate(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
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
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
		},
	}
	err = f.Liquidate(ev)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
	resp, err := f.GetAllFunding()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}

	f.items = append(f.items, &Item{
		exchange:  "test",
		asset:     asset.Spot,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1337),
	})

	resp, err = f.GetAllFunding()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
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
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
	if !f.items[0].available.Equal(decimal.NewFromInt(1337)) {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, decimal.NewFromInt(1337))
	}

	expectedError = ErrFundsNotFound
	err = f.RealisePNL("test2", asset.Futures, currency.BTC, decimal.NewFromInt(1))
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
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
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = gctcommon.ErrNilPointer
	_, err = CreateCollateral(nil, contract)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	_, err = CreateCollateral(collat, nil)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}
}

func TestUpdateCollateral(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	expectedError := common.ErrNilEvent
	err := f.UpdateCollateralForEvent(nil, false)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	ev := &signal.Signal{
		Base: &event.Base{
			Exchange:     exchName,
			AssetType:    asset.Futures,
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
	}
	f.items = append(f.items, &Item{
		exchange:  exchName,
		asset:     asset.Spot,
		currency:  currency.BTC,
		available: decimal.NewFromInt(1336),
	})
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(exchName)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	err = em.Add(exch)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	f.exchangeManager = em

	expectedError = nil
	err = f.UpdateCollateralForEvent(ev, false)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
	}

	expectedError = gctcommon.ErrNotYetImplemented
	f.items = append(f.items, &Item{
		exchange:     exchName,
		asset:        asset.Futures,
		currency:     currency.USD,
		available:    decimal.NewFromInt(1336),
		isCollateral: true,
	})
	err = f.UpdateCollateralForEvent(ev, false)
	if !errors.Is(err, expectedError) {
		t.Errorf("received '%v' expected '%v'", err, expectedError)
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
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v', expected  '%v'", err, gctcommon.ErrNilPointer)
	}

	item := &Item{}
	err = f.LinkCollateralCurrency(item, currency.EMPTYCODE)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v', expected  '%v'", err, gctcommon.ErrNilPointer)
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

func TestSetFunding(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	err := f.SetFunding("", 0, nil, false)
	if !errors.Is(err, engine.ErrExchangeNameIsEmpty) {
		t.Errorf("received '%v', expected  '%v'", err, engine.ErrExchangeNameIsEmpty)
	}

	err = f.SetFunding(exchName, 0, nil, false)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v', expected  '%v'", err, asset.ErrNotSupported)
	}

	err = f.SetFunding(exchName, asset.Spot, nil, false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v', expected  '%v'", err, gctcommon.ErrNilPointer)
	}

	bal := &account.Balance{}
	err = f.SetFunding(exchName, asset.Spot, bal, false)
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Errorf("received '%v', expected  '%v'", err, currency.ErrCurrencyCodeEmpty)
	}

	bal.Currency = currency.BTC
	bal.Total = 1337
	err = f.SetFunding(exchName, asset.Spot, bal, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}
	if len(f.items) != 1 {
		t.Fatalf("received '%v' expected '%v'", len(f.items), 1)
	}
	if !f.items[0].available.Equal(leet) {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, bal.Total)
	}
	if !f.items[0].initialFunds.Equal(leet) {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, bal.Total)
	}

	bal.Total = 1338
	err = f.SetFunding(exchName, asset.Spot, bal, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}
	if !f.items[0].available.Equal(decimal.NewFromFloat(bal.Total)) {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, bal.Total)
	}
	if !f.items[0].initialFunds.Equal(leet) {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, leet)
	}
}

func TestUpdateFundingFromLiveData(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	err := f.UpdateFundingFromLiveData(false)
	if !errors.Is(err, engine.ErrNilSubsystem) {
		t.Errorf("received '%v', expected  '%v'", err, engine.ErrNilSubsystem)
	}

	f.exchangeManager = engine.NewExchangeManager()
	err = f.UpdateFundingFromLiveData(false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}

	ff := &binance.Binance{}
	ff.SetDefaults()
	err = f.exchangeManager.Add(ff)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	err = f.UpdateFundingFromLiveData(false)
	if !errors.Is(err, exchange.ErrCredentialsAreEmpty) {
		t.Errorf("received '%v', expected  '%v'", err, exchange.ErrCredentialsAreEmpty)
	}

	// enter api keys to gain coverage here
	apiKey := ""
	apiSec := ""
	subAccount := ""
	if apiKey == "" || apiSec == "" {
		// this test requires auth to get coverage
		return
	}
	ff.SetCredentials(apiKey, apiSec, "", subAccount, "", "")

	err = f.UpdateFundingFromLiveData(true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}

	err = f.UpdateFundingFromLiveData(false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}
}

func TestUpdateAllCollateral(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	err := f.UpdateAllCollateral(false, false)
	if !errors.Is(err, engine.ErrNilSubsystem) {
		t.Errorf("received '%v', expected  '%v'", err, engine.ErrNilSubsystem)
	}

	f.exchangeManager = engine.NewExchangeManager()
	err = f.UpdateAllCollateral(false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}

	ff := &binance.Binance{}
	ff.SetDefaults()
	err = f.exchangeManager.Add(ff)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	err = f.UpdateAllCollateral(false, false)
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received '%v', expected  '%v'", err, gctcommon.ErrNotYetImplemented)
	}

	f.items = []*Item{
		{
			exchange:     exchName,
			asset:        asset.Spot,
			currency:     currency.BTC,
			isCollateral: true,
		},
	}
	err = f.UpdateAllCollateral(false, false)
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received '%v', expected  '%v'", err, gctcommon.ErrNotYetImplemented)
	}

	f.items[0].trackingCandles = kline.NewDataFromKline()
	err = f.items[0].trackingCandles.SetStream([]data.Event{
		&fakeEvent{},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}

	err = f.UpdateAllCollateral(false, false)
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received '%v', expected  '%v'", err, gctcommon.ErrNotYetImplemented)
	}

	f.items[0].asset = asset.Futures
	err = f.UpdateAllCollateral(false, false)
	if !errors.Is(err, gctcommon.ErrNotYetImplemented) {
		t.Errorf("received '%v', expected  '%v'", err, gctcommon.ErrNotYetImplemented)
	}

	apiKey := ""
	apiSec := ""
	subAccount := ""
	if apiKey == "" || apiSec == "" {
		// this test requires auth to get coverage
		return
	}
	ff.SetCredentials(apiKey, apiSec, "", subAccount, "", "")
	err = f.UpdateAllCollateral(true, true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v', expected  '%v'", err, nil)
	}
}

var leet = decimal.NewFromInt(1337)

// fakeEvent implements common.Event without
// caring about the response, or dealing with import cycles
type fakeEvent struct{}

func (f *fakeEvent) GetHighPrice() decimal.Decimal            { return leet }
func (f *fakeEvent) GetLowPrice() decimal.Decimal             { return leet }
func (f *fakeEvent) GetOpenPrice() decimal.Decimal            { return leet }
func (f *fakeEvent) GetVolume() decimal.Decimal               { return leet }
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
func (f *fakeEvent) AppendReasonf(_ string, _ ...interface{}) {}
func (f *fakeEvent) GetBase() *event.Base                     { return &event.Base{} }
func (f *fakeEvent) GetUnderlyingPair() currency.Pair         { return pair }
func (f *fakeEvent) GetConcatReasons() string                 { return "" }
func (f *fakeEvent) GetReasons() []string                     { return nil }
