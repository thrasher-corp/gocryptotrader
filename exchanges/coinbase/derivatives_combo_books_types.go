package coinbase

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Gateway combo book states; DerivativesComboStateAny selects active and inactive combos, as leaving the state out
// does, which the docs leave out but the gateway accepts
const (
	DerivativesComboStateActive   = "active"
	DerivativesComboStateInactive = "inactive"
	DerivativesComboStateAny      = "any"
)

// DerivativesComboTrade is a trade of a combo structure, which defines a combo's legs or prices one
type DerivativesComboTrade struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is in USD for inverse futures and perpetuals, and in the base currency for options and linear instruments
	Amount float64 `json:"amount"`
	// Direction is buy or sell
	Direction string `json:"direction"`
}

// DerivativesCreateComboWireRequest holds private/create_combo's params
type DerivativesCreateComboWireRequest struct {
	Trades []DerivativesComboTrade `json:"trades"`
}

// DerivativesComboResponse is a combo book and its legs
type DerivativesComboResponse struct {
	// ID is the combo's instrument name
	ID           string `json:"id"`
	InstrumentID uint64 `json:"instrument_id"`
	// State is the combo book's state, active or inactive, or closed once its legs expire, which the docs leave out; it
	// differs from its order book's state
	State             string                `json:"state"`
	StateTimestamp    types.Time            `json:"state_timestamp"`
	CreationTimestamp types.Time            `json:"creation_timestamp"`
	Legs              []DerivativesComboLeg `json:"legs"`
}

// DerivativesComboLeg is a combo leg's instrument and size multiplier
type DerivativesComboLeg struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is negative for a leg traded opposite to the combo
	Amount int64 `json:"amount"`
}

// DerivativesLegPricesRequest selects the leg prices of a combo structure for a price of the whole structure
type DerivativesLegPricesRequest struct {
	Legs []DerivativesComboTrade
	// Price is the price of the whole structure, which may be negative or zero
	Price float64
}

// DerivativesLegPricesWireRequest holds private/get_leg_prices's params; the price is always sent, as zero is a price
type DerivativesLegPricesWireRequest struct {
	Legs  []DerivativesComboTrade `json:"legs"`
	Price float64                 `json:"price"`
}

// DerivativesLegPricesResponse is the leg prices of a combo structure
type DerivativesLegPricesResponse struct {
	Legs []DerivativesLegPrice `json:"legs"`
	// Amount multiplied by a leg's ratio is the leg's trade size
	Amount float64 `json:"amount"`
}

// DerivativesLegPrice is a combo leg's price
type DerivativesLegPrice struct {
	InstrumentName string `json:"instrument_name"`
	// Direction is buy or sell
	Direction string  `json:"direction"`
	Price     float64 `json:"price"`
	// Ratio is the leg's amount relative to the structure's amount
	Ratio uint64 `json:"ratio"`
}

// DerivativesComboIDWireRequest holds the params of a request for a combo
type DerivativesComboIDWireRequest struct {
	ComboID string `json:"combo_id"`
}

// DerivativesComboIDsRequest selects the IDs of a settlement currency's combos
type DerivativesComboIDsRequest struct {
	// Currency is a settlement currency; the gateway does not accept any
	Currency currency.Code
	// State is active, inactive or any for both, the default
	State string
}

// DerivativesComboIDsWireRequest holds public/get_combo_ids's params
type DerivativesComboIDsWireRequest struct {
	// Currency stays a string, the code in the case the gateway takes
	Currency string `json:"currency"`
	State    string `json:"state,omitempty"`
}
