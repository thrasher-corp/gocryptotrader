package bitfinex

import (
	"bytes"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Records REST responses and websocket events share, decoded from the arrays Bitfinex sends them as. A websocket
// event decodes into the same record as the REST endpoint returning it

// Wallet types, which hold balances and which transfers, deposits and withdrawals use
const (
	WalletTypeExchange = "exchange"
	// WalletTypeMargin is also the derivatives wallet of derivative pairs
	WalletTypeMargin  = "margin"
	WalletTypeFunding = "funding"
)

// Notification statuses Bitfinex reports a request's outcome with
const (
	NotificationStatusSuccess = "SUCCESS"
	NotificationStatusError   = "ERROR"
	NotificationStatusFailure = "FAILURE"
)

// Order types. One prefixed with ExchangeOrderTypePrefix trades on the exchange wallet, one without on the margin wallet
const (
	OrderTypeLimit             = "LIMIT"
	OrderTypeMarket            = "MARKET"
	OrderTypeStop              = "STOP"
	OrderTypeStopLimit         = "STOP LIMIT"
	OrderTypeTrailingStop      = "TRAILING STOP"
	OrderTypeFillOrKill        = "FOK"
	OrderTypeImmediateOrCancel = "IOC"
	ExchangeOrderTypePrefix    = "EXCHANGE "
)

// Order flags, which an order's Flags sum; see https://docs.bitfinex.com/docs/flag-values
const (
	OrderFlagHidden uint64 = 1 << 6
	// OrderFlagClose closes the position the order's pair has
	OrderFlagClose uint64 = 1 << 9
	// OrderFlagReduceOnly keeps the order from flipping the position it reduces
	OrderFlagReduceOnly uint64 = 1 << 10
	// OrderFlagPostOnly cancels the order rather than let it match a resting one, other than a hidden one
	OrderFlagPostOnly uint64 = 1 << 12
	// OrderFlagOneCancelsOther pairs the order with an OCO stop order, cancelling either once the other executes
	OrderFlagOneCancelsOther uint64 = 1 << 14
	// OrderFlagNoVariableRates keeps a margin order from taking funding at variable rates
	OrderFlagNoVariableRates uint64 = 1 << 19
)

// Funding offer types
const (
	FundingOfferTypeLimit = "LIMIT"
	// FundingOfferTypeFlashReturnRateDeltaVariable offers at the flash return rate plus the offer's rate, varying with it
	FundingOfferTypeFlashReturnRateDeltaVariable = "FRRDELTAVAR"
	// FundingOfferTypeFlashReturnRateDeltaFixed offers at the flash return rate plus the offer's rate, fixed when taken
	FundingOfferTypeFlashReturnRateDeltaFixed = "FRRDELTAFIX"
)

// Notification is Bitfinex's reply to a write request, and an account event on the websocket, carrying the record the
// request affected as Data
type Notification[T any] struct {
	// Time is when Bitfinex sent the notification, in seconds or milliseconds depending on the endpoint
	Time      types.Time `tuple:"0"`
	Type      string     `tuple:"1"`
	MessageID uint64     `tuple:"2"`
	Data      T          `tuple:"4"`
	Code      int64      `tuple:"5"`
	Status    string     `tuple:"6"`
	Text      string     `tuple:"7"`
}

// UnmarshalJSON decodes a notification array
func (n *Notification[T]) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, n) }

// Err returns an error when the notification reports its request failed
func (n *Notification[T]) Err() error {
	switch n.Status {
	case NotificationStatusError, NotificationStatusFailure:
		return fmt.Errorf("%w: %s %s: %s", errNotificationFailed, n.Type, n.Status, n.Text)
	}
	return nil
}

// Order is an order on a trading pair
type Order struct {
	ID            uint64     `tuple:"0"`
	GroupID       uint64     `tuple:"1"`
	ClientOrderID uint64     `tuple:"2"`
	Symbol        string     `tuple:"3"`
	CreatedAt     types.Time `tuple:"4"`
	UpdatedAt     types.Time `tuple:"5"`
	// Amount is what remains of the order: positive for a buy, negative for a sell
	Amount float64 `tuple:"6"`
	// OriginalAmount is the amount the order was placed, or last updated, with
	OriginalAmount float64 `tuple:"7"`
	// Type is one of the order types, such as LIMIT or EXCHANGE LIMIT
	Type         string `tuple:"8"`
	PreviousType string `tuple:"9"`
	// TimeInForce is when Bitfinex cancels the order, for an order given a time in force
	TimeInForce types.Time `tuple:"10"`
	// Flags is the sum of the order's flags
	Flags uint64 `tuple:"12"`
	// Status is ACTIVE, or the order's history, such as "EXECUTED @ 107.6(-0.2)" or "CANCELED was: PARTIALLY FILLED @
	// 107.6(-0.1)"
	Status              string  `tuple:"13"`
	Price               float64 `tuple:"16"`
	AveragePrice        float64 `tuple:"17"`
	TrailingPrice       float64 `tuple:"18"`
	AuxiliaryLimitPrice float64 `tuple:"19"`
	// Notify is whether operations on the order send notifications
	Notify types.Boolean `tuple:"23"`
	Hidden types.Boolean `tuple:"24"`
	// PlacedID is the order which placed this one, when it is the stop order of a one cancels other pair
	PlacedID uint64 `tuple:"25"`
	// Routing is where the order came from, such as BFX or API>BFX
	Routing string `tuple:"28"`
	// Meta holds more about the order, such as $F7 for post only and $F33 for leverage
	Meta json.RawMessage `tuple:"31"`
}

