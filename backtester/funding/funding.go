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
	ErrNegativeAmountReceived     = errors.New("received negative decimal")
	ErrAlreadyExists              = errors.New("already exists")
	ErrNotEnoughFunds             = errors.New("not enough funds")
	errCannotTransferToSameFunds  = errors.New("cannot send funds to self")
	errTransferMustBeSameCurrency = errors.New("cannot transfer to different currency")
)

// SetupFundingManager creates the funding holder. It carries knowledge about levels of funding
// across all execution handlers and enables fund transfers
func SetupFundingManager(usingExchangeLevelFunding bool) *FundManager {
	return &FundManager{usingExchangeLevelFunding: usingExchangeLevelFunding}
}

// Reset clears all settings
func (f *FundManager) Reset() {
	*f = FundManager{}
}

// Transfer allows transferring funds from one pretend exchange to another
func (f *FundManager) Transfer(amount decimal.Decimal, sender, receiver *Item) error {
	if sender == nil || receiver == nil {
		return common.ErrNilArguments
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrNegativeAmountReceived
	}
	if sender.available.LessThan(amount.Add(sender.TransferFee)) {
		return fmt.Errorf("%w for %v", ErrNotEnoughFunds, sender.Item)
	}
	if sender.Item != receiver.Item {
		return errTransferMustBeSameCurrency
	}
	if sender.Item == receiver.Item &&
		sender.Exchange == receiver.Exchange &&
		sender.Asset == receiver.Asset {
		return fmt.Errorf("%v %v %v %w", sender.Exchange, sender.Asset, sender.Item, errCannotTransferToSameFunds)
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

// AddItem appends a new funding item. Will reject if exists by exchange asset currency
func (f *FundManager) AddItem(exch string, ass asset.Item, ci currency.Code, initialFunds, transferFee decimal.Decimal) error {
	if f.Exists(exch, ass, ci, nil) {
		return fmt.Errorf("cannot add item %v %v %v %w", exch, ass, ci, ErrAlreadyExists)
	}
	if initialFunds.IsNegative() {
		return fmt.Errorf("%v %v %v %w initial funds: %v", exch, ass, ci, ErrNegativeAmountReceived, initialFunds)
	}
	if transferFee.IsNegative() {
		return fmt.Errorf("%v %v %v %w transfer fee: %v", exch, ass, ci, ErrNegativeAmountReceived, transferFee)

	}
	item := &Item{
		Exchange:     exch,
		Asset:        ass,
		Item:         ci,
		initialFunds: initialFunds,
		available:    initialFunds,
		TransferFee:  transferFee,
	}
	f.items = append(f.items, item)
	return nil
}

// Exists verifies whether there is a funding item that exists
// with the same exchange, asset and currency
func (f *FundManager) Exists(exch string, ass asset.Item, ci currency.Code, pairedWith *Item) bool {
	for i := range f.items {
		if f.items[i].Item == ci &&
			f.items[i].Exchange == exch &&
			f.items[i].Asset == ass &&
			(pairedWith == nil && f.items[i].PairedWith == nil) ||
			(pairedWith != nil && f.items[i].PairedWith != nil && *f.items[i].PairedWith == *pairedWith) {
			return true
		}
	}
	return false
}

// AddPair adds two funding items and associates them with one another
// the association allows for the same currency to be used multiple times when
// usingExchangeLevelFunding is false. eg BTC-USDT and LTC-USDT do not share the same
// USDT level funding
func (f *FundManager) AddPair(exch string, ass asset.Item, cp currency.Pair, initialFunds decimal.Decimal) error {
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
	if f.Exists(base.Exchange, base.Asset, base.Item, quote) {
		return fmt.Errorf("cannot add item %v %v %v %w", exch, ass, cp.Base, ErrAlreadyExists)
	}
	if f.Exists(quote.Exchange, quote.Asset, quote.Item, base) {
		return fmt.Errorf("cannot add item %v %v %v %w", exch, ass, cp.Quote, ErrAlreadyExists)
	}
	f.items = append(f.items, base, quote)
	return nil
}

// IsUsingExchangeLevelFunding returns if using usingExchangeLevelFunding
func (f *FundManager) IsUsingExchangeLevelFunding() bool {
	return f.usingExchangeLevelFunding
}

// GetFundingForEvent This will construct a funding based on a backtesting event
func (f *FundManager) GetFundingForEvent(e common.EventHandler) (*Pair, error) {
	return f.GetFundingForEAP(e.GetExchange(), e.GetAssetType(), e.Pair())
}

// GetFundingForEAC This will construct a funding based on the exchange, asset, currency code
func (f *FundManager) GetFundingForEAC(exch string, ass asset.Item, c currency.Code) (*Item, error) {
	for i := range f.items {
		if f.items[i].Item == c &&
			f.items[i].Exchange == exch &&
			f.items[i].Asset == ass {
			return f.items[i], nil
		}
	}
	return nil, ErrFundsNotFound
}

// GetFundingForEAP This will construct a funding based on the exchange, asset, currency pair
func (f *FundManager) GetFundingForEAP(exch string, ass asset.Item, p currency.Pair) (*Pair, error) {
	var resp Pair
	for i := range f.items {
		if f.items[i].Item == p.Quote &&
			f.items[i].Exchange == exch &&
			f.items[i].Asset == ass &&
			(f.usingExchangeLevelFunding || (f.items[i].PairedWith != nil && f.items[i].PairedWith.Item == p.Base)) {
			resp.Quote = f.items[i]
			continue
		}
		if f.items[i].Item == p.Base &&
			f.items[i].Exchange == exch &&
			f.items[i].Asset == ass &&
			(f.usingExchangeLevelFunding || (f.items[i].PairedWith != nil && f.items[i].PairedWith.Item == p.Quote)) {
			resp.Base = f.items[i]
		}
	}
	if resp.Base == nil || resp.Quote == nil {
		return nil, ErrFundsNotFound
	}
	return &resp, nil
}

// BaseInitialFunds returns the initial funds
// from the base in a currency pair
func (p *Pair) BaseInitialFunds() decimal.Decimal {
	return p.Base.initialFunds
}

// QuoteInitialFunds returns the initial funds
// from the quote in a currency pair
func (p *Pair) QuoteInitialFunds() decimal.Decimal {
	return p.Quote.initialFunds

}

// BaseAvailable returns the available funds
// from the base in a currency pair
func (p *Pair) BaseAvailable() decimal.Decimal {
	return p.Base.available
}

// QuoteAvailable returns the available funds
// from the quote in a currency pair
func (p *Pair) QuoteAvailable() decimal.Decimal {
	return p.Quote.available
}

// Reserve allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
// changes which currency to affect based on the order side
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

// Release reduces the amount of funding reserved and adds any difference
// back to the available amount
// changes which currency to affect based on the order side
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

// IncreaseAvailable adds funding to the available amount
// changes which currency to affect based on the order side
func (p *Pair) IncreaseAvailable(amount decimal.Decimal, side order.Side) {
	switch side {
	case order.Buy:
		p.Base.IncreaseAvailable(amount)
	case order.Sell:
		p.Quote.IncreaseAvailable(amount)
	}
}

// CanPlaceOrder does a > 0 check to see if there are any funds
// to place an order with
// changes which currency to affect based on the order side
func (p *Pair) CanPlaceOrder(side order.Side) bool {
	switch side {
	case order.Buy:
		return p.Quote.CanPlaceOrder()
	case order.Sell:
		return p.Base.CanPlaceOrder()
	}
	return false
}

// Reserve allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
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

// Release reduces the amount of funding reserved and adds any difference
// back to the available amount
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

// IncreaseAvailable adds funding to the available amount
func (i *Item) IncreaseAvailable(amount decimal.Decimal) {
	if amount.IsNegative() || amount.IsZero() {
		return
	}
	i.available = i.available.Add(amount)
}

func (i *Item) CanPlaceOrder() bool {
	return i.available.GreaterThan(decimal.Zero)
}
