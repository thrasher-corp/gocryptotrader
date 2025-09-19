package funding

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	assert.NoError(t, err)

	if !f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", true, false)
	}
	if f.disableUSDTracking {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	f, err = SetupFundingManager(&engine.ExchangeManager{}, false, true, true)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	baseItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	err = f.AddItem(baseItem)
	assert.NoError(t, err)

	err = f.Reset()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	err = f.Transfer(decimal.Zero, &Item{}, nil, false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	err = f.Transfer(decimal.Zero, &Item{}, &Item{}, false)
	assert.ErrorIs(t, err, errZeroAmountReceived)

	err = f.Transfer(elite, &Item{}, &Item{}, false)
	assert.ErrorIs(t, err, errNotEnoughFunds)

	item1 := &Item{exchange: "hello", asset: a, currency: base, available: elite}
	err = f.Transfer(elite, item1, item1, false)
	assert.ErrorIs(t, err, errCannotTransferToSameFunds)

	item2 := &Item{exchange: "hello", asset: a, currency: quote}
	err = f.Transfer(elite, item1, item2, false)
	assert.ErrorIs(t, err, errTransferMustBeSameCurrency)

	item2.exchange = "moto"
	item2.currency = base
	err = f.Transfer(elite, item1, item2, false)
	assert.NoError(t, err)

	if !item2.available.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", item2.available, elite)
	}
	if !item1.available.IsZero() {
		t.Errorf("received '%v' expected '%v'", item1.available, decimal.Zero)
	}

	item2.transferFee = one
	err = f.Transfer(elite, item2, item1, true)
	assert.NoError(t, err)

	if !item1.available.Equal(elite.Sub(item2.transferFee)) {
		t.Errorf("received '%v' expected '%v'", item2.available, elite.Sub(item2.transferFee))
	}
}

