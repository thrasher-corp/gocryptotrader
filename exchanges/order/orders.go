package order

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
	"github.com/thrasher-corp/gocryptotrader/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	orderSubmissionValidSides = Buy | Sell | Bid | Ask | Long | Short
	shortSide                 = Short | Sell | Ask
	longSide                  = Long | Buy | Bid

	inactiveStatuses = Filled | Cancelled | InsufficientBalance | MarketUnavailable | Rejected | PartiallyCancelled | PartiallyFilledCancelled | Expired | Closed | AnyStatus | Cancelling | Liquidated
	activeStatuses   = Active | Open | PartiallyFilled | New | PendingCancel | Hidden | AutoDeleverage | Pending
	notPlaced        = InsufficientBalance | MarketUnavailable | Rejected
)

// Public error vars for order package
var (
	ErrUnableToPlaceOrder          = errors.New("order not placed")
	ErrOrderNotFound               = errors.New("order not found")
	ErrUnknownPriceType            = errors.New("unknown price type")
	ErrAmountMustBeSet             = errors.New("amount must be set")
	ErrClientOrderIDMustBeSet      = errors.New("client order ID must be set")
	ErrUnknownSubmissionAmountType = errors.New("unknown submission amount type")
	ErrUnrecognisedOrderType       = errors.New("unrecognised order type")
)

var (
	errUnrecognisedOrderStatus  = errors.New("unrecognised order status")
	errOrderSubmitIsNil         = errors.New("order submit is nil")
	errOrderSubmitResponseIsNil = errors.New("order submit response is nil")
	errOrderDetailIsNil         = errors.New("order detail is nil")
	errAmountIsZero             = errors.New("amount is zero")
)

// IsValidOrderSubmissionSide validates that the order side is a valid submission direction
func IsValidOrderSubmissionSide(s Side) bool {
	return s != UnknownSide && orderSubmissionValidSides&s == s
}

