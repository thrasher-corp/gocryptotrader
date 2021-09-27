package funding

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	fbase "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	exchangeratehost "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/exchangerate.host"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// ErrFundsNotFound used when funds are requested but the funding is not found in the manager
	ErrFundsNotFound = errors.New("funding not found")
	// ErrAlreadyExists used when a matching item or pair is already in the funding manager
	ErrAlreadyExists              = errors.New("funding already exists")
	errCannotAllocate             = errors.New("cannot allocate funds")
	errZeroAmountReceived         = errors.New("amount received less than or equal to zero")
	errNegativeAmountReceived     = errors.New("received negative decimal")
	errNotEnoughFunds             = errors.New("not enough funds")
	errCannotTransferToSameFunds  = errors.New("cannot send funds to self")
	errTransferMustBeSameCurrency = errors.New("cannot transfer to different currency")
)

// SetupFundingManager creates the funding holder. It carries knowledge about levels of funding
// across all execution handlers and enables fund transfers
func SetupFundingManager(usingExchangeLevelFunding bool) *FundManager {
	return &FundManager{usingExchangeLevelFunding: usingExchangeLevelFunding}
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
	}, nil
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

// GenerateReport builds report data for result HTML report
func (f *FundManager) GenerateReport(startDate, endDate time.Time) *Report {
	report := &Report{}
	var items []ReportItem
	var erh exchangeratehost.ExchangeRateHost
	var skipAPICheck bool
	err := erh.Setup(fbase.Settings{Enabled: true})
	if err != nil {
		log.Errorf(log.CommunicationMgr, "issue setting up exchangerate.host API %v", err)
		skipAPICheck = true
	}
	for i := range f.items {
		// exact conversion not required for initial version
		fInitialFunds, _ := f.items[i].initialFunds.Float64()
		fFinalFunds, _ := f.items[i].available.Float64()
		var initialWorthDecimal, finalWorthDecimal decimal.Decimal
		if !skipAPICheck {
			// calculating totals for shared funding across multiple currency pairs is difficult
			// converting totals using a free API is better suited as an initial concept
			// TODO convert currencies without external dependency
			if strings.Contains(f.items[i].currency.String(), "USD") {
				// not worth converting
				initialWorthDecimal = f.items[i].initialFunds
				finalWorthDecimal = f.items[i].available
			} else {
				from := f.items[i].currency.String()
				to := "USD"
				if from == "BTC" {
					// api has conversion difficulties for BTC to USD only
					to = "BUSD"
				}
				if fInitialFunds > 0 {
					initialWorth, err := erh.ConvertCurrency(from, to, "", "", "crypto", startDate, fInitialFunds, 0)
					if err != nil {
						log.Errorf(log.CommunicationMgr, "issue converting %v to %v at %v on exchangerate.host API %v", from, to, startDate, err)
					} else {
						initialWorthDecimal = decimal.NewFromFloat(initialWorth.Result)
					}
				}
				if fFinalFunds > 0 {
					finalWorth, err := erh.ConvertCurrency(from, to, "", "", "crypto", endDate, fFinalFunds, 0)
					if err != nil {
						log.Errorf(log.CommunicationMgr, "issue converting %v to %v at %v on exchangerate.host API %v", from, to, endDate, err)
					} else {
						finalWorthDecimal = decimal.NewFromFloat(finalWorth.Result)
					}
				}
			}
		}
		item := ReportItem{
			Exchange:        f.items[i].exchange,
			Asset:           f.items[i].asset,
			Currency:        f.items[i].currency,
			InitialFunds:    f.items[i].initialFunds,
			InitialFundsUSD: initialWorthDecimal.Round(2),
			TransferFee:     f.items[i].transferFee,
			FinalFunds:      f.items[i].available,
			FinalFundsUSD:   finalWorthDecimal.Round(2),
		}

		if f.items[i].initialFunds.IsZero() {
			item.ShowInfinite = true
		} else {
			item.Difference = f.items[i].available.Sub(f.items[i].initialFunds).Div(f.items[i].initialFunds).Mul(decimal.NewFromInt(100))
		}
		if f.items[i].pairedWith != nil {
			item.PairedWith = f.items[i].pairedWith.currency
		}
		report.InitialTotalUSD = report.InitialTotalUSD.Add(initialWorthDecimal).Round(2)
		report.FinalTotalUSD = report.FinalTotalUSD.Add(finalWorthDecimal).Round(2)
		items = append(items, item)
	}
	if !report.InitialTotalUSD.IsZero() {
		report.Difference = report.FinalTotalUSD.Sub(report.InitialTotalUSD).Div(report.InitialTotalUSD).Mul(decimal.NewFromInt(100))
	}
	report.Items = items
	return report
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
