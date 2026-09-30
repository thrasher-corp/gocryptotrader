package htx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// V5AssetsDeductionCurrenciesResponse stores currencies supported for fee deduction.
type V5AssetsDeductionCurrenciesResponse struct {
	V5Response
	Data struct {
		Currencies []string `json:"currency"`
	} `json:"data"`
}

// V5MultiAssetsMarginCurrenciesResponse stores currencies supported by multi-assets margin.
type V5MultiAssetsMarginCurrenciesResponse struct {
	V5Response
	Data struct {
		Currencies []string `json:"multi_assets"`
	} `json:"data"`
}

// V5EliteRatioResponse stores elite-trader long/short ratios.
type V5EliteRatioResponse struct {
	V5Response
	Data []V5EliteRatio `json:"data"`
}

// V5EliteRatio stores one elite-trader ratio sample.
type V5EliteRatio struct {
	ContractCode string       `json:"contract_code"`
	BuyRatio     types.Number `json:"buy_ratio"`
	SellRatio    types.Number `json:"sell_ratio"`
	Timestamp    types.Time   `json:"ts"`
}

// V5EstimatedSettlementPriceResponse stores estimated settlement prices.
type V5EstimatedSettlementPriceResponse struct {
	V5Response
	Data []V5EstimatedSettlementPrice `json:"data"`
}

// V5EstimatedSettlementPrice stores an estimated contract settlement price.
type V5EstimatedSettlementPrice struct {
	ContractCode             string       `json:"contract_code"`
	SettlementType           string       `json:"settlement_type"`
	EstimatedSettlementPrice types.Number `json:"estimated_settlement_price"`
}

// V5FundingRatesRequest defines the contracts requested from the current funding-rate endpoint.
type V5FundingRatesRequest struct {
	ContractCodes currency.Pairs
}

// V5FundingRatesResponse stores current contract funding rates.
type V5FundingRatesResponse struct {
	V5Response
	Data []V5FundingRate `json:"data"`
}

// V5FundingRate stores the current and next funding information for one contract.
type V5FundingRate struct {
	ContractCode    string       `json:"contract_code"`
	FundingRate     types.Number `json:"funding_rate"`
	FundingTime     types.Time   `json:"funding_time"`
	NextFundingTime types.Time   `json:"next_funding_time"`
	MinimumRate     types.Number `json:"min_funding_rate"`
	MaximumRate     types.Number `json:"max_funding_rate"`
}

// V5FundingRateHistoryRequest defines historical funding-rate filters.
type V5FundingRateHistoryRequest struct {
	ContractCode string
	StartTime    time.Time
	EndTime      time.Time
	From         string
	Limit        uint64
	Direction    string
}

// V5FundingRateHistoryResponse stores historical contract funding rates.
type V5FundingRateHistoryResponse struct {
	V5Response
	Data []V5HistoricalFundingRate `json:"data"`
}

// V5HistoricalFundingRate stores one historical contract funding rate.
type V5HistoricalFundingRate struct {
	ID           string       `json:"id"`
	ContractCode string       `json:"contract_code"`
	FundingRate  types.Number `json:"funding_rate"`
	FundingTime  types.Time   `json:"funding_time"`
}

// V5PriceLimitsResponse stores current contract price limits.
type V5PriceLimitsResponse struct {
	V5Response
	Data []V5PriceLimit `json:"data"`
}

// V5PriceLimit stores the highest permitted buy price and lowest permitted sell price.
type V5PriceLimit struct {
	ContractCode string       `json:"contract_code"`
	High         types.Number `json:"high_limit"`
	Low          types.Number `json:"low_limit"`
}

// V5LiquidationOrdersRequest defines public liquidation-order filters.
type V5LiquidationOrdersRequest struct {
	ContractCode string
	Pair         string
	StartTime    time.Time
	EndTime      time.Time
	Direction    string
	From         string
	Limit        uint64
}

// V5LiquidationOrdersResponse stores public liquidation orders.
type V5LiquidationOrdersResponse struct {
	V5Response
	Data []V5LiquidationOrder `json:"data"`
}

// V5LiquidationOrder stores a public liquidation order.
type V5LiquidationOrder struct {
	ID              string       `json:"id"`
	ContractCode    string       `json:"contract_code"`
	LiquidationTime types.Time   `json:"liquidation_time"`
	Side            string       `json:"side"`
	PositionSide    string       `json:"position_side"`
	Volume          types.Number `json:"volume"`
	Amount          types.Number `json:"amount"`
	BankruptcyPrice types.Number `json:"bankrupt_price"`
	TradeTurnover   types.Number `json:"trade_turnover"`
}

// V5RiskLimitResponse stores contract risk limits.
type V5RiskLimitResponse struct {
	V5Response
	Data []V5RiskLimit `json:"data"`
}

// V5RiskLimit stores one contract risk-limit tier.
type V5RiskLimit struct {
	ContractCode          string       `json:"contract_code"`
	MarginMode            string       `json:"margin_mode"`
	PositionSide          string       `json:"position_side"`
	Tier                  types.Number `json:"tier"`
	MaximumLeverage       types.Number `json:"max_lever"`
	MaintenanceMarginRate types.Number `json:"maintenance_margin_rate"`
	MaximumVolume         types.Number `json:"max_volume"`
	MinimumVolume         types.Number `json:"min_volume"`
	VolumeUnit            string       `json:"volume_unit"`
}

// V5SettlementHistoryRequest defines settlement-history filters.
type V5SettlementHistoryRequest struct {
	ContractCode string
	StartTime    time.Time
	EndTime      time.Time
	Direction    string
	From         string
	Limit        uint64
}

// V5SettlementHistoryResponse stores public settlement history.
type V5SettlementHistoryResponse struct {
	V5Response
	Data []V5Settlement `json:"data"`
}

// V5Settlement stores one contract settlement.
type V5Settlement struct {
	ID              string       `json:"id"`
	ContractCode    string       `json:"contract_code"`
	SettlementTime  types.Time   `json:"settlement_time"`
	ClawbackRatio   types.Number `json:"clawback_ratio"`
	SettlementPrice types.Number `json:"settlement_price"`
}
