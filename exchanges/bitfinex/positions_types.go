package bitfinex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Position types, which a position's Type gives
const (
	PositionTypeMargin     uint64 = 0
	PositionTypeDerivative uint64 = 1
)

// Position funding types, which a position's MarginFundingType gives and UpdatePositionFundingType sets
const (
	PositionFundingTypeDaily uint64 = 0
	PositionFundingTypeTerm  uint64 = 1
)

// ClaimPositionRequest holds the margin position to claim
type ClaimPositionRequest struct {
	PositionID uint64
	// Amount claims only that much of the position; zero claims all of it
	Amount float64
}

// IncreasePositionRequest holds the margin position to open or increase
type IncreasePositionRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	// Amount is positive to increase a long position and negative to increase a short one
	Amount float64
}

// IncreasePositionInfoRequest holds the trading pair, and the amount of a position on it, to get increase position
// info for
type IncreasePositionInfoRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	// Amount is positive for a long position and negative for a short one; the funding info covers a position of it
	Amount float64
}

// IncreasePositionInfoResponse is what increasing a trading pair's position can do: the sizes it can reach, the
// balances it can use and the funding a position of the requested amount needs
type IncreasePositionInfoResponse struct {
	Position        IncreasePositionInfoPosition        `tuple:"0"`
	Balance         IncreasePositionInfoBalance         `tuple:"1"`
	Funding         IncreasePositionInfoFunding         `tuple:"4"`
	FundingCurrency IncreasePositionInfoFundingCurrency `tuple:"5"`
}

// UnmarshalJSON decodes an increase position info array
func (i *IncreasePositionInfoResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, i)
}

// IncreasePositionInfoPosition is the largest position on a pair that increasing it can reach, and the pair's current
// position
type IncreasePositionInfoPosition struct {
	MaximumPosition float64 `tuple:"0"`
	CurrentPosition float64 `tuple:"1"`
}

// UnmarshalJSON decodes an increase position info position array
func (i *IncreasePositionInfoPosition) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, i)
}

// IncreasePositionInfoBalance is the margin wallet's balance and the funding an increase can use
type IncreasePositionInfoBalance struct {
	// BaseCurrencyBalance is the margin wallet's balance of the pair's base currency
	BaseCurrencyBalance float64                             `tuple:"0"`
	TradableBalance     IncreasePositionInfoTradableBalance `tuple:"1"`
	// FundingAvailable is the funding available at daily rates below 0.75%, in the currency a position of the
	// requested amount needs funding in
	FundingAvailable float64 `tuple:"2"`
}

// UnmarshalJSON decodes an increase position info balance array
func (i *IncreasePositionInfoBalance) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, i)
}

// IncreasePositionInfoTradableBalance is the balance a pair's position can be increased with, in each of the pair's
// currencies
type IncreasePositionInfoTradableBalance struct {
	// QuoteCurrency is the tradable balance in the quote currency, net of open orders and positions
	QuoteCurrency float64 `tuple:"0"`
	// QuoteCurrencyTotal is QuoteCurrency before open orders and positions
	QuoteCurrencyTotal float64 `tuple:"1"`
	// BaseCurrency is the tradable balance in the base currency, net of open orders and positions
	BaseCurrency float64 `tuple:"2"`
	// BaseCurrencyTotal is BaseCurrency before open orders and positions
	BaseCurrencyTotal float64 `tuple:"3"`
}

// UnmarshalJSON decodes an increase position info tradable balance array
func (i *IncreasePositionInfoTradableBalance) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, i)
}

// IncreasePositionInfoFunding is the funding a position of the requested amount needs, in the currencies
// IncreasePositionInfoFundingCurrency gives
type IncreasePositionInfoFunding struct {
	// Value is the funding's value in the pair's other currency
	Value    float64 `tuple:"0"`
	Required float64 `tuple:"1"`
}

// UnmarshalJSON decodes an increase position info funding array
func (i *IncreasePositionInfoFunding) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, i)
}