// Validate checks the supplied data and returns whether it's valid
func (s *Submit) Validate(requirements protocol.TradingRequirements, opt ...validate.Checker) error {
	if s == nil {
		return ErrSubmissionIsNil
	}

	if s.Exchange == "" {
		return common.ErrExchangeNameNotSet
	}

	if s.Pair.IsEmpty() {
		return ErrPairIsEmpty
	}

	if s.AssetType == asset.Empty {
		return ErrAssetNotSet
	}

	if !s.AssetType.IsValid() {
		return fmt.Errorf("%q %w", s.AssetType, asset.ErrNotSupported)
	}

	if !IsValidOrderSubmissionSide(s.Side) {
		return fmt.Errorf("%w %v", ErrSideIsInvalid, s.Side)
	}

	if AllOrderTypes&s.Type != s.Type || s.Type == UnknownType {
		return ErrTypeIsInvalid
	}

	if !s.TimeInForce.IsValid() {
		return ErrInvalidTimeInForce
	}

	if s.Amount == 0 && s.QuoteAmount == 0 {
		return fmt.Errorf("submit validation error %w, amount and quote amount cannot be zero", ErrAmountIsInvalid)
	}

	if s.Amount < 0 {
		return fmt.Errorf("submit validation error base %w, suppled: %v", ErrAmountIsInvalid, s.Amount)
	}

	if s.QuoteAmount < 0 {
		return fmt.Errorf("submit validation error quote %w, suppled: %v", ErrAmountIsInvalid, s.QuoteAmount)
	}

	if s.Type == Limit && s.Price <= 0 {
		return ErrPriceMustBeSetIfLimitOrder
	}

	if requirements.ClientOrderID && s.ClientOrderID == "" {
		return fmt.Errorf("submit validation error %w, client order ID must be set to satisfy submission requirements", ErrClientOrderIDMustBeSet)
	}

	if requirements.SpotMarketBuyQuotation && s.QuoteAmount == 0 && s.Type == Market && s.AssetType == asset.Spot && s.Side.IsLong() {
		return fmt.Errorf("submit validation error %w, quote amount to be sold must be set to 'QuoteAmount' field to satisfy trading requirements", ErrAmountMustBeSet)
	}

	if requirements.SpotMarketSellBase && s.Amount == 0 && s.Type == Market && s.AssetType == asset.Spot && s.Side.IsShort() {
		return fmt.Errorf("submit validation error %w, base amount being sold must be set to 'Amount' field to satisfy trading requirements", ErrAmountMustBeSet)
	}

	for _, o := range opt {
		err := o.Check()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetTradeAmount returns the trade amount based on the exchange's trading
// requirements. Some exchanges depending on direction and order type use
// quotation (funds in balance) or base amounts. If the exchange does not have
// any specific requirements it will return the base amount.
func (s *Submit) GetTradeAmount(tr protocol.TradingRequirements) float64 {
	if s == nil {
		return 0
	}
	switch {
	case tr.SpotMarketBuyQuotation && s.AssetType == asset.Spot && s.Type == Market && s.Side.IsLong():
		return s.QuoteAmount
	case tr.SpotMarketSellBase && s.AssetType == asset.Spot && s.Type == Market && s.Side.IsShort():
		return s.Amount
	}
	return s.Amount
}

// UpdateOrderFromDetail Will update an order detail (used in order management)
// by comparing passed in and existing values
func (d *Detail) UpdateOrderFromDetail(m *Detail) error {
	if d == nil {
		return ErrOrderDetailIsNil
	}

	if m == nil {
		return fmt.Errorf("incoming %w", ErrOrderDetailIsNil)
	}

	var updated bool
	if m.TimeInForce != UnknownTIF && d.TimeInForce != m.TimeInForce {
		d.TimeInForce = m.TimeInForce
		updated = true
	}
	if d.HiddenOrder != m.HiddenOrder {
		d.HiddenOrder = m.HiddenOrder
		updated = true
	}
	if m.Price > 0 && m.Price != d.Price {
		d.Price = m.Price
		updated = true
	}
	if m.Amount > 0 && m.Amount != d.Amount {
		d.Amount = m.Amount
		updated = true
	}
	if m.LimitPriceUpper > 0 && m.LimitPriceUpper != d.LimitPriceUpper {
		d.LimitPriceUpper = m.LimitPriceUpper
		updated = true
	}
	if m.LimitPriceLower > 0 && m.LimitPriceLower != d.LimitPriceLower {
		d.LimitPriceLower = m.LimitPriceLower
		updated = true
	}
	if m.TriggerPrice > 0 && m.TriggerPrice != d.TriggerPrice {
		d.TriggerPrice = m.TriggerPrice
		updated = true
	}
	if m.QuoteAmount > 0 && m.QuoteAmount != d.QuoteAmount {
		d.QuoteAmount = m.QuoteAmount
		updated = true
	}
	if m.ExecutedAmount > 0 && m.ExecutedAmount != d.ExecutedAmount {
		d.ExecutedAmount = m.ExecutedAmount
		updated = true
	}
	if m.Fee > 0 && m.Fee != d.Fee {
		d.Fee = m.Fee
		updated = true
	}
	if m.AccountID != "" && m.AccountID != d.AccountID {
		d.AccountID = m.AccountID
		updated = true
	}
	if !m.Pair.IsEmpty() && !m.Pair.Equal(d.Pair) {
		// TODO: Add a check to see if the original pair is empty as well, but
		// error if it is changing from BTC-USD -> LTC-USD.
		d.Pair = m.Pair
		updated = true
	}
	if m.Leverage != 0 && m.Leverage != d.Leverage {
		d.Leverage = m.Leverage
		updated = true
	}
	if m.ClientID != "" && m.ClientID != d.ClientID {
		d.ClientID = m.ClientID
		updated = true
	}
	if m.ClientOrderID != "" && m.ClientOrderID != d.ClientOrderID {
		d.ClientOrderID = m.ClientOrderID
		updated = true
	}
	if m.Type != UnknownType && m.Type != d.Type {
		d.Type = m.Type
		updated = true
	}
	if m.Side != UnknownSide && m.Side != d.Side {
		d.Side = m.Side
		updated = true
	}
	if m.Status != UnknownStatus && m.Status != d.Status {
		d.Status = m.Status
		updated = true
	}
	if m.AssetType != asset.Empty && m.AssetType != d.AssetType {
		d.AssetType = m.AssetType
		updated = true
	}
	for x := range m.Trades {
		var found bool
		for y := range d.Trades {
			if d.Trades[y].TID != m.Trades[x].TID {
				continue
			}
			found = true
			if d.Trades[y].Fee != m.Trades[x].Fee {
				d.Trades[y].Fee = m.Trades[x].Fee
				updated = true
			}
			if m.Trades[x].Price != 0 && d.Trades[y].Price != m.Trades[x].Price {
				d.Trades[y].Price = m.Trades[x].Price
				updated = true
			}
			if d.Trades[y].Side != m.Trades[x].Side {
				d.Trades[y].Side = m.Trades[x].Side
				updated = true
			}
			if d.Trades[y].Type != m.Trades[x].Type {
				d.Trades[y].Type = m.Trades[x].Type
				updated = true
			}
			if d.Trades[y].Description != m.Trades[x].Description {
				d.Trades[y].Description = m.Trades[x].Description
				updated = true
			}
			if m.Trades[x].Amount != 0 && d.Trades[y].Amount != m.Trades[x].Amount {
				d.Trades[y].Amount = m.Trades[x].Amount
				updated = true
			}
			if d.Trades[y].Timestamp != m.Trades[x].Timestamp {
				d.Trades[y].Timestamp = m.Trades[x].Timestamp
				updated = true
			}
			if d.Trades[y].IsMaker != m.Trades[x].IsMaker {
				d.Trades[y].IsMaker = m.Trades[x].IsMaker
				updated = true
			}
		}
		if !found {
			d.Trades = append(d.Trades, m.Trades[x])
			updated = true
		}
		m.RemainingAmount -= m.Trades[x].Amount
	}
	if m.RemainingAmount > 0 && m.RemainingAmount != d.RemainingAmount {
		d.RemainingAmount = m.RemainingAmount
		updated = true
	}
	if updated {
		if d.LastUpdated.Equal(m.LastUpdated) {
			d.LastUpdated = time.Now()
		} else {
			d.LastUpdated = m.LastUpdated
		}
	}
	if d.Exchange == "" {
		d.Exchange = m.Exchange
	}
	if d.OrderID == "" {
		d.OrderID = m.OrderID
	}
	if d.InternalOrderID.IsNil() {
		d.InternalOrderID = m.InternalOrderID
	}
	return nil
}

// UpdateOrderFromModifyResponse Will update an order detail (used in order management)
// by comparing passed in and existing values
func (d *Detail) UpdateOrderFromModifyResponse(m *ModifyResponse) {
	var updated bool
	if m.OrderID != "" && d.OrderID != m.OrderID {
		d.OrderID = m.OrderID
		updated = true
	}
	if d.TimeInForce != m.TimeInForce && m.TimeInForce != UnknownTIF {
		d.TimeInForce = m.TimeInForce
		updated = true
	}
	if m.Price > 0 && m.Price != d.Price {
		d.Price = m.Price
		updated = true
	}
	if m.Amount > 0 && m.Amount != d.Amount {
		d.Amount = m.Amount
		updated = true
	}
	if m.TriggerPrice > 0 && m.TriggerPrice != d.TriggerPrice {
		d.TriggerPrice = m.TriggerPrice
		updated = true
	}
	if !m.Pair.IsEmpty() && !m.Pair.Equal(d.Pair) {
		// TODO: Add a check to see if the original pair is empty as well, but
		// error if it is changing from BTC-USD -> LTC-USD.
		d.Pair = m.Pair
		updated = true
	}
	if m.Type != UnknownType && m.Type != d.Type {
		d.Type = m.Type
		updated = true
	}
	if m.Side != UnknownSide && m.Side != d.Side {
		d.Side = m.Side
		updated = true
	}
	if m.Status != UnknownStatus && m.Status != d.Status {
		d.Status = m.Status
		updated = true
	}
	if m.AssetType != asset.Empty && m.AssetType != d.AssetType {
		d.AssetType = m.AssetType
		updated = true
	}
	if m.RemainingAmount > 0 && m.RemainingAmount != d.RemainingAmount {
		d.RemainingAmount = m.RemainingAmount
		updated = true
	}
	if updated {
		if d.LastUpdated.Equal(m.LastUpdated) {
			d.LastUpdated = time.Now()
		} else {
			d.LastUpdated = m.LastUpdated
		}
	}
}

// MatchFilter will return true if a detail matches the filter criteria
// empty elements are ignored
func (d *Detail) MatchFilter(f *Filter) bool {
	switch {
	case f.Exchange != "" && d.Exchange != f.Exchange:
		return false
	case f.AssetType != asset.Empty && d.AssetType != f.AssetType:
		return false
	case !f.Pair.IsEmpty() && !d.Pair.Equal(f.Pair):
		return false
	case f.OrderID != "" && d.OrderID != f.OrderID:
		return false
	case f.Type != UnknownType && f.Type != AnyType && d.Type != f.Type:
		return false
	case f.Side != UnknownSide && f.Side != AnySide && d.Side != f.Side:
		return false
	case f.Status != UnknownStatus && f.Status != AnyStatus && d.Status != f.Status:
		return false
	case f.ClientOrderID != "" && d.ClientOrderID != f.ClientOrderID:
		return false
	case f.ClientID != "" && d.ClientID != f.ClientID:
		return false
	case !f.InternalOrderID.IsNil() && d.InternalOrderID != f.InternalOrderID:
		return false
	case f.AccountID != "" && d.AccountID != f.AccountID:
		return false
	default:
		return true
	}
}

// IsActive returns true if an order has a status that indicates it is currently
// available on the exchange
func (d *Detail) IsActive() bool {
	return d.Status != UnknownStatus &&
		d.Amount > 0 &&
		d.Amount > d.ExecutedAmount &&
		activeStatuses&d.Status == d.Status
}

// IsInactive returns true if an order has a status that indicates it is
// currently not available on the exchange
func (d *Detail) IsInactive() bool {
	return d.Amount <= 0 ||
		d.Amount <= d.ExecutedAmount ||
		d.Status.IsInactive()
}

// IsInactive returns true if the status indicates it is
// currently not available on the exchange
func (s Status) IsInactive() bool {
	return inactiveStatuses&s == s
}

// WasOrderPlaced returns true if an order has a status that indicates that it
// was accepted by an exchange.
func (d *Detail) WasOrderPlaced() bool {
	if d.Status == UnknownStatus || d.Status == AnyStatus {
		return false
	}
	return notPlaced&d.Status != d.Status
}

// GenerateInternalOrderID sets a new V4 order ID or a V5 order ID if
// the V4 function returns an error
func (d *Detail) GenerateInternalOrderID() {
	if !d.InternalOrderID.IsNil() {
		return
	}
	var err error
	d.InternalOrderID, err = uuid.NewV4()
	if err != nil {
		d.InternalOrderID = uuid.NewV5(uuid.UUID{}, d.OrderID)
	}
}

// CopyToPointer will return the address of a new copy of the order Detail
// WARNING: DO NOT DEREFERENCE USE METHOD Copy().
func (d *Detail) CopyToPointer() *Detail {
	c := d.Copy()
	return &c
}

// Copy makes a full copy of underlying details NOTE: This is Addressable.
func (d *Detail) Copy() Detail {
	c := *d
	if len(d.Trades) > 0 {
		c.Trades = make([]TradeHistory, len(d.Trades))
		copy(c.Trades, d.Trades)
	}
	return c
}

// DeriveSubmitResponse will construct an order SubmitResponse when a successful
// submission has occurred. NOTE: order status is populated as order.Filled for a
// market order else order.New if an order is accepted as default, date and
// lastupdated fields have been populated as time.Now(). All fields can be
// customized in caller scope if needed.
func (s *Submit) DeriveSubmitResponse(orderID string) (*SubmitResponse, error) {
	if s == nil {
		return nil, errOrderSubmitIsNil
	}

	if orderID == "" {
		return nil, ErrOrderIDNotSet
	}

	status := New
	if s.Type == Market { // NOTE: This will need to be scrutinized.
		status = Filled
	}

	return &SubmitResponse{
		Exchange:  s.Exchange,
		Type:      s.Type,
		Side:      s.Side,
		Pair:      s.Pair,
		AssetType: s.AssetType,

		TimeInForce:   s.TimeInForce,
		ReduceOnly:    s.ReduceOnly,
		Leverage:      s.Leverage,
		Price:         s.Price,
		Amount:        s.Amount,
		QuoteAmount:   s.QuoteAmount,
		TriggerPrice:  s.TriggerPrice,
		ClientID:      s.ClientID,
		ClientOrderID: s.ClientOrderID,
		MarginType:    s.MarginType,

		LastUpdated: time.Now(),
		Date:        time.Now(),
		Status:      status,
		OrderID:     orderID,
	}, nil
}

// AdjustBaseAmount will adjust the base amount of a submit response if the
// exchange has modified the amount. This is usually due to decimal place
// restrictions or rounding. This will return an error if the amount is zero
// or the submit response is nil.
func (s *SubmitResponse) AdjustBaseAmount(a float64) error {
	if s == nil {
		return errOrderSubmitResponseIsNil
	}

	if a <= 0 {
		return errAmountIsZero
	}

	if s.Amount == a {
		return nil
	}

	// Warning because amounts should conform to exchange requirements prior to
	// call but this is not fatal.
	log.Warnf(log.ExchangeSys, "Exchange %s: has adjusted OrderID: %s requested base amount from %v to %v",
		s.Exchange,
		s.OrderID,
		s.Amount,
		a)

	s.Amount = a
	return nil
}

// AdjustQuoteAmount will adjust the quote amount of a submit response if the
// exchange has modified the amount. This is usually due to decimal place
// restrictions or rounding. This will return an error if the amount is zero
// or the submit response is nil.
func (s *SubmitResponse) AdjustQuoteAmount(a float64) error {
	if s == nil {
		return errOrderSubmitResponseIsNil
	}

	if a <= 0 {
		return errAmountIsZero
	}

	if s.QuoteAmount == a {
		return nil
	}

	// Warning because amounts should conform to exchange requirements prior to
	// call but this is not fatal.
	log.Warnf(log.ExchangeSys, "Exchange %s: has adjusted OrderID: %s requested quote amount from %v to %v",
		s.Exchange,
		s.OrderID,
		s.Amount,
		a)

	s.QuoteAmount = a
	return nil
}

// DeriveDetail will construct an order detail when a successful submission
// has occurred. Has an optional parameter field internal uuid for internal
// management.
func (s *SubmitResponse) DeriveDetail(internal uuid.UUID) (*Detail, error) {
	if s == nil {
		return nil, errOrderSubmitResponseIsNil
	}

	return &Detail{
		Exchange:  s.Exchange,
		Type:      s.Type,
		Side:      s.Side,
		Pair:      s.Pair,
		AssetType: s.AssetType,

		TimeInForce:   s.TimeInForce,
		ReduceOnly:    s.ReduceOnly,
		Leverage:      s.Leverage,
		Price:         s.Price,
		Amount:        s.Amount,
		QuoteAmount:   s.QuoteAmount,
		TriggerPrice:  s.TriggerPrice,
		ClientID:      s.ClientID,
		ClientOrderID: s.ClientOrderID,

		InternalOrderID: internal,

		LastUpdated: s.LastUpdated,
		Date:        s.Date,
		Status:      s.Status,
		OrderID:     s.OrderID,
		Trades:      s.Trades,
		Fee:         s.Fee,
		Cost:        s.Cost,
	}, nil
}

// CopyPointerOrderSlice returns a copy of all order detail and returns a slice
// of pointers.
func CopyPointerOrderSlice(old []*Detail) []*Detail {
	copySlice := make([]*Detail, len(old))
	for x := range old {
		copySlice[x] = old[x].CopyToPointer()
	}
	return copySlice
}

// DeriveModify populates a modify struct by the managed order details. Note:
// Price, Amount, Trigger price and order execution bools need to be changed
// in scope. This only derives identifiers for ease.
func (d *Detail) DeriveModify() (*Modify, error) {
	if d == nil {
		return nil, errOrderDetailIsNil
	}
	return &Modify{
		Exchange:      d.Exchange,
		OrderID:       d.OrderID,
		ClientOrderID: d.ClientOrderID,
		Type:          d.Type,
		Side:          d.Side,
		AssetType:     d.AssetType,
		Pair:          d.Pair,
	}, nil
}

// DeriveModifyResponse populates a modify response with its identifiers for
// cross exchange standard. NOTE: New OrderID and/or ClientOrderID plus any
// changes *might* need to be populated in scope.
func (m *Modify) DeriveModifyResponse() (*ModifyResponse, error) {
	if m == nil {
		return nil, errOrderDetailIsNil
	}
	return &ModifyResponse{
		Exchange:      m.Exchange,
		OrderID:       m.OrderID,
		ClientOrderID: m.ClientOrderID,
		Type:          m.Type,
		Side:          m.Side,
		AssetType:     m.AssetType,
		Pair:          m.Pair,
		TimeInForce:   m.TimeInForce,
		Price:         m.Price,
		Amount:        m.Amount,
		TriggerPrice:  m.TriggerPrice,
	}, nil
}

// DeriveCancel populates a cancel struct by the managed order details
func (d *Detail) DeriveCancel() (*Cancel, error) {
	if d == nil {
		return nil, errOrderDetailIsNil
	}
	return &Cancel{
		Exchange:      d.Exchange,
		OrderID:       d.OrderID,
		AccountID:     d.AccountID,
		ClientID:      d.ClientID,
		ClientOrderID: d.ClientOrderID,
		Type:          d.Type,
		Side:          d.Side,
		Pair:          d.Pair,
		AssetType:     d.AssetType,
	}, nil
}

// String implements the stringer interface
func (t Type) String() string {
	switch t {
	case StopMarket:
		return orderStopMarket
	case StopLimit:
		return orderStopLimit
	case Limit:
		return orderLimit
	case Market:
		return orderMarket
	case Stop:
		return orderStop
	case ConditionalStop:
		return orderConditionalStop
	case TWAP:
		return orderTWAP
	case Chase:
		return orderChase
	case TakeProfit:
		return orderTakeProfit
	case TakeProfitMarket:
		return orderTakeProfitMarket
	case TrailingStop:
		return orderTrailingStop
	case IOS:
		return orderIOS
	case Liquidation:
		return orderLiquidation
	case Trigger:
		return orderTrigger
	case OCO:
		return orderOCO
	case Bracket:
		return orderBracket
	case OptimalLimit:
		return orderOptimalLimit
	case MarketMakerProtection:
		return orderMarketMakerProtection
	case AnyType:
		return orderAnyType
	default:
		return "UNKNOWN"
	}
}

// Lower returns the type lower case string
func (t Type) Lower() string {
	return strings.ToLower(t.String())
}

// Title returns the type titleized, eg "Limit"
func (t Type) Title() string {
	return cases.Title(language.English).String(t.String())
}

// String implements the stringer interface
func (s Side) String() string {
	switch s {
	case Buy:
		return "BUY"
	case Sell:
		return "SELL"
	case Bid:
		return "BID"
	case Ask:
		return "ASK"
	case Long:
		return "LONG"
	case Short:
		return "SHORT"
	case AnySide:
		return "ANY"
	case ClosePosition:
		return "CLOSE POSITION"
		// Backtester signal types below.
	case DoNothing:
		return "DO NOTHING"
	case TransferredFunds:
		return "TRANSFERRED FUNDS"
	case CouldNotBuy:
		return "COULD NOT BUY"
	case CouldNotSell:
		return "COULD NOT SELL"
	case CouldNotShort:
		return "COULD NOT SHORT"
	case CouldNotLong:
		return "COULD NOT LONG"
	case CouldNotCloseShort:
		return "COULD NOT CLOSE SHORT"
	case CouldNotCloseLong:
		return "COULD NOT CLOSE LONG"
	case MissingData:
		return "MISSING DATA"
	default:
		return "UNKNOWN"
	}
}

// Lower returns the side lower case string
func (s Side) Lower() string {
	return strings.ToLower(s.String())
}

// Title returns the side titleized, eg "Buy"
func (s Side) Title() string {
	return cases.Title(language.English).String(s.String())
}

// IsShort returns if the side is short
func (s Side) IsShort() bool {
	return s != UnknownSide && shortSide&s == s
}

// IsLong returns if the side is long
func (s Side) IsLong() bool {
	return s != UnknownSide && longSide&s == s
}

// String implements the stringer interface
func (s Status) String() string {
	switch s {
	case AnyStatus:
		return "ANY"
	case New:
		return "NEW"
	case Active:
		return "ACTIVE"
	case PartiallyCancelled:
		return "PARTIALLY_CANCELLED"
	case PartiallyFilled:
		return "PARTIALLY_FILLED"
	case PartiallyFilledCancelled:
		return "PARTIALLY_FILLED_CANCELED"
	case Filled:
		return "FILLED"
	case Cancelled:
		return "CANCELLED"
	case PendingCancel:
		return "PENDING_CANCEL"
	case InsufficientBalance:
		return "INSUFFICIENT_BALANCE"
	case MarketUnavailable:
		return "MARKET_UNAVAILABLE"
	case Rejected:
		return "REJECTED"
	case Expired:
		return "EXPIRED"
	case Hidden:
		return "HIDDEN"
	case Open:
		return "OPEN"
	case AutoDeleverage:
		return "ADL"
	case Closed:
		return "CLOSED"
	case Pending:
		return "PENDING"
	case Cancelling:
		return "CANCELLING"
	case Liquidated:
		return "LIQUIDATED"
	case STP:
		return "SELF_TRADE_PREVENTION"
	default:
		return "UNKNOWN"
	}
}

// InferCostsAndTimes infer order costs using execution information and times
// when available
func (d *Detail) InferCostsAndTimes() {
	if d.CostAsset.IsEmpty() {
		d.CostAsset = d.Pair.Quote
	}

	if d.LastUpdated.IsZero() {
		if d.CloseTime.IsZero() {
			d.LastUpdated = d.Date
		} else {
			d.LastUpdated = d.CloseTime
		}
	}

	if d.ExecutedAmount <= 0 {
		return
	}

	if d.AverageExecutedPrice == 0 {
		if d.Cost != 0 {
			d.AverageExecutedPrice = d.Cost / d.ExecutedAmount
		} else {
			d.AverageExecutedPrice = d.Price
		}
	}
	if d.Cost == 0 {
		d.Cost = d.AverageExecutedPrice * d.ExecutedAmount
	}
}

// FilterOrdersBySide removes any order details that don't match the order
// status provided
func FilterOrdersBySide(orders *[]Detail, side Side) {
	if AnySide == side || len(*orders) == 0 {
		return
	}

	target := 0
	for i := range *orders {
		if (*orders)[i].Side == UnknownSide || (*orders)[i].Side == side {
			(*orders)[target] = (*orders)[i]
			target++
		}
	}
	*orders = (*orders)[:target]
}

// FilterOrdersByType removes any order details that don't match the order type
// provided
func FilterOrdersByType(orders *[]Detail, orderType Type) {
	if AnyType == orderType || len(*orders) == 0 {
		return
	}

	target := 0
	for i := range *orders {
		if (*orders)[i].Type == UnknownType || (*orders)[i].Type == orderType {
			(*orders)[target] = (*orders)[i]
			target++
		}
	}
	*orders = (*orders)[:target]
}

// FilterOrdersByTimeRange removes any OrderDetails outside of the time range
func FilterOrdersByTimeRange(orders *[]Detail, startTime, endTime time.Time) error {
	if len(*orders) == 0 {
		return nil
	}

	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		if errors.Is(err, common.ErrDateUnset) {
			return nil
		}
		return fmt.Errorf("cannot filter orders by time range %w", err)
	}

	target := 0
	for i := range *orders {
		if ((*orders)[i].Date.Unix() >= startTime.Unix() && (*orders)[i].Date.Unix() <= endTime.Unix()) ||
			(*orders)[i].Date.IsZero() {
			(*orders)[target] = (*orders)[i]
			target++
		}
	}
	*orders = (*orders)[:target]
	return nil
}

// FilterOrdersByPairs removes any order details that do not match the
// provided currency pairs list. It is forgiving in that the provided pairs can
// match quote or base pairs
func FilterOrdersByPairs(orders *[]Detail, pairs []currency.Pair) {
	if len(pairs) == 0 ||
		(len(pairs) == 1 && pairs[0].IsEmpty()) ||
		len(*orders) == 0 {
		return
	}

	target := 0
	for x := range *orders {
		if (*orders)[x].Pair.IsEmpty() { // If pair not set then keep
			(*orders)[target] = (*orders)[x]
			target++
			continue
		}

		if slices.ContainsFunc(pairs, func(p currency.Pair) bool {
			return (*orders)[x].Pair.EqualIncludeReciprocal(p)
		}) {
			(*orders)[target] = (*orders)[x]
			target++
		}
	}
	*orders = (*orders)[:target]
}

func (b ByPrice) Len() int {
	return len(b)
}

func (b ByPrice) Less(i, j int) bool {
	return b[i].Price < b[j].Price
}

func (b ByPrice) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByPrice the caller function to sort orders
func SortOrdersByPrice(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByPrice(*orders)))
	} else {
		sort.Sort(ByPrice(*orders))
	}
}

