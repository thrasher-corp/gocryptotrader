package funding

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	elite = decimal.NewFromFloat(1337)
	neg   = decimal.NewFromFloat(-1)
	one   = decimal.NewFromFloat(1)
	exch  = "exch"
	ass   = asset.Spot
	curr  = currency.DOGE
	curr2 = currency.XRP
	pair  = currency.NewPair(curr, curr2)
)

func TestTransfer(t *testing.T) {
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
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
	err = f.Transfer(elite, &Item{}, &Item{})
	if !errors.Is(err, ErrNotEnoughFunds) {
		t.Errorf("received '%v' expected '%v'", err, ErrNotEnoughFunds)
	}
	item1 := &Item{Exchange: "hello", Asset: ass, Item: curr, available: elite}
	err = f.Transfer(elite, item1, item1)
	if !errors.Is(err, errCannotTransferToSameFunds) {
		t.Errorf("received '%v' expected '%v'", err, errCannotTransferToSameFunds)
	}

	item2 := &Item{Exchange: "hello", Asset: ass, Item: curr2}
	err = f.Transfer(elite, item1, item2)
	if !errors.Is(err, errTransferMustBeSameCurrency) {
		t.Errorf("received '%v' expected '%v'", err, errTransferMustBeSameCurrency)
	}

	item2.Exchange = "moto"
	item2.Item = curr
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

	item2.TransferFee = one
	err = f.Transfer(elite.Sub(item2.TransferFee), item2, item1)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !item1.available.Equal(elite.Sub(item2.TransferFee)) {
		t.Errorf("received '%v' expected '%v'", item2.available, elite.Sub(item2.TransferFee))
	}
}

func TestAddItem(t *testing.T) {
	f := FundManager{}
	err := f.AddItem(exch, ass, curr, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = f.AddItem(exch, ass, curr, elite, decimal.Zero)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("received '%v' expected '%v'", err, ErrAlreadyExists)
	}
	err = f.AddItem(exch, ass, currency.DOGE, neg, decimal.Zero)
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
	err = f.AddItem(exch, ass, currency.DOGE, elite, neg)
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
}

func TestExists(t *testing.T) {
	f := FundManager{}
	exists := f.Exists(exch, ass, curr, nil)
	if exists {
		t.Errorf("received '%v' expected '%v'", exists, false)
	}
	err := f.AddItem(exch, ass, curr, elite, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	exists = f.Exists(exch, ass, curr, nil)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}
	err = f.AddPair(exch, ass, pair, elite)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	funds, err := f.GetFundingForEAP(exch, ass, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	exists = f.Exists(exch, ass, funds.Base.Item, funds.Quote)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}
	exists = f.Exists(exch, ass, funds.Quote.Item, funds.Base)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}
	// demonstration that you don't need the original *Items
	// to check for existence, just matching fields
	baseCopy := *funds.Base
	quoteCopy := *funds.Quote
	quoteCopy.PairedWith = &baseCopy
	exists = f.Exists(exch, ass, quoteCopy.Item, &baseCopy)
	if !exists {
		t.Errorf("received '%v' expected '%v'", exists, true)
	}

	currFunds, err := f.GetFundingForEAC(exch, ass, curr)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if currFunds.PairedWith != nil {
		t.Errorf("received '%v' expected '%v'", nil, currFunds.PairedWith)
	}
}

func TestAddPair(t *testing.T) {
	f := FundManager{}
	err := f.AddPair(exch, ass, pair, elite)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	resp, err := f.GetFundingForEAP(exch, ass, pair)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if resp.Base.Exchange != exch ||
		resp.Base.Asset != ass ||
		resp.Base.Item != pair.Base {
		t.Error("woah nelly")
	}
	if resp.Quote.Exchange != exch ||
		resp.Quote.Asset != ass ||
		resp.Quote.Item != pair.Quote {
		t.Error("woah nelly")
	}
	if resp.Quote.PairedWith != resp.Base {
		t.Errorf("received '%v' expected '%v'", resp.Base, resp.Quote.PairedWith)
	}
	if resp.Base.PairedWith != resp.Quote {
		t.Errorf("received '%v' expected '%v'", resp.Quote, resp.Base.PairedWith)
	}
	if !resp.Base.initialFunds.Equal(decimal.Zero) {
		t.Errorf("received '%v' expected '%v'", resp.Base.initialFunds, decimal.Zero)

	}
	if !resp.Quote.initialFunds.Equal(elite) {
		t.Errorf("received '%v' expected '%v'", resp.Quote.initialFunds, elite)
	}

	err = f.AddPair(exch, ass, pair, elite)
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("received '%v' expected '%v'", err, ErrAlreadyExists)
	}
}

func TestCanPlaceOrder(t *testing.T) {
	p := Pair{
		Base:  &Item{},
		Quote: &Item{},
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
	i := Item{}
	i.IncreaseAvailable(decimal.NewFromFloat(3))
	if !i.available.Equal(decimal.NewFromFloat(3)) {
		t.Error("expected 3")
	}
	i.IncreaseAvailable(decimal.NewFromFloat(0))
	i.IncreaseAvailable(decimal.NewFromFloat(-1))
	if !i.available.Equal(decimal.NewFromFloat(3)) {
		t.Error("expected 3")
	}
}

func TestRelease(t *testing.T) {
	i := Item{}
	err := i.Release(decimal.Zero, decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = i.Release(decimal.NewFromFloat(1337), decimal.Zero)
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}
	i.Reserved = decimal.NewFromFloat(1337)
	err = i.Release(decimal.NewFromFloat(1337), decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.Release(decimal.NewFromFloat(-1), decimal.Zero)
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
	err = i.Release(decimal.NewFromFloat(1337), decimal.NewFromFloat(-1))
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
}

func TestReserve(t *testing.T) {
	i := Item{}
	err := i.Reserve(decimal.Zero)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = i.Reserve(decimal.NewFromFloat(1337))
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	i.Reserved = decimal.NewFromFloat(1337)
	err = i.Reserve(decimal.NewFromFloat(1337))
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	i.available = decimal.NewFromFloat(1337)
	err = i.Reserve(decimal.NewFromFloat(1337))
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	err = i.Reserve(decimal.NewFromFloat(1337))
	if !errors.Is(err, ErrCannotAllocate) {
		t.Errorf("received '%v' expected '%v'", err, ErrCannotAllocate)
	}

	err = i.Reserve(decimal.NewFromFloat(-1))
	if !errors.Is(err, ErrNegativeAmountReceived) {
		t.Errorf("received '%v' expected '%v'", err, ErrNegativeAmountReceived)
	}
}
