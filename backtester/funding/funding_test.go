package funding

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	elite = decimal.NewFromInt(1337)
	neg   = decimal.NewFromInt(-1)
	one   = decimal.NewFromInt(1)
	exch  = "exch"
	a     = asset.Spot
	base  = currency.DOGE
	quote = currency.XRP
	pair  = currency.NewPair(base, quote)
)

func TestSetupFundingManager(t *testing.T) {
	t.Parallel()
	f := SetupFundingManager(true, false)
	if !f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", true, false)
	}
	if f.disableUSDTracking {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	f = SetupFundingManager(false, true)
	if f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", false, true)
	}
	if !f.disableUSDTracking {
		t.Errorf("expected '%v received '%v'", true, false)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	f := SetupFundingManager(true, false)
	baseItem, err := CreateItem(exch, a, base, decimal.Zero, decimal.Zero, false)
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
	f := SetupFundingManager(true, false)
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
	if !item1.available.Equal(decimal.Zero) {
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

	baseItem, err := CreateItem(exch, a, base, decimal.Zero, decimal.Zero, false)
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
	exists := f.Exists(nil)
	if exists {
		t.Errorf("received '%v' expected '%v'", exists, false)
	}
	conflictingSingleItem, err := CreateItem(exch, a, base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(conflictingSingleItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	exists = f.Exists(conflictingSingleItem)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}
	baseItem, err := CreateItem(exch, a, base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, quote, elite, decimal.Zero, false)
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
	_, err = f.GetFundingForEAP(exch, a, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = f.GetFundingForEAP(exch, a, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	// demonstration that you don't need the original *Items
	// to check for existence, just matching fields
	baseCopy := Item{
		exchange:           baseItem.exchange,
		asset:              baseItem.asset,
		currency:           baseItem.currency,
		initialFunds:       baseItem.initialFunds,
		available:          baseItem.available,
		reserved:           baseItem.reserved,
		transferFee:        baseItem.transferFee,
		pairedWith:         baseItem.pairedWith,
		usdTrackingCandles: baseItem.usdTrackingCandles,
		snapshot:           baseItem.snapshot,
		collateral:         baseItem.collateral,
		collateralCandles:  baseItem.collateralCandles,
	}
	quoteCopy := Item{
		exchange:           quoteItem.exchange,
		asset:              quoteItem.asset,
		currency:           quoteItem.currency,
		initialFunds:       quoteItem.initialFunds,
		available:          quoteItem.available,
		reserved:           quoteItem.reserved,
		transferFee:        quoteItem.transferFee,
		pairedWith:         quoteItem.pairedWith,
		usdTrackingCandles: quoteItem.usdTrackingCandles,
		snapshot:           quoteItem.snapshot,
		collateral:         quoteItem.collateral,
		collateralCandles:  quoteItem.collateralCandles,
	}
	quoteCopy.pairedWith = &baseCopy
	exists = f.Exists(&baseCopy)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}

	currFunds, err := f.GetFundingForEAC(exch, a, base)
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
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
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
	_, err = f.GetFundingForEAP(exch, a, pair)
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

// fakeEvent implements common.EventHandler without
// caring about the response, or dealing with import cycles
type fakeEvent struct{}

func (f *fakeEvent) GetOffset() int64               { return 0 }
func (f *fakeEvent) SetOffset(int64)                {}
func (f *fakeEvent) IsEvent() bool                  { return true }
func (f *fakeEvent) GetTime() time.Time             { return time.Now() }
func (f *fakeEvent) Pair() currency.Pair            { return pair }
func (f *fakeEvent) GetExchange() string            { return exch }
func (f *fakeEvent) GetInterval() gctkline.Interval { return gctkline.OneMin }
func (f *fakeEvent) GetAssetType() asset.Item       { return asset.Spot }
func (f *fakeEvent) GetReason() string              { return "" }
func (f *fakeEvent) AppendReason(string)            {}

func TestGetFundingForEvent(t *testing.T) {
	t.Parallel()
	e := &fakeEvent{}
	f := FundManager{}
	_, err := f.GetFundingForEvent(e)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
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
	_, err := f.GetFundingForEAC(exch, a, base)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(baseItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	fundo, err := f.GetFundingForEAC(exch, a, base)
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
	_, err := f.GetFundingForEAP(exch, a, pair)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
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
	_, err = f.GetFundingForEAP(exch, a, pair)
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

func TestBaseInitialFunds(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	funds := pairItems.BaseInitialFunds()
	if !funds.IsZero() {
		t.Errorf("received '%v' expected '%v'", funds, baseItem.available)
	}
}

func TestQuoteInitialFunds(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	funds := pairItems.QuoteInitialFunds()
	if !funds.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", funds, elite)
	}
}

func TestBaseAvailable(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	funds := pairItems.BaseAvailable()
	if !funds.IsZero() {
		t.Errorf("received '%v' expected '%v'", funds, baseItem.available)
	}
}

func TestQuoteAvailable(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	funds := pairItems.QuoteAvailable()
	if !funds.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", funds, elite)
	}
}

func TestReservePair(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	err = pairItems.Reserve(decimal.Zero, gctorder.Buy)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Buy)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = pairItems.Reserve(decimal.Zero, gctorder.Sell)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Sell)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}
	err = pairItems.Reserve(elite, common.DoNothing)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}
}