func (b ByOrderType) Len() int {
	return len(b)
}

func (b ByOrderType) Less(i, j int) bool {
	return b[i].Type.String() < b[j].Type.String()
}

func (b ByOrderType) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByType the caller function to sort orders
func SortOrdersByType(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderType(*orders)))
	} else {
		sort.Sort(ByOrderType(*orders))
	}
}

func (b ByCurrency) Len() int {
	return len(b)
}

func (b ByCurrency) Less(i, j int) bool {
	return b[i].Pair.String() < b[j].Pair.String()
}

func (b ByCurrency) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByCurrency the caller function to sort orders
func SortOrdersByCurrency(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByCurrency(*orders)))
	} else {
		sort.Sort(ByCurrency(*orders))
	}
}

func (b ByDate) Len() int {
	return len(b)
}

func (b ByDate) Less(i, j int) bool {
	return b[i].Date.Unix() < b[j].Date.Unix()
}

func (b ByDate) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersByDate the caller function to sort orders
func SortOrdersByDate(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByDate(*orders)))
	} else {
		sort.Sort(ByDate(*orders))
	}
}

func (b ByOrderSide) Len() int {
	return len(b)
}

func (b ByOrderSide) Less(i, j int) bool {
	return b[i].Side.String() < b[j].Side.String()
}

