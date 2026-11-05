package htx

import "github.com/thrasher-corp/gocryptotrader/types"

// V5LeverageResponse stores leverage levels.
type V5LeverageResponse struct {
	V5Response
	Leverage []V5Leverage `json:"data"`
}

// V5Leverage stores current and available leverage levels.
type V5Leverage struct {
	ContractCode      string   `json:"contract_code"`
	ContractType      string   `json:"contract_type"`
	MarginMode        string   `json:"margin_mode"`
	PositionSide      string   `json:"position_side"`
	LeverageRate      uint64   `json:"lever_rate"`
	AvailableLeverage []uint64 `json:"available_lever"`
}

// V5SetLeverageRequest defines a leverage change.
type V5SetLeverageRequest struct {
	ContractCode string       `json:"contract_code"`
	MarginMode   string       `json:"margin_mode"`
	PositionSide string       `json:"position_side,omitempty"`
	LeverageRate types.Number `json:"lever_rate"`
}

// V5SetLeverageResponse stores an accepted leverage change.
type V5SetLeverageResponse struct {
	V5Response
	Leverage V5LeverageSetting `json:"data"`
}

// V5AdjustPositionMarginRequest defines an isolated-position margin adjustment.
type V5AdjustPositionMarginRequest struct {
	ContractCode string       `json:"contract_code"`
	PositionSide string       `json:"position_side"`
	Type         string       `json:"type"`
	Amount       types.Number `json:"amount"`
}

// V5SetPositionModeRequest defines an account position-mode change.
type V5SetPositionModeRequest struct {
	PositionMode string `json:"position_mode"`
}

// V5PositionModeResponse stores the account position mode.
type V5PositionModeResponse struct {
	V5Response
	Mode V5PositionMode `json:"data"`
}

// V5LeverageSetting identifies leverage by contract, margin mode and position side.
type V5LeverageSetting struct {
	ContractCode string       `json:"contract_code"`
	MarginMode   string       `json:"margin_mode"`
	PositionSide string       `json:"position_side"`
	LeverageRate types.Number `json:"lever_rate"`
}

// V5PositionMode distinguishes one-way positions from independently held long and short positions.
type V5PositionMode struct {
	PositionMode string `json:"position_mode"`
}
