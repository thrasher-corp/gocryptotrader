package funding

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	ErrNegativeAmountReceived = errors.New("received negative decimal")
	ErrAlreadyExists          = errors.New("already exists")
)

func Setup(usingExchangeLevelFunding bool) *AllFunds {
	return &AllFunds{usingExchangeLevelFunding: usingExchangeLevelFunding}
}

func (a *AllFunds) Reset() {
	*a = AllFunds{}
}

// Transfer allows transferring funds from one pretend exchange to another
func (a *AllFunds) Transfer(amount decimal.Decimal, sender, receiver *Item) error {
	if !sender.available.LessThanOrEqual(amount.Add(sender.TransferFee)) {
		return errors.New("not enough funds")
	}
	if sender.Item != receiver.Item {
		return errors.New("must be like for like")
	}
	if sender.Item == receiver.Item &&
		sender.Exchange == receiver.Exchange &&
		sender.Asset == receiver.Asset {
		return errors.New("why are you sending to the same thing")
	}
	err := sender.Reserve(amount.Add(sender.TransferFee))
	if err != nil {
		return err
	}
	receiver.IncreaseAvailable(amount)
	err = sender.Release(amount.Add(sender.TransferFee), decimal.Zero)
	if err != nil {
		return err
	}

	return nil
}

func (a *AllFunds) AddItem(exch string, ass asset.Item, ci currency.Code, initialFunds decimal.Decimal) error {
	if a.Exists(exch, ass, ci) {
		return fmt.Errorf("cannot add item %v %v %v %w", exch, ass, ci, ErrAlreadyExists)
	}
	item := &Item{
		Exchange:     exch,
		Asset:        ass,
		Item:         ci,
		initialFunds: initialFunds,
		available:    initialFunds,
	}
	a.items = append(a.items, item)
	return nil
}

func (a *AllFunds) Exists(exch string, ass asset.Item, ci currency.Code) bool {
	for i := range a.items {
		if a.items[i].Item == ci &&
			a.items[i].Exchange == exch &&
			a.items[i].Asset == ass {
			return true
		}
	}
	return false
}

func (a *AllFunds) AddPair(exch string, ass asset.Item, cp currency.Pair, initialFunds decimal.Decimal) error {
	base := &Item{
		Exchange: exch,
		Asset:    ass,
		Item:     cp.Base,
	}
	quote := &Item{
		Exchange:     exch,
		Asset:        ass,
		Item:         cp.Quote,
		initialFunds: initialFunds,
		available:    initialFunds,
		PairedWith:   base,
	}
	base.PairedWith = quote
	a.items = append(a.items, base, quote)
	return nil
}

// IsUsingExchangeLevelFunding returns if using usingExchangeLevelFunding
func (a *AllFunds) IsUsingExchangeLevelFunding() bool {
	return a.usingExchangeLevelFunding
}

// GetFundingForEvent This will construct a funding based on a backtesting event
func (a *AllFunds) GetFundingForEvent(e common.EventHandler) (*Pair, error) {
	return a.GetFundingForEAP(e.GetExchange(), e.GetAssetType(), e.Pair())
}

// GetFundingForEAC This will construct a funding based on the exchange, asset, currency code
func (a *AllFunds) GetFundingForEAC(exch string, ass asset.Item, c currency.Code) (*Item, error) {
	for i := range a.items {
		if a.items[i].Item == c &&
			a.items[i].Exchange == exch &&
			a.items[i].Asset == ass {
			return a.items[i], nil
		}
	}
	return nil, ErrFundsNotFound
}