func (b ByOrderSide) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// SortOrdersBySide the caller function to sort orders
func SortOrdersBySide(orders *[]Detail, reverse bool) {
	if reverse {
		sort.Sort(sort.Reverse(ByOrderSide(*orders)))
	} else {
		sort.Sort(ByOrderSide(*orders))
	}
}

// StringToOrderSide for converting case insensitive order side
// and returning a real Side
func StringToOrderSide(side string) (Side, error) {
	side = strings.ToUpper(side)
	switch side {
	case Buy.String():
		return Buy, nil
	case Sell.String():
		return Sell, nil
	case Bid.String():
		return Bid, nil
	case Ask.String():
		return Ask, nil
	case Long.String():
		return Long, nil
	case Short.String():
		return Short, nil
	case AnySide.String():
		return AnySide, nil
	default:
		return UnknownSide, fmt.Errorf("%q %w", side, ErrSideIsInvalid)
	}
}

// UnmarshalJSON parses the JSON-encoded order side and stores the result
// It expects a quoted string input, and uses StringToOrderSide to parse it
func (s *Side) UnmarshalJSON(data []byte) (err error) {
	if !bytes.HasPrefix(data, []byte(`"`)) {
		// Note that we don't need to worry about invalid JSON here, it wouldn't have made it past the deserialiser far
		return &json.UnmarshalTypeError{Value: string(data), Type: reflect.TypeFor[*Side]()}
	}
	*s, err = StringToOrderSide(string(data[1 : len(data)-1])) // Remove quotes
	return
}

