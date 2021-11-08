package funding

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding/trackingcurrencies"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrFundsNotFound used when funds are requested but the funding is not found in the manager
	ErrFundsNotFound = errors.New("funding not found")
	// ErrAlreadyExists used when a matching item or pair is already in the funding manager
	ErrAlreadyExists = errors.New("funding already exists")
	// ErrUSDTrackingDisabled used when attempting to track USD values when disabled
	ErrUSDTrackingDisabled = errors.New("USD tracking disabled")
	// ErrNotCollateral is returned when a user requests collateral from a non-collateral pair
	ErrNotCollateral              = errors.New("not a collateral pair")
	errCannotAllocate             = errors.New("cannot allocate funds")
	errZeroAmountReceived         = errors.New("amount received less than or equal to zero")
	errNegativeAmountReceived     = errors.New("received negative decimal")
	errNotEnoughFunds             = errors.New("not enough funds")
	errCannotTransferToSameFunds  = errors.New("cannot send funds to self")
	errTransferMustBeSameCurrency = errors.New("cannot transfer to different currency")
	errCannotMatchTrackingToItem  = errors.New("cannot match tracking data to funding items")
)

// SetupFundingManager creates the funding holder. It carries knowledge about levels of funding
// across all execution handlers and enables fund transfers
func SetupFundingManager(usingExchangeLevelFunding, disableUSDTracking bool) *FundManager {
	return &FundManager{
		usingExchangeLevelFunding: usingExchangeLevelFunding,
		disableUSDTracking:        disableUSDTracking,
	}
}

// CreateItem creates a new funding item
func CreateItem(exch string, a asset.Item, ci currency.Code, initialFunds, transferFee decimal.Decimal) (*Item, error) {
	if initialFunds.IsNegative() {
		return nil, fmt.Errorf("%v %v %v %w initial funds: %v", exch, a, ci, errNegativeAmountReceived, initialFunds)
	}
	if transferFee.IsNegative() {
		return nil, fmt.Errorf("%v %v %v %w transfer fee: %v", exch, a, ci, errNegativeAmountReceived, transferFee)
	}

	return &Item{
		exchange:     exch,
		asset:        a,
		currency:     ci,
		initialFunds: initialFunds,
		available:    initialFunds,
		transferFee:  transferFee,
		snapshot:     make(map[time.Time]ItemSnapshot),
	}, nil
}

// CreateSnapshot creates a Snapshot for an event's point in time
// as funding.snapshots is a map, it allows for the last event
// in the chronological list to establish the canon at X time
func (f *FundManager) CreateSnapshot(t time.Time) {
	for i := range f.items {
		if f.items[i].snapshot == nil {
			f.items[i].snapshot = make(map[time.Time]ItemSnapshot)
		}
		iss := ItemSnapshot{
			Available: f.items[i].available,
			Time:      t,
		}
		if !f.disableUSDTracking {
			var usdClosePrice decimal.Decimal
			if f.items[i].usdTrackingCandles == nil {
				continue
			}
			usdCandles := f.items[i].usdTrackingCandles.GetStream()
			for j := range usdCandles {
				if usdCandles[j].GetTime().Equal(t) {
					usdClosePrice = usdCandles[j].GetClosePrice()
					break
				}
			}
			iss.USDClosePrice = usdClosePrice
			iss.USDValue = usdClosePrice.Mul(f.items[i].available)
		}

		f.items[i].snapshot[t] = iss
	}
}

// AddUSDTrackingData adds USD tracking data to a funding item
// only in the event that it is not USD and there is data
func (f *FundManager) AddUSDTrackingData(k *kline.DataFromKline) error {
	if f == nil || f.items == nil {
		return common.ErrNilArguments
	}
	if f.disableUSDTracking {
		return ErrUSDTrackingDisabled
	}
	baseSet := false
	quoteSet := false
	var basePairedWith currency.Code
	for i := range f.items {
		if baseSet && quoteSet {
			return nil
		}
		if strings.EqualFold(f.items[i].exchange, k.Item.Exchange) &&
			f.items[i].asset == k.Item.Asset {
			if f.items[i].currency == k.Item.Pair.Base {
				if f.items[i].usdTrackingCandles == nil &&
					trackingcurrencies.CurrencyIsUSDTracked(k.Item.Pair.Quote) {
					f.items[i].usdTrackingCandles = k
					if f.items[i].pairedWith != nil {
						basePairedWith = f.items[i].pairedWith.currency
					}
				}
				baseSet = true
			}
			if trackingcurrencies.CurrencyIsUSDTracked(f.items[i].currency) {
				if f.items[i].pairedWith != nil && f.items[i].currency != basePairedWith {
					continue
				}
				if f.items[i].usdTrackingCandles == nil {
					usdCandles := gctkline.Item{
						Exchange: k.Item.Exchange,
						Pair:     currency.Pair{Delimiter: k.Item.Pair.Delimiter, Base: f.items[i].currency, Quote: currency.USD},
						Asset:    k.Item.Asset,
						Interval: k.Item.Interval,
						Candles:  make([]gctkline.Candle, len(k.Item.Candles)),
					}
					copy(usdCandles.Candles, k.Item.Candles)
					for j := range usdCandles.Candles {
						// usd stablecoins do not always match in value,
						// this is a simplified implementation that can allow
						// USD tracking for many different currencies across many exchanges
						// without retrieving n candle history and exchange rates
						usdCandles.Candles[j].Open = 1
						usdCandles.Candles[j].High = 1
						usdCandles.Candles[j].Low = 1
						usdCandles.Candles[j].Close = 1
					}
					cpy := *k
					cpy.Item = usdCandles
					if err := cpy.Load(); err != nil {
						return err
					}
					f.items[i].usdTrackingCandles = &cpy
				}
				quoteSet = true
			}
		}
	}
	if baseSet {
		return nil
	}
	return fmt.Errorf("%w %v %v %v", errCannotMatchTrackingToItem, k.Item.Exchange, k.Item.Asset, k.Item.Pair)
}

