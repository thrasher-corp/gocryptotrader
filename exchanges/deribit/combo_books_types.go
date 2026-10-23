package deribit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Combo book states; ComboStateAny selects active and inactive combos, where a request accepts it
const (
	ComboStateActive   = "active"
	ComboStateInactive = "inactive"
	ComboStateAny      = "any"
)

// ComboTrade contains a trade of a combo structure, which defines a combo's legs or prices one
type ComboTrade struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is in USD for perpetuals and inverse futures, and in the base currency for options and linear futures
	Amount float64 `json:"amount"`
	// Direction is buy or sell
	Direction string `json:"direction"`
}

// CreateComboRequest contains the trades of a create combo request
type CreateComboRequest struct {
	Trades []ComboTrade `json:"trades"`
}

// ComboResponse contains a combo book and its legs
type ComboResponse struct {
	// ID is the combo's instrument name
	ID           string `json:"id"`
	InstrumentID uint64 `json:"instrument_id"`
	// State is the combo book's state, active or inactive, which differs from its order book's state
	State             string     `json:"state"`
	StateTimestamp    types.Time `json:"state_timestamp"`
	CreationTimestamp types.Time `json:"creation_timestamp"`
	Legs              []ComboLeg `json:"legs"`
}

// ComboLeg contains a combo leg's instrument and size multiplier
type ComboLeg struct {
	InstrumentName string `json:"instrument_name"`
	// Amount is the leg's size multiplier, negative for a leg traded opposite to the combo
	Amount int64 `json:"amount"`
}

// LegPricesRequest requests the leg prices of a combo structure for a price of the whole structure
type LegPricesRequest struct {
	Legs []ComboTrade `json:"legs"`
	// Price is the price of the whole structure
	Price float64 `json:"price"`
}

// LegPricesResponse contains the leg prices of a combo structure
type LegPricesResponse struct {
	Legs []LegPrice `json:"legs"`
	// Amount multiplied by a leg's ratio is the leg's trade size
	Amount float64 `json:"amount"`
}

// LegPrice contains a combo leg's price
type LegPrice struct {
	InstrumentName string `json:"instrument_name"`
	// Direction is buy or sell
	Direction string  `json:"direction"`
	Price     float64 `json:"price"`
	// Ratio is the leg's amount relative to the structure's amount
	Ratio uint64 `json:"ratio"`
}

// ComboIDRequest contains the combo of a request
type ComboIDRequest struct {
	ComboID string `json:"combo_id"`
}

// ComboIDsRequest requests the IDs of a settlement currency's combos
type ComboIDsRequest struct {
	Currency currency.Code `json:"currency"`
	// State is active, inactive or any, the default, for both
	State string `json:"state,omitempty"`
}