// MarshalJSON returns the JSON-encoded order side
func (s Side) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// StringToOrderType for converting case insensitive order type
// and returning a real Type
func StringToOrderType(oType string) (Type, error) {
	oType = strings.ToUpper(oType)
	switch oType {
	case orderLimit, "EXCHANGE LIMIT":
		return Limit, nil
	case orderMarket, "EXCHANGE MARKET":
		return Market, nil
	case orderStop, "STOP LOSS", "STOP_LOSS", "EXCHANGE STOP":
		return Stop, nil
	case orderStopLimit, "EXCHANGE STOP LIMIT", "STOP_LIMIT":
		return StopLimit, nil
	case orderStopMarket, "STOP_MARKET":
		return StopMarket, nil
	case orderTrailingStop, "TRAILING STOP", "EXCHANGE TRAILING STOP", "MOVE_ORDER_STOP":
		return TrailingStop, nil
	case orderIOS:
		return IOS, nil
	case orderAnyType:
		return AnyType, nil
	case orderTrigger:
		return Trigger, nil
	case orderOptimalLimit:
		return OptimalLimit, nil
	case orderOCO:
		return OCO, nil
	case orderConditionalStop:
		return ConditionalStop, nil
	case orderMarketMakerProtection:
		return MarketMakerProtection, nil
	case orderTWAP:
		return TWAP, nil
	case orderChase:
		return Chase, nil
	case orderTakeProfitMarket, "TAKE_PROFIT_MARKET":
		return TakeProfitMarket, nil
	case orderTakeProfit, "TAKE_PROFIT":
		return TakeProfit, nil
	case orderLiquidation:
		return Liquidation, nil
	case orderBracket, "TRIGGER_BRACKET":
		return Bracket, nil
	default:
		return UnknownType, fmt.Errorf("'%v' %w", oType, ErrUnrecognisedOrderType)
	}
}

