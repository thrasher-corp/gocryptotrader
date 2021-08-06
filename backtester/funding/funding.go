package funding

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	ErrCannotAllocate = errors.New("cannot allocate funds")
	ErrFundsNotFound  = errors.New("funding not found")
)

// AllFunds is the benevolent holder of all funding levels across all
// currencies used in the backtester
type AllFunds struct {
	UsingExchangeLevelFunding bool
	Items                     []*Item
}

func (a *AllFunds) AddItem(exch string, ass asset.Item, ci currency.Code, initialFunds float64) error {
	item := &Item{
		Exchange:     exch,
		Asset:        ass,
		Item:         ci,
		InitialFunds: initialFunds,
		Available:    initialFunds,
	}
	a.Items = append(a.Items, item)
	return nil
}

func (a *AllFunds) AddPair(exch string, ass asset.Item, cp currency.Pair, initialFunds float64) error {
	base := &Item{
		Exchange: exch,
		Asset:    ass,
		Item:     cp.Base,
	}
	quote := &Item{
		Exchange:     exch,
		Asset:        ass,
		Item:         cp.Quote,
		InitialFunds: initialFunds,
		Available:    initialFunds,
		PairedWith:   base,
	}
	base.PairedWith = quote
	a.Items = append(a.Items, base, quote)
	return nil
}

// IsUsingExchangeLevelFunding returns if using UsingExchangeLevelFunding
func (a *AllFunds) IsUsingExchangeLevelFunding(e common.EventHandler) bool {
	return a.UsingExchangeLevelFunding
}

// GetFundingForEvent This will construct a funding based on a backtesting event
func (a *AllFunds) GetFundingForEvent(e common.EventHandler) (*Pair, error) {
	return a.GetFundingForEAP(e.GetExchange(), e.GetAssetType(), e.Pair())
}

// GetFundingForEAC This will construct a funding based on the exchange, asset, currency code
func (a *AllFunds) GetFundingForEAC(exch string, ass asset.Item, c currency.Code) (*Item, error) {
	for i := range a.Items {
		if a.Items[i].Item == c &&
			a.Items[i].Exchange == exch &&
			a.Items[i].Asset == ass {
			return a.Items[i], nil
		}
	}
	return nil, ErrFundsNotFound
}

// GetFundingForEAP This will construct a funding based on the exchange, asset, currency pair
func (a *AllFunds) GetFundingForEAP(exch string, ass asset.Item, p currency.Pair) (*Pair, error) {
	var resp Pair
	for i := range a.Items {
		if a.Items[i].Item == p.Quote &&
			a.Items[i].Exchange == exch &&
			a.Items[i].Asset == ass &&
			(!a.UsingExchangeLevelFunding || (a.Items[i].PairedWith != nil && a.Items[i].PairedWith.Item == p.Quote)) {
			resp.Quote = a.Items[i]
			continue
		}
		if a.Items[i].Item == p.Base &&
			a.Items[i].Exchange == exch &&
			a.Items[i].Asset == ass &&
			(!a.UsingExchangeLevelFunding || (a.Items[i].PairedWith != nil && a.Items[i].PairedWith.Item == p.Quote)) {
			resp.Base = a.Items[i]
		}
	}
	if resp.Base == nil || resp.Quote == nil {
		return nil, ErrFundsNotFound
	}
	return &resp, nil
}

type Pair struct {
	Base  *Item
	Quote *Item
}

func (p *Pair) Reserve(amount float64, side order.Side) error {
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

func (p *Pair) Release(amount, diff float64, side order.Side) error {
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

func (p *Pair) Increase(amount float64, side order.Side) {
	switch side {
	case order.Buy:
		p.Base.Increase(amount)
	case order.Sell:
		p.Quote.Increase(amount)
	}
}

func (i *Item) Reserve(amount float64) error {
	if amount > i.Available {
		return fmt.Errorf("%w for %v %v %v. Requested %v Available: %v",
			ErrCannotAllocate,
			i.Exchange,
			i.Asset,
			i.Item,
			amount,
			i.Available)
	}
	i.Available -= amount
	i.Reserved += amount
	return nil
}

// Release lowers the reserved amount and appends any differences
// as a result of any exchange level modifications when ordering
func (i *Item) Release(amount, diff float64) error {
	if amount > i.Reserved {
		return fmt.Errorf("%w for %v %v %v. Requested %v Reserved: %v",
			ErrCannotAllocate,
			i.Exchange,
			i.Asset,
			i.Item,
			amount,
			i.Reserved)
	}
	i.Reserved -= amount
	i.Available += diff
	return nil
}

func (i *Item) Increase(amount float64) {
	i.Available += amount
}

// Item
type Item struct {
	Exchange     string
	Asset        asset.Item
	Item         currency.Code
	InitialFunds float64
	Available    float64
	Reserved     float64
	PairedWith   *Item
}

// IFundingReserver limits funding usage for portfolio event handling
type IFundingReserver interface {
	IsUsingExchangeLevelFunding() bool
	GetFundingForEAC(string, asset.Item, currency.Code) (IItemReserver, error)
	GetFundingForEvent(common.EventHandler) (IPairReserver, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (IPairReserver, error)
}

// IFundingReleaser limits funding usage for exchange event handling
type IFundingReleaser interface {
	IsUsingExchangeLevelFunding() bool
	GetFundingForEAC(string, asset.Item, currency.Code) (IItemReleaser, error)
	GetFundingForEvent(common.EventHandler) (IPairReleaser, error)
	GetFundingForEAP(string, asset.Item, currency.Pair) (IPairReleaser, error)
}

// IPairReserver limits funding usage for portfolio event handling
type IPairReserver interface {
	Reserve(float64, order.Side) error
}

// IPairReleaser limits funding usage for exchange event handling
type IPairReleaser interface {
	Increase(float64, order.Side)
	Release(float64, float64, order.Side) error
}

// IItemReserver limits funding usage for portfolio event handling
type IItemReserver interface {
	Reserve(float64) error
}

// IItemReleaser limits funding usage for exchange event handling
type IItemReleaser interface {
	Increase(float64)
	Release(float64, float64) error
}

// perhaps funding should also include sizing? This would allow sizing to easily occur across portfolio and exchange and stay within size
// but hold off, because scope is really hard here
