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
	if sender.available.LessThan(amount.Add(sender.transferFee)) {
		return fmt.Errorf("%w for %v", ErrNotEnoughFunds, sender.currency)
	}
	if sender.currency != receiver.currency {
		return errTransferMustBeSameCurrency
	}
	if sender.currency == receiver.currency &&
		sender.exchange == receiver.exchange &&
		sender.asset == receiver.asset {
		return fmt.Errorf("%v %v %v %w", sender.exchange, sender.asset, sender.currency, errCannotTransferToSameFunds)
	}
	err := sender.Reserve(amount.Add(sender.transferFee))
	if err != nil {
		return err
	}
	receiver.IncreaseAvailable(amount)
	err = sender.Release(amount.Add(sender.transferFee), decimal.Zero)
	if err != nil {
		return err
	}

	return nil
}

func (f *FundManager) SetupItem(exch string, ass asset.Item, ci currency.Code, initialFunds, transferFee decimal.Decimal) (*Item, error) {
	if initialFunds.IsNegative() {
		return nil, fmt.Errorf("%v %v %v %w initial funds: %v", exch, ass, ci, ErrNegativeAmountReceived, initialFunds)
	}
	if transferFee.IsNegative() {
		return nil, fmt.Errorf("%v %v %v %w transfer fee: %v", exch, ass, ci, ErrNegativeAmountReceived, transferFee)
	}

	return &Item{
		exchange:     exch,
		asset:        ass,
		currency:     ci,
		initialFunds: initialFunds,
		available:    initialFunds,
		transferFee:  transferFee,
	}, nil
}

// AddItem appends a new funding item. Will reject if exists by exchange asset currency
func (f *FundManager) AddItem(item *Item) error {
	if f.Exists(item) {
		return fmt.Errorf("cannot add item %v %v %v %w", item.exchange, item.asset, item.currency, ErrAlreadyExists)
	}
	f.items = append(f.items, item)
	return nil
}

// Exists verifies whether there is a funding item that exists
// with the same exchange, asset and currency
func (f *FundManager) Exists(item *Item) bool {
	for i := range f.items {
		if f.items[i].Equal(item) {
			return true
		}
	}
	return false
}

// AddPair adds two funding items and associates them with one another
// the association allows for the same currency to be used multiple times when
// usingExchangeLevelFunding is false. eg BTC-USDT and LTC-USDT do not share the same
// USDT level funding
func (f *FundManager) AddPair(base, quote *Item) error {
	if base == nil {
		return fmt.Errorf("base %w", common.ErrNilArguments)
	}
	if quote == nil {
		return fmt.Errorf("quote %w", common.ErrNilArguments)
	}
	// copy to prevent the off chance of sending in the same base OR quote
	// to create a new pair with a new base OR quote
	bcpy := *base
	qcpy := *quote
	bcpy.pairedWith = &qcpy
	qcpy.pairedWith = &bcpy
	if f.Exists(&bcpy) {
		return fmt.Errorf("cannot add item %v %v %v %w", bcpy.exchange, bcpy.asset, bcpy.currency, ErrAlreadyExists)
	}
	if f.Exists(&qcpy) {
		return fmt.Errorf("cannot add item %v %v %v %w", qcpy.exchange, qcpy.asset, qcpy.currency, ErrAlreadyExists)
	}
	f.items = append(f.items, &bcpy, &qcpy)
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
		if f.items[i].BasicEqual(exch, ass, c, currency.Code{}) {
			return f.items[i], nil
		}
	}
	return nil, ErrFundsNotFound
}

// GetFundingForEAP This will construct a funding based on the exchange, asset, currency pair
func (f *FundManager) GetFundingForEAP(exch string, ass asset.Item, p currency.Pair) (*Pair, error) {
	var resp Pair
	for i := range f.items {
		if f.items[i].BasicEqual(exch, ass, p.Base, p.Quote) {
			resp.Base = f.items[i]
			continue
		}
		if f.items[i].BasicEqual(exch, ass, p.Quote, p.Base) {
			resp.Quote = f.items[i]
		}
	}
	if resp.Base == nil {
		return nil, fmt.Errorf("base %w", ErrFundsNotFound)
	}
	if resp.Quote == nil {
		return nil, fmt.Errorf("quote %w", ErrFundsNotFound)
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
			p.Base.exchange,
			p.Base.asset,
			p.Base.currency,
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
			p.Base.exchange,
			p.Base.asset,
			p.Base.currency,
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
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%w amount", ErrNegativeAmountReceived)
	}
	if amount.GreaterThan(i.available) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Available: %v",
			ErrCannotAllocate,
			i.exchange,
			i.asset,
			i.currency,
			amount,
			i.available)
	}
	i.available = i.available.Sub(amount)
	i.reserved = i.reserved.Add(amount)
	return nil
}

// Release reduces the amount of funding reserved and adds any difference
// back to the available amount
func (i *Item) Release(amount, diff decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("%w amount", ErrNegativeAmountReceived)
	}
	if diff.IsNegative() {
		return fmt.Errorf("%w diff", ErrNegativeAmountReceived)
	}
	if amount.GreaterThan(i.reserved) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Reserved: %v",
			ErrCannotAllocate,
			i.exchange,
			i.asset,
			i.currency,
			amount,
			i.reserved)
	}
	i.reserved = i.reserved.Sub(amount)
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

func (i *Item) Equal(item *Item) bool {
	if i == nil && item == nil {
		return true
	}
	if item == nil || i == nil {
		return false
	}
	if i.currency == item.currency &&
		i.asset == item.asset &&
		i.exchange == item.exchange {
		if i.pairedWith == nil && item.pairedWith == nil {
			return true
		}
		if i.pairedWith == nil || item.pairedWith == nil {
			return false
		}
		if i.pairedWith.currency == item.pairedWith.currency &&
			i.pairedWith.asset == item.pairedWith.asset &&
			i.pairedWith.exchange == item.pairedWith.exchange {
			return true
		}
	}
	return false
}

func (i *Item) BasicEqual(exch string, ass asset.Item, currency, pairedCurrency currency.Code) bool {
	if i == nil {
		return false
	}
	return i.exchange == exch &&
		i.asset == ass &&
		i.currency == currency &&
		(i.pairedWith == nil ||
			(i.pairedWith != nil && i.pairedWith.currency == pairedCurrency))
}

func (i *Item) MatchesCurrency(c currency.Code) bool {
	if i == nil {
		return false
	}
	return i.currency == c
}

func (i *Item) MatchesItemCurrency(item *Item) bool {
	if i == nil || item == nil {
		return false
	}
	return i.currency == item.currency
}

func (i *Item) MatchesExchange(item *Item) bool {
	if i == nil || item == nil {
		return false
	}
	return i.exchange == item.exchange
}