// UnmarshalJSON decodes an order array
func (o *Order) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, o) }

// AccountTrade is a trade one of the account's orders made
type AccountTrade struct {
	ID         uint64     `tuple:"0"`
	Symbol     string     `tuple:"1"`
	ExecutedAt types.Time `tuple:"2"`
	OrderID    uint64     `tuple:"3"`
	// ExecutedAmount is positive for a buy and negative for a sell
	ExecutedAmount float64 `tuple:"4"`
	ExecutedPrice  float64 `tuple:"5"`
	OrderType      string  `tuple:"6"`
	// OrderPrice is unset for trades GetOrderTrades returns
	OrderPrice float64 `tuple:"7"`
	// Maker is 1 for a trade made as maker and -1 for one taken
	Maker int64 `tuple:"8"`
	// Fee is negative when charged; a websocket trade executed event leaves it and FeeCurrency unset, and the trade's
	// execution update sets them
	Fee           float64       `tuple:"9"`
	FeeCurrency   currency.Code `tuple:"10"`
	ClientOrderID uint64        `tuple:"11"`
}

// UnmarshalJSON decodes an account trade array
func (t *AccountTrade) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, t) }

// Position is a margin or derivative position
type Position struct {
	Symbol string `tuple:"0"`
	// Status is ACTIVE or CLOSED
	Status string `tuple:"1"`
	// Amount is the position's size: positive for a long position, negative for a short one
	Amount float64 `tuple:"2"`
	// BasePrice is the average price of the orders that opened and grew the position
	BasePrice     float64 `tuple:"3"`
	MarginFunding float64 `tuple:"4"`
	// MarginFundingType is 0 for daily funding and 1 for term funding
	MarginFundingType    uint64  `tuple:"5"`
	ProfitLoss           float64 `tuple:"6"`
	ProfitLossPercentage float64 `tuple:"7"`
	// LiquidationPrice is zero until Bitfinex has calculated it
	LiquidationPrice float64    `tuple:"8"`
	Leverage         float64    `tuple:"9"`
	Flags            uint64     `tuple:"10"`
	ID               uint64     `tuple:"11"`
	CreatedAt        types.Time `tuple:"12"`
	UpdatedAt        types.Time `tuple:"13"`
	// Type is 0 for a margin position and 1 for a derivative position
	Type              uint64  `tuple:"15"`
	Collateral        float64 `tuple:"17"`
	MinimumCollateral float64 `tuple:"18"`
	// Meta holds more about the position, such as the trade that last changed it
	Meta json.RawMessage `tuple:"19"`
}

// UnmarshalJSON decodes a position array
func (p *Position) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, p) }

// FundingFlags is the sum of a funding offer's or credit's flags. Bitfinex documents some endpoints' flags as an
// object reserved for future parameters, which decodes as no flags
type FundingFlags uint64

// UnmarshalJSON decodes flags from a number, or as none from an object
func (f *FundingFlags) UnmarshalJSON(data []byte) error {
	if data = bytes.TrimSpace(data); len(data) > 0 && data[0] == '{' {
		*f = 0
		return nil
	}
	var flags uint64
	if err := json.Unmarshal(data, &flags); err != nil {
		return err
	}
	*f = FundingFlags(flags)
	return nil
}

// FundingOffer is an offer to provide or take funding
type FundingOffer struct {
	ID        uint64     `tuple:"0"`
	Symbol    string     `tuple:"1"`
	CreatedAt types.Time `tuple:"2"`
	UpdatedAt types.Time `tuple:"3"`
	// Amount is what remains of the offer: positive to provide funding, negative to take it
	Amount         float64 `tuple:"4"`
	OriginalAmount float64 `tuple:"5"`
	// Type is one of the funding offer types
	Type  string       `tuple:"6"`
	Flags FundingFlags `tuple:"9"`
	// Status is ACTIVE, PARTIALLY FILLED, EXECUTED or CANCELED
	Status string `tuple:"10"`
	// Rate is daily; 0.01 is one percent
	Rate float64 `tuple:"14"`
	// Period is in days
	Period uint64        `tuple:"15"`
	Notify types.Boolean `tuple:"16"`
	Hidden types.Boolean `tuple:"17"`
	Renew  types.Boolean `tuple:"19"`
}