// CreatePair adds two funding items and associates them with one another
// the association allows for the same currency to be used multiple times when
// usingExchangeLevelFunding is false. eg BTC-USDT and LTC-USDT do not share the same
// USDT level funding
func CreatePair(base, quote *Item) (*Pair, error) {
	if base == nil {
		return nil, fmt.Errorf("base %w", common.ErrNilArguments)
	}
	if quote == nil {
		return nil, fmt.Errorf("quote %w", common.ErrNilArguments)
	}
	// copy to prevent the off chance of sending in the same base OR quote
	// to create a new pair with a new base OR quote
	bCopy := *base
	qCopy := *quote
	bCopy.pairedWith = &qCopy
	qCopy.pairedWith = &bCopy
	return &Pair{Base: &bCopy, Quote: &qCopy}, nil
}

// Reset clears all settings
func (f *FundManager) Reset() {
	*f = FundManager{}
}

// USDTrackingDisabled clears all settings
func (f *FundManager) USDTrackingDisabled() bool {
	return f.disableUSDTracking
}

// GenerateReport builds report data for result HTML report
func (f *FundManager) GenerateReport() *Report {
	report := Report{
		USDTotalsOverTime:         make(map[time.Time]ItemSnapshot),
		UsingExchangeLevelFunding: f.usingExchangeLevelFunding,
		DisableUSDTracking:        f.disableUSDTracking,
	}
	var items []ReportItem
	for i := range f.items {
		item := ReportItem{
			Exchange:     f.items[i].exchange,
			Asset:        f.items[i].asset,
			Currency:     f.items[i].currency,
			InitialFunds: f.items[i].initialFunds,
			TransferFee:  f.items[i].transferFee,
			FinalFunds:   f.items[i].available,
		}
		if !f.disableUSDTracking &&
			f.items[i].usdTrackingCandles != nil {
			usdStream := f.items[i].usdTrackingCandles.GetStream()
			item.USDInitialFunds = f.items[i].initialFunds.Mul(usdStream[0].GetClosePrice())
			item.USDFinalFunds = f.items[i].available.Mul(usdStream[len(usdStream)-1].GetClosePrice())
			item.USDInitialCostForOne = usdStream[0].GetClosePrice()
			item.USDFinalCostForOne = usdStream[len(usdStream)-1].GetClosePrice()
			item.USDPairCandle = f.items[i].usdTrackingCandles
		}

		var pricingOverTime []ItemSnapshot
		for _, v := range f.items[i].snapshot {
			pricingOverTime = append(pricingOverTime, v)
			if !f.disableUSDTracking {
				usdTotalForPeriod := report.USDTotalsOverTime[v.Time]
				usdTotalForPeriod.Time = v.Time
				usdTotalForPeriod.USDValue = usdTotalForPeriod.USDValue.Add(v.USDValue)
				report.USDTotalsOverTime[v.Time] = usdTotalForPeriod
			}
		}
		sort.Slice(pricingOverTime, func(i, j int) bool {
			return pricingOverTime[i].Time.Before(pricingOverTime[j].Time)
		})
		item.Snapshots = pricingOverTime

		if f.items[i].initialFunds.IsZero() {
			item.ShowInfinite = true
		} else {
			item.Difference = f.items[i].available.Sub(f.items[i].initialFunds).Div(f.items[i].initialFunds).Mul(decimal.NewFromInt(100))
		}
		if f.items[i].pairedWith != nil {
			item.PairedWith = f.items[i].pairedWith.currency
		}

		items = append(items, item)
	}
	report.Items = items
	return &report
}