func TestReleasePair(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	err = pairItems.Reserve(decimal.Zero, gctorder.Buy)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Buy)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = pairItems.Reserve(decimal.Zero, gctorder.Sell)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Sell)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	err = pairItems.Release(decimal.Zero, decimal.Zero, gctorder.Buy)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = pairItems.Release(elite, decimal.Zero, gctorder.Buy)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = pairItems.Release(elite, decimal.Zero, gctorder.Buy)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	err = pairItems.Release(elite, decimal.Zero, common.DoNothing)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	err = pairItems.Release(elite, decimal.Zero, gctorder.Sell)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}
	err = pairItems.Release(decimal.Zero, decimal.Zero, gctorder.Sell)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
}

func TestIncreaseAvailablePair(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	pairItems.IncreaseAvailable(decimal.Zero, gctorder.Buy)
	if !pairItems.Quote.available.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", elite, pairItems.Quote.available)
	}
	pairItems.IncreaseAvailable(decimal.Zero, gctorder.Sell)
	if !pairItems.Base.available.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", decimal.Zero, pairItems.Base.available)
	}

	pairItems.IncreaseAvailable(elite.Neg(), gctorder.Sell)
	if !pairItems.Quote.available.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", elite, pairItems.Quote.available)
	}
	pairItems.IncreaseAvailable(elite, gctorder.Buy)
	if !pairItems.Base.available.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", elite, pairItems.Base.available)
	}

	pairItems.IncreaseAvailable(elite, common.DoNothing)
	if !pairItems.Base.available.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", elite, pairItems.Base.available)
	}
}

func TestCanPlaceOrderPair(t *testing.T) {
	t.Parallel()
	p := Pair{
		Base:  &Item{},
		Quote: &Item{},
	}
	if p.CanPlaceOrder(common.DoNothing) {
		t.Error("expected false")
	}
	if p.CanPlaceOrder(gctorder.Buy) {
		t.Error("expected false")
	}
	if p.CanPlaceOrder(gctorder.Sell) {
		t.Error("expected false")
	}

	p.Quote.available = decimal.NewFromInt(32)
	if !p.CanPlaceOrder(gctorder.Buy) {
		t.Error("expected true")
	}
	p.Base.available = decimal.NewFromInt(32)
	if !p.CanPlaceOrder(gctorder.Sell) {
		t.Error("expected true")
	}
}

func TestIncreaseAvailable(t *testing.T) {
	t.Parallel()
	i := Item{}
	i.IncreaseAvailable(elite)
	if !i.available.Equal(elite) {
		t.Errorf("expected %v", elite)
	}
	i.IncreaseAvailable(decimal.Zero)
	i.IncreaseAvailable(neg)
	if !i.available.Equal(elite) {
		t.Errorf("expected %v", elite)
	}
}