// IncreasePositionInfoFundingCurrency is the currencies of an IncreasePositionInfoFunding's amounts
type IncreasePositionInfoFundingCurrency struct {
	ValueCurrency    currency.Code `tuple:"0"`
	RequiredCurrency currency.Code `tuple:"1"`
}

// UnmarshalJSON decodes an increase position info funding currency array
func (i *IncreasePositionInfoFundingCurrency) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, i)
}

// PositionsHistoryRequest holds the window, limit and position to get past positions for, each optional
type PositionsHistoryRequest struct {
	Start time.Time
	End   time.Time
	// Limit caps the positions returned, at most 50
	Limit uint64
	// PositionID returns only that position
	PositionID uint64
}

// PositionsSnapshotRequest holds the window, limit and position to get position snapshots for, each optional
type PositionsSnapshotRequest struct {
	Start time.Time
	End   time.Time
	// Limit caps the snapshots returned, at most 500
	Limit uint64
	// PositionID returns only that position's snapshots
	PositionID uint64
}

// PositionsAuditRequest holds the positions, window and limit to audit, each optional
type PositionsAuditRequest struct {
	// PositionIDs audits only those positions; empty audits the latest
	PositionIDs []uint64
	Start       time.Time
	End         time.Time
	// Limit caps the entries returned, at most 250
	Limit uint64
}

// UpdatePositionFundingTypeRequest holds the trading pair of a position and the funding type it is to take
type UpdatePositionFundingTypeRequest struct {
	// Symbol is a trading pair, such as tBTCUSD
	Symbol string
	// FundingType is PositionFundingTypeDaily or PositionFundingTypeTerm
	FundingType uint64
}

// DerivativePositionCollateralRequest holds the collateral a derivative position is to have
type DerivativePositionCollateralRequest struct {
	// Symbol is a derivative pair with an active position, such as tBTCF0:USTF0
	Symbol string
	// Collateral must be within the limits GetDerivativePositionCollateralLimits returns
	Collateral float64
}

// DerivativePositionCollateralResponse is whether a derivative position's collateral was set; Bitfinex sends it in a
// list of one
type DerivativePositionCollateralResponse struct {
	Success types.Boolean `tuple:"0"`
}

// UnmarshalJSON decodes a derivative position collateral status array
func (d *DerivativePositionCollateralResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, d)
}

// DerivativePositionCollateralLimitsResponse is the least and most collateral a derivative position can be given
type DerivativePositionCollateralLimitsResponse struct {
	MinimumCollateral float64 `tuple:"0"`
	MaximumCollateral float64 `tuple:"1"`
}

// UnmarshalJSON decodes a derivative position collateral limits array
func (d *DerivativePositionCollateralLimitsResponse) UnmarshalJSON(data []byte) error {
	return unmarshalTuple(data, d)
}

type claimPositionBody struct {
	ID     uint64 `json:"id"`
	Amount string `json:"amount,omitempty"`
}

type increasePositionBody struct {
	Symbol string `json:"symbol"`
	Amount string `json:"amount"`
}

type increasePositionInfoBody struct {
	Symbol string `json:"symbol"`
	Amount string `json:"amount,omitempty"`
}

type positionsHistoryBody struct {
	historyBody
	ID uint64 `json:"id,omitempty"`
}

type positionsAuditBody struct {
	historyBody
	IDs []uint64 `json:"id,omitempty"`
}

type updatePositionFundingTypeBody struct {
	Symbol string `json:"symbol"`
	// Type is always sent, as 0 selects daily funding
	Type uint64 `json:"type"`
}

type derivativePositionCollateralBody struct {
	Symbol string `json:"symbol"`
	// Collateral is a JSON number, as the docs give it, rather than the string most amounts are sent as
	Collateral json.RawMessage `json:"collateral"`
}

type derivativePositionCollateralLimitsBody struct {
	Symbol string `json:"symbol"`
}
