package order

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
)

var errTimeInForceConflict = errors.New("multiple time in force options applied")

// Validate checks the supplied data and returns whether or not it's valid
func (s *Submit) Validate(opt ...validate.Checker) error {
	if s == nil {
		return ErrSubmissionIsNil
	}

	if s.Pair.IsEmpty() {
		return ErrPairIsEmpty
	}

	if s.AssetType == "" {
		return ErrAssetNotSet
	}

	if s.Side != Buy &&
		s.Side != Sell &&
		s.Side != Bid &&
		s.Side != Ask {
		return ErrSideIsInvalid
	}

	if s.Type != Market && s.Type != Limit {
		return ErrTypeIsInvalid
	}

	if s.ImmediateOrCancel && s.FillOrKill {
		return errTimeInForceConflict
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

	for _, o := range opt {
		err := o.Check()
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateOrderFromDetail Will update an order detail (used in order management)
// by comparing passed in and existing values
func (d *Detail) UpdateOrderFromDetail(m *Detail) {
	var updated bool
	if d.ImmediateOrCancel != m.ImmediateOrCancel {
		d.ImmediateOrCancel = m.ImmediateOrCancel
		updated = true
	}
	if d.HiddenOrder != m.HiddenOrder {
		d.HiddenOrder = m.HiddenOrder
		updated = true
	}
	if d.FillOrKill != m.FillOrKill {
		d.FillOrKill = m.FillOrKill
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
	if m.PostOnly != d.PostOnly {
		d.PostOnly = m.PostOnly
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
	if m.WalletAddress != "" && m.WalletAddress != d.WalletAddress {
		d.WalletAddress = m.WalletAddress
		updated = true
	}
	if m.Type != "" && m.Type != d.Type {
		d.Type = m.Type
		updated = true
	}
	if m.Side != "" && m.Side != d.Side {
		d.Side = m.Side
		updated = true
	}
	if m.Status != "" && m.Status != d.Status {
		d.Status = m.Status
		updated = true
	}
	if m.AssetType != "" && m.AssetType != d.AssetType {
		d.AssetType = m.AssetType
		updated = true
	}
	if m.Trades != nil {
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
	if d.ID == "" {
		d.ID = m.ID
	}
	if d.InternalOrderID == "" {
		d.InternalOrderID = m.InternalOrderID
	}
}

// UpdateOrderFromModify Will update an order detail (used in order management)
// by comparing passed in and existing values
func (d *Detail) UpdateOrderFromModify(m *Modify) {
	var updated bool
	if m.ID != "" && d.ID != m.ID {
		d.ID = m.ID
		updated = true
	}
	if d.ImmediateOrCancel != m.ImmediateOrCancel {
		d.ImmediateOrCancel = m.ImmediateOrCancel
		updated = true
	}
	if d.HiddenOrder != m.HiddenOrder {
		d.HiddenOrder = m.HiddenOrder
		updated = true
	}
	if d.FillOrKill != m.FillOrKill {
		d.FillOrKill = m.FillOrKill
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
	if m.PostOnly != d.PostOnly {
		d.PostOnly = m.PostOnly
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
	if m.WalletAddress != "" && m.WalletAddress != d.WalletAddress {
		d.WalletAddress = m.WalletAddress
		updated = true
	}
	if m.Type != "" && m.Type != d.Type {
		d.Type = m.Type
		updated = true
	}
	if m.Side != "" && m.Side != d.Side {
		d.Side = m.Side
		updated = true
	}
	if m.Status != "" && m.Status != d.Status {
		d.Status = m.Status
		updated = true
	}
	if m.AssetType != "" && m.AssetType != d.AssetType {
		d.AssetType = m.AssetType
		updated = true
	}
	if m.Trades != nil {
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
	if f.Exchange != "" && !strings.EqualFold(d.Exchange, f.Exchange) {
		return false
	}
	if f.AssetType != "" && d.AssetType != f.AssetType {
		return false
	}
	if !f.Pair.IsEmpty() && !d.Pair.Equal(f.Pair) {
		return false
	}
	if f.ID != "" && d.ID != f.ID {
		return false
	}
	if f.Type != "" && f.Type != AnyType && d.Type != f.Type {
		return false
	}
	if f.Side != "" && f.Side != AnySide && d.Side != f.Side {
		return false
	}
	if f.Status != "" && f.Status != AnyStatus && d.Status != f.Status {
		return false
	}
	if f.ClientOrderID != "" && d.ClientOrderID != f.ClientOrderID {
		return false
	}
	if f.ClientID != "" && d.ClientID != f.ClientID {
		return false
	}
	if f.InternalOrderID != "" && d.InternalOrderID != f.InternalOrderID {
		return false
	}
	if f.AccountID != "" && d.AccountID != f.AccountID {
		return false
	}
	if f.WalletAddress != "" && d.WalletAddress != f.WalletAddress {
		return false
	}
	return true
}

// IsActive returns true if an order has a status that indicates it is
// currently available on the exchange
func (d *Detail) IsActive() bool {
	if d.Amount <= 0 || d.Amount <= d.ExecutedAmount {
		return false
	}
	return d.Status == Active || d.Status == Open || d.Status == PartiallyFilled || d.Status == New ||
		d.Status == AnyStatus || d.Status == PendingCancel || d.Status == Hidden || d.Status == UnknownStatus ||
		d.Status == AutoDeleverage || d.Status == Pending
}

// IsInactive returns true if an order has a status that indicates it is
// currently not available on the exchange
func (d *Detail) IsInactive() bool {
	if d.Amount <= 0 || d.Amount <= d.ExecutedAmount {
		return true
	}
	return d.Status == Filled || d.Status == Cancelled || d.Status == InsufficientBalance || d.Status == MarketUnavailable ||
		d.Status == Rejected || d.Status == PartiallyCancelled || d.Status == Expired || d.Status == Closed
}

// GenerateInternalOrderID sets a new V4 order ID or a V5 order ID if
// the V4 function returns an error
func (d *Detail) GenerateInternalOrderID() {
	if d.InternalOrderID == "" {
		var id uuid.UUID
		id, err := uuid.NewV4()
		if err != nil {
			id = uuid.NewV5(uuid.UUID{}, d.ID)
		}
		d.InternalOrderID = id.String()
	}
}

// Copy will return a copy of Detail
func (d *Detail) Copy() Detail {
	c := *d
	if len(d.Trades) > 0 {
		c.Trades = make([]TradeHistory, len(d.Trades))
		copy(c.Trades, d.Trades)
	}
	return c
}

// String implements the stringer interface
func (t Type) String() string {
	return string(t)
}

// Lower returns the type lower case string
func (t Type) Lower() string {
	return strings.ToLower(string(t))
}

// Title returns the type titleized, eg "Limit"
func (t Type) Title() string {
	return strings.Title(strings.ToLower(string(t)))
}

// String implements the stringer interface
func (s Side) String() string {
	return string(s)
}

// Lower returns the side lower case string
func (s Side) Lower() string {
	return strings.ToLower(string(s))
}

// Title returns the side titleized, eg "Buy"
func (s Side) Title() string {
	return strings.Title(strings.ToLower(string(s)))
}

// IsShort returns if the side is short
func (s Side) IsShort() bool {
	return s == Short || s == Sell
}

// IsLong returns if the side is long
func (s Side) IsLong() bool {
	return s == Long || s == Buy
}

// String implements the stringer interface
func (s Status) String() string {
	return string(s)
}

// InferCostsAndTimes infer order costs using execution information and times when available
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

// FilterOrdersBySide removes any order details that don't match the
// order status provided
func FilterOrdersBySide(orders *[]Detail, side Side) {
	if side == "" || side == AnySide {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].Side), string(side)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByType removes any order details that don't match the order type
// provided
func FilterOrdersByType(orders *[]Detail, orderType Type) {
	if orderType == "" || orderType == AnyType {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		if strings.EqualFold(string((*orders)[i].Type), string(orderType)) {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByTimeRange removes any OrderDetails outside of the time range
func FilterOrdersByTimeRange(orders *[]Detail, startTime, endTime time.Time) {
	if startTime.IsZero() ||
		endTime.IsZero() ||
		startTime.Unix() == 0 ||
		endTime.Unix() == 0 ||
		endTime.Before(startTime) {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		if ((*orders)[i].Date.Unix() >= startTime.Unix() && (*orders)[i].Date.Unix() <= endTime.Unix()) ||
			(*orders)[i].Date.IsZero() {
			filteredOrders = append(filteredOrders, (*orders)[i])
		}
	}

	*orders = filteredOrders
}

// FilterOrdersByCurrencies removes any order details that do not match the
// provided currency list. It is forgiving in that the provided currencies can
// match quote or base currencies
func FilterOrdersByCurrencies(orders *[]Detail, currencies []currency.Pair) {
	if len(currencies) == 0 {
		return
	}
	if len(currencies) == 1 && currencies[0].IsEmpty() {
		return
	}

	var filteredOrders []Detail
	for i := range *orders {
		for _, c := range currencies {
			if (*orders)[i].Pair.EqualIncludeReciprocal(c) {
				filteredOrders = append(filteredOrders, (*orders)[i])
				break
			}
		}
	}

	*orders = filteredOrders
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
	switch {
	case strings.EqualFold(side, Buy.String()):
		return Buy, nil
	case strings.EqualFold(side, Sell.String()):
		return Sell, nil
	case strings.EqualFold(side, Bid.String()):
		return Bid, nil
	case strings.EqualFold(side, Ask.String()):
		return Ask, nil
	case strings.EqualFold(side, Long.String()):
		return Long, nil
	case strings.EqualFold(side, Short.String()):
		return Short, nil
	case strings.EqualFold(side, AnySide.String()):
		return AnySide, nil
	default:
		return UnknownSide, errors.New(side + " not recognised as order side")
	}
}

// StringToOrderType for converting case insensitive order type
// and returning a real Type
func StringToOrderType(oType string) (Type, error) {
	switch {
	case strings.EqualFold(oType, Limit.String()),
		strings.EqualFold(oType, "EXCHANGE LIMIT"):
		return Limit, nil
	case strings.EqualFold(oType, Market.String()),
		strings.EqualFold(oType, "EXCHANGE MARKET"):
		return Market, nil
	case strings.EqualFold(oType, ImmediateOrCancel.String()),
		strings.EqualFold(oType, "immediate or cancel"),
		strings.EqualFold(oType, "IOC"),
		strings.EqualFold(oType, "EXCHANGE IOC"):
		return ImmediateOrCancel, nil
	case strings.EqualFold(oType, Stop.String()),
		strings.EqualFold(oType, "stop loss"),
		strings.EqualFold(oType, "stop_loss"),
		strings.EqualFold(oType, "EXCHANGE STOP"):
		return Stop, nil
	case strings.EqualFold(oType, StopLimit.String()),
		strings.EqualFold(oType, "EXCHANGE STOP LIMIT"):
		return StopLimit, nil
	case strings.EqualFold(oType, TrailingStop.String()),
		strings.EqualFold(oType, "trailing stop"),
		strings.EqualFold(oType, "EXCHANGE TRAILING STOP"):
		return TrailingStop, nil
	case strings.EqualFold(oType, FillOrKill.String()),
		strings.EqualFold(oType, "EXCHANGE FOK"):
		return FillOrKill, nil
	case strings.EqualFold(oType, IOS.String()):
		return IOS, nil
	case strings.EqualFold(oType, PostOnly.String()):
		return PostOnly, nil
	case strings.EqualFold(oType, AnyType.String()):
		return AnyType, nil
	case strings.EqualFold(oType, Trigger.String()):
		return Trigger, nil
	default:
		return UnknownType, errors.New(oType + " not recognised as order type")
	}
}

// StringToOrderStatus for converting case insensitive order status
// and returning a real Status
func StringToOrderStatus(status string) (Status, error) {
	switch {
	case strings.EqualFold(status, AnyStatus.String()):
		return AnyStatus, nil
	case strings.EqualFold(status, New.String()),
		strings.EqualFold(status, "placed"):
		return New, nil
	case strings.EqualFold(status, Active.String()),
		strings.EqualFold(status, "STATUS_ACTIVE"): // BTSE case
		return Active, nil
	case strings.EqualFold(status, PartiallyFilled.String()),
		strings.EqualFold(status, "partially matched"),
		strings.EqualFold(status, "partially filled"):
		return PartiallyFilled, nil
	case strings.EqualFold(status, Filled.String()),
		strings.EqualFold(status, "fully matched"),
		strings.EqualFold(status, "fully filled"),
		strings.EqualFold(status, "ORDER_FULLY_TRANSACTED"): // BTSE case
		return Filled, nil
	case strings.EqualFold(status, PartiallyCancelled.String()),
		strings.EqualFold(status, "partially cancelled"),
		strings.EqualFold(status, "ORDER_PARTIALLY_TRANSACTED"): // BTSE case
		return PartiallyCancelled, nil
	case strings.EqualFold(status, Open.String()):
		return Open, nil
	case strings.EqualFold(status, Closed.String()):
		return Closed, nil
	case strings.EqualFold(status, Cancelled.String()),
		strings.EqualFold(status, "CANCELED"),        // Binance and Kraken case
		strings.EqualFold(status, "ORDER_CANCELLED"): // BTSE case
		return Cancelled, nil
	case strings.EqualFold(status, PendingCancel.String()),
		strings.EqualFold(status, "pending cancel"),
		strings.EqualFold(status, "pending cancellation"):
		return PendingCancel, nil
	case strings.EqualFold(status, Rejected.String()):
		return Rejected, nil
	case strings.EqualFold(status, Expired.String()):
		return Expired, nil
	case strings.EqualFold(status, Hidden.String()):
		return Hidden, nil
	case strings.EqualFold(status, InsufficientBalance.String()):
		return InsufficientBalance, nil
	case strings.EqualFold(status, MarketUnavailable.String()):
		return MarketUnavailable, nil
	default:
		return UnknownStatus, errors.New(status + " not recognised as order status")
	}
}

func (o *ClassificationError) Error() string {
	if o.OrderID != "" {
		return fmt.Sprintf("%s - OrderID: %s classification error: %v",
			o.Exchange,
			o.OrderID,
			o.Err)
	}
	return fmt.Sprintf("%s - classification error: %v",
		o.Exchange,
		o.Err)
}

// StandardCancel defines an option in the validator to make sure an ID is set
// for a standard cancel
func (c *Cancel) StandardCancel() validate.Checker {
	return validate.Check(func() error {
		if c.ID == "" {
			return errors.New("ID not set")
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

		if c.AssetType == "" {
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

	var errs common.Errors
	for _, o := range opt {
		err := o.Check()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return errs
	}
	return nil
}

// Validate checks internal struct requirements
func (g *GetOrdersRequest) Validate(opt ...validate.Checker) error {
	if g == nil {
		return ErrGetOrdersRequestIsNil
	}
	if !g.AssetType.IsValid() {
		return fmt.Errorf("assetType %v not supported", g.AssetType)
	}
	var errs common.Errors
	for _, o := range opt {
		err := o.Check()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return errs
	}
	return nil
}

// Validate checks internal struct requirements
func (m *Modify) Validate(opt ...validate.Checker) error {
	if m == nil {
		return ErrModifyOrderIsNil
	}

	if m.Pair.IsEmpty() {
		return ErrPairIsEmpty
	}

	if m.AssetType.String() == "" {
		return ErrAssetNotSet
	}

	var errs common.Errors
	for _, o := range opt {
		err := o.Check()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return errs
	}
	if m.ClientOrderID == "" && m.ID == "" {
		return ErrOrderIDNotSet
	}
	return nil
}
