package gateio

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// UserRiskUnitDetails represents the risk unit details for a user
type UserRiskUnitDetails struct {
	UserID    int64       `json:"user_id"`
	SpotHedge bool        `json:"spot_hedge"`
	RiskUnits []RiskUnits `json:"risk_units"`
}

// RiskUnits represents risk unit details for a specific currency
type RiskUnits struct {
	Symbol         currency.Code `json:"symbol"`
	SpotInUse      types.Number  `json:"spot_in_use"`
	MaintainMargin types.Number  `json:"maintain_margin"`
	InitialMargin  types.Number  `json:"initial_margin"`
	Delta          types.Number  `json:"delta"`
	Gamma          types.Number  `json:"gamma"`
	Theta          types.Number  `json:"theta"`
	Vega           types.Number  `json:"vega"`
}

// RiskTable represents the risk table information
type RiskTable struct {
	Tier            uint8         `json:"tier"`
	RiskLimit       types.Number  `json:"risk_limit"`
	InitialRate     types.Number  `json:"initial_rate"`
	MaintenanceRate types.Number  `json:"maintenance_rate"`
	LeverageMax     types.Number  `json:"leverage_max"`
	Deduction       types.Number  `json:"deduction"`
	Contract        currency.Pair `json:"contract"` // Only available when fetching all risk limit tiers
}