func TestRelease(t *testing.T) {
	t.Parallel()
	i := Item{}
	err := i.Release(decimal.Zero, decimal.Zero)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = i.Release(elite, decimal.Zero)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}
	i.reserved = elite
	err = i.Release(elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	i.reserved = elite
	err = i.Release(elite, one)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.Release(neg, decimal.Zero)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = i.Release(elite, neg)
	if !errors.Is(err, errNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errNegativeAmountReceived)
	}
}

func TestReserve(t *testing.T) {
	t.Parallel()
	i := Item{}
	err := i.Reserve(decimal.Zero)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
	err = i.Reserve(elite)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	i.reserved = elite
	err = i.Reserve(elite)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	i.available = elite
	err = i.Reserve(elite)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.Reserve(elite)
	if !errors.Is(err, errCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, errCannotAllocate)
	}

	err = i.Reserve(neg)
	if !errors.Is(err, errZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, errZeroAmountReceived)
	}
}

func TestMatchesItemCurrency(t *testing.T) {
	t.Parallel()
	i := Item{}
	if i.MatchesItemCurrency(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if baseItem.MatchesItemCurrency(quoteItem) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	if !baseItem.MatchesItemCurrency(baseItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
}

func TestMatchesExchange(t *testing.T) {
	t.Parallel()
	i := Item{}
	if i.MatchesExchange(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	baseItem, err := CreateItem(exch, a, pair.Base, decimal.Zero, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !baseItem.MatchesExchange(quoteItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
	if !baseItem.MatchesExchange(baseItem) {
		t.Errorf("received '%v' expected '%v'", false, true)
	}
}

func TestGenerateReport(t *testing.T) {
	t.Parallel()
	f := FundManager{}
	report := f.GenerateReport()
	if report == nil {
		t.Fatal("shouldn't be nil")
	}
	if len(report.Items) > 0 {
		t.Error("expected 0")
	}
	item := &Item{
		exchange:     exch,
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
		exchange:     exch,
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
			Exchange: exch,
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
	f.items[0].usdTrackingCandles = dfk
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

func TestMatchesCurrency(t *testing.T) {
	t.Parallel()
	i := Item{
		currency: currency.BTC,
	}
	if i.MatchesCurrency(currency.USDT) {
		t.Error("expected false")
	}
	if !i.MatchesCurrency(currency.BTC) {
		t.Error("expected true")
	}
	if i.MatchesCurrency(currency.Code{}) {
		t.Error("expected false")
	}
	if i.MatchesCurrency(currency.NewCode("")) {
		t.Error("expected false")
	}
}

func TestCreateSnapshot(t *testing.T) {
	f := FundManager{}
	f.CreateSnapshot(time.Time{})
	f.items = append(f.items, &Item{
		exchange:           "",
		asset:              "",
		currency:           currency.Code{},
		initialFunds:       decimal.Decimal{},
		available:          decimal.Decimal{},
		reserved:           decimal.Decimal{},
		transferFee:        decimal.Decimal{},
		pairedWith:         nil,
		usdTrackingCandles: nil,
		snapshot:           nil,
	})
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
		exchange:           "test",
		asset:              asset.Spot,
		currency:           currency.BTC,
		initialFunds:       decimal.NewFromInt(1337),
		available:          decimal.NewFromInt(1337),
		reserved:           decimal.NewFromInt(1337),
		transferFee:        decimal.NewFromInt(1337),
		usdTrackingCandles: dfk,
	})
	f.CreateSnapshot(dfk.Item.Candles[0].Time)
}

func TestAddUSDTrackingData(t *testing.T) {
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
	quoteItem, err := CreateItem(exch, a, pair.Quote, elite, decimal.Zero, false)
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
			Exchange: exch,
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

	usdtItem, err := CreateItem(exch, a, currency.USDT, elite, decimal.Zero, false)
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
	f := FundManager{}
	if f.USDTrackingDisabled() {
		t.Error("received true, expected false")
	}
	f.disableUSDTracking = true
	if !f.USDTrackingDisabled() {
		t.Error("received false, expected true")
	}
}