// StringToOrderStatus for converting case insensitive order status
// and returning a real Status
func StringToOrderStatus(status string) (Status, error) {
	status = strings.ToUpper(status)
	switch status {
	case AnyStatus.String():
		return AnyStatus, nil
	case New.String(), "PLACED", "ACCEPTED", "SUBMITTED":
		return New, nil
	case Active.String(), "STATUS_ACTIVE", "LIVE":
		return Active, nil
	case PartiallyFilled.String(), "PARTIAL-FILLED", "PARTIALLY MATCHED", "PARTIALLY FILLED":
		return PartiallyFilled, nil
	case Filled.String(), "FULLY MATCHED", "FULLY FILLED", "ORDER_FULLY_TRANSACTED", "EFFECTIVE":
		return Filled, nil
	case PartiallyCancelled.String(), "PARTIAL-CANCELED", "PARTIALLY CANCELLED", "ORDER_PARTIALLY_TRANSACTED":
		return PartiallyCancelled, nil
	case PartiallyFilledCancelled.String(), "PARTIALLYFILLEDCANCELED":
		return PartiallyFilledCancelled, nil
	case Open.String():
		return Open, nil
	case Closed.String(), "POSITION_CLOSED":
		return Closed, nil
	case Cancelled.String(), "CANCELED", "ORDER_CANCELLED":
		return Cancelled, nil
	case Pending.String():
		return Pending, nil
	case PendingCancel.String(), "PENDING CANCEL", "PENDING CANCELLATION":
		return PendingCancel, nil
	case Rejected.String(), "FAILED", "ORDER_FAILED":
		return Rejected, nil
	case Expired.String():
		return Expired, nil
	case Hidden.String():
		return Hidden, nil
	case InsufficientBalance.String():
		return InsufficientBalance, nil
	case MarketUnavailable.String():
		return MarketUnavailable, nil
	case Cancelling.String():
		return Cancelling, nil
	case Liquidated.String():
		return Liquidated, nil
	case AutoDeleverage.String(), "AUTO_DELEVERAGED":
		return AutoDeleverage, nil
	case STP.String(), "STP":
		return STP, nil
	default:
		return UnknownStatus, fmt.Errorf("%q %w", status, errUnrecognisedOrderStatus)
	}
}