// GetFundingForEAP This will construct a funding based on the exchange, asset, currency pair
func (a *AllFunds) GetFundingForEAP(exch string, ass asset.Item, p currency.Pair) (*Pair, error) {
	var resp Pair
	for i := range a.items {
		if a.items[i].Item == p.Quote &&
			a.items[i].Exchange == exch &&
			a.items[i].Asset == ass &&
			(a.usingExchangeLevelFunding || (a.items[i].PairedWith != nil && a.items[i].PairedWith.Item == p.Base)) {
			resp.Quote = a.items[i]
			continue
		}
		if a.items[i].Item == p.Base &&
			a.items[i].Exchange == exch &&
			a.items[i].Asset == ass &&
			(a.usingExchangeLevelFunding || (a.items[i].PairedWith != nil && a.items[i].PairedWith.Item == p.Quote)) {
			resp.Base = a.items[i]
		}
	}
	if resp.Base == nil || resp.Quote == nil {
		return nil, ErrFundsNotFound
	}
	return &resp, nil
}

func (p *Pair) BaseInitialFunds() decimal.Decimal {
	return p.Base.initialFunds
}

func (p *Pair) QuoteInitialFunds() decimal.Decimal {
	return p.Quote.initialFunds

}

func (p *Pair) BaseAvailable() decimal.Decimal {
	return p.Base.available
}

func (p *Pair) QuoteAvailable() decimal.Decimal {
	return p.Quote.available
}

func (p *Pair) Reserve(amount decimal.Decimal, side order.Side) error {
	switch side {
	case order.Buy:
		return p.Quote.Reserve(amount)
	case order.Sell:
		return p.Base.Reserve(amount)
	default:
		return fmt.Errorf("%w for %v %v %v. Unknown side %v",
			ErrCannotAllocate,
			p.Base.Exchange,
			p.Base.Asset,
			p.Base.Item,
			side)
	}
}

func (p *Pair) Release(amount, diff decimal.Decimal, side order.Side) error {
	switch side {
	case order.Buy:
		return p.Quote.Release(amount, diff)
	case order.Sell:
		return p.Base.Release(amount, diff)
	default:
		return fmt.Errorf("%w for %v %v %v. Unknown side %v",
			ErrCannotAllocate,
			p.Base.Exchange,
			p.Base.Asset,
			p.Base.Item,
			side)
	}
}

func (p *Pair) Increase(amount decimal.Decimal, side order.Side) {
	switch side {
	case order.Buy:
		p.Base.IncreaseAvailable(amount)
	case order.Sell:
		p.Quote.IncreaseAvailable(amount)
	}
}

func (i *Item) Reserve(amount decimal.Decimal) error {
	if amount.IsNegative() {
		return fmt.Errorf("%w amount", ErrNegativeAmountReceived)
	}
	if amount.GreaterThan(i.available) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Available: %v",
			ErrCannotAllocate,
			i.Exchange,
			i.Asset,
			i.Item,
			amount,
			i.available)
	}
	i.available = i.available.Sub(amount)
	i.Reserved = i.Reserved.Add(amount)
	return nil
}

// Release lowers the reserved amount and appends any differences
// as a result of any exchange level modifications when ordering
func (i *Item) Release(amount, diff decimal.Decimal) error {
	if amount.IsNegative() {
		return fmt.Errorf("%w amount", ErrNegativeAmountReceived)
	}
	if diff.IsNegative() {
		return fmt.Errorf("%w diff", ErrNegativeAmountReceived)
	}
	if amount.GreaterThan(i.Reserved) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Reserved: %v",
			ErrCannotAllocate,
			i.Exchange,
			i.Asset,
			i.Item,
			amount,
			i.Reserved)
	}
	i.Reserved = i.Reserved.Sub(amount)
	i.available = i.available.Add(diff)
	return nil
}

func (i *Item) IncreaseAvailable(amount decimal.Decimal) {
	if amount.IsNegative() || amount.IsZero() {
		return
	}
	i.available = i.available.Add(amount)
}

func (p *Pair) CanPlaceOrder(side order.Side) bool {
	switch side {
	case order.Buy:
		return p.Quote.available.GreaterThan(decimal.Zero)
	case order.Sell:
		return p.Base.available.GreaterThan(decimal.Zero)
	}
	return false
}