// Transfer allows transferring funds from one pretend exchange to another
func (f *FundManager) Transfer(amount decimal.Decimal, sender, receiver *Item, inclusiveFee bool) error {
	if sender == nil || receiver == nil {
		return common.ErrNilArguments
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return errZeroAmountReceived
	}
	if inclusiveFee {
		if sender.available.LessThan(amount) {
			return fmt.Errorf("%w for %v", errNotEnoughFunds, sender.currency)
		}
	} else {
		if sender.available.LessThan(amount.Add(sender.transferFee)) {
			return fmt.Errorf("%w for %v", errNotEnoughFunds, sender.currency)
		}
	}

	if sender.currency != receiver.currency {
		return errTransferMustBeSameCurrency
	}
	if sender.currency == receiver.currency &&
		sender.exchange == receiver.exchange &&
		sender.asset == receiver.asset {
		return fmt.Errorf("%v %v %v %w", sender.exchange, sender.asset, sender.currency, errCannotTransferToSameFunds)
	}

	sendAmount := amount
	receiveAmount := amount
	if inclusiveFee {
		receiveAmount = amount.Sub(sender.transferFee)
	} else {
		sendAmount = amount.Add(sender.transferFee)
	}
	err := sender.Reserve(sendAmount)
	if err != nil {
		return err
	}
	receiver.IncreaseAvailable(receiveAmount)
	return sender.Release(sendAmount, decimal.Zero)
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

// AddPair adds a pair to the fund manager if it does not exist
func (f *FundManager) AddPair(p *Pair) error {
	if f.Exists(p.Base) {
		return fmt.Errorf("%w %v", ErrAlreadyExists, p.Base)
	}
	if f.Exists(p.Quote) {
		return fmt.Errorf("%w %v", ErrAlreadyExists, p.Quote)
	}
	f.items = append(f.items, p.Base, p.Quote)
	return nil
}

// IsUsingExchangeLevelFunding returns if using usingExchangeLevelFunding
func (f *FundManager) IsUsingExchangeLevelFunding() bool {
	return f.usingExchangeLevelFunding
}

// GetFundingForEvent This will construct a funding based on a backtesting event
func (f *FundManager) GetFundingForEvent(ev common.EventHandler) (*Pair, error) {
	return f.GetFundingForEAP(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
}

// GetFundingForEAC This will construct a funding based on the exchange, asset, currency code
func (f *FundManager) GetFundingForEAC(exch string, a asset.Item, c currency.Code) (*Item, error) {
	for i := range f.items {
		if f.items[i].BasicEqual(exch, a, c, currency.Code{}) {
			return f.items[i], nil
		}
	}
	return nil, ErrFundsNotFound
}

// GetFundingForEAP This will construct a funding based on the exchange, asset, currency pair
func (f *FundManager) GetFundingForEAP(exch string, a asset.Item, p currency.Pair) (*Pair, error) {
	var resp Pair
	for i := range f.items {
		if f.items[i].BasicEqual(exch, a, p.Base, p.Quote) {
			resp.Base = f.items[i]
			continue
		}
		if f.items[i].BasicEqual(exch, a, p.Quote, p.Base) {
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

func (p *Pair) GetPairReader() (IPairReader, error) {
	return p, nil
}

func (p *Pair) GetCollateralReader() (ICollateralReader, error) {
	return nil, ErrNotCollateral
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
			errCannotAllocate,
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
			errCannotAllocate,
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

func (c *CollateralPair) CanPlaceOrder(_ order.Side) bool {
	return c.Collateral.CanPlaceOrder()
}

// Reserve allocates an amount of funds to be used at a later time
// it prevents multiple events from claiming the same resource
func (i *Item) Reserve(amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return errZeroAmountReceived
	}
	if amount.GreaterThan(i.available) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Available: %v",
			errCannotAllocate,
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
		return errZeroAmountReceived
	}
	if diff.IsNegative() {
		return fmt.Errorf("%w diff", errNegativeAmountReceived)
	}
	if amount.GreaterThan(i.reserved) {
		return fmt.Errorf("%w for %v %v %v. Requested %v Reserved: %v",
			errCannotAllocate,
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

// CanPlaceOrder checks if the item has any funds available
func (i *Item) CanPlaceOrder() bool {
	return i.available.GreaterThan(decimal.Zero)
}

// Equal checks for equality via an Item to compare to
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

// BasicEqual checks for equality via passed in values
func (i *Item) BasicEqual(exch string, a asset.Item, currency, pairedCurrency currency.Code) bool {
	return i != nil &&
		i.exchange == exch &&
		i.asset == a &&
		i.currency == currency &&
		(i.pairedWith == nil ||
			(i.pairedWith != nil && i.pairedWith.currency == pairedCurrency))
}

// MatchesCurrency checks that an item's currency is equal
func (i *Item) MatchesCurrency(c currency.Code) bool {
	return i != nil && i.currency == c
}

// MatchesItemCurrency checks that an item's currency is equal
func (i *Item) MatchesItemCurrency(item *Item) bool {
	return i != nil && item != nil && i.currency == item.currency
}

// MatchesExchange checks that an item's exchange is equal
func (i *Item) MatchesExchange(item *Item) bool {
	return i != nil && item != nil && i.exchange == item.exchange
}