func TestAddItem(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.AddItem(nil)
	assert.NoError(t, err)

	baseItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	err = f.AddItem(baseItem)
	assert.NoError(t, err)

	err = f.AddItem(baseItem)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestExists(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	if f.Exists(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	conflictingSingleItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	err = f.AddItem(conflictingSingleItem)
	assert.NoError(t, err)

	if !f.Exists(conflictingSingleItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
	baseItem, err := CreateItem(exchName, a, base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quoteItem, err := CreateItem(exchName, a, quote, elite, decimal.Zero)
	assert.NoError(t, err)

	p, err := CreatePair(baseItem, quoteItem)
	assert.NoError(t, err)

	err = f.AddPair(p)
	assert.NoError(t, err)

	_, err = f.getFundingForEAP(exchName, a, pair)
	assert.NoError(t, err)

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
	if !f.Exists(&baseCopy) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
}

func TestAddPair(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	assert.NoError(t, err)

	p, err := CreatePair(baseItem, quoteItem)
	assert.NoError(t, err)

	err = f.AddPair(p)
	assert.NoError(t, err)

	_, err = f.getFundingForEAP(exchName, a, pair)
	assert.NoError(t, err)

	p, err = CreatePair(baseItem, quoteItem)
	assert.NoError(t, err)

	err = f.AddPair(p)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestGetFundingForEvent(t *testing.T) {
	t.Parallel()
	e := &fakeEvent{}
	f := FundManager{}
	_, err := f.GetFundingForEvent(e)
	assert.ErrorIs(t, err, ErrFundsNotFound)

	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	assert.NoError(t, err)

	p, err := CreatePair(baseItem, quoteItem)
	assert.NoError(t, err)

	err = f.AddPair(p)
	assert.NoError(t, err)

	_, err = f.GetFundingForEvent(e)
	assert.NoError(t, err)
}

func TestGetFundingForEAP(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	_, err := f.getFundingForEAP(exchName, a, pair)
	assert.ErrorIs(t, err, ErrFundsNotFound)

	baseItem, err := CreateItem(exchName, a, pair.Base, decimal.Zero, decimal.Zero)
	assert.NoError(t, err)

	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	assert.NoError(t, err)

	p, err := CreatePair(baseItem, quoteItem)
	assert.NoError(t, err)

	err = f.AddPair(p)
	assert.NoError(t, err)

	_, err = f.getFundingForEAP(exchName, a, pair)
	assert.NoError(t, err)

	_, err = CreatePair(baseItem, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	_, err = CreatePair(nil, quoteItem)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	p, err = CreatePair(baseItem, quoteItem)
	assert.NoError(t, err)

	err = f.AddPair(p)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestGenerateReport(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	report, err := f.GenerateReport()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
			Pair:     currency.NewBTCUSDT(),
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
	assert.NoError(t, err)

	err = f.AddUSDTrackingData(dfk)
	assert.NoError(t, err)

	f.items[0].trackingCandles = dfk
	err = f.CreateSnapshot(dfk.Item.Candles[0].Time)
	assert.NoError(t, err)

	report, err = f.GenerateReport()
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, gctcommon.ErrDateUnset)

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
	assert.ErrorIs(t, err, data.ErrInvalidEventSupplied)

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
	assert.NoError(t, err)
}

func TestAddUSDTrackingData(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.AddUSDTrackingData(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	err = f.AddUSDTrackingData(kline.NewDataFromKline())
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

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
	assert.ErrorIs(t, err, data.ErrInvalidEventSupplied)

	quoteItem, err := CreateItem(exchName, a, pair.Quote, elite, decimal.Zero)
	assert.NoError(t, err)

	err = f.AddItem(quoteItem)
	assert.NoError(t, err)

	f.disableUSDTracking = true
	err = f.AddUSDTrackingData(dfk)
	assert.ErrorIs(t, err, ErrUSDTrackingDisabled)

	f.disableUSDTracking = false
	err = f.AddUSDTrackingData(dfk)
	assert.ErrorIs(t, err, errCannotMatchTrackingToItem)

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
	assert.NoError(t, err)

	err = f.AddUSDTrackingData(dfk)
	assert.NoError(t, err)

	usdtItem, err := CreateItem(exchName, a, currency.USDT, elite, decimal.Zero)
	assert.NoError(t, err)

	err = f.AddItem(usdtItem)
	assert.NoError(t, err)

	err = f.AddUSDTrackingData(dfk)
	assert.NoError(t, err)
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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

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
			CurrencyPair: currency.NewBTCUSDT(),
		},
	})
	assert.NoError(t, err)

	if !f.items[0].available.IsZero() {
		t.Errorf("received '%v' expected '%v'", f.items[0].available, "0")
	}
}

func TestHasExchangeBeenLiquidated(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	err := f.Liquidate(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

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
			CurrencyPair: currency.NewBTCUSDT(),
		},
	}
	err = f.Liquidate(ev)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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

	err := f.RealisePNL("test", asset.Futures, currency.BTC, decimal.NewFromInt(1))
	require.NoError(t, err, "RealisePNL must not error")
	assert.Equal(t, decimal.NewFromInt(1337), f.items[0].available)

	err = f.RealisePNL("test2", asset.Futures, currency.BTC, decimal.NewFromInt(1))
	assert.ErrorIs(t, err, ErrFundsNotFound)
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

	_, err := CreateCollateral(collat, contract)
	assert.NoError(t, err, "CreateCollateral should not error")

	_, err = CreateCollateral(nil, contract)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	_, err = CreateCollateral(collat, nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)
}

func TestUpdateCollateral(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	err := f.UpdateCollateralForEvent(nil, false)
	assert.ErrorIs(t, err, common.ErrNilEvent)

	ev := &signal.Signal{
		Base: &event.Base{
			Exchange:     exchName,
			AssetType:    asset.Futures,
			CurrencyPair: currency.NewBTCUSD(),
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
	require.NoError(t, err)
	exch.SetDefaults()
	err = em.Add(exch)
	require.NoError(t, err)

	f.exchangeManager = em

	err = f.UpdateCollateralForEvent(ev, false)
	assert.NoError(t, err, "UpdateCollateralForEvent should not error")

	f.items = append(f.items, &Item{
		exchange:     exchName,
		asset:        asset.Futures,
		currency:     currency.USD,
		available:    decimal.NewFromInt(1336),
		isCollateral: true,
	})
	err = f.UpdateCollateralForEvent(ev, false)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)
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
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	item := &Item{}
	err = f.LinkCollateralCurrency(item, currency.EMPTYCODE)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	err = f.LinkCollateralCurrency(item, currency.BTC)
	assert.ErrorIs(t, err, errNotFutures)

	item.asset = asset.Futures
	err = f.LinkCollateralCurrency(item, currency.BTC)
	assert.NoError(t, err)

	if !item.pairedWith.currency.Equal(currency.BTC) {
		t.Errorf("received '%v', expected  '%v'", currency.BTC, item.pairedWith.currency)
	}

	err = f.LinkCollateralCurrency(item, currency.LTC)
	assert.ErrorIs(t, err, ErrAlreadyExists)

	f.items = append(f.items, item.pairedWith)
	item.pairedWith = nil
	err = f.LinkCollateralCurrency(item, currency.BTC)
	assert.NoError(t, err)
}

func TestSetFunding(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	err := f.SetFunding("", 0, nil, false)
	assert.ErrorIs(t, err, gctcommon.ErrExchangeNameNotSet)

	err = f.SetFunding(exchName, 0, nil, false)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	err = f.SetFunding(exchName, asset.Spot, nil, false)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bal := &accounts.Balance{}
	err = f.SetFunding(exchName, asset.Spot, bal, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	bal.Currency = currency.BTC
	bal.Total = 1337
	err = f.SetFunding(exchName, asset.Spot, bal, false)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, engine.ErrNilSubsystem)

	f.exchangeManager = engine.NewExchangeManager()
	err = f.UpdateFundingFromLiveData(false)
	assert.NoError(t, err)

	ff := &binance.Exchange{}
	ff.SetDefaults()
	err = f.exchangeManager.Add(ff)
	require.NoError(t, err)

	err = f.UpdateFundingFromLiveData(false)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)

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
	assert.NoError(t, err)

	err = f.UpdateFundingFromLiveData(false)
	assert.NoError(t, err)
}

func TestUpdateAllCollateral(t *testing.T) {
	t.Parallel()
	f := &FundManager{}
	err := f.UpdateAllCollateral(false, false)
	assert.ErrorIs(t, err, engine.ErrNilSubsystem)

	f.exchangeManager = engine.NewExchangeManager()
	err = f.UpdateAllCollateral(false, false)
	assert.NoError(t, err)

	ff := &binance.Exchange{}
	ff.SetDefaults()
	err = f.exchangeManager.Add(ff)
	require.NoError(t, err)

	err = f.UpdateAllCollateral(false, false)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	f.items = []*Item{
		{
			exchange:     exchName,
			asset:        asset.Spot,
			currency:     currency.BTC,
			isCollateral: true,
		},
	}
	err = f.UpdateAllCollateral(false, false)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	f.items[0].trackingCandles = kline.NewDataFromKline()
	err = f.items[0].trackingCandles.SetStream([]data.Event{
		&fakeEvent{},
	})
	assert.NoError(t, err)

	err = f.UpdateAllCollateral(false, false)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	f.items[0].asset = asset.Futures
	err = f.UpdateAllCollateral(false, false)
	assert.ErrorIs(t, err, gctcommon.ErrNotYetImplemented)

	apiKey := ""
	apiSec := ""
	subAccount := ""
	if apiKey == "" || apiSec == "" {
		// this test requires auth to get coverage
		return
	}
	ff.SetCredentials(apiKey, apiSec, "", subAccount, "", "")
	err = f.UpdateAllCollateral(true, true)
	assert.NoError(t, err)
}

var leet = decimal.NewFromInt(1337)

// fakeEvent implements common.Event without
// caring about the response, or dealing with import cycles
type fakeEvent struct{}

func (f *fakeEvent) GetHighPrice() decimal.Decimal    { return leet }
func (f *fakeEvent) GetLowPrice() decimal.Decimal     { return leet }
func (f *fakeEvent) GetOpenPrice() decimal.Decimal    { return leet }
func (f *fakeEvent) GetVolume() decimal.Decimal       { return leet }
func (f *fakeEvent) GetOffset() int64                 { return 0 }
func (f *fakeEvent) SetOffset(int64)                  {}
func (f *fakeEvent) IsEvent() bool                    { return true }
func (f *fakeEvent) GetTime() time.Time               { return time.Now() }
func (f *fakeEvent) Pair() currency.Pair              { return pair }
func (f *fakeEvent) GetExchange() string              { return exchName }
func (f *fakeEvent) GetInterval() gctkline.Interval   { return gctkline.OneMin }
func (f *fakeEvent) GetAssetType() asset.Item         { return asset.Spot }
func (f *fakeEvent) AppendReason(string)              {}
func (f *fakeEvent) GetClosePrice() decimal.Decimal   { return elite }
func (f *fakeEvent) AppendReasonf(_ string, _ ...any) {}
func (f *fakeEvent) GetBase() *event.Base             { return &event.Base{} }
func (f *fakeEvent) GetUnderlyingPair() currency.Pair { return pair }
func (f *fakeEvent) GetConcatReasons() string         { return "" }
func (f *fakeEvent) GetReasons() []string             { return nil }