func (o *ClassificationError) Error() string {
	if o.OrderID != "" {
		return fmt.Sprintf("Exchange %s: OrderID: %s classification error: %v",
			o.Exchange,
			o.OrderID,
			o.Err)
	}
	return fmt.Sprintf("Exchange %s: classification error: %v",
		o.Exchange,
		o.Err)
}

// StandardCancel defines an option in the validator to make sure an ID is set
// for a standard cancel
func (c *Cancel) StandardCancel() validate.Checker {
	return validate.Check(func() error {
		if c.OrderID == "" {
			return ErrOrderIDNotSet
		}
		return nil
	})
}

// PairAssetRequired is a validation check for when a cancel request
// requires an asset type and currency pair to be present
func (c *Cancel) PairAssetRequired() validate.Checker {
	return validate.Check(func() error {
		if c.Pair.IsEmpty() {
			return ErrPairIsEmpty
		}

		if c.AssetType == asset.Empty {
			return ErrAssetNotSet
		}
		return nil
	})
}

// Validate checks internal struct requirements
func (c *Cancel) Validate(opt ...validate.Checker) error {
	if c == nil {
		return ErrCancelOrderIsNil
	}

	var errs error
	for _, o := range opt {
		err := o.Check()
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// Validate checks internal struct requirements and returns filter requirement
// options for wrapper standardization procedures.
func (g *MultiOrderRequest) Validate(opt ...validate.Checker) error {
	if g == nil {
		return ErrGetOrdersRequestIsNil
	}

	if !g.AssetType.IsValid() {
		return fmt.Errorf("%v %w", g.AssetType, asset.ErrNotSupported)
	}

	if g.Side == UnknownSide {
		return ErrSideIsInvalid
	}

	if g.Type == UnknownType {
		return ErrUnrecognisedOrderType
	}

	var errs error
	for _, o := range opt {
		err := o.Check()
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// Filter reduces slice by optional fields
func (g *MultiOrderRequest) Filter(exch string, orders []Detail) FilteredOrders {
	filtered := make([]Detail, len(orders))
	copy(filtered, orders)
	FilterOrdersByPairs(&filtered, g.Pairs)
	FilterOrdersByType(&filtered, g.Type)
	FilterOrdersBySide(&filtered, g.Side)
	err := FilterOrdersByTimeRange(&filtered, g.StartTime, g.EndTime)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", exch, err)
	}
	return filtered
}

// Validate checks internal struct requirements
func (m *Modify) Validate(opt ...validate.Checker) error {
	if m == nil {
		return ErrModifyOrderIsNil
	}

	if m.Pair.IsEmpty() {
		return ErrPairIsEmpty
	}

	if m.AssetType == asset.Empty {
		return ErrAssetNotSet
	}

	var errs error
	for _, o := range opt {
		err := o.Check()
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	if errs != nil {
		return errs
	}
	if m.ClientOrderID == "" && m.OrderID == "" {
		return ErrOrderIDNotSet
	}
	return nil
}

// String implements the stringer interface
func (t PriceType) String() string {
	switch t {
	case LastPrice:
		return "LastPrice"
	case IndexPrice:
		return "IndexPrice"
	case MarkPrice:
		return "MarkPrice"
	default:
		return ""
	}
}

// StringToPriceType for converting case insensitive order side
// and returning a real Side
func (t PriceType) StringToPriceType(priceType string) (PriceType, error) {
	priceType = strings.ToLower(priceType)
	switch priceType {
	case "lastprice", "":
		return LastPrice, nil
	case "indexprice":
		return IndexPrice, nil
	case "markprice":
		return MarkPrice, nil
	default:
		return UnknownPriceType, ErrUnknownPriceType
	}
}

// String implements the stringer interface
func (t TrackingMode) String() string {
	switch t {
	case Distance:
		return "distance"
	case Percentage:
		return "percentage"
	default:
		return ""
	}
}

// StringToTrackingMode converts TrackingMode instance from string
func StringToTrackingMode(mode string) TrackingMode {
	mode = strings.ToLower(mode)
	switch mode {
	case "distance":
		return Distance
	case "percentage":
		return Percentage
	default:
		return UnknownTrackingMode
	}
}

// Add adds a new orderID and status to the CancelAllResponse
func (c *CancelAllResponse) Add(orderID, status string) {
	if c.Status == nil {
		c.Status = make(map[string]string)
	}
	c.Status[orderID] = status
}
