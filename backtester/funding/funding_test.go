package funding

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	elite = decimal.NewFromInt(1337)
	neg   = decimal.NewFromFloat(-1)
	one   = decimal.NewFromFloat(1)
	exch  = "exch"
	ass   = asset.Spot
	base  = currency.DOGE
	quote = currency.XRP
	pair  = currency.NewPair(base, quote)
)

func TestSetupFundingManager(t *testing.T) {
	t.Parallel()
	f := SetupFundingManager(true)
	if !f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", true, false)
	}
	f = SetupFundingManager(false)
	if f.usingExchangeLevelFunding {
		t.Errorf("expected '%v received '%v'", false, true)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	f := SetupFundingManager(true)
	baseItem, err := CreateItem(exch, ass, base, decimal.Zero, decimal.Zero)
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
	f := SetupFundingManager(true)
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
	err := f.Transfer(decimal.Zero, nil, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}
	err = f.Transfer(decimal.Zero, &Item{}, nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expected '%v'", err, common.ErrNilArguments)
	}
	err = f.Transfer(decimal.Zero, &Item{}, &Item{})
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = f.Transfer(elite, &Item{}, &Item{})
	if !errors.Is(err, ErrNotEnoughFunds) {
		t.Errorf("received '%v' expected '%v'", err, ErrNotEnoughFunds)
	}
	item1 := &Item{exchange: "hello", asset: ass, currency: base, available: elite}
	err = f.Transfer(elite, item1, item1)
	if !errors.Is(err, errCannotTransferToSameFunds) {
		t.Errorf("received '%v' expected '%v'", err, errCannotTransferToSameFunds)
	}

	item2 := &Item{exchange: "hello", asset: ass, currency: quote}
	err = f.Transfer(elite, item1, item2)
	if !errors.Is(err, errTransferMustBeSameCurrency) {
		t.Errorf("received '%v' expected '%v'", err, errTransferMustBeSameCurrency)
	}

	item2.exchange = "moto"
	item2.currency = base
	err = f.Transfer(elite, item1, item2)
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
	err = f.Transfer(elite.Sub(item2.transferFee), item2, item1)
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

	baseItem, err := CreateItem(exch, ass, base, decimal.Zero, decimal.Zero)
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
	conflictingSingleItem, err := CreateItem(exch, ass, base, decimal.Zero, decimal.Zero)
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
	baseItem, err := CreateItem(exch, ass, base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, quote, elite, decimal.Zero)
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
	pairItems, err := f.GetFundingForEAP(exch, ass, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	exists = f.Exists(pairItems.Base)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}
	exists = f.Exists(pairItems.Quote)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}

	funds, err := f.GetFundingForEAP(exch, ass, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	// demonstration that you don't need the original *Items
	// to check for existence, just matching fields
	baseCopy := *funds.Base
	quoteCopy := *funds.Quote
	quoteCopy.pairedWith = &baseCopy
	exists = f.Exists(&baseCopy)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}

	currFunds, err := f.GetFundingForEAC(exch, ass, base)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	resp, err := f.GetFundingForEAP(exch, ass, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if resp.Base.exchange != exch ||
		resp.Base.asset != ass ||
		resp.Base.currency != pair.Base {
		t.Error("woah nelly")
	}
	if resp.Quote.exchange != exch ||
		resp.Quote.asset != ass ||
		resp.Quote.currency != pair.Quote {
		t.Error("woah nelly")
	}
	if resp.Quote.pairedWith != resp.Base {
		t.Errorf("received '%v' expected '%v'", resp.Base, resp.Quote.pairedWith)
	}
	if resp.Base.pairedWith != resp.Quote {
		t.Errorf("received '%v' expected '%v'", resp.Quote, resp.Base.pairedWith)
	}
	if !resp.Base.initialFunds.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", resp.Base.initialFunds, decimal.Zero)
	}
	if !resp.Quote.initialFunds.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", resp.Quote.initialFunds, elite)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	_, err := f.GetFundingForEAC(exch, ass, base)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = f.AddItem(baseItem)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	fundo, err := f.GetFundingForEAC(exch, ass, base)
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
	_, err := f.GetFundingForEAP(exch, ass, pair)
	if !errors.Is(err, ErrFundsNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrFundsNotFound)
	}
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	_, err = f.GetFundingForEAP(exch, ass, pair)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	err = pairItems.Reserve(decimal.Zero, gctorder.Buy)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Buy)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = pairItems.Reserve(decimal.Zero, gctorder.Sell)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Sell)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}
	err = pairItems.Reserve(elite, common.DoNothing)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}
}

func TestReleasePair(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	baseItem.pairedWith = quoteItem
	quoteItem.pairedWith = baseItem
	pairItems := Pair{Base: baseItem, Quote: quoteItem}
	err = pairItems.Reserve(decimal.Zero, gctorder.Buy)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Buy)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = pairItems.Reserve(decimal.Zero, gctorder.Sell)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = pairItems.Reserve(elite, gctorder.Sell)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	err = pairItems.Release(decimal.Zero, decimal.Zero, gctorder.Buy)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = pairItems.Release(elite, decimal.Zero, gctorder.Buy)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = pairItems.Release(elite, decimal.Zero, gctorder.Buy)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	err = pairItems.Release(elite, decimal.Zero, common.DoNothing)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	err = pairItems.Release(elite, decimal.Zero, gctorder.Sell)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}
	err = pairItems.Release(decimal.Zero, decimal.Zero, gctorder.Sell)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
}

func TestIncreaseAvailablePair(t *testing.T) {
	t.Parallel()
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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

	p.Quote.available = decimal.NewFromFloat(32)
	if !p.CanPlaceOrder(gctorder.Buy) {
		t.Error("expected true")
	}
	p.Base.available = decimal.NewFromFloat(32)
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
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = i.Release(elite, decimal.Zero)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
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
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = i.Release(elite, neg)
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
}

func TestReserve(t *testing.T) {
	t.Parallel()
	i := Item{}
	err := i.Reserve(decimal.Zero)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
	err = i.Reserve(elite)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	i.reserved = elite
	err = i.Reserve(elite)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	i.available = elite
	err = i.Reserve(elite)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.Reserve(elite)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	err = i.Reserve(neg)
	if !errors.Is(err, ErrZeroAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrZeroAmountReceived)
	}
}

func TestMatchesCurrency(t *testing.T) {
	t.Parallel()
	i := Item{}
	if i.MatchesItemCurrency(nil) {
		t.Errorf("received '%v' expected '%v'", true, false)
	}
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
	baseItem, err := CreateItem(exch, ass, pair.Base, decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	quoteItem, err := CreateItem(exch, ass, pair.Quote, elite, decimal.Zero)
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