// UnmarshalJSON decodes a funding offer array
func (f *FundingOffer) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// FundingCredit is funding provided or taken that is open: a credit is used in a position, a loan is not and leaves
// PositionPair empty
type FundingCredit struct {
	ID     uint64 `tuple:"0"`
	Symbol string `tuple:"1"`
	// Side is 1 when the account lends, -1 when it borrows and 0 when it does both
	Side      int64        `tuple:"2"`
	CreatedAt types.Time   `tuple:"3"`
	UpdatedAt types.Time   `tuple:"4"`
	Amount    float64      `tuple:"5"`
	Flags     FundingFlags `tuple:"6"`
	Status    string       `tuple:"7"`
	// RateType is FIXED, or VAR for funding at the flash return rate
	RateType string `tuple:"8"`
	// Rate is daily; 0.01 is one percent
	Rate float64 `tuple:"11"`
	// Period is in days
	Period       uint64        `tuple:"12"`
	OpenedAt     types.Time    `tuple:"13"`
	LastPayoutAt types.Time    `tuple:"14"`
	Notify       types.Boolean `tuple:"15"`
	Hidden       types.Boolean `tuple:"16"`
	Renew        types.Boolean `tuple:"18"`
	// RealRate is the rate Bitfinex calculated for funding at, or relative to, the flash return rate
	RealRate     float64       `tuple:"19"`
	NoClose      types.Boolean `tuple:"20"`
	PositionPair string        `tuple:"21"`
}

// UnmarshalJSON decodes a funding credit or loan array
func (f *FundingCredit) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// AccountFundingTrade is a trade one of the account's funding offers made
type AccountFundingTrade struct {
	ID        uint64     `tuple:"0"`
	Symbol    string     `tuple:"1"`
	CreatedAt types.Time `tuple:"2"`
	OfferID   uint64     `tuple:"3"`
	// Amount is positive for funding provided and negative for funding taken
	Amount float64 `tuple:"4"`
	// Rate is daily; 0.01 is one percent
	Rate float64 `tuple:"5"`
	// Period is in days
	Period uint64 `tuple:"6"`
	// Maker is set on websocket events for a trade made as maker
	Maker types.Boolean `tuple:"7"`
}

// UnmarshalJSON decodes an account funding trade array
func (f *AccountFundingTrade) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// MarginInfoBase is the account's margin summary
type MarginInfoBase struct {
	UserProfitLoss float64 `tuple:"0"`
	UserSwaps      float64 `tuple:"1"`
	MarginBalance  float64 `tuple:"2"`
	// MarginNet is MarginBalance with profit and loss
	MarginNet float64 `tuple:"3"`
	// MarginMinimum is the margin the open positions need
	MarginMinimum float64 `tuple:"4"`
}

// UnmarshalJSON decodes a margin summary array
func (m *MarginInfoBase) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, m) }

// MarginInfoBaseResponse is the margin summary keyed "base", as REST and websocket margin info send it
type MarginInfoBaseResponse struct {
	Key  string         `tuple:"0"`
	Info MarginInfoBase `tuple:"1"`
}

// UnmarshalJSON decodes a keyed margin summary array
func (m *MarginInfoBaseResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, m) }

// MarginInfoSymbol is the account's buying power on a trading pair
type MarginInfoSymbol struct {
	TradableBalance float64 `tuple:"0"`
	// GrossBalance is TradableBalance with the funds open positions hold
	GrossBalance float64 `tuple:"1"`
	// BuyAmount is the most the account can buy at the best ask
	BuyAmount float64 `tuple:"2"`
	// SellAmount is the most the account can sell at the best bid
	SellAmount float64 `tuple:"3"`
}

// UnmarshalJSON decodes a trading pair's margin array
func (m *MarginInfoSymbol) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, m) }

// MarginInfoSymbolResponse is a trading pair's margin keyed "sym", as REST and websocket margin info send it
type MarginInfoSymbolResponse struct {
	Key    string           `tuple:"0"`
	Symbol string           `tuple:"1"`
	Info   MarginInfoSymbol `tuple:"2"`
}

// UnmarshalJSON decodes a keyed trading pair margin array
func (m *MarginInfoSymbolResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, m) }

// FundingInfo is the yield and duration of a currency's funding the account provides and takes
type FundingInfo struct {
	// YieldLoan is the weighted average daily rate of funding taken
	YieldLoan float64 `tuple:"0"`
	// YieldLend is the weighted average daily rate of funding provided
	YieldLend float64 `tuple:"1"`
	// DurationLoan is the weighted average duration, in days, of funding taken
	DurationLoan float64 `tuple:"2"`
	// DurationLend is the weighted average duration, in days, of funding provided
	DurationLend float64 `tuple:"3"`
}

// UnmarshalJSON decodes a funding info array
func (f *FundingInfo) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }

// FundingInfoResponse is a funding currency's info keyed "sym", as REST and websocket funding info send it
type FundingInfoResponse struct {
	Key    string      `tuple:"0"`
	Symbol string      `tuple:"1"`
	Info   FundingInfo `tuple:"2"`
}

// UnmarshalJSON decodes a keyed funding info array
func (f *FundingInfoResponse) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, f) }
